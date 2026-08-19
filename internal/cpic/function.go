// SPDX-License-Identifier: Apache-2.0
//
// Part of the cpic package; see cpic.go for the file's provenance header.

package cpic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/oisee/open-rfc-go/internal/appc"
	"github.com/oisee/open-rfc-go/internal/rfcerr"
)

var (
	cpicFunctionRequestPrefix    = mustHex("010100080301010504010003")
	cpicFunctionResponsePrefix   = mustHex("05000000")
	cpicCutFunctionRequestPrefix = mustHex("05020000")
)

// RequestAppcFraming describes how a CPIC request is framed for APPC.
type RequestAppcFraming struct {
	Mode                    string // "compact" or "streamed"
	ApplicationDataLength   int
	FinalSapParameterLength int // 0 or 8
}

// FunctionRequestInput is the input to the first regular Unicode RFC call.
type FunctionRequestInput struct {
	FunctionName         string
	SessionID            []byte
	KernelRelease        string
	MaximumRfcPacketSize *uint32
}

// NamedValue is an import parameter or xRFC parameter (name + bytes).
type NamedValue struct {
	Name  string
	Value []byte
}

// Table is a CUT table input.
type Table struct {
	Name          string
	RowByteLength int
	Rows          [][]byte
}

// CutFunctionRequestInput is the input to an established-session CUT request.
type CutFunctionRequestInput struct {
	FunctionName         string
	RequestedOutputs     []string
	Imports              []NamedValue
	Tables               []Table
	XrfcParameters       []NamedValue
	KernelRelease        string
	MaximumRfcPacketSize *uint32
}

// DecodedFunctionResponse is the structural outcome of a regular RFC response.
type DecodedFunctionResponse struct {
	Success      bool
	Outcome      rfcerr.Outcome
	Status       *int
	ExceptionKey string
	HasException bool
	Fields       []TagLen
}

// DecodedFunctionResultFields carries the decoded application fields.
type DecodedFunctionResultFields struct {
	Success  bool
	Status   *int
	Envelope rfcerr.Envelope
	Fields   []Field
}

func checkedMaximumRfcPacketSize(v *uint32) uint32 {
	if v != nil {
		return *v
	}
	return 0x8500
}

func packetTrailer(packetPrefixAndChainLength int, maximumRfcPacketSize uint32) []byte {
	cpicPacketSize := packetPrefixAndChainLength + 2
	if cpicPacketSize > appc.MaxApplicationDataFragmentLength {
		// Streamed mode: the logical CUT message closes with the field chain's
		// empty End field followed only by the 0xffff packet sentinel.
		return []byte{0xff, 0xff}
	}
	trailer := make([]byte, 10)
	binary.BigEndian.PutUint16(trailer[0:], uint16(TagEnd))
	binary.BigEndian.PutUint16(trailer[2:], 0)
	binary.BigEndian.PutUint16(trailer[4:], uint16(cpicPacketSize))
	binary.BigEndian.PutUint32(trailer[6:], maximumRfcPacketSize)
	return trailer
}

// InspectRequestAppcFraming inspects the bounded CPIC request trailer before
// APPC framing.
func InspectRequestAppcFraming(data []byte) (RequestAppcFraming, error) {
	var zero RequestAppcFraming
	if len(data) >= 10 &&
		binary.BigEndian.Uint16(data[len(data)-10:]) == uint16(TagEnd) &&
		binary.BigEndian.Uint16(data[len(data)-8:]) == 0 &&
		int(binary.BigEndian.Uint16(data[len(data)-6:])) == len(data)-8 &&
		len(data)-8 <= appc.MaxApplicationDataFragmentLength {
		return RequestAppcFraming{Mode: "compact", ApplicationDataLength: len(data) - 8, FinalSapParameterLength: 8}, nil
	}
	if len(data) > appc.MaxApplicationDataFragmentLength && len(data) >= 6 &&
		binary.BigEndian.Uint16(data[len(data)-6:]) == uint16(TagEnd) &&
		binary.BigEndian.Uint16(data[len(data)-4:]) == 0 &&
		binary.BigEndian.Uint16(data[len(data)-2:]) == 0xffff {
		return RequestAppcFraming{Mode: "streamed", ApplicationDataLength: len(data), FinalSapParameterLength: 0}, nil
	}
	return zero, fmt.Errorf("%w: CPIC request has an invalid APPC framing trailer", ErrProtocol)
}

