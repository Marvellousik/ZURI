# Start Zuri Daemon & CLI / TUI dashboard (PowerShell)

Param(
    [switch]$Docker = $false,
    [switch]$TUI = $false
)

Write-Host "===========================================" -ForegroundColor Cyan
Write-Host " Starting Zuri Engineering Memory System   " -ForegroundColor Cyan
Write-Host "===========================================" -ForegroundColor Cyan

# 0. Clean up stale/orphaned background instances holding file locks or ports
Write-Host "[0/3] Cleaning up stale background processes..." -ForegroundColor Gray
Get-Process -Name zuri-daemon, postgres -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

# 1. Compile native binaries
Write-Host "[1/3] Compiling native ZURI binaries (zuri.exe & zuri-daemon.exe)..." -ForegroundColor Yellow
go build -o zuri.exe ./cmd/zuri
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go build -o zuri-daemon.exe ./cmd/daemon
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

if ($Docker) {
    Write-Host "[2/3] Launching pgvector + Zuri Daemon via Docker Compose..." -ForegroundColor Green
    docker compose up -d --build
} else {
    Write-Host "[2/3] Starting Zuri Daemon natively..." -ForegroundColor Green
    Start-Process powershell -ArgumentList "-NoExit", "-Command", ".\zuri-daemon.exe"
}

# Wait for daemon startup (embedded Postgres takes 2-4 seconds on launch)
Write-Host "Waiting for Zuri Daemon & Postgres startup..." -ForegroundColor Gray
$maxRetries = 10
$started = $false
for ($i = 1; $i -le $maxRetries; $i++) {
    Start-Sleep -Seconds 1
    $res = .\zuri.exe status 2>&1
    if ($LASTEXITCODE -eq 0) {
        $started = $true
        break
    }
}

if (-not $started) {
    Write-Host "[!] Daemon startup timed out. Check '.\zuri-daemon.exe' logs." -ForegroundColor Red
} else {
    Write-Host "[3/3] Daemon health verified." -ForegroundColor Green
}

if ($TUI) {
    Write-Host "Launching ZURI Interactive TUI..." -ForegroundColor Cyan
    .\zuri.exe tui
} else {
    Write-Host "ZURI Daemon is running. Use '.\zuri.exe help' or '.\zuri.exe tui' to interact." -ForegroundColor Green
}
