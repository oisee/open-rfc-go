# Surface inventory: src/metadata/
> Mechanical inventory of open-rfc @ commit 847036d, generated as porting input. Every claim cites path:line. See ../provenance.md.

Scope: the eight files in `src/metadata/` and four upstream tests
(`test/recursive-metadata.test.ts`, `test/modern-recursive-metadata.test.ts`,
`test/metadata-repository-runtime.test.ts`, `test/rfc-metadata-get.test.ts`).
Nothing outside that scope was read; every claim below about an imported symbol
whose definition lives elsewhere is marked "not stated in source".

Citation convention matches `../provenance.md`: paths are relative to the
open-rfc repository root.

---

## src/metadata/immutable-map.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `ImmutableMetadataMap` | class | `export class ImmutableMetadataMap<K, V> implements ReadonlyMap<K, V>` | `src/metadata/immutable-map.ts:9` |
| `ImmutableMetadataMap` ctor | constructor | `constructor(entries: readonly (readonly [K, V])[])` | `src/metadata/immutable-map.ts:12` |
| `.size` | accessor | `get size(): number { return this.#values.size; }` | `src/metadata/immutable-map.ts:17` |
| `.get` | method | `get(key: K): V \| undefined { return this.#values.get(key); }` | `src/metadata/immutable-map.ts:18` |
| `.has` | method | `has(key: K): boolean { return this.#values.has(key); }` | `src/metadata/immutable-map.ts:19` |
| `.entries` | method | `entries(): MapIterator<[K, V]> { return this.#values.entries(); }` | `src/metadata/immutable-map.ts:20` |
| `.keys` | method | `keys(): MapIterator<K> { return this.#values.keys(); }` | `src/metadata/immutable-map.ts:21` |
| `.values` | method | `values(): MapIterator<V> { return this.#values.values(); }` | `src/metadata/immutable-map.ts:22` |
| `[Symbol.iterator]` | method | `[Symbol.iterator](): MapIterator<[K, V]>` | `src/metadata/immutable-map.ts:23` |
| `.forEach` | method | `forEach(callbackfn: (value: V, key: K, map: ReadonlyMap<K, V>) => void, thisArg?: unknown,): void` | `src/metadata/immutable-map.ts:26-29` |
| `isImmutableMetadataMap` | function | `export function isImmutableMetadataMap(value: object,): value is ImmutableMetadataMap<unknown, unknown>` | `src/metadata/immutable-map.ts:39-41` |
| `immutableMetadataMapEntries` | function | `export function immutableMetadataMapEntries(value: ImmutableMetadataMap<unknown, unknown>,): readonly (readonly [unknown, unknown])[]` | `src/metadata/immutable-map.ts:47-49` |

No mutating method (`set`, `delete`, `clear`) is declared on the class
(`src/metadata/immutable-map.ts:9-34`).

### Constants, type codes, and enumerations

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| (module-level statement) | `Object.freeze(ImmutableMetadataMap.prototype);` | executed at module load | `src/metadata/immutable-map.ts:36` |

No other constants in this file.

### Errors

This file throws nothing (`src/metadata/immutable-map.ts:1-53`).

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "A small immutable Map implementation used inside metadata snapshots." | `src/metadata/immutable-map.ts:2` |
| "`Object.freeze(new Map())` is still mutable through Map.prototype.set(). This wrapper exposes no mutation operations and keeps its backing Map in a private field. The repository snapshot validator recognizes only genuine instances of this class and traverses their captured entries explicitly." | `src/metadata/immutable-map.ts:4-7` |
| "Internal trust predicate used by the bounded repository snapshot walk." | `src/metadata/immutable-map.ts:38` |
| "Capture entries without exposing the private mutable backing collection." | `src/metadata/immutable-map.ts:46` |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| A subclass `class ForgedMetadataMap<K, V> extends ImmutableMetadataMap<K, V> {}` inside a snapshot is rejected with `/recursively frozen/u`; a direct `new ImmutableMetadataMap([...])` is accepted | `"accepts only the exact immutable metadata map implementation in snapshots"` | `test/metadata-repository-runtime.test.ts:1021`, subclass at `:1025`, rejection at `:1046-1049`, acceptance at `:1050-1052` |
| The accepted map has no `set`: `assert.equal(typeof (accepted.values as { readonly set?: unknown }).set, "undefined")` | `"accepts only the exact immutable metadata map implementation in snapshots"` | `test/metadata-repository-runtime.test.ts:1053-1054` |
| Calling `.set(...)` on a returned map throws `/set is not a function/u` | `"normalizes complete timestamp batches without retaining backend text"` | `test/rfc-metadata-get.test.ts:205-209` |

### Go mapping notes

- The class exists solely because JS `Object.freeze` does not freeze a `Map`'s
  contents (`src/metadata/immutable-map.ts:4`). Go has no analogue of that
  hazard: an unexported `map` field behind a value/struct with no mutating
  method is sufficient.
- `isImmutableMetadataMap` demands prototype identity (`value instanceof
  ImmutableMetadataMap && Object.getPrototypeOf(value) === ImmutableMetadataMap.prototype`,
  `src/metadata/immutable-map.ts:42-43`) — i.e. subclasses are rejected. In Go
  this maps to a concrete unexported struct type, not an interface.

---

## src/metadata/ddif-fieldinfo.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `buildDdIfFieldInfoGetRequest` | function | `export function buildDdIfFieldInfoGetRequest(structureName: string, language = "E",): Buffer` | `src/metadata/ddif-fieldinfo.ts:46-49` |
| `decodeDdIfDfiesRow` | function | `export function decodeDdIfDfiesRow(value: Uint8Array): DecodedDfiesField` | `src/metadata/ddif-fieldinfo.ts:111` |
| `decodeDdIfFieldInfoGetResult` | function | `export function decodeDdIfFieldInfoGetResult(structureName: string, fields: readonly CpicField[],): RfcStructureDefinition` | `src/metadata/ddif-fieldinfo.ts:201-204` |

`DecodedDfiesField` itself is **not** exported: `interface DecodedDfiesField
extends RfcStructureField { readonly componentType: string; }`
(`src/metadata/ddif-fieldinfo.ts:106-108`).

### Constants, type codes, and enumerations

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `DFIES_MINIMUM_UNICODE_ROW_LENGTH` | `1_074` | minimum DFIES row byte length; also the prefix length copied | `src/metadata/ddif-fieldinfo.ts:20`, use at `:117`, `:130-132` |
| `X030L_MINIMUM_UNICODE_LENGTH` | `249` | minimum `X030L_WA` scalar byte length | `src/metadata/ddif-fieldinfo.ts:21`, use at `:164` |
| `MAX_DDIC_STRUCTURE_FIELDS` | `9_999` | maximum DDIC field count | `src/metadata/ddif-fieldinfo.ts:22`, uses at `:187`, `:223` |
| requested outputs | `requestedOutputs: ["DDOBJTYPE", "X030L_WA", "DFIES_TAB"]` | outputs asked of `DDIF_FIELDINFO_GET` | `src/metadata/ddif-fieldinfo.ts:54` |
| import `TABNAME` | `{ name: "TABNAME", value: encodeAbapChar(name, 30) }` | 30-char structure name | `src/metadata/ddif-fieldinfo.ts:56` |
| import `LANGU` | `{ name: "LANGU", value: encodeAbapChar(langu, 1) }` | 1-char language | `src/metadata/ddif-fieldinfo.ts:57` |
| import `ALL_TYPES` | `{ name: "ALL_TYPES", value: encodeAbapChar("X", 1) }` | fixed `"X"` | `src/metadata/ddif-fieldinfo.ts:58` |
| import `UCLEN` | `{ name: "UCLEN", value: Buffer.of(2) }` | "This resolver is explicitly Unicode and therefore always requests two-byte geometry." | `src/metadata/ddif-fieldinfo.ts:59-61` |
| DFIES `TABNAME` | `fieldText(row, 0, 60, 30)` | offset 0, 60 bytes, 30 chars | `src/metadata/ddif-fieldinfo.ts:134` |
| DFIES `FIELDNAME` | `fieldText(row, 60, 60, 30)` | offset 60, 60 bytes, 30 chars | `src/metadata/ddif-fieldinfo.ts:135` |
| DFIES `POSITION` | `numc(row, 122, 8, "DFIES.POSITION")` | offset 122, 8 bytes, NUMC | `src/metadata/ddif-fieldinfo.ts:136` |
| DFIES `OFFSET` | `numc(row, 130, 12, "DFIES.OFFSET")` | offset 130, 12 bytes, NUMC | `src/metadata/ddif-fieldinfo.ts:137` |
| DFIES `INTLEN` | `numc(row, 334, 12, "DFIES.INTLEN")` | offset 334, 12 bytes, NUMC | `src/metadata/ddif-fieldinfo.ts:138` |
| DFIES `DECIMALS` | `numc(row, 358, 12, "DFIES.DECIMALS")` | offset 358, 12 bytes, NUMC | `src/metadata/ddif-fieldinfo.ts:139` |
| DFIES `exid` | `fieldText(row, 378, 2, 1)` | offset 378, 2 bytes, 1 char (comment at `:143` calls this "INTTYPE") | `src/metadata/ddif-fieldinfo.ts:140`, `:143` |
| DFIES `componentType` | `fieldText(row, 1_072, 2, 1)` | offset 1072, 2 bytes, 1 char | `src/metadata/ddif-fieldinfo.ts:141` |
| X030L field count | `value.readUInt16BE(162)` | offset 162, uint16 **big-endian** | `src/metadata/ddif-fieldinfo.ts:177` |
| X030L byte length | `value.readUInt32BE(164)` | offset 164, uint32 **big-endian** | `src/metadata/ddif-fieldinfo.ts:178` |
| X030L table type | `fieldText(value, 172, 2, 1)` | offset 172, 1 char | `src/metadata/ddif-fieldinfo.ts:179` |
| X030L Unicode width | `value.readUInt8(248)` | offset 248, uint8; must equal `2` | `src/metadata/ddif-fieldinfo.ts:180-186` |
| Rejected DDIC object kinds | `objectKind === "DTEL" \|\| objectKind === "TTYP"` | rejected as unsupported | `src/metadata/ddif-fieldinfo.ts:211-215` |
| Rejected table type marker | `tableType === "L"` | "a table/vector type" | `src/metadata/ddif-fieldinfo.ts:192-196` |
| Accepted component types | `field.componentType !== "" && field.componentType !== "E"` rejects | `""` and `"E"` accepted | `src/metadata/ddif-fieldinfo.ts:255-260` |

Anything the DDIC type codes `DTEL`, `TTYP`, `L`, `E` mean beyond the words
quoted above is **not stated in source**.

### Errors

All are thrown; none carry a typed code except the `RangeError`/`TypeError`
built-ins.

| Message (verbatim) | Trigger | Citation |
|---|---|---|
| `` `${path} must contain 1..30 characters without controls` `` (`RangeError`) | non-string, empty, >30 chars, or contains `/[\u0000-\u001f\u007f]/u` | `src/metadata/ddif-fieldinfo.ts:31-33`, predicate `:25-29` |
| `"DDIF language must be one printable ASCII character"` (`RangeError`) | language not matching `/^[\x20-\x7e]$/u` | `src/metadata/ddif-fieldinfo.ts:39-41` |
| `` `DDIF_FIELDINFO_GET response lacks scalar ${name}` `` | required scalar missing | `src/metadata/ddif-fieldinfo.ts:72` |
| `` `${path} must contain NUMC digits` `` | NUMC field not `/^\d+$/u` | `src/metadata/ddif-fieldinfo.ts:97` |
| `` `${path} exceeds the safe integer range` `` (`RangeError`) | parsed NUMC not a safe integer | `src/metadata/ddif-fieldinfo.ts:101` |
| `"DFIES row expects Uint8Array bytes"` (`TypeError`) | argument not `Uint8Array` | `src/metadata/ddif-fieldinfo.ts:113` |
| `` `Unicode DFIES row must contain at least ${DFIES_MINIMUM_UNICODE_ROW_LENGTH} even bytes; received ${byteLength}` `` (`RangeError`) | `byteLength < 1074` or odd | `src/metadata/ddif-fieldinfo.ts:117-124` |
| `"DFIES row contains an empty table, field, or INTTYPE"` | empty tableName/fieldName, or `exid.length !== 1` | `src/metadata/ddif-fieldinfo.ts:142-144` |
| `"DFIES row contains an invalid position"` | `position < 1` | `src/metadata/ddif-fieldinfo.ts:145-147` |
| `` `DDIF_FIELDINFO_GET X030L_WA must contain at least ${X030L_MINIMUM_UNICODE_LENGTH} bytes` `` (`RangeError`) | short X030L | `src/metadata/ddif-fieldinfo.ts:164-168` |
| `` `DDIF_FIELDINFO_GET X030L_WA belongs to ${returnedName}; expected ${structureName}` `` | name mismatch | `src/metadata/ddif-fieldinfo.ts:171-176` |
| `` `DDIF_FIELDINFO_GET selected unsupported Unicode width ${unicodeCharacterBytes}` `` | byte 248 ≠ 2 | `src/metadata/ddif-fieldinfo.ts:181-186` |
| `` `DDIF_FIELDINFO_GET field count exceeds ${MAX_DDIC_STRUCTURE_FIELDS}` `` (`RangeError`) | X030L advertised count > 9999 | `src/metadata/ddif-fieldinfo.ts:187-191` |
| `"DDIF_FIELDINFO_GET returned a table/vector type; a flat structure was required"` | `tableType === "L"` | `src/metadata/ddif-fieldinfo.ts:192-196` |
| `"DDIF_FIELDINFO_GET returned an initial DDOBJTYPE"` | empty `DDOBJTYPE` | `src/metadata/ddif-fieldinfo.ts:208-210` |
| `` `DDIF_FIELDINFO_GET returned unsupported DDIC object kind ${objectKind}` `` | `DTEL` or `TTYP` | `src/metadata/ddif-fieldinfo.ts:211-215` |
| `"DDIF_FIELDINFO_GET response lacks DFIES_TAB"` | table absent | `src/metadata/ddif-fieldinfo.ts:219-222` |
| `` `DDIF_FIELDINFO_GET field count exceeds ${MAX_DDIC_STRUCTURE_FIELDS}` `` (`RangeError`) | actual `DFIES_TAB` rows > 9999 | `src/metadata/ddif-fieldinfo.ts:223-227` |
| `` `DDIF_FIELDINFO_GET X030L_WA advertises ${geometry.fieldCount} fields; DFIES_TAB contains ${table.rows.length}` `` | count disagreement | `src/metadata/ddif-fieldinfo.ts:228-233` |
| `` `DFIES ${field.fieldName} belongs to ${field.tableName}; expected ${name}` `` | foreign row | `src/metadata/ddif-fieldinfo.ts:240-244` |
| `` `DFIES ${field.fieldName} has position ${field.position}; expected ${index + 1}` `` | non-sequential position | `src/metadata/ddif-fieldinfo.ts:245-249` |
| `` `DFIES ${name}.${field.fieldName} has unsupported component type ${field.componentType \|\| "<initial>"}` `` | componentType not `""` or `"E"` | `src/metadata/ddif-fieldinfo.ts:255-260` |
| `` `DFIES contains duplicate field ${field.fieldName}` `` | duplicate name | `src/metadata/ddif-fieldinfo.ts:261-263` |
| `` `DFIES ${field.fieldName} overlaps its preceding field` `` | `field.offset < previousEnd` | `src/metadata/ddif-fieldinfo.ts:264-266` |
| `` `DFIES ${field.fieldName} ends at ${end} beyond structure length ${byteLength}` `` | `end` unsafe or `> byteLength` | `src/metadata/ddif-fieldinfo.ts:267-272` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Build the Note 460089 classic DDIC lookup without prior dynamic metadata." | `src/metadata/ddif-fieldinfo.ts:45` |
| "DDIF's default follows backend/runtime context. This resolver is explicitly Unicode and therefore always requests two-byte geometry." | `src/metadata/ddif-fieldinfo.ts:59-60` |
| "Decode the stable DFIES prefix used by both 7.50 and 7.58." | `src/metadata/ddif-fieldinfo.ts:110` |
| "Later releases append fields to DFIES. Only retain the stable prefix that this decoder consumes, so a maximum-size CPIC row cannot trigger a second maximum-size copy." | `src/metadata/ddif-fieldinfo.ts:125-127` |
| "SAP Note 1691982's DDIF consumer treats both \"E\" and the initial value as elementary. The initial form is emitted for structure components declared directly with a built-in DDIC type. Composite markers remain fail-closed, and validateClassicStructureCodec below still validates each elementary field's type, length, decimals, offsets, and geometry." | `src/metadata/ddif-fieldinfo.ts:250-254` |
| "Normalize the classic DDIF response into the invocation codec descriptor." | `src/metadata/ddif-fieldinfo.ts:200` |

### Wire facts asserted by tests

None of the four in-scope tests import this file (`test/recursive-metadata.test.ts:1-6`,
`test/modern-recursive-metadata.test.ts:1-11`,
`test/metadata-repository-runtime.test.ts:1-16`,
`test/rfc-metadata-get.test.ts:1-17`). Whether coverage exists elsewhere is out
of scope.

### Go mapping notes

- `decodeDdIfDfiesRow` copies the first 1074 bytes (`Buffer.from(... .subarray(0,
  DFIES_MINIMUM_UNICODE_ROW_LENGTH))`, `src/metadata/ddif-fieldinfo.ts:128-133`)
  before decoding — the Go port must copy, not alias, for the same reason
  recorded in `../provenance.md` for `src/protocol/bytes.ts`.
- X030L integers are big-endian (`readUInt16BE`, `readUInt32BE`,
  `src/metadata/ddif-fieldinfo.ts:177-178`) while RFC_FIELDS integers are
  little-endian (`readInt32LE`, `src/metadata/rfc-structure-definition.ts:65-68`).
  These are different structures; do not unify the endianness.
- The final value goes through `validateClassicStructureCodec(...)`
  (`src/metadata/ddif-fieldinfo.ts:285`), imported from
  `../values/classic-structure.js` (`src/metadata/ddif-fieldinfo.ts:18`) — out of
  scope, semantics not stated in source.

---

