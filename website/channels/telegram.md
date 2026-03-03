---
title: Telegram Guide
description: Set up and manage Telegram messaging channel
---

# Telegram

The Telegram channel allows you to interact with BlackCat via Telegram messages.

## 1. Create a Bot

To use Telegram, you must first create a bot:
1. Open Telegram and search for **BotFather**.
2. Start a chat and send the `/newbot` command.
3. Follow the instructions to give your bot a name and a username.
4. BotFather will provide you with a unique API token.

## 2. Add Token

Once you have your token, you can add it to BlackCat using the CLI:

```bash
blackcat channels add --channel telegram --token YOUR_TOKEN
```

Alternatively, you can use the interactive login:

```bash
blackcat channels login --channel telegram
```

The wizard will prompt you for your bot token and save it securely.

## 3. Restrict Access

To ensure only you can interact with BlackCat, you can restrict access using the `allowFrom` configuration:

```yaml
channels:
  telegram:
    enabled: true
    token: "your-bot-token"
    allowFrom: ["your-telegram-username"]
```

## Related

- [blackcat channels](/cli/channels)
- [WhatsApp Guide](/channels/whatsapp)
- [Discord Guide](/channels/discord)
