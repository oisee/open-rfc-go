# Surface inventory: src/transport/

> Mechanical inventory of open-rfc @ commit 847036d, generated as porting input. Every claim cites path:line. See ../provenance.md.

Citation paths are relative to the open-rfc checkout root. Claim types are marked:
**(code)** = quoted from source; **(test)** = quoted from a test assertion or test name;
**(comment)** = quoted from a source comment; **INFERRED:** = my reading, not written down.

Files in scope: `connectivity-socks5-ni.ts` (178 lines), `connectivity-socks5-tunnel.ts` (944),
`message-server-resolver.ts` (346), `ni-socket.ts` (961), `saprouter-ni.ts` (109),
`saprouter-route.ts` (388), `saprouter-tunnel.ts` (731).

---

## src/transport/ni-socket.ts

The core. One TCP connection carrying bounded NI length-prefixed records.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `NiTransportErrorCode` | type | `export type NiTransportErrorCode =` `\| "NI_ABORTED"` `\| "NI_CONNECT_FAILED"` `\| "NI_CONNECT_TIMEOUT"` `\| "NI_CONNECTION_CLOSED"` `\| "NI_PROTOCOL_ERROR"` `\| "NI_RECEIVE_TIMEOUT"` `\| "NI_WRITE_TIMEOUT"` `\| "NI_WRITE_FAILED";` | src/transport/ni-socket.ts:9-17 |
| `DEFAULT_MAX_NI_QUEUED_PAYLOAD_LENGTH` | const | `export const DEFAULT_MAX_NI_QUEUED_PAYLOAD_LENGTH = 64 * 1024 * 1024;` | src/transport/ni-socket.ts:19 |
| `DEFAULT_MAX_NI_QUEUED_FRAME_COUNT` | const | `export const DEFAULT_MAX_NI_QUEUED_FRAME_COUNT = 1_024;` | src/transport/ni-socket.ts:20 |
| `DEFAULT_NI_WRITE_TIMEOUT_MS` | const | `export const DEFAULT_NI_WRITE_TIMEOUT_MS = 30_000;` | src/transport/ni-socket.ts:21 |
| `DEFAULT_NI_CLOSE_TIMEOUT_MS` | const | `export const DEFAULT_NI_CLOSE_TIMEOUT_MS = 5_000;` | src/transport/ni-socket.ts:22 |
| `NiTransportError` | class | `export class NiTransportError extends Error` with `readonly code: NiTransportErrorCode;` `override readonly cause: unknown;` and `constructor(code: NiTransportErrorCode, message: string, cause?: unknown)` | src/transport/ni-socket.ts:24-33 |
| `NiSocketConnectOptions` | interface | fields `host: string`, `port: number`, `connectTimeoutMs?`, `maxPayloadLength?`, `maxQueuedPayloadLength?`, `maxQueuedFrameCount?`, `writeTimeoutMs?`, `closeTimeoutMs?`, `noDelay?: boolean`, `family?: 4 \| 6` (all `readonly`) | src/transport/ni-socket.ts:36-49 |
| `NiConnectedSocket` | interface | structural stream surface: `destroyed?`, `closed?`, `readableEnded?`, `writableEnded?`, `remoteAddress?`, `remotePort?`, `localAddress?`, `localPort?`, `isPaused()`, `pause()`, `resume()`, `destroy(error?)`, `end()`, `write(chunk, callback)`, `on("data"\|"error"\|"end"\|"close", …)`, `once("close", …)` | src/transport/ni-socket.ts:56-78 |
| `NiSocketAdoptOptions` | interface | `socket: NiConnectedSocket`, `initialData?: Uint8Array`, `maxPayloadLength?`, `maxQueuedPayloadLength?`, `maxQueuedFrameCount?`, `writeTimeoutMs?`, `closeTimeoutMs?` | src/transport/ni-socket.ts:80-90 |
| `NiReceiveOptions` | interface | `readonly timeoutMs?: number;` `readonly signal?: AbortSignal;` | src/transport/ni-socket.ts:92-95 |
| `NiTimerHandle` | type | `export type NiTimerHandle = number \| object;` | src/transport/ni-socket.ts:97 |
| `NiTimerScheduler` | interface | `setTimeout(callback: () => void, delayMs: number): NiTimerHandle;` `clearTimeout(handle: NiTimerHandle): void;` | src/transport/ni-socket.ts:104-107 |
| `NiSocketTransport` | class | `export class NiSocketTransport` — constructor is `private` | src/transport/ni-socket.ts:242, 258 |
| `NiSocketTransport.connect` | static method | `static async connect(options: NiSocketConnectOptions, signal?: AbortSignal, scheduler: NiTimerScheduler = systemTimerScheduler,): Promise<NiSocketTransport>` | src/transport/ni-socket.ts:284-288 |
| `NiSocketTransport.adopt` | static method | `static adopt(options: NiSocketAdoptOptions, signal?: AbortSignal, scheduler: NiTimerScheduler = systemTimerScheduler,): NiSocketTransport` | src/transport/ni-socket.ts:397-401 |
| `.state` | getter | `get state(): NiSocketState` (`type NiSocketState = "open" \| "closing" \| "closed";` — **not exported**) | src/transport/ni-socket.ts:553, 124 |
| `.remoteAddress` / `.localAddress` | getter | `get remoteAddress(): string \| undefined` / `get localAddress(): string \| undefined` | src/transport/ni-socket.ts:557, 561 |
| `.localPort` / `.remotePort` | getter | `get localPort(): number \| undefined` / `get remotePort(): number \| undefined` | src/transport/ni-socket.ts:565, 569 |
| `.assertNoQueuedFrames` | method | `assertNoQueuedFrames(): void` | src/transport/ni-socket.ts:578 |
| `.send` | method | `async send(payload: Uint8Array, signal?: AbortSignal): Promise<void>` | src/transport/ni-socket.ts:597 |
| `.receive` | method | `async receive(options: NiReceiveOptions = {}): Promise<Buffer>` | src/transport/ni-socket.ts:687 |
| `.close` | method | `async close(): Promise<void>` | src/transport/ni-socket.ts:761 |

### Timeouts, bounds, and defaults

| Name | Value (verbatim) | What it bounds | Citation |
|---|---|---|---|
| `DEFAULT_MAX_NI_QUEUED_PAYLOAD_LENGTH` | `64 * 1024 * 1024` | Aggregate complete payload bytes retained before `receive()` drains them **(comment: src/transport/ni-socket.ts:41)** | src/transport/ni-socket.ts:19 |
| `DEFAULT_MAX_NI_QUEUED_FRAME_COUNT` | `1_024` | Complete NI frames retained before `receive()` drains them **(comment: src/transport/ni-socket.ts:43)** | src/transport/ni-socket.ts:20 |
| `DEFAULT_NI_WRITE_TIMEOUT_MS` | `30_000` | Per-`send()` write completion | src/transport/ni-socket.ts:21, 387 |
| `DEFAULT_NI_CLOSE_TIMEOUT_MS` | `5_000` | Wait for socket `"close"` after `end()` in `close()` | src/transport/ni-socket.ts:22, 388 |
| connect timeout default | `options.connectTimeoutMs ?? 10_000` | TCP connect | src/transport/ni-socket.ts:143, 303 |
| any `*Ms` field | `!Number.isSafeInteger(value) \|\| value < 0 \|\| value > 0x7fff_ffff` rejects with `` `${field} must be an integer in 0..2147483647` `` | all millisecond fields incl. `receive({timeoutMs})` | src/transport/ni-socket.ts:126-130, 689 |
| `port` | `options.port < 1 \|\| options.port > 0xffff` → `"port must be an integer in 1..65535"` | connect endpoint | src/transport/ni-socket.ts:137-141 |
| `family` | `options.family !== 4 && options.family !== 6` → `"family must be 4 or 6"` | address family | src/transport/ni-socket.ts:145-150 |
| `maxPayloadLength` default | `options.maxPayloadLength ?? DEFAULT_MAX_NI_PAYLOAD_LENGTH` — imported from `../protocol/ni.js`, **value out of scope** | single decoded NI payload | src/transport/ni-socket.ts:4, 152, 380 |
| `maxQueuedPayloadLength` default | `Math.min(maxPayloadLength, DEFAULT_MAX_NI_QUEUED_PAYLOAD_LENGTH)` | retained queue bytes | src/transport/ni-socket.ts:159-160, 382-385, 430-431 |
| `maxQueuedFrameCount` validity | `!Number.isSafeInteger(v) \|\| v < 1` → `"maxQueuedFrameCount must be a positive safe integer"` | queue depth | src/transport/ni-socket.ts:193-197 |
| `receive` timeout default | `const timeoutMs = options.timeoutMs ?? 0;` | **0 means no receive timer at all** | src/transport/ni-socket.ts:688, 734 |
| timeout value `0` | `if (connectTimeoutMs !== 0 && !settled)`, `if (this.#writeTimeoutMs !== 0 && !settled)`, `if (this.#closeTimeoutMs !== 0)` | 0 disables each timer | src/transport/ni-socket.ts:349, 635, 787 |
| `noDelay` default | `socket.setNoDelay(options.noDelay ?? true);` | Nagle | src/transport/ni-socket.ts:309 |

### Error codes

| Code | Trigger condition | Citation |
|---|---|---|
| `NI_ABORTED` | `connect()` called with an already-aborted signal (`"NI connection was aborted before it started"`) | src/transport/ni-socket.ts:296-301 |
| `NI_ABORTED` | signal fires during `connect()` (`"NI connection was aborted"`) | src/transport/ni-socket.ts:343-344 |
| `NI_ABORTED` | `adopt()` with already-aborted signal, before listener handoff — socket destroyed first | src/transport/ni-socket.ts:468-474 |
| `NI_ABORTED` | signal observed aborted *during* listener handoff, or *while resuming*, in `adopt()` | src/transport/ni-socket.ts:515-521, 534-541 |
| `NI_ABORTED` | `send()` with already-aborted signal — **calls `#fail`, killing the transport** | src/transport/ni-socket.ts:607-614 |
| `NI_ABORTED` | signal fires during a pending `send()` | src/transport/ni-socket.ts:631-632 |
| `NI_ABORTED` | `receive()` with already-aborted signal — **calls `#fail`** | src/transport/ni-socket.ts:699-706 |
| `NI_ABORTED` | signal fires during a pending `receive()` — **calls `#fail`** | src/transport/ni-socket.ts:720-723 |
| `NI_CONNECT_FAILED` | socket `"error"` during connect: `` `failed to connect NI socket to ${options.host}:${options.port}` `` | src/transport/ni-socket.ts:336-342 |
| `NI_CONNECT_FAILED` | `scheduler.setTimeout` throws during connect (`"NI connect timer scheduler failed"`) | src/transport/ni-socket.ts:366-374 |
| `NI_CONNECT_TIMEOUT` | `` `NI connection to ${options.host}:${options.port} timed out after ${connectTimeoutMs} ms` `` | src/transport/ni-socket.ts:353-358 |
| `NI_CONNECTION_CLOSED` | socket `"error"` after adoption/construction (`"NI socket failed"`) | src/transport/ni-socket.ts:276-280 |
| `NI_CONNECTION_CLOSED` | `adopt()` on a terminal socket (`"cannot adopt a terminal NI socket"`) | src/transport/ni-socket.ts:443-449 |
| `NI_CONNECTION_CLOSED` | constructor throws while installing listeners (`"adopted NI socket rejected lifecycle listeners"`) | src/transport/ni-socket.ts:487-494 |
| `NI_CONNECTION_CLOSED` | socket became terminal during listener handoff / while resuming | src/transport/ni-socket.ts:495-502, 542-549 |
| `NI_CONNECTION_CLOSED` | `socket.resume()` throws in `adopt()` (`"adopted NI socket could not be resumed"`) | src/transport/ni-socket.ts:525-533 |
| `NI_CONNECTION_CLOSED` | `send`/`receive`/`assertNoQueuedFrames` on a non-open transport with no recorded terminal error | src/transport/ni-socket.ts:582-586, 601-605, 692-697 |
| `NI_CONNECTION_CLOSED` | receive timer scheduler throws (`"NI receive timer scheduler failed"`) | src/transport/ni-socket.ts:749-755 |
| `NI_CONNECTION_CLOSED` | peer `"end"` with a clean decoder (`"NI peer ended the connection"`) | src/transport/ni-socket.ts:855-860 |
| `NI_CONNECTION_CLOSED` | socket `"close"` while state is `"open"` (`"NI socket closed unexpectedly"`) | src/transport/ni-socket.ts:877-884 |
| `NI_CONNECTION_CLOSED` | local `close()` rejects a pending receive (`"NI socket was closed locally"`) | src/transport/ni-socket.ts:765-770 |
| `NI_CONNECTION_CLOSED` | `socket.resume()` throws when draining the queue (`"NI socket could not resume after queued frames were drained"`) | src/transport/ni-socket.ts:923-931 |
| `NI_PROTOCOL_ERROR` | `socket.isPaused()` throws in `adopt()` | src/transport/ni-socket.ts:453-460 |
| `NI_PROTOCOL_ERROR` | adopted socket not paused: `"an adopted NI socket must be paused before listener handoff"` | src/transport/ni-socket.ts:461-467 |
| `NI_PROTOCOL_ERROR` | decoder throws, or the queue-bound `RangeError` propagates, inside `#onData` (`"invalid NI stream"`) | src/transport/ni-socket.ts:824-828 |
| `NI_PROTOCOL_ERROR` | peer `"end"` mid-frame (`"NI peer ended a truncated frame"`) | src/transport/ni-socket.ts:861-869 |
| `NI_PROTOCOL_ERROR` | `assertNoQueuedFrames()` with `#frames.length > 0`: `"unexpected queued NI frame at a request boundary"` | src/transport/ni-socket.ts:588-594 |
| `NI_RECEIVE_TIMEOUT` | `` `NI receive timed out after ${timeoutMs} ms` `` — routed through `#fail`, so it is fatal | src/transport/ni-socket.ts:736-745 |
| `NI_WRITE_TIMEOUT` | `` `NI write timed out after ${this.#writeTimeoutMs} ms` `` | src/transport/ni-socket.ts:638-641 |
| `NI_WRITE_FAILED` | write callback yields an error (`"failed to write NI frame"`) | src/transport/ni-socket.ts:664-671 |
| `NI_WRITE_FAILED` | `socket.write` throws synchronously | src/transport/ni-socket.ts:673-679 |
| `NI_WRITE_FAILED` | write timer scheduler throws (`"NI write timer scheduler failed"`) | src/transport/ni-socket.ts:649-657 |

Non-`NiTransportError` throws from this file: `RangeError` from option validation
(src/transport/ni-socket.ts:128, 141, 149, 154-156, 175-177, 189-191, 194-196),
`TypeError` for a non-object adopt-options / non-Uint8Array initialData / bad scheduler / bad socket
(src/transport/ni-socket.ts:403, 425, 294, 409, 202, 215), and a **plain `Error`** for concurrent
receive: `throw new Error("only one NI receive may be pending on a connection");`
(src/transport/ni-socket.ts:713-715). That one does **not** fail the transport.

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Aggregate complete payload bytes retained before receive() drains them. */` | src/transport/ni-socket.ts:41 |
| `/** Complete NI frames retained before receive() drains them. */` | src/transport/ni-socket.ts:43 |
| `Connected, paused byte stream which can transfer exclusive ownership to an NI transport. The structural surface keeps routed sockets testable without exposing Node's much larger Socket API to the protocol layer.` | src/transport/ni-socket.ts:51-55 |
| `/** Ownership transfers to NiSocketTransport when adopt() is called. */` | src/transport/ni-socket.ts:81 |
| `/** Bytes coalesced after a preceding handshake and before socket.pause(). */` | src/transport/ni-socket.ts:83 |
| `Experimental timer seam for deterministic timeout/cancellation tests on the remove-before-1.0 low-level transport. Callbacks may run synchronously or later; clearTimeout should release any retained callback state.` | src/transport/ni-socket.ts:100-103 |
| `// Listener installation can run caller-provided stream code; avoid carrying an earlier narrowing across that ownership boundary.` | src/transport/ni-socket.ts:225-226 |
| `A single TCP connection carrying bounded NI length-prefixed records. Receive timeout/abort is deliberately fatal: a late RFC response must never be mistaken for the response to a later call on the same connection.` | src/transport/ni-socket.ts:238-241 |
| `Adopt one already-connected, paused stream after an outer route handshake. Listeners and coalesced bytes are installed before resume(), so target NI frames cannot be lost between protocol owners.` | src/transport/ni-socket.ts:392-395 |
| `Fail a synchronous request/response owner closed when a complete inbound frame is still buffered at a request boundary. Sending another request in that state would let the stale frame become the following response.` | src/transport/ni-socket.ts:573-577 |
| `// Timer cleanup must never prevent connection settlement.` | src/transport/ni-socket.ts:318 |
| `// Timer cleanup must never prevent settlement or socket destruction.` | src/transport/ni-socket.ts:944 |
| `/* ownership cleanup is best effort */` (repeated on every `adopt()` failure path) | src/transport/ni-socket.ts:440, 444, 454, 488 |
| `/* bounded close still settles */` | src/transport/ni-socket.ts:781 |
| `/* terminal state is already fixed */` | src/transport/ni-socket.ts:959 |

