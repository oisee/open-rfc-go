// SPDX-License-Identifier: Apache-2.0
//
// DecodeOptions.LenientTemporal: decode keeps a DATS or TIMS value that is not
// YYYYMMDD or HHMMSS as its raw text. Encode stays strict. Original work. See
// docs/provenance.md.

package structure

import (
	"errors"
	"testing"

	"github.com/oisee/open-rfc-go/internal/rfctypes"
)

func temporalDef(dateExid, timeExid string) rfctypes.RfcStructureDefinition {
	return rfctypes.RfcStructureDefinition{
		Name:       "ZTEST",
		ByteLength: 28,
		Fields: []rfctypes.RfcStructureField{
			fld("DVAL", dateExid, 1, 0, 16, 0),
			fld("TVAL", timeExid, 2, 16, 12, 0),
		},
	}
}

func TestDecodeLenientTemporal(t *testing.T) {
	// Write values that are not a date or a time through CHAR fields of the
	// same geometry.
	raw, err := Encode(temporalDef("C", "C"), map[string]any{"DVAL": "2026AB01", "TVAL": "25:00 "})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(temporalDef("D", "T"), raw); err == nil {
		t.Fatal("strict decode accepted an invalid date")
	}
	got, err := DecodeWithOptions(temporalDef("D", "T"), raw, DecodeOptions{LenientTemporal: true})
	if err != nil {
		t.Fatalf("lenient decode failed: %v", err)
	}
	if got["DVAL"] != "2026AB01" || got["TVAL"] != "25:00 " {
		t.Fatalf("got DVAL=%q TVAL=%q", got["DVAL"], got["TVAL"])
	}

	// Valid values decode the same either way.
	valid, err := Encode(temporalDef("D", "T"), map[string]any{"DVAL": "20261012", "TVAL": "235959"})
	if err != nil {
		t.Fatal(err)
	}
	for _, opts := range []DecodeOptions{{}, {LenientTemporal: true}} {
		got, err := DecodeWithOptions(temporalDef("D", "T"), valid, opts)
		if err != nil || got["DVAL"] != "20261012" || got["TVAL"] != "235959" {
			t.Fatalf("%+v, valid values: got %v, err %v", opts, got, err)
		}
	}

	// Encode stays strict.
	if _, err := Encode(temporalDef("D", "T"), map[string]any{"DVAL": "2026AB01", "TVAL": "235959"}); err == nil {
		t.Fatal("encode accepted an invalid date")
	}
}

// A field that is not a date or a time is decoded as before, and its errors are
// not hidden by the option.
func TestDecodeLenientTemporalKeepsOtherErrors(t *testing.T) {
	def := temporalDef("D", "T")
	if _, err := DecodeWithOptions(def, make([]byte, 27), DecodeOptions{LenientTemporal: true}); !errors.Is(err, ErrStructure) {
		t.Fatalf("short value: want ErrStructure, got %v", err)
	}
}
