---
title: Discord Guide
description: Set up and manage Discord messaging channel
---

# Discord

The Discord channel allows you to interact with BlackCat via Discord messages.

## 1. Create a Bot

To use Discord, you must first create a bot:
1. Go to the [Discord Developer Portal](https://discord.com/developers/applications).
2. Create a **New Application**.
3. Under the **Bot** tab, generate a new bot and copy its **Token**.
4. Enable the **Message Content Intent** toggle.

## 2. Add Token

Once you have your token, you can add it to BlackCat using the CLI:

```bash
blackcat channels add --channel discord --token YOUR_TOKEN
```

Alternatively, you can use the interactive login:

```bash
blackcat channels login --channel discord
```

The wizard will prompt you for your bot token and save it securely.

## 3. Interaction

Once the bot is added, invite it to your server and mention it to start interacting:

```bash
@BlackCat, can you show me the files in the current directory?
```

## Related

- [blackcat channels](/cli/channels)
- [WhatsApp Guide](/channels/whatsapp)
- [Telegram Guide](/channels/telegram)
