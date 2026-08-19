// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/recursive-metadata.test.ts at commit 847036d,
// Copyright 2026 Marian Zeis, Apache-2.0. Rewritten for the testing package
// with typed input rows. The getter/proxy/sparse-array cases and the top-level
// UNKNOWN_PROPERTY/ACCESSOR_PROPERTY/MISSING_PROPERTY cases have no Go analogue
// (the typed Input struct removes that attack class); the hostile-text
// INVALID_TEXT case is kept. See docs/provenance.md.

package metadata

import (
	"fmt"
	"reflect"
	"testing"
)

const testTS = "20260716010203"

type trOpts struct {
	typeName, fieldName, fieldType, internalType, componentType, dataType, timestamp, description string
	nucTotal, ucTotal, nucOffset, ucOffset, nucLength, ucLength, decimals                         int
}

func defTR() trOpts {
	return trOpts{componentType: "E", dataType: "CHAR", nucTotal: 8, ucTotal: 8, nucLength: 8, ucLength: 8, timestamp: testTS}
}

func p6(n int) string { return fmt.Sprintf("%06d", n) }

func mkTypeRow(o trOpts) TypeRowInput {
	ts := o.timestamp
	if ts == "" {
		ts = testTS
	}
	if o.componentType == "" {
		o.componentType = "E"
	}
	if o.dataType == "" {
		o.dataType = "CHAR"
	}
	return TypeRowInput{
		TypeName: o.typeName, FieldName: o.fieldName, CompType: o.componentType, FieldType: o.fieldType,
		DataType: o.dataType, TabLength: p6(o.nucTotal), TabLengthUC: p6(o.ucTotal), Description: o.description,
		Decimals: p6(o.decimals), IntType: o.internalType, Offset: p6(o.nucOffset), OffsetUC: p6(o.ucOffset),
		IntLen: p6(o.nucLength), IntLenUC: p6(o.ucLength), Timestamp: ts,
	}
}

type prOpts struct {
	name, parameterClass, tableName, fieldPath, internalType, position, defaultValue, parameterText, functionName string
	optional                                                                                                      bool
}

func paramRow(o prOpts) ParameterRowInput {
	pc := o.parameterClass
	if pc == "" {
		pc = "I"
	}
	exid := o.internalType
	if exid == "" {
		exid = "C"
	}
	pos := o.position
	if pos == "" {
		pos = "1"
	}
	fn := o.functionName
	if fn == "" {
		fn = "Z_GRAPH_TEST"
	}
	opt := ""
	if o.optional {
		opt = "X"
	}
	return ParameterRowInput{
		FuncName: fn, ParamClass: pc, Parameter: o.name, TabName: o.tableName, FieldName: o.fieldPath,
		Exid: exid, Position: pos, Offset: "0", IntLength: "8", Decimals: "0",
		Default: o.defaultValue, ParamText: o.parameterText, Optional: opt,
	}
}

func fnRow() FunctionRow {
	return FunctionRow{FunctionName: "Z_GRAPH_TEST", BasxmlSupported: "", UDat: "20260716", UTime: "010203"}
}

func meta(typeRows []TypeRowInput, params []ParameterRowInput, indirect []IndirectRowInput) Input {
	fns := []FunctionRow{fnRow()}
	return Input{FunctionNames: &fns, DataTypesCont: typeRows, IndirectTypes: indirect, Parameters: &params}
}

func normOK(t *testing.T, in Input, opts *Options) Graph {
	t.Helper()
	g, err := Normalize(in, opts)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	return g
}

func errCode(t *testing.T, in Input, opts *Options, code string) {
	t.Helper()
	_, err := Normalize(in, opts)
	rme, ok := err.(*RecursiveMetadataError)
	if !ok || rme.Code != code {
		t.Fatalf("want code %s, got %v", code, err)
	}
}

