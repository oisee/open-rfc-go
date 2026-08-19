// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/protocol/rfc-error-envelope.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown
// RfcErrorEnvelopeProtocolError became a returned *ProtocolError carrying the
// same reason code; the RfcErrorTag enum became typed uint16 constants; sets
// became map[uint16]bool; optional decode options became pointer fields; the
// per-field type guards (non-object, non-Uint8Array, out-of-uint16 tag) are
// dropped as unrepresentable in Go's type system. See docs/provenance.md.

// Package rfcerr normalizes and classifies the error/control portion of a
// decoded classic RFC response envelope.
//
// The caller owns the outer CPIC prefix, the closing-tag grammar, and the
// two-byte trailer; this package takes the already-split fields and decides
// whether they describe success, a declared ABAP exception, an ABAP runtime
// error, or an ABAP MESSAGE — rejecting anything ambiguous.
package rfcerr

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf16"
)

// Tag is an error-related RFCPRO identifier in a classic RFC response envelope.
type Tag uint16

const (
	TagExceptionKey       Tag = 0x0401
	TagErrorMessage       Tag = 0x0402
	TagRuntimeID          Tag = 0x0403
	TagT100Text           Tag = 0x0404
	TagMessageV1          Tag = 0x0411
	TagMessageV2          Tag = 0x0412
	TagMessageV3          Tag = 0x0413
	TagMessageV4          Tag = 0x0414
	TagMessageClass       Tag = 0x0415
	TagMessageType        Tag = 0x0416
	TagMessageNumber      Tag = 0x0417
	TagCallStack          Tag = 0x0418
	TagUnresolved0420     Tag = 0x0420
	TagUseClassExceptions Tag = 0x0421
	TagClassExceptionInfo Tag = 0x0422
	TagClassException     Tag = 0x0423
	TagClassExceptionEnd  Tag = 0x0424
)

const (
	// EndTag terminates a response envelope.
	EndTag = 0xffff

	DefaultMaxTextByteLength         = 1024 * 1024
	DefaultMaxTotalTextByteLength    = 4 * 1024 * 1024
	DefaultMaxControlByteLength      = 4 * 1024
	DefaultMaxTotalControlByteLength = 64 * 1024
	DefaultMaxControlCount           = 64
	DefaultMaxEnvelopeFieldCount     = 256
)

var classicResponseDataTags = map[uint16]bool{
	0x0102: true, 0x0201: true, 0x0203: true, 0x0205: true, 0x0301: true,
	0x0302: true, 0x0303: true, 0x0304: true, 0x0502: true, 0x0503: true,
	0x0512: true, 0x0514: true,
}

var textTags = map[uint16]bool{
	uint16(TagExceptionKey): true, uint16(TagErrorMessage): true, uint16(TagRuntimeID): true,
	uint16(TagT100Text): true, uint16(TagMessageV1): true, uint16(TagMessageV2): true,
	uint16(TagMessageV3): true, uint16(TagMessageV4): true, uint16(TagMessageClass): true,
	uint16(TagMessageType): true, uint16(TagMessageNumber): true, uint16(TagCallStack): true,
}

var classExceptionTags = map[uint16]bool{
	uint16(TagClassExceptionInfo): true, uint16(TagClassException): true, uint16(TagClassExceptionEnd): true,
}

var messageTextTags = []Tag{TagErrorMessage, TagT100Text}
var messageIdentityTags = []Tag{TagMessageClass, TagMessageType, TagMessageNumber}

var secondaryErrorTags = map[uint16]bool{
	uint16(TagMessageV1): true, uint16(TagMessageV2): true, uint16(TagMessageV3): true,
	uint16(TagMessageV4): true, uint16(TagCallStack): true,
	uint16(TagErrorMessage): true, uint16(TagT100Text): true,
	uint16(TagMessageClass): true, uint16(TagMessageType): true, uint16(TagMessageNumber): true,
}

// Field is one decoded RFCPRO response field.
type Field struct {
	Tag   uint16
	Value []byte
}

// FactProvenance records where a fact was found and how long it was.
type FactProvenance struct {
	Tag        uint16
	Ordinal    int
	ByteLength int
}

