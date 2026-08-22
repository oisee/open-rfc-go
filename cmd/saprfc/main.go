// SPDX-License-Identifier: Apache-2.0
//
// rfc is a small CLI over the open-rfc-go client: describe any function module
// as an MCP-tool JSON Schema, call any FM with native JSON values, and read
// system info — the human-facing half of the RFC tool surface (the MCP server is
// cmd/rfc-mcp). Connection comes from the environment:
//
//	SAP_ASHOST  gateway host (required)      SAP_SYSNR   instance number (default 00)
//	SAP_CLIENT  client (default 001)         SAP_USER    user
//	SAP_PASSWORD (or SAP_PASSWD) password     SAP_LANG    logon language (default EN)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oisee/open-rfc-go/cmd/rfctool"
	"github.com/oisee/open-rfc-go/rfc"
)

var (
	systemName  string
	callTimeout time.Duration
)

func main() {
	args := stripSystemFlag(os.Args[1:])
	if len(args) < 1 {
		usage()
		os.Exit(2)
	}
	if err := run(args[0], args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "rfc:", err)
		os.Exit(1)
	}
}

// stripSystemFlag pulls a global -s/--system <name> out of the argument list.
func stripSystemFlag(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		if (args[i] == "-s" || args[i] == "--system") && i+1 < len(args) {
			systemName = args[i+1]
			i++
			continue
		}
		if args[i] == "--timeout" && i+1 < len(args) {
			if secs, err := strconv.Atoi(args[i+1]); err == nil {
				callTimeout = time.Duration(secs) * time.Second
			}
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func usage() {
	fmt.Fprint(os.Stderr, `saprfc — call SAP classic RFC function modules (SDK-free)

Usage:
  saprfc info                     show system info (RFC_SYSTEM_INFO)
  saprfc describe <FM>            print the FM interface as an MCP-tool JSON Schema
  saprfc call <FM> [json]         call the FM; params as inline JSON, --file, or --stdin
  saprfc search <pattern>         find RFC-enabled FMs (name mask, * wildcard; --all = any)
  saprfc read-table <table>       read a table (RFC_READ_TABLE) as rows of columns
  saprfc ping                     connection test (RFC_PING)
  saprfc session [-c "a; b"]      pin one connection and run several calls on it,
                                  for protocols that keep state in the ABAP session
  saprfc mcp [--expose m] [--hide m]  run the MCP server (stdio) over this client

Flags:
  --file <path>                read call params (JSON object) from a file
  --stdin                      read call params (JSON object) from stdin
  --where <clause>             read-table WHERE clause
  --fields <a,b,c>             read-table column list
  --top <N>                    read-table / search row limit
  --group <name>               search: restrict to a function group (PNAME mask)

Global:
  -s, --system <name>          pick a system from .rfc.json
      --timeout <seconds>      bound one call (default 30); raise it for calls
                               that block server-side, e.g. a debugger listener

Connection: .rfc.json (named systems) and/or env SAP_ASHOST, SAP_SYSNR (00),
SAP_CLIENT (001), SAP_USER, SAP_PASSWORD/SAP_PASSWD, SAP_LANG (EN) — env wins.
`)
}

func run(cmd string, args []string) error {
	ctx := context.Background()
	switch cmd {
	case "info":
		return withClient(ctx, func(c *rfc.Client) error {
			r, err := c.Call(ctx, "RFC_SYSTEM_INFO", nil)
			if err != nil {
				return err
			}
			return emit(r.Get("RFCSI_EXPORT"))
		})
	case "ping":
		return withClient(ctx, func(c *rfc.Client) error {
			if _, err := c.Call(ctx, "RFC_PING", nil); err != nil {
				return err
			}
			fmt.Println("ok")
			return nil
		})
	case "describe":
		if len(args) < 1 {
			return fmt.Errorf("usage: saprfc describe <FM>")
		}
		return withClient(ctx, func(c *rfc.Client) error {
			tool, err := c.DescribeTool(ctx, strings.ToUpper(args[0]))
			if err != nil {
				return err
			}
			return emit(tool)
		})
	case "call":
		if len(args) < 1 {
			return fmt.Errorf("usage: saprfc call <FM> [json] | --file f | --stdin")
		}
		fn := strings.ToUpper(args[0])
		params, err := readParams(args[1:])
		if err != nil {
			return err
		}
		return withClient(ctx, func(c *rfc.Client) error {
			r, err := c.Call(ctx, fn, params)
			if err != nil {
				return err
			}
			return emit(r)
		})
	case "session":
		return runSession(ctx, args)
	case "mcp":
		return runMCP(args)
	case "search":
		if len(args) < 1 {
			return fmt.Errorf("usage: saprfc search <pattern>")
		}
		return runSearch(ctx, args)
	case "read-table", "read_table", "readtable":
		if len(args) < 1 {
			return fmt.Errorf("usage: saprfc read-table <table> [--where ..] [--fields ..] [--top N]")
		}
		return runReadTable(ctx, args)
	default:
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// readParams gathers call parameters from an inline JSON arg, --file, or --stdin,
// then normalizes JSON types to the native Go types the client's codecs expect.
func readParams(args []string) (rfc.Params, error) {
	var raw []byte
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--file needs a path")
			}
			b, err := os.ReadFile(args[i+1])
			if err != nil {
				return nil, err
			}
			raw, i = b, i+1
		case "--stdin":
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, err
			}
			raw = b
		default:
			raw = []byte(args[i])
		}
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return rfc.Params{}, nil
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var obj map[string]any
	if err := dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("params must be a JSON object: %w", err)
	}
	// Values stay JSON-native (json.Number, []any, nested maps); Client.Call
	// coerces them to the exact types each parameter expects (interface-aware).
	return obj, nil
}

