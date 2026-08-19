# Porting plan

The order below is chosen so that each milestone is independently verifiable
against bytes, and so that the layers whose knowledge is expensive to
rediscover come first.

Upstream is roughly 45k lines of TypeScript across ten layers. About 19k lines
(`src/protocol/`, `src/values/`, `src/metadata/`) are near-mechanical
translation; about 18k (`src/client/`, `src/lifecycle/`, `src/pool/`,
`src/destination/`) are a redesign, because most of their bulk is Promise,
`AbortSignal`, and `AsyncLocalStorage` plumbing that Go gets from goroutines
and `context.Context`; and about 7k (`src/compat/`) is dropped outright.

## Milestones

| # | Scope | Upstream | State |
|---|---|---|---|
| 1 | NI framing | `src/protocol/ni.ts` | **done** |
| 2 | APPC records, CPIC field chains, RFCPRO field headers, gateway handshake | `src/protocol/{appc,cpic,rfcpro,gateway}.ts` (+ `bytes`, `password-scramble`, `rfc-error-envelope`) | **done** |
| 3a | Scalar value codecs: packed decimal, decimal float, temporal, unicode-scalar | `src/values/{packed-decimal,decimal-float,classic-temporal,unicode-scalar}.ts` | **done** |
| 3b | Structure & xRFC codecs (classic-structure, classic-xrfc, recursive xRFC) | `src/values/{classic-structure,classic-xrfc,recursive-*}.ts` | **done** — flat structures/tables verified live; all three xRFC codecs (`classic-xrfc`, `recursive-xrfc`, `recursive-classic-xrfc`) ported with oracle tests + fuzz in `internal/xrfc` |
| 4 | Metadata decoders: `RFC_METADATA_GET`, `DDIF_FIELDINFO`, function/structure interface (+ `classic-rfc`) | `src/metadata/{recursive-metadata,recursive-parameter-index,rfc-function-interface,rfc-structure-definition,ddif-fieldinfo,rfc-metadata-get}.ts`, `src/protocol/classic-rfc.ts` | **done** |
| 5 | Transport over `net.Conn`; first live `STFC_CONNECTION` | `src/transport/ni-socket.ts`, `src/client/direct-cpic-session.ts` | **done** (live call verified 2026-08-19 against A4H, kernel 758) |
| 6 | Pool, session contexts, transactions, SAProuter, SOCKS5 | `src/pool/`, `src/lifecycle/`, `src/transport/` | **in progress** — connection pool (`internal/pool`), SAProuter route codec (`internal/saprouter`), SOCKS5 dialer (`internal/socks5`), and session lifecycle + pool integration (`internal/lifecycle`) done, Go-idiomatic; remaining: transaction units (tRFC/qRFC) and wiring router/socks5 into the dial path |

The metadata **repository runtime** (`src/metadata/repository-runtime.ts`) and
**structured diagnostics** (`src/diagnostics/structured-diagnostics.ts`) are not
mechanical ports: they are a generic async cache (AsyncLocalStorage, Promise-based
adapter, in-flight coalescing, eviction) and an async diagnostics event bus. Per the
redesign note above they belong with the milestone-6 client layer that consumes them,
rebuilt Go-idiomatically (goroutines, `context.Context`, `sync`). The metadata
*decoders* — the near-mechanical core — are complete at milestone 4.

Milestone 3 is the fiddliest and carries the most wire knowledge per line.
Milestone 5 is the first point at which anything can be tested against a real
system, which is also the first point at which the port can be shown to be
wrong.

Milestone 5 is verified: on 2026-08-19 the client completed a live
`STFC_CONNECTION` against the A4H developer edition (kernel release 758) over
the gateway port (`sapgw00`, 3300 — *not* the dispatcher 3200), logging on with
password scramble and echoing `REQUTEXT` back through `ECHOTEXT`. Two wire facts
were confirmed only against the live system and are worth stating: the connect
port is the **gateway**, and the classic-RFC server returns **only the export
and table parameters the client names in "requested outputs"** — a call that
lists none gets back control fields and no parameter data. See
`internal/client/live_test.go` (guarded by `OPEN_RFC_LIVE=1`).

