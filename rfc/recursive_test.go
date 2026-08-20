// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// The synthetic type closure below mirrors the live shape that motivated the
// recursive path (TPDAPI_TEST_DEBUGGER → TPDAPI_TAB_MSG → TPDAPI_STR_MSG with
// STRU and TTYP components): a structure with a nested structure and a nested
// table, plus a parameter that is a table type in its own right.

const nestTestFunction = "Z_NEST_TEST"

type nestRow struct {
	typeName, fieldName, compType, fieldType, dataType, intType string
	nucOffset, nucLength, ucOffset, ucLength                    int
	nucTotal, ucTotal                                           int
}

func d6(v int) string { return fmt.Sprintf("%06d", v) }

func (r nestRow) row() metadata.TypeRowInput {
	return metadata.TypeRowInput{
		TypeName: r.typeName, FieldName: r.fieldName, CompType: r.compType, FieldType: r.fieldType,
		DataType: r.dataType, TabLength: d6(r.nucTotal), TabLengthUC: d6(r.ucTotal),
		Description: "", Decimals: d6(0), IntType: r.intType,
		Offset: d6(r.nucOffset), OffsetUC: d6(r.ucOffset),
		IntLen: d6(r.nucLength), IntLenUC: d6(r.ucLength), Timestamp: "20260716010203",
	}
}

func nestParamRow(name, class, tabName, exid string, position int) metadata.ParameterRowInput {
	return metadata.ParameterRowInput{
		FuncName: nestTestFunction, ParamClass: class, Parameter: name, TabName: tabName,
		FieldName: "", Exid: exid, Position: fmt.Sprint(position), Offset: "0",
		IntLength: "8", Decimals: "0", Default: "", ParamText: "", Optional: "",
	}
}

// nestGraph builds a normalized graph for Z_NEST_TEST:
//
//	E_STRUCT (EXID v) → Z_PARENT { NAME:CHAR, CHILD:Z_CHILD (STRU), ROWS:Z_TAB (TTYP) }
//	E_TABLE  (EXID h) → Z_TAB (table of Z_ROW { ID:CHAR, N:INT4 })
//	I_TEXT   (EXID C) → scalar
func nestGraph(t *testing.T) metadata.Graph {
	t.Helper()
	rows := []metadata.TypeRowInput{
		nestRow{typeName: "Z_PARENT", fieldName: "NAME", compType: "E", fieldType: "CHAR4", dataType: "CHAR", intType: "C", nucOffset: 0, nucLength: 4, ucOffset: 0, ucLength: 8, nucTotal: 16, ucTotal: 24}.row(),
		nestRow{typeName: "Z_PARENT", fieldName: "CHILD", compType: "S", fieldType: "Z_CHILD", dataType: "STRU", intType: "u", nucOffset: 4, nucLength: 4, ucOffset: 8, ucLength: 8, nucTotal: 16, ucTotal: 24}.row(),
		nestRow{typeName: "Z_PARENT", fieldName: "ROWS", compType: "L", fieldType: "Z_TAB", dataType: "TTYP", intType: "h", nucOffset: 8, nucLength: 8, ucOffset: 16, ucLength: 8, nucTotal: 16, ucTotal: 24}.row(),
		nestRow{typeName: "Z_CHILD", fieldName: "TEXT", compType: "E", fieldType: "CHAR4", dataType: "CHAR", intType: "C", nucOffset: 0, nucLength: 4, ucOffset: 0, ucLength: 8, nucTotal: 4, ucTotal: 8}.row(),
		nestRow{typeName: "Z_TAB", fieldName: "", compType: "L", fieldType: "Z_ROW", dataType: "TTYP", intType: "v", nucOffset: 0, nucLength: 8, ucOffset: 0, ucLength: 12, nucTotal: 8, ucTotal: 12}.row(),
		nestRow{typeName: "Z_ROW", fieldName: "ID", compType: "E", fieldType: "CHAR4", dataType: "CHAR", intType: "C", nucOffset: 0, nucLength: 4, ucOffset: 0, ucLength: 8, nucTotal: 8, ucTotal: 12}.row(),
		nestRow{typeName: "Z_ROW", fieldName: "N", compType: "E", fieldType: "INT4", dataType: "INT4", intType: "I", nucOffset: 4, nucLength: 4, ucOffset: 8, ucLength: 4, nucTotal: 8, ucTotal: 12}.row(),
	}
	params := []metadata.ParameterRowInput{
		nestParamRow("E_STRUCT", "E", "Z_PARENT", "v", 1),
		nestParamRow("E_TABLE", "E", "Z_TAB", "h", 2),
		nestParamRow("I_TEXT", "I", "", "C", 3),
	}
	fns := []metadata.FunctionRow{{FunctionName: nestTestFunction, BasxmlSupported: "", UDat: "20260716", UTime: "010203"}}
	g, err := metadata.Normalize(metadata.Input{FunctionNames: &fns, DataTypesCont: rows, Parameters: &params}, nil)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	return g
}

