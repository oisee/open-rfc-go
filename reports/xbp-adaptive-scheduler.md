# An adaptive background-job scheduler for SAP, driven from outside over classic RFC

Research and design report. Written 2026-08-21 against a live **A4H, SAP_BASIS 758,
kernel 793, client 001, single application server `vhcala4hci_A4H_00`, 5 batch work
processes** (`rdisp/wp_no_btc = 5`), reached with `open-rfc-go` and the `vsp rfc` CLI.
No product code was written.

## 0. What was probed, and what was left behind

Read-only unless noted: `TFDIR` (remote-enable flags), `DD03L`/`DD04L`/`DD07L`,
`TBTCO`, `TXMILOGRAW`, `RFC_SYSTEM_INFO`, `TH_SERVER_LIST`, `TH_WPINFO`,
`TH_GET_PARAMETER`, and `describe` on ~40 function modules. XBP calls were made
through a pinned `rfc.Session` after `BAPI_XMI_LOGON`.

**Test jobs created (as permitted):** `VSP_SCHED_PROBE1` (jobcount `23415800`) and
`VSP_SCHED_PROBE2` (`23415900`), each one step of the harmless standard report
`RSWAITSEC`. `PROBE2` was deliberately aborted with `BAPI_XBP_JOB_ABORT` to observe a
failure transition, then deleted with `BAPI_XBP_JOB_DELETE`. **`VSP_SCHED_PROBE1` was
left in place, status `F`**, as the surviving evidence artifact — delete it at leisure.
No job that we did not create was touched; no `SAP_*` job was read for anything beyond
its status column.

---

## 1. The API verdict

> **Use XBP 3.0. It is not a close call, and it is not really a choice.**
> Dispatch and control-plane writes go through XBP over a *pinned* connection.
> **Observation does not** — poll `TBTCO` with `RFC_READ_TABLE`, which costs no XMI
> audit row and no XBP session. That hybrid is the actual recommendation; "use XBP"
> on its own would produce a scheduler that quietly fills an audit table.

### Evidence

**The internal route is not available at all.** `JOB_OPEN`, `JOB_SUBMIT`, `JOB_CLOSE`
have blank `TFDIR-FMODE`. So do `BP_JOB_SELECT`, `SPBT_INITIALIZE`,
`SPBT_GET_PP_DESTINATION`, and `SPBT_GET_CURR_RESOURCE_INFO`. A classic-RFC client
cannot call a non-remote-enabled FM; there is no flag, no workaround, no privilege
that changes this. The only ways to reach them are (a) install a Z wrapper FM marked
remote-enabled, which means shipping ABAP into the customer system and owning its
transport, its authorization checks and its upgrade risk, or (b) don't.

**`SUBST_START_REPORT_IN_BATCH` is a dispatch primitive, not an API.** It is
remote-enabled, and on this system it fails with `BATCH_SCHEDULING_FAILED` (XM262).
Even if it worked it would be the wrong tool: its entire output is `EV_JOBCOUNT`,
`EV_STARTRC`, `EV_VARIWRC`. No status, no job log, no abort, no selection, no
resource query. Everything the scheduler needs after the start would have to come
from raw table reads anyway, and everything it needs *before* the start — job class,
target server, print parameters, a step under a different user — is either absent or
a single scalar. It is the "fire a report and hope" call, and the requirement here is
the opposite of hope.

*(Incidentally: both `SUBST_START_REPORT_IN_BATCH` and `SUBST_START_BATCHJOB` expose
`IV_BATCHHOST`/`IV_BATCHINSTANCE`. XBP's `BAPI_XBP_JOB_START_ASAP` **requires**
`TARGET_SERVER` and works; the XM262 failure smells like the same missing explicit
placement. Untested — see open question 1. It would not change the verdict.)*

**XBP is a complete lifecycle API and it is essentially all remote-enabled.** 82 FMs
match `BAPI_XBP*` on this system; **79 have `FMODE = 'R'`**. The three that do not are
the internal introspection helpers `BAPI_XBP_INTRFACE_DESCRIBE_INT`,
`BAPI_XBP_VERSIONS_GET_INT`, `BAPI_XBP_VERSION_CHECK_INT`. `BAPI_XMI_GET_VERSIONS`
reports `XBP 3.0` (alongside XMB 0.1, XOM 0.2, XAL 1.0).

**The whole chain was exercised live**, end to end, from Go: logon → open → step →
start → status transitions → abort → job log → statistics → select → delete.
Two jobs went `R` → (`F`, `A`), free batch work processes tracked the dispatch
(`5 → 3 → 4 → 5`) with a lag of seconds, and the job log came back with the
message ids that a classifier needs.

### The cost of XBP, measured

XBP is gated by an XMI session and every XBP call writes an audit row.
`TXMILOGRAW` held **69 rows in total** on this system; **31 of them were written by our
~30 probe calls**, at `AUDITLEVEL = 0`. One five-minute probe session accounted for
45% of every XMI audit row this system has ever recorded.

That is the reason the recommendation is a hybrid. A scheduler that polls job status
over `BAPI_XBP_JOB_STATUS_GET` every five seconds writes ~17k rows in a day, per
concurrent job. `RFC_READ_TABLE` on `TBTCO` writes none, returns the same status plus
the scheduling and start timestamps the control loop actually needs, and was verified
to work with a `JOBNAME LIKE 'VSP_SCHED_%'` mask.

---

## 2. FM inventory

Everything below is `FMODE = 'R'` (remote-enabled) unless the row says otherwise.
`u` = `EXTERNAL_USER_NAME`, a free-text label the caller supplies; it is not checked
against anything, it lands in the audit log and the job's `LASTCHNAME` context.
All XBP calls also require a prior `BAPI_XMI_LOGON` **on the same connection**.

### Session

| FM | Interface (abridged) | Notes |
|---|---|---|
| `BAPI_XMI_LOGON` | in `*EXTCOMPANY`, `*EXTPRODUCT`, `INTERFACE`, `VERSION`; out `SESSIONID`, `RETURN` | `INTERFACE='XBP'`, `VERSION='3.0'`. Arbitrary company/product accepted — no registration needed, but `S_XMI_PROD` is checked against them. Binds to **this connection**. |
| `BAPI_XMI_LOGOFF` | in `INTERFACE` | Always defer it. |
| `BAPI_XMI_GET_VERSIONS` | out `VERSIONS[]` | Capability probe; cache once. |
| `BAPI_XMI_SELECT_LOG` | in `EXTUSER`, `SESSIONID`, `OBJECT`, `FROM/TOTIMSTMP`; out `LOG[13]`, `NUMBER` | Read your own audit trail back. Useful for a post-mortem. |
| `BAPI_XBP_NEW_FUNC_CHECK` | in `*u`, `GET_NEW_FUNC`, `INTERCEPTION_ACTION`, `PARENTCHILD_ACTION`; out `INTERCEPTION`, `PARENTCHILD`, `NEW_FUNC[4]` | **Live: both `INTERCEPTION` and `PARENTCHILD` returned blank — off on this system.** `NEW_FUNC` listed only the `TIMEZONE` extensions (note 3170310). The `*_ACTION` inputs *switch these on system-wide*; never let the scheduler do that. |

### Dispatch

