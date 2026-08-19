// SPDX-License-Identifier: Apache-2.0
//
// Original conformance tests for the classic RFC syntax layer ported from
// open-rfc src/protocol/classic-rfc.ts. Upstream has no isolated test file for
// it (it is exercised through the structure/xRFC and metadata layers); these
// tests state its wire facts, including the recurring-bug-class fix: an
// RFC_FUNINT row is bounded below by its 402-byte stable prefix and ignores
// appended fields. See docs/provenance.md.

package classicrfc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

func u16(s string) []byte {
	return utf16leEncode(s)
}

func TestAbapCharRoundTrip(t *testing.T) {
	enc, err := EncodeAbapChar("HELLO", 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) != 16 { // 8 chars * 2
		t.Fatalf("len = %d", len(enc))
	}
	if s, _ := DecodeAbapChar(enc); s != "HELLO" {
		t.Fatalf("decode = %q", s)
	}
	if s, _ := DecodeAbapFixedChar(enc, 8); s != "HELLO   " {
		t.Fatalf("fixed = %q", s)
	}
	if _, err := EncodeAbapChar("TOOLONG", 3); !errors.Is(err, ErrRange) {
		t.Fatalf("overlong: %v", err)
	}
	if _, err := DecodeAbapChar([]byte{0x41}); !errors.Is(err, ErrRange) {
		t.Fatalf("odd length: %v", err)
	}
	if _, err := DecodeAbapChar(u16("AB"), 3); !errors.Is(err, ErrRange) {
		t.Fatalf("wrong expected width: %v", err)
	}
}

func TestDecodeRfcTableHeader(t *testing.T) {
	h := make([]byte, 8)
	binary.BigEndian.PutUint32(h[0:], 40)
	binary.BigEndian.PutUint32(h[4:], 3)
	got, err := DecodeRfcTableHeader(h)
	if err != nil || got != (RfcTableHeader{DeclaredRowByteLength: 40, RowCount: 3}) {
		t.Fatalf("header = %+v, %v", got, err)
	}
	if _, err := DecodeRfcTableHeader(make([]byte, 7)); !errors.Is(err, ErrRange) {
		t.Fatalf("short header: %v", err)
	}
}

func buildFunintRow(class, param, table, field, exid string, position, offset, intlen, decimals int32, def, text, optional string) []byte {
	row := make([]byte, RfcFunintUnicodeRowLength)
	off := 0
	putChar := func(s string, chars int) {
		enc, err := EncodeAbapChar(s, chars)
		if err != nil {
			panic(err)
		}
		copy(row[off:], enc)
		off += chars * 2
	}
	putInt := func(v int32) {
		binary.LittleEndian.PutUint32(row[off:], uint32(v))
		off += 4
	}
	putChar(class, 1)
	putChar(param, 30)
	putChar(table, 30)
	putChar(field, 30)
	putChar(exid, 1)
	putInt(position)
	putInt(offset)
	putInt(intlen)
	putInt(decimals)
	putChar(def, 21)
	putChar(text, 79)
	putChar(optional, 1)
	return row
}

func TestDecodeFunintRowExactPrefix(t *testing.T) {
	row := buildFunintRow("I", "REQUTEXT", "", "", "C", 1, 0, 255, 0, "", "Request text", "X")
	got, err := DecodeFunintRow(row)
	if err != nil {
		t.Fatal(err)
	}
	want := FunintParameter{ParameterClass: "I", ParameterName: "REQUTEXT", TableName: "", FieldName: "", Exid: "C", Position: 1, Offset: 0, InternalLength: 255, Decimals: 0, DefaultValue: "", ParameterText: "Request text", Optional: true}
	if got != want {
		t.Fatalf("funint = %+v", got)
	}
}

func TestDecodeFunintRowIgnoresAppendedFields(t *testing.T) {
	// The recurring-bug-class fix: a 404-byte row (later release appended a
	// field) decodes to the same prefix; a 401-byte row is refused.
	row := buildFunintRow("E", "ECHOTEXT", "", "", "C", 2, 0, 255, 0, "", "", "")
	grown := append(append([]byte(nil), row...), 0x00, 0x00) // 404 bytes
	got, err := DecodeFunintRow(grown)
	if err != nil {
		t.Fatalf("404-byte row rejected: %v", err)
	}
	if got.ParameterName != "ECHOTEXT" || got.Optional {
		t.Fatalf("grown = %+v", got)
	}
	if _, err := DecodeFunintRow(row[:RfcFunintUnicodeRowLength-1]); !errors.Is(err, ErrRange) {
		t.Fatalf("short row accepted: %v", err)
	}
	// A non-X, non-empty OPTIONAL is rejected.
	bad := buildFunintRow("I", "P", "", "", "C", 1, 0, 1, 0, "", "", "Z")
	if _, err := DecodeFunintRow(bad); !errors.Is(err, ErrProtocol) {
		t.Fatalf("bad optional: %v", err)
	}
}

func fld(tag cpic.Tag, v []byte) cpic.Field { return cpic.Field{Tag: uint16(tag), Value: v} }

func tableHeaderBytes(rowLen, rowCount uint32) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint32(h[0:], rowLen)
	binary.BigEndian.PutUint32(h[4:], rowCount)
	return h
}

