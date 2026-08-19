// SPDX-License-Identifier: Apache-2.0
//
// A Go-idiomatic SOCKS5 client. This is the milestone-6 redesign of open-rfc's
// src/transport/connectivity-socks5-tunnel.ts, not a mechanical port: the
// upstream ~950-line runtime is an event-driven state machine with injected
// timers and dependency plumbing that Go replaces with a blocking Dial over a
// net.Conn and context deadlines. It speaks RFC 1928 with no-auth and username/
// password (RFC 1929), plus SAP BTP Connectivity's documented method 0x80 JWT
// authentication. See docs/provenance.md.

// Package socks5 dials a target through a SOCKS5 proxy.
package socks5

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	socksVersion        = 0x05
	jwtAuthVersion      = 0x01
	userPassAuthVersion = 0x01
	cmdConnect          = 0x01
	atypIPv4            = 0x01
	atypDomain          = 0x03
	atypIPv6            = 0x04

	methodNoAuth   = 0x00
	methodUserPass = 0x02
	methodSAPJWT   = 0x80
	methodNone     = 0xff
)

var (
	// ErrProtocol reports a malformed SOCKS5 exchange.
	ErrProtocol = errors.New("socks5: protocol error")
	// ErrAuthRejected reports that the proxy refused authentication.
	ErrAuthRejected = errors.New("socks5: authentication rejected")
	// ErrNoAcceptableMethods reports that the proxy accepted none of the
	// offered authentication methods.
	ErrNoAcceptableMethods = errors.New("socks5: no acceptable authentication methods")
)

// replyMessage maps an RFC 1928 CONNECT reply code to its meaning.
var replyMessage = map[byte]string{
	0x00: "succeeded",
	0x01: "general SOCKS server failure",
	0x02: "connection not allowed by ruleset",
	0x03: "network unreachable",
	0x04: "host unreachable",
	0x05: "connection refused",
	0x06: "TTL expired",
	0x07: "command not supported",
	0x08: "address type not supported",
}

// ConnectError reports a non-zero CONNECT reply code from the proxy.
type ConnectError struct {
	Code byte
}

func (e *ConnectError) Error() string {
	msg := replyMessage[e.Code]
	if msg == "" {
		msg = "unknown reply code"
	}
	return fmt.Sprintf("socks5: connect rejected (0x%02x: %s)", e.Code, msg)
}

// Auth negotiates one SOCKS5 authentication method on an established stream.
type Auth interface {
	method() byte
	authenticate(conn io.ReadWriter) error
}

// NoAuth selects the no-authentication method (0x00).
type NoAuth struct{}

func (NoAuth) method() byte                     { return methodNoAuth }
func (NoAuth) authenticate(io.ReadWriter) error { return nil }

// UserPass selects RFC 1929 username/password authentication (0x02).
type UserPass struct {
	Username string
	Password string
}

func (UserPass) method() byte { return methodUserPass }

func (a UserPass) authenticate(conn io.ReadWriter) error {
	if len(a.Username) > 255 || len(a.Password) > 255 {
		return fmt.Errorf("%w: username and password must each be at most 255 bytes", ErrProtocol)
	}
	req := make([]byte, 0, 3+len(a.Username)+len(a.Password))
	req = append(req, userPassAuthVersion, byte(len(a.Username)))
	req = append(req, a.Username...)
	req = append(req, byte(len(a.Password)))
	req = append(req, a.Password...)
	if _, err := conn.Write(req); err != nil {
		return err
	}
	var resp [2]byte
	if _, err := io.ReadFull(conn, resp[:]); err != nil {
		return err
	}
	if resp[0] != userPassAuthVersion {
		return fmt.Errorf("%w: unsupported username/password auth version 0x%02x", ErrProtocol, resp[0])
	}
	if resp[1] != 0x00 {
		return fmt.Errorf("%w: proxy rejected username/password", ErrAuthRejected)
	}
	return nil
}

// SAPJWT selects SAP BTP Connectivity's method 0x80 JWT authentication. The
// LocationID (if set) is base64-encoded on the wire, as the proxy expects.
type SAPJWT struct {
	Token      string
	LocationID string
}

func (SAPJWT) method() byte { return methodSAPJWT }

func (a SAPJWT) authenticate(conn io.ReadWriter) error {
	if len(a.Token) > 0xffff_ffff {
		return fmt.Errorf("%w: JWT token is too long", ErrProtocol)
	}
	var location []byte
	if a.LocationID != "" {
		location = []byte(base64.StdEncoding.EncodeToString([]byte(a.LocationID)))
	}
	if len(location) > 255 {
		return fmt.Errorf("%w: encoded locationId must be at most 255 bytes", ErrProtocol)
	}
	req := make([]byte, 5+len(a.Token)+1+len(location))
	req[0] = jwtAuthVersion
	binary.BigEndian.PutUint32(req[1:], uint32(len(a.Token)))
	copy(req[5:], a.Token)
	req[5+len(a.Token)] = byte(len(location))
	copy(req[6+len(a.Token):], location)
	if _, err := conn.Write(req); err != nil {
		return err
	}
	var resp [2]byte
	if _, err := io.ReadFull(conn, resp[:]); err != nil {
		return err
	}
	if resp[0] != jwtAuthVersion {
		return fmt.Errorf("%w: unsupported JWT auth version 0x%02x", ErrProtocol, resp[0])
	}
	if resp[1] != 0x00 {
		return fmt.Errorf("%w: proxy rejected JWT authentication", ErrAuthRejected)
	}
	return nil
}

