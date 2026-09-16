# Records promo/demo.gif: launches oos with a synthetic HOME (see
# make-demo-db.py), types "bug fix", moves selection, quits.
# Capture via ffmpeg gdigrab (window title match), keys via SendKeys.
#
# Usage: powershell -ExecutionPolicy Bypass -File promo\record-demo.ps1
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms

$Repo     = Split-Path $PSScriptRoot -Parent
$DemoHome = Join-Path $env:TEMP "oosdemo"
$DbPath   = Join-Path $DemoHome ".local\share\opencode\opencode.db"
$OosExe   = Join-Path $Repo "oos.exe"
$OutMp4   = Join-Path $env:TEMP "oosdemo-cap.mp4"
$OutGif   = Join-Path $Repo "demo.gif"

if (-not (Test-Path $DbPath))  { throw "demo DB missing: $DbPath (run promo/make-demo-db.py first)" }
if (-not (Test-Path $OosExe))  { throw "oos.exe missing: run go build first" }
Remove-Item $OutMp4 -ErrorAction SilentlyContinue

$shell = New-Object -ComObject WScript.Shell
function Focus-Demo {
  for ($i = 0; $i -lt 30; $i++) {
    if ($shell.AppActivate("oos demo")) { return }
    Start-Sleep -Milliseconds 250
  }
  throw "cannot focus 'oos demo' window"
}
function Type-Text([string]$text) {
  foreach ($ch in $text.ToCharArray()) {
    [System.Windows.Forms.SendKeys]::SendWait($ch)
    Start-Sleep -Milliseconds 90
  }
}

# 0. preflight: probe must pass (DB + driver healthy) before the take
$env:USERPROFILE = $DemoHome
Write-Host "preflight probe..."
go test -run TestProbeDemoDB -count=1 . 2>&1 | Select-Object -Last 3
if ($LASTEXITCODE -ne 0) { throw "preflight probe FAILED, aborting take" }

# 0b. pre-clean stale demo windows (poll below must match a fresh one)
Get-Process conhost -ErrorAction SilentlyContinue |
  Where-Object { $_.MainWindowTitle -eq "oos demo" } |
  Stop-Process -Force -ErrorAction SilentlyContinue

# 1. launch demo console: explicit conhost (own window, never a WT tab)
# hosting powershell (clean env, no cmd quoting hell) running oos.
# Default 120x30 console fits the 99-col layout, no resizing needed.
$Launcher = Join-Path $DemoHome "launch-oos.ps1"
@(
  '$host.UI.RawUI.WindowTitle = "oos demo"',
  '$env:USERPROFILE = "' + $DemoHome + '"',
  '"USERPROFILE=$env:USERPROFILE" | Out-File "' + (Join-Path $DemoHome "run.log") + '" -Encoding ascii',
  '& "' + $OosExe + '"'
) | Set-Content $Launcher -Encoding ascii
$con = Start-Process conhost.exe -ArgumentList @(
  "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass",
  "-NoExit", "-File", $Launcher) -PassThru
$found = $false
for ($i = 0; $i -lt 40; $i++) {
  Start-Sleep -Milliseconds 250
  if (Get-Process | Where-Object { $_.MainWindowTitle -eq "oos demo" }) {
    $found = $true; break
  }
}
if (-not $found) { throw "demo window never appeared (oos failed to start?)" }
Start-Sleep -Milliseconds 2500

# 2. start capture (ffmpeg exits cleanly by itself after -t seconds)
$ffArgs = @("-y", "-f", "gdigrab", "-framerate", "12",
  "-i", "title=oos demo", "-pix_fmt", "yuv420p", "-t", "22", $OutMp4)
$ff = Start-Process ffmpeg -ArgumentList $ffArgs -PassThru -WindowStyle Hidden
Start-Sleep -Milliseconds 1000

# 3. scripted demo: search -> select -> quit
Focus-Demo
Type-Text "bug fix"
Start-Sleep -Milliseconds 1600
[System.Windows.Forms.SendKeys]::SendWait("{DOWN}")
Start-Sleep -Milliseconds 1200
[System.Windows.Forms.SendKeys]::SendWait("{ESC}")
Start-Sleep -Milliseconds 800

# 4. finish capture + convert to optimized GIF
Wait-Process -Id $ff.Id -Timeout 30 -ErrorAction SilentlyContinue
if (-not $con.HasExited) { Stop-Process -Id $con.Id -Force }
if (-not (Test-Path $OutMp4)) { throw "capture failed: $OutMp4 not created" }

$vf = "fps=10,scale=880:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse"
& ffmpeg -y -i $OutMp4 -vf $vf $OutGif | Out-Null
if (-not (Test-Path $OutGif)) { throw "gif convert failed" }
Remove-Item $OutMp4 -ErrorAction SilentlyContinue

$size = (Get-Item $OutGif).Length
Write-Host "OK: $OutGif ($([math]::Round($size/1KB)) KB)"
Write-Host ("env proof: " + (Get-Content (Join-Path $DemoHome "run.log")))
