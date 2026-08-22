// SPDX-License-Identifier: Apache-2.0

package fastser

import "bytes"

// The record grammar.
//
// A fast-serialization payload is a stream of records. Each record opens with a
// one-byte tag, and the tag decides how its value is delimited:
//
//	length-prefixed   <tag> <len:1> <value: len bytes>
//	length-prefixed   <tag> 0x00 <len:1> <value: len bytes>   (the 0x00 form)
//	fixed-width       <tag> <value: n bytes>                  (n from the tag)
//
// The tags are ASCII type letters where they are readable — 'C' for a character
// field, 'N' for a four-byte integer, 'P' for the type descriptor that precedes
// a value — alongside small binary tags whose meaning is not yet established.
//
// What the 0x00 form means is *not* known. It is not a 16-bit length escape: in
// every capture the byte after it is a plain one-byte length that matches the
// value exactly. It is parsed so the stream stays aligned, and flagged on the
// record so a caller can tell the two forms apart, but nothing is claimed about
// why a peer chooses one over the other.
//
// Derived from live A4H captures (SAP_BASIS 758) taken with this repo's own
// relay sniffer; see docs/discoveries/serializer-selection.md. Records that do
// not fit any known rule are skipped a byte at a time, and DecodeRecords reports
// how much of the payload it actually accounted for, so partial understanding
// stays visible instead of being silently papered over.

// Tag values observed on the wire. Only those whose framing is confirmed by a
// capture are listed; add to this table as more are established.
const (
	TagChar       = 0x43 // 'C' — character field, length-prefixed
	TagInt4       = 0x4e // 'N' — four-byte little-endian integer, fixed width
	TagDescriptor = 0x50 // 'P' — type descriptor, length-prefixed ("\TYPE=...")
	TagName       = 0x03 //       field name, length-prefixed
	TagPadded     = 0x30 // '0' — padded UTF-16LE text, length-prefixed
	TagEnd        = 0x45 // 'E' — end marker, no value
)

// typeDescriptorPrefix opens the value of a TagDescriptor record.
var typeDescriptorPrefix = []byte(`\TYPE=`)

// fixedWidth maps a tag to its value width for the tags that carry no length
// byte. A tag absent from this map and from lengthPrefixed is unknown.
var fixedWidth = map[byte]int{
	TagInt4: 4,
	TagEnd:  0,
}

// lengthPrefixed lists the tags whose value is introduced by a length byte.
var lengthPrefixed = map[byte]bool{
	TagChar:       true,
	TagDescriptor: true,
	TagName:       true,
	TagPadded:     true,
}

// Record is one decoded fast-serialization record.
type Record struct {
	// Offset is where the record's tag byte sat in the payload.
	Offset int
	// Tag is the record's leading byte.
	Tag byte
	// Value is a copy of the record's value bytes, empty for a valueless tag.
	Value []byte
	// LengthFlagged reports that the record used the 0x00 length form. What
	// that form signifies is not established; see the note at the top.
	LengthFlagged bool
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

	if width, fixed := fixedWidth[tag]; fixed {
		end := off + 1 + width
		if end > len(payload) {
			return Record{}, off, false
		}
		return Record{Offset: off, Tag: tag, Value: clone(payload[off+1 : end])}, end, true
	}

	if !lengthPrefixed[tag] {
		return Record{}, off, false
	}

	hdr, flagged := 2, false
	if off+1 >= len(payload) {
		return Record{}, off, false
	}
	length := int(payload[off+1])
	if length == 0 {
		if off+2 >= len(payload) {
			return Record{}, off, false
		}
		length, hdr, flagged = int(payload[off+2]), 3, true
	}
	// A zero-length value carries nothing and is indistinguishable from noise,
	// so it is not accepted as a record.
	if length == 0 || off+hdr+length > len(payload) {
		return Record{}, off, false
	}
	return Record{
		Offset:        off,
		Tag:           tag,
		Value:         clone(payload[off+hdr : off+hdr+length]),
		LengthFlagged: flagged,
	}, off + hdr + length, true
}

// DecodeRecords parses every record it can recognise in a fast-serialization
// payload, in order. Bytes that begin no known record are skipped one at a time,
// so a payload that is only partly understood still yields the records inside it
// rather than nothing at all.
//
// covered is the number of payload bytes consumed by returned records. Compare
// it against len(payload) to see how much of the encoding is actually modelled —
// a caller that needs certainty should check it rather than assume the records
// are the whole story.
func DecodeRecords(payload []byte) (recs []Record, covered int) {
	for i := 0; i < len(payload); {
		rec, next, ok := decodeRecordAt(payload, i)
		if !ok {
			i++
			continue
		}
		recs = append(recs, rec)
		covered += next - i
		i = next
	}
	return recs, covered
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

// DecodeTypedFields groups the record stream into typed fields. Every `\TYPE=`
// descriptor starts a field; an immediately following name record supplies the
// name, and the record after that its value. Descriptors with nothing after them
// are still returned, with HasValue false, because knowing a type was announced
// is useful even when its value has not been decoded.
func DecodeTypedFields(payload []byte) []TypedField {
	recs, _ := DecodeRecords(payload)
	var out []TypedField
	for i := 0; i < len(recs); i++ {
		name, ok := recs[i].TypeName()
		if !ok {
			continue
		}
		f := TypedField{TypeName: name}
		if i+1 < len(recs) && recs[i+1].Tag == TagName {
			f.FieldName = string(recs[i+1].Value)
			if i+2 < len(recs) && !recs[i+2].IsTypeDescriptor() {
				f.Value, f.HasValue = recs[i+2], true
			}
		}
		out = append(out, f)
	}
	return out
}

// EncodeRecord encodes one length-prefixed record as <tag> <len> <value>. The
// plain form is always emitted: the 0x00 form is understood well enough to parse
// but not well enough to choose deliberately, and a peer accepts the plain one.
//
// A value longer than 255 bytes cannot be expressed by the single length byte
// and is rejected, so a caller splits or picks another representation rather
// than silently shipping a truncated field.
func EncodeRecord(tag byte, value []byte) ([]byte, bool) {
	if len(value) == 0 || len(value) > 0xff {
		return nil, false
	}
	out := make([]byte, 0, 2+len(value))
	out = append(out, tag, byte(len(value)))
	return append(out, value...), true
}

func clone(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
