# BlackCat — AI Operations Guide

You are an AI agent operating or maintaining a BlackCat instance.
This document explains how BlackCat works and how to manage it.

## Architecture

```
Message -> Daemon -> Supervisor -> ClassifyMessage -> Role -> Subagent -> LLM -> Response
```

Components:
- **Daemon** (`blackcat start`): Background service, initializes all components
- **Supervisor** (`src/agent/supervisor.ts`): Orchestrates message flow
- **Router** (`src/blackcat/router.ts`): Keyword-based role classification
- **Role workspace**: AGENTS.md (directives) + SOUL.md (personality) + shared skills

## How Message Routing Works

1. User sends a message via Telegram, Discord, or WhatsApp
2. The daemon receives the message via the channel connector
3. The supervisor calls `ClassifyMessage(message, roles)` from `src/blackcat/router.ts`
4. The router scans the message text for keywords, matching the highest-priority role
5. The matched role's workspace loads: AGENTS.md + SOUL.md + applicable skills
6. The subagent generates a response using the configured LLM
7. The response is sent back through the originating channel

If no keywords match, the `oracle` role (priority 100) handles the message.

## The 7 Roles

| Role | Priority | Purpose | Keywords (selection) |
|------|----------|---------|---------------------|
| phantom | 10 | Infrastructure & DevOps | restart, deploy, server, docker, k8s |
| astrology | 20 | Crypto & Web3 | crypto, bitcoin, eth, trading, defi |
| wizard | 30 | Software Engineering | code, bug, fix, typescript, python |
| artist | 40 | Social Media Content | instagram, tiktok, post, content |
| scribe | 50 | Writing & Documentation | write, draft, article, blog |
| explorer | 60 | Research & Information | search, research, web, browse |
| oracle | 100 | Fallback — general assistant | (none) |

Full role reference: [roles.md](roles.md)

## Managing Roles

### View current roles
Roles are defined in `blackcat.example.json5` under `agents.list`.

### Role workspace structure
```
workspaces/<role>/
├── AGENTS.md    # Role-specific directives and behavior rules
└── SOUL.md      # Personality and communication style
```

### Add a new role
1. Add entry to `agents.list` in your config with: `id`, `name`, `workspace`, `model`, `keywords`
2. Create `workspaces/<new-role>/AGENTS.md` and `workspaces/<new-role>/SOUL.md`
3. Update `DEFAULT_ROLES` in `src/blackcat/router.ts` if adding a new default
4. Run: `node node_modules/vitest/vitest.mjs run src/blackcat/` — 17/17 must stay green

## Managing Skills

### Skills format
Skills are Markdown files in subdirectory format:
```
workspaces/shared-skills/<skill-name>/SKILL.md
```
⚠️ Flat `.md` files in the root of shared-skills are IGNORED by the loader.

### Skills SKILL.md frontmatter
```yaml
---
name: Skill Display Name
version: v1.0.0
tags: [tag1, tag2]
requires:             # optional
  bins: [python3]    # required binaries
  env: [API_KEY]     # required env vars
---
```

### Add a new skill
1. Create `workspaces/shared-skills/<name>/SKILL.md` with frontmatter + content
2. Skills auto-load via `skills.load.extraDirs: ["./workspaces/shared-skills"]` in config
3. No restart needed — skills load at subagent creation time

Full skills reference: [skills.md](skills.md)

## Common Management Operations

```bash
# Check daemon status
blackcat status

# Restart after config change
blackcat restart

# Health check
blackcat health

# Diagnose issues
blackcat doctor

# Add a channel
blackcat channels add --channel telegram --token <TOKEN>
blackcat channels add --channel discord --token <TOKEN>

# WhatsApp login (interactive QR scan)
blackcat channels login --channel whatsapp

# List channels
blackcat channels list

# Check channel health
blackcat channels status

# Stop daemon
blackcat stop

# Uninstall daemon
blackcat uninstall --yes
```

## Configuration

Config location: `~/.blackcat/config.json5`
Format: JSON5 (supports comments, trailing commas — NOT YAML)

See full reference: [configuration.md](configuration.md)

## Troubleshooting

| Issue | Diagnosis | Solution |
|-------|-----------|----------|
| Daemon won't start | `blackcat doctor` | Check logs, verify config |
| Messages not routing | Check router keywords | Verify message contains expected keywords |
| WhatsApp not working | `CGO_ENABLED=1` missing | Set env var before starting |
| Channel offline | `blackcat channels status` | Re-authenticate with `channels login` |
| Wrong role triggered | Check keyword match | Edit config keywords or message text |

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `BLACKCAT_PINCHTAB_ENABLED` | Enable PinchTab web browsing |
| `BLACKCAT_PINCHTAB_BASE_URL` | PinchTab API base URL |
| `BLACKCAT_PINCHTAB_TOKEN` | PinchTab authentication token |
| `GEMINI_API_KEY` | Google Gemini (for veo3-video-gen, nano-banana-pro skills) |
| `CGO_ENABLED=1` | Required for WhatsApp SQLite |
