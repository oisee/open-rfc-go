// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/decimal-float.test.ts at commit 847036d,
// Copyright 2026 Marian Zeis, Apache-2.0. Rewritten for the testing package.
// The exact DPD hex vectors and the exhaustive declet sweeps (with Cowlishaw's
// independent Boolean equations as oracle) are reproduced. JS-only cases —
// decimal-object conversion counting, geometry-accessor override, null input,
// and number/bigint input types — are dropped or expressed as string inputs;
// see docs/provenance.md.

package value

import (
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"
	"testing"
)

func encHex16(t *testing.T, v string) string {
	t.Helper()
	b, err := EncodeDecimalFloat16(v, "")
	if err != nil {
		t.Fatalf("encode16(%q): %v", v, err)
	}
	return hex.EncodeToString(b)
}
func dec16(t *testing.T, h string) string {
	t.Helper()
	s, err := DecodeDecimalFloat16(mustHexV(t, h), "")
	if err != nil {
		t.Fatalf("decode16(%q): %v", h, err)
	}
	return s
}

func TestDecimal64Vectors(t *testing.T) {
	vectors := [][3]string{
		{"0", "0000000000003822", "0"},
		{"-0", "00000000000038a2", "-0"},
		{"1", "0100000000003822", "1"},
		{"-1", "01000000000038a2", "-1"},
		{"123.45", "c549000000003022", "123.45"},
		{"123.45E67", "c549000000003c23", "1.2345E+69"},
		{"9.999999999999999E+384", "fffcf3cf3ffffc77", "9.999999999999999E+384"},
		{"1E-383", "0100000000003c00", "1E-383"},
		{"NaN", "000000000000007c", "NaN"},
		{"Infinity", "0000000000000078", "Infinity"},
		{"-Infinity", "00000000000000f8", "-Infinity"},
	}
	for _, v := range vectors {
		if got := encHex16(t, v[0]); got != v[1] {
			t.Fatalf("encode(%q) = %s, want %s", v[0], got, v[1])
		}
		if got := dec16(t, v[1]); got != v[2] {
			t.Fatalf("decode(%q) = %s, want %s", v[1], got, v[2])
		}
	}
}

func TestDecimal128Vectors(t *testing.T) {
	vectors := [][3]string{
		{"0", "00000000000000000000000000000822", "0"},
		{"-0", "000000000000000000000000000008a2", "-0"},
		{"1", "01000000000000000000000000000822", "1"},
		{"-1", "010000000000000000000000000008a2", "-1"},
		{"123.45", "c5490000000000000000000000800722", "123.45"},
		{"123.45E67", "c5490000000000000000000000401822", "1.2345E+69"},
		{"9.999999999999999999999999999999999E+6144", "fffcf3cf3ffffcf3cf3ffffcf3cfff77", "9.999999999999999999999999999999999E+6144"},
		{"1E-6143", "01000000000000000000000000400800", "1E-6143"},
		{"NaN", "0000000000000000000000000000007c", "NaN"},
		{"Infinity", "00000000000000000000000000000078", "Infinity"},
		{"-Infinity", "000000000000000000000000000000f8", "-Infinity"},
	}
	for _, v := range vectors {
		b, err := EncodeDecimalFloat34(v[0], "")
		if err != nil || hex.EncodeToString(b) != v[1] {
			t.Fatalf("encode34(%q) = %x, %v; want %s", v[0], b, err, v[1])
		}
		s, err := DecodeDecimalFloat34(mustHexV(t, v[1]), "")
		if err != nil || s != v[2] {
			t.Fatalf("decode34(%q) = %s, %v; want %s", v[1], s, err, v[2])
		}
	}
}

func oracleEncodeDeclet(value int) int {
	left, middle, right := value/100, (value/10)%10, value%10
	a, b, c, d := (left>>3)&1, (left>>2)&1, (left>>1)&1, left&1
	e, f, g, h := (middle>>3)&1, (middle>>2)&1, (middle>>1)&1, middle&1
	i, j, k, m := (right>>3)&1, (right>>2)&1, (right>>1)&1, right&1
	not := func(x int) int { return x ^ 1 }
	p := b | (a & j) | (a & f & i)
	q := c | (a & k) | (a & g & i)
	r := d
	s := (f & (not(a) | not(i))) | (not(a) & e & j) | (e & i)
	tt := g | (not(a) & e & k) | (a & i)
	u := h
	v := a | e | i
	w := a | (e & i) | (not(e) & j)
	x := e | (a & i) | (not(a) & k)
	y := m
	return (p << 9) | (q << 8) | (r << 7) | (s << 6) | (tt << 5) | (u << 4) | (v << 3) | (w << 2) | (x << 1) | y
}

