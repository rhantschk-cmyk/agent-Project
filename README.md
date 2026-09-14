# VaultAgent — Autonomous AI Email Agent

An autonomous, event-driven email processing and AI agent system built in **Go** and powered by local **Ollama LLMs**. It monitors incoming emails via **IMAP IDLE**, classifies them with a local AI model, gathers context from a knowledge base and long-term memory, and creates ready-to-send **draft replies** directly in the Gmail mailbox — without any human interaction.

The project is designed around a hotel business: it knows room prices, cancellation policies, and reply templates for wellness and tennis packages.

---

## Editions

| Edition | Server version | Source code | Repository | License |
| :--- | :--- | :--- | :--- | :--- |
| **Standard** | `v0.4` | Open source | `agent-Project` (public) | **MIT** |
| **Pro** | `v0.3` | Closed source | `agent-Project-pro` (private) | Commercial terms |
| **CLI** | `0.5` | Open source | Shipped inside `agent-Project` | **MIT** |

The **CLI tool** is open source and manages **both** editions. It is the recommended way to install and update the server — including the closed-source **Pro** edition.

> The Pro edition is intentionally **closed source**. Its source is never bundled with the Standard repository or the CLI. It is only ever reached through its private GitHub repository.

---

## Overview

The software performs the following autonomous workflow:

1. **Real-time email monitoring** via IMAP IDLE (event-driven, no polling).
2. **AI classification** of every email into `SPAM`, `IMPORTANT`, or `STANDARD`.
3. **Autonomous agent loop** that dynamically calls tools to research and understand each request.
4. **Template-aware replies** using Markdown templates from a knowledge base.
5. **Gmail draft creation** directly in `[Gmail]/Drafts` (subject prefixed with `[ENTWURF]`).
6. **Long-term memory** with automatic compression.
7. **Two network interfaces:** an interactive CLI/chat server and a system monitoring service.
8. **Native desktop client** (Odin / raylib) with agent chat and system monitoring.

### Processing pipeline

```
new email  ->  IMAP IDLE  ->  classify  ->  agent loop (tools)  ->  [Pro: verification]  ->  draft in [Gmail]/Drafts
```

`SPAM` emails are detected during classification and ignored — no draft is created.

---

## Features

### Standard edition (v0.4)

| Feature | Description |
| :--- | :--- |
| Event-driven email monitor | Instant processing of incoming emails via IMAP IDLE |
| AI classification | `SPAM` / `IMPORTANT` / `STANDARD` routing with a fast local model |
| Autonomous agent loop | Up to 30 tool-calling turns to fully research each request |
| 7 built-in tools | Memory, inbox search, conversation history, docs, and reply tools |
| Knowledge base | Markdown templates (pricing, policies, reply templates) |
| Long-term memory | Auto-compressed, persistent fact storage |
| Gmail draft creation | Replies placed in `[Gmail]/Drafts` as `[ENTWURF]` emails |
| CLI / chat server | Secure JSON API on port `8080` |
| Monitoring service | System stats on port `9000` |
| Desktop client | Native GUI (Odin / raylib) with chat + monitoring tabs |
| Single mailbox | One IMAP account per instance |

### Pro edition (v0.3)

The **Pro** edition is a separate, **closed-source** product distributed through a private GitHub repository. It contains everything from the Standard edition and adds:

| Feature | Description |
| :--- | :--- |
| Parallel email monitors | Any number of mailboxes monitored **in parallel** (`accounts[]`), one goroutine and IMAP connection per account |
| Multi-account config | Independent IMAP credentials per account, across different providers |
| Sender blacklisting | Emails from blacklisted addresses are dropped before processing |
| Extensible custom tool API | Customers define their own tools as JSON files (`tools/*.json`), no source access required |
| Custom tool action types | `http` (requests to webhooks/APIs), `script` (external commands), `text` (parameterised templates) |
| Persistent CLI chat sessions | `session_id`-based chat memory (`SessionManager`) with 30 min TTL, max 100 sessions |
| Session reset endpoint | `POST /api/agent/reset` clears a conversation ("new chat") |
| Draft verification | A second verification pass (`sys_promts.verify`) checks each draft; failed drafts are regenerated once |
| Priority support | Commercial support and earlier access to new features |