On the same day the runtime metadata path was exercised live end to end: the
client discovers a function's interface (`RFC_GET_FUNCTION_INTERFACE`) and a
structure's layout (`RFC_GET_STRUCTURE_DEFINITION`) at runtime, then encodes a
structure import and decodes both the echoed structure export and the returned
table rows — no interface hardcoded. Proven against `RFC_SYSTEM_INFO` (RFCSI
export) and `STFC_STRUCTURE` (RFCTEST import/export + RFCTABLE). See
`internal/client/live_explore_test.go`. Deep/nested xRFC (`STFC_DEEP_TABLE`,
`STFC_DEEP_STRUCTURE`) is the remaining M3b work.

## Definition of done for a milestone

- Every ported file carries its SPDX and provenance header, and
  `docs/provenance.md` lists it.
- Upstream's tests for the same layer are ported, including the malformed-input
  and boundary cases. A case with no Go analogue is recorded in
  `docs/provenance.md` with the reason.
- Go-specific hazards upstream cannot have are covered: slice aliasing of
  caller memory, and a fuzz target for every decoder that consumes network
  bytes.
- `go test -race -shuffle=on ./...`, `gofmt -l .`, and `go vet ./...` are clean.

## Conformance vectors

`conformance/testdata/vectors/` holds language-neutral vectors — hex in, hex or
error out — so that a wire fact can be stated once and checked by both
implementations. Right now they are authored here, because upstream keeps its
wire facts inside `.test.ts` files where a second implementation cannot reach
them.

The intended end state is the reverse: upstream owns the corpus, this
repository vendors it with a checksum lock, and a scheduled job opens a pull
request when the two diverge. That requires an upstream change, so it is
recorded here as intent rather than presented as fact. See
`conformance/README.md`.

## What is deliberately out of scope

The value files `classic-bcd.ts`, `classic-int8.ts`, and `rfc-value-snapshot.ts` are
also dropped: they are node-rfc representation-mode selectors and a defense
against malicious JavaScript caller objects, none of which has a Go analogue.
See docs/provenance.md.


`src/compat/` — the `node-rfc` and `@sap/cds-rfc` compatibility facades, about
7k lines. They exist to be bug-compatible with an npm package. Go has no such
consumer, so the budget goes to a Go-idiomatic API instead.

## Future direction (idea, not scope)

**A pure-Go SAProuter server (plaintext).** The client is already SAP-binary-free
and `internal/saprouter` handles routing *through* an existing SAProuter. A
natural separate track is to implement the SAProuter *server* in Go — an
auditable, containerizable proxy that replaces SAP's proprietary `saprouter`
binary. Scope: reuse NI framing and route admission, add an NI_ROUTE payload
decoder, NI_PONG/NI_RTERR encoders, a saprouttab-style ACL, and a bidirectional
NI-frame tunnel with router-to-router chaining. SNC (channel encryption) is the
hard, out-of-scope part; a plaintext router still serves the common unencrypted
internal/dev topologies. Not part of milestone 6.


Recorded as intent, not a committed milestone — this port must first reach a
live synchronous call (milestone 5) before any of this is actionable.

**Seamless hybrid ABAP execution.** Run ABAP locally with
[open-abap](https://github.com/open-abap) (an ABAP interpreter), and let a
`CALL FUNCTION` whose target is *not* in the local registry fall through to a
real SAP system over classic RFC — using this client as the forwarding
transport. Local-first for everything open-abap can evaluate; remote only for
the dependencies it cannot, transparently to the caller.

Why it fits here: once milestone 5 lands, this package can dial a real system
(for example the A4H developer edition) and invoke arbitrary function modules
with full metadata resolution (milestone 4). A thin resolver in front of
open-abap — "is this FM local? evaluate it; otherwise forward over RFC" — is
then a small integration layer on top, not new protocol work. The result is
one ABAP program that runs locally where it can and reaches the live system
only where it must.

Open questions to settle before this is more than an idea: how a forwarded call
maps local ABAP values to this client's typed parameters, how errors and
exceptions round-trip across the boundary, and how stateful sessions / commit
units behave when some calls are local and some remote.
