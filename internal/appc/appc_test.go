// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/appc.test.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten for the testing package.
// Assertions that depend on JavaScript-only behaviour have no Go analogue: the
// "ümlaut" non-ASCII string case is kept (Go strings can hold the bytes), while
// the ObservedContinuation subarray-interception check in the limits test is
// reduced to asserting the error, since Go slices cannot be intercepted (the
// underlying "fail before copying" behaviour is still enforced in code — the
// limit check precedes DecodeDataFragment). See docs/provenance.md.

package appc

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func p[T any](v T) *T { return &v }

func sampleOptions() ExtendedInitializeOptions {
	return ExtendedInitializeOptions{
		OptionFlags:                1,
		RootID:                     "0123456789ABCDEF",
		ConnectionID:               "FEDCBA9876543210",
		ConnectionIDSuffix:         1,
		Timeout:                    -2,
		KeepaliveTimeout:           -2,
		ExportTrace:                2,
		StartType:                  0,
		NetworkProtocol:            0,
		LocalAddressV6:             make([]byte, 16),
		LongLogicalUnitName:        "127.0.0.1",
		OperatingSystemUser:        "open-rfc",
		LocalAddressV4:             make([]byte, 4),
		LongTransactionProgramName: "sapdp00",
	}
}

// dataRecord builds an incoming F_SAP_SEND/F_RECEIVE record, mirroring the
// upstream test helper.
func dataRecord(fn Function, vector byte, data []byte, opts ...any) []byte {
	conversationID := "CONV0001"
	sequenceNumber := uint32(7)
	if len(opts) > 0 {
		conversationID = opts[0].(string)
	}
	if len(opts) > 1 {
		sequenceNumber = uint32(opts[1].(int))
	}
	record := make([]byte, RecordHeaderLength+len(data))
	record[0] = ProtocolVersion
	record[1] = byte(fn)
	be32(record, 22, sequenceNumber)
	record[31] = vector
	copy(record[40:], conversationID)
	be16(record, 50, 34_048)
	be16(record, 58, uint16(len(data)))
	be16(record, 76, 0)
	be16(record, 78, 6)
	copy(record[RecordHeaderLength:], data)
	return record
}

func be16(b []byte, off int, v uint16) { b[off] = byte(v >> 8); b[off+1] = byte(v) }
func be32(b []byte, off int, v uint32) {
	b[off] = byte(v >> 24)
	b[off+1] = byte(v >> 16)
	b[off+2] = byte(v >> 8)
	b[off+3] = byte(v)
}

func TestRecognizesControlledOracleFunctions(t *testing.T) {
	info, err := InspectPayload([]byte{ProtocolVersion, byte(FuncSapSend)})
	if err != nil {
		t.Fatal(err)
	}
	if info != (PayloadInfo{ProtocolVersion: 0x06, FunctionCode: 0xcb, FunctionName: "F_SAP_SEND"}) {
		t.Fatalf("info = %+v", info)
	}
	for fn, want := range map[Function]string{FuncReceive: "F_RECEIVE", FuncSetPartnerLuName: "F_SET_PARTNER_LU_NAME"} {
		got, err := InspectPayload([]byte{ProtocolVersion, byte(fn)})
		if err != nil || got.FunctionName != want {
			t.Fatalf("InspectPayload(%#x) = %q, %v", fn, got.FunctionName, err)
		}
	}
}

