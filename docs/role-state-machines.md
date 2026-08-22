# Role state machines

Who may send what, when, in each role this project implements. Written because
the roles were drifting: every server had its own inline frame test, the same
eight keepalive bytes were declared twice, and only the client half had a state
machine written down at all.

Two things are separate throughout and worth keeping separate while reading:

- **Role** — client or server, and which server. Decides who speaks first and who
  owes a reply.
- **Mode** — which serializer the conversation ended up in. Decides how a payload
  is encoded, and nothing else. See
  [serializer selection](discoveries/serializer-selection.md).

A role is a state machine over frames. A mode is a codec choice inside one state.
Conflating them is what makes "why did this conversation stall" hard to answer.

## The shared frame vocabulary

Every role classifies received frames the same way, through `Classify` in
`internal/rfcserver/niframe.go`:

| kind | shape | who owes a reply |
|---|---|---|
| `FrameGatewayRecord` | 64 bytes | server: echo with the capability ack |
| `FrameNIPing` | `NI_PING\0` | **whoever receives it** — always |
| `FrameNIPong` | `NI_PONG\0` | nobody |
| `FrameShortKeepalive` | 8 bytes, neither | nobody |
| `FrameData` | anything else | depends on the record |

Classification is deliberately shallow — length and prefix only, never a payload
decode. A role has to know what kind of frame arrived *before* it may assume the
conversation is in a state where the payload parses.

`IsNIRouteHello` splits the one ambiguous case: the 64-byte NI route hello that
opens a registered-server conversation looks exactly like the gateway's
normal-client record except for its `0x02 0x03` prefix. Only the type-T role
needs the distinction.

## Keepalives — the failure that looks like a hang

**A peer that sends `NI_PING` blocks until it sees `NI_PONG`.** Forgetting to
answer one does not fail the conversation; it stalls it. The symptom is a server
that "stops receiving" partway through a working handshake, which reads like a
decode bug and is not one.

The rules:

- Answer every `NI_PING` with `NI_PONG`, in any state, in any role. It is not
  gated on the conversation having reached a particular point.
- Never answer a `NI_PONG`. Two peers that both reply to pongs loop forever.
- Never treat an 8-byte frame as a record. `FrameShortKeepalive` exists so an
  unrecognised short frame is dropped rather than parsed as a header.

The concrete case that cost us: in the type-T role the client pings *after* the
ALLOCATE accept and blocks on the pong. Without it the conversation never reaches
the first function-call CUT, and the server sits waiting for a call the client
will never send.

Both frames are declared once, in `niframe.go`. They were previously declared
twice with different names, which is exactly the kind of divergence this file
exists to prevent.

## Client role

`internal/appc.ClientSetupStateMachine` — the admitted direct-CPIC setup
sequence, and the only role with an explicit machine.

```
new ──Initialize──▶ initialize-pending ──▶ initialized
                                              │
                        ┌─────────────────────┴──────────────┐
                   SetTpName                          SetPartnerLuName
                        ▼                                    │
                     tp-set ──SetPartnerLuName──▶ partner-set ◀┘
                                                      │
                                                  Allocate
                                                      ▼
                                              allocate-pending ──▶ ready
                                                                    │
        ┌───────────────────────────┬───────────────────────────────┤
     SapSend                  AsyncSendData                    Deallocate
        │                           ▼                               ▼
        │                   send-continuation                    closed
        │                     │ SendData │ Receive
        ▼                     └────┬─────┘
   response-pending ◀───────────────┘
        │
   CompleteResponse
        ▼
      ready
```

Enforced invariants worth knowing, because each corresponds to a real wire rule
rather than defensive coding:

- `F_SAP_SEND` cannot *start* a streamed outgoing message.
- `F_ASEND_DATA` must be followed by `F_RECEIVE`.
- A streaming `F_SEND_DATA` must be followed by its acknowledgement.
- The streamed outgoing `F_RECEIVE` terminator must be the final data record.
- Completing a response is legal only from `response-pending`; from anywhere else
  the machine goes to `closed` rather than guessing.

An illegal transition returns `ErrState` naming both the function code and the
state, which is what makes a misordered conversation debuggable from the error
alone.

## Server roles

Seven roles share one skeleton and differ in where each reply comes from. The
skeleton is: classify the frame, answer the handshake, then answer function
requests until the peer deallocates.

| role | reply source | what it is for |
|---|---|---|
| `ServeReplay` | a recorded server side, in order | request-driven replay; paces to the client, cannot deadlock |
| `ServeGenerate` | one baked accept plus a fixed CUT | function-agnostic; enough for an SM59 Connection Test |
| `ServeSmart` | baked templates, session tokens mirrored | first role that generates rather than replays |
| `ServeContentAddressed` | per-function recorded scripts, tokens patched | handles the `STFC_CONNECTION` server→client callback |
| `ServeConscious` | registered Go handlers | the real server: decode the call, dispatch, encode the reply |
| `ServeTypeT` | registered Go handlers | registered external server (type T); no user auth in the handshake |
| `ServeTicketCatch` | three fixed answers, everything else logged | an instrument, not a server: holds the conversation open to capture a client logon |

Two handshake shapes across those roles:

**Type 3 (the client dials us as an ABAP system).**

```
64B gateway record   -> echo with bytes 29 and 55 set to the capability ack
CPIC init record     -> logon-accept, mirroring this session's conversation id
                        and RFC GUID; which accept depends on the init length
function request CUT -> decode, dispatch, reply
```

**Type T (we register at the gateway and the client is routed to us).**

```
64B NI route hello   -> hello ack
APPC ALLOCATE 0xca   -> 125B accept, both return codes zero, conversation id assigned
  (client pings here — answer it or the conversation stops)
first CUT 0xcb       -> the "logon accepted" turn; a registered server
                        authenticates nobody, so there is no credential check
further CUT 0xcb     -> decode, dispatch, reply
```

The type-T logon turn is the one that surprises: a registered server never sees a
password, because the gateway already authenticated the caller. There is no
serializer negotiation in this role either — which is why the fast-serialization
work had to be done from the type-3 side.

## Unknown function modules

`Dispatcher` answers a call it has no handler for with the `FU_NOT_FOUND`
exception rather than closing the connection. That is deliberate and it is what
makes the server useful as an instrument: a probing client gets a clean ABAP
exception, the conversation survives, and the request CUT it sent is already in
the capture.

## Where the code is

| concern | file |
|---|---|
| frame classification, keepalives | `internal/rfcserver/niframe.go` |
| client setup machine | `internal/appc/conversation.go` |
| dispatch and exceptions | `internal/rfcserver/dispatcher.go` |
| the roles | `internal/rfcserver/serve_*.go`, `replay.go`, `content.go` |
| wire constants | `internal/rfcserver/wire_constants.go` |
