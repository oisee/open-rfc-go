// SPDX-License-Identifier: Apache-2.0

package fastser

import "errors"

// Decompression.
//
// A fast-serialization payload above 512 bytes is compressed, and the codec is
// the standard **LZ4 block format** — not merely LZ77-shaped, but the published
// block layout: a nibble token, 255-chains for both run lengths, a two-byte
// little-endian match offset, a four-byte minimum match, and a final sequence
// that is literals only.
//
// Established by decoding, not by inspection: a strict decoder written to the
// published format reproduces every compressed block in this repository's
// captures — each consuming exactly the declared compressed byte count and
// emitting exactly the declared uncompressed one.
//
// This is the block format alone. There is no LZ4 frame header, no magic, and no
// checksum; the surrounding fast-serialization framing supplies the two sizes.

// ErrCorruptBlock reports a block that does not decode as LZ4.
var ErrCorruptBlock = errors.New("fastser: corrupt LZ4 block")

// maxDecompressed bounds what one block may expand to. A compressed stream can
// name an expansion far larger than itself, so a decoder that trusts the block
// is a memory-exhaustion primitive on hostile input; callers pass the size the
// framing declared and this refuses anything beyond it.
const maxDecompressed = 1 << 24

// DecompressBlock decodes one LZ4 block into exactly want bytes.
//
// want comes from the framing, not from the block, and is enforced: a block that
// expands to a different size is rejected rather than returned short. Returns
// the number of source bytes consumed alongside the output, because a block is
// not self-delimiting — the caller needs to know where the next one starts, and
// a block that does not consume its whole input is a sign the offset was wrong.
func DecompressBlock(src []byte, want int) (out []byte, consumed int, err error) {
	if want < 0 || want > maxDecompressed {
		return nil, 0, ErrCorruptBlock
	}
	dst := make([]byte, 0, want)
	i := 0

	for i < len(src) {
		token := src[i]
		i++

		// Literal run: the high nibble, extended by 255-chains.
		litLen := int(token >> 4)
		if litLen == 15 {
			for {
				if i >= len(src) {
					return nil, 0, ErrCorruptBlock
				}
				b := src[i]
				i++
				litLen += int(b)
				if b != 255 {
					break
				}
				if litLen > want {
					return nil, 0, ErrCorruptBlock
				}
			}
		}
		if litLen > len(src)-i || len(dst)+litLen > want {
			return nil, 0, ErrCorruptBlock
		}
		dst = append(dst, src[i:i+litLen]...)
		i += litLen

		// The last sequence is literals only and has no match, so running out of
		// input here is the normal end of a block rather than an error.
		if i == len(src) {
			break
		}
		if i+2 > len(src) {
			return nil, 0, ErrCorruptBlock
		}

		// Match: a two-byte little-endian distance back into what was produced.
		offset := int(src[i]) | int(src[i+1])<<8
		i += 2
		if offset == 0 || offset > len(dst) {
			return nil, 0, ErrCorruptBlock
		}

		matchLen := int(token & 0x0f)
		if matchLen == 15 {
			for {
				if i >= len(src) {
					return nil, 0, ErrCorruptBlock
				}
				b := src[i]
				i++
				matchLen += int(b)
				if b != 255 {
					break
				}
				if matchLen > want {
					return nil, 0, ErrCorruptBlock
				}
			}
		}
		matchLen += 4 // MINMATCH

		if len(dst)+matchLen > want {
			return nil, 0, ErrCorruptBlock
		}
		// Copied one byte at a time on purpose: an offset smaller than the match
		// length is legal and is how LZ4 encodes a repeating run, so the source
		// overlaps the destination and a bulk copy would produce the wrong bytes.
		start := len(dst) - offset
		for n := 0; n < matchLen; n++ {
			dst = append(dst, dst[start+n])
		}
	}

	if len(dst) != want {
		return nil, 0, ErrCorruptBlock
	}
	return dst, i, nil
}
