# open-rfc-go contributor guide

This guide applies to maintainers, contributors, and automated coding agents.

This repository is a **port** of [`open-rfc`](https://github.com/marianfoo/open-rfc),
not an independent implementation. That fact changes how you work here.

## Before changing anything

Read, in this order:

1. [`docs/recurring-bug-class.md`](docs/recurring-bug-class.md) — the mistake
   the upstream codebase made six times. A translation is an unusually easy way
   to make it a seventh: a decoder that pins a length because the TypeScript
   pinned a length is the same bug with new syntax.
2. [`docs/architecture.md`](docs/architecture.md) — layer boundaries, ownership
   invariants, the implementation ladder, and the evidence hierarchy.
3. [`docs/provenance.md`](docs/provenance.md) — what came from where.

## Porting rules

- **Read the upstream file before writing the Go file.** Keep an `open-rfc`
  checkout alongside this one. Port from the source, not from memory or from
  another port.
- **Port the tests too**, including malformed-input and boundary cases. If a
  case has no Go analogue, record it in `docs/provenance.md` with the reason —
  do not drop it silently.
- **Add what upstream cannot have**: a slice-aliasing test wherever caller
  memory is retained or returned, and a fuzz target for every decoder that
  consumes network bytes.
- **Every ported file carries an SPDX header** naming the upstream file and the
  exact commit, and a line in `docs/provenance.md`. This is an Apache-2.0 §4(b)
  obligation and the index used to propagate upstream fixes.
- **Do not improve the protocol on the way through.** If upstream looks wrong,
  that is an upstream issue, not a local fix — a divergence nobody recorded is
  worse than a shared bug. Fix it upstream, then port the fix.
- **`src/compat/` is out of scope.** It exists to match an npm package's
  behaviour and has no Go consumer.

## Go rules

- Standard library only. A dependency needs a stated reason and a second
  proven consumer.
- Everything under `internal/` until a public API is designed on purpose.
- Errors are wrapped sentinels (`errors.Is`), never matched on message text.
- Anything holding network bytes gets an explicit bound, and the bound is a
  configured limit — never a claim about what the protocol permits.
- Comments explain why an invariant exists, not what the code does.

## Build and test

```sh
go build ./...
go test -race -shuffle=on ./...
gofmt -l .          # must print nothing
go vet ./...
```

Run the fuzz targets when touching a decoder:

```sh
go test ./internal/ni -run TestX -fuzz FuzzFrameDecoderBounds -fuzztime 60s
```

Linting is currently `gofmt` plus `go vet`. Adding `golangci-lint` with a
pinned version is open work; do not add a config file that CI does not run.

## Commits

Sign off every commit (`git commit -s`) as required by [`DCO.md`](DCO.md).
