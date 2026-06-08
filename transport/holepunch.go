package transport

import (
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

var punchPayload = []byte("CHATISS_PUNCH")

// HolePunch sends UDP punch packets to the peer for the given duration.
// Uses the QUIC transport's WriteTo so the packets come from the same
// socket that QUIC will use — essential for the NAT pinhole to work.
func HolePunch(quicTransport *quic.Transport, peerAddr net.Addr, duration time.Duration) {
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		quicTransport.WriteTo(punchPayload, peerAddr)
		time.Sleep(100 * time.Millisecond)
	}
}

// OpenUDPSocket opens a reusable UDP socket on the given port.
func OpenUDPSocket(port string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", addr)
}
