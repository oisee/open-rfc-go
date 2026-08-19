// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/values/classic-xrfc.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown errors became
// returned wrapped sentinels. The int8Mode and bcd representation-mode options
// are dropped (their selector modules classic-int8.ts / classic-bcd.ts are not
// ported): INT8 maps to Go int64 and packed decimals to canonical decimal
// strings, matching internal/structure. See docs/provenance.md.

// Package xrfc encodes and decodes the proven "classic" xRFC XML subset used to
// carry a flat structure or table whose layout contains a STRING/XSTRING field.
//
// A classic-RFC structure with only fixed-width fields travels through the
// internal/structure codec; one that also carries a dynamic STRING or XSTRING
// field cannot, and is serialised instead as this attribute-free XML subset.
package xrfc

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/structure"
	"github.com/oisee/open-rfc-go/internal/value"
)

var (
	// ErrRange reports a value or length outside the admitted range.
	ErrRange = errors.New("xrfc: value out of range")
	// ErrType reports a Go value of the wrong type for a field.
	ErrType = errors.New("xrfc: wrong value type")
	// ErrProtocol reports malformed or non-canonical xRFC XML.
	ErrProtocol = errors.New("xrfc: malformed xRFC XML")
)

// Kind selects whether a parameter carries a single structure or a table.
type Kind string

const (
	// KindStructure is one row with no <item> wrapper.
	KindStructure Kind = "structure"
	// KindTable is zero or more <item>-wrapped rows.
	KindTable Kind = "table"
)

// Limits bounds the size of an xRFC parameter; a nil field takes the default.
type Limits struct {
	MaxCellBytes      *int
	MaxRowBytes       *int
	MaxParameterBytes *int
	MaxRows           *int
}

// NormalizedLimits is a validated, fully-populated Limits.
type NormalizedLimits struct {
	MaxCellBytes      int
	MaxRowBytes       int
	MaxParameterBytes int
	MaxRows           int
}

var (
	simpleXMLName       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	canonicalInteger    = regexp.MustCompile(`^(?:0|-[1-9][0-9]*|[1-9][0-9]*)$`)
	finiteFloatLexical  = regexp.MustCompile(`^[+-]?(?:(?:[0-9]+(?:\.[0-9]*)?)|(?:\.[0-9]+))(?:[eE][+-]?[0-9]+)?$`)
	numericDigits       = regexp.MustCompile(`^[0-9]*$`)
	xrfcDatePattern     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	xrfcTimePattern     = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)
	canonicalBase64     = regexp.MustCompile(`^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$`)
	topLevelTagPattern  = regexp.MustCompile(`^<([A-Za-z_][A-Za-z0-9_]*)>`)
	supportedFieldTypes = map[string]bool{
		"I": true, "C": true, "N": true, "D": true, "T": true, "X": true,
		"P": true, "F": true, "8": true, "g": true, "y": true,
	}
)

func boundedLimit(value *int, fallback, maximum int, label string) (int, error) {
	normalized := fallback
	if value != nil {
		normalized = *value
	}
	if normalized < 0 || normalized > maximum {
		return 0, fmt.Errorf("%w: %s must be an integer in 0..%d", ErrRange, label, maximum)
	}
	return normalized, nil
}

// NormalizeLimits validates limits and fills defaults from the CPIC bounds.
func NormalizeLimits(limits Limits) (NormalizedLimits, error) {
	var out NormalizedLimits
	var err error
	if out.MaxCellBytes, err = boundedLimit(limits.MaxCellBytes, cpic.DefaultMaxFieldLength, cpic.DefaultMaxFieldLength, "maxCellBytes"); err != nil {
		return NormalizedLimits{}, err
	}
	if out.MaxRowBytes, err = boundedLimit(limits.MaxRowBytes, cpic.DefaultMaxFieldLength, cpic.DefaultMaxFieldLength, "maxRowBytes"); err != nil {
		return NormalizedLimits{}, err
	}
	if out.MaxParameterBytes, err = boundedLimit(limits.MaxParameterBytes, cpic.DefaultMaxFieldChainLength, cpic.DefaultMaxFieldChainLength, "maxParameterBytes"); err != nil {
		return NormalizedLimits{}, err
	}
	if out.MaxRows, err = boundedLimit(limits.MaxRows, cpic.DefaultMaxFieldCount, 0xffff_ffff, "maxRows"); err != nil {
		return NormalizedLimits{}, err
	}
	return out, nil
}

