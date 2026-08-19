// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/appc-outgoing-fragmentation.test.ts at commit
// 847036d, Copyright 2026 Marian Zeis, licensed under the Apache License,
// Version 2.0. Modified by open-rfc-go contributors: rewritten for the testing
// package. Cases with no Go analogue are dropped and recorded in
// docs/provenance.md: "planner reads each caller accessor once" (Go structs
// have no getters), the geometry-override half of "planner uses intrinsic
// typed-array geometry" (Go len() cannot be spoofed — the plain oversized/
// streamed bounds behaviour is kept), the `Object.isFrozen` assertions (Go has
// no runtime freeze), and the `1 as boolean` / non-integer / out-of-uint-range
// argument cases (rejected by Go's type system, not at run time).

package appc

import (
	"bytes"
	"strings"
	"testing"
)

const (
	fragSeq     = uint32(0x0102_0304)
	fragCommIdx = uint16(0xabcd)
	fragConnIdx = uint16(0x1234)
)

func fragConvID() []byte { return []byte("CONV0001") }

func patternedBytes(length int) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}

func finalSapParams(applicationDataLength int) []byte {
	p := make([]byte, FinalSapParameterLength)
	be16(p, 2, uint16(applicationDataLength))
	be32(p, 4, 0x8500)
	return p
}

func compactPlan(t *testing.T, data []byte) []OutgoingDataFragment {
	t.Helper()
	seq := fragSeq
	frags, err := PlanOutgoingDataFragments(OutgoingDataPlanInput{
		RecordHeaderInput:  RecordHeaderInput{ConversationID: fragConvID(), SequenceNumber: &seq},
		ApplicationData:    data,
		FinalSapParameters: finalSapParams(len(data)),
		CommunicationIndex: fragCommIdx,
		ConnectionIndex:    fragConnIdx,
	}, OutgoingDataPlannerOptions{CpicStreaming: StreamingEnabled})
	if err != nil {
		t.Fatalf("compactPlan(%d): %v", len(data), err)
	}
	return frags
}

func streamedPlan(t *testing.T, data []byte) []OutgoingDataFragment {
	t.Helper()
	seq := fragSeq
	frags, err := PlanOutgoingDataFragments(OutgoingDataPlanInput{
		RecordHeaderInput:  RecordHeaderInput{ConversationID: fragConvID(), SequenceNumber: &seq},
		ApplicationData:    data,
		CommunicationIndex: fragCommIdx,
		ConnectionIndex:    fragConnIdx,
	}, OutgoingDataPlannerOptions{CpicStreaming: StreamingEnabled})
	if err != nil {
		t.Fatalf("streamedPlan(%d): %v", len(data), err)
	}
	return frags
}

func readySetup(t *testing.T) *ClientSetupStateMachine {
	t.Helper()
	m := NewClientSetupStateMachine()
	mustSend(t, m, FuncInitialize)
	mustReceive(t, m, mustControl(t, FuncInitialize))
	mustSend(t, m, FuncSetPartnerLuName)
	mustSend(t, m, FuncAllocate)
	mustReceive(t, m, mustControl(t, FuncAllocate))
	if m.State() != StateReady {
		t.Fatalf("state = %s", m.State())
	}
	return m
}

func TestKeepsCompactDataThrough28000(t *testing.T) {
	boundary := MaxApplicationDataFragmentLength
	for _, length := range []int{0, 1, boundary - 1, boundary} {
		applicationData := patternedBytes(length)
		frags := compactPlan(t, applicationData)
		if len(frags) != 1 {
			t.Fatalf("length %d: %d fragments", length, len(frags))
		}
		f := frags[0]
		if f.FunctionCode != FuncSapSend || f.FragmentIndex != 0 || f.FragmentCount != 1 || !f.IsFinal || f.Info != 5 || f.Vector != 0x0c || f.SapParameterLength != 8 {
			t.Fatalf("length %d: fragment = %+v", length, f)
		}
		record, err := EncodeOutgoingDataFragment(f)
		if err != nil {
			t.Fatal(err)
		}
		header, err := DecodeHeader(record)
		if err != nil {
			t.Fatal(err)
		}
		operationInfo, err := DecodeExtendedInfo(record[48:80])
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeDataFragment(record)
		if err != nil {
			t.Fatal(err)
		}
		if header.FunctionCode != FuncSapSend || header.SapParameterLen != 8 || header.SequenceNumber != fragSeq || !bytes.Equal(header.ConversationID, fragConvID()) {
			t.Fatalf("length %d: header = %+v", length, header)
		}
		if operationInfo.CommunicationIndex != fragCommIdx || operationInfo.ConnectionIndex != fragConnIdx {
			t.Fatalf("length %d: op info = %+v", length, operationInfo)
		}
		if !bytes.Equal(decoded.Data[:length], applicationData) || !bytes.Equal(decoded.Data[length:], finalSapParams(length)) {
			t.Fatalf("length %d: body mismatch", length)
		}
	}
}

