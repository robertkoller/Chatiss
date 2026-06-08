# Wire Protocol

All packets between peers share a common binary format. After the handshake, every packet is encrypted.

## Packet Layout

```
┌────────┬─────────┬────────────┬───────┬───────────┬───────────┬─────────────────┐
│ Magic  │ Version │ PacketType │ Flags │ SessionID │ PayloadLen│ Payload         │
│ 1 byte │ 1 byte  │ 1 byte     │1 byte │ 4 bytes   │ 4 bytes   │ variable        │
└────────┴─────────┴────────────┴───────┴───────────┴───────────┴─────────────────┘
```

- **Magic**: `0xBC` — identifies a Chatiss packet, rejects garbage data
- **Version**: `0x01`
- **PacketType**: see table below
- **Flags**: reserved, currently `0x00`
- **SessionID**: 32-bit session identifier derived from the ECDH shared secret
- **PayloadLen**: length of the payload in bytes
- **Payload**: type-specific data, encrypted after handshake

## Packet Types

| Type | Value | Direction | Description |
|---|---|---|---|
| `Handshake` | `0x01` | Initiator → Responder | Public key + username |
| `HandshakeAck` | `0x02` | Responder → Initiator | Public key + username |
| `Text` | `0x03` | Both | Encrypted text message |
| `TextAck` | `0x04` | Both | Delivery acknowledgement |
| `CallStart` | `0x05` | Both | Begin a voice call |
| `CallAudio` | `0x06` | Both | Raw audio frame |
| `CallEnd` | `0x07` | Both | End a voice call |
| `FileStart` | `0x08` | Both | File metadata (name, size, chunk count) |
| `FileChunk` | `0x09` | Both | One chunk of a file transfer |
| `FileEnd` | `0x0A` | Both | File transfer complete |
| `Ping` | `0x0C` | Both | Keepalive |
| `Pong` | `0x0D` | Both | Keepalive reply |
| `Disconnect` | `0xFE` | Both | Graceful session close |

## Handshake Flow

```
Initiator                        Responder
    │                                │
    │──── Handshake (pubkey, user) ──►│
    │                                │  ← derives shared secret
    │◄─── HandshakeAck (pubkey, user)─│
    │  ← derives shared secret       │
    │                                │
    │  (all subsequent packets are   │
    │   AES-256-GCM encrypted)       │
```

The shared secret is `ECDH(my_private, their_public)` — both sides derive the same 32-byte secret. The `SessionID` is the first 4 bytes of `SHA-256(shared_secret)`.

## Text Message Encryption

`payload = nonce (12 bytes) || AES-256-GCM(key=shared_secret, nonce, plaintext)`

The nonce is random for each message. AES-GCM provides both confidentiality and integrity — a tampered packet will fail decryption and be dropped.

## File Transfer

Files are split into 32 KB chunks. Each chunk is sent as a separate `FileChunk` packet with a sequential index. The receiver reassembles them in order.

```
FileStart  →  FileChunk[0]  →  FileChunk[1]  →  ...  →  FileEnd
```
