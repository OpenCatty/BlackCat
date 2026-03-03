---
title: OpenAI Provider
description: Set up and configure OpenAI for BlackCat
---

# OpenAI

The default provider for BlackCat. Uses the standard OpenAI Chat Completions API.

## Setup

```bash
blackcat configure --provider openai --api-key sk-your-key --model gpt-5.2
```

## YAML Configuration

```yaml
llm:
  provider: "openai"
  model: "gpt-5.2"
  apiKey: "sk-..."    # or use BLACKCAT_LLM_APIKEY env var
  temperature: 0.7
  maxTokens: 4096
```

## Supported Models

Any model available through the OpenAI API is supported:
- `gpt-5.2` (Default)
- `gpt-4.1`
- `gpt-4.1-mini`
- `o3`
- `o4-mini`
- `gpt-4o`
- `gpt-4o-mini`

## Tips

- Store your API key in the vault for better security: `blackcat vault set openai_api_key sk-your-key`.
- Adjust `temperature` for more creative or precise responses.
