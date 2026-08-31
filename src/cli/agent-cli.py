#!/usr/bin/env python3
"""
VaultAgent — CLI tool (server-side)

A command-line tool to check code, install/manage the systemd service,
inspect config, update the software and query the running agent.

Usage:
    agent-cli check                 Run code checks (go vet, staticcheck, structure)
    agent-cli install               Launch the appropriate installer
    agent-cli start                 Start the vaultagent service (systemd)
    agent-cli stop                  Stop the vaultagent service (systemd)
    agent-cli restart               Restart the vaultagent service (systemd)
    agent-cli status                Show service status
    agent-cli config                Show the current configuration
    agent-cli update                Pull latest code and rebuild
    agent-cli ask <prompt>          Ask the running server agent directly
    agent-cli help                  Show this help
"""

import sys
import os
import json
import shutil
import subprocess
import argparse

VERSION = "0.4 (Standard)"
REPO_URL = "https://github.com/rhantschk-cmyk/agent-Project"
SERVICE_NAME = "vaultagent"

CLIENT_CONFIG_REL = os.path.join("src", "Client", "config.json")
SERVER_CONFIG_PATHS = [
    os.path.join("src", "Server", "config.json"),
    os.path.join(os.sep, "etc", "vaultagent", "config.json"),
]


def banner() -> None:
    print(f"VaultAgent CLI v{VERSION} ({REPO_URL})")


def print_error(msg: str) -> None:
    print(f"[ERROR] {msg}", file=sys.stderr)


def run_cmd(cmd: list, cwd: str = None) -> int:
    print(f"[CLI] running: {' '.join(cmd)}")
    result = subprocess.run(cmd, cwd=cwd)
    return result.returncode


def capture(cmd: list, cwd: str = None) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True)


def is_root() -> bool:
    return os.geteuid() == 0


def require_root() -> bool:
    if is_root():
        return True
    print_error("This command requires root permissions (sudo).")
    return False


# ---------------------------------------------------------------------------
# check
# ---------------------------------------------------------------------------

def check_structure() -> None:
    required = [
        os.path.join("src", "Server", "main.go"),
        os.path.join("src", "Server", "config.go"),
        os.path.join("src", "Server", "agent.go"),
        os.path.join("src", "Server", "mails.go"),
        os.path.join("src", "Server", "tools.go"),
        os.path.join("src", "Server", "memory.go"),
        os.path.join("src", "Server", "cliserver.go"),
        os.path.join("src", "Server", "monitoringserver.go"),
        os.path.join("src", "Server", "go.mod"),
        os.path.join("src", "Client", "main.odin"),
    ]
    ok = True
    for f in required:
        if os.path.exists(f):
            print(f"[OK]   {f}")
        else:
            print(f"[FAIL] missing: {f}")
            ok = False
    return ok


def cmd_check(args) -> int:
    code = 0
    print("=== Structure check ===")
    if not check_structure():
        code = 1

    if os.path.exists(os.path.join("src", "Server", "go.mod")):
        print("\n=== go vet ===")
        if run_cmd(["go", "vet", "./..."], cwd=os.path.join("src", "Server")) != 0:
            code = 1

        print("\n=== go test ===")
        if run_cmd(["go", "test", "./..."], cwd=os.path.join("src", "Server")) != 0:
            code = 1
    else:
        print("[SKIP] go.mod not found; run from repo root")

    print("\n=== staticcheck (if available) ===")
    if shutil.which("staticcheck"):
        if run_cmd(["staticcheck", "./..."], cwd=os.path.join("src", "Server")) != 0:
            code = 1
    else:
        print("[INFO] staticcheck not installed; skipping (install with: go install honnef.co/go/tools/cmd/staticcheck@latest)")

    return code


# ---------------------------------------------------------------------------
# install
# ---------------------------------------------------------------------------

def cmd_install(args) -> int:
    print("=== Installer ===")
    print("Which component do you want to install?")
    print("  [1] Server (Linux / systemd)")
    print("  [2] Desktop client (Windows)")
    choice = input("Choice [1/2]: ").strip()
    if choice == "1":
        if not require_root():
            return 1
        return run_cmd([sys.executable, os.path.join("src", "install", "server_installer.py")])
    if choice == "2":
        py = "python" if os.name == "nt" else sys.executable
        return run_cmd([py, os.path.join("src", "install", "desktop_installer.py")])
    print_error("Invalid choice.")
    return 1


def cmd_uninstall(args) -> int:
    print("=== Uninstaller ===")
    print("Which component do you want to uninstall?")
    print("  [1] Server (Linux / systemd)")
    print("  [2] Desktop client (Windows)")
    choice = input("Choice [1/2]: ").strip()
    if choice == "1":
        if not require_root():
            return 1
        return run_cmd([sys.executable, os.path.join("src", "install", "server_uninstaller.py")])
    if choice == "2":
        py = "python" if os.name == "nt" else sys.executable
        return run_cmd([py, os.path.join("src", "install", "desktop_uninstaller.py")])
    print_error("Invalid choice.")
    return 1


