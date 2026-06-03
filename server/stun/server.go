package stun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const (
	msgRegister   = "register"
	msgPing       = "ping"
	msgDeregister = "deregister"
	msgLookup     = "lookup"

	msgRegisterSuccess = "register_success"
	msgLookupResult    = "lookup_result"
	msgError           = "error"
)

type Envelope struct {
	Type      string `json:"type"`
	Username  string `json:"username,omitempty"`
	PhoneHash string `json:"phone_hash,omitempty"`
	Address   string `json:"address,omitempty"`
	Error     string `json:"error,omitempty"`
}

// HashPhone hashes a phone number so the server never stores the raw number.
func HashPhone(phone string) string {
	phoneHash := sha256.Sum256([]byte(phone))
	return hex.EncodeToString(phoneHash[:])
}

type ClientInfo struct {
	Username  string
	PhoneHash string
	Address   *net.UDPAddr
	LastPing  time.Time
}

type pendingLookup struct {
	requester *net.UDPAddr
	expiresAt time.Time
}

type Server struct {
	conn        *net.UDPConn
	clients     map[string]*ClientInfo    // IP:port → client
	byUsername  map[string]*ClientInfo    // username → client
	byPhoneHash map[string]*ClientInfo    // phone hash → client
	pending     map[string][]pendingLookup // lookup key → waiters
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan struct{}
}

// Configs for our server
type ServerConfig struct {
	ListenAddress string
	ClientTimeout time.Duration
	EnableLogging bool
}

// Default server stats
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ListenAddress: ":3478",
		ClientTimeout: 30 * time.Second,
		EnableLogging: true,
	}
}

// Initializes a new server
func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		clients:     make(map[string]*ClientInfo),
		byUsername:  make(map[string]*ClientInfo),
		byPhoneHash: make(map[string]*ClientInfo),
		pending:     make(map[string][]pendingLookup),
		ctx:         ctx,
		cancel:      cancel,
		done:        make(chan struct{}),
	}
}

// Starts up our server and opens us to connections
func (server *Server) Start(config *ServerConfig) error {
	address, err := net.ResolveUDPAddr("udp", config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}
	connection, err := net.ListenUDP("udp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}
	server.conn = connection

	if config.EnableLogging {
		log.Printf("STUN server started on %s", config.ListenAddress)
	}

	go server.cleanupRoutine(config.ClientTimeout, config.EnableLogging)
	go server.handleMessages(config.EnableLogging)
	return nil
}

// Stops the stun server
func (server *Server) Stop() error {
	server.cancel()
	if server.conn != nil {
		server.conn.Close()
	}
	<-server.done
	return nil
}

// Returns the server connection
func (server *Server) GetConn() *net.UDPConn {
	return server.conn
}

// Returns the number of connected clients
func (server *Server) GetConnectedClients() int {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return len(server.clients)
}

func (server *Server) handleMessages(enableLogging bool) {
	defer close(server.done)
	buf := make([]byte, 1024)

	for {
		select {
		case <-server.ctx.Done():
			return
		default:
		}

		server.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, address, err := server.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if enableLogging {
				log.Printf("UDP read error: %v", err)
			}
			continue
		}

		raw := make([]byte, n)
		copy(raw, buf[:n])
		go server.processMessage(raw, address, enableLogging)
	}
}

func (server *Server) processMessage(data []byte, address *net.UDPAddr, enableLogging bool) {
	if len(data) == 0 || data[0] != '{' {
		return
	}
	var message Envelope
	if err := json.Unmarshal(data, &message); err != nil {
		server.sendError(address, "invalid message format")
		return
	}

	switch message.Type {
	case msgRegister:
		server.handleRegister(message, address, enableLogging)
	case msgPing:
		server.handlePing(address)
	case msgDeregister:
		server.handleDeregister(address, enableLogging)
	case msgLookup:
		server.handleLookup(message, address, enableLogging)
	default:
		server.sendError(address, "unknown message type")
	}
}

