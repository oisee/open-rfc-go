# What an SM59 type-H destination actually sends

Captured 2026-08-21 with a plain HTTP catcher on our own network, against an
SM59 *HTTP Connection to ABAP System* destination pointed at it. Four logon
modes, one connection test each, then three repeats of the last two. The
system is A4H, SAP_BASIS 758.

This was a side experiment in a hunt for how an SAP logon ticket travels over
**classic RFC**. It did not answer that — but it settled what the HTTP path
does, which is worth its own note because none of it is published either.

## Every request carries these, regardless of mode

| Header | Content |
|---|---|
| `Sap-Passport` | 460 bytes, the distributed-trace passport — and it carries the **calling user's name** in clear text |
| `Sap-Dtrace` | `v=1,i=RFC 793 <id>,r=<id>,c=<id>,s=1` |
| `User-Agent` | `SAP NetWeaver Application Server (1.0;758)` — release included |

Worth knowing on its own: a destination that sends *no* credential still tells
the receiver who is calling, and which release is calling.

## What each mode adds

**Do Not Use a User, no ticket** — nothing at all beyond the three above.

**Basic Authentication** — `Authorization: Basic …` over whatever transport the
destination is configured for. On a destination with SSL inactive that is a
password in reversible encoding over plain HTTP, and SM59 will not stop you.

**Send assertion for dedicated target system** — a header, not a cookie:

```
MYSAPSSO2: AjQxMDM…
```

500 base64 characters, and the same shape as the ticket a browser logon puts in
the `MYSAPSSO2` cookie: version `02`, code page `4103`, the user in UTF-16, the
client, the system id and a creation timestamp. The assertion variant is bound
to the target named in the destination (System ID and Client), hence "for
dedicated target system". Same format, different carrier — so a ticket obtained
either way is usable against the HTTP surfaces.

**SAP RFC Logon (with a ticket configured)** — both `MYSAPSSO2` and one more:

```
SAP-R3AUTH: 646A3078…   (2064 hex characters)
```

That decodes in three layers: hex → base64 → the ASCII string `v=1U,` followed
by 768 hex characters, i.e. **384 opaque bytes**.

Those 384 bytes are not a field chain and not readable. Measured entropy is
7.45 bits per byte over 200 distinct values, and **all three captures differ from
the first byte** — so the envelope is encrypted and carries a nonce or a
timestamp. 384 bytes is exactly a 3072-bit block, which is consistent with the
credentials being sealed with the target system's key from its PSE.

## The ticket itself decodes — and that is useful on its own

Both the browser cookie and the assertion header parse cleanly with our own
code: a 5-byte header (`02` version, `4103` code page) followed by a TLV run,
`id(1) len(2 BE) value`. The fields, confirmed against two real tickets:

| id | meaning | form |
|---|---|---|
| `0x01` | user | UTF-16 |
| `0x02` | client | UTF-16 digits |
| `0x03` | issuing system | UTF-16 |
| `0x04` | creation timestamp | UTF-16 `YYYYMMDDhhmmss` |
| `0x0f` | (portal codepage marker) | |
| `0xff` | signature | **PKCS#7 SignedData** (`06 09 2a 86 48 86 f7 0d 01 07 02`) |

The assertion ticket differs from the browser one exactly where it should: it
adds `0x10` = the **recipient system** (`A4H`) and `0x08`, and drops the
browser's `0x06` portal flag. So an assertion ticket is *addressed* — the target
is inside the payload, which is what "for dedicated target system" means — while
the browser ticket is general. Same envelope, one extra field.

The practical consequence: because we can read the envelope, we can tell whose
ticket we hold, for which system, and whether it has expired, before making any
network call — a clear error instead of a rejected logon. That is worth a small
reader on its own, independent of whether ticket-based *RFC* logon ever lands.

## The conclusion for the ticket hunt

Nothing here reveals how a ticket rides classic RFC. The HTTP path either sends
the ticket as a header in the clear, or seals the credential into an opaque
block — neither tells us the CPIC field. What it does confirm is the ticket
*format* is one format across carriers, and that our HTTP routes (ADT, and the
SOAP RFC endpoint) can be driven by a ticket alone, which was verified
separately the same day.

## Addendum: the type-T lead, and what capturing it would take

