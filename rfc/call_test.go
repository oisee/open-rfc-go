// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"bytes"
	"errors"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
)

func param(name, class, exid, table string, length int32) classicrfc.FunintParameter {
	return classicrfc.FunintParameter{ParameterName: name, ParameterClass: class, Exid: exid, TableName: table, InternalLength: length}
}

// zsDef is a one-CHAR-field structure used as an import/table row type.
func zsDef() rfctypes.RfcStructureDefinition {
	return rfctypes.RfcStructureDefinition{Name: "ZS", ByteLength: 20, Fields: []rfctypes.RfcStructureField{
		{TableName: "ZS", FieldName: "F", Exid: "C", Position: 1, Offset: 0, InternalLength: 20},
	}}
}

func zsResolver(name string) (rfctypes.RfcStructureDefinition, error) {
	if name == "ZS" {
		return zsDef(), nil
	}
	return rfctypes.RfcStructureDefinition{}, errors.New("unknown struct " + name)
}

func TestEncodeDecodeScalarRoundTrip(t *testing.T) {
	cases := []struct {
		p       classicrfc.FunintParameter
		in, out any
	}{
		{param("T", "I", "C", "", 10), "hi", "hi"},
		{param("N", "I", "N", "", 4), "12", "0012"},
		{param("I", "I", "I", "", 4), 42, int32(42)},
	}
	for _, c := range cases {
		b, err := encodeScalar(c.p, c.in)
		if err != nil {
			t.Fatalf("%s encode: %v", c.p.Exid, err)
		}
		got, err := decodeScalar(c.p, b)
		if err != nil {
			t.Fatalf("%s decode: %v", c.p.Exid, err)
		}
		if got != c.out {
			t.Fatalf("%s round-trip = %v (%T), want %v", c.p.Exid, got, got, c.out)
		}
	}
}

func TestEncodeScalarBytes(t *testing.T) {
	p := param("X", "I", "X", "", 3)
	b, err := encodeScalar(p, []byte{0xaa})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte{0xaa, 0, 0}) {
		t.Fatalf("X encode = % x, want aa 00 00", b)
	}
	got, err := decodeScalar(p, b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.([]byte), []byte{0xaa, 0, 0}) {
		t.Fatalf("X decode = % x", got)
	}
}

func TestEncodeCall(t *testing.T) {
	iface := metadata.RfcFunctionInterface{Name: "Z_TEST", Parameters: []classicrfc.FunintParameter{
		param("REQUTEXT", "I", "C", "", 10),
		param("IMPORTSTRUCT", "I", "u", "ZS", 20),
		param("ECHOTEXT", "E", "C", "", 10),
		param("RFCTABLE", "T", "h", "ZS", 20),
	}}
	in := Params{"REQUTEXT": "hi", "IMPORTSTRUCT": map[string]any{"F": "x"}}
	input, err := encodeCall(iface, in, zsResolver)
	if err != nil {
		t.Fatalf("encodeCall: %v", err)
	}
	// Requested outputs: exports + tables, not imports.
	wantOut := map[string]bool{"ECHOTEXT": true, "RFCTABLE": true}
	if len(input.RequestedOutputs) != 2 {
		t.Fatalf("requestedOutputs = %v, want ECHOTEXT+RFCTABLE", input.RequestedOutputs)
	}
	for _, o := range input.RequestedOutputs {
		if !wantOut[o] {
			t.Fatalf("unexpected requested output %q", o)
		}
	}
	// Both imports present, structure encoded to its byte length.
	if len(input.Imports) != 2 {
		t.Fatalf("imports = %d, want 2", len(input.Imports))
	}
	for _, imp := range input.Imports {
		if imp.Name == "IMPORTSTRUCT" && len(imp.Value) != 20 {
			t.Fatalf("IMPORTSTRUCT encoded to %d bytes, want 20", len(imp.Value))
		}
	}
}

func TestEncodeCallUnknownParameter(t *testing.T) {
	iface := metadata.RfcFunctionInterface{Name: "Z_TEST", Parameters: []classicrfc.FunintParameter{
		param("REQUTEXT", "I", "C", "", 10),
	}}
	_, err := encodeCall(iface, Params{"NOPE": "x"}, zsResolver)
	if !errors.Is(err, ErrUnknownParameter) {
		t.Fatalf("err = %v, want ErrUnknownParameter", err)
	}
}

func TestEncodeCallTableImport(t *testing.T) {
	iface := metadata.RfcFunctionInterface{Name: "Z_TEST", Parameters: []classicrfc.FunintParameter{
		param("ROWS", "T", "h", "ZS", 20),
	}}
	in := Params{"ROWS": []map[string]any{{"F": "a"}, {"F": "bb"}}}
	input, err := encodeCall(iface, in, zsResolver)
	if err != nil {
		t.Fatalf("encodeCall: %v", err)
	}
	if len(input.Tables) != 1 || input.Tables[0].Name != "ROWS" || len(input.Tables[0].Rows) != 2 {
		t.Fatalf("table import wrong: %+v", input.Tables)
	}
	if input.Tables[0].RowByteLength != 20 {
		t.Fatalf("row byte length = %d, want 20", input.Tables[0].RowByteLength)
	}
}
