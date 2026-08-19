// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/protocol/classic-rfc.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown RangeError/Error
// became returned, wrapped errors; the intrinsic typed-array geometry helpers
// collapse to slices/copies; the borrow-vs-snapshot ownership split is kept as
// a boolean (borrow returns sub-slices of the caller's field bytes, snapshot
// copies). decodeRfcFunintRow bounds the row *below* by the 402-byte stable
// prefix and ignores anything appended after it — the recurring-bug-class fix,
// preserved verbatim, so a later release that grows the row still decodes. This
// file has no isolated upstream test; the Go tests state its wire facts. See
// docs/provenance.md.

// Package classicrfc decodes the classic RFC syntax layer: ABAP CHAR values,
// table headers and rows, the grouped function result, and RFC_FUNINT rows.
package classicrfc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf16"

	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/value"
)

// RfcFunintUnicodeRowLength is the stable prefix width of a Unicode RFC_FUNINT
// row. Later releases append fields; this decoder consumes only the prefix.
const RfcFunintUnicodeRowLength = 402

const (
	defaultMaxTableDecodedBytes       = cpic.DefaultMaxFieldChainLength
	defaultMaxResultTableDecodedBytes = cpic.DefaultMaxFieldChainLength
)

var (
	// ErrRange reports a value or geometry outside the classic RFC contract.
	ErrRange = errors.New("classicrfc: value out of range")
	// ErrProtocol reports a structurally invalid classic RFC record stream.
	ErrProtocol = errors.New("classicrfc: protocol violation")
)

// RfcTableHeader is the eight-byte classic table header.
type RfcTableHeader struct {
	DeclaredRowByteLength uint32
	RowCount              uint32
}

// Scalar is one decoded scalar parameter (lossless wire bytes).
type Scalar struct {
	Name  string
	Value []byte
}

// Table is one decoded table parameter.
type Table struct {
	Name                  string
	DeclaredRowByteLength uint32
	RowByteLength         int
	RowEncoding           string // "flat"|"structured"|"mixed"|"empty"
	RowCompression        string // "none"|"simple"|"mixed"|"empty"
	Rows                  [][]byte
}

// XrfcParameter is the concatenated UTF-8 XML between one 0x3c02 boundary pair.
type XrfcParameter struct {
	Value      []byte
	ChunkCount int
}

// Result is a grouped classic function response.
type Result struct {
	RequestedOutputs []string
	Scalars          []Scalar
	Tables           []Table
	XrfcParameters   []XrfcParameter
}

// FunintParameter is one decoded Unicode RFC_FUNINT row.
type FunintParameter struct {
	ParameterClass string
	ParameterName  string
	TableName      string
	FieldName      string
	Exid           string
	Position       int32
	Offset         int32
	InternalLength int32
	Decimals       int32
	DefaultValue   string
	ParameterText  string
	Optional       bool
}

func characterCount(v int, field string) error {
	if v < 0 || v > 0x7fff {
		return fmt.Errorf("%w: %s must be an integer in 0..32767", ErrRange, field)
	}
	return nil
}

// EncodeAbapChar encodes a fixed-width Unicode CHAR value, space-padded.
func EncodeAbapChar(value_ string, characters int) ([]byte, error) {
	if err := characterCount(characters, "ABAP CHAR length"); err != nil {
		return nil, err
	}
	if err := value.AssertUnicodeScalarText(value_, "ABAP CHAR value"); err != nil {
		return nil, err
	}
	if codeUnitLen(value_) > characters {
		return nil, fmt.Errorf("%w: ABAP CHAR value of %d characters does not fit CHAR(%d)", ErrRange, codeUnitLen(value_), characters)
	}
	padded := value_ + strings.Repeat(" ", characters-codeUnitLen(value_))
	return utf16leEncode(padded), nil
}

func abapCharacterBytes(v []byte, expected int, hasExpected bool) (string, error) {
	if len(v)&1 != 0 {
		return "", fmt.Errorf("%w: Unicode ABAP CHAR must have an even byte length", ErrRange)
	}
	if hasExpected {
		if err := characterCount(expected, "expected ABAP CHAR length"); err != nil {
			return "", err
		}
		if len(v) != expected*2 {
			return "", fmt.Errorf("%w: Unicode ABAP CHAR must contain exactly %d bytes; received %d", ErrRange, expected*2, len(v))
		}
	}
	decoded := utf16leDecode(v)
	if err := value.AssertUnicodeScalarText(decoded, "decoded ABAP CHAR value"); err != nil {
		return "", err
	}
	return decoded, nil
}

