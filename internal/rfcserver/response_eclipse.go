// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
)

// The answer to an Eclipse call, in the shape the system gives one.
//
// Measured over the metadata and ADT exchanges of a live Eclipse↔A4H session
// (2026-09-15), the system's response chain is, in order:
//
//	0500                       start
//	0331 <first table number>  only when the call carried tables
//	0503                       response context
//	0514 <session GUID>        the one the call carried, echoed
//	0420 00000000              success
//	0512                       call context
//	0205 <name> …              one per output returned
//	0201 <name> 0203 <value> … scalar exports
//	3c02  3c05 … 3c02          an xRFC export: "<NAME>" first, then 1024-byte chunks
//	4000 0101  4002 … 4004     a compact export: raw DEFLATE of the document
//	0130 <program, 40 chars>
//	0335 0302 0303 … 0336      each table: number and rows, uncompressed
//	0667 <8 bytes>
//	0523                       after a compact export
//	ffff
//
// Tables go back uncompressed. The system compresses a table above some size
// (0305, SAP LZH) and sends a small one as it is (0303), so a client that reads
// the system reads both; the metadata tables ADT needs are eight kilobytes.
func EncodeEclipseResponse(resp Response, req Request) ([]byte, error) {
	var fields []cpic.Field
	if len(resp.Tables) > 0 && len(resp.Tables[0].ID) == 4 {
		fields = append(fields, cpic.Field{Tag: tagFirstTableID, Value: append([]byte(nil), resp.Tables[0].ID...)})
	}
	fields = append(fields, cpic.Field{Tag: uint16(cpic.TagResponseContext)})
	if len(req.SessionGUID) == 16 {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagSession), Value: append([]byte(nil), req.SessionGUID...)})
	}
	fields = append(fields,
		cpic.Field{Tag: uint16(cpic.TagUnresolved0420), Value: []byte{0, 0, 0, 0}},
		cpic.Field{Tag: uint16(cpic.TagCallContext)},
	)
	// the outputs, in the order the system returns them
	outputs := resp.Outputs
	if outputs == nil {
		for _, e := range resp.Exports {
			outputs = append(outputs, Output{Name: e.Name, Value: e.Value})
		}
		for _, x := range resp.XrfcParameters {
			outputs = append(outputs, Output{Name: x.Name, XML: true, Value: x.Value})
		}
	}
	for _, o := range outputs {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagRequestedOutput), Value: encodeUTF16LE(o.Name)})
	}
	if resp.Compact != nil {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagRequestedOutput), Value: encodeUTF16LE(resp.CompactName)})
	}
	for _, o := range outputs {
		if o.XML {
			fields = append(fields, xrfcSystemChunks(cpic.NamedValue{Name: o.Name, Value: o.Value})...)
			continue
		}
		name, err := encodeName(o.Name)
		if err != nil {
			return nil, err
		}
		fields = append(fields,
			cpic.Field{Tag: uint16(cpic.TagParameterName), Value: name},
			cpic.Field{Tag: uint16(cpic.TagParameterValue), Value: append([]byte(nil), o.Value...)},
		)
	}
	if resp.Compact != nil {
		deflated, err := deflate(resp.Compact)
		if err != nil {
			return nil, err
		}
		fields = append(fields, cpic.Field{Tag: tagCompactFlag, Value: []byte{0x01, 0x01}})
		for off := 0; off < len(deflated); off += compactChunkLength {
			end := min(off+compactChunkLength, len(deflated))
			fields = append(fields, cpic.Field{Tag: tagCompactResponse, Value: deflated[off:end]})
		}
		fields = append(fields, cpic.Field{Tag: tagCompactEnd})
	}
	program, err := classicrfc.EncodeAbapChar("OPEN_RFC_GO", 40)
	if err != nil {
		return nil, err
	}
	fields = append(fields, cpic.Field{Tag: uint16(cpic.TagProgram), Value: program})
	for _, t := range resp.Tables {
		if len(t.ID) != 4 {
			return nil, fmt.Errorf("%w: table %s has no number to answer by", ErrRequest, t.Name)
		}
		header := make([]byte, 12)
		binary.BigEndian.PutUint32(header[0:], 0x0a)
		copy(header[4:], t.ID)
		binary.BigEndian.PutUint32(header[8:], uint32(len(t.Rows)))
		geometry := make([]byte, 8)
		binary.BigEndian.PutUint32(geometry[0:], uint32(t.RowByteLength))
		binary.BigEndian.PutUint32(geometry[4:], uint32(len(t.Rows)))
		fields = append(fields,
			cpic.Field{Tag: tagTableIDHeader, Value: header},
			cpic.Field{Tag: uint16(cpic.TagTableHeader), Value: geometry},
		)
		for i, row := range t.Rows {
			if len(row) != t.RowByteLength {
				return nil, fmt.Errorf("%w: %s row %d has %d bytes, want %d", ErrRequest, t.Name, i, len(row), t.RowByteLength)
			}
			fields = append(fields, cpic.Field{Tag: tagTableRow, Value: append([]byte(nil), row...)})
		}
		fields = append(fields, cpic.Field{Tag: tagTableIDEnd, Value: append([]byte(nil), t.ID...)})
	}
	fields = append(fields, cpic.Field{Tag: 0x0667, Value: append([]byte(nil), s4Metric0667...)})
	if resp.Compact != nil {
		fields = append(fields, cpic.Field{Tag: tagCompactClose})
	}
	fields = append(fields, cpic.Field{Tag: uint16(cpic.TagEnd)})

	chain, err := cpic.EncodeFieldChain(uint16(cpic.TagResponseStart), fields, cpic.FieldChainLimits{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequest, err)
	}
	out := make([]byte, 0, len(cutResponsePrefix)+len(chain)+2)
	out = append(out, cutResponsePrefix...)
	out = append(out, chain...)
	out = append(out, 0xff, 0xff)
	return out, nil
}

