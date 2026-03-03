---
title: Antigravity Provider
description: Access Gemini models through Google IDE API
---

# Antigravity (Google IDE)

Antigravity provides access to Google's Gemini models through the internal Google IDE API (cloudcode). Uses browser-based PKCE OAuth flow.

## Setup

```bash
blackcat configure --provider antigravity
```

You will be prompted to accept the Terms of Service (ToS) risk before proceeding.

## YAML Configuration

```yaml
oauth:
  antigravity:
    enabled: true
    acceptedToS: true    # REQUIRED: acknowledge ToS risk
    clientID: "..."      # built-in default
    clientSecret: "..."  # built-in default
providers:
  antigravity:
    enabled: true
    model: "gemini-2.5-pro"
```

## Supported Models

- `gemini-2.5-pro` (Default)
- `gemini-2.5-flash`

## ToS Risk Warning

Antigravity uses Google's internal cloudcode API (`cloudcode-pa.googleapis.com`), which is intended for Google's own IDE products. **Using this API from third-party applications may violate Google's Terms of Service.**

By setting `acceptedToS: true`, you acknowledge:
- This may stop working at any time without notice.
- Your Google account could potentially be flagged.
- This is NOT an officially supported integration.
- Use at your own risk.

## Related

- [OAuth Concepts](/concepts/oauth)
