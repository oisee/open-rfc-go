// SPDX-License-Identifier: Apache-2.0
//
// rfc-viewer un-tangles an rfc-sniffer capture (JSONL from `rfc-sniffer -dump`)
// using this project's own protocol decoders: the gateway handshake record, the
// APPC record functions, and — where a message reassembles — the CPIC/CUT
// request and response (function name, parameters, tables, outcome/exception).
//
// Output is a human-readable transcript by default; -json emits an annotated
// JSON document; -html writes a self-contained visual inspector next to the
// capture (same name, .html); -serve <addr> serves that inspector over HTTP,
// re-decoding on each refresh (so a growing -dump file updates live). Scalar/
// table VALUES are redacted unless -values (a capture may contain credentials).
package main

import (
	"bufio"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/oisee/open-rfc-go/internal/appc"
	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

type frame struct {
	Dir   string `json:"dir"`
	Index int    `json:"index"`
	Note  string `json:"note"`
	Len   int    `json:"len"`
	Hex   string `json:"hex"`
}

// Annotation is one decoded frame in the output transcript.
type Annotation struct {
	Dir     string       `json:"dir"`
	Index   int          `json:"index"`
	Len     int          `json:"len"`
	Layer   string       `json:"layer"` // "gateway" | "appc" | "raw"
	Gateway *GatewayView `json:"gateway,omitempty"`
	APPC    string       `json:"appc,omitempty"` // APPC function name
	CUT     *CUTView     `json:"cut,omitempty"`  // decoded CPIC/CUT message
	NoteRaw string       `json:"note,omitempty"` // sniffer classification / errors
}

// GatewayView is the decoded gateway normal-client record.
type GatewayView struct {
	Version     int    `json:"version"`
	RequestType int    `json:"requestType"`
	ClientAddr  string `json:"clientAddr"`
	Service     string `json:"service"`
}

// CUTView is a decoded CPIC/CUT request or response.
type CUTView struct {
	Kind         string      `json:"kind"` // "request" | "response" | "cpic" | "error"
	Function     string      `json:"function,omitempty"`
	Outputs      []string    `json:"outputs,omitempty"`
	Imports      []ParamView `json:"imports,omitempty"`
	Exports      []ParamView `json:"exports,omitempty"`
	Tables       []TableView `json:"tables,omitempty"`
	Success      *bool       `json:"success,omitempty"`
	Outcome      string      `json:"outcome,omitempty"`
	ExceptionKey string      `json:"exceptionKey,omitempty"`
	Error        string      `json:"error,omitempty"`
	Bytes        int         `json:"bytes,omitempty"`
}

// ParamView is one scalar parameter (value redacted unless -values).
type ParamView struct {
	Name  string `json:"name"`
	Len   int    `json:"len"`
	Value string `json:"value,omitempty"`
}

// TableView is one table parameter summary.
type TableView struct {
	Name    string `json:"name"`
	Rows    int    `json:"rows"`
	RowSize int    `json:"rowSize"`
}

//go:embed inspector.html
var inspectorHTML string

func main() {
	showValues := flag.Bool("values", false, "include decoded scalar values (may reveal credentials/data)")
	asJSON := flag.Bool("json", false, "emit an annotated JSON transcript to stdout instead of text")
	asHTML := flag.Bool("html", false, "write a self-contained HTML inspector next to the capture (same name, .html)")
	serveAddr := flag.String("serve", "", "serve the HTML inspector over HTTP at this address (e.g. :8080); refresh reloads the capture")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: rfc-viewer [-json | -html | -serve :8080] [-values] <capture.jsonl>")
		os.Exit(2)
	}
	path := flag.Arg(0)

	if *serveAddr != "" {
		if err := serve(*serveAddr, path, *showValues); err != nil {
			fmt.Fprintln(os.Stderr, "rfc-viewer:", err)
			os.Exit(1)
		}
		return
	}

	out, err := decode(path, *showValues)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rfc-viewer:", err)
		os.Exit(1)
	}

	switch {
	case *asHTML:
		outPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".html"
		if err := os.WriteFile(outPath, []byte(renderHTML(out, false)), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "rfc-viewer:", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "rfc-viewer: wrote "+outPath)
	case *asJSON:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"frames": out})
	default:
		renderText(out)
	}
}

