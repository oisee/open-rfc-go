# Getting `orfc` talking to an A4H system

A4H is SAP's *ABAP Platform Developer Edition* — the free trial appliance most
people have to hand. Everything here works the same against any ABAP system; A4H
is just the one with predictable defaults.

Nothing is installed on the server. No SAP NW RFC SDK, no native library, no
`LD_LIBRARY_PATH`, no cgo. One static Go binary and a user that may log on.

## 1. What you need from the system

| | typical A4H value | where it comes from |
|---|---|---|
| host | the appliance's address | your VM or container |
| instance number | `00` | the appliance default |
| **gateway port** | **`3300`** | `3300 + instance`. Not 8000, not 44300 — RFC does not use the HTTP ports |
| client | `001` | A4H ships 000/001; developers use 001 |
| user | e.g. `DEVELOPER` | any dialog or service user |
| language | `EN` | |

Check the port is actually open before blaming the protocol:

```sh
nc -z -w3 <host> 3300 && echo open
```

If that fails it is firewall, port-forwarding or the instance number — no client
can help you.

### A user that can do the job

A dialog user works. A **service** user is better for automation: it cannot be
locked out by a failed interactive logon, and it keeps your own account out of
the blast radius.

The authorization object is `S_RFC`. For read-only exploration
`ACTVT 16` on `FUGR` for the function groups you intend to call is enough. If
`orfc info` works but `orfc describe` does not, it is almost always `S_RFC`
on the metadata function group (`SDIFRUNTIME`/`RFC1`), not a bug.

## 2. Build it

```sh
go build -o orfc ./cmd/orfc
```

Go 1.26+, no other toolchain. The binary is self-contained; copy it anywhere.

## 3. Connect

Either environment variables:

```sh
export SAP_ASHOST=a4h.example        # host
export SAP_SYSNR=00                  # instance -> port 3300
export SAP_CLIENT=001
export SAP_USER=DEVELOPER
export SAP_PASSWORD='…'
export SAP_LANG=EN

./orfc ping
./orfc info
```

Or a `.rfc.json` with named systems, so you stop typing them:

```json
{
  "systems": {
    "a4h": {
      "ashost": "a4h.example",
      "sysnr": "00",
      "client": "001",
      "user": "DEVELOPER",
      "password": "…",
      "lang": "EN"
    }
  }
}
```

```sh
./orfc -s a4h info
```

Flags beat environment, environment beats `.rfc.json`. Keep `.rfc.json` out of
version control — it holds a password.

> **Classic RFC has no transport encryption and no peer authentication.** The
> logon password is *scrambled*, not encrypted, and anyone on the path recovers
> it. Use this on a network you trust, or put a VPN under it. See
> [`SECURITY.md`](../SECURITY.md).

## 4. First calls

```sh
./orfc info                                        # RFC_SYSTEM_INFO
./orfc call STFC_CONNECTION '{"REQUTEXT":"hi"}'    # echo, the classic smoke test
./orfc describe STFC_STRUCTURE                     # the FM interface as a JSON Schema
./orfc search 'BAPI_USER_*'                        # find RFC-enabled FMs
./orfc read-table T000 --top 5                     # a table, as rows of columns
```

`STFC_*` are SAP's own test function modules and exist on every system —
`STFC_CONNECTION`, `STFC_STRUCTURE`, `STFC_STRING`, `STFC_XSTRING`,
`STFC_DEEP_STRUCTURE`, `STFC_DEEP_TABLE`, `STFC_CHANGING`. They are the fastest
way to prove a leg works without writing any ABAP.

## 5. As an MCP server

`orfc mcp` speaks the Model Context Protocol over stdio, so an assistant can
call function modules directly. For Claude Code, a `.mcp.json` in the project:

```json
{
  "mcpServers": {
    "a4h": {
      "command": "/abs/path/to/orfc",
      "args": ["mcp", "--safe", "--expose", "STFC_*,RFC_READ_TABLE,BAPI_USER_GET_DETAIL"],
      "env": {
        "SAP_ASHOST": "a4h.example",
        "SAP_SYSNR": "00",
        "SAP_CLIENT": "001",
        "SAP_USER": "DEVELOPER",
        "SAP_PASSWORD": "…"
      }
    }
  }
}
```

### Decide what it may touch — before you point it at anything

The access controls compose, and the sensible order to reach for them is:

| flag | effect |
|---|---|
| `--read-only` | no writes at all |
| `--safe` | blocks function modules whose names read as mutating, and `BAPI_TRANSACTION_COMMIT` |
| `--allow-commit` | permits the commit again inside `--safe`, when you mean it |
| `--expose 'MASK,…'` | **green list** — only matching RFC-enabled FMs become tools (`*` wildcard) |
| `--hide 'MASK,…'` | red list, subtracted from the green list |
| `--max N` | cap how many tools are generated |

Without `--expose` you get the generic tools (`info`, `ping`, `describe`,
`search`, `read_table`, `call`) and the assistant can still reach anything
through `call`. **`--expose` is what makes the surface small**, and
`--read-only` is what makes it safe. A green list you wrote beats a red list you
hoped was complete.

Exposed tools carry an `outputSchema` and read-only / destructive hints derived
from the FM's own interface, so a well-behaved client can reason about them
before calling.

## 6. When you want to see the wire

`orfc` is the client. Two more tools matter here.

**`orfc-srv` answers calls** — point an SM59 destination at it and your own Z
function module can be tested against this implementation, no second SAP system
needed:

```sh
go run ./cmd/orfc-srv -mode typet -listen :3300   # SM59 type T (registered server)
go run ./cmd/orfc-srv -mode type3 -listen :3313   # SM59 type 3 (an ABAP system)
```

It answers `Z_DOUBLE`, `Z_GREET`, `STFC_CONNECTION` and `RFC_PING`. Anything else
raises `FU_NOT_FOUND`, which keeps the conversation alive — so the request the
client sent is still in the capture, which is usually what you wanted.

**The lab tools are for looking at the protocol itself:**

```sh
go run ./cmd/orfc-lab -target-host <your-sap-host>
#   3200/3300  transparent sniffer -> the real system (-dump cap.jsonl)
#   3313       a generating server, so SM59 can talk to *us*

go run ./cmd/orfc-viewer cap.jsonl            # decoded transcript, values redacted
go run ./cmd/orfc-viewer -html cap.jsonl      # self-contained HTML inspector
```

To make the system dial *you*, create an SM59 **type 3** destination whose
Target Host is the machine running the lab and whose instance number matches the
port it listens on. Then any `CALL FUNCTION … DESTINATION 'YOURS'` in ABAP lands
in the capture.

`orfc-viewer` never touches a live system; it reads a capture file. `-values`
includes decoded values and may therefore print credentials.

> Captures contain the logon frame. Treat a `.jsonl` dump as a secret.

## 7. Things that go wrong, and what they mean

| symptom | cause |
|---|---|
| connection refused on 3300 | wrong instance number, or the gateway is not reachable — check with `nc` first |
| `ping` works, `describe` does not | `S_RFC` on the metadata function group |
| "unknown parameter X" | the FM's parameter is not called what you think — run `describe` and read the real names |
| logon rejected for a user that works in SAP GUI | the user may be locked to a client, or is a *communication* user where a dialog user is required (or the reverse) |
| a call hangs | some FMs block server-side by design; raise `--timeout` rather than assuming a protocol stall |
| works for one system, fails for another | user stores are per-system; a user existing on one is no promise about the other |

## What this does not do

No tRFC/qRFC/bgRFC transactional units, no SNC (so no encrypted transport), and
the fast serializer is decoded but not produced. The client negotiates classic,
which is why none of the fast serializer's own compression applies to it. See
[`docs/roadmap.md`](roadmap.md) for the ranked plan and
[`docs/discoveries/serializer-selection.md`](discoveries/serializer-selection.md)
for what each serializer actually puts on the wire.
