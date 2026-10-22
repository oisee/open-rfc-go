// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/oisee/open-rfc-go/internal/bxml"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfcerr"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// Exercise admission through RFC encoding and dispatch, including the exception
// envelope a client receives, rather than just calling the URL validator.
func originWireCall(t *testing.T, h Handler, target string) cpic.DecodedFunctionResultFields {
	t.Helper()
	value := map[string]any{
		"REQUEST_LINE":  map[string]any{"METHOD": "GET", "URI": target, "VERSION": "HTTP/1.1"},
		"HEADER_FIELDS": []any{}, "MESSAGE_BODY": []byte{},
	}
	xml, err := xrfc.EncodeRecursiveParameter(requestDescriptor, ADTRestGraph(), value, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatal(err)
	}
	packet, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{FunctionName: adtRestFunction, XrfcParameters: []cpic.NamedValue{{Name: "REQUEST", Value: xml}}})
	if err != nil {
		t.Fatal(err)
	}
	d := NewDispatcher()
	d.Handle(adtRestFunction, h)
	response, err := d.Dispatch(context.Background(), packet)
	if err != nil {
		t.Fatal(err)
	}
	return clientDecode(t, response)
}

func TestADTOriginConfinementOnRFCWire(t *testing.T) {
	var escaped, credentials, reached atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escaped.Add(1)
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			credentials.Add(1)
		}
	}))
	defer other.Close()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached.Add(1)
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, other.URL+"/stolen", http.StatusFound)
			return
		}
	}))
	defer backend.Close()
	for _, client := range []*http.Client{nil, backend.Client()} {
		h, err := ADTRestHandler(Backend{URL: backend.URL, User: "bridge", Password: "synthetic-secret"}, client)
		if err != nil {
			t.Fatal(err)
		}
		for _, target := range []string{other.URL + "/absolute", "//" + strings.TrimPrefix(other.URL, "http://") + "/authority", "https:" + strings.TrimPrefix(other.URL, "http:") + "/scheme", backend.URL + "/same-origin-absolute", "/redirect"} {
			t.Run(target, func(t *testing.T) {
				got := originWireCall(t, h, target)
				if got.Success || got.Envelope.Outcome != rfcerr.OutcomeAbapException || got.Envelope.Facts.ExceptionKey != SystemFailureKey {
					t.Errorf("expected RFC SYSTEM_FAILURE, got success=%v outcome=%v key=%q", got.Success, got.Envelope.Outcome, got.Envelope.Facts.ExceptionKey)
				}
			})
		}
	}
	if n := escaped.Load(); n != 0 {
		t.Errorf("second server received %d requests (%d with credentials)", n, credentials.Load())
	}
	if n := reached.Load(); n != 2 {
		t.Errorf("backend received %d requests, want only two redirect requests", n)
	}
}

func TestADTBasePathAndSameOriginRedirect(t *testing.T) {
	var reached atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached.Add(1)
		if r.URL.Path == "/sap/bc/adt/redirect" {
			http.Redirect(w, r, "/sap/bc/adt/discovery", http.StatusFound)
		}
	}))
	defer srv.Close()
	for _, suffix := range []string{"/sap/bc/adt", "/sap/bc/adt/"} {
		h, err := ADTRestHandler(Backend{URL: srv.URL + suffix}, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, target := range []string{"/sap/bc/adt/discovery?x=1", "/sap/bc/adt/redirect"} {
			if got := originWireCall(t, h, target); !got.Success {
				t.Errorf("valid ADT request %s failed", target)
			}
		}
		before := reached.Load()
		for _, target := range []string{"/outside", "/sap/bc/adt-other", "/sap/bc/adt/../outside", "/sap/bc/adt/%2e%2e/outside", "/sap/bc/adt%2f..%2foutside"} {
			if got := originWireCall(t, h, target); got.Success {
				t.Errorf("escaped base path: %s", target)
			}
		}
		if reached.Load() != before {
			t.Error("out-of-base request reached backend")
		}
	}
}

