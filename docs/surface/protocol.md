# Surface inventory: src/protocol/ (framing, handshakes, conversation records)

> Mechanical inventory of open-rfc @ commit 847036d, generated as porting input. Every claim cites path:line. See ../provenance.md.
> src/protocol/ni.ts is excluded: it is already ported as internal/ni.

Citation paths are relative to the open-rfc checkout root. Claim types are marked:
**(code)** = quoted from source; **(test)** = quoted from a test assertion or test name;
**(comment)** = quoted from a source comment; **INFERRED:** = my reading, not written down.

Files in scope: `appc.ts` (2089 lines), `cpic.ts` (1970), `rfcpro.ts` (131), `gateway.ts` (192),
`message-server.ts` (448), `password-scramble.ts` (56), `rfc-error-envelope.ts` (657),
`classic-rfc.ts` (607).

All eight files import `CheckedByteReader` / `CheckedByteWriter` / `intrinsicUint8ArrayByteLength` /
`intrinsicUint8ArrayView` / `snapshotUint8Array` from `./bytes.js` (e.g. src/protocol/appc.ts:1-6,
src/protocol/rfcpro.ts:1, src/protocol/classic-rfc.ts:1-6). `bytes.ts` is not in scope for this
inventory; the porter needs its equivalent before any of these decoders can be ported.

---

## src/protocol/appc.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `APPC_PROTOCOL_VERSION` | const | `export const APPC_PROTOCOL_VERSION = 0x06;` | src/protocol/appc.ts:8 |
| `APPC_COMMON_HEADER_LENGTH` | const | `export const APPC_COMMON_HEADER_LENGTH = 48;` | src/protocol/appc.ts:9 |
| `APPC_RECORD_HEADER_LENGTH` | const | `export const APPC_RECORD_HEADER_LENGTH = 80;` | src/protocol/appc.ts:11 |
| `APPC_EXTENDED_INITIALIZE_OPTIONS_LENGTH` | const | `export const APPC_EXTENDED_INITIALIZE_OPTIONS_LENGTH = 341;` | src/protocol/appc.ts:12 |
| `APPC_INITIALIZE_PARAMETERS_LENGTH` | const | `export const APPC_INITIALIZE_PARAMETERS_LENGTH = 373;` | src/protocol/appc.ts:13 |
| `APPC_PARTNER_PARAMETERS_LENGTH` | const | `export const APPC_PARTNER_PARAMETERS_LENGTH = 144;` | src/protocol/appc.ts:14 |
| `APPC_VECTOR_END_OF_MESSAGE` | const | `export const APPC_VECTOR_END_OF_MESSAGE = 0x04;` | src/protocol/appc.ts:15 |
| `APPC_FINAL_SAP_PARAMETER_LENGTH` | const | `export const APPC_FINAL_SAP_PARAMETER_LENGTH = 8;` | src/protocol/appc.ts:17 |
| `MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH` | const | `export const MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH = 28_000;` | src/protocol/appc.ts:19 |
| `MAX_APPC_ASYNC_SENDS_BEFORE_SYNC` | const | `export const MAX_APPC_ASYNC_SENDS_BEFORE_SYNC = 21;` | src/protocol/appc.ts:21 |
| `MAX_APPC_DATA_FRAGMENT_LENGTH` | const | `export const MAX_APPC_DATA_FRAGMENT_LENGTH =` `  MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH;` | src/protocol/appc.ts:23-24 |
| `MAX_APPC_OUTGOING_MESSAGE_LENGTH` | const | `export const MAX_APPC_OUTGOING_MESSAGE_LENGTH = 0x7fff_ffff;` | src/protocol/appc.ts:26 |
| `DEFAULT_MAX_APPC_OUTGOING_MESSAGE_LENGTH` | const | `export const DEFAULT_MAX_APPC_OUTGOING_MESSAGE_LENGTH = 1_400_000;` | src/protocol/appc.ts:28 |
| `DEFAULT_MAX_APPC_MESSAGE_LENGTH` | const | `export const DEFAULT_MAX_APPC_MESSAGE_LENGTH = 256 * 1024 * 1024;` | src/protocol/appc.ts:29 |
| `DEFAULT_MAX_APPC_MESSAGE_FRAGMENTS` | const | `export const DEFAULT_MAX_APPC_MESSAGE_FRAGMENTS = 65_536;` | src/protocol/appc.ts:30 |
| `AppcCpicStreamingPolicy` | type | `export type AppcCpicStreamingPolicy = "disabled" \| "enabled";` | src/protocol/appc.ts:38 |
| `AppcFunction` | enum | `Initialize = 0x01, Allocate = 0x05, SendData = 0x07, AsyncSendData = 0x08, Receive = 0x09, AsyncReceive = 0x0a, Deallocate = 0x0b, SetTpName = 0x0d, SetPartnerLuName = 0x0f, Flush = 0x1b, SapSend = 0xcb` | src/protocol/appc.ts:40-52 |
| `AppcPayloadInfo` | interface | `protocolVersion: number; functionCode: number; functionName: string` (all `readonly`) | src/protocol/appc.ts:54-58 |
| `AppcHeader` | interface | `extends AppcPayloadInfo` + `protocol, mode, uid, gatewayId, errorLength, info2, traceLevel, time, info3, timeout, info4, sequenceNumber, sapParameterLength, padding, info, vector, appcReturnCode, sapReturnCode: number` and `conversationId: Buffer` | src/protocol/appc.ts:60-80 |
| `AppcDataFragment` | interface | `header: AppcHeader; data: Buffer; isFinal: boolean` | src/protocol/appc.ts:82-86 |
| `AppcExtendedInfo` | interface | `shortDestinationName, logicalUnitName, transactionProgramName: string; connectionType, clientInfo, communicationIndex, connectionIndex: number` | src/protocol/appc.ts:88-96 |
| `AppcAsyncDataInfo` | interface | `dataLength, communicationIndex, connectionIndex: number` | src/protocol/appc.ts:99-103 |
| `AppcDataOperationInfo` | interface | `dataLength, communicationIndex, connectionIndex: number` | src/protocol/appc.ts:106-110 |
| `AppcIncomingDataOperationInfo` | interface | `dataLength, communicationIndex, connectionIndex: number` | src/protocol/appc.ts:113-117 |
| `AppcSynchronousSendAcknowledgement` | interface | `header: AppcHeader; connectionIndex: number` | src/protocol/appc.ts:119-122 |
| `AppcPartnerLogicalUnitInfoInput` | interface | `logicalUnitName: string; partnerHostAddress: Uint8Array; communicationIndex, connectionIndex: number` | src/protocol/appc.ts:125-130 |
| `AppcPartnerLogicalUnitInfo` | interface | `logicalUnitNamePrefix: string; logicalUnitNameLength: number; partnerHostAddress: Uint8Array; communicationIndex, connectionIndex: number` | src/protocol/appc.ts:133-139 |
| `AppcExtendedInitializeOptions` | interface | `optionFlags: number; rootId, connectionId: string; connectionIdSuffix, timeout, keepaliveTimeout, exportTrace, startType, networkProtocol: number; localAddressV6: Uint8Array; longLogicalUnitName, operatingSystemUser: string; localAddressV4: Uint8Array; longTransactionProgramName: string` | src/protocol/appc.ts:141-156 |
| `AppcInitializeParameters` | interface | `clientIdentifier: string; options: AppcExtendedInitializeOptions` | src/protocol/appc.ts:158-161 |
| `AppcPartnerLogicalUnitParameters` | interface | `longLogicalUnitName: string; partnerHostAddress: Uint8Array` | src/protocol/appc.ts:163-166 |
| `AppcRecordHeaderInput` | interface | every header field optional: `protocol?, mode?, uid?, gatewayId?, errorLength?, info2?, traceLevel?, time?, info3?, timeout?, info4?, sequenceNumber?, padding?, info?, vector?, appcReturnCode?, sapReturnCode?: number; conversationId?: Uint8Array` | src/protocol/appc.ts:168-187 |
| `AppcControlRecordInput` | interface | `extends AppcRecordHeaderInput` + `functionCode: AppcFunction; extendedInfo?: AppcExtendedInfo; partnerLogicalUnitInfo?: AppcPartnerLogicalUnitInfoInput; parameters?: Uint8Array` | src/protocol/appc.ts:189-194 |
| `AppcDataRecordInput` | interface | `extends AppcRecordHeaderInput` + `functionCode?: AppcFunction.SapSend \| AppcFunction.SendData \| AppcFunction.Receive; data: Uint8Array; communicationIndex: number; connectionIndex: number; isFinal?: boolean` | src/protocol/appc.ts:196-205 |
| `AppcOutgoingDataPlanInput` | interface | `extends AppcRecordHeaderInput` + `applicationData: Uint8Array; finalSapParameters?: Uint8Array; communicationIndex: number; connectionIndex: number` | src/protocol/appc.ts:207-214 |
| `AppcOutgoingDataPlannerOptions` | interface | `maxApplicationDataLength?: number; maxFragments?: number; cpicStreaming?: AppcCpicStreamingPolicy` | src/protocol/appc.ts:216-221 |
| `AppcOutgoingDataFragment` | interface | `extends AppcRecordHeaderInput` + `functionCode: SapSend\|SendData\|AsyncSendData\|Receive; fragmentIndex, fragmentCount: number; conversationId: Buffer; sequenceNumber: number; applicationData: Buffer; finalSapParameters: Buffer; messageApplicationDataLength, communicationIndex, connectionIndex: number; isFinal: boolean; info, vector: number; sapParameterLength: 0 \| 8` | src/protocol/appc.ts:230-250 |
| `AppcClientSetupState` | type | `"new" \| "initialize-pending" \| "initialized" \| "tp-set" \| "partner-set" \| "allocate-pending" \| "send-continuation" \| "send-barrier-pending" \| "response-pending" \| "ready" \| "closed"` | src/protocol/appc.ts:252-263 |
| `AppcReceiveDisposition` | type | `export type AppcReceiveDisposition =` `  \| "accepted"` `  \| "normal-deallocation";` | src/protocol/appc.ts:265-267 |
| `AppcPeerReturnCodeError` | class | `extends Error` with `readonly appcReturnCode: number; readonly sapReturnCode: number;` and `constructor(functionName: string, appcReturnCode: number, sapReturnCode: number)` | src/protocol/appc.ts:270-287 |
| `AppcNormalDeallocationWithoutDataError` | class | `extends Error`, zero-argument constructor | src/protocol/appc.ts:290-295 |
| `AppcMessage` | interface | `data: Buffer; conversationId: Buffer; sequenceNumber, fragmentCount, communicationIndex, connectionIndex: number` | src/protocol/appc.ts:297-304 |
| `AppcConversationDecoderOptions` | interface | `maxMessageLength?: number; maxFragments?: number; allowInitialReceive?: boolean; validateIncomingDataOperationInfo?: boolean` | src/protocol/appc.ts:306-313 |
| `inspectAppcPayload` | func | `export function inspectAppcPayload(payload: Uint8Array): AppcPayloadInfo` | src/protocol/appc.ts:355 |
| `encodeAppcExtendedInfo` | func | `export function encodeAppcExtendedInfo(info: AppcExtendedInfo): Buffer` | src/protocol/appc.ts:452 |
| `decodeAppcExtendedInfo` | func | `export function decodeAppcExtendedInfo(data: Uint8Array): AppcExtendedInfo` | src/protocol/appc.ts:475 |
| `encodeAppcPartnerLogicalUnitInfo` | func | `export function encodeAppcPartnerLogicalUnitInfo(info: AppcPartnerLogicalUnitInfoInput,): Buffer` | src/protocol/appc.ts:510-512 |
| `decodeAppcPartnerLogicalUnitInfo` | func | `export function decodeAppcPartnerLogicalUnitInfo(data: Uint8Array,): AppcPartnerLogicalUnitInfo` | src/protocol/appc.ts:537-539 |
| `encodeAppcExtendedInitializeOptions` | func | `export function encodeAppcExtendedInitializeOptions(options: AppcExtendedInitializeOptions,): Buffer` | src/protocol/appc.ts:573-575 |
| `decodeAppcExtendedInitializeOptions` | func | `export function decodeAppcExtendedInitializeOptions(data: Uint8Array,): AppcExtendedInitializeOptions` | src/protocol/appc.ts:623-625 |
| `encodeAppcInitializeParameters` | func | `export function encodeAppcInitializeParameters(parameters: AppcInitializeParameters,): Buffer` | src/protocol/appc.ts:710-712 |
| `decodeAppcInitializeParameters` | func | `export function decodeAppcInitializeParameters(data: Uint8Array,): AppcInitializeParameters` | src/protocol/appc.ts:728-730 |
| `encodeAppcPartnerLogicalUnitParameters` | func | `export function encodeAppcPartnerLogicalUnitParameters(parameters: AppcPartnerLogicalUnitParameters,): Buffer` | src/protocol/appc.ts:751-753 |
| `decodeAppcPartnerLogicalUnitParameters` | func | `export function decodeAppcPartnerLogicalUnitParameters(data: Uint8Array,): AppcPartnerLogicalUnitParameters` | src/protocol/appc.ts:769-771 |
| `encodeAppcControlRecord` | func | `export function encodeAppcControlRecord(input: AppcControlRecordInput): Buffer` | src/protocol/appc.ts:803 |
| `decodeAppcDataOperationInfo` | func | `export function decodeAppcDataOperationInfo(data: Uint8Array,): AppcDataOperationInfo` | src/protocol/appc.ts:929-931 |
| `decodeAppcIncomingDataOperationInfo` | func | `export function decodeAppcIncomingDataOperationInfo(data: Uint8Array,): AppcIncomingDataOperationInfo` | src/protocol/appc.ts:966-968 |
| `decodeAppcAsyncDataInfo` | func | `export function decodeAppcAsyncDataInfo(data: Uint8Array,): AppcAsyncDataInfo` | src/protocol/appc.ts:985-987 |
| `planOutgoingAppcDataFragments` | func | `export function planOutgoingAppcDataFragments(input: AppcOutgoingDataPlanInput, options: AppcOutgoingDataPlannerOptions = {},): readonly AppcOutgoingDataFragment[]` | src/protocol/appc.ts:1033-1036 |
| `snapshotOutgoingAppcDataFragment` | func | `export function snapshotOutgoingAppcDataFragment(input: AppcOutgoingDataFragment,): AppcOutgoingDataFragment` | src/protocol/appc.ts:1254-1256 |
| `encodeOutgoingAppcDataFragment` | func | `export function encodeOutgoingAppcDataFragment(input: AppcOutgoingDataFragment,): Buffer` | src/protocol/appc.ts:1331-1333 |
| `encodeAppcDataRecord` | func | `export function encodeAppcDataRecord(input: AppcDataRecordInput): Buffer` | src/protocol/appc.ts:1449 |
| `AppcClientSetupStateMachine` | class | `export class AppcClientSetupStateMachine` — `#state`, `get state()`, `responseComplete(): void`, `sent(functionCode: AppcFunction, isFinalDataRecord = true): void`, `received(payload: Uint8Array): AppcReceiveDisposition` | src/protocol/appc.ts:1542-1682 |
| `decodeAppcHeader` | func | `export function decodeAppcHeader(payload: Uint8Array): AppcHeader` | src/protocol/appc.ts:1685 |
| `decodeAppcSynchronousSendAcknowledgement` | func | `export function decodeAppcSynchronousSendAcknowledgement(payload: Uint8Array,): AppcSynchronousSendAcknowledgement` | src/protocol/appc.ts:1724-1726 |
| `decodeAppcDataFragment` | func | `export function decodeAppcDataFragment(payload: Uint8Array): AppcDataFragment` | src/protocol/appc.ts:1781 |
| `AppcConversationDecoder` | class | `export class AppcConversationDecoder` — `constructor(options: AppcConversationDecoderOptions = {})`, `get bufferedByteLength(): number`, `get fragmentCount(): number`, `push(payload: Uint8Array): AppcMessage[]`, `pushTerminalDeallocation(payload: Uint8Array): AppcMessage[]`, `finish(): void`; private `#push`, `#checkLimits` | src/protocol/appc.ts:1800-2089 |

### Constants and magic values

| Name | Value (verbatim) | What the source says it means | Citation |
|---|---|---|---|
| `APPC_PROTOCOL_VERSION` | `0x06` | written as the first record byte `"protocolVersion"`; `inspectAppcPayload` rejects anything else | src/protocol/appc.ts:8, 1513, 365-367 |
| `APPC_COMMON_HEADER_LENGTH` | `48` | doc comment on `decodeAppcHeader`: `"Decode the fixed 48-byte APPC header shared by the observed version-6 records."` | src/protocol/appc.ts:9, 1684 |
| `APPC_RECORD_HEADER_LENGTH` | `80` | comment: `"All controlled version-6 records contain 32 bytes after the common header."` | src/protocol/appc.ts:10-11 |
| `APPC_EXTENDED_INITIALIZE_OPTIONS_LENGTH` | `341` | doc comment: `"Encode the fixed 341-byte extended initialization-options contract."` | src/protocol/appc.ts:12, 572 |
| `APPC_INITIALIZE_PARAMETERS_LENGTH` | `373` | 32-byte `clientIdentifier` + 341-byte extended options | src/protocol/appc.ts:13, 717-724 |
| `APPC_PARTNER_PARAMETERS_LENGTH` | `144` | 128-byte `longLogicalUnitName` + 16-byte `partnerHostAddress` | src/protocol/appc.ts:14, 758-765 |
| `APPC_VECTOR_END_OF_MESSAGE` | `0x04` | used as `(header.vector & APPC_VECTOR_END_OF_MESSAGE) !== 0` to compute `isFinal` | src/protocol/appc.ts:15, 1795 |
| `APPC_FINAL_SAP_PARAMETER_LENGTH` | `8` | comment: `"Fixed SAP parameter tail carried by a compact F_SAP_SEND record."` | src/protocol/appc.ts:16-17 |
| `MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH` | `28_000` | comment: `"Largest admitted STSEND/F_ASEND_DATA application slice."` | src/protocol/appc.ts:18-19 |
| `MAX_APPC_ASYNC_SENDS_BEFORE_SYNC` | `21` | comment: `"The admitted streaming contract inserts a sync after each 21 async chunks."` | src/protocol/appc.ts:20-21 |
| `MAX_APPC_DATA_FRAGMENT_LENGTH` | alias of the above | comment: `"Backwards-compatible name for the evidenced 28,000-byte application slice."` | src/protocol/appc.ts:22-24 |
| `MAX_APPC_OUTGOING_MESSAGE_LENGTH` | `0x7fff_ffff` | comment: `"Protocol-wide signed INT4 ceiling; the configured default is much smaller."` | src/protocol/appc.ts:25-26 |
| `DEFAULT_MAX_APPC_OUTGOING_MESSAGE_LENGTH` | `1_400_000` | comment: `"Two periodic barriers / 50 data chunks fit the bounded beta envelope."` | src/protocol/appc.ts:27-28 |
| `DEFAULT_MAX_APPC_MESSAGE_LENGTH` | `256 * 1024 * 1024` | decoder default for reassembled message bytes | src/protocol/appc.ts:29, 1809 |
| `DEFAULT_MAX_APPC_MESSAGE_FRAGMENTS` | `65_536` | decoder/planner default fragment cap | src/protocol/appc.ts:30, 1079, 1810 |
| `APPC_RETURN_CODE_DEALLOCATED_NORMAL` (**not exported**) | `18` | comment: `"SAP Note 63347: the peer ended the CPI-C conversation normally."` | src/protocol/appc.ts:31-32 |
| `functionNames` (**not exported**) | `Initialize→"F_INITIALIZE", Allocate→"F_ALLOCATE", SendData→"F_SEND_DATA", AsyncSendData→"F_ASEND_DATA", Receive→"F_RECEIVE", AsyncReceive→"F_ARECEIVE", Deallocate→"F_DEALLOCATE", SetTpName→"F_SET_TP_NAME", SetPartnerLuName→"F_SET_PARTNER_LU_NAME", Flush→"F_FLUSH", SapSend→"F_SAP_SEND"` | unknown codes render as `` `UNKNOWN_0x${functionCode.toString(16).padStart(2, "0")}` `` | src/protocol/appc.ts:325-337, 348-353 |
| `controlFunctions` (**not exported**) | `{Initialize, Allocate, Deallocate, SetTpName, SetPartnerLuName, Flush}` | the set `encodeAppcControlRecord` admits | src/protocol/appc.ts:339-346 |

Header field offsets are implied only by the write order in `encodeAppcRecord`
(src/protocol/appc.ts:1513-1534) and the read order in `decodeAppcHeader`
(src/protocol/appc.ts:1694-1718). Order (code): `protocolVersion u8, functionCode u8, protocol u8,
mode u8, uid u16BE, gatewayId u16BE, errorLength u16BE, info2 u8, traceLevel u8, time u32BE,
info3 u8, timeout i32BE, info4 u8, sequenceNumber u32BE, sapParameterLength u16BE, padding u16BE,
info u8, vector u8, appcReturnCode u32BE, sapReturnCode u32BE, conversationId[8]`, then a 32-byte
`operationInfo`, then trailing data.

Encoder defaults when a header field is omitted (code, src/protocol/appc.ts:1515-1532):
`protocol ?? 2`, `uid ?? 0xffff`, every other numeric field `?? 0`, `conversationId ?? Buffer.alloc(8)`.

Compact/streamed record semantics computed by `outgoingFragmentSemantics`
(src/protocol/appc.ts:869-902), verbatim: single fragment →
`functionCode: AppcFunction.SapSend`, `info: 5`, `vector: 0x0c`, `sapParameterLength: 8`; final of a
multi-fragment plan → `AppcFunction.Receive`, `info: 1`, `vector: 0`, `sapParameterLength: 0`;
periodic sync slice → `AppcFunction.SendData`, `info: 1`; every other slice →
`AppcFunction.AsyncSendData`, `info: 0`. The sync predicate is `fragmentIndex >=
MAX_APPC_ASYNC_SENDS_BEFORE_SYNC && (fragmentIndex - MAX_APPC_ASYNC_SENDS_BEFORE_SYNC) %
MAX_APPC_ASYNC_SENDS_BEFORE_SYNC === 0` (src/protocol/appc.ts:880-884).

`encodeAppcDataRecord` defaults (code, src/protocol/appc.ts:1462-1469): `info ?? (isFinal ? 5 : 1)`,
`vector ?? (isFinal ? 0x0c : functionCode === SapSend ? 0x08 : 0)`, `sapParameterLength = isFinal ? 8 : 0`.

Async data info layout (`encodeAppcAsyncDataInfo`, src/protocol/appc.ts:919-925, code):
`u16BE reserved=0, u16BE dataLength, 24 zero bytes, u16BE communicationIndex, u16BE connectionIndex`.

Incoming (server) data info layout (`decodeAppcIncomingDataOperationInfo`,
src/protocol/appc.ts:977-982, code): `dataLength = readUInt16BE(10)`,
`communicationIndex = readUInt16BE(28)`, `connectionIndex = readUInt16BE(30)`.

Canonical F_SEND_DATA acknowledgement header, all conditions verbatim from
src/protocol/appc.ts:1737-1757 (code): `functionCode === AppcFunction.SendData, protocol === 2,
mode === 0, uid === 0xffff, gatewayId === 0, errorLength === 0, info2 === 0, traceLevel === 0,
time === 0, info3 === 0, timeout === 0, info4 === 2, sequenceNumber === 0, info === 1, vector === 0,
sapParameterLength === 0, padding === 0, appcReturnCode === 0, sapReturnCode === 0`; plus
`encodedOperationInfo.subarray(0, 30)` all zero, `dataLength === 0`, `communicationIndex === 0`
(src/protocol/appc.ts:1766-1774).

### Errors

