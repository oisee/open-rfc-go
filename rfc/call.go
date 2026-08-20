// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/structure"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// Params is the input to a call: parameter name → native Go value. A scalar is
// a Go string / int / []byte; a structure is a map[string]any; a table is a
// []map[string]any.
type Params = map[string]any

// Result holds a call's decoded exports. Scalars and structures are read with
// Get; tables with Table.
type Result struct {
	scalars map[string]any
	tables  map[string][]map[string]any
}

// Get returns a scalar or structure export (nil if absent). A structure is a
// map[string]any.
func (r Result) Get(name string) any { return r.scalars[name] }

// Table returns a table export's rows (nil if absent).
func (r Result) Table(name string) []map[string]any { return r.tables[name] }

// Has reports whether a scalar/structure export named name was returned.
func (r Result) Has(name string) bool { _, ok := r.scalars[name]; return ok }

// structResolver resolves a DDIC structure definition by name.
type structResolver func(name string) (rfctypes.RfcStructureDefinition, error)

// Call invokes one function module with native Go values and returns its
// exports. requestedOutputs are derived from the interface, structure/table
// values are encoded from the discovered layout, and an ABAP-side failure is
// returned as an *ABAPException.
func (c *Client) Call(ctx context.Context, functionName string, in Params) (Result, error) {
	lease, err := c.pool.Acquire(ctx)
	if err != nil {
		return Result{}, err
	}
	defer lease.Release()
	sess := lease.Value()

	iface, err := c.functionInterfaceOn(ctx, sess, functionName)
	if err != nil {
		return Result{}, err
	}
	resolve := func(name string) (rfctypes.RfcStructureDefinition, error) {
		return c.structureDefinitionOn(ctx, sess, name)
	}
	input, err := encodeCall(iface, in, resolve)
	if err != nil {
		return Result{}, err
	}
	req, err := cpic.EncodeCutFunctionRequest(input)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrProtocol, err)
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		lease.Discard()
		return Result{}, fmt.Errorf("%w: %v", ErrTransport, err)
	}
	if exc := exceptionFromEnvelope(res.Envelope); exc != nil {
		return Result{}, exc
	}
	return decodeResult(iface, res.Fields, resolve)
}

func indexParams(iface metadata.RfcFunctionInterface) map[string]classicrfc.FunintParameter {
	byName := make(map[string]classicrfc.FunintParameter, len(iface.Parameters))
	for _, p := range iface.Parameters {
		byName[p.ParameterName] = p
	}
	return byName
}