func TestADTCallerCredentialsAreIgnored(t *testing.T) {
	for _, user := range []string{"", "bridge"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Host == "attacker.invalid" {
				t.Error("caller Host forwarded")
			}
			if got := r.Header.Get("Cookie"); got != "" {
				t.Errorf("caller cookie forwarded: %q", got)
			}
			u, p, ok := r.BasicAuth()
			if user == "" && r.Header.Get("Authorization") != "" {
				t.Error("caller authorization forwarded")
			}
			if user != "" && (!ok || u != user || p != "bridge-secret") {
				t.Error("bridge authentication missing")
			}
		}))
		h, err := ADTRestHandler(Backend{URL: srv.URL, User: user, Password: "bridge-secret"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		adtCall(t, h, "GET", "/sap/bc/adt/discovery", bxml.Attr{Name: "aUtHoRiZaTiOn", Value: "Bearer caller"}, bxml.Attr{Name: "cOoKiE", Value: "session=caller"}, bxml.Attr{Name: "hOsT", Value: "attacker.invalid"})
		srv.Close()
	}
}

func TestADTCSRFProbeRedirectIsConfined(t *testing.T) {
	var escaped, probes atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { escaped.Add(1) }))
	defer other.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probes.Add(1)
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()
	h, err := ADTRestHandler(Backend{URL: srv.URL + "/sap/bc/adt", User: "bridge", Password: "secret"}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	req := compactGet(t, "/sap/bc/adt/discovery")
	tree, err := bxml.Decode(req.Compact)
	if err != nil {
		t.Fatal(err)
	}
	root, err := bxml.PayloadRoot(tree)
	if err != nil {
		t.Fatal(err)
	}
	root.Child("REQUEST_LINE").Child("METHOD").Text = "POST"
	req.Compact, err = bxml.Encode(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h(context.Background(), req)
	if !errors.Is(err, ErrADTTarget) {
		t.Fatalf("probe redirect error = %v", err)
	}
	if probes.Load() != 1 || escaped.Load() != 0 {
		t.Fatalf("probes=%d escaped=%d", probes.Load(), escaped.Load())
	}
}

func TestADTRedirectPolicyAndPathBoundary(t *testing.T) {
	var outside atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sap/bc/adt/redirect" {
			http.Redirect(w, r, "/outside", http.StatusFound)
			return
		}
		outside.Add(1)
	}))
	defer srv.Close()
	var policyCalls int
	client := srv.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error { policyCalls++; return nil }
	h, err := ADTRestHandler(Backend{URL: srv.URL + "/sap/bc/adt"}, client)
	if err != nil {
		t.Fatal(err)
	}
	if got := originWireCall(t, h, "/sap/bc/adt/redirect"); got.Success {
		t.Fatal("redirect escaped base path")
	}
	if outside.Load() != 0 || policyCalls != 0 {
		t.Fatal("unsafe redirect reached caller policy or backend")
	}
	// The handler must not install its policy on the caller's shared client.
	req, err := http.NewRequest("GET", srv.URL+"/outside", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckRedirect(req, nil); err != nil || policyCalls != 1 {
		t.Fatal("caller client was mutated")
	}
}

func FuzzADTRequestTarget(f *testing.F) {
	for _, target := range []string{"/sap/bc/adt/discovery?x=1", "//evil.invalid/x", "https://evil.invalid/x", "/sap/bc/adt/%2e%2e/outside"} {
		f.Add(target)
	}
	base, _ := url.Parse("https://backend.invalid/sap/bc/adt/")
	f.Fuzz(func(t *testing.T, target string) {
		if len(target) > 64<<10 {
			t.Skip()
		}
		req, err := httpRequestOf(context.Background(), base, map[string]any{"REQUEST_LINE": map[string]any{"METHOD": "GET", "URI": target}})
		if err != nil {
			return
		}
		ref, err := url.Parse(target)
		if err != nil || ref.IsAbs() || ref.Host != "" || strings.HasPrefix(target, "//") {
			t.Fatalf("accepted non-path URI %q", target)
		}
		if err := admitBackendURL(base, req.URL); err != nil {
			t.Fatalf("accepted unconfined URI %q: %v", target, err)
		}
	})
}
