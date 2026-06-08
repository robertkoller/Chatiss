package protocol

import (
	"crypto/ecdh"
	"errors"
	"net"
)

// CreateHandshake builds the opening handshake packet carrying the local
// public key and username so the peer can identify us.
func CreateHandshake(publicKey *ecdh.PublicKey, username string) []byte {
	payload := encodeIdentity(publicKey, username)
	packet := createPacket(Version1, TypeHandshake, FlagEmpty, 0, payload)
	return packetToBytes(packet)
}

// encodeIdentity packs [1-byte username length][username][pubkey bytes].
func encodeIdentity(publicKey *ecdh.PublicKey, username string) []byte {
	usernameBytes := []byte(username)
	payload := make([]byte, 1+len(usernameBytes)+len(publicKey.Bytes()))
	payload[0] = byte(len(usernameBytes))
	copy(payload[1:], usernameBytes)
	copy(payload[1+len(usernameBytes):], publicKey.Bytes())
	return payload
}

// decodeIdentity unpacks the payload produced by encodeIdentity.
func decodeIdentity(payload []byte) (*ecdh.PublicKey, string, error) {
	if len(payload) < 1 {
		return nil, "", errors.New("handshake payload too short")
	}
	usernameLen := int(payload[0])
	if len(payload) < 1+usernameLen+32 {
		return nil, "", errors.New("handshake payload too short for username + pubkey")
	}
	username := string(payload[1 : 1+usernameLen])
	pubKey, err := ecdh.X25519().NewPublicKey(payload[1+usernameLen:])
	return pubKey, username, err
}

func processRetrievedHandshake(handshake Packet, private *ecdh.PrivateKey, sm *SessionManager, conn net.Conn, localUsername string, onConnect func(*Session)) error {
	theirPubKey, theirUsername, err := decodeIdentity(handshake.Payload)
	if err != nil {
		return errors.New("failed to decode handshake identity")
	}
	sesID, sharedSecret, err := DeriveSessionInfo(private, theirPubKey)
	if err != nil {
		return err
	}
	session := Session{
		ID:              sesID,
		SharedSecret:    sharedSecret,
		RemotePublicKey: theirPubKey,
		RemoteUsername:  theirUsername,
		Conn:            conn,
	}

	ack := createHandShakeAck(&session, private.PublicKey(), localUsername)
	if _, err := conn.Write(ack); err != nil {
		return err
	}

	sm.Add(&session)
	onConnect(&session)
	return nil
}

func createHandShakeAck(session *Session, publicKey *ecdh.PublicKey, localUsername string) []byte {
	payload := encodeIdentity(publicKey, localUsername)
	ackPacket := createPacket(Version1, TypeHandshakeAck, FlagEmpty, session.ID, payload)
	return packetToBytes(ackPacket)
}

func processRetrievedHandshakeAck(ack Packet, private *ecdh.PrivateKey, sm *SessionManager, conn net.Conn, onConnect func(*Session)) error {
	theirPubKey, theirUsername, err := decodeIdentity(ack.Payload)
	if err != nil {
		return errors.New("failed to decode ack identity")
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
		RemoteUsername:  theirUsername,
		Conn:            conn,
	}
	sm.Add(&session)
	onConnect(&session)
	return nil
}
