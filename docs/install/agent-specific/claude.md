# Installing BlackCat with Claude Code

## Quick Install (Recommended)

Paste this to Claude Code:

```
Read https://raw.githubusercontent.com/OpenCatty/BlackCat/main/docs/install/ai-agent-guide.md and follow the installation steps.
```

## Manual Instructions for Claude

Claude Code can install and configure BlackCat directly. Use the bash tool to:

```bash
# Check Node.js version (must be >= 22.12.0)
node --version

# Install BlackCat
npm install -g blackcat

# Set up config
mkdir -p ~/.blackcat
cp blackcat.example.json5 ~/.blackcat/config.json5
```

## Notes for Claude

- Use `node` commands, not `pnpm` (pnpm not available in all environments)
- Config format is JSON5 (NOT YAML) — use `~/.blackcat/config.json5`
- For WhatsApp: set `CGO_ENABLED=1` before running
- Test the installation: `blackcat --version && blackcat health`

## OpenCode Integration

If you're running Claude inside OpenCode, BlackCat integrates naturally.
The `CLAUDE.md` file in the repo root contains agent-specific working instructions.
