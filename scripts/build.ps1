param (
    [ValidateSet('all', 'modern', 'lite')]
    [string]$Target = 'all',

    [string]$OutDir = 'dist'
)

$ErrorActionPreference = 'Stop'

Write-Host '========================================================' -ForegroundColor Cyan
Write-Host '         RESPECT DESKTOP — AUTOMATED BUILDER            ' -ForegroundColor Cyan
Write-Host '========================================================' -ForegroundColor Cyan
Write-Host "Target  : $Target" -ForegroundColor Yellow
Write-Host "Output  : $OutDir" -ForegroundColor Yellow
Write-Host ''

$rcedit = Join-Path $PSScriptRoot '..\assets\rcedit.exe'
$icon   = Join-Path $PSScriptRoot '..\assets\respect-icon.ico'

# 1. Build Respect Lite (v49)
if ($Target -eq 'all' -or $Target -eq 'lite') {
    Write-Host '[1/2] Membangun Respect Lite (respect-lite.exe v49)...' -ForegroundColor Green
    $liteDir = Join-Path $OutDir 'respect-lite'
    if (!(Test-Path $liteDir)) { New-Item -ItemType Directory -Path $liteDir -Force | Out-Null }

    $liteExe = Join-Path $liteDir 'respect-lite.exe'
    go build -ldflags='-s -w -H windowsgui' -o $liteExe .
    
    if ((Test-Path $rcedit) -and (Test-Path $icon)) {
        & $rcedit $liteExe --set-icon $icon
    }

    $liteBytes = (Get-Item $liteExe).Length
    $liteMB = [math]::Round(($liteBytes / 1048576), 2)
    Write-Host "  -> Selesai: $liteExe ($liteMB MB standalone)" -ForegroundColor Green
}

# 2. Build Respect Modern (v132)
if ($Target -eq 'all' -or $Target -eq 'modern') {
    Write-Host '[2/2] Membangun Respect Modern (respect.exe v132)...' -ForegroundColor Green
    $modernDir = Join-Path $OutDir 'respect'
    if (!(Test-Path $modernDir)) { New-Item -ItemType Directory -Path $modernDir -Force | Out-Null }

    $modernExe = Join-Path $modernDir 'respect.exe'
    go build -tags v132 -ldflags='-s -w -H windowsgui' -o $modernExe .

    if ((Test-Path $rcedit) -and (Test-Path $icon)) {
        & $rcedit $modernExe --set-icon $icon
    }

    $dllSource = ''
    if (Test-Path 'test-v132\blink.dll') {
        $dllSource = 'test-v132\blink.dll'
    } elseif (Test-Path '_research\miniblink132\mb132_x64.dll') {
        $dllSource = '_research\miniblink132\mb132_x64.dll'
    }

    if ($dllSource -ne '') {
        Copy-Item -Path $dllSource -Destination (Join-Path $modernDir 'blink.dll') -Force
        Write-Host "  -> blink.dll disalin dari $dllSource" -ForegroundColor DarkGray
    } else {
        Write-Warning 'blink.dll tidak ditemukan di test-v132 atau _research.'
    }

    $modBytes = (Get-Item $modernExe).Length
    $modMB = [math]::Round(($modBytes / 1048576), 2)
    Write-Host "  -> Selesai: $modernExe ($modMB MB slim binary)" -ForegroundColor Green
}

Write-Host ''
Write-Host 'Semua target berhasil dibangun.' -ForegroundColor Cyan
