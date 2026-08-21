// SPDX-License-Identifier: Apache-2.0
//
// Part of the cpic package; see cpic.go for the file's provenance header.

package cpic

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/oisee/open-rfc-go/internal/rfcerr"
	"github.com/oisee/open-rfc-go/internal/scramble"
)

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

var (
	initialSignature           = mustHex("d9c6c3f0f0f0f0f0f0f0f0f0")
	initialPrefix              = mustHex("010100080301")
	initialResponsePrefix      = mustHex("010100080101010504010003")
	initialErrorResponsePrefix = mustHex("010100080101010101010000")
	initialProtocolVersion     = mustHex("00000e09")
	initialCapabilities        = mustHex("04010003000a0200000023")
)

var initialErrorPreambleTags = []uint16{
	uint16(TagProtocolVersion), uint16(TagCapabilities), uint16(TagSystemCodePage),
	uint16(TagClientAddress), uint16(TagPartnerSystem), uint16(TagPartnerHost),
	uint16(TagConnectionType), uint16(TagKernelPatch), uint16(TagKernelRelease),
	uint16(TagDestination), uint16(TagProgram), uint16(TagResponseStart),
}

var initialRegularResponseTags = map[uint16]bool{
	uint16(TagStart): true, uint16(TagProtocolVersion): true, uint16(TagCapabilities): true,
	uint16(TagLogonStatus): true, uint16(TagUnresolved0420): true, initialCpicUnresolved0450: true,
	uint16(TagSystemCodePage): true, uint16(TagEnd): true,
}

// TagLen is a redaction-safe (tag, byteLength) pair.
type TagLen struct {
	Tag        uint16
	ByteLength int
}

// InitialLogonRequestInput is the semantic input to the initial logon request.
// PartnerSystem, FunctionName, and KernelRelease default when empty.
type InitialLogonRequestInput struct {
	Client               string
	User                 string
	Password             string
	// Ticket, when set, logs on with an SAP logon ticket instead of a password:
	// the credential field becomes TagTicket (0x0670) carrying the ticket's
	// base64 text as UTF-16LE, and the password field is omitted. Give it the
	// ticket in any of its wire forms (canonical base64, the cookie form with
	// '!' for '/', or URL-escaped); NormalizeTicket sorts it out.
	Ticket               string
	Language             string
	ClientAddress        string
	PartnerSystem        string
	PartnerHostName      string
	Destination          string
	ProgramName          string
	FunctionName         string
	KernelRelease        string
	SessionID            []byte
	PasswordSeed         *uint32
	MaximumRfcPacketSize *uint32
}

// DecodedInitialLogonRequest is the structural, non-secret view of a request.
type DecodedInitialLogonRequest struct {
	Fields               []TagLen
	CpicPacketSize       int
	MaximumRfcPacketSize uint32
}

// Rejection is the backend's own reason for a rejected logon.
type Rejection struct {
	Outcome       rfcerr.Outcome
	MessageClass  string
	MessageType   string
	MessageNumber string
	ExceptionKey  string
	RuntimeID     string
	Text          string
}

// DecodedInitialLogonResponse is the structural outcome of the first response.
type DecodedInitialLogonResponse struct {
	Success                   bool
	Status                    *int
	Rejection                 *Rejection
	NegotiatedProtocolVersion uint32
	Fields                    []TagLen
}

var initialTagOrder = []uint16{
	uint16(TagStart), uint16(TagProtocolVersion), uint16(TagCapabilities), uint16(TagLogonMarker),
	uint16(TagSession), uint16(TagClient), uint16(TagUser), uint16(TagPassword), uint16(TagLanguage),
	uint16(TagUnicodeIndicator), uint16(TagClientAddress), uint16(TagPartnerSystem), uint16(TagConnectionType),
	uint16(TagKernelRelease), uint16(TagKernelPatch), uint16(TagPartnerHost), uint16(TagDestination),
	uint16(TagProgram), uint16(TagContextEnd), uint16(TagKernel), uint16(TagFunction), uint16(TagEnd),
}