# ---------------------------------------------------------------------------
# service control
# ---------------------------------------------------------------------------

def cmd_service(action: str) -> int:
    if not require_root():
        return 1
    return run_cmd(["systemctl", action, SERVICE_NAME])


def cmd_status(args=None) -> int:
    print("=== Service status ===")
    return run_cmd(["systemctl", "status", SERVICE_NAME, "--no-pager"])


# ---------------------------------------------------------------------------
# config
# ---------------------------------------------------------------------------

def cmd_config(args=None) -> int:
    for path in SERVER_CONFIG_PATHS:
        if os.path.exists(path):
            print(f"=== Config: {path} ===")
            try:
                with open(path, "r", encoding="utf-8") as fh:
                    print(fh.read())
            except OSError as e:
                print_error(f"Could not read {path}: {e}")
                return 1
            return 0
    print("[WARN] no config.json found")
    return 0


# ---------------------------------------------------------------------------
# update
# ---------------------------------------------------------------------------

def cmd_update(args=None) -> int:
    print("=== Update ===")
    if run_cmd(["git", "pull"]) != 0:
        return 1
    print("\nRebuilding server binary...")
    code = run_cmd(
        ["go", "build", "-o", "/usr/local/bin/vaultagent", "."],
        cwd=os.path.join("src", "Server"),
    )
    print("Restart the service to apply: agent-cli restart")
    return code


# ---------------------------------------------------------------------------
# ask (query running agent over HTTP)
# ---------------------------------------------------------------------------

def parse_json_key(s: str, key: str) -> str:
    import re
    m = re.search(r'"%s"\s*:\s*"([^"]*)"' % re.escape(key), s)
    return m.group(1) if m else ""


def cmd_ask(args) -> int:
    prompt = " ".join(args.prompt)
    server_ip = "localhost"
    verify_key = ""
    if os.path.exists(CLIENT_CONFIG_REL):
        try:
            with open(CLIENT_CONFIG_REL, "r", encoding="utf-8") as fh:
                text = fh.read()
            server_ip = parse_json_key(text, "server_ip") or "localhost"
            verify_key = parse_json_key(text, "verify_key")
        except OSError:
            pass
    url = f"http://{server_ip}:8080/api/agent/ask"
    payload = json.dumps({"verify_key": verify_key, "prompt": prompt})
    return run_cmd([
        "curl", "-s", "-X", "POST", url,
        "-H", "Content-Type: application/json", "-d", payload,
    ])


# ---------------------------------------------------------------------------
# help
# ---------------------------------------------------------------------------

def usage() -> None:
    banner()
    print("")
    print("USAGE:")
    print("  agent-cli check                 Run code checks (go vet, staticcheck, structure)")
    print("  agent-cli install               Launch the appropriate installer")
    print("  agent-cli uninstall             Launch the appropriate uninstaller")
    print("  agent-cli start                 Start the vaultagent service (systemd)")
    print("  agent-cli stop                  Stop the vaultagent service (systemd)")
    print("  agent-cli restart               Restart the vaultagent service (systemd)")
    print("  agent-cli status                Show service status")
    print("  agent-cli config                Show the current configuration")
    print("  agent-cli update                Pull latest code and rebuild")
    print("  agent-cli ask <prompt>          Ask the server agent directly")
    print("  agent-cli help                  Show this help")
    print("")
    print(f"NOTE: Standard version v{VERSION}. Pro version (unlimited monitors) coming soon.")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="agent-cli",
        description=f"VaultAgent CLI v{VERSION}",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="Run 'agent-cli help' for full usage.",
    )
    sub = parser.add_subparsers(dest="command")
    sub.add_parser("help", help="Show help")

    sub.add_parser("check", help="Run code checks")
    sub.add_parser("install", help="Launch the installer")
    sub.add_parser("uninstall", help="Launch the uninstaller")
    sub.add_parser("start", help="Start the service")
    sub.add_parser("stop", help="Stop the service")
    sub.add_parser("restart", help="Restart the service")
    sub.add_parser("status", help="Show service status")
    sub.add_parser("config", help="Show current config")
    sub.add_parser("update", help="Pull latest code and rebuild")

    ask = sub.add_parser("ask", help="Ask the server agent directly")
    ask.add_argument("prompt", nargs="+", help="The prompt to send to the agent")

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()

    if args.command is None or args.command in ("help",):
        usage()
        return 0

    handlers = {
        "check": cmd_check,
        "install": cmd_install,
        "uninstall": cmd_uninstall,
        "start": lambda a: cmd_service("start"),
        "stop": lambda a: cmd_service("stop"),
        "restart": lambda a: cmd_service("restart"),
        "status": cmd_status,
        "config": cmd_config,
        "update": cmd_update,
        "ask": cmd_ask,
    }
    return handlers[args.command](args)


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("\n\nAborted.")
        sys.exit(130)
