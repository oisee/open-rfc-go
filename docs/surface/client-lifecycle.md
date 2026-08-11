# Surface inventory: src/client/ and src/lifecycle/

> Mechanical inventory of open-rfc @ commit 847036d, generated as porting input. Every claim cites path:line. See ../provenance.md.

Conventions used below:

- All `path:line` citations are relative to an `open-rfc` checkout at `847036d`.
- Signatures in tables are **verbatim tokens** with source line breaks collapsed to
  single spaces; nothing else is changed.
- Claim kinds are marked: unmarked rows are **code**; rows prefixed `COMMENT:` or
  `TEST-NAME:` quote a comment or a test title; rows prefixed `INFERRED:` are the
  author's reading, not a quotation.
- Scope read: `src/client/*.ts`, `src/lifecycle/*.ts`, and the eight named tests.
  Nothing else was opened.

---

## src/client/rfc-failure.ts

The failure model. 729 lines, no I/O, no Promises. This file is the single source of
truth for connection disposition and replay policy and must be ported first.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RfcFailureCategory` | enum (string) | `export enum RfcFailureCategory` | `src/client/rfc-failure.ts:9` |
| `RfcFailureOrigin` | enum (string) | `export enum RfcFailureOrigin` | `src/client/rfc-failure.ts:26` |
| `RfcOperationPhase` | enum (string) | `export enum RfcOperationPhase` | `src/client/rfc-failure.ts:38` |
| `RfcTransmissionState` | enum (string) | `export enum RfcTransmissionState` | `src/client/rfc-failure.ts:53` |
| `RfcConnectionDisposition` | enum (string) | `export enum RfcConnectionDisposition` | `src/client/rfc-failure.ts:61` |
| `RfcRecoveryAction` | enum (string) | `export enum RfcRecoveryAction` | `src/client/rfc-failure.ts:68` |
| `RfcReplayPolicy` | enum (string) | `export enum RfcReplayPolicy` | `src/client/rfc-failure.ts:73` |
| `RfcFailureGroup` | enum (numeric) | `export enum RfcFailureGroup` | `src/client/rfc-failure.ts:84` |
| `RfcFailureCode` | enum (numeric) | `export enum RfcFailureCode` | `src/client/rfc-failure.ts:93` |
| `RfcFailureCodeString` | type union | `export type RfcFailureCodeString =` | `src/client/rfc-failure.ts:114` |
| `RfcFailurePolicyContext` | interface | `export interface RfcFailurePolicyContext` | `src/client/rfc-failure.ts:134` |
| `RfcFailurePolicy` | interface | `export interface RfcFailurePolicy` | `src/client/rfc-failure.ts:143` |
| `RfcFailureAbapFacts` | interface | `export interface RfcFailureAbapFacts` | `src/client/rfc-failure.ts:152` |
| `RfcFailureDiagnostic` | interface | `export interface RfcFailureDiagnostic` | `src/client/rfc-failure.ts:168` |
| `RfcFailure` | interface | `export interface RfcFailure` | `src/client/rfc-failure.ts:184` |
| `CreateRfcFailureInput` | interface | `export interface CreateRfcFailureInput extends RfcFailurePolicyContext` | `src/client/rfc-failure.ts:206` |
| `CreateRemoteRfcFailureContext` | interface | `export interface CreateRemoteRfcFailureContext` | `src/client/rfc-failure.ts:215` |
| `resolveRfcFailurePolicy` | function | `export function resolveRfcFailurePolicy( context: RfcFailurePolicyContext, ): RfcFailurePolicy` | `src/client/rfc-failure.ts:359-361` |
| `rfcFailureDiagnostic` | function | `export function rfcFailureDiagnostic( failure: Pick<RfcFailure, …>, ): RfcFailureDiagnostic` (the `Pick` lists `schemaVersion, correlationId, reasonCode, category, origin, phase, transmission, disposition, recoveryAction, replayPolicy, group, code, codeString`) | `src/client/rfc-failure.ts:535-552` |
| `createRfcFailure` | function | `export function createRfcFailure(input: CreateRfcFailureInput): RfcFailure` | `src/client/rfc-failure.ts:570` |
| `createRemoteRfcFailure` | function | `export function createRemoteRfcFailure( envelope: RfcErrorEnvelope, context: CreateRemoteRfcFailureContext, ): RfcFailure` | `src/client/rfc-failure.ts:688-691` |
| `RfcCoreError` | class | `export class RfcCoreError extends Error` with `declare readonly failure: RfcFailure;` and `constructor(failure: RfcFailure)` | `src/client/rfc-failure.ts:709-712` |

### States and transitions

This file has no state machine; it has closed value domains. Every member below is
exhaustive and its exact string/ordinal is wire- and API-visible.

| Domain | Members (verbatim) | Citation |
|---|---|---|
| `RfcFailureCategory` | `InvalidState = "invalidState"`, `InvalidParameter = "invalidParameter"`, `Conversion = "conversion"`, `Serialization = "serialization"`, `Unsupported = "unsupported"`, `Resource = "resource"`, `Communication = "communication"`, `Logon = "logon"`, `AbapRuntime = "abapRuntime"`, `AbapException = "abapException"`, `AbapMessage = "abapMessage"`, `Canceled = "canceled"`, `Timeout = "timeout"`, `MalformedProtocol = "malformedProtocol"` | `src/client/rfc-failure.ts:10-23` |
| `RfcFailureOrigin` | `Api = "api"`, `Codec = "codec"`, `Ni = "ni"`, `Gateway = "gateway"`, `Appc = "appc"`, `Cpic = "cpic"`, `Sap = "sap"`, `Metadata = "metadata"`, `Pool = "pool"` | `src/client/rfc-failure.ts:27-35` |
| `RfcOperationPhase` | `Connect = "connect"`, `GatewaySetup = "gatewaySetup"`, `AppcSetup = "appcSetup"`, `Logon = "logon"`, `Metadata = "metadata"`, `Encode = "encode"`, `Send = "send"`, `Receive = "receive"`, `EnvelopeDecode = "envelopeDecode"`, `ValueDecode = "valueDecode"`, `Close = "close"`, `Replacement = "replacement"` | `src/client/rfc-failure.ts:39-50` |
| `RfcTransmissionState` | `NotStarted = "notStarted"`, `Partial = "partial"`, `Complete = "complete"`, `Unknown = "unknown"` | `src/client/rfc-failure.ts:54-57` |
| `RfcConnectionDisposition` | `Reusable = "reusable"`, `Close = "close"`, `UnknownClose = "unknownClose"` | `src/client/rfc-failure.ts:62-64` |
| `RfcRecoveryAction` | `None = "none"`, `Replace = "replace"` | `src/client/rfc-failure.ts:69-70` |
| `RfcReplayPolicy` | `Never = "never"` — the enum has exactly one member | `src/client/rfc-failure.ts:73-75` |
| `RfcFailureGroup` | `AbapApplicationFailure = 1`, `AbapRuntimeFailure = 2`, `LogonFailure = 3`, `CommunicationFailure = 4`, `ExternalRuntimeFailure = 5` | `src/client/rfc-failure.ts:85-89` |
| `RfcFailureCode` | `CommunicationFailure = 1`, `LogonFailure = 2`, `AbapRuntimeFailure = 3`, `AbapMessage = 4`, `AbapException = 5`, `Closed = 6`, `Canceled = 7`, `Timeout = 8`, `MemoryInsufficient = 9`, `VersionMismatch = 10`, `InvalidProtocol = 11`, `SerializationFailure = 12`, `NotSupported = 18`, `IllegalState = 19`, `InvalidParameter = 20`, `CodepageConversionFailure = 21`, `ConversionFailure = 22`, `UnknownError = 28` | `src/client/rfc-failure.ts:94-111` |

**The disposition decision function** (`resolveRfcFailurePolicy`) is the core rule set.
It is a total function of `(category, origin, phase, transmission, establishedSession)`:

| Category group | Disposition rule | Citation |
|---|---|---|
| `AbapException` | `Reusable` **iff** `origin === Sap && phase === EnvelopeDecode && transmission === Complete && establishedSession`; otherwise `UnknownClose` | `src/client/rfc-failure.ts:366-374` |
| `MalformedProtocol` | always `UnknownClose` | `src/client/rfc-failure.ts:375-377` |
| `InvalidState`, `InvalidParameter`, `Conversion`, `Serialization`, `Unsupported`, `Resource` | `Reusable` **iff** `transmission === NotStarted` **and** (`origin ∈ {Api, Codec, Pool}` or `phase ∈ LOCAL_PRE_SEND_PHASES`); otherwise `UnknownClose`. `LOCAL_PRE_SEND_PHASES = new Set([RfcOperationPhase.Metadata, RfcOperationPhase.Encode])` | `src/client/rfc-failure.ts:378-392`, `src/client/rfc-failure.ts:304-307` |
| `Communication`, `Logon`, `AbapRuntime`, `AbapMessage`, `Canceled`, `Timeout` | always `Close` | `src/client/rfc-failure.ts:393-400` |
| recovery action (all categories) | `return establishedSession && disposition !== RfcConnectionDisposition.Reusable ? RfcRecoveryAction.Replace : RfcRecoveryAction.None;` | `src/client/rfc-failure.ts:346-353`, applied at `:406-409` |
| replay policy (all categories) | `replayPolicy: RfcReplayPolicy.Never` — hardcoded, never a parameter | `src/client/rfc-failure.ts:410` |

`CODE_POLICY` maps each category to its `(group, code, codeString)` triple; it is a
frozen total record over `RfcFailureCategory` (`src/client/rfc-failure.ts:230-302`).

### Errors and error codes

| Code/message (verbatim) | Trigger | Recoverable per source? | Citation |
|---|---|---|---|
| `"RFC failure policy context must be an object"` (`TypeError`) | non-object policy context | n/a — programmer error before any I/O | `src/client/rfc-failure.ts:325-327` |
| `"category must be a supported RfcFailureCategory"` | unknown category | n/a | `src/client/rfc-failure.ts:329-331` |
| `"origin must be a supported RfcFailureOrigin"` | unknown origin | n/a | `src/client/rfc-failure.ts:332-334` |
| `"phase must be a supported RfcOperationPhase"` | unknown phase | n/a | `src/client/rfc-failure.ts:335-337` |
| `"transmission must be a supported RfcTransmissionState"` | unknown transmission | n/a | `src/client/rfc-failure.ts:338-340` |
| `"establishedSession must be a boolean"` | non-boolean flag | n/a | `src/client/rfc-failure.ts:341-343` |
| `` `${field} must contain 1..${maximumLength} safe identifier characters` `` (`RangeError`) | `correlationId` (max 128) or `reasonCode` (max 128) failing `/^[A-Za-z0-9_.:-]+$/u` | n/a | `src/client/rfc-failure.ts:414-429`, called at `:574-575` |
| `"key must be a string"` / `"message must be a string"` | wrong type | n/a | `src/client/rfc-failure.ts:576-581` |
| `"ABAP fact provenance must be an array"` | non-array provenance | n/a | `src/client/rfc-failure.ts:434-436` |
| `` `ABAP fact provenance entry ${index} has an invalid tag` `` | tag outside `0..0xffff` or not a safe integer | n/a | `src/client/rfc-failure.ts:443-445` |
| `` `ABAP fact provenance entry ${index} has a non-increasing ordinal` `` | ordinal `<= previousOrdinal` | n/a | `src/client/rfc-failure.ts:446-454` |
| `` `ABAP fact provenance entry ${index} has an invalid byteLength` `` | byteLength outside `0..0x7fff_ffff` | n/a | `src/client/rfc-failure.ts:455-463` |
| `` `ABAP fact ${field} must be a string` `` | any of 12 named ABAP fact fields non-string | n/a | `src/client/rfc-failure.ts:500-517` |
| `"a successful RFC envelope cannot create a failure"` | `createRemoteRfcFailure` with `envelope.outcome === "success"` | n/a | `src/client/rfc-failure.ts:648` |
| `` `category ${category} is not a remote ABAP failure` `` | `remoteReasonCode` with non-remote category | n/a | `src/client/rfc-failure.ts:661` |
| `RfcCoreError.message` = `` `${failure.codeString}: ${failure.reasonCode} [${failure.correlationId}]` `` | wrapping any `RfcFailure` for throwing | carries `failure.disposition` / `failure.recoveryAction` | `src/client/rfc-failure.ts:713-715` |

Remote reason codes emitted by `createRemoteRfcFailure`: `"RFC_REMOTE_DECLARED_EXCEPTION"`
(AbapException), `"RFC_REMOTE_ABAP_RUNTIME"` (AbapRuntime), `"RFC_REMOTE_ABAP_MESSAGE"`
(AbapMessage) — `src/client/rfc-failure.ts:652-663`.

Remote `key` selection: AbapException → `facts.exceptionKey`; AbapRuntime →
`facts.runtimeId || facts.t100Text || facts.plainText || "RFC_ABAP_RUNTIME_FAILURE"`;
otherwise `facts.t100Text || facts.plainText || "RFC_ABAP_MESSAGE"` —
`src/client/rfc-failure.ts:665-674`.
Remote `message` selection: `facts.plainText` if non-empty, else `facts.t100Text`, else
`facts.runtimeId` only for AbapRuntime, else `""` — `src/client/rfc-failure.ts:676-686`.
Default `key`/`message` when not supplied: `policy.codeString` —
`src/client/rfc-failure.ts:601,607`.

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Disposition of the physical connection generation which saw the failure. */` | `src/client/rfc-failure.ts:60` |
| `/** Client-level action after the old generation's disposition is applied. */` | `src/client/rfc-failure.ts:67` |
| `/** True only after a fully authenticated physical generation exists. */` | `src/client/rfc-failure.ts:139` |
| `* Authoritative beta policy. Call sites cannot weaken connection disposition` / `* or grant replay permission.` | `src/client/rfc-failure.ts:356-357` |
| `/** JSON serialization is deliberately restricted to the safe diagnostic. */` | `src/client/rfc-failure.ts:202` |
| `/** Internal thrown wrapper; its sensitive failure record is non-enumerable. */` | `src/client/rfc-failure.ts:708` |
| `* SAP RFC error groups, kept language-neutral in the core.` … `* Only the groups `CODE_POLICY` can produce are declared, and each keeps its` / `* own explicit ordinal, so the numbering is stable and independent of the` / `* member list.` | `src/client/rfc-failure.ts:77-83` |

### Behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| Category→(group, code) mapping is exactly the 14-row table `[InvalidState,[5,19]] … [MalformedProtocol,[5,11]]`, and every policy has `replayPolicy === RfcReplayPolicy.Never` | `"maps every failure category to its language-neutral SAP group and code"` | `test/rfc-failure.test.ts:91-126` |
| The disposition rule holds for the full cross product of category × origin × phase × transmission × establishedSession, and each returned policy is frozen | `"applies the authoritative policy across every category/origin/phase/transmission state"` | `test/rfc-failure.test.ts:128-157` |
| `Communication` is `Close` for **every** transmission state (including `NotStarted`); a local `Conversion`/`Encode`/`NotStarted` failure is `Reusable`; the same category at `ValueDecode`/`Complete` is `UnknownClose`; a pre-auth `Connect` communication failure is `Close` + `RecoveryAction.None` | `"keeps category, transmission, old-generation disposition, and recovery independent"` | `test/rfc-failure.test.ts:159-205` |
| A declared exception envelope yields `Reusable`/`None` with `message: ""`; runtime and MESSAGE envelopes yield `Close`/`Replace` | `"declared exceptions are reusable while fatal remote outcomes replace established sessions"` | `test/rfc-failure.test.ts:207-275` |
| A SYSTEM_FAILURE envelope keeps all 12 ABAP facts plus per-field `provenance` `{tag, ordinal, byteLength}`, and `JSON.stringify(failure)` does not contain the call stack | `"retains a complete SYSTEM_FAILURE fact graph with terminal receive policy"` | `test/rfc-failure.test.ts:277-352` |
| `createRemoteRfcFailure` throws on a successful envelope | `"does not create a failure from a successful envelope"` | `test/rfc-failure.test.ts:354-363` |
| Failure and its `abap`/`provenance` are frozen; `cause`, `toJSON`, `key`, `message`, `abap` are non-enumerable; `JSON.stringify(failure)` equals `rfcFailureDiagnostic(failure)` and leaks none of key/message/variables/stack/cause | `"creates immutable failures with safe diagnostic-only JSON"` | `test/rfc-failure.test.ts:365-422` |
| `RfcCoreError.failure` is non-enumerable, `error.cause === undefined`, `JSON.stringify(error) === "{}"`, and neither message nor stack leaks the remote text | `"wraps core failures without making the sensitive record enumerable"` | `test/rfc-failure.test.ts:424-452` |
| Caller-supplied `disposition: Reusable` / `recoveryAction: None` on a Communication failure are **ignored**; the result is `Close`/`Replace` | `"rejects unsafe diagnostic identifiers and ignores attempted policy weakening"` | `test/rfc-failure.test.ts:454-484` |
| Every policy context field is validated, including `null` context | `"validates every runtime policy context field"` | `test/rfc-failure.test.ts:486-521` |
| Changing any one of origin/phase/transmission/establishedSession away from the exact reusable tuple makes a declared exception `UnknownClose` | `"reuses a declared exception only for a complete authenticated remote envelope"` | `test/rfc-failure.test.ts:523-556` |
| ABAP facts and provenance are deep-copied; later mutation of the caller's array/object does not change the failure | `"validates and snapshots ABAP provenance supplied by adapters"` | `test/rfc-failure.test.ts:558-619` |

---

## src/client/rfc-errors.ts

