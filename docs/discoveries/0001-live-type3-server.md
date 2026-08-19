# Discovery 0001 — Answering an SM59 type-3 client as our own Go server

Status: in progress (M8). Captured live against A4H (SAP_BASIS 793) on
2026-08-19. This records what the wire actually requires, learned by standing
`cmd/rfc-lab` in front of a real SM59 destination and reading the errors.

## Goal

Make an SM59 **type 3** (RFC to ABAP) Connection Test go green answered by our
Go code — not by a real system behind a transparent proxy.

## Setup

`cmd/rfc-lab` binds several ports on one host; the SM59 connection type / system
number selects the mode:

| SM59 | port | mode | who answers |
|---|---|---|---|
| type 3, sysnr 00 | 3200/3300 | transparent sniffer → real .103 | real system (green, captured) |
| type 3, sysnr 10 | 3310 | our Go server | us |
| type H/G | 8000 | our HTTP server (200) | us — **green** |
| type W | 44300 | our WebSocket server (101 upgrade) | us |

Only type 3 has a real backend to sniff; for the other types nothing is behind
us, so we answer ourselves. The sniffer now tags every captured frame with a
per-connection id and a port label, so concurrent connections segment cleanly.

## What the wire requires (in the order we hit each wall)

1. **Gateway record is not echoed verbatim.** The client sends a 64-byte
   normal-client record; the real server replies with the same record but with
   two bytes changed: offset **29** incremented (`0e`→`0f`) and offset **55**
   `cb`→`fb` (accepted-capability bits). Echoing verbatim is wrong.

2. **The client accepts a replayed CPIC logon-accept enough to proceed** — the
   hardest gate is passable: replaying a real 817-byte logon-accept gets the
   client past logon and into issuing RFC calls.

3. **Conversation id is client-generated and must be mirrored.** The client
   creates the 8-byte CPIC conversation id in its init record (APPC header
   offset 40) and expects every server record to carry the *same* id. Replaying
   a template with another session's id makes the client fall back. Fix:
   rewrite outgoing records' id to this client's (`withConvID`).

4. **RFC assigns a 16-byte connection GUID too (since 46D).** Beyond the
   conversation id, `RSRFCPIN` validates a per-connection RFC GUID. A mismatched
   GUID aborts with **`RFC_INVALID_UUID_DETECTED`** ("RFC data being sent
   between the wrong partners"). The GUID's last 8 bytes are a stable node/host
   suffix (`e100 0000 <host-ip>`); the first 8 identify the session. Fix
   (experimental): find the client's GUID by that suffix and rewrite the
   template's GUID to it (`findRFCGUID`).

5. **The logon-accept is session-specific — replay is a dead end.** Even with
   the gateway bytes, conversation id, and GUID rewritten, the client still
   degrades: against the real system it sends full **721-byte** RFC_PING records
   (the RSRFCPIN callback path); against our replayed accept it sends **80+267
   byte** minimal records, then trips the UUID abort. The 817-byte logon-accept
   carries negotiated capabilities, the server GUID, and a challenge bound to
   the specific init — it cannot be blindly replayed across sessions.

6. **ABAP client speaks fast-serialization, our Go client speaks basXML.** The
   ABAP SM59 client's requests are fast-ser (`0x0502` classic CUT is not what it
   sends); our own client's are basXML (`754…` eyecatchers). A response captured
   for one dialect is read as an "Unknown CPIC function" by the other.

## Conclusion / next step

Reaching `RSRFCPIN` proves our NI + gateway + APPC + CPIC transport is
byte-correct. A green type-3 test now needs the logon-accept to be **generated**
from the client's init (mirror its GUID and conversation id, negotiate
capabilities, answer any challenge), then a generated RFC_PING/RSRFCPIN
response — not replayed. That is the next M8 build.

## Reusable results shipped

- `internal/sniffer`: per-connection id + label on every frame; raw-tee mode for
  non-NI transports (WebSocket/HTTP).
- `internal/rfcserver`: `ServeReplay` (request-driven, conv-id + RFC-GUID
  rewriting), `LoadConnection` (segments by connection id).
- `cmd/rfc-lab`: multi-protocol endpoint (type 3 sniff, type 3 our server,
  type H/G HTTP, type W WebSocket).
