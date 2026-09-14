// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"reflect"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// An HTTP exchange through the graph and back.
//
// Before any of this is wired to a socket, the graph has to be right: a server
// that decodes a request into the wrong shape will answer confidently and
// wrongly, which is worse than refusing. So a whole request goes in — a
// request line, two headers and a body with bytes that are not text — and has
// to come back identical.
//
// The body is deliberately binary. An xstring that round-trips as text would
// pass a weaker test and fail on the first ADT response, which is XML with a
// BOM more often than not.
func TestADTRestRequestRoundTrip(t *testing.T) {
	graph := ADTRestGraph()
	descriptor := classicrfc.FunintParameter{
		ParameterClass: "I", ParameterName: "REQUEST",
		TableName: adtRestRequest, Exid: "v",
	}

	sent := map[string]any{
		"REQUEST_LINE": map[string]any{
			"METHOD":  "GET",
			"URI":     "/sap/bc/adt/discovery",
			"VERSION": "HTTP/1.1",
		},
		"HEADER_FIELDS": []any{
			map[string]any{"NAME": "Accept", "VALUE": "application/atomsvc+xml"},
			map[string]any{"NAME": "X-sap-adt-sessiontype", "VALUE": "stateful"},
		},
		"MESSAGE_BODY": []byte{0x00, 0xef, 0xbb, 0xbf, 0x3c},
	}

	encoded, err := xrfc.EncodeRecursiveParameter(descriptor, graph, sent, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := xrfc.DecodeRecursiveParameter(descriptor, graph, encoded, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("decode: %v\n  encoded as: %s", err, encoded)
	}
	if !reflect.DeepEqual(decoded, any(sent)) {
		t.Fatalf("the request did not survive the round trip:\n got  %#v\n want %#v\n xml  %s",
			decoded, sent, encoded)
	}
}

// The answer travels the same way, and its one oddity is checked here rather
// than discovered later: STATUS_CODE is a string on the wire, not a number.
func TestADTRestResponseRoundTrip(t *testing.T) {
	graph := ADTRestGraph()
	descriptor := classicrfc.FunintParameter{
		ParameterClass: "E", ParameterName: "RESPONSE",
		TableName: adtRestResponse, Exid: "v",
	}

	sent := map[string]any{
		"STATUS_LINE": map[string]any{
			"VERSION":       "HTTP/1.1",
			"STATUS_CODE":   "200",
			"REASON_PHRASE": "OK",
		},
		"HEADER_FIELDS": []any{
			map[string]any{"NAME": "Content-Type", "VALUE": "application/atomsvc+xml"},
		},
		"MESSAGE_BODY": []byte("<?xml version=\"1.0\"?><app:service/>"),
	}

	encoded, err := xrfc.EncodeRecursiveParameter(descriptor, graph, sent, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := xrfc.DecodeRecursiveParameter(descriptor, graph, encoded, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("decode: %v\n  encoded as: %s", err, encoded)
	}
	if !reflect.DeepEqual(decoded, any(sent)) {
		t.Fatalf("the response did not survive the round trip:\n got  %#v\n want %#v\n xml  %s",
			decoded, sent, encoded)
	}
}