## src/metadata/recursive-metadata.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RecursiveMetadataLimits` | interface | `export interface RecursiveMetadataLimits` — fields `maxRows`, `maxNodes`, `maxEdges`, `maxDepth`, `maxProperties`, `maxBytes`, all `readonly ... : number` | `src/metadata/recursive-metadata.ts:5-12` |
| `RecursiveMetadataOptions` | interface | `export interface RecursiveMetadataOptions { readonly limits?: Partial<RecursiveMetadataLimits>; readonly rootTypeNames?: readonly string[]; }` | `src/metadata/recursive-metadata.ts:14-17` |
| `RecursiveMetadataReference` | type | `export type RecursiveMetadataReference = \| Readonly<{ kind: "scalar"; internalType: string; }> \| Readonly<{ kind: "structure" \| "table"; targetType: string; cyclic: boolean; }>` | `src/metadata/recursive-metadata.ts:19-28` |
| `RecursiveMetadataField` | interface | fields `name`, `position`, `componentType`, `associatedType`, `dataType`, `internalType`, `description`, `decimals`, `nucOffset`, `ucOffset`, `nucLength`, `ucLength`, `reference` | `src/metadata/recursive-metadata.ts:30-44` |
| `RecursiveMetadataTypeNode` | interface | `name`, `kind: "structure" \| "table" \| "scalar"`, `nucLength`, `ucLength`, `timestamp`, `fields` | `src/metadata/recursive-metadata.ts:46-53` |
| `RecursiveMetadataFunctionIdentity` | interface | `readonly name: string; readonly remoteBasxmlSupported: boolean; readonly generationToken: string;` | `src/metadata/recursive-metadata.ts:55-59` |
| `RecursiveMetadataParameter` | interface | `functionName`, `name`, `parameterClass: "I" \| "E" \| "C" \| "T" \| "X"`, `position`, `associatedType`, `fieldPath`, `internalType`, `internalLength`, `decimals`, `defaultValue`, `parameterText`, `optional`, `reference` | `src/metadata/recursive-metadata.ts:61-75` |
| `RecursiveMetadataParameterReference` | type | `export type RecursiveMetadataParameterReference = \| RecursiveMetadataReference \| Readonly<{ kind: "table"; scalarLine: Readonly<{ internalType: string }>; cyclic: false; }> \| Readonly<{ kind: "exception" }>` | `src/metadata/recursive-metadata.ts:77-84` |
| `RecursiveMetadataCycle` | interface | `readonly id: string; readonly typeNames: readonly string[];` | `src/metadata/recursive-metadata.ts:86-89` |
| `RecursiveMetadataStatistics` | interface | `rowCount`, `nodeCount`, `edgeCount`, `propertyCount`, `byteCount`, `maximumDepth` | `src/metadata/recursive-metadata.ts:91-98` |
| `RecursiveMetadataGraph` | interface | `readonly version: 1; readonly functionIdentity: RecursiveMetadataFunctionIdentity \| undefined; readonly nodes: ReadonlyMap<string, RecursiveMetadataTypeNode>; readonly parameters: readonly RecursiveMetadataParameter[]; readonly rootTypeNames: readonly string[]; readonly cycles: readonly RecursiveMetadataCycle[]; readonly limits: RecursiveMetadataLimits; readonly statistics: RecursiveMetadataStatistics;` | `src/metadata/recursive-metadata.ts:100-109` |
| `isNormalizedRecursiveMetadataGraph` | function | `export function isNormalizedRecursiveMetadataGraph(value: unknown,): value is RecursiveMetadataGraph` | `src/metadata/recursive-metadata.ts:114-116` |
| `RecursiveMetadataError` | class | `export class RecursiveMetadataError extends Error { readonly code: string; readonly path: string; constructor(code: string, path: string) }` | `src/metadata/recursive-metadata.ts:121-131` |
| `normalizeRecursiveMetadataGraph` | function | `export function normalizeRecursiveMetadataGraph(value: unknown, optionsValue?: RecursiveMetadataOptions,): RecursiveMetadataGraph` | `src/metadata/recursive-metadata.ts:1481-1484` |

`RecursiveMetadataError` message format: `` super(`recursive metadata rejected: ${code} at ${path}`) `` and
`this.name = "RecursiveMetadataError"` (`src/metadata/recursive-metadata.ts:126-128`).

### Constants, type codes, and enumerations

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `DEFAULT_LIMITS` | `Object.freeze({ maxRows: 20_000, maxNodes: 4_096, maxEdges: 20_000, maxDepth: 64, maxProperties: 400_000, maxBytes: 8 * 1024 * 1024, })` | defaults when no options passed | `src/metadata/recursive-metadata.ts:133-140` |
| `ABSOLUTE_LIMITS` | `Object.freeze({ maxRows: 100_000, maxNodes: 20_000, maxEdges: 100_000, maxDepth: 256, maxProperties: 2_000_000, maxBytes: 32 * 1024 * 1024, })` | per-key ceiling a caller may configure | `src/metadata/recursive-metadata.ts:142-149`, enforced `:493` |
| `LIMIT_KEYS` | `["maxRows","maxNodes","maxEdges","maxDepth","maxProperties","maxBytes"]` | allowed keys of `options.limits` | `src/metadata/recursive-metadata.ts:151-158` |
| `INPUT_KEYS` | `["FUNCTIONNAMES","DATATYPESCONT","INDIRECTTYPES","PARAMETERS"]` | the only allowed top-level input keys | `src/metadata/recursive-metadata.ts:160-165` |
| required top-level keys | `["DATATYPESCONT", "INDIRECTTYPES"]` | required; `FUNCTIONNAMES`/`PARAMETERS` optional | `src/metadata/recursive-metadata.ts:1496` |
| `FUNCTION_ROW_KEYS` | `["FUNCTIONNAME","BASXML_SUPPORTED","UDAT","UTIME"]` | exact key set of a `FUNCTIONNAMES` row (both allowed and required) | `src/metadata/recursive-metadata.ts:167-172`, use `:610-615` |
| `TYPE_ROW_KEYS` | `["TYPENAME","FIELDNAME","COMPTYPE","FIELDTYPE","DATATYPE","TABLENGTH","TABLENGTH_UC","DESCRIPTION","DECIMALS","INTTYPE","OFFSET","OFFSET_UC","INTLEN","INTLEN_UC","TIMESTAMP"]` | exact key set of a `DATATYPESCONT` row | `src/metadata/recursive-metadata.ts:174-190`, use `:568-573` |
| `INDIRECT_ROW_KEYS` | `["TABNAME","FIELDNAME","FIELDTYPE"]` | exact key set of an `INDIRECTTYPES` row | `src/metadata/recursive-metadata.ts:192-196` |
| `PARAMETER_ROW_KEYS` | `["FUNCNAME","PARAMCLASS","PARAMETER","TABNAME","FIELDNAME","EXID","POSITION","OFFSET","INTLENGTH","DECIMALS","DEFAULT","PARAMTEXT","OPTIONAL"]` | exact key set of a `PARAMETERS` row | `src/metadata/recursive-metadata.ts:198-212` |
| descriptor-edge internal types | `if (internalType === "u" \|\| internalType === "v" \|\| internalType === "h")` | "RFC_METADATA_GET uses `u` for flat structures and `v` for deep structures… a structured table type is represented as one anonymous `v` row pointing at its line structure." `h` ⇒ `kind: "table"`, `u`/`v` ⇒ `kind: "structure"` | `src/metadata/recursive-metadata.ts:745-756` |
| parameter class charset | `/^[IECXT]$/u` | valid `PARAMCLASS` | `src/metadata/recursive-metadata.ts:693-695` |
| exception class | `if (row.parameterClass === "X") { reference = Object.freeze({ kind: "exception" as const }); }` | `X` rows become exception references | `src/metadata/recursive-metadata.ts:1031-1032` |
| generation token format | `` generationToken: `function:${date}:${time}` `` | from `UDAT`/`UTIME` | `src/metadata/recursive-metadata.ts:629` |
| cycle id format | `` id: `cycle:${result.length}` `` | sequential | `src/metadata/recursive-metadata.ts:1467` |
| composite key separator | `` `${row.tableName}\u0000${row.fieldPath}` `` | NUL-joined indirect key | `src/metadata/recursive-metadata.ts:893`, `:956`, `:1045`, `:1156`, `:1214` |
| parameter dedup key | `` `${row.functionName}\u0000${row.parameterName}` `` | NUL-joined | `src/metadata/recursive-metadata.ts:1026` |
| text field maxima | `TIMESTAMP` 14 & `/^\d{14}$/u`; `COMPTYPE` 1; `DATATYPE` 8; `DESCRIPTION` 60 (non-ASCII allowed); `INTTYPE` 1; metadata names 30 ASCII; `UDAT` 8 & `/^\d{8}$/u`; `UTIME` 6 & `/^\d{6}$/u`; `PARAMCLASS` 1; `EXID` 1; `DEFAULT` 21 (non-ASCII allowed); `PARAMTEXT` 79 (non-ASCII allowed) | per-field length/charset bounds | `:575-576`, `:580`, `:582`, `:585`, `:587`, `:475`, `:617-620`, `:684-692`, `:696`, `:701-709`, `:710-718` |

`version` is the literal `1` (`src/metadata/recursive-metadata.ts:101`, set at `:1584`).

### Errors

Every rejection is `RecursiveMetadataError` built by
`function reject(code: string, path: string): never`
(`src/metadata/recursive-metadata.ts:286-288`); the message is always
`` `recursive metadata rejected: ${code} at ${path}` ``
(`src/metadata/recursive-metadata.ts:126`). The `path` never contains a value,
only a structural path — see the test assertion below.

| Code (verbatim) | Trigger | Citation |
|---|---|---|
| `"PROXY_INPUT"` | `nodeUtilTypes.isProxy(value)` on record/array input | `:295`, `:332`, `:358`, `:529-530` |
| `"HOSTILE_INPUT"` | `Reflect.ownKeys`, `getOwnPropertyDescriptor`, or `getPrototypeOf` throws | `:299-300`, `:314-315`, `:336-337`, `:362-363` |
| `"SYMBOL_PROPERTY"` | any own key is not a string | `:304` |
| `"MISSING_PROPERTY"` | descriptor absent; also missing required key | `:317`, `:351` |
| `"ACCESSOR_PROPERTY"` | own property has no `value` slot | `:318`, `:487`, `:526` |
| `"INVALID_RECORD"` | not an object / null / array | `:329-330` |
| `"INVALID_PROTOTYPE"` | prototype not `Object.prototype`/`null` (records), not `Array.prototype` (arrays) | `:339-340`, `:365`, `:532-533` |
| `"UNKNOWN_PROPERTY"` | key outside the allowed set; non-index own key on an array | `:346`, `:372-373`, `:539-541` |
| `"INVALID_ARRAY"` | value not an array | `:357`, `:530` |
| `"ROW_LIMIT"` | `value.length > remainingRows`, or `budget.rows + count` unsafe/over `maxRows` | `:366-367`, `:383-388` |
| `"PROPERTY_LIMIT"` | `budget.properties + count` unsafe/over `maxProperties` | `:391-396` |
| `"BYTE_LIMIT"` | `budget.bytes + count` unsafe/over `maxBytes` | `:399-404` |
| `"INVALID_TEXT"` | wrong type, over max length, empty when disallowed, control chars, or non-ASCII when `ascii` | `:421-429`, `:552-554`, `:697-699` |
| `"INVALID_INTEGER"` | not number/digit-string, or not a non-negative safe integer | `:445-454` |
| `"INVALID_FLAG"` | flag not `""` or `"X"` | `:464` |
| `"INVALID_LIMIT"` | configured limit not a safe integer in `0..ABSOLUTE_LIMITS[key]` | `:489-496` |
| `"DUPLICATE_ROOT"` | duplicate in `options.rootTypeNames` | `:555` |
| `"INVALID_TIMESTAMP"` | `TIMESTAMP` not `/^\d{14}$/u` | `:576` |
| `"MULTIPLE_FUNCTION_IDENTITIES"` | `FUNCTIONNAMES.length !== 1` when non-empty | `:606-608` |
| `"INVALID_DATE"` / `"INVALID_TIME"` | `UDAT` not 8 digits / `UTIME` not 6 digits | `:619-620` |
| `"MULTIPLE_FUNCTIONS"` | parameter rows name more than one function | `:639` |
| `"MISSING_FUNCTION_IDENTITY"` | `FUNCTIONNAMES` key present but empty while parameters exist | `:640-642` |
| `"FOREIGN_FUNCTION_REFERENCE"` | identity name not among parameter function names | `:643-645` |
| `"INVALID_PARAMETER_CLASS"` | `PARAMCLASS` outside `[IECXT]` | `:693-695` |
| `"MISSING_ASSOCIATED_TYPE"` | `u`/`v`/`h` row with empty `FIELDTYPE`; or parameter with empty `TABNAME` where required | `:750`, `:1043`, `:1057`, `:1153` |
| `"INVALID_GEOMETRY"` | `offset + length` unsafe; overlap or overrun of the effective total length | `:762`, `:833-840` |
| `"NONCONTIGUOUS_TYPE"` | a `TYPENAME` group reappears after being closed | `:777-779` |
| `"NODE_LIMIT"` | `grouped.length > limits.maxNodes`; `roots.length > limits.maxNodes` | `:785`, `:535` |
| `"INVALID_TABLE_SHAPE"` | a group containing a blank `FIELDNAME` has more than one row | `:794-796` |
| `"INCONSISTENT_TOTAL_LENGTH"` | `TABLENGTH`/`TABLENGTH_UC` differ within a group | `:823-828` |
| `"INCONSISTENT_TIMESTAMP"` | `TIMESTAMP` differs within a group | `:829` |
| `"DUPLICATE_FIELD"` | repeated `FIELDNAME` within a group | `:830` |
| `"EDGE_LIMIT"` | `fieldEdges > limits.maxEdges` or `edgeCount > limits.maxEdges` | `:866`, `:1113`, `:1150`, `:1191` |
| `"FOREIGN_TYPE_REFERENCE"` | reference target not among nodes | `:940`, `:983-984`, `:1050`, `:1060`, `:1163`, `:1178` |
| `"REFERENCE_KIND_MISMATCH"` | node kind ≠ reference kind; scalar-table line not a single anonymous scalar; `h` target not a table; structure target not a structure | `:941-943`, `:1084`, `:1088`, `:1103`, `:1128`, `:1144`, `:1171`, `:1180`, `:1183` |
| `"INVALID_INDIRECT_PATH"` | `INDIRECTTYPES` `FIELDNAME` without `-` | `:953-955` |
| `"DUPLICATE_INDIRECT_TYPE"` | duplicate `(TABNAME, FIELDNAME)` | `:957` |
| `"INVALID_FIELD_PATH"` | empty path segment, or a non-structure intermediate segment | `:976`, `:981` |
| `"FOREIGN_FIELD_REFERENCE"` | path segment not a field of the current node | `:978` |
| `"DUPLICATE_PARAMETER"` | repeated `(FUNCNAME, PARAMETER)` | `:1027` |
| `"MISSING_INDIRECT_TYPE"` | dashed field path with no matching `INDIRECTTYPES` row | `:1047`, `:1158` |
| `"FOREIGN_INDIRECT_TYPE"` | an `INDIRECTTYPES` row that no parameter consumed | `:1215` |
| `"DEPTH_LIMIT"` | `maximumDepth > maxDepth`, path `"metadata-graph"` | `:1342` |
| `"FOREIGN_ROOT"` | `options.rootTypeNames` entry not a node name | `:1546` |
| `"FOREIGN_TYPE_NODE"` | some node is unreachable from the roots, path `"DATATYPESCONT"` | `:1574` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Internal trust predicate for graphs produced by the bounded normalizer." | `src/metadata/recursive-metadata.ts:113` |
| "RFC_METADATA_GET legitimately emits zero and duplicate positions. Keep the captured row order as the stable tie-break rather than renumbering." | `src/metadata/recursive-metadata.ts:720-721` |
| "RFC_METADATA_GET uses `u` for flat structures and `v` for deep structures. In particular, a structured table type is represented as one anonymous `v` row pointing at its line structure. Both are descriptor edges; whether the enclosing anonymous node is a table is decided below." | `src/metadata/recursive-metadata.ts:745-748` |
| "RFC_METADATA_GET describes a top-level scalar type with a single anonymous row whose TABLENGTH values are zero while INTLEN carries its real wire width. Structured-table wrapper rows instead carry their aggregate TABLENGTH and a zero INTLEN. Normalize only the former shape; named structure fields remain subject to the strict aggregate bounds." | `src/metadata/recursive-metadata.ts:806-810` |
| "Normalize the optimized RFC_METADATA_GET type closure into a bounded, immutable identity graph. The graph deliberately stores references by type name: cycles stay explicit and no recursive object-freezing walk is needed." | `src/metadata/recursive-metadata.ts:1476-1480` |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| A `u` field yields `reference: { kind: "structure", targetType: "Z_CHILD", cyclic: false }`; `graph.statistics.maximumDepth === 2`; `functionIdentity` equals `{ name: "Z_GRAPH_TEST", remoteBasxmlSupported: false, generationToken: "function:20260716:010203" }`; `graph`, node, `fields`, and `reference` are all `Object.isFrozen`; `"set" in graph.nodes === false` | `"normalizes an immutable nested structure graph with dual geometry"` | `test/recursive-metadata.test.ts:107`, assertions `:136-167` |
| A zero-parameter RFM keeps its identity and `graph.statistics.rowCount === 1` | `"retains a bounded function identity for a zero-parameter RFM"` | `test/recursive-metadata.test.ts:170`, `:189` |
| `POSITION` values `0` and duplicates are accepted and source order is preserved; `-1`, `"-1"`, `"invalid"`, `1.5` all give `INVALID_INTEGER` | `"accepts zero parameter positions and preserves source order for ties"` | `test/recursive-metadata.test.ts:194`, `:201-214` |
| Two `FUNCTIONNAMES` rows ⇒ `MULTIPLE_FUNCTION_IDENTITIES`; identity naming a different function than the parameters ⇒ `FOREIGN_FUNCTION_REFERENCE`; two different `FUNCNAME`s in `PARAMETERS` ⇒ `MULTIPLE_FUNCTIONS` | `"rejects multiple identities and foreign or mixed parameter functions"` | `test/recursive-metadata.test.ts:217-240` |
| An `h` field targets a node whose `kind === "table"`; that table node's single field has `name === ""` and a `structure` reference; two paths to the same line type produce the same `targetType` | `"preserves table-in-structure, structure-in-table, nested XSTRING, and shared identity"` | `test/recursive-metadata.test.ts:242`, `:289-311` |
| An anonymous `v` row becomes `kind: "table"` with a `structure` line reference, and its parameter reference is `{ kind: "table", targetType: "Z_TT_DEEP_ROW", cyclic: false }` | `"accepts RFC_METADATA_GET deep-structure v rows used by structured tables"` | `test/recursive-metadata.test.ts:314`, `:356-366` |
| With `TABLENGTH`/`TABLENGTH_UC` zero, node lengths come from `INTLEN`/`INTLEN_UC` (255 / 510) and `kind === "scalar"` | `"uses INTLEN geometry for anonymous top-level scalar descriptors"` | `test/recursive-metadata.test.ts:376`, `:394-401` |
| A dashed `FIELDNAME` resolves through `INDIRECTTYPES`; `associatedType` stays the raw `TABNAME` (`"Z_OUTER"`) and `fieldPath` stays `"INNER-VALUE"` | `"resolves bounded indirect function field paths without losing associated types"` | `test/recursive-metadata.test.ts:404`, `:431-437` |
| A `PARAMCLASS "T"` with a non-composite `EXID` produces `{ kind: "table", scalarLine: { internalType: "C" }, cyclic: false }`, `rootTypeNames === ["SYST"]`, `statistics.edgeCount === 1`, and the `scalarLine` is frozen | `"represents classic scalar TABLES parameters as table edges to scalar leaves"` | `test/recursive-metadata.test.ts:440`, `:460-469` |
| An elementary anonymous node stays `kind: "scalar"` when reached indirectly and directly | `"keeps indirectly associated elementary descriptors as scalar identity nodes"` | `test/recursive-metadata.test.ts:472`, `:499-512` |
| An anonymous scalar node reached by an incoming `h` field edge is promoted to `kind: "table"` | `"promotes an elementary descriptor to a named scalar table by incoming table edge"` | `test/recursive-metadata.test.ts:515`, `:533-537` |
| A 2-node cycle yields `cycles === [{ id: "cycle:0", typeNames: ["Z_A", "Z_B"] }]`, both references `cyclic: true`, and `maximumDepth === 1` | `"represents descriptor cycles explicitly and keeps shared nodes finite"` | `test/recursive-metadata.test.ts:540`, `:560-569` |
| An input with only empty `DATATYPESCONT`/`INDIRECTTYPES` is accepted; every empty collection is frozen | `"accepts an empty initial graph and freezes every empty collection"` | `test/recursive-metadata.test.ts:572-585` |
| Overlapping offsets ⇒ `INVALID_GEOMETRY`; repeated field ⇒ `DUPLICATE_FIELD`; missing target ⇒ `FOREIGN_TYPE_REFERENCE`; unreachable node ⇒ `FOREIGN_TYPE_NODE` | `"rejects corrupt geometry, duplicates, foreign nodes, and bad targets"` | `test/recursive-metadata.test.ts:587-633` |
| Unconsumed indirect row ⇒ `FOREIGN_INDIRECT_TYPE`; repeated indirect row ⇒ `DUPLICATE_INDIRECT_TYPE` | `"rejects foreign and duplicate indirect declarations"` | `test/recursive-metadata.test.ts:635-657` |
| Each of `maxRows:0`→`ROW_LIMIT`, `maxNodes:0`→`NODE_LIMIT`, `maxEdges:0`→`EDGE_LIMIT`, `maxDepth:2`→`DEPTH_LIMIT`, `maxProperties:1`→`PROPERTY_LIMIT`, `maxBytes:1`→`BYTE_LIMIT`, `maxRows:100_001`→`INVALID_LIMIT` | `"enforces every configurable resource limit"` | `test/recursive-metadata.test.ts:659-722` |
| A getter on `DATATYPESCONT` gives `ACCESSOR_PROPERTY` **and is never invoked** (`assert.equal(getterCalled, false)`); a `Proxy` gives `PROXY_INPUT`; `new Array(1)` gives `MISSING_PROPERTY`; hostile text and hostile key names give `INVALID_TEXT` / `UNKNOWN_PROPERTY` with the secret **absent from the message** | `"rejects getters, proxies, sparse arrays, and hostile text without leaking it"` | `test/recursive-metadata.test.ts:724-772`, non-leak assertions `:757-761`, `:768-771` |
| The graph produced by `normalizeRecursiveMetadataGraph` survives object-spread reconstruction, i.e. downstream code can build `Object.freeze({ ...valid, nodes: new Map(...) })` and it is still typed `RecursiveMetadataGraph` (the projection layer, not this file, rejects it) | `"rejects cycles, foreign references, and mixed function identities"` | `test/modern-recursive-metadata.test.ts:484`, `:509-516` |
| Downstream projection of the same graph reads NUC geometry (`length: 12`, offsets `0`/`4`) and `associatedType` values straight from the graph | `"projects nested structures, tables, and XSTRING with authoritative NUC geometry"` | `test/modern-recursive-metadata.test.ts:253`, `:272-321` |

