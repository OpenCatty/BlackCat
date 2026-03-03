---
title: Configuration Reference
description: Complete YAML configuration reference with all fields and environment variables
---

# Configuration Reference

BlackCat uses a hierarchical YAML configuration file with environment variable overrides. Configuration is loaded in this order (later values override earlier ones):

1. **Built-in defaults** — Sensible defaults for all fields
2. **YAML config file** — `~/.blackcat/config.yaml` (or path from `--config` flag)
3. **Environment variables** — `BLACKCAT_` prefix overrides any YAML value

## Configuration File

The default config file location is `~/.blackcat/config.yaml`. You can specify a different path:

```bash
blackcat daemon --config /path/to/config.yaml
```

The `~` prefix is expanded to the user's home directory automatically.

To generate a starter config:

```bash
blackcat init
```

## Full YAML Reference

### Server

HTTP server settings for the health check and API endpoints.

```yaml
server:
  addr: ":8080"    # Listen address (default: ":8080")
  port: 8080       # Listen port (default: 8080)
```

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `server.addr` | string | `":8080"` | `BLACKCAT_SERVER_ADDR` | HTTP listen address |
| `server.port` | int | `8080` | `BLACKCAT_SERVER_PORT` | HTTP listen port |

### OpenCode

Connection settings for the OpenCode CLI agent running on the same server.

```yaml
opencode:
  addr: "http://127.0.0.1:4096"   # OpenCode address (default)
  password: ""                      # Auth password (use env or vault)
  timeout: "30m"                    # Request timeout (default: 30m)
```

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `opencode.addr` | string | `"http://127.0.0.1:4096"` | `BLACKCAT_OPENCODE_ADDR` | OpenCode agent address |
| `opencode.password` | string | `""` | `BLACKCAT_OPENCODE_PASSWORD` | Authentication password |
| `opencode.timeout` | duration | `"30m"` | `BLACKCAT_OPENCODE_TIMEOUT` | Request timeout |

> **Security:** Never put `opencode.password` in plain YAML. Use the `BLACKCAT_OPENCODE_PASSWORD` environment variable or store it in the vault.

### LLM

Primary LLM provider settings. This configures the default provider used by the agent loop.

```yaml
llm:
  provider: "openai"      # Provider name
  model: "gpt-4o"         # Model name
  apiKey: ""               # API key (use env or vault)
  baseURL: ""              # Custom base URL (optional)
  temperature: 0.7         # Sampling temperature (0.0–2.0)
  maxTokens: 4096          # Max tokens per response
```

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `llm.provider` | string | `""` | `BLACKCAT_LLM_PROVIDER` | Provider: `openai`, `anthropic`, `ollama`, `openrouter`, `copilot`, `antigravity`, `gemini`, `zen` |
| `llm.model` | string | `""` | `BLACKCAT_LLM_MODEL` | Model name (e.g., `gpt-4o`, `claude-3-5-sonnet`) |
| `llm.apiKey` | string | `""` | `BLACKCAT_LLM_APIKEY` | API key for the provider |
| `llm.baseURL` | string | `""` | `BLACKCAT_LLM_BASEURL` | Custom API base URL |
| `llm.temperature` | float | `0.7` | `BLACKCAT_LLM_TEMPERATURE` | Sampling temperature |
| `llm.maxTokens` | int | `4096` | `BLACKCAT_LLM_MAXTOKENS` | Maximum tokens per response |

### Channels

Communication channel settings. Enable one or more channels to receive messages.

#### Telegram

```yaml
channels:
  telegram:
    enabled: false
    token: ""    # Bot token from @BotFather
```

| Field | Type | Default | Env Variable |
|-------|------|---------|-------------|
| `channels.telegram.enabled` | bool | `false` | `BLACKCAT_CHANNELS_TELEGRAM_ENABLED` |
| `channels.telegram.token` | string | `""` | `BLACKCAT_CHANNELS_TELEGRAM_TOKEN` |

#### Discord

```yaml
channels:
  discord:
    enabled: false
    token: ""    # Bot token from Discord Developer Portal
```

