# Surface inventory: src/pool/, src/destination/, src/diagnostics/

> Mechanical inventory of open-rfc @ commit 847036d, generated as porting input. Every claim cites path:line. See ../provenance.md.

Conventions used below:

- **(code)** — quoted from source, verbatim.
- **(comment)** — quoted from a source comment; it is an authorial claim, not
  executable behaviour.
- **(test)** — quoted from a test name or assertion.
- **INFERRED:** — not written anywhere in the source; used sparingly and marked.
- Signatures in tables are the source text with interior line breaks collapsed
  to single spaces. Nothing else is altered — identifiers, defaults, modifiers
  and punctuation are as written.
- All paths are relative to the open-rfc checkout root.

---

## src/pool/connection-pool-runtime.ts

2117 lines (`wc -l`). Imports only `randomUUID` from `node:crypto`
(`src/pool/connection-pool-runtime.ts:1`) and the diagnostics reporter
(`src/pool/connection-pool-runtime.ts:3-7`).

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `ConnectionPoolRuntimeState` | type alias | `export type ConnectionPoolRuntimeState = \| "open" \| "retiring" \| "closing" \| "closed";` | `src/pool/connection-pool-runtime.ts:9-13` |
| `ConnectionPoolRuntimeErrorCode` | type alias | `export type ConnectionPoolRuntimeErrorCode = \| "POOL_CLOSED" \| "POOL_OVERLOADED" \| "ACQUIRE_TIMEOUT" \| "ACQUIRE_ABORTED" \| "WRONG_POOL" \| "STALE_LEASE" \| "DOUBLE_RELEASE" \| "POOL_LEASE_BUSY" \| "ACTIVE_LEASE" \| "LIFECYCLE_TIMEOUT" \| "POOL_SHUTDOWN_TIMEOUT";` | `src/pool/connection-pool-runtime.ts:15-26` |
| `ConnectionPoolRuntimeError` | class | `export class ConnectionPoolRuntimeError extends Error { readonly code: ConnectionPoolRuntimeErrorCode; constructor(code: ConnectionPoolRuntimeErrorCode, message: string) }` | `src/pool/connection-pool-runtime.ts:28-36` |
| `ConnectionPoolScheduledTask` | interface | `export interface ConnectionPoolScheduledTask { cancel(): void; }` | `src/pool/connection-pool-runtime.ts:38-40` |
| `ConnectionPoolScheduler` | interface | `export interface ConnectionPoolScheduler { now(): number; schedule(delayMs: number, callback: () => void): ConnectionPoolScheduledTask; }` | `src/pool/connection-pool-runtime.ts:43-46` |
| `ConnectionPoolLifecycleOperation` | type alias | `export type ConnectionPoolLifecycleOperation = \| "create" \| "validate" \| "reset" \| "destroy";` | `src/pool/connection-pool-runtime.ts:48-52` |
| `ConnectionPoolLifecycleContext` | interface | `export interface ConnectionPoolLifecycleContext { readonly signal: AbortSignal; readonly operation: ConnectionPoolLifecycleOperation; readonly operationId: number; readonly timeoutMs: number; }` | `src/pool/connection-pool-runtime.ts:54-59` |
| `ConnectionCreationContext` | interface | `export interface ConnectionCreationContext extends ConnectionPoolLifecycleContext { readonly creationId: number; }` | `src/pool/connection-pool-runtime.ts:61-64` |
| `ConnectionPoolFactory<T>` | interface | `export interface ConnectionPoolFactory<T extends object> { create(context: ConnectionCreationContext): T \| PromiseLike<T>; destroy(resource: T, context: ConnectionPoolLifecycleContext): void \| PromiseLike<void>; validate?(resource: T, context: ConnectionPoolLifecycleContext): boolean \| PromiseLike<boolean>; reset?(resource: T, context: ConnectionPoolLifecycleContext): void \| PromiseLike<void>; }` | `src/pool/connection-pool-runtime.ts:66-83` |
| `ConnectionPoolRuntimeOptions<T>` | interface | `export interface ConnectionPoolRuntimeOptions<T extends object> { readonly factory: ConnectionPoolFactory<T>; readonly maxConnections: number; readonly maxWaiters: number; readonly acquireTimeoutMs: number; readonly lifecycleTimeoutMs?: number; readonly shutdownTimeoutMs?: number; readonly lifecycleScheduler?: ConnectionPoolScheduler; readonly lowWater?: number; readonly idleHigh?: number; readonly validateOnCheckout?: boolean; readonly resetOnRelease?: boolean; readonly scheduler?: ConnectionPoolScheduler; readonly diagnostics?: RfcDiagnosticEmitter; }` | `src/pool/connection-pool-runtime.ts:85-107` |
| `ConnectionPoolAcquireOptions` | interface | `export interface ConnectionPoolAcquireOptions { readonly timeoutMs?: number; readonly signal?: AbortSignal; }` | `src/pool/connection-pool-runtime.ts:109-112` |
| `ConnectionPoolReleaseOptions` | interface | `export interface ConnectionPoolReleaseOptions { readonly reusable?: boolean; readonly idleHigh?: number; }` | `src/pool/connection-pool-runtime.ts:114-123` |
| `ConnectionPoolShutdownOptions` | interface | `export interface ConnectionPoolShutdownOptions { readonly timeoutMs?: number; }` | `src/pool/connection-pool-runtime.ts:125-127` |
| `ConnectionPoolLease<_T>` | interface | `export interface ConnectionPoolLease<_T extends object> { readonly poolId: number; readonly generation: number; }` | `src/pool/connection-pool-runtime.ts:141-144` |
| `ConnectionPoolMonitor` | interface | 26 readonly fields: `poolId`, `state`, `maxConnections`, `maxWaiters`, `lifecycleTimeoutMs`, `shutdownTimeoutMs`, `lowWater`, `idleHigh`, `connections`, `idle`, `leased`, `creating`, `validating`, `resetting`, `closing`, `waiting`, `lastLeaseGeneration`, `leasesIssued`, `creationFailures`, `creationAborts`, `healthFailures`, `resetFailures`, `destroyFailures`, `lifecycleTimeouts`, `shutdownTimeouts`, `failed` | `src/pool/connection-pool-runtime.ts:146-174` |
| `ConnectionPoolRuntime<T>` | class | `export class ConnectionPoolRuntime<T extends object>` | `src/pool/connection-pool-runtime.ts:407` |

Public members of `ConnectionPoolRuntime<T>`:

| Member | Signature (verbatim) | Citation |
|---|---|---|
| `id` | `get id(): number` (returns `this.#poolId`) | `src/pool/connection-pool-runtime.ts:629-631` |
| `acquireOne` | `acquireOne( options: ConnectionPoolAcquireOptions = {}, ): Promise<ConnectionPoolLease<T>>` | `src/pool/connection-pool-runtime.ts:633-637` |
| `acquire` | `acquire( count = 1, options: ConnectionPoolAcquireOptions = {}, ): Promise<readonly ConnectionPoolLease<T>[]>` | `src/pool/connection-pool-runtime.ts:639-642` |
| `withActiveLease` | `async withActiveLease<R>( lease: ConnectionPoolLease<T>, operation: (resource: T) => R \| PromiseLike<R>, ): Promise<R>` | `src/pool/connection-pool-runtime.ts:797-800` |
| `resetActiveLease` | `async resetActiveLease(lease: ConnectionPoolLease<T>): Promise<void>` | `src/pool/connection-pool-runtime.ts:823` |
| `release` | `async release( lease: ConnectionPoolLease<T>, options: ConnectionPoolReleaseOptions = {}, ): Promise<void>` | `src/pool/connection-pool-runtime.ts:861-864` |
| `close` | `close(options: ConnectionPoolShutdownOptions = {}): Promise<void>` — body `return this.#beginShutdown("closing", options);` | `src/pool/connection-pool-runtime.ts:988-990` |
| `retire` | `retire(options: ConnectionPoolShutdownOptions = {}): Promise<void>` — body `return this.#beginShutdown("retiring", options);` | `src/pool/connection-pool-runtime.ts:997-999` |
| `drain` | `drain(options: ConnectionPoolShutdownOptions = {}): Promise<void>` — body `return this.close(options);` (pure alias for `close`) | `src/pool/connection-pool-runtime.ts:1001-1003` |
| `monitor` | `monitor(): ConnectionPoolMonitor` | `src/pool/connection-pool-runtime.ts:1129` |

Non-exported module constants that are load-bearing for the port:

- `const MAX_TIMER_MS = 2_147_483_647;` — `src/pool/connection-pool-runtime.ts:246`
- `const MAX_EARLY_TIMEOUT_REARMS = 64;` — `src/pool/connection-pool-runtime.ts:247`
- `let nextPoolId = 1;` — module-global monotonic pool id; `#poolId = nextPoolId++` — `src/pool/connection-pool-runtime.ts:400`, `:408`

### Configuration options and defaults

| Name | Default (verbatim) | Effect per source | Citation |
|---|---|---|---|
| `maxConnections` | *required* | `integer(maxConnections, 1, Number.MAX_SAFE_INTEGER, "maxConnections")`. (comment) `/** Hard physical-capacity limit. This is deliberately not idleHigh. */` | `src/pool/connection-pool-runtime.ts:498-503`, `:87` |
| `maxWaiters` | *required* | `integer(maxWaiters, 1, Number.MAX_SAFE_INTEGER, "maxWaiters")`. (comment) `/** Maximum number of pending acquire requests, including the FIFO head. */` | `src/pool/connection-pool-runtime.ts:504-509`, `:89` |
| `acquireTimeoutMs` | *required* | `timeout(acquireTimeoutMs, "acquireTimeoutMs")`; `timeout` requires `Number.isFinite(value)` and `1..MAX_TIMER_MS` | `src/pool/connection-pool-runtime.ts:510-513`, `:299-304` |
| `lifecycleTimeoutMs` | `lifecycleTimeoutMs === undefined ? this.#acquireTimeoutMs : lifecycleTimeoutMs` | Bound for create/validate/reset/destroy callbacks | `src/pool/connection-pool-runtime.ts:514-519`, `:92-93` |
| `shutdownTimeoutMs` | `shutdownTimeoutMs === undefined ? this.#lifecycleTimeoutMs : shutdownTimeoutMs` | Bound for close/retire convergence | `src/pool/connection-pool-runtime.ts:520-525`, `:94-95` |
| `lowWater` | `lowWater === undefined ? 0 : lowWater` (range `0..this.#maxConnections`) | Desired idle floor while open | `src/pool/connection-pool-runtime.ts:526-531`, `:98-99` |
| `idleHigh` | `idleHigh === undefined ? this.#maxConnections : idleHigh` (range `0..this.#maxConnections`) | Max recycled idle resources | `src/pool/connection-pool-runtime.ts:532-537`, `:100-101` |
| (cross-check) | — | `if (this.#lowWater > this.#idleHigh) { throw new RangeError("lowWater must not exceed idleHigh"); }` | `src/pool/connection-pool-runtime.ts:538-540` |
| `validateOnCheckout` | `booleanOption(validateOnCheckout, false, "validateOnCheckout")` | Requires `factory.validate` when true | `src/pool/connection-pool-runtime.ts:541-545`, `:552-557` |
| `resetOnRelease` | `booleanOption(resetOnRelease, false, "resetOnRelease")` | Requires `factory.reset` when true | `src/pool/connection-pool-runtime.ts:546-550`, `:558-560` |
| `scheduler` | `configuredScheduler === undefined ? defaultScheduler : configuredScheduler` | Waiter deadlines | `src/pool/connection-pool-runtime.ts:576-579` |
| `lifecycleScheduler` | `configuredLifecycleScheduler === undefined ? defaultScheduler : configuredLifecycleScheduler` | Lifecycle + shutdown deadlines | `src/pool/connection-pool-runtime.ts:599-602`, `:96-97` |
| `defaultScheduler` | `Object.freeze({ now: () => performance.now(), schedule(delayMs, callback) { const handle = setTimeout(callback, delayMs); return Object.freeze({ cancel: () => clearTimeout(handle) }); }, })` | — | `src/pool/connection-pool-runtime.ts:279-285` |
| `diagnostics` | `undefined` → `createDeferredRfcDiagnosticReporter(diagnostics)` returns `undefined` | (comment) `/** Optional bounded structured diagnostics; never receives resources. */` | `src/pool/connection-pool-runtime.ts:551`, `:105-106`, `src/diagnostics/structured-diagnostics.ts:349` |
| `acquire` `count` | `count = 1` | `integer(count, 1, this.#maxConnections, "acquire count")` | `src/pool/connection-pool-runtime.ts:640`, `:647` |
| `acquire` `timeoutMs` | `configuredTimeoutMs === undefined ? defaultTimeoutMs : configuredTimeoutMs` where `defaultTimeoutMs` is `this.#acquireTimeoutMs` | `timeout(..., "acquire timeoutMs")` | `src/pool/connection-pool-runtime.ts:352-357`, `:648-651` |
| `release` `reusable` | `booleanOption(options.reusable, true, "release reusable")` | (comment) `/** False marks a connection uncertain and evicts it without resetting it. */` | `src/pool/connection-pool-runtime.ts:871`, `:115` |
| `release` `idleHigh` | `undefined` → no per-handoff cap; else `integer(options.idleHigh, 0, this.#maxConnections, "release idleHigh")` | Stored as `physical.recycleIdleHigh` and consumed by `#trimIdleHigh`/`#returnToPool` | `src/pool/connection-pool-runtime.ts:872-879`, `:900`, `:1823-1840`, `:1990` |
| shutdown `timeoutMs` | `configuredTimeoutMs === undefined ? this.#shutdownTimeoutMs : configuredTimeoutMs` | — | `src/pool/connection-pool-runtime.ts:1014-1020` |

### Errors

Codes are `ConnectionPoolRuntimeError.code`; message strings are verbatim.

