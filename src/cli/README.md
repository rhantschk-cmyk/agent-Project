# Agent Software CLI

A server-side command-line tool for the **Agent Software** project, written in **Python**.

## Features

- `check` — verify the repository structure and run Go analysis tools (`go vet`, `staticcheck`).
- `install` — launch the appropriate installer (server or desktop).
- `start` / `stop` / `restart` — control the `email-agent` systemd service.
- `status` — show the service status.
- `config` — display the current `config.json`.
- `update` — pull the latest code and rebuild the binary.
- `ask` — query the running agent directly over the HTTP API (port 8080).

## Usage

```bash
python3 agent-cli.py check
python3 agent-cli.py install
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
