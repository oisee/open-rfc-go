// SPDX-License-Identifier: Apache-2.0

package socks5

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// fakeServer is a minimal, single-connection SOCKS5 server for tests.
type fakeServer struct {
	method        byte   // method to select in the greeting response
	authStatus    byte   // 0 = ok
	replyCode     byte   // CONNECT reply code
	trailing      []byte // bytes to append to the CONNECT reply in the same write
	gotUser       string
	gotPass       string
	gotToken      string
	gotLocation   string
	gotTargetHost string
	gotTargetPort uint16
}

func (s *fakeServer) serve(t *testing.T, ln net.Listener, ready chan<- struct{}) {
	close(ready)
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	// Greeting.
	var g [2]byte
	if _, err := io.ReadFull(conn, g[:]); err != nil {
		return
	}
	methods := make([]byte, g[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}
	conn.Write([]byte{socksVersion, s.method})
	if s.method == methodNone {
		return
	}
	switch s.method {
	case methodUserPass:
		var h [2]byte
		io.ReadFull(conn, h[:])
		user := make([]byte, h[1])
		io.ReadFull(conn, user)
		var pl [1]byte
		io.ReadFull(conn, pl[:])
		pass := make([]byte, pl[0])
		io.ReadFull(conn, pass)
		s.gotUser, s.gotPass = string(user), string(pass)
		conn.Write([]byte{userPassAuthVersion, s.authStatus})
	case methodSAPJWT:
		var h [5]byte
		io.ReadFull(conn, h[:])
		tokLen := binary.BigEndian.Uint32(h[1:])
		token := make([]byte, tokLen)
		io.ReadFull(conn, token)
		var ll [1]byte
		io.ReadFull(conn, ll[:])
		loc := make([]byte, ll[0])
		io.ReadFull(conn, loc)
		s.gotToken, s.gotLocation = string(token), string(loc)
		conn.Write([]byte{jwtAuthVersion, s.authStatus})
	}
	if s.authStatus != 0 {
		return
	}
	// CONNECT request.
	var c [4]byte
	io.ReadFull(conn, c[:])
	switch c[3] {
	case atypIPv4:
		a := make([]byte, 4)
		io.ReadFull(conn, a)
		s.gotTargetHost = net.IP(a).String()
	case atypIPv6:
		a := make([]byte, 16)
		io.ReadFull(conn, a)
		s.gotTargetHost = net.IP(a).String()
	case atypDomain:
		var l [1]byte
		io.ReadFull(conn, l[:])
		a := make([]byte, l[0])
		io.ReadFull(conn, a)
		s.gotTargetHost = string(a)
	}
	var p [2]byte
	io.ReadFull(conn, p[:])
	s.gotTargetPort = binary.BigEndian.Uint16(p[:])

	reply := []byte{socksVersion, s.replyCode, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0}
	reply = append(reply, s.trailing...)
	conn.Write(reply)
	if s.replyCode != 0x00 {
		return
	}
	// Echo tunneled data until the client closes.
	io.Copy(conn, conn)
}

func run(t *testing.T, s *fakeServer) (*Dialer, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{})
	go s.serve(t, ln, ready)
	<-ready
	return &Dialer{ProxyAddress: ln.Addr().String()}, func() { ln.Close() }
}

func TestNoAuthConnectAndTunnel(t *testing.T) {
	s := &fakeServer{method: methodNoAuth, replyCode: 0x00}
	d, closeFn := run(t, s)
	defer closeFn()
	conn, err := d.Dial(context.Background(), "tcp", "sap.example:3300")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if s.gotTargetHost != "sap.example" || s.gotTargetPort != 3300 {
		t.Fatalf("server saw target %s:%d", s.gotTargetHost, s.gotTargetPort)
	}
	conn.Write([]byte("ping"))
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(buf) != "ping" {
		t.Fatalf("echo = %q", buf)
	}
}

func TestTrailingBytesAfterReplyPreserved(t *testing.T) {
	s := &fakeServer{method: methodNoAuth, replyCode: 0x00, trailing: []byte("EARLY")}
	d, closeFn := run(t, s)
	defer closeFn()
	conn, err := d.Dial(context.Background(), "tcp", "10.0.0.1:3300")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	// The bytes the server sent right after the reply must survive.
	buf := make([]byte, 5)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read trailing: %v", err)
	}
	if string(buf) != "EARLY" {
		t.Fatalf("trailing bytes lost: %q", buf)
	}
	if s.gotTargetHost != "10.0.0.1" {
		t.Fatalf("ipv4 target = %s", s.gotTargetHost)
	}
}

