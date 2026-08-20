# Write-FM safety classification for `rfc-mcp`

Design report (2026-08-20). A model — and a declarative file format — for deciding
which RFC function modules an MCP server may call, and under what conditions, so
that exposing a live SAP system to an AI assistant is safe by construction rather
than by operator vigilance alone.

This is the concrete design behind the roadmap's **"Safety gate for write FMs in
`rfc-mcp` beyond `--read-only`"** (P1, `docs/roadmap.md`). It builds on the
[MCP → RFC bridge](../mcp-rfc-bridge.md) tool surface and reuses the runtime
metadata already available through the `rfc` package. It changes **no wire
behaviour**; it is a policy layer above the call boundary.

> **Scope of this doc.** A classification model, its signals, a declarative
> policy format with a schema, the `rfc mcp` integration, a build/maintenance
> workflow, and the honest limits. It does not propose Go changes here — it
> specifies what those changes must implement.


## 1. Problem statement

Classic RFC has **no machine-readable side-effect declaration.** A function
module's name, its interface, and (if you fetch it) its ABAP source are all the
signal there is. Nothing on the wire says "this call mutates the database."

Meanwhile the blast radius is real. An RFC-enabled FM can create, change, or
delete business data, unlock or lock users, post documents, and — with a
following `BAPI_TRANSACTION_COMMIT` — make those changes durable. An assistant
that can call arbitrary FMs can do arbitrary damage.

Today's control is coarse. `rfc mcp --read-only` drops the generic
`rfc_call` tool, but **auto-exposed per-FM tools bypass it** — the operator
curated them, so the flag trusts them. That is too blunt in both directions: it
forbids reads through the generic path while permitting writes through the
exposed path.

The hard part is that **many FMs are dual read/write depending on their
parameters.** Three categories:

**a. Fully safe — read-only regardless of arguments.** These can be called with
any inputs and never mutate state.

- `RFC_READ_TABLE` — read rows of a table/view.
- `RFC_SYSTEM_INFO` — backend identity.
- `BAPI_USER_GET_DETAIL` — read a user's attributes.
- The `*_GETLIST` / `*_GETDETAIL` / `*_GET_LIST` BAPI families — pure queries.
- `RFC_FUNCTION_SEARCH`, `RPY_FUNCTIONMODULE_READ` — metadata reads.

**b. Fully dangerous — mutate, for any argument that does real work.**

- `BAPI_USER_CREATE1` — create a user.
- The `*_CREATE` / `*_CHANGE` / `*_UPDATE` / `*_DELETE` / `*_POST` families.
- `SET_*` / `*_MODIFY` direct updates.
- `BAPI_TRANSACTION_COMMIT` — the durability switch itself. It writes nothing on
  its own but makes every pending BAPI write permanent. It is a special case
  (§4): blocked by default even under a write-enabled policy.

**c. Conditional / dual-use — safe or dangerous depending on a parameter value.**
This is the category a blanket switch cannot express.

- A **test/simulate flag**: an FM that runs a full posting *simulation* when
  `TEST_RUN = 'X'` (or `SIMULATE`, `NO_COMMIT`, `CHECK_ONLY`) and commits real
  changes when it is blank. Safe only when the flag is set.
- An **action/mode selector**: an FM whose `ACTION = 'R'` (or `READ`, `DISPLAY`,
  `SHOW`) reads, while `ACTION` in `{I, U, D, SAVE}` inserts / updates / deletes.
  One FM, one interface, opposite side effects chosen at call time.
- The classic BAPI shape: the FM itself only *stages* a change in the update
  task; nothing is durable until a separate `BAPI_TRANSACTION_COMMIT`. So a
  create BAPI called *without* a following commit is reversible — which the
  connection-level guard in §4 exploits.

A safety model has to cover all three, default to caution on the unknown, and
stay reviewable by a human who knows the system.


## 2. Signals — sources of truth, ranked by cost and reliability

No single signal is authoritative. The model **combines** them, cheapest first,
and records which signal decided each verdict so a reviewer can audit it. Ranked
by cost (low → high) against reliability:

### (a) Name heuristics — free, weak, high recall

The FM name is available from `RFC_FUNCTION_SEARCH` / `TFDIR` without a second
call. Tokenize the name and match verbs:

- **Read verbs** (lean safe): `GET`, `GETLIST`, `GET_LIST`, `GETDETAIL`,
  `GET_DETAIL`, `READ`, `LIST`, `SEARCH`, `FIND`, `SHOW`, `DISPLAY`, `CHECK`,
  `EXISTS`, `VALIDATE`, `SELECT`, `QUERY`, `INFO`, `SIMULATE`, `GET_STATUS`.
- **Write verbs** (lean dangerous): `CREATE`, `CHANGE`, `UPDATE`, `DELETE`,
  `POST`, `SET`, `MODIFY`, `INSERT`, `SAVE`, `COMMIT`, `LOCK`, `UNLOCK`,
  `CANCEL`, `REVERSE`, `CONFIRM`, `RELEASE`, `ADD`, `REMOVE`, `WRITE`, `MAINTAIN`.

This is a **triage signal, never a verdict.** `CHECK` can persist; `SIMULATE` in
the name does not guarantee simulation in the body; a `Z_`-namespace FM can be
named anything. Use it to sort FMs into "probably safe", "probably dangerous",
and "ambiguous" so the more expensive signals are spent where they matter.

### (b) Interface shape — one metadata fetch, moderate reliability

`Client.FunctionInterface` (cached) gives the IMPORTING / EXPORTING / CHANGING /
TABLES parameters and their types. Several shapes correlate with mutation:

- An **EXPORTING `RETURN` of type `BAPIRET2`** (or a `RETURN` TABLES parameter of
  `BAPIRET2` / `BAPIRETURN`) is the BAPI convention — present on most write BAPIs
  (and on some read ones, so it is a flag, not a proof).
- **CHANGING parameters** imply the FM mutates what it is handed; combined with a
  write verb this strengthens the "dangerous" prior.
- A **TEST / SIMULATE / NO_COMMIT / CHECK_ONLY import** is the tell for a
  conditional FM — its presence is exactly what a conditional policy rule keys
  on. Detecting it lets the classifier *promote* an FM from "dangerous" to
  "conditionally safe".
- An **ACTION / MODE / ACTIVITY import** typed as a short CHAR, especially with
  documented domain fixed values, is the other conditional shape.

The interface never *proves* read-only (an FM with only IMPORTING scalars can
still `UPDATE` a table internally), but it reliably surfaces the *conditional*
levers and the BAPI write convention.

### (c) The commit pattern — a runtime guard, not a static signal

Standard BAPIs do not persist until `BAPI_TRANSACTION_COMMIT` runs **on the same
connection**. This yields a *dynamic* safety property the server can enforce
without classifying anything: track, per connection, whether any call that could
have staged a write has happened but not yet been committed — a **"dirty
connection" guard** (§4). It converts "did a write become durable?" from a
classification question into a bookkeeping one.

### (d) Deep-dive: fetch the source, classify with an LLM

For FMs the cheap signals leave ambiguous, fetch the **ABAP source** via
`RPY_FUNCTIONMODULE_READ` together with the interface, and ask an LLM to classify
side effects: does the body `INSERT`/`UPDATE`/`MODIFY`/`DELETE` a database table,
`CALL FUNCTION ... IN UPDATE TASK`, `COMMIT WORK`, or invoke a known write BAPI?
Which parameter values gate those paths?

This is the most reliable signal and the most expensive, and it must never be
trusted blind. The pipeline is:

```
ambiguous FM
  │
  ├─ fetch interface (Client.FunctionInterface)          ── cached
  ├─ fetch source    (RPY_FUNCTIONMODULE_READ)           ── cached
  │
  ├─ LLM classification ──▶ proposed verdict + rationale + the
  │                          parameter conditions it depends on
  │
  ├─ emit a declarative policy entry (draft, marked  source: llm,
  │        reviewed: false )
  │
  ├─ HUMAN SIGN-OFF ──▶ reviewer flips reviewed: true (or edits/rejects)
  │
  └─ cache the signed-off entry in the versioned policy file
```