func encodeCall(iface metadata.RfcFunctionInterface, in Params, resolve structResolver) (cpic.CutFunctionRequestInput, error) {
	input := cpic.CutFunctionRequestInput{FunctionName: iface.Name}
	byName := indexParams(iface)
	for name := range in {
		if _, ok := byName[name]; !ok {
			return input, fmt.Errorf("%w: %s.%s", ErrUnknownParameter, iface.Name, name)
		}
	}
	for _, p := range iface.Parameters {
		// Request every export, changing, and table parameter back.
		if p.ParameterClass == "E" || p.ParameterClass == "C" || p.ParameterClass == "T" {
			input.RequestedOutputs = append(input.RequestedOutputs, p.ParameterName)
		}
		val, supplied := in[p.ParameterName]
		if !supplied {
			continue
		}
		switch p.ParameterClass {
		case "T":
			def, err := resolve(p.TableName)
			if err != nil {
				return input, err
			}
			rows, ok := val.([]map[string]any)
			if !ok {
				return input, fmt.Errorf("%w: %s expects []map[string]any", ErrProtocol, p.ParameterName)
			}
			// A row layout that carries STRING/XSTRING can't be fixed-width
			// serialized — it travels as an xRFC XML parameter instead.
			if defNeedsXrfc(def) {
				b, err := xrfc.EncodeParameter(p.ParameterName, def, xrfc.KindTable, rows, xrfc.Limits{})
				if err != nil {
					return input, fmt.Errorf("%w: %s: %v", ErrProtocol, p.ParameterName, err)
				}
				input.XrfcParameters = append(input.XrfcParameters, cpic.NamedValue{Name: p.ParameterName, Value: b})
				continue
			}
			table := cpic.Table{Name: p.ParameterName, RowByteLength: int(def.ByteLength)}
			for i, row := range rows {
				b, err := structure.Encode(def, row)
				if err != nil {
					return input, fmt.Errorf("%w: %s row %d: %v", ErrProtocol, p.ParameterName, i, err)
				}
				table.Rows = append(table.Rows, b)
			}
			input.Tables = append(input.Tables, table)
		case "I", "C":
			// A table-typed import/changing parameter (EXID 'h') carries rows
			// like a TABLES parameter. If the row layout has STRING/XSTRING it
			// travels as an xRFC XML parameter; otherwise as a classic table.
			if isTableExid(p.Exid) {
				def, err := resolve(p.TableName)
				if err != nil {
					return input, err
				}
				rows, ok := val.([]map[string]any)
				if !ok {
					return input, fmt.Errorf("%w: %s expects []map[string]any", ErrProtocol, p.ParameterName)
				}
				if defNeedsXrfc(def) {
					b, err := xrfc.EncodeParameter(p.ParameterName, def, xrfc.KindTable, rows, xrfc.Limits{})
					if err != nil {
						return input, fmt.Errorf("%w: %s: %v", ErrProtocol, p.ParameterName, err)
					}
					input.XrfcParameters = append(input.XrfcParameters, cpic.NamedValue{Name: p.ParameterName, Value: b})
					continue
				}
				table := cpic.Table{Name: p.ParameterName, RowByteLength: int(def.ByteLength)}
				for i, row := range rows {
					b, err := structure.Encode(def, row)
					if err != nil {
						return input, fmt.Errorf("%w: %s row %d: %v", ErrProtocol, p.ParameterName, i, err)
					}
					table.Rows = append(table.Rows, b)
				}
				input.Tables = append(input.Tables, table)
				continue
			}
			// A structure whose layout carries STRING/XSTRING travels as an
			// xRFC XML parameter, not a fixed-width import.
			if isStructureExid(p.Exid) {
				def, err := resolve(p.TableName)
				if err != nil {
					return input, err
				}
				if defNeedsXrfc(def) {
					m, ok := val.(map[string]any)
					if !ok {
						return input, fmt.Errorf("%w: %s expects map[string]any", ErrProtocol, p.ParameterName)
					}
					b, err := xrfc.EncodeParameter(p.ParameterName, def, xrfc.KindStructure, m, xrfc.Limits{})
					if err != nil {
						return input, fmt.Errorf("%w: %s: %v", ErrProtocol, p.ParameterName, err)
					}
					input.XrfcParameters = append(input.XrfcParameters, cpic.NamedValue{Name: p.ParameterName, Value: b})
					continue
				}
			}
			b, err := encodeImport(p, val, resolve)
			if err != nil {
				return input, err
			}
			input.Imports = append(input.Imports, cpic.NamedValue{Name: p.ParameterName, Value: b})
		default:
			return input, fmt.Errorf("%w: %s has unsupported parameter class %s", ErrProtocol, p.ParameterName, p.ParameterClass)
		}
	}
	return input, nil
}

func encodeImport(p classicrfc.FunintParameter, val any, resolve structResolver) ([]byte, error) {
	if isStructureExid(p.Exid) {
		def, err := resolve(p.TableName)
		if err != nil {
			return nil, err
		}
		m, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: %s expects map[string]any", ErrProtocol, p.ParameterName)
		}
		return structure.Encode(def, m)
	}
	return encodeScalar(p, val)
}

