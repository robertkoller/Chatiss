package protocol

import (
	"crypto/ecdh"
	"errors"
	"net"
)

// RetrievePacket decyphers and routes an incoming packet to the appropriate
// handler. Nil handlers are silently ignored.
func RetrievePacket(sentBytes []byte, private *ecdh.PrivateKey, sm *SessionManager, conn net.Conn, handler PacketHandler) error {
	packet, err := decypherPacket(sentBytes)
	if err != nil {
		return err
	}

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
		return processRetrievedHandshake(packet, private, sm, conn, handler.LocalUsername, handler.OnConnect)

	case TypeHandshakeAck:
		return processRetrievedHandshakeAck(packet, private, sm, conn, handler.OnConnect)

	case TypeText:
		content, err := processRetrievedText(packet, session)
		if err != nil {
			return err
		}
		if handler.OnText != nil {
			handler.OnText(session, content)
		}

	case TypePing:
		if err := processRetrievedPing(session, conn); err != nil {
			return err
		}
		if handler.OnPing != nil {
			handler.OnPing(session)
		}

	case TypePong:
		// Nothing to do — pong just confirms the peer is alive.

	case TypeCallStart:
		if handler.OnCallStart != nil {
			handler.OnCallStart(session)
		}

	case TypeCallAudio:
		if handler.OnCallAudio != nil {
			handler.OnCallAudio(session, packet.Payload)
		}

	case TypeCallEnd:
		if handler.OnCallEnd != nil {
			handler.OnCallEnd(session)
		}

	case TypeFileStart:
		info, err := parseFileStart(packet)
		if err != nil {
			return err
		}
		if handler.OnFileStart != nil {
			handler.OnFileStart(session, info)
		}

	case TypeFileChunk:
		index, data, err := parseFileChunk(packet)
		if err != nil {
			return err
		}
		if handler.OnFileChunk != nil {
			handler.OnFileChunk(session, index, data)
		}

	case TypeFileEnd:
		if handler.OnFileEnd != nil {
			handler.OnFileEnd(session)
		}

	case TypeDisconnect:
		if handler.OnDisconnect != nil {
			handler.OnDisconnect(session)
		}
	}

	return nil
}