// AssertXMLName rejects any name outside the proven simple-XML-name subset.
func AssertXMLName(v, label string) error {
	if !simpleXMLName.MatchString(v) {
		return fmt.Errorf("%w: %s must be a simple XML name supported by the proven xRFC subset", ErrProtocol, label)
	}
	return nil
}

// OpenTagByteLength is the encoded length of "<name>".
func OpenTagByteLength(name string) int { return len(name) + 2 }

// CloseTagByteLength is the encoded length of "</name>".
func CloseTagByteLength(name string) int { return len(name) + 3 }

func assertXMLCodePoint(codePoint int, path string) error {
	if codePoint == 0 ||
		(codePoint < 0x20 && codePoint != 0x09 && codePoint != 0x0a && codePoint != 0x0d) ||
		codePoint == 0xfffe || codePoint == 0xffff {
		return fmt.Errorf("%w: %s contains a character unsupported by XML 1.0", ErrProtocol, path)
	}
	return nil
}

// EscapedXMLByteLength returns the encoded byte length of value as XML text.
func EscapedXMLByteLength(v, path string) (int, error) {
	if err := value.AssertUnicodeScalarText(v, path); err != nil {
		return 0, err
	}
	byteLength := 0
	for _, r := range v {
		if err := assertXMLCodePoint(int(r), path); err != nil {
			return 0, err
		}
		switch r {
		case '&', '<', '>':
			byteLength += 5 // &#38; / &#60; / &#62;
		default:
			byteLength += utf8.RuneLen(r)
		}
	}
	return byteLength, nil
}

func characterCapacity(field rfctypes.RfcStructureField, path string) (int, error) {
	if field.InternalLength&1 != 0 {
		return 0, fmt.Errorf("%w: %s Unicode character width must be even", ErrRange, path)
	}
	return int(field.InternalLength) / 2, nil
}

// utf16Len counts UTF-16 code units, matching JavaScript String.length.
func utf16Len(s string) int { return len(utf16.Encode([]rune(s))) }

func normalizeDefinition(def rfctypes.RfcStructureDefinition) (rfctypes.RfcStructureDefinition, error) {
	normalized, err := structure.ValidateCodec(def, "")
	if err != nil {
		return rfctypes.RfcStructureDefinition{}, err
	}
	dynamic, err := structure.HasDynamicFields(normalized)
	if err != nil {
		return rfctypes.RfcStructureDefinition{}, err
	}
	if !dynamic {
		return rfctypes.RfcStructureDefinition{}, fmt.Errorf("%w: %s has no STRING/XSTRING field requiring xRFC XML", ErrProtocol, normalized.Name)
	}
	for _, field := range normalized.Fields {
		if err := AssertXMLName(field.FieldName, normalized.Name+" field name"); err != nil {
			return rfctypes.RfcStructureDefinition{}, err
		}
		if !supportedFieldTypes[field.Exid] {
			return rfctypes.RfcStructureDefinition{}, fmt.Errorf("%w: %s.%s type %s is not implemented for the proven xRFC XML subset", ErrProtocol, normalized.Name, field.FieldName, field.Exid)
		}
	}
	return normalized, nil
}

// ValidateDefinition validates the supported flat xRFC row subset.
func ValidateDefinition(def rfctypes.RfcStructureDefinition) (rfctypes.RfcStructureDefinition, error) {
	return normalizeDefinition(def)
}

// --- value coercion helpers -------------------------------------------------

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	default:
		return 0, false
	}
}

func integerCell(v any, path string) (string, error) {
	n, ok := asInt64(v)
	if !ok || n < -0x8000_0000 || n > 0x7fff_ffff {
		return "", fmt.Errorf("%w: %s expects a signed 32-bit integer", ErrRange, path)
	}
	return strconv.FormatInt(n, 10), nil
}

// jsFloatString mirrors JavaScript String(number) for a finite float64.
func jsFloatString(f float64) string {
	if f == 0 {
		if math.Signbit(f) {
			return "-0"
		}
		return "0"
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func canonicalDateText(v any, path string) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s expects a DATE string", ErrType, path)
	}
	if err := value.AssertClassicDate(s, path); err != nil {
		return "", err
	}
	if s == "" || s == "        " {
		return "", nil
	}
	return s[0:4] + "-" + s[4:6] + "-" + s[6:8], nil
}

