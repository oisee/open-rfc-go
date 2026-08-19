// SPDX-License-Identifier: Apache-2.0

package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeResource struct {
	id     int
	closed bool
}

type fakeFactory struct {
	created  atomic.Int64
	closed   atomic.Int64
	failNext atomic.Bool // Validate fails once when set
}

func (f *fakeFactory) factory() Factory[*fakeResource] {
	return Factory[*fakeResource]{
		New: func(ctx context.Context) (*fakeResource, error) {
			return &fakeResource{id: int(f.created.Add(1))}, nil
		},
		Validate: func(ctx context.Context, r *fakeResource) error {
			if f.failNext.Swap(false) {
				return errors.New("stale")
			}
			return nil
		},
		Close: func(r *fakeResource) error {
			r.closed = true
			f.closed.Add(1)
			return nil
		},
	}
}

func TestAcquireReleaseReuse(t *testing.T) {
	f := &fakeFactory{}
	p, err := New(f.factory(), Config{MaxSize: 4})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	l1, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	first := l1.Value()
	l1.Release()
	l2, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if l2.Value() != first {
		t.Fatalf("expected the idle resource to be reused")
	}
	l2.Release()
	if got := f.created.Load(); got != 1 {
		t.Fatalf("created = %d, want 1 (reuse)", got)
	}
}

func TestMaxSizeBlocksUntilRelease(t *testing.T) {
	f := &fakeFactory{}
	p, _ := New(f.factory(), Config{MaxSize: 2})
	ctx := context.Background()
	l1, _ := p.Acquire(ctx)
	l2, _ := p.Acquire(ctx)

	// A third acquire must block while both slots are leased.
	blocked, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if _, err := p.Acquire(blocked); !errors.Is(err, ErrPoolExhausted) {
		t.Fatalf("expected ErrPoolExhausted, got %v", err)
	}

	// Releasing one unblocks a waiting acquire.
	done := make(chan *Lease[*fakeResource], 1)
	go func() {
		l, err := p.Acquire(ctx)
		if err != nil {
			t.Errorf("waiter acquire: %v", err)
			done <- nil
			return
		}
		done <- l
	}()
	time.Sleep(10 * time.Millisecond)
	l1.Release()
	select {
	case l := <-done:
		if l == nil {
			t.Fatal("waiter failed")
		}
		l.Release()
	case <-time.After(time.Second):
		t.Fatal("release did not unblock a waiting acquire")
	}
	l2.Release()
	if s := p.Stats(); s.MaxSize != 2 {
		t.Fatalf("stats maxSize = %d", s.MaxSize)
	}
}

func TestValidateDiscardsStale(t *testing.T) {
	f := &fakeFactory{}
	p, _ := New(f.factory(), Config{MaxSize: 2})
	ctx := context.Background()
	l, _ := p.Acquire(ctx)
	l.Release() // now idle
	f.failNext.Store(true)
	l2, err := p.Acquire(ctx) // validation fails → discard + create fresh
	if err != nil {
		t.Fatal(err)
	}
	if f.created.Load() != 2 {
		t.Fatalf("created = %d, want 2 (stale discarded)", f.created.Load())
	}
	if f.closed.Load() != 1 {
		t.Fatalf("closed = %d, want 1 (stale closed)", f.closed.Load())
	}
	l2.Release()
}

func TestDiscardFreesSlot(t *testing.T) {
	f := &fakeFactory{}
	p, _ := New(f.factory(), Config{MaxSize: 1})
	ctx := context.Background()
	l, _ := p.Acquire(ctx)
	l.Discard() // closes and frees the single slot
	if f.closed.Load() != 1 {
		t.Fatalf("closed = %d, want 1", f.closed.Load())
	}
	// The freed slot must allow a new acquire.
	l2, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire after discard: %v", err)
	}
	if f.created.Load() != 2 {
		t.Fatalf("created = %d, want 2", f.created.Load())
	}
	l2.Release()
}

func TestDoubleReleaseIsIdempotent(t *testing.T) {
	f := &fakeFactory{}
	p, _ := New(f.factory(), Config{MaxSize: 2})
	l, _ := p.Acquire(context.Background())
	l.Release()
	l.Release() // no-op
	l.Discard() // no-op
	if s := p.Stats(); s.Leased != 0 || s.Idle != 1 {
		t.Fatalf("stats after double release: %+v", s)
	}
}

func TestIdleEviction(t *testing.T) {
	f := &fakeFactory{}
	p, _ := New(f.factory(), Config{MaxSize: 2, MaxIdleTime: 40 * time.Millisecond})
	defer p.Close(context.Background())
	l, _ := p.Acquire(context.Background())
	l.Release()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if f.closed.Load() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if f.closed.Load() != 1 {
		t.Fatalf("idle resource was not evicted (closed=%d)", f.closed.Load())
	}
	if s := p.Stats(); s.Idle != 0 {
		t.Fatalf("evicted resource still counted idle: %+v", s)
	}
}

func TestCloseClosesIdle(t *testing.T) {
	f := &fakeFactory{}
	p, _ := New(f.factory(), Config{MaxSize: 3})
	ctx := context.Background()
	var leases []*Lease[*fakeResource]
	for i := 0; i < 3; i++ {
		l, _ := p.Acquire(ctx)
		leases = append(leases, l)
	}
	for _, l := range leases {
		l.Release()
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if f.closed.Load() != 3 {
		t.Fatalf("closed = %d, want 3", f.closed.Load())
	}
	if _, err := p.Acquire(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("acquire on closed pool = %v, want ErrClosed", err)
	}
}

func TestFactoryNewError(t *testing.T) {
	boom := errors.New("boom")
	p, _ := New(Factory[*fakeResource]{
		New: func(ctx context.Context) (*fakeResource, error) { return nil, boom },
	}, Config{MaxSize: 1})
	if _, err := p.Acquire(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("acquire = %v, want boom", err)
	}
	// The slot must be returned on New failure, so a later acquire can retry.
	if _, err := p.Acquire(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("second acquire = %v, want boom (slot returned)", err)
	}
}

func TestConcurrentAcquireRelease(t *testing.T) {
	f := &fakeFactory{}
	p, _ := New(f.factory(), Config{MaxSize: 8})
	defer p.Close(context.Background())
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				l, err := p.Acquire(ctx)
				if err != nil {
					t.Errorf("acquire: %v", err)
					return
				}
				l.Release()
			}
		}()
	}
	wg.Wait()
	s := p.Stats()
	if s.Leased != 0 {
		t.Fatalf("leaked leases: %+v", s)
	}
	if s.Open > 8 {
		t.Fatalf("exceeded MaxSize: %+v", s)
	}
}
