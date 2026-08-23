// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/protocol/cpic.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go across cpic.go,
// logon.go, and function.go. The JavaScript diagnostic machinery — Symbol-keyed
// projector functions attached to the decoder, a WeakSet marking structure
// errors, and a WeakMap recording each error's parse stage — collapses to two
// typed errors, *StructureError (carrying Rule and Fields) and *ParseError
// (carrying Stage), read through StructureRuleOf / ParseStageOf. Thrown
// RangeError/Error/TypeError became returned, wrapped errors; the CpicTag enum
// became typed uint16 constants; the cleartext/password fill(0) scrubbing is
// dropped (Go cannot guarantee a []byte is never copied by the GC); the
// intrinsicUint8ArrayView geometry helper collapses to a slice. See
// docs/provenance.md.

// Package cpic encodes and decodes the classic RFC CPIC layer: the chained
// tag/length field grammar, the initial logon request/response, and the
// Unicode function request/response envelopes.
package cpic

import (
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/oisee/open-rfc-go/internal/rfcpro"
)

const (
	DefaultMaxFieldLength      = 256 * 1024 * 1024
	DefaultMaxFieldChainLength = 256 * 1024 * 1024
	DefaultMaxFieldCount       = 100_000
	// ClassicXrfcXMLChunkLength is the admitted request chunk size for classic
	// xRFC XML values.
	ClassicXrfcXMLChunkLength = 16 * 1024
)

// Tag is a CPIC field tag admitted by the bounded direct-CPIC contract.
type Tag uint16

const (
	TagDestination     Tag = 0x0006
	TagClientAddress   Tag = 0x0007
	TagPartnerHost     Tag = 0x0008
	TagKernel          Tag = 0x000b
	TagConnectionType  Tag = 0x0011
	TagKernelRelease   Tag = 0x0012
	TagKernelPatch     Tag = 0x0013
	TagPartnerSystem   Tag = 0x0018
	TagSystemCodePage  Tag = 0x0016
	TagStart           Tag = 0x0101
	TagFunction        Tag = 0x0102
	TagProtocolVersion Tag = 0x0103
	TagCapabilities    Tag = 0x0106
	TagUser            Tag = 0x0111
	TagClient          Tag = 0x0114
	TagLanguage        Tag = 0x0115
	TagPassword        Tag = 0x0117
	// TagTicket carries an SAP logon ticket in place of a password. The value is
	// the ticket's base64 text (the MYSAPSSO2 string) encoded UTF-16LE. Derived
	// clean-room from our own capture; see docs/discoveries.
	TagTicket           Tag = 0x0670
	TagProgram          Tag = 0x0130
	TagLogonStatus      Tag = 0x0161
	TagParameterName    Tag = 0x0201
	TagParameterValue   Tag = 0x0203
	TagRequestedOutput  Tag = 0x0205
	TagTableName        Tag = 0x0301
	TagTableHeader      Tag = 0x0302
	TagTableContent     Tag = 0x0303
	TagTableCompr       Tag = 0x0304
	TagUnresolved0420   Tag = 0x0420
	TagLogonMarker      Tag = 0x0337
	TagUnicodeIndicator Tag = 0x0501
	TagContextEnd       Tag = 0x0502
	TagResponseStart    Tag = 0x0500
	TagResponseContext  Tag = 0x0503
	TagCallContext      Tag = 0x0512
	TagSession          Tag = 0x0514
	TagRfcServerResetDn Tag = 0x0523
	TagXRfcParameter    Tag = 0x3c02
	TagXRfcData         Tag = 0x3c05
	TagEnd              Tag = 0xffff

	// tagAbapErrorMessage is used only in tests/preambles; kept for the error
	// preamble grammar.
	tagAbapErrorMessage Tag = 0x0402
)

// initialCpicUnresolved0450 is the six-byte successful-logon control observed
// on S/4HANA 2023.
const initialCpicUnresolved0450 = 0x0450

// Field is a CPIC field: an encode input and a decoded field share this shape.
type Field struct {
	Tag   uint16
	Value []byte
}