### Wire/behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| A frame split across TCP writes, and two frames coalesced into one write, both decode correctly; `transport.state` is `"open"` then `"closed"` after `close()` | `sends NI frames and receives fragmented and coalesced responses` | test/ni-socket.test.ts:129, 147-155 |
| `adopt()` delivers `initialData` frames **before** frames that arrive after resume; `localAddress`/`localPort`/`remoteAddress`/`remotePort` pass through to the adopted socket | `adopts a paused routed socket without losing coalesced or buffered frames` | test/ni-socket.test.ts:158, 184-197 |
| Overflowing `maxQueuedFrameCount: 1` during `adopt()` throws `NI_PROTOCOL_ERROR` **and destroys the socket** | `bounds queued complete frames and applies socket backpressure` | test/ni-socket.test.ts:204-218 |
| One queued frame ⇒ exactly `pauseCalls === 1`, `resumeCalls === 0`; after `receive()` drains it, `resumeCalls === 1` | `bounds queued complete frames and applies socket backpressure` | test/ni-socket.test.ts:220-228 |
| `receive()` with an already-aborted signal rejects `NI_ABORTED`, sets `state === "closed"`, and destroys the socket — the queued frame is **not** delivered | `an already-aborted receive retires queued data instead of delivering it` | test/ni-socket.test.ts:231-247 |
| `assertNoQueuedFrames()` with an unread frame throws `NI_PROTOCOL_ERROR` matching `/request boundary/u`, closes, destroys, and **writes nothing** (`assert.deepEqual(instrumented.writes, [])`) | `retires a synchronous request boundary with an unread complete frame` | test/ni-socket.test.ts:249-266 |
| A 3-frame batch under `maxQueuedFrameCount: 1` rejects the *pending* receive with `NI_PROTOCOL_ERROR` — no frame is delivered before the bound is checked | `rejects a decoded frame batch atomically before resolving a pending receive` | test/ni-socket.test.ts:268-295 |
| After `send()`, the buffer handed to `socket.write` is all zero bytes | `wipes the retained NI frame copy after a completed write` | test/ni-socket.test.ts:297-303 |
| A flowing (non-paused) socket, an aborted signal, malformed `initialData` (`"ffffffff"` hex), non-`Uint8Array` `initialData`, and an already-destroyed socket each throw and leave `socket.destroyed === true` | `fails closed when an adopted socket is flowing, aborted, or malformed` | test/ni-socket.test.ts:305-378 |
| A socket that synchronously emits `close`/`error` from `resume()` yields `NI_CONNECTION_CLOSED`; one that aborts the signal from `resume()` yields `NI_ABORTED`; both destroy | `rejects synchronous close, error, and abort emitted by resume` | test/ni-socket.test.ts:380-401 |
| After `NI_RECEIVE_TIMEOUT`, `state === "closed"` and **the next `receive()` rejects with the same `NI_RECEIVE_TIMEOUT`** (terminal error is retained and replayed) | `makes receive timeouts fatal so late replies cannot cross calls` | test/ni-socket.test.ts:403-419 |
| A second concurrent `receive()` rejects `/only one NI receive may be pending/`; aborting the first gives `NI_ABORTED` and `state === "closed"` | `aborts a pending receive and rejects concurrent receives` | test/ni-socket.test.ts:421-435 |
| A peer that `end()`s mid-frame produces `NI_PROTOCOL_ERROR` | `rejects a truncated peer stream as a protocol error` | test/ni-socket.test.ts:437-449 |
| 64 sequential connect/send/receive/close cycles leave `peer.closed === 64` and `peer.sockets.size === 0` | `resource harness returns every repeatedly opened NI socket` | test/ni-resource.test.ts:77-96 |
| 16 sockets timed out at `timeoutMs: 1` with **no explicit `close()`** still all reach the peer as closed | `resource harness destroys every timed-out NI socket` | test/ni-resource.test.ts:98-118 |
| Half-close, RST, graceful EOF ⇒ `NI_CONNECTION_CLOSED`; truncated frame and malformed length ⇒ `NI_PROTOCOL_ERROR`; `state === "closed"` in all five | `scripted peer deterministically injects ${fault.name}` | test/scripted-ni-network.test.ts:181-239 |
| A byte-at-a-time split, a short write, and two coalesced frames all decode; `state` stays `"open"` | `scripted peer observes requests and delivers delayed split, short-written, and coalesced frames` | test/scripted-ni-network.test.ts:109-156 |
| A duplicated raw frame is delivered twice — the transport does **not** deduplicate | `scripted peer can duplicate exact raw control bytes` | test/scripted-ni-network.test.ts:158-179 |

### Go mapping notes

- `NiSocketTransport` → a struct owning a `net.Conn` plus a reader goroutine. `#state` (`"open"|"closing"|"closed"`) plus `#terminalError` → a mutex-guarded `state` field and a sticky `err error`; `#fail` is "set err if nil, close everything once", i.e. `sync.Once` + stored error.
- `#pendingReceive` (at most one) → a single `chan frameResult` of capacity 1, or a `sync.Mutex` serialising `Receive`. The "only one receive may be pending" plain `Error` must stay a *distinct* error kind: it is the one error that does not kill the connection (src/transport/ni-socket.ts:713-715).
- `NiTimerScheduler` exists purely to make timeouts deterministic in tests (comment src/transport/ni-socket.ts:100-103). In Go this is a `Clock`/`context` seam, or `SetReadDeadline`. Note the contract that callbacks *may run synchronously* — Go timers never do, which removes the whole `if (settled)` re-check dance at src/transport/ni-socket.ts:362-365, 645-648, 746-747, 793-797.
- `signal?: AbortSignal` on `connect`/`adopt`/`send`/`receive` → `ctx context.Context` as the first parameter. Every `signalAborted()` re-read (src/transport/ni-socket.ts:224-228, comment: `avoid carrying an earlier narrowing across that ownership boundary`) becomes a `ctx.Err()` check at the same point.
- `receive({timeoutMs})` with `0` = no timeout → do **not** call `SetReadDeadline` at all when the caller passes zero.
- `Buffer` returned by `receive()` → `[]byte`. Note the returned slice was produced by the decoder and is *not* zeroed by the transport on the success path; only queued-but-undelivered frames are wiped.

---

## src/transport/saprouter-route.ts

Pure encode/decode plus route admission. No I/O.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `SAPROUTER_DEFAULT_SERVICE` | const | `export const SAPROUTER_DEFAULT_SERVICE = "3299";` | src/transport/saprouter-route.ts:4 |
| `SAPROUTER_ROUTE_INFORMATION_VERSION` | const | `export const SAPROUTER_ROUTE_INFORMATION_VERSION = 2;` | src/transport/saprouter-route.ts:6 |
| `SAPROUTER_DEFAULT_NI_VERSION` | const | `export const SAPROUTER_DEFAULT_NI_VERSION = 40;` | src/transport/saprouter-route.ts:8 |
| `SAPROUTER_ROUTE_HEADER_LENGTH` | const | `export const SAPROUTER_ROUTE_HEADER_LENGTH = 0x18;` | src/transport/saprouter-route.ts:10 |
| `SAPROUTER_MAX_ROUTE_BYTES` | const | `export const SAPROUTER_MAX_ROUTE_BYTES = 2_048;` | src/transport/saprouter-route.ts:11 |
| `SAPROUTER_MAX_ROUTE_HOPS` | const | `export const SAPROUTER_MAX_ROUTE_HOPS = 255;` | src/transport/saprouter-route.ts:12 |
| `SAPROUTER_MAX_RESPONSE_PAYLOAD_BYTES` | const | `export const SAPROUTER_MAX_RESPONSE_PAYLOAD_BYTES = 1_048_576;` | src/transport/saprouter-route.ts:13 |
| `SapRouterRouteHop` | interface | `host: string`, `service: string`, `usesDefaultService: boolean`, `passwordProtected: boolean` | src/transport/saprouter-route.ts:37-42 |
| `SapRouterFirstHop` | interface | `host: string`, `service: string`, `usesDefaultService: boolean` | src/transport/saprouter-route.ts:44-48 |
| `AdmittedSapRouterRoute` | interface | `hopCount: number`, `byteLength: number`, `firstHop: SapRouterFirstHop`, `hops: readonly SapRouterRouteHop[]`, `redactedRouteString: string` | src/transport/saprouter-route.ts:55-61 |
| `SapRouterRouteRequestOptions` | interface | `readonly niVersion?: number;` | src/transport/saprouter-route.ts:63-65 |
| `SapRouterRouteResponse` | type | `Readonly<{ readonly kind: "accepted" }> \| Readonly<{ readonly kind: "rejected"; readonly niVersion: number; readonly returnCode: number; readonly errorTextByteLength: number; }>` | src/transport/saprouter-route.ts:108-115 |
| `assertSapRouterRoutePrefix` | function | `export function assertSapRouterRoutePrefix(input: unknown): asserts input is string` | src/transport/saprouter-route.ts:71 |
| `completeSapRouterRoute` | function | `export function completeSapRouterRoute(prefix: unknown, gatewayHost: unknown, gatewayPort: unknown,): AdmittedSapRouterRoute` | src/transport/saprouter-route.ts:88-92 |
| `admitSapRouterRoute` | function | `export function admitSapRouterRoute(input: unknown): AdmittedSapRouterRoute` | src/transport/saprouter-route.ts:186 |
| `assertAdmittedSapRouterRoute` | function | `export function assertAdmittedSapRouterRoute(route: AdmittedSapRouterRoute,): void` | src/transport/saprouter-route.ts:288-290 |
| `encodeSapRouterRouteRequestPayload` | function | `export function encodeSapRouterRouteRequestPayload(route: AdmittedSapRouterRoute, options: SapRouterRouteRequestOptions = {},): Buffer` | src/transport/saprouter-route.ts:302-305 |
| `decodeSapRouterRouteResponse` | function | `export function decodeSapRouterRouteResponse(input: Uint8Array,): SapRouterRouteResponse` | src/transport/saprouter-route.ts:346-348 |

### Timeouts, bounds, and defaults

| Name | Value (verbatim) | What it bounds | Citation |
|---|---|---|---|
| `SAPROUTER_MAX_ROUTE_BYTES` | `2_048` | both the input route string length and the encoded internal-route byte length | src/transport/saprouter-route.ts:11, 189, 234 |
| `SAPROUTER_MAX_ROUTE_HOPS` | `255` | `if (hops.length > SAPROUTER_MAX_ROUTE_HOPS) invalidRoute();` | src/transport/saprouter-route.ts:12, 224 |
| `SAPROUTER_MAX_RESPONSE_PAYLOAD_BYTES` | `1_048_576` | `decodeSapRouterRouteResponse` input length | src/transport/saprouter-route.ts:13, 351 |
| host field | `value.length < 2 \|\| value.length > 255 \|\| !/^[A-Za-z0-9_.:%\[\]-]+$/u.test(value)` | route host | src/transport/saprouter-route.ts:128-131 |
| service field | `value.length > 63 \|\| !/^[A-Za-z0-9_.-]+$/u.test(value)` | route service | src/transport/saprouter-route.ts:137-139 |
| password field | `value.length > 255 \|\| !/^[\x20-\x2e\x30-\x7e]+$/u.test(value)` | route password (0x2f `/` excluded — `// The omitted 0x2f byte is '/': it is the route-field delimiter.`) | src/transport/saprouter-route.ts:142-147 |
| whole route charset | `!/^[\x20-\x7e]+$/u.test(input)` | printable ASCII only | src/transport/saprouter-route.ts:74, 191 |
| minimum hops | `if (hops.length < 2 \|\| hops[hops.length - 1]?.password !== undefined) { invalidRoute(); }` | at least one router + one target; **no password on the final hop** | src/transport/saprouter-route.ts:227-229 |
| `niVersion` | `(value as number) < 1 \|\| (value as number) > 255` → `"niVersion must be an integer in 1..255"` | encoded byte 10 | src/transport/saprouter-route.ts:294-298 |
| route-prefix sentinel | `ROUTE_PREFIX_SENTINEL_HOST = "open-rfc-target.invalid"`, `ROUTE_PREFIX_SENTINEL_PORT = 65_535` | used to validate a `/H/`-terminated prefix through the full parser | src/transport/saprouter-route.ts:20-21, 82-84 |

### Error codes

This file has no string error codes. It throws:

| Thrown value | Trigger condition | Citation |
|---|---|---|
| `RangeError("SAProuter route string is invalid")` | every route-syntax, charset, bound, hop-count, ordering, and duplicate-token failure (`invalidRoute()`) | src/transport/saprouter-route.ts:117-119 |
| `` RangeError(`SAProuter route response is invalid: ${message}`) `` with messages `"payload bounds"`, `"unexpected acknowledgement"`, `"truncated error header"`, `"invalid error status"`, `"inconsistent error text length"`, `"invalid error trailer"` | response decode failures | src/transport/saprouter-route.ts:336-338, 353, 360, 362, 369, 374, 380 |
| `RangeError("niVersion must be an integer in 1..255")` | out-of-range `niVersion` | src/transport/saprouter-route.ts:295-297 |
| `TypeError("route must be created by admitSapRouterRoute")` | route object not in the `ROUTE_INTERNALS` `WeakMap` | src/transport/saprouter-route.ts:277-285 |
| `Error("internal SAProuter route length mismatch")` | encoder self-check; **zeroes the payload first** (`payload.fill(0);`) | src/transport/saprouter-route.ts:329-332 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** SAProuter's documented default listener service. */` | src/transport/saprouter-route.ts:3 |
| `/** Route-information structure version used by current SAProuter. */` | src/transport/saprouter-route.ts:5 |
| `/** NI protocol version used by current SAProuter route requests. */` | src/transport/saprouter-route.ts:7 |
| `/** Fixed bytes preceding the NUL-separated internal route string. */` | src/transport/saprouter-route.ts:9 |
| `/** Undefined is serialized as an empty field and means service 3299. */` | src/transport/saprouter-route.ts:26 |
| `Immutable, redaction-safe route admitted at the network trust boundary. Password values are kept in an opaque side table and never exposed through object inspection or JSON serialization.` | src/transport/saprouter-route.ts:50-54 |
| `Validate the RFC `SAPROUTER` parameter form. It is a route prefix whose terminal `/H/` placeholder is completed from ASHOST/GWHOST by the transport.` | src/transport/saprouter-route.ts:67-70 |
| `// Reuse the authoritative complete-route parser with a fixed harmless endpoint. This admits passwords only on router hops and rejects /P/.` | src/transport/saprouter-route.ts:80-81 |
| `/** Bind a validated route prefix to exactly one normalized gateway endpoint. */` | src/transport/saprouter-route.ts:87 |
| `The canonical uppercase form is intentional. Legacy `/P/` placement is ambiguous and must be normalized by a caller before crossing this boundary. At least one router and one final target are required; a password on the final target has no successor to protect and is therefore rejected.` | src/transport/saprouter-route.ts:180-184 |
| `/** Encode one NI_ROUTE payload (without the outer four-byte NI length). */` | src/transport/saprouter-route.ts:301 |
| `payload[12] = 0; // NI_MSG_IO: the routed CPIC stream remains NI-framed.` | src/transport/saprouter-route.ts:315 |
| `/** Decode only the route-completion acknowledgement or bounded error status. */` | src/transport/saprouter-route.ts:345 |

