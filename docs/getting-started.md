# Getting Started

Welcome to BlackCat — a Go-based AI agent that orchestrates OpenCode CLI via messaging channels (Telegram, Discord, WhatsApp). BlackCat gives you full server control through natural language conversations.

## Prerequisites

- **Go 1.25+** — BlackCat is written in Go and requires Go 1.25 or later
- **OpenCode CLI** — Must be installed and running on the same server ([opencode.ai](https://opencode.ai))
- **Operating System** — Linux, macOS, or Windows
- **Network** — Outbound HTTPS access for LLM API calls; inbound access if using webhook-based channels

> **Note:** WhatsApp support requires CGO for SQLite. Set `CGO_ENABLED=1` when building with WhatsApp support.

## Installation

### From Source

```bash
go install github.com/startower-observability/blackcat@latest
```

Or clone and build manually:

```bash
git clone https://github.com/startower-observability/blackcat.git
cd blackcat
go build -o blackcat .
```

### Docker

```bash
docker compose up -d
```

See `docker-compose.yml` in the repository root for the full configuration. The Docker setup includes BlackCat, and expects OpenCode CLI to be accessible on the same network.

## Quick Start

### 1. Initialize Configuration

```bash
blackcat init
```

This creates the default configuration directory at `~/.blackcat/` with:
- `config.yaml` — Main configuration file
- `vault.json` — Encrypted secrets vault

You will be prompted to set a vault passphrase for encrypting API keys and tokens.

### 2. Configure an LLM Provider

The fastest way to configure a provider is the interactive wizard:

```bash
blackcat configure
```

This launches a multi-select wizard where you can choose one or more providers and authenticate them. For example, to set up OpenAI:

```bash
blackcat configure --provider openai --api-key sk-your-key --model gpt-4o
```

Or to authenticate GitHub Copilot via device flow:

```bash
blackcat configure --provider copilot
```

See [CLI Configure](./configure-cli.md) for the full command reference.

### 3. Connect a Channel

Edit `~/.blackcat/config.yaml` to enable at least one messaging channel:

**Telegram:**
```yaml
channels:
  telegram:
    enabled: true
    token: "your-telegram-bot-token"
```

**Discord:**
```yaml
channels:
  discord:
    enabled: true
    token: "your-discord-bot-token"
```

**WhatsApp:**
```yaml
channels:
  whatsapp:
    enabled: true
    token: "your-whatsapp-token"
```

> **Tip:** Store sensitive tokens in the vault instead of plain YAML. Use `BLACKCAT_CHANNELS_TELEGRAM_TOKEN` environment variable or save to vault during `blackcat init`.

### 4. Start the Agent

Make sure OpenCode CLI is running first:

```bash
opencode
```

Then start the BlackCat daemon:

```bash
blackcat daemon
```

The daemon will:
1. Load configuration from `~/.blackcat/config.yaml`
2. Connect to the configured LLM provider
3. Start enabled channel adapters (Telegram, Discord, WhatsApp)
4. Begin listening for incoming messages

Verify the agent is running:

```bash
blackcat health
```

### 5. Send Your First Message

Open your configured channel (e.g., Telegram) and send a message to your bot:

```
Hello, can you list the files in the current project?
```

BlackCat will process your request through the agent loop, potentially delegating to OpenCode CLI for coding tasks, and respond in the same channel.

## Troubleshooting

### "vault passphrase required"
Set the `BLACKCAT_VAULT_PASSPHRASE` environment variable or pass `--passphrase` flag.

### "connection refused" to OpenCode
Ensure OpenCode CLI is running and accessible at the address in `opencode.addr` (default: `http://127.0.0.1:4096`).

### "unauthorized" from LLM provider
Verify your API key is correctly set. Run `blackcat configure --provider <name>` to reconfigure.

### WhatsApp build errors
WhatsApp requires CGO. Build with: `CGO_ENABLED=1 go build -o blackcat .`

## Next Steps

## Phase 3 Features

BlackCat Phase 3 introduces advanced management and extensibility features.

### 1. Enable Dashboard
The dashboard provides a web-based UI for monitoring subsystems and scheduled tasks.

Set in `~/.blackcat/config.yaml`:
```yaml
dashboard:
  enabled: true
  token: "your-secret-token"
```

### 2. Access Dashboard
Open `http://localhost:8081/dashboard/` and provide your token in the `Authorization` header:
```bash
curl -H "Authorization: Bearer your-secret-token" http://localhost:8081/dashboard/
```

### 3. Enable Session History
Keep track of user conversations across restarts:
```yaml
session:
  enabled: true
```

### 4. Configure Rules
Add custom behaviors based on file patterns. Create `.md` files with YAML frontmatter in your rules directory:
```yaml
---
name: go-formatting
globs: ["**/*.go"]
---
Always ensure Go files follow standard formatting...
```

- [Configuration Reference](./configuration.md) — Full YAML config guide
- [LLM Providers](./providers.md) — All supported providers
- [OAuth Setup](./oauth.md) — GitHub Copilot and Antigravity auth
- [Zen Coding Plan](./zen-plan.md) — Hosted model access
- [CLI Configure](./configure-cli.md) — Interactive setup wizard
- [Architecture](./architecture.md) — How BlackCat works
