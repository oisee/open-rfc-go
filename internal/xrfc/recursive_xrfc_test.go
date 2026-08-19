// SPDX-License-Identifier: Apache-2.0
//
// Oracle table tests for the recursive xRFC codec, reproduced from
// open-rfc test/recursive-xrfc.test.ts. INT8 fixtures are int64 and
// packed/decimal-float fixtures decimal strings; wire bytes match upstream.

package xrfc

import (
	"reflect"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/metadata"
)

func sfield(name, exid string, pos, uclen, dec int) metadata.MetadataField {
	return metadata.MetadataField{
		Name: name, Position: pos, InternalType: exid, UcLength: uclen, Decimals: dec,
		Reference: metadata.Reference{Kind: "scalar", InternalType: exid},
	}
}

func rfield(name, containerExid, kind, target string, pos int, cyclic bool) metadata.MetadataField {
	return metadata.MetadataField{
		Name: name, Position: pos, InternalType: containerExid,
		Reference: metadata.Reference{Kind: kind, TargetType: target, Cyclic: cyclic},
	}
}

func snode(name string, fields ...metadata.MetadataField) metadata.TypeNode {
	return metadata.TypeNode{Name: name, Kind: "structure", Fields: fields}
}

func tnode(name string, field metadata.MetadataField) metadata.TypeNode {
	return metadata.TypeNode{Name: name, Kind: "table", Fields: []metadata.MetadataField{field}}
}

func param(fn, name, class, exid, assoc, refKind, target string) metadata.Parameter {
	return metadata.Parameter{
		FunctionName: fn, Name: name, ParameterClass: class, InternalType: exid, AssociatedType: assoc,
		Reference: metadata.ParameterReference{Kind: refKind, TargetType: target},
	}
}

func graphOf(fn string, params []metadata.Parameter, nodes ...metadata.TypeNode) metadata.Graph {
	m := map[string]metadata.TypeNode{}
	for _, n := range nodes {
		m[n.Name] = n
	}
	return metadata.Graph{
		Version:          1,
		FunctionIdentity: &metadata.FunctionIdentity{Name: fn},
		Nodes:            m,
		Parameters:       params,
		Limits:           metadata.Limits{MaxNodes: 4096, MaxRows: 20000, MaxEdges: 20000, MaxDepth: 64},
	}
}

func rootGraph() metadata.Graph {
	return graphOf("Z_FN",
		[]metadata.Parameter{param("Z_FN", "ROOT", "I", "u", "Z_ROOT", "structure", "Z_ROOT")},
		snode("Z_ROW", sfield("VALUE", "N", 1, 8, 0), sfield("PAYLOAD", "y", 2, 8, 0)),
		tnode("Z_ROWS", rfield("", "u", "structure", "Z_ROW", 1, false)),
		snode("Z_CHILD", sfield("COUNT", "I", 1, 4, 0)),
		snode("Z_ROOT",
			sfield("TEXT", "C", 1, 12, 0),
			rfield("CHILD", "u", "structure", "Z_CHILD", 2, false),
			rfield("ROWS", "h", "table", "Z_ROWS", 3, false),
			sfield("BLOB", "y", 4, 8, 0),
		),
	)
}

var rootFunint = classicrfc.FunintParameter{ParameterClass: "I", ParameterName: "ROOT", TableName: "Z_ROOT", Exid: "u"}

func TestRecursiveNestedRoundTrip(t *testing.T) {
	g := rootGraph()
	in := map[string]any{
		"TEXT":  "A<&",
		"CHILD": map[string]any{"COUNT": int32(42)},
		"ROWS": []any{
			map[string]any{"VALUE": "0001", "PAYLOAD": hexb(t, "00a5ff")},
			map[string]any{"VALUE": "0002", "PAYLOAD": []byte{}},
		},
		"BLOB": hexb(t, "deadbeef"),
	}
	got, err := EncodeRecursiveParameter(rootFunint, g, in, RecursiveLimits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := `<ROOT><TEXT>A&#60;&#38;</TEXT><CHILD><COUNT>42</COUNT></CHILD>` +
		`<ROWS><item><VALUE>0001</VALUE><PAYLOAD>AKX/</PAYLOAD></item>` +
		`<item><VALUE>0002</VALUE><PAYLOAD></PAYLOAD></item></ROWS><BLOB>3q2+7w==</BLOB></ROOT>`
	if string(got) != want {
		t.Fatalf("encode mismatch:\n got %s\nwant %s", got, want)
	}
	decoded, err := DecodeRecursiveParameter(rootFunint, g, got, RecursiveLimits{})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(decoded, any(in)) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", decoded, in)
	}
}