| Message/code (verbatim) | Trigger | Citation |
|---|---|---|
| `POOL_CLOSED` / `"connection pool is closing or closed"` | acquire while `#state !== "open"`; `withActiveLease` while closing/closed; `resetActiveLease` while not open; arming/handling waiter timeout while not open | `:653`, `:665`, `:676`, `:687`, `:737`, `:806`, `:826`, `:1434`, `:1458`, `:1480`, `:1515` |
| `POOL_CLOSED` / `"connection pool stopped while waiting"` | Waiters rejected at `#beginShutdown` | `:1069` |
| `POOL_CLOSED` / `"connection pool stopped during creation"` | In-flight creations aborted at `#beginShutdown` | `:1075` |
| `POOL_CLOSED` / `"connection pool stopped during lifecycle work"` | `validating`/`resetting` lifecycles aborted at `#beginShutdown` | `:1087` |
| `POOL_CLOSED` / `"connection pool closed during checkout"` | Pool left `open` between waiter cleanup and lease issue | `:1707-1715` |
| `POOL_OVERLOADED` / `"connection pool waiter limit reached"` | `this.#waiters.length >= this.#maxWaiters`, checked twice (before and after reading the clock) | `:658-660`, `:680-682` |
| `ACQUIRE_ABORTED` / `"connection acquire was aborted"` | signal already aborted at admission; abort listener fires; signal observed aborted after registration | `:656`, `:703`, `:742` |
| `ACQUIRE_ABORTED` / `"connection creation is no longer needed"` | `#reconcileCreations` aborts surplus creation | `:1793-1796` |
| `ACQUIRE_TIMEOUT` / `"connection acquire timed out"` | timer fires at/after deadline; or head found overdue in `#waiterCannotDispatch` | `:1536-1541`, `:1596-1599` |
| `LIFECYCLE_TIMEOUT` / `` `connection ${operation} exceeded ${timeoutMs}ms` `` | `#startLifecycle` deadline expiry; increments `#lifecycleTimeouts`; calls `controller.abort(error)` then `reject(error)` | `:1383-1395` |
| `POOL_SHUTDOWN_TIMEOUT` / `` `connection pool shutdown exceeded ${timeoutMs}ms` `` | Shutdown deadline expiry; increments `#shutdownTimeouts`; `#failClose(error)` | `:1049-1060` |
| `WRONG_POOL` / `"lease does not belong to this pool"` | lease not an object, or `lease.poolId !== this.#poolId` | `:2024-2030` |
| `STALE_LEASE` / `"lease token is not recognized"` | lease absent from `#knownLeases` | `:2031-2034` |
| `STALE_LEASE` / `"lease generation is stale"` | inactive record whose `physical.generation !== record.generation` | `:2035-2038` |
| `DOUBLE_RELEASE` / `"lease was already released"` | inactive record with matching generation | `:2039` |
| `POOL_LEASE_BUSY` / `"connection lease already has an active operation"` | `withActiveLease` or `resetActiveLease` when `record.activeOperations !== 0` | `:808-813`, `:831-836` |
| `ACTIVE_LEASE` / `"connection lease has an active operation"` | `release` when `leaseRecord.activeOperations !== 0` | `:881-886` |
| `RangeError("acquire deadline exceeds the finite clock range")` | `!Number.isFinite(now + acquireOptions.timeoutMs)` | `:672-674` |
| `RangeError("runtime deadline exceeds the finite clock range")` | `!Number.isFinite(now + timeoutMs)` in `#startRuntimeDeadline` | `:1237-1239` |
| `Error("connection pool scheduler clock must be finite and monotonic")` | `#readClock` sees non-finite or regressing `now()` | `:1195-1202` |
| `Error("connection pool lifecycle scheduler clock must be finite and monotonic")` | `#readLifecycleClock`, same rule | `:1204-1213` |
| `Error("connection pool scheduler fired a lifecycle deadline without bounded progress")` | early lifecycle-deadline callback with `remaining >= scheduledRemaining` or `earlyRearms > MAX_EARLY_TIMEOUT_REARMS` | `:1304-1317` |
| `Error("connection pool scheduler fired acquire timeout before its deadline without bounded progress")` | same rule on the waiter timeout path | `:1490-1502` |
| `TypeError("connection factory must create a non-null object")` | `create` resolved a non-object/non-function or `null` | `:1885-1893` |
| `Error("connection factory returned a resource already owned by the pool")` | `this.#resourceRecords.has(resource)` | `:1900-1907` |
| `Error("connection factory failed with a non-Error value", { cause: error })` | `creationError()` wrapping a non-`Error` rejection before it reaches a waiter | `:392-398` |
| `AggregateError([error, destroyError], "connection reset and destruction both failed", { cause: error })` | `resetOnRelease` reset failed **and** the follow-up destroy failed | `:922-928` |
| `TypeError("connection pool options must be an object")` | non-object constructor options | `:460-462` |
| `TypeError("connection pool factory must be an object")` | non-object `factory` | `:476-478` |
| `TypeError("connection pool factory requires create and destroy")` | missing `create` or `destroy` | `:483-488` |
| `TypeError("factory.validate must be a function")` / `TypeError("factory.reset must be a function")` | present but non-function | `:489-497` |
| `TypeError("validateOnCheckout requires factory.validate")` / `TypeError("resetOnRelease requires factory.reset")` | flag set without the hook | `:552-560` |
| `TypeError("scheduler requires now and schedule")` / `TypeError("lifecycleScheduler requires now and schedule")` | scheduler shape check | `:580-593`, `:603-617` |
| `TypeError("scheduler must return a cancelable task")` | `bindScheduledTask` on a task without a callable `cancel` | `:367-383` |
| `TypeError("acquire options must be an object")` / `TypeError("acquire signal must be an AbortSignal")` | `canonicalAcquireOptions` | `:322-324`, `:331-342` |
| `TypeError("release options must be an object")` | non-object release options | `:868-870` |
| `TypeError("shutdown options must be an object")` | non-object shutdown options | `:1011-1013` |
| `TypeError("lease operation must be a function")` | `withActiveLease` callback not a function | `:801-803` |
| `TypeError("connection pool factory does not provide reset")` | `resetActiveLease` with no `factory.reset` | `:828-830` |
| `RangeError(`${path} must be an integer in ${minimum}..${maximum}`)` | generic `integer()` helper | `:287-297` |
| `RangeError(`${path} must be finite and in 1..${MAX_TIMER_MS}`)` | generic `timeout()` helper | `:299-304` |
| `TypeError(`${path} must be a boolean`)` | generic `booleanOption()` helper | `:306-316` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** A monotonic scheduling boundary which deterministic tests can replace. */` | `src/pool/connection-pool-runtime.ts:42` |
| `/** Must stop touching the resource after `context.signal` aborts. */` (on `destroy`, `validate`, `reset`) | `:69`, `:73`, `:78` |
| `/** Hard physical-capacity limit. This is deliberately not idleHigh. */` | `:87` |
| `/** Maximum number of pending acquire requests, including the FIFO head. */` | `:89` |
| `/** Finite bound for create/validate/reset/destroy callbacks. */` | `:92` |
| `/** Finite bound for close/retire convergence. */` | `:94` |
| `/** Optional bounded structured diagnostics; never receives resources. */` | `:105` |
| `/** False marks a connection uncertain and evicts it without resetting it. */` | `:115` |
| `/** Optional cap for this recycle handoff. It follows the physical resource through checkout validation so a canceled waiter cannot return it above the caller's retention limit. */` | `:117-122` |
| `/** An immutable ownership token. Physical resources are deliberately not exposed; all use must pass through `withActiveLease()`. */` | `:137-140` |
| `/** Capacity slots in exactly one of the six physical states below. */` | `:155` |
| `// Timing evidence must never affect connector behavior.` | `:255` |
| `/** A bounded generic connection pool state machine. It deliberately has no compatibility-facade behavior; archived idle-high semantics can be adapted without weakening the independent maxConnections limit. */` | `:402-406` |
| `// The scheduler is an external boundary and may reenter acquire(). Do not let the outer request exceed the bounded FIFO after it returns.` | `:678-679` |
| `/** The only resource-use boundary. Callers must not retain the callback-scoped resource; ownership and single-flight tracking end when the Promise settles. */` | `:793-796` |
| `/** Run the pool factory reset under the configured finite lifecycle bound. */` | `:822` |
| `/** Retire one immutable destination generation. New acquisition stops, idle resources close immediately, and already-leased resources remain usable until their final release. */` | `:992-996` |
| `// The physical record retains the failure and shutdown/release surfaces it.` | `:1228` |
| `// A stale task cannot regain ownership through its cancellation hook.` | `:1277` |
| `// Cancellation cannot change a completed lifecycle operation.` | `:1351` |
| `// A reentrant scheduler cannot retain a stale or settled task.` | `:1453` |
| `// A scheduler is an external boundary and may invoke its callback synchronously or early. Recursive rearming can overflow the stack, while a callback with no clock progress can strand the waiter after that overflow unwinds. Invalidate this task and rearm from a microtask under a finite broken-scheduler budget.` | `:1485-1489` |
| `// #armTimeout advances the generation before consulting the scheduler, so a thrown scheduling hook leaves no live task for that newer generation. Any still-active waiter must settle.` | `:1522-1524` |
| `// A broken scheduler must not retain an already-settled waiter.` | `:1555` |
| `// Cleanup remains deterministic for a broken signal implementation.` | `:1608` |
| `// Reading the external scheduler may reenter the pool and settle or replace the FIFO head. Only the original active head may be committed.` | `:1585-1586` |
| `// Validation errors are health failures and never expose the resource.` | `:1742` |
| `// Keep the handoff cap while a waiter owns the validation race. Once the resource reaches stable idle, the one-shot recycle policy is fulfilled and explicit ready(n) may again exceed the public high.` | `:1994-1996` |

### Behaviour facts asserted by tests

`test/connection-pool-runtime.test.ts` — 49 `test(...)` cases. Selected, each
with its verbatim test name.

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| Constructor rejects `maxConnections: 0`, `maxWaiters: 0`, `acquireTimeoutMs: 0`, `lifecycleTimeoutMs: 0`, `shutdownTimeoutMs: Number.POSITIVE_INFINITY`, `lowWater: null`, `idleHigh: null`, `scheduler: null`, `lifecycleScheduler: null`; `lowWater: 2, idleHigh: 1` throws `/lowWater must not exceed idleHigh/`; `validateOnCheckout: true` without `validate` throws `/factory\.validate/`; `resetOnRelease: true` without `reset` throws `/factory\.reset/`. A pre-aborted signal rejects `ACQUIRE_ABORTED` with `creates === 0` and `waiting === 0`. | `"validates finite independent resource-policy boundaries"` | `test/connection-pool-runtime.test.ts:217`, assertions `:223-361` |
| Every constructor option and external method is read/snapshotted exactly once (tracked via `reads`, `calls`, `bindReads`, and a `bind` accessor trap). | `"snapshots every constructor option and external method exactly once"` | `test/connection-pool-runtime.test.ts:365-396` |
| `maxConnections` is independent of `idleHigh` recycling. | `"separates hard capacity from archived idle-high recycling semantics"` | `test/connection-pool-runtime.test.ts:596` |
| Per-release `idleHigh` survives a validating/aborting waiter. | `"preserves a per-release idle-high cap when an aborting waiter is validating another resource"` | `test/connection-pool-runtime.test.ts:670` |
| Waiters are FIFO; overload rejects without exceeding capacity. | `"serves waiters FIFO and rejects overload without exceeding capacity"` | `test/connection-pool-runtime.test.ts:829` |
| Overdue waiters are rejected before dispatch when timer delivery is late. | `"rejects an overdue waiter before dispatch when timer delivery is delayed"` | `test/connection-pool-runtime.test.ts:888` |
| The deadline is re-checked after async checkout validation. | `"rechecks the deadline after asynchronous checkout validation"` | `test/connection-pool-runtime.test.ts:931` |
| Mutable acquire options are snapshotted once; the registration/abort race is closed. | `"snapshots mutable acquire options once and closes the registration abort race"` | `test/connection-pool-runtime.test.ts:976` |
| A synchronous early timer with no clock progress settles in bounded work. | `"rejects a synchronous early timer without clock progress in bounded work"` | `test/connection-pool-runtime.test.ts:1318` |
| A waiter never receives a lease if its own cleanup reentrantly closed the pool. | `"never leases after waiter cleanup reentrantly closes the pool"` | `test/connection-pool-runtime.test.ts:1463` |
| `acquire(N)` is atomic; surplus creation is aborted; capacity recovers. | `"acquire(N) is atomic, aborts surplus creation, and recovers capacity"` | `test/connection-pool-runtime.test.ts:1650` |
| Unhealthy checkout candidates are replaced without exposing a partial lease. | `"replaces unhealthy checkout candidates without exposing a partial lease"` | `test/connection-pool-runtime.test.ts:1699` |
| Wrong-pool, double and stale lease tokens are rejected by generation. | `"rejects wrong-pool, double, and stale lease tokens by generation"` | `test/connection-pool-runtime.test.ts:1845` |
| Release disposition is validated *before* lease ownership is consumed. | `"validates release disposition before consuming lease ownership"` | `test/connection-pool-runtime.test.ts:1869` |
| A resource is never released while a lease operation is active. | `"never releases a resource while a lease operation is active"` | `test/connection-pool-runtime.test.ts:1890` |
| An overlapping operation is rejected *before* it is invoked. | `"rejects an overlapping operation before invoking it"` | `test/connection-pool-runtime.test.ts:1910` |
| `close` waits for an active leased operation and destroys only after release. | `"close waits for an active leased operation and destroys only after release"` | `test/connection-pool-runtime.test.ts:2055` |
| Monitor snapshots reconcile creating/validating/resetting/closing counts. | `"monitor snapshots reconcile creating, validating, resetting, and closing states"` | `test/connection-pool-runtime.test.ts:2158` |
| 520 create-use-evict cycles keep capacity bounded and lease generations monotonic. | `"survives 520 create-use-evict cycles with bounded capacity and monotonic leases"` | `test/connection-pool-runtime.test.ts:2203` |
| 640 seeded mixed transitions preserve pool invariants. | `"preserves pool invariants across 640 seeded mixed state transitions"` | `test/connection-pool-runtime.test.ts:2244` |
| Non-cooperative `create` times out without freeing uncertain capacity. | `"times out a non-cooperative create without freeing uncertain capacity"` | `test/connection-pool-runtime.test.ts:2337` |
| Never-settling `validate`/`reset` time out with abort signals. | `"times out never-settling validate and reset callbacks with abort signals"` | `test/connection-pool-runtime.test.ts:2377` |
| `destroy` rejection is surfaced and physical capacity stays quarantined. | `"surfaces destroy rejection and retains quarantined physical capacity"` | `test/connection-pool-runtime.test.ts:2427` |
| `destroy`/shutdown timeout does not claim physical closure. | `"times out destroy and pool shutdown without claiming physical closure"` | `test/connection-pool-runtime.test.ts:2464` |
| `close` has a finite deadline while a caller retains a lease. | `"close has a finite deadline while a caller retains a lease"` | `test/connection-pool-runtime.test.ts:2497` |
| Retirement rejects new work but permits pinned leases through final release. | `"retirement rejects new work but permits pinned leases through final release"` | `test/connection-pool-runtime.test.ts:2525` |

---

## src/destination/configuration-generation.ts

477 lines. Imports `AsyncLocalStorage` (`:1`) and `createHash` from
`node:crypto` (`:2`).

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `DestinationLaneFactory<T>` | interface | `export interface DestinationLaneFactory<T> { open(): Promise<T>; dispose(connection: T): void \| Promise<void>; retire?(): void \| Promise<void>; }` | `src/destination/configuration-generation.ts:10-15` |
| `DestinationIdentityInput` | interface | `readonly destinationId, endpointId, systemId, client, release, metadataGeneration, language, applicationPrincipalId, repositoryPrincipalId: string` | `src/destination/configuration-generation.ts:17-29` |
| `DestinationSafeIdentity` | interface | `readonly destinationId, systemId, client, release, metadataGeneration, language, structuralBackendKey: string; readonly applicationCapability: MetadataCapabilityKey; readonly repositoryCapability: MetadataCapabilityKey;` | `src/destination/configuration-generation.ts:31-41` |
| `DestinationConfiguration` | interface | `export interface DestinationConfiguration { readonly generationId: string; readonly repositoryMode: MetadataRepositoryMode; readonly identity: DestinationSafeIdentity; }` | `src/destination/configuration-generation.ts:43-47` |
| `DestinationConfigurationGenerationOptions<A, R>` | interface | `readonly generationId: string; readonly repositoryMode: MetadataRepositoryMode; readonly identity: DestinationIdentityInput; readonly applicationFactory: DestinationLaneFactory<A>; readonly repositoryFactory: DestinationLaneFactory<R>;` | `src/destination/configuration-generation.ts:49-55` |
| `DestinationLaneMonitor` | interface | `readonly attempts: number; readonly succeeded: number; readonly failed: number; readonly inFlight: number;` | `src/destination/configuration-generation.ts:57-62` |
| `DestinationGenerationMonitor` | interface | `readonly generationId: string; readonly state: "active" \| "retiring" \| "retired"; readonly application: DestinationLaneMonitor; readonly repository: DestinationLaneMonitor;` | `src/destination/configuration-generation.ts:64-69` |
| `DestinationConfigurationGeneration<A, R>` | class | `export class DestinationConfigurationGeneration<A, R>` | `src/destination/configuration-generation.ts:254` |

Public members:

| Member | Signature (verbatim) | Citation |
|---|---|---|
| `configuration` | `readonly configuration: DestinationConfiguration;` — reassigned as a frozen non-writable, non-configurable own property | `:255`, `:300-310` |
| `openApplication` | `openApplication(): Promise<A>` — `return this.#openLane(this.#applicationFactory, this.#applicationMonitor);` | `:315-317` |
| `openRepository` | `openRepository(): Promise<R>` — `return this.#openLane(this.#repositoryFactory, this.#repositoryMonitor);` | `:319-321` |
| `retire` | `retire(): Promise<void>` | `:323` |
| `monitor` | `monitor(): DestinationGenerationMonitor` | `:405` |

Non-exported but observable: `class DestinationLateOpenDisposalError extends Error`
with `this.name = "DestinationLateOpenDisposalError"` and message
`` `destination generation ${generationId} could not dispose a late-opened connection` ``
— `src/destination/configuration-generation.ts:84-92`.

### Configuration options and defaults

| Name | Default (verbatim) | Effect per source | Citation |
|---|---|---|---|
| `generationId` | *required* | `safeIdentity(options.generationId, "generationId")` | `:272` |
| `repositoryMode` | *required* | `if (!Object.values(MetadataRepositoryMode).includes(repositoryMode))` → `RangeError` | `:273-278` |
| `identity` | *required* | `snapshotIdentity` runs `safeIdentity` on all nine fields and freezes | `:165-191`, `:279` |
| `applicationFactory` / `repositoryFactory` | *required* | `bindFactory` captures `open`, `dispose`, optional `retire` via `Reflect.apply` | `:208-239`, `:280-281` |
| identity field bounds | — | `typeof value !== "string" \|\| value.length < 1 \|\| value.length > 512 \|\| /[\u0000-\u001f\u007f]/u.test(value)` → `RangeError(`${field} must contain 1..512 characters without controls`)` | `:151-163` |

There are no numeric/timeout defaults in this file.

### Errors

