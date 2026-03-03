---
title: blackcat health
description: Hit BlackCat /health endpoint
---

# blackcat health

The health command connects to the running BlackCat daemon's internal health endpoint.

## Usage

```shell
blackcat health
```

## Description

The health command makes a GET request to the `/health` endpoint of the local daemon and prints the JSON response. This is useful for scripted checks or for verifying connectivity to your LLM and channel providers.

## Examples

```shell
# Check local daemon health
blackcat health
```

## Related

- [blackcat status](/cli/status)
- [blackcat doctor](/cli/doctor)