### Go mapping notes

- Nothing in this file recurses. Every traversal uses an explicit stack or index
  loop: iterative DFS with `stack: { node: number; next: number }[]`
  (`:1251-1265`), the reverse-reachability stack (`:1275-1286`), the component
  reach stack (`:1313-1319`), the Kahn queue (`:1327-1341`), the final
  reachability stack (`:1565-1573`), and the segment loop in `resolveFieldPath`
  (`:974-987`). A Go port can keep these as slices; no goroutine or stack-depth
  concern exists.
- Cycles are represented by name, never by pointer:
  "The graph deliberately stores references by type name: cycles stay explicit
  and no recursive object-freezing walk is needed." (`:1478-1480`). In Go this
  means `map[string]TypeNode` with string `TargetType` — do **not** convert to
  `*TypeNode` pointers.
- Strongly-connected components are computed with Kosaraju's two passes
  (forward finish order `:1246-1266`, reverse assignment `:1268-1289`), then a
  component is cyclic iff it has >1 member or a self-edge (`:1291-1296`).
- `budget.bytes` counts `Buffer.byteLength(...)` of strings and a flat `8` per
  numeric field (`:430`, `:444`, `:446`); a Go port must use UTF-8 byte length
  to keep the same limit behaviour.
- The trust registry is a `WeakSet<object>` (`:111`) keyed on the graph object
  identity, populated only at `:1600`. In Go this is a struct field or an
  unexported constructor, not a side table.

---

## src/metadata/recursive-parameter-index.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RecursiveMetadataParameterIndex` | interface | `export interface RecursiveMetadataParameterIndex { readonly functionName: string; readonly parameterCount: number; }` | `src/metadata/recursive-parameter-index.ts:15-18` |
| `RecursiveMetadataParameterIndexDiagnostics` | interface | `readonly broadClassificationNodeVisits: number; readonly broadClassificationFieldVisits: number; readonly broadValidationNodeVisits: number; readonly broadValidationFieldVisits: number; readonly strictDescriptorNodeVisits: number;` | `src/metadata/recursive-parameter-index.ts:36-42` |
| `createRecursiveMetadataParameterIndex` | function | `export function createRecursiveMetadataParameterIndex(graph: RecursiveMetadataGraph,): RecursiveMetadataParameterIndex` | `src/metadata/recursive-parameter-index.ts:54-56` |
| `recursiveMetadataParameterFromIndex` | function | `export function recursiveMetadataParameterFromIndex(graph: RecursiveMetadataGraph, index: RecursiveMetadataParameterIndex, name: string,): RecursiveMetadataParameter \| undefined` | `src/metadata/recursive-parameter-index.ts:176-180` |
| `recursiveMetadataParameterIndexCacheGet` | function | `export function recursiveMetadataParameterIndexCacheGet<T>(graph: RecursiveMetadataGraph, index: RecursiveMetadataParameterIndex, namespace: string, key: string,): T \| undefined` | `src/metadata/recursive-parameter-index.ts:185-190` |
| `recursiveMetadataParameterIndexCacheSet` | function | `export function recursiveMetadataParameterIndexCacheSet<T>(graph: RecursiveMetadataGraph, index: RecursiveMetadataParameterIndex, namespace: string, key: string, value: T,): T` | `src/metadata/recursive-parameter-index.ts:197-203` |
| `recordRecursiveMetadataParameterIndexWork` | function | `export function recordRecursiveMetadataParameterIndexWork(graph: RecursiveMetadataGraph, index: RecursiveMetadataParameterIndex \| undefined, kind: keyof RecursiveMetadataParameterIndexWork,): void` | `src/metadata/recursive-parameter-index.ts:215-219` |
| `recursiveMetadataParameterIndexDiagnostics` | function | `export function recursiveMetadataParameterIndexDiagnostics(graph: RecursiveMetadataGraph, index: RecursiveMetadataParameterIndex,): RecursiveMetadataParameterIndexDiagnostics` | `src/metadata/recursive-parameter-index.ts:226-229` |

### Constants, type codes, and enumerations

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `ABSOLUTE_MAX_PARAMETER_COUNT` | `100_000` | ceiling on the graph's declared `limits.maxRows` | `src/metadata/recursive-parameter-index.ts:44`, use `:96`, `:99` |
| `INDEX_STATE` | `new WeakMap<object, RecursiveMetadataParameterIndexState>()` | module-private side table holding the real state | `src/metadata/recursive-parameter-index.ts:45-48` |
| work counters (initial) | `{ broadClassificationNodeVisits: 0, broadClassificationFieldVisits: 0, broadValidationNodeVisits: 0, broadValidationFieldVisits: 0, strictDescriptorNodeVisits: 0 }` | traversal diagnostics | `src/metadata/recursive-parameter-index.ts:151-157` |

### Errors

| Message (verbatim) | Trigger | Citation |
|---|---|---|
| `"recursive xRFC graph must be a version-1 metadata graph"` (`TypeError`) | not an object, null, or `graph.version !== 1` | `src/metadata/recursive-parameter-index.ts:57-61` |
| `"recursive xRFC metadata lacks a function identity"` | missing/invalid `functionIdentity.name` | `src/metadata/recursive-parameter-index.ts:63-70` |
| `"recursive xRFC metadata parameters must be an array"` (`TypeError`) | `parameters` not an array | `src/metadata/recursive-parameter-index.ts:71-74` |
| `"recursive xRFC metadata parameters must not be a proxy"` (`TypeError`) | `nodeUtilTypes.isProxy(source)` | `src/metadata/recursive-parameter-index.ts:75-79` |
| `"recursive xRFC metadata parameter length must be intrinsic"` (`TypeError`) | `length` is an accessor, missing, non-safe-integer, or negative | `src/metadata/recursive-parameter-index.ts:80-90` |
| `` `recursive xRFC graph maxRows is outside 0..${ABSOLUTE_MAX_PARAMETER_COUNT}` `` (`RangeError`) | `graph.limits?.maxRows` outside `0..100_000` | `src/metadata/recursive-parameter-index.ts:92-101` |
| `` `recursive xRFC graph exceeds its row budget ${declaredMaximum}` `` (`RangeError`) | `parameterCount > declaredMaximum` | `src/metadata/recursive-parameter-index.ts:102-106` |
| `` `recursive xRFC metadata parameter ${position} must be an own data property` `` (`TypeError`) | index descriptor missing or accessor-backed | `src/metadata/recursive-parameter-index.ts:114-118` |
| `` `recursive xRFC metadata parameter ${position} must be an object` `` (`TypeError`) | element not an object | `src/metadata/recursive-parameter-index.ts:120-124` |
| `` `recursive xRFC metadata parameter ${position} name must be non-empty` `` (`TypeError`) | `name` not a non-empty string | `src/metadata/recursive-parameter-index.ts:129-133` |
| `` `${identity.name}.${name} has duplicate recursive metadata` `` | duplicate parameter name | `src/metadata/recursive-parameter-index.ts:134-138` |
| `"recursive xRFC parameter index must be created for the same metadata graph"` (`TypeError`) | index unknown, or bound to a different graph object | `src/metadata/recursive-parameter-index.ts:166-172` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Opaque, invocation-scoped lookup for one recursive metadata graph." | `src/metadata/recursive-parameter-index.ts:10` |
| "The backing map stays module-private so a caller cannot mutate a resolved dispatch plan between preflight and value serialization." | `src/metadata/recursive-parameter-index.ts:12-13` |
| "Validate and index every parameter name once before recursive dispatch. Duplicate names are rejected even when only one of them would be active." | `src/metadata/recursive-parameter-index.ts:51-52` |
| "Read the name exactly once. Besides making the indexing bound explicit, this prevents accessor-backed hand graphs from changing lookup identity between duplicate validation and dispatch." | `src/metadata/recursive-parameter-index.ts:125-127` |
| "Read one cache entry only for a normalizer-produced immutable graph." | `src/metadata/recursive-parameter-index.ts:184` |
| "Store one cache entry only for a normalizer-produced immutable graph." | `src/metadata/recursive-parameter-index.ts:196` |
| "Resolve one parameter from a branded index bound to the same graph." | `src/metadata/recursive-parameter-index.ts:175` |
| "Deterministic internal evidence for traversal-bound regression tests." | `src/metadata/recursive-parameter-index.ts:225` |

### Wire facts asserted by tests

None of the four in-scope tests import this file (`test/recursive-metadata.test.ts:1-6`,
`test/modern-recursive-metadata.test.ts:1-11`,
`test/metadata-repository-runtime.test.ts:1-16`,
`test/rfc-metadata-get.test.ts:1-17`). The diagnostics counters are described as
existing for "traversal-bound regression tests"
(`src/metadata/recursive-parameter-index.ts:225`), but no in-scope test reads
them.

### Go mapping notes

- The public `RecursiveMetadataParameterIndex` value is deliberately a
  two-field opaque handle (`functionName`, `parameterCount`,
  `src/metadata/recursive-parameter-index.ts:142-145`) with the real state in a
  module-private `WeakMap` (`:45-48`). In Go, the natural translation is an
  unexported struct with unexported fields returned by value/pointer — the
  `WeakMap` exists only to keep the state unreachable from JS callers.
- The cache is a two-level `Map<string, Map<string, unknown>>` keyed by
  `(namespace, key)` strings (`:24`, `:206-211`); reads and writes are silently
  skipped when `state.cacheable` is false (`:192`, `:205`), and `cacheable` is
  fixed at construction time as `isNormalizedRecursiveMetadataGraph(graph)`
  (`:149`).
- `state.work[kind] += 1` (`:222`) mutates through a `readonly` state object —
  the freeze at `:146` is shallow, so the `work` record stays mutable. A Go port
  needs an explicitly mutable counters struct (and, if shared, atomics).

---

## src/metadata/repository-runtime.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `MetadataRepositoryMode` | enum | `export enum MetadataRepositoryMode { Auto = "auto", Classic = "classic", OptimizedOnly = "optimizedOnly", LegacyV3 = "legacyV3", }` | `src/metadata/repository-runtime.ts:14-19` |
| `MetadataLoadStrategy` | enum | `export enum MetadataLoadStrategy { Classic = "classic", Optimized = "optimized", LegacyV3 = "legacyV3", }` | `src/metadata/repository-runtime.ts:21-25` |
| `MetadataAccessFailureClassification` | type | `export type MetadataAccessFailureClassification = \| "unavailable" \| "authorization" \| "communication" \| "timeout" \| "canceled" \| "malformed" \| "other";` | `src/metadata/repository-runtime.ts:27-34` |
| `MetadataAccessFailure` | class | `export class MetadataAccessFailure extends Error { readonly classification: MetadataAccessFailureClassification; constructor(classification: MetadataAccessFailureClassification, message: string, options?: ErrorOptions,) }` | `src/metadata/repository-runtime.ts:40-52` |
| `MetadataStructuralKeyInput` | interface | `readonly backendKey: string; readonly metadataGeneration: string; readonly language: string; readonly objectKind: string; readonly objectName: string;` | `src/metadata/repository-runtime.ts:54-60` |
| `MetadataStructuralKey` | interface | `export interface MetadataStructuralKey extends MetadataStructuralKeyInput { readonly id: string; }` | `src/metadata/repository-runtime.ts:63-65` |
| `MetadataCapabilityKeyInput` | interface | `readonly backendKey: string; readonly principalKey: string;` | `src/metadata/repository-runtime.ts:67-70` |
| `MetadataCapabilityKey` | interface | `export interface MetadataCapabilityKey extends MetadataCapabilityKeyInput { readonly id: string; }` | `src/metadata/repository-runtime.ts:73-75` |
| `MetadataLookup` | interface | `readonly structural: MetadataStructuralKey; readonly capability: MetadataCapabilityKey; readonly mode: MetadataRepositoryMode;` | `src/metadata/repository-runtime.ts:77-81` |
| `MetadataSnapshot<T extends object>` | interface | `readonly value: T; readonly retainedBytes: number;` | `src/metadata/repository-runtime.ts:92-96` |
| `MetadataProbeContext` | interface | `readonly capability: MetadataCapabilityKey; readonly mode: MetadataRepositoryMode; readonly signal: AbortSignal;` | `src/metadata/repository-runtime.ts:98-103` |
| `MetadataAccessContext` | interface | `readonly structural; readonly capability; readonly mode; readonly strategy: MetadataLoadStrategy; readonly signal: AbortSignal;` | `src/metadata/repository-runtime.ts:105-112` |
| `MetadataAdapter<T extends object>` | interface | `probeOptimized(context: MetadataProbeContext): Promise<void>; authorize(context: MetadataAccessContext): Promise<void>; load(context: MetadataAccessContext): Promise<MetadataSnapshot<T>>;` | `src/metadata/repository-runtime.ts:115-119` |
| `MetadataRepositoryRuntimeOptions<T extends object>` | interface | `readonly maxEntries: number; readonly maxRetainedBytes: number; readonly maxProbeEntries?: number; readonly maxAuthorizationEntries?: number; readonly maxObjectEpochEntries?: number; readonly maxInFlightLoads?: number; readonly maxSnapshotNodes?: number; readonly maxSnapshotDepth?: number; readonly maxSnapshotProperties?: number; readonly adapter: MetadataAdapter<T>; readonly diagnostics?: RfcDiagnosticEmitter;` | `src/metadata/repository-runtime.ts:121-141` |
| `MetadataRepositoryMonitor` | interface | 39 readonly fields, `state: "active" \| "retired"` first | `src/metadata/repository-runtime.ts:143-183` |
| `createMetadataStructuralKey` | function | `export function createMetadataStructuralKey(input: MetadataStructuralKeyInput,): MetadataStructuralKey` | `src/metadata/repository-runtime.ts:404-406` |
| `createMetadataCapabilityKey` | function | `export function createMetadataCapabilityKey(input: MetadataCapabilityKeyInput,): MetadataCapabilityKey` | `src/metadata/repository-runtime.ts:431-433` |
| `MetadataRepositoryRuntime<T extends object>` | class | `export class MetadataRepositoryRuntime<T extends object>` | `src/metadata/repository-runtime.ts:696` |
| `.constructor` | method | `constructor(options: MetadataRepositoryRuntimeOptions<T>)` | `src/metadata/repository-runtime.ts:747` |
| `.get` | method | `async get(lookup: MetadataLookup, signal?: AbortSignal): Promise<T>` | `src/metadata/repository-runtime.ts:830` |
| `.invalidate` | method | `invalidate(structural: MetadataStructuralKey): boolean` | `src/metadata/repository-runtime.ts:895` |
| `.invalidateAll` | method | `invalidateAll(): number` | `src/metadata/repository-runtime.ts:916` |
| `.retire` | method | `retire(): Promise<void>` | `src/metadata/repository-runtime.ts:936` |
| `.monitor` | method | `monitor(): MetadataRepositoryMonitor` | `src/metadata/repository-runtime.ts:964` |

