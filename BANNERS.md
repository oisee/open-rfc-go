# Banners

Every headline this project has put at the top of its README, newest first.

The README carries only the latest one at full size; the rest live here so the
front page stays readable without the history being thrown away. Each was true
on the date it carries, and none has been rewritten since — where a later
finding corrected an earlier claim, the correction is in
[`CHANGELOG.md`](CHANGELOG.md) and the [discovery notes](docs/discoveries/),
not applied retroactively here.

---

## 2026-08-22: The serializers, mapped, and the fast one decoded

**we can now select any of SAP's four serializers on demand and
read what each puts on the wire.** The destination has two independent knobs and
the second overrides the first, which is why a destination can read *"Fast
serializer"* while storing something else entirely. With that settled, a
controlled differential — vary one parameter, hold the rest, capture both ends —
gave the **fast serializer's record grammar**: tag-dependent framing, `INT4`
little-endian and fixed-width, `char`/`STRING`/`XSTRING` at **one byte per unit**
(not UTF-16, not padded), and the version handshake negotiating `FAST_SER_VERS = 3`.
Payloads above **512 bytes are compressed** — intrinsic to that serializer, not a
switchable transport feature. Our client negotiates classic, so none of it applies
to the client leg today. → [`serializer-selection.md`](docs/discoveries/serializer-selection.md)
· [role state machines](docs/role-state-machines.md)

---

## 2026-08-20: Call any SAP function module, from Go, the shell, or as MCP tools

**the client now calls essentially any FM, and an MCP server turns
a live SAP system into tools an AI can use.** Every scalar type (incl.
STRING/XSTRING, DATE/TIME, packed DEC/TIMESTAMP, FLOAT, UTCLONG), flat **and
deep** structures & tables (STRING/XSTRING via xRFC), and both **classic and
fast** serialization on decode — all round-tripped live against A4H, pure Go.
On top of that: `orfc describe <FM>` renders any function module as an **MCP-tool
JSON Schema**, `orfc call` runs any FM from plain JSON, and **`orfc mcp`
auto-exposes a curated set of FMs as real MCP tools** — point it at your SAP and
an assistant can call RFC directly. Zero SAP libraries.

---

## 2026-08-19: Both directions live against real SAP, with zero SAP libraries

**open-rfc-go now speaks classic RFC _as the server_, not only the client.**
A live SAP system (A4H) ran a real ABAP program of six parametrized calls and
**open-rfc-go answered every one, `rc=0`**, as the server:

| `RFC_PING` | `RFC_SYSTEM_INFO` | `STFC_CONNECTION` | `STFC_STRUCTURE` | `RFC_READ_TABLE` | `STFC_STRING` |
|---|---|---|---|---|---|
| rc=0 | rc=0 · A4H | rc=0 · echo + callback | rc=0 · struct+table | rc=0 · 17×2 | rc=0 |

A single Go endpoint answers **every SM59 test button green** — Connection Test,
Unicode Test, Fast Serialization Test — across three serialization modes. And
open-rfc-go now **decodes S/4HANA classic responses end to end** — scalars and
tables alike (real T000 rows, field lists, structures) — pure Go. The **client**
leg is live-proven too. → the wire story: [`docs/discoveries/0001`](docs/discoveries/0001-live-type3-server.md)
