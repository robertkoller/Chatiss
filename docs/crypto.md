# Cryptography

## Identity Keys

Each user's identity is derived deterministically from their passphrase using Argon2id:

```
seed = Argon2id(passphrase, salt="chatiss-identity-salting", t=1, m=64MB, p=4, len=32)
private_key = X25519(seed)
public_key  = X25519_public(private_key)
```

Argon2id is a memory-hard key derivation function. The 64 MB memory cost makes brute-forcing passphrases expensive.

The same passphrase always produces the same keypair. This means the user's identity is portable — they can log in from any device with just their passphrase and get the same keys.

## Session Key Exchange

When two peers connect, they perform an X25519 ECDH key exchange:

```
shared_secret = ECDH(my_private_key, their_public_key)
```

Both sides compute the same `shared_secret` without it ever being transmitted. This is the basis of forward secrecy at the session level.

The `shared_secret` is used directly as the AES-256-GCM encryption key for all packets in that session.

## Message Encryption (live sessions)

Each message is encrypted with the session's shared secret:

```
nonce      = random(12 bytes)
ciphertext = AES-256-GCM(key=shared_secret, nonce=nonce, plaintext=message)
packet     = header || nonce || ciphertext
```

AES-GCM provides authenticated encryption — the receiver detects any tampering and drops the packet.

## Offline Message Encryption (mailbox)

When the recipient is offline, the sender fetches their public key from the STUN server and encrypts the message for them directly:

```
shared_key = SHA-256(ECDH(sender_private, recipient_public))
nonce      = random(12 bytes)
ciphertext = AES-256-GCM(key=shared_key, nonce=nonce, plaintext=message)
```

The server stores only `(ciphertext, nonce)`. It cannot decrypt the message. When the recipient logs in, they compute:

```
shared_key = SHA-256(ECDH(recipient_private, sender_public))
plaintext  = AES-256-GCM-Decrypt(key=shared_key, nonce=nonce, ciphertext=ciphertext)
```

## Local Message Store

Messages on disk are encrypted with a key derived from the passphrase:

```
db_key = Argon2id(passphrase, salt="chatiss-db-v1", t=1, m=64MB, p=4, len=32)
stored = nonce || AES-256-GCM(key=db_key, nonce=nonce, plaintext=message_content)
```

A different salt is used so the database key is independent from the identity key.

## Mailbox Authentication

Each user has a mailbox bearer token derived from their identity seed:

```
mailbox_token = hex(SHA-256(identity_seed || "chatiss-mailbox-token-v1"))
```

The server stores `SHA-256(mailbox_token)`. The client authenticates by sending the plaintext token; the server hashes it and compares. This is equivalent to a password hash — the server never sees the raw token stored on disk.

## What the Server Knows

| Server | Knows | Does NOT know |
|---|---|---|
| STUN | Username, public key, UDP address | Passphrase, private key, message content |
| TURN | IP addresses of relay session | Message content (it's encrypted) |
| Mailbox | Username, encrypted message blobs, token hash | Passphrase, private key, plaintext messages |
