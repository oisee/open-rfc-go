// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
)

// The two metadata sources report a character field's length in different units:
// a parameter (FUNINT) in characters, a structure field (RFC_FIELDS) in bytes.
// Halving both made every CHAR parameter advertise half its real length, and a
// CHAR1 parameter advertise maxLength 0 — a schema that rejects every value.
func TestParameterMaxLengthIsInCharacters(t *testing.T) {
	for _, tc := range []struct {
		name   string
		length int32
		want   int
	}{
		{"CHAR255", 255, 255},
		{"CHAR32", 32, 32},
		{"CHAR1", 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := paramSchema(classicrfc.FunintParameter{
				ParameterName:  "P",
				Exid:           "C",
				InternalLength: tc.length,
			}, func(string) map[string]any { return nil })
			if got := schema["maxLength"]; got != tc.want {
				t.Fatalf("maxLength = %v, want %d", got, tc.want)
			}
		})
	}
}

func TestStructureFieldMaxLengthIsHalfTheByteLength(t *testing.T) {
	schema := structSchema(rfctypes.RfcStructureDefinition{
		Name: "RFCSI",
		Fields: []rfctypes.RfcStructureField{
			{FieldName: "RFCDEST", Exid: "C", InternalLength: 64}, // CHAR32 in UTF-16
			{FieldName: "RFCDAYST", Exid: "C", InternalLength: 2}, // CHAR1
		},
	})
	props := schema["properties"].(map[string]any)
	if got := props["RFCDEST"].(map[string]any)["maxLength"]; got != 32 {
		t.Fatalf("RFCDEST maxLength = %v, want 32", got)
	}
	if got := props["RFCDAYST"].(map[string]any)["maxLength"]; got != 1 {
		t.Fatalf("RFCDAYST maxLength = %v, want 1", got)
	}
}
