#!/usr/bin/env python3
"""
VaultAgent — CLI tool (server-side) for the Standard and Pro editions.

A command-line tool to check code, install/manage the systemd service,
inspect config, update the software and query the running agent.

The Pro edition is a separate, intentionally private (closed-source)
product. Its source is never bundled with this CLI — it is only ever
reached through its private GitHub repository. The initial clone uses a
temporary GitHub token that is scrubbed again immediately afterwards, and
updates run through a device-bound SSH deploy key.

Usage:
    agent-cli check                 Run code checks (go vet, staticcheck, structure)
    agent-cli install               Launch the installer (Standard/Pro server, desktop, CLI)
    agent-cli self-install          Install the CLI itself as a global program
    agent-cli uninstall             Launch the appropriate uninstaller
    agent-cli start                 Start the vaultagent service (systemd)
    agent-cli stop                  Stop the vaultagent service (systemd)
    agent-cli restart               Restart the vaultagent service (systemd)
    agent-cli status                Show service status
    agent-cli config                Show the current configuration
    agent-cli version               Show CLI version and installed server edition
    agent-cli update                Pull latest code (Standard/Pro aware) and rebuild
    agent-cli ask <prompt>          Ask the running server agent directly
    agent-cli reset [--session ID]  Reset a chat session (Pro)
    agent-cli health                Show remote server health + edition
    agent-cli stats                 Show monitoring stats from :9000
    agent-cli help                  Show this help
"""

import argparse
import getpass
import importlib
import json
import os
import shutil
import subprocess
import sys

VERSION = "0.5 (Standard + Pro)"
REPO_URL = "https://github.com/rhantschk-cmyk/agent-Project"
PRO_REPO_URL = "https://github.com/rhantschk-cmyk/agent-Project-pro"
PRO_REPO_SSH = "git@github.com:rhantschk-cmyk/agent-Project-pro.git"
SERVICE_NAME = "vaultagent"

SERVER_WORK_DIR = "/opt/vaultagent"
SERVER_BINARY = "/usr/local/bin/vaultagent"
SERVER_CONFIG_DIR = "/etc/vaultagent"
EDITION_MARKER = os.path.join(SERVER_CONFIG_DIR, "EDITION")

CLIENT_CONFIG_REL = os.path.join("src", "Client", "config.json")
SERVER_CONFIG_PATHS = [
    os.path.join(SERVER_CONFIG_DIR, "config.json"),
    os.path.join(SERVER_WORK_DIR, "src", "Server", "config.json"),
    os.path.join("src", "Server", "config.json"),
]

STATE_DIR = os.path.join(os.path.expanduser("~"), ".vaultagent")
GLOBAL_CLIENT_CONFIG = os.path.join(STATE_DIR, "config.json")
SESSION_FILE = os.path.join(STATE_DIR, "session.json")


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
# paths & editions
# ---------------------------------------------------------------------------

def load_installer(module_name: str):
    """Import an installer module from the repo tree, the PyInstaller bundle
    or an installed package (pip)."""
    try:
        return importlib.import_module("install." + module_name)
    except ImportError:
        # Source checkout: cli/agent_cli.py lives in <root>/src/cli/.
        sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
        return importlib.import_module("install." + module_name)


def detect_repo_edition(server_dir: str = None) -> str:
    """Detect the edition of a repository checkout from its server sources."""
    server_dir = server_dir or os.path.join(os.getcwd(), "src", "Server")
    # Pro-specific markers only (docs/ exists in both editions).
    pro_markers = ("customtools.go", "chatsession.go", "tools")
    for marker in pro_markers:
        if os.path.exists(os.path.join(server_dir, marker)):
            return "pro"
    return "standard"


def installed_edition() -> str:
    """Edition of the installed server (from marker file or repo layout)."""
    for marker in (EDITION_MARKER, os.path.join(SERVER_WORK_DIR, "EDITION")):
        if os.path.exists(marker):
            try:
                return open(marker, encoding="utf-8").read().strip().lower()
            except OSError:
                return ""
    if os.path.isdir(os.path.join(SERVER_WORK_DIR, "src", "Server")):
        return detect_repo_edition(os.path.join(SERVER_WORK_DIR, "src", "Server"))
    return ""


# ---------------------------------------------------------------------------
# config & state
# ---------------------------------------------------------------------------

def load_client_config() -> dict:
    """Client-side connection config (server_ip / verify_key / port)."""
    for path in (GLOBAL_CLIENT_CONFIG, CLIENT_CONFIG_REL):
        if os.path.exists(path):
            try:
                with open(path, "r", encoding="utf-8") as fh:
                    data = json.load(fh)
                if isinstance(data, dict):
                    return data
            except (OSError, ValueError):
                pass
    return {}


