// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"

	"github.com/oisee/open-rfc-go/internal/appc"
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

// eclipseRecords wraps a response for the wire, in as many APPC records as it
// takes.
//
// A data record carries at most 28000 bytes. The system sends a longer answer
// as an F_SAP_SEND that is not final, then F_RECEIVE records, the last of them
// final: measured on a 36 KB discovery document, which arrived as 28000 + 7996.
// The final record's last eight bytes are the chain's end field and the
// trailer, which the header counts as SAP parameters, so the last piece is
// never shorter than that.
func eclipseRecords(cut, convID []byte, uid uint16) ([][]byte, error) {
	if len(cut) <= maxRecordData {
		one, err := wrapFSapSend(cut, convID, uid)
		if err != nil {
			return nil, err
		}
		return [][]byte{one}, nil
	}
	var pieces [][]byte
	for off := 0; off < len(cut); off += maxRecordData {
		pieces = append(pieces, cut[off:min(off+maxRecordData, len(cut))])
	}
	if last := pieces[len(pieces)-1]; len(last) < 8 {
		prev := pieces[len(pieces)-2]
		take := 8 - len(last)
		pieces[len(pieces)-1] = append(append([]byte(nil), prev[len(prev)-take:]...), last...)
		pieces[len(pieces)-2] = prev[:len(prev)-take]
	}
	records := make([][]byte, 0, len(pieces))
	gwID := uint16(1)
	timeout := int32(-1)
	for i, piece := range pieces {
		final := i == len(pieces)-1
		fn := appc.FuncSapSend
		if i > 0 {
			fn = appc.FuncReceive
		}
		t := uint32(i+1) << 16
		rec, err := appc.EncodeDataRecord(appc.DataRecordInput{
			RecordHeaderInput: appc.RecordHeaderInput{
				UID: &uid, GatewayID: &gwID, ConversationID: convID, Timeout: &timeout, Time: &t,
			},
			FunctionCode: &fn,
			Data:         piece,
			IsFinal:      &final,
		})
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}
