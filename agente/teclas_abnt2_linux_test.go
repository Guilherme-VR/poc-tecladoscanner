package main

import "testing"

func TestABNT2(t *testing.T) {
	casos := []struct {
		cod             uint16
		shift, caps, ag bool
		quer            rune
		morta           bool
	}{
		{30, false, false, false, 'a', false},
		{30, true, false, false, 'A', false},
		{30, false, true, false, 'A', false},
		{30, true, true, false, 'a', false},
		{39, false, false, false, 'ç', false},
		{2, true, false, false, '!', false},
		{7, true, false, false, '¨', true},
		{26, false, false, false, '´', true},
		{16, false, false, true, '/', false},
		{82, false, false, false, '0', false},
	}
	for _, c := range casos {
		r, m, ok := teclaABNT2(c.cod, c.shift, c.caps, c.ag)
		if !ok || r != c.quer || m != c.morta {
			t.Errorf("cod %d: quer %q/%v veio %q/%v ok=%v", c.cod, c.quer, c.morta, r, m, ok)
		}
	}
}
