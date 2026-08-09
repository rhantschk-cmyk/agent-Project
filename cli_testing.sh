#!/bin/bash

# Konfiguration
URL="http://localhost:8080/api/agent/ask"
KEY="abcdefgh"
PROMPT="Gibt es offene Anfragen zum Thema Tennis4One?"

# JSON-Payload erstellen und Anfrage senden
curl -s -X POST "$URL" \
  -H "Content-Type: application/json" \
  -d '{
    "verify_key": "'"$KEY"'",
    "prompt": "'"$PROMPT"'"
  }' | jq .