func TestSwitchesAbove28000ToAsyncPlusReceive(t *testing.T) {
	boundary := MaxApplicationDataFragmentLength
	for _, length := range []int{boundary + 1, 2 * boundary, 2*boundary + 1} {
		applicationData := patternedBytes(length)
		frags := streamedPlan(t, applicationData)
		dataFragmentCount := (length + boundary - 1) / boundary
		if len(frags) != dataFragmentCount+1 {
			t.Fatalf("length %d: %d fragments", length, len(frags))
		}
		var reconstructed []byte
		for index := 0; index < dataFragmentCount; index++ {
			f := frags[index]
			expectedLength := boundary
			if rem := length - index*boundary; rem < expectedLength {
				expectedLength = rem
			}
			if f.FunctionCode != FuncAsyncSendData || f.FragmentIndex != index || f.FragmentCount != dataFragmentCount+1 || f.IsFinal || f.Info != 0 || f.Vector != 0 || f.SapParameterLength != 0 || len(f.FinalSapParameters) != 0 {
				t.Fatalf("length %d idx %d: fragment = %+v", length, index, f)
			}
			record, err := EncodeOutgoingDataFragment(f)
			if err != nil {
				t.Fatal(err)
			}
			header, err := DecodeHeader(record)
			if err != nil {
				t.Fatal(err)
			}
			if header.FunctionCode != FuncAsyncSendData || header.Info != 0 || header.Vector != 0 || header.SapParameterLen != 0 {
				t.Fatalf("header = %+v", header)
			}
			info, err := DecodeAsyncDataInfo(record[48:80])
			if err != nil {
				t.Fatal(err)
			}
			if info != (DataOperationInfo{DataLength: uint16(expectedLength), CommunicationIndex: fragCommIdx, ConnectionIndex: fragConnIdx}) {
				t.Fatalf("async info = %+v", info)
			}
			body := record[RecordHeaderLength:]
			if len(body) != expectedLength || !bytes.Equal(body, f.ApplicationData) {
				t.Fatalf("body mismatch")
			}
			reconstructed = append(reconstructed, body...)
		}
		terminator := frags[len(frags)-1]
		if terminator.FunctionCode != FuncReceive || !terminator.IsFinal || terminator.Info != 1 || terminator.Vector != 0 || terminator.SapParameterLength != 0 || len(terminator.ApplicationData) != 0 || len(terminator.FinalSapParameters) != 0 {
			t.Fatalf("terminator = %+v", terminator)
		}
		tRecord, err := EncodeOutgoingDataFragment(terminator)
		if err != nil {
			t.Fatal(err)
		}
		if len(tRecord) != RecordHeaderLength {
			t.Fatalf("terminator record len = %d", len(tRecord))
		}
		tInfo, err := DecodeAsyncDataInfo(tRecord[48:80])
		if err != nil {
			t.Fatal(err)
		}
		if tInfo != (DataOperationInfo{DataLength: MaxApplicationDataFragmentLength, CommunicationIndex: fragCommIdx, ConnectionIndex: fragConnIdx}) {
			t.Fatalf("terminator info = %+v", tInfo)
		}
		if !bytes.Equal(reconstructed, applicationData) {
			t.Fatalf("reconstructed mismatch")
		}
	}
}