| Message/code (verbatim) | Trigger | Citation |
|---|---|---|
| `TypeError("destination generation options must be an object")` | non-object options | `:266-268` |
| `RangeError(`${field} must contain 1..512 characters without controls`)` | any identity field or `generationId` out of bounds / containing C0 or DEL | `:151-161` |
| `TypeError("destination identity must be an object")` | non-object identity | `:168-170` |
| `RangeError(`unsupported metadata repository mode ${String(repositoryMode)}`)` | mode not in the enum | `:274-277` |
| `TypeError("destination lane factory must be an object")` | factory not object/function or null | `:209-214` |
| `TypeError("destination lane factory must provide open()")` / `...must provide dispose()"` / `"destination lane factory retire must be a function"` | shape checks | `:218-226` |
| `Error(`destination generation ${this.configuration.generationId} is retired`)` | `#openLane` while `#state !== "active"` | `:418-422` |
| `Error(`destination generation ${this.configuration.generationId} retired while opening`)` | open resolved after the state left `active` | `:438-440` |
| `DestinationLateOpenDisposalError` | `factory.dispose` of a late-opened connection threw | `:433-437`, `:84-92` |
| `AggregateError(failures, `destination generation ${this.configuration.generationId} retirement failed`)` | any retire hook rejected, plus any drained `DestinationLateOpenDisposalError` | `:374-386` |
| `Error("optimized metadata generation accounting is inconsistent")` | *(not in this file — see direct-destination-owner)* | — |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Disposes a connection which finishes opening after generation retirement. */` | `:12` |
| `/** Opaque non-secret identity, such as a vault/principal fingerprint. */` | `:25` |
| `/** Opaque non-secret identity for the independent repository lane. */` | `:27` |
| `/* A retirement hook which delegates back to its owning generation must not wait for the operation which is waiting for that hook. AsyncLocalStorage keeps the complete nested owner chain across `await` without making unrelated callers observe an early-completing retirement promise. A weak wait-for graph additionally detects cycles formed by parallel hook branches. */` | `:96-102` |
| `// A fire-and-forget hook call must not retain a dependency after its owner has settled, even when the target remains active indefinitely.` | `:145-146` |
| `/** One immutable destination configuration generation. Credential ownership is kept behind the two factories; only opaque, non-secret identities are stored or exposed by this runtime. */` | `:249-253` |
| `// Snapshot nested caller-owned values as soon as their containing option is read. A later accessor must not mutate an already-selected identity or lane operation before it is captured.` | `:269-271` |
| `// Observe every operation admitted before the synchronous state transition immediately. A completed late-disposal failure must remain retirement-owned even when a slower factory hook outlives the opening promise.` | `:338-340` |
| `// Opens admitted before the synchronous state transition may finish after factory retirement. Drain their success/failure and late-resource disposal without turning the open caller's result into a hook failure.` | `:361-363` |
| `// Publish before the microtask above invokes any hook. External callers observe the authoritative operation; hook-owned reentrant calls receive the acknowledgement which breaks the otherwise unavoidable wait cycle.` | `:388-390` |

### Behaviour facts asserted by tests

`test/destination-configuration-generation.test.ts` — 19 `test(...)` cases.

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| `Object.isFrozen(configuration) === true`; `Object.isFrozen(configuration.identity) === true`; `applicationCapability.id !== repositoryCapability.id`; both share `backendKey`; `structuralBackendKey === applicationCapability.backendKey`; `assert.match(configuration.identity.structuralBackendKey, /^sha256:[0-9a-f]{64}$/u)`; two equivalent generations produce the same `structuralBackendKey`; `JSON.stringify(configuration)` contains none of `"qas.example.invalid"`, `password`, `passwd`, `secret`, `credential`, `applicationFactory`, `repositoryFactory`; `Reflect.set(runtime, "configuration", replacement) === false`; descriptor is `{ value, writable: false, enumerable: true, configurable: false }`. | `"keeps immutable credential-free configuration and principal identities"` | `test/destination-configuration-generation.test.ts:57`, assertions `:62-123` |
| `assert.equal(backendKey.length, 71)` for 512-char identity components; equal for an equivalent identity; different when only `endpointId` differs; `applicationCapability.principalKey === applicationPrincipalId` and `repositoryCapability.principalKey === repositoryPrincipalId`. | `"accepts maximum-length identity components behind a bounded opaque backend key"` | `test/destination-configuration-generation.test.ts:126`, assertions `:158-176` |
| `error instanceof RangeError && error.message === "unsupported metadata repository mode Symbol(invalid-mode)"`. | `"reports an invalid Symbol repository mode as a controlled validation error"` | `test/destination-configuration-generation.test.ts:179`, assertion `:205-208` |
| Lanes open only through their own factories; `monitor()` and `monitor.application` are frozen; each lane reports `{ attempts: 1, succeeded: 1, failed: 0, inFlight: 0 }`. | `"opens application and repository lanes only through their own factories"` | `test/destination-configuration-generation.test.ts:211`, assertions `:219-237` |
| Retirement runs once, blocks new opens, and prior monitor snapshots stay immutable. | `"retires both lanes once, blocks new opens, and leaves prior monitor snapshots immutable"` | `test/destination-configuration-generation.test.ts:240` |
| Failed opens are recorded without crossing lane counters. | `"records failed opens without crossing lane counters"` | `test/destination-configuration-generation.test.ts:262` |
| A connection that finishes opening after retirement is disposed. | `"disposes a connection which finishes opening after retirement"` | `test/destination-configuration-generation.test.ts:306` |
| Both retirement hooks run; state stays `retired` when one hook fails. | `"runs both retirement hooks and remains retired when one hook fails"` | `test/destination-configuration-generation.test.ts:357` |
| Direct hook reentry is acknowledged; every hook runs exactly once. | `"acknowledges direct hook reentry and runs every hook exactly once"` | `test/destination-configuration-generation.test.ts:400` |
| Async retirement hooks may `await` a reentrant retirement without deadlock. | `"async retirement hooks can await reentrant retirement without deadlock"` | `test/destination-configuration-generation.test.ts:450` |
| Cross-generation ancestor retirement cycles are broken exactly once. | `"breaks cross-generation ancestor retirement cycles exactly once"` | `test/destination-configuration-generation.test.ts:518` |
| Parallel cross-generation joins are broken without losing authority. | `"breaks parallel cross-generation retirement joins without losing authority"` | `test/destination-configuration-generation.test.ts:604` |
| Every constructor field and lane operation is snapshotted exactly once. | `"snapshots every constructor field and lane operation exactly once"` | `test/destination-configuration-generation.test.ts:725` |
| Caller-owned `Function.bind` on lane operations is not trusted. | `"does not trust caller-owned Function.bind on lane operations"` | `test/destination-configuration-generation.test.ts:887` |
| Nested destination fields are snapshotted before later option accessors can mutate them. | `"snapshots nested destination fields before later option accessors can mutate them"` | `test/destination-configuration-generation.test.ts:946` |
| Retirement drains an admitted open and its late-connection disposal. | `"retirement drains an admitted open and its late-connection disposal"` | `test/destination-configuration-generation.test.ts:1001` |
| A failed late-connection disposal is reported after draining. | `"retirement reports a failed late-connection disposal after draining"` | `test/destination-configuration-generation.test.ts:1063` |
| Late disposal may await owning retirement without deadlocking the drain. | `"late disposal may await owning retirement without deadlocking the drain"` | `test/destination-configuration-generation.test.ts:1110` |
| A late-disposal failure that settles during a slow hook is retained. | `"retirement retains a late-disposal failure which settles during a slow hook"` | `test/destination-configuration-generation.test.ts:1167` |

---

## src/destination/runtime.ts

448 lines.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `DestinationFunctionDescriptor` | interface | `{ readonly kind: "function"; readonly value: RfcFunctionInterface; }` | `src/destination/runtime.ts:21-24` |
| `DestinationStructureDescriptor` | interface | `{ readonly kind: "structure"; readonly value: RfcStructureDefinition; }` | `src/destination/runtime.ts:26-29` |
| `DestinationRecursiveFunctionDescriptor` | interface | `{ readonly kind: "recursive-function"; readonly value: RecursiveMetadataGraph; }` | `src/destination/runtime.ts:31-34` |
| `DestinationMetadataDescriptor` | type alias | `export type DestinationMetadataDescriptor = \| DestinationFunctionDescriptor \| DestinationStructureDescriptor \| DestinationRecursiveFunctionDescriptor;` | `src/destination/runtime.ts:37-40` |
| `RfcDestinationRuntimeOptions<A, R>` | interface | `{ readonly generation: DestinationConfigurationGeneration<A, R>; readonly repository: MetadataRepositoryRuntime<DestinationMetadataDescriptor>; }` | `src/destination/runtime.ts:42-48` |
| `RfcDestinationRuntimeMonitor` | interface | `{ readonly generation: DestinationGenerationMonitor; readonly repository: MetadataRepositoryMonitor; }` | `src/destination/runtime.ts:50-53` |
| `RfcClientDestinationRuntime` | interface | `readonly configuration; openApplication(): Promise<DirectCpicSession>; getFunctionInterface(functionName: string, signal?: AbortSignal): Promise<RfcFunctionInterface>; getStructureDefinition(...); getRecursiveFunctionMetadata(...); retire(): Promise<void>; monitor(): RfcDestinationRuntimeMonitor;` | `src/destination/runtime.ts:56-73` |
| `RfcDestinationRuntime<A, R>` | class | `export class RfcDestinationRuntime< A extends DirectCpicSession = DirectCpicSession, R = unknown, > implements RfcClientDestinationRuntime` | `src/destination/runtime.ts:260-263` |

Public members: `get configuration()` (`:290-292`), `openApplication(): Promise<A>`
(`:294`), `getFunctionInterface` (`:301-304`), `getStructureDefinition`
(`:314-317`), `getRecursiveFunctionMetadata` (`:327-330`), `retire(): Promise<void>`
(`:345`), `monitor(): RfcDestinationRuntimeMonitor` (`:402`).

### Configuration options and defaults

| Name | Default (verbatim) | Effect per source | Citation |
|---|---|---|---|
| `generation` | *required* | `if (!(generation instanceof DestinationConfigurationGeneration))` → `TypeError` — an `instanceof` check | `:276-280` |
| `repository` | *required* | Duck-typed via `bindRepository`; (comment) `// The runtime uses behavior rather than an instanceof test for the generic repository so an independently bundled copy remains injectable.` | `:282-283`, `:141-176` |
| `#get` `mode` | `mode: MetadataRepositoryMode = this.configuration.repositoryMode` | Per-lookup repository mode | `:409-413` |
| recursive lookups | `MetadataRepositoryMode.OptimizedOnly` (hard-coded, not defaulted) | `getRecursiveFunctionMetadata` always passes `OptimizedOnly` | `:331-336` |

### Errors

| Message/code (verbatim) | Trigger | Citation |
|---|---|---|
| `TypeError("destination runtime options must be an object")` | non-object options | `:272-274` |
| `TypeError("destination runtime generation must be a DestinationConfigurationGeneration")` | failed `instanceof` | `:276-280` |
| `TypeError("destination runtime repository must provide get(), invalidate(), retire(), and monitor()")` | repository not an object, or any of the four members missing/non-function | `:144-162` |
| `TypeError(`${path} must be a function`)` | `callable()` guard over `openApplication`/`retire`/`monitor` on the generation | `:102-107`, `:116-127` |
| `Error(`metadata repository returned ${descriptor.kind} for function ${functionName}`)` | descriptor kind mismatch on `getFunctionInterface` | `:306-310` |
| `Error(`metadata repository returned ${descriptor.kind} for structure ${structureName}`)` | on `getStructureDefinition` | `:319-323` |
| `Error(`metadata repository returned ${descriptor.kind} for recursive function ${functionName}`)` | on `getRecursiveFunctionMetadata` | `:337-341` |
| `Error(`metadata repository returned a mismatched descriptor for ${objectKind} ${objectName}`)` | `descriptorMatches` false — **and the structural key is invalidated first** | `:433-439` |
| `Error(`destination runtime ${this.configuration.generationId} is retired`)` | `openApplication` or `#get` while `#state !== "active"` | `:443-447`, `:295-297`, `:415` |
| `AggregateError(failures, `destination runtime ${this.configuration.generationId} retirement failed`)` | repository and/or generation retirement rejected | `:376-379` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** The repository value keeps unlike descriptor shapes explicitly tagged. */` | `src/destination/runtime.ts:36` |
| `/** Generic-erased façade contract implemented by every destination owner. */` | `:55` |
| `/** Destination-owned production seam shared by compatibility façades. The injected repository adapter is responsible for obtaining backend access through this generation's repository lane. Application sessions are opened only through the application lane, so a context-pinned/LUW connection is never lent to metadata work by this owner. */` | `:252-259` |
| `// The runtime uses behavior rather than an instanceof test for the generic repository so an independently bundled copy remains injectable.` | `:282-283` |
| `// The retirement body starts in a later microtask, so owning generation hooks can never run before external callers can observe this operation.` | `:386-387` |
| `// A malformed adapter result must not become a permanent cache poison.` | `:434` |

### Behaviour facts asserted by tests

There is no `test/destination-runtime.test.ts` in scope. This file is exercised
indirectly through `test/direct-destination-owner.test.ts`, which constructs
`RfcDestinationRuntime` inside `DirectDestinationOwner`
(`src/destination/direct-destination-owner.ts:1908-1911`).

---

## src/destination/direct-destination-owner.ts

2940 lines. This is the composition root.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `DirectDestinationLane` | type alias | `export type DirectDestinationLane = "application" \| "repository";` | `:83` |
| `DirectDestinationSessionOpenContext` | interface | `{ readonly lane: DirectDestinationLane; readonly signal: AbortSignal; }` | `:85-89` |
| `DirectDestinationSelectedSession` | interface | `{ readonly session: DirectCpicSession; readonly selectedConnection: NormalizedDirectConnection; }` | `:91-94` |
| `DirectDestinationSessionOpenResult` | type alias | `export type DirectDestinationSessionOpenResult = \| DirectCpicSession \| DirectDestinationSelectedSession;` | `:96-98` |
| `DirectDestinationSessionFactory` | interface | `{ open(connection, context): DirectDestinationSessionOpenResult \| PromiseLike<DirectDestinationSessionOpenResult>; }` | `:100-107` |
| `DirectDestinationSessionOptions` | interface | `readonly programName?, localAddress?: string; connectTimeoutMs?, operationTimeoutMs?: number; transportFactory?; recursiveSerializerDecisionProvider?` | `:109-116` |
| `DirectDestinationPoolOptions` | interface | `maxConnections?, maxWaiters?, acquireTimeoutMs?, lifecycleTimeoutMs?, shutdownTimeoutMs?, lowWater?, idleHigh?: number; validateOnCheckout?: boolean; scheduler?, lifecycleScheduler?: ConnectionPoolScheduler; diagnostics?: RfcDiagnosticEmitter` | `:118-130` |
| `DirectDestinationMetadataOptions` | interface | `maxEntries?, maxRetainedBytes?, maxProbeEntries?, maxAuthorizationEntries?, maxObjectEpochEntries?, maxInFlightLoads?, maxSnapshotNodes?, maxSnapshotDepth?, maxSnapshotProperties?: number; diagnostics?: RfcDiagnosticEmitter` | `:132-143` |
| `DirectDestinationOwnerOptions` | interface | `readonly connection: NormalizedDirectConnection; readonly generationId: string; readonly identity: DestinationIdentityInput; readonly repositoryMode?: MetadataRepositoryMode; readonly sessionFactory?; readonly session?; readonly applicationPool?; readonly repositoryPool?; readonly metadata?` | `:145-156` |
| `DirectDestinationInvocation` | interface | `export interface DirectDestinationInvocation extends ClassicRfcInvocationOptions { readonly functionName: string; readonly parameters: Readonly<Record<string, unknown>>; }` | `:158-162` |
| `DirectDestinationReleaseOptions` | interface | `{ readonly reusable?: boolean; readonly reset?: boolean; readonly idleHigh?: number; }` | `:164-170` |
| `DirectDestinationApplicationLease` | interface | `export interface DirectDestinationApplicationLease { readonly [applicationLeaseBrand]: true; }` — brand is `Symbol("open-rfc direct destination application lease")` | `:172-179` |
| `DirectDestinationOwnerMonitor` | interface | `state; destination; metadata; applicationPool; repositoryPool; contexts; applicationLeases; contextPinnedApplicationLeases; ordinaryApplicationLeases; activeApplicationOperations; quarantinedApplicationTails; optimizedGenerationTokens; maxOptimizedGenerationTokens; metadataRefreshInFlight: 0 \| 1` | `:181-199` |
| `DirectDestinationMetadataRefreshResult` | interface | `{ readonly checkedFunctionNames: readonly string[]; readonly checkedStructureNames: readonly string[]; readonly invalidatedFunctionNames: readonly string[]; readonly invalidatedStructureNames: readonly string[]; }` | `:201-206` |
| `DirectDestinationMetadataPreflightError` | class | `export class DirectDestinationMetadataPreflightError extends Error { readonly functionName: string; constructor(functionName: string, cause: unknown) }`, message `` `metadata preflight failed for RFC function ${functionName}` ``, `this.name = "DirectDestinationMetadataPreflightError"` | `:208-219` |
| `createProductionDirectDestinationSessionFactory` | function | `export function createProductionDirectDestinationSessionFactory( options?: DirectDestinationSessionOptions, ): DirectDestinationSessionFactory` | `:714-718` |
| `DirectDestinationOwner` | class | `export class DirectDestinationOwner` | `:1764` |
| `classifyDirectDestinationTransactionFailure` | function | `export function classifyDirectDestinationTransactionFailure( failure: unknown, ): TransactionFailureKind` | `:2881-2883` |
| `createDirectDestinationTransactionAdapter` | function | `export function createDirectDestinationTransactionAdapter( owner: DirectDestinationOwner, ): TransactionLeaseAdapter<DirectDestinationApplicationLease>` | `:2898-2900` |

Public members of `DirectDestinationOwner`:

