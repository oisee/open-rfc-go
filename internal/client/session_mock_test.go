// SPDX-License-Identifier: Apache-2.0
//
// Offline end-to-end wiring test for the session over a scripted mock transport:
// gateway + APPC handshake, then a CPIC initial-logon exchange whose reply is a
// hand-built APPC data record wrapping a CPIC logon-success envelope. Proves the
// session's own redesign logic (handshake sequencing, APPC data-plan write, and
// reply reassembly) without a network. Original work. See docs/provenance.md.

package client

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/oisee/open-rfc-go/internal/appc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/gateway"
)

type mockTransport struct {
	replies [][]byte
	sent    [][]byte
}

func (m *mockTransport) Send(payload []byte) error {
	m.sent = append(m.sent, append([]byte(nil), payload...))
	return nil
}
func (m *mockTransport) Receive(ctx context.Context) ([]byte, error) {
	if len(m.replies) == 0 {
		return nil, ErrClosedForTest
	}
	r := m.replies[0]
	m.replies = m.replies[1:]
	return r, nil
}
func (m *mockTransport) Close() error { return nil }

var ErrClosedForTest = errorString("mock transport drained")

type errorString string

func (e errorString) Error() string { return string(e) }

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// incomingDataRecord builds a server-side APPC F_SAP_SEND data record whose
// incoming operation-info carries the actual data length at offset 10, the
// communication index (0) at 28, and the connection index at 30.
func incomingDataRecord(conversationID []byte, connIndex uint16, vector byte, data []byte) []byte {
	record := make([]byte, appc.RecordHeaderLength+len(data))
	record[0] = appc.ProtocolVersion
	record[1] = byte(appc.FuncSapSend)
	record[31] = vector
	copy(record[40:], conversationID)
	binary.BigEndian.PutUint16(record[50:], 34048)             // receive-buffer capacity (op-info offset 2)
	binary.BigEndian.PutUint16(record[58:], uint16(len(data))) // data length (op-info offset 10)
	binary.BigEndian.PutUint16(record[76:], 0)                 // communication index (op-info offset 28)
	binary.BigEndian.PutUint16(record[78:], connIndex)         // connection index (op-info offset 30)
	copy(record[appc.RecordHeaderLength:], data)
	return record
}

func handshakeReplies(t *testing.T, convID []byte, connIndex uint16) [][]byte {
	gw, err := gateway.EncodeNormalClient(gateway.NormalClientRecord{
		Address: "127.0.0.1", Service: "srv", CodePage: "4103", GatewayOptionLevel: 15,
		AppcHeaderVersion: 6,
		AcceptInfo:        gateway.AcceptInfoExtendedInitOptions | gateway.AcceptInfoCodePage | gateway.AcceptInfoErrorInfo,
		Index:             0,
	})
	if err != nil {
		t.Fatal(err)
	}
	initReply, err := appc.EncodeControlRecord(appc.ControlRecordInput{
		RecordHeaderInput: appc.RecordHeaderInput{ConversationID: convID},
		FunctionCode:      appc.FuncInitialize,
		ExtendedInfo:      &appc.ExtendedInfo{ConnectionIndex: connIndex},
	})
	if err != nil {
		t.Fatal(err)
	}
	allocReply, err := appc.EncodeControlRecord(appc.ControlRecordInput{
		RecordHeaderInput: appc.RecordHeaderInput{ConversationID: convID},
		FunctionCode:      appc.FuncAllocate,
	})
	if err != nil {
		t.Fatal(err)
	}
	return [][]byte{gw, initReply, allocReply}
}

func logonSuccessCPIC(t *testing.T) []byte {
	chain, err := cpic.EncodeFieldChain(uint16(cpic.TagStart), []cpic.Field{
		{Tag: uint16(cpic.TagProtocolVersion), Value: mustHex(t, "00000e0b")},
		{Tag: uint16(cpic.TagCapabilities), Value: make([]byte, 11)},
		{Tag: uint16(cpic.TagLogonStatus), Value: []byte{0}},
		{Tag: uint16(cpic.TagUnresolved0420), Value: make([]byte, 4)},
		{Tag: uint16(cpic.TagEnd), Value: nil},
	}, cpic.FieldChainLimits{})
	if err != nil {
		t.Fatal(err)
	}
	out := append([]byte(nil), mustHex(t, "010100080101010504010003")...)
	out = append(out, chain...)
	return append(out, 0xff, 0xff)
}

func TestSessionHandshakeAndLogonOverMock(t *testing.T) {
	convID := []byte("CONV0001")
	connIndex := uint16(7)
	replies := handshakeReplies(t, convID, connIndex)
	// The logon exchange reply: CPIC logon success wrapped in a final APPC record.
	replies = append(replies, incomingDataRecord(convID, connIndex, appc.VectorEndOfMessage|0x08, logonSuccessCPIC(t)))

	mock := &mockTransport{replies: replies}
	ctx := context.Background()
	sess, err := Open(ctx, SessionOptions{Host: "mock", Port: 3200, ApplicationServerService: "sapdp00", Transport: mock})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Handshake sent gateway + Initialize + SetPartnerLuName + Allocate = 4 records.
	if len(mock.sent) != 4 {
		t.Fatalf("handshake sent %d records, want 4", len(mock.sent))
	}
	if err := sess.LogonAndPing(ctx, LogonOptions{Client: "001", User: "TESTER", Password: "secret"}); err != nil {
		t.Fatalf("logon: %v", err)
	}
	if !sess.Authenticated() {
		t.Fatal("session not authenticated after successful logon")
	}
	// Logon sent one more record (the F_SAP_SEND carrying the CPIC logon request).
	if len(mock.sent) != 5 {
		t.Fatalf("after logon sent %d records, want 5", len(mock.sent))
	}
}

func TestSessionRejectsWrongCodePage(t *testing.T) {
	gw, _ := gateway.EncodeNormalClient(gateway.NormalClientRecord{
		Address: "127.0.0.1", Service: "srv", CodePage: "1100", AppcHeaderVersion: 6,
		AcceptInfo: gateway.AcceptInfoExtendedInitOptions | gateway.AcceptInfoCodePage, Index: 0,
	})
	mock := &mockTransport{replies: [][]byte{gw}}
	if _, err := Open(context.Background(), SessionOptions{Host: "mock", Port: 3200, ApplicationServerService: "sapdp00", Transport: mock}); err == nil {
		t.Fatal("expected code-page rejection")
	}
}
