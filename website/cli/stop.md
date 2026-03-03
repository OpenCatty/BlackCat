---
title: blackcat stop
description: Stop the running BlackCat daemon
---

# blackcat stop

The stop command halts the running BlackCat background daemon service.

## Usage

```shell
blackcat stop
```

## Description

The stop command sends a termination signal to the background service managed by your operating system:
- **Linux:** `systemctl --user stop blackcat`
- **macOS:** `launchctl unload ...`
- **Windows:** `schtasks /end ...`

## Examples

```shell
# Stop the running daemon
blackcat stop
```

## Related

- [blackcat start](/cli/start)
- [blackcat restart](/cli/restart)
- [blackcat status](/cli/status)
