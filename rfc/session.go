// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"context"
	"errors"
	"sync"

	"github.com/oisee/open-rfc-go/internal/lifecycle"
	"github.com/oisee/open-rfc-go/internal/pool"
)

// Session is a client pinned to one connection, so consecutive calls run in the
// same ABAP work process and share its roll area. Most calls should go through
// Client.Call, which takes any pooled connection; a Session is for the stateful
// protocols where server-side state must survive between calls — an attached
// ABAP debugger, a blocking listener, an enqueue held across calls, or anything
// that would otherwise fail with a session mismatch.
//
// A Session holds its connection for its whole lifetime, so it consumes one slot
// of the pool: keep it for as long as the stateful conversation lasts and Close
// it as soon as it ends. A Session is safe for sequential use from one goroutine
// at a time; a blocking call (a debugger listener, say) occupies it until it
// returns, so issue unrelated calls through the Client instead.
type Session struct {
	client *Client

	mu    sync.Mutex
	lease *pool.Lease[*lifecycle.Managed]
}

// Pin takes one connection out of the pool and binds it to a Session.
// The caller must Close the Session to return the connection.
func (c *Client) Pin(ctx context.Context) (*Session, error) {
	lease, err := c.pool.Acquire(ctx)
	if err != nil {
		return nil, translate(err)
	}
	return &Session{client: c, lease: lease}, nil
}

// Call invokes a function module on this session's pinned connection.
func (s *Session) Call(ctx context.Context, functionName string, in Params) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lease == nil {
		return Result{}, ErrClosed
	}
	res, err := s.client.callOn(ctx, s.lease.Value(), functionName, in)
	if err != nil && errors.Is(err, ErrTransport) {
		// The conversation is broken: drop the connection rather than return a
		// half-dead session to the pool, and fail subsequent calls fast.
		s.lease.Discard()
		s.lease.Release()
		s.lease = nil
	}
	return res, err
}

// DescribeTool renders a function module's interface as an MCP-tool JSON Schema,
// resolved over this session's connection.
func (s *Session) DescribeTool(ctx context.Context, functionName string) (ToolSchema, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lease == nil {
		return ToolSchema{}, ErrClosed
	}
	return s.client.DescribeTool(ctx, functionName)
}

// Ping keeps an idle pinned session alive (RFC_PING), so a gateway or work-process
// idle timeout does not drop a conversation that is being held for later calls.
//
// A conversation carries one call at a time: while a blocking call is in flight
// (a debugger listener, say) the session is occupied and must not be pinged —
// that call keeps the connection busy on its own. Ping the session only between
// calls; to check a system while a session blocks, call through the Client, which
// uses another pooled connection.
func (s *Session) Ping(ctx context.Context) error {
	_, err := s.Call(ctx, "RFC_PING", nil)
	return err
}

// Close returns the pinned connection to the pool. It is idempotent.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lease == nil {
		return nil
	}
	s.lease.Release()
	s.lease = nil
	return nil
}
