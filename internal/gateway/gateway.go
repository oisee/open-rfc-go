// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/protocol/gateway.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown RangeError/Error
// became returned, wrapped sentinel errors; node:net isIPv4 became
// net.ParseIP with an explicit rejection of colon forms (so IPv4-mapped IPv6
// is refused, as upstream's isIPv4 refuses it); the GatewayAcceptInfo enum
// became typed uint8 flag constants. See docs/provenance.md.

// Package gateway encodes and decodes the version-2 GW_NORMAL_CLIENT record
// exchanged with the SAP gateway before the APPC conversation begins.
//
// The record is a fixed 64-byte structure whose first byte is the protocol
// version. Only version 2 (IPv4) is implemented; the exact length is part of
// that versioned definition, not an observation about one peer, so it is
// checked exactly.
package gateway

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/oisee/open-rfc-go/internal/wire"
)

const (
	// NormalClientLength is the fixed byte length of the version-2 record.
	NormalClientLength = 64
	// ProtocolVersion is the only gateway protocol version implemented.
	ProtocolVersion = 2
	// NormalClientRequest is the request type for a GW_NORMAL_CLIENT record.
	NormalClientRequest = 3
)

// AcceptInfo bits advertise which optional gateway features a peer supports.
// They are OR-ed together into the record's AcceptInfo byte.
const (
	AcceptInfoErrorInfo           uint8 = 0x01
	AcceptInfoPing                uint8 = 0x02
	AcceptInfoSnc                 uint8 = 0x04
	AcceptInfoConnectionExtended  uint8 = 0x08
	AcceptInfoCodePage            uint8 = 0x10
	AcceptInfoNiPing              uint8 = 0x20
	AcceptInfoExtendedInitOptions uint8 = 0x40
	AcceptInfoDistributedTrace    uint8 = 0x80
)

var (
	// ErrRange reports a field value outside its permitted interval or shape.
	ErrRange = errors.New("gateway: value out of range")
	// ErrUnsupported reports a record variant this package does not implement.
	ErrUnsupported = errors.New("gateway: unsupported record variant")
	// ErrMalformed reports a structurally invalid record (bad length or a
	// reserved field that is not zero).
	ErrMalformed = errors.New("gateway: malformed record")
)

var codePagePattern = regexp.MustCompile(`^\d{4}$`)

// NormalClientRecord is the decoded GW_NORMAL_CLIENT record.
//
// GatewayOptionLevel is the last byte of an otherwise-zero six-byte option
// region. Client value 6 and server value 15 are observed; it is kept explicit
// until a second implementation establishes individual option-bit semantics.
type NormalClientRecord struct {
	Address            string
	Service            string
	CodePage           string
	GatewayOptionLevel uint8
	LogicalUnit        string
	TransactionProgram string
	ConversationID     string
	AppcHeaderVersion  uint8
	AcceptInfo         uint8
	Index              int
	ReturnCode         uint32
	EchoData           uint8
}

