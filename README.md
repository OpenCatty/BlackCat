# BlackCat — TypeScript Rewrite

BlackCat multi-channel AI agent daemon, migrated from the OpenClaw TypeScript codebase.

## Migration Note

This directory was forked from `openclaw-analysis/` as part of **Wave 1, Task 1.1** of the blackcat-migration plan.

### What changed (mechanical rebrand only)

- `package.json`: name → `blackcat`, version → `0.1.0`, description updated, bin → `blackcat.mjs`
- `blackcat.mjs`: new entry point (copy of `openclaw.mjs` with branding strings replaced)
- `README.md`: this file (replaces the original OpenClaw README)

### What did NOT change

- `src/` — untouched (Wave 1.2+)
- `packages/` — untouched
- `extensions/` — untouched
- Dependencies and scripts in `package.json` — untouched
- `openclaw.mjs` — preserved (coexists with `blackcat.mjs`)

### Next steps

- Wave 1.2: `pnpm install` + verify build
- Wave 2+: src/ modifications, role router integration, channel rewiring
