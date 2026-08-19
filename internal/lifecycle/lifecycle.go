// SPDX-License-Identifier: Apache-2.0
//
// A Go-idiomatic session lifecycle layer. This is the milestone-6 redesign of
// open-rfc's src/lifecycle/session-context-runtime.ts: the upstream keeps a
// promise queue to serialize calls on one stateful connection and an async
// keep-alive/reset state machine. Go expresses the same guarantees with a mutex
// (one call in flight per session) and context.Context, and wires a session
// into the connection pool as a health-checked, recyclable resource.
// Transaction units (tRFC/qRFC) and explicit server-context reset remain as
// separate wire work. See docs/provenance.md.

// Package lifecycle serializes calls on one RFC session and pools authenticated
// sessions with health validation.
package lifecycle

import (
	"context"
	"errors"
	"sync"

	"github.com/oisee/open-rfc-go/internal/client"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/pool"
)

// ErrClosed reports use of a session that has been closed.
var ErrClosed = errors.New("lifecycle: session is closed")

// PingFunction is the function module used as a liveness probe.
const PingFunction = "RFC_PING"

// Session is the RFC session the lifecycle layer manages. *client.Session
// satisfies it.
type Session interface {
	Call(ctx context.Context, functionName string, imports []cpic.NamedValue, requestedOutputs []string) (client.CallResult, error)
	Authenticated() bool
	Close() error
}

// Managed serializes access to one RFC session. A classic RFC connection
// carries exactly one call at a time; Managed enforces that with a mutex, so a
// pooled session can be handed between goroutines safely. All methods block
// until any in-flight call on the same session completes.
type Managed struct {
	mu     sync.Mutex
	sess   Session
	closed bool
}

// Wrap adds serialization around a session.
func Wrap(sess Session) *Managed { return &Managed{sess: sess} }

// Call runs one function module, serialized against every other operation on
// this session.
func (m *Managed) Call(ctx context.Context, functionName string, imports []cpic.NamedValue, requestedOutputs []string) (client.CallResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return client.CallResult{}, ErrClosed
	}
	return m.sess.Call(ctx, functionName, imports, requestedOutputs)
}

// Ping probes the session with RFC_PING. A nil return means the session is
// alive; it is the natural pool health check.
func (m *Managed) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	_, err := m.sess.Call(ctx, PingFunction, nil, nil)
	return err
}

// Authenticated reports whether the underlying session has logged on.
func (m *Managed) Authenticated() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return false
	}
	return m.sess.Authenticated()
}

// Close closes the session. It is idempotent.
func (m *Managed) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	return m.sess.Close()
}

// PoolOptions configures a pool of managed sessions.
type PoolOptions struct {
	// Open opens, authenticates, and wraps one session. It is the pool's
	// factory; callers supply their own dial + logon.
	Open func(ctx context.Context) (*Managed, error)
	// Pool bounds and tunes the underlying resource pool.
	Pool pool.Config
	// SkipHealthCheck disables the RFC_PING validation on acquire.
	SkipHealthCheck bool
}

// NewPool builds a pool of authenticated, health-checked sessions. Acquire
// returns a lease whose Value is a *Managed; Release returns it for reuse and
// Discard closes a broken one.
func NewPool(opts PoolOptions) (*pool.Pool[*Managed], error) {
	if opts.Open == nil {
		return nil, errors.New("lifecycle: PoolOptions.Open must not be nil")
	}
	factory := pool.Factory[*Managed]{
		New: opts.Open,
		Close: func(m *Managed) error {
			return m.Close()
		},
	}
	if !opts.SkipHealthCheck {
		factory.Validate = func(ctx context.Context, m *Managed) error {
			return m.Ping(ctx)
		}
	}
	return pool.New(factory, opts.Pool)
}
