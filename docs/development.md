# Development

## Setup

Go 1.26 or later. No third-party dependencies, and adding one requires a stated
reason and a second proven consumer.

Keep an `open-rfc` checkout **beside** this one. It is not optional: the porting
rules require reading the upstream file before writing the Go file, and the drift
check assumes the sibling path.

```sh
git clone https://github.com/marianfoo/open-rfc      ~/dev/open-rfc
git clone https://github.com/oisee/open-rfc-go       ~/dev/open-rfc-go
cd ~/dev/open-rfc-go && go build ./...
```

## The loop

```sh
gofmt -l .                          # must print nothing
go vet ./...
go test -race -shuffle=on ./...
```

Run one test while iterating:

```sh
go test ./internal/ni -run TestDecodesFragmentedAndCoalescedFrames -v
```

Run the wire vectors alone:

```sh
go test ./conformance/... -v
```

Coverage for a package you are working on:

```sh
go test -cover ./internal/ni
go test -coverprofile=cover.out ./internal/ni && go tool cover -html=cover.out
```

### `-race` is a security control

Upstream states its ownership invariants in prose: one connection owns its socket
and at most one in-flight call. Here they are enforced by the race detector,
which is why every test run — local and CI — uses `-race`. A change that makes
the detector too slow to run is a change that removes a control, not a
performance win.

`-shuffle=on` catches order dependence between tests, which matters for a
codebase full of stateful decoders.

## Fuzzing

**Every decoder that consumes network bytes gets a fuzz target.** Upstream cannot
have these; Go gives them away, and a length-prefixed binary protocol is exactly
what they are for.

```sh
go test ./internal/ni -run TestX -fuzz FuzzFrameDecoderBounds -fuzztime 60s
go test ./internal/ni -run TestX -fuzz FuzzFrameRoundTrip     -fuzztime 60s
```

`-run TestX` matches no test, so only the fuzz target runs.

Assertions in a fuzz target are necessarily weak — for arbitrary input there is
no expected output — so assert **bounds and invariants**: nothing panics, no
decoded payload exceeds the configured limit, buffered bytes never exceed bytes
pushed, and the error is one of the declared sentinels.

A crash lands in `testdata/fuzz/`. Do not commit the corpus entry alone: turn it
into a named regression test that states the invariant it broke, then commit
both.

## Adding a ported file

1. **Read the upstream file.** Not the surface inventory — the file. The
   inventory is a map, and maps have errors; one inventory's own verification
   pass corrected four of its counts.
2. **Read [`recurring-bug-class.md`](recurring-bug-class.md)** if you are writing
   a decoder. Every time.
3. **Write the provenance header:**
   ```go
   // SPDX-License-Identifier: Apache-2.0
   //
   // Ported from open-rfc src/protocol/<file>.ts at commit <sha>,
   // Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
   // Modified by open-rfc-go contributors: <what changed>.
   // See docs/provenance.md.
   ```
   Record the commit you actually read, not `main`.
4. **Port the tests**, including malformed-input and boundary cases. A case with
   no Go analogue goes in `docs/provenance.md` with the reason — never dropped
   silently.
5. **Add what upstream cannot have:** an aliasing test wherever caller memory is
   retained or returned, and a fuzz target for a decoder.
6. **Add a row to [`provenance.md`](provenance.md).**
7. **Extract conformance vectors** for pure byte-in/byte-out behaviour.

## Translation reference

| TypeScript | Go |
|---|---|
| `Buffer`, `Uint8Array` | `[]byte` |
| `async`/`Promise` | blocking call plus `ctx context.Context` first parameter |
| `AbortSignal` | `context.WithCancel`, `conn.SetDeadline` |
| `AsyncLocalStorage` | explicit handle or `context.Context` value — never implicit |
| `#private` field | unexported field |
| `throw new RangeError(...)` | wrapped sentinel, matched with `errors.Is` |
| enum with numeric values | `type X int32` plus a generated `String()` |
| `Object.freeze` | return by value; do not hand out a mutable pointer |
| `EventEmitter` | `log/slog` handler, or a bounded channel |
| `BigInt` | `int64`/`uint64`, or `math/big` when the range genuinely needs it |
| `Buffer.from(chunk)` | `make` + `copy` — Go slices alias, JS `Buffer.from` does not |
| `chunk.fill(0)` | `clear(chunk)` — weaker in Go; see [`../SECURITY.md`](../SECURITY.md) |
| typed-array geometry guards | nothing; the attack does not exist in Go |

## Error conventions

Wrapped sentinels, never message matching:

```go
var ErrPayloadTooLarge = errors.New("ni: advertised payload exceeds the configured limit")

return nil, fmt.Errorf("%w: %d exceeds %d", ErrPayloadTooLarge, advertised, limit)
```

Callers use `errors.Is(err, ni.ErrPayloadTooLarge)`. Message text is for humans
and may change; the sentinel is the contract.

Prefix messages with the package name (`ni:`), and include the values that made
it fail — an error saying which length exceeded which limit is worth three saying
"invalid frame".

## Conformance vectors

Wire facts belong in `conformance/testdata/vectors/`, not inside test code, so
another implementation can check the same bytes. Format and the intended move
upstream: [`../conformance/README.md`](../conformance/README.md).

Every vector states **why** it matters. If you cannot write the reason, you have
an observation rather than a rule — which is the recurring bug in its larval
form.

## Documents containing escape sequences

Markdown here quotes things like `\u0000` as *text*. Tools that write files can
convert that text into the actual byte, silently. It has happened twice in this
repository, once in a hash domain separator where the exact bytes are
load-bearing.

After editing any document that quotes escape sequences:

```sh
grep -aP '[\x00-\x1f\x7f]' docs/**/*.md      # must match nothing
```

`grep` without `-a` reports "binary file matches" and tells you nothing useful.
If a normal edit will not stick, build the replacement from `chr(92)` in a script
rather than typing the backslash.

## CI

`.github/workflows/ci.yml` runs on pull requests and pushes to `main`:
`gofmt` check, `go vet`, `go build`, `go test -race -shuffle=on`, and a 60-second
fuzz run per decoder target.

Actions are pinned by commit SHA, matching upstream's practice — a tag is
mutable. `actions/setup-go` currently carries a `TODO` to pin it; that must be
resolved before the repository is made public.

## Checklist before a pull request

- [ ] `gofmt -l .` prints nothing
- [ ] `go vet ./...` clean
- [ ] `go test -race -shuffle=on ./...` passes; report pass and fail counts
- [ ] New test seen to fail without the change
- [ ] Provenance header present; `provenance.md` row added
- [ ] Aliasing test where caller memory is retained or returned
- [ ] Fuzz target for a new decoder, run at least 60s
- [ ] Conformance vectors for new byte-in/byte-out behaviour
- [ ] Commit signed off (`git commit -s`)
