// SPDX-License-Identifier: Apache-2.0

package rfcserver

import "bytes"

// Frame classification shared by every server role.
//
// The roles in this package differ in what they do with a conversation, not in
// how they recognise the frames that carry it. Before this file each Serve*
// function repeated the same three tests inline, in three different idioms, and
// the eight keepalive bytes were spelled out in two separate variables — so a
// correction to one role silently left the others behind. The recognition lives
// here once; the roles keep their own behaviour.
//
// Layering note: this is NI-level shape recognition, deliberately shallow. It
// answers "what kind of frame is this" from length and prefix alone and never
// decodes a payload, because a server must classify a frame before it has any
// right to assume the conversation reached a state where the payload parses.

// NIPing and NIPong are the 8-byte NI keepalive frames. A peer that sends
// NI_PING blocks until it sees the pong, so a role that forgets to answer one
// stalls the conversation rather than failing it — which is why every role
// answers, and why they all answer with the same bytes.
var (
	NIPing = []byte("NI_PING\x00")
	NIPong = []byte("NI_PONG\x00")
)

// FrameKind is the shape of one received frame.
type FrameKind int

const (
	// FrameData is anything not recognised as gateway framing or a keepalive:
	// the CPIC/APPC records that carry the actual conversation.
	FrameData FrameKind = iota
	// FrameGatewayRecord is the gateway/NI record, whose fixed length
	// gatewayRecordLen (wire_constants.go) also covers the NI route hello.
	FrameGatewayRecord
	// FrameNIPing is an NI keepalive that must be answered with NIPong.
	FrameNIPing
	// FrameNIPong is the peer's answer to our own keepalive; nothing is owed
	// in reply.
	FrameNIPong
	// FrameShortKeepalive is an 8-byte frame that is neither ping nor pong.
	// It is reported separately rather than folded into FrameData because a
	// role that replies to it as if it were a record corrupts the stream.
	FrameShortKeepalive
)

// Classify reports the shape of a received frame.
func Classify(frame []byte) FrameKind {
	switch {
	case len(frame) == gatewayRecordLen:
		return FrameGatewayRecord
	case len(frame) == len(NIPing) && bytes.Equal(frame, NIPing):
		return FrameNIPing
	case len(frame) == len(NIPong) && bytes.Equal(frame, NIPong):
		return FrameNIPong
	case len(frame) == 8:
		return FrameShortKeepalive
	default:
		return FrameData
	}
}

// IsNIRouteHello reports whether the frame is the 64-byte NI route hello that
// opens a type-T (registered server) conversation, as distinct from the
// gateway's normal-client record of the same length. Only the type-T role needs
// the distinction, but the two-byte prefix that makes it belongs with the rest
// of the frame recognition rather than buried in that role's switch.
func IsNIRouteHello(frame []byte) bool {
	return len(frame) == gatewayRecordLen && frame[0] == 0x02 && frame[1] == 0x03
}
