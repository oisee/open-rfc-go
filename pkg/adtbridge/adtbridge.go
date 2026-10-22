// SPDX-License-Identifier: Apache-2.0
// Shutdown adapted from open-steamgate tools/protocols/server.mjs
// at 2fb365933d89b903d3f8f42286778894cbd47d73 (MIT there), ported by its
// copyright holder under Apache-2.0; Go also joins handlers and read pumps.
// Operational limits, session lifecycle, and demo identity adapted from open-steamgate
// tools/protocols/rfc-server.mjs at 7d8e0617a8e5ded4ead65da534acf1f1b2021863 (MIT there),
// ported by its copyright holder under Apache-2.0. See docs/provenance.md.
// Static auth configuration follows open-steamgate tools/protocols/rfc-server.mjs
// at 13a3c5292eddb240d0c65b15a37a8788661410a4 (MIT there), ported by its copyright holder, who contributes it here under Apache-2.0.
// Modified for Go Options and synchronous Check/Serve validation.
// See docs/provenance.md.

// Package adtbridge is the RFC-to-HTTP bridge as a library: what
// cmd/adt-rfc-bridge is, for a program that wants to mount it beside other
// things — a launcher that brings up the workbench, this bridge and the DIAG
// stub as one command. It is the public face of internal/rfcserver, which
// stays internal: a caller gets a Backend, an Options and Serve, and nothing
// of the record layer leaks.
//
// One connection owns one Eclipse session: its own cookie jar against the
// backend, its own dispatcher with the four function modules an ADT client
// asks for (the REST endpoint and the three that describe it), and the
// conscious loop that answers frames until the client goes. Known lock handles
// may borrow that session from another connection of the same RFC identity.
package adtbridge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

// Backend is the HTTP origin the ADT requests are made against, with its
// credentials and trust; see LoadBackend for the file that describes one.
type Backend = rfcserver.Backend

// LoadBackend reads a backend description from a JSON file; an empty path
// gives an empty backend for the caller to fill in.
func LoadBackend(path string) (Backend, error) { return rfcserver.LoadBackend(path) }

// Options is everything a bridge needs besides a listener.
type Options struct {
	Backend Backend
	// AuthMode is "demo" (the default) or "static". Static credentials are
	// independent of the HTTP backend identity.
	AuthMode    string
	RFCUser     string
	RFCPassword string
	// RFCClient optionally restricts static logons to this client.
	RFCClient string
	// MaxLogonBytes bounds logon decoding; zero means 64 KiB.
	MaxLogonBytes int
	// Timeout bounds one backend request and socket inactivity, matching JS
	// timeoutMs. Zero means 120 seconds.
	Timeout time.Duration
	// MaxConnections caps live sockets in Serve; zero means 64. Excess sockets
	// are closed immediately. ServeConn serves one independent session.
	MaxConnections int
	// ShutdownGrace bounds draining after cancellation or listener failure.
	// Zero means 5 seconds. Backend work retains Timeout; logoff is capped at 5 s.
	ShutdownGrace time.Duration
	// Verbose reports every frame decision through Log.
	Verbose bool
	// Log receives the bridge's lines; nil is quiet.
	Log func(format string, args ...any)
	// Dump, when set, receives every frame in both directions — the recorder
	// cmd/adt-rfc-bridge switches on with STG_DUMP. nil records nothing.
	Dump func(peer, dir string, frame []byte)
}

func (o Options) logf(format string, args ...any) {
	if o.Log != nil {
		o.Log(format, args...)
	}
}

// DefaultTimeout matches OSG timeoutMs for HTTP requests and RFC inactivity.
const DefaultTimeout = 120 * time.Second

// DefaultShutdownGrace matches OSG closeProtocols graceMs.
const DefaultShutdownGrace = 5 * time.Second

// DefaultMaxConnections matches OSG server.maxConnections.
const DefaultMaxConnections = 64

// ErrLimitsConfiguration reports a negative limit (zero selects JS defaults).
var ErrLimitsConfiguration = errors.New("adtbridge: invalid limits configuration")

func (o Options) timeout() time.Duration {
	if o.Timeout == 0 {
		return DefaultTimeout
	}
	return o.Timeout
}

func (o Options) shutdownGrace() time.Duration {
	if o.ShutdownGrace == 0 {
		return DefaultShutdownGrace
	}
	return o.ShutdownGrace
}

// ErrAuthConfiguration reports an incomplete or unsupported auth setup.
var ErrAuthConfiguration = errors.New("adtbridge: invalid authentication configuration")

