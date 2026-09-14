// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"testing"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

// Eclipse's framing is the CUT framing with the first predecessor left off, so
// the two must decode to the same request.
//
// Built rather than captured: a real call carries a session and a user, and
// what is under test is the framing, which a synthetic chain exercises
// exactly as well.
func TestDecodeFunctionRequestReadsEclipseFraming(t *testing.T) {
	const want = "SADT_REST_RFC_ENDPOINT"
	fields := []cpic.Field{
		{Tag: uint16(cpic.TagStart), Value: []byte{0x04, 0x02, 0x01, 0x05, 0x04, 0x01, 0x00, 0x02}},
		{Tag: uint16(cpic.TagFunction), Value: utf16BE(want)},
		{Tag: uint16(cpic.TagEnd)},
	}
	chain, err := cpic.EncodeFieldChain(0x0000, fields, cpic.FieldChainLimits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// What Eclipse puts on the wire: the same chain without the two bytes that
	// would name the first field's predecessor, because it has none.
	onTheWire := chain[2:]

	req, err := DecodeFunctionRequest(onTheWire)
	if err != nil {
		t.Fatalf("Eclipse framing: %v", err)
	}
	if req.FunctionName != want {
		t.Errorf("the function is %q, and Eclipse asked for %q", req.FunctionName, want)
	}
}

// The name Eclipse sends is big-endian, and reading it the other way is not a
// garbled string but a different one — which reaches the dispatcher as a plain
// lookup miss. This is the test that would have named that.
func TestDecodeFunctionRequestReadsEitherByteOrder(t *testing.T) {
	const want = "SADT_REST_RFC_ENDPOINT"
	for _, order := range []struct {
		name     string
		declared byte
		value    []byte
	}{
		{"big-endian, as Eclipse sends it", 0x02, utf16BE(want)},
		{"little-endian, as this server writes it", 0x01, utf16LE(want)},
		// Nothing recognisable in the opening field: the zero bytes decide.
		{"undeclared, big-endian text", 0x7f, utf16BE(want)},
		{"undeclared, little-endian text", 0x7f, utf16LE(want)},
	} {
		fields := []cpic.Field{
			{Tag: uint16(cpic.TagStart), Value: []byte{0x04, order.declared, 0x01, 0x05, 0x04, 0x01, 0x00, 0x02}},
			{Tag: uint16(cpic.TagFunction), Value: order.value},
			{Tag: uint16(cpic.TagEnd)},
		}
		chain, err := cpic.EncodeFieldChain(0x0000, fields, cpic.FieldChainLimits{})
		if err != nil {
			t.Fatalf("%s: encode: %v", order.name, err)
		}
		req, err := DecodeFunctionRequest(chain[2:])
		if err != nil {
			t.Fatalf("%s: %v", order.name, err)
		}
		if req.FunctionName != want {
			t.Errorf("%s: read %q", order.name, req.FunctionName)
		}
	}
}

func utf16BE(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		out = append(out, byte(r>>8), byte(r))
	}
	return out
}

func utf16LE(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		out = append(out, byte(r), byte(r>>8))
	}
	return out
}

// A logon is told from a call by what it is, not by what it lacks.
//
// The regression this pins: both were once distinguished by the absence of the
// CUT prefix, and Eclipse's calls have no CUT prefix either, so every call was
// answered with a logon template.
func TestEclipseLogonIsRecognisedByItsOwnPrefix(t *testing.T) {
	logon := append([]byte{0xd9, 0xc6, 0xc3}, []byte("000000000")...)
	if !isEclipseLogon(logon) {
		t.Error("a logon opening with EBCDIC \"RFC\" was not recognised")
	}
	call := []byte{0x01, 0x01, 0x00, 0x08, 0x04, 0x02, 0x01, 0x05}
	if isEclipseLogon(call) {
		t.Error("a field chain was taken for a logon")
	}
	if isEclipseLogon(nil) {
		t.Error("an empty payload was taken for a logon")
	}
}
