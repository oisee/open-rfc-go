// SPDX-License-Identifier: Apache-2.0
//
// The dispatch layer: bind decoded CUT requests to Go handlers and encode their
// responses. This is the reusable core of an RFC server — the same Dispatcher
// serves an in-process peer, a registered gateway server, and the polyglot
// bridge. Original work for open-rfc-go (milestone 8). See docs/porting-plan.md
// and docs/polyglot-rfc-server.md.

package rfcserver

import (
	"context"
	"errors"
	"sync"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

// UnknownFunctionKey is the exception key returned when no handler is registered
// for a called function.
const UnknownFunctionKey = "FU_NOT_FOUND"

// SystemFailureKey is the exception key returned when a handler fails with a
// non-Exception error.
const SystemFailureKey = "SYSTEM_FAILURE"

// Response is what a handler returns on success: export scalars and tables. The
// values are already-encoded wire bytes (use classicrfc/structure to build them).
type Response struct {
	Exports []cpic.NamedValue
	Tables  []Table
}

// Exception is a handler error that raises a declared ABAP exception with Key.
type Exception struct{ Key string }

func (e *Exception) Error() string { return "rfcserver: ABAP exception " + e.Key }

// Handler processes one decoded request. Returning an *Exception raises that
// ABAP exception to the caller; any other error becomes a SYSTEM_FAILURE.
type Handler func(ctx context.Context, req Request) (Response, error)

// Dispatcher routes decoded CUT requests to registered handlers. It is safe for
// concurrent use.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewDispatcher returns an empty dispatcher.
func NewDispatcher() *Dispatcher { return &Dispatcher{handlers: map[string]Handler{}} }

// Handle registers a handler for a function module name, replacing any previous
// handler for the same name.
func (d *Dispatcher) Handle(functionName string, h Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[functionName] = h
}

func (d *Dispatcher) handler(name string) (Handler, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	h, ok := d.handlers[name]
	return h, ok
}

// Dispatch decodes one inbound CUT request payload, invokes the registered
// handler, and returns the encoded response payload. A malformed request is a
// connection-level error (returned as err); an unknown function or a handler
// failure is returned in-band as an ABAP-exception response the caller can
// decode.
func (d *Dispatcher) Dispatch(ctx context.Context, requestPayload []byte) ([]byte, error) {
	req, err := DecodeCutFunctionRequest(requestPayload)
	if err != nil {
		return nil, err
	}
	h, ok := d.handler(req.FunctionName)
	if !ok {
		return EncodeCutFunctionExceptionResponse(UnknownFunctionKey)
	}
	resp, err := h(ctx, req)
	if err != nil {
		var exc *Exception
		if errors.As(err, &exc) && exc.Key != "" {
			return EncodeCutFunctionExceptionResponse(exc.Key)
		}
		return EncodeCutFunctionExceptionResponse(SystemFailureKey)
	}
	return EncodeCutFunctionResponse(resp.Exports, resp.Tables)
}

// Invoke runs the handler for a decoded request and returns the structured
// Response, or a non-empty exception key if the function is unknown or the
// handler failed. It lets a server encode the reply itself (e.g. with the S4
// envelope) instead of taking Dispatch's pre-encoded classic bytes.
func (d *Dispatcher) Invoke(ctx context.Context, req Request) (Response, string) {
	h, ok := d.handler(req.FunctionName)
	if !ok {
		return Response{}, UnknownFunctionKey
	}
	resp, err := h(ctx, req)
	if err != nil {
		var exc *Exception
		if errors.As(err, &exc) && exc.Key != "" {
			return Response{}, exc.Key
		}
		return Response{}, SystemFailureKey
	}
	return resp, ""
}