| Field | Type | Default | Env Variable |
|-------|------|---------|-------------|
| `channels.discord.enabled` | bool | `false` | `BLACKCAT_CHANNELS_DISCORD_ENABLED` |
| `channels.discord.token` | string | `""` | `BLACKCAT_CHANNELS_DISCORD_TOKEN` |

#### WhatsApp

```yaml
channels:
  whatsapp:
    enabled: false
    token: ""    # WhatsApp API credentials
```

| Field | Type | Default | Env Variable |
|-------|------|---------|-------------|
| `channels.whatsapp.enabled` | bool | `false` | `BLACKCAT_CHANNELS_WHATSAPP_ENABLED` |
| `channels.whatsapp.token` | string | `""` | `BLACKCAT_CHANNELS_WHATSAPP_TOKEN` |

> **Note:** WhatsApp support requires CGO for SQLite. Build with `CGO_ENABLED=1`.

### Security

Security settings including the encrypted vault and command deny-list.

```yaml
security:
  vaultPath: "~/.blackcat/vault.json"
  denyPatterns: []
  autoPermit: false
```

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `security.vaultPath` | string | `"~/.blackcat/vault.json"` | `BLACKCAT_SECURITY_VAULTPATH` | Path to AES-256-GCM encrypted vault |
| `security.denyPatterns` | []string | `[]` | — | Shell command patterns to block |
| `security.autoPermit` | bool | `false` | `BLACKCAT_SECURITY_AUTOPERMIT` | Skip confirmation for tool calls |

The vault stores API keys and OAuth tokens encrypted with AES-256-GCM. Set the passphrase via `BLACKCAT_VAULT_PASSPHRASE` or the `--passphrase` flag.

### Memory

Memory consolidation settings for the agent's persistent knowledge.

```yaml
memory:
  filePath: "MEMORY.md"
  consolidationThreshold: 50
```

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `memory.filePath` | string | `"MEMORY.md"` | `BLACKCAT_MEMORY_FILEPATH` | Path to memory file |
| `memory.consolidationThreshold` | int | `50` | `BLACKCAT_MEMORY_CONSOLIDATIONTHRESHOLD` | Entries before consolidation |

### MCP

Model Context Protocol server configurations for extending agent capabilities.

```yaml
mcp:
  servers:
    - name: "my-mcp-server"
      command: "/usr/local/bin/mcp-server"
      args: ["--config", "/etc/mcp/config.yaml"]
      env:
        LOG_LEVEL: "debug"
```

| Field | Type | Description |
|-------|------|-------------|
| `mcp.servers[].name` | string | Server display name |
| `mcp.servers[].command` | string | Executable path |
| `mcp.servers[].args` | []string | Command-line arguments |
| `mcp.servers[].env` | map[string]string | Environment variables |

### Skills

Custom skills directory for extending agent capabilities.

```yaml
skills:
  dir: "skills/"
```

| Field | Type | Default | Env Variable |
|-------|------|---------|-------------|
| `skills.dir` | string | `"skills/"` | `BLACKCAT_SKILLS_DIR` |

### Logging

Logging output configuration.

```yaml
logging:
  level: "info"
  format: "text"
```

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `logging.level` | string | `"info"` | `BLACKCAT_LOGGING_LEVEL` | `debug`, `info`, `warn`, `error` |
| `logging.format` | string | `"text"` | `BLACKCAT_LOGGING_FORMAT` | `text` (human) or `json` (machine) |

### OAuth

OAuth authentication settings for providers that require it. See [OAuth Setup](/oauth) for detailed walkthroughs.

#### Copilot OAuth

```yaml
oauth:
  copilot:
    enabled: false
    clientID: "01ab8ac9400c4e429b23"   # VS Code client ID (default)
```

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `oauth.copilot.enabled` | bool | `false` | `BLACKCAT_OAUTH_COPILOT_ENABLED` | Enable Copilot OAuth |
| `oauth.copilot.clientID` | string | `"01ab8ac9400c4e429b23"` | `BLACKCAT_OAUTH_COPILOT_CLIENTID` | GitHub OAuth app client ID |

