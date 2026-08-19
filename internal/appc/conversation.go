// SPDX-License-Identifier: Apache-2.0
//
// Part of the appc package; see appc.go for the file's provenance header.

package appc

import (
	"bytes"
	"fmt"
)

// DataFragment is one decoded F_SAP_SEND/F_RECEIVE application record.
type DataFragment struct {
	Header  Header
	Data    []byte
	IsFinal bool
}

// DecodeDataFragment decodes one F_SAP_SEND/F_RECEIVE data record.
func DecodeDataFragment(payload []byte) (DataFragment, error) {
	if len(payload) < RecordHeaderLength {
		return DataFragment{}, fmt.Errorf("%w: an APPC data record needs %d bytes; received %d", ErrRange, RecordHeaderLength, len(payload))
	}
	header, err := DecodeHeader(payload)
	if err != nil {
		return DataFragment{}, err
	}
	if header.FunctionCode != FuncSapSend && header.FunctionCode != FuncReceive {
		return DataFragment{}, fmt.Errorf("%w: %s is not an APPC RFC data fragment", ErrProtocol, header.FunctionName)
	}
	return DataFragment{
		Header:  header,
		Data:    append([]byte(nil), payload[RecordHeaderLength:]...),
		IsFinal: header.Vector&VectorEndOfMessage != 0,
	}, nil
}

// SynchronousSendAcknowledgement is the empty F_SEND_DATA flow-control reply.
type SynchronousSendAcknowledgement struct {
	Header          Header
	ConnectionIndex uint16
}

// DecodeSynchronousSendAcknowledgement decodes the empty, canonical F_SEND_DATA
// acknowledgement.
func DecodeSynchronousSendAcknowledgement(payload []byte) (SynchronousSendAcknowledgement, error) {
	var zero SynchronousSendAcknowledgement
	if len(payload) != RecordHeaderLength {
		return zero, fmt.Errorf("%w: APPC synchronous-send acknowledgement must contain exactly %d bytes; received %d", ErrRange, RecordHeaderLength, len(payload))
	}
	header, err := DecodeHeader(payload)
	if err != nil {
		return zero, err
	}
	if header.FunctionCode != FuncSendData ||
		header.Protocol != 2 ||
		header.Mode != 0 ||
		header.UID != 0xffff ||
		header.GatewayID != 0 ||
		header.ErrorLength != 0 ||
		header.Info2 != 0 ||
		header.TraceLevel != 0 ||
		header.Time != 0 ||
		header.Info3 != 0 ||
		header.Timeout != 0 ||
		header.Info4 != 2 ||
		header.SequenceNumber != 0 ||
		header.Info != 1 ||
		header.Vector != 0 ||
		header.SapParameterLen != 0 ||
		header.Padding != 0 ||
		header.AppcReturnCode != 0 ||
		header.SapReturnCode != 0 {
		return zero, fmt.Errorf("%w: APPC synchronous-send acknowledgement header is not canonical", ErrProtocol)
	}
	encodedOperationInfo := payload[CommonHeaderLength:RecordHeaderLength]
	operationInfo, err := DecodeIncomingDataOperationInfo(encodedOperationInfo)
	if err != nil {
		return zero, err
	}
	for _, b := range encodedOperationInfo[:30] {
		if b != 0 {
			return zero, fmt.Errorf("%w: APPC synchronous-send acknowledgement operation information is not canonical", ErrProtocol)
		}
	}
	if operationInfo.DataLength != 0 || operationInfo.CommunicationIndex != 0 {
		return zero, fmt.Errorf("%w: APPC synchronous-send acknowledgement operation information is not canonical", ErrProtocol)
	}
	return SynchronousSendAcknowledgement{Header: header, ConnectionIndex: operationInfo.ConnectionIndex}, nil
}

// ClientSetupState is the state of the admitted direct-CPIC setup sequence.
type ClientSetupState string

const (
	StateNew                ClientSetupState = "new"
	StateInitializePending  ClientSetupState = "initialize-pending"
	StateInitialized        ClientSetupState = "initialized"
	StateTpSet              ClientSetupState = "tp-set"
	StatePartnerSet         ClientSetupState = "partner-set"
	StateAllocatePending    ClientSetupState = "allocate-pending"
	StateSendContinuation   ClientSetupState = "send-continuation"
	StateSendBarrierPending ClientSetupState = "send-barrier-pending"
	StateResponsePending    ClientSetupState = "response-pending"
	StateReady              ClientSetupState = "ready"
	StateClosed             ClientSetupState = "closed"
)

