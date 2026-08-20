// SPDX-License-Identifier: Apache-2.0

package lifecycle

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oisee/open-rfc-go/internal/client"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/pool"
)

type fakeSession struct {
	active    atomic.Int32
	maxActive atomic.Int32
	pings     atomic.Int64
	calls     atomic.Int64
	pingErr   error
	closed    atomic.Bool
}

func (f *fakeSession) Call(ctx context.Context, fn string, imports []cpic.NamedValue, outputs []string) (client.CallResult, error) {
	n := f.active.Add(1)
	defer f.active.Add(-1)
	for {
		m := f.maxActive.Load()
		if n <= m || f.maxActive.CompareAndSwap(m, n) {
			break
		}
	}
	if fn == PingFunction {
		f.pings.Add(1)
		return client.CallResult{Success: true}, f.pingErr
	}
	f.calls.Add(1)
	time.Sleep(time.Millisecond) // widen any overlap window
	return client.CallResult{Success: true}, nil
}

func (f *fakeSession) CallRaw(ctx context.Context, request []byte) (client.CallResult, error) {
	f.calls.Add(1)
	return client.CallResult{Success: true}, nil
}

func (f *fakeSession) CallWithCallbacks(ctx context.Context, request []byte, _ client.CallbackHandler) (client.CallResult, error) {
	f.calls.Add(1)
	return client.CallResult{Success: true}, nil
}

func (f *fakeSession) Authenticated() bool { return !f.closed.Load() }
func (f *fakeSession) Close() error        { f.closed.Store(true); return nil }

func TestSerializesConcurrentCalls(t *testing.T) {
	f := &fakeSession{}
	m := Wrap(f)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if _, err := m.Call(context.Background(), "STFC_CONNECTION", nil, nil); err != nil {
					t.Errorf("call: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if got := f.maxActive.Load(); got != 1 {
		t.Fatalf("max concurrent calls = %d, want 1 (must be serialized)", got)
	}
	if got := f.calls.Load(); got != 500 {
		t.Fatalf("calls = %d, want 500", got)
	}
}

func TestPingUsesRFCPing(t *testing.T) {
	f := &fakeSession{}
	m := Wrap(f)
	if err := m.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if f.pings.Load() != 1 {
		t.Fatalf("pings = %d, want 1", f.pings.Load())
	}
	f.pingErr = errors.New("dead")
	if err := m.Ping(context.Background()); err == nil {
		t.Fatalf("expected ping error to propagate")
	}
}

func TestCloseIdempotentAndBlocksUse(t *testing.T) {
	f := &fakeSession{}
	m := Wrap(f)
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("second close should be nil, got %v", err)
	}
	if _, err := m.Call(context.Background(), "STFC_CONNECTION", nil, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("call after close = %v, want ErrClosed", err)
	}
	if err := m.Ping(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("ping after close = %v, want ErrClosed", err)
	}
	if m.Authenticated() {
		t.Fatalf("closed session must not report authenticated")
	}
}

func TestNewPoolValidatesWithPing(t *testing.T) {
	var created atomic.Int64
	sessions := make(chan *fakeSession, 8)
	p, err := NewPool(PoolOptions{
		Open: func(ctx context.Context) (*Managed, error) {
			created.Add(1)
			f := &fakeSession{}
			sessions <- f
			return Wrap(f), nil
		},
		Pool: pool.Config{MaxSize: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close(context.Background())
	ctx := context.Background()

	l1, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	first := <-sessions
	l1.Release()              // returns to idle
	l2, err := p.Acquire(ctx) // reuse → Validate runs a ping
	if err != nil {
		t.Fatal(err)
	}
	if first.pings.Load() != 1 {
		t.Fatalf("reused session should have been pinged once, got %d", first.pings.Load())
	}
	_ = l2
	if created.Load() != 1 {
		t.Fatalf("created = %d, want 1 (reuse)", created.Load())
	}
	l2.Discard() // closes it
	if !first.closed.Load() {
		t.Fatalf("discarded session should be closed")
	}
}

func TestNewPoolRequiresOpen(t *testing.T) {
	if _, err := NewPool(PoolOptions{}); err == nil {
		t.Fatalf("expected error when Open is nil")
	}
}
