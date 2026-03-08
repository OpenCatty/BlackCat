---
summary: "CLI reference for `blackcat logs` (tail gateway logs via RPC)"
read_when:
  - You need to tail Gateway logs remotely (without SSH)
  - You want JSON log lines for tooling
title: "logs"
---

# `blackcat logs`

Tail Gateway file logs over RPC (works in remote mode).

Related:

- Logging overview: [Logging](/logging)

## Examples

```bash
blackcat logs
blackcat logs --follow
blackcat logs --json
blackcat logs --limit 500
blackcat logs --local-time
blackcat logs --follow --local-time
```

Use `--local-time` to render timestamps in your local timezone.
