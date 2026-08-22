// SPDX-License-Identifier: Apache-2.0
//
// rfc-lab is the live server-side lab: it runs two things you use together —
//
//	type 3  ports 3200/3300  a transparent sniffer that forwards to the real SAP
//	                         system and captures the wire (JSONL via -dump)
//	type 3  port  3313       the "conscious" server that GENERATES classic RFC
//	                         responses from Go handlers (the polyglot-bridge base)
//
// Point an SM59 type-3 destination's target host at this box: system number 00
// reaches the sniffer (same instance as the real target so the port matches),
// system number 13 reaches the conscious server (set Serializer = Classic).
// Decode captures offline with cmd/rfc-viewer.
//
// A capture may contain credentials (logon scramble, tickets) and data. Treat the
// dump as sensitive: do not commit or share it.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/oisee/open-rfc-go/internal/bridge"
	"github.com/oisee/open-rfc-go/internal/rfcserver"
	"github.com/oisee/open-rfc-go/internal/sniffer"
)

func main() {
	targetHost := flag.String("target-host", "", "real SAP host the sniffer forwards to (required)")
	dump := flag.String("dump", "cap-lab.jsonl", "capture file for the sniffer (tagged JSONL)")
	dumpMax := flag.Int("dump-max", 0, "cap hex bytes per frame (0 = whole payload)")
	flag.Parse()

	// No default: a hostname baked in here would be somebody's real system, and
	// silently forwarding a capture at the wrong host is worse than refusing.
	if *targetHost == "" {
		fmt.Fprintln(os.Stderr, "orfc-lab: -target-host is required (the SAP host the sniffer forwards to)")
		os.Exit(2)
	}

	fmt.Fprintln(os.Stderr, "rfc-lab: captures can contain credentials — do not commit or share the dump file.")

	f, err := os.Create(*dump)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rfc-lab:", err)
		os.Exit(1)
	}
	defer f.Close()
	observe := sniffer.JSONLRecorder(f, *dumpMax)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	type endpoint struct {
		listen string
		role   string
		run    func() error
	}
	var eps []endpoint

	// Sniffer: transparent proxy on the dispatcher + gateway ports (real backend).
	for _, pp := range []struct{ port, label string }{{"3200", "disp"}, {"3300", "gw"}} {
		pp := pp
		p := &sniffer.Proxy{
			Target:  net.JoinHostPort(*targetHost, pp.port),
			Label:   pp.label,
			Observe: func(fr sniffer.Frame) { observe(fr) },
		}
		listen := "0.0.0.0:" + pp.port
		eps = append(eps, endpoint{
			listen: listen,
			role:   fmt.Sprintf("sniff %-4s -> %s", pp.label, p.Target),
			run:    func() error { return p.Serve(ctx, listen) },
		})
	}

	// Conscious server: WE generate classic responses from Go handlers.
	eps = append(eps, endpoint{
		listen: "0.0.0.0:3313",
		role:   "conscious server (sys 13) — generates classic responses via handlers",
		run:    func() error { return serveConsciousLoop(ctx, "0.0.0.0:3313") },
	})

	fmt.Println("rfc-lab: point each SM59 type-3 destination's target host at this box:")
	for _, e := range eps {
		fmt.Printf("  %-16s  %s\n", e.listen, e.role)
	}

	var wg sync.WaitGroup
	for _, e := range eps {
		wg.Add(1)
		go func(e endpoint) {
			defer wg.Done()
			if err := e.run(); err != nil && ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "rfc-lab: %s: %v\n", e.listen, err)
			}
		}(e)
	}
	wg.Wait()
}

func labBridge() *rfcserver.Dispatcher {
	b := bridge.New()
	// STFC_CONNECTION: echo REQUTEXT back and stamp RESPTEXT.
	b.Register(bridge.Func{
		Name: "STFC_CONNECTION",
		Params: []bridge.Param{
			{Name: "REQUTEXT", Dir: bridge.Import, Kind: bridge.Char, Length: 255},
			{Name: "ECHOTEXT", Dir: bridge.Export, Kind: bridge.Char, Length: 255},
			{Name: "RESPTEXT", Dir: bridge.Export, Kind: bridge.Char, Length: 255},
		},
		Call: func(ctx context.Context, in bridge.Values) (bridge.Values, error) {
			echo, _ := in["REQUTEXT"].(string)
			return bridge.Values{"ECHOTEXT": echo, "RESPTEXT": "open-rfc-go polyglot bridge (Go)"}, nil
		},
	})
	// Z_DOUBLE: an ordinary Go function exposed as an RFC FM.
	b.Register(bridge.Func{
		Name: "Z_DOUBLE",
		Params: []bridge.Param{
			{Name: "N", Dir: bridge.Import, Kind: bridge.Int},
			{Name: "RESULT", Dir: bridge.Export, Kind: bridge.Int},
		},
		Call: func(ctx context.Context, in bridge.Values) (bridge.Values, error) {
			n, _ := in["N"].(int32)
			return bridge.Values{"RESULT": n * 2}, nil
		},
	})
	// Z_GREET: another Go function as an FM.
	b.Register(bridge.Func{
		Name: "Z_GREET",
		Params: []bridge.Param{
			{Name: "NAME", Dir: bridge.Import, Kind: bridge.Char, Length: 30},
			{Name: "GREETING", Dir: bridge.Export, Kind: bridge.Char, Length: 90},
		},
		Call: func(ctx context.Context, in bridge.Values) (bridge.Values, error) {
			name, _ := in["NAME"].(string)
			return bridge.Values{"GREETING": "Hello, " + name + " — from Go over RFC"}, nil
		},
	})
	return b.Dispatcher()
}

func serveConsciousLoop(ctx context.Context, listen string) error {
	d := labBridge()
	df, _ := os.Create("cap-gen.jsonl")
	var dmu sync.Mutex
	denc := json.NewEncoder(df)
	dump := func(dir string, frame []byte) {
		dmu.Lock()
		defer dmu.Unlock()
		_ = denc.Encode(map[string]any{"dir": dir, "len": len(frame), "hex": hex.EncodeToString(frame)})
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", listen)
	if err != nil {
		return err
	}
	go func() { <-ctx.Done(); _ = ln.Close() }()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		fmt.Printf("rfc-lab[gen]: client %s\n", conn.RemoteAddr())
		go rfcserver.ServeConscious(conn, d, func(s string) { fmt.Println("  [gen] " + s) }, dump)
	}
}
