# The ABAP debugger over classic RFC — what is now possible

Live run: **A4H**, SAP_BASIS 758, kernel 793, client 001, user `CLAUDE`, 2026-08-21.
Client: `open-rfc-go` (pure Go, no SAP SDK). Driver: `vsp rfc debug`.
Server side: function group `ZADT_DEBUG`, ~200 lines of ABAP.

## 1. What was actually done, in order

1. `vsp rfc debug` pins **one** RFC conversation and holds it.
2. `state` twice → same `roll`, `calls` rising. Through the pool instead: two
   different `roll` values, `calls = 1` both times. The pinning is real, and the
   pool is not a substitute for it.
3. `bp SAPLZADT_DEBUG/LZADT_DEBUGU01 9` → the breakpoint lands in
   `ABDBG_EXTDBPS` and reads back **from a different session**.
4. `catch 150` blocks. `ZADT_DEBUG_LOOP` is then called over a **second RFC
   connection** — it stops at the breakpoint, and the listener returns it.
5. Attach reports `procname ZADT_DEBUG_LOOP`, `post_mortem:false`. The stack is
   the real RFC entry chain:

   | level | program | include | line | procname |
   |---|---|---|---|---|
   | 1 | SAPLZADT_DEBUG | LZADT_DEBUGU01 | 9 | ZADT_DEBUG_LOOP |
   | 2 | SAPLZADT_DEBUG | LZADT_DEBUGU01 | 1 | ZADT_DEBUG_LOOP |
   | 3 | SAPMSSY1 | SAPMSSY1 | 189 | REMOTE_FUNCTION_CALL |
   | 4 | SAPMSSY1 | SAPMSSY1 | 35 | %_RFC_START |

6. Three `step over` walk lines **9 → 14 → 15 → 17**, the stack following each.
7. After `detach` the debuggee runs on: `TVARVC ZADT_DEBUG_COUNTER` advanced, so
   its `UPDATE` and `COMMIT WORK` really executed.

No SAP GUI, no Eclipse, no WebSocket, no ICF, no CSRF, no SAP NW RFC SDK.

## 2. Why a pinned connection is the whole trick

`IF_TPDAPI_SERVICE~ATTACH_DEBUGGEE` returns a **reference** to
`IF_TPDAPI_SESSION`, and control services, stack handler and data services all
hang off that reference. An object reference lives in an ABAP roll area. So the
requirement is not "push messaging" and not "a WebSocket" — it is *one ABAP
session that survives between attach and step*.

That is exactly what a pinned classic-RFC conversation is. `Client.Pin` leases
one connection out of the pool and every call rides it, so the roll area is the
same on call 1 and call 50. The ADR that parked vsp's WebSocket debugger
diagnosed the failure as "attach/step fail due to session mismatch" — the
mismatch disappears by construction here.

## 3. What needs no ABAP at all

The debugger's *read* half is three ordinary transparent tables:

| Table | Answers |
|---|---|
| `ABDBG_ACTIVATION` | who is parked in the debugger right now, in which program/include/line, on which server — **including short dumps** (`DUMPID`), i.e. post-mortem debuggees |
| `ABDBG_EXTDBPS` | which external breakpoints exist, for whom, set by whom |
| `ABDBG_LISTENER` | which listeners are registered, on which server and context |

So `vsp rfc debuggees`, `vsp rfc breakpoints` and `vsp rfc watch` work against a
stock system with nothing installed. "Did anyone hit my breakpoint, and where" —
the question people usually open a debugger to ask — is answerable today.

Note `ABDBG_EXTDBPS.BREAKPOINT` and `.ATTRIBUTES` are LOB columns, so
`RFC_READ_TABLE` returns the keys but not the payload; program and line come
from the facade's `bp_list`.

## 4. Does the facade have to exist? — the Eclipse question

Not necessarily, and this is the most interesting thread left.

Eclipse does not use TPDAPI directly either. It drives **ADT REST resources** —
`/sap/bc/adt/debugger/listeners`, `.../debuggee`, steps, stack, variables — which
are themselves ABAP code calling TPDAPI. The only reason a stateless HTTP client
cannot use them is that ADT keeps the debug session in an ABAP roll area and
selects it with a `sap-contextid` cookie; vsp's HTTP layer models this
(`SessionStateful`) but never turned it on for debugger calls.

