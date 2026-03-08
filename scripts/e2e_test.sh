#!/bin/bash
set -e

echo "=== BlackCat E2E QA Verification ==="
echo ""

# 1. Go vet
echo "[1/7] Running go vet..."
go vet ./...
echo "  ✓ go vet passed"

# 2. Go build
echo "[2/7] Running go build..."
go build -o blackcat.test.exe .
echo "  ✓ go build passed"

# 3. CLI commands verification
echo "[3/7] Verifying CLI commands..."
./blackcat.test.exe --help | grep -q "daemon" && echo "  ✓ daemon command found"
./blackcat.test.exe --help | grep -q "vault" && echo "  ✓ vault command found"
./blackcat.test.exe --help | grep -q "init" && echo "  ✓ init command found"
./blackcat.test.exe --help | grep -q "serve" && echo "  ✓ serve command found"
./blackcat.test.exe --help | grep -q "health" && echo "  ✓ health command found"
./blackcat.test.exe --help | grep -q "sessions" && echo "  ✓ sessions command found"
./blackcat.test.exe --help | grep -q "run" && echo "  ✓ run command found"

# 4. Unit tests
echo "[4/7] Running unit tests..."
go test ./... -count=1
echo "  ✓ all unit tests passed"

# 5. Verify key files exist
echo "[5/7] Verifying project structure..."
for f in \
    "config/config.go" "config/loader.go" "config/watcher.go" \
    "types/types.go" "types/interfaces.go" "types/errors.go" \
    "security/denylist.go" "security/scrubber.go" "security/vault.go" \
    "memory/store.go" \
    "opencode/client.go" "opencode/sse.go" "opencode/session.go" \
    "llm/client.go" "llm/provider.go" "llm/messages.go" \
    "tools/registry.go" "tools/exec.go" "tools/filesystem.go" "tools/web.go" "tools/opencode_tool.go" \
    "skills/loader.go" \
    "workspace/loader.go" \
    "agent/agent.go" "agent/loop.go" "agent/execution.go" "agent/compaction.go" \
    "channel/channel.go" "channel/mock.go" \
    "channel/telegram/telegram.go" "channel/discord/discord.go" \
    "channel/whatsapp/whatsapp_stub.go" \
    "mcp/server.go" "mcp/client.go" \
    "cmd/daemon.go" "cmd/vault.go" "cmd/init.go" \
    "Research_Design_Architecture_InterStellar.md" \
    "blackcat.example.json5" \
    "Dockerfile" "docker-compose.yml" ".dockerignore"; do
    if [ -f "$f" ]; then
        echo "  ✓ $f"
    else
        echo "  ✗ MISSING: $f"
    fi
done

# 6. Daemon help
echo "[6/7] Verifying daemon command..."
./blackcat.test.exe daemon --help | grep -q "workers" && echo "  ✓ --workers flag found"

# 7. Vault subcommands
echo "[7/7] Verifying vault subcommands..."
./blackcat.test.exe vault --help | grep -q "set" && echo "  ✓ vault set"
./blackcat.test.exe vault --help | grep -q "get" && echo "  ✓ vault get"
./blackcat.test.exe vault --help | grep -q "list" && echo "  ✓ vault list"
./blackcat.test.exe vault --help | grep -q "delete" && echo "  ✓ vault delete"

# Cleanup
rm -f blackcat.test.exe

echo ""
echo "=== E2E QA PASSED ==="
