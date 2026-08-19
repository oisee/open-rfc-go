// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/cpic.test.ts at commit 847036d, Copyright 2026
// Marian Zeis, Apache-2.0. Rewritten for the testing package. The reverse
// enum-name lookup (CpicTag[0x0420]) and the Proxy "whole-input read" and
// "value in field" assertions have no Go analogue and are dropped; the parse
// stage and structure rule are read through ParseStageOf / StructureRuleOf
// instead of the Symbol-keyed projectors. See docs/provenance.md.

package cpic

import (
	"reflect"
	"strings"
	"testing"
)

func errorResponse(t *testing.T, fields []Field) []byte {
	t.Helper()
	return concat(hx(t, "010100080101010101010000"), mustChain(t, TagStart, fields), []byte{0xff, 0xff})
}

func TestRejectsContradictoryRegularEnvelopes(t *testing.T) {
	protocol := fld(TagProtocolVersion, hx(t, "00000e0b"))
	status := fld(TagLogonStatus, []byte{0})
	cases := []struct {
		name   string
		fields []Field
		substr string
	}{
		{"error discriminator", []Field{protocol, status, fld(Tag(0x0403), utf16leBytes("FAIL")), fld(TagEnd, nil)}, "unsupported field 0x0403 (8 bytes) at index 2"},
		{"unknown field", []Field{protocol, status, fld(Tag(0x7777), nil), fld(TagEnd, nil)}, "unsupported field 0x7777 (0 bytes) at index 2"},
		{"duplicate protocol", []Field{protocol, protocol, status, fld(TagEnd, nil)}, "protocol version"},
		{"nonempty End", []Field{protocol, status, fld(TagEnd, []byte{1})}, "invalid End field"},
		{"misplaced Start", []Field{protocol, fld(TagStart, nil), status, fld(TagEnd, nil)}, "invalid Start field"},
		{"nonempty Start", []Field{fld(TagStart, []byte{1}), protocol, status, fld(TagEnd, nil)}, "invalid Start field"},
	}
	for _, c := range cases {
		if _, err := DecodeInitialLogonResponse(regularResponse(t, c.fields)); err == nil || !strings.Contains(err.Error(), c.substr) {
			t.Fatalf("%s = %v", c.name, err)
		}
	}
}

func TestClassifiesNw750TerminalErrorEnvelope(t *testing.T) {
	fields := []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagCapabilities, make([]byte, 11)),
		fld(TagSystemCodePage, make([]byte, 4)), fld(TagClientAddress, make([]byte, 15)),
		fld(TagPartnerSystem, make([]byte, 9)), fld(TagPartnerHost, make([]byte, 17)),
		fld(TagConnectionType, make([]byte, 1)), fld(TagKernelPatch, make([]byte, 4)),
		fld(TagKernelRelease, make([]byte, 4)), fld(TagDestination, make([]byte, 10)),
		fld(TagProgram, make([]byte, 8)), fld(TagResponseStart, nil),
		fld(tagAbapErrorMessage, utf16leBytes("Synthetic logon rejection")), fld(TagEnd, nil),
	}
	decoded, err := DecodeInitialLogonResponse(errorResponse(t, fields))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Success || decoded.Rejection == nil {
		t.Fatalf("decoded = %+v", decoded)
	}
	wantRej := &Rejection{Outcome: "abapMessage", Text: "Synthetic logon rejection"}
	if !reflect.DeepEqual(decoded.Rejection, wantRej) || decoded.NegotiatedProtocolVersion != 0x0e0b {
		t.Fatalf("rejection = %+v", decoded.Rejection)
	}
	if !reflect.DeepEqual(decoded.Fields, tagLens(fields)) {
		t.Fatalf("fields = %+v", decoded.Fields)
	}
}

