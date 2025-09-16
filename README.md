# GoDash

Real-time system monitoring dashboard for Windows/Linux/macOS. Single-EXE on Windows with embedded UI. Shows CPU, memory, disk and network metrics with live updates.

## Download (Windows)

- Direct: https://github.com/eyzaun/godash/releases/download/v0.9.0/godash-v0.9.0-windows-x64.zip
- All releases: https://github.com/eyzaun/godash/releases

After download: unzip and run `godash.exe`. The dashboard opens at http://127.0.0.1:8080/

## Quick start (Windows)

1) Download the ZIP above and extract.
2) Double-click `godash.exe`.
3) Open http://127.0.0.1:8080/ if the browser doesn’t open automatically.

Notes:
- Default DB is SQLite. Data file is created next to the executable.
- To open as app/kiosk window on Windows, set `APP_KIOSK=1` before launch.

## Quick start (Developers)

Requirements: Go 1.19+

- Run: `go run main.go`
- Build: `go build -o build/godash.exe .`

Key endpoints:
- Dashboard: http://127.0.0.1:8080/
- Health: http://127.0.0.1:8080/health

Configuration basics:
- Host/port and mode are read from config (defaults to 127.0.0.1:8080, release).
- `SERVER_AUTO_OPEN=0` disables auto-opening the browser.

## License

MIT. See [LICENSE](LICENSE).