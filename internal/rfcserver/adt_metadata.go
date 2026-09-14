// SPDX-License-Identifier: Apache-2.0

package rfcserver

import "github.com/oisee/open-rfc-go/internal/metadata"

// The type graph of SADT_REST_RFC_ENDPOINT, which is how ADT travels over RFC.
//
// A server has to know the shape of what it is being sent before it can decode
// it, and a server has nobody to ask: the client is the one calling. So the
// graph is stated here rather than fetched.
//
// The shape was measured with RFC_METADATA_GET(DEEP) against an AS ABAP 1909
// system rather than guessed, and it is small enough to read: a request is a
// line, a table of headers and a body, which is what an HTTP request is. That
// is also why stating it here is describing an interface rather than copying a
// dictionary — the only surprising part is one asymmetry, and it is noted where
// it happens.
//
// Lengths are not carried because nothing here is fixed-width: every field is a
// string, an xstring or a reference, and over xRFC those are named elements
// rather than byte offsets.
const (
	adtRestFunction = "SADT_REST_RFC_ENDPOINT"

	adtRestRequest     = "SADT_REST_REQUEST"
	adtRestRequestLine = "SADT_REST_REQUEST_LINE"
	adtRestResponse    = "SADT_REST_RESPONSE"
	adtRestStatusLine  = "SADT_REST_STATUS_LINE"
	adtHeaderTable     = "TIHTTPNVP"
	adtHeaderRow       = "IHTTPNVP"
)

func adtScalar(name string, position int, internalType string) metadata.MetadataField {
	// Kind must be stated even for a scalar: the codec dispatches on it, and a
	// field that leaves it empty is taken for a reference to a type with no
	// name, which fails a long way from the cause.
	return metadata.MetadataField{
		Name: name, Position: position, InternalType: internalType,
		// the type is stated twice on purpose: the validator requires the
		// reference to agree with the field, and when it does not the error
		// says "not implemented", which is not what went wrong
		Reference: metadata.Reference{Kind: "scalar", InternalType: internalType},
	}
}

func adtReference(name string, position int, internalType, kind, target string) metadata.MetadataField {
	return metadata.MetadataField{
		Name: name, Position: position, InternalType: internalType,
		AssociatedType: target,
		Reference:      metadata.Reference{Kind: kind, TargetType: target},
	}
}

// ADTRestGraph is the metadata graph for SADT_REST_RFC_ENDPOINT.
func ADTRestGraph() metadata.Graph {
	nodes := []metadata.TypeNode{
		{Name: adtHeaderRow, Kind: "structure", Fields: []metadata.MetadataField{
			adtScalar("NAME", 1, "g"),
			adtScalar("VALUE", 2, "g"),
		}},
		{Name: adtHeaderTable, Kind: "table", Fields: []metadata.MetadataField{
			adtReference("", 1, "u", "structure", adtHeaderRow),
		}},
		{Name: adtRestRequestLine, Kind: "structure", Fields: []metadata.MetadataField{
			adtScalar("METHOD", 1, "g"),
			adtScalar("URI", 2, "g"),
			adtScalar("VERSION", 3, "g"),
		}},
		{Name: adtRestStatusLine, Kind: "structure", Fields: []metadata.MetadataField{
			adtScalar("VERSION", 1, "g"),
			// STATUS_CODE is an SSTR where everything around it is a STRG.
			// It arrives as the same internal type and the asymmetry shows
			// only in the dictionary, which is exactly the kind of detail
			// that costs an afternoon if it is met by surprise.
			adtScalar("STATUS_CODE", 2, "g"),
			adtScalar("REASON_PHRASE", 3, "g"),
		}},
		{Name: adtRestRequest, Kind: "structure", Fields: []metadata.MetadataField{
			adtReference("REQUEST_LINE", 1, "v", "structure", adtRestRequestLine),
			adtReference("HEADER_FIELDS", 2, "h", "table", adtHeaderTable),
			adtScalar("MESSAGE_BODY", 3, "y"),
		}},
		{Name: adtRestResponse, Kind: "structure", Fields: []metadata.MetadataField{
			adtReference("STATUS_LINE", 1, "v", "structure", adtRestStatusLine),
			adtReference("HEADER_FIELDS", 2, "h", "table", adtHeaderTable),
			adtScalar("MESSAGE_BODY", 3, "y"),
		}},
	}
	byName := make(map[string]metadata.TypeNode, len(nodes))
	for _, n := range nodes {
		byName[n.Name] = n
	}
	return metadata.Graph{
		Version:          1,
		FunctionIdentity: &metadata.FunctionIdentity{Name: adtRestFunction, RemoteBasxmlSupported: true},
		Nodes:            byName,
		Parameters: []metadata.Parameter{
			{
				FunctionName: adtRestFunction, Name: "REQUEST", ParameterClass: "I", Position: 1,
				AssociatedType: adtRestRequest, InternalType: "v",
				Reference: metadata.ParameterReference{Kind: "structure", TargetType: adtRestRequest},
			},
			{
				FunctionName: adtRestFunction, Name: "RESPONSE", ParameterClass: "E", Position: 2,
				AssociatedType: adtRestResponse, InternalType: "v",
				Reference: metadata.ParameterReference{Kind: "structure", TargetType: adtRestResponse},
			},
		},
		Limits: metadata.Limits{MaxNodes: 4096, MaxRows: 20000, MaxEdges: 20000, MaxDepth: 64},
	}
}
