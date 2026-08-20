// SPDX-License-Identifier: Apache-2.0
//
// rfc-mcp is a Model Context Protocol server over stdio that exposes the SDK-free
// RFC client as a small set of generic tools: rfc_info, rfc_ping, rfc_describe,
// rfc_search, rfc_read_table, and rfc_call. `rfc_describe` returns an FM interface
// as an MCP-tool JSON Schema; `rfc_call` runs any function module with native
// JSON arguments (the client coerces them per the interface).
//
// Configuration comes from three sources (later wins): .rfc.json (named systems
// with expose/hide/readOnly), environment (SAP_ASHOST/SYSNR/CLIENT/USER/PASSWORD/
// LANG), and flags: -s/--system <name>, --expose <masks>, --hide <masks>, --max N,
// --read-only. --expose turns matching RFC-enabled FMs into per-FM MCP tools.
//
// Transport: newline-delimited JSON-RPC 2.0 on stdin/stdout, no dependencies.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/oisee/open-rfc-go/cmd/rfctool"
	"github.com/oisee/open-rfc-go/rfc"
)

const protocolVersion = "2024-11-05"

var (
	readOnly    bool
	exposeMasks []string // green-list FM name masks (* wildcard) -> per-FM MCP tools
	hideMasks   []string // red-list masks, excluded from the green-list
	maxTools    = 200
	mcpSystem   string
)