| FM | Interface (abridged) | Notes |
|---|---|---|
| `BAPI_XBP_JOB_OPEN` | in `*u`, `*JOBNAME`, `JOBCLASS`; out `JOBCOUNT` | Job class defaults to `C`. Returns the 8-char jobcount that, with the name, is the job's real key. |
| `BAPI_XBP_JOB_ADD_ABAP_STEP` | in `*u`, `*JOBNAME`, `*JOBCOUNT`, `*ABAP_PROGRAM_NAME`, `ABAP_VARIANT_NAME`, `SAP_USER_NAME`, `LANGUAGE`, `SELINFO[]`/`SELINFO_L[]` (RSPARAMS: `SELNAME,KIND,SIGN,OPTION,LOW,HIGH`), `PRINT_PARAMETERS`, `ALLPRIPAR`, `ARCHIVE_PARAMETERS`, `FREE_SELINFO[]`; out `STEP_NUMBER`, `TEMP_VARIANT` | The workhorse. Passing `SELINFO` makes XBP mint a **temporary variant** — live, the job log showed `variant &0000000000000`. Those accumulate in `VARI`/`VARID`; see risks. Without `PRINT_PARAMETERS` the step produces no spool (`TBTCP-LISTIDENT` stays 0). |
| `BAPI_XBP_JOB_ADD_EXT_STEP` | as above for external commands | Not needed here; a big authorization surface. |
| `BAPI_XBP_JOB_START_ASAP` | in `*u`, `*JOBNAME`, `*JOBCOUNT`, `*TARGET_SERVER`, `TARGET_GROUP` | `TARGET_SERVER` is mandatory; take it from `RFC_SYSTEM_INFO-RFCDEST` or `TH_SERVER_LIST`. `TARGET_GROUP` (an RZ12 server group) would delegate placement to SAP — untested here, one app server. |
| `BAPI_XBP_JOB_START_IMMEDIATELY` | same shape | Fails if no WP is free rather than queueing. Useful as an *admission test*, but the resource query is cheaper. |
| `BAPI_XBP_JOB_HEADER_MODIFY` | in `*u`, `*JOBNAME`, `*JOBCOUNT`, `*JOB_HEADER`, `JOBCLASS`, `MASK{TSERVER,THOST,TSRVGRP,STARTCOND,RECIPLNT}`, `DONT_RELEASE` | Re-target or re-class a job after opening it. `MASK` selects which fields the call is allowed to touch. |
| `BAPI_XBP_JOB_COPY` | in `*u`, `*SOURCE_JOBNAME/JOBCOUNT`, `TARGET_JOBNAME`, `STEP_NUMBER`; out `TARGET_JOBCOUNT` | Tempting for "restart an equivalent unit" — **don't**. It copies the step verbatim including the temp variant, so the retry is indistinguishable from the original in every way that matters. Re-open with a fresh, attempt-stamped name instead. |
| `BAPI_XBP_EVENT_RAISE` | in `*u`, `*EVENTID`, `EVENTPARM` | Event-driven starts. `BP_REMOTE_EVENT_RAISE` is the non-XBP equivalent, also remote-enabled. |
| `BAPI_XBP_BTC_EVTHISTORY_GET` / `_CONFIRM` | event history by timestamp/GUID | For an event-driven design; not needed for a push scheduler. |

### Observation

| FM | Interface (abridged) | Notes |
|---|---|---|
| `RFC_READ_TABLE` on `TBTCO` | — | **The recommended poll.** Verified: `--fields JOBNAME,JOBCOUNT,STATUS,SDLSTRTDT,SDLSTRTTM,STRTDATE,STRTTIME,ENDDATE,ENDTIME,REAXSERVER,WPNUMBER --where "JOBNAME LIKE 'VSP_SCHED_%'"`. One call, whole fleet, **no audit row**. Everything the control loop measures comes from these eleven columns. |
| `BAPI_XBP_JOB_STATUS_GET` | in `*u`, `*JOBNAME`, `*JOBCOUNT`; out `STATUS`, `HAS_CHILD` | One job per call. Fine for a spot check, wrong for a poll loop. |
| `BAPI_XBP_JOBLIST_STATUS_GET` | in `*u`, `JOBLIST[JOBNAME,JOBCOUNT]`, `READ_ONLY_STATUS`; out `JOBLIST[+STATUS,HAS_CHILD]` | **N jobs, one call, one audit row.** The right XBP poll if you must poll over XBP. |
| `BAPI_XBP_JOB_STATUS_CHECK` | out `ACTUAL_STATUS`, `STATUS_ACCORDING_TO_DB` | The **zombie detector**. When these disagree, `TBTCO` says `R` but no process is running. There is no other way to learn this from outside. |
| `BAPI_XBP_JOB_COUNT` | in `*u`, `*JOBNAME` (mask), `DONT_LIST_JOBS`; out `NUMBER_OF_JOBS`, `JOB_TABLE[35]` | **The reconciliation primitive.** Live, `JOBNAME='VSP_SCHED_PROBE*'` returned `n=2` and the full 35-column header table for both. No `USERNAME` trap (see next row). |
| `BAPI_XBP_JOB_SELECT` | in `*u`, `*JOB_SELECT_PARAM{JOBNAME,USERNAME,JOBCOUNT,JOBGROUP,EVENTID,FROM/TO dates,NO_DATE,PRELIM,SCHEDUL,READY,RUNNING,FINISHED,ABORTED}`, `ABAPNAME`, `SUSP`; out `SELECTED_JOBS[]`, `JOB_HEAD[35]`, `ERROR_CODE` | Richer filters. **Trap found live: with `USERNAME` blank it returns an empty set and `ERROR_CODE = 0`** — a silent no-match, not an error. Pass `'*'` or the batch user. |
| `BAPI_XBP_JOB_READ` / `_DEFINITION_GET` | full header, steps (`STEPS[60]`), spool attrs (`SPOOL_ATTR[42]`), recipient | Heavy. Use for diagnosis, not for polling. |
| `BAPI_XBP_GET_STEP_INFORMATION` | out `BPJOBSTEP_TBL[26]` | Per-step detail incl. the step's own status. |
| `BAPI_XBP_JOB_JOBLOG_READ` | in `*u`, `*JOBNAME`, `*JOBCOUNT`, `PROT_NEW`, `LINES`, `DIRECTION`; out `JOB_PROTOCOL_NEW[15]` | **The failure oracle.** With `PROT_NEW='X'` each line carries `MSGID`, `MSGNO`, `MSGTYPE`, `MSGV1..4`, `TEXT`, and — crucially — **`RABAXKEY`**, the short-dump key. Live: success ended `00/517 S "Job finished"`; the aborted job ended `00/554 A "Job canceled due to terminated process or program"`. |
| `BAPI_XBP_GET_APPLICATION_RC` | in `*u`, `*JOBNAME`, `*JOBCOUNT`, `*STEPCOUNT`; out `APP_RC`, `APP_RC_DESCR`, `STATUS` | The *designed* answer to "did the unit succeed". **Live it returned `XM232 "No step information found"` for a normally-completed job.** Nothing in a stock report populates it. Do not build the success test on this. |
| `BAPI_XBP_JOB_SPOOLLIST_READ` | in `*u`, job, `*STEP_NUMBER`; out `SPOOL_LIST[1]` | Needs `STEP_NUMBER`, and needs the step to have had print parameters. `_READ_20`, `_READ_RW`, `BAPI_XBP_GET_SPOOL_AS_PDF/HTML/DAT`, `BAPI_XBP_READ_SPOOL_BIN` are format variants. |
| `BAPI_XBP_APPL_LOG_CONTENT_GET` | application log (SLG1) content | The clean way for a unit to report its own outcome, if the ABAP side cooperates. |
| `BAPI_XBP_BTC_STATISTIC_GET` | in `*I_EXTERNAL_USER_NAME`, `*I_T_JOBLIST[]`; out `T_STATDATA[5]` incl. nested `T_STATISTIC` | **Rich and expensive.** Live it returned a full STAD-style record: `PROCTI 45,006,731 µs`, `RESPTI 45,046,962`, `CPUTI`, `DBREQTIME 39,183`, `MAXROLL`, per-connection DB counters, `ROLLINCNT`, memory high-water. This is how you tell "the unit got slower" from "the database got slower". Sample it, don't poll it. |
| `BAPI_XBP_JOB_CHILDREN_GET`, `_PARENT_CHILD_INFO` | child jobs of a job | Only meaningful with parent/child switched on; **off here**. |
| `BAPI_XBP_GET_INTERCEPTED_JOBS`, `BAPI_XBP_MODIFY_CRITERIA_TABLE` | interception queue + `TBCICPT1` criteria | Interception is **off** here. Live, `GET_INTERCEPTED_JOBS` returned an empty set with `RETURN` clean. |
| `BAPI_XBP_SYNCHRONIZE_JOBS` | paged bulk job export, `JOBS[62]`, `MORE`, `MAX` | Built for an external scheduler to mirror the whole job catalogue. Overkill unless you also want to *own* the system's jobs. |
| `BAPI_XBP_READ_SELSCREEN` | in `*u`, `*PROGRAM`, `DEFAULT_VALUES`; out `SELSCREEN_INFO[16]` | Live on `RSWAITSEC` it returned `SELNAME=WAITTIME, KIND=P, DTYP=INT4`. Use it to validate a unit's parameters before dispatching a thousand jobs with a typo. |
| `BAPI_XBP_VARIANT_CREATE/CHANGE/COPY/DELETE`, `_INFO_GET`, `_255` variants | named variants | An alternative to `SELINFO` temp variants: create one named variant per unit shape, reuse it. Fewer `VARI` rows; more state to reconcile. |

