@echo off
setlocal

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\download-local-runtime.ps1"
set EXIT_CODE=%ERRORLEVEL%

if not "%EXIT_CODE%"=="0" (
  echo.
  echo Failed to download local runtime dependencies.
  pause
  exit /b %EXIT_CODE%
)

echo.
echo Local runtime dependencies are ready.
pause