// DecodeAbapChar decodes fixed or variable CHAR bytes and strips trailing
// spaces. Pass an expected character count to require an exact width.
func DecodeAbapChar(v []byte, expected ...int) (string, error) {
	s, err := abapCharacterBytes(v, firstOr(expected, 0), len(expected) > 0)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s, " "), nil
}

// DecodeAbapFixedChar decodes an exact-width value whose spaces are data.
func DecodeAbapFixedChar(v []byte, expected int) (string, error) {
	return abapCharacterBytes(v, expected, true)
}

var controlChar = regexp.MustCompile(`[\x00-\x1f\x7f]`)

func decodeParameterName(v []byte, field string) (string, error) {
	name, err := DecodeAbapChar(v)
	if err != nil {
		return "", err
	}
	if len(name) == 0 || controlChar.MatchString(name) {
		return "", fmt.Errorf("%w: %s is not a valid non-empty RFC parameter name", ErrProtocol, field)
	}
	return name, nil
}

// DecodeRfcTableHeader decodes the eight-byte classic table header.
func DecodeRfcTableHeader(v []byte) (RfcTableHeader, error) {
	if len(v) != 8 {
		return RfcTableHeader{}, fmt.Errorf("%w: classic RFC table header must contain exactly 8 bytes; received %d", ErrRange, len(v))
	}
	return RfcTableHeader{
		DeclaredRowByteLength: binary.BigEndian.Uint32(v[0:]),
		RowCount:              binary.BigEndian.Uint32(v[4:]),
	}, nil
}

var nonApplicationTags = map[uint16]bool{
	uint16(cpic.TagResponseContext): true,
	uint16(cpic.TagSession):         true,
	uint16(cpic.TagUnresolved0420):  true,
	uint16(cpic.TagCallContext):     true,
	uint16(cpic.TagProgram):         true,
	0x0667:                          true,
	uint16(cpic.TagEnd):             true,
}

func retainedWireBuffer(v []byte, borrow bool) []byte {
	if borrow {
		return v
	}
	return append([]byte(nil), v...)
}

func decodeSimpleCompressedTableRow(v []byte, declaredRowByteLength uint32, tableName string, rowIndex int, borrow bool) ([]byte, error) {
	if len(v) == 0 {
		return nil, fmt.Errorf("%w: classic RFC table %s simple-compressed row %d is empty", ErrProtocol, tableName, rowIndex)
	}
	if uint32(len(v)) > declaredRowByteLength {
		return nil, fmt.Errorf("%w: classic RFC table %s simple-compressed row %d has %d encoded bytes; declared row width is %d", ErrProtocol, tableName, rowIndex, len(v), declaredRowByteLength)
	}
	if uint32(len(v)) == declaredRowByteLength {
		return retainedWireBuffer(v, borrow), nil
	}
	if int(declaredRowByteLength) > cpic.DefaultMaxFieldLength {
		return nil, fmt.Errorf("%w: classic RFC table %s simple-compressed row %d expands to %d bytes; maximum is %d", ErrRange, tableName, rowIndex, declaredRowByteLength, cpic.DefaultMaxFieldLength)
	}
	decoded := make([]byte, declaredRowByteLength)
	fill := v[len(v)-1]
	for i := range decoded {
		decoded[i] = fill
	}
	copy(decoded, v)
	return decoded, nil
}

func preflightResultBytes(fields []cpic.Field) error {
	if len(fields) > cpic.DefaultMaxFieldCount {
		return fmt.Errorf("%w: classic RFC result field count exceeds %d", ErrRange, cpic.DefaultMaxFieldCount)
	}
	valueBytes := 0
	for _, f := range fields {
		valueBytes += len(f.Value)
		if valueBytes > cpic.DefaultMaxFieldChainLength {
			return fmt.Errorf("%w: classic RFC result field bytes %d exceed %d", ErrRange, valueBytes, cpic.DefaultMaxFieldChainLength)
		}
	}
	return nil
}

