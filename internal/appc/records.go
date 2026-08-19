// SPDX-License-Identifier: Apache-2.0
//
// Part of the appc package; see appc.go for the file's provenance header.

package appc

import (
	"fmt"

	"github.com/oisee/open-rfc-go/internal/wire"
)

// RecordHeaderInput carries the optional common-header overrides. A nil field
// takes the wire default the encoder applies (mirroring TypeScript `?? default`).
type RecordHeaderInput struct {
	Protocol       *uint8
	Mode           *uint8
	UID            *uint16
	GatewayID      *uint16
	ErrorLength    *uint16
	Info2          *uint8
	TraceLevel     *uint8
	Time           *uint32
	Info3          *uint8
	Timeout        *int32
	Info4          *uint8
	SequenceNumber *uint32
	Padding        *uint16
	Info           *uint8
	Vector         *uint8
	AppcReturnCode *uint32
	SapReturnCode  *uint32
	ConversationID []byte
}

func u8or(p *uint8, d uint8) uint8 {
	if p != nil {
		return *p
	}
	return d
}
func u16or(p *uint16, d uint16) uint16 {
	if p != nil {
		return *p
	}
	return d
}
func u32or(p *uint32, d uint32) uint32 {
	if p != nil {
		return *p
	}
	return d
}
func i32or(p *int32, d int32) int32 {
	if p != nil {
		return *p
	}
	return d
}

// ExtendedInfo is the fixed 32-byte CPIC extended-connection structure.
type ExtendedInfo struct {
	ShortDestinationName   string
	LogicalUnitName        string
	TransactionProgramName string
	ConnectionType         uint8
	ClientInfo             uint8
	CommunicationIndex     uint16
	ConnectionIndex        uint16
}

func defaultExtendedInfo() ExtendedInfo { return ExtendedInfo{} }

// EncodeExtendedInfo encodes the 32-byte extended connection info.
func EncodeExtendedInfo(info ExtendedInfo) ([]byte, error) {
	w, _ := wire.NewWriter(32, "APPC extended info")
	var err error
	put := func(f func() error) {
		if err == nil {
			err = f()
		}
	}
	putField := func(field string, enc func() ([]byte, error)) {
		put(func() error {
			b, e := enc()
			if e != nil {
				return e
			}
			return w.WriteBytes(b, field)
		})
	}
	putField("shortDestinationName", func() ([]byte, error) { return encodeFixedAscii(info.ShortDestinationName, "shortDestinationName") })
	putField("logicalUnitName", func() ([]byte, error) { return encodeFixedAscii(info.LogicalUnitName, "logicalUnitName") })
	putField("transactionProgramName", func() ([]byte, error) { return encodeFixedAscii(info.TransactionProgramName, "transactionProgramName") })
	put(func() error { return w.WriteUint8(info.ConnectionType, "connectionType") })
	put(func() error { return w.WriteUint8(info.ClientInfo, "clientInfo") })
	put(func() error { return w.WriteUint16BE(0, "reserved") })
	put(func() error { return w.WriteUint16BE(info.CommunicationIndex, "communicationIndex") })
	put(func() error { return w.WriteUint16BE(info.ConnectionIndex, "connectionIndex") })
	if err != nil {
		return nil, err
	}
	return w.Finish()
}

// DecodeExtendedInfo decodes the 32-byte extended connection info.
func DecodeExtendedInfo(data []byte) (ExtendedInfo, error) {
	if len(data) != 32 {
		return ExtendedInfo{}, fmt.Errorf("%w: APPC extended info needs exactly 32 bytes; received %d", ErrRange, len(data))
	}
	r := wire.NewReader(data, "APPC extended info")
	var info ExtendedInfo
	var err error
	take := func(f func() error) {
		if err == nil {
			err = f()
		}
	}
	takeAscii := func(field string, dst *string) {
		take(func() error {
			b, e := r.ReadBytes(8, field)
			if e != nil {
				return e
			}
			*dst, e = decodeFixedAscii(b, field)
			return e
		})
	}
	takeAscii("shortDestinationName", &info.ShortDestinationName)
	takeAscii("logicalUnitName", &info.LogicalUnitName)
	takeAscii("transactionProgramName", &info.TransactionProgramName)
	take(func() error { var e error; info.ConnectionType, e = r.ReadUint8("connectionType"); return e })
	take(func() error { var e error; info.ClientInfo, e = r.ReadUint8("clientInfo"); return e })
	take(func() error {
		reserved, e := r.ReadUint16BE("reserved")
		if e == nil && reserved != 0 {
			return fmt.Errorf("%w: APPC extended info reserved field must be zero; received %d", ErrProtocol, reserved)
		}
		return e
	})
	take(func() error { var e error; info.CommunicationIndex, e = r.ReadUint16BE("communicationIndex"); return e })
	take(func() error { var e error; info.ConnectionIndex, e = r.ReadUint16BE("connectionIndex"); return e })
	take(func() error { return r.Finish() })
	if err != nil {
		return ExtendedInfo{}, err
	}
	return info, nil
}

