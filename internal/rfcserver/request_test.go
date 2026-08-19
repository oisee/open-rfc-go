// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

func TestDecodeCutFunctionRequestRoundTrip(t *testing.T) {
	input := cpic.CutFunctionRequestInput{
		FunctionName:     "STFC_STRUCTURE",
		RequestedOutputs: []string{"ECHOSTRUCT", "RESPTEXT", "RFCTABLE"},
		Imports: []cpic.NamedValue{
			{Name: "IMPORTSTRUCT", Value: []byte{0x00, 0x01, 0x02, 0x03}},
			{Name: "FLAG", Value: []byte{0x58, 0x00}}, // "X" in UTF-16LE
		},
		Tables: []cpic.Table{
			{Name: "INTAB", RowByteLength: 4, Rows: [][]byte{{1, 2, 3, 4}, {5, 6, 7, 8}}},
		},
	}
	payload, err := cpic.EncodeCutFunctionRequest(input)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	req, err := DecodeCutFunctionRequest(payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if req.FunctionName != "STFC_STRUCTURE" {
		t.Fatalf("function = %q", req.FunctionName)
	}
	if !reflect.DeepEqual(req.RequestedOutputs, input.RequestedOutputs) {
		t.Fatalf("requested outputs = %v", req.RequestedOutputs)
	}
	if len(req.Imports) != 2 {
		t.Fatalf("imports = %d, want 2", len(req.Imports))
	}
	for i, imp := range req.Imports {
		if imp.Name != input.Imports[i].Name || !bytes.Equal(imp.Value, input.Imports[i].Value) {
			t.Fatalf("import %d = %q/%x, want %q/%x", i, imp.Name, imp.Value, input.Imports[i].Name, input.Imports[i].Value)
		}
	}
	if len(req.Tables) != 1 {
		t.Fatalf("tables = %d, want 1", len(req.Tables))
	}
	tbl := req.Tables[0]
	if tbl.Name != "INTAB" || tbl.RowByteLength != 4 || len(tbl.Rows) != 2 {
		t.Fatalf("table = %+v", tbl)
	}
	if !bytes.Equal(tbl.Rows[0], []byte{1, 2, 3, 4}) || !bytes.Equal(tbl.Rows[1], []byte{5, 6, 7, 8}) {
		t.Fatalf("table rows = %x", tbl.Rows)
	}
}

func TestDecodeCutFunctionRequestNoOutputs(t *testing.T) {
	payload, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName: "RFC_PING",
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	req, err := DecodeCutFunctionRequest(payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if req.FunctionName != "RFC_PING" {
		t.Fatalf("function = %q", req.FunctionName)
	}
	if len(req.Imports) != 0 || len(req.Tables) != 0 || len(req.RequestedOutputs) != 0 {
		t.Fatalf("expected an empty ping request, got %+v", req)
	}
}

func TestDecodeCutFunctionRequestRejectsBadPrefix(t *testing.T) {
	if _, err := DecodeCutFunctionRequest([]byte{0x00, 0x00, 0x00, 0x00, 0x01}); !errors.Is(err, ErrRequest) {
		t.Fatalf("expected ErrRequest for a bad prefix")
	}
	if _, err := DecodeCutFunctionRequest(nil); !errors.Is(err, ErrRequest) {
		t.Fatalf("expected ErrRequest for empty input")
	}
}

func FuzzDecodeCutFunctionRequest(f *testing.F) {
	if payload, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{FunctionName: "RFC_PING"}); err == nil {
		f.Add(payload)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeCutFunctionRequest(data)
	})
}