// EncodeFunctionRequest encodes the first regular Unicode RFC call after logon.
func EncodeFunctionRequest(input FunctionRequestInput) ([]byte, error) {
	kernelRelease := input.KernelRelease
	if kernelRelease == "" {
		kernelRelease = "754"
	}
	if !digits3.MatchString(kernelRelease) {
		return nil, fmt.Errorf("%w: kernelRelease must contain exactly three ASCII digits", ErrRange)
	}
	maximumRfcPacketSize := checkedMaximumRfcPacketSize(input.MaximumRfcPacketSize)
	sessionID, err := exactBytes(input.SessionID, 16, "sessionId")
	if err != nil {
		return nil, err
	}
	functionName, err := unicodeBytes(input.FunctionName, "functionName", 40)
	if err != nil {
		return nil, err
	}
	fields := []Field{
		{Tag: uint16(TagProtocolVersion), Value: initialProtocolVersion},
		{Tag: uint16(TagCapabilities), Value: initialCapabilities},
		{Tag: uint16(TagLogonMarker), Value: nil},
		{Tag: uint16(TagSession), Value: sessionID},
		{Tag: uint16(TagContextEnd), Value: nil},
		{Tag: uint16(TagKernel), Value: utf16leBytes(kernelRelease)},
		{Tag: uint16(TagFunction), Value: functionName},
		{Tag: uint16(TagCallContext), Value: nil},
		{Tag: uint16(TagEnd), Value: nil},
	}
	chainByteLength, err := FieldChainByteLength(uint16(TagStart), fields, FieldChainLimits{})
	if err != nil {
		return nil, err
	}
	trailer := packetTrailer(len(cpicFunctionRequestPrefix)+chainByteLength, maximumRfcPacketSize)
	chain, err := EncodeFieldChain(uint16(TagStart), fields, FieldChainLimits{})
	if err != nil {
		return nil, err
	}
	return concat(cpicFunctionRequestPrefix, chain, trailer), nil
}

func rejectDuplicates(values []string, kind string) error {
	seen := map[string]bool{}
	for _, v := range values {
		if seen[v] {
			return fmt.Errorf("%w: duplicate %s %s", ErrProtocol, kind, v)
		}
		seen[v] = true
	}
	return nil
}

