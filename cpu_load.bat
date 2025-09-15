@echo off
:loop
echo Generating CPU load...
for /L %%i in (1,1,1000000) do rem
timeout /t 1 /nobreak > nul
goto loop