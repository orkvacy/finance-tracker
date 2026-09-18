# Menjalankan aplikasi untuk dicoba di lokal.
#
#   .\scripts\dev.ps1           build penuh, satu binary  (seperti produksi)
#   .\scripts\dev.ps1 -Watch    hot reload untuk ngoding  (Vite + Go)
#   .\scripts\dev.ps1 -Bersih   mulai dari database kosong
#
# Mode default menjalankan artefak yang SAMA dengan yang di-deploy: satu binary
# berisi API dan SPA. Itu yang harus dipakai saat mengecek di iPhone, karena
# mode -Watch menyajikan berkas lewat Vite, bukan lewat Go.
#
# CATATAN: ASCII murni. Windows PowerShell 5.1 membaca .ps1 sebagai ANSI bila
# tidak ber-BOM, sehingga karakter non-ASCII merusak parser.

param(
    [switch]$Watch,
    [switch]$Bersih
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# Alamat LAN dipilih dari adapter yang benar-benar terhubung, bukan adapter
# virtual VirtualBox/WSL yang tidak bisa dijangkau HP.
$lan = (Get-NetIPAddress -AddressFamily IPv4 |
        Where-Object {
            $_.IPAddress -notlike '127.*' -and
            $_.IPAddress -notlike '169.254.*' -and
            $_.InterfaceAlias -notmatch 'vEthernet|VirtualBox|Loopback|Hyper-V'
        } |
        Sort-Object -Property @{ Expression = { $_.InterfaceAlias -match 'Wi-Fi' } } -Descending |
        Select-Object -First 1).IPAddress

if ($Bersih) {
    Write-Host "Menghapus database lokal..." -ForegroundColor Yellow
    Remove-Item 'data' -Recurse -Force -ErrorAction SilentlyContinue
}

if ($Watch) {
    Write-Host "Mode hot reload" -ForegroundColor Cyan
    Write-Host "  Go   : http://localhost:8080  (API)"
    Write-Host "  Vite : http://localhost:5173  (buka yang ini)"
    if ($lan) { Write-Host "  HP   : http://${lan}:5173" }
    Write-Host ""
    Write-Host "Ctrl+C untuk berhenti. Backend jalan di jendela terpisah." -ForegroundColor DarkGray
    Write-Host ""

    Start-Process powershell -ArgumentList '-NoExit', '-Command', "Set-Location '$root'; `$env:ADDR=':8080'; go run ./cmd/server"
    Start-Sleep -Seconds 2
    npm --prefix web run dev -- --host
    exit
}

Write-Host "Build frontend..." -ForegroundColor Cyan
npm --prefix web run build
if ($LASTEXITCODE -ne 0) { Write-Host "Build frontend gagal." -ForegroundColor Red; exit 1 }

Write-Host "Build binary..." -ForegroundColor Cyan
go build -o bin\server.exe .\cmd\server
if ($LASTEXITCODE -ne 0) { Write-Host "Build Go gagal." -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "=================================================="
Write-Host "  Laptop : http://localhost:8080" -ForegroundColor Green
if ($lan) {
    Write-Host "  iPhone : http://${lan}:8080" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Pastikan iPhone di Wi-Fi yang sama." -ForegroundColor DarkGray
    Write-Host "  Windows akan meminta izin firewall saat pertama kali -" -ForegroundColor DarkGray
    Write-Host "  pilih 'Private networks'. Tanpa itu HP tidak bisa konek." -ForegroundColor DarkGray
}
Write-Host "=================================================="
Write-Host ""
Write-Host "Ctrl+C untuk berhenti." -ForegroundColor DarkGray
Write-Host ""

$env:ADDR = ':8080'
$env:DB_PATH = 'data\app.db'
$env:ATTACHMENT_DIR = 'data\attachments'
& .\bin\server.exe
