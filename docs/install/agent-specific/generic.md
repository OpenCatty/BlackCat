# Installing BlackCat with Any AI Agent

## Universal Installation Prompt

Paste this to any AI agent (Claude, GPT-4, Gemini, etc.):

```
Read this installation guide and follow the steps:
https://raw.githubusercontent.com/OpenCatty/BlackCat/main/docs/install/ai-agent-guide.md
```

## What Your Agent Needs Access To

For full installation, your agent needs:

- Terminal/bash execution capability
- File write access (for config setup)
- Internet access (for npm install)

## Manual Fallback

If your agent can't execute commands:

1. Follow the steps in [docs/install/manual.md](../manual.md) yourself
2. Use your agent to generate config values (API keys, tokens)
3. Paste the config into `~/.blackcat/config.json5`
