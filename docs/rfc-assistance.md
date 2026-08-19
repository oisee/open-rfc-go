# RFC assistance: AI-assisted SAP debugging over classic RFC

Design / research note. Original work for open-rfc-go; it builds on the live
capabilities proven at milestone 5 (arbitrary read-only function-module calls
with runtime metadata discovery — see `docs/porting-plan.md` and
`internal/client/live_explore_test.go`) and the milestone-6 pool. Nothing here
requires the NW RFC SDK.

## The idea in one paragraph

The client can already dial a real system, discover any function module's
interface at runtime (`RFC_GET_FUNCTION_INTERFACE`), discover any structure's
layout (`RFC_GET_STRUCTURE_DEFINITION`), call the FM, and decode structures and
tables with no hardcoded interface. That is exactly the substrate an assistant
needs to *triage* problems: an SAP system publishes most of its diagnostic
state as function modules and dictionary tables, and classic RFC can read them.
"RFC assistance" is an agentic loop — **call a diagnostic FM, decode it with
runtime metadata, interpret it, decide the next call** — bounded to a read-only
whitelist, that turns a bare error (a dump ID, a failing BAPI, a red status in
SM58) into a cited root-cause hypothesis with the dump header, the source
around the failing line, the resolved message long-text, and the relevant table
rows attached.

The honest boundary up front: SAP exposes *metadata and structured tables*
cleanly over RFC, but several of the most useful diagnostic artefacts — short
dumps, application-log payloads, syslog, dev traces — are stored **compressed,
segmented, or in server-side files** and have **no standard RFC-enabled reader
that returns them formatted.** Where that is the case this note says so and
names the workaround (usually a small custom RFC wrapper installed on the
target, or a dictionary FM like `RPY_PROGRAM_READ` that *is* RFC-enabled).

---

## 1. Diagnostic surfaces reachable over classic RFC

Legend for the **Access** column:

- **RO** — read-only, non-sensitive; safe to surface to a model.
- **RO/sens** — read-only but the *content* is business data, PII, or
  security-relevant; must pass redaction before it leaves the triage boundary.
- **setup** — not reliably reachable as-is over classic RFC; needs a custom
  RFC-enabled wrapper on the target, or the data is compressed/segmented and
  must be reconstructed.

