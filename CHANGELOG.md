# Changelog

Milestone history for open-rfc-go. Dates are when the milestone was reached
against the live A4H test system (SAP_BASIS 793). Detailed wire findings live in
[`docs/discoveries/`](docs/discoveries/); the porting plan is in
[`docs/porting-plan.md`](docs/porting-plan.md).

## v0.1.0 — first tagged preview — 2026-08-22

The first version with a name and a number. Still a research preview: `0.x` means
the API may move, and classic RFC still has no transport encryption.

**The shipped binary is `orfc`** — **o**pen-**rfc**, so the attribution rides in
the name. It is both the CLI and, as `orfc mcp`, the MCP server — one binary, two
modes, like `vsp`. Plain `rfc` collides with the IETF sense of the word on a
`PATH` and says nothing about whose RFC it speaks; naming it after either mode
(`rfc-mcp`, `mcp-rfc`) would bake half the tool into the name; and a
`sap`-prefixed name was dropped because it mirrors SAP's own binaries
(`saprouter`, `sapcar`, `saplogon`) and so implies an origin the `NOTICE`
explicitly disclaims.

The whole family moved to that root, so there is one instead of two: `orfc`,
`orfc-srv`, `orfc-lab`, `orfc-sniff`, `orfc-viewer`, `orfc-ticketcatch`.

**`orfc-srv` is new** — the server front door, in either role a destination can
address:

```sh
orfc-srv -mode typet -listen :3300   # registered external server (SM59 type T)
orfc-srv -mode type3 -listen :3313   # an ABAP system            (SM59 type 3)
```

Point an SM59 destination at it and every `CALL FUNCTION … DESTINATION` lands in
the dispatcher, so a Z function module can be exercised against this
implementation **without a second SAP system**. Unknown modules raise
`FU_NOT_FOUND` rather than dropping the connection, so the request stays in the
capture. The type-T role previously had its own binary and the type-3 server was
reachable only inside the lab tool.

Getting it running against a system is now written down:
[`docs/quickstart-a4h.md`](docs/quickstart-a4h.md).

### Serialization, mapped end to end

All four of SAP's serializers can be selected on demand, and what each puts on
the wire is recorded. The controlling fact was not where anyone expected: the
destination has **two** independent knobs and the second overrides the first,
which is why a destination can display *"Fast serializer"* while storing a value
that means something else. Details in
[`docs/discoveries/serializer-selection.md`](docs/discoveries/serializer-selection.md).

With that settled, a controlled differential — vary one parameter, hold the rest,
capture both ends — produced the **fast serializer's record grammar**:

- framing is **tag-dependent**; there is no single length rule
- `INT4` is little-endian and fixed-width; `char`, `STRING` and `XSTRING` cost
  **one byte per unit** — not UTF-16, and not padded to the declared width
- the version handshake is deterministic and negotiates `FAST_SER_VERS = 3`
- payloads above **512 bytes are compressed**, and that is intrinsic to the
  serializer rather than a switchable transport feature

Decoded, not produced. The client negotiates classic, so none of the fast
serializer's compression has ever applied to the client leg.

### Roles, written down

[`docs/role-state-machines.md`](docs/role-state-machines.md) records who may send
what and when: the client setup machine's transitions with the wire rule behind
each invariant, all seven server roles and where each gets its replies, both
handshake shapes, and the keepalive rule — answer every ping, never answer a
pong, never parse an eight-byte frame as a record. Forgetting it stalls a
conversation rather than failing it, which reads like a decode bug and is not one.

Internally the roles now share one frame classifier. The eight keepalive bytes
had been declared twice under different names, which is exactly how one role gets
a fix the others miss.

### Also in this release

- `internal/fastser` decodes the record grammar: type descriptors, field names,
  `char`, `INT4`, `STRING`, and the end marker, with a coverage count so "how
  much of this do we actually model" is answerable rather than assumed
- the delta manager is understood: it elides on the **response** side, and its
  degenerate full-table form is what a server may always emit — so it never
  blocked the server track
- classic is complete for the synchronous path, re-verified live: scalar
  `STRING` and `XSTRING`, and deep structures carrying both

## Client — ADT REST over classic RFC — 2026-08-21

A real ADT REST request now travels through the classic-RFC tunnel:
`SADT_REST_RFC_ENDPOINT` answers `GET /sap/bc/adt/discovery` with HTTP 200 and
a 299 KB `application/atomsvc+xml` body, and an ADT source read returns the
program text with its `ETag` and `Last-Modified` headers. No ICF, no HTTP port.

- **Root cause of the earlier failure: line-wrapped base64.** The ABAP xRFC
  serializer emits an XSTRING cell as base64 broken every 76 columns with a
  bare LF. Both xRFC decoders validated the raw cell text, so every XSTRING
  longer than 57 bytes failed with "non-canonical base64" — which is every
  real HTTP body. `unwrapBase64` now joins the lines before the canonicality
  checks, which are otherwise unchanged (spaces, a non-standard alphabet and
  non-zero padding bits are still rejected). See `docs/provenance.md`.