// ReceiveDisposition names what a received reply means to the setup machine.
type ReceiveDisposition string

const (
	DispositionAccepted           ReceiveDisposition = "accepted"
	DispositionNormalDeallocation ReceiveDisposition = "normal-deallocation"
)

// ClientSetupStateMachine validates the admitted client-side setup sequence.
type ClientSetupStateMachine struct {
	state ClientSetupState
}

// NewClientSetupStateMachine returns a machine in the "new" state.
func NewClientSetupStateMachine() *ClientSetupStateMachine {
	return &ClientSetupStateMachine{state: StateNew}
}

// State reports the current setup state.
func (m *ClientSetupStateMachine) State() ClientSetupState { return m.state }

// ResponseComplete marks a pending response as fully received.
func (m *ClientSetupStateMachine) ResponseComplete() error {
	if m.state != StateResponsePending {
		m.state = StateClosed
		return fmt.Errorf("%w: cannot complete an APPC response unless one is pending", ErrState)
	}
	m.state = StateReady
	return nil
}

// Sent advances the machine for a record about to be written.
func (m *ClientSetupStateMachine) Sent(functionCode Function, isFinalDataRecord bool) error {
	if functionCode == FuncSapSend && !isFinalDataRecord {
		return fmt.Errorf("%w: F_SAP_SEND cannot start a streamed outgoing message", ErrState)
	}
	if functionCode == FuncAsyncSendData && isFinalDataRecord {
		return fmt.Errorf("%w: F_ASEND_DATA must be followed by F_RECEIVE", ErrState)
	}
	if functionCode == FuncSendData && isFinalDataRecord {
		return fmt.Errorf("%w: streaming F_SEND_DATA must be followed by its acknowledgement", ErrState)
	}
	if functionCode == FuncReceive && m.state == StateSendContinuation && !isFinalDataRecord {
		return fmt.Errorf("%w: the streamed outgoing F_RECEIVE terminator must be final", ErrState)
	}
	allowed := (m.state == StateNew && functionCode == FuncInitialize) ||
		(m.state == StateInitialized && (functionCode == FuncSetTpName || functionCode == FuncSetPartnerLuName)) ||
		(m.state == StateTpSet && functionCode == FuncSetPartnerLuName) ||
		(m.state == StatePartnerSet && functionCode == FuncAllocate) ||
		(m.state == StateReady && (functionCode == FuncSapSend || functionCode == FuncAsyncSendData || functionCode == FuncDeallocate)) ||
		(m.state == StateSendContinuation && (functionCode == FuncAsyncSendData || functionCode == FuncSendData || functionCode == FuncReceive))
	if !allowed {
		return fmt.Errorf("%w: cannot send %s while APPC client is %s", ErrState, FunctionName(functionCode), m.state)
	}

	switch functionCode {
	case FuncInitialize:
		m.state = StateInitializePending
	case FuncSetTpName:
		m.state = StateTpSet
	case FuncSetPartnerLuName:
		m.state = StatePartnerSet
	case FuncAllocate:
		m.state = StateAllocatePending
	case FuncAsyncSendData:
		m.state = StateSendContinuation
	case FuncSapSend:
		m.state = StateResponsePending
	case FuncSendData:
		m.state = StateSendBarrierPending
	case FuncReceive:
		if isFinalDataRecord {
			m.state = StateResponsePending
		}
	case FuncDeallocate:
		m.state = StateClosed
	}
	return nil
}

