// SPDX-License-Identifier: Apache-2.0
//
// Offline end-to-end wiring test for Session.sendData's CpicStreaming path,
// over the same scripted mock transport as session_mock_test.go. Proves that
// opting into CpicStreaming actually drives the multi-fragment
// F_ASEND_DATA/F_RECEIVE sequence internal/appc.PlanOutgoingDataFragments
// already plans, through Session.writeDataPlan and back through
// Session.receiveMessage — without a live SAP system. That SAP accepts and
// reassembles the stream server-side is a live observation, not something
// this test can show (see rfc.Destination.CpicStreaming). Original work.

package client

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/oisee/open-rfc-go/internal/appc"
)

// incomingDataRecordFunc is incomingDataRecord (session_mock_test.go)
// generalised to the caller's choice of F_SAP_SEND/F_RECEIVE, so a reply that
// follows a streamed (F_RECEIVE-terminated) send can be built.
func incomingDataRecordFunc(fn appc.Function, conversationID []byte, connIndex uint16, vector byte, data []byte) []byte {
	record := make([]byte, appc.RecordHeaderLength+len(data))
	record[0] = appc.ProtocolVersion
	record[1] = byte(fn)
	record[31] = vector
	copy(record[40:], conversationID)
	binary.BigEndian.PutUint16(record[50:], 34048)             // receive-buffer capacity (op-info offset 2)
	binary.BigEndian.PutUint16(record[58:], uint16(len(data))) // data length (op-info offset 10)
	binary.BigEndian.PutUint16(record[76:], 0)                 // communication index (op-info offset 28)
	binary.BigEndian.PutUint16(record[78:], connIndex)         // connection index (op-info offset 30)
	copy(record[appc.RecordHeaderLength:], data)
	return record
}

// TestSendDataStreamsPastCompactLimitOverMock proves that, with CpicStreaming
// enabled, a send whose application data exceeds the compact 28000-byte
// F_SAP_SEND slice is planned and written as F_ASEND_DATA fragments plus the
// F_RECEIVE terminator — not rejected — and that the follow-on reply is
// reassembled correctly. 50000 bytes stays under the 21-fragment sync-barrier
// threshold (MaxAsyncSendsBeforeSync), so this exercises the plain streamed
// path; the barrier itself is already covered at the planning layer by
// internal/appc/fragmentation_test.go.
func TestSendDataStreamsPastCompactLimitOverMock(t *testing.T) {
	convID := []byte("CONV0001")
	connIndex := uint16(7)
	replies := handshakeReplies(t, convID, connIndex)
	replies = append(replies, incomingDataRecord(convID, connIndex, appc.VectorEndOfMessage|0x08, logonSuccessCPIC(t)))

	mock := &mockTransport{replies: replies}
	ctx := context.Background()
	sess, err := Open(ctx, SessionOptions{
		Host: "mock", Port: 3200, ApplicationServerService: "sapdp00",
		Transport: mock, CpicStreaming: true,
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := sess.LogonAndPing(ctx, LogonOptions{Client: "001", User: "TESTER", Password: "secret"}); err != nil {
		t.Fatalf("logon: %v", err)
	}

	appData := make([]byte, 50000)
	for i := range appData {
		appData[i] = byte(i % 251)
	}
	replyPayload := []byte("streamed-reply-ok")
	mock.replies = append(mock.replies, incomingDataRecordFunc(appc.FuncReceive, convID, connIndex, appc.VectorEndOfMessage, replyPayload))

	sentBefore := len(mock.sent)
	async, err := sess.sendData(ctx, appData, nil)
	if err != nil {
		t.Fatalf("sendData: %v", err)
	}
	if !async {
		t.Fatal("sendData reported a compact (non-streamed) send for 50000 bytes with CpicStreaming enabled")
	}
	// ceil(50000/28000) = 2 F_ASEND_DATA fragments + 1 F_RECEIVE terminator.
	if got, want := len(mock.sent)-sentBefore, 3; got != want {
		t.Fatalf("wrote %d records for the streamed send, want %d", got, want)
	}
	for i, fn := range []appc.Function{appc.FuncAsyncSendData, appc.FuncAsyncSendData, appc.FuncReceive} {
		if got := appc.Function(mock.sent[sentBefore+i][1]); got != fn {
			t.Fatalf("record %d: function code %s, want %s", i, appc.FunctionName(got), appc.FunctionName(fn))
		}
	}

	reply, err := sess.receiveMessage(ctx, async)
	if err != nil {
		t.Fatalf("receiveMessage: %v", err)
	}
	if string(reply) != string(replyPayload) {
		t.Fatalf("reassembled reply = %q, want %q", reply, replyPayload)
	}
}

// TestSendDataRejectsPastCompactLimitWithoutStreaming pins the current
// (pre-fix) behaviour for a destination that has not opted in: the 28000-byte
// ceiling still applies, with the same error VSP_ISSUES.md §1 observed live.
func TestSendDataRejectsPastCompactLimitWithoutStreaming(t *testing.T) {
	convID := []byte("CONV0001")
	connIndex := uint16(7)
	mock := &mockTransport{replies: handshakeReplies(t, convID, connIndex)}
	ctx := context.Background()
	sess, err := Open(ctx, SessionOptions{Host: "mock", Port: 3200, ApplicationServerService: "sapdp00", Transport: mock})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	sess.authenticated = true // bypass logon; only sendData's own gate is under test

	_, err = sess.sendData(ctx, make([]byte, 50000), nil)
	if err == nil {
		t.Fatal("expected the compact-slice error for 50000 bytes with CpicStreaming left at its default (disabled)")
	}
}
