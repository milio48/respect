param (
    [ValidateSet('all', 'modern', 'lite')]
    [string]$Target = 'all',

    [string]$OutDir = 'dist',

    [string]$Version = '1.0.0',

    [switch]$Embed
)

$ErrorActionPreference = 'Stop'

Write-Host '========================================================' -ForegroundColor Cyan
Write-Host '         RESPECT DESKTOP — AUTOMATED BUILDER            ' -ForegroundColor Cyan
Write-Host '========================================================' -ForegroundColor Cyan
Write-Host "Target  : $Target" -ForegroundColor Yellow
Write-Host "Output  : $OutDir" -ForegroundColor Yellow
Write-Host "Version : $Version" -ForegroundColor Yellow
Write-Host ''

# Versi PE wajib 4 segmen numerik (contoh: 1.0.0 -> 1.0.0.0, 1.0.0-beta -> 1.0.0.0)
$numeric     = @($Version -split '[^0-9]' | Where-Object { $_ -ne '' })
$fileVersion = (@($numeric + @('0', '0', '0', '0'))[0..3]) -join '.'
$prodVersion = (@($numeric + @('0', '0', '0'))[0..2]) -join '.'
$ldVersion   = "-X respect-app/internal/version.AppVersion=$Version"

$rcedit   = Join-Path $PSScriptRoot '..\assets\rcedit.exe'
$iconLite = Join-Path $PSScriptRoot '..\assets\respect-lite.ico'
$iconFull = Join-Path $PSScriptRoot '..\assets\respect-full.ico'

$totalSteps = if ($Target -eq 'all') { 2 } else { 1 }
$step = 0
$built = @()

# 1. Build Respect Lite (v49)
if ($Target -eq 'all' -or $Target -eq 'lite') {
    $step++
    Write-Host "[$step/$totalSteps] Membangun Respect Lite (respect-lite.exe v49)..." -ForegroundColor Green
    $liteDir = Join-Path $OutDir 'respect-lite'
    if (!(Test-Path $liteDir)) { New-Item -ItemType Directory -Path $liteDir -Force | Out-Null }

    $liteExe = Join-Path $liteDir 'respect-lite.exe'
    go build -ldflags="-s -w -H windowsgui $ldVersion" -o $liteExe .
    if ($LASTEXITCODE -ne 0) { throw "go build Respect Lite gagal (exit code $LASTEXITCODE)" }
    
    if ((Test-Path $rcedit) -and (Test-Path $iconLite)) {
        & $rcedit $liteExe `
            --set-icon $iconLite `
            --set-product-version $prodVersion `
            --set-file-version $fileVersion `
            --set-version-string "ProductName" "Respect Desktop Lite" `
            --set-version-string "FileDescription" "Respect Desktop Builder & Runner (Lite Engine)" `
            --set-version-string "CompanyName" "milio48" `
            --set-version-string "LegalCopyright" "Copyright (c) 2026 milio48" `
            --set-version-string "OriginalFilename" "respect-lite.exe"
        if ($LASTEXITCODE -ne 0) { throw "rcedit gagal untuk $liteExe (exit code $LASTEXITCODE)" }
    } else {
        Write-Warning 'rcedit.exe atau respect-lite.ico tidak ditemukan; icon & metadata dilewati.'
    }

    $liteBytes = (Get-Item $liteExe).Length
    $liteMB = [math]::Round(($liteBytes / 1048576), 2)
    Write-Host "  -> Selesai: $liteExe ($liteMB MB standalone)" -ForegroundColor Green
    $built += 'respect-lite.exe'
}

# 2. Build Respect Modern (v132)
if ($Target -eq 'all' -or $Target -eq 'modern') {
    $step++
    Write-Host "[$step/$totalSteps] Membangun Respect Modern (respect.exe v132)..." -ForegroundColor Green
    $modernDir = Join-Path $OutDir 'respect'
    if (!(Test-Path $modernDir)) { New-Item -ItemType Directory -Path $modernDir -Force | Out-Null }

    $modernExe = Join-Path $modernDir 'respect.exe'
    $tags = if ($Embed) { "v132,embed132" } else { "v132" }
    go build -tags $tags -ldflags="-s -w -H windowsgui $ldVersion" -o $modernExe .
    if ($LASTEXITCODE -ne 0) { throw "go build Respect Modern gagal (exit code $LASTEXITCODE)" }

    if ((Test-Path $rcedit) -and (Test-Path $iconFull)) {
        & $rcedit $modernExe `
            --set-icon $iconFull `
            --set-product-version $prodVersion `
            --set-file-version $fileVersion `
            --set-version-string "ProductName" "Respect Desktop" `
            --set-version-string "FileDescription" "Respect Desktop Builder & Runner (Modern Engine)" `
            --set-version-string "CompanyName" "milio48" `
            --set-version-string "LegalCopyright" "Copyright (c) 2026 milio48" `
            --set-version-string "OriginalFilename" "respect.exe"
        if ($LASTEXITCODE -ne 0) { throw "rcedit gagal untuk $modernExe (exit code $LASTEXITCODE)" }
    } else {
        Write-Warning 'rcedit.exe atau respect-full.ico tidak ditemukan; icon & metadata dilewati.'
    }

    if (-not $Embed) {
        $dllSource = ''
        if (Test-Path 'assets\blink.dll') {
            $dllSource = 'assets\blink.dll'
        } elseif (Test-Path 'test-v132\blink.dll') {
            $dllSource = 'test-v132\blink.dll'
        } elseif (Test-Path '_research\miniblink132\mb132_x64.dll') {
            $dllSource = '_research\miniblink132\mb132_x64.dll'
        }

        $targetDll = Join-Path $modernDir 'blink.dll'
        if ($dllSource -ne '') {
            if (-not (Test-Path $targetDll)) {
                Copy-Item -Path $dllSource -Destination $targetDll -Force
                Write-Host "  -> blink.dll disalin dari $dllSource" -ForegroundColor DarkGray
            } else {
                try {
                    Copy-Item -Path $dllSource -Destination $targetDll -Force -ErrorAction Stop
                    Write-Host "  -> blink.dll diperbarui dari $dllSource" -ForegroundColor DarkGray
                } catch {
                    Write-Host "  -> blink.dll sudah ada (terkunci oleh proses aktif, dipertahankan)" -ForegroundColor DarkGray
                }
            }
        } else {
            Write-Warning 'blink.dll tidak ditemukan di assets, test-v132, atau _research. Binary slim tidak akan berjalan tanpa blink.dll — gunakan -Embed untuk single-file mandiri.'
        }
    }

    $modBytes = (Get-Item $modernExe).Length
    $modMB = [math]::Round(($modBytes / 1048576), 2)
    $typeDesc = if ($Embed) { "standalone single-file" } else { "slim binary" }
    Write-Host "  -> Selesai: $modernExe ($modMB MB $typeDesc)" -ForegroundColor Green
    $built += 'respect.exe'
}

Write-Host ''
Write-Host "Selesai. Target terbangun: $($built -join ', ')" -ForegroundColor Cyan
