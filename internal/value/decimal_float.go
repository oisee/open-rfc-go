// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/values/decimal-float.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. JavaScript bigint →
// math/big.Int; the DecimalFloatInput union → a string argument (Go callers
// format number/bigint themselves, so decimalText's coercion is dropped);
// thrown errors → wrapped sentinels; intrinsic-geometry snapshot → len()/copy.
// IEEE 754 decimal interchange with Cowlishaw's Densely Packed Decimal mapping,
// little-endian as SAP stores it. See docs/provenance.md.

package value

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// ErrDecimalFloat reports a value or byte record outside the DECF16/DECF34
// contract.
var ErrDecimalFloat = errors.New("value: decimal float")

type decimalFloatFormat struct {
	label                       string
	byteLength                  int
	precision                   int
	exponentContinuationBits    uint
	coefficientContinuationBits uint
	exponentBias                int
}

var decf16 = decimalFloatFormat{"DECF16", 8, 16, 8, 50, 398}
var decf34 = decimalFloatFormat{"DECF34", 16, 34, 12, 110, 6176}

const maxDecimalFloatTextLength = 4_096

func encodeDpdDeclet(value int) int {
	left := value / 100
	middle := (value / 10) % 10
	right := value % 10
	largePattern := 0
	if left >= 8 {
		largePattern |= 4
	}
	if middle >= 8 {
		largePattern |= 2
	}
	if right >= 8 {
		largePattern |= 1
	}
	switch largePattern {
	case 0:
		return (left << 7) | (middle << 4) | right
	case 1:
		return (left << 7) | (middle << 4) | 0b1000 | (right & 1)
	case 2:
		return (left << 7) | ((right&0b110)|(middle&1))<<4 | 0b1010 | (right & 1)
	case 4:
		return ((right&0b110)|(left&1))<<7 | (middle << 4) | 0b1100 | (right & 1)
	case 6:
		return ((right&0b110)|(left&1))<<7 | ((middle & 1) << 4) | 0b1110 | (right & 1)
	case 5:
		return ((middle&0b110)|(left&1))<<7 | ((0b010 | (middle & 1)) << 4) | 0b1110 | (right & 1)
	case 3:
		return (left << 7) | ((0b100 | (middle & 1)) << 4) | 0b1110 | (right & 1)
	case 7:
		return ((left & 1) << 7) | ((0b110 | (middle & 1)) << 4) | 0b1110 | (right & 1)
	default:
		panic("unreachable DPD digit classification")
	}
}

func decodeDpdDeclet(code int) int {
	pqr := (code >> 7) & 0b111
	stu := (code >> 4) & 0b111
	v := (code >> 3) & 1
	w := (code >> 2) & 1
	x := (code >> 1) & 1
	y := code & 1
	var left, middle, right int
	switch {
	case v == 0:
		left, middle, right = pqr, stu, code&0b111
	case w == 0 && x == 0:
		left, middle, right = pqr, stu, 8+y
	case w == 0 && x == 1:
		left = pqr
		middle = 8 + ((code >> 4) & 1)
		right = (((code >> 6) & 1) << 2) | (((code >> 5) & 1) << 1) | y
	case w == 1 && x == 0:
		left = 8 + ((code >> 7) & 1)
		middle = stu
		right = (((code >> 9) & 1) << 2) | (((code >> 8) & 1) << 1) | y
	default:
		st := (code >> 5) & 0b11
		switch st {
		case 0:
			left = 8 + ((code >> 7) & 1)
			middle = 8 + ((code >> 4) & 1)
			right = (((code >> 9) & 1) << 2) | (((code >> 8) & 1) << 1) | y
		case 1:
			left = 8 + ((code >> 7) & 1)
			middle = (((code >> 9) & 1) << 2) | (((code >> 8) & 1) << 1) | ((code >> 4) & 1)
			right = 8 + y
		case 2:
			left = pqr
			middle = 8 + ((code >> 4) & 1)
			right = 8 + y
		default:
			left = 8 + ((code >> 7) & 1)
			middle = 8 + ((code >> 4) & 1)
			right = 8 + y
		}
	}
	return left*100 + middle*10 + right
}

func encodeCoefficientContinuation(digits string) *big.Int {
	encoded := new(big.Int)
	for i := 0; i < len(digits); i += 3 {
		declet, _ := strconv.Atoi(digits[i : i+3])
		encoded.Lsh(encoded, 10)
		encoded.Or(encoded, big.NewInt(int64(encodeDpdDeclet(declet))))
	}
	return encoded
}