#### Custom tools (`tools/*.json`)

Custom tools are loaded at runtime from the configured `tools_dir`. Each file declares a name, a description, typed properties, and an action:

| Action type | Behavior | Example |
| :--- | :--- | :--- |
| `http` | Sends an HTTP request (method, URL, headers, body) with `{{param}}` placeholder substitution | `webhook_senden` → POST to a Slack/Teams webhook |
| `script` | Executes an external command with `{{param}}` arguments | `angebot_berechnen` → Python script computing `stunden × satz` |
| `text` | Returns a static, parameterised template string | `antwort_template` → predefined reply snippet |

Example tool definition:

```json
{
  "name": "webhook_senden",
  "description": "Sendet eine Nachricht an einen konfigurierten Webhook (z.B. Slack/Teams)",
  "properties": [
    {
      "name": "nachricht",
      "type": "string",
      "description": "Der Inhalt der zu sendenden Nachricht",
      "required": true
    }
  ],
  "action": {
    "type": "http",
    "method": "POST",
    "url": "https://hooks.example.com/MEIN-WEBHOOK",
    "headers": {
      "Content-Type": "application/json"
    },
    "body": "{\"text\": \"{{nachricht}}\"}"
  }
}
```

#### Chat sessions (Pro)

The chat API on port `8080` accepts a `session_id`. In-coming prompts are appended to the session history, so the agent keeps the full conversation context across requests:

| Endpoint | Method | Purpose |
| :--- | :--- | :--- |
| `POST /api/agent/ask` | POST | Send a prompt; create or continue a session (`session_id`) |
| `POST /api/agent/reset` | POST | Delete a session (start a new chat) |
| `GET /health` | GET | Health, version and uptime |

The Standard edition is **stateless**: every `ask` request is answered without conversation memory, and `reset` is not available.

---

## Installation

The server can be installed three ways:

1. **CLI installer** — recommended; terminal-based, manages Standard and Pro, registers a `systemd` service.
2. **Desktop installer** — Windows, places the native client into `Program Files` and creates a desktop shortcut.
3. **Manual build** — build and run from a source checkout.

### Prerequisites

| Software | Purpose |
| :--- | :--- |
| **Go 1.22+** | Compile the server backend |
| **Ollama** | Local LLM runtime (`ollama serve` must be running) |
| **A pullable model** | e.g. `qwen2.5:14b` |
| **GPU drivers** | NVIDIA CUDA / AMD ROCm (for fast inference) |
| **Python 3** | For the installers and the CLI |
| **Odin compiler** | Only needed to build the desktop client from source |

### Recommended: installation via the CLI

The open-source CLI (`agent-cli`) is the recommended entry point. It performs the system check, clones/pulls the correct repository, builds the binary, writes the config, and registers the `systemd` service.

First, either clone the Standard repository or install the CLI as a standalone program:

```bash
git clone https://github.com/rhantschk-cmyk/agent-Project.git
cd agent-Project
sudo python3 src/cli/agent-cli.py self-install     # global /usr/local/bin/agent-cli
```

Alternative standalone installation methods:

```bash
pip install .                                       # or pipx install .
./compiled/build.sh                                 # single PyInstaller binary (Linux)
```

Then install the server edition of your choice:

```bash
sudo agent-cli install
```

The installer asks for the component:

| Choice | Component |
| :---: | :--- |
| `[1]` | Server — **Standard** edition (open source) |
| `[2]` | Server — **Pro** edition (private repository, temporary key) |
| `[3]` | Desktop client (Windows) |
| `[4]` | CLI tool itself (global command) |