### Capacity

| FM | Interface (abridged) | Notes |
|---|---|---|
| `BAPI_XBP_GET_CURR_BP_RESOURCES` | in `*u`; out `RESOURCE_INFO[SERVER,HOST,BTCWPTOTAL,BTCWPFREE,BTCWPCLSSA]`, `SRVGRP_INFO` | **The ceiling, per server.** Live: `BTCWPTOTAL " 5"`, `BTCWPFREE " 5"` → `" 3"` under two jobs → `" 4"` → `" 5"`. Values are **right-aligned strings with leading blanks — trim them.** `BTCWPCLSSA` is the count reserved for job class A; class-C jobs cannot have those. |
| `TH_WPINFO` (plain RFC, no XMI) | in `MAX_ELEMS`, `SRVNAME`, `WITH_CPU`; out `WPLIST[34]` | Same ceiling, **no audit row**, plus per-WP `WP_TYP`, `WP_STATUS`, `WP_REPORT`, `WP_BNAME`, `WP_ELTIME`, `WP_DUMPS`. Live: 15 rows, `BGD/Waiting: 5`. This is also how you answer *"who is eating the box"* when you back off. |
| `TH_SERVER_LIST` | out `LIST[7]`, `LIST_IPV6[14]` | Legal `TARGET_SERVER` values. Live: one server, `vhcala4hci_A4H_00`. Cache; it changes on instance start/stop. |
| `TH_GET_PARAMETER` | in `PARAMETER_NAME`; out `PARAMETER_VALUE` | Live: `rdisp/wp_no_btc = 5`. Static; read once. |
| `BAPI_XBP_GET_BP_RESRC_ON_DATE`, `_BP_SRVRES_ON_DATE` | resources at a future date/time | Honours operation modes (RZ04). If the customer switches WPs between day and night modes, this is how you *plan* rather than react. |
| `RFC_READ_TABLE` on `TBTCO` without a name filter | `STATUS IN ('S','Y','Z','R')` | System-wide queue depth — are *other people* also waiting? Wider scan; run it every few minutes, not every tick. |

### Deliberately not used

`BAPI_XBP_JOB_ABORT` and `BAPI_XBP_JOB_DELETE` exist and work (both verified), and
**cross-connection**: `PROBE2` was created on one pinned session and deleted from a
different one. The scheduler should call them only for jobs whose name carries its own
prefix *and* its own run id, and should prefer adopting an unexpected job over killing
it. See §7.

---

## 3. The alternatives, and when not to create jobs at all

**Client-side concurrency over plain RFC — consider this first.** If the unit of work
is already an RFC-enabled function module, the simplest correct scheduler calls it N
ways concurrently from Go over the connection pool and never creates a job. You get
the return values and the typed `*ABAPException` directly instead of parsing a job log;
backpressure is the pool; there is no `TBTCO` churn, no XMI, no audit row, no temp
variant, no reconciliation problem, and the failure signal is synchronous and precise.
The control loop in §4 applies unchanged — `C` becomes the pool's in-flight count.

Use background jobs instead when at least one of these is true:
- the unit runs longer than `rdisp/max_wprun_time` (dialog work processes are killed at
  that limit; batch WPs are not);
- the work must survive the client disconnecting, or must be resumable by someone else;
- the unit is a *report* with a selection screen and no callable FM behind it;
- it must run under a specific batch user's authorizations, not the RFC user's;
- it must produce a spool list, or be visible and auditable in SM37 to operations;
- you must not consume dialog work processes, because interactive users need them.

For "many similar screening units", the honest question is whether the unit is a report
or a function module. If someone can expose it as an RFC-enabled FM, do that and skip
this entire report's machinery.

**`SPBT_*` parallel processing is not reachable from outside.** `SPBT_INITIALIZE`,
`SPBT_GET_PP_DESTINATION` and `SPBT_GET_CURR_RESOURCE_INFO` all have blank `FMODE`.
Only `SPBT_PARALLEL_PROCESSING` and the monitor helpers are remote-enabled, and they
are not the fan-out primitive. The `CALL FUNCTION … STARTING NEW TASK … DESTINATION IN
GROUP` pattern is an *in-ABAP* API: driving it means installing an ABAP driver program
and then scheduling *that* — at which point you are back to needing XBP, with an extra
component to maintain. Its one real advantage is that SAP does the admission control
for you (`SPBT_INITIALIZE` tells you the free resources of the RFC server group and the
`RESOURCE_FAILURE` exception is your backpressure). If a Z program is on the table
anyway, this is a serious contender and would collapse the whole control loop into one
ABAP call — at the cost of losing per-unit restartability and external visibility.

**bgRFC / qRFC — right idea, wrong decade for this codebase.** Server-side queues
would solve exactly-once and "restart an equivalent unit" properly: a unit becomes a
queued LUW with a TID, SAP guarantees it executes once, ordering is a queue-name
property, and failed units sit in the queue for inspection and re-execution. Three
reasons not to, today: (1) **`open-rfc-go` does not implement tRFC/qRFC/bgRFC** — it is
roadmap P2 ("substantial new wire work: TID lifecycle, confirm/commit"); (2) you hand
concurrency control to the bgRFC scheduler, tuned in `SBGRFCCONF` by Basis, which is
the *opposite* of the adaptive requirement; (3) the units must be function modules,
not reports. Revisit if exactly-once ever outranks adaptive concurrency.

**Summary of the decision:**

| Approach | Reachable over RFC today | Lifecycle control | Concurrency control | Verdict |
|---|---|---|---|---|
| `JOB_OPEN`/`SUBMIT`/`CLOSE` | **no** (`FMODE` blank) | — | — | impossible |
| `SUBST_START_REPORT_IN_BATCH` | yes, but XM262 here | start only | none | insufficient |
| **XBP 3.0** | **yes, 79/82 FMs** | **full** | **yours** | **recommended** |
| `SPBT_*` in-ABAP | no (needs Z driver) | none per unit | SAP's, good | strong if ABAP is allowed |
| bgRFC/qRFC | not in this client | queue-level | SAP's, not yours | later |
| direct FM calls, client-side fan-out | yes | n/a (synchronous) | yours, trivially | **best if the unit is an FM** |

