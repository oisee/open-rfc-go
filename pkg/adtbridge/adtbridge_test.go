// SPDX-License-Identifier: Apache-2.0
// Disconnect logoff assertion follows open-steamgate test/protocol-rfc.mjs
// at 7d8e0617a8e5ded4ead65da534acf1f1b2021863 (MIT there), ported by its
// copyright holder under Apache-2.0. See docs/provenance.md.
// Assertions adapted from open-steamgate test/protocol-rfc.mjs at
// 13a3c5292eddb240d0c65b15a37a8788661410a4 (MIT there), ported by its copyright holder, who contributes it here under Apache-2.0.
// Modified for Go: net.Pipe, Go encoders, malformed/bounds/ownership tests.
// See docs/provenance.md.

package adtbridge

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfcserver"
	"github.com/oisee/open-rfc-go/internal/scramble"
)

func logon(t testing.TB, password, flavor string) []byte {
	t.Helper()
	field, err := scramble.ScrambleRfcPassword(password, 21)
	if err != nil {
		t.Fatal(err)
	}
	fields := []cpic.Field{
		{Tag: 0x0114, Value: []byte("001")}, {Tag: 0x0111, Value: []byte("DEVELOPER")},
		{Tag: 0x0117, Value: field}, {Tag: 0x0115, Value: []byte("E")},
		{Tag: 0x0514, Value: bytes.Repeat([]byte{7}, 16)}, {Tag: 0xffff},
	}
	chain, err := cpic.EncodeFieldChain(0x0101, fields, cpic.FieldChainLimits{})
	if err != nil {
		t.Fatal(err)
	}
	prefix := []byte{0xd9, 0xc6, 0xc3, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 1, 1, 0, 8, 3, 1}
	if flavor == "jco" {
		prefix[16], prefix[17] = 1, 2
	}
	packet := append(prefix, chain...)
	trailer := make([]byte, 10)
	binary.BigEndian.PutUint16(trailer, 0xffff)
	binary.BigEndian.PutUint16(trailer[4:], uint16(len(packet)+2))
	binary.BigEndian.PutUint32(trailer[6:], 0x8500)
	return append(packet, trailer...)
}

