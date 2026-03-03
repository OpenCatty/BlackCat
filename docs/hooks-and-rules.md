# Hooks and Rules Guide

BlackCat's extensibility system allows you to hook into core events and define conditional rules to customize the agent's behavior.

## Hook System

The hook system provides extension points throughout the agent's lifecycle.

### Available Events

- `PreChat`, `PostChat` — Before and after LLM calls.
- `PreToolExec`, `PostToolExec` — Before and after a tool is executed.
- `PreFileRead`, `PostFileRead` — Before and after reading a file.
- `PreFileWrite`, `PostFileWrite` — Before and after writing to a file.
- `OnSessionStart`, `OnSessionEnd` — Triggered when a conversation session starts or ends.

### Registration and Semantics

Hooks are registered via the `registry.Register(event, function)` call.
- **Pre-events** can short-circuit operations by returning an error.
- **Post-events** collect all errors from registered functions without stopping execution.
- **Panic Recovery** — The system automatically recovers from panics in hook functions.

## Rules System

Rules are `.md` files with YAML frontmatter that define when specific instructions should be injected into the agent's context.

### Creating Rule Files

Place `.md` files in your configured rules directory:

```markdown
---
name: go-style-guide
globs:
  - "internal/**/*.go"
  - "cmd/**/*.go"
---
When working with Go code, follow these principles:
- Use clear, descriptive names.
- Avoid excessive comments; write self-documenting code.
- Ensure all errors are handled explicitly.
```

Rules are injected during the `PostFileRead` hook, ensuring the agent receives relevant context when working with matching files.

## Hierarchical AGENTS.md

BlackCat looks for `AGENTS.md` files starting from the workspace root down to the current directory (up to 3 levels deep).
- Files are merged with a `\n\n---\n\n` separator.
- Merged content is injected into the system prompt.
- Includes symlink protection to prevent infinite loops.

## Skills YAML Frontmatter

Skills can now include YAML frontmatter to define embedded MCP servers:

```yaml
---
name: server-orchestrator
mcpServers:
  - name: docker-mcp
    command: npx
    args: ["-y", "@procedural/docker-mcp"]
---
This skill allows managing Docker containers via MCP...
```

Skills without frontmatter continue to load as plain text for backward compatibility.

## Custom Agent Profiles

Profiles allow you to define system prompt overlays in separate `.md` files.
- Stored in the configured `profiles.dir`.
- Select a profile for a specific request or session.
- Profiles are merged with the base system prompt to tailor the agent's persona or specialized knowledge.
