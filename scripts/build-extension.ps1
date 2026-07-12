param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("Chrome", "Firefox")]
    [string]$Browser
)

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

switch ($Browser.ToLowerInvariant()) {
    "chrome" {
        $BrowserName = "Chrome"
        $PackageName = "@browser-server/extension"
        $ExtensionDir = Join-Path $ProjectRoot "extension"
    }
    "firefox" {
        $BrowserName = "Firefox"
        $PackageName = "@browser-server/extension-firefox"
        $ExtensionDir = Join-Path $ProjectRoot "extension-firefox"
    }
}

if (-not (Get-Command pnpm -ErrorAction SilentlyContinue)) {
    Write-Host "ERROR: 'pnpm' was not found. Please install pnpm 11 or newer." -ForegroundColor Red
    exit 1
}

Set-Location -LiteralPath $ProjectRoot

Write-Host "`n==> Building $BrowserName extension..." -ForegroundColor Cyan
Write-Host "Running: pnpm --filter $PackageName build" -ForegroundColor Gray
pnpm --filter $PackageName build
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: $BrowserName extension build failed." -ForegroundColor Red
    exit $LASTEXITCODE
}

$ManifestPath = Join-Path $ExtensionDir "manifest.json"
$DistDir = Join-Path $ExtensionDir "dist"

if (-not (Test-Path $ManifestPath)) {
    Write-Host "ERROR: $BrowserName extension manifest not found at: $ManifestPath" -ForegroundColor Red
    exit 1
}

if (-not (Test-Path $DistDir)) {
    Write-Host "ERROR: $BrowserName extension dist directory not found after build." -ForegroundColor Red
    exit 1
}

Write-Host "$BrowserName extension built successfully at: $DistDir" -ForegroundColor Green
Write-Host "Load unpacked extension from: $ExtensionDir" -ForegroundColor Yellow