def server_base(cfg: dict, args, default_port: int):
    """Build the base URL + verify key, honouring --server/--port/--key."""
    server = getattr(args, "server", None) or cfg.get("server_ip", "localhost")
    port = getattr(args, "port", None) or cfg.get("port", default_port)
    key = getattr(args, "key", None) or cfg.get("verify_key", "") or ""
    return f"http://{server}:{port}", key


def load_saved_session() -> str:
    try:
        with open(SESSION_FILE, "r", encoding="utf-8") as fh:
            return json.load(fh).get("session_id", "")
    except (OSError, ValueError):
        return ""


def save_session(session_id: str) -> None:
    os.makedirs(STATE_DIR, exist_ok=True)
    with open(SESSION_FILE, "w", encoding="utf-8") as fh:
        json.dump({"session_id": session_id}, fh, indent=2)


def clear_session() -> None:
    if os.path.exists(SESSION_FILE):
        try:
            os.remove(SESSION_FILE)
        except OSError:
            pass


def http_json(url: str, payload: dict = None, timeout: int = 180):
    """GET/POST JSON, returning (status, parsed dict)."""
    import urllib.error
    import urllib.request

    data = None
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST" if data is not None else "GET",
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            body = resp.read().decode("utf-8", "replace")
            try:
                return resp.status, json.loads(body)
            except ValueError:
                return resp.status, {"raw": body}
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", "replace")
        try:
            return e.code, json.loads(body)
        except ValueError:
            return e.code, {"error": body}
    except urllib.error.URLError as e:
        return 0, {"error": str(e.reason)}


# ---------------------------------------------------------------------------
# check
# ---------------------------------------------------------------------------

def check_structure() -> bool:
    edition = detect_repo_edition()
    print(f"[INFO] Detected edition: {edition}")
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
    if edition == "pro":
        required += [
            os.path.join("src", "Server", "chatsession.go"),
            os.path.join("src", "Server", "customtools.go"),
            os.path.join("src", "Server", "tools"),
        ]
    ok = True
    for f in required:
        if os.path.exists(f):
            print(f"[OK]   {f}")
        else:
            print(f"[FAIL] missing: {f}")
            ok = False
    return ok


def cmd_check(args=None) -> int:
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
# install / uninstall
# ---------------------------------------------------------------------------

def _run_installer_module(module_name: str, **kwargs) -> int:
    mod = load_installer(module_name)
    if kwargs:
        mod.main(**kwargs)
    else:
        mod.main()
    return 0


def cmd_install(args=None) -> int:
    print("=== Installer ===")
    print("Which component do you want to install?")
    print("  [1] Server — Standard edition (open source)")
    print("  [2] Server — Pro edition (private repo, temporary key)")
    print("  [3] Desktop client (Windows)")
    print("  [4] CLI tool itself (this program, as a global command)")
    choice = input("Choice [1/2/3/4]: ").strip()
    if choice in ("1", "2"):
        if not require_root():
            return 1
        return _run_installer_module("server_installer", edition="standard" if choice == "1" else "pro")
    if choice == "3":
        return _run_installer_module("desktop_installer")
    if choice == "4":
        if not require_root():
            return 1
        return _run_installer_module("cli_installer")
    print_error("Invalid choice.")
    return 1


def cmd_self_install(args=None) -> int:
    if not require_root():
        return 1
    return _run_installer_module("cli_installer")


def cmd_uninstall(args=None) -> int:
    print("=== Uninstaller ===")
    print("Which component do you want to uninstall?")
    print("  [1] Server (Linux / systemd)")
    print("  [2] Desktop client (Windows)")
    choice = input("Choice [1/2]: ").strip()
    if choice == "1":
        if not require_root():
            return 1
        return _run_installer_module("server_uninstaller")
    if choice == "2":
        return _run_installer_module("desktop_uninstaller")
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
# version / update
# ---------------------------------------------------------------------------

def cmd_version(args=None) -> int:
    banner()
    edition = installed_edition()
    if edition:
        print(f"Installed server edition: {edition}")
        print(f"  work dir : {SERVER_WORK_DIR}")
        print(f"  config   : {SERVER_CONFIG_DIR}")
    else:
        print("Installed server edition: none detected (run 'agent-cli install')")
    print("\nSupported editions:")
    print(f"  Standard — open source ({REPO_URL})")
    print(f"  Pro      — closed source, private repo ({PRO_REPO_URL})")
    return 0


def _remote_url(work: str) -> str:
    try:
        return capture(["git", "-C", work, "remote", "get-url", "origin"]).stdout.strip()
    except Exception:
        return ""


