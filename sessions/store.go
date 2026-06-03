package sessions

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
	_ "modernc.org/sqlite"
)

var dbSalt = []byte("chatiss-db-v1")

type Message struct {
	Peer      string
	Content   string
	Timestamp uint32
	Outgoing  bool
}

type MessageStore struct {
	db  *sql.DB
	key []byte
}

// Creates the database and encrypting for it
func OpenMessageStore(passphrase string) (*MessageStore, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	directory = filepath.Join(directory, "Chatiss")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", filepath.Join(directory, "messages.db"))
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		peer      TEXT    NOT NULL,
		content   BLOB    NOT NULL,
		timestamp INTEGER NOT NULL,
		outgoing  INTEGER NOT NULL
	)`)
	if err != nil {
		return nil, err
	}

	key := argon2.IDKey([]byte(passphrase), dbSalt, 1, 64*1024, 4, 32)
	return &MessageStore{db: db, key: key}, nil
}

// takes a message and encrypts it then stores in the db
func (store *MessageStore) Save(message Message) error {
	encrypted, err := store.encrypt([]byte(message.Content))
	if err != nil {
		return err
	}
	outgoing := 0
	if message.Outgoing {
		outgoing = 1
	}
	_, err = store.db.Exec(
		`INSERT INTO messages (peer, content, timestamp, outgoing) VALUES (?, ?, ?, ?)`,
		message.Peer, encrypted, message.Timestamp, outgoing,
	)
	return err
}

// Loads the user's messages, most recent first, in pages of limit size.
// Results are reversed before returning so oldest appears first in the slice.
func (store *MessageStore) Load(peer string, limit, offset int) ([]Message, error) {
	rows, err := store.db.Query(
		`SELECT content, timestamp, outgoing FROM messages
		 WHERE peer = ? ORDER BY timestamp DESC LIMIT ? OFFSET ?`,
		peer, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var encrypted []byte
		var timestamp uint32
		var outgoing int
		if err := rows.Scan(&encrypted, &timestamp, &outgoing); err != nil {
			return nil, err
		}
		plaintext, err := store.decrypt(encrypted)
		if err != nil {
			return nil, err
		}
		messages = append(messages, Message{
			Peer:      peer,
			Content:   string(plaintext),
			Timestamp: timestamp,
			Outgoing:  outgoing == 1,
		})
	}
	// Reverse so oldest message is first in the slice.
	for x, y := 0, len(messages)-1; x < y; x, y = x+1, y-1 {
		messages[x], messages[y] = messages[y], messages[x]
	}
	return messages, nil
}

// Closes off the db
func (store *MessageStore) Close() error {
	return store.db.Close()
}

// PeerKey converts a raw public key to a hex string for use as a peer identifier.
func PeerKey(pubKeyBytes []byte) string {
	return hex.EncodeToString(pubKeyBytes)
}

// Encrypts the plaintext for adding into the db
func (store *MessageStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(store.key)
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
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypts the text from the db for client view
func (store *MessageStore) decrypt(encrypted []byte) ([]byte, error) {
	block, err := aes.NewCipher(store.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(encrypted) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := encrypted[:gcm.NonceSize()], encrypted[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
