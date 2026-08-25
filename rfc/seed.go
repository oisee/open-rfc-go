// SPDX-License-Identifier: Apache-2.0

package rfc

// Seeded metadata: calling a function module without a runtime metadata lookup.
//
// A client normally discovers a function's interface over the wire —
// RFC_GET_FUNCTION_INTERFACE for the flat descriptor and RFC_METADATA_GET
// (DEEP) for the recursive type graph. On a locked-down system those lookups
// are exactly what S_RFC refuses, even when the target function itself is
// authorized. SeedMetadata removes the lookups for a known function: it takes
// a captured RFC_METADATA_GET response and pre-fills the function and graph
// caches, so FunctionInterface and Call serve the seeded function without a
// metadata call ever leaving the client. Unseeded functions behave exactly as
// before.
//
// The seed is normalized through the same code path as a live response
// (normalizeRecursiveGraph → metadata.NormalizeRecursiveFunctionResult), so a
// capture is validated as strictly as bytes off the wire. The caches have no
// expiry, and every lookup consults them first, so a seeded entry is never
// overwritten by a live lookup.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oisee/open-rfc-go/internal/metadata"
)

// ErrSeed reports a rejected metadata seed: not the JSON shape of an
// RFC_METADATA_GET response, an incomplete type closure, or rows the
// normalizer refuses.
var ErrSeed = errors.New("rfc: invalid metadata seed")

// seedTables are the RFC_METADATA_GET output tables the normalizer reads. A
// capture that omits one carries it as empty, exactly as a live decode does
// (decodeMetadataGetResult pre-seeds every bootstrap table empty).
var seedTables = []string{"FUNCTIONNAMES", "DATATYPESCONT", "INDIRECTTYPES", "PARAMETERS", "FUNC_ERRORS", "DD_ERRORS"}

// SeedMetadata pre-fills the client's metadata caches from a raw
// RFC_METADATA_GET response captured as JSON — the verbatim output of
//
//	RFC_METADATA_GET  DEEP = 'X', FUNCTIONNAMES = [ <one function> ]
//
// with the top-level tables (PARAMETERS, DATATYPESCONT, FUNCTIONNAMES, ...)
// as arrays of UPPERCASE-keyed rows. Numeric cells may be JSON numbers or
// digit strings; both wire forms occur in real captures.
//
// The capture must describe exactly one function and must be complete: a type
// the graph references without DATATYPESCONT rows rejects the whole seed with
// an error naming the type — a half-seeded function would fail later, on the
// wire, with a far worse error. On success the function's flat interface and
// recursive type graph are cached, and neither FunctionInterface nor Call
// performs a metadata lookup for it again. Seeding is all-or-nothing: on
// error the caches are untouched.
func (c *Client) SeedMetadata(raw []byte) error {
	entry, err := parseSeed(raw)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fnCache == nil {
		c.fnCache = map[string]metadata.RfcFunctionInterface{}
	}
	if c.graphCache == nil {
		c.graphCache = map[string]metadata.Graph{}
	}
	c.fnCache[entry.name] = entry.iface
	c.graphCache[entry.name] = entry.graph
	return nil
}

// CheckSeed validates a capture without a client: the same parse, closure
// check, and normalization SeedMetadata performs, minus the cache fill. Use it
// to test an embedded seed where no destination exists to dial.
func CheckSeed(raw []byte) error {
	_, err := parseSeed(raw)
	return err
}

// seedEntry is one parsed capture: the function it describes, its flat
// interface, and its recursive type graph.
type seedEntry struct {
	name  string
	iface metadata.RfcFunctionInterface
	graph metadata.Graph
}

func parseSeed(raw []byte) (seedEntry, error) {
	output, err := decodeSeedCapture(raw)
	if err != nil {
		return seedEntry{}, err
	}
	name, err := seedFunctionName(output)
	if err != nil {
		return seedEntry{}, err
	}
	if err := checkSeedClosure(output, name); err != nil {
		return seedEntry{}, err
	}
	iface, err := metadata.NormalizeFunction(name, output)
	if err != nil {
		return seedEntry{}, fmt.Errorf("%w: %s: %v", ErrSeed, name, err)
	}
	graph, err := normalizeRecursiveGraph(name, output)
	if err != nil {
		return seedEntry{}, fmt.Errorf("%w: %s: %v", ErrSeed, name, err)
	}
	return seedEntry{name: name, iface: iface, graph: graph}, nil
}

