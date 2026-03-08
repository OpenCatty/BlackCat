---
summary: "CLI reference for `blackcat reset` (reset local state/config)"
read_when:
  - You want to wipe local state while keeping the CLI installed
  - You want a dry-run of what would be removed
title: "reset"
---

# `blackcat reset`

Reset local config/state (keeps the CLI installed).

```bash
blackcat reset
blackcat reset --dry-run
blackcat reset --scope config+creds+sessions --yes --non-interactive
```
