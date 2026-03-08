---
summary: "CLI reference for `blackcat config` (get/set/unset/file/validate)"
read_when:
  - You want to read or edit config non-interactively
title: "config"
---

# `blackcat config`

Config helpers: get/set/unset/validate values by path and print the active
config file. Run without a subcommand to open
the configure wizard (same as `blackcat configure`).

## Examples

```bash
blackcat config file
blackcat config get browser.executablePath
blackcat config set browser.executablePath "/usr/bin/google-chrome"
blackcat config set agents.defaults.heartbeat.every "2h"
blackcat config set agents.list[0].tools.exec.node "node-id-or-name"
blackcat config unset tools.web.search.apiKey
blackcat config validate
blackcat config validate --json
```

## Paths

Paths use dot or bracket notation:

```bash
blackcat config get agents.defaults.workspace
blackcat config get agents.list[0].id
```

Use the agent list index to target a specific agent:

```bash
blackcat config get agents.list
blackcat config set agents.list[1].tools.exec.node "node-id-or-name"
```

## Values

Values are parsed as JSON5 when possible; otherwise they are treated as strings.
Use `--strict-json` to require JSON5 parsing. `--json` remains supported as a legacy alias.

```bash
blackcat config set agents.defaults.heartbeat.every "0m"
blackcat config set gateway.port 19001 --strict-json
blackcat config set channels.whatsapp.groups '["*"]' --strict-json
```

## Subcommands

- `config file`: Print the active config file path (resolved from `BLACKCAT_CONFIG_PATH` or default location).

Restart the gateway after edits.

## Validate

Validate the current config against the active schema without starting the
gateway.

```bash
blackcat config validate
blackcat config validate --json
```
