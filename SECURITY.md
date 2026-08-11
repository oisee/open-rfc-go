# Security policy

**open-rfc-go is not usable software.** It is a port in progress: at present it
frames and unframes network records and nothing more. There is no release, no
supported version, and no production use to secure. Do not deploy it.

## Reporting

Report suspected vulnerabilities privately through this repository's GitHub
Security Advisories ("Report a vulnerability"), not as a public issue.

If the issue is in the wire behaviour this project ports rather than in the
translation, it affects [`open-rfc`](https://github.com/marianfoo/open-rfc) too
and should go to that project's security policy. When in doubt, report to both
and say you have done so.

## Inherited boundary

Classic RFC over direct, SAProuter-routed, or message-server-selected transport
provides no transport encryption and no peer authentication. SAProuter is
routing, not confidentiality. Everything upstream says about that applies here
and is not repeated; treat the upstream `SECURITY.md` as the boundary
description, and this port as strictly narrower.

Do not send credentials or classic RFC traffic across an untrusted network.

## Two properties that differ from upstream

**Zeroing is weaker in Go.** Upstream zeroes retained buffers holding partial
frames because they can contain credential material. This port does the same
(`FrameDecoder.Reset`), but Go's garbage collector may have already copied
those bytes elsewhere during a heap move, and nothing in the language lets a
library find or clear those copies. The zeroing narrows the window; it does not
close it. Do not rely on it as a control.

**Data-race freedom is enforced, not asserted.** Upstream states ownership
invariants — one connection owns its socket and at most one in-flight call — in
prose. Here `go test -race` checks them mechanically. Every test run in CI uses
`-race` for that reason; a change that makes the race detector too slow to run
is a change that removes a security control.
