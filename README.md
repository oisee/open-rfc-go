# open-rfc-go

A Go port of [`open-rfc`](https://github.com/marianfoo/open-rfc) — an SDK-free
client for SAP classic synchronous RFC.

**Nothing here connects to an SAP system yet.** This repository is at the first
milestone of a staged port: the wire-framing layer. There is no client, no
connection pool, no support boundary, and no release. Do not depend on it.

## Status

| Layer | Upstream | State |
|---|---|---|
| NI framing | `src/protocol/ni.ts` | ported, tested |
| APPC records | `src/protocol/appc.ts` | not started |
| CPIC field chains | `src/protocol/cpic.ts` | not started |
| RFCPRO field headers | `src/protocol/rfcpro.ts` | not started |
| Gateway handshake | `src/protocol/gateway.ts` | not started |
| Value codecs | `src/values/` | not started |
| Metadata repository | `src/metadata/` | not started |
| Transport, client, pool | `src/transport/`, `src/client/`, `src/pool/` | not started |
| node-rfc compatibility facades | `src/compat/` | **out of scope** — no Go equivalent exists |

`docs/porting-plan.md` has the staged plan and the reasoning behind the order.

## Relationship to open-rfc

open-rfc is the authority on wire behaviour. This port follows it rather than
re-deriving protocol facts: every ported file records its upstream source file
and commit, so an upstream fix can be located here and vice versa. See
`docs/provenance.md`.

Two upstream documents are required reading before changing anything:

- `docs/architecture.md` — the layer boundaries and ownership invariants,
  adapted to Go.
- `docs/recurring-bug-class.md` — a verbatim copy of the upstream document
  describing the mistake that codebase made six times. A port is an unusually
  good opportunity to make it a seventh time.

## Build and test

```sh
go build ./...
go test -race -shuffle=on ./...
gofmt -l .
go vet ./...
```

## License

Apache License 2.0. See `LICENSE` and `NOTICE`. open-rfc-go is a derivative
work of open-rfc, which is Copyright 2026 Marian Zeis and licensed under the
same terms.

SAP, ABAP, SAP S/4HANA, and SAP NetWeaver are trademarks or registered
trademarks of SAP SE or its affiliates. This project is independent of and not
endorsed by SAP SE.
