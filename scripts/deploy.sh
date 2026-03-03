#!/usr/bin/env bash
set -euo pipefail

# ============================================================================
# BlackCat Deploy Script
# Builds on VM, installs binary, deploys systemd services, verifies health.
# Usage: bash scripts/deploy.sh [--no-push]
# ============================================================================

# --- Colors & helpers -------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info()  { echo -e "${CYAN}[deploy]${NC} $*"; }
ok()    { echo -e "${GREEN}[deploy] ✓${NC} $*"; }
warn()  { echo -e "${YELLOW}[deploy] ⚠${NC} $*"; }
fail()  { echo -e "${RED}[deploy] ✗${NC} $*"; exit 1; }

# --- Parse flags ------------------------------------------------------------
NO_PUSH=false
for arg in "$@"; do
  case "$arg" in
    --no-push) NO_PUSH=true ;;
    --help|-h)
      echo "Usage: bash scripts/deploy.sh [--no-push]"
      echo ""
      echo "Flags:"
      echo "  --no-push   Skip git push step"
      echo "  --help      Show this help"
      exit 0
      ;;
    *) fail "Unknown flag: $arg" ;;
  esac
done

# ============================================================================
# Step 1: Load deploy configuration
# ============================================================================
info "Step 1: Loading deploy configuration..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/../deploy/deploy.env"

if [[ ! -f "$ENV_FILE" ]]; then
  fail "deploy/deploy.env not found!
  Copy the example and fill in your values:
    cp deploy/deploy.env.example deploy/deploy.env
    \$EDITOR deploy/deploy.env"
fi

# shellcheck source=/dev/null
source "$ENV_FILE"

# Validate required variables
for var in DEPLOY_HOST DEPLOY_USER DEPLOY_SSH_KEY DEPLOY_HOME DEPLOY_WORKDIR \
           DEPLOY_CONFIG_PATH BLACKCAT_BINARY OPENCODE_BINARY OPENCODE_PORT \
           VAULT_PASSPHRASE; do
  if [[ -z "${!var:-}" ]]; then
    fail "Required variable $var is not set in deploy/deploy.env"
  fi
done

ok "Configuration loaded (host=$DEPLOY_HOST, user=$DEPLOY_USER)"

# --- SSH/SCP helpers --------------------------------------------------------
SSH_CMD="ssh -i $DEPLOY_SSH_KEY -o StrictHostKeyChecking=no $DEPLOY_USER@$DEPLOY_HOST"
SCP_CMD="scp -i $DEPLOY_SSH_KEY -o StrictHostKeyChecking=no"

# ============================================================================
# Step 2: Push local git changes
# ============================================================================
info "Step 2: Pushing local git changes..."

if [[ "$NO_PUSH" == "true" ]]; then
  warn "Skipped (--no-push flag)"
else
  CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
  if git rev-parse --verify "origin/$CURRENT_BRANCH" >/dev/null 2>&1; then
    git push
    ok "Pushed branch '$CURRENT_BRANCH' to origin"
  else
    warn "Branch '$CURRENT_BRANCH' has no remote tracking — skipping push"
  fi
fi

# ============================================================================
# Step 3: Pull latest code on VM
# ============================================================================
info "Step 3: Pulling latest code on VM..."

$SSH_CMD "cd $DEPLOY_WORKDIR && git pull"

ok "Code updated on VM"

# ============================================================================
# Step 4: Build binary on VM
# ============================================================================
info "Step 4: Building binary on VM..."

$SSH_CMD "cd $DEPLOY_WORKDIR && CGO_ENABLED=1 /usr/local/go/bin/go build -tags fts5 -o blackcat ."

ok "Binary built successfully"

# ============================================================================
# Step 5: Install binary
# ============================================================================
info "Step 5: Installing binary to $BLACKCAT_BINARY..."

# Stop services first so the binary is not locked ("text file busy")
$SSH_CMD "sudo systemctl stop blackcat opencode 2>/dev/null || true; sleep 1"
$SSH_CMD "sudo cp $DEPLOY_WORKDIR/blackcat $BLACKCAT_BINARY"

