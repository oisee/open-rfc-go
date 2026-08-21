// SPDX-License-Identifier: Apache-2.0
//
// Go-idiomatic redesign of open-rfc src/client/direct-cpic-session.ts (commit
// 847036d, Copyright 2026 Marian Zeis, Apache-2.0). The upstream 1837-line
// class is Promise/AbortSignal orchestration around a fixed wire sequence:
// gateway GW_NORMAL_CLIENT, APPC Initialize/SetPartnerLuName/Allocate, the CPIC
// initial logon wrapped in an APPC data message, then request/response
// exchanges. Go expresses the same sequence with a blocking transport and
// context; the elaborate RfcFailure taxonomy collapses to wrapped errors. This
// is the flat-call path (compact CPIC packets) that reaches a live
// STFC_CONNECTION; streaming, pooling, and SAProuter are milestone 6. See
// docs/provenance.md.

// Package client drives a direct classic-RFC CPIC session to first login and call.
package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/oisee/open-rfc-go/internal/appc"
	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/gateway"
	"github.com/oisee/open-rfc-go/internal/rfcerr"
	"github.com/oisee/open-rfc-go/internal/saprouter"
	"github.com/oisee/open-rfc-go/internal/transport"
)

// ErrSession reports a session-level protocol or state error.
var ErrSession = errors.New("client: cpic session")

// Transport is the byte-record transport a session drives.
type Transport interface {
	Send(payload []byte) error
	Receive(ctx context.Context) ([]byte, error)
	Close() error
}

var (
	programNamePattern = regexp.MustCompile(`^[\x20-\x7e]{1,64}$`)
	servicePattern     = regexp.MustCompile(`^sapdp\d{2}$`)
)

// SessionOptions configures a direct CPIC session.
type SessionOptions struct {
	Host                     string
	Port                     int
	ApplicationServerHost    string // CPIC destination if the gateway host differs
	ApplicationServerService string // sapdpNN
	ProgramName              string // default "open-rfc"
	LocalAddress             string // advertised IPv4; default 127.0.0.1
	OperationTimeout         time.Duration
	// Router, if set, routes the connection through a SAProuter (its final hop
	// is the target gateway).
	Router *saprouter.Route
	// Proxy, if set, establishes the first TCP hop through a proxy (e.g. a
	// *socks5.Dialer).
	Proxy transport.ContextDialer
	// Transport, if set, is used instead of dialing Host:Port (for tests).
	Transport Transport
}

// LogonOptions carries the initial CPIC logon credentials.
type LogonOptions struct {
	Client          string
	User            string
	Password        string
	Ticket          string // SAP logon ticket, used instead of Password when set
	Language        string // default "E"
	PartnerHostName string // default os.Hostname()
	KernelRelease   string
}

// Session is an open, possibly authenticated, direct CPIC session.
type Session struct {
	transport       Transport
	setup           *appc.ClientSetupStateMachine
	conversationID  []byte
	connectionIndex uint16
	localAddress    string
	destination     string
	programName     string
	cpicSessionID   []byte
	service         string
	authenticated   bool
	closed          bool
	opTimeout       time.Duration
}

func p8(v uint8) *uint8 { return &v }

func short(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func hexSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return strings.ToUpper(hex.EncodeToString(b[:]))
}

func osUsername() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return "open-rfc"
}

func localHostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "localhost"
}

