# GoDash Windows Release Guide

This folder contains quick-start instructions to run GoDash on Windows.

## Option A: Run with Docker (Recommended)
1. Install Docker Desktop for Windows and ensure it's running.
2. In a PowerShell terminal, run:

   .\run-godash-with-docker.bat

This will build and start GoDash with PostgreSQL and Redis. When healthy, your browser will open:
- Dashboard: http://localhost:8080/
- Health: http://localhost:8080/health
- WebSocket: ws://localhost:8080/ws

Stop all services:

   docker-compose down

## Option B: Run the compiled EXE with local DB
1. Ensure you have a PostgreSQL server reachable (default: localhost:5433).
   - You can start the DB via Docker Compose first time: `docker-compose up -d postgres`
2. Build GoDash (or use the pre-built binary):
   - Double-click `build.bat` (or run `./build.bat` in PowerShell)
   - Binaries are in `build/`:
     - `build/godash.exe` (server)
     - `build/godash-cli.exe` (CLI)
3. Start the server using the helper script (sets env vars):

   .\run-godash.bat

Access the dashboard at http://localhost:8080/

## CLI Tool
You can run the CLI without the web server:

   build\godash-cli.exe -help
   build\godash-cli.exe -continuous -processes

## Notes
- Configuration is picked from environment variables set in `run-godash*.bat`.
- Auto-migrations run on startup; the DB schema will be created/updated automatically.
- For production, set `SERVER_MODE=release` and secure credentials.