func TestRecursiveOmittedFields(t *testing.T) {
	g := rootGraph()
	got, err := EncodeRecursiveParameter(rootFunint, g, map[string]any{}, RecursiveLimits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := `<ROOT><TEXT></TEXT><CHILD><COUNT>0</COUNT></CHILD><ROWS></ROWS><BLOB></BLOB></ROOT>`
	if string(got) != want {
		t.Fatalf("encode mismatch:\n got %s\nwant %s", got, want)
	}
	decoded, err := DecodeRecursiveParameter(rootFunint, g, got, RecursiveLimits{})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	want2 := map[string]any{"TEXT": "", "CHILD": map[string]any{"COUNT": int32(0)}, "ROWS": []any{}, "BLOB": []byte{}}
	if !reflect.DeepEqual(decoded, any(want2)) {
		t.Fatalf("decode mismatch:\n got %#v\nwant %#v", decoded, want2)
	}
}

func TestRecursiveAnonScalarTableAndEscaping(t *testing.T) {
	if esc, _ := EscapeTag("/NS/TEXTS"); esc != "_-NS_-TEXTS" {
		t.Fatalf("EscapeTag = %q, want _-NS_-TEXTS", esc)
	}
	g := graphOf("Z_FN",
		[]metadata.Parameter{param("Z_FN", "/NS/TEXTS", "I", "h", "Z_TEXTS", "table", "Z_TEXTS")},
		tnode("Z_TEXTS", sfield("", "g", 1, 8, 0)),
	)
	fp := classicrfc.FunintParameter{ParameterClass: "I", ParameterName: "/NS/TEXTS", TableName: "Z_TEXTS", Exid: "h"}
	got, err := EncodeRecursiveParameter(fp, g, []any{"ONE", "TWO"}, RecursiveLimits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := `<_-NS_-TEXTS><item>ONE</item><item>TWO</item></_-NS_-TEXTS>`
	if string(got) != want {
		t.Fatalf("encode mismatch:\n got %s\nwant %s", got, want)
	}
	name, err := DecodeRecursiveParameterName(got, RecursiveLimits{})
	if err != nil || name != "/NS/TEXTS" {
		t.Fatalf("DecodeRecursiveParameterName = %q, %v", name, err)
	}
	decoded, err := DecodeRecursiveParameter(fp, g, got, RecursiveLimits{})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(decoded, any([]any{"ONE", "TWO"})) {
		t.Fatalf("decode mismatch: %#v", decoded)
	}
}

func TestRecursiveBlankTemporal(t *testing.T) {
	g := graphOf("Z_FN",
		[]metadata.Parameter{param("Z_FN", "TEMPORAL", "I", "v", "Z_TEMPORAL", "structure", "Z_TEMPORAL")},
		snode("Z_TEMPORAL", sfield("DATE", "D", 1, 16, 0), sfield("TIME", "T", 2, 12, 0)),
	)
	fp := classicrfc.FunintParameter{ParameterClass: "I", ParameterName: "TEMPORAL", TableName: "Z_TEMPORAL", Exid: "v"}
	for _, in := range []map[string]any{
		{"DATE": "", "TIME": ""},
		{"DATE": "        ", "TIME": "      "},
	} {
		got, err := EncodeRecursiveParameter(fp, g, in, RecursiveLimits{})
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		want := `<TEMPORAL><DATE></DATE><TIME></TIME></TEMPORAL>`
		if string(got) != want {
			t.Fatalf("encode mismatch: %s", got)
		}
		decoded, err := DecodeRecursiveParameter(fp, g, got, RecursiveLimits{})
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		want2 := map[string]any{"DATE": "", "TIME": ""}
		if !reflect.DeepEqual(decoded, any(want2)) {
			t.Fatalf("decode mismatch: %#v", decoded)
		}
	}
}

func TestRecursiveCyclicRejected(t *testing.T) {
	g := graphOf("Z_FN",
		[]metadata.Parameter{param("Z_FN", "CYC", "I", "u", "Z_CYCLE", "structure", "Z_CYCLE")},
		snode("Z_CYCLE", rfield("SELF", "u", "structure", "Z_CYCLE", 1, true)),
	)
	fp := classicrfc.FunintParameter{ParameterClass: "I", ParameterName: "CYC", TableName: "Z_CYCLE", Exid: "u"}
	if _, _, err := ResolveParameter(g, fp); err == nil {
		t.Fatalf("expected cyclic error")
	}
}

func TestRecursiveDepthLimit(t *testing.T) {
	g := rootGraph()
	in := map[string]any{"CHILD": map[string]any{"COUNT": int32(1)}}
	if _, err := EncodeRecursiveParameter(rootFunint, g, in, RecursiveLimits{MaxDepth: ptr(1)}); err == nil {
		t.Fatalf("expected depth error")
	}
}

func TestRecursiveFlatFixedNotRequired(t *testing.T) {
	g := graphOf("Z_FN",
		[]metadata.Parameter{param("Z_FN", "FLAT", "I", "u", "Z_FLAT", "structure", "Z_FLAT")},
		snode("Z_FLAT", sfield("COUNT", "I", 1, 4, 0)),
	)
	fp := classicrfc.FunintParameter{ParameterClass: "I", ParameterName: "FLAT", TableName: "Z_FLAT", Exid: "u"}
	_, required, err := ResolveParameter(g, fp)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if required {
		t.Fatalf("flat fixed structure must not require xRFC")
	}
}

func FuzzDecodeRecursiveParameter(f *testing.F) {
	g := rootGraph()
	f.Add([]byte(`<ROOT><TEXT></TEXT><CHILD><COUNT>0</COUNT></CHILD><ROWS></ROWS><BLOB></BLOB></ROOT>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeRecursiveParameter(rootFunint, g, data, RecursiveLimits{})
		_, _ = DecodeRecursiveParameterName(data, RecursiveLimits{})
	})
}
