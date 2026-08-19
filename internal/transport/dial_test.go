// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/oisee/open-rfc-go/internal/saprouter"
)

func writeNIFrame(w io.Writer, payload []byte) error {
	frame := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(frame, uint32(len(payload)))
	copy(frame[4:], payload)
	_, err := w.Write(frame)
	return err
}

func readNIFrame(r io.Reader) ([]byte, error) {
	var l [4]byte
	if _, err := io.ReadFull(r, l[:]); err != nil {
		return nil, err
	}
	p := make([]byte, binary.BigEndian.Uint32(l[:]))
	_, err := io.ReadFull(r, p)
	return p, err
}

func TestDialWithRouter(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		if _, err := readNIFrame(conn); err != nil { // the NI_ROUTE request
			return
		}
		if err := writeNIFrame(conn, []byte("NI_PONG\x00")); err != nil {
			return
		}
		io.Copy(conn, conn) // act as the tunneled gateway: echo NI frames
	}()

	route, err := saprouter.Admit("/H/127.0.0.1/S/" + port + "/H/sap.example/S/3300")
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tr, err := DialWith(ctx, "tcp", "unused:0", DialOptions{Router: route})
	if err != nil {
		t.Fatalf("DialWith router: %v", err)
	}
	defer tr.Close()
	if err := tr.Send([]byte("hello-gateway")); err != nil {
		t.Fatalf("send: %v", err)
	}
	got, err := tr.Receive(ctx)
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if string(got) != "hello-gateway" {
		t.Fatalf("echo = %q", got)
	}
}

func TestDialWithRouterRejected(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		readNIFrame(conn)
		e := make([]byte, 20)
		copy(e, "NI_RTERR\x00")
		e[9] = 40
		rc := int32(-94)
		binary.BigEndian.PutUint32(e[12:], uint32(rc))
		writeNIFrame(conn, e)
	}()
	route, _ := saprouter.Admit("/H/127.0.0.1/S/" + port + "/H/sap.example/S/3300")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := DialWith(ctx, "tcp", "unused:0", DialOptions{Router: route}); err == nil {
		t.Fatalf("expected router rejection error")
	}
}

type stubDialer struct {
	gotNetwork, gotAddr string
	conn                net.Conn
}

func (d *stubDialer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	d.gotNetwork, d.gotAddr = network, addr
	return d.conn, nil
}

func TestDialWithProxy(t *testing.T) {
	clientEnd, gatewayEnd := net.Pipe()
	go io.Copy(gatewayEnd, gatewayEnd) // fake gateway behind the proxy: echo NI frames
	stub := &stubDialer{conn: clientEnd}

	tr, err := DialWith(context.Background(), "tcp", "sap.example:3300", DialOptions{Proxy: stub})
	if err != nil {
		t.Fatalf("DialWith proxy: %v", err)
	}
	defer tr.Close()
	if stub.gotAddr != "sap.example:3300" || stub.gotNetwork != "tcp" {
		t.Fatalf("proxy dialed %s/%s, want tcp/sap.example:3300", stub.gotNetwork, stub.gotAddr)
	}
	if err := tr.Send([]byte("through-proxy")); err != nil {
		t.Fatalf("send: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	got, err := tr.Receive(ctx)
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if string(got) != "through-proxy" {
		t.Fatalf("echo = %q", got)
	}
}

// TestDialWithProxyAndRouter proves the two compose: the proxy dials the first
// router hop, then the NI_ROUTE handshake runs over the proxied connection.
func TestDialWithProxyAndRouter(t *testing.T) {
	clientEnd, routerEnd := net.Pipe()
	go func() {
		defer routerEnd.Close()
		readNIFrame(routerEnd)
		writeNIFrame(routerEnd, []byte("NI_PONG\x00"))
		io.Copy(routerEnd, routerEnd)
	}()
	stub := &stubDialer{conn: clientEnd}
	route, _ := saprouter.Admit("/H/router.example/S/3299/H/sap.example/S/3300")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tr, err := DialWith(ctx, "tcp", "unused:0", DialOptions{Proxy: stub, Router: route})
	if err != nil {
		t.Fatalf("DialWith proxy+router: %v", err)
	}
	defer tr.Close()
	// The proxy must have been asked for the first router hop, not the target.
	if stub.gotAddr != "router.example:3299" {
		t.Fatalf("proxy dialed %q, want router.example:3299", stub.gotAddr)
	}
	if err := tr.Send([]byte("hi")); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got, err := tr.Receive(ctx); err != nil || string(got) != "hi" {
		t.Fatalf("round-trip = %q, %v", got, err)
	}
}
