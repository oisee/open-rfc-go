// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/protocol/appc.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go across appc.go,
// records.go, fragmentation.go, and conversation.go. Thrown RangeError/Error/
// TypeError became returned, wrapped sentinel errors; the two custom error
// classes became a *PeerReturnCodeError value and the ErrNormalDeallocation-
// WithoutData sentinel; the `intrinsicUint8ArrayByteLength`/`snapshotUint8Array`
// geometry intrinsics collapsed to len() and slice copies (the class of attack
// they defend is absent in Go — see docs/provenance.md); optional record-header
// fields (TypeScript `?`) became pointer fields so an unset field stays distinct
// from an explicit zero, preserving the `?? default` semantics. Enums became
// typed uint8 constants; `#private` state became unexported struct fields.
// See docs/provenance.md.

// Package appc encodes and decodes the version-6 APPC/CPI-C records that carry
// the classic RFC conversation over the gateway.
//
// A record is a fixed 48-byte common header, followed for controlled records by
// 32 bytes of operation information (the 80-byte record header), followed by
// application data. This file holds the shared constants, the function table,
// the fixed-field ASCII/hex helpers, and the common-header codec; the record
// encoders live in records.go, outgoing fragmentation in fragmentation.go, and
// the stateful conversation reassembly in conversation.go.
package appc

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/oisee/open-rfc-go/internal/wire"
)

const (
	// ProtocolVersion is the APPC version this package speaks (0x06).
	ProtocolVersion = 0x06
	// CommonHeaderLength is the fixed APPC common header size.
	CommonHeaderLength = 48
	// RecordHeaderLength is the common header plus the 32-byte operation
	// information carried by every controlled version-6 record.
	RecordHeaderLength = 80
	// ExtendedInitializeOptionsLength is the fixed size of the extended
	// initialization-options contract.
	ExtendedInitializeOptionsLength = 341
	// InitializeParametersLength is the fixed size of the F_INITIALIZE
	// parameter structure.
	InitializeParametersLength = 373
	// PartnerParametersLength is the fixed size of the F_SET_PARTNER_LU_NAME
	// parameter structure.
	PartnerParametersLength = 144
	// VectorEndOfMessage is the vector bit that marks a final data fragment.
	VectorEndOfMessage = 0x04
	// FinalSapParameterLength is the fixed SAP parameter tail on a compact
	// F_SAP_SEND record.
	FinalSapParameterLength = 8
	// MaxApplicationDataFragmentLength is the largest admitted STSEND/
	// F_ASEND_DATA application slice.
	MaxApplicationDataFragmentLength = 28_000
	// MaxAsyncSendsBeforeSync is how many async chunks the admitted streaming
	// contract sends before inserting a synchronous barrier.
	MaxAsyncSendsBeforeSync = 21
	// MaxOutgoingMessageLength is the protocol-wide signed INT4 ceiling.
	MaxOutgoingMessageLength = 0x7fff_ffff
	// DefaultMaxOutgoingMessageLength is the configured outgoing default.
	DefaultMaxOutgoingMessageLength = 1_400_000
	// DefaultMaxMessageLength bounds a reassembled incoming message.
	DefaultMaxMessageLength = 256 * 1024 * 1024
	// DefaultMaxMessageFragments bounds the fragment count of one message.
	DefaultMaxMessageFragments = 65_536

	// returnCodeDeallocatedNormal is SAP Note 63347: the peer ended the
	// CPI-C conversation normally.
	returnCodeDeallocatedNormal = 18
)

// MaxDataFragmentLength is a backwards-compatible name for the evidenced
// 28,000-byte application slice.
const MaxDataFragmentLength = MaxApplicationDataFragmentLength

// CpicStreamingPolicy records whether the caller has independently approved
// streaming to the target; the observed wire carries no peer-acceptance bit.
type CpicStreamingPolicy string

const (
	// StreamingDisabled forbids messages beyond one compact record.
	StreamingDisabled CpicStreamingPolicy = "disabled"
	// StreamingEnabled permits bounded F_ASEND_DATA streaming.
	StreamingEnabled CpicStreamingPolicy = "enabled"
)

// Function is an APPC/CPI-C function code.
type Function uint8

const (
	FuncInitialize       Function = 0x01
	FuncAllocate         Function = 0x05
	FuncSendData         Function = 0x07
	FuncAsyncSendData    Function = 0x08
	FuncReceive          Function = 0x09
	FuncAsyncReceive     Function = 0x0a
	FuncDeallocate       Function = 0x0b
	FuncSetTpName        Function = 0x0d
	FuncSetPartnerLuName Function = 0x0f
	FuncFlush            Function = 0x1b
	FuncSapSend          Function = 0xcb
)

