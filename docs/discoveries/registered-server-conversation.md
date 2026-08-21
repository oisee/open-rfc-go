# The registered-server conversation, captured whole

2026-08-21. We made our own SDK-free stack impersonate nothing this time: we ran
SAP's own `rfcexec` as a genuine registered server and sniffed the real gateway
relay, so every frame below is what a real SAP gateway and a real ABAP client
actually exchange with a registered external server. Raw captures are in
`.private/gold/` (they carry live logon tickets, so they stay out of the repo).

## How it was captured

`gw/acl_mode = 1` and the gateway flags any registration that reaches it through
a different-host socket as a proxy (`RC 756`, "registration ... from PROXY host
... not allowed"). So registration only succeeds when the whole path stays on
`127.0.0.1` **inside** the container:

- `rfcexec` (from the kernel, `/usr/sap/A4H/SYS/exe/uc/linuxx86_64`) registered
  `VSP_TICKET_CATCH` at the local gateway, via a one-port sniffer also running
  inside the container (`127.0.0.1:3099 → 127.0.0.1:3300`). Peer stays local, no
  proxy flag, registration OK.
- An SM59 type-T *Registered Server Program* destination (`Send Assertion
  Ticket` on) was connection-tested. The gateway relayed the client's logon to
  `rfcexec` over the sniffer, so it landed in the capture.

An `reginfo` with `P TP=* HOST=* ACCESS=* CANCEL=*` had to exist and be re-read
(SMGW → Expert Functions → External Security → Re-Read NI ACL) first.

## The frame sequence (per call)

| dir | fn | bytes | meaning |
|---|---|---|---|
| C→S | `0x0b` | 64 | NI/gateway handshake |
| C→S | `0x68` | 512 | gateway `GW_NORMAL_CLIENT` region |
| C→S | `0xd1` | 80 | **the registered-server request** (gateway → server) |
| S→C | `0xcf` | 125 | **the ACCEPT** — a 125-byte record, function code **`0xcf`** |
| S→C | `0x03` | 1716–2800 | **the relayed CPIC logon** the ABAP client composed |
| C→S | `0x08` | 241 | the server's async reply |
| C→S | `0xd2` / `0x0b` / `0xd1` | 80 | keepalive / next request |

Two things this corrects about our earlier blind attempt (`serve_ticketcatch`):
the accept function code is **`0xcf`, not `0xca`**, and the request the server
must answer is **`0xd1`**. We now hold the real accept frame to copy instead of
zeroing a reject.

## The ticket is the size of the logon

The `0x03` logon comes in two size families, and the split is exactly the
ticket:

- **1716 / 1802 bytes** — *Do Not Send Logon Ticket*: no ticket.
- **2674–2800 bytes** — *Send Assertion Ticket* (various target systems /
  without-reference): ~1000 bytes larger.

So a forwarded assertion ticket adds ≈1000 bytes to the logon. It is **not** the
base64 `MYSAPSSO2` string the HTTP header carried, and **not** a raw PKCS#7 blob
(the signedData OID `2a8648…0702` does not appear in the frame), so inside the
CPIC logon the ticket sits in a field encoded differently — UTF-16, a nested
container, or compressed. Locating that field to the byte is the open task; the
captures needed for it are saved (`logon-with-ticket.hex`,
`logon-no-ticket.hex`), and it needs no live system.

## Why this matters

This is the first time we hold the logon a **real ABAP client** composes for a
registered server — the exact record our `internal/cpic` decoder wants to read,
and the only place a forwarded ticket rides classic RFC. With the real accept
(`0xcf`) frame captured, `serve_ticketcatch` can be finished from a positive
example rather than guesswork, and once the ticket field is located, ticket-based
RFC logon becomes implementable.