Over RFC there is nothing to simulate: **the roll area *is* the conversation.**
And we can already tunnel ADT REST through RFC — `SADT_REST_RFC_ENDPOINT`, which
takes a full HTTP envelope and now decodes correctly (SAP wraps the XSTRING body
in base64 at 76 columns; our decoder used to reject the newlines, so every body
over 57 bytes failed). `vsp rfc adt GET /sap/bc/adt/discovery` returns 299 KB of
service document today.

So the experiment now one command away, in the debug REPL:

```
dbg> adt POST /sap/bc/adt/debugger/listeners?debuggingMode=user&requestUser=CLAUDE
```

If ADT's stateful resources hold their state across two calls on one pinned
conversation, then **the whole debugger works through SAP's own surface with no
Z objects at all** — which matters enormously for other people's systems, where
"install this function group first" is a non-starter.

A cheaper probe of the same question, worth running first because it is
instantly verifiable: `adt POST …?_action=LOCK` in one call and a source PUT
with that handle in the next. If the lock survives, everything stateful in ADT
survives.

Either way the facade keeps two advantages: typed parameters instead of XML, and
it works on systems where the ADT debugger resources are absent or blocked.

## 5. What it cost — five findings that were not in any doc

1. **`MODIFICATION_SUPPORT='NoModification'` is not a permission.** SAP's own
   `IF_ADT_LOCK_RESULT` documents it as *"modification support is not needed
   because object is not in SAP/partner namespace in a customer system"* — the
   healthy answer for every Z object. Read as "read-only", it makes every class
   on a normal system uneditable through ADT. (PR #152 fixes it; found
   independently on NW 7.50.)
2. **ADT locks do not survive between MCP tool calls** — each call is its own
   stateful session. Only single-call workflows work, above all `EDITSOURCE`,
   which locks, updates, activates and unlocks in one go.
3. **An `EDITSOURCE` on a function module accepts more than the FUNCTION block.**
   A `CLASS … DEFINITION`/`IMPLEMENTATION` written before `FUNCTION` compiles
   into the function pool. That is how a self-contained facade fits in one
   module — necessary here, because classes could not be locked at all.
4. **The Remote-Enabled flag is the one thing ADT cannot set.** It lives in the
   module's properties; neither the `fmodules` creation payload nor a source PUT
   touches it. It has to be ticked once in SE37.
5. **Never hand a TPDAPI table to `/UI2/CL_JSON`.** Serialising the raw stack
   table hangs the call, and with the long timeout a blocking listener needs,
   the caller then waits out its whole RFC timeout with the debuggee attached.
   Project the fields you need.

Plus one that only looks like a bug: **`detach` kills the conversation.**
`END_DEBUGGER` ends the debugger's ABAP session together with the debuggee's, so
the transport reports `CM_NO_DATA_RECEIVED` and there is no reply to read. That
is the success case.

## 6. What this unlocks next

- **Post-mortem debugging** — `ABDBG_ACTIVATION.DUMPID` exposes captured short
  dumps as attachable debuggees. vsp cannot do this at all today.
- **Variables** — the one facade operation deliberately not shipped yet; it needs
  a typed walk over `IF_TPDAPI_DATA_{SIMPLE,STRUC,TABLE,OBJREF}`, and eight
  hard-coded `SY-*` fields would be worse than nothing.
- **`vsp-debugd`** — with `vsp rfc debug` holding the session for its own
  lifetime, a daemon is now only needed to survive an MCP client restart.
- **Conditional and non-line breakpoints** — the facade already passes a
  condition; statement, exception and message breakpoints are one method each.

## 7. Authorizations — the honest caveat

The user here is `SAP_ALL`, so nothing above demonstrates a least-privilege
setup. A real user needs `S_DEVELOP` with `ACTVT = 03` on `OBJTYPE = DEBUG`,
and — to debug another user's session — the external debugging authority for
that user. Note that SAP's own RFC-enabled debugger modules (`TPDAPI_TEST_*`)
answer a missing authorization with a silent `RETURN` and an empty result rather
than an error, which is easy to misread as "the API does not work".
