// SPDX-License-Identifier: Apache-2.0
//
// Part of the cpic package; see cpic.go for the file's provenance header.

package cpic

import (
	"crypto/rand"
	"encoding/binary"
)

func randRead(b []byte) (int, error) { return rand.Read(b) }

const initialMaxTextCoordinateByteLength = 255

// grammarCoordinate is one coordinate of the initial RFCPING composite grammar.
// A coordinate declares exactly one of an exact byteLength (a control pinned to
// its width) or a maxByteLength (a name/address that varies with the endpoint).
type grammarCoordinate struct {
	tag           uint16
	byteLength    int
	hasByteLength bool
	maxByteLength int
	optional      bool
}

func exact(tag uint16, n int, optional bool) grammarCoordinate {
	return grammarCoordinate{tag: tag, byteLength: n, hasByteLength: true, optional: optional}
}
func bounded(tag uint16, optional bool) grammarCoordinate {
	return grammarCoordinate{tag: tag, maxByteLength: initialMaxTextCoordinateByteLength, optional: optional}
}

var initialRichPreambleGrammar = []grammarCoordinate{
	exact(uint16(TagProtocolVersion), 4, false),
	exact(uint16(TagCapabilities), 11, false),
	exact(uint16(TagLogonStatus), 1, true),
	exact(uint16(TagSystemCodePage), 8, false),
	exact(initialCpicUnresolved0450, 6, true),
	exact(0x0451, 20, true),
	exact(0x0452, 4, true),
	bounded(0x0453, true),
	bounded(uint16(TagClientAddress), false),
	exact(0x0020, 92, true),
	exact(0x0021, 20, true),
	bounded(uint16(TagPartnerSystem), false),
	bounded(uint16(TagPartnerHost), false),
	exact(uint16(TagConnectionType), 2, false),
	exact(uint16(TagKernelPatch), 8, false),
	exact(uint16(TagKernelRelease), 8, false),
	bounded(uint16(TagDestination), true),
	bounded(uint16(TagProgram), false),
	exact(0x0150, 24, false),
	exact(0x0151, 6, false),
	exact(0x0152, 2, false),
}

var initialRichResponseGrammar = []grammarCoordinate{
	exact(uint16(TagResponseStart), 0, false),
	exact(uint16(TagResponseContext), 0, false),
	exact(uint16(TagSession), 16, false),
	exact(uint16(TagUnresolved0420), 4, false),
	exact(uint16(TagCallContext), 0, false),
	bounded(uint16(TagProgram), false),
	exact(0x0667, 8, false),
	exact(0x0126, 4, true),
	exact(uint16(TagEnd), 0, false),
}

var initialRichGrammar = append(append([]grammarCoordinate{}, initialRichPreambleGrammar...), initialRichResponseGrammar...)

var initialRichTagLimits = func() map[uint16]int {
	m := map[uint16]int{}
	for _, c := range initialRichGrammar {
		m[c.tag]++
	}
	return m
}()

type grammarMatch struct {
	preambleFieldCount  int
	embeddedAllowedTags []uint16
}

func matchesCoordinate(field Field, c grammarCoordinate) bool {
	if field.Tag != c.tag {
		return false
	}
	if c.hasByteLength {
		return len(field.Value) == c.byteLength
	}
	return len(field.Value) >= 1 && len(field.Value) <= c.maxByteLength
}

func matchInitialRichGrammar(fields []Field) (grammarMatch, bool) {
	fieldIndex := 0
	preambleFieldCount := -1
	var embeddedAllowedTags []uint16
	for _, c := range initialRichGrammar {
		if fieldIndex < len(fields) && matchesCoordinate(fields[fieldIndex], c) {
			if c.tag == uint16(TagResponseStart) {
				preambleFieldCount = fieldIndex
			}
			if c.tag == 0x0126 {
				embeddedAllowedTags = append(embeddedAllowedTags, 0x0126)
			}
			fieldIndex++
			continue
		}
		if c.optional {
			continue
		}
		return grammarMatch{}, false
	}
	if fieldIndex != len(fields) || preambleFieldCount < 0 {
		return grammarMatch{}, false
	}
	return grammarMatch{preambleFieldCount: preambleFieldCount, embeddedAllowedTags: embeddedAllowedTags}, true
}

func hasInitialCompositeDuplicate(fields []Field) bool {
	actual := map[uint16]int{}
	for _, f := range fields {
		actual[f.Tag]++
	}
	for tag, limit := range initialRichTagLimits {
		if actual[tag] > limit {
			return true
		}
	}
	return false
}

func decodeRichInitialRfcPingResponse(fields []Field, structural []StructuralField, negotiated uint32) (DecodedInitialLogonResponse, error) {
	var zero DecodedInitialLogonResponse
	ends := filterTag(fields, uint16(TagEnd))
	if len(ends) != 1 || fields[len(fields)-1].Tag != uint16(TagEnd) || len(ends[0].Value) != 0 {
		return zero, failStructure("invalid-end-field", "initial CPIC RFCPING composite response has an invalid End field", structural)
	}
	oneByte := filterTag(fields, uint16(TagLogonStatus))
	if len(oneByte) > 1 || (len(oneByte) == 1 && len(oneByte[0].Value) != 1) {
		return zero, failStructure("malformed-one-byte-status", "initial CPIC RFCPING composite response has malformed logon status", structural)
	}
	callStatuses := filterTag(fields, uint16(TagUnresolved0420))
	if len(callStatuses) != 1 || len(callStatuses[0].Value) != 4 {
		return zero, failStructure("malformed-call-status", "initial CPIC RFCPING composite response has malformed call status", structural)
	}
	if hasInitialCompositeDuplicate(fields) {
		return zero, failStructure("duplicate-control-field", "initial CPIC RFCPING composite response has a duplicate field", structural)
	}
	status := 0
	if len(oneByte) == 1 {
		status = int(oneByte[0].Value[0])
	}
	match, ok := matchInitialRichGrammar(fields)
	if !ok {
		rule := "unsupported-field"
		if status == 0 {
			rule = "unsupported-field-zero-logon-status"
		}
		return zero, failStructure(rule, "initial CPIC RFCPING composite response does not match the bounded composite shape", structural)
	}
	if status != 0 {
		statusCopy := status
		return DecodedInitialLogonResponse{Success: false, Status: &statusCopy, NegotiatedProtocolVersion: negotiated, Fields: tagLens(fields)}, nil
	}
	if binary.BigEndian.Uint32(callStatuses[0].Value) != 0 {
		return zero, failStructure("nonzero-call-status", "initial CPIC RFCPING composite response has nonzero call status", structural)
	}
	embedded, err := decodeFunctionResponseFields(fields[match.preambleFieldCount+1:], match.embeddedAllowedTags)
	if err != nil {
		return zero, err
	}
	if !embedded.Success {
		return zero, failStructure("nonzero-call-status", "initial CPIC RFCPING composite response is not successful", structural)
	}
	statusCopy := status
	return DecodedInitialLogonResponse{Success: true, Status: &statusCopy, NegotiatedProtocolVersion: negotiated, Fields: tagLens(fields)}, nil
}
