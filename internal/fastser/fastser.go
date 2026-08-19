// SPDX-License-Identifier: Apache-2.0

// Package fastser decodes and encodes SAP RFC "fast serialization" parameter
// payloads — the form an ABAP peer uses on the wire inside a CUT function
// result's 0x5001 container, distinct from the classic serialization handled by
// internal/classicrfc.
//
// This is original work for open-rfc-go, reverse-engineered from live A4H
// captures (see docs/discoveries/0001-live-type3-server.md). It currently covers
// the character field encoding, which is the common case for scalar and
// structure-of-char parameters (RFCSI and friends):
//
//	0x43 <len:1> 0x80 <value: len bytes, ASCII>
//
// The descriptor-table grammar that precedes the values, non-character type
// markers, and table (multi-row) encoding are not modelled yet.
package fastser

// charTag, charFlag frame one character field: tag 0x43 ('C'), a one-byte
// length, the flag 0x80, then that many ASCII value bytes.
const (
	charTag  = 0x43
	charFlag = 0x80
)

// CharField is one decoded character field: its raw ASCII value.
type CharField struct {
	Value []byte
}

// DecodeCharFields extracts every character field (0x43 len 0x80 value) from a
// fast-serialization payload, in order. Bytes that do not form a well-shaped
// char field are skipped, so it tolerates the surrounding descriptor table and
// framing until the value region is fully modelled.
func DecodeCharFields(payload []byte) []CharField {
	var out []CharField
	for i := 0; i+3 <= len(payload); {
		if payload[i] == charTag && payload[i+2] == charFlag {
			n := int(payload[i+1])
			if n > 0 && i+3+n <= len(payload) {
				v := make([]byte, n)
				copy(v, payload[i+3:i+3+n])
				out = append(out, CharField{Value: v})
				i += 3 + n
				continue
			}
		}
		i++
	}
	return out
}

// EncodeCharField encodes one character field as 0x43 <len> 0x80 <value>. The
// value must be at most 255 bytes; longer values are truncated to the length the
// single-byte length field can express (callers should split beforehand).
func EncodeCharField(value []byte) []byte {
	if len(value) > 0xff {
		value = value[:0xff]
	}
	out := make([]byte, 0, 3+len(value))
	out = append(out, charTag, byte(len(value)), charFlag)
	out = append(out, value...)
	return out
}
