@echo off
REM GoDash Windows Launcher (local Postgres on localhost:5433)

set SERVER_HOST=0.0.0.0
set SERVER_PORT=8080
set SERVER_MODE=release

REM Database (matches docker-compose exposed port)
set DB_HOST=localhost
set DB_PORT=5433
set DB_USER=godash
set DB_PASSWORD=secure_password_123
set DB_NAME=godash
set DB_SSL_MODE=disable
set DB_MAX_OPEN_CONNS=25
set DB_MAX_IDLE_CONNS=5

REM Optional kiosk mode (set APP_KIOSK=1 to enable)
if "%APP_KIOSK%"=="" set APP_KIOSK=0

if not exist build\godash.exe (
  echo [ERROR] build\godash.exe not found. Run build.bat first.
  exit /b 1
)

set APP_URL=http://localhost:%SERVER_PORT%/

echo.
echo Starting GoDash server (minimized)...
echo Serving at %APP_URL%
echo.

REM Start server in a new minimized window
start "GoDash Server" /min build\godash.exe

REM Wait for health endpoint to be ready (up to 60s)
echo Waiting for server to become healthy...
powershell -NoProfile -Command "\
  $ErrorActionPreference='SilentlyContinue'; \
  $url='%APP_URL%health'; \
  for($i=1;$i -le 60;$i++){ \
    try{ $r=Invoke-WebRequest -UseBasicParsing $url; if($r.StatusCode -eq 200){ exit 0 } }catch{}; \
    Start-Sleep -Seconds 1 \
  }; exit 1"
if not %ERRORLEVEL%==0 (
  echo [WARN] Server health check timed out. Will try to open UI anyway.
)

REM Find Edge or Chrome
set "BROWSER="
if exist "%ProgramFiles(x86)%\Microsoft\Edge\Application\msedge.exe" set "BROWSER=%ProgramFiles(x86)%\Microsoft\Edge\Application\msedge.exe"
if "%BROWSER%"=="" if exist "%ProgramFiles%\Microsoft\Edge\Application\msedge.exe" set "BROWSER=%ProgramFiles%\Microsoft\Edge\Application\msedge.exe"
if "%BROWSER%"=="" if exist "%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe" set "BROWSER=%ProgramFiles(x86)%\Google\Chrome\Application\chrome.exe"
if "%BROWSER%"=="" if exist "%ProgramFiles%\Google\Chrome\Application\chrome.exe" set "BROWSER=%ProgramFiles%\Google\Chrome\Application\chrome.exe"

echo.
echo Opening dashboard window...
if not "%BROWSER%"=="" (
  if "%APP_KIOSK%"=="1" (
    start "GoDash" "%BROWSER%" --new-window --kiosk %APP_URL%
  ) else (
    start "GoDash" "%BROWSER%" --new-window --app=%APP_URL%
  )
) else (
  echo [INFO] Edge/Chrome not found in default locations. Opening in default browser.
  start "" %APP_URL%
)

echo.
echo GoDash is running. Close the server window or press Ctrl+C there to stop.
