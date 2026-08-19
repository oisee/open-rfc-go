// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/gateway.test.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten for the testing package. A
// FuzzDecodeNormalClient target is added per the milestone's decoder-fuzzing
// rule. See docs/provenance.md.

package gateway

import (
	"bytes"
	"errors"
	"testing"
)

func sampleRecord() NormalClientRecord {
	return NormalClientRecord{
		Address:            "127.0.0.1",
		Service:            "sapgw00",
		CodePage:           "1100",
		GatewayOptionLevel: 6,
		LogicalUnit:        "LOCAL",
		TransactionProgram: "NWRFC",
		ConversationID:     "",
		AppcHeaderVersion:  6,
		AcceptInfo: AcceptInfoErrorInfo | AcceptInfoPing | AcceptInfoConnectionExtended |
			AcceptInfoExtendedInitOptions | AcceptInfoDistributedTrace,
		Index:      -1,
		ReturnCode: 0,
		EchoData:   0,
	}
}

func TestRoundTripsProven64ByteRecord(t *testing.T) {
	rec := sampleRecord()
	encoded, err := EncodeNormalClient(rec)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != 64 {
		t.Fatalf("len = %d, want 64", len(encoded))
	}
	if encoded[0] != 2 || encoded[1] != 3 {
		t.Fatalf("version/requestType = %d/%d, want 2/3", encoded[0], encoded[1])
	}
	if got := string(encoded[10:20]); got != "sapgw00\x00\x00\x00" {
		t.Fatalf("service region = %q", got)
	}
	decoded, err := DecodeNormalClient(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != rec {
		t.Fatalf("round-trip = %+v, want %+v", decoded, rec)
	}
}

func TestKeepsPaddingAndOptionLevelExplicit(t *testing.T) {
	encoded, err := EncodeNormalClient(sampleRecord())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded[24:29], make([]byte, 5)) {
		t.Fatalf("reserved2 = %x, want five zeroes", encoded[24:29])
	}
	if encoded[29] != 6 {
		t.Fatalf("gatewayOptionLevel = %d, want 6", encoded[29])
	}
	if got := string(encoded[46:54]); got != "        " {
		t.Fatalf("conversationId region = %q, want eight spaces", got)
	}
}

func TestRejectsUnsupportedVariantsAndMalformedFields(t *testing.T) {
	base := sampleRecord()

	bad := base
	bad.Address = "::1"
	if _, err := EncodeNormalClient(bad); !errors.Is(err, ErrRange) {
		t.Fatalf("IPv6 address = %v, want ErrRange", err)
	}

	bad = base
	bad.LogicalUnit = "123456789"
	if _, err := EncodeNormalClient(bad); !errors.Is(err, ErrRange) {
		t.Fatalf("overlong logicalUnit = %v, want ErrRange", err)
	}

	version3, err := EncodeNormalClient(base)
	if err != nil {
		t.Fatal(err)
	}
	version3[0] = 3
	if _, err := DecodeNormalClient(version3); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("version 3 = %v, want ErrUnsupported", err)
	}
}

// FuzzDecodeNormalClient asserts the decoder never panics on arbitrary input
// and is semantically stable: whenever a decoded record re-encodes, decoding
// that re-encoding yields an equal record.
//
// Byte-for-byte round-tripping deliberately does NOT hold. Two encodings decode
// to the same record — encode NUL-pads the service field while decode trims both
// NUL and space, and decode accepts a codePage that encode's exact-four-digits
// rule would reject — so a peer's bytes and our canonical bytes can differ
// while meaning the same thing. This mirrors upstream and is not a defect;
// stability of the meaning is the property that must hold.
func FuzzDecodeNormalClient(f *testing.F) {
	seed, _ := EncodeNormalClient(sampleRecord())
	f.Add(seed)
	f.Add([]byte{2, 3})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		rec, err := DecodeNormalClient(data)
		if err != nil {
			return
		}
		reencoded, err := EncodeNormalClient(rec)
		if err != nil {
			// Decode is more permissive than encode (e.g. a trimmed codePage
			// that is no longer four digits). That is allowed.
			return
		}
		roundTripped, err := DecodeNormalClient(reencoded)
		if err != nil {
			t.Fatalf("re-encoded record failed to decode: %v", err)
		}
		if roundTripped != rec {
			t.Fatalf("semantic round-trip mismatch:\n got %+v\nwant %+v", roundTripped, rec)
		}
	})
}
