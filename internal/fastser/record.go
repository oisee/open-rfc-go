// SPDX-License-Identifier: Apache-2.0

package fastser

import "bytes"

// The record grammar.
//
// A fast-serialization payload is a stream of records. Each record opens with a
// one-byte tag, and **the tag decides how its value is framed** — there is no
// single length rule. Every entry in the table below was established by a
// controlled differential against a live A4H system (SAP_BASIS 758): one caller
// parameter varied at a time, everything else held fixed, frames captured with
// this repo's own relay sniffer.
//
//	0x43 'C'  char      <len:1> 0x80 <len bytes>   single-byte, NOT UTF-16
//	0x4e 'N'  int4      <4 bytes>                  little-endian, fixed width
//	0x50 'P'  descriptor <len:1> <len bytes>       the `\TYPE=...` announcement
//	0x03      name      <len:1> <len bytes>        field name
//	0x30 '0'  padded    <len:2 BE> <len bytes>     padded UTF-16LE text
//	0x45 'E'  end       (no value)                 closes a value
//
// The evidence for the two that matter most:
//
//	Z_DOUBLE  N=1   -> 4e 01000000     N=2 -> 4e 02000000
//	                  N=256 -> 4e 00010000        (little-endian confirmed)
//	Z_GREET   NAME="AB"       -> 43 02 80 4142
//	          NAME="ABCD"     -> 43 04 80 41424344
//	          NAME="ABCDEFGH" -> 43 08 80 4142434445464748
//
// The Z_GREET frame grew exactly one byte per added character, which is what
// rules out UTF-16 for char values and rules out padding to the declared ABAP
// width — the parameter is CHAR30 and only the significant bytes travel.
//
// Payloads are COMPRESSED above 512 bytes. Measured by bisection on a live
// STFC_STRING call: a 512-character argument travels literally (the frame grows
// exactly one byte per character up to there), and 513 characters collapse the
// frame from 919 bytes to 448 with only a 26-byte literal remnant. The scheme is
// LZ-like — one literal copy of the repeating input survives and the rest becomes
// back-references — and it applies to the whole parameter block, field names
// included, not just to table content.
//
// The practical consequence for this decoder: it reads the literal form only.
// Above the threshold DecodeRecords will account for very little, and that is the
// encoding, not a bug. Check the coverage count rather than assuming a short
// result means a malformed payload. Decompression is not implemented.
//
// See docs/discoveries/serializer-selection.md. What is still NOT modelled: the
// 0x5001 container's nesting; the type-metadata bytes between a `\TYPE=`
// descriptor and the field name (0x03 for I, 0x18 for STRING, 0x06 0x3c 0x00 for
// CHAR30, 0x13 0x86 0x00 for RFCTEST — reading the last two as a little-endian
// width fits CHAR30 at 60 = 30x2 but does not fit RFCTEST, so it is recorded and
// not implemented); and value encodings for the types not listed above. DecodeRecords reports how
// many bytes it accounted for so that gap stays measurable instead of implied.

// Tag values whose framing is confirmed by a capture. Adding a tag here without
// a capture behind it would defeat the point of the coverage count.
const (
	// TagName is the byte seen before a field name in the int4 case. It is NOT
	// universal: that position holds 0x18 before a STRING field's name and 0x00
	// before a CHAR30's, so what precedes a name is type metadata of a shape we
	// have not pinned down. Name extraction therefore works for the confirmed
	// cases and degrades to "no name" elsewhere, rather than guessing.
	TagName       = 0x03
	TagPadded     = 0x30 // '0' — padded UTF-16LE text, two-byte big-endian length
	TagChar       = 0x43 // 'C' — character field, one-byte length then the 0x80 flag
	TagEnd        = 0x45 // 'E' — end marker, no value
	TagInt4       = 0x4e // 'N' — four-byte little-endian integer, no length byte
	TagDescriptor = 0x50 // 'P' — type descriptor, one-byte length
)

// charFlag (declared in fastser.go) sits between a character field's length and
// its value. Its meaning is not established; it is 0x80 in every capture, and
// skipping it is required to read the value at all.

// typeDescriptorPrefix opens the value of a TagDescriptor record.
var typeDescriptorPrefix = []byte(`\TYPE=`)

// framing describes how one tag delimits its value.
type framing int

const (
	framingLen1     framing = iota // <len:1> <value>
	framingLen1Flag                // <len:1> charFlag <value>
	framingLen2BE                  // <len:2 big-endian> <value>
	framingFixed4                  // <4 bytes>, no length
	framingNone                    // no value at all
)

