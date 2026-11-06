//go:build windows

package main

// Captura no Windows via Raw Input: o Windows entrega uma CÓPIA de cada tecla
// junto com o aparelho de origem (hDevice). Nada é bloqueado ou atrasado:
// a tecla original continua indo normalmente para o programa em foco.
// Não precisa de administrador.

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	hid    = windows.NewLazySystemDLL("hid.dll")
	kernel = windows.NewLazySystemDLL("kernel32.dll")

	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pGetMessageW              = user32.NewProc("GetMessageW")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pRegisterRawInputDevices  = user32.NewProc("RegisterRawInputDevices")
	pGetRawInputData          = user32.NewProc("GetRawInputData")
	pGetRawInputDeviceInfoW   = user32.NewProc("GetRawInputDeviceInfoW")
	pToUnicodeEx              = user32.NewProc("ToUnicodeEx")
	pGetKeyboardLayout        = user32.NewProc("GetKeyboardLayout")
	pGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	pGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	pGetKeyState              = user32.NewProc("GetKeyState")
	pMessageBoxW              = user32.NewProc("MessageBoxW")
	pHidDGetProductString     = hid.NewProc("HidD_GetProductString")
	pGetModuleHandleW         = kernel.NewProc("GetModuleHandleW")
)

const (
	wmInput         = 0x00FF
	ridInput        = 0x10000003
	rimTypeKeyboard = 1
	ridevInputSink  = 0x00000100
	ridiDeviceName  = 0x20000007
	riKeyBreak      = 0x01
	riKeyE0         = 0x02
	hwndMessage     = ^uintptr(2) // (HWND)-3

	vkBack    = 0x08
	vkReturn  = 0x0D
	vkShift   = 0x10
	vkControl = 0x11
	vkMenu    = 0x12
	vkCapital = 0x14
	vkLShift  = 0xA0
	vkRShift  = 0xA1
	vkLCtrl   = 0xA2
	vkRCtrl   = 0xA3
	vkLMenu   = 0xA4
	vkRMenu   = 0xA5
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	PtX     int32
	PtY     int32
	Private uint32
}

type rawInputDevice struct {
	UsagePage uint16
	Usage     uint16
	Flags     uint32
	Target    uintptr
}

type rawInputHeader struct {
	Type   uint32
	Size   uint32
	Device uintptr
	WParam uintptr
}

type rawKeyboard struct {
	MakeCode         uint16
	Flags            uint16
	Reserved         uint16
	VKey             uint16
	Message          uint32
	ExtraInformation uint32
}

type rawInputKeyboard struct {
	Header   rawInputHeader
	Keyboard rawKeyboard
}

// Estado dos modificadores de cada aparelho (o Shift de um teclado não
// afeta o que o scanner digita e vice-versa).
type modificadores struct {
	shiftL, shiftR, ctrlL, ctrlR, altL, altR bool
}

var (
	aparelhos = map[uintptr]Dispositivo{}
	mods      = map[uintptr]*modificadores{}
	capsLock  bool
	reVID     = regexp.MustCompile(`(?i)VID[_&](?:000[12])?([0-9a-f]{4})`)
	rePID     = regexp.MustCompile(`(?i)PID[_&]([0-9a-f]{4})`)
)

// No Windows não precisa de permissão especial.
func prepararPermissao() {}

func mostrarErro(texto string) {
	t, _ := windows.UTF16PtrFromString(texto)
	c, _ := windows.UTF16PtrFromString("Teclado x Scanner")
	pMessageBoxW.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
}

