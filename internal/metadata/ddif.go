// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/metadata/ddif-fieldinfo.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown errors →
// returned wrapped sentinels; intrinsic-geometry helpers → slices. The DFIES
// row is bounded below by its 1074-byte stable prefix and appended fields are
// ignored (recurring-bug-class fix). See docs/provenance.md.

package metadata

import (
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/structure"
)

const (
	dfiesMinimumUnicodeRowLength = 1_074
	x030lMinimumUnicodeLength    = 249
	maxDdicStructureFields       = 9_999
)

// ErrDdif reports a malformed DDIF_FIELDINFO_GET response.
var ErrDdif = errors.New("metadata: invalid DDIF_FIELDINFO_GET result")

var (
	ddifNameControl = regexp.MustCompile(`[\x00-\x1f\x7f]`)
	printableASCII1 = regexp.MustCompile(`^[\x20-\x7e]$`)
	numcDigits      = regexp.MustCompile(`^\d+$`)
)

func ddifMetadataName(v, path string) (string, error) {
	if len(v) < 1 || len([]rune(v)) > 30 || ddifNameControl.MatchString(v) {
		return "", fmt.Errorf("%w: %s must contain 1..30 characters without controls", ErrDdif, path)
	}
	return v, nil
}

func sapLanguage(v string) (string, error) {
	if !printableASCII1.MatchString(v) {
		return "", fmt.Errorf("%w: DDIF language must be one printable ASCII character", ErrDdif)
	}
	return v, nil
}

// BuildDdIfFieldInfoGetRequest builds the Note 460089 classic DDIC lookup.
func BuildDdIfFieldInfoGetRequest(structureName, language string) ([]byte, error) {
	if language == "" {
		language = "E"
	}
	name, err := ddifMetadataName(structureName, "structureName")
	if err != nil {
		return nil, err
	}
	langu, err := sapLanguage(language)
	if err != nil {
		return nil, err
	}
	tabname, err := classicrfc.EncodeAbapChar(name, 30)
	if err != nil {
		return nil, err
	}
	languVal, err := classicrfc.EncodeAbapChar(langu, 1)
	if err != nil {
		return nil, err
	}
	allTypes, err := classicrfc.EncodeAbapChar("X", 1)
	if err != nil {
		return nil, err
	}
	return cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName:     "DDIF_FIELDINFO_GET",
		RequestedOutputs: []string{"DDOBJTYPE", "X030L_WA", "DFIES_TAB"},
		Imports: []cpic.NamedValue{
			{Name: "TABNAME", Value: tabname},
			{Name: "LANGU", Value: languVal},
			{Name: "ALL_TYPES", Value: allTypes},
			{Name: "UCLEN", Value: []byte{2}},
		},
	})
}

func ddifRequiredScalar(result classicrfc.Result, name string) ([]byte, error) {
	for _, s := range result.Scalars {
		if s.Name == name {
			return s.Value, nil
		}
	}
	return nil, fmt.Errorf("%w: response lacks scalar %s", ErrDdif, name)
}

func fieldText(row []byte, offset, byteLength, characterLength int) (string, error) {
	return classicrfc.DecodeAbapChar(row[offset:offset+byteLength], characterLength)
}

func numc(row []byte, offset, byteLength int, path string) (int, error) {
	v, err := fieldText(row, offset, byteLength, byteLength/2)
	if err != nil {
		return 0, err
	}
	if !numcDigits.MatchString(v) {
		return 0, fmt.Errorf("%w: %s must contain NUMC digits", ErrDdif, path)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%w: %s exceeds the safe integer range", ErrDdif, path)
	}
	return n, nil
}

type decodedDfiesField struct {
	rfctypes.RfcStructureField
	componentType string
}

// DecodeDdIfDfiesRow decodes the stable DFIES prefix used by 7.50 and 7.58.
func DecodeDdIfDfiesRow(value []byte) (decodedDfiesField, error) {
	var zero decodedDfiesField
	if len(value) < dfiesMinimumUnicodeRowLength || len(value)&1 != 0 {
		return zero, fmt.Errorf("%w: Unicode DFIES row must contain at least %d even bytes; received %d", ErrDdif, dfiesMinimumUnicodeRowLength, len(value))
	}
	row := value[:dfiesMinimumUnicodeRowLength]
	tableName, err := fieldText(row, 0, 60, 30)
	if err != nil {
		return zero, err
	}
	fieldName, err := fieldText(row, 60, 60, 30)
	if err != nil {
		return zero, err
	}
	position, err := numc(row, 122, 8, "DFIES.POSITION")
	if err != nil {
		return zero, err
	}
	offset, err := numc(row, 130, 12, "DFIES.OFFSET")
	if err != nil {
		return zero, err
	}
	internalLength, err := numc(row, 334, 12, "DFIES.INTLEN")
	if err != nil {
		return zero, err
	}
	decimals, err := numc(row, 358, 12, "DFIES.DECIMALS")
	if err != nil {
		return zero, err
	}
	exid, err := fieldText(row, 378, 2, 1)
	if err != nil {
		return zero, err
	}
	componentType, err := fieldText(row, 1_072, 2, 1)
	if err != nil {
		return zero, err
	}
	if len(tableName) == 0 || len(fieldName) == 0 || len(exid) != 1 {
		return zero, fmt.Errorf("%w: DFIES row contains an empty table, field, or INTTYPE", ErrDdif)
	}
	if position < 1 {
		return zero, fmt.Errorf("%w: DFIES row contains an invalid position", ErrDdif)
	}
	return decodedDfiesField{
		RfcStructureField: rfctypes.RfcStructureField{
			TableName: tableName, FieldName: fieldName, Position: int32(position),
			Offset: int32(offset), InternalLength: int32(internalLength), Decimals: int32(decimals), Exid: exid,
		},
		componentType: componentType,
	}, nil
}

