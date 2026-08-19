// SPDX-License-Identifier: Apache-2.0
//
// Part of the appc package; see appc.go for the file's provenance header.

package appc

import (
	"encoding/binary"
	"fmt"

	"github.com/oisee/open-rfc-go/internal/wire"
)

// AsyncDataInfo is decoded 32-byte F_ASEND_DATA operation information.
type AsyncDataInfo = DataOperationInfo

// DataOperationInfo is the numeric operation information client streaming
// records emit.
type DataOperationInfo struct {
	DataLength         uint16
	CommunicationIndex uint16
	ConnectionIndex    uint16
}

// IncomingDataOperationInfo is server-side data-buffer operation information.
type IncomingDataOperationInfo struct {
	DataLength         uint16
	CommunicationIndex uint16
	ConnectionIndex    uint16
}

func encodeAsyncDataInfo(dataLength int, communicationIndex, connectionIndex uint16) ([]byte, error) {
	if dataLength < 1 || dataLength > MaxApplicationDataFragmentLength {
		return nil, fmt.Errorf("%w: async APPC data length must be an integer in 1..%d", ErrRange, MaxApplicationDataFragmentLength)
	}
	w, _ := wire.NewWriter(32, "APPC async-send info")
	var err error
	put := func(f func() error) {
		if err == nil {
			err = f()
		}
	}
	put(func() error { return w.WriteUint16BE(0, "reserved") })
	put(func() error { return w.WriteUint16BE(uint16(dataLength), "dataLength") })
	put(func() error { return w.WriteBytes(make([]byte, 24), "reserved2") })
	put(func() error { return w.WriteUint16BE(communicationIndex, "communicationIndex") })
	put(func() error { return w.WriteUint16BE(connectionIndex, "connectionIndex") })
	if err != nil {
		return nil, err
	}
	return w.Finish()
}

