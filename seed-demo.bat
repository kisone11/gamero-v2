@echo off
setlocal

pushd "%~dp0backend"
go run .\cmd\seed-demo -config config\config.yaml
set EXIT_CODE=%ERRORLEVEL%
if "%EXIT_CODE%"=="0" (
  go run .\cmd\seed-milestones -config config\config.yaml
  set EXIT_CODE=%ERRORLEVEL%
)
if "%EXIT_CODE%"=="0" (
  go run .\cmd\seed-resources -config config\config.yaml
  set EXIT_CODE=%ERRORLEVEL%
)
if "%EXIT_CODE%"=="0" (
  go run .\cmd\seed-risks -config config\config.yaml
  set EXIT_CODE=%ERRORLEVEL%
)
popd

if not "%EXIT_CODE%"=="0" (
  echo.
  echo Failed to seed demo data.
  pause
  exit /b %EXIT_CODE%
)

echo.
echo Demo data has been seeded successfully.
pause
