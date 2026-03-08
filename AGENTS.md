# BlackCat — Agent Guide

BlackCat is a TypeScript-based multi-channel AI agent daemon that routes messages from chat platforms to specialized subagents.

**Module:** `github.com/OpenCatty/BlackCat`

---

## Architecture

BlackCat processes messages through a pipeline of classification and routing:

```
Message → Daemon → Supervisor → ClassifyMessage → Role → Subagent → LLM
```

1. **Message**: Incoming from Telegram, Discord, WhatsApp, or other channels
2. **Daemon**: Entry point that initializes all components (`src/entry.ts`)
3. **Supervisor**: Orchestrates message flow and manages subagent dispatch
4. **ClassifyMessage**: Keyword-based router that determines role assignment
5. **Role**: Matched role configuration with priority, keywords, and workspace
6. **Subagent**: Specialized AI agent that handles the specific task
7. **LLM**: Language model execution with role-specific prompts

---

## Project Structure

```
D:/Projects/StarTower/BlackCat-New/
├── src/
│   ├── blackcat/
│   │   ├── router.ts           # ClassifyMessage, DEFAULT_ROLES, 7-role keyword router
│   │   └── router.test.ts      # 17 tests (ALL must stay green)
│   ├── auto-reply/
│   │   └── reply/
│   │       └── get-reply.ts    # PATCHED with BlackCat routing (lines 15, 76)
│   ├── config/
│   │   └── paths.ts            # Config path constants (~/.blackcat/config.json5)
│   ├── agents/
│   │   └── identity.ts         # Default prefix=[blackcat]
│   └── entry.ts                # process.title='blackcat'
├── workspaces/
│   ├── phantom/                # Infra/DevOps role workspace
│   ├── astrology/              # Crypto/Web3 role workspace
│   ├── wizard/                 # Code/development role workspace
│   ├── artist/                 # Social media role workspace
│   ├── scribe/                 # Writing/documentation role workspace
│   ├── explorer/               # Research role workspace
│   ├── oracle/                 # Fallback role workspace
│   └── shared-skills/          # 18 SKILL.md files (reusable capabilities)
├── blackcat.example.json5      # Config template (JSON5 format)
└── SOUL.md                     # Cat personality (NEVER evict from context)
```

Each role workspace contains:
- `AGENTS.md`: Role-specific directives and capabilities
- `SOUL.md`: Personality and communication style for that role

---

## The 7 Roles

BlackCat uses a priority-based role router. Lower priority numbers have higher precedence.

| Role | Priority | Keywords | Purpose |
|------|----------|----------|---------|
| `phantom` | 10 | restart, deploy, server, docker, nginx, k8s, systemd | Infrastructure and DevOps |
| `astrology` | 20 | crypto, bitcoin, eth, trading, wallet, defi, web3 | Cryptocurrency and Web3 |
| `wizard` | 30 | code, implement, bug, fix, typescript, golang, git | Software engineering |
| `artist` | 40 | instagram, tiktok, post, caption, hashtag, viral | Social media and content |
| `scribe` | 50 | write, draft, blog, email, copy, translate | Writing and documentation |
| `explorer` | 60 | search, find, research, analyze, summarize | Research and information |
| `oracle` | 100 | *fallback* | Default when no other role matches |

The router in `src/blackcat/router.ts` sorts roles by priority and returns the first match based on keyword presence in the message text.

---

## Build and Test Commands

Install dependencies:
```bash
node install
```

**Note:** `pnpm` is NOT available in the shell. Use `node` directly.

Run BlackCat-specific tests (17 tests must all pass):
```bash
node node_modules/vitest/vitest.mjs run src/blackcat/
```

Run all tests:
```bash
node node_modules/vitest/vitest.mjs run
```

Build the project:
```bash
node node_modules/tsdown/bin/tsdown.mjs
```

---

## Adding a New Role

To add a new role to BlackCat:

1. **Add to config**: Update `blackcat.example.json5` with the new role in `agents.list`:
   ```json5
   {
     id: "blackcat-newrole",
     name: "BlackCat NewRole",
     workspace: "./workspaces/newrole",
     model: { primary: "gpt-4o" },
     params: { temperature: 0.3 },
   }
   ```

2. **Create workspace**: Create `workspaces/newrole/` with:
   - `AGENTS.md`: Role directives, capabilities, constraints
   - `SOUL.md`: Personality for this role

3. **Update router**: Add the role to `src/blackcat/router.ts` `DEFAULT_ROLES`:
   ```typescript
   {
     name: "newrole",
     priority: 35,  // Choose appropriate priority
     agentId: "blackcat-newrole",
     keywords: ["keyword1", "keyword2", "keyword3"],
   }
   ```

4. **Run tests**: Verify all 17 tests in `router.test.ts` still pass.

---

## Adding a New Skill

Skills are reusable capabilities stored in `workspaces/shared-skills/`.

1. **Create skill directory**: Create `workspaces/shared-skills/<skill-name>/`

2. **Write SKILL.md**: Add a `SKILL.md` file with frontmatter:
   ```yaml
   ---
   name: skill-name
   version: 1.0.0
   tags: [automation, git]
   requires:
     bins: [git, gh]
     env: [GITHUB_TOKEN]
   ---
   
   # Skill Title
   
   Description of what this skill does...
   ```

**Important:** Skills MUST be in subdirectories. Flat `.md` files in `shared-skills/` are ignored.

Skills auto-load via the `skills.load.extraDirs` config setting.

---

## Configuration Format

BlackCat uses **JSON5** for configuration (NOT YAML).

Config location: `~/.blackcat/config.json5`

See `blackcat.example.json5` for the full template including:
- Skills loading configuration
- Agent defaults and per-agent overrides
- Model settings and temperature controls

Never create YAML config files. All config must be valid JSON5 with comments and trailing commas allowed.

---

## Technical Constraints

### CGO Requirement

`CGO_ENABLED=1` is required for WhatsApp channel support due to SQLite dependencies. Ensure this is set when building for WhatsApp functionality.

### Coding Style

- TypeScript strict mode
- No `any` types
- No `@ts-ignore` comments
- Explicit error handling
- Colocated tests: `*.test.ts` next to source files

### Commit Format

Use Conventional Commits:
```
type(scope): description
```

Types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`

Examples:
- `feat(phantom): add docker container restart command`
- `fix(router): correct priority sorting for equal priorities`
- `docs(oracle): update fallback behavior documentation`

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `src/blackcat/router.ts` | Keyword classification and role routing |
| `src/blackcat/router.test.ts` | Router unit tests (17/17 must pass) |
| `src/entry.ts` | Daemon entry point |
| `src/config/paths.ts` | Config file path resolution |
| `blackcat.example.json5` | Configuration template |
| `SOUL.md` | Core cat personality (load in every context) |

---

## RTK Instructions

If `rtk` (Rust Token Killer) is available, prefix commands for condensed output:

```bash
rtk node node_modules/vitest/vitest.mjs run
rtk git status
rtk git diff
```

RTK provides 60-90% token reduction on common development operations.

---

## Testing Requirements

The router test suite in `src/blackcat/router.test.ts` contains 17 tests that validate:
- Keyword matching for all 7 roles
- Priority ordering
- Fallback behavior to oracle
- Case-insensitive matching

**All 17 tests must pass after any router changes.**

Run with:
```bash
node node_modules/vitest/vitest.mjs run src/blackcat/
```
