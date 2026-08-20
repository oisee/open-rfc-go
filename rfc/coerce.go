// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
)

// coerceParams converts loosely-typed input (as produced by JSON decoding —
// json.Number, float64, base64 strings, []any of objects) into the exact Go
// types the encoders expect, guided by the function interface. This lets callers
// pass JSON-native values (from a CLI or an MCP tool call) directly to Call.
// Unknown parameters are passed through so encodeCall can report them.
func coerceParams(iface metadata.RfcFunctionInterface, in Params, resolve structResolver) (Params, error) {
	byName := indexParams(iface)
	out := make(Params, len(in))
	for name, val := range in {
		p, ok := byName[name]
		if !ok {
			out[name] = val
			continue
		}
		cv, err := coerceParamValue(p, val, resolve)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %v", ErrProtocol, name, err)
		}
		out[name] = cv
	}
	return out, nil
}

func coerceParamValue(p classicrfc.FunintParameter, val any, resolve structResolver) (any, error) {
	if val == nil {
		return nil, nil
	}
	switch {
	// A TABLES parameter (class T) is a table even though its row EXID is a
	// structure (u/v) — check the class before the structure EXID.
	case p.ParameterClass == "T" || isTableExid(p.Exid):
		def, err := resolve(p.TableName)
		if err != nil {
			return nil, err
		}
		rows, err := asSlice(val)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(rows))
		for i, r := range rows {
			m, err := coerceStruct(def, r, resolve)
			if err != nil {
				return nil, fmt.Errorf("row %d: %v", i, err)
			}
			out = append(out, m)
		}
		return out, nil
	case isStructureExid(p.Exid):
		def, err := resolve(p.TableName)
		if err != nil {
			return nil, err
		}
		return coerceStruct(def, val, resolve)
	}
	return coerceScalar(p.Exid, val)
}

func coerceStruct(def rfctypes.RfcStructureDefinition, val any, resolve structResolver) (map[string]any, error) {
	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expects an object")
	}
	byField := make(map[string]rfctypes.RfcStructureField, len(def.Fields))
	for _, f := range def.Fields {
		byField[f.FieldName] = f
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		f, ok := byField[k]
		if !ok {
			out[k] = v
			continue
		}
		cv, err := coerceScalar(f.Exid, v)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", k, err)
		}
		out[k] = cv
	}
	return out, nil
}

// coerceScalar converts one JSON-native value to the Go type the classic/xRFC
// codecs expect for the given EXID.
func coerceScalar(exid string, v any) (any, error) {
	switch exid {
	case "I", "s", "b", "8":
		n, ok := toInt64(v)
		if !ok {
			return nil, fmt.Errorf("expects an integer")
		}
		return n, nil
	case "F", "a", "e":
		f, ok := toFloat64(v)
		if !ok {
			return nil, fmt.Errorf("expects a number")
		}
		return f, nil
	case "P":
		// packed decimal travels as a decimal string
		switch t := v.(type) {
		case string:
			return t, nil
		case json.Number:
			return t.String(), nil
		case float64:
			return strconv.FormatFloat(t, 'f', -1, 64), nil
		}
		return fmt.Sprint(v), nil
	case "X", "y":
		switch t := v.(type) {
		case []byte:
			return t, nil
		case string:
			b, err := base64.StdEncoding.DecodeString(t)
			if err != nil {
				return nil, fmt.Errorf("expects base64 for RAW/XSTRING: %v", err)
			}
			return b, nil
		}
		return nil, fmt.Errorf("expects base64 string or bytes")
	}
	// character-like (C, N, g, D, T, p, ...) — keep as string
	if s, ok := v.(string); ok {
		return s, nil
	}
	if n, ok := v.(json.Number); ok {
		return n.String(), nil
	}
	return fmt.Sprint(v), nil
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	}
	return 0, false
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

func asSlice(v any) ([]any, error) {
	switch t := v.(type) {
	case []any:
		return t, nil
	case []map[string]any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = t[i]
		}
		return out, nil
	}
	return nil, fmt.Errorf("expects an array")
}
