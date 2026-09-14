// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// The template is patched where it should be and nowhere else.
//
// The logon request itself carries credentials, so there is no fixture of one
// and there will not be: a synthetic request serves here, because what is
// under test is which bytes move, not what they mean.
func TestEclipseLogonAcceptPatchesOnlyItsFields(t *testing.T) {
	request := make([]byte, 374)
	for i := range request {
		request[i] = byte(i % 251)
	}
	convID := []byte("01234567")
	sequence := [2]byte{0x00, 0x03}

	reply, err := eclipseLogonAccept(request, convID, sequence)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(reply) != logonAcceptLength {
		t.Fatalf("the reply is %d bytes, and the system always sends %d", len(reply), logonAcceptLength)
	}
	if reply[logonAcceptServerByte] != logonAcceptServerValue {
		t.Errorf("byte %d is 0x%02x", logonAcceptServerByte, reply[logonAcceptServerByte])
	}
	if got := string(reply[logonAcceptConvAt : logonAcceptConvAt+appcInitConvLength]); got != "01234567" {
		t.Errorf("the conversation id reads %q", got)
	}
	if reply[logonAcceptSeqFirst] != 0x00 || reply[logonAcceptSeqSecond] != 0x03 {
		t.Errorf("the sequence reads %02x %02x", reply[logonAcceptSeqFirst], reply[logonAcceptSeqSecond])
	}
}

// The session GUID comes back whole, and from wherever the client put it.
//
// This is the test the first version would have failed. It copied ten bytes of
// the sixteen and passed everything above, because the six it left behind were
// the node part of the machine every capture came from and so were already
// right in the template. Eclipse compared its own uuid against ours and said
// they differed. So the GUID here is deliberately unlike the captured one in
// its last six bytes, and it sits at an offset the captures never showed.
func TestEclipseLogonAcceptEchoesTheWholeSessionGUID(t *testing.T) {
	guid := []byte{
		0x6e, 0xe6, 0x72, 0x50, 0xb0, 0x4e, 0x11, 0xf1,
		0x97, 0x08, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
	}
	request := make([]byte, 400)
	copy(request[300:], append([]byte{0x05, 0x14, 0x00, 0x10}, guid...))

	reply, err := eclipseLogonAccept(request, []byte("01234567"), [2]byte{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	got := reply[logonAcceptGUIDAt : logonAcceptGUIDAt+logonAcceptGUIDLength]
	if !bytes.Equal(got, guid) {
		t.Errorf("the reply carries % x, and the client sent % x", got, guid)
	}
}

// No GUID in the request leaves the template's bytes alone rather than writing
// a partial one, because a truncated uuid is worse than a stale uuid: Eclipse
// would report a mismatch it cannot act on.
func TestEclipseLogonAcceptWithoutAGUIDPatchesNothing(t *testing.T) {
	reply, err := eclipseLogonAccept(make([]byte, 400), []byte("01234567"), [2]byte{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	template, err := hex.DecodeString(eclipseLogonAcceptHex)
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	at, length := logonAcceptGUIDAt, logonAcceptGUIDLength
	if !bytes.Equal(reply[at:at+length], template[at:at+length]) {
		t.Errorf("the GUID field was touched: % x", reply[at:at+length])
	}
}

// A short request must not take the server down with it.
func TestEclipseLogonAcceptSurvivesAShortRequest(t *testing.T) {
	reply, err := eclipseLogonAccept(make([]byte, 90), []byte("00000001"), [2]byte{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(reply) != logonAcceptLength {
		t.Fatalf("the reply is %d bytes", len(reply))
	}
}
