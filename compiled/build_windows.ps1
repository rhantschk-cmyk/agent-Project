# VaultAgent — build bundled Windows executable
#
# Run this on a Windows machine with Python 3.12+ installed.
#
#   python -m pip install pyinstaller
#   .\compiled\build_windows.ps1
#
# Produces: compiled\VaultAgent.exe

$ErrorActionPreference = "Stop"
$ROOT = Split-Path -Parent (Split-Path -Parent $PSCommandPath)
$OUT  = "$ROOT\compiled"
$WORK = "$ROOT\compiled\_build"
$DIST = "$ROOT\compiled\_dist"

Remove-Item -Recurse -Force $WORK -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force $DIST -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $WORK,$DIST,$OUT -Force | Out-Null

python -m PyInstaller `
    --name VaultAgent `
    --onefile `
    --console `
    --distpath $DIST `
    --workpath $WORK `
    --paths "$ROOT\src" `
    --hidden-import install `
    --hidden-import install.server_installer `
    --hidden-import install.desktop_installer `
    --hidden-import install.server_uninstaller `
    --hidden-import install.desktop_uninstaller `
    --noconfirm `
    "$ROOT\src\cli\agent-cli.py"

Copy-Item "$DIST\VaultAgent.exe" "$OUT\VaultAgent.exe" -Force
Write-Host "[build] Windows .exe -> $OUT\VaultAgent.exe"