Authorizations are additive on top of **`S_RFC`** (required for *every* RFC
call; checked against the target's function *group*). Table reads additionally
check **`S_TABU_DIS`** (by table authorization group) or **`S_TABU_NAM`** (by
table name). Monitoring/kernel FMs typically add **`S_ADMI_FCD`** and/or
**`S_RZL_ADM`**.

### 1a. System identity and liveness (always safe, always first)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RFC_SYSTEM_INFO` | FM → `RFCSI` | System ID, host, kernel release, DB system, code page / Unicode flag, timezone, IP | RO | `S_RFC` |
| `RFC_PING` | FM | Liveness / round-trip only | RO | `S_RFC` |
| `RFC_GET_SYSTEM_INFO` variants | FM | Release/patch details | RO | `S_RFC` |

`RFC_SYSTEM_INFO` is proven live in `TestLiveSystemInfo`. It is the natural
first call of every session: kernel release and Unicode flag decide how later
output is decoded and which FMs even exist.

### 1b. Dictionary and interface metadata (the assistant's map)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RFC_GET_FUNCTION_INTERFACE` | FM | Parameter list (class I/E/C/T), types, lengths, exceptions of an FM | RO | `S_RFC` |
| `RFC_GET_STRUCTURE_DEFINITION` | FM | Field names, offsets, lengths, DDIC types of a structure | RO | `S_RFC` |
| `RFC_METADATA_GET` | FM | Batched interface + type metadata in one round trip | RO | `S_RFC` |
| `DDIF_FIELDINFO_GET` | FM | Full DDIC field info incl. data element, domain, conversion exit, F1 text | RO | `S_RFC` if released* |
| `RFC_READ_TABLE` on `DD03L`/`DD03T` | table | Table/structure field catalog, key flags | RO | `S_TABU_*` |
| `RFC_FUNCTION_SEARCH` | FM → `FUNCTIONS` | FM names matching a pattern + group | RO | `S_RFC` |

`RFC_GET_FUNCTION_INTERFACE` and `RFC_GET_STRUCTURE_DEFINITION` are proven live
(`TestLiveFunctionInterface`, `TestLiveStructure`); `RFC_FUNCTION_SEARCH` in
`TestLiveFunctionSearch`. *`DDIF_FIELDINFO_GET` is **not RFC-enabled on every
release** — do not assume it; the client already gets what it needs from
`RFC_GET_STRUCTURE_DEFINITION` plus `DD03L`, so treat `DDIF_FIELDINFO_GET` as a
richer-when-available extra, not a dependency.

### 1c. Generic table access (the workhorse — and its sharp edges)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RFC_READ_TABLE` | FM | Rows of any DB table/view, field-selected, WHERE-filtered | RO/sens | `S_RFC` + `S_TABU_DIS`/`S_TABU_NAM` |
| `RFC_GET_TABLE_ENTRIES` | FM | Similar, older; simpler filter | RO/sens | same |
| `BBP_RFC_READ_TABLE` | FM | `RFC_READ_TABLE` clone (same 512-byte limit) | RO/sens | same |

`RFC_READ_TABLE` is the single most important surface and is proven live
(`TestLiveReadTable`, against `T000`). Its constraints shape almost every
library decision below, so state them plainly:

- **512-byte row cap.** The `DATA` work area is `TAB512` — 512 bytes per row.
  Select a **field subset via the `FIELDS` table** for wide tables or the row
  silently truncates. There is no in-kernel wide successor over classic RFC;
  the "wide read table" FMs shipped by DS/BODS are add-ons, not guaranteed
  present.
- **`OPTIONS` WHERE lines are 72 characters.** The WHERE clause is passed as an
  `OPTIONS` table of 72-char lines and **a token must not be split across a
  line boundary** — a wrapper has to chunk on whitespace, not on byte 72.
- **Everything comes back as character.** Packed/decimal/date/time fields are
  returned formatted-as-text in the work area, sliced by the `FIELDS`
  offset/length. Numeric sign, decimal separator, and date format need
  re-parsing; the raw `WA` is *not* the internal representation.
- **No joins, no aggregation, limited operators.** It is `SELECT ... WHERE`,
  not SQL. `ROWSKIPS`/`ROWCOUNT` give paging.
- **Authority is coarse.** The check is `S_TABU_DIS` on the table's *auth
  group* (or `S_TABU_NAM`), not row-level. A read-only diagnostics role should
  grant only non-sensitive auth groups — see §3.

### 1d. Messages and long text (turning codes into English)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RFC_READ_TABLE` on `T100` | table | Message short text by `ARBGB` (class), `MSGNR`, `SPRSL` (lang) | RO | `S_TABU_*` |
| `T100T`, `T100A`, `T100X` | table | Message-class titles, self-explanatory flags | RO | `S_TABU_*` |
| `BAPI_MESSAGE_GETDETAIL` | FM | Full message text with variables substituted, plus long text | RO | `S_RFC` |
| `MESSAGE_TEXT_BUILD` | FM | Short text with `&1..&4`/`&` substitution | RO | `S_RFC` if released |

`T100` is the reliable base: `WHERE SPRSL = 'E' AND ARBGB = <class> AND MSGNR =
<nr>`. `BAPI_MESSAGE_GETDETAIL` is the better call when released, because it
does the placeholder substitution and returns the SE91 **long text** too.
Long-text-only (documentation) otherwise lives in `DOKHL`/`DOKTL`/`DOKIL` in a
compressed ITF form — reachable but not pleasant; prefer the BAPI.

### 1e. BAPI / function return payloads (the most common failure signal)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `BAPIRET2` / `BAPIRETURN` / `BAPIRET1` in any BAPI response | structure/table | `TYPE` (S/I/W/E/A), `ID`+`NUMBER`, `MESSAGE`, `MESSAGE_V1..V4`, `PARAMETER`/`ROW`/`FIELD` | RO* | `S_RFC` on that BAPI |
| ABAP exception / `SYSTEM_FAILURE` on the wire | RFC error envelope | Exception name, message key, ABAP short-text | RO | — |

*"RO" is about the *return structure*, not the BAPI: only ever **call BAPIs
that are themselves read-only** (getters, `..._GETDETAIL`, `..._GETLIST`).
Every BAPI hands back `BAPIRET2` rows whose `ID`+`NUMBER`+`MESSAGE_V*` feed
straight into the §1d resolver — this is the backbone of "why did my BAPI
fail." The RFC error envelope (already decoded by `internal/rfcerr`) carries
the same key when a call raises instead of returning.

### 1f. Short dumps — ST22 / `SNAP` (reachable, but the honest caveat)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RFC_READ_TABLE` on `SNAP` | table | Dump index + segmented content lines | **setup / RO/sens** | `S_TABU_*` (`SNAP` is a sensitive group) |
| `RFC_READ_TABLE` on `SNAPT` | table | Text elements | setup | `S_TABU_*` |

Honest assessment: a short dump is **stored across many `SNAP` rows** keyed by
`DATUM`, `UZEIT`, `AHOST`, `MANDT`, `UNAME`, `MODNO`, `SEQNO`, with the dump
body **segmented and compressed** into the content column. You can read the
index rows (when, who, which host, the exception class and the ABAP program) —
that alone is often enough to triage — but **reconstructing the fully formatted
ST22 dump from raw `SNAP` over classic RFC is not something SAP supports**, and
there is **no standard RFC-enabled FM that returns a formatted dump.** The
clean paths are: (a) read the `SNAP` *header* fields for triage and stop there;
(b) install a small custom RFC-enabled wrapper on the target that calls the
kernel dump reader and returns text; or (c) accept the header + the source
context obtained separately (§1g). Treat `SNAP` content as **sensitive** — it
can contain live variable values, including PII.

### 1g. Source location (this one *is* clean over RFC)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RPY_PROGRAM_READ` | FM → `SOURCE` | ABAP source lines of a program/include | RO/sens | `S_RFC` (+ `S_DEVELOP` on some systems) |
| `RFC_READ_REPORT` | FM | Report source lines | RO/sens | `S_RFC` |
| `RFC_READ_TABLE` on `TRDIR` | table | Program attributes (author, status, Unicode) | RO | `S_TABU_*` |

`REPOSRC` (the actual source table) stores **compressed** source and is *not*
usefully readable with `RFC_READ_TABLE`. Use the RFC-enabled readers instead:
given a program/include and a line number from the dump header, `RPY_PROGRAM_READ`
returns the surrounding source. Source is customer IP — treat as sensitive.

### 1h. Application log — SLG1 / BAL (index clean, payload not)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RFC_READ_TABLE` on `BALHDR` | table | Log headers: object/subobject, `EXTNUMBER`, user, time, `LOGNUMBER` | RO/sens | `S_TABU_*` |
| `BAL_DB_SEARCH` / `BAL_DB_LOAD` | FM | Find + load logs into memory | setup | `S_RFC` if released, + `S_APPL_LOG` |
| `RFC_READ_TABLE` on `BALDAT` | table | Message data | **setup** | `S_TABU_*` |

`BALHDR` gives a clean, filterable index (by object/subobject/date/user).
`BALDAT` stores the messages in a **compressed `RAWSTRING` (`BAPIRET`-like
blobs), not one row per message** — reading it raw over `RFC_READ_TABLE` yields
bytes you cannot decode without the BAL deserialization. The `BAL_DB_*` FMs are
the supported reader but are **not RFC-enabled by default** on most systems.
Realistic posture: use `BALHDR` for the index and message *counts by severity*;
getting individual message text needs the BAL FMs released (a small config
step) or a wrapper.

### 1i. Update task — SM13

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RFC_READ_TABLE` on `VBHDR` | table | Update request headers, status, user, time | RO/sens | `S_TABU_*` |
| `RFC_READ_TABLE` on `VBERROR` | table | Failed update entries + error message keys | RO/sens | `S_TABU_*` |
| `VBMOD`, `VBDATA` | table | Update module list; **serialized business payload** | RO/sens (payload) | `S_TABU_*` |

`VBHDR`/`VBERROR` triage cleanly: which update failed, for whom, and the
message key (→ §1d resolver). `VBDATA` holds the serialized function parameters
of the update modules — **business payload, high PII risk, and not trivially
decodable** — index it, don't dump it.

### 1j. tRFC / qRFC — SM58 / SMQ1 / SMQ2

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RFC_READ_TABLE` on `ARFCSSTATE` | table | tRFC call status, destination, time, retry | RO/sens | `S_TABU_*` |
| `ARFCSDATA` | table | Serialized tRFC payload | RO/sens | `S_TABU_*` |
| `TRFCQOUT` / `TRFCQIN` | table | qRFC outbound/inbound queue entries + status | RO/sens | `S_TABU_*` |
| `TRFC_QOUT_STATE` etc. | FM | Queue state (when released) | setup | `S_RFC` |

Status/index tables read cleanly and are excellent for "is my async call stuck
and why." Payload tables (`ARFCSDATA`) are serialized blobs — index only.

### 1k. Runtime / work-process / instance monitoring (`TH_*`, SM50/SM51/SM04)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `TH_WPINFO` | FM | Work-process list (SM50): status, action, running program/user | RO/sens | `S_RFC` + `S_ADMI_FCD` |
| `TH_SERVER_LIST` | FM | Application-server instances (SM51) | RO | `S_RFC` + `S_ADMI_FCD` |
| `TH_USER_LIST` / `TH_USER_INFO` | FM | Logged-on users, terminals (SM04/AL08) | RO/sens | `S_RFC` + `S_ADMI_FCD` |
| `TH_GET_VIRT_HEAP`, memory FMs | FM | Memory usage | RO | `S_RFC` + `S_ADMI_FCD` |

`TH_*` are kernel FMs; availability and RFC-callability vary by release and are
often locked down. When present they answer "is the system healthy right now"
and "what is that long-running user/program." User lists are PII (names,
terminals) — sensitive.

### 1l. System log — SM21

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| SM21 syslog | server-side binary `SLOG*` files | Kernel/dispatcher events, errors | **setup** | — |

Honest: the SM21 syslog is stored in **binary files on each application
server**, read by `RSLG*` programs, and there is **no dependable RFC-enabled
reader** exposed by default. Getting it over classic RFC means a custom
wrapper around the syslog read API, or an already-released `TH_`/`SXPG` path
that most systems do not open. Do not promise SM21 without confirming the
target has such a wrapper.

### 1m. RFC destinations, gateway, and traces (mostly *setup* or *sensitive*)

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `RFC_READ_TABLE` on `RFCDES` | table | SM59 destination config | **RO/sens (secrets)** | `S_TABU_*` + `S_RFC_ADM` |
| Gateway monitor (SMGW) | kernel/`GWY_*` | Connections, ACL, gateway trace | setup | `S_ADMI_FCD` |
| Dev traces (`dev_w*`, `dev_rd`, gw trace) | server files | Per-work-process traces, RFC/gateway trace | **setup** | — |

`RFCDES` can contain **stored destination credentials** (encrypted, but still
secret material and connection topology) — block or heavily redact. Gateway
and dev traces are **files / kernel monitor state**; not standard classic-RFC
surfaces. Assume "no" unless a wrapper exists.

### 1n. Performance / workload statistics — ST03 / STAD

| Surface | Kind | Yields | Access | Auth |
|---|---|---|---|---|
| `SAPWL_WORKLOAD_GET_STATISTIC` | FM | Aggregated workload (ST03) | setup | `S_RFC` + `S_ADMI_FCD` |
| `SWNC_*` collector FMs | FM | ST03N workload monitor data | setup | same |
| `SAPWL_STATREC_READ_FILE` | FM | STAD single statistical records | setup | same |

Useful for "why is this slow," but the `SAPWL_*`/`SWNC_*` FMs are **not
uniformly RFC-enabled**; verify per target. The aggregated `MONI` table is
readable via `RFC_READ_TABLE` but is a packed cluster — not directly usable.

### 1o. Blocklist (never call from the assistant)

Not diagnostic, or destructive, or a security exposure — the whitelist in §3
must exclude these and their families:

- `USR02` (password hashes), `USRPWDHISTORY`, `USR*` secret columns, `AGR_*`
  role assignments beyond what a triage needs → **security data**.
- `SXPG_CALL_SYSTEM`, `SXPG_COMMAND_EXECUTE` → OS command execution.
- `RFC_ABAP_INSTALL_AND_RUN`, `EDIT_REPORT`, generated-code executors → code
  execution.
- Any `*_COMMIT`, `*_CREATE`, `*_CHANGE`, `*_DELETE`, `*_POST`, `ENQUEUE_*`,
  `DEQUEUE_*`, `SYSTEM_RESET_RFC_SERVER` → state change.

---

## 2. A debug-trace assistant architecture

### 2.1 The loop

```
            ┌─────────────────────────────────────────────┐
            │  plan: what do I know, what do I need next?  │
            └───────────────────┬─────────────────────────┘
                                │  choose FM + args (whitelist only)
                                ▼
   pool.Lease ──► session.CallRaw(req) ──► rfcerr.Decode / classicrfc.DecodeResult
        ▲                                          │
        │                                          ▼
        │                          metadata repo: iface + struct-def
        │                          (RFC_GET_FUNCTION_INTERFACE,
        │                           RFC_GET_STRUCTURE_DEFINITION) — cached
        │                                          │
        │                                          ▼
        │                          structure.Decode(def, bytes) → typed rows
        │                                          │
        └──────────── interpret: extract keys ─────┤
                     (message id/nr, program, line, │
                      table+key, status flags)      │
                                                    ▼
                    stop when hypothesis is supported, or budget spent
```

Each turn is: **decide → call → decode-with-metadata → interpret → decide**.
The model (or a scripted planner) proposes the next FM *from the whitelist*;
the library executes it and hands back typed, decoded, redacted results; the
interpreter pulls out the keys that drive the next call. The metadata
repository makes the loop cheap — the first time it sees a structure it fetches
the definition, then serves it from cache for every later row of the same type.

### 2.2 Data flow over the existing library

The pieces already exist or are in flight:

- **`internal/pool`** leases a connection/session (one in-flight call per
  connection — the ownership invariant in `docs/architecture.md`). The assistant
  fans out across the pool for independent calls and stays sequential within one
  lease.
- **`cpic.EncodeCutFunctionRequest`** builds the call with `RequestedOutputs`
  (the milestone-5 wire fact: the server returns **only** the export/table
  parameters you name — so the assistant names exactly the fields it will read,
  which is also a natural data-minimization lever).
- **`session.CallRaw`** → **`classicrfc.DecodeResult`** splits scalars/tables;
  **`rfcerr`** decodes the error envelope when the call raises.
- **metadata repository runtime** (M6) caches interfaces and struct
  definitions; **`structure.Decode(def, row)`** yields `map[string]any`.

Nothing new at the protocol layer — RFC assistance is an *outer* layer over the
public API, plus the convenience wrappers in §4.

### 2.3 Worked triage: a failing BAPI

Input: "`BAPI_SALESORDER_CREATEFROMDAT2` failed."

1. **Capture the return.** The BAPI's `RETURN` (`BAPIRET2`) rows are decoded.
   One row: `TYPE=E ID=V1 NUMBER=849 MESSAGE_V1=<matnr> MESSAGE_V2=<plant>`.
2. **Resolve the message.** `BAPI_MESSAGE_GETDETAIL(ID=V1, NUMBER=849,
   V1..V4=...)` — or `RFC_READ_TABLE T100 WHERE SPRSL='E' AND ARBGB='V1' AND
   MSGNR='849'` — → "Material &1 not maintained for plant &2" → substituted
   long text.
3. **Confirm the data.** From the substituted variables, `RFC_READ_TABLE MARC
   WHERE MATNR='<matnr>' AND WERKS='<plant>'` (field subset) → zero rows →
   confirms the material/plant view is missing.
4. **Hypothesis.** "The order failed because the material-plant view (MARC) for
   `<matnr>`/`<plant>` does not exist; message V1 849. Create it (MM01, view
   Sales/Plant) or use a maintained plant." Cited: the `BAPIRET2` row, the T100
   text, the empty MARC read.

### 2.4 Worked triage: a short dump ID

Input: a dump timestamp/user, or "check ST22 for user X around 14:05."

1. **Index the dump.** `RFC_READ_TABLE SNAP` filtered by `DATUM`/`UZEIT`
   window and `UNAME=X`, selecting header fields (exception class, ABAP
   program, include, source line, category). *This is the reliably reachable
   part.*
2. **Fetch source context.** `RPY_PROGRAM_READ(include, from=line-10,
   to=line+10)` → the failing statement in context.
3. **Resolve any message.** If the dump carries a message key, §1d resolver.
4. **Optional deep body.** If a `SNAP`-reader wrapper is installed, pull the
   variable section; otherwise stop at header + source and **say so** — do not
   fabricate a body.
5. **Hypothesis** with the exception class, the source line, and (if available)
   the offending variable value, each cited to its call.

The pattern is the same everywhere: **an index/status surface that reads
cleanly → a message resolver → a targeted table or source read → a cited
hypothesis**, degrading gracefully to "header only" wherever the body is a
compressed/segmented surface (§1f/§1h/§1j).

---

## 3. Safety and scope

### 3.1 Read-only by default

- **Whitelist, not blocklist, at the core.** The assistant may only invoke FMs
  on an explicit allow-list (the RO rows of §1) plus `RFC_READ_TABLE` against an
  allow-listed set of tables. Everything else — including any unknown FM — is
  denied. A blocklist alone is unsafe: classic RFC has no "this call is
  read-only" flag, and a single synchronous call *can* commit.
- **No implicit state.** Never send an explicit `COMMIT`/BgRFC/transactional
  unit from the assistant path; keep calls stateless (the pool already isolates
  sessions).
- **Fail closed.** If metadata discovery shows an FM has changing parameter
  classes or an unfamiliar name pattern (`*_CHANGE`, `*_POST`, …), refuse.

### 3.2 Authorization (авторизация)

- **Dedicated least-privilege technical user**, never `SAP_ALL`. Build a
  "diagnostics reader" role: `S_RFC` limited to the function *groups* of the
  whitelisted FMs; `S_TABU_DIS`/`S_TABU_NAM` for **only the non-sensitive table
  authorization groups**; `S_ADMI_FCD` only if `TH_*`/monitoring is in scope.
- **Rely on SAP's own check as the backstop.** Even with the whitelist, the
  target enforces `S_RFC`/`S_TABU_*` — so a misconfigured whitelist cannot
  exceed the role. Defense in depth: whitelist *and* least-privilege role.
- **Per-target scoping.** Discover availability at runtime (`RFC_FUNCTION_SEARCH`,
  interface probes) rather than assuming; an FM released on one system may be
  absent or locked on another.

### 3.3 PII / secret redaction in traces

- **Column-level redaction in the decode path.** A redaction hook runs on
  `structure.Decode` output before results leave the boundary: known-secret
  columns (`USR02` hash fields, `RFCDES` credential fields, bank/card/tax
  fields, name/address) are dropped or masked; whole tables (`USR02`, payload
  blobs `VBDATA`/`ARFCSDATA`/`BALDAT`) are index-only or blocked.
- **Data minimization by construction.** Because the server returns only the
  `RequestedOutputs` named, the assistant fetches the *fewest* fields that
  answer the question — redaction is the second layer, minimization the first.
- **Redact before the model, not after.** Secrets must never enter the model
  context or logs; the trace the user sees is already redacted.
- **Dump/source content is sensitive.** `SNAP` bodies and `RPY_PROGRAM_READ`
  source can carry live values and customer IP — mark them, and let the caller
  gate whether they are shown.

### 3.4 Rate limiting and budgets

- **Per-triage call budget** (e.g. ≤ N FM calls, ≤ M rows total) so a loop
  cannot walk a whole table or spin.
- **`ROWCOUNT` always set** on `RFC_READ_TABLE`; paging is explicit, never
  "read all."
- **Concurrency bounded by the pool**, one in-flight per connection; a global
  calls/second cap protects a production target from a debugging session.
- **Timeouts via `context.Context`** on every call (the live tests already use
  a 30s context), and a wall-clock cap on the whole triage.

---

## 4. What the library would need to add (prioritized)

1. **`ReadTable` convenience wrapper — highest value.** Typed API over
   `RFC_READ_TABLE`: field selection, a **WHERE builder that chunks into 72-char
   `OPTIONS` lines without splitting tokens**, `ROWCOUNT`/`ROWSKIPS` paging, and
   **typed column decoding** that uses the returned `FIELDS` layout (auto-fetched
   struct def) to slice `WA` and re-parse packed/decimal/date/time from their
   character form. Detect and report 512-byte row truncation. Everything in §1
   leans on this.
2. **Message resolution helper.** `T100` lookup with `&`/`&1..&4` variable
   substitution rules, a `BAPI_MESSAGE_GETDETAIL` wrapper for long text, and
   **typed `BAPIRET2`/`BAPIRETURN`/`BAPIRET1` decoding with a severity model**
   (S/I/W/E/A). This turns raw return rows into cited English.
3. **Metadata repository runtime, reused as the assistant's cache** (already
   scheduled for M6): interface + struct-def caching with in-flight coalescing,
   exposed so triage never re-fetches a definition.
