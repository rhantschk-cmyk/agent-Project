#!/usr/bin/env python3
"""
VaultAgent — Server Installer (Linux / systemd) — Standard & Pro

A terminal-based installer that:
  1. Checks the system (OS, Go, Ollama, GPU).
  2. Clones / pulls the repository — Standard (public, open source) or
     Pro (private, closed source).
  3. For the private Pro repository a *temporary* GitHub token is used for
     the clone only and is scrubbed from the git remote immediately after.
  4. Optionally sets up a device-bound SSH deploy key, so that later
     updates run via SSH without any temporary key being re-used or
     intercepted.
  5. Builds the Go binary.
  6. Asks interactively for config and writes config.json.
  7. Creates and enables a systemd service (vaultagent).
  8. Starts the service automatically.

Run with sudo:
    sudo python3 server_installer.py
"""

import getpass
import json
import os
import platform
import shutil
import subprocess
import sys

VERSION = "v0.5 (Standard + Pro)"

REPO_URLS = {
    "standard": "https://github.com/rhantschk-cmyk/agent-Project.git",
    "pro": "https://github.com/rhantschk-cmyk/agent-Project-pro.git",
}
PRO_REPO_HTTPS = "https://github.com/rhantschk-cmyk/agent-Project-pro"
PRO_REPO_SSH = "git@github.com:rhantschk-cmyk/agent-Project-pro.git"

SERVICE_NAME = "vaultagent"
BINARY_PATH = "/usr/local/bin/vaultagent"
CONFIG_DIR = "/etc/vaultagent"
CONFIG_PATH = os.path.join(CONFIG_DIR, "config.json")
EDITION_MARKER = os.path.join(CONFIG_DIR, "EDITION")
WORK_DIR = "/opt/vaultagent"
SSH_KEY_DIR = os.path.join(CONFIG_DIR, ".ssh")
SSH_KEY_PATH = os.path.join(SSH_KEY_DIR, "pro_deploy_key")


def banner() -> None:
    print("==============================================")
    print("  VaultAgent Server Installer")
    print(f"  Version {VERSION}")
    print("  Standard (open source) + Pro (private repo)")
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


def run(cmd: list, check: bool = True) -> subprocess.CompletedProcess:
    result = subprocess.run(cmd, capture_output=True, text=True)
    if check and result.returncode != 0:
        error(f"Command failed: {' '.join(cmd)}\n{result.stderr}")
    return result


def is_root() -> bool:
    return os.geteuid() == 0 if hasattr(os, "geteuid") else False


def confirm(prompt: str, default: bool = False) -> bool:
    suffix = "(y/N)" if not default else "(Y/n)"
    answer = input(f"{prompt} {suffix}: ").strip().lower()
    if not answer:
        return default
    return answer in ("y", "yes")


def check_os() -> None:
    info("Checking operating system...")
    distro = platform.system()
    if distro != "Linux":
        error(f"This installer is for Linux only (detected: {distro}).")
    ok(f"OS: Linux ({platform.dist()[0] if hasattr(platform, 'dist') else 'unknown'})")


def check_go() -> None:
    info("Checking Go toolchain...")
    go = shutil.which("go")
    if not go:
        error("Go is not installed. Install Go 1.22+ first.")
    result = run([go, "version"], check=False)
    if result.returncode != 0:
        error("Could not run 'go version'.")
    version_line = result.stdout.strip().split()
    if len(version_line) < 3:
        error("Could not parse Go version.")
    version = version_line[2].lstrip("go")
    ok(f"Go {version} found at {go}")


def check_ollama() -> None:
    info("Checking Ollama...")
    ollama = shutil.which("ollama")
    if not ollama:
        warn("Ollama is not in PATH. Model inference will fail if it is not running.")
        return
    result = run([ollama, "list"], check=False)
    if result.returncode != 0:
        warn("Ollama is installed but not running / responding. Run 'ollama serve'.")
        return
    ok("Ollama is reachable")


def check_gpu() -> None:
    info("Checking GPU...")
    nvidia_smi = shutil.which("nvidia-smi")
    if nvidia_smi:
        result = run([nvidia_smi, "--query-gpu=name", "--format=csv,noheader"], check=False)
        if result.returncode == 0:
            ok(f"GPU: {result.stdout.strip()}")
            return
    rocm_smi = shutil.which("rocm-smi")
    if rocm_smi:
        ok("AMD ROCm GPU detected")
        return
    warn("No GPU detected. LLM inference may be slow (CPU only).")