func canonicalTimeText(v any, path string) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s expects a TIME string", ErrType, path)
	}
	if err := value.AssertClassicTime(s, path); err != nil {
		return "", err
	}
	if s == "" || s == "      " {
		return "", nil
	}
	return s[0:2] + ":" + s[2:4] + ":" + s[4:6], nil
}

// initialCell returns the default cell value for an unsupplied field.
func initialCell(field rfctypes.RfcStructureField) (any, error) {
	switch field.Exid {
	case "I":
		return int32(0), nil
	case "C", "g", "N":
		return "", nil
	case "D":
		return "00000000", nil
	case "T":
		return "000000", nil
	case "X", "y":
		return []byte{}, nil
	case "P":
		return "0", nil
	case "F":
		return float64(0), nil
	case "8":
		return int64(0), nil
	default:
		return nil, fmt.Errorf("%w: unsupported xRFC field type %s", ErrProtocol, field.Exid)
	}
}

// --- planning ---------------------------------------------------------------

type plannedCell struct {
	field             rfctypes.RfcStructureField
	isBytes           bool
	text              string
	bytes             []byte
	encodedByteLength int
}

type plannedRow struct {
	cells             []plannedCell
	encodedByteLength int
}

func plannedBytesCell(field rfctypes.RfcStructureField, v []byte, path string, limits NormalizedLimits, exactLength int, fixed bool) (plannedCell, error) {
	byteLength := len(v)
	if fixed && byteLength > exactLength {
		return plannedCell{}, fmt.Errorf("%w: %s accepts at most %d bytes", ErrRange, path, exactLength)
	}
	plannedByteLength := byteLength
	if fixed {
		plannedByteLength = exactLength
	}
	encodedByteLength := (plannedByteLength + 2) / 3 * 4
	if encodedByteLength > limits.MaxCellBytes || encodedByteLength > limits.MaxRowBytes || encodedByteLength > limits.MaxParameterBytes {
		return plannedCell{}, fmt.Errorf("%w: %s base64 value exceeds the configured encoded-byte limits", ErrRange, path)
	}
	bytes := append([]byte(nil), v...)
	if fixed && exactLength != byteLength {
		padded := make([]byte, exactLength)
		copy(padded, bytes)
		bytes = padded
	}
	return plannedCell{field: field, isBytes: true, bytes: bytes, encodedByteLength: encodedByteLength}, nil
}