#### Pro installation flow (choice `[2]`)

```
system check  ->  temporary GitHub token  ->  clone private repo  ->  token scrubbed  ->  optional SSH deploy key  ->  build  ->  config  ->  systemd
```

1. The installer checks the system (OS, Go, Ollama, GPU).
2. You provide a **temporary GitHub token** (fine-grained, read-only, repository scope) for the private `agent-Project-pro` repository. It is used **only** for the clone.
3. The token is **scrubbed from the git remote immediately** — the remote is reset to plain HTTPS, so no credential is stored on the machine.
4. Optionally, a **device-bound SSH deploy key** is generated (ed25519). You add the printed **public key** as a read-only deploy key to the private repository. From then on, `agent-cli update` runs over SSH with that key — nothing can be intercepted during an update.
   - If you skip the deploy key, every `agent-cli update` asks for a fresh temporary token.
5. The Go binary is built to `/usr/local/bin/vaultagent`.
6. You answer interactive questions for IMAP accounts, model, CLI secret key, custom tools directory and blacklist (Pro), then the config is written.
7. A `systemd` service `vaultagent` is created, enabled and started.

> The Pro edition (`agent-Project-pro`) is **closed source**. Access requires a temporary GitHub token with access to the private repository. The repository URL must not be shared publicly.

### Desktop installer (Windows)

```bash
cd agent-Project/src/install
python desktop_installer.py
```

The installer:

