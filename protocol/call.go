package protocol

func CreateCallStart(session *Session) []byte {
	return packetToBytes(createPacket(Version1, TypeCallStart, FlagEmpty, session.ID, nil))
}

// CreateCallAudio wraps a raw audio frame in a packet.
// The audio bytes should already be encoded (e.g. Opus) before passing here.
func CreateCallAudio(session *Session, audioFrame []byte) []byte {
	return packetToBytes(createPacket(Version1, TypeCallAudio, FlagEmpty, session.ID, audioFrame))
}

func CreateCallEnd(session *Session) []byte {
	return packetToBytes(createPacket(Version1, TypeCallEnd, FlagEmpty, session.ID, nil))
}
