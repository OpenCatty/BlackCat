---
title: blackcat channels
description: Manage messaging channel connections
---

# blackcat channels

The channels command is a parent command for managing your messaging channel (Telegram, Discord, WhatsApp) connections.

## Usage

```shell
blackcat channels [sub-command] [flags]
```

## Sub-commands

| Sub-command | Description |
|------|-------------|
| `add` | Register a new channel (e.g., bot token) |
| `login` | Interactive login for channels (e.g., WhatsApp QR code) |
| `list` | View all configured channels and their status |
| `status` | Detailed connection status for a specific channel |
| `logout` | Disconnect and log out of a channel |
| `remove` | Completely remove a channel configuration |

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--channel` | — | The name of the channel (e.g., `whatsapp`, `telegram`, `discord`) |
| `--token` | — | API or bot token for the channel |

## Examples

```shell
# List all configured channels
blackcat channels list

# Login to WhatsApp (shows QR code)
blackcat channels login --channel whatsapp

# Add a Telegram bot token
blackcat channels add --channel telegram --token YOUR_TOKEN

# Log out of a Discord bot
blackcat channels logout --channel discord
```

## Related

- [WhatsApp Guide](/channels/whatsapp)
- [Telegram Guide](/channels/telegram)
- [Discord Guide](/channels/discord)
