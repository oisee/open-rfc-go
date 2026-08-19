# Changelog

Milestone history for open-rfc-go. Dates are when the milestone was reached
against the live A4H test system (SAP_BASIS 793). Detailed wire findings live in
[`docs/discoveries/`](docs/discoveries/); the porting plan is in
[`docs/porting-plan.md`](docs/porting-plan.md).

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
