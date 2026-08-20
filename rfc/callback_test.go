// SPDX-License-Identifier: Apache-2.0
package rfc

import (
	"testing"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

// TestFrameCallbackResponse checks that a framed callback response is a valid
// compact CPIC message the session can split and send.
func TestFrameCallbackResponse(t *testing.T) {
	// A small server response ends with the 0xffff End sentinel.
	resp := append([]byte{0x05, 0x00, 0x00, 0x00}, make([]byte, 40)...)
	resp = append(resp, 0xff, 0xff)
	framed := frameCallbackResponse(resp)
	if len(framed) != len(resp)+8 {
		t.Fatalf("framed length = %d, want %d", len(framed), len(resp)+8)
	}
	fr, err := cpic.InspectRequestAppcFraming(framed)
	if err != nil {
		t.Fatalf("InspectRequestAppcFraming: %v", err)
	}
	if fr.Mode != "compact" {
		t.Fatalf("mode = %q, want compact", fr.Mode)
	}
	if fr.ApplicationDataLength != len(resp) {
		t.Fatalf("ApplicationDataLength = %d, want %d", fr.ApplicationDataLength, len(resp))
	}
	if fr.FinalSapParameterLength != 8 {
		t.Fatalf("FinalSapParameterLength = %d, want 8", fr.FinalSapParameterLength)
	}
}
