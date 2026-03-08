# Contributing to BlackCat

We welcome contributions, even from dogs. (Just kidding. No dogs.)

---

## Getting Started

Fork the repo, clone it, and install dependencies:

```bash
git clone https://github.com/OpenCatty/BlackCat.git
cd BlackCat
node install
```

**Note:** `pnpm` is NOT available in the shell. Use `node` directly.

---

## Development Setup

Key things to know about the codebase:

- **Language:** TypeScript (ESM, strict mode)
- **Test runner:** Vitest
- **Config format:** JSON5 (not YAML)
- **Module:** `github.com/OpenCatty/BlackCat`

---

## Testing

Before submitting PRs, run the test suite:

```bash
# Run BlackCat router tests (17/17 must pass)
node node_modules/vitest/vitest.mjs run src/blackcat/

# Run all tests
node node_modules/vitest/vitest.mjs run
```

All 17 router tests must pass. These validate the 7-role keyword classification system.

---

## Adding a Role

To add a new agent role:

1. **Config**: Add entry to `blackcat.example.json5` in `agents.list`
2. **Workspace**: Create `workspaces/<role>/` with `AGENTS.md` + `SOUL.md`
3. **Router**: Add role to `src/blackcat/router.ts` `DEFAULT_ROLES`
4. **Test**: Run router tests, verify 17/17 pass

See `AGENTS.md` for full details.

---

## Adding a Skill

To add a reusable skill:

1. Create `workspaces/shared-skills/<skill-name>/SKILL.md`
2. Add frontmatter with `name`, `version`, `tags`, optional `requires`
3. Skills auto-load via config

Skills MUST be in subdirectories. Flat `.md` files are ignored.

---

## Pull Request Process

- Keep PRs small and focused (one feature per PR)
- Include tests for new functionality
- Follow existing code patterns
- Use Conventional Commit format: `type(scope): description`
- No `openclaw` references in new code

Commit types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`

---

## Code Style

- TypeScript strict mode, no `any`, no `@ts-ignore`
- Explicit error handling
- Colocated tests: `*.test.ts` next to source
- Follow existing patterns in `src/blackcat/`
- Keep files under 500 lines when possible

---

## Issues

Found a bug? Have an idea? Open an issue:

https://github.com/OpenCatty/BlackCat/issues

---

## Cat Tax

BlackCat has personality. If you're adding user-facing strings, read `SOUL.md` to understand the tone. Direct, decisive, occasionally sassy. No corporate speak.

Nyaa~ 🐱
