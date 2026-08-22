// SPDX-License-Identifier: Apache-2.0

package fastser

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
)

// mustHex decodes a fixture captured from the wire.
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	return b
}

// The fixtures below come from a controlled differential against a live A4H
// (SAP_BASIS 758): `Z_CALL_RFC` drove `Z_DOUBLE`/`Z_GREET` over a fast-serialized
// type-3 destination while exactly one caller parameter was varied, and the
// frames were captured with this repo's relay sniffer.

// intField256 is the whole `N` parameter of a Z_DOUBLE request with N = 256:
//
//	'P' 07 "\TYPE=I"  03 0a "TABLE_LINE"  'N' 00010000  'E'
const intField256 = "50075c545950453d49030a5441424c455f4c494e454e0001000045"

// charFieldABCD is the whole `NAME` parameter of a Z_GREET request with
// NAME = "ABCD". The parameter is CHAR30, and only four bytes travel:
//
//	'P' 0c "\TYPE=CHAR30"  06 3c 00  0a "TABLE_LINE"  'C' 04 80 "ABCD"  'E'
//
// The 06 3c 00 run is type metadata this decoder does not model; it must be
// skipped without derailing the records around it.
const charFieldABCD = "500c5c545950453d434841523330063c000a5441424c455f4c494e454304804142434445"

func TestDecodeIntFieldIsLittleEndian(t *testing.T) {
	fields := DecodeTypedFields(mustHex(t, intField256))
	if len(fields) != 1 {
		t.Fatalf("got %d typed fields, want 1: %+v", len(fields), fields)
	}
	f := fields[0]
	if f.TypeName != "I" {
		t.Errorf("TypeName = %q, want %q", f.TypeName, "I")
	}
	// The name lives in the field-description list, which this fixture does not
	// include (it starts at the descriptor). fields_test.go covers that path.
	if !f.HasValue || f.Value.Tag != TagInt4 {
		t.Fatalf("value = %+v, want an int4 record", f.Value)
	}
	// 256 is the value that settles endianness: 00010000 little-endian, and
	// 00000100 would be big. The caller passed N = 256.
	if got := binary.LittleEndian.Uint32(f.Value.Value); got != 256 {
		t.Errorf("decoded N = %d, want 256 (bytes %x)", got, f.Value.Value)
	}
}

func TestStrictStreamStopsAtTheFieldList(t *testing.T) {
	payload := mustHex(t, intField256)
	recs, covered := DecodeRecords(payload)
	// 0x03 after the descriptor is the INT4 *type code* in the field-description
	// list (fields.go), not a record — so the strict record stream stops there.
	if len(recs) != 1 || !recs[0].IsTypeDescriptor() {
		t.Fatalf("want the descriptor and then a stop, got %+v", recs)
	}
	// Stopping is correct: what follows the descriptor is a field-description
	// list, not more records, and walking into it would invent them.
	if covered >= len(payload) {
		t.Errorf("covered %d of %d — the field list is not a record stream", covered, len(payload))
	}
}

func TestDecodeCharFieldSkipsTheFlagByte(t *testing.T) {
	// The strict record stream stops at the unmodelled metadata after the
	// descriptor, so the value comes from the anchored extractor.
	fields := DecodeTypedFields(mustHex(t, charFieldABCD))
	if len(fields) != 1 {
		t.Fatalf("got %d typed fields, want 1: %+v", len(fields), fields)
	}
	if fields[0].TypeName != "CHAR30" {
		t.Errorf("TypeName = %q, want CHAR30", fields[0].TypeName)
	}
	if !fields[0].HasValue {
		t.Fatalf("no character value proven in %+v", fields[0])
	}
	char := &fields[0].Value
	// The bug this pins: 0x80 sits between the length and the value. A decoder
	// that treats the field as <tag><len><value> reads "\x80ABC" and is wrong
	// by one byte for every character field on the wire.
	if got := string(char.Value); got != "ABCD" {
		t.Errorf("char value = %q, want %q", got, "ABCD")
	}
	if len(char.Value) != 4 {
		t.Errorf("char value = %d bytes, want 4 — one byte per character, not UTF-16", len(char.Value))
	}
}

