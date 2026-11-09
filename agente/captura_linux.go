//go:build linux

package main

// Captura no Linux lendo /dev/input/eventN. Cada arquivo é um aparelho, então
// a origem é conhecida pelo arquivo lido; o ID é a porta física do aparelho.
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
	evKey     = 1
	tamEvento = 24 // struct input_event em 64 bits
	keyEnter  = 28
)

var (
	abertosMu sync.Mutex
	abertos   = map[string]bool{}
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
			// ID único do aparelho: a porta física (phys). Dois scanners iguais
			// em portas USB diferentes ficam com IDs diferentes.
			id := lerSys(sys, "device/phys")
			if id == "" {
				id = nome
			}
			abertosMu.Lock()
			abertos[nome] = true
			abertosMu.Unlock()
			go ler(f, nome, id, lerSys(sys, "device/name"))
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

// ler emite um evento a cada tecla apertada neste aparelho.
func ler(f *os.File, arquivo, id, nome string) {
	defer func() {
		f.Close()
		abertosMu.Lock()
		delete(abertos, arquivo)
		abertosMu.Unlock()
	}()

	buf := make([]byte, tamEvento*64)
	for {
		n, err := f.Read(buf)
		if err != nil {
			return // aparelho desconectado
		}
		for i := 0; i+tamEvento <= n; i += tamEvento {
			ev := buf[i : i+tamEvento]
			tipo := binary.LittleEndian.Uint16(ev[16:])
			val := binary.LittleEndian.Uint32(ev[20:]) // 0 solta, 1 aperta, 2 repete
			if tipo == evKey && val == 1 {
				emitir(Evento{Tipo: "tecla", ID: id, Nome: nome})
			}
		}
	}
}