func oracleDecodeDeclet(code int) int {
	p, q, r := (code>>9)&1, (code>>8)&1, (code>>7)&1
	s, tt, u := (code>>6)&1, (code>>5)&1, (code>>4)&1
	v, w, x, y := (code>>3)&1, (code>>2)&1, (code>>1)&1, code&1
	not := func(b int) int { return b ^ 1 }
	a := (v & w) & (not(s) | tt | not(x))
	b := p & (not(v) | not(w) | (s & not(tt) & x))
	c := q & (not(v) | not(w) | (s & not(tt) & x))
	d := r
	e := v & ((not(w) & x) | (not(tt) & x) | (s & x))
	f := (s & (not(v) | not(x))) | (p & not(s) & tt & v & w & x)
	g := (tt & (not(v) | not(x))) | (q & not(s) & tt & w)
	h := u
	i := v & ((not(w) & not(x)) | (w & x & (s | tt)))
	j := (not(v) & w) | (s & v & not(w) & x) | (p & w & (not(x) | (not(s) & not(tt))))
	k := (not(v) & x) | (tt & not(w) & x) | (q & v & w & (not(x) | (not(s) & not(tt))))
	m := y
	left := (a << 3) | (b << 2) | (c << 1) | d
	middle := (e << 3) | (f << 2) | (g << 1) | h
	right := (i << 3) | (j << 2) | (k << 1) | m
	return left*100 + middle*10 + right
}

func TestExhaustivelyEmitsEveryCanonicalDeclet(t *testing.T) {
	mask := big.NewInt(0x3ff)
	for value := 0; value <= 999; value++ {
		encoded, err := EncodeDecimalFloat16(strconv.Itoa(value), "")
		if err != nil {
			t.Fatalf("encode %d: %v", value, err)
		}
		got := int(new(big.Int).And(readLittleEndian(encoded), mask).Int64())
		if got != oracleEncodeDeclet(value) {
			t.Fatalf("declet %d: got %d, oracle %d", value, got, oracleEncodeDeclet(value))
		}
		if s, _ := DecodeDecimalFloat16(encoded, ""); s != strconv.Itoa(value) {
			t.Fatalf("decode %d = %s", value, s)
		}
	}
}

func TestDecodesAll1024Declets(t *testing.T) {
	zero := readLittleEndian(mustHexEncode16(t, "0"))
	notMask := new(big.Int).Not(big.NewInt(0x3ff))
	redundant := 0
	for code := 0; code < 1024; code++ {
		expected := oracleDecodeDeclet(code)
		encodedInt := new(big.Int).Or(new(big.Int).And(zero, notMask), big.NewInt(int64(code)))
		encoded := writeLittleEndian(encodedInt, 8)
		s, err := DecodeDecimalFloat16(encoded, "")
		if err != nil || s != strconv.Itoa(expected) {
			t.Fatalf("code %d = %s, %v; want %d", code, s, err, expected)
		}
		if oracleEncodeDeclet(expected) != code {
			redundant++
		}
	}
	if redundant != 24 {
		t.Fatalf("redundant = %d, want 24", redundant)
	}
}

