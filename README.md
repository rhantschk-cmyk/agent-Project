# 🛡️ Autonomous AI Email Agent & Server Engine (Go-Backend)

Ein extrem schnelles, autonomes und event-getriebenes E-Mail-Verarbeitungs- und KI-Agenten-System, entwickelt in **Go** und betrieben durch lokale **Ollama LLMs**.

---

## 📌 Übersicht & Zweck

Das System überwacht kontinuierlich eingehende E-Mails über IMAP, klassifiziert diese automatisch mittels KI, sammelt Kontext aus Dokumenten und dem Langzeitgedächtnis und erstellt fertige Antwort-Entwürfe direkt im Gmail-Postfach.

Zusätzlich bietet das Backend zwei Netzwerk-Schnittstellen:
1. **Interaktiver CLI-Server (Port 8080):** Ein abgesicherter JSON-Endpoint für externe Befehle und Live-Anfragen an den Agenten.
2. **Monitoring CLI / Server (Port 9000):** Ein schlanker Service zur Ausgabe aktueller Hardware- & System-Metriken.

---

## ⚡ System-Anforderungen

Das System ist speziell für **dedizierte Server mit lokaler GPU-Beschleunigung** ausgelegt, um schnelle Inference-Zeiten der lokalen Modelle (z. B. `qwen2.5:14b`) zu garantieren.

### 🖥️ Hardware-Anforderungen
> ⚠️ **Wichtiger Hinweis:** Der Betrieb auf schwächerer Hardware kann zu sehr hohen Antwortzeiten, Timeouts oder Systemabstürzen führen!

* **Arbeitsspeicher (RAM):** Mindestens **16 GB** (32 GB empfohlen)
* **Grafikspeicher (VRAM):** Mindestens **16 GB VRAM** (dedizierte GPU)
* **Prozessor (CPU):** Mindestens **10 Cores** mit hoher Single-Core-Performance
* **Kühlung:** Ausreichende und leistungsfähige Kühlung (aufgrund dauerhafter GPU/CPU-Last bei LLM-Generierung)
* **Netzwerk:** Stabile Internetverbindung für IMAP/SMTP-Synchronisation

### 📦 Software-Anforderungen
* **Betriebssystem:** Linux (Debian 12 / Ubuntu Server empfohlen)
* **Go:** Version 1.22 oder neuer
* **Treiber:** Aktuelle GPU-Treiber (NVIDIA CUDA / AMD ROCm)
* **Ollama:** Lokal installiert und als Service aktiv (`ollama serve`)

---

## ⚠️ Disclaimer / Haftungsausschluss

HAFTUNGSAUSSCHLUSS:
Dieses Programm wird "wie es ist" (as is) zur privaten/eigenverantwortlichen Nutzung bereitgestellt. 
Der Entwickler übernimmt keinerlei Haftung für die Sicherheit des Servers, Datenverluste, fehlerhaft 
generierte E-Mail-Entwürfe, Ausfälle oder Hardware-Schäden, die durch hohe Systemlasten oder 
unzureichende Kühlung entstehen. Die Absicherung des Servers liegt zu 100% beim Betreiber.

---

## 🔄 Funktionsweise & Architektur

### 1. E-Mail Polling & Parsing
* Das Backend verbindet sich über IMAP (`imap.gmail.com:993`) mit dem Gmail-Konto (unter Verwendung eines Gmail App-Tokens).
* Eingehende E-Mails werden eingelesen und in interne Go-`structs` geparst.

### 2. KI-Klassifizierung
Jede E-Mail wird vorab durch ein schnelles Modell analysiert und in eine von drei Kategorien eingeordnet:
* `SPAM`: Unerwünschte E-Mails (werden ignoriert).
* `IMPORTANT`: Wichtige oder dringende Anfragen mit hoher Priorität.
* `STANDARD`: Normale Geschäfts- oder Kundenanfragen.

### 3. Agenten-Loop & Werkzeuge (Tools)
Erfordert eine E-Mail eine Antwort, startet der autonome Agenten-Loop. Das Modell kann folgende Tools dynamisch aufrufen:

| Tool-Name | Parameter | Beschreibung |
| :--- | :--- | :--- |
| `read_memory` | *keine* | Liest abgetrennte Fakten, Vereinbarungen & Notizen aus `memory.txt`. |
| `write_memory` | `fact` (string) | Speichert neue wichtige Fakten/Preise dauerhaft im Langzeitgedächtnis. |
| `search_inbox` | `query` (string) | Durchsucht den E-Mail-Posteingang nach früheren Nachrichten zu einem Thema. |
| `get_conversation_history` | `email` (string), `count` (int) | Ruft die letzten N E-Mails eines bestimmten Absenders ab. |
| `list_all_docs` | *keine* | Listet alle verfügbaren Vorlagen/Dokumente im `docs/`-Ordner auf. |
| `read_docs` | `doc` (string) | Liest den genauen Inhalt eines spezifischen Markdown-Templates aus. |
| `finish_and_reply` | `text` (string), `notes` (string) | Beendet die Recherche und übergibt den finalen E-Mail-Text. |

### 4. Template-Verarbeitung & Draft-Erstellung
* Findet der Agent über `read_docs` ein passendes Template, übernimmt er den Wortlaut und passt nur variable Daten (Namen, Daten, Preise) an.
* Nach Aufruf von `finish_and_reply` wird der Text in eine Gmail-konforme MIME-Struktur konvertiert und mit dem Betreff **`[ENTWURF] ...`** im Ordner `[Gmail]/Drafts` abgelegt.

---

## 🌐 Server-Dienste & Ports

### 💬 CLI / Chat-Server (`Port 8080`)
* Wartet auf eingehende JSON-Anfragen über TCP/HTTP.
* **Sicherheit:** Prüft den mitgesendeten Key gegen `cli_secret_key` aus der `config.json`.
* **Funktion:** Ermöglicht die direkte Interaktion mit dem Agenten-Loop über externe Terminal-Tools oder Apps.

### 📊 Monitoring Service (`Port 9000`)
* Liefert auf Anfrage aktuelle Hardware-Statistiken als JSON zurück.
* **Metriken:** CPU-Auslastung (%), RAM-Belegung / Gesamtspeicher, freier Festplattenspeicher (GB) & System-Uptime.

---

## 🛠️ Konfiguration (`config.json`)

Die Konfiguration wird über eine zentrale `config.json` geregelt:

{
  "e-mail": {
    "username": "user@gmail.com",
    "app_token": "xxxx-xxxx-xxxx-xxxx",
    "server": "imap.gmail.com:993",
    "draft_folder": "[Gmail]/Drafts"
  },
  "program": {
    "model": "qwen2.5:14b",
    "knowledge_dir": "docs",
    "cli_secret_key": "DEIN_GEHEIMER_KEY"
  },
  "memory": {
    "memory_compression_time": 5,
    "memory_file": "memory.txt",
    "memory_compress_promt": "..."
  },
  "sys_promts": {
    "standard": "...",
    "important": "...",
    "classify": "...",
    "cli": "..."
  }
}
