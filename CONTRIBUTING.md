# Contributing

open-rfc-go is a port of [`open-rfc`](https://github.com/marianfoo/open-rfc) to
Go. It is early: there is no client, no release, and no support boundary. See
[`docs/porting-plan.md`](docs/porting-plan.md) for what exists and what is next.

## Where a change belongs

**Wire behaviour belongs upstream.** If you have found something this port gets
wrong *and* upstream gets wrong, report it to open-rfc — that is where the
protocol knowledge lives, and a fix there reaches both implementations. Only
mistakes made in translation belong here.

If you are unsure which it is: does the TypeScript do the same thing? If yes,
it is upstream's. If no, it is ours.

## Before you open a pull request

- Read [`AGENTS.md`](AGENTS.md). It applies to humans too.
- Add the smallest failing test before the change, and confirm it fails without
  the change. A test that has never failed is not yet a test.
- Run `gofmt -l .`, `go vet ./...`, and `go test -race -shuffle=on ./...`.
- If you ported a file, add its SPDX provenance header and its row in
  [`docs/provenance.md`](docs/provenance.md).
- Report pass and fail counts from a run of the tree you are pushing.

## Developer Certificate of Origin

Every commit must be signed off, certifying [`DCO.md`](DCO.md):

```sh
git commit -s
```

## Licensing of contributions

Contributions are accepted under the Apache License 2.0, the license of this
repository and of the upstream work it derives from. Do not contribute code
adapted from another project unless you say so in the pull request and its
license permits redistribution under Apache-2.0 — in which case it needs an
attribution entry, as `docs/provenance.md` does for upstream.

Never commit credentials, customer data, network traces, or vendor binaries.
