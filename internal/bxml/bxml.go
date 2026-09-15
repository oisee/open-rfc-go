// SPDX-License-Identifier: Apache-2.0
//
// SAP Binary XML 0.7, the form ADT rides in over RFC.
//
// SADT_REST_RFC_ENDPOINT carries one HTTP exchange as a recursive parameter,
// and on the wire that parameter is not the text xRFC XML the rest of this
// package speaks — it is SAP's Binary XML: a token stream with a name table,
// UTF-8 payloads, and lengths written as code points. This file reads it and
// writes it, and nothing else in the tree knows it exists.
//
// The grammar was measured against a live Eclipse↔A4H capture, not guessed, and
// the measurement is the test: a correct reader consumes a document to its last
// byte, and a correct writer reproduces the bytes a reader accepts. The tokens:
//
//	3f  header pair      len key, len value
//	2b  declare name     len name; appended to the name table
//	3c  open element     two operand bytes: name ref, namespace ref
//	3e  close element
//	40  attribute        two operand bytes: name ref, namespace ref
//	41  attribute value  len value
//	3a  bind namespace   two operand bytes
//	2a  xmlns marker      one operand byte
//	54  text             len text (element character content)
//	42  message body     len text (opaque content, not parsed as markup)
//	43  string           len text (a second string form, treated as text)
//
// A name reference is the name's table index plus two: indices zero and one are
// reserved, so the first declared name is referenced as 2. Lengths are the
// Unicode scalar value of the byte count, encoded as UTF-8 (one byte below
// 0x80, up to four for a multi-megabyte body).
//
// Original work for open-rfc-go. Nothing here is ported from a SAP source.
package bxml

import (
	"bytes"
	"errors"
	"fmt"
)

// ErrBXML reports a malformed SAP Binary XML document.
var ErrBXML = errors.New("bxml: malformed document")

// magic opens every document.
var magic = []byte("BXML")

// Tokens.
const (
	tokHeader   = 0x3f
	tokName     = 0x2b
	tokOpen     = 0x3c
	tokClose    = 0x3e
	tokAttr     = 0x40
	tokAttrVal  = 0x41
	tokBind     = 0x3a
	tokXmlns    = 0x2a
	tokText     = 0x54
	tokBody     = 0x42
	tokString   = 0x43
	nameRefBias = 2
)

// Element is one node of a decoded document: a name, optional attributes,
// children, and — for a leaf — either text or an opaque message body.
type Element struct {
	Name     string
	Attrs    []Attr
	Children []*Element
	Text     string
	HasText  bool
	Body     string
	HasBody  bool
}

// Attr is one attribute of an element.
type Attr struct {
	Name  string
	Value string
}

