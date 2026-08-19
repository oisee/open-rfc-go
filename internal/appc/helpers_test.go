// SPDX-License-Identifier: Apache-2.0
package appc

import (
	"encoding/hex"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

func hexOf(b []byte) string { return hex.EncodeToString(b) }
