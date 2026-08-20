// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oisee/open-rfc-go/internal/lifecycle"
	"github.com/oisee/open-rfc-go/internal/pool"
	"github.com/oisee/open-rfc-go/internal/rfcerr"
	"github.com/oisee/open-rfc-go/internal/transport"
)

// The error taxonomy for the public package. Wrap-and-match with errors.Is;
// an ABAP failure is an *ABAPException, matched with errors.As.
var (
	// ErrClosed reports use of a client or session that has been closed.
	ErrClosed = errors.New("rfc: closed")
	// ErrNotAuthenticated reports a call before a successful logon.
	ErrNotAuthenticated = errors.New("rfc: not authenticated")
	// ErrLogonRejected reports that the system refused the logon credentials.
	ErrLogonRejected = errors.New("rfc: logon rejected")
	// ErrProtocol reports a malformed or unexpected server exchange.
	ErrProtocol = errors.New("rfc: protocol error")
	// ErrTransport reports a connection-level failure.
	ErrTransport = errors.New("rfc: transport error")
	// ErrUnknownParameter reports a parameter absent from the function interface.
	ErrUnknownParameter = errors.New("rfc: unknown parameter")
	// ErrPoolExhausted reports that no pooled connection became available before
	// the acquire deadline.
	ErrPoolExhausted = errors.New("rfc: connection pool exhausted")
	// ErrTimeout reports that the context deadline elapsed.
	ErrTimeout = errors.New("rfc: timeout")
)

// translate maps an internal package's sentinel onto the public taxonomy so
// callers can match with errors.Is against this package's errors alone. The
// original error stays wrapped for diagnostics.
func translate(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pool.ErrPoolExhausted):
		return fmt.Errorf("%w: %v", ErrPoolExhausted, err)
	case errors.Is(err, pool.ErrClosed), errors.Is(err, lifecycle.ErrClosed), errors.Is(err, transport.ErrClosed):
		return fmt.Errorf("%w: %v", ErrClosed, err)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	case errors.Is(err, context.Canceled):
		return err
	}
	return err
}

// ExceptionKind classifies an ABAP-side failure.
type ExceptionKind string

const (
	// KindException is a declared RAISING/EXCEPTIONS exception.
	KindException ExceptionKind = "exception"
	// KindRuntime is an untrapped ABAP runtime error (short dump).
	KindRuntime ExceptionKind = "runtime"
	// KindMessage is a MESSAGE ... RAISING carried as a T100 message.
	KindMessage ExceptionKind = "message"
)

// ABAPException is a typed ABAP-side failure returned by a call. It is an
// error; recover it with errors.As.
type ABAPException struct {
	Kind          ExceptionKind
	Key           string // exception key / name
	PlainText     string
	RuntimeID     string
	T100Text      string
	MessageClass  string
	MessageType   string
	MessageNumber string
	MessageV1     string
	MessageV2     string
	MessageV3     string
	MessageV4     string
}

func (e *ABAPException) Error() string {
	var b strings.Builder
	b.WriteString("rfc: ABAP ")
	b.WriteString(string(e.Kind))
	if e.Key != "" {
		fmt.Fprintf(&b, " %q", e.Key)
	}
	switch {
	case e.T100Text != "":
		fmt.Fprintf(&b, ": %s", e.T100Text)
	case e.PlainText != "":
		fmt.Fprintf(&b, ": %s", e.PlainText)
	case e.MessageClass != "":
		fmt.Fprintf(&b, ": message %s%s %s", e.MessageClass, e.MessageNumber, e.MessageType)
	case e.RuntimeID != "":
		fmt.Fprintf(&b, ": %s", e.RuntimeID)
	}
	return b.String()
}

func kindFromOutcome(o rfcerr.Outcome) ExceptionKind {
	switch o {
	case rfcerr.OutcomeAbapRuntime:
		return KindRuntime
	case rfcerr.OutcomeAbapMessage:
		return KindMessage
	default:
		return KindException
	}
}

// exceptionFromEnvelope returns a typed exception for a non-success envelope,
// or nil when the call succeeded.
func exceptionFromEnvelope(env rfcerr.Envelope) *ABAPException {
	if env.Outcome == rfcerr.OutcomeSuccess {
		return nil
	}
	f := env.Facts
	return &ABAPException{
		Kind:          kindFromOutcome(env.Outcome),
		Key:           f.ExceptionKey,
		PlainText:     f.PlainText,
		RuntimeID:     f.RuntimeID,
		T100Text:      f.T100Text,
		MessageClass:  f.MessageClass,
		MessageType:   f.MessageType,
		MessageNumber: f.MessageNumber,
		MessageV1:     f.MessageV1,
		MessageV2:     f.MessageV2,
		MessageV3:     f.MessageV3,
		MessageV4:     f.MessageV4,
	}
}
