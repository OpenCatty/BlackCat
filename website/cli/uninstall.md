---
title: blackcat uninstall
description: Remove BlackCat daemon and config
---

# blackcat uninstall

The uninstall command cleanly removes the background daemon service and its associated configuration.

## Usage

```shell
blackcat uninstall [flags]
```

## Description

The uninstall command handles the safe removal of BlackCat from your system:
- **Service Removal:** Stops and uninstalls the background daemon service.
- **Config Removal:** Optional deletion of the `~/.blackcat` directory.

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--yes` | `false` | Skip confirmation prompt |
| `--purge` | `false` | Also delete the configuration and vault data |

## Examples

```shell
# Remove the daemon and configuration
blackcat uninstall --yes --purge
```

## Related

- [blackcat stop](/cli/stop)
- [blackcat installation](/installation)
