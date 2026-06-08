#!/usr/bin/env bash
# Run on the droplet to stop all server processes, clear logs, and wipe the mailbox.

cd /root/chatiss

pkill -f chatiss-server  2>/dev/null && echo "STUN server stopped."    || echo "STUN server was not running."
pkill -f chatiss-turn    2>/dev/null && echo "TURN server stopped."    || echo "TURN server was not running."
pkill -f chatiss-mailbox 2>/dev/null && echo "Mailbox server stopped." || echo "Mailbox server was not running."

# Truncate logs so the next run starts clean.
mkdir -p logs
truncate -s 0 logs/stun.log    && echo "STUN log cleared."
truncate -s 0 logs/turn.log    && echo "TURN log cleared."
truncate -s 0 logs/mailbox.log && echo "Mailbox log cleared."

# Wipe the mailbox SQLite database.
rm -f ~/chatiss-mailbox.db && echo "Mailbox database wiped."
