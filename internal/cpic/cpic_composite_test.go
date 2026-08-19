// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/cpic.test.ts ("admits only the bounded rich initial
// RFCPING composite responses") at commit 847036d, Copyright 2026 Marian Zeis,
// licensed under the Apache License, Version 2.0. Rewritten for the testing
// package. This is the bounded-grammar test that the recurring-bug-class
// document exists for: variable-width coordinates (names, addresses) are
// asserted positively across many widths, while control coordinates stay
// pinned. See docs/provenance.md.

package cpic

import (
	"reflect"
	"testing"
)

func clone(fields []Field) []Field { return append([]Field(nil), fields...) }

func replaceIdx(fields []Field, i int, f Field) []Field {
	out := clone(fields)
	out[i] = f
	return out
}

func withoutIdx(fields []Field, i int) []Field {
	out := make([]Field, 0, len(fields)-1)
	out = append(out, fields[:i]...)
	out = append(out, fields[i+1:]...)
	return out
}

func insertAt(fields []Field, i int, f Field) []Field {
	out := make([]Field, 0, len(fields)+1)
	out = append(out, fields[:i]...)
	out = append(out, f)
	out = append(out, fields[i:]...)
	return out
}

func assertRichSuccess(t *testing.T, name string, fields []Field) {
	t.Helper()
	decoded, err := DecodeInitialLogonResponse(regularResponse(t, fields))
	if err != nil {
		t.Fatalf("%s: unexpected error %v", name, err)
	}
	if !decoded.Success || decoded.Status == nil || *decoded.Status != 0 || decoded.NegotiatedProtocolVersion != 0x0e0b {
		t.Fatalf("%s: decoded = %+v", name, decoded)
	}
	if !reflect.DeepEqual(decoded.Fields, tagLens(fields)) {
		t.Fatalf("%s: fields = %+v", name, decoded.Fields)
	}
}

