// SPDX-License-Identifier: Apache-2.0
//
// Original conformance tests for RFC_METADATA_GET ported from open-rfc
// src/metadata/rfc-metadata-get.ts. Upstream's rfc-metadata-get.test.ts drives
// the same functions through decoded output objects; these Go tests state the
// same wire facts over map[string]any outputs: the hardcoded bootstrap, the
// invocation builders, and the function/structure/recursive/timestamp
// normalizers (including the C/N/D/T Unicode length halving). See
// docs/provenance.md.

package metadata

import (
	"reflect"
	"testing"
)

func TestBootstrapInterface(t *testing.T) {
	b := RfcMetadataGetBootstrapValue
	if b.Metadata.Name != "RFC_METADATA_GET" || b.Metadata.RemoteCall != "R" {
		t.Fatalf("metadata = %+v", b.Metadata)
	}
	if len(b.Metadata.Parameters) != 12 {
		t.Fatalf("params = %d", len(b.Metadata.Parameters))
	}
	for _, name := range []string{"RFCFUNCTIONNAME", "RFC_METADATA_PARAMS", "RFC_METADATA_DDIC", "RFC_DD_ERROR"} {
		if _, ok := b.Structures[name]; !ok {
			t.Fatalf("missing bootstrap structure %s", name)
		}
	}
	if b.Structures["RFC_METADATA_PARAMS"].ByteLength != 464 || len(b.Structures["RFC_METADATA_PARAMS"].Fields) != 13 {
		t.Fatalf("PARAMS structure = %+v", b.Structures["RFC_METADATA_PARAMS"])
	}
	ts := RfcMetadataGetTimestampBootstrapValue
	if ts.Metadata.Name != "RFC_METADATA_GET_TIMESTAMP" || len(ts.Metadata.Parameters) != 4 {
		t.Fatalf("timestamp bootstrap = %+v", ts.Metadata)
	}
}

func TestInvocationBuilders(t *testing.T) {
	fi, err := CreateFunctionInvocation("STFC_CONNECTION", "E")
	if err != nil {
		t.Fatal(err)
	}
	fn := fi.Input["FUNCTIONNAMES"].([]map[string]any)
	if len(fn) != 1 || fn[0]["FUNCTIONNAME"] != "STFC_CONNECTION" || fi.Input["DEEP"] != "X" {
		t.Fatalf("function invocation = %+v", fi.Input)
	}
	si, err := CreateStructureInvocation("SYST", "E")
	if err != nil {
		t.Fatal(err)
	}
	dt := si.Input["DATATYPES"].([]map[string]any)
	if len(dt) != 1 || dt[0]["TABNAME"] != "SYST" {
		t.Fatalf("structure invocation = %+v", si.Input)
	}
	if _, err := CreateFunctionInvocation("", "E"); err == nil {
		t.Fatal("empty name accepted")
	}
}

func paramRowMap(funcName, class, name, tab, field, exid string, pos, off, intlen, dec int, def, text, opt string) map[string]any {
	return map[string]any{
		"FUNCNAME": funcName, "PARAMCLASS": class, "PARAMETER": name, "TABNAME": tab, "FIELDNAME": field,
		"EXID": exid, "POSITION": int32(pos), "OFFSET": int32(off), "INTLENGTH": int32(intlen), "DECIMALS": int32(dec),
		"DEFAULT": def, "PARAMTEXT": text, "OPTIONAL": opt,
	}
}

