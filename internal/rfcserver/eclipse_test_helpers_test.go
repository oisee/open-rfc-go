// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"encoding/binary"
	"testing"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

// What Eclipse puts on the wire, built by hand from the measured layout so the
// decoder under test is not checked against its own encoder.

// eclipseCall frames a call the way Eclipse does: the opening field declares
// big-endian, the session GUID is carried, names are big-endian, and the chain
// begins without the leading "previous tag" bytes.
type eclipseCallInput struct {
	function   string
	guid       []byte
	outputs    []string
	imports    []cpic.NamedValue // values already in Eclipse's byte order
	tables     []Table           // Name, RowByteLength, ID
	compact    []byte
	compressed bool
}

func eclipseCall(t *testing.T, in eclipseCallInput) []byte {
	t.Helper()
	fields := []cpic.Field{
		{Tag: uint16(cpic.TagStart), Value: []byte{0x04, 0x02, 0x01, 0x05, 0x04, 0x01, 0x00, 0x02}},
		{Tag: uint16(cpic.TagProtocolVersion), Value: []byte{0, 0, 0x0e, 0x0b}},
	}
	if in.guid != nil {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagSession), Value: in.guid})
	}
	fields = append(fields,
		cpic.Field{Tag: uint16(cpic.TagContextEnd)},
		cpic.Field{Tag: uint16(cpic.TagKernel), Value: utf16BE("754")},
		cpic.Field{Tag: uint16(cpic.TagFunction), Value: utf16BE(in.function)},
		cpic.Field{Tag: uint16(cpic.TagCallContext)},
	)
	for _, o := range in.outputs {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagRequestedOutput), Value: utf16BE(o)})
	}
	for _, p := range in.imports {
		fields = append(fields,
			cpic.Field{Tag: uint16(cpic.TagParameterName), Value: utf16BE(p.Name)},
			cpic.Field{Tag: uint16(cpic.TagParameterValue), Value: p.Value},
		)
	}
	for _, tb := range in.tables {
		geometry := make([]byte, 8)
		binary.BigEndian.PutUint32(geometry, uint32(tb.RowByteLength))
		fields = append(fields,
			cpic.Field{Tag: uint16(cpic.TagTableName), Value: utf16BE(tb.Name)},
			cpic.Field{Tag: tagTableID, Value: tb.ID},
			cpic.Field{Tag: uint16(cpic.TagTableHeader), Value: geometry},
		)
	}
	if in.compact != nil {
		payload := in.compact
		flag := byte(0x00)
		if in.compressed {
			var err error
			payload, err = deflate(in.compact)
			if err != nil {
				t.Fatal(err)
			}
			flag = 0x01
		}
		fields = append(fields, cpic.Field{Tag: tagCompactFlag, Value: []byte{0x01, flag}})
		for off := 0; off < len(payload); off += compactChunkLength {
			end := min(off+compactChunkLength, len(payload))
			fields = append(fields, cpic.Field{Tag: tagCompactRequest, Value: payload[off:end]})
		}
		fields = append(fields, cpic.Field{Tag: tagCompactEnd})
	}
	fields = append(fields, cpic.Field{Tag: uint16(cpic.TagEnd)})
	chain, err := cpic.EncodeFieldChain(0x0000, fields, cpic.FieldChainLimits{})
	if err != nil {
		t.Fatal(err)
	}
	return chain[2:] // Eclipse does not write the first "previous tag"
}

// responseFields reads the chain of a response the server encoded.
func responseFields(t *testing.T, cut []byte) []cpic.Field {
	t.Helper()
	if len(cut) < 6 || string(cut[:4]) != string(cutResponsePrefix) {
		t.Fatalf("response does not open with the CUT response prefix: %x", cut[:min(8, len(cut))])
	}
	decoded, err := cpic.DecodeFieldChainPrefix(cut[4:], uint16(cpic.TagResponseStart), uint16(cpic.TagEnd), cpic.FieldChainLimits{})
	if err != nil {
		t.Fatalf("response chain: %v", err)
	}
	return decoded.Fields
}

func tagsOf(fields []cpic.Field) []uint16 {
	out := make([]uint16, len(fields))
	for i, f := range fields {
		out[i] = f.Tag
	}
	return out
}

func sameTags(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func valuesOf(fields []cpic.Field, tag uint16) [][]byte {
	var out [][]byte
	for _, f := range fields {
		if f.Tag == tag {
			out = append(out, f.Value)
		}
	}
	return out
}
