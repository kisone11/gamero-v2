$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Backend = Join-Path $Root "backend"
$Frontend = Join-Path $Root "frontend"
$Runtime = Join-Path $Backend ".runtime"

function Test-Port {
    param(
        [string]$HostName,
        [int]$Port
    )

    $client = New-Object System.Net.Sockets.TcpClient
    try {
        $result = $client.BeginConnect($HostName, $Port, $null, $null)
        if (-not $result.AsyncWaitHandle.WaitOne(500, $false)) {
            return $false
        }
        $client.EndConnect($result)
        return $true
    }
    catch {
        return $false
    }
    finally {
        $client.Close()
    }
}

function Start-IfPortClosed {
    param(
        [string]$Name,
        [string]$HostName,
        [int]$Port,
        [string]$FilePath,
        [string]$ArgumentList,
        [string]$WorkingDirectory,
        [string]$OutLog,
        [string]$ErrLog
    )

    if (Test-Port -HostName $HostName -Port $Port) {
        Write-Host "$Name already running on ${HostName}:$Port"
        return
    }

    Write-Host "Starting $Name..."
    if ($OutLog -and $ErrLog) {
        Start-Process -FilePath $FilePath -ArgumentList $ArgumentList -WorkingDirectory $WorkingDirectory -RedirectStandardOutput $OutLog -RedirectStandardError $ErrLog -WindowStyle Minimized
    }
    else {
        Start-Process -FilePath $FilePath -ArgumentList $ArgumentList -WorkingDirectory $WorkingDirectory -WindowStyle Minimized
    }
}

function Stop-PortProcess {
    param(
        [string]$Name,
        [int]$Port
    )

    $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    foreach ($connection in $connections) {
        $process = Get-Process -Id $connection.OwningProcess -ErrorAction SilentlyContinue
        if ($process) {
            Write-Host "Stopping existing $Name on port ${Port} (pid $($process.Id))..."
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }
}

if (-not (Test-Path -LiteralPath $Runtime)) {
    New-Item -ItemType Directory -Path $Runtime | Out-Null
}

$PostgresBin = Join-Path $Runtime "postgres\pgsql\bin"
$PostgresData = Join-Path $Runtime "pgdata"
$RedisDir = Join-Path $Runtime "redis"
$RedisData = Join-Path $Runtime "redis-data"
$MinioDir = Join-Path $Runtime "minio"
$MinioData = Join-Path $Runtime "minio-data"
$MailpitDir = Join-Path $Runtime "mailpit"

Start-IfPortClosed `
    -Name "PostgreSQL" `
    -HostName "127.0.0.1" `
    -Port 5432 `
    -FilePath (Join-Path $PostgresBin "pg_ctl.exe") `
    -ArgumentList "start -D `"$PostgresData`" -l `"$(Join-Path $Runtime "postgres.log")`"" `
    -WorkingDirectory $PostgresBin `
    -OutLog $null `
    -ErrLog $null

Start-IfPortClosed `
    -Name "Redis" `
    -HostName "127.0.0.1" `
    -Port 6379 `
    -FilePath (Join-Path $RedisDir "redis-server.exe") `
    -ArgumentList "--dir `"$RedisData`" --port 6379" `
    -WorkingDirectory $RedisDir `
    -OutLog $null `
    -ErrLog $null

$env:MINIO_ROOT_USER = "gamero"
$env:MINIO_ROOT_PASSWORD = "gamero123"
$env:MINIO_API_CORS_ALLOW_ORIGIN = "http://127.0.0.1:5714,http://localhost:5714"
Start-IfPortClosed `
    -Name "MinIO" `
    -HostName "127.0.0.1" `
    -Port 9000 `
    -FilePath (Join-Path $MinioDir "minio.exe") `
    -ArgumentList "server `"$MinioData`" --address 127.0.0.1:9000 --console-address 127.0.0.1:9001" `
    -WorkingDirectory $MinioDir `
    -OutLog (Join-Path $MinioDir "minio.out.log") `
    -ErrLog (Join-Path $MinioDir "minio.err.log")

Start-IfPortClosed `
    -Name "Mailpit" `
    -HostName "127.0.0.1" `
    -Port 1025 `
    -FilePath (Join-Path $MailpitDir "mailpit.exe") `
    -ArgumentList "--smtp 127.0.0.1:1025 --listen 127.0.0.1:8025 --database `"$(Join-Path $MailpitDir "mailpit.db")`" --log-file `"$(Join-Path $MailpitDir "mailpit.log")`"" `
    -WorkingDirectory $MailpitDir `
    -OutLog $null `
    -ErrLog $null

Start-Sleep -Seconds 5

$ServerExe = Join-Path $Backend "server-local.exe"
Stop-PortProcess -Name "Backend" -Port 8081
Write-Host "Building backend..."
Push-Location $Backend
try {
    go build -o server-local.exe .\cmd\server
}
finally {
    Pop-Location
}

Start-IfPortClosed `
    -Name "Backend" `
    -HostName "127.0.0.1" `
    -Port 8081 `
    -FilePath $ServerExe `
    -ArgumentList "-config config\config.yaml" `
    -WorkingDirectory $Backend `
    -OutLog (Join-Path $Runtime "backend.out.log") `
    -ErrLog (Join-Path $Runtime "backend.err.log")

Stop-PortProcess -Name "Frontend" -Port 5714
Start-IfPortClosed `
    -Name "Frontend" `
    -HostName "127.0.0.1" `
    -Port 5714 `
    -FilePath "npm.cmd" `
    -ArgumentList "run dev -- --host 127.0.0.1 --port 5714" `
    -WorkingDirectory $Frontend `
    -OutLog (Join-Path $Frontend ".vite.out.log") `
    -ErrLog (Join-Path $Frontend ".vite.err.log")

Start-Sleep -Seconds 3

Write-Host ""
Write-Host "Local Gamero is starting/running:"
Write-Host "Frontend: http://127.0.0.1:5714"
Write-Host "Backend:  http://127.0.0.1:8081/health"
Write-Host "MinIO:    http://127.0.0.1:9001  user: gamero  password: gamero123"
Write-Host "Mailpit:  http://127.0.0.1:8025"
