// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/cpic.test.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten for the testing package.
// Cases with no Go analogue are dropped and recorded in docs/provenance.md:
// the {tag:-1} range case (uint16), the non-integer limit cases (1.5 / NaN;
// int-typed), the Proxy-based CUT preflight test and the geometry-override
// framing test (Go len() cannot be spoofed and a 256 MiB allocation is not a
// unit test), the scramble(null) type case, and the "value" in field /
// Object.isFrozen assertions.

package cpic

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/scramble"
)

func fld(tag Tag, value []byte) Field { return Field{Tag: uint16(tag), Value: value} }

func hx(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

func mustChain(t *testing.T, prev Tag, fields []Field) []byte {
	t.Helper()
	b, err := EncodeFieldChain(uint16(prev), fields, FieldChainLimits{})
	if err != nil {
		t.Fatalf("EncodeFieldChain: %v", err)
	}
	return b
}

func TestEncodesDecodesChainedGrammar(t *testing.T) {
	encoded := mustChain(t, TagSession, []Field{
		fld(TagClient, []byte("001")),
		fld(TagUser, []byte("USER")),
		fld(TagEnd, nil),
	})
	if hex.EncodeToString(encoded) != "051401140003303031011401110004555345520111ffff0000" {
		t.Fatalf("encoded = %s", hex.EncodeToString(encoded))
	}
	decoded, err := DecodeFieldChain(encoded, uint16(TagSession), FieldChainLimits{})
	if err != nil {
		t.Fatal(err)
	}
	want := []Field{fld(TagClient, []byte("001")), fld(TagUser, []byte("USER")), {Tag: uint16(TagEnd), Value: []byte{}}}
	if !fieldsEqual(decoded, want) {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func fieldsEqual(a, b []Field) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Tag != b[i].Tag || !bytes.Equal(a[i].Value, b[i].Value) {
			return false
		}
	}
	return true
}

func TestDecodesPrefixWithoutTrailer(t *testing.T) {
	fields := mustChain(t, TagSession, []Field{fld(TagClient, []byte("001")), fld(TagEnd, nil)})
	trailer := hx(t, "ffff0000012000008500")
	message := concat(fields, trailer)
	decoded, err := DecodeFieldChainPrefix(message, uint16(TagSession), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if decoded.BytesConsumed != len(fields) || !bytes.Equal(message[decoded.BytesConsumed:], trailer) {
		t.Fatalf("bytesConsumed = %d", decoded.BytesConsumed)
	}
	n := len(fields)
	if b, err := DecodeFieldChainPrefix(message, uint16(TagSession), uint16(TagEnd), FieldChainLimits{MaxChainLength: &n}); err != nil || b.BytesConsumed != len(fields) {
		t.Fatalf("exactly bounded = %v", err)
	}
	nm1 := len(fields) - 1
	if _, err := DecodeFieldChainPrefix(message, uint16(TagSession), uint16(TagEnd), FieldChainLimits{MaxChainLength: &nm1}); err == nil || !strings.Contains(err.Error(), "exceeds configured limit") {
		t.Fatalf("under limit = %v", err)
	}
}

func TestRequiresTerminalTag(t *testing.T) {
	fields := mustChain(t, TagSession, []Field{fld(TagClient, []byte("001"))})
	if _, err := DecodeFieldChainPrefix(fields, uint16(TagSession), uint16(TagEnd), FieldChainLimits{}); err == nil || !strings.Contains(err.Error(), "ended before terminal tag 0xffff") {
		t.Fatalf("err = %v", err)
	}
}

func TestRejectsBrokenChainAndTruncated(t *testing.T) {
	broken := hx(t, "05140114000330303100010111000455534552")
	if _, err := DecodeFieldChain(broken, uint16(TagSession), FieldChainLimits{}); err == nil || !strings.Contains(err.Error(), "expected previous tag 0x0114") || !strings.Contains(err.Error(), "received 0x0001") {
		t.Fatalf("broken = %v", err)
	}
	truncated := hx(t, "051401140004303031")
	if _, err := DecodeFieldChain(truncated, uint16(TagSession), FieldChainLimits{}); err == nil || !strings.Contains(err.Error(), "need 4 bytes") {
		t.Fatalf("truncated = %v", err)
	}
}

func TestCompactAndExtendedRfcproInChains(t *testing.T) {
	compact := mustChain(t, TagParameterName, []Field{fld(TagParameterValue, bytesRep(0x11, 65_534))})
	if hex.EncodeToString(compact[0:6]) != "02010203fffe" {
		t.Fatalf("compact header = %s", hex.EncodeToString(compact[0:6]))
	}
	if d, err := DecodeFieldChain(compact, uint16(TagParameterName), FieldChainLimits{}); err != nil || !bytes.Equal(d[0].Value, bytesRep(0x11, 65_534)) {
		t.Fatalf("compact decode = %v", err)
	}
	sentinel := mustChain(t, TagParameterName, []Field{fld(TagParameterValue, bytesRep(0x22, 65_535))})
	if hex.EncodeToString(sentinel[0:10]) != "02010203ffff0000ffff" {
		t.Fatalf("sentinel header = %s", hex.EncodeToString(sentinel[0:10]))
	}
	extended := mustChain(t, TagParameterName, []Field{fld(TagParameterValue, bytesRep(0x33, 65_536)), fld(TagEnd, nil)})
	if hex.EncodeToString(extended[0:10]) != "02010203ffff00010000" {
		t.Fatalf("extended header = %s", hex.EncodeToString(extended[0:10]))
	}
	if d, err := DecodeFieldChain(extended, uint16(TagParameterName), FieldChainLimits{}); err != nil || len(d[0].Value) != 65_536 {
		t.Fatalf("extended decode = %v", err)
	}
}

func TestRejectsBoundedExtendedFieldLengths(t *testing.T) {
	fl := 65_535
	if _, err := EncodeFieldChain(uint16(TagSession), []Field{fld(TagUser, make([]byte, 0x1_0000))}, FieldChainLimits{MaxFieldLength: &fl}); err == nil || !strings.Contains(err.Error(), "field length 65536 exceeds configured limit 65535") {
		t.Fatalf("encode = %v", err)
	}
	advertised := hx(t, "02010203ffff00010000")
	if _, err := DecodeFieldChain(advertised, uint16(TagParameterName), FieldChainLimits{MaxFieldLength: &fl}); err == nil || !strings.Contains(err.Error(), "exceeds configured limit 65535") {
		t.Fatalf("decode = %v", err)
	}
	fl2, cl2 := 65_535, 65_545
	atLimit, err := EncodeFieldChain(uint16(TagParameterName), []Field{fld(TagParameterValue, make([]byte, 65_535))}, FieldChainLimits{MaxFieldLength: &fl2, MaxChainLength: &cl2})
	if err != nil {
		t.Fatal(err)
	}
	d, err := DecodeFieldChain(atLimit, uint16(TagParameterName), FieldChainLimits{MaxFieldLength: &fl2, MaxChainLength: &cl2})
	if err != nil || len(d[0].Value) != 65_535 {
		t.Fatalf("at limit = %v", err)
	}
}

func TestEnforcesChainByteAndFieldCountLimits(t *testing.T) {
	fields := []Field{fld(TagClient, []byte("001")), fld(TagEnd, nil)}
	encoded := mustChain(t, TagSession, fields)
	cl, fc := len(encoded), len(fields)
	if b, err := EncodeFieldChain(uint16(TagSession), fields, FieldChainLimits{MaxChainLength: &cl, MaxFieldCount: &fc}); err != nil || len(b) != len(encoded) {
		t.Fatalf("at limit = %v", err)
	}
	clm1 := len(encoded) - 1
	if _, err := EncodeFieldChain(uint16(TagSession), fields, FieldChainLimits{MaxChainLength: &clm1}); err == nil || !strings.Contains(err.Error(), "exceeds configured limit") {
		t.Fatalf("chain limit = %v", err)
	}
	fcm1 := len(fields) - 1
	if _, err := EncodeFieldChain(uint16(TagSession), fields, FieldChainLimits{MaxFieldCount: &fcm1}); err == nil || !strings.Contains(err.Error(), "field count 2 exceeds configured limit 1") {
		t.Fatalf("field count = %v", err)
	}
	if _, err := DecodeFieldChain(encoded, uint16(TagSession), FieldChainLimits{MaxChainLength: &clm1}); err == nil || !strings.Contains(err.Error(), "exceeds configured limit") {
		t.Fatalf("decode chain limit = %v", err)
	}
	if _, err := DecodeFieldChain(encoded, uint16(TagSession), FieldChainLimits{MaxFieldCount: &fcm1}); err == nil || !strings.Contains(err.Error(), "field count exceeds configured limit 1") {
		t.Fatalf("decode field count = %v", err)
	}
}

func TestRejectsInvalidChainLimitOptions(t *testing.T) {
	empty := []byte{}
	neg := -1
	for _, limits := range []FieldChainLimits{{MaxFieldLength: &neg}, {MaxFieldCount: &neg}} {
		if _, err := DecodeFieldChain(empty, uint16(TagSession), limits); err == nil || !errors.Is(err, ErrRange) {
			t.Fatalf("limits %+v = %v", limits, err)
		}
	}
}

func TestRejectsOversizedTruncatedExtendedBeforeCopy(t *testing.T) {
	maxFitting := DefaultMaxFieldChainLength - 10
	advertised := make([]byte, 10)
	be16(advertised, 0, uint16(TagParameterName))
	be16(advertised, 2, uint16(TagParameterValue))
	be16(advertised, 4, 0xffff)
	be32(advertised, 6, uint32(maxFitting))
	if _, err := DecodeFieldChain(advertised, uint16(TagParameterName), FieldChainLimits{}); err == nil || !strings.Contains(err.Error(), "need 268435446 bytes") {
		t.Fatalf("oversized fit = %v", err)
	}
	be32(advertised, 6, uint32(maxFitting+1))
	if _, err := DecodeFieldChain(advertised, uint16(TagParameterName), FieldChainLimits{}); err == nil || !strings.Contains(err.Error(), "exceeds configured limit") {
		t.Fatalf("over limit = %v", err)
	}
}

func TestRejectsClosingTagMismatchAfterExtended(t *testing.T) {
	encoded := mustChain(t, TagParameterName, []Field{fld(TagParameterValue, make([]byte, 65_535)), fld(TagEnd, nil)})
	be16(encoded, 65_545, uint16(TagParameterName))
	if _, err := DecodeFieldChain(encoded, uint16(TagParameterName), FieldChainLimits{}); err == nil || !strings.Contains(err.Error(), "expected previous tag 0x0203") || !strings.Contains(err.Error(), "received 0x0201") {
		t.Fatalf("mismatch = %v", err)
	}
}

func TestStreamingCutTrailerAboveBoundary(t *testing.T) {
	value := bytesRep(0x22, 65_536)
	encoded, err := EncodeCutFunctionRequest(CutFunctionRequestInput{FunctionName: "RFC_PING", Imports: []NamedValue{{Name: "INPUT", Value: value}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) <= 0xffff || hex.EncodeToString(encoded[len(encoded)-6:]) != "ffff0000ffff" {
		t.Fatalf("streamed trailer = %s", hex.EncodeToString(encoded[len(encoded)-6:]))
	}
	framing, err := InspectRequestAppcFraming(encoded)
	if err != nil || framing != (RequestAppcFraming{Mode: "streamed", ApplicationDataLength: len(encoded), FinalSapParameterLength: 0}) {
		t.Fatalf("framing = %+v, %v", framing, err)
	}
	decoded, err := DecodeFieldChainPrefix(encoded[4:], uint16(TagContextEnd), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, f := range decoded.Fields {
		if f.Tag == uint16(TagParameterValue) {
			found = len(f.Value)
		}
	}
	if found != len(value) {
		t.Fatalf("param value len = %d", found)
	}
}

func TestSwitchesBetween28000And28001(t *testing.T) {
	compact, err := EncodeCutFunctionRequest(CutFunctionRequestInput{FunctionName: "RFC_PING", Imports: []NamedValue{{Name: "INPUT", Value: make([]byte, 27_926)}}})
	if err != nil {
		t.Fatal(err)
	}
	streamed, err := EncodeCutFunctionRequest(CutFunctionRequestInput{FunctionName: "RFC_PING", Imports: []NamedValue{{Name: "INPUT", Value: make([]byte, 27_927)}}})
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := InspectRequestAppcFraming(compact); f != (RequestAppcFraming{Mode: "compact", ApplicationDataLength: 28_000, FinalSapParameterLength: 8}) {
		t.Fatalf("compact = %+v", f)
	}
	if f, _ := InspectRequestAppcFraming(streamed); f != (RequestAppcFraming{Mode: "streamed", ApplicationDataLength: 28_001, FinalSapParameterLength: 0}) {
		t.Fatalf("streamed = %+v", f)
	}
}

func TestDistinguishesCompactFromStreamed(t *testing.T) {
	compact, err := EncodeCutFunctionRequest(CutFunctionRequestInput{FunctionName: "RFC_PING"})
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := InspectRequestAppcFraming(compact); f != (RequestAppcFraming{Mode: "compact", ApplicationDataLength: len(compact) - 8, FinalSapParameterLength: 8}) {
		t.Fatalf("compact = %+v", f)
	}
	for _, malformed := range [][]byte{{}, compact[:len(compact)-1]} {
		if _, err := InspectRequestAppcFraming(malformed); err == nil || !strings.Contains(err.Error(), "invalid APPC framing trailer") {
			t.Fatalf("malformed = %v", err)
		}
	}
}

func TestRejectsOversizedResponseTrailer(t *testing.T) {
	chain := mustChain(t, TagResponseStart, []Field{fld(TagUnresolved0420, make([]byte, 4)), fld(TagEnd, nil)})
	malformed := concat(hx(t, "05000000"), chain, bytesRep(0xff, 64*1024))
	if _, err := DecodeFunctionResultFields(malformed); err == nil || !strings.Contains(err.Error(), "response trailer is invalid") {
		t.Fatalf("err = %v", err)
	}
}

func TestRejectsTruncatedExtendedLengthBeforeValue(t *testing.T) {
	header := hx(t, "02010203ffff00010000")
	for length := 1; length < len(header); length++ {
		_, err := DecodeFieldChain(header[:length], uint16(TagParameterName), FieldChainLimits{})
		if err == nil || (!strings.Contains(err.Error(), "need 4 bytes") && !strings.Contains(err.Error(), "need 6 bytes")) {
			t.Fatalf("truncation at %d = %v", length, err)
		}
	}
}

func TestScramblePinnedVector(t *testing.T) {
	out, err := scramble.ScrambleRfcPassword("secret", 0x5ae0_b7a3)
	if err != nil || hex.EncodeToString(out) != "a3b7e05a048eaa683470" {
		t.Fatalf("out = %x, %v", out, err)
	}
}

func TestScrambleFreshSeedAndAscii(t *testing.T) {
	first, _ := scramble.ScrambleRfcPasswordRandomSeed("secret")
	second, _ := scramble.ScrambleRfcPasswordRandomSeed("secret")
	if len(first) != 10 || len(second) != 10 || bytes.Equal(first, second) {
		t.Fatalf("fresh seed: %x %x", first, second)
	}
	if _, err := scramble.ScrambleRfcPassword("pässword", 0); err == nil {
		t.Fatal("non-ascii accepted")
	}
	if _, err := scramble.ScrambleRfcPassword(strings.Repeat("x", 41), 0); err == nil {
		t.Fatal("overlong accepted")
	}
}

func captureLogonRequest(t *testing.T) []byte {
	t.Helper()
	seed := uint32(0x1234_5678)
	encoded, err := EncodeInitialLogonRequest(InitialLogonRequestInput{
		Client: "001", User: "RFCUSR", Password: strings.Repeat("x", 25), Language: "E",
		ClientAddress: "127.0.0.1", PartnerHostName: "host.example.test", Destination: "127.0.0.1",
		ProgramName: "open-rfc01", SessionID: bytesRep(0x5a, 16), PasswordSeed: &seed,
	})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestEncodesCaptureSizedInitialLogonRequest(t *testing.T) {
	encoded := captureLogonRequest(t)
	if len(encoded) != 296 {
		t.Fatalf("len = %d", len(encoded))
	}
	if hex.EncodeToString(encoded[0:18]) != "d9c6c3f0f0f0f0f0f0f0f0f0010100080301" {
		t.Fatalf("prefix = %s", hex.EncodeToString(encoded[0:18]))
	}
	if hex.EncodeToString(encoded[len(encoded)-10:]) != "ffff0000012000008500" {
		t.Fatalf("trailer = %s", hex.EncodeToString(encoded[len(encoded)-10:]))
	}
	decoded, err := DecodeInitialLogonRequest(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.CpicPacketSize != 288 || decoded.MaximumRfcPacketSize != 0x8500 {
		t.Fatalf("decoded = %+v", decoded)
	}
	wantTags := [][2]int{{0x0101, 0}, {0x0103, 4}, {0x0106, 11}, {0x0337, 0}, {0x0514, 16}, {0x0114, 3}, {0x0111, 6}, {0x0117, 29}, {0x0115, 1}, {0x0501, 1}, {0x0007, 9}, {0x0018, 3}, {0x0011, 1}, {0x0012, 3}, {0x0013, 3}, {0x0008, 17}, {0x0006, 9}, {0x0130, 10}, {0x0502, 0}, {0x000b, 3}, {0x0102, 7}, {0xffff, 0}}
	if len(decoded.Fields) != len(wantTags) {
		t.Fatalf("field count = %d", len(decoded.Fields))
	}
	for i, w := range wantTags {
		if int(decoded.Fields[i].Tag) != w[0] || decoded.Fields[i].ByteLength != w[1] {
			t.Fatalf("field %d = %+v, want %v", i, decoded.Fields[i], w)
		}
	}
}

func TestRejectsMalformedInitialLogonFields(t *testing.T) {
	base := func() InitialLogonRequestInput {
		seed := uint32(1)
		return InitialLogonRequestInput{Client: "001", User: "RFCUSR", Password: "secret", Language: "E",
			ClientAddress: "127.0.0.1", PartnerHostName: "host.example.test", Destination: "127.0.0.1",
			ProgramName: "open-rfc", SessionID: make([]byte, 16), PasswordSeed: &seed}
	}
	bad := base()
	bad.Client = "01"
	if _, err := EncodeInitialLogonRequest(bad); err == nil || !strings.Contains(err.Error(), "three ASCII digits") {
		t.Fatalf("client = %v", err)
	}
	bad = base()
	bad.User = "ÜSER"
	if _, err := EncodeInitialLogonRequest(bad); err == nil || !strings.Contains(err.Error(), "user") {
		t.Fatalf("user = %v", err)
	}
	brokenChain := captureLogonRequest(t)
	be16(brokenChain, 26, 0x0104)
	if _, err := DecodeInitialLogonRequest(brokenChain); err == nil || !strings.Contains(err.Error(), "expected previous tag 0x0104") {
		t.Fatalf("broken chain = %v", err)
	}
	brokenSize := captureLogonRequest(t)
	be16(brokenSize, len(brokenSize)-6, 1)
	if _, err := DecodeInitialLogonRequest(brokenSize); err == nil || !strings.Contains(err.Error(), "packet size 1 does not match derived size") {
		t.Fatalf("broken size = %v", err)
	}
}

func regularResponse(t *testing.T, fields []Field) []byte {
	t.Helper()
	return concat(hx(t, "010100080101010504010003"), mustChain(t, TagStart, fields), []byte{0xff, 0xff})
}

func TestDecodesRedactionSafeInitialResponse(t *testing.T) {
	fields := []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")),
		fld(TagCapabilities, hx(t, "0401000300030200000023")),
		fld(TagLogonStatus, []byte{0}),
		fld(TagSystemCodePage, []byte{'1', 0, '1', 0, '0', 0, '0', 0}),
		fld(TagEnd, nil),
	}
	response := regularResponse(t, fields)
	decoded, err := DecodeInitialLogonResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Success || decoded.Status == nil || *decoded.Status != 0 || decoded.NegotiatedProtocolVersion != 0x0e0b {
		t.Fatalf("decoded = %+v", decoded)
	}
	wantFields := []TagLen{{0x0103, 4}, {0x0106, 11}, {0x0161, 1}, {0x0016, 8}, {0xffff, 0}}
	if !reflect.DeepEqual(decoded.Fields, wantFields) {
		t.Fatalf("fields = %+v", decoded.Fields)
	}

	rejected := regularResponse(t, fields)
	statusOffset := 12 + 6 + 4 + 6 + 11 + 6
	rejected[statusOffset] = 7
	dec2, err := DecodeInitialLogonResponse(rejected)
	if err != nil {
		t.Fatal(err)
	}
	if dec2.Success || dec2.Status == nil || *dec2.Status != 7 {
		t.Fatalf("rejected = %+v", dec2)
	}
}

func TestAcceptsNw750AndS4LogonStatusForms(t *testing.T) {
	nw := regularResponse(t, []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")),
		fld(TagCapabilities, make([]byte, 11)),
		fld(TagUnresolved0420, make([]byte, 4)),
		fld(TagEnd, nil),
	})
	if d, err := DecodeInitialLogonResponse(nw); err != nil || !d.Success || *d.Status != 0 {
		t.Fatalf("nw = %+v, %v", d, err)
	}

	rejected := regularResponse(t, []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")),
		fld(TagLogonStatus, []byte{7}),
		fld(TagUnresolved0420, make([]byte, 4)),
		fld(TagEnd, nil),
	})
	if d, err := DecodeInitialLogonResponse(rejected); err != nil || d.Success || *d.Status != 7 {
		t.Fatalf("rejected companion = %+v, %v", d, err)
	}

	s4 := regularResponse(t, []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")),
		fld(TagCapabilities, make([]byte, 11)),
		fld(TagLogonStatus, []byte{0}),
		fld(TagUnresolved0420, make([]byte, 4)),
		fld(Tag(0x0450), make([]byte, 6)),
		fld(TagSystemCodePage, make([]byte, 8)),
		fld(TagEnd, nil),
	})
	if d, err := DecodeInitialLogonResponse(s4); err != nil || !d.Success || *d.Status != 0 {
		t.Fatalf("s4 = %+v, %v", d, err)
	}

	malformed := map[string][]Field{
		"wrong length":      {fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagCapabilities, make([]byte, 11)), fld(TagLogonStatus, []byte{0}), fld(TagUnresolved0420, make([]byte, 4)), fld(Tag(0x0450), make([]byte, 5)), fld(TagEnd, nil)},
		"wrong position":    {fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagCapabilities, make([]byte, 11)), fld(TagLogonStatus, []byte{0}), fld(Tag(0x0450), make([]byte, 6)), fld(TagUnresolved0420, make([]byte, 4)), fld(TagEnd, nil)},
		"missing companion": {fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagCapabilities, make([]byte, 11)), fld(TagLogonStatus, []byte{0}), fld(TagSystemCodePage, make([]byte, 8)), fld(Tag(0x0450), make([]byte, 6)), fld(TagEnd, nil)},
		"duplicate":         {fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagCapabilities, make([]byte, 11)), fld(TagLogonStatus, []byte{0}), fld(TagUnresolved0420, make([]byte, 4)), fld(Tag(0x0450), make([]byte, 6)), fld(Tag(0x0450), make([]byte, 6)), fld(TagEnd, nil)},
	}
	for name, f := range malformed {
		if _, err := DecodeInitialLogonResponse(regularResponse(t, f)); StructureRuleOf(err) != "malformed-vendor-logon-control" {
			t.Fatalf("%s = %v (rule %q)", name, err, StructureRuleOf(err))
		}
	}

	statusCases := []struct {
		name   string
		fields []Field
		rule   string
	}{
		{"nonzero call-only", []Field{fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagUnresolved0420, hx(t, "00000001")), fld(TagEnd, nil)}, "nonzero-call-status"},
		{"conflicting nonzero", []Field{fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagLogonStatus, []byte{0}), fld(TagUnresolved0420, hx(t, "00000001")), fld(TagEnd, nil)}, "nonzero-call-status"},
		{"malformed call status", []Field{fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagUnresolved0420, make([]byte, 3)), fld(TagEnd, nil)}, "malformed-call-status"},
		{"duplicate call status", []Field{fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagUnresolved0420, make([]byte, 4)), fld(TagUnresolved0420, make([]byte, 4)), fld(TagEnd, nil)}, "malformed-call-status"},
		{"missing status", []Field{fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagEnd, nil)}, "missing-logon-status"},
		{"duplicate one-byte", []Field{fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagLogonStatus, []byte{0}), fld(TagLogonStatus, []byte{0}), fld(TagEnd, nil)}, "malformed-one-byte-status"},
	}
	for _, c := range statusCases {
		if _, err := DecodeInitialLogonResponse(regularResponse(t, c.fields)); StructureRuleOf(err) != c.rule {
			t.Fatalf("%s = %v (rule %q, want %q)", c.name, err, StructureRuleOf(err), c.rule)
		}
	}
}

func bytesRep(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func be16(b []byte, off int, v uint16) { b[off] = byte(v >> 8); b[off+1] = byte(v) }
func be32(b []byte, off int, v uint32) {
	b[off] = byte(v >> 24)
	b[off+1] = byte(v >> 16)
	b[off+2] = byte(v >> 8)
	b[off+3] = byte(v)
}
