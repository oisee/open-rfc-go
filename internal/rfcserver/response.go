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

	"github.com/oisee/open-rfc-go/internal/classicrfc"
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

// EncodeCutFunctionResponseS4 encodes a successful response with the full
// S/4HANA classic envelope a live ABAP client requires to read the exports: the
// response context (0x0503), the session RFC GUID (0x0514, byte-swapped as
// server records carry it), the zero success control (0x0420), the call context
// (0x0512), an echo of the requested outputs (0x0205), then the export scalars
// and tables. guid is the client's connection GUID (init byte order); pass nil
// to omit it.
func EncodeCutFunctionResponseS4(exports []cpic.NamedValue, tables []Table, guid []byte, requestedOutputs []string) ([]byte, error) {
	fields := []cpic.Field{{Tag: uint16(cpic.TagResponseContext), Value: nil}}
	if len(guid) == 16 {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagSession), Value: swapRFCGUID(guid)})
	}
	fields = append(fields, cpic.Field{Tag: 0x0420, Value: []byte{0, 0, 0, 0}})
	fields = append(fields, cpic.Field{Tag: uint16(cpic.TagCallContext), Value: nil})
	// 0x0205 echoes only the outputs actually returned — the export names — not
	// the caller's full requested-output list (which also names imports). Echoing
	// an output with no matching value makes the client look for a missing param.
	_ = requestedOutputs
	for _, e := range exports {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagRequestedOutput), Value: encodeUTF16LE(e.Name)})
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
		for _, row := range t.Rows {
			fields = append(fields, cpic.Field{Tag: uint16(cpic.TagTableCompr), Value: append([]byte(nil), row...)})
		}
	}
	// Trailing S4 envelope: program name, control metric, S4 metadata block.
	program, err := classicrfc.EncodeAbapChar("OPEN_RFC_GO", 40)
	if err != nil {
		return nil, err
	}
	meta0104 := append([]byte(nil), s4Meta0104...)
	if len(guid) == 16 {
		// 0x0104[205] is a session-GUID-derived byte: the swapped GUID's first
		// byte minus 2 (established by diffing live sessions). The rest of 0x0104
		// (an 8-byte metric and a counter) varies per call and so cannot be a
		// value the client checks byte-exact.
		meta0104[205] = swapRFCGUID(guid)[0] - 2
	}
	fields = append(fields,
		cpic.Field{Tag: uint16(cpic.TagProgram), Value: program},
		cpic.Field{Tag: 0x0667, Value: append([]byte(nil), s4Metric0667...)},
		cpic.Field{Tag: 0x0104, Value: meta0104},
		cpic.Field{Tag: uint16(cpic.TagEnd), Value: nil},
	)

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

// EncodeCutFunctionResponseClassic mirrors the lean classic response a live A4H
// server actually sends for a normal function call (verified against .105:
// 0x0503, 0x0514 swapped GUID, 0x0420, 0x0512, per-export 0x0205+0x0201+0x0203,
// 0x0130 program, 0x0667 metric, End) — crucially WITHOUT the 0x0104 S4 block,
// whose stale baked GUIDs a live client rejects.
func EncodeCutFunctionResponseClassic(exports []cpic.NamedValue, tables []Table, guid []byte) ([]byte, error) {
	fields := []cpic.Field{{Tag: uint16(cpic.TagResponseContext), Value: nil}}
	if len(guid) == 16 {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagSession), Value: swapRFCGUID(guid)})
	}
	fields = append(fields, cpic.Field{Tag: 0x0420, Value: []byte{0, 0, 0, 0}})
	fields = append(fields, cpic.Field{Tag: uint16(cpic.TagCallContext), Value: nil})
	for _, e := range exports {
		fields = append(fields, cpic.Field{Tag: uint16(cpic.TagRequestedOutput), Value: encodeUTF16LE(e.Name)})
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
		for _, row := range t.Rows {
			fields = append(fields, cpic.Field{Tag: uint16(cpic.TagTableCompr), Value: append([]byte(nil), row...)})
		}
	}
	program, err := classicrfc.EncodeAbapChar("SAPLZTST", 40)
	if err != nil {
		return nil, err
	}
	fields = append(fields,
		cpic.Field{Tag: uint16(cpic.TagProgram), Value: program},
		cpic.Field{Tag: 0x0667, Value: append([]byte(nil), s4Metric0667...)},
		cpic.Field{Tag: uint16(cpic.TagEnd), Value: nil},
	)
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
