package protocol

import (
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"golang.org/x/crypto/argon2"
)

// fixed domain seperator so that a random other app doesnt get the same phrase as us
var identitySalt = []byte("chatiss-identity-salting")

// DeriveIdentitySeed returns the raw Argon2 seed for a passphrase.
// Call this once and pass the result to both DeriveKeyPairFromSeed and DeriveMailboxToken.
func DeriveIdentitySeed(passphrase string) []byte {
	return argon2.IDKey([]byte(passphrase), identitySalt, 1, 64*1024, 4, 32)
}

// This takes in a user inputted passphrase and outputs the private/public keys
func DeriveKeyPairFromPassphrase(passphrase string) (*ecdh.PrivateKey, *ecdh.PublicKey, error) {
	return DeriveKeyPairFromSeed(DeriveIdentitySeed(passphrase))
}

// DeriveKeyPairFromSeed derives an X25519 keypair from a raw 32-byte seed.
func DeriveKeyPairFromSeed(seed []byte) (*ecdh.PrivateKey, *ecdh.PublicKey, error) {
	privateKey, err := ecdh.X25519().NewPrivateKey(seed)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, privateKey.PublicKey(), nil
}

// DeriveMailboxToken derives a bearer token for mailbox authentication from an
// identity seed. The token is a different domain from the identity keypair so
// that the two secrets never overlap.
func DeriveMailboxToken(seed []byte) string {
	h := sha256.New()
	h.Write(seed)
	h.Write([]byte("chatiss-mailbox-token-v1"))
	return hex.EncodeToString(h.Sum(nil))
}

// Derives a the sharedSecret from a A's public and B's private
// which will be identical to deriving from A's private and B's public.
// We then hash it to use it as our ID
func DeriveSessionInfo(myPrivKey *ecdh.PrivateKey, theirPubKey *ecdh.PublicKey) (uint32, []byte, error) {
	sharedSecret, err := myPrivKey.ECDH(theirPubKey)
	if err != nil {
		return 0, nil, err
	}
	hash := sha256.Sum256(sharedSecret)
	return binary.BigEndian.Uint32(hash[:4]), sharedSecret, nil
}
