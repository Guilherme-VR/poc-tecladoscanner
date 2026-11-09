# Teclado x Scanner

Mostra cada leitura e de onde ela veio: **SCANNER 1**, **SCANNER 2** ou
**TECLADO** (qualquer aparelho USB que não seja um scanner cadastrado).
Funciona com scanner USB (que "digita" como teclado) e scanner serial (COMx).
Só escuta, nunca bloqueia nada.

- `agente/` — Go. Escuta os teclados/scanners USB (Windows: Raw Input, Linux:
  /dev/input) e escreve um JSON por tecla só com a origem:
  `{"tipo":"tecla","id":"..."}`. Também abre as portas seriais (9600 8N1) e
  escreve cada leitura completa: `{"tipo":"serial","id":"COM3","texto":"..."}`.
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
2. Clique em **Adicionar scanner** e faça uma leitura no scanner (USB ou
   serial). A origem dessa leitura vira o Scanner 1. Repita para o Scanner 2
   (máximo 2). O "x" remove um scanner.
3. Com a janela do app em foco, digite no teclado ou passe o leitor: cada
   leitura aparece como SCANNER 1, SCANNER 2 ou TECLADO. Leituras seriais
   aparecem mesmo com o app sem foco.

Portas seriais: durante o cadastro o agente abre todas (9600 8N1) para
descobrir de onde veio a leitura; fora dele, só as dos scanners seriais
cadastrados. Assim não segura a porta de outro equipamento (balança, pinpad).

## Linux

- Permissão, uma vez só por usuário: `sudo usermod -aG input $USER`. Não precisa
  sair da sessão: o agente ativa o grupo sozinho. Depois disso roda sem sudo.
- Scanner serial: o usuário precisa estar no grupo `dialout`
  (`sudo usermod -aG dialout $USER`, depois sair e entrar na sessão).
