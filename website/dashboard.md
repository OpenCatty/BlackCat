---
title: Dashboard
description: Web dashboard for monitoring and managing your BlackCat deployment
---

# Dashboard Guide

The BlackCat Dashboard provides a real-time web interface for monitoring subsystem health, scheduled tasks, and system events.

## Setup

Enable the dashboard in your `config.yaml`:

```yaml
dashboard:
  enabled: true
  addr: ":8081"
  token: "your-secret-auth-token"
```

## First Access

1. Start the daemon: `blackcat daemon`
2. Open `http://localhost:8081/dashboard/`
3. Provide your token in the `Authorization: Bearer <token>` header or through the UI prompt.

## Available Pages

- **Agents** — Displays the health and status of all registered subsystems.
- **Tasks** — Lists all scheduled jobs from the `scheduler` and their current status.
- **Health** — Provides a JSON endpoint for system health checks at `/dashboard/api/health`.

## SSE Real-Time Events

Real-time updates are pushed to the dashboard via Server-Sent Events (SSE) at `/dashboard/events`. This endpoint is used for live UI updates and does not require authentication in the current version (keepalive only).

## Token Management

The dashboard token can be:
- Set directly in `config.yaml` under `dashboard.token`.
- Provided via the `BLACKCAT_DASHBOARD_TOKEN` environment variable.
- Stored in the encrypted vault for better security.

## Template Customization

For development, you can hot-reload dashboard templates by setting the `BLACKCAT_DEV_TEMPLATE_DIR` environment variable to your local templates path.

## Configuration Reference

| Field | YAML key | Default | Description |
|-------|----------|---------|-------------|
| Enabled | `enabled` | `false` | Enable web dashboard |
| Addr | `addr` | `":8081"` | Listen address |
| Token | `token` | `""` | Bearer auth token |
