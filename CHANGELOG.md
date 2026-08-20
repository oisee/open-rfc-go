# Changelog

Milestone history for open-rfc-go. Dates are when the milestone was reached
against the live A4H test system (SAP_BASIS 793). Detailed wire findings live in
[`docs/discoveries/`](docs/discoveries/); the porting plan is in
[`docs/porting-plan.md`](docs/porting-plan.md).

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
  `internal/sniffer` per-connection tagging + raw-tee, and `cmd/rfc-lab`, a
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
