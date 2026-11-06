//go:build linux

package main

// Tabela de teclas ABNT2 (código evdev -> caractere).
// No Windows não precisa: o próprio sistema converte usando o layout ativo.

type tecla struct {
	normal, shift  rune
	mortaN, mortaS bool // é tecla morta (acento) sem/com shift
	letra          bool // Caps Lock afeta
}

var abnt2 = map[uint16]tecla{
	2: {normal: '1', shift: '!'}, 3: {normal: '2', shift: '@'}, 4: {normal: '3', shift: '#'},
	5: {normal: '4', shift: '$'}, 6: {normal: '5', shift: '%'}, 7: {normal: '6', shift: '¨', mortaS: true},
	8: {normal: '7', shift: '&'}, 9: {normal: '8', shift: '*'}, 10: {normal: '9', shift: '('},
	11: {normal: '0', shift: ')'}, 12: {normal: '-', shift: '_'}, 13: {normal: '=', shift: '+'},
	26: {normal: '´', shift: '`', mortaN: true, mortaS: true},
	27: {normal: '[', shift: '{'},
	39: {normal: 'ç', shift: 'Ç', letra: true},
	40: {normal: '~', shift: '^', mortaN: true, mortaS: true},
	41: {normal: '\'', shift: '"'},
	43: {normal: ']', shift: '}'},
	51: {normal: ',', shift: '<'}, 52: {normal: '.', shift: '>'}, 53: {normal: ';', shift: ':'},
	57: {normal: ' ', shift: ' '},
	86: {normal: '\\', shift: '|'},
	89: {normal: '/', shift: '?'},
	// teclado numérico
	55: {normal: '*', shift: '*'}, 71: {normal: '7', shift: '7'}, 72: {normal: '8', shift: '8'},
	73: {normal: '9', shift: '9'}, 74: {normal: '-', shift: '-'}, 75: {normal: '4', shift: '4'},
	76: {normal: '5', shift: '5'}, 77: {normal: '6', shift: '6'}, 78: {normal: '+', shift: '+'},
	79: {normal: '1', shift: '1'}, 80: {normal: '2', shift: '2'}, 81: {normal: '3', shift: '3'},
	82: {normal: '0', shift: '0'}, 83: {normal: ',', shift: ','}, 98: {normal: '/', shift: '/'},
	121: {normal: '.', shift: '.'},
}

var altGrABNT2 = map[uint16]rune{
	2: '¹', 3: '²', 4: '³', 5: '£', 6: '¢', 7: '¬', 13: '§',
	16: '/', 17: '?', 18: '°', 27: 'ª', 43: 'º',
}

func init() {
	linhas := []struct {
		inicio uint16
		letras string
	}{{16, "qwertyuiop"}, {30, "asdfghjkl"}, {44, "zxcvbnm"}}
	for _, l := range linhas {
		for i, c := range l.letras {
			abnt2[l.inicio+uint16(i)] = tecla{normal: c, shift: c - 32, letra: true}
		}
	}
}

func teclaABNT2(cod uint16, shift, caps, altGr bool) (r rune, morta, ok bool) {
	if altGr {
		r, ok = altGrABNT2[cod]
		return r, false, ok
	}
	t, ok := abnt2[cod]
	if !ok {
		return 0, false, false
	}
	usaShift := shift
	if t.letra && caps {
		usaShift = !usaShift
	}
	if usaShift {
		return t.shift, t.mortaS, true
	}
	return t.normal, t.mortaN, true
}
