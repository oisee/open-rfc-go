# MCP → RFC bridge design

Design report (2026-08-19). An MCP (Model Context Protocol) server that bridges
an AI assistant to SAP over **classic RFC**, built on this library. It joins two
sibling SDK-free Go bridges by the same author: **[vibing-steampunk
(vsp)](https://github.com/oisee/vibing-steampunk)** (ADT ↔ MCP — dev objects,
code, debug) and **[odata_mcp_go](https://github.com/oisee/odata_mcp_go)** (OData
↔ MCP — business data). This bridge is the **RFC** leg: runtime function-module
calls, table reads, BAPIs.

## What the library already gives us

Every tool below maps onto a call pattern already proven live in
`internal/client/live_explore_test.go`:

| Capability | Entry point |
|---|---|
| Open + logon | `client.Open` + `sess.LogonAndPing` |
| Typed call | `sess.Call(ctx, fn, imports, requestedOutputs)` |
| Pre-built (metadata) call | `sess.CallRaw(ctx, request)` |
| Serialize + pool | `lifecycle.Managed` (one call at a time), `lifecycle.NewPool` (health-checked, recyclable) |
| Discover FM interface at runtime | `metadata.BuildRfcGetFunctionInterfaceRequest` + `DecodeRfcFunctionInterfaceResult` |
| Discover structure layout | `metadata.BuildRfcGetStructureDefinitionRequest` + `DecodeRfcStructureDefinitionResult` |
| Encode/decode values | `classicrfc`, `structure`, `xrfc` |

Two live wire facts shape the bridge: connect to the **gateway** port; the server
returns **only the outputs you name** (so the bridge computes requested outputs
from metadata). Two hard boundaries: **auth is user+password only, no SNC**
(safe only on a trusted segment / non-production, or tunnelled via the shipped
SOCKS5/SAProuter dialers); **metadata calls must not borrow a context-pinned
application connection**.

> **Prerequisite:** a separate MCP-server module cannot import another module's
> `internal/`. The bridge depends on the P0 public **`rfc/` facade**
> (`docs/roadmap.md`). Build that first.

## v1 — a small, safe, fixed tool set

Read-first; credentials live only in the server's destination registry (tools
reference a destination *by name*, never carry secrets); metadata discovery lets
the bridge describe any FM to the model on demand.

- **`rfc_system_info`** — backend identity (RFC_SYSTEM_INFO). Read-only.
- **`rfc_function_search`** — FMs matching an ABAP pattern (RFC_FUNCTION_SEARCH). Read-only.
- **`rfc_get_function_interface`** — parameters/types/optionality/exceptions of an FM. The keystone: how the model learns to fill `rfc_call`.
- **`rfc_read_table`** — rows of a table/view (RFC_READ_TABLE) with column projection, WHERE, `row_count`/`row_skip` pagination; returns clean `{columns, rows}` (FIELDS-driven slicing). Read-only, with row/where caps and an optional table allow-list.
- **`rfc_call`** — guarded generic invoke: deny-by-default allow-list; values validated + encoded against discovered metadata before send; `requested_outputs` defaults to all E/T params; **mutations require `confirm_mutation: true` + a write-enabled destination**; no implicit commit.

Session management: one `pool.Pool[*lifecycle.Managed]` per destination; each
tool call takes a lease (metadata discovery on its own lease/cache), is
`ctx`-bounded, and `Discard()`s the lease on any protocol/transport error.
Credentials come from env/secret store at process start; a redaction layer keeps
the scramble seed and password-typed fields out of logs/results.

## v2 — dynamic tools, typed I/O, streaming, transactions, diagnostics

- **Dynamic tool generation** from discovered metadata: mint a typed MCP tool
  per allow-listed FM, its JSON schema generated from the interface (scalars →
  string/integer with length/range; packed/decfloat → decimal string; X/XSTRING
  → base64; structures → nested object; tables → array). Trade-off: precise/self-
  documenting vs tool-list bloat — offer lazy enable or keep `rfc_call` as
  default and promote hot FMs.
- **Typed I/O**: two-layer validation (JSON-schema shape, then ABAP-type/length);
  exceptions become structured `{error:"exception", name, message}`.
- **Streaming/pagination**: `RFC_READ_TABLE` cursor (`row_skip`), and MCP-level
  windowing of large FM tables via a short-TTL server-side cursor store.
- **Transactions/stateful sessions**: `rfc_session_begin/end` pin one pooled
  connection; explicit `rfc_commit`/`rfc_rollback` (BAPI_TRANSACTION_*); tRFC/qRFC
  `rfc_call_transactional`. Depends on the M6 transaction wire work.
- **Diagnostics**: `rfc_diagnostics` (pool stats), `rfc_trace_last_call` (redacted
  structured trace) — see `docs/rfc-assistance.md`.

Safety of exposing arbitrary FMs to an LLM: deny-by-default allow-list checked
against the *tool argument* (never against names embedded in returned data); no
auto-commit; metadata-driven validation before send; row/where/rate caps;
destination-name indirection for credentials; a low-privilege backend service
user as the real fence; `dry_run` for mutating calls.

## vsp (vibing-steampunk) integration — concrete

vsp is an **ADT ↔ MCP** bridge with a "hyperfocused" universal tool
`SAP(action, target, params)` that already routes through one safety chain
(`--read-only`, `--allowed-ops`, `--allowed-packages`). The RFC bridge is the
natural **third transport** alongside vsp's ADT and odata_mcp_go's OData.

Two integration shapes:

1. **Separate MCP server (loose coupling).** Ship the RFC bridge as its own MCP
   server; vsp's host (or any MCP client) connects to both. Cleanest boundary —
   RFC identity/credentials/policy stay in one auditable place. Recommended v1.
2. **RFC transport inside vsp (tight coupling).** Add an RFC action namespace to
   vsp's universal tool, e.g. `SAP(action="rfc.call", target="STFC_STRUCTURE",
   params={…})`, `SAP(action="rfc.read_table", …)`, `SAP(action="rfc.system_info")`,
   backed by this library. vsp gains **runtime** SAP access (call FMs, read
   tables, run BAPIs) on top of its **design-time** ADT access — a big
   capability jump, still SDK-free, reusing vsp's existing safety chain and
   destination/config model. This is the high-value option once the `rfc/`
   facade exists.

vsp already **transpiles** code into ABAP (TS→OO ABAP, WASM/QuickJS→ABAP); the
RFC bridge is complementary — it makes external systems *callable from* ABAP and
gives the assistant live runtime data. Together with `docs/polyglot-rfc-server.md`
(expose any binary to ABAP over RFC, with vsp deploying the generated proxies),
the three form a full agentic-SAP toolchain: **read/write/deploy code (vsp) +
call runtime + read data (RFC/OData) + bring any language into ABAP (polyglot
server).**

### Integration seam

Keep the bridge vsp-agnostic behind one interface so the coupling choice is
deferrable:

```go
type Platform interface {
    ResolveDestination(ctx, principal) (dest string, policy Policy, err error) // identity → SAP dest + creds
    Emit(ctx, DiagnosticEvent)                                                 // bridge → host telemetry
}
```

Credential resolution stays on the bridge/host side; the model/tool layer passes
an identity, never SAP passwords.

## Feasibility

v1 is buildable on today's M5/M6 surface — the only genuinely new code is
JSON↔ABAP marshalling driven by metadata, the policy engine, the destination/
secret registry, and MCP glue. The one structural prerequisite is the public
`rfc/` facade. v2's transaction/stateful tools depend on the still-open M6
tRFC/qRFC + server-context-reset work.
