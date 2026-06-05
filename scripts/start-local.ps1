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
$PostgresPort = 55432
$RedisDir = Join-Path $Runtime "redis"
$RedisData = Join-Path $Runtime "redis-data"
$MinioDir = Join-Path $Runtime "minio"
$MinioData = Join-Path $Runtime "minio-data"
$MailpitDir = Join-Path $Runtime "mailpit"

function Stop-RuntimePostgresIfNeeded {
    param([string]$RuntimePostgresBin)

    if (Test-Port -HostName "127.0.0.1" -Port $PostgresPort) {
        return
    }

    $runtimeBin = (Resolve-Path -LiteralPath $RuntimePostgresBin -ErrorAction SilentlyContinue)
    if (-not $runtimeBin) {
        return
    }

    $processes = Get-Process -Name "postgres" -ErrorAction SilentlyContinue
    foreach ($process in $processes) {
        if ($process.Path -and $process.Path.StartsWith($runtimeBin.Path, [System.StringComparison]::OrdinalIgnoreCase)) {
            Write-Host "Stopping old bundled PostgreSQL process (pid $($process.Id)) before switching to port $PostgresPort..."
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        }
    }
}

function Ensure-PostgresDatabase {
    $psql = Join-Path $PostgresBin "psql.exe"
    $createdb = Join-Path $PostgresBin "createdb.exe"

    for ($i = 0; $i -lt 20; $i++) {
        if (Test-Port -HostName "127.0.0.1" -Port $PostgresPort) {
            break
        }
        Start-Sleep -Seconds 1
    }

    if (-not (Test-Port -HostName "127.0.0.1" -Port $PostgresPort)) {
        throw "PostgreSQL did not start on 127.0.0.1:$PostgresPort. Run download-local-runtime.bat first if runtime files are missing."
    }

    Write-Host "Ensuring PostgreSQL role/database for Gamero..."
    & $psql -h localhost -p $PostgresPort -U postgres -d postgres -v ON_ERROR_STOP=1 -c "DO `$`$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'gamero') THEN CREATE ROLE gamero WITH LOGIN PASSWORD 'gamero123'; ELSE ALTER ROLE gamero WITH LOGIN PASSWORD 'gamero123'; END IF; END `$`$;" | Out-Host
    $exists = (& $psql -h localhost -p $PostgresPort -U postgres -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = 'gamero'").Trim()
    if ($exists -ne "1") {
        & $createdb -h localhost -p $PostgresPort -U postgres -O gamero gamero | Out-Host
    }
}

Stop-RuntimePostgresIfNeeded -RuntimePostgresBin $PostgresBin

Start-IfPortClosed `
    -Name "PostgreSQL" `
    -HostName "127.0.0.1" `
    -Port $PostgresPort `
    -FilePath (Join-Path $PostgresBin "pg_ctl.exe") `
    -ArgumentList "start -D `"$PostgresData`" -o `"-p $PostgresPort`" -l `"$(Join-Path $Runtime "postgres.log")`"" `
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
Ensure-PostgresDatabase

$ServerExe = Join-Path $Backend "server-local.exe"
Stop-PortProcess -Name "Backend" -Port 8081
Write-Host "Migrating database..."
Push-Location $Backend
try {
    go run .\cmd\migrate -config config\config.yaml
}
finally {
    Pop-Location
}

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