// PartnerLogicalUnitInfoInput is the semantic input to F_SET_PARTNER_LU_NAME.
type PartnerLogicalUnitInfoInput struct {
	LogicalUnitName    string
	PartnerHostAddress []byte
	CommunicationIndex uint16
	ConnectionIndex    uint16
}

// PartnerLogicalUnitInfo is the decoded 32-byte F_SET_PARTNER_LU_NAME info.
type PartnerLogicalUnitInfo struct {
	LogicalUnitNamePrefix string
	LogicalUnitNameLength uint32
	PartnerHostAddress    []byte
	CommunicationIndex    uint16
	ConnectionIndex       uint16
}

// EncodePartnerLogicalUnitInfo encodes the 32-byte F_SET_PARTNER_LU_NAME block.
func EncodePartnerLogicalUnitInfo(info PartnerLogicalUnitInfoInput) ([]byte, error) {
	if !printableASCII.MatchString(info.LogicalUnitName) || len(info.LogicalUnitName) > 128 {
		return nil, fmt.Errorf("%w: logicalUnitName must contain at most 128 ASCII bytes", ErrRange)
	}
	name := info.LogicalUnitName
	if len(name) > 8 {
		name = name[:8]
	}
	prefix, err := encodeFixedAscii(name, "logicalUnitNamePrefix")
	if err != nil {
		return nil, err
	}
	if len(info.PartnerHostAddress) != 16 {
		return nil, fmt.Errorf("%w: partnerHostAddress must contain exactly 16 bytes; received %d", ErrRange, len(info.PartnerHostAddress))
	}
	w, _ := wire.NewWriter(32, "APPC partner logical-unit info")
	var werr error
	put := func(f func() error) {
		if werr == nil {
			werr = f()
		}
	}
	put(func() error { return w.WriteBytes(prefix, "logicalUnitNamePrefix") })
	put(func() error { return w.WriteUint32BE(uint32(len(info.LogicalUnitName)), "logicalUnitNameLength") })
	put(func() error { return w.WriteBytes(info.PartnerHostAddress, "partnerHostAddress") })
	put(func() error { return w.WriteUint16BE(info.CommunicationIndex, "communicationIndex") })
	put(func() error { return w.WriteUint16BE(info.ConnectionIndex, "connectionIndex") })
	if werr != nil {
		return nil, werr
	}
	return w.Finish()
}