// xrfcSystemChunks frames an xRFC export the way the system does: the root
// open tag alone in the first chunk, the rest in chunks of 1024.
func xrfcSystemChunks(x cpic.NamedValue) []cpic.Field {
	fields := []cpic.Field{{Tag: uint16(cpic.TagXRfcParameter)}}
	open := "<" + x.Name + ">"
	rest := x.Value
	if bytes.HasPrefix(rest, []byte(open)) {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagXRfcData), Value: []byte(open)})
		rest = rest[len(open):]
	}
	for off := 0; off < len(rest); off += 1024 {
		end := min(off+1024, len(rest))
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagXRfcData), Value: append([]byte(nil), rest[off:end]...)})
	}
	return append(fields, cpic.Field{Tag: uint16(cpic.TagXRfcParameter)})
}

// deflate writes a raw DEFLATE stream: no zlib header, no gzip wrapper, which
// is what the system sends after 4000 01 01 and what Eclipse reads.
func deflate(doc []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(doc); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// eclipseRecords wraps a response for the wire, in the record header the
// gateway gives a response — not the one an SM59 client is answered with.
//
// The header is a near-constant frame measured over every response of a live
// session (125 of them): gwid 6, time 00010000, timeout 000001f4, info4 2,
// sequence 1, and a 32-byte operation-info block whose first four bytes hold
// the message length, its content without the eight-byte final SAP trailer,
// and whose last four are 00060002. Our answers carried none of it — an
// all-zero header the gateway never produces — and Eclipse refused them at the
// CPIC receive with "CMRCV; null", which is a transport refusal, not a
// complaint about the answer inside.
//
// A record carries at most 28000 bytes; a longer answer is an F_SAP_SEND that
// is not final followed by F_RECEIVE records, the last final. The final piece
// is never shorter than the eight-byte trailer, so a short tail is topped up
// from the record before it.
func eclipseRecords(cut, convID []byte, uid uint16) ([][]byte, error) {
	total := len(cut)
	if total <= maxRecordData {
		return [][]byte{buildEclipseRecord(cut, convID, uid, appcFSapSend, true, total)}, nil
	}
	var pieces [][]byte
	for off := 0; off < total; off += maxRecordData {
		pieces = append(pieces, cut[off:min(off+maxRecordData, total)])
	}
	if last := pieces[len(pieces)-1]; len(last) < 8 {
		prev := pieces[len(pieces)-2]
		take := 8 - len(last)
		pieces[len(pieces)-1] = append(append([]byte(nil), prev[len(prev)-take:]...), last...)
		pieces[len(pieces)-2] = prev[:len(prev)-take]
	}
	records := make([][]byte, 0, len(pieces))
	for i, piece := range pieces {
		final := i == len(pieces)-1
		fn := byte(appcFSapSend)
		if i > 0 {
			fn = appcFReceive
		}
		records = append(records, buildEclipseRecord(piece, convID, uid, fn, final, total))
	}
	return records, nil
}

// appcFReceive is F_RECEIVE, the verb a continued response uses after the first
// F_SAP_SEND record.
const appcFReceive = 0x09

// buildEclipseRecord frames one response record with the gateway's own header.
//
// The operation-info block is the one the ADT gateway responses carry (63 of
// them, all identical but for the length): a leading 00006d60 00000002, this
// record's data length, then 00000001 00000000, the gateway protocol level
// "4103", and a 00000002 tail. The length is the record's own data, not the
// message minus a trailer — that was a different protocol's frame, and Eclipse
// aborted the conversation (an F_0x0b reply) when it saw it.
func buildEclipseRecord(data, convID []byte, uid uint16, fn byte, final bool, msgLen int) []byte {
	rec := make([]byte, appcHeaderLen+len(data))
	h := rec[:appcHeaderLen]
	h[0] = appcProtocol // 0x06
	h[1] = fn
	h[2] = 0x02 // protocol
	binary.BigEndian.PutUint16(h[4:], uid)
	binary.BigEndian.PutUint16(h[6:], 0x0007) // gateway id
	binary.BigEndian.PutUint32(h[12:], 0x00010000)
	h[16] = 0x01                                   // info3: as every ADT response carries it
	binary.BigEndian.PutUint32(h[17:], 0xffffffff) // timeout
	h[21] = 0x02                                   // info4
	binary.BigEndian.PutUint32(h[22:], 1)          // sequence
	if final {
		binary.BigEndian.PutUint16(h[26:], 8) // final SAP parameter length
		h[30] = 0x05                          // info: last record of a message
		h[31] = 0x0c                          // vector
	} else {
		h[30] = 0x01
		h[31] = 0x08
	}
	copy(h[40:48], convID)
	op := h[48:80]
	binary.BigEndian.PutUint32(op[0:], 0x00006d60)
	binary.BigEndian.PutUint32(op[4:], 0x00000002)
	binary.BigEndian.PutUint32(op[8:], uint32(len(data))) // this record's data length
	binary.BigEndian.PutUint32(op[12:], 0x00000001)
	// op[16:20] zero
	copy(op[20:], []byte{0x00, 0x34, 0x31, 0x30, 0x33}) // 00 "4103"
	// the last word varies across responses (a counter the client does not
	// check); 00010000 is the value most of them carry
	binary.BigEndian.PutUint32(op[28:], 0x00010000)
	copy(rec[appcHeaderLen:], data)
	return rec
}