// UnresolvedControlFact is a retained 0x0420 control value, as hex.
type UnresolvedControlFact struct {
	Tag        Tag
	Ordinal    int
	ByteLength int
	ValueHex   string
}

// RemoteErrorFacts holds the decoded backend error facts.
type RemoteErrorFacts struct {
	ExceptionKey   string
	PlainText      string
	RuntimeID      string
	T100Text       string
	MessageClass   string
	MessageType    string
	MessageNumber  string
	MessageV1      string
	MessageV2      string
	MessageV3      string
	MessageV4      string
	CallStack      string
	Provenance     []FactProvenance
	Unresolved0420 []UnresolvedControlFact
}

// Outcome classifies a response envelope.
type Outcome string

const (
	OutcomeSuccess       Outcome = "success"
	OutcomeAbapException Outcome = "abapException"
	OutcomeAbapRuntime   Outcome = "abapRuntime"
	OutcomeAbapMessage   Outcome = "abapMessage"
)

// SuccessControl records whether the zero 0x0420 success control was present.
type SuccessControl string

const (
	SuccessControlZero          SuccessControl = "zeroControl"
	SuccessControlNotApplicable SuccessControl = "notApplicable"
)

// Envelope is the classified result of decoding a response envelope.
type Envelope struct {
	Outcome        Outcome
	SuccessControl SuccessControl
	Facts          RemoteErrorFacts
}

// DecodeOptions tunes the decoder; a nil field takes the documented default.
type DecodeOptions struct {
	MaxTextByteLength         *int
	MaxTotalTextByteLength    *int
	MaxControlByteLength      *int
	MaxTotalControlByteLength *int
	MaxControlCount           *int
	MaxFieldCount             *int
	AdditionalAllowedTags     []uint16
}

// ProtocolError reports a structurally invalid or ambiguous envelope, carrying
// a stable reason code.
type ProtocolError struct {
	ReasonCode string
	Msg        string
}

func (e *ProtocolError) Error() string { return e.Msg }

func protocolError(reasonCode, msg string) error {
	return &ProtocolError{ReasonCode: reasonCode, Msg: msg}
}

func tagText(tag uint16) string { return fmt.Sprintf("0x%04x", tag) }

func orInt(p *int, d int) int {
	if p != nil {
		return *p
	}
	return d
}

func boundedInteger(value, minimum, maximum int, field string) error {
	if value < minimum || value > maximum {
		return fmt.Errorf("rfcerr: %s must be an integer in %d..%d", field, minimum, maximum)
	}
	return nil
}

func decodeStrictUTF16LE(value []byte, tag uint16, maximumByteLength int) (string, error) {
	if len(value) > maximumByteLength {
		return "", protocolError("RFC_ERROR_ENVELOPE_TEXT_TOO_LARGE", fmt.Sprintf("RFCPRO error fact %s exceeds the configured text limit", tagText(tag)))
	}
	if len(value)&1 != 0 {
		return "", protocolError("RFC_ERROR_ENVELOPE_ODD_UTF16_LENGTH", fmt.Sprintf("RFCPRO error fact %s has an odd UTF-16LE byte length", tagText(tag)))
	}
	units := make([]uint16, 0, len(value)/2)
	for offset := 0; offset < len(value); offset += 2 {
		unit := uint16(value[offset]) | uint16(value[offset+1])<<8
		if unit == 0 {
			return "", protocolError("RFC_ERROR_ENVELOPE_EMBEDDED_NUL", fmt.Sprintf("RFCPRO error fact %s contains NUL", tagText(tag)))
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			if offset+2 >= len(value) {
				return "", protocolError("RFC_ERROR_ENVELOPE_UNPAIRED_SURROGATE", fmt.Sprintf("RFCPRO error fact %s ends with an unpaired surrogate", tagText(tag)))
			}
			low := uint16(value[offset+2]) | uint16(value[offset+3])<<8
			if low < 0xdc00 || low > 0xdfff {
				return "", protocolError("RFC_ERROR_ENVELOPE_UNPAIRED_SURROGATE", fmt.Sprintf("RFCPRO error fact %s contains an unpaired surrogate", tagText(tag)))
			}
			units = append(units, unit, low)
			offset += 2
		} else if unit >= 0xdc00 && unit <= 0xdfff {
			return "", protocolError("RFC_ERROR_ENVELOPE_UNPAIRED_SURROGATE", fmt.Sprintf("RFCPRO error fact %s contains an unpaired surrogate", tagText(tag)))
		} else {
			units = append(units, unit)
		}
	}
	return strings.TrimRight(string(utf16.Decode(units)), " "), nil
}