// DecodePartnerLogicalUnitInfo decodes and validates an F_SET_PARTNER_LU_NAME block.
func DecodePartnerLogicalUnitInfo(data []byte) (PartnerLogicalUnitInfo, error) {
	var zero PartnerLogicalUnitInfo
	if len(data) != 32 {
		return zero, fmt.Errorf("%w: APPC partner logical-unit info needs exactly 32 bytes; received %d", ErrRange, len(data))
	}
	r := wire.NewReader(data, "APPC partner logical-unit info")
	encodedName, err := r.ReadBytes(8, "logicalUnitNamePrefix")
	if err != nil {
		return zero, err
	}
	nameLength, err := r.ReadUint32BE("logicalUnitNameLength")
	if err != nil {
		return zero, err
	}
	if nameLength > 128 {
		return zero, fmt.Errorf("%w: APPC partner logical-unit name length must be at most 128; received %d", ErrProtocol, nameLength)
	}
	prefix, err := decodeFixedAscii(encodedName, "logicalUnitNamePrefix")
	if err != nil {
		return zero, err
	}
	expected := int(nameLength)
	if expected > 8 {
		expected = 8
	}
	if len(prefix) != expected {
		return zero, fmt.Errorf("%w: APPC partner logical-unit name prefix length %d does not match declared length %d", ErrProtocol, len(prefix), nameLength)
	}
	host, err := r.ReadBytes(16, "partnerHostAddress")
	if err != nil {
		return zero, err
	}
	commIdx, err := r.ReadUint16BE("communicationIndex")
	if err != nil {
		return zero, err
	}
	connIdx, err := r.ReadUint16BE("connectionIndex")
	if err != nil {
		return zero, err
	}
	if err := r.Finish(); err != nil {
		return zero, err
	}
	return PartnerLogicalUnitInfo{
		LogicalUnitNamePrefix: prefix,
		LogicalUnitNameLength: nameLength,
		PartnerHostAddress:    host,
		CommunicationIndex:    commIdx,
		ConnectionIndex:       connIdx,
	}, nil
}

// ExtendedInitializeOptions is the fixed 341-byte extended init-options contract.
type ExtendedInitializeOptions struct {
	OptionFlags                uint8
	RootID                     string
	ConnectionID               string
	ConnectionIDSuffix         uint32
	Timeout                    int32
	KeepaliveTimeout           int32
	ExportTrace                uint8
	StartType                  uint8
	NetworkProtocol            uint8
	LocalAddressV6             []byte
	LongLogicalUnitName        string
	OperatingSystemUser        string
	LocalAddressV4             []byte
	LongTransactionProgramName string
}

// EncodeExtendedInitializeOptions encodes the fixed 341-byte options block.
func EncodeExtendedInitializeOptions(o ExtendedInitializeOptions) ([]byte, error) {
	w, _ := wire.NewWriter(ExtendedInitializeOptionsLength, "APPC extended initialize options")
	var err error
	put := func(f func() error) {
		if err == nil {
			err = f()
		}
	}
	putEnc := func(field string, enc func() ([]byte, error)) {
		put(func() error {
			b, e := enc()
			if e != nil {
				return e
			}
			return w.WriteBytes(b, field)
		})
	}
	put(func() error { return w.WriteUint8(1, "version") })
	put(func() error { return w.WriteUint8(o.OptionFlags, "optionFlags") })
	putEnc("protocolName", func() ([]byte, error) { return encodePaddedAscii("CPIC", 8, 0, "protocolName") })
	putEnc("rootId", func() ([]byte, error) { return fixedHexID(o.RootID, "rootId") })
	putEnc("connectionId", func() ([]byte, error) { return fixedHexID(o.ConnectionID, "connectionId") })
	put(func() error { return w.WriteUint32BE(o.ConnectionIDSuffix, "connectionIdSuffix") })
	put(func() error { return w.WriteInt32BE(o.Timeout, "timeout") })
	put(func() error { return w.WriteInt32BE(o.KeepaliveTimeout, "keepaliveTimeout") })
	put(func() error { return w.WriteUint8(o.ExportTrace, "exportTrace") })
	put(func() error { return w.WriteUint8(o.StartType, "startType") })
	put(func() error { return w.WriteUint8(o.NetworkProtocol, "networkProtocol") })
	putEnc("localAddressV6", func() ([]byte, error) { return exactBytes(o.LocalAddressV6, 16, "localAddressV6") })
	putEnc("longLogicalUnitName", func() ([]byte, error) { return encodePaddedAscii(o.LongLogicalUnitName, 128, 0, "longLogicalUnitName") })
	put(func() error { return w.WriteBytes(make([]byte, 16), "reserved1") })
	putEnc("operatingSystemUser", func() ([]byte, error) {
		return encodePaddedAscii(o.OperatingSystemUser, 12, 0x20, "operatingSystemUser")
	})
	put(func() error { return w.WriteBytes(make([]byte, 8), "reserved2") })
	put(func() error { return w.WriteBytes(make([]byte, 4), "reserved3") })
	put(func() error { return w.WriteBytes(make([]byte, 12), "reserved4") })
	put(func() error { return w.WriteBytes(make([]byte, 16), "reserved5") })
	putEnc("localAddressV4", func() ([]byte, error) { return exactBytes(o.LocalAddressV4, 4, "localAddressV4") })
	put(func() error { return w.WriteBytes(make([]byte, 4), "reserved6") })
	putEnc("longTransactionProgramName", func() ([]byte, error) {
		return encodePaddedAscii(o.LongTransactionProgramName, 64, 0, "longTransactionProgramName")
	})
	if err != nil {
		return nil, err
	}
	return w.Finish()
}

