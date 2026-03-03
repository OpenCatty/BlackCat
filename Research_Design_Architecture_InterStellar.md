# InterStellar Research, Design, and Architecture

## Executive Summary

InterStellar is a Go based AI agent daemon engineered to operate alongside the OpenCode CLI. It functions as a bridge between consumer messaging channels such as Telegram, Discord, and WhatsApp and the OpenCode execution environment. By leveraging the go-openai library for its primary orchestration and reasoning, InterStellar delegates complex coding tasks to the OpenCode engine. The architecture is the result of extensive research into six existing agent frameworks, combining their most effective patterns into a single, cohesive system designed for high performance and reliability.

The core philosophy of InterStellar is to provide a "personal server" experience where the user can interact with their development environment from anywhere. It handles the complexities of asynchronous message handling, stateful conversation management, and secure tool execution. By using a fan-in message bus, the system can handle multiple incoming streams from different platforms simultaneously, routing them through a standardized agent loop that follows the "think, act, observe" pattern. This approach ensures that the agent remains grounded in the current context while having access to a broad suite of tools, including the ability to spawn and manage OpenCode sessions for deep technical work.

Security and simplicity are paramount. The system avoids the overhead of traditional databases by using file based memory systems, which increases transparency and portability. A multi layered security model protects the host system from common exploits such as prompt injection and unauthorized shell access. InterStellar is designed for single binary deployment, making it easy to set up and maintain while providing a robust platform for AI driven development.

## Architecture Overview

InterStellar follows a modular, event driven architecture designed for concurrency and scalability. The system is composed of four primary layers: the Channel Layer, the Message Bus, the Agent Loop, and the Tool Integration Layer.

```
+------------+  +------------+  +------------+
|  Telegram  |  |  Discord   |  |  WhatsApp  |
+-----+------+  +-----+------+  +-----+------+
      |               |               |
      v               v               v
+------------------------------------------+
|           Message Bus (fan-in)           |
+--------------------+---------------------+
                     |
                     v
+------------------------------------------+
|             Agent Loop                    |
|  think -> act -> observe -> repeat       |
+----+----------+----------+----------+----+
     |          |          |          |
     v          v          v          v
+--------+ +--------+ +--------+ +--------+
|  LLM   | | Tools  | | Memory | | Skills |
|go-openai| |exec,fs | |MEMORY  | | *.md   |
|        | |web,oc  | |  .md   | |        |
+--------+ +---+----+ +--------+ +--------+
               |
               v
+------------------------------------------+
|        OpenCode REST API + SSE           |
|     http://127.0.0.1:4096                |
+------------------------------------------+
```

The data flow starts when a user sends a message through one of the supported messaging channels. Each channel adapter (Telegram, Discord, or WhatsApp) translates platform specific events into a unified message format and pushes them onto the fan-in message bus. The bus ensures that messages are processed in the order they are received and provides a central point for monitoring and logging.

The Agent Loop is the central nervous system of InterStellar. It retrieves messages from the bus and initiates a "think, act, observe" cycle. During the "think" phase, the agent uses the go-openai client to determine the next step based on the user's intent and current state. If an action is required, the "act" phase triggers a tool execution. These tools range from simple filesystem operations to complex coding tasks delegated to the OpenCode REST API.

OpenCode integration is handled via a dedicated client that communicates over HTTP and listens for real time updates via Server Sent Events (SSE). This allows InterStellar to stream progress back to the user as the coding task unfolds. Once a tool returns a result, the "observe" phase updates the agent's internal state, and the loop repeats until the task is finished or requires user intervention. Responses are then routed back through the original channel to the user.

## Research Findings

### OpenClaw (TypeScript)

OpenClaw is a sophisticated gateway daemon and WebSocket control plane with 238k stars on GitHub. Its architecture is built around a rigorous state machine for agent loops, transitioning through phases such as idle, queued (session lane), queued (global lane), attempting, streaming, compacting, and finally completed or failed. This granular approach to state management allows for high observability and error recovery.

