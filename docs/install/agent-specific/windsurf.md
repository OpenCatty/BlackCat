# Installing BlackCat with Windsurf

## Quick Install

In Windsurf's Cascade chat, paste:

```
Read https://raw.githubusercontent.com/OpenCatty/BlackCat/main/docs/install/ai-agent-guide.md and follow the steps.
```

## Windsurf-Specific Notes

- Cascade can execute terminal commands and edit files
- Use Cascade's terminal integration for installation
- Config format: JSON5 at `~/.blackcat/config.json5`
- Windsurf's `.windsurfrules` can point to BlackCat's AGENTS.md for context

## After Installation

Verify: `blackcat health` should return `{"status":"ok"}`
