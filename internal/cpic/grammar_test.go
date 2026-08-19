// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/cpic-initial-logon-grammar.test.ts at commit
// 847036d, Copyright 2026 Marian Zeis, Apache-2.0. Rewritten for the testing
// package. The rule/stage are read through StructureRuleOf; the builder's
// "number | null" options become explicit fields plus omit-flags. See
// docs/provenance.md.

package cpic

import (
	"strings"
	"testing"
)

const regularPrefixHex = "010100080101010504010003"
const errorPrefixHex = "010100080101010101010000"

func gfield(tag Tag, n int) Field { return Field{Tag: uint16(tag), Value: make([]byte, n)} }

func respG(t *testing.T, fields []Field, prefixHex string) []byte {
	t.Helper()
	return concat(hx(t, prefixHex), mustChain(t, TagStart, fields), []byte{0xff, 0xff})
}

type pOpts struct {
	logonStatus       int
	omitLogonStatus   bool
	vendorControl     int
	omitVendorControl bool
	clientAddress     int
	noExtraControls   bool
	partnerSystem     int
	partnerHost       int
	destination       int
	omitDestination   bool
	program           int
}

func defaultPOpts() pOpts {
	return pOpts{logonStatus: 0, vendorControl: 20, clientAddress: 30, partnerSystem: 20, partnerHost: 34, destination: 22, program: 16}
}

func preambleFields(t *testing.T, o pOpts) []Field {
	f := []Field{fld(TagProtocolVersion, hx(t, "00000e0b")), gfield(TagCapabilities, 11)}
	if !o.omitLogonStatus {
		f = append(f, fld(TagLogonStatus, []byte{byte(o.logonStatus)}))
	}
	f = append(f, gfield(TagSystemCodePage, 8))
	if !o.omitVendorControl {
		f = append(f, gfield(Tag(0x0450), 6), gfield(Tag(0x0451), 20), gfield(Tag(0x0452), 4), gfield(Tag(0x0453), o.vendorControl))
	}
	f = append(f, gfield(TagClientAddress, o.clientAddress))
	if !o.noExtraControls {
		f = append(f, gfield(Tag(0x0020), 92), gfield(Tag(0x0021), 20))
	}
	f = append(f, gfield(TagPartnerSystem, o.partnerSystem), gfield(TagPartnerHost, o.partnerHost),
		gfield(TagConnectionType, 2), gfield(TagKernelPatch, 8), gfield(TagKernelRelease, 8))
	if !o.omitDestination {
		f = append(f, gfield(TagDestination, o.destination))
	}
	f = append(f, gfield(TagProgram, o.program), gfield(Tag(0x0150), 24), gfield(Tag(0x0151), 6), gfield(Tag(0x0152), 2))
	return f
}

func embeddedFields(control bool, program int) []Field {
	f := []Field{gfield(TagResponseStart, 0), gfield(TagResponseContext, 0), gfield(TagSession, 16),
		gfield(TagUnresolved0420, 4), gfield(TagCallContext, 0), gfield(TagProgram, program), gfield(Tag(0x0667), 8)}
	if control {
		f = append(f, gfield(Tag(0x0126), 4))
	}
	return append(f, gfield(TagEnd, 0))
}

func compositeFields(t *testing.T, o pOpts, control bool, program int) []Field {
	return append(preambleFields(t, o), embeddedFields(control, program)...)
}

func TestGrammarAdmitsRetiredAllowlist(t *testing.T) {
	type entry struct {
		name   string
		fields []Field
	}
	rich := func(o pOpts) pOpts { return o }
	c0 := defaultPOpts()
	c0.vendorControl, c0.partnerHost, c0.omitDestination = 42, 30, true
	c3 := defaultPOpts()
	c3.destination = 20
	c4 := defaultPOpts()
	c4.omitDestination = true
	c5 := defaultPOpts()
	c5.omitLogonStatus, c5.omitVendorControl, c5.noExtraControls, c5.partnerSystem = true, true, true, 18
	c6 := c5
	c6.destination = 20
	entries := []entry{
		{"#0", compositeFields(t, rich(c0), true, 80)},
		{"#1", compositeFields(t, defaultPOpts(), false, 80)},
		{"#2", compositeFields(t, defaultPOpts(), true, 80)},
		{"#3", compositeFields(t, c3, false, 80)},
		{"#4", compositeFields(t, c4, false, 80)},
		{"#5", compositeFields(t, c5, false, 80)},
		{"#6", compositeFields(t, c6, false, 80)},
	}
	for _, e := range entries {
		decoded, err := DecodeInitialLogonResponse(respG(t, e.fields, regularPrefixHex))
		if err != nil || !decoded.Success || decoded.NegotiatedProtocolVersion != 0x0e0b || len(decoded.Fields) != len(e.fields) {
			t.Fatalf("%s: decoded=%+v err=%v", e.name, decoded, err)
		}
	}
}

