// SPDX-License-Identifier: Apache-2.0
//
// rfc-server makes THIS host answer a real SAP RFC client (an SM59 type-3
// destination whose target host points here). Its first mode is -replay: given
// an rfc-sniffer capture of a known-good session, it plays the recorded
// server->client frames back, request-driven, in lockstep with the live client.
//
// If a real SAP Connection Test reaches rc=0 against pure replay, our
// server-side NI + gateway + APPC transport is byte-correct end to end. A live
// experiment against A4H showed the client accepts our replayed gateway record
// and CPIC logon-accept and proceeds to issue RFC calls (the hardest gate); the
// open work is generating the responses (see docs/roadmap.md).
//
// A capture may contain credentials. Only replay a capture you own, on a host
// you control, against your own SAP system.
package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

func main() {
	listen := flag.String("listen", ":3300", "listen address (SM59 target host:sapgw<nn> maps here)")
	capPath := flag.String("replay", "", "rfc-sniffer capture (JSONL) to replay the server side from")
	connN := flag.Int("conn", 1, "1-based connection within the capture to replay")
	label := flag.String("label", "", "only use frames captured under this proxy label")
	verbose := flag.Bool("v", false, "log every scripted frame")
	once := flag.Bool("once", false, "serve a single connection then exit")
	flag.Parse()

	if *capPath == "" {
		fmt.Fprintln(os.Stderr, "rfc-server: -replay <capture.jsonl> is required (only replay mode exists so far)")
		os.Exit(2)
	}
	script, err := rfcserver.LoadConnection(*capPath, *connN, *label)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rfc-server:", err)
		os.Exit(1)
	}
	nS, nC := 0, 0
	for _, s := range script {
		if s.Dir == "S->C" {
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
		logf := func(s string) {
			if *verbose {
				fmt.Println("  " + s)
			}
		}
		rfcserver.ServeReplay(conn, script, logf)
		if *once {
			return
		}
	}
}
