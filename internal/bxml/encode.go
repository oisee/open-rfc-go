// SPDX-License-Identifier: Apache-2.0

package bxml

// The writer.
//
// Every document opens with the same asXML envelope — the BXML magic, the
// VER/ENC headers, the asx namespace bound and declared, <asx:abap
// version="1.0"> and <asx:values> — before the payload root. The envelope is a
// constant of the format, so it is written by the same primitives as the
// content and checked against the captured bytes in the test rather than
// pasted in as hex nobody can read.

// encoder accumulates a document and its name table.
type encoder struct {
	out   []byte
	names map[string]int
	count int
}

func (e *encoder) ref(name string) int {
	if idx, ok := e.names[name]; ok {
		return idx + nameRefBias
	}
	idx := e.count
	e.count++
	e.names[name] = idx
	e.out = append(e.out, tokName)
	e.out = writeLength(e.out, len(name))
	e.out = append(e.out, name...)
	return idx + nameRefBias
}

func (e *encoder) header(key, value string) {
	e.out = append(e.out, tokHeader)
	e.out = writeLength(e.out, len(key))
	e.out = append(e.out, key...)
	e.out = writeLength(e.out, len(value))
	e.out = append(e.out, value...)
}

func (e *encoder) str(tok byte, s string) {
	e.out = append(e.out, tok)
	e.out = writeLength(e.out, len(s))
	e.out = append(e.out, s...)
}

// Namespace references, as the envelope uses them.
const (
	nsNone = 1 // no prefix
	nsAsx  = 2 // the asx prefix (name table index 0 + bias)
	// nsAsxURL is what the table attribute `lines` is written under: the
	// reference of the abapxml URL itself rather than the prefix. Copied from
	// the capture, where every table carries it; what it means to the reader
	// is not known and does not need to be, only that this is what is sent.
	nsAsxURL = 3
)

// envelope writes everything before the payload root and returns the encoder
// positioned to continue the name table.
func envelope() *encoder {
	e := &encoder{names: map[string]int{}}
	e.out = append(e.out, magic...)
	e.header("VER", "0.7")
	e.header("ENC", "utf-8")
	asx := e.ref("asx")
	asxURL := e.ref("http://www.sap.com/abapxml")
	e.out = append(e.out, tokBind, byte(asx), byte(asxURL))
	abap := e.ref("abap")
	e.out = append(e.out, tokOpen, byte(abap), nsAsx)
	version := e.ref("version")
	e.out = append(e.out, tokAttr, byte(version), nsNone)
	e.str(tokAttrVal, "1.0")
	hint := e.ref("asxhint")
	hintURL := e.ref("http://www.sap.com/abapxml/hint")
	e.out = append(e.out, tokBind, byte(hint), byte(hintURL))
	e.out = append(e.out, tokXmlns, byte(asxURL))
	values := e.ref("values")
	e.out = append(e.out, tokOpen, byte(values), nsAsx)
	return e
}

func (e *encoder) element(el *Element) {
	ref := e.ref(el.Name)
	e.out = append(e.out, tokOpen, byte(ref), nsNone)
	for _, a := range el.Attrs {
		aref := e.ref(a.Name)
		ns := byte(nsNone)
		if a.Name == "lines" {
			ns = nsAsxURL
		}
		e.out = append(e.out, tokAttr, byte(aref), ns)
		e.str(tokAttrVal, a.Value)
	}
	switch {
	case el.HasBody:
		e.str(tokBody, el.Body)
	case el.HasText:
		e.str(tokText, el.Text)
	}
	for _, c := range el.Children {
		e.element(c)
	}
	e.out = append(e.out, tokClose)
}

// Encode writes a document whose payload root is root: the envelope, the root
// and its subtree, then the two closes for asx:values and asx:abap.
//
// Name references are one byte, so a document may declare at most 253 distinct
// names; ADT's request and response shapes use eighteen.
func Encode(root *Element) ([]byte, error) {
	e := envelope()
	e.element(root)
	e.out = append(e.out, tokClose, tokClose)
	if e.count+nameRefBias > 0xff {
		return nil, ErrBXML
	}
	return e.out, nil
}
