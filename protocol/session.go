package protocol

import (
	"crypto/ecdh"
	"net"
	"sync"
)

// This holds our connections with peers
type Session struct {
	ID              uint32
	SharedSecret    []byte
	RemotePublicKey *ecdh.PublicKey
	RemoteUsername  string
	Conn            net.Conn
}

// This manages sessions so we dont overload on connections
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[uint32]*Session
}

// Initializes the session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[uint32]*Session),
	}
}

// Add a session to the session manager
func (sm *SessionManager) Add(s *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[s.ID] = s
}

// Get a session from the session manager
func (sm *SessionManager) Get(id uint32) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, ok := sm.sessions[id]
	return session, ok
}

// Remove a session from the session manager
func (sm *SessionManager) Remove(id uint32) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, id)
}
