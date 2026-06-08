package protocol

import "net"

func CreatePing(session *Session) []byte {
	return packetToBytes(createPacket(Version1, TypePing, FlagEmpty, session.ID, nil))
}

func CreatePong(session *Session) []byte {
	return packetToBytes(createPacket(Version1, TypePong, FlagEmpty, session.ID, nil))
}

func CreateDisconnect(session *Session) []byte {
	return packetToBytes(createPacket(Version1, TypeDisconnect, FlagEmpty, session.ID, nil))
}

func processRetrievedPing(session *Session, conn net.Conn) error {
	_, err := conn.Write(CreatePong(session))
	return err
}
