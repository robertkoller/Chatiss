# Server Infrastructure

All three servers run on a single DigitalOcean droplet at `178.128.151.84`.

## STUN Server (`server/stun/`)

The STUN server is Chatiss's own custom signalling server — it is NOT the standard STUN protocol.

**What it does:**
- Lets clients register a username + public key + UDP address
- Lets clients look up another user by username
- When two users look each other up, it sends a `connect` signal to both, assigning roles (initiator/responder) and preventing duplicate signals via a `paired` map
- Stores public keys so offline message encryption can work (`get_pubkey`)
- Forwards TURN relay addresses between peers

**Protocol:** Custom JSON over UDP on port `13478`.

**Key correctness property:** The `paired` map ensures each pair gets exactly one set of connect signals, even if both clients look each other up simultaneously. The pair entry is cleared when either client disconnects, so reconnecting works correctly.

## TURN Server (`server/turn/`)

The TURN server is a simple TCP relay. It is used when QUIC hole punching fails (strict NAT, symmetric NAT, or firewall).

**What it does:**
- Accepts `allocate` requests from the initiator, creating a relay session on a random high port
- Accepts `join` requests from the responder, connecting them to the same relay session
- Forwards raw bytes between the two TCP connections

**Protocol:** JSON control messages over TCP on port `13479` (or `443` for firewall bypass), then raw byte relay.

**Security note:** The TURN server sees the encrypted bytes of the session. It cannot decrypt them since it does not have the session keys.

## Mailbox Server (`server/mailbox/`)

The mailbox stores encrypted messages for offline delivery.

**What it does:**
- Stores encrypted message blobs when the recipient is offline
- Returns pending messages when the recipient logs in
- Deletes acknowledged messages

**Protocol:** JSON over HTTP on port `8080`.

**Auth:** Each user has a deterministic bearer token derived from their passphrase (see [crypto.md](crypto.md)). The server stores only the SHA-256 hash of the token.

**API:**

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/users` | None | Register / update token hash |
| POST | `/messages` | Sender | Upload encrypted message |
| GET | `/messages` | Recipient | Fetch pending messages |
| DELETE | `/messages/{id}` | Recipient | Acknowledge delivery |

## Deployment

```bash
# Deploy code and build server binaries on the droplet
./scripts/deploy.sh

# Start all three servers
./scripts/start.sh   # → logs/stun.log, logs/turn.log, logs/mailbox.log

# Stop all servers
./scripts/stop.sh
```

Binaries are built on the droplet (not cross-compiled locally) to avoid libc version issues.
