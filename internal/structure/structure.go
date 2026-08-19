// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/values/classic-structure.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Structure input/output
// is map[string]any (field name -> typed Go value) rather than a JS object; the
// int8Mode/bcd node-rfc representation options are dropped (INT8 -> int64, BCD
// and DECF16/DECF34 -> precision-preserving decimal strings, per the dropped
// classic-bcd/classic-int8 facades); the WeakSet definition cache and intrinsic-
// geometry helpers collapse; thrown errors -> returned wrapped sentinels.
// STRING/XSTRING (g/y) require the xRFC serializer and are refused by the fixed
// codec, as upstream. See docs/provenance.md.

// Package structure encodes and decodes fixed-layout classic Unicode ABAP
// structures against their RFC field definitions.
package structure

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf16"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/value"
)

const (
	maxClassicStructureFields     = 100_000
	maxClassicStructureByteLength = cpic.DefaultMaxFieldLength
)

// ErrStructure reports an invalid structure definition, geometry, or value.
var ErrStructure = errors.New("structure: classic structure")

var numericDigits = regexp.MustCompile(`^\d*$`)

// SnapshotDefinition validates fixed-structure geometry before value allocation.
func SnapshotDefinition(def rfctypes.RfcStructureDefinition, requestedName string) (rfctypes.RfcStructureDefinition, error) {
	var zero rfctypes.RfcStructureDefinition
	expectedName := requestedName
	if expectedName == "" {
		expectedName = def.Name
	}
	if def.Name == "" || def.Name != expectedName {
		return zero, fmt.Errorf("%w: %s structure definition has an invalid name", ErrStructure, expectedName)
	}
	if def.ByteLength < 0 {
		return zero, fmt.Errorf("%w: %s structure byteLength must be non-negative", ErrStructure, def.Name)
	}
	if int(def.ByteLength) > maxClassicStructureByteLength {
		return zero, fmt.Errorf("%w: %s structure byteLength exceeds %d", ErrStructure, def.Name, maxClassicStructureByteLength)
	}
	if len(def.Fields) > maxClassicStructureFields {
		return zero, fmt.Errorf("%w: %s structure field count exceeds %d", ErrStructure, def.Name, maxClassicStructureFields)
	}
	names := map[string]bool{}
	var previousEnd int32
	fields := make([]rfctypes.RfcStructureField, 0, len(def.Fields))
	for index, field := range def.Fields {
		if field.TableName != def.Name || field.FieldName == "" || names[field.FieldName] ||
			field.Position != int32(index+1) || field.Offset < previousEnd || field.InternalLength < 0 || field.Decimals < 0 {
			return zero, fmt.Errorf("%w: %s structure field %d has invalid geometry", ErrStructure, def.Name, index)
		}
		end := field.Offset + field.InternalLength
		if end > def.ByteLength {
			return zero, fmt.Errorf("%w: %s.%s exceeds the structure byteLength", ErrStructure, def.Name, field.FieldName)
		}
		names[field.FieldName] = true
		previousEnd = end
		fields = append(fields, field)
	}
	return rfctypes.RfcStructureDefinition{Name: def.Name, ByteLength: def.ByteLength, Fields: fields}, nil
}

func fieldPath(def rfctypes.RfcStructureDefinition, field rfctypes.RfcStructureField) string {
	return def.Name + "." + field.FieldName
}

func characterLength(field rfctypes.RfcStructureField, path string) (int, error) {
	if field.InternalLength&1 != 0 {
		return 0, fmt.Errorf("%w: %s Unicode character width must be even", ErrStructure, path)
	}
	return int(field.InternalLength) / 2, nil
}

