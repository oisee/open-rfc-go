# Logon and frame-exchange reference

The single document a new contributor reads to understand how a classic-RFC /
CPIC logon happens and how frames are exchanged, across every mode this project
has probed. It consolidates the discovery notes, the private byte-analyses, and
the codecs in `internal/` so the picture does not have to be re-derived. Classic
RFC has **no public specification**; almost everything below was learned by
capturing live traffic against real A4H systems (SAP_BASIS 758 / 793) and
decoding it with this repo's own codecs. Where a fact comes from elsewhere it is
tagged.

## How to read this document

Every non-obvious fact carries its origin:

- **[capture]** — observed in one of our own live wire dumps (clean-room; no
  vendor code or spec involved).
- **[our-code]** — a codec or constant in this repo, i.e. a definition we rely on
  in running code.
- **[doc]** — public SAP documentation or an SAP KB / Note.
- **[gpl-ref]** — a fact *read for reference only* from a GPL source (pysap,
  Wireshark dissectors) and restated here in our own words. No code was copied.
- **[inferred]** — reasoned from the above, not directly observed.

"Clean-room" means: derived from our captures and our code, not from reading
SAP's implementation. The field ids, offsets, encodings, and procedures here are
that kind of knowledge. **No credential value, real ticket string, decoded
ticket bytes, or password appears in this document** — only structure.

A companion glossary is in [`../glossary.md`](../glossary.md); the layer-by-layer
primer is [`../protocol-primer.md`](../protocol-primer.md); the wire constants are
catalogued in [`0002-wire-constants.md`](0002-wire-constants.md).

---

## 1. Transports — where a logon can happen at all

RFC logon is not one thing on one port. There are five distinct surfaces; we have
proven some live and only reasoned about others.

| Surface | Port | What it is | Status |
|---|---|---|---|
| **Direct type-3 client → AS** | gateway `3300 + nn` (`sapgwNN`) | An ABAP or external client dials the gateway, does the gateway handshake, then sends the initial CPIC logon (`0x03`) with credentials. This is the normal RFC path. | password logon **proven live** [capture]; ticket logon **rejected / not characterised** (see §5) |
| **Registered server (type-T)** | gateway `3300 + nn` | An external program registers a Program ID at the gateway; an ABAP client's call (and its logon, incl. a forwarded ticket) is *relayed* by the gateway to that program. | **proven live** end to end with SAP's own `rfcexec` as the server, sniffed on both legs [capture] |
| **Dispatcher** | `3200 + nn` (`sapdpNN`) | The work-process dispatcher. **Classic RFC does not connect here** — pinning RFC to the `32NN` range is a documented recurring bug. Listed only to say: not this port. | n/a [our-code] |
| **HTTP / ADT** | `8000` / `44300` (type-H/G/W) | SM59 type-H "HTTP Connection to ABAP System"; ADT REST; the SOAP RFC endpoint. A ticket authenticates these as a header/cookie. | HTTP catcher **proven** for what a type-H destination sends [capture]; ticket-authenticated ADT + SOAP RFC endpoint verified live the same day (reasoned/verified, no wire dump kept) |
| **SOAP RFC endpoint** | over HTTP(S) | `/sap/bc/soap/rfc` — exposes the remote-enabled function surface over SOAP, authenticated by a cookie/ticket. | reasoned + verified live; the design note that would detail it (`docs/design/http-only-systems.md`) is **referenced by the task but is not present in the repo** — see the reconciliation notes at the end |

The gateway port `3300 + nn` is the one that matters for classic RFC: the
handshake, the APPC conversation, and the CPIC logon all ride it. Service name
`sapgwNN`; the dispatcher `sapdpNN` on `3200 + nn` is a different thing. [our-code][doc]

Our test rig (`cmd/rfc-lab`) binds several ports on one host and the SM59
connection type / system number selects the mode: a transparent sniffer in front
of a real system (to capture), or our own Go server answering directly. [capture]

---

## 2. The frame / function-code map

A frame is one NI record: a 4-byte big-endian length prefix followed by that many
payload bytes (`internal/ni`). Inside, the first byte is the layer/version and
the second is the function code — **but the meaning of the function-code byte
depends on the framing layer**, and several byte values are overloaded between
the gateway layer and the APPC layer. The table below is assembled from our
captures (`.private/gold/cap-registered-server.jsonl`, `cap-leg2.jsonl`) and the
codecs in `internal/appc` / `internal/gateway`.

