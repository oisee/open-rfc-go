# Discovery 0002 — the wire constants, explained

Every "magic" byte the server track uses, what it means, and how it was learned
(all reverse-engineered from live A4H captures — SAP_BASIS 793 — since classic
RFC has no public spec). Companion to [0001](0001-live-type3-server.md). Where a
constant appears in code it now carries a short comment; this is the full table.

## Framing layers

| Constant | Where | Meaning |
|---|---|---|
| 4-byte big-endian length prefix | NI record | `internal/ni` frames every record as `len(4 BE) ‖ payload` |
| `0x06` | APPC record byte 0 | APPC protocol version (all records start with it) |
| `0x03` | APPC record byte 1 | function **CPIC-init** (the logon record) |
| `0xcb` | APPC record byte 1 | function **F_SAP_SEND** (carries CPIC/CUT data) |
| `0x01`/`0x05`/`0x07`/`0x08`/`0x09`/`0x0b` | APPC byte 1 | Initialize / Allocate / SendData / AsyncSend / Receive / Deallocate |

### APPC F_SAP_SEND record header (80 bytes)

| Offset | Field | Notes |
|---|---|---|
| 0 | protocol `0x06` | |
| 1 | function `0xcb` | |
| 4:6 | **uid** (BE) | mirrored back on the response |
| 6:8 | gatewayId (BE) | server sets `1` on its records |
| 40:48 | **conversation id** (8 ASCII digits) | client-generated; the server must echo it |
| 48:80 | operation info (32 B) | comm/connection index, data length |
| 80… | payload (the CUT message) | |

### Gateway normal-client record (64 bytes)

The client sends it; the server replies with the **same 64 bytes but two changed**:

| Offset | Client → Server |
|---|---|
| 29 | `0x0e` → **`0x0f`** (protocol level accepted) |
| 55 | `0xcb` → **`0xfb`** (capability bits acknowledged) |

Echoing verbatim is rejected; these two bytes are the capability ack.

## CUT (function call/result) envelope

| Constant | Meaning |
|---|---|
| `05 02 00 00` | CUT **request** prefix |
| `05 00 00 00` | CUT **response** prefix |
| `ff ff 00 00 ff ff` | the field-chain **End** marker (2-byte `ff ff` trailer follows) |

### RFCPRO / CPIC field tags (2-byte BE, then 2-byte length, then value)

| Tag | Name | Tag | Name |
|---|---|---|---|
| `0x0102` | Function name | `0x0420` | success control (`00000000` = ok) |
| `0x000b` | Kernel release | `0x0500` | response start |
| `0x0201`/`0x0203` | scalar name / value | `0x0503` | response context |
| `0x0205` | requested output (echo **exports only**) | `0x0512` | call context |
| `0x0301`/`0x0302` | table name / header | `0x0514` | session RFC GUID (16 B, **byte-swapped** in responses) |
| `0x0303`/`0x0304` | table rows (uncompressed / compressed) | `0x0130` | implementing program (80 B, UTF-16LE) |
| `0x0401` | exception key | `0x0667` | 8-byte perf metric (varies per call) |

Values are UTF-16LE; a table header is `rowByteLength(4 BE) ‖ rowCount(4 BE)`.

## RFC connection GUID (16 bytes)

Assigned per connection (since 46D); `RSRFCPIN` aborts with
`RFC_INVALID_UUID_DETECTED` on a mismatch.

| Constant | Meaning |
|---|---|
| node suffix `e1 00 00 00 ac 11 00 03` | last 8 bytes = RFC node id + host IP (172.17.0.3); locates a GUID in a frame |
| **structured swap** | requests carry the GUID one way; server records swap the first three components (4+2+2 bytes reversed, last 2+6 unchanged) — `swapRFCGUID` |

## S/4HANA classic extension tags

Present only when the peer is S/4HANA. On top of the classic tags above:

