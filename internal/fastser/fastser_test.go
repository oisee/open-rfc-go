// SPDX-License-Identifier: Apache-2.0

package fastser

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// A real RFC_SYSTEM_INFO response CUT captured live from A4H (SAP_BASIS 793).
// Its 0x5001 container holds the RFCSI_EXPORT structure as character fields.
const sysInfoResponseHex = "05000000050005030000050305140010c905856a6099510be1000000ac1100030514042000040000000004205001010424480303004103002300c0250000510c52464353495f4558504f5254441400060600060800060600060600064000061000061000061000064000061400060800060a00061400060c00060200061e00060800064000061800065a004f300031003100430480343130334f4c00490054004f490045003300431180766863616c61346863695f4134485f3030430880766863616c613468430380413448430380413448430a80766863616c613468636943038048444243038037353843058020203339304305804c696e757843068020202020203049430a803137322e31372e302e33430380373933430a80766863616c613468636949430a803137322e31372e302e33455001013000505300410050004c0053005200460043002000200020002000200020002000200020002000200020002000200020002000200020002000200020002000200020002000200020002000200020002000200001300667000800000000009aa7400667010400ec100402000c000187680000044c00000bb810040b0020ff7ffa0d78b737def6196e9325bf1597ef73feebdb51fd91c6342040000000001004040008002c0010001f001010040d00100000001f000000b30000002c000000b310041600020022100417000200411004190002000010041e00080000057a00000b30100409000338303010041d0001381004240008000006f100000b9710042800080000067600000b7c1004130045046a8500e799600b56e1000000ac110003016a8505c499600b51e1000000ac110003006a8505c799600b51e1000000ac110003006a8566099dc20d36e1000000ac110003010104ffff0000ffff0000029200006d60"

func TestDecodeCharFieldsRFCSI(t *testing.T) {
	payload, err := hex.DecodeString(sysInfoResponseHex)
	if err != nil {
		t.Fatal(err)
	}
	fields := DecodeCharFields(payload)
	if len(fields) < 10 {
		t.Fatalf("expected the RFCSI character fields, got %d", len(fields))
	}
	var got []string
	for _, f := range fields {
		got = append(got, string(f.Value))
	}
	// Values that must appear, from the live system identity.
	for _, want := range []string{"A4H", "Linux", "172.17.0.3", "vhcala4hci_A4H_00", "793", "HDB"} {
		found := false
		for _, g := range got {
			if g == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected a field %q among %v", want, got)
		}
	}
}

func TestCharFieldRoundTrip(t *testing.T) {
	for _, v := range []string{"A4H", "172.17.0.3", "vhcala4hci_A4H_00", "X"} {
		enc := EncodeCharField([]byte(v))
		if enc[0] != charTag || enc[2] != charFlag || int(enc[1]) != len(v) {
			t.Fatalf("bad framing for %q: %x", v, enc)
		}
		dec := DecodeCharFields(enc)
		if len(dec) != 1 || !bytes.Equal(dec[0].Value, []byte(v)) {
			t.Fatalf("round-trip failed for %q: %v", v, dec)
		}
	}
}

func FuzzDecodeCharFields(f *testing.F) {
	f.Add([]byte{charTag, 0x03, charFlag, 'A', '4', 'H'})
	f.Add([]byte{charTag, 0xff, charFlag})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		fields := DecodeCharFields(b) // must not panic or read out of bounds
		for _, fld := range fields {
			if len(fld.Value) == 0 || len(fld.Value) > 0xff {
				t.Fatalf("decoded field with out-of-range length %d", len(fld.Value))
			}
		}
	})
}
