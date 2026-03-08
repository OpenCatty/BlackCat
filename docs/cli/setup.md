---
summary: "CLI reference for `blackcat setup` (initialize config + workspace)"
read_when:
  - You’re doing first-run setup without the full onboarding wizard
  - You want to set the default workspace path
title: "setup"
---

# `blackcat setup`

Initialize `~/.blackcat/blackcat.json` and the agent workspace.

Related:

- Getting started: [Getting started](/start/getting-started)
- Wizard: [Onboarding](/start/onboarding)

## Examples

```bash
blackcat setup
blackcat setup --workspace ~/.blackcat/workspace
```

To run the wizard via setup:

```bash
blackcat setup --wizard
```