// decode reads a JSONL capture and returns the annotated frames.
func decode(path string, showValues bool) ([]Annotation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoders := map[string]*appc.ConversationDecoder{}
	seenGateway := map[string]bool{}
	var out []Annotation

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var fr frame
		if json.Unmarshal(sc.Bytes(), &fr) != nil {
			continue
		}
		payload, err := hex.DecodeString(fr.Hex)
		if err != nil {
			continue
		}
		// A new connection resets per-direction state (indices restart at 0).
		if fr.Index == 0 {
			seenGateway[fr.Dir] = false
			delete(decoders, fr.Dir)
		}
		a := Annotation{Dir: fr.Dir, Index: fr.Index, Len: fr.Len}

		if !seenGateway[fr.Dir] && fr.Len == 64 {
			seenGateway[fr.Dir] = true
			a.Layer = "gateway"
			a.Gateway = describeGateway(payload)
			out = append(out, a)
			continue
		}
		info, err := appc.InspectPayload(payload)
		if err != nil {
			a.Layer = "raw"
			a.NoteRaw = err.Error()
			out = append(out, a)
			continue
		}
		a.Layer = "appc"
		a.APPC = info.FunctionName
		dec := decoders[fr.Dir]
		if dec == nil {
			dec, _ = appc.NewConversationDecoder(appc.ConversationDecoderOptions{})
			decoders[fr.Dir] = dec
		}
		messages, perr := dec.Push(payload)
		if perr == nil {
			for _, m := range messages {
				a.CUT = describeMessage(m.Data, showValues)
			}
		}
		out = append(out, a)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// renderHTML embeds the decoded frames into the self-contained inspector page.
// When fetch is true the page loads its data from frames.json (server mode);
// otherwise the frames are baked into the file.
func renderHTML(out []Annotation, fetch bool) string {
	var script string
	if fetch {
		script = `<script>fetch('frames.json').then(r=>r.json()).then(load).catch(e=>alert("load failed: "+e));</script>`
	} else {
		b, _ := json.Marshal(map[string]any{"frames": out})
		script = `<script>load(` + string(b) + `);</script>`
	}
	if i := strings.LastIndex(inspectorHTML, "</body>"); i >= 0 {
		return inspectorHTML[:i] + script + "\n" + inspectorHTML[i:]
	}
	return inspectorHTML + "\n" + script
}

// serve runs the HTML inspector over HTTP. The capture is decoded on each request
// to frames.json, so refreshing shows new frames as a live -dump file grows.
func serve(addr, path string, showValues bool) error {
	page := renderHTML(nil, true)
	mux := http.NewServeMux()
	mux.HandleFunc("/frames.json", func(w http.ResponseWriter, r *http.Request) {
		out, err := decode(path, showValues)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"frames": out})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, page)
	})
	fmt.Fprintf(os.Stderr, "rfc-viewer: serving %s at http://localhost%s  (refresh to reload)\n", path, addr)
	return http.ListenAndServe(addr, mux)
}

func renderText(out []Annotation) {
	for _, a := range out {
		prefix := fmt.Sprintf("%-4s #%-3d %5dB", a.Dir, a.Index, a.Len)
		switch a.Layer {
		case "gateway":
			g := a.Gateway
			fmt.Printf("%s  gateway: version=%d requestType=%d clientAddr=%s service=%q\n", prefix, g.Version, g.RequestType, g.ClientAddr, g.Service)
		case "raw":
			fmt.Printf("%s  [%s]\n", prefix, a.NoteRaw)
		default:
			fmt.Printf("%s  APPC %s\n", prefix, a.APPC)
		}
		if a.CUT != nil {
			renderCUT(a.CUT)
		}
	}
}

