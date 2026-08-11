# Cheat sheet

Everything you need on one page. Every value is cited into the upstream
TypeScript at commit `847036d`; nothing here is from memory or from general SAP
knowledge. Paths without a repository prefix are upstream
(`../open-rfc/<path>`).

---

## Ports

**Direct RFC dials the gateway: `3300 + sysnr`.**

| `sysnr` | Gateway port |
|---|---|
| `00` | 3300 |
| `01` | 3301 |
| `42` | 3342 |

Override with `gwserv` (numeric, or the `sapgwNN` service name) or `port`
(`compat/connection-parameters.ts:234-249`).

> **Not the dispatcher port.** `32NN` is the dispatcher. The normalized config
> records `sapdp<NN>` as a label (`:328`) but connects to the gateway. Pinning
> the range 3200–3299 is one of the six documented recurring bugs — it broke
> every port-offset landscape.

Preview routes, all outside the supported boundary:

| Route | Port | Citation |
|---|---|---|
| Message server | service name `sapms<SID>` via `/etc/services`; **no numeric default** | `transport/message-server-resolver.ts:232` |
| SAProuter | `3299` | `transport/saprouter-route.ts:4` |
| BTP Connectivity SOCKS5 | from the service binding, normally `20004` | `transport/connectivity-socks5-tunnel.ts:83` |

The message-server case refuses to guess `3600+NN` and errors with *"TCP service
X is not defined in /etc/services; provide a numeric msserv"* (`:217`) — a
deliberate refusal to invent a default that happens to work on one landscape.

---

## Authentication

**One mechanism is implemented: user + password.**

| Parameter | Notes |
|---|---|
| `user` | required |
| `passwd` | printable ASCII `0x20`–`0x7e`, **≤ 40 bytes** (`protocol/password-scramble.ts:16-27`) |
| `client` | 1–3 digits, zero-padded to 3 (`connection-parameters.ts:228-232`) |
| `lang` | ISO or SAP single-character code |

The password is scrambled with a fixed 64-byte table plus a random 4-byte seed
(`password-scramble.ts:3-10`). Out-of-range characters are refused as *"outside
the proven ASCII baseline"* rather than transcoded.

**Rejected, not ignored** (`connection-parameters.ts:289-297`):

| Parameter | Error |
|---|---|
| `snc_mode` | `snc_mode connections are not implemented; use direct ashost/sysnr` |
| `wshost` | same shape — WebSocket RFC |
| `saprouter` | same shape on the direct path; reachable via the route planner (`connection-route.ts:593-660`) |

`EnumSncQop` is exported for `node-rfc` API compatibility only; importing it does
not widen the SNC boundary. **No x509, SSO ticket, SAML/JWT, or trusted-system
RFC exists anywhere in the tree.**

⚠️ No SNC means no transport encryption and no peer authentication. Do not send
credentials or RFC traffic across an untrusted network.

---

## Layer map

```
                                     upstream              open-rfc-go
  ┌─────────────────────────────┐
  │ values, metadata            │   src/values/            internal/value/
  │                             │   src/metadata/          internal/metadata/
  ├─────────────────────────────┤
  │ RFCPRO field headers        │   protocol/rfcpro.ts     internal/rfcpro/
  ├─────────────────────────────┤
  │ CPIC field chains           │   protocol/cpic.ts       internal/cpic/
  ├─────────────────────────────┤
  │ APPC conversation records   │   protocol/appc.ts       internal/appc/
  ├─────────────────────────────┤
  │ gateway handshake           │   protocol/gateway.ts    internal/gateway/
  ├─────────────────────────────┤
  │ NI record framing           │   protocol/ni.ts         internal/ni/      ✅
  ├─────────────────────────────┤
  │ TCP                         │   src/transport/         internal/transport/
  └─────────────────────────────┘
```

Every byte above TCP is inside an NI record. See
[protocol-primer.md](protocol-primer.md) for what each layer does.

---

## Connect sequence

Order verified in `client/direct-cpic-session.ts`:

