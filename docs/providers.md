# LLM Providers

BlackCat supports multiple LLM providers for agent orchestration. You can use API-key-based providers, OAuth-authenticated providers, or local models.

## Provider Overview

| Provider | Auth Method | Wire Format | Default Model | Status |
|----------|------------|-------------|---------------|--------|
| OpenAI | API Key | OpenAI | `gpt-5.2` | Stable |
| Anthropic | API Key | OpenAI-compat | `claude-sonnet-4-6` | Stable |
| Google Gemini | API Key | Gemini | `gemini-2.5-flash` | Stable |
| GitHub Copilot | OAuth Device Flow | OpenAI-compat | `gpt-4.1` | New |
| Antigravity | OAuth PKCE | Gemini | `gemini-2.5-pro` | New (ToS Risk) |
| OpenRouter | API Key | OpenAI | — | Stable |
| Ollama | None | OpenAI | `llama3.3` | Stable |
| Zen Coding Plan | API Key | OpenAI | `opencode/claude-sonnet-4-6` | New |

## OpenAI

The default provider. Uses the standard OpenAI Chat Completions API.

**Setup:**
```bash
blackcat configure --provider openai --api-key sk-your-key --model gpt-5.2
```

**YAML configuration:**
```yaml
llm:
  provider: "openai"
  model: "gpt-5.2"
  apiKey: "sk-..."    # or use BLACKCAT_LLM_APIKEY env var
  temperature: 0.7
  maxTokens: 4096
```

**Supported models:** Any model available through the OpenAI API (e.g., `gpt-5.2`, `gpt-4.1`, `gpt-4.1-mini`, `o3`, `o4-mini`, `gpt-4o`, `gpt-4o-mini`).

## Anthropic

Anthropic Claude models via an OpenAI-compatible API endpoint. Configure `baseURL` if using a proxy or direct Anthropic API.

**Setup:**
```bash
blackcat configure --provider anthropic --api-key sk-ant-your-key --model claude-sonnet-4-6
```

**YAML configuration:**
```yaml
llm:
  provider: "anthropic"
  model: "claude-sonnet-4-6"
  apiKey: "sk-ant-..."
  baseURL: "https://api.anthropic.com/v1"  # optional
```

**Supported models:** `claude-opus-4-6`, `claude-sonnet-4-6`, `claude-haiku-4-5`, and other Anthropic models.

## Google Gemini (Official)

Direct access to Google's Gemini models using the official Gemini API with API key authentication.

**Setup:**
```bash
blackcat configure --provider gemini --api-key your-google-api-key --model gemini-2.5-flash
```

**YAML configuration:**
```yaml
providers:
  gemini:
    enabled: true
    model: "gemini-2.5-flash"
    apiKey: "your-google-api-key"    # or BLACKCAT_PROVIDERS_GEMINI_APIKEY
```

**Supported models:** `gemini-2.5-pro`, `gemini-2.5-flash`, `gemini-3.1-pro-preview`, `gemini-3-flash-preview`, and other models available through the Gemini API.

> **Note:** Gemini uses its own wire format (not OpenAI-compatible). BlackCat handles the format conversion automatically via the built-in Gemini codec.

## GitHub Copilot

Uses your existing GitHub Copilot subscription as an LLM provider. Authentication is via OAuth device code flow (RFC 8628) — you authorize through GitHub in your browser.

**Architecture:**
- OAuth token (long-lived) → exchanged for Copilot API token (~30 min TTL, auto-refreshed)
- Chat endpoint: `api.githubcopilot.com/chat/completions` (OpenAI-compatible)
- Required headers: `User-Agent: GitHubCopilotChat/0.37.5`, `Editor-Version: vscode/1.109.2`, `Copilot-Integration-Id: vscode-chat`

**Setup:**
```bash
blackcat configure --provider copilot
```

This triggers the device code flow:
1. BlackCat displays a URL and user code
2. Open the URL in your browser and enter the code
3. Authorize the GitHub OAuth application
4. Token is automatically saved to the encrypted vault

**YAML configuration:**
```yaml
oauth:
  copilot:
    enabled: true
    clientID: "01ab8ac9400c4e429b23"   # default VS Code client ID
providers:
  copilot:
    enabled: true
    model: "gpt-4.1"
```

