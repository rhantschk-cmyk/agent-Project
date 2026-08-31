#!/usr/bin/env python3
"""
VaultAgent — Desktop Uninstaller (Windows)

A terminal-based tool that removes the installed desktop client.

Removes:
  - C:\\Program Files\\VaultAgent\\ (executable + config)
  - the desktop shortcut "VaultAgent.lnk"

Run:
    python desktop_uninstaller.py
"""

import os
import sys
import shutil
import platform

INSTALL_DIR = os.path.join(os.environ.get("ProgramFiles", "C:\\Program Files"), "VaultAgent")
SHORTCUT_NAME = "VaultAgent.lnk"


def banner() -> None:
    print("==============================================")
    print("  VaultAgent Desktop Uninstaller")
    print("==============================================")


def info(msg: str) -> None:
    print(f"[INFO]  {msg}")


def ok(msg: str) -> None:
    print(f"[OK]    {msg}")


def confirm(prompt: str, default: bool = False) -> bool:
    suffix = "(y/N)" if not default else "(Y/n)"
    answer = input(f"{prompt} {suffix}: ").strip().lower()
    if not answer:
        return default
    return answer in ("y", "yes")


def remove_install_dir() -> None:
    if os.path.isdir(INSTALL_DIR):
        shutil.rmtree(INSTALL_DIR, ignore_errors=True)
        if not os.path.isdir(INSTALL_DIR):
            ok(f"Removed install directory: {INSTALL_DIR}")
        else:
            info(f"Could not fully remove (in use?): {INSTALL_DIR}")
    else:
        ok(f"No install directory present: {INSTALL_DIR}")


def remove_shortcut() -> None:
    desktop = os.path.join(os.environ.get("USERPROFILE", ""), "Desktop")
    shortcut = os.path.join(desktop, SHORTCUT_NAME)
    if os.path.exists(shortcut):
        os.remove(shortcut)
        ok(f"Removed desktop shortcut: {shortcut}")
    else:
        ok(f"No desktop shortcut present: {shortcut}")


def final_message() -> None:
    print("\n==============================================")
    print("  Desktop uninstall complete")
    print(f"  Install dir: {INSTALL_DIR} (removed)")
    print(f"  Shortcut:    {SHORTCUT_NAME} (removed)")
    print("==============================================")


def main() -> None:
    banner()
    if os.name != "nt":
        info("Not running on Windows; paths may not resolve. Continuing anyway.")

    if not confirm("This will remove the VaultAgent desktop app. Continue?", default=False):
        print("Aborted.")
        return

    remove_install_dir()
    remove_shortcut()
    final_message()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nUninstall cancelled by user.")
        sys.exit(130)
