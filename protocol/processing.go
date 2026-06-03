package protocol

import (
	"crypto/ecdh"
	"errors"
	"net"
)

// RetrievePacket decyphers and routes an incoming packet.
// onText is called with the decrypted content when a text packet arrives.
// onConnect is called when a handshake completes and a session is established.
func RetrievePacket(sentBytes []byte, private *ecdh.PrivateKey, sm *SessionManager, conn net.Conn, onText func(*Session, string), onConnect func(*Session)) error {
	packet, err := decypherPacket(sentBytes)
	if err != nil {
		return err
	}

	// Handshake and ack arrive before a session exists — everything else must match a known session.
	var session *Session
	if packet.Header.PacketType != TypeHandshake && packet.Header.PacketType != TypeHandshakeAck {
		var ok bool
		session, ok = sm.Get(packet.Header.SessionID)
		if !ok {
			return errors.New("packet rejected: unknown session ID")
		}
	}

	switch packet.Header.PacketType {
	case TypeHandshake:
		return processRetrievedHandshake(packet, private, sm, conn)
	case TypeHandshakeAck:
		return processRetrievedHandshakeAck(packet, private, sm, conn, onConnect)
	case TypeText:
		content, err := processRetrievedText(packet, session)
		if err != nil {
			return err
		}
		onText(session, content)
		return nil
	}

	return nil
}