// EncodeCutFunctionRequest encodes an established-session classic Unicode (CUT)
// RFC request.
func EncodeCutFunctionRequest(input CutFunctionRequestInput) ([]byte, error) {
	kernelRelease := input.KernelRelease
	if kernelRelease == "" {
		kernelRelease = "754"
	}
	if !digits3.MatchString(kernelRelease) {
		return nil, fmt.Errorf("%w: kernelRelease must contain exactly three ASCII digits", ErrRange)
	}
	maximumRfcPacketSize := checkedMaximumRfcPacketSize(input.MaximumRfcPacketSize)

	names := func(vs []NamedValue) []string {
		out := make([]string, len(vs))
		for i, v := range vs {
			out[i] = v.Name
		}
		return out
	}
	if err := rejectDuplicates(input.RequestedOutputs, "requested output"); err != nil {
		return nil, err
	}
	if err := rejectDuplicates(names(input.Imports), "import"); err != nil {
		return nil, err
	}
	tableNames := make([]string, len(input.Tables))
	for i, t := range input.Tables {
		tableNames[i] = t.Name
	}
	if err := rejectDuplicates(tableNames, "table"); err != nil {
		return nil, err
	}
	if err := rejectDuplicates(names(input.XrfcParameters), "xRFC parameter"); err != nil {
		return nil, err
	}
	inputNames := map[string]bool{}
	addNames := func(ns []string) error {
		for _, n := range ns {
			if inputNames[n] {
				return fmt.Errorf("%w: duplicate input parameter %s", ErrProtocol, n)
			}
			inputNames[n] = true
		}
		return nil
	}
	if err := addNames(names(input.Imports)); err != nil {
		return nil, err
	}
	if err := addNames(tableNames); err != nil {
		return nil, err
	}
	if err := addNames(names(input.XrfcParameters)); err != nil {
		return nil, err
	}

	functionName, err := unicodeBytes(input.FunctionName, "functionName", 40)
	if err != nil {
		return nil, err
	}
	fields := []Field{
		{Tag: uint16(TagKernel), Value: utf16leBytes(kernelRelease)},
		{Tag: uint16(TagFunction), Value: functionName},
		{Tag: uint16(TagCallContext), Value: nil},
	}
	for _, name := range input.RequestedOutputs {
		v, err := unicodeBytes(name, "requested output name", 30)
		if err != nil {
			return nil, err
		}
		fields = append(fields, Field{Tag: uint16(TagRequestedOutput), Value: v})
	}
	for _, parameter := range input.Imports {
		v, err := unicodeBytes(parameter.Name, "import name", 30)
		if err != nil {
			return nil, err
		}
		fields = append(fields,
			Field{Tag: uint16(TagParameterName), Value: v},
			Field{Tag: uint16(TagParameterValue), Value: parameter.Value},
		)
	}
	for _, table := range input.Tables {
		if table.RowByteLength < 0 || table.RowByteLength > 0xffff_ffff {
			return nil, fmt.Errorf("%w: %s rowByteLength must be an unsigned 32-bit integer", ErrRange, table.Name)
		}
		name, err := unicodeBytes(table.Name, "table name", 30)
		if err != nil {
			return nil, err
		}
		header := make([]byte, 8)
		binary.BigEndian.PutUint32(header[0:], uint32(table.RowByteLength))
		binary.BigEndian.PutUint32(header[4:], uint32(len(table.Rows)))
		fields = append(fields,
			Field{Tag: uint16(TagTableName), Value: name},
			Field{Tag: uint16(TagTableHeader), Value: header},
		)
		for index, row := range table.Rows {
			if len(row) != table.RowByteLength {
				return nil, fmt.Errorf("%w: %s row %d contains %d bytes; expected %d", ErrRange, table.Name, index, len(row), table.RowByteLength)
			}
			fields = append(fields, Field{Tag: uint16(TagTableCompr), Value: row})
		}
	}
	for _, parameter := range input.XrfcParameters {
		if _, err := unicodeBytes(parameter.Name, "xRFC parameter name", 30); err != nil {
			return nil, err
		}
		if len(parameter.Value) == 0 {
			return nil, fmt.Errorf("%w: %s xRFC XML value must not be empty", ErrRange, parameter.Name)
		}
		fields = append(fields, Field{Tag: uint16(TagXRfcParameter), Value: nil})
		for offset := 0; offset < len(parameter.Value); offset += ClassicXrfcXMLChunkLength {
			end := offset + ClassicXrfcXMLChunkLength
			if end > len(parameter.Value) {
				end = len(parameter.Value)
			}
			fields = append(fields, Field{Tag: uint16(TagXRfcData), Value: append([]byte(nil), parameter.Value[offset:end]...)})
		}
		fields = append(fields, Field{Tag: uint16(TagXRfcParameter), Value: nil})
	}
	fields = append(fields, Field{Tag: uint16(TagEnd), Value: nil})

	chainByteLength, err := FieldChainByteLength(uint16(TagContextEnd), fields, FieldChainLimits{})
	if err != nil {
		return nil, err
	}
	trailer := packetTrailer(len(cpicCutFunctionRequestPrefix)+chainByteLength, maximumRfcPacketSize)
	chain, err := EncodeFieldChain(uint16(TagContextEnd), fields, FieldChainLimits{})
	if err != nil {
		return nil, err
	}
	return concat(cpicCutFunctionRequestPrefix, chain, trailer), nil
}

func decodeFunctionResponseEnvelope(data []byte, additionalAllowedTags []uint16) (DecodedFunctionResultFields, error) {
	var zero DecodedFunctionResultFields
	if len(data) < len(cpicFunctionResponsePrefix)+8 {
		return zero, fmt.Errorf("%w: CPIC function response is truncated", ErrRange)
	}
	if !bytes.Equal(data[0:len(cpicFunctionResponsePrefix)], cpicFunctionResponsePrefix) {
		return zero, fmt.Errorf("%w: CPIC function response prefix is invalid", ErrProtocol)
	}
	decoded, err := DecodeFieldChainPrefix(data[len(cpicFunctionResponsePrefix):], uint16(TagResponseStart), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		return zero, err
	}
	trailerOffset := len(cpicFunctionResponsePrefix) + decoded.BytesConsumed
	if len(data)-trailerOffset != 2 {
		return zero, fmt.Errorf("%w: CPIC function response trailer is invalid", ErrProtocol)
	}
	if binary.BigEndian.Uint16(data[trailerOffset:]) != uint16(TagEnd) {
		return zero, fmt.Errorf("%w: CPIC function response trailer is invalid", ErrProtocol)
	}
	return decodeFunctionResponseFields(decoded.Fields, additionalAllowedTags)
}