def cmd_update(args=None) -> int:
    if not require_root():
        return 1
    print("=== Update ===")
    work = SERVER_WORK_DIR if os.path.isdir(os.path.join(SERVER_WORK_DIR, ".git")) else os.getcwd()
    if not os.path.isdir(os.path.join(work, ".git")):
        print_error(f"No git repository found at {work}.")
        return 1
    server_dir = os.path.join(work, "src", "Server")
    if not os.path.exists(os.path.join(server_dir, "go.mod")):
        print_error(f"No server source (go.mod) found at {server_dir}.")
        return 1

    edition = installed_edition() or detect_repo_edition(server_dir)
    print(f"[INFO] Edition: {edition}")

    if edition == "pro":
        origin = _remote_url(work)
        print(f"[INFO] Pro remote: {origin or 'unknown'}")
        result = capture(["git", "-C", work, "pull"])
        if result.returncode != 0:
            print("[INFO] Plain pull failed — the private Pro repo needs authentication.")
            if origin.startswith("git@") or "ssh://" in origin:
                print_error("SSH deploy-key remote failed. Check /etc/vaultagent/.ssh.")
                return result.returncode
            token = getpass.getpass("Temporary GitHub token: ").strip()
            if not token:
                print_error("Aborted (no token provided).")
                return 1
            tmp_url = f"https://x-access-token:{token}@github.com/rhantschk-cmyk/agent-Project-pro.git"
            print("[INFO] Pulling with temporary token (not stored)...")
            code = run_cmd(["git", "-C", work, "pull", tmp_url, "HEAD"])
            if code != 0:
                return code
    else:
        if run_cmd(["git", "-C", work, "pull"]) != 0:
            return 1

    print("\nRebuilding server binary...")
    code = run_cmd(["go", "build", "-o", SERVER_BINARY, "."], cwd=server_dir)
    if code != 0:
        return code
    print("Restart the service to apply: agent-cli restart")
    return 0


# ---------------------------------------------------------------------------
# ask / reset / health / stats (query the running agent)
# ---------------------------------------------------------------------------

def cmd_ask(args) -> int:
    cfg = load_client_config()
    base, key = server_base(cfg, args, 8080)
    prompt = " ".join(args.prompt)
    if not prompt:
        print_error("Empty prompt.")
        return 1

    session_id = args.session
    if session_id is None and not args.new:
        session_id = load_saved_session() or None

    payload = {"verify_key": key, "prompt": prompt}
    if session_id:
        payload["session_id"] = session_id

    status, data = http_json(f"{base}/api/agent/ask", payload)
    if status == 401:
        print(f"\n[ERROR] Unauthorized. Check the verify key (--key or config).")
        return 1
    if data.get("error"):
        print(f"\n[ERROR] {data['error']}")
        return 1

    print("\n" + (data.get("response") or "(no response)"))

    new_sid = data.get("session_id")
    if new_sid:
        save_session(new_sid)
        print(f"\n[CLI] Chat session: {new_sid}")
        print('[CLI] Continue: agent-cli ask "..."  |  New chat: agent-cli ask --new "..."')
    elif session_id is None and not args.new:
        print("[CLI] Standard server: stateless chat (no session support).")
    return 0


def cmd_reset(args) -> int:
    cfg = load_client_config()
    base, key = server_base(cfg, args, 8080)
    session_id = args.session or load_saved_session()
    if not session_id:
        print("[WARN] No chat session to reset.")
        return 0

    status, data = http_json(
        f"{base}/api/agent/reset",
        {"verify_key": key, "session_id": session_id},
        timeout=30,
    )
    if status in (404, 405):
        print("[INFO] Standard server: no session management available.")
        return 0
    if status == 401:
        print_error("Unauthorized. Check the verify key (--key or config).")
        return 1
    if data.get("error"):
        print_error(data["error"])
        return 1
    clear_session()
    print(f"[OK] Session {data.get('reset', session_id)} reset.")
    return 0


def cmd_health(args=None) -> int:
    cfg = load_client_config()
    base, _ = server_base(cfg, args, 8080)
    status, data = http_json(f"{base}/health", timeout=30)
    if status != 200 or not data.get("status"):
        print_error(f"Server not reachable at {base}/health.")
        return 1
    version = data.get("version", "?")
    edition = "Pro" if "Pro" in version else "Standard"
    print(f"Status : {data.get('status')}")
    print(f"Version: {version}")
    print(f"Edition: {edition}")
    print(f"Uptime : {data.get('uptime')}")
    return 0


