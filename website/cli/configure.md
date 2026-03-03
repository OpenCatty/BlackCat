---
title: blackcat configure
description: Interactive wizard for LLM provider setup
---

# blackcat configure

The configure command provides an interactive terminal UI (TUI) to set up LLM providers and their authentication details.

## Usage

```shell
blackcat configure [flags]
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--provider` | — | The name of the LLM provider (e.g., `openai`, `anthropic`, `copilot`) |
| `--api-key` | — | API key for the provider |
| `--model` | — | Model ID to use (e.g., `gpt-4o`, `claude-3-5-sonnet`) |
| `--base-url` | — | Optional base URL for the API endpoint |

## Examples

```shell
# Launch the interactive provider wizard
blackcat configure

# Set up OpenAI with flags
blackcat configure --provider openai --api-key sk-your-key --model gpt-4o

# Set up GitHub Copilot via device flow
blackcat configure --provider copilot
```

## Related

- [blackcat onboard](/cli/onboard)
- [blackcat vault](/cli/vault)
- [LLM Providers](/providers)
