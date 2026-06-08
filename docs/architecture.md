# Chatiss Architecture

Chatiss is a peer-to-peer encrypted messaging app. Messages travel directly between clients when both are online. When one party is offline, messages are stored encrypted on a relay server and delivered on next login.

## System Components

```
┌──────────────────────────────────────────────────────────────┐
│  Client A (macOS .app)          Client B (macOS .app)        │
│  ┌──────────┐ ┌────────┐        ┌──────────┐ ┌────────┐     │
│  │  React   │ │  Go    │        │  React   │ │  Go    │     │
│  │  (UI)    │◄►│service │        │  (UI)    │◄►│service │     │
│  └──────────┘ └───┬────┘        └──────────┘ └───┬────┘     │
└───────────────────┼──────────────────────────────┼──────────┘
                    │                              │
         ┌──────────┼──────────────────────────────┼──────────┐
         │ Droplet  │                              │          │
         │  ┌───────▼──────┐  ┌─────────────┐     │          │
         │  │  STUN Server │  │TURN Server  │     │          │
         │  │  (discovery) │  │(relay conn) │     │          │
         │  └──────────────┘  └─────────────┘     │          │
         │  ┌──────────────────────────────────────┘          │
         │  │  Mailbox Server                                  │
         │  │  (offline message storage)                       │
         │  └──────────────────────────────────────────────────┘
         └─────────────────────────────────────────────────────┘
```

## Connection Flow

1. **Discovery** — Both clients register with the STUN server on startup, advertising their UDP address and public key. To start a chat, the initiating client sends a `lookup` to STUN.

2. **Hole punching** — The STUN server sends a `connect` signal to both parties, assigning one as initiator and one as responder. Both punch a hole through their NAT by sending UDP packets to each other for 600ms.

3. **QUIC** — The initiator dials a QUIC connection over the punched hole. QUIC is a modern transport protocol that multiplexes streams over UDP.

4. **TURN fallback** — If QUIC fails (strict NAT, firewall), the initiator allocates a relay slot on the TURN server and forwards its address to the responder via STUN. Both connect to the relay over TCP.

5. **Handshake** — The initiator sends its X25519 public key and username. The responder replies with the same. Both derive the same shared secret via ECDH. All subsequent packets are AES-256-GCM encrypted with this secret.

6. **Offline delivery** — If the recipient is not online, the sender fetches their public key from STUN, encrypts the message locally, and uploads the ciphertext to the Mailbox server. On next login the recipient downloads and decrypts it.

## Packages

| Package | Role |
|---|---|
| `app/` | Persistent background service + Wails bridge to React UI |
| `protocol/` | Wire protocol: packet format, encryption, handshake, session management |
| `transport/` | QUIC dial/listen, UDP socket management, hole punching |
| `sessions/` | SQLite store for messages and contacts |
| `mailbox/` | HTTP client for the mailbox server |
| `server/stun/` | STUN server + client (peer discovery, pubkey exchange) |
| `server/turn/` | TURN server (TCP relay fallback) |
| `server/mailbox/` | Mailbox HTTP server + SQLite store |
| `cmd/app/` | Wails desktop app entry point (not used — main.go at root) |
| `cmd/server/` | STUN server binary |
| `cmd/turn/` | TURN server binary |
| `cmd/mailbox/` | Mailbox server binary |
| `cmd/client/` | Legacy CLI client (kept for debugging) |
| `frontend/` | React UI (Vite + TypeScript) |