1. TCP connect to the gateway port
2. **Gateway normal-client record** — 64 bytes, protocol version 2, request type 3 (`:767`)
3. **Check `acceptInfo` capability bits** — fail closed if `ExtendedInitOptions` or `CodePage` is absent (`:798-802`)
4. **APPC `Initialize` control record** (`:812-813`)
5. CPIC initial logon fields, password scrambled
6. RFC call

---

## Constants

### NI framing — `protocol/ni.ts`

| Name | Value |
|---|---|
| length prefix | 4 bytes, big-endian |
| `DEFAULT_MAX_NI_PAYLOAD_LENGTH` | `256 * 1024 * 1024` (`:5`) |
| derived default `maxQueuedPayloadLength` | `min(maxPayloadLength, 64 MiB)` = 64 MiB |
| `DEFAULT_MAX_NI_QUEUED_FRAME_COUNT` | `1_024` (`transport/ni-socket.ts:21`) |

### Gateway — `protocol/gateway.ts`

| Name | Value |
|---|---|
| `GATEWAY_NORMAL_CLIENT_LENGTH` | `64` (`:5`) |
| `GATEWAY_PROTOCOL_VERSION` | `2` (`:6`) |
| `GATEWAY_NORMAL_CLIENT_REQUEST` | `3` (`:7`) |

### APPC — `protocol/appc.ts`

| Name | Value |
|---|---|
| `APPC_PROTOCOL_VERSION` | `0x06` (`:8`) |
| `APPC_COMMON_HEADER_LENGTH` | `48` (`:9`) |
| `APPC_RECORD_HEADER_LENGTH` | `80` (`:11`) |
| `APPC_EXTENDED_INITIALIZE_OPTIONS_LENGTH` | `341` (`:12`) |
| `APPC_INITIALIZE_PARAMETERS_LENGTH` | `373` (`:13`) |
| `APPC_PARTNER_PARAMETERS_LENGTH` | `144` (`:14`) |
| `MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH` | `28_000` (`:19`) |
| `MAX_APPC_ASYNC_SENDS_BEFORE_SYNC` | `21` (`:21`) |
| `DEFAULT_MAX_APPC_OUTGOING_MESSAGE_LENGTH` | `1_400_000` (`:28`) |
| `DEFAULT_MAX_APPC_MESSAGE_LENGTH` | `256 * 1024 * 1024` (`:29`) |
| `DEFAULT_MAX_APPC_MESSAGE_FRAGMENTS` | `65_536` (`:30`) |

### CPIC — `protocol/cpic.ts`

| Name | Value |
|---|---|
| `DEFAULT_MAX_CPIC_FIELD_LENGTH` | `256 * 1024 * 1024` (`:21`) |
| `DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH` | `256 * 1024 * 1024` (`:22`) |
| `DEFAULT_MAX_CPIC_FIELD_COUNT` | `100_000` (`:23`) |
| `CLASSIC_XRFC_XML_CHUNK_LENGTH` | `16 * 1024` (`:25`) |

Note the exact name — `..._FIELD_CHAIN_LENGTH`, not `..._CHAIN_LENGTH`. All are
per-connection configurable defaults (`:628-631`), **not protocol limits**.

### RFCPRO — `protocol/rfcpro.ts`

| Name | Value |
|---|---|
| `RFC_PRO_EXTENDED_LENGTH_SENTINEL` | `0xffff` (`:3`) |
| `RFC_PRO_COMPACT_LENGTH_MAX` | `0xfffe` (`:4`) |
| `RFC_PRO_VALUE_LENGTH_MAX` | `0x7fff_ffff` (`:5`) |
| header width | 4 bytes compact, 8 bytes extended (`:41-45`) |

### Row lengths — `protocol/classic-rfc.ts`

| Name | Value | How it is used |
|---|---|---|
| `RFC_FUNINT_UNICODE_ROW_LENGTH` | `402` (`:16`) | **lower bound + read window**, not equality |

> Longer rows are accepted and truncated; a 404-byte row is already evidenced in
> the wild. Short rows are refused, because *"completing one with ABAP initial
> bytes would invent values the peer never sent"* (`:554-563`). This is the
> inverse of the recurring bug, not an instance of it.