// DecodeDataOperationInfo decodes and validates fixed F_ASEND_DATA operation
// information.
func DecodeDataOperationInfo(data []byte) (DataOperationInfo, error) {
	var zero DataOperationInfo
	if len(data) != 32 {
		return zero, fmt.Errorf("%w: APPC data operation info must contain exactly 32 bytes; received %d", ErrRange, len(data))
	}
	r := wire.NewReader(data, "APPC data operation info")
	reserved, err := r.ReadUint16BE("reserved")
	if err != nil {
		return zero, err
	}
	if reserved != 0 {
		return zero, fmt.Errorf("%w: APPC data operation reserved word must be zero", ErrRange)
	}
	dataLength, err := r.ReadUint16BE("dataLength")
	if err != nil {
		return zero, err
	}
	reserved2, err := r.ReadBytes(24, "reserved2")
	if err != nil {
		return zero, err
	}
	for _, b := range reserved2 {
		if b != 0 {
			return zero, fmt.Errorf("%w: APPC data operation reserved bytes must be zero", ErrRange)
		}
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
	return DataOperationInfo{DataLength: dataLength, CommunicationIndex: commIdx, ConnectionIndex: connIdx}, nil
}

// DecodeIncomingDataOperationInfo decodes server-side data-buffer info. Unlike
// client F_ASEND_DATA, the reply's actual byte count is at offset 10; the word
// at offset 2 is a receive-buffer capacity and must not frame the payload.
func DecodeIncomingDataOperationInfo(data []byte) (IncomingDataOperationInfo, error) {
	if len(data) != 32 {
		return IncomingDataOperationInfo{}, fmt.Errorf("%w: incoming APPC data operation info must contain exactly 32 bytes; received %d", ErrRange, len(data))
	}
	return IncomingDataOperationInfo{
		DataLength:         binary.BigEndian.Uint16(data[10:]),
		CommunicationIndex: binary.BigEndian.Uint16(data[28:]),
		ConnectionIndex:    binary.BigEndian.Uint16(data[30:]),
	}, nil
}

// DecodeAsyncDataInfo decodes F_ASEND_DATA info and enforces the 1..28000 slice.
func DecodeAsyncDataInfo(data []byte) (AsyncDataInfo, error) {
	decoded, err := DecodeDataOperationInfo(data)
	if err != nil {
		return AsyncDataInfo{}, err
	}
	if decoded.DataLength < 1 || decoded.DataLength > MaxApplicationDataFragmentLength {
		return AsyncDataInfo{}, fmt.Errorf("%w: APPC async-send data length must be in 1..%d", ErrRange, MaxApplicationDataFragmentLength)
	}
	return decoded, nil
}

func outgoingOperationInfo(functionCode Function, dataLength int, communicationIndex, connectionIndex uint16) ([]byte, error) {
	if functionCode == FuncAsyncSendData || functionCode == FuncSendData {
		return encodeAsyncDataInfo(dataLength, communicationIndex, connectionIndex)
	}
	return EncodeExtendedInfo(ExtendedInfo{CommunicationIndex: communicationIndex, ConnectionIndex: connectionIndex})
}

func validateFinalSapParameters(parameters []byte, applicationDataLength int) error {
	if len(parameters) != FinalSapParameterLength {
		return fmt.Errorf("%w: finalSapParameters must contain exactly %d bytes; received %d", ErrRange, FinalSapParameterLength, len(parameters))
	}
	if binary.BigEndian.Uint16(parameters[0:]) != 0 {
		return fmt.Errorf("%w: finalSapParameters reserved field must be zero", ErrRange)
	}
	declaredPacketLength := int(binary.BigEndian.Uint16(parameters[2:]))
	if declaredPacketLength != applicationDataLength {
		return fmt.Errorf("%w: finalSapParameters declare %d application bytes; received %d", ErrRange, declaredPacketLength, applicationDataLength)
	}
	return nil
}

type fragmentSemantics struct {
	functionCode       Function
	isFinal            bool
	info               uint8
	vector             uint8
	sapParameterLength int
}

func outgoingFragmentSemantics(fragmentIndex, fragmentCount int) fragmentSemantics {
	isSingle := fragmentCount == 1
	isFinal := isSingle || fragmentIndex == fragmentCount-1
	isSyncStreaming := !isSingle && !isFinal &&
		fragmentIndex >= MaxAsyncSendsBeforeSync &&
		(fragmentIndex-MaxAsyncSendsBeforeSync)%MaxAsyncSendsBeforeSync == 0

	fc := FuncAsyncSendData
	switch {
	case isSingle:
		fc = FuncSapSend
	case isFinal:
		fc = FuncReceive
	case isSyncStreaming:
		fc = FuncSendData
	}
	var info uint8
	switch {
	case isSingle:
		info = 5
	case isFinal || isSyncStreaming:
		info = 1
	}
	var vector uint8
	if isSingle {
		vector = 0x0c
	}
	sap := 0
	if isSingle {
		sap = 8
	}
	return fragmentSemantics{functionCode: fc, isFinal: isFinal, info: info, vector: vector, sapParameterLength: sap}
}

// OutgoingDataPlanInput describes one logical outgoing CPIC message to plan.
type OutgoingDataPlanInput struct {
	RecordHeaderInput
	// ApplicationData is the CPIC bytes before the compact SAP tail, or the
	// complete streamed packet.
	ApplicationData []byte
	// FinalSapParameters is present (non-nil) only for compact CPIC packets.
	FinalSapParameters []byte
	CommunicationIndex uint16
	ConnectionIndex    uint16
}

// OutgoingDataPlannerOptions tunes the outgoing planner. A nil field takes the
// documented default.
type OutgoingDataPlannerOptions struct {
	MaxApplicationDataLength *int
	MaxFragments             *int
	CpicStreaming            CpicStreamingPolicy
}

// OutgoingDataFragment is one immutable semantic step in an outgoing plan.
// Header carries passthrough common-header overrides; the concrete Info,
// Vector, SequenceNumber, and ConversationID below take precedence over any set
// on Header.
type OutgoingDataFragment struct {
	Header                       RecordHeaderInput
	FunctionCode                 Function
	FragmentIndex                int
	FragmentCount                int
	ConversationID               []byte
	SequenceNumber               uint32
	ApplicationData              []byte
	FinalSapParameters           []byte
	MessageApplicationDataLength int
	CommunicationIndex           uint16
	ConnectionIndex              uint16
	IsFinal                      bool
	Info                         uint8
	Vector                       uint8
	SapParameterLength           int
}

func (f OutgoingDataFragment) headerInput() RecordHeaderInput {
	hdr := f.Header
	hdr.ConversationID = f.ConversationID
	seq := f.SequenceNumber
	info := f.Info
	vector := f.Vector
	hdr.SequenceNumber = &seq
	hdr.Info = &info
	hdr.Vector = &vector
	return hdr
}

// PlanOutgoingDataFragments plans one logical outgoing CPIC message. Compact
// messages use one F_SAP_SEND. Larger messages use bounded F_ASEND_DATA slices
// followed by the empty F_RECEIVE terminator the admitted STSEND path requires.
func PlanOutgoingDataFragments(input OutgoingDataPlanInput, options OutgoingDataPlannerOptions) ([]OutgoingDataFragment, error) {
	maxApplicationDataLength := DefaultMaxOutgoingMessageLength
	if options.MaxApplicationDataLength != nil {
		maxApplicationDataLength = *options.MaxApplicationDataLength
	}
	maxFragments := DefaultMaxMessageFragments
	if options.MaxFragments != nil {
		maxFragments = *options.MaxFragments
	}
	cpicStreaming := options.CpicStreaming
	if cpicStreaming == "" {
		cpicStreaming = StreamingDisabled
	}

	if maxApplicationDataLength < 0 || maxApplicationDataLength > MaxOutgoingMessageLength {
		return nil, fmt.Errorf("%w: maxApplicationDataLength must be an integer in 0..%d", ErrRange, MaxOutgoingMessageLength)
	}
	if maxFragments < 1 {
		return nil, fmt.Errorf("%w: maxFragments must be a positive integer", ErrRange)
	}
	if cpicStreaming != StreamingDisabled && cpicStreaming != StreamingEnabled {
		return nil, fmt.Errorf("%w: cpicStreaming must be disabled or enabled", ErrRange)
	}

	msgLen := len(input.ApplicationData)
	if msgLen > maxApplicationDataLength {
		return nil, fmt.Errorf("%w: CPIC application data length %d exceeds configured limit %d", ErrRange, msgLen, maxApplicationDataLength)
	}

	var finalSapParameters []byte
	hasFinalSap := input.FinalSapParameters != nil
	if hasFinalSap {
		if len(input.FinalSapParameters) != FinalSapParameterLength {
			return nil, fmt.Errorf("%w: finalSapParameters must contain exactly %d bytes; received %d", ErrRange, FinalSapParameterLength, len(input.FinalSapParameters))
		}
		finalSapParameters = append([]byte(nil), input.FinalSapParameters...)
		if err := validateFinalSapParameters(finalSapParameters, msgLen); err != nil {
			return nil, err
		}
		if msgLen > MaxApplicationDataFragmentLength {
			return nil, fmt.Errorf("%w: compact CPIC application data cannot exceed 28000 bytes", ErrRange)
		}
	} else if msgLen <= MaxApplicationDataFragmentLength {
		return nil, fmt.Errorf("%w: a streamed CPIC packet without final SAP parameters must exceed 28000 bytes", ErrRange)
	}

	useSingleRecord := hasFinalSap && msgLen <= MaxApplicationDataFragmentLength
	if !useSingleRecord && cpicStreaming != StreamingEnabled {
		return nil, fmt.Errorf("%w: CPIC streaming is disabled; enable this destination before sending more than 28000 application bytes", ErrState)
	}
	dataFragmentCount := 1
	if !useSingleRecord {
		dataFragmentCount = (msgLen + MaxApplicationDataFragmentLength - 1) / MaxApplicationDataFragmentLength
	}
	fragmentCount := 1
	if !useSingleRecord {
		fragmentCount = dataFragmentCount + 1
	}
	if fragmentCount > maxFragments {
		return nil, fmt.Errorf("%w: APPC fragment count %d exceeds configured limit %d", ErrRange, fragmentCount, maxFragments)
	}

	conversationID := append([]byte(nil), input.ConversationID...)
	if input.ConversationID == nil {
		conversationID = make([]byte, 8)
	}
	if len(conversationID) != 8 {
		return nil, fmt.Errorf("%w: conversationId must contain exactly 8 bytes; received %d", ErrRange, len(conversationID))
	}

	// Exercise the authoritative encoder once so every header/index bound
	// fails before the first transport write.
	preflightFunc := FuncSapSend
	preflightDataLen := msgLen + len(finalSapParameters)
	preflightSap := FinalSapParameterLength
	if !useSingleRecord {
		preflightFunc = FuncAsyncSendData
		preflightDataLen = msgLen
		if preflightDataLen > MaxApplicationDataFragmentLength {
			preflightDataLen = MaxApplicationDataFragmentLength
		}
		preflightSap = 0
	}
	preflightOpInfo, err := outgoingOperationInfo(preflightFunc, preflightDataLen, input.CommunicationIndex, input.ConnectionIndex)
	if err != nil {
		return nil, err
	}
	if _, err := encodeRecord(preflightFunc, input.RecordHeaderInput, preflightOpInfo, preflightSap, [][]byte{{}}, "outgoing APPC plan"); err != nil {
		return nil, err
	}

	fragments := make([]OutgoingDataFragment, 0, fragmentCount)
	for i := 0; i < fragmentCount; i++ {
		sem := outgoingFragmentSemantics(i, fragmentCount)
		// Clamp start to the message length: the F_RECEIVE terminator's index
		// runs one past the data fragments, so its start would otherwise exceed
		// the slice. Upstream relies on Buffer.subarray clamping; Go slicing
		// panics, so the clamp is explicit here.
		start := i * MaxApplicationDataFragmentLength
		if start > msgLen {
			start = msgLen
		}
		end := start
		if sem.functionCode != FuncReceive {
			end = start + MaxApplicationDataFragmentLength
			if end > msgLen {
				end = msgLen
			}
		}
		var fragFinalSap []byte
		if useSingleRecord {
			fragFinalSap = append([]byte(nil), finalSapParameters...)
		}
		fragments = append(fragments, OutgoingDataFragment{
			Header:                       input.RecordHeaderInput,
			FunctionCode:                 sem.functionCode,
			FragmentIndex:                i,
			FragmentCount:                fragmentCount,
			ConversationID:               conversationID,
			SequenceNumber:               u32or(input.SequenceNumber, 0),
			ApplicationData:              append([]byte(nil), input.ApplicationData[start:end]...),
			FinalSapParameters:           fragFinalSap,
			MessageApplicationDataLength: msgLen,
			CommunicationIndex:           input.CommunicationIndex,
			ConnectionIndex:              input.ConnectionIndex,
			IsFinal:                      sem.isFinal,
			Info:                         sem.info,
			Vector:                       sem.vector,
			SapParameterLength:           sem.sapParameterLength,
		})
	}
	return fragments, nil
}

func invalidOutgoingFragment(reason string) error {
	return fmt.Errorf("%w: invalid outgoing APPC fragment: %s", ErrRange, reason)
}

// SnapshotOutgoingDataFragment copies an externally supplied plan step and
// applies the geometry bounds before it is validated or encoded.
func SnapshotOutgoingDataFragment(in OutgoingDataFragment) (OutgoingDataFragment, error) {
	if len(in.ConversationID) != 8 {
		return OutgoingDataFragment{}, invalidOutgoingFragment("conversationId must contain exactly 8 bytes")
	}
	if len(in.ApplicationData) > MaxApplicationDataFragmentLength {
		return OutgoingDataFragment{}, invalidOutgoingFragment("applicationData exceeds the 28000-byte slice bound")
	}
	if len(in.FinalSapParameters) > FinalSapParameterLength {
		return OutgoingDataFragment{}, invalidOutgoingFragment("finalSapParameters exceeds 8 bytes")
	}
	out := in
	out.ConversationID = append([]byte(nil), in.ConversationID...)
	out.ApplicationData = append([]byte(nil), in.ApplicationData...)
	out.FinalSapParameters = append([]byte(nil), in.FinalSapParameters...)
	return out, nil
}

// EncodeOutgoingDataFragment encodes one validated semantic step from the planner.
func EncodeOutgoingDataFragment(in OutgoingDataFragment) ([]byte, error) {
	fragment, err := SnapshotOutgoingDataFragment(in)
	if err != nil {
		return nil, err
	}
	if fragment.FragmentCount < 1 {
		return nil, invalidOutgoingFragment("fragmentCount must be a positive integer")
	}
	if fragment.FragmentIndex < 0 || fragment.FragmentIndex >= fragment.FragmentCount {
		return nil, invalidOutgoingFragment("fragmentIndex must identify a fragment in the plan")
	}
	if fragment.MessageApplicationDataLength < 0 || fragment.MessageApplicationDataLength > MaxOutgoingMessageLength {
		return nil, invalidOutgoingFragment("messageApplicationDataLength is outside the proven range")
	}

	expected := outgoingFragmentSemantics(fragment.FragmentIndex, fragment.FragmentCount)
	if fragment.FunctionCode != expected.functionCode ||
		fragment.IsFinal != expected.isFinal ||
		fragment.Info != expected.info ||
		fragment.Vector != expected.vector ||
		fragment.SapParameterLength != expected.sapParameterLength {
		return nil, invalidOutgoingFragment("function, final marker, info, vector, or parameter length is inconsistent")
	}

	switch {
	case fragment.FragmentCount == 1:
		if len(fragment.ApplicationData) > MaxApplicationDataFragmentLength {
			return nil, invalidOutgoingFragment("compact F_SAP_SEND data length is invalid")
		}
		if err := validateFinalSapParameters(fragment.FinalSapParameters, fragment.MessageApplicationDataLength); err != nil {
			return nil, invalidOutgoingFragment(err.Error())
		}
	case fragment.FunctionCode == FuncAsyncSendData || fragment.FunctionCode == FuncSendData:
		if len(fragment.ApplicationData) < 1 || len(fragment.ApplicationData) > MaxApplicationDataFragmentLength {
			return nil, invalidOutgoingFragment(fmt.Sprintf("%s slice length is invalid", FunctionName(fragment.FunctionCode)))
		}
		if len(fragment.FinalSapParameters) != 0 {
			return nil, invalidOutgoingFragment(fmt.Sprintf("%s cannot carry SAP parameters", FunctionName(fragment.FunctionCode)))
		}
	default:
		if len(fragment.ApplicationData) != 0 || len(fragment.FinalSapParameters) != 0 {
			return nil, invalidOutgoingFragment("the async F_RECEIVE terminator must be empty")
		}
	}

	var operationInfo []byte
	if fragment.FunctionCode == FuncReceive && fragment.FragmentCount > 1 {
		operationInfo, err = encodeAsyncDataInfo(MaxApplicationDataFragmentLength, fragment.CommunicationIndex, fragment.ConnectionIndex)
	} else {
		operationInfo, err = outgoingOperationInfo(fragment.FunctionCode, len(fragment.ApplicationData), fragment.CommunicationIndex, fragment.ConnectionIndex)
	}
	if err != nil {
		return nil, err
	}
	return encodeRecord(fragment.FunctionCode, fragment.headerInput(), operationInfo, fragment.SapParameterLength,
		[][]byte{fragment.ApplicationData, fragment.FinalSapParameters}, "outgoing APPC data fragment")
}
