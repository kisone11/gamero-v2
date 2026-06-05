$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Backend = Join-Path $Root "backend"
$Runtime = Join-Path $Backend ".runtime"
$Downloads = Join-Path $Runtime "downloads"

$PostgresDir = Join-Path $Runtime "postgres"
$PostgresPgsql = Join-Path $PostgresDir "pgsql"
$PostgresBin = Join-Path $PostgresPgsql "bin"
$PostgresData = Join-Path $Runtime "pgdata"
$RedisDir = Join-Path $Runtime "redis"
$RedisData = Join-Path $Runtime "redis-data"
$MinioDir = Join-Path $Runtime "minio"
$MinioData = Join-Path $Runtime "minio-data"
$MailpitDir = Join-Path $Runtime "mailpit"
$PostgresPort = 55432

function Ensure-Directory {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        New-Item -ItemType Directory -Path $Path | Out-Null
    }
}

function Download-File {
    param(
        [string[]]$Urls,
        [string]$Destination
    )

    if (Test-Path -LiteralPath $Destination) {
        return
    }

    foreach ($url in $Urls) {
        try {
            Write-Host "Downloading $url"
            Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $Destination
            return
        }
        catch {
            Write-Host "Download failed: $url" -ForegroundColor Yellow
            if (Test-Path -LiteralPath $Destination) {
                Remove-Item -LiteralPath $Destination -Force
            }
        }
    }

    throw "All download URLs failed for $Destination"
}

function Expand-ZipOnce {
    param(
        [string]$ZipPath,
        [string]$Destination,
        [string]$ExpectedFile
    )

    if (Test-Path -LiteralPath $ExpectedFile) {
        return
    }

    Ensure-Directory $Destination
    Expand-Archive -LiteralPath $ZipPath -DestinationPath $Destination -Force
}

function Install-Postgres {
    $archive = Join-Path $Downloads "postgresql-windows-x64-binaries.zip"
    Download-File `
        -Urls @(
            "https://get.enterprisedb.com/postgresql/postgresql-17.10-1-windows-x64-binaries.zip",
            "https://get.enterprisedb.com/postgresql/postgresql-17.6-1-windows-x64-binaries.zip",
            "https://get.enterprisedb.com/postgresql/postgresql-16.10-1-windows-x64-binaries.zip"
        ) `
        -Destination $archive

    Expand-ZipOnce -ZipPath $archive -Destination $PostgresDir -ExpectedFile (Join-Path $PostgresBin "postgres.exe")

    if (-not (Test-Path -LiteralPath (Join-Path $PostgresBin "postgres.exe"))) {
        throw "PostgreSQL was downloaded but postgres.exe was not found under $PostgresBin"
    }

    if (-not (Test-Path -LiteralPath $PostgresData)) {
        Write-Host "Initializing PostgreSQL data directory..."
        & (Join-Path $PostgresBin "initdb.exe") -D $PostgresData -U postgres -A trust --encoding=UTF8 --locale=C | Out-Host
    }

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ([string]::IsNullOrWhiteSpace($userPath)) {
        [Environment]::SetEnvironmentVariable("Path", $PostgresBin, "User")
    }
    elseif ($userPath.Split(';') -notcontains $PostgresBin) {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$PostgresBin", "User")
    }
}

function Ensure-PostgresDatabase {
    $pgCtl = Join-Path $PostgresBin "pg_ctl.exe"
    $psql = Join-Path $PostgresBin "psql.exe"
    $createdb = Join-Path $PostgresBin "createdb.exe"

    $startedHere = $false
    $portOpen = Test-NetConnection -ComputerName 127.0.0.1 -Port $PostgresPort -InformationLevel Quiet -WarningAction SilentlyContinue
    if (-not $portOpen) {
        Write-Host "Starting PostgreSQL temporarily to create gamero database..."
        & $pgCtl start -D $PostgresData -o "-p $PostgresPort" -l (Join-Path $Runtime "postgres-init.log") | Out-Host
        $startedHere = $true
        Start-Sleep -Seconds 3
    }

    try {
        & $psql -h localhost -p $PostgresPort -U postgres -d postgres -v ON_ERROR_STOP=1 -c "DO `$`$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'gamero') THEN CREATE ROLE gamero WITH LOGIN PASSWORD 'gamero123'; ELSE ALTER ROLE gamero WITH LOGIN PASSWORD 'gamero123'; END IF; END `$`$;" | Out-Host
        & $psql -h localhost -p $PostgresPort -U postgres -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = 'gamero'" | ForEach-Object {
            if ($_.Trim() -eq "1") { $script:GameroDbExists = $true }
        }
        if (-not $script:GameroDbExists) {
            & $createdb -h localhost -p $PostgresPort -U postgres -O gamero gamero | Out-Host
        }
    }
    finally {
        if ($startedHere) {
            & $pgCtl stop -D $PostgresData -m fast | Out-Host
        }
    }
}

function Install-Redis {
    $archive = Join-Path $Downloads "redis-windows-x64.zip"
    Download-File `
        -Urls @("https://github.com/tporadowski/redis/releases/download/v5.0.14.1/Redis-x64-5.0.14.1.zip") `
        -Destination $archive
    Expand-ZipOnce -ZipPath $archive -Destination $RedisDir -ExpectedFile (Join-Path $RedisDir "redis-server.exe")
    Ensure-Directory $RedisData
}

function Install-Minio {
    Ensure-Directory $MinioDir
    Ensure-Directory $MinioData
    $minioExe = Join-Path $MinioDir "minio.exe"
    Download-File `
        -Urls @("https://dl.min.io/server/minio/release/windows-amd64/minio.exe") `
        -Destination $minioExe
}

function Install-Mailpit {
    $archive = Join-Path $Downloads "mailpit-windows-amd64.zip"
    Download-File `
        -Urls @("https://github.com/axllent/mailpit/releases/latest/download/mailpit-windows-amd64.zip") `
        -Destination $archive
    Expand-ZipOnce -ZipPath $archive -Destination $MailpitDir -ExpectedFile (Join-Path $MailpitDir "mailpit.exe")
}

Ensure-Directory $Runtime
Ensure-Directory $Downloads

Write-Host "[1/5] Installing portable PostgreSQL..." -ForegroundColor Cyan
Install-Postgres
Ensure-PostgresDatabase

Write-Host "[2/5] Installing portable Redis..." -ForegroundColor Cyan
Install-Redis

Write-Host "[3/5] Installing portable MinIO..." -ForegroundColor Cyan
Install-Minio

Write-Host "[4/5] Installing portable Mailpit..." -ForegroundColor Cyan
Install-Mailpit

Write-Host "[5/5] Runtime dependency check..." -ForegroundColor Cyan
& (Join-Path $PostgresBin "psql.exe") --version | Out-Host
& (Join-Path $RedisDir "redis-server.exe") --version | Out-Host
& (Join-Path $MinioDir "minio.exe") --version | Select-Object -First 1 | Out-Host
& (Join-Path $MailpitDir "mailpit.exe") version | Out-Host

Write-Host ""
Write-Host "Local runtime dependencies are ready under:" -ForegroundColor Green
Write-Host $Runtime
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. Open a new PowerShell if you want to use psql directly from PATH."
Write-Host "  2. Run start-local.bat to start PostgreSQL, Redis, MinIO, Mailpit, backend, and frontend."
