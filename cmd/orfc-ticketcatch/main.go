// SPDX-License-Identifier: Apache-2.0
//
// rfc-ticketcatch plays the SAP gateway for a type-T registered-server
// destination and dumps the CPIC logon the client sends after the ALLOCATE is
// accepted — the one place a forwarded logon ticket appears on the classic-RFC
// wire. Point an SM59 type-T destination's Gateway Host at this box, tick
// "Send Assertion Ticket", and run the connection test.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

func main() {
	listen := ":3300"
	if len(os.Args) > 1 {
		listen = os.Args[1]
	}
	dumpPath := "cap-ticketcatch.jsonl"
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
	fmt.Printf("rfc-ticketcatch: gateway impersonator on %s, dump -> %s\n", listen, dumpPath)
	fmt.Println("point an SM59 type-T destination's Gateway Host here, tick Send Assertion Ticket, run the test")
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		fmt.Printf("client %s\n", conn.RemoteAddr())
		go rfcserver.ServeTicketCatch(conn, func(s string) { fmt.Println("  " + s) }, dump)
	}
}
