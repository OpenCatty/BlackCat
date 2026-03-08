# BlackCat Configuration Reference

BlackCat uses JSON5 format. JSON5 supports:
- Single-line comments (`//`) and multi-line comments (`/* */`)
- Trailing commas in objects and arrays
- Unquoted object keys
- Single-quoted strings

## Config File Location

```
~/.blackcat/config.json5
```

## Skills Configuration (`skills`)

### `skills.load.extraDirs`
Array of directories to load skills from.

```json5
skills: {
  load: {
    extraDirs: ["./workspaces/shared-skills"],
  },
}
```

| Key | Type | Description |
|-----|------|-------------|
| `skills.load.extraDirs` | string[] | Directories containing skill subdirectories |

---

## Agent Configuration (`agents`)

### `agents.defaults`
Default settings applied to all agents unless overridden.

```json5
agents: {
  defaults: {
    model: { primary: "gpt-4o" },
  },
}
```

| Key | Type | Description |
|-----|------|-------------|
| `agents.defaults.model.primary` | string | Default LLM model for all agents |

### `agents.list`
Array of agent (role) configurations.

```json5
agents: {
  list: [
    {
      id: "blackcat-phantom",
      name: "BlackCat Phantom",
      workspace: "./workspaces/phantom",
      model: { primary: "gpt-4o" },
      params: { temperature: 0.2 },
    },
  ],
}
```

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `id` | string | Yes | Unique agent identifier (used in router mapping) |
| `name` | string | Yes | Display name for the agent |
| `workspace` | string | Yes | Path to the role's workspace directory |
| `model.primary` | string | No | LLM model override (uses default if omitted) |
| `params.temperature` | number | No | Sampling temperature (0.0-2.0) |
| `default` | boolean | No | Whether this is the fallback agent (typically oracle) |

---

## Complete Example Configuration

```json5
// BlackCat Configuration
// Located at ~/.blackcat/config.json5

{
  skills: {
    load: {
      extraDirs: ["./workspaces/shared-skills"],
    },
  },
  agents: {
    defaults: {
      model: { primary: "gpt-4o" },
    },
    list: [
      {
        id: "blackcat-phantom",
        name: "BlackCat Phantom",
        workspace: "./workspaces/phantom",
        model: { primary: "gpt-4o" },
        params: { temperature: 0.2 },
      },
      {
        id: "blackcat-astrology",
        name: "BlackCat Astrology",
        workspace: "./workspaces/astrology",
        model: { primary: "gpt-4o" },
        params: { temperature: 0.4 },
      },
      {
        id: "blackcat-wizard",
        name: "BlackCat Wizard",
        workspace: "./workspaces/wizard",
        model: { primary: "gpt-4o" },
        params: { temperature: 0.2 },
      },
      {
        id: "blackcat-artist",
        name: "BlackCat Artist",
        workspace: "./workspaces/artist",
        model: { primary: "gpt-4o" },
        params: { temperature: 0.7 },
      },
      {
        id: "blackcat-scribe",
        name: "BlackCat Scribe",
        workspace: "./workspaces/scribe",
        model: { primary: "gpt-4o" },
        params: { temperature: 0.5 },
      },
      {
        id: "blackcat-explorer",
        name: "BlackCat Explorer",
        workspace: "./workspaces/explorer",
        model: { primary: "gpt-4o" },
        params: { temperature: 0.3 },
      },
      {
        id: "blackcat-oracle",
        name: "BlackCat Oracle",
        default: true,
        workspace: "./workspaces/oracle",
        model: { primary: "gpt-4o" },
        params: { temperature: 0.5 },
      },
    ],
  },
}
```

---

## Agent ID to Role Mapping

Agent IDs in config must match the `agentId` values in `src/blackcat/router.ts`:

| Role | Agent ID | Router agentId |
|------|----------|----------------|
| phantom | `blackcat-phantom` | `blackcat-phantom` |
| astrology | `blackcat-astrology` | `blackcat-astrology` |
| wizard | `blackcat-wizard` | `blackcat-wizard` |
| artist | `blackcat-artist` | `blackcat-artist` |
| scribe | `blackcat-scribe` | `blackcat-scribe` |
| explorer | `blackcat-explorer` | `blackcat-explorer` |
| oracle | `blackcat-oracle` | `blackcat-oracle` |

---

## Environment Variables

These environment variables affect BlackCat behavior:

| Variable | Required By | Purpose |
|----------|-------------|---------|
| `BLACKCAT_PINCHTAB_ENABLED` | pinchtab-browsing skill | Enable web browsing |
| `BLACKCAT_PINCHTAB_BASE_URL` | pinchtab-browsing skill | PinchTab API URL |
| `BLACKCAT_PINCHTAB_TOKEN` | pinchtab-browsing skill | PinchTab auth token |
| `GEMINI_API_KEY` | veo3-video-gen, nano-banana-pro | Google Gemini API |
| `CGO_ENABLED=1` | WhatsApp channel | SQLite support for WhatsApp |
| `OPENAI_API_KEY` | GPT models | OpenAI API access |
| `ANTHROPIC_API_KEY` | Claude models | Anthropic API access |

---

## Temperature Guidelines

| Temperature | Use Case | Roles |
|-------------|----------|-------|
| 0.1-0.3 | Precise, deterministic | phantom (infrastructure), wizard (coding) |
| 0.4-0.5 | Balanced | astrology (trading), scribe (writing), oracle (general) |
| 0.6-0.8 | Creative | artist (social media) |
