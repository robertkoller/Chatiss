#!/usr/bin/env bash
# deploy.sh — push code to the droplet and build server binaries.
# Usage: ./scripts/deploy.sh [droplet-ip]

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

DROPLET_IP="178.128.151.84"

if [ -n "${1:-}" ]; then
    if [[ "$1" == -* ]]; then
        echo "deploy.sh does not accept flags. Usage: ./deploy.sh [droplet-ip]"
        exit 1
    fi
    DROPLET_IP="$1"
fi

REMOTE_USER="root"
REMOTE_DIR="/root/chatiss"

# Socket in /tmp so it always exists and is writable.
# Include PID so parallel deploys don't collide.
CTL_SOCK="/tmp/chatiss-deploy-$$.sock"

# Use the SSH key if it exists; fall back to password auth if not.
SSH_KEY="${HOME}/.ssh/mosaic-droplet"
if [ -f "${SSH_KEY}" ]; then
    BASE_OPTS="-i ${SSH_KEY}"
else
    BASE_OPTS=""
fi

# All subsequent ssh/rsync calls share this multiplexed connection.
CTL_OPTS="-o ControlMaster=auto -o ControlPath=${CTL_SOCK} -o ControlPersist=60s"
SSH_OPTS="${BASE_OPTS} ${CTL_OPTS} -o StrictHostKeyChecking=accept-new"

# Clean up the master connection on exit (normal or error).
cleanup() {
    ssh -o "ControlPath=${CTL_SOCK}" -O exit "${REMOTE_USER}@${DROPLET_IP}" 2>/dev/null || true
    rm -f "${CTL_SOCK}"
}
trap cleanup EXIT

# ── Open master connection — only password/passphrase prompt in the script ──
echo "Connecting to ${DROPLET_IP}…"
ssh ${SSH_OPTS} -fN "${REMOTE_USER}@${DROPLET_IP}"
echo "Connected."
echo ""

echo "Deploying to ${REMOTE_USER}@${DROPLET_IP}:${REMOTE_DIR}"
echo ""

# ── Sync ────────────────────────────────────────────────────────────────────
echo "Syncing code…"
rsync -az --delete \
    --exclude='.git' \
    --exclude='bin/' \
    --exclude='*.log' \
    --exclude='*.pid' \
    --exclude='*.db' \
    -e "ssh ${SSH_OPTS}" \
    "${REPO_ROOT}/" "${REMOTE_USER}@${DROPLET_IP}:${REMOTE_DIR}/"
echo "✓ Code synced"
echo ""

# ── Build ────────────────────────────────────────────────────────────────────
echo "Building on server…"
ssh ${SSH_OPTS} "${REMOTE_USER}@${DROPLET_IP}" bash << 'EOF'
set -e
cd /root/chatiss
export PATH=$PATH:/usr/local/go/bin
mkdir -p bin
go build -o bin/chatiss-server  ./cmd/server/
go build -o bin/chatiss-turn    ./cmd/turn/
go build -o bin/chatiss-client  ./cmd/client/
go build -o bin/chatiss-mailbox ./cmd/mailbox/
echo "✓ Build complete"
EOF

echo ""
echo "Done. Run ./scripts/start.sh on the droplet to restart services."
