#!/usr/bin/env bash
# Show the running status of all three Chatiss server processes.

cd /root/chatiss 2>/dev/null || true

GREEN='\033[32m'
RED='\033[31m'
YELLOW='\033[33m'
BOLD='\033[1m'
RESET='\033[0m'

check() {
    local label="$1"
    local pattern="$2"
    local port_check="$3"  # optional: "tcp:<port>" or "udp:<port>"

    pid=$(pgrep -f "$pattern" | head -1)

    if [ -z "$pid" ]; then
        printf "  ${RED}●${RESET} %-20s ${RED}not running${RESET}\n" "$label"
        return
    fi

    # Get uptime and memory from ps
    info=$(ps -p "$pid" -o pid=,etime=,rss= 2>/dev/null | awk '{
        pid=$1; etime=$2; rss=$3
        mb = rss/1024
        printf "PID %s  up %s  %.1f MB", pid, etime, mb
    }')

    # Check if the port is actually bound (optional, best-effort)
    port_status=""
    if [ -n "$port_check" ]; then
        proto=$(echo "$port_check" | cut -d: -f1)
        port=$(echo "$port_check" | cut -d: -f2)
        if ss -ln --"$proto" 2>/dev/null | grep -q ":$port "; then
            port_status=" ${GREEN}:$port bound${RESET}"
        else
            port_status=" ${YELLOW}:$port not bound?${RESET}"
        fi
    fi

    printf "  ${GREEN}●${RESET} %-20s ${GREEN}running${RESET}  %s%b\n" "$label" "$info" "$port_status"
}

check_http() {
    local url="$1"
    code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 2 "$url" 2>/dev/null)
    if [ "$code" = "401" ] || [ "$code" = "200" ] || [ "$code" = "204" ]; then
        printf "    ${GREEN}✓${RESET} HTTP responding (status $code)\n"
    else
        printf "    ${YELLOW}✗${RESET} HTTP not responding\n"
    fi
}

printf "\n${BOLD}Chatiss Server Status${RESET}\n"
printf "%s\n" "────────────────────────────────────────────────"

check "STUN"    "chatiss-server"  "udp:13478"
check "TURN"    "chatiss-turn"    "tcp:13479"
check "Mailbox" "chatiss-mailbox" "tcp:8080"
check_http "http://localhost:8080/messages"

printf "%s\n\n" "────────────────────────────────────────────────"
