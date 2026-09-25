param (
    [ValidateSet('all', 'modern', 'lite', 'lite-x86')]
    [string]$Target = 'all',

    [string]$OutDir = 'dist',

    [string]$Version = '1.0.0',

    [switch]$Embed,

    [switch]$BuildDemo
)

$ErrorActionPreference = 'Stop'

Write-Host '========================================================' -ForegroundColor Cyan
Write-Host '         RESPECT DESKTOP - AUTOMATED BUILDER            ' -ForegroundColor Cyan
Write-Host '========================================================' -ForegroundColor Cyan
Write-Host "Target  : $Target" -ForegroundColor Yellow
Write-Host "Output  : $OutDir" -ForegroundColor Yellow
Write-Host "Version : $Version" -ForegroundColor Yellow
Write-Host "Embed   : $(if ($Embed) { 'Yes (ZSTD Standalone Single-File)' } else { 'No (Slim Binary)' })" -ForegroundColor Yellow
Write-Host "Demo    : $(if ($BuildDemo) { 'Yes (Build demo-stress-testing executables)' } else { 'Auto (When Target=all and Embed)' })" -ForegroundColor Yellow
Write-Host ''

# Versi PE wajib 4 segmen numerik (contoh: 1.0.0 -> 1.0.0.0, 1.0.0-beta -> 1.0.0.0)
$numeric     = @($Version -split '[^0-9]' | Where-Object { $_ -ne '' })
$fileVersion = (@($numeric + @('0', '0', '0', '0'))[0..3]) -join '.'
$prodVersion = (@($numeric + @('0', '0', '0'))[0..2]) -join '.'
$ldVersion   = "-X respect-app/internal/version.AppVersion=$Version"

$rcedit     = Join-Path $PSScriptRoot '..\assets\rcedit.exe'
$iconLite   = Join-Path $PSScriptRoot '..\assets\respect-lite.ico'
$iconFull   = Join-Path $PSScriptRoot '..\assets\respect-full.ico'
$iconStress = Join-Path $PSScriptRoot '..\assets\respect-stress.ico'
$stressDir  = Join-Path $PSScriptRoot '..\stress-testing'

$totalSteps = if ($Target -eq 'all') { 3 } else { 1 }
$step = 0
$built = @()

