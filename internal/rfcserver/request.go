// SPDX-License-Identifier: Apache-2.0
//
// Server-side of the classic RFC CUT protocol. Original work for open-rfc-go
// (milestone 8, RFC-server track): where internal/cpic encodes a CUT request
// as a client and decodes the response, this decodes an inbound CUT request as
// a server — the input half of dispatching an incoming CALL FUNCTION to a Go
// handler. It is the mirror of cpic.EncodeCutFunctionRequest and round-trips
// against it. See docs/porting-plan.md and docs/polyglot-rfc-server.md.

// Package rfcserver decodes inbound classic RFC (CUT) requests and (later)
// encodes their responses, for an RFC server that SAP calls into.
package rfcserver

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

// ErrRequest reports a malformed inbound CUT request.
var ErrRequest = errors.New("rfcserver: malformed CUT request")

var cutRequestPrefix = []byte{0x05, 0x02, 0x00, 0x00}

// Request is a decoded inbound CUT function call.
type Request struct {
	FunctionName     string
	KernelRelease    string
	RequestedOutputs []string
	Imports          []cpic.NamedValue
	Tables           []Table
	XrfcParameters   []cpic.NamedValue
}

// Table is one decoded inbound table parameter.
type Table struct {
	Name          string
	RowByteLength int
	Rows          [][]byte
}

// DecodeCutFunctionRequest decodes one inbound CUT request payload (the CPIC
// application data of a client's call), the mirror of
// cpic.EncodeCutFunctionRequest.
func DecodeCutFunctionRequest(payload []byte) (Request, error) {
	var req Request
	if len(payload) < len(cutRequestPrefix) || string(payload[:len(cutRequestPrefix)]) != string(cutRequestPrefix) {
		return req, fmt.Errorf("%w: bad CUT request prefix", ErrRequest)
	}
	decoded, err := cpic.DecodeFieldChainPrefix(payload[len(cutRequestPrefix):], uint16(cpic.TagContextEnd), uint16(cpic.TagEnd), cpic.FieldChainLimits{})
	if err != nil {
		return req, fmt.Errorf("%w: %v", ErrRequest, err)
	}

	var pendingParam string
	var havePendingParam bool
	var pendingXrfc string
	var haveXrfc bool
	var cur *Table
	var pendingXrfcData []byte

	for _, f := range decoded.Fields {
		switch cpic.Tag(f.Tag) {
		case cpic.TagKernel:
			req.KernelRelease = decodeUTF16LE(f.Value)
		case cpic.TagFunction:
			req.FunctionName = decodeUTF16LE(f.Value)
		case cpic.TagCallContext:
			// carries no application data for dispatch
		case cpic.TagRequestedOutput:
			req.RequestedOutputs = append(req.RequestedOutputs, decodeUTF16LE(f.Value))
		case cpic.TagParameterName:
			if havePendingParam {
				return req, fmt.Errorf("%w: parameter name without a value", ErrRequest)
			}
			pendingParam = decodeUTF16LE(f.Value)
			havePendingParam = true
		case cpic.TagParameterValue:
			if !havePendingParam {
				return req, fmt.Errorf("%w: parameter value without a name", ErrRequest)
			}
			req.Imports = append(req.Imports, cpic.NamedValue{Name: pendingParam, Value: append([]byte(nil), f.Value...)})
			havePendingParam = false
		case cpic.TagTableName:
			req.Tables = append(req.Tables, Table{Name: decodeUTF16LE(f.Value)})
			cur = &req.Tables[len(req.Tables)-1]
		case cpic.TagTableHeader:
			if cur == nil || len(f.Value) < 8 {
				return req, fmt.Errorf("%w: table header out of place", ErrRequest)
			}
			cur.RowByteLength = int(binary.BigEndian.Uint32(f.Value[0:]))
		case cpic.TagTableCompr:
			if cur == nil {
				return req, fmt.Errorf("%w: table row without a table", ErrRequest)
			}
			cur.Rows = append(cur.Rows, append([]byte(nil), f.Value...))
		case cpic.TagXRfcParameter:
			if haveXrfc {
				// second boundary closes the parameter
				req.XrfcParameters = append(req.XrfcParameters, cpic.NamedValue{Name: pendingXrfc, Value: pendingXrfcData})
				haveXrfc = false
				pendingXrfcData = nil
			} else {
				haveXrfc = true
				pendingXrfc = decodeUTF16LE(f.Value)
			}
		case cpic.TagXRfcData:
			pendingXrfcData = append(pendingXrfcData, f.Value...)
		case cpic.TagEnd:
			// terminal
		default:
			// tolerate unknown control fields
		}
	}
	if havePendingParam {
		return req, fmt.Errorf("%w: trailing parameter name without a value", ErrRequest)
	}
	if req.FunctionName == "" {
		return req, fmt.Errorf("%w: request lacks a function name", ErrRequest)
	}
	return req, nil
}

func decodeUTF16LE(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	n := len(b) / 2
	u := make([]uint16, n)
	for i := 0; i < n; i++ {
		u[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return strings.TrimRight(string(utf16.Decode(u)), " \x00")
}