### Wire/behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| Header layout: `payload[0..9] === "NI_ROUTE\0"`, `payload[9] === 2`, `payload[10] === SAPROUTER_DEFAULT_NI_VERSION`, `payload[11] === 2` (hop count), `payload[12] === 0`, `payload.readUInt16BE(13) === 0`, `payload[15] === 1` (hops−1), `payload.readUInt32BE(16) === internalRoute.length`, `payload.readUInt32BE(20) === byteLength("router\0"+"3299\0"+"secret\0")` | `encodes the documented NI_ROUTE header and NUL-separated route fields` | test/saprouter-route.test.ts:140-168 |
| Internal route body is `"router\0" + "3299\0" + "secret\0" + "target\0" + "sapgw01\0" + "\0"` — three NUL-terminated fields per hop, absent service/password serialised as empty | same | test/saprouter-route.test.ts:145-153, 168 |
| `decodeSapRouterRouteResponse(Buffer.from("NI_PONG\0","ascii"))` ⇒ `{ kind: "accepted" }`; `"NI_PONG"` without the NUL and `"NI_PONG\0extra"` are both rejected | `decodes exact NI_PONG and bounded NI_RTERR route responses` / `rejects malformed route acknowledgements and error bounds` | test/saprouter-route.test.ts:183-186, 216-217 |
| Both the 20+text form and the 24+text ("modern", 4 zero trailer bytes) `NI_RTERR` form decode identically | `decodes exact NI_PONG and bounded NI_RTERR route responses` | test/saprouter-route.test.ts:189-211 |
| Rejected: `/h/` lowercase, single-hop, duplicate `/S/`, duplicate `/W/`, `/W/` before `/S/`, legacy `/P/`, unknown `/X/`, password on the final target, host with a space, trailing newline, 256-byte host, 2030-byte host, 256 hops. Error message **never contains the input** | `rejects malformed, ambiguous, legacy, and oversized route strings` | test/saprouter-route.test.ts:104-133 |
| `/W/` password never appears in `inspect()`, `JSON.stringify()`, or `redactedRouteString` (which renders `/W/[REDACTED]`) | `admits a canonical multi-hop route without exposing passwords` | test/saprouter-route.test.ts:91-98 |
| A spread copy `{ ...route }` is rejected by the encoder (`/admitSapRouterRoute/u`) — identity, not shape, is the credential | `encodes the documented NI_ROUTE header and NUL-separated route fields` | test/saprouter-route.test.ts:176-179 |

### Go mapping notes

- `AdmittedSapRouterRoute` + `ROUTE_INTERNALS` `WeakMap` (src/transport/saprouter-route.ts:35, 270-273) is a capability pattern: the public object carries no passwords, and only an object minted by `admitSapRouterRoute` can be encoded. In Go: an unexported field on an exported struct, or an unexported struct returned behind an exported interface. There is no `WeakMap` needed — the unexported field *is* the side table.
- `toJSON` / `util.inspect.custom` redaction (src/transport/saprouter-route.ts:257-268) → implement `String()`/`Format()` and `MarshalJSON()` on the route type so `%v` and `json.Marshal` cannot leak the password.
- Encoding is all fixed-offset big-endian into a `[]byte` of `0x18 + byteLength`; `encoding/binary.BigEndian` maps 1:1.

---

## src/transport/saprouter-tunnel.ts

First-hop TCP connect + one `NI_ROUTE` exchange, handing back a **paused** stream.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `SapRouterTransportErrorCode` | type | union of `"SAPROUTER_ABORTED" \| "SAPROUTER_CONNECT_FAILED" \| "SAPROUTER_CONNECT_TIMEOUT" \| "SAPROUTER_CONNECTION_CLOSED" \| "SAPROUTER_HANDSHAKE_TIMEOUT" \| "SAPROUTER_PROTOCOL_ERROR" \| "SAPROUTER_ROUTE_DENIED" \| "SAPROUTER_ROUTE_REJECTED" \| "SAPROUTER_UNSUPPORTED_SERVICE" \| "SAPROUTER_WRITE_FAILED"` | src/transport/saprouter-tunnel.ts:21-31 |
| `SapRouterTransportError` | class | `constructor(code: SapRouterTransportErrorCode, message: string, options: { readonly routerReturnCode?: number; readonly cause?: unknown; } = {},)` | src/transport/saprouter-tunnel.ts:34, 39-46 |
| `SapRouterRouteSocket` | interface | `destroyed?`, `write(chunk, callback)`, `pause()`, `destroy()`, `on("data"\|"error"\|"end"\|"close", …)`, `removeListener(…)` — **no `resume()`, no `end()`, no `isPaused()`** | src/transport/saprouter-tunnel.ts:82-96 |
| `SapRouterTimerHandle` | type | `export type SapRouterTimerHandle = unknown;` | src/transport/saprouter-tunnel.ts:98 |
| `SapRouterTimerScheduler` | interface | `setTimeout(callback, delayMs): SapRouterTimerHandle; clearTimeout(handle): void;` | src/transport/saprouter-tunnel.ts:100-103 |
| `SapRouterHandshakeOptions` | interface | `timeoutMs?`, `maxResponsePayloadBytes?`, `niVersion?` | src/transport/saprouter-tunnel.ts:114-118 |
| `SapRouterConnectOptions` | interface | `connectTimeoutMs?`, `handshakeTimeoutMs?`, `maxResponsePayloadBytes?`, `niVersion?`, `family?: 4 \| 6`, `noDelay?: boolean` | src/transport/saprouter-tunnel.ts:120-127 |
| `SapRouterFirstHopConnectOptions` | interface | `timeoutMs: number`, `family: 4 \| 6 \| undefined`, `noDelay: boolean`, `signal: AbortSignal \| undefined`, `scheduler: SapRouterTimerScheduler` | src/transport/saprouter-tunnel.ts:141-147 |
| `SapRouterFirstHopConnector` | type | `export type SapRouterFirstHopConnector = (endpoint: SapRouterFirstHop, options: SapRouterFirstHopConnectOptions,) => Promise<SapRouterRouteSocket>;` | src/transport/saprouter-tunnel.ts:149-152 |
| `SapRouterConnectionDependencies` | interface | `connect?: SapRouterFirstHopConnector`, `scheduler?: SapRouterTimerScheduler` | src/transport/saprouter-tunnel.ts:154-157 |
| `EstablishedSapRouterRoute` | interface | `socket: SapRouterRouteSocket`, `initialData: Buffer`, `hopCount: number`, `firstHop: SapRouterFirstHop` | src/transport/saprouter-tunnel.ts:159-166 |
| `establishSapRouterRoute` | function | `export function establishSapRouterRoute(socket: SapRouterRouteSocket, route: AdmittedSapRouterRoute, options?: SapRouterHandshakeOptions, signal?: AbortSignal, scheduler: SapRouterTimerScheduler = SYSTEM_TIMER_SCHEDULER,): Promise<EstablishedSapRouterRoute>` | src/transport/saprouter-tunnel.ts:296-302 |
| `connectSapRouterRoute` | function | `export async function connectSapRouterRoute(routeInput: string \| AdmittedSapRouterRoute, options?: SapRouterConnectOptions, signal?: AbortSignal, dependencies: SapRouterConnectionDependencies = {},): Promise<EstablishedSapRouterRoute>` | src/transport/saprouter-tunnel.ts:672-677 |

### Timeouts, bounds, and defaults

| Name | Value (verbatim) | What it bounds | Citation |
|---|---|---|---|
| `DEFAULT_TIMEOUT_MS` | `10_000` | both `timeoutMs` (handshake) and `connectTimeoutMs` when unset | src/transport/saprouter-tunnel.ts:16, 192, 230 |
| `MAX_TIMEOUT_MS` | `300_000` | upper bound for both timeouts | src/transport/saprouter-tunnel.ts:17, 195, 233 |
| handshake `timeoutMs` range | `boundedInteger(…, "timeoutMs", 1, MAX_TIMEOUT_MS)` — **minimum 1, 0 is rejected** | NI_ROUTE negotiation | src/transport/saprouter-tunnel.ts:191-196 |
| `connectTimeoutMs` range | `boundedInteger(…, "connectTimeoutMs", 1, MAX_TIMEOUT_MS)` | first-hop TCP connect | src/transport/saprouter-tunnel.ts:229-234 |
| `maxResponsePayloadBytes` | default `SAPROUTER_MAX_RESPONSE_PAYLOAD_BYTES`, range `8`..`SAPROUTER_MAX_RESPONSE_PAYLOAD_BYTES` (`1_048_576`) | declared NI length of the route response | src/transport/saprouter-tunnel.ts:197-202 |
| `niVersion` | default `SAPROUTER_DEFAULT_NI_VERSION` (`40`), range `1..255` | encoded byte | src/transport/saprouter-tunnel.ts:203-208 |
| `noDelay` | `noDelay: options?.noDelay ?? true` | Nagle on the first hop | src/transport/saprouter-tunnel.ts:236 |
| declared-length floor | `if (declaredLength < 8 \|\| declaredLength > normalized.maxResponsePayloadBytes)` → protocol error | 4-byte NI length prefix of the response | src/transport/saprouter-tunnel.ts:417-426 |
| `"saprouter"` service | `if (service === "saprouter") return 3_299;` | first-hop port when the service is the literal name | src/transport/saprouter-tunnel.ts:546 |
| numeric service | `if (!/^[0-9]{1,5}$/u.test(service))` → `SAPROUTER_UNSUPPORTED_SERVICE`; then `if (port < 1 \|\| port > 65_535)` → `SAPROUTER_UNSUPPORTED_SERVICE` | first-hop port | src/transport/saprouter-tunnel.ts:547-560 |
| route-denied sentinel | `const denied = response.returnCode === -94;` | `SAPROUTER_ROUTE_DENIED` vs `SAPROUTER_ROUTE_REJECTED` | src/transport/saprouter-tunnel.ts:462 |

### Error codes

| Code | Trigger condition | Citation |
|---|---|---|
| `SAPROUTER_ABORTED` | already-aborted signal at `establishSapRouterRoute` entry — **destroys the socket, then rejects** | src/transport/saprouter-tunnel.ts:307-310 |
| `SAPROUTER_ABORTED` | signal fires during the handshake (`onAbort` → `fail(aborted())`) | src/transport/saprouter-tunnel.ts:374-376 |
| `SAPROUTER_ABORTED` | already-aborted at `connectSapRouterRoute` entry; after the connector throws; after the connector returns (destroys the socket) | src/transport/saprouter-tunnel.ts:691, 709, 716-719 |
| `SAPROUTER_CONNECT_FAILED` | `createConnection`/`setNoDelay` throws | src/transport/saprouter-tunnel.ts:583-589 |
| `SAPROUTER_CONNECT_FAILED` | first-hop socket `"error"` before connect completes | src/transport/saprouter-tunnel.ts:617-623 |
| `SAPROUTER_CONNECT_FAILED` | first-hop socket `"close"` before negotiation | src/transport/saprouter-tunnel.ts:624-629 |
| `SAPROUTER_CONNECT_FAILED` | connect-timer scheduler throws | src/transport/saprouter-tunnel.ts:653-658 |
| `SAPROUTER_CONNECT_FAILED` | bracketed first-hop host that is not a valid IPv6 literal (`"the SAProuter first-hop IP literal is invalid"`) | src/transport/saprouter-tunnel.ts:571-576 |
| `SAPROUTER_CONNECT_FAILED` | injected connector throws a non-`SapRouterTransportError`, or returns a non-socket (`"SAProuter first-hop connector failed"`) | src/transport/saprouter-tunnel.ts:706-715 |
| `SAPROUTER_CONNECT_TIMEOUT` | `` `SAProuter first-hop connection timed out after ${options.timeoutMs} ms` `` | src/transport/saprouter-tunnel.ts:642-645 |
| `SAPROUTER_CONNECTION_CLOSED` | stream `"error"` during handshake (`"…failed before route handoff"`) | src/transport/saprouter-tunnel.ts:378-384 |
| `SAPROUTER_CONNECTION_CLOSED` | stream `"end"` with **no** partial response received | src/transport/saprouter-tunnel.ts:386-393 |
| `SAPROUTER_CONNECTION_CLOSED` | stream `"close"` — **`fail(…, false)`: does NOT destroy** | src/transport/saprouter-tunnel.ts:396-401 |
| `SAPROUTER_CONNECTION_CLOSED` | `socket.on(...)` throws while installing listeners | src/transport/saprouter-tunnel.ts:481-487 |
| `SAPROUTER_HANDSHAKE_TIMEOUT` | `` `SAProuter route negotiation timed out after ${normalized.timeoutMs} ms` `` | src/transport/saprouter-tunnel.ts:495-501 |
| `SAPROUTER_HANDSHAKE_TIMEOUT` | handshake timer scheduler throws (`"SAProuter route timer scheduler failed"`) | src/transport/saprouter-tunnel.ts:508-514 |
| `SAPROUTER_PROTOCOL_ERROR` | chunk is not a `Buffer` (`"SAProuter stream must provide unencoded byte buffers"`) | src/transport/saprouter-tunnel.ts:405-408 |
| `SAPROUTER_PROTOCOL_ERROR` | declared length `< 8` or `>` bound | src/transport/saprouter-tunnel.ts:417-426 |
| `SAPROUTER_PROTOCOL_ERROR` | `"SAProuter route response decoder is inconsistent"` | src/transport/saprouter-tunnel.ts:430-434 |
| `SAPROUTER_PROTOCOL_ERROR` | `socket.pause()` throws (`"SAProuter stream could not be paused for handoff"`) | src/transport/saprouter-tunnel.ts:443-448 |
| `SAPROUTER_PROTOCOL_ERROR` | `decodeSapRouterRouteResponse` throws; the thrown `cause.message` is forwarded verbatim | src/transport/saprouter-tunnel.ts:450-460 |
| `SAPROUTER_PROTOCOL_ERROR` | `"SAProuter ended a truncated route response"` (EOF with partial bytes) | src/transport/saprouter-tunnel.ts:386-389 |
| `SAPROUTER_ROUTE_DENIED` | `returnCode === -94`; message `"SAProuter denied the requested route"`; `routerReturnCode` carried | src/transport/saprouter-tunnel.ts:461-469 |
| `SAPROUTER_ROUTE_REJECTED` | any other negative return code; `` `SAProuter rejected route setup with return code ${response.returnCode}` `` | src/transport/saprouter-tunnel.ts:461-469 |
| `SAPROUTER_UNSUPPORTED_SERVICE` | non-numeric, non-`"saprouter"` first-hop service, or numeric out of 1..65535 | src/transport/saprouter-tunnel.ts:547-560 |
| `SAPROUTER_WRITE_FAILED` | write callback error or synchronous `write` throw (`"SAProuter NI_ROUTE request write failed"`) | src/transport/saprouter-tunnel.ts:523-541 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Redaction-safe failure from first-hop connection or NI_ROUTE negotiation. */` | src/transport/saprouter-tunnel.ts:33 |
| `/** Minimal unencoded stream surface shared by net.Socket and compatible TLS streams. */` | src/transport/saprouter-tunnel.ts:81 |
| `/** Paused routed stream. Attach the NI/CPIC owner before resuming it. */` | src/transport/saprouter-tunnel.ts:160 |
| `/** Raw bytes received after the NI_PONG frame in the same data event. */` | src/transport/saprouter-tunnel.ts:162 |
| `// AbortSignal changes asynchronously even though its property is readonly; keeping the read behind a function prevents stale control-flow narrowing.` | src/transport/saprouter-tunnel.ts:271-273 |
| `Send exactly one NI_ROUTE request on an already-connected first-hop stream. The returned stream is paused so an NI/CPIC owner can attach without losing coalesced target bytes. Every failure is terminal and destroys the stream.` | src/transport/saprouter-tunnel.ts:291-295 |
| `/* the rejection remains authoritative */` | src/transport/saprouter-tunnel.ts:346 |
| `/** Validate, connect the first hop, negotiate NI_ROUTE, and return the stream. */` | src/transport/saprouter-tunnel.ts:671 |
| `// Admission and all bounded scalar validation happen before a connector is selected or invoked. Invalid caller input therefore performs no I/O.` | src/transport/saprouter-tunnel.ts:678-679 |
| `/* connector failure remains authoritative */` | src/transport/saprouter-tunnel.ts:707 |

