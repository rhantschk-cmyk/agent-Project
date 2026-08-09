#!/bin/bash

# Konfiguration
URL="http://localhost:9000/api/stats"

echo "-> Schicke Anfrage an Monitoring-Server auf Port 9000..."

# Anfrage mit curl senden
RESPONSE=$(curl -s -X GET "$URL")

# Prüfen, ob eine Antwort kam
if [ -z "$RESPONSE" ]; then
  echo "❌ Fehler: Keinen Antwort vom Monitoring-Server erhalten (läuft der Server?)."
  exit 1
fi

echo "-> Antwort erhalten:"
echo "$RESPONSE" | jq .
