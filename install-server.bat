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
echo 
curl.exe -o pointserver.exe https://cdn.lu2000luk.com/pointhole/server/server.exe
echo Download complete
echo Adding to path...
setx PATH "%PATH%;%APPDATA%\pointhole"
echo Installation complete, you can now run the server with the command "pointserver" in the terminal.
goto end

:arm
echo ARM is not supported by this script. You can manually build the server from source code tho.
goto end

:end
endlocal
cd /d %~dp0