func assertFieldCodec(field rfctypes.RfcStructureField, path string) error {
	if value.IsClassicTemporalExid(field.Exid) {
		bl, _ := value.ClassicTemporalByteLength(value.TemporalExid(field.Exid))
		if int(field.InternalLength) != bl {
			return fmt.Errorf("%w: %s compact temporal type %s must occupy %d bytes", ErrStructure, path, field.Exid, bl)
		}
		return nil
	}
	switch field.Exid {
	case "C", "N":
		_, err := characterLength(field, path)
		return err
	case "D":
		return exactLen(field, 16, "DATE", "Unicode bytes", path)
	case "T":
		return exactLen(field, 12, "TIME", "Unicode bytes", path)
	case "X":
		return nil
	case "F":
		return exactLen(field, 8, "FLOAT", "bytes", path)
	case "I":
		return exactLen(field, 4, "INT4", "bytes", path)
	case "s":
		return exactLen(field, 2, "INT2", "bytes", path)
	case "b":
		return exactLen(field, 1, "INT1", "byte", path)
	case "8":
		return exactLen(field, 8, "INT8", "bytes", path)
	case "P":
		_, err := value.EncodePackedDecimal("0", int(field.InternalLength), int(field.Decimals), path)
		return err
	case "a":
		return exactLen(field, 8, "DECF16", "bytes", path)
	case "e":
		return exactLen(field, 16, "DECF34", "bytes", path)
	case "g":
		return exactLen(field, 8, "STRING descriptor", "bytes", path)
	case "y":
		return exactLen(field, 8, "XSTRING descriptor", "bytes", path)
	default:
		return fmt.Errorf("%w: %s classic RFC type %s is not implemented", ErrStructure, path, field.Exid)
	}
}

func exactLen(field rfctypes.RfcStructureField, n int, name, unit, path string) error {
	if int(field.InternalLength) != n {
		return fmt.Errorf("%w: %s %s must occupy %d %s", ErrStructure, path, name, n, unit)
	}
	return nil
}

// ValidateCodec validates every field and the captured row geometry.
func ValidateCodec(def rfctypes.RfcStructureDefinition, requestedName string) (rfctypes.RfcStructureDefinition, error) {
	normalized, err := SnapshotDefinition(def, requestedName)
	if err != nil {
		return normalized, err
	}
	for _, field := range normalized.Fields {
		if err := assertFieldCodec(field, fieldPath(normalized, field)); err != nil {
			return rfctypes.RfcStructureDefinition{}, err
		}
	}
	return normalized, nil
}

// HasDynamicFields reports whether the structure needs the xRFC XML serializer.
func HasDynamicFields(def rfctypes.RfcStructureDefinition) (bool, error) {
	normalized, err := ValidateCodec(def, "")
	if err != nil {
		return false, err
	}
	for _, field := range normalized.Fields {
		if field.Exid == "g" || field.Exid == "y" {
			return true, nil
		}
	}
	return false, nil
}

func assertFixed(def rfctypes.RfcStructureDefinition) error {
	dynamic, err := HasDynamicFields(def)
	if err != nil {
		return err
	}
	if dynamic {
		return fmt.Errorf("%w: %s contains STRING/XSTRING fields and requires xRFC XML serialization", ErrStructure, def.Name)
	}
	return nil
}

func asInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	default:
		return 0, false
	}
}

func requireInt(v any, min, max int64, path string) (int64, error) {
	n, ok := asInt(v)
	if !ok || n < min || n > max {
		return 0, fmt.Errorf("%w: %s must be an integer in %d..%d", ErrStructure, path, min, max)
	}
	return n, nil
}

func utf16le(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(out[i*2:], u)
	}
	return out
}

func initialValue(field rfctypes.RfcStructureField, path string) (any, error) {
	if value.IsClassicTemporalExid(field.Exid) {
		return value.ClassicTemporalInitialValue(value.TemporalExid(field.Exid))
	}
	switch field.Exid {
	case "C":
		return "", nil
	case "N":
		chars, err := characterLength(field, path)
		if err != nil {
			return nil, err
		}
		return strings.Repeat("0", chars), nil
	case "D":
		return "00000000", nil
	case "T":
		return "000000", nil
	case "X":
		return []byte{}, nil
	case "F":
		return float64(0), nil
	case "I", "s", "b":
		return int(0), nil
	case "8":
		return int64(0), nil
	case "P":
		return "0", nil
	case "a", "e":
		return "0", nil
	default:
		return nil, fmt.Errorf("%w: %s classic RFC type %s is not implemented", ErrStructure, path, field.Exid)
	}
}

