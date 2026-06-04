@echo off
setlocal

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\stop-local.ps1"

if errorlevel 1 (
  echo.
  echo Failed to stop local Gamero services.
  pause
  exit /b 1
)

echo.
echo Local Gamero services have been stopped.
pause