func nestParam(name, class, tabName, exid string) classicrfc.FunintParameter {
	return classicrfc.FunintParameter{
		ParameterClass: class, ParameterName: name, TableName: tabName, Exid: exid, InternalLength: 8,
	}
}

func nestPlan(t *testing.T, g metadata.Graph, params ...classicrfc.FunintParameter) *layoutPlan {
	t.Helper()
	plan := &layoutPlan{graph: g, recursive: map[string]xrfc.ResolvedParameter{}}
	for _, p := range params {
		rp, needed, err := xrfc.ResolveParameter(g, p)
		if err != nil {
			t.Fatalf("ResolveParameter %s: %v", p.ParameterName, err)
		}
		if !needed {
			t.Fatalf("ResolveParameter %s: want the recursive codec, got the flat path", p.ParameterName)
		}
		plan.recursive[p.ParameterName] = rp
	}
	return plan
}

func TestResolveParameterSkipsScalars(t *testing.T) {
	g := nestGraph(t)
	_, needed, err := xrfc.ResolveParameter(g, nestParam("I_TEXT", "I", "", "C"))
	if err != nil {
		t.Fatalf("ResolveParameter: %v", err)
	}
	if needed {
		t.Fatal("a CHAR scalar must stay on the flat path")
	}
}

// TestRecursiveStructureRoundTrip covers a nested structure: a structure whose
// components are themselves a structure and a table.
func TestRecursiveStructureRoundTrip(t *testing.T) {
	g := nestGraph(t)
	p := nestParam("E_STRUCT", "E", "Z_PARENT", "v")
	plan := nestPlan(t, g, p)
	rp, _ := plan.lookup("E_STRUCT")

	// JSON-native input, as a CLI or MCP tool call would deliver it.
	var in any
	if err := json.Unmarshal([]byte(`{"NAME":"abcd","CHILD":{"TEXT":"xy"},"ROWS":[{"ID":"r1","N":7},{"ID":"r2","N":-3}]}`), &in); err != nil {
		t.Fatal(err)
	}
	value, err := coerceGraphParam(g, rp, in)
	if err != nil {
		t.Fatalf("coerceGraphParam: %v", err)
	}
	encoded, err := xrfc.EncodeRecursiveParameter(p, g, value, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("EncodeRecursiveParameter: %v", err)
	}
	name, err := xrfc.DecodeRecursiveParameterName(encoded, xrfc.RecursiveLimits{})
	if err != nil || name != "E_STRUCT" {
		t.Fatalf("DecodeRecursiveParameterName = %q, %v", name, err)
	}
	decoded, err := xrfc.DecodeRecursiveParameter(p, g, encoded, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("DecodeRecursiveParameter: %v", err)
	}
	got, ok := normalizeGraphValue(decoded).(map[string]any)
	if !ok {
		t.Fatalf("want map[string]any, got %T", decoded)
	}
	if got["NAME"] != "abcd" {
		t.Errorf("NAME = %#v", got["NAME"])
	}
	child, ok := got["CHILD"].(map[string]any)
	if !ok {
		t.Fatalf("CHILD is %T, want a nested object", got["CHILD"])
	}
	if child["TEXT"] != "xy" {
		t.Errorf("CHILD.TEXT = %#v", child["TEXT"])
	}
	rows, ok := got["ROWS"].([]map[string]any)
	if !ok {
		t.Fatalf("ROWS is %T, want a nested table", got["ROWS"])
	}
	if len(rows) != 2 || rows[0]["ID"] != "r1" || rows[1]["ID"] != "r2" {
		t.Fatalf("ROWS = %#v", rows)
	}
	if !reflect.DeepEqual(rows[0]["N"], int32(7)) || !reflect.DeepEqual(rows[1]["N"], int32(-3)) {
		t.Errorf("ROWS[].N = %#v, %#v", rows[0]["N"], rows[1]["N"])
	}
}

