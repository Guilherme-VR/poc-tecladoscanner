package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func capturarSaida(f func()) []Evento {
	var b bytes.Buffer
	saidaEnc = json.NewEncoder(&b)
	f()
	var evs []Evento
	dec := json.NewDecoder(&b)
	for dec.More() {
		var e Evento
		dec.Decode(&e)
		evs = append(evs, e)
	}
	return evs
}

func TestAcentos(t *testing.T) {
	d := Dispositivo{Disp: "x", ID: "046D:C31C"}
	casos := []struct {
		f    func()
		quer string
	}{
		{func() { emitirMorta(d, '´'); emitirChar(d, 'a') }, "á"},
		{func() { emitirMorta(d, '~'); emitirChar(d, 'A') }, "Ã"},
		{func() { emitirMorta(d, '^'); emitirChar(d, 'e') }, "ê"},
		{func() { emitirMorta(d, '´'); emitirChar(d, ' ') }, "´"},
		{func() { emitirMorta(d, '´'); emitirChar(d, 'x') }, "´x"},
		{func() { emitirChar(d, '7') }, "7"},
	}
	for _, c := range casos {
		s := ""
		for _, e := range capturarSaida(c.f) {
			s += e.Texto
		}
		if s != c.quer {
			t.Errorf("quer %q, veio %q", c.quer, s)
		}
	}
}

func TestDoisAparelhosNaoMisturamAcento(t *testing.T) {
	a := Dispositivo{Disp: "a"}
	b := Dispositivo{Disp: "b"}
	evs := capturarSaida(func() { emitirMorta(a, '´'); emitirChar(b, '1'); emitirChar(a, 'e') })
	if len(evs) != 2 || evs[0].Texto != "1" || evs[1].Texto != "é" {
		t.Errorf("veio %+v", evs)
	}
}