### Constants, type codes, and enumerations

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `AVAILABLE_PROBE` | `Object.freeze({ available: true })` | shared positive probe result | `src/metadata/repository-runtime.ts:229` |
| `MAX_AUXILIARY_ENTRIES` | `1_000_000` | ceiling on every derived auxiliary limit and on snapshot node/property limits | `:230`, uses `:339`, `:342-347`, `:808`, `:823` |
| `DEFAULT_MAX_SNAPSHOT_NODES` | `100_000` | default `maxSnapshotNodes` | `:231`, use `:804` |
| `DEFAULT_MAX_SNAPSHOT_DEPTH` | `256` | default `maxSnapshotDepth` | `:232`, use `:811` |
| `DEFAULT_MAX_SNAPSHOT_PROPERTIES` | `1_000_000` | default `maxSnapshotProperties` | `:233`, use `:818-819` |
| `MAX_SNAPSHOT_DEPTH` | `4_096` | ceiling on configurable `maxSnapshotDepth` | `:234`, use `:814` |
| `safeApply` | `Reflect.apply` | captured intrinsic | `:235` |
| `metadataRetirementContext` | `new AsyncLocalStorage<MetadataRetirementContext>()` | see §Immutability and caching model | `:236-237` |
| derived probe limit | `derivedAuxiliaryLimit(maxProbeEntries, maxEntries, 2, 16, "maxProbeEntries")` | multiplier 2, floor 16 | `:776-782` |
| derived authorization limit | `derivedAuxiliaryLimit(maxAuthorizationEntries, maxEntries, 4, 32, "maxAuthorizationEntries")` | multiplier 4, floor 32 | `:783-789` |
| derived object-epoch limit | `derivedAuxiliaryLimit(maxObjectEpochEntries, maxEntries, 2, 32, "maxObjectEpochEntries")` | multiplier 2, floor 32 | `:790-796` |
| derived in-flight-load limit | `derivedAuxiliaryLimit(maxInFlightLoads, maxEntries, 2, 8, "maxInFlightLoads")` | multiplier 2, floor 8 | `:797-803` |
| identity charset | `value.length < 1 \|\| value.length > 512 \|\| /[\u0000-\u001f\u007f]/u.test(value)` | opaque identity fields: 1..512, no controls | `:361-372` |
| tuple id encoding | `function tupleId(values: readonly string[]): string { return JSON.stringify(values); }` | key serialization | `:375-377` |
| epoch id encoding | `` return `${epoch.global.toString(36)}:${epoch.object.toString(36)}`; `` | base-36 bigints | `:351-353` |
| fallback-eligible classifications | `return classification === "unavailable" \|\| classification === "authorization";` | the only classifications permitting auto fallback | `:451-454` |
| array-index upper bound | `index < 4_294_967_295` | what counts as an array index property | `:463` |

### Errors

| Message (verbatim) | Trigger | Citation |
|---|---|---|
| `"metadata repository lookup was canceled"` (`MetadataAccessFailure`, classification `"canceled"`) | caller signal aborted; also used as the retirement abort reason | `:239-244`, `:833`, `:838`, `:285`, `:301`, `:958` |
| `"metadata lookup signal must be an AbortSignal"` (`TypeError`) | signal not an object, or `aborted`/`addEventListener`/`removeEventListener` wrong type | `:249-262`, `:266-269` |
| `` `${field} must be an integer in ${minimum}..${maximum}` `` (`RangeError`) | any bounded option out of range | `:318-329` |
| `` `metadata repository ${kind} capacity ${maximum} is exhausted` `` | probe/authorization/load capacity exhausted | `:355-359`, thrown `:1089`, `:1103`, `:1206`, `:1277`, `:1291` |
| `` `${field} must contain 1..512 characters without controls` `` (`RangeError`) | invalid identity component | `:361-372` |
| `"metadata adapter must provide probeOptimized()"` (`TypeError`) | missing method | `:385-387` |
| `"metadata adapter must provide authorize()"` (`TypeError`) | missing method | `:388-390` |
| `"metadata adapter must provide load()"` (`TypeError`) | missing method | `:391-393` |
| `"metadata snapshot graph must not contain Proxy objects"` (`TypeError`) | any node in the snapshot graph is a Proxy | `:487-491` |
| `"metadata snapshot value must be recursively frozen"` (`TypeError`) | function value; unfrozen `ImmutableMetadataMap`; `Map`/`Set`/`Date`/`ArrayBuffer`/typed-array view; prototype not `Object.prototype`/`Array.prototype`; extensible object; configurable, writable, or accessor descriptor | `:492-493`, `:496-498`, `:534-541`, `:542-545`, `:546-548`, `:561-570` |
| `` `metadata snapshot graph exceeds property limit ${maxProperties}` `` (`RangeError`) | property budget exceeded | `:500-504`, `:555-559` |
| `` `metadata snapshot graph exceeds depth limit ${maxDepth}` `` (`RangeError`) | child depth > `maxDepth` | `:517-521`, `:579-583` |
| `` `metadata snapshot graph exceeds node limit ${maxNodes}` `` (`RangeError`) | distinct scheduled nodes ≥ `maxNodes` | `:522-526`, `:584-589` |
| `"metadata adapter must return an object snapshot"` (`TypeError`) | snapshot or `snapshot.value` not an object | `:606-611`, `:617-619` |
| `"metadata adapter snapshot must not be a Proxy object"` (`TypeError`) | snapshot container is a Proxy | `:612-614` |
| `` `unsupported metadata repository mode ${String(value)}` `` (`RangeError`) | mode outside the enum | `:633-641` |
| `"metadata lookup must be an object"` (`TypeError`) | lookup not an object | `:645-647` |
| `"metadata structural and capability keys must identify the same backend"` | `structural.backendKey !== capability.backendKey` | `:651-655` |
| `"metadata structural key must be an object"` (`TypeError`) | key not an object | `:665-667` |
| `"metadata structural key does not match its canonical fields"` | supplied `id` ≠ recomputed `id` | `:670-674` |
| `"metadata capability key must be an object"` (`TypeError`) | key not an object | `:679-681` |
| `"metadata capability key does not match its canonical fields"` | supplied `id` ≠ recomputed `id` | `:684-688` |
| `"metadata repository options must be an object"` (`TypeError`) | options not an object | `:748-750` |
| `"metadata repository is retired"` | any `get()` after retirement (checked twice) | `:1009-1013`, called `:831`, `:837`, `:1158`, `:1162` |
| `` `optimized metadata is ${probe.fallbackClassification}` `` (`MetadataAccessFailure`, classification = `probe.fallbackClassification!`) | `OptimizedOnly` with an unavailable probe | `:1067-1070` |
| `"metadata cache accounting invariant was violated"` | eviction loop found an empty cache | `:1439-1441` |
| `"metadata cache retained-byte sum is unsafe"` | `retainedBytes` sum not a safe integer | `:1446-1448` |
| `"metadata cache retained-byte subtraction is unsafe"` | subtraction not a safe integer, or negative | `:1462-1465` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "A safe classification boundary for repository access failures. The runtime never examines message text when deciding whether fallback is permitted." | `src/metadata/repository-runtime.ts:37-38` |
| "A descriptor identity which deliberately contains no authenticated principal." | `src/metadata/repository-runtime.ts:62` |
| "A backend capability/authorization identity scoped to one opaque principal." | `src/metadata/repository-runtime.ts:72` |
| "Private, canonical lookup state. The mode-scoped capability identity keeps authorization and probe decisions separate without adding a principal to the structurally shared descriptor key." | `src/metadata/repository-runtime.ts:83-87` |
| "Must be recursively frozen before it crosses the adapter boundary." | `src/metadata/repository-runtime.ts:93` |
| "Aborted only when the owning repository generation is retired." | `src/metadata/repository-runtime.ts:101`, `:110` |
| "Transport-independent adapter implemented later by a destination/session lane." | `src/metadata/repository-runtime.ts:114` |
| "Maximum physically active descriptor loads, including invalidated loads." | `src/metadata/repository-runtime.ts:133` |
| "Optional bounded structured diagnostics; never receives metadata values." | `src/metadata/repository-runtime.ts:139` |
| "A hostile signal cleanup hook must not strand or replace the metadata operation's already-determined result." | `src/metadata/repository-runtime.ts:294-295` |
| "An abort can occur between the initial read and listener registration." | `src/metadata/repository-runtime.ts:310` |
| "Bounded, generation-local metadata state. Structural values may be shared, while every read is gated by a separately cached principal authorization." | `src/metadata/repository-runtime.ts:692-695` |
| "Lookup normalization reads caller-owned accessors. They may reenter and retire this generation, so gate again before any repository work starts." | `src/metadata/repository-runtime.ts:835-836` |
| "No operation can be admitted after the state transition. Loop so this also remains correct if a tracked chain is replaced before it observes that retirement has begun." | `src/metadata/repository-runtime.ts:946-948` |
| "Publish before abort listeners run: a cooperative adapter may reenter retire() synchronously from its generation-retirement signal." | `src/metadata/repository-runtime.ts:953-954` |
| "Dropping an object epoch in isolation could make a pre-invalidation operation match the default epoch again. Advance the global identity and clear every epoch-qualified admission instead." | `src/metadata/repository-runtime.ts:1041-1043` |
| "A fallback remains correct for this call even when every bounded capability slot is physically active; only its reuse is skipped." | `src/metadata/repository-runtime.ts:1401-1402` |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| A structural key has no `principalKey`; changing `language` or `metadataGeneration` changes `id`; keys are frozen | `"keeps structural and principal-scoped capability keys separate"` | `test/metadata-repository-runtime.test.ts:84-101` |
| A forged `id` rejects with `/structural key does not match/`, a cross-backend capability with `/same backend/`, and **zero** adapter calls occur | `"rejects forged or cross-backend identities before adapter access"` | `test/metadata-repository-runtime.test.ts:103-128` |
| Caller-owned key/mode objects are snapshotted before authorization; later mutation does not change the context the adapter sees | `"snapshots caller-owned identity and mode before authorization"` | `test/metadata-repository-runtime.test.ts:130`, `:193-201` |
| A mutated caller lookup cannot redirect the load or poison another structural key | `"caller mutation cannot redirect a load or poison another structural cache key"` | `test/metadata-repository-runtime.test.ts:226`, `:272-285` |
| A `get mode()` accessor calling `retire()` yields `/metadata repository is retired/` with `probes: 0, authorizations: 0, loads: 0` | `"rechecks retirement after lookup accessors before backend work"` | `test/metadata-repository-runtime.test.ts:288`, `:318-335` |
| Adapter methods are captured at construction; later reassignment never runs | `"snapshots and binds adapter operations for the lifetime of a repository generation"` | `test/metadata-repository-runtime.test.ts:338`, `:358-376` |
| Missing `probeOptimized`/`authorize`/`load` each throw at construction | `"rejects incomplete adapters during repository construction"` | `test/metadata-repository-runtime.test.ts:379-394` |
| Every option and adapter method is read **exactly once** (`assert.equal(reads.get(field), 1, field)`) | `"snapshots every repository option and adapter operation exactly once"` | `test/metadata-repository-runtime.test.ts:396`, `:476-492` |
| Two concurrent cold `get()`s issue one load and `inFlightJoins === 1`; after a failure the next `get()` retries | `"deduplicates concurrent cold loads and retries a failed load"` | `test/metadata-repository-runtime.test.ts:495`, `:521-531` |
| Two lookups on different structural keys with the same capability issue one probe | `"deduplicates an optimized capability probe for one backend principal"` | `test/metadata-repository-runtime.test.ts:534`, `:553-557` |
| Probe and authorization identities are mode-scoped: `Auto` and `OptimizedOnly` probe separately and authorize separately, and an `Auto`-cached value is still returned to an `OptimizedOnly` lookup after its own authorization | `"separates probe and authorization identities by repository mode"` | `test/metadata-repository-runtime.test.ts:560`, `:594-621` |
| Two principals on the same structural key each get their own physical load (`inFlight === 2`) | `"does not share a cold-load promise across different principals"` | `test/metadata-repository-runtime.test.ts:624`, `:645-650` |
| Eviction is LRU by last `get()`, `entries === 2`, `retainedBytes === 20`, `evictions === 1` | `"evicts by deterministic LRU order and retained-byte budget"` | `test/metadata-repository-runtime.test.ts:653`, `:671-677` |
| The byte budget evicts independently of entry count | `"enforces the retained-byte budget independently of entry count"` | `test/metadata-repository-runtime.test.ts:680`, `:692-701` |
| An oversize snapshot is returned but never cached; `oversizeSkips === 2` | `"does not cache a snapshot larger than the byte budget"` | `test/metadata-repository-runtime.test.ts:704`, `:716-719` |
| Values already handed out survive `invalidate` and `invalidateAll` unchanged and stay frozen | `"per-object and whole invalidation preserve returned immutable snapshots"` | `test/metadata-repository-runtime.test.ts:722`, `:735-746` |
| A load that completes after its own invalidation returns to its caller but does not repopulate the cache (`entries === 0`) | `"an invalidation during a load prevents stale cache resurrection"` | `test/metadata-repository-runtime.test.ts:749`, `:774-780` |
| A principal denied the optimized probe still reads the value another principal cached — `assert.equal(fullFirst, restrictedSecond)` — with one probe per principal | `"caches optimized probes per backend and principal without cross-principal poisoning"` | `test/metadata-repository-runtime.test.ts:783`, `:807-823` |
| Positive object authorization is never shared across principals | `"does not share positive object authorization across principals"` | `test/metadata-repository-runtime.test.ts:826`, `:841-848` |
| Auto falls back only for `"unavailable"`/`"authorization"`; `"communication"`, `"timeout"`, `"canceled"`, `"malformed"` propagate with `loads === 0` | `"auto falls back only for unavailable or authorization failures"` | `test/metadata-repository-runtime.test.ts:851-899` |
| Auto also falls back on an optimized `authorize`/`load` failure; `OptimizedOnly` never falls back (`classicLoads === 0`) | `"auto also classifies optimized authorization/load failures but optimized-only never falls back"` | `test/metadata-repository-runtime.test.ts:901`, `:922-946` |
| Terminal optimized load failures leave `strategies` as `[Optimized]` only | `"terminal optimized-load failures never fall back to classic"` | `test/metadata-repository-runtime.test.ts:949`, `:967-974` |
| `Classic`/`LegacyV3` never probe | `"classic and legacy modes select only their explicit loader strategies"` | `test/metadata-repository-runtime.test.ts:977`, `:995-999` |
| A mutable loader value rejects `/recursively frozen/`; the next immutable one succeeds | `"rejects mutable loader values and retries after the loader returns an immutable snapshot"` | `test/metadata-repository-runtime.test.ts:1002`, `:1017-1018` |
| Only the exact `ImmutableMetadataMap` class is accepted (see immutable-map section) | `"accepts only the exact immutable metadata map implementation in snapshots"` | `test/metadata-repository-runtime.test.ts:1021-1055` |
| A frozen object with a symbol-keyed mutable property rejects `/recursively frozen/` | `"rejects frozen snapshots with mutable symbol-linked state"` | `test/metadata-repository-runtime.test.ts:1057`, `:1083-1085` |
| A `Proxy` around a frozen target rejects `/Proxy/u` | `"rejects Proxy values which only appear recursively frozen"` | `test/metadata-repository-runtime.test.ts:1088`, `:1112-1114` |
| A `Proxy` snapshot *container* rejects before any field read (`propertyReads === 0`) | `"rejects Proxy snapshot containers before reading their fields"` | `test/metadata-repository-runtime.test.ts:1117`, `:1137-1141` |
| Primitive-only own properties count against `maxSnapshotProperties`: `/metadata snapshot graph exceeds property limit 2/u` | `"bounds inspected snapshot properties including primitive-only values"` | `test/metadata-repository-runtime.test.ts:1144`, `:1170-1176` |
| A `new Array(4_294_967_295)` sparse array is counted by logical slots, not stored keys | `"counts sparse array logical slots against the snapshot property bound"` | `test/metadata-repository-runtime.test.ts:1179`, `:1183`, `:1200-1211` |
| A 20 000-deep frozen chain fails with exactly `"metadata snapshot graph exceeds depth limit 16"` and no stack overflow | `"bounds immutable snapshot traversal by depth without recursive stack growth"` | `test/metadata-repository-runtime.test.ts:1214`, `:1219`, `:1236-1250` |
| A wide graph exceeding `maxSnapshotNodes: 3` fails with `"metadata snapshot graph exceeds node limit 3"`; a **self-referential frozen cycle is accepted** at `maxSnapshotNodes: 1, maxSnapshotDepth: 1` | `"bounds distinct immutable snapshot nodes and accepts a bounded cycle"` | `test/metadata-repository-runtime.test.ts:1253`, `:1276-1281`, `:1283-1299` |
| `monitor()` returns a frozen snapshot that does not change when the repository changes | `"retirement clears state, blocks new work, and monitor snapshots never mutate"` | `test/metadata-repository-runtime.test.ts:1302`, `:1313-1319` |
| An invalidation mid-authorization forces reauthorization into the new epoch, then both readers join one load | `"reauthorizes and rejoins current work after invalidation changes an admission epoch"` | `test/metadata-repository-runtime.test.ts:1322`, `:1351-1370` |
| A lone lookup that reauthorizes into the current epoch still caches its result | `"caches a lone lookup after it reauthorizes into the current epoch"` | `test/metadata-repository-runtime.test.ts:1373`, `:1404-1407` |
| Retained-byte accounting stays exact at `Number.MAX_SAFE_INTEGER` | `"keeps retained-byte accounting safe at Number.MAX_SAFE_INTEGER"` | `test/metadata-repository-runtime.test.ts:1410`, `:1433-1440` |
| Probe/authorization caches evict settled entries in deterministic LRU order | `"bounds probe and authorization decisions with deterministic settled-entry LRU"` | `test/metadata-repository-runtime.test.ts:1443`, `:1477-1501` |
| Exceeding `maxObjectEpochEntries` clears all object epochs and increments `objectEpochCompactions` | `"bounds object invalidation epochs by deterministic global compaction"` | `test/metadata-repository-runtime.test.ts:1504`, `:1513-1524` |
| Occupied probe/authorization/load capacity rejects new work with `/probe capacity 1 is exhausted/`, `/authorization capacity 1 is exhausted/`, `/load capacity 1 is exhausted/` | `"rejects new active work when probe, authorization, or load capacity is occupied"` | `test/metadata-repository-runtime.test.ts:1527`, `:1546-1551`, `:1571-1574`, `:1601-1604` |
| With `maxEntries: 0`, every derived limit is still a positive safe integer; `Number.POSITIVE_INFINITY` and `0` are rejected per field | `"derives finite auxiliary limits and rejects invalid explicit capacities"` | `test/metadata-repository-runtime.test.ts:1613`, `:1619-1665` |
| A poisoned `Function.prototype.bind` on adapter methods does not run; `Reflect.apply` is used instead | `"does not trust caller-owned Function.bind on adapter operations"` | `test/metadata-repository-runtime.test.ts:1668`, `:1688-1709` |
| `retire()` flips `state` to `"retired"` and clears entries immediately, but its promise resolves only after the in-flight load drains | `"retirement drains an admitted metadata load after invalidating its result"` | `test/metadata-repository-runtime.test.ts:1712`, `:1730-1745` |
| A caller `AbortSignal` is **not** the adapter's signal (`assert.notEqual(physicalSignal, caller.signal)`); aborting it rejects only that waiter with classification `"canceled"` while the physical load continues (`physicalSignal?.aborted === false`) | `"caller cancellation releases only that waiter and preserves a shared physical load"` | `test/metadata-repository-runtime.test.ts:1748`, `:1770`, `:1779-1785` |
| A `removeEventListener` that throws cannot strand settlement | `"a hostile abort-listener cleanup cannot strand caller settlement"` | `test/metadata-repository-runtime.test.ts:1789`, `:1813-1817` |
| Retirement aborts the adapter signal at each of probe/authorize/load, rejecting with classification `"canceled"` | `"generation retirement aborts cooperative probe, authorization, and load operations"` | `test/metadata-repository-runtime.test.ts:1820`, `:1861-1870` |
| After an optimized `authorize`/`load` fallback, the same principal skips optimized on the next object, while another principal still probes; `optimizedFallbacks === 2` | `"auto remembers optimized authorize and load fallback per principal"` | `test/metadata-repository-runtime.test.ts:1875`, `:1927-1937` |
| A late fallback memo does not exceed `maxProbeEntries: 1`; `probeEntries` stays 1 with `probeEvictions === 1` | `"a late optimized fallback never exceeds the bounded principal decision cache"` | `test/metadata-repository-runtime.test.ts:1943`, `:1985-1997` |
| A pre-invalidation lookup reauthorizes before it is allowed to read a post-invalidation cache hit (`authorizations === [FULL, RESTRICTED, FULL]`) | `"a pre-invalidation lookup reauthorizes before reading a post-invalidation cache hit"` | `test/metadata-repository-runtime.test.ts:2001`, `:2026-2043` |
| An abort-cleanup handler may call `retire()` again from inside the retirement dispatch without deadlock, receiving a **different** promise | `"abort cleanup may await owning repository retirement without deadlock"` | `test/metadata-repository-runtime.test.ts:2047`, `:2100-2101`, `:2124-2125` |

