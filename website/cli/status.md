---
title: blackcat status
description: Check BlackCat daemon and subsystems status
---

# blackcat status

The status command provides a comprehensive view of the running BlackCat daemon and its connected components.

## Usage

```shell
blackcat status
```

## Description

The status command displays information about the daemon service:
- **Service Status:** Running, Stopped, or Uninstalled
- **PID:** Process ID if running
- **Config Path:** Location of the active configuration file
- **Uptime:** Current run duration
- **Subsystems:** Connectivity status of LLM providers and channels

## Examples

```shell
# Check current daemon status
blackcat status
```

## Related

- [blackcat onboard](/cli/onboard)
- [blackcat health](/cli/health)
- [blackcat start](/cli/start)