The LLM's output is a **reviewable proposal in the declarative format below**,
never a live decision in the call path. Nothing an LLM classified reaches
"callable" without a human flipping `reviewed: true`. This keeps the fast path
deterministic and auditable, and confines model fallibility to a review queue.

### (e) Curated lists — SAP documentation and community knowledge

SAP publishes released-BAPI and RFC catalogues; the `*_GETLIST`/`*_GETDETAIL`
naming is a documented convention; some FMs have well-known verdicts. Seed the
policy file's allowlist/denylist from these. Highest reliability where it exists,
zero coverage where it does not — a starting corpus, not a substitute for the
signals above.

### Signal precedence

When signals disagree, the model resolves in this order (most authoritative
first): **human-reviewed policy entry** → **curated list** → **LLM deep-dive
(reviewed)** → **interface shape** → **name heuristic** → **default**. The
default for an unmatched write-capable or unknown FM is **deny** (§3, §6).


## 3. The declarative safety policy format

A versioned file, YAML (authoring) or JSON (equivalent, machine-generated),
carried in-repo and overridable per system. It expresses a safe allowlist, a
dangerous denylist, and **conditional rules keyed on parameter values.**

### 3.1 Schema (informal)

```
version: 1                       # schema version, required
metadata:
  name: string                   # human label for this policy
  generated: date                # when last built/refined
  system: string | "*"           # which backend(s) this applies to; "*" = any

defaults:
  unknown: deny | ask            # verdict for an FM matched by no rule (default: deny)
  read_verbs_safe: bool          # allow name-heuristic "safe" verbs without an
                                 #   explicit entry (default: false — explicit only)

# Order of evaluation for a given FM name:
#   1. deny  (a match here is final — deny wins over everything)
#   2. rules (first matching conditional rule decides)
#   3. safe  (an allowlist match → safe)
#   4. mask rules (namespace/pattern fallback)
#   5. defaults.unknown

safe:                            # fully-safe allowlist — callable with any args
  - FM_NAME
  - FM_NAME: { note: string }    # optional annotation

deny:                            # fully-dangerous denylist — never callable
  - FM_NAME
  - FM_NAME: { reason: string }

rules:                           # conditional, per-parameter
  FM_NAME:
    safe_when:      { PARAM: value | [values] , ... }   # ALL must match → safe
    dangerous_when: { PARAM: value | [values] , ... }   # ANY match → dangerous
    else: safe | dangerous | ask                        # neither matched
    require_confirm: bool        # even when "safe", require a confirm step
    note: string
    source: heuristic | interface | llm | curated | human
    reviewed: bool               # true only after human sign-off

masks:                           # namespace / pattern fallback, lowest priority
  - match: "GLOB"                # e.g. "*_GETLIST", "Z_RO_*", "/ACME/*"
    verdict: safe | dangerous | ask
    note: string
```

**Value matching.** Parameter comparisons are on the *string* form of the value
after the server's own upper-casing/padding rules for CHAR fields; a scalar `"X"`
matches a checkbox flag, a list matches any-of. A `safe_when` with multiple
parameters requires **all** to match (AND); `dangerous_when` fires if **any**
listed condition matches (OR). `dangerous_when` is evaluated before `safe_when`
so an explicit danger cannot be masked by a coincidental safe match.

**Missing parameter.** If a rule references a parameter the caller omitted, treat
it as **not matching `safe_when`** (fail safe) and, for `dangerous_when` on a
defaulted write parameter, consult `else`. When in doubt the verdict falls
through to `else`, whose own default is `dangerous`.

### 3.2 How masks and namespaces interact

Masks are the **lowest-priority** tier, applied only when no `deny`, `rules`, or
`safe` entry names the FM. They let an operator say "everything in my read-only
`Z_RO_*` namespace is safe" or "all `*_GETLIST` are safe" without enumerating
each FM, while still letting a specific `deny` entry carve out an exception.
Masks compose with `rfc mcp`'s existing `--expose` / `--hide` masks but are
**orthogonal**: `--expose`/`--hide` decide *which FMs become tools*; the policy
masks decide *whether a tool is callable and how*. An FM must survive both — be
exposed **and** be permitted — to be invoked. Deny always wins over any mask.