func TestNormalizeFunctionResult(t *testing.T) {
	output := map[string]any{
		"FUNC_ERRORS": []map[string]any{},
		"DD_ERRORS":   []map[string]any{},
		"FUNCTIONNAMES": []map[string]any{
			{"FUNCTIONNAME": "STFC_CONNECTION", "BASXML_SUPPORTED": "X", "UDAT": "20260716", "UTIME": "010203"},
		},
		"PARAMETERS": []map[string]any{
			paramRowMap("STFC_CONNECTION", "I", "REQUTEXT", "", "", "C", 1, 0, 510, 0, "", "Request", ""),
			paramRowMap("STFC_CONNECTION", "E", "ECHOTEXT", "", "", "C", 2, 0, 510, 0, "", "", ""),
			paramRowMap("STFC_CONNECTION", "X", "SYSTEM_FAILURE", "", "", "C", 0, 0, 0, 0, "", "", ""),
		},
	}
	r, err := NormalizeFunctionResult("STFC_CONNECTION", output)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Value.RemoteBasxmlSupported || r.GenerationToken != "function:20260716:010203" {
		t.Fatalf("result = %+v", r)
	}
	if len(r.Value.Parameters) != 2 || r.Value.Parameters[0].ParameterName != "REQUTEXT" || r.Value.Parameters[0].InternalLength != 255 {
		t.Fatalf("params = %+v", r.Value.Parameters) // C length 510 bytes -> 255 chars
	}
	if len(r.Value.Exceptions) != 1 || r.Value.Exceptions[0] != "SYSTEM_FAILURE" {
		t.Fatalf("exceptions = %+v", r.Value.Exceptions)
	}
}

func TestNormalizeFunctionResultError(t *testing.T) {
	output := map[string]any{
		"FUNC_ERRORS":   []map[string]any{{"FUNCNAME": "ZBAD", "EXCEPTION": "FU_NOT_FOUND", "EXCEPTION_TEXT": ""}},
		"DD_ERRORS":     []map[string]any{},
		"FUNCTIONNAMES": []map[string]any{}, "PARAMETERS": []map[string]any{},
	}
	if _, err := NormalizeFunctionResult("ZBAD", output); err == nil {
		t.Fatal("expected error for FU_NOT_FOUND")
	}
}

func ddicRow(typeName, fieldName, comp, exid string, tablenUC, offUC, intlenUC, dec int, ts string) map[string]any {
	return map[string]any{
		"TYPENAME": typeName, "FIELDNAME": fieldName, "COMPTYPE": comp, "FIELDTYPE": "", "DATATYPE": "CHAR",
		"TABLENGTH": itoa(tablenUC), "TABLENGTH_UC": itoa(tablenUC), "DESCRIPTION": "", "DECIMALS": itoa(dec),
		"INTTYPE": exid, "OFFSET": itoa(offUC), "OFFSET_UC": itoa(offUC), "INTLEN": itoa(intlenUC), "INTLEN_UC": itoa(intlenUC),
		"TIMESTAMP": ts,
	}
}

func TestNormalizeStructureResult(t *testing.T) {
	ts := "20260716010203"
	output := map[string]any{
		"DD_ERRORS": []map[string]any{},
		"DATATYPESCONT": []map[string]any{
			ddicRow("ZS", "A", "E", "C", 12, 0, 8, 0, ts),
			ddicRow("ZS", "B", "E", "I", 12, 8, 4, 0, ts),
		},
	}
	r, err := NormalizeStructureResult("ZS", output)
	if err != nil {
		t.Fatal(err)
	}
	if r.Value.Name != "ZS" || r.Value.ByteLength != 12 || len(r.Value.Fields) != 2 || r.GenerationToken != "structure:"+ts {
		t.Fatalf("structure = %+v token %s", r.Value, r.GenerationToken)
	}
	if r.Value.Fields[1].FieldName != "B" || r.Value.Fields[1].Offset != 8 || r.Value.Fields[1].Exid != "I" {
		t.Fatalf("field B = %+v", r.Value.Fields[1])
	}
}

