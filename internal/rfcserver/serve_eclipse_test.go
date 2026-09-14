// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

// A real Eclipse conversation, replayed against our own server.
//
// The fixture is one complete gateway conversation captured between Eclipse
// ADT and an AS ABAP 1909 sandbox, recorded by a forwarder sitting on the
// gateway port. Seven frames, both directions: the gateway record and its
// acknowledgement, F_INITIALIZE and its reply, F_SET_PARTNER_LU_NAME, which
// is answered by silence, and F_ALLOCATE with its reply.
//
// The bytes are as captured except that addresses, host and user names have
// been replaced by documentation values of exactly the same length, so every
// offset still lands where it did on the wire. This is a public repository
// and a capture is not a fixture until nobody's name is in it.
//
// The question it asks is the only one that matters before any of this is
// built: given what Eclipse actually sends, does our server say what the
// system said? A handler that dispatches correctly is worth nothing if the
// client never gets far enough to call it, and the capture shows Eclipse
// opening with F_INITIALIZE, F_SET_PARTNER_LU_NAME and F_ALLOCATE — a
// sequence ServeConscious does not implement today.
//
// So this test is expected to fail, and the failure is the specification: it
// names the first frame where we diverge and shows both sides of it.
type capturedFrame struct {
	Dir string `json:"dir"`
	Len int    `json:"len"`
	Hex string `json:"hex"`
}

func loadEclipseConversation(t *testing.T) []capturedFrame {
	t.Helper()
	raw, err := os.ReadFile("testdata/eclipse-gateway-conversation.json")
	if err != nil {
		t.Skipf("no captured conversation: %v", err)
	}
	var frames []capturedFrame
	if err := json.Unmarshal(raw, &frames); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if len(frames) == 0 {
		t.Fatal("fixture is empty")
	}
	return frames
}

func TestServeConsciousAnswersEclipseHandshake(t *testing.T) {
	frames := loadEclipseConversation(t)

	client, server := net.Pipe()
	defer client.Close()
	go func() {
		defer server.Close()
		ServeConscious(server, DefaultDispatcher(), func(string) {}, nil)
	}()

	for at, frame := range frames {
		payload, err := hex.DecodeString(frame.Hex)
		if err != nil {
			t.Fatalf("frame %d: %v", at, err)
		}
		switch frame.Dir {
		case "C->S":
			// the capture holds NI payloads; the length prefix the sniffer
			// stripped has to go back on, or the server's decoder waits for
			// a frame that never arrives and the test blames the wrong thing
			_ = client.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if _, err := client.Write(niFrame(payload)); err != nil {
				t.Fatalf("frame %d: the server would not take what Eclipse sent: %v", at, err)
			}
		case "S->C":
			_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
			got, err := readNiFrame(client)
			if err != nil {
				t.Fatalf("frame %d: the system answered %d bytes here and we answered nothing: %v",
					at, len(payload), err)
			}
			// One field is ours to choose rather than to match: the
			// conversation id the system writes into its F_INITIALIZE reply.
			// It is masked out of the comparison and checked for its shape
			// instead — eight ASCII digits — because a test that accepted
			// anything there would also accept an empty one.
			if isInitializeReply(payload) {
				id := got[appcInitConvOffset : appcInitConvOffset+appcInitConvLength]
				for _, c := range id {
					if c < '0' || c > '9' {
						t.Fatalf("frame %d: the conversation id is %q, not eight digits", at, id)
					}
				}
				copy(got[appcInitConvOffset:], payload[appcInitConvOffset:appcInitConvOffset+appcInitConvLength])
			}
			if string(got) != string(payload) {
				t.Fatalf("frame %d diverges.\n  system: %d bytes, %s…\n  ours:   %d bytes, %s…\n  first differing byte: %d",
					at, len(payload), hex.EncodeToString(payload[:min(24, len(payload))]),
					len(got), hex.EncodeToString(got[:min(24, len(got))]), firstDifference(payload, got))
			}
		default:
			t.Fatalf("frame %d: unknown direction %q", at, frame.Dir)
		}
	}
}

// NI framing, by hand rather than through the package the server uses, so a
// fault in that package cannot hide inside the harness that checks it.
func niFrame(payload []byte) []byte {
	framed := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(framed, uint32(len(payload)))
	copy(framed[4:], payload)
	return framed
}

func readNiFrame(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	payload := make([]byte, binary.BigEndian.Uint32(header[:]))
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isInitializeReply(frame []byte) bool {
	return len(frame) == appcInitReplyLength && frame[0] == 0x06 && frame[1] == 0x01
}

func firstDifference(a, b []byte) int {
	for i := 0; i < min(len(a), len(b)); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}