func encodeScalar(p classicrfc.FunintParameter, val any) ([]byte, error) {
	n := int(p.InternalLength)
	switch p.Exid {
	case "C":
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s expects a string", ErrProtocol, p.ParameterName)
		}
		return classicrfc.EncodeAbapChar(s, n)
	case "N":
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s expects a numeric string", ErrProtocol, p.ParameterName)
		}
		if len(s) > n {
			return nil, fmt.Errorf("%w: %s does not fit NUMC(%d)", ErrProtocol, p.ParameterName, n)
		}
		return classicrfc.EncodeAbapChar(strings.Repeat("0", n-len(s))+s, n)
	case "I":
		v, ok := asInt32(val)
		if !ok {
			return nil, fmt.Errorf("%w: %s expects an integer", ErrProtocol, p.ParameterName)
		}
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(v))
		return b, nil
	case "X":
		b, ok := val.([]byte)
		if !ok {
			return nil, fmt.Errorf("%w: %s expects bytes", ErrProtocol, p.ParameterName)
		}
		if len(b) > n {
			return nil, fmt.Errorf("%w: %s accepts at most %d bytes", ErrProtocol, p.ParameterName, n)
		}
		out := make([]byte, n)
		copy(out, b)
		return out, nil
	case "g":
		// STRING: variable-length UTF-8 with a trailing NUL terminator, the
		// field chain delimits its length (no fixed width, no UTF-16 — unlike a
		// CHAR field).
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("%w: %s expects a string", ErrProtocol, p.ParameterName)
		}
		return append([]byte(s), 0), nil
	case "y":
		// XSTRING: variable-length raw bytes.
		b, ok := val.([]byte)
		if !ok {
			return nil, fmt.Errorf("%w: %s expects bytes", ErrProtocol, p.ParameterName)
		}
		return append([]byte(nil), b...), nil
	default:
		// DATE(D)/TIME(T)/packed DEC(P, incl. TIMESTAMP)/FLOAT(F)/INT1,2,8/
		// DECF16,34/UTCLONG(p): reuse the per-field structure codec via a
		// single-field layout. The value types match those used inside structures.
		b, err := structure.Encode(scalarFieldDef(p, n), map[string]any{"V": val})
		if err != nil {
			return nil, fmt.Errorf("%w: %s scalar type %s: %v", ErrProtocol, p.ParameterName, p.Exid, err)
		}
		return b, nil
	}
}

// scalarFieldDef builds a single-field structure definition so a scalar whose
// classic type is not in the fast path can be (de)serialized by the per-field
// structure codec. byteLen is the field's byte width (the target width when
// encoding, the actual value length when decoding).
func scalarFieldDef(p classicrfc.FunintParameter, byteLen int) rfctypes.RfcStructureDefinition {
	return rfctypes.RfcStructureDefinition{
		Name:       "SCALAR",
		ByteLength: int32(byteLen),
		Fields: []rfctypes.RfcStructureField{{
			TableName:      "SCALAR",
			FieldName:      "V",
			Position:       1,
			Offset:         0,
			InternalLength: int32(byteLen),
			Decimals:       p.Decimals,
			Exid:           p.Exid,
		}},
	}
}

