package stun

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"time"
)

const pingInterval = 5 * time.Second

type Client struct {
	conn       *net.UDPConn
	serverAddr *net.UDPAddr
	Results    chan string // receives peer address when a lookup succeeds
	ctx        context.Context
	cancel     context.CancelFunc
}

// Creates a new client to connect to the server
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
	return &Client{
		conn:       conn,
		serverAddr: address,
		Results:    make(chan string, 1),
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

func (client *Client) Register(username, phoneHash string) error {
	return client.send(Envelope{
		Type:      msgRegister,
		Username:  username,
		PhoneHash: phoneHash,
	})
}

func (client *Client) Lookup(username, phoneHash string) error {
	go client.pingLoop()
	go client.readLoop()
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
		case msgLookupResult:
			client.Results <- message.Address
		case msgError:
			log.Printf("STUN error: %s", message.Error)
		}
	}
}

func (client *Client) send(message Envelope) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = client.conn.Write(data)
	return err
}