func decodeCoefficientContinuation(bits *big.Int, digitCount int) string {
	decletCount := digitCount / 3
	groups := make([]string, decletCount)
	remainder := new(big.Int).Set(bits)
	mask := big.NewInt(0x3ff)
	tmp := new(big.Int)
	for i := decletCount - 1; i >= 0; i-- {
		code := tmp.And(remainder, mask).Int64()
		groups[i] = leftPad(strconv.Itoa(decodeDpdDeclet(int(code))), 3)
		remainder.Rsh(remainder, 10)
	}
	return strings.Join(groups, "")
}

func leftPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
}

func trailingZeroCount(digits string) int {
	i := len(digits)
	for i > 0 && digits[i-1] == '0' {
		i--
	}
	return len(digits) - i
}

var (
	infinityPattern = regexp.MustCompile(`(?i)^([+-]?)(?:inf|infinity)$`)
	nanPattern      = regexp.MustCompile(`(?i)^([+-]?)(s?nan)(\d*)$`)
	finitePattern   = regexp.MustCompile(`^([+-]?)(?:(\d+)(?:\.(\d*))?|\.(\d+))(?:[eE]([+-]?\d+))?$`)
)

type specialDecimal struct {
	negative   bool
	kind       string // "infinity" | "nan" | "snan"
	diagnostic string
}

func parseSpecial(text string, format decimalFloatFormat, path string) (*specialDecimal, error) {
	if m := infinityPattern.FindStringSubmatch(text); m != nil {
		return &specialDecimal{negative: m[1] == "-", kind: "infinity"}, nil
	}
	m := nanPattern.FindStringSubmatch(text)
	if m == nil {
		return nil, nil
	}
	diagnostic := strings.TrimLeft(m[3], "0")
	capacity := format.precision - 1
	if len(diagnostic) > capacity {
		return nil, fmt.Errorf("%w: %s exceeds its %d-digit NaN payload", ErrDecimalFloat, path, capacity)
	}
	kind := "nan"
	if strings.ToLower(m[2]) == "snan" {
		kind = "snan"
	}
	return &specialDecimal{negative: m[1] == "-", kind: kind, diagnostic: diagnostic}, nil
}

type finiteDecimal struct {
	negative    bool
	coefficient string
	exponent    int
}

func parseFinite(text string, format decimalFloatFormat, path string) (finiteDecimal, error) {
	var zero finiteDecimal
	m := finitePattern.FindStringSubmatch(text)
	if m == nil {
		return zero, fmt.Errorf("%w: %s expects a valid decimal", ErrDecimalFloat, path)
	}
	integer := m[2]
	var fraction string
	if m[2] == "" {
		fraction = m[4]
	} else {
		fraction = m[3]
	}
	unpadded := strings.TrimLeft(integer+fraction, "0")
	coefficient := unpadded
	if coefficient == "" {
		coefficient = "0"
	}

	explicit := new(big.Int)
	if m[5] != "" {
		if _, ok := explicit.SetString(m[5], 10); !ok {
			return zero, fmt.Errorf("%w: %s has an exponent too large to represent", ErrDecimalFloat, path)
		}
	}
	exponent := new(big.Int).Sub(explicit, big.NewInt(int64(len(fraction))))

	if coefficient != "0" && len(coefficient) > format.precision {
		excess := len(coefficient) - format.precision
		if trailingZeroCount(coefficient) < excess {
			return zero, fmt.Errorf("%w: %s exceeds %d significant digits without rounding", ErrDecimalFloat, path, format.precision)
		}
		coefficient = coefficient[:len(coefficient)-excess]
		exponent.Add(exponent, big.NewInt(int64(excess)))
	}

	minExp := big.NewInt(int64(-format.exponentBias))
	maxEncoded := new(big.Int).Sub(new(big.Int).Mul(big.NewInt(3), new(big.Int).Lsh(big.NewInt(1), format.exponentContinuationBits)), big.NewInt(1))
	maxExp := new(big.Int).Sub(maxEncoded, big.NewInt(int64(format.exponentBias)))

	if coefficient == "0" {
		if exponent.Cmp(minExp) < 0 {
			exponent.Set(minExp)
		}
		if exponent.Cmp(maxExp) > 0 {
			exponent.Set(maxExp)
		}
	} else if exponent.Cmp(maxExp) > 0 {
		requiredZeros := new(big.Int).Sub(exponent, maxExp)
		availableDigits := int64(format.precision - len(coefficient))
		if requiredZeros.Cmp(big.NewInt(availableDigits)) > 0 {
			return zero, fmt.Errorf("%w: %s is outside %s range without rounding", ErrDecimalFloat, path, format.label)
		}
		coefficient += strings.Repeat("0", int(requiredZeros.Int64()))
		exponent.Set(maxExp)
	} else if exponent.Cmp(minExp) < 0 {
		requiredZeros := new(big.Int).Sub(minExp, exponent)
		tz := int64(trailingZeroCount(coefficient))
		if requiredZeros.Cmp(big.NewInt(tz)) > 0 {
			return zero, fmt.Errorf("%w: %s is outside %s range without rounding", ErrDecimalFloat, path, format.label)
		}
		coefficient = coefficient[:len(coefficient)-int(requiredZeros.Int64())]
		exponent.Set(minExp)
	}

	return finiteDecimal{negative: m[1] == "-", coefficient: coefficient, exponent: int(exponent.Int64())}, nil
}