### Wire/behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| Exactly one write happens, it is an NI frame whose payload starts `"NI_ROUTE\0"`; the written buffer is zeroed after the write completes (`socket.writtenBuffers[0]!.every((byte) => byte === 0)`) | `writes NI_ROUTE once and preserves fragmented and coalesced handoff bytes` | test/saprouter-tunnel.test.ts:114, 125-132 |
| The response may arrive in arbitrary fragments (2 / 7 / rest); bytes after the NI_PONG frame become `initialData` exactly; `pauseCalls === 1`, `destroyCalls === 0`, timer cleared once | same | test/saprouter-tunnel.test.ts:134-149 |
| `Socket extends SapRouterRouteSocket` compiles — Node's `net.Socket` structurally satisfies the interface | (type-level assertion, not a named test) | test/saprouter-tunnel.test.ts:16-17 |
| `returnCode === -94` ⇒ `SAPROUTER_ROUTE_DENIED` with `routerReturnCode === -94`; the route password never appears in `message` or `JSON.stringify(error)`; socket destroyed once | `maps a denied route to a redaction-safe typed error` | test/saprouter-tunnel.test.ts:152-166 |
| An already-aborted signal ⇒ **`socket.writes.length === 0`** — nothing is sent | `already aborted sends nothing` | test/saprouter-tunnel.test.ts:202-218 |
| A declared length of 33 against `maxResponsePayloadBytes: 32` ⇒ `SAPROUTER_PROTOCOL_ERROR`; a declared length of 7 (below the floor of 8) likewise | `oversized declared response` / `undersized response` | test/saprouter-tunnel.test.ts:239-252, 419-428 |
| A string chunk (`"encoded text must not be accepted"`) ⇒ `SAPROUTER_PROTOCOL_ERROR` | `non-buffer input` | test/saprouter-tunnel.test.ts:254-261 |
| Route validation runs **before** the connector: `assert.equal(calls, 0)` for `/H/router.example.test/W/secret` | `validates a route fully before invoking the first-hop connector` | test/saprouter-tunnel.test.ts:264-280 |
| The connector receives only `{host, service, usesDefaultService}` — no password reaches it | same | test/saprouter-tunnel.test.ts:288-296 |
| `"close"` during handshake ⇒ `SAPROUTER_CONNECTION_CLOSED` with `assert.equal(socket.destroyCalls, 0)` — close does **not** re-destroy | `close without destroy replay` | test/saprouter-tunnel.test.ts:409-417 |
| Two bytes then `"end"` ⇒ `SAPROUTER_PROTOCOL_ERROR` (truncated), not `CONNECTION_CLOSED` | `truncated EOF` | test/saprouter-tunnel.test.ts:399-407 |
| A scheduler whose `setTimeout` fires synchronously still yields `SAPROUTER_HANDSHAKE_TIMEOUT`, clears the handle once, and writes nothing | `synchronous timer expiry` | test/saprouter-tunnel.test.ts:503-525 |
| Invalid handshake options (`timeoutMs: 0`, `1.5`, `300_001`; `maxResponsePayloadBytes: 7`, `1_048_577`; `niVersion: 0`, `256`) all reject with `socket.writes.length === 0` | `rejects malformed handshake options, schedulers, and socket seams before writing` | test/saprouter-tunnel.test.ts:340-368 |
| A connector returning an object with only `destroy()` ⇒ `SAPROUTER_CONNECT_FAILED` and `invalidDestroyCalls === 1` | `rejects malformed connect boundaries and post-connect aborts before NI_ROUTE` | test/saprouter-tunnel.test.ts:577-588 |
| Aborting *inside* the connector ⇒ `SAPROUTER_ABORTED` and the returned socket is destroyed once | same | test/saprouter-tunnel.test.ts:604-617 |
| A real end-to-end first hop returns `hopCount === 2` and `result.socket.destroyed === false` | `connects the numeric first hop and hands the routed socket back paused` | test/saprouter-tunnel.test.ts:313-338 |

### Go mapping notes

- The `#responseHeader` (4 bytes) + `#responsePayload` incremental reassembly (src/transport/saprouter-tunnel.ts:312-316, 410-441) becomes a plain blocking read: `io.ReadFull(conn, hdr[:4])`, bounds-check, `io.ReadFull(conn, payload)`. The whole event-driven cursor machinery disappears.
- **The leftover-bytes problem does not disappear.** `succeed(Buffer.from(chunk.subarray(cursor)))` (src/transport/saprouter-tunnel.ts:472) captures target bytes that arrived in the same TCP segment. In Go this is whatever a `bufio.Reader` has buffered past the response — return the `*bufio.Reader` (or its `Buffered()` bytes) alongside the `net.Conn`, never the bare `net.Conn`.
- `socket.pause()` before handoff (src/transport/saprouter-tunnel.ts:444) has **no Go equivalent and needs none** — nothing reads from a `net.Conn` until someone calls `Read`. Drop the pause/`isPaused` contract entirely; replace it with "the connector returns `(net.Conn, []byte, error)` and the caller owns the conn from that instant".
- `fail(error, shouldDestroy = false)` for the `close` event (src/transport/saprouter-tunnel.ts:349-359, 396-401) is the only path that does not close the socket. In Go, `defer conn.Close()` on the error path is safe everywhere (double `Close` is allowed), so this distinction can be collapsed.
- `SapRouterTimerScheduler` → `context.WithTimeout` + `SetDeadline`.

---

## src/transport/saprouter-ni.ts

Thin adapter: SAProuter route → `NiSocketTransport`.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `SapRouterRouteConnector` | type | `export type SapRouterRouteConnector = (route: AdmittedSapRouterRoute, options: SapRouterConnectOptions, signal?: AbortSignal,) => Promise<EstablishedSapRouterRoute>;` | src/transport/saprouter-ni.ts:19-23 |
| `SapRouterNiDependencies` | interface | `readonly connectRoute?: SapRouterRouteConnector;` | src/transport/saprouter-ni.ts:25-27 |
| `createSapRouterDirectCpicTransportFactory` | function | `export function createSapRouterDirectCpicTransportFactory(routePrefix: string, dependencies: SapRouterNiDependencies = {},): (options: NiSocketConnectOptions, signal?: AbortSignal,) => Promise<NiSocketTransport>` | src/transport/saprouter-ni.ts:42-48 |

### Timeouts, bounds, and defaults

| Name | Value (verbatim) | What it bounds | Citation |
|---|---|---|---|
| `DEFAULT_ROUTE_TIMEOUT_MS` | `const DEFAULT_ROUTE_TIMEOUT_MS = 10_000;` (module-private) | fallback when the caller gives no `connectTimeoutMs` | src/transport/saprouter-ni.ts:17, 59 |
| timeout fan-out | `connectTimeoutMs: timeoutMs, handshakeTimeoutMs: timeoutMs,` | **one** caller timeout drives **both** phases | src/transport/saprouter-ni.ts:71-72 |
| `noDelay` | `noDelay: options.noDelay ?? true` | first hop | src/transport/saprouter-ni.ts:74 |

### Error codes

No error codes of its own. It throws `TypeError`s and propagates:

| Thrown value | Trigger condition | Citation |
|---|---|---|
| `RangeError("SAProuter route string is invalid")` | `assertSapRouterRoutePrefix(routePrefix)` at factory-construction time | src/transport/saprouter-ni.ts:49 |
| `TypeError("SAProuter NI dependencies must be an object")` | non-object `dependencies` | src/transport/saprouter-ni.ts:50-52 |
| `TypeError("SAProuter route connector must be a function")` | non-function `connectRoute` | src/transport/saprouter-ni.ts:54-56 |
| `TypeError("SAProuter route connector must return a route")` | connector returns a non-object | src/transport/saprouter-ni.ts:78-80 |
| `TypeError("SAProuter route connector must return buffered initialData")` | `!Buffer.isBuffer(established.initialData)` | src/transport/saprouter-ni.ts:82-86 |
| anything from `NiSocketTransport.adopt` | handoff failure | src/transport/saprouter-ni.ts:88-96 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `// NiSocketTransport.adopt performs the authoritative structural validation and destroys invalid streams after ownership transfer.` | src/transport/saprouter-ni.ts:30-31 |
| `Create one NI transport factory for an admitted SAProuter route. Every invocation negotiates a fresh route for exactly one target connection; no routed stream is shared or replayed. The compatibility name is retained because direct CPIC was the first consumer, but Message Server NI uses the same framing and ownership boundary.` | src/transport/saprouter-ni.ts:35-41 |
| `// The handoff failure remains authoritative.` | src/transport/saprouter-ni.ts:104 |

### Wire/behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| The prefix `/H/router…/S/3299/W/router-secret/H/` plus `host: "gateway.fixture.invalid", port: 3_342` completes to exactly two hops with the gateway service rendered as the decimal string `"3342"` | `adopts each established SAProuter stream with its coalesced NI bytes` | test/saprouter-ni.test.ts:83-107 |
| The connector receives a frozen options object exactly `{ connectTimeoutMs: 1_234, handshakeTimeoutMs: 1_234, family: 4, noDelay: false }` and the caller's `AbortSignal` unchanged | same | test/saprouter-ni.test.ts:109-116 |
| After a successful adopt, the caller's `initialData` buffer is **fully zeroed** (`initialData.equals(Buffer.alloc(initialData.length))`) | same | test/saprouter-ni.test.ts:121 |
| On handoff failure (flowing socket) the route socket is destroyed, `initialData` is zeroed, and `calls === 1` — **no retry** | `destroys a flowing route handoff and never retries it` | test/saprouter-ni.test.ts:126-156 |
| An empty prefix, a non-function connector, and a prefix not ending in `/H/` all throw at factory-construction time | `validates SAProuter NI dependencies before creating a transport` | test/saprouter-ni.test.ts:158-176 |

### Go mapping notes

- The factory is a closure over a validated route prefix; in Go, a struct with an unexported `route` field and a `Dial(ctx, host, port) (*NiTransport, error)` method.
- `Reflect.apply(connectRoute, undefined, [...])` (src/transport/saprouter-ni.ts:68) exists to defeat `this`-binding tricks from injected dependencies; in Go a func value has no receiver to hijack — drop it.
- `initialData.fill(0)` on **both** the success and the failure path (src/transport/saprouter-ni.ts:97, 100) must be preserved: the leftover bytes can contain the first target frame. Go: `defer func() { for i := range initialData { initialData[i] = 0 } }()` or `crypto/subtle`-free explicit wipe.

---

## src/transport/connectivity-socks5-tunnel.ts

SAP Connectivity SOCKS5 with the method-`0x80` JWT extension.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `ConnectivitySocks5ErrorCode` | type | union of `"CONNECTIVITY_SOCKS5_ABORTED" \| "…_AUTHENTICATION_REJECTED" \| "…_CONNECTION_CLOSED" \| "…_CONNECT_FAILED" \| "…_CONNECT_REJECTED" \| "…_CONNECT_TIMEOUT" \| "…_PROTOCOL_ERROR" \| "…_TIMEOUT" \| "…_WRITE_FAILED"` | src/transport/connectivity-socks5-tunnel.ts:36-45 |
| `ConnectivitySocks5Error` | class | `constructor(code: ConnectivitySocks5ErrorCode, message: string, replyCode?: number,)` with `readonly code`, `readonly replyCode: number \| undefined` | src/transport/connectivity-socks5-tunnel.ts:48, 52-56 |
| `ConnectivitySocks5ConfigInput` | interface | `proxyHost`, `proxyPort`, `targetHost`, `targetPort`, `accessToken`, `locationId?`, `timeoutMs?`, `maxBufferedBytes?` | src/transport/connectivity-socks5-tunnel.ts:82-95 |
| `AdmittedConnectivitySocks5Config` | interface | same fields, all required except `locationId: string \| undefined` | src/transport/connectivity-socks5-tunnel.ts:97-106 |
| `ConnectivitySocks5Socket` | interface | `destroyed?`, `write`, `pause`, `destroy`, `on`, `removeListener` (identical shape to `SapRouterRouteSocket`) | src/transport/connectivity-socks5-tunnel.ts:109-123 |
| `ConnectivitySocks5TimerHandle` | type | `export type ConnectivitySocks5TimerHandle = unknown;` | src/transport/connectivity-socks5-tunnel.ts:125 |
| `ConnectivitySocks5TimerScheduler` | interface | `setTimeout(callback, delayMs)`, `clearTimeout(handle)` | src/transport/connectivity-socks5-tunnel.ts:127-133 |
| `EstablishedConnectivitySocks5Tunnel` | interface | `readonly socket: ConnectivitySocks5Socket;` `readonly initialData: Buffer;` | src/transport/connectivity-socks5-tunnel.ts:144-149 |
| `ConnectivitySocks5ProxyConnectOptions` | interface | `proxyHost`, `proxyPort`, `timeoutMs`, `signal`, `scheduler` | src/transport/connectivity-socks5-tunnel.ts:151-157 |
| `ConnectivitySocks5ProxyConnector` | type | `(options: ConnectivitySocks5ProxyConnectOptions) => Promise<ConnectivitySocks5Socket>` | src/transport/connectivity-socks5-tunnel.ts:159-161 |
| `ConnectivitySocks5ConnectionDependencies` | interface | `connect?`, `scheduler?` | src/transport/connectivity-socks5-tunnel.ts:163-166 |
| `admitConnectivitySocks5Config` | function | `export function admitConnectivitySocks5Config(input: ConnectivitySocks5ConfigInput \| Readonly<Record<string, unknown>>,): AdmittedConnectivitySocks5Config` | src/transport/connectivity-socks5-tunnel.ts:346-348 |
| `assertAdmittedConnectivitySocks5Config` | function | `export function assertAdmittedConnectivitySocks5Config(input: unknown,): asserts input is AdmittedConnectivitySocks5Config` | src/transport/connectivity-socks5-tunnel.ts:387-389 |
| `establishConnectivitySocks5Tunnel` | function | `export function establishConnectivitySocks5Tunnel(socket: ConnectivitySocks5Socket, config: AdmittedConnectivitySocks5Config, signal?: AbortSignal, scheduler: ConnectivitySocks5TimerScheduler = SYSTEM_TIMER_SCHEDULER,): Promise<EstablishedConnectivitySocks5Tunnel>` | src/transport/connectivity-socks5-tunnel.ts:503-508 |
| `connectConnectivitySocks5Tunnel` | function | `export async function connectConnectivitySocks5Tunnel(input: ConnectivitySocks5ConfigInput \| AdmittedConnectivitySocks5Config, signal?: AbortSignal, dependencies: ConnectivitySocks5ConnectionDependencies = {},): Promise<EstablishedConnectivitySocks5Tunnel>` | src/transport/connectivity-socks5-tunnel.ts:879-883 |

### Timeouts, bounds, and defaults

