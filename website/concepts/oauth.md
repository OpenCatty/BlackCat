---
title: OAuth Setup
description: GitHub Copilot device flow and Antigravity PKCE OAuth walkthrough
---

# OAuth Setup

BlackCat supports OAuth authentication for LLM providers that require it. Currently two providers use OAuth:

| Provider | OAuth Flow | Token Lifetime | Refresh |
|----------|-----------|----------------|---------|
| [GitHub Copilot](/providers/copilot) | Device Code (RFC 8628) | Long-lived (OAuth) / ~30min (API) | Auto |
| [Antigravity](/providers/antigravity) | Browser PKCE | Standard OAuth2 | Auto |

## Overview

OAuth tokens are stored encrypted in the BlackCat vault (`~/.blackcat/vault.json`) using AES-256-GCM encryption. The vault passphrase is required to access stored tokens.

Both OAuth flows can be initiated via:
- The interactive wizard: `blackcat configure`
- Non-interactive command: `blackcat configure --provider copilot` or `--provider antigravity`

## GitHub Copilot — Device Code Flow

GitHub Copilot uses the OAuth 2.0 Device Authorization Grant (RFC 8628). This flow is designed for devices that don't have a browser — you authenticate in a separate browser window using a one-time code.

### How It Works

```
┌─────────────┐                ┌───────────┐                ┌──────────────────┐
│ BlackCat  │                │  GitHub   │                │ Copilot API      │
│   CLI       │                │  OAuth    │                │ (chat endpoint)  │
└──────┬──────┘                └─────┬─────┘                └────────┬─────────┘
       │  1. Request device code     │                               │
       │────────────────────────────>│                               │
       │  2. Return code + URL       │                               │
       │<────────────────────────────│                               │
       │                             │                               │
       │  [User opens URL, enters code, authorizes]                  │
       │                             │                               │
       │  3. Poll for token          │                               │
       │────────────────────────────>│                               │
       │  4. Return OAuth token      │                               │
       │<────────────────────────────│                               │
       │                             │                               │
       │  5. Exchange for API token  │                               │
       │─────────────────────────────────────────────────────────────>│
       │  6. Return Copilot API token (~30min TTL)                   │
       │<─────────────────────────────────────────────────────────────│
       │                                                             │
       │  7. Chat completion request (OpenAI-compatible)             │
       │─────────────────────────────────────────────────────────────>│
```

### Setup

Run the configure command:

```bash
blackcat configure --provider copilot
```

The CLI will display a verification URL and a user code. Open the URL in your browser, enter the code, and click "Authorize".

## Antigravity — Browser PKCE Flow

Antigravity uses OAuth 2.0 with PKCE (Proof Key for Code Exchange). A local HTTP server handles the callback — your browser opens automatically for Google authentication.

### How It Works

```
┌─────────────┐          ┌──────────────┐         ┌──────────────────────────┐
│ BlackCat  │          │ Local HTTP   │         │ Google OAuth             │
│   CLI       │          │ :51121       │         │ accounts.google.com      │
└──────┬──────┘          └──────┬───────┘         └────────────┬─────────────┘
       │  1. Generate PKCE      │                              │
       │     code_verifier +    │                              │
       │     code_challenge     │                              │
       │  2. Start local server │                              │
       │───────────────────────>│                              │
       │  3. Open browser with auth URL                        │
       │──────────────────────────────────────────────────────>│
       │                        │                              │
       │  [User authenticates in browser]                      │
       │                        │                              │
       │                        │  4. Redirect with auth code  │
       │                        │<─────────────────────────────│
       │  5. Exchange code for  │                              │
       │     token (with PKCE   │                              │
       │     code_verifier)     │                              │
       │──────────────────────────────────────────────────────>│
       │  6. Return access token                               │
       │<──────────────────────────────────────────────────────│
```

### Setup

Run the configure command:

```bash
blackcat configure --provider antigravity
```

Accept the Terms of Service (ToS) risk and follow the instructions to authenticate in your browser.

## Token Storage

All OAuth tokens are stored in the BlackCat vault (`~/.blackcat/vault.json`), encrypted with AES-256-GCM.

| Key | Contents |
|-----|----------|
| `oauth.copilot` | GitHub Copilot OAuth token (JSON) |
| `oauth.antigravity` | Google Antigravity OAuth token (JSON) |

Access the vault by setting the `BLACKCAT_VAULT_PASSPHRASE` environment variable or using the `--passphrase` flag.

## Related

- [LLM Providers](/providers)
- [Architecture](/concepts/architecture)
- [blackcat configure](/cli/configure)
