package app

import (
	"context"
	"crypto/ecdh"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/robertkoller/Chatiss/mailbox"
	"github.com/robertkoller/Chatiss/protocol"
	"github.com/robertkoller/Chatiss/server/stun"
	"github.com/robertkoller/Chatiss/sessions"
	"github.com/robertkoller/Chatiss/transport"
)

const (
	stunServer    = "178.128.151.84:13478"
	turnServer    = "178.128.151.84"
	mailboxServer = "http://178.128.151.84:8080"
	listenPort    = "4242"
	quicTimeout   = 8 * time.Second
	punchDur      = 600 * time.Millisecond
)

// UIMessage is a message entry passed to the frontend.
type UIMessage struct {
	From      string `json:"from"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
	Outgoing  bool   `json:"outgoing"`
	Pending   bool   `json:"pending"`
}

// UIContact is a contact entry passed to the frontend.
type UIContact struct {
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

// Service is the persistent backend. It stays registered with STUN, accepts
// incoming connections, and routes messages to/from the UI via emitEvent.
type Service struct {
	identity   *protocol.LocalIdentity
	username   string
	stunClient *stun.Client
	mailboxCli *mailbox.Client
	store      *sessions.MessageStore
	udpConn    *net.UDPConn
	qt         *quic.Transport
	listener   *transport.Listener
	sm         *protocol.SessionManager

	mu         sync.RWMutex
	sessions   map[string]*protocol.Session // peerUsername → active session
	online     map[string]bool
	fetchingMu sync.Mutex // prevents concurrent mailbox fetches

	ctx    context.Context
	cancel context.CancelFunc

	emitEvent func(name string, data ...any)
}

// NewService creates and starts the background service for the given user.
func NewService(passphrase, username string, emitEvent func(string, ...any)) (*Service, error) {
	identity, err := protocol.Login(passphrase)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	store, err := sessions.OpenMessageStore(passphrase)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	udpConn, err := transport.OpenUDPSocket(listenPort)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("open UDP socket: %w", err)
	}

	qt := transport.NewTransport(udpConn)

	listener, err := transport.NewListener(qt)
	if err != nil {
		udpConn.Close()
		store.Close()
		return nil, fmt.Errorf("open QUIC listener: %w", err)
	}

	stunClient, err := stun.NewClient(stunServer)
	if err != nil {
		listener.Close()
		udpConn.Close()
		store.Close()
		return nil, fmt.Errorf("connect to STUN: %w", err)
	}

	pubKeyHex := hex.EncodeToString(identity.PublicKey.Bytes())
	if err := stunClient.Register(username, "", pubKeyHex, listenPort); err != nil {
		stunClient.Close()
		listener.Close()
		udpConn.Close()
		store.Close()
		return nil, err
	}

	mailboxCli := mailbox.NewClient(mailboxServer, username, identity.MailboxToken)
	if err := mailboxCli.Register(); err != nil {
		log.Printf("Mailbox register failed (offline messages unavailable): %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	svc := &Service{
		identity:   identity,
		username:   username,
		stunClient: stunClient,
		mailboxCli: mailboxCli,
		store:      store,
		udpConn:    udpConn,
		qt:         qt,
		listener:   listener,
		sm:         protocol.NewSessionManager(),
		sessions:   make(map[string]*protocol.Session),
		online:     make(map[string]bool),
		ctx:        ctx,
		cancel:     cancel,
		emitEvent:  emitEvent,
	}

	// Start the STUN keepalive so the server doesn't deregister us.
	stunClient.StartPingLoop()

	go svc.connectSignalLoop()
	go svc.fetchPendingMessages()
	go svc.connectAllContacts()

	return svc, nil
}

// Stop shuts down the service cleanly.
func (svc *Service) Stop() {
	svc.cancel()
	svc.stunClient.Close()
	svc.listener.Close()
	svc.udpConn.Close()
	svc.store.Close()
}

// SendMessage sends a text message to a peer. Tries live P2P first, falls back
// to the mailbox if the peer is offline.
func (svc *Service) SendMessage(peerUsername, text string) error {
	svc.mu.RLock()
	session := svc.sessions[peerUsername]
	svc.mu.RUnlock()

	if session != nil {
		pkt, err := protocol.CreateText(text, session)
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		if _, err := session.Conn.Write(pkt); err != nil {
			return fmt.Errorf("send: %w", err)
		}
		svc.saveMessage(peerUsername, sessions.PeerKey(session.RemotePublicKey.Bytes()), text, true, time.Now().Unix())
		return nil
	}

	// Peer is offline — kick off a STUN lookup so if they come online within
	// 30s the server pairs us automatically and we get a live connection.
	go func() { _ = svc.stunClient.Lookup(peerUsername, "") }()

	// Encrypt and upload to mailbox so the message is delivered even if the
	// peer doesn't come online within the STUN window.
	contact, err := svc.store.GetContact(peerUsername)
	if err != nil {
		return fmt.Errorf("unknown contact %q — add them first", peerUsername)
	}
	pubKeyBytes, err := hex.DecodeString(contact.PubKeyHex)
	if err != nil {
		return fmt.Errorf("bad pubkey in contacts: %w", err)
	}
	recipientPub, err := ecdh.X25519().NewPublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("parse pubkey: %w", err)
	}
	if err := svc.mailboxCli.StorePending(svc.identity.PrivateKey, recipientPub, peerUsername, text); err != nil {
		return fmt.Errorf("mailbox: %w", err)
	}
	svc.saveMessage(peerUsername, contact.PubKeyHex, text, true, time.Now().Unix())
	return nil
}

// Connect explicitly triggers a STUN lookup to establish a live P2P session
// with the given peer. Call this when opening a chat window or starting a
// conversation — if the peer is online, the STUN server will pair you and
// connectSignalLoop will handle the rest.
func (svc *Service) Connect(peerUsername string) {
	if peerUsername == svc.username {
		return
	}
	go func() { _ = svc.stunClient.Lookup(peerUsername, "") }()
}

// AddContact resolves a username's public key via STUN and stores the contact.
func (svc *Service) AddContact(username string) error {
	if err := svc.stunClient.GetPubKey(username); err != nil {
		return fmt.Errorf("STUN get_pubkey: %w", err)
	}
	select {
	case result := <-svc.stunClient.PubKeys:
		if result.Username != username {
			return fmt.Errorf("unexpected pubkey response for %q", result.Username)
		}
		if err := svc.store.AddContact(username, result.PublicKey); err != nil {
			return fmt.Errorf("store contact: %w", err)
		}
		// Peer is online right now (we got their pubkey) — try to connect.
		go func() { _ = svc.stunClient.Lookup(username, "") }()
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timed out waiting for pubkey — is %q registered?", username)
	}
}

// GetContacts returns all contacts with live online status merged in.
func (svc *Service) GetContacts() ([]UIContact, error) {
	contacts, err := svc.store.GetContacts()
	if err != nil {
		return nil, err
	}
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	out := make([]UIContact, len(contacts))
	for i, c := range contacts {
		out[i] = UIContact{Username: c.Username, Online: svc.online[c.Username]}
	}
	return out, nil
}

// GetMessages returns the stored message history with a contact.
func (svc *Service) GetMessages(peerUsername string) ([]UIMessage, error) {
	contact, err := svc.store.GetContact(peerUsername)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}
	msgs, err := svc.store.Load(contact.PubKeyHex, 200, 0)
	if err != nil {
		return nil, err
	}
	out := make([]UIMessage, len(msgs))
	for i, m := range msgs {
		from := peerUsername
		if m.Outgoing {
			from = svc.username
		}
		out[i] = UIMessage{
			From:      from,
			Text:      m.Content,
			Timestamp: int64(m.Timestamp),
			Outgoing:  m.Outgoing,
		}
	}
	return out, nil
}

// --- internal goroutines ---

func (svc *Service) connectSignalLoop() {
	for {
		select {
		case <-svc.ctx.Done():
			return
		case signal := <-svc.stunClient.Connects:
			go svc.handleConnectSignal(signal)
		}
	}
}

func (svc *Service) handleConnectSignal(signal stun.ConnectSignal) {
	peer := signal.PeerUsername
	log.Printf("[connect] signal from STUN: peer=%q initiator=%v udp=%s", peer, signal.Initiator, signal.UDPAddr)

	peerUDPAddr, err := net.ResolveUDPAddr("udp", signal.UDPAddr)
	if err != nil {
		log.Printf("[connect] bad peer UDP addr %q: %v", signal.UDPAddr, err)
		return
	}

	log.Printf("[connect] hole punching %s for 600ms…", peerUDPAddr)
	transport.HolePunch(svc.qt, peerUDPAddr, punchDur)

	var conn net.Conn
	if signal.Initiator {
		log.Printf("[connect] dialing QUIC → %s", peerUDPAddr)
		conn, err = transport.Dial(svc.qt, peerUDPAddr, quicTimeout)
		if err != nil {
			log.Printf("[connect] QUIC dial failed (%v), allocating TURN relay…", err)
			conn = svc.turnAllocate(peer)
		} else {
			log.Printf("[connect] QUIC dial succeeded")
		}
	} else {
		log.Printf("[connect] waiting for QUIC connection from peer (timeout %s)…", quicTimeout)
		acceptCtx, cancel := context.WithTimeout(svc.ctx, quicTimeout)
		defer cancel()
		conn, err = svc.listener.Accept(acceptCtx)
		if err != nil {
			log.Printf("[connect] QUIC accept failed (%v), waiting for TURN relay…", err)
			conn = svc.turnJoin()
		} else {
			log.Printf("[connect] QUIC accept succeeded")
		}
	}

	if conn == nil {
		log.Printf("[connect] no transport available for %q — giving up", peer)
		return
	}

	if signal.Initiator {
		log.Printf("[connect] sending handshake to %q…", peer)
		if _, werr := conn.Write(protocol.CreateHandshake(svc.identity.PublicKey, svc.username)); werr != nil {
			log.Printf("[connect] handshake write failed: %v", werr)
			conn.Close()
			return
		}
	} else {
		log.Printf("[connect] waiting for handshake from %q…", peer)
	}

	svc.runReadLoop(conn)
}

func (svc *Service) runReadLoop(conn net.Conn) {
	handler := protocol.PacketHandler{
		LocalUsername: svc.username,
		OnConnect: func(s *protocol.Session) {
			pubKeyHex := hex.EncodeToString(s.RemotePublicKey.Bytes())

			// Auto-save as contact if not already known so the sidebar shows them.
			isNew := false
			if _, err := svc.store.GetContact(s.RemoteUsername); err != nil {
				if addErr := svc.store.AddContact(s.RemoteUsername, pubKeyHex); addErr != nil {
					log.Printf("auto-add contact %q: %v", s.RemoteUsername, addErr)
				} else {
					isNew = true
					log.Printf("auto-added new contact %q", s.RemoteUsername)
				}
			}

			svc.mu.Lock()
			svc.sessions[s.RemoteUsername] = s
			svc.online[s.RemoteUsername] = true
			svc.mu.Unlock()

			if isNew {
				svc.emit("contact:added", map[string]string{"username": s.RemoteUsername})
			}
			svc.emit("contact:online", map[string]string{"username": s.RemoteUsername})
			log.Printf("[connect] session established with %s (new contact: %v)", s.RemoteUsername, isNew)

			// Fetch any messages that arrived while we were offline.
			go svc.fetchPendingMessages()
		},
		OnDisconnect: func(s *protocol.Session) {
			svc.mu.Lock()
			delete(svc.sessions, s.RemoteUsername)
			svc.online[s.RemoteUsername] = false
			svc.mu.Unlock()
			svc.emit("contact:offline", map[string]string{"username": s.RemoteUsername})
			log.Printf("Session closed with %s", s.RemoteUsername)
		},
		OnText: func(s *protocol.Session, content string) {
			svc.saveMessage(s.RemoteUsername, sessions.PeerKey(s.RemotePublicKey.Bytes()), content, false, time.Now().Unix())
			svc.emit("message:received", UIMessage{
				From:      s.RemoteUsername,
				Text:      content,
				Timestamp: time.Now().Unix(),
				Outgoing:  false,
			})
		},
		OnPing:      func(s *protocol.Session) {},
		OnCallStart: func(s *protocol.Session) {},
		OnCallAudio: func(s *protocol.Session, frame []byte) {},
		OnCallEnd:   func(s *protocol.Session) {},
		OnFileStart: func(s *protocol.Session, info protocol.FileInfo) {},
		OnFileChunk: func(s *protocol.Session, index uint32, data []byte) {},
		OnFileEnd:   func(s *protocol.Session) {},
	}

	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		if err := protocol.RetrievePacket(buf[:n], svc.identity.PrivateKey, svc.sm, conn, handler); err != nil {
			log.Printf("packet error: %v", err)
		}
	}
}

func (svc *Service) fetchPendingMessages() {
	// Only one fetch at a time — prevents duplicate delivery when multiple
	// OnConnect events fire in quick succession.
	if !svc.fetchingMu.TryLock() {
		return
	}
	defer svc.fetchingMu.Unlock()

	msgs, err := svc.mailboxCli.FetchPending()
	if err != nil {
		log.Printf("Fetch pending: %v", err)
		return
	}
	for _, m := range msgs {
		contact, err := svc.store.GetContact(m.From)
		if err != nil {
			log.Printf("Pending message from unknown sender %q — skipping", m.From)
			continue
		}
		pubKeyBytes, _ := hex.DecodeString(contact.PubKeyHex)
		senderPub, err := ecdh.X25519().NewPublicKey(pubKeyBytes)
		if err != nil {
			continue
		}
		text, err := mailbox.DecryptMessage(svc.identity.PrivateKey, senderPub, m)
		if err != nil {
			log.Printf("Decrypt pending from %q: %v", m.From, err)
			continue
		}
		// Use the original send time so messages sort correctly in history.
		svc.saveMessage(m.From, contact.PubKeyHex, text, false, m.CreatedAt)
		svc.emit("message:received", UIMessage{
			From:      m.From,
			Text:      text,
			Timestamp: m.CreatedAt,
			Outgoing:  false,
			Pending:   true,
		})
		svc.mailboxCli.Ack(m.ID)
	}
}

// connectAllContacts triggers a STUN lookup for every stored contact so that
// if any of them are online we establish a live session automatically at startup.
func (svc *Service) connectAllContacts() {
	contacts, err := svc.store.GetContacts()
	if err != nil {
		log.Printf("connectAllContacts: %v", err)
		return
	}
	for _, c := range contacts {
		if c.Username == svc.username {
			continue
		}
		go func(username string) { _ = svc.stunClient.Lookup(username, "") }(c.Username)
	}
}

func (svc *Service) saveMessage(peerUsername, peerKey, text string, outgoing bool, ts int64) {
	if ts == 0 {
		ts = time.Now().Unix()
	}
	svc.store.Save(sessions.Message{
		Peer:      peerKey,
		Content:   text,
		Timestamp: uint32(ts),
		Outgoing:  outgoing,
	})
}

func (svc *Service) emit(name string, data any) {
	if svc.emitEvent != nil {
		svc.emitEvent(name, data)
	}
}

// --- TURN helpers ---

type turnControlMsg struct {
	Type    string `json:"type"`
	Session string `json:"session,omitempty"`
	Port    int    `json:"port,omitempty"`
}

func (svc *Service) turnAllocate(peerUsername string) net.Conn {
	session := fmt.Sprintf("chatiss-%d", time.Now().UnixNano())
	for _, port := range []string{"13479", "443"} {
		ctrl, err := turnDial(port)
		if err != nil {
			continue
		}
		json.NewEncoder(ctrl).Encode(turnControlMsg{Type: "allocate", Session: session})
		var resp turnControlMsg
		json.NewDecoder(ctrl).Decode(&resp)
		ctrl.Close()
		if resp.Type != "allocated" {
			continue
		}
		relayAddr := fmt.Sprintf("%s:%d", turnServer, resp.Port)
		if err := svc.stunClient.Forward(peerUsername, relayAddr); err != nil {
			log.Printf("STUN forward failed: %v", err)
		}
		conn, err := net.DialTimeout("tcp", relayAddr, 15*time.Second)
		if err != nil {
			continue
		}
		return conn
	}
	return nil
}

func (svc *Service) turnJoin() net.Conn {
	select {
	case relayAddr := <-svc.stunClient.RelayOffers:
		conn, err := net.DialTimeout("tcp", relayAddr, 15*time.Second)
		if err != nil {
			return nil
		}
		return conn
	case <-time.After(20 * time.Second):
		return nil
	case <-svc.ctx.Done():
		return nil
	}
}

func turnDial(port string) (net.Conn, error) {
	addr := turnServer + ":" + port
	if port == "443" {
		return tls.DialWithDialer(
			&net.Dialer{Timeout: 4 * time.Second},
			"tcp", addr,
			&tls.Config{InsecureSkipVerify: true},
		)
	}
	return net.DialTimeout("tcp", addr, 4*time.Second)
}