---

## 4. The control loop

### 4.1 Capacity signals and what they cost

| Signal | Source | Cost | Poll it |
|---|---|---|---|
| Hard ceiling: free batch WPs | `TH_WPINFO` count of `WP_TYP='BGD' AND WP_STATUS='Waiting'` | 1 plain-RFC call, ~15 rows, no audit row | every tick |
| Same, authoritative + per-server + class-A reservation | `BAPI_XBP_GET_CURR_BP_RESOURCES` | 1 XBP call, **1 audit row** | every 30–60 s, or on ceiling change |
| Total BTC WPs | `TH_GET_PARAMETER rdisp/wp_no_btc` | 1 call | once |
| Legal target servers | `TH_SERVER_LIST` / `RFC_SYSTEM_INFO-RFCDEST` | 1 call | once, refresh hourly |
| Our fleet: status, queue wait, run time | `RFC_READ_TABLE TBTCO WHERE JOBNAME LIKE '<prefix>%'` | 1 call, 11 columns × in-flight rows, **no audit row** | every tick |
| Everyone's queue depth | `RFC_READ_TABLE TBTCO WHERE STATUS IN ('S','Y','Z','R')` | wider scan | every 2–5 min |
| Why a job was slow | `BAPI_XBP_BTC_STATISTIC_GET` | large nested payload + 1 audit row | 1 job in 20 |
| Who else is on the box | `TH_WPINFO` `WP_REPORT`/`WP_BNAME`/`WP_ELTIME` | free, same call as the ceiling | only when backing off |

The tick is the poll of `TBTCO` plus `TH_WPINFO`: two plain-RFC calls, no audit rows,
sub-second. Make the interval adaptive — `max(2s, p50_run/10)`, capped at 30 s — so that
short units are not polled five hundred times each and long ones are not polled once.

### 4.2 The scaling rule — AIMD on completion latency under a hard ceiling

Let `C` be the number of our jobs allowed in flight (statuses `S`,`Y`,`Z`,`R`).

```
Cmax  = max(1, min(MaxJobs, floor(alpha * BTCWPTOTAL) , BTCWPFREE_now + inflight_ours - reserve))
Cmin  = 1                      alpha = 0.5 (default)   reserve = 1 (default)
```

`alpha` and `reserve` are the two knobs a human sets per system. On this A4H:
`BTCWPTOTAL 5`, so `Cmax = 2`. That will feel small. It is correct: filling the last
batch WP starves `SAP_REORG_*`, `SAP_COLLECTOR_FOR_PERFMONITOR` and the spool
housekeeping, and the operator will notice before you do.

Every **epoch** — defined as "at least `C` completions have been observed since the last
change to `C`", i.e. one full generation at the current setting, never a wall-clock
interval:

```
p90_run  = quantile(0.90, run  times over the last W completions)     W = max(20, 10 min of completions)
p90_wait = quantile(0.90, wait times over the same window)
baseline = EWMA of p90_run over the first B=10 post-warm-up completions,
           ratcheting DOWN only (it follows genuine speed-ups, never drifts up to bless a degradation)

if  p90_run <= 1.2 * baseline  and  p90_wait <= Wait_ok  and  no failure in window:
        C = min(C + 1, Cmax)                        # additive increase, at most once per epoch
elif p90_run >  1.5 * baseline  or   p90_wait > 3 * Wait_ok:
        C = max(Cmin, C / 2)                        # multiplicative decrease, allowed at any time
else:   hold
```

`Wait_ok` defaults to 30 s.

**Why AIMD.** The contended resource is a small shared pool that you do not own — SAP's
own jobs, other schedulers, and interactive load all draw on the same batch WPs, DB
connections and buffers. That is precisely the setting AIMD was designed for, and its
relevant property is not that it is optimal but that **independent controllers using it
converge to a fair share without coordinating**. Additive increase probes one job at a
time, so the blast radius of a wrong guess is one job. Multiplicative decrease sheds
load fast, which matters because SAP degradation is superlinear once paging or lock
contention starts: by the time you notice, halving is the conservative move.

Rejected alternatives: *fill every free WP* maximises your throughput and destroys
everyone else's, and makes the queue wait for SAP housekeeping unbounded. *Target
utilisation with a PID controller* needs a plant model and per-system tuning; AIMD
needs two thresholds and a window. *Little's-law sizing* (`C = throughput × latency`)
assumes a stable service time, which is exactly the assumption the requirement says is
false.

**Oscillation, and the four things that prevent it.** The failure mode is a limit cycle:
increase → latency rises → halve → latency falls → increase, forever, with a period
equal to the measurement window, delivering worse throughput than a fixed `C`.

1. **Completion-driven epochs.** A change to `C` is judged only by jobs that ran
   *entirely* under it. Evaluating on a wall clock means judging the new setting partly
   by jobs dispatched under the old one, which is how you get a controller chasing its
   own tail. This is the single most important defence.
2. **A hysteresis band.** Increase at `≤1.2×`, decrease at `>1.5×`; between them, hold.
   A single threshold guarantees a cycle, because the controller is never at rest.
3. **Quantiles, not averages.** `p90` over `W ≥ 20` samples. A mean is dragged by one
   10× outlier into a spurious halving and by one fast job into a spurious increase. An
   empirical quantile over a 20–50 sample window is a few microseconds to compute; there
   is no reason to approximate it.
4. **A cool-down after every decrease.** Hold `C` for at least `2 × p90_run` before
   allowing an increase, so you are not re-probing while the system is still draining the
   overload you just caused.

Asymmetry is deliberate: at most one increase per epoch, but a decrease is permitted the
moment the evidence appears. Down is the safe direction.

**Dispatch admission.** Per tick, start at most `min(C, Cmax) − inflight` jobs, and no
more than `R` starts per second (dispatch is three XBP round trips per job and each
start takes an enqueue on the job tables). One pinned XBP session serialising dispatch
gives you this rate limit for free.

### 4.3 Slowdown detection

Three times, all derivable from the one `TBTCO` read, and they mean different things:

- **Queue wait** = `STRTDATE/STRTTIME − SDLSTRTDT/SDLSTRTTM`. Rising wait with flat run
  time means the batch WPs are saturated — by us or by someone else. Adding jobs cannot
  help: throughput is already at the ceiling and every new job only lengthens the queue.
  **This is what gates the additive increase.**
- **Run time** = `ENDDATE/ENDTIME − STRTDATE/STRTTIME`. Rising run time with flat wait
  means the unit got more expensive or the substrate degraded. **This is what triggers
  the multiplicative decrease.**
- **Dispatch latency** — our own wall clock from `START_ASAP` to first observation of
  status `R`. Rising means the batch scheduler itself is struggling, and it is also what
  catches the XM262-class "it never picks a server" failure.

Separating "the unit got bigger" from "the system got slower": normalise run time by the
unit's own size (rows, records, whatever the plan knows) and track the *per-item*
quantile; and on a 1-in-20 sample call `BAPI_XBP_BTC_STATISTIC_GET`. If `PROCTI` grows
while `CPUTI` stays flat, the job is waiting, not working. If `DBREQTIME` grows, it is
the database. If `MAXROLL`/`MEMSUM` grow, it is memory and you are about to see
`TSV_TNEW_PAGE_ALLOC_FAILED`.

Two measurement caveats. `TBTCO` times are `HHMMSS` — **one-second resolution, no
sub-second**, so a 3-second job measures as 3±1 s; if your units are seconds long, jobs
are the wrong mechanism. And date and time are separate `CHAR` fields: compose them into
a real timestamp (system timezone from `RFC_SYSTEM_INFO-RFCTZONE`) or every job that
spans midnight will report a negative duration.

