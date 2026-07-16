# oos Windows installer
$ErrorActionPreference = "Stop"

$Repo = "wsaaaqqq/oos"
$InstallDir = "$env:USERPROFILE\.local\bin"

$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "arm64" }

$LatestUrl = "https://api.github.com/repos/$Repo/releases/latest"
$Release = Invoke-RestMethod -Uri $LatestUrl
$Asset = $Release.assets | Where-Object { $_.name -eq "oos_windows_${Arch}.exe" }

if (-not $Asset) {
  Write-Error "Could not find release for windows/${Arch}"
  exit 1
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Write-Host "Downloading oos windows/${Arch}..."
Invoke-WebRequest -Uri $Asset.browser_download_url -OutFile "$InstallDir\oos.exe"

$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$InstallDir*") {
  Write-Host ""
  Write-Host "Add to your PATH to use 'oos' globally:"
  Write-Host "  [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$InstallDir', 'User')"
}

Write-Host ""
Write-Host "oos installed to $InstallDir\oos.exe"
Write-Host "Try: oos"