### Compact temporal ranges — `values/classic-temporal.ts:69-111`

| Spec | Width | `maximumRaw` |
|---|---|---|
| `UTCLONG` | 8 | 3_155_380_704_000_000_000 |
| `UTCSECOND` | 8 | 315_538_070_400 |
| `UTCMINUTE` | 8 | 5_258_967_840 |
| `DTDAY` | 4 | 3_652_061 |
| `DTWEEK` | 4 | 521_725 |
| `DTMONTH` | 4 | 119_988 |
| `TSECOND` | 4 | 86_401 |
| `TMINUTE` | 2 | 1_441 |
| `CDAY` | 2 | 366 |

Every maximum is below its width's **signed** ceiling, which is why the signed
read in `classic-temporal.ts` and the unsigned assembly in `recursive-xrfc.ts`
accept and reject identically. Port as `uint64` + explicit bound.

---

## Go API — what exists

```go
import "github.com/oisee/open-rfc-go/internal/ni"   // internal: not importable
```

| Symbol | Signature |
|---|---|
| `ni.DefaultMaxPayloadLength` | `= 256 * 1024 * 1024` |
| `ni.EncodeFrame` | `func([]byte) ([]byte, error)` |
| `ni.NewFrameDecoder` | `func(maxPayloadLength int) (*FrameDecoder, error)` |
| `(*FrameDecoder).Push` | `func([]byte) ([][]byte, error)` |
| `(*FrameDecoder).Finish` | `func() error` |
| `(*FrameDecoder).Reset` | `func()` |
| `(*FrameDecoder).Buffered` | `func() int` |

Sentinel errors, matched with `errors.Is`:

| Error | Meaning |
|---|---|
| `ni.ErrPayloadTooLong` | payload exceeds the uint32 length field |
| `ni.ErrPayloadTooLarge` | advertised length above the configured limit — **discard the connection** |
| `ni.ErrTruncatedStream` | peer closed mid-record |
| `ni.ErrNegativeLimit` | invalid decoder configuration |
| `ni.ErrInconsistentQueue` | internal accounting fault; unreachable by construction |

---

## Commands

```sh
go build ./...
go test -race -shuffle=on ./...              # -race is a security control, not a nicety
gofmt -l .                                   # must print nothing
go vet ./...
go test -cover ./internal/...
go test ./conformance/...                    # language-neutral wire vectors
go test ./internal/ni -run TestX -fuzz FuzzFrameDecoderBounds -fuzztime 60s
go test -race -run TestDecodesFragmented ./internal/ni -v    # one test
```

Check whether a ported file has drifted from upstream:

```sh
git -C ../open-rfc log --oneline 847036d..origin/main -- src/protocol/ni.ts
```

After editing any document containing `\uXXXX` escape text:

```sh
grep -aP '[\x00-\x1f\x7f]' docs/**/*.md      # must match nothing
```

---

## Cross-language hazards

| Hazard | Symptom | Rule |
|---|---|---|
| `JSON.stringify` ≠ `encoding/json` | different identity strings and hashes for input containing `<`, `>`, `&`, U+2028 | write an explicit canonical encoder; `SetEscapeHTML(false)` is not enough |
| `localeCompare` ≠ byte order | different digest for mixed-case node names | decide whether the digest is a wire value first |
| unpaired surrogates | unrepresentable in Go | record as a known divergence; do not paper over |
| Number precision | 53-bit float limits in upstream arithmetic | `int64`/`uint64`/`math/big`, never `float64` |
| Slice aliasing | Go slices alias; JS `Buffer.from` copies | copy on retain and on return; add an aliasing test |
| GC and zeroing | `clear()` cannot reach GC-moved copies | narrows the window, is not a control |

---

## Where things belong

| Kind of problem | Goes to |
|---|---|
| Wire behaviour wrong in both implementations | [open-rfc](https://github.com/marianfoo/open-rfc/issues) |
| Translation mistake — aliasing, width, encoding, concurrency | here |
| Security issue | privately, per [`SECURITY.md`](../SECURITY.md) |

Test: *does the TypeScript do the same thing?* Yes → upstream. No → here.