// FieldChainLimits bounds a decoded or encoded field chain. A nil field takes
// its documented default.
type FieldChainLimits struct {
	MaxFieldLength *int
	MaxChainLength *int
	MaxFieldCount  *int
}

type resolvedLimits struct {
	maxFieldLength int
	maxChainLength int
	maxFieldCount  int
}

// DecodedFieldChainPrefix is the result of decoding a chain or chain prefix.
type DecodedFieldChainPrefix struct {
	Fields        []Field
	BytesConsumed int
}

var (
	// ErrRange reports a value outside its permitted interval.
	ErrRange = errors.New("cpic: value out of range")
	// ErrChain reports a malformed chained field region.
	ErrChain = errors.New("cpic: malformed field chain")
	// ErrProtocol reports a structurally invalid request or response.
	ErrProtocol = errors.New("cpic: protocol violation")
)

// StructuralField is a redaction-safe record of one decoded field: its tag,
// byte length, and position, but never its value.
type StructuralField struct {
	Tag        uint16
	ByteLength int
	Index      int
}

// StructureError reports a well-formed but structurally invalid initial-logon
// response, carrying a stable rule and the redaction-safe field shape.
type StructureError struct {
	Rule   string
	Msg    string
	Fields []StructuralField
}

func (e *StructureError) Error() string { return e.Msg }

// ParseError reports where initial-logon decoding failed, carrying a stable
// stage name.
type ParseError struct {
	Stage string
	Msg   string
	Err   error
}

func (e *ParseError) Error() string { return e.Msg }
func (e *ParseError) Unwrap() error { return e.Err }

// StructureRuleOf returns the rule of a *StructureError, or "".
func StructureRuleOf(err error) string {
	var se *StructureError
	if errors.As(err, &se) {
		return se.Rule
	}
	return ""
}

// ParseStageOf returns the parse stage of an initial-logon decode failure:
// a *ParseError's stage, "structural" for a *StructureError, or "".
func ParseStageOf(err error) string {
	var pe *ParseError
	if errors.As(err, &pe) {
		return pe.Stage
	}
	var se *StructureError
	if errors.As(err, &se) {
		return "structural"
	}
	return ""
}

func failParse(stage, msg string) error {
	return &ParseError{Stage: stage, Msg: msg}
}

func wrapParse(stage string, err error) error {
	return &ParseError{Stage: stage, Msg: err.Error(), Err: err}
}

func failStructure(rule, msg string, fields []StructuralField) error {
	return &StructureError{Rule: rule, Msg: msg, Fields: fields}
}

var (
	digits3   = regexp.MustCompile(`^\d{3}$`)
	oneLetter = regexp.MustCompile(`^[A-Za-z]$`)
	asciiRE   = regexp.MustCompile(`^[\x20-\x7e]*$`)
)

func tagText(tag uint16) string { return fmt.Sprintf("0x%04x", tag) }

func orInt(p *int, d int) int {
	if p != nil {
		return *p
	}
	return d
}

func asciiBytes(value, field string, minimum, maximum int) ([]byte, error) {
	if !asciiRE.MatchString(value) || len(value) < minimum || len(value) > maximum {
		return nil, fmt.Errorf("%w: %s must contain %d..%d ASCII bytes", ErrRange, field, minimum, maximum)
	}
	return []byte(value), nil
}

func exactBytes(value []byte, length int, field string) ([]byte, error) {
	if len(value) != length {
		return nil, fmt.Errorf("%w: %s must contain exactly %d bytes; received %d", ErrRange, field, length, len(value))
	}
	return append([]byte(nil), value...), nil
}

func unicodeBytes(value, field string, maximumCharacters int) ([]byte, error) {
	units := utf16.Encode([]rune(value))
	if len(units) < 1 || len(units) > maximumCharacters || !utf8.ValidString(value) || containsNUL(value) {
		return nil, fmt.Errorf("%w: %s must contain 1..%d Unicode scalar characters without NUL", ErrRange, field, maximumCharacters)
	}
	out := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(out[i*2:], u)
	}
	return out, nil
}