func decodeResult(iface metadata.RfcFunctionInterface, fields []cpic.Field, resolve structResolver) (Result, error) {
	classic, err := classicrfc.DecodeResult(fields)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrProtocol, err)
	}
	byName := indexParams(iface)
	out := Result{scalars: map[string]any{}, tables: map[string][]map[string]any{}}

	for _, sc := range classic.Scalars {
		p, ok := byName[sc.Name]
		if !ok {
			out.scalars[sc.Name] = sc.Value
			continue
		}
		if isStructureExid(p.Exid) {
			def, err := resolve(p.TableName)
			if err != nil {
				return Result{}, err
			}
			m, err := structure.Decode(def, sc.Value)
			if err != nil {
				return Result{}, fmt.Errorf("%w: %s: %v", ErrProtocol, sc.Name, err)
			}
			out.scalars[sc.Name] = m
			continue
		}
		v, err := decodeScalar(p, sc.Value)
		if err != nil {
			return Result{}, err
		}
		out.scalars[sc.Name] = v
	}

	for _, t := range classic.Tables {
		p, ok := byName[t.Name]
		rows := make([]map[string]any, 0, len(t.Rows))
		if ok && p.TableName != "" {
			def, err := resolve(p.TableName)
			if err != nil {
				return Result{}, err
			}
			for i, row := range t.Rows {
				m, err := structure.Decode(def, row)
				if err != nil {
					return Result{}, fmt.Errorf("%w: %s row %d: %v", ErrProtocol, t.Name, i, err)
				}
				rows = append(rows, m)
			}
		}
		out.tables[t.Name] = rows
	}

	// Surface xRFC XML parameters (STRING/XSTRING-bearing structures and tables).
	// The parameter name lives inside the XML, so recover it, then decode against
	// the interface's layout for that parameter.
	for _, x := range classic.XrfcParameters {
		name, err := xrfc.DecodeParameterName(x.Value, xrfc.Limits{})
		if err != nil {
			return Result{}, fmt.Errorf("%w: xRFC parameter name: %v", ErrProtocol, err)
		}
		p, ok := byName[name]
		if !ok {
			continue
		}
		def, err := resolve(p.TableName)
		if err != nil {
			return Result{}, err
		}
		kind := xrfc.KindStructure
		if p.ParameterClass == "T" || isTableExid(p.Exid) {
			kind = xrfc.KindTable
		}
		v, err := xrfc.DecodeParameter(name, def, kind, x.Value, xrfc.Limits{})
		if err != nil {
			return Result{}, fmt.Errorf("%w: %s: %v", ErrProtocol, name, err)
		}
		if kind == xrfc.KindTable {
			rows, _ := v.([]map[string]any)
			out.tables[name] = rows
		} else {
			out.scalars[name] = v
		}
	}
	return out, nil
}

// defNeedsXrfc reports whether a structure/table row layout carries a STRING or
// XSTRING field (EXID g/y). Such a layout cannot be fixed-width serialized by the
// classic codec and must travel as an xRFC XML parameter instead.
func defNeedsXrfc(def rfctypes.RfcStructureDefinition) bool {
	for _, f := range def.Fields {
		if f.Exid == "g" || f.Exid == "y" {
			return true
		}
	}
	return false
}

func decodeScalar(p classicrfc.FunintParameter, b []byte) (any, error) {
	n := int(p.InternalLength)
	switch p.Exid {
	case "C", "N":
		return classicrfc.DecodeAbapChar(b, n)
	case "I":
		if len(b) < 4 {
			return nil, fmt.Errorf("%w: %s INT4 value is short", ErrProtocol, p.ParameterName)
		}
		return int32(binary.LittleEndian.Uint32(b)), nil
	case "X":
		return append([]byte(nil), b...), nil
	case "g":
		// STRING: variable-length UTF-8 with a trailing NUL terminator.
		return strings.TrimSuffix(string(b), "\x00"), nil
	case "y":
		// XSTRING: variable-length raw bytes.
		return append([]byte(nil), b...), nil
	default:
		// DATE/TIME/packed DEC/FLOAT/INT1,2,8/DECF/UTCLONG: decode through the
		// per-field structure codec, sized to the actual value bytes. Fall back
		// to raw bytes if the per-field codec cannot read it.
		m, err := structure.Decode(scalarFieldDef(p, len(b)), b)
		if err != nil {
			return append([]byte(nil), b...), nil
		}
		return m["V"], nil
	}
}

func asInt32(v any) (int32, bool) {
	switch n := v.(type) {
	case int:
		return int32(n), true
	case int32:
		return n, true
	case int64:
		return int32(n), true
	default:
		return 0, false
	}
}

// isStructureExid reports whether a parameter's EXID denotes a nested structure
// (u/v) rather than a scalar. A scalar CHAR may carry a TableName (its data
// element), so classification is by EXID, not by TableName.
func isStructureExid(exid string) bool { return exid == "u" || exid == "v" }

// isTableExid reports whether a parameter's EXID denotes a table type (a table
// passed as an import/changing parameter rather than a classic TABLES parameter).
func isTableExid(exid string) bool { return exid == "h" }
