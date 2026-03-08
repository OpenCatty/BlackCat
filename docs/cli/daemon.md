---
summary: "CLI reference for `blackcat daemon` (legacy alias for gateway service management)"
read_when:
  - You still use `blackcat daemon ...` in scripts
  - You need service lifecycle commands (install/start/stop/restart/status)
title: "daemon"
---

# `blackcat daemon`

Legacy alias for Gateway service management commands.

`blackcat daemon ...` maps to the same service control surface as `blackcat gateway ...` service commands.

## Usage

```bash
blackcat daemon status
blackcat daemon install
blackcat daemon start
blackcat daemon stop
blackcat daemon restart
blackcat daemon uninstall
```

## Subcommands

- `status`: show service install state and probe Gateway health
- `install`: install service (`launchd`/`systemd`/`schtasks`)
- `uninstall`: remove service
- `start`: start service
- `stop`: stop service
- `restart`: restart service

## Common options

- `status`: `--url`, `--token`, `--password`, `--timeout`, `--no-probe`, `--deep`, `--json`
- `install`: `--port`, `--runtime <node|bun>`, `--token`, `--force`, `--json`
- lifecycle (`uninstall|start|stop|restart`): `--json`

Notes:

- `status` resolves configured auth SecretRefs for probe auth when possible.
- When token auth requires a token and `gateway.auth.token` is SecretRef-managed, `install` validates that the SecretRef is resolvable but does not persist the resolved token into service environment metadata.
- If token auth requires a token and the configured token SecretRef is unresolved, install fails closed.
- If both `gateway.auth.token` and `gateway.auth.password` are configured and `gateway.auth.mode` is unset, install is blocked until mode is set explicitly.

## Prefer

Use [`blackcat gateway`](/cli/gateway) for current docs and examples.