func TestNormalizesNestedStructureGraph(t *testing.T) {
	g := normOK(t, meta(
		[]TypeRowInput{
			mkTypeRow(trOpts{typeName: "Z_PARENT", fieldName: "CHILD", fieldType: "Z_CHILD", internalType: "u", componentType: "S", dataType: "STRU", nucTotal: 8, ucTotal: 8, nucLength: 8, ucLength: 8}),
			mkTypeRow(trOpts{typeName: "Z_CHILD", fieldName: "TEXT", fieldType: "CHAR4", internalType: "C", nucTotal: 4, ucTotal: 8, nucLength: 4, ucLength: 8, componentType: "E", dataType: "CHAR"}),
		},
		[]ParameterRowInput{paramRow(prOpts{name: "VALUE", tableName: "Z_PARENT", internalType: "u", defaultValue: "'DEFAULT'", parameterText: "Nested value", optional: true})},
		nil,
	), nil)

	parent := g.Nodes["Z_PARENT"]
	if parent.Kind != "structure" {
		t.Fatalf("kind = %s", parent.Kind)
	}
	want := MetadataField{Name: "CHILD", Position: 1, ComponentType: "S", AssociatedType: "Z_CHILD", DataType: "STRU", InternalType: "u", Description: "", Decimals: 0, NucOffset: 0, UcOffset: 0, NucLength: 8, UcLength: 8, Reference: Reference{Kind: "structure", TargetType: "Z_CHILD", Cyclic: false}}
	if !reflect.DeepEqual(parent.Fields[0], want) {
		t.Fatalf("field = %+v", parent.Fields[0])
	}
	if g.Statistics.MaximumDepth != 2 {
		t.Fatalf("depth = %d", g.Statistics.MaximumDepth)
	}
	if g.Parameters[0].Reference.Kind != "structure" || g.Parameters[0].DefaultValue != "'DEFAULT'" || g.Parameters[0].ParameterText != "Nested value" || !g.Parameters[0].Optional {
		t.Fatalf("param = %+v", g.Parameters[0])
	}
	if g.FunctionIdentity == nil || *g.FunctionIdentity != (FunctionIdentity{Name: "Z_GRAPH_TEST", RemoteBasxmlSupported: false, GenerationToken: "function:20260716:010203"}) {
		t.Fatalf("identity = %+v", g.FunctionIdentity)
	}
}

func TestZeroParameterRFMIdentity(t *testing.T) {
	fns := []FunctionRow{{FunctionName: "Z_EMPTY_RFM", BasxmlSupported: "X", UDat: "20260715", UTime: "235959"}}
	empty := []ParameterRowInput{}
	g := normOK(t, Input{FunctionNames: &fns, DataTypesCont: nil, IndirectTypes: nil, Parameters: &empty}, nil)
	if g.FunctionIdentity == nil || *g.FunctionIdentity != (FunctionIdentity{Name: "Z_EMPTY_RFM", RemoteBasxmlSupported: true, GenerationToken: "function:20260715:235959"}) {
		t.Fatalf("identity = %+v", g.FunctionIdentity)
	}
	if len(g.Parameters) != 0 || g.Statistics.RowCount != 1 {
		t.Fatalf("params=%d rows=%d", len(g.Parameters), g.Statistics.RowCount)
	}
}

func TestZeroPositionsAndOrder(t *testing.T) {
	g := normOK(t, meta(nil, []ParameterRowInput{
		paramRow(prOpts{name: "ZERO_FIRST", position: "0"}),
		paramRow(prOpts{name: "ZERO_SECOND", position: "0"}),
		paramRow(prOpts{name: "LATER", position: "2"}),
	}, nil), nil)
	got := []struct {
		name string
		pos  int
	}{}
	for _, p := range g.Parameters {
		got = append(got, struct {
			name string
			pos  int
		}{p.Name, p.Position})
	}
	if got[0].name != "ZERO_FIRST" || got[0].pos != 0 || got[1].name != "ZERO_SECOND" || got[2].name != "LATER" || got[2].pos != 2 {
		t.Fatalf("order = %+v", got)
	}
	for _, bad := range []string{"-1", "invalid", "1.5"} {
		errCode(t, meta(nil, []ParameterRowInput{paramRow(prOpts{name: "BAD", position: bad})}, nil), nil, "INVALID_INTEGER")
	}
}

