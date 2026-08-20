// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/metadata/rfc-structure-definition.ts at commit
// 847036d, Copyright 2026 Marian Zeis, licensed under the Apache License,
// Version 2.0. Modified by open-rfc-go contributors: rewritten in Go. Thrown
// errors → returned wrapped sentinels; the FIELDS row width is bounded below by
// the 138-byte stable prefix (recurring-bug-class fix). See docs/provenance.md.

package metadata

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/wire"
)

// RfcFieldsUnicodeRowLength is the stable prefix width of a Unicode RFC_FIELDS
// row; later releases append fields.
const RfcFieldsUnicodeRowLength = 138

const maxRfcStructureFields = 20_000

// ErrStructureDefinition reports a malformed RFC_GET_STRUCTURE_DEFINITION result.
var ErrStructureDefinition = errors.New("metadata: invalid structure definition")

// RfcStructureField is one decoded RFC_FIELDS row (shared type).
type RfcStructureField = rfctypes.RfcStructureField

// RfcStructureDefinition is a decoded RFC_GET_STRUCTURE_DEFINITION result
// (shared type).
type RfcStructureDefinition = rfctypes.RfcStructureDefinition

// BuildRfcGetStructureDefinitionRequest builds the structure bootstrap call.
func BuildRfcGetStructureDefinitionRequest(structureName string) ([]byte, error) {
	tabname, err := classicrfc.EncodeAbapChar(structureName, 30)
	if err != nil {
		return nil, err
	}
	return cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName:     "RFC_GET_STRUCTURE_DEFINITION",
		RequestedOutputs: []string{"TABLENGTH", "FIELDS"},
		Imports:          []cpic.NamedValue{{Name: "TABNAME", Value: tabname}},
	})
}

// DecodeRfcFieldsRow decodes one Unicode RFC_FIELDS row from its stable prefix.
func DecodeRfcFieldsRow(value []byte) (RfcStructureField, error) {
	var zero RfcStructureField
	if len(value) < RfcFieldsUnicodeRowLength {
		return zero, fmt.Errorf("%w: Unicode RFC_FIELDS row must contain at least %d bytes; received %d", ErrStructureDefinition, RfcFieldsUnicodeRowLength, len(value))
	}
	r := wire.NewReader(value[:RfcFieldsUnicodeRowLength], "RFC_FIELDS row")
	var field RfcStructureField
	var err error
	readChar := func(n, chars int, dst *string) {
		if err != nil {
			return
		}
		var b []byte
		b, err = r.ReadBytes(n, "field")
		if err != nil {
			return
		}
		*dst, err = classicrfc.DecodeAbapChar(b, chars)
	}
	readInt := func(dst *int32) {
		if err != nil {
			return
		}
		*dst, err = r.ReadInt32LE("field")
	}
	readChar(60, 30, &field.TableName)
	readChar(60, 30, &field.FieldName)
	readInt(&field.Position)
	readInt(&field.Offset)
	readInt(&field.InternalLength)
	readInt(&field.Decimals)
	readChar(2, 1, &field.Exid)
	if err != nil {
		return zero, err
	}
	if field.TableName == "" || field.FieldName == "" {
		return zero, fmt.Errorf("%w: RFC_FIELDS row contains an empty table or field name", ErrStructureDefinition)
	}
	if field.Position < 1 || field.Offset < 0 || field.InternalLength < 0 || field.Decimals < 0 {
		return zero, fmt.Errorf("%w: RFC_FIELDS row contains a negative or invalid numeric property", ErrStructureDefinition)
	}
	return field, nil
}

