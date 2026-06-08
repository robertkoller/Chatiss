package stun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	msgRegister   = "register"
	msgPing       = "ping"
	msgDeregister = "deregister"
	msgLookup     = "lookup"
	msgForward    = "forward"
	msgGetPubkey  = "get_pubkey"

	msgRegisterSuccess = "register_success"
	msgConnect         = "connect"        // sent to both parties — start hole punching
	msgRelayOffer      = "relay_offer"    // forwarded TURN relay address
	msgPubkeyResult    = "pubkey_result"  // response to get_pubkey
	msgError           = "error"
)

type Envelope struct {
	Type       string `json:"type"`
	Username   string `json:"username,omitempty"`
	PhoneHash  string `json:"phone_hash,omitempty"`
	PublicKey  string `json:"public_key,omitempty"`
	ListenPort string `json:"listen_port,omitempty"`
	Address    string `json:"address,omitempty"`
	UDPAddr    string `json:"udp_addr,omitempty"`
	Initiator  bool   `json:"initiator,omitempty"` // true = dial QUIC + allocate TURN, false = listen + join TURN
	Target     string `json:"target,omitempty"`
	RelayAddr  string `json:"relay_addr,omitempty"`
	Error      string `json:"error,omitempty"`
}

// HashPhone hashes a phone number so the server never stores the raw number.
func HashPhone(phone string) string {
	phoneHash := sha256.Sum256([]byte(phone))
	return hex.EncodeToString(phoneHash[:])
}

type ClientInfo struct {
	Username   string
	PhoneHash  string
	PublicKey  string       // hex-encoded, used to enforce username uniqueness
	ListenAddr string       // TCP address (UDP source IP + client's listen port)
	Address    *net.UDPAddr // UDP source address for sending responses back
	LastPing   time.Time
}

type pendingLookup struct {
	requester     *net.UDPAddr
	requesterInfo *ClientInfo // stored so we can notify the target when they register
	expiresAt     time.Time
}

type Server struct {
	conn         *net.UDPConn
	clients      map[string]*ClientInfo     // IP:port → client
	byUsername   map[string]*ClientInfo     // username → client
	byPhoneHash  map[string]*ClientInfo     // phone hash → client
	pending      map[string][]pendingLookup // lookup key → waiters
	paired       map[string]bool            // sorted "a:b" → true once connect signals sent
	usernameKeys map[string]string          // username → pubkey, permanent binding
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	done         chan struct{}
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
		ListenAddress: ":13478",
		ClientTimeout: 30 * time.Second,
		EnableLogging: true,
	}
}

// Initializes a new server
func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		clients:      make(map[string]*ClientInfo),
		byUsername:   make(map[string]*ClientInfo),
		byPhoneHash:  make(map[string]*ClientInfo),
		pending:      make(map[string][]pendingLookup),
		paired:       make(map[string]bool),
		usernameKeys: make(map[string]string),
		ctx:          ctx,
		cancel:       cancel,
		done:         make(chan struct{}),
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
	case msgForward:
		server.handleForward(message, address, enableLogging)
	case msgGetPubkey:
		server.handleGetPubkey(message, address, enableLogging)
	default:
		server.sendError(address, "unknown message type")
	}
}

func (server *Server) handleRegister(message Envelope, address *net.UDPAddr, enableLogging bool) {
	if message.Username == "" && message.PhoneHash == "" {
		server.sendError(address, "username or phone_hash required")
		return
	}
	if message.PublicKey == "" {
		server.sendError(address, "public_key required")
		return
	}
	if message.ListenPort == "" {
		server.sendError(address, "listen_port required")
		return
	}

	server.mu.Lock()
	defer server.mu.Unlock()

	// Enforce permanent username → pubkey binding.
	// Once a username is registered with a public key it can never be claimed
	// by a different key, even after the original client disconnects.
	if message.Username != "" {
		if boundKey, ok := server.usernameKeys[message.Username]; ok {
			if boundKey != message.PublicKey {
				server.sendError(address, "username already taken")
				return
			}
		}
	}

	clientID := address.String()

	// Remove old lookup entries if this client is re-registering.
	if existing, ok := server.clients[clientID]; ok {
		delete(server.byUsername, existing.Username)
		delete(server.byPhoneHash, existing.PhoneHash)
	}

	// Construct TCP listen address from the client's UDP source IP + their declared port.
	listenAddr := fmt.Sprintf("%s:%s", address.IP.String(), message.ListenPort)

	client := &ClientInfo{
		Username:   message.Username,
		PhoneHash:  message.PhoneHash,
		PublicKey:  message.PublicKey,
		ListenAddr: listenAddr,
		Address:    address,
		LastPing:   time.Now(),
	}
	server.clients[clientID] = client

	if message.Username != "" {
		server.byUsername[message.Username] = client
		server.usernameKeys[message.Username] = message.PublicKey
	}
	if message.PhoneHash != "" {
		server.byPhoneHash[message.PhoneHash] = client
	}

	if enableLogging {
		log.Printf("Client registered: %s (username=%q, tcp=%s)", clientID, message.Username, listenAddr)
	}
	server.send(address, Envelope{Type: msgRegisterSuccess})
	server.notifyPending("u:"+message.Username, client, enableLogging)
	server.notifyPending("p:"+message.PhoneHash, client, enableLogging)
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

	requester := server.clients[address.String()]

	if target != nil {
		if enableLogging {
			log.Printf("Lookup hit: %s → %s", key, target.ListenAddr)
		}
		if requester != nil {
			server.notifyBoth(requester, target, enableLogging)
		}
		return
	}

	// Target not online yet — queue the lookup for up to 30 seconds.
	server.pending[key] = append(server.pending[key], pendingLookup{
		requester:     address,
		requesterInfo: requester,
		expiresAt:     time.Now().Add(30 * time.Second),
	})
	if enableLogging {
		log.Printf("Lookup queued: %s waiting for %s", address, key)
	}
}

