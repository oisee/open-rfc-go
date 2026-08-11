# Conformance vectors

Language-neutral wire vectors: hex in, hex or a named error out. They exist so
that a wire fact can be stated once and checked by more than one
implementation.

## Why they are here and not upstream

[`open-rfc`](https://github.com/marianfoo/open-rfc) is the authority on wire
behaviour, but its wire facts live inside `.test.ts` files, where they are
reachable only by importing TypeScript functions. A second implementation
cannot consume them, so a port drifts silently: both suites pass, and the two
decoders disagree about bytes no test compares.

These vectors are therefore **authored here**, derived from upstream's tests and
sources at commit `847036d`, and they carry a `provenance` field saying so.

## The intended end state

Upstream owns the corpus; this repository vendors a copy with a checksum lock
(`vectors.lock.json`), and a scheduled job opens a pull request when the two
diverge. That requires an upstream change, so it is recorded as intent, not
presented as fact — there is no lock file yet because there is nothing upstream
to lock against.

Until then, treat a vector here as this project's reading of upstream, not as
upstream's own statement.

## Format

One file per wire rule, `<layer>-<rule>.v<schema>.json`.

| Field | Meaning |
|---|---|
| `rule` | The invariant in one sentence. If you cannot write it, the vector is an observation, not a rule — see `docs/recurring-bug-class.md`. |
| `provenance` | Where the fact came from, and at which commit. |
| `encode[]` | `payloadHex` → `frameHex`. |
| `decode[]` | `chunksHex[]` pushed in order → `payloadsHex[]`, with `residualBytes` retained. |
| `error` | Expected failure during decode, as a neutral name (`payload-too-large`). |
| `finishError` | Expected failure when the stream ends (`truncated-stream`). |
| `why` | Why the case matters. A case that cannot explain itself gets deleted, not kept for coverage. |

Each implementation maps the neutral error names onto its own error values;
the JSON never names a Go or TypeScript identifier.

## Running them

```sh
go test ./conformance/...
```
