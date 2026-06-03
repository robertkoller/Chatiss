package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/robertkoller/Chatiss/protocol"
	"github.com/robertkoller/Chatiss/sessions"
)

const listenPort = ":4242"

func main() {
	// 1. Login
	fmt.Print("Passphrase: ")
	var passphrase string
	fmt.Scanln(&passphrase)

	identity, err := protocol.Login(passphrase)
	if err != nil {
		log.Fatal("Login failed:", err)
	}

	store, err := sessions.OpenMessageStore(passphrase)
	if err != nil {
		log.Fatal("Failed to open message store:", err)
	}
	defer store.Close()

	sm := protocol.NewSessionManager()

	// 2. Choose mode
	fmt.Print("Peer address to connect to (leave blank to wait for connection): ")
	var peerAddr string
	fmt.Scanln(&peerAddr)

	sessionReady := make(chan *protocol.Session, 1)

	onText := func(session *protocol.Session, content string) {
		store.Save(sessions.Message{
			Peer:      sessions.PeerKey(session.RemotePublicKey.Bytes()),
			Content:   content,
			Timestamp: uint32(time.Now().Unix()),
			Outgoing:  false,
		})
		fmt.Printf("\n[them] %s\n> ", content)
	}

	onConnect := func(session *protocol.Session) {
		sessionReady <- session
	}

	var conn net.Conn

	if peerAddr != "" {
		// Dialer: connect to peer and send handshake
		conn, err = net.DialTimeout("tcp", peerAddr, 10*time.Second)
		if err != nil {
			log.Fatal("Failed to connect:", err)
		}
		fmt.Println("Connected — sending handshake...")
		if _, err := conn.Write(protocol.CreateHandshake(identity.PublicKey)); err != nil {
			log.Fatal("Failed to send handshake:", err)
		}
	} else {
		// Listener: wait for peer to connect
		fmt.Printf("Listening on %s...\n", listenPort)
		listener, err := net.Listen("tcp", listenPort)
		if err != nil {
			log.Fatal("Failed to listen:", err)
		}
		conn, err = listener.Accept()
		listener.Close()
		if err != nil {
			log.Fatal("Failed to accept connection:", err)
		}
		fmt.Println("Peer connected.")
		// Listener side: session is created when handshake is processed.
		// Signal sessionReady when processRetrievedHandshake adds it.
		onConnect = func(session *protocol.Session) {}
		go func() {
			// Give the handshake time to be processed then signal ready.
			// The session is added to sm by processRetrievedHandshake.
			// We watch for it by waiting for the first packet.
		}()
	}
	defer conn.Close()

	// 3. Read loop
	go func() {
		buf := make([]byte, 65536)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Println("\nConnection closed.")
				os.Exit(0)
			}
			if err := protocol.RetrievePacket(buf[:n], identity.PrivateKey, sm, conn, onText, onConnect); err != nil {
				log.Println("Packet error:", err)
			}
		}
	}()

	// Listener side: session is ready after the handshake ack is sent back.
	// Signal it once the session appears in sm.
	if peerAddr == "" {
		go func() {
			for {
				time.Sleep(50 * time.Millisecond)
				// Handshake was processed — session is in sm but we don't know the ID yet.
				// For now, signal ready after a short delay once conn is active.
				// TODO: wire onConnect through processRetrievedHandshake for listener side.
				sessionReady <- nil
				return
			}
		}()
	}

	// 4. Wait for session to be established
	session := <-sessionReady
	if session == nil {
		// Listener side: look up the session after the handshake
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("Session established! Type messages (Ctrl+C to quit):")

	// 5. Send loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		text := scanner.Text()
		if text == "" {
			continue
		}

		if session == nil {
			fmt.Println("Session not ready yet.")
			continue
		}

		packet, err := protocol.CreateText(text, session)
		if err != nil {
			log.Println("Encrypt error:", err)
			continue
		}
		if _, err := conn.Write(packet); err != nil {
			log.Println("Send error:", err)
			break
		}
		store.Save(sessions.Message{
			Peer:      sessions.PeerKey(session.RemotePublicKey.Bytes()),
			Content:   text,
			Timestamp: uint32(time.Now().Unix()),
			Outgoing:  true,
		})
	}
}
