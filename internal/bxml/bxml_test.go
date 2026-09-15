// SPDX-License-Identifier: Apache-2.0

package bxml

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

// The envelope, byte for byte, as Eclipse and the system both write it.
//
// These are the first 144 bytes of every captured document in both directions:
// magic, two headers, the asx namespace, <asx:abap version="1.0">, the hint
// namespace, and <asx:values>. Nothing in them names a system, a user or a
// session; it is the asXML envelope and only that. If the writer drifts from
// it, this is where the drift is named.
const capturedEnvelopeHex = "42584d4c" +
	"3f0356455203302e37" + // ?VER=0.7
	"3f03454e43057574662d38" + // ?ENC=utf-8
	"2b03617378" + // +asx
	"2b1a687474703a2f2f7777772e7361702e636f6d2f61626170786d6c" + // +http://www.sap.com/abapxml
	"3a0203" + // :bind asx→url
	"2b0461626170" + // +abap
	"3c0402" + // <asx:abap
	"2b0776657273696f6e" + // +version
	"400501" + // @version
	"4103312e30" + // A"1.0"
	"2b0761737868696e74" + // +asxhint
	"2b1f687474703a2f2f7777772e7361702e636f6d2f61626170786d6c2f68696e74" + // +hint url
	"3a0607" + // :bind asxhint→url
	"2a03" + // *xmlns
	"2b0676616c756573" + // +values
	"3c0802" // <asx:values

func TestEnvelopeMatchesCapture(t *testing.T) {
	want, err := hex.DecodeString(capturedEnvelopeHex)
	if err != nil {
		t.Fatal(err)
	}
	got := envelope().out
	if !bytes.Equal(got, want) {
		t.Fatalf("envelope differs\n got %x\nwant %x", got, want)
	}
	if len(want) != 144 {
		t.Fatalf("envelope is %d bytes, the capture's is 144", len(want))
	}
}

// A request in the shape Eclipse sends: the discovery GET, with documentation
// values where the capture had a request id and a user agent.
func sampleRequest() *Element {
	return &Element{Name: "REQUEST", Children: []*Element{
		{Name: "REQUEST_LINE", Children: []*Element{
			{Name: "METHOD", Text: "GET", HasText: true},
			{Name: "URI", Text: "/sap/bc/adt/core/discovery", HasText: true},
			{Name: "VERSION", Text: "HTTP/1.1", HasText: true},
		}},
		{Name: "HEADER_FIELDS", Attrs: []Attr{{Name: "lines", Value: "2"}}, Children: []*Element{
			{Name: "item", Children: []*Element{
				{Name: "NAME", Text: "sap-adt-request-id", HasText: true},
				{Name: "VALUE", Text: "00000000000000000000000000000000", HasText: true},
			}},
			{Name: "item", Children: []*Element{
				{Name: "NAME", Text: "Accept", HasText: true},
				{Name: "VALUE", Text: "application/atomsvc+xml", HasText: true},
			}},
		}},
		{Name: "MESSAGE_BODY"},
	}}
}

func TestRequestRoundTrip(t *testing.T) {
	doc, err := Encode(sampleRequest())
	if err != nil {
		t.Fatal(err)
	}
	tree, err := Decode(doc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	root, err := PayloadRoot(tree)
	if err != nil {
		t.Fatal(err)
	}
	if root.Name != "REQUEST" {
		t.Fatalf("root is %q", root.Name)
	}
	line := root.Child("REQUEST_LINE")
	if got := line.ChildText("METHOD") + " " + line.ChildText("URI") + " " + line.ChildText("VERSION"); got != "GET /sap/bc/adt/core/discovery HTTP/1.1" {
		t.Fatalf("request line: %q", got)
	}
	headers := root.Child("HEADER_FIELDS")
	if len(headers.Attrs) != 1 || headers.Attrs[0] != (Attr{"lines", "2"}) {
		t.Fatalf("lines attribute: %+v", headers.Attrs)
	}
	if len(headers.Children) != 2 || headers.Children[1].ChildText("NAME") != "Accept" {
		t.Fatalf("headers: %+v", headers.Children)
	}
	body := root.Child("MESSAGE_BODY")
	if body == nil || body.HasText || body.HasBody {
		t.Fatalf("an empty body should decode as an empty element: %+v", body)
	}
	// and the same bytes again: a writer that is its own reader's inverse
	again, err := Encode(root)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, doc) {
		t.Fatalf("re-encoding changed the document\n got %x\nwant %x", again, doc)
	}
}