One of its most innovative features is the two level queue system. The session lane ensures that messages within a specific session are processed serially, preventing race conditions and context fragmentation. The global lane manages resource caps across all active sessions, ensuring that the system does not become overwhelmed by concurrent requests.

OpenClaw also implements a 5 step tool policy pipeline. This pipeline includes pre execution checks, approval flows (where the user must explicitly permit certain actions), and post execution validation. Memory management is handled through a hybrid approach, using Markdown files for human readability and SQLite with the sqlite-vec extension for high performance vector search. It uses BM25 and vector search combined with Maximal Marginal Relevance (MMR) re-ranking and temporal decay to ensure that the most relevant and recent information is always in context.

The system supports over 20 messaging channels and uses sub-agents for specialized tasks. These sub-agents can be spawned dynamically, with a depth limited nesting cap of 5 to prevent infinite loops. Configuration is handled via JSON5, and the workspace is bootstrapped with standardized files such as AGENTS.md, SOUL.md, and MEMORY.md. Communication with external tools follows the ACP protocol, a custom JSON-RPC 2.0 implementation over stdio.

### Rowboat (TypeScript)

Rowboat, with 8.8k stars, focuses on a named agent graph with handoff-based routing. It uses an explicit control stack, represented as a slice of strings, to track the chain of parent agents. This enables sophisticated "return-to-caller" semantics, where a specialized agent can complete a task and then pass control back to the agent that invoked it.

To prevent runaway loops, Rowboat implements transfer rate limiting based on the frequency of handoffs between specific agents. It categorizes agents into four types: conversation (user-facing), post-process (internal, auto-returning), escalation, and pipeline. The PipelineStateManager handles ordered sequences of agent steps, managing context injection and advancement through the workflow.

The tool dispatch table is highly flexible, supporting Mock tools, MCP (Model Context Protocol) servers, Composio, and Gemini Image tools. For streaming, it uses a two phase SSE approach where a POST request creates a Redis cache key, and a subsequent GET request on the /stream-response/:streamId endpoint initiates the stream. This separates the command from the data stream, improving reliability in distributed environments. The codebase follows Clean Architecture principles with a dependency injection (DI) container and interface based repositories, ensuring that agents remain stateless across turns.

### nanobot-ai/nanobot (Go)

The nanobot-ai project is a standalone MCP Host written in Go. It introduces a structured "Execution" object that acts as an immutable carrier for per-turn state. This struct contains the original request, the populated request (after template expansion), the response, tool outputs, and compacted messages. This design makes the entire turn execution easy to trace and debug.

Context compaction is a key focus for nanobot-ai. When the context window reaches 83.5% capacity, the system performs incremental re-summarization. The resulting summary is tagged with a specific metadata key, "ai.nanobot.meta/compaction-summary", allowing the agent to distinguish between raw history and summarized context.

The system uses the goja runtime to provide JavaScript hooks for configuration and execution (runBefore, runAfter). Persistence is handled through GORM, allowing it to support various SQL backends. State is namespaced by thread within sessions, providing a clear boundary for multi user interactions. It also features a Cobra based CLI and an embedded SvelteKit UI for management, while using Docker sandboxing to isolate MCP server executions.

### HKUDS/nanobot (Python)

HKUDS/nanobot is a lightweight implementation (~4,000 lines) that emphasizes a data driven approach. Instead of complex conditional logic for provider management, it uses a ProviderSpec dataclass that includes keywords, environment keys, and detection patterns. Adding a new provider is a simple two step process of defining the spec and registering it.

Skills are handled by injecting Markdown files into the system prompt. Any .md file in the skills directory is loaded at startup and becomes part of the agent's core capabilities. A heartbeat system executes proactive periodic tasks every 30 minutes, allowing the agent to perform background maintenance or check for updates independently.

