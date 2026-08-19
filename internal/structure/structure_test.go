// SPDX-License-Identifier: Apache-2.0
//
// Original conformance tests for the classic-structure codec ported from
// open-rfc src/values/classic-structure.ts. Upstream has no isolated test
// (it is exercised via classic-xRFC); these pin the per-type encode/decode
// round-trip, initial values, and the geometry/dynamic-field/unknown-field
// rejections. See docs/provenance.md.

package structure

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/oisee/open-rfc-go/internal/rfctypes"
)

func fld(name, exid string, pos, offset, intlen, decimals int32) rfctypes.RfcStructureField {
	return rfctypes.RfcStructureField{TableName: "ZTEST", FieldName: name, Position: pos, Offset: offset, InternalLength: intlen, Decimals: decimals, Exid: exid}
}

func flatDef() rfctypes.RfcStructureDefinition {
	return rfctypes.RfcStructureDefinition{
		Name:       "ZTEST",
		ByteLength: 46,
		Fields: []rfctypes.RfcStructureField{
			fld("TEXT", "C", 1, 0, 8, 0),    // CHAR(4)
			fld("NUM", "N", 2, 8, 6, 0),     // NUMC(3)
			fld("COUNT", "I", 3, 14, 4, 0),  // INT4
			fld("SHORT", "s", 4, 18, 2, 0),  // INT2
			fld("BYTE", "b", 5, 20, 1, 0),   // INT1
			fld("PAD", "X", 6, 21, 1, 0),    // 1 byte to realign
			fld("AMOUNT", "P", 7, 22, 4, 2), // packed, 2 decimals
			fld("DVAL", "D", 8, 26, 16, 0),  // DATE
			fld("RAW", "X", 9, 42, 4, 0),    // RAW(4)
		},
	}
}

func TestStructureRoundTrip(t *testing.T) {
	def := flatDef()
	input := map[string]any{
		"TEXT":   "hi",
		"NUM":    "42",
		"COUNT":  int32(1000),
		"SHORT":  int16(-7),
		"BYTE":   uint8(255),
		"PAD":    []byte{0},
		"AMOUNT": "12.34",
		"DVAL":   "20260716",
		"RAW":    []byte{1, 2, 3, 4},
	}
	enc, err := Encode(def, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) != 46 {
		t.Fatalf("len = %d", len(enc))
	}
	out, err := Decode(def, enc)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"TEXT":   "hi",
		"NUM":    "042", // N pads with leading zeros to width 3
		"COUNT":  int32(1000),
		"SHORT":  int16(-7),
		"BYTE":   uint8(255),
		"PAD":    []byte{0},
		"AMOUNT": "12.34",
		"DVAL":   "20260716",
		"RAW":    []byte{1, 2, 3, 4},
	}
	for k, w := range want {
		if !reflect.DeepEqual(out[k], w) {
			t.Fatalf("field %s = %#v (%T), want %#v (%T)", k, out[k], out[k], w, w)
		}
	}
}

func TestStructureInitialValues(t *testing.T) {
	def := flatDef()
	enc, err := Encode(def, map[string]any{}) // all fields default
	if err != nil {
		t.Fatal(err)
	}
	out, err := Decode(def, enc)
	if err != nil {
		t.Fatal(err)
	}
	if out["TEXT"] != "" || out["NUM"] != "000" || out["COUNT"].(int32) != 0 || out["AMOUNT"] != "0.00" || out["DVAL"] != "00000000" {
		t.Fatalf("initials = %+v", out)
	}
	if !bytes.Equal(out["RAW"].([]byte), []byte{0, 0, 0, 0}) {
		t.Fatalf("RAW initial = %x", out["RAW"])
	}
}

func TestStructureInt8AndFloatAndTemporal(t *testing.T) {
	def := rfctypes.RfcStructureDefinition{
		Name:       "ZNUM",
		ByteLength: 24,
		Fields: []rfctypes.RfcStructureField{
			{TableName: "ZNUM", FieldName: "BIG", Position: 1, Offset: 0, InternalLength: 8, Exid: "8"},
			{TableName: "ZNUM", FieldName: "F", Position: 2, Offset: 8, InternalLength: 8, Exid: "F"},
			{TableName: "ZNUM", FieldName: "TS", Position: 3, Offset: 16, InternalLength: 8, Exid: "n"}, // UTCSECOND
		},
	}
	enc, err := Encode(def, map[string]any{"BIG": int64(-9000000000), "F": 3.5, "TS": "2002-02-04T20:15:01"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := Decode(def, enc)
	if err != nil {
		t.Fatal(err)
	}
	if out["BIG"].(int64) != -9000000000 || out["F"].(float64) != 3.5 || out["TS"].(string) != "2002-02-04T20:15:01" {
		t.Fatalf("out = %+v", out)
	}
}

func TestStructureRejects(t *testing.T) {
	def := flatDef()
	if _, err := Encode(def, map[string]any{"UNKNOWN": "x"}); !errors.Is(err, ErrStructure) {
		t.Fatalf("unknown field: %v", err)
	}
	if _, err := Encode(def, map[string]any{"COUNT": "notint"}); !errors.Is(err, ErrStructure) {
		t.Fatalf("wrong type: %v", err)
	}
	if _, err := Decode(def, make([]byte, 10)); !errors.Is(err, ErrStructure) {
		t.Fatalf("byte length mismatch: %v", err)
	}
	// STRING/XSTRING (g/y) require the xRFC serializer.
	dynamic := rfctypes.RfcStructureDefinition{Name: "ZD", ByteLength: 8, Fields: []rfctypes.RfcStructureField{{TableName: "ZD", FieldName: "S", Position: 1, Offset: 0, InternalLength: 8, Exid: "g"}}}
	if _, err := Encode(dynamic, map[string]any{}); err == nil {
		t.Fatal("dynamic field accepted")
	}
	// Bad geometry: field position out of order.
	bad := rfctypes.RfcStructureDefinition{Name: "ZB", ByteLength: 8, Fields: []rfctypes.RfcStructureField{{TableName: "ZB", FieldName: "A", Position: 2, Offset: 0, InternalLength: 8, Exid: "C"}}}
	if _, err := Encode(bad, map[string]any{}); !errors.Is(err, ErrStructure) {
		t.Fatalf("bad geometry: %v", err)
	}
}

func TestStructureFixedCharKeepsSpacesInData(t *testing.T) {
	// A CHAR field round-trips trimmed trailing spaces (decodeAbapChar).
	def := rfctypes.RfcStructureDefinition{Name: "ZC", ByteLength: 20, Fields: []rfctypes.RfcStructureField{
		{TableName: "ZC", FieldName: "A", Position: 1, Offset: 0, InternalLength: 20, Exid: "C"},
	}}
	enc, err := Encode(def, map[string]any{"A": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	out, _ := Decode(def, enc)
	if out["A"] != "hello" {
		t.Fatalf("A = %q", out["A"])
	}
}

func FuzzDecodeStructure(f *testing.F) {
	def := flatDef()
	seed, _ := Encode(def, map[string]any{})
	f.Add(seed)
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Decode(def, data)
	})
}
