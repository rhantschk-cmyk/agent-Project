# VaultAgent CLI

A server-side command-line tool for the **VaultAgent** project, written in **Python**.
It manages both the **Standard** (open source) and the **Pro** (closed source, private
repository) edition of the server.

## Features

- `check` — verify the repository structure and run Go analysis tools (`go vet`, `go test`, `staticcheck`). Detects the edition automatically.
- `install` — launch the installer: **Standard server**, **Pro server** (private repo), **desktop client** or the **CLI itself**.
- `self-install` — install the CLI itself as a standalone global program.
- `uninstall` — launch the appropriate uninstaller (server or desktop).
- `start` / `stop` / `restart` — control the `vaultagent` systemd service.
- `status` — show the service status.
- `config` — display the current `config.json`.
- `version` — show the CLI version and the installed server edition.
- `update` — pull the latest code (Standard/Pro aware) and rebuild the binary.
- `ask <prompt>` — query the running agent over the HTTP API (port 8080).
  - `--session <id>` — continue a specific chat session (Pro).
  - `--new` — start a fresh chat session (Pro). The last `session_id` is stored per user in `~/.vaultagent/session.json`.
- `reset [--session <id>]` — reset a chat session (Pro only).
- `health` — show the remote server health, version and edition.
- `stats` — show the monitoring stats from port 9000.
- `help` — show this help.

`--server <ip>`, `--port <port>` and `--key <verify_key>` override the connection
settings for `ask`, `reset`, `health` and `stats`.

## Usage (from the repository)

```bash
python3 src/cli/agent-cli.py version
python3 src/cli/agent-cli.py check
sudo python3 src/cli/agent-cli.py install
sudo python3 src/cli/agent-cli.py uninstall
sudo python3 src/cli/agent-cli.py start
sudo python3 src/cli/agent-cli.py status
python3 src/cli/agent-cli.py config
sudo python3 src/cli/agent-cli.py update
python3 src/cli/agent-cli.py ask "What are your hotel room prices?"
python3 src/cli/agent-cli.py reset
python3 src/cli/agent-cli.py help
```

## Install as a standalone program

Three options:

### 1. Self-install (`agent-cli self-install`)

```bash
sudo python3 src/cli/agent-cli.py self-install
```

Copies the `cli` + `install` packages to `/usr/local/lib/vaultagent-cli/` and creates a
`/usr/local/bin/agent-cli` launcher. The CLI then works from any directory:

```bash
agent-cli version
```

### 2. pip / pipx

```bash
pip install .
# or
pipx install .
agent-cli version
```

### 3. Pre-compiled binaries

```bash
./compiled/build.sh           # Linux
./compiled/build.sh windows   # Windows (requires wine) — or run build_windows.ps1 on Windows
sudo compiled/VaultAgent self-install
```

Both scripts use [PyInstaller](https://pyinstaller.org/) (`python -m pip install pyinstaller`)
and the result is a single self-contained `VaultAgent` executable.

## Pro edition (private repository)

The **Pro** edition is **closed source**. It is installed through its private GitHub
repository (`https://github.com/rhantschk-cmyk/agent-Project-pro`):

1. `agent-cli install` → choose `[2] Server — Pro edition`.
2. Enter a **temporary GitHub token** (fine-grained, read-only, repo scope). It is used
   for the clone only and scrubbed from the git remote immediately afterwards.
3. The installer offers to set up a **device-bound SSH deploy key**. Add the printed
   public key as a read-only deploy key on the private repository. From then on,
   `agent-cli update` runs over SSH with that key — no token is ever re-entered or
   stored on the machine, so nothing can be intercepted during an update.

If you skip the deploy key, `update` will ask you for a new temporary token each time.