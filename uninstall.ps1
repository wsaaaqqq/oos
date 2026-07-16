# oos Windows uninstaller
$InstallDir = "$env:USERPROFILE\.local\bin"
$Target = "$InstallDir\oos.exe"

if (Test-Path $Target) {
  Remove-Item $Target -Force
  Write-Host "oos removed from $InstallDir"
} else {
  Write-Host "oos not found in $InstallDir"
}
