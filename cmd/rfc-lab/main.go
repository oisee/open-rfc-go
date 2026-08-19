// SPDX-License-Identifier: Apache-2.0
//
// rfc-lab is a multi-protocol RFC endpoint for exercising SM59 destinations of
// several connection types against one host. Point each destination's target at
// this box; the connection type decides what answers:
//
//	type 3  (RFC to ABAP)         ports 3200/3300  transparent sniffer -> real target
//	type H/G(HTTP)                port  8000       our HTTP server answers 200 (green)
//	type W  (WebSocket RFC)       port  44300      our WebSocket server completes the upgrade
//	type T  (external program)    port  3310       our rfc-server replays a capture (optional)
//
// Only type 3 has a real backend to sniff (the live ABAP system). For the other
// types there is nothing behind us, so we answer ourselves — the point is to see
// exactly what each SM59 connection type sends and to make its Test Connection
// reach us. Every request is logged.
//
// A capture may contain credentials. Run only against your own SAP system.
package main

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/oisee/open-rfc-go/internal/bridge"
	"github.com/oisee/open-rfc-go/internal/rfcserver"
	"github.com/oisee/open-rfc-go/internal/sniffer"
)

func main() {
	targetHost := flag.String("target-host", "192.168.8.103", "real SAP host the type-3 sniffer forwards to")
	dump := flag.String("dump", "cap-lab.jsonl", "capture file for the type-3 sniffer (tagged JSONL)")
	dumpMax := flag.Int("dump-max", 0, "cap hex bytes per frame (0 = whole payload)")
	httpPort := flag.String("http-port", "8000", "port for our HTTP responder (SM59 type H/G)")
	wsPort := flag.String("ws-port", "44300", "port for our WebSocket responder (SM59 type W)")
	replay := flag.String("replay", "", "capture to replay on the type-T port 3310 (optional)")
	replayConn := flag.Int("replay-conn", 1, "1-based connection in -replay to serve")
	replayLabel := flag.String("replay-label", "gw", "only replay frames captured under this label")
	oracle := flag.String("oracle", "", "capture whose handshake seeds our generating type-3 server (port 3310)")
	oracleConn := flag.Int("oracle-conn", 1, "1-based connection in -oracle to take the handshake from")
	program := flag.String("program", "", "capture to build the content-addressed responder from (port 3312)")
	flag.Parse()

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
		typ    string
		listen string
		role   string
		run    func() error
	}
	var eps []endpoint

	// type 3: transparent sniffer on dispatcher + gateway ports (real backend).
	for _, pp := range []struct{ port, label string }{{"3200", "disp"}, {"3300", "gw"}} {
		pp := pp
		p := &sniffer.Proxy{
			Target:  net.JoinHostPort(*targetHost, pp.port),
			Label:   pp.label,
			Observe: func(fr sniffer.Frame) { observe(fr) },
		}
		listen := "0.0.0.0:" + pp.port
		eps = append(eps, endpoint{
			typ:    "3",
			listen: listen,
			role:   fmt.Sprintf("sniff %-4s -> %s", pp.label, p.Target),
			run:    func() error { return p.Serve(ctx, listen) },
		})
	}

	// type H/G: our own HTTP server — answers 200 so the test goes green.
	eps = append(eps, endpoint{
		typ:    "H/G",
		listen: "0.0.0.0:" + *httpPort,
		role:   "our HTTP responder (200 OK to any path)",
		run:    func() error { return serveHTTP(ctx, "0.0.0.0:"+*httpPort) },
	})

	// type W: our own WebSocket server — completes the RFC6455 upgrade.
	eps = append(eps, endpoint{
		typ:    "W",
		listen: "0.0.0.0:" + *wsPort,
		role:   "our WebSocket responder (101 Switching Protocols)",
		run:    func() error { return serveWS(ctx, "0.0.0.0:"+*wsPort) },
	})

	// type T: our rfc-server replaying a capture (only if one was given).
	if *replay != "" {
		script, err := rfcserver.LoadConnection(*replay, *replayConn, *replayLabel)
		if err != nil {
			fmt.Fprintln(os.Stderr, "rfc-lab: -replay:", err)
			os.Exit(1)
		}
		eps = append(eps, endpoint{
			typ:    "3(ours)",
			listen: "0.0.0.0:3310",
			role:   fmt.Sprintf("serve replay (conn %d, label %q, %d frames)", *replayConn, *replayLabel, len(script)),
			run:    func() error { return serveReplayLoop(ctx, "0.0.0.0:3310", script) },
		})
	}

	// type 3 (ours): our generating APPC server — WE answer, not the real system.
	if *oracle != "" {
		steps, err := rfcserver.LoadConnection(*oracle, *oracleConn, "")
		if err != nil {
			fmt.Fprintln(os.Stderr, "rfc-lab: -oracle:", err)
			os.Exit(1)
		}
		var server [][]byte
		for _, st := range steps {
			if st.Dir == "S->C" {
				server = append(server, st.Payload)
			}
		}
		if len(server) < 2 {
			fmt.Fprintln(os.Stderr, "rfc-lab: -oracle connection has no logon-accept frame")
			os.Exit(1)
		}
		logonAccept := server[1]
		respCUT := rfcserver.KnownGoodPingResponse
		eps = append(eps, endpoint{
			typ:    "3(ours)",
			listen: "0.0.0.0:3310",
			role:   fmt.Sprintf("our generating type-3 server (logon-accept %dB, ping-resp %dB)", len(logonAccept), len(respCUT)),
			run:    func() error { return serveGenerateLoop(ctx, "0.0.0.0:3310", logonAccept, respCUT) },
		})
	}

	// type 3 (ours, generated): a state machine that GENERATES replies (no
	// capture needed). SM59 target host = this box, system number 11.
	eps = append(eps, endpoint{
		typ:    "3(smart)",
		listen: "0.0.0.0:3311",
		role:   "our state-machine type-3 server (generates logon-accept + RFC_PING)",
		run:    func() error { return serveSmartLoop(ctx, "0.0.0.0:3311") },
	})

	// type 3 (content-addressed): match each request to a recorded reply script.
	if *program != "" {
		tmpl, err := rfcserver.LoadTemplates(strings.Split(*program, ","), "gw")
		if err != nil {
			fmt.Fprintln(os.Stderr, "rfc-lab: -program:", err)
			os.Exit(1)
		}
		eps = append(eps, endpoint{
			typ:    "3(prog)",
			listen: "0.0.0.0:3312",
			role:   "content-addressed responder (from " + *program + ")",
			run:    func() error { return serveContentLoop(ctx, "0.0.0.0:3312", tmpl) },
		})
	}

	// type 3 (conscious): GENERATE responses from Go handlers (needs SM59
	// Serializer = Classic serializer). The base for the polyglot bridge.
	eps = append(eps, endpoint{
		typ:    "3(gen)",
		listen: "0.0.0.0:3313",
		role:   "conscious server — generates classic responses via handlers",
		run:    func() error { return serveConsciousLoop(ctx, "0.0.0.0:3313") },
	})

	fmt.Println("rfc-lab: endpoints — point each SM59 destination's target at this box:")
	for _, e := range eps {
		fmt.Printf("  type %-4s  %-16s  %s\n", e.typ, e.listen, e.role)
	}
	if *replay == "" {
		fmt.Println("  (type T replay disabled — pass -replay <capture.jsonl> to enable)")
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
	fmt.Println("rfc-lab: stopped")
}

