@echo off
REM GoDash Windows Launcher with Docker Compose (bundles DB)

if not exist docker-compose.yml (
  echo [ERROR] docker-compose.yml not found in current directory.
  exit /b 1
)

echo Bringing up Docker services (Postgres, Redis, GoDash)...
docker-compose up -d --build
if %ERRORLEVEL% neq 0 (
  echo [ERROR] docker-compose up failed.
  exit /b 1
)

echo.
echo Waiting for GoDash to become healthy...
REM Simple wait loop (up to ~60s)
setlocal enabledelayedexpansion
for /l %%i in (1,1,30) do (
  timeout /t 2 >nul
  docker ps --filter "name=godash-monitor" --filter "health=healthy" --format "table {{.Names}}\t{{.Status}}" | find /i "healthy" >nul
  if !ERRORLEVEL! EQU 0 (
    echo Service is healthy.
    goto :open
  )
  echo Waiting (%%i/30)...
)

:open
echo Opening http://localhost:8080/ ...
start "" http://localhost:8080/
echo Done.