func capturar() error {
	runtime.LockOSThread()

	inst, _, _ := pGetModuleHandleW.Call(0)
	nomeClasse, _ := windows.UTF16PtrFromString("TecladoScannerAgente")
	wc := wndClassEx{
		WndProc:   pDefWindowProcW.Addr(),
		Instance:  inst,
		ClassName: nomeClasse,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	if r, _, err := pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("RegisterClassEx: %v", err)
	}

	// Janela invisível, só para receber mensagens.
	hwnd, _, err := pCreateWindowExW.Call(0, uintptr(unsafe.Pointer(nomeClasse)), 0, 0,
		0, 0, 0, 0, hwndMessage, 0, inst, 0)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx: %v", err)
	}

	// Teclados (usage page 1, usage 6). INPUTSINK = recebe mesmo sem foco.
	rid := rawInputDevice{UsagePage: 0x01, Usage: 0x06, Flags: ridevInputSink, Target: hwnd}
	if r, _, err := pRegisterRawInputDevices.Call(uintptr(unsafe.Pointer(&rid)), 1, unsafe.Sizeof(rid)); r == 0 {
		return fmt.Errorf("RegisterRawInputDevices: %v", err)
	}

	ks, _, _ := pGetKeyState.Call(vkCapital)
	capsLock = ks&1 != 0

	var m msg
	for {
		r, _, err := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) == -1 {
			return fmt.Errorf("GetMessage: %v", err)
		}
		if r == 0 {
			return nil
		}
		if m.Message == wmInput {
			tratarRawInput(m.LParam)
		}
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func tratarRawInput(lParam uintptr) {
	var ri rawInputKeyboard
	tam := uint32(unsafe.Sizeof(ri))
	r, _, _ := pGetRawInputData.Call(lParam, ridInput, uintptr(unsafe.Pointer(&ri)),
		uintptr(unsafe.Pointer(&tam)), unsafe.Sizeof(ri.Header))
	if int32(r) <= 0 || ri.Header.Type != rimTypeKeyboard {
		return
	}

	kb := ri.Keyboard
	if kb.VKey == 0xFF { // tecla "falsa" gerada pelo Windows
		return
	}
	hDev := ri.Header.Device
	d := dispositivo(hDev)
	md := mods[hDev]
	if md == nil {
		md = &modificadores{}
		mods[hDev] = md
	}

	solta := kb.Flags&riKeyBreak != 0
	e0 := kb.Flags&riKeyE0 != 0

	switch kb.VKey {
	case vkShift:
		if kb.MakeCode == 0x36 {
			md.shiftR = !solta
		} else {
			md.shiftL = !solta
		}
		return
	case vkControl:
		if e0 {
			md.ctrlR = !solta
		} else {
			md.ctrlL = !solta
		}
		return
	case vkMenu:
		if e0 {
			md.altR = !solta
		} else {
			md.altL = !solta
		}
		return
	case vkCapital:
		if !solta {
			capsLock = !capsLock // Caps Lock é global no Windows
		}
		return
	}
	if solta {
		return
	}

	switch kb.VKey {
	case vkReturn:
		emitirEnter(d)
		return
	case vkBack:
		emitirApagar(d)
		return
	}

	ctrl := md.ctrlL || md.ctrlR
	alt := md.altL || md.altR
	if ctrl != alt { // Ctrl+X, Alt+Tab etc. não são texto (Ctrl+Alt = AltGr é)
		return
	}

	var estado [256]byte
	marcar := func(on bool, vks ...int) {
		if on {
			for _, v := range vks {
				estado[v] = 0x80
			}
		}
	}
	marcar(md.shiftL, vkShift, vkLShift)
	marcar(md.shiftR, vkShift, vkRShift)
	marcar(md.ctrlL, vkControl, vkLCtrl)
	marcar(md.ctrlR, vkControl, vkRCtrl)
	marcar(md.altL, vkMenu, vkLMenu)
	marcar(md.altR, vkMenu, vkRMenu)
	if capsLock {
		estado[vkCapital] = 0x01
	}

	// Usa o layout do programa em foco (ABNT2, US etc.).
	fg, _, _ := pGetForegroundWindow.Call()
	tid, _, _ := pGetWindowThreadProcessId.Call(fg, 0)
	hkl, _, _ := pGetKeyboardLayout.Call(tid)

	var buf [8]uint16
	scan := uintptr(kb.MakeCode)
	if e0 {
		scan |= 0xE000
	}
	// Flag 0x4: não altera o estado de teclado do Windows (não interfere
	// em acentos digitados no programa em foco).
	n, _, _ := pToUnicodeEx.Call(uintptr(kb.VKey), scan, uintptr(unsafe.Pointer(&estado[0])),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0x4, hkl)

	switch {
	case int32(n) < 0:
		emitirMorta(d, rune(buf[0]))
	case n > 0:
		for _, r := range syscall.UTF16ToString(buf[:n]) {
			if r >= 0x20 {
				emitirChar(d, r)
			}
		}
	}
}

// dispositivo descobre (uma vez só, depois fica em memória) o VID:PID do aparelho.
func dispositivo(h uintptr) Dispositivo {
	if d, ok := aparelhos[h]; ok {
		return d
	}
	d := Dispositivo{Disp: "h" + strconv.FormatUint(uint64(h), 16)}
	if h == 0 {
		// Tecla injetada por software (teclado virtual, área de trabalho remota...).
		d.ID = "SEM-APARELHO"
		d.Nome = "Entrada por software"
		aparelhos[h] = d
		return d
	}

	caminho := nomeAparelho(h)
	vid := reVID.FindStringSubmatch(caminho)
	pid := rePID.FindStringSubmatch(caminho)
	if vid != nil && pid != nil {
		d.ID = strings.ToUpper(vid[1] + ":" + pid[1])
	} else {
		// Sem VID/PID (ex: teclado interno de notebook "ACPI#PNP0303#...").
		partes := strings.Split(caminho, "#")
		if len(partes) > 1 {
			d.ID = strings.ToUpper(partes[1])
		} else {
			d.ID = "DESCONHECIDO"
		}
	}
	d.Nome = nomeProduto(caminho)
	aparelhos[h] = d
	return d
}

func nomeAparelho(h uintptr) string {
	var tam uint32
	pGetRawInputDeviceInfoW.Call(h, ridiDeviceName, 0, uintptr(unsafe.Pointer(&tam)))
	if tam == 0 {
		return ""
	}
	buf := make([]uint16, tam+1)
	pGetRawInputDeviceInfoW.Call(h, ridiDeviceName, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&tam)))
	return windows.UTF16ToString(buf)
}

func nomeProduto(caminho string) string {
	if caminho == "" {
		return ""
	}
	p, err := windows.UTF16PtrFromString(caminho)
	if err != nil {
		return ""
	}
	// Acesso 0: só consulta informações, não abre o teclado para leitura.
	f, err := windows.CreateFile(p, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(f)
	var buf [127]uint16
	r, _, _ := pHidDGetProductString.Call(uintptr(f), uintptr(unsafe.Pointer(&buf[0])), unsafe.Sizeof(buf))
	if r == 0 {
		return ""
	}
	return strings.TrimSpace(windows.UTF16ToString(buf[:]))
}