// EncodeInitialLogonRequest encodes the bounded initial CPIC logon request.
func EncodeInitialLogonRequest(input InitialLogonRequestInput) ([]byte, error) {
	if !digits3.MatchString(input.Client) {
		return nil, fmt.Errorf("%w: client must contain exactly three ASCII digits", ErrRange)
	}
	if !oneLetter.MatchString(input.Language) {
		return nil, fmt.Errorf("%w: language must contain one ASCII letter", ErrRange)
	}
	kernelRelease := input.KernelRelease
	if kernelRelease == "" {
		kernelRelease = "754"
	}
	if !digits3.MatchString(kernelRelease) {
		return nil, fmt.Errorf("%w: kernelRelease must contain exactly three ASCII digits", ErrRange)
	}
	maximumRfcPacketSize := uint32(0x8500)
	if input.MaximumRfcPacketSize != nil {
		maximumRfcPacketSize = *input.MaximumRfcPacketSize
	}

	var password []byte
	var err error
	if input.PasswordSeed != nil {
		password, err = scramble.ScrambleRfcPassword(input.Password, *input.PasswordSeed)
	} else {
		password, err = scramble.ScrambleRfcPasswordRandomSeed(input.Password)
	}
	if err != nil {
		return nil, err
	}

	sessionID := input.SessionID
	if sessionID == nil {
		sessionID = make([]byte, 16)
		if _, err := randRead(sessionID); err != nil {
			return nil, err
		}
	}
	sessionID, err = exactBytes(sessionID, 16, "sessionId")
	if err != nil {
		return nil, err
	}

	partnerSystem := input.PartnerSystem
	if partnerSystem == "" {
		partnerSystem = "::1"
	}
	functionName := input.FunctionName
	if functionName == "" {
		functionName = "RFCPING"
	}

	user, err := asciiBytes(input.User, "user", 1, 40)
	if err != nil {
		return nil, err
	}
	clientAddress, err := asciiBytes(input.ClientAddress, "clientAddress", 1, 64)
	if err != nil {
		return nil, err
	}
	partnerSystemBytes, err := asciiBytes(partnerSystem, "partnerSystem", 1, 64)
	if err != nil {
		return nil, err
	}
	partnerHost, err := asciiBytes(input.PartnerHostName, "partnerHostName", 1, 120)
	if err != nil {
		return nil, err
	}
	destination, err := asciiBytes(input.Destination, "destination", 1, 120)
	if err != nil {
		return nil, err
	}
	program, err := asciiBytes(input.ProgramName, "programName", 1, 64)
	if err != nil {
		return nil, err
	}
	functionBytes, err := asciiBytes(functionName, "functionName", 1, 40)
	if err != nil {
		return nil, err
	}

	// The credential is either a password (TagPassword) or a logon ticket
	// (TagTicket): a real ticket-bearing logon carries the ticket field and no
	// password field at all, so we swap rather than add.
	credential := Field{Tag: uint16(TagPassword), Value: password}
	if input.Ticket != "" {
		credential = Field{Tag: uint16(TagTicket), Value: encodeTicketField(input.Ticket)}
	}

	fields := []Field{
		{Tag: uint16(TagStart), Value: nil},
		{Tag: uint16(TagProtocolVersion), Value: initialProtocolVersion},
		{Tag: uint16(TagCapabilities), Value: initialCapabilities},
		{Tag: uint16(TagLogonMarker), Value: nil},
		{Tag: uint16(TagSession), Value: sessionID},
		{Tag: uint16(TagClient), Value: []byte(input.Client)},
		{Tag: uint16(TagUser), Value: user},
		credential,
		{Tag: uint16(TagLanguage), Value: []byte(upperASCII(input.Language))},
		{Tag: uint16(TagUnicodeIndicator), Value: []byte{1}},
		{Tag: uint16(TagClientAddress), Value: clientAddress},
		{Tag: uint16(TagPartnerSystem), Value: partnerSystemBytes},
		{Tag: uint16(TagConnectionType), Value: []byte("E")},
		{Tag: uint16(TagKernelRelease), Value: []byte(kernelRelease)},
		{Tag: uint16(TagKernelPatch), Value: []byte(kernelRelease)},
		{Tag: uint16(TagPartnerHost), Value: partnerHost},
		{Tag: uint16(TagDestination), Value: destination},
		{Tag: uint16(TagProgram), Value: program},
		{Tag: uint16(TagContextEnd), Value: nil},
		{Tag: uint16(TagKernel), Value: []byte(kernelRelease)},
		{Tag: uint16(TagFunction), Value: functionBytes},
		{Tag: uint16(TagEnd), Value: nil},
	}
	chainByteLength, err := FieldChainByteLength(uint16(TagStart), fields, FieldChainLimits{})
	if err != nil {
		return nil, err
	}
	cpicPacketSize := len(initialSignature) + len(initialPrefix) + chainByteLength + 2
	if cpicPacketSize > 0xffff {
		return nil, fmt.Errorf("%w: initial CPIC packet size %d exceeds 65535", ErrRange, cpicPacketSize)
	}
	chain, err := EncodeFieldChain(uint16(TagStart), fields, FieldChainLimits{})
	if err != nil {
		return nil, err
	}
	trailer := make([]byte, 10)
	binary.BigEndian.PutUint16(trailer[0:], uint16(TagEnd))
	binary.BigEndian.PutUint16(trailer[2:], 0)
	binary.BigEndian.PutUint16(trailer[4:], uint16(cpicPacketSize))
	binary.BigEndian.PutUint32(trailer[6:], maximumRfcPacketSize)

	out := make([]byte, 0, len(initialSignature)+len(initialPrefix)+len(chain)+len(trailer))
	out = append(out, initialSignature...)
	out = append(out, initialPrefix...)
	out = append(out, chain...)
	out = append(out, trailer...)
	return out, nil
}