| Message text (verbatim) | Type | Condition | Citation |
|---|---|---|---|
| `` `${functionName} failed with APPC return code ${appcReturnCode} and SAP return code ${sapReturnCode}` `` | `AppcPeerReturnCodeError` | thrown by `received()` when `appcReturnCode !== 0 \|\| sapReturnCode !== 0` (after the normal-deallocation check) | src/protocol/appc.ts:280-282, 1649-1656 |
| `"connection closed without message (CM_NO_DATA_RECEIVED)"` | `AppcNormalDeallocationWithoutDataError` | terminal deallocation carrying no data record, or a data record with `data.byteLength === 0` | src/protocol/appc.ts:291, 1884, 1918 |
| `"an APPC payload needs at least a version and function byte"` | `RangeError` | `payload.byteLength < 2`, or either byte `undefined` | src/protocol/appc.ts:357, 363 |
| `` `unsupported APPC protocol version 0x${protocolVersion.toString(16)}` `` | `Error` | first byte ≠ `0x06` | src/protocol/appc.ts:366 |
| `` `${field} must contain at most 8 ASCII bytes` `` | `RangeError` | non-`[\x20-\x7e]` or >8 bytes in a fixed 8-byte name | src/protocol/appc.ts:378 |
| `` `${field} contains a non-ASCII byte` `` | `Error` | decode of a fixed 8-byte name / padded field sees a byte outside `0x20..0x7e` and not NUL | src/protocol/appc.ts:390, 420 |
| `` `${field} must contain at most ${width} ASCII bytes` `` | `RangeError` | padded-ASCII encode overflow | src/protocol/appc.ts:403 |
| `` `${field} contains data after its first padding byte` `` | `Error` | padded-ASCII decode sees non-padding after the first padding byte | src/protocol/appc.ts:425 |
| `` `${field} must be a Uint8Array` `` | `TypeError` | `exactBytes` type guard | src/protocol/appc.ts:433 |
| `` `${field} must contain exactly ${length} bytes; received ${byteLength}` `` | `RangeError` | `exactBytes` length guard | src/protocol/appc.ts:437-439 |
| `` `${field} must contain exactly 16 uppercase hexadecimal characters` `` | `RangeError` | `rootId`/`connectionId` fails `/^[0-9A-F]{16}$/` | src/protocol/appc.ts:446 |
| `` `APPC extended info needs exactly 32 bytes; received ${data.byteLength}` `` | `RangeError` | `decodeAppcExtendedInfo` length ≠ 32 | src/protocol/appc.ts:477 |
| `` `APPC extended info reserved field must be zero; received ${reserved}` `` | `Error` | u16 at offset 26 ≠ 0 | src/protocol/appc.ts:498 |
| `"logicalUnitName must contain at most 128 ASCII bytes"` | `RangeError` | `encodeAppcPartnerLogicalUnitInfo` name check | src/protocol/appc.ts:514 |
| `` `partnerHostAddress must contain exactly 16 bytes; received ${...}` `` | `RangeError` | address ≠ 16 bytes | src/protocol/appc.ts:522-524 |
| `` `APPC partner logical-unit info needs exactly 32 bytes; received ${data.byteLength}` `` | `RangeError` | decode length ≠ 32 | src/protocol/appc.ts:541-543 |
| `` `APPC partner logical-unit name length must be at most 128; received ${logicalUnitNameLength}` `` | `Error` | declared length > 128 | src/protocol/appc.ts:549-551 |
| `` `APPC partner logical-unit name prefix length ${...} does not match declared length ${logicalUnitNameLength}` `` | `Error` | prefix length ≠ `Math.min(declared, 8)` | src/protocol/appc.ts:555-559 |
| `` `APPC extended initialize options need exactly ${APPC_EXTENDED_INITIALIZE_OPTIONS_LENGTH} bytes; received ${data.byteLength}` `` | `RangeError` | length ≠ 341 | src/protocol/appc.ts:627-629 |
| `` `unsupported APPC extended initialize options version ${version}` `` | `Error` | first byte ≠ 1 | src/protocol/appc.ts:634 |
| `` `unsupported APPC extended initialize protocol ${protocolName}` `` | `Error` | protocol-name field ≠ `"CPIC"` | src/protocol/appc.ts:643 |
| `` `APPC extended initialize ${field} must be zero` `` | `Error` | any of `reserved1` (16), `reserved2` (8), `reserved3` (4), `reserved4` (12), `reserved5` (16) non-zero | src/protocol/appc.ts:671, 687 |
| `"APPC extended initialize reserved6 must be zero"` | `Error` | 4-byte `reserved6` non-zero | src/protocol/appc.ts:693 |
| `` `APPC initialize parameters need exactly ${APPC_INITIALIZE_PARAMETERS_LENGTH} bytes; received ${data.byteLength}` `` | `RangeError` | length ≠ 373 | src/protocol/appc.ts:732-734 |
| `` `APPC partner logical-unit parameters need exactly ${APPC_PARTNER_PARAMETERS_LENGTH} bytes; received ${data.byteLength}` `` | `RangeError` | length ≠ 144 | src/protocol/appc.ts:773-775 |
| `` `${functionName(input.functionCode)} is not a setup/control function` `` | `Error` | control encoder given a non-control function | src/protocol/appc.ts:805 |
| `` `APPC control parameter length ${parameters.byteLength} exceeds 65535` `` | `RangeError` | control parameters > 0xffff | src/protocol/appc.ts:810 |
| `"an APPC control record cannot contain two operation-info variants"` | `Error` | both `extendedInfo` and `partnerLogicalUnitInfo` supplied | src/protocol/appc.ts:814 |
| `"partnerLogicalUnitInfo is only valid for F_SET_PARTNER_LU_NAME"` | `Error` | partner info on another function | src/protocol/appc.ts:821 |
| `"F_SET_PARTNER_LU_NAME requires partnerLogicalUnitInfo"` | `Error` | missing partner info | src/protocol/appc.ts:828 |
| `` `finalSapParameters must contain exactly ${APPC_FINAL_SAP_PARAMETER_LENGTH} bytes; received ${parameters.byteLength}` `` | `RangeError` | SAP tail ≠ 8 bytes | src/protocol/appc.ts:852-853, 1118-1120 |
| `"finalSapParameters reserved field must be zero"` | `RangeError` | `readUInt16BE(0) !== 0` in the 8-byte tail | src/protocol/appc.ts:858 |
| `` `finalSapParameters declare ${declaredPacketLength} application bytes; received ${applicationDataLength}` `` | `RangeError` | `readUInt16BE(2)` ≠ application byte count | src/protocol/appc.ts:863-864 |
| `` `async APPC data length must be an integer in 1..${MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH}` `` | `RangeError` | async info encode with `dataLength` outside 1..28000 | src/protocol/appc.ts:915-916 |
| `"APPC data operation info must be a Uint8Array"` | `TypeError` | type guard | src/protocol/appc.ts:933 |
| `` `APPC data operation info must contain exactly 32 bytes; received ${data.byteLength}` `` | `RangeError` | length ≠ 32 | src/protocol/appc.ts:937 |
| `"APPC data operation reserved word must be zero"` | `RangeError` | u16 at offset 0 ≠ 0 | src/protocol/appc.ts:945 |
| `"APPC data operation reserved bytes must be zero"` | `RangeError` | any of the 24 bytes at offset 4 ≠ 0 | src/protocol/appc.ts:950 |
| `"incoming APPC data operation info must be a Uint8Array"` | `TypeError` | type guard | src/protocol/appc.ts:970 |
| `` `incoming APPC data operation info must contain exactly 32 bytes; received ${data.byteLength}` `` | `RangeError` | length ≠ 32 | src/protocol/appc.ts:973-974 |
| `` `APPC async-send data length must be in 1..${MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH}` `` | `RangeError` | decoded async `dataLength` out of range | src/protocol/appc.ts:994-995 |
| `"outgoing APPC applicationData must be a Uint8Array"` | `TypeError` | planner type guard | src/protocol/appc.ts:1065 |
| `"outgoing APPC finalSapParameters must be a Uint8Array when present"` | `TypeError` | planner type guard | src/protocol/appc.ts:1071-1073 |
| `` `maxApplicationDataLength must be an integer in 0..${MAX_APPC_OUTGOING_MESSAGE_LENGTH}` `` | `RangeError` | option out of range | src/protocol/appc.ts:1088-1089 |
| `"maxFragments must be a positive safe integer"` | `RangeError` | option out of range (planner and decoder) | src/protocol/appc.ts:1093, 1818 |
| `"cpicStreaming must be disabled or enabled"` | `RangeError` | unknown policy string | src/protocol/appc.ts:1096 |
| `"outgoing APPC message length is unsafe"` | `RangeError` | non-safe-integer byte length | src/protocol/appc.ts:1103 |
| `` `CPIC application data length ${messageApplicationDataLength} exceeds configured limit ${maxApplicationDataLength}` `` | `RangeError` | over configured limit | src/protocol/appc.ts:1107-1108 |
| `"compact CPIC application data cannot exceed 28000 bytes"` | `RangeError` | compact packet (SAP tail present) longer than 28000 | src/protocol/appc.ts:1137 |
| `"a streamed CPIC packet without final SAP parameters must exceed 28000 bytes"` | `RangeError` | no SAP tail and ≤28000 bytes | src/protocol/appc.ts:1145 |
| `"CPIC streaming is disabled; enable this destination before sending more than 28000 application bytes"` | `Error` | multi-record plan while `cpicStreaming !== "enabled"` | src/protocol/appc.ts:1154 |
| `` `APPC fragment count ${fragmentCount} exceeds configured limit ${maxFragments}` `` | `RangeError` | planner cap; the decoder throws the same text with its own numbers | src/protocol/appc.ts:1168, 2076 |
| `"conversationId must be a Uint8Array"` | `TypeError` | planner guard | src/protocol/appc.ts:1174 |
| `` `conversationId must contain exactly 8 bytes; received ${...}` `` | `RangeError` | planner and `encodeAppcRecord` | src/protocol/appc.ts:1181, 1489 |
| `` `invalid outgoing APPC fragment: ${reason}` `` | `RangeError` | every fragment-validation failure; reasons: `"conversationId must be a Uint8Array"`, `"conversationId must contain exactly 8 bytes"`, `"applicationData must be a Uint8Array"`, `"applicationData exceeds the 28000-byte slice bound"`, `"finalSapParameters must be a Uint8Array"`, `"finalSapParameters exceeds 8 bytes"`, `"fragmentCount must be a positive safe integer"`, `"fragmentIndex must identify a fragment in the plan"`, `"messageApplicationDataLength is outside the proven range"`, `"function, final marker, info, vector, or parameter length is inconsistent"`, `"compact F_SAP_SEND data length is invalid"`, `` `${functionName(...)} slice length is invalid` ``, `` `${functionName(...)} cannot carry SAP parameters` ``, `"the async F_RECEIVE terminator must be empty"` | src/protocol/appc.ts:1249-1251, 1259-1421 |
| `"sapParameterLength must be an integer in 0..65535"` | `RangeError` | `encodeAppcRecord` guard | src/protocol/appc.ts:1494 |
| `` `trailingData[${index}] must be a Uint8Array` `` | `TypeError` | `encodeAppcRecord` guard | src/protocol/appc.ts:1502 |
| `"trailingData length exceeds the safe integer range"` | `RangeError` | `encodeAppcRecord` guard | src/protocol/appc.ts:1506 |
| `"cannot complete an APPC response unless one is pending"` | `Error` | `responseComplete()` outside `"response-pending"`; also sets state to `"closed"` | src/protocol/appc.ts:1551-1552 |
| `"F_SAP_SEND cannot start a streamed outgoing message"` | `Error` | `sent(SapSend, false)` | src/protocol/appc.ts:1559 |
| `"F_ASEND_DATA must be followed by F_RECEIVE"` | `Error` | `sent(AsyncSendData, true)` | src/protocol/appc.ts:1562 |
| `"streaming F_SEND_DATA must be followed by its acknowledgement"` | `Error` | `sent(SendData, true)` | src/protocol/appc.ts:1565 |
| `"the streamed outgoing F_RECEIVE terminator must be final"` | `Error` | `sent(Receive, false)` in `"send-continuation"` | src/protocol/appc.ts:1572 |
| `` `cannot send ${functionName(functionCode)} while APPC client is ${this.#state}` `` | `Error` | transition not in the allowed table | src/protocol/appc.ts:1591 |
| `"APPC reply must be a Uint8Array"` | `TypeError` | `received()` guard; sets `"closed"` | src/protocol/appc.ts:1620 |
| `` `an APPC reply needs ${APPC_RECORD_HEADER_LENGTH} bytes; received ${payload.byteLength}` `` | `RangeError` | reply < 80 bytes; sets `"closed"` | src/protocol/appc.ts:1625-1626 |
| `` `cannot receive ${header.functionName} while APPC client is ${this.#state}` `` | `Error` | receive transition not allowed; sets `"closed"` | src/protocol/appc.ts:1671 |
| `` `an APPC common header needs ${APPC_COMMON_HEADER_LENGTH} bytes; received ${payload.byteLength}` `` | `RangeError` | header decode < 48 bytes | src/protocol/appc.ts:1688 |
| `"APPC synchronous-send acknowledgement must be a Uint8Array"` | `TypeError` | guard | src/protocol/appc.ts:1728 |
| `` `APPC synchronous-send acknowledgement must contain exactly ${APPC_RECORD_HEADER_LENGTH} bytes; received ${payload.byteLength}` `` | `RangeError` | length ≠ 80 | src/protocol/appc.ts:1732-1733 |
| `"APPC synchronous-send acknowledgement header is not canonical"` | `Error` | any of the 19 pinned header conditions fails | src/protocol/appc.ts:1759 |
| `"APPC synchronous-send acknowledgement operation information is not canonical"` | `Error` | first 30 info bytes non-zero, or `dataLength`/`communicationIndex` non-zero | src/protocol/appc.ts:1771-1773 |
| `` `an APPC data record needs ${APPC_RECORD_HEADER_LENGTH} bytes; received ${payload.byteLength}` `` | `RangeError` | data-fragment decode < 80 bytes | src/protocol/appc.ts:1784-1785 |
| `` `${header.functionName} is not an APPC RFC data fragment` `` | `Error` | function is neither `SapSend` nor `Receive` | src/protocol/appc.ts:1790 |
| `"maxMessageLength must be a non-negative safe integer"` | `RangeError` | decoder option guard | src/protocol/appc.ts:1815 |
| `"allowInitialReceive must be a boolean"` / `"validateIncomingDataOperationInfo must be a boolean"` | `TypeError` | decoder option guards | src/protocol/appc.ts:1821, 1824 |
| `"terminal APPC deallocation requires APPC return code 18 and SAP return code 0"` | `Error` | `pushTerminalDeallocation` on a non-18 record | src/protocol/appc.ts:1857-1859 |
| `"normal deallocation must use the terminal decoder"` | `Error` | `push()` on a return-code-18 record | src/protocol/appc.ts:1865-1867 |
| `` `${header.functionName} cannot be decoded after APPC return code ${header.appcReturnCode} and SAP return code ${header.sapReturnCode}` `` | `Error` | any other non-zero return code | src/protocol/appc.ts:1870-1871 |
| `` `${header.functionName} interrupted a fragmented message before its final APPC record` `` | `Error` | a control record arrives mid-message | src/protocol/appc.ts:1880 |
| `` `incoming APPC data length ${operationInfo.dataLength} does not match record payload length ${fragment.data.byteLength}` `` | `Error` | only when `validateIncomingDataOperationInfo` is enabled | src/protocol/appc.ts:1912-1913 |
| `"received terminal F_RECEIVE without a preceding F_SAP_SEND"` | `Error` | terminal `F_RECEIVE`, no pending message, `allowInitialReceive` false | src/protocol/appc.ts:1927 |
| `"normal deallocation started a new F_SAP_SEND during a fragmented message"` | `Error` | terminal `F_SAP_SEND` while pending | src/protocol/appc.ts:1942 |
| `"APPC conversation ID changed at normal deallocation"` / `"APPC sequence number changed at normal deallocation"` / `"APPC connection indices changed at normal deallocation"` | `Error` | continuity checks on the terminal record | src/protocol/appc.ts:1947, 1951, 1958 |
| `"received a new F_SAP_SEND during an unfinished fragmented message"` | `Error` | ordinary path | src/protocol/appc.ts:1977 |
| `"received F_RECEIVE without a preceding fragmented F_SAP_SEND"` | `Error` | ordinary path, `allowInitialReceive` false | src/protocol/appc.ts:2007 |
| `"APPC conversation ID changed within a fragmented message"` / `"APPC sequence number changed within a fragmented message"` / `"APPC connection indices changed within a fragmented message"` | `Error` | ordinary continuity checks | src/protocol/appc.ts:2034, 2037, 2043 |
| `` `APPC message length ${byteLength} exceeds configured limit ${this.#maxMessageLength}` `` | `RangeError` | `#checkLimits` | src/protocol/appc.ts:2071 |
| `` `truncated APPC message: ${...} fragment(s), ${...} bytes buffered` `` | `Error` | `finish()` with a pending message | src/protocol/appc.ts:2084-2085 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `"All controlled version-6 records contain 32 bytes after the common header."` | src/protocol/appc.ts:10 |
| `"CPIC streaming has no proven peer-acceptance bit on the observed wire. \`enabled\` therefore means the caller has independently approved the target."` | src/protocol/appc.ts:34-37 |
| `"Captures distinguish an absent name (all NUL) from a present short name (ASCII plus spaces). Preserve that wire distinction."` | src/protocol/appc.ts:380-381 |
| `"Encode the observed fixed 32-byte CPIC extended-connection structure."` | src/protocol/appc.ts:451 |
| `"Unlike client F_ASEND_DATA, the reply's actual byte count is stored at offset 10. The word at offset 2 is a receive-buffer capacity and must not be used to frame the application payload."` | src/protocol/appc.ts:962-964 |
| `"Plan one logical outgoing CPIC message. Compact messages use one F_SAP_SEND. Larger messages use bounded F_ASEND_DATA slices followed by an empty F_RECEIVE terminator required by the admitted STSEND path."` | src/protocol/appc.ts:1028-1032 |
| `"Caller objects can expose accessors or proxies. Read each property exactly once, then perform every validation and emitted-record decision against this plain normalized snapshot."` | src/protocol/appc.ts:1037-1039 |
| `"Exercise the authoritative record encoder once before returning a plan so every header/index bound fails before the first transport write."` | src/protocol/appc.ts:1189-1190 |
| `"\`applicationData\` and \`conversationId\` are views of planner-owned snapshots. Buffer bytes remain mutable for efficient encoding; callers must treat them as readonly and must not modify or reuse a plan."` | src/protocol/appc.ts:226-228 |
| `"A remote ABAP MESSAGE/runtime failure can terminate CPI-C while the same final data record still carries the RFC error envelope. Publish terminal state first, then let the CPIC layer decode that payload."` | src/protocol/appc.ts:1643-1645 |
| `"Reject an over-budget continuation before decodeAppcDataFragment owns a copy of the peer-controlled application bytes. NI has already admitted the individual record; the APPC aggregate is the tighter resource boundary for a fragmented message."` | src/protocol/appc.ts:1894-1897 |
| `"Decode the data returned with CM_DEALLOCATED_NORMAL. SAP Note 63347 establishes return code 18 as a normal terminal conversation outcome; the admitted peer contract uses this status rather than the ordinary APPC vector, terminates the still-valid RFC error envelope."` (sentence as written, including its grammar) | src/protocol/appc.ts:1840-1845 |

### Wire facts asserted by tests

| What the test asserts | Test name (verbatim) | Citation |
|---|---|---|
| `inspectAppcPayload([0x06, 0xcb])` deep-equals `{protocolVersion: 0x06, functionCode: 0xcb, functionName: "F_SAP_SEND"}`; `0x09 → "F_RECEIVE"`; `0x0f → "F_SET_PARTNER_LU_NAME"` | `"recognizes the APPC functions seen in the controlled oracle capture"` | test/appc.test.ts:65-85 |
| partner-LU info is 32 bytes; bytes 0..8 are `"NWRFC   "` (space-padded); `readUInt32BE(8) === 5` (full-name length) | `"encodes and decodes F_SET_PARTNER_LU_NAME operation info"` | test/appc.test.ts:87-109 |
| extended init options are 341 bytes; `encoded[0] === 1`; `encoded.subarray(2,10).toString("hex") === "4350494300000000"` (`"CPIC"` NUL-padded); `readInt32BE(46) === -2` and `readInt32BE(50) === -2` | `"encodes and decodes semantic extended initialization options"` | test/appc.test.ts:111-122 |
| initialize parameters are 373 bytes; first 32 bytes are `"NWRFC"` + 27 spaces | `"encodes the proven 373-byte F_INITIALIZE parameter structure"` | test/appc.test.ts:124-135 |
| partner parameters are 144 bytes; name occupies bytes 0..9, bytes 9..128 are all `0x20` | `"encodes the proven 144-byte F_SET_PARTNER_LU_NAME parameters"` | test/appc.test.ts:137-147 |
| control record with 144-byte parameters is 224 bytes; operation info at bytes 48..56 is `"NWRFC   "`, `readUInt32BE(56) === 5`, `sapParameterLength === 144` | `"places the semantic partner operation info in the APPC control record"` | test/appc.test.ts:177-193 |
| unknown function codes render as `"UNKNOWN_0xaa"` | `"retains unknown function codes for future protocol expansion"` | test/appc.test.ts:232-234 |
| a 48-byte header with hand-placed fields decodes to `uid 0x1234` (offset 4), `gatewayId 0x5678` (6), `timeout -1` (17, int32BE), `sequenceNumber 17` (22), `padding 0x789a` (28), `info 0x7b` (30), `sapReturnCode 6` (36), `conversationId "CONV1234"` (40) | `"decodes the fixed version-6 APPC common header"` | test/appc.test.ts:240-274 |
| a 47-byte input is rejected with `/needs 48 bytes/` | `"rejects a truncated APPC common header"` | test/appc.test.ts:276-278 |
| control record with no names has bytes 48..72 all zero | `"uses the observed NUL-filled extension when control names are absent"` | test/appc.test.ts:383-386 |
| default client data record: 80 + 7 = 87 bytes, `functionCode = SapSend`, `sapParameterLength = 8`, `info = 5`, `vector = 0x0c`, `readUInt16BE(76) === 0xffff` (communicationIndex), `readUInt16BE(78) === 6` (connectionIndex), payload at offset 80 | `"encodes the proven client F_SAP_SEND data-record defaults"` | test/appc.test.ts:353-370 |
| a non-final `F_SAP_SEND` gets `vector === 0x08` and identical application bytes | `"marks non-final F_SAP_SEND fragments without changing application bytes"` | test/appc.test.ts:372-381 |
| state walk `new → initialize-pending → initialized → partner-set → allocate-pending → ready → response-pending → ready → closed` | `"enforces the proven client setup and teardown sequence"` | test/appc.test.ts:411-432 |
| `appcReturnCode 18` + `sapReturnCode 0` on a data reply in `response-pending` returns `"normal-deallocation"` and closes; `{17,0}` and `{18,1}` throw `/failed with APPC return code/` | `"admits only a normal-deallocation data reply for terminal RFC decoding"` | test/appc.test.ts:465-504 |
| `vector 0x0c` ⇒ `isFinal === true`; application data starts at offset 80 | `"decodes the application data after the observed 80-byte APPC record header"` | test/appc.test.ts:506-513 |
| a return-code-18 record must go through `pushTerminalDeallocation`; an empty one raises `CM_NO_DATA_RECEIVED`; an orphan terminal `F_RECEIVE` is refused unless `allowInitialReceive`; `F_SAP_SEND`(0x08) + terminal `F_RECEIVE` concatenates to `"fragment at deallocation"` with `fragmentCount 2` | `"uses normal deallocation as the terminal delimiter for its RFC payload"` | test/appc.test.ts:528-596 |
| `vector 0x08` then `0x00` then `0x0c` reassembles `"first-middle-last"` with `fragmentCount 3` | `"assembles F_SAP_SEND plus multiple F_RECEIVE continuations"` | test/appc.test.ts:598-615 |
| an over-budget continuation fails **before** `subarray(APPC_RECORD_HEADER_LENGTH)` is ever called (asserted via a `Uint8Array` subclass observing `subarray`) | `"enforces message byte and fragment limits before retaining more data"` | test/appc.test.ts:647-677 |

Note (test): the shared `dataRecord` helper writes `sequenceNumber` at offset 22, `vector` at
offset 31, `conversationId` at offset 40, `readUInt16BE(50) = 34_048`, `readUInt16BE(58) =
data.byteLength`, `readUInt16BE(76) = 0`, `readUInt16BE(78) = 6` (test/appc.test.ts:44-63). Offset
58 is inside the 32-byte operation-info block and is `48 + 10`, matching
`decodeAppcIncomingDataOperationInfo`'s `readUInt16BE(10)` (src/protocol/appc.ts:979). The value
`34_048` at offset 50 (= info-block offset 2) is not explained by any comment; the source only says
offset 2 `"is a receive-buffer capacity and must not be used to frame the application payload"`
(src/protocol/appc.ts:963-964).

### Go mapping notes

- **`Buffer` vs `Uint8Array`.** Every encoder returns `Buffer`; decoders accept `Uint8Array`. In Go
  all of this is `[]byte`. The distinction that *does* survive: `snapshotUint8Array` copies,
  `intrinsicUint8ArrayView` borrows (src/protocol/appc.ts:1-6). The planner explicitly documents
  that fragment `applicationData` **borrows** planner-owned memory
  (src/protocol/appc.ts:226-228) — in Go this must be either an explicit sub-slice contract with the
  same "do not reuse a plan" rule, or a copy.
- **Typed-array geometry guards.** `intrinsicUint8ArrayByteLength` / `intrinsicUint8ArrayView` exist
  because a JS caller can subclass `Uint8Array` and override `byteLength`/`subarray`; the appc test
  at test/appc.test.ts:651-668 exercises exactly that. Go slices cannot lie about their length, so
  **all of these guards vanish** and the corresponding `TypeError`s (`"…must be a Uint8Array"`,
  src/protocol/appc.ts:433, 933, 970, 1065, 1174, 1502, 1620, 1728) become dead code, not ported.
- **`#private` fields.** `AppcClientSetupStateMachine.#state` and `AppcConversationDecoder`'s five
  `#` fields (src/protocol/appc.ts:1543, 1801-1805) become unexported struct fields.
