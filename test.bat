@echo off
REM Root wrapper: the local message test lives in source_code\.
setlocal
cd /d "%~dp0source_code"
call test.bat %*
exit /b %errorlevel%