399 lines. **`docs/provenance.md` already records this file as deliberately not
ported** (it is one of the two files containing third-party SAP SE / node-rfc code);
see the "Not ported, deliberately" table in `../provenance.md`. Inventoried here only
so the decision can be re-checked.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RFCErrorCode` | enum (numeric) | `export enum RFCErrorCode` | `src/client/rfc-errors.ts:169` |
| `RFCErrorProperties` | interface | `export interface RFCErrorProperties` | `src/client/rfc-errors.ts:206` |
| `RFCError` | class | `export class RFCError extends Error` with `constructor(message: string, properties: RFCErrorProperties)`, `static isRFCError(value: unknown): value is RFCError`, `static isABAPError(value: unknown): value is ABAPError` | `src/client/rfc-errors.ts:215-239` |
| `ABAPError` | class | `export class ABAPError extends RFCError` with `constructor(source: LegacyAbapExceptionProperties | RfcFailure)` | `src/client/rfc-errors.ts:289-299` |
| `rfcFailureToPublicError` | function | `export function rfcFailureToPublicError(failure: RfcFailure): RFCError` | `src/client/rfc-errors.ts:356` |
| `NodeRfcError` | class | `export class NodeRfcError extends Error` with `constructor(message: string)` | `src/client/rfc-errors.ts:394-395` |

### States and transitions

None. `RFCErrorCode` is a 34-member ordinal enum `RFC_OK = 0 … RFC_LOCKING_FAILURE = 33`
(`src/client/rfc-errors.ts:169-204`). Projection routing: categories
`AbapException`, `AbapRuntime`, `AbapMessage` → `ABAPError`; everything else →
`RFCError` named `"RfcLibError"` (`src/client/rfc-errors.ts:356-370`).

### Errors and error codes

| Code/message (verbatim) | Trigger | Recoverable per source? | Citation |
|---|---|---|---|
| `ABAPError` display for a declared exception: `` `${prefix}Number:${messageNumber || "000"}` `` optionally followed by `` ` ${detail}` ``, with `prefix` = `"ID:<class> Type:<type> "` or a single space when both are absent | `RfcFailureCategory.AbapException` projection | inherits `failure.disposition` (not represented on the public error) | `src/client/rfc-errors.ts:261-276`, `:302-309` |
| `RFCError.name = "ABAPError"` / `"RfcLibError"` | projection | — | `src/client/rfc-errors.ts:311`, `:365` |
| `NodeRfcError.name = "nodeRfcError"` | constructed by compat pool code only | — | `src/client/rfc-errors.ts:397` |

Note: `group`, `code`, `codeString`, `key` are copied straight from the `RfcFailure`
(`src/client/rfc-errors.ts:310-316`, `:364-370`) — the public error carries **no**
disposition, recovery, or replay field. A Go port that keeps this façade would be
dropping the recovery signal at the API boundary.

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Copy only the exact frozen, payload-free assertion installed by CPIC. */` | `src/client/rfc-errors.ts:70` |
| `// Archived node-rfc preserves a single leading separator when a declared` / `// exception has neither a message class nor a message type.` | `src/client/rfc-errors.ts:272-273` |
| `/** Loader-independent recognition across the ESM and CommonJS builds. */` | `src/client/rfc-errors.ts:231` |
| `// This verifier is retrieved from the exact package instance that defined it.` / `// Its WeakMap membership check keeps recognition scoped to one module copy` / `// without adding a declared static member or root export.` | `src/client/rfc-errors.ts:247-249` |
| `* Modified and adapted by open-rfc contributors. Every name and ordinal below` / `* is pinned to the RFC_RC enum in the archived Apache-2.0 node-rfc v3.3.1 source` | `src/client/rfc-errors.ts:158-159` |
| `/** node-rfc-compatible shape for all remote ABAP failures. */` | `src/client/rfc-errors.ts:288` |

### Behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| Declared exception projects to `{name:"ABAPError", group:1, code:5, codeString:"RFC_ABAP_EXCEPTION", key:"RAISE_EXCEPTION", message:"ID:SR Type:E Number:006 Method = 1"}` and the four `abapMsgV*` fields | `"exposes a node-rfc-compatible declared ABAP exception shape"` | `test/errors.test.ts:18-65` |
| With neither class nor type, `error.message === " Number:000"` (leading single space) and `abapMsgNumber === "000"` | `"preserves the archived empty-message declared-exception display shape"` | `test/errors.test.ts:67-78` |
| Public errors carry no `failure` property, no enumerable `cause`, and neither `JSON.stringify` nor `inspect` reveals the remote call stack or the low-level cause | `"projects complete core ABAP failures without exposing remote stack or cause"` | `test/errors.test.ts:88-171` |
| MESSAGE A/E/X project as `group:2, code:4, codeString:"RFC_ABAP_MESSAGE"`, `key` = T100 template, `message` = rendered text | `"projects MESSAGE A, E, and X with all public ABAP fields"` | `test/errors.test.ts:173-231` |

---

## src/client/classic-invocation.ts

2306 lines. Pure, synchronous encode/decode of a classic CPIC "CUT" function call from
`RFC_GET_FUNCTION_INTERFACE`-shaped metadata. No Promises, no signals, no sockets —
this is a near-mechanical translation target, not a redesign target.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `ClassicRfcInput` | type | `export type ClassicRfcInput = Readonly<Record<string, unknown>>;` | `src/client/classic-invocation.ts:97` |
| `ClassicRfcOutput` | type | `export type ClassicRfcOutput = Readonly<Record<string, unknown>>;` | `src/client/classic-invocation.ts:98` |
| `RfcStructureRepository` | type | `export type RfcStructureRepository = ReadonlyMap<string, RfcStructureDefinition>;` | `src/client/classic-invocation.ts:99` |
| `ClassicRfcInvocationOptions` | interface | fields `notRequested?`, `activated?`, `deactivated?`, `maxApplicationDataLength?`, `int8Mode?`, `bcd?` | `src/client/classic-invocation.ts:121-133` |
| `CapturedClassicRfcInvocation` | interface | `readonly input: ClassicRfcInput; readonly options: ClassicRfcInvocationOptions;` | `src/client/classic-invocation.ts:135-138` |
| `ClassicInvocationMetadataNeeds` | interface | `readonly containerParameters: ReadonlySet<string>; readonly optionalRecursive: boolean; readonly requiredRecursive: boolean;` | `src/client/classic-invocation.ts:141-148` |
| `classicInvocationRecursiveMetadataParameters` | function | `export function classicInvocationRecursiveMetadataParameters( metadata: RfcFunctionInterface, input: ClassicRfcInput, options: ClassicRfcInvocationOptions = {}, ): readonly RfcFunintParameter[]` | `src/client/classic-invocation.ts:533-537` |
| `captureClassicRfcInvocation` | function | `export function captureClassicRfcInvocation( metadata: RfcFunctionInterface, input: ClassicRfcInput, options: ClassicRfcInvocationOptions = {}, ): CapturedClassicRfcInvocation` | `src/client/classic-invocation.ts:1145-1149` |
| `classifyClassicInvocationMetadataNeeds` | function | `export function classifyClassicInvocationMetadataNeeds( metadata: RfcFunctionInterface, inputValue: ClassicRfcInput, options: ClassicRfcInvocationOptions, ): ClassicInvocationMetadataNeeds` | `src/client/classic-invocation.ts:1234-1238` |
| `buildClassicRfcInvocationRequest` | function | `export function buildClassicRfcInvocationRequest( metadata: RfcFunctionInterface, input: ClassicRfcInput, structures: RfcStructureRepository = new Map(), options: ClassicRfcInvocationOptions = {}, recursiveMetadata?: RecursiveMetadataGraph, ): Buffer` | `src/client/classic-invocation.ts:1687-1693` |
| `decodeClassicRfcInvocationResult` | function | `export function decodeClassicRfcInvocationResult( metadata: RfcFunctionInterface, fields: readonly CpicField[], structures: RfcStructureRepository = new Map(), options: ClassicRfcInvocationOptions = {}, recursiveMetadata?: RecursiveMetadataGraph, ): ClassicRfcOutput` | `src/client/classic-invocation.ts:1977-1983` |
| `decodeOwnedClassicRfcInvocationResult` | function | `export function decodeOwnedClassicRfcInvocationResult( metadata: RfcFunctionInterface, fields: readonly CpicField[], structures: RfcStructureRepository = new Map(), options: ClassicRfcInvocationOptions = {}, recursiveMetadata?: RecursiveMetadataGraph, ): ClassicRfcOutput` | `src/client/classic-invocation.ts:1995-2001` |

### States and transitions

No runtime state machine. What the code enforces instead is a **parameter activation
lattice**, which is protocol semantics and must be preserved exactly:

| Rule | Citation |
|---|---|
| `deactivated` wins over everything: `if (deactivated.has(parameter.parameterName)) return Object.freeze({ present: false });` | `src/client/classic-invocation.ts:234-236` |
| an explicit input value makes the parameter active with that value | `src/client/classic-invocation.ts:237-242` |
| an **optional** parameter with no value and not in `activated` stays inactive | `src/client/classic-invocation.ts:246-248` |
| a **mandatory** parameter with no value becomes active at its ABAP initial value (`Object.freeze([])` for class `T`) | `src/client/classic-invocation.ts:249-254` |
| output requested: class `E` always; classes `C` and `T` only when their input side is active; never when deactivated | `src/client/classic-invocation.ts:257-277` |
| `deactivated` is the **union** of `notRequested` and `deactivated` | `src/client/classic-invocation.ts:565-568`, `:1727-1730`, `:2049-2052`, union helper at `:1132-1137` |
| `captureClassicRfcInvocation` adds every supplied input name to `activated` | `src/client/classic-invocation.ts:1206` |
| class `T` + `exid "u"` stays on the binary-table path and is excluded from recursive-xRFC selection | `src/client/classic-invocation.ts:517-518` |
| classic TABLES parameters stay on the binary table path even when optimized metadata exists (`if (parameter.parameterClass === "T") continue;`) | `src/client/classic-invocation.ts:1283-1285` |

Initial values by `exid` (`initialScalarValue`): `"C"|"N"|"g"` → `""`; `"D"` →
`"00000000"`; `"T"` → `"000000"`; `"X"|"y"` → `Buffer.alloc(0)`; `"F"|"I"|"s"|"b"` →
`0`; `"8"` → `classicInt8InitialValue(int8Mode)`; `"P"` → `""`; `"a"|"e"` → `"0"`;
`"u"|"v"` → `Object.freeze({})`; `"h"` → `Object.freeze([])` —
`src/client/classic-invocation.ts:184-225`.

Serializer selection for deep parameters: the strict `recursive-classic-xrfc` resolver
is tried first and the broader `recursive-xrfc` codec is admitted only if **both** its
own resolver and its complete validator succeed; otherwise the strict error is
rethrown — `src/client/classic-invocation.ts:376-421`.

### Errors and error codes

All are plain `Error`/`TypeError`/`RangeError` with no code field. Selected verbatim
messages (template literals shown with their placeholders):

| Code/message (verbatim) | Trigger | Recoverable per source? | Citation |
|---|---|---|---|
| `` `${parameter.parameterName} classic RFC type ${parameter.exid} is not implemented` `` | unknown `exid` in initial-value / encode / decode / byte-length paths | pre-wire | `:221-223`, `:692-695`, `:1519-1522`, `:1639-1642` |
| `` `metadata parameter count exceeds ${DEFAULT_MAX_CPIC_FIELD_COUNT}` `` | metadata too large | pre-wire | `src/client/classic-invocation.ts:311-315` |
| `` `${required.parameterName} requires recursive xRFC metadata` `` | active `exid "v"` with no graph | pre-wire | `src/client/classic-invocation.ts:361-367` |
| `` `${parameter.parameterName} has unsupported recursive parameter class ${parameter.parameterClass}` `` | recursive parameter not in `/^[IECT]$/u` | pre-wire | `:380-384`, `:429-433` |
| `` `${parameter.parameterName} requires recursive metadata for its deep structure` `` | `exid "v"` reaching `classicXrfcKind` | pre-wire | `src/client/classic-invocation.ts:1318-1322` |
| `` `${parameter.parameterName} classic TABLES rows cannot contain STRING/XSTRING fields` `` | dynamic fields in a TABLES row structure | pre-wire | `:707-711`, `:1326-1330` |
| `` `${parameter.parameterName} has a deep scalar table line which requires an unsupported negotiated serializer` `` | `exid "g"`/`"y"` as a table line | pre-wire | `src/client/classic-invocation.ts:714-719` |
| `` `classic RFC request field count exceeds ${DEFAULT_MAX_CPIC_FIELD_COUNT}` `` | preflight bound | pre-wire | `src/client/classic-invocation.ts:822-827` |
| `` `classic RFC request field chain exceeds ${DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH} bytes` `` | preflight bound | pre-wire | `src/client/classic-invocation.ts:828-832` |
| `` `classic RFC request application length exceeds configured limit ${maximum}` `` | `maxApplicationDataLength` bound, counting `CPIC_CUT_FIXED_APPLICATION_BYTES = 6` | pre-wire | `:833-840`, constant at `:150` |
| `` `classic RFC field length exceeds ${DEFAULT_MAX_CPIC_FIELD_LENGTH} bytes` `` | single field too long | pre-wire | `src/client/classic-invocation.ts:843-851` |
| `"maxApplicationDataLength must be an integer in 0..2147483647"` | out-of-range option | pre-wire | `src/client/classic-invocation.ts:812-819` |
| `` `duplicate xRFC input parameter ${parameter.parameterName}` `` | two encodings for one name | pre-wire | `src/client/classic-invocation.ts:922-924` |
| `` `notRequested contains unknown parameter ${name}` `` / `` `${label} contains unknown parameter ${name}` `` | name not in metadata | pre-wire | `:1075-1077`, `:1092-1095` |
| `` `${label} entry count exceeds metadata parameter count ${metadataParameterCount}` `` | oversized state set | pre-wire | `src/client/classic-invocation.ts:1120-1125` |
| `` `unknown parameter ${name}` `` | input name not in metadata | pre-wire | `:1193`, `:1740` |
| `` `export parameter ${name} cannot be supplied as input` `` | class `E` supplied as input | pre-wire | `:1195`, `:1742` |
| `` `input parameter count exceeds metadata parameter count ${parameterCount}` `` | too many input keys | pre-wire | `src/client/classic-invocation.ts:1186-1190` |
| `` `recursive metadata identity does not match function ${metadata.name}` `` | graph identity mismatch | pre-wire | `src/client/classic-invocation.ts:1335-1347` |
| `` `${parameter.parameterName} requires unresolved structure ${parameter.tableName}` `` | missing structure definition | pre-wire | `src/client/classic-invocation.ts:1296-1306` |
| `` `${parameter.parameterName} encoded length changed after request preflight` `` (`RangeError`) | value changed between preflight and encode | pre-wire; this is the anti-TOCTOU guard | `src/client/classic-invocation.ts:1868-1872` |
| `` `${parameter.parameterName}[${index}] encoded row length changed after request preflight` `` | same, per row | pre-wire | `src/client/classic-invocation.ts:1945-1949` |
| `` `classic RFC response contains duplicate parameter ${name}` `` (three call sites: scalars, tables, xRFC) | duplicate reply parameter | response-side; raised inside decode | `:2133`, `:2157`, `:2245` |
| `` `classic RFC response returned unknown parameter ${scalar.name}` `` / `…unknown table ${table.name}` / `…unknown xRFC parameter ${name}` | reply names an unknown parameter | response-side | `:2139`, `:2163`, `:2251` |
| `` `classic RFC response returned non-output parameter ${scalar.name}` `` / `…non-table parameter…` / `…non-output xRFC parameter…` | direction mismatch | response-side | `:2142-2144`, `:2166`, `:2258` |
| `` `classic RFC response returned a binary table for recursive parameter ${table.name}` `` | serializer mismatch | response-side | `:2170-2172` |
| `` `${table.name} declared row width ${…} does not match metadata width ${…}` `` (+ optional `` ` or exact field width ${…}` ``) | table geometry mismatch | response-side | `:2193-2199` |
| `` `${table.name} row 0 width ${…} does not match metadata width ${…}` `` | first-row geometry mismatch | response-side | `:2207-2213` |
| `` `${table.name} row ${index} width ${…} does not match first row width ${…}` `` | inconsistent rows | response-side | `:2218-2221` |
| `` `classic RFC response returned xRFC XML for non-deep parameter ${name}` `` | XML for a flat parameter | response-side | `:2266-2268` |
| `` `classic RFC response lacks requested output ${parameter.parameterName}` `` | requested output missing from reply | response-side | `:2300-2302` |
| `` `${parameter.parameterName} STRING lacks one trailing NUL terminator` `` | `exid "g"` decode with wrong terminator or interior NUL | response-side | `:1614-1623` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `// NW RFC function containers start mandatory input directions active at` / `// their ABAP initial value. Optional inputs remain inactive until a value is` / `// supplied or the caller explicitly activates them.` | `src/client/classic-invocation.ts:243-245` |
| `// Preserve the independently qualified strict codec as the authoritative` / `// path for every graph shape it supports. The broader graph codec is used` / `// only after it explicitly resolves and validates a shape outside that` / `// strict subset.` | `src/client/classic-invocation.ts:375-378` |
| `// The broader codec may extend the strict subset, but it never gets to` / `// reinterpret an invalid graph: fallback is admitted only after its` / `// own resolver and complete validator both succeed.` | `src/client/classic-invocation.ts:403-405` |
| `// Classic TABLES parameters with u rows remain on the binary table path.` | `src/client/classic-invocation.ts:517` |
| `* Select active deep parameters that need optimized recursive metadata.` / `* Excluded parameters must not create an authorization dependency.` | `src/client/classic-invocation.ts:530-531` |
| `* Some SAP kernels omit unused trailing alignment bytes from binary TABLES` / `* reply rows even though DDIC reports the complete in-memory structure size.` / `* Admit only the exact end of the final validated field; this cannot hide a` / `* truncated field or an interior geometry mismatch.` | `src/client/classic-invocation.ts:728-733` |
| `* Capture caller-owned input and activation options before an exchange. Direct` / `* sessions use this one snapshot for request encoding and response validation,` / `* so an accessor or Proxy cannot change activation between those phases.` | `src/client/classic-invocation.ts:1140-1143` |
| `* Classify repository work only after caller activation state has been` / `* captured. Inactive optional inputs and suppressed deep outputs must not` / `* trigger RFC_METADATA_GET. A suppressed classic `u` output still needs its` / `* flat structure descriptor to preserve the established metadata-shaped ABAP` / `* initial value.` | `src/client/classic-invocation.ts:1228-1233` |
| `// Mature SDKs always keep classic TABLES parameters on the binary table` / `// path, even when optimized metadata is available.` | `src/client/classic-invocation.ts:1283-1284` |
| `// RFCTYPE_TABLE parameters use an indirect row descriptor even though` / `// RFC_GET_FUNCTION_INTERFACE exposes them in I/E/C directions.` | `src/client/classic-invocation.ts:1314-1315` |
| `// Grouping either detached this value for a public call or transferred a` / `// session-owned CPIC buffer into the internal result. Returning RAW bytes can` / `// therefore transfer that buffer without another full-size copy.` | `src/client/classic-invocation.ts:1533-1535` |
| `// RFCTYPE_TABLE (EXID h) is the modern table-container descriptor used for` / `// parameters declared with TABLE OF rather than classic TABLES. The classic` / `// CUT serializer cannot activate it, but an explicitly inactive output has` / `// the same observable initial value as every other table: no rows.` | `src/client/classic-invocation.ts:1652-1656` |
| `// Resolving the shape here rejects unsupported deep output metadata` / `// before a request can be sent.` | `src/client/classic-invocation.ts:945-946` |
| `case "g": { // ABAP STRING: UTF-8 with one trailing NUL in classic CUT` | `src/client/classic-invocation.ts:1493` |
| `/** Consume CPIC-session-owned reply fields without a second full wire copy. @internal */` | `src/client/classic-invocation.ts:1994` |

### Behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| An I/E `RFCTYPE_TABLE` deep call emits `RequestedOutput` for `["EXPORT_TAB","RESPTEXT"]`, no `ParameterName` for the deep input, and exactly the tag sequence `XRfcParameter, XRfcData, XRfcParameter` carrying the exact XML `"<IMPORT_TAB><item>…</IMPORT_TAB>"` | `"builds the real I/E RFCTYPE_TABLE STFC_DEEP_TABLE request shape"` | `test/classic-xrfc-invocation.test.ts:127-154` |
| The table type (`STFCCPLXT_T`), not its line type (`STFCCPLXT`), is the structure-repository key; supplying only the line type throws `/requires unresolved structure STFCCPLXT_T/` | `"resolves table type to its distinct recursive line definition"` | `test/classic-xrfc-invocation.test.ts:156-168` |
| A deep table split across two `XRfcData` chunks decodes into one array, alongside a flat scalar output in the same reply | `"decodes the captured two-chunk deep table beside classic output"` | `test/classic-xrfc-invocation.test.ts:170-186` |
| A mandatory deep input with no caller value is sent as the explicit empty document `"<IMPORT_TAB></IMPORT_TAB>"` | `"sends an explicit empty xRFC table for mandatory initial input"` | `test/classic-xrfc-invocation.test.ts:188-201` |
| Deactivating a deep input emits no `XRfcData`, and deactivating a deep output still yields its initial value `[]` in the decoded result | `"deactivation suppresses deep input and returns initial deep output"` | `test/classic-xrfc-invocation.test.ts:203-229` |
| A caller accessor on a row field is read **exactly once** (`assert.equal(rowReads, 1)`), later mutation of the source bytes does not change the request, and `maxApplicationDataLength` is enforced against the encoded size | `"keeps one preflight XML snapshot and enforces application bounds"` | `test/classic-xrfc-invocation.test.ts:231-268` |
| Multiple deep parameters each get their own `XRfcParameter … XRfcData … XRfcParameter` envelope, in metadata order | `"keeps multiple deep parameters as independent invocation envelopes"` | `test/classic-xrfc-invocation.test.ts:270-317` |
| Unknown / duplicate / non-deep / missing deep outputs each throw a distinct message | `"rejects unknown, duplicate, mismatched, and missing deep outputs"` | `test/classic-xrfc-invocation.test.ts:319-371` |

---

## src/client/direct-cpic-session.ts

1837 lines. One allocated CPIC conversation over one NI transport. This is the file
where wire semantics and Promise plumbing are entangled; the split is made explicit in
the last two sections of this document.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `DirectCpicTransportFactory` | type | `export type DirectCpicTransportFactory = ( options: NiSocketConnectOptions, signal?: AbortSignal, ) => NiSocketTransport | PromiseLike<NiSocketTransport>;` | `src/client/direct-cpic-session.ts:124-127` |
| `DirectCpicSessionOptions` | interface | fields `host`, `port`, `applicationServerHost?`, `applicationServerService`, `programName?`, `localAddress?`, `connectTimeoutMs?`, `operationTimeoutMs?`, `signal?`, `transportFactory?`, `cpicStreaming?: "disabled" | "enabled"`, `recursiveSerializerDecisionProvider?` | `src/client/direct-cpic-session.ts:129-158` |
| `DirectCpicPreWireError` | class | `export class DirectCpicPreWireError extends Error` with `override readonly cause: unknown;` and `constructor(cause: unknown)` | `src/client/direct-cpic-session.ts:164-177` |
| `DirectCpicSessionInfo` | interface | `localAddress`, `peerCodePage`, `peerAcceptInfo`, `generationHandle`, `connectionIndex`, `selectedApplicationServerHost?`, `selectedGatewayHost?`, `selectedSystemNumber?` | `src/client/direct-cpic-session.ts:179-193` |
| `DirectCpicLogonOptions` | interface | `client`, `user`, `password`, `language?`, `partnerHostName?`, `kernelRelease?` | `src/client/direct-cpic-session.ts:231-238` |
| `DirectCpicLogonResult` | interface | `readonly negotiatedProtocolVersion: number; readonly responseFieldCount: number;` | `src/client/direct-cpic-session.ts:240-243` |
| `DirectCpicPingResult` | interface | `readonly responseFieldCount: number;` | `src/client/direct-cpic-session.ts:245-247` |
| `CpicLogonError` | class | `export class CpicLogonError extends Error` with `readonly status: number` and message `` `SAP rejected the initial CPIC logon with status ${status}` `` | `src/client/direct-cpic-session.ts:249-257` |
| `CpicCallError` | class | `export class CpicCallError extends Error` with `readonly status: number` and message `` `SAP rejected the CPIC RFC call with status ${status}` `` | `src/client/direct-cpic-session.ts:259-267` |
| `DirectCpicOutgoingTransport` | interface | `send(payload: Uint8Array, signal?: AbortSignal): Promise<void>; receive?(options: { readonly timeoutMs: number; readonly signal?: AbortSignal; }): Promise<Uint8Array>; close(): Promise<void>;` | `src/client/direct-cpic-session.ts:383-390` |
| `DirectCpicOutgoingWriteError` | class | `export class DirectCpicOutgoingWriteError extends Error` with `readonly transmission: RfcTransmissionState.Partial | RfcTransmissionState.Unknown;` and `constructor(transmission, cause)` | `src/client/direct-cpic-session.ts:396-409` |
| `assertDirectCpicResponseIdentity` | function | `export function assertDirectCpicResponseIdentity( message: AppcMessage, conversationId: Uint8Array, connectionIndex: number, ): void` | `src/client/direct-cpic-session.ts:412-416` |
| `writeOutgoingAppcDataPlan` | function | `export async function writeOutgoingAppcDataPlan( transport: DirectCpicOutgoingTransport, setup: AppcClientSetupStateMachine, fragments: readonly AppcOutgoingDataFragment[], signal?: AbortSignal, barrierTimeoutMs = 30_000, ): Promise<void>` | `src/client/direct-cpic-session.ts:478-484` |
| `DirectCpicSession` | class | `export class DirectCpicSession` — see method list below | `src/client/direct-cpic-session.ts:660` |

`DirectCpicSession` public members (all verbatim):

| Member | Signature | Citation |
|---|---|---|
| `info` | `readonly info: DirectCpicSessionInfo;` | `:679` |
| `open` | `static async open(options: DirectCpicSessionOptions): Promise<DirectCpicSession>` | `:715` |
| `state` | `get state(): "allocated" | "authenticated" | "closed"` | `:958` |
| `logonAndPing` | `async logonAndPing( options: DirectCpicLogonOptions, signal?: AbortSignal, ): Promise<DirectCpicLogonResult>` | `:963-966` |
| `ping` | `async ping(signal?: AbortSignal): Promise<DirectCpicPingResult>` | `:1044` |
| `resetServerContext` | `async resetServerContext(signal?: AbortSignal): Promise<void>` | `:1060` |
| `getFunctionInterface` | `async getFunctionInterface( functionName: string, signal?: AbortSignal, ): Promise<RfcFunctionInterface>` | `:1097-1100` |
| `getOptimizedFunctionInterface` | `async getOptimizedFunctionInterface( functionName: string, language = "E", signal?: AbortSignal, ): Promise<RfcFunctionInterface>` | `:1124-1128` |
| `getOptimizedFunctionDescriptor` | `async getOptimizedFunctionDescriptor( functionName: string, language = "E", signal?: AbortSignal, ): Promise<RfcMetadataGetFunctionResult>` | `:1141-1145` |
| `getOptimizedRecursiveFunctionDescriptor` | `async getOptimizedRecursiveFunctionDescriptor( functionName: string, language = "E", signal?: AbortSignal, ): Promise<RfcMetadataGetRecursiveFunctionResult>` | `:1168-1172` |
| `invokeClassic` | `async invokeClassic( functionName: string, input: ClassicRfcInput, signal?: AbortSignal, options: ClassicRfcInvocationOptions = {}, ): Promise<ClassicRfcOutput>` | `:1191-1196` |
| `invokeClassicWithMetadata` | `async invokeClassicWithMetadata( metadata: RfcFunctionInterface, input: ClassicRfcInput, structures: RfcStructureRepository, signal?: AbortSignal, options: ClassicRfcInvocationOptions = {}, recursiveMetadata?: RecursiveMetadataGraph, ): Promise<ClassicRfcOutput>` | `:1322-1329` |
| `getStructureDefinition` | `async getStructureDefinition( structureName: string, signal?: AbortSignal, ): Promise<RfcStructureDefinition>` | `:1415-1418` |
| `getOptimizedStructureDefinition` | `async getOptimizedStructureDefinition( structureName: string, language = "E", signal?: AbortSignal, ): Promise<RfcStructureDefinition>` | `:1441-1445` |
| `getOptimizedStructureDescriptor` | `async getOptimizedStructureDescriptor( structureName: string, language = "E", signal?: AbortSignal, ): Promise<RfcMetadataGetStructureResult>` | `:1454-1458` |
| `getOptimizedMetadataTimestamps` | `async getOptimizedMetadataTimestamps( functionNames: readonly string[], structureNames: readonly string[], signal?: AbortSignal, ): Promise<RfcMetadataTimestampBatch>` | `:1481-1485` |
| `getLegacyStructureDefinition` | `async getLegacyStructureDefinition( structureName: string, signal?: AbortSignal, ): Promise<RfcStructureDefinition>` | `:1509-1512` |
| `exchange` | `async exchange(data: Uint8Array, signal?: AbortSignal): Promise<Buffer>` | `:1534` |
| `close` | `async close(): Promise<void>` | `:1804` |

`CpicLogonError` and `CpicCallError` are exported but **never thrown anywhere in
`src/`** — the only constructions are in `test/direct-cpic-session.test.ts:544-545`.
Logon rejection instead produces an `RfcCoreError` (`:1019-1030`).

### States and transitions

Public state, verbatim: `get state(): "allocated" | "authenticated" | "closed"`, computed as
`if (this.#closed || this.#setup.state === "closed") return "closed"; return this.#authenticated ? "authenticated" : "allocated";`
— `src/client/direct-cpic-session.ts:958-961`.

Backing private state: `#busy = false`, `#compoundOperationOwner: symbol | undefined`,
`#closed = false`, `#authenticated = false`, `#cpicSessionId: Buffer | undefined`
(`src/client/direct-cpic-session.ts:670-676`).

| Transition | Enforcement | Citation |
|---|---|---|
| (none) → `allocated` | `DirectCpicSession.open` completes gateway `GW_NORMAL_CLIENT` + APPC `Initialize` → `SetPartnerLuName` → `Allocate` and only then constructs the session | `:766-915` |
| open failure → transport closed | `catch (error) { await transport.close().catch(() => undefined); throw error; }` | `:916-919` |
| `allocated` → `authenticated` | `this.#cpicSessionId = Buffer.from(cpicSessionId); this.#authenticated = true;` after a successful decoded logon | `:1032-1033` |
| second logon rejected | `if (this.#authenticated) { throw new Error("direct CPIC session is already authenticated"); }` | `:967-969` |
| any op before auth rejected | `if (!this.#authenticated || this.#cpicSessionId === undefined) throw new Error("direct CPIC session must be authenticated before …")` — 7 call sites | `:1045-1047`, `:1061-1065`, `:1101-1105`, `:1146-1150`, `:1173-1177`, `:1419-1423`, `:1459-1463`, `:1486-1490`, `:1513-1517` |
| `*` → `closed` (failure) | `#terminateGeneration()` sets `#closed = true`, zeroes and drops `#cpicSessionId`, clears the metadata and structure caches, closes the transport | `:1794-1802` |
| `*` → `closed` (orderly) | `close()` sends `AppcFunction.Deallocate` only `if (this.#setup.state === "ready")`, then always `await this.#transport.close()` | `:1804-1836` |
| close during an exchange rejected | `if (this.#busy || this.#compoundOperationOwner !== undefined) { throw new Error("cannot close a direct CPIC session during an exchange"); }` | `:1806-1808` |
| single-flight | `#assertExchangeAvailable(owner?)` throws `RfcCoreError` with `reasonCode: "RFC_CONCURRENT_CALL"` when `#busy` or when a compound operation owned by a different symbol is active | `:1538-1555` |
| compound operation | `resetServerContext` allocates `const owner = Symbol("reset-server-context")`, sets `#compoundOperationOwner`, runs `SYSTEM_RESET_RFC_SERVER` then a full `RFC_PING` refresh, and clears the owner in `finally` | `:1060-1095` |
| exchange on a closed session | `if (this.#closed) throw new Error("direct CPIC session is closed");` | `:1562` |
| APPC write precondition | `if (setup.state !== "ready") { throw new Error(`cannot start an outgoing APPC message while client is ${setup.state}`); }` | `:540-542` |

Gateway handshake acceptance rules enforced at open: `returnCode !== 0` rejected
(`:790-792`); `appcHeaderVersion !== 6` rejected (`:793-797`);
`ExtendedInitOptions` bit required (`:798-800`); `CodePage` bit **and**
`gateway.codePage === "4103"` required (`:801-808`).

Receive loop dispositions (`#exchange`): `disposition === "normal-deallocation"` →
`decoder.pushTerminalDeallocation(payload)` and, on a complete message, the generation
is terminated before the buffered reply is returned (`:1638-1656`); otherwise the
transport is re-checked for a coalesced second frame and `this.#setup.responseComplete()`
is called (`:1657-1663`).

### Errors and error codes