func writeLittleEndian(value *big.Int, byteLength int) []byte {
	out := make([]byte, byteLength)
	remainder := new(big.Int).Set(value)
	mask := big.NewInt(0xff)
	tmp := new(big.Int)
	for i := 0; i < byteLength; i++ {
		out[i] = byte(tmp.And(remainder, mask).Int64())
		remainder.Rsh(remainder, 8)
	}
	return out
}

func readLittleEndian(value []byte) *big.Int {
	result := new(big.Int)
	for i := len(value) - 1; i >= 0; i-- {
		result.Lsh(result, 8)
		result.Or(result, big.NewInt(int64(value[i])))
	}
	return result
}

func encodeSpecial(value specialDecimal, format decimalFloatFormat) []byte {
	totalBits := format.byteLength * 8
	sign := new(big.Int)
	if value.negative {
		sign.Lsh(big.NewInt(1), uint(totalBits-1))
	}
	combShift := uint(totalBits - 6)
	if value.kind == "infinity" {
		return writeLittleEndian(new(big.Int).Or(sign, new(big.Int).Lsh(big.NewInt(0b11110), combShift)), format.byteLength)
	}
	diagnosticDigits := leftPad(value.diagnostic, format.precision-1)
	diagnostic := encodeCoefficientContinuation(diagnosticDigits)
	signaling := new(big.Int)
	if value.kind == "snan" {
		signaling.Lsh(big.NewInt(1), format.coefficientContinuationBits+format.exponentContinuationBits-1)
	}
	encoded := new(big.Int).Or(sign, new(big.Int).Lsh(big.NewInt(0b11111), combShift))
	encoded.Or(encoded, signaling)
	encoded.Or(encoded, diagnostic)
	return writeLittleEndian(encoded, format.byteLength)
}

func encodeDecimalFloat(value string, format decimalFloatFormat, path string) ([]byte, error) {
	if len(value) > maxDecimalFloatTextLength {
		return nil, fmt.Errorf("%w: %s decimal text exceeds %d characters", ErrDecimalFloat, path, maxDecimalFloatTextLength)
	}
	special, err := parseSpecial(value, format, path)
	if err != nil {
		return nil, err
	}
	if special != nil {
		return encodeSpecial(*special, format), nil
	}
	finite, err := parseFinite(value, format, path)
	if err != nil {
		return nil, err
	}
	digits := leftPad(finite.coefficient, format.precision)
	msd := int(digits[0] - '0')
	coeffCont := encodeCoefficientContinuation(digits[1:])
	encodedExponent := finite.exponent + format.exponentBias
	exponentMSB := encodedExponent >> format.exponentContinuationBits
	exponentContMask := (1 << format.exponentContinuationBits) - 1
	exponentCont := encodedExponent & exponentContMask
	var combination int
	if msd <= 7 {
		combination = (exponentMSB << 3) | msd
	} else {
		combination = 0b11000 | (exponentMSB << 1) | (msd - 8)
	}
	totalBits := format.byteLength * 8
	sign := new(big.Int)
	if finite.negative {
		sign.Lsh(big.NewInt(1), uint(totalBits-1))
	}
	encoded := new(big.Int).Set(sign)
	encoded.Or(encoded, new(big.Int).Lsh(big.NewInt(int64(combination)), uint(totalBits-6)))
	encoded.Or(encoded, new(big.Int).Lsh(big.NewInt(int64(exponentCont)), format.coefficientContinuationBits))
	encoded.Or(encoded, coeffCont)
	return writeLittleEndian(encoded, format.byteLength), nil
}

