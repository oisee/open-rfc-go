// SPDX-License-Identifier: Apache-2.0
// Session assertions adapted from open-steamgate test/adt-rfc-context.mjs and
// test/protocol-rfc.mjs at 7d8e0617a8e5ded4ead65da534acf1f1b2021863
// (MIT there), ported by its copyright holder under Apache-2.0.
// Modified for Go: synthetic HTTP backend, TCP clients, and operational limit tests.
// See docs/provenance.md.

package adtbridge

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

func bridgeAddress(t *testing.T, o Options) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, ln, o) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	return ln.Addr().String()
}
func dialBridge(t *testing.T, addr string) net.Conn {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	c.SetDeadline(time.Now().Add(5 * time.Second))
	return c
}
func loggedClient(t *testing.T, addr, user, client string) net.Conn {
	t.Helper()
	c := dialBridge(t, addr)
	for _, b := range [][]byte{make([]byte, 64), appc(1, 453), appc(5, 80)} {
		send(t, c, b)
		if _, err := receive(c); err != nil {
			t.Fatal(err)
		}
	}
	packet := logon(t, "secret", "probe")
	if len(user) != len("DEVELOPER") || len(client) != 3 {
		t.Fatal("test identity width")
	}
	packet = bytes.Replace(packet, []byte("DEVELOPER"), []byte(user), 1)
	packet = bytes.Replace(packet, []byte("001"), []byte(client), 1)
	send(t, c, data(packet))
	if _, err := receive(c); err != nil {
		t.Fatal(err)
	}
	return c
}
func callPacket(t testing.TB, method, path string) []byte {
	return callPacketSessiontype(t, method, path, nil)
}
func callPacketSessiontype(t testing.TB, method, path string, sessiontype *string) []byte {
	t.Helper()
	path = strings.ReplaceAll(path, "&", "&amp;")
	xml := fmt.Sprintf(`<REQUEST><REQUEST_LINE><METHOD>%s</METHOD><URI>%s</URI><VERSION>HTTP/1.1</VERSION></REQUEST_LINE><HEADER_FIELDS><item><NAME>Authorization</NAME><VALUE>Basic U1BPT0ZFRDpwdw==</VALUE></item><item><NAME>Cookie</NAME><VALUE>stolen=1</VALUE></item></HEADER_FIELDS><MESSAGE_BODY></MESSAGE_BODY></REQUEST>`, method, path)
	if sessiontype != nil {
		xml = strings.Replace(xml, "</HEADER_FIELDS>", "<item><NAME>X-SAP-ADT-Sessiontype</NAME><VALUE>"+*sessiontype+"</VALUE></item></HEADER_FIELDS>", 1)
	}
	packet, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{FunctionName: "SADT_REST_RFC_ENDPOINT", RequestedOutputs: []string{"RESPONSE"}, XrfcParameters: []cpic.NamedValue{{Name: "REQUEST", Value: []byte(xml)}}})
	if err != nil {
		t.Fatal(err)
	}
	return data(packet)
}
func callBody(t *testing.T, c net.Conn, method, path, want string) {
	t.Helper()
	send(t, c, callPacket(t, method, path))
	reply, err := receive(c)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(reply, []byte(base64.StdEncoding.EncodeToString([]byte(want)))) {
		t.Fatalf("response missing body %q: %s", want, reply)
	}
}

func TestLimitsDefaultsAndValidation(t *testing.T) {
	if (Options{}).timeout() != 120*time.Second || DefaultMaxConnections != 64 || (Options{}).shutdownGrace() != 5*time.Second {
		t.Fatal("JS defaults changed")
	}
	for _, o := range []Options{{Timeout: -time.Second}, {MaxConnections: -1}, {ShutdownGrace: -time.Second}} {
		o.Backend.URL = "http://127.0.0.1"
		if err := Serve(context.Background(), noAccept{}, o); !errors.Is(err, ErrLimitsConfiguration) {
			t.Fatal(err)
		}
	}
}

