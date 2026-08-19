// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/oisee/open-rfc-go/internal/appc"
	"github.com/oisee/open-rfc-go/internal/transport"
)

// KnownGoodPingResponse is the CUT body of a real RFC_PING success response,
// captured from a live A4H system and accepted by our own client (a success
// control plus a transaction id and a padded program context). RFC_PING carries
// no export parameters, so this body is reused verbatim as our ping answer while
// we learn what the ABAP client minimally requires.
var KnownGoodPingResponse, _ = hex.DecodeString(
	"050000000500050300000503051400103379b550926340cd6210b1ea480b3cdf" +
		"05140420000400000000042005120000051201300050530041005000" +
		"4c005300520046004300200020002000200020002000200020002000" +
		"200020002000200020002000200020002000200020002000200020002000" +
		"20002000200020002000200020000130066700080000000000606c40" +
		"0667ffff0000ffff")

// Wire constants used below are explained in docs/discoveries/0002-wire-constants.md.
//
// ServeGenerate answers one live SAP RFC client as our own server. It replays a
// proven-good handshake (echo the 64-byte gateway record, then send the recorded
// CPIC logon-accept), then for every F_SAP_SEND function request it sends
// respCUT wrapped in an F_SAP_SEND record that mirrors the client's conversation
// id and uid. This is deliberately function-agnostic: an SM59 type-3 Connection
// Test issues only RFC_PING, and the ABAP client's fast-serialized request is
// not classic-decodable, so we answer every request with the same ping body.
func ServeGenerate(conn net.Conn, logonAccept, respCUT []byte, logf func(string)) {
	defer conn.Close()
	log := func(s string) {
		if logf != nil {
			logf(s)
		}
	}
	t := transport.New(conn, transport.Options{})
	ctx := context.Background()
	for {
		frame, err := t.Receive(ctx)
		if err != nil {
			log(fmt.Sprintf("client closed: %v", err))
			return
		}
		switch {
		case len(frame) == gatewayRecordLen: // gateway normal-client record
			// Gateway normal-client record: echo it back.
			if err := t.Send(frame); err != nil {
				return
			}
			log(fmt.Sprintf("C->S gateway 64B -> echoed"))
		case len(frame) >= 2 && frame[0] == appc.ProtocolVersion && frame[1] == appcInit: // CPIC-init (logon)
			// CPIC initialize + logon: reply with the recorded logon-accept.
			if err := t.Send(logonAccept); err != nil {
				return
			}
			log(fmt.Sprintf("C->S CPIC-init %dB -> logon-accept %dB", len(frame), len(logonAccept)))
		case len(frame) >= 80 && frame[0] == appc.ProtocolVersion && frame[1] == byte(appc.FuncSapSend):
			// F_SAP_SEND carrying a function request: answer with respCUT.
			uid := binary.BigEndian.Uint16(frame[appcUIDOffset:])                    // APPC header uid (BE)
			convID := append([]byte(nil), frame[appcConvOffset:appcConvOffset+8]...) // 8-byte conversation id
			resp, werr := wrapFSapSend(respCUT, convID, uid)
			if werr != nil {
				log("wrap error: " + werr.Error())
				return
			}
			if err := t.Send(resp); err != nil {
				return
			}
			log(fmt.Sprintf("C->S F_SAP_SEND %dB (uid=%04x) -> response %dB", len(frame), uid, len(resp)))
		default:
			fn := byte(0)
			if len(frame) >= 2 {
				fn = frame[1]
			}
			log(fmt.Sprintf("C->S %dB appc-func=0x%02x -> (ignored)", len(frame), fn))
		}
	}
}

// wrapFSapSend builds a final F_SAP_SEND data record carrying cut, mirroring the
// client's conversation id and uid and marking the server direction.
func wrapFSapSend(cut, convID []byte, uid uint16) ([]byte, error) {
	final := true
	gwID := uint16(1) // server->client direction, as seen in captured responses
	return appc.EncodeDataRecord(appc.DataRecordInput{
		RecordHeaderInput: appc.RecordHeaderInput{
			UID:            &uid,
			GatewayID:      &gwID,
			ConversationID: convID,
		},
		Data:    cut,
		IsFinal: &final,
	})
}