// decodeSeedCapture parses a captured RFC_METADATA_GET response into the
// map[string]any-of-row-slices shape the normalizers consume — the same shape
// decodeMetadataGetResult produces for a live response. Every cell becomes a
// string: JSON numbers keep their literal (the capture format carries
// INTLENGTH as a number in PARAMETERS and as a digit string in DATATYPESCONT).
func decodeSeedCapture(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var top map[string]any
	if err := dec.Decode(&top); err != nil {
		return nil, fmt.Errorf("%w: not a JSON object: %v", ErrSeed, err)
	}
	output := make(map[string]any, len(seedTables))
	for _, name := range seedTables {
		output[name] = []map[string]any{}
	}
	for key, value := range top {
		rows, ok := value.([]any)
		if !ok {
			// Scalar echoes (DEEP, LANGUAGE, ...) that a capture may carry;
			// the normalizer reads only tables.
			continue
		}
		converted := make([]map[string]any, 0, len(rows))
		for i, r := range rows {
			obj, ok := r.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%w: %s[%d] is not an object", ErrSeed, key, i)
			}
			row := make(map[string]any, len(obj))
			for k, v := range obj {
				switch t := v.(type) {
				case string:
					row[k] = t
				case json.Number:
					row[k] = t.String()
				default:
					return nil, fmt.Errorf("%w: %s[%d].%s must be a string or a number, got %T", ErrSeed, key, i, k, v)
				}
			}
			converted = append(converted, row)
		}
		output[key] = converted
	}
	return output, nil
}

// seedFunctionName reads the one function the capture describes.
func seedFunctionName(output map[string]any) (string, error) {
	rows := output["FUNCTIONNAMES"].([]map[string]any)
	if len(rows) != 1 {
		return "", fmt.Errorf("%w: FUNCTIONNAMES carries %d rows; a seed describes exactly one function", ErrSeed, len(rows))
	}
	name, _ := rows[0]["FUNCTIONNAME"].(string)
	if name == "" {
		return "", fmt.Errorf("%w: FUNCTIONNAMES[0].FUNCTIONNAME is missing", ErrSeed)
	}
	return name, nil
}

// checkSeedClosure verifies the capture's type closure is self-contained:
// every type the parameters, the type rows, or the indirect mappings reference
// must have DATATYPESCONT rows of its own. The graph normalizer would reject
// such a hole too, but with a positional code; a seed is authored by a person,
// so the error names the type instead.
func checkSeedClosure(output map[string]any, functionName string) error {
	typeRows := output["DATATYPESCONT"].([]map[string]any)
	have := make(map[string]bool, len(typeRows))
	for _, r := range typeRows {
		if n, _ := r["TYPENAME"].(string); n != "" {
			have[n] = true
		}
	}
	missing := func(referrer, target string) error {
		return fmt.Errorf("%w: seed for %s is incomplete: %s references type %s, which has no DATATYPESCONT rows",
			ErrSeed, functionName, referrer, target)
	}
	for _, r := range output["PARAMETERS"].([]map[string]any) {
		exid, _ := r["EXID"].(string)
		if exid != "u" && exid != "v" && exid != "h" {
			continue
		}
		field, _ := r["FIELDNAME"].(string)
		if strings.Contains(field, "-") {
			continue // resolves through INDIRECTTYPES, whose targets are checked below
		}
		target, _ := r["TABNAME"].(string)
		if target == "" {
			continue // the normalizer rejects the missing associated type itself
		}
		if !have[target] {
			name, _ := r["PARAMETER"].(string)
			return missing("parameter "+name, target)
		}
	}
	for _, r := range typeRows {
		intType, _ := r["INTTYPE"].(string)
		if intType != "u" && intType != "v" && intType != "h" {
			continue
		}
		target, _ := r["FIELDTYPE"].(string)
		if target == "" {
			continue
		}
		if !have[target] {
			owner, _ := r["TYPENAME"].(string)
			field, _ := r["FIELDNAME"].(string)
			return missing("field "+owner+"-"+field, target)
		}
	}
	for _, r := range output["INDIRECTTYPES"].([]map[string]any) {
		target, _ := r["FIELDTYPE"].(string)
		if target == "" {
			continue
		}
		if !have[target] {
			owner, _ := r["TABNAME"].(string)
			field, _ := r["FIELDNAME"].(string)
			return missing("indirect type "+owner+"-"+field, target)
		}
	}
	return nil
}
