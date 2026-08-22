// SPDX-License-Identifier: Apache-2.0

package fastser

import "testing"

// greetParameter is the whole NAME parameter of a live Z_GREET request, taken
// off the wire. The caller passed 'Claude', and the serializer announced it with
// a type it generated for the call:
//
//	'D' 01                                   one field
//	'P' 22 "\TYPE=%_T00006S00000000O0000000298"
//	06 0c00                                  CHAR, declared 12 bytes
//	0a "TABLE_LINE"                          the field name
//	'C' 06 80 "Claude"                       the value, six characters
//	'E'
const greetParameter = "440150225c545950453d255f5430303030365330303030" +
	"303030304f30303030303030323938060c000a5441424c455f4c494e45430680436c61756465" +
	"45"

func TestDecodeParameterFieldList(t *testing.T) {
	payload := mustHex(t, greetParameter)
	p, next, ok := DecodeParameter(payload, 0)
	if !ok {
		t.Fatal("the parameter announcement did not decode")
	}
	if len(p.Fields) != 1 {
		t.Fatalf("got %d fields, want 1: %+v", len(p.Fields), p.Fields)
	}
	f := p.Fields[0]
	if f.TypeCode != TypeChar {
		t.Errorf("type code = %#x, want %#x (CHAR)", f.TypeCode, TypeChar)
	}
	if !f.HasWidth || f.Width != 12 {
		t.Errorf("width = %d (present=%v), want 12 — six characters, UTF-16 counted", f.Width, f.HasWidth)
	}
	if f.Name != "TABLE_LINE" {
		t.Errorf("name = %q, want TABLE_LINE", f.Name)
	}
	// The value follows the description list, and the walk must have landed on it.
	val, _, ok := decodeRecordAt(payload, next)
	if !ok || val.Tag != TagChar || string(val.Value) != "Claude" {
		t.Errorf("next record = %+v, want the char value 'Claude'", val)
	}
}

func TestGeneratedTypeIsRecognised(t *testing.T) {
	p, _, ok := DecodeParameter(mustHex(t, greetParameter), 0)
	if !ok {
		t.Fatal("did not decode")
	}
	// This is the distinction that resolved a long-running confusion: the width
	// tracks the value here only because the serializer synthesised a type sized
	// to the value. Nothing may read a programmer's declaration out of it.
	if !p.Generated() {
		t.Errorf("TypeName %q should be recognised as serializer-generated", p.TypeName)
	}
	if p.Fields[0].Width != len("Claude")*2 {
		t.Errorf("a generated type's width should track the value; got %d", p.Fields[0].Width)
	}
}

func TestFieldCountIsEnforced(t *testing.T) {
	payload := mustHex(t, greetParameter)
	// Claim two fields where one is described. The list then cannot come out,
	// and a plausible prefix must not be returned instead.
	bad := append([]byte(nil), payload...)
	bad[1] = 2
	if _, _, ok := DecodeParameter(bad, 0); ok {
		t.Error("a field count that does not come out must be refused, not truncated")
	}
	// And a count of zero describes no fields at all.
	zero := append([]byte(nil), payload...)
	zero[1] = 0
	p, _, ok := DecodeParameter(zero, 0)
	if !ok || len(p.Fields) != 0 {
		t.Errorf("zero fields should decode to an empty list, got ok=%v %+v", ok, p.Fields)
	}
}

func TestDecodeParameterRefusesNonParameters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{"empty", nil},
		{"header only", []byte{parameterHeader, 1}},
		{"header without a descriptor", []byte{parameterHeader, 1, 0x99, 0x99, 0x99}},
		{"wrong header byte", mustHex(t, "450150225c545950453d41")},
	} {
		if _, _, ok := DecodeParameter(tc.payload, 0); ok {
			t.Errorf("%s: should not decode as a parameter", tc.name)
		}
	}
}

func TestWidthOperandOnlyForWidthParameterisedCodes(t *testing.T) {
	// INT4 (0x03) takes no operand: its width follows from the code. If the
	// decoder read two bytes here it would swallow the name length and drift.
	payload := []byte{parameterHeader, 1,
		TagDescriptor, 7, '\\', 'T', 'Y', 'P', 'E', '=', 'I',
		TypeInt4, 0x0a, 'T', 'A', 'B', 'L', 'E', '_', 'L', 'I', 'N', 'E'}
	p, _, ok := DecodeParameter(payload, 0)
	if !ok {
		t.Fatal("the int4 field list did not decode")
	}
	if p.Fields[0].HasWidth {
		t.Errorf("INT4 must carry no width operand, got %d", p.Fields[0].Width)
	}
	if p.Fields[0].Name != "TABLE_LINE" {
		t.Errorf("name = %q, want TABLE_LINE", p.Fields[0].Name)
	}
}