- **Sentinel errors.** `AppcPeerReturnCodeError` (carries `appcReturnCode`, `sapReturnCode`,
  src/protocol/appc.ts:270-287) and `AppcNormalDeallocationWithoutDataError`
  (src/protocol/appc.ts:290-295) are the two the caller branches on — they need Go error types with
  `errors.As` support. `AppcNormalDeallocationWithoutDataError` has a fixed message and no fields, so
  it is a package-level sentinel `var`. Everything else is `RangeError`/`Error` with a formatted
  message and is a plain `fmt.Errorf`.
- **`AppcReceiveDisposition`** is a two-valued string union returned by `received()`
  (src/protocol/appc.ts:265-267, 1647, 1680); in Go a small named string/int type, not a bool — the
  distinction between `"accepted"` and `"normal-deallocation"` drives whether the caller may still
  decode a payload (src/protocol/appc.ts:1643-1645).
- **Shape that would be wrong in Go:** `AppcOutgoingDataFragment extends AppcRecordHeaderInput`
  where every header field is optional (`number | undefined`) and the encoder applies `?? default`
  at write time (src/protocol/appc.ts:1515-1532). Go zero values are not the same as these defaults
  — `protocol` defaults to `2` and `uid` to `0xffff`, both non-zero. A naive struct port silently
  emits `protocol=0, uid=0`. Either use pointer fields or a constructor that stamps the defaults.
- **`Object.freeze`** on returned records (src/protocol/appc.ts:955, 978, 1231, 1246, 1298, 1775) has
  no Go analogue; return values, not pointers.
- No `Promise`/`async`/`AbortSignal` anywhere in appc.ts — this file is fully synchronous.

---

## src/protocol/cpic.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `DEFAULT_MAX_CPIC_FIELD_LENGTH` | const | `export const DEFAULT_MAX_CPIC_FIELD_LENGTH = 256 * 1024 * 1024;` | src/protocol/cpic.ts:21 |
| `DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH` | const | `export const DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH = 256 * 1024 * 1024;` | src/protocol/cpic.ts:22 |
| `DEFAULT_MAX_CPIC_FIELD_COUNT` | const | `export const DEFAULT_MAX_CPIC_FIELD_COUNT = 100_000;` | src/protocol/cpic.ts:23 |
| `CLASSIC_XRFC_XML_CHUNK_LENGTH` | const | `export const CLASSIC_XRFC_XML_CHUNK_LENGTH = 16 * 1024;` | src/protocol/cpic.ts:25 |
| `CpicTag` | enum | see the tag table below | src/protocol/cpic.ts:28-86 |
| `CpicField` | interface | `readonly tag: number;` `readonly value: Uint8Array;` | src/protocol/cpic.ts:91-94 |
| `CpicRequestAppcFraming` | interface | `mode: "compact" \| "streamed"; applicationDataLength: number; finalSapParameterLength: 0 \| 8` | src/protocol/cpic.ts:96-101 |
| `CpicFieldChainLimits` | interface | `maxFieldLength?: number; maxChainLength?: number; maxFieldCount?: number` | src/protocol/cpic.ts:103-107 |
| `DecodedCpicFieldChainPrefix` | interface | `readonly fields: DecodedCpicField[];` `readonly bytesConsumed: number;` | src/protocol/cpic.ts:109-112 |
| `CpicInitialLogonRequestInput` | interface | `client, user, password, language, clientAddress: string; partnerSystem?: string; partnerHostName, destination, programName: string; functionName?, kernelRelease?: string; sessionId?: Uint8Array; passwordSeed?, maximumRfcPacketSize?: number` | src/protocol/cpic.ts:119-134 |
| `DecodedCpicInitialLogonRequest` | interface | `fields: ReadonlyArray<{tag: number; byteLength: number}>; cpicPacketSize: number; maximumRfcPacketSize: number` | src/protocol/cpic.ts:136-143 |
| `DecodedCpicInitialLogonRejection` | interface | `outcome: RfcErrorEnvelope["outcome"]; messageClass, messageType, messageNumber, exceptionKey, runtimeId, text: string` | src/protocol/cpic.ts:156-164 |
| `DecodedCpicInitialLogonResponse` | interface | `success: boolean; status?: number; rejection?: DecodedCpicInitialLogonRejection; negotiatedProtocolVersion: number; fields: ReadonlyArray<{tag, byteLength}>` | src/protocol/cpic.ts:166-177 |
| `CpicFunctionRequestInput` | interface | `functionName: string; sessionId: Uint8Array; kernelRelease?: string; maximumRfcPacketSize?: number` | src/protocol/cpic.ts:325-330 |
| `CpicCutFunctionRequestInput` | interface | `functionName: string; requestedOutputs?: readonly string[]; imports?: ReadonlyArray<{name: string; value: Uint8Array}>; tables?: ReadonlyArray<{name: string; rowByteLength: number; rows: readonly Uint8Array[]}>; xrfcParameters?: ReadonlyArray<{name: string; value: Uint8Array}>; kernelRelease?: string; maximumRfcPacketSize?: number` | src/protocol/cpic.ts:332-351 |
| `DecodedCpicFunctionResponse` | interface | `success: boolean; outcome: RfcErrorEnvelopeOutcome; status: number \| undefined; exceptionKey?: string; fields: ReadonlyArray<{tag, byteLength}>` | src/protocol/cpic.ts:353-362 |
| `DecodedCpicFunctionResultFields` | interface | `success: boolean; status: number \| undefined; envelope: RfcErrorEnvelope; fields: ReadonlyArray<{tag: number; value: Buffer}>` | src/protocol/cpic.ts:364-372 |
| `encodeCpicFieldChain` | func | `export function encodeCpicFieldChain(initialPreviousTag: number, fields: readonly CpicField[], limits: CpicFieldChainLimits = {},): Buffer` | src/protocol/cpic.ts:426-430 |
| `decodeCpicFieldChain` | func | `export function decodeCpicFieldChain(data: Uint8Array, initialPreviousTag: number, limits: CpicFieldChainLimits = {},): DecodedCpicField[]` | src/protocol/cpic.ts:490-494 |
| `decodeCpicFieldChainPrefix` | func | `export function decodeCpicFieldChainPrefix(data: Uint8Array, initialPreviousTag: number, terminalTag: number, limits: CpicFieldChainLimits = {},): DecodedCpicFieldChainPrefix` | src/protocol/cpic.ts:511-516 |
| `encodeCpicInitialLogonRequest` | func | `export function encodeCpicInitialLogonRequest(input: CpicInitialLogonRequestInput,): Buffer` | src/protocol/cpic.ts:1016-1018 |
| `decodeCpicInitialLogonRequest` | func | `export function decodeCpicInitialLogonRequest(data: Uint8Array,): DecodedCpicInitialLogonRequest` | src/protocol/cpic.ts:1128-1130 |
| `decodeCpicInitialLogonResponse` | func | `export function decodeCpicInitialLogonResponse(data: Uint8Array,): DecodedCpicInitialLogonResponse` | src/protocol/cpic.ts:1192-1194 |
| `inspectCpicRequestAppcFraming` | func | `export function inspectCpicRequestAppcFraming(data: Uint8Array,): CpicRequestAppcFraming` | src/protocol/cpic.ts:1481-1483 |
| `encodeCpicFunctionRequest` | func | `export function encodeCpicFunctionRequest(input: CpicFunctionRequestInput,): Buffer` | src/protocol/cpic.ts:1533-1535 |
| `encodeCpicCutFunctionRequest` | func | `export function encodeCpicCutFunctionRequest(input: CpicCutFunctionRequestInput,): Buffer` | src/protocol/cpic.ts:1584-1586 |
| `decodeCpicFunctionResultFields` | func | `export function decodeCpicFunctionResultFields(data: Uint8Array,): DecodedCpicFunctionResultFields` | src/protocol/cpic.ts:1794-1796 |
| `decodeCpicResetServerContextResultFields` | func | `export function decodeCpicResetServerContextResultFields(data: Uint8Array,): DecodedCpicFunctionResultFields` | src/protocol/cpic.ts:1811-1813 |
| `decodeCpicSessionRefreshResultFields` | func | `export function decodeCpicSessionRefreshResultFields(data: Uint8Array,): DecodedCpicFunctionResultFields` | src/protocol/cpic.ts:1861-1863 |
| `decodeCpicFunctionResponse` | func | `export function decodeCpicFunctionResponse(data: Uint8Array,): DecodedCpicFunctionResponse` | src/protocol/cpic.ts:1951-1953 |

**Leak in the type surface:** `DecodedCpicFieldChainPrefix.fields` and the return of
`decodeCpicFieldChain` are typed `DecodedCpicField[]`, but `interface DecodedCpicField` is declared
**without `export`** (src/protocol/cpic.ts:114-117).

### Constants and magic values

`CpicTag` (code, src/protocol/cpic.ts:28-86), doc comment: `"Tags admitted by the bounded
direct-CPIC contract."`

| Member | Value | Citation |
|---|---|---|
| `Destination` | `0x0006` | src/protocol/cpic.ts:29 |
| `ClientAddress` | `0x0007` | src/protocol/cpic.ts:30 |
| `PartnerHost` | `0x0008` | src/protocol/cpic.ts:31 |
| `Kernel` | `0x000b` | src/protocol/cpic.ts:32 |
| `ConnectionType` | `0x0011` | src/protocol/cpic.ts:33 |
| `KernelRelease` | `0x0012` | src/protocol/cpic.ts:34 |
| `KernelPatch` | `0x0013` | src/protocol/cpic.ts:35 |
| `PartnerSystem` | `0x0018` | src/protocol/cpic.ts:36 |
| `SystemCodePage` | `0x0016` | src/protocol/cpic.ts:37 |
| `Start` | `0x0101` | src/protocol/cpic.ts:38 |
| `Function` | `0x0102` | src/protocol/cpic.ts:39 |
| `ProtocolVersion` | `0x0103` | src/protocol/cpic.ts:40 |
| `Capabilities` | `0x0106` | src/protocol/cpic.ts:41 |
| `User` | `0x0111` | src/protocol/cpic.ts:42 |
| `Client` | `0x0114` | src/protocol/cpic.ts:43 |
| `Language` | `0x0115` | src/protocol/cpic.ts:44 |
| `Password` | `0x0117` | src/protocol/cpic.ts:45 |
| `Program` | `0x0130` | src/protocol/cpic.ts:46 |
| `LogonStatus` | `0x0161` | src/protocol/cpic.ts:47 |
| `ParameterName` | `0x0201` | src/protocol/cpic.ts:48 |
| `ParameterValue` | `0x0203` | src/protocol/cpic.ts:49 |
| `RequestedOutput` | `0x0205` | src/protocol/cpic.ts:50 |
| `TableName` | `0x0301` | src/protocol/cpic.ts:51 |
| `TableHeader` | `0x0302` | src/protocol/cpic.ts:52 |
| `TableContent` | `0x0303` | src/protocol/cpic.ts:53 |
| `TableCompr` | `0x0304` | src/protocol/cpic.ts:54 |
| `AbapExceptionKey` … `AbapCallStack` | `0x0401, 0x0402, 0x0403, 0x0404, 0x0411, 0x0412, 0x0413, 0x0414, 0x0415, 0x0416, 0x0417, 0x0418` | src/protocol/cpic.ts:55-66 |
| `Unresolved0420` | `0x0420` | src/protocol/cpic.ts:67 |
| `UseClassExceptions` / `ClassExceptionInfo` / `ClassException` / `ClassExceptionEnd` | `0x0421` / `0x0422` / `0x0423` / `0x0424` | src/protocol/cpic.ts:68-71 |
| `LogonMarker` | `0x0337` | src/protocol/cpic.ts:72 |
| `UnicodeIndicator` | `0x0501` | src/protocol/cpic.ts:73 |
| `ContextEnd` | `0x0502` | src/protocol/cpic.ts:74 |
| `ResponseStart` | `0x0500` | src/protocol/cpic.ts:75 |
| `ResponseContext` | `0x0503` | src/protocol/cpic.ts:76 |
| `CallContext` | `0x0512` | src/protocol/cpic.ts:77 |
| `Session` | `0x0514` | src/protocol/cpic.ts:78 |
| `RfcServerResetDone` | `0x0523` — comment: `"Successful reply marker for SYSTEM_RESET_RFC_SERVER."` | src/protocol/cpic.ts:79-80 |
| `XRfcParameter` | `0x3c02` — comment: `"Empty open/close boundary surrounding one classic xRFC XML parameter."` | src/protocol/cpic.ts:81-82 |
| `XRfcData` | `0x3c05` — comment: `"UTF-8 xRFC XML data chunk inside XRfcParameter boundaries."` | src/protocol/cpic.ts:83-84 |
| `End` | `0xffff` | src/protocol/cpic.ts:85 |

Unexported byte constants (all `Buffer.from(<hex>, "hex")`):

| Name | Value (verbatim) | What the source says it means | Citation |
|---|---|---|---|
| `INITIAL_CPIC_UNRESOLVED_0450` | `0x0450` | comment: `"Unexported six-byte successful-logon control observed on S/4HANA 2023."` | src/protocol/cpic.ts:88-89 |
| `INITIAL_CPIC_SIGNATURE` | `"d9c6c3f0f0f0f0f0f0f0f0f0"` (12 bytes) | purpose not stated in source beyond the name `SIGNATURE`; prepended to the initial logon request and required by its decoder | src/protocol/cpic.ts:654, 1116, 1137 |
| `INITIAL_CPIC_PREFIX` | `"010100080301"` (6 bytes) | initial-logon request prefix | src/protocol/cpic.ts:655, 1117, 1140 |
| `INITIAL_CPIC_RESPONSE_PREFIX` | `"010100080101010504010003"` (12 bytes) | regular initial-logon response prefix; also the session-refresh prefix | src/protocol/cpic.ts:656-659, 1204, 1875 |
| `INITIAL_CPIC_ERROR_RESPONSE_PREFIX` | `"010100080101010101010000"` (12 bytes) | error-class initial-logon response prefix | src/protocol/cpic.ts:660-663, 1205 |
| `CPIC_FUNCTION_REQUEST_PREFIX` | `"010100080301010504010003"` (12 bytes) | first regular function request | src/protocol/cpic.ts:865-868, 1568 |
| `CPIC_FUNCTION_RESPONSE_PREFIX` | `"05000000"` (4 bytes) | regular function response | src/protocol/cpic.ts:869, 1738 |
| `CPIC_CUT_FUNCTION_REQUEST_PREFIX` | `"05020000"` (4 bytes) | established-session CUT request | src/protocol/cpic.ts:870, 1725 |
| `INITIAL_PROTOCOL_VERSION` | `"00000e09"` | value sent in tag `0x0103` | src/protocol/cpic.ts:871, 1052 |
| `INITIAL_CAPABILITIES` | `"04010003000a0200000023"` (11 bytes) | value sent in tag `0x0106` | src/protocol/cpic.ts:872, 1053 |
| `INITIAL_TAG_ORDER` | `Start, ProtocolVersion, Capabilities, LogonMarker, Session, Client, User, Password, Language, UnicodeIndicator, ClientAddress, PartnerSystem, ConnectionType, KernelRelease, KernelPatch, PartnerHost, Destination, Program, ContextEnd, Kernel, Function, End` | exact required order for `decodeCpicInitialLogonRequest` | src/protocol/cpic.ts:873-896, 1148-1157 |
| `MAX_SESSION_REFRESH_PREAMBLE_FIELDS` | `32` | session-refresh preamble field cap | src/protocol/cpic.ts:1851, 1906 |
| `MAX_SESSION_REFRESH_PREAMBLE_BYTES` | `16 * 1024` | session-refresh preamble byte cap | src/protocol/cpic.ts:1852, 1921 |
| `INITIAL_CPIC_MAX_TEXT_COORDINATE_BYTE_LENGTH` | `255` | see the "Invariants" quote — a **bound**, deliberately not a pin | src/protocol/cpic.ts:703-711 |

Chain wire format (code, `encodeCpicFieldChain` src/protocol/cpic.ts:439-448): per field
`u16BE previousTag`, then the RFCPRO tag/length header (4 or 8 bytes, see rfcpro.ts), then the value;
`previousTag` for the first field is the caller-supplied `initialPreviousTag`.

Field-chain byte length (code, src/protocol/cpic.ts:472-475):
`2 + rfcProFieldHeaderByteLength(value.byteLength) + value.byteLength` per field.

Initial-logon request field values (code, src/protocol/cpic.ts:1050-1097): `Start` empty,
`ProtocolVersion = INITIAL_PROTOCOL_VERSION`, `Capabilities = INITIAL_CAPABILITIES`, `LogonMarker`
empty, `Session` = 16 bytes, `Client` ASCII, `User` ASCII 1..40, `Password` = scrambled buffer,
`Language` = `input.language.toUpperCase()`, `UnicodeIndicator = Buffer.of(1)`, `ClientAddress` 1..64,
`PartnerSystem` default `"::1"` 1..64, `ConnectionType = "E"`, `KernelRelease` and `KernelPatch` and
`Kernel` all = `kernelRelease` (default `"754"`), `PartnerHost` 1..120, `Destination` 1..120,
`Program` 1..64, `ContextEnd` empty, `Function` default `"RFCPING"` 1..40, `End` empty.
Default `maximumRfcPacketSize` is `0x8500` (src/protocol/cpic.ts:1031, 1519).

Initial-logon request trailer (code, src/protocol/cpic.ts:1110-1114): 10 bytes —
`u16BE End(0xffff)`, `u16BE 0`, `u16BE cpicPacketSize`, `u32BE maximumRfcPacketSize`, where
`cpicPacketSize = 12 + 6 + chainByteLength + 2` (src/protocol/cpic.ts:1099-1103).

`packetTrailer` (code, src/protocol/cpic.ts:1456-1474): if `cpicPacketSize >
MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH` return the 2-byte `"ffff"` sentinel; else the same 10-byte
trailer as above. `cpicPacketSize = packetPrefixAndChainLength + 2`.

`inspectCpicRequestAppcFraming` decision rule (code, src/protocol/cpic.ts:1489-1515) —
**compact** iff `byteLength >= 10 && readUInt16BE(len-10) === 0xffff && readUInt16BE(len-8) === 0 &&
readUInt16BE(len-6) === len-8 && len-8 <= 28000`; **streamed** iff `byteLength > 28000 && byteLength
>= 6 && readUInt16BE(len-6) === 0xffff && readUInt16BE(len-4) === 0 && readUInt16BE(len-2) === 0xffff`.

CUT table encoding (code, src/protocol/cpic.ts:1661-1682): header is 8 bytes,
`u32BE rowByteLength` then `u32BE rows.length`; every row is emitted with tag `TableCompr` (0x0304)
at full declared width.

xRFC encoding (code, src/protocol/cpic.ts:1698-1712): empty `XRfcParameter` boundary, then
`XRfcData` chunks of at most `CLASSIC_XRFC_XML_CHUNK_LENGTH` bytes, then a second empty
`XRfcParameter` boundary.

The rich initial-RFCPING grammar (code, src/protocol/cpic.ts:720-784) is an ordered coordinate list.
Preamble, in order, with pinned widths: `ProtocolVersion 4`, `Capabilities 11`, `LogonStatus 1
(optional)`, `SystemCodePage 8`, `0x0450 6 (optional)`, `0x0451 20 (optional)`, `0x0452 4 (optional)`,
`0x0453 ≤255 (optional)`, `ClientAddress ≤255`, `0x0020 92 (optional)`, `0x0021 20 (optional)`,
`PartnerSystem ≤255`, `PartnerHost ≤255`, `ConnectionType 2`, `KernelPatch 8`, `KernelRelease 8`,
`Destination ≤255 (optional)`, `Program ≤255`, `0x0150 24`, `0x0151 6`, `0x0152 2`. Embedded response:
`ResponseStart 0`, `ResponseContext 0`, `Session 16`, `Unresolved0420 4`, `CallContext 0`,
`Program ≤255`, `0x0667 8`, `0x0126 4 (optional)`, `End 0`.

`INITIAL_CPIC_REGULAR_RESPONSE_TAGS` (code, src/protocol/cpic.ts:678-687): `Start,
ProtocolVersion, Capabilities, LogonStatus, Unresolved0420, 0x0450, SystemCodePage, End`.

`INITIAL_CPIC_ERROR_PREAMBLE_TAGS` (code, src/protocol/cpic.ts:664-677, exact order):
`ProtocolVersion, Capabilities, SystemCodePage, ClientAddress, PartnerSystem, PartnerHost,
ConnectionType, KernelPatch, KernelRelease, Destination, Program, ResponseStart`.

`SESSION_REFRESH_PREAMBLE_TAGS` (code, src/protocol/cpic.ts:1831-1850, a `Set`, order not enforced):
`ProtocolVersion, Capabilities, LogonStatus, SystemCodePage, ClientAddress, PartnerSystem,
PartnerHost, ConnectionType, KernelRelease, KernelPatch, Destination, Program, 0x0020, 0x0021,
0x0450, 0x0451, 0x0452, 0x0453`.

### Errors

Two internal error-classification mechanisms exist alongside plain `Error`/`RangeError`:

- `class CpicInitialLogonStructureError extends Error` — **not exported**; `rule` and `fields` are
  non-enumerable, non-writable own properties, and membership is tracked in a module-level
  `WeakSet` (src/protocol/cpic.ts:207, 212-253). Doc comment: `"Internal redaction-safe detail
  copied into a hidden public assertion."` (src/protocol/cpic.ts:211).
- Two `Symbol.for` projectors installed as non-enumerable properties **on the
  `decodeCpicInitialLogonResponse` function object**:
  `"open-rfc.internal.initial-cpic-logon-structure-projector/v1"` and
  `"open-rfc.internal.initial-cpic-logon-parse-stage-projector/v1"`
  (src/protocol/cpic.ts:201-206, 274-294). Parse stages are recorded in a `WeakMap` keyed by the
  thrown error object (src/protocol/cpic.ts:208-209, 296-304).

`InitialCpicLogonStructureRule` values (src/protocol/cpic.ts:179-189): `"unsupported-field"`,
`"unsupported-field-zero-logon-status"`, `"invalid-end-field"`, `"invalid-start-field"`,
`"malformed-vendor-logon-control"`, `"duplicate-control-field"`, `"malformed-one-byte-status"`,
`"malformed-call-status"`, `"missing-logon-status"`, `"nonzero-call-status"`.

`InitialCpicLogonParseStage` values (src/protocol/cpic.ts:191-199): `"truncated"`, `"prefix"`,
`"field-chain"`, `"trailer"`, `"protocol"`, `"error-preamble"`, `"error-envelope"`, `"structural"`.

