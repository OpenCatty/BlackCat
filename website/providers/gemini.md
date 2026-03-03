---
title: Google Gemini Provider
description: Direct access to Google Gemini models
---

# Google Gemini (Official)

Direct access to Google's Gemini models using the official Gemini API with API key authentication.

## Setup

```bash
blackcat configure --provider gemini --api-key your-google-api-key --model gemini-2.5-flash
```

## YAML Configuration

```yaml
providers:
  gemini:
    enabled: true
    model: "gemini-2.5-flash"
    apiKey: "your-google-api-key"    # or BLACKCAT_PROVIDERS_GEMINI_APIKEY
```

## Supported Models

- `gemini-2.5-pro`
- `gemini-2.5-flash` (Default)
- `gemini-3.1-pro-preview`
- `gemini-3-flash-preview`
- Other models available through the Gemini API.

## Tips

- Gemini uses its own wire format (not OpenAI-compatible). BlackCat handles the format conversion automatically via the built-in Gemini codec.
- Gemini models are very capable and often have large context windows.
