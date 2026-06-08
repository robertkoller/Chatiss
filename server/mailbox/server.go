package mailbox

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Server struct {
	store *Store
	mux   *http.ServeMux
}

func NewServer(store *Store) *Server {
	s := &Server{store: store, mux: http.NewServeMux()}
	s.mux.HandleFunc("POST /users", s.handleRegisterUser)
	s.mux.HandleFunc("POST /messages", s.handleStoreMessage)
	s.mux.HandleFunc("GET /messages", s.handleFetchMessages)
	s.mux.HandleFunc("DELETE /messages/{id}", s.handleAckMessage)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(addr string) error {
	log.Printf("Mailbox server listening on %s", addr)
	return http.ListenAndServe(addr, s)
}

// POST /users  body: {"username":"alice","token_hash":"hex..."}
func (s *Server) handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username  string `json:"username"`
		TokenHash string `json:"token_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.TokenHash == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := s.store.RegisterUser(req.Username, req.TokenHash); err != nil {
		log.Printf("RegisterUser %q: %v", req.Username, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /messages  body: {"to":"bob","from":"alice","ciphertext":"hex","nonce":"hex"}
// Auth: X-Username + Authorization: Bearer <token>
func (s *Server) handleStoreMessage(w http.ResponseWriter, r *http.Request) {
	sender, ok := s.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		To         string `json:"to"`
		From       string `json:"from"`
		Ciphertext string `json:"ciphertext"` // hex
		Nonce      string `json:"nonce"`      // hex
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.To == "" || req.From == "" || req.Ciphertext == "" || req.Nonce == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}
	if req.From != sender {
		http.Error(w, "from/auth mismatch", http.StatusForbidden)
		return
	}

	ciphertext, err := hex.DecodeString(req.Ciphertext)
	if err != nil {
		http.Error(w, "invalid ciphertext encoding", http.StatusBadRequest)
		return
	}
	nonce, err := hex.DecodeString(req.Nonce)
	if err != nil {
		http.Error(w, "invalid nonce encoding", http.StatusBadRequest)
		return
	}

	id := newID()
	if err := s.store.StoreMessage(id, sender, req.To, ciphertext, nonce); err != nil {
		log.Printf("StoreMessage %s→%s: %v", sender, req.To, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	log.Printf("Stored offline message %s: %s → %s", id, sender, req.To)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// GET /messages
// Auth: X-Username + Authorization: Bearer <token>
func (s *Server) handleFetchMessages(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	msgs, err := s.store.FetchMessages(username)
	if err != nil {
		log.Printf("FetchMessages %q: %v", username, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	type wireMsg struct {
		ID         string `json:"id"`
		From       string `json:"from"`
		Ciphertext string `json:"ciphertext"`
		Nonce      string `json:"nonce"`
		CreatedAt  int64  `json:"created_at"`
	}
	out := make([]wireMsg, len(msgs))
	for i, m := range msgs {
		out[i] = wireMsg{
			ID:         m.ID,
			From:       m.Sender,
			Ciphertext: hex.EncodeToString(m.Ciphertext),
			Nonce:      hex.EncodeToString(m.Nonce),
			CreatedAt:  m.CreatedAt,
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// DELETE /messages/{id}
// Auth: X-Username + Authorization: Bearer <token>
func (s *Server) handleAckMessage(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	if err := s.store.AckMessage(id, username); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// authenticate extracts username + bearer token from headers and validates them.
func (s *Server) authenticate(r *http.Request) (string, bool) {
	username := r.Header.Get("X-Username")
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if username == "" || token == "" || token == auth {
		return "", false
	}
	return username, s.store.Authenticate(username, token)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("rand.Read: %v", err))
	}
	return hex.EncodeToString(b)
}
