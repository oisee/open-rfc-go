// SPDX-License-Identifier: Apache-2.0

package saprouter

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestAdmitTwoHop(t *testing.T) {
	r, err := Admit("/H/router.example/S/3299/H/sap.example/S/3200")
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if r.HopCount != 2 {
		t.Fatalf("hopCount = %d, want 2", r.HopCount)
	}
	if r.ByteLength != 39 {
		t.Fatalf("byteLength = %d, want 39", r.ByteLength)
	}
	wantFirst := FirstHop{Host: "router.example", Service: "3299", UsesDefaultService: false}
	if r.FirstHop != wantFirst {
		t.Fatalf("firstHop = %+v", r.FirstHop)
	}
	wantHops := []Hop{
		{Host: "router.example", Service: "3299", UsesDefaultService: false, PasswordProtected: false},
		{Host: "sap.example", Service: "3200", UsesDefaultService: false, PasswordProtected: false},
	}
	if !reflect.DeepEqual(r.Hops, wantHops) {
		t.Fatalf("hops = %+v", r.Hops)
	}
	if r.RedactedRouteString != "/H/router.example/S/3299/H/sap.example/S/3200" {
		t.Fatalf("redacted = %q", r.RedactedRouteString)
	}
}

func TestAdmitDefaultServiceAndPassword(t *testing.T) {
	r, err := Admit("/H/router.example/W/secret/H/sap.example/S/3200")
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if !r.Hops[0].UsesDefaultService || r.Hops[0].Service != DefaultService {
		t.Fatalf("hop0 should use default service: %+v", r.Hops[0])
	}
	if !r.Hops[0].PasswordProtected {
		t.Fatalf("hop0 should be password protected")
	}
	if r.RedactedRouteString != "/H/router.example/W/[REDACTED]/H/sap.example/S/3200" {
		t.Fatalf("redacted leaks or wrong: %q", r.RedactedRouteString)
	}
	if s := r.String(); bytes.Contains([]byte(s), []byte("secret")) {
		t.Fatalf("String() leaked the password: %q", s)
	}
}

func TestEncodeRouteRequestPayload(t *testing.T) {
	r, err := Admit("/H/router.example/S/3299/H/sap.example/S/3200")
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	p, err := EncodeRouteRequestPayload(r, 0) // default niVersion 40
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(p) != RouteHeaderLength+r.ByteLength {
		t.Fatalf("len = %d, want %d", len(p), RouteHeaderLength+r.ByteLength)
	}
	if !bytes.HasPrefix(p, []byte("NI_ROUTE\x00")) {
		t.Fatalf("eyecatcher missing: % x", p[:9])
	}
	if p[9] != RouteInformationVersion || p[10] != DefaultNIVersion || p[11] != 2 || p[15] != 1 {
		t.Fatalf("header fields wrong: ver=%d ni=%d hops=%d hops-1=%d", p[9], p[10], p[11], p[15])
	}
	if got := binary.BigEndian.Uint32(p[16:]); got != 39 {
		t.Fatalf("route byteLength field = %d, want 39", got)
	}
	if got := binary.BigEndian.Uint32(p[20:]); got != 21 {
		t.Fatalf("first-hop byteLength field = %d, want 21", got)
	}
	wantHops := "router.example\x003299\x00\x00sap.example\x003200\x00\x00"
	if got := string(p[RouteHeaderLength:]); got != wantHops {
		t.Fatalf("hop region = %q, want %q", got, wantHops)
	}
}

func TestEncodeNiVersionRange(t *testing.T) {
	r, _ := Admit("/H/a.example/H/b.example")
	if _, err := EncodeRouteRequestPayload(r, 256); err == nil {
		t.Fatalf("expected niVersion range error")
	}
	if _, err := EncodeRouteRequestPayload(r, 1); err != nil {
		t.Fatalf("niVersion 1 should be valid: %v", err)
	}
}

func TestCompleteRoute(t *testing.T) {
	r, err := CompleteRoute("/H/router.example/S/3299/H/", "sap.example", 3200)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if r.HopCount != 2 || r.Hops[1].Host != "sap.example" || r.Hops[1].Service != "3200" {
		t.Fatalf("unexpected completed route: %+v", r.Hops)
	}
	// A prefix must end with /H/.
	if err := AssertRoutePrefix("/H/router.example/S/3299"); err == nil {
		t.Fatalf("prefix without trailing /H/ should be rejected")
	}
	if err := AssertRoutePrefix("/H/router.example/S/3299/H/"); err != nil {
		t.Fatalf("valid prefix rejected: %v", err)
	}
}