func containsNUL(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return true
		}
	}
	return false
}

func utf16leBytes(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(out[i*2:], u)
	}
	return out
}

func resolveLimits(limits FieldChainLimits) (resolvedLimits, error) {
	r := resolvedLimits{
		maxFieldLength: orInt(limits.MaxFieldLength, DefaultMaxFieldLength),
		maxChainLength: orInt(limits.MaxChainLength, DefaultMaxFieldChainLength),
		maxFieldCount:  orInt(limits.MaxFieldCount, DefaultMaxFieldCount),
	}
	if r.maxFieldLength < 0 || r.maxFieldLength > rfcpro.ValueLengthMax {
		return r, fmt.Errorf("%w: maxFieldLength must be an integer in 0..%d", ErrRange, rfcpro.ValueLengthMax)
	}
	if r.maxChainLength < 0 || r.maxChainLength > rfcpro.ValueLengthMax {
		return r, fmt.Errorf("%w: maxChainLength must be an integer in 0..%d", ErrRange, rfcpro.ValueLengthMax)
	}
	if r.maxFieldCount < 0 {
		return r, fmt.Errorf("%w: maxFieldCount must be a non-negative integer", ErrRange)
	}
	return r, nil
}

// FieldChainByteLength computes the encoded length of a field chain, applying
// the limits.
func FieldChainByteLength(initialPreviousTag uint16, fields []Field, limits FieldChainLimits) (int, error) {
	r, err := resolveLimits(limits)
	if err != nil {
		return 0, err
	}
	if len(fields) > r.maxFieldCount {
		return 0, fmt.Errorf("%w: CPIC field count %d exceeds configured limit %d", ErrRange, len(fields), r.maxFieldCount)
	}
	byteLength := 0
	for _, field := range fields {
		if len(field.Value) > r.maxFieldLength {
			return 0, fmt.Errorf("%w: CPIC field length %d exceeds configured limit %d", ErrRange, len(field.Value), r.maxFieldLength)
		}
		hl, err := rfcpro.FieldHeaderByteLength(len(field.Value))
		if err != nil {
			return 0, err
		}
		byteLength += 2 + hl + len(field.Value)
		if byteLength > r.maxChainLength {
			return 0, fmt.Errorf("%w: CPIC field chain length exceeds configured limit %d", ErrRange, r.maxChainLength)
		}
	}
	return byteLength, nil
}

// EncodeFieldChain encodes CPIC's chained field grammar. Each record names the
// previous and current tag, so a decoder can detect dropped or reordered fields.
func EncodeFieldChain(initialPreviousTag uint16, fields []Field, limits FieldChainLimits) ([]byte, error) {
	byteLength, err := FieldChainByteLength(initialPreviousTag, fields, limits)
	if err != nil {
		return nil, err
	}
	encoded := make([]byte, byteLength)
	offset := 0
	previousTag := initialPreviousTag
	for _, field := range fields {
		binary.BigEndian.PutUint16(encoded[offset:], previousTag)
		offset += 2
		header, err := rfcpro.EncodeFieldHeader(field.Tag, len(field.Value))
		if err != nil {
			return nil, err
		}
		copy(encoded[offset:], header)
		offset += len(header)
		copy(encoded[offset:], field.Value)
		offset += len(field.Value)
		previousTag = field.Tag
	}
	return encoded, nil
}

// DecodeFieldChain decodes and validates a complete chained CPIC field region.
func DecodeFieldChain(data []byte, initialPreviousTag uint16, limits FieldChainLimits) ([]Field, error) {
	decoded, err := decodeFieldChainRegion(data, initialPreviousTag, -1, limits)
	if err != nil {
		return nil, err
	}
	if decoded.BytesConsumed != len(data) {
		return nil, fmt.Errorf("%w: CPIC field-chain decoder invariant failed", ErrChain)
	}
	return decoded.Fields, nil
}