func appc(fn byte, length int) []byte {
	b := make([]byte, length)
	b[0], b[1], b[30], b[31] = 6, fn, 5, 12
	copy(b[40:48], "00000001")
	return b
}
func data(payload []byte) []byte { return append(appc(0xcb, 80), payload...) }
func send(t testing.TB, c net.Conn, payload []byte) {
	t.Helper()
	b := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(b, uint32(len(payload)))
	copy(b[4:], payload)
	if _, err := c.Write(b); err != nil {
		t.Fatal(err)
	}
}
func receive(c net.Conn) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(c, header[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(header[:])
	if n > 1<<20 {
		return nil, fmt.Errorf("oversized test reply: %d", n)
	}
	b := make([]byte, n)
	_, err := io.ReadFull(c, b)
	return b, err
}
func session(t *testing.T, o Options) net.Conn {
	t.Helper()
	c, s := net.Pipe()
	done := make(chan struct{})
	go func() { defer close(done); ServeConn(s, o) }()
	t.Cleanup(func() { c.Close(); <-done })
	c.SetDeadline(time.Now().Add(5 * time.Second))
	for _, b := range [][]byte{make([]byte, 64), appc(1, 453), appc(5, 80)} {
		send(t, c, b)
		if _, err := receive(c); err != nil {
			t.Fatal(err)
		}
	}
	return c
}
func staticOptions(url string) Options {
	return Options{Backend: Backend{URL: url}, AuthMode: "static", RFCUser: "DEVELOPER", RFCPassword: "secret", RFCClient: "001"}
}

// A poisoned listener proves setup returns before accepting or starting work.
type noAccept struct{ net.Listener }

func (noAccept) Accept() (net.Conn, error) { panic("invalid configuration reached Accept") }
func TestIncompleteAuthIsSynchronous(t *testing.T) {
	for _, mutate := range []func(*Options){
		func(o *Options) { o.RFCUser = "" }, func(o *Options) { o.RFCPassword = "" },
		func(o *Options) { o.AuthMode = "unknown" }, func(o *Options) { o.Backend.User = ""; o.Backend.Password = "orphan" },
		func(o *Options) { o.MaxLogonBytes = -1 },
	} {
		o := staticOptions("http://127.0.0.1:3030")
		mutate(&o)
		if err := Check(o); !errors.Is(err, ErrAuthConfiguration) {
			t.Fatalf("Check: %v", err)
		}
		if err := Serve(context.Background(), noAccept{}, o); !errors.Is(err, ErrAuthConfiguration) {
			t.Fatalf("Serve: %v", err)
		}
	}
	if err := Check(Options{Backend: Backend{URL: "http://127.0.0.1"}}); err != nil {
		t.Fatal(err)
	}
}

func TestStaticRejectsBeforeBackend(t *testing.T) {
	var requests atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
	defer backend.Close()
	for _, kind := range []string{"password", "user", "client", "ticket", "missing-password", "call-before-logon", "cpic-init"} {
		t.Run(kind, func(t *testing.T) {
			o := staticOptions(backend.URL)
			logs := make(chan string, 8)
			o.Log = func(f string, args ...any) { logs <- fmt.Sprintf(f, args...) }
			packet := logon(t, "secret", "")
			switch kind {
			case "password":
				packet = logon(t, "secreu", "")
			case "user":
				o.RFCUser = "OTHER"
			case "client":
				o.RFCClient = "002"
			case "ticket":
				at := bytes.Index(packet, []byte{0x01, 0x17})
				packet[at], packet[at+1] = 6, 0x70
				next := at + 4 + 10
				packet[next], packet[next+1] = 6, 0x70
			case "missing-password":
				at := bytes.Index(packet, []byte{0x01, 0x17})
				packet[at+1] = 0x18
				next := at + 4 + 10
				packet[next+1] = 0x18
			case "call-before-logon":
				packet = []byte("not a logon")
			}
			c := session(t, o)
			if kind == "cpic-init" {
				send(t, c, appc(3, 80))
			} else {
				send(t, c, data(packet))
			}
			if b, err := receive(c); err == nil {
				t.Fatalf("rejected logon received %d bytes", len(b))
			}
			found := false
			for !found {
				select {
				case line := <-logs:
					found = line == "RFC authentication failed"
				case <-time.After(time.Second):
					t.Fatal("missing JS-compatible failure log")
				}
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatalf("backend contacted %d times", requests.Load())
	}
}

func TestStaticAcceptAndForwardCookiesCSRF(t *testing.T) {
	var probes, calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "BACKEND" || p != "backend-secret" {
			t.Errorf("backend identity: %q %v", u, ok)
		}
		if r.URL.Query().Get("sap-client") != "777" || r.URL.Query().Get("sap-language") != "DE" {
			t.Error("backend client/language missing")
		}
		if r.URL.Path == "/sap/public/bc/icf/logoff" {
			if r.Header.Get("Cookie") != "bridge-cookie=present" {
				t.Error("logoff cookie missing")
			}
			return
		}
		if r.Header.Get("X-CSRF-Token") == "fetch" {
			n := probes.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "bridge-cookie", Value: "present", Path: "/"})
			w.Header().Set("X-CSRF-Token", fmt.Sprintf("token-%d", n))
			return
		}
		if r.Header.Get("Cookie") != "bridge-cookie=present" {
			t.Error("session cookie missing")
		}
		n := calls.Add(1)
		want := "token-rotated"
		if n == 1 {
			want = "token-1"
		}
		if n == 2 {
			want = "token-2"
		}
		if r.Header.Get("X-CSRF-Token") != want {
			t.Errorf("call %d token=%q want %q", n, r.Header.Get("X-CSRF-Token"), want)
		}
		if n == 1 {
			w.Header().Set("X-CSRF-Token", "Required")
			w.WriteHeader(403)
			return
		}
		if n == 2 {
			w.Header().Set("X-CSRF-Token", "token-rotated")
		}
		fmt.Fprintf(w, "answer-%d", n-1)
	}))
	defer backend.Close()
	o := staticOptions(backend.URL)
	o.Backend.User = "BACKEND"
	o.Backend.Password = "backend-secret"
	o.Backend.Client = "777"
	o.Backend.Language = "DE"
	for _, flavor := range []string{"probe", "jco"} {
		t.Run(flavor, func(t *testing.T) {
			probes.Store(0)
			calls.Store(0)
			c := session(t, o)
			send(t, c, data(logon(t, "secret", flavor)))
			reply, err := receive(c)
			if err != nil {
				t.Fatal(err)
			}
			if len(reply) != 746 {
				t.Fatalf("accept length %d", len(reply))
			}
			if !bytes.Contains(reply, []byte{'D', 0, 'E', 0, 'V', 0, 'E', 0}) {
				t.Fatal("logon reply lost caller identity")
			}
			for n := 1; n <= 2; n++ {
				xml := fmt.Sprintf(`<REQUEST><REQUEST_LINE><METHOD>POST</METHOD><URI>/call-%d</URI><VERSION>HTTP/1.1</VERSION></REQUEST_LINE><HEADER_FIELDS><item><NAME>Authorization</NAME><VALUE>Bearer caller</VALUE></item><item><NAME>Cookie</NAME><VALUE>stolen=1</VALUE></item></HEADER_FIELDS><MESSAGE_BODY></MESSAGE_BODY></REQUEST>`, n)
				packet, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{FunctionName: "SADT_REST_RFC_ENDPOINT", RequestedOutputs: []string{"RESPONSE"}, XrfcParameters: []cpic.NamedValue{{Name: "REQUEST", Value: []byte(xml)}}})
				if err != nil {
					t.Fatal(err)
				}
				send(t, c, data(packet))
				reply, err = receive(c)
				if err != nil {
					t.Fatal(err)
				}
				want := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("answer-%d", n)))
				if !bytes.Contains(reply, []byte(want)) {
					t.Fatalf("reply missing %s: %x", want, reply)
				}
			}
			if probes.Load() != 2 || calls.Load() != 3 {
				t.Fatalf("probes=%d calls=%d", probes.Load(), calls.Load())
			}
		})
	}
}