func assertRichReject(t *testing.T, name string, fields []Field, substr string) {
	t.Helper()
	_, err := DecodeInitialLogonResponse(regularResponse(t, fields))
	if err == nil || !contains(err.Error(), substr) {
		t.Fatalf("%s = %v, want %q", name, err, substr)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestAdmitsBoundedRichComposite(t *testing.T) {
	pv := hx(t, "00000e0b")
	richFields := []Field{
		fld(TagProtocolVersion, pv), fld(TagCapabilities, make([]byte, 11)), fld(TagLogonStatus, []byte{0}),
		fld(TagSystemCodePage, make([]byte, 8)), fld(Tag(0x0450), make([]byte, 6)), fld(Tag(0x0451), make([]byte, 20)),
		fld(Tag(0x0452), make([]byte, 4)), fld(Tag(0x0453), make([]byte, 42)), fld(TagClientAddress, make([]byte, 30)),
		fld(Tag(0x0020), make([]byte, 92)), fld(Tag(0x0021), make([]byte, 20)), fld(TagPartnerSystem, make([]byte, 20)),
		fld(TagPartnerHost, make([]byte, 30)), fld(TagConnectionType, make([]byte, 2)), fld(TagKernelPatch, make([]byte, 8)),
		fld(TagKernelRelease, make([]byte, 8)), fld(TagProgram, make([]byte, 16)), fld(Tag(0x0150), make([]byte, 24)),
		fld(Tag(0x0151), make([]byte, 6)), fld(Tag(0x0152), make([]byte, 2)),
		fld(TagResponseStart, nil), fld(TagResponseContext, nil), fld(TagSession, make([]byte, 16)),
		fld(TagUnresolved0420, make([]byte, 4)), fld(TagCallContext, nil), fld(TagProgram, make([]byte, 80)),
		fld(Tag(0x0667), make([]byte, 8)), fld(Tag(0x0126), make([]byte, 4)), fld(TagEnd, nil),
	}
	assertRichSuccess(t, "rich", richFields)

	compactRichFields := []Field{
		fld(TagProtocolVersion, pv), fld(TagCapabilities, make([]byte, 11)), fld(TagLogonStatus, []byte{0}),
		fld(TagSystemCodePage, make([]byte, 8)), fld(Tag(0x0450), make([]byte, 6)), fld(Tag(0x0451), make([]byte, 20)),
		fld(Tag(0x0452), make([]byte, 4)), fld(Tag(0x0453), make([]byte, 20)), fld(TagClientAddress, make([]byte, 30)),
		fld(Tag(0x0020), make([]byte, 92)), fld(Tag(0x0021), make([]byte, 20)), fld(TagPartnerSystem, make([]byte, 20)),
		fld(TagPartnerHost, make([]byte, 34)), fld(TagConnectionType, make([]byte, 2)), fld(TagKernelPatch, make([]byte, 8)),
		fld(TagKernelRelease, make([]byte, 8)), fld(TagDestination, make([]byte, 22)), fld(TagProgram, make([]byte, 16)),
		fld(Tag(0x0150), make([]byte, 24)), fld(Tag(0x0151), make([]byte, 6)), fld(Tag(0x0152), make([]byte, 2)),
		fld(TagResponseStart, nil), fld(TagResponseContext, nil), fld(TagSession, make([]byte, 16)),
		fld(TagUnresolved0420, make([]byte, 4)), fld(TagCallContext, nil), fld(TagProgram, make([]byte, 80)),
		fld(Tag(0x0667), make([]byte, 8)), fld(TagEnd, nil),
	}
	assertRichSuccess(t, "compact", compactRichFields)

	compactShortDest := replaceIdx(compactRichFields, 16, fld(TagDestination, make([]byte, 20)))
	assertRichSuccess(t, "compact short destination", compactShortDest)

	compactWithControl := insertAt(compactRichFields, len(compactRichFields)-1, fld(Tag(0x0126), make([]byte, 4)))
	assertRichSuccess(t, "compact with embedded control", compactWithControl)

	compactWithoutDest := filterOutTag(compactRichFields, uint16(TagDestination))
	assertRichSuccess(t, "compact without destination", compactWithoutDest)

	callStatusOnly := []Field{
		fld(TagProtocolVersion, pv), fld(TagCapabilities, make([]byte, 11)), fld(TagSystemCodePage, make([]byte, 8)),
		fld(TagClientAddress, make([]byte, 30)), fld(TagPartnerSystem, make([]byte, 18)), fld(TagPartnerHost, make([]byte, 34)),
		fld(TagConnectionType, make([]byte, 2)), fld(TagKernelPatch, make([]byte, 8)), fld(TagKernelRelease, make([]byte, 8)),
		fld(TagDestination, make([]byte, 22)), fld(TagProgram, make([]byte, 16)), fld(Tag(0x0150), make([]byte, 24)),
		fld(Tag(0x0151), make([]byte, 6)), fld(Tag(0x0152), make([]byte, 2)),
		fld(TagResponseStart, nil), fld(TagResponseContext, nil), fld(TagSession, make([]byte, 16)),
		fld(TagUnresolved0420, make([]byte, 4)), fld(TagCallContext, nil), fld(TagProgram, make([]byte, 80)),
		fld(Tag(0x0667), make([]byte, 8)), fld(TagEnd, nil),
	}
	assertRichSuccess(t, "call-status-only", callStatusOnly)
	callStatusOnlyShortDest := replaceIdx(callStatusOnly, 9, fld(TagDestination, make([]byte, 20)))
	assertRichSuccess(t, "call-status-only short destination", callStatusOnlyShortDest)

	assertRichReject(t, "call-status-only missing preamble coordinate", withoutIdx(callStatusOnly, 5), "composite shape")
	assertRichReject(t, "call-status-only malformed call status", replaceIdx(callStatusOnly, 17, fld(TagUnresolved0420, make([]byte, 3))), "malformed call status")
	assertRichReject(t, "call-status-only nonzero call status", replaceIdx(callStatusOnly, 17, fld(TagUnresolved0420, hx(t, "00000001"))), "nonzero call status")

	compactRejected, err := DecodeInitialLogonResponse(regularResponse(t, replaceIdx(compactRichFields, 2, fld(TagLogonStatus, []byte{1}))))
	if err != nil || compactRejected.Success || *compactRejected.Status != 1 {
		t.Fatalf("compact rejected = %+v, %v", compactRejected, err)
	}

	assertRichReject(t, "compact missing opaque preamble control", withoutIdx(compactRichFields, 17), "composite shape")
	assertRichReject(t, "compact unknown embedded control", replaceIdx(compactRichFields, 27, fld(Tag(0x7777), make([]byte, 8))), "composite shape")
	assertRichReject(t, "compact embedded program changed to destination", replaceIdx(compactRichFields, 26, fld(TagDestination, compactRichFields[26].Value)), "duplicate")
	assertRichReject(t, "compact duplicate embedded control", insertAt(compactWithControl, indexOfTag(compactWithControl, 0x0126)+1, fld(Tag(0x0126), make([]byte, 4))), "duplicate field")
	{
		reordered := clone(compactShortDest)
		reordered[16], reordered[17] = reordered[17], reordered[16]
		assertRichReject(t, "short destination reorder", reordered, "composite shape")
	}
	assertRichReject(t, "short destination duplicate", insertAt(compactShortDest, 17, fld(Tag(compactShortDest[16].Tag), append([]byte(nil), compactShortDest[16].Value...))), "duplicate field")
	assertRichReject(t, "compact malformed embedded call status", replaceIdx(compactRichFields, 24, fld(TagUnresolved0420, make([]byte, 3))), "malformed call status")
	assertRichReject(t, "compact nonzero embedded call status", replaceIdx(compactRichFields, 24, fld(TagUnresolved0420, hx(t, "00000001"))), "nonzero call status")
	assertRichReject(t, "destination-free missing preamble program", withoutIdx(compactWithoutDest, 16), "composite shape")
	assertRichReject(t, "destination-free embedded opaque length drift", replaceIdx(compactWithoutDest, 26, fld(Tag(compactWithoutDest[26].Tag), make([]byte, 7))), "composite shape")

	// Variable-width coordinates: asserted POSITIVELY across widths.
	for _, width := range []int{1, 18, 19, 20, 21, 22, 23, 64, 255} {
		assertRichSuccess(t, "call-status-only PartnerHost width", replaceIdx(callStatusOnly, 5, fld(TagPartnerHost, make([]byte, width))))
		assertRichSuccess(t, "call-status-only Destination width", replaceIdx(callStatusOnlyShortDest, 9, fld(TagDestination, make([]byte, width))))
		assertRichSuccess(t, "compact 0x0453 width", replaceIdx(compactRichFields, 7, fld(Tag(0x0453), make([]byte, width))))
		assertRichSuccess(t, "compact Destination width", replaceIdx(compactShortDest, 16, fld(TagDestination, make([]byte, width))))
	}

	assertRichSuccess(t, "call-status-only with one-byte status", insertAt(callStatusOnly, 2, fld(TagLogonStatus, []byte{0})))
	assertRichSuccess(t, "short destination with embedded control", insertAt(compactShortDest, len(compactShortDest)-1, fld(Tag(0x0126), make([]byte, 4))))
	assertRichSuccess(t, "destination-free with embedded control", insertAt(compactWithoutDest, len(compactWithoutDest)-1, fld(Tag(0x0126), make([]byte, 4))))

	swapped := clone(richFields)
	swapped[5], swapped[6] = swapped[6], swapped[5]
	assertRichReject(t, "preamble order", swapped, "composite shape")
	assertRichReject(t, "duplicate preamble field", insertAt(richFields, 6, richFields[5]), "duplicate")
	assertRichReject(t, "third Program field", insertAt(richFields, 26, richFields[25]), "duplicate")
	assertRichReject(t, "nonzero embedded call status", replaceIdx(richFields, 23, fld(TagUnresolved0420, hx(t, "00000001"))), "nonzero call status")
	assertRichReject(t, "malformed embedded call status", replaceIdx(richFields, 23, fld(TagUnresolved0420, make([]byte, 3))), "malformed call status")
	assertRichReject(t, "unknown embedded control", replaceIdx(richFields, 27, fld(Tag(0x7777), make([]byte, 4))), "composite shape")
	assertRichReject(t, "missing embedded Program", withoutIdx(richFields, 25), "composite shape")

	ordinary := concat(hx(t, "05000000"), mustChain(t, TagResponseStart, []Field{
		fld(TagUnresolved0420, make([]byte, 4)), fld(Tag(0x0126), make([]byte, 4)), fld(TagEnd, nil),
	}), []byte{0xff, 0xff})
	if _, err := DecodeFunctionResultFields(ordinary); err == nil || !contains(err.Error(), "unknown tag 0x0126") {
		t.Fatalf("ordinary 0x0126 = %v", err)
	}
}

func filterOutTag(fields []Field, tag uint16) []Field {
	var out []Field
	for _, f := range fields {
		if f.Tag != tag {
			out = append(out, f)
		}
	}
	return out
}

func indexOfTag(fields []Field, tag uint16) int {
	for i, f := range fields {
		if f.Tag == tag {
			return i
		}
	}
	return -1
}
