package main

import (
	"encoding/json"
	"os"
	"sync"
)

// Evento é escrito como uma linha JSON no stdout.
//
//	tipo:  "pronto" | "tecla" | "serial" | "aviso" | "erro"
//	id:    aparelho de origem (tecla: ID único do teclado/scanner USB;
//	       serial: a porta, ex: "COM3" ou "/dev/ttyACM0")
//	nome:  nome do aparelho, quando o sistema informa
//	texto: leitura serial completa, ou a mensagem de aviso/erro
//
// Teclas USB não são convertidas em caractere: o texto chega ao app pelo
// caminho normal do sistema; o agente só diz de qual aparelho veio cada tecla.
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
