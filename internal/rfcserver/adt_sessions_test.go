// SPDX-License-Identifier: Apache-2.0
// Assertions adapted from open-steamgate tools/protocols/rfc-server.mjs and
// test/adt-rfc-context.mjs at 7d8e0617a8e5ded4ead65da534acf1f1b2021863
// (MIT there), ported by its copyright holder under Apache-2.0.
// Modified for Go: direct cleanup deadlines and session ownership assertions.
// See docs/provenance.md.

package rfcserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"
)

type sessionRoundTripper func(*http.Request) (*http.Response, error)

func (f sessionRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLogoffDeadlineAndNoRetry(t *testing.T) {
	for _, timeout := range []time.Duration{25 * time.Millisecond, 120 * time.Second} {
		for _, failure := range []bool{false, true} {
			jar, _ := cookiejar.New(nil)
			calls, logs := 0, 0
			client := &http.Client{Jar: jar, Transport: sessionRoundTripper(func(r *http.Request) (*http.Response, error) {
				calls++
				deadline, ok := r.Context().Deadline()
				want := timeout
				if want > 5*time.Second {
					want = 5 * time.Second
				}
				if !ok || time.Until(deadline) > want || time.Until(deadline) < want-time.Second {
					t.Errorf("cleanup deadline = %v, want %v", time.Until(deadline), want)
				}
				if r.URL.Path != "/sap/public/bc/icf/logoff" || r.Method != "GET" {
					t.Error("wrong logoff target")
				}
				if r.Header.Get("Cookie") != "ctx=private" {
					t.Error("missing logoff cookie")
				}
				if failure {
					return nil, context.DeadlineExceeded
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
			})}
			session := NewADTSession(client)
			base, _ := url.Parse("http://backend")
			session.client.Jar.SetCookies(base, []*http.Cookie{{Name: "ctx", Value: "private", Path: "/sap/bc/adt"}})
			session.csrf.set("private-token")
			session.End(Backend{URL: base.String()}, timeout, func(f string, a ...any) { logs++ })
			session.End(Backend{URL: base.String()}, timeout, nil)
			if calls != 1 || logs != 1 || session.csrf.get() != "" {
				t.Fatalf("calls=%d logs=%d token=%q", calls, logs, session.csrf.get())
			}
		}
	}
	// With no cookies, disconnect does no HTTP work.
	jar, _ := cookiejar.New(nil)
	empty := NewADTSession(&http.Client{Jar: jar, Transport: sessionRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Error("empty session logged off")
		return nil, errors.New("unexpected")
	})})
	empty.End(Backend{URL: "http://backend"}, time.Second, nil)
}

func TestEditingHandleShapeOwnershipAndDisconnect(t *testing.T) {
	sessions := NewADTSessions()
	owner, other := NewADTSession(&http.Client{}), NewADTSession(&http.Client{})
	identity := LogonIdentity{User: "DEVELOPER", Client: "001"}
	target, _ := url.Parse("http://backend/object?_action=LOCK")
	decoded := map[string]any{"REQUEST_LINE": map[string]any{"METHOD": "POST"}}
	for _, handle := range []string{strings.Repeat("a", 39), strings.Repeat("g", 40), strings.Repeat("a", 41)} {
		sessions.record(owner, nil, identity, target, decoded, []byte("<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>"))
	}
	if len(sessions.handles) != 0 {
		t.Fatal("invalid handle admitted")
	}
	handle := strings.Repeat("aB", 20)
	body := []byte("<LOCK_HANDLE>" + handle + "</LOCK_HANDLE>")
	sessions.record(owner, nil, identity, target, decoded, body)
	for i := range body {
		body[i] = 0
	}
	selected, _ := sessions.selectSession(other, handle, identity)
	if selected != owner {
		t.Fatal("handle did not retain owned bytes")
	}
	sessions.Disconnect(owner)
	sessions.record(owner, nil, identity, target, decoded, []byte("<LOCK_HANDLE>"+handle+"</LOCK_HANDLE>"))
	selected, _ = sessions.selectSession(other, handle, identity)
	if selected != other || len(sessions.handles) != 0 {
		t.Fatal("closed owner retained or reacquired a handle")
	}
}
