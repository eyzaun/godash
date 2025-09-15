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

if not exist build\godash.exe (
  echo [ERROR] build\godash.exe not found. Run build.bat first.
  exit /b 1
)

echo.
echo Starting GoDash on http://%SERVER_HOST%:%SERVER_PORT% ...
echo (Press Ctrl+C to stop)
echo.

build\godash.exe