// The token stream of the request, checked against what Eclipse actually wrote
// for the same structure: each element declares its name once and references
// it by index+2, repeated rows reuse the reference, and the table attribute is
// written under namespace reference 3. These offsets are the capture's.
func TestRequestTokensMatchEclipse(t *testing.T) {
	doc, err := Encode(sampleRequest())
	if err != nil {
		t.Fatal(err)
	}
	// after the 144-byte envelope: +REQUEST <9,1 +REQUEST_LINE <10,1 +METHOD <11,1 T"GET" >
	want := "2b07524551554553543c0901" +
		"2b0c524551554553545f4c494e453c0a01" +
		"2b064d4554484f443c0b01" +
		"54034745543e"
	got := hex.EncodeToString(doc[144 : 144+len(want)/2])
	if got != want {
		t.Fatalf("content tokens differ\n got %s\nwant %s", got, want)
	}
	// the second header row reuses <16,1 <17,1 … <18,1 without redeclaring
	if !strings.Contains(hex.EncodeToString(doc), "3c10013c110154064163636570743e") {
		t.Fatalf("second row does not reuse the item/NAME references")
	}
	// the table attribute: @15,3 A"2"
	if !strings.Contains(hex.EncodeToString(doc), "2b056c696e6573400f03410132") {
		t.Fatalf("lines attribute is not written as the capture writes it")
	}
}

// A response with a body that needs a four-byte length.
func TestLargeBodyLength(t *testing.T) {
	big := strings.Repeat("<x/>", 80_000) // 320,000 bytes: past 0xffff, four-byte length
	resp := &Element{Name: "RESPONSE", Children: []*Element{
		{Name: "STATUS_LINE", Children: []*Element{
			{Name: "VERSION", Text: "HTTP/1.1", HasText: true},
			{Name: "STATUS_CODE", Text: "200 ", HasText: true},
			{Name: "REASON_PHRASE", Text: "OK", HasText: true},
		}},
		{Name: "HEADER_FIELDS", Attrs: []Attr{{Name: "lines", Value: "1"}}, Children: []*Element{
			{Name: "IHTTPNVP", Children: []*Element{
				{Name: "NAME", Text: "Content-Type", HasText: true},
				{Name: "VALUE", Text: "application/xml", HasText: true},
			}},
		}},
		{Name: "MESSAGE_BODY", Body: big, HasBody: true},
	}}
	doc, err := Encode(resp)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := Decode(doc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	root, _ := PayloadRoot(tree)
	if got := root.Child("MESSAGE_BODY"); got == nil || !got.HasBody || got.Body != big {
		t.Fatalf("body did not survive a four-byte length")
	}
}

func TestLengthEncodings(t *testing.T) {
	for _, n := range []int{0, 1, 0x7f, 0x80, 151, 196, 0x7ff, 0x800, 0xffff, 0x10000, 299658, 0x1fffff} {
		enc := writeLength(nil, n)
		got, next, err := readLength(enc, 0)
		if err != nil || got != n || next != len(enc) {
			t.Fatalf("length %d: wrote %x, read %d (next %d, err %v)", n, enc, got, next, err)
		}
	}
	// the two-byte examples measured in the capture
	if got := writeLength(nil, 151); !bytes.Equal(got, []byte{0xc2, 0x97}) {
		t.Fatalf("151 -> %x, capture says c2 97", got)
	}
	if got := writeLength(nil, 196); !bytes.Equal(got, []byte{0xc3, 0x84}) {
		t.Fatalf("196 -> %x, capture says c3 84", got)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	for _, bad := range [][]byte{nil, []byte("BXM"), []byte("BXML\x3c"), []byte("BXML\x3e"), []byte("BXML\x99")} {
		if _, err := Decode(bad); err == nil {
			t.Fatalf("%q decoded", bad)
		}
	}
}

func FuzzDecode(f *testing.F) {
	doc, _ := Encode(sampleRequest())
	f.Add(doc)
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = Decode(b)
	})
}
