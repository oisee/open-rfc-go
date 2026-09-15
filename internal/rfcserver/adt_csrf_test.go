// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/oisee/open-rfc-go/internal/bxml"
)

// A backend that behaves as ADT does: it refuses a modifying request without a
// token, says so in the header the way ICF says it, and mints one for a probe.
type csrfBackend struct {
	token    atomic.Value // string
	probes   atomic.Int32
	posts    atomic.Int32
	headOnly bool // when false, HEAD mints no token and the client must fall back to GET
}

func (c *csrfBackend) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-CSRF-Token") == "fetch" {
			c.probes.Add(1)
			if r.Method == http.MethodHead && !c.headOnly {
				w.WriteHeader(http.StatusOK) // some systems mint nothing on HEAD
				return
			}
			w.Header().Set("X-CSRF-Token", c.token.Load().(string))
			w.WriteHeader(http.StatusOK)
			return
		}
		if isModifyingMethod(r.Method) {
			c.posts.Add(1)
			if r.Header.Get("X-CSRF-Token") != c.token.Load().(string) {
				w.Header().Set("X-CSRF-Token", "Required")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte("CSRF token validation failed"))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<ok/>"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<read/>"))
	}
}

func adtCall(t *testing.T, handler Handler, method, uri string, extra ...bxml.Attr) *bxml.Element {
	t.Helper()
	text := func(n, v string) *bxml.Element { return &bxml.Element{Name: n, Text: v, HasText: true} }
	rows := []*bxml.Element{
		{Name: "item", Children: []*bxml.Element{text("NAME", "Accept"), text("VALUE", "application/xml")}},
	}
	for _, a := range extra {
		rows = append(rows, &bxml.Element{Name: "item", Children: []*bxml.Element{text("NAME", a.Name), text("VALUE", a.Value)}})
	}
	doc, err := bxml.Encode(&bxml.Element{Name: "REQUEST", Children: []*bxml.Element{
		{Name: "REQUEST_LINE", Children: []*bxml.Element{text("METHOD", method), text("URI", uri), text("VERSION", "HTTP/1.1")}},
		{Name: "HEADER_FIELDS", Attrs: []bxml.Attr{{Name: "lines", Value: "1"}}, Children: rows},
		{Name: "MESSAGE_BODY", Body: "<payload/>", HasBody: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	req, err := DecodeFunctionRequest(eclipseCall(t, eclipseCallInput{
		function: adtRestFunction, outputs: []string{"RESPONSE"}, compact: doc, guid: bytes.Repeat([]byte{3}, 16),
	}))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, uri, err)
	}
	tree, err := bxml.Decode(resp.Compact)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := bxml.PayloadRoot(tree)
	return root
}

func statusOf(root *bxml.Element) string {
	return strings.TrimSpace(root.Child("STATUS_LINE").ChildText("STATUS_CODE"))
}

// The failure the live run showed: a POST with no token comes back 403. With
// the dance, the bridge mints one and the POST succeeds.
func TestCSRFTokenIsFetchedForAModifyingRequest(t *testing.T) {
	be := &csrfBackend{headOnly: true}
	be.token.Store("tok-1")
	srv := httptest.NewServer(be.handler(t))
	defer srv.Close()
	handler, err := ADTRestHandler(Backend{URL: srv.URL}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	// a read needs no token and must not provoke a probe
	if got := statusOf(adtCall(t, handler, "GET", "/sap/bc/adt/discovery")); got != "200" {
		t.Fatalf("GET status %s", got)
	}
	if n := be.probes.Load(); n != 0 {
		t.Fatalf("a read provoked %d token probes", n)
	}
	// the write mints a token and succeeds
	if got := statusOf(adtCall(t, handler, "POST", "/sap/bc/adt/repository/typestructure")); got != "200" {
		t.Fatalf("POST status %s, want 200", got)
	}
	if n := be.probes.Load(); n != 1 {
		t.Fatalf("%d token probes, want 1", n)
	}
	// a second write reuses the token it already has
	if got := statusOf(adtCall(t, handler, "POST", "/sap/bc/adt/repository/typestructure")); got != "200" {
		t.Fatalf("second POST status %s", got)
	}
	if n := be.probes.Load(); n != 1 {
		t.Fatalf("the token was not reused: %d probes", n)
	}
}

// A token the server has stopped honouring: the answer names the refusal, and
// the bridge mints a new token and sends the request once more.
func TestStaleCSRFTokenIsRefreshedOnce(t *testing.T) {
	be := &csrfBackend{headOnly: true}
	be.token.Store("tok-1")
	srv := httptest.NewServer(be.handler(t))
	defer srv.Close()
	handler, err := ADTRestHandler(Backend{URL: srv.URL}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if got := statusOf(adtCall(t, handler, "POST", "/x")); got != "200" {
		t.Fatalf("first POST %s", got)
	}
	be.token.Store("tok-2") // the session rolled over; ours is now stale
	if got := statusOf(adtCall(t, handler, "POST", "/x")); got != "200" {
		t.Fatalf("POST after the token went stale: %s, want 200", got)
	}
	if n := be.probes.Load(); n != 2 {
		t.Fatalf("%d probes, want 2 (one per mint)", n)
	}
	if n := be.posts.Load(); n != 3 {
		t.Fatalf("%d posts reached the backend, want 3 (ok, refused, retried)", n)
	}
}

// A 403 that is not a CSRF refusal is the backend's answer and reaches Eclipse
// unaltered — it must not be retried or turned into an error.
func TestPlainForbiddenIsPassedThrough(t *testing.T) {
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-CSRF-Token") == "fetch" {
			w.Header().Set("X-CSRF-Token", "tok")
			return
		}
		posts.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<msg>not authorised</msg>"))
	}))
	defer srv.Close()
	handler, err := ADTRestHandler(Backend{URL: srv.URL}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	root := adtCall(t, handler, "POST", "/x")
	if got := statusOf(root); got != "403" {
		t.Fatalf("status %s, want the backend's 403", got)
	}
	if body := root.Child("MESSAGE_BODY"); !body.HasBody || body.Body != "<msg>not authorised</msg>" {
		t.Fatalf("the backend's body did not reach the caller: %+v", body)
	}
	if n := posts.Load(); n != 1 {
		t.Fatalf("%d attempts, want 1: a plain 403 must not be retried", n)
	}
}

// Some systems mint nothing on HEAD; the probe falls back to GET.
func TestCSRFProbeFallsBackToGet(t *testing.T) {
	be := &csrfBackend{headOnly: false}
	be.token.Store("tok-1")
	srv := httptest.NewServer(be.handler(t))
	defer srv.Close()
	handler, err := ADTRestHandler(Backend{URL: srv.URL}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if got := statusOf(adtCall(t, handler, "POST", "/x")); got != "200" {
		t.Fatalf("POST status %s; the GET fallback did not mint a token", got)
	}
	if n := be.probes.Load(); n != 2 {
		t.Fatalf("%d probes, want 2 (HEAD then GET)", n)
	}
}

// A token the caller put on the request belongs to a session the backend never
// saw, so it is replaced rather than forwarded.
func TestCallersOwnTokenIsReplaced(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-CSRF-Token") == "fetch" {
			w.Header().Set("X-CSRF-Token", "ours")
			return
		}
		seen = r.Header.Get("X-CSRF-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	handler, err := ADTRestHandler(Backend{URL: srv.URL}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	adtCall(t, handler, "POST", "/x", bxml.Attr{Name: "X-CSRF-Token", Value: "eclipses-own"})
	if seen != "ours" {
		t.Fatalf("the backend saw token %q, want ours", seen)
	}
}

func TestIsCSRFTokenRejectsRequired(t *testing.T) {
	for _, v := range []string{"", "Required"} {
		if isCSRFToken(v) {
			t.Fatalf("%q counted as a token", v)
		}
	}
	if !isCSRFToken("abc") {
		t.Fatal("a real token was rejected")
	}
}

// A backend with no CSRF protection at all — our own façade is one — must be
// reachable: the probe mints nothing, and the request goes without a token.
func TestBackendWithoutCSRFStillWorks(t *testing.T) {
	var sawToken bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-CSRF-Token") != "" && r.Header.Get("X-CSRF-Token") != "fetch" {
			sawToken = true
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("<made/>"))
	}))
	defer srv.Close()
	handler, err := ADTRestHandler(Backend{URL: srv.URL}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if got := statusOf(adtCall(t, handler, "POST", "/x")); got != "201" {
		t.Fatalf("POST status %s, want 201", got)
	}
	if sawToken {
		t.Fatal("a token was sent to a backend that mints none")
	}
}
