---
title: BlackCat Manual Installation
---

# BlackCat — Manual Installation Guide

Welcome, human. You've decided to install BlackCat manually. Brave — or stubborn. Possibly both.

BlackCat is an AI agent router that connects your chat channels (Telegram, Discord, WhatsApp) to specialized AI subagents. This guide walks you through the installation step by step.

## What You'll Need

Before we begin, make sure you have:

- Node.js version 22.12.0 or higher (`node --version` will tell you)
- npm version 10.0 or higher
- Git installed and working
- A terminal that isn't afraid of you

For WhatsApp support, you'll also need `CGO_ENABLED=1` in your environment. Don't worry about what that means yet. Just know it's important for the database stuff.

## Option A: The Easy Way (npm)

This is the fastest path to a working BlackCat:

```bash
npm install -g blackcat
blackcat --version
```

If you see a version number, you're golden. Skip ahead to "Setting Up Your Config."

## Option B: The Source Way

Maybe you want to poke around the code. Maybe you don't trust pre-built packages. Either way, here's how to build from source:

```bash
git clone https://github.com/OpenCatty/BlackCat.git
cd BlackCat
node --version
npm install
npm link
blackcat --version
```

The `npm link` command makes `blackcat` available globally on your system. Think of it as telling your shell "hey, this command exists now."

## Option C: The Docker Way

Containers aren't just for shipping things across oceans anymore. If you prefer containerized everything:

```bash
git clone https://github.com/OpenCatty/BlackCat.git
cd BlackCat
cp blackcat.example.json5 config.json5
```

Now edit that config file with your settings, then:

```bash
docker compose up -d
```

Docker handles the rest. Check logs with `docker compose logs -f blackcat`.

## Setting Up Your Config

BlackCat needs a home for its configuration. Create it:

```bash
mkdir -p ~/.blackcat
cp blackcat.example.json5 ~/.blackcat/config.json5
```

Now open `~/.blackcat/config.json5` in your favorite editor. This is JSON5 format, which means you can use comments and trailing commas. Much friendlier than strict JSON.

You need to configure two things minimum:

1. **Your LLM provider** — Under `agents.list`, set which model you want to use and provide your API key
2. **At least one channel** — Telegram, Discord, or WhatsApp

Don't panic. We'll walk through each channel below.

## Connecting Telegram

1. Open Telegram on your phone or desktop
2. Message `@BotFather` (he's the official bot for making bots)
3. Send `/newbot` and follow his instructions
4. When done, he'll give you a token that looks like `123456:ABCdefGHIjklMNOpqrSTUvwxyz`
5. Copy that token into your config:

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

Save the file. You're connected.

## Connecting Discord

1. Go to https://discord.com/developers/applications
2. Click "New Application", give it a name
3. Go to the "Bot" tab on the left
4. Click "Reset Token" and copy the token
5. Scroll down and enable these two toggles:
   - Message Content Intent
   - Server Members Intent
6. Use the OAuth2 URL Generator to create an invite link
7. Add the bot to your server
8. Paste your token into the config:

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

## Connecting WhatsApp

WhatsApp is the trickiest of the three because it uses a local SQLite database. Make sure `CGO_ENABLED=1` is set in your environment before starting.

Add to your config:

```json5
{
  channels: {
    whatsapp: {
      enabled: true
    }
  }
}
```

When you first run BlackCat, it'll print a QR code to your terminal. Scan it with WhatsApp on your phone (Settings → Linked Devices → Link a Device).

## Starting BlackCat

You've made it this far. Time to flip the switch:

```bash
blackcat start
```

Verify it's actually running:

```bash
blackcat status
```

You should see something like `Status: running` along with a PID and uptime.

## Health Check

Make sure everything's healthy:

```bash
blackcat health
```

You want to see `{"status":"ok"}` in the output. If something's wrong, run:

```bash
blackcat doctor
```

This will diagnose common issues and suggest fixes.

## Troubleshooting

**"blackcat: command not found"**

Your shell doesn't know where BlackCat lives. Find where npm puts global packages:

```bash
npm bin -g
```

Add that directory to your PATH.

**WhatsApp QR code never appears**

Double-check `CGO_ENABLED=1` is set before starting. This is required for the SQLite driver.

**Channel isn't responding to messages**

Check the channel status:

```bash
blackcat channels status
```

This shows which channels are connected and whether they're healthy.

## What's Next?

Your BlackCat is running. Now you can:

- Read the [operations guide](../ops/ai-operations-guide.md) for day-to-day management
- Learn about [roles](../ops/roles.md) to understand how messages get routed
- Explore [skills](../ops/skills.md) to see what BlackCat can do

Welcome to the network. Your AI agents are waiting.
