# Type-T registered server: answering a live FM call

Status: **DONE — a live `CALL FUNCTION ... DESTINATION` is answered end to end.**
`Z_CALL_RFC` on A4H drove `Z_DOUBLE` and `Z_GREET` over `ZSNIFF_TCP` into
`ServeTypeT`; both returned correct values with `sy-subrc = 0`:

```
GREETING = "Hello, Claude! — from open-rfc-go type-T server"
RC01 = 0   RC02 = 0   RESULT = 42        (Z_DOUBLE: 21 -> 42)
```

This is the first time an open-rfc-go registered server answered real function
calls through the live SAP gateway. Below is the whole journey — the handshake,
the trusted CUT, the GUID, the serializer, and the reply record framing that was
the last piece.

Every fact here is `[capture]` from our own live runs against A4H (a4h-105 in the
`i5`/`ubullama` docker host), decoded clean-room. No credentials appear on this
path — a registered server authenticates no one.

## The route

SM59 destination `ZSNIFF_TCP` (from `RFCDES`):

```
H=%%RFCSERVER%% G=localhost g=3388 N=VSP_TICKET_CATCH J=X n=A4H p=001 5=1100 7=6041   (type T)
```

So `CALL FUNCTION ... DESTINATION 'ZSNIFF_TCP'` connects to gateway
**localhost:3388** inside the container and asks for program `VSP_TICKET_CATCH`.
Our impersonator (`cmd/rfc-typet`, `rfcserver.ServeTypeT`) listens there and
plays gateway + registered server in one. `5=1100` / `7=6041` are the SM59
Special-Options serializer/flag bits — see the fast-serializer note below.

Driver: `Z_CALL_RFC` (FUGR `ZTST`) takes `DESTINATION`, `N`, `NAME` and does
`CALL FUNCTION 'Z_DOUBLE' DESTINATION dest` (n→result) then
`CALL FUNCTION 'Z_GREET' DESTINATION dest` (name→greeting). `rc01`/`rc02` are the
two `sy-subrc` values; in its EXCEPTIONS map `communication_failure=1`,
`system_failure=2`. rfcexec answering an unknown FM returns `rc=2`
(system_failure), *not* a comms error — a reachable server, unimplemented FM.

## The handshake (client leg, direct into us)

```
C->S 64B  NI route hello          -> S->C 64B  ack   (level byte 29: 0e->0f; caps cb->fb; identical otherwise)
C->S 453B ALLOCATE (fn 0xca)      -> S->C 125B accept (fn 0xca)
C->S 8B   NI_PING                 -> S->C 8B   NI_PONG      <-- required; without it the client blocks here
C->S ...  CUT (fn 0xcb)           -> S->C ...  reply (fn 0xcb)
```

Two things a static template got wrong, both found by diffing a live accept
against the ALLOCATE that provoked it:

1. **The accept must echo per-connection fields from the ALLOCATE.** A real
   gateway copies, into its 125-byte accept:
   - `accept[4:6]  <- allocate[4:6]`   the request uid (APPC header)
   - `accept[28]   <- allocate[28]`    a request flag
   - `accept[88:113] <- ` the request GUID block: a `0x00`, 16 ASCII-hex chars,
     an 8-byte token, sitting just past the destination/program strings.
   The conversation id (`accept[40:48]`, ASCII digits) is the gateway's to
   assign. Our first attempt reused a captured accept verbatim, so those bytes
   held another conversation's values (and, worse, leftover stack-pointer bytes
   from the reject frame it was cut from); the client could not correlate the
   accept to its request, pinged to check liveness, then gave up.

2. **NI_PING must be answered with NI_PONG.** In the relayed capture the gateway
   answered the ping transparently so it never showed on that leg; direct into
   us, the client pings right after the accept and blocks until the pong.

## The trusted-RFC call CUT

For a trusted destination (`J=X n=A4H p=001`) the first — and only — CUT carries
the logon/trust context **and** the function call in one frame (2125 B for
`Z_DOUBLE`):

```
[80B F_SAP_SEND header][~1544B logon/trust context chain][function-call CUT]
```

The function-call CUT does **not** sit at a fixed offset; its prefix
`05 02 00 00` is past the trust context (offset 1624 in our capture, function
name `Z_DOUBLE` right after). `ServeTypeT` finds it by scanning for that prefix
from offset 48. A CUT with no such prefix is a pure logon/handshake turn and is
simply accepted.

`DecodeCutFunctionRequest` on that slice recovers `FunctionName=Z_DOUBLE`,
`KernelRelease=758` — but **no import parameters** (see below).

## The response GUID