func planCell(def rfctypes.RfcStructureDefinition, field rfctypes.RfcStructureField, v any, limits NormalizedLimits) (plannedCell, error) {
	path := def.Name + "." + field.FieldName
	var text string
	switch field.Exid {
	case "I":
		t, err := integerCell(v, path)
		if err != nil {
			return plannedCell{}, err
		}
		text = t
	case "C":
		s, ok := v.(string)
		if !ok {
			return plannedCell{}, fmt.Errorf("%w: %s expects a string", ErrType, path)
		}
		if err := value.AssertUnicodeScalarText(s, path); err != nil {
			return plannedCell{}, err
		}
		capacity, err := characterCapacity(field, path)
		if err != nil {
			return plannedCell{}, err
		}
		if utf16Len(s) > capacity {
			return plannedCell{}, fmt.Errorf("%w: %s does not fit CHAR(%d)", ErrRange, path, capacity)
		}
		text = s
	case "N":
		capacity, err := characterCapacity(field, path)
		if err != nil {
			return plannedCell{}, err
		}
		maxPlanned := min3(limits.MaxCellBytes, limits.MaxRowBytes, limits.MaxParameterBytes)
		if capacity > maxPlanned {
			return plannedCell{}, fmt.Errorf("%w: %s padded NUM value exceeds the configured encoded-byte limits", ErrRange, path)
		}
		s, ok := v.(string)
		if !ok || !numericDigits.MatchString(s) || len(s) > capacity {
			return plannedCell{}, fmt.Errorf("%w: %s expects at most %d decimal digits", ErrType, path, capacity)
		}
		text = strings.Repeat("0", capacity-len(s)) + s
	case "D":
		t, err := canonicalDateText(v, path)
		if err != nil {
			return plannedCell{}, err
		}
		text = t
	case "T":
		t, err := canonicalTimeText(v, path)
		if err != nil {
			return plannedCell{}, err
		}
		text = t
	case "X":
		b, ok := v.([]byte)
		if !ok {
			return plannedCell{}, fmt.Errorf("%w: %s expects bytes", ErrType, path)
		}
		return plannedBytesCell(field, b, path, limits, int(field.InternalLength), true)
	case "P":
		s, ok := v.(string)
		if !ok {
			return plannedCell{}, fmt.Errorf("%w: %s expects a decimal string", ErrType, path)
		}
		packed, err := value.EncodePackedDecimal(s, int(field.InternalLength), int(field.Decimals), path)
		if err != nil {
			return plannedCell{}, err
		}
		t, err := value.DecodePackedDecimal(packed, int(field.Decimals), path)
		if err != nil {
			return plannedCell{}, err
		}
		text = t
	case "F":
		f, ok := v.(float64)
		if !ok || math.IsInf(f, 0) || math.IsNaN(f) {
			return plannedCell{}, fmt.Errorf("%w: %s expects a finite number", ErrType, path)
		}
		text = jsFloatString(f)
	case "8":
		n, ok := asInt64(v)
		if !ok {
			return plannedCell{}, fmt.Errorf("%w: %s expects a 64-bit integer", ErrType, path)
		}
		text = strconv.FormatInt(n, 10)
	case "g":
		s, ok := v.(string)
		if !ok {
			return plannedCell{}, fmt.Errorf("%w: %s expects Unicode text", ErrType, path)
		}
		if err := value.AssertNulFreeUnicodeScalarText(s, path); err != nil {
			return plannedCell{}, err
		}
		text = s
	case "y":
		b, ok := v.([]byte)
		if !ok {
			return plannedCell{}, fmt.Errorf("%w: %s expects bytes", ErrType, path)
		}
		return plannedBytesCell(field, b, path, limits, 0, false)
	default:
		return plannedCell{}, fmt.Errorf("%w: %s has an unsupported xRFC field type", ErrProtocol, path)
	}
	encodedByteLength, err := EscapedXMLByteLength(text, path)
	if err != nil {
		return plannedCell{}, err
	}
	if encodedByteLength > limits.MaxCellBytes {
		return plannedCell{}, fmt.Errorf("%w: %s XML value exceeds %d encoded bytes", ErrRange, path, limits.MaxCellBytes)
	}
	return plannedCell{field: field, text: text, encodedByteLength: encodedByteLength}, nil
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func planRow(def rfctypes.RfcStructureDefinition, input map[string]any, limits NormalizedLimits, itemWrapper bool, rowIndex int) (plannedRow, error) {
	rowPath := def.Name
	if rowIndex >= 0 {
		rowPath = fmt.Sprintf("%s[%d]", def.Name, rowIndex)
	}
	known := map[string]bool{}
	for _, f := range def.Fields {
		known[f.FieldName] = true
	}
	for name := range input {
		if !known[name] {
			return plannedRow{}, fmt.Errorf("%w: %s contains unknown field %s", ErrProtocol, rowPath, name)
		}
	}
	cells := make([]plannedCell, 0, len(def.Fields))
	encodedByteLength := 0
	if itemWrapper {
		encodedByteLength = 13 // <item></item>
	}
	for _, field := range def.Fields {
		v, supplied := input[field.FieldName]
		if !supplied {
			iv, err := initialCell(field)
			if err != nil {
				return plannedRow{}, err
			}
			v = iv
		}
		cell, err := planCell(def, field, v, limits)
		if err != nil {
			return plannedRow{}, err
		}
		encodedByteLength += OpenTagByteLength(field.FieldName) + cell.encodedByteLength + CloseTagByteLength(field.FieldName)
		if encodedByteLength > limits.MaxRowBytes {
			return plannedRow{}, fmt.Errorf("%w: %s XML row exceeds %d encoded bytes", ErrRange, rowPath, limits.MaxRowBytes)
		}
		cells = append(cells, cell)
	}
	return plannedRow{cells: cells, encodedByteLength: encodedByteLength}, nil
}

// --- writing ----------------------------------------------------------------

func writeEscapedText(b *strings.Builder, v string) {
	for _, r := range v {
		switch r {
		case '&':
			b.WriteString("&#38;")
		case '<':
			b.WriteString("&#60;")
		case '>':
			b.WriteString("&#62;")
		default:
			b.WriteRune(r)
		}
	}
}

func writePlannedRow(b *strings.Builder, row plannedRow, itemWrapper bool) {
	if itemWrapper {
		b.WriteString("<item>")
	}
	for _, cell := range row.cells {
		b.WriteByte('<')
		b.WriteString(cell.field.FieldName)
		b.WriteByte('>')
		if cell.isBytes {
			b.WriteString(base64.StdEncoding.EncodeToString(cell.bytes))
		} else {
			writeEscapedText(b, cell.text)
		}
		b.WriteString("</")
		b.WriteString(cell.field.FieldName)
		b.WriteByte('>')
	}
	if itemWrapper {
		b.WriteString("</item>")
	}
}

// EncodeParameter encodes one flat structure or table as the classic xRFC XML
// subset. For KindStructure, value must be map[string]any; for KindTable, it
// must be []map[string]any.
func EncodeParameter(parameterName string, def rfctypes.RfcStructureDefinition, kind Kind, v any, limits Limits) ([]byte, error) {
	if err := AssertXMLName(parameterName, "xRFC parameter name"); err != nil {
		return nil, err
	}
	if kind != KindStructure && kind != KindTable {
		return nil, fmt.Errorf("%w: xRFC parameter kind must be structure or table", ErrType)
	}
	normalized, err := normalizeDefinition(def)
	if err != nil {
		return nil, err
	}
	normalizedLimits, err := NormalizeLimits(limits)
	if err != nil {
		return nil, err
	}

	var rows []plannedRow
	byteLength := OpenTagByteLength(parameterName) + CloseTagByteLength(parameterName)
	if byteLength > normalizedLimits.MaxParameterBytes {
		return nil, fmt.Errorf("%w: %s xRFC XML exceeds %d bytes", ErrRange, parameterName, normalizedLimits.MaxParameterBytes)
	}
	appendRow := func(row plannedRow) error {
		byteLength += row.encodedByteLength
		if byteLength > normalizedLimits.MaxParameterBytes {
			return fmt.Errorf("%w: %s xRFC XML exceeds %d bytes", ErrRange, parameterName, normalizedLimits.MaxParameterBytes)
		}
		rows = append(rows, row)
		return nil
	}

	if kind == KindTable {
		table, ok := v.([]map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: %s expects an array of rows", ErrType, parameterName)
		}
		if len(table) > normalizedLimits.MaxRows {
			return nil, fmt.Errorf("%w: %s row count exceeds %d", ErrRange, parameterName, normalizedLimits.MaxRows)
		}
		for index, row := range table {
			planned, err := planRow(normalized, row, normalizedLimits, true, index)
			if err != nil {
				return nil, err
			}
			if err := appendRow(planned); err != nil {
				return nil, err
			}
		}
	} else {
		row, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: %s expects a structure object", ErrType, parameterName)
		}
		planned, err := planRow(normalized, row, normalizedLimits, false, -1)
		if err != nil {
			return nil, err
		}
		if err := appendRow(planned); err != nil {
			return nil, err
		}
	}

	var b strings.Builder
	b.Grow(byteLength)
	b.WriteByte('<')
	b.WriteString(parameterName)
	b.WriteByte('>')
	for _, row := range rows {
		writePlannedRow(&b, row, kind == KindTable)
	}
	b.WriteString("</")
	b.WriteString(parameterName)
	b.WriteByte('>')
	out := []byte(b.String())
	if len(out) != byteLength {
		return nil, fmt.Errorf("%w: %s xRFC XML encoder length invariant failed", ErrProtocol, parameterName)
	}
	return out, nil
}

