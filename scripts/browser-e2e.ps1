# Browser automation smoke test (API level).
#
# Exercises the extension command channel end to end WITHOUT a real browser:
#   server starts -> extension registers -> tab snapshot -> AI enqueues a
#   command -> extension-shaped processing ack + result completes it.
#
# Usage:
#   powershell -File scripts/browser-e2e.ps1
#
# Requires a built server binary (bin/server.exe). Build with scripts/build.ps1
# or `go build -o bin/server.exe ./cmd/server`.
$ErrorActionPreference = 'Stop'

$Root = Split-Path -Parent $PSScriptRoot
$ServerBin = Join-Path $Root 'bin\server.exe'
if (-not (Test-Path $ServerBin)) {
    Write-Host "server.exe not found at $ServerBin - building..." -ForegroundColor Yellow
    Push-Location $Root
    go build -o bin/server.exe ./cmd/server
    Pop-Location
}

$DataDir = Join-Path $env:TEMP ("bs-browser-e2e-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $DataDir | Out-Null
$Token = (1..64 | ForEach-Object { '{0:x}' -f (Get-Random -Max 16) }) -join ''
$TokenPath = Join-Path $DataDir '.bs-token'
Set-Content -Path $TokenPath -Value $Token
$env:SERVER_TOKEN_PATH = $TokenPath

$Port = 19391
$Base = "http://localhost:$Port"
$env:DATA_PATH = $DataDir

Write-Host "Starting server on :$Port (data: $DataDir)" -ForegroundColor Cyan
$proc = Start-Process -FilePath $ServerBin -ArgumentList "--port", "$Port" -PassThru -WindowStyle Hidden -RedirectStandardOutput (Join-Path $DataDir 'stdout.log') -RedirectStandardError (Join-Path $DataDir 'stderr.log')
try {
    $deadline = (Get-Date).AddSeconds(30)
    do {
        Start-Sleep -Milliseconds 500
        try {
            Invoke-RestMethod -Uri "$Base/health" -TimeoutSec 2 | Out-Null
            $ready = $true
        } catch {
            $ready = $false
        }
    } while (-not $ready -and (Get-Date) -lt $deadline)
    if (-not $ready) { throw 'server did not become ready' }

    $Headers = @{ Authorization = "Bearer $Token" }

    # 1. Extension registers.
    $inst = Invoke-RestMethod -Method Post -Uri "$Base/api/browser/register" -Headers $Headers -ContentType 'application/json' -Body (@{ instance_id = 'e2e-inst'; user_id = 1; browser = 'chrome'; label = 'E2E Chrome' } | ConvertTo-Json)
    if (-not $inst.online) { throw 'instance not online after register' }
    Write-Host "OK  register instance ($($inst.instance_id))" -ForegroundColor Green

    # 2. Tab snapshot.
    $tabsBody = @{ instance_id = 'e2e-inst'; tabs = @(
        @{ tab_uuid = 'e2e-tab-1'; instance_id = 'e2e-inst'; tab_id = 1; window_id = 1; url = 'https://example.com/'; title = 'Example'; active = $true }
    ) } | ConvertTo-Json -Depth 5
    Invoke-RestMethod -Method Post -Uri "$Base/api/browser/tabs" -Headers $Headers -ContentType 'application/json' -Body $tabsBody | Out-Null
    Write-Host "OK  tab snapshot synced" -ForegroundColor Green

    # 3. AI enqueues a command (same shape as a browser_scrape tool call).
    $cmdBody = @{
        target = @{ tab = @{ uuid = 'e2e-tab-1' }; session_id = 'e2e-task' }
        action = 'scrape'
        params = @{ extract = @('text', 'links'); max_links = 10 }
        timeout_ms = 5000
    } | ConvertTo-Json -Depth 6
    $cmd = Invoke-RestMethod -Method Post -Uri "$Base/api/browser/cmd" -Headers $Headers -ContentType 'application/json' -Body $cmdBody
    if ($cmd.status -ne 'queued') { throw "expected queued, got $($cmd.status)" }
    Write-Host "OK  command enqueued ($($cmd.command_id))" -ForegroundColor Green

    # 4. Queue fallback path (what the extension alarm polls).
    $queued = Invoke-RestMethod -Uri "$Base/api/browser/queue?instance_id=e2e-inst" -Headers $Headers
    if (@($queued).Count -ne 1) { throw "expected 1 queued command, got $(@($queued).Count)" }
    Write-Host "OK  queue fallback sees the command" -ForegroundColor Green

    # 5. Extension acknowledges processing, then completes it.
    $ackBody = @{ command_id = $cmd.command_id; status = 'processing' } | ConvertTo-Json
    Invoke-RestMethod -Method Post -Uri "$Base/api/browser/result" -Headers $Headers -ContentType 'application/json' -Body $ackBody | Out-Null
    $acked = Invoke-RestMethod -Uri "$Base/api/browser/commands/$($cmd.command_id)" -Headers $Headers
    if ($acked.status -ne 'sent') { throw "expected sent after processing ack, got $($acked.status)" }
    Write-Host "OK  processing ack marks command sent" -ForegroundColor Green

    $resultBody = @{ command_id = $cmd.command_id; status = 'succeeded'; result = @{
        page = @{ url = 'https://example.com/'; title = 'Example' }
        scrape = @{ text = @('Example Domain'); links = @() }
    } } | ConvertTo-Json -Depth 8
    Invoke-RestMethod -Method Post -Uri "$Base/api/browser/result" -Headers $Headers -ContentType 'application/json' -Body $resultBody | Out-Null

    # 6. CLI-style poll sees the final state.
    $final = Invoke-RestMethod -Uri "$Base/api/browser/commands/$($cmd.command_id)" -Headers $Headers
    if ($final.status -ne 'succeeded') { throw "expected succeeded, got $($final.status)" }
    Write-Host "OK  command completed and pollable ($($final.status))" -ForegroundColor Green

    # 7. Ambiguity policy: a second identical tab makes url-glob ambiguous.
    $tabsBody2 = @{ instance_id = 'e2e-inst'; tabs = @(
        @{ tab_uuid = 'e2e-tab-1'; instance_id = 'e2e-inst'; tab_id = 1; window_id = 1; url = 'https://example.com/'; title = 'Example'; active = $true },
        @{ tab_uuid = 'e2e-tab-2'; instance_id = 'e2e-inst'; tab_id = 2; window_id = 1; url = 'https://example.com/'; title = 'Example 2'; active = $false }
    ) } | ConvertTo-Json -Depth 5
    Invoke-RestMethod -Method Post -Uri "$Base/api/browser/tabs" -Headers $Headers -ContentType 'application/json' -Body $tabsBody2 | Out-Null
    try {
        Invoke-RestMethod -Method Post -Uri "$Base/api/browser/cmd" -Headers $Headers -ContentType 'application/json' -Body (@{ target = @{ tab = @{ url = 'https://example.com/*' } }; action = 'scrape' } | ConvertTo-Json -Depth 5) | Out-Null
        throw 'expected ambiguity error'
    } catch {
        $resp = $_.ErrorDetails.Message | ConvertFrom-Json
        if ($resp.code -ne 'tab_ambiguous') { throw "expected tab_ambiguous, got $($resp.code)" }
        Write-Host "OK  ambiguity policy returns candidates (tab_ambiguous)" -ForegroundColor Green
    }

    # 8. Session-owner lock: a different session targeting the busy tab is refused.
    $second = @{
        target = @{ tab = @{ uuid = 'e2e-tab-1' }; session_id = 'other-session' }
        action = 'scrape'
        params = @{ extract = @('text') }
        timeout_ms = 1000
    } | ConvertTo-Json -Depth 6
    try {
        Invoke-RestMethod -Method Post -Uri "$Base/api/browser/cmd" -Headers $Headers -ContentType 'application/json' -Body $second | Out-Null
        throw 'expected tab_busy for second session'
    } catch {
        if ($_ -like '*tab_busy*') {
            Write-Host "OK  session-owner lock refuses second session (tab_busy)" -ForegroundColor Green
        } else {
            throw
        }
    }

    # 9. Release the owner with a late result and verify the lock clears.
    Invoke-RestMethod -Method Post -Uri "$Base/api/browser/result" -Headers $Headers -ContentType 'application/json' -Body (`
        @{ command_id = $cmd.command_id; status = 'succeeded'; result = @{ page = @{ url = 'https://example.com/'; title = 'Example' } } } | ConvertTo-Json -Depth 4) | Out-Null
    $third = Invoke-RestMethod -Method Post -Uri "$Base/api/browser/cmd" -Headers $Headers -ContentType 'application/json' -Body (`
        @{ target = @{ tab = @{ uuid = 'e2e-tab-1' }; session_id = 'other-session' }; action = 'scrape'; params = @{ extract = @('text') }; timeout_ms = 1000 } | ConvertTo-Json -Depth 6)
    if ($third.status -ne 'queued') { throw "expected lock released, got $($third.status)" }
    Write-Host "OK  lock releases after result" -ForegroundColor Green

    Write-Host ""
    Write-Host "Browser automation smoke test PASSED" -ForegroundColor Green
} finally {
    if ($proc -and -not $proc.HasExited) {
        Stop-Process -Id $proc.Id -Force
    }
    Remove-Item Env:DATA_PATH -ErrorAction SilentlyContinue
    Remove-Item Env:SERVER_TOKEN_PATH -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force $DataDir -ErrorAction SilentlyContinue
}