#### Antigravity OAuth

```yaml
oauth:
  antigravity:
    enabled: false
    acceptedToS: false
    clientID: "1071006060591-..."      # Default Google client ID
    clientSecret: "GOCSPX-..."         # Default Google client secret
```

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `oauth.antigravity.enabled` | bool | `false` | `BLACKCAT_OAUTH_ANTIGRAVITY_ENABLED` | Enable Antigravity |
| `oauth.antigravity.acceptedToS` | bool | `false` | `BLACKCAT_OAUTH_ANTIGRAVITY_ACCEPTEDTOS` | Accept ToS risk |
| `oauth.antigravity.clientID` | string | *(built-in)* | `BLACKCAT_OAUTH_ANTIGRAVITY_CLIENTID` | Google OAuth client ID |
| `oauth.antigravity.clientSecret` | string | *(built-in)* | `BLACKCAT_OAUTH_ANTIGRAVITY_CLIENTSECRET` | Google OAuth client secret |

> **Warning:** Antigravity uses Google's internal API. You must set `acceptedToS: true` to acknowledge the Terms of Service risk. See [OAuth Setup](/oauth#tos-risk-acknowledgment).

### Zen Coding Plan

Zen provides curated hosted model access through the OpenCode API.

```yaml
zen:
  enabled: false
  apiKey: ""                              # Zen API key
  baseURL: "https://api.opencode.ai/v1"  # Default endpoint
  models: []                              # Override curated model list
```

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `zen.enabled` | bool | `false` | `BLACKCAT_ZEN_ENABLED` | Enable Zen provider |
| `zen.apiKey` | string | `""` | `BLACKCAT_ZEN_APIKEY` | Zen API key |
| `zen.baseURL` | string | `"https://api.opencode.ai/v1"` | `BLACKCAT_ZEN_BASEURL` | API base URL |
| `zen.models` | []string | `[]` | — | Override curated model list |

See [Zen Coding Plan](/zen-plan) for setup details and available models.

### Providers

Per-provider enable/model settings for the new backend providers (Copilot, Antigravity, Gemini, Zen). These work alongside the primary `llm.*` settings.

#### Copilot Provider

```yaml
providers:
  copilot:
    enabled: false
    model: "gpt-4o"
```

| Field | Type | Default | Env Variable |
|-------|------|---------|-------------|
| `providers.copilot.enabled` | bool | `false` | `BLACKCAT_PROVIDERS_COPILOT_ENABLED` |
| `providers.copilot.model` | string | `""` | `BLACKCAT_PROVIDERS_COPILOT_MODEL` |

#### Antigravity Provider

```yaml
providers:
  antigravity:
    enabled: false
    model: "gemini-2.5-pro"
```

| Field | Type | Default | Env Variable |
|-------|------|---------|-------------|
| `providers.antigravity.enabled` | bool | `false` | `BLACKCAT_PROVIDERS_ANTIGRAVITY_ENABLED` |
| `providers.antigravity.model` | string | `""` | `BLACKCAT_PROVIDERS_ANTIGRAVITY_MODEL` |

#### Gemini Provider

```yaml
providers:
  gemini:
    enabled: false
    model: "gemini-1.5-pro"
    apiKey: ""
```

| Field | Type | Default | Env Variable |
|-------|------|---------|-------------|
| `providers.gemini.enabled` | bool | `false` | `BLACKCAT_PROVIDERS_GEMINI_ENABLED` |
| `providers.gemini.model` | string | `""` | `BLACKCAT_PROVIDERS_GEMINI_MODEL` |
| `providers.gemini.apiKey` | string | `""` | `BLACKCAT_PROVIDERS_GEMINI_APIKEY` |

#### Zen Provider

```yaml
providers:
  zen:
    enabled: false
    model: "opencode/claude-opus-4-6"
```