// --- parsing ----------------------------------------------------------------

// parser walks the decoded UTF-8 text by byte offset. Because tokens are ASCII
// and the payload is valid UTF-8, a single byte offset is both the slicing
// position and the byte-length accumulator the limits are checked against.
type parser struct {
	text   string
	limits NormalizedLimits
	offset int
}

func (p *parser) startsWithTag(name string, closing bool) bool {
	token := "<" + name + ">"
	if closing {
		token = "</" + name + ">"
	}
	return strings.HasPrefix(p.text[p.offset:], token)
}

func (p *parser) open(name string) error {
	token := "<" + name + ">"
	if !strings.HasPrefix(p.text[p.offset:], token) {
		return fmt.Errorf("%w: expected %s at byte %d", ErrProtocol, token, p.offset)
	}
	p.offset += len(token)
	return nil
}

func (p *parser) close(name string) error {
	token := "</" + name + ">"
	if !strings.HasPrefix(p.text[p.offset:], token) {
		return fmt.Errorf("%w: expected %s at byte %d", ErrProtocol, token, p.offset)
	}
	p.offset += len(token)
	return nil
}

func (p *parser) cell(path string) (string, error) {
	rel := strings.IndexByte(p.text[p.offset:], '<')
	if rel < 0 {
		return "", fmt.Errorf("%w: %s is truncated", ErrProtocol, path)
	}
	raw := p.text[p.offset : p.offset+rel]
	p.offset += rel
	if strings.Contains(raw, "]]>") {
		return "", fmt.Errorf("%w: %s contains invalid XML character data", ErrProtocol, path)
	}
	if len(raw) > p.limits.MaxCellBytes {
		return "", fmt.Errorf("%w: %s XML value exceeds %d encoded bytes", ErrRange, path, p.limits.MaxCellBytes)
	}
	return decodeEntities(raw, path)
}