// DecodeFieldChainPrefix decodes a chained field prefix through a required
// terminal tag, leaving any following protocol trailer to its own decoder.
func DecodeFieldChainPrefix(data []byte, initialPreviousTag uint16, terminalTag uint16, limits FieldChainLimits) (DecodedFieldChainPrefix, error) {
	return decodeFieldChainRegion(data, initialPreviousTag, int(terminalTag), limits)
}

func decodeFieldChainRegion(data []byte, initialPreviousTag uint16, terminalTag int, limits FieldChainLimits) (DecodedFieldChainPrefix, error) {
	var zero DecodedFieldChainPrefix
	r, err := resolveLimits(limits)
	if err != nil {
		return zero, err
	}
	if terminalTag < 0 && len(data) > r.maxChainLength {
		return zero, fmt.Errorf("%w: CPIC field chain length %d exceeds configured limit %d", ErrRange, len(data), r.maxChainLength)
	}
	var fields []Field
	expectedPreviousTag := initialPreviousTag
	offset := 0
	for offset < len(data) {
		if len(fields) >= r.maxFieldCount {
			return zero, fmt.Errorf("%w: CPIC field count exceeds configured limit %d", ErrRange, r.maxFieldCount)
		}
		if err := enforceChainLength(offset+6, r.maxChainLength); err != nil {
			return zero, err
		}
		if err := requireInput(data, offset, 6, "fieldHeader"); err != nil {
			return zero, err
		}
		previousTag := binary.BigEndian.Uint16(data[offset:])
		if previousTag != expectedPreviousTag {
			return zero, fmt.Errorf("%w: CPIC field chain expected previous tag %s; received %s", ErrChain, tagText(expectedPreviousTag), tagText(previousTag))
		}

		headerSnapshot := append([]byte(nil), data[offset+2:offset+6]...)
		if binary.BigEndian.Uint16(data[offset+4:]) == rfcpro.ExtendedLengthSentinel {
			if err := enforceChainLength(offset+10, r.maxChainLength); err != nil {
				return zero, err
			}
			if err := requireInput(data, offset+6, 4, "extendedLength"); err != nil {
				return zero, err
			}
			headerSnapshot = make([]byte, 8)
			copy(headerSnapshot, data[offset+2:offset+6])
			copy(headerSnapshot[4:], data[offset+6:offset+10])
		}
		header, err := rfcpro.DecodeFieldHeaderLimited(headerSnapshot, r.maxFieldLength)
		if err != nil {
			return zero, err
		}
		valueOffset := offset + 2 + header.BytesConsumed
		nextOffset := valueOffset + header.Length
		if err := enforceChainLength(nextOffset, r.maxChainLength); err != nil {
			return zero, err
		}
		if err := requireInput(data, valueOffset, header.Length, "value"); err != nil {
			return zero, err
		}
		fields = append(fields, Field{Tag: header.Tag, Value: append([]byte(nil), data[valueOffset:nextOffset]...)})
		offset = nextOffset
		expectedPreviousTag = header.Tag
		if terminalTag >= 0 && int(header.Tag) == terminalTag {
			return DecodedFieldChainPrefix{Fields: fields, BytesConsumed: offset}, nil
		}
	}
	if terminalTag >= 0 {
		return zero, fmt.Errorf("%w: CPIC field chain ended before terminal tag %s", ErrChain, tagText(uint16(terminalTag)))
	}
	return DecodedFieldChainPrefix{Fields: fields, BytesConsumed: offset}, nil
}

func enforceChainLength(byteLength, maximum int) error {
	if byteLength > maximum {
		return fmt.Errorf("%w: CPIC field chain length %d exceeds configured limit %d", ErrRange, byteLength, maximum)
	}
	return nil
}

func requireInput(data []byte, offset, byteLength int, field string) error {
	remaining := len(data) - offset
	if remaining < 0 {
		remaining = 0
	}
	if byteLength > remaining {
		return fmt.Errorf("%w: CPIC field chain.%s: need %d bytes at offset %d; %d remain", ErrRange, field, byteLength, offset, remaining)
	}
	return nil
}