func TestTagLettersInFieldNamesDoNotBecomeRecords(t *testing.T) {
	// "TABLE_LINE" contains 'E' (TagEnd) and 'N' (TagInt4). Neither may be read
	// as a record, and the real value after them must still be found. This is
	// what the byte-stepping resynchroniser got wrong.
	for _, tc := range []struct {
		name  string
		fx    string
		value string
	}{
		{"int4", intField256, "\x00\x01\x00\x00"},
		{"char", charFieldABCD, "ABCD"},
	} {
		fields := DecodeTypedFields(mustHex(t, tc.fx))
		if len(fields) != 1 || !fields[0].HasValue {
			t.Fatalf("%s: want one field with a proven value, got %+v", tc.name, fields)
		}
		if got := string(fields[0].Value.Value); got != tc.value {
			t.Errorf("%s: value = %q, want %q", tc.name, got, tc.value)
		}
	}
}

func TestDecodeRecordsRejectsCharWithoutFlag(t *testing.T) {
	// Same shape as a character field but missing the 0x80. Accepting it would
	// silently shift the value by one byte.
	payload := []byte{TagChar, 0x02, 0x00, 'A', 'B'}
	for _, r := range mustRecords(payload) {
		if r.Tag == TagChar {
			t.Errorf("accepted a char record without the flag byte: %+v", r)
		}
	}
}

func TestDecodeRecordsRejectsTruncatedValue(t *testing.T) {
	payload := []byte{TagChar, 200, charFlag, 'a', 'b', 'c'}
	recs, covered := DecodeRecords(payload)
	if len(recs) != 0 || covered != 0 {
		t.Errorf("got %d records covering %d bytes, want none from a truncated value", len(recs), covered)
	}
}

func TestEncodeRecordRoundTripsEveryKnownFraming(t *testing.T) {
	for _, tc := range []struct {
		name  string
		tag   byte
		value []byte
	}{
		{"char", TagChar, []byte("open-rfc-go")},
		{"descriptor", TagDescriptor, []byte(`\TYPE=I`)},
		{"int4", TagInt4, []byte{0x2a, 0, 0, 0}},
		{"padded", TagPadded, make([]byte, 300)}, // exercises the 2-byte length
	} {
		enc, ok := EncodeRecord(tc.tag, tc.value)
		if !ok {
			t.Errorf("%s: EncodeRecord refused a legal value", tc.name)
			continue
		}
		recs, covered := DecodeRecords(enc)
		if covered != len(enc) {
			t.Errorf("%s: covered %d of %d encoded bytes", tc.name, covered, len(enc))
		}
		if len(recs) != 1 {
			t.Errorf("%s: got %d records, want 1", tc.name, len(recs))
			continue
		}
		if recs[0].Tag != tc.tag || string(recs[0].Value) != string(tc.value) {
			t.Errorf("%s: round trip changed the record", tc.name)
		}
	}
}

func TestEncodeRecordRefusesUnrepresentable(t *testing.T) {
	if _, ok := EncodeRecord(TagChar, nil); ok {
		t.Error("an empty value should be refused, not encoded as a zero length")
	}
	if _, ok := EncodeRecord(TagChar, make([]byte, 256)); ok {
		t.Error("256 bytes exceeds the char length byte and should be refused")
	}
	if _, ok := EncodeRecord(TagInt4, []byte{1, 2, 3}); ok {
		t.Error("an int4 must be exactly four bytes")
	}
	if _, ok := EncodeRecord(0xAB, []byte("x")); ok {
		t.Error("a tag with no known framing must be refused, not guessed")
	}
}

func TestDecodedValueIsACopy(t *testing.T) {
	payload := mustHex(t, intField256)
	recs, _ := DecodeRecords(payload)
	if len(recs) == 0 {
		t.Fatal("no records")
	}
	before := string(recs[0].Value)
	for i := range payload {
		payload[i] = 0
	}
	if string(recs[0].Value) != before {
		t.Error("decoded value aliases the caller's payload; it must be copied")
	}
}

