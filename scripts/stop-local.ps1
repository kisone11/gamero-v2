$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Backend = Join-Path $Root "backend"
$Runtime = Join-Path $Backend ".runtime"
$ComposeFile = Join-Path $Root "docker-compose.local.yml"

function Stop-PortProcess {
    param(
        [string]$Name,
        [int]$Port
    )

    $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    foreach ($connection in $connections) {
        $process = Get-Process -Id $connection.OwningProcess -ErrorAction SilentlyContinue
        if ($process) {
            Write-Host "Stopping $Name on port ${Port} (pid $($process.Id))..."
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }
}

function Stop-ProcessByPath {
    param(
        [string]$Name,
        [string]$Path
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }
    $fullPath = [System.IO.Path]::GetFullPath($Path)
    Get-Process -ErrorAction SilentlyContinue | Where-Object { $_.Path -eq $fullPath } | ForEach-Object {
        Write-Host "Stopping $Name (pid $($_.Id))..."
        Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "Stopping Gamero app processes..."
Stop-PortProcess -Name "Frontend" -Port 5714
Stop-PortProcess -Name "Backend" -Port 8081

Write-Host "Stopping native local services..."
$PostgresCtl = Join-Path $Runtime "postgres\pgsql\bin\pg_ctl.exe"
$PostgresData = Join-Path $Runtime "pgdata"
if ((Test-Path -LiteralPath $PostgresCtl) -and (Test-Path -LiteralPath $PostgresData)) {
    & $PostgresCtl stop -D $PostgresData -m fast | Out-Host
}

Stop-ProcessByPath -Name "Redis" -Path (Join-Path $Runtime "redis\redis-server.exe")
Stop-ProcessByPath -Name "MinIO" -Path (Join-Path $Runtime "minio\minio.exe")
Stop-ProcessByPath -Name "Mailpit" -Path (Join-Path $Runtime "mailpit\mailpit.exe")

if ((Get-Command docker -ErrorAction SilentlyContinue) -and (Test-Path -LiteralPath $ComposeFile)) {
    Write-Host "Stopping Docker local services..."
    docker compose -f $ComposeFile down | Out-Host
}

Write-Host "Done."
