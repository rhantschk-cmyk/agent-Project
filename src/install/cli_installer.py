#!/usr/bin/env python3
"""
VaultAgent — CLI self-install

Installs the CLI tool itself as a standalone, global program:

  - when run from the frozen PyInstaller binary, the binary is copied to
    /usr/local/bin/agent-cli;
  - when run from a source checkout, the `cli` and `install` packages are
    copied to /usr/local/lib/vaultagent-cli/ and a launcher is created at
    /usr/local/bin/agent-cli.

Afterwards `agent-cli` works from any directory without the repository.

Run with sudo:
    sudo agent-cli self-install
"""

import os
import shutil
import sys

INSTALL_LIB = "/usr/local/lib/vaultagent-cli"
LAUNCHER = "/usr/local/bin/agent-cli"
VERSION = "v0.5 (Standard + Pro)"


def banner() -> None:
    print("==============================================")
    print("  VaultAgent CLI self-install")
    print(f"  Version {VERSION}")
    print("==============================================")


def error(msg: str) -> None:
    print(f"[ERROR] {msg}")
    sys.exit(1)


def info(msg: str) -> None:
    print(f"[INFO]  {msg}")


def ok(msg: str) -> None:
    print(f"[OK]    {msg}")


def is_root() -> bool:
    return os.geteuid() == 0 if hasattr(os, "geteuid") else False


def copy_tree(src: str, dst: str) -> None:
    if os.path.isdir(dst):
        shutil.rmtree(dst)
    shutil.copytree(src, dst)


def install_from_source() -> None:
    """Copy cli + install packages into INSTALL_LIB and create a launcher."""
    here = os.path.dirname(os.path.abspath(__file__))   # .../install
    src_root = os.path.dirname(here)                    # .../src

    info("Creating install library...")
    os.makedirs(INSTALL_LIB, exist_ok=True)

    info("Copying cli package...")
    copy_tree(os.path.join(src_root, "cli"), os.path.join(INSTALL_LIB, "cli"))
    info("Copying install package...")
    copy_tree(os.path.join(src_root, "install"), os.path.join(INSTALL_LIB, "install"))

    launcher = f"""#!/usr/bin/env python3
import sys
sys.path.insert(0, {INSTALL_LIB!r})
from cli.agent_cli import main

if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("\\n\\nAborted.")
        sys.exit(130)
"""
    with open(LAUNCHER, "w", encoding="utf-8") as fh:
        fh.write(launcher)
    os.chmod(LAUNCHER, 0o755)
    ok(f"Sources installed to {INSTALL_LIB}")
    ok(f"Launcher installed to {LAUNCHER}")


def install_from_binary() -> None:
    shutil.copy(sys.executable, LAUNCHER)
    os.chmod(LAUNCHER, 0o755)
    ok(f"Binary installed to {LAUNCHER}")


def main() -> None:
    banner()
    if not is_root():
        error("Please run this installer as root / with sudo.")
    if "site-packages" in os.path.abspath(__file__):
        info("Running from an installed package (pip).")
        if os.path.exists(LAUNCHER):
            ok(f"agent-cli already available at {LAUNCHER}")
        else:
            info("The pip-installed 'agent-cli' command should already be on your PATH.")
        return
    if getattr(sys, "_MEIPASS", None):
        install_from_binary()
    else:
        install_from_source()
    print("\nTry it: agent-cli version")


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nInstallation cancelled by user.")
        sys.exit(130)