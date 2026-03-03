---
title: blackcat start / stop / restart
description: Manage the background daemon service
---

# blackcat start / stop / restart

These commands control the background daemon service that keeps BlackCat running and listening for messages.

## Usage

```shell
blackcat start [flags]
blackcat stop
blackcat restart
```

## Description

The start command installs and starts the daemon service in the background. It uses system-native service managers:
- **Linux:** systemd --user
- **macOS:** LaunchAgent
- **Windows:** Scheduled Tasks (schtasks)

The stop command halts the running daemon, and the restart command performs a stop followed by a start, which is useful after making manual changes to your `config.yaml`.

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `~/.blackcat/config.yaml` | Path to the configuration file |

## Examples

```shell
# Install and start the daemon
blackcat start

# Stop the running daemon
blackcat stop

# Restart the daemon after config changes
blackcat restart
```

## Related

- [blackcat status](/cli/status)
- [blackcat onboard](/cli/onboard)
