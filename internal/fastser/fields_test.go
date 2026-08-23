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

// rfctestParameter is the RFCTEST parameter announcement exactly as it appears
// on the wire, recovered from a live STFC_STRUCTURE response by decompressing
// its LZ4 block. Structures only ever travel in compressed frames here, so this
// content was unreachable until DecompressBlock existed.
const rfctestParameter = "440c500d5c545950453d524643544553541308524643464c4f41540602000852464343" +
	"484152310207524643494e54320107524643494e543106080008524643434841523403" +
	"07524643494e543417030007524643484558330604000852464343484152320e075246" +
	"4354494d450c0752464344415445066400085246434441544131066400085246434441" +
	"544132"

// rfctestDDIC is what SE11 says RFCTEST is, queried from the live system. The
// wire has to agree with it field for field — this is the cross-check that
// settles both the type codes and the two width conventions.
var rfctestDDIC = []struct {
	name  string
	code  byte
	width int // 0 = the code implies the width and carries no operand
}{
	{"RFCFLOAT", TypeFloat, 0},
	{"RFCCHAR1", TypeChar, 2}, // CHAR(1), UTF-16 units
	{"RFCINT2", TypeInt2, 0},
	{"RFCINT1", TypeInt1, 0},
	{"RFCCHAR4", TypeChar, 8}, // CHAR(4)
	{"RFCINT4", TypeInt4, 0},
	{"RFCHEX3", TypeRaw, 3},   // RAW(3) — bytes, NOT doubled
	{"RFCCHAR2", TypeChar, 4}, // CHAR(2)
	{"RFCTIME", TypeTims, 0},
	{"RFCDATE", TypeDats, 0},
	{"RFCDATA1", TypeChar, 100}, // CHAR(50)
	{"RFCDATA2", TypeChar, 100}, // CHAR(50)
}

func TestRealStructureAgreesWithDDIC(t *testing.T) {
	p, _, ok := DecodeParameter(mustHex(t, rfctestParameter), 0)
	if !ok {
		t.Fatal("the captured RFCTEST announcement did not decode")
	}
	if p.TypeName != "RFCTEST" {
		t.Fatalf("TypeName = %q, want RFCTEST", p.TypeName)
	}
	if p.Generated() {
		t.Error("RFCTEST is a real DDIC type, not one the serializer synthesised")
	}
	if len(p.Fields) != len(rfctestDDIC) {
		t.Fatalf("got %d fields, want %d", len(p.Fields), len(rfctestDDIC))
	}
	for i, want := range rfctestDDIC {
		got := p.Fields[i]
		if got.Name != want.name {
			t.Errorf("field %d: name = %q, want %q", i, got.Name, want.name)
			continue
		}
		if got.TypeCode != want.code {
			t.Errorf("%s: type code = %#x, want %#x", want.name, got.TypeCode, want.code)
		}
		if want.width == 0 {
			if got.HasWidth {
				t.Errorf("%s: carries a width operand (%d); its code implies the width", want.name, got.Width)
			}
			continue
		}
		if !got.HasWidth || got.Width != want.width {
			t.Errorf("%s: width = %d (present=%v), want %d", want.name, got.Width, got.HasWidth, want.width)
		}
	}
}

func TestRawWidthCountsBytesAndCharWidthCountsUnits(t *testing.T) {
	// The pair that separates the two conventions, side by side in one structure:
	// RAW(3) travels as 3 while CHAR(1) travels as 2. A decoder that doubles
	// everything gets every hex field wrong by a factor of two.
	p, _, ok := DecodeParameter(mustHex(t, rfctestParameter), 0)
	if !ok {
		t.Fatal("did not decode")
	}
	byName := map[string]Field{}
	for _, f := range p.Fields {
		byName[f.Name] = f
	}
	if got := byName["RFCHEX3"].Width; got != 3 {
		t.Errorf("RFCHEX3 width = %d, want 3 — RAW counts bytes", got)
	}
	if got := byName["RFCCHAR1"].Width; got != 2 {
		t.Errorf("RFCCHAR1 width = %d, want 2 — CHAR counts UTF-16 units", got)
	}
}
