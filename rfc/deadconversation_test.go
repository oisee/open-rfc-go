// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/client"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/lifecycle"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/pool"
	"github.com/oisee/open-rfc-go/internal/rfcerr"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/transport"
)

// fakeConvSession scripts one conversation: what a function call and a raw
// (metadata) call return, and whether the reply arrives with a server-side
// normal deallocation — the APPC RC-18 shape a MESSAGE, short dump, or auth
// refusal produces, where the reply itself decodes but the conversation is
// over. A call on an already-closed conversation fails the way the real CPIC
// session does.
type fakeConvSession struct {
	callResult     client.CallResult
	callErr        error
	closeOnCall    bool
	callRawResult  client.CallResult
	callRawErr     error
	closeOnCallRaw bool
	callRawCount   int
	closed         bool
}

func successResult() client.CallResult {
	return client.CallResult{Success: true, Envelope: rfcerr.Envelope{Outcome: rfcerr.OutcomeSuccess}}
}

func abapMessageResult() client.CallResult {
	return client.CallResult{Envelope: rfcerr.Envelope{
		Outcome: rfcerr.OutcomeAbapMessage,
		Facts:   rfcerr.RemoteErrorFacts{T100Text: "You are not authorized to use function module"},
	}}
}

func (f *fakeConvSession) Call(ctx context.Context, fn string, imports []cpic.NamedValue, outputs []string) (client.CallResult, error) {
	if f.closed {
		return client.CallResult{}, errors.New("cpic session: session is closed")
	}
	return successResult(), nil
}

func (f *fakeConvSession) CallRaw(ctx context.Context, request []byte) (client.CallResult, error) {
	f.callRawCount++
	if f.closed {
		return client.CallResult{}, errors.New("cpic session: session is closed")
	}
	if f.closeOnCallRaw {
		f.closed = true
	}
	return f.callRawResult, f.callRawErr
}

func (f *fakeConvSession) CallWithCallbacks(ctx context.Context, request []byte, _ client.CallbackHandler) (client.CallResult, error) {
	if f.closed {
		return client.CallResult{}, errors.New("cpic session: session is closed")
	}
	if f.closeOnCall {
		f.closed = true
	}
	return f.callResult, f.callErr
}

func (f *fakeConvSession) Authenticated() bool { return true }
func (f *fakeConvSession) Closed() bool        { return f.closed }
func (f *fakeConvSession) Close() error        { f.closed = true; return nil }

// flatIface is a minimal flat interface: no layout-bearing parameters, so a
// call plans no recursive layout and touches no structure metadata.
func flatIface() metadata.RfcFunctionInterface {
	return metadata.RfcFunctionInterface{Name: "Z_FLAT", Parameters: []classicrfc.FunintParameter{
		param("REQUTEXT", "I", "C", "", 10),
		param("ECHOTEXT", "E", "C", "", 10),
	}}
}

// deepIface has one EXID 'v' parameter, which is always graph-serialized, so
// every call must fetch the recursive type graph first.
func deepIface() metadata.RfcFunctionInterface {
	return metadata.RfcFunctionInterface{Name: "Z_DEEP", Parameters: []classicrfc.FunintParameter{
		param("DATA", "I", "v", "ZDEEP", 0),
	}}
}

// cachedClient is a poolless client whose function cache is pre-filled, so
// callOn never fetches an interface over the wire.
func cachedClient(ifaces ...metadata.RfcFunctionInterface) *Client {
	c := &Client{fnCache: map[string]metadata.RfcFunctionInterface{}, graphCache: map[string]metadata.Graph{}}
	for _, iface := range ifaces {
		c.fnCache[iface.Name] = iface
	}
	return c
}

// pooledClient wires the same pre-filled client to a real lifecycle pool over
// scripted fake sessions. Health checks are skipped so the tests observe the
// discard-vs-release decision itself, not the acquire-time validation that
// would otherwise mask it.
func pooledClient(t *testing.T, maxSize int, open func() *fakeConvSession) (*Client, *int) {
	t.Helper()
	opens := new(int)
	p, err := lifecycle.NewPool(lifecycle.PoolOptions{
		Open: func(ctx context.Context) (*lifecycle.Managed, error) {
			*opens++
			return lifecycle.Wrap(open()), nil
		},
		Pool:            pool.Config{MaxSize: maxSize},
		SkipHealthCheck: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close(context.Background()) })
	c := cachedClient(flatIface(), deepIface())
	c.pool = p
	return c, opens
}

