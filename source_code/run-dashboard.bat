@echo off
REM Starts the built ChitChat Bot dashboard. The data\ folder (accounts,
REM settings, data\eva_chat TXT files) is created next to this file.
setlocal
cd /d "%~dp0"
set "EXE=%~dp0build\bin\chitchatui.exe"
set "ERRLOG=%~dp0build\bin\startup-error.log"
if not exist "%EXE%" (
    echo build\bin\chitchatui.exe not found. Run install.bat first.
    pause
    exit /b 1
)
if exist "%ERRLOG%" del "%ERRLOG%" >nul 2>&1
echo Starting the dashboard ...
start "" "%EXE%"
timeout /t 6 /nobreak >nul
tasklist /fi "imagename eq chitchatui.exe" 2>nul | find /i "chitchatui.exe" >nul
if errorlevel 1 (
    echo.
    echo   [X] The dashboard closed right after it started.
    if exist "%ERRLOG%" (
        echo   Reason:
        type "%ERRLOG%"
    ) else (
        echo   No startup-error.log was written - Windows blocked the exe
        echo   ^(antivirus / SmartScreen^), or WebView2 is missing.
    )
    echo.
    pause
    exit /b 1
)
echo Dashboard is running.
exit /b 0
