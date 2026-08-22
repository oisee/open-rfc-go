// SPDX-License-Identifier: Apache-2.0

// Package rfctool holds the small shared plumbing for the rfc CLI and the
// rfc-mcp server: connection config (.rfc.json + environment), opening a client,
// and a RFC_READ_TABLE helper. It depends only on the public rfc package so the
// whole cmd/ tool set can be extracted into a standalone repository.
package orfctool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/oisee/open-rfc-go/rfc"
)

// System is one named destination plus its tool policy, as stored in .rfc.json.
type System struct {
	Ashost   string   `json:"ashost"`
	Sysnr    string   `json:"sysnr"`
	Client   string   `json:"client"`
	User     string   `json:"user"`
	Password string   `json:"password"`
	Ticket   string   `json:"ticket"` // SAP logon ticket (MYSAPSSO2), instead of a password
	Lang     string   `json:"lang"`
	Expose   []string `json:"expose"`   // green-list FM name masks (rfc-mcp)
	Hide     []string `json:"hide"`     // red-list masks (rfc-mcp)
	ReadOnly bool     `json:"readOnly"` // disable rfc_call (rfc-mcp)
	MaxTools int      `json:"maxTools"` // cap on auto-discovered tools (rfc-mcp)
}

// Config is the on-disk .rfc.json shape: named systems and a default.
type Config struct {
	Systems map[string]System `json:"systems"`
	Default string            `json:"default"`
}

// Options carries the resolved tool policy (after config + flag overrides).
type Options struct {
	Expose   []string
	Hide     []string
	ReadOnly bool
	MaxTools int
}

// Load reads .rfc.json from the current directory, else the home directory.
// A missing file is not an error — env-only operation is supported.
func Load() Config {
	for _, p := range configPaths() {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var c Config
		if json.Unmarshal(b, &c) == nil {
			return c
		}
	}
	return Config{}
}

func configPaths() []string {
	paths := []string{".rfc.json"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".rfc.json"))
	}
	return paths
}

// resolve merges the named system (or the config default, or a single system)
// with environment overrides. Environment always wins over the file so secrets
// can stay out of .rfc.json.
func (c Config) resolve(name string) (System, error) {
	if name == "" {
		name = c.Default
	}
	if name == "" && len(c.Systems) == 1 {
		for k := range c.Systems {
			name = k
		}
	}
	sys := c.Systems[name] // zero value if absent (env-only)
	// Environment overrides.
	sys.Ashost = envOr("SAP_ASHOST", sys.Ashost)
	sys.Sysnr = envOr("SAP_SYSNR", firstNonEmpty(sys.Sysnr, "00"))
	sys.Client = envOr("SAP_CLIENT", firstNonEmpty(sys.Client, "001"))
	sys.User = envOr("SAP_USER", sys.User)
	sys.Lang = envOr("SAP_LANG", firstNonEmpty(sys.Lang, "EN"))
	if p := os.Getenv("SAP_PASSWORD"); p != "" {
		sys.Password = p
	} else if p := os.Getenv("SAP_PASSWD"); p != "" {
		sys.Password = p
	}
	if t := os.Getenv("SAP_TICKET"); t != "" {
		sys.Ticket = t
	}
	if sys.Ashost == "" {
		return System{}, fmt.Errorf("no host: set SAP_ASHOST or a system in .rfc.json (system %q)", name)
	}
	return sys, nil
}

// Open resolves the named system and dials a client with the default per-call
// timeout. It also returns the system's tool policy (Expose/Hide/ReadOnly/
// MaxTools) for rfc-mcp.
func Open(ctx context.Context, name string) (*rfc.Client, Options, error) {
	return OpenWithTimeout(ctx, name, 0)
}

// OpenWithTimeout is Open with an explicit bound on a single call (0 = the
// client default). Raise it for calls that block server-side — a debugger
// listener, a long-running report — where the default would time out first.
func OpenWithTimeout(ctx context.Context, name string, timeout time.Duration) (*rfc.Client, Options, error) {
	cfg := Load()
	sys, err := cfg.resolve(name)
	if err != nil {
		return nil, Options{}, err
	}
	n, err := strconv.Atoi(sys.Sysnr)
	if err != nil {
		return nil, Options{}, fmt.Errorf("sysnr must be numeric: %q", sys.Sysnr)
	}
	lang := sys.Lang
	if lang == "" {
		lang = "E"
	}
	c, err := rfc.Open(ctx, rfc.Destination{
		OperationTimeout: timeout,
		Host:             sys.Ashost,
		Port:             3300 + n,
		Service:          fmt.Sprintf("sapdp%02d", n),
		Client:           sys.Client,
		User:             sys.User,
		Password:         sys.Password,
		Ticket:           sys.Ticket,
		Language:         string([]rune(strings.ToUpper(lang))[0:1]),
	})
	if err != nil {
		return nil, Options{}, err
	}
	opts := Options{Expose: sys.Expose, Hide: sys.Hide, ReadOnly: sys.ReadOnly, MaxTools: sys.MaxTools}
	return c, opts, nil
}

// ReadTable runs RFC_READ_TABLE and splits each row into a column->value map by
// the FIELDS metadata the call returns.
func ReadTable(ctx context.Context, c *rfc.Client, table, where string, fields []string, top int) ([]map[string]string, error) {
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
			fs = append(fs, map[string]any{"FIELDNAME": strings.ToUpper(f)})
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

// LoadOptions returns the tool policy (Expose/Hide/ReadOnly/MaxTools) for a named
// system from .rfc.json, without dialing. rfc-mcp merges it with command-line
// flags (flags win).
func LoadOptions(name string) Options {
	c := Load()
	if name == "" {
		name = c.Default
	}
	if name == "" && len(c.Systems) == 1 {
		for k := range c.Systems {
			name = k
		}
	}
	s := c.Systems[name]
	return Options{Expose: s.Expose, Hide: s.Hide, ReadOnly: s.ReadOnly, MaxTools: s.MaxTools}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