| Byte | Layer | Direction (typical) | Size seen | Meaning |
|---|---|---|---|---|
| `0x03` | **overloaded** | C→S | 64 B (gateway) / 1.7–2.8 KB (APPC) | As a **gateway** request type: `GW_NORMAL_CLIENT` (request type 3) [our-code]. As an **APPC** function byte: the **initial CPIC logon / relayed logon** record. In the registered-server relay the client's whole logon arrives server-side as `0x03`. [capture] |
| `0x0b` | gateway / APPC | C→S | 64–80 B | In the registered-server leg-1 capture, the **NI/gateway handshake** frame. As an APPC function code `0x0b` is `F_DEALLOCATE` [our-code]; the 64-byte handshake use is the gateway layer, not APPC — a genuine overloaded byte. [capture] |
| `0x68` | gateway | C→S | 512 B | The **`GW_NORMAL_CLIENT` region** the gateway exchanges in the registered-server flow (a larger gateway bookkeeping block than the 64-byte record). [capture] |
| `0xca` | APPC (gateway-relayed) | C→S request; S→C accept | 453 B (ALLOCATE); 125 B (accept) | **ALLOCATE** (the routing request naming the destination / program id / caller) and, on the **client-facing leg**, the **client-side ACCEPT** (125 B, function code `0xca`). [capture] Note: `internal/appc` defines `F_ALLOCATE = 0x05` for the *direct* client conversation [our-code]; the gateway-relayed ALLOCATE is observed as `0xca`. The `0xc?`/`0xd?` range is the gateway-relayed family (see reconciliation notes). |
| `0xcb` | APPC | both | 241 B – 2.8 KB | **`F_SAP_SEND`** — carries a CUT (the function-call/result message, and on the client-facing leg the **ticket-bearing logon**). `F_SAP_SEND = 0xcb` is confirmed in code and capture. [our-code][capture] |
| `0xcf` | APPC (gateway-relayed) | S→C | 125 B | The **server-side ACCEPT** in the registered-server flow. Distinct from the client-side accept `0xca`. [capture] |
| `0xd1` | gateway-relayed | GW→server | 80 B | The **registered-server request** — the gateway handing a queued conversation to the registered program. [capture] |
| `0xd2` | gateway-relayed | GW↔server | 80 B | **keepalive / next request** in the registered-server session. [capture] |
| `0x08` | APPC | C→S | 241 B | The **async reply** (`F_ASYNC_SEND_DATA = 0x08`). [our-code][capture] |
| `NI_PING` / `NI_PONG` | NI | either | 8 B | `4e49 5049 4e47 00` (`NI_PING\0`) answered with `4e49 504f 4e47 00` (`NI_PONG\0`). A server must service these or the client trips `757` (see §8). [capture][our-code] |

Two flows use two different subsets of this table; they are laid out step by step
in §6.

---

## 3. The APPC record header (48 / 80 bytes)

Every `0x06`-versioned record (CPIC logon, `F_SAP_SEND`, ALLOCATE, accept, reject)
begins with the fixed **48-byte common header**; data records
(`F_SAP_SEND`/`F_RECEIVE`) carry a further **32-byte operation-info block** for
an **80-byte record header** before the payload. Field order and offsets are from
`internal/appc/records.go` (encode) and `internal/appc/appc.go` (decode). [our-code]

| Offset | Size | Field | Notes |
|---|---|---|---|
| 0 | 1 | `protocolVersion` | always `0x06` — selects the APPC layer (not the 64-byte gateway record) |
| 1 | 1 | `functionCode` | see §2 (`0x03`, `0xcb`, `0xca`, …) |
| 2 | 1 | `protocol` | client default `2`; CPIC selector `0x3` [gpl-ref] |
| 3 | 1 | `mode` | |
| 4 | 2 | `uid` (BE) | mirrored back on the response |
| 6 | 2 | `gatewayId` (BE) | server sets `1` on its records |
| 8 | 2 | `errorLength` (BE) | non-zero only when an error trailer follows; **zero on an accept** |
| 10 | 1 | `info2` | |
| 11 | 1 | `traceLevel` | |
| 12 | 4 | `time` (BE) | |
| 16 | 1 | `info3` | |
| 17 | 4 | `timeout` (signed BE) | |
| 21 | 1 | `info4` | |
| 22 | 4 | `sequenceNumber` (BE) | |
| 26 | 2 | `sapParameterLength` (BE) | derived from the payload |
| 28 | 2 | `padding` (BE) | |
| 30 | 1 | `info` | |
| 31 | 1 | `vector` | |
| **32** | 4 | **`appcReturnCode`** (BE) | the CPI-C (`cmRc`) half of the verdict |
| **36** | 4 | **`sapReturnCode`** (BE) | the SAP (`thRc`) half of the verdict |
| **40** | 8 | **`conversationId`** | client-generated (8 ASCII digits); the server must **echo it** on every record |
| 48 | 32 | `operationInfo` | data records only — comm/connection index, data length; header total = 80 |
| 80… | | payload | the CUT message |