func runMCP(cliArgs []string) error {
	mcpSystem = systemName // inherit the global -s/--system
	var exposeSet, hideSet, readOnlySet, maxSet bool
	args := cliArgs
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--read-only":
			readOnly, readOnlySet = true, true
		case "--expose":
			if i+1 < len(args) {
				exposeMasks = append(exposeMasks, splitMasks(args[i+1])...)
				exposeSet = true
				i++
			}
		case "--hide":
			if i+1 < len(args) {
				hideMasks = append(hideMasks, splitMasks(args[i+1])...)
				hideSet = true
				i++
			}
		case "--max":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					maxTools, maxSet = n, true
				}
				i++
			}
		case "--system", "-s":
			if i+1 < len(args) {
				mcpSystem = args[i+1]
				i++
			}
		}
	}
	// Config (.rfc.json) provides defaults; command-line flags win.
	opts := rfctool.LoadOptions(mcpSystem)
	if !exposeSet {
		exposeMasks = opts.Expose
	}
	if !hideSet {
		hideMasks = opts.Hide
	}
	if !readOnlySet {
		readOnly = opts.ReadOnly
	}
	if !maxSet && opts.MaxTools > 0 {
		maxTools = opts.MaxTools
	}
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	for in.Scan() {
		line := strings.TrimSpace(in.Text())
		if line == "" {
			continue
		}
		var req rpcReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		resp, isNotification := handle(&req)
		if isNotification {
			continue
		}
		b, _ := json.Marshal(resp)
		out.Write(b)
		out.WriteByte('\n')
		out.Flush()
	}
	return nil
}

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcErr         `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func handle(req *rpcReq) (rpcResp, bool) {
	resp := rpcResp{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "open-rfc-go", "version": "0.1.0"},
		}
	case "notifications/initialized", "notifications/cancelled":
		return resp, true // notification: no reply
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		resp.Result = map[string]any{"tools": toolList()}
	case "tools/call":
		resp.Result, resp.Error = callTool(req.Params)
	default:
		resp.Error = &rpcErr{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp, len(req.ID) == 0
}

func toolList() []map[string]any {
	obj := func(props map[string]any, req ...string) map[string]any {
		s := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			s["required"] = req
		}
		return s
	}
	str := map[string]any{"type": "string"}
	tools := []map[string]any{
		{"name": "rfc_info", "description": "SAP system info (RFC_SYSTEM_INFO): sysid, release, host, unicode.", "inputSchema": obj(map[string]any{})},
		{"name": "rfc_ping", "description": "Connection test (RFC_PING).", "inputSchema": obj(map[string]any{})},
		{"name": "rfc_describe", "description": "Describe an RFC function module's interface as an MCP-tool JSON Schema (input/output).", "inputSchema": obj(map[string]any{"function": str}, "function")},
		{"name": "rfc_search", "description": "Find RFC-enabled function modules by name mask (* wildcard).", "inputSchema": obj(map[string]any{
			"pattern": str, "all": map[string]any{"type": "boolean"}, "top": map[string]any{"type": "integer"}, "group": str}, "pattern")},
		{"name": "rfc_read_table", "description": "Read a database table via RFC_READ_TABLE.", "inputSchema": obj(map[string]any{
			"table": str, "where": str, "fields": map[string]any{"type": "array", "items": str}, "top": map[string]any{"type": "integer"}}, "table")},
	}
	if !readOnly {
		tools = append(tools, map[string]any{
			"name": "rfc_call", "description": "Call any function module with native JSON arguments (coerced per the interface). Use rfc_describe first for the schema.",
			"inputSchema": obj(map[string]any{"function": str, "params": map[string]any{"type": "object"}}, "function"),
		})
	}
	// Auto-discovered per-FM tools (from --expose/--hide masks): each exposed FM
	// becomes a real MCP tool whose inputSchema is the FM's own interface.
	tools = append(tools, resolveExposed(context.Background())...)
	return tools
}

// splitMasks splits a comma-separated list of FM name masks.
func splitMasks(csv string) []string {
	var out []string
	for _, m := range strings.Split(csv, ",") {
		if m = strings.TrimSpace(m); m != "" {
			out = append(out, m)
		}
	}
	return out
}

var (
	exposedOnce  sync.Once
	exposedTools []map[string]any  // per-FM tool definitions for tools/list
	exposedFM    map[string]string // MCP tool name -> function module name
)

// resolveExposed matches the green-list masks against the system's RFC-enabled
// function modules, drops the red-list, and renders each as an MCP tool from its
// interface. Resolved once, then cached. Failures degrade to no per-FM tools.
func resolveExposed(ctx context.Context) []map[string]any {
	exposedOnce.Do(func() {
		exposedFM = map[string]string{}
		if len(exposeMasks) == 0 {
			return
		}
		c, err := client(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "rfc-mcp: expose:", err)
			return
		}
		// One query per mask (RFC_READ_TABLE's WHERE rejects OR/parentheses),
		// deduplicated; precise glob filtering happens client-side below.
		seen := map[string]bool{}
		var candidates []string
		for _, m := range exposeMasks {
			like := strings.ReplaceAll(strings.ToUpper(m), "*", "%")
			rows, err := rfctool.ReadTable(ctx, c, "TFDIR", "FMODE = 'R' AND FUNCNAME LIKE '"+like+"'", []string{"FUNCNAME"}, 0)
			if err != nil {
				fmt.Fprintln(os.Stderr, "rfc-mcp: expose:", err)
				continue
			}
			for _, r := range rows {
				if fm := r["FUNCNAME"]; !seen[fm] {
					seen[fm] = true
					candidates = append(candidates, fm)
				}
			}
		}
		for _, fm := range candidates {
			if !anyGlob(exposeMasks, fm) || anyGlob(hideMasks, fm) {
				continue
			}
			if len(exposedTools) >= maxTools {
				fmt.Fprintf(os.Stderr, "rfc-mcp: expose: capped at %d tools\n", maxTools)
				break
			}
			tool, err := c.DescribeTool(ctx, fm)
			if err != nil {
				continue
			}
			exposedFM[tool.Name] = fm
			exposedTools = append(exposedTools, map[string]any{
				"name": tool.Name, "description": tool.Description, "inputSchema": tool.InputSchema,
			})
		}
	})
	return exposedTools
}

func globMatch(mask, name string) bool {
	var b strings.Builder
	b.WriteString("(?i)^")
	for _, r := range mask {
		if r == '*' {
			b.WriteString(".*")
		} else {
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	return err == nil && re.MatchString(name)
}

func anyGlob(masks []string, name string) bool {
	for _, m := range masks {
		if globMatch(m, name) {
			return true
		}
	}
	return false
}

func callTool(raw json.RawMessage) (any, *rpcErr) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpcErr{Code: -32602, Message: "invalid params"}
	}
	ctx := context.Background()
	c, err := client(ctx)
	if err != nil {
		return toolError(err), nil
	}
	arg := func(k string) string { s, _ := p.Arguments[k].(string); return s }
	// Auto-discovered per-FM tool: the tool name maps to an FM; its arguments are
	// the FM's parameters directly (coerced per the interface by Client.Call).
	resolveExposed(ctx)
	if fm, ok := exposedFM[p.Name]; ok {
		r, err := c.Call(ctx, fm, rfc.Params(p.Arguments))
		if err != nil {
			return toolError(err), nil
		}
		return toolJSON(r), nil
	}
	switch p.Name {
	case "rfc_info":
		r, err := c.Call(ctx, "RFC_SYSTEM_INFO", nil)
		if err != nil {
			return toolError(err), nil
		}
		return toolJSON(r.Get("RFCSI_EXPORT")), nil
	case "rfc_ping":
		if _, err := c.Call(ctx, "RFC_PING", nil); err != nil {
			return toolError(err), nil
		}
		return toolText("ok"), nil
	case "rfc_describe":
		tool, err := c.DescribeTool(ctx, strings.ToUpper(arg("function")))
		if err != nil {
			return toolError(err), nil
		}
		return toolJSON(tool), nil
	case "rfc_search":
		rows, err := search(ctx, c, p.Arguments)
		if err != nil {
			return toolError(err), nil
		}
		return toolJSON(rows), nil
	case "rfc_read_table":
		rows, err := readTableArgs(ctx, c, p.Arguments)
		if err != nil {
			return toolError(err), nil
		}
		return toolJSON(rows), nil
	case "rfc_call":
		if readOnly {
			return toolError(fmt.Errorf("rfc_call is disabled (--read-only)")), nil
		}
		params, _ := p.Arguments["params"].(map[string]any)
		r, err := c.Call(ctx, strings.ToUpper(arg("function")), rfc.Params(params))
		if err != nil {
			return toolError(err), nil
		}
		return toolJSON(r), nil
	}
	return nil, &rpcErr{Code: -32602, Message: "unknown tool: " + p.Name}
}

// --- tool result helpers (MCP content) ---

func toolJSON(v any) map[string]any {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolError(err)
	}
	return toolText(string(b))
}

func toolText(s string) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": s}}}
}

func toolError(err error) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": "error: " + err.Error()}}, "isError": true}
}

// --- RFC helpers ---

func search(ctx context.Context, c *rfc.Client, a map[string]any) (any, error) {
	pattern := strings.ToUpper(strVal(a["pattern"]))
	like := strings.ReplaceAll(pattern, "*", "%")
	if !strings.Contains(like, "%") {
		like = "%" + like + "%"
	}
	where := "FUNCNAME LIKE '" + like + "'"
	if all, _ := a["all"].(bool); !all {
		where += " AND FMODE = 'R'"
	}
	if g := strVal(a["group"]); g != "" {
		where += " AND PNAME LIKE 'SAPL" + strings.ToUpper(strings.ReplaceAll(g, "*", "%")) + "%'"
	}
	top := intVal(a["top"], 100)
	return rfctool.ReadTable(ctx, c, "TFDIR", where, []string{"FUNCNAME", "PNAME"}, top)
}

func readTableArgs(ctx context.Context, c *rfc.Client, a map[string]any) (any, error) {
	var fields []string
	if fs, ok := a["fields"].([]any); ok {
		for _, f := range fs {
			fields = append(fields, strings.ToUpper(strVal(f)))
		}
	}
	return rfctool.ReadTable(ctx, c, strings.ToUpper(strVal(a["table"])), strVal(a["where"]), fields, intVal(a["top"], 0))
}

func strVal(v any) string { s, _ := v.(string); return s }

func intVal(v any, def int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return i
		}
	}
	return def
}

// --- lazy client (opened once from env) ---

var (
	clientOnce sync.Once
	sharedC    *rfc.Client
	clientErr  error
)

func client(ctx context.Context) (*rfc.Client, error) {
	clientOnce.Do(func() {
		sharedC, _, clientErr = rfctool.Open(ctx, mcpSystem)
	})
	return sharedC, clientErr
}
