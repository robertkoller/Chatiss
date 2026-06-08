#!/usr/bin/env bash
# Run on the droplet to (re)start all three server processes.
# Safe to run when servers are already running — stops them first.

set -e
cd /root/chatiss
mkdir -p logs

# ── Stop any running instances ───────────────────────────────────────────────

stopped=0
if pkill -f chatiss-server  2>/dev/null; then echo "Stopped STUN server.";    stopped=1; fi
if pkill -f chatiss-turn    2>/dev/null; then echo "Stopped TURN server.";    stopped=1; fi
if pkill -f chatiss-mailbox 2>/dev/null; then echo "Stopped Mailbox server."; stopped=1; fi

# Give the OS a moment to release ports before we rebind them.
if [ $stopped -eq 1 ]; then sleep 1; fi

# ── Start servers ────────────────────────────────────────────────────────────

nohup ./bin/chatiss-server  > logs/stun.log    2>&1 & echo "STUN server started    (PID $!) → logs/stun.log"
nohup ./bin/chatiss-turn    > logs/turn.log    2>&1 & echo "TURN server started    (PID $!) → logs/turn.log"
nohup ./bin/chatiss-mailbox > logs/mailbox.log 2>&1 & echo "Mailbox server started (PID $!) → logs/mailbox.log"
