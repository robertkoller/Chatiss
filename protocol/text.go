package protocol

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// Encrypts the text and creates the packet for sending
func CreateText(content string, session *Session) ([]byte, error) {
	block, err := aes.NewCipher(session.SharedSecret)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	encrypted := gcm.Seal(nonce, nonce, []byte(content), nil)
	packet := createPacket(Version1, TypeText, FlagEmpty, session.ID, encrypted)
	return packetToBytes(packet), nil
}

// Decrypts a received text packet and returns the plaintext content.
// The caller is responsible for saving the message to the store.
func processRetrievedText(packet Packet, session *Session) (string, error) {
	block, err := aes.NewCipher(session.SharedSecret)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	encrypted := packet.Payload
	if len(encrypted) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := encrypted[:gcm.NonceSize()], encrypted[gcm.NonceSize():]
	decrypted, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}
