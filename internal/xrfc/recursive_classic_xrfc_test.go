// SPDX-License-Identifier: Apache-2.0
//
// Oracle table tests for the recursive classic xRFC codec, reproduced from
// open-rfc test/recursive-classic-xrfc.test.ts. INT4 fixtures are int32; wire
// bytes match upstream.

package xrfc

import (
	"reflect"
	"testing"

	"github.com/oisee/open-rfc-go/internal/metadata"
)

func rcGraph() metadata.Graph {
	return graphOf("Z_RECURSIVE_XRFC",
		[]metadata.Parameter{
			param("Z_RECURSIVE_XRFC", "INPUT", "I", "v", "Z_ROOT", "structure", "Z_ROOT"),
		},
		snode("Z_ROOT",
			sfield("ID", "I", 1, 4, 0),
			rfield("CHILD", "u", "structure", "Z_CHILD", 2, false),
			rfield("ROWS", "h", "table", "Z_ROW_T", 3, false),
			sfield("BLOB", "y", 4, 8, 0),
		),
		snode("Z_CHILD", sfield("TEXT", "C", 1, 8, 0), sfield("LABEL", "g", 2, 8, 0)),
		tnode("Z_ROW_T", rfield("", "v", "structure", "Z_ROW", 1, false)),
		snode("Z_ROW",
			sfield("COUNT", "I", 1, 4, 0),
			rfield("DETAIL", "u", "structure", "Z_CHILD", 2, false),
			rfield("CHUNKS", "h", "table", "Z_CHUNK_T", 3, false),
		),
		tnode("Z_CHUNK_T", rfield("", "v", "structure", "Z_CHUNK", 1, false)),
		snode("Z_CHUNK", sfield("DATA", "y", 1, 8, 0)),
	)
}

var rcIdentity = RecursiveClassicIdentity{
	FunctionName: "Z_RECURSIVE_XRFC", ParameterName: "INPUT",
	ParameterClass: "I", AssociatedType: "Z_ROOT", InternalType: "v",
}

func rcValue(t *testing.T) map[string]any {
	return map[string]any{
		"ID":    int32(7),
		"CHILD": map[string]any{"TEXT": "A&B", "LABEL": "Grüße 🌍"},
		"ROWS": []any{
			map[string]any{
				"COUNT":  int32(1),
				"DETAIL": map[string]any{"TEXT": "ONE", "LABEL": "first"},
				"CHUNKS": []any{map[string]any{"DATA": hexb(t, "deadbeef")}},
			},
			map[string]any{
				"COUNT":  int32(-2),
				"DETAIL": map[string]any{"TEXT": "TWO", "LABEL": ""},
				"CHUNKS": []any{},
			},
		},
		"BLOB": hexb(t, "000102"),
	}
}

const rcExpectedXML = `<INPUT><ID>7</ID><CHILD><TEXT>A&#38;B</TEXT><LABEL>Grüße 🌍</LABEL></CHILD>` +
	`<ROWS><item><COUNT>1</COUNT><DETAIL><TEXT>ONE</TEXT><LABEL>first</LABEL></DETAIL>` +
	`<CHUNKS><item><DATA>3q2+7w==</DATA></item></CHUNKS></item>` +
	`<item><COUNT>-2</COUNT><DETAIL><TEXT>TWO</TEXT><LABEL></LABEL></DETAIL><CHUNKS></CHUNKS></item></ROWS>` +
	`<BLOB>AAEC</BLOB></INPUT>`

