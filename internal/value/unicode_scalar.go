// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/values/unicode-scalar.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown RangeError/Error
// became returned, wrapped errors. Go strings are UTF-8, in which surrogate code
// points cannot appear, so AssertUnicodeScalarText validates UTF-8 well-formed-
// ness — the Go-native equivalent of rejecting isolated UTF-16 surrogates. The
// character-reference value is taken from the number, not the digit count
// (the recurring-bug-class fix), preserved here verbatim. See docs/provenance.md.

package value

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	// ErrScalarText reports non-scalar or NUL-bearing text.
	ErrScalarText = errors.New("value: invalid unicode scalar text")
	// ErrXMLEntity reports a malformed or out-of-range XML entity reference.
	ErrXMLEntity = errors.New("value: invalid XML entity")
)

// AssertUnicodeScalarText rejects text that is not valid Unicode scalar text.
// In Go's UTF-8 strings this means rejecting invalid encodings, which is where
// an isolated UTF-16 surrogate would otherwise hide.
func AssertUnicodeScalarText(value, path string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s contains an isolated surrogate code unit", ErrScalarText, path)
	}
	return nil
}

// AssertNulFreeUnicodeScalarText additionally rejects NUL, which the classic
// wire uses as a value terminator.
func AssertNulFreeUnicodeScalarText(value, path string) error {
	if err := AssertUnicodeScalarText(value, path); err != nil {
		return err
	}
	if strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("%w: %s contains NUL", ErrScalarText, path)
	}
	return nil
}

var namedXMLEntityCodePoints = map[string]int{
	"amp": 0x26, "lt": 0x3c, "gt": 0x3e, "quot": 0x22, "apos": 0x27,
}

const maximumCharacterReferenceRun = 32

var (
	decimalRun     = regexp.MustCompile(`^[0-9]+$`)
	hexadecimalRun = regexp.MustCompile(`^[0-9A-Fa-f]+$`)
)

// characterReferenceValue returns the value of one character-reference digit
// run, ignoring how it is padded.
func characterReferenceValue(digits string, radix int, run *regexp.Regexp, path string) (int, error) {
	if len(digits) > maximumCharacterReferenceRun || !run.MatchString(digits) {
		return 0, fmt.Errorf("%w: %s contains an unsupported XML entity", ErrXMLEntity, path)
	}
	// An all-zero run denotes U+0000, so keep one digit.
	significant := strings.TrimLeft(digits, "0")
	if significant == "" {
		significant = "0"
	}
	v, err := strconv.ParseInt(significant, radix, 64)
	if err != nil {
		// Overflow: larger than any scalar, rejected by the range check below.
		return 0x110000, nil
	}
	return int(v), nil
}

// DecodeXMLEntityReference decodes the XML reference starting at raw[start],
// which must be '&', returning the code point and the consumed length.
func DecodeXMLEntityReference(raw string, start int, path string) (codePoint int, length int, err error) {
	semicolon := strings.IndexByte(raw[start+1:], ';')
	if semicolon < 0 {
		return 0, 0, fmt.Errorf("%w: %s contains a truncated XML entity", ErrXMLEntity, path)
	}
	semicolon += start + 1
	body := raw[start+1 : semicolon]
	switch {
	case len(body) == 0:
		return 0, 0, fmt.Errorf("%w: %s contains an empty XML entity", ErrXMLEntity, path)
	case body[0] != '#':
		named, ok := namedXMLEntityCodePoints[body]
		if !ok {
			return 0, 0, fmt.Errorf("%w: %s contains an unsupported XML entity", ErrXMLEntity, path)
		}
		codePoint = named
	case len(body) > 1 && body[1] == 'x':
		codePoint, err = characterReferenceValue(body[2:], 16, hexadecimalRun, path)
	default:
		codePoint, err = characterReferenceValue(body[1:], 10, decimalRun, path)
	}
	if err != nil {
		return 0, 0, err
	}
	if codePoint > 0x10ffff || (codePoint >= 0xd800 && codePoint <= 0xdfff) {
		return 0, 0, fmt.Errorf("%w: %s contains an out-of-range XML entity", ErrXMLEntity, path)
	}
	return codePoint, semicolon + 1 - start, nil
}
