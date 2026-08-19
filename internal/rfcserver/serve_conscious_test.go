// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
)

// TestConsciousDispatchEcho drives the conscious dispatch path end to end without
// a socket: encode an STFC_CONNECTION request, dispatch it through the default
// handlers, and decode the generated response — the echoed REQUTEXT must return.
func TestConsciousDispatchEcho(t *testing.T) {
	req, err := classicrfc.EncodeAbapChar("hello conscious", 255)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName: "STFC_CONNECTION",
		Imports:      []cpic.NamedValue{{Name: "REQUTEXT", Value: req}},
	})
	if err != nil {
		t.Fatal(err)
	}
	respPayload, err := DefaultDispatcher().Dispatch(context.Background(), payload)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	env, err := cpic.DecodeFunctionResultFields(respPayload)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !env.Success {
		t.Fatalf("expected a successful response")
	}
	res, err := classicrfc.DecodeResult(env.Fields)
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	var echo string
	for _, s := range res.Scalars {
		if s.Name == "ECHOTEXT" {
			echo, _ = classicrfc.DecodeAbapChar(s.Value)
		}
	}
	if echo != "hello conscious" {
		t.Fatalf("ECHOTEXT = %q, want %q", echo, "hello conscious")
	}
}

func TestConsciousDispatchUnknownFunction(t *testing.T) {
	payload, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{FunctionName: "NO_SUCH_FM"})
	if err != nil {
		t.Fatal(err)
	}
	respPayload, err := DefaultDispatcher().Dispatch(context.Background(), payload)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	env, err := cpic.DecodeFunctionResultFields(respPayload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Success {
		t.Fatalf("expected an exception for an unknown function")
	}
}