// DecodeExtendedInitializeOptions decodes the options block, rejecting non-zero
// reserved bytes.
func DecodeExtendedInitializeOptions(data []byte) (ExtendedInitializeOptions, error) {
	var zero ExtendedInitializeOptions
	if len(data) != ExtendedInitializeOptionsLength {
		return zero, fmt.Errorf("%w: APPC extended initialize options need exactly %d bytes; received %d", ErrRange, ExtendedInitializeOptionsLength, len(data))
	}
	r := wire.NewReader(data, "APPC extended initialize options")
	var o ExtendedInitializeOptions

	version, err := r.ReadUint8("version")
	if err != nil {
		return zero, err
	}
	if version != 1 {
		return zero, fmt.Errorf("%w: unsupported APPC extended initialize options version %d", ErrUnsupported, version)
	}
	if o.OptionFlags, err = r.ReadUint8("optionFlags"); err != nil {
		return zero, err
	}
	protocolNameBytes, err := r.ReadBytes(8, "protocolName")
	if err != nil {
		return zero, err
	}
	protocolName, err := decodePaddedAscii(protocolNameBytes, 0, "protocolName")
	if err != nil {
		return zero, err
	}
	if protocolName != "CPIC" {
		return zero, fmt.Errorf("%w: unsupported APPC extended initialize protocol %s", ErrUnsupported, protocolName)
	}
	rootIDBytes, err := r.ReadBytes(16, "rootId")
	if err != nil {
		return zero, err
	}
	o.RootID = string(rootIDBytes)
	if _, err = fixedHexID(o.RootID, "rootId"); err != nil {
		return zero, err
	}
	connIDBytes, err := r.ReadBytes(16, "connectionId")
	if err != nil {
		return zero, err
	}
	o.ConnectionID = string(connIDBytes)
	if _, err = fixedHexID(o.ConnectionID, "connectionId"); err != nil {
		return zero, err
	}
	if o.ConnectionIDSuffix, err = r.ReadUint32BE("connectionIdSuffix"); err != nil {
		return zero, err
	}
	if o.Timeout, err = r.ReadInt32BE("timeout"); err != nil {
		return zero, err
	}
	if o.KeepaliveTimeout, err = r.ReadInt32BE("keepaliveTimeout"); err != nil {
		return zero, err
	}
	if o.ExportTrace, err = r.ReadUint8("exportTrace"); err != nil {
		return zero, err
	}
	if o.StartType, err = r.ReadUint8("startType"); err != nil {
		return zero, err
	}
	if o.NetworkProtocol, err = r.ReadUint8("networkProtocol"); err != nil {
		return zero, err
	}
	if o.LocalAddressV6, err = r.ReadBytes(16, "localAddressV6"); err != nil {
		return zero, err
	}
	longLuBytes, err := r.ReadBytes(128, "longLogicalUnitName")
	if err != nil {
		return zero, err
	}
	if o.LongLogicalUnitName, err = decodePaddedAscii(longLuBytes, 0, "longLogicalUnitName"); err != nil {
		return zero, err
	}
	if err = readZeroReserved(r, "reserved1", 16); err != nil {
		return zero, err
	}
	osUserBytes, err := r.ReadBytes(12, "operatingSystemUser")
	if err != nil {
		return zero, err
	}
	if o.OperatingSystemUser, err = decodePaddedAscii(osUserBytes, 0x20, "operatingSystemUser"); err != nil {
		return zero, err
	}
	for _, res := range []struct {
		field  string
		length int
	}{{"reserved2", 8}, {"reserved3", 4}, {"reserved4", 12}, {"reserved5", 16}} {
		if err = readZeroReserved(r, res.field, res.length); err != nil {
			return zero, err
		}
	}
	if o.LocalAddressV4, err = r.ReadBytes(4, "localAddressV4"); err != nil {
		return zero, err
	}
	if err = readZeroReserved(r, "reserved6", 4); err != nil {
		return zero, err
	}
	longTpBytes, err := r.ReadBytes(64, "longTransactionProgramName")
	if err != nil {
		return zero, err
	}
	if o.LongTransactionProgramName, err = decodePaddedAscii(longTpBytes, 0, "longTransactionProgramName"); err != nil {
		return zero, err
	}
	if err = r.Finish(); err != nil {
		return zero, err
	}
	return o, nil
}