| Member | Signature (verbatim) | Citation |
|---|---|---|
| `configuration` | `get configuration(): DestinationConfiguration` | `:1938-1940` |
| `acquireApplication` | `async acquireApplication( options: ConnectionPoolAcquireOptions = {}, ): Promise<DirectDestinationApplicationLease>` | `:1942-1948` |
| `acquireApplications` | `async acquireApplications( count: number, options: ConnectionPoolAcquireOptions = {}, ): Promise<readonly DirectDestinationApplicationLease[]>` — (comment) `/** Atomic multi-acquire used by node-rfc Pool compatibility. */` | `:1950-1957` |
| `beginContext` | `beginContext(): Promise<SessionContextToken>; beginContext(token: SessionContextToken): Promise<SessionContextToken>;` | `:1959-1965` |
| `invokeContext` | `invokeContext( token: SessionContextToken, invocation: DirectDestinationInvocation, signal?: AbortSignal, ): Promise<ClassicRfcOutput>` | `:1967-1971` |
| `pingContext` | `pingContext( token: SessionContextToken, signal?: AbortSignal, ): Promise<DirectCpicPingResult>` | `:1982-1985` |
| `endContext` | `endContext(token: SessionContextToken): Promise<void>` | `:1994` |
| `invoke` | `invoke( lease: DirectDestinationApplicationLease, invocation: DirectDestinationInvocation, signal?: AbortSignal, ): Promise<ClassicRfcOutput>` | `:2019-2023` |
| `pingApplication` | `pingApplication( lease: DirectDestinationApplicationLease, signal?: AbortSignal, ): Promise<DirectCpicPingResult>` | `:2168-2171` |
| `applicationInfo` | `applicationInfo( lease: DirectDestinationApplicationLease, ): Promise<DirectCpicSessionInfo>` | `:2180-2182` |
| `resetApplication` | `resetApplication( lease: DirectDestinationApplicationLease, signal?: AbortSignal, ): Promise<void>` | `:2191-2194` |
| `releaseApplication` | `async releaseApplication( lease: DirectDestinationApplicationLease, options: DirectDestinationReleaseOptions = {}, ): Promise<void>` | `:2203-2206` |
| `getFunctionInterface` | `async getFunctionInterface( name: string, signal?: AbortSignal, ): Promise<RfcFunctionInterface>` | `:2253-2256` |
| `getRecursiveFunctionMetadata` | `async getRecursiveFunctionMetadata( name: string, signal?: AbortSignal, ): Promise<RecursiveMetadataGraph>` | `:2261-2264` |
| `getStructureDefinition` | `async getStructureDefinition( name: string, signal?: AbortSignal, ): Promise<RfcStructureDefinition>` | `:2269-2272` |
| `refreshOptimizedMetadata` | `refreshOptimizedMetadata( functionNames: readonly string[], structureNames: readonly string[], signal?: AbortSignal, ): Promise<DirectDestinationMetadataRefreshResult>` | `:2282-2286` |
| `retire` | `retire(): Promise<void>` | `:2352` |
| `monitor` | `monitor(): DirectDestinationOwnerMonitor` | `:2391` |

### Configuration options and defaults

| Name | Default (verbatim) | Effect per source | Citation |
|---|---|---|---|
| `DEFAULT_APPLICATION_POOL` | `Object.freeze({ maxConnections: 32, maxWaiters: 128, acquireTimeoutMs: 30_000, lifecycleTimeoutMs: 45_000, shutdownTimeoutMs: 60_000, lowWater: 0, validateOnCheckout: true, })` | (comment) `// Covers the archived node-rfc retained-4 plus acquire-5 pool sequence.` | `:368-377` |
| `DEFAULT_REPOSITORY_POOL` | `Object.freeze({ maxConnections: 2, maxWaiters: 64, acquireTimeoutMs: 30_000, lifecycleTimeoutMs: 45_000, shutdownTimeoutMs: 60_000, lowWater: 0, validateOnCheckout: true, })` | — | `:378-386` |
| `DEFAULT_METADATA` | `Object.freeze({ maxEntries: 512, maxRetainedBytes: 64 * 1_024 * 1_024, maxProbeEntries: 64, maxAuthorizationEntries: 1_024, maxObjectEpochEntries: 1_024, maxInFlightLoads: 64, maxSnapshotNodes: 100_000, maxSnapshotDepth: 256, maxSnapshotProperties: 1_000_000, })` | — | `:387-397` |
| pool `idleHigh` | `input?.idleHigh ?? maxConnections` | Per-pool | `:598` |
| `session.programName` | `input?.programName ?? "open-rfc"` | Must match `/^[\x20-\x7e]{1,64}$/u` | `:536`, `:543-545` |
| `session.connectTimeoutMs` | `input?.connectTimeoutMs ?? 10_000` | `finiteTimeout` 1..2_147_483_647 | `:538`, `:566` |
| `session.operationTimeoutMs` | `input?.operationTimeoutMs ?? 30_000` | `finiteTimeout` | `:539`, `:567-570` |
| `repositoryMode` | `options.repositoryMode ?? MetadataRepositoryMode.Classic` | — | `:1809` |
| `resetOnRelease` (application pool) | `resetOnRelease: false` — hard-coded, overrides caller options | (comment) `// TransactionRuntime performs its explicit same-lease reset exactly once.` | `:1872-1873` |
| `resetOnRelease` (repository pool) | `resetOnRelease: false` — hard-coded | (comment) `// This lane runs only connector-owned metadata RFMs and cannot carry caller application context. Avoid a two-call SYSTEM_RESET/refresh on every descriptor lookup; failed generations are still non-reusable.` | `:1890-1893` |
| `#maxOptimizedGenerationTokens` | `metadataOptions.maxEntries` (default `512`) | Bound on `#optimizedGenerationTokens` | `:1833`, `:388` |
| context `operationTimeoutMs` | `Math.max( sessionOptions.operationTimeoutMs, applicationPoolOptions.acquireTimeoutMs, applicationPoolOptions.lifecycleTimeoutMs, )` | `SessionContextRuntime` bound | `:1928-1932` |
| `MAX_METADATA_REFRESH_NAMES_PER_KIND` | `512` | Per-kind cap on `refreshOptimizedMetadata` input | `:1244`, `:1253-1257` |
| `MAX_TIMER_MS` | `2_147_483_647` | Timeout ceiling | `:367` |
| `release` `reusable` | `optionalBoolean(options.reusable, true, "application release reusable")` | — | `:1743-1747` |
| `release` `reset` | `optionalBoolean(options.reset, false, "application release reset")` | (comment) `/** Reset the same physical session after the operation tail, before reuse. */` | `:1748-1752`, `:166` |
| `release` `idleHigh` | `undefined`; must be a non-negative safe integer and `<= applicationPool.monitor().maxConnections` | — | `:1735-1741`, `:2208-2216` |
| pool `validate` hook | `validate: async (session, context) => { await session.ping(context.signal); return true; }` (both lanes) | — | `:1865-1868`, `:1883-1886` |
| pool `destroy` hook | `destroy: (session) => session.close()` (both lanes) | — | `:1864`, `:1882` |

### Errors

| Message/code (verbatim) | Trigger | Citation |
|---|---|---|
| `TypeError("direct destination owner options must be an object")` | non-object / array options | `:1799-1805` |
| `TypeError("connection must be normalized direct connection data")` | non-object connection | `:448-450` |
| `RangeError("connection.port must be an integer in 1..65535")` | port out of range | `:467-469` |
| `RangeError("connection.sysnr must contain two decimal digits")` | `!/^\d{2}$/u` | `:470-472` |
| `RangeError("connection.applicationServerService must match connection.sysnr")` | service `!== \`sapdp${sysnr}\`` | `:473-477` |
| `RangeError("connection.client must contain three decimal digits")` | `!/^\d{3}$/u` | `:478-480` |
| `RangeError("connection.language must be one SAP language code")` | `!/^[A-Z0-9]$/u` | `:481-483` |
| `RangeError("connection.cpicStreaming must be disabled or enabled")` | other value | `:484-486` |
| `TypeError("identity must be an object")` | non-object identity | `:502-504` |
| `Error("identity.client must match the normalized connection client")` | mismatch | `:1823-1825` |
| `Error("identity.language must match the normalized connection language")` | mismatch | `:1826-1830` |
| `RangeError("session programName must contain 1..64 ASCII bytes")` | regex failure | `:543-545` |
| `TypeError("session localAddress must be a non-empty string")` | — | `:546-551` |
| `TypeError("session transportFactory must be a function")` | — | `:552-554` |
| `TypeError("session recursiveSerializerDecisionProvider must be a function")` | — | `:555-562` |
| `RangeError(`${path} must contain 1..30 ASCII bytes`)` | `classicMetadataObjectName` on function/structure names | `:411-421` |
| `RangeError("invocation.functionName must contain 1..30 ASCII bytes")` | `snapshotInvocation` | `:1209-1213` |
| `TypeError("invocation must be an object")` / `TypeError("invocation parameters must be an object")` | — | `:1198-1203`, `:1169-1171` |
| `TypeError("sessionFactory must be an object")` / `TypeError("sessionFactory.open must return a session")` / `TypeError("selected session results must use own data properties")` / `TypeError("sessionFactory.open must return a DirectCpicSession")` | session-factory boundary | `:731-736`, `:742-744`, `:753-760`, `:881-883` |
| `Error("sessionFactory.open returned a session already owned by a pool")` | `#admittedSessions.has(session)` | `:2437-2439` |
| `Error(`destination ${lane} session opened outside its pool scope`)` | `#openRawSession` called outside the `AsyncLocalStorage` lane scope, or in the wrong lane | `:2427-2430` |
| `Error("destination generation returned an unbound session")` | bound session missing from the WeakMap | `:2480-2482` |
| `Error(`optimized ${objectKind} metadata returned an invalid generation token`)` | token fails `/^function:\d{8}:\d{6}$/u` (function/recursive-function) or `/^structure:\d{14}$/u` (structure) | `:811-823` |
| `Error(`optimized recursive-function metadata returned a mismatched identity for ${name}`)` | `functionIdentity.name !== name` or `functionIdentity.generationToken !== descriptor.generationToken` | `:1009-1017` |
| `MetadataAccessFailure("unavailable", "session does not implement optimized RFC metadata")` | missing optimized session method | `:932-935`, `:947-950`, `:976-979`, `:1037-1040` |
| `MetadataAccessFailure("unavailable", "session does not implement recursive optimized RFC metadata")` | missing recursive optimized method | `:994-998` |
| `MetadataAccessFailure("unavailable", "session does not implement optimized RFC metadata timestamps")` | missing timestamp method | `:1056-1059` |
| `MetadataAccessFailure("unavailable", "recursive RFC metadata requires the optimized repository path")` | recursive load under a non-optimized strategy | `:2536-2541` |
| `MetadataAccessFailure("unavailable", "RFC_METADATA_GET is unavailable on this backend", { cause: error })` | `RfcCoreError` with `category === RfcFailureCategory.AbapException` and key in `{"FU_NOT_FOUND","FUNCTION_NOT_EXIST","RFC_NOT_FOUND"}` | `:1657-1661`, `:1671-1680` |
| `MetadataAccessFailure("authorization", "RFC_METADATA_GET is not authorized for this repository principal", { cause: error })` | failure key or `abap.runtimeId` in `{"CALL_FUNCTION_NO_AUTHORITY","RFC_NO_AUTHORITY","RFC_AUTHORIZATION_FAILURE"}` | `:1662-1666`, `:1681-1690` |
| `MetadataAccessFailure("canceled", "optimized metadata timestamp refresh was canceled")` | caller aborted, or owner retirement aborts the refresh controller | `:1292-1297`, `:2386` |
| `Error("another optimized metadata timestamp refresh is in progress")` | a refresh with a different key is in flight | `:2314-2319` |
| `RangeError(`metadata timestamp refresh accepts at most 512 ${objectKind} names`)` | over `MAX_METADATA_REFRESH_NAMES_PER_KIND` | `:1253-1257` |
| `Error(`duplicate ${objectKind} name ${name}`)` | duplicate in the refresh request | `:1269-1271` |
| `Error("optimized metadata timestamp batch contains a foreign function")` / `"...a foreign function error"` / `"...a foreign structure"` / `"...a foreign structure error"` | response name not in the request set | `:1473-1475`, `:1503-1505`, `:1523-1525`, `:1553-1555` |
| `Error(`optimized metadata timestamp batch contains duplicate function ${rawName}`)` / `...structure ${rawName}` | duplicate response entry | `:1476-1480`, `:1526-1530` |
| `Error(`optimized metadata timestamp batch contains an invalid function error for ${rawName}`)` / `...structure error for...` | error string fails `/^[A-Z0-9_]{1,30}$/u`, or collides with a token | `:1506-1515`, `:1556-1565` |
| `Error(`optimized metadata timestamp batch has no outcome for function ${name}`)` / `...structure ${name}` | requested name absent from both tokens and errors | `:1568-1581` |
| `TypeError(`optimized metadata timestamp batch ${name} must be a map`)` and 6 sibling iterator-shape messages | hostile/malformed iterable | `:1358-1453` |
| `RangeError(`optimized metadata timestamp batch ${name} has too many entries`)` | entries exceed `requested.length + 1` | `:1429-1433` |
| `Error(`unsupported metadata object kind ${objectKind}`)` | kind outside `function`/`recursive-function`/`structure` | `:2493-2499`, `:2618-2624` |
| `Error(`unsupported function metadata object kind ${objectKind}`)` | during refresh invalidation | `:2707-2712` |
| `Error("optimized metadata generation accounting is inconsistent")` | eviction loop found no oldest key | `:2635-2637` |
| `RangeError("recursive metadata retained-byte estimate is unsafe")` | `!Number.isSafeInteger(retained)` | `:1600-1602` |
| `Error(`function and recursive metadata generations disagree for ${functionName}`)` | flat and recursive descriptors carry different tokens | `:2094-2103` |
| `Error(`${parameter.parameterName} lacks its structure type name`)` | container parameter with empty `tableName` | `:2127-2131` |
| `DirectDestinationMetadataPreflightError` | any metadata failure in `invoke` before application entry, **and** a caller abort observed after preflight | `:2140-2153` |
| `Error("application lease has already been released")` | `releaseApplication`/`#ownedLeaseRecord`/`#runApplicationOperation` on a non-`owned` record | `:2218-2220`, `:2792-2793`, `:2867-2869` |
| `Error("application lease already has an active operation")` | `record.active` | `:2795-2799` |
| `TypeError("application lease must be an opaque owner token")` | non-object lease | `:2850-2855` |
| `Error("application lease does not belong to this destination")` | lease absent from `#knownLeases` | `:2856-2859` |
| `RangeError("application release idleHigh exceeds application pool capacity")` | `idleHigh > applicationPool.monitor().maxConnections` | `:2208-2216` |
| `RangeError("application release idleHigh must be a non-negative integer")` | not a non-negative safe integer | `:1736-1741` |
| `Error(`direct destination ${this.configuration.generationId} is retired`)` | any operation while `#state !== "active"` | `:2873-2877` |
| `AggregateError(failures, `direct destination ${this.configuration.generationId} retirement failed`)` | any of contexts / application pool / destination / repository pool retirement rejected | `:2374-2379` |
| `AggregateError([primary, ...failures], "destination retired while atomic application acquire was completing", { cause: primary })` | retirement raced an atomic acquire and cleanup also failed | `:2833-2839` |
| `AggregateError([primary, cleanup], message, { cause: primary })` via `aggregatePrimaryAndCleanup` with messages `"application reset and destruction both failed"`, `"metadata access and repository-session cleanup both failed"`, `"session binding and cleanup both failed"`, `"direct CPIC logon and session cleanup both failed"` | primary + cleanup both failed | `:1695-1701`, `:2235-2240`, `:2776-2780`, `:2448-2452`, `:698-702` |