// DecodeRfcStructureDefinitionResult normalizes and validates the output.
func DecodeRfcStructureDefinitionResult(structureName string, fields []cpic.Field) (RfcStructureDefinition, error) {
	var zero RfcStructureDefinition
	for index := 0; index < len(fields); index++ {
		if fields[index].Tag != uint16(cpic.TagTableName) {
			continue
		}
		name, derr := classicrfc.DecodeAbapChar(fields[index].Value)
		if derr != nil || name != "FIELDS" {
			continue
		}
		if index+1 >= len(fields) || fields[index+1].Tag != uint16(cpic.TagTableHeader) {
			continue
		}
		header, herr := classicrfc.DecodeRfcTableHeader(fields[index+1].Value)
		if herr != nil {
			return zero, herr
		}
		if header.RowCount > maxRfcStructureFields {
			return zero, fmt.Errorf("%w: FIELDS must contain at most %d rows", ErrStructureDefinition, maxRfcStructureFields)
		}
	}
	result, err := classicrfc.DecodeResult(fields)
	if err != nil {
		return zero, err
	}
	lengthValue, err := requiredStructScalar(result.Scalars, "TABLENGTH")
	if err != nil {
		return zero, err
	}
	if len(lengthValue) != 4 {
		return zero, fmt.Errorf("%w: TABLENGTH must be INT4", ErrStructureDefinition)
	}
	byteLength := int32(binary.LittleEndian.Uint32(lengthValue))
	if byteLength < 0 {
		return zero, fmt.Errorf("%w: returned negative TABLENGTH", ErrStructureDefinition)
	}
	var fieldTable *classicrfc.Table
	for i := range result.Tables {
		if result.Tables[i].Name == "FIELDS" {
			fieldTable = &result.Tables[i]
		}
	}
	if fieldTable == nil {
		return zero, fmt.Errorf("%w: response lacks FIELDS table", ErrStructureDefinition)
	}
	if fieldTable.RowByteLength < RfcFieldsUnicodeRowLength {
		return zero, fmt.Errorf("%w: FIELDS row width is %d; expected at least %d", ErrStructureDefinition, fieldTable.RowByteLength, RfcFieldsUnicodeRowLength)
	}

	decoded := make([]RfcStructureField, 0, len(fieldTable.Rows))
	names := map[string]bool{}
	var previousEnd int32
	for index, row := range fieldTable.Rows {
		field, err := DecodeRfcFieldsRow(row)
		if err != nil {
			return zero, err
		}
		expectedPosition := int32(index + 1)
		if field.Position != expectedPosition {
			return zero, fmt.Errorf("%w: RFC_FIELDS %s has position %d; expected %d", ErrStructureDefinition, field.FieldName, field.Position, expectedPosition)
		}
		if field.TableName != structureName {
			return zero, fmt.Errorf("%w: RFC_FIELDS %s belongs to %s; expected %s", ErrStructureDefinition, field.FieldName, field.TableName, structureName)
		}
		if names[field.FieldName] {
			return zero, fmt.Errorf("%w: RFC_FIELDS contains duplicate field %s", ErrStructureDefinition, field.FieldName)
		}
		if field.Offset < previousEnd {
			return zero, fmt.Errorf("%w: RFC_FIELDS %s overlaps its preceding field", ErrStructureDefinition, field.FieldName)
		}
		end := field.Offset + field.InternalLength
		if end > byteLength {
			return zero, fmt.Errorf("%w: RFC_FIELDS %s ends at %d beyond structure length %d", ErrStructureDefinition, field.FieldName, end, byteLength)
		}
		names[field.FieldName] = true
		previousEnd = end
		decoded = append(decoded, field)
	}
	return RfcStructureDefinition{Name: structureName, ByteLength: byteLength, Fields: decoded}, nil
}

func requiredStructScalar(scalars []classicrfc.Scalar, name string) ([]byte, error) {
	for _, s := range scalars {
		if s.Name == name {
			return s.Value, nil
		}
	}
	return nil, fmt.Errorf("%w: response lacks scalar %s", ErrStructureDefinition, name)
}

// RowStructureName inspects a RFC_GET_STRUCTURE_DEFINITION result and returns the
// structure name the FIELDS rows actually belong to. When the queried name is a
// TABLE TYPE, RFC_FIELDS returns the fields of its row (line) structure — whose
// TableName differs from the queried table-type name. The caller can then resolve
// that row structure instead of the (unresolvable) table type. Returns "" if the
// FIELDS all belong to the queried name (i.e. it is an ordinary structure).
func RowStructureName(queried string, fields []cpic.Field) (string, error) {
	result, err := classicrfc.DecodeResult(fields)
	if err != nil {
		return "", err
	}
	for _, t := range result.Tables {
		if t.Name != "FIELDS" {
			continue
		}
		for _, row := range t.Rows {
			f, err := DecodeRfcFieldsRow(row)
			if err != nil {
				return "", err
			}
			if f.TableName != "" && f.TableName != queried {
				return f.TableName, nil
			}
		}
	}
	return "", nil
}