func TestStreamsAboveUINT2WithoutCompactParams(t *testing.T) {
	frags := streamedPlan(t, patternedBytes(65_536))
	if len(frags) != 4 {
		t.Fatalf("%d fragments", len(frags))
	}
	wantCodes := []Function{FuncAsyncSendData, FuncAsyncSendData, FuncAsyncSendData, FuncReceive}
	wantLens := []int{28_000, 28_000, 9_536, 0}
	var async []byte
	for i, f := range frags {
		if f.FunctionCode != wantCodes[i] || len(f.ApplicationData) != wantLens[i] || len(f.FinalSapParameters) != 0 {
			t.Fatalf("fragment %d = %+v", i, f)
		}
		if f.FunctionCode == FuncAsyncSendData {
			async = append(async, f.ApplicationData...)
		}
	}
	if !bytes.Equal(async, patternedBytes(65_536)) {
		t.Fatalf("async reconstruction mismatch")
	}
}

func TestInsertsPeriodicSynchronousBarriers(t *testing.T) {
	frags := streamedPlan(t, patternedBytes(50*MaxApplicationDataFragmentLength))
	if MaxAsyncSendsBeforeSync != 21 {
		t.Fatal("barrier constant")
	}
	var want []Function
	for i := 0; i < 21; i++ {
		want = append(want, FuncAsyncSendData)
	}
	want = append(want, FuncSendData)
	for i := 0; i < 20; i++ {
		want = append(want, FuncAsyncSendData)
	}
	want = append(want, FuncSendData)
	for i := 0; i < 7; i++ {
		want = append(want, FuncAsyncSendData)
	}
	want = append(want, FuncReceive)
	if len(frags) != len(want) {
		t.Fatalf("%d fragments, want %d", len(frags), len(want))
	}
	for i, f := range frags {
		if f.FunctionCode != want[i] {
			t.Fatalf("fragment %d = %s, want %s", i, FunctionName(f.FunctionCode), FunctionName(want[i]))
		}
	}
	for _, f := range frags {
		if f.FunctionCode != FuncSendData {
			continue
		}
		if f.Info != 1 || f.Vector != 0 || f.SapParameterLength != 0 || f.IsFinal {
			t.Fatalf("barrier fragment = %+v", f)
		}
		record, err := EncodeOutgoingDataFragment(f)
		if err != nil {
			t.Fatal(err)
		}
		info, err := DecodeAsyncDataInfo(record[48:80])
		if err != nil {
			t.Fatal(err)
		}
		if info != (DataOperationInfo{DataLength: MaxApplicationDataFragmentLength, CommunicationIndex: fragCommIdx, ConnectionIndex: fragConnIdx}) {
			t.Fatalf("barrier info = %+v", info)
		}
		if len(record[RecordHeaderLength:]) != MaxApplicationDataFragmentLength {
			t.Fatalf("barrier body len = %d", len(record[RecordHeaderLength:]))
		}
	}
}