// --- Fix 1: the ErrTransport wrap must keep the cause chain ------------------

func TestTransportWrapKeepsCauseChain(t *testing.T) {
	causes := []error{context.Canceled, os.ErrDeadlineExceeded, transport.ErrClosed}
	for _, cause := range causes {
		t.Run(cause.Error(), func(t *testing.T) {
			c := cachedClient(flatIface())
			sess := lifecycle.Wrap(&fakeConvSession{callErr: cause})
			_, err := c.callOn(context.Background(), sess, "Z_FLAT", Params{"REQUTEXT": "hi"})
			if !errors.Is(err, ErrTransport) {
				t.Fatalf("err = %v, want ErrTransport", err)
			}
			if !errors.Is(err, cause) {
				t.Fatalf("cause %v is not reachable through the wrap: %v", cause, err)
			}
		})
	}
}

func TestMetadataTransportWrapKeepsCauseChain(t *testing.T) {
	c := cachedClient()
	sess := lifecycle.Wrap(&fakeConvSession{callRawErr: os.ErrDeadlineExceeded})
	if _, err := c.functionInterfaceOn(context.Background(), sess, "Z_ANY"); !errors.Is(err, ErrTransport) || !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("interface fetch: cause lost: %v", err)
	}
	if _, err := c.structureDefinitionOn(context.Background(), sess, "ZS_ANY"); !errors.Is(err, ErrTransport) || !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("structure fetch: cause lost: %v", err)
	}
}

// --- Fix 2: server-side deallocation must not become a delayed time bomb -----

// TestPinnedSessionDiscardsAfterServerDeallocation: a call whose reply arrives
// with a normal deallocation succeeds, but the pinned session must drop its
// dead connection so the next call fails clean with ErrClosed instead of a
// CPIC-level "session is closed".
func TestPinnedSessionDiscardsAfterServerDeallocation(t *testing.T) {
	c, opens := pooledClient(t, 2, func() *fakeConvSession {
		return &fakeConvSession{callResult: successResult(), closeOnCall: true}
	})
	s, err := c.Pin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Call(context.Background(), "Z_FLAT", nil); err != nil {
		t.Fatalf("the deallocated call itself must still return its result: %v", err)
	}
	if _, err := s.Call(context.Background(), "Z_FLAT", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("next call = %v, want ErrClosed", err)
	}
	if *opens != 1 {
		t.Fatalf("opens = %d, want 1 (a discarded pin must not silently reconnect)", *opens)
	}
}

// TestPinnedSessionDiscardsAfterAbapExceptionWithDeallocation covers the auth
// MESSAGE shape: the call returns a clean *ABAPException, the conversation is
// gone, and the pinned session must not stay bound to it.
func TestPinnedSessionDiscardsAfterAbapExceptionWithDeallocation(t *testing.T) {
	c, _ := pooledClient(t, 2, func() *fakeConvSession {
		return &fakeConvSession{callResult: abapMessageResult(), closeOnCall: true}
	})
	s, err := c.Pin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, err = s.Call(context.Background(), "Z_FLAT", nil)
	var exc *ABAPException
	if !errors.As(err, &exc) || exc.Kind != KindMessage {
		t.Fatalf("err = %v, want the original *ABAPException", err)
	}
	if _, err := s.Call(context.Background(), "Z_FLAT", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("next call = %v, want ErrClosed", err)
	}
}

// TestPooledCallDiscardsDeadConversation: a pooled call whose conversation the
// server deallocated must discard the connection, so the next call opens a
// fresh one instead of reusing the corpse.
func TestPooledCallDiscardsDeadConversation(t *testing.T) {
	c, opens := pooledClient(t, 1, func() *fakeConvSession {
		return &fakeConvSession{callResult: successResult(), closeOnCall: true}
	})
	if _, err := c.Call(context.Background(), "Z_FLAT", nil); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := c.Call(context.Background(), "Z_FLAT", nil); err != nil {
		t.Fatalf("second call must run on a fresh connection: %v", err)
	}
	if *opens != 2 {
		t.Fatalf("opens = %d, want 2 (the dead connection must not re-enter the pool)", *opens)
	}
}

