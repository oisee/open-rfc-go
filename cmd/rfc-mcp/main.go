// SPDX-License-Identifier: Apache-2.0
//
// rfc-mcp is a Model Context Protocol server over stdio that exposes the SDK-free
// RFC client as a small set of generic tools: rfc_info, rfc_ping, rfc_describe,
// rfc_search, rfc_read_table, and rfc_call. `rfc_describe` returns an FM interface
// as an MCP-tool JSON Schema; `rfc_call` runs any function module with native
// JSON arguments (the client coerces them per the interface). Connection is taken
// from the environment (SAP_ASHOST/SYSNR/CLIENT/USER/PASSWORD/LANG); pass
// --read-only to disable rfc_call.
//
// Transport: newline-delimited JSON-RPC 2.0 on stdin/stdout, no dependencies.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/oisee/open-rfc-go/rfc"
)

const protocolVersion = "2024-11-05"

var readOnly bool

func main() {
	for _, a := range os.Args[1:] {
		if a == "--read-only" {
			readOnly = true
		}
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
	return tools
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
	return readTable(ctx, c, "TFDIR", where, []string{"FUNCNAME", "PNAME"}, top)
}

func readTableArgs(ctx context.Context, c *rfc.Client, a map[string]any) (any, error) {
	var fields []string
	if fs, ok := a["fields"].([]any); ok {
		for _, f := range fs {
			fields = append(fields, strings.ToUpper(strVal(f)))
		}
	}
	return readTable(ctx, c, strings.ToUpper(strVal(a["table"])), strVal(a["where"]), fields, intVal(a["top"], 0))
}

func readTable(ctx context.Context, c *rfc.Client, table, where string, fields []string, top int) ([]map[string]string, error) {
	in := rfc.Params{"QUERY_TABLE": table, "DELIMITER": "|"}
	if top > 0 {
		in["ROWCOUNT"] = int64(top)
	}
	if where != "" {
		in["OPTIONS"] = []map[string]any{{"TEXT": where}}
	}
	if len(fields) > 0 {
		fs := make([]map[string]any, 0, len(fields))
		for _, f := range fields {
			fs = append(fs, map[string]any{"FIELDNAME": f})
		}
		in["FIELDS"] = fs
	}
	r, err := c.Call(ctx, "RFC_READ_TABLE", in)
	if err != nil {
		return nil, err
	}
	var cols []string
	for _, fr := range r.Table("FIELDS") {
		cols = append(cols, strings.TrimSpace(fmt.Sprint(fr["FIELDNAME"])))
	}
	var out []map[string]string
	for _, dr := range r.Table("DATA") {
		parts := strings.Split(fmt.Sprint(dr["WA"]), "|")
		row := map[string]string{}
		for i, col := range cols {
			if i < len(parts) {
				row[col] = strings.TrimRight(parts[i], " ")
			}
		}
		out = append(out, row)
	}
	return out, nil
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
		host := os.Getenv("SAP_ASHOST")
		if host == "" {
			clientErr = fmt.Errorf("SAP_ASHOST is required")
			return
		}
		n, err := strconv.Atoi(envOr("SAP_SYSNR", "00"))
		if err != nil {
			clientErr = fmt.Errorf("SAP_SYSNR must be numeric")
			return
		}
		pass := os.Getenv("SAP_PASSWORD")
		if pass == "" {
			pass = os.Getenv("SAP_PASSWD")
		}
		lang := envOr("SAP_LANG", "EN")
		sharedC, clientErr = rfc.Open(ctx, rfc.Destination{
			Host: host, Port: 3300 + n, Service: fmt.Sprintf("sapdp%02d", n),
			Client: envOr("SAP_CLIENT", "001"), User: os.Getenv("SAP_USER"),
			Password: pass, Language: string([]rune(strings.ToUpper(lang))[0:1]),
		})
	})
	return sharedC, clientErr
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