// Open connects, performs the gateway and APPC setup handshake, and returns a
// session ready for logon.
func Open(ctx context.Context, opts SessionOptions) (*Session, error) {
	programName := opts.ProgramName
	if programName == "" {
		programName = "open-rfc"
	}
	if !programNamePattern.MatchString(programName) {
		return nil, fmt.Errorf("%w: programName must contain 1..64 ASCII bytes", ErrSession)
	}
	if !servicePattern.MatchString(opts.ApplicationServerService) {
		return nil, fmt.Errorf("%w: applicationServerService must use the form sapdpNN", ErrSession)
	}
	localAddress := opts.LocalAddress
	if localAddress == "" {
		localAddress = "127.0.0.1"
	}
	opTimeout := opts.OperationTimeout
	if opTimeout == 0 {
		opTimeout = 30 * time.Second
	}
	destination := opts.ApplicationServerHost
	if destination == "" {
		destination = opts.Host
	}

	tr := opts.Transport
	if tr == nil {
		transportOpts := transport.Options{ReadTimeout: opTimeout, WriteTimeout: opTimeout}
		addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
		var dialed *transport.Transport
		var err error
		if opts.Router != nil || opts.Proxy != nil {
			dialed, err = transport.DialWith(ctx, "tcp", addr, transport.DialOptions{Options: transportOpts, Proxy: opts.Proxy, Router: opts.Router})
		} else {
			dialed, err = transport.Dial(ctx, "tcp", addr, transportOpts)
		}
		if err != nil {
			return nil, err
		}
		tr = dialed
	}

	s := &Session{transport: tr, localAddress: localAddress, destination: destination, programName: programName, opTimeout: opTimeout, service: opts.ApplicationServerService}
	if err := s.handshake(ctx); err != nil {
		_ = tr.Close()
		return nil, err
	}
	return s, nil
}