`classifyDirectDestinationTransactionFailure` return values (verbatim):
`"recoverable"` for `DirectDestinationMetadataPreflightError` (`:2884-2886`),
`"recoverable"` for `DirectCpicPreWireError` (`:2887`), `"recoverable"` for
`RfcCoreError` with `failure.disposition === RfcConnectionDisposition.Reusable`
(`:2888-2893`), otherwise `"ambiguous"` (`:2894`).

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Pool-owned lifecycle signal; an implementation must stop I/O on abort. */` | `:87` |
| `/** Injectable authenticated-session boundary used by deterministic tests. */` | `:100` |
| `/** Already normalized direct-application-server connection data. */` | `:146` |
| `/** Reset the same physical session after the operation tail, before reuse. */` | `:166` |
| `/** Optional idle-retention cap for this application recycle handoff. */` | `:168` |
| `/** Nominal, resource-free token. Its DirectCpicSession remains owner-private. */` | `:176` |
| `/** Release handoffs waiting for a previously admitted operation to settle. */` | `:192` |
| `/** Same-response generation tokens retained for optimized descriptors. */` | `:194` |
| `/** Explicit timestamp refreshes are bounded to one physical batch. */` | `:197` |
| `/** A failure proven to have happened before the application lease was entered. */` | `:208` |
| `/** Absent only for a legacy injected test/session boundary. */` | `:331` |
| `// Covers the archived node-rfc retained-4 plus acquire-5 pool sequence.` | `:369` |
| `// Preserve compatibility with deterministic injected sessions written before the detailed same-response API existed. Such descriptors are deliberately not eligible for timestamp tracking.` | `:972-974` |
| `// A hostile cleanup hook cannot replace an already-selected result.` | `:1313` |
| `// A malformed reason getter cannot prevent cancellation from settling.` | `:1137` |
| `// An incomplete DDIC closure is local to this recursive lookup. It may be bypassed only when every active container has an independently validated flat descriptor; it must not demote the optimized repository globally.` | `:1647-1649` |
| `/** Production composition root for one immutable direct-CPIC destination. Raw sessions exist only inside captured lane resources. Repository metadata completes before application-pool entry, and every application operation is serialized behind a lease-local tail which release must drain. */` | `:1757-1763` |
| `// TransactionRuntime performs its explicit same-lease reset exactly once.` | `:1872` |
| `// This lane runs only connector-owned metadata RFMs and cannot carry caller application context. Avoid a two-call SYSTEM_RESET/refresh on every descriptor lookup; failed generations are still non-reusable.` | `:1890-1892` |
| `// One owner has one immutable repository-principal identity. An optimized load still executes under that principal before its descriptor enters this generation's cache.` | `:1901-1903` |
| `/** Atomic multi-acquire used by node-rfc Pool compatibility. */` | `:1950` |
| `// This synchronous transition is the once-only ownership handoff.` | `:2221` |
| `/** Compare one explicit, bounded descriptor batch with SAP's current generations. The caller chooses when to pay for this check; there is no timer, guessed TTL, or background I/O. */` | `:2277-2281` |
| `// Caller-controlled array or signal traps cannot retire the owner and then smuggle a refresh past the admission gate.` | `:2291-2292` |
| `// Publish the terminal before context eviction can enter a caller-owned session close and re-enter retire(). The context gate then closes in the same turn, before either pool starts rejecting lifecycle handoffs.` | `:2381-2383` |
| `// A classic or legacy fallback must never inherit a token from an older optimized descriptor for the same structural key.` | `:2505-2506` |
| `// Validate while the lease is still active. A malformed injected/session boundary is therefore disposed with the same conservative policy as a decoder failure, before any cache state is changed.` | `:2656-2658` |
| `// A descriptor reloaded while the timestamp call was in flight owns a newer record and must not be invalidated by this stale comparison.` | `:2714-2715` |
| `// Begin in a microtask so reentrant release always observes the published tail.` | `:2803` |
| `/** Conservative business-call classifier suitable for TransactionRuntime. */` | `:2880` |
| `/** Capture an owner's opaque-lease operations for direct TransactionRuntime use. */` | `:2897` |
| `// Ownership is already transferred; release intentionally ignores abort.` | `:2932` |
| `/** Internal composition helper for route adapters which must resolve a direct endpoint immediately before each physical pool connection is created. */` | `:710-713` |

### Behaviour facts asserted by tests

`test/direct-destination-owner.test.ts` — 47 `test(...)` cases. Selected.

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| Every option/connection/identity/session/pool/metadata validation message listed in the Errors table fires before any session I/O; with all optional groups deleted, `owner.configuration.generationId === "validation-owner-1"`, `owner.monitor().state === "active"`, `retire()` is idempotent and leaves `"retired"`. | `"validates the complete destination composition before any session I/O"` | `test/direct-destination-owner.test.ts:748`, assertions `:749-909` |
| Two owners isolate principals, caches, cancellation, and retirement. | `"production compatibility owners isolate principals, caches, cancellation, and retirement"` | `test/direct-destination-owner.test.ts:912` |
| `Object.isFrozen(monitor) === true`; `monitor.applicationPool.maxConnections === 2`; `monitor.repositoryPool.maxConnections === 1`; `monitor.metadata.entries === 2`; **`assert.equal("password" in monitor, false)`**. | `"rejects malformed destination operations before lease or repository I/O"` | `test/direct-destination-owner.test.ts:1022`, assertions `:1254-1259` |
| Classic metadata is preflighted on a bounded repository lane before same-lease application use; `Object.isFrozen(info) === true`. | `"preflights classic metadata on a bounded repository lane before same-lease application use"` | `test/direct-destination-owner.test.ts:1206`, `:1213` |
| Auto mode probes then loads optimized metadata on the repository lane. | `"auto mode probes and loads optimized metadata on the repository lane"` | `test/direct-destination-owner.test.ts:1317` |
| Mixed optimized function generations are rejected before application entry, matching `/function and recursive metadata generations disagree/u`. | `"rejects mixed optimized function generations before application entry"` | `test/direct-destination-owner.test.ts:1718`, `:1756` |
| Timestamp refresh invalidates a stale tagged recursive descriptor. | `"timestamp refresh invalidates a stale tagged recursive descriptor"` | `test/direct-destination-owner.test.ts:1772` |
| A recursive descriptor whose same-response generation was detached is rejected. | `"rejects a recursive descriptor whose same-response generation was detached"` | `test/direct-destination-owner.test.ts:1851` |
| `Object.isFrozen(functionNames)`/`Object.isFrozen(structureNames)` are true in the session callback; `Object.isFrozen(unchangedResult.checkedFunctionNames) === true`. | `"explicit timestamp refresh invalidates only changed or typed-missing optimized descriptors"` | `test/direct-destination-owner.test.ts:1871`, `:1886-1887`, `:1925` |
| Refreshes are deduplicated; malformed or failed batches cause no partial invalidation and no fallback. | `"deduplicates refreshes and rejects malformed or failed batches without partial invalidation or fallback"` | `test/direct-destination-owner.test.ts:1965` |
| Caller cancellation is isolated while a deduplicated refresh continues. | `"isolates caller cancellation while one deduplicated timestamp refresh continues"` | `test/direct-destination-owner.test.ts:2044` |
| An in-flight refresh cannot invalidate a descriptor reloaded on another repository session. | `"does not let an in-flight refresh invalidate a descriptor reloaded on another repository session"` | `test/direct-destination-owner.test.ts:2091` |
| Hostile iterables reject as `TypeError` whose message excludes the secret (`assert(error instanceof TypeError && !error.message.includes(secret))`), the pair accessor is never invoked (`assert.equal(pairGetterCalled, false)`), invalidation count is unchanged, and both repository sessions are closed. | `"rejects hostile timestamp iterables without reading pair accessors or leaking their errors"` | `test/direct-destination-owner.test.ts:2132`, assertions `:2178-2193` |
| Optimized token state is bounded and dropped for classic fallback loads. | `"bounds optimized token state and drops it for classic fallback loads"` | `test/direct-destination-owner.test.ts:2225` |
| Retirement aborts a timestamp refresh, clears tokens, and drains its repository lease. | `"retirement aborts a timestamp refresh, clears tokens, and drains its repository lease"` | `test/direct-destination-owner.test.ts:2254` |
| Auto mode falls back only after a *classified* optimized capability miss / object-authorization failure, and never hides an unclassified probe failure. | `"auto mode falls back only after a classified optimized capability miss"`; `"auto mode falls back after a classified optimized object authorization failure"`; `"auto mode never hides an unclassified optimized probe failure"` | `test/direct-destination-owner.test.ts:2282`, `:2304`, `:2329` |
| Invalid metadata names reject before repository acquisition. | `"rejects invalid metadata names before repository acquisition"` | `test/direct-destination-owner.test.ts:2346` |
| Concurrent descriptor loads are deduplicated without lending application leases to metadata; `Object.isFrozen(acquired) === true`. | `"deduplicates concurrent descriptor loads without lending application leases to metadata"` | `test/direct-destination-owner.test.ts:2368`, `:2381` |
| Nested invocation values are captured before async metadata preflight; the observed parameters and their nested `INPUT`/`ROWS` are frozen. | `"captures nested invocation values before asynchronous metadata preflight"` | `test/direct-destination-owner.test.ts:2411`, `:2449-2451` |
| Release is claimed once and quarantined behind a hung application tail. | `"claims release once and quarantines it behind a hung application tail"` | `test/direct-destination-owner.test.ts:2456` |
| Reset-on-release waits for the admitted tail and reuses the same session. | `"reset-on-release waits for the admitted tail and reuses the same session"` | `test/direct-destination-owner.test.ts:2488` |
| Repository preflight failures are marked recoverable without touching the application session. | `"marks repository preflight failures recoverable without touching the application session"` | `test/direct-destination-owner.test.ts:2529` |
| A newly opened raw session is closed when method binding rejects. | `"closes a newly opened raw session when method binding rejects"` | `test/direct-destination-owner.test.ts:2595` |
| Context begin/end reference-counts one pinned application lease. | `"context begin/end reference-counts one pinned application lease"` | `test/direct-destination-owner.test.ts:2611` |
| Owner retirement closes the context gate before draining its application pool, and publishes one retirement before a context close can reenter it. | `"owner retirement closes the context gate before draining its application pool"`; `"owner publishes one retirement before a context close can reenter it"` | `test/direct-destination-owner.test.ts:2781`, `:2823` |
| The injected factory, session methods, connection and invocation inputs are captured once; `Object.isFrozen(connection) === true` and `connection.password === ["fixture","password"].join("-")` inside the factory. | `"captures the injected factory, session methods, connection, and invocation inputs once"` | `test/direct-destination-owner.test.ts:2878`, `:530-531` |
| Retirement drains both finite pools and the destination generation. | `"retirement drains both finite pools and the destination generation"` | `test/direct-destination-owner.test.ts:2907` |

---

## src/diagnostics/structured-diagnostics.ts

723 lines. Imports `node:fs`/`node:fs/promises`/`node:path` only for the file
sink (`:1-9`).

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RFC_DIAGNOSTIC_CATEGORIES` | frozen const tuple | `export const RFC_DIAGNOSTIC_CATEGORIES = Object.freeze([...] as const);` | `:11-19` |
| `RfcDiagnosticCategory` | type alias | `export type RfcDiagnosticCategory = (typeof RFC_DIAGNOSTIC_CATEGORIES)[number];` | `:20` |
| `RFC_DIAGNOSTIC_LEVELS` | frozen const tuple | `export const RFC_DIAGNOSTIC_LEVELS = Object.freeze([...] as const);` | `:22-28` |
| `RfcDiagnosticLevel` | type alias | `export type RfcDiagnosticLevel = (typeof RFC_DIAGNOSTIC_LEVELS)[number];` | `:29` |
| `RFC_DIAGNOSTIC_CODES` | frozen const tuple | `export const RFC_DIAGNOSTIC_CODES = Object.freeze([...] as const);` | `:31-67` |
| `RfcDiagnosticCode` | type alias | `export type RfcDiagnosticCode = (typeof RFC_DIAGNOSTIC_CODES)[number];` | `:68` |
| `RFC_DIAGNOSTIC_STATES` | frozen const tuple | `export const RFC_DIAGNOSTIC_STATES = Object.freeze([...] as const);` | `:70-80` |
| `RfcDiagnosticState` | type alias | `export type RfcDiagnosticState = (typeof RFC_DIAGNOSTIC_STATES)[number];` | `:81` |
| `RFC_DIAGNOSTIC_PHASES` | frozen const tuple | `export const RFC_DIAGNOSTIC_PHASES = Object.freeze([...] as const);` | `:83-96` |
| `RfcDiagnosticPhase` | type alias | `export type RfcDiagnosticPhase = (typeof RFC_DIAGNOSTIC_PHASES)[number];` | `:97` |
| `RFC_DIAGNOSTIC_DISPOSITIONS` | frozen const tuple | `export const RFC_DIAGNOSTIC_DISPOSITIONS = Object.freeze([...] as const);` | `:99-104` |
| `RfcDiagnosticDisposition` | type alias | `export type RfcDiagnosticDisposition = (typeof RFC_DIAGNOSTIC_DISPOSITIONS)[number];` | `:105-106` |
| `RfcDiagnosticInput` | interface | `{ readonly category; readonly level; readonly code; readonly correlationId?: string; readonly state?; readonly phase?; readonly disposition?; readonly durationMs?: number; readonly count?: number; }` | `:108-118` |
| `RfcDiagnosticEvent` | interface | `export interface RfcDiagnosticEvent extends RfcDiagnosticInput { readonly schemaVersion: 1; readonly sequence: number; readonly timestamp: string; }` | `:120-124` |
| `RfcDiagnosticSink` | interface | `export interface RfcDiagnosticSink { readonly write: (event: RfcDiagnosticEvent) => void \| Promise<void>; readonly close?: () => void \| Promise<void>; }` | `:126-129` |
| `RfcDiagnosticEmitter` | interface | `export interface RfcDiagnosticEmitter { readonly emit: (input: RfcDiagnosticInput) => boolean; }` | `:137-139` |
| `RFC_RUNTIME_DIAGNOSTIC_BUFFER_LIMIT` | const | `export const RFC_RUNTIME_DIAGNOSTIC_BUFFER_LIMIT = 256;` | `:141` |
| `RfcDiagnosticReporter` | type alias | `export type RfcDiagnosticReporter = (input: RfcDiagnosticInput) => boolean;` | `:143` |
| `snapshotRfcDiagnosticEmitter` | function | `export function snapshotRfcDiagnosticEmitter( emitter: RfcDiagnosticEmitter, path = "runtime diagnostics", ): RfcDiagnosticEmitter` | `:145-148` |
| `RfcDiagnosticLevels` | type alias | `export type RfcDiagnosticLevels = Readonly< Partial<Record<RfcDiagnosticCategory, RfcDiagnosticLevel>> >;` | `:165-167` |
| `RfcDiagnosticDispatcherOptions` | interface | `{ readonly sink: RfcDiagnosticSink; readonly level?: RfcDiagnosticLevel; readonly levels?: RfcDiagnosticLevels; readonly maxQueued?: number; }` | `:169-174` |
| `RfcDiagnosticMonitor` | interface | `{ readonly closed: boolean; readonly maxQueued: number; readonly queued: number; readonly accepted: number; readonly delivered: number; readonly filtered: number; readonly dropped: number; readonly sinkFailures: number; readonly droppedByCategory: Readonly<Record<RfcDiagnosticCategory, number>>; }` | `:176-186` |
| `createDeferredRfcDiagnosticReporter` | function | `export function createDeferredRfcDiagnosticReporter( emitter: RfcDiagnosticEmitter \| undefined, ): RfcDiagnosticReporter \| undefined` | `:346-348` |
| `RfcDiagnosticDispatcher` | class | `export class RfcDiagnosticDispatcher` | `:431` |
| `BoundedRolloverDiagnosticSinkOptions` | interface | `{ readonly path: string; readonly maxBytes?: number; readonly maxFiles?: number; }` — (comment) `/** Total files including the active file. */` | `:603-608` |
| `createBoundedRolloverDiagnosticSink` | function | `export async function createBoundedRolloverDiagnosticSink( options: BoundedRolloverDiagnosticSinkOptions, ): Promise<RfcDiagnosticSink>` | `:643-645` |

Public members of `RfcDiagnosticDispatcher`: `constructor(options: RfcDiagnosticDispatcherOptions)`
(`:449`), `setLevel(category, level): void` (`:479`), `setLevels(levels): void`
(`:487`), `emit(input: RfcDiagnosticInput): boolean` (`:505`),
`monitor(): RfcDiagnosticMonitor` (`:533`), `async flush(): Promise<void>` (`:547`),
`close(): Promise<void>` (`:552`).

### Configuration options and defaults

| Name | Default (verbatim) | Effect per source | Citation |
|---|---|---|---|
| `sink` | *required* | Must expose callable `write` and, if present, callable `close` | `:458-466` |
| `maxQueued` | `options.maxQueued ?? 1_024` | `!Number.isSafeInteger(maxQueued) \|\| maxQueued < 1 \|\| maxQueued > MAX_QUEUE` → `RangeError`; `MAX_QUEUE = 65_536` | `:467-471`, `:211` |
| `level` | `assertLevel(options.level ?? "info", "diagnostic level")` | Applied to every category | `:472-475` |
| `levels` | `undefined` → not applied | `if (options.levels !== undefined) this.setLevels(options.levels);` | `:476` |
| option key set | — | `exactOptionKeys(options, new Set(["sink", "level", "levels", "maxQueued"]), ...)` — unknown keys and accessors rejected | `:453-457`, `:402-412` |
| sink `maxBytes` | `options.maxBytes ?? 1_048_576` | Range `MAX_EVENT_BYTES (2_048) .. 1_073_741_824` | `:661`, `:663-665` |
| sink `maxFiles` | `options.maxFiles ?? 3` | Range `1..10` | `:662`, `:666-668` |
| sink option key set | — | `FILE_OPTION_KEYS = new Set(["path", "maxBytes", "maxFiles"])` | `:610`, `:649` |
| `MAX_DURATION_MS` | `86_400_000` | Upper bound on `durationMs` | `:210`, `:320-325` |
| `MAX_EVENT_BYTES` | `2_048` | Serialized JSONL line ceiling | `:212`, `:700-702` |
| `RFC_RUNTIME_DIAGNOSTIC_BUFFER_LIMIT` | `256` | Per-runtime deferred-reporter queue bound | `:141`, `:384` |
| `SAFE_CORRELATION_ID` | `/^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/iu` | Only a UUIDv4 shape is admitted | `:208-209` |
| file mode | `0o600` at open **and** an explicit `handle.chmod(0o600)` | Owner-only | `:616-624` |
| open flags | `constants.O_APPEND \| constants.O_CREAT \| constants.O_WRONLY \| noFollow` where `noFollow = "O_NOFOLLOW" in constants ? constants.O_NOFOLLOW : 0` | Refuses to follow symlinks where supported | `:613-618` |

### Errors

| Message/code (verbatim) | Trigger | Citation |
|---|---|---|
| `TypeError(`${path} must expose emit()`)` | emitter not object/function/null, or `emit` not a function | `:149-158` |
| `TypeError(`${path} must be a plain object`)` | event input not a plain object, or with a non-`Object.prototype`/non-null prototype | `:214-221` |
| `TypeError(`${path} must not contain symbol keys`)` | symbol own key on the input | `:222-224` |
| `TypeError(`${path}.${key} is not allowed`)` | any own key outside `INPUT_KEYS` | `:226` |
| `TypeError(`${path}.${key} must be an own data property`)` | accessor property on the input | `:227-230` |
| `TypeError(`${path} is not a supported value`)` | value outside the frozen vocabulary for category/level/code/state/phase/disposition | `:235-244` |
| `TypeError("diagnostic event.code must belong to its category")` | `!code.startsWith(`${category}.`)` | `:290-292` |
| `TypeError("diagnostic event.correlationId is not a safe identifier")` | fails `SAFE_CORRELATION_ID` | `:293-299` |
| `RangeError(`${path} must be a bounded non-negative ${integer ? "integer" : "number"}`)` | `durationMs`/`count` non-finite, negative, over bound, or non-integer where required | `:254-271` |
| `TypeError("diagnostic dispatcher options must be an object")` | non-object/array options | `:450-452` |
| `TypeError(`${path} contains an unsupported option`)` / `TypeError(`${path}.${key} must be an own data property`)` | `exactOptionKeys` | `:402-412` |
| `TypeError("diagnostic sink must expose write() and optional close()")` | sink shape | `:458-464` |
| `RangeError(`diagnostic maxQueued must be an integer from 1 to ${MAX_QUEUE}`)` | out of range | `:468-470` |
| `Error("diagnostic dispatcher is closed")` | `setLevel` or `emit` after `close()` | `:480`, `:506` |
| `TypeError("diagnostic levels must be an object")` | `setLevels` on non-object/array | `:488-490` |
| `TypeError(`diagnostic levels.${category} must be an own data property`)` | accessor in a levels map | `:497-500` |
| `TypeError("diagnostic file options must be an object")` | non-object sink options | `:646-648` |
| `TypeError("diagnostic file path must be a non-empty path")` | non-string, empty, or NUL-containing path | `:650-656` |
| `TypeError("diagnostic file path must name a file")` | `basename(path).length === 0 \|\| dirname(path) === path` | `:658-660` |
| `RangeError("diagnostic maxBytes must be an integer from 2048 to 1073741824")` | out of range | `:663-665` |
| `RangeError("diagnostic maxFiles must be an integer from 1 to 10")` | out of range | `:666-668` |
| `Error("diagnostic destination must be a regular file")` | opened handle is not a regular file (handle is closed first) | `:619-623` |
| `Error("diagnostic destination must be a regular non-symlink file")` | `lstat` says not a file, or a symlink; `ENOENT` is tolerated | `:628-637` |
| `Error("diagnostic file sink is closed")` | `write` after `close` | `:698` |
| `RangeError("diagnostic event exceeds its fixed byte bound")` | serialized line `> MAX_EVENT_BYTES` | `:700-702` |

Swallowed (never surfaced to the caller):
emitter exceptions in the deferred reporter (`:367-370`), invalid observer input
(`:388-391`), `queueMicrotask` scheduling failure — the queue is cleared
(`:377-380`), sink `write` rejection — `#sinkFailures += 1` (`:586-590`), sink
`close` rejection — `#sinkFailures += 1` (`:561-567`).

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Runtime-facing boundary implemented by {@link RfcDiagnosticDispatcher}. Connector state machines bind this interface through a fixed-size deferred queue; arbitrary emitter code is never entered inline with their state transitions. */` | `:131-136` |
| `/** Bind an optional runtime emitter behind a fixed-size, later-microtask queue. The queue deliberately has no close/flush ownership: the application owns the supplied dispatcher, while each runtime owns only this bounded handoff. */` | `:341-345` |
| `// Diagnostics are evidence only and cannot change runtime state.` | `:369` |
| `// An invalid observer event must not alter the authoritative operation.` | `:389` |
| `/** Bounded structured diagnostic dispatcher. Sink callbacks always run from a later microtask and are serialized, so callers never perform observer or file I/O inline with ownership/state transitions. */` | `:426-430` |
| `/** Total files including the active file. */` | `:605` |
| `/** Creates an initialized, owner-readable JSON-lines sink. The directory must already exist; connector code never chooses or creates a trace directory. */` | `:639-642` |