**An accept is this header with both return codes zeroed.** Setting bytes 32–39
all to zero, `errorLength = 0`, mirroring the conversation id from offset 40, and
dropping any error trailer *is* the accept — confirmed by decoding a real reject
frame at these offsets: bytes 32–35 = `00 00 00 02` (`appcReturnCode` 2), bytes
36–39 = `00 00 02 a7` (`sapReturnCode` 679). Flipping those to zero is the whole
difference between a reject and an accept. [capture][our-code] `RecordHeaderInput`
already exposes `AppcReturnCode` and `SapReturnCode` as settable fields, so
producing an accept is expressible today. [our-code]

The conversation id at offset 40 is load-bearing beyond the accept: the client
creates it and expects **every** server record to carry the same 8 bytes;
replaying a record with another session's id makes the client fall back. [capture]

---

## 4. The initial CPIC logon (`0x03`)

The record `internal/cpic/logon.go` encodes and decodes for a **direct** client →
AS logon. [our-code]

### Prefix and signature

```
signature  d9 c6 c3 f0 f0 f0 f0 f0 f0 f0 f0 f0   (12 B) = EBCDIC "RFC000000000"
prefix     01 01 00 08 03 01                      (6 B)
<field chain, initialPreviousTag = 0x0101 (Start)>
trailer    ff ff 00 00                            (End marker + 2-byte zero)
```

The signature decodes as EBCDIC "RFC000000000". [our-code] The whole record is
guarded to stay under `0xffff` bytes. Note this is a **different record** from the
registered-server relayed logon (§5), which has no EBCDIC signature and starts its
chain at `initialPreviousTag = 0x0002`.

### Field chain and tag order

The chain is a run of `prevTag(2 BE) · tag(2 BE) · length · value` fields
(compact 4-byte or extended 8-byte header from `internal/rfcpro`), carrying the
previous tag so a decoder detects dropped or reordered fields. The decoder is
**positional-strict**: it checks the decoded tag sequence against exactly this
array (`initialTagOrder`) and rejects anything else. [our-code]

| # | Tag | Name | Content |
|---|---|---|---|
| 1 | `0x0101` | Start | empty |
| 2 | `0x0103` | ProtocolVersion | |
| 3 | `0x0106` | Capabilities | fixed blob |
| 4 | `0x0337` | LogonMarker | empty |
| 5 | `0x0514` | Session | session id |
| 6 | `0x0114` | **Client** | 1–3 digit SAP client |
| 7 | `0x0111` | **User** | **UTF-16** (see below) |
| 8 | `0x0117` | **Password** | scrambled block (see below) |
| 9 | `0x0115` | **Language** | upper-ASCII |
| 10 | `0x0501` | UnicodeIndicator | `01` |
| 11 | `0x0007` | ClientAddress | |
| 12 | `0x0018` | PartnerSystem | |
| 13 | `0x0011` | ConnectionType | `E` |
| 14 | `0x0012` | KernelRelease | |
| 15 | `0x0013` | KernelPatch | |
| 16 | `0x0008` | PartnerHost | |
| 17 | `0x0006` | Destination | |
| 18 | `0x0130` | Program | 80 B UTF-16LE |
| 19 | `0x0502` | ContextEnd | |
| 20 | `0x000b` | Kernel | |
| 21 | `0x0102` | Function | |
| 22 | `0xffff` | End | |

### The credential-bearing fields

- **User `0x0111` is UTF-16.** `CLAUDE` (6 chars) encodes to 12 bytes; a
  12-character user encodes to 24. Our encoder writes ASCII there and the system
  has never complained, which shows the field is *tolerant*, not that it is ASCII.
  [capture]