func TestPlannerSnapshotsCallerBuffers(t *testing.T) {
	source := patternedBytes(73)
	expected := append([]byte(nil), source...)
	callerConversationID := fragConvID()
	callerSap := finalSapParams(len(source))
	expectedSap := append([]byte(nil), callerSap...)
	seq := fragSeq
	frags, err := PlanOutgoingDataFragments(OutgoingDataPlanInput{
		RecordHeaderInput:  RecordHeaderInput{ConversationID: callerConversationID, SequenceNumber: &seq},
		ApplicationData:    source,
		FinalSapParameters: callerSap,
		CommunicationIndex: fragCommIdx,
		ConnectionIndex:    fragConnIdx,
	}, OutgoingDataPlannerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	f := frags[0]
	for i := range source {
		source[i] = 0xff
	}
	for i := range callerConversationID {
		callerConversationID[i] = 0x20
	}
	for i := range callerSap {
		callerSap[i] = 0xff
	}
	if !bytes.Equal(f.ApplicationData, expected) || !bytes.Equal(f.ConversationID, fragConvID()) || !bytes.Equal(f.FinalSapParameters, expectedSap) {
		t.Fatalf("planner did not snapshot caller buffers")
	}
}

func TestPlannerBoundsUseRealLength(t *testing.T) {
	// Go analogue of the intrinsic-geometry test: len() is authoritative, so an
	// over-limit input is rejected by its real length.
	oversized := make([]byte, 50_000)
	max := 28_001
	seq := fragSeq
	_, err := PlanOutgoingDataFragments(OutgoingDataPlanInput{
		RecordHeaderInput: RecordHeaderInput{ConversationID: fragConvID(), SequenceNumber: &seq},
		ApplicationData:   oversized, CommunicationIndex: fragCommIdx, ConnectionIndex: fragConnIdx,
	}, OutgoingDataPlannerOptions{CpicStreaming: StreamingEnabled, MaxApplicationDataLength: &max})
	if err == nil || !strings.Contains(err.Error(), "application data length 50000 exceeds configured limit 28001") {
		t.Fatalf("oversized = %v", err)
	}

	streamed := patternedBytes(28_001)
	frags, err := PlanOutgoingDataFragments(OutgoingDataPlanInput{
		RecordHeaderInput: RecordHeaderInput{ConversationID: fragConvID(), SequenceNumber: &seq},
		ApplicationData:   streamed, CommunicationIndex: fragCommIdx, ConnectionIndex: fragConnIdx,
	}, OutgoingDataPlannerOptions{CpicStreaming: StreamingEnabled, MaxApplicationDataLength: &max})
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, f := range frags {
		total += len(f.ApplicationData)
	}
	if total != 28_001 {
		t.Fatalf("total = %d", total)
	}
}

func TestPlannerFailsClosed(t *testing.T) {
	base := func() OutgoingDataPlanInput {
		seq := fragSeq
		return OutgoingDataPlanInput{
			RecordHeaderInput:  RecordHeaderInput{ConversationID: fragConvID(), SequenceNumber: &seq},
			ApplicationData:    make([]byte, 2),
			FinalSapParameters: finalSapParams(2),
			CommunicationIndex: fragCommIdx,
			ConnectionIndex:    fragConnIdx,
		}
	}
	mustFail := func(name string, in OutgoingDataPlanInput, o OutgoingDataPlannerOptions, want string) {
		t.Helper()
		if _, err := PlanOutgoingDataFragments(in, o); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s = %v", name, err)
		}
	}

	for _, m := range []int{-1, MaxOutgoingMessageLength + 1} {
		mm := m
		mustFail("maxApplicationDataLength", base(), OutgoingDataPlannerOptions{MaxApplicationDataLength: &mm}, "maxApplicationDataLength")
	}
	okMax := DefaultMaxMessageLength
	if _, err := PlanOutgoingDataFragments(base(), OutgoingDataPlannerOptions{MaxApplicationDataLength: &okMax}); err != nil {
		t.Fatalf("default max: %v", err)
	}
	one := 1
	mustFail("max=1", base(), OutgoingDataPlannerOptions{MaxApplicationDataLength: &one}, "application data length 2 exceeds configured limit 1")

	for _, m := range []int{-1, 0} {
		mm := m
		mustFail("maxFragments", base(), OutgoingDataPlannerOptions{MaxFragments: &mm}, "maxFragments")
	}
	twoFrags := 2
	streamInput := base()
	streamInput.ApplicationData = make([]byte, 28_001)
	streamInput.FinalSapParameters = nil
	mustFail("fragment count", streamInput, OutgoingDataPlannerOptions{MaxFragments: &twoFrags, CpicStreaming: StreamingEnabled}, "fragment count 3 exceeds configured limit 2")

	sap7 := base()
	sap7.FinalSapParameters = make([]byte, 7)
	mustFail("sap width", sap7, OutgoingDataPlannerOptions{}, "exactly 8 bytes")

	reservedInput := base()
	reserved := finalSapParams(2)
	be16(reserved, 0, 1)
	reservedInput.FinalSapParameters = reserved
	mustFail("reserved", reservedInput, OutgoingDataPlannerOptions{}, "reserved field must be zero")

	declareInput := base()
	declareInput.FinalSapParameters = finalSapParams(1)
	mustFail("declare mismatch", declareInput, OutgoingDataPlannerOptions{}, "declare 1 application bytes")

	streamedTooSmall := base()
	streamedTooSmall.ApplicationData = make([]byte, 28_000)
	streamedTooSmall.FinalSapParameters = nil
	mustFail("streamed too small", streamedTooSmall, OutgoingDataPlannerOptions{}, "must exceed 28000 bytes")

	if _, err := PlanOutgoingDataFragments(func() OutgoingDataPlanInput {
		in := base()
		in.ApplicationData = make([]byte, 28_001)
		in.FinalSapParameters = nil
		return in
	}(), OutgoingDataPlannerOptions{CpicStreaming: StreamingEnabled}); err != nil {
		t.Fatalf("streamed 28001: %v", err)
	}

	streamingDisabled := base()
	streamingDisabled.ApplicationData = make([]byte, 28_001)
	streamingDisabled.FinalSapParameters = nil
	mustFail("streaming disabled", streamingDisabled, OutgoingDataPlannerOptions{}, "CPIC streaming is disabled")

	mustFail("cpicStreaming", base(), OutgoingDataPlannerOptions{CpicStreaming: CpicStreamingPolicy("automatic")}, "cpicStreaming must be disabled or enabled")

	badConv := base()
	badConv.ConversationID = make([]byte, 7)
	mustFail("conversationId", badConv, OutgoingDataPlannerOptions{}, "conversationId")
}