func TestAdmitRejects(t *testing.T) {
	for name, in := range map[string]string{
		"empty":                 "",
		"no leading slash":      "H/a.example/H/b.example",
		"one hop":               "/H/sap.example/S/3200",
		"short host":            "/H/a/H/b.example",
		"password last hop":     "/H/router.example/H/sap.example/W/pw",
		"lowercase token":       "/h/router.example/H/sap.example",
		"P placement":           "/H/router.example/P/pw/H/sap.example",
		"non ascii":             "/H/roûter/H/sap.example",
		"duplicate service":     "/H/router.example/S/1/S/2/H/sap.example",
		"unknown token":         "/H/router.example/X/1/H/sap.example",
		"service then w then s": "/H/router.example/W/pw/S/1/H/sap.example",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Admit(in); err == nil {
				t.Fatalf("expected rejection for %q", in)
			}
		})
	}
}

func TestDecodeRouteResponse(t *testing.T) {
	// NI_PONG → accepted.
	resp, err := DecodeRouteResponse([]byte("NI_PONG\x00"))
	if err != nil || resp.Kind != Accepted {
		t.Fatalf("pong = %+v, %v", resp, err)
	}
	// NI_RTERR with a valid error status → rejected.
	e := make([]byte, 20)
	copy(e, "NI_RTERR\x00")
	e[9] = 40 // niVersion
	e[10] = 0 // opcode
	e[11] = 0 // padding
	rc3 := int32(-3)
	binary.BigEndian.PutUint32(e[12:], uint32(rc3)) // returnCode < 0
	binary.BigEndian.PutUint32(e[16:], 0)           // errorTextByteLength 0
	resp, err = DecodeRouteResponse(e)
	if err != nil {
		t.Fatalf("rterr: %v", err)
	}
	if resp.Kind != Rejected || resp.NIVersion != 40 || resp.ReturnCode != -3 {
		t.Fatalf("rterr decode = %+v", resp)
	}
	// Positive return code is invalid.
	bad := append([]byte(nil), e...)
	binary.BigEndian.PutUint32(bad[12:], 5)
	if _, err := DecodeRouteResponse(bad); err == nil {
		t.Fatalf("expected invalid error status")
	}
	// Unexpected acknowledgement.
	if _, err := DecodeRouteResponse([]byte("HELLO\x00")); err == nil {
		t.Fatalf("expected unexpected-ack error")
	}
	// With error text and the modern 4-byte trailer.
	withText := make([]byte, 20+5+4)
	copy(withText, "NI_RTERR\x00")
	withText[9] = 40
	rc1 := int32(-1)
	binary.BigEndian.PutUint32(withText[12:], uint32(rc1))
	binary.BigEndian.PutUint32(withText[16:], 5)
	resp, err = DecodeRouteResponse(withText)
	if err != nil || resp.ErrorTextByteLength != 5 {
		t.Fatalf("modern rterr = %+v, %v", resp, err)
	}
}

func FuzzDecodeRouteResponse(f *testing.F) {
	f.Add([]byte("NI_PONG\x00"))
	f.Add([]byte("NI_RTERR\x00"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeRouteResponse(data)
	})
}

func FuzzAdmit(f *testing.F) {
	f.Add("/H/router.example/S/3299/H/sap.example/S/3200")
	f.Fuzz(func(t *testing.T, s string) {
		if r, err := Admit(s); err == nil {
			// A successfully admitted route must always encode.
			if _, err := EncodeRouteRequestPayload(r, 0); err != nil {
				t.Fatalf("admitted route failed to encode: %v (%q)", err, s)
			}
		}
	})
}

// --- exact oracle vectors from test/saprouter-route.test.ts ------------------

func hexb(t *testing.T, s string) []byte {
	t.Helper()
	clean := ""
	for _, r := range s {
		if r != ' ' && r != '\n' && r != '\t' {
			clean += string(r)
		}
	}
	b := make([]byte, len(clean)/2)
	for i := 0; i < len(b); i++ {
		var v int
		_, err := fmtSscanHexByte(clean[2*i:2*i+2], &v)
		if err != nil {
			t.Fatalf("bad hex at %d: %v", i, err)
		}
		b[i] = byte(v)
	}
	return b
}

