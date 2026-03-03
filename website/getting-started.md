---
title: Quick Start
description: Get up and running with BlackCat in 4 simple steps
---

# Quick Start

Follow these steps to install BlackCat, configure your first provider, and start the agent daemon.

## 1. Install

Run the one-line installer to download and set up the BlackCat binary:

```bash
curl -fsSL https://raw.githubusercontent.com/startower-observability/BlackCat/main/scripts/install.sh | sh
```

For Windows or alternative methods, see the [Installation](/installation) guide.

## 2. Onboard

Run the interactive onboarding wizard to configure your environment:

```bash
blackcat onboard
```

The wizard will guide you through choosing an LLM provider, setting up a messaging channel, and installing the background daemon.

## 3. Check Status

Verify that the daemon is running and check your current configuration:

```bash
blackcat status
```

## 4. You're Done

Your BlackCat agent is now active. You can monitor its activity through the [Dashboard](/concepts/dashboard) or start interacting with it via your configured [Channel](/channels/whatsapp).

### Next Steps

- [CLI Reference](/cli/onboard) — Explore all available commands
- [Provider Guides](/providers) — Detailed setup for OpenAI, Anthropic, and more
- [Architecture](/concepts/architecture) — Learn how BlackCat works under the hood