A real reply on this path carries **no RFC GUID at all** — no `0x0514` session
field, no node-suffix. `EncodeCutFunctionResponseS4` adds a swapped session GUID
(correct and required for the type-3 path), and doing so here triggers the
client's **"RFC GUID inconsistency discovered"** ABAP runtime error. Passing
`guid=nil` for the type-T reply clears it: the client accepts the response
envelope and moves past the GUID check.

## The serializer: request solved, reply framing remains

**First attempt (fast serializer).** With the accept fixed, the ping answered,
the call CUT found and decoded, and the GUID omitted, the client accepted our
response header but blocked. The cause was the serializer: the request arrived
**fast-serialized**, which our classic codec does not read — the parameter
section was the fast-RFC container form (`\TYPE=I`, `TABLE_LINEN`,
length-prefixed ASCII names, the value `15 00 00 00` = 21 for `N` inline), not
classic `0x0201`/`0x0203` UTF-16 name/value fields, so `DecodeCutFunctionRequest`
returned zero imports.

**Fix that worked — force classic on the destination.** Setting the SM59
Special-Options serializer to **Classic serializer, no delta** (the `5=` bits in
`RFCDES`; the destination was `5=1100`) flipped the request to classic. Result,
live: the call CUT now decodes fully — `N raw = 15 00 00 00` (21, LE), so our
`Z_DOUBLE` handler computes `RESULT = 42` and dispatches. **The request side is
solved.**

**Last piece — the reply record framing (solved).** Our content was already
correct: decoding our 502-byte reply as a field chain gives the full classic-S4
envelope with `0x0203 = 2a 00 00 00` (RESULT = 42). The block was the **APPC
record header**, not the content. `wrapFSapSend` builds the type-3 / S4 data
record (protocol=2, gatewayID=1); the client did not recognise it as its response
and hung. A real type-T reply record differs:

```
real type-T reply hdr: 06 cb 07 00 <uid=63ac> 00 00 ...   protocol=7, echoes request uid, gatewayID=0
our wrapFSapSend hdr:  06 cb 02 00 <uid>       00 01 ...   protocol=2,                    gatewayID=1
```

and further in info3=1, info4=2, padding=0x0100, info=1, vector=8, appcRC=18. The
fix (`wrapFSapSendTypeT`) clones a captured 80-byte type-T reply header, patches
only the per-connection fields (uid echoed from the request `[4:6]`, conversation
id), and appends our unchanged content. With that, the client consumed the reply
and returned `RESULT=42` / `GREETING` / `sy-subrc=0`.

(The 347-byte reply we had been comparing against was rfcexec's `system_failure`
*exception* — its content envelope legitimately differs from a success reply, so
the "no 0x0503/0x0500" observation was a red herring; the header was the real
lesson, and it is content-agnostic.)

### Progress ladder (each rung proven live, in order)

1. NI hello / ack — ok
2. ALLOCATE accept with echoed uid + GUID block — ok
3. NI_PING / NI_PONG — ok
4. trusted CUT located and `Z_DOUBLE` decoded — ok
5. GUID omitted, response accepted past the consistency check — ok
6. classic serializer → imports decode, `N=21`, `RESULT=42` — ok
7. type-T reply record header (`wrapFSapSendTypeT`) → client consumes the reply,
   `RC=0` — **ok**

## Still open (not blocking)

- **Fast serializer.** This worked only because the destination was forced to the
  classic serializer. A default (fast) destination sends fast-RFC params and
  expects a fast-RFC reply, which we do not yet speak. Implementing the fast-RFC
  container decode/encode remains the general unlock (it blocks full type-3 calls
  too). The `5=` bits in `RFCDES` select the serializer per destination.
- **Non-echo fields in the accept / reply header** are cloned from a live capture
  (uid, conv, GUID block are patched; the rest stands). Good enough for this
  driver; a fully synthesised header would need each field explained.

## Tooling

- `internal/rfcserver/serve_typet.go` — `ServeTypeT`: hello/ack, ALLOCATE accept
  built from the client's ALLOCATE (`typeTBuildAccept`), NI_PONG, CUT prefix
  finder (`indexCutRequest`), dispatch, reply.
- `cmd/rfc-typet` — listener + a dispatcher with `Z_DOUBLE`, `Z_GREET`,
  `STFC_CONNECTION` handlers; dumps every frame to JSONL.
- Deploy: cross-compile `GOOS=linux GOARCH=amd64`, `docker cp` into `a4h-105`,
  run on `127.0.0.1:3388`. The relay sniffer (`rfc-relaysniff` 3388->3300) is the
  positive-capture instrument for the same flow through the real gateway+rfcexec.
