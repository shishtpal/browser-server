#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Create, start, and stop a Windows service for a given executable command.

.DESCRIPTION
    Manages a Windows service wrapping: server.exe --port=9191
    Uses NSSM (Non-Sucking Service Manager) if available, otherwise falls back
    to sc.exe with a wrapper approach via PowerShell's built-in service cmdlets.

.PARAMETER Action
    The action to perform: Create | Start | Stop | Restart | Remove | Status

.PARAMETER ServiceName
    Name of the Windows service (default: "mysBrowserServer")

.PARAMETER ExePath
    Full path to server.exe (default: resolves from current directory)

.EXAMPLE
    .\Manage-Service.ps1 -Action Create
    .\Manage-Service.ps1 -Action Start
    .\Manage-Service.ps1 -Action Stop
    .\Manage-Service.ps1 -Action Restart
    .\Manage-Service.ps1 -Action Remove
    .\Manage-Service.ps1 -Action Status
#>

param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("Create", "Start", "Stop", "Restart", "Remove", "Status")]
    [string]$Action,

    [string]$ServiceName = "mysBrowserServer",

    [string]$ExePath = "D:\Codings\lang-Go\browser-server\bin\server.exe",

    [string]$ServiceArgs = "--port=9191",

    [string]$DisplayName = "My Browser Server Service",

    [string]$Description = "Runs server.exe --port=9191 as a Windows service"
)

# ── Helpers ────────────────────────────────────────────────────────────────────

function Write-Step([string]$msg) {
    Write-Host "`n>> $msg" -ForegroundColor Cyan
}

function Write-OK([string]$msg) {
    Write-Host "   [OK] $msg" -ForegroundColor Green
}

function Write-Fail([string]$msg) {
    Write-Host "   [FAIL] $msg" -ForegroundColor Red
}

function Get-ServiceStatus {
    return Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
}

# ── NSSM detection (preferred wrapper for arbitrary CLI args) ──────────────────

function Get-NssmPath {
    $nssm = Get-Command nssm -ErrorAction SilentlyContinue
    if ($nssm) { return $nssm.Source }

    # Common install locations
    foreach ($p in @("C:\nssm\nssm.exe", "C:\tools\nssm\nssm.exe", "$env:ProgramFiles\nssm\nssm.exe")) {
        if (Test-Path $p) { return $p }
    }
    return $null
}

# ── Actions ────────────────────────────────────────────────────────────────────

