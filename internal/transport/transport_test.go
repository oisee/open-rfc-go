// SPDX-License-Identifier: Apache-2.0
//
// Tests for the NI transport redesign. Original work (upstream ni-socket.test
// drives Node stream mocks that have no Go analogue); these exercise the framing
// contract over net.Pipe. See docs/provenance.md.

package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/oisee/open-rfc-go/internal/ni"
)

func TestSendReceiveRoundTrip(t *testing.T) {
	a, b := net.Pipe()
	client := New(a, Options{})
	server := New(b, Options{})
	defer client.Close()
	defer server.Close()

	payloads := [][]byte{[]byte("hello"), []byte("second frame"), {}}
	go func() {
		for _, p := range payloads {
			if err := client.Send(p); err != nil {
				t.Errorf("send: %v", err)
				return
			}
		}
	}()
	for i, want := range payloads {
		got, err := server.Receive(context.Background())
		if err != nil {
			t.Fatalf("receive %d: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("frame %d = %q, want %q", i, got, want)
		}
	}
}

func TestReceiveReassemblesSplitFrame(t *testing.T) {
	a, b := net.Pipe()
	client := New(b, Options{})
	defer client.Close()
	defer a.Close()

	// Write one NI frame in two raw chunks; the decoder must reassemble it.
	frame, _ := ni.EncodeFrame([]byte("reassembled"))
	go func() {
		a.Write(frame[:3])
		time.Sleep(5 * time.Millisecond)
		a.Write(frame[3:])
	}()
	got, err := client.Receive(context.Background())
	if err != nil || string(got) != "reassembled" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestReceiveClosedConnection(t *testing.T) {
	a, b := net.Pipe()
	New(b, Options{})
	a.Close()
	client := New(b, Options{})
	if _, err := client.Receive(context.Background()); err == nil {
		t.Fatal("expected error on closed connection")
	}
}

func TestReceiveContextCancel(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	client := New(b, Options{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Receive(ctx); err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestReceiveRejectsOversizedLength(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	client := New(b, Options{MaxPayloadLength: 16})
	// Advertise a payload larger than the configured maximum.
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, 1_000_000)
	go func() { a.Write(header) }()
	if _, err := client.Receive(context.Background()); err == nil {
		t.Fatal("expected oversize rejection")
	}
}
