// SPDX-License-Identifier: Apache-2.0

package fastser

import (
	"encoding/binary"
	"encoding/hex"
	"testing"
	"unicode/utf16"
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

// zDoubleTypedField is the parameter region of a live Z_DOUBLE request captured
// against A4H (SAP_BASIS 758) with the relay sniffer: the ABAP caller passed
// N = 21, and the field rides as
//
//	'P' 07 "\TYPE=I"  0x03 0x0a "TABLE_LINE"  'N' 15000000
const zDoubleTypedField = "50075c545950453d49030a5441424c455f4c494e454e15000000"

// pingResponsePadded is the tail of a live RFC_PING response: a one-byte 'P'
// record followed by a 0x00-form padded text record carrying the calling
// program name in UTF-16LE, space-padded to 80 bytes.
const pingResponsePadded = "5001013000505300410050004c0053005200460043002000200020002000" +
	"200020002000200020002000200020002000200020002000200020002000" +
	"2000200020002000200020002000200020002000200020002000"

func TestDecodeTypedFieldZDouble(t *testing.T) {
	fields := DecodeTypedFields(mustHex(t, zDoubleTypedField))
	if len(fields) != 1 {
		t.Fatalf("got %d typed fields, want 1: %+v", len(fields), fields)
	}
	f := fields[0]
	if f.TypeName != "I" {
		t.Errorf("TypeName = %q, want %q", f.TypeName, "I")
	}
	if f.FieldName != "TABLE_LINE" {
		t.Errorf("FieldName = %q, want %q", f.FieldName, "TABLE_LINE")
	}
	if !f.HasValue {
		t.Fatal("HasValue = false, want the 'N' value record")
	}
	if f.Value.Tag != TagInt4 {
		t.Errorf("value tag = %#x, want %#x", f.Value.Tag, TagInt4)
	}
	if len(f.Value.Value) != 4 {
		t.Fatalf("value length = %d, want 4", len(f.Value.Value))
	}
	// The ABAP side sent N = 21; INT4 rides little-endian on this system.
	if got := binary.LittleEndian.Uint32(f.Value.Value); got != 21 {
		t.Errorf("decoded N = %d, want 21", got)
	}
}

func TestDecodeRecordsCoversTheWholeTypedField(t *testing.T) {
	payload := mustHex(t, zDoubleTypedField)
	recs, covered := DecodeRecords(payload)
	if covered != len(payload) {
		t.Errorf("covered %d of %d bytes; unmodelled bytes remain", covered, len(payload))
	}
	wantTags := []byte{TagDescriptor, TagName, TagInt4}
	if len(recs) != len(wantTags) {
		t.Fatalf("got %d records, want %d: %+v", len(recs), len(wantTags), recs)
	}
	for i, want := range wantTags {
		if recs[i].Tag != want {
			t.Errorf("record %d tag = %#x, want %#x", i, recs[i].Tag, want)
		}
		if recs[i].LengthFlagged {
			t.Errorf("record %d used the 0x00 length form; the capture uses the plain one", i)
		}
	}
}

func TestDecodeRecordsLengthFlaggedForm(t *testing.T) {
	payload := mustHex(t, pingResponsePadded)
	recs, covered := DecodeRecords(payload)
	if covered != len(payload) {
		t.Errorf("covered %d of %d bytes", covered, len(payload))
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(recs), recs)
	}
	if recs[0].Tag != TagDescriptor || recs[0].LengthFlagged {
		t.Errorf("first record = tag %#x flagged %v, want tag %#x plain", recs[0].Tag, recs[0].LengthFlagged, TagDescriptor)
	}
	padded := recs[1]
	if padded.Tag != TagPadded {
		t.Fatalf("second record tag = %#x, want %#x", padded.Tag, TagPadded)
	}
	if !padded.LengthFlagged {
		t.Error("second record should have used the 0x00 length form")
	}
	if len(padded.Value) != 80 {
		t.Fatalf("padded value = %d bytes, want 80", len(padded.Value))
	}
	if got := trimUTF16(padded.Value); got != "SAPLSRFC" {
		t.Errorf("padded text = %q, want %q", got, "SAPLSRFC")
	}
}

func TestDecodeRecordsResynchronises(t *testing.T) {
	// Two junk bytes that open no known record, then the real field. The
	// decoder must skip them and still find every record.
	payload := append([]byte{0xff, 0xfe}, mustHex(t, zDoubleTypedField)...)
	recs, covered := DecodeRecords(payload)
	if len(recs) != 3 {
		t.Fatalf("got %d records, want 3 after resynchronising: %+v", len(recs), recs)
	}
	if covered != len(payload)-2 {
		t.Errorf("covered %d bytes, want %d (all but the two junk bytes)", covered, len(payload)-2)
	}
	if recs[0].Offset != 2 {
		t.Errorf("first record Offset = %d, want 2", recs[0].Offset)
	}
}

func TestDecodeRecordsRejectsTruncatedValue(t *testing.T) {
	// 'C' announcing 200 bytes with only a few present must not be accepted,
	// and must not read past the slice.
	payload := []byte{TagChar, 200, 'a', 'b', 'c'}
	recs, covered := DecodeRecords(payload)
	if len(recs) != 0 {
		t.Errorf("got %d records, want none from a truncated value: %+v", len(recs), recs)
	}
	if covered != 0 {
		t.Errorf("covered = %d, want 0", covered)
	}
}

func TestEncodeRecordRoundTrip(t *testing.T) {
	value := []byte("open-rfc-go")
	enc, ok := EncodeRecord(TagChar, value)
	if !ok {
		t.Fatal("EncodeRecord refused a 11-byte value")
	}
	recs, covered := DecodeRecords(enc)
	if covered != len(enc) {
		t.Errorf("covered %d of %d bytes", covered, len(enc))
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	if recs[0].Tag != TagChar || string(recs[0].Value) != string(value) {
		t.Errorf("round trip = tag %#x value %q, want tag %#x value %q",
			recs[0].Tag, recs[0].Value, TagChar, value)
	}
}

func TestEncodeRecordRefusesUnrepresentable(t *testing.T) {
	if _, ok := EncodeRecord(TagChar, nil); ok {
		t.Error("an empty value should be refused, not encoded as a zero length")
	}
	if _, ok := EncodeRecord(TagChar, make([]byte, 256)); ok {
		t.Error("256 bytes cannot be expressed by the length byte and should be refused")
	}
}

func TestDecodedValueIsACopy(t *testing.T) {
	payload := mustHex(t, zDoubleTypedField)
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
	f.Add([]byte{TagChar, 200, 'a'})
	if b, err := hex.DecodeString(zDoubleTypedField); err == nil {
		f.Add(b)
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

// trimUTF16 decodes little-endian UTF-16 and trims the trailing blanks SAP pads
// fixed-width character fields with.
func trimUTF16(b []byte) string {
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		u = append(u, binary.LittleEndian.Uint16(b[i:i+2]))
	}
	s := string(utf16.Decode(u))
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == 0) {
		s = s[:len(s)-1]
	}
	return s
}
