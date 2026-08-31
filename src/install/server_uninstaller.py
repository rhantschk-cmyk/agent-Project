#!/usr/bin/env python3
"""
Agent Software — Server Uninstaller (Linux / systemd)

A terminal-based tool that removes the email-agent systemd service and all
files the server installer created.

Removes:
  - systemd service (stopped + disabled + unit file removed)
  - binary /usr/local/bin/email-agent
  - config dir /etc/email-agent (including config.json and docs)
  - optional: the cloned repo at /opt/email-agent

Run with sudo:
    sudo python3 server_uninstaller.py
"""

import os
import sys
import shutil
import subprocess

SERVICE_NAME = "email-agent"
BINARY_PATH = "/usr/local/bin/email-agent"
CONFIG_DIR = "/etc/email-agent"
WORK_DIR = "/opt/email-agent"
UNIT_PATH = f"/etc/systemd/system/{SERVICE_NAME}.service"


def banner() -> None:
    print("==============================================")
    print("  Agent Software Server Uninstaller")
    print("==============================================")


def error(msg: str) -> None:
    print(f"[ERROR] {msg}")
    sys.exit(1)


def info(msg: str) -> None:
    print(f"[INFO]  {msg}")


def ok(msg: str) -> None:
    print(f"[OK]    {msg}")


def run(cmd: list, check: bool = True) -> subprocess.CompletedProcess:
    result = subprocess.run(cmd, capture_output=True, text=True)
    if check and result.returncode != 0:
        print(f"[WARN]  Command failed: {' '.join(cmd)}\n{result.stderr.strip()}")
    return result


def is_root() -> bool:
    return os.geteuid() == 0


def confirm(prompt: str, default: bool = False) -> bool:
    suffix = "(y/N)" if not default else "(Y/n)"
    answer = input(f"{prompt} {suffix}: ").strip().lower()
    if not answer:
        return default
    return answer in ("y", "yes")


def stop_and_remove_service() -> None:
    info(f"Stopping and disabling systemd service '{SERVICE_NAME}'...")
    run(["systemctl", "stop", SERVICE_NAME], check=False)
    run(["systemctl", "disable", SERVICE_NAME], check=False)
    if os.path.exists(UNIT_PATH):
        run(["rm", "-f", UNIT_PATH])
        run(["systemctl", "daemon-reload"])
        run(["systemctl", "reset-failed", SERVICE_NAME], check=False)
        ok(f"Removed systemd unit {UNIT_PATH}")
    else:
        ok("No systemd unit found")


def remove_files() -> None:
    for path, label in [
        (BINARY_PATH, "binary"),
        (CONFIG_DIR, "config directory (config.json + docs)"),
    ]:
        if os.path.exists(path):
            if os.path.isdir(path):
                shutil.rmtree(path)
            else:
                os.remove(path)
            ok(f"Removed {label}: {path}")
        else:
            ok(f"No {label} present: {path}")


def remove_repo() -> None:
    if os.path.isdir(WORK_DIR):
        if confirm(f"Remove cloned repository at {WORK_DIR}?", default=False):
            shutil.rmtree(WORK_DIR)
            ok(f"Removed repository {WORK_DIR}")
        else:
            ok("Keeping repository (left in place)")


def final_message() -> None:
    print("\n==============================================")
    print("  Uninstall complete")
    print(f"  Service: {SERVICE_NAME} (stopped + disabled)")
    print(f"  Binary:  {BINARY_PATH} (removed)")
    print(f"  Config:  {CONFIG_DIR} (removed)")
    print("==============================================")
    print("  Note: your Gmail App Token was stored in the removed config.")
    print("  If you reinstall, create a new App Token and update config.json.")


def main() -> None:
    banner()
    if not is_root():
        error("Please run this uninstaller as root / with sudo.")

    if not confirm("This will remove the email-agent service and config. Continue?", default=False):
        print("Aborted.")
        return

    stop_and_remove_service()
    remove_files()
    remove_repo()
    final_message()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nUninstall cancelled by user.")
        sys.exit(130)
