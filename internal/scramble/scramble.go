// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/protocol/password-scramble.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown TypeError/
// RangeError became returned, wrapped sentinel errors; the seed argument is a
// uint32, so its "unsigned 32-bit integer" guard is enforced by the type; the
// optional random-seed default is exposed as ScrambleRfcPasswordRandomSeed,
// since a uint32 cannot double as "unset". The per-position term is computed in
// int64 and masked to a byte, reproducing JavaScript's `& 0xff` on a value the
// seed drives negative. The cleartext-zeroing finally block is dropped: Go
// cannot guarantee a []byte is not copied by the GC, so the defense is
// illusory rather than merely unavailable. See docs/provenance.md.

// Package scramble produces the CPIC/WebSocket logon-password field.
package scramble

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
)

var scrambleTable = [64]byte{
	0xf0, 0xed, 0x53, 0xb8, 0x32, 0x44, 0xf1, 0xf8, 0x76, 0xc6, 0x79, 0x59, 0xfd,
	0x4f, 0x13, 0xa2, 0xc1, 0x51, 0x95, 0xec, 0x54, 0x83, 0xc2, 0x34, 0x77, 0x49,
	0x43, 0xa2, 0x7d, 0xe2, 0x65, 0x96, 0x5e, 0x53, 0x98, 0x78, 0x9a, 0x17, 0xa3,
	0x3c, 0xd3, 0x83, 0xa8, 0xb8, 0x29, 0xfb, 0xdc, 0xa5, 0x55, 0xd7, 0x02, 0x77,
	0x84, 0x13, 0xac, 0xdd, 0xf9, 0xb8, 0x31, 0x16, 0x61, 0x0e, 0x6d, 0xfa,
}

var (
	// ErrPasswordTooLong reports a password over the 40-byte cap.
	ErrPasswordTooLong = errors.New("scramble: password must contain at most 40 bytes")
	// ErrPasswordNotASCII reports a password outside the proven ASCII baseline.
	ErrPasswordNotASCII = errors.New("scramble: password contains characters outside the proven ASCII baseline")
)

var printableASCII = regexp.MustCompile(`^[\x20-\x7e]*$`)

// ScrambleRfcPassword produces the logon-password field for the given seed.
func ScrambleRfcPassword(password string, seed uint32) ([]byte, error) {
	if len(password) > 40 {
		return nil, ErrPasswordTooLong
	}
	if !printableASCII.MatchString(password) {
		return nil, ErrPasswordNotASCII
	}
	clear := []byte(password)
	result := make([]byte, 4+len(clear))
	binary.LittleEndian.PutUint32(result[0:], seed)
	mixedSeed := seed ^ (seed >> 5)
	startIndex := mixedSeed ^ (seed << 1)
	for index := 0; index < len(clear); index++ {
		tableValue := scrambleTable[(startIndex+uint32(index))&0x3f]
		// The 40-byte cap keeps seed*index*index within int64; the mask
		// reproduces JavaScript's `& 0xff` on the seed-driven negative term.
		term := int64(seed)*int64(index)*int64(index) - int64(index)
		result[4+index] = clear[index] ^ tableValue ^ byte(term&0xff)
	}
	return result, nil
}

// ScrambleRfcPasswordRandomSeed produces the field with a fresh random seed.
func ScrambleRfcPasswordRandomSeed(password string) ([]byte, error) {
	var seedBytes [4]byte
	if _, err := rand.Read(seedBytes[:]); err != nil {
		return nil, fmt.Errorf("scramble: reading random seed: %w", err)
	}
	return ScrambleRfcPassword(password, binary.LittleEndian.Uint32(seedBytes[:]))
}
