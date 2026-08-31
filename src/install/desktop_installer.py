#!/usr/bin/env python3
"""
VaultAgent — Desktop Installer (Windows)

A terminal-based installer that:
  1. Checks for the Odin compiler.
  2. Builds the desktop client (Odin / raylib).
  3. Copies the executable + config into C:\\Program Files\\VaultAgent\\.
  4. Creates a desktop shortcut.
  5. Asks for the server IP and verify key.

Run (as normal user, not admin required for per-user install):
    python desktop_installer.py
"""

import os
import sys
import json
import shutil
import subprocess
import platform

VERSION = "v0.4 (Standard)"
REPO_DIR = None  # filled later
CLIENT_SRC = os.path.join("src", "Client")
INSTALL_DIR = os.path.join(os.environ.get("ProgramFiles", "C:\\Program Files"), "VaultAgent")
APP_NAME = "VaultAgent"
EXE_NAME = "VaultAgent.exe"
SHORTCUT_NAME = "VaultAgent.lnk"


def banner() -> None:
    print("==============================================")
    print("  VaultAgent Desktop Installer")
    print(f"  Version {VERSION}")
    print("==============================================")


def error(msg: str) -> None:
    print(f"[ERROR] {msg}")
    sys.exit(1)


def info(msg: str) -> None:
    print(f"[INFO]  {msg}")


def ok(msg: str) -> None:
    print(f"[OK]    {msg}")


def warn(msg: str) -> None:
    print(f"[WARN]  {msg}")


def run(cmd: list, cwd: str = None, check: bool = True) -> subprocess.CompletedProcess:
    result = subprocess.run(cmd, capture_output=True, text=True, cwd=cwd)
    if check and result.returncode != 0:
        error(f"Command failed: {' '.join(cmd)}\n{result.stdout}\n{result.stderr}")
    return result


def check_windows() -> None:
    if os.name != "nt":
        warn("This installer is designed for Windows. Running anyway...")
    info(f"OS: {platform.system()}")


def find_repo() -> str:
    """Locate the repository root from the installer's location."""
    script_dir = os.path.dirname(os.path.abspath(__file__))
    candidate = os.path.abspath(os.path.join(script_dir, "..", ".."))
    if os.path.isdir(os.path.join(candidate, "src", "Client")):
        return candidate
    error("Could not locate the repository root (src/Client not found).")
    return ""


def check_odin() -> str:
    info("Checking Odin compiler...")
    odin = shutil.which("odin")
    if not odin:
        # Common install location
        odin = os.path.join(os.environ.get("USERPROFILE", ""), "odin", "odin.exe")
        if not os.path.exists(odin):
            error("Odin compiler not found. Install Odin and add it to PATH.")
    ok(f"Odin found at {odin}")
    return odin


def build_client(odin: str, repo: str) -> str:
    info("Building desktop client...")
    client_dir = os.path.join(repo, CLIENT_SRC)
    out_dir = os.path.join(repo, "build")
    os.makedirs(out_dir, exist_ok=True)
    out_path = os.path.join(out_dir, EXE_NAME)
    run([odin, "build", client_dir, "-out:" + out_path], cwd=repo)
    if not os.path.exists(out_path):
        error("Build finished but the executable was not found.")
    ok(f"Built {out_path}")
    return out_path


def ask_server_config() -> dict:
    print("\n--- Client <-> Server connection ---")
    server_ip = input("Server IP or hostname [localhost]: ").strip() or "localhost"
    verify_key = input("Verify key (must match server cli_secret_key): ").strip()
    if not verify_key:
        warn("Verify key left empty. Requests from this client will be rejected.")
    return {"server_ip": server_ip, "verify_key": verify_key}


def install_files(exe_path: str, config: dict) -> None:
    info(f"Installing to {INSTALL_DIR}...")
    os.makedirs(INSTALL_DIR, exist_ok=True)
    shutil.copy(exe_path, os.path.join(INSTALL_DIR, EXE_NAME))
    config_path = os.path.join(INSTALL_DIR, "config.json")
    with open(config_path, "w", encoding="utf-8") as fh:
        json.dump(config, fh, indent=2)
    ok("Files installed.")


def create_shortcut() -> None:
    info("Creating desktop shortcut...")
    desktop = os.path.join(os.environ.get("USERPROFILE", ""), "Desktop")
    skip = False
    if not os.path.isdir(desktop):
        warn("Desktop directory not found.")
        skip = True
    if not skip:
        exe = os.path.join(INSTALL_DIR, EXE_NAME)
        shortcut = os.path.join(desktop, SHORTCUT_NAME)
        try:
            # Prefer win32com (pip install pywin32) for a proper .lnk
            import win32com.client  # type: ignore
            shell = win32com.client.Dispatch("WScript.Shell")
            lnk = shell.CreateShortCut(shortcut)
            lnk.TargetPath = exe
            lnk.WorkingDirectory = INSTALL_DIR
            lnk.IconLocation = exe
            lnk.Save()
        except ImportError:
            # Fallback: create a simple .bat-based shortcut via PowerShell
            ps = (
                f"$s=(New-Object -COM WScript.Shell).CreateShortcut('{shortcut}');"
                f"$s.TargetPath='{exe}';$s.WorkingDirectory='{INSTALL_DIR}';$s.Save()"
            )
            result = subprocess.run(
                ["powershell", "-NoProfile", "-Command", ps],
                capture_output=True, text=True, check=False,
            )
            if result.returncode != 0:
                warn("Could not create shortcut via PowerShell. You can create it manually.")
                return
        ok(f"Shortcut created at {shortcut}")


def final_message() -> None:
    print("\n==============================================")
    print("  Desktop installation complete!")
    print(f"  Version: {VERSION}")
    print(f"  Install dir: {INSTALL_DIR}")
    print(f"  Shortcut:    {SHORTCUT_NAME}")
    print("\n  Pro version (coming soon): unlimited monitors.")
    print("==============================================")


def main() -> None:
    global REPO_DIR
    banner()
    check_windows()
    REPO_DIR = find_repo()
    odin = check_odin()
    exe = build_client(odin, REPO_DIR)
    config = ask_server_config()
    install_files(exe, config)
    create_shortcut()
    final_message()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nInstallation cancelled by user.")
        sys.exit(130)
