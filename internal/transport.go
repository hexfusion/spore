package internal

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"sync"

	"github.com/quic-go/quic-go"
)

type Transport interface {
	// Unary sends a single message and returns the response or error.
    Unary(ctx context.Context, peerID string, msg []byte) ([]byte, error)
	// Stream opens or retrieves a persistent bidirectional stream.
    // The caller can use the stream for ongoing send/recv.
    Stream(ctx context.Context, peerID string) (Stream, error)
	// Close shutsdown the transport and all related streams.
    Close() error
	// Healthy returns true if the peer is considered healthy
    Healthy(peerID string) bool
	// ListPeers returns a list of peers managed by this transport.
    ListPeers() []string
}

type quicTransport struct {
    tlsConf   *tls.Config
    quicConf  *quic.Config
    peers     map[string]quic.Connection
    streams   map[string]*quicStream
    mu        sync.Mutex
}

func NewQUICTransport(tlsConf *tls.Config, quicConf *quic.Config) *quicTransport {
    return &quicTransport{
        tlsConf:  tlsConf,
        quicConf: quicConf,
        peers:    make(map[string]quic.Connection),
        streams:  make(map[string]*quicStream),
    }
}

func (t *quicTransport) Stream(ctx context.Context, peerID string) (Stream, error) {
    t.mu.Lock()
    if s, ok := t.streams[peerID]; ok {
        t.mu.Unlock()
        return s, nil
    }
    t.mu.Unlock()

    conn, err := t.getOrDial(ctx, peerID)
    if err != nil {
        return nil, err
    }

    stream, err := conn.OpenStreamSync(ctx)
    if err != nil {
        return nil, err
    }

    qs := &quicStream{stream: stream}
    t.mu.Lock()
    t.streams[peerID] = qs
    t.mu.Unlock()
    return qs, nil
}

func (t *quicTransport) getOrDial(ctx context.Context, peerID string) (quic.Connection, error) {
    t.mu.Lock()
    defer t.mu.Unlock()

    if conn, ok := t.peers[peerID]; ok {
        return conn, nil
    }

    conn, err := quic.DialAddr(ctx, peerID, t.tlsConf, t.quicConf)
    if err != nil {
        return nil, err
    }

    t.peers[peerID] = conn
    return conn, nil
}

func (t *quicTransport) Close() error {
    t.mu.Lock()
    defer t.mu.Unlock()
    var firstErr error
    for _, s := range t.streams {
        if err := s.Close(); err != nil && firstErr == nil {
            firstErr = err
        }
    }
    for _, conn := range t.peers {
        _ = conn.CloseWithError(0, "shutdown")
    }
    return firstErr
}

// Write: 4-byte length prefix + payload
func writeFramed(w io.Writer, msg []byte) error {
    length := uint32(len(msg))
    if err := binary.Write(w, binary.BigEndian, length); err != nil {
        return err
    }
    _, err := w.Write(msg)
    return err
}

// Read: read length prefix then payload
func readFramed(r io.Reader) ([]byte, error) {
    var length uint32
    if err := binary.Read(r, binary.BigEndian, &length); err != nil {
        return nil, err
    }
    buf := make([]byte, length)
    _, err := io.ReadFull(r, buf)
    return buf, err
}