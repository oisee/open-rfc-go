# Polyglot RFC server: expose any binary to ABAP over RFC

Design / research note. Original idea (Alice, 2026-08-19), building on the
milestone-6 RFC-server direction (see `docs/porting-plan.md`) and the SDK-free
client already proven live at milestone 5. Nothing here needs the NW RFC SDK.

## The idea in one paragraph

Point the tool at an **external library or binary** — Python, C, Swift, Rust,
Go, anything with a callable contract — and it (1) stands up an **RFC server**
that registers at a SAP gateway and dispatches incoming RFC calls into that
library, and (2) **generates the ABAP wrappers** so the library looks native
from ABAP: a proxy remote-enabled function module, or a proxy interface
`ZIF_*` plus a class `ZCL_*` whose methods each turn into
`CALL FUNCTION … DESTINATION`. An ABAP developer then writes
`zcl_mylib=>do_thing( … )` and the body executes Python/Rust/C. It is a
polyglot FFI-to-ABAP bridge: **any binary becomes a callable ABAP service.**

## Why this is possible without the SDK

The RFC *server* direction (SAP → us) reuses everything the client already
does, mirrored: NI framing, APPC/CPIC field chains, the CUT request/response
codec, structure and xRFC codecs, and the ABAP-exception envelope. What is new
is the gateway **registration** handshake (register a Program ID / TPNAME) and
a **serving loop** that decodes an inbound call, dispatches, and encodes a
reply. See the RFC-server track in `docs/porting-plan.md`.

## The full loop

```
                       ┌──────────────────────── generate ────────────────────────┐
                       │                                                            │
  external library     │   this tool (Go)                         SAP system       │
  (py/c/rust/swift/…)  │   ┌───────────────────────┐             ┌──────────────┐  │
   contract ───────────┼──▶│ 1. introspect contract│             │  ABAP        │  │
   (signatures)        │   │ 2. gen RFC-server      │             │              │  │
                       │   │    dispatch → lib      │             │  ZCL_MYLIB   │  │
                       │   │ 3. gen ABAP proxy ─────┼── vsp/ADT ──▶  (deployed)  │  │
                       │   │ 4. register at gateway │◀── RFC ─────  CALL FUNCTION │  │
   invoke ◀────────────┼───┤    (Program ID)        │   DESTINATION   … DEST 'X'  │  │
   result ─────────────┼──▶│    serve + dispatch    │─── reply ───▶  zcl_mylib=> │  │
                       │   └───────────────────────┘             └──────────────┘  │
                       └────────────────────────────────────────────────────────────┘
```

1. **Introspect the external contract.** Extract function signatures and types
   from the target: a C header, a Rust `pub` API / `cbindgen`, a Python module
   (type hints / `inspect`), a Swift interface, a Go package (`go/types`), a
   protobuf/gRPC service, or an OpenAPI spec. Normalise to a small intermediate
   representation (name, params with direction + type, return, errors).
2. **Generate the Go RFC-server dispatch.** For each contract function, a
   handler that decodes the RFC imports/tables into the library's argument
   types, invokes the library (in-process via cgo/FFI, or out-of-process via a
   subprocess/IPC for isolation), and encodes the result — or maps a library
   error to an ABAP exception.
3. **Generate the ABAP proxy.** Two shapes, caller's choice:
   - a **proxy function module** `Z_MYLIB_DO_THING` (remote-enabled interface in
     DDIC, thin body that is really served externally), or
   - a **proxy interface + class**: `ZIF_MYLIB` with a method per contract
     function and `ZCL_MYLIB` implementing each as
     `CALL FUNCTION 'Z_MYLIB_DO_THING' DESTINATION 'MYLIB' …`.
   The generator emits typed ABAP signatures from the same IR, so the ABAP side
   is strongly typed, not a generic blob call.
4. **Register at the gateway.** The RFC server dials the gateway and registers
   its Program ID; SM59 has a registered-server destination `MYLIB` pointing at
   it. No inbound firewall hole — the server dials out.
