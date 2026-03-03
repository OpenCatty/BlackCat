---
title: blackcat doctor
description: Diagnoses and reports environment issues
---

# blackcat doctor

The doctor command identifies common issues in your environment that could prevent BlackCat from running.

## Usage

```shell
blackcat doctor
```

## Description

The doctor command performs several checks on your system:
- **Binary Path:** Verifies BlackCat is correctly installed and accessible.
- **Daemon Status:** Checks if the background service is running.
- **Config Validity:** Validates the contents of your `config.yaml`.
- **Connectivity:** Tests connections to LLM providers and OpenCode.
- **Vault:** Confirms the encrypted vault is accessible with the provided passphrase.

## Examples

```shell
# Run diagnostics on your system
blackcat doctor
```

## Related

- [blackcat status](/cli/status)
- [blackcat health](/cli/health)
- [blackcat onboard](/cli/onboard)
