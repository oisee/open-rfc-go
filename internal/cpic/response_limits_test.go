// SPDX-License-Identifier: Apache-2.0
//
// ResponseLimits: the field count of one function response is bounded by
// DefaultMaxFieldCount unless the caller configures another limit. Original
// work. See docs/provenance.md.

package cpic

import (
	"errors"
	"strings"
	"testing"
)

func TestDecodeFunctionResultFieldsWithLimits(t *testing.T) {
	response := concat(hx(t, "05000000"), mustChain(t, TagResponseStart, []Field{
		fld(TagResponseContext, nil), fld(TagSession, bytesRep(1, 16)), fld(TagUnresolved0420, make([]byte, 4)),
		fld(TagCallContext, nil), fld(TagParameterName, utf16leBytes("RESULT")), fld(TagParameterValue, utf16leBytes("x")), fld(TagEnd, nil),
	}), []byte{0xff, 0xff})

	for _, limits := range []ResponseLimits{{}, {MaxFieldCount: 7}, {MaxFieldCount: 2 * DefaultMaxFieldCount}} {
		decoded, err := DecodeFunctionResultFieldsWithLimits(response, limits)
		if err != nil || !decoded.Success || len(decoded.Fields) != 7 {
			t.Fatalf("%+v: decoded = %+v, %v", limits, decoded, err)
		}
	}
	if _, err := DecodeFunctionResultFieldsWithLimits(response, ResponseLimits{MaxFieldCount: 6}); !errors.Is(err, ErrRange) ||
		!strings.Contains(err.Error(), "field count exceeds") {
		t.Fatalf("a limit below the field count: want ErrRange, got %v", err)
	}
	if _, err := DecodeFunctionResultFieldsWithLimits(response, ResponseLimits{MaxFieldCount: -1}); !errors.Is(err, ErrRange) {
		t.Fatalf("a negative limit: want ErrRange, got %v", err)
	}
}
