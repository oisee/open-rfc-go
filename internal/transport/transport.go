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
