// SPDX-License-Identifier: Apache-2.0

package fastser

import "bytes"

// The field-description list.
//
// A parameter is announced by a header, a type descriptor, and then one
// description per field:
//
//	0x44 'D' <fieldcount:1>
//	0x50 'P' <len:1> "\TYPE=<name>"
//	repeated fieldcount times:
//	    <typecode:1> [<width:2 LE>] <namelen:1> <NAME>
//
// The type code is a single byte and the codes below are the ones the captures
// establish. A code that is width-parameterised carries a two-byte operand; one
// whose width follows from the code itself does not.
//
// This corrects an earlier reading of this package. The byte before a field name
// was taken to be a "name tag" because 0x03 sat there in every int4 capture — it
// is in fact the type code for INT4, sitting in the same slot as 0x01 for INT1
// and 0x02 for INT2, exactly where DDIC says those fields are.
//
// The field count is a usable checksum: a description list that does not yield
// exactly fieldcount entries was mis-parsed, and DecodeParameter says so rather
// than returning a plausible prefix.
//
// What the width operand measures: the declared width of the type named in the
// descriptor, in bytes, counting UTF-16 units. That is not always the width the
// ABAP programmer wrote. When a caller passes a value whose type the serializer
// synthesises — the descriptor then names a generated `%_T…` type — the
// generated type is sized to the value, so the width tracks the value. When the
// descriptor names a real DDIC type the width is that type's, and a short value
// does not shrink it: `\TYPE=CHAR30` carries 60 whether four characters travel
// or thirty. Both are the same rule read on different declarations.

// ABAP type codes seen in a field description.
const (
	TypeInt1    = 0x01
	TypeInt2    = 0x02
	TypeInt4    = 0x03
	TypeChar    = 0x06 // width-parameterised
	TypeDats    = 0x0c
	TypeTims    = 0x0e
	TypeFloat   = 0x13
	TypeRaw     = 0x17 // width-parameterised
	TypeString  = 0x18
	TypeXString = 0x19
)

// widthParameterised lists the type codes that carry a two-byte width operand.
// The set is deliberately explicit rather than inferred: the captures show it is
// not simply "the variable-length ones", and guessing here mis-aligns the whole
// description list.
var widthParameterised = map[byte]bool{
	TypeChar: true,
	TypeRaw:  true,
}

// parameterHeader opens a parameter block: 'D' then the field count.
const parameterHeader = 0x44

// Field is one entry of a parameter's field-description list.
type Field struct {
	// TypeCode is the ABAP type code.
	TypeCode byte
	// Width is the declared width in bytes, UTF-16 counted, for the codes that
	// carry one. HasWidth says whether it was present.
	Width    int
	HasWidth bool
	// Name is the field name.
	Name string
}

// Parameter is a decoded parameter announcement: the type it names and the
// fields it describes.
type Parameter struct {
	// TypeName is what the `\TYPE=` descriptor named. A name beginning `%_T` is
	// a type the serializer generated for this call rather than one the ABAP
	// side declared — see the note above before reading anything into its width.
	TypeName string
	Fields   []Field
	// Offset is where the 0x44 header sat.
	Offset int
}

// Generated reports whether the parameter's type was synthesised by the
// serializer rather than declared in ABAP.
func (p Parameter) Generated() bool { return bytes.HasPrefix([]byte(p.TypeName), []byte("%_T")) }

// DecodeParameter reads the parameter announcement starting at off, or reports
// ok=false when off does not begin one or the list does not hold together.
func DecodeParameter(payload []byte, off int) (p Parameter, next int, ok bool) {
	if off+1 >= len(payload) || payload[off] != parameterHeader {
		return Parameter{}, off, false
	}
	count := int(payload[off+1])
	i := off + 2

	desc, after, ok := decodeRecordAt(payload, i)
	if !ok || !desc.IsTypeDescriptor() {
		return Parameter{}, off, false
	}
	name, _ := desc.TypeName()
	i = after

	fields := make([]Field, 0, count)
	for n := 0; n < count; n++ {
		if i >= len(payload) {
			return Parameter{}, off, false
		}
		f := Field{TypeCode: payload[i]}
		i++
		if widthParameterised[f.TypeCode] {
			if i+1 >= len(payload) {
				return Parameter{}, off, false
			}
			f.Width = int(payload[i]) | int(payload[i+1])<<8
			f.HasWidth = true
			i += 2
		}
		if i >= len(payload) {
			return Parameter{}, off, false
		}
		nameLen := int(payload[i])
		i++
		if nameLen == 0 || i+nameLen > len(payload) {
			return Parameter{}, off, false
		}
		raw := payload[i : i+nameLen]
		if !isPlainName(raw) {
			return Parameter{}, off, false
		}
		f.Name = string(raw)
		i += nameLen
		fields = append(fields, f)
	}

	// The declared count is the checksum. Reaching here with the wrong number of
	// fields means the walk drifted, and a plausible prefix is worse than none.
	if len(fields) != count {
		return Parameter{}, off, false
	}
	return Parameter{TypeName: name, Fields: fields, Offset: off}, i, true
}

// DecodeParameters finds every parameter announcement in a payload. It anchors
// on the 0x44 header followed by a well-formed descriptor, which together are
// specific enough that a chance byte run does not pass — and the field count
// then has to come out exactly.
func DecodeParameters(payload []byte) []Parameter {
	var out []Parameter
	for i := 0; i+2 < len(payload); {
		p, next, ok := DecodeParameter(payload, i)
		if !ok {
			i++
			continue
		}
		out = append(out, p)
		i = next
	}
	return out
}