func TestPlannedRecordEncoderRejectsForgedSemantics(t *testing.T) {
	only := compactPlan(t, []byte("one"))[0]
	forge := []func(f OutgoingDataFragment) OutgoingDataFragment{
		func(f OutgoingDataFragment) OutgoingDataFragment { f.FunctionCode = FuncReceive; return f },
		func(f OutgoingDataFragment) OutgoingDataFragment { f.Vector = 0x08; return f },
		func(f OutgoingDataFragment) OutgoingDataFragment { f.Info = 1; return f },
		func(f OutgoingDataFragment) OutgoingDataFragment { f.SapParameterLength = 0; return f },
		func(f OutgoingDataFragment) OutgoingDataFragment { f.FragmentIndex = 1; return f },
		func(f OutgoingDataFragment) OutgoingDataFragment { f.FragmentCount = 2; return f },
		func(f OutgoingDataFragment) OutgoingDataFragment {
			f.ApplicationData = make([]byte, MaxApplicationDataFragmentLength+1)
			return f
		},
		func(f OutgoingDataFragment) OutgoingDataFragment { f.FinalSapParameters = []byte{}; return f },
		func(f OutgoingDataFragment) OutgoingDataFragment { f.FinalSapParameters = nil; return f },
		func(f OutgoingDataFragment) OutgoingDataFragment { f.MessageApplicationDataLength = 2; return f },
	}
	for i, mutate := range forge {
		if _, err := EncodeOutgoingDataFragment(mutate(cloneFragment(only))); err == nil || !strings.Contains(err.Error(), "invalid outgoing APPC fragment") {
			t.Fatalf("forge %d = %v", i, err)
		}
	}

	async := streamedPlan(t, make([]byte, 28_001))
	bad := cloneFragment(async[0])
	bad.FinalSapParameters = make([]byte, 8)
	if _, err := EncodeOutgoingDataFragment(bad); err == nil || !strings.Contains(err.Error(), "F_ASEND_DATA cannot carry SAP parameters") {
		t.Fatalf("async sap = %v", err)
	}
	badTerm := cloneFragment(async[len(async)-1])
	badTerm.ApplicationData = []byte{1}
	if _, err := EncodeOutgoingDataFragment(badTerm); err == nil || !strings.Contains(err.Error(), "F_RECEIVE terminator must be empty") {
		t.Fatalf("terminator = %v", err)
	}
}

func cloneFragment(f OutgoingDataFragment) OutgoingDataFragment {
	c := f
	c.ConversationID = append([]byte(nil), f.ConversationID...)
	c.ApplicationData = append([]byte(nil), f.ApplicationData...)
	c.FinalSapParameters = append([]byte(nil), f.FinalSapParameters...)
	return c
}

func TestDecodesAndRejectsMalformedAsyncInfo(t *testing.T) {
	f := streamedPlan(t, make([]byte, 28_001))[0]
	record, err := EncodeOutgoingDataFragment(f)
	if err != nil {
		t.Fatal(err)
	}
	info, err := DecodeAsyncDataInfo(record[48:80])
	if err != nil || info != (DataOperationInfo{DataLength: 28_000, CommunicationIndex: fragCommIdx, ConnectionIndex: fragConnIdx}) {
		t.Fatalf("info = %+v, %v", info, err)
	}
	for _, malformed := range [][]byte{record[48:79], append(append([]byte(nil), record[48:80]...), 0)} {
		if _, err := DecodeAsyncDataInfo(malformed); err == nil || !strings.Contains(err.Error(), "exactly 32 bytes") {
			t.Fatalf("malformed length = %v", err)
		}
	}
	reserved := append([]byte(nil), record[48:80]...)
	reserved[4] = 1
	if _, err := DecodeAsyncDataInfo(reserved); err == nil || !strings.Contains(err.Error(), "reserved bytes must be zero") {
		t.Fatalf("reserved = %v", err)
	}
}

