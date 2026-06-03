package protocol

import (
	"crypto/ecdh"
)

// Struct for our keys
type LocalIdentity struct {
	PrivateKey *ecdh.PrivateKey
	PublicKey  *ecdh.PublicKey
}

// Create and store the key
func Login(passphrase string) (*LocalIdentity, error) {
	private, public, err := DeriveKeyPairFromPassphrase(passphrase)
	if err != nil {
		return nil, err
	}
	return &LocalIdentity{PrivateKey: private, PublicKey: public}, nil
}
