// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"fmt"
	"net"

	"github.com/oisee/open-rfc-go/internal/transport"
)

// ServeTypeT impersonates the SAP gateway *and* the registered server for a
// type-T client, then answers real function calls from a registered Go handler.
//
// Unlike ServeConscious (type-3 AS, where the client logs on as a user and we
// fake the whole logon/serializer negotiation), a registered server never
// authenticates anyone: the calling system already vouched for the user, and the
// gateway just hands us the function call. That makes this the *smaller*
// emulation — we accept the conversation and dispatch, nothing more.
//
// The client-facing conversation, captured live (.private/gold/cap-leg2.jsonl):
//
//	64B NI hello        -> hello ack (level+1, caps byte cb->fb)
//	APPC ALLOCATE 0xca  -> 125B accept (both return codes zero, conv id assigned)
//	first CUT 0xcb      -> 241B reply  (the "logon accepted" turn; no user auth)
//	further CUT 0xcb    -> DECODE as a function request, dispatch, reply
//
// The FM-call CUT (the last line) is the one frame we had never driven into this
// role; its application data is the same classic CUT that ServeConscious already
// decodes, so the content layer is shared. The discriminator is exact: a function
// request's application data opens with the CUT request prefix (05 02 00 00); the
// logon CUT does not, so DecodeCutFunctionRequest rejects it and we fall back to
// the logon reply.
func ServeTypeT(conn net.Conn, d *Dispatcher, logf func(string), dump func(dir string, frame []byte)) {
	defer conn.Close()
	log := func(s string) {
		if logf != nil {
			logf(s)
		}
	}
	t := transport.New(conn, transport.Options{})
	ctx := context.Background()
	send := func(b []byte) error {
		if dump != nil {
			dump("S->C", b)
		}
		return t.Send(b)
	}

	var convID, guid []byte
	for {
		got, err := t.Receive(ctx)
		if err != nil {
			log(fmt.Sprintf("client closed: %v", err))
			return
		}
		if dump != nil {
			dump("C->S", got)
		}
		switch {
		case IsNIRouteHello(got): // NI route hello
			if send(typeTHelloAck(got)) != nil {
				return
			}
			log("CONNECT: NI hello acknowledged")

		case Classify(got) == FrameNIPing: // keepalive
			// The client pings after the ALLOCATE accept and blocks until it sees
			// the pong; without it the conversation never reaches the call CUT.
			if send(NIPong) != nil {
				return
			}
			log("NI_PING -> NI_PONG")

		case len(got) >= 48 && got[0] == 0x06 && got[1] == 0xca: // ALLOCATE
			acc := typeTBuildAccept(got)
			convID = append([]byte(nil), acc[40:48]...)
			if send(acc) != nil {
				return
			}
			log(fmt.Sprintf("ALLOCATE: accepted (conv=%s)", string(convID)))

		case len(got) > 80 && got[0] == 0x06 && got[1] == 0xcb: // CUT: logon or function call
			guid = findRFCGUID(got)
			// A trusted-RFC CUT carries the logon/trust context first and the
			// function call after it, so the call's CUT prefix is not at a fixed
			// offset — find it. No prefix (or a payload that will not decode) means
			// a pure logon/handshake CUT; registered servers authenticate no one,
			// so we just accept the turn.
			off := indexCutRequest(got, 48)
			var req Request
			var derr error
			if off < 0 {
				derr = fmt.Errorf("no function-call CUT prefix")
			} else {
				req, derr = DecodeCutFunctionRequest(got[off:])
			}
			if derr != nil {
				if send(patchSession(append([]byte(nil), typeTLogonReply...), convID, guid)) != nil {
					return
				}
				log(fmt.Sprintf("CUT: handshake (%d bytes) -> accepted", len(got)))
				continue
			}
			fn := req.FunctionName
			var respCUT []byte
			var werr error
			if resp, excKey := d.Invoke(ctx, req); excKey != "" {
				respCUT, werr = EncodeCutFunctionExceptionResponse(excKey)
				log(fmt.Sprintf("CALL: %s -> exception %s", fn, excKey))
			} else {
				// A trusted-RFC reply carries no session GUID (the real reply has
				// no 0x0514 field), so pass nil — echoing one triggers the client's
				// "RFC GUID inconsistency" check.
				respCUT, werr = EncodeCutFunctionResponseS4(resp.Exports, resp.Tables, nil, req.RequestedOutputs)
				log(fmt.Sprintf("CALL: %s -> generated (%d exports, %d tables)", fn, len(resp.Exports), len(resp.Tables)))
			}
			if werr != nil {
				log("CALL: encode error: " + werr.Error())
				return
			}
			if send(wrapFSapSendTypeT(respCUT, got, convID)) != nil {
				return
			}

		default:
			log(fmt.Sprintf("frame: %d bytes fn=0x%02x (no reply)", len(got), frameFn(got)))
		}
	}
}

// wrapFSapSendTypeT builds the F_SAP_SEND reply record for the type-T path. The
// generic wrapFSapSend emits the type-3 / S4 record (protocol=2, gatewayID=1),
// which this client will not recognise as its response and blocks on. A real
// type-T reply record uses a different header (protocol=7, gatewayID=0, and a
// specific info/vector/return-code set), and echoes the request's uid at [4:6].
// We clone a captured type-T reply header (cap-zdouble.jsonl) and patch only the
// per-connection fields, then append the (already classic-serialized) content.
func wrapFSapSendTypeT(content, requestFrame, convID []byte) []byte {
	hdr := append([]byte(nil), typeTReplyHeaderBase...)
	if len(requestFrame) >= 6 {
		copy(hdr[4:6], requestFrame[4:6]) // echo the request uid
	}
	if len(convID) == 8 {
		copy(hdr[40:48], convID)
	}
	out := make([]byte, 0, len(hdr)+len(content))
	out = append(out, hdr...)
	out = append(out, content...)
	return out
}

