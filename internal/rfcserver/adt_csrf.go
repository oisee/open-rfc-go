// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// The CSRF dance, which over RFC is the bridge's to do.
//
// ADT refuses a modifying request that carries no X-CSRF-Token: the answer is
// 403 with "X-CSRF-Token: Required" and a body saying validation failed. A
// client normally asks for a token, keeps it beside its session cookie, and
// sends it back.
//
// This file exists because the handler used to assume Eclipse does that here.
// It does not, and cannot: over RFC there is no HTTP session between Eclipse
// and anything — it hands the bridge a request and expects the far side to be
// somebody already. Measured against a live A4H on 2026-09-15: every GET of
// the project-open sequence answered 200, and the first POST, of
// /sap/bc/adt/repository/typestructure, came back 403 "X-CSRF-Token: Required"
// with no token anywhere in the exchange. The bridge terminates HTTP and
// re-originates it, so the session the token belongs to is the bridge's, and
// so is the dance.
//
// Read off vsp pkg/adt/http.go, which has this worked out against real systems,
// and narrowed on the way: vsp refreshes on any 403, which is right for a tool
// driving a whole session, whereas here a 403 that is not a CSRF refusal is the
// backend's answer and belongs to Eclipse unaltered.

// csrfProbePath is where a token is asked for. Any ADT resource will mint one;
// discovery is the cheapest and is always there.
const csrfProbePath = "/sap/bc/adt/core/discovery"

// csrfState is one connection's token. A conversation is one caller, and the
// token belongs to the session its cookie jar holds, so it lives beside that
// jar rather than in the process.
type csrfState struct {
	mu    sync.Mutex
	token string
}

func (c *csrfState) get() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token
}

func (c *csrfState) set(v string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = v
}

// isCSRFToken reports whether a header value is a token rather than the
// server's way of saying it wants one. "Required" arrives in the same header as
// a real token and is not one; storing it produces a second 403 that looks like
// the first and is not.
func isCSRFToken(v string) bool { return v != "" && v != "Required" }

// isModifyingMethod reports whether ADT will want a token for this method.
func isModifyingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	default:
		return false
	}
}

// fetchCSRFToken asks the backend for a token on this handler's own session,
// so that the token and the cookies it is paired with belong together.
//
// HEAD first, then GET. Some systems answer the HEAD with no token at all —
// vsp carries the scars — and a missing token there is not a failure, only a
// reason to ask the other way. The response body is drained and closed so the
// connection can be reused.
//
// A backend that mints no token at all is not an error either: it is a backend
// with no CSRF protection, which our own façade is, and the request goes on
// without the header. Only a backend that cannot be reached is an error. If
// such a backend does want a token after all it says so, and the caller's
// retry turns that into the 403 Eclipse should see rather than a loop.
func fetchCSRFToken(ctx context.Context, client *http.Client, base *url.URL, backend Backend) (string, error) {
	ref, err := url.Parse(csrfProbePath)
	if err != nil {
		return "", err
	}
	target := base.ResolveReference(ref).String()
	var lastStatus int
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		probe, err := http.NewRequestWithContext(ctx, method, target, nil)
		if err != nil {
			return "", fmt.Errorf("rfcserver: CSRF probe: %w", err)
		}
		probe.Header.Set("X-CSRF-Token", "fetch")
		probe.Header.Set("Accept", "*/*")
		backend.apply(probe)
		answer, err := client.Do(probe)
		if err != nil {
			return "", fmt.Errorf("rfcserver: CSRF probe: %w", err)
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(answer.Body, 1<<20))
		answer.Body.Close()
		lastStatus = answer.StatusCode
		if token := answer.Header.Get("X-CSRF-Token"); isCSRFToken(token) {
			return token, nil
		}
	}
	_ = lastStatus
	return "", nil
}

// wantsCSRFToken reports whether an answer is the backend refusing for want of
// a token, rather than refusing on its own merits. Only this shape is worth a
// retry: any other 403 is the backend's answer and belongs to the caller as it
// stands.
func wantsCSRFToken(answer *http.Response) bool {
	return answer.StatusCode == http.StatusForbidden && answer.Header.Get("X-CSRF-Token") == "Required"
}