var tagFraming = map[byte]framing{
	TagName:       framingLen1,
	TagDescriptor: framingLen1,
	TagChar:       framingLen1Flag,
	TagPadded:     framingLen2BE,
	TagInt4:       framingFixed4,
	TagEnd:        framingNone,
}

// Record is one decoded fast-serialization record.
type Record struct {
	// Offset is where the record's tag byte sat in the payload.
	Offset int
	// Tag is the record's leading byte.
	Tag byte
	// Value is a copy of the record's value bytes, nil for a valueless tag.
	Value []byte
}

// IsTypeDescriptor reports whether the record is a `\TYPE=` descriptor.
func (r Record) IsTypeDescriptor() bool {
	return r.Tag == TagDescriptor && bytes.HasPrefix(r.Value, typeDescriptorPrefix)
}

// TypeName returns the DDIC or built-in type named by a `\TYPE=` descriptor, and
// whether the record was one. Composite parameters are referenced by DDIC type
// name rather than by an inline layout, so this is the key a caller resolves
// through metadata before decoding the value that follows.
func (r Record) TypeName() (string, bool) {
	if !r.IsTypeDescriptor() {
		return "", false
	}
	return string(r.Value[len(typeDescriptorPrefix):]), true
}

// decodeRecordAt parses one record at off. It returns the record and the offset
// just past it, or ok=false when no known rule applies there.
func decodeRecordAt(payload []byte, off int) (rec Record, next int, ok bool) {
	if off >= len(payload) {
		return Record{}, off, false
	}
	tag := payload[off]
	f, known := tagFraming[tag]
	if !known {
		return Record{}, off, false
	}

	var start, length int
	switch f {
	case framingNone:
		return Record{Offset: off, Tag: tag}, off + 1, true
	case framingFixed4:
		start, length = off+1, 4
	case framingLen1:
		if off+1 >= len(payload) {
			return Record{}, off, false
		}
		start, length = off+2, int(payload[off+1])
	case framingLen1Flag:
		if off+2 >= len(payload) || payload[off+2] != charFlag {
			return Record{}, off, false
		}
		start, length = off+3, int(payload[off+1])
	case framingLen2BE:
		if off+2 >= len(payload) {
			return Record{}, off, false
		}
		start, length = off+3, int(payload[off+1])<<8|int(payload[off+2])
	}

	// A zero-length value carries nothing and is indistinguishable from noise,
	// so it is not accepted as a record.
	if length == 0 || start+length > len(payload) {
		return Record{}, off, false
	}
	return Record{Offset: off, Tag: tag, Value: clone(payload[start : start+length])}, start + length, true
}

// DecodeRecordsFrom parses records contiguously starting at off and STOPS at the
// first byte that begins no known record. It never skips ahead.
//
// Resynchronising by stepping one byte at a time was tried and removed: the tags
// are ASCII letters, so 'E' (TagEnd) and 'N' (TagInt4) occur inside ordinary
// field names like "TABLE_LINE". A skipping decoder happily reads those as
// records, drifts, and swallows the real value that follows. Phantom records are
// worse than parsing less, so this stops instead of guessing, and the caller
// decides where to resume from a signature it trusts.
func DecodeRecordsFrom(payload []byte, off int) (recs []Record, next int) {
	for off < len(payload) {
		rec, after, ok := decodeRecordAt(payload, off)
		if !ok {
			return recs, off
		}
		recs = append(recs, rec)
		off = after
	}
	return recs, off
}

// DecodeRecords parses records contiguously from the start of the payload,
// stopping at the first byte it cannot account for. covered is how many bytes
// that consumed — compare it with len(payload) to see how much of the encoding
// is modelled rather than assuming the records are the whole story.
func DecodeRecords(payload []byte) (recs []Record, covered int) {
	recs, next := DecodeRecordsFrom(payload, 0)
	return recs, next
}

// descriptorAt reports whether a type descriptor starts at off, which is the one
// signature in this format strong enough to anchor on: the tag, a length, and a
// `\TYPE=` prefix that the length must agree with.
func descriptorAt(payload []byte, off int) (Record, int, bool) {
	rec, next, ok := decodeRecordAt(payload, off)
	if !ok || !rec.IsTypeDescriptor() {
		return Record{}, off, false
	}
	return rec, next, true
}

