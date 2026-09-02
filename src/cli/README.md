# VaultAgent CLI

A server-side command-line tool for the **VaultAgent** project, written in **Python**.

## Features

- `check` — verify the repository structure and run Go analysis tools (`go vet`, `go test`, `staticcheck`).
- `install` — launch the appropriate installer (server or desktop).
- `uninstall` — launch the appropriate uninstaller (server or desktop).
- `start` / `stop` / `restart` — control the `vaultagent` systemd service.
- `status` — show the service status.
- `config` — display the current `config.json`.
- `update` — pull the latest code and rebuild the binary.
- `ask` — query the running agent directly over the HTTP API (port 8080).

## Usage

```bash
python3 agent-cli.py check
python3 agent-cli.py install
sudo python3 agent-cli.py uninstall
sudo python3 agent-cli.py start
sudo python3 agent-cli.py status
python3 agent-cli.py config
python3 agent-cli.py update
python3 agent-cli.py ask "What are your hotel room prices?"
python3 agent-cli.py help
```

## Optional: install as a global command

```bash
sudo cp agent-cli.py /usr/local/bin/agent-cli
sudo chmod +x /usr/local/bin/agent-cli
# then simply:
agent-cli status
```

## Pre-compiled binaries

The CLI can be bundled together with the installer modules into a single
self-contained executable (no Python required on the target machine):

- `compiled/build.sh` — Linux ELF binary (`compiled/VaultAgent`).
- `compiled/build_windows.ps1` — Windows `.exe` (`compiled/VaultAgent.exe`).

```bash
./compiled/build.sh           # Linux
./compiled/build.sh windows   # Windows (requires wine) — or run the .ps1 on Windows
```

Both scripts use [PyInstaller](https://pyinstaller.org/) (`python -m pip install pyinstaller`).
