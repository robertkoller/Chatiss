package protocol

import (
	"crypto/ecdh"
	"errors"
	"net"
)

// Creates the handshake packet
func CreateHandshake(PublicKey *ecdh.PublicKey) []byte {
	packet := createPacket(Version1, TypeHandshake, FlagEmpty, 0, PublicKey.Bytes())
	return packetToBytes(packet)
}

// We retrieve a handshake and add it to our session manager
func processRetrievedHandshake(handshake Packet, private *ecdh.PrivateKey, sm *SessionManager, conn net.Conn) error {
	theirPubKey, err := ecdh.X25519().NewPublicKey(handshake.Payload)
	if err != nil {
		return errors.New("failed to decypher retrieved public key")
	}
	sesID, sharedSecret, err := DeriveSessionInfo(private, theirPubKey)
	if err != nil {
		return err
	}
	session := Session{
		ID:              sesID,
		SharedSecret:    sharedSecret,
		RemotePublicKey: theirPubKey,
		Conn:            conn,
	}

	ack := createHandShakeAck(&session, private.PublicKey())

	if _, err := conn.Write(ack); err != nil {
		return err
	}

	sm.Add(&session)

	return nil
}

// We got their handshake so we send back the ack to say we were sucessful
func createHandShakeAck(session *Session, public *ecdh.PublicKey) []byte {
	ackPacket := createPacket(Version1, TypeHandshakeAck, FlagEmpty, session.ID, public.Bytes())
	ack := packetToBytes(ackPacket)
	return ack
}

// Processes a recieved ack
func processRetrievedHandshakeAck(ack Packet, private *ecdh.PrivateKey, sm *SessionManager, conn net.Conn, onConnect func(*Session)) error {
	theirPubKey, err := ecdh.X25519().NewPublicKey(ack.Payload)
	if err != nil {
		return errors.New("failed to decode public key from ack")
	}
	sesID, sharedSecret, err := DeriveSessionInfo(private, theirPubKey)
	if err != nil {
		return err
	}
	if sesID != ack.Header.SessionID {
		conn.Close()
		return errors.New("session ID mismatch, possible tampering")
	}
	session := Session{
		ID:              sesID,
		SharedSecret:    sharedSecret,
		RemotePublicKey: theirPubKey,
		Conn:            conn,
	}
	sm.Add(&session)
	onConnect(&session)
	return nil
}
