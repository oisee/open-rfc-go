// SPDX-License-Identifier: Apache-2.0

package rfc

// Recursive (nested) parameter layouts.
//
// The flat metadata source — RFC_GET_STRUCTURE_DEFINITION — models a DDIC
// structure as one list of fields with an offset, a length and an EXID. That
// model cannot express a component that is itself a structure (DD03L DATATYPE
// 'STRU') or a table type ('TTYP'): the RFC_FIELDS rows the server returns for
// such a structure nest, so their offsets overlap and the flat validator
// rejects them with "RFC_FIELDS <F> overlaps its preceding field". Every
// function module with a nested parameter was therefore uncallable, and
// undescribable — a very common shape (anything carrying BAL_S_MSG, BAPIRET,
// a log context, …).
//
// The fix is to fetch the *type closure* instead of one flat layout:
// RFC_METADATA_GET with DEEP = 'X' returns the whole graph of types a function
// reaches (DATATYPESCONT / INDIRECTTYPES / PARAMETERS), which
// metadata.Normalize turns into a bounded, cycle-aware metadata.Graph, and
// which internal/xrfc serializes with the recursive xRFC codec.
//
// Design notes / trade-offs:
//
//   - RFC_METADATA_GET is verified RFC-enabled on the reference system
//     (TFDIR-FMODE = 'R' for both RFC_METADATA_GET and
//     RFC_METADATA_GET_TIMESTAMP). It is called through the hardcoded
//     bootstrap descriptor in internal/metadata (RfcMetadataGetBootstrapValue)
//     rather than through Client.Call, because resolving its own interface
//     over the wire would be circular and because the bootstrap pins the row
//     layouts the normalizer expects.
//
//   - The graph is fetched *lazily*, and only for the parameters that need
//     it. A parameter goes recursive when (a) its EXID is 'v' (deep structure)
//     or 'h' (table type as a non-TABLES parameter) — those have no
//     fixed-width form at all, and a server answers a classic table for either
//     with "Wrong parameter type in an RFC call"; (b) its flat layout cannot
//     be decoded; or (c) its flat layout itself declares a u/v/h component,
//     which the fixed-width codec rejects with "classic RFC type h is not
//     implemented". Everything else — plain structures (EXID 'u') and classic
//     TABLES parameters — keeps the existing fast path (one
//     RFC_GET_FUNCTION_INTERFACE plus one RFC_GET_STRUCTURE_DEFINITION per
//     structure) and costs nothing extra. The cost of that choice is that a
//     nested structure whose flat RFC_FIELDS happens to validate anyway keeps
//     its flattened layout.
//
//   - If RFC_METADATA_GET is missing or refuses (an old or locked-down
//     system), the fetch failure is cached and the call falls back to the flat
//     path, which then reports its own original error. Recursion is an
//     upgrade, never a new failure mode.