function Invoke-Create {
    Write-Step "Creating service '$ServiceName'"

    if (Get-ServiceStatus) {
        Write-Fail "Service '$ServiceName' already exists. Run -Action Remove first to recreate."
        return
    }

    if (-not (Test-Path $ExePath)) {
        Write-Fail "Executable not found: $ExePath"
        Write-Host "   Set -ExePath to the full path of server.exe" -ForegroundColor Yellow
        exit 1
    }

    $nssm = Get-NssmPath

    if ($nssm) {
        # ── NSSM path (handles arguments natively) ──────────────────────────
        Write-Host "   Using NSSM: $nssm"

        # Derive working dir and log dir from the exe location (not the script
        # location) so LocalSystem can write logs without permission issues.
        $appDir = Split-Path $ExePath -Parent
        $logDir = Join-Path $appDir "logs"
        New-Item -ItemType Directory -Force $logDir | Out-Null
        Write-Host "   Log directory: $logDir"

        & $nssm install $ServiceName $ExePath $ServiceArgs | Out-Null
        & $nssm set $ServiceName DisplayName    $DisplayName  | Out-Null
        & $nssm set $ServiceName Description    $Description  | Out-Null
        & $nssm set $ServiceName Start          SERVICE_AUTO_START | Out-Null
        & $nssm set $ServiceName AppDirectory   $appDir       | Out-Null
        & $nssm set $ServiceName AppStdout      "$logDir\stdout.log" | Out-Null
        & $nssm set $ServiceName AppStderr      "$logDir\stderr.log" | Out-Null
        & $nssm set $ServiceName AppRotateFiles 1 | Out-Null

    } else {
        # ── Fallback: sc.exe via New-Service ────────────────────────────────
        # sc.exe / New-Service require a quoted binary path; embed args directly.
        Write-Host "   NSSM not found — using New-Service (sc.exe). Args embedded in BinaryPathName."
        Write-Host "   Tip: install NSSM for richer service management." -ForegroundColor Yellow

        $binaryPath = "`"$ExePath`" $ServiceArgs"

        New-Service `
            -Name        $ServiceName `
            -BinaryPathName $binaryPath `
            -DisplayName $DisplayName `
            -Description $Description `
            -StartupType Automatic `
            | Out-Null
    }

    if (Get-ServiceStatus) {
        Write-OK "Service '$ServiceName' created successfully."
        Write-Host ""
        Invoke-Status
    } else {
        Write-Fail "Service creation failed. Check permissions or NSSM output above."
        exit 1
    }
}

function Invoke-Start {
    Write-Step "Starting service '$ServiceName'"

    $svc = Get-ServiceStatus
    if (-not $svc) {
        Write-Fail "Service '$ServiceName' does not exist. Run -Action Create first."
        exit 1
    }

    if ($svc.Status -eq "Running") {
        Write-OK "Service is already running."
        return
    }

    Start-Service -Name $ServiceName -ErrorAction Stop
    Start-Sleep -Seconds 2

    $svc = Get-ServiceStatus
    if ($svc.Status -eq "Running") {
        Write-OK "Service started. Status: $($svc.Status)"
    } else {
        Write-Fail "Service did not reach Running state. Status: $($svc.Status)"
        exit 1
    }
}

function Invoke-Stop {
    Write-Step "Stopping service '$ServiceName'"

    $svc = Get-ServiceStatus
    if (-not $svc) {
        Write-Fail "Service '$ServiceName' does not exist."
        exit 1
    }

    if ($svc.Status -eq "Stopped") {
        Write-OK "Service is already stopped."
        return
    }

    Stop-Service -Name $ServiceName -Force -ErrorAction Stop
    Start-Sleep -Seconds 2

    $svc = Get-ServiceStatus
    if ($svc.Status -eq "Stopped") {
        Write-OK "Service stopped. Status: $($svc.Status)"
    } else {
        Write-Fail "Service did not reach Stopped state. Status: $($svc.Status)"
        exit 1
    }
}

function Invoke-Restart {
    Write-Step "Restarting service '$ServiceName'"

    $svc = Get-ServiceStatus
    if (-not $svc) {
        Write-Fail "Service '$ServiceName' does not exist. Run -Action Create first."
        exit 1
    }

    if ($svc.Status -eq "Running") {
        Write-Host "   Stopping service..."
        Stop-Service -Name $ServiceName -Force -ErrorAction Stop
        Start-Sleep -Seconds 2

        $svc = Get-ServiceStatus
        if ($svc.Status -ne "Stopped") {
            Write-Fail "Service did not reach Stopped state. Status: $($svc.Status)"
            exit 1
        }
        Write-OK "Service stopped."
    }

    Write-Host "   Starting service..."
    Start-Service -Name $ServiceName -ErrorAction Stop
    Start-Sleep -Seconds 2

    $svc = Get-ServiceStatus
    if ($svc.Status -eq "Running") {
        Write-OK "Service restarted. Status: $($svc.Status)"
    } else {
        Write-Fail "Service did not reach Running state. Status: $($svc.Status)"
        exit 1
    }
}

function Invoke-Remove {
    Write-Step "Removing service '$ServiceName'"

    $svc = Get-ServiceStatus
    if (-not $svc) {
        Write-Fail "Service '$ServiceName' does not exist."
        return
    }

    # Stop first if running
    if ($svc.Status -ne "Stopped") {
        Write-Host "   Stopping service before removal..."
        Stop-Service -Name $ServiceName -Force -ErrorAction Stop
        Start-Sleep -Seconds 2
    }

    $nssm = Get-NssmPath
    if ($nssm) {
        & $nssm remove $ServiceName confirm | Out-Null
    } else {
        sc.exe delete $ServiceName | Out-Null
    }

    Start-Sleep -Seconds 1

    if (-not (Get-ServiceStatus)) {
        Write-OK "Service '$ServiceName' removed successfully."
    } else {
        Write-Fail "Failed to remove service. Try running sc.exe delete $ServiceName manually."
        exit 1
    }
}

function Invoke-Status {
    Write-Step "Status of service '$ServiceName'"

    $svc = Get-ServiceStatus
    if (-not $svc) {
        Write-Host "   Service '$ServiceName' does not exist." -ForegroundColor Yellow
        return
    }

    $color = switch ($svc.Status) {
        "Running" { "Green" }
        "Stopped" { "Red"   }
        default   { "Yellow"}
    }

    Write-Host "   Name        : $($svc.Name)"
    Write-Host "   Display Name: $($svc.DisplayName)"
    Write-Host "   Status      : " -NoNewline
    Write-Host $svc.Status -ForegroundColor $color
    Write-Host "   StartType   : $($svc.StartType)"
}

# ── Entry point ────────────────────────────────────────────────────────────────

switch ($Action) {
    "Create"  { Invoke-Create  }
    "Start"   { Invoke-Start   }
    "Stop"    { Invoke-Stop    }
    "Restart" { Invoke-Restart }
    "Remove"  { Invoke-Remove  }
    "Status"  { Invoke-Status  }
}