func FuzzDecodeRecords(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte{TagChar, 200, charFlag, 'a'})
	for _, s := range []string{intField256, charFieldABCD} {
		if b, err := hex.DecodeString(s); err == nil {
			f.Add(b)
		}
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		recs, covered := DecodeRecords(payload)
		if covered > len(payload) {
			t.Fatalf("covered %d exceeds payload %d", covered, len(payload))
		}
		last := -1
		for _, r := range recs {
			if r.Offset <= last {
				t.Fatalf("records out of order: offset %d after %d", r.Offset, last)
			}
			if r.Offset < 0 || r.Offset >= len(payload) {
				t.Fatalf("offset %d outside payload of %d", r.Offset, len(payload))
			}
			last = r.Offset
		}
		DecodeTypedFields(payload)
	})
}

func mustRecords(payload []byte) []Record {
	recs, _ := DecodeRecords(payload)
	return recs
}

// stringField8 is the whole `QUESTION` parameter of a live STFC_STRING request
// with an eight-character argument:
//
//	'P' 0c "\TYPE=STRING"  18  0a "TABLE_LINE"  'S' 08c0 0800 "ABCDEFGH"
//
// The 0x18 is the STRING type-metadata byte, unmodelled like CHAR30's.
const stringField8 = "500c5c545950453d535452494e47180a5441424c455f4c494e455308c008004142434445464748"

func TestDecodeStringFieldDoubleLength(t *testing.T) {
	fields := DecodeTypedFields(mustHex(t, stringField8))
	if len(fields) != 1 {
		t.Fatalf("got %d typed fields, want 1: %+v", len(fields), fields)
	}
	if fields[0].TypeName != "STRING" {
		t.Errorf("TypeName = %q, want STRING", fields[0].TypeName)
	}
	if !fields[0].HasValue || fields[0].Value.Tag != TagString {
		t.Fatalf("value = %+v, want a STRING record", fields[0].Value)
	}
	if got := string(fields[0].Value.Value); got != "ABCDEFGH" {
		t.Errorf("value = %q, want ABCDEFGH", got)
	}
	// One byte per character, established by a length differential on the wire.
	if len(fields[0].Value.Value) != 8 {
		t.Errorf("value = %d bytes for 8 characters; STRING is one byte per character", len(fields[0].Value.Value))
	}
}

func TestStringHeaderLengthsMustAgree(t *testing.T) {
	// The two lengths corroborate each other. A header whose copies disagree is
	// not a STRING, and accepting it would let arbitrary bytes pose as one.
	bad := []byte{TagString, 0x08, 0xc0, 0x09, 0x00, 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I'}
	for _, r := range mustRecords(bad) {
		if r.Tag == TagString {
			t.Errorf("accepted a STRING whose two lengths disagree: %+v", r)
		}
	}
	// And the flag must actually be set on the first copy.
	noFlag := []byte{TagString, 0x08, 0x00, 0x08, 0x00, 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H'}
	for _, r := range mustRecords(noFlag) {
		if r.Tag == TagString {
			t.Errorf("accepted a STRING without the length flag: %+v", r)
		}
	}
}

func TestCompressedStringIsRejectedNotMisread(t *testing.T) {
	// Above 512 bytes the payload is compressed, so the declared length far
	// exceeds what is actually present. The decoder must refuse rather than
	// return a short or over-read value — the declared length is the original
	// size, not the encoded one.
	payload := append([]byte{TagString, 0xb8, 0xcb, 0xb8, 0x0b}, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")...)
	recs, covered := DecodeRecords(payload)
	if len(recs) != 0 {
		t.Errorf("got %d records from a compressed STRING, want none: %+v", len(recs), recs)
	}
	if covered != 0 {
		t.Errorf("covered %d bytes, want 0", covered)
	}
}

// hexBytes decodes a fixture outside a test context, for fuzz seeds.
func hexBytes(s string) ([]byte, error) { return hex.DecodeString(s) }
