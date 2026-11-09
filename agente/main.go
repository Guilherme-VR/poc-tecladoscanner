// Agente que escuta todos os teclados da máquina (sem bloquear nada) e as
// portas seriais. Para cada tecla informa de qual aparelho ela veio; para cada
// leitura serial informa a porta e o texto.
//
// Modos:
//   - "--agente": escuta as teclas e escreve um JSON por linha no stdout.
//   - sem argumentos: se o executável foi gerado com o app Flutter embutido
//     (build tag "pacote"), extrai e abre o app; senão, roda como agente.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	modoAgente := len(os.Args) > 1 && os.Args[1] == "--agente"

	if !modoAgente && len(pacoteApp) > 0 {
		if err := abrirApp(); err != nil {
			mostrarErro(fmt.Sprintf("Não foi possível abrir o app: %v", err))
			os.Exit(1)
		}
		return
	}

	prepararPermissao()

	// O app manda comandos pelo stdin, um JSON por linha. Quando o app fecha,
	// o stdin fecha junto: encerra.
	if modoAgente {
		go lerComandos()
	} else {
		definirPortas(true, nil) // rodando sozinho no terminal: escuta todas
	}

	emitir(Evento{Tipo: "pronto"})
	go capturarSeriais()
	if err := capturar(); err != nil {
		emitir(Evento{Tipo: "erro", Texto: err.Error()})
		os.Exit(1)
	}
}

// Comando do app para o agente.
//
//	{"cmd":"portas","todas":true}               cadastro: abre todas as seriais
//	{"cmd":"portas","lista":["COM3","COM4"]}    só as dos scanners cadastrados
type Comando struct {
	Cmd   string   `json:"cmd"`
	Todas bool     `json:"todas"`
	Lista []string `json:"lista"`
}

func lerComandos() {
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		var c Comando
		if json.Unmarshal(sc.Bytes(), &c) == nil && c.Cmd == "portas" {
			definirPortas(c.Todas, c.Lista)
		}
	}
	os.Exit(0)
}
