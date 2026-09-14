// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"encoding/hex"
)

// The answer to Eclipse's logon.
//
// An SM59 client logs on with its own APPC verb and the CPIC-init branch
// answers it. Eclipse does not: it opens the conversation with F_INITIALIZE
// and F_ALLOCATE, and only then sends the logon, wrapped inside an F_SAP_SEND
// whose payload is not a CUT function request at all. Fed to the call decoder
// it says "bad CUT request prefix", which is exactly what a live Eclipse got
// from this server on 2026-09-14 after the handshake had otherwise worked.
//
// The answer is a template. Across eleven captured conversations the exchange
// is always 374 bytes in and 746 out, and those 746 differ in nineteen bytes:
//
//	7          the server's own, 0x08 in ten of eleven and 0x06 or 0x07 once
//	40..47     the conversation id, the same one F_INITIALIZE handed out
//	77, 79     the sequence the client offered, as everywhere else here
//	606..621   the client's session GUID, echoed whole
//	735..736   server-generated, correlating with nothing in the request
//
// The GUID is the one that nearly went wrong. The difference analysis said ten
// bytes varied at 606, so the first version copied ten — and the field is
// sixteen. The last six are the node part of the machine the captures were
// taken on, identical in all eleven only because there was one machine. A
// client on another host would have been handed six bytes of somebody else's
// network card, and Eclipse says so plainly: "Invalid uuid detected, own …
// given …". So the field is found by its tag rather than its offset, and
// copied whole.
//
// The last of those is taken from the template and left alone. It is not
// derived from anything the client sent and nothing suggests the client reads
// it back; if that turns out to be wrong, a live Eclipse will say so in one
// attempt, which is cheaper than deriving it from first principles.
//
// The template did carry identifiers, and a comment here used to say it did
// not. It said so because the check behind it looked for runs of printable
// ASCII, and every string in this structure is UTF-16LE: a NUL after each
// character, so a ten-letter host name never looks like four printable bytes.
// The scan reported clean and the template held the captured system's host
// name, its instance, its address, the logon string and the user. A sixth
// identifier was not a string at all — the last six bytes of the session GUID
// are the client's own IPv4, packed, which is how a LAN address travels
// through a uuid without ever spelling itself out.
//
// All of them are replaced here by same-length documentation values, so every
// offset above still counts to the same place, and the fields the bridge
// overwrites anyway — the conversation id and the GUID — are zeroed rather
// than left plausible. A nil uuid is a better failure than a stale one: it is
// obviously wrong the first time somebody sees it.
//
// The values now describe this bridge rather than somebody's system, which is
// what they should have been from the start: we are not A4H, and answering
// Eclipse with A4H's name was a lie as well as a leak.
const eclipseLogonAcceptHex = "" +
	"06cb0200ffff0006000000000001000001000001f4020000000100080000050c00000000000000003030303030303030" +
	"00006d60000000020000029a000000010000000000343130330000000000000001010008010101050401000301010103" +
	"000400000e0b01030106000b040100030103020000002301060161000100016100160008310031003000300000160450" +
	"00064f00530044000450045100144f005300440020004200520049004400470045000451045200043000300004520453" +
	"00146f00730064002d0062007200690064006700650004530007001e3100390032002e0030002e0032002e0031002000" +
	"2000200020002000200000070020005c0000000000000000000000000000000000000000000000000000000000000000" +
	"000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" +
	"000000000000000000000000002000210014000000000000000000000000000000000000000000210018001431003900" +
	"32002e0030002e0032002e00310020000018000800226f00730064002d006200720069006400670065005f004f005300" +
	"44005f003000300000080011000233000011001300083700390033002000001300120008370035003800200000120006" +
	"00224f00530044005f003000300031005f006200720069006400670065005f0065006e00000601300010530041005000" +
	"4c0053005900530055000130015000184f00530044005500530045005200200020002000200020000150015100063000" +
	"300031000151015200024500015205000000050005030000050305140010000000000000000000000000000000000514" +
	"04200004000000000420051200000512013000505300410050004c005300590053005500200020002000200020002000" +
	"200020002000200020002000200020002000200020002000200020002000200020002000200020002000200020002000" +
	"2000200001300667000800000000007086400667ffff0000ffff"

