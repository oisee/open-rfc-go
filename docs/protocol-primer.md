# Protocol primer

What classic RFC does on the wire, layer by layer.

Every claim here is cited into the upstream TypeScript at commit `847036d`
(paths without a prefix are `../open-rfc/<path>`). Where the source does not say
something, this document does not either — see
[Where this document stops](#where-this-document-stops).

Classic RFC has **no public specification**. Everything below was learned by
implementing it, and the evidence for each fact is ranked in
[`architecture.md`](architecture.md#evidence-hierarchy). Read
[`recurring-bug-class.md`](recurring-bug-class.md) before treating any length
here as a rule.

---

## The stack

```
  ┌──────────────────────────────────────────────┐
  │  values and metadata                         │  what a parameter means
  ├──────────────────────────────────────────────┤
  │  RFCPRO field headers                        │  tag + length per field
  ├──────────────────────────────────────────────┤
  │  CPIC field chains                           │  logon and call fields
  ├──────────────────────────────────────────────┤
  │  APPC conversation records                   │  who talks next, fragmentation
  ├──────────────────────────────────────────────┤
  │  gateway handshake                           │  capability negotiation
  ├──────────────────────────────────────────────┤
  │  NI record framing                           │  where a record ends
  ├──────────────────────────────────────────────┤
  │  TCP                                         │
  └──────────────────────────────────────────────┘
```

Each layer only knows about the one below it. That is not architectural
preference — it is why the layers can be ported and tested one at a time.

---

## 1. NI — record framing

**Job: say where a record ends.** TCP gives you a byte stream with no boundaries;
every layer above needs whole records.

An NI record is a four-byte big-endian length followed by exactly that many
payload bytes:

```
  00 00 00 03   52 46 43
  └── length ┘  └ "RFC" ┘
```

That is the entire layer. It carries no semantics — no type, no version, no
checksum.

Three consequences drive the implementation:

**Reads have no relationship to records.** A record may arrive one byte at a
time, four records may arrive in one read, and the length prefix itself may be
split across two reads. A decoder that assumes four contiguous bytes works on a
LAN and fails on a slow link.

**The length is attacker-controlled.** A peer can advertise 4 GiB. The bound must
be checked *before* allocating, which is why `DEFAULT_MAX_NI_PAYLOAD_LENGTH` is
256 MiB (`protocol/ni.ts:5`) and why exceeding it is fatal to the connection
rather than skippable — once you have read a length you cannot trust, you no
longer know where the next record starts.

**A partial record can hold credentials.** Retained buffers are zeroed on reset.
In Go that is weaker than it looks; see [`../SECURITY.md`](../SECURITY.md).

Ported as `internal/ni`. Vectors: `conformance/testdata/vectors/ni-framing.v1.json`.

---

## 2. Gateway handshake — capability negotiation

**Job: agree on what both sides can do, before anything else happens.**

RFC connects to the **gateway** on `3300 + sysnr` — not the dispatcher on
`32NN`. The connection opens with a 64-byte normal-client record
(`protocol/gateway.ts:5-7`):

| Field | Value |
|---|---|
| `GATEWAY_NORMAL_CLIENT_LENGTH` | `64` |
| `GATEWAY_PROTOCOL_VERSION` | `2` |
| `GATEWAY_NORMAL_CLIENT_REQUEST` | `3` |

The client requests a set of capability bits and the gateway replies with the
ones it accepts — `ErrorInfo`, `Ping`, `ConnectionExtendedInfo`, `CodePage`,
`ExtendedInitOptions`, `DistributedTrace`
(`client/direct-cpic-session.ts:777-782`).

**The reply is checked, and missing capabilities are fatal.** If
`ExtendedInitOptions` or `CodePage` is absent, the session fails closed
(`:798-802`) rather than continuing in a mode it cannot serialize correctly.
This is the general pattern: *a capability that cannot be verified must refuse
rather than guess.*

---

## 3. APPC — the conversation

**Job: track whose turn it is to speak, and split messages that do not fit.**

Classic RFC is a half-duplex conversation with explicit turn-taking. APPC records
carry that state.

| Constant | Value | Citation |
|---|---|---|
| `APPC_PROTOCOL_VERSION` | `0x06` | `protocol/appc.ts:8` |
| `APPC_COMMON_HEADER_LENGTH` | `48` | `:9` |
| `APPC_RECORD_HEADER_LENGTH` | `80` | `:11` |
| `MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH` | `28_000` | `:19` |
| `MAX_APPC_ASYNC_SENDS_BEFORE_SYNC` | `21` | `:21` |

After the gateway handshake the client sends an APPC `Initialize` control record
(`client/direct-cpic-session.ts:812-813`) carrying the extended initialize
options and partner parameters (341 and 144 bytes respectively).

**Fragmentation is where correctness gets subtle.** A message larger than 28,000
bytes of application data is split across records. If a send fails partway
through, the conversation state depends on *which* fragment failed:

- fragment 0 fails → transmission state **`Unknown`**
- a later fragment fails → transmission state **`Partial`**

In both cases the transport is closed before the error escapes, and **the call is
never re-sent** (`client/direct-cpic-session.ts:565-577`). This is the ambiguous
send rule: the replay policy has exactly one value, `Never`, expressed as a
literal type so no caller can widen it. A retried call that already reached the
backend can commit an LUW twice.

---

## 4. CPIC — field chains

**Job: carry named fields — logon data, function name, parameters.**

CPIC serializes tag/length/value fields into chains. Tags identify what a field
is (`Destination = 0x0006`, `ClientAddress = 0x0007`, and others in
`protocol/cpic.ts`).

Three bounds, all configurable per connection, defaulting to
(`protocol/cpic.ts:21-23`):

| Bound | Default |
|---|---|
| field length | 256 MiB |
| chain length | 256 MiB |
| field count | 100,000 |

These are **defaults, not protocol limits** — the distinction matters. A limit
that a peer's release exceeds is a configuration problem, not a malformed
message.

### The initial logon reply, and why it is the cautionary tale

The logon reply is where the recurring bug cost three weeks. An early parser
pinned the exact byte lengths of text fields in the reply — faithful to the one
system it was developed against, and wrong for any host whose name is a different
length. The failure surfaced as `RFC_INVALID_PROTOCOL`, which was read as "SAP
rejected the password", so working authentication code was replaced repeatedly.
The password handling had been correct the whole time.

The current implementation parses the reply with a **grammar** rather than a
layout, and the test names are the historical record
(`test/cpic-initial-logon-grammar.test.ts`):

- *"the grammar admits every graph the retired allowlist enumerated"*
- *"the grammar admits the 2026-08-05 S/4HANA reply that was refused"*
- *"the grammar admits the 2026-08-04 NetWeaver reply that was refused"*
- *"a text coordinate parses identically at every legal length"*
- *"reordering, duplication and truncation still fail closed"*
- *"a nonzero logon status is preserved as a rejection, not a protocol error"*

Note what that last one protects: an authentication *rejection* must reach the
caller as a rejection, not be reclassified as a protocol fault. That confusion is
what made the original bug take three weeks to find.

### Passwords

The password field is scrambled with a fixed 64-byte table and a random 4-byte
seed (`protocol/password-scramble.ts:3-10`). Input is constrained to printable
ASCII `0x20`–`0x7e`, at most 40 bytes, and anything outside that is refused as
*"outside the proven ASCII baseline"* rather than transcoded — a refusal to guess
at an encoding that was never verified.

**This is not encryption.** Scrambling is obfuscation with a published table.
Without SNC — which neither implementation supports — credentials cross the
network protected only by whatever the underlying transport provides, which for
plain TCP is nothing.

---

## 5. RFCPRO — field headers

**Job: delimit fields inside an RFC payload.**

Two header widths (`protocol/rfcpro.ts:3-5`, `:41-45`):

| Form | Width | When |
|---|---|---|
| compact | 4 bytes | length ≤ `0xfffe` |
| extended | 8 bytes | length ≥ `0xffff` |

A compact length of `0xffff` is a **sentinel**, not a length: it means "the real
length is in the extended header". Maximum value length is `0x7fff_ffff`.

---

## 6. Values and metadata

**Job: turn bytes into typed values, and know what the types are.**

You cannot decode a function module's parameters without knowing its interface,
so a client bootstraps metadata first — `RFC_METADATA_GET` where available, or
`RFC_FUNINT`/`DDIF_FIELDINFO_GET` rows otherwise.

`RFC_FUNINT_UNICODE_ROW_LENGTH = 402` (`protocol/classic-rfc.ts:16`) is the
clearest illustration of the project's decoding discipline. It is used as a
**lower bound plus a read window**, never as an equality check:

> *"The row width is a property of the peer's release, not of the wire format:
> later releases append fields to RFC_FUNINT, and one profile is already
> evidenced declaring a 404-byte row. Bound the row below by the stable prefix
> this decoder consumes and ignore anything appended after it… A short row is
> still refused — completing one with ABAP initial bytes would invent values the
> peer never sent."* (`:554-563`)

Both halves matter. Accept more than you have seen; never fabricate what you have
not received.

Value encodings include packed decimal/BCD, IEEE 754 decimal float, compact
temporal types, and two xRFC XML serializations (flat and recursive). Compact
temporal values carry per-type maxima
(`values/classic-temporal.ts:69-111`) — `UTCLONG` at 3.16e18, `CDAY` at 366 —
every one below its width's signed ceiling.

---

## Putting it together: one call

```
  TCP connect  ──────────────►  gateway, port 3300 + sysnr
  gateway normal-client record (64 bytes, version 2, request 3)
                               ◄──  accepted capability bits
  check ExtendedInitOptions and CodePage, or fail closed
  APPC Initialize control record
  CPIC logon fields (client, user, scrambled password, language)
                               ◄──  logon reply, parsed by grammar
  CPIC call fields (function name, parameters as RFCPRO fields)
                               ◄──  reply fields, or an ABAP exception
```

Every arrow is inside an NI record. Order verified in
`client/direct-cpic-session.ts:767-851`.

---

## Where this document stops

Deliberately absent, because the source does not state them and inventing them
would be exactly the documented failure mode:

- **Complete byte layouts** of the gateway, APPC, and CPIC records. The
  constants above are cited; the field-by-field layouts live in the code and in
  [`surface/protocol.md`](surface/protocol.md), which cites each one.
- **Why** any given constant has its value. `APPC_PROTOCOL_VERSION = 0x06` is
  what the wire requires; the source does not say why, and neither does this.
- **Non-Unicode systems, SNC, WebSocket RFC.** Not implemented, therefore not
  described.

For signatures, constants, and error messages of the unported layers, the cited
inventories in [`surface/`](surface/) go deeper than this page. For the wire
itself, the upstream source is the authority — and even it ranks its own claims,
which is worth understanding before relying on any of them.