func preflightTableBytes(fields []cpic.Field) error {
	resultDecodedBytes := 0
	for index := 0; index < len(fields); index++ {
		field := fields[index]
		if field.Tag != uint16(cpic.TagTableName) {
			continue
		}
		if index+1 >= len(fields) || fields[index+1].Tag != uint16(cpic.TagTableHeader) {
			continue
		}
		name, err := decodeParameterName(field.Value, "table parameter name")
		if err != nil {
			return err
		}
		header, err := DecodeRfcTableHeader(fields[index+1].Value)
		if err != nil {
			return err
		}
		tableDecodedBytes := 0
		rowCount := uint32(0)
		cursor := index + 2
		for rowCount < header.RowCount && cursor < len(fields) &&
			(fields[cursor].Tag == uint16(cpic.TagTableContent) || fields[cursor].Tag == uint16(cpic.TagTableCompr)) {
			row := fields[cursor]
			enc := len(row.Value)
			compr := row.Tag == uint16(cpic.TagTableCompr)
			label := "uncompressed"
			if compr {
				label = "simple-compressed"
			}
			if enc == 0 {
				return fmt.Errorf("%w: classic RFC table %s %s row %d is empty", ErrProtocol, name, label, rowCount)
			}
			if uint32(enc) > header.DeclaredRowByteLength {
				encodedLabel := ""
				if compr {
					encodedLabel = " encoded"
				}
				return fmt.Errorf("%w: classic RFC table %s %s row %d has %d%s bytes; declared row width is %d", ErrProtocol, name, label, rowCount, enc, encodedLabel, header.DeclaredRowByteLength)
			}
			if compr && uint32(enc) < header.DeclaredRowByteLength && int(header.DeclaredRowByteLength) > cpic.DefaultMaxFieldLength {
				return fmt.Errorf("%w: classic RFC table %s simple-compressed row %d expands to %d bytes; maximum is %d", ErrRange, name, rowCount, header.DeclaredRowByteLength, cpic.DefaultMaxFieldLength)
			}
			if compr {
				tableDecodedBytes += int(header.DeclaredRowByteLength)
			} else {
				tableDecodedBytes += enc
			}
			rowCount++
			cursor++
		}
		if rowCount != header.RowCount {
			return fmt.Errorf("%w: classic RFC table %s declares %d rows but found %d", ErrProtocol, name, header.RowCount, rowCount)
		}
		if tableDecodedBytes > defaultMaxTableDecodedBytes {
			return fmt.Errorf("%w: classic RFC table %s decoded bytes %d exceed table limit %d", ErrRange, name, tableDecodedBytes, defaultMaxTableDecodedBytes)
		}
		resultDecodedBytes += tableDecodedBytes
		if resultDecodedBytes > defaultMaxResultTableDecodedBytes {
			return fmt.Errorf("%w: classic RFC decoded table bytes %d exceed result limit %d", ErrRange, resultDecodedBytes, defaultMaxResultTableDecodedBytes)
		}
		index = cursor - 1
	}
	return nil
}

// DecodeResult groups a decoded function response into lossless scalar/table
// wire values. Unknown application records are rejected.
func DecodeResult(fields []cpic.Field) (Result, error) {
	return decodeResult(fields, false)
}

// DecodeOwnedResult decodes values already owned by the current CPIC session
// without a second snapshot; returned buffers may alias fields.
func DecodeOwnedResult(fields []cpic.Field) (Result, error) {
	return decodeResult(fields, true)
}

