# Cross-layer answers

Each surface inventory was deliberately scoped to one layer, so several of them
end with a question whose answer lives in a layer they were not allowed to read.
Those answers are collected here, with citations, so the port does not
re-derive them — or worse, guess.

Upstream baseline: `847036dce5e29015bbc266a4d19cc9c15295a831` (open-rfc 0.2.3).

## 1. Metadata identity keys, for `src/destination/runtime.ts`

`pool-destination-diagnostics.md` cannot port `src/destination/runtime.ts`
without `MetadataCapabilityKey.id` and `MetadataStructuralKey`. Both are in
`src/metadata/repository-runtime.ts`.

**`MetadataStructuralKey`** — five fields plus a derived `id`
(`repository-runtime.ts:54-66`), documented as *"A descriptor identity which
deliberately contains no authenticated principal."*

```ts
readonly backendKey: string
readonly metadataGeneration: string
readonly language: string
readonly objectKind: string
readonly objectName: string
readonly id: string          // tupleId of the five above, in that order (:404-424)
```

**`MetadataCapabilityKey`** — two fields plus a derived `id`
(`repository-runtime.ts:67-75`), documented as *"A backend
capability/authorization identity scoped to one opaque principal."*

```ts
readonly backendKey: string
readonly principalKey: string
readonly id: string          // tupleId([backendKey, principalKey]) (:431-441)
```

The split is the point: the structural key has **no principal**, so descriptors
cache across principals; the capability key **has** one, so authorization and
probe decisions do not. A Go port that merges them into one cache key leaks
authorization state between principals.

Both builders run every field through `opaqueIdentity`, which requires a string
of 1..512 characters containing no `\u0000`-`\u001f` and no `\u007f`, and throws
`` `${field} must contain 1..512 characters without controls` `` otherwise
(`:361-373`). Both freeze their result. Supplied ids are re-derived and compared
rather than trusted (`:671-675`).

## 2. The canonical-id encoding hazard

`tupleId` is exactly this (`repository-runtime.ts:375-377`):

```ts
function tupleId(values: readonly string[]): string {
  return JSON.stringify(values);
}
```

`pool-destination-diagnostics.md` open question 11 raises the same hazard one
layer down, where `backendKey` itself is built with `JSON.stringify` and hashed
(`src/destination/configuration-generation.ts:194-201`).

**`encoding/json` does not produce the same bytes as `JSON.stringify`.** For a
JavaScript array of strings the two differ in at least three ways, and every one
of them changes a cache key or a hash:

| Input contains | `JSON.stringify` | Go `json.Marshal` |
|---|---|---|
| `<`, `>`, `&` | emitted raw | escaped `\u003c`, `\u003e`, `\u0026` |
| U+2028, U+2029 | emitted raw | escaped `\u2028`, `\u2029` |
| an unpaired surrogate | escaped `\udXXX` | not representable in a Go string; invalid UTF-8 becomes U+FFFD |

The first two are reachable: `opaqueIdentity` rejects only controls, so a
function-module name, language, or backend key containing `&` passes validation
and would hash differently in Go. `SetEscapeHTML(false)` on a `json.Encoder`
fixes the first row and not the second.

**Therefore the Go port must not use `encoding/json` for any identity or hash
input.** Write an explicit canonical encoder — a small function that emits the
JavaScript form — and pin it with a conformance vector covering `&`, `<`,
U+2028, and a multi-byte character. The unpaired-surrogate row is a genuine
semantic gap rather than an encoding choice: Go strings cannot hold one, so that
input is unrepresentable rather than differently encoded. Record it as a known
divergence; do not paper over it.

This is worth stating plainly because it is invisible: both implementations look
correct in isolation, all tests pass on both sides, and the two disagree only on
identity strings for inputs nobody thought to test.

## 3. NI framing, for the transport port

Answered in full at the end of `transport.md`. In brief:
`DEFAULT_MAX_NI_PAYLOAD_LENGTH` is 256 MiB (`src/protocol/ni.ts:5`), making the
derived default `maxQueuedPayloadLength` 64 MiB; and decoded payloads never
alias the pushed chunk in either implementation, so the transport port must not
attempt a zero-copy decoder.

## 4. Compact temporal: signed read vs unsigned assembly

