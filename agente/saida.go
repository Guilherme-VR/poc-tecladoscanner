package main

import (
	"encoding/json"
	"os"
	"sync"
)

// Evento é escrito como uma linha JSON no stdout.
//
//	tipo:  "pronto" | "tecla" | "enter" | "apagar" | "erro"
//	disp:  identifica a unidade física nesta sessão (separa o texto de 2 aparelhos iguais)
//	id:    VID:PID do aparelho (ex: "046D:C31C") - é o que define TECLADO x SCANNER
//	nome:  nome do aparelho, quando o sistema informa
//	texto: o caractere digitado (tipo "tecla") ou a mensagem (tipo "erro")
type Evento struct {
	Tipo  string `json:"tipo"`
	Disp  string `json:"disp,omitempty"`
	ID    string `json:"id,omitempty"`
	Nome  string `json:"nome,omitempty"`
	Texto string `json:"texto,omitempty"`
}

// Dispositivo é a origem de uma tecla.
type Dispositivo struct {
	Disp string
	ID   string
	Nome string
}

var (
	saidaMu  sync.Mutex
	saidaEnc = json.NewEncoder(os.Stdout)
)

func emitir(e Evento) {
	saidaMu.Lock()
	defer saidaMu.Unlock()
	saidaEnc.Encode(e)
}
