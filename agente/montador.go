package main

import (
	"sync"

	"golang.org/x/text/unicode/norm"
)

// Junta acento (tecla morta) + letra, por aparelho, e emite o resultado.
// Ex: "´" seguido de "a" vira "á". Cada aparelho tem seu próprio acento pendente.

var combinantes = map[rune]rune{
	'´':  0x0301,
	'\'': 0x0301,
	'`':  0x0300,
	'~':  0x0303,
	'^':  0x0302,
	'¨':  0x0308,
	'"':  0x0308,
}

var (
	pendentesMu sync.Mutex
	pendentes   = map[string]rune{}
)

func emitirChar(d Dispositivo, r rune) {
	pendentesMu.Lock()
	morta, tem := pendentes[d.Disp]
	delete(pendentes, d.Disp)
	pendentesMu.Unlock()

	if !tem {
		emitirTexto(d, string(r))
		return
	}
	if r == ' ' {
		emitirTexto(d, string(morta))
		return
	}
	if c, ok := combinantes[morta]; ok {
		junto := []rune(norm.NFC.String(string([]rune{r, c})))
		if len(junto) == 1 {
			emitirTexto(d, string(junto))
			return
		}
	}
	emitirTexto(d, string([]rune{morta, r}))
}

func emitirMorta(d Dispositivo, morta rune) {
	pendentesMu.Lock()
	anterior, tem := pendentes[d.Disp]
	pendentes[d.Disp] = morta
	pendentesMu.Unlock()
	if tem {
		emitirTexto(d, string(anterior))
	}
}

func emitirEnter(d Dispositivo) {
	pendentesMu.Lock()
	morta, tem := pendentes[d.Disp]
	delete(pendentes, d.Disp)
	pendentesMu.Unlock()
	if tem {
		emitirTexto(d, string(morta))
	}
	emitir(Evento{Tipo: "enter", Disp: d.Disp, ID: d.ID, Nome: d.Nome})
}

func emitirApagar(d Dispositivo) {
	pendentesMu.Lock()
	_, tem := pendentes[d.Disp]
	delete(pendentes, d.Disp)
	pendentesMu.Unlock()
	if !tem {
		emitir(Evento{Tipo: "apagar", Disp: d.Disp, ID: d.ID, Nome: d.Nome})
	}
}

func emitirTexto(d Dispositivo, s string) {
	emitir(Evento{Tipo: "tecla", Disp: d.Disp, ID: d.ID, Nome: d.Nome, Texto: s})
}