func decodeResult(fields []cpic.Field, borrow bool) (Result, error) {
	var zero Result
	if err := preflightResultBytes(fields); err != nil {
		return zero, err
	}
	if err := preflightTableBytes(fields); err != nil {
		return zero, err
	}
	var result Result
	names := map[string]bool{}

	for index := 0; index < len(fields); index++ {
		field := fields[index]
		if nonApplicationTags[field.Tag] {
			continue
		}
		switch field.Tag {
		case uint16(cpic.TagRequestedOutput):
			name, err := decodeParameterName(field.Value, "requested output name")
			if err != nil {
				return zero, err
			}
			result.RequestedOutputs = append(result.RequestedOutputs, name)
		case uint16(cpic.TagXRfcData):
			return zero, fmt.Errorf("%w: classic RFC response contains xRFC XML data without an opening boundary", ErrProtocol)
		case uint16(cpic.TagXRfcParameter):
			if len(field.Value) != 0 {
				return zero, fmt.Errorf("%w: classic RFC xRFC XML opening boundary must be empty", ErrProtocol)
			}
			var chunks [][]byte
			byteLength := 0
			index++
			for index < len(fields) && fields[index].Tag == uint16(cpic.TagXRfcData) {
				chunk := fields[index].Value
				if len(chunk) == 0 {
					return zero, fmt.Errorf("%w: classic RFC xRFC XML data chunk must not be empty", ErrProtocol)
				}
				byteLength += len(chunk)
				if byteLength > cpic.DefaultMaxFieldChainLength {
					return zero, fmt.Errorf("%w: classic RFC xRFC XML parameter exceeds %d bytes", ErrRange, cpic.DefaultMaxFieldChainLength)
				}
				chunks = append(chunks, chunk)
				index++
			}
			if len(chunks) == 0 {
				return zero, fmt.Errorf("%w: classic RFC xRFC XML boundary contains no data chunk", ErrProtocol)
			}
			if index >= len(fields) || fields[index].Tag != uint16(cpic.TagXRfcParameter) {
				return zero, fmt.Errorf("%w: classic RFC xRFC XML parameter lacks its closing boundary", ErrProtocol)
			}
			if len(fields[index].Value) != 0 {
				return zero, fmt.Errorf("%w: classic RFC xRFC XML closing boundary must be empty", ErrProtocol)
			}
			var xvalue []byte
			if borrow && len(chunks) == 1 {
				xvalue = chunks[0]
			} else {
				xvalue = concatChunks(chunks, byteLength)
			}
			result.XrfcParameters = append(result.XrfcParameters, XrfcParameter{Value: xvalue, ChunkCount: len(chunks)})
		case uint16(cpic.TagParameterValue):
			return zero, fmt.Errorf("%w: classic RFC response contains a value without a parameter name", ErrProtocol)
		case uint16(cpic.TagTableHeader), uint16(cpic.TagTableContent), uint16(cpic.TagTableCompr):
			return zero, fmt.Errorf("%w: classic RFC response contains a table record without a table name", ErrProtocol)
		case uint16(cpic.TagParameterName):
			name, err := decodeParameterName(field.Value, "scalar parameter name")
			if err != nil {
				return zero, err
			}
			if index+1 >= len(fields) || fields[index+1].Tag != uint16(cpic.TagParameterValue) {
				return zero, fmt.Errorf("%w: classic RFC scalar %s is not followed by its value", ErrProtocol, name)
			}
			if names[name] {
				return zero, fmt.Errorf("%w: classic RFC response contains duplicate parameter %s", ErrProtocol, name)
			}
			names[name] = true
			result.Scalars = append(result.Scalars, Scalar{Name: name, Value: retainedWireBuffer(fields[index+1].Value, borrow)})
			index++
		case uint16(cpic.TagTableName):
			name, err := decodeParameterName(field.Value, "table parameter name")
			if err != nil {
				return zero, err
			}
			if index+1 >= len(fields) || fields[index+1].Tag != uint16(cpic.TagTableHeader) {
				return zero, fmt.Errorf("%w: classic RFC table %s is not followed by its header", ErrProtocol, name)
			}
			if names[name] {
				return zero, fmt.Errorf("%w: classic RFC response contains duplicate parameter %s", ErrProtocol, name)
			}
			header, err := DecodeRfcTableHeader(fields[index+1].Value)
			if err != nil {
				return zero, err
			}
			var rows [][]byte
			sawUncompressed, sawSimpleCompressed := false, false
			rowByteLength := int(header.DeclaredRowByteLength)
			index += 2
			for uint32(len(rows)) < header.RowCount && index < len(fields) &&
				(fields[index].Tag == uint16(cpic.TagTableContent) || fields[index].Tag == uint16(cpic.TagTableCompr)) {
				rowField := fields[index]
				var row []byte
				if rowField.Tag == uint16(cpic.TagTableCompr) {
					sawSimpleCompressed = true
					row, err = decodeSimpleCompressedTableRow(rowField.Value, header.DeclaredRowByteLength, name, len(rows), borrow)
					if err != nil {
						return zero, err
					}
				} else {
					sawUncompressed = true
					row = retainedWireBuffer(rowField.Value, borrow)
					if len(row) == 0 {
						return zero, fmt.Errorf("%w: classic RFC table %s uncompressed row %d is empty", ErrProtocol, name, len(rows))
					}
					if uint32(len(row)) > header.DeclaredRowByteLength {
						return zero, fmt.Errorf("%w: classic RFC table %s uncompressed row %d has %d bytes; declared row width is %d", ErrProtocol, name, len(rows), len(row), header.DeclaredRowByteLength)
					}
				}
				if len(rows) == 0 {
					rowByteLength = len(row)
				}
				rows = append(rows, row)
				index++
			}
			if uint32(len(rows)) != header.RowCount {
				return zero, fmt.Errorf("%w: classic RFC table %s declares %d rows but found %d", ErrProtocol, name, header.RowCount, len(rows))
			}
			rowEncoding, rowCompression := "empty", "empty"
			switch {
			case sawUncompressed && sawSimpleCompressed:
				rowEncoding, rowCompression = "mixed", "mixed"
			case sawUncompressed:
				rowEncoding, rowCompression = "flat", "none"
			case sawSimpleCompressed:
				rowEncoding, rowCompression = "structured", "simple"
			}
			index--
			names[name] = true
			result.Tables = append(result.Tables, Table{
				Name: name, DeclaredRowByteLength: header.DeclaredRowByteLength, RowByteLength: rowByteLength,
				RowEncoding: rowEncoding, RowCompression: rowCompression, Rows: rows,
			})
		default:
			return zero, fmt.Errorf("%w: classic RFC response contains unsupported tag 0x%04x", ErrProtocol, field.Tag)
		}
	}
	return result, nil
}