def ask_edition() -> str:
    print("\nWhich edition do you want to install?")
    print("  [1] Standard — open source (public repository)")
    print("  [2] Pro      — closed source (private repository, temporary key required)")
    choice = input("Choice [1/2]: ").strip()
    if choice == "2":
        return "pro"
    if choice == "1":
        return "standard"
    error("Invalid choice.")
    return ""


def check_credentials() -> tuple:
    """Ask for Gmail credentials securely and return them."""
    while True:
        print("\n--- Gmail / IMAP account ---")
        username = input("Gmail address (e.g. user@gmail.com): ").strip()
        if not username:
            print("Username cannot be empty.")
            continue
        app_token = getpass.getpass("Gmail App Token: ").strip()
        if not app_token:
            print("App token cannot be empty.")
            continue
        confirm_ = input(f"Is '{username}' correct? [y/N]: ").strip().lower()
        if confirm_ == "y":
            return username, app_token


def ask_temp_token() -> str:
    print("\n--- Pro edition: private repository ---")
    print("The Pro repository (agent-Project-pro) is NOT open source.")
    print("Provide a temporary GitHub token (fine-grained, read-only, repo scope).")
    print("It is used for the clone only and removed again immediately afterwards.")
    while True:
        token = getpass.getpass("Temporary GitHub token: ").strip()
        if token:
            return token
        print("Token cannot be empty.")


def make_token_url(base_url: str, token: str) -> str:
    """https://github.com/... -> https://x-access-token:<TOKEN>@github.com/..."""
    return base_url.replace("https://", f"https://x-access-token:{token}@", 1)


def remote_url(work: str) -> str:
    result = run(["git", "-C", work, "remote", "get-url", "origin"], check=False)
    return result.stdout.strip()


def scrub_pro_remote() -> None:
    info("Scrubbing temporary token from git remote...")
    run(["git", "-C", WORK_DIR, "remote", "set-url", "origin", PRO_REPO_HTTPS])
    ok("Remote is now plain HTTPS (no credentials stored).")


def setup_pro_deploy_key() -> None:
    print("\n--- Pro edition: device verification for updates ---")
    print("To run updates WITHOUT re-entering a temporary key, this device")
    print("gets its own read-only SSH deploy key. The private key never leaves")
    print("this machine, so no key can be intercepted during a later update.")
    if not confirm("Set up an SSH deploy key now? (recommended)", default=True):
        warn("Updates will ask for a fresh temporary token each time.")
        return

    os.makedirs(SSH_KEY_DIR, mode=0o700, exist_ok=True)
    if not os.path.exists(SSH_KEY_PATH):
        info("Generating device deploy key...")
        run(["ssh-keygen", "-t", "ed25519", "-N", "", "-C", "vaultagent-pro-device", "-f", SSH_KEY_PATH])
    os.chmod(SSH_KEY_PATH, 0o600)
    pub_path = SSH_KEY_PATH + ".pub"
    try:
        pub = open(pub_path, encoding="utf-8").read().strip()
    except OSError:
        warn("Could not read the public key.")
        return

    print("\nAdd this PUBLIC key as a read-only deploy key for the Pro repo:")
    print(f"  Repository : {PRO_REPO_SSH}")
    print(f"  GitHub     : Repository -> Settings -> Deploy keys -> Add deploy key")
    print(f"  Allow write: NO (read-only)")
    print(f"  Public key:\n  {pub}")

    if confirm("Have you added the deploy key to GitHub? [y/N]", default=False):
        info("Switching Pro remote to SSH...")
        run(["git", "-C", WORK_DIR, "remote", "set-url", "origin", PRO_REPO_SSH])
        run(["git", "-C", WORK_DIR, "config", "core.sshCommand",
             f"ssh -i {SSH_KEY_PATH} -o IdentitiesOnly=yes"])
        fetch = run(["git", "-C", WORK_DIR, "fetch"], check=False)
        if fetch.returncode == 0:
            ok("Device deploy key configured. Updates run via SSH without any token.")
        else:
            warn("SSH fetch failed — check that the deploy key was added and is read-only.")
            info("Keeping plain-HTTPS remote; updates will need a temporary token.")
            run(["git", "-C", WORK_DIR, "remote", "set-url", "origin", PRO_REPO_HTTPS])
            run(["git", "-C", WORK_DIR, "config", "--unset", "core.sshCommand"], check=False)
    else:
        info("Keeping plain-HTTPS remote; updates will need a new temporary token.")


