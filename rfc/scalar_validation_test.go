// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
)

// The four fail-open scalar paths share one shape: upstream refuses the input
// and this port used to pass it through. Each test names the upstream predicate
// it mirrors; the fixes are recorded under "Port corrections" in
// docs/provenance.md.

// INT4 is 32 bits. Upstream bounds the value at the call site
// (scalarInteger(parameter, value, -0x8000_0000, 0x7fff_ffff)) before it writes
// the four little-endian bytes, so an out-of-range input is refused rather than
// wrapped into a number the peer echoes back as success.
func TestInt4ScalarRejectsOutOfRangeValue(t *testing.T) {
	p := classicrfc.FunintParameter{ParameterName: "TEST_INT", Exid: "I", InternalLength: 4}
	for _, v := range []any{int64(3000000000), int64(2147483648), int64(-2147483649)} {
		b, err := encodeScalar(p, v)
		if err == nil {
			t.Errorf("encodeScalar(I, %d) = % x, nil; want an error", v, b)
			continue
		}
		if !errors.Is(err, ErrProtocol) {
			t.Errorf("encodeScalar(I, %d): err = %v; want errors.Is(_, ErrProtocol)", v, err)
		}
	}

	// The bounds themselves stay legal, and int32 inputs are unaffected.
	for _, tc := range []struct {
		in   any
		want int32
	}{
		{int64(2147483647), 2147483647},
		{int64(-2147483648), -2147483648},
		{int64(0), 0},
		{int64(-1294967296), -1294967296},
		{int32(-1294967296), -1294967296},
	} {
		b, err := encodeScalar(p, tc.in)
		if err != nil {
			t.Fatalf("encodeScalar(I, %d): %v", tc.in, err)
		}
		if got := int32(binary.LittleEndian.Uint32(b)); got != tc.want {
			t.Errorf("encodeScalar(I, %d) = %d; want %d", tc.in, got, tc.want)
		}
	}
}

// Upstream asserts /^\d*$/u on a NUMC before padStart, so a non-digit can never
// be zero-padded into a plausible numeral.
func TestNumcScalarRejectsNonDigits(t *testing.T) {
	p := classicrfc.FunintParameter{ParameterName: "TEST_NUMC", Exid: "N", InternalLength: 4}
	for _, s := range []string{"AB", "1A", "A1", "1 2", "-1", "1.0", "１"} {
		b, err := encodeScalar(p, s)
		if err == nil {
			t.Errorf("encodeScalar(N, %q) = % x, nil; want an error", s, b)
			continue
		}
		if !errors.Is(err, ErrProtocol) {
			t.Errorf("encodeScalar(N, %q): err = %v; want errors.Is(_, ErrProtocol)", s, err)
		}
	}

	// Digits still zero-pad, and a blank NUMC is legal: /^\d*$/u accepts the
	// empty string, which is how ABAP spells an initial numeric.
	for _, tc := range []struct{ in, want string }{
		{"42", "0042"},
		{"0000", "0000"},
		{"1", "0001"},
		{"", "0000"},
	} {
		b, err := encodeScalar(p, tc.in)
		if err != nil {
			t.Fatalf("encodeScalar(N, %q): %v", tc.in, err)
		}
		if want := utf16le(tc.want); !bytes.Equal(b, want) {
			t.Errorf("encodeScalar(N, %q) = % x; want % x", tc.in, b, want)
		}
	}
}

// Upstream calls assertNulFreeUnicodeScalarText before it appends the trailing
// NUL, because that terminator is what delimits the value on the classic wire.
func TestStringScalarRejectsNUL(t *testing.T) {
	p := classicrfc.FunintParameter{ParameterName: "TEST_STR", Exid: "g"}
	for _, s := range []string{"A\x00B", "\x00", "A\x00", "\x00A"} {
		b, err := encodeScalar(p, s)
		if err == nil {
			t.Errorf("encodeScalar(g, %q) = % x, nil; want an error", s, b)
			continue
		}
		if !errors.Is(err, ErrProtocol) {
			t.Errorf("encodeScalar(g, %q): err = %v; want errors.Is(_, ErrProtocol)", s, err)
		}
	}

	// Go's invalid UTF-8 is where upstream's isolated surrogate would hide, so
	// the shared predicate rejects it too.
	if _, err := encodeScalar(p, "A\xffB"); err == nil {
		t.Error("encodeScalar(g, invalid UTF-8): want an error")
	}

	// Text still round-trips with exactly one terminator, and the terminator is
	// counted in bytes, not appended twice.
	for _, tc := range []struct{ in, want string }{
		{"AB", "AB\x00"},
		{"", "\x00"},
		{"café", "café\x00"},
	} {
		b, err := encodeScalar(p, tc.in)
		if err != nil {
			t.Fatalf("encodeScalar(g, %q): %v", tc.in, err)
		}
		if !bytes.Equal(b, []byte(tc.want)) {
			t.Errorf("encodeScalar(g, %q) = % x; want % x", tc.in, b, []byte(tc.want))
		}
	}
}