func TestStaticDecoderBoundsAndOwnership(t *testing.T) {
	packet := logon(t, "secret", "jco")
	auth := func(b []byte, limit int) (rfcserver.LogonIdentity, error) {
		return rfcserver.AuthenticateStaticLogon(b, "DEVELOPER", "secret", "", rfcserver.LogonIdentity{}, limit)
	}
	identity, err := auth(packet, len(packet))
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < len(packet); n++ {
		if _, err := auth(packet[:n], 0); !errors.Is(err, rfcserver.ErrAuthentication) {
			t.Fatalf("truncated %d: %v", n, err)
		}
	}
	if _, err := auth(packet, len(packet)-1); !errors.Is(err, rfcserver.ErrAuthentication) {
		t.Fatal("limit ignored")
	}
	for _, at := range []int{0, 12, 16, 18, len(packet) - 10, len(packet) - 6} {
		bad := bytes.Clone(packet)
		bad[at] ^= 0xff
		if _, err := auth(bad, 0); err == nil {
			t.Fatalf("malformed byte %d admitted", at)
		}
	}
	for i := range packet {
		packet[i] = 0
	}
	if identity.User != "DEVELOPER" || identity.Client != "001" || identity.Language != "E" {
		t.Fatal("identity aliases caller storage")
	}
}
func FuzzStaticLogon(f *testing.F) {
	f.Add(logon(f, "secret", "probe"))
	f.Add(logon(f, "secret", "jco"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		who, err := rfcserver.AuthenticateStaticLogon(b, "DEVELOPER", "secret", "001", rfcserver.LogonIdentity{}, 4096)
		if err == nil && (who.User != "DEVELOPER" || who.Client != "001" || strings.TrimSpace(who.Language) == "") {
			t.Fatal("authentication invariant")
		}
	})
}

func TestDemoRemainsDefault(t *testing.T) {
	c := session(t, Options{Backend: Backend{URL: "http://127.0.0.1"}})
	send(t, c, data(logon(t, "anything", "probe")))
	if b, err := receive(c); err != nil || len(b) != 746 {
		t.Fatalf("demo logon: %d bytes, %v", len(b), err)
	}
}

func TestStaticPasswordRange(t *testing.T) {
	for n := 1; n <= 40; n++ {
		password := strings.Repeat("x", n)
		for _, seed := range []uint32{0, 21, 0xffffffff} {
			packet := logon(t, password, "probe")
			// Replace the complete password field with the same-length encoding for
			// another seed; the chain and trailer sizes must stay valid.
			at := bytes.Index(packet, []byte{1, 0x17}) + 4
			encoded, err := scramble.ScrambleRfcPassword(password, seed)
			if err != nil {
				t.Fatal(err)
			}
			copy(packet[at:], encoded)
			if _, err := rfcserver.AuthenticateStaticLogon(packet, "DEVELOPER", password, "001", rfcserver.LogonIdentity{}, 0); err != nil {
				t.Fatalf("length %d seed %x: %v", n, seed, err)
			}
			if _, err := rfcserver.AuthenticateStaticLogon(packet, "DEVELOPER", password+"!", "001", rfcserver.LogonIdentity{}, 0); err == nil {
				t.Fatal("longer wrong password admitted")
			}
		}
	}
}

func TestStaticFragmentedLogonBudget(t *testing.T) {
	o := staticOptions("http://127.0.0.1")
	o.MaxLogonBytes = 40
	c := session(t, o)
	first := data(bytes.Repeat([]byte{1}, 20))
	first[30] = 0
	send(t, c, first)
	send(t, c, data(bytes.Repeat([]byte{2}, 21)))
	if reply, err := receive(c); err == nil {
		t.Fatalf("oversized fragmented logon received %d bytes", len(reply))
	}
}
