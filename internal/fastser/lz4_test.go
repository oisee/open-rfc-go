// SPDX-License-Identifier: Apache-2.0

package fastser

import (
	"bytes"
	"strings"
	"testing"
)

// liveBlock is a compressed block lifted from a live STFC_STRING conversation:
// 96 bytes on the wire, 577 bytes of fast-serialization payload inside. Both
// numbers come from the framing around it, and the decoder is held to them.
const liveBlock = "f50654085155455354494f4e54084d59414e53574552511400ff2c44015" +
	"00c5c545950453d535452494e47180a5441424c455f4c494e455301c2010241424344454647" +
	"48494a4b4c4d4e4f505152535455565758595a1a00ffd1505051525345"

func TestDecompressLiveBlock(t *testing.T) {
	src := mustHex(t, liveBlock)
	out, consumed, err := DecompressBlock(src, 577)
	if err != nil {
		t.Fatalf("a real captured block failed to decode: %v", err)
	}
	if consumed != len(src) {
		t.Errorf("consumed %d of %d source bytes; a block should be used whole", consumed, len(src))
	}
	if len(out) != 577 {
		t.Fatalf("produced %d bytes, want the declared 577", len(out))
	}
	// The output is fast-serialization payload, so it must contain the framing
	// the rest of this package knows how to read.
	for _, want := range []string{"QUESTION", "MYANSWER", `\TYPE=STRING`, "TABLE_LINE"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("decompressed payload is missing %q", want)
		}
	}
	// And the caller's argument, a repeating alphabet, reconstructed through
	// back-references rather than stored literally.
	if !bytes.Contains(out, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJ")) {
		t.Error("the repeated alphabet did not come back — matches are not being replayed")
	}
}

func TestDecompressEnforcesTheDeclaredSize(t *testing.T) {
	src := mustHex(t, liveBlock)
	for _, want := range []int{576, 578, 0} {
		if _, _, err := DecompressBlock(src, want); err == nil {
			t.Errorf("want=%d should have been rejected; the framing's size is authoritative", want)
		}
	}
}

func TestDecompressRefusesHostileBlocks(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  []byte
		want int
	}{
		{"offset past the output", []byte{0x10, 'a', 0xff, 0x7f, 0x00}, 64},
		{"zero offset", []byte{0x10, 'a', 0x00, 0x00, 0x00}, 64},
		{"literal run past the end", []byte{0xf0, 0xff, 'a'}, 64},
		{"absurd expansion", []byte{0x1f, 'a', 0x01, 0x00, 0xff, 0xff, 0xff}, 8},
	} {
		if _, _, err := DecompressBlock(tc.src, tc.want); err == nil {
			t.Errorf("%s: should have been refused", tc.name)
		}
	}
	// A declared size beyond the cap is refused before any allocation happens.
	if _, _, err := DecompressBlock([]byte{0x00}, maxDecompressed+1); err == nil {
		t.Error("an oversized declared length must be refused, not allocated")
	}
}

func TestDecompressOverlappingMatch(t *testing.T) {
	// Offset 1 with a match longer than the offset: LZ4's way of writing a run.
	// A bulk copy would get this wrong; byte-at-a-time is required.
	//   token 0x1f: 1 literal, match 15+4 -> extended by the 0x00 chain byte
	src := []byte{0x1f, 'x', 0x01, 0x00, 0x00}
	out, _, err := DecompressBlock(src, 20)
	if err != nil {
		t.Fatalf("overlapping match failed: %v", err)
	}
	if string(out) != strings.Repeat("x", 20) {
		t.Errorf("got %q, want twenty x — the match must replay byte by byte", out)
	}
}

func FuzzDecompressBlock(f *testing.F) {
	if b, err := hexBytes(liveBlock); err == nil {
		f.Add(b, 577)
	}
	f.Add([]byte{0x10, 'a', 0x01, 0x00}, 8)
	f.Fuzz(func(t *testing.T, src []byte, want int) {
		out, consumed, err := DecompressBlock(src, want)
		if err != nil {
			return
		}
		if len(out) != want {
			t.Fatalf("returned %d bytes for want=%d without an error", len(out), want)
		}
		if consumed > len(src) {
			t.Fatalf("consumed %d of %d source bytes", consumed, len(src))
		}
	})
}
