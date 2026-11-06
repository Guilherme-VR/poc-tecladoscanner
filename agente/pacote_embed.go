//go:build pacote

package main

import _ "embed"

// Build com "-tags pacote": o app Flutter (zip) vai dentro do executável.
// O arquivo pacote.zip é gerado pelos scripts em ../scripts.
//
//go:embed pacote.zip
var pacoteApp []byte