| Message text (verbatim) | Type | Condition | Citation |
|---|---|---|---|
| `` `${field} must be an integer in 0..65535` `` | `RangeError` | `uint16` guard on `initialPreviousTag`, `tag`, `terminalTag` | src/protocol/cpic.ts:376 |
| `` `${field} must be an integer in ${minimum}..${maximum}` `` | `RangeError` | limit-option guard | src/protocol/cpic.ts:388-389 |
| `` `${field} must contain ${minimum}..${maximum} ASCII bytes` `` | `RangeError` | `ascii()` bound check | src/protocol/cpic.ts:406 |
| `` `${field} must contain exactly ${length} bytes; received ${result.byteLength}` `` | `RangeError` | `exactBytes` (used for `sessionId`, 16) | src/protocol/cpic.ts:415-417 |
| `` `CPIC field count ${fields.length} exceeds configured limit ${...maxFieldCount}` `` | `RangeError` | encoder count cap | src/protocol/cpic.ts:461 |
| `` `CPIC field length ${field.value.byteLength} exceeds configured limit ${...maxFieldLength}` `` | `RangeError` | encoder per-field cap | src/protocol/cpic.ts:468 |
| `` `CPIC field chain length exceeds configured limit ${...maxChainLength}` `` | `RangeError` | encoder aggregate cap | src/protocol/cpic.ts:481 |
| `"CPIC field-chain decoder invariant failed"` | `Error` | `decodeCpicFieldChain` consumed ≠ input length | src/protocol/cpic.ts:502 |
| `` `CPIC field chain length ${data.byteLength} exceeds configured limit ${...}` `` | `RangeError` | whole-region decode cap | src/protocol/cpic.ts:539 |
| `` `CPIC field count exceeds configured limit ${...maxFieldCount}` `` | `RangeError` | decoder count cap | src/protocol/cpic.ts:548 |
| `` `CPIC field chain expected previous tag ${...}; received ${...}` `` | `Error` | chained previous-tag mismatch | src/protocol/cpic.ts:558-559 |
| `` `CPIC field chain ended before terminal tag ${tagText(terminalTag)}` `` | `Error` | prefix decode ran out of input | src/protocol/cpic.ts:593 |
| `` `CPIC field chain length ${byteLength} exceeds configured limit ${maximum}` `` | `RangeError` | `enforceCpicChainLength`, applied **before** each read | src/protocol/cpic.ts:604-605 |
| `` `CPIC field chain.${field}: need ${byteLength} bytes at offset ${offset}; ${remaining} remain` `` | `RangeError` | `requireCpicInput`; `field` ∈ `"fieldHeader"`, `"extendedLength"`, `"value"` | src/protocol/cpic.ts:618-619 |
| `"client must contain exactly three ASCII digits"` | `RangeError` | logon `client` | src/protocol/cpic.ts:1020 |
| `"language must contain one ASCII letter"` | `RangeError` | logon `language` | src/protocol/cpic.ts:1023 |
| `"kernelRelease must contain exactly three ASCII digits"` | `RangeError` | all three request encoders | src/protocol/cpic.ts:1028, 1538, 1590 |
| `"maximumRfcPacketSize must be an unsigned 32-bit integer"` | `RangeError` | logon and `checkedMaximumRfcPacketSize` | src/protocol/cpic.ts:1038, 1526 |
| `` `initial CPIC packet size ${cpicPacketSize} exceeds 65535` `` | `RangeError` | logon request too large for its u16 size word | src/protocol/cpic.ts:1106 |
| `"initial CPIC logon request is truncated"` | `RangeError` | shorter than prefix + 10 | src/protocol/cpic.ts:1135 |
| `"initial CPIC logon signature is invalid"` / `"initial CPIC logon prefix is invalid"` | `Error` | signature/prefix mismatch | src/protocol/cpic.ts:1138, 1141 |
| `"initial CPIC logon fields do not match the required tag order"` | `Error` | count or order ≠ `INITIAL_TAG_ORDER` | src/protocol/cpic.ts:1154-1155 |
| `"initial CPIC protocol-version field is unsupported"` / `"initial CPIC capabilities field is unsupported"` | `Error` | field 1 / field 2 byte mismatch | src/protocol/cpic.ts:1159, 1162 |
| `"initial CPIC logon request has an invalid trailer length"` | `Error` | trailer ≠ 10 bytes | src/protocol/cpic.ts:1166 |
| `"initial CPIC logon trailer marker is invalid"` | `Error` | trailer `u16BE(0) !== 0xffff` or `u16BE(2) !== 0` | src/protocol/cpic.ts:1173 |
| `` `initial CPIC packet size ${cpicPacketSize} does not match derived size ${trailerOffset + 2}` `` | `Error` | size word inconsistent | src/protocol/cpic.ts:1178 |
| `"initial CPIC logon response is truncated"` | `RangeError`, stage `"truncated"` | < prefix + 8 | src/protocol/cpic.ts:1197-1201 |
| `"initial CPIC logon response prefix is invalid"` | stage `"prefix"` | neither regular nor error prefix | src/protocol/cpic.ts:1207-1210 |
| `"initial CPIC logon response trailer is invalid"` | stage `"trailer"` | trailer ≠ 2 bytes or ≠ `0xffff` | src/protocol/cpic.ts:1228-1231 |
| `"initial CPIC logon response lacks its protocol version"` | stage `"protocol"` | missing, wrong width, or duplicated | src/protocol/cpic.ts:1238-1241, 1301-1304 |
| `"initial CPIC logon error response has an invalid preamble"` / `"…has duplicate preamble fields"` | stage `"error-preamble"` | error-prefix path preamble checks | src/protocol/cpic.ts:1256-1259, 1263-1266 |
| `"initial CPIC logon error response lacks a rejected outcome"` | stage `"error-envelope"` | envelope classified `"success"` | src/protocol/cpic.ts:1278-1281 |
| `` `initial CPIC logon response contains unsupported field ${tagText(field.tag)} (${field.value.byteLength} bytes) at index ${index}` `` | structural, rule `"unsupported-field"` | tag outside the regular set | src/protocol/cpic.ts:1315-1319 |
| `"initial CPIC logon response has an invalid End field"` | rule `"invalid-end-field"` | not exactly one, last, zero-length | src/protocol/cpic.ts:1328-1332 |
| `"initial CPIC logon response has an invalid Start field"` | rule `"invalid-start-field"` | >1, or not first, or non-empty | src/protocol/cpic.ts:1343-1347 |
| `"initial CPIC logon response has malformed 0x0450 control"` | rule `"malformed-vendor-logon-control"` | >1, or width ≠ 6, or not at semantic index 4 preceded by `Unresolved0420` | src/protocol/cpic.ts:1365-1369 |
| `"initial CPIC logon response has duplicate control fields"` | rule `"duplicate-control-field"` | `Capabilities` or `SystemCodePage` repeated | src/protocol/cpic.ts:1373-1377 |
| `"initial CPIC logon response has malformed one-byte status"` | rule `"malformed-one-byte-status"` | >1 `LogonStatus`, or width ≠ 1 | src/protocol/cpic.ts:1388-1392 |
| `"initial CPIC logon response has malformed call status"` | rule `"malformed-call-status"` | >1 `Unresolved0420`, or width ≠ 4 | src/protocol/cpic.ts:1401-1405 |
| `"initial CPIC logon response lacks a recognized logon status"` | rule `"missing-logon-status"` | neither status form present | src/protocol/cpic.ts:1408-1412 |
| `"initial CPIC logon response has nonzero call status"` | rule `"nonzero-call-status"` | `readUInt32BE(0) !== 0` | src/protocol/cpic.ts:1418-1422 |
| `"initial CPIC RFCPING composite response has an invalid End field"` / `"…has malformed logon status"` / `"…has malformed call status"` / `"…has a duplicate field"` / `"…does not match the bounded composite shape"` / `"…has nonzero call status"` / `"…is not successful"` | rules `invalid-end-field`, `malformed-one-byte-status`, `malformed-call-status`, `duplicate-control-field`, `unsupported-field`/`unsupported-field-zero-logon-status`, `nonzero-call-status` | composite-response path | src/protocol/cpic.ts:913-917, 927-931, 940-944, 947-951, 966-972, 986-990, 998-1002 |
| `` `${field} must contain 1..${maximumCharacters} Unicode scalar characters without NUL` `` | `RangeError` | `unicode()` — rejects empty, over-long, embedded NUL, or any lone/paired surrogate code unit | src/protocol/cpic.ts:1449-1451 |
| `"CPIC request data must be a Uint8Array"` | `TypeError` | framing inspector guard | src/protocol/cpic.ts:1485 |
| `"CPIC request has an invalid APPC framing trailer"` | `Error` | neither compact nor streamed shape | src/protocol/cpic.ts:1515 |
| `` `duplicate ${kind} ${value}` `` | `Error` | kinds `"requested output"`, `"import"`, `"table"`, `"xRFC parameter"` | src/protocol/cpic.ts:1574, 1600-1612 |
| `` `duplicate input parameter ${parameter.name}` `` | `Error` | same name across imports/tables/xrfc | src/protocol/cpic.ts:1617 |
| `` `${table.name} rowByteLength must be an unsigned 32-bit integer` `` | `RangeError` | table header guard | src/protocol/cpic.ts:1653 |
| `` `${table.name} row count exceeds the unsigned 32-bit range` `` | `RangeError` | table header guard | src/protocol/cpic.ts:1657-1659 |
| `` `${table.name} row ${index} contains ${row.byteLength} bytes; expected ${table.rowByteLength}` `` | `RangeError` | row width mismatch | src/protocol/cpic.ts:1674-1675 |
| `` `${parameter.name} xRFC XML value must be Uint8Array bytes` `` | `TypeError` | xRFC guard | src/protocol/cpic.ts:1689 |
| `` `${parameter.name} xRFC XML value must not be empty` `` | `RangeError` | xRFC guard | src/protocol/cpic.ts:1696 |
| `"CPIC function response is truncated"` | `RangeError` | < prefix + 8 | src/protocol/cpic.ts:1733 |
| `"CPIC function response prefix is invalid"` | `Error` | ≠ `"05000000"` | src/protocol/cpic.ts:1739 |
| `"CPIC function response trailer is invalid"` | `Error` | trailer ≠ 2 bytes, or ≠ `0xffff` | src/protocol/cpic.ts:1749, 1753 |
| `"SYSTEM_RESET_RFC_SERVER response reset-done control must be empty and unique"` | `Error` | >1 `0x0523`, or non-empty | src/protocol/cpic.ts:1824-1826 |
| `"CPIC session-refresh response is truncated"` | `RangeError` | < prefix + 8 | src/protocol/cpic.ts:1870 |
| `"CPIC session-refresh response prefix is invalid"` / `"…trailer is invalid"` | `Error` | prefix/trailer checks | src/protocol/cpic.ts:1877, 1890 |
| `"CPIC session-refresh response must contain one empty embedded response marker"` | `Error` | not exactly one empty `ResponseStart` | src/protocol/cpic.ts:1899-1901 |
| `"CPIC session-refresh preamble field count is invalid"` | `Error` | `responseStartIndex < 2` or `> 32` | src/protocol/cpic.ts:1908 |
| `"CPIC session-refresh preamble contains an unknown or duplicate field"` | `Error` | tag outside `SESSION_REFRESH_PREAMBLE_TAGS`, or repeated | src/protocol/cpic.ts:1915-1917 |
| `"CPIC session-refresh preamble exceeds its byte limit"` | `RangeError` | > 16 KiB of preamble values | src/protocol/cpic.ts:1922 |
| `"CPIC session-refresh preamble lacks its protocol and Unicode headers"` | `Error` | fields 0/1 not `ProtocolVersion(4)` / `Capabilities(11)` | src/protocol/cpic.ts:1933-1935 |
| `"CPIC session-refresh preamble has a nonzero status"` | `Error` | `LogonStatus` present with width ≠ 1 or value ≠ 0 | src/protocol/cpic.ts:1944 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `"Encode CPIC's chained field grammar. Each record names both the previous and current tag, allowing a decoder to detect dropped or reordered fields."` | src/protocol/cpic.ts:422-425 |
| `"Upper bound for a text coordinate. SAP pads these to fixed internal widths, so the exact value is a property of the endpoint's own names, not of the wire format. Bounding rather than pinning is the whole point of this grammar: an earlier revision enumerated whole response graphs with every width fixed, which made a two-character difference in a host name indistinguishable from a malformed response, and reported successful logons as RFC_INVALID_PROTOCOL."` | src/protocol/cpic.ts:703-710 |
| `"The logon/session preamble. Order is exact and every tag must be listed; only the marked coordinates may be absent, and only the \`maxByteLength\` coordinates may vary in width. Unknown tags, reordering, duplication, a missing required coordinate and a control coordinate of the wrong width all still fail closed."` | src/protocol/cpic.ts:713-719 |
| `"\`byteLength\` pins a control coordinate to its exact width. \`maxByteLength\` bounds a coordinate that carries a name or address and therefore varies with the endpoint rather than with the protocol. A coordinate must declare exactly one of the two."` | src/protocol/cpic.ts:690-694 |
| `"The embedded ordinary RFCPING response, opened by the sole empty \`ResponseStart\` that partitions the composite. The four-byte call status stays mandatory and exact: it is what decides success."` | src/protocol/cpic.ts:765-769 |
| `"Walk the field chain against the grammar in order. Tags disambiguate every choice, so the walk is deterministic and needs no backtracking… Trailing unmatched fields fail."` | src/protocol/cpic.ts:821-826 |
| `"Still fails closed either way; only the rule differs. A zero logon status means the server did not reject the credential, so this is a decoder gap and not an authentication failure. Reading the resulting RFC_INVALID_PROTOCOL as a rejected password is the misdiagnosis this distinction exists to prevent."` | src/protocol/cpic.ts:961-965 |
| `"A nonzero one-byte logon status is a valid SAP rejection, not malformed protocol. Preserve that status and do not interpret an embedded call success control after authentication itself failed."` | src/protocol/cpic.ts:974-976 |
| `"S/4HANA 2023 emits the authoritative one-byte logon status together with a zero 0x0420 control. NetWeaver 7.50 is also observed emitting only that zero control for successful logon."` | src/protocol/cpic.ts:1424-1426 |
| `"An error-class initial response carries the server's own reason. An earlier revision decoded that envelope purely to confirm the outcome was not \`success\` and then discarded it, so a rejection reached the caller with no reason at all and every rejection looked alike."` | src/protocol/cpic.ts:147-151 |
| `"The identity fields are an SAP message coordinate (class / type / number), not free text; \`text\` is the backend's own message and callers that persist evidence must treat it as backend text and omit it."` | src/protocol/cpic.ts:151-155 |
| `"The admitted contract switches to streamed F_ASEND_DATA records above the 28,000-byte STSEND application boundary. In that mode the logical CUT message closes with the field chain's existing empty End field followed only by the 0xffff packet sentinel; the compact SAP8 words are omitted."` | src/protocol/cpic.ts:1462-1466 |
| `"Compact calls carry eight final SAP-parameter bytes; streamed calls end in the admitted \`0xffff\` packet-size sentinel and carry no separate maximum-size word."` | src/protocol/cpic.ts:1477-1480 |
| `"A full-width payload is a valid (non-shortened) simple-compression record. Keep this established compatible encoding until an encoder-side compression policy is introduced deliberately."` | src/protocol/cpic.ts:1678-1680 |
| `"S/4HANA 2023 emits one zero-length RFCID.RfcServerResetDone (0x0523); NetWeaver 7.50 returns the otherwise identical zero-status success envelope without it. Keep the marker optional-but-singleton here and fatal in every other response state."` | src/protocol/cpic.ts:1806-1809 |
| `"SAP prepends a bounded session-header refresh to an embedded regular response. Only this state accepts that initial prefix; ordinary calls remain strict regular envelopes."` | src/protocol/cpic.ts:1857-1859 |
| `"Reuse an existing internal-module export as the host for a loader-local, declaration-invisible projector. The public error adapter imports this exact function object, so the WeakSet provenance remains scoped to the same module instance without widening the package's declaration surface."` | src/protocol/cpic.ts:270-273 |
| `"Requested outputs and input name/value records stay separate so CHANGING parameters may intentionally occur in both collections."` | src/protocol/cpic.ts:1580-1582 |
| `"Validate the diagnostic name with the same admitted RFC-name bounds even though the XML root is the on-wire discriminator."` | src/protocol/cpic.ts:1685-1686 |

### Wire facts asserted by tests

| What the test asserts | Test name (verbatim) | Citation |
|---|---|---|
| `encodeCpicFieldChain(0x0514, [{0x0114,"001"},{0x0111,"USER"},{0xffff,""}])` produces exactly `"051401140003303031011401110004555345520111ffff0000"` and round-trips; the decoded values survive `encoded.fill(0)` (i.e. the decoder copies) | `"encodes and decodes the chained CPIC tag grammar"` | test/cpic.test.ts:24-43 |
| a prefix decode stops at `End` and leaves the 10-byte trailer `"ffff0000012000008500"` untouched; `maxChainLength` exactly equal to the chain passes, one less throws | `"decodes a CPIC field prefix without consuming its protocol trailer"` | test/cpic.test.ts:45-82 |
| a chain lacking the terminal tag throws `/ended before terminal tag 0xffff/` | `"requires the requested CPIC terminal tag"` | test/cpic.test.ts:84-92 |
| a corrupted previous-tag word throws `/expected previous tag 0x0114.*received 0x0001/`; a truncated value throws `/need 4 bytes/` | `"rejects a broken CPIC tag chain and truncated values"` | test/cpic.test.ts:94-109 |
| length 65 534 encodes compactly as `"02010203fffe"`; length 65 535 encodes as `"02010203ffff0000ffff"` (sentinel + extended int32); length 65 536 as `"02010203ffff00010000"` | `"encodes compact and extended RFCPRO lengths inside CPIC chains"` | test/cpic.test.ts:111-157 |
| every truncation of `"02010203ffff00010000"` at lengths 1..9 throws `/need [46] bytes/` before the value is read | `"rejects every truncated CPIC extended length before reading its value"` | test/cpic.test.ts:435-445 |
| a CUT request whose import value only *claims* an over-limit `byteLength` fails preflight with zero payload property reads | `"preflights the bounded CUT chain before reading an oversized import"` | test/cpic.test.ts:300-322 |
| a 65 536-byte import produces a request > 0xffff bytes ending in `"ffff0000ffff"`, framed as `{mode:"streamed", applicationDataLength: encoded.byteLength, finalSapParameterLength: 0}` | `"uses the proven streaming CUT trailer above the 28,000-byte boundary"` | test/cpic.test.ts:324-348 |
| a 27 926-byte import yields a **compact** request of exactly 28 000 application bytes + 8; a 27 927-byte import yields a **streamed** request of exactly 28 001 bytes | `"switches generated CPIC requests strictly between 28,000 and 28,001 application bytes"` | test/cpic.test.ts:350-371 |
| `inspectCpicRequestAppcFraming` uses intrinsic geometry: overriding `buffer`/`byteOffset`/`byteLength` on the request does not change its answer | `"inspects CPIC framing with intrinsic typed-array geometry"` | test/cpic.test.ts:388-401 |
| an oversized trailer is rejected **without** the trailer sub-array ever being requested | `"rejects an oversized CPIC response trailer before snapshotting it"` | test/cpic.test.ts:403-433 |
| a full initial logon request is 296 bytes; first 18 bytes are `"d9c6c3f0f0f0f0f0f0f0f0f0010100080301"`; last 10 are `"ffff0000012000008500"`; `cpicPacketSize` 288; `maximumRfcPacketSize` `0x8500`; field tag/length list is `[0x0101,0] [0x0103,4] [0x0106,11] [0x0337,0] [0x0514,16] [0x0114,3] [0x0111,6] [0x0117,29] [0x0115,1] [0x0501,1] [0x0007,9] [0x0018,3] [0x0011,1] [0x0012,3] [0x0013,3] [0x0008,17] [0x0006,9] [0x0130,10] [0x0502,0] [0x000b,3] [0x0102,7] [0xffff,0]`; the decoded field objects have **no** `value` property | `"encodes the capture-sized semantic initial CPIC logon request"` | test/cpic.test.ts:469-520 |
| a logon response with `ProtocolVersion "00000e0b"`, `Capabilities "0401000300030200000023"`, `LogonStatus 0x00`, 8-byte `SystemCodePage`, `End` decodes to `{success:true, status:0, negotiatedProtocolVersion:0x0e0b, …}`; flipping the status byte to `7` gives `{success:false, status:7}` | `"decodes a redaction-safe initial CPIC logon response"` | test/cpic.test.ts:557-594 |
| a response with **only** a zero 4-byte `Unresolved0420` and no `LogonStatus` is `{success:true, status:0}` | `"accepts the observed NetWeaver 7.50 and S/4HANA 2023 logon status forms"` | test/cpic.test.ts:596-797 |
| the first regular function request is 129 bytes, starts `"010100080301010504010003"`, ends `"ffff0000007900008500"`, and carries fields `[0x0103,4] [0x0106,11] [0x0337,0] [0x0514,16] [0x0502,0] [0x000b,6] [0x0102,16] [0x0512,0] [0xffff,0]` with the function name as UTF-16LE `"RFC_PING"` | `"encodes the capture-sized first Unicode RFC_PING request"` | test/cpic.test.ts:1598-1629 |
| the CUT metadata request is 408 bytes, starts `"05020000"`, ends `"ffff0000019000008500"`, with tag/length list `[Kernel,6] [Function,52] [CallContext,0] [RequestedOutput,46] [RequestedOutput,22] [RequestedOutput,22] [RequestedOutput,12] [RequestedOutput,40] [ParameterName,16] [ParameterValue,60] [ParameterName,38] [ParameterValue,2] [End,0]` | `"encodes the capture-verified CUT metadata request semantically"` | test/cpic.test.ts:1658-1703 |
| table header for `rowByteLength 4`, 2 rows is `"0000000400000002"`; both rows are emitted as `TableCompr` with their exact bytes | `"encodes CUT table inputs as full-width simple-compression records"` | test/cpic.test.ts:1727-1771 |
| a success response `"05000000"` + chain(`ResponseContext`, `Session`(16), `Unresolved0420`(4 zero), `CallContext`, `End`) + `"ffff"` decodes to `{success:true, outcome:"success", status:0}` | `"decodes a redaction-safe regular RFC success response"` | test/cpic.test.ts:1631-1656 |

### Go mapping notes

- **`Symbol.for` projectors + `WeakSet`/`WeakMap` provenance** (src/protocol/cpic.ts:201-304) have no
  Go equivalent and no reason to exist in Go: the whole mechanism is a workaround for exporting
  diagnostic detail without widening the TypeScript `.d.ts` surface. In Go this is simply an
  unexported error type with exported-through-`errors.As` accessor methods, or two exported
  `Rule`/`Stage` string types on an unexported struct. **Do not port the symbol plumbing.**
- **Sentinel errors.** The two classification enums (`InitialCpicLogonStructureRule`,
  `InitialCpicLogonParseStage`, src/protocol/cpic.ts:179-199) are the real API here — they should
  become exported Go constants on a single error type. Everything else in this file is an anonymous
  `Error`/`RangeError`; the callers that matter branch on the rule/stage, not on the message.
- **`Proxy`-resistant reads.** `intrinsicUint8ArrayView` (src/protocol/cpic.ts:1487, 1691, 1864) and
  the preflight-before-read ordering exist because a JS caller can hand in a `Proxy` whose
  `byteLength` lies (test/cpic.test.ts:303-321, 415-426). In Go these guards are unnecessary, but the
  **ordering invariant they enforce is still worth preserving**: limits must be checked before any
  allocation sized from peer input (src/protocol/cpic.ts:552, 568, 579).
- **`BigInt`** is not used in cpic.ts, but the sibling `classic-rfc.ts` preflight does use it;
  see that section.
- **`0 | 8` literal union** for `finalSapParameterLength` (src/protocol/cpic.ts:100) — in Go, a plain
  `int` with the two values documented, or a bool `compact`.
- **`status: number | undefined`** in `DecodedCpicFunctionResponse` (src/protocol/cpic.ts:356) is a
  real tri-state: absent means "no zero-control was present". Go needs `*int` or an explicit
  `HasStatus bool`; `0` is a distinct, meaningful value (src/protocol/cpic.ts:1779-1781).
- **Password material.** `encodeCpicInitialLogonRequest` zero-fills `chain` and `password` in a
  `finally` block (src/protocol/cpic.ts:1121-1124). Go has no `finally`; use `defer` with explicit
  zeroing, and be aware `Buffer.concat` at src/protocol/cpic.ts:1115-1120 already copied the
  password into the returned buffer, which is **not** zeroed.
- No `Promise`/`async`/`AbortSignal` in cpic.ts.

---

## src/protocol/rfcpro.ts