1. Checks for the Odin compiler and builds the desktop client.
2. Copies the executable and config into `C:\Program Files\VaultAgent\`.
3. Creates a **desktop shortcut**.
4. Configures the `server_ip` and `verify_key` to point at your server.

### Manual build

**Server:**

```bash
cd src/Server
cp config.json.example config.json     # then edit your credentials
go build -o vaultagent .
./vaultagent
```

> The server is a multi-file `package main`. Always build/run the **whole package** with the dot — `go build .` or `go run .`. Running `go run main.go` alone fails with `undefined` errors, because the other files (config, agent, mails, tools, memory, servers) are not included.

**Desktop client (Odin):**

```bash
cd src/Client
odin build . -out:VaultAgent
```

### Uninstall

**Server** (removes the `systemd` service, the binary and `/etc/vaultagent`):

```bash
sudo python3 src/install/server_uninstaller.py
```

**Desktop** (removes the app and the desktop shortcut):

```bash
python src/install/desktop_uninstaller.py
```

Alternatively, use the CLI:

```bash
sudo agent-cli uninstall      # then pick server or desktop
```

---

## Server Services & Ports

| Service | Port | Endpoints | Purpose |
| :--- | :---: | :--- | :--- |
| **CLI / chat server** | `8080` | `POST /api/agent/ask`, `GET /health` | JSON requests routed through the same agent loop used for email processing; authenticated via `cli_secret_key` |
| **Session API (Pro)** | `8080` | `POST /api/agent/reset` | Reset a persistent chat session |
| **Monitoring service** | `9000` | `GET /api/stats` | System metrics: CPU, RAM, disk, uptime, GPU |

---

## Configuration (`config.json`)

The entire system is configured through a central `config.json`. See `config.json.example` for a template with placeholders.

### Standard edition

| Section | Keys | Description |
| :--- | :--- | :--- |
| `e-mail` | `username`, `app_token`, `server`, `draft_folder` | Single IMAP account & draft target folder |
| `program` | `model`, `knowledge_dir`, `cli_secret_key` | LLM model, knowledge base path, API auth key |
| `memory` | `memory_compression_time`, `memory_file`, `memory_compress_promt` | Memory file and auto-compression settings |
| `sys_promts` | `standard`, `important`, `classify`, `cli` | System prompts for each context |

### Pro edition

The Pro config extends the Standard one:

| Section | Pro extra keys | Description |
| :--- | :--- | :--- |
| `e-mail` | `accounts[]` (instead of single account), `blacklisted[]` | Any number of IMAP accounts + blacklisted sender addresses |
| `program` | `tools_dir` | Directory with custom JSON tools |
| `sys_promts` | `verify` | Prompt for the draft-verification pass |

> **Security:** never commit real credentials. The included `config.json.example` uses placeholders — copy it to `config.json` and fill in your own values.

---

## CLI Tool

Located in `src/cli/` and written in **Python**. It is a server-side command-line tool that manages **both** the Standard and the Pro edition. It can be installed globally via `self-install`, `pip`/`pipx`, or as a single PyInstaller binary (`compiled/`).

| Command | Purpose | Pro-only |
| :--- | :--- | :---: |
| `check` | Verify repo structure + run `go vet`, `go test`, `staticcheck`; auto-detects the edition | |
| `install` | Launch the installer (Standard/Pro server, desktop, CLI) | |
| `self-install` | Install the CLI itself as a standalone global program | |
| `uninstall` | Launch the appropriate uninstaller (server or desktop) | |
| `start` / `stop` / `restart` | Control the `vaultagent` systemd service | |
| `status` | Show service status | |
| `config` | Show the current `config.json` | |
| `version` | Show CLI version and the installed server edition | |
| `update` | Pull latest code (Standard/Pro aware) and rebuild the binary | |
| `ask <prompt>` | Query the running agent over port `8080` | |
| `ask --session <id>` | Continue a specific chat session | Yes |
| `ask --new` | Start a fresh chat session | Yes |
| `reset [--session <id>]` | Reset a chat session | Yes |
| `health` | Show remote server health, version and edition | |
| `stats` | Show monitoring stats from port `9000` | |

Connection settings (`server_ip`, `port`, `verify_key`) can be overridden per command with `--server`, `--port` and `--key`. The last `session_id` is stored per user in `~/.vaultagent/session.json`.

```bash
agent-cli version
agent-cli check
sudo agent-cli install          # [1] Standard, [2] Pro, [3] Desktop, [4] CLI
sudo agent-cli start
agent-cli status
agent-cli ask "Wieviel kostet ein Zimmer?"
sudo agent-cli update           # Pro: runs over the device SSH deploy key
```

---

## Project Structure

### Standard edition

```
agent-Project/
├── README.md
├── GUIDE.md
├── LICENSE                       # MIT
├── config.json.example
├── pyproject.toml                # pip/pipx installation of the CLI (agent-cli)
├── compiled/
│   ├── build.sh                  # build Linux CLI executable (PyInstaller)
│   └── build_windows.ps1         # build Windows CLI .exe (PyInstaller)
└── src/
    ├── Server/                   # Go backend (package main)
    │   ├── main.go               # entry point, banner, graceful shutdown
    │   ├── config.go             # config loader
    │   ├── agent.go              # LLM agent loop & tool execution
    │   ├── mails.go              # IMAP handling & draft creation
    │   ├── tools.go              # tool definitions for the LLM
    │   ├── memory.go             # long-term memory + auto-compression
    │   ├── cliserver.go          # HTTP API server (port 8080)
    │   ├── monitoringserver.go   # monitoring service (port 9000)
    │   ├── logging.go            # logging helper (stdout + file)
    │   ├── docs/                 # knowledge base (templates, pricing)
    │   └── *_test.go             # Go unit tests
    ├── Client/
    │   └── main.odin             # desktop GUI (Odin / raylib)
    ├── cli/
    │   ├── agent-cli.py          # shim entry point
    │   ├── agent_cli.py          # CLI implementation
    │   └── README.md
    └── install/
        ├── server_installer.py   # systemd installer (Linux; Standard + Pro)
        ├── desktop_installer.py  # Windows desktop installer
        ├── cli_installer.py      # installs the CLI itself (self-install)
        ├── server_uninstaller.py
        └── desktop_uninstaller.py
