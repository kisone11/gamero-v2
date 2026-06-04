$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Backend = Join-Path $Root "backend"
$Frontend = Join-Path $Root "frontend"
$Config = Join-Path $Backend "config\config.yaml"
$ConfigExample = Join-Path $Backend "config\config.example.yaml"
$ComposeFile = Join-Path $Root "docker-compose.local.yml"
$Runtime = Join-Path $Backend ".runtime"

function Require-Command {
    param([string]$Name, [string]$InstallHint)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "$Name was not found. $InstallHint"
    }
}

function Wait-Postgres {
    for ($i = 0; $i -lt 40; $i++) {
        docker exec gamero-postgres pg_isready -U postgres *> $null
        if ($LASTEXITCODE -eq 0) { return }
        Start-Sleep -Seconds 2
    }
    throw "PostgreSQL did not become ready in time."
}

Require-Command "docker" "Install Docker Desktop, then run this script again."
Require-Command "node" "Install Node.js LTS, then run this script again."
Require-Command "npm" "Install Node.js LTS with npm, then run this script again."
Require-Command "go" "Install Go 1.25 or newer, then run this script again."

if (-not (Test-Path -LiteralPath $Config)) {
    if (-not (Test-Path -LiteralPath $ConfigExample)) {
        throw "Missing backend config template: $ConfigExample"
    }
    Copy-Item -LiteralPath $ConfigExample -Destination $Config
    Write-Host "Created backend config from config.example.yaml"
}

Write-Host "[1/8] Pulling and starting infrastructure services..."
docker compose -f $ComposeFile up -d postgres redis minio mailpit

Write-Host "[2/8] Waiting for PostgreSQL..."
Wait-Postgres

Write-Host "[3/8] Dropping and recreating database/user..."
docker exec gamero-postgres psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'gamero';"
docker exec gamero-postgres psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS gamero;"
docker exec gamero-postgres psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "DROP ROLE IF EXISTS gamero;"
docker exec gamero-postgres psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "CREATE ROLE gamero WITH LOGIN PASSWORD 'gamero123';"
docker exec gamero-postgres psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE gamero OWNER gamero;"

Write-Host "[4/8] Creating MinIO bucket..."
docker run --rm --network gamero-local_default minio/mc:latest sh -c "mc alias set local http://gamero-minio:9000 gamero gamero123 && mc mb -p local/gamero && mc anonymous set download local/gamero" | Out-Host

Write-Host "[5/8] Installing dependencies..."
Push-Location $Frontend
try {
    npm install
} finally {
    Pop-Location
}

Push-Location $Backend
try {
    go mod download
    go mod tidy
} finally {
    Pop-Location
}

Write-Host "[6/8] Migrating database and inserting seed data..."
Push-Location $Backend
try {
    go run .\cmd\migrate -config config\config.yaml
    go run .\cmd\seed-admin -config config\config.yaml -email admin@gamero.local -password Gamero123 -username admin -nickname Admin
    go run .\cmd\seed-demo -config config\config.yaml
} finally {
    Pop-Location
}

Write-Host "[7/8] Building backend and frontend..."
Push-Location $Backend
try {
    go build -o server-local.exe .\cmd\server
} finally {
    Pop-Location
}

Push-Location $Frontend
try {
    npm run build
} finally {
    Pop-Location
}

Write-Host "[8/8] Starting app processes..."
if (-not (Test-Path -LiteralPath $Runtime)) {
    New-Item -ItemType Directory -Path $Runtime | Out-Null
}

$BackendExe = Join-Path $Backend "server-local.exe"
Start-Process -FilePath $BackendExe -ArgumentList "-config config\config.yaml" -WorkingDirectory $Backend -RedirectStandardOutput (Join-Path $Runtime "backend.out.log") -RedirectStandardError (Join-Path $Runtime "backend.err.log") -WindowStyle Minimized
Start-Process -FilePath "npm.cmd" -ArgumentList "run dev -- --host 127.0.0.1 --port 5714" -WorkingDirectory $Frontend -RedirectStandardOutput (Join-Path $Frontend ".vite.out.log") -RedirectStandardError (Join-Path $Frontend ".vite.err.log") -WindowStyle Minimized

Write-Host ""
Write-Host "Deployment complete."
Write-Host "Frontend: http://127.0.0.1:5714"
Write-Host "Backend:  http://127.0.0.1:8081/health"
Write-Host "MinIO:    http://127.0.0.1:9001  user: gamero  password: gamero123"
Write-Host "Mailpit:  http://127.0.0.1:8025"
Write-Host "Admin:    admin@gamero.local / Gamero123"