// DecodeInitialLogonRequest decodes only the structural, non-secret properties.
func DecodeInitialLogonRequest(data []byte) (DecodedInitialLogonRequest, error) {
	var zero DecodedInitialLogonRequest
	prefixLength := len(initialSignature) + len(initialPrefix)
	if len(data) < prefixLength+10 {
		return zero, fmt.Errorf("%w: initial CPIC logon request is truncated", ErrRange)
	}
	if !bytes.Equal(data[0:12], initialSignature) {
		return zero, fmt.Errorf("%w: initial CPIC logon signature is invalid", ErrProtocol)
	}
	if !bytes.Equal(data[12:prefixLength], initialPrefix) {
		return zero, fmt.Errorf("%w: initial CPIC logon prefix is invalid", ErrProtocol)
	}
	decoded, err := DecodeFieldChainPrefix(data[prefixLength:], uint16(TagStart), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		return zero, err
	}
	if len(decoded.Fields) != len(initialTagOrder) {
		return zero, fmt.Errorf("%w: initial CPIC logon fields do not match the required tag order", ErrProtocol)
	}
	for i, field := range decoded.Fields {
		if field.Tag != initialTagOrder[i] {
			return zero, fmt.Errorf("%w: initial CPIC logon fields do not match the required tag order", ErrProtocol)
		}
	}
	if !bytes.Equal(decoded.Fields[1].Value, initialProtocolVersion) {
		return zero, fmt.Errorf("%w: initial CPIC protocol-version field is unsupported", ErrProtocol)
	}
	if !bytes.Equal(decoded.Fields[2].Value, initialCapabilities) {
		return zero, fmt.Errorf("%w: initial CPIC capabilities field is unsupported", ErrProtocol)
	}
	trailerOffset := prefixLength + decoded.BytesConsumed
	if len(data)-trailerOffset != 10 {
		return zero, fmt.Errorf("%w: initial CPIC logon request has an invalid trailer length", ErrProtocol)
	}
	trailer := data[trailerOffset:]
	if binary.BigEndian.Uint16(trailer[0:]) != uint16(TagEnd) || binary.BigEndian.Uint16(trailer[2:]) != 0 {
		return zero, fmt.Errorf("%w: initial CPIC logon trailer marker is invalid", ErrProtocol)
	}
	cpicPacketSize := int(binary.BigEndian.Uint16(trailer[4:]))
	if cpicPacketSize != trailerOffset+2 {
		return zero, fmt.Errorf("%w: initial CPIC packet size %d does not match derived size %d", ErrProtocol, cpicPacketSize, trailerOffset+2)
	}
	return DecodedInitialLogonRequest{
		Fields:               tagLens(decoded.Fields),
		CpicPacketSize:       cpicPacketSize,
		MaximumRfcPacketSize: binary.BigEndian.Uint32(trailer[6:]),
	}, nil
}

