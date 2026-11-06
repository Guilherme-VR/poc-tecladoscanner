//go:build linux

package main

// Captura no Linux lendo /dev/input/eventN. Cada arquivo é um aparelho, então
// a origem é conhecida pelo arquivo lido; o VID:PID é lido uma vez, ao abrir.
// Leitura sem "grab": nada é bloqueado, as teclas seguem normalmente.
// Precisa que o usuário esteja no grupo "input".

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	evKey        = 1
	tamEvento    = 24 // struct input_event em 64 bits
	keyBackspace = 14
	keyEnter     = 28
	keyKPEnter   = 96
	keyCapsLock  = 58
	keyLShift    = 42
	keyRShift    = 54
	keyLCtrl     = 29
	keyRCtrl     = 97
	keyLAlt      = 56
	keyRAlt      = 100 // AltGr
)

var (
	abertosMu sync.Mutex
	abertos   = map[string]bool{}
	capsMu    sync.Mutex
	capsLock  bool
)

// prepararPermissao: se o usuário já está no grupo "input" mas a sessão ainda
// não carregou o grupo (entrou no grupo e não fez logout), o agente se reabre
// com "sg input", que ativa o grupo sem pedir senha. Assim não precisa sair
// da sessão nem usar sudo.
func prepararPermissao() {
	if os.Getenv("TECLADO_SG") == "1" {
		return // já reaberto, evita repetir
	}
	g, err := user.LookupGroup("input")
	if err != nil {
		return
	}
	gid, _ := strconv.Atoi(g.Gid)
	atuais, _ := os.Getgroups()
	if slices.Contains(atuais, gid) || os.Geteuid() == 0 {
		return // já tem acesso
	}
	u, err := user.Current()
	if err != nil {
		return
	}
	grupos, _ := u.GroupIds()
	if !slices.Contains(grupos, g.Gid) {
		return // não está no grupo: o erro de permissão vai orientar
	}
	sg, err := exec.LookPath("sg")
	if err != nil {
		return
	}
	eu, err := os.Executable()
	if err != nil {
		return
	}
	cmd := "'" + strings.ReplaceAll(eu, "'", `'\''`) + "'"
	for _, a := range os.Args[1:] {
		cmd += " '" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	// Substitui o processo atual; stdin/stdout continuam ligados ao app.
	syscall.Exec(sg, []string{"sg", "input", "-c", cmd}, append(os.Environ(), "TECLADO_SG=1"))
}

func mostrarErro(texto string) {
	fmt.Fprintln(os.Stderr, texto)
}

func capturar() error {
	avisouPermissao := false
	for {
		eventos, _ := filepath.Glob("/sys/class/input/event*")
		for _, sys := range eventos {
			nome := filepath.Base(sys)
			abertosMu.Lock()
			ja := abertos[nome]
			abertosMu.Unlock()
			if ja || !ehTeclado(sys) {
				continue
			}
			f, err := os.Open("/dev/input/" + nome)
			if err != nil {
				if errors.Is(err, fs.ErrPermission) && !avisouPermissao {
					avisouPermissao = true
					emitir(Evento{Tipo: "erro", Texto: "Sem permissão para ler os teclados. Rode: sudo usermod -aG input $USER  (depois saia e entre de novo na sessão)"})
				}
				continue
			}
			d := Dispositivo{
				Disp: nome,
				ID:   strings.ToUpper(lerSys(sys, "device/id/vendor") + ":" + lerSys(sys, "device/id/product")),
				Nome: lerSys(sys, "device/name"),
			}
			abertosMu.Lock()
			abertos[nome] = true
			abertosMu.Unlock()
			go ler(f, d)
		}
		time.Sleep(2 * time.Second) // detecta aparelhos plugados depois
	}
}

func lerSys(sys, arq string) string {
	b, _ := os.ReadFile(filepath.Join(sys, arq))
	return strings.TrimSpace(string(b))
}

// ehTeclado: aparelho que tem as teclas "1" e Enter.
func ehTeclado(sys string) bool {
	palavras := strings.Fields(lerSys(sys, "device/capabilities/key"))
	if len(palavras) == 0 {
		return false
	}
	baixo, err := strconv.ParseUint(palavras[len(palavras)-1], 16, 64)
	if err != nil {
		return false
	}
	return baixo&(1<<2) != 0 && baixo&(1<<keyEnter) != 0
}

func ler(f *os.File, d Dispositivo) {
	defer func() {
		f.Close()
		abertosMu.Lock()
		delete(abertos, d.Disp)
		abertosMu.Unlock()
	}()

	var shiftL, shiftR, ctrl, alt, altGr bool
	buf := make([]byte, tamEvento*64)
	for {
		n, err := f.Read(buf)
		if err != nil {
			return // aparelho desconectado
		}
		for i := 0; i+tamEvento <= n; i += tamEvento {
			ev := buf[i : i+tamEvento]
			tipo := binary.LittleEndian.Uint16(ev[16:])
			cod := binary.LittleEndian.Uint16(ev[18:])
			val := int32(binary.LittleEndian.Uint32(ev[20:])) // 0 solta, 1 aperta, 2 repete
			if tipo != evKey {
				continue
			}
			apertada := val != 0
			switch cod {
			case keyLShift:
				shiftL = apertada
				continue
			case keyRShift:
				shiftR = apertada
				continue
			case keyLCtrl, keyRCtrl:
				ctrl = apertada
				continue
			case keyLAlt:
				alt = apertada
				continue
			case keyRAlt:
				altGr = apertada
				continue
			case keyCapsLock:
				if val == 1 {
					capsMu.Lock()
					capsLock = !capsLock
					capsMu.Unlock()
				}
				continue
			}
			if !apertada {
				continue
			}
			switch cod {
			case keyEnter, keyKPEnter:
				emitirEnter(d)
				continue
			case keyBackspace:
				emitirApagar(d)
				continue
			}
			if ctrl || alt {
				continue
			}
			capsMu.Lock()
			caps := capsLock
			capsMu.Unlock()
			if r, morta, ok := teclaABNT2(cod, shiftL || shiftR, caps, altGr); ok {
				if morta {
					emitirMorta(d, r)
				} else {
					emitirChar(d, r)
				}
			}
		}
	}
}
