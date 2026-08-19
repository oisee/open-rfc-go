// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"

	"github.com/oisee/open-rfc-go/internal/transport"
)

// ReplayStep is one scripted frame from a capture: a direction and the NI
// payload (without the 4-byte length prefix) to send or expect.
type ReplayStep struct {
	Dir     string // "C->S" | "S->C"
	Payload []byte
}

// capLine mirrors the rfc-sniffer JSONL record. Conn/Label are absent (zero) in
// captures made before per-connection tagging.
type capLine struct {
	Dir   string `json:"dir"`
	Conn  int    `json:"conn"`
	Label string `json:"label"`
	Index int    `json:"index"`
	Len   int    `json:"len"`
	Hex   string `json:"hex"`
}

// LoadConnection reads an rfc-sniffer capture and returns the ordered frames of
// its nth (1-based) connection. When the capture carries per-connection ids
// (conn>0) it segments by id — correct even when connections interleave.
// Otherwise it falls back to the heuristic that a connection begins at a C->S
// 64-byte gateway record. An optional label filters to one proxy's frames.
func LoadConnection(path string, n int, label string) ([]ReplayStep, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []capLine
	dec := json.NewDecoder(bytes.NewReader(data))
	tagged := false
	for {
		var l capLine
		if dec.Decode(&l) != nil {
			break
		}
		if label != "" && l.Label != label {
			continue
		}
		if l.Conn > 0 {
			tagged = true
		}
		lines = append(lines, l)
	}

	var groups [][]capLine
	if tagged {
		byConn := map[int][]capLine{}
		var order []int
		for _, l := range lines {
			if _, ok := byConn[l.Conn]; !ok {
				order = append(order, l.Conn)
			}
			byConn[l.Conn] = append(byConn[l.Conn], l)
		}
		sort.Ints(order)
		for _, id := range order {
			groups = append(groups, byConn[id])
		}
	} else {
		var cur []capLine
		for _, l := range lines {
			if l.Dir == "C->S" && l.Index == 0 && l.Len == 64 {
				if len(cur) > 0 {
					groups = append(groups, cur)
				}
				cur = nil
			}
			cur = append(cur, l)
		}
		if len(cur) > 0 {
			groups = append(groups, cur)
		}
	}

	if n < 1 || n > len(groups) {
		return nil, fmt.Errorf("connection %d out of range (capture has %d)", n, len(groups))
	}
	var steps []ReplayStep
	for _, l := range groups[n-1] {
		payload, err := hex.DecodeString(l.Hex)
		if err != nil {
			continue
		}
		steps = append(steps, ReplayStep{Dir: l.Dir, Payload: payload})
	}
	return steps, nil
}

// ServeReplay answers one live client by replaying a recorded server side. It is
// request-driven: for every frame the client sends, it replies with the next
// recorded server->client frame, in order. This paces to the client and cannot
// deadlock, because a live client's frame sizes and retry counts differ from the
// recording. Recorded C->S frames are used only to count; their content is
// ignored. logf, if non-nil, receives one line per frame.
func ServeReplay(conn net.Conn, script []ReplayStep, logf func(string)) {
	defer conn.Close()
	log := func(s string) {
		if logf != nil {
			logf(s)
		}
	}
	t := transport.New(conn, transport.Options{})
	ctx := context.Background()
	var server [][]byte
	for _, s := range script {
		if s.Dir == "S->C" {
			server = append(server, s.Payload)
		}
	}
	// The client generates the 8-byte conversation id in its CPIC-init record
	// (offset 40) and expects every server record to carry the SAME id. A
	// recorded template holds the capture session's id, so rewrite each outgoing
	// APPC record's id to this client's before sending, or the client falls back.
	// Beyond the conversation id, RFC assigns each connection a 16-byte GUID
	// (anti cross-talk, since 46D). The template holds the capture session's
	// GUID; rewrite it to this client's, found in the client's records, or
	// RSRFCPIN aborts with RFC_INVALID_UUID_DETECTED. goldenGUID is the GUID the
	// template's own frames carry.
	var goldenGUID []byte
	for _, st := range script {
		if g := findRFCGUID(st.Payload); g != nil {
			goldenGUID = g
			break
		}
	}
	var clientConvID, clientGUID []byte
	si := 0
	for {
		got, err := t.Receive(ctx)
		if err != nil {
			log(fmt.Sprintf("client closed after %d server frame(s): %v", si, err))
			return
		}
		if clientConvID == nil {
			if id := convIDOf(got); id != nil {
				clientConvID = id
			}
		}
		if clientGUID == nil {
			if g := findRFCGUID(got); g != nil {
				clientGUID = g
				log(fmt.Sprintf("C->S %5dB %s  rfc-guid=%x", len(got), preview(got), g))
			} else {
				log(fmt.Sprintf("C->S %5dB %s", len(got), preview(got)))
			}
		} else {
			log(fmt.Sprintf("C->S %5dB %s", len(got), preview(got)))
		}
		if si >= len(server) {
			log("(no more recorded server frames; client frame unanswered)")
			continue
		}
		out := withConvID(server[si], clientConvID)
		if goldenGUID != nil && clientGUID != nil {
			out = bytes.ReplaceAll(out, goldenGUID, clientGUID)
		}
		if err := t.Send(out); err != nil {
			log(fmt.Sprintf("S->C send (%dB) failed: %v", len(out), err))
			return
		}
		log(fmt.Sprintf("S->C %5dB %s (recorded #%d)", len(out), preview(out), si))
		si++
	}
}

// convIDOf returns the 8-byte conversation id of an APPC record (protocol 0x06,
// id at offset 40), or nil for a gateway record or a frame too short to hold one.
func convIDOf(frame []byte) []byte {
	if len(frame) >= 48 && frame[0] == 0x06 {
		return append([]byte(nil), frame[40:48]...)
	}
	return nil
}

// withConvID returns a copy of an APPC server record with its conversation id
// (offset 40) overwritten by id. Non-APPC frames (the 64-byte gateway record)
// and a nil id are returned unchanged.
func withConvID(frame, id []byte) []byte {
	if id == nil || len(frame) < 48 || frame[0] != 0x06 {
		return frame
	}
	out := append([]byte(nil), frame...)
	copy(out[40:48], id)
	return out
}

// rfcGUIDNodeSuffix is the last 8 bytes of the RFC connection GUID: the RFC node
// id (0xe100...) plus the host address. It is stable for one SAP host, so a GUID
// is located by finding this suffix and taking the 16 bytes ending at it. This
// value is specific to the A4H test host (172.17.0.3) — an experiment aid, not a
// general solution; proper generation would echo the client's GUID by field.
var rfcGUIDNodeSuffix, _ = hex.DecodeString("e1000000ac110003")

// findRFCGUID returns the first 16-byte RFC connection GUID in b (a window whose
// last 8 bytes are the node suffix), or nil.
func findRFCGUID(b []byte) []byte {
	i := bytes.Index(b, rfcGUIDNodeSuffix)
	if i < 8 {
		return nil
	}
	return append([]byte(nil), b[i-8:i+8]...)
}

func preview(b []byte) string {
	if len(b) > 12 {
		b = b[:12]
	}
	return hex.EncodeToString(b)
}