- **Client `0x0114`** — 1–3 digit tenant, zero-padded to three. Required even
  with a ticket (the ticket's own client is the *issuing* client, independent of
  the destination client). [our-code][inferred]
- **Password `0x0117` — an absent password is a 20-byte block, not empty.** With a
  password the scrambled block is 28 bytes; with *Current User* (no password) it
  is **20** bytes. So "no password" is the block's fixed header sent alone, not an
  omitted field. Our scrambler cannot currently emit those 20 bytes. The scramble
  is a published fixed-table obfuscation (not encryption) over printable ASCII;
  without SNC the credential crosses the wire protected only by the transport.
  [capture][our-code]
- **Language `0x0115`** — upper-ASCII, unchanged across logon variants. [capture]

Diffing nine connection tests, **only two fields ever change length** — the user
`0x0111` and the password `0x0117`; every other field is byte-identical. [capture]

### The response shapes

`DecodeInitialLogonResponse` distinguishes two prefixes: [our-code]

```
success prefix   01 01 00 08 01 01 01 05 04 01 00 03
error prefix     01 01 00 08 01 01 01 01 01 01 00 00
```

- A **regular (success)** response is a field chain of a small admitted tag set
  (Start, ProtocolVersion, Capabilities, **LogonStatus `0x0161`**, `0x0420`,
  `0x0450`, SystemCodePage, End), terminated by `0xffff`. A nonzero LogonStatus
  is preserved as a *rejection*, never reclassified as a protocol error.
- An **error** response is an `rfcerr` envelope whose message text (the ABAP
  message field **`0x0402`**) carries the human-readable reason.

**Known bug: some genuine "wrong credential" rejects are mis-reported as an
invalid preamble.** A direct ticket-logon reject (§5) came back as a well-formed
CPIC error response — status field `0x0161`, message field `0x0402` carrying
"Name or password is incorrect (repeat logon)" — yet `DecodeInitialLogonResponse`
returned `"initial CPIC logon error response has an invalid preamble"`. The
error-preamble grammar (`initialErrorPreambleTags`, a positional-strict match) is
too strict for this reject variant, so a real authentication failure surfaces as
a parser error instead of the message. The raw bytes are captured; loosening the
grammar to a bounded set turns the useless error into the actual message. Backlog.
`SAP_DEBUG_LOGON=1` dumps the raw logon response to stderr for exactly this kind
of diagnosis. [capture][our-code]

---

## 5. The SAP logon ticket, end to end

### The ticket object (one format, three carriers)

An SAP logon ticket is a signed, versioned TLV blob, base64-encoded and handed
around as ASCII text. Both a browser `MYSAPSSO2` cookie and an SM59 assertion
header parse with our own code: [capture]

```
header:  02  "4103"          version byte 0x02, code page 4103
then a run of  id(1) len(2 BE) value :
  0x01  user               UTF-16
  0x02  client             UTF-16 digits
  0x03  issuing system     UTF-16
  0x04  creation timestamp UTF-16  YYYYMMDDhhmmss
  0x0f  recipient client / portal codepage marker
  0x10  recipient system   (assertion variant only)
  0xff  signature          PKCS#7 SignedData (OID 06 09 2a 86 48 86 f7 0d 01 07 02)
```

The **assertion** variant differs from the browser ticket exactly where it
should: it adds `0x10` (recipient system) and `0x08`, and drops the browser's
`0x06` portal flag — so an assertion ticket is *addressed* to one target system +
client ("for dedicated target system"), while a browser ticket is general. Same
envelope, one extra field. [capture] Because the envelope decodes offline, a
client can tell whose ticket it holds, for which system, and whether it has
expired, before any network call. [our-code]

### Three transports of the same object

| Carrier | Where | Encoding |
|---|---|---|
| **HTTP header / cookie** `MYSAPSSO2` | SM59 type-H "Send assertion…"; browser cookie | **plain base64 ASCII** [capture] |
| **Classic RFC field `0x0670`** | registered-server relayed logon / client CUT | the same base64 string laid down as **UTF-16LE** (each base64 char + `0x00`) [capture][our-code] |
| (HTTP) `SAP-R3AUTH` | SM59 type-H "SAP RFC Logon" | a sealed opaque 384-byte block (hex→base64→`v=1U,`+384 bytes); encrypted, carries a nonce; *not* the field-chain ticket — a different sealed-credential path [capture] |

