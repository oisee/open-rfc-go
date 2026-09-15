// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/bxml"
	"github.com/oisee/open-rfc-go/internal/cpic"
)

// Every call of a real Eclipse session, through this server's decoder and,
// for the calls it answers itself, through its handlers — with the answer's
// shape set beside the system's.
//
// Runs only when OPEN_RFC_ECLIPSE_ORACLE names a gateway capture; the capture
// carries a session and is not in the repository. What it proves: every
// frame Eclipse sent decodes; every SADT_REST_RFC_ENDPOINT call carries a
// REQUEST document this server can read; and for RFC_GET_FUNCTION_INTERFACE
// and DDIF_FIELDINFO_GET the sequence of tags this server answers with is the
// sequence the system answered with, allowing for the one difference by
// design — the system compresses a large table (0305 chunks, 0306 end) and
// this server sends its rows as they are (0303).
func TestOracleEclipseCalls(t *testing.T) {
	path := os.Getenv("OPEN_RFC_ECLIPSE_ORACLE")
	if path == "" {
		t.Skip("OPEN_RFC_ECLIPSE_ORACLE not set")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	type frame struct {
		Dir   string `json:"dir"`
		Conn  int    `json:"conn"`
		Index int    `json:"index"`
		Hex   string `json:"hex"`
	}
	var frames []frame
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		var fr frame
		if json.Unmarshal(sc.Bytes(), &fr) == nil && fr.Hex != "" {
			frames = append(frames, fr)
		}
	}
	d := NewDispatcher()
	d.Handle("RFC_GET_FUNCTION_INTERFACE", FunctionInterfaceHandler())
	d.Handle("DDIF_FIELDINFO_GET", FieldInfoHandler())

	calls := map[string]int{}
	compared := 0
	for i, fr := range frames {
		b, _ := hex.DecodeString(fr.Hex)
		if fr.Dir != "C->S" || len(b) < 83 || b[0] != 0x06 || b[1] != 0xcb || isEclipseLogon(b[80:]) || b[30] != recordInfoFinal {
			continue
		}
		req, err := DecodeFunctionRequest(b[80:])
		if err != nil {
			t.Errorf("conn %d frame %d: %v", fr.Conn, fr.Index, err)
			continue
		}
		calls[req.FunctionName]++
		// Metadata calls open with 0502 and declare no byte order; their names
		// still decode by the fallback. Only the compact SADT call opens with
		// 0101 and declares one.
		// Every call carries the session GUID. Eclipse-ness is a property of
		// the conversation (it did an Eclipse logon), not of a call's bytes,
		// so it is not asserted on the bare decoder here.
		if len(req.SessionGUID) != 16 {
			t.Errorf("conn %d frame %d: no session GUID", fr.Conn, fr.Index)
		}
		switch req.FunctionName {
		case adtRestFunction:
			if req.Compact == nil {
				t.Errorf("conn %d frame %d: no compact parameter", fr.Conn, fr.Index)
				continue
			}
			tree, err := bxml.Decode(req.Compact)
			if err != nil {
				t.Errorf("conn %d frame %d: %v", fr.Conn, fr.Index, err)
				continue
			}
			if root, _ := bxml.PayloadRoot(tree); root == nil || root.Name != "REQUEST" || root.Child("REQUEST_LINE").ChildText("METHOD") == "" {
				t.Errorf("conn %d frame %d: not a REQUEST document", fr.Conn, fr.Index)
			}
		case "RFC_GET_FUNCTION_INTERFACE", "DDIF_FIELDINFO_GET":
			resp, key, cause := d.Invoke(context.Background(), req)
			if key != "" {
				t.Errorf("conn %d frame %d: %s -> %s: %v", fr.Conn, fr.Index, req.FunctionName, key, cause)
				continue
			}
			cut, err := EncodeEclipseResponse(resp, req)
			if err != nil {
				t.Errorf("conn %d frame %d: %v", fr.Conn, fr.Index, err)
				continue
			}
			ours := shape(responseFields(t, cut))
			// the system's answer is the next frame the other way on this connection
			var theirs string
			for _, later := range frames[i+1:] {
				if later.Conn == fr.Conn && later.Dir == "S->C" {
					sb, _ := hex.DecodeString(later.Hex)
					body := sb[80:]
					if !(body[0] == 0 && body[1] == 0) {
						body = append([]byte{0, 0}, body...)
					}
					decoded, err := cpic.DecodeFieldChainPrefix(body, 0x0000, uint16(cpic.TagEnd), cpic.FieldChainLimits{})
					if err != nil {
						t.Errorf("conn %d frame %d: the system's answer: %v", later.Conn, later.Index, err)
						break
					}
					fields := decoded.Fields
					// drop the session preamble the system sometimes prepends
					for at, fl := range fields {
						if fl.Tag == uint16(cpic.TagResponseStart) {
							fields = fields[at+1:]
							break
						}
					}
					theirs = shape(fields)
					break
				}
			}
			if ours != theirs {
				t.Errorf("conn %d frame %d %s:\n ours   %s\n theirs %s", fr.Conn, fr.Index, req.FunctionName, ours, theirs)
			}
			compared++
		}
	}
	t.Logf("calls: %v; %d metadata answers compared", calls, compared)
	if calls[adtRestFunction] == 0 {
		t.Fatal("the capture held no ADT calls")
	}
}

// shape names a response's tags and, for the fields whose length is part of
// the contract, their lengths. Compressed table chunks are folded into the
// uncompressed form so the two sides compare.
func shape(fields []cpic.Field) string {
	var parts []string
	for _, f := range fields {
		switch f.Tag {
		case 0x0306:
			continue // the compressed-table end marker; our uncompressed rows have none
		case 0x0303, 0x0305:
			// a table row, compressed (0305) on the system's side or plain
			// (0303) on ours; compared as "rows" so the framing choice does
			// not fail an otherwise-identical answer
			parts = append(parts, "rows")
		case 0x0203, 0x0302, 0x0335:
			parts = append(parts, fmt.Sprintf("%04x[%d]", f.Tag, len(f.Value)))
		case 0x3c05:
			parts = append(parts, "3c05")
		default:
			parts = append(parts, fmt.Sprintf("%04x", f.Tag))
		}
	}
	// collapse runs of 3c05 and 0303
	var out []string
	for _, p := range parts {
		if len(out) > 0 && out[len(out)-1] == p && (p == "3c05" || p == "rows") {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, " ")
}