var (
	// ErrRange reports a value or byte length outside its permitted interval.
	ErrRange = errors.New("appc: value out of range")
	// ErrProtocol reports a structurally invalid or non-canonical record.
	ErrProtocol = errors.New("appc: protocol violation")
	// ErrUnsupported reports a record variant this package does not implement.
	ErrUnsupported = errors.New("appc: unsupported record variant")
	// ErrState reports an illegal setup/teardown transition.
	ErrState = errors.New("appc: illegal state transition")
	// ErrNormalDeallocationWithoutData reports CM_DEALLOCATED_NORMAL returned
	// without a decodable data payload (CM_NO_DATA_RECEIVED).
	ErrNormalDeallocationWithoutData = errors.New("appc: connection closed without message (CM_NO_DATA_RECEIVED)")
)

// PeerReturnCodeError is a non-success CPI-C/APPC status returned by the peer.
type PeerReturnCodeError struct {
	FunctionName   string
	AppcReturnCode uint32
	SapReturnCode  uint32
}

func (e *PeerReturnCodeError) Error() string {
	return fmt.Sprintf("%s failed with APPC return code %d and SAP return code %d",
		e.FunctionName, e.AppcReturnCode, e.SapReturnCode)
}

// PayloadInfo names the version and function a payload begins with.
type PayloadInfo struct {
	ProtocolVersion uint8
	FunctionCode    Function
	FunctionName    string
}

// Header is the decoded fixed 48-byte APPC common header.
type Header struct {
	PayloadInfo
	Protocol        uint8
	Mode            uint8
	UID             uint16
	GatewayID       uint16
	ErrorLength     uint16
	Info2           uint8
	TraceLevel      uint8
	Time            uint32
	Info3           uint8
	Timeout         int32
	Info4           uint8
	SequenceNumber  uint32
	SapParameterLen uint16
	Padding         uint16
	Info            uint8
	Vector          uint8
	AppcReturnCode  uint32
	SapReturnCode   uint32
	ConversationID  []byte
}

var functionNames = map[Function]string{
	FuncInitialize:       "F_INITIALIZE",
	FuncAllocate:         "F_ALLOCATE",
	FuncSendData:         "F_SEND_DATA",
	FuncAsyncSendData:    "F_ASEND_DATA",
	FuncReceive:          "F_RECEIVE",
	FuncAsyncReceive:     "F_ARECEIVE",
	FuncDeallocate:       "F_DEALLOCATE",
	FuncSetTpName:        "F_SET_TP_NAME",
	FuncSetPartnerLuName: "F_SET_PARTNER_LU_NAME",
	FuncFlush:            "F_FLUSH",
	FuncSapSend:          "F_SAP_SEND",
}

var controlFunctions = map[Function]bool{
	FuncInitialize:       true,
	FuncAllocate:         true,
	FuncDeallocate:       true,
	FuncSetTpName:        true,
	FuncSetPartnerLuName: true,
	FuncFlush:            true,
}

// FunctionName returns the mnemonic for a function code, or UNKNOWN_0xNN.
func FunctionName(code Function) string {
	if name, ok := functionNames[code]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN_0x%02x", uint8(code))
}

// InspectPayload reads the version and function from the front of a payload.
func InspectPayload(payload []byte) (PayloadInfo, error) {
	if len(payload) < 2 {
		return PayloadInfo{}, fmt.Errorf("%w: an APPC payload needs at least a version and function byte", ErrRange)
	}
	protocolVersion := payload[0]
	functionCode := Function(payload[1])
	if protocolVersion != ProtocolVersion {
		return PayloadInfo{}, fmt.Errorf("%w: unsupported APPC protocol version 0x%x", ErrUnsupported, protocolVersion)
	}
	return PayloadInfo{
		ProtocolVersion: protocolVersion,
		FunctionCode:    functionCode,
		FunctionName:    FunctionName(functionCode),
	}, nil
}

var printableASCII = regexp.MustCompile(`^[\x20-\x7e]*$`)
var hexIDPattern = regexp.MustCompile(`^[0-9A-F]{16}$`)

// encodeFixedAscii encodes an 8-byte fixed field. An absent name (empty) is
// all-NUL; a present short name is ASCII plus spaces — a wire distinction the
// captures preserve.
func encodeFixedAscii(value, field string) ([]byte, error) {
	if !printableASCII.MatchString(value) || len(value) > 8 {
		return nil, fmt.Errorf("%w: %s must contain at most 8 ASCII bytes", ErrRange, field)
	}
	pad := byte(0x20)
	if len(value) == 0 {
		pad = 0
	}
	out := bytes.Repeat([]byte{pad}, 8)
	copy(out, value)
	return out, nil
}

func decodeFixedAscii(value []byte, field string) (string, error) {
	for _, b := range value {
		if b != 0 && (b < 0x20 || b > 0x7e) {
			return "", fmt.Errorf("%w: %s contains a non-ASCII byte", ErrProtocol, field)
		}
	}
	return trimRightBytes(string(value), "\x00 "), nil
}