func TestRejectsMultipleAndForeignFunctions(t *testing.T) {
	fns2 := []FunctionRow{fnRow(), {FunctionName: "Z_OTHER", UDat: "20260716", UTime: "010203"}}
	empty := []ParameterRowInput{}
	errCode(t, Input{FunctionNames: &fns2, Parameters: &empty}, nil, "MULTIPLE_FUNCTION_IDENTITIES")

	fnsOther := []FunctionRow{{FunctionName: "Z_OTHER", UDat: "20260716", UTime: "010203"}}
	params := []ParameterRowInput{paramRow(prOpts{name: "VALUE"})}
	errCode(t, Input{FunctionNames: &fnsOther, Parameters: &params}, nil, "FOREIGN_FUNCTION_REFERENCE")

	mixed := []ParameterRowInput{paramRow(prOpts{name: "LEFT"}), paramRow(prOpts{name: "RIGHT", functionName: "Z_OTHER"})}
	errCode(t, Input{DataTypesCont: nil, IndirectTypes: nil, Parameters: &mixed}, nil, "MULTIPLE_FUNCTIONS")
}

func TestTableInStructureAndSharedIdentity(t *testing.T) {
	g := normOK(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_ROOT", fieldName: "ROWS", fieldType: "Z_TT_ROW", internalType: "h", componentType: "T", dataType: "TTYP", nucTotal: 16, ucTotal: 16, nucLength: 8, ucLength: 8}),
		mkTypeRow(trOpts{typeName: "Z_ROOT", fieldName: "SHARED", fieldType: "Z_ROW", internalType: "u", componentType: "S", dataType: "STRU", nucTotal: 16, ucTotal: 16, nucOffset: 8, ucOffset: 8, nucLength: 8, ucLength: 8}),
		mkTypeRow(trOpts{typeName: "Z_TT_ROW", fieldName: "", fieldType: "Z_ROW", internalType: "u", componentType: "S", dataType: "STRU"}),
		mkTypeRow(trOpts{typeName: "Z_ROW", fieldName: "PAYLOAD", fieldType: "RAWSTRING", internalType: "y", dataType: "RSTR"}),
	}, []ParameterRowInput{paramRow(prOpts{name: "ROOT", tableName: "Z_ROOT", internalType: "u"})}, nil), nil)

	root := g.Nodes["Z_ROOT"]
	table := g.Nodes["Z_TT_ROW"]
	if root.Fields[0].Reference != (Reference{Kind: "table", TargetType: "Z_TT_ROW"}) {
		t.Fatalf("root ref = %+v", root.Fields[0].Reference)
	}
	if table.Kind != "table" || table.Fields[0].Name != "" || table.Fields[0].Reference != (Reference{Kind: "structure", TargetType: "Z_ROW"}) {
		t.Fatalf("table = %+v", table)
	}
	if g.Nodes["Z_ROW"].Fields[0].Reference != (Reference{Kind: "scalar", InternalType: "y"}) {
		t.Fatalf("row ref = %+v", g.Nodes["Z_ROW"].Fields[0].Reference)
	}
	if root.Fields[1].Reference.TargetType != table.Fields[0].Reference.TargetType {
		t.Fatalf("shared identity mismatch")
	}
}

func TestDeepVRows(t *testing.T) {
	g := normOK(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_TT_DEEP_ROW", fieldName: "", fieldType: "Z_DEEP_ROW", internalType: "v", componentType: "S", dataType: "STRU", nucTotal: 8, ucTotal: 8, nucLength: 0, ucLength: 0}),
		mkTypeRow(trOpts{typeName: "Z_DEEP_ROW", fieldName: "TEXT", fieldType: "STRING", internalType: "g", dataType: "STRG", nucTotal: 16, ucTotal: 16}),
		mkTypeRow(trOpts{typeName: "Z_DEEP_ROW", fieldName: "BYTES", fieldType: "XSTRING", internalType: "y", dataType: "RSTR", nucOffset: 8, ucOffset: 8, nucTotal: 16, ucTotal: 16}),
	}, []ParameterRowInput{paramRow(prOpts{name: "ROWS", parameterClass: "I", tableName: "Z_TT_DEEP_ROW", internalType: "h", position: "0"})}, nil), nil)

	if g.Nodes["Z_TT_DEEP_ROW"].Kind != "table" || g.Nodes["Z_TT_DEEP_ROW"].Fields[0].Reference != (Reference{Kind: "structure", TargetType: "Z_DEEP_ROW"}) {
		t.Fatalf("deep table = %+v", g.Nodes["Z_TT_DEEP_ROW"])
	}
	if g.Parameters[0].Reference != (ParameterReference{Kind: "table", TargetType: "Z_TT_DEEP_ROW"}) {
		t.Fatalf("param ref = %+v", g.Parameters[0].Reference)
	}
	refs := []Reference{g.Nodes["Z_DEEP_ROW"].Fields[0].Reference, g.Nodes["Z_DEEP_ROW"].Fields[1].Reference}
	if refs[0] != (Reference{Kind: "scalar", InternalType: "g"}) || refs[1] != (Reference{Kind: "scalar", InternalType: "y"}) {
		t.Fatalf("deep row refs = %+v", refs)
	}
}

