package transport

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

// QUICConn wraps a QUIC connection + stream so it satisfies net.Conn.
type QUICConn struct {
	conn   *quic.Conn
	stream *quic.Stream
}

func (q *QUICConn) Read(b []byte) (int, error)         { return q.stream.Read(b) }
func (q *QUICConn) Write(b []byte) (int, error)        { return q.stream.Write(b) }
func (q *QUICConn) Close() error                       { return q.conn.CloseWithError(0, "closed") }
func (q *QUICConn) LocalAddr() net.Addr                { return q.conn.LocalAddr() }
func (q *QUICConn) RemoteAddr() net.Addr               { return q.conn.RemoteAddr() }
func (q *QUICConn) SetDeadline(t time.Time) error      { return q.stream.SetDeadline(t) }
func (q *QUICConn) SetReadDeadline(t time.Time) error  { return q.stream.SetReadDeadline(t) }
func (q *QUICConn) SetWriteDeadline(t time.Time) error { return q.stream.SetWriteDeadline(t) }

// NewTransport creates a QUIC transport from an existing UDP socket.
// The same socket is used for hole punching before passing here.
func NewTransport(udpConn *net.UDPConn) *quic.Transport {
	return &quic.Transport{Conn: udpConn}
}

// Listen accepts a single incoming QUIC connection and opens a stream on it.
func Listen(quicTransport *quic.Transport, timeout time.Duration) (*QUICConn, error) {
	tlsConfig, err := serverTLS()
	if err != nil {
		return nil, err
	}
	listener, err := quicTransport.Listen(tlsConfig, &quic.Config{})
	if err != nil {
		return nil, err
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := listener.Accept(ctx)
	if err != nil {
		return nil, err
	}
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	return &QUICConn{conn: conn, stream: stream}, nil
}

// Listener is a persistent QUIC listener that can accept many connections
// over its lifetime — unlike Listen() which closes after the first accept.
type Listener struct {
	inner *quic.Listener
}

// NewListener opens a persistent QUIC listener on the transport.
func NewListener(qt *quic.Transport) (*Listener, error) {
	tlsConfig, err := serverTLS()
	if err != nil {
		return nil, err
	}
	l, err := qt.Listen(tlsConfig, &quic.Config{})
	if err != nil {
		return nil, err
	}
	return &Listener{inner: l}, nil
}

// Accept blocks until the next incoming QUIC connection arrives.
func (l *Listener) Accept(ctx context.Context) (*QUICConn, error) {
	conn, err := l.inner.Accept(ctx)
	if err != nil {
		return nil, err
	}
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	return &QUICConn{conn: conn, stream: stream}, nil
}

// Close shuts down the listener.
func (l *Listener) Close() error {
	return l.inner.Close()
}

// Dial connects to a peer via QUIC and opens a stream.
func Dial(quicTransport *quic.Transport, peerAddr net.Addr, timeout time.Duration) (*QUICConn, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"chatiss"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := quicTransport.Dial(ctx, peerAddr, tlsConfig, &quic.Config{})
	if err != nil {
		return nil, err
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &QUICConn{conn: conn, stream: stream}, nil
}

// Make this stuff TLS so going through port 443 has less issues
func serverTLS() (*tls.Config, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{certDER},
			PrivateKey:  key,
		}},
		NextProtos: []string{"chatiss"},
	}, nil
}
