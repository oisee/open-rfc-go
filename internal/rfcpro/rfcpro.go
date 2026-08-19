// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/protocol/rfcpro.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown RangeErrors
// became returned, wrapped sentinel errors; the tag argument is a uint16, so
// its 0..65535 guard is enforced by the type rather than at run time; the
// optional decode options object became an explicit DecodeFieldHeaderLimited
// variant, because 0 is a meaningful maxValueLength and cannot double as
// "unset". See docs/provenance.md.

// Package rfcpro encodes and decodes the RFCPRO tag/length field header that
// prefixes every parameter value in the classic RFC conversation.
//
// The header is four bytes when the value fits in a uint16, and eight when it
// does not: a compact length of 0xffff is a sentinel meaning "an int32 length
// follows". Decoding never allocates the advertised value; it reports the
// length so the caller can apply its own allocation policy first.
package rfcpro

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// ExtendedLengthSentinel is the compact-length value that signals an
	// int32 extended length follows instead.
	ExtendedLengthSentinel = 0xffff

	// CompactLengthMax is the largest length expressible in the compact,
	// four-byte header (one below the sentinel).
	CompactLengthMax = ExtendedLengthSentinel - 1

	// ValueLengthMax is the largest length the extended header can carry: the
	// field is a signed int32 and negative lengths are rejected.
	ValueLengthMax = 0x7fff_ffff
)

// LengthEncoding names which of the two header forms carried a length.
type LengthEncoding string

const (
	// EncodingCompact is the four-byte header.
	EncodingCompact LengthEncoding = "compact"
	// EncodingExtended is the eight-byte header.
	EncodingExtended LengthEncoding = "extended"
)

var (
	// ErrRange reports a tag, length, or limit outside its permitted interval.
	ErrRange = errors.New("rfcpro: value out of range")

	// ErrShortHeader reports a header prefix too short to decode.
	ErrShortHeader = errors.New("rfcpro: short header")

	// ErrNegativeLength reports a decoded extended length that is negative.
	ErrNegativeLength = errors.New("rfcpro: negative extended length")

	// ErrLimitExceeded reports a decoded length above the configured maximum.
	ErrLimitExceeded = errors.New("rfcpro: length exceeds configured limit")
)

// FieldHeader is a decoded RFCPRO tag/length header.
type FieldHeader struct {
	Tag           uint16
	Length        int
	Encoding      LengthEncoding
	BytesConsumed int // 4 or 8
}

func validateValueLength(length int, field string) error {
	if length < 0 || length > ValueLengthMax {
		return fmt.Errorf("%w: %s must be an integer in 0..%d, got %d", ErrRange, field, ValueLengthMax, length)
	}
	return nil
}

// FieldHeaderByteLength reports whether a value of the given length needs the
// four- or eight-byte header, without allocating the value.
func FieldHeaderByteLength(length int) (int, error) {
	if err := validateValueLength(length, "RFCPRO length"); err != nil {
		return 0, err
	}
	if length <= CompactLengthMax {
		return 4, nil
	}
	return 8, nil
}

// EncodeFieldHeader encodes the canonical tag/length header for a value of the
// given length. It never allocates space for the value itself.
func EncodeFieldHeader(tag uint16, length int) ([]byte, error) {
	byteLength, err := FieldHeaderByteLength(length)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, byteLength)
	binary.BigEndian.PutUint16(buf[0:], tag)
	if byteLength == 4 {
		binary.BigEndian.PutUint16(buf[2:], uint16(length))
	} else {
		binary.BigEndian.PutUint16(buf[2:], ExtendedLengthSentinel)
		binary.BigEndian.PutUint32(buf[4:], uint32(int32(length)))
	}
	return buf, nil
}

// DecodeFieldHeader decodes a header prefix, applying the default maximum value
// length (ValueLengthMax) as the allocation limit.
func DecodeFieldHeader(data []byte) (FieldHeader, error) {
	return DecodeFieldHeaderLimited(data, ValueLengthMax)
}

// DecodeFieldHeaderLimited decodes a header prefix and rejects any advertised
// length above maxValueLength, before the caller reads or allocates the value.
// It reads at most the eight-byte header and ignores trailing bytes.
func DecodeFieldHeaderLimited(data []byte, maxValueLength int) (FieldHeader, error) {
	if err := validateValueLength(maxValueLength, "maxValueLength"); err != nil {
		return FieldHeader{}, err
	}

	// Bound the view to the fixed-size header so decoding stays independent of
	// any advertised value or trailing data that follows it.
	n := len(data)
	if n > 8 {
		n = 8
	}
	buf := data[:n]

	if len(buf) < 2 {
		return FieldHeader{}, fmt.Errorf("%w: RFCPRO field header.tag: need 2 bytes at offset 0; %d remain", ErrShortHeader, len(buf))
	}
	tag := binary.BigEndian.Uint16(buf[0:])

	if len(buf) < 4 {
		return FieldHeader{}, fmt.Errorf("%w: RFCPRO field header.length: need 2 bytes at offset 2; %d remain", ErrShortHeader, len(buf)-2)
	}
	compactLength := int(binary.BigEndian.Uint16(buf[2:]))
	if compactLength != ExtendedLengthSentinel {
		if compactLength > maxValueLength {
			return FieldHeader{}, fmt.Errorf("%w: RFCPRO length %d exceeds configured limit %d", ErrLimitExceeded, compactLength, maxValueLength)
		}
		return FieldHeader{Tag: tag, Length: compactLength, Encoding: EncodingCompact, BytesConsumed: 4}, nil
	}

	if len(buf) < 8 {
		return FieldHeader{}, fmt.Errorf("%w: RFCPRO field header.extendedLength: need 4 bytes at offset 4; %d remain", ErrShortHeader, len(buf)-4)
	}
	extendedLength := int(int32(binary.BigEndian.Uint32(buf[4:])))
	if extendedLength < 0 {
		return FieldHeader{}, fmt.Errorf("%w: RFCPRO extended length %d is negative", ErrNegativeLength, extendedLength)
	}
	if extendedLength > maxValueLength {
		return FieldHeader{}, fmt.Errorf("%w: RFCPRO length %d exceeds configured limit %d", ErrLimitExceeded, extendedLength, maxValueLength)
	}
	return FieldHeader{Tag: tag, Length: extendedLength, Encoding: EncodingExtended, BytesConsumed: 8}, nil
}