| Tag | Meaning |
|---|---|
| `0x0104` | 236-byte trace/metadata block; **byte [205] = swapped-GUID[0] − 2**, byte [222] is a per-call counter, rest constant |
| `0x0331` | S4 table id / marker (4 B) |
| `0x0333` + `0x0334` | S4-native table: descriptor (`kind,id,rowCount`) + rows concatenated at fixed width |
| `0x0335` | S4 mixed-table descriptor, followed by classic `0x0302` header + `0x0304` rows |
| `0x0336` | S4 per-table trailer (4 B) |

## Fast serialization (a distinct encoding, negotiated at logon)

| Constant | Meaning |
|---|---|
| `0x5001` | fast-ser parameter container (holds a scalar/structure) |
| `0x43 <len:1> 0x80 <value>` | a **character field** inside 0x5001 (value is ASCII) — `internal/fastser` |

The serializer (Classic / basXML / Fast) is chosen by the **server's logon-accept**,
not the SM59 destination setting; an accept borrowed from a fast-ser session makes
the client send fast-ser parameters regardless of its Classic preference.

## NI keepalive

| Constant | Meaning |
|---|---|
| `4e 49 5f 50 49 4e 47 00` | `NI_PING\0` — answer with… |
| `4e 49 5f 50 4f 4e 47 00` | `NI_PONG\0` |

## How each was learned

By capturing live traffic (`cmd/rfc-sniffer` / `cmd/rfc-lab`), decoding it with
this project's own codecs, and diffing sessions to separate constant structure
from per-session tokens. Provenance for each is the capture and the diff recorded
in [0001](0001-live-type3-server.md); none of these values comes from SAP
documentation.

## Where the constants live in code

Named, commented constants central to the server track:

| File | Constants |
|---|---|
| `internal/rfcserver/wire_constants.go` | `appcProtocol` 0x06, `appcInit` 0x03, `appcFSapSend` 0xcb, `appcUIDOffset` 4, `appcConvOffset` 40, `appcHeaderLen` 80, `gatewayRecordLen` 64, `gatewayAckOffset1` 29 / `gatewayAckLevel` 0x0f, `gatewayAckOffset2` 55 / `gatewayAckCaps` 0xfb, `cutReqTag0/1` 0x05/0x02, `cutRespT1` 0x00, `niKeepaliveLen` 8, `initMinLen` 200, `cutPrefixLen` 4 |
| `internal/rfcserver/content.go` | `niPing`/`niPong` ("NI_PING\0"/"NI_PONG\0"); the `isInit`/`isFSapSend`/`isFuncRequest` classifiers |
| `internal/rfcserver/replay.go` | `rfcGUIDNodeSuffix` `e1000000ac110003`, `swapRFCGUID` (structured swap), `convIDOf`/`withConvID` (conv id at offset 40) |
| `internal/rfcserver/response.go` | CUT prefixes 05 00 00 00 / trailer ff ff; response tags 0x0503/0x0514/0x0420/0x0512/0x0205/0x0201/0x0203/0x0130/0x0667; `0x0104[205] = swapRFCGUID(guid)[0] - 2` |
| `internal/rfcserver/s4_envelope.go` | baked `0x0667` metric and `0x0104` metadata templates |
| `internal/rfcserver/serve_smart.go` / `serve_generate.go` / `serve_conscious.go` | gateway ack bytes, APPC offsets, the ping dance |
| `internal/fastser/fastser.go` | `charTag` 0x43, `charFlag` 0x80 (the fast-ser character field); single-byte length cap 255 |
| `internal/cpic/cpic.go` (ported) | the RFCPRO tag constants the response tags above reference (`TagSession` 0x0514, `TagCallContext` 0x0512, `TagRequestedOutput` 0x0205, …) |

The S4 extension tags (0x0104, 0x0331/0333/0334/0335/0336) are tolerated in
`internal/cpic/function.go` (envelope) and parsed in
`internal/classicrfc/classicrfc.go` (data); both are annotated there.

Provenance for every value is the live-capture diff recorded in
[0001](0001-live-type3-server.md) and the `/tmp/gold-*.jsonl` captures — none of
it is from SAP documentation.