func (s *Session) handshake(ctx context.Context) error {
	// 1. Gateway GW_NORMAL_CLIENT.
	gwReq, err := gateway.EncodeNormalClient(gateway.NormalClientRecord{
		Address: s.localAddress, Service: short(s.programName, 9), CodePage: "1100", GatewayOptionLevel: 6,
		LogicalUnit: short(localHostname(), 8), TransactionProgram: short(s.programName, 8), ConversationID: "",
		AppcHeaderVersion: 6,
		AcceptInfo: gateway.AcceptInfoErrorInfo | gateway.AcceptInfoPing | gateway.AcceptInfoConnectionExtended |
			gateway.AcceptInfoCodePage | gateway.AcceptInfoExtendedInitOptions | gateway.AcceptInfoDistributedTrace,
		Index: -1,
	})
	if err != nil {
		return err
	}
	if err := s.transport.Send(gwReq); err != nil {
		return err
	}
	gwReplyBytes, err := s.transport.Receive(ctx)
	if err != nil {
		return err
	}
	gw, err := gateway.DecodeNormalClient(gwReplyBytes)
	if err != nil {
		return err
	}
	if gw.ReturnCode != 0 {
		return fmt.Errorf("%w: gateway rejected GW_NORMAL_CLIENT with return code %d", ErrSession, gw.ReturnCode)
	}
	if gw.AppcHeaderVersion != 6 {
		return fmt.Errorf("%w: gateway selected unsupported APPC header version %d", ErrSession, gw.AppcHeaderVersion)
	}
	if gw.AcceptInfo&gateway.AcceptInfoExtendedInitOptions == 0 {
		return fmt.Errorf("%w: gateway did not accept extended initialization options", ErrSession)
	}
	if gw.AcceptInfo&gateway.AcceptInfoCodePage == 0 || gw.CodePage != "4103" {
		return fmt.Errorf("%w: direct classic RFC supports only little-endian Unicode partner code page 4103", ErrSession)
	}

	// 2. APPC Initialize.
	s.setup = appc.NewClientSetupStateMachine()
	svc := s.service
	initParams, err := appc.EncodeInitializeParameters(appc.InitializeParameters{
		ClientIdentifier: "NWRFC",
		Options: appc.ExtendedInitializeOptions{
			OptionFlags: 1, RootID: hexSessionID(), ConnectionID: hexSessionID(), ConnectionIDSuffix: 1,
			Timeout: -2, KeepaliveTimeout: -2, ExportTrace: 2, StartType: 0, NetworkProtocol: 0,
			LocalAddressV6: make([]byte, 16), LongLogicalUnitName: s.localAddress,
			OperatingSystemUser: short(osUsername(), 12), LocalAddressV4: make([]byte, 4),
			LongTransactionProgramName: svc,
		},
	})
	if err != nil {
		return err
	}
	initialize, err := appc.EncodeControlRecord(appc.ControlRecordInput{
		RecordHeaderInput: appc.RecordHeaderInput{ConversationID: bytesRepeat(0x20, 8), Info2: p8(1), Info3: p8(0xc0), Info4: p8(4), Info: p8(5)},
		FunctionCode:      appc.FuncInitialize,
		ExtendedInfo: &appc.ExtendedInfo{
			ShortDestinationName: "NWRFC", LogicalUnitName: short(s.localAddress, 8), TransactionProgramName: svc,
			ConnectionType: 0x49, ClientInfo: 1, CommunicationIndex: 0, ConnectionIndex: 0xffff,
		},
		Parameters: initParams,
	})
	if err != nil {
		return err
	}
	if err := s.setup.Sent(appc.FuncInitialize, true); err != nil {
		return err
	}
	if err := s.transport.Send(initialize); err != nil {
		return err
	}
	initReply, err := s.transport.Receive(ctx)
	if err != nil {
		return err
	}
	if _, err := s.setup.Received(initReply); err != nil {
		return err
	}
	initHeader, err := appc.DecodeHeader(initReply)
	if err != nil {
		return err
	}
	if len(initReply) < appc.RecordHeaderLength {
		return fmt.Errorf("%w: APPC initialize reply is truncated", ErrSession)
	}
	initInfo, err := appc.DecodeExtendedInfo(initReply[appc.CommonHeaderLength:appc.RecordHeaderLength])
	if err != nil {
		return err
	}
	s.conversationID = initHeader.ConversationID
	s.connectionIndex = initInfo.ConnectionIndex

	// 3. APPC SetPartnerLuName (no reply).
	if err := s.setup.Sent(appc.FuncSetPartnerLuName, true); err != nil {
		return err
	}
	setPartner, err := appc.EncodeControlRecord(appc.ControlRecordInput{
		RecordHeaderInput: appc.RecordHeaderInput{ConversationID: s.conversationID, Info2: p8(1), Info: p8(4)},
		FunctionCode:      appc.FuncSetPartnerLuName,
		PartnerLogicalUnitInfo: &appc.PartnerLogicalUnitInfoInput{
			LogicalUnitName: s.localAddress, PartnerHostAddress: make([]byte, 16), CommunicationIndex: 0xffff, ConnectionIndex: s.connectionIndex,
		},
		Parameters: mustPartnerParams(s.localAddress),
	})
	if err != nil {
		return err
	}
	if err := s.transport.Send(setPartner); err != nil {
		return err
	}

	// 4. APPC Allocate (one reply).
	if err := s.setup.Sent(appc.FuncAllocate, true); err != nil {
		return err
	}
	allocate, err := appc.EncodeControlRecord(appc.ControlRecordInput{
		RecordHeaderInput: appc.RecordHeaderInput{ConversationID: s.conversationID, Info: p8(1)},
		FunctionCode:      appc.FuncAllocate,
		ExtendedInfo:      &appc.ExtendedInfo{CommunicationIndex: 0xffff, ConnectionIndex: s.connectionIndex},
	})
	if err != nil {
		return err
	}
	if err := s.transport.Send(allocate); err != nil {
		return err
	}
	allocReply, err := s.transport.Receive(ctx)
	if err != nil {
		return err
	}
	if _, err := s.setup.Received(allocReply); err != nil {
		return err
	}
	return nil
}