// DecodeFunintRow decodes one Unicode RFC_FUNINT row. The row is bounded below
// by the 402-byte stable prefix; anything appended after it is ignored, so a
// later release that grows the row still decodes. A short row is refused.
func DecodeFunintRow(v []byte) (FunintParameter, error) {
	var zero FunintParameter
	if len(v) < RfcFunintUnicodeRowLength {
		return zero, fmt.Errorf("%w: Unicode RFC_FUNINT row must contain at least %d bytes; received %d", ErrRange, RfcFunintUnicodeRowLength, len(v))
	}
	prefix := v[:RfcFunintUnicodeRowLength]
	off := 0
	take := func(n int) []byte { b := prefix[off : off+n]; off += n; return b }
	var err error
	dc := func(b []byte, chars int) string {
		if err != nil {
			return ""
		}
		var s string
		s, err = DecodeAbapChar(b, chars)
		return s
	}
	parameterClass := dc(take(2), 1)
	parameterName := dc(take(60), 30)
	tableName := dc(take(60), 30)
	fieldName := dc(take(60), 30)
	exid := dc(take(2), 1)
	position := int32(binary.LittleEndian.Uint32(take(4)))
	offset := int32(binary.LittleEndian.Uint32(take(4)))
	internalLength := int32(binary.LittleEndian.Uint32(take(4)))
	decimals := int32(binary.LittleEndian.Uint32(take(4)))
	defaultValue := dc(take(42), 21)
	parameterText := dc(take(158), 79)
	optionalText := dc(take(2), 1)
	if err != nil {
		return zero, err
	}
	if optionalText != "" && optionalText != "X" {
		return zero, fmt.Errorf("%w: RFC_FUNINT OPTIONAL contains unsupported value %s", ErrProtocol, optionalText)
	}
	return FunintParameter{
		ParameterClass: parameterClass, ParameterName: parameterName, TableName: tableName, FieldName: fieldName,
		Exid: exid, Position: position, Offset: offset, InternalLength: internalLength, Decimals: decimals,
		DefaultValue: defaultValue, ParameterText: parameterText, Optional: optionalText == "X",
	}, nil
}

func codeUnitLen(s string) int { return len(utf16.Encode([]rune(s))) }

func utf16leEncode(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(out[i*2:], u)
	}
	return out
}

func utf16leDecode(b []byte) string {
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(units))
}

func concatChunks(chunks [][]byte, total int) []byte {
	out := make([]byte, 0, total)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}

func firstOr(s []int, d int) int {
	if len(s) > 0 {
		return s[0]
	}
	return d
}