import (
	"context"
	"fmt"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/lifecycle"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/structure"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// maxGraphSchemaDepth bounds nested JSON-Schema / coercion recursion. The
// codec enforces its own depth budget; this only keeps schema rendering and
// input coercion finite on a cyclic graph.
const maxGraphSchemaDepth = 32

// graphEntry caches one RFC_METADATA_GET outcome, success or failure, so a
// system without recursive metadata is probed at most once per function.
type graphEntry struct {
	graph metadata.Graph
	err   error
}

// layoutPlan records, for one function, which parameters cannot be modelled
// flatly and must travel on the recursive xRFC codec instead. A nil plan means
// "everything is flat" — the fast path.
type layoutPlan struct {
	graph     metadata.Graph
	recursive map[string]xrfc.ResolvedParameter
}

// lookup returns the recursive plan for a parameter, if it has one.
func (p *layoutPlan) lookup(name string) (xrfc.ResolvedParameter, bool) {
	if p == nil {
		return xrfc.ResolvedParameter{}, false
	}
	rp, ok := p.recursive[name]
	return rp, ok
}

// paramHasLayout reports whether a parameter carries a DDIC layout (a
// structure, a table type, or a classic TABLES parameter) rather than a scalar.
func paramHasLayout(p classicrfc.FunintParameter) bool {
	return p.ParameterClass == "T" || isStructureExid(p.Exid) || isTableExid(p.Exid)
}

// defHasNestedField reports whether a flat layout admits, in its own EXIDs,
// that a component is itself a structure (u/v) or a table type (h). Such a
// layout is not fixed-width serializable: the classic codec rejects it with
// "classic RFC type h is not implemented". It is the second symptom of a
// recursive type — the first being RFC_FIELDS rows that overlap and so fail to
// decode at all.
func defHasNestedField(def rfctypes.RfcStructureDefinition) bool {
	for _, f := range def.Fields {
		if isStructureExid(f.Exid) || isTableExid(f.Exid) {
			return true
		}
	}
	return false
}

// planLayoutOn decides the per-parameter codec for one function. It resolves
// every layout-bearing parameter flatly first; only if some parameter's flat
// layout is unusable — it cannot be decoded, or it declares a nested component
// — does it fetch the recursive graph, and then only those parameters switch
// codec. Returns nil when the flat model covers everything.
func (c *Client) planLayoutOn(ctx context.Context, sess *lifecycle.Managed, iface metadata.RfcFunctionInterface, resolve structResolver) *layoutPlan {
	var unresolved []classicrfc.FunintParameter
	for _, p := range iface.Parameters {
		if !paramHasLayout(p) || p.TableName == "" {
			continue
		}
		if isTableExid(p.Exid) || p.Exid == "v" {
			// EXID 'h' (a table type passed as a non-TABLES parameter) and 'v'
			// (a deep structure) have no fixed-width form at all: the server
			// answers a classic table for either with "Wrong parameter type in
			// an RFC call". They are always graph-serialized.
			unresolved = append(unresolved, p)
			continue
		}
		def, err := resolve(p.TableName)
		if err != nil || defHasNestedField(def) {
			unresolved = append(unresolved, p)
		}
	}
	if len(unresolved) == 0 {
		return nil
	}
	graph, err := c.recursiveGraphOn(ctx, sess, iface.Name)
	if err != nil {
		// No recursive metadata available — leave the flat path to report the
		// original resolution failure.
		return nil
	}
	plan := &layoutPlan{graph: graph, recursive: map[string]xrfc.ResolvedParameter{}}
	for _, p := range unresolved {
		rp, needed, err := xrfc.ResolveParameter(graph, p)
		if err != nil || !needed {
			continue
		}
		plan.recursive[p.ParameterName] = rp
	}
	if len(plan.recursive) == 0 {
		return nil
	}
	return plan
}

// recursiveGraphOn returns the normalized RFC_METADATA_GET type graph for one
// function, cached per client (failures included).
func (c *Client) recursiveGraphOn(ctx context.Context, sess *lifecycle.Managed, functionName string) (metadata.Graph, error) {
	if e, ok := c.cachedGraph(functionName); ok {
		return e.graph, e.err
	}
	graph, err := fetchRecursiveGraph(ctx, sess, functionName, c.dest.Language)
	c.mu.Lock()
	c.graphCache[functionName] = graphEntry{graph: graph, err: err}
	c.mu.Unlock()
	return graph, err
}

func (c *Client) cachedGraph(name string) (graphEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.graphCache[name]
	return e, ok
}

// fetchRecursiveGraph runs RFC_METADATA_GET (DEEP) for one function through the
// hardcoded bootstrap descriptor and normalizes the answer into a type graph.
func fetchRecursiveGraph(ctx context.Context, sess *lifecycle.Managed, functionName, language string) (metadata.Graph, error) {
	if language == "" {
		language = "E"
	}
	inv, err := metadata.CreateFunctionInvocation(functionName, language[:1])
	if err != nil {
		return metadata.Graph{}, fmt.Errorf("%w: %v", ErrProtocol, err)
	}
	req, err := buildMetadataGetRequest(inv)
	if err != nil {
		return metadata.Graph{}, err
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		return metadata.Graph{}, fmt.Errorf("%w: %v", ErrTransport, err)
	}
	if exc := exceptionFromEnvelope(res.Envelope); exc != nil {
		return metadata.Graph{}, exc
	}
	output, err := decodeMetadataGetResult(res.Fields)
	if err != nil {
		return metadata.Graph{}, err
	}
	result, err := metadata.NormalizeRecursiveFunctionResult(functionName, output)
	if err != nil {
		return metadata.Graph{}, fmt.Errorf("%w: %v", ErrProtocol, err)
	}
	return result.Value, nil
}

// buildMetadataGetRequest serializes one RFC_METADATA_GET invocation from the
// bootstrap descriptor.
func buildMetadataGetRequest(inv metadata.RfcMetadataGetInvocation) ([]byte, error) {
	boot := metadata.RfcMetadataGetBootstrapValue
	in := cpic.CutFunctionRequestInput{FunctionName: boot.Metadata.Name}
	for _, p := range boot.Metadata.Parameters {
		value, supplied := inv.Input[p.ParameterName]
		if p.ParameterClass == "T" {
			in.RequestedOutputs = append(in.RequestedOutputs, p.ParameterName)
			def, ok := boot.Structures[p.TableName]
			if !ok {
				return nil, fmt.Errorf("%w: RFC_METADATA_GET bootstrap lacks %s", ErrProtocol, p.TableName)
			}
			rows, _ := value.([]map[string]any)
			table := cpic.Table{Name: p.ParameterName, RowByteLength: int(def.ByteLength)}
			for i, row := range rows {
				b, err := structure.Encode(def, row)
				if err != nil {
					return nil, fmt.Errorf("%w: RFC_METADATA_GET %s row %d: %v", ErrProtocol, p.ParameterName, i, err)
				}
				table.Rows = append(table.Rows, b)
			}
			in.Tables = append(in.Tables, table)
			continue
		}
		if !supplied {
			continue
		}
		s, _ := value.(string)
		b, err := classicrfc.EncodeAbapChar(s, int(p.InternalLength))
		if err != nil {
			return nil, fmt.Errorf("%w: RFC_METADATA_GET %s: %v", ErrProtocol, p.ParameterName, err)
		}
		in.Imports = append(in.Imports, cpic.NamedValue{Name: p.ParameterName, Value: b})
	}
	req, err := cpic.EncodeCutFunctionRequest(in)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProtocol, err)
	}
	return req, nil
}