const (
	logonAcceptLength      = 746
	logonAcceptServerByte  = 7
	logonAcceptServerValue = 0x08
	logonAcceptConvAt      = 40
	logonAcceptSeqFirst    = 77
	logonAcceptSeqSecond   = 79
	logonAcceptGUIDAt      = 606
	logonAcceptGUIDLength  = 16
)

// Who the bridge says it is. Eclipse is configured with a system id before it
// connects, and the logon answer carries one back; scrubbing the captured A4H
// out of the template and leaving a hard-coded OSD in its place traded a leak
// for a lie that happened to be visible — the uuid error gave way to
// "Unexpected exception in logon operation" in the same step. So these come
// from the bridge's own configuration, and the default is deliberately not a
// real system's name.
type LogonIdentity struct {
	SystemID string // three characters, as every SAP system id is
	Host     string
	User     string
	Client   string
	Language string
}

// Fixed-width slots. The template's strings sit inside length-prefixed records
// whose lengths are bytes elsewhere in the structure, so a value of a
// different length would need those rebuilt; until something needs that, a
// value is padded or cut to the width the template already has. A host name
// longer than ten characters is therefore reported truncated rather than
// wrong — visible, and not a protocol error.
const (
	logonSystemIDAt, logonSystemIDWidth = 146, 3
	logonDescriptionAt, logonDescWidth  = 158, 10
	logonHostAt, logonHostWidth         = 194, 10
	logonInstanceAt, logonInstanceWidth = 406, 17
	logonLogonStrAt, logonLogonStrWidth = 482, 17
	logonUserAt, logonUserWidth         = 544, 12
)

// writeUTF16Field puts value into a fixed-width UTF-16LE slot, space-padded
// the way the template pads its own.
func writeUTF16Field(reply []byte, at, width int, value string) {
	runes := []rune(value)
	if len(runes) > width {
		runes = runes[:width]
	}
	for len(runes) < width {
		runes = append(runes, ' ')
	}
	for i, r := range runes {
		reply[at+i*2] = byte(r)
		reply[at+i*2+1] = byte(r >> 8)
	}
}

// eclipseLogonAccept builds the answer to one logon out of the template.
func eclipseLogonAccept(request, convID []byte, sequence [2]byte, who LogonIdentity) ([]byte, error) {
	reply, err := hex.DecodeString(eclipseLogonAcceptHex)
	if err != nil {
		return nil, err
	}
	reply[logonAcceptServerByte] = logonAcceptServerValue
	if len(convID) == appcInitConvLength {
		copy(reply[logonAcceptConvAt:], convID)
	}
	reply[logonAcceptSeqFirst] = sequence[0]
	reply[logonAcceptSeqSecond] = sequence[1]
	if guid := sessionGUIDOf(request); guid != nil {
		copy(reply[logonAcceptGUIDAt:], guid)
	}

	if who.SystemID != "" {
		writeUTF16Field(reply, logonSystemIDAt, logonSystemIDWidth, who.SystemID)
		writeUTF16Field(reply, logonDescriptionAt, logonDescWidth, who.SystemID+" BRIDGE")
		writeUTF16Field(reply, logonHostAt, logonHostWidth, who.Host)
		writeUTF16Field(reply, logonInstanceAt, logonInstanceWidth, who.Host+"_"+who.SystemID+"_00")
		writeUTF16Field(reply, logonLogonStrAt, logonLogonStrWidth,
			who.SystemID+"_"+who.Client+"_"+who.User+"_"+who.Language)
		writeUTF16Field(reply, logonUserAt, logonUserWidth, who.User)
	}
	return reply, nil
}

// sessionGUIDOf finds the client's session GUID in a logon: the 0x0514 tag,
// a length of sixteen, and then the value.
//
// By tag and not by offset. The offset is the same in every capture taken so
// far and every one of those came from the same client on the same machine,
// which is not evidence that it is fixed — it is evidence that nothing has
// varied yet.
func sessionGUIDOf(request []byte) []byte {
	const tagAndLength = "\x05\x14\x00\x10"
	at := bytes.Index(request, []byte(tagAndLength))
	if at < 0 || at+len(tagAndLength)+logonAcceptGUIDLength > len(request) {
		return nil
	}
	return request[at+len(tagAndLength) : at+len(tagAndLength)+logonAcceptGUIDLength]
}