The tag/length header primitive shared by every CPIC field.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RFC_PRO_EXTENDED_LENGTH_SENTINEL` | const | `export const RFC_PRO_EXTENDED_LENGTH_SENTINEL = 0xffff;` | src/protocol/rfcpro.ts:3 |
| `RFC_PRO_COMPACT_LENGTH_MAX` | const | `export const RFC_PRO_COMPACT_LENGTH_MAX = RFC_PRO_EXTENDED_LENGTH_SENTINEL - 1;` | src/protocol/rfcpro.ts:4 |
| `RFC_PRO_VALUE_LENGTH_MAX` | const | `export const RFC_PRO_VALUE_LENGTH_MAX = 0x7fff_ffff;` | src/protocol/rfcpro.ts:5 |
| `RfcProLengthEncoding` | type | `export type RfcProLengthEncoding = "compact" \| "extended";` | src/protocol/rfcpro.ts:7 |
| `RfcProFieldHeader` | interface | `readonly tag: number;` `readonly length: number;` `readonly encoding: RfcProLengthEncoding;` `readonly bytesConsumed: 4 \| 8;` | src/protocol/rfcpro.ts:9-14 |
| `RfcProFieldHeaderDecodeOptions` | interface | `readonly maxValueLength?: number;` | src/protocol/rfcpro.ts:16-18 |
| `rfcProFieldHeaderByteLength` | func | `export function rfcProFieldHeaderByteLength(length: number): 4 \| 8` | src/protocol/rfcpro.ts:41 |
| `encodeRfcProFieldHeader` | func | `export function encodeRfcProFieldHeader(tag: number, length: number): Buffer` | src/protocol/rfcpro.ts:47 |
| `decodeRfcProFieldHeader` | func | `export function decodeRfcProFieldHeader(data: Uint8Array, options: RfcProFieldHeaderDecodeOptions = {},): RfcProFieldHeader` | src/protocol/rfcpro.ts:68-71 |

### Constants and magic values

| Name | Value (verbatim) | What the source says it means | Citation |
|---|---|---|---|
| `RFC_PRO_EXTENDED_LENGTH_SENTINEL` | `0xffff` | written as `"extendedLengthSentinel"` in place of the compact length word to signal a following `int32BE` | src/protocol/rfcpro.ts:3, 55-59 |
| `RFC_PRO_COMPACT_LENGTH_MAX` | `0xffff - 1` = 65534 | `rfcProFieldHeaderByteLength` returns `4` at or below this, `8` above | src/protocol/rfcpro.ts:4, 43 |
| `RFC_PRO_VALUE_LENGTH_MAX` | `0x7fff_ffff` | the only accepted length ceiling; the extended length is written and read as **signed** `int32BE` | src/protocol/rfcpro.ts:5, 59, 114 |

Wire layout (code, src/protocol/rfcpro.ts:51-60): `u16BE tag`, then either `u16BE length` (compact,
4 bytes total) or `u16BE 0xffff` + `i32BE length` (extended, 8 bytes total).

Decoder tolerance (code, src/protocol/rfcpro.ts:82): it snapshots `data.subarray(0, 8)` — trailing
bytes past the header are ignored, and an extended header may legally encode a length that would
have fit compactly.

### Errors

| Message text (verbatim) | Type | Condition | Citation |
|---|---|---|---|
| `` `${field} must be an integer in ${minimum}..${maximum}` `` | `RangeError` | `field` ∈ `"RFCPRO tag"` (0..65535), `"RFCPRO length"` (0..2147483647), `"maxValueLength"` (0..2147483647) | src/protocol/rfcpro.ts:27-29, 34, 38, 73-78 |
| `` `RFCPRO field header.tag: need 2 bytes at offset 0; ${bytes.byteLength} remain` `` | `RangeError` | fewer than 2 bytes | src/protocol/rfcpro.ts:85 |
| `` `RFCPRO field header.length: need 2 bytes at offset 2; ${bytes.byteLength - 2} remain` `` | `RangeError` | fewer than 4 bytes | src/protocol/rfcpro.ts:91 |
| `` `RFCPRO field header.extendedLength: need 4 bytes at offset 4; ${bytes.byteLength - 4} remain` `` | `RangeError` | sentinel seen but fewer than 8 bytes | src/protocol/rfcpro.ts:111 |
| `` `RFCPRO length ${compactLength} exceeds configured limit ${maxValueLength}` `` | `RangeError` | compact length over limit | src/protocol/rfcpro.ts:98 |
| `` `RFCPRO extended length ${extendedLength} is negative` `` | `RangeError` | `i32BE` read < 0 | src/protocol/rfcpro.ts:117 |
| `` `RFCPRO length ${extendedLength} exceeds configured limit ${maxValueLength}` `` | `RangeError` | extended length over limit | src/protocol/rfcpro.ts:122 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `"Encode the canonical RFCPRO tag/length header without allocating a value."` | src/protocol/rfcpro.ts:46 |
| `"Decode an RFCPRO tag/length header prefix and apply the allocation policy before a caller reads or allocates the advertised value."` | src/protocol/rfcpro.ts:64-67 |
| `"Snapshot at most the fixed-size header. This keeps decoding independent of later caller mutation without copying an advertised value or trailing data."` | src/protocol/rfcpro.ts:80-81 |

### Wire facts asserted by tests

| What the test asserts | Test name (verbatim) | Citation |
|---|---|---|
| `(0x0203, 0)` → `"02030000"`; `(0x0203, 65534)` → `"0203fffe"`; `(0x0203, 65535)` → `"0203ffff0000ffff"`; `(0x0203, 65536)` → `"0203ffff00010000"`; `(0x0203, 0x7fffffff)` → `"0203ffff7fffffff"` | `"encodes canonical compact and extended RFCPRO field headers"` | test/rfcpro.test.ts:14-40 |
| `"0203ffff00010000aabb"` decodes to length 65536, `bytesConsumed: 8` — trailing bytes ignored; `"0203ffff0000fffe"` decodes to length 65534 with `encoding: "extended"` (a legacy over-long encoding is **accepted**) | `"decodes compact, canonical extended, and tolerated legacy extended lengths"` | test/rfcpro.test.ts:42-85 |
| `rfcProFieldHeaderByteLength`: `0→4`, `65534→4`, `65535→8`, `0x7fffffff→8` | `"reports exact RFCPRO header lengths without allocating payload space"` | test/rfcpro.test.ts:87-92 |
| `"0203ffffffffffff"` → `/extended length -1 is negative/`; `"0203ffff80000000"` → `/extended length -2147483648 is negative/` | `"rejects invalid RFCPRO tags, lengths, and configured maxima"` | test/rfcpro.test.ts:174-181 |
| every truncation of `"0203ffff00010000"` at lengths 0..7 throws `/need [24] bytes/` | `"rejects every truncated RFCPRO extended-length header"` | test/rfcpro.test.ts:184-193 |

### Go mapping notes

- The whole file is ~10 lines of real Go. `Buffer` → `[]byte`; `bytesConsumed: 4 | 8` → `int`.
- `Object.freeze` on the returned header (src/protocol/rfcpro.ts:101, 125) — return a value struct.
- **Signedness matters.** The extended length is `writeInt32BE`/`readInt32BE`
  (src/protocol/rfcpro.ts:59, 114) with an explicit negative check. Go must read `int32`, not
  `uint32`, or the `-1` and `-2147483648` test cases (test/rfcpro.test.ts:174-181) silently become
  huge positive lengths.
- All three errors are `RangeError` with formatted messages; a single `ErrRfcProHeader` sentinel
  wrapped with `%w` is enough — no caller in scope branches on which one.

---

## src/protocol/gateway.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `GATEWAY_NORMAL_CLIENT_LENGTH` | const | `export const GATEWAY_NORMAL_CLIENT_LENGTH = 64;` | src/protocol/gateway.ts:5 |
| `GATEWAY_PROTOCOL_VERSION` | const | `export const GATEWAY_PROTOCOL_VERSION = 2;` | src/protocol/gateway.ts:6 |
| `GATEWAY_NORMAL_CLIENT_REQUEST` | const | `export const GATEWAY_NORMAL_CLIENT_REQUEST = 3;` | src/protocol/gateway.ts:7 |
| `GatewayAcceptInfo` | enum | `ErrorInfo = 0x01, Ping = 0x02, Snc = 0x04, ConnectionExtendedInfo = 0x08, CodePage = 0x10, NiPing = 0x20, ExtendedInitOptions = 0x40, DistributedTrace = 0x80` | src/protocol/gateway.ts:9-18 |
| `GatewayNormalClientRecord` | interface | `address, service, codePage: string; gatewayOptionLevel: number; logicalUnit, transactionProgram, conversationId: string; appcHeaderVersion, acceptInfo, index, returnCode, echoData: number` (all `readonly`) | src/protocol/gateway.ts:20-38 |
| `encodeGatewayNormalClient` | func | `export function encodeGatewayNormalClient(record: GatewayNormalClientRecord): Buffer` | src/protocol/gateway.ts:85 |
| `decodeGatewayNormalClient` | func | `export function decodeGatewayNormalClient(data: Uint8Array): GatewayNormalClientRecord` | src/protocol/gateway.ts:124 |

### Constants and magic values

| Name | Value (verbatim) | What the source says it means | Citation |
|---|---|---|---|
| `GATEWAY_NORMAL_CLIENT_LENGTH` | `64` | exact required record size for version 2 | src/protocol/gateway.ts:5, 135-140 |
| `GATEWAY_PROTOCOL_VERSION` | `2` | first record byte; doc comment: `"Encode the version-2 GW_NORMAL_CLIENT record used before APPC setup."` | src/protocol/gateway.ts:6, 84, 94 |
| `GATEWAY_NORMAL_CLIENT_REQUEST` | `3` | second record byte, request type | src/protocol/gateway.ts:7, 95, 145-147 |
| `GatewayAcceptInfo` | bit flags `0x01`..`0x80` | member names only; **no comment explains any individual bit's wire meaning** | src/protocol/gateway.ts:9-18 |

Field layout, from the write order (code, src/protocol/gateway.ts:94-119) — total 64 bytes:
`u8 version(=2)`, `u8 requestType(=3)`, `4B address (IPv4 dotted quad)`, `u32BE reserved1(=0)`,
`10B service (ASCII, NUL-padded)`, `4B codePage (exactly four ASCII digits, no padding)`,
`5B reserved2(=0)`, `u8 gatewayOptionLevel`, `8B logicalUnit (ASCII, 0x20-padded)`,
`8B transactionProgram (ASCII, 0x20-padded)`, `8B conversationId (ASCII, 0x20-padded)`,
`u8 appcHeaderVersion`, `u8 acceptInfo`, `u16BE index (record.index & 0xffff)`,
`u32BE returnCode`, `u8 echoData`, `u8 filler(=0)`.

`index` is written as `record.index & 0xffff` after a signed-16 validation
(src/protocol/gateway.ts:78-82, 116) and read back as
`unsignedIndex > 0x7fff ? unsignedIndex - 0x1_0000 : unsignedIndex` (src/protocol/gateway.ts:170) —
i.e. a signed 16-bit field.

`decodeAscii` strips a trailing run of NUL **and space** characters: `.replace(/[\x00 ]+$/u, "")`
(src/protocol/gateway.ts:64).

### Errors

| Message text (verbatim) | Type | Condition | Citation |
|---|---|---|---|
| `` `${field} must contain at most ${length} ASCII bytes` `` | `RangeError` | non-string, over-long, or non-printable-ASCII name | src/protocol/gateway.ts:51 |
| `` `${field} contains a non-ASCII byte` `` | `Error` | decode sees a byte that is neither NUL nor `0x20..0x7e` | src/protocol/gateway.ts:61 |
| `"address must be an IPv4 address for gateway protocol version 2"` | `RangeError` | `isIPv4(address)` false | src/protocol/gateway.ts:69 |
| `"index must be a signed 16-bit integer"` | `RangeError` | outside `-0x8000..0x7fff` | src/protocol/gateway.ts:80 |
| `"codePage must contain exactly four ASCII digits"` | `RangeError` | fails `/^\d{4}$/` | src/protocol/gateway.ts:88 |
| `"gateway normal-client record needs at least 2 bytes"` | `RangeError` | decode input < 2 bytes | src/protocol/gateway.ts:126 |
| `"gateway protocol version 3 IPv6 records are not implemented"` | `Error` | first byte === 3 | src/protocol/gateway.ts:130 |
| `` `unsupported gateway protocol version ${version}` `` | `Error` | first byte neither 2 nor 3 | src/protocol/gateway.ts:133 |
| `` `gateway version-2 normal-client record needs ${GATEWAY_NORMAL_CLIENT_LENGTH} bytes; received ${data.byteLength}` `` | `RangeError` | length ≠ 64 | src/protocol/gateway.ts:137-138 |
| `` `expected GW_NORMAL_CLIENT request type 3; received ${requestType}` `` | `Error` | second byte ≠ 3 | src/protocol/gateway.ts:146 |
| `"gateway normal-client reserved1 field must be zero"` | `Error` | `u32BE` at offset 6 ≠ 0 | src/protocol/gateway.ts:150 |
| `"gateway normal-client reserved2 field must be zero"` | `Error` | the 5 bytes at offset 24 not all zero | src/protocol/gateway.ts:155 |
| `"gateway normal-client filler field must be zero"` | `Error` | last byte ≠ 0 | src/protocol/gateway.ts:174 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `"Last byte of the otherwise zero six-byte gateway option region. Client value 6 and server value 15 are observed. Keep it explicit until a second implementation establishes individual option-bit semantics."` | src/protocol/gateway.ts:24-28 |
| `"Encode the version-2 GW_NORMAL_CLIENT record used before APPC setup."` | src/protocol/gateway.ts:84 |
| `"Decode the supported version-2 GW_NORMAL_CLIENT request or response."` | src/protocol/gateway.ts:123 |

### Wire facts asserted by tests

| What the test asserts | Test name (verbatim) | Citation |
|---|---|---|
| the record is 64 bytes; `encoded[0] === 2`, `encoded[1] === 3`; bytes 10..20 are `"sapgw00\x00\x00\x00"` (NUL padding for `service`); a full round trip preserves every field including `index: -1` | `"round-trips the proven 64-byte gateway normal-client record"` | test/gateway.test.ts:30-37 |
| bytes 24..29 are five zeros; `encoded[29] === 6` (the client `gatewayOptionLevel`); bytes 46..54 are eight spaces (an empty `conversationId` is space-padded) | `"keeps gateway padding and option level explicit"` | test/gateway.test.ts:39-44 |
| an IPv6 address is refused; a 9-byte `logicalUnit` is refused; setting the first byte to `3` throws `/version 3.*not implemented/` | `"rejects unsupported gateway variants and malformed fixed fields"` | test/gateway.test.ts:46-65 |

Test fixture (test/gateway.test.ts:10-28) uses `gatewayOptionLevel: 6`, `appcHeaderVersion: 6`,
`codePage: "1100"`, and `acceptInfo = ErrorInfo | Ping | ConnectionExtendedInfo |
ExtendedInitOptions | DistributedTrace` (= `0x01|0x02|0x08|0x40|0x80` = `0xcb`). The source does not
state that this combination is required — it is one fixture.

### Go mapping notes

- `isIPv4` (`node:net`) → `net.ParseIP(addr).To4() != nil` plus a dotted-quad form check;
  `address.split(".").map(Number)` (src/protocol/gateway.ts:71) does **not** validate range on its
  own — it relies entirely on `isIPv4`.
- `GatewayAcceptInfo` → `type GatewayAcceptInfo uint8` with `iota`-shifted constants; it is a bitmask
  and the record field is a plain `number` (`acceptInfo: number`, src/protocol/gateway.ts:34), so
  don't over-type it.
- **Signed `index`.** The manual sign extension at src/protocol/gateway.ts:170 becomes
  `int16(binary.BigEndian.Uint16(...))` in Go — one conversion, and the TS ceremony disappears.
- All errors here are unconditional shape violations; one `ErrGatewayRecord` sentinel with wrapped
  detail is sufficient. The version-3 case (src/protocol/gateway.ts:130) is worth its own sentinel:
  it is `"not implemented"`, not `"malformed"`, and a caller may want to distinguish them.
- `conversationId: ""` encodes to eight `0x20` bytes and decodes back to `""` — the empty/absent
  distinction that appc.ts preserves via NUL padding (src/protocol/appc.ts:380-382) is **not**
  preserved here; gateway pads names with `0x20` (src/protocol/gateway.ts:103-113).

---

## src/protocol/message-server.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `MESSAGE_SERVER_HEADER_LENGTH` | const | `export const MESSAGE_SERVER_HEADER_LENGTH = 110;` | src/protocol/message-server.ts:3 |
| `MESSAGE_SERVER_RFC_GROUP_REQUEST_LENGTH` | const | `export const MESSAGE_SERVER_RFC_GROUP_REQUEST_LENGTH = 206;` | src/protocol/message-server.ts:4 |
| `MAX_MESSAGE_SERVER_PAYLOAD_LENGTH` | const | `export const MAX_MESSAGE_SERVER_PAYLOAD_LENGTH = 512;` | src/protocol/message-server.ts:5 |
| `MessageServerProtocolErrorCode` | type | `"MS_OPCODE_REJECTED" \| "MS_PROTOCOL_ERROR" \| "MS_SERVER_REJECTED" \| "MS_UNSUPPORTED_VERSION"` | src/protocol/message-server.ts:22-26 |
| `MessageServerProtocolError` | class | `extends Error` with `readonly code: MessageServerProtocolErrorCode; readonly serverError: number \| undefined; readonly opcodeError: number \| undefined; override readonly cause: unknown;` and `constructor(code, message, properties: {cause?, serverError?, opcodeError?} = {})` | src/protocol/message-server.ts:28-50 |
| `MessageServerRfcGroupTarget` | interface | `applicationServerHost: string; dispatcherPort: number; gatewayPort: number; gatewayService: string; systemNumber: string` (all `readonly`) | src/protocol/message-server.ts:52-60 |
| `encodeMessageServerLoginRequest` | func | `export function encodeMessageServerLoginRequest(): Buffer` | src/protocol/message-server.ts:229 |
| `encodeMessageServerLogoutRequest` | func | `export function encodeMessageServerLogoutRequest(): Buffer` | src/protocol/message-server.ts:247 |
| `decodeMessageServerLoginResponse` | func | `export function decodeMessageServerLoginResponse(payload: Uint8Array): void` | src/protocol/message-server.ts:264 |
| `encodeMessageServerRfcGroupRequest` | func | `export function encodeMessageServerRfcGroupRequest(group: string): Buffer` | src/protocol/message-server.ts:286 |
| `decodeMessageServerRfcGroupResponse` | func | `export function decodeMessageServerRfcGroupResponse(payload: Uint8Array, expectedGroup: string,): MessageServerRfcGroupTarget` | src/protocol/message-server.ts:348-351 |

### Constants and magic values

| Name | Value (verbatim) | What the source says it means | Citation |
|---|---|---|---|
| `MESSAGE_SERVER_HEADER_LENGTH` | `110` | fixed header size; login request/response are exactly this | src/protocol/message-server.ts:3, 266 |
| `MESSAGE_SERVER_RFC_GROUP_REQUEST_LENGTH` | `206` | fixed RFC-group request size | src/protocol/message-server.ts:4, 289 |
| `MAX_MESSAGE_SERVER_PAYLOAD_LENGTH` | `512` | upper bound on an RFC-group response | src/protocol/message-server.ts:5, 356 |
| `MESSAGE_EYECATCHER` (**not exported**) | `Buffer.from("**MESSAGE**\0", "ascii")` (12 bytes) | first field of every record | src/protocol/message-server.ts:7, 115 |
| `MESSAGE_VERSION` (**not exported**) | `4` | 13th byte; mismatch gives `MS_UNSUPPORTED_VERSION` | src/protocol/message-server.ts:8, 116, 154-158 |
| `NAME_FIELD_LENGTH` (**not exported**) | `40` | width of `toName` and `fromName` | src/protocol/message-server.ts:9, 84 |
| `MESSAGE_SERVER_NAME` (**not exported**) | `"MSG_SERVER"` | expected `fromName` in responses, `toName` in the group request | src/protocol/message-server.ts:10, 279, 294, 373 |
| `RFC_GROUP_OPCODE` (**not exported**) | `0x2c` | opcode byte of the RFC-group exchange | src/protocol/message-server.ts:11, 301, 376 |
| `RFC_GROUP_OPCODE_VERSION` (**not exported**) | `1` | opcode version byte | src/protocol/message-server.ts:12, 303, 379 |
| `RFC_GROUP_REPLY_CHARSET` (**not exported**) | `3` | required charset byte **in the reply** (the request sends `0`) | src/protocol/message-server.ts:13, 304, 385-389 |
| `RFC_GROUP_MAX_BYTES` (**not exported**) | `40` | group name field width | src/protocol/message-server.ts:14, 309 |
| `RFC_GROUP_MAX_HOST_BYTES` (**not exported**) | `255` | hostname length cap | src/protocol/message-server.ts:15, 419 |
| `PORT_BLOCK_STRIDE` (**not exported**) | `100` | comment: `"Distance between the sapdpNN and sapgwNN service blocks, and their width."` | src/protocol/message-server.ts:16-17 |
| `MAX_TCP_PORT` (**not exported**) | `0xffff` | derived gateway port must not exceed it | src/protocol/message-server.ts:18, 410 |
| `RFC_LOGON_SELECTOR` (**not exported**) | `0x34` | comment: `"Logon-group selector opcode used by the lgtst RFC-group exchange."` | src/protocol/message-server.ts:19-20, 315 |

Header layout (code, `writeHeader` src/protocol/message-server.ts:115-141): `12B eyecatcher`,
`u8 version(=4)`, `u8 error`, `40B toName`, `u8 messageType(=0)`, `u8 reserved(=0)`, `u8 domain(=0)`,
`u8 reserved2`, `8B key(=0)`, `u8 flag`, `u8 interfaceFlag`, `40B fromName`, `u16BE portOrPadding(=0)`.

Per-message header expectations (code):

| Message | toName / padding | reserved2 | flag | interfaceFlag | fromName / padding | Citation |
|---|---|---|---|---|---|---|
| login request | `"-"` / `0` | `0` | `2` | `8` | `"-"` / `0x20` | src/protocol/message-server.ts:234-242 |
| logout request | `"-"` / `0` | `0` | `0` | `4` | `"-"` / `0x20` | src/protocol/message-server.ts:252-260 |
| login response (expected) | `"-"` / `0x20` | `1` | `2` | `8` | `"MSG_SERVER"` / `0x20` | src/protocol/message-server.ts:273-281 |
| RFC-group request | `"MSG_SERVER"` / `0` | `0` | `2` | `1` | `"-"` / `0x20` | src/protocol/message-server.ts:292-300 |
| RFC-group response (expected) | `"-"` / `0x20` | `0` | `3` | `1` | `"MSG_SERVER"` / `0` | src/protocol/message-server.ts:367-375 |

RFC-group request body after the header (code, src/protocol/message-server.ts:301-317):
`u8 opcode(0x2c)`, `u8 opcodeError(0)`, `u8 opcodeVersion(1)`, `u8 opcodeCharset(0)`,
`40B zero requestPrefix`, `u32BE groupBlockLength = group.byteLength + 4`, `group bytes`,
`(40 - group.byteLength)B zero padding`, `u8 resultVersion(1)`, `u8 resultReserved(0)`,
`u16BE resultStatus(0)`, `u16BE logonSelector(0x34)`, `u16BE hostLength(0)`.

RFC-group response body (code, src/protocol/message-server.ts:376-434): `u8 opcode(=0x2c)`,
`u8 opcodeError`, `u8 opcodeVersion(=1)`, `u8 opcodeCharset(=3)`, then `decodeGroupEcho`
(40 zero bytes, `u32BE blockLength = group+4`, echoed group, zero padding to 40),
`u8 resultVersion(=1)`, `u8 resultReserved(=0)`, `u16BE resultStatus(=0)`, `u16BE dispatcherPort`,
`u16BE hostLength`, `hostLength` host bytes, `u8 trailer(=0x20)`.

Derived route (code, src/protocol/message-server.ts:409, 437-446): `gatewayPort = dispatcherPort +
100`; `systemNumber = (dispatcherPort % PORT_BLOCK_STRIDE).toString(10).padStart(2, "0")`;
`gatewayService` is the template `sapgw${systemNumber}`.

Host validation (code, src/protocol/message-server.ts:428): the regex admits bytes `0x21` through
`0x7e` only, so a space is rejected inside the host even though the **trailer** byte must be `0x20`.

### Errors

Every failure is a `MessageServerProtocolError`; `guardedDecode` wraps any other throw as
`MS_PROTOCOL_ERROR` with the message `malformed message-server <description>` and the original as
`cause` (src/protocol/message-server.ts:204-215).

| Message text (verbatim) | Code | Condition | Citation |
|---|---|---|---|
| `` `invalid message-server ${field}` `` | `MS_PROTOCOL_ERROR` | `equalBytes` mismatch; fields: `"eyecatcher"`, `"toName"`, `"key"`, `"fromName"`, `"RFC-group response prefix"`, `"RFC-group echo"`, `"RFC-group padding"` | src/protocol/message-server.ts:95-98 |
| `` `invalid message-server ${field}: expected ${expected}, received ${actual}` `` | `MS_PROTOCOL_ERROR` | `exactByte` mismatch; fields: `"message type"`, `"reserved byte"`, `"domain"`, `"second reserved byte"`, `"message flag"`, `"interface flag"`, `"port/padding field"`, `"RFC-group opcode"`, `"RFC-group response charset"`, `"RFC-group result version"`, `"RFC-group result reserved byte"`, `"RFC-group result status"`, `"RFC-group response trailer"` | src/protocol/message-server.ts:104-107 |
| `` `unsupported message-server version ${version}` `` | `MS_UNSUPPORTED_VERSION` | version byte not 4 | src/protocol/message-server.ts:156 |
| `` `message server rejected the request with error ${serverError}` `` | `MS_SERVER_REJECTED` (carries `serverError`) | header `error` byte nonzero, checked **after** every other header field | src/protocol/message-server.ts:196-200 |
| `` `malformed message-server ${description}` `` | `MS_PROTOCOL_ERROR` (carries `cause`) | any non-protocol throw inside a decode; descriptions `"login response"`, `"RFC-group response"` | src/protocol/message-server.ts:210-213 |
| `"group must contain 1..40 printable ASCII bytes"` | `RangeError` (**not** a protocol error) | group fails the printable-ASCII 1..40 regex | src/protocol/message-server.ts:222-224 |
| `` `${field} must fit a 40-byte printable ASCII field` `` | `RangeError` | `toName`/`fromName` overflow or non-printable | src/protocol/message-server.ts:82 |
| `` `message-server login response must be ${MESSAGE_SERVER_HEADER_LENGTH} bytes` `` | `MS_PROTOCOL_ERROR` | length not 110 | src/protocol/message-server.ts:267-270 |
| `"message-server RFC-group response length is outside its bounded shape"` | `MS_PROTOCOL_ERROR` | below 110 or above 512 | src/protocol/message-server.ts:358-361 |
| `"message-server RFC-group block length does not match the request"` | `MS_PROTOCOL_ERROR` | echoed block length is not `group.byteLength + 4` | src/protocol/message-server.ts:331-334 |
| `` `unsupported message-server RFC-group opcode version ${opcodeVersion}` `` | `MS_UNSUPPORTED_VERSION` | opcode version not 1 | src/protocol/message-server.ts:380-383 |
| `` `message-server RFC-group lookup failed with opcode error ${opcodeError}` `` | `MS_OPCODE_REJECTED` (carries `opcodeError`) | opcode error byte nonzero, checked **after** the charset byte | src/protocol/message-server.ts:391-395 |
| `` `message-server returned an unusable dispatcher port ${dispatcherPort}` `` | `MS_PROTOCOL_ERROR` | `dispatcherPort < 1` or `dispatcherPort + 100 > 0xffff` | src/protocol/message-server.ts:411-414 |
| `"message-server returned an invalid RFC-group hostname length"` | `MS_PROTOCOL_ERROR` | `hostLength < 1`, above 255, or `reader.remaining !== hostLength + 1` | src/protocol/message-server.ts:417-425 |
| `"message-server returned a non-ASCII or empty RFC-group hostname"` | `MS_PROTOCOL_ERROR` | host fails the `0x21..0x7e` regex | src/protocol/message-server.ts:428-432 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `"Distance between the sapdpNN and sapgwNN service blocks, and their width."` | src/protocol/message-server.ts:16 |
| `"Logon-group selector opcode used by the lgtst RFC-group exchange."` | src/protocol/message-server.ts:19 |
| `"One-way MS_LOGOUT control record observed after a completed lookup."` | src/protocol/message-server.ts:246 |
| `"Dispatcher port as returned by the message server (32NN by default)."` | src/protocol/message-server.ts:54 |
| `"Gateway port in the block one hundred ports above the dispatcher block."` | src/protocol/message-server.ts:56 |
| `"Every read below is bounded by CheckedByteReader and guardedDecode maps a short read to MS_PROTOCOL_ERROR, so the body needs no length pre-check."` | src/protocol/message-server.ts:397-398 |
| `"SAP allocates sapdpNN and sapgwNN as contiguous per-instance blocks one hundred ports apart; 3200/3300 is the default block offset, not a protocol constant. Bound the selected and derived endpoints structurally instead of pinning one landscape's offset, so an offset or non-standard instance profile is not refused for a structurally valid reply."` | src/protocol/message-server.ts:404-408 |

### Wire facts asserted by tests

The test file pins four complete records as hex. These are the strongest conformance vectors in the
whole protocol layer.

| What the test asserts | Test name (verbatim) | Citation |
|---|---|---|
| `encodeMessageServerLoginRequest()` deep-equals a pinned 110-byte record; `encodeMessageServerRfcGroupRequest("RFC_GROUP")` deep-equals a pinned 206-byte record; `encodeMessageServerLogoutRequest()` deep-equals a pinned 110-byte record; the pinned login response decodes without throwing | `"encodes the captured version-4 login and version-1 RFC-group request shape"` | test/message-server.test.ts:57-62, fixtures at 15-38 |
| the pinned group response decodes to `{applicationServerHost:"app.example.test", dispatcherPort:3200, gatewayPort:3300, gatewayService:"sapgw00", systemNumber:"00"}` | `"decodes the echoed group and derives the direct gateway route"` | test/message-server.test.ts:64-75 |
| for **every** `dispatcherPort` in 1 through `0xffff-100`, the derived route is `{dispatcherPort, gatewayPort: dispatcherPort+100, systemNumber: (port % 100) zero-padded to 2, gatewayService: "sapgw" + systemNumber}` | `"derives the same route shape at every legal dispatcher port"` | test/message-server.test.ts:77-103 |
| group inputs empty, 41 chars, one containing NUL, and one containing a non-ASCII letter all throw the 1..40 printable-ASCII message | `"rejects invalid group inputs before allocating a request"` | test/message-server.test.ts:105-112 |
| a login response one byte short, one byte long, or mutated at offsets 0, 12, 13, 57, 66, 67 is rejected with `MS_PROTOCOL_ERROR`, `MS_SERVER_REJECTED`, or `MS_UNSUPPORTED_VERSION` | `"rejects truncated, extended, and structurally inconsistent login replies"` | test/message-server.test.ts:114-133 |
| a group response truncated to 110 bytes with the error byte set to 84 yields `MS_SERVER_REJECTED` with `serverError === 84`; truncated to 114 bytes with offset 111 set to 5 yields `MS_OPCODE_REJECTED` with `opcodeError === 5` — i.e. **both rejections are decodable from a short reply** | `"distinguishes message-server and opcode rejection from malformed replies"` | test/message-server.test.ts:135-152 |
| mutating offsets 66, 67, 110, 112, 113, and the offsets written as `114+84`, `114+85`, `114+86`, `114+43`, `114+44` is rejected | `"rejects every pinned type, opcode, version, charset, and echo mismatch"` | test/message-server.test.ts:154-174 |
| dispatcher port 0, dispatcher port `0xffff` (derived gateway port out of range), `hostLength = 0xffff`, a NUL inside the host, a non-`0x20` trailer, one byte short, and one byte long are all `MS_PROTOCOL_ERROR` | `"enforces dispatcher-port, hostname, trailer, and total-length bounds"` | test/message-server.test.ts:176-203 |
| decoding the pinned response against group `"OTHER"` is `MS_PROTOCOL_ERROR` | `"does not accept a response for a different requested group"` | test/message-server.test.ts:205-210 |

### Go mapping notes

- **This is the cleanest sentinel-error case in the layer.** `MessageServerProtocolError` already
  carries a four-valued `code` plus two optional numeric details
  (src/protocol/message-server.ts:28-50). In Go: one exported struct type implementing `error`, with
  a `Code` field of an exported string/int type and `ServerError`/`OpcodeError` as `*int` — they are
  `number | undefined`, and `0` is a meaningful value that means "no rejection".
- **`guardedDecode` is a try/catch re-wrapper** (src/protocol/message-server.ts:204-215). In Go this
  becomes explicit error wrapping at each read site, since a Go reader returns errors rather than
  throwing.
- `decodeMessageServerLoginResponse` returns `void` and communicates purely by throwing
  (src/protocol/message-server.ts:264). In Go it returns `error`.
- **Ordering is load-bearing and must be preserved:** the `serverError` check happens *after* all
  other header fields (src/protocol/message-server.ts:195-201), and the `opcodeError` check happens
  *after* the charset byte (src/protocol/message-server.ts:390-396). The test at
  test/message-server.test.ts:135-152 depends on truncated payloads still reaching those checks.
- `Object.freeze` on the returned target (src/protocol/message-server.ts:440) becomes a value struct.
- No Promise/async/AbortSignal; this file is pure encode/decode. The transport that uses it is out of
  scope (`src/transport/message-server-resolver.ts`).

---

## src/protocol/password-scramble.ts

56 lines, one exported function.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `scrambleRfcPassword` | func | `export function scrambleRfcPassword(password: string, seed?: number): Buffer` | src/protocol/password-scramble.ts:12 |

### Constants and magic values

| Name | Value (verbatim) | What the source says it means | Citation |
|---|---|---|---|
| `PASSWORD_SCRAMBLE_TABLE` (**not exported**) | `0xf0, 0xed, 0x53, 0xb8, 0x32, 0x44, 0xf1, 0xf8, 0x76, 0xc6, 0x79, 0x59, 0xfd, 0x4f, 0x13, 0xa2, 0xc1, 0x51, 0x95, 0xec, 0x54, 0x83, 0xc2, 0x34, 0x77, 0x49, 0x43, 0xa2, 0x7d, 0xe2, 0x65, 0x96, 0x5e, 0x53, 0x98, 0x78, 0x9a, 0x17, 0xa3, 0x3c, 0xd3, 0x83, 0xa8, 0xb8, 0x29, 0xfb, 0xdc, 0xa5, 0x55, 0xd7, 0x02, 0x77, 0x84, 0x13, 0xac, 0xdd, 0xf9, 0xb8, 0x31, 0x16, 0x61, 0x0e, 0x6d, 0xfa` (64 bytes) | no comment states its origin or meaning; it is indexed modulo 64 | src/protocol/password-scramble.ts:3-9, 45 |
| maximum password length | `40` | comment: `"Every admitted character is one-byte ASCII, so this cheap code-unit check rejects oversized local input before regex scanning or Buffer allocation."` | src/protocol/password-scramble.ts:16-19 |
| admitted character set | printable ASCII `0x20` through `0x7e` | the error text calls it `"the proven ASCII baseline"` | src/protocol/password-scramble.ts:21-24 |
| default seed | `randomBytes(4).readUInt32LE(0)` | used when the caller passes no seed | src/protocol/password-scramble.ts:29 |

Algorithm, verbatim (code, src/protocol/password-scramble.ts:38-48):

    result = Buffer.alloc(4 + clear.byteLength);
    result.writeUInt32LE(actualSeed, 0);
    const mixedSeed = (actualSeed ^ (actualSeed >>> 5)) >>> 0;
    const startIndex = (mixedSeed ^ ((actualSeed << 1) >>> 0)) >>> 0;
    for (let index = 0; index < clear.byteLength; index += 1) {
      const tableValue = PASSWORD_SCRAMBLE_TABLE[(startIndex + index) & 0x3f]!;
      const seedTerm = (actualSeed * index * index - index) & 0xff;
      result[4 + index] = clear[index]! ^ tableValue ^ seedTerm;
    }

Output shape: `4 + password.length` bytes, seed as **little-endian** u32 in the first four bytes.

### Errors

| Message text (verbatim) | Type | Condition | Citation |
|---|---|---|---|
| `"password must be a string"` | `TypeError` | `typeof password !== "string"` | src/protocol/password-scramble.ts:14 |
| `"password must contain at most 40 bytes"` | `RangeError` | `password.length > 40`, checked on **code units**, before the regex | src/protocol/password-scramble.ts:19 |
| `"password contains characters outside the proven ASCII baseline"` | `RangeError` | fails the printable-ASCII regex | src/protocol/password-scramble.ts:22-24 |
| `"password seed must be an unsigned 32-bit integer"` | `RangeError` | seed not a safe integer in 0..0xffffffff | src/protocol/password-scramble.ts:35 |

Cleanup: on any throw the partially built `result` is zero-filled; `clear` is always zero-filled in
`finally` (src/protocol/password-scramble.ts:50-55).

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `"Internal CPIC/WebSocket logon-password field producer."` | src/protocol/password-scramble.ts:11 |
| `"Every admitted character is one-byte ASCII, so this cheap code-unit check rejects oversized local input before regex scanning or Buffer allocation."` | src/protocol/password-scramble.ts:16-17 |
| `"The 40-byte cap above keeps actualSeed * index * index under 2 ** 53, so number arithmetic stays exact and the mask needs no wider type."` | src/protocol/password-scramble.ts:43-44 |
| (test file) `"The producer feeds a wire field a server either accepts or rejects, so its bytes are a fixed point: any refactor here has to reproduce them exactly. These expectations were captured by running the sweep below against the built producer and pinning what it emitted, so a regression shows up as a byte difference rather than as an assertion someone has to re-reason about."` | test/password-scramble-equivalence.test.ts:7-15 |

### Wire facts asserted by tests

| What the test asserts | Test name (verbatim) | Citation |
|---|---|---|
| a 21 592-vector sweep (every seed 0..0xff and 0xffffff00..0xffffffff plus 10 named seeds, times every length 0..40; plus every printable character repeated 40 times at seeds 0 and 0x5ae0b7a3) hashes to SHA-256 `"f3e0b74e48219b80e926e4ff2684c045e4fc93015eca9221a5ddb6a50cd8ff82"`, each vector followed by a `0xff` separator byte | `"produces the frozen field bytes across the full input sweep"` | test/password-scramble-equivalence.test.ts:57-85 |
| six named boundary vectors, reproduced in the conformance-vector section below | `"produces the frozen field bytes for the named boundary cases"` | test/password-scramble-equivalence.test.ts:87-117 |
| `scrambleRfcPassword("secret", 0x5ae0_b7a3).toString("hex") === "a3b7e05a048eaa683470"` | `"scrambles RFC passwords with the pinned current fixed vector"` | test/cpic.test.ts:447-452 |
| default-seed calls produce 10 bytes for `"secret"` and differ between calls | `"uses a fresh seed by default and enforces the proven ASCII baseline"` | test/cpic.test.ts:454-467 |

### Go mapping notes

- **`>>> 0` is JavaScript's only way to get an unsigned 32-bit value.** Every occurrence
  (src/protocol/password-scramble.ts:40-41) disappears in Go if the arithmetic is done in `uint32`.
- **`actualSeed * index * index` is float64 arithmetic in JS**, which the comment at
  src/protocol/password-scramble.ts:43-44 relies on staying exact below 2**53. With `index <= 39` and
  `seed <= 2**32-1` the product is at most about 2**42.6, so it is exact.
  **In Go this must be computed in 64-bit, not `uint32`**, or the masked result differs for large
  seeds. This is the single highest-risk line in the file; the boundary vector for a 40-character
  password at seed `0xffff_ffff` (test/password-scramble-equivalence.test.ts:102-107) exists
  precisely to catch it.
- `- index` before the mask means the intermediate can be **negative** when `seed === 0` — the test
  comment says `"Seed 0 drives the per-position term negative from the second byte on."`
  (test/password-scramble-equivalence.test.ts:90). JS masks a negative value via ToInt32; Go must use
  signed 64-bit arithmetic and then mask, not unsigned subtraction.
- `randomBytes(4).readUInt32LE(0)` becomes `crypto/rand` plus `binary.LittleEndian.Uint32`.
- Zeroing in `finally` (src/protocol/password-scramble.ts:50-55) becomes `defer` plus an explicit
  loop; Go has no guaranteed-zeroing primitive either.

---

## src/protocol/rfc-error-envelope.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RfcErrorTag` | enum | `ExceptionKey = 0x0401, ErrorMessage = 0x0402, RuntimeId = 0x0403, T100Text = 0x0404, MessageV1 = 0x0411, MessageV2 = 0x0412, MessageV3 = 0x0413, MessageV4 = 0x0414, MessageClass = 0x0415, MessageType = 0x0416, MessageNumber = 0x0417, CallStack = 0x0418, Unresolved0420 = 0x0420, UseClassExceptions = 0x0421, ClassExceptionInfo = 0x0422, ClassException = 0x0423, ClassExceptionEnd = 0x0424` | src/protocol/rfc-error-envelope.ts:7-25 |
| `RFC_ERROR_ENVELOPE_END_TAG` | const | `export const RFC_ERROR_ENVELOPE_END_TAG = 0xffff;` | src/protocol/rfc-error-envelope.ts:27 |
| `DEFAULT_MAX_RFC_ERROR_TEXT_BYTE_LENGTH` | const | `export const DEFAULT_MAX_RFC_ERROR_TEXT_BYTE_LENGTH = 1024 * 1024;` | src/protocol/rfc-error-envelope.ts:28 |
| `DEFAULT_MAX_RFC_ERROR_TOTAL_TEXT_BYTE_LENGTH` | const | `export const DEFAULT_MAX_RFC_ERROR_TOTAL_TEXT_BYTE_LENGTH = 4 * 1024 * 1024;` | src/protocol/rfc-error-envelope.ts:29 |
| `DEFAULT_MAX_RFC_ERROR_CONTROL_BYTE_LENGTH` | const | `export const DEFAULT_MAX_RFC_ERROR_CONTROL_BYTE_LENGTH = 4 * 1024;` | src/protocol/rfc-error-envelope.ts:30 |
| `DEFAULT_MAX_RFC_ERROR_TOTAL_CONTROL_BYTE_LENGTH` | const | `export const DEFAULT_MAX_RFC_ERROR_TOTAL_CONTROL_BYTE_LENGTH = 64 * 1024;` | src/protocol/rfc-error-envelope.ts:31 |
| `DEFAULT_MAX_RFC_ERROR_CONTROL_COUNT` | const | `export const DEFAULT_MAX_RFC_ERROR_CONTROL_COUNT = 64;` | src/protocol/rfc-error-envelope.ts:32 |
| `DEFAULT_MAX_RFC_ERROR_ENVELOPE_FIELD_COUNT` | const | `export const DEFAULT_MAX_RFC_ERROR_ENVELOPE_FIELD_COUNT = 256;` | src/protocol/rfc-error-envelope.ts:33 |
| `RfcErrorEnvelopeField` | interface | `readonly tag: number;` `readonly value: Uint8Array;` | src/protocol/rfc-error-envelope.ts:93-96 |
| `RfcErrorFactProvenance` | interface | `readonly tag: number;` `readonly ordinal: number;` `readonly byteLength: number;` | src/protocol/rfc-error-envelope.ts:98-102 |
| `RfcUnresolvedControlFact` | interface | `readonly tag: RfcErrorTag.Unresolved0420;` `readonly ordinal: number;` `readonly byteLength: number;` `readonly valueHex: string;` | src/protocol/rfc-error-envelope.ts:104-110 |
| `RfcRemoteErrorFacts` | interface | `exceptionKey, plainText, runtimeId, t100Text, messageClass, messageType, messageNumber, messageV1, messageV2, messageV3, messageV4, callStack: string; provenance: readonly RfcErrorFactProvenance[]; unresolved0420: readonly RfcUnresolvedControlFact[]` | src/protocol/rfc-error-envelope.ts:112-127 |
| `RfcErrorEnvelopeOutcome` | type | `"success" \| "abapException" \| "abapRuntime" \| "abapMessage"` | src/protocol/rfc-error-envelope.ts:129-133 |
| `RfcErrorEnvelope` | interface | `readonly outcome: RfcErrorEnvelopeOutcome;` `readonly successControl: "zeroControl" \| "notApplicable";` `readonly facts: RfcRemoteErrorFacts;` | src/protocol/rfc-error-envelope.ts:135-141 |
| `RfcErrorEnvelopeDecodeOptions` | interface | `maxTextByteLength?, maxTotalTextByteLength?, maxControlByteLength?, maxTotalControlByteLength?, maxControlCount?, maxFieldCount?: number; additionalAllowedTags?: readonly number[]` | src/protocol/rfc-error-envelope.ts:143-155 |
| `RfcErrorEnvelopeReasonCode` | type | 19-member string union, listed below | src/protocol/rfc-error-envelope.ts:157-176 |
| `RfcErrorEnvelopeProtocolError` | class | `extends Error` with `readonly reasonCode: RfcErrorEnvelopeReasonCode;` `override readonly cause: unknown;` and `constructor(reasonCode, message: string, cause?: unknown)` | src/protocol/rfc-error-envelope.ts:178-192 |
| `decodeRfcErrorEnvelope` | func | `export function decodeRfcErrorEnvelope(fields: readonly RfcErrorEnvelopeField[], options: RfcErrorEnvelopeDecodeOptions = {},): RfcErrorEnvelope` | src/protocol/rfc-error-envelope.ts:320-323 |

