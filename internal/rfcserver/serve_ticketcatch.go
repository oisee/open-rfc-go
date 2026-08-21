// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"fmt"
	"net"

	"github.com/oisee/open-rfc-go/internal/transport"
)

// ServeTicketCatch plays the SAP gateway for a type-T *registered server*
// destination: it accepts the client's ALLOCATE for a program id it never
// actually registered, so the ABAP client then sends its CPIC logon — the first
// logon composed by a real client that we get to read, and the only place a
// forwarded logon ticket can appear on the classic-RFC wire.
//
// It answers exactly three things and logs/dumps everything else:
//
//	64-byte GW_NORMAL_CLIENT   -> capability ack (same as the type-3 server)
//	APPC ALLOCATE (fn 0xca)    -> accept: the 48-byte header with both return
//	                             codes and the error length zeroed, conversation
//	                             id assigned, no error trailer
//	CPIC logon (fn 0x03)       -> logon-accept, so the conversation completes and
//	                             the client does not retry; the logon itself is in
//	                             the dump by then
//
// It is a capture instrument, not a product server. The register side (our
// program registering at a real gateway) is a separate, larger task.
func ServeTicketCatch(conn net.Conn, logf func(string), dump func(dir string, frame []byte)) {
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

	var convID []byte
	var guid []byte
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

		case len(got) >= 48 && got[0] == 0x06 && got[1] == 0xca: // ALLOCATE
			acc := ticketCatchAccept(got)
			convID = append([]byte(nil), acc[40:48]...)
			if send(acc) != nil {
				return
			}
			log(fmt.Sprintf("ALLOCATE: accepted (conv=%s) — awaiting the client logon", string(convID)))

		case len(got) >= 48 && got[0] == 0x06 && got[1] == 0x03: // CPIC logon
			guid = findRFCGUID(got)
			log(fmt.Sprintf("LOGON: %d bytes — the ticket, if any, is in this frame", len(got)))
			acc := patchSession(acceptFor(len(got)), got[40:48], guid)
			if send(acc) != nil {
				return
			}

		case len(got) == 8 && string(got) == "NI_PING\x00": // keepalive
			if send(niPongTC) != nil {
				return
			}
			log("NI_PING -> NI_PONG (holding the conversation open for the logon)")

		default:
			log(fmt.Sprintf("frame: %d bytes fn=0x%02x (no reply)", len(got), frameFn(got)))
		}
	}
}
