# Live test plan

Test scenarios for verifying an implementation against a real SAP system.

**This port cannot connect yet.** Milestone 5 is the earliest a call can
succeed. So this plan has two lives:

1. **Now** — run it against [`open-rfc`](https://github.com/marianfoo/open-rfc)
   (the TypeScript implementation), on a real system, to capture ground truth as
   conformance vectors. That gives the Go port an oracle *before* it is written,
   which is the difference between porting and guessing.
2. **At milestone 5** — the same scenarios become this port's acceptance suite,
   run against the same system, compared against the same recorded evidence.

Function-module names and error codes below are the ones this codebase actually
references; anything marked *(verify availability)* is conventional but not
referenced by the source, so confirm it exists on your target before relying on
it.

---

## Authentication: what is supported

**Exactly one mechanism: user and password.**

| Mechanism | State | Evidence |
|---|---|---|
| **User + password** | ✅ the only implemented path | `connection-parameters.ts:358-359` |
| SNC (`snc_mode`) | ❌ rejected with *"snc_mode connections are not implemented; use direct ashost/sysnr"* | `connection-parameters.ts:289-297` |
| WebSocket RFC (`wshost`) | ❌ rejected, same shape | same |
| x509 client certificate | ❌ does not exist in the tree | — |
| SSO / logon ticket (MYSAPSSO2) | ❌ does not exist | — |
| SAML / JWT / OAuth to ABAP | ❌ does not exist | — |
| Trusted-system RFC | ❌ does not exist | — |
| Principal propagation (`business_user_token`) | ⚠️ parameters recognized for route planners; outside the supported boundary | `connection-parameters.ts:64`, upstream `SECURITY.md` |

Password constraints, enforced **before** anything is sent
(`password-scramble.ts:16-27`):

| Constraint | Value |
|---|---|
| Character set | printable ASCII `0x20`–`0x7e` only |
| Maximum length | 40 bytes |
| Out-of-range input | refused locally as *"outside the proven ASCII baseline"* — never transcoded, never sent |
| On the wire | scrambled with a fixed 64-byte table plus a random 4-byte seed |

> ⚠️ **Scrambling is not encryption.** The table is published. Without SNC —
> which is not implemented — credentials cross the network protected only by the
> transport, which for plain TCP is nothing. Every scenario here assumes a
> trusted network segment to a non-production system. If you cannot assert that,
> do not run this plan.

`EnumSncQop` is exported for `node-rfc` API compatibility; importing it does not
widen the SNC boundary.

---

## Prerequisites

### System

| Requirement | Notes |
|---|---|
| A **non-production** SAP system | S/4HANA 2023 or NetWeaver 7.50 are upstream's stated targets |
| Application server host | `ashost` |
| System number | `sysnr`, two digits — determines the gateway port `3300 + sysnr` |
| Client | `client`, three digits |
| Network path to the **gateway** port | `3300 + sysnr`, *not* the dispatcher on `32NN` |

### The test user

A dedicated named user, not a shared or personal one, with:

- **`S_RFC`** authorization for exactly the function groups under test, and
  nothing wider. One scenario deliberately tests that a *denied* call fails
  correctly, which needs a second user or a removable authorization.
- Type **Dialog** or **Communication** — a Communication user cannot log into the
  GUI, which is the safer default for a test credential.
- A password inside the ASCII/40-byte constraint above. Choose one that also
  exercises a boundary: include a space (`0x20`) and a tilde (`0x7e`).

### Environment

```sh
export SAP_ASHOST=... SAP_SYSNR=00 SAP_CLIENT=100
export SAP_USER=... SAP_PASSWD=...
export SAP_LANG=EN
```

Run against the TypeScript implementation today with the quick-start script in
upstream's `README.md`.

---

## Safety and evidence rules

These are not optional, and one of them is a repository rule.

1. **Never commit a network trace, credential, or backend identity.** Both
   repositories forbid it. Evidence is a *derived, sanitized* statement of a
   structural fact — "the row was 404 bytes on this release" — never a capture.
2. **Conformance vectors use synthetic data.** If a real reply teaches you a
   structural rule, encode the rule with invented payload bytes. A vector must
   never contain customer data or a real hostname.
3. **Read-only function modules first.** Scenarios are ordered so that
   everything before L6 is read-only.
4. **Record the exact system release and kernel patch level** with every
   result. "Works on my system" is the symptom of the bug class this project
   exists to avoid; a result without a release label is not evidence.
5. **One finding per vector.** If a scenario teaches three rules, write three
   vectors.

---

## Scenario catalogue

Each scenario gives what it proves, how to run it, what passing looks like, what
a failure would mean, and which Go milestone it gates.

### L0 — Reachability

#### L0.1 Gateway port is reachable
- **Proves:** the port arithmetic and the network path.
- **Run:** TCP connect to `ashost:3300+sysnr`.
- **Pass:** connection established.
- **Failure means:** wrong port, firewall, or a **port-offset landscape** where
  the gateway is not at `3300+NN`. The latter is important — pinning that range
  is one of the six documented recurring bugs. Record the actual port.
- **Gates:** milestone 5.

#### L0.2 The dispatcher is a different port
- **Proves:** the gateway/dispatcher distinction empirically.
- **Run:** TCP connect to `3200+sysnr` and attempt an RFC handshake.
- **Pass:** it does **not** behave like a gateway.
- **Why:** documents the distinction with evidence instead of folklore.

### L1 — Handshake and capability negotiation

#### L1.1 Gateway accepts the normal-client record
- **Proves:** the 64-byte record, protocol version 2, request type 3.
- **Pass:** a gateway reply carrying accepted capability bits.
- **Evidence:** **record exactly which bits the system returns.**

#### L1.2 Required capabilities are present
- **Proves:** the fail-closed rule at `direct-cpic-session.ts:798-802`.
- **Pass:** `ExtendedInitOptions` and `CodePage` are both present.
- **Failure means:** a system this implementation cannot serialize for. It must
  refuse, not continue. **Do not "fix" this by relaxing the check.**
- **Evidence:** a capability-bit matrix per release — one of the most valuable
  artifacts this plan produces, since it is a per-release fact.

#### L1.3 APPC initialize is accepted
- **Proves:** the 373-byte initialize parameters and 341-byte extended options.
- **Gates:** milestone 2 vectors, milestone 5 end-to-end.

### L2 — Logon (the historically dangerous one)

This layer cost three weeks upstream. A parser pinned byte lengths in the logon
reply; it broke on any host with a different-length name, surfaced as
`RFC_INVALID_PROTOCOL`, and was misread as "SAP rejected the password" — so
working authentication code was replaced repeatedly. Every scenario here exists
to keep that from recurring.

#### L2.1 Valid credentials succeed
- **Pass:** logon completes; a subsequent `RFC_PING` returns.

#### L2.2 Wrong password is a **rejection**, not a protocol error
- **Run:** deliberately wrong password.
- **Pass:** the error is a logon failure (`RFC_LOGON_FAILURE`), reaching the
  caller as a rejection with the backend's reason.
- **Failure means:** if this surfaces as `RFC_INVALID_PROTOCOL`, the reply
  parser is pinning a layout. **This is the exact three-week bug.** Stop and fix
  the parser.
- **Gates:** milestone 5. Highest-value negative test in the plan.

#### L2.3 Locked / expired / unknown user, and wrong client
- **Run:** each independently.
- **Pass:** each is a logon rejection carrying the backend's reason, distinct
  from a protocol fault.
- **Evidence:** the status value per case, per release.

#### L2.4 Host-name length independence
- **Proves:** the original defect directly.
- **Run:** connect via names of clearly different lengths for the same host
  (short alias, FQDN, IP literal).
- **Pass:** identical logon behaviour for all three.
- **Failure means:** something is pinning a text-field length again.

#### L2.5 Password boundaries
- **Run:** a 40-byte password; one containing `0x20` and `0x7e`; and locally, a
  41-byte and a non-ASCII password.
- **Pass:** the first two authenticate; the last two are **refused locally**
  and never reach the network.
- **Evidence:** confirm the rejection happens client-side — no TCP payload.

### L3 — First call

#### L3.1 `RFC_PING`
- **Proves:** the smallest complete call round trip.
- **Gates:** milestone 5 — this is the milestone-5 definition of done.

#### L3.2 `STFC_CONNECTION` echo
- **Run:** call with `REQUTEXT` and read `ECHOTEXT` / `RESPTEXT`.
- **Pass:** the echoed text matches exactly.
- **Also proves:** parameter names are case-sensitive — `REQUTEXT`, never
  `requtext`.

#### L3.3 Non-ASCII round trip
- **Run:** `REQUTEXT` containing non-ASCII (accented Latin, CJK, an emoji).
- **Pass:** the echo is byte-identical after decoding.
- **Failure means:** a codepage or UTF-16 conversion defect. Record which
  characters survive — this is release- and codepage-dependent.

### L4 — Metadata bootstrap

#### L4.1 Which metadata path the system offers
- **Run:** attempt `RFC_METADATA_GET`; observe whether it exists.
- **Pass:** either it works, or the fallback to `RFC_GET_FUNCTION_INTERFACE` /
  `DDIF_FIELDINFO_GET` engages cleanly.
- **Evidence:** availability per release.

#### L4.2 **`RFC_FUNINT` row length — the headline measurement**
- **Proves:** the lower-bound-plus-window rule.
- **Run:** capture the actual Unicode row length returned.
- **Pass:** rows **≥ 402** bytes are accepted; anything beyond the stable prefix
  is ignored.
- **Evidence:** the exact length on each release. Upstream states 402 with a
  404-byte profile already evidenced. **If you observe a third value, that is a
  significant finding for upstream** and belongs in an issue.
- **Failure means:** a decoder comparing for equality — the recurring bug.

#### L4.3 `RFC_FIELDS` row length
- Same procedure, same reasoning. Upstream's history pinned 138.

#### L4.4 A structure with a DDIC built-in `COMPTYPE`
- **Proves:** the fix for the `COMPTYPE = "E"` pin, one of the six.
- **Run:** metadata for a structure whose component uses a built-in type.
- **Pass:** accepted.

### L5 — Value codecs

Call a function module returning each family and compare decoded values against
the backend's own display of the same data.

| Family | What to exercise |
|---|---|
| Packed decimal / BCD | positive, negative, zero, maximum scale |
| Decimal float (DECF16/34) | boundary exponents, negative zero |
| Compact temporal | `UTCLONG`, `DTDAY`, `CDAY` at minimum and maximum; the **1582 calendar gap**; an ISO **week 53** |
| Character | trailing-space semantics, fixed-width padding |
| Integers | `INT1`/`INT2`/`INT4`/`INT8` boundaries |
| Tables | empty, one row, many rows |
| Strings / XSTRING | empty, and a long value |

*(verify availability)* `STFC_STRUCTURE` is the conventional test module for
mixed scalar types; confirm it exists on your target, or use any read-only
module in scope whose interface covers the families above.

- **Evidence:** every case that is pure byte-in/byte-out becomes a conformance
  vector — with **synthetic** payload bytes encoding the learned rule.

### L6 — Size, fragmentation, and limits

#### L6.1 The 28,000-byte APPC fragment boundary
- **Run:** a table payload just under, exactly at, and just over
  `MAX_APPC_APPLICATION_DATA_FRAGMENT_LENGTH`.
- **Pass:** all three round-trip identically.
- **Why:** off-by-one at a fragment boundary is a classic defect and invisible
  below the boundary.

#### L6.2 The 1.4 MB default outgoing limit
- **Run:** a payload above `DEFAULT_MAX_APPC_OUTGOING_MESSAGE_LENGTH`.
- **Pass:** a clear, bounded local error — not a hang, not a truncated send.

#### L6.3 Many fields
- **Run:** approach `DEFAULT_MAX_CPIC_FIELD_COUNT` (100,000).
- **Pass:** bounded behaviour, clear error at the limit.

### L7 — Errors and failure classification

Each must produce its own distinguishable error, not a generic one.

| Scenario | Expected |
|---|---|
| Non-existent function module | `RFC_FUNC_ERROR` |
| ABAP exception raised by the module | `RFC_ABAP_EXCEPTION`, carrying the exception name |
| ABAP runtime failure (short dump) | `RFC_ABAP_RUNTIME_FAILURE` |
| Wrong parameter name or type | `RFC_INVALID_PARAMETER` |
| `S_RFC` denies the call | authorization failure, distinct from "not found" |
| Unknown DDIC type | `RFC_DD_ERROR` |
| Peer closes mid-call (kill the connection) | `RFC_COMMUNICATION_FAILURE` |
| Server-side timeout | `RFC_TIMEOUT` |

- **Failure means:** any collapse into a generic error loses the caller's ability
  to decide whether to retry — which is a correctness bug, not a cosmetic one.

### L8 — Cancellation, ambiguity, and state

#### L8.1 Cancel an in-flight call
- **Run:** cancel during a long-running module.
- **Pass:** `RFC_CANCELED`; the connection is closed and not reused; no replay.

#### L8.2 **The ambiguous send**
- **Proves:** the rule that outranks every other retry consideration.
- **Run:** drop the network mid-send (e.g. a firewall rule) after bytes have
  gone out.
- **Pass:** the failure is reported as **unknown outcome**, the transport is
  closed, and **the call is never re-sent**.
- **Failure means:** a retry that commits an ABAP LUW twice. There is no safe
  version of this bug.

#### L8.3 Stateful session context
- **Run:** two calls in one context that share ABAP session state.
- **Pass:** state is visible to the second call, and gone after reset.

#### L8.4 Transactions
- **Run:** a change in a module followed by `BAPI_TRANSACTION_COMMIT`; separately
  by `BAPI_TRANSACTION_ROLLBACK`.
- **Pass:** committed and rolled back respectively, verified in the backend.
- ⚠️ First scenario in this plan that **writes**. Non-production only, with
  data you created.

#### L8.5 Pool and reconnect
- **Run:** exhaust the pool; take a connection past the backend's idle timeout;
  restart the SAP instance mid-test.
- **Pass:** clean errors, correct reconnection, no leaked connections.

#### L8.6 Soak
- **Run:** sustained calls for several hours.
- **Pass:** stable memory, no descriptor leak, no degradation.
- **Evidence:** where upstream's roadmap gate `V1-04` gets its data.

---

## Landscape matrix

The same scenarios on more than one system is where per-release facts appear.

| Axis | Why it matters |
|---|---|
| S/4HANA 2023 **and** NetWeaver 7.50 | upstream's stated targets; L1.2 and L4.2 differ across them |
| Two different `sysnr` values | proves the port arithmetic rather than one lucky default |
| A port-offset landscape, if you have one | the case that broke the pinned dispatcher range |
| Different-length host names | L2.4, the original three-week defect |
| Unicode system | the only kind targeted; record the codepage |

---

## Milestone gating

| Milestone | Scenarios that must pass |
|---|---|
| 2 — framing, APPC, CPIC, gateway | L1.1–L1.3 as recorded vectors |
| 3 — value codecs | L5 vectors |
| 4 — metadata | L4.1–L4.4 |
| **5 — first live call** | L0, L1, L2, L3.1, L3.2 |
| 6 — pool, contexts, transactions | L6, L7, L8 |

L2.2 and L8.2 are the two that must never be waived. One is the historical
three-week bug; the other can double-commit a business transaction.

---

## Out of scope

Not tested because not implemented: SNC, WebSocket RFC, x509, SSO tickets,
SAML/JWT, trusted-system RFC. If your landscape *requires* SNC, this
implementation cannot connect to it at all, and no scenario here changes that.

Message-server and SAProuter routes are preview-only upstream and outside this
port's plan until milestone 6.