func TestDecodesOnlyCanonicalSynchronousSendAck(t *testing.T) {
	seq := uint32(0)
	ack, err := EncodeDataRecord(DataRecordInput{
		RecordHeaderInput:  RecordHeaderInput{ConversationID: fragConvID(), SequenceNumber: &seq, Info4: p[uint8](2)},
		FunctionCode:       p(FuncSendData),
		CommunicationIndex: 0, ConnectionIndex: fragConnIdx, IsFinal: p(false), Data: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSynchronousSendAcknowledgement(ack)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.Header.ConversationID, fragConvID()) || decoded.ConnectionIndex != fragConnIdx {
		t.Fatalf("decoded = %+v", decoded)
	}

	trailing := append(append([]byte(nil), ack...), 0)
	if _, err := DecodeSynchronousSendAcknowledgement(trailing); err == nil || !strings.Contains(err.Error(), "exactly 80 bytes") {
		t.Fatalf("trailing = %v", err)
	}
	wrongHeader := append([]byte(nil), ack...)
	wrongHeader[30] = 0
	if _, err := DecodeSynchronousSendAcknowledgement(wrongHeader); err == nil || !strings.Contains(err.Error(), "header is not canonical") {
		t.Fatalf("wrong header = %v", err)
	}
	for _, offset := range []int{2, 3, 4, 6, 8, 10, 11, 12, 16, 17, 22, 26, 28, 31, 32, 36} {
		malformed := append([]byte(nil), ack...)
		malformed[offset] ^= 1
		if _, err := DecodeSynchronousSendAcknowledgement(malformed); err == nil || !strings.Contains(err.Error(), "header is not canonical") {
			t.Fatalf("header offset %d = %v", offset, err)
		}
	}
	for offset := 48; offset < 78; offset++ {
		malformed := append([]byte(nil), ack...)
		malformed[offset] = 1
		if _, err := DecodeSynchronousSendAcknowledgement(malformed); err == nil || !strings.Contains(err.Error(), "operation information is not canonical") {
			t.Fatalf("op-info offset %d = %v", offset-48, err)
		}
	}
}

func TestClientSetupAllowsOnlyCapturedAsyncSendSequence(t *testing.T) {
	m := readySetup(t)
	if err := m.Sent(FuncSapSend, false); err == nil || !strings.Contains(err.Error(), "F_SAP_SEND cannot start a streamed outgoing message") {
		t.Fatalf("sapsend nonfinal = %v", err)
	}
	if err := m.Sent(FuncAsyncSendData, false); err != nil || m.State() != StateSendContinuation {
		t.Fatalf("async1 = %v state %s", err, m.State())
	}
	if err := m.Sent(FuncAsyncSendData, false); err != nil || m.State() != StateSendContinuation {
		t.Fatalf("async2 = %v state %s", err, m.State())
	}
	if err := m.Sent(FuncDeallocate, true); err == nil || !strings.Contains(err.Error(), "cannot send F_DEALLOCATE") || !strings.Contains(err.Error(), "send-continuation") {
		t.Fatalf("dealloc = %v", err)
	}
	if err := m.Sent(FuncReceive, false); err == nil || !strings.Contains(err.Error(), "F_RECEIVE terminator must be final") {
		t.Fatalf("nonfinal receive = %v", err)
	}
	if err := m.Sent(FuncReceive, true); err != nil || m.State() != StateResponsePending {
		t.Fatalf("final receive = %v state %s", err, m.State())
	}
	if err := m.Sent(FuncReceive, true); err == nil || !strings.Contains(err.Error(), "cannot send F_RECEIVE") || !strings.Contains(err.Error(), "response-pending") {
		t.Fatalf("receive after = %v", err)
	}
}

func TestClientSetupWaitsForBarrierAck(t *testing.T) {
	m := readySetup(t)
	if err := m.Sent(FuncAsyncSendData, false); err != nil {
		t.Fatal(err)
	}
	if err := m.Sent(FuncSendData, false); err != nil || m.State() != StateSendBarrierPending {
		t.Fatalf("senddata = %v state %s", err, m.State())
	}
	if err := m.Sent(FuncAsyncSendData, false); err == nil || !strings.Contains(err.Error(), "send-barrier-pending") {
		t.Fatalf("async during barrier = %v", err)
	}
	ack, err := EncodeDataRecord(DataRecordInput{
		RecordHeaderInput:  RecordHeaderInput{ConversationID: fragConvID(), Info4: p[uint8](2)},
		FunctionCode:       p(FuncSendData),
		CommunicationIndex: 0, ConnectionIndex: fragConnIdx, IsFinal: p(false), Data: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Received(ack); err != nil || m.State() != StateSendContinuation {
		t.Fatalf("ack = %v state %s", err, m.State())
	}
	if err := m.Sent(FuncReceive, true); err != nil || m.State() != StateResponsePending {
		t.Fatalf("final receive = %v state %s", err, m.State())
	}
}

func TestInitialReceiveRequiresAsyncContext(t *testing.T) {
	seq := fragSeq + 1
	response, err := EncodeDataRecord(DataRecordInput{
		RecordHeaderInput:  RecordHeaderInput{ConversationID: fragConvID(), SequenceNumber: &seq},
		FunctionCode:       p(FuncReceive),
		CommunicationIndex: fragCommIdx, ConnectionIndex: fragConnIdx, Data: []byte("response"),
	})
	if err != nil {
		t.Fatal(err)
	}
	be16(response, 50, 34_048)
	be16(response, 58, uint16(len(response)-80))
	if _, err := newDecoder(t, ConversationDecoderOptions{}).Push(response); err == nil || !strings.Contains(err.Error(), "F_RECEIVE") || !strings.Contains(err.Error(), "F_SAP_SEND") {
		t.Fatalf("without allow = %v", err)
	}
	d := newDecoder(t, ConversationDecoderOptions{AllowInitialReceive: true})
	msgs, err := d.Push(response)
	if err != nil || string(msgs[0].Data) != "response" || msgs[0].SequenceNumber != fragSeq+1 {
		t.Fatalf("with allow = %+v, %v", msgs, err)
	}
	if err := d.Finish(); err != nil {
		t.Fatal(err)
	}
}

func TestIncomingRecordsUseActualLengthAtOffset10(t *testing.T) {
	response, err := EncodeDataRecord(DataRecordInput{
		RecordHeaderInput:  RecordHeaderInput{ConversationID: fragConvID()},
		FunctionCode:       p(FuncSapSend),
		CommunicationIndex: 0, ConnectionIndex: fragConnIdx, Data: []byte("reply"),
	})
	if err != nil {
		t.Fatal(err)
	}
	be16(response, 50, 34_048)
	be16(response, 58, uint16(len(response)-80))
	info, err := DecodeIncomingDataOperationInfo(response[48:80])
	if err != nil {
		t.Fatal(err)
	}
	if info != (IncomingDataOperationInfo{DataLength: 5, CommunicationIndex: 0, ConnectionIndex: fragConnIdx}) {
		t.Fatalf("info = %+v", info)
	}
	msgs, err := newDecoder(t, ConversationDecoderOptions{ValidateIncomingDataOperationInfo: true}).Push(response)
	if err != nil || string(msgs[0].Data) != "reply" {
		t.Fatalf("valid = %+v, %v", msgs, err)
	}

	inconsistent := append([]byte(nil), response...)
	be16(inconsistent, 58, 4)
	if _, err := newDecoder(t, ConversationDecoderOptions{ValidateIncomingDataOperationInfo: true}).Push(inconsistent); err == nil || !strings.Contains(err.Error(), "data length 4 does not match record payload length 5") {
		t.Fatalf("inconsistent = %v", err)
	}
}

func TestGenericConversationDecodingForCompactRecords(t *testing.T) {
	request, err := EncodeDataRecord(DataRecordInput{
		RecordHeaderInput:  RecordHeaderInput{ConversationID: fragConvID()},
		CommunicationIndex: 0xffff, ConnectionIndex: fragConnIdx, Data: []byte("request"),
	})
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := newDecoder(t, ConversationDecoderOptions{}).Push(request)
	if err != nil || string(msgs[0].Data) != "request" || msgs[0].CommunicationIndex != 0xffff || msgs[0].ConnectionIndex != fragConnIdx {
		t.Fatalf("msgs = %+v, %v", msgs, err)
	}
}
