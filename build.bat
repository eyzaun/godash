@echo off
REM GoDash Build Script for Windows

echo Building GoDash System Monitor...

REM Clean previous builds
if exist build rmdir /s /q build
mkdir build

REM Download dependencies
echo Downloading dependencies...
go mod download
go mod tidy

REM Build main application
echo Building main application...
go build -o build\godash.exe .
if %ERRORLEVEL% neq 0 (
    echo Error: Failed to build main application
    exit /b 1
)

REM Build CLI application
echo Building CLI application...
go build -o build\godash-cli.exe .\cmd\cli
if %ERRORLEVEL% neq 0 (
    echo Error: Failed to build CLI application
    exit /b 1
)

REM Run tests
echo Running tests...
go test .\...
if %ERRORLEVEL% neq 0 (
    echo Warning: Some tests failed
)

echo.
echo Build completed successfully!
echo Binaries are in the build\ directory:
echo   - build\godash.exe      (Main application)
echo   - build\godash-cli.exe  (CLI application)
echo.
echo To run the application:
echo   build\godash.exe
echo   build\godash-cli.exe -help

pause