@echo off
REM Package GoDash Windows distribution folder with assets

set DIST_DIR=build\dist\godash-windows

echo [1/5] Building binaries...
set NO_PAUSE=1
call .\build.bat
if %ERRORLEVEL% neq 0 (
  echo [ERROR] Build failed.
  exit /b 1
)

echo [2/5] Preparing distribution directory: %DIST_DIR%
if exist %DIST_DIR% rmdir /s /q %DIST_DIR%
mkdir %DIST_DIR%
mkdir %DIST_DIR%\web\static\css
mkdir %DIST_DIR%\web\static\js
mkdir %DIST_DIR%\web\templates
mkdir %DIST_DIR%\configs

echo [3/5] Copying binaries and assets...
copy /y build\godash.exe %DIST_DIR%\ >nul
copy /y build\godash-cli.exe %DIST_DIR%\ >nul
xcopy /e /i /y web\static %DIST_DIR%\web\static\ >nul
xcopy /e /i /y web\templates %DIST_DIR%\web\templates\ >nul

REM Provide a minimal production config sample
copy /y configs\production.yaml %DIST_DIR%\configs\production.yaml >nul

echo [4/5] Creating run script...
(
  echo @echo off
  echo set SERVER_HOST=0.0.0.0
  echo set SERVER_PORT=8080
  echo set SERVER_MODE=release
  echo rem Configure your database below or use docker-compose in project root
  echo set DB_HOST=localhost
  echo set DB_PORT=5433
  echo set DB_USER=godash
  echo set DB_PASSWORD=secure_password_123
  echo set DB_NAME=godash
  echo set DB_SSL_MODE=disable
  echo echo Starting GoDash on http://%%SERVER_HOST%%:%%SERVER_PORT%% ...
  echo .\godash.exe
) > %DIST_DIR%\run.bat

echo [5/6] Creating ZIP archive...
powershell -NoProfile -Command "Compress-Archive -Path '%DIST_DIR%\*' -DestinationPath 'build\dist\godash-windows.zip' -Force" >nul 2>&1
if %ERRORLEVEL% neq 0 (
  echo [WARN] Failed to create ZIP archive. You can zip the folder manually.
)

echo [6/6] Done. Distribution is ready at %DIST_DIR%
echo Also created: build\dist\godash-windows.zip
echo Contents:
dir /b %DIST_DIR%
