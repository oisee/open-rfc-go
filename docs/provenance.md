# Provenance

Every file in this repository that was ported from
[`open-rfc`](https://github.com/marianfoo/open-rfc) is listed here with the
exact upstream file and commit it came from, and what changed in translation.

This table exists for two reasons. Apache License 2.0 §4(b) requires prominent
notices stating which files were changed. And it is the index used to propagate
an upstream fix: when open-rfc changes `src/protocol/ni.ts`, this table says
which Go file to look at, and the recorded commit says how far behind we are.

Upstream baseline: `847036dce5e29015bbc266a4d19cc9c15295a831` (open-rfc 0.2.3).

To check whether a ported file has drifted, with an open-rfc checkout alongside
this one:

```sh
git -C ../open-rfc log --oneline 847036d..origin/main -- src/protocol/ni.ts
```

## Ported source

| open-rfc file | open-rfc commit | open-rfc-go file | Modifications |
|---|---|---|---|
| `src/protocol/ni.ts` | `847036d` | `internal/ni/frame.go` | Rewritten in Go. `Buffer` → `[]byte`; thrown `RangeError`/`Error` → returned wrapped sentinel errors; `#private` fields → unexported fields; the `bytes.ts` typed-array intrinsic guards are dropped as inapplicable (see below). |
| `test/ni.test.ts` | `847036d` | `internal/ni/frame_test.go` | Rewritten for `testing`. One upstream case has no Go analogue (see below); three Go-specific cases added for slice aliasing and a fuzz target. |
| `test/protocol-property.test.ts` (NI cases only) | `847036d` | `internal/ni/property_test.go` | Rewritten for `testing`. Upstream's hand-rolled `DeterministicRandom` → `math/rand/v2` PCG with a fixed seed, so the sequences differ while the invariant asserted is the same. |

## Copied documentation

| open-rfc file | open-rfc commit | open-rfc-go file | Modifications |
|---|---|---|---|
| `docs/recurring-bug-class.md` | `847036d` | `docs/recurring-bug-class.md` | Verbatim prose, with an attribution header prepended and four relative links to `src/`/`test/` repointed at the upstream repository at that commit, since those paths do not exist here. |
| `docs/architecture.md` | `847036d` | `docs/architecture.md` | Adapted: layer table rewritten for Go packages, Node-specific ladder steps replaced, evidence hierarchy kept unchanged. |
| `DCO.md` | `847036d` | `DCO.md` | Verbatim (the document forbids modification). |
| `LICENSE` | `847036d` | `LICENSE` | Verbatim Apache License 2.0. |

## Not ported, deliberately

| Upstream | Why |
|---|---|
| `src/compat/**` | Exists to be bug-compatible with the archived `node-rfc` npm package and with `@sap/cds-rfc`. Neither has a Go consumer. |
| `src/compat/node-rfc-public-surface.ts`, `src/client/rfc-errors.ts` | The only upstream files containing third-party (SAP SE, `node-rfc`) code. Because they are not ported, this repository carries no `THIRD_PARTY_NOTICES.md`; if either is ever ported, that file and its Apache-2.0 attribution to SAP SE must come with it. |
| `src/protocol/bytes.ts` | Defends against JavaScript callers that override `Uint8Array` geometry accessors. Go slices expose no user-overridable geometry, so the entire class of attack is absent. The *intent* — never alias caller memory — is preserved by copying in `EncodeFrame` and `Push`. |
| `test/ni.test.ts`, case "encodes from intrinsic typed-array geometry" | Tests the above. No Go analogue is possible; `internal/ni/frame_test.go` covers the underlying intent with `TestEncodeFrameDoesNotAliasCallerPayload`. |

## Rules for adding to this table

- A ported file carries an SPDX header naming its upstream file and commit.
- A file with no upstream counterpart carries an SPDX header and no provenance
  line; it is original work.
- Record the commit you actually read, not `main`.