Stripping the UTF-16LE wrapper from a `0x0670` value yields byte-for-byte the same
ticket object the HTTP path carries. Only the transport wrapper differs. [capture]
The field codec is in `internal/cpic/ticket.go`: `NormalizeTicket` (undo
URL-escaping and the cookie's `!`→`/` substitution → canonical base64),
`encodeTicketField` (canonical base64 → UTF-16LE), `DecodeTicketField` /
`TicketFromLogonFields` (inverse). [our-code]

### The crucial structural point

**In the registered-server flow the ticket rides an `F_SAP_SEND` CUT (`0xcb`), not
the initial logon.** The gateway passes the `0x0670` field through unchanged: it
appears client-side inside the `0xcb` frame and server-side inside the relayed
`0x03` logon, same tag, same UTF-16LE-base64 encoding, chained
`prev=0x0002 → 0x0670 → 0x0114` (right after the repeated local-host field,
immediately before the client). Only the frame's function code differs
(`0xcb` client-side ↔ `0x03` server-side). [capture]

Cross-mode confirmation across the captured SM59 modes: no-ticket logons omit the
field entirely; with *Send Assertion Ticket* the field appears and the recipient
inside it (`0x10`/`0x0f`) tracks the configured target — the real `A4H`/`001`
versus a bogus `SNIFF`/`999` — which is what proves it is the assertion ticket and
not something incidental. [capture]

### Direct client → AS ticket logon is NOT yet characterised

Stated plainly: **we cannot yet log on directly with a ticket, and we do not know
the direct wire format.** A password logon against the real gateway succeeds; a
direct ticket logon with the same client — `0x0670` in the *initial* logon,
password omitted — is **rejected** with "Name or password is incorrect (repeat
logon)". [capture] The reason is structural: the real client authenticates with
the ticket in a **separate `F_SAP_SEND` CUT** (chain `0x0002`-linked, carrying
`0x0670`), not in the initial CPIC logon (chain `0x0101`-linked with
`0x0111`/`0x0114`/`0x0117`). The AS ignored `0x0670` in the initial logon and
fell through to password auth. [capture][inferred]

Every capture we hold is the *registered-server* relayed form. A direct client →
AS ticket logon (what a JCo/NW-RFC-SDK client does with `MYSAPSSO2`) may place the
ticket in the initial logon or in a CUT of its own — we cannot tell without a
positive capture of a real direct ticket-logon client, which we do not have. The
direct format likely also requires the user field empty (the ticket carries the
user) or a distinct flag. Until such a capture exists, direct ticket-based classic
RFC logon stays unimplemented; the field codec and the HTTP ticket paths are done.
`encodeTicketField` keeps a `SAP_TICKET_ENC` escape hatch (utf16le default,
utf16be, ascii, raw) and `EncodeInitialLogonRequest` a `SAP_TICKET_TAG` override,
precisely because the direct wire form is still open. [our-code]

At the API level the ticket replaces the *password*, not the *user* (SAP documents
the user name as optional with `MYSAPSSO2`); there is one parameter for both logon
and assertion tickets. [doc][gpl-ref]

---

## 6. The registered-server flow, step by step

This is the only flow in which we have watched a **real ABAP client's** logon (and
a forwarded ticket) cross classic RFC. It was captured by running SAP's own
`rfcexec` as a genuine registered server and sniffing the real gateway relay, with
the whole path kept on `127.0.0.1` inside the container so nothing tripped the
ACLs (§7). [capture]

There are two legs. **Leg-1** is the server ↔ gateway side (what `rfcexec` sees).
**Leg-2** is the client ↔ gateway side (what the ABAP client sends). The gateway
relays between them.

### Leg-1 — server ↔ gateway (per call)

| Dir | Fn | Bytes | Meaning |
|---|---|---|---|
| C→S | `0x0b` | 64 | NI/gateway handshake |
| C→S | `0x68` | 512 | gateway `GW_NORMAL_CLIENT` region |
| C→S | `0xd1` | 80 | the **registered-server request** (gateway → server) |
| S→C | `0xcf` | 125 | the **ACCEPT** (server-side, function code `0xcf`) |
| S→C | `0x03` | 1716–2800 | the **relayed CPIC logon** the ABAP client composed |
| C→S | `0x08` | 241 | the server's **async reply** |
| C→S | `0xd2` / `0x0b` / `0xd1` | 80 | keepalive / next request |