### 3.3 Worked examples

```yaml
version: 1
metadata:
  name: baseline
  generated: 2026-08-20
  system: "*"

defaults:
  unknown: deny
  read_verbs_safe: false

safe:
  - RFC_READ_TABLE                       # read rows; still subject to table/row caps
  - RFC_SYSTEM_INFO
  - RFC_FUNCTION_SEARCH
  - BAPI_USER_GET_DETAIL
  - BAPI_USER_GETLIST

deny:
  - BAPI_USER_CREATE1:   { reason: "creates a user" }
  - BAPI_USER_DELETE:    { reason: "deletes a user" }
  - BAPI_TRANSACTION_COMMIT: { reason: "makes staged writes durable; see §4" }

rules:
  # A pure test flag: safe only when TEST_RUN is set.
  BAPI_ACC_DOCUMENT_CHECK:
    safe_when:  { }                      # this FM only checks — always safe
    else: safe
    source: curated
    reviewed: true

  Z_ACME_POST_INVOICE:
    safe_when:  { TEST_RUN: "X" }        # simulation only
    dangerous_when: { TEST_RUN: ["", " "] }
    else: dangerous
    require_confirm: true                # even the simulation asks first
    source: llm
    reviewed: true
    note: "Body COMMITs when TEST_RUN is blank (verified in source)."

  # An action/mode selector: read vs. mutate chosen by ACTION.
  BAPI_MATERIAL_MAINTAINDATA:
    safe_when:  { ACTION: [R, READ, DISPLAY, SHOW] }
    dangerous_when: { ACTION: [I, U, D, SAVE, INSERT, UPDATE, DELETE] }
    else: dangerous
    source: interface
    reviewed: true

  Z_ACME_CUSTOMER:
    safe_when:  { MODE: ["03"] }         # 03 = display (SAP activity convention)
    dangerous_when: { MODE: ["01", "02", "06"] }  # create/change/delete
    else: ask
    source: heuristic
    reviewed: false                      # heuristic guess — NOT yet callable

masks:
  - match: "*_GETLIST"
    verdict: safe
    note: "GETLIST BAPIs are queries by convention; deny overrides per FM."
  - match: "*_GETDETAIL"
    verdict: safe
  - match: "Z_ACME_RO_*"
    verdict: safe
    note: "Team convention: RO_ namespace is read-only by contract."
  - match: "*_CREATE"
    verdict: dangerous
  - match: "*_DELETE"
    verdict: dangerous
```

Note `Z_ACME_CUSTOMER` carries `reviewed: false`: it is a heuristic proposal
sitting in the review queue. With `defaults.unknown: deny` an unreviewed rule
does **not** make the FM callable — it fails closed until a human signs off. That
is the safety-critical default: **unreviewed == not callable.**


## 4. Integration with `rfc mcp`

The policy is a gate the server consults after resolving that an FM is exposed
and before it calls. Flags and modes:

| Flag | Effect |
|---|---|
| `--safe` | Only FMs the policy resolves to **safe** are callable. Conditional FMs are callable **only** when their `safe_when` matches at call time; everything else is denied. The strict default posture for AI exposure. |
| `--policy <file>` | Load the declarative policy from `<file>` (YAML or JSON). Without it, `--safe` falls back to a built-in baseline (name+interface heuristics, default-deny). |
| `--allow-commit` | Opt-in that lets `BAPI_TRANSACTION_COMMIT` (and any FM the policy tags as a durability switch) be callable. **Off by default** — writes stage but never persist unless the operator explicitly turns this on. |
| `--confirm` | For conditional / `require_confirm` FMs, require a two-step confirm (dry-run → confirm) instead of denying. |
| (existing) `--read-only` | Retained, now redefined as sugar for `--safe` with `unknown: deny` and no write path. |