| Field | Type | Default | Env Variable |
|-------|------|---------|-------------|
| `providers.zen.enabled` | bool | `false` | `BLACKCAT_PROVIDERS_ZEN_ENABLED` |
| `providers.zen.model` | string | `""` | `BLACKCAT_PROVIDERS_ZEN_MODEL` |

## Environment Variables

All configuration values can be overridden using environment variables with the `BLACKCAT_` prefix. The naming convention maps YAML paths to `UPPER_SNAKE_CASE`:

```
yaml path              → environment variable
─────────────────────────────────────────────
server.addr            → BLACKCAT_SERVER_ADDR
llm.provider           → BLACKCAT_LLM_PROVIDER
llm.apiKey             → BLACKCAT_LLM_APIKEY
channels.telegram.token→ BLACKCAT_CHANNELS_TELEGRAM_TOKEN
oauth.copilot.enabled  → BLACKCAT_OAUTH_COPILOT_ENABLED
zen.apiKey             → BLACKCAT_ZEN_APIKEY
```

Full list of supported environment variables:

| Variable | Section |
|----------|---------|
| `BLACKCAT_SERVER_ADDR` | Server |
| `BLACKCAT_SERVER_PORT` | Server |
| `BLACKCAT_OPENCODE_ADDR` | OpenCode |
| `BLACKCAT_OPENCODE_PASSWORD` | OpenCode |
| `BLACKCAT_OPENCODE_TIMEOUT` | OpenCode |
| `BLACKCAT_LLM_PROVIDER` | LLM |
| `BLACKCAT_LLM_MODEL` | LLM |
| `BLACKCAT_LLM_APIKEY` | LLM |
| `BLACKCAT_LLM_BASEURL` | LLM |
| `BLACKCAT_LLM_TEMPERATURE` | LLM |
| `BLACKCAT_LLM_MAXTOKENS` | LLM |
| `BLACKCAT_CHANNELS_TELEGRAM_ENABLED` | Channels |
| `BLACKCAT_CHANNELS_TELEGRAM_TOKEN` | Channels |
| `BLACKCAT_CHANNELS_DISCORD_ENABLED` | Channels |
| `BLACKCAT_CHANNELS_DISCORD_TOKEN` | Channels |
| `BLACKCAT_CHANNELS_WHATSAPP_ENABLED` | Channels |
| `BLACKCAT_CHANNELS_WHATSAPP_TOKEN` | Channels |
| `BLACKCAT_SECURITY_VAULTPATH` | Security |
| `BLACKCAT_SECURITY_AUTOPERMIT` | Security |
| `BLACKCAT_MEMORY_FILEPATH` | Memory |
| `BLACKCAT_MEMORY_CONSOLIDATIONTHRESHOLD` | Memory |
| `BLACKCAT_SKILLS_DIR` | Skills |
| `BLACKCAT_LOGGING_LEVEL` | Logging |
| `BLACKCAT_LOGGING_FORMAT` | Logging |
| `BLACKCAT_OAUTH_COPILOT_ENABLED` | OAuth |
| `BLACKCAT_OAUTH_COPILOT_CLIENTID` | OAuth |
| `BLACKCAT_OAUTH_ANTIGRAVITY_ENABLED` | OAuth |
| `BLACKCAT_OAUTH_ANTIGRAVITY_ACCEPTEDTOS` | OAuth |
| `BLACKCAT_OAUTH_ANTIGRAVITY_CLIENTID` | OAuth |
| `BLACKCAT_OAUTH_ANTIGRAVITY_CLIENTSECRET` | OAuth |
| `BLACKCAT_ZEN_ENABLED` | Zen |
| `BLACKCAT_ZEN_APIKEY` | Zen |
| `BLACKCAT_ZEN_BASEURL` | Zen |
| `BLACKCAT_PROVIDERS_COPILOT_ENABLED` | Providers |
| `BLACKCAT_PROVIDERS_COPILOT_MODEL` | Providers |
| `BLACKCAT_PROVIDERS_ANTIGRAVITY_ENABLED` | Providers |
| `BLACKCAT_PROVIDERS_ANTIGRAVITY_MODEL` | Providers |
| `BLACKCAT_PROVIDERS_GEMINI_ENABLED` | Providers |
| `BLACKCAT_PROVIDERS_GEMINI_MODEL` | Providers |
| `BLACKCAT_PROVIDERS_GEMINI_APIKEY` | Providers |
| `BLACKCAT_PROVIDERS_ZEN_ENABLED` | Providers |
| `BLACKCAT_PROVIDERS_ZEN_MODEL` | Providers |