func (p *parser) finish() error {
	if p.offset != len(p.text) {
		return fmt.Errorf("%w: xRFC XML has trailing content at byte %d", ErrProtocol, p.offset)
	}
	return nil
}

func decodeEntities(raw, path string) (string, error) {
	var b strings.Builder
	offset := 0
	for offset < len(raw) {
		amp := strings.IndexByte(raw[offset:], '&')
		if amp < 0 {
			b.WriteString(raw[offset:])
			break
		}
		amp += offset
		b.WriteString(raw[offset:amp])
		codePoint, length, err := value.DecodeXMLEntityReference(raw, amp, path)
		if err != nil {
			return "", err
		}
		if err := assertXMLCodePoint(codePoint, path); err != nil {
			return "", err
		}
		b.WriteRune(rune(codePoint))
		offset = amp + length
	}
	result := b.String()
	if err := value.AssertUnicodeScalarText(result, path); err != nil {
		return "", err
	}
	for _, r := range result {
		if err := assertXMLCodePoint(int(r), path); err != nil {
			return "", err
		}
	}
	return result, nil
}

// DecodeBase64 decodes one canonical, unpadded-or-padded base64 cell value.
func DecodeBase64(v, path string, maximum int) ([]byte, error) {
	if len(v) == 0 {
		return []byte{}, nil
	}
	if len(v)&3 != 0 || !canonicalBase64.MatchString(v) {
		return nil, fmt.Errorf("%w: %s contains non-canonical base64", ErrProtocol, path)
	}
	decodedByteLength := len(v) / 4 * 3
	switch {
	case strings.HasSuffix(v, "=="):
		decodedByteLength -= 2
	case strings.HasSuffix(v, "="):
		decodedByteLength -= 1
	}
	if decodedByteLength > maximum {
		return nil, fmt.Errorf("%w: %s decoded bytes exceed %d", ErrRange, path, maximum)
	}
	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("%w: %s contains non-canonical base64", ErrProtocol, path)
	}
	if base64.StdEncoding.EncodeToString(decoded) != v {
		return nil, fmt.Errorf("%w: %s contains non-canonical base64", ErrProtocol, path)
	}
	return decoded, nil
}

type decodeBudget struct {
	limits         NormalizedLimits
	projectedBytes int
}

func (b *decodeBudget) consume(byteLength int, path string) error {
	if byteLength > b.limits.MaxCellBytes {
		return fmt.Errorf("%w: %s decoded value exceeds the %d-byte cell limit", ErrRange, path, b.limits.MaxCellBytes)
	}
	b.projectedBytes += byteLength
	if b.projectedBytes > b.limits.MaxParameterBytes {
		return fmt.Errorf("%w: %s decoded output exceeds the %d-byte parameter limit", ErrRange, path, b.limits.MaxParameterBytes)
	}
	return nil
}