`RfcErrorEnvelopeReasonCode` members, verbatim (src/protocol/rfc-error-envelope.ts:158-176):
`"RFC_ERROR_ENVELOPE_INVALID_FIELD"`, `"RFC_ERROR_ENVELOPE_MISSING_END"`,
`"RFC_ERROR_ENVELOPE_INVALID_END"`, `"RFC_ERROR_ENVELOPE_DUPLICATE_FACT"`,
`"RFC_ERROR_ENVELOPE_TEXT_TOO_LARGE"`, `"RFC_ERROR_ENVELOPE_TOTAL_TEXT_TOO_LARGE"`,
`"RFC_ERROR_ENVELOPE_ODD_UTF16_LENGTH"`, `"RFC_ERROR_ENVELOPE_EMBEDDED_NUL"`,
`"RFC_ERROR_ENVELOPE_UNPAIRED_SURROGATE"`, `"RFC_ERROR_ENVELOPE_EMPTY_DISCRIMINATOR"`,
`"RFC_ERROR_ENVELOPE_CONFLICTING_DISCRIMINATORS"`, `"RFC_ERROR_ENVELOPE_AMBIGUOUS_FACTS"`,
`"RFC_ERROR_ENVELOPE_UNKNOWN_TAG"`, `"RFC_ERROR_ENVELOPE_CLASS_EXCEPTION_UNSUPPORTED"`,
`"RFC_ERROR_ENVELOPE_CONTROL_TOO_LARGE"`, `"RFC_ERROR_ENVELOPE_TOTAL_CONTROL_TOO_LARGE"`,
`"RFC_ERROR_ENVELOPE_TOO_MANY_CONTROLS"`, `"RFC_ERROR_ENVELOPE_TOO_MANY_FIELDS"`,
`"RFC_ERROR_ENVELOPE_UNRESOLVED_SUCCESS_CONTROL"`.

### Constants and magic values

| Name | Value (verbatim) | What the source says it means | Citation |
|---|---|---|---|
| `CLASSIC_RESPONSE_DATA_TAGS` (**not exported**) | `0x0102 // function`, `0x0201 // parameter name`, `0x0203 // parameter value`, `0x0205 // requested output`, `0x0301 // table name`, `0x0302 // table header`, `0x0303 // uncompressed table content`, `0x0304 // simple-compressed table content`, `0x0502 // context end`, `0x0503 // response context`, `0x0512 // call context`, `0x0514 // session` | comment: `"Classic response tags which are not part of an error record."` — the inline `//` comments above are the only in-source naming of these tags | src/protocol/rfc-error-envelope.ts:35-49 |
| `TEXT_TAGS` (**not exported**) | `ExceptionKey, ErrorMessage, RuntimeId, T100Text, MessageV1..V4, MessageClass, MessageType, MessageNumber, CallStack` (12 tags) | tags decoded as strict UTF-16LE, singleton | src/protocol/rfc-error-envelope.ts:51-64 |
| `CLASS_EXCEPTION_TAGS` (**not exported**) | `ClassExceptionInfo(0x0422), ClassException(0x0423), ClassExceptionEnd(0x0424)` | reaching this set rejects with `CLASS_EXCEPTION_UNSUPPORTED` (0x0422 is handled earlier) | src/protocol/rfc-error-envelope.ts:66-70, 560-565 |
| `MESSAGE_TEXT_TAGS` (**not exported**) | `ErrorMessage(0x0402), T100Text(0x0404)` | either one non-empty implies `abapMessage` | src/protocol/rfc-error-envelope.ts:72-75, 614 |
| `MESSAGE_IDENTITY_TAGS` (**not exported**) | `MessageClass(0x0415), MessageType(0x0416), MessageNumber(0x0417)` | **all three** non-empty implies `abapMessage` | src/protocol/rfc-error-envelope.ts:77-81, 296-303 |
| `SECONDARY_ERROR_TAGS` (**not exported**) | `MessageV1..V4, CallStack` + `MESSAGE_TEXT_TAGS` + `MESSAGE_IDENTITY_TAGS` | presence of any of these without a discriminator is `AMBIGUOUS_FACTS` | src/protocol/rfc-error-envelope.ts:83-91, 618-622 |

Classification order (code, src/protocol/rfc-error-envelope.ts:607-638), verbatim in effect:
1. `ExceptionKey` present → `"abapException"`.
2. else `RuntimeId` present → `"abapRuntime"`.
3. else a non-empty `ErrorMessage`/`T100Text`, **or** all three of class/type/number non-empty →
   `"abapMessage"`.
4. else any secondary error tag present → throw `AMBIGUOUS_FACTS`.
5. else require **exactly one** `0x0420` control of exactly 4 bytes with `valueHex === "00000000"`
   → `"success"` with `successControl: "zeroControl"`; anything else throws
   `UNRESOLVED_SUCCESS_CONTROL`.

Strict UTF-16LE decoding (code, `decodeStrictUtf16Le`, src/protocol/rfc-error-envelope.ts:219-270):
rejects odd byte length, any zero code unit, an unpaired high or low surrogate; then
`bytes.toString("utf16le").replace(/ +$/u, "")` — **only trailing spaces are trimmed**.

Control facts retain `valueHex` (`Buffer.from(value).toString("hex")`) and never a byte view; the
interface comment says `"Lowercase hexadecimal; retained internally without a mutable byte view."`
(src/protocol/rfc-error-envelope.ts:108, 484).

### Errors

Every failure is `RfcErrorEnvelopeProtocolError` via the `protocolError` helper
(src/protocol/rfc-error-envelope.ts:194-200), except the `boundedInteger` option guards which throw
plain `RangeError`.