func TestGrammarAdmitsRefusedReplies(t *testing.T) {
	c := defaultPOpts()
	c.vendorControl, c.destination = 20, 20
	fields := compositeFields(t, c, true, 80)
	if len(fields) != 30 {
		t.Fatalf("len = %d", len(fields))
	}
	if d, err := DecodeInitialLogonResponse(respG(t, fields, regularPrefixHex)); err != nil || !d.Success || *d.Status != 0 {
		t.Fatalf("2026-08-05 = %+v, %v", d, err)
	}

	nw := defaultPOpts()
	nw.omitLogonStatus, nw.omitVendorControl, nw.noExtraControls, nw.partnerSystem, nw.destination = true, true, true, 18, 20
	nwFields := compositeFields(t, nw, false, 80)
	if len(nwFields) != 22 {
		t.Fatalf("nw len = %d", len(nwFields))
	}
	if d, err := DecodeInitialLogonResponse(respG(t, nwFields, regularPrefixHex)); err != nil || !d.Success {
		t.Fatalf("2026-08-04 = %+v, %v", d, err)
	}
}

func TestGrammarTextCoordinateAnyLegalLength(t *testing.T) {
	ref, err := DecodeInitialLogonResponse(respG(t, compositeFields(t, defaultPOpts(), false, 80), regularPrefixHex))
	if err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{1, 2, 7, 16, 19, 20, 21, 22, 23, 40, 64, 120, 200, 255} {
		for _, tag := range []string{"destination", "partnerHost", "partnerSystem", "program"} {
			o := defaultPOpts()
			switch tag {
			case "destination":
				o.destination = width
			case "partnerHost":
				o.partnerHost = width
			case "partnerSystem":
				o.partnerSystem = width
			case "program":
				o.program = width
			}
			d, err := DecodeInitialLogonResponse(respG(t, compositeFields(t, o, false, 80), regularPrefixHex))
			if err != nil || d.Success != ref.Success || *d.Status != *ref.Status || d.NegotiatedProtocolVersion != ref.NegotiatedProtocolVersion || len(d.Fields) != len(ref.Fields) {
				t.Fatalf("%s=%d: %+v, %v", tag, width, d, err)
			}
		}
	}
}

func TestGrammarTextOutsideBoundFailsClosed(t *testing.T) {
	for _, width := range []int{0, 256, 300} {
		o := defaultPOpts()
		o.destination = width
		_, err := DecodeInitialLogonResponse(respG(t, compositeFields(t, o, false, 80), regularPrefixHex))
		if StructureRuleOf(err) != "unsupported-field-zero-logon-status" {
			t.Fatalf("destination %d = %v (rule %q)", width, err, StructureRuleOf(err))
		}
	}
}

func TestGrammarUnknownTagFailsClosed(t *testing.T) {
	fields := compositeFields(t, defaultPOpts(), false, 80)
	withUnknown := append(append(clone(fields[:6]), gfield(Tag(0x0999), 4)), fields[6:]...)
	if _, err := DecodeInitialLogonResponse(respG(t, withUnknown, regularPrefixHex)); StructureRuleOf(err) == "" {
		t.Fatalf("unknown tag = %v", err)
	}
}

func TestGrammarControlCoordinatesExactWidths(t *testing.T) {
	cases := []struct {
		tag Tag
		n   int
	}{
		{TagCapabilities, 12}, {TagSystemCodePage, 9}, {TagConnectionType, 3},
		{TagKernelRelease, 7}, {TagSession, 15}, {Tag(0x0450), 7},
	}
	for _, c := range cases {
		fields := compositeFields(t, defaultPOpts(), false, 80)
		for i := range fields {
			if fields[i].Tag == uint16(c.tag) {
				fields[i] = gfield(c.tag, c.n)
			}
		}
		if _, err := DecodeInitialLogonResponse(respG(t, fields, regularPrefixHex)); err == nil {
			t.Fatalf("control %#x width %d accepted", uint16(c.tag), c.n)
		}
	}
}

