// SPDX-License-Identifier: Apache-2.0
//
// Go-specific fuzz targets, per the milestone rule that every decoder of
// network bytes carry one. No upstream analogue. See docs/provenance.md.

package cpic

import "testing"

func FuzzDecodeFieldChain(f *testing.F) {
	f.Add(mustHexFuzz("051401140003303031011401110004555345520111ffff0000"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		fields, err := DecodeFieldChain(data, uint16(TagSession), FieldChainLimits{})
		if err != nil {
			return
		}
		for _, x := range fields {
			if len(x.Value) < 0 {
				t.Fatal("negative length")
			}
		}
	})
}

func FuzzDecodeInitialLogonResponse(f *testing.F) {
	f.Add(mustHexFuzz("010100080101010504010003"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeInitialLogonResponse(data)
	})
}

func FuzzDecodeFunctionResultFields(f *testing.F) {
	f.Add(mustHexFuzz("05000000"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeFunctionResultFields(data)
	})
}

func mustHexFuzz(s string) []byte {
	b := make([]byte, len(s)/2)
	for i := 0; i < len(b); i++ {
		hi := fromHexNibble(s[i*2])
		lo := fromHexNibble(s[i*2+1])
		b[i] = hi<<4 | lo
	}
	return b
}

func fromHexNibble(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	default:
		return 0
	}
}