### Behaviour facts asserted by tests

`test/structured-diagnostics.test.ts` — 6 `test(...)` cases.

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| `emit({ ...base, message: "secret" })` throws `/message is not allowed/u`; `code: "network.connect"` under `category: "call"` throws `/must belong to its category/u`; `correlationId: "contains whitespace"` throws `/safe identifier/u`; `durationMs: Number.POSITIVE_INFINITY` throws `/bounded non-negative/u`; an accessor `state` throws `/own data property/u`. | `"structured diagnostics reject payload-like, accessor, mismatched, and unbounded input"` | `test/structured-diagnostics.test.ts:21`, `:30-48` |
| `assert.deepEqual(events, [], "sink must not run inline")`; after `flush()` the event equals `{ schemaVersion: 1, sequence: 1, timestamp: <ISO>, category: "metadata", level: "debug", code: "metadata.cache-hit", count: 1 }`; `Object.isFrozen(event) === true`; monitor and `monitor.droppedByCategory` frozen; `{ accepted: 1, delivered: 1, filtered: 1 }`. | `"dispatcher filters per category, queues asynchronously, and exposes immutable counters"` | `test/structured-diagnostics.test.ts:51`, `:75-100` |
| With `maxQueued: 2`, the 4th `emit` returns `false`; final monitor is `{ accepted: 3, delivered: 1, dropped: 1, poolDropped: 1, sinkFailures: 2 }`; after `close()`, `emit` throws `/closed/u`. Sink failures never propagate to the caller. | `"bounded queue drops deterministically and sink failures cannot affect callers"` | `test/structured-diagnostics.test.ts:103`, `:120-142` |
| File mode is `0o600` for the active file and every rolled file; with `maxBytes: 2_048, maxFiles: 2` and 40 events the directory contains exactly `["connector.jsonl", "connector.jsonl.1"]`; each parsed line has `schemaVersion === 1`, `category === "performance"`, and **`assert.equal("message" in event, false)`**. | `"rollover sink initializes before use, stays owner-only, and keeps a bounded JSONL set"` | `test/structured-diagnostics.test.ts:145`, `:160-187` |
| The complete rejected-value matrix: `category: "unknown"`, `level: "verbose"`, `code: "call.unknown"`, `correlationId: 7`, `correlationId: "x".repeat(129)`, `state: "unknown"`, `phase: "unknown"`, `disposition: "unknown"`, `durationMs: -1`, `durationMs: 86_400_001`, `count: 1.5`, `count: Number.MAX_SAFE_INTEGER + 1`. A fully-populated valid event round-trips exactly. `close()` is idempotent (`secondClose === firstClose`, `closeCalls === 1`), a failing sink `close` yields `sinkFailures === 1`, and `setLevel` after close throws `/closed/u`. | `"dispatcher validates configuration, levels, and the complete event vocabulary"` | `test/structured-diagnostics.test.ts:190`, `:265-322` |
| Rejects `null`/`[]`/`"bad"` options, `""`/`"bad\0path"`/`7`/`"/"` paths, `maxBytes` of `NaN`/`2_047`/`1_073_741_825`, `maxFiles` of `NaN`/`0`/`11`; rejects a directory and a symlink with `/regular non-symlink file/u`; `maxFiles: 1` leaves only `["single.jsonl"]`; an oversized event rejects `/fixed byte bound/u`; double `close()` is safe and a post-close `write` rejects `/sink is closed/u`. | `"file sink rejects unsafe destinations and covers single-file rollover"` | `test/structured-diagnostics.test.ts:325`, `:326-400` |

---

## Locking discipline

The upstream architecture document states (verbatim, `docs/architecture.md:51-52`):

> **Logging, monitor publication, observer callbacks, and file I/O never run
> while a pool, repository, lifecycle, or context lock is held.**