// decodeMetadataGetResult turns an RFC_METADATA_GET answer into the
// map[string]any of row slices metadata.NormalizeRecursiveFunctionResult
// expects. Every bootstrap table is pre-seeded empty so an omitted table is an
// empty array rather than a missing key.
func decodeMetadataGetResult(fields []cpic.Field) (map[string]any, error) {
	boot := metadata.RfcMetadataGetBootstrapValue
	result, err := classicrfc.DecodeResult(fields)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProtocol, err)
	}
	rowType := map[string]string{}
	output := map[string]any{}
	for _, p := range boot.Metadata.Parameters {
		if p.ParameterClass != "T" {
			continue
		}
		rowType[p.ParameterName] = p.TableName
		output[p.ParameterName] = []map[string]any{}
	}
	for _, t := range result.Tables {
		def, ok := boot.Structures[rowType[t.Name]]
		if !ok {
			continue
		}
		rows := make([]map[string]any, 0, len(t.Rows))
		for i, row := range t.Rows {
			m, err := decodeFixedRow(def, row)
			if err != nil {
				return nil, fmt.Errorf("%w: RFC_METADATA_GET %s row %d: %v", ErrProtocol, t.Name, i, err)
			}
			rows = append(rows, m)
		}
		output[t.Name] = rows
	}
	return output, nil
}

// --- input coercion against a type graph ------------------------------------

// coerceGraphParam converts loosely typed input (JSON-native values) into the
// shapes xrfc.EncodeRecursiveParameter expects: map[string]any for a structure
// parameter and []any for a table parameter, recursively.
func coerceGraphParam(g metadata.Graph, rp xrfc.ResolvedParameter, v any) (any, error) {
	if rp.Kind == xrfc.KindTable {
		return coerceGraphTable(g, rp.Node, v, 0)
	}
	return coerceGraphStructure(g, rp.Node, v, 0)
}

func coerceGraphTable(g metadata.Graph, node metadata.TypeNode, v any, depth int) (any, error) {
	if depth > maxGraphSchemaDepth {
		return nil, fmt.Errorf("nested value is deeper than %d levels", maxGraphSchemaDepth)
	}
	rows, err := asSlice(v)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(rows))
	for i, row := range rows {
		cv, err := coerceGraphLine(g, node, row, depth+1)
		if err != nil {
			return nil, fmt.Errorf("row %d: %v", i, err)
		}
		out = append(out, cv)
	}
	return out, nil
}

