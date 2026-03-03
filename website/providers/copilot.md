---
title: GitHub Copilot Provider
description: Use your GitHub Copilot subscription as an LLM provider
---

# GitHub Copilot

Uses your existing GitHub Copilot subscription as an LLM provider. Authentication is via OAuth device code flow (RFC 8628) — you authorize through GitHub in your browser.

## Architecture

- **OAuth Token (long-lived):** Exchanged for Copilot API token (~30 min TTL, auto-refreshed).
- **Chat Endpoint:** `api.githubcopilot.com/chat/completions` (OpenAI-compatible).
- **Required Headers:** `User-Agent: GitHubCopilotChat/0.37.5`, `Editor-Version: vscode/1.109.2`, `Copilot-Integration-Id: vscode-chat`.

## Setup

```bash
blackcat configure --provider copilot
```

This triggers the device code flow:
1. BlackCat displays a URL and user code.
2. Open the URL in your browser and enter the code.
3. Authorize the GitHub OAuth application.
4. Token is automatically saved to the encrypted vault.

## YAML Configuration

```yaml
oauth:
  copilot:
    enabled: true
    clientID: "01ab8ac9400c4e429b23"   # default VS Code client ID
providers:
  copilot:
    enabled: true
    model: "gpt-4.1"
```

## Supported Models

- `gpt-4.1` (Default)
- `gpt-4o`
- `gpt-4o-mini`

## Related

- [OAuth Concepts](/concepts/oauth)
