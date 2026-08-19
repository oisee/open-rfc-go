// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/password-scramble-equivalence.test.ts at commit
// 847036d, Copyright 2026 Marian Zeis, licensed under the Apache License,
// Version 2.0. Modified by open-rfc-go contributors: rewritten for the testing
// package. The frozen SHA-256 over the full input sweep is reproduced exactly;
// if the Go producer diverges by a single byte, the digest changes.

package scramble

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func printableRange() []byte {
	var codes []byte
	for c := 0x20; c <= 0x7e; c++ {
		codes = append(codes, byte(c))
	}
	return codes
}

func sweepSeeds() []uint32 {
	var seeds []uint32
	for s := uint64(0); s <= 0xff; s++ {
		seeds = append(seeds, uint32(s))
	}
	for s := uint64(0xffff_ff00); s <= 0xffff_ffff; s++ {
		seeds = append(seeds, uint32(s))
	}
	seeds = append(seeds,
		0x0000_0100, 0x0000_8000, 0x0000_ffff, 0x0001_0000, 0x5ae0_b7a3,
		0x7fff_ffff, 0x8000_0000, 0xdead_beef, 0xfedc_ba98, 0x1234_5678,
	)
	return seeds
}

func sweepPassword(length int, printable []byte) string {
	b := make([]byte, length)
	for pos := 0; pos < length; pos++ {
		b[pos] = printable[(pos*7+length)%len(printable)]
	}
	return string(b)
}

func TestProducesFrozenFieldBytesAcrossSweep(t *testing.T) {
	printable := printableRange()
	digest := sha256.New()
	vectors := 0
	absorb := func(password string, seed uint32) {
		out, err := ScrambleRfcPassword(password, seed)
		if err != nil {
			t.Fatalf("scramble(%q, %#x): %v", password, seed, err)
		}
		digest.Write(out)
		digest.Write([]byte{0xff})
		vectors++
	}

	for _, seed := range sweepSeeds() {
		for length := 0; length <= 40; length++ {
			absorb(sweepPassword(length, printable), seed)
		}
	}
	for _, code := range printable {
		absorb(repeat(code, 40), 0)
		absorb(repeat(code, 40), 0x5ae0_b7a3)
	}

	if vectors != 21_592 {
		t.Fatalf("vectors = %d, want 21592", vectors)
	}
	got := hex.EncodeToString(digest.Sum(nil))
	if got != "f3e0b74e48219b80e926e4ff2684c045e4fc93015eca9221a5ddb6a50cd8ff82" {
		t.Fatalf("digest = %s", got)
	}
}

func TestProducesFrozenFieldBytesForBoundaryCases(t *testing.T) {
	cases := []struct {
		password string
		seed     uint32
		expected string
	}{
		{"", 0, "00000000"},
		{"AB", 0, "00000000b150"},
		{"secret", 0x15, "150000008981dc9b914e"},
		{repeat('x', 40), 0x15, "15000000829cc7918c42d277b89294e3e555310d2a1dab67283465f464236b89eee52cab0e1299ba2cca2145"},
		{repeat('x', 40), 0xffff_ffff, "ffffffff157c7261c7229cf431269cc2656bab279b1413adb1a62a23123a4d3def405bbafd707c3f2c82d487"},
		{"~", 0xffff_ffff, "ffffffff13"},
	}
	for _, c := range cases {
		out, err := ScrambleRfcPassword(c.password, c.seed)
		if err != nil {
			t.Fatalf("scramble(%q, %#x): %v", c.password, c.seed, err)
		}
		if got := hex.EncodeToString(out); got != c.expected {
			t.Fatalf("password %d bytes at seed %#x = %s, want %s", len(c.password), c.seed, got, c.expected)
		}
	}
}

func repeat(b byte, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return string(out)
}
