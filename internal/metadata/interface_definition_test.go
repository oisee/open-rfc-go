// SPDX-License-Identifier: Apache-2.0
//
// Original conformance tests for the RFC_GET_FUNCTION_INTERFACE and
// RFC_GET_STRUCTURE_DEFINITION decoders ported from open-rfc
// src/metadata/rfc-function-interface.ts and rfc-structure-definition.ts.
// Upstream has no isolated test for these (they are exercised through
// rfc-metadata-get); these state their wire facts, including the >=402 / >=138
// row-width bound (recurring-bug-class fix). See docs/provenance.md.

package metadata

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
)

func u16m(s string) []byte { return classicMustChar(s, len([]rune(s))) }

func classicMustChar(s string, chars int) []byte {
	b, err := classicrfc.EncodeAbapChar(s, chars)
	if err != nil {
		panic(err)
	}
	return b
}

func fldm(tag cpic.Tag, v []byte) cpic.Field { return cpic.Field{Tag: uint16(tag), Value: v} }

func tableHeader(rowLen, rowCount uint32) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint32(h[0:], rowLen)
	binary.BigEndian.PutUint32(h[4:], rowCount)
	return h
}

func funintRow(class, param, exid string, intlen int32, optional string) []byte {
	row := make([]byte, classicrfc.RfcFunintUnicodeRowLength)
	off := 0
	putChar := func(s string, chars int) { copy(row[off:], classicMustChar(s, chars)); off += chars * 2 }
	putInt := func(v int32) { binary.LittleEndian.PutUint32(row[off:], uint32(v)); off += 4 }
	putChar(class, 1)
	putChar(param, 30)
	putChar("", 30) // TABNAME
	putChar("", 30) // FIELDNAME
	putChar(exid, 1)
	putInt(1) // POSITION
	putInt(0) // OFFSET
	putInt(intlen)
	putInt(0)       // DECIMALS
	putChar("", 21) // DEFAULT
	putChar("", 79) // PARAMTEXT
	putChar(optional, 1)
	return row
}

func TestDecodeFunctionInterfaceResult(t *testing.T) {
	fields := []cpic.Field{
		fldm(cpic.TagParameterName, u16m("REMOTE_BASXML_SUPPORTED")), fldm(cpic.TagParameterValue, classicMustChar("X", 1)),
		fldm(cpic.TagParameterName, u16m("REMOTE_CALL")), fldm(cpic.TagParameterValue, classicMustChar("R", 1)),
		fldm(cpic.TagParameterName, u16m("UPDATE_TASK")), fldm(cpic.TagParameterValue, classicMustChar("", 1)),
		fldm(cpic.TagTableName, u16m("PARAMS")), fldm(cpic.TagTableHeader, tableHeader(classicrfc.RfcFunintUnicodeRowLength, 2)),
		fldm(cpic.TagTableContent, funintRow("I", "REQUTEXT", "C", 255, "")),
		fldm(cpic.TagTableContent, funintRow("X", "SYSTEM_FAILURE", "C", 0, "")),
		fldm(cpic.TagTableName, u16m("RESUMABLE_EXCEPTIONS")), fldm(cpic.TagTableHeader, tableHeader(classicrfc.RfcFunintUnicodeRowLength, 0)),
		fldm(cpic.TagEnd, nil),
	}
	iface, err := DecodeRfcFunctionInterfaceResult("STFC_CONNECTION", fields)
	if err != nil {
		t.Fatal(err)
	}
	if iface.Name != "STFC_CONNECTION" || !iface.RemoteBasxmlSupported || iface.RemoteCall != "R" || iface.UpdateTask {
		t.Fatalf("iface = %+v", iface)
	}
	if len(iface.Parameters) != 1 || iface.Parameters[0].ParameterName != "REQUTEXT" || iface.Parameters[0].InternalLength != 255 {
		t.Fatalf("params = %+v", iface.Parameters)
	}
	if len(iface.Exceptions) != 1 || iface.Exceptions[0] != "SYSTEM_FAILURE" {
		t.Fatalf("exceptions = %+v", iface.Exceptions)
	}
}