// TestRecursiveTableRoundTrip covers a table-typed parameter (EXID 'h') — the
// shape of TPDAPI_TEST_DEBUGGER's E_TAB_MSG.
func TestRecursiveTableRoundTrip(t *testing.T) {
	g := nestGraph(t)
	p := nestParam("E_TABLE", "E", "Z_TAB", "h")
	plan := nestPlan(t, g, p)
	rp, _ := plan.lookup("E_TABLE")
	if rp.Kind != xrfc.KindTable {
		t.Fatalf("kind = %v, want table", rp.Kind)
	}

	value, err := coerceGraphParam(g, rp, []map[string]any{{"ID": "a", "N": json.Number("1")}, {"ID": "b", "N": 2}})
	if err != nil {
		t.Fatalf("coerceGraphParam: %v", err)
	}
	encoded, err := xrfc.EncodeRecursiveParameter(p, g, value, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("EncodeRecursiveParameter: %v", err)
	}
	decoded, err := xrfc.DecodeRecursiveParameter(p, g, encoded, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("DecodeRecursiveParameter: %v", err)
	}
	rows, ok := normalizeGraphValue(decoded).([]map[string]any)
	if !ok {
		t.Fatalf("want []map[string]any, got %T", decoded)
	}
	want := []map[string]any{{"ID": "a", "N": int32(1)}, {"ID": "b", "N": int32(2)}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %#v, want %#v", rows, want)
	}
}

// TestRecursiveEmptyTableRoundTrip pins the empty case, which is what a live
// export of a nested table most often carries.
func TestRecursiveEmptyTableRoundTrip(t *testing.T) {
	g := nestGraph(t)
	p := nestParam("E_TABLE", "E", "Z_TAB", "h")
	plan := nestPlan(t, g, p)
	rp, _ := plan.lookup("E_TABLE")
	value, err := coerceGraphParam(g, rp, []any{})
	if err != nil {
		t.Fatalf("coerceGraphParam: %v", err)
	}
	encoded, err := xrfc.EncodeRecursiveParameter(p, g, value, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("EncodeRecursiveParameter: %v", err)
	}
	decoded, err := xrfc.DecodeRecursiveParameter(p, g, encoded, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("DecodeRecursiveParameter: %v", err)
	}
	rows, ok := normalizeGraphValue(decoded).([]map[string]any)
	if !ok || len(rows) != 0 {
		t.Fatalf("want an empty []map[string]any, got %#v", decoded)
	}
}

func TestCoerceGraphParamRejectsBadShape(t *testing.T) {
	g := nestGraph(t)
	p := nestParam("E_STRUCT", "E", "Z_PARENT", "v")
	plan := nestPlan(t, g, p)
	rp, _ := plan.lookup("E_STRUCT")
	if _, err := coerceGraphParam(g, rp, []any{}); err == nil {
		t.Fatal("a structure parameter must reject an array")
	}
	if _, err := coerceGraphParam(g, rp, map[string]any{"ROWS": "not an array"}); err == nil {
		t.Fatal("a nested table must reject a string")
	}
}