func TestConcurrentConnectionCapAndSlotReuse(t *testing.T) {
	for _, limit := range []int{0, 1, 3} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			gone := make(chan struct{}, 128)
			addr := bridgeAddress(t, Options{Backend: Backend{URL: "http://127.0.0.1"}, MaxConnections: limit, Log: func(f string, a ...any) {
				if strings.HasSuffix(fmt.Sprintf(f, a...), " gone") {
					gone <- struct{}{}
				}
			}})
			count := limit
			if count == 0 {
				count = 64
			}
			var clients []net.Conn
			for i := 0; i < count; i++ {
				c := dialBridge(t, addr)
				send(t, c, make([]byte, 64))
				if _, err := receive(c); err != nil {
					t.Fatalf("slot %d: %v", i, err)
				}
				clients = append(clients, c)
			}
			excess := dialBridge(t, addr)
			if b, err := receive(excess); err == nil {
				t.Fatalf("excess socket answered: %x", b)
			}
			clients[0].Close()
			select {
			case <-gone:
			case <-time.After(time.Second):
				t.Fatal("socket slot not released")
			}
			next := dialBridge(t, addr)
			send(t, next, make([]byte, 64))
			if _, err := receive(next); err != nil {
				t.Fatal("released slot unavailable:", err)
			}
		})
	}
}

func TestSocketIdleTimeoutResetsOnTraffic(t *testing.T) {
	const timeout = 200 * time.Millisecond
	addr := bridgeAddress(t, Options{Backend: Backend{URL: "http://127.0.0.1"}, Timeout: timeout})
	c := dialBridge(t, addr)
	for i := 0; i < 5; i++ {
		send(t, c, make([]byte, 64))
		if _, err := receive(c); err != nil {
			t.Fatal("active socket closed:", err)
		}
		time.Sleep(timeout / 3)
	}
	start := time.Now()
	if _, err := receive(c); err == nil {
		t.Fatal("idle socket remained open")
	}
	if time.Since(start) > 3*timeout {
		t.Fatal("idle close exceeded configured timeout")
	}
}

func TestBackendRequestTimeout(t *testing.T) {
	canceled := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done(); close(canceled) }))
	defer backend.Close()
	o := Options{Backend: Backend{URL: backend.URL}, Timeout: 100 * time.Millisecond}
	c := session(t, o)
	send(t, c, data(logon(t, "anything", "probe")))
	if _, err := receive(c); err != nil {
		t.Fatal(err)
	}
	send(t, c, callPacket(t, "GET", "/slow"))
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("backend request did not honor Timeout")
	}
}

func TestCallerIdentityToHTTPMatchesJS(t *testing.T) {
	for _, tc := range []struct{ mode, user, want string }{
		{"demo", "", "DEVELOPER"}, {"demo", "BACKEND", "BACKEND"},
		{"static", "", ""}, {"static", "BACKEND", "BACKEND"},
	} {
		t.Run(tc.mode+"/"+tc.user, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, password, ok := r.BasicAuth()
				if user != tc.want || password != "" || ok != (tc.want != "") {
					t.Errorf("Basic auth = %q/%q/%v", user, password, ok)
				}
				if r.Header.Get("Cookie") != "" {
					t.Error("caller cookie forwarded")
				}
				if r.URL.Query().Get("sap-client") != "777" || r.URL.Query().Get("sap-language") != "DE" {
					t.Error("backend client/language replaced by caller")
				}
				if r.Header.Get("X-SAP-ADT-Sessiontype") != "stateful" {
					t.Error("sessiontype missing")
				}
				fmt.Fprint(w, "identity-ok")
			}))
			defer backend.Close()
			o := staticOptions(backend.URL)
			o.AuthMode = tc.mode
			o.Backend.User = tc.user
			o.Backend.Client, o.Backend.Language = "777", "DE"
			addr := bridgeAddress(t, o)
			c := loggedClient(t, addr, "DEVELOPER", "001")
			callBody(t, c, "GET", "/identity", "identity-ok")
		})
	}
}