def cmd_stats(args=None) -> int:
    cfg = load_client_config()
    base, _ = server_base(cfg, args, 9000)
    status, data = http_json(f"{base}/api/stats", timeout=30)
    if status != 200:
        print_error(f"Monitoring not reachable at {base}/api/stats.")
        return 1
    print("=== Monitoring ===")
    print(f"CPU   : {data.get('cpu_usage_percent', 0):.1f} %")
    print(f"RAM   : {data.get('ram_used_mb', 0)} MB / {data.get('ram_total_mb', 0)} MB ({data.get('ram_usage_percent', 0):.1f} %)")
    print(f"Disk  : {data.get('disk_free_gb', 0)} GB free / {data.get('disk_total_gb', 0)} GB")
    print(f"Uptime: {data.get('uptime_formated', '?')} ({data.get('uptime_seconds', 0)} s)")
    print(f"GPU   : {data.get('gpu_info', '?')}")
    return 0


# ---------------------------------------------------------------------------
# help
# ---------------------------------------------------------------------------

def usage() -> None:
    banner()
    print("")
    print("USAGE:")
    print("  agent-cli check                Run code checks (go vet, staticcheck, structure)")
    print("  agent-cli install              Launch the installer (Standard/Pro server, desktop, CLI)")
    print("  agent-cli self-install         Install the CLI itself as a global program")
    print("  agent-cli uninstall            Launch the appropriate uninstaller")
    print("  agent-cli start                Start the vaultagent service (systemd)")
    print("  agent-cli stop                 Stop the vaultagent service (systemd)")
    print("  agent-cli restart              Restart the vaultagent service (systemd)")
    print("  agent-cli status               Show service status")
    print("  agent-cli config               Show the current configuration")
    print("  agent-cli version              Show CLI version and installed server edition")
    print("  agent-cli update               Pull latest code (Standard/Pro aware) and rebuild")
    print("  agent-cli ask <prompt>         Ask the server agent directly")
    print("      --session <id>             Use a specific chat session (Pro)")
    print("      --new                      Start a fresh chat session (Pro)")
    print("      --server/--port/--key      Override connection settings")
    print("  agent-cli reset [--session <id>]  Reset a chat session (Pro)")
    print("  agent-cli health               Show server health + edition")
    print("  agent-cli stats                Show monitoring stats from :9000")
    print("  agent-cli help                 Show this help")
    print("")
    print(f"NOTE: Standard v{VERSION.split(' ')[0]} is open source. The Pro edition is a separate,")
    print("      closed-source product distributed via a private repository using a temporary key.")


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
    sub.add_parser("self-install", help="Install the CLI itself as a global program")
    sub.add_parser("uninstall", help="Launch the uninstaller")
    sub.add_parser("start", help="Start the service")
    sub.add_parser("stop", help="Stop the service")
    sub.add_parser("restart", help="Restart the service")
    sub.add_parser("status", help="Show service status")
    sub.add_parser("config", help="Show current config")
    sub.add_parser("version", help="Show CLI version and installed server edition")
    sub.add_parser("update", help="Pull latest code and rebuild")

    ask = sub.add_parser("ask", help="Ask the server agent directly")
    ask.add_argument("prompt", nargs="+", help="The prompt to send to the agent")
    ask.add_argument("--session", help="Use a specific chat session (Pro)")
    ask.add_argument("--new", action="store_true", help="Start a new chat session (Pro)")
    ask.add_argument("--server", help="Server IP/hostname (overrides config)")
    ask.add_argument("--port", type=int, help="Server port (overrides config)")
    ask.add_argument("--key", help="Verify key (overrides config)")

    reset = sub.add_parser("reset", help="Reset a chat session (Pro)")
    reset.add_argument("--session", help="Session id to reset (defaults to the saved one)")
    reset.add_argument("--server", help="Server IP/hostname (overrides config)")
    reset.add_argument("--port", type=int, help="Server port (overrides config)")
    reset.add_argument("--key", help="Verify key (overrides config)")

    for name in ("health", "stats"):
        p = sub.add_parser(name, help=f"Query the {name} endpoint")
        p.add_argument("--server", help="Server IP/hostname (overrides config)")
        p.add_argument("--port", type=int, help="Server port (overrides config)")
        p.add_argument("--key", help="Verify key (overrides config)")

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
        "self-install": cmd_self_install,
        "uninstall": cmd_uninstall,
        "start": lambda a: cmd_service("start"),
        "stop": lambda a: cmd_service("stop"),
        "restart": lambda a: cmd_service("restart"),
        "status": cmd_status,
        "config": cmd_config,
        "version": cmd_version,
        "update": cmd_update,
        "ask": cmd_ask,
        "reset": cmd_reset,
        "health": cmd_health,
        "stats": cmd_stats,
    }
    return handlers[args.command](args)


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("\n\nAborted.")
        sys.exit(130)