// TestGraphParamSchema pins the describe contract: a nested structure renders
// as a nested JSON-Schema object, a nested table as an array.
func TestGraphParamSchema(t *testing.T) {
	g := nestGraph(t)
	plan := nestPlan(t, g,
		nestParam("E_STRUCT", "E", "Z_PARENT", "v"),
		nestParam("E_TABLE", "E", "Z_TAB", "h"),
	)

	rp, _ := plan.lookup("E_STRUCT")
	sc := graphParamSchema(g, rp)
	if sc["type"] != "object" {
		t.Fatalf("E_STRUCT type = %v", sc["type"])
	}
	props := sc["properties"].(map[string]any)
	child := props["CHILD"].(map[string]any)
	if child["type"] != "object" || child["description"] != "Z_CHILD" {
		t.Errorf("CHILD schema = %#v", child)
	}
	if _, ok := child["properties"].(map[string]any)["TEXT"]; !ok {
		t.Errorf("CHILD.TEXT is missing from %#v", child)
	}
	rowsSchema := props["ROWS"].(map[string]any)
	if rowsSchema["type"] != "array" {
		t.Fatalf("ROWS schema = %#v", rowsSchema)
	}
	item := rowsSchema["items"].(map[string]any)
	if item["type"] != "object" || item["description"] != "Z_ROW" {
		t.Errorf("ROWS.items = %#v", item)
	}
	if n := item["properties"].(map[string]any)["N"].(map[string]any); n["type"] != "integer" {
		t.Errorf("Z_ROW.N schema = %#v", n)
	}
	if name := props["NAME"].(map[string]any); name["type"] != "string" || name["maxLength"] != 4 {
		t.Errorf("NAME schema = %#v", name)
	}

	rt, _ := plan.lookup("E_TABLE")
	ts := graphParamSchema(g, rt)
	if ts["type"] != "array" || ts["items"].(map[string]any)["description"] != "Z_ROW" {
		t.Fatalf("E_TABLE schema = %#v", ts)
	}
}

func TestLayoutPlanLookupOnNilPlan(t *testing.T) {
	var plan *layoutPlan
	if _, ok := plan.lookup("ANY"); ok {
		t.Fatal("a nil plan must report no recursive parameters")
	}
}

func TestDefHasNestedField(t *testing.T) {
	flat := rfctypes.RfcStructureDefinition{Name: "F", ByteLength: 4, Fields: []rfctypes.RfcStructureField{
		{TableName: "F", FieldName: "A", Position: 1, Offset: 0, InternalLength: 4, Exid: "C"},
	}}
	if defHasNestedField(flat) {
		t.Error("a flat layout must not be reported as nested")
	}
	nested := flat
	nested.Fields = append([]rfctypes.RfcStructureField{}, flat.Fields...)
	nested.Fields = append(nested.Fields, rfctypes.RfcStructureField{TableName: "F", FieldName: "T", Position: 2, Offset: 4, InternalLength: 8, Exid: "h"})
	if !defHasNestedField(nested) {
		t.Error("a layout with an EXID 'h' component must be reported as nested")
	}
}

// TestPadTrailingFill pins the DDIC-alignment tolerance that lets
// RFC_METADATA_PARAMS (declared 464, transferred 462) decode.
func TestPadTrailingFill(t *testing.T) {
	def := rfctypes.RfcStructureDefinition{Name: "S", ByteLength: 8, Fields: []rfctypes.RfcStructureField{
		{TableName: "S", FieldName: "A", Position: 1, Offset: 0, InternalLength: 6, Exid: "C"},
	}}
	if got := padTrailingFill(def, make([]byte, 6)); len(got) != 8 {
		t.Errorf("a row covering every field must be padded to the declared length, got %d", len(got))
	}
	if got := padTrailingFill(def, make([]byte, 4)); len(got) != 4 {
		t.Errorf("a truncated row must be left alone, got %d", len(got))
	}
	if got := padTrailingFill(def, make([]byte, 8)); len(got) != 8 {
		t.Errorf("a full row must be left alone, got %d", len(got))
	}
}
