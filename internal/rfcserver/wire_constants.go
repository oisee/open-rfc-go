// SPDX-License-Identifier: Apache-2.0

package rfcserver

// Wire constants reverse-engineered from live A4H captures. Each is explained in
// docs/discoveries/0002-wire-constants.md; the short names below let the server
// code read intent instead of bare bytes.
const (
	appcProtocol   = 0x06 // APPC record byte 0: protocol version
	appcInit       = 0x03 // APPC byte 1: CPIC-init (logon) record
	appcFSapSend   = 0xcb // APPC byte 1: F_SAP_SEND (carries CPIC/CUT data)
	appcUIDOffset  = 4    // APPC header: uid (2 bytes BE) at [4:6]
	appcConvOffset = 40   // APPC header: 8-byte conversation id at [40:48]
	appcHeaderLen  = 80   // APPC record header length; payload follows

	gatewayRecordLen  = 64   // gateway normal-client record length
	gatewayAckOffset1 = 29   // gateway reply: set to gatewayAckLevel
	gatewayAckLevel   = 0x0f // client sends 0x0e; server acks with 0x0f
	gatewayAckOffset2 = 55   // gateway reply: set to gatewayAckCaps
	gatewayAckCaps    = 0xfb // client sends 0xcb; server acks capabilities with 0xfb

	// The gateway answers with its own protocol level rather than echoing the
	// client's. Measured over fourteen conversations captured between Eclipse
	// ADT and an AS ABAP 1909 sandbox: a client offering "1100" (with 0x06 at
	// gatewayAckOffset1) and a client offering "4103" (with 0x0e) were both
	// answered "4103". Whether "4103" is that release's level or the same
	// everywhere is not known from one system, which is why it is named for
	// what it does rather than for what it might mean.
	// F_INITIALIZE, the APPC verb Eclipse opens a conversation with. The
	// system answers with the client's own first eighty bytes and three
	// changes, measured over eleven captured conversations: a level byte
	// raised, a conversation id written where the client left spaces, and the
	// client's own sequence counter copied over the 0xffff placeholder at the
	// end. Everything else is echoed exactly.
	appcInitLevelOffset = 21 // client sends 0x04, the system answers 0x06
	appcInitLevelAck    = 0x06
	appcInitConvOffset  = 40 // eight spaces in, eight ASCII digits out
	appcInitConvLength  = 8
	appcInitCounterFrom = 76 // the two bytes the system copies…
	appcInitCounterTo   = 78 // …over the 0xffff it replaces
	appcInitReplyLength = 80

	// F_ALLOCATE, the verb that opens the conversation proper. Its reply is
	// the request again with four changes, at the same eight offsets in all
	// eleven captured conversations: two status bytes, the gateway's own
	// protocol level written where the client left zeroes, and the sequence
	// the client offered at F_INITIALIZE put back over the 0xffff placeholder.
	appcAllocReplyLength = 80
	appcAllocFlagOffset  = 16
	appcAllocFlagValue   = 0x01
	appcAllocStateOffset = 21
	appcAllocStateValue  = 0x02
	appcAllocLevelOffset = 69
	appcAllocCounterAt   = 76

	gatewayAckLevelOffset = 20
	gatewayAckLevelText   = "4103"

	cutReqTag0 = 0x05 // CUT request/response prefix byte 0
	cutReqTag1 = 0x02 // CUT request prefix byte 1 (0x00 for a response)
	cutRespT1  = 0x00 // CUT response prefix byte 1
)

// Tags Eclipse's calls use that the classic and S/4 vocabularies above do not.
//
// Measured over a live Eclipse↔A4H session (2026-09-15). The compact family
// carries SADT_REST_RFC_ENDPOINT's recursive parameters as SAP Binary XML —
// see internal/bxml — instead of the xRFC boundaries 0x3c02/0x3c05:
//
//	4000  two bytes: 01 and a flag, 00 = the payload follows as it is,
//	      01 = the payload is a raw DEFLATE stream (Eclipse sends 00,
//	      the system answers 01)
//	4001  the request payload, in chunks of at most 16384 bytes
//	4002  the response payload, chunked the same way
//	4004  empty, closes the parameter
//
// The table family is how a server refers to a table the client sent: the
// client numbers each table (0x0330 after its name), and the answer names it by
// number rather than by name.
const (
	tagCompactFlag     = 0x4000
	tagCompactRequest  = 0x4001
	tagCompactResponse = 0x4002
	tagCompactEnd      = 0x4004
	compactChunkLength = 16384

	tagTableID       = 0x0330 // request: the number the client gives a table
	tagFirstTableID  = 0x0331 // response: the first table's number, right after 0x0500
	tagTableIDHeader = 0x0335 // response: 0000000a, the number, the row count
	tagTableRow      = 0x0303 // response: one row, as it is
	tagTableIDEnd    = 0x0336 // response: the number again, after the rows
	tagCompactClose  = 0x0523 // response: empty, after the compact parameter

	// header byte 30 of a data record: 5 on the last record of a message,
	// 1 on one that continues
	recordInfoFinal = 0x05

	// what the system says a single response record may carry before it
	// continues in an F_RECEIVE record; the value it writes at header
	// offset 48 of every data record
	maxRecordData = 28000
)