- `cmd/rfc-server`: standalone replay server.

## Update — first green: our server answers a live SM59 Connection Test

After the walls above were mapped, the fixes landed and the type-3 Connection
Test went **green answered entirely by our Go server** (sysnr 10 → port 3310),
repeatably across many clicks. What it took, on top of the earlier list:

- **GUID byte order.** The connection GUID is stored one way in the client's
  records (init order) and byte-swapped in the server's (SAP structured GUID:
  first three components reversed). Rewriting only one order tripped
  `RFC_INVALID_UUID_DETECTED`; rewriting both orders (`swapRFCGUID`) cleared it.
- **The logon-accept is 97% constant.** Diffing 9 real logons: only ~19 of 817
  bytes vary — the uid byte, the conversation id (an ASCII counter at offset
  40), and the GUID/timestamp bytes. Everything else is server identity. So the
  accept is *generatable* from a template plus the client's session tokens.
- **The connection is pooled.** SAP reuses one TCP socket for several logons;
  each begins with a fresh CPIC-init carrying a new conversation id and GUID. The
  server must re-capture both per init and rewind its reply sequence to the
  logon-accept, or the second logon starves and hangs.

With those, `ServeReplay` (request-driven, per-logon token rewriting) answers
Connection Test after Connection Test.

## Next wall — parameterized calls need fast-ser (Delta Manager)

