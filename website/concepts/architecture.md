---
title: Architecture
description: How BlackCat works internally — agent loop, channels, memory, and tool execution
---

# Architecture

BlackCat is a Go-based AI agent that orchestrates OpenCode CLI via messaging channels.

## System Overview

```
┌──────────────────────────────────────────────────────────────┐
│                     BlackCat Agent                          │
│                                                              │
│  ┌────────────┐   ┌────────────┐   ┌────────────────────┐   │
│  │  Telegram   │   │  Discord   │   │     WhatsApp       │   │
│  │  Adapter    │   │  Adapter   │   │     Adapter        │   │
│  └─────┬──────┘   └─────┬──────┘   └──────────┬─────────┘   │
│        │                │                      │             │
│        └────────────────┼──────────────────────┘             │
│                         │                                    │
│                  ┌──────▼──────┐                              │
│                  │ Message Bus │  (fan-in / fan-out)          │
│                  └──────┬──────┘                              │
│                         │                                    │
│                  ┌──────▼──────┐                              │
│                  │ Agent Loop  │  (max 50 turns)              │
│                  └──┬───┬───┬─┘                              │
│                     │   │   │                                │
│           ┌─────────┘   │   └─────────┐                      │
│           │             │             │                      │
│    ┌──────▼─────┐ ┌────▼─────┐ ┌─────▼──────┐               │
│    │ LLM Backend│ │  Tools   │ │  Memory    │               │
│    │  System    │ │ Registry │ │  System    │               │
│    └──────┬─────┘ └────┬─────┘ └────────────┘               │
│           │            │                                     │
│    ┌──────▼─────┐ ┌────▼─────┐                               │
│    │ Provider   │ │ OpenCode │                               │
│    │  Registry   │ │ Delegate │                               │
│    └────────────┘ └──────────┘                               │
│                                                              │
│    ┌────────────┐ ┌────────────┐ ┌────────────┐              │
│    │  Security  │ │  Vault     │ │    MCP     │              │
│    │  Scrubber  │ │ AES-256   │ │ Server/Cli │              │
│    └────────────┘ └────────────┘ └────────────┘              │
└──────────────────────────────────────────────────────────────┘
```

## Core Components

### Agent Loop

**Package:** `agent/` — `loop.go`, `execution.go`, `compaction.go`

The agent loop is the central orchestrator. It receives a user message and iterates up to `maxTurns` (default 50), calling the LLM, executing tool calls, and collecting results until the LLM produces a final text response.

```
User Message → Build System Prompt → LLM Chat →
  ├─ Text Response → Return to user
  └─ Tool Calls → Execute each tool → Append results → Loop back to LLM
```

### LLM Backend System

**Package:** `llm/` — `backend.go`, `provider.go`, `client.go`, `openai_backend.go`

The `Backend` interface defines the contract all LLM providers must implement. The **Backend Registry** (`provider.go`) is a global concurrent-safe map of `BackendFactory` functions keyed by provider name.

### Channel Adapters

**Package:** `channel/` — `channel.go`, plus `telegram/`, `discord/`, `whatsapp/` sub-packages

The `MessageBus` fans-in messages from all registered channel adapters into a single Go channel and routes outbound responses back to the correct adapter.

### Tools Registry

**Package:** `tools/` — Tool interface with built-in tools and MCP-discovered tools

The `tools.Registry` holds all available tools. Built-in tools include shell execution (with security scrubbing), file operations, and OpenCode delegation.

### Memory System

**Package:** `memory/` — `memory.go`

File-based persistent memory using `MEMORY.md`. Supports automatic consolidation when the entry count exceeds a configurable threshold.

### Security

**Package:** `security/` — `vault.go`, `scrubber.go`

- **Vault:** AES-256-GCM encrypted JSON storage for API keys and tokens.
- **Scrubber:** Command deny-list that blocks dangerous shell commands (e.g., `rm -rf /`).

### MCP (Model Context Protocol)

**Package:** `mcp/`

Implements both MCP server and client for tool discovery and invocation across different systems.

## Request Lifecycle

1. **Channel receives message** — A Telegram/Discord/WhatsApp adapter receives a user message.
2. **MessageBus fan-in** — The adapter pushes the message into the shared `incoming` channel.
3. **Daemon dispatch** — The daemon creates a context and starts the agent loop.
4. **Agent loop starts** — `Loop.Run()` builds a system prompt including workspace context.
5. **LLM call** — The agent calls `Backend.Chat()` with history and tool definitions.
6. **Tool execution** — LLM tool calls are executed via `tools.Registry`.
7. **Iteration** — Steps 5-6 repeat until a text response is returned.
8. **Response routing** — The final response is sent back to the originating channel.
9. **Memory update** — Interaction details are appended to `MEMORY.md`.

## Provider Architecture

BlackCat supports 8 LLM providers across two wire formats (OpenAI and Gemini). All providers implement `llm.Backend` and register themselves via `llm.RegisterBackend()`.

## Related

- [Quick Start](/getting-started)
- [Installation](/installation)
- [LLM Providers](/providers)
- [OAuth Concepts](/concepts/oauth)
