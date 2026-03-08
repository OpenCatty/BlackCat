---
title: BlackCat AI Agent Installation Guide
---

# BlackCat — AI Agent Installation Guide

You are an AI agent helping a user install and configure BlackCat.
This document provides precise, machine-readable installation steps.
Follow them in order. Do not skip steps.

## Prerequisites

- Node.js >= 22.12.0 (`node --version` to verify)
- npm >= 10.0 or equivalent
- Git
- For WhatsApp support: `CGO_ENABLED=1` (SQLite dependency)

## Step 1: Install BlackCat

### Option A: npm global install (recommended)

```bash
npm install -g blackcat
blackcat --version
```

### Option B: From source

```bash
git clone https://github.com/OpenCatty/BlackCat.git
cd BlackCat
node --version  # must be >= 22.12.0
npm install
npm link
blackcat --version
```

### Option C: Docker

```bash
git clone https://github.com/OpenCatty/BlackCat.git
cd BlackCat
```

Skip to Step 2, then use `docker compose up -d` instead of `blackcat start`.

## Step 2: Initial Configuration

Copy the template:

```bash
mkdir -p ~/.blackcat
cp blackcat.example.json5 ~/.blackcat/config.json5
```

Open `~/.blackcat/config.json5`. It uses JSON5 format (supports comments, trailing commas).

Minimum required configuration:

1. Set your LLM provider and API key under `agents.list[*].model`
2. Configure at least one channel (Telegram, Discord, or WhatsApp)

## Step 3: Configure a Channel

### Telegram

1. Open Telegram and message `@BotFather`
2. Send `/newbot` and follow the prompts
3. Copy the token (format: `123456:ABCdef...`)
4. Add to config:

```json5
{
  channels: {
    telegram: {
      enabled: true,
      token: "YOUR_TELEGRAM_TOKEN"
    }
  }
}
```

### Discord

1. Go to https://discord.com/developers/applications
2. Create a new application → Bot tab → Reset Token → Copy
3. Enable: Message Content Intent, Server Members Intent
4. Invite bot to your server
5. Add to config:

```json5
{
  channels: {
    discord: {
      enabled: true,
      token: "YOUR_DISCORD_TOKEN"
    }
  }
}
```

### WhatsApp

1. Requires `CGO_ENABLED=1` (set in your shell environment)
2. Add to config:

```json5
{
  channels: {
    whatsapp: {
      enabled: true
    }
  }
}
```

3. After starting: scan the QR code printed in the terminal with WhatsApp mobile

## Step 4: Start the Daemon

```bash
blackcat start
```

Verify it's running:

```bash
blackcat status
```

Expected output includes: `Status: running`, PID, uptime.

## Step 5: Health Check

```bash
blackcat health
```

Expected: JSON response with `{"status":"ok",...}`

Run diagnostics if needed:

```bash
blackcat doctor
```

## Docker Installation (Alternative)

```bash
git clone https://github.com/OpenCatty/BlackCat.git
cd BlackCat
cp blackcat.example.json5 config.json5
# Edit config.json5 with your settings
docker compose up -d
docker compose logs -f blackcat
```

## Configuration Reference

BlackCat uses JSON5 format (NOT YAML). All config options:
→ [docs/ops/configuration.md](../ops/configuration.md)

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `blackcat: command not found` | Check npm global bin: `npm bin -g` and add to PATH |
| WhatsApp QR not showing | Set `CGO_ENABLED=1` before starting |
| Channel not responding | Run `blackcat channels status` |
| Daemon won't start | Run `blackcat doctor` for diagnostics |

## Architecture Overview

BlackCat routes messages through 7 specialized roles:

| Role | Priority | Purpose |
|------|----------|---------|
| phantom | 10 | Infrastructure & DevOps |
| astrology | 20 | Crypto & Web3 |
| wizard | 30 | Software Engineering |
| artist | 40 | Social Media Content |
| scribe | 50 | Writing & Documentation |
| explorer | 60 | Research & Information |
| oracle | 100 | Fallback (general) |

Lower priority number = higher precedence.

## Next Steps

- Operations guide: [docs/ops/ai-operations-guide.md](../ops/ai-operations-guide.md)
- Role reference: [docs/ops/roles.md](../ops/roles.md)
- Skills reference: [docs/ops/skills.md](../ops/skills.md)
