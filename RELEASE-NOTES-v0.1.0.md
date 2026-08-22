# open-rfc-go v0.1.0 — first tagged preview

SAP classic synchronous RFC in pure Go — client **and** server. No SAP NW RFC
SDK, no native library, no cgo. One static binary.

This is the first version with a name and a number. It is a **research preview**:
`0.x` means the API may move, and classic RFC has no transport encryption, so
nothing here belongs on an untrusted network without a VPN under it.

## The binary is `saprfc`

One binary, two modes — the CLI, and `saprfc mcp` for the Model Context Protocol
server, in the shape `vsp` uses. It was previously built from `cmd/rfc`; the name
changed so it says whose RFC it speaks and does not collide with the IETF sense
of "rfc" on a `PATH`. The lab tools keep their `rfc-*` names; they are
development tools, not something you install.

```sh
go build -o saprfc ./cmd/saprfc

export SAP_ASHOST=a4h.example SAP_SYSNR=00 SAP_CLIENT=001 \
       SAP_USER=DEVELOPER SAP_PASSWORD='…'

./saprfc ping
./saprfc info
./saprfc call STFC_CONNECTION '{"REQUTEXT":"hi"}'
./saprfc describe STFC_STRUCTURE          # the FM interface as an MCP-tool JSON Schema
```

Full setup, including a user that actually works and the errors you will hit:
**[`docs/quickstart-a4h.md`](docs/quickstart-a4h.md)**.

## As MCP, with a surface you chose

```sh
./saprfc mcp --read-only --expose 'STFC_*,BAPI_USER_GET_DETAIL' --hide '*_DELETE'
```

| flag | effect |
|---|---|
| `--read-only` | no writes at all |
| `--safe` | blocks FMs whose names read as mutating, and `BAPI_TRANSACTION_COMMIT` |
| `--allow-commit` | permits that commit again inside `--safe`, when you mean it |
| `--expose 'MASK,…'` | green list — only matching FMs become tools |
| `--hide 'MASK,…'` | red list, subtracted from the green list |
| `--max N` | cap the number of generated tools |

Without `--expose` you get the generic tools and an assistant can still reach
anything through `call`. **`--expose` is what makes the surface small; a green
list you wrote beats a red list you hoped was complete.**

A note on the serializer, since it comes up: there is no "force classic" switch,
because the client is classic by construction. Forcing classic is a *server*-side
capability — in that role we issue the logon accept, and can decline fast.

## What is new in this release

**All four serializers are selectable on demand, and what each writes is
recorded.** The controlling fact was not where we expected: a destination has two
independent knobs and the second overrides the first, which is why one can
display *"Fast serializer"* while storing a value that means something else.

**The fast serializer's record grammar is decoded**, from a controlled
differential rather than from staring at big frames — vary one parameter, hold
the rest, capture both ends:

- framing is **tag-dependent**; there is no single length rule
- `INT4` little-endian, fixed width; `char`, `STRING`, `XSTRING` at **one byte
  per unit** — not UTF-16, not padded to the declared width
- the version handshake is deterministic and negotiates `FAST_SER_VERS = 3`
- payloads above **512 bytes are compressed**, intrinsic to the serializer rather
  than a switchable transport feature

Decoded, not produced — and since the client negotiates classic, none of that
compression has ever applied to the client leg.

**The roles are written down** — the client setup machine, all seven server
roles, both handshake shapes, and the keepalive rule. Internally they now share
one frame classifier; the eight keepalive bytes had been declared twice under
different names, which is how one role gets a fix the others miss.

**Classic is complete for the synchronous path**, re-verified live: scalar
`STRING` and `XSTRING`, and deep structures carrying both.

## What is not in it

No tRFC/qRFC/bgRFC transactional units. No SNC, so no encrypted transport. The
fast serializer is decoded but not produced, and its above-512 compression is not
implemented — which blocks nothing today, but would matter for a server answering
a fast-serialized client with a large payload.

## Credit

`open-rfc-go` is a Go port of **[`open-rfc`](https://github.com/marianfoo/open-rfc)**
by [Marian Zeis](https://github.com/marianfoo), Apache-2.0. Reimplementing a
protocol with no public specification is mostly a matter of knowing which byte
matters, and that knowledge is expensive; `open-rfc` published it and made
everything downstream possible, this port included. Where we went further, it was
from a position `open-rfc` established.

Every ported file records the upstream file and commit it came from, and the
modifications made, in [`docs/provenance.md`](docs/provenance.md). Everything not
in that table — the fast serialization codec, the server side, xRFC and the
recursive metadata graph, callbacks, pinned sessions, the debugger and the ADT
tunnel — was developed clean-room from our own captures.

`open-rfc`'s maintainers do not support this port and are not responsible for it.
SAP, ABAP and SAP S/4HANA are trademarks of SAP SE; this project is independent
of and not endorsed by SAP SE.
