# BlackCat-New — MVP Scope Definition

> CRITICAL: This file is the authority for what is IN and OUT of MVP.
> All PRs/tasks must be checked against this document.
> When in doubt, the answer is OUT.

## MVP — IN (Must have for first working version)

### Channels
- Telegram (grammY)
- Discord (discord.js)
- WhatsApp (Baileys — device relink required, no whatsmeow state migration)

### Core Runtime
- 7-role keyword-priority router (phantom/astrology/wizard/artist/scribe/explorer/oracle)
- Supervisor overlay per role (system prompt, model, provider, temperature, allowed tools)
- Simplified outer channel/session routing (channel → agent, no guild/team/role-based tiers)
- Guardrails pipeline (input / tool / output — all default enabled)
- AES-256-GCM Vault with Argon2id (Go-compatible disk format: {salt,nonce,data} base64 JSON)

### LLM / Providers
- OpenAI, Anthropic, Google Gemini, GitHub Copilot, Ollama (local)
- Fallback chain (config-driven, not hardcoded)

### Persistence
- SQLite sessions (versioned schema)
- SQLite memory store (core + archival, text search MVP)

### Config
- JSON5 format (replaces YAML)
- Zod schema validation
- BLACKCAT_ env var overrides
- blackcat migrate-config CLI command (blackcat.yaml → blackcat.json5)

### Tools (subset)
- exec (shell commands with approval)
- filesystem (read/write limited)
- web (HTTP fetch)
- memory (core + archival)
- usage/status

### Skills
- Local directory loading only
- No remote marketplace

### CLI
- blackcat start / stop / status / doctor / onboard / migrate-config

## MVP — OUT (Explicitly excluded)

### Channels OUT
- Signal, iMessage, BlueBubbles, IRC, MS Teams, Matrix, Feishu, LINE
- Mattermost, Nextcloud Talk, Nostr, Synology Chat, Tlon, Twitch, Zalo, WebChat, Google Chat

### OpenClaw Surfaces OUT
- Full Gateway WebSocket control plane
- Control UI / Dashboard (post-MVP)
- macOS menu bar, iOS/Android node, Voice Wake, Canvas/A2UI

### OpenClaw Features OUT
- Browser control (CDP), canvas.* tools, screen recording/camera
- Cron/scheduler (post-MVP), Webhooks, Gmail Pub/Sub
- Tailscale, Docker sandbox, multi-node
- Multi-agent session spawn, advanced session pruning
- Guild+roles routing, team matching, parent-peer threading (Discord/Slack advanced tiers)
- Full Plugin SDK parity, ClawHub registry, remote extension install
- Full image/video/audio media pipeline, Whisper transcription (post-MVP)
