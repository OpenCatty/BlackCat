# Installing BlackCat with Cursor

## Quick Install

In Cursor's AI chat (Cmd/Ctrl+L), paste:

```
Read https://raw.githubusercontent.com/OpenCatty/BlackCat/main/docs/install/ai-agent-guide.md and follow the steps.
```

## Cursor-Specific Notes

- Cursor's Cascade can execute terminal commands directly
- Use the built-in terminal for installation commands
- Config file: `~/.blackcat/config.json5` (JSON5 format)

## Cursor Rules Integration

After installation, add to your `.cursor/rules/blackcat.md`:

```markdown
# BlackCat

- Config: ~/.blackcat/config.json5 (JSON5 format)
- Test: node node_modules/vitest/vitest.mjs run src/blackcat/
- See AGENTS.md for full architecture guide
```