// notifyPending fulfils any queued lookups waiting for key. Must be called with mu held.
func (server *Server) notifyPending(key string, client *ClientInfo, enableLogging bool) {
	waiters, ok := server.pending[key]
	if !ok {
		return
	}
	for _, waiter := range waiters {
		if waiter.requesterInfo != nil {
			server.notifyBoth(waiter.requesterInfo, client, enableLogging)
		}
	}
	delete(server.pending, key)
}

// notifyBoth sends connect signals to two clients exactly once per pair.
// requester is the initiator (dials QUIC, allocates TURN).
// target is the responder (listens QUIC, joins TURN).
// Must be called with mu held.
func (server *Server) notifyBoth(requester, target *ClientInfo, enableLogging bool) {
	// Never connect two clients that share the same username — same credentials
	// on two machines should not create a session with themselves.
	if requester.Username == target.Username {
		if enableLogging {
			log.Printf("Skipping self-pair for username %q", requester.Username)
		}
		return
	}
	key := pairKey(requester.Username, target.Username)
	if server.paired[key] {
		if enableLogging {
			log.Printf("Pair %s already connected, skipping duplicate signals", key)
		}
		return
	}
	server.paired[key] = true
	if enableLogging {
		log.Printf("Connecting pair %s: initiator=%s responder=%s", key, requester.Username, target.Username)
	}
	// Username carries the *peer's* username so the receiver knows who is connecting.
	server.send(requester.Address, Envelope{Type: msgConnect, Address: target.ListenAddr, UDPAddr: target.Address.String(), Initiator: true, Username: target.Username})
	server.send(target.Address, Envelope{Type: msgConnect, Address: requester.ListenAddr, UDPAddr: requester.Address.String(), Initiator: false, Username: requester.Username})
}

// pairKey returns a stable key for a pair of usernames regardless of order.
func pairKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

// handleForward forwards a TURN relay address to another registered user.
func (server *Server) handleForward(message Envelope, address *net.UDPAddr, enableLogging bool) {
	server.mu.RLock()
	target, ok := server.byUsername[message.Target]
	server.mu.RUnlock()
	if !ok {
		server.sendError(address, "target user not found")
		return
	}
	server.send(target.Address, Envelope{Type: msgRelayOffer, RelayAddr: message.RelayAddr})
	if enableLogging {
		log.Printf("Forwarded relay offer to %s: %s", message.Target, message.RelayAddr)
	}
}

// handleGetPubkey returns the registered public key for a username.
// This lets clients encrypt offline messages for a peer without needing them to be online.
func (server *Server) handleGetPubkey(message Envelope, address *net.UDPAddr, enableLogging bool) {
	server.mu.RLock()
	client, ok := server.byUsername[message.Username]
	server.mu.RUnlock()
	if !ok {
		server.sendError(address, "user not found")
		return
	}
	server.send(address, Envelope{Type: msgPubkeyResult, Username: message.Username, PublicKey: client.PublicKey})
	if enableLogging {
		log.Printf("PubKey lookup: %s → %s", message.Username, client.PublicKey[:8]+"...")
	}
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
	// Clear any pair entries involving this client so they can re-pair on reconnect.
	for key := range server.paired {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 && (parts[0] == client.Username || parts[1] == client.Username) {
			delete(server.paired, key)
			if enableLogging {
				log.Printf("Cleared pair entry %s (client %s left)", key, client.Username)
			}
		}
	}
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