func fmtSscanHexByte(s string, v *int) (int, error) {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		var d int
		switch {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case c >= 'a' && c <= 'f':
			d = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = int(c-'A') + 10
		default:
			return 0, errBadHex
		}
		n = n*16 + d
	}
	*v = n
	return 1, nil
}

var errBadHex = errBad("bad hex")

type errBad string

func (e errBad) Error() string { return string(e) }

func TestEncodeExactOracleVector(t *testing.T) {
	r, err := Admit("/H/router/S/3299/W/secret/H/target/S/sapgw01")
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if r.ByteLength != 35 {
		t.Fatalf("byteLength = %d, want 35", r.ByteLength)
	}
	got, err := EncodeRouteRequestPayload(r, 0)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := hexb(t, `
		4e 49 5f 52 4f 55 54 45 00 02 28 02 00 00 00 01
		00 00 00 23 00 00 00 13 72 6f 75 74 65 72 00 33
		32 39 39 00 73 65 63 72 65 74 00 74 61 72 67 65
		74 00 73 61 70 67 77 30 31 00 00`)
	if !bytes.Equal(got, want) {
		t.Fatalf("payload mismatch:\n got % x\nwant % x", got, want)
	}
	// niVersion override changes only byte 10.
	got36, _ := EncodeRouteRequestPayload(r, 36)
	if got36[10] != 36 {
		t.Fatalf("niVersion override: byte[10]=%d, want 36", got36[10])
	}
}

func TestAdmitThreeHopOracle(t *testing.T) {
	r, err := Admit("/H/router.example.test/W/first-secret/H/second_router/S/saprouter/W/second-secret/H/application.internal/S/sapgw01")
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if r.HopCount != 3 || r.ByteLength != 102 {
		t.Fatalf("hopCount=%d byteLength=%d, want 3/102", r.HopCount, r.ByteLength)
	}
	if !r.Hops[0].UsesDefaultService || !r.Hops[0].PasswordProtected {
		t.Fatalf("hop0 = %+v", r.Hops[0])
	}
	want := "/H/router.example.test/W/[REDACTED]/H/second_router/S/saprouter/W/[REDACTED]/H/application.internal/S/sapgw01"
	if r.RedactedRouteString != want {
		t.Fatalf("redacted = %q", r.RedactedRouteString)
	}
	// The wire form writes an EMPTY service field for the default-service hop.
	p, _ := EncodeRouteRequestPayload(r, 0)
	if !bytes.HasPrefix(p[RouteHeaderLength:], []byte("router.example.test\x00\x00first-secret\x00")) {
		t.Fatalf("default-service hop should emit an empty service field: % x", p[RouteHeaderLength:RouteHeaderLength+35])
	}
}

func TestCompleteRouteOracle(t *testing.T) {
	r, err := CompleteRoute("/H/router.example.test/S/3299/W/router-secret/H/", "gateway.internal", 3342)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if r.HopCount != 2 || r.ByteLength != 62 {
		t.Fatalf("hopCount=%d byteLength=%d, want 2/62", r.HopCount, r.ByteLength)
	}
	if r.RedactedRouteString != "/H/router.example.test/S/3299/W/[REDACTED]/H/gateway.internal/S/3342" {
		t.Fatalf("redacted = %q", r.RedactedRouteString)
	}
}

func TestDecodeRTERROracle(t *testing.T) {
	modern := hexb(t, `
		4e 49 5f 52 54 45 52 52 00 28 00 00 ff ff ff a2
		00 00 00 17 72 6f 75 74 65 20 70 65 72 6d 69 73
		73 69 6f 6e 20 64 65 6e 69 65 64 00 00 00 00`)
	resp, err := DecodeRouteResponse(modern)
	if err != nil {
		t.Fatalf("modern: %v", err)
	}
	if resp.Kind != Rejected || resp.NIVersion != 40 || resp.ReturnCode != -94 || resp.ErrorTextByteLength != 23 {
		t.Fatalf("modern rterr = %+v", resp)
	}
	documented := modern[:len(modern)-4]
	resp, err = DecodeRouteResponse(documented)
	if err != nil {
		t.Fatalf("documented: %v", err)
	}
	if resp.ReturnCode != -94 || resp.ErrorTextByteLength != 23 {
		t.Fatalf("documented rterr = %+v", resp)
	}
}
