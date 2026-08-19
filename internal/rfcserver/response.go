// SPDX-License-Identifier: Apache-2.0
//
// Server-side response encoder. Original work for open-rfc-go (milestone 8):
// the mirror of internal/cpic's response decoder. It builds the CPIC
// application data a server returns for a successfully dispatched CUT call —
// the zero 0x0420 success control, the export scalars, and the tables — so a
// client's cpic.DecodeFunctionResultFields reads it as a success.

package rfcserver

import (
	"encoding/binary"
	"fmt"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

var cutResponsePrefix = []byte{0x05, 0x00, 0x00, 0x00}

// EncodeCutFunctionResponse encodes a successful CUT response carrying the given
// export scalars and tables. Scalar/table names travel as UTF-16LE; the zero
// 0x0420 control marks success.
func EncodeCutFunctionResponse(exports []cpic.NamedValue, tables []Table) ([]byte, error) {
	fields := []cpic.Field{
		{Tag: 0x0420, Value: []byte{0, 0, 0, 0}}, // zero success control
	}
	for _, e := range exports {
		name, err := encodeName(e.Name)
		if err != nil {
			return nil, err
		}
		fields = append(fields,
			cpic.Field{Tag: uint16(cpic.TagParameterName), Value: name},
			cpic.Field{Tag: uint16(cpic.TagParameterValue), Value: append([]byte(nil), e.Value...)},
		)
	}
	for _, t := range tables {
		name, err := encodeName(t.Name)
		if err != nil {
			return nil, err
		}
		header := make([]byte, 8)
		binary.BigEndian.PutUint32(header[0:], uint32(t.RowByteLength))
		binary.BigEndian.PutUint32(header[4:], uint32(len(t.Rows)))
		fields = append(fields,
			cpic.Field{Tag: uint16(cpic.TagTableName), Value: name},
			cpic.Field{Tag: uint16(cpic.TagTableHeader), Value: header},
		)
		for i, row := range t.Rows {
			if t.RowByteLength != 0 && len(row) != t.RowByteLength {
				return nil, fmt.Errorf("%w: %s row %d has %d bytes, want %d", ErrRequest, t.Name, i, len(row), t.RowByteLength)
			}
			fields = append(fields, cpic.Field{Tag: uint16(cpic.TagTableCompr), Value: append([]byte(nil), row...)})
		}
	}
	fields = append(fields, cpic.Field{Tag: uint16(cpic.TagEnd), Value: nil})

	chain, err := cpic.EncodeFieldChain(uint16(cpic.TagResponseStart), fields, cpic.FieldChainLimits{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequest, err)
	}
	out := make([]byte, 0, len(cutResponsePrefix)+len(chain)+2)
	out = append(out, cutResponsePrefix...)
	out = append(out, chain...)
	out = append(out, 0xff, 0xff) // the 2-byte End trailer
	return out, nil
}

func encodeName(name string) ([]byte, error) {
	if name == "" || len(name) > 30 {
		return nil, fmt.Errorf("%w: parameter name %q must be 1..30 characters", ErrRequest, name)
	}
	return encodeUTF16LE(name), nil
}

func encodeUTF16LE(s string) []byte {
	rs := []rune(s)
	out := make([]byte, 0, len(rs)*2)
	var tmp [2]byte
	for _, r := range rs {
		if r > 0xffff {
			r = 0xfffd // BMP-only names in practice; replace astral defensively
		}
		binary.LittleEndian.PutUint16(tmp[:], uint16(r))
		out = append(out, tmp[0], tmp[1])
	}
	return out
}

// EncodeCutFunctionExceptionResponse encodes a declared-exception response with
// the given exception key, which a client decodes as an ABAP exception.
func EncodeCutFunctionExceptionResponse(exceptionKey string) ([]byte, error) {
	if exceptionKey == "" {
		return nil, fmt.Errorf("%w: exception key must not be empty", ErrRequest)
	}
	fields := []cpic.Field{
		{Tag: 0x0401, Value: encodeUTF16LE(exceptionKey)}, // TagExceptionKey
		{Tag: uint16(cpic.TagEnd), Value: nil},
	}
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