// --- Fix 3: metadata failures fail clean, and are never cached ---------------

func TestGraphCacheDoesNotRetainFailures(t *testing.T) {
	c := cachedClient(deepIface())
	fake := &fakeConvSession{callRawErr: errors.New("recv: connection reset by peer")}
	sess := lifecycle.Wrap(fake)
	if _, err := c.recursiveGraphOn(context.Background(), sess, "Z_DEEP"); !errors.Is(err, ErrTransport) {
		t.Fatalf("err = %v, want ErrTransport", err)
	}
	if _, ok := c.cachedGraph("Z_DEEP"); ok {
		t.Fatal("a failed lookup must not land in the graph cache")
	}
	if _, err := c.recursiveGraphOn(context.Background(), sess, "Z_DEEP"); err == nil {
		t.Fatal("second lookup should have retried and failed again")
	}
	if fake.callRawCount != 2 {
		t.Fatalf("callRawCount = %d, want 2 (the failure must be re-probed, not served from cache)", fake.callRawCount)
	}
}

// panicResolver asserts that a 'v' parameter never resolves flatly.
func panicResolver(name string) (rfctypes.RfcStructureDefinition, error) {
	panic("planLayoutOn resolved " + name + " flatly")
}

func TestPlanLayoutFailsWhenMetadataLookupKillsTheConversation(t *testing.T) {
	cases := []struct {
		name string
		fake *fakeConvSession
	}{
		{"transport error", &fakeConvSession{callRawErr: errors.New("connection reset")}},
		{"auth MESSAGE with deallocation", &fakeConvSession{callRawResult: abapMessageResult(), closeOnCallRaw: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := cachedClient(deepIface())
			sess := lifecycle.Wrap(tc.fake)
			plan, err := c.planLayoutOn(context.Background(), sess, deepIface(), panicResolver)
			if err == nil {
				t.Fatalf("plan = %v, want an error: the conversation is dead", plan)
			}
			if !strings.Contains(err.Error(), "Z_DEEP") || !strings.Contains(err.Error(), "metadata lookup failed") {
				t.Fatalf("the error must name the metadata lookup, got: %v", err)
			}
		})
	}
}

func TestPlanLayoutFallsBackWhenTheConversationSurvives(t *testing.T) {
	// A clean ABAP exception without deallocation is the "old release without
	// RFC_METADATA_GET" shape: fall back silently to the flat path.
	fake := &fakeConvSession{callRawResult: client.CallResult{Envelope: rfcerr.Envelope{
		Outcome: rfcerr.OutcomeAbapException,
		Facts:   rfcerr.RemoteErrorFacts{ExceptionKey: "FU_NOT_FOUND"},
	}}}
	c := cachedClient(deepIface())
	plan, err := c.planLayoutOn(context.Background(), lifecycle.Wrap(fake), deepIface(), panicResolver)
	if err != nil {
		t.Fatalf("a live conversation must fall back, got: %v", err)
	}
	if plan != nil {
		t.Fatalf("plan = %v, want nil (flat fallback)", plan)
	}
}

// TestDeniedMetadataLookupOnPinnedSession is the real-world trace: on a
// locked-down system the metadata lookup for a deep parameter is denied by
// S_RFC, the auth MESSAGE deallocates the conversation, and the user must see
// the denied lookup — not "session is closed" — with the pinned session
// unbound so the next call fails clean.
func TestDeniedMetadataLookupOnPinnedSession(t *testing.T) {
	c, _ := pooledClient(t, 2, func() *fakeConvSession {
		return &fakeConvSession{callRawResult: abapMessageResult(), closeOnCallRaw: true}
	})
	s, err := c.Pin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, err = s.Call(context.Background(), "Z_DEEP", Params{"DATA": map[string]any{}})
	if err == nil || !strings.Contains(err.Error(), "metadata lookup failed") {
		t.Fatalf("err = %v, want the denied metadata lookup named", err)
	}
	var exc *ABAPException
	if !errors.As(err, &exc) {
		t.Fatalf("the original ABAP message must stay reachable: %v", err)
	}
	if _, ok := c.cachedGraph("Z_DEEP"); ok {
		t.Fatal("the denial must not be cached")
	}
	if _, err := s.Call(context.Background(), "Z_DEEP", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("next call = %v, want ErrClosed", err)
	}
}