func TestBackendLogoffOnDisconnect(t *testing.T) {
	for _, cookie := range []bool{false, true} {
		t.Run(fmt.Sprint(cookie), func(t *testing.T) {
			logoffs := make(chan struct{}, 2)
			ended := make(chan struct{}, 2)
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/sap/public/bc/icf/logoff" {
					if r.Header.Get("Cookie") != "ctx=owned" {
						t.Error("logoff lost path-scoped cookie")
					}
					if user, _, _ := r.BasicAuth(); user != "DEVELOPER" {
						t.Error("logoff lost conversation identity")
					}
					logoffs <- struct{}{}
					return
				}
				if cookie {
					http.SetCookie(w, &http.Cookie{Name: "ctx", Value: "owned", Path: "/sap/bc/adt"})
				}
				fmt.Fprint(w, "ready")
			}))
			defer backend.Close()
			o := Options{Backend: Backend{URL: backend.URL}, Log: func(f string, a ...any) {
				if f == "backend-logoff: %d" {
					ended <- struct{}{}
				}
			}}
			c := session(t, o)
			send(t, c, data(logon(t, "anything", "probe")))
			if _, err := receive(c); err != nil {
				t.Fatal(err)
			}
			callBody(t, c, "GET", "/sap/bc/adt/ready", "ready")
			c.Close()
			if cookie {
				select {
				case <-ended:
				case <-time.After(time.Second):
					t.Fatal("no backend logoff")
				}
				if len(logoffs) != 1 {
					t.Fatal("expected exactly one backend logoff")
				}
			}
		})
	}
}

// Synthetic equivalent of upstream's editing-context tests: the backend echoes
// the session selected by the bridge, so no ABAP runtime is required.
func TestLockHandlesAcrossConnections(t *testing.T) {
	const handle = "0123456789abcdef0123456789abcdef01234567"
	var next atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("ctx")
		id := ""
		if cookie != nil {
			id = cookie.Value
		}
		if id == "" {
			id = fmt.Sprint(next.Add(1))
			http.SetCookie(w, &http.Cookie{Name: "ctx", Value: id, Path: "/"})
		}
		if r.Header.Get("X-CSRF-Token") == "fetch" {
			w.Header().Set("X-CSRF-Token", "token-"+id)
			return
		}
		if r.URL.Path == "/sap/public/bc/icf/logoff" {
			return
		}
		if r.Method != "GET" && r.Header.Get("X-CSRF-Token") != "token-"+id {
			t.Errorf("CSRF token does not belong to selected session: %q/%s", r.Header.Get("X-CSRF-Token"), id)
		}
		if r.URL.Query().Get("_action") == "LOCK" {
			fmt.Fprint(w, "<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>")
			return
		}
		fmt.Fprint(w, "ctx-"+id)
	}))
	defer backend.Close()
	addr := bridgeAddress(t, Options{Backend: Backend{URL: backend.URL}})
	one := loggedClient(t, addr, "DEVELOPER", "001")
	two := loggedClient(t, addr, "DEVELOPER", "001")
	foreign := loggedClient(t, addr, "OTHERUSER", "001")
	otherClient := loggedClient(t, addr, "DEVELOPER", "002")
	callBody(t, one, "GET", "/state", "ctx-1")
	callBody(t, two, "GET", "/state", "ctx-2")
	callBody(t, foreign, "GET", "/state", "ctx-3")
	callBody(t, otherClient, "GET", "/state", "ctx-4")
	lock := "/object?_action=LOCK"
	held := "/object?lockHandle=" + handle
	callBody(t, one, "POST", lock, "<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>")
	callBody(t, foreign, "PUT", held, "ctx-3")
	callBody(t, otherClient, "PUT", held, "ctx-4")
	callBody(t, foreign, "POST", held+"&_action=UNLOCK", "ctx-3")
	callBody(t, two, "PUT", held, "ctx-1")
	callBody(t, two, "PUT", "/object?lockHandle="+strings.Repeat("0", 40), "ctx-2")
	callBody(t, two, "POST", held+"&_action=UNLOCK", "ctx-1")
	callBody(t, two, "PUT", held, "ctx-2")
	callBody(t, one, "POST", lock, "<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>")
	callBody(t, two, "GET", "/sap/public/bc/icf/logoff?lockHandle="+handle, "")
	callBody(t, two, "PUT", held, "ctx-2")
	// A handle belongs to this listener, never to another bridge instance.
	callBody(t, one, "POST", lock, "<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>")
	separate := bridgeAddress(t, Options{Backend: Backend{URL: backend.URL}})
	outsider := loggedClient(t, separate, "DEVELOPER", "001")
	callBody(t, outsider, "GET", "/state", "ctx-5")
	callBody(t, outsider, "PUT", held, "ctx-5")
	one.Close()
	// Owner close cleanup is covered separately below with deterministic barriers.
}

