// Agente que escuta todos os teclados da máquina (sem bloquear nada) e
// informa, para cada tecla, de qual aparelho (VID:PID) ela veio.
//
// Modos:
//   - "--agente": escuta as teclas e escreve um JSON por linha no stdout.
//   - sem argumentos: se o executável foi gerado com o app Flutter embutido
//     (build tag "pacote"), extrai e abre o app; senão, roda como agente.
package main

import (
	"bufio"
	"fmt"
	"io"
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

	// Quando o app Flutter fecha, o stdin do agente fecha junto: encerra.
	if modoAgente {
		go func() {
			io.Copy(io.Discard, bufio.NewReader(os.Stdin))
			os.Exit(0)
		}()
	}

	emitir(Evento{Tipo: "pronto"})
	if err := capturar(); err != nil {
		emitir(Evento{Tipo: "erro", Texto: err.Error()})
		os.Exit(1)
	}
}