The Go port's own copy already carries the adapted form
(`../architecture.md:58-59` in open-rfc-go: "Logging, metrics, observer
callbacks, and file I/O never run while a pool, repository, lifecycle, or
context lock is held.").

### What plays the role of a "lock" in the TypeScript source

There is no mutex primitive anywhere in the three directories:
`grep -rniE 'mutex|\block\b' src/pool src/destination src/diagnostics` returns no
matches. The exclusive regions are **synchronous JavaScript turns** plus explicit
boolean re-entrancy guards. The three guards are:

1. `#dispatching` / `#pumpScheduled` / `#pumpRequestedWhileDispatching` on the
   pool (`src/pool/connection-pool-runtime.ts:441-443`).
2. `#scheduled` / `#draining` on the dispatcher
   (`src/diagnostics/structured-diagnostics.ts:439-440`) and the same pair as
   closure variables in the deferred reporter (`:353-354`).
3. `record.active` / `record.tail` / `record.state` on an application lease
   (`src/destination/direct-destination-owner.ts:354-359`).

INFERRED: because a JS turn is the critical section, "never hold the lock while
calling out" is expressed as "never call out synchronously inside the turn that
mutates state". Every mechanism below is an instance of that.

### Where state transitions happen and what is deferred out of them

**Pool dispatch.** `#requestPump` never dispatches inline. It sets
`#pumpScheduled = true` and defers the whole dispatch loop into
`queueMicrotask`; if a pump is requested *during* dispatch it only records
`#pumpRequestedWhileDispatching = true` and re-requests in the `finally`
(`src/pool/connection-pool-runtime.ts:1618-1639`). The dispatch loop itself is
`async #dispatch()` (`:1641`), so all factory calls (`#validate` at `:1661-1663`)
happen from awaited continuations, not from the caller's `acquire()` frame.

**Pool diagnostics.** The pool never calls the caller's emitter. It stores
`this.#report = createDeferredRfcDiagnosticReporter(diagnostics);`
(`src/pool/connection-pool-runtime.ts:551`). That factory validates and snapshots
the input **synchronously** into a bounded array, then drains via
`queueMicrotask` (`src/diagnostics/structured-diagnostics.ts:352-395`). The
snapshot is taken before queueing (`:387`) so a later caller mutation cannot
change what is delivered. The comment states the contract
(`:341-345`, `:131-136`).

Where the pool reports, it reports **after** the transition:

- Acquire success/failure are reported in `promise.then(...)` continuations of
  `#observeAcquire`, i.e. after the waiter has already settled
  (`src/pool/connection-pool-runtime.ts:755-791`).
- Shutdown terminal events are reported in `void this.#closePromise.then(...)`
  (`:1092-1125`), never inside `#beginShutdown`'s synchronous body.
- The one report inside a synchronous body — `pool.wait` at `:720-728` — is
  emitted *after* `this.#waiters.push(waiter)` at `:719`, and it only enqueues.

**Monitor publication.** `monitor()` on all four runtimes is a pure read of
already-committed fields plus `Object.freeze` of a fresh object; it invokes no
caller code and performs no I/O:
`src/pool/connection-pool-runtime.ts:1129-1193`,
`src/destination/configuration-generation.ts:405-412`,
`src/destination/runtime.ts:402-407`,
`src/destination/direct-destination-owner.ts:2391-2424`,
`src/diagnostics/structured-diagnostics.ts:533-545`.

**Retirement.** Three separate places publish the terminal promise *before* the
body that can call back into caller code runs:

- `DestinationConfigurationGeneration.retire`:
  `retirement = Promise.resolve().then(async () => { ... })` (`:349`), then
  `this.#retirement = retirement;` (`:391`), with the comment
  `// Publish before the microtask above invokes any hook.` (`:388-390`).
- `RfcDestinationRuntime.retire`: same shape (`src/destination/runtime.ts:357`,
  `:388`), comment `// The retirement body starts in a later microtask, so
  owning generation hooks can never run before external callers can observe this
  operation.` (`:386-387`).
- `DirectDestinationOwner.retire`: `this.#retirement = retirement;` (`:2384`) is
  executed *before* `contextRetirement = this.#contexts.retire();` (`:2387`),
  with the comment at `:2381-2383`.

**Lease-local operation tails.** `#runApplicationOperation` sets
`record.active = true` and publishes `record.tail` synchronously, then starts the
operation from a microtask: `return Promise.resolve().then(operation)`
(`src/destination/direct-destination-owner.ts:2800-2812`), comment
`// Begin in a microtask so reentrant release always observes the published
tail.` (`:2803`). `releaseApplication` performs the once-only ownership handoff
synchronously (`record.state = "releaseClaimed";`, `:2222`, comment `:2221`) and
only then awaits the tail (`:2225`).

### How callbacks are dispatched

The dispatcher is the only component that runs caller-supplied sinks, and it
does so from a dedicated serialized drain:

- `emit()` validates, freezes and enqueues, then calls `#scheduleDrain()`
  (`src/diagnostics/structured-diagnostics.ts:505-531`).
- `#scheduleDrain()` returns immediately if `#scheduled || #draining`, otherwise
  arms one `queueMicrotask` (`:570-577`).
- `#drain()` is guarded by `#draining` and processes the queue one event at a
  time with `await Reflect.apply(this.#sink.write, this.#sink, [event]);`
  (`:579-600`). Two drains never overlap.
- File I/O inside the rollover sink is itself serialized behind a promise chain:
  `const operation = tail.then(async () => { ... }); tail = operation.catch(() => {});`
  (`:703-711`), and `close()` chains on the same tail (`:713-720`).

Failure containment: an emitter throw is caught and discarded (`:365-371`), an
invalid event returns `false` without throwing (`:386-391`), and a sink write or
close rejection only increments `#sinkFailures` (`:586-590`, `:561-567`).

Back-pressure is **drop, not block**: the runtime reporter refuses at
`RFC_RUNTIME_DIAGNOSTIC_BUFFER_LIMIT = 256` (`:141`, `:384`) and the dispatcher
drops at `#maxQueued` while incrementing `#dropped` and
`#droppedByCategory[snapshot.category]` (`:516-520`).

### Test evidence that callbacks are outside transitions

| Evidence | Citation |
|---|---|
| Test name: `"low-level pool diagnostics cover wait, acquire, release, retire, and shutdown outside transitions"` | `test/runtime-diagnostics-integration.test.ts:47` |
| The emitter under test deliberately re-enters the pool (`pool.monitor()`), with the comment `// If this executes synchronously inside a transition it can reenter the state machine. The deferred boundary must instead see reconciled state.` | `test/runtime-diagnostics-integration.test.ts:52-61` |
| `const acquiring = pool.acquireOne();` then `assert.equal(observerRanInline, false);` — the observer has not run at the end of the acquiring turn | `test/runtime-diagnostics-integration.test.ts:76-77` |
| After draining: `assert.equal(observerRanInline, true);` and `assert.equal(observerInputFrozen, true);` | `test/runtime-diagnostics-integration.test.ts:83-84` |
| Exact ordering: `["pool.wait","pool.acquire","pool.release","pool.retired","pool.shutdown","pool.retired","pool.closed"]`, all frozen, none containing `message`, with correlation ids paired `(0,1)`, `(2,3)`, `(4,5)`, `(5,6)` | `test/runtime-diagnostics-integration.test.ts:85-102` |
| `assert.deepEqual(events, [], "sink must not run inline");` | `test/structured-diagnostics.test.ts:75` |
| `assert.equal(delivered, 0, "observer must not run inline");` after 400 synchronous `invalidate` calls, then `assert.equal(delivered, 256, "one runtime reporter has a fixed buffer bound");` while `repository.monitor().invalidations` is still `400` — i.e. dropped diagnostics do not change authoritative state | `test/runtime-diagnostics-integration.test.ts:324-330` |
| Test name: `"runtime diagnostic buffering is bounded and observer failures never change metadata state"`; its emitter always throws (`throw new Error("hostile observer")`) | `test/runtime-diagnostics-integration.test.ts:293-299` |
| The test helper `drain()` documents the two-hop deferral: `// Runtime reporters defer the external emitter; the dispatcher then defers its sink. Cross both scheduling boundaries before asserting evidence.` | `test/runtime-diagnostics-integration.test.ts:26-33` |

### Go design implication

1. **`sync.Mutex` scope must be the state mutation only.** Each of
   `ConnectionPoolRuntime`, `DestinationConfigurationGeneration`,
   `RfcDestinationRuntime` and `DirectDestinationOwner` becomes a struct with one
   `sync.Mutex` protecting exactly the fields the TS `#private` fields cover
   (e.g. `#records`, `#idle`, `#creating`, `#waiters`, `#leased`, `#state`, and
   the failure counters at `src/pool/connection-pool-runtime.ts:423-457`). No
   factory hook (`create`/`validate`/`reset`/`destroy`,
   `src/pool/connection-pool-runtime.ts:66-83`), no scheduler call
   (`:43-46`), no emitter call, and no `os.File.Write` may appear between
   `Lock()` and `Unlock()`. The clock reads `#readClock`/`#readLifecycleClock`
   (`:1195-1213`) call *caller-supplied* `now()`, and the source already treats
   that as a reentrancy hazard (`:678-679`, `:1585-1586`) — in Go those calls
   must also move outside the lock, or the scheduler interface must be
   documented as non-reentrant.

2. **Collect events under the lock; emit after unlock.** Mirror
   `createDeferredRfcDiagnosticReporter` exactly: while holding the mutex, append
   a *value copy* of the event to a local slice (the TS analogue is
   `snapshotInput` at `src/diagnostics/structured-diagnostics.ts:273-339`, which
   builds a fresh frozen object before queueing at `:392`). Unlock, then hand the
   slice to the dispatcher. Because `RfcDiagnosticInput` has only scalar fields
   (`:108-118`), the Go struct is copyable by value and no aliasing question
   arises — this is why the key allowlist matters structurally, not just for
   redaction.

3. **Channel buffering with non-blocking send.** The bounds are explicit:
   `RFC_RUNTIME_DIAGNOSTIC_BUFFER_LIMIT = 256` per runtime
   (`src/diagnostics/structured-diagnostics.ts:141`) and dispatcher `maxQueued`
   default `1_024`, max `65_536` (`:467`, `:211`). In Go:
   `make(chan Event, 256)` per runtime reporter and `make(chan Event, maxQueued)`
   in the dispatcher, with `select { case ch <- ev: default: dropped++;
   droppedByCategory[ev.Category]++ }`. A blocking send would reintroduce exactly
   the coupling the invariant forbids — the TS code returns `false` rather than
   waiting (`:384`, `:516-520`). One drain goroutine per dispatcher reproduces
   the `#draining` serialization (`:579-600`); `flush()` becomes a
   `sync.WaitGroup` or a reply channel matching `#flushWaiters` (`:436`,
   `:547-550`, `:596-598`).

4. **Why `go test -race` plus `-mutexprofile` can enforce this mechanically.**
   The TypeScript version can only *assert* the property behaviourally, with
   `observerRanInline` flags (`test/runtime-diagnostics-integration.test.ts:50`,
   `:77`) and `"sink must not run inline"` (`test/structured-diagnostics.test.ts:75`) —
   because single-threaded JS cannot exhibit a data race at all. Go can do
   better on both halves:
   - `go test -race` detects the *failure mode* the TS guards are standing in for.
     If a factory hook, scheduler callback, or diagnostic sink reads pool state
     that a concurrent goroutine mutates under the mutex, the race detector flags
     it. In TS that same bug shows up only as reentrancy, which is why the source
     needs the `#dispatching` guard (`src/pool/connection-pool-runtime.ts:1620-1623`)
     and the explicit "external boundary may reenter" comments (`:678-679`,
     `:1485-1489`, `:1585-1586`). Those comments become race-detector assertions.
   - `go test -mutexprofile` measures *contention*, which is the direct proxy for
     lock hold time. If an emitter, a `time.Timer` callback, or a file write ever
     runs inside the critical section, the profile shows contention proportional
     to sink latency — the blocked-sink case that
     `test/structured-diagnostics.test.ts:103-137` constructs deliberately (the
     first write awaits `blocked` and the test still asserts callers are
     unaffected). A regression test can therefore assert an upper bound on
     cumulative mutex delay while a sink is artificially blocked, which is a
     mechanical check the TS suite cannot express.
   - Combine with `-cpu=1,4` so the FIFO ordering asserted at
     `test/connection-pool-runtime.test.ts:829` and `:858` is exercised under
     real parallelism rather than the TS microtask order.

---

## Generation and identity model

### How a destination generation is computed

A generation is one immutable `DestinationConfigurationGeneration` instance. Its
`configuration` is computed once in the constructor and then locked down twice —
`Object.freeze` (`src/destination/configuration-generation.ts:300-304`) followed
by an explicit `Object.defineProperty(this, "configuration", { value:
this.configuration, writable: false, enumerable: true, configurable: false })`
(`:305-310`).

Inputs (all nine required, each `safeIdentity`-checked to 1..512 chars without
C0/DEL controls, `:151-163`, `:171-190`):
`destinationId`, `endpointId`, `systemId`, `client`, `release`,
`metadataGeneration`, `language`, `applicationPrincipalId`,
`repositoryPrincipalId` (`:17-29`).

### The hash — what is hashed and what is not

```
function backendKey(identity: CanonicalDestinationIdentity): string {
  const canonicalIdentity = JSON.stringify([
    identity.endpointId,
    identity.systemId,
    identity.client,
    identity.release,
    identity.metadataGeneration,
    identity.language,
  ]);
  return `sha256:${createHash("sha256")
    .update("open-rfc:metadata-backend:v1\u0000", "utf8")
    .update(canonicalIdentity, "utf8")
    .digest("hex")}`;
}
```
— `src/destination/configuration-generation.ts:193-206`.

Facts the porter must preserve exactly:

- **Domain separation prefix**: the literal `"open-rfc:metadata-backend:v1\u0000"`
  is hashed first, as its own `update` call, UTF-8, terminated by a NUL byte
  (`:203`). It is a version-tagged domain separator; changing it changes every
  key.
- **Canonicalization is `JSON.stringify` of a 6-element array**, in that field
  order (`:194-201`). This is what makes `["A","B"]` and `["AB",""]`
  unambiguous — JSON quoting and escaping do the framing. A Go port must
  reproduce JSON string escaping, not just concatenate with a separator, or the
  keys will differ from upstream.
- **Six fields are hashed**: `endpointId`, `systemId`, `client`, `release`,
  `metadataGeneration`, `language`.
- **Three fields are deliberately excluded**: `destinationId`,
  `applicationPrincipalId`, `repositoryPrincipalId`. Two backends with different
  `destinationId` but the same endpoint/system/client/release/metadata
  generation/language share a `structuralBackendKey`.
- **Output form**: `sha256:` + 64 lowercase hex chars = 71 chars. Test:
  `assert.equal(backendKey.length, 71)`
  (`test/destination-configuration-generation.test.ts:160`) and
  `assert.match(configuration.identity.structuralBackendKey, /^sha256:[0-9a-f]{64}$/u)`
  (`:77-80`).

### Cache / identity keys

There are four distinct keys.

1. **`structuralBackendKey`** — the hash above. Stored on
   `DestinationSafeIdentity` (`src/destination/configuration-generation.ts:38`).
   Same for both lanes: the test asserts
   `applicationCapability.backendKey === repositoryCapability.backendKey`
   (`test/destination-configuration-generation.test.ts:69-72`) and
   `structuralBackendKey === applicationCapability.backendKey` (`:73-76`).

2. **`applicationCapability` / `repositoryCapability`** —
   `createMetadataCapabilityKey({ backendKey: structuralBackendKey, principalKey:
   canonicalIdentity.applicationPrincipalId })` and the repository equivalent
   (`src/destination/configuration-generation.ts:291-298`). These are the
   *principal-scoped* keys; the test asserts they differ by `.id`
   (`test/destination-configuration-generation.test.ts:65-68`) while sharing a
   backend key, and that `.principalKey` round-trips the input verbatim
   (`:169-176`). `MetadataRepositoryMode` and `createMetadataCapabilityKey` live
   in `src/metadata/repository-runtime.js`, outside this inventory's scope.

3. **`MetadataStructuralKey`** — computed per lookup, not stored:

   ```
   const structural = createMetadataStructuralKey({
     backendKey: identity.structuralBackendKey,
     metadataGeneration: identity.metadataGeneration,
     language: identity.language,
     objectKind,
     objectName,
   });
   ```
   — `src/destination/runtime.ts:418-424`. The full repository lookup adds the
   principal capability and the mode:
   `Object.freeze({ structural, capability: identity.repositoryCapability, mode })`
   (`:426-431`). Note `metadataGeneration` and `language` appear **both** inside
   the backend-key hash and again as explicit structural-key fields.

4. **Optimized generation-token key** — a separate, owner-local index:
   `` return `${objectKind}\n${objectName}`; `` where `objectKind` is
   `"function" | "recursive-function" | "structure"`
   (`src/destination/direct-destination-owner.ts:1278-1283`). The newline is the
   separator; `objectName` is already constrained to 1..30 printable ASCII
   (`:411-421`), so it cannot contain one.

### What invalidates a generation / a cached descriptor

| Trigger | Effect | Citation |
|---|---|---|
| Descriptor kind/name mismatch from the adapter | `this.#repository.invalidate(structural);` then throw — comment `// A malformed adapter result must not become a permanent cache poison.` | `src/destination/runtime.ts:433-439` |
| A non-optimized load (`context.strategy !== MetadataLoadStrategy.Optimized`) | `this.#optimizedGenerationTokens.delete(generationKey);` **before** the load, comment `// A classic or legacy fallback must never inherit a token from an older optimized descriptor for the same structural key.` | `src/destination/direct-destination-owner.ts:2504-2508` |
| Optimized load returning no token (legacy injected session) | `delete` from both the key map and the descriptor WeakMap | `:2592-2595` |
| Optimized load returning a token | `#optimizedDescriptorTokens.set(value.value, canonicalToken)` and `#rememberOptimizedGeneration(context.structural, generationToken)` | `:2596-2603` |
| `refreshOptimizedMetadata` sees a changed token or a typed error for a tracked name | `this.#repository.invalidate(record.structural); this.#optimizedGenerationTokens.delete(key);` and the name is added to `invalidatedFunctionNames` / `invalidatedStructureNames` | `:2696-2732` |
| A descriptor reloaded while the timestamp call was in flight | `if (this.#optimizedGenerationTokens.get(key) !== record) continue;` — identity comparison of the frozen record object, not the token string | `:2716`, `:2728`, comment `:2714-2715` |
| Untracked names (`records.length === 0` / `record === undefined`) | Never invalidated: `invalidateFunctions.push(records.length > 0 && (...))` and `invalidateStructures.push(record !== undefined && (...))` | `:2667-2692` |
| Owner retirement | `this.#optimizedGenerationTokens.clear();` and `this.#metadataRefreshRetirement.abort(metadataRefreshCanceled());` | `:2385-2386` |
| Token-map pressure | `#rememberOptimizedGeneration` deletes the key, then evicts from the front of the `Map` (`keys().next().value`) while `size >= #maxOptimizedGenerationTokens`, then re-inserts — insertion-order eviction, bounded by `metadata.maxEntries` (default `512`) | `:2610-2644`, `:1833`, `:388` |
| Generation retirement (lane level) | `#state` moves `active → retiring → retired`; `#openLane` rejects with `` `destination generation ${id} is retired` ``; opens admitted before the transition are drained and their late connections disposed | `src/destination/configuration-generation.ts:323-393`, `:414-454` |

### Token grammar

`canonicalOptimizedGenerationToken` accepts only two shapes
(`src/destination/direct-destination-owner.ts:811-824`):

- `function` and `recursive-function`: `/^function:\d{8}:\d{6}$/u`
- `structure`: `/^structure:\d{14}$/u`

A recursive descriptor must additionally self-agree:
`descriptor.value.functionIdentity?.name !== name ||
descriptor.value.functionIdentity.generationToken !== descriptor.generationToken`
throws (`:1009-1017`). And at invocation time a flat descriptor and a recursive
graph carrying different tokens is a hard error
(`:2089-2103`), test `"rejects mixed optimized function generations before
application entry"` (`test/direct-destination-owner.test.ts:1718`).

---

## Redaction rules

Every rule below is a security property. Quotes are verbatim.

### R1 — The diagnostic event key allowlist is closed

```
const INPUT_KEYS = new Set([
  "category",
  "level",
  "code",
  "correlationId",
  "state",
  "phase",
  "disposition",
  "durationMs",
  "count",
]);
```
— `src/diagnostics/structured-diagnostics.ts:197-207`.

Any other own key throws: `` throw new TypeError(`${path}.${key} is not allowed`); ``
(`:226`). There is **no** `message`, `error`, `detail`, `host`, `user`,
`functionName`, `parameters`, or `resource` field anywhere in
`RfcDiagnosticInput` (`:108-118`) or `RfcDiagnosticEvent` (`:120-124`).

Test: `assert.throws(() => dispatcher.emit({ ...base, message: "secret" } as never), /message is not allowed/u);`
— `test/structured-diagnostics.test.ts:30-33`.
Test: `assert.equal("message" in event, false);` for every persisted JSONL line —
`test/structured-diagnostics.test.ts:185`.
Test: `assert.equal(events.some((event) => "message" in event), false);` —
`test/runtime-diagnostics-integration.test.ts:98`.

### R2 — Symbol keys are rejected

`` if (typeof key !== "string") { throw new TypeError(`${path} must not contain symbol keys`); } ``
— `src/diagnostics/structured-diagnostics.ts:222-224`. Test seeds
`symbolInput[Symbol("hidden")] = "secret";` and expects `/symbol keys/u`
(`test/structured-diagnostics.test.ts:269-271`).

### R3 — Accessor properties are rejected (no getter can smuggle a payload)

`` if (descriptor === undefined || !("value" in descriptor)) { throw new TypeError(`${path}.${key} must be an own data property`); } ``
— `src/diagnostics/structured-diagnostics.ts:227-230`; same rule for dispatcher
options (`:406-410`) and for `levels` entries (`:497-500`). Test at
`test/structured-diagnostics.test.ts:46-48`.

### R4 — `correlationId` must be a UUIDv4, nothing else

```
const SAFE_CORRELATION_ID =
  /^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/iu;
```
— `src/diagnostics/structured-diagnostics.ts:208-209`; violation throws
`"diagnostic event.correlationId is not a safe identifier"` (`:298`). The pool
generates them with `randomUUID()` and silently omits the field if that throws
(`src/pool/connection-pool-runtime.ts:271-277`). This is the only free-form
string field in the whole event, and it is constrained to a value with no
caller-controlled content.

### R5 — Numeric fields are bounded

`durationMs` ≤ `MAX_DURATION_MS = 86_400_000` and non-negative finite
(`src/diagnostics/structured-diagnostics.ts:210`, `:320-325`); `count` a
non-negative safe integer (`:326-331`). The pool clamps before reporting:
`Number.isFinite(elapsed) ? Math.min(86_400_000, Math.max(0, elapsed)) : 0`
(`src/pool/connection-pool-runtime.ts:257-259`), and only forwards `count` when
`Number.isSafeInteger(count) && count >= 0` (`:785`).

### R6 — `code` must belong to its `category`

`` if (!code.startsWith(`${category}.`)) { throw new TypeError("diagnostic event.code must belong to its category"); } ``
— `src/diagnostics/structured-diagnostics.ts:290-292`. Prevents a caller
mislabelling an event into a category whose configured level would let it
through.

### R7 — Every event is size-bounded on disk

`const MAX_EVENT_BYTES = 2_048;` (`:212`) enforced at the sink:
`if (line.byteLength > MAX_EVENT_BYTES) { return Promise.reject(new RangeError("diagnostic event exceeds its fixed byte bound")); }`
(`:700-702`). `maxBytes` cannot be configured below that (`:663-665`).

### R8 — Trace files are owner-only, regular, non-symlink

- Open flags include `O_NOFOLLOW` where available and mode `0o600`
  (`src/diagnostics/structured-diagnostics.ts:613-618`).
- `if (!status.isFile()) { await handle.close(); throw new Error("diagnostic destination must be a regular file"); }` (`:619-623`).
- Explicit `await handle.chmod(0o600);` after open (`:624`).
- Pre-open `lstat` check: `if (!status.isFile() || status.isSymbolicLink()) { throw new Error("diagnostic destination must be a regular non-symlink file"); }` (`:628-637`).
- (comment) `/** Creates an initialized, owner-readable JSON-lines sink. The directory must already exist; connector code never chooses or creates a trace directory. */` (`:639-642`).
- Test asserts `(await lstat(path)).mode & 0o777 === 0o600` on the active file
  and every rolled file (`test/structured-diagnostics.test.ts:159-161`, `:176-178`),
  and that a directory and a symlink both reject with `/regular non-symlink file/u`
  (`:355-367`).

### R9 — Diagnostics never receive resources

(comment) `/** Optional bounded structured diagnostics; never receives resources. */`
— `src/pool/connection-pool-runtime.ts:105`. Structurally guaranteed by R1: the
event type has no field capable of holding one.

### R10 — Physical resources are never exposed through lease tokens

- (comment) `/** An immutable ownership token. Physical resources are deliberately not exposed; all use must pass through `withActiveLease()`. */`
  — `src/pool/connection-pool-runtime.ts:137-140`. `ConnectionPoolLease` carries
  only `poolId` and `generation` (`:141-144`).
- (comment) `/** Nominal, resource-free token. Its DirectCpicSession remains owner-private. */`
  — `src/destination/direct-destination-owner.ts:176`. The token is
  `Object.create(null)` with a single non-enumerable symbol brand, then frozen
  (`:2001-2008`).
- (comment) `/** The only resource-use boundary. Callers must not retain the callback-scoped resource; ownership and single-flight tracking end when the Promise settles. */`
  — `src/pool/connection-pool-runtime.ts:793-796`.
- (comment) `Raw sessions exist only inside captured lane resources.`
  — `src/destination/direct-destination-owner.ts:1760`.

### R11 — Validation failures never expose the resource

(comment) `// Validation errors are health failures and never expose the resource.`
— `src/pool/connection-pool-runtime.ts:1742`. `#validate` returns
`{ physical, healthy: false, ...(lifecycle.expired ? { error } : {}) }` — the
underlying error is propagated **only** when the lifecycle deadline expired
(`:1744-1748`).

### R12 — Only opaque, non-secret identities are stored or exposed

- (comment) `/** One immutable destination configuration generation. Credential ownership is kept behind the two factories; only opaque, non-secret identities are stored or exposed by this runtime. */`
  — `src/destination/configuration-generation.ts:249-253`.
- (comment) `/** Opaque non-secret identity, such as a vault/principal fingerprint. */` (`:25`)
  and `/** Opaque non-secret identity for the independent repository lane. */` (`:27`).
- `DestinationSafeIdentity` (`:31-41`) **omits** `endpointId`,
  `applicationPrincipalId` and `repositoryPrincipalId` as raw fields; the first
  survives only inside the hash, the latter two only inside the capability keys.
- Test: `JSON.stringify(configuration)` contains none of `"qas.example.invalid"`
  (the endpoint host), `password`, `passwd`, `secret`, `credential`,
  `applicationFactory`, `repositoryFactory`
  (`test/destination-configuration-generation.test.ts:86-97`).

### R13 — The owner monitor exposes no credential

`assert.equal("password" in monitor, false);`
— `test/direct-destination-owner.test.ts:1259`. `DirectDestinationOwnerMonitor`
(`src/destination/direct-destination-owner.ts:181-199`) is entirely counters,
enum states and nested monitors.

### R14 — Hostile third-party error text is never re-thrown

`refreshMapEntries` catches every iterator/`next`/property failure and rethrows a
fixed `TypeError` string, never the caught error or its message
(`src/destination/direct-destination-owner.ts:1358-1453`; e.g. `:1371-1375`,
`:1381-1386`, `:1407-1414`). Test:
`error instanceof TypeError && !error.message.includes(secret)` where
`secret = "private iterator payload"`, plus
`assert.equal(pairGetterCalled, false)`
(`test/direct-destination-owner.test.ts:2133`, `:2182-2185`, `:2190`).

### R15 — Runtime diagnostics carry no error text and no identifiers (end-to-end)

| Assertion (verbatim) | Test name (verbatim) | Citation |
|---|---|---|
| `assert.doesNotMatch(serialized, /waiter limit\|timed out/u);` — the pool's own error messages never reach the event stream | `"pool timeout and rejection diagnostics expose classifications without error text"` | `test/runtime-diagnostics-integration.test.ts:143-144` |
| `assert.doesNotMatch(JSON.stringify(events), /secret-backend\|secret-principal\|SECRET_FUNCTION\|private adapter failure/u);` — no backend key, principal key, object name, or adapter error text | `"metadata diagnostics distinguish lookup, miss, hit, failure, and invalidation without identities"` | `test/runtime-diagnostics-integration.test.ts:204-207` |
| `assert.doesNotMatch(JSON.stringify(events), /secret-host\|secret-user\|secret-password\|Z_SECRET\|TOKEN\|private response/u);` — no host, user, password, RFC name, parameter name, or response value | `"compatibility Client diagnostics cover open, invoke, cancel, and close without request data"` | `test/runtime-diagnostics-integration.test.ts:286-289` |

### R16 — A rejected diagnostic never alters the operation

(comment) `// An invalid observer event must not alter the authoritative operation.`
(`src/diagnostics/structured-diagnostics.ts:389`) and
`// Diagnostics are evidence only and cannot change runtime state.` (`:369`).
Test: 400 invalidations with an always-throwing emitter still yield
`repository.monitor().invalidations === 400`
(`test/runtime-diagnostics-integration.test.ts:330`).

---

## Diagnostics event model

All six enumerations are `Object.freeze([...] as const)` tuples; the derived
types are `(typeof X)[number]`. Membership is enforced at runtime through `Set`s
built from the same tuples (`src/diagnostics/structured-diagnostics.ts:188-193`).

### `RfcDiagnosticCategory` — 7 members

`"call"`, `"metadata"`, `"network"`, `"lifecycle"`, `"pool"`, `"locking"`,
`"performance"` — `src/diagnostics/structured-diagnostics.ts:11-19`.

### `RfcDiagnosticLevel` — 5 members, ordered

`"error"`, `"warn"`, `"info"`, `"debug"`, `"trace"` —
`src/diagnostics/structured-diagnostics.ts:22-28`.

Order is semantic: `LEVEL_INDEX` maps each to its array index (`:194-196`) and
filtering is `LEVEL_INDEX.get(snapshot.level)! > LEVEL_INDEX.get(configured)!`
→ filtered (`:509-515`). So index 0 = `error` is the most severe and the
threshold is "index ≤ configured index".

### `RfcDiagnosticCode` — 35 members

`src/diagnostics/structured-diagnostics.ts:31-67`:

| Category | Codes |
|---|---|
| `call` | `"call.started"`, `"call.succeeded"`, `"call.failed"`, `"call.canceled"`, `"call.timed-out"` |
| `metadata` | `"metadata.lookup"`, `"metadata.cache-hit"`, `"metadata.cache-miss"`, `"metadata.invalidated"`, `"metadata.failed"` |
| `network` | `"network.connect"`, `"network.opened"`, `"network.closed"`, `"network.failed"`, `"network.timed-out"` |
| `lifecycle` | `"lifecycle.opened"`, `"lifecycle.reset"`, `"lifecycle.replaced"`, `"lifecycle.closed"`, `"lifecycle.failed"` |
| `pool` | `"pool.acquire"`, `"pool.release"`, `"pool.wait"`, `"pool.timed-out"`, `"pool.rejected"`, `"pool.retired"`, `"pool.shutdown"`, `"pool.closed"`, `"pool.failed"` |
| `locking` | `"locking.wait"`, `"locking.acquired"`, `"locking.released"`, `"locking.contention"` |
| `performance` | `"performance.sample"`, `"performance.budget-exceeded"` |

The `locking.*` codes exist in the vocabulary but are **not emitted anywhere in
the three directories in scope** (`grep` for `locking.` in `src/pool`,
`src/destination`, `src/diagnostics` matches only the declaration at
`src/diagnostics/structured-diagnostics.ts:61-64`).

### `RfcDiagnosticState` — 9 members

`"connecting"`, `"open"`, `"closing"`, `"closed"`, `"retired"`, `"waiting"`,
`"leased"`, `"idle"`, `"failed"` — `src/diagnostics/structured-diagnostics.ts:70-80`.

### `RfcDiagnosticPhase` — 12 members

`"connect"`, `"logon"`, `"metadata"`, `"encode"`, `"send"`, `"receive"`,
`"decode"`, `"reset"`, `"cancel"`, `"close"`, `"acquire"`, `"release"` —
`src/diagnostics/structured-diagnostics.ts:83-96`.

### `RfcDiagnosticDisposition` — 4 members

`"reusable"`, `"close"`, `"unknownClose"`, `"replace"` —
`src/diagnostics/structured-diagnostics.ts:99-104`.

### Envelope fields added by the dispatcher

```
const event = Object.freeze({
  schemaVersion: 1 as const,
  sequence: ++this.#sequence,
  timestamp: new Date().toISOString(),
  ...snapshot,
});
```
— `src/diagnostics/structured-diagnostics.ts:521-526`. `sequence` starts at 1
(`#sequence = 0` at `:438`, pre-incremented). `timestamp` is ISO-8601; the test
matches `/^\d{4}-\d{2}-\d{2}T/u` (`test/structured-diagnostics.test.ts:92`).
Note the spread order: `...snapshot` comes **last**, but `snapshotInput` has
already rejected any key outside `INPUT_KEYS` (R1), so it cannot overwrite
`schemaVersion`, `sequence` or `timestamp`.

### Emission map observed in the pool

| Code | level | state | phase | Other | Citation |
|---|---|---|---|---|---|
| `pool.wait` | `debug` | `waiting` | `acquire` | `count` | `src/pool/connection-pool-runtime.ts:720-728` |
| `pool.acquire` | `info` | `leased` | `acquire` | `durationMs`, `count: leases.length` | `:763-772` |
| `pool.timed-out` / `pool.rejected` | `warn` | `failed` | `acquire` | `durationMs`, conditional `count` | `:776-787` |
| `pool.release` | `info` | `idle` \| `retired` | `release` | `disposition: reusable ? "reusable" : "close"`, `durationMs`, `count: 1` | `:964-974` |
| `pool.retired` (non-reusable release) | `info` | `retired` | — | `disposition: "close"`, `count: 1` | `:976-984` |
| `pool.shutdown` | `info` | `closing` | `close` | — | `:1031-1038` |
| `pool.retired` (retire path) | `info` | `retired` | `close` | — | `:1095-1102` |
| `pool.closed` | `info` | `closed` | `close` | `durationMs` | `:1104-1112` |
| `pool.failed` | `error` | `failed` | `release` \| `close` | `durationMs` | `:946-954`, `:1115-1123` |

### Can `log`/`slog` express this?

**Partly, and not the parts that matter.**

What `log/slog` gives for free: structured key/value output, level filtering, a
JSON handler, and `slog.Handler` as a sink seam. `RfcDiagnosticSink`
(`src/diagnostics/structured-diagnostics.ts:126-129`) maps cleanly onto
`slog.Handler.Handle`.

What it cannot express, each backed by a cited requirement:

1. **A closed key allowlist.** `slog` accepts arbitrary `slog.Attr`s by design.
   R1 (`:197-207`, `:226`) is the central security property and `slog` has no
   mechanism to enforce it — a single `slog.String("functionName", name)`
   anywhere in the codebase silently violates it.
2. **A closed value vocabulary.** `slog.Level` is an `int` with `Debug/Info/Warn/Error`
   only; there is no `trace` (`:22-28` has 5 levels), and category/code/state/
   phase/disposition have no `slog` analogue at all.
3. **The category↔code prefix rule.** `:290-292` has no `slog` equivalent.
4. **Bounded queue with per-category drop accounting.** `slog` handlers are
   synchronous by contract; there is no built-in queue, no `maxQueued`
   (`:467`), no `dropped`/`droppedByCategory` (`:516-520`), and no
   `sinkFailures` (`:586-590`). A `slog.Handler` that panics or blocks takes the
   caller with it, violating the locking discipline above.
5. **`flush()` / `close()` semantics.** `:547-557` — `slog` has neither.

**Recommended Go shape** (INFERRED design, derived from the cited constraints):

```go
type Category uint8   // Call, Metadata, Network, Lifecycle, Pool, Locking, Performance
type Level    uint8   // Error, Warn, Info, Debug, Trace  — lower value = more severe
type Code     uint8   // 34 constants; Code.Category() must equal the event's Category
type State    uint8   // 9 constants
type Phase    uint8   // 12 constants
type Disposition uint8 // 4 constants

type Event struct {          // no pointers, no strings except CorrelationId
    SchemaVersion int        // always 1
    Sequence      uint64
    Timestamp     time.Time
    Category      Category
    Level         Level
    Code          Code
    CorrelationId string     // "" or a uuid.UUID v4 rendering
    State         State      // zero value = unset
    Phase         Phase
    Disposition   Disposition
    DurationMs    float64    // 0..86_400_000
    Count         int64      // >= 0
}
```

The struct-with-typed-enums shape is what mechanically enforces R1: there is no
field to put a secret in, which is exactly the property `Object.freeze` +
`INPUT_KEYS` buys in TypeScript. `String()` methods on each enum, a
`Code.Category()` method checked in the constructor, and a `Validate()` mirroring
`snapshotInput` (`:273-339`) complete the boundary. Keep `slog` as an *optional
adapter at the sink*, translating `Event` → `slog.Record`, never as the event
type itself. `encoding/json` on this struct with `omitempty` reproduces the JSONL
line at `:699`.

---

## Open questions for the porter

1. **Out-of-scope dependencies.** `MetadataRepositoryRuntime`,
   `MetadataRepositoryMode`, `MetadataLoadStrategy`, `MetadataAccessFailure`,
   `createMetadataCapabilityKey`, `createMetadataStructuralKey`
   (`src/metadata/repository-runtime.js`), `SessionContextRuntime`
   (`src/lifecycle/session-context-runtime.js`), `TransactionLeaseAdapter`
   (`src/lifecycle/transaction-runtime.js`), and `DirectCpicSession`
   (`src/client/direct-cpic-session.js`) are referenced throughout
   `src/destination/` but were not read. The exact shape of
   `MetadataCapabilityKey.id` (asserted distinct at
   `test/destination-configuration-generation.test.ts:65-68`) and of
   `MetadataStructuralKey` are unknown; both are needed before
   `src/destination/runtime.ts` can be ported. **This is the one blocking gap.**

2. **`Object.freeze` has no Go analogue.** 91 `Object.freeze` occurrences across
   the five source files (52 of them in `direct-destination-owner.ts` alone), and
   22 `Object.isFrozen(...)` assertions across the five test files
   (e.g. `test/structured-diagnostics.test.ts:93`,
   `test/destination-configuration-generation.test.ts:62-63`,
   `test/direct-destination-owner.test.ts:1255`). Value types plus unexported
   fields cover most of it, but the "caller cannot mutate what we already
   snapshotted" tests (`"snapshots every constructor option and external method
   exactly once"`, `test/connection-pool-runtime.test.ts:365`) have no Go
   analogue at all — Go has no property accessors to trap. Decide whether these
   drop out (like `src/protocol/bytes.ts` did, `../provenance.md:44`) or get a
   defensive-copy analogue.

3. **`AsyncLocalStorage` retirement-cycle detection.** Both
   `src/destination/configuration-generation.ts:96-149` and
   `src/destination/runtime.ts:97-218` use `AsyncLocalStorage` plus a
   `WeakMap<object, Set<object>>` wait-for graph to let a retirement hook
   re-enter `retire()` without deadlock. Four tests depend on it
   (`test/destination-configuration-generation.test.ts:400`, `:450`, `:518`,
   `:604`). Go has no ambient async context — `context.Context` must be threaded
   explicitly, and `WeakMap` has no equivalent. Is the reentrancy contract worth
   preserving, or should Go simply document that a retire hook must not call
   `Retire()` on its owner?

4. **`queueMicrotask` ordering.** The correctness of several sequences depends on
   microtask ordering (e.g. `test/runtime-diagnostics-integration.test.ts:26-33`
   crosses exactly two hops; `src/pool/connection-pool-runtime.ts:1320-1327`
   rearms a deadline from a microtask). Goroutines give no such ordering
   guarantee. Which of these need explicit sequencing in Go, and which were only
   a JS implementation detail?

5. **`monitor.leasesIssued` duplicates `lastLeaseGeneration`.** Both are
   `this.#lastLeaseGeneration` (`src/pool/connection-pool-runtime.ts:1182-1183`).
   Intentional compatibility alias, or should the Go monitor expose one field?

6. **`drain()` is a pure alias for `close()`** (`:1001-1003`). Same question.

7. **`#optimizedGenerationTokens` eviction is insertion-order, not LRU.**
   `#rememberOptimizedGeneration` deletes then re-inserts the refreshed key
   (`:2627`, `:2640`), so a *reload* refreshes recency but a cache *hit* does
   not. Is that intended, or an artefact of `Map` iteration order?

8. **Clock monotonicity is a hard error, not a clamp.** `#readClock` throws
   `"connection pool scheduler clock must be finite and monotonic"` on any
   regression (`:1195-1202`). Go's `time.Now()` monotonic reading makes this
   unreachable for the default scheduler but not for an injected one. Keep the
   check?

9. **`RfcDiagnosticState` overlaps `ConnectionPoolRuntimeState` but is not
   identical.** Pool states are `"open" | "retiring" | "closing" | "closed"`
   (`:9-13`); diagnostic states add `"connecting"`, `"retired"`, `"waiting"`,
   `"leased"`, `"idle"`, `"failed"` and omit `"retiring"`
   (`src/diagnostics/structured-diagnostics.ts:70-80`). The mapping is done
   ad hoc at each call site (e.g. `state: "closing"` for `pool.shutdown` at
   `:1031-1038` even though `#state` may be `"retiring"`). Should Go make this
   mapping explicit and total?

10. **The `locking.*` code family is declared but unused** in the ported scope
    (`src/diagnostics/structured-diagnostics.ts:61-64`). Port the constants
    anyway for wire compatibility with upstream JSONL, or drop them?

11. **`backendKey` canonicalization depends on JavaScript `JSON.stringify`
    string escaping** (`src/destination/configuration-generation.ts:194-201`).
    Go's `encoding/json` escapes `<`, `>`, `&` as `<` etc. by default,
    which would produce different hashes for identity components containing
    those characters. A Go port must either disable `SetEscapeHTML` or hand-roll
    the canonical form, and needs a cross-implementation test vector.