| Code/message (verbatim) | Trigger | Recoverable per source? | Citation |
|---|---|---|---|
| `reasonCode: "RFC_CONCURRENT_CALL"`, `key: "RFC_ILLEGAL_STATE"`, `message: "direct CPIC session already has an in-flight operation"`, category `InvalidState`, origin `Api`, phase `Send`, transmission `NotStarted` | second concurrent exchange | disposition `Reusable` by policy (NotStarted + Api) → `RecoveryAction.None`; the session is **not** terminated | `:1544-1553`, policy at `src/client/rfc-failure.ts:378-392` |
| `reasonCode: "RFC_APPC_REQUEST_ENCODING_FAILED"`, `key: "RFC_SERIALIZATION_FAILURE"`, `message: "RFC request could not be encoded for APPC"`, category `Serialization`, origin `Codec`, phase `Encode`, transmission `NotStarted` | request too large, non-`Uint8Array`, or framing inspection failure | `Reusable`; no `#terminateGeneration()` on this path | `:1597-1608` |
| `"RFC request exceeds the direct CPIC request envelope"` | `requestByteLength > DEFAULT_MAX_APPC_OUTGOING_MESSAGE_LENGTH + APPC_FINAL_SAP_PARAMETER_LENGTH` | wrapped into the above | `:1570-1578` |
| `reasonCode: "RFC_CPIC_LOGON_RESPONSE_MALFORMED"`, `key: "RFC_INVALID_PROTOCOL"`, `message: "CPIC RFC logon response is malformed"`, category `MalformedProtocol`, origin `Cpic`, phase `Logon`, transmission `Complete`, `establishedSession: false` | logon response fails to decode | `UnknownClose`; `await this.#terminateGeneration()` before throwing | `:993-1005` |
| `reasonCode` = `"RFC_CPIC_LOGON_REJECTED"` or `` `RFC_CPIC_LOGON_STATUS_${decoded.status}` ``, `key: "RFC_LOGON_FAILURE"`, category `Logon`, origin `Sap`, phase `Logon`, transmission `Complete`, `establishedSession: false` | decoded logon not successful; message is the backend rejection text when present | `Close`; generation terminated | `:1007-1030` |
| `reasonCode: "RFC_CPIC_RESPONSE_MALFORMED"`, `key: "RFC_INVALID_PROTOCOL"`, `message: "CPIC RFC response is malformed"`, category `MalformedProtocol`, origin `Cpic`, phase `EnvelopeDecode`, transmission `Complete` | envelope decode failure for regular/reset/sessionRefresh replies | `UnknownClose`; generation terminated | `:1733-1746` |
| remote failure via `createRemoteRfcFailure(decoded.envelope, …)`; origin forced to `RfcFailureOrigin.Appc` when `terminalTransport` | `decoded.envelope.outcome !== "success"` | generation terminated **only** `if (failure.disposition !== RfcConnectionDisposition.Reusable)` | `:1748-1757` |
| `reasonCode: "CM_DEALLOCATED_NORMAL"`, `key: "RFC_COMMUNICATION_FAILURE"`, `message: "the peer normally deallocated the conversation after its response"`, category `Communication`, origin `Appc`, phase `EnvelopeDecode`, transmission `Complete` | a **successful** envelope arriving after `this.#setup.state === "closed"` | `Close` | `:1758-1769` |
| `reasonCode: "RFC_RESPONSE_VALUE_MALFORMED"`, `key: "RFC_INVALID_PROTOCOL"`, `message: "RFC response values are malformed"`, category `MalformedProtocol`, origin `Codec`, phase `ValueDecode`, transmission `Complete` | value-decode throw, except `ClassicBcdConversionError` which is rethrown unchanged | `UnknownClose`; generation terminated | `:1773-1791` |
| `reasonCode: "CM_NO_DATA_RECEIVED"` | `AppcNormalDeallocationWithoutDataError` | category `Communication`, `Close` | `:1689-1691` |
| `` reasonCode: `RFC_APPC_RETURN_${transportCause.appcReturnCode}_SAP_${transportCause.sapReturnCode}` `` | `AppcPeerReturnCodeError` | category `Communication`, `Close` | `:1692-1693` |
| `reasonCode` = `niError?.code ?? "RFC_APPC_RESPONSE_MALFORMED"` | NI transport error, or an unrecognized throw | category mapping below | `:1694`, `:1704` |
| `"APPC response identity does not match the active conversation"` | conversation id ≠ 8 bytes, id mismatch, `communicationIndex !== 0`, or `connectionIndex` mismatch | thrown inside the receive loop → wrapped as a transport failure | `:412-425` |
| `"APPC reply contained more than one RFC message"` | decoder yields >1 message | same | `:1642-1644` |
| `"APPC synchronous-send acknowledgement identity changed"` | streaming barrier ack with wrong conversation id or connection index | wrapped in `DirectCpicOutgoingWriteError` | `:555-562` |
| `"outgoing APPC message write failed"` (`DirectCpicOutgoingWriteError`) | any send/barrier failure inside the plan | see the ambiguous-send section | `:404` |
| `"outgoing APPC plan must contain at least one fragment"` / `` `outgoing APPC plan fragment count ${…} exceeds limit ${DEFAULT_MAX_APPC_MESSAGE_FRAGMENTS}` `` / `"outgoing APPC plan fragment order is inconsistent"` / `"outgoing APPC plan identity changed between fragments"` / `"outgoing APPC plan application length is unsafe"` / `"outgoing APPC plan application length is inconsistent"` | plan validation | pre-wire | `:431-468`, `:486-497` |
| `"outgoing APPC transport needs send and close methods"` / `"outgoing APPC transport needs receive for synchronous streaming barriers"` | transport shape | pre-wire | `:513-515`, `:532-539` |
| `"barrierTimeoutMs must be an integer in 0..2147483647"` | bad barrier timeout | pre-wire | `:503-509` |
| `DirectCpicPreWireError` — message is `cause.message` when the cause is an `Error`, else `"classic RFC invocation preparation failed"` | any throw while preparing a classic invocation (serializer admission, request build) | `COMMENT:` `Proven local invocation-preparation failure. No application request byte was handed to the CPIC exchange, so the authenticated generation remains safe.` | `:160-177`, thrown at `:1275`, `:1401` |
| `"gateway rejected GW_NORMAL_CLIENT with return code …"` / `"gateway selected unsupported APPC header version …"` / `"gateway did not accept extended initialization options"` / `"direct classic RFC supports only little-endian Unicode partner code page 4103 (M12)"` | handshake validation at open | plain `Error`; the transport is closed by the open-path `catch` | `:790-808`, `:916-919` |
| `"applicationServerService must use the direct application-server form sapdpNN"` / `"operationTimeoutMs must be an integer in 0..2147483647"` / `"recursiveSerializerDecisionProvider must be a function"` / `"applicationServerHost must contain 1..64 ASCII bytes"` / `"programName must contain 1..64 ASCII bytes"` / `"cpicStreaming must be disabled or enabled"` / `"transportFactory must be a function"` / `"transportFactory must return a NiSocketTransport"` / `"direct CPIC gateway version 2 needs an IPv4 localAddress override"` | option validation at open | pre-connect | `:600-629`, `:718-724`, `:736-747`, `:749-756` |

Transport-error → category mapping in `#exchange` (`:1678-1688`), verbatim structure:
`NI_ABORTED` → `Canceled`; `NI_RECEIVE_TIMEOUT` / `NI_CONNECT_TIMEOUT` /
`NI_WRITE_TIMEOUT` → `Timeout`; `AppcPeerReturnCodeError` or
`AppcNormalDeallocationWithoutDataError` → `Communication`; `NI_PROTOCOL_ERROR` **or no
`NiTransportError` at all** → `MalformedProtocol`; any other NI code → `Communication`.
Origin is `Ni` when an `NiTransportError` is present, otherwise `Appc` (`:1697`).

Optional-recursive-metadata fallback is admitted only for
`RecursiveMetadataError` with `code === "REMOTE_DDIC_RESOLUTION_ERRORS"`, or an
`RfcCoreError` whose `failure.disposition === RfcConnectionDisposition.Reusable`
**and** whose key is in `OPTIMIZED_METADATA_UNAVAILABLE_KEYS = {"FU_NOT_FOUND",
"FUNCTION_NOT_EXIST", "RFC_NOT_FOUND"}` (only for `AbapException`) or in
`OPTIMIZED_METADATA_AUTHORIZATION_KEYS = {"CALL_FUNCTION_NO_AUTHORITY",
"RFC_NO_AUTHORITY", "RFC_AUTHORIZATION_FAILURE"}` (key or `abap.runtimeId`) —
`:195-219`, used at `:1234-1242`.

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `* Proven local invocation-preparation failure. No application request byte was` / `* handed to the CPIC exchange, so the authenticated generation remains safe.` | `:161-163` |
| `* Pre-encodes the complete bounded plan, then writes one NI payload at a time.` / `* A transport failure is terminal and is never retried or replayed.` | `:471-473` |
| `* This low-level helper is exported from its module for deterministic fault` / `* tests but is intentionally not part of the package root API.` | `:474-476` |
| `// The original transmission failure remains authoritative.` (in the close-on-failure `catch`) | `:569` |
| `/** Enforce response identity without correlating the independently sequenced reply. */` | `:411` |
| `// Capture every caller-owned property before validation or I/O. Setup spans` / `// several awaits; rereading a mutable/accessor-backed option later could mix` / `// endpoint, timeout, signal, and CPIC identity from different generations.` | `:637-639` |
| `// The APPC setup contract always carries this initialization flag.` / `// Streaming stays a caller-supplied local policy because no` / `// peer-negotiation bit for it is known.` | `:830-832` |
| `// The backend explains itself; decoding that and then dropping it is` / `// what made every rejection look alike to a caller.` | `:1011-1012` |
| `// Reset clears SAP's RFC session-header state. Re-establish it immediately` / `// with the independently observed full RFC_PING form so callers and pools` / `// only regain a connection after the refreshed envelope is validated.` | `:1078-1080` |
| `* Execute with a destination-owned immutable metadata snapshot. This keeps` / `* repository-lane failures outside the application-session disposition path.` | `:1319-1320` |
| `// A complete frame left by the previous invocation must never be` / `// attributed to this request. This check also retires the transport.` | `:1614-1615` |
| `// CPI-C is already in Reset and its conversation ID is invalid.` / `// The buffered RFC envelope may be decoded once, but this physical` / `// generation must never be lent or reused again.` | `:1653-1655` |
| `// Reject a coalesced second complete NI frame before returning the` / `// first response to the caller. A later frame is caught by the` / `// pre-send boundary above.` | `:1658-1660` |
| `// exchange() can return bytes after CM_DEALLOCATED_NORMAL, but it closes` / `// the generation before doing so. That terminal transport fact must` / `// override otherwise-reusable RFC envelope outcomes.` | `:1722-1724` |
| `/** Internal connection seam used by direct TCP and already-routed streams. */` | `:123` |
| `* IPv4 address advertised inside CPIC setup. Loopback is the interoperable` / `* default for an outbound client behind NAT; routed callback deployments may` / `* override it with an address reachable from the SAP gateway.` | `:137-141` |
| `* Paired, release-specific observation required for a recursive live send.` / `* Flat/classic calls do not require this policy.` | `:153-156` |
| `/** Process-local identity of this physical session generation. */` (`generationHandle`) and `/** SAP gateway connection-table index; the peer may recycle this value. */` (`connectionIndex`) | `:183`, `:185` |
| `* Load one function descriptor through SAP Note 1456826's bounded classic` / `* RFC_METADATA_GET form. The destination repository owns cross-session` / `* caching and capability fallback; this session method performs one call.` | `:1120-1122` |
| `/** Reset backend function-pool state while preserving the synchronized session. */` | `:1059` |
| `/** Explicit compatibility path for pre-DDIF legacy-v3 repositories only. */` | `:1508` |

### Behaviour facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| Full open → exchange → deallocate over a scripted TCP peer, asserting the gateway record, `Initialize` with a blank 8-byte conversation id, `optionFlags: 1`, `longLogicalUnitName: "127.0.0.1"`, then the APPC function-code sequence ending in `AppcFunction.Deallocate` | `"opens, exchanges, and deallocates a semantic direct-CPIC session"` | `test/direct-cpic-session.test.ts:61-535` |
| `DirectCpicPreWireError` message falls back to `"classic RFC invocation preparation failed"` for a non-Error cause; `CpicLogonError(7).status === 7`; `CpicCallError(8).status === 8`; open rejects non-object options, bad `operationTimeoutMs`, bad `cpicStreaming`, `sapgw00`, and a newline in `applicationServerHost` | `"rejects unsupported direct application-server service names"` | `test/direct-cpic-session.test.ts:537-589` |
| A gateway offering code page `"4102"`, or not setting the `CodePage` accept bit, is rejected with `/only little-endian Unicode partner code page 4103 \(M12\)/u` | `"rejects a gateway that does not select little-endian Unicode code page 4103"` | `test/direct-cpic-session-errors.test.ts:90-118` |
| Logon status 7 → `Logon`/`Close`/`RFC_LOGON_FAILURE`/`RFC_CPIC_LOGON_STATUS_7`; terminal error envelope → `RFC_CPIC_LOGON_REJECTED`; malformed → `MalformedProtocol`/`UnknownClose`/`RFC_CPIC_LOGON_RESPONSE_MALFORMED`. All three give `RecoveryAction.None` (pre-auth) and leave `session.state === "closed"` | `"terminally closes rejected and malformed initial-logon generations"` | `test/direct-cpic-session-errors.test.ts:120-164` |
| The malformed-logon `failure.cause` is a `CpicInitialLogonStructureError` carrying `rule` and a per-field `{tag, byteLength, index}` list, and `JSON.stringify(diagnostic) === "{}"` | `"retains redaction-safe initial-logon structure diagnostics as the failure cause"` | `test/direct-cpic-session-errors.test.ts:166-207` |
| A declared exception leaves `disposition === Reusable`, `recoveryAction === None`, `session.state === "authenticated"`, and a **second call on the same physical connection succeeds** (`peer.connectionCount === 1`, `peer.regularRequestCount(0) === 2`) | `"keeps a validated declared exception reusable for a same-generation follow-up"` | `test/direct-cpic-session-errors.test.ts:209-240` |
| ABAP runtime → `Close`; MESSAGE X → `Close`; malformed reply fields → `UnknownClose`. All three: `RecoveryAction.Replace`, `session.state === "closed"` | `"closes runtime, MESSAGE, and malformed response generations"` | `test/direct-cpic-session-errors.test.ts:242-301` |
| With APPC return code 18 (normal deallocation), a *declared exception* becomes `UnknownClose` and a *successful* envelope becomes a `Communication`/`Close` failure; both `Replace`, both close the session | `"normal deallocation overrides reusable and successful RFC envelopes"` | `test/direct-cpic-session-errors.test.ts:303-339` |
| APPC 18 with no data → `reasonCode "CM_NO_DATA_RECEIVED"`, message `"connection closed without message (CM_NO_DATA_RECEIVED)"`; APPC 17 → `"RFC_APPC_RETURN_17_SAP_0"`, message `"F_SAP_SEND failed with APPC return code 17 and SAP return code 0"`; both `Communication`/`Close`/`Replace`/`key "RFC_COMMUNICATION_FAILURE"`/closed | `"classifies peer statuses and empty normal deallocation as communication failures"` | `test/direct-cpic-session-errors.test.ts:341-377` |
| Socket close → `Communication`; silence → `Timeout`; `AbortController.abort()` → `Canceled`. All three: `RecoveryAction.Replace` and `session.state === "closed"` | `"terminally closes transport, timeout, and abort generations"` | `test/direct-cpic-session-errors.test.ts:379-399` |

---

## src/lifecycle/session-context-runtime.ts

1371 lines. A destination-generation-owned registry of stateful RFC contexts
(`begin`/`run`/`end` with reference counting), sitting on a pool-independent lease
adapter.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `SessionContextScope` | interface | `readonly destinationId: string; readonly configurationGenerationId: string;` | `:1-4` |
| `SessionContextToken` | interface | `export interface SessionContextToken extends SessionContextScope` + `readonly runtimeId: number; readonly contextId: number;` | `:10-13` |
| `SessionContextAcquireContext<_C>` | interface | `readonly token: SessionContextToken; readonly scope: SessionContextScope; readonly signal: AbortSignal;` | `:15-20` |
| `SessionContextCleanupContext` | interface | `readonly token: SessionContextToken; readonly signal: AbortSignal;` | `:22-26` |
| `SessionContextOperationContext` | interface | `readonly token: SessionContextToken; readonly signal: AbortSignal;` | `:28-32` |
| `SessionContextReleaseReason` | type | `"context-end" | "begin-rollback" | "reset-failed" | "fatal-operation" | "runtime-retire"` | `:34-39` |
| `SessionContextReleaseDisposition` | interface | `readonly reusable: boolean; readonly reason: SessionContextReleaseReason;` | `:41-45` |
| `SessionContextLeaseAdapter<L, C>` | interface | `acquire(context)`, `resource(lease): C`, `reset(lease, resource, context)`, `release(lease, disposition, context)` | `:53-69` |
| `SessionContextFailureContext` | interface | `readonly token: SessionContextToken;` | `:71-73` |
| `SessionContextFatalEvent` | interface | `readonly token; readonly cause: unknown; readonly signal: AbortSignal;` | `:75-80` |
| `SessionContextScheduledTask` | interface | `cancel(): void;` | `:82-84` |
| `SessionContextScheduler` | interface | `now(): number; schedule(delayMs: number, callback: () => void): SessionContextScheduledTask;` | `:87-93` |
| `SessionContextRuntimeOptions<L, C>` | interface | `scope`, `leases`, `operationTimeoutMs`, `scheduler?`, `isFatal?`, `onFatal?` | `:95-113` |
| `SessionContextRuntimeState` | type | `export type SessionContextRuntimeState = "open" | "retiring" | "retired";` | `:115` |
| `SessionContextRuntimeErrorCode` | type | 9-member union, listed below | `:117-126` |
| `SessionContextRuntimeError` | class | `export class SessionContextRuntimeError extends Error` with `readonly code: SessionContextRuntimeErrorCode;` and `constructor(code, message)` | `:128-136` |
| `SessionContextRuntimeMonitor` | interface | 30 readonly counters | `:138-171` |
| `SessionContextRuntime<L, C>` | class | `constructor(options: SessionContextRuntimeOptions<L, C>)`; `get scope(): SessionContextScope`; `begin(): Promise<SessionContextToken>` / `begin(token: SessionContextToken): Promise<SessionContextToken>`; `async run<R>(token, operation): Promise<R>`; `async end(token: SessionContextToken): Promise<void>`; `retire(): Promise<void>`; `close(): Promise<void>`; `monitor(): SessionContextRuntimeMonitor` | `:456`, `:487`, `:509`, `:513-515`, `:626-632`, `:722`, `:786`, `:821`, `:825` |

### States and transitions

Two independent state variables.

**Runtime state** `"open" | "retiring" | "retired"` (`:115`), stored in `#state`
(`:481`):

| Transition | Enforcement | Citation |
|---|---|---|
| `open` → `retiring` | `retire()` sets `this.#state = "retiring";` **synchronously**, before any await | `:794` |
| `retiring` → `retired` | at the end of `#completeRetirement`, after all openings, terminals and late cleanups settle | `:1355` |
| idempotence | `if (this.#retirement !== undefined) return this.#retirement;` | `:788` |
| `close()` | `return this.retire();` — pure alias | `:821-823` |
| gate | `#requireOpen(operation)` throws `RUNTIME_RETIRED` whenever `#state !== "open"` | `:857-866` |

**Per-context state** `type ContextState = "ready" | "ending" | "retiring" | "fatal" | "closed"`
(`:196`), plus `terminalKind?: "normal" | "fatal" | "retired"` (`:203`):

