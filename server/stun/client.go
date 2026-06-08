package stun

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const pingInterval = 5 * time.Second

// ConnectSignal carries both addresses and the role assigned by the STUN server.
type ConnectSignal struct {
	ListenAddr   string // peer's QUIC/TCP listen address
	UDPAddr      string // peer's UDP address for hole punching
	Initiator    bool   // true = dial QUIC and allocate TURN; false = listen and join TURN
	PeerUsername string // the username of the peer connecting to us
}

// PubKeyResult is the response to a GetPubKey call.
type PubKeyResult struct {
	Username  string
	PublicKey string // hex-encoded X25519 public key
}

type Client struct {
	conn        *net.UDPConn
	serverAddr  *net.UDPAddr
	Connects    chan ConnectSignal // fires when a lookup match is found
	RelayOffers chan string        // fires when a TURN relay is offered
	PubKeys     chan PubKeyResult  // fires when a get_pubkey response arrives
	registered  chan struct{}      // closed when register_success is received
	registerErr chan string        // carries server error before registration succeeds
	pingOnce    sync.Once         // ensures the ping loop starts exactly once
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewClient(serverAddr string) (*Client, error) {
	address, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, address)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		conn:        conn,
		serverAddr:  address,
		Connects:    make(chan ConnectSignal, 1),
		RelayOffers: make(chan string, 1),
		PubKeys:     make(chan PubKeyResult, 4),
		registered:  make(chan struct{}),
		registerErr: make(chan string, 1),
		ctx:         ctx,
		cancel:      cancel,
	}
	go client.readLoop()
	return client, nil
}

// Register sends the registration and retries until the server acknowledges it.
func (client *Client) Register(username, phoneHash, publicKey, listenPort string) error {
	envelope := Envelope{
		Type:       msgRegister,
		Username:   username,
		PhoneHash:  phoneHash,
		PublicKey:  publicKey,
		ListenPort: listenPort,
	}

	for attempt := 1; attempt <= 3; attempt++ {
		if err := client.send(envelope); err != nil {
			return err
		}
		select {
		case <-client.registered:
			return nil
		case msg := <-client.registerErr:
			// Server explicitly rejected us — no point retrying.
			return fmt.Errorf("%s", msg)
		case <-time.After(2 * time.Second):
			log.Printf("STUN register attempt %d timed out, retrying...", attempt)
		case <-client.ctx.Done():
			return fmt.Errorf("client closed")
		}
	}
	return fmt.Errorf("STUN registration failed after 3 attempts")
}

// StartPingLoop begins the keepalive ticker. Safe to call multiple times —
// only the first call has any effect.
func (client *Client) StartPingLoop() {
	client.pingOnce.Do(func() { go client.pingLoop() })
}

func (client *Client) Lookup(username, phoneHash string) error {
	client.StartPingLoop()
	return client.send(Envelope{
		Type:      msgLookup,
		Username:  username,
		PhoneHash: phoneHash,
	})
}

func (client *Client) Close() {
	client.send(Envelope{Type: msgDeregister})
	client.cancel()
	client.conn.Close()
}

func (client *Client) pingLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-client.ctx.Done():
			return
		case <-ticker.C:
			client.send(Envelope{Type: msgPing})
		}
	}
}

func (client *Client) readLoop() {
	buf := make([]byte, 1024)
	for {
		select {
		case <-client.ctx.Done():
			return
		default:
		}

		client.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := client.conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		var message Envelope
		if err := json.Unmarshal(buf[:n], &message); err != nil {
			continue
		}

		switch message.Type {
		case msgRegisterSuccess:
			select {
			case <-client.registered:
				// already closed, nothing to do
			default:
				close(client.registered)
			}
		case msgConnect:
			select {
			case client.Connects <- ConnectSignal{ListenAddr: message.Address, UDPAddr: message.UDPAddr, Initiator: message.Initiator, PeerUsername: message.Username}:
			default:
				// channel full — connect signal already queued
			}
		case msgRelayOffer:
			select {
			case client.RelayOffers <- message.RelayAddr:
			default:
			}
		case msgPubkeyResult:
			select {
			case client.PubKeys <- PubKeyResult{Username: message.Username, PublicKey: message.PublicKey}:
			default:
			}
		case msgError:
			// If registration hasn't succeeded yet, route to Register() so it
			// can fail fast instead of timing out.
			select {
			case <-client.registered:
				// Already registered — this error is from a lookup or forward.
				log.Printf("STUN error: %s", message.Error)
			default:
				select {
				case client.registerErr <- message.Error:
				default:
				}
			}
		}
	}
}

// GetPubKey requests the registered public key for a username.
// The result arrives on the PubKeys channel.
func (client *Client) GetPubKey(username string) error {
	return client.send(Envelope{Type: msgGetPubkey, Username: username})
}

// Forward sends a TURN relay address to another registered user via the STUN server.
func (client *Client) Forward(targetUsername, relayAddr string) error {
	return client.send(Envelope{
		Type:      msgForward,
		Target:    targetUsername,
		RelayAddr: relayAddr,
	})
}

func (client *Client) send(message Envelope) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = client.conn.Write(data)
	return err
}
