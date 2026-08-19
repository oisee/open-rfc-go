// SPDX-License-Identifier: Apache-2.0

package sniffer

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/oisee/open-rfc-go/internal/ni"
)

func readFrame(t *testing.T, r io.Reader) []byte {
	t.Helper()
	var l [4]byte
	if _, err := io.ReadFull(r, l[:]); err != nil {
		t.Fatalf("read frame len: %v", err)
	}
	p := make([]byte, binary.BigEndian.Uint32(l[:]))
	if _, err := io.ReadFull(r, p); err != nil {
		t.Fatalf("read frame body: %v", err)
	}
	return p
}

func frame(t *testing.T, payload []byte) []byte {
	t.Helper()
	f, err := ni.EncodeFrame(payload)
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	return f
}

func TestProxyForwardsAndObserves(t *testing.T) {
	// Fake SAP: read one frame from the client, record it, reply with NI_PONG.
	serverLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer serverLn.Close()
	serverGot := make(chan []byte, 1)
	go func() {
		conn, err := serverLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		serverGot <- readFrame(t, conn)
		conn.Write(frame(t, []byte("NI_PONG\x00")))
	}()

	// Observer collects frames from both directions.
	var mu sync.Mutex
	var frames []Frame
	proxy := &Proxy{
		Target: serverLn.Addr().String(),
		Observe: func(f Frame) {
			mu.Lock()
			defer mu.Unlock()
			cp := append([]byte(nil), f.Payload...)
			frames = append(frames, Frame{Direction: f.Direction, Index: f.Index, Payload: cp, Note: f.Note})
		},
	}
	proxyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go proxy.ServeListener(ctx, proxyLn)

	// Client connects to the proxy and sends one xRFC-XML-looking frame.
	client, err := net.Dial("tcp", proxyLn.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	xmlPayload := []byte("<ROOT><FIELD>value</FIELD></ROOT>")
	if _, err := client.Write(frame(t, xmlPayload)); err != nil {
		t.Fatal(err)
	}

	// The fake server must receive exactly what the client sent (verbatim).
	select {
	case got := <-serverGot:
		if string(got) != string(xmlPayload) {
			t.Fatalf("server got %q, want %q (not forwarded verbatim)", got, xmlPayload)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server never received the forwarded frame")
	}

	// The client must receive the server's reply verbatim.
	reply := readFrame(t, client)
	if string(reply) != "NI_PONG\x00" {
		t.Fatalf("client got reply %q, want NI_PONG", reply)
	}

	// Give the observer a moment to see both frames, then check classification.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(frames)
		mu.Unlock()
		if n >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	var sawXML, sawPong bool
	for _, f := range frames {
		if f.Direction == ClientToServer && contains(f.Note, "xRFC XML") {
			sawXML = true
		}
		if f.Direction == ServerToClient && f.Note == "NI_PONG" {
			sawPong = true
		}
	}
	if !sawXML {
		t.Fatalf("did not observe/classify the xRFC XML frame: %+v", frames)
	}
	if !sawPong {
		t.Fatalf("did not observe/classify the NI_PONG reply: %+v", frames)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
