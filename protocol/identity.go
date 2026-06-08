package protocol

import (
	"crypto/ecdh"
)

// Struct for our keys
type LocalIdentity struct {
	PrivateKey   *ecdh.PrivateKey
	PublicKey    *ecdh.PublicKey
	MailboxToken string // bearer token for mailbox API authentication
}

// Create and store the key
func Login(passphrase string) (*LocalIdentity, error) {
	seed := DeriveIdentitySeed(passphrase)
	private, public, err := DeriveKeyPairFromSeed(seed)
	if err != nil {
		return nil, err
	}
	return &LocalIdentity{
		PrivateKey:   private,
		PublicKey:    public,
		MailboxToken: DeriveMailboxToken(seed),
	}, nil
}
