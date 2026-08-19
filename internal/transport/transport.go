// SPDX-License-Identifier: Apache-2.0
//
// Go-idiomatic redesign of open-rfc src/transport/ni-socket.ts (commit 847036d,
// Copyright 2026 Marian Zeis, Apache-2.0). The upstream 961-line class is
// event-driven JS plumbing — pause/resume backpressure, coalesced-byte
// adoption, timer schedulers, a bounded frame queue — around a plain job:
// carry length-prefixed NI records over one TCP connection so a reply cannot be
// mistaken for a later call's. Go collapses that to a blocking net.Conn plus
// deadlines and the streaming ni.FrameDecoder. The SAProuter/SOCKS5 routed
// adoption path is milestone 6. See docs/provenance.md.

// Package transport carries bounded NI length-prefixed records over one TCP
// connection.
package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/oisee/open-rfc-go/internal/ni"
	"github.com/oisee/open-rfc-go/internal/saprouter"
)

// ErrClosed reports a read or write on a connection the peer has closed.
var ErrClosed = errors.New("transport: connection closed")

// Options tunes a Transport. A zero timeout disables that deadline.
type Options struct {
	MaxPayloadLength int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
}

func (o Options) maxPayload() int {
	if o.MaxPayloadLength > 0 {
		return o.MaxPayloadLength
	}
	return ni.DefaultMaxPayloadLength
}

// Transport is one TCP connection framed as NI records.
type Transport struct {
	conn  net.Conn
	dec   *ni.FrameDecoder
	queue [][]byte
	chunk []byte
	opts  Options
}

// Dial opens a TCP connection to addr ("host:port") and frames it as NI.
func Dial(ctx context.Context, network, addr string, opts Options) (*Transport, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("transport: dial %s: %w", addr, err)
	}
	return New(conn, opts), nil
}

// ContextDialer dials a (possibly tunneled) TCP connection. *socks5.Dialer
// satisfies it, so a SOCKS5 proxy can front the connection.
type ContextDialer interface {
	Dial(ctx context.Context, network, addr string) (net.Conn, error)
}

// DialOptions extends Options with routed/proxied dialing. A nil Proxy dials
// directly; a nil Router connects straight to addr.
type DialOptions struct {
	Options
	// Proxy, if set, establishes the first TCP hop through a proxy (e.g. SOCKS5)
	// instead of a direct dial.
	Proxy ContextDialer
	// Router, if set, is a completed SAProuter route; the dialer connects to its
	// first hop and performs the NI_ROUTE handshake, tunneling to the final hop.
	Router *saprouter.Route
	// RouterNIVersion overrides the SAProuter NI version (0 = default).
	RouterNIVersion int
}

// DialWith opens a connection, optionally through a SOCKS5 proxy and/or a
// SAProuter route, and frames it as NI. When Router is set, addr is ignored (the
// route carries the target); otherwise the dialer connects to addr.
func DialWith(ctx context.Context, network, addr string, opts DialOptions) (*Transport, error) {
	dialAddr := addr
	if opts.Router != nil {
		dialAddr = net.JoinHostPort(opts.Router.FirstHop.Host, opts.Router.FirstHop.Service)
	}
	var conn net.Conn
	var err error
	if opts.Proxy != nil {
		conn, err = opts.Proxy.Dial(ctx, network, dialAddr)
	} else {
		var d net.Dialer
		conn, err = d.DialContext(ctx, network, dialAddr)
	}
	if err != nil {
		return nil, fmt.Errorf("transport: dial %s: %w", dialAddr, err)
	}
	if opts.Router != nil {
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		}
		if err := saprouter.PerformRoute(conn, opts.Router, opts.RouterNIVersion); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("transport: saprouter handshake: %w", err)
		}
		_ = conn.SetDeadline(time.Time{})
	}
	return New(conn, opts.Options), nil
}

// New adopts an already-connected stream (for example a routed socket).
func New(conn net.Conn, opts Options) *Transport {
	dec, _ := ni.NewFrameDecoder(opts.maxPayload())
	return &Transport{conn: conn, dec: dec, chunk: make([]byte, 32*1024), opts: opts}
}

// Send writes one NI record carrying payload.
func (t *Transport) Send(payload []byte) error {
	frame, err := ni.EncodeFrame(payload)
	if err != nil {
		return err
	}
	if t.opts.WriteTimeout > 0 {
		_ = t.conn.SetWriteDeadline(time.Now().Add(t.opts.WriteTimeout))
	}
	if _, err := t.conn.Write(frame); err != nil {
		return fmt.Errorf("transport: write: %w", err)
	}
	return nil
}

// Receive returns the payload of the next complete NI record, reading from the
// connection until one is available. If ctx carries a deadline it bounds the
// read; ReadTimeout bounds each individual read.
func (t *Transport) Receive(ctx context.Context) ([]byte, error) {
	for len(t.queue) == 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		t.applyReadDeadline(ctx)
		n, err := t.conn.Read(t.chunk)
		if n > 0 {
			frames, derr := t.dec.Push(t.chunk[:n])
			if derr != nil {
				return nil, derr
			}
			t.queue = append(t.queue, frames...)
		}
		if err != nil {
			if len(t.queue) > 0 {
				break
			}
			if errors.Is(err, io.EOF) {
				return nil, ErrClosed
			}
			return nil, fmt.Errorf("transport: read: %w", err)
		}
	}
	frame := t.queue[0]
	t.queue = t.queue[1:]
	return frame, nil
}

func (t *Transport) applyReadDeadline(ctx context.Context) {
	var deadline time.Time
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	}
	if t.opts.ReadTimeout > 0 {
		perRead := time.Now().Add(t.opts.ReadTimeout)
		if deadline.IsZero() || perRead.Before(deadline) {
			deadline = perRead
		}
	}
	if !deadline.IsZero() {
		_ = t.conn.SetReadDeadline(deadline)
	}
}

// Close closes the underlying connection.
func (t *Transport) Close() error { return t.conn.Close() }