func readZeroReserved(r *wire.Reader, field string, length int) error {
	b, err := r.ReadBytes(length, field)
	if err != nil {
		return err
	}
	for _, x := range b {
		if x != 0 {
			return fmt.Errorf("%w: APPC extended initialize %s must be zero", ErrProtocol, field)
		}
	}
	return nil
}

// InitializeParameters is the fixed client identifier plus extended options.
type InitializeParameters struct {
	ClientIdentifier string
	Options          ExtendedInitializeOptions
}

// EncodeInitializeParameters encodes the 373-byte F_INITIALIZE structure.
func EncodeInitializeParameters(p InitializeParameters) ([]byte, error) {
	clientID, err := encodePaddedAscii(p.ClientIdentifier, 32, 0x20, "clientIdentifier")
	if err != nil {
		return nil, err
	}
	options, err := EncodeExtendedInitializeOptions(p.Options)
	if err != nil {
		return nil, err
	}
	w, _ := wire.NewWriter(InitializeParametersLength, "APPC initialize parameters")
	if err := w.WriteBytes(clientID, "clientIdentifier"); err != nil {
		return nil, err
	}
	if err := w.WriteBytes(options, "extendedOptions"); err != nil {
		return nil, err
	}
	return w.Finish()
}

// DecodeInitializeParameters decodes the 373-byte F_INITIALIZE structure.
func DecodeInitializeParameters(data []byte) (InitializeParameters, error) {
	var zero InitializeParameters
	if len(data) != InitializeParametersLength {
		return zero, fmt.Errorf("%w: APPC initialize parameters need exactly %d bytes; received %d", ErrRange, InitializeParametersLength, len(data))
	}
	r := wire.NewReader(data, "APPC initialize parameters")
	clientIDBytes, err := r.ReadBytes(32, "clientIdentifier")
	if err != nil {
		return zero, err
	}
	clientID, err := decodePaddedAscii(clientIDBytes, 0x20, "clientIdentifier")
	if err != nil {
		return zero, err
	}
	optionsBytes, err := r.ReadBytes(ExtendedInitializeOptionsLength, "extendedOptions")
	if err != nil {
		return zero, err
	}
	options, err := DecodeExtendedInitializeOptions(optionsBytes)
	if err != nil {
		return zero, err
	}
	if err := r.Finish(); err != nil {
		return zero, err
	}
	return InitializeParameters{ClientIdentifier: clientID, Options: options}, nil
}

// PartnerLogicalUnitParameters is the fixed 144-byte partner parameter block.
type PartnerLogicalUnitParameters struct {
	LongLogicalUnitName string
	PartnerHostAddress  []byte
}

// EncodePartnerLogicalUnitParameters encodes the 144-byte partner block.
func EncodePartnerLogicalUnitParameters(p PartnerLogicalUnitParameters) ([]byte, error) {
	name, err := encodePaddedAscii(p.LongLogicalUnitName, 128, 0x20, "longLogicalUnitName")
	if err != nil {
		return nil, err
	}
	host, err := exactBytes(p.PartnerHostAddress, 16, "partnerHostAddress")
	if err != nil {
		return nil, err
	}
	w, _ := wire.NewWriter(PartnerParametersLength, "APPC partner logical-unit parameters")
	if err := w.WriteBytes(name, "longLogicalUnitName"); err != nil {
		return nil, err
	}
	if err := w.WriteBytes(host, "partnerHostAddress"); err != nil {
		return nil, err
	}
	return w.Finish()
}