func TestDisconnectWaitsForPendingBackendWork(t *testing.T) {
	for _, during := range []string{"LOCK", "pooled SAVE"} {
		t.Run(during, func(t *testing.T) {
			const handle = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			started, release, logoff := make(chan struct{}), make(chan struct{}), make(chan struct{}, 4)
			gone := make(chan struct{}, 4)
			var pause atomic.Bool
			var next atomic.Int32
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/sap/public/bc/icf/logoff" {
					logoff <- struct{}{}
					return
				}
				cookie, _ := r.Cookie("ctx")
				id := ""
				if cookie != nil {
					id = cookie.Value
				}
				if id == "" {
					id = fmt.Sprint(next.Add(1))
					http.SetCookie(w, &http.Cookie{Name: "ctx", Value: id, Path: "/"})
				}
				if r.Header.Get("X-CSRF-Token") == "fetch" {
					w.Header().Set("X-CSRF-Token", "token-"+id)
					return
				}
				if pause.CompareAndSwap(true, false) {
					close(started)
					<-release
				}
				if r.URL.Query().Get("_action") == "LOCK" {
					fmt.Fprint(w, "<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>")
					return
				}
				fmt.Fprint(w, "ctx-"+id)
			}))
			defer backend.Close()
			// Releasing on any failure keeps backend.Close from waiting forever.
			var released atomic.Bool
			unblock := func() {
				if released.CompareAndSwap(false, true) {
					close(release)
				}
			}
			defer unblock()
			addr := bridgeAddress(t, Options{Backend: Backend{URL: backend.URL}, Log: func(f string, a ...any) {
				if strings.HasSuffix(fmt.Sprintf(f, a...), " gone") {
					gone <- struct{}{}
				}
			}})
			owner := loggedClient(t, addr, "DEVELOPER", "001")
			borrower := loggedClient(t, addr, "DEVELOPER", "001")
			pending := owner
			if during == "pooled SAVE" {
				callBody(t, owner, "POST", "/object?_action=LOCK", "<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>")
				pending = borrower
			}
			pause.Store(true)
			method, path := "POST", "/object?_action=LOCK"
			if during == "pooled SAVE" {
				method, path = "PUT", "/object?lockHandle="+handle
			}
			send(t, pending, callPacket(t, method, path))
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("backend work not started")
			}
			owner.Close()
			if during == "pooled SAVE" {
				// The owner is idle, so its read loop observes EOF immediately.
				select {
				case <-gone:
				case <-time.After(time.Second):
					t.Fatal("owner did not close")
				}
			}
			select {
			case <-logoff:
				t.Fatal("logged off before pending work finished")
			default:
			}
			unblock()
			if pending == borrower {
				if _, err := receive(borrower); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case <-logoff:
			case <-time.After(time.Second):
				t.Fatal("no logoff after pending work")
			}
			// The pending LOCK must not leave an adoptable handle after owner close.
			callBody(t, borrower, "PUT", "/object?lockHandle="+handle, "ctx-2")
		})
	}
}