func encodePaddedAscii(value string, width int, padding byte, field string) ([]byte, error) {
	if !printableASCII.MatchString(value) || len(value) > width {
		return nil, fmt.Errorf("%w: %s must contain at most %d ASCII bytes", ErrRange, field, width)
	}
	out := bytes.Repeat([]byte{padding}, width)
	copy(out, value)
	return out, nil
}

// decodePaddedAscii treats the first padding byte as a terminator and rejects
// any non-padding data after it.
func decodePaddedAscii(encoded []byte, padding byte, field string) (string, error) {
	end := bytes.IndexByte(encoded, padding)
	if end < 0 {
		end = len(encoded)
	}
	for i := 0; i < end; i++ {
		if encoded[i] < 0x20 || encoded[i] > 0x7e {
			return "", fmt.Errorf("%w: %s contains a non-ASCII byte", ErrProtocol, field)
		}
	}
	for i := end; i < len(encoded); i++ {
		if encoded[i] != padding {
			return "", fmt.Errorf("%w: %s contains data after its first padding byte", ErrProtocol, field)
		}
	}
	return string(encoded[:end]), nil
}

func exactBytes(value []byte, length int, field string) ([]byte, error) {
	if len(value) != length {
		return nil, fmt.Errorf("%w: %s must contain exactly %d bytes; received %d", ErrRange, field, length, len(value))
	}
	return append([]byte(nil), value...), nil
}

func fixedHexID(value, field string) ([]byte, error) {
	if !hexIDPattern.MatchString(value) {
		return nil, fmt.Errorf("%w: %s must contain exactly 16 uppercase hexadecimal characters", ErrRange, field)
	}
	return []byte(value), nil
}

func trimRightBytes(s, cutset string) string {
	end := len(s)
	for end > 0 && bytes.IndexByte([]byte(cutset), s[end-1]) >= 0 {
		end--
	}
	return s[:end]
}

// DecodeHeader decodes the fixed 48-byte APPC common header shared by the
// observed version-6 records.
func DecodeHeader(payload []byte) (Header, error) {
	if len(payload) < CommonHeaderLength {
		return Header{}, fmt.Errorf("%w: an APPC common header needs %d bytes; received %d", ErrRange, CommonHeaderLength, len(payload))
	}
	r := wire.NewReader(payload[:CommonHeaderLength], "APPC common header")

	var h Header
	var err error
	take := func(f func() error) {
		if err == nil {
			err = f()
		}
	}

	var protocolVersion, functionCode uint8
	take(func() error { var e error; protocolVersion, e = r.ReadUint8("protocolVersion"); return e })
	take(func() error { var e error; functionCode, e = r.ReadUint8("functionCode"); return e })
	take(func() error {
		info, e := InspectPayload([]byte{protocolVersion, functionCode})
		if e == nil {
			h.PayloadInfo = info
		}
		return e
	})
	take(func() error { var e error; h.Protocol, e = r.ReadUint8("protocol"); return e })
	take(func() error { var e error; h.Mode, e = r.ReadUint8("mode"); return e })
	take(func() error { var e error; h.UID, e = r.ReadUint16BE("uid"); return e })
	take(func() error { var e error; h.GatewayID, e = r.ReadUint16BE("gatewayId"); return e })
	take(func() error { var e error; h.ErrorLength, e = r.ReadUint16BE("errorLength"); return e })
	take(func() error { var e error; h.Info2, e = r.ReadUint8("info2"); return e })
	take(func() error { var e error; h.TraceLevel, e = r.ReadUint8("traceLevel"); return e })
	take(func() error { var e error; h.Time, e = r.ReadUint32BE("time"); return e })
	take(func() error { var e error; h.Info3, e = r.ReadUint8("info3"); return e })
	take(func() error { var e error; h.Timeout, e = r.ReadInt32BE("timeout"); return e })
	take(func() error { var e error; h.Info4, e = r.ReadUint8("info4"); return e })
	take(func() error { var e error; h.SequenceNumber, e = r.ReadUint32BE("sequenceNumber"); return e })
	take(func() error { var e error; h.SapParameterLen, e = r.ReadUint16BE("sapParameterLength"); return e })
	take(func() error { var e error; h.Padding, e = r.ReadUint16BE("padding"); return e })
	take(func() error { var e error; h.Info, e = r.ReadUint8("info"); return e })
	take(func() error { var e error; h.Vector, e = r.ReadUint8("vector"); return e })
	take(func() error { var e error; h.AppcReturnCode, e = r.ReadUint32BE("appcReturnCode"); return e })
	take(func() error { var e error; h.SapReturnCode, e = r.ReadUint32BE("sapReturnCode"); return e })
	take(func() error { var e error; h.ConversationID, e = r.ReadBytes(8, "conversationId"); return e })
	take(func() error { return r.Finish() })

	if err != nil {
		return Header{}, err
	}
	return h, nil
}