// DecodePartnerLogicalUnitParameters decodes the 144-byte partner block.
func DecodePartnerLogicalUnitParameters(data []byte) (PartnerLogicalUnitParameters, error) {
	var zero PartnerLogicalUnitParameters
	if len(data) != PartnerParametersLength {
		return zero, fmt.Errorf("%w: APPC partner logical-unit parameters need exactly %d bytes; received %d", ErrRange, PartnerParametersLength, len(data))
	}
	r := wire.NewReader(data, "APPC partner logical-unit parameters")
	nameBytes, err := r.ReadBytes(128, "longLogicalUnitName")
	if err != nil {
		return zero, err
	}
	name, err := decodePaddedAscii(nameBytes, 0x20, "longLogicalUnitName")
	if err != nil {
		return zero, err
	}
	host, err := r.ReadBytes(16, "partnerHostAddress")
	if err != nil {
		return zero, err
	}
	if err := r.Finish(); err != nil {
		return zero, err
	}
	return PartnerLogicalUnitParameters{LongLogicalUnitName: name, PartnerHostAddress: host}, nil
}

// ControlRecordInput is the semantic input to a setup/control APPC record.
type ControlRecordInput struct {
	RecordHeaderInput
	FunctionCode           Function
	ExtendedInfo           *ExtendedInfo
	PartnerLogicalUnitInfo *PartnerLogicalUnitInfoInput
	Parameters             []byte
}

// EncodeControlRecord encodes a setup/control record; its parameter length is
// always derived from the semantic parameter bytes.
func EncodeControlRecord(in ControlRecordInput) ([]byte, error) {
	if !controlFunctions[in.FunctionCode] {
		return nil, fmt.Errorf("%w: %s is not a setup/control function", ErrProtocol, FunctionName(in.FunctionCode))
	}
	parameters := append([]byte(nil), in.Parameters...)
	if len(parameters) > 0xffff {
		return nil, fmt.Errorf("%w: APPC control parameter length %d exceeds 65535", ErrRange, len(parameters))
	}
	if in.ExtendedInfo != nil && in.PartnerLogicalUnitInfo != nil {
		return nil, fmt.Errorf("%w: an APPC control record cannot contain two operation-info variants", ErrProtocol)
	}
	if in.PartnerLogicalUnitInfo != nil && in.FunctionCode != FuncSetPartnerLuName {
		return nil, fmt.Errorf("%w: partnerLogicalUnitInfo is only valid for F_SET_PARTNER_LU_NAME", ErrProtocol)
	}
	if in.FunctionCode == FuncSetPartnerLuName && in.PartnerLogicalUnitInfo == nil {
		return nil, fmt.Errorf("%w: F_SET_PARTNER_LU_NAME requires partnerLogicalUnitInfo", ErrProtocol)
	}

	var operationInfo []byte
	var err error
	if in.PartnerLogicalUnitInfo != nil {
		operationInfo, err = EncodePartnerLogicalUnitInfo(*in.PartnerLogicalUnitInfo)
	} else {
		info := defaultExtendedInfo()
		if in.ExtendedInfo != nil {
			info = *in.ExtendedInfo
		}
		operationInfo, err = EncodeExtendedInfo(info)
	}
	if err != nil {
		return nil, err
	}
	return encodeRecord(in.FunctionCode, in.RecordHeaderInput, operationInfo, len(parameters), [][]byte{parameters}, "APPC control record")
}

// DataRecordInput is the semantic input to one F_SAP_SEND/F_RECEIVE data record.
type DataRecordInput struct {
	RecordHeaderInput
	FunctionCode       *Function
	Data               []byte
	CommunicationIndex uint16
	ConnectionIndex    uint16
	IsFinal            *bool
}

// EncodeDataRecord encodes one data record with proven CPIC defaults.
func EncodeDataRecord(in DataRecordInput) ([]byte, error) {
	data := append([]byte(nil), in.Data...)
	isFinal := in.IsFinal == nil || *in.IsFinal
	functionCode := FuncSapSend
	if in.FunctionCode != nil {
		functionCode = *in.FunctionCode
	}
	operationInfo, err := EncodeExtendedInfo(ExtendedInfo{
		CommunicationIndex: in.CommunicationIndex,
		ConnectionIndex:    in.ConnectionIndex,
	})
	if err != nil {
		return nil, err
	}

	header := in.RecordHeaderInput
	if header.Info == nil {
		v := uint8(1)
		if isFinal {
			v = 5
		}
		header.Info = &v
	}
	if header.Vector == nil {
		var v uint8
		switch {
		case isFinal:
			v = 0x0c
		case functionCode == FuncSapSend:
			v = 0x08
		default:
			v = 0
		}
		header.Vector = &v
	}
	sapParameterLength := 0
	if isFinal {
		sapParameterLength = 8
	}
	return encodeRecord(functionCode, header, operationInfo, sapParameterLength, [][]byte{data}, "APPC data record")
}