// DecodeInitialLogonResponse decodes the structural outcome of the first
// server logon/RFCPING response.
func DecodeInitialLogonResponse(data []byte) (DecodedInitialLogonResponse, error) {
	var zero DecodedInitialLogonResponse
	if len(data) < len(initialResponsePrefix)+8 {
		return zero, failParse("truncated", "initial CPIC logon response is truncated")
	}
	prefix := data[0:len(initialResponsePrefix)]
	isRegular := bytes.Equal(prefix, initialResponsePrefix)
	isError := bytes.Equal(prefix, initialErrorResponsePrefix)
	if !isRegular && !isError {
		return zero, failParse("prefix", "initial CPIC logon response prefix is invalid")
	}
	decoded, err := DecodeFieldChainPrefix(data[len(initialResponsePrefix):], uint16(TagStart), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		return zero, wrapParse("field-chain", err)
	}
	trailerOffset := len(initialResponsePrefix) + decoded.BytesConsumed
	if len(data)-trailerOffset != 2 || binary.BigEndian.Uint16(data[trailerOffset:]) != uint16(TagEnd) {
		return zero, failParse("trailer", "initial CPIC logon response trailer is invalid")
	}
	var protocol *Field
	protocolCount := 0
	for i := range decoded.Fields {
		if decoded.Fields[i].Tag == uint16(TagProtocolVersion) {
			protocolCount++
			if protocol == nil {
				protocol = &decoded.Fields[i]
			}
		}
	}
	if protocol == nil || len(protocol.Value) != 4 {
		return zero, failParse("protocol", "initial CPIC logon response lacks its protocol version")
	}
	structural := structuralFields(decoded.Fields)
	negotiated := binary.BigEndian.Uint32(protocol.Value)

	if isError {
		if len(decoded.Fields) <= len(initialErrorPreambleTags)+1 || !preambleMatches(decoded.Fields, initialErrorPreambleTags) ||
			len(decoded.Fields[len(initialErrorPreambleTags)-1].Value) != 0 {
			return zero, failParse("error-preamble", "initial CPIC logon error response has an invalid preamble")
		}
		for _, tag := range initialErrorPreambleTags {
			if countTag(decoded.Fields, tag) != 1 {
				return zero, failParse("error-preamble", "initial CPIC logon error response has duplicate preamble fields")
			}
		}
		envelope, err := rfcerr.Decode(toRfcErrFields(decoded.Fields), rfcerr.DecodeOptions{AdditionalAllowedTags: initialErrorPreambleTags})
		if err != nil {
			return zero, wrapParse("error-envelope", err)
		}
		if envelope.Outcome == rfcerr.OutcomeSuccess {
			return zero, failParse("error-envelope", "initial CPIC logon error response lacks a rejected outcome")
		}
		return DecodedInitialLogonResponse{
			Success: false,
			Rejection: &Rejection{
				Outcome:       envelope.Outcome,
				MessageClass:  envelope.Facts.MessageClass,
				MessageType:   envelope.Facts.MessageType,
				MessageNumber: envelope.Facts.MessageNumber,
				ExceptionKey:  envelope.Facts.ExceptionKey,
				RuntimeID:     envelope.Facts.RuntimeID,
				Text:          envelope.Facts.PlainText,
			},
			NegotiatedProtocolVersion: negotiated,
			Fields:                    tagLens(decoded.Fields),
		}, nil
	}

	if protocolCount != 1 {
		return zero, failParse("protocol", "initial CPIC logon response lacks its protocol version")
	}
	if anyTag(decoded.Fields, uint16(TagResponseStart)) {
		return decodeRichInitialRfcPingResponse(decoded.Fields, structural, negotiated)
	}
	for index, field := range decoded.Fields {
		if !initialRegularResponseTags[field.Tag] {
			return zero, failStructure("unsupported-field",
				fmt.Sprintf("initial CPIC logon response contains unsupported field %s (%d bytes) at index %d", tagText(field.Tag), len(field.Value), index),
				structural)
		}
	}
	if err := validateEndField(decoded.Fields, structural); err != nil {
		return zero, err
	}
	startCount := countTag(decoded.Fields, uint16(TagStart))
	if startCount > 1 || (startCount == 1 && (decoded.Fields[0].Tag != uint16(TagStart) || len(decoded.Fields[0].Value) != 0)) {
		return zero, failStructure("invalid-start-field", "initial CPIC logon response has an invalid Start field", structural)
	}
	// Validate the S/4 0x0450 control against the semantic response shape so
	// transport decoration cannot shift an otherwise identical grammar.
	semantic := decoded.Fields
	if startCount == 1 {
		semantic = decoded.Fields[1:]
	}
	var s4 *Field
	s4count := 0
	s4semanticIndex := -1
	for i := range semantic {
		if semantic[i].Tag == initialCpicUnresolved0450 {
			s4count++
			if s4 == nil {
				s4 = &semantic[i]
				s4semanticIndex = i
			}
		}
	}
	if s4count > 1 || (s4count == 1 && (len(s4.Value) != 6 || s4semanticIndex != 4 ||
		(len(semantic) < 4 || semantic[3].Tag != uint16(TagUnresolved0420)))) {
		return zero, failStructure("malformed-vendor-logon-control", "initial CPIC logon response has malformed 0x0450 control", structural)
	}
	for _, singleton := range []uint16{uint16(TagCapabilities), uint16(TagSystemCodePage)} {
		if countTag(decoded.Fields, singleton) > 1 {
			return zero, failStructure("duplicate-control-field", "initial CPIC logon response has duplicate control fields", structural)
		}
	}
	oneByte := filterTag(decoded.Fields, uint16(TagLogonStatus))
	if len(oneByte) > 1 || (len(oneByte) == 1 && len(oneByte[0].Value) != 1) {
		return zero, failStructure("malformed-one-byte-status", "initial CPIC logon response has malformed one-byte status", structural)
	}
	callStatuses := filterTag(decoded.Fields, uint16(TagUnresolved0420))
	if len(callStatuses) > 1 || (len(callStatuses) == 1 && len(callStatuses[0].Value) != 4) {
		return zero, failStructure("malformed-call-status", "initial CPIC logon response has malformed call status", structural)
	}
	if len(oneByte) == 0 && len(callStatuses) == 0 {
		return zero, failStructure("missing-logon-status", "initial CPIC logon response lacks a recognized logon status", structural)
	}
	if len(callStatuses) == 1 && binary.BigEndian.Uint32(callStatuses[0].Value) != 0 {
		return zero, failStructure("nonzero-call-status", "initial CPIC logon response has nonzero call status", structural)
	}
	status := 0
	if len(oneByte) == 1 {
		status = int(oneByte[0].Value[0])
	}
	statusCopy := status
	return DecodedInitialLogonResponse{
		Success:                   status == 0,
		Status:                    &statusCopy,
		NegotiatedProtocolVersion: negotiated,
		Fields:                    tagLens(decoded.Fields),
	}, nil
}

func upperASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 32
		}
	}
	return string(b)
}

func tagLens(fields []Field) []TagLen {
	out := make([]TagLen, len(fields))
	for i, f := range fields {
		out[i] = TagLen{Tag: f.Tag, ByteLength: len(f.Value)}
	}
	return out
}

func structuralFields(fields []Field) []StructuralField {
	out := make([]StructuralField, len(fields))
	for i, f := range fields {
		out[i] = StructuralField{Tag: f.Tag, ByteLength: len(f.Value), Index: i}
	}
	return out
}

func toRfcErrFields(fields []Field) []rfcerr.Field {
	out := make([]rfcerr.Field, len(fields))
	for i, f := range fields {
		out[i] = rfcerr.Field{Tag: f.Tag, Value: f.Value}
	}
	return out
}

func countTag(fields []Field, tag uint16) int {
	n := 0
	for _, f := range fields {
		if f.Tag == tag {
			n++
		}
	}
	return n
}

func anyTag(fields []Field, tag uint16) bool { return countTag(fields, tag) > 0 }

func filterTag(fields []Field, tag uint16) []Field {
	var out []Field
	for _, f := range fields {
		if f.Tag == tag {
			out = append(out, f)
		}
	}
	return out
}

func preambleMatches(fields []Field, tags []uint16) bool {
	for i, tag := range tags {
		if i >= len(fields) || fields[i].Tag != tag {
			return false
		}
	}
	return true
}

func validateEndField(fields []Field, structural []StructuralField) error {
	ends := filterTag(fields, uint16(TagEnd))
	if len(ends) != 1 || fields[len(fields)-1].Tag != uint16(TagEnd) || len(ends[0].Value) != 0 {
		return failStructure("invalid-end-field", "initial CPIC logon response has an invalid End field", structural)
	}
	return nil
}