### 4.4 Backoff and the 20–30 minute review

Two distinct mechanisms, and it matters that they are distinct:

- The **AIMD loop** runs continuously and reacts within an epoch. It handles ordinary
  jitter and does not need a human.
- **Backoff-and-review** is a state the scheduler *enters* when the slowdown is
  structural rather than transient. This is what the maintainer asked for.

Enter backoff when any of:
- `C` has been driven to `Cmin` by two consecutive multiplicative decreases **and**
  `p90_run` is still `> 2 × baseline`;
- `p90_wait > 5 min` — our jobs are sitting in a queue, which means someone else owns
  the machine and no amount of local tuning will help;
- `BTCWPFREE = 0` for `k` consecutive ticks with none of the occupants ours.

On entry:
1. **Stop dispatching. Do not abort anything that is running** — in-flight jobs are
   already paid for, and killing them wastes the work and adds rollback load.
2. Snapshot the evidence: current quantiles, the baseline, the `BTCWPFREE` history, the
   last N job headers, and `TH_WPINFO`'s `WP_REPORT`/`WP_BNAME`/`WP_ELTIME` — the last of
   these names the culprit, which is the single most useful thing a human will want.
3. Set the review timer to **`T = 20 min + U(0, 10 min)`**. The jitter inside the
   maintainer's 20–30 minute window is not decoration: it is what stops two schedulers,
   or a scheduler and whatever slowed the system down, from resynchronising into a
   thundering herd every time they both back off.

At review, re-measure. If recovered (`p90_run ≤ 1.2 × baseline` **and** `BTCWPFREE ≥ 2`):
resume at **`C = 1`** and let AIMD re-ramp. Never resume at the old `C` — that is the
setting that produced the collapse. If not recovered: re-arm with exponential backoff
(20–30, 40–60, 80–120 min, capped at ~2 h) and, **after the second failed review, page a
human**. A scheduler is not in a position to decide that a production system's slowness
is acceptable.

---

## 5. Failure taxonomy

### 5.1 Telling "failed" from "finished"

| Observation | Meaning | Action |
|---|---|---|
| `STATUS='F'`, log ends `00/517 S "Job finished"`, no `A`/`E` lines | batch-level success | **still verify the unit's own outcome** — see 5.2 |
| `STATUS='F'`, log contains `MSGTYPE='E'` or `'A'` lines before the end | the report reported errors and kept going | failed; classify from `MSGID`/`MSGNO` |
| `STATUS='A'` | cancelled: dump, hard cancel, WP death, or an explicit abort | failed; classify. Live, our abort produced `00/554 A "Job canceled due to terminated process or program"` |
| `STATUS='A'` and a log line with `RABAXKEY ≠ ''` | ABAP short dump | classify from the dump key; usually poison or resource |
| `STATUS='R'` but `ACTUAL_STATUS ≠ STATUS_ACCORDING_TO_DB` | **zombie** — DB says running, no process | failed; re-dispatch after confirming no live claim |
| `STATUS` in `'Z'`/`'S'`/`'Y'` past the dispatch SLA | never picked up. **This system has 65 `Z` rows stranded since 2017** — real, permanent limbo | scheduling failure; clean up the ghost, re-dispatch |
| no `TBTCO` row for a name we journalled | the start never committed | re-dispatch (safe iff idempotent) |

`TBTCO-STATUS` uses domain `CHAR1` with **no fixed value list** — the codes are
convention, not a DDIC constraint, so validate defensively. Observed in this system's
5000 most recent rows: `F 4925`, `Z 65`, `A 9`, `S 1`. The standard set is
`P` scheduled, `S` released, `Y` ready, `Z` handover-to-WP, `R` active, `F` finished,
`A` cancelled, `X` unknown.

### 5.2 "Finished" is not "succeeded"

`BAPI_XBP_GET_APPLICATION_RC` is the designed answer and it does not work out of the
box — live it returned `XM232 "No step information found"` for a cleanly completed job,
because nothing in a stock report registers a step return code. One of the following must
be true, and the maintainer has to pick:

- **(a) Recommended — the unit writes its own result row.** A Z table keyed by
  `unit_id` with `status`, `attempt`, `message`, `changed_at`. The scheduler reads it with
  `RFC_READ_TABLE` (cheap, no audit row, batched by key range) and *that table is the
  authority on what is done*. This also gives you the idempotency claim in §6 for free.
- **(b)** The unit writes an application log; read it with
  `BAPI_XBP_APPL_LOG_CONTENT_GET`.
- **(c)** The unit writes a spool list; read it with `BAPI_XBP_JOB_SPOOLLIST_READ`.
  Requires `PRINT_PARAMETERS` at `ADD_ABAP_STEP` (otherwise `TBTCP-LISTIDENT` stays 0)
  and means parsing a formatted list. Brittle.
- **(d)** Trust `STATUS='F'`. Only defensible if the unit is written so that *any*
  failure terminates the job — and you should say so out loud in the config.

### 5.3 Retryable vs poisonous

Classify on `(status, MSGID/MSGNO, RABAXKEY, the unit's own result row)`:

- **Transient / infrastructure — retry with backoff; does not count against the unit's
  poison budget.** WP death (`00/554`), update termination, `SYSTEM_CANCELED`, enqueue
  or lock-wait timeouts, `DBIF_RSQL_SQL_ERROR` with a deadlock sub-reason, `TIME_OUT`,
  `RESOURCE_FAILURE`, `MEMORY_NO_MORE_PAGING`, "no free work process", a job stranded
  in `Z`.
- **Data / logic — poison; do not retry the same unit unchanged.** Uncaught `CX_SY_*`,
  `ASSERTION_FAILED`, `CONVT_NO_NUMBER`, `MESSAGE_TYPE_X`, `CALL_FUNCTION_NOT_FOUND`,
  and anything the unit's own result row reports as a validation failure. Retrying burns
  a work process to reproduce the same dump. Park it with its diagnosis and move on.
- **Ambiguous — bounded retries (2), then poison.** `TSV_TNEW_PAGE_ALLOC_FAILED` is the
  archetype: transient at low concurrency, structural at high. Retry it once *at
  `C = 1`*, which doubles as a diagnostic.
- **Environmental / global — retry nothing, enter backoff-and-review.** Authorization
  failures, `BATCH_SCHEDULING_FAILED`/XM262, XMI logon rejection, or more than `k`
  consecutive failures across *different* units.

The rule that keeps this honest: **a unit-level retry needs a unit-level cause; a global
pause needs a global cause.** The discriminator is whether consecutive failures are the
same unit or different ones. Five different units failing in a row is a system fault.
The same unit failing three times is poison.

Caveat on diagnosis depth: **short dumps are not readable over RFC by the obvious
route.** `SNAP` is a cluster table — `RFC_READ_TABLE` returns `TABLE_NOT_AVAILABLE`. The
job log's `RABAXKEY` tells you a dump happened and identifies it, but reading its text
needs `RS_ST22_RFC` (remote-enabled, but a generic `TOOL`/`P_TOOL_TAB` dispatcher that
needs working out) or a small Z FM. **First cut: classify from `MSGID`/`MSGNO` and the
presence of `RABAXKEY`, and put the key in the report so a human can open ST22.**

---

## 6. Idempotency — making "restart an equivalent unit" safe

The phrase hides the whole risk: a retried job must not double-process. In order of
importance:

1. **Stable, caller-chosen unit ids.** `unit_id` derives from the work itself (the
   screening batch key), not from a job count, a row number or a timestamp. The plan is
   a set of `unit_id`s; it is content-addressed, not position-addressed, so restarting
   the scheduler cannot shift the mapping.
