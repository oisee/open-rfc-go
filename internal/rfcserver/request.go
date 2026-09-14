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
	"github.com/oisee/open-rfc-go/internal/xrfc"
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
	if !hasCutRequestPrefix(payload) {
		return req, fmt.Errorf("%w: bad CUT request prefix", ErrRequest)
	}
	return decodeFunctionRequest(payload[len(cutRequestPrefix):], uint16(cpic.TagContextEnd))
}

// DecodeFunctionRequest decodes a call however it is framed: with the CUT
// prefix, or the way Eclipse sends one.
//
// The chain grammar writes, for each field, the *previous* field's tag, then
// the tag, the length and the value — so a decoder can tell a dropped or
// reordered field from a truncated one. A CUT request opens with a 0x0502
// field of length zero, which is what the four-byte prefix is, and every
// record after it names its predecessor.
//
// Eclipse omits the leading record. Its first call reads
//
//	0101 0008 0402010504010002    tag 0x0101, length 8, no predecessor named
//	0101 | 0103 0004 00000e0b     predecessor 0x0101, tag 0x0103, length 4
//	0103 | 0106 000b ...          predecessor 0x0103, tag 0x0106, length 11
//
// which is the same grammar with the first predecessor left off — and 0x0101
// is TagStart, so there is genuinely nothing before it to name. Supplying the
// two bytes it does not send lets the existing decoder read the rest
// unchanged, which is worth more than a second decoder that drifts from this
// one.
func DecodeFunctionRequest(payload []byte) (Request, error) {
	if hasCutRequestPrefix(payload) {
		return decodeFunctionRequest(payload[len(cutRequestPrefix):], uint16(cpic.TagContextEnd))
	}
	chain := make([]byte, 0, len(payload)+2)
	chain = append(chain, 0x00, 0x00)
	chain = append(chain, payload...)
	return decodeFunctionRequest(chain, 0x0000)
}

