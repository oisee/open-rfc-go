// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/values/packed-decimal.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. The PackedDecimalInput
// union (string | number | bigint | {toString}) collapses to a decimal string
// argument — Go callers format their own value, so decimalText's coercion is
// dropped; thrown TypeError/RangeError became returned, wrapped errors; the
// intrinsic-geometry snapshot collapses to len()/copy. See docs/provenance.md.

// Package value holds the classic RFC scalar value codecs: packed decimal,
// decimal float, temporal, INT8, BCD projection, and Unicode-scalar text.
package value

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const maxPackedDecimalTextLength = 4_096

var (
	// ErrPackedRange reports a value or geometry outside the packed-decimal
	// contract.
	ErrPackedRange = errors.New("value: packed decimal out of range")
	// ErrPackedDecode reports a malformed packed-decimal byte record.
	ErrPackedDecode = errors.New("value: packed decimal decode")
)

var decimalPattern = regexp.MustCompile(`^([+-]?)(?:(\d+)(?:\.(\d*))?|\.(\d+))(?:[eE]([+-]?\d+))?$`)

type scaled struct {
	digits   string
	negative bool
}

func scaledDigits(text string, decimals, capacity int, path string) (scaled, error) {
	m := decimalPattern.FindStringSubmatch(text)
	if m == nil {
		return scaled{}, fmt.Errorf("%w: %s is not a decimal value", ErrPackedRange, path)
	}
	integer := m[2]
	fraction := m[3]
	if fraction == "" {
		fraction = m[4]
	}
	coefficient := integer + fraction
	nonzero := strings.ContainsAny(coefficient, "123456789")

	var exponent int
	exponentOK := true
	if m[5] != "" {
		e, err := strconv.Atoi(m[5])
		if err != nil {
			exponentOK = false
			// Preserve the sign of the overflowing exponent for the branch below.
			if strings.HasPrefix(m[5], "-") {
				exponent = -1
			} else {
				exponent = 1
			}
		} else {
			exponent = e
		}
	}
	if !exponentOK {
		if !nonzero {
			return scaled{digits: "0"}, nil
		}
		if exponent > 0 {
			return scaled{}, fmt.Errorf("%w: %s exceeds its %d-digit packed capacity", ErrPackedRange, path, capacity)
		}
		return scaled{}, fmt.Errorf("%w: %s has more than %d fractional digits", ErrPackedRange, path, decimals)
	}

	shift := exponent + decimals - len(fraction)
	var scaledText string
	if shift >= 0 {
		significantLength := len(strings.TrimLeft(coefficient, "0"))
		if significantLength+shift > capacity {
			return scaled{}, fmt.Errorf("%w: %s exceeds its %d-digit packed capacity", ErrPackedRange, path, capacity)
		}
		scaledText = coefficient + strings.Repeat("0", shift)
	} else {
		removed := -shift
		split := len(coefficient) - removed
		if split < 0 {
			split = 0
		}
		if strings.ContainsAny(coefficient[split:], "123456789") {
			return scaled{}, fmt.Errorf("%w: %s has more than %d fractional digits", ErrPackedRange, path, decimals)
		}
		scaledText = coefficient[:split]
		if scaledText == "" {
			scaledText = "0"
		}
	}

	significant := trimLeadingZeros(scaledText)
	if len(significant) > capacity {
		return scaled{}, fmt.Errorf("%w: %s exceeds its %d-digit packed capacity", ErrPackedRange, path, capacity)
	}
	return scaled{
		digits:   significant,
		negative: m[1] == "-" && strings.ContainsAny(significant, "123456789"),
	}, nil
}

// trimLeadingZeros removes leading zeros but keeps at least one digit.
func trimLeadingZeros(s string) string {
	i := 0
	for i < len(s)-1 && s[i] == '0' {
		i++
	}
	return s[i:]
}

func packedGeometry(byteLength, decimals int, path string) (int, error) {
	if byteLength < 1 || byteLength > 16 {
		return 0, fmt.Errorf("%w: %s packed length must be an integer in 1..16", ErrPackedRange, path)
	}
	digits := byteLength*2 - 1
	maximumDecimals := 14
	if digits < maximumDecimals {
		maximumDecimals = digits
	}
	if decimals < 0 || decimals > maximumDecimals {
		return 0, fmt.Errorf("%w: %s decimals must be an integer in 0..%d", ErrPackedRange, path, maximumDecimals)
	}
	return digits, nil
}

// EncodePackedDecimal encodes an ABAP TYPE P packed BCD value with a trailing
// C/D sign nibble. The value is a decimal string.
func EncodePackedDecimal(value string, byteLength, decimals int, path string) ([]byte, error) {
	if path == "" {
		path = "packed decimal"
	}
	capacity, err := packedGeometry(byteLength, decimals, path)
	if err != nil {
		return nil, err
	}
	if value == "" {
		value = "0"
	}
	if len(value) > maxPackedDecimalTextLength {
		return nil, fmt.Errorf("%w: %s decimal text exceeds %d characters", ErrPackedRange, path, maxPackedDecimalTextLength)
	}
	s, err := scaledDigits(value, decimals, capacity, path)
	if err != nil {
		return nil, err
	}
	digits := padStartZeros(s.digits, capacity)
	sign := "C"
	if s.negative {
		sign = "D"
	}
	nibbles := digits + sign
	out := make([]byte, byteLength)
	for i := 0; i < byteLength; i++ {
		b, err := strconv.ParseUint(nibbles[i*2:i*2+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("%w: %s internal nibble error", ErrPackedRange, path)
		}
		out[i] = byte(b)
	}
	return out, nil
}

func padStartZeros(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
}

var decimalDigitsOnly = regexp.MustCompile(`^\d+$`)

// DecodePackedDecimal decodes an ABAP TYPE P value to a precision-preserving
// decimal string.
func DecodePackedDecimal(value []byte, decimals int, path string) (string, error) {
	if path == "" {
		path = "packed decimal"
	}
	capacity, err := packedGeometry(len(value), decimals, path)
	if err != nil {
		return "", err
	}
	h := strings.ToUpper(hex.EncodeToString(value))
	digits := h[:capacity]
	if !decimalDigitsOnly.MatchString(digits) {
		return "", fmt.Errorf("%w: %s contains a non-decimal digit nibble", ErrPackedDecode, path)
	}
	sign := h[len(h)-1]
	positive := sign == 'A' || sign == 'C' || sign == 'E' || sign == 'F'
	negative := sign == 'B' || sign == 'D'
	if !positive && !negative {
		return "", fmt.Errorf("%w: %s contains invalid sign nibble %c", ErrPackedDecode, path, sign)
	}

	var integerDigits string
	if decimals == 0 {
		integerDigits = digits
	} else if len(digits) > decimals {
		integerDigits = digits[:len(digits)-decimals]
	} else {
		integerDigits = "0"
	}
	integer := trimLeadingZeros(integerDigits)
	fraction := ""
	if decimals != 0 {
		fraction = digits[len(digits)-decimals:]
	}
	nonzero := strings.ContainsAny(digits, "123456789")
	prefix := ""
	if negative && nonzero {
		prefix = "-"
	}
	out := prefix + integer
	if decimals != 0 {
		out += "." + fraction
	}
	return out, nil
}