func decodeFunctionResponseFields(fields []Field, additionalAllowedTags []uint16) (DecodedFunctionResultFields, error) {
	var zero DecodedFunctionResultFields
	allowed := append([]uint16{uint16(TagProgram), 0x0667}, additionalAllowedTags...)
	envelope, err := rfcerr.Decode(toRfcErrFields(fields), rfcerr.DecodeOptions{
		MaxFieldCount:         intptr(DefaultMaxFieldCount),
		AdditionalAllowedTags: allowed,
	})
	if err != nil {
		return zero, err
	}
	var status *int
	if envelope.Outcome == rfcerr.OutcomeSuccess && len(envelope.Facts.Unresolved0420) == 1 && envelope.Facts.Unresolved0420[0].ByteLength == 4 {
		n, perr := strconv.ParseUint(envelope.Facts.Unresolved0420[0].ValueHex, 16, 32)
		if perr == nil {
			s := int(n)
			status = &s
		}
	}
	return DecodedFunctionResultFields{
		Success:  envelope.Outcome == rfcerr.OutcomeSuccess,
		Status:   status,
		Envelope: envelope,
		Fields:   fields,
	}, nil
}

// DecodeFunctionResultFields decodes application fields for the value-codec
// layer. Keep these values out of logs.
func DecodeFunctionResultFields(data []byte) (DecodedFunctionResultFields, error) {
	// s4ClassicExtensionTags are S/4HANA classic-serialization extension tags
	// observed live and beyond upstream open-rfc's scope (classic non-S4): 0x0104
	// accompanies scalar/structure exports, and the 0x033x family carries S4 table
	// data. Tolerated so an S/4 classic response classifies instead of being
	// rejected as an unknown tag.
	var s4ClassicExtensionTags = []uint16{
		uint16(TagXRfcParameter), uint16(TagXRfcData),
		0x0104, 0x0331, 0x0333, 0x0334, 0x0335, 0x0336,
	}

	// (legacy note) 0x0104 and 0x0331 are S/4HANA classic-serialization extension tags observed
	// live (RFC_SYSTEM_INFO carries 0x0104; STFC_STRUCTURE carries 0x0331). They
	// are beyond upstream open-rfc's scope (classic non-S4) — an open-rfc-go
	// extension — and are tolerated here so an S/4 classic response's envelope
	// classifies instead of being rejected as an unknown tag.
	return decodeFunctionResponseEnvelope(data, s4ClassicExtensionTags)
}

// DecodeResetServerContextResultFields decodes the reply to
// SYSTEM_RESET_RFC_SERVER.
func DecodeResetServerContextResultFields(data []byte) (DecodedFunctionResultFields, error) {
	decoded, err := decodeFunctionResponseEnvelope(data, []uint16{uint16(TagRfcServerResetDn)})
	if err != nil {
		return DecodedFunctionResultFields{}, err
	}
	resetDone := filterTag(decoded.Fields, uint16(TagRfcServerResetDn))
	if len(resetDone) > 1 || (len(resetDone) == 1 && len(resetDone[0].Value) != 0) {
		return DecodedFunctionResultFields{}, fmt.Errorf("%w: SYSTEM_RESET_RFC_SERVER response reset-done control must be empty and unique", ErrProtocol)
	}
	return decoded, nil
}

var sessionRefreshPreambleTags = map[uint16]bool{
	uint16(TagProtocolVersion): true, uint16(TagCapabilities): true, uint16(TagLogonStatus): true,
	uint16(TagSystemCodePage): true, uint16(TagClientAddress): true, uint16(TagPartnerSystem): true,
	uint16(TagPartnerHost): true, uint16(TagConnectionType): true, uint16(TagKernelRelease): true,
	uint16(TagKernelPatch): true, uint16(TagDestination): true, uint16(TagProgram): true,
	0x0020: true, 0x0021: true, initialCpicUnresolved0450: true, 0x0451: true, 0x0452: true, 0x0453: true,
}

const (
	maxSessionRefreshPreambleFields = 32
	maxSessionRefreshPreambleBytes  = 16 * 1024
)

