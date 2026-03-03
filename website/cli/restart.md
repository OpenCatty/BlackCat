---
title: blackcat restart
description: Restart the BlackCat daemon
---

# blackcat restart

The restart command stops and then starts the background BlackCat daemon service.

## Usage

```shell
blackcat restart
```

## Description

The restart command is particularly useful after manual changes to your `config.yaml` or when the daemon needs to be reloaded. It ensures all service components are properly restarted.

## Examples

```shell
# Restart the daemon
blackcat restart
```

## Related

- [blackcat start](/cli/start)
- [blackcat stop](/cli/stop)
- [blackcat status](/cli/status)