// serveHTTP answers every request 200 and logs it, so an SM59 type H/G Test
// Connection reaches us and goes green.
func serveHTTP(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("rfc-lab[HTTP]: %s %s from %s  host=%q ua=%q\n",
			r.Method, r.URL.Path, r.RemoteAddr, r.Host, r.UserAgent())
		w.Header().Set("Server", "open-rfc-go/rfc-lab")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "open-rfc-go rfc-lab: HTTP endpoint alive")
	})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { <-ctx.Done(); _ = srv.Close() }()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

const wsMagic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// serveWS completes the RFC6455 handshake (101) for an SM59 type W destination,
// then logs the WebSocket bytes it receives. It proves the transport connects;
// the RFC-over-WebSocket payload framing is a later track.
func serveWS(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Sec-WebSocket-Key")
		fmt.Printf("rfc-lab[WS]: %s %s from %s  upgrade=%q key=%q\n",
			r.Method, r.URL.Path, r.RemoteAddr, r.Header.Get("Upgrade"), key)
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || key == "" {
			http.Error(w, "expected a WebSocket upgrade", http.StatusBadRequest)
			return
		}
		h := sha1.Sum([]byte(key + wsMagic))
		accept := base64.StdEncoding.EncodeToString(h[:])
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijack", http.StatusInternalServerError)
			return
		}
		conn, brw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		resp := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + accept + "\r\n"
		if proto := r.Header.Get("Sec-WebSocket-Protocol"); proto != "" {
			// Echo the first requested subprotocol (SAP sends "sap.rfc" here).
			first := strings.TrimSpace(strings.Split(proto, ",")[0])
			resp += "Sec-WebSocket-Protocol: " + first + "\r\n"
		}
		resp += "\r\n"
		_, _ = brw.WriteString(resp)
		_ = brw.Flush()
		fmt.Println("rfc-lab[WS]: upgrade complete (101) — reading frames")
		logWSBytes(conn, brw.Reader)
	})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { <-ctx.Done(); _ = srv.Close() }()
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func logWSBytes(conn net.Conn, r *bufio.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			fmt.Printf("rfc-lab[WS]: %d bytes  head=%x\n", n, buf[:min(16, n)])
		}
		if err != nil {
			return
		}
	}
}

func serveReplayLoop(ctx context.Context, listen string, script []rfcserver.ReplayStep) error {
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
		fmt.Printf("rfc-lab[type T]: client %s\n", conn.RemoteAddr())
		go rfcserver.ServeReplay(conn, script, func(s string) { fmt.Println("  [type T] " + s) })
	}
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

func serveContentLoop(ctx context.Context, listen string, tmpl *rfcserver.Templates) error {
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
		fmt.Printf("rfc-lab[prog]: client %s\n", conn.RemoteAddr())
		go rfcserver.ServeContentAddressed(conn, tmpl, func(s string) { fmt.Println("  [prog] " + s) })
	}
}

func serveSmartLoop(ctx context.Context, listen string) error {
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
		fmt.Printf("rfc-lab[smart]: client %s\n", conn.RemoteAddr())
		go rfcserver.ServeSmart(conn, func(s string) { fmt.Println("  [smart] " + s) })
	}
}

func serveGenerateLoop(ctx context.Context, listen string, logonAccept, respCUT []byte) error {
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
		fmt.Printf("rfc-lab[type3-ours]: client %s\n", conn.RemoteAddr())
		go rfcserver.ServeGenerate(conn, logonAccept, respCUT, func(s string) { fmt.Println("  [type3-ours] " + s) })
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
