[CmdletBinding()]
param(
    [ValidateSet("Frontend", "Server", "All")]
    [string]$Target = "All"
)

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $ProjectRoot

$BinDir = Join-Path $ProjectRoot "bin"
if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir | Out-Null
}

function Build-Frontend {
    $FrontendDir = Join-Path $ProjectRoot "frontend"

    Write-Host "`n==> Building frontend..." -ForegroundColor Cyan
    Set-Location -LiteralPath $FrontendDir

    if (Get-Command bun -ErrorAction SilentlyContinue) {
        Write-Host "Running: bun run build" -ForegroundColor Gray
        bun run build
    } elseif (Get-Command npm -ErrorAction SilentlyContinue) {
        Write-Host "Running: npm run build" -ForegroundColor Gray
        npm run build
    } else {
        Write-Host "ERROR: Neither 'bun' nor 'npm' found. Please install one of them." -ForegroundColor Red
        exit 1
    }

    $FrontendDist = Join-Path $FrontendDir "dist"
    if (-not (Test-Path $FrontendDist)) {
        Write-Host "ERROR: Frontend dist directory not found after build." -ForegroundColor Red
        exit 1
    }
    Write-Host "Frontend built successfully at: $FrontendDist" -ForegroundColor Green

    Write-Host "`n==> Copying frontend dist into Go binary directory for static serving..." -ForegroundColor Cyan
    $TargetDir = Join-Path $BinDir "frontend\dist"
    if (Test-Path $TargetDir) {
        Remove-Item -LiteralPath $TargetDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $TargetDir -Force | Out-Null
    Copy-Item -Path "$FrontendDist\*" -Destination $TargetDir -Recurse -Force
    Write-Host "Frontend copied to: $TargetDir" -ForegroundColor Green

    Set-Location -LiteralPath $ProjectRoot
}

function Build-Server {
    Write-Host "`n==> Building Go binary..." -ForegroundColor Cyan
    Set-Location -LiteralPath $ProjectRoot

    $GoOutput = Join-Path $BinDir "server.exe"
    Write-Host "Running: go build -o `"$GoOutput`" ./cmd/server" -ForegroundColor Gray
    go build -o $GoOutput ./cmd/server

    if (-not (Test-Path $GoOutput)) {
        Write-Host "ERROR: Go binary not found after build." -ForegroundColor Red
        exit 1
    }
    Write-Host "Go binary built successfully: $GoOutput" -ForegroundColor Green
}

switch ($Target) {
    "Frontend" {
        Build-Frontend
    }
    "Server" {
        Build-Server
    }
    "All" {
        Build-Frontend
        Build-Server
    }
}

Write-Host "`n==> Build complete! (Target: $Target)" -ForegroundColor Green
if ($Target -ne "Frontend") {
    $GoOutput = Join-Path $BinDir "server.exe"
    Write-Host "Run: $GoOutput" -ForegroundColor Yellow
}
