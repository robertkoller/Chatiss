package protocol

// PacketHandler holds callbacks for every packet type.
// Set only the ones your app needs — nil callbacks are silently ignored.
type PacketHandler struct {
	LocalUsername string // included in handshake so the peer knows who we are
	OnConnect     func(*Session)
	OnDisconnect  func(*Session)
	OnText        func(*Session, string)
	OnPing        func(*Session)
	OnCallStart   func(*Session)
	OnCallAudio   func(*Session, []byte)
	OnCallEnd     func(*Session)
	OnFileStart   func(*Session, FileInfo)
	OnFileChunk   func(*Session, uint32, []byte)
	OnFileEnd     func(*Session)
}
