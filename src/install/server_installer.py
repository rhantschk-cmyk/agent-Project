#!/usr/bin/env python3
"""
VaultAgent — Server Installer (Linux / systemd)

A terminal-based installer that:
  1. Checks the system (OS, Go, Ollama, GPU).
  2. Clones / pulls the repository.
  3. Builds the Go binary.
  4. Asks interactively for config and writes config.json.
  5. Creates and enables a systemd service (vaultagent).
  6. Starts the service automatically.

Run with sudo:
    sudo python3 server_installer.py
"""

import os
import sys
import shutil
import json
import subprocess
import platform
import getpass

VERSION = "v0.4 (Standard)"
REPO_URL = "https://github.com/rhantschk-cmyk/agent-Project.git"
SERVICE_NAME = "vaultagent"
BINARY_PATH = "/usr/local/bin/vaultagent"
CONFIG_DIR = "/etc/vaultagent"
CONFIG_PATH = os.path.join(CONFIG_DIR, "config.json")
WORK_DIR = "/opt/vaultagent"


def banner() -> None:
    print("==============================================")
    print("  VaultAgent Server Installer")
    print(f"  Version {VERSION}")
    print(f"  {REPO_URL}")
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


def check_credentials() -> str:
    """Ask for Gmail credentials securely and return them."""
    while True:
        print("\n--- Gmail / IMAP setup ---")
        username = input("Gmail address (e.g. user@gmail.com): ").strip()
        if not username:
            print("Username cannot be empty.")
            continue
        app_token = getpass.getpass("Gmail App Token: ").strip()
        if not app_token:
            print("App token cannot be empty.")
            continue
        confirm = input(f"Is '{username}' correct? [y/N]: ").strip().lower()
        if confirm == "y":
            return username, app_token


def ask_config() -> dict:
    print("\n--- Configuration ---")
    model = input("Ollama model name [qwen2.5:14b]: ").strip() or "qwen2.5:14b"
    secret = getpass.getpass("CLI secret key (for port 8080 API): ").strip() or "changeme"
    knowledge_dir = input("Knowledge directory relative to server [docs]: ").strip() or "docs"

    username, app_token = check_credentials()

    config = {
        "e-mail": {
            "username": username,
            "app_token": app_token,
            "server": "imap.gmail.com:993",
            "draft_folder": "[Gmail]/Drafts",
        },
        "program": {
            "model": model,
            "knowledge_dir": knowledge_dir,
            "cli_secret_key": secret,
        },
        "memory": {
            "memory_compression_time": 5,
            "memory_file": "memory.txt",
            "memory_compress_promt": (
                "Fasse das folgende Langzeitgedächtnis zusammen. Entferne Duplikate, "
                "veraltete Angaben und behalte nur wichtige Fakten über Personen, "
                "Stundensätze, Preise und Projektvereinbarungen stichpunktartig bei:"
            ),
        },
        "sys_promts": {
            "standard": "",
            "important": "",
            "classify": (
                "Du bist ein Email Klassifizierer und darfst nur in einem Wort antworten. "
                "SPAM für spam Emails, IMPORTANT für wichtige emails, STANDARD für die, "
                "die weder noch sind. WICHTIG: Antworte nur in einem Wort"
            ),
            "cli": "",
        },
    }
    return config


def clone_or_pull() -> str:
    info(f"Preparing working directory {WORK_DIR}...")
    if os.path.isdir(os.path.join(WORK_DIR, ".git")):
        info("Repository exists, pulling latest...")
        run(["git", "-C", WORK_DIR, "pull"])
    elif os.path.isdir(WORK_DIR):
        warn("Directory exists but is not a git repo. Using it as-is.")
    else:
        info(f"Cloning repository from {REPO_URL}...")
        run(["git", "clone", REPO_URL, WORK_DIR])
    return os.path.join(WORK_DIR, "src", "Server")


def build_binary(server_dir: str) -> None:
    info("Building Go binary...")
    env = os.environ.copy()
    build_cmd = ["go", "build", "-o", BINARY_PATH, "."]
    result = subprocess.run(build_cmd, cwd=server_dir, env=env, capture_output=True, text=True)
    if result.returncode != 0:
        error(f"Build failed:\n{result.stdout}\n{result.stderr}")
    ok(f"Binary installed to {BINARY_PATH}")


def write_config(config: dict) -> None:
    info(f"Writing config to {CONFIG_PATH}...")
    os.makedirs(CONFIG_DIR, exist_ok=True)
    with open(CONFIG_PATH, "w", encoding="utf-8") as fh:
        json.dump(config, fh, indent=2, ensure_ascii=False)
    ok("Config written.")


def copy_docs(server_dir: str) -> None:
    docs_src = os.path.join(server_dir, "docs")
    if os.path.isdir(docs_src):
        docs_dst = os.path.join(CONFIG_DIR, "docs")
        info(f"Copying knowledge base to {docs_dst}...")
        run(["cp", "-r", docs_src, docs_dst])
        ok("Knowledge base copied.")


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


def final_message() -> None:
    print("\n==============================================")
    print("  Installation complete!")
    print(f"  Version: {VERSION}")
    print(f"  Service: {SERVICE_NAME}")
    print(f"  Config:  {CONFIG_PATH}")
    print("\n  Useful commands:")
    print(f"    systemctl status {SERVICE_NAME}")
    print(f"    journalctl -u {SERVICE_NAME} -f")
    print("\n  Pro version (coming soon): unlimited monitors.")
    print("==============================================")


def main() -> None:
    banner()
    if not is_root():
        error("Please run this installer as root / with sudo.")
    check_os()
    check_go()
    check_ollama()
    check_gpu()

    print("\nDo you want to review/edit the current systemd unit after install?")
    server_dir = clone_or_pull()
    build_binary(server_dir)

    config = ask_config()
    write_config(config)
    copy_docs(server_dir)

    unit_path = create_systemd_unit()
    enable_and_start()
    final_message()


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nInstallation cancelled by user.")
        sys.exit(130)
