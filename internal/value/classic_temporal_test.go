// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/classic-temporal.test.ts at commit 847036d,
// Copyright 2026 Marian Zeis, Apache-2.0. Rewritten for the testing package.
// The exact little-endian vectors, per-EXID min/max boundaries, and the
// 1582 calendar-gap cases are reproduced. JS-only cases (number-as-string
// inputs, coercible objects, geometry-accessor override) have no Go analogue;
// see docs/provenance.md.

package value

import (
	"encoding/hex"
	"strings"
	"testing"
)

func encT(t *testing.T, exid TemporalExid, v string) string {
	t.Helper()
	b, err := EncodeClassicTemporal(exid, v, "")
	if err != nil {
		t.Fatalf("encode(%s, %q): %v", exid, v, err)
	}
	return hex.EncodeToString(b)
}

func decT(t *testing.T, exid TemporalExid, h string) string {
	t.Helper()
	s, err := DecodeClassicTemporal(exid, mustHexV(t, h), "")
	if err != nil {
		t.Fatalf("decode(%s, %q): %v", exid, h, err)
	}
	return s
}

func TestClassicDatsTimsFixedForms(t *testing.T) {
	for _, v := range []string{"", "00000000", "19000229", "20260229", "99991231", "        "} {
		if err := AssertClassicDate(v, "DATE"); err != nil {
			t.Fatalf("date %q rejected: %v", v, err)
		}
	}
	for _, v := range []string{"", "000000", "235959", "240000", "999999", "      "} {
		if err := AssertClassicTime(v, "TIME"); err != nil {
			t.Fatalf("time %q rejected: %v", v, err)
		}
	}
	if err := AssertClassicDate("2026-07-15", "DATE"); err == nil || !strings.Contains(err.Error(), "YYYYMMDD") {
		t.Fatalf("date dashes: %v", err)
	}
	if err := AssertClassicDate("１２３４５６７８", "DATE"); err == nil {
		t.Fatal("fullwidth digits accepted")
	}
	if err := AssertClassicTime("12:00:00", "TIME"); err == nil || !strings.Contains(err.Error(), "HHMMSS") {
		t.Fatalf("time colons: %v", err)
	}
	if err := AssertClassicDate("202607  ", "DATE"); err == nil || !strings.Contains(err.Error(), "eight spaces") {
		t.Fatalf("partial spaces date: %v", err)
	}
	if err := AssertClassicTime("1200  ", "TIME"); err == nil || !strings.Contains(err.Error(), "six spaces") {
		t.Fatalf("partial spaces time: %v", err)
	}
}

func TestCompactTemporalVectors(t *testing.T) {
	vectors := []struct {
		exid  TemporalExid
		value string
		hex   string
	}{
		{"p", "2002-02-04T20:15:01.1234567", "08272f17627dc308"},
		{"n", "2002-02-04T20:15:01", "c685f3b30e000000"},
		{"w", "2002-02-04T20:15", "8086bb3e00000000"},
		{"d", "2002-02-04", "07270b00"},
		{"7", "2020-W53", "b99b0100"},
		{"x", "2002-02", "ce5d0000"},
		{"t", "20:15:01", "c61c0100"},
		{"i", "20:15", "c004"},
		{"c", "02-04", "2300"},
	}
	for _, v := range vectors {
		if got := encT(t, v.exid, v.value); got != v.hex {
			t.Fatalf("encode(%s, %q) = %s, want %s", v.exid, v.value, got, v.hex)
		}
		if got := decT(t, v.exid, v.hex); got != v.value {
			t.Fatalf("decode(%s, %q) = %s, want %s", v.exid, v.hex, got, v.value)
		}
	}
}

func TestTemporalRawZeroInitials(t *testing.T) {
	if encT(t, "p", utclongInitial) != "0000000000000000" || encT(t, "p", "") != "0000000000000000" {
		t.Fatal("UTCLONG initial")
	}
	if s, _ := DecodeClassicTemporal("p", make([]byte, 8), ""); s != utclongInitial {
		t.Fatalf("UTCLONG decode initial = %s", s)
	}
	for _, exid := range []TemporalExid{"n", "w", "d", "7", "x", "t", "i", "c"} {
		width, _ := ClassicTemporalByteLength(exid)
		if encT(t, exid, "") != strings.Repeat("00", width) {
			t.Fatalf("%s empty encode", exid)
		}
		if s, _ := DecodeClassicTemporal(exid, make([]byte, width), ""); s != "" {
			t.Fatalf("%s empty decode = %q", exid, s)
		}
	}
}

func TestCompactTemporalBoundaries(t *testing.T) {
	boundaries := []struct {
		exid                     TemporalExid
		min, minRaw, max, maxRaw string
	}{
		{"p", "0001-01-01T00:00:00.0000000", "0100000000000000", "9999-12-31T23:59:59.9999999", "00c00a49082aca2b"},
		{"n", "0001-01-01T00:00:00", "0100000000000000", "9999-12-31T23:59:59", "80db887749000000"},
		{"w", "0001-01-01T00:00", "0100000000000000", "9999-12-31T23:59", "207b753901000000"},
		{"d", "0001-01-01", "01000000", "9999-12-31", "ddb93700"},
		{"7", "0000-W53", "01000000", "9999-W52", "fdf50700"},
		{"x", "0001-01", "01000000", "9999-12", "b4d40100"},
		{"t", "00:00:00", "01000000", "24:00:00", "81510100"},
		{"i", "00:00", "0100", "24:00", "a105"},
		{"c", "01-01", "0100", "12-31", "6e01"},
	}
	for _, b := range boundaries {
		if encT(t, b.exid, b.min) != b.minRaw || decT(t, b.exid, b.minRaw) != b.min {
			t.Fatalf("%s minimum", b.exid)
		}
		if encT(t, b.exid, b.max) != b.maxRaw || decT(t, b.exid, b.maxRaw) != b.max {
			t.Fatalf("%s maximum", b.exid)
		}
	}
}

