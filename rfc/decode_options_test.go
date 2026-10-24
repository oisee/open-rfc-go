// SPDX-License-Identifier: Apache-2.0
package rfc

import (
	"context"
	"errors"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/structure"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// DecodeOptions.LenientTemporal reaches the structures and table rows of a
// call's result. Without it, a DATS value that is not YYYYMMDD fails the call.
func TestDecodeResultLenientTemporal(t *testing.T) {
	def := func(exid string) rfctypes.RfcStructureDefinition {
		return rfctypes.RfcStructureDefinition{Name: "ZTEMPORAL", ByteLength: 16, Fields: []rfctypes.RfcStructureField{
			{TableName: "ZTEMPORAL", FieldName: "DVAL", Position: 1, Offset: 0, InternalLength: 16, Exid: exid},
		}}
	}
	raw, err := structure.Encode(def("C"), map[string]any{"DVAL": "2026AB01"})
	if err != nil {
		t.Fatal(err)
	}
	iface := metadata.RfcFunctionInterface{Name: "Z_TEMPORAL", Parameters: []classicrfc.FunintParameter{
		{ParameterClass: "E", ParameterName: "ROW", TableName: "ZTEMPORAL", Exid: "u", InternalLength: 16},
		{ParameterClass: "T", ParameterName: "ROWS", TableName: "ZTEMPORAL", Exid: "h", InternalLength: 16},
	}}
	header := []byte{0, 0, 0, 16, 0, 0, 0, 1}
	fields := []cpic.Field{
		{Tag: uint16(cpic.TagParameterName), Value: utf16le("ROW")},
		{Tag: uint16(cpic.TagParameterValue), Value: raw},
		{Tag: uint16(cpic.TagTableName), Value: utf16le("ROWS")},
		{Tag: uint16(cpic.TagTableHeader), Value: header},
		{Tag: uint16(cpic.TagTableContent), Value: raw},
		{Tag: uint16(cpic.TagEnd)},
	}
	resolve := func(string) (rfctypes.RfcStructureDefinition, error) { return def("D"), nil }

	if _, err := decodeResult(iface, fields, resolve, nil, DecodeOptions{}); !errors.Is(err, ErrProtocol) {
		t.Fatalf("strict: want ErrProtocol, got %v", err)
	}
	r, err := decodeResult(iface, fields, resolve, nil, DecodeOptions{LenientTemporal: true})
	if err != nil {
		t.Fatalf("lenient: %v", err)
	}
	row, _ := r.Get("ROW").(map[string]any)
	rows := r.Table("ROWS")
	if row["DVAL"] != "2026AB01" || len(rows) != 1 || rows[0]["DVAL"] != "2026AB01" {
		t.Fatalf("lenient: ROW=%v ROWS=%v", row, rows)
	}
}

// MaxResponseFieldCount reaches the xRFC decoders as their row and cell limits.
// The node limit of a recursive parameter is raised with it, never lowered.
func TestXrfcLimits(t *testing.T) {
	c, r := xrfcLimits(DecodeOptions{})
	if c.MaxRows != nil || r.MaxRows != nil || r.MaxCells != nil || r.MaxNodes != nil {
		t.Fatalf("zero: want the decoders' defaults, got %+v %+v", c, r)
	}
	c, r = xrfcLimits(DecodeOptions{MaxResponseFieldCount: 500})
	if *c.MaxRows != 500 || *r.MaxRows != 500 || *r.MaxCells != 500 || r.MaxNodes != nil {
		t.Fatalf("500: got %+v %+v", c, r)
	}
	_, r = xrfcLimits(DecodeOptions{MaxResponseFieldCount: MaxResponseFieldCountLimit})
	if *r.MaxNodes != xrfc.AbsoluteRuntimeMaxNodes {
		t.Fatalf("limit: nodes %d, want the ceiling %d", *r.MaxNodes, xrfc.AbsoluteRuntimeMaxNodes)
	}
}

// A limit that cannot hold is refused before anything is dialled.
func TestOpenRefusesBadDecodeOptions(t *testing.T) {
	for _, n := range []int{-1, MaxResponseFieldCountLimit + 1} {
		_, err := Open(context.Background(), Destination{Host: "unreachable.invalid", User: "TESTUSER",
			Decode: DecodeOptions{MaxResponseFieldCount: n}})
		if !errors.Is(err, ErrProtocol) {
			t.Fatalf("MaxResponseFieldCount %d: want ErrProtocol, got %v", n, err)
		}
	}
}

func utf16le(s string) []byte {
	b, err := classicrfc.EncodeAbapChar(s, len(s))
	if err != nil {
		panic(err)
	}
	return b
}
