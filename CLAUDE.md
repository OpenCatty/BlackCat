# BlackCat — Claude Agent Instructions

**Module:** `github.com/OpenCatty/BlackCat`

**Working Directory:** `D:/Projects/StarTower/BlackCat-New/`

---

## Key Commands

Tests (router must pass 17/17):
```bash
node node_modules/vitest/vitest.mjs run src/blackcat/
```

All tests:
```bash
node node_modules/vitest/vitest.mjs run
```

Build:
```bash
node node_modules/tsdown/bin/tsdown.mjs
```

**Note:** `pnpm` is NOT available. Use `node` directly.

---

## Key Files

| File | Purpose |
|------|---------|
| `src/blackcat/router.ts` | 7-role router (ClassifyMessage, DEFAULT_ROLES) |
| `blackcat.example.json5` | Config template (JSON5 format) |
| `SOUL.md` | Cat personality (never edit without care) |
| `workspaces/` | Role workspaces + shared-skills |

---

## RTK

If `rtk` is available, prefix commands:
```bash
rtk node ...
rtk git ...
```

---

## Constraints

- **CGO_ENABLED=1** for WhatsApp channel (SQLite)
- **Config format:** JSON5 (NOT YAML)
- **Config path:** `~/.blackcat/config.json5`
- **Router tests:** 17/17 must always pass
- **No new Go deps**
