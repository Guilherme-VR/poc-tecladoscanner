#!/usr/bin/env bash
# Gera dist/teclado-scanner: um executável único (app Flutter + agente Go).
set -euo pipefail
cd "$(dirname "$0")/.."
RAIZ="$PWD"

echo "==> Flutter (release)"
(cd app && flutter build linux --release)
BUNDLE="$RAIZ/app/build/linux/x64/release/bundle"

echo "==> Compactando o app"
rm -f agente/pacote.zip
(cd "$BUNDLE" && zip -qr "$RAIZ/agente/pacote.zip" .)

echo "==> Go (executável único)"
mkdir -p dist
(cd agente && CGO_ENABLED=0 go build -tags pacote -ldflags "-s -w" -o "$RAIZ/dist/teclado-scanner" .)
rm -f agente/pacote.zip

echo "Pronto: dist/teclado-scanner"
