# open-rfc-go

A Go port of [`open-rfc`](https://github.com/marianfoo/open-rfc) — an SDK-free
client, and now an experimental server, for SAP classic synchronous RFC.

> ## 🎉 Both directions are live against real SAP — with zero SAP libraries
>
> **2026-08-19 — open-rfc-go now speaks classic RFC _as the server_, not only the client.**
> A live SAP system (A4H, SAP_BASIS 793) ran a real ABAP program of six parametrized
> calls and **open-rfc-go answered every one, `rc=0`**, as the server:
>
> | call | result |
> |---|---|
> | `RFC_PING` | rc=0 |
> | `RFC_SYSTEM_INFO` | rc=0 · sysid=A4H |
> | `STFC_CONNECTION` | rc=0 · echo returned (with its server→client callback) |
> | `STFC_STRUCTURE` | rc=0 · structure + table |
> | `RFC_READ_TABLE` | rc=0 · 17 cols × 2 rows |
> | `STFC_STRING` | rc=0 |
>
> The **client** leg is live-proven too (STFC_CONNECTION / STFC_STRUCTURE / RFC_READ_TABLE
> against A4H). Pure Go — no NW RFC SDK, no native library.
>
> 📖 The full wire journey — gateway record, CPIC logon-accept, per-session token
> mirroring (conversation id + RFC GUID in both byte orders), the RSRFCPIN dance —
> is written up in [`docs/discoveries/0001`](docs/discoveries/0001-live-type3-server.md).

**Still pre-release.** There is no published release and no support boundary. The
server today answers by matching each request to a captured reply and patching this
session's tokens; a meaningful fast-ser codec that generates responses from values
is the next step. Do not depend on it yet.

📋 **[Cheat sheet](docs/cheatsheet.md)** — ports, auth, constants, layer map,
commands, and the known cross-language hazards, on one page.

---

## Table of contents

- [What this is](#what-this-is)
- [What "SDK-free" means](#what-sdk-free-means)
- [Status](#status)
- [Installation](#installation)
- [What works today](#what-works-today)
- [How the port works](#how-the-port-works)
- [Repository layout](#repository-layout)
- [Documentation](#documentation)
- [Development](#development)
- [Conformance vectors](#conformance-vectors)
- [Known cross-language hazards](#known-cross-language-hazards)
- [Scope](#scope)
- [Roadmap](#roadmap)
- [Licensing and attribution](#licensing-and-attribution)
- [Reporting problems](#reporting-problems)
- [FAQ](#faq)

---

## What this is

SAP systems expose function modules over **classic RFC**, a stateful binary
protocol that has no public specification. The conventional way to speak it is
SAP's proprietary NW RFC SDK — a native library you download from SAP under a
license, link against, and ship with your application.

`open-rfc` reimplements the protocol directly, in TypeScript, with no SDK and no
native code. This repository ports that work to Go.

The value being ported is not the code so much as the **wire knowledge**: which
bytes go where, which lengths are real protocol rules and which are one system's
habits, and which failures mean "the password is wrong" versus "your parser
pinned a hostname length". Upstream's `docs/recurring-bug-class.md` documents six
occasions where that distinction was gotten wrong — one of them cost three
weeks. A port is an unusually easy way to reintroduce all six, which is why this
repository is organised the way it is.

## What "SDK-free" means

| | SDK-based clients | open-rfc / open-rfc-go |
|---|---|---|
| Native dependency | SAP NW RFC SDK (`libsapnwrfc`) | none |
| Install step | download SDK under SAP license, set library path | `go get` |
| Cross-compilation | constrained by the native build | ordinary Go cross-compilation |
| Wire behaviour | decided by SAP's library | decided by this code, and testable |
| Protocol coverage | complete | narrow and explicitly bounded |

The trade is real and it goes both ways: you gain a dependency-free, inspectable,
testable client, and you give up everything SAP's library does that this project
has not yet proven it can do. See [Scope](#scope).

## Status

| Layer | Upstream source | State |
|---|---|---|
| NI record framing | `src/protocol/ni.ts` | **ported, tested** |
| Bounded byte reader/writer | `src/protocol/bytes.ts` | next |
| APPC conversation records | `src/protocol/appc.ts` | next |
| CPIC field chains | `src/protocol/cpic.ts` | next |
| RFCPRO field headers | `src/protocol/rfcpro.ts` | next |
| Gateway handshake | `src/protocol/gateway.ts` | next |
| Message-server resolution | `src/protocol/message-server.ts` | planned |
| Password scrambling | `src/protocol/password-scramble.ts` | planned |
| Value codecs (BCD, decimal float, temporal, xRFC) | `src/values/` | planned |
| Metadata repository | `src/metadata/` | planned |
| Transport, client, pool, transactions | `src/transport/`, `src/client/`, `src/pool/`, `src/lifecycle/` | planned |
| `node-rfc` compatibility facades | `src/compat/` | **out of scope** |

Progress is tracked in [`docs/porting-plan.md`](docs/porting-plan.md).

## Installation

```sh
go get github.com/oisee/open-rfc-go
```

There is nothing importable yet — `rfc/` is an empty package documenting that
fact. Requires Go 1.26 or later. No third-party dependencies, and none are
planned; the standard library is a design constraint, not a coincidence.

## What works today

One package, `internal/ni`, implementing SAP's Network Interface record framing:
a four-byte big-endian length followed by that many payload bytes.

```go
frame, err := ni.EncodeFrame([]byte("RFC"))
// frame == 00 00 00 03 52 46 43

decoder, err := ni.NewFrameDecoder(ni.DefaultMaxPayloadLength)
payloads, err := decoder.Push(bytesFromSocket)   // zero or more complete records
err = decoder.Finish()                           // errors if the stream ended mid-record
decoder.Reset()                                  // zeroes retained bytes
```

It is `internal/`, so you cannot import it. That is deliberate: a public API gets
designed once, on purpose, when there is something worth committing to.

Properties worth knowing, because they are the ones that were easy to get wrong:

- **Reads have no relationship to record boundaries.** A record may arrive one
  byte at a time, four records may arrive in one read, and the length prefix
  itself may straddle two reads. All three are tested.
- **Nothing aliases caller memory.** `EncodeFrame` copies the payload and `Push`
  copies the chunk, so the caller may reuse its read buffer immediately.
  Decoded payloads never alias the retained queue either.
- **The payload bound is enforced before allocation**, because the length prefix
  is attacker-controlled.
- **Retained bytes are zeroed on `Reset`**, since a partial record can contain
  credential material — with the caveat in [`SECURITY.md`](SECURITY.md) that
  Go's garbage collector may have already copied them.

## How the port works

This repository does not re-derive the protocol. It follows upstream, and it
keeps the relationship machine-checkable.

**Every ported file records where it came from.** An SPDX header names the
upstream file and the exact commit:

```go
// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/protocol/ni.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. ...
```

[`docs/provenance.md`](docs/provenance.md) aggregates those headers into one
table. It serves two purposes: it satisfies Apache-2.0 §4(b), which requires
prominent notices of modification, and it is the index used to propagate an
upstream fix. When open-rfc changes a file, the table says which Go file to look
at, and the recorded commit says how far behind you are:

```sh
git -C ../open-rfc log --oneline 847036d..origin/main -- src/protocol/ni.ts
```

**Wire bugs belong upstream.** If this port and open-rfc are both wrong about the
protocol, the fix goes to open-rfc, where the knowledge lives, and comes back
here as a port. Only translation mistakes are fixed here. A divergence nobody
recorded is worse than a shared bug.

**Tests come across too**, including the malformed-input and boundary cases. A
case with no Go analogue — upstream has several that defend against JavaScript
callers overriding typed-array accessors — is recorded in `docs/provenance.md`
with the reason rather than dropped silently. Go-specific hazards that upstream
cannot have get new tests: slice aliasing, and a fuzz target for every decoder
that consumes network bytes.

## Repository layout

```
rfc/                      public API (empty; see doc.go)
internal/
  ni/                     NI record framing                    ← ported
  bytes/ appc/ cpic/      framing, conversation records        ← next
  rfcpro/ gateway/
  msgserver/ value/       resolution, value codecs
  metadata/ transport/
conformance/
  testdata/vectors/       language-neutral hex vectors
  ni_vectors_test.go      the runner
docs/
  cheatsheet.md           one-page reference
  protocol-primer.md      how classic RFC works on the wire
  glossary.md             SAP and protocol vocabulary
  architecture.md         layers, invariants, evidence hierarchy
  development.md          workflow, testing, fuzzing
  porting-plan.md         milestones and definition of done
  provenance.md           upstream mapping and modifications
  recurring-bug-class.md  the mistake to avoid (verbatim from upstream)
  surface/                cited inventories of the unported layers
```

## Documentation

Start at [`docs/README.md`](docs/README.md), or go straight to what you need:

| Document | Read it when |
|---|---|
| [Cheat sheet](docs/cheatsheet.md) | You want ports, constants, or commands right now |
| [Protocol primer](docs/protocol-primer.md) | You want to know what classic RFC actually does on the wire |
| [Glossary](docs/glossary.md) | A three-letter acronym is in your way |
| [Architecture](docs/architecture.md) | You are about to change the design |
| [Recurring bug class](docs/recurring-bug-class.md) | You are about to write or change a decoder — **required** |
| [Development](docs/development.md) | You are about to write code |
| [Live test plan](docs/live-test-plan.md) | You have a real SAP system and want to verify against it |
| [Porting plan](docs/porting-plan.md) | You want to know what is next and why in that order |
| [Provenance](docs/provenance.md) | You want to know where a file came from |
| [Surface inventories](docs/surface/) | You are porting a layer and want its signatures and constants |

The [surface inventories](docs/surface/) deserve a note. They are ~7,800 lines of
mechanical inventory of the unported upstream layers — exported signatures,
constant values, error messages, comment-stated invariants, and the wire facts
the tests assert, each with a `path:line` citation. They are **porting input, not
truth.** One inventory's own verification pass corrected four counts it had
asserted from `grep` output rather than from reading. Nothing gets ported without
reading the upstream file.

## Development

```sh
go build ./...
go test -race -shuffle=on ./...    # -race is not optional; see SECURITY.md
gofmt -l .                          # must print nothing
go vet ./...
```

Fuzz a decoder you have touched:

```sh
go test ./internal/ni -run TestX -fuzz FuzzFrameDecoderBounds -fuzztime 60s
```

Full workflow, including how to run one test and what CI checks, is in
[`docs/development.md`](docs/development.md). Rules for contributors and coding
agents are in [`AGENTS.md`](AGENTS.md) and [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Conformance vectors

Wire facts live in [`conformance/testdata/vectors/`](conformance/testdata/vectors/)
as language-neutral JSON — hex in, hex or a named error out — rather than inside
test code:

```json
{
  "name": "one record split across every byte boundary",
  "maxPayloadLength": 4096,
  "chunksHex": ["00", "00", "00", "03", "52", "46", "43"],
  "payloadsHex": ["524643"],
  "residualBytes": 0,
  "why": "The length prefix itself may straddle reads; a decoder that assumes four contiguous bytes fails on a slow link."
}
```

The reason is drift. Upstream's wire facts live inside `.test.ts` files, where a
second implementation cannot reach them — so both suites pass while the two
decoders quietly disagree about bytes no test compares. Vectors are the fix, and
they earn their keep immediately: writing the first file surfaced an error in the
vectors themselves within minutes.

Every vector states **why** it matters. A case that cannot explain itself gets
deleted rather than kept for coverage. See
[`conformance/README.md`](conformance/README.md), including the intended end
state where upstream owns the corpus and this repository vendors it under a
checksum lock.

## Known cross-language hazards

Places where a faithful-looking translation is wrong. Each is recorded with
citations in [`docs/surface/cross-layer-answers.md`](docs/surface/cross-layer-answers.md).

**`JSON.stringify` is not `encoding/json`.** Upstream builds identity strings and
hash inputs with `JSON.stringify`. Go escapes `<`, `>`, `&` and U+2028/U+2029
differently, and cannot represent an unpaired surrogate at all. This is
reachable: input validation rejects only control characters, so a backend key
containing `&` validates fine and hashes differently in Go. `SetEscapeHTML(false)`
is not sufficient. The port needs an explicit canonical encoder.

**`localeCompare` is not byte order.** Graph nodes are sorted with
`localeCompare` before hashing. ICU collation puts `"a"` before `"B"`; Go's byte
order does not. Any graph with mixed-case node names digests differently.

**Not every reported divergence is one.** A signed-versus-unsigned read of
compact temporal values looked like an `int64`/`uint64` decision. It is not:
every `maximumRaw` sits below its width's signed ceiling, so both paths reject
identical byte patterns and no input can distinguish them. Verified against the
spec table — which is the standard this project holds itself to before calling
something a bug.

## Scope

Inherited from upstream, and narrower.

**In scope:** direct application-server classic RFC, classic serialization,
password authentication.

**Out of scope, deliberately:** the `node-rfc` and `@sap/cds-rfc` compatibility
facades (~7k lines upstream). They exist to be bug-compatible with an npm
package. Go has no such consumer, so that budget goes to a Go-idiomatic API.

**Not implemented anywhere, upstream or here:** SNC, WebSocket RFC, x509, SSO
tickets, SAML/JWT, and trusted-system RFC. `snc_mode` is rejected outright rather
than ignored. Note what that means: **classic RFC over direct, SAProuter-routed,
or message-server-selected transport has no transport encryption and no peer
authentication.** Do not send credentials or RFC traffic across an untrusted
network. See [`SECURITY.md`](SECURITY.md).

## Roadmap

| Milestone | Scope | State |
|---|---|---|
| 1 | NI framing | **done** |
| 2 | Bounded byte reader, APPC, CPIC, RFCPRO, gateway | **done** |
| 3 | Value codecs | next |
| 4 | Metadata repository, `classic-rfc.ts` | |
| 5 | Transport over `net.Conn`; first live `STFC_CONNECTION` | |
| 6 | Pool, session contexts, transactions, SAProuter, SOCKS5 | |

Milestone 5 is the first point at which anything can run against a real system —
and therefore the first point at which this port can be shown to be wrong.
`classic-rfc.ts` is deferred to milestone 4 on purpose: it has no direct upstream
tests, and porting a decoder with no behavioural oracle is how you get a decoder
that memorises one system's habits.

## Licensing and attribution

Apache License 2.0 — see [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).

This is a **derivative work** of open-rfc, Copyright 2026 Marian Zeis, licensed
under Apache-2.0. That license is permissive but not obligation-free: §4 requires
shipping the license, retaining upstream's `NOTICE` attributions, and stating
prominently which files were changed. This repository does all three, per file.

`NOTICE` names both copyright holders. `docs/provenance.md` records every ported
file with its upstream source and modifications, and a "not ported, deliberately"
section explaining why no `THIRD_PARTY_NOTICES.md` exists — the only upstream
files containing third-party code (SAP SE's `node-rfc`) are not ported, and if
they ever are, that file and its attribution come with them.

Contributions are accepted under Apache-2.0 with a
[DCO](DCO.md) sign-off (`git commit -s`).

SAP, ABAP, SAP S/4HANA, and SAP NetWeaver are trademarks or registered trademarks
of SAP SE or its affiliates. This project is independent of, and not endorsed by,
SAP SE.

## Reporting problems

**Wire behaviour → [open-rfc](https://github.com/marianfoo/open-rfc/issues).**
Does the TypeScript do the same thing? Then it is upstream's, and fixing it there
reaches both implementations.

**Translation mistakes → here.** Aliasing, integer width, encoding, concurrency,
anything where Go and TypeScript disagree.

**Security issues →** privately, per [`SECURITY.md`](SECURITY.md). If it affects
the wire behaviour rather than the translation, it affects upstream too.

## FAQ

**Can I use this in production?** No. It cannot connect to an SAP system.

**When will it be usable?** Milestone 5 is the earliest a call can succeed. No
date; the ordering is chosen so each milestone is verifiable against bytes.

**Why not wrap the SAP NW RFC SDK with cgo?** That is a different project with
different trade-offs — you would get complete protocol coverage, and a native
dependency, an SAP license obligation, and constrained cross-compilation. This
project exists to find out how far a pure implementation goes.

**Why not generate the Go code from the TypeScript?** Because the interesting
parts do not survive translation. `AbortSignal` becomes `context.Context`,
Promise plumbing becomes goroutines, and roughly 800 lines of upstream's
lifecycle layer disappear into Go primitives. A generated port would carry that
scaffolding across and still get the encoding hazards above wrong.

**Is this affiliated with open-rfc?** No. It is an independent derivative work.
Upstream's maintainers do not support it, and behaviour is not guaranteed to
match.

**Why is everything in `internal/`?** So that nothing becomes a compatibility
promise by accident. Go turns that into a compile error rather than a review
finding.