| Name | Value (verbatim) | What it bounds | Citation |
|---|---|---|---|
| `DEFAULT_TIMEOUT_MS` | `10_000` | proxy connect **and** handshake (same `config.timeoutMs`) | src/transport/connectivity-socks5-tunnel.ts:16, 358, 767, 863 |
| `DEFAULT_MAX_BUFFERED_BYTES` | `16_384` | handshake response bytes retained | src/transport/connectivity-socks5-tunnel.ts:17, 364 |
| `MAX_TIMEOUT_MS` | `300_000` | `timeoutMs` upper bound (`1..300_000`) | src/transport/connectivity-socks5-tunnel.ts:18, 357-362 |
| `MAX_BUFFERED_BYTES` | `1_048_576` | `maxBufferedBytes` upper bound (`8..1_048_576`) | src/transport/connectivity-socks5-tunnel.ts:19, 363-368 |
| `MAX_ACCESS_TOKEN_BYTES` | `65_536` | `accessToken` ASCII byte length | src/transport/connectivity-socks5-tunnel.ts:20, 297-308 |
| SOCKS constants | `SOCKS_VERSION = 0x05`, `JWT_AUTHENTICATION_METHOD = 0x80`, `JWT_AUTHENTICATION_VERSION = 0x01`, `CONNECT_COMMAND = 0x01`, `DOMAIN_ADDRESS = 0x03`, `IPV4_ADDRESS = 0x01`, `IPV6_ADDRESS = 0x04` | wire bytes | src/transport/connectivity-socks5-tunnel.ts:8-14 |
| `ALLOWED_CONFIG_PROPERTIES` | `"proxyHost", "proxyPort", "targetHost", "targetPort", "accessToken", "locationId", "timeoutMs", "maxBufferedBytes"` — anything else is a `TypeError` | config surface | src/transport/connectivity-socks5-tunnel.ts:24-33, 190-193 |
| `accessToken` charset | `!/^[\x21-\x7e]+$/u.test(value)` — visible ASCII, no spaces | token | src/transport/connectivity-socks5-tunnel.ts:301 |
| `locationId` bounds | `value.length > 189` **and** `Buffer.byteLength(value, "utf8") > 189` both rejected; control characters rejected by the class `\u0000-\u001f` plus `\u007f` | location header field | src/transport/connectivity-socks5-tunnel.ts:311-328 |
| host name bounds | `normalized.length > 253`, `!/^[A-Za-z0-9.-]+$/u`, `normalized.includes("..")`; per-label `length > 63` and `/^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$/u` | proxyHost/targetHost | src/transport/connectivity-socks5-tunnel.ts:250-266 |
| IPv6 target ban | `"targetHost cannot be IPv6 because the SAP Connectivity SOCKS5 endpoint documents only IPv4 and DOMAIN targets"` | `targetHost` only (`ipv6Allowed` is `true` for `proxyHost`, `false` for `targetHost`) | src/transport/connectivity-socks5-tunnel.ts:236-240, 351-354 |
| response buffer bound | `if (chunk.length > config.maxBufferedBytes - buffered.length)` → protocol error | inbound handshake buffer | src/transport/connectivity-socks5-tunnel.ts:709-714 |
| CONNECT reply lengths | IPv4 `succeed(10)`; DOMAIN `const responseLength = 7 + domainLength;`; IPv6 `succeed(22)` | reply framing | src/transport/connectivity-socks5-tunnel.ts:668-692 |
| proxy `noDelay` | `socket.setNoDelay(true);` — **hardcoded**, not configurable; `createConnection` is called with **no `family`** | proxy TCP socket | src/transport/connectivity-socks5-tunnel.ts:796-800 |

### Error codes

| Code | Trigger condition | Citation |
|---|---|---|
| `CONNECTIVITY_SOCKS5_ABORTED` | already-aborted at `establishConnectivitySocks5Tunnel` entry — **destroys the socket first** | src/transport/connectivity-socks5-tunnel.ts:512-518 |
| `CONNECTIVITY_SOCKS5_ABORTED` | signal fires during the handshake | src/transport/connectivity-socks5-tunnel.ts:739-744 |
| `CONNECTIVITY_SOCKS5_ABORTED` | already-aborted in `systemConnectProxy`, or the connect-phase abort listener fires | src/transport/connectivity-socks5-tunnel.ts:788-793, 845-850 |
| `CONNECTIVITY_SOCKS5_ABORTED` | already-aborted at `connectConnectivitySocks5Tunnel` entry; after the connector throws with an aborted signal; after the connector returns with an aborted signal (destroys the socket) | src/transport/connectivity-socks5-tunnel.ts:901-906, 920-925, 931-937 |
| `CONNECTIVITY_SOCKS5_AUTHENTICATION_REJECTED` | method-selection byte 1 `!== 0x80` (`"…did not accept JWT authentication"`) | src/transport/connectivity-socks5-tunnel.ts:615-620 |
| `CONNECTIVITY_SOCKS5_AUTHENTICATION_REJECTED` | auth-response byte 1 `!== 0x00` (`"…rejected authentication"`) | src/transport/connectivity-socks5-tunnel.ts:637-642 |
| `CONNECTIVITY_SOCKS5_CONNECTION_CLOSED` | socket `"error"` / `"end"` / `"close"` during setup | src/transport/connectivity-socks5-tunnel.ts:721-738 |
| `CONNECTIVITY_SOCKS5_CONNECT_FAILED` | `createConnection` throws; proxy `"error"`; proxy `"close"` before negotiation; connect-timer scheduler throws; injected connector fails or returns a non-socket | src/transport/connectivity-socks5-tunnel.ts:801-805, 833-843, 869-873, 926-929 |
| `CONNECTIVITY_SOCKS5_CONNECT_REJECTED` | CONNECT reply byte 1 `!== 0x00`; **carries `replyCode`** | src/transport/connectivity-socks5-tunnel.ts:658-665 |
| `CONNECTIVITY_SOCKS5_CONNECT_TIMEOUT` | `"Connectivity SOCKS5 proxy connection timed out"` | src/transport/connectivity-socks5-tunnel.ts:860-863 |
| `CONNECTIVITY_SOCKS5_PROTOCOL_ERROR` | method-response byte 0 `!== 0x05`; auth-response byte 0 `!== 0x01`; CONNECT reply byte 0 `!== 0x05` **or** byte 2 `!== 0x00`; `domainLength === 0`; unknown address type; non-`Buffer` chunk; `maxBufferedBytes` exceeded; `socket.pause()` throws; listener install throws; timer scheduler throws | src/transport/connectivity-socks5-tunnel.ts:608-613, 630-635, 651-657, 676-682, 693-697, 702-708, 709-715, 558-566, 752-757, 774-779 |
| `CONNECTIVITY_SOCKS5_TIMEOUT` | handshake timer expiry (`"Connectivity SOCKS5 handshake timed out"`) | src/transport/connectivity-socks5-tunnel.ts:764-767 |
| `CONNECTIVITY_SOCKS5_WRITE_FAILED` | write callback error or synchronous `write` throw | src/transport/connectivity-socks5-tunnel.ts:583-597 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** A bounded failure that never includes tokens, locations, or endpoint names. */` | src/transport/connectivity-socks5-tunnel.ts:47 |
| `/** Host and SOCKS5 port from the Connectivity binding (normally 20004). */` | src/transport/connectivity-socks5-tunnel.ts:83 |
| `/** Cloud Connector TCP virtual host and virtual port; never resolved locally. */` | src/transport/connectivity-socks5-tunnel.ts:86 |
| `/** Raw Connectivity service JWT. Do not include a `Bearer ` prefix. */` | src/transport/connectivity-socks5-tunnel.ts:89 |
| `/** Unencoded Cloud Connector location ID. */` | src/transport/connectivity-socks5-tunnel.ts:91 |
| `/** Minimal connected byte stream shared by net.Socket and compatible streams. */` | src/transport/connectivity-socks5-tunnel.ts:108 |
| `/** Paused stream. Attach the target protocol consumer before resuming it. */` | src/transport/connectivity-socks5-tunnel.ts:145 |
| `/** Bytes received after the CONNECT response in the same data event. */` | src/transport/connectivity-socks5-tunnel.ts:147 |
| `// AbortSignal is mutable across async and caller-controlled boundaries.` | src/transport/connectivity-socks5-tunnel.ts:415 |
| `/** Validate and snapshot every byte-affecting field before network I/O. */` | src/transport/connectivity-socks5-tunnel.ts:345 |
| `// Admission already constrains a normalized domain to 1..253 ASCII bytes.` | src/transport/connectivity-socks5-tunnel.ts:484 |
| `Negotiate SAP's documented method 0x80 JWT SOCKS5 extension on an already connected stream. This function is for the binding's SOCKS5/TCP endpoint, not the distinct RFC/LDAP proxy endpoint.` | src/transport/connectivity-socks5-tunnel.ts:498-502 |
| `/** Validate, connect to the explicit SOCKS5 endpoint, and negotiate the tunnel. */` | src/transport/connectivity-socks5-tunnel.ts:878 |
| `/* failure remains authoritative */` | src/transport/connectivity-socks5-tunnel.ts:550 |

### Wire/behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| First write is exactly `[0x05, 0x01, 0x80]` and is sent before any response arrives | `performs SAP JWT method 0x80 and preserves fragmented target bytes` | test/connectivity-socks5-tunnel.test.ts:224, 237 |
| Auth frame layout: `[0]=0x01`, `writeUInt32BE(token.length, 1)`, token at 5, `[5+len]=locationLength`, base64(locationId) at `6+len` — the location ID is **base64 of the UTF-8 bytes** | same | test/connectivity-socks5-tunnel.test.ts:242-250 |
| The written auth buffer is zeroed after the write completes (`socket.writtenBuffers[1]!.every((byte) => byte === 0)`) | same | test/connectivity-socks5-tunnel.test.ts:251 |
| DOMAIN CONNECT request is `[0x05,0x01,0x00,0x03,len] + host + uint16be(port)` | same | test/connectivity-socks5-tunnel.test.ts:255-259 |
| Response may be split byte-wise across `[0x05,0x00,0x00]`, `[0x03,0x05,0x70]`, remainder; trailing bytes become `initialData` verbatim; `pauseCalls === 1`, `destroyCalls === 0` | same | test/connectivity-socks5-tunnel.test.ts:261-277 |
| An IPv4 `targetHost` is encoded as `[0x05,0x01,0x00,0x01,192,0,2,25,0x01,0xbb]` with **no DNS resolution**; omitted `locationId` yields a trailing `0` length byte | `encodes an IPv4 target without resolving it and omits location ID` | test/connectivity-socks5-tunnel.test.ts:281-302 |
| An IPv6 bound address (ATYP `0x04`, 16+2 bytes) in the reply is accepted and the remainder becomes `initialData` | `accepts an RFC 1928 IPv6 bound address in a successful reply` | test/connectivity-socks5-tunnel.test.ts:304-325 |
| `Socket extends ConnectivitySocks5Socket` compiles | (type-level assertion) | test/connectivity-socks5-tunnel.test.ts:23-26 |
| Admitted config is a snapshot: mutating the source afterwards does not change it; `Object.isFrozen(admitted)`; neither `JSON.stringify` nor `inspect` leaks token or location | `admits an immutable, redaction-safe SOCKS5 configuration snapshot` | test/connectivity-socks5-tunnel.test.ts:127-145 |
| `proxyHost: "[2001:db8::1]"` normalizes to `"2001:db8::1"`; `targetHost: "sap-virtual.example.test."` normalizes by stripping the trailing dot | `rejects ambiguous input, unsupported addresses, and unbounded secrets before I/O` | test/connectivity-socks5-tunnel.test.ts:189-195 |
| A spread copy of an admitted config is rejected (`/must come from admitConnectivitySocks5Config/u`); getters are never invoked (`getterCalls === 0`); a `Proxy` is rejected without running its traps | same | test/connectivity-socks5-tunnel.test.ts:196-222 |
| Reply-code mapping: method `0xff` ⇒ `AUTHENTICATION_REJECTED`; auth status `0x01` ⇒ `AUTHENTICATION_REJECTED`; CONNECT reply `0x02` ⇒ `CONNECT_REJECTED` with `replyCode === 2`; reserved byte `0x01` ⇒ `PROTOCOL_ERROR`; ATYP `0x7f` ⇒ `PROTOCOL_ERROR`; version `0x04` ⇒ `PROTOCOL_ERROR`; auth version `0x02` ⇒ `PROTOCOL_ERROR`; domain length `0x00` ⇒ `PROTOCOL_ERROR`. Every message/JSON/inspect is checked against `/header\|payload\|signature\|sap-virtual/u` | `maps proxy rejections and malformed responses to redaction-safe errors` | test/connectivity-socks5-tunnel.test.ts:500-573 |
| Already-aborted ⇒ `socket.writes.length === 0` **and** `socket.destroyCalls === 1` | `already aborted writes nothing` | test/connectivity-socks5-tunnel.test.ts:723-739 |
| A 9-byte chunk against `maxBufferedBytes: 8` ⇒ `PROTOCOL_ERROR` | `buffer bound` | test/connectivity-socks5-tunnel.test.ts:801-814 |
| A write that fails **after** the token frame was handed to the socket still zeroes that frame | `write failure releases token frame` | test/connectivity-socks5-tunnel.test.ts:816-831 |
| A `removeListener` that throws during cleanup still settles the promise and destroys the socket once | `throwing listener cleanup still settles and destroys` | test/connectivity-socks5-tunnel.test.ts:690-719 |
| A partially-installed listener set is fully unwound: `socket.listenerCount("data") === 0` and `destroyCalls === 1` | `partial listener installation throws` | test/connectivity-socks5-tunnel.test.ts:671-688 |
| End to end against a real TCP proxy, the trailing `[0x00,0x00,0x00,0x02,0x4e,0x49]` written in the same segment as the CONNECT reply is returned as `initialData` | `connects a real TCP stream and completes the documented handshake` | test/connectivity-socks5-tunnel.test.ts:327-409 |

### Go mapping notes

- The `phase` state machine (`"method-response" \| "authentication-response" \| "connect-response"`, src/transport/connectivity-socks5-tunnel.ts:493-496, 604-699) driven by a re-entrant `process()` over a growing `buffered` becomes three sequential blocking reads on a `*bufio.Reader`. The `if (buffered.length < N) return;` short-reads vanish.
- `buffered` is reallocated and the old copy zeroed on **every** append and every `consume` (src/transport/connectivity-socks5-tunnel.ts:599-603, 716-719). With a `bufio.Reader` you cannot zero the internal buffer; if the token-in-memory property matters, read into a caller-owned `[]byte` and wipe it explicitly.
- `pendingWrites: Set<Buffer>` (src/transport/connectivity-socks5-tunnel.ts:525) tracks frames handed to an async `write` so `fail()` can zero them even if the callback never runs. Go's `conn.Write` is synchronous — wipe with `defer` immediately after `Write` returns.
- `admitConnectivitySocks5Config` + `ADMITTED_CONFIGS` `WeakSet` (src/transport/connectivity-socks5-tunnel.ts:34, 383) is the same capability pattern as the SAProuter route; the exotic-object defences (`isProxy`, `Reflect.ownKeys`, own-data-property checks, prototype check — src/transport/connectivity-socks5-tunnel.ts:168-202) have **no Go analogue** and should be dropped, not translated: a Go struct value is already a snapshot.

---

## src/transport/connectivity-socks5-ni.ts

Thin adapter: SOCKS5 tunnel → `NiSocketTransport`. Structurally identical to `saprouter-ni.ts`.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `ConnectivitySocks5DirectCpicProxyInput` | interface | `proxyHost: string`, `proxyPort: number`, `accessToken: string`, `locationId?: string`, `timeoutMs?: number`, `maxBufferedBytes?: number` — **no target fields** | src/transport/connectivity-socks5-ni.ts:23-33 |
| `ConnectivitySocks5TunnelConnector` | type | `export type ConnectivitySocks5TunnelConnector = (config: AdmittedConnectivitySocks5Config, signal?: AbortSignal,) => Promise<EstablishedConnectivitySocks5Tunnel>;` | src/transport/connectivity-socks5-ni.ts:35-38 |
| `ConnectivitySocks5NiDependencies` | interface | `readonly connectTunnel?: ConnectivitySocks5TunnelConnector;` | src/transport/connectivity-socks5-ni.ts:40-42 |
| `createConnectivitySocks5DirectCpicTransportFactory` | function | `export function createConnectivitySocks5DirectCpicTransportFactory(proxyInput: ConnectivitySocks5DirectCpicProxyInput, dependencies: ConnectivitySocks5NiDependencies = {},): DirectCpicTransportFactory` | src/transport/connectivity-socks5-ni.ts:118-121 |

### Timeouts, bounds, and defaults

| Name | Value (verbatim) | What it bounds | Citation |
|---|---|---|---|
| `DEFAULT_TIMEOUT_MS` | `const DEFAULT_TIMEOUT_MS = 10_000;` (module-private) | last-resort tunnel timeout | src/transport/connectivity-socks5-ni.ts:13, 139 |
| timeout precedence | `timeoutMs: proxy.timeoutMs ?? options.connectTimeoutMs ?? DEFAULT_TIMEOUT_MS,` | **a configured proxy `timeoutMs` overrides the caller's connect timeout** | src/transport/connectivity-socks5-ni.ts:139 |
| `ALLOWED_PROXY_PROPERTIES` | `"proxyHost", "proxyPort", "accessToken", "locationId", "timeoutMs", "maxBufferedBytes"` | proxy option surface | src/transport/connectivity-socks5-ni.ts:14-21, 69-73 |
| validation placeholder | `targetHost: "validation.invalid", targetPort: 1,` | used only to run the full admission at factory-construction time | src/transport/connectivity-socks5-ni.ts:88-89 |

### Error codes