Registration itself (topology A) is the `GW_REGISTER_TP` request, gateway request
type `0x0b` (with `GW_UNREGISTER_TP = 0x0c`), initiated by the external program
which knows gateway host, service (`sapgwNN`), and program id (case-sensitive,
must equal the SM59 Registered Server Program id). [gpl-ref][doc] Its exact byte
layout is **not public** — neither pysap nor Wireshark models the register body —
so implementing our own registration still needs one capture. [gpl-ref]

### Leg-2 — client ↔ gateway (per call)

| Dir | Fn | Bytes | Meaning |
|---|---|---|---|
| C→S | `0x03` | 64 | gateway record (`GW_NORMAL_CLIENT`, request type 3) |
| C→S | `0xca` | 453 | **ALLOCATE** (names destination / program id / `%%RFCSERVER%%` / caller) |
| S→C | `0xca` | 125 | the **client-side ACCEPT** (function code `0xca`, not the server-side `0xcf`) |
| C→S | `0xcb` | 2714–2800 | **`F_SAP_SEND`** carrying the client's logon **+ the ticket** (`0x0670`) |
| S→C | `0xcb` | 241 | reply |

The two accepts differ by function code — `0xcf` on the server-facing leg, `0xca`
on the client-facing leg — but both are the same object: an APPC header with the
two return codes zeroed and the conversation id mirrored (§3). The ticket field is
identical on both sides (§5); the gateway relays it unchanged. [capture]

### The ticket-size signature

The relayed `0x03` logon comes in two size families and the split is exactly the
ticket: [capture]

- **1716 / 1802 B** — *Do Not Send Logon Ticket*: `0x0670` absent.
- **2674–2800 B** — *Send Assertion Ticket*: `0x0670` present, adding ≈1 KB. (A
  992-byte value = 496 UTF-16 units ≈ 495 base64 chars ≈ 370 decoded ticket
  bytes.)

### How the accept was proven

Earlier blind attempts (`serve_ticketcatch`) progressed by trial: no accept →
`679` "not registered"; an 80-byte zeroed header → `225` "Unknown CPIC function"
(too short); a **125-byte real reject frame with return codes zeroed** →
conversation established, SAP then sent `NI_PING`; answering `NI_PING → NI_PONG`
sometimes tripped `757` (timing/role dependent). The `rfcexec` capture then gave
the positive accept frames (`0xcf` server-side, `0xca` client-side, both 125 B) to
copy instead of guessing. [capture]

---

## 7. Gateway ACLs and their failure signatures

The gateway enforces two ACL layers, both of which can refuse us regardless of our
code correctness. [doc]

| Control | Governs | Failure we saw |
|---|---|---|
| `gw/acl_mode` | master switch (`= 1` in the captured system) | with `acl_mode = 1`, any registration that reaches the gateway through a *different-host* socket is flagged as a **proxy** |
| `reginfo` | which programs may **register** | `RC 756` — "registration ... from PROXY host ... not allowed" when the path is not local; a permissive `reginfo` (`P TP=* HOST=* ACCESS=* CANCEL=*`) must exist and be re-read (SMGW → Expert Functions → External Security → Re-Read NI ACL) |
| `secinfo` | which callers may **access** a registered server | `RC 748` — "access to registered server is not permitted" for a proxied client |