2. **Attempt ids.** Each dispatch is `(unit_id, attempt_n)`, encoded in the job name, so
   a retry is distinguishable from the original in SM37, in `TBTCO`, and in the journal.
3. **A server-side claim, not client-side bookkeeping.** The unit program's first action
   is: `ENQUEUE` on `unit_id`; read the result row; if it is `DONE`, exit immediately as
   a no-op; otherwise write `RUNNING` with this attempt id and proceed. This is the only
   construction that survives the case that actually matters — **the scheduler believes a
   job died but it is still running** (network partition, scheduler restart, zombie
   status). A client cannot distinguish "dead" from "unreachable", so no amount of
   client-side state can prevent the double execution; only the server can.
4. **Post-conditions over pre-conditions.** A unit is done because its result row says
   `DONE`, never because the scheduler remembers dispatching it.
5. **If the ABAP side cannot be changed**, be explicit about the weaker guarantee:
   either the unit is naturally idempotent (a pure recompute into a keyed result), or the
   scheduler runs **at-most-once** — retry only units whose job never reached status `R`,
   and report the rest for manual decision. That is a real and defensible mode. It should
   be a named configuration setting, not an accident.

---

## 7. State, restart and reconciliation

State lives in three tiers, and the top one is deliberately thin:

1. **Authoritative in SAP** — which units are done (the unit result table), which jobs
   exist and in what state (`TBTCO`), and what happened (the job log).
2. **Authoritative in the scheduler, must be durable** — the plan, the attempt counters,
   the poison set with diagnoses, and the AIMD state (`C`, baseline quantiles, review
   deadline). One local SQLite file, or an append-only JSONL journal plus periodic
   snapshot.
3. **Recomputable** — the latency window and everything derived from `TBTCO`.

**Write the intent before the call.** Journal `"about to dispatch U attempt 3 as
ZSCR_7K2P_U0041_03"`, fsync, *then* `JOB_OPEN`. That ordering is what makes recovery
decidable: after a crash the claimed name either exists in `TBTCO` or it does not, and
both cases have a defined answer. The reverse ordering leaves states that cannot be
distinguished.

**Startup reconciliation, before dispatching anything:**

1. Fetch reality: `RFC_READ_TABLE TBTCO WHERE JOBNAME LIKE '<prefix>%'` (cheap, no audit
   row) or `BAPI_XBP_JOB_COUNT` with `JOBNAME='<prefix>*'` — verified live to return the
   full 35-column header table in one call. Prefer either over `BAPI_XBP_JOB_SELECT`,
   which silently returns nothing when `USERNAME` is blank.
2. Parse each name back into `(run_id, unit_id, attempt)`.
3. Reconcile the three quadrants:
   - **journalled and present in `TBTCO`** → adopt, keep watching.
   - **journalled, absent from `TBTCO`** → the start never took effect. Re-dispatch, but
     only after the unit table confirms it is not `DONE`/`RUNNING`.
   - **present in `TBTCO`, not journalled** → the dangerous quadrant: a previous run's
     orphan, or one we started and crashed before journalling. If `R`/`Y`/`Z`: **adopt
     it, do not abort, do not re-dispatch the unit.** If `F`/`A`: fold the outcome in.
     Never abort a job you cannot attribute to your own prefix *and* run id, and even
     then prefer adoption.
4. Cross-check against the unit result table. `DONE` wins over anything the journal
   thinks. A unit in `RUNNING` with no live job is a crashed attempt — reset it. That
   reset is the scheduler's only write to the unit table, and it must be conditional on
   there being no live job for that `unit_id`.
5. Only then start dispatching, at **`C = 1`**, and re-ramp. Load conditions at restart
   are unknown by definition.

### Naming and ownership

`TBTCO-JOBNAME` is `CHAR32` (data element `BTCJOB`, domain `CHAR32`).
`JOBCOUNT` is `CHAR8` (`BTCJOBCNT`).

```
<PREFIX>_<RUNID>_<UNIT>_<ATT>       e.g.  ZSCR_7K2P4M_U0000041_03
   4       6        ≤16      2      = 31 chars, fits CHAR32
```

- **`PREFIX`** is configuration, reserved by convention, must not collide with `SAP_*`.
  Verify at startup that no job outside the scheduler's own history already uses it.
- **`RUNID`** (base36) distinguishes this plan from the previous one while staying
  greppable in SM37.
- **`UNIT`** must be `A–Z0–9_` only (job names are upper-cased in practice). If the
  natural id does not fit, use a short deterministic hash and keep the mapping in both
  the journal and the unit result table.
- **`ATT`** makes retries visible in SM37 and makes the name a natural dedup key: never
  start `(run, unit, attempt)` twice, because if the name is already in `TBTCO` the
  dispatch already happened.
- Second, independent ownership filter: set the step's `SAP_USER_NAME` to a dedicated
  batch user, so `BAPI_XBP_JOB_SELECT` with `USERNAME=<that user>` cross-checks the
  prefix.

**Important caveat: job names are not unique in SAP.** The key is
`(JOBNAME, JOBCOUNT)`, and any number of jobs may share a name. The name-as-dedup-key is
a convention the scheduler enforces by checking before dispatch, not a constraint the
system enforces for it.

---

## 8. The Go shape

### Packages

Following the layering the repo already uses (`rfc` core stays dependency-free, the
`cmd/*` set is an extractable subproject):

- **`rfc/`** — unchanged.
- **`batch/`** (new, public, depends only on `rfc` + stdlib) — a typed XBP wrapper: the
  XMI logon lifecycle, `JobRef`, `JobHeader`, `Status`, `LogLine`, `Resources`. Nothing
  about scheduling. Independently useful, and testable against a recorded conversation.
- **`sched/`** (new, public, depends on `batch` + `rfc`) — the control loop and its
  interfaces. No policy hard-coded.
- **`cmd/rfc-sched/`** (new) — CLI, `.rfc.json` config in the established shape,
  progress rendering, `--dry-run`, `--prefix`, `--max-concurrency`, `--alpha`.

### Interfaces

```go
// batch — one pinned, XMI-logged-on conversation.
type Session struct{ /* wraps *rfc.Session */ }

func Logon(ctx context.Context, c *rfc.Client, id Ident) (*Session, error) // Pin + BAPI_XMI_LOGON
func (s *Session) Close(ctx context.Context) error                        // BAPI_XMI_LOGOFF + release

func (s *Session) Open(ctx context.Context, name string, class byte) (JobRef, error)
func (s *Session) AddABAPStep(ctx context.Context, j JobRef, st Step) (stepNo int, err error)
func (s *Session) StartASAP(ctx context.Context, j JobRef, targetServer string) error
func (s *Session) StatusMany(ctx context.Context, js []JobRef) ([]Status, error) // JOBLIST_STATUS_GET
func (s *Session) Log(ctx context.Context, j JobRef) ([]LogLine, error)          // PROT_NEW='X'
func (s *Session) Resources(ctx context.Context) ([]ServerResources, error)
func (s *Session) FindByMask(ctx context.Context, mask string) ([]Header, error) // JOB_COUNT
func (s *Session) Abort(ctx context.Context, j JobRef) error
```

