# Zen Coding Plan

The Zen Coding Plan provides curated hosted model access for BlackCat through the OpenCode API. It replicates the OpenCode Zen experience, offering a selection of high-quality models with a simple API key authentication.

## Overview

Zen acts as a managed LLM provider — you get access to a curated list of models without needing separate accounts with each model provider. The API is OpenAI-compatible, making integration straightforward.

Key features:
- **Curated model list** — Pre-selected high-quality models, tested for agent workflows
- **Single API key** — One key for all Zen models
- **OpenAI-compatible** — Uses the standard Chat Completions API format
- **Hosted endpoint** — No infrastructure to manage

## Setup

### 1. Obtain an API Key

Get a Zen API key from your OpenCode account or through the Zen Coding Plan portal.

### 2. Configure via CLI

```bash
blackcat configure --provider zen --api-key your-zen-key --model opencode/claude-sonnet-4-6
```

Or use the interactive wizard:

```bash
blackcat configure
# Select "zen" from the provider list
# Enter your API key when prompted
```

### 3. Configure via YAML

```yaml
zen:
  enabled: true
  apiKey: "your-zen-key"                   # or BLACKCAT_ZEN_APIKEY env var
  baseURL: "https://api.opencode.ai/v1"   # default, usually no need to change

providers:
  zen:
    enabled: true
    model: "opencode/claude-sonnet-4-6"    # pick from available models
```

### Configuration Fields

| Field | Type | Default | Env Variable | Description |
|-------|------|---------|-------------|-------------|
| `zen.enabled` | bool | `false` | `BLACKCAT_ZEN_ENABLED` | Enable Zen provider |
| `zen.apiKey` | string | `""` | `BLACKCAT_ZEN_APIKEY` | Zen API key |
| `zen.baseURL` | string | `"https://api.opencode.ai/v1"` | `BLACKCAT_ZEN_BASEURL` | API endpoint |
| `zen.models` | []string | `[]` | — | Override curated model list |
| `providers.zen.enabled` | bool | `false` | `BLACKCAT_PROVIDERS_ZEN_ENABLED` | Enable Zen as active backend |
| `providers.zen.model` | string | `""` | `BLACKCAT_PROVIDERS_ZEN_MODEL` | Selected model |

### Environment Variables

```bash
export BLACKCAT_ZEN_ENABLED=true
export BLACKCAT_ZEN_APIKEY=your-zen-key
export BLACKCAT_PROVIDERS_ZEN_ENABLED=true
export BLACKCAT_PROVIDERS_ZEN_MODEL=opencode/claude-sonnet-4-6
```

## Available Models

The curated model list provides a balance of capability and cost:

| Model | Description | Best For |
|-------|-------------|----------|
| `opencode/claude-opus-4-6` | Most capable Claude model | Complex reasoning, architecture decisions |
| `opencode/claude-sonnet-4-6` | Balanced performance Claude model | General coding, day-to-day tasks |
| `opencode/gemini-3-pro` | Google Gemini 3 Pro | Large context, multi-modal tasks |

The default model is `opencode/claude-opus-4-6` (first in the curated list). You can select a different model via the `providers.zen.model` config field or the `--model` flag.

> **Note:** The curated list may be updated as new models become available. You can override it with the `zen.models` config field if needed.

## Usage

Once configured, Zen works like any other provider. Start the daemon and it will use Zen for LLM calls:

```bash
blackcat daemon
```

Zen is checked as part of the auto-detection priority order:
1. Copilot (if enabled + OAuth token)
2. Antigravity (if enabled + OAuth token)
3. Gemini (if enabled + API key)
4. **Zen (if enabled + API key)**
5. Fallback to primary `llm.*` config

To make Zen the primary provider, either:
- Disable other providers in the config, or
- Set `llm.provider: "zen"` as the primary

## Billing

Zen uses per-request billing through your OpenCode account. Usage is tracked per API call and billed based on:
- Model used (different models have different rates)
- Input tokens (prompt size)
- Output tokens (completion size)

Check your usage and billing details through the OpenCode dashboard.

## Troubleshooting

### "zen: API key is required"
Set the API key via config, environment variable, or vault:
```bash
export BLACKCAT_ZEN_APIKEY=your-key
```

### "zen: model not in curated list"
The requested model isn't in the default curated list. Either use one of the available models or set `zen.models` to override the list.

### Connection errors
Verify you can reach the Zen endpoint:
```bash
curl -H "Authorization: Bearer your-key" https://api.opencode.ai/v1/models
```

### Slow responses
Model response time varies by complexity and load. Consider using `opencode/claude-sonnet-4-6` for faster responses on routine tasks.

## See Also

- [LLM Providers](./providers.md) — All provider details
- [Configuration Reference](./configuration.md) — Zen config fields
- [Getting Started](./getting-started.md) — Quick start with Zen
