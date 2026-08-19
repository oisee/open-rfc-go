# open-rfc-go roadmap: beyond and better than upstream

Design report (2026-08-19), grounded in the actual code. Milestones M1–M5 are
done and verified live against A4H; M6 (pool, lifecycle, SAProuter route codec,
SOCKS5) has landed. The whole implementation lives under `internal/`; the public
`rfc/` package is still an empty placeholder.

The wire-level port is in excellent shape and, in several respects
(compile-enforced layering, race-proven connection ownership, type-erased guard
classes that upstream needed at runtime, pervasive fuzzing), already **more
trustworthy than the TypeScript original**. The gap between "correct codecs" and
"a library a Go team adopts" is now almost entirely *above* the wire.

## Ranked roadmap

| Rank | Item | Rationale |
|---|---|---|
| **P0** | Public `rfc` package (typed `Client`/`Destination`/`Call`) | The full resolve→encode→call→decode pipeline exists only in tests (`live_explore_test.go`); there is no consumable API. Prerequisite for MCP, debug-trace, codegen, RFC-server. |
| **P0** | Surface the ABAP error envelope through the call boundary | `cpic` decodes a rich `rfcerr.Envelope` (exceptions, MESSAGE, runtime, class-based) but `client.CallResult` discards it — a failed FM currently reduces to `Success:false`. |
| **P0** | Error taxonomy for the public package (`errors.Is/As` tree) | Each internal package has its own sentinels; consumers need one documented set: `ErrLogonRejected`, `*ABAPException`, `ErrProtocol`, `ErrTransport`, `ErrTimeout`, `ErrPoolExhausted`, `ErrClosed`. |
| **P1** | Metadata repository runtime (cache + in-flight coalescing) | Every typed call needs interface + structure metadata; without a cache each call re-fetches. Upstream had it (`repository-runtime.ts`); deferred here. |
| **P1** | Deep/nested xRFC live verification (`STFC_DEEP_TABLE/STRUCTURE`) | Recursive codecs are ported + fuzzed but the plan flags deep xRFC as the remaining unproven-live M3b work. Cheap to close. |
| **P1** | Codegen: DDIC metadata → typed Go structs (`rfcgen`) | The normalized graph + `RfcStructureDefinition` already exist; the single highest-leverage ergonomic differentiator over node-rfc/upstream. |
| **P1** | Observability hooks (`log/slog` + OpenTelemetry) | None today. Greenfield; the "no I/O under a lock" rule makes a clean hook contract easy. |
| **P1** | Typed `ReadTable` wrapper + message/`BAPIRET2` resolver | The two most-used diagnostic/data primitives; enabler for the debug-trace assistant (`docs/rfc-assistance.md`). |
| **P1** | Fuzz the runtime metadata row decoders | `DecodeDdIfDfiesRow`, `DecodeRfcFieldsRow`, `DecodeRfcFunctionInterfaceResult`, `classicrfc.DecodeResult` consume live server bytes but have no fuzz target — violates the port's own "fuzz every decoder" rule. |
| **P1** | Wire SAProuter + SOCKS5 into the dial path | Both codecs are done and tested in isolation; `transport.Dial` still does a plain `net.Dial`, so routed/proxied dialing is unreachable from the client. |
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
