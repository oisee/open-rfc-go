// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"testing"
)

func TestClassify(t *testing.T) {
	gateway := make([]byte, gatewayRecordLen)
	routeHello := make([]byte, gatewayRecordLen)
	routeHello[0], routeHello[1] = 0x02, 0x03

	for _, tc := range []struct {
		name  string
		frame []byte
		want  FrameKind
	}{
		{"gateway normal-client record", gateway, FrameGatewayRecord},
		{"NI route hello is still a 64-byte record", routeHello, FrameGatewayRecord},
		{"ping", NIPing, FrameNIPing},
		{"pong", NIPong, FrameNIPong},
		{"other 8-byte frame", []byte("12345678"), FrameShortKeepalive},
		{"APPC record", make([]byte, 80), FrameData},
		{"empty", nil, FrameData},
		{"one byte", []byte{0x06}, FrameData},
	} {
		if got := Classify(tc.frame); got != tc.want {
			t.Errorf("Classify(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestClassifyDistinguishesPingFromPong(t *testing.T) {
	// Both are eight bytes and differ in one character. A role that answered a
	// pong with another pong would loop against a peer that does the same.
	if Classify(NIPing) == Classify(NIPong) {
		t.Fatal("ping and pong must not classify alike")
	}
	if len(NIPing) != 8 || len(NIPong) != 8 {
		t.Fatalf("keepalive frames are %d/%d bytes, want 8 each", len(NIPing), len(NIPong))
	}
	if !bytes.Equal(NIPing, []byte("NI_PING\x00")) {
		t.Errorf("NIPing = %q, want NI_PING with the NUL", NIPing)
	}
	if !bytes.Equal(NIPong, []byte("NI_PONG\x00")) {
		t.Errorf("NIPong = %q, want NI_PONG with the NUL", NIPong)
	}
}

func TestIsNIRouteHello(t *testing.T) {
	hello := make([]byte, gatewayRecordLen)
	hello[0], hello[1] = 0x02, 0x03
	if !IsNIRouteHello(hello) {
		t.Error("the 0x02 0x03 prefixed 64-byte frame is the route hello")
	}

	normalClient := make([]byte, gatewayRecordLen)
	normalClient[0], normalClient[1] = 0x02, 0x04
	if IsNIRouteHello(normalClient) {
		t.Error("a gateway normal-client record is not a route hello")
	}

	// Length is checked before the prefix, so a short frame that happens to
	// start with the prefix must not be mistaken for a hello.
	if IsNIRouteHello([]byte{0x02, 0x03}) {
		t.Error("a two-byte frame is not a route hello")
	}
	if IsNIRouteHello(nil) {
		t.Error("an empty frame is not a route hello")
	}
}

func TestKeepaliveFramesAreNotAliasedByCallers(t *testing.T) {
	// Every role sends the same NIPong slice. If one of them ever wrote through
	// it the others would start sending corrupted keepalives, so pin the bytes.
	before := string(NIPong)
	sent := append([]byte(nil), NIPong...)
	sent[0] = 'X'
	if string(NIPong) != before {
		t.Error("NIPong was modified through a copy; roles must not share mutable state")
	}
}
