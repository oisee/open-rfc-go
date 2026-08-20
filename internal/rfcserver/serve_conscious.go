// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/transport"
)

// ServeConscious answers a live SM59 type-3 client by GENERATING each response
// from a registered Go handler, not by replaying a capture. It reuses the proven
// handshake (echo the gateway record, generate the logon-accept by mirroring this
// client's conversation id and RFC GUID), then for every function request it
// decodes the call, dispatches it to d, and wraps the handler's classic-encoded
// response in an F_SAP_SEND record. This is the "conscious" server — the base for
// the polyglot RFC bridge (docs/polyglot-rfc-server.md).
//
// It generates classic-serialization responses, so the client must speak classic
// (SM59 Special Options → Serializer = Classic serializer). RFC_PING falls back
// to the baked ping dance when no handler is registered for it.
func ServeConscious(conn net.Conn, d *Dispatcher, logf func(string), dump func(dir string, frame []byte)) {
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
	pingStep := 0
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
		case len(got) == 64: // gateway record
			reply := append([]byte(nil), got...)
			reply[gatewayAckOffset1] = gatewayAckLevel
			reply[gatewayAckOffset2] = gatewayAckCaps
			if send(reply) != nil {
				return
			}
			log("CONNECT: gateway acknowledged")
		case len(got) >= 48 && got[0] == 0x06 && got[1] == 0x03: // CPIC-init
			convID = append([]byte(nil), got[appcConvOffset:appcConvOffset+8]...)
			guid = findRFCGUID(got)
			// Always serve the classic-only accept (advertises classic via
			// 0x0512, no fast-ser 0x5001): per SAP docs fast serialization is
			// used only if BOTH sides support it, so refusing to advertise it
			// makes the ABAP client fall back to classic (verified live: the
			// client then sends 0x0201 classic params). acceptFor templates are
			// kept for the handshake-test ports (see cmd/rfc-lab).
			template := classicWorkAccept
			_ = acceptFor
			acc := patchSession(template, convID, guid)
			if send(acc) != nil {
				return
			}
			pingStep = 0
			log(fmt.Sprintf("LOGON: init=%dB accept=%dB (conv=%s)", len(got), len(acc), string(convID)))
		case len(got) > 80 && got[0] == 0x06 && got[1] == byte(0xcb): // function request
			req, derr := DecodeCutFunctionRequest(got[80:])
			if derr != nil {
				log("SESSION: decode error: " + derr.Error())
				return
			}
			fn := req.FunctionName
			if _, ok := d.handler(fn); !ok && fn == "RFC_PING" {
				// no ping handler: keep the client alive with the recorded dance
				resp := patchSession(smartPingSteps[pingStep%len(smartPingSteps)], convID, guid)
				if send(resp) != nil {
					return
				}
				pingStep++
				log("SESSION: RFC_PING (ping dance)")
				continue
			}
			var respCUT []byte
			var werr error
			if resp, excKey := d.Invoke(ctx, req); excKey != "" {
				respCUT, werr = EncodeCutFunctionExceptionResponse(excKey)
				log(fmt.Sprintf("SESSION: %s -> exception %s", fn, excKey))
			} else {
				// Lean classic response (no 0x0104 S4 block), matching what a
				// live A4H server sends for a normal call — verified vs .105.
				respCUT, werr = EncodeCutFunctionResponseClassic(resp.Exports, resp.Tables, guid)
				log(fmt.Sprintf("SESSION: %s -> generated (%d exports, %d tables, %dB)", fn, len(resp.Exports), len(resp.Tables), len(respCUT)))
			}
			if werr != nil {
				log("SESSION: encode error: " + werr.Error())
				return
			}
			wrapped, werr := wrapFSapSend(respCUT, convID, binary.BigEndian.Uint16(got[4:6]))
			if werr != nil {
				log("SESSION: wrap error: " + werr.Error())
				return
			}
			if send(wrapped) != nil {
				return
			}
		default:
			// control frames need no reply
		}
	}
}

// DefaultDispatcher returns a dispatcher with a few demonstration handlers, so a
// conscious server answers common calls out of the box: RFC_PING (empty success)
// and STFC_CONNECTION (echoes REQUTEXT back as ECHOTEXT with a RESPTEXT banner).
// Real deployments register their own handlers — the base for the polyglot bridge.
func DefaultDispatcher() *Dispatcher {
	d := NewDispatcher()
	// RFC_PING is intentionally left unregistered so ServeConscious answers it with
	// the proven ping dance; STFC_CONNECTION exercises the generated path.
	d.Handle("STFC_CONNECTION", func(ctx context.Context, req Request) (Response, error) {
		var echo []byte
		for _, imp := range req.Imports {
			if imp.Name == "REQUTEXT" {
				echo = imp.Value
			}
		}
		resp, err := classicrfc.EncodeAbapChar("open-rfc-go conscious server", 255)
		if err != nil {
			return Response{}, err
		}
		return Response{Exports: []cpic.NamedValue{
			{Name: "ECHOTEXT", Value: echo},
			{Name: "RESPTEXT", Value: resp},
		}}, nil
	})
	return d
}
