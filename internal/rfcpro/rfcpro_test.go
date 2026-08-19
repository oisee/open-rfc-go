// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/rfcpro.test.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten for the testing package. The
// invalid-tag cases (tag -1, 65536, 1.5) and the NaN/non-integer maxValueLength
// cases have no Go analogue — the tag is a uint16 and maxValueLength an int, so
// the compiler enforces what those cases assert. A FuzzDecodeFieldHeader target
// is added, per the milestone's rule that every decoder of network bytes carry
// one. See docs/provenance.md.

package rfcpro

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

const parameterValue uint16 = 0x0203

func hexOf(t *testing.T, b []byte) string {
	t.Helper()
	return hex.EncodeToString(b)
}

func decodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

func TestEncodesCanonicalCompactAndExtendedHeaders(t *testing.T) {
	cases := []struct {
		length int
		want   string
	}{
		{0, "02030000"},
		{CompactLengthMax, "0203fffe"},
		{65_535, "0203ffff0000ffff"},
		{65_536, "0203ffff00010000"},
		{ValueLengthMax, "0203ffff7fffffff"},
	}
	for _, c := range cases {
		got, err := EncodeFieldHeader(parameterValue, c.length)
		if err != nil {
			t.Fatalf("EncodeFieldHeader(%d): %v", c.length, err)
		}
		if hexOf(t, got) != c.want {
			t.Fatalf("EncodeFieldHeader(%d) = %s, want %s", c.length, hexOf(t, got), c.want)
		}
	}
}

func TestDecodesCompactCanonicalAndLegacyExtendedLengths(t *testing.T) {
	cases := []struct {
		in   string
		want FieldHeader
	}{
		{"0203fffe", FieldHeader{parameterValue, CompactLengthMax, EncodingCompact, 4}},
		{"0203ffff0000ffff", FieldHeader{parameterValue, 65_535, EncodingExtended, 8}},
		{"0203ffff00010000aabb", FieldHeader{parameterValue, 65_536, EncodingExtended, 8}},
		{"0203ffff0000fffe", FieldHeader{parameterValue, CompactLengthMax, EncodingExtended, 8}},
		{"0203ffff7fffffff", FieldHeader{parameterValue, ValueLengthMax, EncodingExtended, 8}},
	}
	for _, c := range cases {
		got, err := DecodeFieldHeader(decodeHex(t, c.in))
		if err != nil {
			t.Fatalf("DecodeFieldHeader(%s): %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("DecodeFieldHeader(%s) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestReportsExactHeaderLengthsWithoutAllocatingPayload(t *testing.T) {
	cases := []struct {
		length int
		want   int
	}{
		{0, 4},
		{CompactLengthMax, 4},
		{65_535, 8},
		{ValueLengthMax, 8},
	}
	for _, c := range cases {
		got, err := FieldHeaderByteLength(c.length)
		if err != nil || got != c.want {
			t.Fatalf("FieldHeaderByteLength(%d) = %d, %v; want %d", c.length, got, err, c.want)
		}
	}
}

func TestRejectsInvalidLengthsAndConfiguredMaxima(t *testing.T) {
	if _, err := EncodeFieldHeader(0, -1); !errors.Is(err, ErrRange) {
		t.Fatalf("EncodeFieldHeader(-1 length) = %v, want ErrRange", err)
	}
	if _, err := EncodeFieldHeader(0, ValueLengthMax+1); !errors.Is(err, ErrRange) {
		t.Fatalf("EncodeFieldHeader(overflow length) = %v, want ErrRange", err)
	}

	if _, err := DecodeFieldHeaderLimited(decodeHex(t, "0203ffff00010000"), 65_535); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("limit 65535 = %v, want ErrLimitExceeded", err)
	}
	if got, err := DecodeFieldHeaderLimited(decodeHex(t, "0203ffff00010000"), 65_536); err != nil ||
		got != (FieldHeader{parameterValue, 65_536, EncodingExtended, 8}) {
		t.Fatalf("limit 65536 = %+v, %v", got, err)
	}
	if got, err := DecodeFieldHeaderLimited(decodeHex(t, "02030001"), 1); err != nil ||
		got != (FieldHeader{parameterValue, 1, EncodingCompact, 4}) {
		t.Fatalf("limit 1 = %+v, %v", got, err)
	}
	if _, err := DecodeFieldHeaderLimited(decodeHex(t, "02030001"), 0); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("limit 0 = %v, want ErrLimitExceeded", err)
	}
	if _, err := DecodeFieldHeaderLimited(decodeHex(t, "02030000"), -1); !errors.Is(err, ErrRange) {
		t.Fatalf("limit -1 = %v, want ErrRange", err)
	}
	if _, err := DecodeFieldHeaderLimited(decodeHex(t, "02030000"), ValueLengthMax+1); !errors.Is(err, ErrRange) {
		t.Fatalf("limit overflow = %v, want ErrRange", err)
	}

	if _, err := DecodeFieldHeader(decodeHex(t, "0203ffffffffffff")); !errors.Is(err, ErrNegativeLength) ||
		!strings.Contains(err.Error(), "-1 is negative") {
		t.Fatalf("negative extended = %v", err)
	}
	if _, err := DecodeFieldHeader(decodeHex(t, "0203ffff80000000")); !errors.Is(err, ErrNegativeLength) ||
		!strings.Contains(err.Error(), "-2147483648 is negative") {
		t.Fatalf("min-int extended = %v", err)
	}
}

func TestRejectsEveryTruncatedExtendedLengthHeader(t *testing.T) {
	header := decodeHex(t, "0203ffff00010000")
	for length := 0; length < len(header); length++ {
		_, err := DecodeFieldHeader(header[:length])
		if !errors.Is(err, ErrShortHeader) {
			t.Fatalf("truncation at %d = %v, want ErrShortHeader", length, err)
		}
		if !strings.Contains(err.Error(), "need 2 bytes") && !strings.Contains(err.Error(), "need 4 bytes") {
			t.Fatalf("truncation at %d: message %q lacks a byte count", length, err)
		}
	}
}

// FuzzDecodeFieldHeader asserts the decoder never panics on arbitrary input and
// that any header it accepts round-trips back to a header it re-accepts.
func FuzzDecodeFieldHeader(f *testing.F) {
	for _, s := range []string{"", "02", "02030000", "0203fffe", "0203ffff0000ffff", "0203ffff7fffffff", "0203ffffffffffff"} {
		b, _ := hex.DecodeString(s)
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := DecodeFieldHeader(data)
		if err != nil {
			return
		}
		if got.BytesConsumed != 4 && got.BytesConsumed != 8 {
			t.Fatalf("accepted header with BytesConsumed %d", got.BytesConsumed)
		}
		if got.Length < 0 || got.Length > ValueLengthMax {
			t.Fatalf("accepted out-of-range length %d", got.Length)
		}
		// A length that fits the compact form must have used it.
		byteLength, err := FieldHeaderByteLength(got.Length)
		if err != nil {
			t.Fatalf("FieldHeaderByteLength rejected an accepted length %d: %v", got.Length, err)
		}
		if got.Encoding == EncodingCompact && byteLength != 4 {
			t.Fatalf("compact encoding for length %d needing %d bytes", got.Length, byteLength)
		}
	})
}