Sessions are stored as JSON files in the user's home directory, keyed by a combination of channel and user ID. Memory management is append-only to a MEMORY.md file, with periodic consolidation by the LLM when the number of entries exceeds 50. This keeps the memory file both human readable and machine processable. It supports 10+ channels through a shared asyncio message bus and provides workspace sandboxing via a simple configuration flag.

### GoClaw and Praktor (Go)

GoClaw is a multi-agent gateway in Go that prioritizes security and versatility. It supports 13+ LLM providers and uses PostgreSQL for state management. Its security model is particularly robust, featuring five layers: rate limiting, prompt injection detection, credential scrubbing (to prevent API keys from leaking in tool outputs), shell deny patterns, and SSRF protection.

Praktor, while smaller in scale, introduces the concept of per-agent Docker containers. It integrates with the Claude Code SDK and uses a NATS message bus for inter-agent communication. For secret management, it uses an AES-256-GCM vault with keys derived using Argon2id. It also supports Nix for reproducible environments and Tailscale for secure networking.

Both projects use Docker Compose for deployment, ensuring that services are automatically restarted and have named volumes for persistence. They also implement fsnotify for hot reloading configuration files without requiring a service restart. The shell deny patterns are comprehensive, blocking dangerous commands like curl piped to shell, reverse shells, and base64 encoded payloads.

### Go Ecosystem and OpenCode API

The Go ecosystem provides a wealth of libraries that make it ideal for building an agent daemon. The sashabaranov/go-openai client is the de facto standard for interacting with OpenAI compatible APIs. For more complex needs, langchaingo offers a full framework, although it can sometimes introduce unnecessary abstraction.

The mark3labs/mcp-go library is the leading implementation of the Model Context Protocol in Go, supporting the latest specification. SSE handling is often done with tmaxmax/go-sse, which provides reliable reconnection support. For CLI development, spf13/cobra and spf13/viper are the undisputed leaders.

OpenCode itself provides a REST API that InterStellar consumes. Key endpoints include /global/health for monitoring, /session for lifecycle management, and /session/:id/prompt_async for non blocking task execution. The SSE stream from OpenCode provides granular events like session.updated and tool.active, which InterStellar uses to provide real time feedback to the user.

## Design Decisions

The design of InterStellar is informed by the strengths and weaknesses of the researched projects. Every decision was made to balance power, security, and ease of use.

### Go Programming Language
Go was chosen for its single binary deployment model and its excellent goroutine concurrency primitives. This makes it perfect for a multi channel listener that needs to handle many simultaneous network connections and background tasks without the complexity of a heavy runtime. The standard library's strong support for HTTP, JSON, and cryptography also reduces dependency bloat.

### go-openai Client
While langchaingo is a powerful framework, we chose the sashabaranov/go-openai client to avoid framework lock-in. A custom agent loop gives us full control over the "think, act, observe" cycle, allowing us to implement specific features like our context compaction strategy and tool policy pipeline without fighting against a framework's abstractions.

### V1 Messaging Channels
Telegram, Discord, and WhatsApp were selected as the initial channels because they are the primary communication tools for our target audience. Stable, well maintained Go libraries (go-telegram-bot-api, discordgo, and whatsmeow) are available for all three, ensuring a reliable integration from day one.

### File-based Memory (MEMORY.md)
Inspired by HKUDS/nanobot, we opted for a file based memory system using MEMORY.md. This eliminates the need for a database dependency, making the system more portable and transparent. Users can easily read or edit their agent's memory using any text editor. Atomic writes are guaranteed by writing to a temporary file and using os.Rename.

### Open and Deny-list Server Control
We implement a balance of power where the user has control over their personal server, but the system proactively blocks known dangerous patterns. This approach, borrowed from GoClaw, provides a safety net without being overly restrictive for legitimate development tasks.

### Per-task Session Routing
To keep the model simple, each user request typically maps to a new OpenCode session. However, the system can reference previous session IDs to maintain continuity across multiple interactions. This provides a clean separation of concerns while allowing for complex, multi step workflows.