// Received advances the machine for a decoded reply.
func (m *ClientSetupStateMachine) Received(payload []byte) (ReceiveDisposition, error) {
	if len(payload) < RecordHeaderLength {
		m.state = StateClosed
		return "", fmt.Errorf("%w: an APPC reply needs %d bytes; received %d", ErrRange, RecordHeaderLength, len(payload))
	}
	header, err := DecodeHeader(payload)
	if err != nil {
		m.state = StateClosed
		return "", err
	}
	normalDeallocation := header.AppcReturnCode == returnCodeDeallocatedNormal &&
		header.SapReturnCode == 0 &&
		m.state == StateResponsePending &&
		(header.FunctionCode == FuncSapSend || header.FunctionCode == FuncReceive)
	if normalDeallocation {
		// A remote ABAP MESSAGE/runtime failure can terminate CPI-C while the
		// same final data record still carries the RFC error envelope. Publish
		// terminal state first, then let the CPIC layer decode that payload.
		m.state = StateClosed
		return DispositionNormalDeallocation, nil
	}
	if header.AppcReturnCode != 0 || header.SapReturnCode != 0 {
		m.state = StateClosed
		return "", &PeerReturnCodeError{FunctionName: header.FunctionName, AppcReturnCode: header.AppcReturnCode, SapReturnCode: header.SapReturnCode}
	}

	allowed := (m.state == StateInitializePending && header.FunctionCode == FuncInitialize) ||
		(m.state == StateAllocatePending && header.FunctionCode == FuncAllocate) ||
		(m.state == StateSendBarrierPending && header.FunctionCode == FuncSendData) ||
		(m.state == StateResponsePending && (header.FunctionCode == FuncSapSend || header.FunctionCode == FuncReceive))
	if !allowed {
		m.state = StateClosed
		return "", fmt.Errorf("%w: cannot receive %s while APPC client is %s", ErrState, header.FunctionName, m.state)
	}

	switch m.state {
	case StateInitializePending:
		m.state = StateInitialized
	case StateAllocatePending:
		m.state = StateReady
	case StateSendBarrierPending:
		m.state = StateSendContinuation
	}
	return DispositionAccepted, nil
}

// Message is one reassembled RFC application message.
type Message struct {
	Data               []byte
	ConversationID     []byte
	SequenceNumber     uint32
	FragmentCount      int
	CommunicationIndex uint16
	ConnectionIndex    uint16
}

// ConversationDecoderOptions tunes the reassembler. A nil int field takes the
// documented default.
type ConversationDecoderOptions struct {
	MaxMessageLength                  *int
	MaxFragments                      *int
	AllowInitialReceive               bool
	ValidateIncomingDataOperationInfo bool
}

type pendingMessage struct {
	chunks             [][]byte
	conversationID     []byte
	sequenceNumber     uint32
	byteLength         int
	fragmentCount      int
	communicationIndex uint16
	connectionIndex    uint16
}

// ConversationDecoder reassembles RFC application messages across
// F_SAP_SEND/F_RECEIVE records.
type ConversationDecoder struct {
	maxMessageLength                  int
	maxFragments                      int
	allowInitialReceive               bool
	validateIncomingDataOperationInfo bool
	pending                           *pendingMessage
}

// NewConversationDecoder returns a reassembler configured by options.
func NewConversationDecoder(options ConversationDecoderOptions) (*ConversationDecoder, error) {
	maxMessageLength := DefaultMaxMessageLength
	if options.MaxMessageLength != nil {
		maxMessageLength = *options.MaxMessageLength
	}
	maxFragments := DefaultMaxMessageFragments
	if options.MaxFragments != nil {
		maxFragments = *options.MaxFragments
	}
	if maxMessageLength < 0 {
		return nil, fmt.Errorf("%w: maxMessageLength must be a non-negative integer", ErrRange)
	}
	if maxFragments < 1 {
		return nil, fmt.Errorf("%w: maxFragments must be a positive integer", ErrRange)
	}
	return &ConversationDecoder{
		maxMessageLength:                  maxMessageLength,
		maxFragments:                      maxFragments,
		allowInitialReceive:               options.AllowInitialReceive,
		validateIncomingDataOperationInfo: options.ValidateIncomingDataOperationInfo,
	}, nil
}

// BufferedByteLength reports how many bytes are held for the pending message.
func (d *ConversationDecoder) BufferedByteLength() int {
	if d.pending == nil {
		return 0
	}
	return d.pending.byteLength
}

// FragmentCount reports how many fragments are held for the pending message.
func (d *ConversationDecoder) FragmentCount() int {
	if d.pending == nil {
		return 0
	}
	return d.pending.fragmentCount
}

// Push decodes an ordinary data record and returns any completed messages.
func (d *ConversationDecoder) Push(payload []byte) ([]Message, error) {
	return d.push(payload, false)
}

// PushTerminalDeallocation decodes the data returned with CM_DEALLOCATED_NORMAL
// (SAP Note 63347 return code 18), the admitted terminal conversation outcome.
func (d *ConversationDecoder) PushTerminalDeallocation(payload []byte) ([]Message, error) {
	return d.push(payload, true)
}