// coerceGraphLine coerces one table row: an anonymous single-field node is a
// table of a line type (scalar, structure or table), anything else a structure.
func coerceGraphLine(g metadata.Graph, node metadata.TypeNode, v any, depth int) (any, error) {
	if len(node.Fields) == 1 && node.Fields[0].Name == "" {
		return coerceGraphReference(g, node.Fields[0], v, depth)
	}
	return coerceGraphStructure(g, node, v, depth)
}

func coerceGraphStructure(g metadata.Graph, node metadata.TypeNode, v any, depth int) (any, error) {
	if depth > maxGraphSchemaDepth {
		return nil, fmt.Errorf("nested value is deeper than %d levels", maxGraphSchemaDepth)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expects an object")
	}
	byName := make(map[string]metadata.MetadataField, len(node.Fields))
	for _, f := range node.Fields {
		byName[f.Name] = f
	}
	out := make(map[string]any, len(m))
	for k, val := range m {
		f, ok := byName[k]
		if !ok {
			// Leave it in place: the encoder rejects unknown fields with a
			// message that names the offending component.
			out[k] = val
			continue
		}
		cv, err := coerceGraphReference(g, f, val, depth+1)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", k, err)
		}
		out[k] = cv
	}
	return out, nil
}

func coerceGraphReference(g metadata.Graph, f metadata.MetadataField, v any, depth int) (any, error) {
	if f.Reference.Kind == "scalar" {
		return coerceScalar(f.InternalType, v)
	}
	target, ok := g.Nodes[f.Reference.TargetType]
	if !ok || f.Reference.Cyclic {
		return v, nil
	}
	if target.Kind == "table" {
		return coerceGraphTable(g, target, v, depth)
	}
	return coerceGraphStructure(g, target, v, depth)
}

// --- output normalization ---------------------------------------------------

// normalizeGraphValue rewrites the codec's []any row slices into
// []map[string]any so decoded nested tables carry the same Go type as flat
// table exports. Scalar-line tables (rows that are not objects) are left as-is.
func normalizeGraphValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, e := range t {
			t[k] = normalizeGraphValue(e)
		}
		return t
	case []any:
		rows := make([]map[string]any, 0, len(t))
		for i := range t {
			t[i] = normalizeGraphValue(t[i])
			if m, ok := t[i].(map[string]any); ok {
				rows = append(rows, m)
			}
		}
		if len(rows) == len(t) {
			return rows
		}
		return t
	}
	return v
}

// --- JSON Schema over a type graph ------------------------------------------

// graphParamSchema renders a recursive parameter as JSON Schema: a nested
// structure becomes a nested object, a nested table an array of them.
func graphParamSchema(g metadata.Graph, rp xrfc.ResolvedParameter) map[string]any {
	if rp.Kind == xrfc.KindTable {
		return map[string]any{"type": "array", "description": rp.Node.Name, "items": graphLineSchema(g, rp.Node, 0)}
	}
	return graphNodeSchema(g, rp.Node, 0)
}

func graphLineSchema(g metadata.Graph, node metadata.TypeNode, depth int) map[string]any {
	if len(node.Fields) == 1 && node.Fields[0].Name == "" {
		return graphFieldSchema(g, node.Fields[0], depth)
	}
	return graphNodeSchema(g, node, depth)
}

func graphNodeSchema(g metadata.Graph, node metadata.TypeNode, depth int) map[string]any {
	if node.Kind == "table" {
		return map[string]any{"type": "array", "description": node.Name, "items": graphLineSchema(g, node, depth+1)}
	}
	props := map[string]any{}
	for _, f := range node.Fields {
		props[f.Name] = graphFieldSchema(g, f, depth+1)
	}
	return map[string]any{"type": "object", "description": node.Name, "properties": props}
}

func graphFieldSchema(g metadata.Graph, f metadata.MetadataField, depth int) map[string]any {
	if f.Reference.Kind == "scalar" {
		return fieldSchema(f.InternalType, int32(f.UcLength), int32(f.Decimals))
	}
	target, ok := g.Nodes[f.Reference.TargetType]
	if !ok || f.Reference.Cyclic || depth > maxGraphSchemaDepth {
		note := f.Reference.TargetType
		if f.Reference.Cyclic {
			note += " (cyclic)"
		}
		if f.Reference.Kind == "table" {
			return map[string]any{"type": "array", "description": note}
		}
		return map[string]any{"type": "object", "description": note}
	}
	return graphNodeSchema(g, target, depth)
}
