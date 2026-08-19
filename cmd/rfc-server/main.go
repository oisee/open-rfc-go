// SPDX-License-Identifier: Apache-2.0
//
// rfc-server is the M8 server-side experiment: it makes THIS host answer a real
// SAP RFC client (an SM59 type-3 destination whose target host points here).
//
// Its first mode is -replay: given an rfc-sniffer capture of a known-good
// session, it plays the recorded server->client frames back in lockstep with
// the live client's client->server frames. If a real SAP Connection Test / the
// traffic generator reaches rc=0 against pure replay, our server-side NI +
// gateway + APPC transport is byte-correct end to end, and we have isolated
// exactly which bytes the handshake and each response need — before investing in
// generating them from the Dispatcher.
//
// A capture may contain credentials (the logon scramble). Only replay a capture
// you own, on a host you control, against your own SAP system.
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/oisee/open-rfc-go/internal/transport"
)

// capLine mirrors the rfc-sniffer JSONL record (dir/index/len/hex).
type capLine struct {
	Dir   string `json:"dir"`
	Index int    `json:"index"`
	Len   int    `json:"len"`
	Hex   string `json:"hex"`
}

// step is one scripted frame: a direction and the NI payload to send/expect.
type step struct {
	dir     string
	payload []byte
}

func main() {
	listen := flag.String("listen", ":3300", "listen address (SM59 target host:sapgw<nn> maps here)")
	capPath := flag.String("replay", "", "rfc-sniffer capture (JSONL) to replay the server side from")
	connN := flag.Int("conn", 1, "1-based connection within the capture to replay")
	verbose := flag.Bool("v", false, "log every scripted frame")
	once := flag.Bool("once", false, "serve a single connection then exit")
	flag.Parse()

	if *capPath == "" {
		fmt.Fprintln(os.Stderr, "rfc-server: -replay <capture.jsonl> is required (only replay mode exists so far)")
		os.Exit(2)
	}
	script, err := loadConnection(*capPath, *connN)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rfc-server:", err)
		os.Exit(1)
	}
	nS, nC := 0, 0
	for _, s := range script {
		if s.dir == "S->C" {
			nS++
		} else {
			nC++
		}
	}
	fmt.Printf("rfc-server: replaying connection %d — %d frames (%d server, %d client) on %s\n",
		*connN, len(script), nS, nC, *listen)

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rfc-server:", err)
		os.Exit(1)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, "rfc-server: accept:", err)
			continue
		}
		fmt.Printf("rfc-server: client connected from %s\n", conn.RemoteAddr())
		serveReplay(conn, script, *verbose)
		if *once {
			return
		}
	}
}

// serveReplay answers one live client. It is request-driven: for every frame the
// client sends, it replies with the next recorded server->client frame, in order.
// This paces to the client and cannot deadlock (unlike strict positional replay),
// which matters because a live client's frame sizes and retry counts differ from
// the recording. The recorded C->S frames are used only to count/skip; content is
// ignored.
func serveReplay(conn net.Conn, script []step, verbose bool) {
	defer conn.Close()
	t := transport.New(conn, transport.Options{})
	ctx := context.Background()
	var server [][]byte
	for _, s := range script {
		if s.dir == "S->C" {
			server = append(server, s.payload)
		}
	}
	si := 0
	for {
		got, err := t.Receive(ctx)
		if err != nil {
			fmt.Printf("  client closed after %d server frame(s): %v\n", si, err)
			return
		}
		if verbose {
			fmt.Printf("  C->S  %5dB  %s\n", len(got), preview(got))
		}
		if si >= len(server) {
			fmt.Printf("  (no more recorded server frames; %d client frame(s) unanswered)\n", 1)
			continue
		}
		if err := t.Send(server[si]); err != nil {
			fmt.Printf("  S->C send (%dB) failed: %v\n", len(server[si]), err)
			return
		}
		if verbose {
			fmt.Printf("  S->C  %5dB  %s  (recorded #%d)\n", len(server[si]), preview(server[si]), si)
		}
		si++
	}
}

func preview(b []byte) string {
	if len(b) > 12 {
		b = b[:12]
	}
	return hex.EncodeToString(b)
}

// loadConnection reads the capture and returns the ordered frames of the nth
// (1-based) connection. A connection begins at a C->S 64-byte gateway record
// (index 0).
func loadConnection(path string, n int) ([]step, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conns [][]step
	var cur []step
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var l capLine
		if err := dec.Decode(&l); err != nil {
			break
		}
		payload, err := hex.DecodeString(l.Hex)
		if err != nil {
			continue
		}
		if l.Dir == "C->S" && l.Index == 0 && l.Len == 64 {
			if len(cur) > 0 {
				conns = append(conns, cur)
			}
			cur = nil
		}
		cur = append(cur, step{dir: l.Dir, payload: payload})
	}
	if len(cur) > 0 {
		conns = append(conns, cur)
	}
	if n < 1 || n > len(conns) {
		return nil, fmt.Errorf("connection %d out of range (capture has %d)", n, len(conns))
	}
	return conns[n-1], nil
}