SM59's **type-T** (TCP/IP Connection) destination has, unlike type-3, a
*Logon with Ticket* block with **Send Logon Ticket Without Ref. to a Target
System** and **Send Assertion Ticket for Dedicated Target System** — and type-T
is CPIC, not HTTP. So this is the one destination that would put a ticket into
the classic-RFC logon, in the very field we could not observe.

Capturing it is not a configuration click, though:

- **Registered Server Program** routes the call — and the ticket — to an
  external program that has registered a Program ID at the gateway. Our server
  (`internal/rfcserver`, `cmd/orfc-lab`) only listens and sniffs; it does not dial
  out to a gateway and register (`F_SAP_GW` register, `TP_NAME`). The gateway
  record codec exists (`internal/gateway`), so the registration handshake is
  implementable, but it is real work.
- **Start on Explicit Host** (the other activation type) starts a program on the
  named host via the gateway; it never reaches us.

So the capture needs one of: (a) gateway registration implemented in our server,
so it can *be* the registered program the gateway hands the ticket to; or (b) a
genuine registered server routed through our sniffer's 3300, so the ticket
passes in transit.

**Tried the cheap capture, 2026-08-21 — it cannot work.** A type-T *Registered
Server Program* destination was pointed with its Gateway Host at our sniffer and
*Send Assertion Ticket* switched on, then connection-tested. The idea was that
the routing request would pass through the sniffer even though nothing is
registered. It did pass through — a 453-byte ALLOCATE to the gateway naming the
destination, the program id `VSP_TICKET_CATCH`, `%%RFCSERVER%%` and the caller
`CLAUDE` — but **it carries no ticket**. The test failed at ALLOCATE
(`CM_ALLOCATE_FAILURE_RETRY`, "Transaction program not registered") *before* any
CPIC conversation opened. The logon, and with it the ticket, is only sent once a
registered server accepts the conversation. So a sniffer in transit can never
see it: our server has to *be* the registered program and accept, and only then
does the ticket-bearing logon arrive.

**Parked, deliberately.** A ticket already authenticates every HTTP path (ADT
and the SOAP RFC endpoint), verified live, and the ticket format already
decodes. Ticket-based *classic-RFC* logon is completeness, not capability — it
matters only on a landscape with neither an open HTTP surface nor a gateway
password, which we do not currently have. When it does matter, the decisive
experiment is: implement gateway registration in `orfc-lab`, point a type-T
destination's Program ID at it with *Send Assertion Ticket* on, and read the
inbound logon.

## Addendum 2: impersonating the gateway got us to conversation-established

2026-08-21, topology B built and run live (`cmd/orfc-ticketcatch`,
`internal/rfcserver/serve_ticketcatch.go`). Our process played the gateway for a
type-T *Registered Server Program* destination whose Gateway Host pointed at us.
Progression across runs, each a real SM59 connection test:

| Our accept | SAP result |
|---|---|
| none (real gateway, nothing registered) | `679` transaction program not registered |
| 80-byte header, zeroed return codes | `225` Unknown CPIC function (frame too short / malformed) |
| **125-byte real reject frame with return codes zeroed** | **no ABEND — conversation established**; SAP then sent two `NI_PING` frames |
| same, plus we answer `NI_PING`→`NI_PONG` | `757` client has not answered PING messages (timing/role dependent) |

So the accept verdict reading was right: a real gateway response frame with
`appcReturnCode`/`sapReturnCode` zeroed is accepted, and the client proceeds past
allocate. The wall is the next phase — NI keepalive cadence and which side sends
the logon first in the registered-server model — which behaves differently run to
run and which we are guessing blind, because we hold no capture of a *successful*
registered-server conversation.

**Decisive next step, not more guessing: a positive capture.** Run the kernel's
`rfcexec` as a registered server (`rfcexec -a VSP_TICKET_CATCH -g <gwhost> -x
sapgw00`) on the SAP host, call it through the sniffer, and record the real
accept → keepalive → logon sequence. Then `serve_ticketcatch` reproduces it
exactly, and the client's logon — carrying the ticket when the destination is set
to send one — lands in our existing `internal/cpic` decoder. The instrument that
got us this far is committed; only the reference capture is missing.