func TestGrammarReorderDuplicationTruncation(t *testing.T) {
	fields := compositeFields(t, defaultPOpts(), false, 80)
	swapped := clone(fields)
	swapped[0], swapped[1] = swapped[1], swapped[0]
	if _, err := DecodeInitialLogonResponse(respG(t, swapped, regularPrefixHex)); err == nil {
		t.Fatal("reorder accepted")
	}
	duplicated := append(append(clone(fields[:2]), gfield(TagCapabilities, 11)), fields[2:]...)
	if _, err := DecodeInitialLogonResponse(respG(t, duplicated, regularPrefixHex)); err == nil {
		t.Fatal("duplicate accepted")
	}
	withoutCall := filterOutTag(fields, uint16(TagUnresolved0420))
	if _, err := DecodeInitialLogonResponse(respG(t, withoutCall, regularPrefixHex)); err == nil {
		t.Fatal("missing call status accepted")
	}
	trailing := append(clone(fields), gfield(Tag(0x0667), 8))
	if _, err := DecodeInitialLogonResponse(respG(t, trailing, regularPrefixHex)); err == nil {
		t.Fatal("trailing accepted")
	}
}

func TestGrammarNonzeroLogonStatusPreserved(t *testing.T) {
	o := defaultPOpts()
	o.logonStatus = 3
	d, err := DecodeInitialLogonResponse(respG(t, compositeFields(t, o, false, 80), regularPrefixHex))
	if err != nil || d.Success || *d.Status != 3 {
		t.Fatalf("decoded = %+v, %v", d, err)
	}
}

func TestGrammarNonzeroEmbeddedCallStatusFailsClosed(t *testing.T) {
	fields := compositeFields(t, defaultPOpts(), false, 80)
	for i := range fields {
		if fields[i].Tag == uint16(TagUnresolved0420) {
			fields[i] = fld(TagUnresolved0420, hx(t, "00000001"))
		}
	}
	if _, err := DecodeInitialLogonResponse(respG(t, fields, regularPrefixHex)); StructureRuleOf(err) != "nonzero-call-status" {
		t.Fatalf("nonzero embedded = %v (rule %q)", err, StructureRuleOf(err))
	}
}

func TestGrammarErrorClassRejectionReachesCaller(t *testing.T) {
	errorFields := []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")), gfield(TagCapabilities, 11), gfield(TagSystemCodePage, 8),
		gfield(TagClientAddress, 30), gfield(TagPartnerSystem, 18), gfield(TagPartnerHost, 34),
		gfield(TagConnectionType, 2), gfield(TagKernelPatch, 8), gfield(TagKernelRelease, 8),
		gfield(TagDestination, 20), gfield(TagProgram, 16), gfield(TagResponseStart, 0),
		fld(tagAbapErrorMessage, utf16leBytes("Name or password is incorrect")), gfield(TagEnd, 0),
	}
	if len(errorFields) != 14 {
		t.Fatalf("len = %d", len(errorFields))
	}
	d, err := DecodeInitialLogonResponse(respG(t, errorFields, errorPrefixHex))
	if err != nil || d.Success || d.Rejection == nil {
		t.Fatalf("decoded = %+v, %v", d, err)
	}
	if d.Rejection.Outcome != "abapMessage" || d.Rejection.Text != "Name or password is incorrect" || d.Rejection.ExceptionKey != "" || d.Rejection.RuntimeID != "" {
		t.Fatalf("rejection = %+v", d.Rejection)
	}
}

func TestGrammarErrorClassRejectionKeepsMessageIdentity(t *testing.T) {
	errorFields := []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")), gfield(TagCapabilities, 11), gfield(TagSystemCodePage, 8),
		gfield(TagClientAddress, 30), gfield(TagPartnerSystem, 18), gfield(TagPartnerHost, 34),
		gfield(TagConnectionType, 2), gfield(TagKernelPatch, 8), gfield(TagKernelRelease, 8),
		gfield(TagDestination, 20), gfield(TagProgram, 16), gfield(TagResponseStart, 0),
		fld(Tag(0x0415), utf16leBytes("00")), fld(Tag(0x0416), utf16leBytes("E")), fld(Tag(0x0417), utf16leBytes("054")),
		gfield(TagEnd, 0),
	}
	d, err := DecodeInitialLogonResponse(respG(t, errorFields, errorPrefixHex))
	if err != nil || d.Success || d.Rejection == nil {
		t.Fatalf("decoded = %+v, %v", d, err)
	}
	if d.Rejection.MessageClass != "00" || d.Rejection.MessageType != "E" || d.Rejection.MessageNumber != "054" {
		t.Fatalf("rejection = %+v", d.Rejection)
	}
}

var _ = strings.Contains
