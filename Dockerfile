# Stage 1: Build
FROM golang:1.25-bookworm AS builder

RUN apt-get update && apt-get install -y gcc && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=1 go build -o blackcat -ldflags="-s -w" .

# Stage 2: Runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates sqlite3 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/blackcat /usr/local/bin/blackcat
COPY workspace/ /app/workspace/
COPY blackcat.example.yaml /app/blackcat.example.yaml

# Create data directories
RUN mkdir -p /data/memory /data/vault /data/whatsapp /data/skills

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8081/health || exit 1

ENTRYPOINT ["blackcat"]
CMD ["daemon"]