## Example Configurations

### Minimal (OpenAI only)

```yaml
llm:
  provider: "openai"
  model: "gpt-4o"
  apiKey: "sk-your-key"
channels:
  telegram:
    enabled: true
    token: "your-bot-token"
```

### Multi-Provider with OAuth

```yaml
llm:
  provider: "openai"
  model: "gpt-4o"
oauth:
  copilot:
    enabled: true
  antigravity:
    enabled: true
    acceptedToS: true
providers:
  copilot:
    enabled: true
    model: "gpt-4o"
  antigravity:
    enabled: true
    model: "gemini-2.5-pro"
channels:
  telegram:
    enabled: true
  discord:
    enabled: true
```

### Zen Coding Plan

```yaml
zen:
  enabled: true
  apiKey: "your-zen-key"
providers:
  zen:
    enabled: true
    model: "opencode/claude-sonnet-4-6"
channels:
  telegram:
    enabled: true
    token: "your-bot-token"
```

### Local Ollama

```yaml
llm:
  provider: "ollama"
  model: "llama3"
  baseURL: "http://localhost:11434/v1"
channels:
  telegram:
    enabled: true
    token: "your-bot-token"
```

For the complete example with all fields, see [`blackcat.example.yaml`](../blackcat.example.yaml).

### Dashboard

Web-based monitoring dashboard settings.

```yaml
dashboard:
  enabled: false
  addr: ":8081"
  token: ""
```

| Field | YAML key | Default | Description |
|-------|----------|---------|-------------|
| Enabled | `enabled` | `false` | Enable web dashboard |
| Addr | `addr` | `":8081"` | Listen address |
| Token | `token` | `""` | Bearer auth token |

### Scheduler

Cron-based task scheduling settings.

```yaml
scheduler:
  enabled: false
  jobs: []
```

| Field | YAML key | Default | Description |
|-------|----------|---------|-------------|
| Enabled | `enabled` | `false` | Enable cron scheduler |
| Jobs | `jobs` | `[]` | List of scheduled jobs |

### Orchestrator

Parallel sub-agent execution settings.

```yaml
orchestrator:
  max_concurrent: 5
  sub_agent_timeout: "5m"
```

| Field | YAML key | Default | Description |
|-------|----------|---------|-------------|
| MaxConcurrent | `max_concurrent` | `5` | Max parallel sub-agents (hard cap: 10) |
| SubAgentTimeout | `sub_agent_timeout` | `"5m"` | Per-sub-agent timeout |

### Session

Conversation history and session management settings.

```yaml
session:
  enabled: false
  store_dir: "~/.blackcat/sessions"
  max_history: 50
```

| Field | YAML key | Default | Description |
|-------|----------|---------|-------------|
| Enabled | `enabled` | `false` | Enable conversation history |
| StoreDir | `store_dir` | `"~/.blackcat/sessions"` | Storage directory |
| MaxHistory | `max_history` | `50` | Max messages per session |

### Rules

Conditional rule system settings.

```yaml
rules:
  dir: "rules/"
```

| Field | YAML key | Description |
|-------|----------|-------------|
| Dir | `dir` | Rules `.md` files directory |

### Profiles

Custom agent profile settings.
,
```yaml
profiles:
  dir: "profiles/"
```

| Field | YAML key | Description |
|-------|----------|-------------|
| Dir | `dir` | Profiles `.md` files directory |
],op:
## See Also

- [Getting Started](/getting-started) — Quick start guide
- [LLM Providers](/providers) — Provider-specific setup
- [OAuth Setup](/oauth) — OAuth authentication details
- [CLI Configure](/configure-cli) — Interactive configuration wizard
