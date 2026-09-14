// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// The name of an xRFC parameter is not on the wire.
//
// EncodeCutFunctionRequest checks the name, and then writes the boundary with
// no value at all — twice, opening and closing. The only place the name
// survives is the root element of the XML between them. A server that reads
// the boundary gets an empty string and cannot tell REQUEST from anything
// else, which matters the moment a function has more than one.
func TestDecodeRequestRecoversXrfcParameterName(t *testing.T) {
	graph := ADTRestGraph()
	descriptor := classicrfc.FunintParameter{
		ParameterClass: "I", ParameterName: "REQUEST",
		TableName: adtRestRequest, Exid: "v",
	}
	body := map[string]any{
		"REQUEST_LINE":  map[string]any{"METHOD": "GET", "URI": "/sap/bc/adt/discovery", "VERSION": "HTTP/1.1"},
		"HEADER_FIELDS": []any{},
		"MESSAGE_BODY":  []byte{},
	}
	xml, err := xrfc.EncodeRecursiveParameter(descriptor, graph, body, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	packet, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName:   adtRestFunction,
		XrfcParameters: []cpic.NamedValue{{Name: "REQUEST", Value: xml}},
	})
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}

	got, err := DecodeCutFunctionRequest(packet)
	if err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if got.FunctionName != adtRestFunction {
		t.Fatalf("function name is %q", got.FunctionName)
	}
	if len(got.XrfcParameters) != 1 {
		t.Fatalf("expected one xRFC parameter, got %d", len(got.XrfcParameters))
	}
	if got.XrfcParameters[0].Name != "REQUEST" {
		t.Fatalf("the parameter name was lost: got %q, want %q — it is only in the XML",
			got.XrfcParameters[0].Name, "REQUEST")
	}
	if string(got.XrfcParameters[0].Value) != string(xml) {
		t.Fatalf("the XML did not survive:\n got  %s\n want %s", got.XrfcParameters[0].Value, xml)
	}
}