func TestEncodeDecodePartnerLogicalUnitInfo(t *testing.T) {
	host := mustHex(t, "00000000000000000000ffff7f000001")
	encoded, err := EncodePartnerLogicalUnitInfo(PartnerLogicalUnitInfoInput{
		LogicalUnitName: "NWRFC", PartnerHostAddress: host, CommunicationIndex: 0xffff, ConnectionIndex: 6,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != 32 {
		t.Fatalf("len = %d", len(encoded))
	}
	if string(encoded[0:8]) != "NWRFC   " {
		t.Fatalf("prefix = %q", encoded[0:8])
	}
	if be32of(encoded, 8) != 5 {
		t.Fatalf("nameLength = %d", be32of(encoded, 8))
	}
	decoded, err := DecodePartnerLogicalUnitInfo(encoded)
	if err != nil {
		t.Fatal(err)
	}
	want := PartnerLogicalUnitInfo{LogicalUnitNamePrefix: "NWRFC", LogicalUnitNameLength: 5, PartnerHostAddress: host, CommunicationIndex: 0xffff, ConnectionIndex: 6}
	if !reflect.DeepEqual(decoded, want) {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestEncodeDecodeExtendedInitializeOptions(t *testing.T) {
	opts := sampleOptions()
	encoded, err := EncodeExtendedInitializeOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != 341 {
		t.Fatalf("len = %d", len(encoded))
	}
	if encoded[0] != 1 {
		t.Fatalf("version = %d", encoded[0])
	}
	if got := hexOf(encoded[2:10]); got != "4350494300000000" {
		t.Fatalf("protocolName = %s", got)
	}
	if int32(be32of(encoded, 46)) != -2 || int32(be32of(encoded, 50)) != -2 {
		t.Fatalf("timeouts = %d, %d", int32(be32of(encoded, 46)), int32(be32of(encoded, 50)))
	}
	decoded, err := DecodeExtendedInitializeOptions(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, opts) {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestEncode373ByteInitializeParameters(t *testing.T) {
	encoded, err := EncodeInitializeParameters(InitializeParameters{ClientIdentifier: "NWRFC", Options: sampleOptions()})
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != 373 {
		t.Fatalf("len = %d", len(encoded))
	}
	if string(encoded[0:32]) != "NWRFC"+strings.Repeat(" ", 27) {
		t.Fatalf("clientId = %q", encoded[0:32])
	}
	decoded, err := DecodeInitializeParameters(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, InitializeParameters{ClientIdentifier: "NWRFC", Options: sampleOptions()}) {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestEncode144BytePartnerParameters(t *testing.T) {
	params := PartnerLogicalUnitParameters{LongLogicalUnitName: "127.0.0.1", PartnerHostAddress: make([]byte, 16)}
	encoded, err := EncodePartnerLogicalUnitParameters(params)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != 144 {
		t.Fatalf("len = %d", len(encoded))
	}
	if string(encoded[0:9]) != "127.0.0.1" {
		t.Fatalf("name = %q", encoded[0:9])
	}
	for _, b := range encoded[9:128] {
		if b != 0x20 {
			t.Fatalf("padding not space: %x", encoded[9:128])
		}
	}
	decoded, err := DecodePartnerLogicalUnitParameters(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, params) {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestRejectsInvalidInitializationIDsPaddingAndAddressWidths(t *testing.T) {
	bad := sampleOptions()
	bad.RootID = "lowercase1234567"
	if _, err := EncodeExtendedInitializeOptions(bad); !errors.Is(err, ErrRange) || !strings.Contains(err.Error(), "16 uppercase hexadecimal") {
		t.Fatalf("rootId = %v", err)
	}
	bad = sampleOptions()
	bad.LocalAddressV6 = make([]byte, 15)
	if _, err := EncodeExtendedInitializeOptions(bad); !errors.Is(err, ErrRange) || !strings.Contains(err.Error(), "exactly 16 bytes") {
		t.Fatalf("localAddressV6 = %v", err)
	}
	malformed, err := EncodeExtendedInitializeOptions(sampleOptions())
	if err != nil {
		t.Fatal(err)
	}
	malformed[93] = 0
	malformed[94] = 0x41
	if _, err := DecodeExtendedInitializeOptions(malformed); !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "longLogicalUnitName contains data after its first padding byte") {
		t.Fatalf("padding = %v", err)
	}
}

func TestPlacesPartnerOperationInfoInControlRecord(t *testing.T) {
	record, err := EncodeControlRecord(ControlRecordInput{
		FunctionCode: FuncSetPartnerLuName,
		PartnerLogicalUnitInfo: &PartnerLogicalUnitInfoInput{
			LogicalUnitName: "NWRFC", PartnerHostAddress: make([]byte, 16), CommunicationIndex: 0xffff, ConnectionIndex: 6,
		},
		Parameters: make([]byte, 144),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(record) != 224 {
		t.Fatalf("len = %d", len(record))
	}
	if string(record[48:56]) != "NWRFC   " {
		t.Fatalf("op info = %q", record[48:56])
	}
	if be32of(record, 56) != 5 {
		t.Fatalf("nameLength = %d", be32of(record, 56))
	}
	h, err := DecodeHeader(record)
	if err != nil || h.SapParameterLen != 144 {
		t.Fatalf("sapParameterLength = %d, %v", h.SapParameterLen, err)
	}
}

func TestRejectsMalformedOrMisplacedPartnerInfo(t *testing.T) {
	if _, err := EncodePartnerLogicalUnitInfo(PartnerLogicalUnitInfoInput{LogicalUnitName: "NWRFC", PartnerHostAddress: make([]byte, 15)}); !errors.Is(err, ErrRange) || !strings.Contains(err.Error(), "exactly 16 bytes") {
		t.Fatalf("host width = %v", err)
	}
	badLength := make([]byte, 32)
	copy(badLength, "NWRFC   ")
	be32(badLength, 8, 4)
	if _, err := DecodePartnerLogicalUnitInfo(badLength); !errors.Is(err, ErrProtocol) || !strings.Contains(err.Error(), "prefix length 5 does not match declared length 4") {
		t.Fatalf("prefix mismatch = %v", err)
	}
	if _, err := EncodeControlRecord(ControlRecordInput{FunctionCode: FuncSetPartnerLuName}); !strings.Contains(err.Error(), "requires partnerLogicalUnitInfo") {
		t.Fatalf("missing partner = %v", err)
	}
	if _, err := EncodeControlRecord(ControlRecordInput{FunctionCode: FuncAllocate, PartnerLogicalUnitInfo: &PartnerLogicalUnitInfoInput{LogicalUnitName: "NWRFC", PartnerHostAddress: make([]byte, 16)}}); !strings.Contains(err.Error(), "only valid for F_SET_PARTNER_LU_NAME") {
		t.Fatalf("misplaced partner = %v", err)
	}
}

func TestRetainsUnknownFunctionCodes(t *testing.T) {
	info, err := InspectPayload([]byte{0x06, 0xaa})
	if err != nil || info.FunctionName != "UNKNOWN_0xaa" {
		t.Fatalf("info = %+v, %v", info, err)
	}
}

func TestRejectsUnsupportedProtocolVersions(t *testing.T) {
	if _, err := InspectPayload([]byte{0x05, 0xcb}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v", err)
	}
}

func TestDecodesFixedVersion6CommonHeader(t *testing.T) {
	h := make([]byte, 48)
	h[0] = 0x06
	h[1] = byte(FuncAllocate)
	h[2] = 0x02
	h[3] = 0x03
	be16(h, 4, 0x1234)
	be16(h, 6, 0x5678)
	be16(h, 8, 9)
	h[10] = 0xaa
	h[11] = 4
	be32(h, 12, 0x1020_3040)
	h[16] = 0xbb
	be32(h, 17, 0xffff_ffff)
	h[21] = 0xcc
	be32(h, 22, 17)
	be16(h, 26, 23)
	be16(h, 28, 0x789a)
	h[30] = 0x7b
	h[31] = 0x04
	be32(h, 32, 5)
	be32(h, 36, 6)
	copy(h[40:], "CONV1234")

	d, err := DecodeHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if d.FunctionName != "F_ALLOCATE" || d.UID != 0x1234 || d.GatewayID != 0x5678 || d.Timeout != -1 ||
		d.SequenceNumber != 17 || d.Padding != 0x789a || d.Info != 0x7b || d.SapReturnCode != 6 || string(d.ConversationID) != "CONV1234" {
		t.Fatalf("decoded = %+v", d)
	}
}

func TestRejectsTruncatedCommonHeader(t *testing.T) {
	if _, err := DecodeHeader(make([]byte, 47)); !strings.Contains(err.Error(), "needs 48 bytes") {
		t.Fatalf("err = %v", err)
	}
}

func TestEncodeDecodeExtendedConnectionInfo(t *testing.T) {
	info := ExtendedInfo{ShortDestinationName: "NWRFC", LogicalUnitName: "127.0.0.", TransactionProgramName: "sapdp00", ConnectionType: 0x49, ClientInfo: 1, CommunicationIndex: 0xffff, ConnectionIndex: 6}
	encoded, err := EncodeExtendedInfo(info)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != 32 || string(encoded[0:8]) != "NWRFC   " {
		t.Fatalf("encoded = %x", encoded)
	}
	decoded, err := DecodeExtendedInfo(encoded)
	if err != nil || decoded != info {
		t.Fatalf("decoded = %+v, %v", decoded, err)
	}
}

func TestRejectsNonASCIIOrOversizedNames(t *testing.T) {
	base := ExtendedInfo{ShortDestinationName: "NWRFC", LogicalUnitName: "LOCAL", TransactionProgramName: "sapdp00", ConnectionType: 0x49, ClientInfo: 1}
	bad := base
	bad.LogicalUnitName = "123456789"
	if _, err := EncodeExtendedInfo(bad); !strings.Contains(err.Error(), "at most 8 ASCII bytes") {
		t.Fatalf("oversized = %v", err)
	}
	bad = base
	bad.LogicalUnitName = "ümlaut"
	if _, err := EncodeExtendedInfo(bad); !strings.Contains(err.Error(), "ASCII") {
		t.Fatalf("non-ascii = %v", err)
	}
}

func TestDerivesControlRecordLengths(t *testing.T) {
	parameters := []byte{1, 2, 3, 4}
	encoded, err := EncodeControlRecord(ControlRecordInput{
		RecordHeaderInput: RecordHeaderInput{Info2: p[uint8](1), Info3: p[uint8](0xc0), Info4: p[uint8](4), Info: p[uint8](5)},
		FunctionCode:      FuncInitialize,
		ExtendedInfo:      &ExtendedInfo{ShortDestinationName: "NWRFC", LogicalUnitName: "LOCAL", TransactionProgramName: "sapdp00", ConnectionType: 0x49, ClientInfo: 1, CommunicationIndex: 0xffff, ConnectionIndex: 6},
		Parameters:        parameters,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != RecordHeaderLength+len(parameters) {
		t.Fatalf("len = %d", len(encoded))
	}
	h, err := DecodeHeader(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if h.FunctionCode != FuncInitialize || h.Protocol != 2 || h.UID != 0xffff || int(h.SapParameterLen) != len(parameters) {
		t.Fatalf("header = %+v", h)
	}
	if !reflect.DeepEqual(encoded[RecordHeaderLength:], parameters) {
		t.Fatalf("params = %x", encoded[RecordHeaderLength:])
	}
}

func TestEncodesProvenClientSapSendDefaults(t *testing.T) {
	record, err := EncodeDataRecord(DataRecordInput{
		RecordHeaderInput:  RecordHeaderInput{ConversationID: []byte("CONV0001")},
		CommunicationIndex: 0xffff, ConnectionIndex: 6, Data: []byte("payload"),
	})
	if err != nil {
		t.Fatal(err)
	}
	h, err := DecodeHeader(record)
	if err != nil {
		t.Fatal(err)
	}
	if len(record) != 87 || h.FunctionCode != FuncSapSend || h.SapParameterLen != 8 || h.Info != 5 || h.Vector != 0x0c || string(h.ConversationID) != "CONV0001" {
		t.Fatalf("header = %+v len=%d", h, len(record))
	}
	if be16of(record, 76) != 0xffff || be16of(record, 78) != 6 || string(record[80:]) != "payload" {
		t.Fatalf("op info/data mismatch: %x", record)
	}
}

func TestMarksNonFinalSapSendFragments(t *testing.T) {
	record, err := EncodeDataRecord(DataRecordInput{CommunicationIndex: 1, ConnectionIndex: 2, IsFinal: p(false), Data: []byte{0, 1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	h, err := DecodeHeader(record)
	if err != nil || h.Vector != 0x08 {
		t.Fatalf("vector = %d, %v", h.Vector, err)
	}
	if !reflect.DeepEqual(record[80:], []byte{0, 1, 2}) {
		t.Fatalf("data = %x", record[80:])
	}
}

func TestUsesNULFilledExtensionWhenNamesAbsent(t *testing.T) {
	encoded, err := EncodeControlRecord(ControlRecordInput{FunctionCode: FuncAllocate})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(encoded[48:72], make([]byte, 24)) {
		t.Fatalf("extension = %x", encoded[48:72])
	}
}

func TestControlEncoderRejectsDataFunctionsOversizedParamsAndBadConvIDs(t *testing.T) {
	if _, err := EncodeControlRecord(ControlRecordInput{FunctionCode: FuncSapSend}); !strings.Contains(err.Error(), "not a setup/control function") {
		t.Fatalf("data function = %v", err)
	}
	if _, err := EncodeControlRecord(ControlRecordInput{FunctionCode: FuncInitialize, Parameters: make([]byte, 0x1_0000)}); !strings.Contains(err.Error(), "65535") {
		t.Fatalf("oversized params = %v", err)
	}
	if _, err := EncodeControlRecord(ControlRecordInput{RecordHeaderInput: RecordHeaderInput{ConversationID: make([]byte, 7)}, FunctionCode: FuncAllocate}); !strings.Contains(err.Error(), "exactly 8 bytes") {
		t.Fatalf("bad conv id = %v", err)
	}
}

func mustControl(t *testing.T, fn Function) []byte {
	t.Helper()
	b, err := EncodeControlRecord(ControlRecordInput{FunctionCode: fn})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestEnforcesClientSetupAndTeardownSequence(t *testing.T) {
	m := NewClientSetupStateMachine()
	if m.State() != StateNew {
		t.Fatal("initial state")
	}
	mustSend(t, m, FuncInitialize)
	if m.State() != StateInitializePending {
		t.Fatal(m.State())
	}
	mustReceive(t, m, mustControl(t, FuncInitialize))
	if m.State() != StateInitialized {
		t.Fatal(m.State())
	}
	mustSend(t, m, FuncSetPartnerLuName)
	if m.State() != StatePartnerSet {
		t.Fatal(m.State())
	}
	mustSend(t, m, FuncAllocate)
	if m.State() != StateAllocatePending {
		t.Fatal(m.State())
	}
	mustReceive(t, m, mustControl(t, FuncAllocate))
	if m.State() != StateReady {
		t.Fatal(m.State())
	}
	mustSend(t, m, FuncSapSend)
	mustReceive(t, m, dataRecord(FuncSapSend, 0x0c, nil))
	if m.State() != StateResponsePending {
		t.Fatal(m.State())
	}
	if err := m.ResponseComplete(); err != nil {
		t.Fatal(err)
	}
	mustSend(t, m, FuncDeallocate)
	if m.State() != StateClosed {
		t.Fatal(m.State())
	}
}

func mustSend(t *testing.T, m *ClientSetupStateMachine, fn Function) {
	t.Helper()
	if err := m.Sent(fn, true); err != nil {
		t.Fatalf("Sent(%s): %v", FunctionName(fn), err)
	}
}

func mustReceive(t *testing.T, m *ClientSetupStateMachine, payload []byte) {
	t.Helper()
	if _, err := m.Received(payload); err != nil {
		t.Fatalf("Received: %v", err)
	}
}

func TestRejectsIllegalSetupTransitionsAndFailedReplies(t *testing.T) {
	outOfOrder := NewClientSetupStateMachine()
	if err := outOfOrder.Sent(FuncAllocate, true); !strings.Contains(err.Error(), "cannot send F_ALLOCATE while APPC client is new") {
		t.Fatalf("out of order = %v", err)
	}

	failed := NewClientSetupStateMachine()
	mustSend(t, failed, FuncInitialize)
	reply := mustControl(t, FuncInitialize)
	be32(reply, 32, 6)
	if _, err := failed.Received(reply); err == nil || !strings.Contains(err.Error(), "APPC return code 6") {
		t.Fatalf("failed reply = %v", err)
	}
	if failed.State() != StateClosed {
		t.Fatal("state not closed")
	}

	truncated := NewClientSetupStateMachine()
	mustSend(t, truncated, FuncInitialize)
	mustReceive(t, truncated, mustControl(t, FuncInitialize))
	mustSend(t, truncated, FuncSetPartnerLuName)
	mustSend(t, truncated, FuncAllocate)
	if _, err := truncated.Received(mustControl(t, FuncAllocate)[:48]); err == nil || !strings.Contains(err.Error(), "APPC reply needs 80 bytes") {
		t.Fatalf("truncated = %v", err)
	}
	if truncated.State() != StateClosed {
		t.Fatal("state not closed")
	}
}

func TestAdmitsOnlyNormalDeallocationForTerminalDecoding(t *testing.T) {
	terminal := NewClientSetupStateMachine()
	mustSend(t, terminal, FuncInitialize)
	mustReceive(t, terminal, mustControl(t, FuncInitialize))
	mustSend(t, terminal, FuncSetPartnerLuName)
	mustSend(t, terminal, FuncAllocate)
	mustReceive(t, terminal, mustControl(t, FuncAllocate))
	mustSend(t, terminal, FuncSapSend)
	normalDeallocation := dataRecord(FuncReceive, 0x0c, []byte("terminal RFC envelope"))
	be32(normalDeallocation, 32, 18)
	disp, err := terminal.Received(normalDeallocation)
	if err != nil || disp != DispositionNormalDeallocation {
		t.Fatalf("disposition = %q, %v", disp, err)
	}
	if terminal.State() != StateClosed {
		t.Fatal("state not closed")
	}

	for _, rc := range []struct{ appc, sap uint32 }{{17, 0}, {18, 1}} {
		failed := NewClientSetupStateMachine()
		mustSend(t, failed, FuncInitialize)
		reply, err := EncodeControlRecord(ControlRecordInput{
			RecordHeaderInput: RecordHeaderInput{AppcReturnCode: p(rc.appc), SapReturnCode: p(rc.sap)},
			FunctionCode:      FuncInitialize,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := failed.Received(reply); err == nil || !strings.Contains(err.Error(), "failed with APPC return code") {
			t.Fatalf("rc %+v = %v", rc, err)
		}
		if failed.State() != StateClosed {
			t.Fatal("state not closed")
		}
	}
}

func TestDecodesApplicationDataAfterRecordHeader(t *testing.T) {
	fragment, err := DecodeDataFragment(dataRecord(FuncSapSend, 0x0c, []byte("complete")))
	if err != nil {
		t.Fatal(err)
	}
	if fragment.Header.FunctionName != "F_SAP_SEND" || !fragment.IsFinal || string(fragment.Data) != "complete" {
		t.Fatalf("fragment = %+v", fragment)
	}
}

func TestAssemblesOneCompleteSapSend(t *testing.T) {
	d, err := NewConversationDecoder(ConversationDecoderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	messages, err := d.Push(dataRecord(FuncSapSend, 0x0c, []byte("one")))
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || string(messages[0].Data) != "one" || messages[0].FragmentCount != 1 || string(messages[0].ConversationID) != "CONV0001" {
		t.Fatalf("messages = %+v", messages)
	}
	if d.BufferedByteLength() != 0 {
		t.Fatal("buffered")
	}
	if err := d.Finish(); err != nil {
		t.Fatal(err)
	}
}

func newDecoder(t *testing.T, o ConversationDecoderOptions) *ConversationDecoder {
	t.Helper()
	d, err := NewConversationDecoder(o)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestNormalDeallocationTerminalDelimiter(t *testing.T) {
	terminal := dataRecord(FuncSapSend, 0x08, []byte("complete terminal envelope"))
	be32(terminal, 32, 18)
	d := newDecoder(t, ConversationDecoderOptions{})
	messages, err := d.PushTerminalDeallocation(terminal)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || string(messages[0].Data) != "complete terminal envelope" || messages[0].FragmentCount != 1 {
		t.Fatalf("messages = %+v", messages)
	}
	if err := d.Finish(); err != nil {
		t.Fatal(err)
	}

	if _, err := newDecoder(t, ConversationDecoderOptions{}).Push(terminal); err == nil || !strings.Contains(err.Error(), "normal deallocation must use the terminal decoder") {
		t.Fatalf("push terminal = %v", err)
	}

	empty := dataRecord(FuncSapSend, 0x08, nil)
	be32(empty, 32, 18)
	if _, err := newDecoder(t, ConversationDecoderOptions{}).PushTerminalDeallocation(empty); !errors.Is(err, ErrNormalDeallocationWithoutData) {
		t.Fatalf("empty terminal = %v", err)
	}

	ordinary := dataRecord(FuncSapSend, 0x08, []byte("not terminal"))
	if _, err := newDecoder(t, ConversationDecoderOptions{}).PushTerminalDeallocation(ordinary); err == nil || !strings.Contains(err.Error(), "requires APPC return code 18") {
		t.Fatalf("ordinary = %v", err)
	}

	orphanedReceive := dataRecord(FuncReceive, 0x08, []byte("orphaned terminal receive"))
	be32(orphanedReceive, 32, 18)
	if _, err := newDecoder(t, ConversationDecoderOptions{}).PushTerminalDeallocation(orphanedReceive); err == nil || !strings.Contains(err.Error(), "terminal F_RECEIVE without a preceding F_SAP_SEND") {
		t.Fatalf("orphaned = %v", err)
	}
	msgs, err := newDecoder(t, ConversationDecoderOptions{AllowInitialReceive: true}).PushTerminalDeallocation(orphanedReceive)
	if err != nil || string(msgs[0].Data) != "orphaned terminal receive" {
		t.Fatalf("allowInitialReceive = %+v, %v", msgs, err)
	}

	first := dataRecord(FuncSapSend, 0x08, []byte("fragment "))
	last := dataRecord(FuncReceive, 0x08, []byte("at deallocation"))
	be32(last, 32, 18)
	fragmented := newDecoder(t, ConversationDecoderOptions{})
	if msgs, err := fragmented.Push(first); err != nil || len(msgs) != 0 {
		t.Fatalf("first = %+v, %v", msgs, err)
	}
	assembled, err := fragmented.PushTerminalDeallocation(last)
	if err != nil || string(assembled[0].Data) != "fragment at deallocation" || assembled[0].FragmentCount != 2 {
		t.Fatalf("assembled = %+v, %v", assembled, err)
	}
	if err := fragmented.Finish(); err != nil {
		t.Fatal(err)
	}
}

func TestAssemblesSapSendPlusMultipleReceiveContinuations(t *testing.T) {
	d := newDecoder(t, ConversationDecoderOptions{})
	if msgs, err := d.Push(dataRecord(FuncSapSend, 0x08, []byte("first-"))); err != nil || len(msgs) != 0 {
		t.Fatalf("first = %+v, %v", msgs, err)
	}
	if msgs, err := d.Push(dataRecord(FuncReceive, 0x00, []byte("middle-"))); err != nil || len(msgs) != 0 {
		t.Fatalf("middle = %+v, %v", msgs, err)
	}
	messages, err := d.Push(dataRecord(FuncReceive, 0x0c, []byte("last")))
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || string(messages[0].Data) != "first-middle-last" || messages[0].FragmentCount != 3 {
		t.Fatalf("messages = %+v", messages)
	}
	if err := d.Finish(); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsContinuationWithoutPrecedingSend(t *testing.T) {
	d := newDecoder(t, ConversationDecoderOptions{})
	if _, err := d.Push(dataRecord(FuncReceive, 0x0c, []byte("orphan"))); err == nil || !strings.Contains(err.Error(), "F_RECEIVE") || !strings.Contains(err.Error(), "F_SAP_SEND") {
		t.Fatalf("orphan = %v", err)
	}
}

func TestRejectsConversationAndSequenceChanges(t *testing.T) {
	cd := newDecoder(t, ConversationDecoderOptions{})
	cd.Push(dataRecord(FuncSapSend, 0x08, []byte("a")))
	if _, err := cd.Push(dataRecord(FuncReceive, 0x0c, []byte("b"), "CONV0002")); err == nil || !strings.Contains(err.Error(), "conversation ID changed") {
		t.Fatalf("conv change = %v", err)
	}

	sd := newDecoder(t, ConversationDecoderOptions{})
	sd.Push(dataRecord(FuncSapSend, 0x08, []byte("a")))
	if _, err := sd.Push(dataRecord(FuncReceive, 0x0c, []byte("b"), "CONV0001", 8)); err == nil || !strings.Contains(err.Error(), "sequence number changed") {
		t.Fatalf("seq change = %v", err)
	}
}

func TestEnforcesMessageByteAndFragmentLimits(t *testing.T) {
	byteLimited := newDecoder(t, ConversationDecoderOptions{MaxMessageLength: p(3)})
	byteLimited.Push(dataRecord(FuncSapSend, 0x08, []byte("abc")))
	if _, err := byteLimited.Push(dataRecord(FuncReceive, 0x0c, []byte("d"))); err == nil || !strings.Contains(err.Error(), "message length 4 exceeds configured limit 3") {
		t.Fatalf("byte limit = %v", err)
	}

	fragmentLimited := newDecoder(t, ConversationDecoderOptions{MaxFragments: p(2)})
	fragmentLimited.Push(dataRecord(FuncSapSend, 0x08, []byte("a")))
	fragmentLimited.Push(dataRecord(FuncReceive, 0x00, []byte("b")))
	if _, err := fragmentLimited.Push(dataRecord(FuncReceive, 0x0c, []byte("c"))); err == nil || !strings.Contains(err.Error(), "fragment count 3 exceeds configured limit 2") {
		t.Fatalf("fragment limit = %v", err)
	}
}

func TestRejectsNewSendControlOrEndOfStreamDuringContinuation(t *testing.T) {
	sendDecoder := newDecoder(t, ConversationDecoderOptions{})
	sendDecoder.Push(dataRecord(FuncSapSend, 0x08, []byte("a")))
	if _, err := sendDecoder.Push(dataRecord(FuncSapSend, 0x0c, []byte("b"))); err == nil || !strings.Contains(err.Error(), "new F_SAP_SEND") {
		t.Fatalf("new send = %v", err)
	}

	controlDecoder := newDecoder(t, ConversationDecoderOptions{})
	controlDecoder.Push(dataRecord(FuncSapSend, 0x08, []byte("a")))
	if _, err := controlDecoder.Push(dataRecord(FuncDeallocate, 0, nil)); err == nil || !strings.Contains(err.Error(), "F_DEALLOCATE") || !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("control = %v", err)
	}

	finishDecoder := newDecoder(t, ConversationDecoderOptions{})
	finishDecoder.Push(dataRecord(FuncSapSend, 0x08, []byte("a")))
	if err := finishDecoder.Finish(); err == nil || !strings.Contains(err.Error(), "truncated APPC message") {
		t.Fatalf("finish = %v", err)
	}
}

// test helpers shared with fragmentation tests

func be16of(b []byte, off int) uint16 { return uint16(b[off])<<8 | uint16(b[off+1]) }
func be32of(b []byte, off int) uint32 {
	return uint32(b[off])<<24 | uint32(b[off+1])<<16 | uint32(b[off+2])<<8 | uint32(b[off+3])
}
