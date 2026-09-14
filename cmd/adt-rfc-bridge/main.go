// SPDX-License-Identifier: Apache-2.0
//
// adt-rfc-bridge answers ADT over RFC and forwards it to anything that speaks
// HTTP.
//
// Eclipse's ABAP Development Tools do not reach a system over the ICM: every
// request travels inside one RFC call to SADT_REST_RFC_ENDPOINT, on the
// gateway port, carrying a whole HTTP exchange as its argument. Measured, not
// assumed — a captured session put 579 KB across the gateway and not one byte
// across the ICM.
//
// So this listens where the gateway would, unwraps the exchange, makes it
// against --backend, and wraps the answer. It has no opinion about ADT at all.
//
// That is what makes it testable, and the reason --backend exists from the
// first line rather than later:
//
//	--backend http://a-real-system:50000   proves the TRANSPORT, because the
//	                                       backend is known to be right
//	--backend http://localhost:8099        proves the BACKEND, because the
//	                                       transport already is
//
// Two unknowns that would otherwise hide inside each other, separated by one
// flag. Without it a failure means "something is wrong" and nothing more.
//
// The gateway is not encrypted and carries a logon, so point this at a sandbox
// and at nothing else.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

func main() {
	listen := flag.String("listen", ":3300", "gateway address Eclipse connects to")
	backend := flag.String("backend", "", "origin the ADT requests are made against, e.g. http://localhost:8099")
	config := flag.String("config", "", "JSON file with the backend and its credentials; see -help")
	timeout := flag.Duration("timeout", 120*time.Second, "how long one backend request may take")
	verbose := flag.Bool("verbose", false, "log every frame decision")
	flag.Parse()

	target, err := rfcserver.LoadBackend(*config)
	if err != nil {
		log.Fatalf("adt-rfc-bridge: %v", err)
	}
	if *backend != "" {
		target.URL = *backend
	}
	if target.URL == "" {
		fmt.Fprintln(os.Stderr, "adt-rfc-bridge: no backend; give --backend, or --config a file that names one")
		flag.Usage()
		os.Exit(2)
	}

	handler, err := rfcserver.ADTRestHandler(target, &http.Client{Timeout: *timeout})
	if err != nil {
		log.Fatalf("adt-rfc-bridge: %v", err)
	}
	dispatcher := rfcserver.NewDispatcher()
	dispatcher.Handle("SADT_REST_RFC_ENDPOINT", handler)

	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("adt-rfc-bridge: %v", err)
	}
	defer listener.Close()
	log.Printf("adt-rfc-bridge: %s -> %s", *listen, target.Describe())
	log.Printf("adt-rfc-bridge: point an ABAP project at this host, and at the instance this port belongs to")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("adt-rfc-bridge: accept: %v", err)
			return
		}
		go func() {
			peer := conn.RemoteAddr().String()
			log.Printf("adt-rfc-bridge: %s connected", peer)
			logf := func(string) {}
			if *verbose {
				logf = func(s string) { log.Printf("adt-rfc-bridge: %s: %s", peer, s) }
			}
			rfcserver.ServeConscious(conn, dispatcher, logf, nil)
			log.Printf("adt-rfc-bridge: %s gone", peer)
		}()
	}
}