func TestRCEncodeDecodeRoundTrip(t *testing.T) {
	resolved, err := ResolveRecursiveClassic(rcGraph(), rcIdentity, RecursiveClassicLimits{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Kind != KindStructure {
		t.Fatalf("kind = %q, want structure", resolved.Kind)
	}
	got, err := EncodeRecursiveClassic(resolved, rcValue(t), RecursiveClassicLimits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if string(got) != rcExpectedXML {
		t.Fatalf("encode mismatch:\n got %s\nwant %s", got, rcExpectedXML)
	}
	decoded, err := DecodeRecursiveClassic(resolved, got, RecursiveClassicLimits{})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(decoded, any(rcValue(t))) {
		t.Fatalf("round-trip mismatch:\n got %#v\nwant %#v", decoded, rcValue(t))
	}
}

func TestRCEncodeInitial(t *testing.T) {
	resolved, err := ResolveRecursiveClassic(rcGraph(), rcIdentity, RecursiveClassicLimits{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got, err := EncodeRecursiveClassic(resolved, map[string]any{}, RecursiveClassicLimits{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	want := `<INPUT><ID>0</ID><CHILD><TEXT></TEXT><LABEL></LABEL></CHILD><ROWS></ROWS><BLOB></BLOB></INPUT>`
	if string(got) != want {
		t.Fatalf("initial encode mismatch:\n got %s\nwant %s", got, want)
	}
	initial := InitialRecursiveClassicValue(resolved)
	wantInitial := map[string]any{
		"ID": int32(0), "CHILD": map[string]any{"TEXT": "", "LABEL": ""},
		"ROWS": []any{}, "BLOB": []byte{},
	}
	if !reflect.DeepEqual(initial, any(wantInitial)) {
		t.Fatalf("initial value mismatch:\n got %#v\nwant %#v", initial, wantInitial)
	}
}

func TestRCResolveErrors(t *testing.T) {
	// functionName mismatch
	bad := rcIdentity
	bad.FunctionName = "Z_OTHER"
	if _, err := ResolveRecursiveClassic(rcGraph(), bad, RecursiveClassicLimits{}); err == nil {
		t.Fatalf("expected identity mismatch error")
	}
	// associatedType mismatch
	bad = rcIdentity
	bad.AssociatedType = "Z_OTHER"
	if _, err := ResolveRecursiveClassic(rcGraph(), bad, RecursiveClassicLimits{}); err == nil {
		t.Fatalf("expected descriptor mismatch error")
	}
	// cyclic
	cyc := graphOf("Z_FN",
		[]metadata.Parameter{param("Z_FN", "VALUE", "I", "v", "Z_A", "structure", "Z_A")},
		snode("Z_A", rfield("B", "u", "structure", "Z_B", 1, false)),
		snode("Z_B", rfield("A", "u", "structure", "Z_A", 1, false)),
	)
	cid := RecursiveClassicIdentity{FunctionName: "Z_FN", ParameterName: "VALUE", ParameterClass: "I", AssociatedType: "Z_A", InternalType: "v"}
	if _, err := ResolveRecursiveClassic(cyc, cid, RecursiveClassicLimits{}); err == nil {
		t.Fatalf("expected cyclic error")
	}
	// unsupported scalar N
	uns := graphOf("Z_FN",
		[]metadata.Parameter{param("Z_FN", "VALUE", "I", "v", "Z_U", "structure", "Z_U")},
		snode("Z_U", sfield("VALUE", "N", 1, 8, 0)),
	)
	uid := RecursiveClassicIdentity{FunctionName: "Z_FN", ParameterName: "VALUE", ParameterClass: "I", AssociatedType: "Z_U", InternalType: "v"}
	if _, err := ResolveRecursiveClassic(uns, uid, RecursiveClassicLimits{}); err == nil {
		t.Fatalf("expected unsupported-scalar error")
	}
	// resolve depth/node bounds
	if _, err := ResolveRecursiveClassic(rcGraph(), rcIdentity, RecursiveClassicLimits{MaxDepth: ptr(2)}); err == nil {
		t.Fatalf("expected descriptor depth error")
	}
	if _, err := ResolveRecursiveClassic(rcGraph(), rcIdentity, RecursiveClassicLimits{MaxNodes: ptr(4)}); err == nil {
		t.Fatalf("expected descriptor node error")
	}
}

func TestRCEncodeErrors(t *testing.T) {
	resolved, err := ResolveRecursiveClassic(rcGraph(), rcIdentity, RecursiveClassicLimits{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// unknown field
	bad := rcValue(t)
	bad["UNKNOWN"] = 1
	if _, err := EncodeRecursiveClassic(resolved, bad, RecursiveClassicLimits{}); err == nil {
		t.Fatalf("expected unknown field error")
	}
	// limits
	for _, lim := range []RecursiveClassicLimits{
		{MaxRows: ptr(2)},
		{MaxParameterBytes: ptr(100)},
		{MaxRowBytes: ptr(80)},
	} {
		if _, err := EncodeRecursiveClassic(resolved, rcValue(t), lim); err == nil {
			t.Fatalf("expected limit error for %+v", lim)
		}
	}
}

func TestRCDecodeErrors(t *testing.T) {
	resolved, err := ResolveRecursiveClassic(rcGraph(), rcIdentity, RecursiveClassicLimits{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	cases := map[string][]byte{
		"xml prolog":  []byte(`<?xml version="1.0"?>` + rcExpectedXML),
		"doctype":     []byte(`<!DOCTYPE INPUT>` + rcExpectedXML),
		"attribute":   []byte(replaceOnce(rcExpectedXML, "<INPUT>", `<INPUT attr="x">`)),
		"cdata close": []byte(replaceOnce(rcExpectedXML, "A&#38;B", "]]>")),
		"surrogate":   []byte(replaceOnce(rcExpectedXML, "A&#38;B", "A&#xD800;B")),
		"duplicate":   []byte(replaceOnce(rcExpectedXML, "<ID>7</ID>", "<ID>7</ID><ID>8</ID>")),
		"reorder":     []byte(replaceOnce(rcExpectedXML, "<ID>7</ID><CHILD>", "<CHILD><ID>7</ID>")),
		"bad base64":  []byte(replaceOnce(rcExpectedXML, "3q2+7w==", "3q2+7w")),
		"trailing":    []byte(rcExpectedXML + "trailing"),
		"utf8 bom":    append([]byte{0xef, 0xbb, 0xbf}, []byte(rcExpectedXML)...),
		"row limit":   []byte(rcExpectedXML), // paired with MaxRows below
	}
	for name, in := range cases {
		lim := RecursiveClassicLimits{}
		if name == "row limit" {
			lim.MaxRows = ptr(2)
		}
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeRecursiveClassic(resolved, in, lim); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func replaceOnce(s, old, new string) string {
	i := indexOf(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + new + s[i+len(old):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func FuzzDecodeRecursiveClassic(f *testing.F) {
	resolved, err := ResolveRecursiveClassic(rcGraph(), rcIdentity, RecursiveClassicLimits{})
	if err != nil {
		f.Fatalf("resolve: %v", err)
	}
	f.Add([]byte(rcExpectedXML))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeRecursiveClassic(resolved, data, RecursiveClassicLimits{})
	})
}