func TestSharedContextSerializesCookieAndCSRFRotation(t *testing.T) {
	const handle = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	entered, release := make(chan struct{}, 2), make(chan struct{})
	var writes atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sap/public/bc/icf/logoff" {
			return
		}
		if r.Header.Get("X-CSRF-Token") == "fetch" {
			http.SetCookie(w, &http.Cookie{Name: "ctx", Value: "initial", Path: "/"})
			w.Header().Set("X-CSRF-Token", "token-initial")
			return
		}
		if r.URL.Query().Get("_action") == "LOCK" {
			fmt.Fprint(w, "<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>")
			return
		}
		n := writes.Add(1)
		entered <- struct{}{}
		want := "initial"
		if n == 1 {
			<-release
			http.SetCookie(w, &http.Cookie{Name: "ctx", Value: "rotated", Path: "/"})
			w.Header().Set("X-CSRF-Token", "token-rotated")
		} else {
			want = "rotated"
		}
		cookie, err := r.Cookie("ctx")
		if err != nil || cookie.Value != want || r.Header.Get("X-CSRF-Token") != "token-"+want {
			t.Errorf("write %d lost serialized cookie/token state", n)
		}
		fmt.Fprint(w, "saved")
	}))
	defer backend.Close()
	var released atomic.Bool
	unblock := func() {
		if released.CompareAndSwap(false, true) {
			close(release)
		}
	}
	defer unblock()
	addr := bridgeAddress(t, Options{Backend: Backend{URL: backend.URL}})
	owner := loggedClient(t, addr, "DEVELOPER", "001")
	two := loggedClient(t, addr, "DEVELOPER", "001")
	three := loggedClient(t, addr, "DEVELOPER", "001")
	callBody(t, owner, "POST", "/object?_action=LOCK", "<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>")
	packet := callPacket(t, "PUT", "/object?lockHandle="+handle)
	send(t, two, packet)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first SAVE did not start")
	}
	send(t, three, packet)
	select {
	case <-entered:
		t.Fatal("shared context ran concurrent SAVE requests")
	case <-time.After(50 * time.Millisecond):
	}
	unblock()
	for _, c := range []net.Conn{two, three} {
		if _, err := receive(c); err != nil {
			t.Fatal(err)
		}
	}
	if writes.Load() != 2 {
		t.Fatal("pending SAVE lost")
	}
}

func TestDemoLogonBoundsAndOwnership(t *testing.T) {
	packet := logon(t, "anything", "probe")
	for n := 0; n < len(packet); n++ {
		if _, err := rfcserver.DecodeDemoLogon(packet[:n], rfcserver.LogonIdentity{}, 0); err == nil {
			t.Fatalf("truncated demo logon admitted at %d", n)
		}
	}
	if _, err := rfcserver.DecodeDemoLogon(packet, rfcserver.LogonIdentity{}, len(packet)-1); err == nil {
		t.Fatal("demo logon budget ignored")
	}
	identity, err := rfcserver.DecodeDemoLogon(packet, rfcserver.LogonIdentity{}, len(packet))
	if err != nil {
		t.Fatal(err)
	}
	for i := range packet {
		packet[i] = 0
	}
	if identity.User != "DEVELOPER" || identity.Client != "001" || identity.Language != "E" {
		t.Fatal("demo identity aliases network bytes")
	}
}

func FuzzDemoLogon(f *testing.F) {
	f.Add(logon(f, "anything", "probe"))
	f.Add(logon(f, "anything", "jco"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		who, err := rfcserver.DecodeDemoLogon(b, rfcserver.LogonIdentity{}, 4096)
		if err == nil {
			if who.User == "" || who.Client == "" || who.Language == "" {
				t.Fatal("empty demo identity")
			}
			for _, field := range []string{who.User, who.Client, who.Language} {
				for _, ch := range []byte(field) {
					if ch < 0x20 || ch > 0x7e {
						t.Fatal("non-ASCII demo identity")
					}
				}
			}
		}
	})
}

func TestExplicitSessiontypeIsPreserved(t *testing.T) {
	for _, value := range []string{"stateless", ""} {
		t.Run(value, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				values, present := r.Header[http.CanonicalHeaderKey("X-SAP-ADT-Sessiontype")]
				if !present || len(values) != 1 || values[0] != value {
					t.Errorf("sessiontype = %q, present=%v", values, present)
				}
			}))
			defer backend.Close()
			c := session(t, Options{Backend: Backend{URL: backend.URL}})
			send(t, c, data(logon(t, "anything", "probe")))
			if _, err := receive(c); err != nil {
				t.Fatal(err)
			}
			send(t, c, callPacketSessiontype(t, "GET", "/state", &value))
			if _, err := receive(c); err != nil {
				t.Fatal(err)
			}
		})
	}
}