func decodeFunctionRequest(body []byte, initialPreviousTag uint16) (Request, error) {
	var req Request
	decoded, err := cpic.DecodeFieldChainPrefix(body, initialPreviousTag, uint16(cpic.TagEnd), cpic.FieldChainLimits{})
	if err != nil {
		return req, fmt.Errorf("%w: %v", ErrRequest, err)
	}

	var pendingParam string
	var havePendingParam bool
	var pendingXrfc string
	var haveXrfc bool
	var cur *Table
	var pendingXrfcData []byte

	// The sender says which way round it writes text, in the opening field.
	// TagStart is the first field of every chain, so this is known before any
	// name is read.
	names := utf16Order(decoded.Fields)

	for _, f := range decoded.Fields {
		switch cpic.Tag(f.Tag) {
		case cpic.TagKernel:
			req.KernelRelease = decodeUTF16(f.Value, names)
		case cpic.TagFunction:
			req.FunctionName = decodeUTF16(f.Value, names)
		case cpic.TagCallContext:
			// carries no application data for dispatch
		case cpic.TagRequestedOutput:
			req.RequestedOutputs = append(req.RequestedOutputs, decodeUTF16(f.Value, names))
		case cpic.TagParameterName:
			if havePendingParam {
				return req, fmt.Errorf("%w: parameter name without a value", ErrRequest)
			}
			pendingParam = decodeUTF16(f.Value, names)
			havePendingParam = true
		case cpic.TagParameterValue:
			if !havePendingParam {
				return req, fmt.Errorf("%w: parameter value without a name", ErrRequest)
			}
			req.Imports = append(req.Imports, cpic.NamedValue{Name: pendingParam, Value: append([]byte(nil), f.Value...)})
			havePendingParam = false
		case cpic.TagTableName:
			req.Tables = append(req.Tables, Table{Name: decodeUTF16(f.Value, names)})
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
				// The closing boundary, and the first moment the name can be
				// known. EncodeCutFunctionRequest writes both boundaries with
				// no value at all, so the name never crosses the wire as a
				// field: the only place it survives is the root element of
				// the XML in between. Reading the boundary gives "" and a
				// server then cannot tell one parameter from another, which
				// starts to matter as soon as a function has two.
				name, err := xrfc.DecodeRecursiveParameterName(pendingXrfcData, xrfc.RecursiveLimits{})
				if err != nil {
					return req, fmt.Errorf("%w: xRFC parameter name: %v", ErrRequest, err)
				}
				if pendingXrfc != "" && pendingXrfc != name {
					return req, fmt.Errorf("%w: xRFC boundary names %q and the XML names %q",
						ErrRequest, pendingXrfc, name)
				}
				req.XrfcParameters = append(req.XrfcParameters, cpic.NamedValue{Name: name, Value: pendingXrfcData})
				haveXrfc = false
				pendingXrfcData = nil
			} else {
				haveXrfc = true
				// kept only to be checked against the XML: a boundary that
				// does carry a name must not disagree with it
				pendingXrfc = decodeUTF16(f.Value, names)
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

// utf16Order reads the byte order out of the chain's opening field.
//
// Measured over one captured Eclipse session, 824 frames carrying an opening
// field: every one of Eclipse's 444 has 0x02 at index 1 and every one of the
// system's 380 has 0x01, and that is exactly the split between the text each
// sends — Eclipse big-endian, the system little-endian. Nothing else in the
// field varies with it.
//
//	Eclipse    04 02 01 05 04 01 00 02
//	the system 04 01 01 05 04 01 00 03
//	              ^^
//
// A correlation from one client pair is not a specification, so an unexpected
// value falls back to looking at where the zero bytes fall rather than
// insisting. What is *not* done is the reverse — treating the guess as the
// rule — because the two orders do not produce a garbled string, they produce
// a different one, and a wrong function name arrives as an ordinary lookup
// miss with nothing in it to suggest an encoding.
//
// The codepage field was the other candidate and it is not this: tag 0x0016
// appears only in the system's direction, 380 times, always "1100" — Latin-1,
// the system describing itself. Eclipse never sends one.
func utf16Order(fields []cpic.Field) binary.ByteOrder {
	for _, f := range fields {
		if cpic.Tag(f.Tag) != cpic.TagStart || len(f.Value) < 2 {
			continue
		}
		switch f.Value[1] {
		case 0x01:
			return binary.LittleEndian
		case 0x02:
			return binary.BigEndian
		}
		break
	}
	return nil
}

// decodeUTF16 reads one UTF-16 name in the order the sender declared, or, when
// it declared nothing this decoder recognises, in the order its zero bytes
// imply.
//
// Names only. Every caller is a function, parameter, table or xRFC parameter
// name, and those are ASCII, which is what makes the fallback decidable. A
// parameter's *value* never comes through here: it stays bytes, and what it
// means is the handler's business.
func decodeUTF16(b []byte, order binary.ByteOrder) string {
	if order == nil {
		if utf16LooksBigEndian(b[:len(b)/2*2]) {
			order = binary.BigEndian
		} else {
			order = binary.LittleEndian
		}
	}
	if len(b) == 0 {
		return ""
	}
	n := len(b) / 2
	u := make([]uint16, n)
	for i := 0; i < n; i++ {
		u[i] = order.Uint16(b[2*i:])
	}
	return strings.TrimRight(string(utf16.Decode(u)), " \x00")
}

// decodeUTF16LE reads a UTF-16 field in whichever order the sender wrote it.
//
// The name is now half a lie and kept anyway, because the little-endian case
// is still what this server writes. The two directions genuinely differ, and
// the capture says so: across 1774 server frames every string A4H sends is
// little-endian, and across the whole conversation Eclipse sends the function
// name big-endian 374 times and little-endian never. Each side writes in its
// own order — A4H on x86, Eclipse in Java, where big-endian is the default —
// and each decodes the other's.
//
// Reading Eclipse's names as little-endian is not a garbled string, it is a
// different string: "SADT_REST_RFC_ENDPOINT" arrives and becomes a run of CJK
// characters, the dispatcher finds no such function, and the answer is
// FU_NOT_FOUND — a lookup miss, with nothing about it to suggest an encoding.
//
// The order is detected rather than configured. Every name in this protocol is
// ASCII, so exactly one byte of each pair is zero and which one says the
// order, with no negotiation to get wrong and no per-connection state to
// carry. The honest limit: a genuinely non-ASCII value defeats it, and the
// principled fix is to read the byte order the logon declares — a field this
// server does not yet decode. Nothing in ADT has needed it so far.
func decodeUTF16LE(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	n := len(b) / 2
	order := binary.ByteOrder(binary.LittleEndian)
	if utf16LooksBigEndian(b[:n*2]) {
		order = binary.BigEndian
	}
	u := make([]uint16, n)
	for i := 0; i < n; i++ {
		u[i] = order.Uint16(b[2*i:])
	}
	return strings.TrimRight(string(utf16.Decode(u)), " \x00")
}

// utf16LooksBigEndian reports which half of each pair holds the zero byte.
//
// Undecidable on its own for text that is not ASCII, and for an empty or
// all-zero field either answer gives the same string, so those fall through to
// little-endian.
func utf16LooksBigEndian(b []byte) bool {
	var low, high int
	for i := 0; i+1 < len(b); i += 2 {
		if b[i] == 0 && b[i+1] != 0 {
			high++ // zero first: big-endian
		}
		if b[i+1] == 0 && b[i] != 0 {
			low++ // zero second: little-endian
		}
	}
	return high > low
}

// hasCutRequestPrefix reports whether a payload carries the CUT prefix.
func hasCutRequestPrefix(payload []byte) bool {
	return len(payload) >= len(cutRequestPrefix) &&
		string(payload[:len(cutRequestPrefix)]) == string(cutRequestPrefix)
}

// eclipseLogonPrefix is "RFC" in EBCDIC, which is what a logon opens with.
//
// This is the field that tells a logon from a call, and finding it took two
// wrong guesses. First the logon was identified by its length, which held
// until a call was a similar size. Then by the *absence* of the CUT prefix,
// which was worse: Eclipse's calls have no CUT prefix either, so every call
// after the logon was answered with a logon template, and Eclipse said
// "Response status code is not an integer value" — it had been handed a logon
// and asked to read an HTTP status out of it.
//
// The wire says it plainly. A logon opens d9 c6 c3 f0 f0 … : "RFC000000000" in
// EBCDIC, the classic header. A call opens with a field chain. So the test is
// what the payload *is*, not what it is not.
var eclipseLogonPrefix = []byte{0xd9, 0xc6, 0xc3}

// isEclipseLogon reports whether this payload is a logon rather than a call.
func isEclipseLogon(payload []byte) bool {
	return len(payload) >= len(eclipseLogonPrefix) &&
		string(payload[:len(eclipseLogonPrefix)]) == string(eclipseLogonPrefix)
}
