// SPDX-License-Identifier: Apache-2.0
//
// rfc-relaysniff is a one-port transparent RFC sniffer: it listens on -listen,
// forwards every connection to -target, and records both directions as tagged
// JSONL. It exists to sit between a registered server (rfcexec) and the SAP
// gateway on the same host — where the gateway already owns 3300, so the sniffer
// must take a different port and forward to the real gateway on 3300. The
// registered server registers through this port; the gateway then relays the
// client's logon over that connection, so the logon (and any ticket it carries)
// passes through here.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/oisee/open-rfc-go/internal/sniffer"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:3355", "address to listen on")
	target := flag.String("target", "127.0.0.1:3300", "real gateway to forward to")
	dump := flag.String("dump", "cap-relay.jsonl", "capture file (tagged JSONL)")
	dumpMax := flag.Int("dump-max", 0, "cap hex bytes per frame (0 = whole payload)")
	flag.Parse()

	fmt.Fprintln(os.Stderr, "rfc-relaysniff: captures can contain credentials — do not share the dump.")
	f, err := os.Create(*dump)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	observe := sniffer.JSONLRecorder(f, *dumpMax)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	p := &sniffer.Proxy{
		Target:  *target,
		Label:   "relay",
		Observe: func(fr sniffer.Frame) { observe(fr) },
	}
	fmt.Printf("rfc-relaysniff: %s -> %s, dump -> %s\n", *listen, *target, *dump)
	if err := p.Serve(ctx, *listen); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
