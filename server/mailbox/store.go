package mailbox

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type PendingMessage struct {
	ID         string
	Sender     string
	Recipient  string
	Ciphertext []byte
	Nonce      []byte
	CreatedAt  int64
}

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			username   TEXT PRIMARY KEY,
			token_hash TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS messages (
			id         TEXT    PRIMARY KEY,
			sender     TEXT    NOT NULL,
			recipient  TEXT    NOT NULL,
			ciphertext BLOB    NOT NULL,
			nonce      BLOB    NOT NULL,
			created_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_messages_recipient ON messages(recipient);
	`)
	return err
}

// RegisterUser upserts a user's token hash. If the username already exists with
// a different token hash the old record is overwritten — same semantics as STUN
// re-registration after a passphrase change.
func (s *Store) RegisterUser(username, tokenHash string) error {
	_, err := s.db.Exec(
		`INSERT INTO users (username, token_hash) VALUES (?, ?)
		 ON CONFLICT(username) DO UPDATE SET token_hash = excluded.token_hash`,
		username, tokenHash,
	)
	return err
}

// Authenticate returns true when the provided bearer token matches the stored
// hash for the given username.
func (s *Store) Authenticate(username, token string) bool {
	h := sha256.Sum256([]byte(token))
	got := hex.EncodeToString(h[:])
	var stored string
	err := s.db.QueryRow(`SELECT token_hash FROM users WHERE username = ?`, username).Scan(&stored)
	return err == nil && got == stored
}

// StoreMessage saves an encrypted message for later delivery.
func (s *Store) StoreMessage(id, sender, recipient string, ciphertext, nonce []byte) error {
	_, err := s.db.Exec(
		`INSERT INTO messages (id, sender, recipient, ciphertext, nonce, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, sender, recipient, ciphertext, nonce, time.Now().Unix(),
	)
	return err
}

// FetchMessages returns all pending messages for a recipient, oldest first.
func (s *Store) FetchMessages(recipient string) ([]PendingMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, sender, ciphertext, nonce, created_at FROM messages
		 WHERE recipient = ? ORDER BY created_at ASC`,
		recipient,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []PendingMessage
	for rows.Next() {
		var m PendingMessage
		m.Recipient = recipient
		if err := rows.Scan(&m.ID, &m.Sender, &m.Ciphertext, &m.Nonce, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// AckMessage deletes a message only if it belongs to the given recipient.
func (s *Store) AckMessage(id, recipient string) error {
	res, err := s.db.Exec(
		`DELETE FROM messages WHERE id = ? AND recipient = ?`, id, recipient,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("message not found")
	}
	return nil
}
