package main

import (
	"encoding/json"
	"os"
	"sync"
)

// Evento é escrito como uma linha JSON no stdout.
//
//	tipo:  "pronto" | "tecla" | "erro"
//	id:    VID:PID do aparelho de onde veio a tecla (ex: "046D:C31C")
//	nome:  nome do aparelho, quando o sistema informa
//	texto: mensagem de erro
//
// Não há conversão de tecla em caractere: o texto chega ao app pelo caminho
// normal do sistema. O agente só diz de qual aparelho veio cada tecla.
type Evento struct {
	Tipo  string `json:"tipo"`
	ID    string `json:"id,omitempty"`
	Nome  string `json:"nome,omitempty"`
	Texto string `json:"texto,omitempty"`
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