```go
// sched — everything swappable is an interface.
type Unit struct{ ID, Program, Variant string; Sel []Selection; Size int }

type Plan interface { Next(ctx context.Context) (Unit, bool, error); Remaining() int }

type Verifier interface {   // "did the unit actually succeed?"  — see 5.2
    Verify(ctx context.Context, u Unit, h batch.Header, log []batch.LogLine) (Outcome, error)
}
type Classifier interface { // "why did it fail, and is it worth retrying?" — see 5.3
    Classify(u Unit, h batch.Header, log []batch.LogLine) Disposition
}
type Governor interface {   // the scaling rule — see 4.2
    Target() int
    Ceiling(Capacity)
    Observe(Sample)                       // one completion: wait, run, ok/fail
    ReviewDue() (at time.Time, paused bool)
}
type Store interface {      // durable scheduler state — see §7
    Journal(context.Context, Event) error
    Load(context.Context) (State, error)
}
type Reporter interface{ Progress(Snapshot) }

type Scheduler struct {
    Dispatch *batch.Session // writes  — XBP, pinned, serialised
    Observe  *rfc.Client    // reads   — RFC_READ_TABLE, pooled, no XMI
    Gov      Governor
    Plan     Plan
    Verify   Verifier
    Class    Classifier
    Store    Store
    Report   Reporter
}
func (s *Scheduler) Run(ctx context.Context) error
```

### The XMI session question, answered from live evidence

Three facts were established against A4H:

1. **XMI logon state is bound to the connection.** Calling `BAPI_XBP_JOB_COUNT` on a
   pooled connection that had not done `BAPI_XMI_LOGON` returned
   `XM028 "Not logged on in interface XBP (function BAPI_XBP_JOB_COUNT)"`, while the same
   call on the pinned, logged-on session succeeded. Since `rfc.Client.Call` takes an
   arbitrary pooled connection, **XBP calls must never go through `Client.Call` — only
   through `Client.Pin`.** This is the single most important implementation constraint in
   this report.
2. **Concurrent XMI sessions are fine.** Three pinned sessions logged on in parallel and
   got three distinct `SESSIONID`s.
3. **There is no job-to-session affinity.** `VSP_SCHED_PROBE2`, created on one pinned
   session, was read *and deleted* from a different one. Jobs are addressed by
   `(JOBNAME, JOBCOUNT)`; the XMI session is only an interface gate.

Consequences:

- **One pinned dispatch session is the right default.** Dispatch is three round trips per
  job (open / add step / start); at LAN latencies one conversation sustains tens of
  dispatches per second, far above any sane rate. Serialising through it also *is* the
  dispatch rate limiter, in one place.
- Add a small pool of XBP sessions (2–4, each with its own logon) only if dispatch is
  measured to be the bottleneck. Never share one `*rfc.Session` across goroutines for
  concurrent calls — it is documented as safe for sequential use from one goroutine at a
  time, and a conversation carries one call at a time.
- **Keep the pinned connection alive.** `Session.Ping` on an idle timer, or a gateway or
  work-process idle timeout drops the conversation and the XMI logon with it.
- **Recovery must be asymmetric.** On `ErrTransport`: re-Pin, re-`BAPI_XMI_LOGON`, and
  retry *reads* freely. **Never blindly retry `JOB_OPEN`/`START`** — that is precisely how
  you double-dispatch. Instead, reconcile by name: look the job name up in `TBTCO` and
  decide from reality.
- **Keep observation off the XBP session entirely.** A second, unpinned `rfc.Client`
  doing `RFC_READ_TABLE` costs no audit row, does not occupy the dispatch conversation,
  and can be polled concurrently.

### Rate limits — four, all necessary

| Limit | Protects | Default |
|---|---|---|
| starts / second | the batch scheduler and the enqueue on the job tables | 2/s |
| concurrency `C` | batch work processes (the AIMD variable) | see 4.2 |
| poll interval | the app server, and your own noise | `max(2s, p50_run/10)`, cap 30 s |
| XBP calls / hour | the XMI audit log | budgeted, warn on breach |

### Progress reporting

Two channels, deliberately sharing a source of truth: the **journal event stream** (the
same events that make the state durable, so the log and the recovery state can never
disagree) and a periodic **summary** — `done/total, in-flight, C/Cmax, p50/p90 run,
p90 wait, failures by class, poisoned, ETA`. Compute ETA from the completion-rate EWMA,
not from `remaining × p50 / C`, which lies whenever `C` is moving — which is always.

---

## 9. Risks and limits

- **XMI audit-log growth — measured, and the top operational risk.** ~1 `TXMILOGRAW` row
  per XBP call even at `AUDITLEVEL = 0`. Mitigations, in order: observe over
  `RFC_READ_TABLE`; batch XBP status reads with `BAPI_XBP_JOBLIST_STATUS_GET`; keep the
  audit level at 0; agree an `RSXMILOGREORG` schedule with Basis before go-live; use a
  distinctive `EXTCOMPANY`/`EXTPRODUCT` so the rows are attributable to you.
- **XMI session leakage.** The session dies with the TCP connection, so leakage is bounded
  by connection lifetime — but a leaked *pinned* connection holds a work process for as
  long as it lives. Always defer logoff + close, and cap the pool.
- **Temporary variants.** Passing `SELINFO` to `ADD_ABAP_STEP` mints a temp variant
  (observed: `&0000000000000`). Thousands of jobs means thousands of `VARI`/`VARID` rows.
  Either use named variants created once per unit shape, or confirm that deleting the job
  reaps the variant, or plan the housekeeping.
- **Authorizations.** `S_XMI_PROD` (on the `EXTCOMPANY`/`EXTPRODUCT`/`INTERFACE` you
  pass) to log on at all; `S_BTCH_JOB` with `RELE` to release, `SHOW`/`PROT` to read the
  log, `DELE` to delete, plus `JOBGROUP`; `S_BTCH_NAM`/`S_BTCH_ADM` only if the step runs
  under a different user than the caller — `S_BTCH_ADM='Y'` is effectively cross-user job
  administration and is the line item an auditor will object to, so ask for
  `S_BTCH_NAM` scoped to one batch user instead; `S_RZL_ADM` for `TH_WPINFO`/
  `TH_SERVER_LIST`; `S_TABU_NAM`/`S_TABU_DIS` for `RFC_READ_TABLE` on `TBTCO`/`TBTCP` and
  the unit result table; `S_PROGRAM` for the unit report; `S_RFC` for the FM groups.
- **The scheduler dies mid-flight.** In-flight jobs keep running; SAP does not care that
  you left. Nothing is aborted, nothing rolls back. The exposure is the *next* start
  double-dispatching, and the defences are journal-before-dispatch, name dedup, startup
  reconciliation, and — the only one that actually closes the hole — the server-side
  claim in §6.
- **Load you can inflict.** Batch WPs are a hard, small, shared pool: 5 here, typically
  6–20 in production. Filling them starves `SAP_*` housekeeping, update processing and
  the spool. Each job also writes `TBTCO`/`TBTCP`/`TBTCS` rows and a job log, so tens of
  thousands of tiny jobs is itself a load and a table-growth problem. Prefer fewer,
  larger units: the fixed cost per job is a WP roll-in, several table writes and a log.
- **Class-A reservation.** `BTCWPCLSSA` is reserved for job class A and is unavailable to
  the class-C jobs the scheduler creates. Compute the ceiling from what class C can
  actually get.
- **A shared ceiling signal is contaminated.** `BTCWPFREE` is system-wide and
  instantaneous: it reflects other people's jobs too. Scaling to fill it means two
  schedulers oscillating against each other. AIMD's fairness property is the reason it is
  safe to steer on a shared signal at all.
- **Short dumps are not readable over RFC** by the obvious route (`SNAP` →
  `TABLE_NOT_AVAILABLE`). Detection yes (`RABAXKEY`), text no.
- **`BAPI_XBP_GET_APPLICATION_RC` does not work out of the box** (`XM232` here). Do not
  build the success test on it.
- **Second-resolution timestamps** in `TBTCO`. Fine for minute-scale units, useless for
  second-scale ones.
