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
| 2 | APPC records, CPIC field chains, RFCPRO field headers, gateway handshake | `src/protocol/{appc,cpic,rfcpro,gateway}.ts` | next |
| 3 | Value codecs: BCD, packed decimal, decimal float, temporal, xRFC, recursive xRFC | `src/values/` | |
| 4 | Metadata: `RFC_METADATA_GET`, `DDIF_FIELDINFO`, repository | `src/metadata/` | |
| 5 | Transport over `net.Conn`; first live `STFC_CONNECTION` | `src/transport/`, `src/client/` | |
| 6 | Pool, session contexts, transactions, SAProuter, SOCKS5 | `src/pool/`, `src/lifecycle/`, `src/transport/` | |

Milestone 3 is the fiddliest and carries the most wire knowledge per line.
Milestone 5 is the first point at which anything can be tested against a real
system, which is also the first point at which the port can be shown to be
wrong.

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

`src/compat/` — the `node-rfc` and `@sap/cds-rfc` compatibility facades, about
7k lines. They exist to be bug-compatible with an npm package. Go has no such
consumer, so the budget goes to a Go-idiomatic API instead.