### Go mapping notes

- Every map in the class is a JS `Map` relied on for insertion-order iteration
  (`#cache`, `#inFlight`, `#authorizations`, `#probes`, `#objectEpochs`,
  `:708-712`). LRU is implemented by `delete` + `set`
  (`:1178-1179`, `:1083-1084`, `:1269-1270`, `:1420-1421`, `:1032-1033`,
  `:1405-1406`) and eviction by `this.#cache.keys().next().value` (`:1438`).
  Go maps have **no** iteration order — the port needs an explicit
  ordered structure (e.g. `container/list` + `map`), otherwise the
  determinism the tests assert (`test/metadata-repository-runtime.test.ts:653`,
  `:1443`) is lost.
- Epochs are `bigint` (`#globalEpoch = 0n`, `:722`; `object: bigint`, `:216-217`;
  `existing + 1n`, `:1033`). `uint64` overflow behaviour is not stated in
  source; the JS version cannot overflow.
- The whole class is single-threaded: there is no lock, and every check-then-act
  pair (e.g. `#activeLoads >= #maxInFlightLoads` then `#activeLoads += 1`,
  `:1204-1217`) relies on the JS event loop. A Go port must add a mutex around
  the entire state, or the capacity, epoch, and LRU invariants break.
- `Promise` deduplication (`#inFlight`, `#probes.promise`,
  `#authorizations.promise`) becomes `golang.org/x/sync/singleflight` or a
  `chan struct{}`-gated entry — but note the key includes the epoch
  (`:1191-1193`), so a stale entry must not be joined.

---

## src/metadata/rfc-function-interface.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RfcFunctionInterface` | interface | `readonly name: string; readonly remoteBasxmlSupported: boolean; readonly remoteCall: string; readonly updateTask: boolean; readonly parameters: readonly RfcFunintParameter[]; readonly exceptions: readonly string[]; readonly resumableExceptionRowCount: number;` | `src/metadata/rfc-function-interface.ts:22-30` |
| `buildRfcGetFunctionInterfaceRequest` | function | `export function buildRfcGetFunctionInterfaceRequest(functionName: string): Buffer` | `src/metadata/rfc-function-interface.ts:33` |
| `decodeRfcFunctionInterfaceResult` | function | `export function decodeRfcFunctionInterfaceResult(functionName: string, fields: readonly CpicField[],): RfcFunctionInterface` | `src/metadata/rfc-function-interface.ts:64-67` |

### Constants, type codes, and enumerations

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `FUNCTION_INTERFACE_OUTPUTS` | `["REMOTE_BASXML_SUPPORTED","REMOTE_CALL","UPDATE_TASK","PARAMS","RESUMABLE_EXCEPTIONS"]` | requested outputs of `RFC_GET_FUNCTION_INTERFACE` | `src/metadata/rfc-function-interface.ts:14-20`, use `:36` |
| import `FUNCNAME` | `{ name: "FUNCNAME", value: encodeAbapChar(functionName, 30) }` | 30-char function name | `src/metadata/rfc-function-interface.ts:38` |
| import `NONE_UNICODE_LENGTH` | `{ name: "NONE_UNICODE_LENGTH", value: encodeAbapChar("X", 1) }` | fixed `"X"` | `src/metadata/rfc-function-interface.ts:39` |
| exception marker | `parameter.parameterClass === "X"` | `X` rows become `exceptions`, everything else `parameters` | `src/metadata/rfc-function-interface.ts:96-101` |
| flag alphabet | `if (decoded !== "" && decoded !== "X")` | boolean flags are `""` or `"X"` | `src/metadata/rfc-function-interface.ts:56-60` |
| `RFC_FUNINT_UNICODE_ROW_LENGTH` | imported from `"../protocol/classic-rfc.js"` | minimum `PARAMS` row width; the numeric value is **not stated in source** (out of scope) | `src/metadata/rfc-function-interface.ts:2`, use `:80-84` |

### Errors

| Message (verbatim) | Trigger | Citation |
|---|---|---|
| `` `RFC_GET_FUNCTION_INTERFACE response lacks scalar ${name}` `` | required scalar missing | `src/metadata/rfc-function-interface.ts:50` |
| `` `${name} contains unsupported flag value ${decoded}` `` | flag not `""`/`"X"` | `src/metadata/rfc-function-interface.ts:58` |
| `"RFC_GET_FUNCTION_INTERFACE response lacks PARAMS table"` | `PARAMS` absent | `src/metadata/rfc-function-interface.ts:71` |
| `` `RFC_GET_FUNCTION_INTERFACE PARAMS row width is ${params.rowByteLength}; expected at least ${RFC_FUNINT_UNICODE_ROW_LENGTH}` `` | row narrower than the stable prefix | `src/metadata/rfc-function-interface.ts:80-85` |
| `"RFC_GET_FUNCTION_INTERFACE response lacks RESUMABLE_EXCEPTIONS table"` | table absent | `src/metadata/rfc-function-interface.ts:89-93` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Build the classic metadata bootstrap call without requiring prior metadata." | `src/metadata/rfc-function-interface.ts:32` |
| "`rowByteLength` is the first row's own width, or the declared width when the table is empty. Both grow with the peer's release - a 404-byte declaration is already evidenced - so bound the width below by the stable prefix the row decoder consumes instead of pinning it. Narrower rows are still refused. This replaces an explicit 402/404 exception that only held for an empty table; a populated table on the same release would have failed every metadata lookup." | `src/metadata/rfc-function-interface.ts:73-79` |
| "Normalize a successful RFC_GET_FUNCTION_INTERFACE classic response." | `src/metadata/rfc-function-interface.ts:63` |

### Wire facts asserted by tests

None of the four in-scope tests import this file. `RfcFunctionInterface` is
only exercised indirectly, as the return type of
`normalizeRfcMetadataGetFunction` (`src/metadata/rfc-metadata-get.ts:842-847`,
asserted at `test/rfc-metadata-get.test.ts:657-678`).

### Go mapping notes

- `remoteCall` is a decoded 1-character string, not a boolean
  (`decodeAbapChar(requiredScalar(result.scalars, "REMOTE_CALL"), 1)`,
  `src/metadata/rfc-function-interface.ts:108-111`). Its permitted values are
  **not stated in source**; `rfc-metadata-get.ts` hardcodes `remoteCall: "R"`
  (`src/metadata/rfc-metadata-get.ts:828`, `:189`, `:240`).
- `resumableExceptionRowCount` is a row **count**, never the rows themselves
  (`resumableExceptions.rows.length`, `src/metadata/rfc-function-interface.ts:118`).

---

## src/metadata/rfc-metadata-get.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RfcMetadataGetBootstrap` | interface | `readonly metadata: RfcFunctionInterface; readonly structures: ReadonlyMap<string, RfcStructureDefinition>;` | `src/metadata/rfc-metadata-get.ts:209-212` |
| `RFC_METADATA_GET_BOOTSTRAP` | const | `export const RFC_METADATA_GET_BOOTSTRAP: RfcMetadataGetBootstrap = Object.freeze({ metadata: BOOTSTRAP_METADATA, structures: new ImmutableMap(...) })` | `src/metadata/rfc-metadata-get.ts:214-221` |
| `RFC_METADATA_GET_TIMESTAMP_BOOTSTRAP` | const | `export const RFC_METADATA_GET_TIMESTAMP_BOOTSTRAP: RfcMetadataGetBootstrap = Object.freeze({ metadata: TIMESTAMP_BOOTSTRAP_METADATA, structures: new ImmutableMap(...) })` | `src/metadata/rfc-metadata-get.ts:267-274` |
| `RfcMetadataGetInvocation` | interface | `readonly input: Readonly<Record<string, unknown>>;` | `src/metadata/rfc-metadata-get.ts:276-278` |
| `RfcMetadataGetTimestampInvocation` | interface | `export interface RfcMetadataGetTimestampInvocation extends RfcMetadataGetInvocation { readonly functionNames: readonly string[]; readonly structureNames: readonly string[]; }` | `src/metadata/rfc-metadata-get.ts:280-285` |
| `createRfcMetadataGetFunctionInvocation` | function | `export function createRfcMetadataGetFunctionInvocation(functionName: string, language = "E",): RfcMetadataGetInvocation` | `src/metadata/rfc-metadata-get.ts:315-318` |
| `createRfcMetadataGetStructureInvocation` | function | `export function createRfcMetadataGetStructureInvocation(structureName: string, language = "E",): RfcMetadataGetInvocation` | `src/metadata/rfc-metadata-get.ts:325-328` |
| `createRfcMetadataGetTimestampInvocation` | function | `export function createRfcMetadataGetTimestampInvocation(functionNames: readonly string[], structureNames: readonly string[],): RfcMetadataGetTimestampInvocation` | `src/metadata/rfc-metadata-get.ts:368-371` |
| `RfcFunctionMetadataTimestamp` | interface | `readonly functionName: string; readonly date: string; readonly time: string; readonly token: string;` | `src/metadata/rfc-metadata-get.ts:515-520` |
| `RfcStructureMetadataTimestamp` | interface | `readonly structureName: string; readonly timestamp: string; readonly token: string;` | `src/metadata/rfc-metadata-get.ts:522-526` |
| `RfcMetadataTimestampBatch` | interface | `readonly functions: ReadonlyMap<string, RfcFunctionMetadataTimestamp>; readonly structures: ReadonlyMap<string, RfcStructureMetadataTimestamp>; readonly functionErrors: ReadonlyMap<string, string>; readonly structureErrors: ReadonlyMap<string, string>;` | `src/metadata/rfc-metadata-get.ts:528-533` |
| `RfcMetadataGetFunctionResult` | interface | `readonly value: RfcFunctionInterface; readonly generationToken: string;` | `src/metadata/rfc-metadata-get.ts:540-543` |
| `RfcMetadataGetStructureResult` | interface | `readonly value: RfcStructureDefinition; readonly generationToken: string;` | `src/metadata/rfc-metadata-get.ts:546-549` |
| `RfcMetadataGetRecursiveFunctionResult` | interface | `readonly value: RecursiveMetadataGraph; readonly generationToken: string;` | `src/metadata/rfc-metadata-get.ts:555-558` |
| `normalizeRfcMetadataGetTimestamps` | function | `export function normalizeRfcMetadataGetTimestamps(functionNames: readonly string[], structureNames: readonly string[], value: unknown,): RfcMetadataTimestampBatch` | `src/metadata/rfc-metadata-get.ts:602-606` |
| `normalizeRfcMetadataGetFunctionResult` | function | `export function normalizeRfcMetadataGetFunctionResult(functionName: string, value: unknown,): RfcMetadataGetFunctionResult` | `src/metadata/rfc-metadata-get.ts:757-760` |
| `normalizeRfcMetadataGetFunction` | function | `export function normalizeRfcMetadataGetFunction(functionName: string, value: unknown,): RfcFunctionInterface` | `src/metadata/rfc-metadata-get.ts:842-845` |
| `normalizeRfcMetadataGetRecursiveFunctionResult` | function | `export function normalizeRfcMetadataGetRecursiveFunctionResult(functionName: string, value: unknown,): RfcMetadataGetRecursiveFunctionResult` | `src/metadata/rfc-metadata-get.ts:938-941` |
| `normalizeRfcMetadataGetStructureResult` | function | `export function normalizeRfcMetadataGetStructureResult(structureName: string, value: unknown,): RfcMetadataGetStructureResult` | `src/metadata/rfc-metadata-get.ts:1006-1009` |
| `normalizeRfcMetadataGetStructure` | function | `export function normalizeRfcMetadataGetStructure(structureName: string, value: unknown,): RfcStructureDefinition` | `src/metadata/rfc-metadata-get.ts:1107-1110` |

Note: this file declares its **own** private `class ImmutableMap<K, V>`
(`src/metadata/rfc-metadata-get.ts:23-46`) that is byte-for-byte the same shape
as `ImmutableMetadataMap` (`src/metadata/immutable-map.ts:9-34`) but is a
different class and is **not** the one `isImmutableMetadataMap` accepts
(`src/metadata/immutable-map.ts:42-43`).

