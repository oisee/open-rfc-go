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

	cutReqTag0 = 0x05 // CUT request/response prefix byte 0
	cutReqTag1 = 0x02 // CUT request prefix byte 1 (0x00 for a response)
	cutRespT1  = 0x00 // CUT response prefix byte 1
)
