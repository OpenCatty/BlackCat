---
title: blackcat onboard
description: Interactive onboarding wizard for initial configuration
---

# blackcat onboard

The onboard command launches a 4-step wizard to get your BlackCat environment ready for use.

## Usage

```shell
blackcat onboard [flags]
```

## Steps

1. **LLM Provider** — Select and configure your primary AI model.
2. **Messaging Channel** — Choose a channel (Telegram, Discord, or WhatsApp) to interact with the agent.
3. **Daemon Installation** — Install the background service to keep BlackCat running.
4. **Health Check** — Verify that all subsystems are connected and functional.

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--non-interactive` | `false` | Run without interactive prompts (requires other flags) |
| `--install-daemon` | `false` | Automatically install the daemon service |

## Examples

```shell
# Start the interactive wizard
blackcat onboard

# Run with automatic daemon installation
blackcat onboard --install-daemon
```

## Related

- [blackcat configure](/cli/configure)
- [blackcat start](/cli/start)
- [blackcat status](/cli/status)