func mustPartnerParams(localAddress string) []byte {
	b, err := appc.EncodePartnerLogicalUnitParameters(appc.PartnerLogicalUnitParameters{LongLogicalUnitName: localAddress, PartnerHostAddress: make([]byte, 16)})
	if err != nil {
		panic(err)
	}
	return b
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

// LogonAndPing performs the initial CPIC logon (an RFCPING) and, on success,
// marks the session authenticated.
func (s *Session) LogonAndPing(ctx context.Context, opts LogonOptions) error {
	if s.authenticated {
		return fmt.Errorf("%w: session is already authenticated", ErrSession)
	}
	language := opts.Language
	if language == "" {
		language = "E"
	}
	partnerHost := opts.PartnerHostName
	if partnerHost == "" {
		partnerHost = localHostname()
	}
	sessionID := make([]byte, 16)
	if _, err := rand.Read(sessionID); err != nil {
		return err
	}
	req, err := cpic.EncodeInitialLogonRequest(cpic.InitialLogonRequestInput{
		Client: opts.Client, User: opts.User, Password: opts.Password, Ticket: opts.Ticket, Language: language,
		ClientAddress: s.localAddress, PartnerHostName: partnerHost, Destination: s.destination,
		ProgramName: s.programName, KernelRelease: opts.KernelRelease, FunctionName: "RFCPING", SessionID: sessionID,
	})
	if err != nil {
		return err
	}
	response, err := s.exchange(ctx, req)
	if err != nil {
		return err
	}
	decoded, err := cpic.DecodeInitialLogonResponse(response)
	if err != nil {
		return fmt.Errorf("%w: CPIC logon response is malformed: %v", ErrSession, err)
	}
	if !decoded.Success {
		if decoded.Rejection != nil && decoded.Rejection.Text != "" {
			return fmt.Errorf("%w: SAP rejected logon: %s", ErrSession, decoded.Rejection.Text)
		}
		if decoded.Status != nil {
			return fmt.Errorf("%w: SAP rejected logon with status %d", ErrSession, *decoded.Status)
		}
		return fmt.Errorf("%w: SAP rejected the initial CPIC logon", ErrSession)
	}
	s.cpicSessionID = sessionID
	s.authenticated = true
	return nil
}

// exchange sends one CPIC request wrapped in APPC and reassembles the reply.
// exchange sends one CPIC request and returns the single reassembled reply. It is
// used for setup/logon and for calls without callbacks.
func (s *Session) exchange(ctx context.Context, request []byte) ([]byte, error) {
	async, err := s.sendRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	return s.receiveMessage(ctx, async)
}

// sendRequest splits a pre-framed CPIC request into application data and final
// SAP parameters and writes it. It reports whether the first outgoing fragment
// was F_ASEND_DATA (which the reply decoder must be told to allow as its initial
// receive).
func (s *Session) sendRequest(ctx context.Context, request []byte) (bool, error) {
	framing, err := cpic.InspectRequestAppcFraming(request)
	if err != nil {
		return false, err
	}
	var finalSap []byte
	if framing.FinalSapParameterLength != 0 {
		finalSap = request[framing.ApplicationDataLength:]
	}
	return s.sendData(ctx, request[:framing.ApplicationDataLength], finalSap)
}

// sendData frames raw CPIC application data (+ optional final SAP parameters) as
// an outgoing APPC data message and writes it. Used for requests and for the
// response our client sends back to a server-initiated callback.
func (s *Session) sendData(ctx context.Context, appData, finalSap []byte) (bool, error) {
	if s.closed {
		return false, fmt.Errorf("%w: session is closed", ErrSession)
	}
	plan, err := appc.PlanOutgoingDataFragments(appc.OutgoingDataPlanInput{
		RecordHeaderInput:  appc.RecordHeaderInput{ConversationID: s.conversationID},
		ApplicationData:    appData,
		FinalSapParameters: finalSap,
		CommunicationIndex: 0xffff,
		ConnectionIndex:    s.connectionIndex,
	}, appc.OutgoingDataPlannerOptions{CpicStreaming: appc.StreamingDisabled})
	if err != nil {
		return false, err
	}
	if err := s.writeDataPlan(ctx, plan); err != nil {
		return false, err
	}
	return len(plan) > 0 && plan[0].FunctionCode == appc.FuncAsyncSendData, nil
}

// receiveMessage receives APPC records and returns the next reassembled RFC
// message, advancing the setup state machine. allowInitialReceive must be true
// when the immediately preceding send began with F_ASEND_DATA.
func (s *Session) receiveMessage(ctx context.Context, allowInitialReceive bool) ([]byte, error) {
	var decoder *appc.ConversationDecoder
	for {
		payload, err := s.transport.Receive(ctx)
		if err != nil {
			return nil, err
		}
		disposition, err := s.setup.Received(payload)
		if err != nil {
			return nil, err
		}
		if decoder == nil {
			decoder, err = appc.NewConversationDecoder(appc.ConversationDecoderOptions{
				AllowInitialReceive: allowInitialReceive, ValidateIncomingDataOperationInfo: true,
			})
			if err != nil {
				return nil, err
			}
		}
		var messages []appc.Message
		if disposition == appc.DispositionNormalDeallocation {
			messages, err = decoder.PushTerminalDeallocation(payload)
		} else {
			messages, err = decoder.Push(payload)
		}
		if err != nil {
			return nil, err
		}
		if len(messages) == 0 {
			continue
		}
		if len(messages) != 1 {
			return nil, fmt.Errorf("%w: APPC reply contained more than one RFC message", ErrSession)
		}
		if err := decoder.Finish(); err != nil {
			return nil, err
		}
		message := messages[0]
		if len(message.ConversationID) != 8 || !equalBytes(message.ConversationID, s.conversationID) ||
			message.CommunicationIndex != 0 || message.ConnectionIndex != s.connectionIndex {
			return nil, fmt.Errorf("%w: APPC response identity does not match the active conversation", ErrSession)
		}
		if disposition == appc.DispositionNormalDeallocation {
			s.closed = true
		} else if err := s.setup.ResponseComplete(); err != nil {
			return nil, err
		}
		return message.Data, nil
	}
}

func (s *Session) writeDataPlan(ctx context.Context, plan []appc.OutgoingDataFragment) error {
	if len(plan) == 0 {
		return fmt.Errorf("%w: outgoing APPC plan is empty", ErrSession)
	}
	for _, fragment := range plan {
		record, err := appc.EncodeOutgoingDataFragment(fragment)
		if err != nil {
			return err
		}
		isFinal := fragment.FunctionCode == appc.FuncSapSend || fragment.FunctionCode == appc.FuncReceive
		if err := s.setup.Sent(fragment.FunctionCode, isFinal); err != nil {
			return err
		}
		if err := s.transport.Send(record); err != nil {
			return err
		}
		if fragment.FunctionCode == appc.FuncSendData {
			ack, err := s.transport.Receive(ctx)
			if err != nil {
				return err
			}
			if _, err := appc.DecodeSynchronousSendAcknowledgement(ack); err != nil {
				return err
			}
			if _, err := s.setup.Received(ack); err != nil {
				return err
			}
		}
	}
	return nil
}

// CallResult holds the decoded application fields of one RFC call, plus the
// decoded error envelope (outcome, ABAP exception/message/runtime facts).
type CallResult struct {
	Success  bool
	Fields   []cpic.Field
	Envelope rfcerr.Envelope
}

// Call invokes one classic Unicode (CUT) function with the given imports,
// declaring which export/table parameters the server should return
// (requestedOutputs), and returns its decoded application result fields.
func (s *Session) Call(ctx context.Context, functionName string, imports []cpic.NamedValue, requestedOutputs []string) (CallResult, error) {
	if !s.authenticated {
		return CallResult{}, fmt.Errorf("%w: session must be authenticated before a call", ErrSession)
	}
	req, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{FunctionName: functionName, Imports: imports, RequestedOutputs: requestedOutputs})
	if err != nil {
		return CallResult{}, err
	}
	response, err := s.exchange(ctx, req)
	if err != nil {
		return CallResult{}, err
	}
	decoded, err := cpic.DecodeFunctionResultFields(response)
	if err != nil {
		return CallResult{}, err
	}
	return CallResult{Success: decoded.Success, Fields: decoded.Fields, Envelope: decoded.Envelope}, nil
}