func TestAnonymousScalarGeometry(t *testing.T) {
	g := normOK(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "CHAR255", fieldName: "", fieldType: "CHAR255", internalType: "C", nucTotal: 0, ucTotal: 0, nucLength: 255, ucLength: 510}),
	}, []ParameterRowInput{paramRow(prOpts{name: "TEXT", tableName: "CHAR255", internalType: "C"})}, nil), nil)
	node := g.Nodes["CHAR255"]
	if node.Kind != "scalar" || node.NucLength != 255 || node.UcLength != 510 || node.Fields[0].Reference != (Reference{Kind: "scalar", InternalType: "C"}) {
		t.Fatalf("node = %+v", node)
	}
}

func TestIndirectFunctionFieldPaths(t *testing.T) {
	g := normOK(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_WRAPPER", fieldName: "VALUE", fieldType: "Z_VALUE", internalType: "u", componentType: "S", dataType: "STRU"}),
		mkTypeRow(trOpts{typeName: "Z_VALUE", fieldName: "TEXT", fieldType: "CHAR8", internalType: "C"}),
	}, []ParameterRowInput{paramRow(prOpts{name: "NESTED", tableName: "Z_OUTER", fieldPath: "INNER-VALUE", internalType: "u"})},
		[]IndirectRowInput{{TabName: "Z_OUTER", FieldName: "INNER-VALUE", FieldType: "Z_WRAPPER"}}), nil)
	p := g.Parameters[0]
	if p.AssociatedType != "Z_OUTER" || p.FieldPath != "INNER-VALUE" || p.Reference != (ParameterReference{Kind: "structure", TargetType: "Z_WRAPPER"}) {
		t.Fatalf("param = %+v", p)
	}
}

func TestScalarTablesParameters(t *testing.T) {
	g := normOK(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "SYST", fieldName: "LISEL", fieldType: "CHAR255", internalType: "C", nucTotal: 255, ucTotal: 510, nucLength: 255, ucLength: 510}),
	}, []ParameterRowInput{paramRow(prOpts{name: "LINES", parameterClass: "T", tableName: "SYST", fieldPath: "LISEL", internalType: "C"})}, nil), nil)
	if g.Parameters[0].Reference != (ParameterReference{Kind: "table", HasScalarLine: true, ScalarLineInternalType: "C"}) {
		t.Fatalf("ref = %+v", g.Parameters[0].Reference)
	}
	if !reflect.DeepEqual(g.RootTypeNames, []string{"SYST"}) || g.Statistics.EdgeCount != 1 {
		t.Fatalf("roots=%v edges=%d", g.RootTypeNames, g.Statistics.EdgeCount)
	}
}

func TestIndirectElementaryScalarNodes(t *testing.T) {
	g := normOK(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_ELEMENT", fieldName: "", fieldType: "CHAR8", internalType: "C"}),
	}, []ParameterRowInput{
		paramRow(prOpts{name: "VALUE", tableName: "Z_OUTER", fieldPath: "INNER-VALUE", internalType: "C"}),
		paramRow(prOpts{name: "DIRECT", tableName: "Z_ELEMENT", internalType: "C", position: "2"}),
	}, []IndirectRowInput{{TabName: "Z_OUTER", FieldName: "INNER-VALUE", FieldType: "Z_ELEMENT"}}), nil)
	if g.Nodes["Z_ELEMENT"].Kind != "scalar" || g.Nodes["Z_ELEMENT"].Fields[0].Reference != (Reference{Kind: "scalar", InternalType: "C"}) {
		t.Fatalf("element = %+v", g.Nodes["Z_ELEMENT"])
	}
	if g.Parameters[0].Reference != (ParameterReference{Kind: "scalar", InternalType: "C"}) || g.Parameters[1].Reference != (ParameterReference{Kind: "scalar", InternalType: "C"}) {
		t.Fatalf("param refs = %+v %+v", g.Parameters[0].Reference, g.Parameters[1].Reference)
	}
	if !reflect.DeepEqual(g.RootTypeNames, []string{"Z_ELEMENT"}) {
		t.Fatalf("roots = %v", g.RootTypeNames)
	}
}

