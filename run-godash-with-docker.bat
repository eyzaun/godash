@echo off
REM GoDash Windows Launcher with Docker Compose (bundles DB)

if not exist docker-compose.yml (
  echo [ERROR] docker-compose.yml not found in current directory.
  exit /b 1
)

REM Ensure Docker CLI exists
where docker >nul 2>&1
if errorlevel 1 (
  echo [ERROR] Docker CLI not found. Please install Docker Desktop for Windows.
  exit /b 1
)

REM Check if Docker engine is running; if not, try to launch Docker Desktop
docker info >nul 2>&1
if errorlevel 1 (
  echo Docker engine is not running. Attempting to start Docker Desktop...
  set "DOCKER_DESKTOP=%ProgramFiles%\Docker\Docker\Docker Desktop.exe"
  if not exist "%DOCKER_DESKTOP%" set "DOCKER_DESKTOP=%ProgramFiles(x86)%\Docker\Docker\Docker Desktop.exe"
  if exist "%DOCKER_DESKTOP%" (
    start "" "%DOCKER_DESKTOP%"
  ) else (
    echo [WARN] Could not find Docker Desktop at default path. Please start Docker Desktop manually.
  )

  echo Waiting for Docker engine to become ready...
  setlocal enabledelayedexpansion
  for /l %%i in (1,1,60) do (
    timeout /t 2 >nul
    docker info >nul 2>&1
    if !ERRORLEVEL! EQU 0 (
      echo Docker engine is ready.
      goto :docker_ready
    )
    if %%i EQU 1 echo (this can take ~1-2 minutes the first time)
  )
  echo [WARN] Docker engine did not report ready in time. Will try to proceed.
  REM Double-check with a lightweight docker ps
  docker ps >nul 2>&1
  if errorlevel 1 (
    echo [ERROR] Docker CLI cannot reach the engine. Please open Docker Desktop and try again.
    exit /b 1
  ) else (
    echo Docker appears to be reachable. Continuing...
  )
)

:docker_ready
REM Choose compose command (docker compose preferred)
set "COMPOSE_CMD=docker compose"
docker compose version >nul 2>&1
if errorlevel 1 (
  where docker-compose >nul 2>&1
  if not errorlevel 1 set "COMPOSE_CMD=docker-compose"
)

echo Bringing up Docker services (Postgres, Redis, GoDash)...
%COMPOSE_CMD% up -d --build
if %ERRORLEVEL% neq 0 (
  echo [ERROR] Compose up failed.
  exit /b 1
)

echo.
echo Waiting for GoDash HTTP health at http://localhost:8080/health ...
REM Poll HTTP endpoint (up to ~120s)
setlocal enabledelayedexpansion
for /l %%i in (1,1,120) do (
  powershell -NoProfile -Command "try { $r=Invoke-WebRequest -UseBasicParsing http://localhost:8080/health; if($r.StatusCode -eq 200){ exit 0 } } catch { exit 1 }" >nul 2>&1
  if !ERRORLEVEL! EQU 0 (
    echo Service is healthy.
    goto :open
  )
  if %%i EQU 1 echo (first startup may take 1-2 minutes)
  timeout /t 1 >nul
)

:open
echo Opening http://localhost:8080/ ...
start "" http://localhost:8080/
echo Done.