func TestTemporalJulianGregorianGap(t *testing.T) {
	if encT(t, "d", "1582-10-04") != "c9d00800" || encT(t, "d", "1582-10-15") != "cad00800" {
		t.Fatal("gap ordinals")
	}
	if decT(t, "d", "c9d00800") != "1582-10-04" || decT(t, "d", "cad00800") != "1582-10-15" {
		t.Fatal("gap decode")
	}
	for _, ok := range []string{"1500-02-29", "1600-02-29", "2000-02-29"} {
		if _, err := EncodeClassicTemporal("d", ok, ""); err != nil {
			t.Fatalf("%s should be valid: %v", ok, err)
		}
	}
	for _, bad := range []struct{ v, sub string }{{"1582-10-05", "calendar gap"}, {"1582-10-14", "calendar gap"}, {"1700-02-29", "invalid day"}, {"1900-02-29", "invalid day"}} {
		if _, err := EncodeClassicTemporal("d", bad.v, ""); err == nil || !strings.Contains(err.Error(), bad.sub) {
			t.Fatalf("%s = %v, want %q", bad.v, err, bad.sub)
		}
	}
}

func TestTemporalTimeRanges(t *testing.T) {
	cases := []struct {
		exid   TemporalExid
		v, sub string
	}{
		{"p", "2026-07-15T24:00:00.0000000", "hours must be in 00..23"},
		{"n", "2026-07-15T23:60:00", "minutes must be in 00..59"},
		{"w", "2026-07-15T24:00", "hours must be in 00..23"},
		{"t", "24:00:01", "24:00:00"},
		{"i", "24:01", "24:00"},
		{"t", "23:59:60", "seconds"},
	}
	for _, c := range cases {
		if _, err := EncodeClassicTemporal(c.exid, c.v, ""); err == nil || !strings.Contains(err.Error(), c.sub) {
			t.Fatalf("%s %q = %v, want %q", c.exid, c.v, err, c.sub)
		}
	}
}

func TestTemporalWeek53(t *testing.T) {
	if encT(t, "7", "0000-W53") != "01000000" || encT(t, "7", "0001-W01") != "02000000" || encT(t, "7", "0005-W53") != "06010000" || encT(t, "7", "2020-W53") != "b99b0100" {
		t.Fatal("week53 vectors")
	}
	for _, bad := range []struct{ v, sub string }{{"0000-W52", "only 0000-W53"}, {"0004-W53", "does not have week 53"}, {"2021-W53", "does not have week 53"}} {
		if _, err := EncodeClassicTemporal("7", bad.v, ""); err == nil || !strings.Contains(err.Error(), bad.sub) {
			t.Fatalf("%s = %v, want %q", bad.v, err, bad.sub)
		}
	}
}

func TestTemporalRejectsMalformed(t *testing.T) {
	for _, c := range []struct {
		exid TemporalExid
		v    string
	}{
		{"p", "2002-02-04T20:15:01,1234567"}, {"n", "2002-02-04 20:15:01"}, {"w", "2002-02-04T20:15Z"},
		{"d", "20020204"}, {"7", "2020-w53"}, {"x", "2002-2"}, {"t", "20:15"}, {"i", "20:15:00"}, {"c", "--02-04"},
	} {
		if _, err := EncodeClassicTemporal(c.exid, c.v, ""); err == nil || !strings.Contains(err.Error(), "expects") {
			t.Fatalf("%s %q accepted: %v", c.exid, c.v, err)
		}
	}
	if _, err := EncodeClassicTemporal("p", " 0001-01-01T00:00:00.0000000", ""); err == nil {
		t.Fatal("leading space accepted")
	}
	if _, err := DecodeClassicTemporal("d", make([]byte, 3), ""); err == nil || !strings.Contains(err.Error(), "expects 4 raw bytes") {
		t.Fatalf("short raw: %v", err)
	}
	if _, err := DecodeClassicTemporal("d", mustHexV(t, "deb93700"), ""); err == nil || !strings.Contains(err.Error(), "outside its valid raw range") {
		t.Fatalf("over-range d: %v", err)
	}
	if _, err := DecodeClassicTemporal("i", mustHexV(t, "ffff"), ""); err == nil || !strings.Contains(err.Error(), "outside its valid raw range") {
		t.Fatalf("over-range i: %v", err)
	}
	if _, err := ClassicTemporalByteLength("?"); err == nil || !strings.Contains(err.Error(), "unsupported classic temporal EXID") {
		t.Fatalf("bad exid: %v", err)
	}
}

func TestTemporalExidRoster(t *testing.T) {
	expected := map[TemporalExid]int{"p": 8, "n": 8, "w": 8, "d": 4, "7": 4, "x": 4, "t": 4, "i": 2, "c": 2}
	for exid, width := range expected {
		if !IsClassicTemporalExid(string(exid)) {
			t.Fatalf("%s not recognized", exid)
		}
		if w, _ := ClassicTemporalByteLength(exid); w != width {
			t.Fatalf("%s width = %d, want %d", exid, w, width)
		}
	}
	for _, e := range []string{"D", "T", "P", "?", "", "pp"} {
		if IsClassicTemporalExid(e) {
			t.Fatalf("%q wrongly recognized", e)
		}
	}
}