No codes of its own. Throws `TypeError`s at construction (src/transport/connectivity-socks5-ni.ts:55-59, 61-63, 67-68, 70-72, 75-77, 123-129) and, per call:
`TypeError("Connectivity SOCKS5 tunnel connector must return a tunnel")`
(src/transport/connectivity-socks5-ni.ts:149-153) and
`TypeError("Connectivity SOCKS5 tunnel connector must return buffered initialData")`
(src/transport/connectivity-socks5-ni.ts:155-159); everything else propagates from
`admitConnectivitySocks5Config` or `NiSocketTransport.adopt`.

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Connectivity binding host and `onpremise_socks5_proxy_port`. */` | src/transport/connectivity-socks5-ni.ts:24 |
| `/** Raw Connectivity access token, without `Bearer `. */` | src/transport/connectivity-socks5-ni.ts:27 |
| `/** Optional fixed per-phase timeout; otherwise the CPIC connect timeout wins. */` | src/transport/connectivity-socks5-ni.ts:30 |
| `Route classic CPIC/NI through an explicitly configured Connectivity TCP mapping. This intentionally does not consume the distinct RFC proxy port.` | src/transport/connectivity-socks5-ni.ts:114-117 |
| `/* handoff failure wins */` | src/transport/connectivity-socks5-ni.ts:174 |

### Wire/behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| Per-call config carries `targetHost`/`targetPort` from the *call* options and proxy fields from the *factory* snapshot; `timeoutMs === 4_321` (the proxy override) and `maxBufferedBytes === 8_192` | `adopts an explicitly configured Connectivity SOCKS5 TCP tunnel for NI` | test/connectivity-socks5-ni.test.ts:50, 82-90 |
| `initialData` is fully zeroed after a successful adopt | same | test/connectivity-socks5-ni.test.ts:95 |
| With no proxy `timeoutMs`, the call's `connectTimeoutMs: 2_345` is used verbatim | `uses the direct connection timeout when no proxy override is configured` | test/connectivity-socks5-ni.test.ts:99-120 |
| Factory construction rejects a bad `proxyHost`, a non-function connector, non-plain objects, a `Proxy`, unknown properties, symbol keys, accessor properties, and null dependencies | `validates fixed proxy options before constructing the NI factory` | test/connectivity-socks5-ni.test.ts:122-195 |
| A non-object tunnel result rejects `/must return a tunnel/u`; a non-Buffer `initialData` rejects `/buffered initialData/u` **and destroys the socket once**; a flowing socket rejects `/paused/u`, destroys, and zeroes `initialData` | `destroys invalid or unadoptable tunnel handoffs without retrying` | test/connectivity-socks5-ni.test.ts:197-258 |

### Go mapping notes

- Identical to `saprouter-ni.ts`: a struct closing over a validated proxy snapshot with a `Dial(ctx, host, port)` method. The precedence rule at src/transport/connectivity-socks5-ni.ts:139 is the only non-obvious behaviour and must be preserved exactly.
- `snapshotProxy` distinguishes "user set `timeoutMs`" from "defaulted" by `values.has("timeoutMs") ? admitted.timeoutMs : undefined` (src/transport/connectivity-socks5-ni.ts:105). In Go use `*int`/`*time.Duration` or an explicit `timeoutSet bool` — a zero value cannot encode this.

---

## src/transport/message-server-resolver.ts

Not a route. Resolves an RFC logon group to a concrete application server over one throwaway NI connection.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `MessageServerResolutionErrorCode` | type | `export type MessageServerResolutionErrorCode = \| "MS_SERVICE_AMBIGUOUS" \| "MS_SERVICE_NOT_FOUND" \| "MS_SERVICE_TABLE_INVALID";` | src/transport/message-server-resolver.ts:25-28 |
| `MessageServerResolutionError` | class | `constructor(code: MessageServerResolutionErrorCode, message: string, cause?: unknown,)` | src/transport/message-server-resolver.ts:30, 34-38 |
| `MessageServerTransport` | interface | `send(payload: Uint8Array, signal?: AbortSignal): Promise<void>;` `receive(options?: NiReceiveOptions): Promise<Buffer>;` `close(): Promise<void>;` | src/transport/message-server-resolver.ts:46-50 |
| `MessageServerTransportFactory` | type | `(options: NiSocketConnectOptions, signal?: AbortSignal) => Promise<MessageServerTransport>` | src/transport/message-server-resolver.ts:52-55 |
| `MessageServerServicePortResolver` | type | `(service: string, signal?: AbortSignal) => Promise<number>` | src/transport/message-server-resolver.ts:57-60 |
| `MessageServerRfcGroupResolverOptions` | interface | `messageServerHost: string`, `messageServerService?: string \| number`, `systemId: string`, `group: string`, `connectTimeoutMs?`, `operationTimeoutMs?`, `signal?`, `servicePortResolver?`, `transportFactory?` | src/transport/message-server-resolver.ts:62-74 |
| `parseTcpServicePort` | function | `export function parseTcpServicePort(servicesText: string, service: string,): number \| undefined` | src/transport/message-server-resolver.ts:140-143 |
| `defaultMessageServerServicePortResolver` | function | `export async function defaultMessageServerServicePortResolver(service: string, signal?: AbortSignal,): Promise<number>` | src/transport/message-server-resolver.ts:196-199 |
| `resolveMessageServerRfcGroup` | function | `export async function resolveMessageServerRfcGroup(options: MessageServerRfcGroupResolverOptions,): Promise<MessageServerRfcGroupTarget>` | src/transport/message-server-resolver.ts:259-261 |

### Timeouts, bounds, and defaults

| Name | Value (verbatim) | What it bounds | Citation |
|---|---|---|---|
| `DEFAULT_TIMEOUT_MS` | `10_000` | both `connectTimeoutMs` and `operationTimeoutMs` | src/transport/message-server-resolver.ts:19, 270, 274 |
| `MAX_TIMEOUT_MS` | `0x7fff_ffff` | both timeouts; range is `1..MAX_TIMEOUT_MS` — **0 is rejected** | src/transport/message-server-resolver.ts:20, 76-85 |
| `MAX_SERVICES_FILE_BYTES` | `1024 * 1024` | `/etc/services` text | src/transport/message-server-resolver.ts:21, 148 |
| `MAX_SERVICES_FILE_LINES` | `100_000` | line count | src/transport/message-server-resolver.ts:22, 162 |
| `MAX_SERVICES_LINE_BYTES` | `4_096` | per-line bytes | src/transport/message-server-resolver.ts:23, 171 |
| `messageServerHost` | `/^[\x21-\x7e]{1,255}$/u` | host | src/transport/message-server-resolver.ts:87-97 |
| `systemId` | `/^[A-Za-z0-9]{3}$/u` | R3NAME/SYSID | src/transport/message-server-resolver.ts:99-106 |
| service name | `/^[\x21-\x7e]{1,64}$/u` and `!value.includes("/")` | `/etc/services` key | src/transport/message-server-resolver.ts:108-115 |
| default service | `const selected = service ?? `sapms${systemId}`;` | derived when MSSERV is omitted | src/transport/message-server-resolver.ts:232 |
| transport options | `{ host, port, connectTimeoutMs, maxPayloadLength: MAX_MESSAGE_SERVER_PAYLOAD_LENGTH, writeTimeoutMs: operationTimeoutMs, noDelay: true }` — **no `maxQueuedFrameCount`/`closeTimeoutMs` is passed** | the NI socket the resolver opens | src/transport/message-server-resolver.ts:300-307 |
| `MAX_MESSAGE_SERVER_PAYLOAD_LENGTH` | imported from `../protocol/message-server.js`; **(test)** `assert.equal(options.maxPayloadLength, 512);` | NI payload cap on this connection | src/transport/message-server-resolver.ts:9; test/message-server-resolver.test.ts:268 |

### Error codes

| Code | Trigger condition | Citation |
|---|---|---|
| `MS_SERVICE_TABLE_INVALID` | services text over `MAX_SERVICES_FILE_BYTES` | src/transport/message-server-resolver.ts:148-153 |
| `MS_SERVICE_TABLE_INVALID` | `servicesText.includes("\0")` (`"services table contains a NUL byte"`) | src/transport/message-server-resolver.ts:154-159 |
| `MS_SERVICE_TABLE_INVALID` | more than `MAX_SERVICES_FILE_LINES` lines | src/transport/message-server-resolver.ts:162-167 |
| `MS_SERVICE_TABLE_INVALID` | any line over `MAX_SERVICES_LINE_BYTES` | src/transport/message-server-resolver.ts:171-176 |
| `MS_SERVICE_TABLE_INVALID` | `readFile("/etc/services")` fails (`"failed to read /etc/services for the message-server service"`) | src/transport/message-server-resolver.ts:203-212 |
| `MS_SERVICE_AMBIGUOUS` | two different TCP ports for the same name: `` `TCP service ${name} maps to conflicting ports ${selected} and ${port}` `` | src/transport/message-server-resolver.ts:185-190 |
| `MS_SERVICE_NOT_FOUND` | `` `TCP service ${name} is not defined in /etc/services; provide a numeric msserv` `` | src/transport/message-server-resolver.ts:214-219 |
| `NI_ABORTED` (`NiTransportError`) | signal aborted at any `throwIfAborted` point or after a resolver/read failure | src/transport/message-server-resolver.ts:124-134, 206, 242, 286, 293 |
| `Error("message-server resolver completed without a selected server")` | `finally` ran clean but `resolved === undefined` | src/transport/message-server-resolver.ts:342-344 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Decimal port or an /etc/services TCP name such as sapmsTST. */` | src/transport/message-server-resolver.ts:65 |
| `/** R3NAME/SYSID; used to derive sapms<SID> when MSSERV is omitted. */` | src/transport/message-server-resolver.ts:67 |
| `Parse one bounded services table without consulting DNS or opening sockets. Conflicting declarations are rejected rather than depending on file order.` | src/transport/message-server-resolver.ts:136-139 |
| `// Validate the wire-bound field before service lookup or network I/O.` | src/transport/message-server-resolver.ts:267 |
| `Resolve one RFC logon group on one fresh Message Server connection.` / `The exchange is intentionally one-shot: it never retries, fails over, or replays after a write. The returned target is resolved before any direct RFC owner or business-call session is created by a provider.` | src/transport/message-server-resolver.ts:252-257 |

### Wire/behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| Exactly three frames are observed by the peer, in order: login, RFC-group, logout — even when the responses arrive split `[1,2,3,108]` and short-written | `resolves through one fragmented scripted NI exchange and then closes` | test/message-server-resolver.test.ts:68, 72-79, 101-105 |
| Peer EOF after login ⇒ `NI_CONNECTION_CLOSED`; a malformed NI declared length ⇒ `NI_PROTOCOL_ERROR`; in both cases `peer.observedFrames` is exactly `[login]` — **no logout, no replay** | `treats EOF and an oversized NI declaration as terminal without replay` | test/message-server-resolver.test.ts:108-149 |
| A receive timeout propagates the original error object, `factoryCalls === 1`, `closeCount === 1`, and only the login frame was sent — **no reconnect** | `propagates receive timeout once, closes once, and does not reconnect` | test/message-server-resolver.test.ts:151-176 |
| Aborting during the second receive still calls `close()` exactly once and never re-opens | `aborts a pending second receive, closes, and never sends or opens again` | test/message-server-resolver.test.ts:178-216 |
| Every field is validated before either the service resolver or the transport factory runs: `serviceCalls === 0 && factoryCalls === 0` for 7 invalid inputs | `validates every route field before service lookup or transport creation` | test/message-server-resolver.test.ts:218-245 |
| Omitted `messageServerService` resolves the name `"sapmsTST"`; an explicit name is passed through verbatim | `uses r3name/sysid for the default service and accepts an explicit msserv` | test/message-server-resolver.test.ts:247-274 |
| A pre-aborted signal rejects `NI_ABORTED` with `serviceCalls === 0 && factoryCalls === 0` | `rejects a pre-aborted lookup before resolving a service or opening a socket` | test/message-server-resolver.test.ts:276-301 |
| `/etc/services` parsing: aliases resolve (`"message-tst"` ⇒ 3600), `udp` entries are ignored, comments are stripped, missing names return `undefined` | `parses bounded /etc/services TCP records, aliases, and comments` | test/message-server-resolver.test.ts:303-314 |
| Conflicting ports ⇒ `MS_SERVICE_AMBIGUOUS`; a 4100-byte line and an embedded NUL ⇒ `MS_SERVICE_TABLE_INVALID` | `rejects conflicting and unbounded service-table data` | test/message-server-resolver.test.ts:316-338 |

### Go mapping notes

- The `try/finally` (src/transport/message-server-resolver.ts:325-340) does logout-then-close, records the **first** cleanup failure (`cleanupFailure ??= closeError`), and rethrows it *after* the main path — so a cleanup failure can mask a success. That ordering is observable and must be reproduced; Go: named return `err`, `defer func(){ ... }()` that only assigns when it has a first failure to report.
- The logout `send` is issued **without** the caller's signal: `await transport.send(encodeMessageServerLogoutRequest());` (src/transport/message-server-resolver.ts:329). Deliberate — it must still run after an abort. In Go use `context.WithoutCancel(ctx)` or a fresh short-deadline context, not the caller's.
- `readFile("/etc/services", { signal })` (src/transport/message-server-resolver.ts:204) → `os.ReadFile` with a size check; Go has no cancellable file read, so check `ctx.Err()` before and after.

---

## Socket lifecycle and ownership

This is the section that decides the port. Everything below is about one question: *at each
instant, who may call `pause`/`resume`/`destroy`/`write` on the byte stream, and what happens to
the bytes already in flight when ownership moves.*

### 1. The four states of a stream in this layer

| Phase | Who owns it | Read mode | Citation |
|---|---|---|---|
| Being connected (direct) | `NiSocketTransport.connect`'s promise executor | not yet readable | src/transport/ni-socket.ts:304-376 |
| Being connected (routed) | `systemConnectFirstHop` / `systemConnectProxy` promise executor | flowing but with no `"data"` listener | src/transport/saprouter-tunnel.ts:592-660, connectivity-socks5-tunnel.ts:808-875 |
| Handshaking | `establishSapRouterRoute` / `establishConnectivitySocks5Tunnel` | flowing; the handshake owns `"data"` | src/transport/saprouter-tunnel.ts:476-480, connectivity-socks5-tunnel.ts:747-751 |
| Handed off | `NiSocketTransport` | **paused**, then resumed by `adopt()` | src/transport/ni-socket.ts:444, 559, 461-467, 524 |

### 2. Paused vs resumed — every call site

- `establishSapRouterRoute` calls `socket.pause()` **immediately after** the full route response
  has been reassembled and **before** decoding it: `try { socket.pause(); } catch { fail(protocolError("SAProuter stream could not be paused for handoff")); return; }`
  (src/transport/saprouter-tunnel.ts:443-448). Confirmed by test: `assert.equal(socket.pauseCalls, 1);`
  (test/saprouter-tunnel.test.ts:146).
- `establishConnectivitySocks5Tunnel` calls `socket.pause()` inside `succeed()`, **after** slicing
  `initialData` out of `buffered` and **before** resolving
  (src/transport/connectivity-socks5-tunnel.ts:555-571). Test: `assert.equal(socket.pauseCalls, 1);`
  (test/connectivity-socks5-tunnel.test.ts:275).
- `NiSocketTransport.adopt` **requires** the socket to already be paused and destroys it otherwise:
  `if (!paused) { socket.destroy(); throw new NiTransportError("NI_PROTOCOL_ERROR", "an adopted NI socket must be paused before listener handoff"); }`
  (src/transport/ni-socket.ts:461-467). Test: `fails closed when an adopted socket is flowing, aborted, or malformed`
  (test/ni-socket.test.ts:322-329).
- `adopt` resumes only at the very end, and only conditionally:
  `if (!transport.#pausedForQueue) socket.resume();` (src/transport/ni-socket.ts:524). If the
  coalesced `initialData` already filled the queue, the stream stays paused.
- Steady-state backpressure: `#pauseForQueue()` (src/transport/ni-socket.ts:912-916) is called from
  `#onData` whenever a decoded batch leaves at least one frame queued
  (`if (queuedFrameCount > 0) this.#pauseForQueue();`, src/transport/ni-socket.ts:814-816).
  `#resumeAfterQueue()` (src/transport/ni-socket.ts:918-932) is called from `receive()` exactly when
  the queue empties: `if (this.#frames.length === 0) this.#resumeAfterQueue();`
  (src/transport/ni-socket.ts:710). It is a no-op unless `#pausedForQueue && #state === "open"`
  (src/transport/ni-socket.ts:919).