func hasNonEmptyFact(tags []Tag, present map[uint16]bool, values map[uint16]string) bool {
	for _, tag := range tags {
		if present[uint16(tag)] && len(values[uint16(tag)]) > 0 {
			return true
		}
	}
	return false
}

func hasCoherentMessageIdentity(present map[uint16]bool, values map[uint16]string) bool {
	for _, tag := range messageIdentityTags {
		if !present[uint16(tag)] || len(values[uint16(tag)]) == 0 {
			return false
		}
	}
	return true
}

func hasAnyTag(tags map[uint16]bool, present map[uint16]bool) bool {
	for tag := range tags {
		if present[tag] {
			return true
		}
	}
	return false
}

// Decode normalizes and classifies the error/control portion of a response.
func Decode(fields []Field, options DecodeOptions) (Envelope, error) {
	var zero Envelope
	maxTextByteLength := orInt(options.MaxTextByteLength, DefaultMaxTextByteLength)
	maxTotalTextByteLength := orInt(options.MaxTotalTextByteLength, DefaultMaxTotalTextByteLength)
	maxControlByteLength := orInt(options.MaxControlByteLength, DefaultMaxControlByteLength)
	maxTotalControlByteLength := orInt(options.MaxTotalControlByteLength, DefaultMaxTotalControlByteLength)
	maxControlCount := orInt(options.MaxControlCount, DefaultMaxControlCount)
	maxFieldCount := orInt(options.MaxFieldCount, DefaultMaxEnvelopeFieldCount)
	for _, c := range []struct {
		v, lo, hi int
		name      string
	}{
		{maxTextByteLength, 0, 0x7fff_ffff, "maxTextByteLength"},
		{maxTotalTextByteLength, 0, 0x7fff_ffff, "maxTotalTextByteLength"},
		{maxControlByteLength, 0, 0x7fff_ffff, "maxControlByteLength"},
		{maxTotalControlByteLength, 0, 0x7fff_ffff, "maxTotalControlByteLength"},
		{maxControlCount, 0, 0x7fff_ffff, "maxControlCount"},
		{maxFieldCount, 1, 0x7fff_ffff, "maxFieldCount"},
	} {
		if err := boundedInteger(c.v, c.lo, c.hi, c.name); err != nil {
			return zero, err
		}
	}

	allowedTags := map[uint16]bool{}
	for k := range classicResponseDataTags {
		allowedTags[k] = true
	}
	for _, tag := range options.AdditionalAllowedTags {
		allowedTags[tag] = true
	}

	if len(fields) > maxFieldCount {
		return zero, protocolError("RFC_ERROR_ENVELOPE_TOO_MANY_FIELDS", "RFCPRO response exceeds the configured envelope field-count limit")
	}
	var endIndices []int
	for ordinal, field := range fields {
		if field.Tag == EndTag {
			endIndices = append(endIndices, ordinal)
		}
	}
	if len(endIndices) == 0 {
		return zero, protocolError("RFC_ERROR_ENVELOPE_MISSING_END", "RFCPRO response lacks its terminal End field")
	}
	endIndex := endIndices[0]
	if len(endIndices) != 1 || endIndex != len(fields)-1 || len(fields[endIndex].Value) != 0 {
		return zero, protocolError("RFC_ERROR_ENVELOPE_INVALID_END", "RFCPRO response End field must occur once, last, with zero length")
	}

	values := map[uint16]string{}
	present := map[uint16]bool{}
	var provenance []FactProvenance
	var unresolved0420 []UnresolvedControlFact
	totalTextByteLength := 0
	totalControlByteLength := 0
	controlCount := 0
	sawUseClassExceptions := false
	sawSupplementalClassExceptionInfo := false

	for ordinal := 0; ordinal < endIndex; ordinal++ {
		field := fields[ordinal]
		tag := field.Tag
		value := field.Value

		switch {
		case textTags[tag]:
			if present[tag] {
				return zero, protocolError("RFC_ERROR_ENVELOPE_DUPLICATE_FACT", fmt.Sprintf("RFCPRO response repeats singleton error fact %s", tagText(tag)))
			}
			if len(value) > maxTextByteLength {
				return zero, protocolError("RFC_ERROR_ENVELOPE_TEXT_TOO_LARGE", fmt.Sprintf("RFCPRO error fact %s exceeds the configured text limit", tagText(tag)))
			}
			if totalTextByteLength > maxTotalTextByteLength-len(value) {
				return zero, protocolError("RFC_ERROR_ENVELOPE_TOTAL_TEXT_TOO_LARGE", "RFCPRO error facts exceed the configured aggregate text limit")
			}
			totalTextByteLength += len(value)
			present[tag] = true
			decoded, err := decodeStrictUTF16LE(value, tag, maxTextByteLength)
			if err != nil {
				return zero, err
			}
			values[tag] = decoded
			provenance = append(provenance, FactProvenance{Tag: tag, Ordinal: ordinal, ByteLength: len(value)})

		case tag == uint16(TagUnresolved0420):
			if controlCount >= maxControlCount {
				return zero, protocolError("RFC_ERROR_ENVELOPE_TOO_MANY_CONTROLS", "RFCPRO response exceeds the configured control-count limit")
			}
			if len(value) > maxControlByteLength {
				return zero, protocolError("RFC_ERROR_ENVELOPE_CONTROL_TOO_LARGE", "unresolved RFCPRO control 0x0420 exceeds the configured limit")
			}
			if totalControlByteLength > maxTotalControlByteLength-len(value) {
				return zero, protocolError("RFC_ERROR_ENVELOPE_TOTAL_CONTROL_TOO_LARGE", "RFCPRO response controls exceed the configured aggregate byte limit")
			}
			controlCount++
			totalControlByteLength += len(value)
			unresolved0420 = append(unresolved0420, UnresolvedControlFact{
				Tag: TagUnresolved0420, Ordinal: ordinal, ByteLength: len(value), ValueHex: hex.EncodeToString(value),
			})
			provenance = append(provenance, FactProvenance{Tag: tag, Ordinal: ordinal, ByteLength: len(value)})

		case tag == uint16(TagUseClassExceptions):
			if controlCount >= maxControlCount {
				return zero, protocolError("RFC_ERROR_ENVELOPE_TOO_MANY_CONTROLS", "RFCPRO response exceeds the configured control-count limit")
			}
			if sawUseClassExceptions {
				return zero, protocolError("RFC_ERROR_ENVELOPE_DUPLICATE_FACT", "RFCPRO response repeats singleton class-exception control 0x0421")
			}
			if len(value) > maxControlByteLength {
				return zero, protocolError("RFC_ERROR_ENVELOPE_CONTROL_TOO_LARGE", "RFCPRO class-exception control exceeds the configured limit")
			}
			if totalControlByteLength > maxTotalControlByteLength-len(value) {
				return zero, protocolError("RFC_ERROR_ENVELOPE_TOTAL_CONTROL_TOO_LARGE", "RFCPRO response controls exceed the configured aggregate byte limit")
			}
			controlCount++
			totalControlByteLength += len(value)
			sawUseClassExceptions = true
			provenance = append(provenance, FactProvenance{Tag: tag, Ordinal: ordinal, ByteLength: len(value)})

		case tag == uint16(TagClassExceptionInfo):
			if controlCount >= maxControlCount {
				return zero, protocolError("RFC_ERROR_ENVELOPE_TOO_MANY_CONTROLS", "RFCPRO response exceeds the configured control-count limit")
			}
			if sawSupplementalClassExceptionInfo {
				return zero, protocolError("RFC_ERROR_ENVELOPE_DUPLICATE_FACT", "RFCPRO response repeats singleton class-exception info 0x0422")
			}
			if len(value) > maxControlByteLength {
				return zero, protocolError("RFC_ERROR_ENVELOPE_CONTROL_TOO_LARGE", "RFCPRO class-exception info exceeds the configured limit")
			}
			if totalControlByteLength > maxTotalControlByteLength-len(value) {
				return zero, protocolError("RFC_ERROR_ENVELOPE_TOTAL_CONTROL_TOO_LARGE", "RFCPRO response controls exceed the configured aggregate byte limit")
			}
			controlCount++
			totalControlByteLength += len(value)
			sawSupplementalClassExceptionInfo = true
			provenance = append(provenance, FactProvenance{Tag: tag, Ordinal: ordinal, ByteLength: len(value)})

		case classExceptionTags[tag]:
			return zero, protocolError("RFC_ERROR_ENVELOPE_CLASS_EXCEPTION_UNSUPPORTED", fmt.Sprintf("RFCPRO class-exception fact %s is not supported", tagText(tag)))

		case !allowedTags[tag]:
			return zero, protocolError("RFC_ERROR_ENVELOPE_UNKNOWN_TAG", fmt.Sprintf("RFCPRO response contains unknown tag %s", tagText(tag)))
		}
	}

	exceptionKey := values[uint16(TagExceptionKey)]
	runtimeID := values[uint16(TagRuntimeID)]
	if present[uint16(TagExceptionKey)] && len(exceptionKey) == 0 {
		return zero, protocolError("RFC_ERROR_ENVELOPE_EMPTY_DISCRIMINATOR", "RFCPRO declared-exception key is empty")
	}
	if present[uint16(TagRuntimeID)] && len(runtimeID) == 0 {
		return zero, protocolError("RFC_ERROR_ENVELOPE_EMPTY_DISCRIMINATOR", "RFCPRO runtime identifier is empty")
	}
	if present[uint16(TagExceptionKey)] && present[uint16(TagRuntimeID)] {
		return zero, protocolError("RFC_ERROR_ENVELOPE_CONFLICTING_DISCRIMINATORS", "RFCPRO response contains both declared-exception and runtime identifiers")
	}
	if sawSupplementalClassExceptionInfo && (!present[uint16(TagExceptionKey)] || sawUseClassExceptions) {
		return zero, protocolError("RFC_ERROR_ENVELOPE_CLASS_EXCEPTION_UNSUPPORTED", "RFCPRO class-exception info is only supported as supplemental data for a classic declared exception")
	}

	outcome := OutcomeSuccess
	successControl := SuccessControlNotApplicable
	switch {
	case present[uint16(TagExceptionKey)]:
		outcome = OutcomeAbapException
	case present[uint16(TagRuntimeID)]:
		outcome = OutcomeAbapRuntime
	case hasNonEmptyFact(messageTextTags, present, values) || hasCoherentMessageIdentity(present, values):
		outcome = OutcomeAbapMessage
	case hasAnyTag(secondaryErrorTags, present):
		return zero, protocolError("RFC_ERROR_ENVELOPE_AMBIGUOUS_FACTS", "RFCPRO response contains secondary error facts without a discriminator")
	default:
		if len(unresolved0420) != 1 || unresolved0420[0].ByteLength != 4 || unresolved0420[0].ValueHex != "00000000" {
			return zero, protocolError("RFC_ERROR_ENVELOPE_UNRESOLVED_SUCCESS_CONTROL", "RFCPRO response lacks the zero 0x0420 success control")
		}
		outcome = OutcomeSuccess
		successControl = SuccessControlZero
	}

	facts := RemoteErrorFacts{
		ExceptionKey:   exceptionKey,
		PlainText:      values[uint16(TagErrorMessage)],
		RuntimeID:      runtimeID,
		T100Text:       values[uint16(TagT100Text)],
		MessageClass:   values[uint16(TagMessageClass)],
		MessageType:    values[uint16(TagMessageType)],
		MessageNumber:  values[uint16(TagMessageNumber)],
		MessageV1:      values[uint16(TagMessageV1)],
		MessageV2:      values[uint16(TagMessageV2)],
		MessageV3:      values[uint16(TagMessageV3)],
		MessageV4:      values[uint16(TagMessageV4)],
		CallStack:      values[uint16(TagCallStack)],
		Provenance:     provenance,
		Unresolved0420: unresolved0420,
	}
	return Envelope{Outcome: outcome, SuccessControl: successControl, Facts: facts}, nil
}
