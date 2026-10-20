@echo off
REM Root wrapper: starts the built desktop app from source_code\.
setlocal
cd /d "%~dp0source_code"
call run-dashboard.bat
exit /b %errorlevel%
