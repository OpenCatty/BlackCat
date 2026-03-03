---
title: WhatsApp Guide
description: Set up and manage WhatsApp messaging channel
---

# WhatsApp

The WhatsApp channel allows you to interact with BlackCat via WhatsApp messages.

## 1. Prerequisites

WhatsApp support requires CGO for SQLite. If you are building from source, ensure you build with `CGO_ENABLED=1`:

```bash
CGO_ENABLED=1 go build -o blackcat .
```

## 2. Login

To connect your WhatsApp account, run the login command and scan the QR code that appears in your terminal:

```bash
blackcat channels login --channel whatsapp
```

Scanning the QR code will link BlackCat as a "companion device" in your WhatsApp settings.

## 3. Access Control

You can restrict who can interact with BlackCat using the `allowFrom` and `dmPolicy` fields in your `config.yaml`:

```yaml
channels:
  whatsapp:
    enabled: true
    allowFrom: ["1234567890@s.whatsapp.net"]
    dmPolicy: "restrict"
```

## 4. Troubleshooting

- **QR Expired:** If the QR code expires before you can scan it, simply re-run the login command.
- **Not Linked:** If you lose connection, verify the "Linked Devices" section in your WhatsApp mobile app and re-run the login command if necessary.
- **Reconnect:** Use `blackcat channels logout` and then `login` to refresh your session.

## Related

- [blackcat channels](/cli/channels)
- [Telegram Guide](/channels/telegram)
- [Discord Guide](/channels/discord)
