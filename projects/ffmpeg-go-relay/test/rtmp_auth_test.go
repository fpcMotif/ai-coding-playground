package test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"ffmpeg-go-relay/internal/auth"
	"ffmpeg-go-relay/internal/logger"
	"ffmpeg-go-relay/internal/relay"
	"ffmpeg-go-relay/internal/rtmp"
)

func TestRelayRTMPAuth(t *testing.T) {
	// 1. Mock Upstream that expects valid RTMP handshake
	upstreamReady := make(chan struct{})
	upstreamReceivedConnect := make(chan []byte, 1)

	upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("upstream listen: %v", err)
	}
	defer upstreamListener.Close()

	go func() {
		close(upstreamReady)
		conn, err := upstreamListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Server side of upstream handshake
		fmt.Println("Upstream: Starting Handshake")
		if err := rtmp.ServerHandshake(conn, nil); err != nil {
			// t.Logf("upstream handshake failed: %v", err)
			return
		}
		fmt.Println("Upstream: Handshake Done")

		// Read the forwarded connect command
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		upstreamReceivedConnect <- buf[:n]
	}()

	<-upstreamReady

	// 2. Start Relay
	relayListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("relay listen: %v", err)
	}
	relayAddr := relayListener.Addr().String()
	relayListener.Close()

	authenticator := auth.NewTokenAuthenticator([]string{"secret-token"})

	server := &relay.Server{
		ListenAddr: relayAddr,
		Upstream:   upstreamListener.Addr().String(),
		Log:        logger.New(),
		Auth:       authenticator,
		ReadBuf:    4096,
		WriteBuf:   4096,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Run(ctx)
	time.Sleep(100 * time.Millisecond)

	// 3. Client Connection
	client, err := net.Dial("tcp", relayAddr)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer client.Close()

	// Client Handshake
	fmt.Println("Client: Starting Handshake")
	if err := rtmp.ClientHandshake(client, nil); err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	fmt.Println("Client: Handshake Done")

	// 4. Send Connect Command
	// Payload: ["connect", 1.0, {app: "live", token: "secret-token"}]
	payload := encodeConnectCommand("live", "secret-token")

	// Header
	header := make([]byte, 12)
	header[0] = 0x03 // fmt 0, csid 3
	// timestamp 0 (3 bytes)
	// length (3 bytes)
	length := uint32(len(payload))
	header[4] = byte(length >> 16)
	header[5] = byte(length >> 8)
	header[6] = byte(length)
	header[7] = 20 // Type AMF0 Command
	// StreamID 0 (4 bytes LE) -> already 0s

	// Write Header + Payload
	if _, err := client.Write(header); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := client.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	// 5. Verify Upstream Received it
	select {
	case data := <-upstreamReceivedConnect:
		// Basic check that we got bytes forwarded
		if len(data) == 0 {
			t.Fatal("upstream received empty data")
		}
		// In a real test we would decode this again to verify content
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for upstream to receive data")
	}
}

func encodeConnectCommand(app, token string) []byte {
	buf := new(bytes.Buffer)

	// String "connect"
	writeAMFString(buf, "connect")
	// Number 1.0 (Transaction ID)
	writeAMFNumber(buf, 1.0)
	// Object
	writeAMFObjectStart(buf)
	writeAMFObjectProperty(buf, "app", app)
	writeAMFObjectProperty(buf, "token", token) // Our custom auth field
	writeAMFObjectEnd(buf)

	return buf.Bytes()
}

func writeAMFString(w io.Writer, s string) {
	w.Write([]byte{0x02}) // String Marker
	length := uint16(len(s))
	binary.Write(w, binary.BigEndian, length)
	w.Write([]byte(s))
}

func writeAMFNumber(w io.Writer, n float64) {
	w.Write([]byte{0x00}) // Number Marker
	binary.Write(w, binary.BigEndian, n)
}

func writeAMFObjectStart(w io.Writer) {
	w.Write([]byte{0x03}) // Object Marker
}

func writeAMFObjectEnd(w io.Writer) {
	w.Write([]byte{0x00, 0x00, 0x09}) // Empty string key + Object End Marker
}

func writeAMFObjectProperty(w io.Writer, key, val string) {
	// Key string
	length := uint16(len(key))
	binary.Write(w, binary.BigEndian, length)
	w.Write([]byte(key))

	// Value (assuming string for simplicity)
	writeAMFString(w, val)
}
