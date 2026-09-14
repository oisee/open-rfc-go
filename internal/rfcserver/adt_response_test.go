// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// A handler that answers in xRFC must be able to say so.
//
// Response carries exports and tables, which is everything a classic function
// needs and nothing a recursive one does. SADT_REST_RFC_ENDPOINT exports a
// structure holding a structure, a table and an xstring, and that travels as
// xRFC XML between two 0x3c02 boundaries in 0x3c05 chunks — the same shape the
// request arrives in. Without a way to emit it the server can decode Eclipse
// perfectly and has no way to answer.
func TestEncodeResponseCarriesXrfcParameter(t *testing.T) {
	graph := ADTRestGraph()
	descriptor := classicrfc.FunintParameter{
		ParameterClass: "E", ParameterName: "RESPONSE",
		TableName: adtRestResponse, Exid: "v",
	}
	answer := map[string]any{
		"STATUS_LINE": map[string]any{
			"VERSION": "HTTP/1.1", "STATUS_CODE": "200", "REASON_PHRASE": "OK",
		},
		"HEADER_FIELDS": []any{
			map[string]any{"NAME": "Content-Type", "VALUE": "application/atomsvc+xml"},
		},
		"MESSAGE_BODY": []byte("<?xml version=\"1.0\"?><app:service/>"),
	}
	xml, err := xrfc.EncodeRecursiveParameter(descriptor, graph, answer, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("encode parameter: %v", err)
	}

	encoded, err := EncodeCutFunctionResponse(nil, nil, []cpic.NamedValue{{Name: "RESPONSE", Value: xml}})
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}

	decoded, err := cpic.DecodeFunctionResultFields(encoded)
	if err != nil {
		t.Fatalf("a client could not read the response: %v", err)
	}
	if !decoded.Success {
		t.Fatal("the response did not classify as successful")
	}

	// reassemble the chunks the way a client does
	var got []byte
	boundaries := 0
	for _, f := range decoded.Fields {
		switch cpic.Tag(f.Tag) {
		case cpic.TagXRfcParameter:
			boundaries++
		case cpic.TagXRfcData:
			got = append(got, f.Value...)
		}
	}
	if boundaries != 2 {
		t.Fatalf("expected an opening and a closing boundary, got %d", boundaries)
	}
	if string(got) != string(xml) {
		t.Fatalf("the XML did not survive the response:\n got  %s\n want %s", got, xml)
	}

	name, err := xrfc.DecodeRecursiveParameterName(got, xrfc.RecursiveLimits{})
	if err != nil || name != "RESPONSE" {
		t.Fatalf("the parameter does not name itself: %q, %v", name, err)
	}
}