def clone_or_pull(edition: str, token: str = None) -> str:
    if edition not in REPO_URLS:
        error(f"Unknown edition {edition!r}.")
    info(f"Preparing working directory {WORK_DIR}...")

    if os.path.isdir(os.path.join(WORK_DIR, ".git")):
        info(f"{edition.capitalize()} repository exists, pulling latest...")
        if edition == "pro" and not remote_url(WORK_DIR).startswith("git@"):
            info("Pro remote is plain HTTPS; pull may need a token.")
            setup_pro_deploy_key()
        result = run(["git", "-C", WORK_DIR, "pull"], check=False)
        if result.returncode != 0:
            error(f"Pull failed:\n{result.stderr}")
    elif os.path.isdir(WORK_DIR):
        warn("Directory exists but is not a git repo. Using it as-is.")
    else:
        if edition == "pro":
            token = token or ask_temp_token()
            clone_url = make_token_url(REPO_URLS["pro"], token)
            info("Cloning private Pro repository with temporary token (not stored)...")
        else:
            clone_url = REPO_URLS["standard"]
            info(f"Cloning public repository {REPO_URLS['standard']}...")

        result = run(["git", "clone", clone_url, WORK_DIR], check=False)
        if result.returncode != 0:
            error(f"Clone failed:\n{result.stderr}\n(Pro repo is private — the temporary token must have read access.)")
        if edition == "pro":
            scrub_pro_remote()
            setup_pro_deploy_key()

    return os.path.join(WORK_DIR, "src", "Server")


def build_binary(server_dir: str) -> None:
    info("Building Go binary...")
    env = os.environ.copy()
    build_cmd = ["go", "build", "-o", BINARY_PATH, "."]
    result = subprocess.run(build_cmd, cwd=server_dir, env=env, capture_output=True, text=True)
    if result.returncode != 0:
        error(f"Build failed:\n{result.stdout}\n{result.stderr}")
    ok(f"Binary installed to {BINARY_PATH}")


def ask_config(edition: str) -> dict:
    print("\n--- Configuration ---")
    model = input("Ollama model name [qwen2.5:14b]: ").strip() or "qwen2.5:14b"
    secret = getpass.getpass("CLI secret key (for port 8080 API): ").strip() or "changeme"
    knowledge_dir = input("Knowledge directory relative to server [docs]: ").strip() or "docs"

    accounts = []
    print("\n--- Gmail / IMAP accounts ---")
    while True:
        username, app_token = check_credentials()
        accounts.append({
            "username": username,
            "app_token": app_token,
            "server": "imap.gmail.com:993",
            "draft_folder": "[Gmail]/Drafts",
        })
        if edition == "pro":
            more = input("Add another email account? [y/N]: ").strip().lower()
            if more != "y":
                break
        else:
            break

    blacklisted = []
    if edition == "pro":
        raw = input("Blacklisted sender addresses (comma separated, optional): ").strip()
        blacklisted = [x.strip() for x in raw.split(",") if x.strip()]

    memory = {
        "memory_compression_time": 5,
        "memory_file": "memory.txt",
        "memory_compress_promt": (
            "Fasse das folgende Langzeitgedächtnis zusammen. Entferne Duplikate, "
            "veraltete Angaben und behalte nur wichtige Fakten über Personen, "
            "Stundensätze, Preise und Projektvereinbarungen stichpunktartig bei:"
        ),
    }
    sys_promts = {
        "standard": "",
        "important": "",
        "classify": (
            "Du bist ein Email Klassifizierer und darfst nur in einem Wort antworten. "
            "SPAM für spam Emails, IMPORTANT für wichtige emails, STANDARD für die, "
            "die weder noch sind. WICHTIG: Antworte nur in einem Wort"
        ),
        "cli": "",
    }

    if edition == "pro":
        tools_dir = input("Custom tools directory relative to server [tools]: ").strip() or "tools"
        email = {"accounts": accounts, "blacklisted": blacklisted}
        program = {
            "model": model,
            "knowledge_dir": knowledge_dir,
            "cli_secret_key": secret,
            "tools_dir": tools_dir,
        }
    else:
        email = accounts[0]
        program = {
            "model": model,
            "knowledge_dir": knowledge_dir,
            "cli_secret_key": secret,
        }

    return {
        "e-mail": email,
        "program": program,
        "memory": memory,
        "sys_promts": sys_promts,
    }


