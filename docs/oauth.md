# OAuth Setup

BlackCat supports OAuth authentication for LLM providers that require it. Currently two providers use OAuth:

| Provider | OAuth Flow | Token Lifetime | Refresh |
|----------|-----------|----------------|---------|
| GitHub Copilot | Device Code (RFC 8628) | Long-lived (OAuth) / ~30min (API) | Auto |
| Antigravity | Browser PKCE | Standard OAuth2 | Auto |

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
│ BlackCat    │                │  GitHub   │                │ Copilot API      │
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

**Two-token architecture:**
1. **OAuth token** (long-lived) — obtained from GitHub OAuth, stored in vault
2. **Copilot API token** (~30 minute TTL) — obtained by exchanging the OAuth token at `api.github.com/copilot_internal/v2/token`, auto-refreshed by the Copilot backend

### Setup

Run the configure command:

```bash
blackcat configure --provider copilot
```

The CLI will:
1. Display a verification URL (e.g., `https://github.com/login/device`)
2. Display a user code (e.g., `ABCD-1234`)
3. Wait for you to authorize (up to 10 minutes)

In your browser:
1. Open the displayed URL
2. Enter the user code
3. Click "Authorize" on the GitHub consent page
4. Return to the terminal — the token is saved automatically

**Manual YAML configuration** (after OAuth token is in vault):
```yaml
oauth:
  copilot:
    enabled: true
    clientID: "01ab8ac9400c4e429b23"   # VS Code client ID (default)
providers:
  copilot:
    enabled: true
    model: "gpt-4o"
```

### Token Management

- **Storage:** OAuth token is saved in vault under key `oauth.copilot`
- **Format:** JSON-encoded `TokenSet` with `access_token`, `token_type`, `scope`, `refresh_token` fields
- **API token refresh:** The Copilot backend automatically fetches a new API token from `api.github.com/copilot_internal/v2/token` when the current one expires (~30 min)
- **Re-authentication:** If the OAuth token is revoked, re-run `blackcat configure --provider copilot`

## Antigravity — Browser PKCE Flow

Antigravity uses OAuth 2.0 with PKCE (Proof Key for Code Exchange). A local HTTP server handles the callback — your browser opens automatically for Google authentication.

### How It Works

```
┌─────────────┐          ┌──────────────┐         ┌──────────────────────────┐
│ BlackCat    │          │ Local HTTP   │         │ Google OAuth             │
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

The CLI will:
1. Ask you to accept the Terms of Service risk
2. Start a local HTTP server for the OAuth callback
3. Display or open a Google authentication URL
4. Wait for you to authenticate (up to 5 minutes)
5. Save the token to the vault

**Manual YAML configuration** (after OAuth token is in vault):
```yaml
oauth:
  antigravity:
    enabled: true
    acceptedToS: true        # REQUIRED
    clientID: "..."          # built-in default
    clientSecret: "..."      # built-in default
providers:
  antigravity:
    enabled: true
    model: "gemini-2.5-pro"
```

### ToS Risk Acknowledgment

Antigravity uses Google's internal `cloudcode-pa.googleapis.com` API, which is not an official public API. **You must explicitly accept the risk** by either:

- Answering "Yes, I accept" in the interactive wizard, or
- Setting `oauth.antigravity.acceptedToS: true` in config

**Risks:**
- Google has been blocking unauthorized third-party access since February 2026
- Your Google account could be flagged for Terms of Service violation
- The API endpoint may change or stop working without notice
- This is NOT an officially supported integration

If you need reliable Google model access, use the official [Gemini provider](./providers.md#google-gemini-official) with an API key instead.

### Token Management

- **Storage:** Token is saved in vault under key `oauth.antigravity`
- **Format:** JSON-encoded `TokenSet` with `access_token`, `token_type`, `refresh_token`, `expiry` fields
- **Refresh:** Standard OAuth2 refresh flow using refresh token
- **Re-authentication:** Re-run `blackcat configure --provider antigravity`

## Token Storage

All OAuth tokens are stored in the BlackCat vault (`~/.blackcat/vault.json`), encrypted with AES-256-GCM.

**Vault keys:**
| Key | Contents |
|-----|----------|
| `oauth.copilot` | GitHub Copilot OAuth token (JSON) |
| `oauth.antigravity` | Google Antigravity OAuth token (JSON) |
| `provider.<name>.apikey` | API keys for key-based providers |

**Accessing the vault:**
- Set passphrase via `BLACKCAT_VAULT_PASSPHRASE` environment variable
- Or pass `--passphrase` flag to commands that access the vault
- Or you will be prompted interactively

## Troubleshooting

### "device code request: client_id is required"
The Copilot client ID is missing. Ensure `oauth.copilot.clientID` is set (defaults to VS Code client ID).

### "PKCE flow: context deadline exceeded"
The browser authentication timed out (5 minutes for Antigravity). Re-run the configure command.

### "open vault: vault passphrase required"
Set `BLACKCAT_VAULT_PASSPHRASE` environment variable or pass `--passphrase` flag.

### "token poll: access_denied"
The GitHub authorization was denied. Re-run `blackcat configure --provider copilot` and authorize the application.

### Copilot "unauthorized" after working previously
The OAuth token may have been revoked. Re-authenticate:
```bash
blackcat configure --provider copilot
```

### Antigravity "403 Forbidden"
Google may be blocking third-party access to the cloudcode API. Consider switching to the official Gemini provider.

## See Also

- [LLM Providers](./providers.md) — All provider details
- [Configuration Reference](./configuration.md) — OAuth config fields
- [CLI Configure](./configure-cli.md) — Interactive auth setup