### Constants, type codes, and enumerations

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `MAX_METADATA_ROWS` | `100_000` | default per-table row cap in `rows()` | `src/metadata/rfc-metadata-get.ts:14`, use `:414` |
| `MAX_RECURSIVE_METADATA_ROWS` | `20_000` | total row cap across the four recursive tables | `src/metadata/rfc-metadata-get.ts:17`, use `:453` |
| `MAX_STRUCTURE_FIELDS` | `9_999` | per-structure field cap | `src/metadata/rfc-metadata-get.ts:18`, use `:1027` |
| `MAX_TIMESTAMP_NAMES_PER_KIND` | `512` | timestamp batch size per kind | `src/metadata/rfc-metadata-get.ts:19`, use `:342` |
| `REMOTE_DDIC_RESOLUTION_ERRORS` | `"REMOTE_DDIC_RESOLUTION_ERRORS"` | `RecursiveMetadataError` code for a partial type closure | `src/metadata/rfc-metadata-get.ts:21`, use `:962` |
| recursive row-budget tables | `["FUNCTIONNAMES","DATATYPESCONT","INDIRECTTYPES","PARAMETERS"]` | the four tables summed against the 20 000 budget | `src/metadata/rfc-metadata-get.ts:442-447` |
| `BOOTSTRAP_METADATA.name` | `"RFC_METADATA_GET"` | | `src/metadata/rfc-metadata-get.ts:187` |
| `BOOTSTRAP_METADATA` scalars | `remoteBasxmlSupported: false, remoteCall: "R", updateTask: false` | | `src/metadata/rfc-metadata-get.ts:188-190` |
| `BOOTSTRAP_METADATA.exceptions` | `Object.freeze(["INVALID_MODE", "INTERNAL_ERROR"])` | | `src/metadata/rfc-metadata-get.ts:205` |
| `BOOTSTRAP_METADATA.parameters` | `DEEP`/`LANGUAGE`/`GET_CLIENT_DEP_FIELDS`/`GET_TIMESTAMPS` as `"I"` with `internalLength: 1, optional: true`; then `FUNCTIONNAMES`→`RFCFUNCTIONNAME`, `DATATYPES`→`RFC_MD_DDIC_NAME`, `KNOWN_DATATYPES`→`RFC_MD_DDIC_NAME`, `PARAMETERS`→`RFC_METADATA_PARAMS`, `DATATYPESCONT`→`RFC_METADATA_DDIC`, `INDIRECTTYPES`→`RFC_METADATA_DDIC_INDIRECT`, `FUNC_ERRORS`→`RFC_FUNC_ERROR` (optional), `DD_ERRORS`→`RFC_DD_ERROR` (optional), all `"T"` | | `src/metadata/rfc-metadata-get.ts:191-204` |
| default `exid` | `options.exid ?? (parameterClass === "T" ? "u" : "C")` | table params default to `"u"`, others to `"C"` | `src/metadata/rfc-metadata-get.ts:82` |
| `RFCFUNCTIONNAME` | byteLength `90`; `FUNCTIONNAME` 1,0,60,"C"; `BASXML_SUPPORTED` 2,60,2,"C"; `UDAT` 3,62,16,"D"; `UTIME` 4,78,12,"T" | bootstrap DDIC geometry | `src/metadata/rfc-metadata-get.ts:126-131` |
| `RFC_MD_DDIC_NAME` | byteLength `120`; `TABNAME` 1,0,60,"C"; `FIELDNAME` 2,60,60,"C" | | `src/metadata/rfc-metadata-get.ts:132-135` |
| `RFC_METADATA_PARAMS` | byteLength `464`; `FUNCNAME` 1,0,60,"C"; `PARAMCLASS` 2,60,2,"C"; `PARAMETER` 3,62,60,"C"; `TABNAME` 4,122,60,"C"; `FIELDNAME` 5,182,60,"C"; `EXID` 6,242,2,"C"; `POSITION` 7,244,4,"I"; `OFFSET` 8,248,4,"I"; `INTLENGTH` 9,252,4,"I"; `DECIMALS` 10,256,4,"I"; `DEFAULT` 11,260,42,"C"; `PARAMTEXT` 12,302,158,"C"; `OPTIONAL` 13,460,2,"C" | | `src/metadata/rfc-metadata-get.ts:136-150` |
| `RFC_METADATA_DDIC` | byteLength `424`; `TYPENAME` 1,0,60,"C"; `FIELDNAME` 2,60,60,"C"; `COMPTYPE` 3,120,2,"C"; `FIELDTYPE` 4,122,60,"C"; `DATATYPE` 5,182,8,"C"; `TABLENGTH` 6,190,12,"N"; `TABLENGTH_UC` 7,202,12,"N"; `DESCRIPTION` 8,214,120,"C"; `DECIMALS` 9,334,12,"N"; `INTTYPE` 10,346,2,"C"; `OFFSET` 11,348,12,"N"; `OFFSET_UC` 12,360,12,"N"; `INTLEN` 13,372,12,"N"; `INTLEN_UC` 14,384,12,"N"; `TIMESTAMP` 15,396,28,"C" | | `src/metadata/rfc-metadata-get.ts:151-167` |
| `RFC_METADATA_DDIC_INDIRECT` | byteLength `180`; `TABNAME` 1,0,60,"C"; `FIELDNAME` 2,60,60,"C"; `FIELDTYPE` 3,120,60,"C" | | `src/metadata/rfc-metadata-get.ts:168-172` |
| `RFC_FUNC_ERROR` | byteLength `630`; `FUNCNAME` 1,0,60,"C"; `EXCEPTION` 2,60,60,"C"; `EXCEPTION_TEXT` 3,120,510,"C" | | `src/metadata/rfc-metadata-get.ts:173-177` |
| `RFC_DD_ERROR` | byteLength `690`; `TABNAME` 1,0,60,"C"; `FIELDNAME` 2,60,60,"C"; `EXCEPTION` 3,120,60,"C"; `EXCEPTION_TEXT` 4,180,510,"C" | | `src/metadata/rfc-metadata-get.ts:178-183` |
| `RFC_METADATA_FUNC_TIMESTAMP` | byteLength `88`; `FUNCNAME` 1,0,60,"C"; `UDAT` 2,60,16,"D"; `UTIME` 3,76,12,"T" | | `src/metadata/rfc-metadata-get.ts:224-228` |
| `RFC_METADATA_DDIC_TIMESTAMP` | byteLength `88`; `TYPENAME` 1,0,60,"C"; `TIMESTAMP` 2,60,28,"C" | | `src/metadata/rfc-metadata-get.ts:229-232` |
| `TIMESTAMP_BOOTSTRAP_METADATA.name` | `"RFC_METADATA_GET_TIMESTAMP"` | | `src/metadata/rfc-metadata-get.ts:238` |
| timestamp bootstrap exceptions | `Object.freeze([])` | none | `src/metadata/rfc-metadata-get.ts:258` |
| base request input | `{ DEEP: "X", LANGUAGE: language, GET_TIMESTAMPS: "X", FUNCTIONNAMES: [], DATATYPES: [], KNOWN_DATATYPES: [], PARAMETERS: [], DATATYPESCONT: [], INDIRECTTYPES: [], FUNC_ERRORS: [], DD_ERRORS: [] }` | note `GET_CLIENT_DEP_FIELDS` is **not** sent | `src/metadata/rfc-metadata-get.ts:287-301` |
| function generation token | `` `function:${date}:${time}` `` | `UDAT` (8 digits) + `UTIME` (6 digits) | `src/metadata/rfc-metadata-get.ts:647`, `:838` |
| structure generation token | `` `structure:${timestamp}` `` | 14-digit `TIMESTAMP` | `src/metadata/rfc-metadata-get.ts:678`, `:1103` |
| exception name charset | `/^[A-Z0-9_]{1,30}$/u` | valid `EXCEPTION` value | `src/metadata/rfc-metadata-get.ts:588`, `:749` |
| Unicode-halved `EXID` set | `if (!["C", "N", "D", "T"].includes(exid)) return value;` | only these have `INTLENGTH` divided by 2 | `src/metadata/rfc-metadata-get.ts:730-734` |
| composite `INTTYPE` set | `exid === "u" \|\| exid === "h" \|\| exid === "v"` | rejected by the flat structure normalizer | `src/metadata/rfc-metadata-get.ts:1066-1073` |
| accepted `COMPTYPE` set | `(componentType !== "" && componentType !== "E")` rejects | `""` and `"E"` accepted | `src/metadata/rfc-metadata-get.ts:1066-1073` |
| UTCLONG fallback shape | `TABNAME "UTCLONG"`, `FIELDNAME ""`, `EXCEPTION "NOT_FOUND"`, parameter `PARAMCLASS "C"`, `FIELDNAME ""`, `EXID "p"`, `INTLENGTH 8`, `DECIMALS 0`, `OPTIONAL ""` | the single admitted DDIC-miss shape | `src/metadata/rfc-metadata-get.ts:860-907` |
| structure geometry columns | `TABLENGTH_UC`, `OFFSET_UC`, `INTLEN_UC` | the flat structure normalizer reads **only** the Unicode columns | `src/metadata/rfc-metadata-get.ts:1040`, `:1074-1075` |

### Errors

| Message (verbatim) | Trigger | Citation |
|---|---|---|
| `` `${path} must contain 1..30 ASCII bytes` `` (`RangeError`) | metadata name invalid | `src/metadata/rfc-metadata-get.ts:48-57` |
| `"language must contain one printable SAP language code"` (`RangeError`) | language not `/^[\x20-\x7e]$/u` | `src/metadata/rfc-metadata-get.ts:60-64` |
| `` `${kind} names must be an array` `` (`TypeError`) | non-array names | `src/metadata/rfc-metadata-get.ts:339-341` |
| `` `RFC_METADATA_GET_TIMESTAMP accepts at most ${MAX_TIMESTAMP_NAMES_PER_KIND} ${kind} names` `` (`RangeError`) | >512 names | `src/metadata/rfc-metadata-get.ts:342-346` |
| `` `duplicate ${kind} name ${name}` `` | duplicate requested name | `src/metadata/rfc-metadata-get.ts:358-360` |
| `` `${path} must be an object` `` (`TypeError`) | non-object / array where a record is required | `src/metadata/rfc-metadata-get.ts:387-391` |
| `` `${path}.${name} must be an own data property` `` (`TypeError`) | accessor, missing, or throwing descriptor | `src/metadata/rfc-metadata-get.ts:394-408` |
| `` `RFC_METADATA_GET output ${name} must contain at most ${maximum} rows` `` (`RangeError`) | non-array or over the cap | `src/metadata/rfc-metadata-get.ts:417-421` |
| `` `RFC_METADATA_GET output ${name} must be an array` `` (`TypeError`) | recursive budget check on a non-array | `src/metadata/rfc-metadata-get.ts:449-451` |
| `` `RFC_METADATA_GET recursive metadata must contain at most ${MAX_RECURSIVE_METADATA_ROWS} total rows` `` (`RangeError`) | summed rows > 20 000 | `src/metadata/rfc-metadata-get.ts:453-458` |
| `` `${path}.${name} contains invalid text` `` | non-string, too long, or control characters | `src/metadata/rfc-metadata-get.ts:468-475` |
| `` `${path}.${name} must be a non-negative safe integer` `` | invalid integer | `src/metadata/rfc-metadata-get.ts:485-492` |
| `` `${path} must be initial or X` `` | flag not `""`/`"X"` | `src/metadata/rfc-metadata-get.ts:495-499` |
| `` `${path}.${name} must contain exactly ${length} digits` `` | fixed-digit field wrong | `src/metadata/rfc-metadata-get.ts:508-512` |
| `` `RFC_METADATA_GET_TIMESTAMP returned unrequested ${kind} ${objectName}` `` | foreign error row | `src/metadata/rfc-metadata-get.ts:577-581` |
| `` `RFC_METADATA_GET_TIMESTAMP returned duplicate outcome for ${kind} ${objectName}` `` | two outcomes for one object | `src/metadata/rfc-metadata-get.ts:582-586` |
| `` `${path}.EXCEPTION is invalid` `` | exception name outside `/^[A-Z0-9_]{1,30}$/u` | `src/metadata/rfc-metadata-get.ts:588-590`, `:749-751` |
| `` `RFC_METADATA_GET_TIMESTAMP returned unrequested function ${functionName}` `` | foreign success row | `src/metadata/rfc-metadata-get.ts:630-634` |
| `` `RFC_METADATA_GET_TIMESTAMP returned duplicate outcome for function ${functionName}` `` | duplicate | `src/metadata/rfc-metadata-get.ts:635-639` |
| `` `RFC_METADATA_GET_TIMESTAMP returned unrequested structure ${structureName}` `` | foreign success row | `src/metadata/rfc-metadata-get.ts:663-667` |
| `` `RFC_METADATA_GET_TIMESTAMP returned duplicate outcome for structure ${structureName}` `` | duplicate | `src/metadata/rfc-metadata-get.ts:668-672` |
| `` `RFC_METADATA_GET_TIMESTAMP returned no outcome for function ${name}` `` | requested but absent | `src/metadata/rfc-metadata-get.ts:698-704` |
| `` `RFC_METADATA_GET_TIMESTAMP returned no outcome for structure ${name}` `` | requested but absent | `src/metadata/rfc-metadata-get.ts:705-711` |
| `` `${path}.INTLENGTH has an odd Unicode byte width` `` | odd length for `C`/`N`/`D`/`T` | `src/metadata/rfc-metadata-get.ts:731-734` |
| `` `RFC_METADATA_GET could not resolve function ${name} (${failure})` `` | matching `FUNC_ERRORS` row | `src/metadata/rfc-metadata-get.ts:763-766` |
| `` `RFC_METADATA_GET returned ${matches.length} identities for function ${name}` `` | ≠1 identity row | `src/metadata/rfc-metadata-get.ts:768-772` |
| `` `${path}.PARAMCLASS is unsupported` `` | outside `[IECXT]` | `src/metadata/rfc-metadata-get.ts:787-790` |
| `` `RFC_METADATA_GET returned duplicate parameter ${parameterName}` `` | duplicate parameter | `src/metadata/rfc-metadata-get.ts:793-795` |
| `"RFC_METADATA_GET recursive metadata returned a foreign function error"` | any residual `FUNC_ERRORS` row | `src/metadata/rfc-metadata-get.ts:946-953` |
| `RecursiveMetadataError("REMOTE_DDIC_RESOLUTION_ERRORS", `DD_ERRORS:${ddicErrors.length}`)` | `DD_ERRORS` non-empty and not the UTCLONG shape | `src/metadata/rfc-metadata-get.ts:954-965` |
| `` `RFC_METADATA_GET recursive metadata identity does not match function ${name}` `` | graph identity mismatch | `src/metadata/rfc-metadata-get.ts:989-994` |
| `` `RFC_METADATA_GET recursive metadata generation does not match function ${name}` `` | graph token ≠ flat token | `src/metadata/rfc-metadata-get.ts:995-999` |
| `` `RFC_METADATA_GET could not resolve structure ${name} (${failure})` `` | matching `DD_ERRORS` row | `src/metadata/rfc-metadata-get.ts:1012-1015` |
| `` `RFC_METADATA_GET returned no type rows for structure ${name}` `` | zero matching rows | `src/metadata/rfc-metadata-get.ts:1024-1026` |
| `` `RFC_METADATA_GET structure ${name} exceeds ${MAX_STRUCTURE_FIELDS} fields` `` (`RangeError`) | >9999 rows | `src/metadata/rfc-metadata-get.ts:1027-1031` |
| `` `RFC_METADATA_GET structure ${name} has inconsistent lengths` `` | differing `TABLENGTH_UC` | `src/metadata/rfc-metadata-get.ts:1042-1044` |
| `` `RFC_METADATA_GET structure ${name} has inconsistent timestamps` `` | differing `TIMESTAMP` | `src/metadata/rfc-metadata-get.ts:1048-1052` |
| `` `RFC_METADATA_GET structure ${name} has duplicate field ${fieldName}` `` | duplicate field | `src/metadata/rfc-metadata-get.ts:1055-1057` |
| `` `RFC_METADATA_GET structure ${name}.${fieldName} requires a negotiated recursive serializer` `` | composite `COMPTYPE` or `INTTYPE` | `src/metadata/rfc-metadata-get.ts:1066-1073` |
| `` `RFC_METADATA_GET structure ${name}.${fieldName} has invalid geometry` `` | unsafe end, overlap, or overrun | `src/metadata/rfc-metadata-get.ts:1077-1083` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Keep the recursive wrapper aligned with recursive-metadata's default total row budget so the broader flat normalizer cannot allocate first." | `src/metadata/rfc-metadata-get.ts:15-16` |
| "Classic bootstrap for SAP's bounded metadata-generation lookup. Its exact four TABLES parameters are available on the beta's 7.50 and 7.58 lines and avoid loading a descriptor merely to decide whether a cached one is stale." | `src/metadata/rfc-metadata-get.ts:262-266` |
| "Captured request identities used to validate the asynchronous response." | `src/metadata/rfc-metadata-get.ts:283` |
| "Snapshot one bounded timestamp batch before asynchronous metadata I/O." | `src/metadata/rfc-metadata-get.ts:367` |
| "One optimized function descriptor and the generation observed in the same RFC_METADATA_GET response. Keeping these values inseparable prevents a descriptor/timestamp race between two backend calls." | `src/metadata/rfc-metadata-get.ts:535-539` |
| "A complete function type closure and the function generation captured by the same RFC_METADATA_GET response." | `src/metadata/rfc-metadata-get.ts:551-554` |
| "Normalize a complete timestamp batch. Every requested object must have exactly one success or typed error outcome; foreign rows cannot poison a structural cache and localized backend text is deliberately discarded." | `src/metadata/rfc-metadata-get.ts:597-601` |
| "RFC_METADATA_GET reports the Unicode byte width for fixed character-like scalar parameters. RfcFunintParameter, like RFC_GET_FUNCTION_INTERFACE, stores their logical character width; the invocation codec applies the Unicode factor when it writes them. Other scalar and structure lengths are already byte widths and must remain unchanged." | `src/metadata/rfc-metadata-get.ts:725-729` |
| "RFC_METADATA_GET legitimately emits zero and duplicate positions. The response row order remains authoritative for ties." | `src/metadata/rfc-metadata-get.ts:797-798` |
| "Some S/4 releases report the built-in UTCLONG scalar as an unresolved DDIC object even though PARAMETERS already carries its complete classic scalar codec. Admit only that one observed, self-contained shape. This is deliberately not a general \"ignore DD_ERRORS\" escape hatch: any field lookup, different exception, incomplete parameter, or contradictory DDIC row remains a hard failure." | `src/metadata/rfc-metadata-get.ts:854-859` |
| "Normalize a DEEP function response without splitting descriptor and generation reads. The flat normalizer first validates SAP's function-error table and exact identity; the recursive normalizer then validates the full bounded type graph from those same captured rows." | `src/metadata/rfc-metadata-get.ts:932-937` |
| "A matching error was already projected by the flat normalizer. Any remaining row therefore belongs to an unrequested function identity." | `src/metadata/rfc-metadata-get.ts:948-949` |
| "A partial type closure is never safe to cache or flatten. Do not retain localized backend text from the error rows in the public failure." | `src/metadata/rfc-metadata-get.ts:959-960` |
| "COMPTYPE is DDIC's declaration classification, not the wire type: per SAP Note 1691982 the initial value and \"E\" are both elementary, the initial form being emitted for components declared with a built-in DDIC type. decodeDdIfFieldInfoGetResult already admits both. INTTYPE below still routes every composite (u/h/v) to the recursive serializer, and the geometry and codec validation further down stay unchanged." | `src/metadata/rfc-metadata-get.ts:1060-1065` |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| The 12 bootstrap parameters and 7 bootstrap structure byte lengths are pinned exactly (`RFCFUNCTIONNAME 90`, `RFC_MD_DDIC_NAME 120`, `RFC_METADATA_PARAMS 464`, `RFC_METADATA_DDIC 424`, `RFC_METADATA_DDIC_INDIRECT 180`, `RFC_FUNC_ERROR 630`, `RFC_DD_ERROR 690`), and the bootstrap is frozen | `"pins the classic RFC_METADATA_GET bootstrap to Note 1456826 geometry"` | `test/rfc-metadata-get.test.ts:19-53` |
| Function and structure invocations emit exactly the base input with one populated table; `""` name and `"EN"` language are rejected | `"builds bounded function and structure metadata requests"` | `test/rfc-metadata-get.test.ts:55-94` |
| Timestamp bootstrap pins 4 parameters (`FUNC_ERRORS`/`DD_ERRORS` optional) and byte lengths `88, 88, 630, 690`; caller arrays are snapshotted (post-call mutation has no effect); duplicates and 513 names are rejected | `"pins and snapshots the classic RFC_METADATA_GET_TIMESTAMP contract"` | `test/rfc-metadata-get.test.ts:96-156` |
| Tokens are `"function:20260716:010203"` and `"structure:20260716010203"`; `EXCEPTION_TEXT` never appears in the result (`JSON.stringify(result).includes("private backend") === false`) | `"normalizes complete timestamp batches without retaining backend text"` | `test/rfc-metadata-get.test.ts:158`, `:185-204` |
| Missing outcome, unrequested name, duplicate outcome, malformed `UTIME`, over-cap rows (`/at most 1 rows/u`), sparse arrays, and an accessor row (never invoked, `hostileRowGetterCalled === false`) all reject | `"rejects incomplete, foreign, duplicate, and malformed timestamp batches"` | `test/rfc-metadata-get.test.ts:212-297` |
| The identity row is snapshotted before the token is built (post-call `UDAT` mutation does not change `generationToken`); malformed `UDAT` and inconsistent structure timestamps reject | `"binds optimized descriptors to generations from the same metadata response"` | `test/rfc-metadata-get.test.ts:299`, `:311-314`, `:340-386` |
| The recursive result's `generationToken` equals the graph identity token; foreign identity, foreign `DD_ERRORS` (code `REMOTE_DDIC_RESOLUTION_ERRORS`, path `"DD_ERRORS:1"`, text not leaked), foreign `FUNC_ERRORS`, and 20 001 `PARAMETERS` rows (`/recursive metadata must contain at most 20000 total rows/u`) all reject | `"binds a recursive DEEP graph to its same-response function generation"` | `test/rfc-metadata-get.test.ts:389`, `:459-466`, `:468-508` |
| The UTCLONG DDIC-miss shape is admitted and yields `{ kind: "scalar", internalType: "p" }`; 13 near-miss variants are each rejected with `REMOTE_DDIC_RESOLUTION_ERRORS` and no leaked text | `"admits only metadata-complete built-in UTCLONG scalar DDIC misses"` | `test/rfc-metadata-get.test.ts:511`, `:547-556`, `:571-612` |
| `EXID "C"` with `INTLENGTH 40` becomes `internalLength: 20` (halved); `PARAMCLASS "X"` becomes an entry in `exceptions`; `INTLENGTH 3` on `"C"` rejects `/odd Unicode byte width/u`; the resolve failure message is exactly `"RFC_METADATA_GET could not resolve function MISSING (FU_NOT_FOUND)"` and omits `EXCEPTION_TEXT` | `"normalizes optimized function rows without merging raw error text"` | `test/rfc-metadata-get.test.ts:615`, `:657-678`, `:681-721` |
| `POSITION` `0`, string `"0"`, and duplicates are accepted in source order; `-1`, `"-1"`, `"invalid"`, `1.5` reject `/POSITION must be a non-negative safe integer/u` | `"accepts zero parameter positions and preserves optimized row order for ties"` | `test/rfc-metadata-get.test.ts:724`, `:759-777` |
| The flat structure normalizer uses only the `_UC` columns (`INTLEN_UC "000020"` → `internalLength: 20`, `TABLENGTH_UC "000024"` → `byteLength: 24`), and `COMPTYPE "S"` / `INTTYPE "u"` rejects `/requires a negotiated recursive serializer/u` | `"normalizes flat optimized type rows and rejects recursive classic geometry"` | `test/rfc-metadata-get.test.ts:780`, `:820-863` |
| `COMPTYPE` `""` and `"E"` decode identically; `["S","C"]`, `["L","C"]`, `["R","C"]`, `["","u"]`, `["","h"]`, `["","v"]`, `["E","u"]` all reject | `"admits both elementary COMPTYPE spellings and still refuses composites"` | `test/rfc-metadata-get.test.ts:866`, `:899-922` |