def write_config(config: dict) -> None:
    info(f"Writing config to {CONFIG_PATH}...")
    os.makedirs(CONFIG_DIR, exist_ok=True)
    with open(CONFIG_PATH, "w", encoding="utf-8") as fh:
        json.dump(config, fh, indent=2, ensure_ascii=False)
    ok("Config written.")


def write_edition_marker(edition: str) -> None:
    with open(EDITION_MARKER, "w", encoding="utf-8") as fh:
        fh.write(edition.strip().lower() + "\n")
    ok(f"Edition marker: {EDITION_MARKER} ({edition})")


def copy_docs(server_dir: str, edition: str) -> None:
    docs_src = os.path.join(server_dir, "docs")
    if os.path.isdir(docs_src):
        docs_dst = os.path.join(CONFIG_DIR, "docs")
        info(f"Copying knowledge base to {docs_dst}...")
        run(["cp", "-r", docs_src, docs_dst])
        ok("Knowledge base copied.")
    if edition == "pro":
        tools_src = os.path.join(server_dir, "tools")
        if os.path.isdir(tools_src):
            tools_dst = os.path.join(CONFIG_DIR, "tools")
            info(f"Copying custom tools to {tools_dst}...")
            run(["cp", "-r", tools_src, tools_dst])
            ok("Custom tools copied.")


def create_systemd_unit() -> str:
    info("Creating systemd unit...")
    unit_content = f"""[Unit]
Description=VaultAgent - Autonomous AI Email Agent
After=network-online.target ollama.service
Wants=network-online.target

[Service]
Type=simple
ExecStart={BINARY_PATH}
WorkingDirectory={CONFIG_DIR}
Restart=always
RestartSec=5
Environment=HOME={CONFIG_DIR}

[Install]
WantedBy=multi-user.target
"""
    unit_path = f"/etc/systemd/system/{SERVICE_NAME}.service"
    temp_path = "/tmp/vaultagent.service"
    with open(temp_path, "w", encoding="utf-8") as fh:
        fh.write(unit_content)
    run(["cp", temp_path, unit_path])
    os.remove(temp_path)
    ok(f"systemd unit created at {unit_path}")
    return unit_path


def enable_and_start() -> None:
    info("Reloading systemd and enabling service...")
    run(["systemctl", "daemon-reload"])
    run(["systemctl", "enable", SERVICE_NAME])
    run(["systemctl", "start", SERVICE_NAME])
    ok(f"Service '{SERVICE_NAME}' enabled and started.")


def final_message(edition: str) -> None:
    print("\n==============================================")
    print("  Installation complete!")
    print(f"  Edition: {edition.capitalize()}")
    print(f"  Version: {VERSION}")
    print(f"  Service: {SERVICE_NAME}")
    print(f"  Config:  {CONFIG_PATH}")
    print("\n  Useful commands:")
    print(f"    systemctl status {SERVICE_NAME}")
    print(f"    journalctl -u {SERVICE_NAME} -f")
    if edition == "pro":
        print("\n  Updates: agent-cli update")
        print("  (Pro runs via the device SSH deploy key — no temporary key needed)")
    print("==============================================")


def main(edition: str = None, token: str = None) -> None:
    banner()
    if not is_root():
        error("Please run this installer as root / with sudo.")
    if edition is None:
        edition = ask_edition()
    if edition not in REPO_URLS:
        error(f"Unknown edition {edition!r}.")
    info(f"Edition: {edition}")

    check_os()
    check_go()
    check_ollama()
    check_gpu()

    server_dir = clone_or_pull(edition, token)
    build_binary(server_dir)

    config = ask_config(edition)
    write_config(config)
    write_edition_marker(edition)
    copy_docs(server_dir, edition)

    create_systemd_unit()
    enable_and_start()
    final_message(edition)


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nInstallation cancelled by user.")
        sys.exit(130)