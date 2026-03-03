---
title: blackcat vault
description: Manages encrypted secrets storage
---

# blackcat vault

The vault command provides a secure, encrypted storage for API keys and tokens using AES-256-GCM encryption.

## Usage

```shell
blackcat vault [sub-command] [flags]
```

## Description

The vault command stores and retrieves sensitive information for use by the BlackCat daemon. It uses a master passphrase for encryption.

## Sub-commands

| Sub-command | Description |
|------|-------------|
| `set` | Save a new key-value secret to the vault |
| `get` | Retrieve the value of a specific secret |
| `list` | View all keys stored in the vault (values are masked) |
| `delete` | Remove a secret from the vault |

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--passphrase` | — | Master passphrase for vault encryption |

## Examples

```shell
# Set an OpenAI API key
blackcat vault set openai_api_key sk-your-key

# Get a secret from the vault
blackcat vault get openai_api_key

# List all stored secrets
blackcat vault list

# Delete a secret
blackcat vault delete openai_api_key
```

## Related

- [blackcat configure](/cli/configure)
- [blackcat onboard](/cli/onboard)
