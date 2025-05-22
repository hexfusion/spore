package internal

import (
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/quic-go/quic-go"
)

type StreamState int

const (
    StreamHealthy StreamState = iota
    StreamDegraded // temporarily unusable; retryable
    StreamClosed   // peer sent GOAWAY or connection was closed
)

type Stream interface {
    // Send sends a message on the stream
    Send(msg []byte) error
    // Recv blocks until a message is received or error occurs
    Recv() ([]byte, error)
    // Close cleanly closes the stream
    Close() error
}

type quicStream struct {
    mu     sync.Mutex
    stream quic.Stream
	state StreamState
}

func (s *quicStream) Send(msg []byte) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.state == StreamClosed {
        return errors.New("stream closed")
    }

    err := writeFramed(s.stream, msg)
    if err != nil {
        if isTerminalError(err) {
            s.state = StreamClosed
        } else {
            s.state = StreamDegraded
        }
        return err
    }

    s.state = StreamHealthy
    return nil
}

func (s *quicStream) Recv() ([]byte, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
	return readFramed(s.stream)
}

func (s *quicStream) Close() error {
    return s.stream.Close()
}

func isTerminalError(err error) bool {
    if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "application error") {
        return true
    }
    return false
}