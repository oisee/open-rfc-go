// SPDX-License-Identifier: Apache-2.0
//
// Original tests for the unicode-scalar codec ported from open-rfc
// src/values/unicode-scalar.ts (no dedicated upstream test file; exercised via
// xRFC). The zero-padding-transparency case pins the recurring-bug-class fix:
// a reference's value, not its digit count, decides what it denotes. See
// docs/provenance.md.

package value

import (
	"errors"
	"testing"
)

func TestDecodeXMLEntityReference(t *testing.T) {
	cases := []struct {
		raw     string
		wantCP  int
		wantLen int
	}{
		{"&amp;", 0x26, 5},
		{"&lt;", 0x3c, 4},
		{"&gt;", 0x3e, 4},
		{"&quot;", 0x22, 6},
		{"&apos;", 0x27, 6},
		{"&#65;", 0x41, 5},
		{"&#x41;", 0x41, 6},
		{"&#00;", 0x00, 5},
		{"&#x0041;", 0x41, 8},
	}
	for _, c := range cases {
		cp, n, err := DecodeXMLEntityReference(c.raw, 0, "p")
		if err != nil || cp != c.wantCP || n != c.wantLen {
			t.Fatalf("%q = %#x/%d, %v; want %#x/%d", c.raw, cp, n, err, c.wantCP, c.wantLen)
		}
	}
}

func TestXMLEntityZeroPaddingTransparent(t *testing.T) {
	// The recurring-bug-class fix: value, not digit count, decides the entity.
	forms := []string{"&#65;", "&#065;", "&#0000065;", "&#x41;", "&#x00041;"}
	for _, f := range forms {
		cp, _, err := DecodeXMLEntityReference(f, 0, "p")
		if err != nil || cp != 0x41 {
			t.Fatalf("%q = %#x, %v; want 0x41", f, cp, err)
		}
	}
}

func TestDecodeXMLEntityReferenceRejects(t *testing.T) {
	for _, raw := range []string{"&amp", "&;", "&foo;", "&#xD800;", "&#55296;", "&#1114112;", "&#x110000;"} {
		if _, _, err := DecodeXMLEntityReference(raw, 0, "p"); !errors.Is(err, ErrXMLEntity) {
			t.Fatalf("%q accepted: %v", raw, err)
		}
	}
}

func TestAssertUnicodeScalarText(t *testing.T) {
	if err := AssertUnicodeScalarText("hello \U0001F642 é", "p"); err != nil {
		t.Fatalf("valid text rejected: %v", err)
	}
	if err := AssertNulFreeUnicodeScalarText("a\x00b", "p"); !errors.Is(err, ErrScalarText) {
		t.Fatalf("NUL not rejected: %v", err)
	}
	// Invalid UTF-8 (WTF-8 lone surrogate 0xED 0xA0 0x80) → rejected.
	if err := AssertUnicodeScalarText(string([]byte{0xed, 0xa0, 0x80}), "p"); !errors.Is(err, ErrScalarText) {
		t.Fatalf("isolated surrogate not rejected: %v", err)
	}
}

func FuzzDecodeXMLEntityReference(f *testing.F) {
	f.Add("&amp;")
	f.Add("&#x41;")
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) == 0 || raw[0] != '&' {
			return
		}
		cp, n, err := DecodeXMLEntityReference(raw, 0, "p")
		if err != nil {
			return
		}
		if cp < 0 || cp > 0x10ffff || (cp >= 0xd800 && cp <= 0xdfff) {
			t.Fatalf("accepted out-of-range code point %#x", cp)
		}
		if n <= 0 || n > len(raw) {
			t.Fatalf("bad length %d for %q", n, raw)
		}
	})
}