// Check validates backend and authentication setup synchronously. Call it
// before opening listeners or starting Serve in a goroutine so startup failures
// reach the launcher directly. Serve also checks before its first Accept.
func Check(o Options) error {
	if o.Timeout < 0 || o.MaxConnections < 0 || o.ShutdownGrace < 0 {
		return fmt.Errorf("%w: Timeout, MaxConnections and ShutdownGrace must be nonnegative", ErrLimitsConfiguration)
	}
	if o.MaxLogonBytes < 0 {
		return fmt.Errorf("%w: MaxLogonBytes must be nonnegative", ErrAuthConfiguration)
	}
	if o.AuthMode != "" && o.AuthMode != "demo" && o.AuthMode != "static" {
		return fmt.Errorf("%w: RFC auth mode must be demo or static", ErrAuthConfiguration)
	}
	if o.AuthMode == "static" && (o.RFCUser == "" || o.RFCPassword == "") {
		return fmt.Errorf("%w: static RFC auth requires STG_RFC_USER and STG_RFC_PASSWORD", ErrAuthConfiguration)
	}
	if o.Backend.Password != "" && o.Backend.User == "" {
		return fmt.Errorf("%w: backend password requires STG_ADT_USER", ErrAuthConfiguration)
	}
	if o.Backend.URL == "" {
		return errors.New("adtbridge: no backend URL")
	}
	_, err := rfcserver.ADTRestHandler(o.Backend, nil)
	return err
}

// Serve accepts connections on ln until the context ends or the listener
// fails, and serves each on its own goroutine. Closing the listener is how
// the context stops it, so a caller that owns ln should not also close it
// while Serve runs. Shutdown drains for ShutdownGrace, closes surviving sockets,
// and waits for handlers, read pumps, and bounded backend cleanup.
func Serve(ctx context.Context, ln net.Listener, o Options) error {
	if err := Check(o); err != nil {
		return err
	}
	stop := make(chan struct{})
	watchDone := make(chan struct{})
	defer func() { close(stop); <-watchDone }()
	go func() {
		defer close(watchDone)
		select {
		case <-ctx.Done():
			ln.Close()
		case <-stop:
		}
	}()
	limit := o.MaxConnections
	if limit == 0 {
		limit = DefaultMaxConnections
	}
	slots := make(chan struct{}, limit)
	sessions := rfcserver.NewADTSessions()
	var workers sync.WaitGroup
	var mu sync.Mutex
	active := make(map[net.Conn]struct{})
	defer func() {
		ln.Close()
		done := make(chan struct{})
		go func() { workers.Wait(); close(done) }()
		timer := time.NewTimer(o.shutdownGrace())
		defer timer.Stop()
		select {
		case <-done:
			return
		case <-timer.C:
		}
		mu.Lock()
		survivors := make([]net.Conn, 0, len(active))
		for conn := range active {
			survivors = append(survivors, conn)
		}
		mu.Unlock()
		for _, conn := range survivors {
			conn.Close()
		}
		<-done
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if ctx.Err() != nil {
			conn.Close()
			return nil
		}
		select {
		case slots <- struct{}{}:
			counted := &closeConn{Conn: conn, onClose: func() { <-slots }}
			mu.Lock()
			active[counted] = struct{}{}
			mu.Unlock()
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer func() { mu.Lock(); delete(active, counted); mu.Unlock() }()
				serveConn(counted, o, sessions)
			}()
		default:
			conn.Close()
		}
	}
}

// ServeConn is one client from connect to gone.
func ServeConn(conn net.Conn, o Options) {
	serveConn(conn, o, rfcserver.NewADTSessions())
}

