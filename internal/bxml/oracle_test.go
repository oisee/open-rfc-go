// SPDX-License-Identifier: Apache-2.0

package bxml

import (
	"bufio"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// Every document of a real capture, read and written back.
//
// The capture is not in the repository — it carries a session — so this runs
// only when OPEN_RFC_BXML_ORACLE names a gateway capture (the JSONL a tap
// writes) and is skipped otherwise. What it proves when it runs: the reader
// consumes every 4001 and 4002 payload in both directions to the last byte,
// and the writer reproduces each one exactly from what the reader made of it.
// That second half is the part that matters for a server: the bytes we send
// are the bytes the system sends.
func TestOracleRoundTrip(t *testing.T) {
	path := os.Getenv("OPEN_RFC_BXML_ORACLE")
	if path == "" {
		t.Skip("OPEN_RFC_BXML_ORACLE not set")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	var docs, roundTripped int
	for sc.Scan() {
		var r struct {
			Dir string `json:"dir"`
			Hex string `json:"hex"`
		}
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			continue
		}
		frame, _ := hex.DecodeString(r.Hex)
		if len(frame) < 83 || frame[0] != 0x06 || frame[1] != 0xcb || frame[80] == 0xd9 {
			continue
		}
		body := frame[80:]
		if !(body[0] == 0 && body[1] == 0) {
			body = append([]byte{0, 0}, body...)
		}
		fields := chainFields(body)
		var payload []byte
		for _, tag := range []uint16{0x4001, 0x4002} {
			for _, v := range fields[tag] {
				payload = append(payload, v...)
			}
		}
		if len(payload) == 0 {
			continue
		}
		if flag := fields[0x4000]; len(flag) == 1 && len(flag[0]) == 2 && flag[0][1] == 0x01 {
			inflated, err := io.ReadAll(flate.NewReader(bytes.NewReader(payload)))
			if err != nil {
				// a response split across records is incomplete here; not this test's business
				continue
			}
			payload = inflated
		}
		docs++
		tree, err := Decode(payload)
		if err != nil {
			t.Errorf("%s document %d: %v", r.Dir, docs, err)
			continue
		}
		root, err := PayloadRoot(tree)
		if err != nil {
			t.Errorf("%s document %d: %v", r.Dir, docs, err)
			continue
		}
		again, err := Encode(root)
		if err != nil {
			t.Errorf("%s document %d: encode: %v", r.Dir, docs, err)
			continue
		}
		if !bytes.Equal(again, payload) {
			at := 0
			for at < len(again) && at < len(payload) && again[at] == payload[at] {
				at++
			}
			t.Errorf("%s document %d (%s): re-encoded bytes differ at %d of %d", r.Dir, docs, root.Name, at, len(payload))
			continue
		}
		roundTripped++
	}
	if docs == 0 {
		t.Fatal("the capture held no BXML documents")
	}
	t.Logf("%d documents read, %d written back identically", docs, roundTripped)
	if roundTripped != docs {
		t.Fail()
	}
	_ = strings.TrimSpace
}

func chainFields(d []byte) map[uint16][][]byte {
	out := map[uint16][][]byte{}
	prev := uint16(0)
	off := 0
	for off+6 <= len(d) {
		p := binary.BigEndian.Uint16(d[off:])
		t := binary.BigEndian.Uint16(d[off+2:])
		n := int(binary.BigEndian.Uint16(d[off+4:]))
		if p != prev || off+6+n > len(d) {
			break
		}
		out[t] = append(out[t], d[off+6:off+6+n])
		off += 6 + n
		prev = t
		if t == 0xffff {
			break
		}
	}
	return out
}
