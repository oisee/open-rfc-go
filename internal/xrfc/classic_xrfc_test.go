// SPDX-License-Identifier: Apache-2.0
//
// Oracle table tests for the classic xRFC XML codec, reproduced from
// open-rfc test/classic-xrfc.test.ts. INT8 fixtures are expressed as Go int64
// and packed-decimal fixtures as decimal strings; the wire bytes are identical
// to the upstream vectors (see the package header on the dropped modes).

package xrfc

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/rfctypes"
)

func hexb(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

func fld(table, name, exid string, pos, off, ilen, dec int32) rfctypes.RfcStructureField {
	return rfctypes.RfcStructureField{
		TableName: table, FieldName: name, Exid: exid,
		Position: pos, Offset: off, InternalLength: ilen, Decimals: dec,
	}
}

// STFC_ROW: I(INT4) C(CHAR10) STR(STRING) XSTR(XSTRING), byteLength 40.
func stfcRow() rfctypes.RfcStructureDefinition {
	const n = "STFC_ROW"
	return rfctypes.RfcStructureDefinition{Name: n, ByteLength: 40, Fields: []rfctypes.RfcStructureField{
		fld(n, "I", "I", 1, 0, 4, 0),
		fld(n, "C", "C", 2, 4, 20, 0),
		fld(n, "STR", "g", 3, 24, 8, 0),
		fld(n, "XSTR", "y", 4, 32, 8, 0),
	}}
}

// EXTENDED_ROW exercises every scalar type alongside a trailing STRING.
func extendedRow() rfctypes.RfcStructureDefinition {
	const n = "EXTENDED_ROW"
	return rfctypes.RfcStructureDefinition{Name: n, ByteLength: 66, Fields: []rfctypes.RfcStructureField{
		fld(n, "NUM", "N", 1, 0, 8, 0),
		fld(n, "DATE", "D", 2, 8, 16, 0),
		fld(n, "TIME", "T", 3, 24, 12, 0),
		fld(n, "BYTE", "X", 4, 36, 2, 0),
		fld(n, "BCD", "P", 5, 38, 4, 2),
		fld(n, "FLOAT", "F", 6, 42, 8, 0),
		fld(n, "INT8", "8", 7, 50, 8, 0),
		fld(n, "TEXT", "g", 8, 58, 8, 0),
	}}
}

func TestEncodeTableEscapingAndBase64(t *testing.T) {
	rows := []map[string]any{
		{"I": int32(42), "C": "ROW_ONE", "STR": "A<&\"-nested", "XSTR": hexb(t, "00a5ff")},
		{"I": int32(-7), "C": "ROW_TWO", "STR": "second-row", "XSTR": hexb(t, "10203040")},
	}
	got, err := EncodeParameter("IMPORT_TAB", stfcRow(), KindTable, rows, Limits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := `<IMPORT_TAB><item><I>42</I><C>ROW_ONE</C><STR>A&#60;&#38;"-nested</STR><XSTR>AKX/</XSTR></item>` +
		`<item><I>-7</I><C>ROW_TWO</C><STR>second-row</STR><XSTR>ECAwQA==</XSTR></item></IMPORT_TAB>`
	if string(got) != want {
		t.Fatalf("encode mismatch:\n got %s\nwant %s", got, want)
	}
	name, err := DecodeParameterName(got, Limits{})
	if err != nil || name != "IMPORT_TAB" {
		t.Fatalf("DecodeParameterName = %q, %v", name, err)
	}
}

func TestEncodeUnicodeAndEmptyTable(t *testing.T) {
	rows := []map[string]any{
		{"I": int32(42), "C": "UNICODE", "STR": "Grüße 🌍", "XSTR": hexb(t, "deadbeef")},
		{"I": int32(-7), "C": "EMPTY", "STR": "", "XSTR": []byte{}},
	}
	got, err := EncodeParameter("IMPORT_TAB", stfcRow(), KindTable, rows, Limits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := `<IMPORT_TAB><item><I>42</I><C>UNICODE</C><STR>Grüße 🌍</STR><XSTR>3q2+7w==</XSTR></item>` +
		`<item><I>-7</I><C>EMPTY</C><STR></STR><XSTR></XSTR></item></IMPORT_TAB>`
	if string(got) != want {
		t.Fatalf("encode mismatch:\n got %s\nwant %s", got, want)
	}
	if len(got) != 163 {
		t.Fatalf("byteLength = %d, want 163", len(got))
	}
}

func TestEncodeExtendedScalars(t *testing.T) {
	row := map[string]any{
		"NUM": "12", "DATE": "20260717", "TIME": "154530",
		"BYTE": hexb(t, "aa"), "BCD": "12.34", "FLOAT": negZero(),
		"INT8": int64(-9007199254740993), "TEXT": "ready",
	}
	got, err := EncodeParameter("ROW", extendedRow(), KindStructure, row, Limits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := `<ROW><NUM>0012</NUM><DATE>2026-07-17</DATE><TIME>15:45:30</TIME><BYTE>qgA=</BYTE>` +
		`<BCD>12.34</BCD><FLOAT>-0</FLOAT><INT8>-9007199254740993</INT8><TEXT>ready</TEXT></ROW>`
	if string(got) != want {
		t.Fatalf("encode mismatch:\n got %s\nwant %s", got, want)
	}
}

func TestEncodeBlankDateTime(t *testing.T) {
	for _, row := range []map[string]any{
		{"DATE": "", "TIME": ""},
		{"DATE": "        ", "TIME": "      "},
	} {
		got, err := EncodeParameter("ROW", extendedRow(), KindStructure, row, Limits{})
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		if !bytes.Contains(got, []byte("<DATE></DATE><TIME></TIME>")) {
			t.Fatalf("blank date/time not canonicalized: %s", got)
		}
	}
}

func TestEncodeStructureNoItem(t *testing.T) {
	row := map[string]any{"I": int32(42), "C": "ROW_ONE", "STR": "hi", "XSTR": []byte{}}
	got, err := EncodeParameter("ROW", stfcRow(), KindStructure, row, Limits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if bytes.Contains(got, []byte("<item>")) {
		t.Fatalf("structure must not wrap in <item>: %s", got)
	}
	if !bytes.HasPrefix(got, []byte("<ROW><I>42</I>")) {
		t.Fatalf("unexpected prefix: %s", got)
	}
}

func TestEncodeErrors(t *testing.T) {
	def := stfcRow()
	cases := []struct {
		name string
		v    any
		lim  Limits
	}{
		{"unknown field", []map[string]any{{"EXTRA": int32(1)}}, Limits{}},
		{"int32 overflow", []map[string]any{{"I": int64(2147483648)}}, Limits{}},
		{"char overflow", []map[string]any{{"C": "12345678901"}}, Limits{}},
		{"nul in string", []map[string]any{{"STR": "nul\x00"}}, Limits{}},
		{"control char", []map[string]any{{"STR": "control\x01"}}, Limits{}},
		{"max rows", []map[string]any{{"I": int32(1), "C": "A", "STR": "too long"}, {}}, Limits{MaxRows: ptr(1)}},
		{"max cell bytes", []map[string]any{{"STR": "too long"}}, Limits{MaxCellBytes: ptr(3)}},
		{"max row bytes", []map[string]any{{"STR": "too long"}}, Limits{MaxRowBytes: ptr(40)}},
		{"max parameter bytes", []map[string]any{{"STR": "too long"}}, Limits{MaxParameterBytes: ptr(70)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := EncodeParameter("IMPORT_TAB", def, KindTable, c.v, c.lim); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestEncodePreflightExpandsMetadata(t *testing.T) {
	numDef := rfctypes.RfcStructureDefinition{Name: "D", ByteLength: 2056, Fields: []rfctypes.RfcStructureField{
		fld("D", "NUM", "N", 1, 0, 2048, 0),
		fld("D", "TEXT", "g", 2, 2048, 8, 0),
	}}
	if _, err := EncodeParameter("P", numDef, KindStructure, map[string]any{}, Limits{MaxCellBytes: ptr(4)}); err == nil {
		t.Fatalf("expected NUM preflight error")
	}
	byteDef := rfctypes.RfcStructureDefinition{Name: "D", ByteLength: 1032, Fields: []rfctypes.RfcStructureField{
		fld("D", "BYTES", "X", 1, 0, 1024, 0),
		fld("D", "TEXT", "g", 2, 1024, 8, 0),
	}}
	if _, err := EncodeParameter("P", byteDef, KindStructure, map[string]any{}, Limits{MaxCellBytes: ptr(4)}); err == nil {
		t.Fatalf("expected BYTES preflight error")
	}
}

func TestDecodeTableNumericEntities(t *testing.T) {
	in := `<EXPORT_TAB><item><I>42</I><C>ROW_ONE</C><STR>A&#60;&#38;&#34;-nested</STR><XSTR>AKX/</XSTR></item>` +
		`<item><I>-7</I><C>ROW_TWO</C><STR>second-row</STR><XSTR>ECAwQA==</XSTR></item>` +
		`<item><I>10</I><C>Appended</C><STR>20260716</STR><XSTR>3q2+7w==</XSTR></item></EXPORT_TAB>`
	got, err := DecodeParameter("EXPORT_TAB", stfcRow(), KindTable, []byte(in), Limits{})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := []map[string]any{
		{"I": int32(42), "C": "ROW_ONE", "STR": "A<&\"-nested", "XSTR": hexb(t, "00a5ff")},
		{"I": int32(-7), "C": "ROW_TWO", "STR": "second-row", "XSTR": hexb(t, "10203040")},
		{"I": int32(10), "C": "Appended", "STR": "20260716", "XSTR": hexb(t, "deadbeef")},
	}
	if !reflect.DeepEqual(got, any(want)) {
		t.Fatalf("decode mismatch:\n got %#v\nwant %#v", got, want)
	}
}

func TestDecodeExtendedScalars(t *testing.T) {
	in := `<ROW><NUM>0012</NUM><DATE>2026-07-17</DATE><TIME>15:45:30</TIME><BYTE>qgA=</BYTE>` +
		`<BCD>12.34</BCD><FLOAT>-0</FLOAT><INT8>-9007199254740993</INT8><TEXT>ready</TEXT></ROW>`
	got, err := DecodeParameter("ROW", extendedRow(), KindStructure, []byte(in), Limits{})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	m := got.(map[string]any)
	if m["NUM"] != "0012" || m["DATE"] != "20260717" || m["TIME"] != "154530" ||
		m["BCD"] != "12.34" || m["INT8"] != int64(-9007199254740993) || m["TEXT"] != "ready" {
		t.Fatalf("scalar mismatch: %#v", m)
	}
	if !bytes.Equal(m["BYTE"].([]byte), hexb(t, "aa00")) {
		t.Fatalf("BYTE = %x", m["BYTE"])
	}
	f, ok := m["FLOAT"].(float64)
	if !ok || !isNegZero(f) {
		t.Fatalf("FLOAT should be -0, got %v", m["FLOAT"])
	}
}

func TestDecodeFloatLexical(t *testing.T) {
	accept := map[string]float64{
		"1.5": 1.5, "+1.5": 1.5, "01.5": 1.5, "1.": 1, ".5": 0.5,
		"-2": -2, "1e3": 1000, "+1.5E+02": 150, "0": 0,
	}
	for lex, exp := range accept {
		in := "<ROW><NUM></NUM><DATE></DATE><TIME></TIME><BYTE>qgA=</BYTE><BCD>0.00</BCD><FLOAT>" +
			lex + "</FLOAT><INT8>0</INT8><TEXT></TEXT></ROW>"
		got, err := DecodeParameter("ROW", extendedRow(), KindStructure, []byte(in), Limits{})
		if err != nil {
			t.Fatalf("float %q: %v", lex, err)
		}
		if got.(map[string]any)["FLOAT"].(float64) != exp {
			t.Fatalf("float %q = %v, want %v", lex, got.(map[string]any)["FLOAT"], exp)
		}
	}
	for _, lex := range []string{"", ".", "+", "1.5.5", "0x10", "1e", "NaN", "Infinity", " 1"} {
		in := "<ROW><NUM></NUM><DATE></DATE><TIME></TIME><BYTE>qgA=</BYTE><BCD>0.00</BCD><FLOAT>" +
			lex + "</FLOAT><INT8>0</INT8><TEXT></TEXT></ROW>"
		if _, err := DecodeParameter("ROW", extendedRow(), KindStructure, []byte(in), Limits{}); err == nil {
			t.Fatalf("float %q should be rejected", lex)
		}
	}
}

func TestDecodeErrors(t *testing.T) {
	def := stfcRow()
	const valid = `<I>1</I><C>A</C><STR>x</STR><XSTR>AA==</XSTR>`
	cases := map[string][]byte{
		"xml prolog":          []byte(`<?xml version="1.0"?><EXPORT_TAB></EXPORT_TAB>`),
		"attributes":          []byte(`<EXPORT_TAB kind="x"></EXPORT_TAB>`),
		"field order":         []byte(`<EXPORT_TAB><item><C>A</C><I>1</I><STR>x</STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"cdata close":         []byte(`<EXPORT_TAB><item><I>1</I><C>A</C><STR>]]></STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"codepoint nul":       []byte(`<EXPORT_TAB><item><I>1</I><C>A</C><STR>&#x0;</STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"codepoint control":   []byte(`<EXPORT_TAB><item><I>1</I><C>A</C><STR>&#x1;</STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"surrogate":           []byte(`<EXPORT_TAB><item><I>1</I><C>A</C><STR>&#xD800;</STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"above max codepoint": []byte(`<EXPORT_TAB><item><I>1</I><C>A</C><STR>&#x110000;</STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"unknown entity":      []byte(`<EXPORT_TAB><item><I>1</I><C>A</C><STR>&nbsp;</STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"capital hex marker":  []byte(`<EXPORT_TAB><item><I>1</I><C>A</C><STR>&#X41;</STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"missing close":       []byte(`<EXPORT_TAB><item>` + valid + `</item>`),
		"trailing content":    []byte(`<EXPORT_TAB><item>` + valid + `</item></EXPORT_TAB>tail`),
		"unwrapped rows":      []byte(`<EXPORT_TAB>` + valid + `</EXPORT_TAB>`),
		"non-canonical int4":  []byte(`<EXPORT_TAB><item><I>01</I><C>A</C><STR>x</STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"int4 range":          []byte(`<EXPORT_TAB><item><I>2147483648</I><C>A</C><STR>x</STR><XSTR>AA==</XSTR></item></EXPORT_TAB>`),
		"non-canonical b64":   []byte(`<EXPORT_TAB><item><I>1</I><C>A</C><STR>x</STR><XSTR>AB==</XSTR></item></EXPORT_TAB>`),
		"b64 length":          []byte(`<EXPORT_TAB><item><I>1</I><C>A</C><STR>x</STR><XSTR>A</XSTR></item></EXPORT_TAB>`),
		"utf8 bom":            {0xef, 0xbb, 0xbf, 0x3c, 0x45, 0x3e},
		"invalid utf8":        {0x3c, 0x45, 0x3e, 0xc3, 0x28},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeParameter("EXPORT_TAB", def, KindTable, in, Limits{}); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestDecodeParameterBounds(t *testing.T) {
	rows := []map[string]any{
		{"I": int32(1), "C": "A", "STR": "too long", "XSTR": []byte{}},
		{"I": int32(2), "C": "B", "STR": "row two!!", "XSTR": []byte{}},
	}
	encoded, err := EncodeParameter("EXPORT_TAB", stfcRow(), KindTable, rows, Limits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	for _, lim := range []Limits{
		{MaxParameterBytes: ptr(len(encoded) - 1)},
		{MaxRows: ptr(1)},
		{MaxCellBytes: ptr(3)},
		{MaxRowBytes: ptr(40)},
	} {
		if _, err := DecodeParameter("EXPORT_TAB", stfcRow(), KindTable, encoded, lim); err == nil {
			t.Fatalf("expected bound error for %+v", lim)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	rows := []map[string]any{
		{},
		{"I": int32(1), "C": "😀", "STR": "Grüße 😀 é", "XSTR": hexb(t, "00ff102080")},
	}
	encoded, err := EncodeParameter("T", stfcRow(), KindTable, rows, Limits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeParameter("T", stfcRow(), KindTable, encoded, Limits{})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := []map[string]any{
		{"I": int32(0), "C": "", "STR": "", "XSTR": []byte{}},
		{"I": int32(1), "C": "😀", "STR": "Grüße 😀 é", "XSTR": hexb(t, "00ff102080")},
	}
	if !reflect.DeepEqual(decoded, any(want)) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", decoded, want)
	}
}

func FuzzDecodeParameter(f *testing.F) {
	def := stfcRow()
	f.Add([]byte(`<T><item><I>1</I><C>A</C><STR>x</STR><XSTR>AA==</XSTR></item></T>`))
	f.Add([]byte(`<T><I>1</I></T>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		// Must never panic on arbitrary input.
		_, _ = DecodeParameter("T", def, KindTable, data, Limits{})
		_, _ = DecodeParameter("T", def, KindStructure, data, Limits{})
		_, _ = DecodeParameterName(data, Limits{})
	})
}

// --- helpers ---------------------------------------------------------------

func ptr(n int) *int { return &n }

func negZero() float64 { return math.Copysign(0, -1) }

func isNegZero(f float64) bool { return f == 0 && math.Signbit(f) }

// TestDecodeBase64LineWrapped covers the same server behaviour on the flat
// classic xRFC codec: an XSTRING cell arrives MIME-wrapped at 76 columns.
func TestDecodeBase64LineWrapped(t *testing.T) {
	var b strings.Builder
	payload := bytes.Repeat([]byte{0xde, 0xad, 0xbe, 0xef}, 64) // 256 bytes
	encoded := base64.StdEncoding.EncodeToString(payload)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(encoded[i:end])
	}
	got, err := DecodeBase64(b.String(), "X", 1<<20)
	if err != nil {
		t.Fatalf("wrapped: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("wrapped decode mismatch")
	}
	if _, err := DecodeBase64(strings.ReplaceAll(b.String(), "\n", "\r\n"), "X", 1<<20); err != nil {
		t.Fatalf("crlf wrapped: %v", err)
	}
	if _, err := DecodeBase64(strings.ReplaceAll(b.String(), "\n", " "), "X", 1<<20); err == nil {
		t.Fatalf("space-separated base64 must stay rejected")
	}
}