// typeTReplyHeaderBase is a real 80-byte type-T F_SAP_SEND reply record header
// captured live (cap-zdouble.jsonl). wrapFSapSendTypeT patches its uid and
// conversation id; the rest (protocol, info/vector, return codes, operation
// info) stands. No credentials.
var typeTReplyHeaderBase = mustHexTC("06cb070063ac0000000000000000000001000000000200000000000002000108000000120000000034303530363837340000010b000000020000010b0000000000000000003131303000000000060001")

// indexCutRequest finds the CUT function-request prefix (05 02 00 00) at or
// after min, or -1. In a plain call it sits right after the 80-byte header; in a
// trusted-RFC call it sits past the logon/trust context.
func indexCutRequest(b []byte, min int) int {
	if min < 0 {
		min = 0
	}
	for i := min; i+4 <= len(b); i++ {
		if b[i] == 0x05 && b[i+1] == 0x02 && b[i+2] == 0x00 && b[i+3] == 0x00 {
			return i
		}
	}
	return -1
}

// typeTHelloAck turns the client's 64-byte NI route hello into the gateway's ack.
// Two bytes change, located structurally so a different hostname length is fine:
// the level byte at offset 29 (0x0e -> 0x0f) and the capability byte in the
// trailing "06 cb ff ff" marker (cb -> fb). Derived from cap-leg2.jsonl.
func typeTHelloAck(hello []byte) []byte {
	ack := append([]byte(nil), hello...)
	if len(ack) > 29 {
		ack[29] = 0x0f
	}
	for i := 0; i+3 < len(ack); i++ {
		if ack[i] == 0x06 && ack[i+1] == 0xcb && ack[i+2] == 0xff && ack[i+3] == 0xff {
			ack[i+1] = 0xfb
			break
		}
	}
	return ack
}

// typeTBuildAccept builds the 125-byte ALLOCATE accept the client will act on.
// A real gateway echoes three per-connection fields from the client's ALLOCATE
// into its accept; a static template (our first attempt) left stale values
// there, so the client could not correlate the accept to its request and
// stalled after the NI ping instead of sending the call. Derived by diffing a
// live accept against the ALLOCATE that provoked it (cap-zdouble.jsonl):
//
//	acc[4:6]    <- allocate[4:6]     the request uid (APPC header)
//	acc[28]     <- allocate[28]      a request flag
//	acc[88:113] <- the request GUID block: 0x00, 16 ASCII-hex chars, an 8-byte
//	               token, sitting just past the destination/program strings
//
// The conversation id (acc[40:48]) is the gateway's to assign, so the base
// value stands and we reuse it for the rest of the conversation.
func typeTBuildAccept(allocate []byte) []byte {
	acc := append([]byte(nil), typeTAcceptBase...)
	if len(allocate) >= 29 {
		acc[4], acc[5] = allocate[4], allocate[5]
		acc[28] = allocate[28]
	}
	if g := findAllocGUIDBlock(allocate); g >= 0 && g+25 <= len(allocate) {
		copy(acc[88:113], allocate[g:g+25])
	}
	return acc
}

// findAllocGUIDBlock locates the request GUID block in an ALLOCATE: a run of 16
// ASCII hex characters preceded by a 0x00. Returns the index of that 0x00 (the
// block start), or -1. Located by pattern so a different destination-name length
// does not shift it.
func findAllocGUIDBlock(b []byte) int {
	isHex := func(c byte) bool {
		return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')
	}
	for i := 1; i+16 <= len(b); i++ {
		if b[i-1] != 0x00 {
			continue
		}
		ok := true
		for j := 0; j < 16; j++ {
			if !isHex(b[i+j]) {
				ok = false
				break
			}
		}
		if ok {
			return i - 1
		}
	}
	return -1
}

// typeTAcceptBase is a real 125-byte ALLOCATE accept captured live
// (cap-zdouble.jsonl) for the trusted-RFC path. typeTBuildAccept overwrites its
// per-connection fields; the structural bytes stand. No credentials.
var typeTAcceptBase = mustHexTC("06ca070063ac0000000000000000000080000000000200000000000002000100000000000000000034303530363837340000000000000000000000000000000000000000000000000000000000060001010000000000000000443739433938433731363833423737426a85d4fa99600b53e1000000ac11000300000000")

// typeTLogonReply is the real 241-byte reply the gateway/registered server sends
// after the first (handshake) CUT (cap-leg2.jsonl). It carries the server's own
// system identity, not the caller's credentials. patchSession stamps our
// conversation id and RFC GUID onto it.
var typeTLogonReply = mustHexTC("06cb07006043000000000000000000000100000000020000000000000100010800000012000000003337383832303838000000a100000002000000a1000000000000000000313130300000000000000201010008030101050401000301010103000400000e1b01030106000b04010003000a02000000230106000700143100370032002e00310037002e0030002e003300000700110002450000110012000637003900330000120013001c37003900330020003100370032002e00310037002e0030002e00330000130130000e72006600630065007800650063000130066700080000000000e06d400667ffff0000ffff")
