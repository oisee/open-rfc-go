// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfcerr"
)

func clientRequest(t *testing.T, fn string, imports []cpic.NamedValue, outputs []string) []byte {
	t.Helper()
	b, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName: fn, Imports: imports, RequestedOutputs: outputs,
	})
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	return b
}

func clientDecode(t *testing.T, response []byte) cpic.DecodedFunctionResultFields {
	t.Helper()
	decoded, err := cpic.DecodeFunctionResultFields(response)
	if err != nil {
		t.Fatalf("client decode: %v", err)
	}
	return decoded
}

func TestDispatchEcho(t *testing.T) {
	d := NewDispatcher()
	d.Handle("STFC_CONNECTION", func(ctx context.Context, req Request) (Response, error) {
		var echo []byte
		for _, imp := range req.Imports {
			if imp.Name == "REQUTEXT" {
				echo = imp.Value
			}
		}
		return Response{Exports: []cpic.NamedValue{{Name: "ECHOTEXT", Value: echo}}}, nil
	})

	reqText, _ := classicrfc.EncodeAbapChar("hello dispatcher", 40)
	req := clientRequest(t, "STFC_CONNECTION", []cpic.NamedValue{{Name: "REQUTEXT", Value: reqText}}, []string{"ECHOTEXT"})
	resp, err := d.Dispatch(context.Background(), req)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	decoded := clientDecode(t, resp)
	if !decoded.Success {
		t.Fatalf("expected success")
	}
	classic, err := classicrfc.DecodeResult(decoded.Fields)
	if err != nil {
		t.Fatalf("DecodeResult: %v", err)
	}
	for _, s := range classic.Scalars {
		if s.Name == "ECHOTEXT" {
			got, _ := classicrfc.DecodeAbapChar(s.Value)
			if got != "hello dispatcher" {
				t.Fatalf("echo = %q", got)
			}
			return
		}
	}
	t.Fatalf("no ECHOTEXT in response")
}

func TestDispatchUnknownFunction(t *testing.T) {
	d := NewDispatcher()
	resp, err := d.Dispatch(context.Background(), clientRequest(t, "Z_NOPE", nil, nil))
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	decoded := clientDecode(t, resp)
	if decoded.Success {
		t.Fatalf("unknown function should not be success")
	}
	if decoded.Envelope.Outcome != rfcerr.OutcomeAbapException || decoded.Envelope.Facts.ExceptionKey != UnknownFunctionKey {
		t.Fatalf("outcome=%v key=%q, want AbapException/%s", decoded.Envelope.Outcome, decoded.Envelope.Facts.ExceptionKey, UnknownFunctionKey)
	}
}

func TestDispatchHandlerException(t *testing.T) {
	d := NewDispatcher()
	d.Handle("Z_RAISE", func(ctx context.Context, req Request) (Response, error) {
		return Response{}, &Exception{Key: "ZCX_MY_ERROR"}
	})
	resp, err := d.Dispatch(context.Background(), clientRequest(t, "Z_RAISE", nil, nil))
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	decoded := clientDecode(t, resp)
	if decoded.Envelope.Outcome != rfcerr.OutcomeAbapException || decoded.Envelope.Facts.ExceptionKey != "ZCX_MY_ERROR" {
		t.Fatalf("outcome=%v key=%q, want AbapException/ZCX_MY_ERROR", decoded.Envelope.Outcome, decoded.Envelope.Facts.ExceptionKey)
	}
}

func TestDispatchHandlerErrorBecomesSystemFailure(t *testing.T) {
	d := NewDispatcher()
	d.Handle("Z_BOOM", func(ctx context.Context, req Request) (Response, error) {
		return Response{}, errors.New("something broke")
	})
	resp, err := d.Dispatch(context.Background(), clientRequest(t, "Z_BOOM", nil, nil))
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	decoded := clientDecode(t, resp)
	if decoded.Envelope.Facts.ExceptionKey != SystemFailureKey {
		t.Fatalf("key = %q, want %s", decoded.Envelope.Facts.ExceptionKey, SystemFailureKey)
	}
}

func TestDispatchMalformedRequestIsConnectionError(t *testing.T) {
	d := NewDispatcher()
	if _, err := d.Dispatch(context.Background(), []byte{0x00, 0x01, 0x02}); err == nil {
		t.Fatalf("expected a connection-level error for a malformed request")
	}
}