func encodeASCII(value string, length int, field string, padding byte) ([]byte, error) {
	if len(value) > length || !isPrintableASCII(value) {
		return nil, fmt.Errorf("%w: %s must contain at most %d ASCII bytes", ErrRange, field, length)
	}
	out := bytes.Repeat([]byte{padding}, length)
	copy(out, value)
	return out, nil
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

func decodeASCII(data []byte, field string) (string, error) {
	for _, b := range data {
		if b != 0 && (b < 0x20 || b > 0x7e) {
			return "", fmt.Errorf("%w: %s contains a non-ASCII byte", ErrMalformed, field)
		}
	}
	return strings.TrimRight(string(data), "\x00 "), nil
}

func encodeIPv4(address string) ([]byte, error) {
	if strings.Contains(address, ":") {
		return nil, fmt.Errorf("%w: address must be an IPv4 address for gateway protocol version 2", ErrRange)
	}
	ip := net.ParseIP(address)
	if ip == nil || ip.To4() == nil {
		return nil, fmt.Errorf("%w: address must be an IPv4 address for gateway protocol version 2", ErrRange)
	}
	return append([]byte(nil), ip.To4()...), nil
}

func decodeIPv4(data []byte) string {
	return fmt.Sprintf("%d.%d.%d.%d", data[0], data[1], data[2], data[3])
}

// EncodeNormalClient encodes the version-2 GW_NORMAL_CLIENT record.
func EncodeNormalClient(record NormalClientRecord) ([]byte, error) {
	if record.Index < -0x8000 || record.Index > 0x7fff {
		return nil, fmt.Errorf("%w: index must be a signed 16-bit integer", ErrRange)
	}
	if !codePagePattern.MatchString(record.CodePage) {
		return nil, fmt.Errorf("%w: codePage must contain exactly four ASCII digits", ErrRange)
	}

	address, err := encodeIPv4(record.Address)
	if err != nil {
		return nil, err
	}
	service, err := encodeASCII(record.Service, 10, "service", 0)
	if err != nil {
		return nil, err
	}
	logicalUnit, err := encodeASCII(record.LogicalUnit, 8, "logicalUnit", 0x20)
	if err != nil {
		return nil, err
	}
	transactionProgram, err := encodeASCII(record.TransactionProgram, 8, "transactionProgram", 0x20)
	if err != nil {
		return nil, err
	}
	conversationID, err := encodeASCII(record.ConversationID, 8, "conversationId", 0x20)
	if err != nil {
		return nil, err
	}

	w, err := wire.NewWriter(NormalClientLength, "gateway normal-client record")
	if err != nil {
		return nil, err
	}
	writes := []func() error{
		func() error { return w.WriteUint8(ProtocolVersion, "version") },
		func() error { return w.WriteUint8(NormalClientRequest, "requestType") },
		func() error { return w.WriteBytes(address, "address") },
		func() error { return w.WriteUint32BE(0, "reserved1") },
		func() error { return w.WriteBytes(service, "service") },
		func() error { return w.WriteBytes([]byte(record.CodePage), "codePage") },
		func() error { return w.WriteBytes(make([]byte, 5), "reserved2") },
		func() error { return w.WriteUint8(record.GatewayOptionLevel, "gatewayOptionLevel") },
		func() error { return w.WriteBytes(logicalUnit, "logicalUnit") },
		func() error { return w.WriteBytes(transactionProgram, "transactionProgram") },
		func() error { return w.WriteBytes(conversationID, "conversationId") },
		func() error { return w.WriteUint8(record.AppcHeaderVersion, "appcHeaderVersion") },
		func() error { return w.WriteUint8(record.AcceptInfo, "acceptInfo") },
		func() error { return w.WriteUint16BE(uint16(record.Index&0xffff), "index") },
		func() error { return w.WriteUint32BE(record.ReturnCode, "returnCode") },
		func() error { return w.WriteUint8(record.EchoData, "echoData") },
		func() error { return w.WriteUint8(0, "filler") },
	}
	for _, write := range writes {
		if err := write(); err != nil {
			return nil, err
		}
	}
	return w.Finish()
}

// DecodeNormalClient decodes a version-2 GW_NORMAL_CLIENT request or response.
func DecodeNormalClient(data []byte) (NormalClientRecord, error) {
	var zero NormalClientRecord
	if len(data) < 2 {
		return zero, fmt.Errorf("%w: gateway normal-client record needs at least 2 bytes", ErrMalformed)
	}
	switch version := data[0]; {
	case version == 3:
		return zero, fmt.Errorf("%w: gateway protocol version 3 IPv6 records are not implemented", ErrUnsupported)
	case version != ProtocolVersion:
		return zero, fmt.Errorf("%w: unsupported gateway protocol version %d", ErrUnsupported, version)
	}
	if len(data) != NormalClientLength {
		return zero, fmt.Errorf("%w: gateway version-2 normal-client record needs %d bytes; received %d", ErrMalformed, NormalClientLength, len(data))
	}

	r := wire.NewReader(data, "gateway normal-client record")
	var record NormalClientRecord
	var err error
	read := func(f func() error) {
		if err == nil {
			err = f()
		}
	}

	read(func() error { _, e := r.ReadUint8("version"); return e })
	var requestType uint8
	read(func() error { var e error; requestType, e = r.ReadUint8("requestType"); return e })
	read(func() error {
		if err == nil && requestType != NormalClientRequest {
			return fmt.Errorf("%w: expected GW_NORMAL_CLIENT request type 3; received %d", ErrMalformed, requestType)
		}
		return nil
	})
	read(func() error {
		addr, e := r.ReadBytes(4, "address")
		if e == nil {
			record.Address = decodeIPv4(addr)
		}
		return e
	})
	read(func() error {
		v, e := r.ReadUint32BE("reserved1")
		if e == nil && v != 0 {
			return fmt.Errorf("%w: gateway normal-client reserved1 field must be zero", ErrMalformed)
		}
		return e
	})
	read(func() error { var e error; record.Service, e = readASCII(r, 10, "service"); return e })
	read(func() error { var e error; record.CodePage, e = readASCII(r, 4, "codePage"); return e })
	read(func() error {
		v, e := r.ReadBytes(5, "reserved2")
		if e == nil && !bytes.Equal(v, make([]byte, 5)) {
			return fmt.Errorf("%w: gateway normal-client reserved2 field must be zero", ErrMalformed)
		}
		return e
	})
	read(func() error { var e error; record.GatewayOptionLevel, e = r.ReadUint8("gatewayOptionLevel"); return e })
	read(func() error { var e error; record.LogicalUnit, e = readASCII(r, 8, "logicalUnit"); return e })
	read(func() error {
		var e error
		record.TransactionProgram, e = readASCII(r, 8, "transactionProgram")
		return e
	})
	read(func() error { var e error; record.ConversationID, e = readASCII(r, 8, "conversationId"); return e })
	read(func() error { var e error; record.AppcHeaderVersion, e = r.ReadUint8("appcHeaderVersion"); return e })
	read(func() error { var e error; record.AcceptInfo, e = r.ReadUint8("acceptInfo"); return e })
	read(func() error {
		v, e := r.ReadUint16BE("index")
		if e == nil {
			if v > 0x7fff {
				record.Index = int(v) - 0x1_0000
			} else {
				record.Index = int(v)
			}
		}
		return e
	})
	read(func() error { var e error; record.ReturnCode, e = r.ReadUint32BE("returnCode"); return e })
	read(func() error { var e error; record.EchoData, e = r.ReadUint8("echoData"); return e })
	read(func() error {
		v, e := r.ReadUint8("filler")
		if e == nil && v != 0 {
			return fmt.Errorf("%w: gateway normal-client filler field must be zero", ErrMalformed)
		}
		return e
	})
	read(func() error { return r.Finish() })

	if err != nil {
		return zero, err
	}
	return record, nil
}

func readASCII(r *wire.Reader, length int, field string) (string, error) {
	raw, err := r.ReadBytes(length, field)
	if err != nil {
		return "", err
	}
	return decodeASCII(raw, field)
}
