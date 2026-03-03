---
title: Zen Coding Plan Provider
description: Curated hosted models from OpenCode API
---

# Zen Coding Plan

Curated hosted model access through the OpenCode API. Provides a selection of high-quality models with per-request billing.

## Setup

```bash
blackcat configure --provider zen --api-key your-zen-key --model opencode/claude-sonnet-4-6
```

## YAML Configuration

```yaml
zen:
  enabled: true
  apiKey: "your-zen-key"
  baseURL: "https://api.opencode.ai/v1"   # default
providers:
  zen:
    enabled: true
    model: "opencode/claude-sonnet-4-6"
```

## Curated Models

- `opencode/claude-opus-4-6`
- `opencode/claude-sonnet-4-6` (Default)
- `opencode/gemini-3.1-pro`

## Tips

- Zen Coding Plan is optimized for use with the OpenCode CLI.
- No complex setup or API accounts are needed — just your Zen API key.