```

### Pro edition (additional/overridden files)

```
agent-Project-pro/src/Server/
├── agent.go               # agent loop + session-aware history handling
├── mails.go               # parallel monitors (accounts[]) + blacklist + verification
├── cliserver.go           # session-aware API (+ /api/agent/reset)
├── tools.go               # central ToolRegistry (built-in + custom tools)
├── customtools.go         # JSON-driven custom tool loader (http / script / text)
├── chatsession.go         # SessionManager for persistent chat sessions
├── config.go              # accounts[], blacklisted[], tools_dir, verify prompt
├── tools/                 # example custom tools (webhook_senden, angebot_berechnen, antwort_template)
└── *_test.go              # unit tests incl. custom tools & chat sessions
```

---

## How the Agent Works

### 1. Email Monitoring & Parsing

The backend connects to Gmail via IMAP (`imap.gmail.com:993`) using Gmail App Tokens. Incoming emails are parsed into internal Go structs. The **Pro** edition runs one independent IMAP connection per configured account, in parallel.

### 2. AI Classification

Each email is analyzed by a fast model and sorted into:

- `SPAM` — unwanted emails (ignored, no draft).
- `IMPORTANT` — important or urgent requests.
- `STANDARD` — normal business or customer inquiries.

### 3. Agent Loop & Built-in Tools

When an email needs an answer, the autonomous agent loop starts. The model can dynamically call the following built-in tools:

| Tool | Parameters | Description |
| :--- | :--- | :--- |
| `read_memory` | *none* | Reads saved facts, agreements & notes from `memory.txt`. |
| `write_memory` | `fact` (string) | Stores new important facts/prices permanently. |
| `search_inbox` | `query` (string) | Searches the inbox for earlier messages on a topic. |
| `get_conversation_history` | `sender_email` (string), `count` (int) | Fetches the last N emails from a sender. |
| `list_all_docs` | *none* | Lists all available templates/documents in the `docs/` folder. |
| `search_docs` | `doc_name` (string) | Reads the exact content of a specific Markdown template. |
| `finish_and_reply` | `response` (string), `notes` (string) | Finishes the research and hands over the final email text. |

The loop is limited to **30 turns** as a safety measure against infinite tool loops. In the **Pro** edition, the built-in tools are extended by all custom tools from `tools_dir` at runtime.

### 4. Verification (Pro)

After a draft is generated, the **Pro** edition runs a second verification pass: a model call with the `verify` system prompt checks whether the email is a sensible answer and whether templates were used correctly, answering only **JA** or **NEIN**.

| Verification result | Behavior |
| :--- | :--- |
| `JA` | Draft is kept. |
| `NEIN` | Email is generated once more (second attempt, no re-verification). |
| other output | Hazy model output — email is kept without verification. |

### 5. Template Processing & Draft Creation

When the agent finds a matching template via `search_docs`, it adopts the wording and only adjusts variable data (names, dates, prices). After calling `finish_and_reply`, the text is converted into a Gmail-compatible MIME structure and placed in `[Gmail]/Drafts` with the subject `[ENTWURF] ...`.

---

## Disclaimer / Liability

THIS SOFTWARE IS PROVIDED "AS IS" FOR PRIVATE/SELF-RESPONSIBLE USE. The developer assumes no liability for server security, data loss, incorrectly generated email drafts, outages, or hardware damage caused by high system load or insufficient cooling. The security of your server is 100% your responsibility.

---

## License

- The **Standard edition** (this repository) is licensed under the **MIT License** — see the `LICENSE` file for details.
- The **CLI tool** is part of the Standard repository and is likewise MIT-licensed.
- The **Pro edition** is a separate product and is **not** covered by the MIT license; it is distributed under its own commercial terms. Contact the developer for Pro licensing.

---

## Contact

- Repository: [github.com/rhantschk-cmyk/agent-Project](https://github.com/rhantschk-cmyk/agent-Project)
- Pro licensing & support: r.hantschk@gmail.com
- Issues & feature requests: open an issue on GitHub