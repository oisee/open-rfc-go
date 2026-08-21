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

## Resolved: the CPIC field that carries a forwarded ticket

Derived from our own capture (`.private/gold/`), clean-room — this field is named
in neither Wireshark's dissectors nor pysap.

A forwarded SAP logon ticket rides the CPIC logon in field **`0x0670`**, chained
`prev=0x0002 → 0x0670 → 0x0114` (right after the repeated local-host field,
immediately before the client). Its value is the ticket's base64 text — the same
`MYSAPSSO2` string the HTTP header carries — laid down as **UTF-16LE** (each
base64 character followed by `0x00`). In the 2800-byte capture the field value is
992 bytes = 496 UTF-16 units ≈ 495 base64 characters ≈ 370 decoded ticket bytes.

Stripping the UTF-16LE wrapper yields byte-for-byte the same ticket object the
HTTP path carries: the `02`/`4103` header, user/client/issuing-system/timestamp,
the assertion recipient field, and the `0xff` PKCS#7 SignedData signature. Only
the transport wrapper differs between HTTP (`MYSAPSSO2` cookie/header, plain
base64) and classic RFC (CPIC field `0x0670`, base64-as-UTF-16LE).

Cross-checked across four SM59 modes: no-ticket logons omit the field entirely;
with `Send Assertion Ticket` the field appears and the recipient inside it tracks
the configured target (`A4H`/`001` vs the bogus `SNIFF`/`999`), which is what
confirms it is the assertion ticket and not something else.

**What this unlocks.** Both directions are now specified against
`internal/cpic`: *reading* a ticket from a logon (find `0x0670`, drop the
UTF-16LE wrapper, base64-decode, then the existing ticket TLV reader), and
*composing* one into an outbound logon (emit `0x0670` in the chain with the
ticket base64 as UTF-16LE) — the latter is ticket-based classic-RFC logon, the
thing the whole hunt was for. The byte-level detail is in the gitignored
`.private/gold/ticket-in-logon.md`; the raw ticket stays out of the repo.

## Live test of direct ticket logon: rejected (and why it is informative)

2026-08-21, our client run on the host `.105` against the real gateway
(`127.0.0.1:3300`). A **password** logon succeeds (`rfc ping` → ok). A **ticket**
logon with the same client — the ticket in field `0x0670` as UTF-16LE base64,
password field omitted — is **rejected**, and the raw response (dumped with
`SAP_DEBUG_LOGON=1`) ends in the text **"Name or password is incorrect (repeat
logon)"**.

So the `0x0670` encoding that a real SAP client composes and a registered server
*accepts* (proven with rfcexec) is **not** accepted in a **direct** client→AS
logon. The direct format differs — likely the user field must be empty (the
ticket carries the user), or a different field/flag marks a ticket logon. Pinning
it needs a positive capture of a real *direct* ticket logon, which we do not have
(our captures are the relayed, server-side form). Parked there.

Second finding, independently fixable: that reject is a well-formed CPIC error
response (status field `0x0161`, message field `0x0402` carrying the text), yet
`DecodeInitialLogonResponse` reports "initial CPIC logon error response has an
invalid preamble". Its error-preamble grammar is too strict for this reject
variant, so genuine "wrong credential" failures surface as a parser error rather
than the message. The raw bytes are captured; fixing the grammar turns the
useless error into "Name or password is incorrect". Backlog.

The `SAP_DEBUG_LOGON` env (dumps the raw logon response to stderr) is kept as a
debugging aid for exactly this kind of diagnosis.

## Leg 2: the client-sent side, and why our direct attempt was wrong

2026-08-21. With rfcexec registered at the real gateway, the ABAP client was
routed through an **in-container** sniffer (127.0.0.1:3388 → 127.0.0.1:3300) so
the whole path stayed local — a sniffer on any other host is refused by the
gateway ACLs (`RC 756` for registration via a proxy, `RC 748` "access to
registered server is not permitted" for a proxied client). Inside the container
the connection test went green and the client's own frames were captured.

**The client-sent conversation:**

| dir | fn | bytes | meaning |
|---|---|---|---|
| C→S | `0x03` | 64 | gateway record |
| C→S | `0xca` | 453 | ALLOCATE |
| S→C | `0xca` | 125 | **the client-side ACCEPT** (function code `0xca`, not the server-side `0xcf`) |
| C→S | `0xcb` | 2714–2800 | **F_SAP_SEND** carrying the client's logon + the ticket |
| S→C | `0xcb` | 241 | reply |

Two conclusions.

1. **The client-side ticket is the same field, same encoding.** In the `0xcb`
   frame the ticket sits at the same place: chain `prev=0x0002 → 0x0670 → len
   0x03d8 (984) → value`, the value being the base64 ticket as UTF-16LE
   (`41 00 6a 00 51 00…` = "AjQ…"). So the gateway passes the ticket field
   through unchanged; only the frame's function code differs (`0xcb` client-side
   vs `0x03` server-side).

2. **The ticket rides a CUT, not the initial CPIC logon.** This is why our
   direct attempt failed. We put `0x0670` into the *initial* logon
   (`EncodeInitialLogonRequest`, the `0x03` record whose chain uses tags
   `0x0101`/`0x0111`/`0x0114`/`0x0117`). The real client authenticates with the
   ticket in a **separate `F_SAP_SEND` CUT** whose chain is `0x0002`-linked and
   carries `0x0670`. The initial logon and the ticket-bearing CUT are different
   frames with different chains; the AS ignored `0x0670` in the initial logon and
   fell through to password auth ("Name or password is incorrect").

### Two wins, one wall

- **Win — role Y (gateway impersonation) is now buildable from a positive
  example:** the client-side ACCEPT is `0xca`, 125 bytes, captured here.
  `serve_ticketcatch` can send the real accept instead of the guessed one.
- **Win — the ticket encoding is confirmed identical on both sides:** `0x0670`,
  UTF-16LE base64, so `internal/cpic`'s ticket reader/writer is correct as a
  *field* codec.
- **Wall — direct client→AS ticket logon still unknown.** Every capture we have
  is the *registered-server* flow, where the ticket rides an `F_SAP_SEND` CUT. A
  direct client→AS logon (what JCo does with `jco.client.mysapsso2`) may place
  the ticket in the initial logon or in a CUT of its own — we cannot tell without
  a capture of a real direct ticket-logon client (a JCo/SDK program through the
  sniffer). Until then, ticket-based *direct* RFC logon stays unimplemented; the
  field codec and the HTTP ticket paths are done.