| Message text (verbatim) | Reason code | Condition | Citation |
|---|---|---|---|
| `` `${field} must be an integer in ${minimum}..${maximum}` `` | (plain `RangeError`) | any of `maxTextByteLength`, `maxTotalTextByteLength`, `maxControlByteLength`, `maxTotalControlByteLength`, `maxControlCount` (0..0x7fffffff), `maxFieldCount` (1..0x7fffffff), `additionalAllowedTags entry` (0..0xffff) | src/protocol/rfc-error-envelope.ts:213-215, 338-369 |
| `"RFCPRO response fields must be an array"` | `INVALID_FIELD` | `!Array.isArray(fields)` | src/protocol/rfc-error-envelope.ts:371-376 |
| `"RFCPRO response exceeds the configured envelope field-count limit"` | `TOO_MANY_FIELDS` | `fields.length > maxFieldCount` | src/protocol/rfc-error-envelope.ts:377-382 |
| `` `RFCPRO response field ${ordinal} is invalid` `` | `INVALID_FIELD` | non-object, null, tag not a safe integer in 0..0xffff, or value not a `Uint8Array` | src/protocol/rfc-error-envelope.ts:385-397 |
| `"RFCPRO response lacks its terminal End field"` | `MISSING_END` | no `0xffff` field | src/protocol/rfc-error-envelope.ts:400-405 |
| `"RFCPRO response End field must occur once, last, with zero length"` | `INVALID_END` | more than one End, not last, or non-empty | src/protocol/rfc-error-envelope.ts:407-416 |
| `` `RFCPRO response repeats singleton error fact ${tagText(tag)}` `` | `DUPLICATE_FACT` | a text tag seen twice | src/protocol/rfc-error-envelope.ts:433-436 |
| `` `RFCPRO error fact ${tagText(tag)} exceeds the configured text limit` `` | `TEXT_TOO_LARGE` | per-fact byte cap (checked in two places) | src/protocol/rfc-error-envelope.ts:225-228, 438-442 |
| `"RFCPRO error facts exceed the configured aggregate text limit"` | `TOTAL_TEXT_TOO_LARGE` | running text total over cap | src/protocol/rfc-error-envelope.ts:444-448 |
| `` `RFCPRO error fact ${tagText(tag)} has an odd UTF-16LE byte length` `` | `ODD_UTF16_LENGTH` | odd byte length | src/protocol/rfc-error-envelope.ts:230-234 |
| `` `RFCPRO error fact ${tagText(tag)} contains NUL` `` | `EMBEDDED_NUL` | any zero code unit | src/protocol/rfc-error-envelope.ts:240-244 |
| `` `RFCPRO error fact ${tagText(tag)} ends with an unpaired surrogate` `` / `` `…contains an unpaired surrogate` `` | `UNPAIRED_SURROGATE` | high surrogate at end, high not followed by low, or a lone low surrogate | src/protocol/rfc-error-envelope.ts:246-266 |
| `"RFCPRO response exceeds the configured control-count limit"` | `TOO_MANY_CONTROLS` | control count cap, checked for `0x0420`, `0x0421`, `0x0422` | src/protocol/rfc-error-envelope.ts:457-461, 492-494, 525-527 |
| `"unresolved RFCPRO control 0x0420 exceeds the configured limit"` | `CONTROL_TOO_LARGE` | single `0x0420` over `maxControlByteLength` | src/protocol/rfc-error-envelope.ts:463-467 |
| `"RFCPRO class-exception control exceeds the configured limit"` | `CONTROL_TOO_LARGE` | `0x0421` over cap | src/protocol/rfc-error-envelope.ts:502-506 |
| `"RFCPRO class-exception info exceeds the configured limit"` | `CONTROL_TOO_LARGE` | `0x0422` over cap | src/protocol/rfc-error-envelope.ts:536-540 |
| `"RFCPRO response controls exceed the configured aggregate byte limit"` | `TOTAL_CONTROL_TOO_LARGE` | running control total over cap (three sites) | src/protocol/rfc-error-envelope.ts:469-477, 508-516, 542-550 |
| `"RFCPRO response repeats singleton class-exception control 0x0421"` | `DUPLICATE_FACT` | second `0x0421` | src/protocol/rfc-error-envelope.ts:496-501 |
| `"RFCPRO response repeats singleton class-exception info 0x0422"` | `DUPLICATE_FACT` | second `0x0422` | src/protocol/rfc-error-envelope.ts:530-535 |
| `` `RFCPRO class-exception fact ${tagText(tag)} is not supported` `` | `CLASS_EXCEPTION_UNSUPPORTED` | tag `0x0423` or `0x0424` | src/protocol/rfc-error-envelope.ts:560-565 |
| `` `RFCPRO response contains unknown tag ${tagText(tag)}` `` | `UNKNOWN_TAG` | tag outside `CLASSIC_RESPONSE_DATA_TAGS` plus `additionalAllowedTags` | src/protocol/rfc-error-envelope.ts:566-571 |
| `"RFCPRO declared-exception key is empty"` / `"RFCPRO runtime identifier is empty"` | `EMPTY_DISCRIMINATOR` | discriminator present but decodes to `""` (including all-spaces, which right-trims to empty) | src/protocol/rfc-error-envelope.ts:576-587 |
| `"RFCPRO response contains both declared-exception and runtime identifiers"` | `CONFLICTING_DISCRIMINATORS` | both `0x0401` and `0x0403` present | src/protocol/rfc-error-envelope.ts:588-596 |
| `"RFCPRO class-exception info is only supported as supplemental data for a classic declared exception"` | `CLASS_EXCEPTION_UNSUPPORTED` | `0x0422` present without `0x0401`, or together with `0x0421` | src/protocol/rfc-error-envelope.ts:597-605 |
| `"RFCPRO response contains secondary error facts without a discriminator"` | `AMBIGUOUS_FACTS` | classification step 4 | src/protocol/rfc-error-envelope.ts:618-622 |
| `"RFCPRO response lacks the zero 0x0420 success control"` | `UNRESOLVED_SUCCESS_CONTROL` | classification step 5 | src/protocol/rfc-error-envelope.ts:624-635 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `"Error-related RFCPRO identifiers used in classic RFC response envelopes. Identifier 0x0420 deliberately remains unresolved: the only form observed in successful responses is a single four-byte zero."` | src/protocol/rfc-error-envelope.ts:1-6 |
| `"Classic response tags which are not part of an error record."` | src/protocol/rfc-error-envelope.ts:35 |
| `"Lowercase hexadecimal; retained internally without a mutable byte view."` | src/protocol/rfc-error-envelope.ts:108 |
| `"Aggregate limit across all decoded UTF-16LE error facts."` | src/protocol/rfc-error-envelope.ts:146 |
| `"Aggregate limit across all copied unresolved/control values."` | src/protocol/rfc-error-envelope.ts:148 |
| `"Includes application-data fields and the terminal End field."` | src/protocol/rfc-error-envelope.ts:151 |
| `"Additional state-specific data tags accepted but not interpreted here."` | src/protocol/rfc-error-envelope.ts:153 |
| `"Normalize and classify the error/control portion of a decoded RFCPRO response. The caller remains responsible for the outer CPIC prefix, chained closing-tag grammar, and final two-byte trailer."` | src/protocol/rfc-error-envelope.ts:315-319 |
| `"NetWeaver 7.50 can append this opaque basXML-related record to a complete classic declared-exception envelope. Its classic fields are authoritative; retain only bounded provenance and never the payload."` | src/protocol/rfc-error-envelope.ts:554-556 |

### Wire facts asserted by tests

| What the test asserts | Test name (verbatim) | Citation |
|---|---|---|
| a full declared-exception envelope yields `outcome "abapException"`, `successControl "notApplicable"`, `messageV1 "Method = 1"` (5 trailing spaces trimmed), a non-BMP `messageV4 "fourth 🙂"`, `provenance` = one entry per pre-End field with `{tag, ordinal, byteLength}`, and empty `unresolved0420`; the whole result graph is frozen and assignment throws `TypeError` | `"normalizes every classic declared-exception fact without aliasing V1 to text"` | test/rfc-error-envelope.test.ts:52-101 |
| `RuntimeId` present → `"abapRuntime"` regardless of field order; class+type+number+text → `"abapMessage"` | `"classifies runtime and MESSAGE facts independently of field order"` | test/rfc-error-envelope.test.ts:103-129 |
| either `ErrorMessage` or `T100Text` alone is enough for `"abapMessage"` | `"requires plain/T100 text or a coherent class/type/number MESSAGE identity"` | test/rfc-error-envelope.test.ts:131-169 |
| permuted field order gives identical semantic facts but **different provenance ordinals** | `"produces identical semantic facts for deterministic permutations"` | test/rfc-error-envelope.test.ts:171-193 |
| a 4-byte zero `0x0420` with data tags `0x0503`/`0x0201` present → `outcome "success"`, `successControl "zeroControl"`, `unresolved0420 === [{tag:0x0420, ordinal:2, byteLength:4, valueHex:"00000000"}]`; **none** of: absent, `00000001`, 3 bytes, or two controls is accepted | `"recognizes only the observed four-byte zero 0x0420 success control"` | test/rfc-error-envelope.test.ts:195-227 |
| two malformed `0x0420` controls (`"dead"`, `"beef"`) alongside an `ExceptionKey` do **not** prevent `"abapException"`; both are retained as provenance | `"does not let unresolved 0x0420 override an independently classified error"` | test/rfc-error-envelope.test.ts:229-241 |
| duplicate `ExceptionKey` → `DUPLICATE_FACT`; `ExceptionKey` + `RuntimeId` → `CONFLICTING_DISCRIMINATORS` | `"rejects duplicate, conflicting, ambiguous, class-exception, and unknown facts"` | test/rfc-error-envelope.test.ts:243-259 |
| an all-spaces `ExceptionKey`/`RuntimeId` is `EMPTY_DISCRIMINATOR` (right-trim makes it empty) | `"requires non-empty exception/runtime discriminators"` | test/rfc-error-envelope.test.ts:343-350 |
| a surrogate pair and a combining mark round-trip; a 1-byte value is `ODD_UTF16_LENGTH`; `"A\0B"` is `EMBEDDED_NUL`; `[0x00,0xd8]`, `[0x00,0xdc]`, and `[0x00,0xd8,0x41,0x00]` are all `UNPAIRED_SURROGATE` | `"strict UTF-16LE accepts scalar pairs and rejects malformed code units"` | test/rfc-error-envelope.test.ts:352-390 |
| **leading** spaces are preserved (`"  DECLARED"`), trailing padding is trimmed; `maxTextByteLength`, `maxControlByteLength`, `maxTotalTextByteLength` each fire their own reason code | `"preserves leading spaces, trims only right padding, and enforces limits"` | test/rfc-error-envelope.test.ts:392-421 |
| exact-boundary options (`maxFieldCount 5`, `maxControlCount 3`, `maxControlByteLength 4`, `maxTotalControlByteLength 12`) accept three 4-byte controls | `"bounds envelope fields and aggregate unresolved controls before copying"` | test/rfc-error-envelope.test.ts:423-492 |
| decoded text, controls, and provenance are independent of the caller's buffers | `"snapshots text, controls, and provenance independently of caller buffers"` | test/rfc-error-envelope.test.ts:494-516 |
| End placement and field structure are validated | `"validates terminal End placement and field structure"` | test/rfc-error-envelope.test.ts:518-540 |

### Go mapping notes

- **`RfcErrorEnvelopeReasonCode` is the whole error API.** 19 codes on one error type
  (src/protocol/rfc-error-envelope.ts:157-192) — port as exported Go constants of a named type on an
  exported struct error, with `errors.As`. The message strings are diagnostic only.
- **`Object.freeze` is used pervasively and *is asserted by tests***
  (test/rfc-error-envelope.test.ts:91-100). In Go, immutability comes free from returning values and
  unexported slice backing; the freeze assertions have no Go analogue and should not be reproduced as
  runtime checks.
- **`valueHex: string`** (src/protocol/rfc-error-envelope.ts:109, 484) is a deliberate choice to
  avoid handing out a mutable byte view. In Go, a `[]byte` copy would be idiomatic, but note the
  classification at src/protocol/rfc-error-envelope.ts:629 compares
  `control.valueHex !== "00000000"` — port that as a byte comparison, and keep the hex string only if
  something downstream formats it.
- **`Number.isSafeInteger` field-shape validation** (src/protocol/rfc-error-envelope.ts:388-391) is
  guarding against arbitrary JS objects being passed as `fields`. In Go the type system does this;
  `INVALID_FIELD` becomes unreachable and should not be ported.
- **UTF-16LE is not Go's native string encoding.** `decodeStrictUtf16Le`
  (src/protocol/rfc-error-envelope.ts:219-270) must be hand-written against
  `unicode/utf16` + explicit surrogate checks; `utf16.Decode` silently substitutes U+FFFD for
  unpaired surrogates, which would lose the `UNPAIRED_SURROGATE` reason code that
  test/rfc-error-envelope.test.ts:377-389 pins.
- **Right-trim is `/ +$/u` — spaces only**, not `\t`/`\n`/NBSP
  (src/protocol/rfc-error-envelope.ts:269). `strings.TrimRight(s, " ")`, not `TrimSpace`.
- **Overflow-safe accumulation:** the aggregate checks are written as
  `total > maximum - value.byteLength` (src/protocol/rfc-error-envelope.ts:444, 470, 509, 543) rather
  than `total + value > maximum`. Preserve that form in Go or use a wider accumulator.
- No Promise/async/AbortSignal; no `#private`; no `BigInt`.

---

## src/protocol/classic-rfc.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RFC_FUNINT_UNICODE_ROW_LENGTH` | const | `export const RFC_FUNINT_UNICODE_ROW_LENGTH = 402;` | src/protocol/classic-rfc.ts:16 |
| `DEFAULT_MAX_CLASSIC_RFC_TABLE_DECODED_BYTES` | const | `export const DEFAULT_MAX_CLASSIC_RFC_TABLE_DECODED_BYTES =` `  DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH;` | src/protocol/classic-rfc.ts:17-18 |
| `DEFAULT_MAX_CLASSIC_RFC_RESULT_TABLE_DECODED_BYTES` | const | `export const DEFAULT_MAX_CLASSIC_RFC_RESULT_TABLE_DECODED_BYTES =` `  DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH;` | src/protocol/classic-rfc.ts:19-20 |
| `RfcTableHeader` | interface | `readonly declaredRowByteLength: number;` `readonly rowCount: number;` | src/protocol/classic-rfc.ts:22-25 |
| `ClassicRfcScalar` | interface | `readonly name: string;` `readonly value: Buffer;` | src/protocol/classic-rfc.ts:27-30 |
| `ClassicRfcTable` | interface | `name: string; declaredRowByteLength: number; rowByteLength: number; rowEncoding: "flat" \| "structured" \| "mixed" \| "empty"; rowCompression: "none" \| "simple" \| "mixed" \| "empty"; rows: readonly Buffer[]` | src/protocol/classic-rfc.ts:32-44 |
| `ClassicRfcXrfcParameter` | interface | `readonly value: Buffer;` `readonly chunkCount: number;` | src/protocol/classic-rfc.ts:46-50 |
| `ClassicRfcResult` | interface | `requestedOutputs: readonly string[]; scalars: readonly ClassicRfcScalar[]; tables: readonly ClassicRfcTable[]; xrfcParameters: readonly ClassicRfcXrfcParameter[]` | src/protocol/classic-rfc.ts:52-57 |
| `RfcFunintParameter` | interface | `parameterClass, parameterName, tableName, fieldName, exid: string; position, offset, internalLength, decimals: number; defaultValue, parameterText: string; optional: boolean` (all `readonly`) | src/protocol/classic-rfc.ts:59-72 |
| `encodeAbapChar` | func | `export function encodeAbapChar(value: string, characters: number): Buffer` | src/protocol/classic-rfc.ts:81 |
| `decodeAbapChar` | func | `export function decodeAbapChar(value: Uint8Array, expectedCharacters?: number,): string` | src/protocol/classic-rfc.ts:116-119 |
| `decodeAbapFixedChar` | func | `export function decodeAbapFixedChar(value: Uint8Array, expectedCharacters: number,): string` | src/protocol/classic-rfc.ts:124-127 |
| `decodeRfcTableHeader` | func | `export function decodeRfcTableHeader(value: Uint8Array): RfcTableHeader` | src/protocol/classic-rfc.ts:145 |
| `decodeClassicRfcResult` | func | `export function decodeClassicRfcResult(fields: readonly CpicField[],): ClassicRfcResult` | src/protocol/classic-rfc.ts:336-338 |
| `decodeOwnedClassicRfcResult` | func | `export function decodeOwnedClassicRfcResult(fields: readonly CpicField[],): ClassicRfcResult` — marked `@internal` | src/protocol/classic-rfc.ts:348-350, 347 |
| `decodeRfcFunintRow` | func | `export function decodeRfcFunintRow(value: Uint8Array): RfcFunintParameter` | src/protocol/classic-rfc.ts:564 |

### Constants and magic values

| Name | Value (verbatim) | What the source says it means | Citation |
|---|---|---|---|
| `RFC_FUNINT_UNICODE_ROW_LENGTH` | `402` | see the dedicated section below | src/protocol/classic-rfc.ts:16 |
| `DEFAULT_MAX_CLASSIC_RFC_TABLE_DECODED_BYTES` | `DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH` (= `256 * 1024 * 1024`, src/protocol/cpic.ts:22) | per-table decoded-byte cap | src/protocol/classic-rfc.ts:17-18, 245 |
| `DEFAULT_MAX_CLASSIC_RFC_RESULT_TABLE_DECODED_BYTES` | same | whole-result decoded-byte cap across tables | src/protocol/classic-rfc.ts:19-20, 246 |
| `NON_APPLICATION_TAGS` (**not exported**) | `CpicTag.ResponseContext (0x0503), CpicTag.Session (0x0514), CpicTag.Unresolved0420 (0x0420), CpicTag.CallContext (0x0512), CpicTag.Program (0x0130), 0x0667, CpicTag.End (0xffff)` | skipped without interpretation | src/protocol/classic-rfc.ts:158-166, 368 |
| ABAP CHAR length bound | `0x7fff` | `characterCount` guard, error text `"must be an integer in 0..32767"` | src/protocol/classic-rfc.ts:74-78 |

ABAP CHAR encoding (code, src/protocol/classic-rfc.ts:89): `Buffer.from(value.padEnd(characters, " "),
"utf16le")` — fixed width, **space**-padded, UTF-16LE, two bytes per character.

`decodeAbapChar` right-trims spaces (`.replace(/ +$/u, "")`, src/protocol/classic-rfc.ts:120);
`decodeAbapFixedChar` does **not** (src/protocol/classic-rfc.ts:128) — its doc comment says
`"Decode an exact-width character value whose spaces are semantic data."`
(src/protocol/classic-rfc.ts:123).

Table header (code, src/protocol/classic-rfc.ts:145-156): exactly 8 bytes,
`u32BE declaredRowByteLength`, `u32BE rowCount`.

Simple-compression expansion (code, `decodeSimpleCompressedTableRow`,
src/protocol/classic-rfc.ts:168-208): if the encoded row is shorter than the declared width, allocate
`declaredRowByteLength` bytes **filled with the row's last byte**, then copy the encoded bytes over
the front. Equal-width rows are returned as-is. Longer-than-declared and empty rows are rejected.

Row-shape classification (code, src/protocol/classic-rfc.ts:521-530): both kinds present →
`rowEncoding "mixed"`, `rowCompression "mixed"`; only `TableContent` → `"flat"` / `"none"`; only
`TableCompr` → `"structured"` / `"simple"`; no rows → `"empty"` / `"empty"`.
`rowByteLength` is set from the **first** row only (src/protocol/classic-rfc.ts:510-512), defaulting
to `header.declaredRowByteLength` when there are no rows (src/protocol/classic-rfc.ts:471).

xRFC grouping (code, src/protocol/classic-rfc.ts:377-421): an empty `XRfcParameter` opens a group;
one or more non-empty `XRfcData` chunks follow; a second empty `XRfcParameter` closes it. A bare
`XRfcData` outside a group, a non-empty boundary, an empty chunk, a group with no chunks, and a
missing closing boundary are all rejected.

`decodeRfcFunintRow` field layout (code, src/protocol/classic-rfc.ts:576-588), reading the first 402
bytes and ignoring the rest — offsets are cumulative from the read order:
`PARAMCLASS 2B (1 char)`, `PARAMETER 60B (30)`, `TABNAME 60B (30)`, `FIELDNAME 60B (30)`,
`EXID 2B (1)`, `POSITION i32LE`, `OFFSET i32LE`, `INTLENGTH i32LE`, `DECIMALS i32LE`,
`DEFAULT 42B (21)`, `PARAMTEXT 158B (79)`, `OPTIONAL 2B (1)`. Note these are **little-endian**
integers (`readInt32LE`), unlike every other integer in this layer.
`OPTIONAL` must decode to `""` or `"X"` (src/protocol/classic-rfc.ts:590-592).

### Errors

| Message text (verbatim) | Type | Condition | Citation |
|---|---|---|---|
| `` `${field} must be an integer in 0..32767` `` | `RangeError` | `"ABAP CHAR length"` / `"expected ABAP CHAR length"` out of range | src/protocol/classic-rfc.ts:76, 83, 101 |
| `` `ABAP CHAR value of ${value.length} characters does not fit CHAR(${characters})` `` | `RangeError` | encode overflow | src/protocol/classic-rfc.ts:86-87 |
| `"Unicode ABAP CHAR must have an even byte length"` | `RangeError` | odd byte length | src/protocol/classic-rfc.ts:98 |
| `` `Unicode ABAP CHAR must contain exactly ${expectedBytes} bytes; received ${encoded.byteLength}` `` | `RangeError` | fixed-width mismatch | src/protocol/classic-rfc.ts:105-106 |
| `` `${field} is not a valid non-empty RFC parameter name` `` | `Error` | empty name, or any C0/DEL control character; `field` ∈ `"requested output name"`, `"scalar parameter name"`, `"table parameter name"` | src/protocol/classic-rfc.ts:134, 254, 372, 435, 457 |
| `` `classic RFC table header must contain exactly 8 bytes; received ${value.byteLength}` `` | `RangeError` | header ≠ 8 bytes | src/protocol/classic-rfc.ts:148 |
| `` `classic RFC table ${tableName} simple-compressed row ${rowIndex} is empty` `` | `Error` | zero-length compressed row | src/protocol/classic-rfc.ts:180 |
| `` `classic RFC table ${tableName} simple-compressed row ${rowIndex} has ${encodedByteLength} encoded bytes; declared row width is ${declaredRowByteLength}` `` | `Error` | compressed row longer than declared | src/protocol/classic-rfc.ts:185-187 |
| `` `classic RFC table ${tableName} simple-compressed row ${rowIndex} expands to ${declaredRowByteLength} bytes; maximum is ${DEFAULT_MAX_CPIC_FIELD_LENGTH}` `` | `RangeError` | expansion target over the CPIC field cap | src/protocol/classic-rfc.ts:196-199, 291-293 |
| `` `classic RFC result field count exceeds ${DEFAULT_MAX_CPIC_FIELD_COUNT}` `` | `RangeError` | preflight field count | src/protocol/classic-rfc.ts:228 |
| `` `classic RFC result field bytes ${valueBytes} exceed ${maximum}` `` | `RangeError` | preflight aggregate value bytes (accumulated as `BigInt`) | src/protocol/classic-rfc.ts:236 |
| `` `classic RFC table ${name} ${"simple-compressed"\|"uncompressed"} row ${rowCount} is empty` `` | `Error` | preflight empty row | src/protocol/classic-rfc.ts:268-270 |
| `` `classic RFC table ${name} … row ${rowCount} has ${encodedByteLength}${encodedLabel} bytes; declared row width is ${header.declaredRowByteLength}` `` | `Error` | preflight over-wide row | src/protocol/classic-rfc.ts:278-282 |
| `` `classic RFC table ${name} declares ${header.rowCount} rows but found ${rowCount}` `` | `Error` | row count mismatch (preflight and main decode) | src/protocol/classic-rfc.ts:306, 518 |
| `` `classic RFC table ${name} decoded bytes ${tableDecodedBytes} exceed table limit ${tableLimit}` `` | `RangeError` | per-table cap | src/protocol/classic-rfc.ts:311-312 |
| `` `classic RFC decoded table bytes ${resultDecodedBytes} exceed result limit ${resultLimit}` `` | `RangeError` | whole-result cap | src/protocol/classic-rfc.ts:317-318 |
| `"classic RFC response contains xRFC XML data without an opening boundary"` | `Error` | bare `XRfcData` | src/protocol/classic-rfc.ts:378 |
| `"classic RFC xRFC XML opening boundary must be empty"` | `Error` | non-empty `0x3c02` opening | src/protocol/classic-rfc.ts:382 |
| `"classic RFC xRFC XML data chunk must not be empty"` | `Error` | zero-length `0x3c05` | src/protocol/classic-rfc.ts:390 |
| `` `classic RFC xRFC XML parameter exceeds ${DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH} bytes` `` | `RangeError` | aggregated chunk bytes over cap | src/protocol/classic-rfc.ts:398 |
| `"classic RFC xRFC XML boundary contains no data chunk"` | `Error` | opening immediately closed | src/protocol/classic-rfc.ts:405 |
| `"classic RFC xRFC XML parameter lacks its closing boundary"` | `Error` | next field is not `0x3c02` | src/protocol/classic-rfc.ts:409 |
| `"classic RFC xRFC XML closing boundary must be empty"` | `Error` | non-empty closing | src/protocol/classic-rfc.ts:412 |
| `"classic RFC response contains a value without a parameter name"` | `Error` | orphan `ParameterValue` | src/protocol/classic-rfc.ts:424 |
| `"classic RFC response contains a table record without a table name"` | `Error` | orphan `TableHeader`/`TableContent`/`TableCompr` | src/protocol/classic-rfc.ts:431 |
| `` `classic RFC scalar ${name} is not followed by its value` `` | `Error` | missing `ParameterValue` | src/protocol/classic-rfc.ts:438 |
| `` `classic RFC response contains duplicate parameter ${name}` `` | `Error` | repeated scalar or table name | src/protocol/classic-rfc.ts:441, 463 |
| `` `classic RFC table ${name} is not followed by its header` `` | `Error` | missing `TableHeader` | src/protocol/classic-rfc.ts:460 |
| `` `classic RFC table ${name} uncompressed row ${rows.length} is empty` `` | `Error` | zero-length `TableContent` | src/protocol/classic-rfc.ts:499 |
| `` `classic RFC table ${name} uncompressed row ${rows.length} has ${row.byteLength} bytes; declared row width is ${header.declaredRowByteLength}` `` | `Error` | over-wide `TableContent` | src/protocol/classic-rfc.ts:504-506 |
| `` `classic RFC response contains unsupported tag 0x${field.tag.toString(16).padStart(4, "0")}` `` | `Error` | any tag not handled above | src/protocol/classic-rfc.ts:545-547 |
| `` `Unicode RFC_FUNINT row must contain at least ${RFC_FUNINT_UNICODE_ROW_LENGTH} bytes; received ${value.byteLength}` `` | `RangeError` | row shorter than 402 | src/protocol/classic-rfc.ts:566-568 |
| `` `RFC_FUNINT OPTIONAL contains unsupported value ${result.optionalText}` `` | `Error` | `OPTIONAL` neither `""` nor `"X"` | src/protocol/classic-rfc.ts:591 |