func writeValue(target []byte, def rfctypes.RfcStructureDefinition, field rfctypes.RfcStructureField, v any) error {
	path := fieldPath(def, field)
	offset := int(field.Offset)
	if value.IsClassicTemporalExid(field.Exid) {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%w: %s expects a string temporal value", ErrStructure, path)
		}
		enc, err := value.EncodeClassicTemporal(value.TemporalExid(field.Exid), s, path)
		if err != nil {
			return err
		}
		copy(target[offset:], enc)
		return nil
	}
	switch field.Exid {
	case "C":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%w: %s expects a string", ErrStructure, path)
		}
		chars, err := characterLength(field, path)
		if err != nil {
			return err
		}
		enc, err := classicrfc.EncodeAbapChar(s, chars)
		if err != nil {
			return err
		}
		copy(target[offset:], enc)
	case "N":
		s, ok := v.(string)
		chars, err := characterLength(field, path)
		if err != nil {
			return err
		}
		if !ok || !numericDigits.MatchString(s) || len(s) > chars {
			return fmt.Errorf("%w: %s expects at most %d decimal digits", ErrStructure, path, chars)
		}
		enc, err := classicrfc.EncodeAbapChar(strings.Repeat("0", chars-len(s))+s, chars)
		if err != nil {
			return err
		}
		copy(target[offset:], enc)
	case "D":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%w: %s expects YYYYMMDD", ErrStructure, path)
		}
		wire, err := value.ClassicDateWireText(s, path)
		if err != nil {
			return err
		}
		copy(target[offset:], utf16le(wire))
	case "T":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%w: %s expects HHMMSS", ErrStructure, path)
		}
		wire, err := value.ClassicTimeWireText(s, path)
		if err != nil {
			return err
		}
		copy(target[offset:], utf16le(wire))
	case "X":
		b, ok := v.([]byte)
		if !ok {
			return fmt.Errorf("%w: %s expects []byte", ErrStructure, path)
		}
		if len(b) > int(field.InternalLength) {
			return fmt.Errorf("%w: %s accepts at most %d bytes", ErrStructure, path, field.InternalLength)
		}
		copy(target[offset:], b)
	case "F":
		f, ok := v.(float64)
		if !ok || math.IsInf(f, 0) || math.IsNaN(f) {
			return fmt.Errorf("%w: %s expects a finite 8-byte float", ErrStructure, path)
		}
		binary.LittleEndian.PutUint64(target[offset:], math.Float64bits(f))
	case "I":
		n, err := requireInt(v, math.MinInt32, math.MaxInt32, path)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(target[offset:], uint32(int32(n)))
	case "s":
		n, err := requireInt(v, math.MinInt16, math.MaxInt16, path)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint16(target[offset:], uint16(int16(n)))
	case "b":
		n, err := requireInt(v, 0, 0xff, path)
		if err != nil {
			return err
		}
		target[offset] = byte(n)
	case "8":
		n, err := requireInt(v, math.MinInt64, math.MaxInt64, path)
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(target[offset:], uint64(n))
	case "P":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%w: %s expects a decimal string", ErrStructure, path)
		}
		enc, err := value.EncodePackedDecimal(s, int(field.InternalLength), int(field.Decimals), path)
		if err != nil {
			return err
		}
		copy(target[offset:], enc)
	case "a":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%w: %s expects a decimal string", ErrStructure, path)
		}
		enc, err := value.EncodeDecimalFloat16(s, path)
		if err != nil {
			return err
		}
		copy(target[offset:], enc)
	case "e":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%w: %s expects a decimal string", ErrStructure, path)
		}
		enc, err := value.EncodeDecimalFloat34(s, path)
		if err != nil {
			return err
		}
		copy(target[offset:], enc)
	default:
		return fmt.Errorf("%w: %s classic RFC type %s is not implemented", ErrStructure, path, field.Exid)
	}
	return nil
}

