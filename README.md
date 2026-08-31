# 🛡️ VaultAgent — Autonomous AI Email Agent

An extremely fast, autonomous, event-driven email processing and AI agent system built in **Go** and powered by local **Ollama LLMs**.

> **Current version:** `v0.4 (Standard)`  
> **Pro version:** coming soon — unlimited parallel email monitors, fully configurable in `config.json`.

---

## 📌 Overview

This software continuously monitors incoming emails via **IMAP IDLE**, classifies them automatically using a local AI model, gathers context from a knowledge base and long-term memory, and creates ready-to-send **draft replies** directly in the Gmail mailbox — all autonomously.

The system is designed for a hotel business and knows room prices, cancellation policies, and reply templates for wellness and tennis packages.

### What it does at a glance

1. **Real-time email monitoring** via IMAP IDLE (event-driven, no polling).
2. **AI classification** of every email into `SPAM`, `IMPORTANT`, or `STANDARD`.
3. **Autonomous agent loop** that dynamically calls tools to research and understand each request.
4. **Template-aware replies** using Markdown templates from a knowledge base.
5. **Gmail draft creation** directly in `[Gmail]/Drafts` (subject prefixed with `[ENTWURF]`).
6. **Long-term memory** with automatic compression.
7. **Two network interfaces:** an interactive CLI/chat server and a system monitoring service.
8. **Native desktop client** (Odin / raylib) with agent chat and system monitoring.

---

## ✨ Features

### Standard (v0.4)

| Feature | Description |
| :--- | :--- |
| **Event-driven email monitor** | Instant processing of incoming emails via IMAP IDLE |
| **AI classification** | `SPAM` / `IMPORTANT` / `STANDARD` routing with a fast local model |
| **Autonomous agent loop** | Up to 30 tool-calling turns to fully research each request |
| **7 built-in tools** | Memory, inbox search, conversation history, docs, and reply tools |
| **Knowledge base** | Markdown templates (pricing, policies, reply templates) |
| **Long-term memory** | Auto-compressed, persistent fact storage |
| **Gmail draft creation** | Replies placed in `[Gmail]/Drafts` as `[ENTWURF]` emails |
| **CLI / chat server** | Secure JSON API on port `8080` |
| **Monitoring service** | System stats on port `9000` |
| **Desktop client** | Native GUI (Odin / raylib) with chat + monitoring tabs |

### Pro (coming soon)

The **Pro version** takes the agent to the next level:

- **Unlimited parallel email monitors** — instead of a single IMAP thread, the Pro version can monitor **any number of mailboxes in parallel**. The number of concurrent monitors is fully configurable via `config.json`.
- **Extended tool API** for deeper integration and custom automations.
- **Multi-account management** across different providers.
- **Priority support** and earlier access to new features.

> 👉 The Standard version is deliberately limited to **one email monitor**. If you need more, either build the Pro version yourself or contact the developer.

---

## 📥 Installation

There are three ways to install the software:

1. **Server installer (recommended for Linux servers)** — a terminal-based Python installer that builds the binary and registers it as a `systemd` service.
2. **Desktop installer (Windows)** — a terminal-based Python installer that places the desktop client into `Program Files` and creates a desktop shortcut.
3. **Manual build** — follow the steps below.

### Prerequisites

Before you begin, make sure the following are installed on your system:

| Software | Purpose |
| :--- | :--- |
| **Go 1.22+** | Compile the server backend |
| **Ollama** | Local LLM runtime (`ollama serve` must be running) |
| **A pullable model** | e.g. `qwen2.5:14b` |
| **GPU drivers** | NVIDIA CUDA / AMD ROCm (for fast inference) |
| **Python 3** | For the installers |
| **Odin compiler** | Only needed if you build the desktop client from source |

### Option A — Server Installer (Linux)

```bash
git clone https://github.com/rhantschk-cmyk/agent-Project.git
cd agent-Project/src/install
sudo python3 server_installer.py
```

The installer will:

1. Check the system (OS, Go, Ollama, GPU).
2. Clone or pull the repository.
3. Build the Go binary into `/usr/local/bin/vaultagent`.
4. Ask interactively for your IMAP, model, and security settings, then write the config.
5. Create and enable a **systemd service** named `vaultagent`.
6. Start the service automatically.

### Option B — Desktop Installer (Windows)

```bash
git clone https://github.com/rhantschk-cmyk/agent-Project.git
cd agent-Project/src/install
python desktop_installer.py
```

The installer will:

1. Check for the Odin compiler and build the desktop client.
2. Copy the executable and config into `C:\Program Files\VaultAgent\`.
3. Create a **desktop shortcut**.
4. Configure the `server_ip` and `verify_key` to point at your server.

### Option D — Uninstall

Both components can be fully removed back to a clean state:

**Server (removes the systemd service, binary and `/etc/vaultagent`):**

```bash
sudo python3 src/install/server_uninstaller.py
```

**Desktop (removes the app and the desktop shortcut):**

```bash
python src/install/desktop_uninstaller.py
```

Alternatively, use the CLI:

```bash
sudo agent-cli uninstall   # then pick server or desktop
```

### Option C — Manual Build

**Server:**

```bash
cd src/Server
cp config.json.example config.json   # then edit your credentials
go build -o vaultagent .
./vaultagent
```

> ⚠️ **Important:** the server is a multi-file `package main`, so always build/run
> the **whole package** — use `go build .` or `go run .` (with the dot).
> Running `go run main.go` alone **will fail** with `undefined` errors, because the
> other files (config, agent, mails, tools, memory, cliserver, monitoringserver,
> logging) are not included.

**Desktop client (Odin):**

```bash
cd src/Client
odin build . -out:VaultAgent
```

---

## ⚙️ Server Services & Ports

### 💬 CLI / Chat Server (port `8080`)

Accepts JSON requests over TCP/HTTP and routes them through the same agent loop used for email processing. Authenticated via `cli_secret_key` from the config.

### 📊 Monitoring Service (port `9000`)

Returns JSON system metrics on request: CPU usage (%), RAM usage, free disk space (GB), and system uptime.

---

## 🛠️ Configuration (`config.json`)

The entire system is configured through a central `config.json`. See [`config.json.example`](./config.json.example) for a template with placeholders.

| Section | Keys | Description |
| :--- | :--- | :--- |
| `e-mail` | `username`, `app_token`, `server`, `draft_folder` | IMAP credentials & draft target folder |
| `program` | `model`, `knowledge_dir`, `cli_secret_key` | LLM model, knowledge base path, API auth key |
| `memory` | `memory_compression_time`, `memory_file`, `memory_compress_promt` | Memory file and auto-compression settings |
| `sys_promts` | `standard`, `important`, `classify`, `cli` | Detailed system prompts for each context |

> ⚠️ **Security:** never commit real credentials. The included `config.json.example` uses placeholders — copy it to `config.json` and fill in your own values.

---

## 🖥️ Desktop Client

Located in `src/Client/` and written in **Odin** using `raylib`. It connects to the server over HTTP and provides two tabs:

- **Agent Chat** — send prompts to the server's port `8080` API and display the response.
- **Monitoring** — auto-refreshes system stats from port `9000` every 3 seconds with visual progress bars.

---

## 🧰 CLI Tool

Located in `src/cli/` and written in **Python**. It is a server-side command-line tool for checking code, managing the service, and updating the software. See the [CLI tool's own documentation](./src/cli/README.md) for usage.

---

## 📁 Project Structure

```
├── README.md
├── config.json.example
└── src/
    ├── install/
    │   ├── server_installer.py      # systemd installer (Linux)
    │   ├── desktop_installer.py     # Windows desktop installer
    │   ├── server_uninstaller.py    # removes the systemd service + files
    │   └── desktop_uninstaller.py   # removes the desktop app + shortcut
    ├── cli/
    │   ├── agent-cli.py              # server-side CLI tool
    │   └── README.md
    ├── Client/
    │   ├── config.json              # client <-> server connection config
    │   └── main.odin                # desktop GUI (raylib)
    └── Server/
        ├── main.go                  # entry point
        ├── agent.go                 # LLM agent loop & tool execution
        ├── cliserver.go             # HTTP API server (port 8080)
        ├── monitoringserver.go      # monitoring service (port 9000)
        ├── config.go                # config loader
        ├── mails.go                 # IMAP handling & draft creation
        ├── memory.go                # long-term memory system
        ├── tools.go                 # tool definitions for the LLM
        └── docs/                    # knowledge base (templates, pricing)
```

---

## 🔄 How the Agent Works

### 1. Email Polling & Parsing

The backend connects to Gmail via IMAP (`imap.gmail.com:993`) using a Gmail App Token. Incoming emails are parsed into internal Go structs.

### 2. AI Classification

Each email is analyzed by a fast model and sorted into:
- `SPAM` — unwanted emails (ignored).
- `IMPORTANT` — important or urgent requests.
- `STANDARD` — normal business or customer inquiries.

### 3. Agent Loop & Tools

If an email needs an answer, the autonomous agent loop starts. The model can dynamically call the following tools:

| Tool | Parameters | Description |
| :--- | :--- | :--- |
| `read_memory` | *none* | Reads saved facts, agreements & notes from `memory.txt`. |
| `write_memory` | `fact` (string) | Stores new important facts/prices permanently. |
| `search_inbox` | `query` (string) | Searches the inbox for earlier messages on a topic. |
| `get_conversation_history` | `email` (string), `count` (int) | Fetches the last N emails from a sender. |
| `list_all_docs` | *none* | Lists all available templates/documents in the `docs/` folder. |
| `read_docs` | `doc` (string) | Reads the exact content of a specific Markdown template. |
| `finish_and_reply` | `text` (string), `notes` (string) | Finishes the research and hands over the final email text. |

### 4. Template Processing & Draft Creation

If the agent finds a matching template via `read_docs`, it adopts the wording and only adjusts variable data (names, dates, prices). After calling `finish_and_reply`, the text is converted into a Gmail-compatible MIME structure and placed in `[Gmail]/Drafts` with the subject `[ENTWURF] ...`.

---

## 🚀 Roadmap

- [x] Single email monitor (Standard, v0.4)
- [ ] Unlimited parallel monitors (Pro)
- [ ] Extended tool API (Pro)
- [ ] Multi-provider support (Pro)

---

## ⚠️ Disclaimer / Liability

THIS SOFTWARE IS PROVIDED "AS IS" FOR PRIVATE/SELF-RESPONSIBLE USE. The developer assumes no liability for server security, data loss, incorrectly generated email drafts, outages, or hardware damage caused by high system load or insufficient cooling. The security of your server is 100% your responsibility.

---

## 📄 License

Proprietary — all rights reserved. This software is **not** open-source. See the [LICENSE](./LICENSE) file for details. Contact the developer for commercial / Pro licensing.

---

## 📬 Contact

- Repository: [github.com/rhantschk-cmyk/agent-Project](https://github.com/rhantschk-cmyk/agent-Project)
- Issues & feature requests: open an issue on GitHub