func (d *ConversationDecoder) push(payload []byte, terminalDeallocation bool) ([]Message, error) {
	header, err := DecodeHeader(payload)
	if err != nil {
		return nil, err
	}
	if terminalDeallocation {
		if header.AppcReturnCode != returnCodeDeallocatedNormal || header.SapReturnCode != 0 {
			return nil, fmt.Errorf("%w: terminal APPC deallocation requires APPC return code 18 and SAP return code 0", ErrProtocol)
		}
	} else if header.AppcReturnCode == returnCodeDeallocatedNormal && header.SapReturnCode == 0 {
		return nil, fmt.Errorf("%w: normal deallocation must use the terminal decoder", ErrProtocol)
	} else if header.AppcReturnCode != 0 || header.SapReturnCode != 0 {
		return nil, fmt.Errorf("%w: %s cannot be decoded after APPC return code %d and SAP return code %d", ErrProtocol, header.FunctionName, header.AppcReturnCode, header.SapReturnCode)
	}

	isData := header.FunctionCode == FuncSapSend || header.FunctionCode == FuncReceive
	if !isData {
		if d.pending != nil {
			return nil, fmt.Errorf("%w: %s interrupted a fragmented message before its final APPC record", ErrProtocol, header.FunctionName)
		}
		if terminalDeallocation {
			return nil, ErrNormalDeallocationWithoutData
		}
		return []Message{}, nil
	}

	if len(payload) >= RecordHeaderLength {
		incomingDataLength := len(payload) - RecordHeaderLength
		pendingByteLength := 0
		pendingFragmentCount := 0
		if d.pending != nil {
			pendingByteLength = d.pending.byteLength
			pendingFragmentCount = d.pending.fragmentCount
		}
		// Reject an over-budget continuation before we own a copy of the
		// peer-controlled application bytes.
		if err := d.checkLimits(pendingByteLength+incomingDataLength, pendingFragmentCount+1); err != nil {
			return nil, err
		}
	}
	fragment, err := DecodeDataFragment(payload)
	if err != nil {
		return nil, err
	}
	operationInfo, err := DecodeIncomingDataOperationInfo(payload[CommonHeaderLength:RecordHeaderLength])
	if err != nil {
		return nil, err
	}
	if d.validateIncomingDataOperationInfo && int(operationInfo.DataLength) != len(fragment.Data) {
		return nil, fmt.Errorf("%w: incoming APPC data length %d does not match record payload length %d", ErrProtocol, operationInfo.DataLength, len(fragment.Data))
	}

	if terminalDeallocation {
		if len(fragment.Data) == 0 {
			return nil, ErrNormalDeallocationWithoutData
		}
		if d.pending == nil {
			if header.FunctionCode == FuncReceive && !d.allowInitialReceive {
				return nil, fmt.Errorf("%w: received terminal F_RECEIVE without a preceding F_SAP_SEND", ErrProtocol)
			}
			if err := d.checkLimits(len(fragment.Data), 1); err != nil {
				return nil, err
			}
			return []Message{singleMessage(header, fragment.Data, operationInfo)}, nil
		}
		if header.FunctionCode != FuncReceive {
			return nil, fmt.Errorf("%w: normal deallocation started a new F_SAP_SEND during a fragmented message", ErrProtocol)
		}
		if err := d.matchPending(header, operationInfo, "at normal deallocation"); err != nil {
			return nil, err
		}
		byteLength := d.pending.byteLength + len(fragment.Data)
		fragmentCount := d.pending.fragmentCount + 1
		if err := d.checkLimits(byteLength, fragmentCount); err != nil {
			return nil, err
		}
		message := d.assemblePending(fragment.Data, byteLength, fragmentCount)
		d.pending = nil
		return []Message{message}, nil
	}

	if header.FunctionCode == FuncSapSend {
		if d.pending != nil {
			return nil, fmt.Errorf("%w: received a new F_SAP_SEND during an unfinished fragmented message", ErrProtocol)
		}
		if err := d.checkLimits(len(fragment.Data), 1); err != nil {
			return nil, err
		}
		if fragment.IsFinal {
			return []Message{singleMessage(header, fragment.Data, operationInfo)}, nil
		}
		d.pending = newPending(header, fragment.Data, operationInfo)
		return []Message{}, nil
	}

	if d.pending == nil {
		if !d.allowInitialReceive {
			return nil, fmt.Errorf("%w: received F_RECEIVE without a preceding fragmented F_SAP_SEND", ErrProtocol)
		}
		if err := d.checkLimits(len(fragment.Data), 1); err != nil {
			return nil, err
		}
		if fragment.IsFinal {
			return []Message{singleMessage(header, fragment.Data, operationInfo)}, nil
		}
		d.pending = newPending(header, fragment.Data, operationInfo)
		return []Message{}, nil
	}
	if err := d.matchPending(header, operationInfo, "within a fragmented message"); err != nil {
		return nil, err
	}
	byteLength := d.pending.byteLength + len(fragment.Data)
	fragmentCount := d.pending.fragmentCount + 1
	if err := d.checkLimits(byteLength, fragmentCount); err != nil {
		return nil, err
	}
	d.pending.chunks = append(d.pending.chunks, append([]byte(nil), fragment.Data...))
	d.pending.byteLength = byteLength
	d.pending.fragmentCount = fragmentCount
	if !fragment.IsFinal {
		return []Message{}, nil
	}
	message := Message{
		Data:               concatChunks(d.pending.chunks, d.pending.byteLength),
		ConversationID:     append([]byte(nil), d.pending.conversationID...),
		SequenceNumber:     d.pending.sequenceNumber,
		FragmentCount:      d.pending.fragmentCount,
		CommunicationIndex: d.pending.communicationIndex,
		ConnectionIndex:    d.pending.connectionIndex,
	}
	d.pending = nil
	return []Message{message}, nil
}