`assertUnicodeScalarText` is imported from `../values/unicode-scalar.js`
(src/protocol/classic-rfc.ts:14) and applied to both encode and decode of ABAP CHAR
(src/protocol/classic-rfc.ts:83, 111). That module is out of scope; its error text is not quoted here.

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `"Legacy wire-tag label retained for API compatibility. \`flat\` means an uncompressed TableContent record; \`structured\` means TableCompr. Prefer rowCompression when making protocol decisions."` | src/protocol/classic-rfc.ts:36-40 |
| `"Concatenated UTF-8 XML bytes between one proven 0x3c02 boundary pair."` | src/protocol/classic-rfc.ts:47 |
| `"Decode the eight-byte classic table header observed in Unicode CUT calls. The header exposes a declared row width. Uncompressed row bytes are retained exactly for the metadata-aware consumer; simple-compressed records repeat their last byte to the declared width."` | src/protocol/classic-rfc.ts:139-143 |
| `"Group a decoded function response into lossless classic scalar/table wire values. Unknown application records are rejected so a protocol change cannot be mistaken for a successful, partially decoded call. This syntax-layer API deliberately retains a bounded short uncompressed row because the raw field stream has no structure metadata with which to decide whether a named wire owner permits it. \`decodeClassicRfcInvocationResult\` is the public-call semantic boundary and rejects every short ordinary row except an explicitly evidence-owned metadata case."` | src/protocol/classic-rfc.ts:326-334 |
| `"Decode values already owned by the current CPIC session without retaining a second full reply snapshot. The returned buffers may borrow \`fields\` and must therefore be consumed synchronously before those fields are released."` | src/protocol/classic-rfc.ts:342-346 |
| `"Decode an exact-width character value whose spaces are semantic data."` | src/protocol/classic-rfc.ts:123 |
| the `decodeRfcFunintRow` block comment — quoted in full in the next section | src/protocol/classic-rfc.ts:554-563 |

### Wire facts asserted by tests

Only one test file in scope imports this module (`test/classic-xrfc-protocol.test.ts:14`), and it
exercises **only** the xRFC grouping path.

| What the test asserts | Test name (verbatim) | Citation |
|---|---|---|
| two `XRfcData` chunks between one boundary pair are concatenated into `"<EXPORT_TAB><item></item></EXPORT_TAB>"` with `chunkCount === 2`, and the result survives zero-filling the source buffers (`decodeClassicRfcResult` **copies**) | `"groups captured response chunks and clones their XML bytes"` | test/classic-xrfc-protocol.test.ts:161-186 |
| two consecutive boundary pairs produce two independent parameters `["<A></A>", "<B></B>"]`; six malformed grammars all throw — bare `XRfcData`; non-empty opening boundary; boundary pair with no chunk; empty chunk; missing closing boundary; non-empty closing boundary | `"preserves multiple xRFC envelopes and rejects malformed boundary grammar"` | test/classic-xrfc-protocol.test.ts:188-239 |

**Untested in scope:** `encodeAbapChar`, `decodeAbapChar`, `decodeAbapFixedChar`,
`decodeRfcTableHeader`, `decodeOwnedClassicRfcResult`, `decodeRfcFunintRow`, and the entire scalar
and table decoding path of `decodeClassicRfcResult` have **no test in the files listed in this
inventory's scope** — a `grep -rln` over `test/` for those symbols returns no match. The porter has
no oracle for the table-compression and RFC_FUNINT logic beyond the source itself.

### Go mapping notes

- **`BigInt` is used for the preflight accumulators** (src/protocol/classic-rfc.ts:231-239, 245-246,
  296-320). This is the only `BigInt` in the layer. In Go, `uint64` covers the same range with no
  ceremony; the limits are `256 * 1024 * 1024`, far below overflow.
- **Borrow-vs-copy is an explicit, exported distinction.** `decodeClassicRfcResult` copies;
  `decodeOwnedClassicRfcResult` borrows and is marked `@internal`
  (src/protocol/classic-rfc.ts:336-352). `borrowedWireBuffer` constructs a `Buffer` over the same
  `ArrayBuffer` (src/protocol/classic-rfc.ts:210-213). In Go, sub-slicing borrows by default, so the
  **copying** variant is the one that needs explicit work. Keep both, keep the borrow one unexported,
  and preserve the "consume synchronously" contract in the doc comment.
- **`decodeOwnedClassicRfcResult` should be unexported in Go** — it is `@internal`
  (src/protocol/classic-rfc.ts:347) and its buffers alias caller memory.
- **`rowEncoding` is dead weight.** The interface comment calls it
  `"Legacy wire-tag label retained for API compatibility"` and says
  `"Prefer rowCompression when making protocol decisions"` (src/protocol/classic-rfc.ts:36-40).
  The two fields are always set together (src/protocol/classic-rfc.ts:521-530) and carry identical
  information. A Go port has no compatibility obligation and should carry only `RowCompression`.
- **UTF-16LE again** (src/protocol/classic-rfc.ts:89, 110), plus `assertUnicodeScalarText` from the
  values layer. Same caveat as rfc-error-envelope: Go's `utf16.Decode` substitutes U+FFFD rather than
  erroring.
- **Little-endian integers in `decodeRfcFunintRow`** (`readInt32LE`, src/protocol/classic-rfc.ts:581-584)
  are the exception in a big-endian layer. Easy to get wrong.
- `padEnd(characters, " ")` operates on **UTF-16 code units**, not scalars
  (src/protocol/classic-rfc.ts:89); combined with `value.length > characters`
  (src/protocol/classic-rfc.ts:85), a non-BMP character counts as two. Go's `utf16.Encode` gives the
  same unit count — count units, not runes.
- All errors are anonymous `Error`/`RangeError` with formatted messages; no error class is exported
  from this file. INFERRED: a Go port would want at least one sentinel to distinguish
  "table row width violation" from "unknown tag", since these mean different things operationally,
  but the TypeScript source does not make that distinction and I am not asserting it does.

---

## One specific question to answer while reading classic-rfc.ts

**Definition, verbatim:**

    export const RFC_FUNINT_UNICODE_ROW_LENGTH = 402;

— src/protocol/classic-rfc.ts:16. There is **no comment attached to the declaration itself**; line 16
sits between the import block and the next `export const` with no annotation.

**The comment explaining the row length**, verbatim, is on `decodeRfcFunintRow`
(src/protocol/classic-rfc.ts:554-563):

    /**
     * Decode one Unicode RFC_FUNINT row returned by metadata bootstrap.
     *
     * The row width is a property of the peer's release, not of the wire format:
     * later releases append fields to RFC_FUNINT, and one profile is already
     * evidenced declaring a 404-byte row. Bound the row below by the stable prefix
     * this decoder consumes and ignore anything appended after it, exactly as
     * `decodeDdIfDfiesRow` does for DFIES. A short row is still refused - completing
     * one with ABAP initial bytes would invent values the peer never sent.
     */

**What the code actually does with it** (code, src/protocol/classic-rfc.ts:565-574):

    if (value.byteLength < RFC_FUNINT_UNICODE_ROW_LENGTH) {
      throw new RangeError(
        `Unicode RFC_FUNINT row must contain at least ${RFC_FUNINT_UNICODE_ROW_LENGTH} ` +
          `bytes; received ${value.byteLength}`,
      );
    }
    const reader = new CheckedByteReader(
      value.subarray(0, RFC_FUNINT_UNICODE_ROW_LENGTH),
      "RFC_FUNINT row",
    );

So the constant is used as a **lower bound and a read window**, not as an equality pin:

- a row of exactly 402 bytes is accepted;
- a row **longer** than 402 bytes is accepted and everything past byte 402 is discarded by the
  `subarray(0, 402)`;
- a row **shorter** than 402 bytes is refused, with the comment's stated reason:
  `"completing one with ABAP initial bytes would invent values the peer never sent."`

The 402 bytes are fully consumed by the twelve fields listed in the layout table above
(2 + 60 + 60 + 60 + 2 + 4 + 4 + 4 + 4 + 42 + 158 + 2 = 402), and `reader.finish()`
(src/protocol/classic-rfc.ts:589) asserts the window is exactly consumed.

**Relevance to the recurring bug class.** The comment names a second observation —
`"one profile is already evidenced declaring a 404-byte row"` (src/protocol/classic-rfc.ts:558-559) —
and the code accommodates it by ignoring the trailing bytes rather than pinning 404 or rejecting it.
This is the *inverse* of the recurring bug: the row length is deliberately **not** treated as a rule
derived from one observation.

**But the constraint does not survive intact at the metadata boundary.** A `grep -rn` over `src/`
shows `src/metadata/rfc-function-interface.ts:2` imports `RFC_FUNINT_UNICODE_ROW_LENGTH`, and
`src/metadata/rfc-function-interface.ts:80` reads
`if (params.rowByteLength < RFC_FUNINT_UNICODE_ROW_LENGTH) {` with an error message at line 83
containing `` `expected at least ${RFC_FUNINT_UNICODE_ROW_LENGTH}` ``. (I did not read that file —
`src/metadata/` is outside this inventory's scope — so I am quoting only the grep-matched lines.)
That is consistent with the lower-bound reading, and it confirms the premise: the metadata layer
carries the constraint symbolically and never states `402` itself. **A Go port must export the
constant from the protocol package, not re-derive it in the metadata package**, or the two layers
can drift.

INFERRED: nothing in the source explains *why* 402 specifically, i.e. which release's RFC_FUNINT
layout the twelve-field prefix corresponds to. The field names (`PARAMCLASS`, `PARAMETER`, `TABNAME`,
`FIELDNAME`, `EXID`, `POSITION`, `OFFSET`, `INTLENGTH`, `DECIMALS`, `DEFAULT`, `PARAMTEXT`,
`OPTIONAL`) and their widths are the only evidence in the file.

---

## Candidate conformance vectors

Pure byte-in / byte-out cases from the in-scope tests, suitable for a language-neutral vector file.
Input and expected output are verbatim from the cited test.

### RFCPRO field headers (test/rfcpro.test.ts)

| Input | Expected output (hex) | Citation |
|---|---|---|
| `encodeRfcProFieldHeader(0x0203, 0)` | `02030000` | test/rfcpro.test.ts:15-18 |
| `encodeRfcProFieldHeader(0x0203, 65534)` | `0203fffe` | test/rfcpro.test.ts:19-25 |
| `encodeRfcProFieldHeader(0x0203, 65535)` | `0203ffff0000ffff` | test/rfcpro.test.ts:26-29 |
| `encodeRfcProFieldHeader(0x0203, 65536)` | `0203ffff00010000` | test/rfcpro.test.ts:30-33 |
| `encodeRfcProFieldHeader(0x0203, 0x7fffffff)` | `0203ffff7fffffff` | test/rfcpro.test.ts:34-39 |
| decode `0203fffe` | `{tag: 0x0203, length: 65534, encoding: "compact", bytesConsumed: 4}` | test/rfcpro.test.ts:43-48 |
| decode `0203ffff0000ffff` | `{tag: 0x0203, length: 65535, encoding: "extended", bytesConsumed: 8}` | test/rfcpro.test.ts:49-57 |
| decode `0203ffff00010000aabb` | `{tag: 0x0203, length: 65536, encoding: "extended", bytesConsumed: 8}` (trailing bytes ignored) | test/rfcpro.test.ts:58-66 |
| decode `0203ffff0000fffe` | `{tag: 0x0203, length: 65534, encoding: "extended", bytesConsumed: 8}` (over-long encoding accepted) | test/rfcpro.test.ts:67-75 |
| decode `0203ffffffffffff` | error `extended length -1 is negative` | test/rfcpro.test.ts:174-177 |
| decode `0203ffff80000000` | error `extended length -2147483648 is negative` | test/rfcpro.test.ts:178-181 |

### CPIC field chain (test/cpic.test.ts)

| Input | Expected output (hex) | Citation |
|---|---|---|
| `encodeCpicFieldChain(0x0514, [{0x0114, "001"}, {0x0111, "USER"}, {0xffff, <empty>}])` | `051401140003303031011401110004555345520111ffff0000` | test/cpic.test.ts:24-34 |
| `encodeCpicFieldChain(0x0201, [{0x0203, 65534 bytes of 0x11}])` first 6 bytes | `02010203fffe` | test/cpic.test.ts:112-118 |
| `encodeCpicFieldChain(0x0201, [{0x0203, 65535 bytes of 0x22}])` first 10 bytes | `02010203ffff0000ffff` | test/cpic.test.ts:124-133 |
| `encodeCpicFieldChain(0x0201, [{0x0203, 65536 bytes of 0x33}, {0xffff, <empty>}])` first 10 bytes | `02010203ffff00010000` | test/cpic.test.ts:139-149 |
| decode `05140114000330303100010111000455534552` with initial tag `0x0514` | error `expected previous tag 0x0114 … received 0x0001` | test/cpic.test.ts:94-102 |
| decode `051401140004303031` with initial tag `0x0514` | error matching `/need 4 bytes/` | test/cpic.test.ts:104-108 |

### CPIC request framing (test/cpic.test.ts)

| Input | Expected output | Citation |
|---|---|---|
| initial logon request with `client "001"`, `user "RFCUSR"`, `password "x"*25`, `language "E"`, `clientAddress "127.0.0.1"`, `partnerHostName "host.example.test"`, `destination "127.0.0.1"`, `programName "open-rfc01"`, `sessionId 16×0x5a`, `passwordSeed 0x12345678` | 296 bytes; first 18 = `d9c6c3f0f0f0f0f0f0f0f0f0010100080301`; last 10 = `ffff0000012000008500` | test/cpic.test.ts:469-488 |
| `encodeCpicFunctionRequest({functionName: "RFC_PING", sessionId: 16×0x5a})` | 129 bytes; first 12 = `010100080301010504010003`; last 10 = `ffff0000007900008500` | test/cpic.test.ts:1598-1608 |
| CUT metadata request (`RFC_GET_FUNCTION_INTERFACE`, 5 requested outputs, 2 imports, exact inputs at test/cpic.test.ts:1659-1675) | 408 bytes; first 4 = `05020000`; last 10 = `ffff0000019000008500` | test/cpic.test.ts:1658-1679 |
| CUT table `{name:"ROWS", rowByteLength:4, rows:[01020304, 05060708]}` | header value = `0000000400000002`; two `0x0304` fields with values `01020304` and `05060708` | test/cpic.test.ts:1727-1762 |
| CUT request with a 27 926-byte import | framing `{mode:"compact", applicationDataLength:28000, finalSapParameterLength:8}` | test/cpic.test.ts:351-363 |
| CUT request with a 27 927-byte import | framing `{mode:"streamed", applicationDataLength:28001, finalSapParameterLength:0}` | test/cpic.test.ts:355-368 |

### Password scramble (test/password-scramble-equivalence.test.ts, test/cpic.test.ts)

| Input (password, seed) | Expected output (hex) | Citation |
|---|---|---|
| `("", 0)` | `00000000` | test/password-scramble-equivalence.test.ts:89-90 |
| `("AB", 0)` | `00000000b150` | test/password-scramble-equivalence.test.ts:91-92 |
| `("secret", 0x15)` | `150000008981dc9b914e` | test/password-scramble-equivalence.test.ts:93-94 |
| `("x"×40, 0x15)` | `15000000829cc7918c42d277b89294e3e555310d2a1dab67283465f464236b8` + `9eee52cab0e1299ba2cca2145` | test/password-scramble-equivalence.test.ts:95-100 |
| `("x"×40, 0xffffffff)` | `ffffffff157c7261c7229cf431269cc2656bab279b1413adb1a62a23123a4d3d` + `ef405bbafd707c3f2c82d487` | test/password-scramble-equivalence.test.ts:101-107 |
| `("~", 0xffffffff)` | `ffffffff13` | test/password-scramble-equivalence.test.ts:108 |
| `("secret", 0x5ae0b7a3)` | `a3b7e05a048eaa683470` | test/cpic.test.ts:447-452 |
| 21 592-vector sweep, `0xff`-separated, SHA-256 | `f3e0b74e48219b80e926e4ff2684c045e4fc93015eca9221a5ddb6a50cd8ff82` | test/password-scramble-equivalence.test.ts:57-85 |

### Message server (test/message-server.test.ts)

Four complete records are pinned as hex literals and compared with `deepEqual`. They are the single
best cross-language vector set in this layer. Reproduce them by value from the cited lines rather
than re-deriving them:

| Vector | Length | Citation |
|---|---|---|
| `LOGIN_REQUEST` = `encodeMessageServerLoginRequest()` | 110 bytes | test/message-server.test.ts:15-18 (fixture), 58 (assertion) |
| `LOGIN_RESPONSE` — decodes without error | 110 bytes | test/message-server.test.ts:20-23, 61 |
| `LOGOUT_REQUEST` = `encodeMessageServerLogoutRequest()` | 110 bytes | test/message-server.test.ts:25-28, 60 |
| `GROUP_REQUEST` = `encodeMessageServerRfcGroupRequest("RFC_GROUP")` | 206 bytes | test/message-server.test.ts:30-33, 59 |
| `GROUP_RESPONSE` → `{applicationServerHost:"app.example.test", dispatcherPort:3200, gatewayPort:3300, gatewayService:"sapgw00", systemNumber:"00"}` | 223 bytes | test/message-server.test.ts:35-38, 64-75 |

### Gateway (test/gateway.test.ts)

| Input | Expected output | Citation |
|---|---|---|
| the fixture record at test/gateway.test.ts:10-28 | 64 bytes; `[0]=2`, `[1]=3`; bytes 10..20 = `"sapgw00\0\0\0"`; bytes 24..29 = five zeros; `[29]=6`; bytes 46..54 = eight `0x20`; full round-trip equality | test/gateway.test.ts:30-44 |

### APPC (test/appc.test.ts)

| Input | Expected output | Citation |
|---|---|---|
| `encodeAppcExtendedInitializeOptions(<fixture at test/appc.test.ts:27-42>)` | 341 bytes; `[0]=1`; bytes 2..10 = `4350494300000000`; `readInt32BE(46) = -2`; `readInt32BE(50) = -2`; round-trips | test/appc.test.ts:111-122 |
| `encodeAppcPartnerLogicalUnitInfo({logicalUnitName:"NWRFC", partnerHostAddress: 00000000000000000000ffff7f000001, communicationIndex: 0xffff, connectionIndex: 6})` | 32 bytes; bytes 0..8 = `"NWRFC   "`; `readUInt32BE(8) = 5` | test/appc.test.ts:87-109 |
| `encodeAppcDataRecord({conversationId:"CONV0001", communicationIndex:0xffff, connectionIndex:6, data:"payload"})` | 87 bytes; `sapParameterLength=8`, `info=5`, `vector=0x0c`; `readUInt16BE(76)=0xffff`; `readUInt16BE(78)=6`; payload at offset 80 | test/appc.test.ts:353-370 |
| same with `isFinal: false` | `vector = 0x08` | test/appc.test.ts:372-381 |
| `encodeAppcInitializeParameters({clientIdentifier:"NWRFC", options:<fixture>})` | 373 bytes; first 32 = `"NWRFC"` + 27 spaces | test/appc.test.ts:124-135 |
| `encodeAppcPartnerLogicalUnitParameters({longLogicalUnitName:"127.0.0.1", partnerHostAddress: 16 zero bytes})` | 144 bytes; bytes 0..9 = `"127.0.0.1"`; bytes 9..128 all `0x20` | test/appc.test.ts:137-147 |

### RFC error envelope (test/rfc-error-envelope.test.ts)

These are field-list-in / classification-out rather than byte-in / byte-out, but they are
deterministic and language-neutral:

| Input | Expected output | Citation |
|---|---|---|
| `[{0x0503, <empty>}, {0x0201, "RESULT" utf16le}, {0x0420, 4 zero bytes}, {0xffff, <empty>}]` | `outcome "success"`, `successControl "zeroControl"`, `unresolved0420 = [{tag:0x0420, ordinal:2, byteLength:4, valueHex:"00000000"}]` | test/rfc-error-envelope.test.ts:195-211 |
| same with `0x0420` = `00000001`, or 3 bytes, or absent, or duplicated | reason code `RFC_ERROR_ENVELOPE_UNRESOLVED_SUCCESS_CONTROL` | test/rfc-error-envelope.test.ts:213-226 |
| `[{0x0401, 1 byte 0x41}, {0xffff, <empty>}]` | reason code `RFC_ERROR_ENVELOPE_ODD_UTF16_LENGTH` | test/rfc-error-envelope.test.ts:363-369 |
| `[{0x0401, [0x00,0xd8]}, …]`, `[{0x0401, [0x00,0xdc]}, …]`, `[{0x0401, [0x00,0xd8,0x41,0x00]}, …]` | reason code `RFC_ERROR_ENVELOPE_UNPAIRED_SURROGATE` | test/rfc-error-envelope.test.ts:377-389 |

---

## Open questions for the porter

1. **`src/protocol/bytes.ts` is not in this inventory but every file depends on it.** All eight files
   import `CheckedByteReader`/`CheckedByteWriter` and the geometry helpers
   (src/protocol/appc.ts:1-6, src/protocol/rfcpro.ts:1, src/protocol/gateway.ts:3,
   src/protocol/message-server.ts:1, src/protocol/classic-rfc.ts:1-6, src/protocol/cpic.ts:3-5).
   Its exact error message formats (`` `${context}.${field}: need N bytes at offset M; K remain` ``,
   inferred from src/protocol/rfcpro.ts:85 which reproduces that shape by hand) and the semantics of
   `reader.remaining` (relied on at src/protocol/message-server.ts:420) and `reader.finish()` need
   their own inventory before any of this is portable.
2. **What is the word at APPC operation-info offset 2?** The source says only that it
   `"is a receive-buffer capacity and must not be used to frame the application payload"`
   (src/protocol/appc.ts:963-964), and the appc test writes `34_048` there
   (test/appc.test.ts:57). `34_048` = `0x8500`, which is also the default `maximumRfcPacketSize` in
   cpic.ts (src/protocol/cpic.ts:1031, 1519). Whether that correspondence is meaningful or
   coincidental is not stated anywhere in the source. **I am not asserting a relationship.**
3. **`classic-rfc.ts` has almost no test coverage in scope.** `decodeRfcFunintRow`,
   `decodeRfcTableHeader`, `encodeAbapChar`/`decodeAbapChar`/`decodeAbapFixedChar`,
   `decodeOwnedClassicRfcResult`, and the whole scalar/table branch of `decodeClassicRfcResult` are
   not exercised by any file listed in this inventory's scope. Either coverage exists in files
   outside the scope I was given (`src/client/classic-invocation.ts` and `src/values/`
   appear in a `grep` for these symbols), or the Go port has no behavioural oracle for the
   simple-compression expansion rule at src/protocol/classic-rfc.ts:202-207.
4. **`decodeClassicRfcInvocationResult` is named as the semantic boundary but does not live here.**
   The comment at src/protocol/classic-rfc.ts:332-334 says it
   `"is the public-call semantic boundary and rejects every short ordinary row except an explicitly
   evidence-owned metadata case."` That function is not exported from `src/protocol/`; the short-row
   policy split between the two layers is invisible from this inventory alone.
5. **The `0x0450` control and the `0x0020`/`0x0021`/`0x0126`/`0x0150`/`0x0151`/`0x0152`/`0x0451`/
   `0x0452`/`0x0453`/`0x0667` grammar coordinates have widths but no names or meanings** anywhere in
   cpic.ts (src/protocol/cpic.ts:722-784). Only `0x0450` gets a comment
   (`"Unexported six-byte successful-logon control observed on S/4HANA 2023."`,
   src/protocol/cpic.ts:88). A Go port should carry them as opaque numbered coordinates with the
   same pinned/bounded distinction, and must not invent names.
6. **Is the `INITIAL_CPIC_SIGNATURE` value `d9c6c3f0f0f0f0f0f0f0f0f0` meaningful?** The source gives
   it no comment (src/protocol/cpic.ts:654). It is required byte-for-byte by
   `decodeCpicInitialLogonRequest` (src/protocol/cpic.ts:1137-1139). Purpose not stated in source.
7. **The `filler` / `padding` / `reserved` fields differ in strictness between files.**
   gateway.ts rejects non-zero reserved bytes (src/protocol/gateway.ts:150, 155, 174); appc.ts
   rejects non-zero `reserved1..reserved6` in extended init options (src/protocol/appc.ts:671, 687,
   693) but writes and ignores `padding` in the common header (src/protocol/appc.ts:1528, 1712).
   A Go port should preserve this asymmetry exactly rather than normalizing it — but the source does
   not say why the two differ.
