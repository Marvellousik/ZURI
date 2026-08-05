# Single-word launch script for ZURI in PowerShell
$exePath = Join-Path $PSScriptRoot "zuri.exe"
if (Test-Path $exePath) {
    & $exePath @args
} else {
    Write-Host "Compiling ZURI binaries..." -ForegroundColor Yellow
    go build -o (Join-Path $PSScriptRoot "zuri.exe") ./cmd/zuri
    go build -o (Join-Path $PSScriptRoot "zuri-daemon.exe") ./cmd/daemon
    & $exePath @args
}