// Child returns the first child named name, or nil.
func (e *Element) Child(name string) *Element {
	if e == nil {
		return nil
	}
	for _, c := range e.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// ChildText returns the text of the first child named name, or "".
func (e *Element) ChildText(name string) string {
	if c := e.Child(name); c != nil {
		return c.Text
	}
	return ""
}

// readLength reads a byte count written as a UTF-8-encoded scalar.
func readLength(doc []byte, i int) (int, int, error) {
	if i >= len(doc) {
		return 0, 0, fmt.Errorf("%w: length past end of document", ErrBXML)
	}
	c := doc[i]
	switch {
	case c < 0x80:
		return int(c), i + 1, nil
	case c < 0xe0:
		if i+1 >= len(doc) {
			return 0, 0, fmt.Errorf("%w: truncated 2-byte length", ErrBXML)
		}
		return (int(c&0x1f) << 6) | int(doc[i+1]&0x3f), i + 2, nil
	case c < 0xf0:
		if i+2 >= len(doc) {
			return 0, 0, fmt.Errorf("%w: truncated 3-byte length", ErrBXML)
		}
		return (int(c&0x0f) << 12) | (int(doc[i+1]&0x3f) << 6) | int(doc[i+2]&0x3f), i + 3, nil
	default:
		if i+3 >= len(doc) {
			return 0, 0, fmt.Errorf("%w: truncated 4-byte length", ErrBXML)
		}
		return (int(c&0x07) << 18) | (int(doc[i+1]&0x3f) << 12) | (int(doc[i+2]&0x3f) << 6) | int(doc[i+3]&0x3f), i + 4, nil
	}
}

// writeLength appends a byte count as a UTF-8-encoded scalar.
func writeLength(out []byte, n int) []byte {
	switch {
	case n < 0x80:
		return append(out, byte(n))
	case n < 0x800:
		return append(out, byte(0xc0|(n>>6)), byte(0x80|(n&0x3f)))
	case n < 0x10000:
		return append(out, byte(0xe0|(n>>12)), byte(0x80|((n>>6)&0x3f)), byte(0x80|(n&0x3f)))
	default:
		return append(out, byte(0xf0|(n>>18)), byte(0x80|((n>>12)&0x3f)), byte(0x80|((n>>6)&0x3f)), byte(0x80|(n&0x3f)))
	}
}

// readString reads a length then that many bytes.
func readString(doc []byte, i int) (string, int, error) {
	n, j, err := readLength(doc, i)
	if err != nil {
		return "", 0, err
	}
	if j+n > len(doc) {
		return "", 0, fmt.Errorf("%w: string of %d bytes runs past end", ErrBXML, n)
	}
	return string(doc[j : j+n]), j + n, nil
}

// Decode parses a document into its root asx:abap element, or reports the first
// byte at which the grammar breaks.
//
// The two outer elements (asx:abap, asx:values) are returned as the tree's
// spine; the payload root — REQUEST or RESPONSE — is the single grandchild.
func Decode(doc []byte) (*Element, error) {
	if !bytes.HasPrefix(doc, magic) {
		return nil, fmt.Errorf("%w: no BXML magic", ErrBXML)
	}
	i := len(magic)
	var names []string
	root := &Element{Name: "#document"}
	stack := []*Element{root}
	top := func() *Element { return stack[len(stack)-1] }
	nameByRef := func(ref int) (string, error) {
		idx := ref - nameRefBias
		if idx < 0 || idx >= len(names) {
			return "", fmt.Errorf("%w: name reference %d out of range (%d declared)", ErrBXML, ref, len(names))
		}
		return names[idx], nil
	}
	for i < len(doc) {
		tok := doc[i]
		switch tok {
		case tokHeader:
			_, j, err := readString(doc, i+1)
			if err != nil {
				return nil, err
			}
			_, j, err = readString(doc, j)
			if err != nil {
				return nil, err
			}
			i = j
		case tokName:
			s, j, err := readString(doc, i+1)
			if err != nil {
				return nil, err
			}
			names = append(names, s)
			i = j
		case tokOpen:
			if i+2 >= len(doc) {
				return nil, fmt.Errorf("%w: truncated open at %d", ErrBXML, i)
			}
			name, err := nameByRef(int(doc[i+1]))
			if err != nil {
				return nil, err
			}
			el := &Element{Name: name}
			top().Children = append(top().Children, el)
			stack = append(stack, el)
			i += 3
		case tokClose:
			if len(stack) <= 1 {
				return nil, fmt.Errorf("%w: close without a matching open at %d", ErrBXML, i)
			}
			stack = stack[:len(stack)-1]
			i++
		case tokAttr:
			if i+2 >= len(doc) {
				return nil, fmt.Errorf("%w: truncated attribute at %d", ErrBXML, i)
			}
			name, err := nameByRef(int(doc[i+1]))
			if err != nil {
				return nil, err
			}
			// value follows as its own token
			if i+3 >= len(doc) || doc[i+3] != tokAttrVal {
				return nil, fmt.Errorf("%w: attribute %q without a value at %d", ErrBXML, name, i)
			}
			v, j, err := readString(doc, i+4)
			if err != nil {
				return nil, err
			}
			top().Attrs = append(top().Attrs, Attr{Name: name, Value: v})
			i = j
		case tokAttrVal:
			// a bare value with no preceding attribute name: tolerate by
			// attaching to the last attribute if present, else skip its bytes
			_, j, err := readString(doc, i+1)
			if err != nil {
				return nil, err
			}
			i = j
		case tokBind:
			if i+2 >= len(doc) {
				return nil, fmt.Errorf("%w: truncated namespace bind at %d", ErrBXML, i)
			}
			i += 3
		case tokXmlns:
			if i+1 >= len(doc) {
				return nil, fmt.Errorf("%w: truncated xmlns marker at %d", ErrBXML, i)
			}
			i += 2
		case tokText:
			s, j, err := readString(doc, i+1)
			if err != nil {
				return nil, err
			}
			top().Text = s
			top().HasText = true
			i = j
		case tokBody:
			s, j, err := readString(doc, i+1)
			if err != nil {
				return nil, err
			}
			top().Body = s
			top().HasBody = true
			i = j
		case tokString:
			s, j, err := readString(doc, i+1)
			if err != nil {
				return nil, err
			}
			top().Text = s
			top().HasText = true
			i = j
		default:
			return nil, fmt.Errorf("%w: unknown token 0x%02x at %d", ErrBXML, tok, i)
		}
	}
	if len(stack) != 1 {
		return nil, fmt.Errorf("%w: %d elements left open", ErrBXML, len(stack)-1)
	}
	return root, nil
}

// PayloadRoot returns the REQUEST/RESPONSE element: the grandchild under
// asx:abap → asx:values.
func PayloadRoot(document *Element) (*Element, error) {
	abap := document.Child("abap")
	if abap == nil && len(document.Children) > 0 {
		abap = document.Children[0]
	}
	values := abap.Child("values")
	if values == nil && abap != nil && len(abap.Children) > 0 {
		values = abap.Children[0]
	}
	if values == nil || len(values.Children) == 0 {
		return nil, fmt.Errorf("%w: no payload element under asx:values", ErrBXML)
	}
	return values.Children[0], nil
}