func withClient(ctx context.Context, fn func(*rfc.Client) error) error {
	c, _, err := rfctool.OpenWithTimeout(ctx, systemName, callTimeout)
	if err != nil {
		return err
	}
	defer c.Close(ctx)
	return fn(c)
}

func emit(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// hasFlag reports whether a boolean --name flag is present in args.
func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

// flagValue returns the value after --name in args, and true if present.
func flagValue(args []string, name string) (string, bool) {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			return args[i+1], true
		}
	}
	return "", false
}

// runSearch lists function modules whose name matches a mask (* wildcard),
// via RFC_READ_TABLE over TFDIR.
func runSearch(ctx context.Context, args []string) error {
	pattern := strings.ToUpper(args[0])
	like := strings.ReplaceAll(pattern, "*", "%")
	if !strings.Contains(like, "%") {
		like = "%" + like + "%"
	}
	where := "FUNCNAME LIKE '" + like + "'"
	// Default to remote-enabled modules only; --all lifts it. TFDIR-FMODE has
	// two remote values, not one: 'R' is a remote-enabled module and 'X' a
	// remote-enabled module whose interface is basXML-capable (SAP flags every
	// FM carrying deep/nested parameters that way — SADT_REST_RFC_ENDPOINT
	// among them). Filtering on 'R' alone hides them and makes a perfectly
	// callable FM look local.
	if !hasFlag(args, "--all") {
		where += " AND FMODE IN ( 'R', 'X' )"
	}
	if g, ok := flagValue(args, "--group"); ok {
		where += " AND PNAME LIKE 'SAPL" + strings.ToUpper(strings.ReplaceAll(g, "*", "%")) + "%'"
	}
	top := 100
	if v, ok := flagValue(args, "--top"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			top = n
		}
	}
	return withClient(ctx, func(c *rfc.Client) error {
		rows, err := rfctool.ReadTable(ctx, c, "TFDIR", where, []string{"FUNCNAME", "PNAME"}, top)
		if err != nil {
			return err
		}
		return emit(rows)
	})
}

// runReadTable reads a table's rows through RFC_READ_TABLE and returns each row
// as a column->value object.
func runReadTable(ctx context.Context, args []string) error {
	table := strings.ToUpper(args[0])
	where, _ := flagValue(args, "--where")
	var fields []string
	if f, ok := flagValue(args, "--fields"); ok {
		for _, x := range strings.Split(f, ",") {
			if x = strings.TrimSpace(x); x != "" {
				fields = append(fields, strings.ToUpper(x))
			}
		}
	}
	top := 0
	if v, ok := flagValue(args, "--top"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			top = n
		}
	}
	return withClient(ctx, func(c *rfc.Client) error {
		rows, err := rfctool.ReadTable(ctx, c, table, where, fields, top)
		if err != nil {
			return err
		}
		return emit(rows)
	})
}
