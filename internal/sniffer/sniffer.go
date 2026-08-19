// SPDX-License-Identifier: Apache-2.0
//
// A framing-aware RFC proxy/sniffer. Original work for open-rfc-go. It sits
// between an RFC client and a real SAP gateway, forwards the raw byte stream
// unchanged in both directions, and hands each complete NI record to an
// observer for logging/capture. Because it forwards bytes verbatim it is
// transport-transparent; the decoding is purely observational.
//
// Uses: watch exactly what a client sends (learn GW_REGISTER and the inbound
// serving flow for the RFC-server track), capture real wire vectors for deep
// xRFC (STFC_DEEP_*), and validate the decoders against live bidirectional
// traffic. See docs/porting-plan.md.

// Package sniffer proxies and observes NI-framed RFC traffic.
package sniffer

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oisee/open-rfc-go/internal/gateway"
	"github.com/oisee/open-rfc-go/internal/ni"
)

// Direction is which way an observed frame travels.
type Direction string

const (
	// ClientToServer is a frame from the RFC client toward SAP.
	ClientToServer Direction = "C->S"
	// ServerToClient is a frame from SAP toward the RFC client.
	ServerToClient Direction = "S->C"
)

// Frame is one observed NI record.
type Frame struct {
	Direction Direction
	ConnID    int    // per-proxy connection ordinal, starting at 1
	Label     string // the proxy's role/port tag (e.g. "gw", "disp")
	Index     int    // per-(connection,direction) ordinal, starting at 0
	Payload   []byte // the NI payload (without the 4-byte length prefix)
	Note      string // best-effort classification
}

// Observer receives every complete frame. It must not retain Payload beyond the
// call unless it copies it, and it must be safe for concurrent use (frames from
// both directions arrive on separate goroutines).
type Observer func(Frame)

// Proxy forwards NI traffic between a listener and one target, observing frames.
type Proxy struct {
	// Target is the "host:port" of the real SAP gateway.
	Target string
	// MaxPayload bounds a single observed NI payload (0 = default).
	MaxPayload int
	// Observe receives every frame. Required.
	Observe Observer
	// DialTimeout bounds dialing the target (0 = no explicit timeout).
	DialTimeout time.Duration
	// Label tags every observed frame with this proxy's role/port (optional).
	Label string
	// Raw tees each read chunk verbatim instead of reassembling NI frames. Use
	// for non-NI transports (WebSocket, HTTP) so their bytes are still captured.
	Raw bool

	connSeq atomic.Int64
}

// Serve listens on listenAddr and proxies every accepted connection to Target
// until ctx is cancelled. It blocks.
func (p *Proxy) Serve(ctx context.Context, listenAddr string) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("sniffer: listen %s: %w", listenAddr, err)
	}
	return p.ServeListener(ctx, ln)
}

// ServeListener proxies every connection accepted on ln until ctx is cancelled.
// It closes ln on return. It blocks.
func (p *Proxy) ServeListener(ctx context.Context, ln net.Listener) error {
	defer ln.Close()
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("sniffer: accept: %w", err)
		}
		go func() {
			if err := p.handle(ctx, conn); err != nil {
				// Connection-level errors are expected on close; ignore.
				_ = err
			}
		}()
	}
}

// handle proxies one client connection to a fresh target connection.
func (p *Proxy) handle(ctx context.Context, client net.Conn) error {
	defer client.Close()
	var d net.Dialer
	dialCtx := ctx
	if p.DialTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, p.DialTimeout)
		defer cancel()
	}
	server, err := d.DialContext(dialCtx, "tcp", p.Target)
	if err != nil {
		return fmt.Errorf("sniffer: dial target %s: %w", p.Target, err)
	}
	defer server.Close()

	connID := int(p.connSeq.Add(1))
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); p.pump(connID, ClientToServer, client, server) }()
	go func() { defer wg.Done(); p.pump(connID, ServerToClient, server, client) }()
	wg.Wait()
	return nil
}

// pump copies src→dst verbatim while reassembling and observing NI frames. On
// any read/write error it closes both ends so the peer pump also unwinds.
func (p *Proxy) pump(connID int, dir Direction, src, dst net.Conn) {
	if p.Raw {
		p.pumpRaw(connID, dir, src, dst)
		return
	}
	max := p.MaxPayload
	if max <= 0 {
		max = ni.DefaultMaxPayloadLength
	}
	dec, _ := ni.NewFrameDecoder(max)
	buf := make([]byte, 32*1024)
	index := 0
	defer func() { _ = src.Close(); _ = dst.Close() }()
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			// Forward the exact bytes first, so the proxy never alters the wire.
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return
			}
			frames, derr := dec.Push(buf[:n])
			if derr != nil {
				// Lost framing sync; keep forwarding raw but stop observing.
				p.Observe(Frame{Direction: dir, ConnID: connID, Label: p.Label, Index: index, Note: "framing error: " + derr.Error()})
				p.forwardRaw(src, dst)
				return
			}
			for _, f := range frames {
				p.Observe(Frame{Direction: dir, ConnID: connID, Label: p.Label, Index: index, Payload: f, Note: classify(dir, f)})
				index++
			}
		}
		if rerr != nil {
			return
		}
	}
}

// pumpRaw copies src→dst verbatim, teeing each read chunk to the observer as one
// frame. For transports that are not NI-framed (WebSocket, HTTP over TLS).
func (p *Proxy) pumpRaw(connID int, dir Direction, src, dst net.Conn) {
	buf := make([]byte, 32*1024)
	index := 0
	defer func() { _ = src.Close(); _ = dst.Close() }()
	for {
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return
			}
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			p.Observe(Frame{Direction: dir, ConnID: connID, Label: p.Label, Index: index, Payload: chunk, Note: "raw chunk"})
			index++
		}
		if rerr != nil {
			return
		}
	}
}

func (p *Proxy) forwardRaw(src, dst net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// classify makes a best-effort, read-only guess at what a frame is, for logging.
func classify(dir Direction, payload []byte) string {
	if len(payload) == 0 {
		return "empty"
	}
	// Text eyecatchers travel in some control frames and route/handshake records.
	if a := asciiEyecatcher(payload); a != "" {
		return a
	}
	if len(payload) == gateway.NormalClientLength {
		if _, err := gateway.DecodeNormalClient(payload); err == nil {
			return "GW_NORMAL_CLIENT"
		}
	}
	if xml := xmlPreview(payload); xml != "" {
		return "xRFC XML: " + xml
	}
	return fmt.Sprintf("%d bytes, head=%x", len(payload), payload[:min(16, len(payload))])
}

func asciiEyecatcher(payload []byte) string {
	for _, tag := range []string{"NI_ROUTE", "NI_PONG", "NI_RTERR"} {
		if len(payload) >= len(tag) && string(payload[:len(tag)]) == tag {
			return tag
		}
	}
	return ""
}

// xmlPreview returns a short preview if the payload looks like it carries xRFC
// XML (a "<name>" opening somewhere near the start), else "".
func xmlPreview(payload []byte) string {
	limit := min(64, len(payload))
	s := string(payload[:limit])
	if i := strings.IndexByte(s, '<'); i >= 0 && i+1 < len(s) {
		c := s[i+1]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' {
			return strings.Map(printableOnly, s[i:])
		}
	}
	return ""
}

func printableOnly(r rune) rune {
	if r >= 0x20 && r < 0x7f {
		return r
	}
	return '.'
}
