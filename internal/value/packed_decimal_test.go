// SPDX-License-Identifier: Apache-2.0
//
// Original conformance vectors and round-trip/fuzz tests for the packed-decimal
// codec ported from open-rfc src/values/packed-decimal.ts. Upstream has no
// dedicated packed-decimal.test.ts (it is exercised through the xRFC layer);
// these vectors state the wire facts the port must hold. See docs/provenance.md.

package value

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestPackedDecimalKnownVectors(t *testing.T) {
	cases := []struct {
		value      string
		byteLength int
		decimals   int
		wantHex    string
	}{
		{"0", 1, 0, "0c"},
		{"123", 2, 0, "123c"},
		{"-1", 1, 0, "1d"},
		{"1.5", 2, 1, "015c"},
		{"-12.34", 3, 2, "01234d"},
		{"0.00", 2, 2, "000c"},
	}
	for _, c := range cases {
		got, err := EncodePackedDecimal(c.value, c.byteLength, c.decimals, "p")
		if err != nil {
			t.Fatalf("encode(%q, %d, %d): %v", c.value, c.byteLength, c.decimals, err)
		}
		if hex.EncodeToString(got) != c.wantHex {
			t.Fatalf("encode(%q) = %s, want %s", c.value, hex.EncodeToString(got), c.wantHex)
		}
	}
}

func TestPackedDecimalRoundTrip(t *testing.T) {
	cases := []struct {
		value      string
		byteLength int
		decimals   int
		want       string
	}{
		{"0", 1, 0, "0"},
		{"123", 2, 0, "123"},
		{"-1", 1, 0, "-1"},
		{"1.5", 2, 1, "1.5"},
		{"-12.34", 3, 2, "-12.34"},
		{"0.00", 2, 2, "0.00"},
		{"999999999999999", 8, 0, "999999999999999"},
	}
	for _, c := range cases {
		encoded, err := EncodePackedDecimal(c.value, c.byteLength, c.decimals, "p")
		if err != nil {
			t.Fatalf("encode(%q): %v", c.value, err)
		}
		decoded, err := DecodePackedDecimal(encoded, c.decimals, "p")
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if decoded != c.want {
			t.Fatalf("round-trip(%q) = %q, want %q", c.value, decoded, c.want)
		}
	}
}

func TestPackedDecimalDecodeSignNibbles(t *testing.T) {
	// A/C/E/F positive, B/D negative.
	for _, sign := range []string{"a", "c", "e", "f"} {
		v, err := DecodePackedDecimal(mustHexV(t, "1"+sign), 0, "p")
		if err != nil || v != "1" {
			t.Fatalf("positive sign %s = %q, %v", sign, v, err)
		}
	}
	for _, sign := range []string{"b", "d"} {
		v, err := DecodePackedDecimal(mustHexV(t, "1"+sign), 0, "p")
		if err != nil || v != "-1" {
			t.Fatalf("negative sign %s = %q, %v", sign, v, err)
		}
	}
	if _, err := DecodePackedDecimal(mustHexV(t, "10"), 0, "p"); !errors.Is(err, ErrPackedDecode) {
		t.Fatalf("invalid sign nibble not rejected: %v", err)
	}
	if _, err := DecodePackedDecimal(mustHexV(t, "aaac"), 0, "p"); !errors.Is(err, ErrPackedDecode) {
		t.Fatalf("non-decimal digit nibble not rejected: %v", err)
	}
}

func TestPackedDecimalRejects(t *testing.T) {
	if _, err := EncodePackedDecimal("1", 0, 0, "p"); !errors.Is(err, ErrPackedRange) {
		t.Fatalf("byteLength 0: %v", err)
	}
	if _, err := EncodePackedDecimal("1", 17, 0, "p"); !errors.Is(err, ErrPackedRange) {
		t.Fatalf("byteLength 17: %v", err)
	}
	if _, err := EncodePackedDecimal("1000", 2, 0, "p"); !errors.Is(err, ErrPackedRange) || !strings.Contains(err.Error(), "packed capacity") {
		t.Fatalf("capacity overflow: %v", err)
	}
	if _, err := EncodePackedDecimal("1.234", 3, 2, "p"); !errors.Is(err, ErrPackedRange) || !strings.Contains(err.Error(), "fractional digits") {
		t.Fatalf("too many fractional: %v", err)
	}
	if _, err := EncodePackedDecimal("notnum", 2, 0, "p"); !errors.Is(err, ErrPackedRange) {
		t.Fatalf("non-decimal: %v", err)
	}
}

// FuzzDecodePackedDecimal asserts the decoder never panics on arbitrary input
// and that anything it accepts re-encodes to a value that decodes equal.
func FuzzDecodePackedDecimal(f *testing.F) {
	f.Add([]byte{0x0c}, 0)
	f.Add([]byte{0x12, 0x3c}, 0)
	f.Add([]byte{0x01, 0x5c}, 1)
	f.Fuzz(func(t *testing.T, data []byte, decimals int) {
		s, err := DecodePackedDecimal(data, decimals, "p")
		if err != nil {
			return
		}
		reencoded, err := EncodePackedDecimal(s, len(data), decimals, "p")
		if err != nil {
			return // decode is more permissive than encode; allowed
		}
		again, err := DecodePackedDecimal(reencoded, decimals, "p")
		if err != nil || again != s {
			t.Fatalf("semantic round-trip: %q -> %q", s, again)
		}
	})
}

func mustHexV(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}
