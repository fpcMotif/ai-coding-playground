package rtmp

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestHandshake(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	clientRand := bytes.NewReader(make([]byte, handshakeSize))
	serverRand := bytes.NewReader(make([]byte, handshakeSize))

	clientErr := make(chan error, 1)
	serverErr := make(chan error, 1)

	go func() {
		clientErr <- ClientHandshake(clientConn, &HandshakeOptions{
			Now:  func() uint32 { return 1 },
			Rand: clientRand,
		})
	}()

	go func() {
		serverErr <- ServerHandshake(serverConn, &HandshakeOptions{
			Now:  func() uint32 { return 2 },
			Rand: serverRand,
		})
	}()

	timeout := time.After(2 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case err := <-clientErr:
			if err != nil {
				t.Fatalf("client handshake failed: %v", err)
			}
		case err := <-serverErr:
			if err != nil {
				t.Fatalf("server handshake failed: %v", err)
			}
		case <-timeout:
			t.Fatal("handshake timed out")
		}
	}
}