- `NiSocketTransport.connect` **never** pauses. The constructor attaches `"data"`
  (src/transport/ni-socket.ts:274), which puts a Node socket into flowing mode.
  **INFERRED:** for the direct route the stream is flowing from the moment the transport exists.

### 3. Ownership transfer, precisely

The handoff object is `{ socket, initialData }` — a paused stream plus the bytes the previous owner
over-read (src/transport/saprouter-tunnel.ts:159-166, connectivity-socks5-tunnel.ts:144-149). The
comments make the contract explicit: `/** Paused routed stream. Attach the NI/CPIC owner before resuming it. */`
(src/transport/saprouter-tunnel.ts:160) and `/** Ownership transfers to NiSocketTransport when adopt() is called. */`
(src/transport/ni-socket.ts:81).

`adopt()` performs the transfer in this exact order (src/transport/ni-socket.ts:397-551):

1. Validate options; on any throw, `socket.destroy()` then rethrow (src/transport/ni-socket.ts:439-442).
2. `socketTerminal(socket)` check → destroy + `NI_CONNECTION_CLOSED` (src/transport/ni-socket.ts:443-449).
3. `socket.isPaused()`; a throw → destroy + `NI_PROTOCOL_ERROR` (src/transport/ni-socket.ts:450-460).
4. `!paused` → destroy + `NI_PROTOCOL_ERROR` (src/transport/ni-socket.ts:461-467).
5. `signalAborted(signal)` → destroy + `NI_ABORTED` (src/transport/ni-socket.ts:468-474).
6. Construct the transport — **this is where the four listeners are installed**
   (`"data"`, `"end"`, `"error"`, `"close"`; src/transport/ni-socket.ts:274-281). A throw here →
   destroy + `NI_CONNECTION_CLOSED` (src/transport/ni-socket.ts:487-494).
7. Re-check `socketTerminal` → `transport.#fail(error)` (src/transport/ni-socket.ts:495-502).
8. **Feed `initialData` through the normal `#onData` path** (src/transport/ni-socket.ts:503-505) —
   so coalesced target bytes are decoded, bounded, and queued by exactly the same code as live bytes.
9. Re-check `#state !== "open"` (src/transport/ni-socket.ts:506-514) and `signalAborted`
   (src/transport/ni-socket.ts:515-522).
10. `socket.resume()` unless already paused for queue (src/transport/ni-socket.ts:523-533).
11. Re-check abort and terminality **after** resume (src/transport/ni-socket.ts:534-549) — because
    `resume()` can synchronously emit `data`/`error`/`close`. Test: `rejects synchronous close, error, and abort emitted by resume`
    (test/ni-socket.test.ts:380-401).

The adapters then wipe the leftover buffer on **both** paths:
`initialData.fill(0); return transport;` / `catch { initialData?.fill(0); try { socket?.destroy(); } ... throw error; }`
(src/transport/saprouter-ni.ts:97-107; identical at connectivity-socks5-ni.ts:170-176).

### 4. What happens on abort

| Abort point | Effect | Citation |
|---|---|---|
| Before `connect()` | throw `NI_ABORTED`; no socket created | src/transport/ni-socket.ts:296-301 |
| During `connect()` | `settle(aborted(...))` → `socket.destroy(); reject(error);` | src/transport/ni-socket.ts:325-332, 343-344 |
| Before `adopt()` | `socket.destroy()` then throw | src/transport/ni-socket.ts:468-474 |
| During/after `adopt()` handoff | `transport.#fail(error)` — which destroys the socket and poisons the transport | src/transport/ni-socket.ts:515-522, 534-549 |
| Before/during `send()` | `this.#fail(error)` — **the whole transport dies, not just the write** | src/transport/ni-socket.ts:607-614, 631-632 |
| Before/during `receive()` | `this.#fail(error)` — fatal by design (comment src/transport/ni-socket.ts:238-241) | src/transport/ni-socket.ts:699-706, 720-723 |
| SAProuter handshake | `fail(aborted())` → removes listeners, destroys, wipes buffers, rejects | src/transport/saprouter-tunnel.ts:374-376, 349-359 |
| SOCKS5 handshake | `fail(...ABORTED)` → cleanup, wipe, destroy, reject | src/transport/connectivity-socks5-tunnel.ts:739-744, 543-552 |
| `connectSapRouterRoute` after the connector returns | `socket.destroy()` then throw `SAPROUTER_ABORTED` | src/transport/saprouter-tunnel.ts:716-719 |
| `connectConnectivitySocks5Tunnel` after the connector returns | `socket.destroy()` then throw `..._ABORTED` | src/transport/connectivity-socks5-tunnel.ts:931-937 |

Abort listeners are always registered `{ once: true }` and explicitly removed on settle
(src/transport/ni-socket.ts:325, 624, 732, 892; saprouter-tunnel.ts:327, 480; connectivity-socks5-tunnel.ts:536, 751).

### 5. Cleanup on each error path

`NiSocketTransport.#fail` is the single terminal funnel (src/transport/ni-socket.ts:952-960):

```
if (this.#state === "closed") return;      // idempotent
this.#terminalError ??= error;             // first error wins and is retained
this.#state = "closed";
this.#rejectPending(this.#terminalError);  // also clears timer + abort listener
this.#clearQueuedFrames();                 // zeroes every queued frame
this.#decoder.reset();
try { this.#socket.destroy(); } catch { }
```

The retained `#terminalError` is what makes a later `receive()` reject with the *original*
`NI_RECEIVE_TIMEOUT` rather than a generic closed error — asserted at test/ni-socket.test.ts:414-418.

`close()` is the only non-destructive path (src/transport/ni-socket.ts:761-807): state → `"closing"`,
pending receive rejected `NI_CONNECTION_CLOSED`, queue zeroed, decoder reset, then
`this.#socket.once("close", () => settle(false));` and `this.#socket.end()`. The socket is destroyed
**only** if `closeTimeoutMs` elapses first (`settle(true)`, src/transport/ni-socket.ts:789-791) or
`end()` throws (src/transport/ni-socket.ts:803). `close()` is idempotent via `#closePromise`
(src/transport/ni-socket.ts:762).

`#onClose` distinguishes the two: `"closing"` → silently `"closed"`; `"open"` → `#fail(NI_CONNECTION_CLOSED)`
(src/transport/ni-socket.ts:872-885).

The SAProuter handshake's `fail(error, shouldDestroy = true)` (src/transport/saprouter-tunnel.ts:349-359)
is called with `false` in exactly one place — `onClose` (src/transport/saprouter-tunnel.ts:396-401) —
because the stream is already gone. Test: `assert.equal(socket.destroyCalls, 0);`
(test/saprouter-tunnel.test.ts:416). `destroySocket()` is itself guarded by `destroyInvoked`
(src/transport/saprouter-tunnel.ts:343-347) so no path destroys twice.

### 6. Every place a buffer is zeroed

| Site | Code | Citation |
|---|---|---|
| after every `send()`, success or failure | `} finally { frame.fill(0); }` | src/transport/ni-socket.ts:682-684 |
| queue-bound violation in a decoded batch | `for (const frame of frames) frame.fill(0);` before the `RangeError` | src/transport/ni-socket.ts:844 |
| queue-bound violation on a single frame | `frame.fill(0);` before the `RangeError` | src/transport/ni-socket.ts:903 |
| all queued frames on `close()` and `#fail()` | `for (const frame of this.#frames) frame.fill(0); this.#frames.length = 0; this.#queuedPayloadLength = 0;` | src/transport/ni-socket.ts:934-938 |
| SAProuter route payload on an internal length mismatch | `payload.fill(0);` | src/transport/saprouter-route.ts:330 |
| SAProuter NI_ROUTE payload after framing | `request = encodeNiFrame(payload); payload.fill(0);` | src/transport/saprouter-tunnel.ts:521-522 |
| SAProuter request frame after the write completes | `request?.fill(0); request = undefined;` | src/transport/saprouter-tunnel.ts:532-533 |
| SAProuter transient buffers on any settle | `request?.fill(0); … responseHeader.fill(0); responsePayload?.fill(0);` | src/transport/saprouter-tunnel.ts:335-341 |
| SOCKS5 JWT token and location after building the auth frame | `token.fill(0); location.fill(0);` | src/transport/connectivity-socks5-tunnel.ts:460-461 |
| SOCKS5 target address bytes after building CONNECT | `ipv4.fill(0);` / `domain.fill(0);` | src/transport/connectivity-socks5-tunnel.ts:480, 489 |
| SOCKS5 in-flight write frames on failure | `for (const frame of pendingWrites) frame.fill(0); pendingWrites.clear();` | src/transport/connectivity-socks5-tunnel.ts:527-530 |
| SOCKS5 each write frame in its own callback and in the sync-throw path | `frame.fill(0);` (three sites: settled-early, callback, catch) | src/transport/connectivity-socks5-tunnel.ts:575, 582, 592 |
| SOCKS5 handshake buffer on failure | `buffered.fill(0); buffered = Buffer.alloc(0);` | src/transport/connectivity-socks5-tunnel.ts:548-549 |
| SOCKS5 handshake buffer on success | `buffered.fill(0); buffered = Buffer.alloc(0);` | src/transport/connectivity-socks5-tunnel.ts:556-557 |
| SOCKS5 `initialData` if `pause()` then throws | `initialData.fill(0);` | src/transport/connectivity-socks5-tunnel.ts:561 |
| SOCKS5 old buffer on every `consume()` | `const remainder = …; buffered.fill(0); buffered = remainder;` | src/transport/connectivity-socks5-tunnel.ts:599-603 |
| SOCKS5 old buffer on every append | `const next = Buffer.concat(…); buffered.fill(0); buffered = next;` | src/transport/connectivity-socks5-tunnel.ts:716-718 |
| adapters: leftover handoff bytes, success **and** failure | `initialData.fill(0);` / `initialData?.fill(0);` | src/transport/saprouter-ni.ts:97, 100; connectivity-socks5-ni.ts:170, 173 |

Tests that pin the wipe: `wipes the retained NI frame copy after a completed write`
(test/ni-socket.test.ts:297-303), `writes NI_ROUTE once…` (test/saprouter-tunnel.test.ts:132),
`performs SAP JWT method 0x80…` (test/connectivity-socks5-tunnel.test.ts:251),
`write failure releases token frame` (test/connectivity-socks5-tunnel.test.ts:830),
`adopts each established SAProuter stream…` (test/saprouter-ni.test.ts:121),
`destroys a flowing route handoff and never retries it` (test/saprouter-ni.test.ts:155),
`adopts an explicitly configured Connectivity SOCKS5 TCP tunnel for NI` (test/connectivity-socks5-ni.test.ts:95).

### 7. Go equivalent, item by item

| TS mechanism | Go equivalent |
|---|---|
| `socket.pause()` before handoff (src/transport/saprouter-tunnel.ts:444, connectivity-socks5-tunnel.ts:559) | **Nothing.** A `net.Conn` delivers bytes only when someone calls `Read`. Delete the concept; the handshake function simply stops reading and returns. The `"must be paused"` precondition at src/transport/ni-socket.ts:461-467 becomes a Go type-level fact: the constructor takes ownership of a `net.Conn` that no goroutine is reading. |
| `initialData` (src/transport/saprouter-tunnel.ts:472, connectivity-socks5-tunnel.ts:555) | Return `(net.Conn, *bufio.Reader)` or `(net.Conn, []byte)`. The NI transport must read the leftover **first**. Encode this in the type so it cannot be forgotten: `type handoff struct { conn net.Conn; prefix []byte }`, and the transport's reader is `io.MultiReader(bytes.NewReader(prefix), conn)`. |
| listeners installed before resume (src/transport/ni-socket.ts:274-281 → 524), comment `so target NI frames cannot be lost between protocol owners` (src/transport/ni-socket.ts:394-395) | Vacuous in Go — the frames are in `prefix` and in the kernel buffer; nothing is lost because nothing is pushed. **This removes the entire class of bug the ordering exists to prevent.** |
| `#pauseForQueue` / `#resumeAfterQueue` (src/transport/ni-socket.ts:912-932) | A buffered channel of decoded frames with capacity `maxQueuedFrameCount`. The reader goroutine blocks on send when full; that *is* the pause. A byte-budget counter guarded by a mutex enforces `maxQueuedPayloadLength` alongside it (a channel cannot bound bytes). |
| the reader goroutine's lifetime | Owned by the transport; `Close()` closes the conn, the blocked `Read` returns, the goroutine exits. Never leave it blocked on an unbuffered send — use `select { case ch <- f: case <-done: }`. |
| `#pendingReceive` (at most one) + plain `Error` on a second concurrent receive (src/transport/ni-socket.ts:713-715) | `Receive` takes a mutex with `TryLock`; failing to acquire returns a distinct `ErrReceiveInProgress`. Do **not** make this error terminal — it is the one error in the file that leaves the connection usable. |
| `AbortSignal` on connect/adopt/send/receive | `ctx context.Context` as the first parameter of each. `signalAborted()` re-reads (src/transport/ni-socket.ts:224-228) become `ctx.Err() != nil` checks at the same points. |
| receive timeout, write timeout, connect timeout | `conn.SetReadDeadline` / `SetWriteDeadline` / `net.Dialer{Timeout}`. A watcher goroutine on `ctx.Done()` that calls `conn.SetDeadline(time.Now())` gives abort-during-blocking-read for free — that replaces the `onAbort` listener plumbing entirely. |
| `#terminalError ??= error` + `#state` (src/transport/ni-socket.ts:952-956) | `sync.Once` for the teardown plus an `atomic.Value`/mutex-guarded `err error`. Every public method starts with "if err != nil, return err". |
| `try { socket.destroy(); } catch { }` everywhere | `defer conn.Close()` on the failure path — Go's `Close` is safe to call twice, so `destroyInvoked` (src/transport/saprouter-tunnel.ts:343-347) and `shouldDestroy=false` (src/transport/saprouter-tunnel.ts:396-401) both collapse away. |
| `close()`: `end()` then wait up to `closeTimeoutMs` for `"close"`, else `destroy()` (src/transport/ni-socket.ts:773-805) | `conn.(*net.TCPConn).CloseWrite()`, then read-to-EOF with a `SetReadDeadline(now+5s)`, then `Close()`. This is the one place where Go needs *more* code, not less, to preserve behaviour. |
| `frame.fill(0)` / `buffered.fill(0)` | Explicit loops or `clear(b)` (Go 1.21+). Note that `bufio.Reader`'s internal buffer cannot be wiped — if the token-in-memory property is to be preserved, read into caller-owned slices. |
| `NiTimerScheduler` / `SapRouterTimerScheduler` / `ConnectivitySocks5TimerScheduler` | One `Clock` interface (`Now`, `After`, `AfterFunc`) injected via a struct field, or drop the seam and test with real short deadlines. The TS comment warns callbacks `may run synchronously` (src/transport/ni-socket.ts:102) — Go timers never do, so all `if (settled)` re-checks after `setTimeout` disappear. |

---

## Backpressure and queue bounds

### NI transport (the only place with a real queue)

| Bound | Value | Enforcement | Citation |
|---|---|---|---|
| complete frames retained | `maxQueuedFrameCount`, default `1_024` | `this.#frames.length + additionalCount > this.#maxQueuedFrameCount` (batch preflight) and `this.#frames.length >= this.#maxQueuedFrameCount` (per frame) | src/transport/ni-socket.ts:20, 840, 900 |
| retained payload bytes | `maxQueuedPayloadLength`, default `Math.min(maxPayloadLength, 64 * 1024 * 1024)` | `BigInt(this.#queuedPayloadLength) + additionalBytes > BigInt(this.#maxQueuedPayloadLength)` and `queuedPayloadLength > this.#maxQueuedPayloadLength` | src/transport/ni-socket.ts:19, 841-842, 901 |

Overflow behaviour, verbatim (src/transport/ni-socket.ts:839-848):

