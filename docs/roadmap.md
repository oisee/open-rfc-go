# open-rfc-go roadmap: beyond and better than upstream

Design report (2026-08-19), grounded in the actual code. Milestones M1–M5 are
done and verified live against A4H; M6 (pool, lifecycle, SAProuter route codec,
SOCKS5) has landed.

> **Update — 2026-08-20.** Several P0/P1 items below are now **done** and
> verified live against A4H (.105): the public **`rfc` package** (`Open`/`Call`,
> native values, typed `*ABAPException`, metadata cache), **deep/nested xRFC**
> live round-trip (`STFC_DEEP_STRUCTURE`/`STFC_DEEP_TABLE`, STRING/XSTRING), and
> **every scalar type** (incl. STRING/XSTRING, DATE/TIME, packed DEC/TIMESTAMP,
> FLOAT, UTCLONG). The client now calls essentially any FM. Two new tracks are
> added below: an **RFC tool surface** (`rfc` CLI + `saprfc mcp`) and
> **RFC callback** support.

## New tracks (added 2026-08-20)

| Rank | Item | Rationale |
|---|---|---|
| ✅ **P0 done** | RFC tool surface — `rfc` CLI + `saprfc mcp` MCP server | Built and live-verified. `cmd/saprfc`: info/describe/search/call/read-table/ping. `cmd/rfc-mcp`: JSON-RPC-2.0 stdio server, dependency-free; generic tools rfc_info/ping/describe/search/read_table/call **plus autodiscovery** — `--expose`/`--hide` masks turn matching RFC-enabled FMs into real per-FM MCP tools (inputSchema = the FM's interface). `describe`/per-FM tools share one EXID→JSON-Schema mapper (`Client.DescribeTool`). Config via `.rfc.json` (named systems + expose/hide/readOnly) + env + flags (flags win); shared `cmd/rfctool`. Core `rfc` stays dependency-free; the `cmd/*` set is an extractable subproject. See the design notes below. |
| ✅ **P1 done** | RFC **callback** (server→client, DESTINATION 'BACK') on the client | Done: `Session.CallWithCallbacks` runs a re-entrant loop that dispatches inbound callback requests to per-FM handlers (`rfc.Destination.Callbacks`) and replies until the response arrives. Verified live vs A4H (directly, through the sniffer, and a real .105↔.103 callback via A4H@SNIFF), `STFC_CONNECTION_BACK`/`ZSTFC_CONNECTION_BACK` with NRBACK=1..3. |
| 🟡 **P1 first cut** | Safety gate for write FMs in `saprfc mcp` beyond `--read-only` | Done (tier 1): `--safe` blocks/hides heuristically-write FMs (mutating name verbs) and `BAPI_TRANSACTION_COMMIT` (unless `--allow-commit`); tools carry `readOnlyHint`/`destructiveHint`. Remaining: the interface-signal + LLM-deep-dive tiers and the conditional per-parameter policy file. Full design: [`docs/design/write-fm-safety.md`](design/write-fm-safety.md). |

### RFC tool surface — design notes & open items (thought through 2026-08-20)

Shipped shape: **generic describe+call** (RFC has tens of thousands of FMs — can't
pre-generate a tool per FM like odata-mcp does per entity), **plus opt-in
autodiscovery** that renders a *curated* subset as real per-FM MCP tools. One
`RFC-interface → JSON-Schema` mapper backs both `saprfc describe` and the per-FM
tools. Three config sources (flags > env > `.rfc.json`), a shared `cmd/rfctool`,
core stays dependency-free.

Open items / decisions still to make:
- **Per-FM tool output schema.** Today per-FM tools carry only `inputSchema`; add
  `outputSchema` (we already compute it) for MCP structured output where clients
  support it.
- **Refresh / `tools/list_changed`.** The exposed set resolves once and caches;
  no notification if FMs appear/change. Fine for stdio; revisit for long-lived HTTP.
- **Startup cost.** Autodiscovery does one `DescribeTool` (interface + struct
  fetches) per exposed FM at first `tools/list`; large green-lists are slow.
  Options: concurrency, a `--max` cap (done), or lazy per-tool describe.
- **Name collisions & namespaces.** FM names are `RS38L-NAME` (≤30 chars, so under
  MCP's 64); `/NS/NAME` → `_NS_NAME`. Two different FMs can still sanitize to the
  same tool name — need a collision policy (suffix) and a reverse map (kept).
- **Description quality.** Uses FM short text + exceptions; could pull the FM long
  documentation and per-parameter texts for richer tool descriptions.
- **Transports.** stdio only; add HTTP/SSE + streamable-HTTP (odata-mcp has them).
- **Config ergonomics.** `.rfc.json` mirrors `.vsp.json`; consider an
  `--expose-file` list, SAProuter/SOCKS fields, and a redacted `rfc config`/`rfc
  systems` command. Never require the password in the file (env supported).
- **Extraction.** `cmd/saprfc` (CLI + `saprfc mcp`) and `cmd/rfctool` depend only on public
  `rfc`; extraction = move them to a new module requiring open-rfc-go.
- **Library `ReadTable`.** `cmd/rfctool.ReadTable` is a candidate to promote into
  the public `rfc` package (roadmap P1 "typed ReadTable").

The whole wire implementation lives under `internal/`; the public `rfc/` package
is now real and consumable (the P0 that unblocked all of the above).

The wire-level port is in excellent shape and, in several respects
(compile-enforced layering, race-proven connection ownership, type-erased guard
classes that upstream needed at runtime, pervasive fuzzing), already **more
trustworthy than the TypeScript original**. The gap between "correct codecs" and
"a library a Go team adopts" is now almost entirely *above* the wire.

## Ranked roadmap

| Rank | Item | Rationale |
|---|---|---|
| ✅ **P0 done** | Public `rfc` package (typed `Client`/`Destination`/`Call`) | Landed and live-proven: `rfc.Open`/`Client.Call`, native values, metadata cache. Unblocked the tool surface, xRFC wiring, and scalar coverage. |
| ✅ **P0 done** | Surface the ABAP error envelope through the call boundary | `Client.Call` now returns a typed `*ABAPException` (key/message/type/number/class), `errors.As`-able; live-verified. |
| ✅ **P0 done** | Error taxonomy for the public package (`errors.Is/As` tree) | One documented set in `rfc`: `ErrClosed`, `ErrNotAuthenticated`, `ErrLogonRejected`, `ErrProtocol`, `ErrTransport`, `ErrUnknownParameter`, `ErrPoolExhausted`, `ErrTimeout`, plus `*ABAPException` via `errors.As`. Internal sentinels (pool/lifecycle/transport, context deadline) are translated onto it at the call boundary, with the original wrapped for diagnostics. |
| ✅ **P1 done (cache)** | Metadata repository runtime (cache + in-flight coalescing) | Function-interface and structure-definition caches landed; in-flight coalescing still open. |
| ✅ **P1 done** | Deep/nested xRFC live verification (`STFC_DEEP_TABLE/STRUCTURE`) | Verified live on .105 (2026-08-20): deep structures & tables round-trip incl. STRING/XSTRING; xRFC codec wired into `rfc.Call`. |
| **P1** | Codegen: DDIC metadata → typed Go structs (`rfcgen`) | The normalized graph + `RfcStructureDefinition` already exist; the single highest-leverage ergonomic differentiator over node-rfc/upstream. |
| **P1** | Observability hooks (`log/slog` + OpenTelemetry) | None today. Greenfield; the "no I/O under a lock" rule makes a clean hook contract easy. |
| **P1** | Typed `ReadTable` wrapper + message/`BAPIRET2` resolver | The two most-used diagnostic/data primitives; enabler for the debug-trace assistant. Now also folded into `cmd/rfc read-table`. |
| **P1** | Fuzz the runtime metadata row decoders | `DecodeDdIfDfiesRow`, `DecodeRfcFieldsRow`, `DecodeRfcFunctionInterfaceResult`, `classicrfc.DecodeResult` consume live server bytes but have no fuzz target — violates the port's own "fuzz every decoder" rule. |
| ✅ **P1 done** | Wire SAProuter + SOCKS5 into the dial path | Wired: `rfc.Destination.Router` / `.SOCKS5` flow through `transport.DialWith`, so routed and proxied dialing is reachable from the public client. |
| **P2** | tRFC/qRFC/bgRFC transactional units | Valuable for integration; substantial new wire work (TID lifecycle, confirm/commit). |
| **P2** | Registered/inbound RFC server | Large, high-value, new protocol direction; see `docs/polyglot-rfc-server.md`. Do after the client API stabilizes. |
| **P2** | Zero-copy / pooled decode path | Up to four copies socket→field today; a measured optimization once benchmarked. |
| **P2** | Shared language-neutral conformance corpus (bidirectional) | Only `ni-framing.v1.json` exists; the "upstream owns the corpus" end-state needs an upstream change. |
| **P2** | Pure-Go SAProuter server | Compelling infra product, separate track; SNC is the hard blocker. |
| **P2** | Hybrid open-abap + RFC forwarding | Research direction; depends on a stable typed API and error round-trip. |
| **P2** | SNC (channel encryption) | Blocks secure production use; requires a GSS-API/crypto binding — the biggest single effort. |

## Correctness & robustness findings (from reading the code)

Already banked Go wins: `internal/`-only layering makes accidental API exposure
a compile error; thrown errors became wrapped sentinels read with `errors.Is`;
`context.Context` bounds every read; `go test -race` *proves* the one-goroutine-
owns-the-conn invariant; fixed-width types retire whole classes of upstream
guards.

Concrete gaps to fix, most in the P0 work:

1. **ABAP error envelope decoded then discarded.** `cpic.DecodedFunctionResultFields`
   carries `Outcome` + `Envelope` but `client.Call` maps only `Success`+`Fields`.
   The public `Call` should return a typed `*ABAPException` (key/message/type/
   number/class) and keep the raw envelope for diagnostics.
2. **No public error taxonomy** — define it in `rfc/` and wrap internal sentinels
   so `errors.Is` climbs cleanly.
3. **Credential memory safety is partial** — `LogonOptions.Password` is an
   immutable `string` that can't be zeroed; offer a `[]byte`/callback credential
   provider and document the GC caveat rather than implying safety.
4. **Fuzzing gap** on the four runtime-facing metadata/result decoders above.
5. **Recurring-bug-class discipline** (stable-prefix, tolerate-append) should be
   a shared helper + test-vector convention so it can't regress silently.
6. **`equalBytes`/`bytesRepeat` in `session.go` reimplement stdlib** — use
   `bytes.Equal`/`bytes.Repeat` (and `crypto/subtle` if a comparison ever touches
   a secret).

## The public `rfc` API shape (proposed)

```go
type Destination struct {
    Host, Client, User string
    Password  string           // or a CredentialFunc
    Service   string           // sapdpNN / gateway
    Router    string           // "/H/.../S/..." → internal/saprouter
    SOCKS5    *SOCKS5Options
    Pool      PoolConfig
    Tracer    Tracer           // OTel-shaped hook; nil = off
    Logger    *slog.Logger
}
func Open(ctx context.Context, d Destination) (*Client, error)

func (c *Client) Call(ctx context.Context, fn string, in Params) (Result, error)
type Params = map[string]any               // structs/tables as map/[]map
type Result struct { /* Get(name), Table(name) */ }

func Invoke[I, O any](ctx context.Context, c *Client, fn string, in I) (O, error) // typed, paired with rfcgen

type ABAPException struct { Key, MessageType, MessageClass, MessageNumber, Message string; Kind ExceptionKind }
func (e *ABAPException) Error() string

func (c *Client) FunctionInterface(ctx, fn string) (metadata.RfcFunctionInterface, error)     // cached
func (c *Client) StructureDefinition(ctx, name string) (rfctypes.RfcStructureDefinition, error) // cached
```

Principles: `Call` computes `requestedOutputs` from the resolved interface
(removing the most surprising wire fact); values are native Go (maps, `int64`,
decimal strings); the pool is transparent (`Client` wraps `pool.Pool[*Managed]`);
ABAP failures are `errors.As`-able.

## Performance (gate on a benchmark first)

A socket byte is copied up to four times before a caller sees a field: retained
decoder chunk (`ni/frame.go`), assembled payload (`ni/frame.go`), private reader
copy (`wire.go`), per-field copy (`wire.go`). Wins: a `NewReaderOwned` that skips
the defensive copy on the proven-owned internal path; `sync.Pool` for encode/
frame buffers; decode straight into structs on the typed `rfcgen` path; a ring
buffer for `transport.queue`. Benchmark a representative `STFC_STRUCTURE` +
large-table call before optimizing.

## Bottom line

Four items convert a faithful port into a differentiated product, all P0/P1:
**a real public `rfc` package, ABAP errors as typed errors, a metadata cache,
and codegen.** tRFC/qRFC, an RFC server (see `docs/polyglot-rfc-server.md`), and
SNC are the larger, later bets.

## Backlog: content-addressed response replay (mock RFC server)

`cmd/rfc-server -replay` today drives one recorded connection in strict
lockstep: it sends the Nth recorded server frame after the Nth client frame,
ignoring what was actually asked. That proves the server-side transport but
desyncs the moment a live client calls functions in a different order or with
different arguments.

The next step is a **content-addressed responder**: index a capture (or a
directory of captures) by the decoded request, and answer a live call by
matching it — not by position. Match tiers, most-specific first:

1. **FM name + all import/table parameters equal** — exact replay; safest.
2. **FM name + a subset of parameters** (a configurable key set, e.g. match
   `QUERY_TABLE` for `RFC_READ_TABLE` but ignore `ROWCOUNT`) — parameterized.
3. **FM name only** — return the last recorded response for that function; a
   loose fallback for stateless calls (`RFC_PING`, `RFC_SYSTEM_INFO`).

Miss policy is explicit: a configurable order of tiers, then either a recorded
default or a `FU_NOT_FOUND`/`SYSTEM_FAILURE` exception via the existing
`rfcserver` encoders. This turns a capture into a faithful mock of a real
system — for tests, demos, and offline development — and is the natural bridge
from replay to a Dispatcher that generates responses.

Prereqs already in place: `rfcserver.DecodeCutFunctionRequest` (request key),
`rfcserver.EncodeCutFunctionResponse` / `EncodeCutFunctionExceptionResponse`
(answers), and the `-replay` transport loop (accept + NI + gateway + APPC). The
open work is the request→response index, the tiered matcher, and reproducing the
CPIC init/logon-accept per fresh connection (the one part replay currently gets
for free by echoing recorded bytes).