5. **Deploy the ABAP proxy with [vsp](https://github.com/oisee/vibing-steampunk).**
   vsp is an ADT↔MCP bridge that can create and activate DDIC/FM/class objects.
   It writes the generated `ZIF_*`/`ZCL_*`/FM into the system, closing the loop
   without a human copy-pasting ABAP.

## Where it fits with the sibling tools (all SDK-free, all Go, all oisee)

| Tool | Bridge | Direction |
|---|---|---|
| **open-rfc-go** (this) | classic **RFC** | runtime FM calls, and — this idea — an RFC **server** exposing external libs |
| **vibing-steampunk (vsp)** | **ADT** ↔ MCP | read/write/deploy/debug ABAP dev objects; also **transpiles** TS→OO ABAP and WASM/QuickJS→ABAP |
| **odata_mcp_go** | **OData** ↔ MCP | SAP business data over OData |

vsp already moves code *into* ABAP (transpilation). This idea is the
complementary direction: keep the code in its native language and make it
*callable from* ABAP as a first-class RFC service. The two compose — vsp is the
deployment arm for the ABAP proxies this generator emits.

## Value

- **Any ecosystem's libraries become ABAP-callable** without rewriting them in
  ABAP and without the NW RFC SDK: ML/inference (Python), crypto/parsers
  (Rust/C), image/media, cloud SDKs, whatever.
- **Typed, native-feeling ABAP API** (`ZCL_*` methods), not stringly-typed glue.
- **Isolation options**: in-process (cgo, fast) or out-of-process (subprocess,
  safe — a crashing library can't take the RFC server down).
- **Testability**: the same server, driven by the in-process peer (see the
  RFC-server track), gives offline end-to-end tests of the whole path.

## Discovery & marshalling (the translator-interfacer)

An exposed library must be both *callable* and *self-describing* over RFC, so a
client can discover it at runtime without a pre-defined DDIC interface. The core
is one language-neutral **interface descriptor (IR)** per exposed function:

```
FuncDescriptor{ Name string; Params []{ Name, Class (I|E|C|T), Type, Optional } }
Type = scalar(exid,len) | structure(fields[]) | table(rowType)   // recursive
```

Built once, at registration time, from the target's own contract — a C header
(cbindgen), Python type hints (`inspect`), a Go package (`go/types`), a
protobuf/gRPC service, or an OpenAPI spec — or hand-declared in Go.

Three pieces sit on top of the existing `Dispatcher`:

1. **A registry** mapping each function name to its `FuncDescriptor` plus a
   library adapter `func(ctx, map[string]any) (map[string]any, error)`.
2. **Metadata handlers** — register handlers for `RFC_GET_FUNCTION_INTERFACE`,
   `RFC_GET_STRUCTURE_DEFINITION`, and `RFC_METADATA_GET` that answer *from the
   registry*. This is what makes the wrapper discoverable: any RFC client (this
   project's client, PyRFC, node-rfc — and ABAP once its DDIC proxy is
   generated) resolves the interface at runtime, SDK-free, with no predefined
   DDIC. The new code here is the server-side **encoders** for the FUNINT /
   FIELDS metadata rows — the mirror of the milestone-4 decoders.
3. **An IR-driven marshaller** — decode each inbound CUT parameter (bytes →
   native Go scalar/structure/table via `structure`/`xrfc`/`classicrfc`), call
   the adapter, and encode the outputs back into a CUT response per the
   descriptor. This is the **inverse of the milestone-7 client marshaller in
   `rfc/call.go`**, so much of that logic is reused, not rewritten.

So: registry (IR + adapters) + metadata encoders + IR marshaller + the existing
`Dispatcher`. The ABAP-side DDIC proxy (FM / `ZIF_*` / `ZCL_*`) is generated
from the *same* IR and deployed via vsp, but is optional — a generic RFC client
discovers and calls the wrapper directly.

## Open questions / risks

- **Type mapping fidelity** external ↔ ABAP: strings/encodings, decimals
  (ABAP packed vs native float — reuse the client's decimal-string decision),
  structures/tables ↔ external structs/arrays, optionality, binary (XSTRING).
- **Error/exception round-trip**: library error → ABAP exception with a useful
  message (ties into the P0 typed-error work).
- **Lifecycle & concurrency**: registered-server connection count, one call at
  a time per conn, subprocess pool for out-of-process libraries.
- **Security**: the registered server runs external code on behalf of ABAP
  callers — authorization, sandboxing of the target library, allow-list of
  exposed functions, no secrets in generated artifacts.
- **SNC**: still the production-encryption gap shared with the rest of the port.

## Status

Idea, recorded as a track that sits on top of the milestone-6 **RFC-server**
work (first milestone there: an in-process peer + registered-server handshake).
The contract-introspection and ABAP/Go codegen are new components; the wire
serving path reuses the existing codecs. Deployment integrates with vsp.