// Dialer dials targets through a SOCKS5 proxy.
type Dialer struct {
	// ProxyAddress is the "host:port" of the SOCKS5 proxy.
	ProxyAddress string
	// Auth selects the authentication method. Nil means NoAuth.
	Auth Auth
	// NetDialer dials the proxy. Nil uses a zero-value net.Dialer.
	NetDialer *net.Dialer
}

// Dial connects to addr ("host:port") through the proxy and returns the tunneled
// connection. network must be "tcp". It honours ctx for the whole handshake.
func (d *Dialer) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("%w: only tcp is supported, got %q", ErrProtocol, network)
	}
	host, port, err := splitTarget(addr)
	if err != nil {
		return nil, err
	}
	auth := d.Auth
	if auth == nil {
		auth = NoAuth{}
	}
	nd := d.NetDialer
	if nd == nil {
		nd = &net.Dialer{}
	}
	conn, err := nd.DialContext(ctx, "tcp", d.ProxyAddress)
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := d.handshake(ctx, conn, auth, host, port); err != nil {
		_ = conn.Close()
		return nil, err
	}
	// Clear the handshake deadline; the caller owns timeouts from here on.
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

func (d *Dialer) handshake(ctx context.Context, conn net.Conn, auth Auth, host string, port uint16) error {
	// Greeting: offer exactly the configured method.
	if _, err := conn.Write([]byte{socksVersion, 0x01, auth.method()}); err != nil {
		return err
	}
	var selection [2]byte
	if _, err := io.ReadFull(conn, selection[:]); err != nil {
		return err
	}
	if selection[0] != socksVersion {
		return fmt.Errorf("%w: proxy returned SOCKS version 0x%02x", ErrProtocol, selection[0])
	}
	if selection[1] == methodNone {
		return ErrNoAcceptableMethods
	}
	if selection[1] != auth.method() {
		return fmt.Errorf("%w: proxy selected unoffered method 0x%02x", ErrProtocol, selection[1])
	}
	if err := auth.authenticate(conn); err != nil {
		return err
	}
	if _, err := conn.Write(connectRequest(host, port)); err != nil {
		return err
	}
	return readConnectReply(conn)
}

func connectRequest(host string, port uint16) []byte {
	var req []byte
	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			req = make([]byte, 0, 4+4+2)
			req = append(req, socksVersion, cmdConnect, 0x00, atypIPv4)
			req = append(req, v4...)
		} else {
			req = make([]byte, 0, 4+16+2)
			req = append(req, socksVersion, cmdConnect, 0x00, atypIPv6)
			req = append(req, ip.To16()...)
		}
	} else {
		req = make([]byte, 0, 5+len(host)+2)
		req = append(req, socksVersion, cmdConnect, 0x00, atypDomain, byte(len(host)))
		req = append(req, host...)
	}
	var p [2]byte
	binary.BigEndian.PutUint16(p[:], port)
	return append(req, p[:]...)
}

// readConnectReply reads exactly the CONNECT reply, leaving any subsequent
// tunneled data buffered in the connection.
func readConnectReply(conn net.Conn) error {
	var head [4]byte
	if _, err := io.ReadFull(conn, head[:]); err != nil {
		return err
	}
	if head[0] != socksVersion || head[2] != 0x00 {
		return fmt.Errorf("%w: malformed CONNECT reply", ErrProtocol)
	}
	if head[1] != 0x00 {
		return &ConnectError{Code: head[1]}
	}
	// Consume the bound address so the stream is positioned at tunneled data.
	var addrLen int
	switch head[3] {
	case atypIPv4:
		addrLen = 4
	case atypIPv6:
		addrLen = 16
	case atypDomain:
		var l [1]byte
		if _, err := io.ReadFull(conn, l[:]); err != nil {
			return err
		}
		addrLen = int(l[0])
	default:
		return fmt.Errorf("%w: unsupported bound address type 0x%02x", ErrProtocol, head[3])
	}
	if _, err := io.ReadFull(conn, make([]byte, addrLen+2)); err != nil { // address + 2-byte port
		return err
	}
	return nil
}

func splitTarget(addr string) (string, uint16, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("%w: invalid target address %q", ErrProtocol, addr)
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return "", 0, fmt.Errorf("%w: invalid target port %q", ErrProtocol, portStr)
	}
	if host == "" || len(host) > 255 {
		return "", 0, fmt.Errorf("%w: invalid target host", ErrProtocol)
	}
	return host, uint16(port), nil
}