- **`TFDIR-FMODE` has two remote values, not one.** `'R'` is a remote-enabled
  module; `'X'` is a remote-enabled module whose interface is basXML-capable —
  SAP flags every FM carrying deep/nested parameters that way, including
  `SADT_REST_RFC_ENDPOINT` and `SADT_PROTECTED_DISCOVERY`.
  `RFC_GET_FUNCTION_INTERFACE` returns `FMODE` verbatim in `REMOTE_CALL` and
  sets `REMOTE_BASXML_SUPPORTED` exactly when it is `'X'`. `rfc search` and the
  MCP tool listing filtered on `FMODE = 'R'` and so hid those modules
  entirely; both now accept `'R'` and `'X'`.

## Client — recursive (nested) metadata — 2026-08-20

`rfc.Client` can now call and describe function modules whose parameters are
**nested**: a DDIC structure with a `STRU` or `TTYP` component, or a table type
passed as a non-TABLES parameter.

- The flat source (`RFC_GET_STRUCTURE_DEFINITION`) returns overlapping
  `RFC_FIELDS` rows for such a type and was rejected outright — e.g.
  `TPDAPI_TEST_DEBUGGER` failed on `E_TAB_MSG` (`TPDAPI_TAB_MSG` →
  `TPDAPI_STR_MSG`, which carries `CONTEXT`/`PARAMS`/`CALLBACK` as `STRU` and
  `T_PAR` as `TTYP`) with "RFC_FIELDS TABNAME overlaps its preceding field".
- Those parameters now resolve through `RFC_METADATA_GET` (DEEP) →
  `metadata.Normalize` → a cached `metadata.Graph`, and (de)serialize with the
  recursive xRFC codec in `internal/xrfc`. Values stay native Go, so
  `Client.Call`'s contract is unchanged; `DescribeTool` renders a nested
  structure as a nested JSON-Schema object and a nested table as an array.
- Simple interfaces keep the flat fast path untouched; the graph is fetched
  lazily and its failure is cached, so a system without `RFC_METADATA_GET`
  behaves exactly as before.
- Also fixed: a structure whose DDIC length includes trailing alignment fill
  (`RFC_METADATA_PARAMS` is 464 in DDIC, 462 on the wire) no longer fails the
  fixed-width codec.

## Server track (M8) — 2026-08-19

open-rfc-go now speaks classic RFC **as the server**, not only the client.

- **A real ABAP program runs fully green against our Go server.** `ZLOCAL_RFC_TEST`
  issues six parametrized calls — RFC_PING, RFC_SYSTEM_INFO, STFC_CONNECTION (with
  its server→client callback), STFC_STRUCTURE, RFC_READ_TABLE (17 cols × 2 rows),
  STFC_STRING — and every one returns `rc=0`.
- **Every SM59 test button passes green** from one Go endpoint — Connection Test,
  Unicode Test, and Fast Serialization Test — across three serialization modes.
- Proven end to end: NI framing, gateway record, CPIC logon-accept **generated**
  from the client's init, per-session token mirroring (conversation id + RFC GUID
  in both wire byte orders), the RSRFCPIN ping dance, connection-pool re-logon,
  NI keepalives, and a mode-aware content-addressed responder that keys reply
  scripts by `acceptLen|FUNCTION`.
- Request decoding needed no new code: the existing classic decoder reads live
  ABAP fast-serialized requests with zero errors.
- New: `internal/rfcserver` (ServeReplay / ServeSmart / ServeContentAddressed),
  `internal/sniffer` per-connection tagging + raw-tee, and `cmd/orfc-lab`, a
  multi-protocol endpoint (type 3 sniff/replay/smart/content, HTTP, WebSocket).
- Known next step: a fast-ser codec that **generates** responses from values (so
  it works for inputs never captured), then real Go/JS function implementations
  behind a bridge adapter. See [`docs/discoveries/0001`](docs/discoveries/0001-live-type3-server.md).

## Client (M1–M6) — through 2026-08

- **M5 — first live call.** `STFC_CONNECTION`, `STFC_STRUCTURE`, `RFC_READ_TABLE`
  succeed against A4H; ABAP exceptions surface as typed `*rfc.ABAPException`.
- **M7 — public `rfc` facade.** `rfc.Open` / `Client.Call` with native Go values,
  a metadata cache, and an `errors.As`-able error taxonomy.
- **M6 — pool, lifecycle, SAProuter, SOCKS5** transport wiring.
- **M3b — deep xRFC codecs**; **M4 — metadata**; **M3 — value codecs**;
  **M2 — bounded reader, APPC, CPIC, RFCPRO, gateway**; **M1 — NI framing**.

Milestones are staged so each is verifiable against real bytes before the next
begins.