// encodeRecord assembles the 80-byte record header plus trailing data parts.
func encodeRecord(functionCode Function, in RecordHeaderInput, operationInfo []byte, sapParameterLength int, trailing [][]byte, context string) ([]byte, error) {
	conversationID := append([]byte(nil), in.ConversationID...)
	if in.ConversationID == nil {
		conversationID = make([]byte, 8)
	}
	if len(conversationID) != 8 {
		return nil, fmt.Errorf("%w: conversationId must contain exactly 8 bytes; received %d", ErrRange, len(conversationID))
	}
	encodedOperationInfo, err := exactBytes(operationInfo, 32, "operationInfo")
	if err != nil {
		return nil, err
	}
	if sapParameterLength < 0 || sapParameterLength > 0xffff {
		return nil, fmt.Errorf("%w: sapParameterLength must be an integer in 0..65535", ErrRange)
	}
	trailingDataLength := 0
	for _, part := range trailing {
		trailingDataLength += len(part)
	}

	w, err := wire.NewWriter(RecordHeaderLength+trailingDataLength, context)
	if err != nil {
		return nil, err
	}
	var werr error
	put := func(f func() error) {
		if werr == nil {
			werr = f()
		}
	}
	put(func() error { return w.WriteUint8(ProtocolVersion, "protocolVersion") })
	put(func() error { return w.WriteUint8(uint8(functionCode), "functionCode") })
	put(func() error { return w.WriteUint8(u8or(in.Protocol, 2), "protocol") })
	put(func() error { return w.WriteUint8(u8or(in.Mode, 0), "mode") })
	put(func() error { return w.WriteUint16BE(u16or(in.UID, 0xffff), "uid") })
	put(func() error { return w.WriteUint16BE(u16or(in.GatewayID, 0), "gatewayId") })
	put(func() error { return w.WriteUint16BE(u16or(in.ErrorLength, 0), "errorLength") })
	put(func() error { return w.WriteUint8(u8or(in.Info2, 0), "info2") })
	put(func() error { return w.WriteUint8(u8or(in.TraceLevel, 0), "traceLevel") })
	put(func() error { return w.WriteUint32BE(u32or(in.Time, 0), "time") })
	put(func() error { return w.WriteUint8(u8or(in.Info3, 0), "info3") })
	put(func() error { return w.WriteInt32BE(i32or(in.Timeout, 0), "timeout") })
	put(func() error { return w.WriteUint8(u8or(in.Info4, 0), "info4") })
	put(func() error { return w.WriteUint32BE(u32or(in.SequenceNumber, 0), "sequenceNumber") })
	put(func() error { return w.WriteUint16BE(uint16(sapParameterLength), "sapParameterLength") })
	put(func() error { return w.WriteUint16BE(u16or(in.Padding, 0), "padding") })
	put(func() error { return w.WriteUint8(u8or(in.Info, 0), "info") })
	put(func() error { return w.WriteUint8(u8or(in.Vector, 0), "vector") })
	put(func() error { return w.WriteUint32BE(u32or(in.AppcReturnCode, 0), "appcReturnCode") })
	put(func() error { return w.WriteUint32BE(u32or(in.SapReturnCode, 0), "sapReturnCode") })
	put(func() error { return w.WriteBytes(conversationID, "conversationId") })
	put(func() error { return w.WriteBytes(encodedOperationInfo, "operationInfo") })
	for i, part := range trailing {
		part := part
		i := i
		put(func() error { return w.WriteBytes(part, fmt.Sprintf("trailingData[%d]", i)) })
	}
	if werr != nil {
		return nil, werr
	}
	return w.Finish()
}
