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
