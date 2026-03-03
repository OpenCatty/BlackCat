---
title: Anthropic Provider
description: Set up and configure Anthropic Claude for BlackCat
---

# Anthropic

Anthropic Claude models via an OpenAI-compatible API endpoint. Configure `baseURL` if using a proxy or direct Anthropic API.

## Setup

```bash
blackcat configure --provider anthropic --api-key sk-ant-your-key --model claude-sonnet-4-6
```

## YAML Configuration

```yaml
llm:
  provider: "anthropic"
  model: "claude-sonnet-4-6"
  apiKey: "sk-ant-..."
  baseURL: "https://api.anthropic.com/v1"  # optional
```

## Supported Models

- `claude-opus-4-6`
- `claude-sonnet-4-6` (Default)
- `claude-haiku-4-5`
- Other Anthropic models available via the API.

## Tips

- Anthropic models are well-known for their reasoning and structured output, making them excellent choices for coding tasks.
- Ensure your `baseURL` is correct if you're routing through an intermediary API.