func serveConn(conn net.Conn, o Options, sessions *rfcserver.ADTSessions) {
	defer conn.Close()
	if err := Check(o); err != nil {
		o.logf("%v", err)
		return
	}
	peer := conn.RemoteAddr().String()
	o.logf("%s connected", peer)
	jar, err := cookiejar.New(nil)
	if err != nil {
		o.logf("%s: %v", peer, err)
		return
	}
	timeout := o.timeout()
	client := &http.Client{Timeout: timeout, Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if tr, terr := o.Backend.Transport(); terr != nil {
		o.logf("%s: %v", peer, terr)
		return
	} else if tr != nil {
		client.Transport = tr
	}
	session := rfcserver.NewADTSession(client)
	backend := o.Backend
	defer func() { sessions.Disconnect(session); session.End(backend, timeout, o.logf) }()
	idle := newIdleConn(conn, timeout, func() { sessions.Disconnect(session) })
	defer func() { idle.Close(); <-idle.done }()
	conn = idle
	dispatcher := rfcserver.NewDispatcher()
	dispatcher.Handle("RFC_GET_FUNCTION_INTERFACE", rfcserver.FunctionInterfaceHandler())
	dispatcher.Handle("DDIF_FIELDINFO_GET", rfcserver.FieldInfoHandler())
	dispatcher.Handle("RFC_GET_STRUCTURE_DEFINITION", rfcserver.StructureDefinitionHandler())
	dispatcher.Identity = o.Backend.LogonIdentity()
	dispatcher.MaxLogonBytes = o.MaxLogonBytes
	loggedOn := false
	dispatcher.AuthenticateLogon = func(payload []byte) (rfcserver.LogonIdentity, error) {
		if loggedOn {
			return rfcserver.LogonIdentity{}, rfcserver.ErrAuthentication
		}
		var identity rfcserver.LogonIdentity
		var err error
		if o.AuthMode == "static" {
			identity, err = rfcserver.AuthenticateStaticLogon(payload, o.RFCUser, o.RFCPassword, o.RFCClient, dispatcher.Identity, o.MaxLogonBytes)
		} else {
			identity, err = rfcserver.DecodeDemoLogon(payload, dispatcher.Identity, o.MaxLogonBytes)
		}
		if err != nil {
			o.logf("%v", err)
			return identity, err
		}
		if o.AuthMode != "static" && backend.User == "" {
			backend.User = identity.User
		}
		handler, err := rfcserver.ADTSessionHandler(backend, session, sessions, identity)
		if err != nil {
			return identity, err
		}
		dispatcher.Handle("SADT_REST_RFC_ENDPOINT", handler)
		loggedOn = true
		return identity, nil
	}
	logf := func(string) {}
	if o.Verbose {
		logf = func(s string) { o.logf("%s: %s", peer, s) }
	}
	var dump func(dir string, frame []byte)
	if o.Dump != nil {
		dump = func(dir string, frame []byte) { o.Dump(peer, dir, frame) }
	}
	rfcserver.ServeConscious(conn, dispatcher, logf, dump)
	o.logf("%s gone", peer)
}

// User names who the backend is reached as, for a banner.
func User(b Backend) string {
	if b.User == "" {
		return "an anonymous client"
	}
	return b.User
}

// closeConn releases a listener slot when the socket closes, independently of
// the best-effort HTTP cleanup which can outlive it.
type closeConn struct {
	net.Conn
	once    sync.Once
	onClose func()
}

func (c *closeConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.onClose)
	return err
}

// Node's socket timer measures inactivity even while HTTP work is in flight.
// Read deadlines alone would only start counting once the handler returned.
type idleConn struct {
	net.Conn
	mu        sync.Mutex
	timer     *time.Timer
	incoming  *io.PipeReader
	writer    *io.PipeWriter
	timeout   time.Duration
	done      chan struct{}
	closeDone chan struct{}
	closed    bool
	onClose   func()
}

func newIdleConn(conn net.Conn, timeout time.Duration, onClose func()) *idleConn {
	c := &idleConn{Conn: conn, timeout: timeout, onClose: onClose, done: make(chan struct{}), closeDone: make(chan struct{})}
	c.incoming, c.writer = io.Pipe()
	c.mu.Lock()
	c.timer = time.AfterFunc(timeout, func() { c.Close() })
	c.mu.Unlock()
	// Reading independently observes disconnects and resets the idle timer while
	// an HTTP handler is busy. The pipe applies backpressure with at most one
	// 32 KiB chunk retained here, rather than queuing unbounded network bytes.
	go func() {
		defer close(c.done)
		defer c.Close()
		chunk := make([]byte, 32*1024)
		for {
			n, err := c.Conn.Read(chunk)
			if n > 0 {
				c.touch()
				if _, werr := c.writer.Write(chunk[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				c.writer.CloseWithError(err)
				return
			}
		}
	}()
	return c
}
func (c *idleConn) touch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.timer.Reset(c.timeout)
	}
}
func (c *idleConn) Read(b []byte) (int, error) {
	return c.incoming.Read(b)
}
func (c *idleConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		c.touch()
	}
	if err != nil {
		c.Close()
	}
	return n, err
}
func (c *idleConn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		<-c.closeDone
		return nil
	}
	c.closed = true
	c.timer.Stop()
	c.mu.Unlock()
	c.onClose()
	err := c.Conn.Close()
	c.incoming.Close()
	c.writer.Close()
	close(c.closeDone)
	return err
}
