# Teclado x Scanner

Mostra o que é digitado e de onde veio: **TECLADO** (o aparelho identificado) ou
**SCANNER** (qualquer outro). A identificação é pelo **VID:PID** do aparelho.
Só escuta as teclas, nunca bloqueia nada.

- `agente/` — Go. Escuta os teclados (Windows: Raw Input, Linux: /dev/input) e
  escreve um JSON por tecla só com a origem: `{"tipo":"tecla","id":"VID:PID"}`.
  Não converte tecla em caractere.
- `app/` — Flutter. O caractere vem do próprio Flutter (o sistema já converte);
  cada tecla do app é casada, na ordem, com a tecla informada pelo agente, que
  diz se veio do TECLADO ou do SCANNER.

## Gerar o executável único

**Pelo GitHub (recomendado):** a cada push, o GitHub Actions
(`.github/workflows/build.yml`) gera os dois executáveis. Baixe em
Actions → execução → Artifacts. Também dá para rodar manualmente em
Actions → build → Run workflow.

**Na máquina:**

| Sistema | Comando | Resultado |
|---|---|---|
| Windows | `powershell -ExecutionPolicy Bypass -File scripts\build-windows.ps1` | `dist\teclado-scanner.exe` |
| Linux | `./scripts/build-linux.sh` | `dist/teclado-scanner` |

O build do Windows precisa ser feito no Windows (Flutter + Visual Studio com
"Desktop development with C++" + Go).

## Rodar em desenvolvimento

```bash
cd agente && go build -o ../app/build/agente .        # Linux
cd app && TECLADO_AGENTE=$PWD/build/agente flutter run -d linux
```

No Windows: `go build -o ..\app\build\agente.exe .` e
`$env:TECLADO_AGENTE="$PWD\build\agente.exe"; flutter run -d windows`.

Para ver só o agente no terminal: `go run . --agente`.

## Como testar

1. Abra o executável.
2. Clique em **Identificar teclado** e aperte uma tecla no teclado de verdade.
3. Com a janela do app em foco, digite no teclado ou passe o leitor: cada letra
   aparece como TECLADO ou SCANNER (a linha vai para "Leituras" no Enter).

## Linux

- Permissão, uma vez só por usuário: `sudo usermod -aG input $USER`. Não precisa
  sair da sessão: o agente ativa o grupo sozinho. Depois disso roda sem sudo.