func formatFinite(negative bool, coefficient string, exponent int) string {
	digits := strings.TrimLeft(coefficient, "0")
	if digits == "" {
		digits = "0"
	}
	adjustedExponent := exponent + len(digits) - 1
	var body string
	if exponent <= 0 && adjustedExponent >= -6 {
		if exponent == 0 {
			body = digits
		} else {
			point := len(digits) + exponent
			if point > 0 {
				body = digits[:point] + "." + digits[point:]
			} else {
				body = "0." + strings.Repeat("0", -point) + digits
			}
		}
	} else {
		var significand string
		if len(digits) == 1 {
			significand = digits
		} else {
			significand = digits[:1] + "." + digits[1:]
		}
		exp := strconv.Itoa(adjustedExponent)
		if adjustedExponent >= 0 {
			exp = "+" + exp
		}
		body = significand + "E" + exp
	}
	if negative {
		return "-" + body
	}
	return body
}

func decodeDecimalFloat(value []byte, format decimalFloatFormat, path string) (string, error) {
	if len(value) != format.byteLength {
		return "", fmt.Errorf("%w: %s expects exactly %d bytes", ErrDecimalFloat, path, format.byteLength)
	}
	encoded := readLittleEndian(value)
	totalBits := format.byteLength * 8
	negative := encoded.Bit(totalBits-1) == 1
	combination := int(new(big.Int).And(new(big.Int).Rsh(encoded, uint(totalBits-6)), big.NewInt(0b11111)).Int64())
	coeffMask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), format.coefficientContinuationBits), big.NewInt(1))
	coeffCont := new(big.Int).And(encoded, coeffMask)

	sign := ""
	if negative {
		sign = "-"
	}
	if combination == 0b11110 {
		return sign + "Infinity", nil
	}
	if combination == 0b11111 {
		signalingBit := format.coefficientContinuationBits + format.exponentContinuationBits - 1
		signaling := encoded.Bit(int(signalingBit)) == 1
		diagnostic := strings.TrimLeft(decodeCoefficientContinuation(coeffCont, format.precision-1), "0")
		marker := "NaN"
		if signaling {
			marker = "sNaN"
		}
		return sign + marker + diagnostic, nil
	}

	var exponentMSB, msd int
	if combination < 0b11000 {
		exponentMSB = combination >> 3
		msd = combination & 0b111
	} else {
		exponentMSB = (combination >> 1) & 0b11
		msd = 8 + (combination & 1)
	}
	exponentContMask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), format.exponentContinuationBits), big.NewInt(1))
	exponentCont := int(new(big.Int).And(new(big.Int).Rsh(encoded, format.coefficientContinuationBits), exponentContMask).Int64())
	exponent := (exponentMSB << format.exponentContinuationBits) | exponentCont
	coefficient := strconv.Itoa(msd) + decodeCoefficientContinuation(coeffCont, format.precision-1)
	return formatFinite(negative, coefficient, exponent-format.exponentBias), nil
}

// EncodeDecimalFloat16 encodes a decimal64 DPD value in SAP's DECF16 form.
func EncodeDecimalFloat16(value, path string) ([]byte, error) {
	if path == "" {
		path = "DECF16"
	}
	return encodeDecimalFloat(value, decf16, path)
}

// DecodeDecimalFloat16 decodes SAP DECF16 to a precision-preserving string.
func DecodeDecimalFloat16(value []byte, path string) (string, error) {
	if path == "" {
		path = "DECF16"
	}
	return decodeDecimalFloat(value, decf16, path)
}

// EncodeDecimalFloat34 encodes a decimal128 DPD value in SAP's DECF34 form.
func EncodeDecimalFloat34(value, path string) ([]byte, error) {
	if path == "" {
		path = "DECF34"
	}
	return encodeDecimalFloat(value, decf34, path)
}

// DecodeDecimalFloat34 decodes SAP DECF34 to a precision-preserving string.
func DecodeDecimalFloat34(value []byte, path string) (string, error) {
	if path == "" {
		path = "DECF34"
	}
	return decodeDecimalFloat(value, decf34, path)
}
