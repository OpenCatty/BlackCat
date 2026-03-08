---
summary: "CLI reference for `blackcat voicecall` (voice-call plugin command surface)"
read_when:
  - You use the voice-call plugin and want the CLI entry points
  - You want quick examples for `voicecall call|continue|status|tail|expose`
title: "voicecall"
---

# `blackcat voicecall`

`voicecall` is a plugin-provided command. It only appears if the voice-call plugin is installed and enabled.

Primary doc:

- Voice-call plugin: [Voice Call](/plugins/voice-call)

## Common commands

```bash
blackcat voicecall status --call-id <id>
blackcat voicecall call --to "+15555550123" --message "Hello" --mode notify
blackcat voicecall continue --call-id <id> --message "Any questions?"
blackcat voicecall end --call-id <id>
```

## Exposing webhooks (Tailscale)

```bash
blackcat voicecall expose --mode serve
blackcat voicecall expose --mode funnel
blackcat voicecall expose --mode off
```

Security note: only expose the webhook endpoint to networks you trust. Prefer Tailscale Serve over Funnel when possible.