### Go mapping notes

- The `INTLENGTH` halving applies **only** to `EXID` in `{"C","N","D","T"}`
  (`src/metadata/rfc-metadata-get.ts:730`) and is asserted by a test
  (`test/rfc-metadata-get.test.ts:672`, `:704`). It is a normalization, not a
  wire fact — the raw wire value is the Unicode byte width.
- The flat structure normalizer reads exclusively `TABLENGTH_UC`, `OFFSET_UC`,
  `INTLEN_UC` (`:1040`, `:1074-1075`) while the recursive normalizer keeps both
  NUC and UC (`src/metadata/recursive-metadata.ts:583-591`). Do not collapse
  them.
- Both `rows()` (`:411-436`) and `dataProperty()` (`:394-409`) exist to defeat
  JS accessor properties and sparse arrays; in Go, decoding from `[]byte` or a
  typed struct removes the entire class. Keep the *count* bounds
  (`MAX_METADATA_ROWS`, `MAX_RECURSIVE_METADATA_ROWS`, `MAX_STRUCTURE_FIELDS`,
  `MAX_TIMESTAMP_NAMES_PER_KIND`), drop the accessor defences.
- `matchingError()` calls `rows(output, tableName)` with the **default** cap of
  100 000 (`:743`), whereas `timestampErrorRows()` caps at `requested.size`
  (`:569`). These are two different bounds on the same tables.

---

## src/metadata/rfc-structure-definition.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RFC_FIELDS_UNICODE_ROW_LENGTH` | const | `export const RFC_FIELDS_UNICODE_ROW_LENGTH = 138;` | `src/metadata/rfc-structure-definition.ts:14` |
| `RfcStructureField` | interface | `readonly tableName: string; readonly fieldName: string; readonly position: number; readonly offset: number; readonly internalLength: number; readonly decimals: number; readonly exid: string;` | `src/metadata/rfc-structure-definition.ts:17-25` |
| `RfcStructureDefinition` | interface | `readonly name: string; readonly byteLength: number; readonly fields: readonly RfcStructureField[];` | `src/metadata/rfc-structure-definition.ts:27-31` |
| `buildRfcGetStructureDefinitionRequest` | function | `export function buildRfcGetStructureDefinitionRequest(structureName: string,): Buffer` | `src/metadata/rfc-structure-definition.ts:34-36` |
| `decodeRfcFieldsRow` | function | `export function decodeRfcFieldsRow(value: Uint8Array): RfcStructureField` | `src/metadata/rfc-structure-definition.ts:51` |
| `decodeRfcStructureDefinitionResult` | function | `export function decodeRfcStructureDefinitionResult(structureName: string, fields: readonly CpicField[],): RfcStructureDefinition` | `src/metadata/rfc-structure-definition.ts:98-101` |

### Constants, type codes, and enumerations

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `RFC_FIELDS_UNICODE_ROW_LENGTH` | `138` | minimum/consumed prefix of one `RFC_FIELDS` row | `src/metadata/rfc-structure-definition.ts:14` |
| `MAX_RFC_STRUCTURE_FIELDS` | `20_000` | maximum `FIELDS` rows (checked from the table header before decoding) | `src/metadata/rfc-structure-definition.ts:15`, use `:109` |
| requested outputs | `requestedOutputs: ["TABLENGTH", "FIELDS"]` | | `src/metadata/rfc-structure-definition.ts:39` |
| import `TABNAME` | `[{ name: "TABNAME", value: encodeAbapChar(structureName, 30) }]` | | `src/metadata/rfc-structure-definition.ts:40` |
| row layout | `TABNAME` 60 bytes/30 chars; `FIELDNAME` 60 bytes/30 chars; `POSITION` `readInt32LE`; `OFFSET` `readInt32LE`; `INTLENGTH` `readInt32LE`; `DECIMALS` `readInt32LE`; `EXID` 2 bytes/1 char — read sequentially, total 138 | little-endian int32s | `src/metadata/rfc-structure-definition.ts:62-71` |
| `TABLENGTH` width | `if (lengthValue.byteLength !== 4)` | must be INT4 | `src/metadata/rfc-structure-definition.ts:118-120` |

### Errors

| Message (verbatim) | Trigger | Citation |
|---|---|---|
| `` `Unicode RFC_FIELDS row must contain at least ${RFC_FIELDS_UNICODE_ROW_LENGTH} bytes; received ${value.byteLength}` `` (`RangeError`) | short row | `src/metadata/rfc-structure-definition.ts:52-57` |
| `"RFC_FIELDS row contains an empty table or field name"` | empty `TABNAME`/`FIELDNAME` | `src/metadata/rfc-structure-definition.ts:72-74` |
| `"RFC_FIELDS row contains a negative or invalid numeric property"` | `position < 1`, or any of `offset`/`internalLength`/`decimals` negative | `src/metadata/rfc-structure-definition.ts:75-82` |
| `` `RFC_GET_STRUCTURE_DEFINITION response lacks scalar ${name}` `` | scalar missing | `src/metadata/rfc-structure-definition.ts:92` |
| `` `RFC_GET_STRUCTURE_DEFINITION FIELDS must contain at most ${MAX_RFC_STRUCTURE_FIELDS} rows` `` (`RangeError`) | header row count > 20 000 | `src/metadata/rfc-structure-definition.ts:109-114` |
| `"RFC_GET_STRUCTURE_DEFINITION TABLENGTH must be INT4"` | scalar not 4 bytes | `src/metadata/rfc-structure-definition.ts:118-120` |
| `"RFC_GET_STRUCTURE_DEFINITION returned negative TABLENGTH"` | negative length | `src/metadata/rfc-structure-definition.ts:122-124` |
| `"RFC_GET_STRUCTURE_DEFINITION response lacks FIELDS table"` | table missing | `src/metadata/rfc-structure-definition.ts:126-128` |
| `` `RFC_GET_STRUCTURE_DEFINITION FIELDS row width is ${fieldTable.rowByteLength}; expected at least ${RFC_FIELDS_UNICODE_ROW_LENGTH}` `` | narrow rows | `src/metadata/rfc-structure-definition.ts:129-134` |
| `` `RFC_FIELDS ${field.fieldName} has position ${field.position}; expected ${expectedPosition}` `` | non-sequential position | `src/metadata/rfc-structure-definition.ts:141-146` |
| `` `RFC_FIELDS ${field.fieldName} belongs to ${field.tableName}; expected ${structureName}` `` | foreign row | `src/metadata/rfc-structure-definition.ts:147-152` |
| `` `RFC_FIELDS contains duplicate field ${field.fieldName}` `` | duplicate name | `src/metadata/rfc-structure-definition.ts:153-155` |
| `` `RFC_FIELDS ${field.fieldName} overlaps its preceding field` `` | `offset < previousEnd` | `src/metadata/rfc-structure-definition.ts:156-158` |
| `` `RFC_FIELDS ${field.fieldName} ends at ${end} beyond structure length ${byteLength}` `` | unsafe or overrunning end | `src/metadata/rfc-structure-definition.ts:159-164` |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Build the classic structure-metadata bootstrap call." | `src/metadata/rfc-structure-definition.ts:33` |
| "Decode one Unicode RFC_FIELDS bootstrap row." | `src/metadata/rfc-structure-definition.ts:45` |
| "As with RFC_FUNINT, the row width belongs to the peer's release rather than to the wire format, so bound it below by the stable prefix this decoder consumes and ignore appended fields. A short row is still refused." | `src/metadata/rfc-structure-definition.ts:47-49` |
| "Normalize and validate RFC_GET_STRUCTURE_DEFINITION output." | `src/metadata/rfc-structure-definition.ts:97` |

### Wire facts asserted by tests

None of the four in-scope tests import this file. `RfcStructureField` /
`RfcStructureDefinition` shapes are asserted indirectly through
`normalizeRfcMetadataGetStructure` (`test/rfc-metadata-get.test.ts:820-843`,
`:884-896`).

### Go mapping notes

- The row-count bound is checked **before** decoding, from the CPIC table
  header, by scanning the raw field list for `CpicTag.TableName === "FIELDS"`
  followed by `CpicTag.TableHeader` (`src/metadata/rfc-structure-definition.ts:102-115`).
  Port this pre-check, not just the post-decode length.
- Unlike DDIF, these integers are little-endian (`reader.readInt32LE(...)`,
  `src/metadata/rfc-structure-definition.ts:65-68`).
- `decodeRfcFieldsRow` slices to exactly 138 bytes before reading and then calls
  `reader.finish()` (`:58-61`, `:71`), i.e. trailing release-added bytes are
  ignored but the consumed prefix must be fully consumed.

---

## Immutability and caching model

The architecture doc's two claims are implemented as follows.

### "Normalized metadata descriptors are immutable"

Immutability is achieved by **three distinct mechanisms**, all shallow-freeze
based; there is no deep-freeze helper anywhere in this layer.

**1. Exhaustive per-object `Object.freeze` at construction.**
`normalizeRecursiveMetadataGraph` freezes every object it produces on the way
out: each field reference (`src/metadata/recursive-metadata.ts:1363-1366`,
`:1370-1376`), each field (`:1378-1392`), each node's `fields` array (`:1400`)
and the node itself (`:1394-1401`), each map entry pair (`:1402`), each
parameter (`:1441-1455`) and the parameter array (`:1415`), each cycle
(`:1466-1471`) and the cycle array (`:1473`), the statistics record (`:1591`),
the roots array (`:1588`), and finally the graph itself (`:1583`). The `limits`
object is frozen either as `DEFAULT_LIMITS` (`:133`) or by `safeLimitOptions`
(`:499`). The same pattern holds in `rfc-function-interface.ts:98`, `:102`,
`:116-117`; `rfc-structure-definition.ts:168-174`; `ddif-fieldinfo.ts:148`,
`:197`, `:275`, `:285-289`; `rfc-metadata-get.ts:77`, `:102`, `:118-122`,
`:643-648`, `:675-679`, `:805`, `:825-833`, `:836-839`.

**2. `ImmutableMetadataMap` for the map-valued members.** Maps cannot be frozen
usefully in JS: "`Object.freeze(new Map())` is still mutable through
Map.prototype.set()" (`src/metadata/immutable-map.ts:4`). The class keeps its
backing store in `readonly #values: Map<K, V>`
(`src/metadata/immutable-map.ts:10`), calls `Object.freeze(this)` in the
constructor (`:14`), freezes the prototype at module load (`:36`), and declares
no `set`/`delete`/`clear` (`:9-34`). `RecursiveMetadataGraph.nodes` is one of
these (`finalNodes` returns `new ImmutableMetadataMap(entries)`,
`src/metadata/recursive-metadata.ts:1404`), as is the internal indirect-type map
(`:961`). `rfc-metadata-get.ts` uses a **separate private class of the same
shape**, also named `ImmutableMap` (`src/metadata/rfc-metadata-get.ts:23-46`),
for `RFC_METADATA_GET_BOOTSTRAP.structures` (`:217`),
`RFC_METADATA_GET_TIMESTAMP_BOOTSTRAP.structures` (`:270`), and all four maps of
`RfcMetadataTimestampBatch` (`:712-717`).

**3. A validating walk at the adapter boundary.** Everything an adapter returns
is checked by `assertBoundedRecursivelyFrozen`
(`src/metadata/repository-runtime.ts:473-598`), reached from `normalizeSnapshot`
(`:626`). Per the doc comment: "Must be recursively frozen before it crosses the
adapter boundary." (`:93`). It rejects, in order: `Proxy` nodes (`:487-491`),
functions (`:492-493`), an `ImmutableMetadataMap` that is somehow not frozen
(`:496-498`), any `Map`/`Set`/`Date`/`ArrayBuffer`/`ArrayBuffer.isView` value
(`:534-541`), any prototype other than `Object.prototype` or `Array.prototype`
(`:542-545`), extensible objects (`:546-548`), and any configurable, writable,
or accessor property descriptor (`:561-570`). Keys come from
`Reflect.ownKeys(current.value)` (`:549`), so symbol-keyed properties are
inspected too — asserted by
`test/metadata-repository-runtime.test.ts:1057`. `ImmutableMetadataMap`
instances are recognized only through `isImmutableMetadataMap`, which demands
exact prototype identity (`src/metadata/immutable-map.ts:42-43`) — a subclass
fails, asserted at `test/metadata-repository-runtime.test.ts:1021-1049`. Their
contents are read through `immutableMetadataMapEntries`, which freezes the
returned array and each `[key, value]` pair
(`src/metadata/immutable-map.ts:50-52`).

**Trust registries.** Two side tables record "this object came from our own
bounded constructor":
`const normalizedRecursiveMetadataGraphs = new WeakSet<object>()`
(`src/metadata/recursive-metadata.ts:111`), written only at `:1600` and read by
`isNormalizedRecursiveMetadataGraph` (`:114-119`); and
`const INDEX_STATE = new WeakMap<object, RecursiveMetadataParameterIndexState>()`
(`src/metadata/recursive-parameter-index.ts:45-48`), written only at `:146`.

**Where immutability is deliberately shallow.**
`RecursiveMetadataParameterIndexState` is frozen (`:146`) but its `work` record
is mutated in place: `state.work[kind] += 1`
(`src/metadata/recursive-parameter-index.ts:222`); and its `caches` is a live
`Map` (`:24`, `:206-211`).

### "Repository calls use their own logical execution lane"

**`AsyncLocalStorage` appears exactly once in this layer**, at
`src/metadata/repository-runtime.ts:1` (import) and `:236-237`:

```
const metadataRetirementContext =
  new AsyncLocalStorage<MetadataRetirementContext>();
```

What it carries is one field:

```
interface MetadataRetirementContext {
  readonly owners: ReadonlySet<object>;
}
```
(`src/metadata/repository-runtime.ts:225-227`).

It carries **retirement ownership only** — no lookup, no key, no principal, no
cancellation token. Its whole lifecycle is inside `retire()`
(`src/metadata/repository-runtime.ts:936-962`):

- `const inheritedContext = metadataRetirementContext.getStore();` (`:937`)
- `if (inheritedContext?.owners.has(this) === true) return this.#reentrantRetirementAcknowledgement;` (`:938-940`)
- `const owners = new Set(inheritedContext?.owners); owners.add(this);` (`:942-943`)
- `metadataRetirementContext.run(context, () => { this.#retirementController.abort(canceledWait()); });` (`:957-959`)

That is, the store exists so a cooperative adapter's abort listener can call
`retire()` again — synchronously or after an `await` — without deadlocking:
"Publish before abort listeners run: a cooperative adapter may reenter retire()
synchronously from its generation-retirement signal." (`:953-954`). The
behaviour is asserted by
`"abort cleanup may await owning repository retirement without deadlock"`
(`test/metadata-repository-runtime.test.ts:2047`, reentrancy at `:2065-2077`,
distinct promise at `:2101`).

**INFERRED:** the architecture doc's "logical execution lane" therefore does
*not* correspond to an ambient context carrying a repository handle in this
code; the repository is passed explicitly as `this`, and the only ambient state
is retirement ownership.

**The other lane mechanism is a per-generation `AbortController`.**
`readonly #retirementController = new AbortController()` (`:714`); its `signal`
is the *only* signal handed to the adapter, in `MetadataProbeContext` (`:1112`)
and `MetadataAccessContext` (`:1214`, `:1305`), documented as "Aborted only when
the owning repository generation is retired." (`:101`, `:110`). A caller's own
`AbortSignal` is bound separately by `bindCallerSignal` (`:246-278`) and
composed by `waitForCaller` (`:280-316`), which releases only that waiter.
Asserted by `assert.notEqual(physicalSignal, caller.signal)` and
`assert.equal(physicalSignal?.aborted, false)` in
`"caller cancellation releases only that waiter and preserves a shared physical load"`
(`test/metadata-repository-runtime.test.ts:1770`, `:1779`).

