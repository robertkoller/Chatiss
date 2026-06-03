package protocol

import (
	"crypto/ecdh"
	"crypto/sha256"
	"encoding/binary"

	"golang.org/x/crypto/argon2"
)

// fixed domain seperator so that a random other app doesnt get the same phrase as us
var identitySalt = []byte("chatiss-identity-salting")

// This takes in a user inputted passphrase and outputs the private/public keys
func DeriveKeyPairFromPassphrase(passphrase string) (*ecdh.PrivateKey, *ecdh.PublicKey, error) {
	seed := argon2.IDKey([]byte(passphrase), identitySalt, 1, 64*1024, 4, 32)
	privateKey, err := ecdh.X25519().NewPrivateKey(seed)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, privateKey.PublicKey(), nil
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