// CallRaw sends a pre-built classic-RFC (CUT) request and returns the decoded
// application result fields. It is the low-level entry the metadata builders in
// internal/metadata target: they produce request bytes, and callers pass the
// resulting fields to the matching decoder.
func (s *Session) CallRaw(ctx context.Context, request []byte) (CallResult, error) {
	if !s.authenticated {
		return CallResult{}, fmt.Errorf("%w: session must be authenticated before a call", ErrSession)
	}
	return s.CallWithCallbacks(ctx, request, nil)
}

// CallbackHandler answers a server-initiated callback: given the raw CPIC request
// the server sent back over the same conversation, it returns the raw CPIC
// response to send. nil means callbacks are unsupported (an incoming callback is
// then an error).
type CallbackHandler func(request []byte) (response []byte, err error)

// CallWithCallbacks issues a function call and, while awaiting its response,
// services any server-initiated callbacks (RFC "DESTINATION 'BACK'") over the
// same re-entrant conversation: each callback request is handed to onCallback and
// its response is sent back, until the function's own response arrives.
func (s *Session) CallWithCallbacks(ctx context.Context, request []byte, onCallback CallbackHandler) (CallResult, error) {
	if !s.authenticated {
		return CallResult{}, fmt.Errorf("%w: session must be authenticated before a call", ErrSession)
	}
	async, err := s.sendRequest(ctx, request)
	if err != nil {
		return CallResult{}, err
	}
	for {
		data, err := s.receiveMessage(ctx, async)
		if err != nil {
			return CallResult{}, err
		}
		if isCallbackRequest(data) {
			if onCallback == nil {
				return CallResult{}, fmt.Errorf("%w: server sent an RFC callback but no handler is registered", ErrSession)
			}
			resp, cbErr := onCallback(data)
			if cbErr != nil {
				return CallResult{}, cbErr
			}
			// resp is a fully framed CPIC message (compact final-SAP trailer or
			// streamed sentinel); split and send it like any request.
			if async, err = s.sendRequest(ctx, resp); err != nil {
				return CallResult{}, err
			}
			continue
		}
		decoded, err := cpic.DecodeFunctionResultFields(data)
		if err != nil {
			return CallResult{}, err
		}
		return CallResult{Success: decoded.Success, Fields: decoded.Fields, Envelope: decoded.Envelope}, nil
	}
}

