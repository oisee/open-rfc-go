// SPDX-License-Identifier: Apache-2.0
package cpic

import (
	"encoding/binary"
	"encoding/hex"
	"unicode/utf16"
)

func hexEncode(b []byte) string { return hex.EncodeToString(b) }

func fromUTF16LE(b []byte) string {
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(units))
}
