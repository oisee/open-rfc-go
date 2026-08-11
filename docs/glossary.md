# Glossary

SAP and protocol vocabulary, defined as this codebase uses it. Where a term maps
to code, the citation is into the upstream TypeScript at commit `847036d`.

---

### ABAP

SAP's application programming language. Function modules that RFC calls are
written in it. Relevant here mainly through **ABAP initial values** — the
type-specific "empty" values (`0`, spaces, zeroed bytes) that a decoder must
never invent to complete a short record.

### APPC

*Advanced Program-to-Program Communication.* The conversation layer above NI:
control records, data records, and the fragmentation rules for messages larger
than one record. Carries a 48-byte common header and an 80-byte record header
(`protocol/appc.ts:9-11`). Fragments cap at 28,000 bytes of application data
(`:19`), and at most 21 asynchronous sends may precede a synchronising one
(`:21`).

### BAPI

*Business Application Programming Interface.* A conventionally-named, documented
subset of ABAP function modules. Not a protocol concept — the wire cannot tell a
BAPI from any other function module, which is why nothing below the client layer
mentions the term.

### BCD

*Binary-coded decimal.* Packed-decimal encoding for ABAP `P` fields: two decimal
digits per byte with a sign nibble. Implemented in `values/classic-bcd.ts` and
`values/packed-decimal.ts`. A place where JavaScript number semantics and Go's
differ enough to need care.

### CPIC / CPI-C

*Common Programming Interface for Communications.* The field-chain layer:
tag/length/value fields carrying logon data, function names, and parameters.
Field, chain, and count limits are configurable with defaults of 256 MiB, 256
MiB, and 100,000 (`protocol/cpic.ts:21-23`).

### Classic RFC

SAP's original synchronous RFC protocol, as opposed to WebSocket RFC or the
HTTP-based alternatives. Stateful, binary, negotiated, and **without a public
specification** — the single fact that shapes this codebase.

### Client (SAP client)

A three-digit tenant identifier within an SAP system, e.g. `100`. Unrelated to
"client" in the network sense. Validated as 1–3 digits and zero-padded to three
(`compat/connection-parameters.ts:228-232`).

### DDIC

*Data Dictionary.* SAP's central metadata repository, describing types,
structures, and tables. `DDIF_FIELDINFO_GET` is the function module that returns
structure field information; see `metadata/ddif-fieldinfo.ts`.

### Destination

A named, reusable connection configuration. In this codebase a destination owns
identity and generation: what invalidates cached metadata, and what a connection
belongs to (`src/destination/`).

### Dispatcher

The SAP work-process dispatcher, conventionally on port `32NN`. **Not what
classic RFC connects to** — RFC dials the gateway on `33NN`. Pinning the
dispatcher range is one of the six documented recurring bugs.

### Function module

A callable ABAP procedure, e.g. `STFC_CONNECTION` (the conventional
connectivity smoke test) or `RFC_METADATA_GET`. Parameter names are
case-sensitive and carry exact metadata casing: `REQUTEXT`, never `requtext`.

### Gateway

The SAP process that accepts RFC connections, conventionally on port
`3300 + sysnr` (service name `sapgwNN`). The connection opens with a 64-byte
normal-client record, protocol version 2, request type 3
(`protocol/gateway.ts:5-7`).

### Generation

A monotonic marker for a destination's configuration state. Bumping it
invalidates cached metadata and connections without mutating shared structures —
the mechanism that makes "immutable descriptors" and "reconfigurable
destinations" coexist.

### LUW

*Logical Unit of Work.* SAP's transaction boundary. Matters for RFC because
whether a failed call left an LUW open decides whether it is safe to retry.
Related: **ambiguous send** — a call whose outcome is unknown because the failure
happened after bytes reached the wire. This codebase never replays one; the
replay policy has exactly one value, `Never`.

### Message server

The process that selects an application server from a logon group, conventionally
reached by the service name `sapms<SID>`. This codebase resolves that name via
`/etc/services` and **refuses to guess a numeric port**
(`transport/message-server-resolver.ts:217-232`).

### NI

*Network Interface.* SAP's record-framing layer: a four-byte big-endian payload
length followed by that many bytes. Everything above TCP is inside an NI record.
Ported as `internal/ni`.

### RFC

*Remote Function Call.* SAP's mechanism for invoking a function module across
systems.

### RFCPRO

The field-header encoding used within RFC payloads. Compact headers are 4 bytes;
a length of `0xffff` is a sentinel selecting the 8-byte extended form
(`protocol/rfcpro.ts:3-5`).

### RFC_FUNINT

The internal function-interface metadata structure returned during metadata
bootstrap. Its Unicode row is **at least** 402 bytes; a 404-byte row is already
evidenced, and later releases may append more. The decoder bounds below and
ignores the remainder (`protocol/classic-rfc.ts:16`, `:554-563`).

### RFC_METADATA_GET

The function module returning function-interface metadata in one call. See
`metadata/rfc-metadata-get.ts`.

### SAProuter

An SAP-provided TCP relay, conventionally on port `3299`
(`transport/saprouter-route.ts:4`). **Routing, not security** — it provides
neither confidentiality nor peer authentication.

### SNC

*Secure Network Communications.* SAP's transport security layer for classic RFC.
**Not implemented** in either implementation; `snc_mode` is rejected outright
rather than silently ignored. Its absence is why classic RFC here has no
transport encryption.

### `sysnr`

The two-digit **system number** identifying an instance on a host. Determines the
conventional ports: gateway `3300 + sysnr`, dispatcher `3200 + sysnr`. One or two
decimal digits, defaulting to `00` (`compat/connection-parameters.ts:219-221`).

### `STFC_CONNECTION`

The conventional read-only RFC connectivity test function module. Milestone 5's
target: the first call this port can attempt against a real system.

### Unicode / non-Unicode systems

SAP systems come in both flavours, and they serialize differently on the wire.
This project targets Unicode systems; the `RFC_FUNINT_UNICODE_ROW_LENGTH` name
carries that assumption explicitly.

### xRFC

An XML-based serialization used for some RFC values, distinct from the classic
binary form. Both exist, and both are implemented (`values/classic-xrfc.ts`,
`values/recursive-xrfc.ts`). "Recursive" variants handle nested and
self-referential structures under explicit bounds.

---

## Terms from this project, not from SAP

### Conformance vector

A language-neutral wire fact: hex in, hex or a named error out, plus a stated
reason it matters. Lives in `conformance/testdata/vectors/` so more than one
implementation can check the same bytes.

### Provenance header

The SPDX comment at the top of a ported file naming its upstream source file,
commit, and modifications. Required by Apache-2.0 §4(b), and used as the index
for propagating upstream fixes.

### Recurring bug class

"A decoder that memorises what one system happened to send." Documented in
[`recurring-bug-class.md`](recurring-bug-class.md) with six instances. The tell
is a comparison against a literal sitting on something that varies by peer,
release, or configuration; the symptom is "works on my system".

### Surface inventory

A cited, mechanical map of an unported upstream layer — signatures, constants,
error messages, invariants, test-asserted facts. Porting input, not truth. See
[`surface/`](surface/).