func TestNormalizeRecursiveFunctionResult(t *testing.T) {
	output := map[string]any{
		"FUNC_ERRORS": []map[string]any{},
		"DD_ERRORS":   []map[string]any{},
		"FUNCTIONNAMES": []map[string]any{
			{"FUNCTIONNAME": "STFC_CONNECTION", "BASXML_SUPPORTED": "", "UDAT": "20260716", "UTIME": "010203"},
		},
		"DATATYPESCONT": []map[string]any{},
		"INDIRECTTYPES": []map[string]any{},
		"PARAMETERS": []map[string]any{
			paramRowMap("STFC_CONNECTION", "I", "REQUTEXT", "", "", "C", 1, 0, 510, 0, "", "", ""),
		},
	}
	r, err := NormalizeRecursiveFunctionResult("STFC_CONNECTION", output)
	if err != nil {
		t.Fatal(err)
	}
	if r.GenerationToken != "function:20260716:010203" {
		t.Fatalf("token = %s", r.GenerationToken)
	}
	if r.Value.FunctionIdentity == nil || r.Value.FunctionIdentity.Name != "STFC_CONNECTION" || len(r.Value.Parameters) != 1 {
		t.Fatalf("graph = %+v", r.Value)
	}
}

func TestNormalizeTimestamps(t *testing.T) {
	inv, err := CreateTimestampInvocation([]string{"STFC_CONNECTION"}, []string{"SYST"})
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.FunctionNames) != 1 || len(inv.StructureNames) != 1 {
		t.Fatalf("invocation = %+v", inv)
	}
	output := map[string]any{
		"FUNCTION_TIMESTAMPS": []map[string]any{{"FUNCNAME": "STFC_CONNECTION", "UDAT": "20260716", "UTIME": "010203"}},
		"DDIC_TIMESTAMPS":     []map[string]any{{"TYPENAME": "SYST", "TIMESTAMP": "20260716010203"}},
		"FUNC_ERRORS":         []map[string]any{},
		"DD_ERRORS":           []map[string]any{},
	}
	batch, err := NormalizeTimestamps([]string{"STFC_CONNECTION"}, []string{"SYST"}, output)
	if err != nil {
		t.Fatal(err)
	}
	if batch.Functions["STFC_CONNECTION"].Token != "function:20260716:010203" {
		t.Fatalf("func ts = %+v", batch.Functions)
	}
	if batch.Structures["SYST"].Token != "structure:20260716010203" {
		t.Fatalf("struct ts = %+v", batch.Structures)
	}
	// Missing outcome fails closed.
	empty := map[string]any{"FUNCTION_TIMESTAMPS": []map[string]any{}, "DDIC_TIMESTAMPS": []map[string]any{}, "FUNC_ERRORS": []map[string]any{}, "DD_ERRORS": []map[string]any{}}
	if _, err := NormalizeTimestamps([]string{"STFC_CONNECTION"}, nil, empty); err == nil {
		t.Fatal("missing outcome accepted")
	}
}

func TestBootstrapStructuresMatchExpectedShape(t *testing.T) {
	want := map[string][2]int{ // name -> {byteLength, fieldCount}
		"RFCFUNCTIONNAME": {90, 4}, "RFC_MD_DDIC_NAME": {120, 2}, "RFC_METADATA_DDIC": {424, 15},
		"RFC_METADATA_DDIC_INDIRECT": {180, 3}, "RFC_FUNC_ERROR": {630, 3}, "RFC_DD_ERROR": {690, 4},
	}
	for name, ws := range want {
		def := RfcMetadataGetBootstrapValue.Structures[name]
		if int(def.ByteLength) != ws[0] || len(def.Fields) != ws[1] {
			t.Fatalf("%s = byteLen %d fields %d, want %v", name, def.ByteLength, len(def.Fields), ws)
		}
	}
	if !reflect.DeepEqual(RfcMetadataGetBootstrapValue.Metadata.Exceptions, []string{"INVALID_MODE", "INTERNAL_ERROR"}) {
		t.Fatalf("exceptions = %v", RfcMetadataGetBootstrapValue.Metadata.Exceptions)
	}
}