| Transition | Trigger | Release disposition passed to the adapter | Citation |
|---|---|---|---|
| (none) → `ready` | `begin()` after `acquire` + `resource`, with `references: 1` | — | `:579-595` |
| `ready` → `ready` (refcount++) | `begin(token)` with an existing token, rejected if `entry.activeOperation` | — | `:519-532` |
| `ready` → `ending` | final `end(token)` (`references` reaches 0) | `reset` first; then `reusable = resetError === undefined && this.#state === "open"`, reason `"reset-failed"` / `"context-end"` / `"runtime-retire"` | `:769-780`, `:925-982`, `:956-962` |
| `ready` → `fatal` | `run()` failure classified fatal (or an `OPERATION_TIMEOUT`, or a throwing `isFatal`) | `reusable: false`, reason `"fatal-operation"`; **no reset call** | `:671-716`, `:984-1011`, `:1000` |
| `ready` → `retiring` | `retire()` claims every `ready` entry synchronously | `reusable: false`, reason `"runtime-retire"`; **no reset call** | `:799-804`, `:1013-1041`, `:1034` |
| `ending`/`retiring`/`fatal` → `closed` | `#closeEntry` sets `entry.state = "closed"`, deletes from `#entries` (unless retained for notification) and decrements `pinnedLeases` | — | `:1130-1143` |
| begin rollback | `begin()` failure after a lease was acquired | `reusable: false`, reason `this.#state === "open" ? "begin-rollback" : "runtime-retire"` | `:596-618`, `:604-609` |
| late acquire | lease arrives after the bounded acquire has already timed out | same rule as begin rollback, via `#cleanupLateAcquire` | `:1145-1156` |
| release once-only | `if (entry.releaseClaimed) return; entry.releaseClaimed = true;` before crossing the adapter boundary | — | `:1072-1087` |
| fatal ordering | on fatal, `references` is zeroed, the entry is removed from ownership, release runs, **then** `#notifyFatal`, **then** `this.#entries.delete(entry)` | — | `:984-1011` |
| last-instant reusable re-check | inside `#releaseLease`: `const effectiveReusable = reusable && this.#state === "open" && !signal.aborted;` and when `reusable` was requested but denied, the reason is rewritten to `"runtime-retire"` | — | `:1099-1111` |

`#requireReady` rejection order: `terminalKind === "fatal"` → `CONTEXT_FATAL`;
`terminalKind === "retired"` → `RUNTIME_RETIRED`; `state === "ending"` →
`CONTEXT_ENDING`; `state !== "ready" || references < 1` → `CONTEXT_CLOSED` (`:868-890`).

### Errors and error codes

| Code/message (verbatim) | Trigger | Recoverable per source? | Citation |
|---|---|---|---|
| `"INVALID_CONTEXT_TOKEN"` / `"session context token does not belong to this runtime"` | non-object token, or a token not in the runtime's `WeakMap` | no state change | `:837-855` |
| `"CONTEXT_CLOSED"` / `` `cannot ${operation}: session context is closed` `` | entry not `ready` or `references < 1` | terminal for that context | `:884-889` |
| `"CONTEXT_FATAL"` / `` `cannot ${operation}: session context is fatal` `` and `"session context was removed after a fatal operation"` | use of a fatally-removed context; `end()` on it | terminal | `:869-874`, `:732-738` |
| `"CONTEXT_ENDING"` / `` `cannot ${operation}: session context end is in progress` `` and `"session context end is already in progress"` | overlapping `end()` | terminal for that context | `:878-883`, `:746-752` |
| `"UNMATCHED_CONTEXT_END"` / `"session context has no unmatched begin"` | `end()` on a closed entry, or with `references < 1` | — | `:739-745`, `:761-767` |
| `"CONCURRENT_CONTEXT_OPERATION"` / `"session context already has an active operation"` | second `run()` on the same context | context stays reusable | `:639-645` |
| `"ACTIVE_CONTEXT_OPERATION"` / `"cannot nest begin while a context operation disposition is active"` and `"cannot end a session context during an active operation"` | `begin(token)` or `end(token)` during `run()` | context stays reusable | `:523-527`, `:753-760` |
| `"RUNTIME_RETIRED"` / `` `cannot ${operation}: session context runtime is retiring or retired` `` | any call while `#state !== "open"`; also used as the abort reason broadcast by `retire()` | terminal | `:861-866`, `:810-813` |
| `"OPERATION_TIMEOUT"` / `` `${label} exceeded ${this.#operationTimeoutMs}ms` `` | any bounded boundary exceeding its deadline | **treated as fatal for `run()`**: `let fatal = error instanceof SessionContextRuntimeError && error.code === "OPERATION_TIMEOUT";` | `:1236-1239`, `:671-673` |
| `"session context scheduler clock must be finite and monotonic"` | `now()` non-finite or going backwards | boundary rejects | `:1166-1175` |
| `"session context deadline must be finite"` | `now() + operationTimeoutMs` non-finite | boundary rejects | `:1202-1203` |
| `"session context scheduler fired before its deadline without bounded progress"` | scheduler fires early ≥64 times or without shrinking `remaining` | boundary rejects | `:1265-1276`, constant `MAX_EARLY_TIMER_REARMS = 64` at `:258` |
| `AggregateError` with message `"session context begin and lease rollback both failed"` / `"session operation and fatal classification both failed"` / `"fatal session operation and eviction both failed"` / `"session context reset and eviction both failed"` / `"session context retirement had multiple cleanup failures"` | paired failures | `cause` is set to the primary failure (`:392-398`) | `:610-616`, `:690-696`, `:707-714`, `:972-977`, `:1362-1368` |
| `TypeError`s: `"session context runtime options must be an object"`, `"session context scope must be an object"`, `"session context lease adapter must be an object"`, `` `${path} must be a function` ``, `"scheduler requires now and schedule"`, `"scheduler must return a cancelable task"`, `"leases.acquire must resolve to an object lease"`, `"leases.resource must return an object resource"`, `"isFatal must be a function"`, `"onFatal must be a function"`, `"isFatal must return a boolean"`, `"session context operation must be a function"` | construction / adapter contract | pre-I/O | `:488-489`, `:290-291`, `:317-318`, `:305-307`, `:349-350`, `:366-367`, `:570`, `:575`, `:497-502`, `:685-687`, `:633-634` |
| `RangeError`: `` `${path} must contain 1..512 characters without controls` ``, `` `operationTimeoutMs must be finite and in 1..${MAX_TIMER_MS}` `` (`MAX_TIMER_MS = 2_147_483_647`) | scope/timeout validation | pre-I/O | `:275-287`, `:376-383`, `:257` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `* An explicit, immutable context identity. Token object identity is also` / `* checked, so copying these public fields cannot forge ownership.` | `:6-9` |
| `/** Aborted when the owning destination generation retires or the step times out. */` | `:18` |
| `/** Aborted on a deadline and, for reusable work, on runtime retirement. */` | `:24` |
| `/** False requires the adapter to close or evict the physical connection. */` | `:42` |
| `* Pool-independent ownership seam. A release call is a once-only ownership` / `* transfer. When `reusable` is false, it must converge closure/eviction before` / `* settling, including when it rejects. Implementations must observe the` / `* supplied signals so a generation can retire within its configured bound.` | `:47-52` |
| `/** A monotonic scheduling boundary which deterministic tests can replace. */` | `:86` |
| `/** Finite deadline shared by operations and every adapter/observer step. */` | `:101` |
| `/** Synchronous because ownership must be decided before another operation. */` | `:104` |
| `/** Invoked after fatal registry removal and physical lease convergence. */` | `:109` |
| `/** Published contexts. A fatal context is removed before cleanup begins. */` | `:141` |
| `* Destination-generation-owned stateful RFC context registry.` / `* Ownership transitions happen synchronously before an external Promise or` / `* callback is entered. Every asynchronous boundary has a deadline and signal;` / `* retirement first closes ownership, then aborts and converges prior work.` | `:449-455` |
| `// `resource` is synchronous but caller-owned and may reenter retire().` | `:577` |
| `* Synchronously closes the ownership gate and idempotently converges every` / `* opening or pinned context. Unmatched/lost tokens are therefore harmless.` | `:782-784` |
| `// State and all ready entries are closed before abort listeners can reenter.` | `:809` |
| `// Keep the terminal in #entries through notification so a concurrent` / `// retire waits for the bounded owner callback too.` | `:1004-1005` |
| `// Observer failure cannot resurrect or change the fatal disposition.` | `:1067` |
| `// Claim before crossing the adapter boundary. No race can invoke release` / `// twice, even when the first call times out or reenters retire().` | `:1078-1079` |
| `// The scheduler is an external/reentrant boundary. Re-check the` / `// ownership gate at the last possible instant so it cannot turn a` / `// retired generation into a reusable hand-off.` | `:1099-1101` |
| `// Eviction must continue during retirement. A reusable hand-off is` / `// instead aborted so the adapter cannot recycle a retired generation.` | `:1120-1121` |
| `// Avoid an `await undefined` yield: an idle/lost-token context should` / `// enter physical eviction in the same retirement turn. Active contexts` / `// still wait for their already-bounded operation to relinquish access.` | `:1030-1032` |
| `// A hostile abort listener cannot reopen ownership or prevent deadlines.` | `:1182` |
| `// Late acquisitions which arrive before logical retirement completes are` / `// also converged. A lease arriving after its bounded acquire has timed out` / `// is still evicted asynchronously by #bounded's late-fulfillment hook.` | `:1349-1351` |
| `// Context deadlines are safety bounds, not work which should keep Node` / `// alive after all application handles have closed.` | `:267-268` |

### Behaviour facts asserted by tests

All citations `test/session-context-runtime.test.ts`.

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| One generation-scoped lease is pinned and identities are immutable | `"pins one generation-scoped lease and publishes immutable identities"` | `:219` |
| Nested `begin`/`end` reference-count without a second acquire or an early release | `"nested begin/end reference-counts without acquiring or releasing early"` | `:273` |
| A failed acquire publishes no token or lease and a later `begin` can retry | `"failed acquire publishes no token or lease and a later begin can retry"` | `:302` |
| `resource()` failure evicts the partially acquired lease exactly once | `"resource setup failure evicts the partial lease exactly once"` | `:324` |
| A rollback failure is retained alongside the begin failure | `"a rollback failure is retained with the begin failure"` | `:348` |
| A second concurrent `run` rejects deterministically and the lease survives | `"rejects a second operation deterministically while preserving the lease"` | `:370` |
| `end` during an operation rejects without decrementing or releasing | `"end during an operation rejects without decrementing or releasing"` | `:396` |
| Concurrent final `end`s produce one reset and one release | `"concurrent final ends have one reset and one release"` | `:417` |
| Reset failure evicts once, never returns reusable, and closes the token | `"reset failure evicts once, never returns reusable, and closes the token"` | `:435` |
| Reset and eviction failures are both visible and cleanup is not retried | `"reset and eviction failures are both visible but cleanup is not retried"` | `:458` |
| A non-fatal operation failure leaves the context reusable: `adapter.releases.length === 0` and the next `run` on the same token succeeds | `"a nonfatal operation failure leaves the same context reusable"` | `:485-503` |
| Fatal failure evicts with `{reusable: false, reason: "fatal-operation"}`, removes the context from `contexts`/`references` **before** notification, and the owner is notified only after the physical release settles | `"fatal failure removes ownership, evicts once, then notifies the owner"` | `:505-549` |
| `onFatal` may call `begin()` reentrantly without deadlock | `"owner notification can reenter after fatal cleanup without deadlock"` | `:551-574` |
| A throwing `onFatal` increments `ownerNotificationFailures` and does not change the disposition (`pinnedLeases === 0`) | `"notification failure cannot change the fatal disposition"` | `:576-597` |
| A throwing `isFatal` conservatively evicts | `"classification failure conservatively evicts an uncertain session"` | `:599` |
| A reentrant classifier cannot start work before disposition | `"a reentrant classifier cannot start work before disposition"` | `:625` |
| Fatal eviction failure is combined and notification stays once | `"fatal eviction failure is combined and owner notification remains once"` | `:654` |
| Tokens cannot be forged, copied, or used with another runtime | `"tokens cannot be forged, copied, or used with another runtime"` | `:686` |
| Adapter methods are snapshotted; poisoned `bind` properties are ignored | `"snapshots caller methods and ignores poisoned bind properties"` | `:706` |
| Scope snapshot resists cross-field getter mutation and rejects control characters | `"scope snapshot resists cross-field getter mutation and rejects controls"` | `:784` |
| Monitor snapshots stay frozen and reconciled through an opening lease | `"monitor snapshots remain frozen and reconciled through an opening lease"` | `:816` |
| `retire()` is idempotent and evicts a context whose token was lost | `"retire is an idempotent ownership barrier and evicts a lost-token context"` | `:833` |
| `retire()` aborts an opening acquire and evicts a late lease exactly once | `"retire aborts an opening acquire and evicts a late lease exactly once"` | `:865` |
| `retire()` aborts and bounds an active operation before a single eviction | `"retire aborts and bounds an active operation before one eviction"` | `:903` |
| `retire()` during `ending` aborts a hung reset and preserves its timeout | `"retire during ending aborts a hung reset and preserves its timeout"` | `:935` |
| `retire()` waits for a bounded fatal eviction without a second release | `"retire waits for a bounded fatal eviction without invoking release twice"` | `:965` |
| A hung lost-token eviction still closes logical ownership by deadline | `"a lost-token eviction which hangs still closes logical ownership by deadline"` | `:1006` |
| Fatal owner notification has a signal and cannot exceed its deadline | `"fatal owner notification has a signal and cannot exceed its deadline"` | `:1028` |
| A non-fatal classifier cannot reentrantly add a context reference | `"a nonfatal classifier cannot reentrantly add a context reference"` | `:1057` |
| The scheduler boundary is validated and snapshotted | `"validates and snapshots the finite scheduler boundary"` | `:1087` |
| Retirement reentered by the scheduler cannot publish a reusable release | `"retirement reentered by the scheduler cannot publish a reusable release"` | `:1127` |

---

## src/lifecycle/transaction-runtime.ts

1663 lines. One-shot BAPI LUW coordinator over one exclusively pinned lease.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `TransactionRuntimeState` | type | 11-member union, listed below | `:3-14` |
| `TransactionOutcome` | type | 6-member union, listed below | `:16-22` |
| `TransactionToken` | interface | `readonly runtimeId: number; readonly transactionId: number;` | `:25-28` |
| `TransactionInvocationKind` | type | `export type TransactionInvocationKind = "business" | "commit" | "rollback";` | `:30` |
| `TransactionInvocation` | interface | `readonly kind; readonly functionName: string; readonly parameters: Readonly<Record<string, unknown>>; readonly notRequested?: readonly string[];` | `:32-38` |
| `TransactionCallOptions` | interface | `readonly notRequested?: readonly string[];` | `:40-42` |
| `TransactionAcquireContext` | interface | `readonly token: TransactionToken; readonly signal: AbortSignal;` | `:44-47` |
| `TransactionOperationContext` | interface | `readonly token; readonly operation: TransactionInvocationKind | "reset" | "release"; readonly signal: AbortSignal;` | `:49-53` |
| `TransactionReleaseReason` | type | `"commit" | "rollback" | "close-rollback" | "begin-rollback" | "ambiguous" | "control-rejected" | "reset-failed"` | `:55-62` |
| `TransactionReleaseDisposition` | interface | `readonly reusable: boolean; readonly reason: TransactionReleaseReason; readonly outcome: TransactionOutcome;` | `:64-68` |
| `TransactionLeaseAdapter<L>` | interface | `acquire`, `invoke`, `reset`, `release` | `:75-102` |
| `TransactionScheduledTask` | interface | `cancel(): void;` | `:104-106` |
| `TransactionScheduler` | interface | `now(): number; schedule(delayMs: number, callback: () => void): TransactionScheduledTask;` | `:109-112` |
| `TransactionFailureKind` | type | `export type TransactionFailureKind = "recoverable" | "ambiguous";` | `:114` |
| `TransactionFailureContext` | interface | `readonly token: TransactionToken; readonly invocation: TransactionInvocation;` | `:116-119` |
| `TransactionRuntimeOptions<L>` | interface | `leases`, `operationTimeoutMs`, `scheduler?`, `classifyFailure?` | `:121-137` |
| `TransactionRuntimeErrorCode` | type | `"INVALID_TRANSACTION_TOKEN" | "INVALID_TRANSACTION_STATE" | "TRANSACTION_CLOSING" | "CONCURRENT_TRANSACTION_OPERATION" | "OPERATION_TIMEOUT" | "INVALID_CONTROL_RESULT"` | `:139-145` |
| `TransactionRuntimeError` | class | `readonly code: TransactionRuntimeErrorCode;`, `constructor(code, message)` | `:147-155` |
| `TransactionBapiReturn` | interface | `readonly type: string; readonly id: string; readonly number: string; readonly message: string;` | `:157-162` |
| `TransactionBapiError` | class | `readonly code = "BAPI_REJECTED" as const; readonly operation: "commit" | "rollback"; readonly returns: readonly TransactionBapiReturn[]; readonly outcome = "rejected" as const;` | `:164-184` |
| `TransactionTerminalError` | class | `export class TransactionTerminalError extends AggregateError` with `readonly code = "TRANSACTION_TERMINAL_FAILURE" as const; readonly outcome: Exclude<TransactionOutcome, "none" | "active">;` | `:187-200` |
| `TransactionRuntimeMonitor` | interface | 27 readonly counters | `:202-234` |
| `TransactionRuntime<L>` | class | `constructor(options)`; `async begin(): Promise<TransactionToken>`; `async call(token, functionName, parameters, options = {})`; `commit(token): Promise<void>`; `rollback(token): Promise<void>`; `abort(token): Promise<void>`; `cancel(token): Promise<void>`; `close(): Promise<void>`; `release(): Promise<void>`; `monitor(): TransactionRuntimeMonitor` | `:600`, `:632`, `:647`, `:732-737`, `:861`, `:873`, `:888`, `:905`, `:916`, `:967`, `:971` |