`values.md` reports as blocking that `src/values/classic-temporal.ts:431-435`
reads the raw **signed** (`readBigInt64LE` / `readInt32LE` / `readInt16LE`, then
rejects `raw < 0n`) while `src/values/recursive-xrfc.ts:1036-1041` assembles the
same bytes **unsigned**, and asks whether Go needs `int64` or `uint64`.

**It is not a divergence, and the Go answer is `uint64` either way.** Every
`maximumRaw` in the spec table sits far below the signed maximum for its width
(`classic-temporal.ts:69-111`):

| Spec | Width | `maximumRaw` | Signed max for that width |
|---|---|---|---|
| `UTCLONG` | 8 | 3_155_380_704_000_000_000 | 9_223_372_036_854_775_807 |
| `UTCSECOND` | 8 | 315_538_070_400 | 9_223_372_036_854_775_807 |
| `UTCMINUTE` | 8 | 5_258_967_840 | 9_223_372_036_854_775_807 |
| `DTDAY` | 4 | 3_652_061 | 2_147_483_647 |
| `DTWEEK` | 4 | 521_725 | 2_147_483_647 |
| `DTMONTH` | 4 | 119_988 | 2_147_483_647 |
| `TSECOND` | 4 | 86_401 | 2_147_483_647 |
| `TMINUTE` | 2 | 1_441 | 32_767 |
| `CDAY` | 2 | 366 | 32_767 |

So any byte pattern with the high bit set is rejected on both paths — as
negative on one, as `> maximumRaw` on the other. The accept/reject sets are
identical, which is why no test distinguishes them: there is no input that
distinguishes them.

**Port it as `uint64` assembly followed by an explicit `> maximumRaw` check.**
That reproduces both behaviours exactly and removes the question, rather than
translating a signed read whose sign can never be observed.

**One genuine asymmetry remains, and it is not the one asked about.**
`recursive-xrfc.ts:1579` bounds the parsed text by `1n << BigInt(width * 8)` —
the full unsigned width — rather than by `spec.maximumRaw`, so it admits raws
above the valid temporal range that `classic-temporal.ts` would reject. Upstream
states the principle that "a reader may accept more than the writer emits; that
asymmetry is deliberate and correct" (`docs/architecture.md`), so this may be
intentional. Whether it is depends on whether that function is on the inbound or
outbound path, which the values inventory did not establish. **Open upstream
question; the Go port should bound by `maximumRaw` in both places until it is
answered, because the tighter bound is never wrong.**

## 5. CPIC limit constants used by the values layer

`values.md` notes that three defaults it depends on are defined outside its
scope. They are in `src/protocol/cpic.ts:21-25`:

| Constant | Value |
|---|---|
| `DEFAULT_MAX_CPIC_FIELD_LENGTH` | `256 * 1024 * 1024` |
| `DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH` | `256 * 1024 * 1024` |
| `DEFAULT_MAX_CPIC_FIELD_COUNT` | `100_000` |
| `CLASSIC_XRFC_XML_CHUNK_LENGTH` | `16 * 1024` |

Note the exact name `DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH`, not
`DEFAULT_MAX_CPIC_CHAIN_LENGTH`. All three are per-connection configurable
(`cpic.ts:628-631`); the constants are defaults, not protocol limits.

## 6. `localeCompare` in the graph digest

`recursive-serializer-classification.ts:143` sorts graph nodes with
`left.localeCompare(right)` before `JSON.stringify` and SHA-256
(`:154-158`). This is a second, independent instance of the hazard in section 2,
and a worse one: `localeCompare` uses ICU collation, so `"a".localeCompare("B")`
is negative while Go's byte-wise ordering puts `"B"` first. Any graph with
mixed-case node names sorts differently, producing a different digest.

Reproducing ICU root collation in Go means `golang.org/x/text/collate` — a
dependency, and still not a guarantee of byte-identical ordering across ICU
versions.

**Decide by what the digest is for.** If it is internal content identity — a
cache key or change detector local to one process — the Go port should sort by
byte order, document the divergence, and never compare a digest across
implementations. If it is ever emitted, logged for correlation, or compared with
a value produced by the TypeScript implementation, it is a wire value and needs
an exact canonical ordering pinned by a conformance vector. The values inventory
did not establish which; resolve it before porting
`recursiveMetadataGraphSha256`.