func x030lGeometry(value []byte, structureName string) (byteLength, fieldCount int, err error) {
	if len(value) < x030lMinimumUnicodeLength {
		return 0, 0, fmt.Errorf("%w: X030L_WA must contain at least %d bytes", ErrDdif, x030lMinimumUnicodeLength)
	}
	returnedName, err := fieldText(value, 0, 60, 30)
	if err != nil {
		return 0, 0, err
	}
	if returnedName != structureName {
		return 0, 0, fmt.Errorf("%w: X030L_WA belongs to %s; expected %s", ErrDdif, returnedName, structureName)
	}
	fieldCount = int(binary.BigEndian.Uint16(value[162:]))
	byteLength = int(binary.BigEndian.Uint32(value[164:]))
	tableType, err := fieldText(value, 172, 2, 1)
	if err != nil {
		return 0, 0, err
	}
	unicodeCharacterBytes := value[248]
	if unicodeCharacterBytes != 2 {
		return 0, 0, fmt.Errorf("%w: selected unsupported Unicode width %d", ErrDdif, unicodeCharacterBytes)
	}
	if fieldCount > maxDdicStructureFields {
		return 0, 0, fmt.Errorf("%w: field count exceeds %d", ErrDdif, maxDdicStructureFields)
	}
	if tableType == "L" {
		return 0, 0, fmt.Errorf("%w: returned a table/vector type; a flat structure was required", ErrDdif)
	}
	return byteLength, fieldCount, nil
}

// DecodeDdIfFieldInfoGetResult normalizes the response into a structure codec
// descriptor.
func DecodeDdIfFieldInfoGetResult(structureName string, fields []cpic.Field) (rfctypes.RfcStructureDefinition, error) {
	var zero rfctypes.RfcStructureDefinition
	name, err := ddifMetadataName(structureName, "structureName")
	if err != nil {
		return zero, err
	}
	result, err := classicrfc.DecodeResult(fields)
	if err != nil {
		return zero, err
	}
	objTypeVal, err := ddifRequiredScalar(result, "DDOBJTYPE")
	if err != nil {
		return zero, err
	}
	objectKind, err := classicrfc.DecodeAbapChar(objTypeVal)
	if err != nil {
		return zero, err
	}
	if len(objectKind) == 0 {
		return zero, fmt.Errorf("%w: returned an initial DDOBJTYPE", ErrDdif)
	}
	if objectKind == "DTEL" || objectKind == "TTYP" {
		return zero, fmt.Errorf("%w: returned unsupported DDIC object kind %s", ErrDdif, objectKind)
	}
	x030l, err := ddifRequiredScalar(result, "X030L_WA")
	if err != nil {
		return zero, err
	}
	byteLength, fieldCount, err := x030lGeometry(x030l, name)
	if err != nil {
		return zero, err
	}
	var table *classicrfc.Table
	for i := range result.Tables {
		if result.Tables[i].Name == "DFIES_TAB" {
			table = &result.Tables[i]
		}
	}
	if table == nil {
		return zero, fmt.Errorf("%w: response lacks DFIES_TAB", ErrDdif)
	}
	if len(table.Rows) > maxDdicStructureFields {
		return zero, fmt.Errorf("%w: field count exceeds %d", ErrDdif, maxDdicStructureFields)
	}
	if fieldCount != len(table.Rows) {
		return zero, fmt.Errorf("%w: X030L_WA advertises %d fields; DFIES_TAB contains %d", ErrDdif, fieldCount, len(table.Rows))
	}

	names := map[string]bool{}
	var previousEnd int32
	normalized := make([]rfctypes.RfcStructureField, 0, len(table.Rows))
	for index, rowBytes := range table.Rows {
		field, err := DecodeDdIfDfiesRow(rowBytes)
		if err != nil {
			return zero, err
		}
		if field.TableName != name {
			return zero, fmt.Errorf("%w: DFIES %s belongs to %s; expected %s", ErrDdif, field.FieldName, field.TableName, name)
		}
		if int(field.Position) != index+1 {
			return zero, fmt.Errorf("%w: DFIES %s has position %d; expected %d", ErrDdif, field.FieldName, field.Position, index+1)
		}
		if field.componentType != "" && field.componentType != "E" {
			ct := field.componentType
			if ct == "" {
				ct = "<initial>"
			}
			return zero, fmt.Errorf("%w: DFIES %s.%s has unsupported component type %s", ErrDdif, name, field.FieldName, ct)
		}
		if names[field.FieldName] {
			return zero, fmt.Errorf("%w: DFIES contains duplicate field %s", ErrDdif, field.FieldName)
		}
		if field.Offset < previousEnd {
			return zero, fmt.Errorf("%w: DFIES %s overlaps its preceding field", ErrDdif, field.FieldName)
		}
		end := field.Offset + field.InternalLength
		if int(end) > byteLength {
			return zero, fmt.Errorf("%w: DFIES %s ends at %d beyond structure length %d", ErrDdif, field.FieldName, end, byteLength)
		}
		names[field.FieldName] = true
		previousEnd = end
		normalized = append(normalized, field.RfcStructureField)
	}
	return structure.ValidateCodec(rfctypes.RfcStructureDefinition{Name: name, ByteLength: int32(byteLength), Fields: normalized}, name)
}