**The practical rule:** inside the container everything must stay on `127.0.0.1`
to be treated as local. Registration succeeded only when `rfcexec`, the sniffer,
and the relay were all on `127.0.0.1` (sniffer `127.0.0.1:3099 → 127.0.0.1:3300`
for leg-1; `127.0.0.1:3388 → 127.0.0.1:3300` for leg-2). A sniffer on any other
host is refused (`756` for registration via a proxy, `748` for a proxied client).
[capture] A separate register-not-allowed condition (`720`, "GW: Registration of
tp not allowed") is the `reginfo` gate for our own future registration. [doc]

Relevant login profile parameters read on the captured system:

| Parameter | Value | Effect |
|---|---|---|
| `login/accept_sso2_ticket` | `1` | the system accepts SSO2 ticket logon (`0` would give SSO error 20) [doc] |
| `login/create_sso2_ticket` | `2` | the system issues SSO2 tickets [doc] |
| `login/ticket_expiration_time` | `8:00` | a ticket lives ~8 hours from issue [doc] |

---

## 8. Return-code appendix

Every APPC / SAP / CPIC return code we have seen, with its meaning and where we
saw it. The two APPC verdict codes live at offsets 32 (`appcReturnCode`, the
CPI-C / `cmRc` half) and 36 (`sapReturnCode`, the SAP / `thRc` half) of the
record header (§3).

### APPC / gateway codes (observed)

| Half | Code | Meaning | Where |
|---|---|---|---|
| `cmRc` (appc, @32) | `0` | accept — conversation may proceed | derived + proven [capture] |
| `cmRc` | `2` | `CM_ALLOCATE_FAILURE_RETRY` — allocate failed, temporary | decoded from a reject frame [capture][doc] |
| `thRc`/SAP (@36) | `0` | OK | accept [capture] |
| `thRc`/SAP | `225` | "Unknown CPIC function" (frame too short / malformed) | our 80-byte zeroed accept attempt [capture] |
| `thRc`/SAP | `679` | "transaction program not registered" | reject when nothing registered [capture][doc] |
| `thRc`/SAP | `720` | "GW: Registration of tp not allowed" (`reginfo` gate) | [doc] |
| `thRc`/SAP | `748` | "access to registered server is not permitted" (`secinfo`, proxied client) | [capture][doc] |
| `thRc`/SAP | `756` | "registration ... from PROXY host ... not allowed" (`reginfo`, non-local) | [capture][doc] |
| `thRc`/SAP | `757` | "client has not answered PING messages" (NI keepalive cadence) | [capture] |

### SSO2 / ticket logon codes (SAP Note 320991)

Surfaced at the RFC API as `RFC_ERROR_LOGON_FAILURE` (group 103). [doc]

| Code | Meaning |
|---|---|
| 20 | ticket logon generally deactivated (`login/accept_sso2_ticket = 0`) |
| 21 | syntax error in ticket / reentrance ticket not valid |
| 22 | digital signature check fails (cert present but does not verify) |
| 23 | ticket issuer not in the ACL table (`TWPSSO2ACL`, via STRUSTSSO2) |
| 24 | ticket expired |
| 25 | assertion-ticket receiver is not the addressed recipient |
| 26 | ticket contains no / an empty ABAP user id |
| 27 | reauthorization check: ticket does not match current user |
| 28 | ticket logon denied by security policy |

Mapping the common cases: expired → 24; issuer untrusted → 23 (not in ACL) or 22
(signature fails); `accept_sso2_ticket = 0` → 20. [doc]

> **Not observed in our captures:** the task brief also lists `cmRc` 17/19/20 and
> `thRc` 236/239/242 as codes to include. These are standard CPI-C / `thRC` values
> in SAP's documentation, but **no source file in this repo records us observing
> them**, so they are not tabulated above as ours. See the reconciliation notes.

---

## 9. Status table — proven / inferred / unknown

| Mode / claim | Status | Evidence |
|---|---|---|
| Gateway handshake (64-byte `GW_NORMAL_CLIENT`, resp. bytes 29/55 changed) | **proven live** | `internal/gateway`; [0001](0001-live-type3-server.md); [0002](0002-wire-constants.md) |
| Direct type-3 **password** logon | **proven live** (green Connection Test answered by our Go) | [0001](0001-live-type3-server.md) |
| Direct type-3 **ticket** logon | **rejected / unknown format** | §5 [capture] `registered-server-conversation.md` |
| Absent-password = 20-byte block; user `0x0111` is UTF-16 | **proven** | `logon-credential-block.md` [capture] |
| Ticket object format (`02`/`4103` TLV + PKCS#7) | **proven** (decodes with our code) | `http-destination-logon-modes.md`; `ticket-in-logon.md` [capture] |
| Ticket over HTTP (`MYSAPSSO2` header/cookie, plain base64) | **proven** | `http-destination-logon-modes.md` [capture] |
| Ticket over classic RFC = field `0x0670`, UTF-16LE base64 | **proven** (relayed form) | `registered-server-conversation.md`; `ticket-in-logon.md` [capture] |
| Ticket rides an `F_SAP_SEND` CUT, not the initial logon | **proven** (leg-2) | §6 [capture] |
| Registered-server accept = zeroed-return-code header (`0xcf`/`0xca`, 125 B) | **proven** (positive `rfcexec` capture) | `registered-server-conversation.md` [capture] |
| Our own gateway registration (`GW_REGISTER_TP` body) | **unknown byte layout** | `gw-register-accept.md` [gpl-ref] |
| Gateway ACL failure signatures (`748`/`756`/`720`) | **proven** (748/756 observed; 720 from doc) | §7 [capture][doc] |
| `SAP-R3AUTH` sealed 384-byte block | **observed, opaque** | `http-destination-logon-modes.md` [capture] |
| SOAP RFC endpoint / ADT ticket-auth | **verified live, no wire dump** | `http-destination-logon-modes.md` (referenced design note absent) |
| Serializer (Classic / basXML / Fast) chosen by the **server's** logon-accept | **proven** | [0001](0001-live-type3-server.md) [capture] |

---

## Reconciliation notes — contradictions and unresolved points

Things in the sources that conflicted, or that could not be pinned:

1. **`docs/design/http-only-systems.md` does not exist.** The task lists it as a
   source of truth for the cookie→RFC analysis, `MYSAPSSO2` vs `SAP_SESSIONID`,
   `login`/`accept_sso2_ticket`, and the SOAP RFC endpoint. No such file is in the
   repo (`docs/design/` contains only `write-fm-safety.md`). The HTTP-side facts
   here were reconstructed from `http-destination-logon-modes.md` and
   `.private/sso2-ticket-logon.md`. In particular **`SAP_SESSIONID`** is named in
   the task but appears in no repo source, so it is not described above beyond the
   `MYSAPSSO2` cookie it would sit alongside.

2. **`0xca` (captured ALLOCATE) vs `0x05` (`F_ALLOCATE` in `internal/appc`).** The
   direct-conversation codec defines `F_ALLOCATE = 0x05`, but every gateway-relayed
   ALLOCATE we captured is function byte `0xca`. The `0xc?`/`0xd?` range
   (`0xca`, `0xcb`, `0xcf`, `0xd1`, `0xd2`) is the **gateway-relayed / SAP** family;
   only `0xcb` (`F_SAP_SEND`) overlaps a named `internal/appc` constant. `0xca`,
   `0xcf`, `0xd1`, `0xd2` are capture-only opcodes not yet given constants in code.
   Treat the `internal/appc` `0x05`-style codes as the *direct* client-conversation
   set and the `0xc?`/`0xd?` set as the *relayed* set until they are unified.

3. **Overloaded function-code bytes.** `0x03` means both the gateway
   `GW_NORMAL_CLIENT` request type *and* the APPC initial/relayed CPIC logon;
   `0x0b` means both the 64-byte gateway handshake (leg-1) *and* APPC
   `F_DEALLOCATE`. The byte alone is ambiguous — the framing layer disambiguates.
   The §2 table flags each case; do not read a bare function byte without knowing
   its layer.

4. **`cmRc` 17/19/20 and `thRc` 236/239/242** requested by the task brief are not
   recorded as observed in any repo source; only `cmRc` 2 and the `thRc` set
   above were actually seen by us. They are omitted from the "observed" appendix on
   purpose (clean-room discipline: we tabulate what we saw). If they were seen in a
   capture that is not committed, the appendix should be extended when that source
   is added.

5. **Two conflicting predictions about where a ticket sits, one now resolved.**
   `.private/sso2-ticket-logon.md` predicted the ticket would ride the *initial*
   logon in a `0x011x`–`0x012x` tag, replacing the password. The later capture
   analysis (`ticket-in-logon.md`) proved that wrong for the relayed record: the
   tag is `0x0670`, the chain is `0x0002`-linked, and there is no `0x0117` field at
   all. The earlier note is kept for its API-level facts (which remain correct) but
   its wire-tag guess is superseded. The *direct*-logon placement (the original
   prediction's actual scope) remains genuinely unknown (§5).

6. **The `0x5001` block grows ~86 bytes when a ticket is present**, yet contains no
   base64 and no PKCS#7 — it is the RFC call / TH header, not a second ticket copy.
   Why it expands is unexplained; the smallest next experiment is a byte-diff of the
   two `0x5001` values aligned on their `24 48 02 03 00 4103` header. Offline, no
   live system needed. [capture]