# 1. Build Respect Lite (v49 64-bit)
if ($Target -eq 'all' -or $Target -eq 'lite') {
    $step++
    Write-Host "[$step/$totalSteps] Membangun Respect Lite (respect-lite.exe 64-bit)..." -ForegroundColor Green
    $liteDir = Join-Path $OutDir 'respect-lite'
    if (!(Test-Path $liteDir)) { New-Item -ItemType Directory -Path $liteDir -Force | Out-Null }

    $liteExe = Join-Path $liteDir 'respect-lite.exe'
    go build -ldflags="-s -w -H windowsgui $ldVersion" -o $liteExe .
    if ($LASTEXITCODE -ne 0) { throw "go build Respect Lite (64-bit) gagal (exit code $LASTEXITCODE)" }
    
    if ((Test-Path $rcedit) -and (Test-Path $iconLite)) {
        & $rcedit $liteExe `
            --set-icon $iconLite `
            --set-product-version $prodVersion `
            --set-file-version $fileVersion `
            --set-version-string "ProductName" "Respect Desktop Lite" `
            --set-version-string "FileDescription" "Respect Desktop Builder & Runner (Lite Engine 64-bit)" `
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

# 1b. Build Respect Lite x86 (v49 32-bit untuk Windows lawas)
if ($Target -eq 'all' -or $Target -eq 'lite-x86') {
    $step++
    Write-Host "[$step/$totalSteps] Membangun Respect Lite x86 (respect-lite-x86.exe 32-bit)..." -ForegroundColor Green
    $liteDir = Join-Path $OutDir 'respect-lite'
    if (!(Test-Path $liteDir)) { New-Item -ItemType Directory -Path $liteDir -Force | Out-Null }

    $liteX86Exe = Join-Path $liteDir 'respect-lite-x86.exe'
    $env:GOARCH = "386"
    try {
        go build -ldflags="-s -w -H windowsgui $ldVersion" -o $liteX86Exe .
        if ($LASTEXITCODE -ne 0) { throw "go build Respect Lite x86 (32-bit) gagal (exit code $LASTEXITCODE)" }
    } finally {
        $env:GOARCH = "amd64"
    }

    if ((Test-Path $rcedit) -and (Test-Path $iconLite)) {
        & $rcedit $liteX86Exe `
            --set-icon $iconLite `
            --set-product-version $prodVersion `
            --set-file-version $fileVersion `
            --set-version-string "ProductName" "Respect Desktop Lite (32-bit)" `
            --set-version-string "FileDescription" "Respect Desktop Builder & Runner (Lite Engine 32-bit)" `
            --set-version-string "CompanyName" "milio48" `
            --set-version-string "LegalCopyright" "Copyright (c) 2026 milio48" `
            --set-version-string "OriginalFilename" "respect-lite-x86.exe"
        if ($LASTEXITCODE -ne 0) { throw "rcedit gagal untuk $liteX86Exe (exit code $LASTEXITCODE)" }
    } else {
        Write-Warning 'rcedit.exe atau respect-lite.ico tidak ditemukan; icon & metadata dilewati.'
    }

    $liteX86Bytes = (Get-Item $liteX86Exe).Length
    $liteX86MB = [math]::Round(($liteX86Bytes / 1048576), 2)
    Write-Host "  -> Selesai: $liteX86Exe ($liteX86MB MB standalone 32-bit)" -ForegroundColor Green
    $built += 'respect-lite-x86.exe'
}

# 2. Build Respect Modern (v132)
if ($Target -eq 'all' -or $Target -eq 'modern') {
    $step++
    Write-Host "[$step/$totalSteps] Membangun Respect Modern (respect.exe v132)..." -ForegroundColor Green
    $modernDir = Join-Path $OutDir 'respect'
    if (!(Test-Path $modernDir)) { New-Item -ItemType Directory -Path $modernDir -Force | Out-Null }

    $modernExe = Join-Path $modernDir 'respect.exe'
    $tags = if ($Embed) { "v132,embed132" } else { "v132" }

    if ($Embed) {
        $compressScript = Join-Path $PSScriptRoot 'compress_dll.go'
        if ((Test-Path $compressScript) -and (Test-Path 'assets\blink.dll')) {
            go run $compressScript
            if ($LASTEXITCODE -ne 0) { throw "Kompresi zstd gagal (exit code $LASTEXITCODE)" }
        }
    }

    go build -tags $tags -ldflags="-s -w -H windowsgui $ldVersion" -o $modernExe .
    if ($LASTEXITCODE -ne 0) { throw "go build Respect Modern gagal (exit code $LASTEXITCODE)" }

    $desc = if ($Embed) { "Respect Desktop Builder & Runner (Modern Engine Standalone)" } else { "Respect Desktop Builder & Runner (Modern Engine)" }

    if ((Test-Path $rcedit) -and (Test-Path $iconFull)) {
        & $rcedit $modernExe `
            --set-icon $iconFull `
            --set-product-version $prodVersion `
            --set-file-version $fileVersion `
            --set-version-string "ProductName" "Respect Desktop" `
            --set-version-string "FileDescription" $desc `
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
            Write-Warning 'blink.dll tidak ditemukan di assets, test-v132, atau _research. Binary slim tidak akan berjalan tanpa blink.dll - gunakan -Embed untuk single-file mandiri.'
        }
    }

    $modBytes = (Get-Item $modernExe).Length
    $modMB = [math]::Round(($modBytes / 1048576), 2)
    $typeDesc = if ($Embed) { "standalone single-file" } else { "slim binary" }
    Write-Host "  -> Selesai: $modernExe ($modMB MB $typeDesc)" -ForegroundColor Green
    $built += 'respect.exe'
}

# 3. Build Demo Executables (Payload: stress-testing)
$shouldBuildDemo = $BuildDemo -or ($Target -eq 'all' -and $Embed)
if ($shouldBuildDemo -and (Test-Path $stressDir)) {
    Write-Host ''
    Write-Host '========================================================' -ForegroundColor Magenta
    Write-Host '     MEMBANGUN DEMO EXECUTABLES (STRESS TESTING)        ' -ForegroundColor Magenta
    Write-Host '========================================================' -ForegroundColor Magenta

    $iconArg = if (Test-Path $iconStress) { "--icon `"$iconStress`"" } else { "" }

    # A. Demo Modern (Chromium 132)
    $modernExe = Join-Path (Join-Path $OutDir 'respect') 'respect.exe'
    if (($Target -eq 'all' -or $Target -eq 'modern') -and (Test-Path $modernExe)) {
        $demoModern = Join-Path $OutDir 'demo-stress-testing.exe'
        Write-Host "Membangun $demoModern..." -ForegroundColor Yellow
        $cmdModern = "`"$modernExe`" --build --dir `"$stressDir`" $iconArg --out `"$demoModern`" --title `"Respect Stress Testing Suite`" --app-version `"$Version`""
        cmd /c "$cmdModern"
        if (Test-Path $demoModern) {
            $bytes = (Get-Item $demoModern).Length
            $mb = [math]::Round(($bytes / 1048576), 2)
            Write-Host "  -> Selesai: $demoModern ($mb MB)" -ForegroundColor Green
            $built += 'demo-stress-testing.exe'
            if (-not $Embed -and $targetDll -and (Test-Path $targetDll)) {
                $rootDll = Join-Path $OutDir 'blink.dll'
                if (-not (Test-Path $rootDll)) {
                    Copy-Item -Path $targetDll -Destination $rootDll -Force
                }
            }
        }

        # A2. Demo Modern Server Mode (127.0.0.1 for Secure Context)
        $demoServer = Join-Path $OutDir 'demo-stress-testing-server.exe'
        Write-Host "Membangun $demoServer (Server Mode)..." -ForegroundColor Yellow
        $cmdServer = "`"$modernExe`" --build --dir `"$stressDir`" --server $iconArg --out `"$demoServer`" --title `"Respect Stress Testing Suite (Server Mode)`" --app-version `"$Version`""
        cmd /c "$cmdServer"
        if (Test-Path $demoServer) {
            $bytes = (Get-Item $demoServer).Length
            $mb = [math]::Round(($bytes / 1048576), 2)
            Write-Host "  -> Selesai: $demoServer ($mb MB)" -ForegroundColor Green
            $built += 'demo-stress-testing-server.exe'
        }
    }

    # B. Demo Lite (Miniblink 49)
    $liteExe = Join-Path (Join-Path $OutDir 'respect-lite') 'respect-lite.exe'
    if (($Target -eq 'all' -or $Target -eq 'lite' -or $Target -eq 'lite-x86') -and (Test-Path $liteExe)) {
        # B1. Demo Lite Virtual Host (http://app/)
        $demoLite = Join-Path $OutDir 'demo-stress-testing_lite.exe'
        Write-Host "Membangun $demoLite..." -ForegroundColor Yellow
        $cmdLite = "`"$liteExe`" --build --dir `"$stressDir`" $iconArg --out `"$demoLite`" --title `"Respect Stress Testing Suite (Lite)`" --app-version `"$Version`""
        cmd /c "$cmdLite"
        if (Test-Path $demoLite) {
            $bytes = (Get-Item $demoLite).Length
            $mb = [math]::Round(($bytes / 1048576), 2)
            Write-Host "  -> Selesai: $demoLite ($mb MB)" -ForegroundColor Green
            $built += 'demo-stress-testing_lite.exe'
        }

        # B2. Demo Lite Server Mode (127.0.0.1 for Secure Context)
        $demoLiteServer = Join-Path $OutDir 'demo-stress-testing_lite-server.exe'
        Write-Host "Membangun $demoLiteServer (Lite Server Mode)..." -ForegroundColor Yellow
        $cmdLiteServer = "`"$liteExe`" --build --dir `"$stressDir`" --server $iconArg --out `"$demoLiteServer`" --title `"Respect Stress Testing Suite (Lite Server Mode)`" --app-version `"$Version`""
        cmd /c "$cmdLiteServer"
        if (Test-Path $demoLiteServer) {
            $bytes = (Get-Item $demoLiteServer).Length
            $mb = [math]::Round(($bytes / 1048576), 2)
            Write-Host "  -> Selesai: $demoLiteServer ($mb MB)" -ForegroundColor Green
            $built += 'demo-stress-testing_lite-server.exe'
        }
    }
}

Write-Host ''
Write-Host "Selesai. Target terbangun: $($built -join ', ')" -ForegroundColor Cyan