### MCP tool annotations

Each exposed per-FM tool carries MCP hint annotations derived from its policy
verdict, so a well-behaved client can render and gate them:

- `readOnlyHint: true` — for `safe` FMs and for conditional FMs *in their current
  safe configuration*.
- `destructiveHint: true` — for `deny` FMs (if surfaced at all) and conditional
  FMs whose `dangerous_when` could fire.
- A structured description note when a tool is conditional: "safe only when
  `TEST_RUN='X'`" — so the model itself is told the safe path.

Annotations are advisory (hints, per MCP); the **server-side gate is
authoritative** and enforces regardless of what the client does with the hint.

### The conditional dry-run / confirmation path

When a conditional FM is called and its arguments do not clearly match
`safe_when`, or the rule sets `require_confirm`, the server does **not** execute.
It returns a structured result describing: the resolved verdict, which parameter
decided it, and — where the FM supports it — an offer to run the **simulation**
form (set the detected `TEST_RUN`/`SIMULATE` flag and call). The caller (or a
human in the loop) must re-issue with an explicit `confirm: true` to run the
mutating form. This mirrors the [MCP → RFC bridge](../mcp-rfc-bridge.md) v1
`rfc_call` rule that "mutations require `confirm_mutation: true`."

### The connection-level uncommitted-write guard

Independent of classification, the server tracks per connection whether a
possibly-staging call has run without a matching commit — the **dirty-connection
guard** from §2(c):

- Any call the policy does not resolve as `safe` marks the connection *dirty*.
- `BAPI_TRANSACTION_COMMIT` is blocked unless `--allow-commit`; when allowed and
  run, it clears *dirty*.
- On lease release / connection discard while *dirty*, the server issues
  `BAPI_TRANSACTION_ROLLBACK` (and relies on no implicit commit) so staged
  changes do not leak into durability by accident.
- The guard is defence-in-depth: even a misclassified write cannot become durable
  without an explicit, separately-gated commit.

### Composition with `--expose` / `--hide` and `.rfc.json`

Evaluation order for any call:

```
1. Is the FM exposed?      (--expose / --hide masks, .rfc.json)   — else: no such tool
2. Does the policy permit? (deny → rules → safe → masks → default) — else: denied
3. Is it a durability switch not allowed?  (--allow-commit)        — else: denied
4. Conditional & unconfirmed?  (--confirm path)                    — else: dry-run reply
5. Call.
```

`.rfc.json` gains an optional `policy` field per named system (a path, or an
inline `safe`/`deny`/`rules`/`masks` block), and a `readOnly`/`safe` boolean that
selects the strict posture — so different backends can carry different policies
(a sandbox may allow writes a production destination forbids). Flags win over
`.rfc.json`, matching the existing config precedence (flags > env > `.rfc.json`).


## 5. Build & maintenance workflow

The policy file is built up in tiers, cheapest signal first, and shipped
versioned:

1. **Seed from name heuristics + curated lists.** Run the exposed set through the
   verb tokenizer and any SAP/community catalogue. Emit a draft policy: obvious
   reads into `safe`, obvious writes into `deny`, `*_GETLIST`/`*_CREATE`-style
   masks. Everything else lands as `reviewed: false`.
2. **Refine with interface signals.** For each ambiguous FM, fetch the cached
   interface; promote FMs with a `TEST`/`SIMULATE`/`ACTION`/`MODE` lever into
   `rules` with a proposed condition; strengthen the danger prior for `RETURN
   BAPIRET2` + write-verb FMs.
3. **Deep-dive the still-ambiguous ones.** Fetch source via
   `RPY_FUNCTIONMODULE_READ`, run the LLM pipeline (§2d), emit proposed `rules`
   entries with `source: llm`, `reviewed: false`, and the rationale in `note`.
4. **Human sign-off.** A reviewer works the `reviewed: false` queue — accept,
   edit, or reject each — flipping `reviewed: true`. Only reviewed entries are
   callable under `--safe`.
