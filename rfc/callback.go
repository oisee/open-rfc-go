// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"context"
	"encoding/binary"

	"github.com/oisee/open-rfc-go/internal/client"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

// frameCallbackResponse turns a server-response CUT payload (which ends with the
// 0xffff End sentinel) into a fully framed compact CPIC message the session can
// send, by appending the 8-byte final-SAP trailer [0x0000, len, maxRfcPacketSize]
// after that sentinel. Small messages must travel compact; large ones (rare for
// a callback) stay streamed.
func frameCallbackResponse(resp []byte) []byte {
	const maxRfcPacketSize = 0x8500
	if len(resp) < 2 || len(resp) > 28000 {
		return resp // streamed sentinel already present
	}
	trailer := make([]byte, 8)
	binary.BigEndian.PutUint16(trailer[0:], 0)
	binary.BigEndian.PutUint16(trailer[2:], uint16(len(resp)))
	binary.BigEndian.PutUint32(trailer[4:], maxRfcPacketSize)
	return append(append([]byte(nil), resp...), trailer...)
}

// CallbackRequest is a server-initiated callback into this client (the server
// executed CALL FUNCTION <Function> DESTINATION 'BACK'). Import values are the
// raw classic-encoded bytes as they arrived; decode with the classicrfc/xrfc
// helpers if needed.
type CallbackRequest struct {
	Function string
	Imports  map[string][]byte
}

// CallbackResponse is what the client returns for a callback: export values by
// name, raw classic-encoded bytes.
type CallbackResponse struct {
	Exports map[string][]byte
}

// CallbackFunc handles one callback function module. Register handlers per FM
// name in Destination.Callbacks.
type CallbackFunc func(ctx context.Context, req CallbackRequest) (CallbackResponse, error)

// callbackHandler adapts the registered Destination.Callbacks into the low-level
// session callback handler (raw request bytes → raw response bytes). It returns
// nil when no callbacks are registered, so a call runs the plain path.
func (c *Client) callbackHandler(ctx context.Context) client.CallbackHandler {
	if len(c.dest.Callbacks) == 0 {
		return nil
	}
	return func(raw []byte) ([]byte, error) {
		req, err := rfcserver.DecodeCutFunctionRequest(raw)
		if err != nil {
			return nil, err
		}
		fn, ok := c.dest.Callbacks[req.FunctionName]
		if !ok {
			// Unknown callback FM: answer with a declared exception the server
			// can observe, rather than tearing down the conversation.
			exc, err := rfcserver.EncodeCutFunctionExceptionResponse("FU_NOT_FOUND")
			if err != nil {
				return nil, err
			}
			return frameCallbackResponse(exc), nil
		}
		imports := make(map[string][]byte, len(req.Imports))
		for _, imp := range req.Imports {
			imports[imp.Name] = imp.Value
		}
		resp, err := fn(ctx, CallbackRequest{Function: req.FunctionName, Imports: imports})
		if err != nil {
			return nil, err
		}
		exports := make([]cpic.NamedValue, 0, len(resp.Exports))
		for name, val := range resp.Exports {
			exports = append(exports, cpic.NamedValue{Name: name, Value: val})
		}
		encoded, err := rfcserver.EncodeCutFunctionResponse(exports, nil)
		if err != nil {
			return nil, err
		}
		return frameCallbackResponse(encoded), nil
	}
}
