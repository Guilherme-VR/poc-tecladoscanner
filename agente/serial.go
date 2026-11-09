package main

// Escuta as portas seriais (COMx no Windows, /dev/ttyUSB* e /dev/ttyACM* no
// Linux) no padrão dos leitores: 9600 8N1. Cada leitura vira um evento com o
// texto completo. A leitura termina no CR/LF ou após uma pausa curta, para
// scanners configurados sem sufixo.
//
// Só ficam abertas as portas que o app pede (comando "portas" no stdin):
//   - durante o cadastro de um scanner: todas, para descobrir de onde veio;
//   - fora dele: só as dos scanners seriais cadastrados.
// Assim o agente não segura a porta de outro equipamento (balança, pinpad...).

import (
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

const pausaFimLeitura = 150 * time.Millisecond

var (
	seriaisMu sync.Mutex
	seriais   = map[string]serial.Port{} // portas abertas agora
	abrindo   = map[string]bool{}        // portas sendo abertas
	avisou    = map[string]bool{}        // erro de abertura já informado
	todas     bool                       // cadastro: abre todas as portas
	pedidas   []string                   // portas dos scanners cadastrados
	acordar   = make(chan struct{}, 1)
)

// definirPortas é chamado quando o app manda o comando "portas".
func definirPortas(abrirTodas bool, lista []string) {
	seriaisMu.Lock()
	todas = abrirTodas
	pedidas = lista
	seriaisMu.Unlock()
	select {
	case acordar <- struct{}{}:
	default:
	}
}

// desejadas: as portas que devem estar abertas agora.
func desejadas() map[string]bool {
	seriaisMu.Lock()
	t, lista := todas, pedidas
	seriaisMu.Unlock()

	r := map[string]bool{}
	if !t {
		for _, p := range lista {
			r[p] = true
		}
		return r
	}
	sistema, _ := serial.GetPortsList()
	for _, p := range sistema {
		if !strings.HasPrefix(p, "/dev/ttyS") { // Linux: portas da placa-mãe, quase sempre inexistentes
			r[p] = true
		}
	}
	return r
}

func querida(nome string) bool {
	seriaisMu.Lock()
	defer seriaisMu.Unlock()
	if todas {
		return true
	}
	for _, p := range pedidas {
		if p == nome {
			return true
		}
	}
	return false
}

// capturarSeriais abre/fecha as portas conforme o pedido do app. Revê a cada
// 2 segundos (portas plugadas depois) ou na hora, quando o pedido muda.
func capturarSeriais() {
	for {
		querer := desejadas()

		seriaisMu.Lock()
		for nome, p := range seriais {
			if !querer[nome] {
				p.Close() // a leitura termina sozinha
				delete(seriais, nome)
			}
		}
		for nome := range querer {
			if seriais[nome] == nil && !abrindo[nome] {
				abrindo[nome] = true
				// Cada porta abre em paralelo: no Windows, abrir uma COM de
				// Bluetooth pode demorar vários segundos.
				go abrir(nome)
			}
		}
		seriaisMu.Unlock()

		select {
		case <-acordar:
		case <-time.After(2 * time.Second):
		}
	}
}

func abrir(nome string) {
	p, err := serial.Open(nome, &serial.Mode{
		BaudRate: 9600, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit,
	})

	seriaisMu.Lock()
	defer seriaisMu.Unlock()
	delete(abrindo, nome)
	if err != nil {
		if !avisou[nome] {
			avisou[nome] = true
			emitir(Evento{Tipo: "aviso", Texto: "Não foi possível abrir " + nome + ": " + err.Error()})
		}
		return
	}
	delete(avisou, nome)
	p.SetReadTimeout(50 * time.Millisecond)
	p.ResetInputBuffer() // descarta o que chegou antes de abrir
	seriais[nome] = p
	go lerSerial(p, nome)
}

func lerSerial(p serial.Port, nome string) {
	defer func() {
		p.Close()
		seriaisMu.Lock()
		if seriais[nome] == p {
			delete(seriais, nome)
		}
		seriaisMu.Unlock()
	}()

	var texto []byte
	var ultimo time.Time
	enviar := func() {
		t := strings.TrimSpace(strings.ToValidUTF8(string(texto), "?"))
		texto = texto[:0]
		// Confere de novo: a porta pode ter deixado de ser pedida (fim do cadastro).
		if t != "" && querida(nome) {
			emitir(Evento{Tipo: "serial", ID: nome, Nome: nome, Texto: t})
		}
	}

	buf := make([]byte, 256)
	for {
		n, err := p.Read(buf)
		if err != nil {
			return // porta fechada ou removida
		}
		if n == 0 { // tempo esgotado sem dados
			if len(texto) > 0 && time.Since(ultimo) >= pausaFimLeitura {
				enviar()
			}
			continue
		}
		ultimo = time.Now()
		for _, b := range buf[:n] {
			if b == '\r' || b == '\n' {
				enviar()
			} else {
				texto = append(texto, b)
			}
		}
	}
}
