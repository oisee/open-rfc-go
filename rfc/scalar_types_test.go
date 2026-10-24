// SPDX-License-Identifier: Apache-2.0
package rfc

import (
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
)

// TestScalarTypeDelegation round-trips scalar types that the fast path does not
// handle directly (FLOAT, INT8, DATE) through the per-field structure codec, to
// confirm encodeScalar/decodeScalar delegate correctly.
func TestScalarTypeDelegation(t *testing.T) {
	cases := []struct {
		name string
		p    classicrfc.FunintParameter
		in   any
		want any
	}{
		{"FLOAT", classicrfc.FunintParameter{ParameterName: "F", Exid: "F", InternalLength: 8}, float64(3.5), float64(3.5)},
		{"INT8", classicrfc.FunintParameter{ParameterName: "L", Exid: "8", InternalLength: 8}, int64(1234567890123), int64(1234567890123)},
		{"DATE", classicrfc.FunintParameter{ParameterName: "D", Exid: "D", InternalLength: 16}, "20260820", "20260820"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := encodeScalar(c.p, c.in)
			if err != nil {
				t.Fatalf("encodeScalar %s: %v", c.name, err)
			}
			got, err := decodeScalar(c.p, b)
			if err != nil {
				t.Fatalf("decodeScalar %s: %v", c.name, err)
			}
			if got != c.want {
				t.Fatalf("%s round-trip: got %#v (%T), want %#v", c.name, got, got, c.want)
			}
		})
	}
}

// TestScalarTemporalWidth covers DATS and TIMS scalars whose RFC_FUNINT length
// is given in characters (8, 6), as for CHAR, instead of in bytes (16, 12).
// Both forms are 8 or 6 characters on the wire, as open-rfc encodes them.
func TestScalarTemporalWidth(t *testing.T) {
	cases := []struct {
		name string
		p    classicrfc.FunintParameter
		in   string
		size int
	}{
		{"DATS 8", classicrfc.FunintParameter{ParameterName: "D", Exid: "D", InternalLength: 8}, "20260820", 16},
		{"DATS 16", classicrfc.FunintParameter{ParameterName: "D", Exid: "D", InternalLength: 16}, "20260820", 16},
		{"TIMS 6", classicrfc.FunintParameter{ParameterName: "T", Exid: "T", InternalLength: 6}, "235959", 12},
		{"TIMS 12", classicrfc.FunintParameter{ParameterName: "T", Exid: "T", InternalLength: 12}, "235959", 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, err := encodeScalar(c.p, c.in)
			if err != nil {
				t.Fatalf("encodeScalar: %v", err)
			}
			if len(b) != c.size {
				t.Fatalf("encoded %d bytes, want %d", len(b), c.size)
			}
			got, err := decodeScalar(c.p, b)
			if err != nil || got != c.in {
				t.Fatalf("decodeScalar = %#v, %v; want %q", got, err, c.in)
			}
		})
	}
}
