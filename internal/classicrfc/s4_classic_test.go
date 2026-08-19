// SPDX-License-Identifier: Apache-2.0

package classicrfc

import (
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

// decodeS4Vector loads a live S/4HANA classic-serialization response CUT (trimmed
// at its ffff0000ffff marker) and runs the full decode: envelope + result.
func decodeS4Vector(t *testing.T, name string) Result {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	cut, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	env, err := cpic.DecodeFunctionResultFields(cut)
	if err != nil {
		t.Fatalf("envelope decode: %v", err)
	}
	if !env.Success {
		t.Fatalf("expected a successful response")
	}
	res, err := DecodeResult(env.Fields)
	if err != nil {
		t.Fatalf("result decode: %v", err)
	}
	return res
}

func TestDecodeS4NativeTable(t *testing.T) {
	// STFC_STRUCTURE: two scalars plus RFCTABLE as an S4-native table (0x0333/0x0334).
	res := decodeS4Vector(t, "s4_STFC_STRUCTURE.hex")
	if len(res.Scalars) < 2 {
		t.Errorf("expected the structure/text scalars, got %d", len(res.Scalars))
	}
	if len(res.Tables) != 1 {
		t.Fatalf("expected one table (RFCTABLE), got %d", len(res.Tables))
	}
	if got := len(res.Tables[0].Rows); got != 1 {
		t.Errorf("expected 1 table row, got %d", got)
	}
}

func TestDecodeS4MixedTable(t *testing.T) {
	// RFC_READ_TABLE: DATA and FIELDS as S4 mixed tables (0x0335 + classic rows).
	res := decodeS4Vector(t, "s4_RFC_READ_TABLE.hex")
	if len(res.Tables) < 2 {
		t.Fatalf("expected the DATA and FIELDS tables, got %d", len(res.Tables))
	}
	var sawData, sawFields bool
	for _, tb := range res.Tables {
		for _, row := range tb.Rows {
			s, err := DecodeAbapChar(row)
			if err != nil {
				continue
			}
			if strings.Contains(s, "SAP SE") {
				sawData = true
			}
			if strings.HasPrefix(s, "MANDT") {
				sawFields = true
			}
		}
	}
	if !sawData {
		t.Error("expected a T000 row containing \"SAP SE\"")
	}
	if !sawFields {
		t.Error("expected a FIELDS row for MANDT")
	}
}
