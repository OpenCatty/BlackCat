---
title: OpenRouter Provider
description: Access multiple LLMs through OpenRouter
---

# OpenRouter

OpenRouter provides access to multiple LLM providers through a single API key, with usage-based pricing.

## Setup

```bash
blackcat configure --provider openrouter --api-key sk-or-your-key --model anthropic/claude-3.5-sonnet
```

## YAML Configuration

```yaml
llm:
  provider: "openrouter"
  model: "anthropic/claude-3.5-sonnet"
  apiKey: "sk-or-..."
  baseURL: "https://openrouter.ai/api/v1"   # optional, auto-detected
```

## Supported Models

Any model available on [OpenRouter](https://openrouter.ai/models) is supported. Use the `provider/model` format (e.g., `anthropic/claude-3.5-sonnet`).

## Tips

- OpenRouter is a great choice for switching between different models without needing multiple API accounts.
- Monitor your usage and spending directly on the OpenRouter dashboard.