func TestDecodeResultScalarsAndTables(t *testing.T) {
	fields := []cpic.Field{
		fld(cpic.TagRequestedOutput, u16("RESULT")),
		fld(cpic.TagParameterName, u16("ECHOTEXT")),
		fld(cpic.TagParameterValue, u16("hello")),
		fld(cpic.TagTableName, u16("ROWS")),
		fld(cpic.TagTableHeader, tableHeaderBytes(4, 3)),
		fld(cpic.TagTableContent, []byte{1, 2, 3, 4}),
		fld(cpic.TagTableContent, []byte{5, 6, 7, 8}),
		fld(cpic.TagTableCompr, []byte{0x41, 0x42}), // expands to 41 42 42 42
		fld(cpic.TagEnd, nil),
	}
	r, err := DecodeResult(fields)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.RequestedOutputs) != 1 || r.RequestedOutputs[0] != "RESULT" {
		t.Fatalf("outputs = %v", r.RequestedOutputs)
	}
	if len(r.Scalars) != 1 || r.Scalars[0].Name != "ECHOTEXT" || !bytes.Equal(r.Scalars[0].Value, u16("hello")) {
		t.Fatalf("scalars = %+v", r.Scalars)
	}
	if len(r.Tables) != 1 {
		t.Fatalf("tables = %d", len(r.Tables))
	}
	tab := r.Tables[0]
	if tab.Name != "ROWS" || tab.DeclaredRowByteLength != 4 || tab.RowCompression != "mixed" || len(tab.Rows) != 3 {
		t.Fatalf("table = %+v", tab)
	}
	if !bytes.Equal(tab.Rows[2], []byte{0x41, 0x42, 0x42, 0x42}) {
		t.Fatalf("compressed row = %x", tab.Rows[2])
	}
}

func TestDecodeResultXrfc(t *testing.T) {
	fields := []cpic.Field{
		fld(cpic.TagXRfcParameter, nil),
		fld(cpic.TagXRfcData, []byte("<a>")),
		fld(cpic.TagXRfcData, []byte("</a>")),
		fld(cpic.TagXRfcParameter, nil),
		fld(cpic.TagEnd, nil),
	}
	r, err := DecodeResult(fields)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.XrfcParameters) != 1 || r.XrfcParameters[0].ChunkCount != 2 || string(r.XrfcParameters[0].Value) != "<a></a>" {
		t.Fatalf("xrfc = %+v", r.XrfcParameters)
	}
}

func TestDecodeResultRejects(t *testing.T) {
	cases := []struct {
		name   string
		fields []cpic.Field
		substr string
	}{
		{"value without name", []cpic.Field{fld(cpic.TagParameterValue, u16("x"))}, "value without a parameter name"},
		{"table record without name", []cpic.Field{fld(cpic.TagTableHeader, tableHeaderBytes(4, 0))}, "table record without a table name"},
		{"scalar without value", []cpic.Field{fld(cpic.TagParameterName, u16("P")), fld(cpic.TagEnd, nil)}, "not followed by its value"},
		{"duplicate parameter", []cpic.Field{fld(cpic.TagParameterName, u16("P")), fld(cpic.TagParameterValue, u16("a")), fld(cpic.TagParameterName, u16("P")), fld(cpic.TagParameterValue, u16("b"))}, "duplicate parameter"},
		{"unsupported tag", []cpic.Field{fld(cpic.Tag(0x7777), nil)}, "unsupported tag 0x7777"},
		{"xrfc data without boundary", []cpic.Field{fld(cpic.TagXRfcData, []byte("x"))}, "without an opening boundary"},
		{"row count mismatch", []cpic.Field{fld(cpic.TagTableName, u16("T")), fld(cpic.TagTableHeader, tableHeaderBytes(4, 5)), fld(cpic.TagTableContent, []byte{1, 2, 3, 4}), fld(cpic.TagEnd, nil)}, "declares 5 rows but found 1"},
	}
	for _, c := range cases {
		if _, err := DecodeResult(c.fields); err == nil || !strings.Contains(err.Error(), c.substr) {
			t.Fatalf("%s = %v, want %q", c.name, err, c.substr)
		}
	}
}

func TestDecodeOwnedResultAliases(t *testing.T) {
	val := u16("payload")
	fields := []cpic.Field{fld(cpic.TagParameterName, u16("P")), fld(cpic.TagParameterValue, val), fld(cpic.TagEnd, nil)}
	r, err := DecodeOwnedResult(fields)
	if err != nil {
		t.Fatal(err)
	}
	// Owned decode borrows: mutating the field value shows through.
	val[0] ^= 0xff
	if r.Scalars[0].Value[0] != val[0] {
		t.Fatal("owned result did not borrow")
	}
	// Non-owned decode copies: independent of caller mutation.
	fields2 := []cpic.Field{fld(cpic.TagParameterName, u16("P")), fld(cpic.TagParameterValue, u16("payload")), fld(cpic.TagEnd, nil)}
	r2, _ := DecodeResult(fields2)
	fields2[1].Value[0] ^= 0xff
	if r2.Scalars[0].Value[0] == fields2[1].Value[0] {
		t.Fatal("non-owned result aliased")
	}
}

func FuzzDecodeFunintRow(f *testing.F) {
	f.Add(buildFunintRow("I", "P", "", "", "C", 1, 0, 1, 0, "", "", ""))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeFunintRow(data)
	})
}