func TestPromotesScalarTableByIncomingEdge(t *testing.T) {
	roots := []string{"Z_ROOT"}
	g := normOK(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_ROOT", fieldName: "VALUES", fieldType: "Z_TT_TEXT", internalType: "h", componentType: "T", dataType: "TTYP"}),
		mkTypeRow(trOpts{typeName: "Z_TT_TEXT", fieldName: "", fieldType: "CHAR8", internalType: "C"}),
	}, nil, nil), &Options{RootTypeNames: roots})
	if g.Nodes["Z_TT_TEXT"].Kind != "table" || g.Nodes["Z_TT_TEXT"].Fields[0].Reference != (Reference{Kind: "scalar", InternalType: "C"}) {
		t.Fatalf("promoted = %+v", g.Nodes["Z_TT_TEXT"])
	}
}

func TestDescriptorCycles(t *testing.T) {
	g := normOK(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_A", fieldName: "B", fieldType: "Z_B", internalType: "u", componentType: "S", dataType: "STRU"}),
		mkTypeRow(trOpts{typeName: "Z_B", fieldName: "A", fieldType: "Z_A", internalType: "u", componentType: "S", dataType: "STRU"}),
	}, nil, nil), &Options{RootTypeNames: []string{"Z_A"}})
	if len(g.Cycles) != 1 || g.Cycles[0].ID != "cycle:0" || !reflect.DeepEqual(g.Cycles[0].TypeNames, []string{"Z_A", "Z_B"}) {
		t.Fatalf("cycles = %+v", g.Cycles)
	}
	if !g.Nodes["Z_A"].Fields[0].Reference.Cyclic || !g.Nodes["Z_B"].Fields[0].Reference.Cyclic {
		t.Fatalf("not cyclic")
	}
	if g.Statistics.MaximumDepth != 1 {
		t.Fatalf("depth = %d", g.Statistics.MaximumDepth)
	}
}

func TestEmptyGraph(t *testing.T) {
	g := normOK(t, Input{DataTypesCont: nil, IndirectTypes: nil}, nil)
	if len(g.Nodes) != 0 || len(g.Parameters) != 0 || len(g.RootTypeNames) != 0 || len(g.Cycles) != 0 || g.Statistics.MaximumDepth != 0 {
		t.Fatalf("graph = %+v", g)
	}
}

func TestRejectsCorruptGeometryDuplicatesForeign(t *testing.T) {
	errCode(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_BAD", fieldName: "A", fieldType: "CHAR8", internalType: "C", nucTotal: 12, ucTotal: 12, nucLength: 8, ucLength: 8}),
		mkTypeRow(trOpts{typeName: "Z_BAD", fieldName: "B", fieldType: "CHAR8", internalType: "C", nucTotal: 12, ucTotal: 12, nucOffset: 4, ucOffset: 4, nucLength: 8, ucLength: 8}),
	}, nil, nil), nil, "INVALID_GEOMETRY")

	errCode(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_DUP", fieldName: "A", fieldType: "CHAR", internalType: "C"}),
		mkTypeRow(trOpts{typeName: "Z_DUP", fieldName: "A", fieldType: "CHAR", internalType: "C"}),
	}, nil, nil), nil, "DUPLICATE_FIELD")

	errCode(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_ROOT", fieldName: "CHILD", fieldType: "Z_MISSING", internalType: "u", componentType: "S", dataType: "STRU"}),
	}, nil, nil), nil, "FOREIGN_TYPE_REFERENCE")

	errCode(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_ROOT", fieldName: "A", fieldType: "CHAR", internalType: "C"}),
		mkTypeRow(trOpts{typeName: "Z_FOREIGN", fieldName: "B", fieldType: "CHAR", internalType: "C"}),
	}, nil, nil), &Options{RootTypeNames: []string{"Z_ROOT"}}, "FOREIGN_TYPE_NODE")
}

