# BlackCat

A Go-based AI agent that orchestrates [OpenCode CLI](https://opencode.ai) via messaging channels. Deploy BlackCat alongside OpenCode on a server, then interact with your development environment through Telegram, Discord, or WhatsApp.

BlackCat receives your natural language requests, processes them through an LLM-powered agent loop, delegates coding tasks to OpenCode, and responds back in your messaging channel — giving you full server control from anywhere.

## Features

- **Multi-channel messaging** — Telegram, Discord, and WhatsApp adapters
- **8 LLM providers** — OpenAI, Anthropic, GitHub Copilot, Antigravity, Google Gemini, Zen, OpenRouter, Ollama
- **OAuth authentication** — Device code flow (Copilot) and PKCE flow (Antigravity)
- **Zen Coding Plan** — Curated hosted models via OpenCode API
- **Interactive setup** — `blackcat configure` wizard for provider setup
- **OpenCode delegation** — Full access to OpenCode CLI for coding tasks
- **MCP support** — Model Context Protocol server/client integration
- **Encrypted vault** — AES-256-GCM encrypted storage for API keys and tokens
- **Memory consolidation** — Persistent agent memory via MEMORY.md
- **Security** — Command deny-list, shell sandboxing, auto-permit controls
- **Docker support** — Docker Compose deployment

## Supported Providers

| Provider | Auth Method | Wire Format | Status |
|----------|------------|-------------|--------|
| OpenAI | API Key | OpenAI | Stable |
| Anthropic | API Key | OpenAI-compat | Stable |
| Google Gemini | API Key | Gemini | Stable |
| GitHub Copilot | OAuth Device Flow | OpenAI-compat | New |
| Antigravity | OAuth PKCE | Gemini | New (ToS Risk) |
| OpenRouter | API Key | OpenAI | Stable |
| Ollama | None (local) | OpenAI | Stable |
| Zen Coding Plan | API Key | OpenAI | New |

## Quick Start

### 1. Install

```bash
go install github.com/startower-observability/blackcat@latest
```

Or build from source:

```bash
git clone https://github.com/startower-observability/blackcat.git
cd blackcat
go build -o blackcat .
```

### 2. Initialize

```bash
blackcat init
```

### 3. Configure a Provider

Interactive wizard:

```bash
blackcat configure
```

Or non-interactive:

```bash
blackcat configure --provider openai --api-key sk-your-key --model gpt-4o
```

For GitHub Copilot (uses your existing subscription):

```bash
blackcat configure --provider copilot
```

### 4. Enable a Channel

Edit `~/.blackcat/config.yaml`:

```yaml
channels:
  telegram:
    enabled: true
    token: "your-bot-token"
```

### 5. Start

```bash
# Ensure OpenCode CLI is running
opencode

# Start BlackCat
blackcat daemon
```

## Deployment

Deploy BlackCat to a Linux VM with a single command:

### Prerequisites

1. Copy the deploy environment template and fill in your VM details:
   ```bash
   cp deploy/deploy.env.example deploy/deploy.env
   $EDITOR deploy/deploy.env
   ```

2. Ensure your SSH key has access to the VM.

### Deploy

```bash
make deploy
```

This single command:
- Pushes your local git changes to the remote
- SSHes into the VM, pulls the latest code, and builds the binary
- Installs the binary to `/usr/local/bin/blackcat`
- Deploys and reloads the `blackcat` and `opencode` systemd services
- Runs a health check to confirm the service is up

### Quick Redeploy (skip git push)

```bash
make deploy-no-push
```

### Health Check Only

```bash
make verify
```

See [`deploy/README.md`](deploy/README.md) for full setup instructions including SSH key configuration and service file details.

## Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](docs/getting-started.md) | Prerequisites, installation, quick start |
| [Configuration](docs/configuration.md) | Full YAML reference, environment variables, examples |
| [LLM Providers](docs/providers.md) | All 8 providers: setup, models, configuration |
| [OAuth Setup](docs/oauth.md) | Copilot device flow and Antigravity PKCE walkthrough |
| [Zen Coding Plan](docs/zen-plan.md) | Curated hosted models, setup, billing |
| [CLI Configure](docs/configure-cli.md) | Interactive wizard and flag-mode reference |
| [Architecture](docs/architecture.md) | How BlackCat works internally |

## Configuration

BlackCat is configured via YAML file (`~/.blackcat/config.yaml`) with environment variable overrides using the `BLACKCAT_` prefix.

See [`blackcat.example.yaml`](blackcat.example.yaml) for a complete example with all fields documented.

Key environment variables:

```bash
BLACKCAT_LLM_PROVIDER=openai
BLACKCAT_LLM_APIKEY=sk-your-key
BLACKCAT_CHANNELS_TELEGRAM_TOKEN=your-bot-token
BLACKCAT_VAULT_PASSPHRASE=your-passphrase
BLACKCAT_ZEN_APIKEY=your-zen-key
```

## Docker

```bash
docker compose up -d
```

See `docker-compose.yml` for the full setup. Requires OpenCode CLI to be accessible on the same network.

## Requirements

- Go 1.25+
- OpenCode CLI running on the same server
- At least one messaging channel configured
- At least one LLM provider configured

> **Note:** WhatsApp support requires CGO for SQLite. Build with `CGO_ENABLED=1`.

## License

See [LICENSE](LICENSE) for details.