- **Incidental defect found in this repo.** `Client.DescribeTool` reports `maxLength` as
  **half the character length** of `CHAR` fields. Verified: `JOBNAME` → 16 where
  `BTCJOB` is `CHAR32`; `JOBCOUNT` → 4 where `BTCJOBCNT` is `CHAR8`;
  `ABAP_PROGRAM_NAME` → 20 (`PROGRAM`, `CHAR40`); `ABAP_VARIANT_NAME` → 7 (`CHAR14`);
  `SAP_USER_NAME` → 6 (`CHAR12`); and `LANGUAGE` (`LANGU`, `CHAR1`) → **0**. It looks like
  the mapper is emitting the non-Unicode byte length (`INTLEN/4`, or `LENG/2`) instead of
  the character length. An MCP client that enforces `maxLength` will reject a valid
  30-character job name — and reject *every* value for any `CHAR1` field. Worth a
  separate issue.

### Where a human must stay in the loop

1. **Approving the ceiling and the prefix** on anything that is not a sandbox.
2. **The second consecutive failed 20–30 minute review.** The scheduler pages; it does
   not decide that a production system's slowness is acceptable.
3. **Any abort or delete of a job the scheduler did not start.**
4. **Poisoned units.** The scheduler parks them with a diagnosis; a human decides whether
   the data is wrong or the program is.
5. **Job interception.** `BAPI_XBP_NEW_FUNC_CHECK` reports interception and parent/child
   both **off** here. `BAPI_XBP_NEW_FUNC_CHECK`'s `INTERCEPTION_ACTION` /
   `PARENTCHILD_ACTION` switch them on **system-wide**, which changes the behaviour of
   *everyone's* jobs (they are held in status `S` for an external scheduler to release).
   Never automate that.

---

## 10. Open questions

1. Does `SUBST_START_REPORT_IN_BATCH` get past XM262 when `IV_BATCHINSTANCE` /
   `IV_BATCHHOST` are set explicitly? XBP's mandatory `TARGET_SERVER` suggests the same
   missing-placement cause. Five-minute test; does not change the verdict.
2. **Can the unit's ABAP program be changed?** Everything in §6 hinges on this. If not,
   the guarantee drops to at-most-once and that must be stated in the config, not
   discovered in production.
3. Is the unit a report or could it be exposed as an RFC-enabled FM? If the latter, most
   of this design is unnecessary — see §3.
4. `RS_ST22_RFC`'s `TOOL` / `P_TOOL_TAB` protocol: can it fetch a dump by `RABAXKEY`?
   That would complete the failure classifier without a Z FM.
5. `BAPI_XBP_BTC_STATISTIC_GET` cost at scale — the record is large. Is 1-in-20 sampling
   enough to discriminate DB-bound from CPU-bound degradation?
6. **Multi-instance systems.** This A4H has one application server. With several, the
   ceiling and the AIMD state become per-server (or per server group), and `TARGET_SERVER`
   turns into a placement decision. `TARGET_GROUP` with an RZ12 server group would let SAP
   place jobs instead — untestable here, and worth trying before building placement logic.
7. Is job **interception** acceptable to Basis? If it is, the scheduler can become a
   gatekeeper rather than a pusher, with control over jobs it did not create. Powerful,
   and a much larger blast radius.
8. Does deleting a job reap its temporary variant, or do `VARI` rows accumulate?

---

## Appendix — raw observations worth keeping

```
XMI:      BAPI_XMI_GET_VERSIONS -> XBP 3.0, XMB 0.1, XOM 0.2, XAL 1.0
          XBP call on a non-logged-on connection -> XM028 "Not logged on in interface XBP"
          3 concurrent pinned sessions -> 3 distinct SESSIONIDs, all usable
          cross-session write proven: PROBE2 created on session A, deleted from session B

Capacity: TH_GET_PARAMETER rdisp/wp_no_btc = 5
          TH_WPINFO (15 rows): DIA/Waiting 6, BGD/Waiting 5, DIA/Running 1, UPD 1, SPO 1, UP2 1
          BAPI_XBP_GET_CURR_BP_RESOURCES: BTCWPTOTAL " 5"  BTCWPFREE " 5" -> " 3" -> " 4" -> " 5"
                                          (leading blanks are real; trim before parsing)

Lifecycle (VSP_SCHED_PROBE1/2, report RSWAITSEC, param WAITTIME, TARGET_SERVER vhcala4hci_A4H_00):
          t+0s   both R,  BTCWPFREE 3
          t+8s   both R,  BTCWPFREE 3
          abort PROBE2 -> t+11s  PROBE1 R, PROBE2 A,  BTCWPFREE 4
          t+56s  PROBE1 F,        PROBE2 A,  BTCWPFREE 5
          BAPI_XBP_JOB_STATUS_CHECK PROBE1: ACTUAL_STATUS F == STATUS_ACCORDING_TO_DB F

Job log (PROT_NEW='X', table JOB_PROTOCOL_NEW, 15 cols incl. RABAXKEY):
          00/516 S  "Job VSP_SCHED_PROBE1 23415800 started"
          00/550 S  "Step 001 started (program RSWAITSEC, variant &0000000000000, user ID CLAUDE)"
          00/517 S  "Job finished"                                  <- success marker
          00/554 A  "Job canceled due to terminated process or program; see the system log"

Statistics (BAPI_XBP_BTC_STATISTIC_GET, PROBE1): PROCTI 45006731us  RESPTI 45046962us
          CPUTI 10000  DBREQTIME 39183  MAXROLL 7980784  MEMSUM 491273  ROLLINCNT 2
          REPORT RSWAITSEC  ACCOUNT CLAUDE  TASKTYPE BA  TARGETTIMEZONE UTC

Selection: BAPI_XBP_JOB_SELECT JOBNAME='VSP_SCHED_PROBE2' USERNAME=''  -> [] , ERROR_CODE 0  (!)
                               JOBNAME='VSP_SCHED_PROBE2' USERNAME=CLAUDE -> 1 job
                               JOBNAME='VSP_SCHED_PROBE*' USERNAME='*'    -> 2 jobs
           BAPI_XBP_JOB_COUNT  JOBNAME='VSP_SCHED_PROBE*'  -> NUMBER_OF_JOBS 2 + JOB_TABLE[35]

TBTCO:     domain of STATUS is CHAR1 with NO fixed values
           5000 most recent rows: F 4925, Z 65, A 9, S 1
           the 65 Z rows are SAP_* / RSBK* jobs with SDLSTRTDT 2017-05..2017-10 and no STRTDATE
           poll verified: RFC_READ_TABLE TBTCO fields JOBNAME,JOBCOUNT,STATUS,SDLSTRTDT,SDLSTRTTM,
                          STRTDATE,STRTTIME,ENDDATE,ENDTIME,REAXSERVER,WPNUMBER
                          where "JOBNAME LIKE 'VSP_SCHED_%'"

Audit:     TXMILOGRAW total rows on the system: 69;  written by this probe session: 31
           (EXTPRODUCT LIKE 'SCHEDPROBE%'), all at AUDITLEVEL 0

Not remote-enabled (TFDIR-FMODE blank): JOB_OPEN, JOB_SUBMIT, JOB_CLOSE, BP_JOB_SELECT,
           SPBT_INITIALIZE, SPBT_GET_PP_DESTINATION, SPBT_GET_CURR_RESOURCE_INFO,
           BP_JOBLOG_READ, RS_ST22_GET_DUMPS (and the rest of the RS_ST22_GET_* family)
Not readable via RFC_READ_TABLE: SNAP (cluster table) -> TABLE_NOT_AVAILABLE / DA131
```