ok "Binary installed"

# ============================================================================
# Step 6: Upload service files
# ============================================================================
info "Step 6: Uploading service file templates..."

DEPLOY_DIR="$SCRIPT_DIR/../deploy"
$SCP_CMD "$DEPLOY_DIR/blackcat.service" "$DEPLOY_USER@$DEPLOY_HOST:/tmp/blackcat.service"
$SCP_CMD "$DEPLOY_DIR/opencode.service" "$DEPLOY_USER@$DEPLOY_HOST:/tmp/opencode.service"

ok "Service files uploaded to /tmp/"

# ============================================================================
# Step 7: Substitute placeholders in service files
# ============================================================================
info "Step 7: Substituting placeholders in service files..."

$SSH_CMD << EOF
  # blackcat.service placeholders
  sed -i "s|__DEPLOY_USER__|$DEPLOY_USER|g" /tmp/blackcat.service
  sed -i "s|__DEPLOY_GROUP__|$DEPLOY_USER|g" /tmp/blackcat.service
  sed -i "s|__DEPLOY_HOME__|$DEPLOY_HOME|g" /tmp/blackcat.service
  sed -i "s|__BLACKCAT_BINARY__|$BLACKCAT_BINARY|g" /tmp/blackcat.service
  sed -i "s|__DEPLOY_CONFIG_PATH__|$DEPLOY_CONFIG_PATH|g" /tmp/blackcat.service
  sed -i "s|__VAULT_PASSPHRASE__|$VAULT_PASSPHRASE|g" /tmp/blackcat.service

  # opencode.service placeholders
  sed -i "s|__DEPLOY_USER__|$DEPLOY_USER|g" /tmp/opencode.service
  sed -i "s|__DEPLOY_GROUP__|$DEPLOY_USER|g" /tmp/opencode.service
  sed -i "s|__DEPLOY_HOME__|$DEPLOY_HOME|g" /tmp/opencode.service
  sed -i "s|__BLACKCAT_BINARY__|$BLACKCAT_BINARY|g" /tmp/opencode.service
  sed -i "s|__OPENCODE_PORT__|$OPENCODE_PORT|g" /tmp/opencode.service
  sed -i "s|__OPENCODE_BINARY__|$OPENCODE_BINARY|g" /tmp/opencode.service
EOF

ok "Placeholders substituted"

# ============================================================================
# Step 8: Install service files
# ============================================================================
info "Step 8: Installing service files to /etc/systemd/system/..."

$SSH_CMD << 'INSTALL_EOF'
  sudo cp /tmp/blackcat.service /etc/systemd/system/blackcat.service
  sudo cp /tmp/opencode.service /etc/systemd/system/opencode.service
  rm -f /tmp/blackcat.service /tmp/opencode.service
INSTALL_EOF

ok "Service files installed"

# ============================================================================
# Step 9: Reload and restart services
# ============================================================================
info "Step 9: Reloading systemd and restarting services..."

$SSH_CMD "sudo systemctl daemon-reload && sudo systemctl restart opencode blackcat"

ok "Services restarted"

# ============================================================================
# Step 10: Verify health
# ============================================================================
info "Step 10: Verifying health..."

HEALTH_URL="http://$DEPLOY_HOST:8080/health"
HEALTH_OK=false

for i in $(seq 1 5); do
  info "  Health check attempt $i/5..."
  if curl -sf "$HEALTH_URL" >/dev/null 2>&1; then
    HEALTH_OK=true
    break
  fi
  sleep 2
done

if [[ "$HEALTH_OK" == "true" ]]; then
  ok "Health check passed ($HEALTH_URL)"
else
  fail "Health check failed after 10s — $HEALTH_URL not responding"
fi

# ============================================================================
echo ""
ok "Deploy complete! 🚀"
echo -e "  ${CYAN}Host:${NC}     $DEPLOY_HOST"
echo -e "  ${CYAN}Binary:${NC}   $BLACKCAT_BINARY"
echo -e "  ${CYAN}Services:${NC} blackcat, opencode"
echo -e "  ${CYAN}Health:${NC}   $HEALTH_URL"