// FindTypeDescriptors locates every `\TYPE=` descriptor in the payload. Because
// the signature carries its own length, a false positive would have to be a byte
// run that spells the tag, a matching length and the prefix — which is why this,
// and not a bare tag, is what the field extractor anchors on.
func FindTypeDescriptors(payload []byte) []Record {
	var out []Record
	for i := 0; i+len(typeDescriptorPrefix)+2 <= len(payload); i++ {
		if rec, _, ok := descriptorAt(payload, i); ok {
			out = append(out, rec)
		}
	}
	return out
}

// TypedField is a `\TYPE=` descriptor together with the field name and value
// records that follow it — the shape a scalar parameter takes on the wire:
//
//	'P' "\TYPE=I"  0x03 "TABLE_LINE"  'N' <4 bytes>  'E'
type TypedField struct {
	// TypeName is the type named by the descriptor, e.g. "I", "CHAR90",
	// "STRING", or a DDIC name such as "RFCSI" or "RFCTEST".
	TypeName string
	// FieldName is the name record following the descriptor, if there was one.
	FieldName string
	// Value is the value record following the name, if there was one.
	Value Record
	// HasValue reports whether a value record followed.
	HasValue bool
}

// maxMetadataGap bounds how far past a descriptor the value may sit. Between the
// descriptor and the value there is type metadata this decoder does not model
// (0x03 for I, 0x18 for STRING, 0x06 0x3c 0x00 for CHAR30, 0x13 0x86 0x00 for
// RFCTEST) followed by the field name. The bound keeps a miss local instead of
// letting the scan wander into the next field.
const maxMetadataGap = 64

// DecodeTypedFields extracts the scalar parameters it can prove. Each `\TYPE=`
// descriptor anchors a field; the value is then the first *validated* value
// record within maxMetadataGap bytes — validated meaning a character record whose
// flag byte is present, or a four-byte integer immediately followed by the end
// marker. Requiring that corroboration is what keeps an 'E' or an 'N' inside a
// field name from being mistaken for a record.
//
// A descriptor whose value cannot be proven is still returned, with HasValue
// false: knowing a type was announced is useful even when its value has not been
// decoded, and it keeps the unmodelled cases visible.
func DecodeTypedFields(payload []byte) []TypedField {
	var out []TypedField
	for i := 0; i < len(payload); i++ {
		desc, after, ok := descriptorAt(payload, i)
		if !ok {
			continue
		}
		name, _ := desc.TypeName()
		f := TypedField{TypeName: name}

		limit := min(after+maxMetadataGap, len(payload))
		for j := after; j < limit; j++ {
			rec, next, ok := decodeRecordAt(payload, j)
			if !ok {
				continue
			}
			switch rec.Tag {
			case TagChar:
				// decodeRecordAt already required the flag byte.
				f.Value, f.HasValue = rec, true
			case TagInt4:
				if next < len(payload) && payload[next] == TagEnd {
					f.Value, f.HasValue = rec, true
				}
			case TagName:
				if f.FieldName == "" && isPlainName(rec.Value) {
					f.FieldName = string(rec.Value)
				}
			}
			if f.HasValue {
				break
			}
		}
		out = append(out, f)
		i = after - 1
	}
	return out
}

// isPlainName reports whether the bytes look like an ABAP identifier, which is
// the only shape a field name takes.
func isPlainName(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for _, c := range b {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '/') {
			return false
		}
	}
	return true
}

// EncodeRecord encodes one record for the tags whose framing is known. It
// reports false for a tag it cannot frame, or a value the tag's length field
// cannot express — a caller splits or picks another representation rather than
// silently shipping a truncated field.
func EncodeRecord(tag byte, value []byte) ([]byte, bool) {
	f, known := tagFraming[tag]
	if !known {
		return nil, false
	}
	switch f {
	case framingNone:
		return []byte{tag}, len(value) == 0
	case framingFixed4:
		if len(value) != 4 {
			return nil, false
		}
		return append([]byte{tag}, value...), true
	case framingLen1:
		if len(value) == 0 || len(value) > 0xff {
			return nil, false
		}
		return append([]byte{tag, byte(len(value))}, value...), true
	case framingLen1Flag:
		if len(value) == 0 || len(value) > 0xff {
			return nil, false
		}
		return append([]byte{tag, byte(len(value)), charFlag}, value...), true
	case framingLen2BE:
		if len(value) == 0 || len(value) > 0xffff {
			return nil, false
		}
		return append([]byte{tag, byte(len(value) >> 8), byte(len(value))}, value...), true
	}
	return nil, false
}

func clone(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