### States and transitions

`TransactionRuntimeState` verbatim members: `"idle" | "opening" | "active" | "calling" | "committing" | "rollingBack" | "resetting" | "releasing" | "closing" | "failed" | "closed"` (`:3-14`).

`TransactionOutcome` verbatim members: `"none" | "active" | "committed" | "rolledBack" | "rejected" | "ambiguous"` (`:16-22`).

| Transition | Trigger / guard | Release disposition | Citation |
|---|---|---|---|
| `idle` → `opening` | `begin()`; any other starting state throws `INVALID_TRANSACTION_STATE` | — | `:647-656` |
| `opening` → `active`, outcome `active` | acquire resolves and `this.#state === "opening"` still holds | — | `:684-697` |
| `opening` → `idle` or `failed` | begin failure: `this.#state = this.#acquireQuarantines.size === 0 ? "idle" : "failed";` | `{reusable:false, reason:"begin-rollback", outcome:"none"}` | `:698-725`, `:703-710` |
| `active` → `calling` | `call()`; a second call throws `CONCURRENT_TRANSACTION_OPERATION` | — | `:732-763` |
| `calling` → `active` | business result, `#stateIs("closing") === false` | — | `:786-794` |
| `calling` → `active` (failure) | classifier returned `"recoverable"` | — | `:830-833` |
| `calling` → `failed`, outcome `ambiguous` | classifier absent/`"ambiguous"`/throwing, or `OPERATION_TIMEOUT` | `{reusable:false, reason:"ambiguous", outcome:"ambiguous"}` via `#finishReleaseOnly` | `:805-854`, `:838-840` |
| `active` → `committing` → `resetting` → `releasing` → `closed`, outcome `committed` | `commit(token)` with a clean `RETURN` | `{reusable: cleanup.length === 0, reason: "commit" | "reset-failed", outcome:"committed"}` | `:861-871`, `:1128-1235` |
| `active` → `rollingBack` → … → `closed`, outcome `rolledBack` | `rollback(token)` | reason `"rollback"` | `:873-885` |
| control call rejected by BAPI | `TransactionBapiError` from `inspectControlReturn` | outcome `rejected`; `{reusable:false, reason:"control-rejected", outcome:"rejected"}` | `:1159-1169` |
| control call failed otherwise | any non-BAPI error from `invoke`/`resultObject` | outcome `ambiguous`; `{reusable:false, reason:"ambiguous", outcome:"ambiguous"}` | `:1170-1173` |
| `active` → `closing` → rollback | `close()` while `active` | reason `"close-rollback"` | `:939-944` |
| `calling` → `closing` | `close()` or `cancel(token)` during a call — both route to `#closeDuringCall` | see the ambiguous-send section | `:905-914`, `:936-938`, `:1071-1078` |
| `opening` → `closing` | `close()` while opening; aborts closable work after claiming the terminal | `:926-935`, `:1080-1102` |
| `idle` → `closed`, `closed` → `closed` | `close()` short-circuits to a resolved terminal | — | `:920-925`, `:945-949` |
| `failed` (no token, no terminal) → `closing` | `close()` after a failed open | — | `:950-957` |
| any other state | `close()` rejects with `` `cannot close transaction while runtime is ${this.#state}` `` | — | `:958-963` |
| `abort(token)` | `closing` with a published terminal → returns it; `calling` → `cancel(token)`; otherwise → `rollback(token)` | — | `:888-899` |
| `release()` | `return this.close();` | — | `:967-969` |
| terminal is once-only | `#claimTerminal`: `if (this.#terminal !== undefined) return this.#terminal;` | — | `:1049-1060` |
| release is once-only | `#releasePublishedLease`: `if (this.#releaseClaimed) return; this.#releaseClaimed = true;` | — | `:1273-1292` |

Control invocations are frozen constants:
`COMMIT_INVOCATION = Object.freeze({ kind: "commit", functionName: "BAPI_TRANSACTION_COMMIT", parameters: Object.freeze({ WAIT: "X" }) })` (`:306-310`) and
`ROLLBACK_INVOCATION = Object.freeze({ kind: "rollback", functionName: "BAPI_TRANSACTION_ROLLBACK", parameters: Object.freeze({}) })` (`:312-316`).

BAPI `RETURN` inspection: `RETURN` must exist and be a structure or non-empty array of
structures; each row must have `TYPE`; `TYPE` must match `/^(?:|A|E|I|S|W|X)$/u` after
`trim().toUpperCase()`; rows matching `/^(?:A|E|X)$/u` raise `TransactionBapiError`
(`:511-577`). `ID`, `NUMBER`, `MESSAGE` are truncated to 1024 characters (`:511-517`).

### Errors and error codes

| Code/message (verbatim) | Trigger | Recoverable per source? | Citation |
|---|---|---|---|
| `"INVALID_TRANSACTION_TOKEN"` / `"transaction token does not belong to this runtime"` | token is not the identical object `this.#token` | no state change | `:984-995` |
| `"INVALID_TRANSACTION_STATE"` / `` `cannot ${operation} while transaction runtime is ${this.#state}` `` and `` `cannot begin transaction while runtime is ${this.#state}` `` and `"transaction has no pinned lease"` and `` `${operation} requires a pinned transaction lease` `` | wrong state | — | `:997-1007`, `:649-655`, `:1016-1024`, `:1122-1127` |
| `"TRANSACTION_CLOSING"` / `"transaction closed while its lease was opening"`, `"transaction closed while a business call was completing"`, `"transaction closed while opening"`, `"transaction call was canceled"`, `"transaction closed during a business call"`, `` `${label} was aborted` `` | close/cancel racing an operation | the call rejects; the LUW disposition is decided by `#activeCallDisposition` | `:687-692`, `:786-792`, `:932`, `:913`, `:937`, `:1624-1631` |
| `"CONCURRENT_TRANSACTION_OPERATION"` / `"transaction already has an active business call"` | second `call()` | LUW unaffected | `:739-744` |
| `"OPERATION_TIMEOUT"` / `` `${label} exceeded ${this.#operationTimeoutMs}ms` `` | any bounded boundary | **never classified**: the classifier is skipped and the LUW is ambiguous | `:1549-1556`, `:806-821` |
| `"INVALID_CONTROL_RESULT"` / `` `${operation} must return an object` ``, `` `BAPI_TRANSACTION_${…} result must contain RETURN` ``, `` `${operation} RETURN must contain at least one structure` ``, `` `${operation} RETURN[${index}] must be a structure` ``, `` `${operation} RETURN[${index}].TYPE is required` ``, `` `${path} must be a string` ``, `` `${path} must be blank or one of A, E, I, S, W, X` `` | malformed control reply | treated as a control failure → outcome `ambiguous`, eviction | `:498-577`, routed at `:1155-1173` |
| `TransactionBapiError` — `code = "BAPI_REJECTED"`, `outcome = "rejected"`, message `` `BAPI transaction ${operation} rejected: ${first.message}` `` or `` `BAPI transaction ${operation} rejected with message type ${first?.type ?? "?"}` `` | RETURN row of type A, E, or X | outcome `rejected`, lease evicted, never reusable | `:164-184`, `:576` |
| `TransactionTerminalError` — `code = "TRANSACTION_TERMINAL_FAILURE"`, an `AggregateError` carrying `outcome` | a terminal semantic outcome plus one or more cleanup failures; messages: `"ambiguous business call and lease eviction both failed"`, `` `transaction ${operation} completed but its physical lease remains quarantined` ``, `` `transaction ${operation} completed but cleanup failed` ``, `` `transaction ended ${outcome} while its physical lease remains quarantined` ``, `` `transaction ended ${outcome} and lease eviction failed` `` | — | outcome is preserved through cleanup failure | `:186-200`, `:848-852`, `:1209-1214`, `:1229-1234`, `:1250-1255`, `:1265-1270` |
| `AggregateError "transaction begin and lease rollback both failed"`, `"business call and transaction failure classification both failed"` | paired failures | `cause` is the primary | `:711-720`, `:814-820` |
| `TypeError`s: `"transaction runtime options must be an object"`, `"transaction lease adapter must be an object"`, `` `${path} must be a function` ``, `"scheduler requires now and schedule"`, `"scheduler must return a cancelable task"`, `"classifyFailure must be a function"`, `"classifyFailure must return recoverable or ambiguous"`, `"leases.acquire must resolve to an object lease"`, `"transaction call parameters must be an object"`, `"transaction call options must be an object"`, `"transaction call options.notRequested must be an own data property"`, `"transaction call options.notRequested must be an array"`, `` `transaction call options.notRequested[${index}] must be a non-empty string` `` | contract violations | pre-I/O | `:633-634`, `:352-355`, `:335`, `:386-388`, `:402-404`, `:640-642`, `:1041-1045`, `:684-686`, `:463-465`, `:475-477`, `:479-481`, `:482-484`, `:488-493` |
| `RangeError`: `"functionName must contain 1..30 ASCII bytes"`, `` `operationTimeoutMs must be finite and in 1..${MAX_TIMER_MS}` `` | validation | pre-I/O | `:446-457`, `:339-346` |
| `"transaction scheduler clock must be finite and monotonic"`, `"transaction operation deadline must be finite"`, `"transaction scheduler fired early without bounded progress"` | scheduler contract | boundary rejects | `:1478-1487`, `:1500-1501`, `:1583-1589` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| `/** Identity for one SAP LUW. Object identity is part of validation. */` | `:24` |
| `/** Output parameters deactivated for this business invocation. */` | `:37` |
| `* Transport seam for one exclusively pinned physical SAP session. Every` / `* method is called at most once for a given state transition and receives a` / `* finite-deadline AbortSignal.` | `:70-74` |
| `* A once-only ownership handoff. The adapter must retain responsibility for` / `* eventual pool return/destruction once called, even if its returned Promise` / `* rejects, it throws, or its signal expires. The implementation therefore` / `* must claim ownership before any failure can escape. In particular, a` / `* non-reusable handoff must serialize behind any adapter work whose earlier` / `* signal was aborted; it must never blindly call a pool release while that` / `* work still owns the physical resource.` | `:88-96` |
| `* Classifies business-call failures only. Missing, invalid, or throwing` / `* classifiers conservatively make the LUW ambiguous and non-reusable.` / `* Return `recoverable` only when the adapter has proved that the RFC reply` / `* was fully decoded and the physical session remains synchronized. RFC` / `* communication/runtime/cancel failures and ABAP A/E/X message termination` / `* are ambiguous here; a normal BAPI RETURN structure is not an exception.` | `:125-132` |
| `/** A terminal semantic outcome followed by one or more cleanup failures. */` | `:186` |
| `/** Calls which transferred lease ownership to the release adapter. */` | `:222` |
| `/** Raw adapter operations still owning the published physical lease. */` | `:228` |
| `/** Timed-out acquires whose eventual lease/rejection is still being owned. */` | `:230` |
| `* One-shot BAPI LUW coordinator. It never retries an application or control` / `* invocation and never transfers its physical lease before terminal cleanup.` | `:596-598` |
| `// Parameter getters are caller code and may reenter commit/rollback/close.` / `// Revalidate before claiming the single-flight operation.` | `:755-756` |
| `// A hostile classifier may have reentered close(). Its stable/ambiguous` / `// result still decides whether close can issue a rollback safely.` | `:822-823` |
| `/** Explicit abort is a rollback while safe and a cancellation while active. */` | `:887` |
| `* Cancels an active call, waits for its bounded disposition, and never sends` / `* rollback on an ambiguous session.` | `:901-904` |
| `/** Pool-facing release uses the same safe rollback/eviction policy as close. */` | `:966` |
| `// External adapter/classifier/scheduler boundaries may synchronously` / `// reenter and change state; this method deliberately prevents TypeScript` / `// from treating a prior local assignment as an immutable narrowing.` | `:1010-1012` |
| `// Calling release is the ownership-transfer point. Do it before any` / `// scheduler boundary which could fail or reenter, then bound only the` / `// adapter's acknowledgement of that already-completed handoff.` | `:1319-1321` |
| `// Observe the handed-off work before crossing that scheduler boundary.` / `// Otherwise a scheduler setup failure could leave a later adapter` / `// rejection unhandled even though ownership has already transferred.` | `:1323-1325` |
| `// A detached settlement rejects only if its late ownership cleanup did.` | `:1466` |
| `// User-supplied clocks/schedulers may reenter close() synchronously.` / `// Never cross the adapter boundary once that close has aborted this` / `// operation, even if the scheduled deadline itself remains active.` | `:1621-1623` |
| `/** Includes settlement and any onLateFulfilled ownership cleanup. */` (`onDetached`) | `:296` |

### Behaviour facts asserted by tests

All citations `test/transaction-runtime.test.ts`.

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| One lease serves business work and a `WAIT: "X"` commit, then a reusable release | `"pins one lease for business work and WAIT=X commit through reusable cleanup"` | `:294` |
| Commit rejects RETURN A/E/X and neither retries nor resets | `"commit rejects BAPI RETURN A, E, and X and never retries or resets"` | `:389` |
| A malformed `RETURN.TYPE` is ambiguous and can never become success | `"malformed control RETURN.TYPE is ambiguous and can never become success"` | `:429` |
| Blank, S, I, W remain non-fatal | `"blank, S, I, and W control RETURN.TYPE values remain nonfatal"` | `:451` |
| A missing commit `RETURN` is ambiguous and **is not replayed** | `"missing commit RETURN is an ambiguous invalid result and is not replayed"` | `:466` |
| An undefined control rejection stays a rejection | `"an undefined control rejection remains a rejection and cannot become success"` | `:483` |
| A hung commit stays quarantined until its late settlement can be evicted | `"a hung commit stays quarantined until its late settlement can be evicted"` | `:504` |
| A hardened pool lease is evicted only after its late active callback settles | `"a hardened pool lease is evicted only after its late active callback settles"` | `:545` |
| A pool callback beyond every deadline stays explicitly owned until late eviction | `"a pool callback beyond every deadline remains explicitly owned until late eviction"` | `:575` |
| `classifyFailure: () => "recoverable"` leaves `state === "active"`, `adapter.releases.length === 0`; the later rollback runs on the **same lease** and releases `{reusable:true, reason:"rollback", outcome:"rolledBack"}` | `"recoverable business failure keeps the lease for explicit rollback"` | `:614-643` |
| With no classifier, a business failure invokes only `["business"]` — **no rollback is sent** — and releases `{reusable:false, reason:"ambiguous", outcome:"ambiguous"}`; a later `close()` adds no second release | `"unclassified business failure is ambiguous and evicts without rollback"` | `:645-668` |
| `close()` and `abort(token)` on a safe active LUW both send exactly one `BAPI_TRANSACTION_ROLLBACK` with `parameters: {}`, one reset, and a reusable release (`reason` `"close-rollback"` vs `"rollback"`) | `"close and explicit abort rollback a safe active LUW"` | `:670-694` |
| `close()` during a call aborts the call signal immediately but releases nothing; once the reply arrives decoded the call rejects `TRANSACTION_CLOSING` and a rollback **is** sent, with a reusable release | `"close during a call never releases early and rolls back after a stable reply"` | `:696-724` |
| When the adapter rejects after abort, both `close()` and `cancel()` invoke only `["business"]` and release `{reusable:false, reason:"ambiguous", outcome:"ambiguous"}` | `"close or cancel after an aborted call evicts without sending rollback"` | `:726-765` |
| An adapter ignoring abort is bounded twice (call deadline, then convergence deadline); `quarantinedOperations === 1`, no release until the late settlement, then one non-reusable release | `"cancel bounds an adapter which ignores abort before evicting"` | `:767-802` |
| A reentrant `abort()` from an abort listener returns the already-published terminal; `rollbackCalls === 0`, `cancelCalls === 1`, one non-reusable release | `"reentrant abort observes the already-published cancellation terminal"` | `:804-839` |
| Reset failure after a successful commit preserves the committed outcome and evicts | `"reset failure after successful commit preserves committed outcome and evicts"` | `:841` |
| A terminal release failure is visible and cleanup runs exactly once | `"terminal release failure is visible and cleanup remains exactly once"` | `:865` |
| A hung release has a finite signal and preserves the rolled-back outcome | `"hung release has a finite signal and preserves the rolled-back outcome"` | `:889` |
| The release handoff stays observed and abortable when scheduler setup fails | `"release handoff stays observed and abortable when scheduler setup fails"` | `:917` |
| Rollback RETURN errors reject and evict without reset or retry | `"rollback RETURN errors reject and evict without reset or retry"` | `:968` |
| `close()` during commit shares the terminal and does not cancel the commit | `"close during commit shares the terminal promise and does not cancel commit"` | `:984` |
| `close()` during reset shares bounded cleanup and cannot duplicate control calls | `"close during reset shares bounded cleanup and cannot duplicate control calls"` | `:1008` |
| `close()` during opening is bounded and evicts a late lease with its original token | `"close during opening is bounded and evicts a late lease with its original token"` | `:1045` |
| `close()` surfaces a late-acquire eviction failure during its convergence bound | `"close surfaces a late-acquire eviction failure during its convergence bound"` | `:1081` |
| A failed acquire can be retried before close | `"failed acquire can retry before close"` | `:1113` |
| A reentrant failure classifier cannot leak work; close still rolls back safely | `"reentrant failure classifier cannot leak work and close safely rolls back"` | `:1132` |
| Reentrant parameter getters cannot start business work after close | `"reentrant parameter getters cannot start business work after close"` | `:1159` |
| A reentrant scheduler close cannot cross the lease-acquisition boundary | `"reentrant scheduler close cannot cross the lease acquisition boundary"` | `:1183` |
| An acquire abort listener observes the already-published close terminal | `"an acquire abort listener observes the already-published close terminal"` | `:1211` |
| Token, concurrency, parameter and constructor boundaries reject before I/O | `"token, concurrency, parameters, and constructor boundaries reject before I/O"` | `:1239` |
| Adapter and scheduler methods are snapshotted against later replacement | `"snapshots adapter and scheduler methods against later replacement"` | `:1268` |

