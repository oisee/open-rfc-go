# open-rfc-go

A pure-Go, **SDK-free** implementation of SAP classic synchronous RFC — client
**and** server. No NW RFC SDK, no native library, no cgo. A Go port of
[`open-rfc`](https://github.com/marianfoo/open-rfc).

> ## 🎉 Both directions are live against real SAP — with zero SAP libraries
>
> **2026-08-19 — open-rfc-go now speaks classic RFC _as the server_, not only the client.**
> A live SAP system (A4H) ran a real ABAP program of six parametrized calls and
> **open-rfc-go answered every one, `rc=0`**, as the server:
>
> | `RFC_PING` | `RFC_SYSTEM_INFO` | `STFC_CONNECTION` | `STFC_STRUCTURE` | `RFC_READ_TABLE` | `STFC_STRING` |
> |---|---|---|---|---|---|
> | rc=0 | rc=0 · A4H | rc=0 · echo + callback | rc=0 · struct+table | rc=0 · 17×2 | rc=0 |
>
> A single Go endpoint answers **every SM59 test button green** — Connection Test,
> Unicode Test, Fast Serialization Test — across three serialization modes. The
> **client** leg is live-proven too. → the wire story: [`docs/discoveries/0001`](docs/discoveries/0001-live-type3-server.md)

> ⚠️ **Research preview.** No release, no stable API, no support boundary. Classic
> RFC has no transport encryption — don't send credentials across an untrusted
> network ([`SECURITY.md`](SECURITY.md)). Don't depend on it yet.

## What you can do now

**Call an ABAP function module** (pure Go, native values in and out):

```go
ctx := context.Background()
c, _ := rfc.Open(ctx, rfc.Destination{
    Host: "sap.example", Port: 3300, // gateway of instance 00
    Client: "001", User: "DEVELOPER", Password: "…", Language: "E",
})
defer c.Close(ctx)

res, err := c.Call(ctx, "STFC_CONNECTION", rfc.Params{"REQUTEXT": "hello from Go"})
// res.Get("ECHOTEXT"), res.Table("RFCTABLE"), …
var ex *rfc.ABAPException
if errors.As(err, &ex) { /* typed ABAP-side failure */ }
```

**Be an RFC server / lab** — make this host answer real SM59 destinations. Point
a destination's target host at this box; the connection type and system number
pick the mode:

```sh
go run ./cmd/rfc-lab -target-host <your-real-sap-host>
#   sys 00  ports 3200/3300  transparent sniffer → the real system (captures the wire)
#   sys 11  port  3311       our state-machine server: Connection/Unicode tests go green
#   sys 12  port  3312       content-addressed server: a whole ABAP program runs green
#   type H  port  8000       our HTTP responder
#   type W  port  44300      our WebSocket upgrade
```

**Inspect the wire** — a framing-aware proxy and a decoder that speaks our own
protocol stack:

```sh
go run ./cmd/rfc-sniffer -listen :3300 -target <sap-host>:3300 -dump cap.jsonl
go run ./cmd/rfc-viewer cap.jsonl          # decoded transcript (values redacted)
```

## Status

| | state |
|---|---|
| **Client** | live-proven against A4H — `rfc.Open` / `Client.Call`, metadata cache, typed ABAP errors |
| **Server** | answers all SM59 test buttons + a real program (via captured, token-patched replies) |
| **Next** | a fast-ser codec that generates responses from values, then Go/JS functions behind a bridge |

Full history: [`CHANGELOG.md`](CHANGELOG.md).

## Learn more

| Document | For |
|---|---|
| [**About** — what it is & how it's built](docs/about.md) | rationale, porting discipline, scope, cross-language hazards |
| [Discoveries 0001 — live type-3 server](docs/discoveries/0001-live-type3-server.md) | the whole server wire journey |
| [Cheat sheet](docs/cheatsheet.md) | ports, auth, constants, commands — one page |
| [Docs index](docs/README.md) | everything else (primer, glossary, architecture, dev) |
| [Porting plan](docs/porting-plan.md) | what's next and why in that order |

## Licensing

Apache-2.0, a derivative work of [`open-rfc`](https://github.com/marianfoo/open-rfc)
(© 2026 Marian Zeis). See [`LICENSE`](LICENSE), [`NOTICE`](NOTICE), and
[`docs/provenance.md`](docs/provenance.md); contributions take a [DCO](DCO.md)
sign-off. SAP, ABAP, and SAP S/4HANA are trademarks of SAP SE; this project is
independent of and not endorsed by SAP SE.