func (d *ConversationDecoder) matchPending(header Header, operationInfo IncomingDataOperationInfo, where string) error {
	if !bytes.Equal(d.pending.conversationID, header.ConversationID) {
		return fmt.Errorf("%w: APPC conversation ID changed %s", ErrProtocol, where)
	}
	if d.pending.sequenceNumber != header.SequenceNumber {
		return fmt.Errorf("%w: APPC sequence number changed %s", ErrProtocol, where)
	}
	if d.pending.communicationIndex != operationInfo.CommunicationIndex || d.pending.connectionIndex != operationInfo.ConnectionIndex {
		return fmt.Errorf("%w: APPC connection indices changed %s", ErrProtocol, where)
	}
	return nil
}

func (d *ConversationDecoder) assemblePending(finalData []byte, byteLength, fragmentCount int) Message {
	chunks := append(append([][]byte(nil), d.pending.chunks...), append([]byte(nil), finalData...))
	return Message{
		Data:               concatChunks(chunks, byteLength),
		ConversationID:     append([]byte(nil), d.pending.conversationID...),
		SequenceNumber:     d.pending.sequenceNumber,
		FragmentCount:      fragmentCount,
		CommunicationIndex: d.pending.communicationIndex,
		ConnectionIndex:    d.pending.connectionIndex,
	}
}

func (d *ConversationDecoder) checkLimits(byteLength, fragmentCount int) error {
	if byteLength > d.maxMessageLength {
		return fmt.Errorf("%w: APPC message length %d exceeds configured limit %d", ErrRange, byteLength, d.maxMessageLength)
	}
	if fragmentCount > d.maxFragments {
		return fmt.Errorf("%w: APPC fragment count %d exceeds configured limit %d", ErrRange, fragmentCount, d.maxFragments)
	}
	return nil
}

// Finish reports a truncated message left buffered.
func (d *ConversationDecoder) Finish() error {
	if d.pending != nil {
		return fmt.Errorf("%w: truncated APPC message: %d fragment(s), %d bytes buffered", ErrProtocol, d.pending.fragmentCount, d.pending.byteLength)
	}
	return nil
}

func singleMessage(header Header, data []byte, operationInfo IncomingDataOperationInfo) Message {
	return Message{
		Data:               append([]byte(nil), data...),
		ConversationID:     append([]byte(nil), header.ConversationID...),
		SequenceNumber:     header.SequenceNumber,
		FragmentCount:      1,
		CommunicationIndex: operationInfo.CommunicationIndex,
		ConnectionIndex:    operationInfo.ConnectionIndex,
	}
}

func newPending(header Header, data []byte, operationInfo IncomingDataOperationInfo) *pendingMessage {
	return &pendingMessage{
		chunks:             [][]byte{append([]byte(nil), data...)},
		conversationID:     append([]byte(nil), header.ConversationID...),
		sequenceNumber:     header.SequenceNumber,
		byteLength:         len(data),
		fragmentCount:      1,
		communicationIndex: operationInfo.CommunicationIndex,
		connectionIndex:    operationInfo.ConnectionIndex,
	}
}

func concatChunks(chunks [][]byte, total int) []byte {
	out := make([]byte, 0, total)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}