func isCallbackRequest(data []byte) bool {
	return len(data) >= 4 && data[0] == 0x05 && data[1] == 0x02 && data[2] == 0x00 && data[3] == 0x00
}

// CallSTFCConnection invokes STFC_CONNECTION with REQUTEXT and returns ECHOTEXT.
// It is the canonical smoke-test call: the echo must equal the request.
func (s *Session) CallSTFCConnection(ctx context.Context, requtext string) (echo string, resp string, err error) {
	value, err := classicrfc.EncodeAbapChar(requtext, 255)
	if err != nil {
		return "", "", err
	}
	result, err := s.Call(ctx, "STFC_CONNECTION", []cpic.NamedValue{{Name: "REQUTEXT", Value: value}}, []string{"ECHOTEXT", "RESPTEXT"})
	if err != nil {
		return "", "", err
	}
	classic, err := classicrfc.DecodeResult(result.Fields)
	if err != nil {
		return "", "", err
	}
	for _, scalar := range classic.Scalars {
		switch scalar.Name {
		case "ECHOTEXT":
			echo, err = classicrfc.DecodeAbapChar(scalar.Value)
			if err != nil {
				return "", "", err
			}
		case "RESPTEXT":
			resp, err = classicrfc.DecodeAbapChar(scalar.Value)
			if err != nil {
				return "", "", err
			}
		}
	}
	return echo, resp, nil
}

// Close closes the session transport.
func (s *Session) Close() error {
	s.closed = true
	return s.transport.Close()
}

// Authenticated reports whether the session has logged on.
func (s *Session) Authenticated() bool { return s.authenticated }

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
