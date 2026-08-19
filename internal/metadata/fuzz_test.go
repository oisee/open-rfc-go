// SPDX-License-Identifier: Apache-2.0
//
// Go-specific fuzz target for the metadata normalizer. It asserts that
// Normalize never lets a non-*RecursiveMetadataError panic escape (the
// recover boundary must convert every rejection into a returned error). No
// upstream analogue. See docs/provenance.md.

package metadata

import "testing"

func FuzzNormalize(f *testing.F) {
	f.Add("Z_T", "F", "u", "C", "20260716010203")
	f.Add("", "", "", "", "")
	f.Fuzz(func(t *testing.T, typeName, fieldName, intType, exid, ts string) {
		fns := []FunctionRow{{FunctionName: "Z_F", BasxmlSupported: "", UDat: "20260716", UTime: "010203"}}
		params := []ParameterRowInput{{
			FuncName: "Z_F", ParamClass: "I", Parameter: "P", TabName: typeName, FieldName: "",
			Exid: exid, Position: "1", Offset: "0", IntLength: "8", Decimals: "0",
		}}
		in := Input{
			FunctionNames: &fns,
			DataTypesCont: []TypeRowInput{{
				TypeName: typeName, FieldName: fieldName, CompType: "E", FieldType: "X", DataType: "CHAR",
				TabLength: "000008", TabLengthUC: "000008", Description: "", Decimals: "000000",
				IntType: intType, Offset: "000000", OffsetUC: "000000", IntLen: "000008", IntLenUC: "000008",
				Timestamp: ts,
			}},
			IndirectTypes: nil,
			Parameters:    &params,
		}
		// Must return normally (graph or *RecursiveMetadataError); a non-RME
		// panic would fail the test via the recover re-panic in Normalize.
		_, _ = Normalize(in, nil)
	})
}