func TestRejectsForeignAndDuplicateIndirect(t *testing.T) {
	rows := []TypeRowInput{mkTypeRow(trOpts{typeName: "Z_TARGET", fieldName: "A", fieldType: "CHAR", internalType: "C"})}
	indirect := []IndirectRowInput{{TabName: "Z_OUTER", FieldName: "A-B", FieldType: "Z_TARGET"}}
	errCode(t, meta(rows, nil, indirect), nil, "FOREIGN_INDIRECT_TYPE")
	errCode(t, meta(rows, []ParameterRowInput{paramRow(prOpts{name: "VALUE", tableName: "Z_OUTER", fieldPath: "A-B", internalType: "u"})}, append(append([]IndirectRowInput{}, indirect...), indirect...)), nil, "DUPLICATE_INDIRECT_TYPE")
}

func TestEnforcesResourceLimits(t *testing.T) {
	scalar := mkTypeRow(trOpts{typeName: "Z_ONE", fieldName: "A", fieldType: "CHAR", internalType: "C"})
	lim := func(l Limits) *Options { return &Options{Limits: &l} }
	full := func() Limits { return absoluteLimits }
	set := func(field string, v int) Limits {
		l := full()
		switch field {
		case "maxRows":
			l.MaxRows = v
		case "maxNodes":
			l.MaxNodes = v
		case "maxEdges":
			l.MaxEdges = v
		case "maxDepth":
			l.MaxDepth = v
		case "maxProperties":
			l.MaxProperties = v
		case "maxBytes":
			l.MaxBytes = v
		}
		return l
	}
	errCode(t, meta([]TypeRowInput{scalar}, nil, nil), lim(set("maxRows", 0)), "ROW_LIMIT")
	errCode(t, meta([]TypeRowInput{scalar}, nil, nil), lim(set("maxNodes", 0)), "NODE_LIMIT")
	errCode(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_A", fieldName: "B", fieldType: "Z_B", internalType: "u", componentType: "S", dataType: "STRU"}),
		mkTypeRow(trOpts{typeName: "Z_B", fieldName: "A", fieldType: "CHAR", internalType: "C"}),
	}, nil, nil), lim(set("maxEdges", 0)), "EDGE_LIMIT")
	errCode(t, meta([]TypeRowInput{
		mkTypeRow(trOpts{typeName: "Z_A", fieldName: "B", fieldType: "Z_B", internalType: "u", componentType: "S", dataType: "STRU"}),
		mkTypeRow(trOpts{typeName: "Z_B", fieldName: "C", fieldType: "Z_C", internalType: "u", componentType: "S", dataType: "STRU"}),
		mkTypeRow(trOpts{typeName: "Z_C", fieldName: "A", fieldType: "CHAR", internalType: "C"}),
	}, nil, nil), &Options{RootTypeNames: []string{"Z_A"}, Limits: func() *Limits { l := set("maxDepth", 2); return &l }()}, "DEPTH_LIMIT")
	errCode(t, meta([]TypeRowInput{scalar}, nil, nil), lim(set("maxProperties", 1)), "PROPERTY_LIMIT")
	errCode(t, meta([]TypeRowInput{scalar}, nil, nil), lim(set("maxBytes", 1)), "BYTE_LIMIT")
	errCode(t, meta(nil, nil, nil), lim(set("maxRows", 100_001)), "INVALID_LIMIT")
}

func TestRejectsHostileText(t *testing.T) {
	// The getter/proxy/sparse cases have no Go analogue; the NUL-bearing field
	// name (INVALID_TEXT) is kept.
	hostile := mkTypeRow(trOpts{typeName: "Z_SAFE", fieldName: "backend-secret\x00", fieldType: "CHAR", internalType: "C"})
	errCode(t, meta([]TypeRowInput{hostile}, nil, nil), nil, "INVALID_TEXT")
}
