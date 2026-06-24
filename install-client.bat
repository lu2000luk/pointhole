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

echo Downloading pointhole/client...
echo 
curl.exe -O https://cdn.lu2000luk.com/pointhole/client/client.exe
echo Download complete
echo Creating shortcut...
powershell -NoProfile -Command "$s=(New-Object -COM WScript.Shell).CreateShortcut('%APPDATA%\Microsoft\Windows\Start Menu\Programs\Pointhole Client.lnk');$s.TargetPath='%APPDATA%\pointhole\client.exe';$s.Save()"
echo Shortcut created
echo Installation complete, you can now run the client from the Start Menu.
goto end

:arm
echo ARM is not supported by this script. You can manually build the client from source code tho.
goto end

:end
endlocal
cd /d %~dp0
pause