func TestFunctionInterfaceRejectsNarrowParams(t *testing.T) {
	fields := []cpic.Field{
		fldm(cpic.TagParameterName, u16m("REMOTE_BASXML_SUPPORTED")), fldm(cpic.TagParameterValue, classicMustChar("", 1)),
		fldm(cpic.TagParameterName, u16m("REMOTE_CALL")), fldm(cpic.TagParameterValue, classicMustChar("", 1)),
		fldm(cpic.TagParameterName, u16m("UPDATE_TASK")), fldm(cpic.TagParameterValue, classicMustChar("", 1)),
		fldm(cpic.TagTableName, u16m("PARAMS")), fldm(cpic.TagTableHeader, tableHeader(400, 1)),
		fldm(cpic.TagTableContent, make([]byte, 400)),
		fldm(cpic.TagTableName, u16m("RESUMABLE_EXCEPTIONS")), fldm(cpic.TagTableHeader, tableHeader(classicrfc.RfcFunintUnicodeRowLength, 0)),
		fldm(cpic.TagEnd, nil),
	}
	if _, err := DecodeRfcFunctionInterfaceResult("F", fields); err == nil || !strings.Contains(err.Error(), "PARAMS row width is 400") {
		t.Fatalf("narrow params: %v", err)
	}
}

func fieldsRow(table, field string, position, offset, intlen int32) []byte {
	row := make([]byte, RfcFieldsUnicodeRowLength)
	off := 0
	putChar := func(s string, chars int) { copy(row[off:], classicMustChar(s, chars)); off += chars * 2 }
	putInt := func(v int32) { binary.LittleEndian.PutUint32(row[off:], uint32(v)); off += 4 }
	putChar(table, 30)
	putChar(field, 30)
	putInt(position)
	putInt(offset)
	putInt(intlen)
	putInt(0) // DECIMALS
	putChar("C", 1)
	return row
}

func TestDecodeStructureDefinitionResult(t *testing.T) {
	tablen := make([]byte, 4)
	binary.LittleEndian.PutUint32(tablen, 12)
	fields := []cpic.Field{
		fldm(cpic.TagParameterName, u16m("TABLENGTH")), fldm(cpic.TagParameterValue, tablen),
		fldm(cpic.TagTableName, u16m("FIELDS")), fldm(cpic.TagTableHeader, tableHeader(RfcFieldsUnicodeRowLength, 2)),
		fldm(cpic.TagTableContent, fieldsRow("Z_S", "A", 1, 0, 8)),
		fldm(cpic.TagTableContent, fieldsRow("Z_S", "B", 2, 8, 4)),
		fldm(cpic.TagEnd, nil),
	}
	def, err := DecodeRfcStructureDefinitionResult("Z_S", fields)
	if err != nil {
		t.Fatal(err)
	}
	if def.Name != "Z_S" || def.ByteLength != 12 || len(def.Fields) != 2 {
		t.Fatalf("def = %+v", def)
	}
	if def.Fields[1].FieldName != "B" || def.Fields[1].Offset != 8 || def.Fields[1].InternalLength != 4 {
		t.Fatalf("field B = %+v", def.Fields[1])
	}
}

func TestStructureDefinitionRejects(t *testing.T) {
	tablen := make([]byte, 4)
	binary.LittleEndian.PutUint32(tablen, 4)
	// Field ends beyond structure length.
	fields := []cpic.Field{
		fldm(cpic.TagParameterName, u16m("TABLENGTH")), fldm(cpic.TagParameterValue, tablen),
		fldm(cpic.TagTableName, u16m("FIELDS")), fldm(cpic.TagTableHeader, tableHeader(RfcFieldsUnicodeRowLength, 1)),
		fldm(cpic.TagTableContent, fieldsRow("Z_S", "A", 1, 0, 8)),
		fldm(cpic.TagEnd, nil),
	}
	if _, err := DecodeRfcStructureDefinitionResult("Z_S", fields); err == nil || !strings.Contains(err.Error(), "beyond structure length") {
		t.Fatalf("overflow: %v", err)
	}
}

func TestBuildRequests(t *testing.T) {
	fi, err := BuildRfcGetFunctionInterfaceRequest("STFC_CONNECTION")
	if err != nil || len(fi) == 0 {
		t.Fatalf("build fi: %v", err)
	}
	sd, err := BuildRfcGetStructureDefinitionRequest("Z_S")
	if err != nil || len(sd) == 0 {
		t.Fatalf("build sd: %v", err)
	}
	// The request must be inspectable as a valid compact CPIC CUT frame.
	if _, err := cpic.InspectRequestAppcFraming(fi); err != nil {
		t.Fatalf("fi framing: %v", err)
	}
}
