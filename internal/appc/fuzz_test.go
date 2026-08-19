// SPDX-License-Identifier: Apache-2.0
//
// Go-specific fuzz targets, added per the milestone rule that every decoder of
// network bytes carry one. No upstream analogue. See docs/provenance.md.

package appc

import "testing"

// FuzzDecodeHeader asserts the common-header decoder never panics and that an
// accepted header re-inspects consistently.
func FuzzDecodeHeader(f *testing.F) {
	f.Add(dataRecord(FuncSapSend, 0x0c, []byte("x")))
	f.Add(make([]byte, 48))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		h, err := DecodeHeader(data)
		if err != nil {
			return
		}
		if h.ProtocolVersion != ProtocolVersion {
			t.Fatalf("accepted header with version %#x", h.ProtocolVersion)
		}
		if len(h.ConversationID) != 8 {
			t.Fatalf("accepted header with %d-byte conversation id", len(h.ConversationID))
		}
		if h.FunctionName != FunctionName(h.FunctionCode) {
			t.Fatalf("function name %q inconsistent with code %#x", h.FunctionName, h.FunctionCode)
		}
	})
}

// FuzzConversationDecoderPush drives the peer-controlled reassembly path with
// arbitrary bytes, asserting it never panics and honours its byte budget.
func FuzzConversationDecoderPush(f *testing.F) {
	f.Add(dataRecord(FuncSapSend, 0x0c, []byte("one")))
	f.Add(dataRecord(FuncReceive, 0x00, []byte("part")))
	f.Add(make([]byte, 80))
	f.Fuzz(func(t *testing.T, data []byte) {
		max := 4096
		frags := 8
		d, err := NewConversationDecoder(ConversationDecoderOptions{
			MaxMessageLength: &max, MaxFragments: &frags, AllowInitialReceive: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		messages, err := d.Push(data)
		if err != nil {
			return
		}
		for _, m := range messages {
			if len(m.Data) > max {
				t.Fatalf("message length %d exceeds budget %d", len(m.Data), max)
			}
		}
		_ = d.BufferedByteLength()
	})
}

// FuzzDecodeSynchronousSendAcknowledgement asserts the strict canonical decoder
// never panics on arbitrary input.
func FuzzDecodeSynchronousSendAcknowledgement(f *testing.F) {
	seq := uint32(0)
	ack, _ := EncodeDataRecord(DataRecordInput{
		RecordHeaderInput: RecordHeaderInput{ConversationID: []byte("CONV0001"), SequenceNumber: &seq, Info4: p[uint8](2)},
		FunctionCode:      p(FuncSendData), CommunicationIndex: 0, ConnectionIndex: 6, IsFinal: p(false),
	})
	f.Add(ack)
	f.Add(make([]byte, 80))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeSynchronousSendAcknowledgement(data)
	})
}