Running the traffic generator (real `CALL FUNCTION` with structures and tables)
against our replay dies with **`DELTA_NO_OBJECT`** ("Delta Manager: 1 is not a
valid object"): the ABAP client applies table/row results through the RFC
fast-serialization Delta Manager, and our replayed ping responses carry no valid
deltas. Connection Test (pure RFC_PING) is green; real data calls are not,
because replay has no meaningful response for them.

## Direction — a state machine, not replay

The next build replaces replay with a real server:

```
CONNECT  accept gateway record, reply (bytes 29,55 set)
LOGON    on CPIC-init: GENERATE the logon-accept (constant template + mirror
         the client's conversation id and GUID) — no capture needed
SESSION  per F_SAP_SEND: decode function + params -> handler -> encode response
         (re-init on the same socket -> back to LOGON)
```

Handshake + RFC_PING are reachable with what is already known. Parameterized
responses need a **fast-ser encoder** (the Delta Manager format) — the large,
open piece, of which `DELTA_NO_OBJECT` is the first glimpse. This is the
`docs/polyglot-rfc-server.md` server, now grounded in live wire facts.

## Update — the logon-accept is init-dependent (per SM59 test)

Stage 2 (decode requests) needed no new code: the existing classic
`DecodeCutFunctionRequest` reads live ABAP fast-ser requests with zero errors
across 27 captured calls (RFC_PING, RFC_SYSTEM_INFO, STFC_CONNECTION,
STFC_STRUCTURE, STFC_STRING, RFC_READ_TABLE). `ServeSmart` now decodes and
dispatches by name; RFC_PING drives the baked dance, RFC_SYSTEM_INFO returns a
captured RFCSI (patched per session), others return SYSTEM_FAILURE.

But the SM59 **Unicode Test** exposed the next wall. Captured live, it issues **no
RFC function at all** — it performs the logon twice and checks the handshake:

```
gateway; INIT 1444B -> accept 1079B; 80B; INIT 1444B -> accept 1079B; 80B
```

Its init (1444 bytes) and its accept (1079 bytes) both differ from the
Connection Test's (1818 / 817). So the **logon-accept is a function of the init**:
different SM59 tests send different inits (requesting different capabilities) and
expect correspondingly different accepts. A single baked accept template — enough
for the Connection Test — cannot satisfy the Unicode Test; the client stalls
after logon (sends an 80-byte control and never proceeds).

Conclusion: baking one template per test does not scale. The real server must
**generate the logon-accept from the client's init** — parse the requested
capabilities and emit the matching accept. That, plus the fast-ser Delta encoder
for table results, are the two large research pieces remaining. Everything up to
here — transport, gateway, per-session token mirroring, request decode/dispatch,
and a green Connection Test answered by our own Go — is done and on main.

## Update — accept selection passes Unicode; programs prove generation is required

Keeping two accept templates and selecting by the client's init length makes the
**Unicode Test pass green** (init 1444B → accept 1079B; Connection Test 1818B →
817B). But a real calling program (`ZLOCAL_RFC_TEST`) logs on repeatedly with
*different* init sizes — 1444, 1668, and more — each expecting its own accept.
Template selection cannot cover an open set of inits, so it stalls (the client
sends an 80-byte control and never issues its call).

Definitive: the logon-accept must be **generated from the init** — parse what the
init requests and emit the corresponding accept — not chosen from captures. Two
research pieces remain for a general server: (1) init→accept generation, (2) the
fast-ser Delta encoder for parameterized results. Fixed handshake tests
(Connection, Unicode) work today; arbitrary programs need both.

## Update — full calling-program flow captured (the material for both paths)

Running the real `ZLOCAL_RFC_TEST` program through the sniffer captured its whole
type-3 session:

```
gateway
INIT 1444B -> accept 1079B      (a metadata/negotiation logon, then a burst of 8B NI pings)
INIT 1668B -> accept  807B      (the work logon)
REQ RFC_SYSTEM_INFO -> RESP 746B
REQ STFC_CONNECTION -> RESP 2039B + 661B   (multi-frame)
REQ STFC_STRUCTURE  -> RESP 765B
REQ RFC_READ_TABLE  -> RESP 1480B
REQ STFC_STRING     -> RESP ...
```

Findings:
- A program logs on **twice** (metadata init 1444B, then work init 1668B), each
  with its own accept — confirming again that the accept is init-dependent, and
  adding accept sizes 807 (init 1668) to the map {1444->1079, 1668->807, 1818->817}.
- Between logons the client exchanges a burst of **8-byte NI ping/pong** frames a
  server must also service.
- **Every function now has a real captured response** (RFC_SYSTEM_INFO,
  STFC_CONNECTION — multi-frame, STFC_STRUCTURE, RFC_READ_TABLE, STFC_STRING).

This is the raw material for both remaining paths. The pragmatic one is a
**content-addressed responder**: decode the request's function name, serve the
captured response for it (patched with this session's tokens). For a fixed
program with fixed inputs it works end to end without a fast-ser encoder — the
original backlog idea, now feasible. The general one is true fast-ser Delta
generation. Captures preserved at /tmp/gold-program.jsonl and /tmp/gold-params.jsonl.

## MILESTONE — a real ABAP RFC program runs fully green against our Go server

The content-addressed responder (`ServeContentAddressed`, rfc-lab system number
12) makes the live `ZLOCAL_RFC_TEST` program complete **all six calls rc=0**
answered by our Go — RFC_PING, RFC_SYSTEM_INFO, STFC_CONNECTION (with its
server→client callback), STFC_STRUCTURE (structure), RFC_READ_TABLE (17 cols × 2
rows), STFC_STRING:

```
RFC_PING rc=0; RFC_SYSTEM_INFO rc=0 sysid=A4H; STFC_CONNECTION rc=0 echo=...;
STFC_STRUCTURE rc=0 rows=1; RFC_READ_TABLE rc=0 cols=17 rows=2; STFC_STRING rc=0
```

How: parse the capture into a gateway reply, init-length-keyed accepts, and a
per-function reply script; service NI keepalives; on each request run the script
with this session's tokens patched (conversation id + RFC GUID in both byte
orders). Grounded in the measurement that a request is 98.6–99.1% fixed (only the
GUID and a sequence byte vary), so patching a captured reply is sound.

Known edge: the SM59 Connection Test against this program-tuned endpoint fails
(`CALL_RFC_TABLE_MULTIREF` / RFCTAB40) because RFC_PING falls back to the baked
dance and a replayed table's multiref collides across contexts; Connection Test
works on the smart (sys 11) and replay (sys 10) endpoints. Next: a meaningful
fast-ser codec (generate responses from values, resolving multiref per session)
and, on top, real Go/JS function implementations behind a bridge adapter.

## Update — mode-aware responder: every SM59 test + the program, one endpoint

The response depends on the **serialization mode negotiated at logon**, not just
the function name. Measured: the SM59 **Fast Serialization Test** logs on with
init 1394/1423B -> accept 811/834B and gets RFC_PING=469B, RFC_SYSTEM_INFO=1147B —
different bytes from the normal path (accept 807B, RFC_SYSTEM_INFO=746B) for the
*same* functions. So the responder keys reply scripts by "acceptLen|FUNCTION" (the
accept in effect = the mode) and ingests several captures at once.

Result: a single endpoint (rfc-lab sys 12) now answers, all green from our Go —
**Connection Test, Unicode Test, Fast Serialization Test, and the full
ZLOCAL_RFC_TEST program** — across three serialization modes (807 normal, 811/834
fast-ser, 817/1079 handshake tests). Step 1 (addressable content) is complete for
the observed surface. Remaining: a real fast-ser codec that generates these
responses from values (so it works for inputs we never captured) and, on top,
real Go/JS function implementations behind a bridge adapter.

## Step 2 kickoff — the fast-ser payload is decodable (tag 0x5001)

Feeding a real RFC_SYSTEM_INFO response to the classic decoder gets past the
framing, GUID (0x0514) and success control (0x0420) but stops at an unknown tag
**0x5001** — the fast-serialization parameter container — plus an extra 8-byte
trailer after `ffff0000ffff`. Decoding a 0x5001 block by hand cracked the format:

```
0x5001 <len:2>
  "<PARAM_NAME>"            ASCII, e.g. RFCSI_EXPORT
  <field descriptor table>  0006 06LL … (per-field type/length)
  <field values>, each:  0x43 <len:1> 0x80 <value>   (char field, ASCII value)
```

Verified against live bytes: `43 03 80 "A4H"` (RFCSYSID), `43 11 80
"vhcala4hci_A4H_00"` (RFCDEST, len 0x11), `43 0a 80 "172.17.0.3"` (RFCIPADDR),
`43 05 80 "Linux"` (RFCOPSYS), `43 03 80 "793"` (RFCKERNRL), `43 03 80 "HDB"`.
Values are ASCII here, not UTF-16LE. A second 0x5001 block carries the program
context ("SAPLSRFC…").

So a meaningful codec is tractable: to generate a response, emit `param name +
descriptors + (0x43 len 0x80 value) per field`, wrap in 0x5001, and add the
success/GUID framing. Remaining to map: the descriptor table's exact grammar,
the non-char type markers, table (multi-row) encoding, and the trailing 8 bytes
(`0000 <len?> 0000 6d60`). This is the concrete Step 2 work — no longer opaque.

## Step 2 — the SM59 serializer switch reframes the problem

SM59 → Special Options → Select Transfer Protocol → **Serializer** is a
per-destination dropdown: Classic / basXML / Force basXML / Fast serializer. So
the fast serialization we had been fighting was simply the destination's default
("Fast serializer"), not something inherent.

Setting the destination to **Classic serializer** and capturing live: the
fast-ser container **tag 0x5001 disappears entirely** (the Delta Manager /
multiref beast — gone). What remains is a small set of S/4HANA classic-extension
tags our ported (non-S4) `classicrfc` does not yet know — the connection GUID
field **0x0514**, and tags **0x0104 / 0x0512** — which our own Go client never
triggers (its responses decode cleanly). "Client or server is S/4 HANA system"
is checked on the destination, which is what adds them.

This narrows Step 2 dramatically: instead of implementing fast serialization
(hard: stateful multiref), the classic-serializer path needs only a handful of
S4 classic-extension tags decoded/encoded on top of the existing classic codecs.
Captures: /tmp/gold-classic.jsonl (ABAP classic) alongside the fast-ser golds.

## Step 2 — narrowed to a few S/4 classic-extension tags

With **Classic serializer + Deactivate delta manager** (SM59 Special RFC Flags,
see the flags memory), the wire is at its simplest: no fast-ser 0x5001, no
multiref. Our real RFCPRO decoder now parses the framing, the connection GUID
(0x0514), and the success control (0x0420) of an ABAP response, and stops at
exactly two unknown tags:

- **0x0104** — appears in the RFC_SYSTEM_INFO response
- **0x0331** — appears in the STFC_STRUCTURE response (near the classic table
  tags 0x0301–0x0304)

These are S/4HANA classic-serialization extensions the ported (non-S4) RFCPRO
grammar in `internal/cpic/grammar.go` does not model. That is the entire
remaining gap for the classic path: teach the grammar these tags (and their
counterparts on the encode side), verify against the saved golds
(/tmp/gold-classic-deltaoff.jsonl), and the server can then GENERATE classic
responses from values using the existing classicrfc codecs — no fast-ser encoder
needed. This is code work against captures, not more live SAP interaction.

## Step 2 — S/4HANA classic tag family mapped; scalars decode

With Classic serializer + delta off, decoding an S/4 classic response needed two
layers taught the S4 extension tags: the rfcerr envelope (tolerate) and
classicrfc (extract). Full S4 tag inventory from the golds:

- **0x0104** — trailing annotation on scalar/structure exports (skippable; the
  values are the classic 0x0201/0x0203 fields). **Done** — scalars now decode
  (RFC_SYSTEM_INFO, STFC_CONNECTION echo, STFC_STRING).
- **0x033x table family** (a distinct S4 table serialization):
  - `0x0331` (len 4) — S4 table id/marker (one per table)
  - `0x0333` (len 12) + `0x0334` (len N) — S4-native table descriptor + row data
    (STFC_STRUCTURE's RFCTABLE uses this)
  - `0x0335` (len 12) — S4 table descriptor that precedes *classic* `0x0302`
    header + `0x0304` rows (RFC_READ_TABLE uses this mixed form)
  - `0x0336` (len 4) — S4 table trailer / row count

Row data is plainly present and readable (RFC_READ_TABLE T000 rows "000SAP SE",
FIELDS "MANDT/MTEXT/…"; STFC_STRUCTURE RFCTABLE with "A4H"). The remaining Step 2
work is to parse this 0x033x family into tables (both the S4-native 0x0333/0x0334
form and the 0x0335+classic mixed form), then the encode side. Golds:
/tmp/gold-classic-deltaoff.jsonl.

## Step 2 — S/4HANA classic DECODE complete (scalars + tables)

The 0x033x table family is now parsed, so the decode side of the S/4 classic
path is done. Both forms extract, verified against live vectors
(internal/classicrfc/testdata/s4_*.hex):

- **mixed** — 0x0335 descriptor + classic 0x0302 header + 0x0303/0x0304 rows
  (RFC_READ_TABLE): recovers the T000 rows ("…SAP SE…Walldorf") and the FIELDS
  descriptions (MANDT, MTEXT, …).
- **native** — 0x0333 descriptor + 0x0334 rows at fixed width (STFC_STRUCTURE):
  recovers the RFCTABLE row.

0x0331 (table id) and 0x0336 (trailer) are positional markers; table names are
positional (S4TAB<id>) because the wire carries only ids — real names come from
the request's table order. With Classic serializer + delta manager off, an S/4
classic response now decodes end to end: envelope classify → scalars → tables.

**Next:** the ENCODE side — a "conscious" smart server that GENERATES these
responses from values (inverse of the decode above), then the polyglot RFC
bridge (`docs/polyglot-rfc-server.md`).

## Step 3/4 — the serializer is negotiated by the SERVER's logon-accept

Calling the real custom FMs from our Go **client** works fully: Z_DOUBLE(21)=42,
Z_GREET("world"), against live A4H, direct and via the sniffer — metadata
discovery + typed call + decode, all classic, pure Go.

The **server** (conscious/bridge) hit the serializer-negotiation wall. Dumping our
server's own exchange (cap-gen.jsonl, added to ServeConscious) showed the S4
client sending its import parameter inside a **fast-serialization 0x5001
container**, which our request decoder does not read (imports came out empty →
Z_DOUBLE rc=2). Crucially: the SM59 destination A4H@GEN is set to **Classic
serializer**, yet the request is fast — because the actual serializer is
negotiated by the SERVER's logon-accept, not the client's preference. Our server
replays an accept borrowed from a fast-ser session, so it advertises fast; the
client obeys. The real .103, given the same Classic destination, advertised
classic and got classic parameters (0x0201/0x0203), which we decode.

So a fully-working conscious server needs either (A) to GENERATE a classic
logon-accept for the client's init (the init-dependent accept generation, which
also selects the serializer), or (B) full fast-serialization both ways (decode
0x5001 requests and encode fast-ser responses with multiref). The client side is
complete; the server's serializer negotiation is the open piece.
