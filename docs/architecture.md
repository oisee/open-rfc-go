<!--
Adapted from docs/architecture.md in open-rfc
(https://github.com/marianfoo/open-rfc) at commit 847036d, Copyright 2026
Marian Zeis, licensed under the Apache License, Version 2.0. The layer table
and implementation ladder were rewritten for Go; the ownership invariants and
evidence hierarchy are upstream's, kept because they are the reason the
codebase is shaped this way.
-->

# Architecture

Classic RFC is a stateful binary protocol with negotiated behaviour and no
public specification. That single fact drives almost every decision here, in
Go exactly as in the TypeScript original.

## Layers

```
rfc (public API)
  └── internal/metadata ── internal/value
        └── internal/{gateway,msgserver}
              └── internal/{appc,cpic,rfcpro}
                    └── internal/ni
                          └── internal/transport (net.Conn)
```

| Layer | Package | Upstream |
|---|---|---|
| Consumer facade | `rfc/` | `src/client/` (not `src/compat/`) |
| Destination, pool, lifecycle | `internal/destination`, `internal/pool` | `src/destination/`, `src/pool/`, `src/lifecycle/` |
| Metadata repository | `internal/metadata` | `src/metadata/` |
| Value serialization | `internal/value` | `src/values/` |
| APPC/CPIC conversation, NI framing | `internal/appc`, `internal/cpic`, `internal/rfcpro`, `internal/gateway`, `internal/msgserver`, `internal/ni` | `src/protocol/` |
| Transport | `internal/transport` | `src/transport/` |

**Upward layers consume semantic results. Downward layers never know about
`node-rfc`, CAP, BAPI, or any application-specific name.** Go enforces the
outer half of this for free: everything under `internal/` is invisible to
consumers, so an accidental export is a compile error rather than a review
finding.

## Ownership invariants

These are upstream's, and they are the rules that concurrency bugs violate.
Go changes how they are enforced, not what they say.

- A leased connection **exclusively owns its `net.Conn` and at most one
  in-flight call.** In Go this is one goroutine owning the conn, and
  `go test -race` proves it — the TypeScript original can only assert it.
- Repository calls use their **own connection** and never borrow a
  context-pinned application connection, even when credentials are shared.
- Normalized metadata descriptors are **immutable**: returned by value, never
  as a shared pointer a caller can mutate.
- A session context is destination-and-generation scoped, explicitly
  identified, nestable, reference-counted, and pins one connection until reset
  or close. Upstream carries it in `AsyncLocalStorage`; here it is an explicit
  handle or a `context.Context` value, never implicit.
- **Logging, metrics, observer callbacks, and file I/O never run while a pool,
  repository, lifecycle, or context lock is held.**

## The implementation ladder

For every change, stop at the first step that holds:

1. Remove behaviour the current milestone does not require.
2. Prefer the Go standard library — `net`, `context`, `errors`, `encoding/binary`,
   `log/slog`, `testing`.
3. Reuse an existing checked primitive that already expresses the wire rule.
4. Add the narrowest implementation that passes the current evidence and tests.
5. Keep validation, bounds, useful errors, redaction, cleanup, and a smoke test.
6. Stop when the diff is locally understandable.

An abstraction is justified by a **second proven consumer or negotiated wire
variant**, not by anticipation. Assembly or cgo is justified by a **measured
bottleneck**, not by expectation.

## How the code is written

- Packages and identifiers use protocol vocabulary. Not `manager`, `helper`,
  or `util`.
- Parsed records are immutable data; state changes live in small methods.
- Illegal states and unsupported capabilities **fail explicitly** rather than
  falling through.
- Numeric fields have named constants, a stated byte order, bounds, and
  field-path errors.
- Exported errors are wrapped sentinels, so callers use `errors.Is` rather
  than matching on message text.
- Tests state the wire invariant in their name and fail if that invariant
  breaks.
- Comments explain **why an invariant exists**, never what the code does.

## Evidence hierarchy

There is no public specification for this protocol, so "how do we know?" is a
real question with a ranked answer. Strongest first:

1. A passing differential or live experiment against a real SAP system.
2. SAP documentation, or a relevant SAP Note, converted into a testable
   requirement. Cite the Note number.
3. A repeated, structurally decoded capture observation.
4. An independent open-source implementation, used as corroboration.
5. A hypothesis — clearly marked, and never emitted on a live connection until
   verified.

**For this port there is a sixth consideration that outranks nothing but must
be stated: open-rfc itself is a behavioural corpus, not a wire oracle.** It is
this project's primary source, and it is very good, but a Go file that matches
the TypeScript is only evidence that the port is faithful — not that the wire
rule is right. Where upstream marks something as a hypothesis, it stays a
hypothesis here.

## Before you propose a design change

Read [`recurring-bug-class.md`](recurring-bug-class.md) first. The mistake it
describes has been made six times in the upstream codebase, and a translation
is an unusually easy way to make it a seventh — a decoder that pins a length
because the TypeScript pinned a length is the same bug with a new syntax.