func decodeCell(def rfctypes.RfcStructureDefinition, field rfctypes.RfcStructureField, v string, limits NormalizedLimits, budget *decodeBudget) (any, error) {
	path := def.Name + "." + field.FieldName
	switch field.Exid {
	case "I":
		if !canonicalInteger.MatchString(v) {
			return nil, fmt.Errorf("%w: %s contains a non-canonical INT4 value", ErrProtocol, path)
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < -0x8000_0000 || n > 0x7fff_ffff {
			return nil, fmt.Errorf("%w: %s INT4 value is out of range", ErrRange, path)
		}
		return int32(n), nil
	case "C":
		if err := value.AssertUnicodeScalarText(v, path); err != nil {
			return nil, err
		}
		capacity, err := characterCapacity(field, path)
		if err != nil {
			return nil, err
		}
		if utf16Len(v) > capacity {
			return nil, fmt.Errorf("%w: %s does not fit CHAR(%d)", ErrRange, path, capacity)
		}
		return v, nil
	case "N":
		capacity, err := characterCapacity(field, path)
		if err != nil {
			return nil, err
		}
		if !numericDigits.MatchString(v) || len(v) > capacity {
			return nil, fmt.Errorf("%w: %s contains a non-canonical NUM value", ErrProtocol, path)
		}
		if err := budget.consume(capacity, path); err != nil {
			return nil, err
		}
		return strings.Repeat("0", capacity-len(v)) + v, nil
	case "D":
		if len(v) == 0 {
			return "", nil
		}
		if !xrfcDatePattern.MatchString(v) {
			return nil, fmt.Errorf("%w: %s contains a non-canonical xRFC DATE", ErrProtocol, path)
		}
		date := strings.ReplaceAll(v, "-", "")
		if err := value.AssertClassicDate(date, path); err != nil {
			return nil, err
		}
		return date, nil
	case "T":
		if len(v) == 0 {
			return "", nil
		}
		if !xrfcTimePattern.MatchString(v) {
			return nil, fmt.Errorf("%w: %s contains a non-canonical xRFC TIME", ErrProtocol, path)
		}
		t := strings.ReplaceAll(v, ":", "")
		if err := value.AssertClassicTime(t, path); err != nil {
			return nil, err
		}
		return t, nil
	case "X":
		if err := budget.consume(int(field.InternalLength), path); err != nil {
			return nil, err
		}
		decoded, err := DecodeBase64(v, path, limits.MaxCellBytes)
		if err != nil {
			return nil, err
		}
		if len(decoded) != int(field.InternalLength) {
			return nil, fmt.Errorf("%w: %s fixed byte value must contain %d bytes", ErrRange, path, field.InternalLength)
		}
		return decoded, nil
	case "P":
		packed, err := value.EncodePackedDecimal(v, int(field.InternalLength), int(field.Decimals), path)
		if err != nil {
			return nil, err
		}
		return value.DecodePackedDecimal(packed, int(field.Decimals), path)
	case "F":
		if !finiteFloatLexical.MatchString(v) {
			return nil, fmt.Errorf("%w: %s contains an invalid FLOAT", ErrProtocol, path)
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, fmt.Errorf("%w: %s contains an invalid FLOAT", ErrProtocol, path)
		}
		return f, nil
	case "8":
		if !canonicalInteger.MatchString(v) || len(v) > 20 {
			return nil, fmt.Errorf("%w: %s contains a non-canonical INT8 value", ErrProtocol, path)
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %s INT8 value is out of range", ErrRange, path)
		}
		return n, nil
	case "g":
		if err := value.AssertNulFreeUnicodeScalarText(v, path); err != nil {
			return nil, err
		}
		return v, nil
	case "y":
		decoded, err := DecodeBase64(v, path, limits.MaxCellBytes)
		if err != nil {
			return nil, err
		}
		if err := budget.consume(len(decoded), path); err != nil {
			return nil, err
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("%w: %s has an unsupported xRFC field type", ErrProtocol, path)
	}
}

func parseRow(p *parser, def rfctypes.RfcStructureDefinition, limits NormalizedLimits, itemWrapper bool, budget *decodeBudget) (map[string]any, error) {
	start := p.offset
	if itemWrapper {
		if err := p.open("item"); err != nil {
			return nil, err
		}
	}
	result := make(map[string]any, len(def.Fields))
	for _, field := range def.Fields {
		if err := p.open(field.FieldName); err != nil {
			return nil, err
		}
		text, err := p.cell(def.Name + "." + field.FieldName)
		if err != nil {
			return nil, err
		}
		if err := p.close(field.FieldName); err != nil {
			return nil, err
		}
		decoded, err := decodeCell(def, field, text, limits, budget)
		if err != nil {
			return nil, err
		}
		result[field.FieldName] = decoded
	}
	if itemWrapper {
		if err := p.close("item"); err != nil {
			return nil, err
		}
	}
	if p.offset-start > limits.MaxRowBytes {
		return nil, fmt.Errorf("%w: %s XML row exceeds %d encoded bytes", ErrRange, def.Name, limits.MaxRowBytes)
	}
	return result, nil
}

func snapshotAndValidate(parameterName string, v []byte, limits NormalizedLimits) (string, error) {
	byteLength := len(v)
	if byteLength == 0 || byteLength > limits.MaxParameterBytes {
		return "", fmt.Errorf("%w: %s xRFC XML must contain 1..%d bytes", ErrRange, parameterName, limits.MaxParameterBytes)
	}
	if byteLength >= 3 && v[0] == 0xef && v[1] == 0xbb && v[2] == 0xbf {
		return "", fmt.Errorf("%w: %s xRFC XML must not contain a UTF-8 BOM", ErrProtocol, parameterName)
	}
	if !utf8.Valid(v) {
		return "", fmt.Errorf("%w: %s xRFC XML is not valid UTF-8", ErrProtocol, parameterName)
	}
	return string(v), nil
}

// DecodeParameterName returns the strict top-level tag name without accepting
// any XML prolog.
func DecodeParameterName(v []byte, limits Limits) (string, error) {
	normalizedLimits, err := NormalizeLimits(limits)
	if err != nil {
		return "", err
	}
	text, err := snapshotAndValidate("xRFC XML parameter", v, normalizedLimits)
	if err != nil {
		return "", err
	}
	match := topLevelTagPattern.FindStringSubmatch(text)
	if match == nil {
		return "", fmt.Errorf("%w: xRFC XML parameter lacks a supported top-level tag", ErrProtocol)
	}
	return match[1], nil
}

// DecodeParameter decodes the exact, attribute-free flat xRFC XML subset. For
// KindStructure it returns map[string]any; for KindTable, []map[string]any.
func DecodeParameter(parameterName string, def rfctypes.RfcStructureDefinition, kind Kind, v []byte, limits Limits) (any, error) {
	if err := AssertXMLName(parameterName, "xRFC parameter name"); err != nil {
		return nil, err
	}
	if kind != KindStructure && kind != KindTable {
		return nil, fmt.Errorf("%w: xRFC parameter kind must be structure or table", ErrType)
	}
	normalized, err := normalizeDefinition(def)
	if err != nil {
		return nil, err
	}
	normalizedLimits, err := NormalizeLimits(limits)
	if err != nil {
		return nil, err
	}
	text, err := snapshotAndValidate(parameterName, v, normalizedLimits)
	if err != nil {
		return nil, err
	}
	p := &parser{text: text, limits: normalizedLimits}
	budget := &decodeBudget{limits: normalizedLimits}
	if err := p.open(parameterName); err != nil {
		return nil, err
	}
	if kind == KindStructure {
		row, err := parseRow(p, normalized, normalizedLimits, false, budget)
		if err != nil {
			return nil, err
		}
		if err := p.close(parameterName); err != nil {
			return nil, err
		}
		if err := p.finish(); err != nil {
			return nil, err
		}
		return row, nil
	}
	rows := []map[string]any{}
	for !p.startsWithTag(parameterName, true) {
		if len(rows) >= normalizedLimits.MaxRows {
			return nil, fmt.Errorf("%w: %s row count exceeds %d", ErrRange, parameterName, normalizedLimits.MaxRows)
		}
		row, err := parseRow(p, normalized, normalizedLimits, true, budget)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if err := p.close(parameterName); err != nil {
		return nil, err
	}
	if err := p.finish(); err != nil {
		return nil, err
	}
	return rows, nil
}