// Upstream's decodeScalar throws on every malformed width; it has no branch that
// returns the raw bytes as a successful value. This port used to swallow the
// codec's error and hand the undecodable input back as []byte with a nil error.
func TestDecodeScalarReportsCodecRejection(t *testing.T) {
	p := classicrfc.FunintParameter{ParameterName: "TEST_DATE", Exid: "D", InternalLength: 8}
	for _, b := range [][]byte{utf16le("2026092"), {0x1, 0x2, 0x3, 0x4}, nil} {
		v, err := decodeScalar(p, b)
		if err == nil {
			t.Errorf("decodeScalar(D, % x) = %#v, nil; want an error", b, v)
			continue
		}
		if !errors.Is(err, ErrProtocol) {
			t.Errorf("decodeScalar(D, % x): err = %v; want errors.Is(_, ErrProtocol)", b, err)
		}
	}

	// The fix must not reject input the codec can read.
	got, err := decodeScalar(p, utf16le("20260928"))
	if err != nil {
		t.Fatalf("decodeScalar(D, valid): %v", err)
	}
	if got != "20260928" {
		t.Errorf("decodeScalar(D, valid) = %#v; want %q", got, "20260928")
	}
}

// The raw exids are raw by definition and must stay non-erroring, and the
// decoder must copy rather than return caller memory: a later write to the CUT
// buffer would otherwise mutate a decoded value.
func TestDecodeScalarRawPathsCopyAndDoNotError(t *testing.T) {
	for _, exid := range []string{"X", "y"} {
		p := classicrfc.FunintParameter{ParameterName: "TEST_RAW", Exid: exid, InternalLength: 4}
		in := []byte{1, 2, 3, 4}
		v, err := decodeScalar(p, in)
		if err != nil {
			t.Fatalf("decodeScalar(%s, raw): %v", exid, err)
		}
		out, ok := v.([]byte)
		if !ok {
			t.Fatalf("decodeScalar(%s) = %T; want []byte", exid, v)
		}
		if !bytes.Equal(out, in) {
			t.Fatalf("decodeScalar(%s) = % x; want % x", exid, out, in)
		}
		in[0] = 9
		if out[0] != 1 {
			t.Errorf("decodeScalar(%s) aliases the caller's buffer", exid)
		}
	}
}

// Every decoder that consumes network bytes gets a fuzz target: malformed input
// may be rejected, but it may never panic, and it may never be reported as a
// decoded value whose length contradicts its type.
func FuzzDecodeScalarNeverPanics(f *testing.F) {
	for _, s := range []struct {
		exid string
		b    []byte
	}{
		{"D", utf16le("20260928")},
		{"D", []byte{1, 2, 3, 4}},
		{"D", nil},
		{"T", utf16le("124500")},
		{"I", []byte{0, 0, 0, 0}},
		{"8", make([]byte, 8)},
		{"P", []byte{0x12, 0x3c}},
		{"a", make([]byte, 8)},
		{"g", []byte("AB\x00")},
		{"y", []byte{0xff, 0x00}},
		{"u", []byte{1, 2, 3, 4}},
		{"", []byte{1, 2, 3, 4}},
	} {
		f.Add(s.exid, s.b)
	}
	f.Fuzz(func(t *testing.T, exid string, b []byte) {
		if len(exid) > 1 {
			return // classic EXIDs are single characters
		}
		p := classicrfc.FunintParameter{ParameterName: "P", Exid: exid, InternalLength: int32(len(b))}
		v, err := decodeScalar(p, b)
		if err != nil {
			return
		}
		// A decoded temporal value must have come from its type's own width.
		switch exid {
		case "D":
			if len(b) != 16 {
				t.Fatalf("D decoded from %d bytes with a nil error: %#v", len(b), v)
			}
		case "T":
			if len(b) != 12 {
				t.Fatalf("T decoded from %d bytes with a nil error: %#v", len(b), v)
			}
		}
	})
}
