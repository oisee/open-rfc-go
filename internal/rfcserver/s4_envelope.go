// SPDX-License-Identifier: Apache-2.0

package rfcserver

import "encoding/hex"

// The S/4HANA classic response envelope ends, after the export parameters, with
// three trailing fields a live ABAP client expects (observed on A4H, SAP_BASIS
// 793): the implementing program name (0x0130), an 8-byte control/metric
// (0x0667), and a 236-byte S4 metadata block (0x0104). The metric and metadata
// are replayed as captured constants; whether the client validates the session
// GUIDs embedded in 0x0104 is still open, so this is a best-effort first cut.
var (
	s4Metric0667, _ = hex.DecodeString("0000000000207240")
	s4Meta0104, _   = hex.DecodeString("100402000c000187680000044c00000bb810040b0020ff7ffa0d78b737def6196e9325bf1597ef73feebdb51fd91c6342040000000001004040008002c0010001f001010040d00100000001f000000b30000002c000000b310041600020022100417000200411004190002000010041e00080000057a00000b30100409000338303010041d0001381004240008000006f100000b9710042800080000067600000b7c1004130045046a85a8d699600b54e1000000ac110003016a858b11995f0b55e1000000ac110003006a858b14995f0b55e1000000ac110003006a857e329dc70d31e1000000ac11000301")
)
