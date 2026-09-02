#!/usr/bin/env bash
#
# VaultAgent — build bundled CLI executable
#
# Bundles agent-cli.py + all four installer modules into a single onefile binary.
#
#   ./compiled/build.sh          Linux ELF
#   ./compiled/build.sh windows  Windows .exe (needs wine + Python3 in wine)
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

OUT="$ROOT/compiled"
WORK="$ROOT/compiled/_build"
DIST="$ROOT/compiled/_dist"

rm -rf "$WORK" "$DIST"
mkdir -p "$OUT" "$WORK" "$DIST"

check_pyinstaller() {
    if ! python3 -c 'import PyInstaller' 2>/dev/null; then
        echo "[build] PyInstaller not found — installing into a local venv..."
        python3 -m venv "$ROOT/compiled/.venv"
        "$ROOT/compiled/.venv/bin/pip" install pyinstaller >/dev/null
        PYTHON="$ROOT/compiled/.venv/bin/python3"
    else
        PYTHON="$(command -v python3)"
    fi
}

build_linux() {
    check_pyinstaller
    echo "[build] Python : $PYTHON"
    $PYTHON -c 'import PyInstaller; print("[build] PyInstaller", PyInstaller.__version__)'
    echo ""

    cd "$OUT"   # keep PyInstaller .spec artifacts inside compiled/, not repo root
    rm -f VaultAgent.spec
    $PYTHON -m PyInstaller \
        --name VaultAgent \
        --onefile \
        --console \
        --distpath "$DIST" \
        --workpath "$WORK" \
        --paths "$ROOT/src" \
        --hidden-import install \
        --hidden-import install.server_installer \
        --hidden-import install.desktop_installer \
        --hidden-import install.server_uninstaller \
        --hidden-import install.desktop_uninstaller \
        --noconfirm \
        "$ROOT/src/cli/agent-cli.py"

    cp "$DIST/VaultAgent" "$OUT/VaultAgent"
    chmod +x "$OUT/VaultAgent"
    echo "[build] Linux binary -> $OUT/VaultAgent"
}

build_windows() {
    if ! command -v wine >/dev/null 2>&1; then
        echo "[build] 'wine' not installed."
        echo "[build] On Windows, run:"
        echo "  python -m pip install pyinstaller"
        echo "  python -m PyInstaller --name VaultAgent --onefile --console --paths src --hidden-import install --hidden-import install.server_installer --hidden-import install.desktop_installer --hidden-import install.server_uninstaller --hidden-import install.desktop_uninstaller --noconfirm src/cli/agent-cli.py"
        echo "  copy _dist\\VaultAgent.exe compiled\\VaultAgent.exe"
        exit 1
    fi
    echo "[build] Building Windows .exe via wine..."

    wine python -m PyInstaller \
        --name VaultAgent \
        --onefile \
        --console \
        --distpath "$(winepath -w "$DIST")" \
        --workpath "$(winepath -w "$WORK")" \
        --paths "$(winepath -w "$ROOT/src")" \
        --hidden-import install \
        --hidden-import install.server_installer \
        --hidden-import install.desktop_installer \
        --hidden-import install.server_uninstaller \
        --hidden-import install.desktop_uninstaller \
        --noconfirm \
        "$(winepath -w "$ROOT/src/cli/agent-cli.py")"

    cp "$DIST/VaultAgent.exe" "$OUT/VaultAgent.exe"
    echo "[build] Windows .exe -> $OUT/VaultAgent.exe"
}

case "${1:-linux}" in
    linux|Linux) build_linux ;;
    windows|Windows|win|exe) build_windows ;;
    *) echo "Usage: $0 [linux|windows]"; exit 1 ;;
esac

echo "[build] Compiled files:"
ls -la "$OUT"