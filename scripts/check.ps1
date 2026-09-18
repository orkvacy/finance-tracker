# Definition of Done yang bisa dijalankan.
#
# Jalankan SEBELUM push:   .\scripts\check.ps1
#
# Daftar centang di docs/SCRUM.md gampang dilewati saat buru-buru. Skrip ini
# tidak bisa. Setiap pemeriksaan di sini pernah menangkap bug sungguhan di
# proyek ini, bukan formalitas.
#
# CATATAN: berkas ini sengaja ASCII murni. Windows PowerShell 5.1 membaca .ps1
# sebagai ANSI bila tidak ber-BOM, sehingga karakter non-ASCII merusak parser.

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$script:gagal = @()
$script:no = 0

function Langkah($nama) {
    $script:no++
    Write-Host ""
    Write-Host ("[{0}] {1}" -f $script:no, $nama) -ForegroundColor Cyan
}
function Lolos($pesan) { Write-Host "    OK    $pesan" -ForegroundColor Green }
function Gagal($pesan) {
    Write-Host "    GAGAL $pesan" -ForegroundColor Red
    $script:gagal += $pesan
}

Write-Host "=== Cek sebelum push - Finance Track ===" -ForegroundColor White

# ---------------------------------------------------------------- 1
Langkah "Kompilasi Go"
go build ./...
if ($LASTEXITCODE -eq 0) { Lolos "go build bersih" } else { Gagal "go build error" }

# ---------------------------------------------------------------- 2
Langkah "Analisis statis"
go vet ./...
if ($LASTEXITCODE -eq 0) { Lolos "go vet bersih" } else { Gagal "go vet menemukan masalah" }

# ---------------------------------------------------------------- 3
Langkah "Test"
go test ./...
if ($LASTEXITCODE -eq 0) { Lolos "semua test hijau" } else { Gagal "ada test yang gagal" }

# ---------------------------------------------------------------- 4
# Tanpa .gitkeep, //go:embed all:dist gagal pada clone baru - dan itu baru
# ketahuan saat build pertama di Coolify, bukan di laptop.
Langkah "Penjaga build clone baru"
if (Test-Path 'web\dist\.gitkeep') {
    Lolos "web/dist/.gitkeep ada"
} else {
    Gagal "web/dist/.gitkeep hilang - clone baru tidak akan bisa di-build"
}

# ---------------------------------------------------------------- 5
Langkah "Build frontend"
npm --prefix web run build
if ($LASTEXITCODE -eq 0) { Lolos "vite build berhasil" } else { Gagal "vite build error" }

# ---------------------------------------------------------------- 6
# NFR-11: bundle awal <= 60 KB gzip. Diukur, tidak diasumsikan. Inilah yang
# dulu mengungkap React 19 memakan 72 KB untuk aplikasi yang praktis kosong.
# CSS ikut dihitung: ia render-blocking, jadi sama-sama menentukan waktu
# tampil pertama. Mengukur JS saja membuat anggaran ini bohong.
Langkah "Anggaran bundle (NFR-11: maks 60 KB gzip, JS + CSS)"
$total = 0
Get-ChildItem 'web\dist\assets\*.js', 'web\dist\assets\*.css' -ErrorAction SilentlyContinue | ForEach-Object {
    $raw = [System.IO.File]::ReadAllBytes($_.FullName)
    $mem = New-Object System.IO.MemoryStream
    $gz  = New-Object System.IO.Compression.GZipStream($mem, [System.IO.Compression.CompressionLevel]::Optimal)
    $gz.Write($raw, 0, $raw.Length)
    $gz.Close()
    $kb = [math]::Round($mem.ToArray().Length / 1024, 2)
    $total += $kb
    Write-Host ("          {0}  {1} KB gzip" -f $_.Name, $kb)
}
if ($total -eq 0) {
    Gagal "tidak ada bundle JS ditemukan"
} elseif ($total -le 60) {
    $sisa = [math]::Round(60 - $total, 2)
    Lolos ("total {0} KB gzip, sisa anggaran {1} KB" -f $total, $sisa)
} else {
    Gagal ("total {0} KB gzip, MELEWATI anggaran 60 KB" -f $total)
}

# ---------------------------------------------------------------- 7
Langkah "Binary dengan SPA ter-embed"
go build -o bin\server.exe .\cmd\server
if ($LASTEXITCODE -eq 0) {
    $mb = [math]::Round((Get-Item 'bin\server.exe').Length / 1MB, 1)
    Lolos ("binary terbentuk ({0} MB)" -f $mb)
} else {
    Gagal "build binary error"
}

# ---------------------------------------------------------------- 8
# Membuktikan aplikasinya benar-benar hidup, bukan sekadar bisa dikompilasi.
# Database uji dibuang setelahnya supaya data sungguhan tidak tersentuh.
Langkah "Uji jalan (health check menyentuh database)"
$dirUji = Join-Path $env:TEMP ("ftcheck-" + [guid]::NewGuid().ToString('N'))
$env:DB_PATH = Join-Path $dirUji 'app.db'
$env:ATTACHMENT_DIR = Join-Path $dirUji 'attachments'
$env:ADDR = ':8471'
$proc = Start-Process -FilePath 'bin\server.exe' -PassThru -WindowStyle Hidden

$sehat = $false
foreach ($i in 1..20) {
    Start-Sleep -Milliseconds 250
    try {
        $r = Invoke-WebRequest 'http://127.0.0.1:8471/healthz' -UseBasicParsing -TimeoutSec 2
        if ($r.StatusCode -eq 200) { $sehat = $true; break }
    } catch { }
}
if ($sehat) {
    try {
        $spa = Invoke-WebRequest 'http://127.0.0.1:8471/' -UseBasicParsing -TimeoutSec 2
        if ($spa.Content -match '<div id="root">') {
            Lolos "healthz OK dan SPA tersaji"
        } else {
            Gagal "SPA tidak tersaji dengan benar"
        }
    } catch {
        Gagal "SPA tidak bisa diambil"
    }
} else {
    Gagal "server tidak pernah sehat dalam 5 detik"
}

if ($proc -and -not $proc.HasExited) { Stop-Process -Id $proc.Id -Force }
Remove-Item $dirUji -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item Env:DB_PATH, Env:ATTACHMENT_DIR, Env:ADDR -ErrorAction SilentlyContinue

# ---------------------------------------------------------------- hasil
Write-Host ""
Write-Host ("=" * 52)
if ($script:gagal.Count -eq 0) {
    Write-Host "SEMUA LOLOS - aman di-push" -ForegroundColor Green
    Write-Host ""
    Write-Host "Sisa DoD yang tidak bisa diperiksa mesin (docs/SCRUM.md):"
    Write-Host "  - Ter-deploy dan dibuka dari iPhone 11 SUNGGUHAN"
    Write-Host "  - Data selamat melewati satu redeploy"
    exit 0
} else {
    Write-Host ("{0} PEMERIKSAAN GAGAL - jangan push dulu" -f $script:gagal.Count) -ForegroundColor Red
    $script:gagal | ForEach-Object { Write-Host "  - $_" -ForegroundColor Red }
    exit 1
}
