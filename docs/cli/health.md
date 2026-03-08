---
summary: "CLI reference for `blackcat health` (gateway health endpoint via RPC)"
read_when:
  - You want to quickly check the running Gateway’s health
title: "health"
---

# `blackcat health`

Fetch health from the running Gateway.

```bash
blackcat health
blackcat health --json
blackcat health --verbose
```

Notes:

- `--verbose` runs live probes and prints per-account timings when multiple accounts are configured.
- Output includes per-agent session stores when multiple agents are configured.