func mustHexEncode16(t *testing.T, v string) []byte {
	t.Helper()
	b, err := EncodeDecimalFloat16(v, "")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPreservesCohortsSignedZeroSubnormals(t *testing.T) {
	rt16 := func(v string) string { s, _ := DecodeDecimalFloat16(mustHexEncode16(t, v), ""); return s }
	if rt16("1.2300") != "1.2300" {
		t.Fatal("cohort 1.2300")
	}
	if encHex16(t, "-0.00") != "00000000000030a2" || rt16("-0.00") != "-0.00" {
		t.Fatal("-0.00")
	}
	if encHex16(t, "1E-398") != "0100000000000000" || rt16("1E-398") != "1E-398" {
		t.Fatal("1E-398")
	}
	if rt16("10E-399") != "1E-398" {
		t.Fatal("10E-399")
	}
	if encHex16(t, "1E+384") != "000000000000fc47" || rt16("1E+384") != "1.000000000000000E+384" {
		t.Fatal("1E+384")
	}
	if rt16("0E-999") != "0E-398" || rt16("-0E+999") != "-0E+369" {
		t.Fatal("zero clamps")
	}
	rt34 := func(v string) string {
		b, _ := EncodeDecimalFloat34(v, "")
		s, _ := DecodeDecimalFloat34(b, "")
		return s
	}
	if b, _ := EncodeDecimalFloat34("1E-6176", ""); hex.EncodeToString(b) != "01000000000000000000000000000000" {
		t.Fatal("1E-6176 hex")
	}
	if rt34("1E-6176") != "1E-6176" {
		t.Fatal("1E-6176")
	}
	if b, _ := EncodeDecimalFloat34("1E+6144", ""); hex.EncodeToString(b) != "00000000000000000000000000c0ff47" {
		t.Fatal("1E+6144 hex")
	}
	if rt34("1E+6144") != "1."+strings.Repeat("0", 33)+"E+6144" {
		t.Fatal("1E+6144")
	}
	if rt34("0E-99999") != "0E-6176" {
		t.Fatal("0E-99999")
	}
}

func TestRescalesExcessTrailingZeros(t *testing.T) {
	if encHex16(t, "12345678901234560") != "568ee2c1b9343d26" {
		t.Fatal("rescale hex")
	}
	if dec16(t, "568ee2c1b9343d26") != "1.234567890123456E+16" {
		t.Fatal("rescale decode")
	}
	if b, _ := EncodeDecimalFloat34("12345678901234567890123456789012340", ""); hex.EncodeToString(b) != "3435827771123c6fe5281e9c4b530826" {
		t.Fatal("decf34 rescale hex")
	}
	if _, err := EncodeDecimalFloat16("12345678901234567", ""); err == nil || !strings.Contains(err.Error(), "exceeds 16 significant digits") {
		t.Fatalf("16 sig: %v", err)
	}
	if _, err := EncodeDecimalFloat34("12345678901234567890123456789012341", ""); err == nil || !strings.Contains(err.Error(), "exceeds 34 significant digits") {
		t.Fatalf("34 sig: %v", err)
	}
}

func TestAcceptsSpecials(t *testing.T) {
	rt := func(v string) string { s, _ := DecodeDecimalFloat16(mustHexEncode16(t, v), ""); return s }
	if rt("inf") != "Infinity" || rt("+INFINITY") != "Infinity" || rt("sNaN") != "sNaN" || rt("-NaN8275") != "-NaN8275" {
		t.Fatal("specials 16")
	}
	if b, _ := EncodeDecimalFloat34("sNaN123456789", ""); func() bool { s, _ := DecodeDecimalFloat34(b, ""); return s != "sNaN123456789" }() {
		t.Fatal("snan34")
	}
	if s, _ := DecodeDecimalFloat16(mustHexV(t, "ffffffffffffffff"), ""); s != "-sNaN999999999999999" {
		t.Fatalf("allOnes = %s", s)
	}
	if s, _ := DecodeDecimalFloat16(mustHexV(t, "fffffffffffffffb"), ""); s != "-Infinity" {
		t.Fatalf("inf payload = %s", s)
	}
}

func TestDecimalFloatRejects(t *testing.T) {
	bad16 := []struct{ v, sub string }{
		{"1.2345678901234567", "exceeds 16 significant digits"},
		{"1E+385", "outside DECF16 range"},
		{"1E-399", "outside DECF16 range"},
		{"NaN1234567890123456", "15-digit NaN payload"},
		{"", "valid decimal"},
		{" 1", "valid decimal"},
		{"1,2", "valid decimal"},
		{".", "valid decimal"},
	}
	for _, c := range bad16 {
		if _, err := EncodeDecimalFloat16(c.v, ""); err == nil || !strings.Contains(err.Error(), c.sub) {
			t.Fatalf("encode16(%q) = %v, want %q", c.v, err, c.sub)
		}
	}
	if _, err := EncodeDecimalFloat34("1E+6145", ""); err == nil || !strings.Contains(err.Error(), "outside DECF34 range") {
		t.Fatalf("6145: %v", err)
	}
	if _, err := EncodeDecimalFloat34("1E-6177", ""); err == nil || !strings.Contains(err.Error(), "outside DECF34 range") {
		t.Fatalf("-6177: %v", err)
	}
	if _, err := DecodeDecimalFloat16(make([]byte, 7), ""); err == nil || !strings.Contains(err.Error(), "exactly 8 bytes") {
		t.Fatalf("short16: %v", err)
	}
	if _, err := DecodeDecimalFloat34(make([]byte, 17), ""); err == nil || !strings.Contains(err.Error(), "exactly 16 bytes") {
		t.Fatalf("long34: %v", err)
	}
}

func FuzzDecodeDecimalFloat16(f *testing.F) {
	f.Add([]byte{0xc5, 0x49, 0, 0, 0, 0, 0x30, 0x22})
	f.Add(make([]byte, 8))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 8 {
			return
		}
		_, _ = DecodeDecimalFloat16(data, "")
	})
}
