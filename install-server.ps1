#Requires -RunAsAdministrator

# ARM check
if ($env:PROCESSOR_ARCHITECTURE -match "ARM" -or $env:PROCESSOR_ARCHITEW6432 -match "ARM") {
    Write-Host "ARM is not supported by this script. You can manually build the server from source code tho." -ForegroundColor Yellow
    exit 1
}

# Setup
$installDir = Join-Path $env:APPDATA "pointhole"
$serverPath = Join-Path $installDir "pointserver.exe"

New-Item -ItemType Directory -Path $installDir -Force | Out-Null
Set-Location $installDir

# Download
Write-Host "Downloading pointhole/server..."
Invoke-WebRequest -Uri "https://cdn.lu2000luk.com/pointhole/server/server.exe?commit=e80d169" -OutFile $serverPath -UseBasicParsing # add last commit to the url to bust the cache
Write-Host "Download complete" -ForegroundColor Green

# Add to PATH
Write-Host "Adding to path..."
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$pathEntries = $userPath -split ";" | Where-Object { $_ -ne "" }

if ($installDir -notin $pathEntries) {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    # Also update current session
    $env:Path = "$env:Path;$installDir"
    Write-Host "Added to PATH" -ForegroundColor Green
} else {
    Write-Host "Already in PATH" -ForegroundColor DarkGray
}

Write-Host 'Installation complete, you can now run the server with the command "pointserver" in the terminal.' -ForegroundColor Cyan