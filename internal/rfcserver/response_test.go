// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
)

func TestEncodeResponseClientReadsSuccess(t *testing.T) {
	echo, _ := classicrfc.EncodeAbapChar("hi", 5)
	resp, _ := classicrfc.EncodeAbapChar("SAP", 5)
	exports := []cpic.NamedValue{
		{Name: "ECHOTEXT", Value: echo},
		{Name: "RESPTEXT", Value: resp},
	}
	tables := []Table{{Name: "RFCTABLE", RowByteLength: 4, Rows: [][]byte{{1, 2, 3, 4}}}}

	payload, err := EncodeCutFunctionResponse(exports, tables)
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}

	// The client's response decoder must read it as a success.
	decoded, err := cpic.DecodeFunctionResultFields(payload)
	if err != nil {
		t.Fatalf("client DecodeFunctionResultFields: %v", err)
	}
	if !decoded.Success {
		t.Fatalf("response did not decode as success")
	}

	classic, err := classicrfc.DecodeResult(decoded.Fields)
	if err != nil {
		t.Fatalf("DecodeResult: %v", err)
	}
	scalars := map[string][]byte{}
	for _, s := range classic.Scalars {
		scalars[s.Name] = s.Value
	}
	if !bytes.Equal(scalars["ECHOTEXT"], echo) {
		t.Fatalf("ECHOTEXT round-trip = %x, want %x", scalars["ECHOTEXT"], echo)
	}
	if !bytes.Equal(scalars["RESPTEXT"], resp) {
		t.Fatalf("RESPTEXT round-trip mismatch")
	}
	var got *classicrfc.Table
	for i := range classic.Tables {
		if classic.Tables[i].Name == "RFCTABLE" {
			got = &classic.Tables[i]
		}
	}
	if got == nil || len(got.Rows) != 1 || !bytes.Equal(got.Rows[0], []byte{1, 2, 3, 4}) {
		t.Fatalf("RFCTABLE round-trip = %+v", got)
	}
}

// TestFullCUTRoundTrip walks the whole protocol path with both sides ours: a
// client encodes a request, the server decodes it, echoes the import back as an
// export, and the client decodes the response. This is the offline in-process
// peer at the protocol level.
func TestFullCUTRoundTrip(t *testing.T) {
	reqText, _ := classicrfc.EncodeAbapChar("ping-123", 20)
	request, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName:     "STFC_CONNECTION",
		RequestedOutputs: []string{"ECHOTEXT"},
		Imports:          []cpic.NamedValue{{Name: "REQUTEXT", Value: reqText}},
	})
	if err != nil {
		t.Fatalf("client encode request: %v", err)
	}

	// Server side: decode the inbound call and dispatch (echo REQUTEXT→ECHOTEXT).
	req, err := DecodeCutFunctionRequest(request)
	if err != nil {
		t.Fatalf("server decode request: %v", err)
	}
	if req.FunctionName != "STFC_CONNECTION" || len(req.Imports) != 1 || req.Imports[0].Name != "REQUTEXT" {
		t.Fatalf("server saw unexpected request: %+v", req)
	}
	response, err := EncodeCutFunctionResponse(
		[]cpic.NamedValue{{Name: "ECHOTEXT", Value: req.Imports[0].Value}}, nil)
	if err != nil {
		t.Fatalf("server encode response: %v", err)
	}

	// Client side: decode the response and read the echo.
	decoded, err := cpic.DecodeFunctionResultFields(response)
	if err != nil || !decoded.Success {
		t.Fatalf("client decode response: %v (success=%v)", err, decoded.Success)
	}
	classic, err := classicrfc.DecodeResult(decoded.Fields)
	if err != nil {
		t.Fatalf("DecodeResult: %v", err)
	}
	for _, s := range classic.Scalars {
		if s.Name == "ECHOTEXT" {
			echoed, err := classicrfc.DecodeAbapChar(s.Value)
			if err != nil {
				t.Fatalf("decode echo: %v", err)
			}
			if echoed != "ping-123" {
				t.Fatalf("echo = %q, want ping-123", echoed)
			}
			return
		}
	}
	t.Fatalf("no ECHOTEXT in response")
}