// DecodeSessionRefreshResultFields decodes the first successful call after
// SYSTEM_RESET_RFC_SERVER.
func DecodeSessionRefreshResultFields(data []byte) (DecodedFunctionResultFields, error) {
	var zero DecodedFunctionResultFields
	if len(data) < len(initialResponsePrefix)+8 {
		return zero, fmt.Errorf("%w: CPIC session-refresh response is truncated", ErrRange)
	}
	if !bytes.Equal(data[0:len(initialResponsePrefix)], initialResponsePrefix) {
		return zero, fmt.Errorf("%w: CPIC session-refresh response prefix is invalid", ErrProtocol)
	}
	decoded, err := DecodeFieldChainPrefix(data[len(initialResponsePrefix):], uint16(TagStart), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		return zero, err
	}
	trailerOffset := len(initialResponsePrefix) + decoded.BytesConsumed
	if len(data)-trailerOffset != 2 || binary.BigEndian.Uint16(data[trailerOffset:]) != uint16(TagEnd) {
		return zero, fmt.Errorf("%w: CPIC session-refresh response trailer is invalid", ErrProtocol)
	}
	responseStartIndex := -1
	responseStartCount := 0
	for i, f := range decoded.Fields {
		if f.Tag == uint16(TagResponseStart) {
			responseStartCount++
			if responseStartIndex < 0 {
				responseStartIndex = i
			}
			if len(f.Value) != 0 {
				return zero, fmt.Errorf("%w: CPIC session-refresh response must contain one empty embedded response marker", ErrProtocol)
			}
		}
	}
	if responseStartCount != 1 {
		return zero, fmt.Errorf("%w: CPIC session-refresh response must contain one empty embedded response marker", ErrProtocol)
	}
	if responseStartIndex < 2 || responseStartIndex > maxSessionRefreshPreambleFields {
		return zero, fmt.Errorf("%w: CPIC session-refresh preamble field count is invalid", ErrProtocol)
	}
	preambleBytes := 0
	seen := map[uint16]bool{}
	for index := 0; index < responseStartIndex; index++ {
		field := decoded.Fields[index]
		if !sessionRefreshPreambleTags[field.Tag] || seen[field.Tag] {
			return zero, fmt.Errorf("%w: CPIC session-refresh preamble contains an unknown or duplicate field", ErrProtocol)
		}
		seen[field.Tag] = true
		preambleBytes += len(field.Value)
		if preambleBytes > maxSessionRefreshPreambleBytes {
			return zero, fmt.Errorf("%w: CPIC session-refresh preamble exceeds its byte limit", ErrRange)
		}
	}
	if len(decoded.Fields) < 2 ||
		decoded.Fields[0].Tag != uint16(TagProtocolVersion) || len(decoded.Fields[0].Value) != 4 ||
		decoded.Fields[1].Tag != uint16(TagCapabilities) || len(decoded.Fields[1].Value) != len(initialCapabilities) {
		return zero, fmt.Errorf("%w: CPIC session-refresh preamble lacks its protocol and Unicode headers", ErrProtocol)
	}
	for _, field := range decoded.Fields[:responseStartIndex] {
		if field.Tag == uint16(TagLogonStatus) {
			if len(field.Value) != 1 || field.Value[0] != 0 {
				return zero, fmt.Errorf("%w: CPIC session-refresh preamble has a nonzero status", ErrProtocol)
			}
			break
		}
	}
	return decodeFunctionResponseFields(decoded.Fields[responseStartIndex+1:], nil)
}

// DecodeFunctionResponse decodes structural outcome and status from a regular
// Unicode RFC response.
func DecodeFunctionResponse(data []byte) (DecodedFunctionResponse, error) {
	decoded, err := decodeFunctionResponseEnvelope(data, []uint16{uint16(TagXRfcParameter), uint16(TagXRfcData)})
	if err != nil {
		return DecodedFunctionResponse{}, err
	}
	resp := DecodedFunctionResponse{
		Success: decoded.Success,
		Outcome: decoded.Envelope.Outcome,
		Status:  decoded.Status,
		Fields:  tagLens(decoded.Fields),
	}
	if decoded.Envelope.Outcome == rfcerr.OutcomeAbapException {
		resp.ExceptionKey = decoded.Envelope.Facts.ExceptionKey
		resp.HasException = true
	}
	return resp, nil
}

func intptr(v int) *int { return &v }

func concat(parts ...[]byte) []byte {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	out := make([]byte, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
