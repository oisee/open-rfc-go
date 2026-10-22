// SPDX-License-Identifier: Apache-2.0
// Adapted from open-steamgate tools/protocols/rfc-server.mjs at
// 7d8e0617a8e5ded4ead65da534acf1f1b2021863 (MIT there), ported by its
// copyright holder and contributed here under Apache-2.0.
// Modified for Go: mutex-serialized HTTP sessions and explicit close cleanup.
// See docs/provenance.md.

package rfcserver

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ADTSession owns the HTTP state of one RFC socket. Requests arriving through
// other sockets may borrow it only through a known lock of the same principal.
type ADTSession struct {
	mu      sync.Mutex
	pending sync.WaitGroup
	client  *http.Client
	csrf    csrfState
	closed  atomic.Bool
}

func NewADTSession(client *http.Client) *ADTSession {
	if client.Jar != nil {
		client.Jar = &logoffJar{CookieJar: client.Jar, pairs: make(map[string]string)}
	}
	return &ADTSession{client: client}
}

// OSG's logoff sends every cookie pair it has seen, including cookies scoped
// to an ADT path which a normal HTTP jar would omit from the logoff path.
type logoffJar struct {
	http.CookieJar
	mu    sync.Mutex
	pairs map[string]string
}

func (j *logoffJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.CookieJar.SetCookies(u, cookies)
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, cookie := range cookies {
		j.pairs[cookie.Name] = cookie.Value
	}
}
func (j *logoffJar) header() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	names := make([]string, 0, len(j.pairs))
	for name := range j.pairs {
		names = append(names, name)
	}
	sort.Strings(names)
	pairs := make([]string, 0, len(names))
	for _, name := range names {
		pairs = append(pairs, name+"="+j.pairs[name])
	}
	return strings.Join(pairs, "; ")
}

type adtPrincipal struct{ client, user string }
type editingOwner struct {
	session   *ADTSession
	principal adtPrincipal
}

// ADTSessions is scoped to one listener, matching one JS server instance.
// Its lock protects bookkeeping only; backend I/O holds the session lock.
type ADTSessions struct {
	mu      sync.Mutex
	handles map[string]editingOwner
}

func NewADTSessions() *ADTSessions { return &ADTSessions{handles: make(map[string]editingOwner)} }

func principalOf(identity LogonIdentity) adtPrincipal {
	return adtPrincipal{identity.Client, identity.User}
}

func (s *ADTSessions) selectSession(own *ADTSession, handle string, identity LogonIdentity) (*ADTSession, *ADTSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	owner, ok := s.handles[handle]
	if ok && owner.principal == principalOf(identity) && !owner.session.closed.Load() {
		owner.session.pending.Add(1)
		return owner.session, owner.session
	}
	// OSG retains the lookup for UNLOCK bookkeeping even when adoption fails.
	own.pending.Add(1)
	return own, owner.session
}

var editingHandle = regexp.MustCompile(`<LOCK_HANDLE>([a-fA-F0-9]{40})</LOCK_HANDLE>`)

func (s *ADTSessions) record(session, owner *ADTSession, identity LogonIdentity, target *url.URL, decoded any, body []byte) {
	fields, _ := decoded.(map[string]any)
	line, _ := fields["REQUEST_LINE"].(map[string]any)
	method, _ := line["METHOD"].(string)
	s.mu.Lock()
	defer s.mu.Unlock()
	query := target.Query()
	if method == http.MethodPost && query.Get("_action") == "LOCK" && !session.closed.Load() {
		if match := editingHandle.FindSubmatch(body); match != nil {
			s.handles[string(match[1])] = editingOwner{session, principalOf(identity)}
		}
	}
	if method == http.MethodPost && query.Get("_action") == "UNLOCK" && owner == session {
		delete(s.handles, query.Get("lockHandle"))
	}
	if target.Path == "/sap/public/bc/icf/logoff" {
		s.forgetLocked(session)
	}
}

func (s *ADTSessions) forgetLocked(session *ADTSession) {
	for handle, owner := range s.handles {
		if owner.session == session {
			delete(s.handles, handle)
		}
	}
}

// Disconnect immediately prevents new lock adoption. Cleanup waits for an
// in-flight request before logging off, since that request may issue cookies.
func (s *ADTSessions) Disconnect(session *ADTSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session.closed.Store(true)
	s.forgetLocked(session)
}

// End sends one best-effort logoff only for a cookie-bearing session. The
// independent deadline survives the RFC socket closing and is capped at 5 s.
func (s *ADTSession) End(backend Backend, timeout time.Duration, logf func(string, ...any)) {
	s.pending.Wait()
	s.mu.Lock()
	var status int
	var unavailable bool
	defer func() {
		s.mu.Unlock()
		if logf != nil && (status != 0 || unavailable) {
			if unavailable {
				logf("backend-logoff: unavailable")
			} else {
				logf("backend-logoff: %d", status)
			}
		}
	}()
	defer s.client.CloseIdleConnections()
	if s.client.Jar == nil {
		return
	}
	base, err := backend.origin()
	if err != nil {
		return
	}
	target := base.ResolveReference(&url.URL{Path: "/sap/public/bc/icf/logoff"})
	// Keep the Go bridge's configured subtree boundary for bridge-owned traffic.
	if err := admitBackendURL(base, target); err != nil {
		return
	}
	cookies := s.client.Jar.(*logoffJar).header()
	if cookies == "" {
		return
	}
	if timeout > 5*time.Second {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return
	}
	backend.apply(request)
	request.Header.Set("Cookie", cookies)
	client := *s.client
	client.Jar = nil
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	answer, err := client.Do(request)
	if err == nil {
		_, err = io.Copy(io.Discard, io.LimitReader(answer.Body, 8<<20+1))
		answer.Body.Close()
	}
	if err != nil {
		unavailable = true
	} else {
		status = answer.StatusCode
	}
	s.csrf.set("")
	// No request may use this session again, and dropping the jar also releases
	// cookies with paths other than the logoff path.
	s.client.Jar = nil
}
