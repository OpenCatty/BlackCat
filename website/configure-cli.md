---
title: CLI Configure
description: Interactive wizard and flag-mode reference for the blackcat configure command
---

# CLI Configure Command

The `blackcat configure` command provides an interactive wizard for setting up LLM providers, authentication, and model selection.

## Overview

The configure command supports two modes:
- **Interactive mode** — Multi-step wizard with selection prompts (default)
- **Flag mode** — Non-interactive, suitable for scripting and automation

## Usage

```
blackcat configure [flags]
```

### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--provider` | string | Provider to configure (activates non-interactive mode) |
| `--api-key` | string | API key for the provider |
| `--model` | string | Model to use (defaults to provider's default) |

### Interactive Mode

Run without flags to start the wizard:

```bash
blackcat configure
```

**Step 1: Provider Selection**

A multi-select list shows all available providers with their authentication method:

```
Select LLM providers to configure:
  [ ] openai (api-key)
  [ ] anthropic (api-key)
  [ ] copilot (oauth-device)
  [ ] antigravity (oauth-pkce)
  [ ] gemini (api-key)
  [ ] zen (api-key)
  [ ] openrouter (api-key)
  [ ] ollama (none)
```

Use arrow keys and space to select providers, then Enter to continue.

**Step 2: Authentication**

For each selected provider, the wizard prompts for authentication based on the provider type:

- **API key providers** (openai, anthropic, gemini, zen, openrouter): Prompted for API key and model name. The API key is stored in the encrypted vault.
- **OAuth device flow** (copilot): Triggers the GitHub device code flow — displays a URL and code for browser authentication.
- **OAuth PKCE** (antigravity): First asks to accept ToS risk, then opens a browser for Google authentication.
- **No auth** (ollama): Prompted for endpoint URL and model name.

**Step 3: Summary**

After configuring all selected providers, the wizard displays a summary:

```
Configuration Summary
---------------------
  [+] openai
  [+] copilot
  [+] zen

Next steps:
  1. Review config: cat ~/.blackcat/config.yaml
  2. Start the daemon: blackcat daemon
  3. Send a message via Telegram or Discord
```

### Flag Mode

For automation and scripting, use flags instead of the interactive wizard:

```bash
blackcat configure --provider <name> [--api-key <key>] [--model <model>]
```

The `--provider` flag activates non-interactive mode. The behavior depends on the provider's auth method:

**API key providers:**
```bash
# Required: --provider and --api-key
blackcat configure --provider openai --api-key sk-your-key --model gpt-4o
blackcat configure --provider anthropic --api-key sk-ant-your-key
blackcat configure --provider zen --api-key your-zen-key --model opencode/claude-sonnet-4-6
```

**OAuth providers** (still triggers interactive auth flow):
```bash
# --api-key not needed; triggers device flow / PKCE flow
blackcat configure --provider copilot
blackcat configure --provider antigravity
```

**No-auth providers:**
```bash
blackcat configure --provider ollama --model llama3
```

## Configuration Steps in Detail

### 1. LLM Provider Selection

Each provider has a default model that is used if you don't specify one:

| Provider | Default Model | Auth Method |
|----------|---------------|-------------|
| openai | `gpt-4o` | API key |
| anthropic | `claude-3-5-sonnet-20241022` | API key |
| copilot | `gpt-4o` | OAuth device flow |
| antigravity | `gemini-2.5-pro` | OAuth PKCE |
| gemini | `gemini-1.5-pro` | API key |
| zen | `opencode/claude-sonnet-4-6` | API key |
| openrouter | — | API key |
| ollama | `llama3` | None |

### 2. Authentication Setup

**Vault storage:** API keys are stored in the encrypted vault under the key `provider.<name>.apikey`. OAuth tokens are stored under `oauth.<provider>`.

| Provider | Vault Key | What's Stored |
|----------|-----------|---------------|
| openai | `provider.openai.apikey` | API key |
| anthropic | `provider.anthropic.apikey` | API key |
| copilot | `oauth.copilot` | JSON-encoded OAuth token |
| antigravity | `oauth.antigravity` | JSON-encoded OAuth token |
| gemini | `provider.gemini.apikey` | API key |
| zen | `provider.zen.apikey` | API key |
| openrouter | `provider.openrouter.apikey` | API key |
| ollama | — | Nothing (no auth) |

The vault passphrase can be set via:
- `BLACKCAT_VAULT_PASSPHRASE` environment variable
- `--passphrase` flag
- Interactive prompt

### 3. Zen Coding Plan (Optional)

When configuring Zen, you provide your Zen API key and optionally select a model from the curated list. Both `zen.enabled` and `providers.zen.enabled` are set automatically.

### 4. Configuration Review

The configure command updates two things:
1. **Vault** — Stores API keys and OAuth tokens encrypted
2. **Config** — Updates provider enable flags and model selections in `~/.blackcat/config.yaml`

## Examples

### Set up OpenAI as the primary provider

```bash
blackcat configure --provider openai --api-key sk-your-key --model gpt-4o
```

### Authenticate GitHub Copilot

```bash
blackcat configure --provider copilot
# Follow browser instructions to authorize
```

### Set up multiple providers interactively

```bash
blackcat configure
# Select: openai, copilot, zen
# Configure each one in sequence
```

### Switch model for an existing provider

```bash
blackcat configure --provider openai --api-key sk-your-key --model gpt-4o-mini
```

### Set up local Ollama

```bash
blackcat configure --provider ollama --model codellama
```

## See Also

- [Getting Started](/getting-started) — Full setup guide
- [Configuration Reference](/configuration) — Manual configuration
- [OAuth Setup](/oauth) — OAuth authentication details
- [LLM Providers](/providers) — Available providers
