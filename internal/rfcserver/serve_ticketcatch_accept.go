// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"encoding/binary"
	"fmt"
)

// ticketCatchAccept builds the accept from a real captured gateway response
// frame with its two return codes zeroed. We hold a *reject*, not a success, so
// this is a hypothesis: the reject's trailer carries error diagnostics (it
// contains stack addresses), which a genuine accept would not — but it is the
// only structurally-complete gateway response we have, so we send it and let the
// client tell us what it will not accept.
func ticketCatchAccept(allocate []byte) []byte {
	_ = allocate
	acc := append([]byte(nil), rejectTemplate...)
	binary.BigEndian.PutUint16(acc[8:10], 0)  // errorLength = 0
	binary.BigEndian.PutUint32(acc[32:36], 0) // appcReturnCode = 0 (was 2, CM_ALLOCATE_FAILURE_RETRY)
	binary.BigEndian.PutUint32(acc[36:40], 0) // sapReturnCode = 0 (was 679, "not registered")
	return acc
}

// rejectTemplate is a real 125-byte gateway ALLOCATE response captured live —
// the "transaction program not registered" reject. Zeroing its return codes is
// our first guess at an accept. It is a protocol control frame, not a credential.
var rejectTemplate = mustHexTC("06ca070045e7000000000000000000008000000000020000000000000100010000000002000002a73230333831323536000000000000000000000000000000000000000000000000000000000006000100000000000000000078dbff7f00002c4378dbff7f000070096a85cfd499600b53e1000000ac11000300000000")

func mustHexTC(s string) []byte {
	b := make([]byte, len(s)/2)
	for i := 0; i < len(b); i++ {
		var v int
		if _, err := fmt.Sscanf(s[2*i:2*i+2], "%02x", &v); err != nil {
			panic(err)
		}
		b[i] = byte(v)
	}
	return b
}

// niPongTC is the 8-byte NI keepalive reply ("NI_PONG\\0").
var niPongTC = []byte("NI_PONG\x00")

// frameFn returns the APPC function-code byte for logging, or 0 for a short frame.
func frameFn(b []byte) byte {
	if len(b) > 1 {
		return b[1]
	}
	return 0
}
