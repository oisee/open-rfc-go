// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"bytes"
	"testing"
	"unicode/utf16"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
)

// A function parameter's InternalLength is a character count for C/N/D/T: the
// metadata reader halves the Unicode byte width RFC_METADATA_GET reports
// (metadata.normalizedFunctionInternalLength), because the classic codec applies
// the two-bytes-per-character factor itself. DATS is 8 characters and TIMS is 6,
// so an encoded DATE is 16 bytes and an encoded TIME is 12 — exactly the widths
// the per-field structure codec asserts. Before the fix these two cases died
// inside that assertion, with "DATE must occupy 16 Unicode bytes" and "TIME must
// occupy 12 Unicode bytes".

// utf16le is the wire form of a character field on a Unicode RFC connection.
func utf16le(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, len(units)*2)
	for _, u := range units {
		out = append(out, byte(u), byte(u>>8))
	}
	return out
}

func TestDateParameterEncodesSixteenUnicodeBytes(t *testing.T) {
	p := classicrfc.FunintParameter{ParameterName: "TEST_DATE", Exid: "D", InternalLength: 8}
	got, err := encodeScalar(p, "20260928")
	if err != nil {
		t.Fatalf("DATE reported as 8 characters: %v", err)
	}
	if want := utf16le("20260928"); !bytes.Equal(got, want) {
		t.Fatalf("DATE bytes = % x, want % x", got, want)
	}
	back, err := decodeScalar(p, got)
	if err != nil {
		t.Fatalf("decodeScalar(DATE): %v", err)
	}
	if back != "20260928" {
		t.Fatalf("DATE round-trip = %#v, want %#v", back, "20260928")
	}
}

func TestTimeParameterEncodesTwelveUnicodeBytes(t *testing.T) {
	p := classicrfc.FunintParameter{ParameterName: "TEST_TIME", Exid: "T", InternalLength: 6}
	got, err := encodeScalar(p, "124500")
	if err != nil {
		t.Fatalf("TIME reported as 6 characters: %v", err)
	}
	if want := utf16le("124500"); !bytes.Equal(got, want) {
		t.Fatalf("TIME bytes = % x, want % x", got, want)
	}
	back, err := decodeScalar(p, got)
	if err != nil {
		t.Fatalf("decodeScalar(TIME): %v", err)
	}
	if back != "124500" {
		t.Fatalf("TIME round-trip = %#v, want %#v", back, "124500")
	}
}

// The encoded width of a DATE or TIME is fixed by its type, so it must not move
// with the width a peer happens to report. A peer reporting the character width
// (8/6) and one reporting the Unicode byte width (16/12) must produce identical
// wire bytes. This is the property the fix restores: the previous code passed
// the reported width straight into the structure codec, which reads bytes.
func TestTemporalParameterWidthIsTypeFixed(t *testing.T) {
	for _, tc := range []struct{ exid, value string }{
		{"D", "20260928"},
		{"T", "124500"},
	} {
		t.Run(tc.exid, func(t *testing.T) {
			want, err := encodeScalar(classicrfc.FunintParameter{
				ParameterName:  "P",
				Exid:           tc.exid,
				InternalLength: int32(len(tc.value)),
			}, tc.value)
			if err != nil {
				t.Fatalf("encode at character width: %v", err)
			}
			for _, reported := range []int{len(tc.value), len(tc.value) * 2} {
				got, err := encodeScalar(classicrfc.FunintParameter{
					ParameterName:  "P",
					Exid:           tc.exid,
					InternalLength: int32(reported),
				}, tc.value)
				if err != nil {
					t.Fatalf("encode at reported width %d: %v", reported, err)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("reported width %d: bytes = % x, want % x", reported, got, want)
				}
			}
		})
	}
}

// Widening the accepted width is not permission to accept malformed input: the
// type's digit canonicalization still runs.
func TestTemporalParameterRejectsMalformedValue(t *testing.T) {
	for _, tc := range []struct{ exid, value string }{
		{"D", "2026-09-28"},
		{"D", "2026092"},
		{"D", "202609281"},
		{"T", "1245"},
		{"T", "12:45:00"},
		{"T", "1245000"},
	} {
		p := classicrfc.FunintParameter{ParameterName: "P", Exid: tc.exid, InternalLength: 8}
		if _, err := encodeScalar(p, tc.value); err == nil {
			t.Fatalf("%s %q: expected rejection", tc.exid, tc.value)
		}
	}
}

// An empty value stays legal: the codec writes the blank wire form, which is how
// ABAP represents an initial DATE or TIME.
func TestTemporalParameterAcceptsBlankValue(t *testing.T) {
	for _, tc := range []struct {
		exid      string
		charWidth int
		want      []byte
	}{
		{"D", 8, utf16le("        ")},
		{"T", 6, utf16le("      ")},
	} {
		p := classicrfc.FunintParameter{ParameterName: "P", Exid: tc.exid, InternalLength: int32(tc.charWidth)}
		got, err := encodeScalar(p, "")
		if err != nil {
			t.Fatalf("%s blank: %v", tc.exid, err)
		}
		if !bytes.Equal(got, tc.want) {
			t.Fatalf("%s blank bytes = % x, want % x", tc.exid, got, tc.want)
		}
	}
}
