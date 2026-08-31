# Agent Software — Entwickler- & Betreiber-Guide

Dieser Guide erklärt den vollständigen aktuellen Stand des Projekts: was es tut,
welche Komponenten es gibt, wie man installiert/deinstalliert, was die CLI kann,
und was in der aktuellen Sitzung alles geändert wurde.

---

## 1. Überblick

Ein autonomer KI-E-Mail-Agent für ein Hotel. Die Software überwacht ein Gmail-
Postfach per IMAP IDLE (ereignisgesteuert), klassifiziert eingehende Mails
(SPAM / IMPORTANT / STANDARD), recherchiert mit einem LLM-Agenten in der
Wissensbasis und dem Langzeitgedächtnis und erstellt **Entwürfe** (Drafts)
direkt im Postfach.

- **Standard (v0.4):** schlichter Ablauf `neue Mail → klassifizieren → Agent → Entwurf → fertig`.
- **Pro (geplant):** separates Programm auf Basis des Standards — mit unbegrenzt
  vielen parallelen E-Mail-Monitoren, autonomen Versand (SMTP), Web-Dashboard,
  Whitelist, Eskalation, besseres Monitoring und Backup. Aktivierung später über
  einen Lizenzschlüssel in der CLI.

---

## 2. Projektstruktur

```
agent-Project/
├── README.md                      # Produkt-README (englisch)
├── GUIDE.md                       # dieser Guide
├── config.json.example            # Config-Template (Wurzel)
└── src/
    ├── Server/                    # Go-Backend (package main)
    │   ├── main.go                # Einstieg, Banner, Shutdown, Orchestrierung
    │   ├── config.go              # Config-Structs + LoadConfig()
    │   ├── agent.go               # Agent-Loop + Tool-Ausführung + Ollama-Aufrufe
    │   ├── mails.go               # IMAP-Empfang (IDLE), Klassifizierung, Draft-Erstellung
    │   ├── tools.go               # Tool-Definitionen + Dispatch
    │   ├── memory.go              # Langzeitgedächtnis + Auto-Kompression
    │   ├── cliserver.go           # HTTP-Server :8080 (/api/agent/ask, /health)
    │   ├── monitoringserver.go    # HTTP-Server :9000 (/api/stats)
    │   ├── logging.go             # Logging-Helper (stdout + Datei)
    │   ├── config_test.go         # Unit-Tests Config
    │   ├── memory_test.go         # Unit-Tests Memory
    │   ├── config.json.example    # lokales Config-Template
    │   └── docs/                  # Wissensbasis (Templates, Preise, Policies)
    ├── Client/                    # Odin-Desktop-Client (raylib GUI)
    │   ├── main.odin
    │   └── config.json            # server_ip + verify_key
    ├── install/                   # Python-Installer/Uninstaller
    │   ├── server_installer.py
    │   ├── desktop_installer.py
    │   ├── server_uninstaller.py
    │   └── desktop_uninstaller.py
    └── cli/
        ├── agent-cli.py           # Server-seitiges CLI-Tool (Python)
        └── README.md              # CLI-Doku
```

---

## 3. Wie startet man den Server?

> ⚠️ **Wichtig:** der Server ist ein **Multifile-`package main`**. Immer das
> **ganze Paket** bauen/starten mit dem Punkt:
> - richtig: `go run .`  oder  `go build .`
> - **falsch:** `go run main.go` → führt zu `undefined: logf`, `undefined: LoadConfig`, usw.

```bash
cd src/Server
cp config.json.example config.json    # einmalig: Credentials eintragen
go run .
```

Beim Start erscheint ein Banner, danach gestartete Logs mit Zeitstempel:

```
Agent Software v0.4 (Standard)
Pro Version coming soon — unlimited monitors
...
-> Loading Config-File
-> Done
-> [Memory] Started Memory Compressor
-> [CLI] Starte Server auf :8080
-> [Monitoring] Starte Server auf :9000
-> Listening for Emails
```

Beenden: `Ctrl+C` oder `kill -TERM <pid>`. Das Programm fährt **graceful** herunter
(alle Komponenten: Memory-Compressor, MailSection, CLI-Server).

---

## 4. Logging

- Alle Meldungen gehen über `logf(...)` (in `logging.go`).
- Ausgabe **immer nach stdout**, optional zusätzlich **in eine Datei**.
- Datei-Logging aktivieren: `AGENT_LOG_FILE=/var/log/email-agent.log ./email-agent`
- Jede Zeile hat einen Zeitstempel `[2026-08-31 16:40:41]`.

---

## 5. Server-Endpoints

| Endpoint | Port | Beschreibung |
| :--- | :--- | :--- |
| `POST /api/agent/ask` | 8080 | Fragt den Agenten (Auth via `verify_key` = `cli_secret_key`) |
| `GET /health` | 8080 | Healthcheck: `{"status":"ok","version":"...","uptime":"..."}` |
| `GET /api/stats` | 9000 | Systemmetriken (CPU, RAM, Disk, Uptime) |

---

## 6. Installieren als Systemdienst (systemd)

Der **Server-Installer** erledigt alles automatisch:

```bash
git clone https://github.com/rhantschk-cmyk/agent-Project.git
cd agent-Project/src/install
sudo python3 server_installer.py
```

Ablauf:
1. System-Check (OS, Go, Ollama, GPU)
2. Repo nach `/opt/email-agent` klonen/pullen
3. Go-Binary nach `/usr/local/bin/email-agent` bauen
4. Interaktiv Config abfragen → `/etc/email-agent/config.json` (inkl. Wissensbasis in `/etc/email-agent/docs`)
5. systemd-Unit `email-agent.service` anlegen, aktivieren und starten

Kontrollieren:

```bash
systemctl status email-agent
journalctl -u email-agent -f      # Logs ansehen
curl localhost:8080/health        # Healthcheck
```

Die systemd-Unit:

```ini
[Unit]
Description=Agent Software - Autonomous AI Email Agent
After=network-online.target ollama.service
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/email-agent
WorkingDirectory=/etc/email-agent
Restart=always
RestartSec=5
Environment=HOME=/etc/email-agent

[Install]
WantedBy=multi-user.target
```

---

## 7. Deinstallieren (Server)

```bash
sudo python3 src/install/server_uninstaller.py
```

Entfernt sauber:
- systemd-Service `email-agent` (gestoppt + disabled + Unit gelöscht)
- Binary `/usr/local/bin/email-agent`
- Config-Verzeichnis `/etc/email-agent` (inkl. config.json und docs)
- optional das geklonte Repo `/opt/email-agent`

---

## 8. Desktop-Client (Windows)

**Installieren:**

```bash
cd src/install
python desktop_installer.py
```

Ablauf:
1. Odin-Compiler prüfen und Client bauen (`EmailAgent.exe`)
2. Nach `C:\Program Files\EmailAgent\` kopieren (inkl. `config.json`)
3. Desktop-Verknüpfung `EmailAgent.lnk` erstellen
4. `server_ip` + `verify_key` abfragen

**Deinstallieren:**

```bash
python desktop_uninstaller.py
```

Entfernt den Installationsordner und die Desktop-Verknüpfung.

---

## 9. Das CLI-Tool (`agent-cli`)

Eine server-seitige Befehlszeilen-Werkzeug in Python (`src/cli/agent-cli.py`).

Installieren als globaler Befehl (optional):

```bash
sudo cp src/cli/agent-cli.py /usr/local/bin/agent-cli
sudo chmod +x /usr/local/bin/agent-cli
```

### Alle Befehle

| Befehl | Was passiert | Root nötig? |
| :--- | :--- | :---: |
| `agent-cli check` | Struktur prüfen + `go vet` + `go test` + `staticcheck` (falls installiert) | nein |
| `agent-cli install` | Fragt Server/Desktop und startet den passenden Installer | Server: ja |
| `agent-cli uninstall` | Fragt Server/Desktop und startet den passenden Uninstaller | Server: ja |
| `agent-cli start` | `systemctl start email-agent` | ja |
| `agent-cli stop` | `systemctl stop email-agent` | ja |
| `agent-cli restart` | `systemctl restart email-agent` | ja |
| `agent-cli status` | `systemctl status email-agent` | nein |
| `agent-cli config` | Zeigt `config.json` (lokal oder `/etc/email-agent`) | nein |
| `agent-cli update` | `git pull` + Server-Binary neu bauen | nein |
| `agent-cli ask <prompt>` | Fragt den laufenden Agenten über `:8080` | nein |
| `agent-cli help` | Zeigt Hilfe | – |

Beispiele:

```bash
agent-cli check                       # Codeprüfung
sudo agent-cli install                # installieren
sudo agent-cli uninstall              # deinstallieren
sudo agent-cli status                 # Dienststatus
agent-cli ask "Wie viel kostet ein Zimmer?"    # Agent direkt fragen
```

---

## 10. Unit-Tests

```bash
cd src/Server
go test ./...
```

Abgedeckt:
- `config_test.go` — gültige Config laden, fehlende Datei, ungültiges JSON
- `memory_test.go` — Fakt speichern, leeres Gedächtnis, Lesen, Anhängen

`agent-cli check` führt diese Tests automatisch mit aus.

---

## 11. Was in dieser Sitzung (neu) geändert wurde

### Deutsche Übersicht der letzten Arbeiten:

1. **Neue README** (englisch) — beschreibt Ablauf, Pro-Version, Installation, CLI.
2. **`config.json.example`** — Template ohne echte Zugangsdaten.
3. **Python-Installer** — `server_installer.py` (systemd) und `desktop_installer.py` (Windows).
4. **Python-Uninstaller** — `server_uninstaller.py` und `desktop_uninstaller.py` (sauber entfernen).
5. **Python-CLI** — `agent-cli.py` mit `check`, `install`, `uninstall`, `start/stop/restart/status`, `config`, `update`, `ask`.
6. **Sicherheit:**
   - echte Gmail-Zugangsdaten aus `config.json` aus dem Git-Index entfernt (untracked + ignoriert)
   - `.gitignore` erweitert (config, memory, logs, build, pycache, Binary)
   - Binärdatei `agent-Projekt` und `__pycache__` getrackt entfernt
7. **Banner** in `main.go` — `Agent Software v0.4 (Standard)` + Pro-Hinweis beim Start.
8. **Logging-System** — neues `logging.go`, alle Ausgaben via `logf`, optional Datei.
9. **Graceful Shutdown** — SIGINT/SIGTERM beendet alle Komponenten sauber
   (inkl. Fix, dass IMAP-IDLE jetzt abgebrochen werden kann).
10. **`GET /health`** — auf Port 8080.
11. **Ollama-Fehlerbehandlung** — `agent.go`/`memory.go` loggen Fehler sauber statt
    still abzubrechen; `finish_and_reply` über definierten Sentinel; `new(bool)`-Leak gefixt.
12. **Unit-Tests** — `config_test.go`, `memory_test.go`.

---

## 12. Noch offen / geplant (Pro)

- Unbegrenzte parallele E-Mail-Monitore (Pro)
- Autonomer Versand (SMTP) (Pro)
- Web-Dashboard / Review-Queue (Pro)
- Absender-Whitelist + Eskalation (Pro)
- Besseres Monitoring + Backup (Pro)
- Lizenzschlüssel-Aktivierung in der CLI (Pro)