Facts from `test/modern-transaction-contract.test.ts` (the façade over this runtime;
its source file is outside the read scope, so only the assertions about
transaction/lifecycle semantics are recorded here):

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| A proven pre-wire business failure preserves the lease for a safe close rollback | `"a proven pre-wire business failure preserves the lease for safe close rollback"` | `test/modern-transaction-contract.test.ts:1072` |
| A reset failure after a semantic commit evicts and poisons the logical connection | `"reset failure after semantic commit evicts and poisons the logical connection"` | `test/modern-transaction-contract.test.ts:1102` |
| A, E, X control returns reject, evict the lease, and poison the logical connection | `"A, E, and X control returns reject, evict the lease, and poison the logical connection"` | `test/modern-transaction-contract.test.ts:1233` |
| Missing or malformed control `RETURN` is ambiguous and never reusable | `"missing or malformed control RETURN is ambiguous and never reusable"` | `test/modern-transaction-contract.test.ts:1263` |
| A fatal business failure is **not replayed** and evicts its one physical lease | `"a fatal business failure is not replayed and evicts its one physical lease"` | `test/modern-transaction-contract.test.ts:1313` |
| Close racing a business call aborts once, waits for its tail, and never sends an unsafe rollback | `"close racing a business call aborts once, waits for its tail, and never sends an unsafe rollback"` | `test/modern-transaction-contract.test.ts:1556` |
| Close racing a successful commit joins that terminal operation without adding a rollback | `"close racing a successful commit joins that terminal operation without adding rollback"` | `test/modern-transaction-contract.test.ts:1616` |
| Back-to-back terminal operation and close claim exactly one control call | `"back-to-back terminal operation and close claim exactly one control call"` | `test/modern-transaction-contract.test.ts:1641` |
| Repository metadata failure does not disturb the pinned application lease | `"repository metadata failure does not disturb the pinned application lease"` | `test/modern-transaction-contract.test.ts:1461` |
| Racing executes after commit share one lazy cycle and never acquire two replacement leases | `"racing executes after commit share one lazy cycle and never acquire two replacement leases"` | `test/modern-transaction-contract.test.ts:990` |

---

## The ambiguous-send rule

`test/ambiguous-send-no-replay.test.ts` is named after this rule. Stated precisely, in
three layers:

**1. The wire layer records how much of the message reached the peer.**

`writeOutgoingAppcDataPlan` pre-encodes the whole fragment plan, then writes one NI
payload per iteration. Any throw from `send`, from the synchronous-send barrier
`receive`, or from the barrier identity check is caught once, per
`src/client/direct-cpic-session.ts:565-577`:

```ts
} catch (cause) {
  try {
    await Reflect.apply(closeMethod, transport, []);
  } catch {
    // The original transmission failure remains authoritative.
  }
  throw new DirectCpicOutgoingWriteError(
    index === 0
      ? RfcTransmissionState.Unknown
      : RfcTransmissionState.Partial,
    cause,
  );
}
```

So: failure on the **first** fragment ⇒ `RfcTransmissionState.Unknown`; failure on any
**later** fragment ⇒ `RfcTransmissionState.Partial`
(`src/client/direct-cpic-session.ts:571-576`). The loop `break`s by throwing — there is
no retry, no re-send of the failed fragment, and no continuation to the remaining
fragments. The transport is closed before the error escapes
(`src/client/direct-cpic-session.ts:566-570`). The documented intent, verbatim:
`* Pre-encodes the complete bounded plan, then writes one NI payload at a time.` /
`* A transport failure is terminal and is never retried or replayed.`
(`src/client/direct-cpic-session.ts:471-473`).

**2. The failure layer refuses replay structurally, not by convention.**

`RfcReplayPolicy` has exactly one member — `Never = "never"`
(`src/client/rfc-failure.ts:73-75`). `RfcFailurePolicy.replayPolicy` is typed
`RfcReplayPolicy.Never`, a literal type, not the enum
(`src/client/rfc-failure.ts:149`), as is `RfcFailure.replayPolicy`
(`src/client/rfc-failure.ts:194`). `resolveRfcFailurePolicy` hardcodes
`replayPolicy: RfcReplayPolicy.Never` (`src/client/rfc-failure.ts:410`) and
`createRfcFailure` writes it again (`src/client/rfc-failure.ts:594`). There is no
input, option, or override that can produce any other value, and
`test/rfc-failure.test.ts:454-484` asserts that a caller-supplied weakening of
disposition/recovery is discarded.

`#exchange` propagates the transmission state into the failure: it starts as
`RfcTransmissionState.NotStarted` (`src/client/direct-cpic-session.ts:1612`), becomes
`RfcTransmissionState.Complete` only after the entire plan has been written
(`src/client/direct-cpic-session.ts:1624`), and is overwritten by the write error's own
value when one occurred: `if (cause instanceof DirectCpicOutgoingWriteError) { transmission = cause.transmission; }`
(`src/client/direct-cpic-session.ts:1670-1672`). The resulting failure carries that
transmission (`src/client/direct-cpic-session.ts:1699`), and the generation is
terminated before the error is thrown (`src/client/direct-cpic-session.ts:1711-1712`).

Note the asymmetry the policy encodes: for a `Communication`, `Canceled`, or `Timeout`
category, the disposition is `Close` **regardless of transmission state**
(`src/client/rfc-failure.ts:393-400`), asserted for every transmission value at
`test/rfc-failure.test.ts:159-171`. The transmission field is therefore *evidence
carried to the caller*, not an input to the disposition decision for these categories.

**3. The LUW layer refuses to send a rollback over a session whose last call is of
unknown outcome.**

- A business-call failure is `"ambiguous"` unless a classifier explicitly returns
  `"recoverable"`; a missing classifier returns `"ambiguous"`
  (`src/lifecycle/transaction-runtime.ts:1031-1032`), and a throwing classifier is
  forced to `"ambiguous"` (`src/lifecycle/transaction-runtime.ts:811-820`). An
  `OPERATION_TIMEOUT` bypasses the classifier entirely
  (`src/lifecycle/transaction-runtime.ts:806-821`).
- An ambiguous business failure sets `this.#outcome = "ambiguous"; this.#state = "failed";`
  and goes straight to `#finishReleaseOnly(NO_FAILURE, "ambiguous", "ambiguous")` —
  a release-only path with **no `invoke` of any kind**
  (`src/lifecycle/transaction-runtime.ts:835-840`, `:1237-1271`).
- On close/cancel during a call, `#finishCloseDuringCall` waits for the call to settle
  and then branches on the recorded disposition:
  `if (this.#activeCallDisposition === "stable") { await this.#finishControl("rollback", "close-rollback"); return; }` … otherwise ambiguous release-only
  (`src/lifecycle/transaction-runtime.ts:1104-1114`). `#activeCallDisposition` is set to
  `"stable"` only when a decoded result arrived
  (`src/lifecycle/transaction-runtime.ts:786-792`) or the classifier said
  `"recoverable"` (`src/lifecycle/transaction-runtime.ts:824-829`).
- `COMMENT:` `* Cancels an active call, waits for its bounded disposition, and never sends` / `* rollback on an ambiguous session.` (`src/lifecycle/transaction-runtime.ts:901-904`).
- `COMMENT:` `* One-shot BAPI LUW coordinator. It never retries an application or control` / `* invocation and never transfers its physical lease before terminal cleanup.` (`src/lifecycle/transaction-runtime.ts:596-598`).

**Test evidence.** `test/ambiguous-send-no-replay.test.ts` drives a gated transport
whose `send` never resolves on its own, fails write #0, #1, or #2 of a three-fragment
plan, and asserts for each: the error is a `DirectCpicOutgoingWriteError`, its
`transmission` is `Unknown` (first) or `Partial` (middle, final), its `cause` is the
original error, `transport.closeCount === 1`, and — after an extra event-loop turn —
`transport.writes.length` is still exactly `failAt + 1`, i.e. **no further write was
attempted** (`test/ambiguous-send-no-replay.test.ts:70-113`). Test names:
`"first APPC write failure is terminal and never replayed"` (`:103`),
`"middle APPC write failure is terminal and never replayed"` (`:107`),
`"final APPC write failure is terminal and never replayed"` (`:111`).

**Port requirement.** In Go this must survive as: (a) one `SendPlan` loop that returns
on first error with a `Transmission` value of `Unknown` for fragment 0 and `Partial`
otherwise, closing the conn before returning; (b) a `ReplayPolicy` that is a constant,
not a field a caller can set; (c) a transaction coordinator that reaches
`BAPI_TRANSACTION_ROLLBACK` only from a state proving a decoded reply. `context.Context`
cancellation must not be allowed to shortcut (a): the close-and-classify path must run
even when the context is already done.

---

## Cancellation model

### What `AbortSignal` is wired to

| Wiring | Citation |
|---|---|
| `DirectCpicSessionOptions.signal?: AbortSignal` — a session-lifetime signal captured in the options snapshot | `src/client/direct-cpic-session.ts:145`, `:649` |
| The session signal is passed to the transport factory: `Reflect.apply(transportFactory, undefined, [transportOptions, sessionOptions.signal])` | `src/client/direct-cpic-session.ts:739-742` |
| …and to every handshake send and receive during `open` | `:760-764`, `:787`, `:852`, `:880`, `:899` |
| Per-operation `signal?: AbortSignal` on `logonAndPing`, `ping`, `resetServerContext`, all metadata getters, `invokeClassic`, `invokeClassicWithMetadata`, `exchange` | `:965`, `:1044`, `:1060`, `:1099`, `:1194`, `:1326`, `:1534` |
| Inside `#exchange`, the per-operation signal reaches `writeOutgoingAppcDataPlan(…, signal, this.#operationTimeoutMs)` and `this.#transport.receive({ timeoutMs: this.#operationTimeoutMs, signal })` | `:1617-1623`, `:1628-1631` |
| Inside `writeOutgoingAppcDataPlan`, the signal reaches each `send` and each barrier `receive` | `:547`, `:549-551` |
| Lifecycle: every adapter/observer callback receives a signal minted by `#bounded`; `SessionContextRuntime` registers those controllers in `#abortOnRetire` when `abortOnRetire` is set | `src/lifecycle/session-context-runtime.ts:1186-1197` |
| `retire()` aborts every registered controller with the reason `RUNTIME_RETIRED`, **after** the state gate is closed and every ready entry is claimed | `src/lifecycle/session-context-runtime.ts:794-813` |
| `TransactionRuntime` registers controllers in `#abortOnClose` when `abortOnClose` is set and aborts them via `#abortClosableWork(reason)` | `src/lifecycle/transaction-runtime.ts:1496`, `:1472-1476` |
| The release path uses its **own** `AbortController`, forwarding the bounded signal into it, so ownership transfer happens before any abort can interfere | `src/lifecycle/transaction-runtime.ts:1307-1356` |

### What happens to the socket and the conversation on cancel

| Fact | Citation |
|---|---|
| An aborted NI operation surfaces as `NiTransportError` with `code === "NI_ABORTED"`, mapped to `RfcFailureCategory.Canceled` | `src/client/direct-cpic-session.ts:1678-1679` |
| `Canceled` → group `CommunicationFailure`, code `7`, `codeString "RFC_CANCELED"` | `src/client/rfc-failure.ts:287-291` |
| `Canceled` → disposition `Close`, always | `src/client/rfc-failure.ts:393-400` |
| With `establishedSession === true`, recovery is `Replace` | `src/client/rfc-failure.ts:346-353` |
| Every failure leaving `#exchange` runs `await this.#terminateGeneration();` first | `src/client/direct-cpic-session.ts:1711` |
| `#terminateGeneration` sets `#closed = true`, zero-fills and drops the CPIC session id, clears both metadata caches, and closes the transport (swallowing close errors) | `src/client/direct-cpic-session.ts:1794-1802` |
| Cancel during the write phase also closes the transport inside `writeOutgoingAppcDataPlan` before the error propagates | `src/client/direct-cpic-session.ts:566-570` |
| No `Deallocate` control record is sent on the cancel path — `Deallocate` is only sent from `close()` and only `if (this.#setup.state === "ready")` | `src/client/direct-cpic-session.ts:1815-1832` |

### Can a cancelled call be retried?

No, and the session is unusable afterwards:

- `state` becomes `"closed"` because `#closed === true`
  (`src/client/direct-cpic-session.ts:958-961`, `:1796`).
- Any subsequent `#exchange` throws `"direct CPIC session is closed"`
  (`src/client/direct-cpic-session.ts:1562`).
- Any subsequent authenticated operation throws
  `"direct CPIC session must be authenticated before …"` because `#cpicSessionId` was
  cleared (`src/client/direct-cpic-session.ts:1797-1798`, guard at `:1045-1047`).
- The failure itself carries `replayPolicy: RfcReplayPolicy.Never`
  (`src/client/rfc-failure.ts:594`) and `recoveryAction: Replace` for an established
  session — the correct response is a **new physical generation**, not a retry.
- `TEST-NAME:` `"terminally closes transport, timeout, and abort generations"` asserts
  `failure.category === RfcFailureCategory.Canceled`,
  `recoveryAction === RfcRecoveryAction.Replace`, and `session.state === "closed"` for
  the abort case (`test/direct-cpic-session-errors.test.ts:379-399`).

In the LUW layer, `cancel(token)` is terminal for the transaction: it moves the runtime
to `"closing"` and claims the single terminal
(`src/lifecycle/transaction-runtime.ts:905-914`, `:1071-1078`); the LUW ends `ambiguous`
with a non-reusable release unless the call had already produced a decoded reply
(`src/lifecycle/transaction-runtime.ts:1104-1114`).

### Go equivalent, point by point

| Upstream mechanism | Go replacement |
|---|---|
| `DirectCpicSessionOptions.signal` captured at open and threaded into connect + handshake | A session-scoped `context.Context` stored on the session struct, or passed to `Open(ctx, …)`; use it for `net.Dialer.DialContext` and for the handshake I/O. |
| Per-operation `signal?: AbortSignal` on every public method | `ctx context.Context` as the first parameter of every exported method. Do not store per-call contexts on the struct. |
| `transport.receive({ timeoutMs: this.#operationTimeoutMs, signal })` | `conn.SetReadDeadline(time.Now().Add(operationTimeout))` plus a watchdog goroutine that calls `conn.SetDeadline(time.Now())` when `ctx.Done()` fires, so a blocked `Read` returns immediately. Distinguish the two: `ctx.Err() == context.Canceled` → `Canceled`; `os.ErrDeadlineExceeded` with no ctx error → `Timeout`. That distinction is load-bearing — the two categories share a group but not a `code` (`7` vs `8`, `src/client/rfc-failure.ts:100-101`). |
| `NI_ABORTED` / `NI_RECEIVE_TIMEOUT` / `NI_WRITE_TIMEOUT` / `NI_CONNECT_TIMEOUT` / `NI_PROTOCOL_ERROR` string codes driving the category switch | Sentinel error values in the NI/transport package, matched with `errors.Is`, feeding the identical switch. The "no recognized transport error at all → `MalformedProtocol`" default (`src/client/direct-cpic-session.ts:1686-1687`) must be preserved. |
| `await this.#terminateGeneration()` on every exchange failure | A `terminate()` method with `sync.Once` semantics: set closed, zero the session id (`crypto/subtle` not needed; a simple loop over the byte slice matches `#cpicSessionId?.fill(0)`), clear caches, `conn.Close()` ignoring the error. |
| `#busy` single-flight + `#compoundOperationOwner` symbol | A `sync.Mutex` for the single-flight gate is *not* equivalent: upstream **rejects** rather than blocks (`RFC_CONCURRENT_CALL`, `src/client/direct-cpic-session.ts:1544-1553`). Use a `bool`/`atomic.Bool` guarded by a mutex and return the error; use a comparable owner token (a pointer or an incrementing id) for the compound-operation re-entry allowance. |
| `#abortOnRetire` / `#abortClosableWork` broadcast | One `context.CancelFunc` per runtime, cancelled at the top of retire/close; every bounded operation derives its context from it via `context.WithTimeout`. |
| `#bounded`'s late-fulfillment / quarantine hooks | A goroutine that owns the in-flight adapter call and a channel for its result; on deadline the coordinator stops waiting but keeps the channel and evicts the lease when the value arrives. `errgroup`/`sync.WaitGroup` at close time replaces `#completeRetirement`'s `Promise.allSettled` loop. |
| Release-path private `AbortController` (`src/lifecycle/transaction-runtime.ts:1307`) | A separate `context.WithCancel` for the release call, not derived from the closing context, so eviction is never cancelled by the close that triggered it. `SessionContextRuntime` states the same rule explicitly for the reusable-vs-eviction split (`{ abortOnRetire: reusable }`, `src/lifecycle/session-context-runtime.ts:1120-1122`). |

---

## Async plumbing that should NOT be ported

Line counts are of the cited ranges, to size the reduction.

