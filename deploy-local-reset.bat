@echo off
setlocal

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\deploy-local-reset.ps1"

if errorlevel 1 (
  echo.
  echo Gamero local deployment failed.
  pause
  exit /b 1
)

echo.
echo Gamero local deployment completed.
echo Frontend: http://127.0.0.1:5714
echo Backend:  http://127.0.0.1:8081/health
echo MinIO:    http://127.0.0.1:9001
echo Mailpit:  http://127.0.0.1:8025
echo.
pause
