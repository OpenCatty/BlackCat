---
title: LLM Providers
description: Overview of supported LLM providers
---

# LLM Providers

BlackCat supports multiple LLM providers for agent orchestration. You can use API-key-based providers, OAuth-authenticated providers, or local models.

## Provider Overview

| Provider | Auth Method | Wire Format | Default Model | Status |
|----------|------------|-------------|---------------|--------|
| [OpenAI](/providers/openai) | API Key | OpenAI | `gpt-5.2` | Stable |
| [Anthropic](/providers/anthropic) | API Key | OpenAI-compat | `claude-sonnet-4-6` | Stable |
| [Google Gemini](/providers/gemini) | API Key | Gemini | `gemini-2.5-flash` | Stable |
| [GitHub Copilot](/providers/copilot) | OAuth Device Flow | OpenAI-compat | `gpt-4.1` | New |
| [Antigravity](/providers/antigravity) | OAuth PKCE | Gemini | `gemini-2.5-pro` | New (ToS Risk) |
| [OpenRouter](/providers/openrouter) | API Key | OpenAI | — | Stable |
| [Ollama](/providers/ollama) | None | OpenAI | `llama3.3` | Stable |
| [Zen Coding Plan](/providers/zen) | API Key | OpenAI | `opencode/claude-sonnet-4-6` | New |

## Auto-Detection

When the daemon starts, BlackCat attempts to create an active backend in this priority order:

1. **Copilot** — if `providers.copilot.enabled` and OAuth token exists in vault
2. **Antigravity** — if `providers.antigravity.enabled` and OAuth token exists in vault
3. **Gemini** — if `providers.gemini.enabled` and API key is configured
4. **Zen** — if `providers.zen.enabled` and API key is configured

If none of the above are available, it falls back to the primary `llm.*` configuration (OpenAI, Anthropic, Ollama, OpenRouter, etc.).

You can always override the provider explicitly using `blackcat configure` or by editing the YAML config.

## Related

- [blackcat configure](/cli/configure)
- [blackcat vault](/cli/vault)
