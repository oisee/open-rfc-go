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

	"github.com/oisee/open-rfc-go/rfc"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "rfc:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `rfc — call SAP classic RFC function modules (SDK-free)

Usage:
  rfc info                     show system info (RFC_SYSTEM_INFO)
  rfc describe <FM>            print the FM interface as an MCP-tool JSON Schema
  rfc call <FM> [json]         call the FM; params as inline JSON, --file, or --stdin
  rfc ping                     connection test (RFC_PING)

Flags:
  --file <path>                read call params (JSON object) from a file
  --stdin                      read call params (JSON object) from stdin

Connection via env: SAP_ASHOST, SAP_SYSNR (00), SAP_CLIENT (001), SAP_USER,
SAP_PASSWORD/SAP_PASSWD, SAP_LANG (EN).
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
			return fmt.Errorf("usage: rfc describe <FM>")
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
			return fmt.Errorf("usage: rfc call <FM> [json] | --file f | --stdin")
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
	return normalizeObject(obj), nil
}

// normalizeObject/normalizeValue convert decoded JSON into the native Go shapes
// the client expects: json.Number -> int64/float64, arrays of objects ->
// []map[string]any (tables), objects -> map[string]any (structures).
func normalizeObject(m map[string]any) map[string]any {
	for k, v := range m {
		m[k] = normalizeValue(v)
	}
	return m
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	case map[string]any:
		return normalizeObject(t)
	case []any:
		norm := make([]any, len(t))
		allObj := len(t) > 0
		for i, x := range t {
			norm[i] = normalizeValue(x)
			if _, ok := norm[i].(map[string]any); !ok {
				allObj = false
			}
		}
		if allObj {
			rows := make([]map[string]any, len(norm))
			for i := range norm {
				rows[i] = norm[i].(map[string]any)
			}
			return rows
		}
		return norm
	}
	return v
}

func withClient(ctx context.Context, fn func(*rfc.Client) error) error {
	host := os.Getenv("SAP_ASHOST")
	if host == "" {
		return fmt.Errorf("SAP_ASHOST is required")
	}
	sysnr := envOr("SAP_SYSNR", "00")
	n, err := strconv.Atoi(sysnr)
	if err != nil {
		return fmt.Errorf("SAP_SYSNR must be numeric: %q", sysnr)
	}
	pass := os.Getenv("SAP_PASSWORD")
	if pass == "" {
		pass = os.Getenv("SAP_PASSWD")
	}
	lang := envOr("SAP_LANG", "EN")
	c, err := rfc.Open(ctx, rfc.Destination{
		Host:     host,
		Port:     3300 + n,
		Service:  fmt.Sprintf("sapdp%02d", n),
		Client:   envOr("SAP_CLIENT", "001"),
		User:     os.Getenv("SAP_USER"),
		Password: pass,
		Language: string([]rune(strings.ToUpper(lang))[0:1]),
	})
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

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
