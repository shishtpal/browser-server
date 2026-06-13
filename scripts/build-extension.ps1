[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$ExtensionDir = Join-Path $ProjectRoot "extension"

Set-Location -LiteralPath $ExtensionDir

Write-Host "`n==> Building browser extension..." -ForegroundColor Cyan
Write-Host "Working directory: $ExtensionDir" -ForegroundColor Gray

if (Get-Command pnpm -ErrorAction SilentlyContinue) {
    Write-Host "Running: pnpm build" -ForegroundColor Gray
    pnpm build
} elseif (Get-Command npm -ErrorAction SilentlyContinue) {
    Write-Host "Running: npm run build" -ForegroundColor Gray
    npm run build
} else {
    Write-Host "ERROR: Neither 'pnpm' nor 'npm' found. Please install one of them." -ForegroundColor Red
    exit 1
}

$ManifestPath = Join-Path $ExtensionDir "manifest.json"
$DistDir = Join-Path $ExtensionDir "dist"

if (-not (Test-Path $ManifestPath)) {
    Write-Host "ERROR: Extension manifest not found." -ForegroundColor Red
    exit 1
}

if (-not (Test-Path $DistDir)) {
    Write-Host "ERROR: Extension dist directory not found after build." -ForegroundColor Red
    exit 1
}

Write-Host "Extension built successfully at: $DistDir" -ForegroundColor Green
Write-Host "Load unpacked extension from: $ExtensionDir" -ForegroundColor Yellow
