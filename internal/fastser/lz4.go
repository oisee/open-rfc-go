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

// The compression header.
//
// A compressed block is introduced by eight bytes immediately before it:
//
//	<uncompressed size:4 LE> <compressed size:4 LE> <block>
//
// Both sizes are present, so nothing has to be inferred: the block is not
// self-delimiting, and guessing where it ends does not work — a block truncated
// early can still finish on LZ4's literals-only final sequence, so several wrong
// lengths look valid. The header is what makes a block locatable at all.
//
// Established over the whole capture corpus: every compressed block found this
// way decodes to exactly its declared uncompressed size while consuming exactly
// its declared compressed size, and no other offset in any frame satisfies both.

// compressionHeaderLen is the size of the two length fields.
const compressionHeaderLen = 8

// Compressed is one located block: its declared sizes and the bytes it yielded.
type Compressed struct {
	// Offset is where the eight-byte header began.
	Offset int
	// CompressedSize and UncompressedSize are as the header declared them.
	CompressedSize   int
	UncompressedSize int
	// Data is the decompressed payload.
	Data []byte
}

// DecodeCompressedAt reads the header at off and decompresses the block that
// follows, or reports ok=false when off does not begin one. Both declared sizes
// are enforced against what the block actually does, which is what keeps a
// coincidental pair of plausible numbers from being read as a header.
func DecodeCompressedAt(payload []byte, off int) (c Compressed, next int, ok bool) {
	if off < 0 || off+compressionHeaderLen > len(payload) {
		return Compressed{}, off, false
	}
	uncompressed := int(payload[off]) | int(payload[off+1])<<8 |
		int(payload[off+2])<<16 | int(payload[off+3])<<24
	compressed := int(payload[off+4]) | int(payload[off+5])<<8 |
		int(payload[off+6])<<16 | int(payload[off+7])<<24

	// A block never expands, and both sizes have to fit what is here.
	if compressed <= 0 || uncompressed <= 0 || compressed > uncompressed ||
		uncompressed > maxDecompressed {
		return Compressed{}, off, false
	}
	start := off + compressionHeaderLen
	if start+compressed > len(payload) {
		return Compressed{}, off, false
	}

	data, used, err := DecompressBlock(payload[start:start+compressed], uncompressed)
	if err != nil || used != compressed {
		return Compressed{}, off, false
	}
	return Compressed{
		Offset:           off,
		CompressedSize:   compressed,
		UncompressedSize: uncompressed,
		Data:             data,
	}, start + compressed, true
}

// FindCompressed locates every compressed block in a frame.
//
// It searches rather than walking, because the framing above the compression
// layer is not fully modelled and a caller rarely knows the offset. The search
// is safe to run blind: a candidate has to carry two sizes that agree with each
// other and with a block that then decodes exactly, which arbitrary bytes do not.
func FindCompressed(payload []byte) []Compressed {
	var out []Compressed
	for i := 0; i+compressionHeaderLen < len(payload); {
		c, next, ok := DecodeCompressedAt(payload, i)
		if !ok {
			i++
			continue
		}
		out = append(out, c)
		i = next
	}
	return out
}
