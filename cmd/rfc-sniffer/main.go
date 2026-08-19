// SPDX-License-Identifier: Apache-2.0
//
// rfc-sniffer is a framing-aware RFC proxy: point an RFC client (or SAP
// destination) at -listen, and it forwards to -target (a real SAP gateway)
// while logging every NI record it observes in both directions. Use it to learn
// the wire (e.g. GW_REGISTER for the RFC-server track) and to capture real
// xRFC / deep-structure traffic. It never alters the bytes it forwards.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/oisee/open-rfc-go/internal/sniffer"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:3399", "local address to listen on")
	target := flag.String("target", "", "host:port of the real SAP gateway (e.g. sap.example:3300)")
	verbose := flag.Bool("v", false, "log a hex preview of each frame payload")
	flag.Parse()
	if *target == "" {
		log.Fatal("rfc-sniffer: -target is required (host:port of the SAP gateway)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	p := &sniffer.Proxy{
		Target: *target,
		Observe: func(f sniffer.Frame) {
			if *verbose && len(f.Payload) > 0 {
				n := len(f.Payload)
				if n > 48 {
					n = 48
				}
				log.Printf("%s #%-3d %s | % x", f.Direction, f.Index, f.Note, f.Payload[:n])
			} else {
				log.Printf("%s #%-3d %s", f.Direction, f.Index, f.Note)
			}
		},
	}
	log.Printf("rfc-sniffer: %s -> %s (Ctrl-C to stop)", *listen, *target)
	if err := p.Serve(ctx, *listen); err != nil && ctx.Err() == nil {
		log.Fatalf("rfc-sniffer: %v", err)
	}
}
