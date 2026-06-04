@echo off
setlocal enabledelayedexpansion

set "ROOT=%~dp0"
set "FRONTEND=%ROOT%frontend"
set "BACKEND=%ROOT%backend"
set "HARMONY=%ROOT%harmony"

echo ============================================================
echo Gamero dependency installer and local configuration helper
echo ============================================================
echo.

where node >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Node.js was not found in PATH.
  echo Install Node.js LTS, then run this script again.
  pause
  exit /b 1
)

where npm >nul 2>nul
if errorlevel 1 (
  echo [ERROR] npm was not found in PATH.
  echo Reinstall Node.js LTS with npm enabled.
  pause
  exit /b 1
)

where go >nul 2>nul
if errorlevel 1 (
  echo [ERROR] Go was not found in PATH.
  echo Install Go 1.25 or newer, then run this script again.
  pause
  exit /b 1
)

echo [1/5] Installing web frontend dependencies...
if not exist "%FRONTEND%\package.json" (
  echo [ERROR] frontend\package.json not found.
  pause
  exit /b 1
)
pushd "%FRONTEND%"
call npm install
if errorlevel 1 (
  popd
  echo [ERROR] npm install failed.
  pause
  exit /b 1
)
popd

echo.
echo [2/5] Downloading backend Go modules...
if not exist "%BACKEND%\go.mod" (
  echo [ERROR] backend\go.mod not found.
  pause
  exit /b 1
)
pushd "%BACKEND%"
go mod download
if errorlevel 1 (
  popd
  echo [ERROR] go mod download failed.
  pause
  exit /b 1
)
go mod tidy
if errorlevel 1 (
  popd
  echo [ERROR] go mod tidy failed.
  pause
  exit /b 1
)
popd

echo.
echo [3/5] Checking Harmony project files...
if not exist "%HARMONY%\oh-package.json5" (
  echo [ERROR] harmony\oh-package.json5 not found.
  pause
  exit /b 1
)

where ohpm >nul 2>nul
if errorlevel 1 (
  echo [INFO] ohpm was not found in PATH.
  echo DevEco Studio can install Harmony oh-package dependencies when opening the harmony project.
) else (
  pushd "%HARMONY%"
  call ohpm install
  if errorlevel 1 (
    popd
    echo [WARN] ohpm install failed. You can still sync from DevEco Studio.
  ) else (
    popd
    echo [OK] Harmony oh-package dependencies installed.
  )
)

where hvigor >nul 2>nul
if errorlevel 1 (
  echo [INFO] hvigor was not found in PATH.
  echo Open the harmony folder in DevEco Studio 6.1.1.280 once to sync Harmony dependencies.
) else (
  pushd "%HARMONY%"
  call hvigor --sync
  if errorlevel 1 (
    echo [WARN] hvigor sync failed. You can still sync from DevEco Studio.
  )
  popd
)

echo.
echo [4/5] Running quick build checks...
pushd "%FRONTEND%"
call npm run build
if errorlevel 1 (
  popd
  echo [WARN] Frontend build failed. Check the log above.
) else (
  popd
  echo [OK] Frontend build passed.
)

pushd "%BACKEND%"
go build ./...
if errorlevel 1 (
  popd
  echo [WARN] Backend build failed. Check the log above.
) else (
  popd
  echo [OK] Backend build passed.
)

echo.
echo [5/5] Local configuration summary
echo Web frontend: http://127.0.0.1:5714
echo Backend API:  http://127.0.0.1:8081/api/v1
echo Harmony emulator API: http://10.0.2.2:8081/api/v1
echo Harmony project: %HARMONY%
echo.
echo If you run Harmony on a real phone, edit:
echo harmony\entry\src\main\ets\services\ApiConfig.ets
echo and replace 10.0.2.2 with your PC LAN IP.
echo.
echo Done. You can now run start-local.bat, then open harmony in DevEco Studio.
pause