func TestUserPassAuth(t *testing.T) {
	s := &fakeServer{method: methodUserPass, replyCode: 0x00}
	d, closeFn := run(t, s)
	defer closeFn()
	d.Auth = UserPass{Username: "proxy-user", Password: "secret"}
	conn, err := d.Dial(context.Background(), "tcp", "sap.example:3300")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.Close()
	if s.gotUser != "proxy-user" || s.gotPass != "secret" {
		t.Fatalf("server saw creds %q/%q", s.gotUser, s.gotPass)
	}
}

func TestUserPassRejected(t *testing.T) {
	s := &fakeServer{method: methodUserPass, authStatus: 0x01}
	d, closeFn := run(t, s)
	defer closeFn()
	d.Auth = UserPass{Username: "u", Password: "bad"}
	if _, err := d.Dial(context.Background(), "tcp", "sap.example:3300"); !errors.Is(err, ErrAuthRejected) {
		t.Fatalf("dial = %v, want ErrAuthRejected", err)
	}
}

func TestSAPJWTAuth(t *testing.T) {
	s := &fakeServer{method: methodSAPJWT, replyCode: 0x00}
	d, closeFn := run(t, s)
	defer closeFn()
	d.Auth = SAPJWT{Token: "eyJ.jwt.token", LocationID: "my-cloud-connector"}
	conn, err := d.Dial(context.Background(), "tcp", "sap.example:3300")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.Close()
	if s.gotToken != "eyJ.jwt.token" {
		t.Fatalf("server saw token %q", s.gotToken)
	}
	wantLoc := base64.StdEncoding.EncodeToString([]byte("my-cloud-connector"))
	if s.gotLocation != wantLoc {
		t.Fatalf("locationId = %q, want base64 %q", s.gotLocation, wantLoc)
	}
}

func TestConnectRejected(t *testing.T) {
	s := &fakeServer{method: methodNoAuth, replyCode: 0x05} // connection refused
	d, closeFn := run(t, s)
	defer closeFn()
	_, err := d.Dial(context.Background(), "tcp", "sap.example:3300")
	var ce *ConnectError
	if !errors.As(err, &ce) || ce.Code != 0x05 {
		t.Fatalf("dial = %v, want ConnectError 0x05", err)
	}
}

func TestNoAcceptableMethods(t *testing.T) {
	s := &fakeServer{method: methodNone}
	d, closeFn := run(t, s)
	defer closeFn()
	if _, err := d.Dial(context.Background(), "tcp", "sap.example:3300"); !errors.Is(err, ErrNoAcceptableMethods) {
		t.Fatalf("dial = %v, want ErrNoAcceptableMethods", err)
	}
}

func TestConnectRequestAddressTypes(t *testing.T) {
	cases := []struct {
		host string
		atyp byte
	}{
		{"10.0.0.1", atypIPv4},
		{"::1", atypIPv6},
		{"sap.example", atypDomain},
	}
	for _, c := range cases {
		req := connectRequest(c.host, 3300)
		if req[3] != c.atyp {
			t.Fatalf("host %q atyp = 0x%02x, want 0x%02x", c.host, req[3], c.atyp)
		}
		if !bytes.HasPrefix(req, []byte{socksVersion, cmdConnect, 0x00, c.atyp}) {
			t.Fatalf("host %q bad prefix: % x", c.host, req[:4])
		}
		if binary.BigEndian.Uint16(req[len(req)-2:]) != 3300 {
			t.Fatalf("host %q port not trailing", c.host)
		}
	}
}

func TestContextDeadline(t *testing.T) {
	// A server that accepts but never answers the greeting.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			time.Sleep(2 * time.Second)
			conn.Close()
		}
	}()
	d := &Dialer{ProxyAddress: ln.Addr().String()}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := d.Dial(ctx, "tcp", "sap.example:3300"); err == nil {
		t.Fatalf("expected a deadline error")
	}
}

func TestUnsupportedNetwork(t *testing.T) {
	d := &Dialer{ProxyAddress: "127.0.0.1:1"}
	if _, err := d.Dial(context.Background(), "udp", "sap.example:3300"); !errors.Is(err, ErrProtocol) {
		t.Fatalf("udp = %v, want ErrProtocol", err)
	}
}