| Code | Lines | Citation | Go replacement |
|---|---|---|---|
| `SessionContextRuntime#bounded` — AbortController wiring, deadline arming, generation counters, early-rearm defence, late-fulfilment hook | 149 | `src/lifecycle/session-context-runtime.ts:1186-1334` | `ctx, cancel := context.WithTimeout(parent, d)` plus a `select` on the worker's result channel. The `MAX_EARLY_TIMER_REARMS = 64` / monotonic-clock defences (`:1245-1300`, `:1166-1175`) exist because `setTimeout` and a caller-supplied `now()` can lie; Go's `time.After`/`context` deadlines cannot, so they vanish. |
| `TransactionRuntime#bounded` — same, plus `mustQuarantine` detachment | 174 | `src/lifecycle/transaction-runtime.ts:1489-1662` | Same, plus one goroutine + result channel for the detached case. |
| Quarantine bookkeeping: `#trackAcquireQuarantine`, `#settleAcquireQuarantineState`, `#trackOperationQuarantine`, `#prepareReleaseAfterQuarantine`, `#startLateRelease` | 82 | `src/lifecycle/transaction-runtime.ts:1389-1470` | A single `chan result` per in-flight adapter call plus a `WaitGroup`; "quarantine" is just "the goroutine still holds the lease". |
| `#completeRetirement` — `Promise.allSettled` over openings, terminals, then a `while` loop draining `#lateCleanups` | 35 | `src/lifecycle/session-context-runtime.ts:1336-1370` | `errgroup.Group.Wait()` / `sync.WaitGroup.Wait()`; the drain loop disappears because a `WaitGroup` counts work registered before `Wait` returns. |
| `#trackLateCleanup` | 7 | `src/lifecycle/session-context-runtime.ts:1158-1164` | `wg.Add(1)` in the goroutine that owns the late lease. |
| `completion()` deferred-promise factories | 13 + 9 | `src/lifecycle/session-context-runtime.ts:400-412`, `src/lifecycle/transaction-runtime.ts:410-418` | `chan struct{}` closed once, or `sync.Once` + a channel. |
| Terminal-promise plumbing: `#newTerminal` / `#settleTerminal`; `#claimTerminal` / `#settleTerminal` | 16 + 21 | `src/lifecycle/session-context-runtime.ts:908-923`, `src/lifecycle/transaction-runtime.ts:1049-1069` | One `sync.Once` guarding a `terminalErr` field and a `done chan struct{}`. **Keep the once-only semantics** — see the next section. |
| Adapter/scheduler binding and re-binding: `bindLeaseAdapter`, `bindScheduler`, `bindScheduledTask`, `callable`; `bindAdapter`, `bindScheduler`, `bindTask`, `callable` | 64 + 6 and 59 + 4 | `src/lifecycle/session-context-runtime.ts:304-374`, `src/lifecycle/transaction-runtime.ts:334-408` | Nothing. These exist to snapshot JavaScript methods against later replacement of a property on a caller object; a Go interface value cannot be swapped after it is stored. (`TEST-NAME:` `"snapshots adapter and scheduler methods against later replacement"`, `test/transaction-runtime.test.ts:1268`; `"snapshots caller methods and ignores poisoned bind properties"`, `test/session-context-runtime.test.ts:706`.) |
| `defaultScheduler` with `handle.unref()` | 13 + 8 | `src/lifecycle/session-context-runtime.ts:261-273`, `src/lifecycle/transaction-runtime.ts:318-325` | Nothing. `unref()` exists so a pending timer does not keep the Node event loop alive; Go has no equivalent hazard. The injectable-scheduler seam is worth keeping **only** as a `clock` interface for deterministic tests. |
| `#readClock` monotonicity guards | 10 + 10 | `src/lifecycle/session-context-runtime.ts:1166-1175`, `src/lifecycle/transaction-runtime.ts:1478-1487` | Nothing. `time.Now()` monotonic readings and `context` deadlines are trustworthy. |
| Monitor counter machinery (interface + mutable shape + `emptyMonitor`) | 96 and 81 | `src/lifecycle/session-context-runtime.ts:138-171,221-250,414-445`, `src/lifecycle/transaction-runtime.ts:202-234,262-284,420-444` | Keep the *counters that tests assert on* (`releases`, `evictions`, `ambiguousFailures`, `boundaryTimeouts`, `quarantinedOperations`); a Go port can express them as `expvar`/atomic fields. Roughly half are pure introspection of Promise bookkeeping that will not exist. |
| `snapshotDirectClassicNameSet`, `snapshotDirectClassicInput`, `snapshotDirectClassicCaller` — deep-copying caller objects before any await | 92 | `src/client/direct-cpic-session.ts:296-380` | Partly droppable: Go maps/slices still alias, so the *copy* stays, but the `Symbol.iterator`/`Reflect.apply` hostile-iterator defence (`:304-315`) has no Go analogue. Same argument as `src/protocol/bytes.ts` in `../provenance.md`. |
| `snapshotSessionOptions` — freezing every option before validation or I/O | 27 | `src/client/direct-cpic-session.ts:631-657` | A plain struct copy; Go option structs cannot have accessors. The stated reason (`:637-639`) is exactly the JS-only hazard. |
| `snapshotParameterStateSet` bounded-iterator wrapper | 32 | `src/client/classic-invocation.ts:1099-1130` | `map[string]struct{}` copy; the bound check on entry count stays (it is a protocol bound), the iterator-hostility wrapper goes. |
| `async`/`Promise` return types on methods that never await: `#decodeApplicationResult`, `#decodeRegularResponse`'s `try` block | — | `src/client/direct-cpic-session.ts:1773-1792`, `:1726-1747` | Plain functions returning `(T, error)`; only the `terminateGeneration` call inside them is I/O. |

Estimated shrink for `src/lifecycle/` alone from the rows above: roughly 750–800 lines
of the 3034 are Promise/AbortSignal scaffolding with a direct Go primitive replacement.

---

## Protocol/state semantics that MUST be preserved

These look like plumbing and are not.

| Rule | Why it is semantics | Citation |
|---|---|---|
| First-fragment failure = `Unknown`, later = `Partial`; never retry either | This is the ambiguous-send rule; see that section | `src/client/direct-cpic-session.ts:571-576` |
| The transport is closed **before** the write error escapes | Prevents a half-sent APPC message from being followed by anything else on the same conversation | `src/client/direct-cpic-session.ts:566-570` |
| `assertNoQueuedFrames()` before send **and** after a complete reply | A coalesced second NI frame must never be attributed to the wrong request; the comment says the check "also retires the transport" | `src/client/direct-cpic-session.ts:1616`, `:1661`, comments `:1614-1615`, `:1658-1660` |
| `assertDirectCpicResponseIdentity`: 8-byte conversation id equality, `communicationIndex === 0`, matching `connectionIndex` | Reply/conversation binding, done without sequence correlation | `src/client/direct-cpic-session.ts:412-425` |
| On `"normal-deallocation"`, decode the buffered envelope once, then terminate the generation before returning it | CPI-C is in Reset and the conversation id is invalid | `src/client/direct-cpic-session.ts:1652-1656` |
| `terminalTransport` (a closed setup machine) overrides an otherwise-reusable or successful envelope: origin is forced to `Appc`, and a *successful* envelope becomes a `CM_DEALLOCATED_NORMAL` communication failure | Transport truth beats application truth | `src/client/direct-cpic-session.ts:1725`, `:1748-1769`; asserted by `"normal deallocation overrides reusable and successful RFC envelopes"`, `test/direct-cpic-session-errors.test.ts:303-339` |
| The exact reusable tuple for a declared exception: `Sap` + `EnvelopeDecode` + `Complete` + `establishedSession` | The single case where a connection survives an error; every deviation is `UnknownClose` | `src/client/rfc-failure.ts:366-374`; asserted at `test/rfc-failure.test.ts:523-556` and end-to-end at `test/direct-cpic-session-errors.test.ts:209-240` |
| `Communication`/`Logon`/`AbapRuntime`/`AbapMessage`/`Canceled`/`Timeout` are `Close` for **all** transmission states | A caller must not "optimize" a `NotStarted` communication failure into a reusable connection | `src/client/rfc-failure.ts:393-400`; asserted at `test/rfc-failure.test.ts:159-171` |
| `replayPolicy` is a literal type, unsettable | Structural, not conventional, prevention of replay | `src/client/rfc-failure.ts:73-75,149,194,410,594` |
| Caller-supplied `disposition`/`recoveryAction` are ignored | `COMMENT:` `Call sites cannot weaken connection disposition or grant replay permission.` | `src/client/rfc-failure.ts:356-357`; asserted at `test/rfc-failure.test.ts:471-483` |
| `key`/`message`/`abap`/`cause`/`toJSON` are non-enumerable and `toJSON` returns only the diagnostic | Prevents remote ABAP text, variables, and stacks from reaching logs | `src/client/rfc-failure.ts:599-633`; asserted at `test/rfc-failure.test.ts:365-422` |
| Preflight-then-encode with a byte-length equality check per value and per row | Defends the wire length against a value that changes between the two passes; the resulting `RangeError` is a hard failure, not a re-preflight | `src/client/classic-invocation.ts:1868-1872`, `:1945-1949` |
| `captureClassicRfcInvocation` snapshots input and activation **once**, used for both request encoding and response validation | Otherwise the reply validator could be run against a different activation set than the request | `src/client/classic-invocation.ts:1140-1143`, used at `src/client/direct-cpic-session.ts:1199-1203`, `:1308-1315` |
| The activation lattice (deactivate > explicit value > optional-inactive > mandatory-initial) | Decides which fields exist on the wire at all | `src/client/classic-invocation.ts:234-254`, `:257-277` |
| Strict recursive-xRFC codec first; broad codec only after both its resolver and its validator succeed; otherwise rethrow the strict error | An invalid graph must not be reinterpreted by the looser codec | `src/client/classic-invocation.ts:376-421` |
| Structure repository is keyed by table type, not line type | `test/classic-xrfc-invocation.test.ts:156-168` | `src/client/classic-invocation.ts:1296-1306` |
| `structuredExactFieldByteLength`: accept a reply row that ends exactly at the last validated field, and nothing else | Real kernels omit trailing alignment; this tolerance is deliberately minimal | `src/client/classic-invocation.ts:728-745`, applied at `:2184-2213` |
| `DirectCpicPreWireError` marks "no application byte reached the wire" | It is the *only* signal that lets a caller keep the session after a failed call preparation | `src/client/direct-cpic-session.ts:160-177` |
| `resetServerContext` must be followed, in the same compound operation, by a full `RFC_PING` re-handshake before the connection is handed back | `COMMENT:` `Reset clears SAP's RFC session-header state.` | `src/client/direct-cpic-session.ts:1078-1089` |
| The optimized-metadata fallback key sets and the `disposition === Reusable` precondition | Determines when a missing/forbidden `RFC_METADATA_GET` may be silently downgraded; the wrong set turns an authorization error into a silent capability loss | `src/client/direct-cpic-session.ts:195-219`, `:1234-1242` |
| Gateway acceptance: `returnCode === 0`, `appcHeaderVersion === 6`, `ExtendedInitOptions` bit, `CodePage` bit **and** `codePage === "4103"` | Wire compatibility gate | `src/client/direct-cpic-session.ts:790-808` |
| `#assertExchangeAvailable` **rejects** a concurrent call rather than queueing it | A second in-flight call on one CPIC conversation is a protocol error, not a scheduling problem | `src/client/direct-cpic-session.ts:1538-1555` |
| Ownership transitions happen synchronously before any external callback is entered | `COMMENT:` `Ownership transitions happen synchronously before an external Promise or callback is entered.` — in Go, take the mutex, mutate state, release, *then* call the adapter | `src/lifecycle/session-context-runtime.ts:449-455`; `retire()` at `:794`, `:799-813` |
| `entry.releaseClaimed` / `#releaseClaimed` once-only release | Double release of a pooled connection is a correctness bug, not a performance one | `src/lifecycle/session-context-runtime.ts:1072-1087`, `src/lifecycle/transaction-runtime.ts:1273-1292` |
| Last-instant `effectiveReusable = reusable && state === "open" && !signal.aborted`, with the reason rewritten to `"runtime-retire"` when denied | A retired generation must never be handed back as reusable | `src/lifecycle/session-context-runtime.ts:1099-1111`; asserted by `"retirement reentered by the scheduler cannot publish a reusable release"`, `test/session-context-runtime.test.ts:1127` |
| Eviction continues during retirement while a reusable hand-off is aborted: `{ abortOnRetire: reusable }` | The signature of the whole cancellation design | `src/lifecycle/session-context-runtime.ts:1120-1122` |
| Fatal ordering: zero references → remove from ownership → release → notify → delete | The owner must not see a context that still owns a lease | `src/lifecycle/session-context-runtime.ts:984-1011`; asserted by `"fatal failure removes ownership, evicts once, then notifies the owner"`, `test/session-context-runtime.test.ts:505-549` |
| An `OPERATION_TIMEOUT` is fatal/ambiguous without consulting the classifier | A timed-out call has unknown outcome by definition | `src/lifecycle/session-context-runtime.ts:671-673`, `src/lifecycle/transaction-runtime.ts:806-810` |
| A missing or throwing classifier means `"ambiguous"` | Fail-safe default | `src/lifecycle/transaction-runtime.ts:1031-1032`, `:811-820` |
| `call()` re-validates state **after** reading caller parameters | `COMMENT:` `Parameter getters are caller code and may reenter commit/rollback/close.` In Go the getters vanish, but the *terminal-claimed* re-check before starting work must stay for the concurrent-close case | `src/lifecycle/transaction-runtime.ts:755-757` |
| Release is called before crossing any scheduler boundary, and its returned promise is observed immediately | `COMMENT:` `Calling release is the ownership-transfer point.` In Go: hand the lease to the pool, *then* wait for acknowledgement with a timeout | `src/lifecycle/transaction-runtime.ts:1319-1333` |
| BAPI `RETURN` grammar: required `RETURN`, ≥1 structure, required `TYPE`, `TYPE ∈ {blank,A,E,I,S,W,X}` after trim/upper, A/E/X ⇒ rejection | Wire-level contract with `BAPI_TRANSACTION_COMMIT`/`ROLLBACK` | `src/lifecycle/transaction-runtime.ts:511-577` |
| `BAPI_TRANSACTION_COMMIT` always carries `WAIT: "X"` | `src/lifecycle/transaction-runtime.ts:306-310` | asserted by `"pins one lease for business work and WAIT=X commit through reusable cleanup"`, `test/transaction-runtime.test.ts:294` |
| Terminal promise is claimed once and shared by `close`/`abort`/`cancel`/reentrant callers | Prevents a second control call | `src/lifecycle/transaction-runtime.ts:1049-1060`, `:894-896`; asserted by `"back-to-back terminal operation and close claim exactly one control call"`, `test/modern-transaction-contract.test.ts:1641` |

---

## Open questions for the porter

1. **`CpicLogonError` and `CpicCallError` are dead exports.** They are declared at
   `src/client/direct-cpic-session.ts:249-267` and constructed only in
   `test/direct-cpic-session.test.ts:544-545`. No `src/` code throws them; logon
   rejection produces an `RfcCoreError` instead (`:1019-1030`). Port them only if a
   consumer outside the read scope needs them.

2. **`DirectCpicPreWireError`'s consumer is outside the read scope.** A `grep` (not a
   read) shows `src/destination/direct-destination-owner.ts:2887` containing
   `if (failure instanceof DirectCpicPreWireError) return "recoverable";`. The Go
   equivalent of "pre-wire failure ⇒ the LUW stays recoverable" therefore lives in a
   layer this inventory did not cover; confirm it before designing the Go error type.

3. **`AsyncLocalStorage` is not used in either directory.** `grep -rl AsyncLocalStorage src/`
   matches only `src/metadata/repository-runtime.ts`, `src/destination/configuration-generation.ts`,
   `src/destination/runtime.ts`, `src/destination/direct-destination-owner.ts`. The
   porting plan's framing (`docs/porting-plan.md`) is right about the layer group but
   the ALS work is entirely in `src/destination/` and `src/metadata/`, not here.

4. **`src/client/rfc-errors.ts` is already marked "not ported" in `../provenance.md`,
   but `rfcFailureToPublicError` is the only projection from `RfcFailure` to a public
   error type.** If the Go API exposes errors at all, something must replace it — and
   whatever replaces it should carry `disposition`/`recoveryAction`, which the current
   public shape drops entirely (`src/client/rfc-errors.ts:310-316`, `:364-370`).

5. **`AppcClientSetupStateMachine` state names are consumed but not defined here.**
   `src/client/direct-cpic-session.ts` reads `setup.state === "ready"` (`:540`, `:1815`)
   and `=== "closed"` (`:959`, `:1725`) and calls `sent`, `received`,
   `responseComplete`, `pushTerminalDeallocation`. The full state set lives in
   `src/protocol/appc.ts`, which is out of scope; the `#exchange` and `close()` logic
   cannot be ported without it.

6. **Single-flight is reject-not-queue, but only inside one session.** Upstream returns
   `RFC_CONCURRENT_CALL` (`:1544-1553`). A Go port that fronts `DirectCpicSession` with
   a mutex would silently change this into serialization. Confirm whether any upstream
   caller relies on the rejection (the pool layer, `src/pool/`, was not read).

7. **`operationTimeoutMs` defaults differ between layers**: `30_000` in
   `DirectCpicSession.open` (`:757`) and in `writeOutgoingAppcDataPlan`'s
   `barrierTimeoutMs = 30_000` (`:483`), while both lifecycle runtimes require an
   explicit finite `operationTimeoutMs` in `1..2_147_483_647` and have no default
   (`src/lifecycle/session-context-runtime.ts:376-383`,
   `src/lifecycle/transaction-runtime.ts:339-346`). Decide whether the Go port keeps
   the asymmetry.

8. **Monitor surface.** 30 counters in `SessionContextRuntimeMonitor` (`:138-171`) and
   27 in `TransactionRuntimeMonitor` (`:202-234`). Several count Promise-bookkeeping
   states that will not exist in Go (`quarantinedAcquires`, `opening`, `fatalCleaning`).
   Which of them are part of an observable contract, and which were test affordances,
   is not decidable from the source alone.