func TestRejectsMalformedTerminalErrorEnvelopes(t *testing.T) {
	preamble := []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagCapabilities, make([]byte, 11)),
		fld(TagSystemCodePage, make([]byte, 4)), fld(TagClientAddress, make([]byte, 15)),
		fld(TagPartnerSystem, make([]byte, 9)), fld(TagPartnerHost, make([]byte, 17)),
		fld(TagConnectionType, make([]byte, 1)), fld(TagKernelPatch, make([]byte, 4)),
		fld(TagKernelRelease, make([]byte, 4)), fld(TagDestination, make([]byte, 10)),
		fld(TagProgram, make([]byte, 8)), fld(TagResponseStart, nil),
	}
	msg := fld(tagAbapErrorMessage, utf16leBytes("Rejected"))
	end := fld(TagEnd, nil)

	outOfOrder := append([]Field{preamble[1], preamble[0]}, preamble[2:]...)
	outOfOrder = append(outOfOrder, msg, end)
	dupTag := append(clone(preamble), preamble[0], msg, end)
	noOutcome := append(clone(preamble), fld(TagUnresolved0420, make([]byte, 4)), end)

	cases := []struct {
		name   string
		fields []Field
		substr string
	}{
		{"out-of-order preamble", outOfOrder, "invalid preamble"},
		{"duplicate preamble tag", dupTag, "duplicate preamble fields"},
		{"no rejected outcome", noOutcome, "lacks a rejected outcome"},
	}
	for _, c := range cases {
		if _, err := DecodeInitialLogonResponse(errorResponse(t, c.fields)); err == nil || !strings.Contains(err.Error(), c.substr) {
			t.Fatalf("%s = %v", c.name, err)
		}
	}
}

func TestEncodesCaptureSizedFunctionRequest(t *testing.T) {
	encoded, err := EncodeFunctionRequest(FunctionRequestInput{FunctionName: "RFC_PING", SessionID: bytesRep(0x5a, 16)})
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != 129 {
		t.Fatalf("len = %d", len(encoded))
	}
	if hexs(encoded[0:12]) != "010100080301010504010003" || hexs(encoded[len(encoded)-10:]) != "ffff0000007900008500" {
		t.Fatalf("framing = %s .. %s", hexs(encoded[0:12]), hexs(encoded[len(encoded)-10:]))
	}
	decoded, err := DecodeFieldChainPrefix(encoded[12:], uint16(TagStart), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]int{{0x0103, 4}, {0x0106, 11}, {0x0337, 0}, {0x0514, 16}, {0x0502, 0}, {0x000b, 6}, {0x0102, 16}, {0x0512, 0}, {0xffff, 0}}
	if len(decoded.Fields) != len(want) {
		t.Fatalf("field count %d", len(decoded.Fields))
	}
	for i, w := range want {
		if int(decoded.Fields[i].Tag) != w[0] || len(decoded.Fields[i].Value) != w[1] {
			t.Fatalf("field %d = %+v want %v", i, decoded.Fields[i], w)
		}
	}
	if fromUTF16LE(decoded.Fields[6].Value) != "RFC_PING" {
		t.Fatalf("function name = %q", fromUTF16LE(decoded.Fields[6].Value))
	}
}

func TestDecodesRegularRfcSuccessResponse(t *testing.T) {
	response := concat(hx(t, "05000000"), mustChain(t, TagResponseStart, []Field{
		fld(TagResponseContext, nil), fld(TagSession, bytesRep(1, 16)), fld(TagUnresolved0420, make([]byte, 4)),
		fld(TagCallContext, nil), fld(TagEnd, nil),
	}), []byte{0xff, 0xff})
	decoded, err := DecodeFunctionResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Success || decoded.Outcome != "success" || decoded.Status == nil || *decoded.Status != 0 {
		t.Fatalf("decoded = %+v", decoded)
	}
	want := []TagLen{{0x0503, 0}, {0x0514, 16}, {0x0420, 4}, {0x0512, 0}, {0xffff, 0}}
	if !reflect.DeepEqual(decoded.Fields, want) {
		t.Fatalf("fields = %+v", decoded.Fields)
	}
}

