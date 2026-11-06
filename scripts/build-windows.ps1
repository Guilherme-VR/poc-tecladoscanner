# Gera dist\teclado-scanner.exe: um executável único (app Flutter + agente Go).
# Rodar no Windows, com Flutter (Visual Studio "Desktop development with C++") e Go instalados.
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")
$raiz = (Get-Location).Path

Write-Host "==> Flutter (release)"
Push-Location app
flutter build windows --release
if ($LASTEXITCODE -ne 0) { throw "flutter build falhou" }
Pop-Location
$bundle = Join-Path $raiz "app\build\windows\x64\runner\Release"

# Runtime do Visual C++ junto, para rodar em máquinas sem ele instalado.
foreach ($dll in "msvcp140.dll", "vcruntime140.dll", "vcruntime140_1.dll") {
    $origem = Join-Path $env:WINDIR "System32\$dll"
    if (Test-Path $origem) { Copy-Item $origem $bundle -Force }
}

Write-Host "==> Compactando o app"
$zip = Join-Path $raiz "agente\pacote.zip"
if (Test-Path $zip) { Remove-Item $zip }
Compress-Archive -Path (Join-Path $bundle "*") -DestinationPath $zip

Write-Host "==> Go (executável único)"
New-Item -ItemType Directory -Force dist | Out-Null
Push-Location agente
$env:CGO_ENABLED = "0"
go build -tags pacote -ldflags "-s -w -H windowsgui" -o (Join-Path $raiz "dist\teclado-scanner.exe") .
if ($LASTEXITCODE -ne 0) { throw "go build falhou" }
Pop-Location
Remove-Item $zip

Write-Host "Pronto: dist\teclado-scanner.exe"
