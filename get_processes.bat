@echo off
cd o:\Web_Projects\godash\godash
start /b go run main.go
timeout /t 5 /nobreak > nul
curl -s "http://localhost:8080/api/v1/metrics/processes" > processes_response_final.json
type processes_response_final.json