5. **Ship a versioned policy file.** Commit it (e.g. `policies/baseline.yaml`)
   with `version` and `metadata.generated`. Treat edits as reviewable diffs; a
   change from `dangerous` to `safe` is a security-relevant change and reviewed
   as one.
6. **Per-system overrides.** `.rfc.json` points each named system at its own
   policy (or overlays a few entries), so a sandbox and a production box built
   from the same baseline diverge only where intended.

A `rfc policy` helper command (CLI) would drive steps 1–3 (`build`, `refine`,
`deep-dive`), print the review queue (`rfc policy review`), and validate a file
against the schema (`rfc policy lint`) — but that tooling is future work, not
part of this doc's commitment.


## 6. Risks & limits

- **No classifier is perfect.** Name heuristics miss; interfaces hide internal
  writes; LLMs hallucinate side effects in both directions. The design's answer
  is not accuracy but **posture**: default-deny, unreviewed-means-uncallable, and
  a commit gate that stops even a misclassified write from persisting.
- **Default-deny is the load-bearing decision.** An FM nobody has classified is
  not callable under `--safe`. This trades recall for safety on purpose; the cost
  is operator curation effort, which the tiered workflow is designed to minimize.
- **The operator still curates the exposed set.** The policy narrows what an
  *exposed* FM may do; it does not decide what to expose. `--expose`/`--hide` and
  `.rfc.json` remain the operator's responsibility. The two layers are
  independent and both must permit a call.
- **Reads can still leak sensitive data.** A "fully safe" read of `USR02`,
  payroll, or `BSEG` mutates nothing yet may expose data the assistant should not
  see. Data-sensitivity of *reads* is a **separate concern** (table allow-lists,
  column projection, redaction — see the bridge doc's caps) and is explicitly out
  of scope here. "Safe" in this document means "does not mutate," not "safe to
  disclose."
- **Conditional rules trust the caller's stated parameters.** The gate matches on
  the arguments presented; an FM whose true behaviour depends on system state the
  arguments don't capture can still surprise. Where the stakes are high, prefer
  `require_confirm` or `deny` over a clever conditional.
- **Policy drift.** An FM's source can change across support packages; a policy
  built against one system version may misjudge another. `metadata.system` and
  `metadata.generated` scope a policy; treat cross-system reuse as a review event.


## 7. Roadmap entry (paste into `docs/roadmap.md`)

> **P1 — Write-FM safety classification for `rfc-mcp`.** Replace the coarse
> `--read-only` (which only drops the generic `rfc_call` and is bypassed by
> auto-exposed per-FM tools) with a per-FM, per-parameter safety model. A
> versioned declarative policy file (YAML/JSON) expresses a safe allowlist, a
> dangerous denylist, and **conditional rules keyed on parameter values**
> (`safe_when: {TEST_RUN: "X"}`, `ACTION` read-vs-write selectors), with
> namespace masks and default-deny for the unknown. The server gains `--safe`,
> `--policy <file>`, `--allow-commit`, and `--confirm`; emits MCP
> `readOnlyHint`/`destructiveHint` annotations per tool; runs a dry-run/confirm
> path for conditional FMs; blocks `BAPI_TRANSACTION_COMMIT` unless explicitly
> allowed; and enforces a connection-level uncommitted-write guard that rolls back
> staged changes on discard. The policy is built in tiers — name heuristics →
> interface shape → LLM deep-dive over `RPY_FUNCTIONMODULE_READ` source, each
> proposal human-signed-off before it becomes callable — and overridable
> per-system via `.rfc.json`. See [`docs/design/write-fm-safety.md`](design/write-fm-safety.md).


## Bottom line

Safety here is a **layered posture**, not a classifier: cheap signals triage,
expensive signals (LLM over fetched source) resolve the ambiguous, a human signs
off, and the result is a versioned, reviewable declarative file. Conditional
rules keyed on parameter values capture the dual-use FMs a switch cannot. Two
runtime backstops — default-deny for the unknown and a commit gate with a
dirty-connection rollback — mean even a misclassification cannot quietly make a
write durable. Reading sensitive data remains a separate problem, deliberately
out of scope.
