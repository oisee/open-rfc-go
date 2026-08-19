// SPDX-License-Identifier: Apache-2.0
//
// A Go-idiomatic connection pool. This is the milestone-6 redesign of
// open-rfc's src/pool/connection-pool-runtime.ts, not a mechanical port: the
// upstream runtime is ~2100 lines of Promise scheduling, an injectable async
// scheduler, and monitor plumbing that Go gets for free from goroutines,
// channels, and context.Context. The essential semantics are preserved — a
// bounded set of resources, health validation on acquire, idle eviction, and
// graceful shutdown — expressed with a FIFO waiter queue that hands a freed
// resource or slot directly to the next waiter, so there are no lost wakeups.
// See docs/provenance.md.

// Package pool provides a generic, context-aware resource pool. It is used to
// pool authenticated RFC sessions, but carries no RFC knowledge itself.
package pool

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrClosed reports an operation on a pool that has been shut down.
	ErrClosed = errors.New("pool: closed")
	// ErrPoolExhausted reports that no resource became available before the
	// acquire deadline. It wraps the context error that ended the wait.
	ErrPoolExhausted = errors.New("pool: no resource available before deadline")
)

// Factory creates, validates, and closes pooled resources. New is required;
// Validate and Close may be nil.
type Factory[T any] struct {
	// New creates one fresh resource, honouring ctx for cancellation.
	New func(ctx context.Context) (T, error)
	// Validate checks an idle resource before it is handed out. A non-nil
	// error causes the resource to be discarded and another acquired. Nil
	// skips validation.
	Validate func(ctx context.Context, resource T) error
	// Close releases a resource. Nil means the resource needs no teardown.
	Close func(resource T) error
}

// Config bounds a pool.
type Config struct {
	// MaxSize is the maximum number of live resources (idle plus leased).
	// Zero selects DefaultMaxSize.
	MaxSize int
	// MaxIdleTime evicts a resource left idle for longer than this. Zero
	// disables idle eviction.
	MaxIdleTime time.Duration
	// AcquireTimeout bounds one Acquire call in addition to its context.
	// Zero relies solely on the caller's context.
	AcquireTimeout time.Duration
}

// DefaultMaxSize is used when Config.MaxSize is zero.
const DefaultMaxSize = 8

// Stats is a point-in-time snapshot of pool occupancy.
type Stats struct {
	MaxSize int
	Idle    int
	Leased  int
	Open    int // Idle + Leased
}

type entry[T any] struct {
	resource T
	lastUsed time.Time
}

// grant is handed from a releaser/evictor to a waiting Acquire. When
// hasResource is true the resource is ready to use (subject to validation);
// otherwise it is a permit to create a fresh resource in a reserved slot.
type grant[T any] struct {
	resource    T
	hasResource bool
}

// Pool is a bounded, context-aware pool of resources of type T. A Pool is safe
// for concurrent use.
type Pool[T any] struct {
	factory Factory[T]
	cfg     Config

	mu        sync.Mutex
	idle      []entry[T]
	leased    int
	freeSlots int
	waiters   []chan grant[T]
	closed    bool
	closedCh  chan struct{}

	janitorStop chan struct{}
	janitorDone chan struct{}
}

// New builds a pool from a factory and config. It returns an error if the
// factory has no New function.
func New[T any](factory Factory[T], cfg Config) (*Pool[T], error) {
	if factory.New == nil {
		return nil, errors.New("pool: factory.New must not be nil")
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = DefaultMaxSize
	}
	p := &Pool[T]{
		factory:   factory,
		cfg:       cfg,
		freeSlots: cfg.MaxSize,
		closedCh:  make(chan struct{}),
	}
	if cfg.MaxIdleTime > 0 {
		p.janitorStop = make(chan struct{})
		p.janitorDone = make(chan struct{})
		go p.janitor()
	}
	return p, nil
}

