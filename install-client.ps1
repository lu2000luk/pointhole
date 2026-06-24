# ARM check
if ($env:PROCESSOR_ARCHITECTURE -match "ARM" -or $env:PROCESSOR_ARCHITEW6432 -match "ARM") {
    Write-Host "ARM is not supported by this script. You can manually build the client from source code tho." -ForegroundColor Yellow
    exit 1
}

# Setup
$installDir = Join-Path $env:APPDATA "pointhole"
$clientPath = Join-Path $installDir "client.exe"
$shortcutPath = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Pointhole Client.lnk"

New-Item -ItemType Directory -Path $installDir -Force | Out-Null
Set-Location $installDir

# Download
Write-Host "Downloading pointhole/client..."
Invoke-WebRequest -Uri "https://cdn.lu2000luk.com/pointhole/client/client.exe?commit=e80d169" -OutFile $clientPath -UseBasicParsing # add last commit to the url to bust the cache
Write-Host "Download complete" -ForegroundColor Green

# Create shortcut
Write-Host "Creating shortcut..."
$WshShell = New-Object -ComObject WScript.Shell
$shortcut = $WshShell.CreateShortcut($shortcutPath)
$shortcut.TargetPath = $clientPath
$shortcut.Save()
[Runtime.InteropServices.Marshal]::ReleaseComObject($WshShell) | Out-Null
Write-Host "Shortcut created" -ForegroundColor Green

Write-Host "Installation complete, you can now run the client from the Start Menu." -ForegroundColor Cyan