**Draining.** `#activeOperations = new Set<Promise<unknown>>()` (`:713`),
populated by `#trackOperation` (`:1343-1349`); `retire()` awaits
`while (this.#activeOperations.size > 0) { await Promise.allSettled([...this.#activeOperations]); }`
(`:949-951`).

### Cache keys

There are five distinct keyed structures in `MetadataRepositoryRuntime`
(`src/metadata/repository-runtime.ts:708-712`). All string keys are built with
`function tupleId(values: readonly string[]): string { return JSON.stringify(values); }`
(`:375-377`).

| Structure | Key expression (verbatim) | Contains a principal? | Citation |
|---|---|---|---|
| `#cache` (descriptor values) | `lookup.structural.id`, where `id: tupleId([backendKey, metadataGeneration, language, objectKind, objectName])` | **No** — "A descriptor identity which deliberately contains no authenticated principal." | `:708`, `:1168`, `:1449`; id `:421-427`; comment `:62` |
| `#probes` | `lookup.capabilityIdentity`, where `capabilityIdentity: tupleId([capability.id, mode])` and `capability.id: tupleId([backendKey, principalKey])` | Yes, plus mode | `:711`, `:1078`; `:660`; `:439` |
| `#authorizations` | `tupleId([lookup.capabilityIdentity, strategy, lookup.structural.id, epochIdentity(epoch)])` | Yes, plus mode, strategy, object, epoch | `:710`, `:1356-1361` |
| `#inFlight` | `` `${lookup.structural.id}\n${lookup.capabilityIdentity}\n${strategy}\n` + epochIdentity(admissionEpoch) `` | Yes, plus mode, strategy, epoch | `:709`, `:1191-1193` |
| `#objectEpochs` | `structuralId` → `bigint` | No | `:712`, `:1029-1049` |

`epochIdentity` is
`` `${epoch.global.toString(36)}:${epoch.object.toString(36)}` `` (`:351-353`),
over `interface OperationEpoch { readonly global: bigint; readonly object: bigint; }`
(`:214-217`).

Consequences the tests pin down: a descriptor cached by one principal is
returned to another (after that principal's own `authorize`) —
`assert.equal(fullFirst, restrictedSecond)`
(`test/metadata-repository-runtime.test.ts:811`); positive authorization is
never shared (`:848`); two principals get two physical loads
(`inFlight === 2`, `:646`); and `Auto` vs `OptimizedOnly` probe and authorize
separately (`:595-604`).

### Eviction and consistency

- `#cache` LRU: hit re-inserts (`delete` then `set`, `:1178-1179`); eviction
  takes `this.#cache.keys().next().value` (`:1438`) while
  `size >= maxEntries || retainedBytes > maxRetainedBytes - snapshot.retainedBytes`
  (`:1433-1437`).
- Snapshots larger than the whole budget are returned but not cached, and their
  authorizations are dropped (`:1424-1431`).
- `#probes` and `#authorizations` evict only **settled** entries (those with
  `candidate.promise === undefined`), else they throw a capacity error
  (`:1091-1105`, `:1279-1296`).
- `invalidate` deletes the entry, advances the object epoch, forgets that
  object's authorizations, and drops matching `#inFlight` keys by
  `key.startsWith(`${canonicalStructural.id}\n`)` (`:895-914`).
- Exhausting `#objectEpochs` triggers a global compaction: `#globalEpoch += 1n`
  plus clearing epochs, authorizations, and in-flight entries (`:1044-1048`),
  with the rationale quoted at `:1041-1043`.

---

## Recursion and allocation bounds

Every bound found in scope, with the exceeded-behaviour.

### src/metadata/recursive-metadata.ts

| Bound | Value / expression (verbatim) | On exceed | Citation |
|---|---|---|---|
| `maxRows` default / absolute | `20_000` / `100_000` | `reject("ROW_LIMIT", path)` → `RecursiveMetadataError` | `:134`, `:143`; `:367`, `:387` |
| `maxNodes` default / absolute | `4_096` / `20_000` | `reject("NODE_LIMIT", ...)` | `:135`, `:144`; `:535`, `:785` |
| `maxEdges` default / absolute | `20_000` / `100_000` | `reject("EDGE_LIMIT", ...)` | `:136`, `:145`; `:866`, `:1113`, `:1150`, `:1191` |
| `maxDepth` default / absolute | `64` / `256` | `reject("DEPTH_LIMIT", "metadata-graph")` | `:137`, `:146`; `:1342` |
| `maxProperties` default / absolute | `400_000` / `2_000_000` | `reject("PROPERTY_LIMIT", ...)` | `:138`, `:147`; `:393-396` |
| `maxBytes` default / absolute | `8 * 1024 * 1024` / `32 * 1024 * 1024` | `reject("BYTE_LIMIT", ...)` | `:139`, `:148`; `:401-404` |
| any configured limit above its absolute | `candidate > ABSOLUTE_LIMITS[key]` | `reject("INVALID_LIMIT", ...)` | `:489-496` |
| pre-check before iterating an array | `const remainingRows = budget.limits.maxRows - budget.rows; if (value.length > remainingRows) reject("ROW_LIMIT", path);` | rejects **before** the loop | `:366-367` |
| every counter add | `if (!Number.isSafeInteger(next) \|\| next > limit)` | reject with the matching code | `:385`, `:393`, `:401` |
| depth measurement | longest path over the condensation DAG, `maximumDepth` | compared once at `:1342` | `:1328-1342` |

Depth is measured on **components**, not nodes: cyclic 2-node graph gives
`maximumDepth === 1` (`test/recursive-metadata.test.ts:569`), and a 2-level
acyclic graph gives `2` (`:152`).

**No function in this file calls itself.** Traversals use explicit stacks:
`:1251-1265` (iterative DFS with a `{node, next}` frame), `:1275-1286`,
`:1314-1319`, `:1331-1341`, `:1566-1573`; `resolveFieldPath` iterates segments
(`:974-987`).

### src/metadata/recursive-parameter-index.ts

| Bound | Value / expression (verbatim) | On exceed | Citation |
|---|---|---|---|
| `ABSOLUTE_MAX_PARAMETER_COUNT` | `100_000` | `RangeError` `` `recursive xRFC graph maxRows is outside 0..${ABSOLUTE_MAX_PARAMETER_COUNT}` `` | `:44`, `:92-101` |
| parameter count vs declared budget | `if (parameterCount > (declaredMaximum as number))` | `RangeError` `` `recursive xRFC graph exceeds its row budget ${declaredMaximum}` `` | `:102-106` |
| indexing loop bound | `for (let position = 0; position < parameterCount; position += 1)` where `parameterCount` came from the intrinsic `length` descriptor | — | `:91`, `:109` |

The cache (`state.caches`) has **no** size bound
(`src/metadata/recursive-parameter-index.ts:206-212`).

### src/metadata/repository-runtime.ts

| Bound | Value / expression (verbatim) | On exceed | Citation |
|---|---|---|---|
| `maxEntries` | caller-supplied, `0..Number.MAX_SAFE_INTEGER` | LRU eviction, `#evictions += 1` | `:762-767`, `:1433-1444` |
| `maxRetainedBytes` | caller-supplied, `0..Number.MAX_SAFE_INTEGER` | LRU eviction; snapshot larger than the budget → `#oversizeSkips += 1`, not cached | `:768-773`, `:1424-1444` |
| `MAX_AUXILIARY_ENTRIES` | `1_000_000` | `RangeError` from `boundedInteger` | `:230`, `:339` |
| derived probe / authorization / epoch / load limits | multipliers `2`/`4`/`2`/`2`, floors `16`/`32`/`32`/`8`, capped at `MAX_AUXILIARY_ENTRIES` | see below | `:331-349`, `:776-803` |
| active probes | `if (this.#activeProbes >= this.#maxProbeEntries)` | `#probeCapacityRejections += 1;` then `throw capacityError("probe", ...)` | `:1087-1090` |
| settled probes | `if (this.#probes.size >= this.#maxProbeEntries)` | evict one settled entry, else throw | `:1091-1105` |
| active authorizations | `if (this.#activeAuthorizations >= this.#maxAuthorizationEntries)` | `throw capacityError("authorization", ...)` | `:1275-1278` |
| settled authorizations | `if (this.#authorizations.size >= this.#maxAuthorizationEntries)` | evict one settled entry, else throw | `:1279-1296` |
| active loads | `if (this.#activeLoads >= this.#maxInFlightLoads)` | `throw capacityError("load", ...)` | `:1204-1207` |
| object epochs | `if (this.#objectEpochs.size < this.#maxObjectEpochEntries)` | global compaction: `#globalEpoch += 1n`, clear epochs/authorizations/in-flight, `#objectEpochCompactions += 1` | `:1036-1048` |
| `maxSnapshotNodes` | default `100_000`, range `1..MAX_AUXILIARY_ENTRIES` | `RangeError` `` `metadata snapshot graph exceeds node limit ${maxNodes}` `` | `:231`, `:804-810`, `:522-526`, `:584-589` |
| `maxSnapshotDepth` | default `256`, range `1..MAX_SNAPSHOT_DEPTH` (`4_096`) | `RangeError` `` `metadata snapshot graph exceeds depth limit ${maxDepth}` `` | `:232`, `:234`, `:811-817`, `:517-521`, `:579-583` |
| `maxSnapshotProperties` | default `1_000_000`, range `1..MAX_AUXILIARY_ENTRIES` | `RangeError` `` `metadata snapshot graph exceeds property limit ${maxProperties}` `` | `:233`, `:818-825`, `:500-504`, `:555-559` |
| sparse-array accounting | `const logicalProperties = Array.isArray(current.value) ? keys.length + current.value.length - arrayIndexPropertyCount(keys) : keys.length;` | counts logical slots, so `new Array(4_294_967_295)` trips the property limit | `:550-554`, `:456-471`; test `test/metadata-repository-runtime.test.ts:1179-1211` |
| identity string length | `value.length < 1 \|\| value.length > 512` | `RangeError` | `:361-372` |
| retained-byte arithmetic | `if (!Number.isSafeInteger(retainedBytes))` / `< 0` | `Error "metadata cache retained-byte sum is unsafe"` / `"...subtraction is unsafe"` | `:1446-1448`, `:1462-1465` |

`assertBoundedRecursivelyFrozen` is **iterative**: `const pending: Array<{
readonly value: object; readonly depth: number }> = [{ value: root, depth: 1 }];`
plus `while (pending.length > 0) { const current = pending.pop()!; ... }`
(`:480-486`). A 20 000-deep frozen chain therefore fails with the depth error
rather than a stack overflow
(`test/metadata-repository-runtime.test.ts:1214-1250`). Already-scheduled nodes
are skipped via `scheduled` (`:479`, `:578`), so a frozen self-cycle is accepted
(`test/metadata-repository-runtime.test.ts:1283-1299`).

### src/metadata/rfc-metadata-get.ts

| Bound | Value / expression (verbatim) | On exceed | Citation |
|---|---|---|---|
| `MAX_METADATA_ROWS` | `100_000` (default `rows()` cap) | `RangeError` `` `RFC_METADATA_GET output ${name} must contain at most ${maximum} rows` `` | `:14`, `:414-421` |
| `MAX_RECURSIVE_METADATA_ROWS` | `20_000` summed across `FUNCTIONNAMES`+`DATATYPESCONT`+`INDIRECTTYPES`+`PARAMETERS`, checked **incrementally** inside the loop | `RangeError` `` `RFC_METADATA_GET recursive metadata must contain at most ${MAX_RECURSIVE_METADATA_ROWS} total rows` `` | `:17`, `:438-459` |
| `MAX_STRUCTURE_FIELDS` | `9_999` | `RangeError` `` `RFC_METADATA_GET structure ${name} exceeds ${MAX_STRUCTURE_FIELDS} fields` `` | `:18`, `:1027-1031` |
| `MAX_TIMESTAMP_NAMES_PER_KIND` | `512` per kind | `RangeError` `` `RFC_METADATA_GET_TIMESTAMP accepts at most ${MAX_TIMESTAMP_NAMES_PER_KIND} ${kind} names` `` | `:19`, `:342-346` |
| timestamp success rows | `rows(output, "FUNCTION_TIMESTAMPS", requestedFunctions.length)` / `rows(output, "DDIC_TIMESTAMPS", requestedStructures.length)` | `RangeError` from `rows()` | `:617-621`, `:651-655` |
| timestamp error rows | `rows(output, tableName, requested.size)` | `RangeError` from `rows()` | `:569` |
| text field maxima | default `255`; `TABNAME`/`FIELDNAME`/`PARAMETER`/`EXCEPTION` 30; `PARAMCLASS`/`EXID`/`OPTIONAL`/`BASXML_SUPPORTED` 1; `DEFAULT` 21; `PARAMTEXT` 79; `TIMESTAMP` 14; `UDAT` 8; `UTIME` 6 | `Error` `` `${path}.${name} contains invalid text` `` | `:466`, `:808-820`, `:834-835`, `:1045` |

`assertRecursiveMetadataRowBudget` runs **before** the flat normalizer
(`:944`) — the comment at `:15-16` states the reason: "so the broader flat
normalizer cannot allocate first."

### src/metadata/ddif-fieldinfo.ts and rfc-structure-definition.ts

| Bound | Value (verbatim) | On exceed | Citation |
|---|---|---|---|
| `DFIES_MINIMUM_UNICODE_ROW_LENGTH` | `1_074` (also the max copied per row) | `RangeError` | `ddif-fieldinfo.ts:20`, `:117-124`, `:128-133` |
| `X030L_MINIMUM_UNICODE_LENGTH` | `249` | `RangeError` | `ddif-fieldinfo.ts:21`, `:164-168` |
| `MAX_DDIC_STRUCTURE_FIELDS` | `9_999` — checked twice: against the advertised X030L count and the actual row count | `RangeError` | `ddif-fieldinfo.ts:22`, `:187-191`, `:223-227` |
| `RFC_FIELDS_UNICODE_ROW_LENGTH` | `138` (minimum and consumed prefix) | `RangeError` | `rfc-structure-definition.ts:14`, `:52-57`, `:59` |
| `MAX_RFC_STRUCTURE_FIELDS` | `20_000` — checked from the CPIC table header before decoding | `RangeError` | `rfc-structure-definition.ts:15`, `:102-115` |

### src/metadata/immutable-map.ts

No bounds. `immutableMetadataMapEntries` materializes `[...value].map(...)`
(`:51`) — an unbounded copy proportional to map size. The bound comes from the
caller: `assertBoundedRecursivelyFrozen` checks
`entries.length > Math.floor((maxProperties - inspectedProperties) / 2)`
**after** materializing (`src/metadata/repository-runtime.ts:499-504`).

---

## Open questions for the porter

1. **`ImmutableMap` is duplicated.** `src/metadata/rfc-metadata-get.ts:23-46`
   declares a private class identical in shape to
   `src/metadata/immutable-map.ts:9-34` but is a different class, so
   `isImmutableMetadataMap` (`src/metadata/immutable-map.ts:42-43`) rejects it.
   Whether that is intentional isolation or drift is not stated in source.
   `RFC_METADATA_GET_BOOTSTRAP.structures` therefore could not pass through
   `assertBoundedRecursivelyFrozen` as a snapshot value. Is one Go type correct,
   or must the distinction survive?

2. **`RFC_FUNINT_UNICODE_ROW_LENGTH` has no value in scope.** It is imported
   from `../protocol/classic-rfc.js` (`src/metadata/rfc-function-interface.ts:2`)
   and only compared, never printed with a literal
   (`src/metadata/rfc-function-interface.ts:80-84`). The comment mentions "a
   404-byte declaration is already evidenced" and "an explicit 402/404
   exception" (`:74-78`), but the constant's actual value is **not stated in
   source**. Needed before porting.

3. **DDIC type-code semantics are undocumented here.** The meaning of `EXID` /
   `INTTYPE` letters (`u`, `v`, `h`, `C`, `N`, `D`, `T`, `p`, `y`, `g`, `I`,
   …), `COMPTYPE` letters (`""`, `E`, `S`, `L`, `R`, `T`), `DDOBJTYPE` values
   (`DTEL`, `TTYP`), and X030L `tableType` `"L"` are only partially described:
   `u`/`v` = flat/deep structure, `h` = table
   (`src/metadata/recursive-metadata.ts:745-756`), `""`/`E` = elementary
   (`src/metadata/ddif-fieldinfo.ts:250-254`), `L` = "a table/vector type"
   (`src/metadata/ddif-fieldinfo.ts:194`). Everything else is **not stated in
   source**; the exhaustive scalar mapping lives in `src/compat/modern-metadata.ts`
   (asserted at `test/modern-recursive-metadata.test.ts:435-482`), which
   `../provenance.md` marks "Not ported, deliberately".

4. **Go maps have no iteration order.** `MetadataRepositoryRuntime` relies on JS
   `Map` insertion order for LRU eviction (`:1438`), settled-entry eviction
   (`:1093`, `:1281`), and object-epoch refresh (`:1032-1033`). Two tests assert
   the resulting order is deterministic
   (`test/metadata-repository-runtime.test.ts:653`, `:1443`). Which ordered
   structure the Go port uses is a design decision this inventory cannot make.

5. **Concurrency model.** The runtime has no lock and every capacity check is a
   non-atomic check-then-act (`src/metadata/repository-runtime.ts:1087-1108`,
   `:1204-1217`, `:1275-1299`). Does the Go port serialize the whole runtime
   behind one mutex, or is the intended target one runtime per connection lane?
   Not stated in source.

6. **`retainedBytes` is adapter-supplied and unverified** beyond
   `boundedInteger(retainedBytes, 0, Number.MAX_SAFE_INTEGER, "metadata retainedBytes")`
   (`src/metadata/repository-runtime.ts:620-625`). Nothing measures the actual
   snapshot size. The Go equivalent of `Number.MAX_SAFE_INTEGER` (2^53−1) is not
   `math.MaxInt64`; picking the wrong ceiling changes the eviction test at
   `test/metadata-repository-runtime.test.ts:1410-1440`.

7. **`epochIdentity` uses base-36 bigints** (`src/metadata/repository-runtime.ts:351-353`).
   With `uint64` the string form differs, which changes nothing functionally but
   does change any test that pins a key string. No in-scope test pins one.

8. **The four in-scope tests cover only three of the eight source files.**
   `ddif-fieldinfo.ts`, `rfc-function-interface.ts`,
   `rfc-structure-definition.ts`, and `recursive-parameter-index.ts` have no
   in-scope test. Whether coverage exists in other upstream test files is
   outside this inventory's scope and should be checked before porting them.