func renderCUT(c *CUTView) {
	switch c.Kind {
	case "request":
		fmt.Printf("        └─ CALL FUNCTION %q  outputs=%v\n", c.Function, c.Outputs)
		for _, p := range c.Imports {
			fmt.Printf("             import  %-20s %s\n", p.Name, paramText(p))
		}
		for _, t := range c.Tables {
			fmt.Printf("             table   %-20s %d row(s) × %dB\n", t.Name, t.Rows, t.RowSize)
		}
	case "response":
		if c.Success != nil && *c.Success {
			fmt.Printf("        └─ response: SUCCESS\n")
			for _, p := range c.Exports {
				fmt.Printf("             export  %-20s %s\n", p.Name, paramText(p))
			}
			for _, t := range c.Tables {
				fmt.Printf("             table   %-20s %d row(s) × %dB\n", t.Name, t.Rows, t.RowSize)
			}
		} else {
			fmt.Printf("        └─ response: EXCEPTION %s key=%q\n", c.Outcome, c.ExceptionKey)
		}
	case "error":
		fmt.Printf("        └─ %s (%v)\n", firstNonEmpty(c.Function, "CPIC message"), c.Error)
	default:
		fmt.Printf("        └─ CPIC packet, %dB [logon/other — redacted]\n", c.Bytes)
	}
}

func paramText(p ParamView) string {
	if p.Value != "" {
		return fmt.Sprintf("%dB %q", p.Len, p.Value)
	}
	return fmt.Sprintf("<%dB>", p.Len)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func describeGateway(p []byte) *GatewayView {
	if len(p) < 12 {
		return &GatewayView{}
	}
	return &GatewayView{
		Version:     int(p[0]),
		RequestType: int(p[1]),
		ClientAddr:  fmt.Sprintf("%d.%d.%d.%d", p[2], p[3], p[4], p[5]),
		Service:     trimField(p[10:]),
	}
}

func trimField(b []byte) string {
	s := string(b)
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, " ")
}

func describeMessage(data []byte, showValues bool) *CUTView {
	switch {
	case hasPrefix(data, []byte{0x05, 0x02, 0x00, 0x00}):
		req, err := rfcserver.DecodeCutFunctionRequest(data)
		if err != nil {
			return &CUTView{Kind: "error", Function: "CUT request", Error: err.Error()}
		}
		c := &CUTView{Kind: "request", Function: req.FunctionName, Outputs: req.RequestedOutputs}
		for _, imp := range req.Imports {
			c.Imports = append(c.Imports, paramView(imp.Name, imp.Value, showValues))
		}
		for _, t := range req.Tables {
			c.Tables = append(c.Tables, TableView{Name: t.Name, Rows: len(t.Rows), RowSize: t.RowByteLength})
		}
		return c
	case hasPrefix(data, []byte{0x05, 0x00, 0x00, 0x00}):
		decoded, err := cpic.DecodeFunctionResultFields(data)
		if err != nil {
			return &CUTView{Kind: "error", Function: "CUT response", Error: err.Error()}
		}
		success := decoded.Success
		c := &CUTView{Kind: "response", Success: &success, Outcome: string(decoded.Envelope.Outcome), ExceptionKey: decoded.Envelope.Facts.ExceptionKey}
		if success {
			if classic, err := classicrfc.DecodeResult(decoded.Fields); err == nil {
				for _, s := range classic.Scalars {
					c.Exports = append(c.Exports, paramView(s.Name, s.Value, showValues))
				}
				for _, t := range classic.Tables {
					c.Tables = append(c.Tables, TableView{Name: t.Name, Rows: len(t.Rows), RowSize: t.RowByteLength})
				}
			} else {
				c.Error = err.Error()
			}
		}
		return c
	default:
		return &CUTView{Kind: "cpic", Bytes: len(data)}
	}
}

func paramView(name string, v []byte, show bool) ParamView {
	p := ParamView{Name: name, Len: len(v)}
	if show {
		if s, err := classicrfc.DecodeAbapChar(v); err == nil {
			p.Value = s
		} else {
			p.Value = hex.EncodeToString(v)
		}
	}
	return p
}

func hasPrefix(b, p []byte) bool { return len(b) >= len(p) && string(b[:len(p)]) == string(p) }