func padEnd(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func TestEncodesCaptureVerifiedCutMetadataRequest(t *testing.T) {
	encoded, err := EncodeCutFunctionRequest(CutFunctionRequestInput{
		FunctionName:     "RFC_GET_FUNCTION_INTERFACE",
		RequestedOutputs: []string{"REMOTE_BASXML_SUPPORTED", "REMOTE_CALL", "UPDATE_TASK", "PARAMS", "RESUMABLE_EXCEPTIONS"},
		Imports: []NamedValue{
			{Name: "FUNCNAME", Value: utf16leBytes(padEnd("STFC_CONNECTION", 30))},
			{Name: "NONE_UNICODE_LENGTH", Value: utf16leBytes("X")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != 408 || hexs(encoded[0:4]) != "05020000" || hexs(encoded[len(encoded)-10:]) != "ffff0000019000008500" {
		t.Fatalf("framing len=%d %s .. %s", len(encoded), hexs(encoded[0:4]), hexs(encoded[len(encoded)-10:]))
	}
	decoded, err := DecodeFieldChainPrefix(encoded[4:], uint16(TagContextEnd), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]int{{0x000b, 6}, {0x0102, 52}, {0x0512, 0}, {0x0205, 46}, {0x0205, 22}, {0x0205, 22}, {0x0205, 12}, {0x0205, 40}, {0x0201, 16}, {0x0203, 60}, {0x0201, 38}, {0x0203, 2}, {0xffff, 0}}
	if len(decoded.Fields) != len(want) {
		t.Fatalf("field count %d", len(decoded.Fields))
	}
	for i, w := range want {
		if int(decoded.Fields[i].Tag) != w[0] || len(decoded.Fields[i].Value) != w[1] {
			t.Fatalf("field %d = %+v want %v", i, decoded.Fields[i], w)
		}
	}
}

func TestRejectsAmbiguousCutRecords(t *testing.T) {
	if _, err := EncodeCutFunctionRequest(CutFunctionRequestInput{FunctionName: "RFC_PING", RequestedOutputs: []string{"RESULT", "RESULT"}}); err == nil || !strings.Contains(err.Error(), "duplicate requested output RESULT") {
		t.Fatalf("dup output = %v", err)
	}
	if _, err := EncodeCutFunctionRequest(CutFunctionRequestInput{FunctionName: "RFC_PING", Imports: []NamedValue{{Name: "INPUT", Value: nil}, {Name: "INPUT", Value: nil}}}); err == nil || !strings.Contains(err.Error(), "duplicate import INPUT") {
		t.Fatalf("dup import = %v", err)
	}
}

func TestEncodesCutTableInputs(t *testing.T) {
	encoded, err := EncodeCutFunctionRequest(CutFunctionRequestInput{
		FunctionName: "Z_TABLE_CALL",
		Tables:       []Table{{Name: "ROWS", RowByteLength: 4, Rows: [][]byte{hx(t, "01020304"), hx(t, "05060708")}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeFieldChainPrefix(encoded[4:], uint16(TagContextEnd), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		t.Fatal(err)
	}
	off := indexOfTag(decoded.Fields, uint16(TagTableName))
	if fromUTF16LE(decoded.Fields[off].Value) != "ROWS" || hexs(decoded.Fields[off+1].Value) != "0000000400000002" {
		t.Fatalf("table header wrong: %s / %s", fromUTF16LE(decoded.Fields[off].Value), hexs(decoded.Fields[off+1].Value))
	}
	if decoded.Fields[off+2].Tag != uint16(TagTableCompr) || hexs(decoded.Fields[off+2].Value) != "01020304" ||
		decoded.Fields[off+3].Tag != uint16(TagTableCompr) || hexs(decoded.Fields[off+3].Value) != "05060708" {
		t.Fatalf("rows wrong")
	}
	if _, err := EncodeCutFunctionRequest(CutFunctionRequestInput{FunctionName: "Z_TABLE_CALL", Tables: []Table{{Name: "ROWS", RowByteLength: 4, Rows: [][]byte{make([]byte, 3)}}}}); err == nil || !strings.Contains(err.Error(), "ROWS row 0 contains 3 bytes; expected 4") {
		t.Fatalf("row width = %v", err)
	}
}

func TestReturnsClonedApplicationFields(t *testing.T) {
	secret := utf16leBytes("sensitive")
	response := concat(hx(t, "05000000"), mustChain(t, TagResponseStart, []Field{
		fld(TagResponseContext, nil), fld(TagSession, bytesRep(1, 16)), fld(TagUnresolved0420, make([]byte, 4)),
		fld(TagCallContext, nil), fld(TagParameterName, utf16leBytes("RESULT")), fld(TagParameterValue, secret), fld(TagEnd, nil),
	}), []byte{0xff, 0xff})
	decoded, err := DecodeFunctionResultFields(response)
	if err != nil || !decoded.Success || !equalBytes(decoded.Fields[5].Value, secret) {
		t.Fatalf("decoded = %+v, %v", decoded, err)
	}
	for i := range response {
		response[i] = 0
	}
	if !equalBytes(decoded.Fields[5].Value, secret) {
		t.Fatal("decoder aliased caller input")
	}
}

func TestClassifiesDeclaredAbapException(t *testing.T) {
	response := concat(hx(t, "05000000"), mustChain(t, TagResponseStart, []Field{
		fld(Tag(0x0415), utf16leBytes("SR")), fld(Tag(0x0416), utf16leBytes("E")), fld(Tag(0x0417), utf16leBytes("006")),
		fld(Tag(0x0411), utf16leBytes("Method = 1")), fld(Tag(0x0401), utf16leBytes("RAISE_EXCEPTION")), fld(TagEnd, nil),
	}), []byte{0xff, 0xff})
	decoded, err := DecodeFunctionResultFields(response)
	if err != nil {
		t.Fatal(err)
	}
	f := decoded.Envelope.Facts
	if decoded.Envelope.Outcome != "abapException" || f.ExceptionKey != "RAISE_EXCEPTION" || f.MessageClass != "SR" || f.MessageType != "E" || f.MessageNumber != "006" || f.MessageV1 != "Method = 1" {
		t.Fatalf("facts = %+v", f)
	}
	if decoded.Success || decoded.Status != nil {
		t.Fatalf("success/status = %v/%v", decoded.Success, decoded.Status)
	}
	resp, err := DecodeFunctionResponse(response)
	if err != nil || resp.ExceptionKey != "RAISE_EXCEPTION" {
		t.Fatalf("response exceptionKey = %q, %v", resp.ExceptionKey, err)
	}
}

func TestKeepsErrorTagsAndApplicationFields(t *testing.T) {
	response := concat(hx(t, "05000000"), mustChain(t, TagResponseStart, []Field{
		fld(TagParameterName, utf16leBytes("RESULT")), fld(TagParameterValue, utf16leBytes("retained")),
		fld(Tag(0x0402), utf16leBytes("Runtime text")), fld(Tag(0x0403), utf16leBytes("RUNTIME_ID")),
		fld(Tag(0x0418), utf16leBytes("private stack")), fld(TagEnd, nil),
	}), []byte{0xff, 0xff})
	decoded, err := DecodeFunctionResultFields(response)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Success || decoded.Envelope.Outcome != "abapRuntime" || decoded.Envelope.Facts.RuntimeID != "RUNTIME_ID" || decoded.Envelope.Facts.CallStack != "private stack" {
		t.Fatalf("decoded = %+v", decoded.Envelope)
	}
	if fromUTF16LE(decoded.Fields[1].Value) != "retained" {
		t.Fatalf("retained = %q", fromUTF16LE(decoded.Fields[1].Value))
	}
}

func TestRetainsErrorState0420AsProvenance(t *testing.T) {
	response := concat(hx(t, "05000000"), mustChain(t, TagResponseStart, []Field{
		fld(Tag(0x0401), utf16leBytes("RAISE_EXCEPTION")), fld(TagUnresolved0420, hx(t, "deadbeef")), fld(TagEnd, nil),
	}), []byte{0xff, 0xff})
	decoded, err := DecodeFunctionResultFields(response)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Envelope.Outcome != "abapException" || decoded.Status != nil {
		t.Fatalf("decoded = %+v", decoded)
	}
	if len(decoded.Envelope.Facts.Unresolved0420) != 1 || decoded.Envelope.Facts.Unresolved0420[0].ValueHex != "deadbeef" || decoded.Envelope.Facts.Unresolved0420[0].Ordinal != 1 {
		t.Fatalf("unresolved = %+v", decoded.Envelope.Facts.Unresolved0420)
	}
}

func TestAcceptsResetDoneOnlyInResetState(t *testing.T) {
	resp := func(fields []Field) []byte {
		return concat(hx(t, "05000000"), mustChain(t, TagResponseStart, fields), []byte{0xff, 0xff})
	}
	successfulReset := resp([]Field{fld(TagResponseContext, nil), fld(TagUnresolved0420, make([]byte, 4)), fld(TagRfcServerResetDn, nil), fld(TagEnd, nil)})
	if d, err := DecodeResetServerContextResultFields(successfulReset); err != nil || !d.Success {
		t.Fatalf("reset = %+v, %v", d, err)
	}
	nw := resp([]Field{fld(TagResponseContext, nil), fld(TagUnresolved0420, make([]byte, 4)), fld(TagEnd, nil)})
	if d, err := DecodeResetServerContextResultFields(nw); err != nil || !d.Success {
		t.Fatalf("nw reset = %+v, %v", d, err)
	}
	if _, err := DecodeFunctionResultFields(successfulReset); err == nil || !strings.Contains(err.Error(), "unknown tag 0x0523") {
		t.Fatalf("reset via plain = %v", err)
	}
	for _, fields := range [][]Field{
		{fld(TagUnresolved0420, make([]byte, 4)), fld(TagRfcServerResetDn, nil), fld(TagRfcServerResetDn, nil), fld(TagEnd, nil)},
		{fld(TagUnresolved0420, make([]byte, 4)), fld(TagRfcServerResetDn, []byte{1}), fld(TagEnd, nil)},
	} {
		if _, err := DecodeResetServerContextResultFields(resp(fields)); err == nil || !strings.Contains(err.Error(), "reset-done control must be empty and unique") {
			t.Fatalf("malformed reset = %v", err)
		}
	}
	remote := resp([]Field{fld(Tag(0x0403), utf16leBytes("RESET_FAILED")), fld(TagEnd, nil)})
	if d, err := DecodeResetServerContextResultFields(remote); err != nil || d.Envelope.Outcome != "abapRuntime" {
		t.Fatalf("remote failure = %+v, %v", d, err)
	}
}

func TestDecodesBoundedSessionRefresh(t *testing.T) {
	resp := func(fields []Field) []byte {
		return concat(hx(t, "010100080101010504010003"), mustChain(t, TagStart, fields), []byte{0xff, 0xff})
	}
	successful := resp([]Field{
		fld(TagProtocolVersion, make([]byte, 4)), fld(TagCapabilities, make([]byte, 11)), fld(TagLogonStatus, []byte{0}),
		fld(TagSystemCodePage, make([]byte, 8)), fld(TagResponseStart, nil), fld(TagResponseContext, nil),
		fld(TagUnresolved0420, make([]byte, 4)), fld(TagEnd, nil),
	})
	decoded, err := DecodeSessionRefreshResultFields(successful)
	if err != nil || !decoded.Success {
		t.Fatalf("refresh = %+v, %v", decoded, err)
	}
	want := []TagLen{{0x0503, 0}, {0x0420, 4}, {0xffff, 0}}
	if !reflect.DeepEqual(tagLens(decoded.Fields), want) {
		t.Fatalf("fields = %+v", tagLens(decoded.Fields))
	}
	if _, err := DecodeFunctionResultFields(successful); err == nil || !strings.Contains(err.Error(), "prefix is invalid") {
		t.Fatalf("via plain = %v", err)
	}
	for _, fields := range [][]Field{
		{fld(TagProtocolVersion, make([]byte, 4)), fld(TagCapabilities, make([]byte, 11)), fld(TagResponseStart, []byte{1}), fld(TagUnresolved0420, make([]byte, 4)), fld(TagEnd, nil)},
		{fld(TagProtocolVersion, make([]byte, 4)), fld(TagCapabilities, make([]byte, 11)), fld(Tag(0x7777), nil), fld(TagResponseStart, nil), fld(TagUnresolved0420, make([]byte, 4)), fld(TagEnd, nil)},
		{fld(TagProtocolVersion, make([]byte, 4)), fld(TagCapabilities, make([]byte, 11)), fld(TagLogonStatus, []byte{1}), fld(TagResponseStart, nil), fld(TagUnresolved0420, make([]byte, 4)), fld(TagEnd, nil)},
	} {
		if _, err := DecodeSessionRefreshResultFields(resp(fields)); err == nil || !strings.Contains(err.Error(), "session-refresh") {
			t.Fatalf("malformed refresh = %v", err)
		}
	}
}

func TestFailsClosedOnUnknownRegularEnvelopes(t *testing.T) {
	resp := func(fields []Field) []byte {
		return concat(hx(t, "05000000"), mustChain(t, TagResponseStart, fields), []byte{0xff, 0xff})
	}
	for _, fields := range [][]Field{
		{fld(Tag(0x7777), nil), fld(TagUnresolved0420, make([]byte, 4)), fld(TagEnd, nil)},
		{fld(Tag(0x0423), nil), fld(TagEnd, nil)},
	} {
		if _, err := DecodeFunctionResultFields(resp(fields)); err == nil {
			t.Fatalf("expected error for %+v", fields)
		}
	}
}

func TestMalformedDiagnosticsExposeStructuralFacts(t *testing.T) {
	fields := []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagCapabilities, make([]byte, 11)),
		fld(TagLogonStatus, []byte{0}), fld(TagUnresolved0420, make([]byte, 4)),
		fld(Tag(0x0450), make([]byte, 5)), fld(TagEnd, nil),
	}
	response := concat(hx(t, "010100080101010504010003"), mustChain(t, TagStart, fields), []byte{0xff, 0xff})
	_, err := DecodeInitialLogonResponse(response)
	se, ok := err.(*StructureError)
	if !ok || se.Rule != "malformed-vendor-logon-control" {
		t.Fatalf("err = %v", err)
	}
	want := []StructuralField{
		{Tag: 0x0103, ByteLength: 4, Index: 0}, {Tag: 0x0106, ByteLength: 11, Index: 1},
		{Tag: 0x0161, ByteLength: 1, Index: 2}, {Tag: 0x0420, ByteLength: 4, Index: 3},
		{Tag: 0x0450, ByteLength: 5, Index: 4}, {Tag: 0xffff, ByteLength: 0, Index: 5},
	}
	if !reflect.DeepEqual(se.Fields, want) {
		t.Fatalf("fields = %+v", se.Fields)
	}
}

func TestProjectsParseStageEnum(t *testing.T) {
	regularPrefix := "010100080101010504010003"
	errorPrefix := "010100080101010101010000"
	protocol := fld(TagProtocolVersion, hx(t, "00000e0b"))
	status := fld(TagLogonStatus, []byte{0})
	end := fld(TagEnd, nil)
	respWith := func(prefixHex string, fields []Field, trailer []byte) []byte {
		return concat(hx(t, prefixHex), mustChain(t, TagStart, fields), trailer)
	}
	ffff := []byte{0xff, 0xff}
	preamble := []Field{
		protocol, fld(TagCapabilities, make([]byte, 11)), fld(TagSystemCodePage, make([]byte, 4)),
		fld(TagClientAddress, make([]byte, 15)), fld(TagPartnerSystem, make([]byte, 9)), fld(TagPartnerHost, make([]byte, 17)),
		fld(TagConnectionType, make([]byte, 1)), fld(TagKernelPatch, make([]byte, 4)), fld(TagKernelRelease, make([]byte, 4)),
		fld(TagDestination, make([]byte, 10)), fld(TagProgram, make([]byte, 8)), fld(TagResponseStart, nil),
	}
	valid := respWith(regularPrefix, []Field{protocol, status, end}, ffff)
	brokenChain := append([]byte(nil), valid...)
	be16(brokenChain, len(hx(t, regularPrefix)), 0x7777)

	cases := []struct {
		stage    string
		response []byte
	}{
		{"truncated", []byte{}},
		{"prefix", make([]byte, len(valid))},
		{"field-chain", brokenChain},
		{"trailer", respWith(regularPrefix, []Field{protocol, status, end}, make([]byte, 2))},
		{"protocol", respWith(regularPrefix, []Field{status, end}, ffff)},
		{"error-preamble", respWith(errorPrefix, []Field{protocol, fld(tagAbapErrorMessage, utf16leBytes("X")), end}, ffff)},
		{"error-envelope", respWith(errorPrefix, append(append(clone(preamble), fld(TagUnresolved0420, make([]byte, 4))), end), ffff)},
		{"structural", respWith(regularPrefix, []Field{protocol, status, fld(Tag(0x7777), nil), end}, ffff)},
	}
	for _, c := range cases {
		_, err := DecodeInitialLogonResponse(c.response)
		if got := ParseStageOf(err); got != c.stage {
			t.Fatalf("%s: stage = %q (err %v)", c.stage, got, err)
		}
	}
	if ParseStageOf(errPlain("initial CPIC prefix is invalid")) != "" {
		t.Fatal("plain error should have no stage")
	}
}

type plainErr string

func (e plainErr) Error() string { return string(e) }
func errPlain(s string) error    { return plainErr(s) }

func hexs(b []byte) string { return hexEncode(b) }

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
