package turn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const relayPortBase = 40000

type relay struct {
	listener net.Listener
	clientA  net.Conn
	clientB  net.Conn
	mu       sync.Mutex
}

type Server struct {
	publicIP string
	nextPort atomic.Int32
	relays   map[string]*relay // session → relay
	mu       sync.Mutex
}

type controlMsg struct {
	Type    string `json:"type"`
	Session string `json:"session,omitempty"`
	Port    int    `json:"port,omitempty"`
	Error   string `json:"error,omitempty"`
}

func NewServer(publicIP string) *Server {
	s := &Server{
		publicIP: publicIP,
		relays:   make(map[string]*relay),
	}
	s.nextPort.Store(relayPortBase)
	return s
}

// Start listens on all provided addresses. Port 443 gets wrapped in TLS
// so traffic looks like HTTPS to firewalls and passes deep packet inspection.
func (s *Server) Start(addrs ...string) error {
	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return fmt.Errorf("TURN TLS config: %w", err)
	}
	for _, addr := range addrs {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("TURN listen on %s: %w", addr, err)
		}
		// Wrap port 443 in TLS so it genuinely looks like HTTPS to firewalls.
		if addr == ":443" {
			ln = tls.NewListener(ln, tlsConfig)
			log.Printf("TURN server listening on %s (TLS, public IP: %s)", addr, s.publicIP)
		} else {
			log.Printf("TURN server listening on %s (public IP: %s)", addr, s.publicIP)
		}
		go s.acceptLoop(ln)
	}
	return nil
}

func generateTLSConfig() (*tls.Config, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certDER},
			PrivateKey:  key,
		}},
	}, nil
}

func (s *Server) acceptLoop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go s.handleControl(conn)
	}
}

// handleControl reads one JSON message from the client:
//   {"type":"allocate","session":"abc"} → opens a relay port, responds with port number
//   {"type":"join","session":"abc"}     → connects to existing relay as the second peer
func (s *Server) handleControl(conn net.Conn) {
	defer conn.Close()

	var msg controlMsg
	if err := json.NewDecoder(conn).Decode(&msg); err != nil {
		return
	}

	switch msg.Type {
	case "allocate":
		s.handleAllocate(conn, msg.Session)
	case "join":
		s.handleJoin(conn, msg.Session)
	default:
		json.NewEncoder(conn).Encode(controlMsg{Type: "error", Error: "unknown type"})
	}
}

func (s *Server) handleAllocate(controlConn net.Conn, session string) {
	port := int(s.nextPort.Add(1))
	relayListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		json.NewEncoder(controlConn).Encode(controlMsg{Type: "error", Error: "failed to open relay port"})
		return
	}

	r := &relay{listener: relayListener}
	s.mu.Lock()
	s.relays[session] = r
	s.mu.Unlock()

	json.NewEncoder(controlConn).Encode(controlMsg{Type: "allocated", Port: port})
	controlConn.Close()

	// Accept both peers on the relay port.
	go func() {
		defer relayListener.Close()
		connA, err := relayListener.Accept()
		if err != nil {
			return
		}
		connB, err := relayListener.Accept()
		if err != nil {
			connA.Close()
			return
		}
		log.Printf("TURN relay active: session=%s port=%d", session, port)
		go io.Copy(connA, connB)
		io.Copy(connB, connA)
		log.Printf("TURN relay closed: session=%s", session)

		s.mu.Lock()
		delete(s.relays, session)
		s.mu.Unlock()
	}()
}

func (s *Server) handleJoin(controlConn net.Conn, session string) {
	s.mu.Lock()
	r, ok := s.relays[session]
	s.mu.Unlock()
	if !ok {
		json.NewEncoder(controlConn).Encode(controlMsg{Type: "error", Error: "session not found"})
		return
	}
	json.NewEncoder(controlConn).Encode(controlMsg{Type: "joined", Port: 0})
	_ = r
}

// RelayAddr returns the public TCP address for a given relay port.
func (s *Server) RelayAddr(port int) string {
	return fmt.Sprintf("%s:%d", s.publicIP, port)
}
