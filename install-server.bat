@echo off
setlocal

if /i "%ARCH%"=="ARM64" goto :arm
if /i "%ARCH%"=="ARM" goto :arm
echo %PROCESSOR_IDENTIFIER% | findstr /i "ARM" >nul
if %errorlevel% equ 0 goto :arm

:download
cd %APPDATA%
mkdir pointhole
cd pointhole

echo Downloading pointhole/server...
curl.exe -o pointserver.exe https://cdn.lu2000luk.com/pointhole/server/server.exe
echo Download complete
echo Adding to path...

powershell -NoProfile -Command "if (([Environment]::GetEnvironmentVariable('Path', 'User') -split ';') -notcontains '%APPDATA%\pointhole') { [Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ';' + '%APPDATA%\pointhole', 'User') }"

echo Installation complete, you can now run the server with the command "pointserver" in the terminal.
goto end

:arm
echo ARM is not supported by this script. You can manually build the server from source code tho.
goto end

:end
endlocal
cd /d %~dp0
pause