// Lease is a checked-out resource. Exactly one of Release or Discard must be
// called; both are idempotent.
type Lease[T any] struct {
	pool     *Pool[T]
	resource T
	once     sync.Once
}

// Value returns the leased resource.
func (l *Lease[T]) Value() T { return l.resource }

// Release returns a healthy resource to the pool for reuse.
func (l *Lease[T]) Release() { l.once.Do(func() { l.pool.putBack(l.resource) }) }

// Discard closes a resource and drops it from the pool. Use it when the
// resource is known to be broken.
func (l *Lease[T]) Discard() { l.once.Do(func() { l.pool.discard(l.resource) }) }

// grantAvailable hands idle resources and free slots to queued waiters until
// one runs out. The caller must hold p.mu.
func (p *Pool[T]) grantAvailable() {
	for len(p.waiters) > 0 {
		switch {
		case len(p.idle) > 0:
			n := len(p.idle) - 1
			e := p.idle[n]
			p.idle[n] = entry[T]{}
			p.idle = p.idle[:n]
			p.leased++
			p.popWaiter() <- grant[T]{resource: e.resource, hasResource: true}
		case p.freeSlots > 0:
			p.freeSlots--
			p.leased++
			p.popWaiter() <- grant[T]{}
		default:
			return
		}
	}
}

func (p *Pool[T]) popWaiter() chan grant[T] {
	w := p.waiters[0]
	p.waiters[0] = nil
	p.waiters = p.waiters[1:]
	return w
}

func (p *Pool[T]) removeWaiter(target chan grant[T]) bool {
	for i, w := range p.waiters {
		if w == target {
			p.waiters = append(p.waiters[:i], p.waiters[i+1:]...)
			return true
		}
	}
	return false
}

// Acquire checks out a resource, creating one if the pool is below MaxSize and
// no idle resource is available, or blocking until one frees otherwise. It
// honours ctx and Config.AcquireTimeout.
func (p *Pool[T]) Acquire(ctx context.Context) (*Lease[T], error) {
	if p.cfg.AcquireTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.cfg.AcquireTimeout)
		defer cancel()
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, ErrClosed
		}
		if n := len(p.idle); n > 0 {
			e := p.idle[n-1]
			p.idle[n-1] = entry[T]{}
			p.idle = p.idle[:n-1]
			p.leased++
			p.mu.Unlock()
			lease, retry := p.useIdle(ctx, e.resource)
			if retry {
				continue
			}
			return lease, nil
		}
		if p.freeSlots > 0 {
			p.freeSlots--
			p.leased++
			p.mu.Unlock()
			return p.createLeased(ctx)
		}
		// No idle resource and no free slot: queue and wait for a handoff.
		w := make(chan grant[T], 1)
		p.waiters = append(p.waiters, w)
		p.mu.Unlock()

		select {
		case g := <-w:
			if !g.hasResource {
				return p.createLeased(ctx)
			}
			lease, retry := p.useIdle(ctx, g.resource)
			if retry {
				continue
			}
			return lease, nil
		case <-ctx.Done():
			return p.abandon(w, errors.Join(ErrPoolExhausted, ctx.Err()))
		case <-p.closedCh:
			return p.abandon(w, ErrClosed)
		}
	}
}

// useIdle validates a leased-from-idle resource. It returns (lease, false) when
// healthy, or (nil, true) when the resource was discarded and Acquire should
// retry. The caller has already counted the resource as leased.
func (p *Pool[T]) useIdle(ctx context.Context, resource T) (*Lease[T], bool) {
	if p.factory.Validate == nil || p.factory.Validate(ctx, resource) == nil {
		return &Lease[T]{pool: p, resource: resource}, false
	}
	p.closeResource(resource)
	p.mu.Lock()
	p.leased--
	p.freeSlots++
	p.grantAvailable()
	p.mu.Unlock()
	return nil, true
}