func readValue(source []byte, def rfctypes.RfcStructureDefinition, field rfctypes.RfcStructureField) (any, error) {
	path := fieldPath(def, field)
	seg := source[field.Offset : field.Offset+field.InternalLength]
	if value.IsClassicTemporalExid(field.Exid) {
		return value.DecodeClassicTemporal(value.TemporalExid(field.Exid), seg, path)
	}
	switch field.Exid {
	case "C", "N":
		chars, err := characterLength(field, path)
		if err != nil {
			return nil, err
		}
		return classicrfc.DecodeAbapChar(seg, chars)
	case "D":
		chars, _ := characterLength(field, path)
		decoded, err := classicrfc.DecodeAbapFixedChar(seg, chars)
		if err != nil {
			return nil, err
		}
		return value.ClassicDatePublicText(decoded, path)
	case "T":
		chars, _ := characterLength(field, path)
		decoded, err := classicrfc.DecodeAbapFixedChar(seg, chars)
		if err != nil {
			return nil, err
		}
		return value.ClassicTimePublicText(decoded, path)
	case "X":
		return append([]byte(nil), seg...), nil
	case "F":
		f := math.Float64frombits(binary.LittleEndian.Uint64(seg))
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, fmt.Errorf("%w: %s received a non-finite 8-byte float", ErrStructure, path)
		}
		return f, nil
	case "I":
		return int32(binary.LittleEndian.Uint32(seg)), nil
	case "s":
		return int16(binary.LittleEndian.Uint16(seg)), nil
	case "b":
		return seg[0], nil
	case "8":
		return int64(binary.LittleEndian.Uint64(seg)), nil
	case "P":
		return value.DecodePackedDecimal(seg, int(field.Decimals), path)
	case "a":
		return value.DecodeDecimalFloat16(seg, path)
	case "e":
		return value.DecodeDecimalFloat34(seg, path)
	default:
		return nil, fmt.Errorf("%w: %s classic RFC type %s is not implemented", ErrStructure, path, field.Exid)
	}
}

// Encode encodes one fixed-layout classic Unicode structure.
func Encode(def rfctypes.RfcStructureDefinition, input map[string]any) ([]byte, error) {
	normalized, err := ValidateCodec(def, "")
	if err != nil {
		return nil, err
	}
	if err := assertFixed(normalized); err != nil {
		return nil, err
	}
	fields := map[string]bool{}
	for _, f := range normalized.Fields {
		fields[f.FieldName] = true
	}
	for name := range input {
		if !fields[name] {
			return nil, fmt.Errorf("%w: %s contains unknown field %s", ErrStructure, normalized.Name, name)
		}
	}
	result := make([]byte, normalized.ByteLength)
	for index, field := range normalized.Fields {
		v, supplied := input[field.FieldName]
		if !supplied {
			v, err = initialValue(field, fieldPath(normalized, field))
			if err != nil {
				return nil, err
			}
		}
		if err := writeValue(result, normalized, field, v); err != nil {
			return nil, err
		}
		if field.Exid == "C" || field.Exid == "N" {
			fieldEnd := int(field.Offset + field.InternalLength)
			nextOffset := int(normalized.ByteLength)
			if index+1 < len(normalized.Fields) {
				nextOffset = int(normalized.Fields[index+1].Offset)
			}
			paddingLength := nextOffset - fieldEnd
			if paddingLength&1 != 0 {
				return nil, fmt.Errorf("%w: %s has an odd Unicode alignment tail", ErrStructure, fieldPath(normalized, field))
			}
			if paddingLength > 0 {
				fill := " "
				if field.Exid == "N" {
					fill = "0"
				}
				copy(result[fieldEnd:], utf16le(strings.Repeat(fill, paddingLength/2)))
			}
		}
	}
	return result, nil
}

// Decode decodes one fixed-layout classic Unicode structure into plain values.
func Decode(def rfctypes.RfcStructureDefinition, value_ []byte) (map[string]any, error) {
	normalized, err := ValidateCodec(def, "")
	if err != nil {
		return nil, err
	}
	if err := assertFixed(normalized); err != nil {
		return nil, err
	}
	if len(value_) != int(normalized.ByteLength) {
		return nil, fmt.Errorf("%w: %s structure must contain exactly %d bytes; received %d", ErrStructure, normalized.Name, normalized.ByteLength, len(value_))
	}
	source := append([]byte(nil), value_...)
	result := make(map[string]any, len(normalized.Fields))
	for _, field := range normalized.Fields {
		v, err := readValue(source, normalized, field)
		if err != nil {
			return nil, err
		}
		result[field.FieldName] = v
	}
	return result, nil
}
