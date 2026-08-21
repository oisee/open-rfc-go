# open-rfc-go

A pure-Go, **SDK-free** implementation of SAP classic synchronous RFC — client
**and** server. No NW RFC SDK, no native library, no cgo. A Go port of
[`open-rfc`](https://github.com/marianfoo/open-rfc).

> ## 🎉 Call any SAP function module — from Go, the shell, or as MCP tools
>
> **2026-08-20 — the client now calls essentially any FM, and an MCP server turns
> a live SAP system into tools an AI can use.** Every scalar type (incl.
> STRING/XSTRING, DATE/TIME, packed DEC/TIMESTAMP, FLOAT, UTCLONG), flat **and
> deep** structures & tables (STRING/XSTRING via xRFC), and both **classic and
> fast** serialization on decode — all round-tripped live against A4H, pure Go.
> On top of that: `rfc describe <FM>` renders any function module as an **MCP-tool
> JSON Schema**, `rfc call` runs any FM from plain JSON, and **`rfc-mcp`
> auto-exposes a curated set of FMs as real MCP tools** — point it at your SAP and
> an assistant can call RFC directly. Zero SAP libraries.

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
> Unicode Test, Fast Serialization Test — across three serialization modes. And
> open-rfc-go now **decodes S/4HANA classic responses end to end** — scalars and
> tables alike (real T000 rows, field lists, structures) — pure Go. The **client**
> leg is live-proven too. → the wire story: [`docs/discoveries/0001`](docs/discoveries/0001-live-type3-server.md)

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

**Drive RFC from the shell or an AI** — `rfc` is the client: a CLI, and (as
`rfc mcp`) an MCP server over the same library. Connection from `.rfc.json`, env,
or flags:

```sh
export SAP_ASHOST=sap.example SAP_USER=DEVELOPER SAP_PASSWORD=…   # or a .rfc.json system

go run ./cmd/rfc info                                  # system info
go run ./cmd/rfc describe STFC_STRUCTURE               # FM interface as an MCP-tool JSON Schema
go run ./cmd/rfc call STFC_CONNECTION '{"REQUTEXT":"hi"}'   # call any FM with JSON
go run ./cmd/rfc search 'BAPI_USER_*'                  # find RFC-enabled FMs

# MCP server (stdio, like `vsp mcp`): every matching FM becomes a tool an assistant can call
go run ./cmd/rfc mcp --expose 'BAPI_*,Z_*' --hide '*_DELETE,*_CREATE'
```

**Debug ABAP over RFC** — a pinned conversation is a stable ABAP session, which
is exactly what the debugger needs: `attach_debuggee( )` hands back an object
reference and every later operation hangs off it. So `Client.Pin` plus a thin
ABAP facade gives real breakpoints, attach, and stepping — no SAP GUI, no
Eclipse, no WebSocket:

```go
session, _ := c.Pin(ctx)      // one connection, one roll area, held open
defer session.Close()
session.Call(ctx, "ZADT_DEBUG_RFC", rfc.Params{"I_OP": "listen", "I_TIMEOUT": 120})
session.Call(ctx, "ZADT_DEBUG_RFC", rfc.Params{"I_OP": "attach", "I_DEBUGGEE_ID": id})
session.Call(ctx, "ZADT_DEBUG_RFC", rfc.Params{"I_OP": "step", "I_KIND": "over"})
```

Live on A4H: a breakpoint set from one connection, hit by a function module
called over a *second* connection, attached to, and stepped through — with the
stack showing the real RFC entry chain `%_RFC_START` → `REMOTE_FUNCTION_CALL` →
the module.

**And it needs nothing installed on the server.** SAP's own ADT debugger
resources — the ones Eclipse drives — reach the same place through
`SADT_REST_RFC_ENDPOINT` on a pinned conversation:

```
POST /sap/bc/adt/debugger/listeners?debuggingMode=user&requestUser=…   → the debuggee that stopped
POST /sap/bc/adt/debugger?method=attach&debuggeeId=…                   → attached
GET  /sap/bc/adt/debugger/stack?method=getStack                        → dbg:stack, source URIs and all
POST /sap/bc/adt/debugger?method=stepOver                              → dbg:step
```

A stateless HTTP client cannot use these, because ADT keeps the debug session in
an ABAP roll area and can only find it again through a `sap-contextid` cookie.
Over RFC the roll area *is* the conversation, so there is nothing to correlate —
and the answers come back with everything ADT knows: per-frame source URIs, DYNP
screen frames, authorization flags, the action catalogue. SAP itself labels the
session `RFC session: <instance>`.

Driver and the typed ABAP facade:
[vibing-steampunk](https://github.com/oisee/vibing-steampunk) (`vsp rfc debug`).

*Which path to pick is not only a technical question.* "Install nothing" is a
usability argument; on a customer system the rules may point the other way,
because SAP's API Policy expressly carves out custom-developed ABAP interfaces
on-premise while saying nothing so clear about SAP-delivered but unpublished
function modules. The terrain, with sources, is mapped in
[`reports/eclipse-mcp-integration.md`](reports/eclipse-mcp-integration.md) —
which also covers SAP's own MCP server, shipped inside Eclipse ADT since 3.60.
Full write-up: [`reports/debugger-over-rfc.md`](reports/debugger-over-rfc.md).

**The debugger is only the hard case.** `SADT_REST_RFC_ENDPOINT` carries an
arbitrary HTTP request, so *any* ADT resource is reachable the same way — source
read and write, activation, ATC, unit tests, transports, search, refactoring:
the whole surface, over the gateway port, where ICF is closed, HTTPS terminates
somewhere inconvenient, or CSRF and cookies are a fight. Proven so far:
`discovery` (200, 299 KB), program source (200 with `ETag`/`Last-Modified`), a
missing object (404 with ADT's own exception document), and the debugger's
listen/attach/stack/step.

The stateful half is proven too: `LOCK` → `PUT` → `UNLOCK` → `ACTIVATE`, four
separate HTTP requests on one pinned conversation, and the lock handle from the
first still valid in the second. Over HTTP that fails — a short-lived client
cannot hold the ABAP session a lock is bound to. The object used for the test
was one an HTTP client could not even lock (`MODIFICATION_SUPPORT=NoModification`,
which is SAP saying "no modification assistant needed", not "read-only"). So
writing ABAP over RFC is not merely equivalent to writing over HTTP — on that
system it is strictly more capable.

**Sniff & emulate** — `rfc-lab` runs a transparent sniffer and a generating
("conscious") server together; point SM59 type-3 destinations at this box, then
decode captures offline with `rfc-viewer`:

```sh
go run ./cmd/rfc-lab -target-host <your-real-sap-host>
#   ports 3200/3300   sniffer → the real system, captures the wire (-dump cap-lab.jsonl)
#   port  3313        conscious server (sys 13) — generates classic responses (Serializer=Classic)
go run ./cmd/rfc-viewer cap-lab.jsonl                # decoded text transcript (values redacted)
go run ./cmd/rfc-viewer -html cap-lab.jsonl         # writes cap-lab.html — a self-contained visual inspector
go run ./cmd/rfc-viewer -serve :8080 cap-lab.jsonl  # HTTP inspector at localhost:8080 (refresh reloads a growing dump)
```

`rfc-viewer` is offline — it reads a capture file, never a live SAP system;
`-values` includes decoded scalar/table values (may reveal credentials/data).

## Status

| | state |
|---|---|
| **Client** | live-proven against A4H — `rfc.Open` / `Client.Call`, metadata cache, typed ABAP errors. Scalars (incl. STRING/XSTRING, DATE/TIME, packed DEC, FLOAT), flat **and deep** structures & tables (STRING/XSTRING via xRFC), classic **and** fast-serialization responses |
| **Server** | answers all SM59 test buttons + a real program (via captured, token-patched replies); a generating "conscious" server is WIP |
| **Decode** | S/4HANA classic responses decode fully — scalars + tables (native & mixed) |
| **Tooling** | live: an `rfc` CLI (`info`/`describe`/`search`/`call`/`read-table`/`ping`) and an MCP server (`rfc mcp`, stdio) — `describe <FM>` emits an MCP-tool JSON Schema, generic `call` runs any FM (JSON args, coerced per interface), and **`--expose`/`--hide` masks auto-generate real per-FM MCP tools** (with `outputSchema` + read-only/destructive hints; `--safe` blocks write FMs). Config via `.rfc.json` + env + flags. Dependency-free, extractable subproject |
| **Debugger** | ✅ the ABAP debugger driven end to end over classic RFC on a pinned session (`Client.Pin`) — **including SAP's own ADT debugger resources, with nothing installed on the server**: listen, attach, step, stack, live-verified against A4H |
| **Callback** | ✅ server→client RFC callbacks (DESTINATION 'BACK') — register `Destination.Callbacks`; live-verified |
| **Next** | per-FM `outputSchema` + HTTP transport for `rfc mcp`; write-FM safety gate ([design](docs/design/write-fm-safety.md)); finish the generating server |

Full history: [`CHANGELOG.md`](CHANGELOG.md); the ranked plan: [`docs/roadmap.md`](docs/roadmap.md).

## Learn more

| Document | For |
|---|---|
| [**About** — what it is & how it's built](docs/about.md) | rationale, porting discipline, scope, cross-language hazards |
| [Discoveries 0001 — live type-3 server](docs/discoveries/0001-live-type3-server.md) | the whole server wire journey |
| [Cheat sheet](docs/cheatsheet.md) | ports, auth, constants, commands — one page |
| [Docs index](docs/README.md) | everything else (primer, glossary, architecture, dev) |
| [Porting plan](docs/porting-plan.md) | what's next and why in that order |
| [Roadmap](docs/roadmap.md) | ranked plan — what's done, the tool surface, callbacks, and later bets |

## Licensing

Apache-2.0, a derivative work of [`open-rfc`](https://github.com/marianfoo/open-rfc)
(© 2026 Marian Zeis). See [`LICENSE`](LICENSE), [`NOTICE`](NOTICE), and
[`docs/provenance.md`](docs/provenance.md); contributions take a [DCO](DCO.md)
sign-off. SAP, ABAP, and SAP S/4HANA are trademarks of SAP SE; this project is
independent of and not endorsed by SAP SE.