**Supported models:** Models available through your Copilot subscription (typically `gpt-4.1`, `gpt-4o`, `gpt-4o-mini`).

See [OAuth Setup](./oauth.md#github-copilot--device-code-flow) for detailed authentication guide.

## Antigravity (Google IDE)

Antigravity provides access to Google's Gemini models through the internal Google IDE API (cloudcode). Uses browser-based PKCE OAuth flow.

**Setup:**
```bash
blackcat configure --provider antigravity
```

You will be prompted to accept the Terms of Service risk before proceeding.

**YAML configuration:**
```yaml
oauth:
  antigravity:
    enabled: true
    acceptedToS: true    # REQUIRED: acknowledge ToS risk
    clientID: "..."      # built-in default
    clientSecret: "..."  # built-in default
providers:
  antigravity:
    enabled: true
    model: "gemini-2.5-pro"
```

**Supported models:** Gemini models available through the cloudcode API (e.g., `gemini-2.5-pro`, `gemini-2.5-flash`).

### ToS Risk Warning

Antigravity uses Google's internal cloudcode API (`cloudcode-pa.googleapis.com`), which is intended for Google's own IDE products. **Using this API from third-party applications may violate Google's Terms of Service.** Google has been actively blocking unauthorized third-party access since February 2026.

By setting `acceptedToS: true`, you acknowledge:
- This may stop working at any time without notice
- Your Google account could potentially be flagged
- This is NOT an officially supported integration
- Use at your own risk

See [OAuth Setup](./oauth.md#tos-risk-acknowledgment) for details.

## OpenRouter

OpenRouter provides access to multiple LLM providers through a single API key, with usage-based pricing.

**Setup:**
```bash
blackcat configure --provider openrouter --api-key sk-or-your-key --model anthropic/claude-3.5-sonnet
```

**YAML configuration:**
```yaml
llm:
  provider: "openrouter"
  model: "anthropic/claude-3.5-sonnet"
  apiKey: "sk-or-..."
  baseURL: "https://openrouter.ai/api/v1"   # optional, auto-detected
```

**Supported models:** Any model available on [OpenRouter](https://openrouter.ai/models). Use the `provider/model` format.

## Ollama

Run LLM models locally using Ollama. No API key required — the model runs on your own hardware.

**Setup:**
1. Install Ollama: [ollama.com](https://ollama.com)
2. Pull a model: `ollama pull llama3.3`
3. Configure BlackCat:

```bash
blackcat configure --provider ollama --model llama3.3
```

**YAML configuration:**
```yaml
llm:
  provider: "ollama"
  model: "llama3.3"
  baseURL: "http://localhost:11434/v1"
```

**Supported models:** Any model available in your Ollama installation. Run `ollama list` to see installed models.

## Zen Coding Plan

Curated hosted model access through the OpenCode API. Provides a selection of high-quality models with per-request billing.

**Setup:**
```bash
blackcat configure --provider zen --api-key your-zen-key --model opencode/claude-sonnet-4-6
```

**YAML configuration:**
```yaml
zen:
  enabled: true
  apiKey: "your-zen-key"
  baseURL: "https://api.opencode.ai/v1"   # default
providers:
  zen:
    enabled: true
    model: "opencode/claude-sonnet-4-6"
```

**Curated models:**
- `opencode/claude-opus-4-6`
- `opencode/claude-sonnet-4-6`
- `opencode/gemini-3.1-pro`

See [Zen Coding Plan](./zen-plan.md) for details.

## Auto-Detection

When the daemon starts, BlackCat attempts to create an active backend in this priority order:

1. **Copilot** — if `providers.copilot.enabled` and OAuth token exists in vault
2. **Antigravity** — if `providers.antigravity.enabled` and OAuth token exists in vault
3. **Gemini** — if `providers.gemini.enabled` and API key is configured
4. **Zen** — if `providers.zen.enabled` and API key is configured

If none of the above are available, it falls back to the primary `llm.*` configuration (OpenAI, Anthropic, Ollama, OpenRouter, etc.).

You can always override the provider explicitly using `blackcat configure` or by editing the YAML config.

## See Also

- [Configuration Reference](./configuration.md) — Full config reference
- [OAuth Setup](./oauth.md) — OAuth flow details
- [Zen Coding Plan](./zen-plan.md) — Zen-specific guide