4. **Structured diagnostic error type.** Fold `internal/rfcerr` +
   `SYSTEM_FAILURE`/ABAP-exception + `BAPIRET2` into one typed error carrying
   `message_id`, `message_number`, and variables — so §4.2 resolution is
   automatic on any failure.
5. **Read-only whitelist + redaction hooks as first-class config.** A curated FM
   allow-list, a table allow-list with per-column redaction rules, and a decode
   hook that applies them — the enforcement point for §3.
6. **Batching / fan-out helper.** Sequential pipelining within one lease and
   fan-out across the pool for independent reads, with the per-triage budget and
   rate limit enforced centrally.
7. **Availability probe.** A cheap "is this FM RFC-callable here" check
   (interface probe / `RFC_FUNCTION_SEARCH`) so the assistant degrades
   gracefully per target instead of erroring mid-loop.

Items 1–2 are the minimum viable assistant substrate; 3–5 make it safe and
cheap at scale; 6–7 are optimization and portability.

---

## What is honestly *not* there without extra setup

To keep expectations calibrated:

- **Formatted ST22 dumps** — only `SNAP` header/index is reliable; the body is
  compressed/segmented with no standard RFC reader (§1f).
- **SM21 syslog** — binary server files, no dependable RFC reader (§1l).
- **Application-log message text** — `BALHDR` index is clean; `BALDAT` payload
  needs the `BAL_DB_*` FMs released or a wrapper (§1h).
- **Update/tRFC/qRFC payloads** — status tables are clean; `VBDATA`/`ARFCSDATA`
  are serialized blobs (§1i/§1j).
- **Dev/gateway traces, ST03/STAD workload** — files or non-uniformly-released
  FMs; verify per target (§1m/§1n).

The reliably-reachable core — `RFC_SYSTEM_INFO`, all metadata FMs,
`RFC_READ_TABLE` over dictionary/status/index tables, `T100` +
`BAPI_MESSAGE_GETDETAIL`, `BAPIRET2` returns, and `RPY_PROGRAM_READ` source —
is already enough to triage the most common failures end to end. The rest is a
matter of releasing a handful of FMs or installing one small custom wrapper on
the target, and this note names each case so the gap is a decision, not a
surprise.
