// Package mailbox provides the client-side API for the Chatiss mailbox server.
// It handles registration, storing encrypted offline messages, fetching pending
// messages, and acknowledging delivery.
package mailbox

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client talks to the mailbox HTTP server on behalf of a single user.
type Client struct {
	baseURL  string
	username string
	token    string       // bearer token (plaintext — hashed before storage)
	http     *http.Client
}

// PendingMessage is a message retrieved from the mailbox, still encrypted.
type PendingMessage struct {
	ID         string
	From       string
	Ciphertext []byte
	Nonce      []byte
	CreatedAt  int64
}

func NewClient(baseURL, username, token string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		token:    token,
		http:     &http.Client{},
	}
}

// Register upserts this user's auth credentials on the mailbox server.
// Call once at startup after STUN registration.
func (c *Client) Register() error {
	h := sha256.Sum256([]byte(c.token))
	body, _ := json.Marshal(map[string]string{
		"username":   c.username,
		"token_hash": hex.EncodeToString(h[:]),
	})
	resp, err := c.post("/users", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("register: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// StorePending encrypts plaintext for the recipient and uploads it to the mailbox.
// recipientPub is the recipient's X25519 public key (obtained from STUN get_pubkey).
func (c *Client) StorePending(senderPriv *ecdh.PrivateKey, recipientPub *ecdh.PublicKey, recipientUsername, plaintext string) error {
	ciphertext, nonce, err := encryptForRecipient(senderPriv, recipientPub, []byte(plaintext))
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	body, _ := json.Marshal(map[string]string{
		"to":         recipientUsername,
		"from":       c.username,
		"ciphertext": hex.EncodeToString(ciphertext),
		"nonce":      hex.EncodeToString(nonce),
	})
	resp, err := c.post("/messages", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("store: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// FetchPending returns all pending messages for this user from the mailbox.
func (c *Client) FetchPending() ([]PendingMessage, error) {
	req, _ := http.NewRequest(http.MethodGet, c.baseURL+"/messages", nil)
	c.setAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch: unexpected status %d", resp.StatusCode)
	}

	var wire []struct {
		ID         string `json:"id"`
		From       string `json:"from"`
		Ciphertext string `json:"ciphertext"`
		Nonce      string `json:"nonce"`
		CreatedAt  int64  `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return nil, err
	}

	out := make([]PendingMessage, 0, len(wire))
	for _, w := range wire {
		ct, err := hex.DecodeString(w.Ciphertext)
		if err != nil {
			continue
		}
		nonce, err := hex.DecodeString(w.Nonce)
		if err != nil {
			continue
		}
		out = append(out, PendingMessage{
			ID:         w.ID,
			From:       w.From,
			Ciphertext: ct,
			Nonce:      nonce,
			CreatedAt:  w.CreatedAt,
		})
	}
	return out, nil
}

// Ack tells the mailbox server a message was received and can be deleted.
func (c *Client) Ack(id string) error {
	req, _ := http.NewRequest(http.MethodDelete, c.baseURL+"/messages/"+id, nil)
	c.setAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("ack: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// DecryptMessage decrypts a pending message using the recipient's private key
// and the sender's public key (both sides of ECDH produce the same shared key).
func DecryptMessage(recipientPriv *ecdh.PrivateKey, senderPub *ecdh.PublicKey, msg PendingMessage) (string, error) {
	plain, err := decryptFromSender(recipientPriv, senderPub, msg.Ciphertext, msg.Nonce)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (c *Client) post(path string, body []byte) (*http.Response, error) {
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)
	return c.http.Do(req)
}

func (c *Client) setAuth(req *http.Request) {
	req.Header.Set("X-Username", c.username)
	req.Header.Set("Authorization", "Bearer "+c.token)
}

// encryptForRecipient encrypts plaintext using ECDH(senderPriv, recipientPub) as the AES-256-GCM key.
func encryptForRecipient(senderPriv *ecdh.PrivateKey, recipientPub *ecdh.PublicKey, plaintext []byte) (ciphertext, nonce []byte, err error) {
	shared, err := senderPriv.ECDH(recipientPub)
	if err != nil {
		return nil, nil, err
	}
	key := sha256.Sum256(shared) // stretch to 32 bytes
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nonce, nil
}

// decryptFromSender mirrors encryptForRecipient for the recipient side.
func decryptFromSender(recipientPriv *ecdh.PrivateKey, senderPub *ecdh.PublicKey, ciphertext, nonce []byte) ([]byte, error) {
	shared, err := recipientPriv.ECDH(senderPub)
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256(shared)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}