// createLeased builds a fresh resource for an already-reserved slot (leased and
// freeSlots already accounted). On failure it returns the slot.
func (p *Pool[T]) createLeased(ctx context.Context) (*Lease[T], error) {
	resource, err := p.factory.New(ctx)
	if err != nil {
		p.mu.Lock()
		p.leased--
		p.freeSlots++
		p.grantAvailable()
		p.mu.Unlock()
		return nil, err
	}
	return &Lease[T]{pool: p, resource: resource}, nil
}

// abandon handles a waiter whose context or the pool ended. If a grant was
// delivered concurrently it is returned to the pool so no resource leaks.
func (p *Pool[T]) abandon(w chan grant[T], cause error) (*Lease[T], error) {
	p.mu.Lock()
	if p.removeWaiter(w) {
		p.mu.Unlock()
		return nil, cause
	}
	p.mu.Unlock()
	g := <-w // a grant was in flight
	if g.hasResource {
		p.putBack(g.resource)
	} else {
		p.mu.Lock()
		p.leased--
		p.freeSlots++
		p.grantAvailable()
		p.mu.Unlock()
	}
	return nil, cause
}

// putBack returns a healthy resource: handed to the next waiter, or parked idle.
func (p *Pool[T]) putBack(resource T) {
	p.mu.Lock()
	if p.closed {
		p.leased--
		p.mu.Unlock()
		p.closeResource(resource)
		return
	}
	if len(p.waiters) > 0 {
		p.popWaiter() <- grant[T]{resource: resource, hasResource: true}
		p.mu.Unlock() // leased ownership transfers to the waiter
		return
	}
	p.leased--
	p.idle = append(p.idle, entry[T]{resource: resource, lastUsed: time.Now()})
	p.mu.Unlock()
}

func (p *Pool[T]) discard(resource T) {
	p.mu.Lock()
	p.leased--
	p.freeSlots++
	p.grantAvailable()
	p.mu.Unlock()
	p.closeResource(resource)
}

func (p *Pool[T]) closeResource(resource T) {
	if p.factory.Close != nil {
		_ = p.factory.Close(resource)
	}
}

func (p *Pool[T]) janitor() {
	defer close(p.janitorDone)
	interval := p.cfg.MaxIdleTime
	if interval > time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-p.janitorStop:
			return
		case <-ticker.C:
			p.evictExpired()
		}
	}
}

func (p *Pool[T]) evictExpired() {
	cutoff := time.Now().Add(-p.cfg.MaxIdleTime)
	var expired []T
	p.mu.Lock()
	kept := p.idle[:0]
	for _, e := range p.idle {
		if e.lastUsed.Before(cutoff) {
			expired = append(expired, e.resource)
			p.freeSlots++
		} else {
			kept = append(kept, e)
		}
	}
	for i := len(kept); i < len(p.idle); i++ {
		p.idle[i] = entry[T]{}
	}
	p.idle = kept
	if len(expired) > 0 {
		p.grantAvailable()
	}
	p.mu.Unlock()
	for _, resource := range expired {
		p.closeResource(resource)
	}
}

// Stats returns a snapshot of pool occupancy.
func (p *Pool[T]) Stats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()
	return Stats{
		MaxSize: p.cfg.MaxSize,
		Idle:    len(p.idle),
		Leased:  p.leased,
		Open:    len(p.idle) + p.leased,
	}
}

// Close shuts the pool down and closes every idle resource. Leased resources
// are closed when they are later released; queued waiters wake with ErrClosed.
// It honours ctx while waiting for the janitor to stop.
func (p *Pool[T]) Close(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	p.closed = true
	close(p.closedCh)
	idle := p.idle
	p.idle = nil
	p.mu.Unlock()

	if p.janitorStop != nil {
		close(p.janitorStop)
		select {
		case <-p.janitorDone:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	for _, e := range idle {
		p.closeResource(e.resource)
	}
	return nil
}