```
if (
  this.#frames.length + additionalCount > this.#maxQueuedFrameCount ||
  BigInt(this.#queuedPayloadLength) + additionalBytes >
    BigInt(this.#maxQueuedPayloadLength)
) {
  for (const frame of frames) frame.fill(0);
  throw new RangeError(
    "NI complete-frame queue exceeds its configured resource limit",
  );
}
```

The `RangeError` is thrown inside `#onData`'s `try`, so it is caught at src/transport/ni-socket.ts:824
and converted to `NI_PROTOCOL_ERROR` with message `"invalid NI stream"`, which runs `#fail` and kills
the connection. **Overflow is fatal, not lossy.**

Two properties matter for the port:

1. **The check is atomic across a decoded batch and runs before any frame is delivered.** The
   preflight excludes exactly one frame if a receive is pending
   (`const consumedByPending = this.#pendingReceive === undefined || frames.length === 0 ? 0 : 1;`,
   src/transport/ni-socket.ts:832-834), then counts the rest. Test: `rejects a decoded frame batch atomically before resolving a pending receive`
   — the pending receive rejects with `NI_PROTOCOL_ERROR` and never sees `"expected"`
   (test/ni-socket.test.ts:268-295).
2. **BigInt is used for the byte sum** (src/transport/ni-socket.ts:835-842) so a huge batch cannot
   overflow the accumulator. In Go, `uint64` arithmetic with an explicit overflow guard, or compare
   against the remaining budget instead of summing.

Backpressure proper: the socket is paused the moment a batch leaves anything queued
(src/transport/ni-socket.ts:814-816) and resumed only when `receive()` empties the queue
(src/transport/ni-socket.ts:710). It is **frame-granular, not byte-granular** — one queued frame
pauses the whole stream. Test: `assert.equal(paused.state.pauseCalls, 1); assert.equal(paused.state.resumeCalls, 0);`
then `resumeCalls === 1` after the single `receive()` (test/ni-socket.test.ts:220-228).

### SOCKS5 handshake buffer

| Bound | Value | Overflow behaviour | Citation |
|---|---|---|---|
| `maxBufferedBytes` | default `16_384`, range `8..1_048_576` | `if (chunk.length > config.maxBufferedBytes - buffered.length) { fail(…"Connectivity SOCKS5 response exceeded the configured byte bound") }` → `CONNECTIVITY_SOCKS5_PROTOCOL_ERROR`, socket destroyed | src/transport/connectivity-socks5-tunnel.ts:17, 19, 363-368, 709-714 |
| `MAX_ACCESS_TOKEN_BYTES` | `65_536` | rejected at admission, before any I/O | src/transport/connectivity-socks5-tunnel.ts:20, 302-307 |
| in-flight writes | `pendingWrites: Set<Buffer>` — **no numeric bound**; at most three frames exist across the three handshake phases (method / auth / connect) | wiped on failure | src/transport/connectivity-socks5-tunnel.ts:525, 527-530, 781, 624, 646 |

### SAProuter handshake buffer

| Bound | Value | Overflow behaviour | Citation |
|---|---|---|---|
| `maxResponsePayloadBytes` | default `1_048_576`, range `8..1_048_576` | `if (declaredLength < 8 \|\| declaredLength > normalized.maxResponsePayloadBytes)` → `SAPROUTER_PROTOCOL_ERROR` **before** allocating | src/transport/saprouter-tunnel.ts:197-202, 417-426 |
| response allocation | `responsePayload = Buffer.alloc(declaredLength);` — allocated only after the bound check | at most `maxResponsePayloadBytes` bytes | src/transport/saprouter-tunnel.ts:427 |
| route string | `SAPROUTER_MAX_ROUTE_BYTES = 2_048`, `SAPROUTER_MAX_ROUTE_HOPS = 255` | `RangeError` at admission, no I/O | src/transport/saprouter-route.ts:11-12, 189, 224, 234 |

### Services table (message-server resolver)

`1024 * 1024` bytes, `100_000` lines, `4_096` bytes per line — each a `MS_SERVICE_TABLE_INVALID`
(src/transport/message-server-resolver.ts:21-23, 148-176). Test: `rejects conflicting and unbounded service-table data`
(test/message-server-resolver.test.ts:316-338).

### Go equivalent

- Frame queue → `chan []byte` with capacity `maxQueuedFrameCount`. A full channel blocks the reader
  goroutine, which stops calling `conn.Read`, which fills the kernel receive buffer, which shrinks
  the TCP window. That is the same backpressure, expressed by blocking rather than by `pause()`.
- The byte budget cannot ride on a channel. Keep a mutex-guarded `queuedBytes uint64` updated on
  every send and receive, and check *before* sending. To preserve the atomic-batch property, the
  decoder must produce the whole batch first, check both bounds, and only then push — exactly as
  `#preflightDecodedFrames` does.
- Overflow must return a **terminal** error that also closes the conn, matching
  src/transport/ni-socket.ts:824-828 → `#fail`. Do not silently drop.
- The handshake buffers become fixed-size `[]byte` read with `io.ReadFull` after a bounds check;
  neither handshake needs a growable buffer in Go.

---

## Route composition

### The shape

There are three transport factories with the **same** call signature and one resolver that has a
different shape entirely.

| Route | Entry point | Produces | Citation |
|---|---|---|---|
| direct | `NiSocketTransport.connect(options, signal, scheduler)` | `Promise<NiSocketTransport>` | src/transport/ni-socket.ts:284-288 |
| SAProuter | `createSapRouterDirectCpicTransportFactory(routePrefix, deps)` → `(options: NiSocketConnectOptions, signal?) => Promise<NiSocketTransport>` | `Promise<NiSocketTransport>` | src/transport/saprouter-ni.ts:42-48 |
| Connectivity SOCKS5 | `createConnectivitySocks5DirectCpicTransportFactory(proxyInput, deps)` → `DirectCpicTransportFactory` | `Promise<NiSocketTransport>` | src/transport/connectivity-socks5-ni.ts:118-121 |
| message-server-resolved | `resolveMessageServerRfcGroup(options)` | `Promise<MessageServerRfcGroupTarget>` — **an endpoint, not a connection** | src/transport/message-server-resolver.ts:259-261 |

### How the wrapping actually works

Both routed factories follow the identical five-step shape:

1. Build a route/config from the **factory-fixed** part plus the **per-call** `options.host`/`options.port`
   (src/transport/saprouter-ni.ts:60-64; connectivity-socks5-ni.ts:132-141).
2. Call an injectable connector that performs a TCP connect **and** a protocol handshake, returning
   `{ socket, initialData }` (src/transport/saprouter-ni.ts:68-77; connectivity-socks5-ni.ts:145-148).
3. Structurally validate the returned object (src/transport/saprouter-ni.ts:78-87; connectivity-socks5-ni.ts:149-159).
4. `NiSocketTransport.adopt({ socket, initialData, …caller bounds }, signal)`
   (src/transport/saprouter-ni.ts:88-96; connectivity-socks5-ni.ts:161-169).
5. Wipe `initialData` and return; on any throw, wipe and `socket?.destroy()`
   (src/transport/saprouter-ni.ts:97-107; connectivity-socks5-ni.ts:170-176).

So the composition is **not** a chain. Each route is a *replacement* for the TCP connect step, and
all three converge on the same `NiSocketTransport`. The only thing that crosses the seam is a
connected byte stream plus over-read bytes.

The message-server resolver composes *orthogonally*: it takes a `MessageServerTransportFactory`
(defaulting to `NiSocketTransport.connect`, src/transport/message-server-resolver.ts:247-250), uses
it for one throwaway login/group/logout exchange, and returns a `MessageServerRfcGroupTarget`
describing an application server. The `applicationServerHost` / `gatewayPort` / `gatewayService` it
yields (asserted at test/message-server-resolver.test.ts:94-100) are then the input to a *separate*
connection made by one of the three routes above. Nothing in `src/transport/` wires that second step —
that lives outside this layer.

**Nothing in this layer composes two routes.** No test constructs SAProuter-over-SOCKS5 or
SOCKS5-over-SAProuter. The comment at src/transport/saprouter-ni.ts:37-38 (`Every invocation
negotiates a fresh route for exactly one target connection; no routed stream is shared or replayed.`)
is the strongest statement of intent available.

### Should the Go equivalent be a `net.Conn` decorator chain?

**Partly, and the seams differ from what the TS suggests.**

Where a decorator works:

- `EstablishedSapRouterRoute` and `EstablishedConnectivitySocks5Tunnel` are structurally identical
  (`{socket, initialData}`; src/transport/saprouter-tunnel.ts:159-166 vs
  connectivity-socks5-tunnel.ts:144-149). In Go both become a single
  `type Dialer interface { DialContext(ctx, host string, port int) (net.Conn, error) }`, where the
  returned `net.Conn` already has the leftover bytes spliced in front of its `Read`. That is a
  textbook decorator, and it is strictly cleaner than TS: the `initialData` field disappears from
  the public surface.
- `net.Dialer`, a SAProuter dialer, and a SOCKS5 dialer all satisfy it, and the NI transport takes a
  `Dialer` rather than three factories.

Where it breaks down — four places, each needing a decision:

1. **The leftover bytes are secret material that must be wiped** (src/transport/saprouter-ni.ts:97,
   connectivity-socks5-ni.ts:170; tests test/saprouter-ni.test.ts:121, connectivity-socks5-ni.test.ts:95).
   A `net.Conn` decorator that holds a `[]byte` prefix has no lifecycle hook to zero it. Either the
   NI transport wipes the prefix after draining it, or the decorator exposes an explicit
   `Handoff() (net.Conn, []byte)` instead of hiding the prefix. The wipe is asserted by tests, so it
   cannot be dropped.
2. **The per-call bounds are the NI transport's, not the route's.** `adopt` receives
   `maxPayloadLength`, `maxQueuedPayloadLength`, `maxQueuedFrameCount`, `writeTimeoutMs`,
   `closeTimeoutMs` from the caller's `NiSocketConnectOptions` and passes them straight through
   (src/transport/saprouter-ni.ts:91-95; connectivity-socks5-ni.ts:164-168), while the route consumes
   only `connectTimeoutMs`, `family`, `noDelay` (src/transport/saprouter-ni.ts:70-75). A `Dialer`
   interface must therefore **not** carry NI bounds; split the options struct along that line.
3. **Timeout precedence differs between the two routes.** SAProuter fans one value into both phases
   (`connectTimeoutMs: timeoutMs, handshakeTimeoutMs: timeoutMs`, src/transport/saprouter-ni.ts:71-72),
   while SOCKS5 lets a factory-configured `timeoutMs` **override** the caller
   (`proxy.timeoutMs ?? options.connectTimeoutMs ?? DEFAULT_TIMEOUT_MS`, src/transport/connectivity-socks5-ni.ts:139).
   A single `ctx` deadline flattens both and would change behaviour. Keep an explicit per-dialer
   timeout field.
4. **The message-server resolver is not a decorator at all.** It consumes a transport factory and
   emits an endpoint. In Go it is a function `ResolveRFCGroup(ctx, opts) (Target, error)` taking a
   `Dialer`, not a `Dialer` itself. Do not force it into the chain.

Also note the error types do not compose: `NiTransportError`, `SapRouterTransportError`, and
`ConnectivitySocks5Error` are three unrelated classes with three disjoint code unions
(src/transport/ni-socket.ts:9-17, saprouter-tunnel.ts:21-31, connectivity-socks5-tunnel.ts:36-45),
and the adapters propagate route errors unchanged (src/transport/saprouter-ni.ts:106,
connectivity-socks5-ni.ts:175). In Go this is three error types plus `errors.As`; do **not** flatten
them into one enum, because callers distinguish them (test/saprouter-tunnel.test.ts:564,
test/connectivity-socks5-tunnel.test.ts:435-437).

---

## Open questions for the porter

1. **`DEFAULT_MAX_NI_PAYLOAD_LENGTH` and `MAX_MESSAGE_SERVER_PAYLOAD_LENGTH` values are outside this
   scope.** They are imported at src/transport/ni-socket.ts:4 and message-server-resolver.ts:9. A
   test asserts the message-server one is `512` (test/message-server-resolver.test.ts:268); the NI
   one is never asserted in the files read. Read `src/protocol/ni.js` before setting the Go default —
   it also determines the default `maxQueuedPayloadLength` via
   `Math.min(maxPayloadLength, 64MiB)` (src/transport/ni-socket.ts:160).
2. **`NiFrameDecoder` semantics are assumed, not documented here.** `push()` returns an array of
   complete frames, `finish()` throws if a frame is truncated, `reset()` discards state
   (src/transport/ni-socket.ts:812, 854, 772, 958). Whether `push` returns slices that alias the
   input chunk matters enormously in Go (the queue holds them, and `#clearQueuedFrames` zeroes them).
   Out of scope — check `src/protocol/ni.ts`.
3. **`DirectCpicTransportFactory`** is imported from `../client/direct-cpic-session.js`
   (src/transport/connectivity-socks5-ni.ts:3) and is the declared return type of the SOCKS5 factory,
   while the SAProuter factory spells its signature out inline (src/transport/saprouter-ni.ts:45-48).
   Confirm the two are actually identical before unifying them behind one Go interface.
4. **`close()`'s half-close dance has no test in scope.** `close()` calls `end()` and waits up to
   `DEFAULT_NI_CLOSE_TIMEOUT_MS` (`5_000`) for `"close"` before destroying
   (src/transport/ni-socket.ts:786-804). Tests only assert `state === "closed"` afterwards
   (test/ni-socket.test.ts:154-155) and that peers observe the close (test/ni-resource.test.ts:93).
   Whether the remote is expected to see FIN-then-FIN or FIN-then-RST is unspecified; pick
   `CloseWrite` + drain and verify against a real SAP gateway.
5. **`NiTimerHandle = number | object` and the "callbacks may run synchronously" contract**
   (src/transport/ni-socket.ts:97, 102) exist only for tests. Decide whether the Go port keeps a
   `Clock` seam or tests against real deadlines; if the seam is dropped, several code paths
   (src/transport/ni-socket.ts:362-365, 645-648, 746-747; saprouter-tunnel.ts:503-507;
   connectivity-socks5-tunnel.ts:769-773) have no Go analogue and should not be ported at all.
6. **Is `send()`'s abort meant to be fatal?** `send` with an aborted signal calls `#fail`
   (src/transport/ni-socket.ts:607-614, 631-632), destroying the connection. The class comment
   justifies this for *receive* (`a late RFC response must never be mistaken for the response to a
   later call`, src/transport/ni-socket.ts:239-241) but says nothing about writes. No test in scope
   covers `send` abort. If it is deliberate, document it; if incidental, the Go port must still
   match it to stay behaviour-compatible.
7. **`pendingWrites` has no bound** (src/transport/connectivity-socks5-tunnel.ts:525). It is bounded
   in practice by the three-phase handshake, but nothing enforces that. Not a Go problem —
   `conn.Write` is synchronous — but worth confirming no fourth write can be queued.

## Answers to this document's open questions, from the ported NI layer

Both questions concern `src/protocol/ni.ts`, which is outside this inventory's
scope but is already ported. Answered here so the transport port does not have
to re-derive them.

**`DEFAULT_MAX_NI_PAYLOAD_LENGTH` is `256 * 1024 * 1024`** (256 MiB), at
`src/protocol/ni.ts:5`, ported as `ni.DefaultMaxPayloadLength` in
`internal/ni/frame.go`. The default `maxQueuedPayloadLength` derived from it as
`Math.min(maxPayloadLength, 64 MiB)` is therefore **64 MiB**.

**Decoded payloads never alias the pushed chunk, in either implementation.**
Upstream `#consume` allocates with `Buffer.allocUnsafe(length)` and copies into
it (`src/protocol/ni.ts:103-119`); the queue itself holds `Buffer.from(chunk)`
copies (`src/protocol/ni.ts:46`). So there are three separate byte regions —
the caller's chunk, the retained queue copy, and the returned payload — and
zeroing the queue cannot affect a payload already handed out.

The Go port preserves exactly this: `Push` copies the caller's chunk before
retaining it, and `consume` allocates a fresh slice per payload
(`internal/ni/frame.go`), with `TestPushDoesNotAliasCallerChunk` pinning the
first half of the property.

Consequence for the transport port: **it must not try to slice.** A zero-copy
NI decoder is a different design from the one being ported, and adopting it
here would mean the queue-zeroing that upstream relies on for credential
hygiene would silently start clearing bytes a caller still holds.
