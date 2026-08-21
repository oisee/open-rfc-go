// SPDX-License-Identifier: Apache-2.0
//
// rfc-typet impersonates the gateway *and* the registered server for a type-T
// destination and ANSWERS the function calls the client makes — the smaller
// sibling of the type-3 conscious server (a registered server authenticates no
// one, so there is no logon/serializer negotiation to fake).
//
// Live drive: point an SM59 type-T destination's Gateway Host at this box
// (progid does not matter — we accept any), then from ABAP call, e.g.
//
//	CALL FUNCTION 'STFC_CONNECTION' DESTINATION 'ZSNIFF_TCP'
//	     EXPORTING requtext = 'hi' IMPORTING echotext = t resptext = r.
//
// The default dispatcher answers STFC_CONNECTION (echoes REQUTEXT) and RFC_PING.
// Every frame is dumped so a first real function-call CUT is captured for study.
package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

// ztstDispatcher answers the FMs that Z_CALL_RFC drives over its destination
// (Z_DOUBLE, Z_GREET) plus the STFC_CONNECTION echo. It is the first real
// function call answered by the type-T (registered-server) role.
func ztstDispatcher() *rfcserver.Dispatcher {
	d := rfcserver.DefaultDispatcher() // STFC_CONNECTION + RFC_PING baseline

	// Z_DOUBLE: IMPORTING n TYPE i -> EXPORTING result TYPE i. ABAP INT4 rides
	// the classic wire as 4 little-endian bytes on this LE unicode system.
	d.Handle("Z_DOUBLE", func(ctx context.Context, req rfcserver.Request) (rfcserver.Response, error) {
		var n int32
		for _, imp := range req.Imports {
			if imp.Name == "N" {
				fmt.Printf("  [Z_DOUBLE] N raw = %x (%d bytes)\n", imp.Value, len(imp.Value))
				if len(imp.Value) >= 4 {
					n = int32(binary.LittleEndian.Uint32(imp.Value[:4]))
				}
			}
		}
		out := make([]byte, 4)
		binary.LittleEndian.PutUint32(out, uint32(n*2))
		return rfcserver.Response{Exports: []cpic.NamedValue{{Name: "RESULT", Value: out}}}, nil
	})

	// Z_GREET: IMPORTING name TYPE char -> EXPORTING greeting TYPE char90.
	d.Handle("Z_GREET", func(ctx context.Context, req rfcserver.Request) (rfcserver.Response, error) {
		name := ""
		for _, imp := range req.Imports {
			if imp.Name == "NAME" {
				name, _ = classicrfc.DecodeAbapChar(imp.Value)
			}
		}
		greeting, err := classicrfc.EncodeAbapChar("Hello, "+name+"! — from open-rfc-go type-T server", 90)
		if err != nil {
			return rfcserver.Response{}, err
		}
		return rfcserver.Response{Exports: []cpic.NamedValue{{Name: "GREETING", Value: greeting}}}, nil
	})
	return d
}

func main() {
	listen := ":3300"
	if len(os.Args) > 1 {
		listen = os.Args[1]
	}
	dumpPath := "cap-typet.jsonl"
	if len(os.Args) > 2 {
		dumpPath = os.Args[2]
	}
	df, err := os.Create(dumpPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer df.Close()
	var mu sync.Mutex
	enc := json.NewEncoder(df)
	dump := func(dir string, frame []byte) {
		mu.Lock()
		defer mu.Unlock()
		_ = enc.Encode(map[string]any{"dir": dir, "len": len(frame), "hex": hex.EncodeToString(frame)})
	}

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	d := ztstDispatcher()
	fmt.Printf("rfc-typet: type-T registered-server on %s, dump -> %s\n", listen, dumpPath)
	fmt.Println("point an SM59 type-T destination's Gateway Host here, then CALL FUNCTION ... DESTINATION <it>")
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		fmt.Printf("client %s\n", conn.RemoteAddr())
		go rfcserver.ServeTypeT(conn, d, func(s string) { fmt.Println("  " + s) }, dump)
	}
}