### 5-layer Security Model
We adopted the security architecture from GoClaw and Praktor. This includes rate limiting to prevent abuse, prompt injection detection, credential scrubbing to protect secrets, shell deny patterns to block dangerous commands, and SSRF protection to prevent the agent from accessing internal network resources.

### YAML and fsnotify
Configuration is handled via YAML files, with fsnotify providing hot reload capabilities. This allows users to change the agent's behavior or update credentials without restarting the daemon, a pattern proven successful in nanobot and Praktor.

### mark3labs/mcp-go
For tool integration, we use the mark3labs/mcp-go library. As the most popular Go implementation of the Model Context Protocol, it ensures compatibility with a wide range of external tools and servers while following the latest spec (v2025-11-05).

## Technology Stack

| Purpose | Library | Stars | Notes |
|---------|---------|-------|-------|
| CLI | spf13/cobra | 43.3k | De facto standard for Go CLI applications. |
| Config | spf13/viper | — | Handles YAML, environment variables, and watch mode. |
| LLM Client | sashabaranov/go-openai | 10.6k | Reliable, lightweight OpenAI-compatible client. |
| MCP | mark3labs/mcp-go | 8.2k | Implementation of the Model Context Protocol. |
| SSE Client | tmaxmax/go-sse | 509 | Used for streaming updates from OpenCode. |
| Supervision | cirello.io/oversight/v2 | — | Erlang style supervision for long running tasks. |
| Token Count | pkoukk/tiktoken-go | — | Essential for managing the LLM context window. |
| Config Watch | fsnotify/fsnotify | — | Enables hot reloading of configuration files. |
| Crypto | golang.org/x/crypto | — | Used for the Argon2id KDF in the secret vault. |
| Telegram | go-telegram-bot-api | — | Comprehensive wrapper for the Telegram Bot API. |
| Discord | bwmarrin/discordgo | — | Handles Discord gateway and REST interactions. |
| WhatsApp | tulir/whatsmeow | — | The most stable unofficial WhatsApp library for Go. |

## Module Architecture

The InterStellar codebase is organized into distinct packages, each with a clear responsibility:

- **config/**: Responsible for loading and validating YAML configuration. It uses Viper and fsnotify to support hot reloading.
- **security/**: Implements the 5 layer security model. It includes the shell deny-list, the credential scrubber, and the AES-256-GCM vault for secrets.
- **memory/**: Manages the MEMORY.md file store. It handles atomic writes and provides an interface for the LLM to search and update its long term memory.
- **opencode/**: Contains the client and types for interacting with the OpenCode REST API and SSE stream. It manages session lifecycles and provides a supervisor for async tasks.
- **llm/**: A wrapper around go-openai that manages the provider registry and handles the complexities of tool calling and response parsing.
- **tools/**: A registry of available tools, including built-in commands for filesystem access, web searches, and the specialized opencode_task tool.
- **agent/**: Implements the core "think, act, observe" loop. It uses an Execution state carrier to track the progress of each turn and handles context compaction.
- **channel/**: Provides a unified interface for messaging platforms. It includes adapters for Telegram, Discord, and WhatsApp, all feeding into a central message bus.
- **mcp/**: Handles both the server side (exposing InterStellar tools) and the client side (consuming external MCP servers).
- **skills/**: Loads and manages the Markdown based skills system, injecting relevant content into the agent's system prompt.
- **workspace/**: Provides templates for bootstrapping new workspaces with the necessary AGENTS.md and SOUL.md files.
- **cmd/**: The entry point for the application, using Cobra to define commands like serve, run, health, and vault.

## Security Model

The security model of InterStellar is designed to be proactive and multi layered.

The shell deny-list consists of a set of compiled regular expressions that identify and block dangerous command patterns. These include attempts to use curl or wget piped to shell (curl|sh), reverse shell setups, eval statements, fork bombs, and destructive commands like rm -rf /. This list is checked before any command is executed via a shell tool.

Credential scrubbing is a critical feature that prevents the agent from accidentally leaking sensitive information. The scrubber scans all tool outputs and LLM responses for patterns matching API keys (e.g., sk-, ghp_, xoxb-), AWS access keys, and passwords embedded in URLs. Any detected secrets are replaced with a placeholder before the message is sent to the user or back to the LLM.

Secrets are stored in an AES-256-GCM encrypted vault. The encryption key is derived from a user provided master password using the Argon2id key derivation function, ensuring high resistance to brute force attacks. This vault stores channel credentials, LLM API keys, and other sensitive configuration data.

SSRF protection is implemented by blocking the agent from making network requests to private or internal IP ranges (e.g., 10.0.0.0/8, 192.168.0.0/16). This prevents the agent from being used to probe the local network or access internal services. Finally, workspace sandboxing restricts filesystem operations to a designated directory, preventing path traversal attacks.

## Deployment Strategy

InterStellar is designed to be deployed as a Docker container, making it easy to run on a variety of platforms. A multi stage Dockerfile is used to keep the final image size small. The first stage uses the golang:1.23-alpine image to build the binary, and the final stage uses a minimal alpine:latest image to run it.

A typical deployment uses Docker Compose to manage the InterStellar daemon and its dependencies. Named volumes are used to persist the workspace files, the secret vault, the MEMORY.md file, and the WhatsApp session data. The container includes a health check endpoint at /health, which the Docker daemon can use to monitor the service's status.

The system supports graceful shutdown, catching SIGINT and SIGTERM signals to ensure that all active sessions are saved and connections are closed properly. Configuration can be provided via a YAML file or environment variables, allowing for flexible setup in different environments. For a complete development setup, an optional OpenCode sidecar container can be added to the Compose file.

## What Was NOT Adopted

During our research, we evaluated several patterns that were ultimately excluded from the first version of InterStellar.

| Pattern | Source | Reason for Exclusion |
|---------|--------|----------------------|
| SQLite + Vector Memory | OpenClaw | Too complex for version 1. MEMORY.md provides sufficient functionality with less overhead. |
| ACP Protocol | OpenClaw | We prefer the standardized Model Context Protocol (MCP) for better interoperability. |
| MongoDB Persistence | Rowboat | Adds a heavy database dependency that contradicts our "lightweight daemon" goal. |
| Redis for SSE | Rowboat | Unnecessary complexity for a single binary deployment. |
| goja (JS) Hooks | nanobot-ai | Adds a JavaScript dependency and runtime overhead to a pure Go project. |
| SvelteKit UI | nanobot-ai | The focus for v1 is on messaging channels, not a web interface. |
| Heartbeat System | HKUDS/nanobot | Proactive tasks are not a priority for the initial release. |
| PostgreSQL | GoClaw | We aim for zero database dependencies to simplify deployment and maintenance. |
| Per-agent Containers | Praktor | Too much resource overhead for a personal server use case. |
| NATS Message Bus | Praktor | Go's built-in channels and synchronization primitives are sufficient for our needs. |
| Langchaingo | Ecosystem | We prefer direct control over the agent loop using the lightweight go-openai client. |

## References

- **OpenClaw**: https://github.com/openclaw/openclaw
- **Rowboat**: https://github.com/rowboat-ai/rowboat
- **nanobot-ai/nanobot**: https://github.com/nanobot-ai/nanobot
- **HKUDS/nanobot**: https://github.com/hkuds/nanobot
- **GoClaw**: https://github.com/goclaw/goclaw
- **Praktor**: https://github.com/praktor/praktor
- **go-openai**: https://github.com/sashabaranov/go-openai
- **mcp-go**: https://github.com/mark3labs/mcp-go
- **OpenCode**: https://github.com/opencode/opencode
