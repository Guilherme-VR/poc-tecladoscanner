package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// abrirApp extrai o app Flutter embutido para a pasta de cache do usuário
// (só na primeira vez de cada versão) e abre o app. O app chama este mesmo
// executável com "--agente" (caminho passado na variável TECLADO_AGENTE).
func abrirApp() error {
	soma := sha256.Sum256(pacoteApp)
	cache, err := os.UserCacheDir()
	if err != nil {
		cache = os.TempDir()
	}
	pasta := filepath.Join(cache, "teclado-scanner", hex.EncodeToString(soma[:])[:12])

	if _, err := os.Stat(filepath.Join(pasta, ".ok")); err != nil {
		if err := extrair(pasta); err != nil {
			return err
		}
	}

	nomeApp := "teclado_scanner"
	if runtime.GOOS == "windows" {
		nomeApp += ".exe"
	}
	eu, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(filepath.Join(pasta, nomeApp))
	cmd.Dir = pasta
	cmd.Env = append(os.Environ(), "TECLADO_AGENTE="+eu)
	return cmd.Start()
}

func extrair(destino string) error {
	tmp := destino + ".tmp"
	os.RemoveAll(tmp)

	zr, err := zip.NewReader(bytes.NewReader(pacoteApp), int64(len(pacoteApp)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		// Compress-Archive do Windows pode gravar caminhos com "\".
		nome := strings.ReplaceAll(f.Name, "\\", "/")
		alvo := filepath.Join(tmp, filepath.FromSlash(nome))
		if !strings.HasPrefix(alvo, filepath.Clean(tmp)+string(os.PathSeparator)) {
			return fmt.Errorf("caminho inválido no pacote: %s", f.Name)
		}
		if strings.HasSuffix(nome, "/") {
			os.MkdirAll(alvo, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(alvo), 0o755); err != nil {
			return err
		}
		modo := f.Mode().Perm()
		if modo == 0 {
			modo = 0o644
		}
		if runtime.GOOS != "windows" {
			modo |= 0o100 // garante permissão de execução no Linux
		}
		if err := gravar(f, alvo, modo); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(tmp, ".ok"), nil, 0o644); err != nil {
		return err
	}
	os.RemoveAll(destino)
	return os.Rename(tmp, destino)
}

func gravar(f *zip.File, alvo string, modo os.FileMode) error {
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := os.OpenFile(alvo, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, modo)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}