func (server *Server) handleRegister(message Envelope, address *net.UDPAddr, enableLogging bool) {
	if message.Username == "" && message.PhoneHash == "" {
		server.sendError(address, "username or phone_hash required")
		return
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	clientID := address.String()

	// Remove old lookup entries if this client is re-registering.
	if existing, ok := server.clients[clientID]; ok {
		delete(server.byUsername, existing.Username)
		delete(server.byPhoneHash, existing.PhoneHash)
	}

	client := &ClientInfo{
		Username:  message.Username,
		PhoneHash: message.PhoneHash,
		Address:   address,
		LastPing:  time.Now(),
	}
	server.clients[clientID] = client

	if message.Username != "" {
		server.byUsername[message.Username] = client
	}
	if message.PhoneHash != "" {
		server.byPhoneHash[message.PhoneHash] = client
	}

	if enableLogging {
		log.Printf("Client registered: %s (username=%q)", clientID, message.Username)
	}
	server.send(address, Envelope{Type: msgRegisterSuccess})
	server.notifyPending("u:"+message.Username, address)
	server.notifyPending("p:"+message.PhoneHash, address)
}

func (server *Server) handlePing(address *net.UDPAddr) {
	server.mu.Lock()
	defer server.mu.Unlock()
	if client, ok := server.clients[address.String()]; ok {
		client.LastPing = time.Now()
		client.Address = address
	}
}

func (server *Server) handleDeregister(address *net.UDPAddr, enableLogging bool) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.removeClient(address.String(), enableLogging)
}

func (server *Server) handleLookup(message Envelope, address *net.UDPAddr, enableLogging bool) {
	server.mu.Lock()
	defer server.mu.Unlock()

	var target *ClientInfo
	var key string
	if message.Username != "" {
		target = server.byUsername[message.Username]
		key = "u:" + message.Username
	} else if message.PhoneHash != "" {
		target = server.byPhoneHash[message.PhoneHash]
		key = "p:" + message.PhoneHash
	} else {
		server.sendError(address, "username or phone_hash required")
		return
	}

	if target != nil {
		if enableLogging {
			log.Printf("Lookup hit: %s → %s", key, target.Address)
		}
		server.send(address, Envelope{Type: msgLookupResult, Address: target.Address.String()})
		return
	}

	// Target not online yet — queue the lookup for up to 30 seconds.
	server.pending[key] = append(server.pending[key], pendingLookup{
		requester: address,
		expiresAt: time.Now().Add(30 * time.Second),
	})
	if enableLogging {
		log.Printf("Lookup queued: %s waiting for %s", address, key)
	}
}

// notifyPending fulfils any queued lookups waiting for key. Must be called with mu held.
func (server *Server) notifyPending(key string, address *net.UDPAddr) {
	waiters, ok := server.pending[key]
	if !ok {
		return
	}
	for _, waiter := range waiters {
		server.send(waiter.requester, Envelope{Type: msgLookupResult, Address: address.String()})
	}
	delete(server.pending, key)
}

func (server *Server) cleanupRoutine(timeout time.Duration, enableLogging bool) {
	ticker := time.NewTicker(timeout / 2)
	defer ticker.Stop()
	for {
		select {
		case <-server.ctx.Done():
			return
		case <-ticker.C:
			server.cleanupInactiveClients(timeout, enableLogging)
		}
	}
}

func (server *Server) cleanupInactiveClients(timeout time.Duration, enableLogging bool) {
	server.mu.Lock()
	defer server.mu.Unlock()

	now := time.Now()
	for id, client := range server.clients {
		if now.Sub(client.LastPing) > timeout {
			if enableLogging {
				log.Printf("Removing inactive client %s (username=%q)", id, client.Username)
			}
			server.removeClient(id, false)
		}
	}

	// Waiters are appended in time order so the slice is naturally sorted by
	// expiresAt — stop at the first non-expired entry instead of scanning all.
	for key, waiters := range server.pending {
		firstValid := len(waiters)
		for i, waiter := range waiters {
			if now.Before(waiter.expiresAt) {
				firstValid = i
				break
			}
			if enableLogging {
				log.Printf("Pending lookup expired: %s waiting for %s", waiter.requester, key)
			}
		}
		if firstValid == len(waiters) {
			delete(server.pending, key)
		} else {
			server.pending[key] = waiters[firstValid:]
		}
	}
}

// removeClient removes a client from all maps. Must be called with mu held.
func (server *Server) removeClient(id string, enableLogging bool) {
	client, ok := server.clients[id]
	if !ok {
		return
	}
	delete(server.clients, id)
	delete(server.byUsername, client.Username)
	delete(server.byPhoneHash, client.PhoneHash)
	if enableLogging {
		log.Printf("Client removed: %s", id)
	}
}

// Sends a JSON envelope to an address
func (server *Server) send(address *net.UDPAddr, message Envelope) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to serialize message: %v", err)
		return
	}
	if _, err := server.conn.WriteToUDP(data, address); err != nil {
		log.Printf("Failed to send to %s: %v", address, err)
	}
}

// Sends an error message to an address
func (server *Server) sendError(address *net.UDPAddr, errMsg string) {
	server.send(address, Envelope{Type: msgError, Error: errMsg})
}
