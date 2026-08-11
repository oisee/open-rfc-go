# Surface inventory: src/values/

> Mechanical inventory of open-rfc @ commit 847036d, generated as porting input. Every claim cites path:line. See ../provenance.md.

Scope: the twelve files in `src/values/` and the in-scope tests. Constants imported from
`../protocol/cpic.js` (`DEFAULT_MAX_CPIC_FIELD_LENGTH`, `DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH`,
`DEFAULT_MAX_CPIC_FIELD_COUNT`) and from `../protocol/bytes.js`
(`intrinsicUint8ArrayByteLength`, `snapshotUint8Array`) are referenced by these files but their
values/bodies are **not stated in source within this scope**.

---

## classic-bcd.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `ClassicBcdConverter` | type | `export type ClassicBcdConverter = (value: string) => unknown;` | src/values/classic-bcd.ts:2 |
| `ClassicBcdMode` | type | `export type ClassicBcdMode = "string" \| "number" \| ClassicBcdConverter;` | src/values/classic-bcd.ts:5 |
| `ClassicBcdConversionError` | class | `export class ClassicBcdConversionError extends Error` with `readonly path: string;` and `constructor(path: string, cause: unknown)` | src/values/classic-bcd.ts:12-19 |
| `snapshotClassicBcdMode` | function | `export function snapshotClassicBcdMode(\n  value: unknown,\n  label = "bcd",\n): ClassicBcdMode` | src/values/classic-bcd.ts:23-26 |
| `projectClassicBcdOutput` | function | `export function projectClassicBcdOutput(\n  value: string,\n  mode: ClassicBcdMode,\n  path = "BCD",\n): unknown` | src/values/classic-bcd.ts:39-43 |

### Numeric and format constants

None declared in this file.

### Errors

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `` `BCD output conversion failed at ${path}` `` (with `{ cause }`), `name = "ClassicBcdConversionError"` | caller-supplied converter threw | src/values/classic-bcd.ts:16-17, 48-50 |
| `` `${label} must be "string", "number", or a function` `` (TypeError) | value is not `undefined`, not `"string"`, not `"number"`, not a function | src/values/classic-bcd.ts:27-31 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Marks an exception thrown by caller-owned conversion code after the complete RFC reply has already been consumed. Transport layers use this distinction to keep the otherwise healthy connection reusable." | src/values/classic-bcd.ts:7-11 |
| "Capture the archived node-rfc BCD option without invoking caller code." | src/values/classic-bcd.ts:22 |
| "Project one already-validated canonical decimal string. Custom converters are ordinary function calls, matching the archived binding; constructor-only ES classes therefore fail in the same way as any other non-callable hook." | src/values/classic-bcd.ts:34-38 |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| `snapshotClassicBcdMode(undefined) === "string"` (default mode is string) | `"snapshots the archived node-rfc BCD output modes exactly"` | test/classic-bcd.test.ts:10, 13 |
| `projectClassicBcdOutput("123.4500", "string", "AMOUNT") === "123.4500"`; `..."number"... === 123.45` | `"projects exact BCD text as string, number, or one ordinary function call"` | test/classic-bcd.test.ts:26-28 |
| `projectClassicBcdOutput("1E+6144", "number", "AMOUNT") === Number.POSITIVE_INFINITY` | `"projects exact BCD text as string, number, or one ordinary function call"` | test/classic-bcd.test.ts:29-32 |
| converter is called with `receiver: undefined` and the exact value string | `"projects exact BCD text as string, number, or one ordinary function call"` | test/classic-bcd.test.ts:39-43 |
| thrown error message must not contain the decimal value | `"wraps converter exceptions without copying the decimal value into diagnostics"` | test/classic-bcd.test.ts:46, 65 |

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `if (mode === "number") return Number(value);` | src/values/classic-bcd.ts:45 | `Number(decimalString)` is IEEE-754 binary64 parsing with silent overflow to `±Inf` and silent precision loss. `strconv.ParseFloat(s, 64)` returns `ErrRange` **and** `±Inf` on overflow — the test at test/classic-bcd.test.ts:29-32 requires `+Inf`, not an error, so the range error must be swallowed. Precision loss must likewise not be an error. |
| `return Reflect.apply(mode, undefined, [value]);` | src/values/classic-bcd.ts:47 | No Go equivalent for a user-supplied JS callable. Model as `func(string) (any, error)`. |

### Go mapping notes

- `ClassicBcdMode` is a three-way union of two string literals and a callable. In Go use a struct with a `Kind` enum plus an optional `Converter func(string) (any, error)`; do not use `any`.
- `ClassicBcdConversionError` carries `path` and `cause` and is explicitly a *reusable-connection* marker (src/values/classic-bcd.ts:7-11). Make it a distinct exported error type with `Unwrap()`, checkable via `errors.As`.
- The TypeError from `snapshotClassicBcdMode` is a programming-input error, not a wire error — a sentinel `ErrInvalidBcdMode` is appropriate.

---

## classic-int8.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `ClassicInt8Mode` | type | `export type ClassicInt8Mode = "number" \| "bigint" \| "string";` | src/values/classic-int8.ts:2 |
| `ClassicInt8Value` | type | `export type ClassicInt8Value = number \| bigint \| string;` | src/values/classic-int8.ts:4 |
| `snapshotClassicInt8Mode` | function | `export function snapshotClassicInt8Mode(\n  value: unknown,\n  label = "int8Mode",\n): ClassicInt8Mode` | src/values/classic-int8.ts:11-14 |
| `encodeClassicInt8` | function | `export function encodeClassicInt8(\n  value: unknown,\n  mode: ClassicInt8Mode,\n  path = "INT8",\n): bigint` | src/values/classic-int8.ts:30-34 |
| `decodeClassicInt8` | function | `export function decodeClassicInt8(\n  value: bigint,\n  mode: ClassicInt8Mode,\n  path = "INT8",\n): ClassicInt8Value` | src/values/classic-int8.ts:65-69 |
| `classicInt8InitialValue` | function | `export function classicInt8InitialValue(mode: ClassicInt8Mode): ClassicInt8Value` | src/values/classic-int8.ts:90 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `INT8_MIN` | `-(1n << 63n)` | signed 64-bit lower bound | src/values/classic-int8.ts:6 |
| `INT8_MAX` | `(1n << 63n) - 1n` | signed 64-bit upper bound | src/values/classic-int8.ts:7 |
| `CANONICAL_SIGNED_DECIMAL` | `/^(?:0\|-[1-9][0-9]*\|[1-9][0-9]*)$/u` | accepted string-mode spelling: no `+`, no leading zeros, no `-0` | src/values/classic-int8.ts:8 |
| (inline) string length cap | `value.length > 20` | rejects string-mode inputs longer than 20 characters before `BigInt()` | src/values/classic-int8.ts:55 |
| default mode | `if (value === undefined) return "bigint";` | "the core defaults to exact bigint values" | src/values/classic-int8.ts:10, 15 |

### Errors

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `` `${label} must be number, bigint, or string` `` (TypeError) | mode option is not one of the three literals | src/values/classic-int8.ts:19 |
| `` `${path} expects a signed 64-bit integer` `` (RangeError) | value outside `INT8_MIN..INT8_MAX`; also bigint-mode type rejection | src/values/classic-int8.ts:24, 48 |
| `` `${path} expects a safe integer number in number mode` `` (RangeError) | number mode and value is not a number or not `Number.isSafeInteger` | src/values/classic-int8.ts:38 |
| `` `${path} expects a canonical signed decimal string in string mode` `` (TypeError) | string mode and non-string, length > 20, or non-canonical spelling | src/values/classic-int8.ts:58 |
| `` `${path} INT8 result exceeds JavaScript's safe integer range; ` + "use bigint or string mode" `` (RangeError) | decoding to number mode a value outside ±2^53−1 | src/values/classic-int8.ts:79-82 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Exact JavaScript representation selected for an ABAP signed INT8 value." | src/values/classic-int8.ts:1 |
| "Capture an untrusted option once; the core defaults to exact bigint values." | src/values/classic-int8.ts:10 |
| "Normalize one mode-specific caller value to the exact wire integer." | src/values/classic-int8.ts:29 |
| "Project one exact wire integer without allowing silent precision loss." | src/values/classic-int8.ts:64 |
| "Mode-correct ABAP initial value." | src/values/classic-int8.ts:89 |

### Wire facts asserted by tests

No dedicated `classic-int8.test.ts` exists in scope. Behaviour is asserted indirectly:

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| `INT8: "-9007199254740993"` survives encode and decode in `int8Mode: "string"`, and appears on the wire as `<INT8>-9007199254740993</INT8>` | `"round-trips the extended flat xRFC scalar set with compatibility modes"` | test/classic-xrfc.test.ts:244, 256, 265, 282 |
| `INT8: -9_007_199_254_740_993n` round-trips through recursive xRFC | `"round-trips every recursive scalar wire form with xRFC DATE/TIME lexical values"` | test/recursive-xrfc.test.ts:1074, 1117, 1137-1140 |

Note: `-9007199254740993` is `−(2^53+1)`, deliberately **outside** `Number.isSafeInteger`.

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `const INT8_MIN = -(1n << 63n);` / `const INT8_MAX = (1n << 63n) - 1n;` | src/values/classic-int8.ts:6-7 | These are exactly `math.MinInt64` / `math.MaxInt64`. Use `int64` — no `math/big` needed. |
| `if (typeof value !== "number" \|\| !Number.isSafeInteger(value)) { throw ... }` then `assertSignedInt8(BigInt(value), path)` | src/values/classic-int8.ts:37-40 | "number mode" input is a float64 restricted to \|v\| ≤ 2^53−1 **and** integral. In Go the analogue is `float64` input: reject `v != math.Trunc(v)` or `math.Abs(v) > 1<<53 - 1` before `int64(v)`. |
| bigint-mode also accepts a `number` when `Number.isSafeInteger(value)` | src/values/classic-int8.ts:42-46 | bigint mode is not strict: a safe-integer float64 is silently widened. Preserve this acceptance if the Go API keeps mode parity. |
| `value.length > 20 \|\| !CANONICAL_SIGNED_DECIMAL.test(value)` then `BigInt(value)` | src/values/classic-int8.ts:53-60 | `strconv.ParseInt(s, 10, 64)` accepts `+1` and `-0` and leading zeros — all of which this code rejects. Apply the regex first, then `ParseInt`. The `> 20` length guard is a pre-parse DoS bound, not a range check. |
| `const projected = Number(normalized); if (!Number.isSafeInteger(projected)) throw` | src/values/classic-int8.ts:77-83 | This is the *only* place the code refuses to lose precision. In Go the equivalent is: converting `int64` → `float64` for a "number mode" projection must error when `\|v\| > 2^53−1`. Do not skip this check just because Go's `int64` is lossless. |
| `case "bigint": return 0n;` vs `case "number": return 0;` vs `case "string": return "0";` | src/values/classic-int8.ts:90-99 | Initial value is mode-dependent and must stay so. |

### Go mapping notes

- The whole file becomes `int64` arithmetic. `bigint` has no Go analogue; the three modes collapse to a tagged result type (`Int64Value{Mode, I int64, S string}`) or three accessor methods.
- `snapshotClassicInt8Mode` returning the *default* `"bigint"` for `undefined` matters: an unset Go option field must map to the exact-integer mode, not to float.
- Sentinel errors: `ErrInt8OutOfRange` (RangeError paths at :24, :38, :48, :79) and `ErrInt8NonCanonical` (TypeError at :58) — the distinction between RangeError and TypeError is used by callers' `assert.throws` regexes only, so a single family with distinct sentinels is sufficient.
- `INT8_MIN`/`INT8_MAX` should be unexported package constants.

---

## packed-decimal.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `PackedDecimalInput` | type | `export type PackedDecimalInput = string \| number \| bigint \| { toString(): string };` | src/values/packed-decimal.ts:6 |
| `encodePackedDecimal` | function | `export function encodePackedDecimal(\n  value: PackedDecimalInput,\n  byteLength: number,\n  decimals: number,\n  path = "packed decimal",\n): Buffer` | src/values/packed-decimal.ts:108-113 |
| `decodePackedDecimal` | function | `export function decodePackedDecimal(\n  value: Uint8Array,\n  decimals: number,\n  path = "packed decimal",\n): string` | src/values/packed-decimal.ts:133-137 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `MAX_PACKED_DECIMAL_TEXT_LENGTH` | `4_096` | maximum characters of decimal text accepted before parsing | src/values/packed-decimal.ts:8 |
| digit capacity | `const digits = byteLength * 2 - 1;` | digit capacity of a TYPE P field: two nibbles per byte minus the sign nibble | src/values/packed-decimal.ts:97 |
| byte-length domain | `byteLength < 1 \|\| byteLength > 16` | packed length must be an integer in 1..16 | src/values/packed-decimal.ts:94 |
| decimals domain | `const maximumDecimals = Math.min(14, digits);` | decimals must be an integer in `0..maximumDecimals` | src/values/packed-decimal.ts:98-103 |
| sign nibbles written | `` const nibbles = `${digits}${negative ? "D" : "C"}`; `` | encoder emits `C` for non-negative, `D` for negative | src/values/packed-decimal.ts:124 |
| sign nibbles accepted (positive) | `const positive = sign === "A" \|\| sign === "C" \|\| sign === "E" \|\| sign === "F";` | decoder accepts four positive sign nibbles | src/values/packed-decimal.ts:148 |
| sign nibbles accepted (negative) | `const negative = sign === "B" \|\| sign === "D";` | decoder accepts two negative sign nibbles | src/values/packed-decimal.ts:149 |
| decimal lexical grammar | `/^([+-]?)(?:(\d+)(?:\.(\d*))?\|\.(\d+))(?:[eE]([+-]?\d+))?$/u` | accepted input spelling | src/values/packed-decimal.ts:43 |

### Errors

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `` `${path} expects a finite decimal` `` (TypeError) | number input that is `NaN` or `±Infinity` | src/values/packed-decimal.ts:13 |
| `` `${path} expects a decimal string, number, bigint, or decimal object` `` (TypeError) | non-object, non-scalar input | src/values/packed-decimal.ts:24 |
| `` `${path} decimal object must provide toString()` `` (TypeError) | object input whose `toString` is not a function | src/values/packed-decimal.ts:28 |
| `` `${path} decimal object's toString() must return a string` `` (TypeError) | `toString()` returned a non-string | src/values/packed-decimal.ts:32 |
| `` `${path} is not a decimal value` `` (TypeError) | text does not match the decimal grammar | src/values/packed-decimal.ts:44 |
| `` `${path} exceeds its ${capacity}-digit packed capacity` `` (RangeError) | significant digits + left shift exceed capacity | src/values/packed-decimal.ts:53, 62, 71, 85 |
| `` `${path} has more than ${decimals} fractional digits` `` (RangeError) | a non-zero digit would be discarded by rescaling | src/values/packed-decimal.ts:55, 64, 78 |
| `` `${path} packed length must be an integer in 1..16` `` (RangeError) | byteLength out of domain | src/values/packed-decimal.ts:95 |
| `` `${path} decimals must be an integer in 0..${maximumDecimals}` `` (RangeError) | decimals out of domain | src/values/packed-decimal.ts:100-102 |
| `` `${path} decimal text exceeds ${MAX_PACKED_DECIMAL_TEXT_LENGTH} characters` `` (RangeError) | input text longer than 4096 characters | src/values/packed-decimal.ts:117-119 |
| `` `${path} expects Uint8Array bytes` `` (TypeError) | decode input is not a `Uint8Array` | src/values/packed-decimal.ts:139 |
| `` `${path} contains a non-decimal digit nibble` `` (Error) | any of the `capacity` digit nibbles is A–F | src/values/packed-decimal.ts:146 |
| `` `${path} contains invalid sign nibble ${sign}` `` (Error) | trailing nibble is `0`–`9` | src/values/packed-decimal.ts:150 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Encode ABAP TYPE P packed BCD with a trailing C/D sign nibble." | src/values/packed-decimal.ts:107 |
| "Decode ABAP TYPE P to node-rfc's precision-preserving default string." | src/values/packed-decimal.ts:132 |
| "The encoder owns the exact 1..16 byte and decimal-scale rules." (at the `P` validation call site) | src/values/classic-structure.ts:271 |

### Wire facts asserted by tests

No `packed-decimal.test.ts` exists in scope. The only in-scope byte-level assertions are indirect:

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| a `P` field with `internalLength: 4, decimals: 2` carrying `"12.34"` renders as `<BCD>12.34</BCD>` and decodes to `12.34` under `bcd: "number"` | `"round-trips the extended flat xRFC scalar set with compatibility modes"` | test/classic-xrfc.test.ts:90-92, 244, 264, 280 |
| `P` with `ucLength: 3, decimals: 2` carries `"12.34"`/`"56.78"` through recursive xRFC | `"projects BCD values through recursive xRFC structures and tables"` | test/recursive-xrfc.test.ts:867-902 |
| `P` with `ucLength: 3, nucLength: 3, decimals: 2` round-trips `"12.34"` | `"round-trips every recursive scalar wire form with xRFC DATE/TIME lexical values"` | test/recursive-xrfc.test.ts:1081, 1112, 1137-1140 |

The exact packed byte layout is **not asserted by any in-scope test**; it is only stated by the encoder body (src/values/packed-decimal.ts:122-129).

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `if (typeof value === "number") { if (!Number.isFinite(value)) throw ...; return value.toString(); }` | src/values/packed-decimal.ts:12-15 | **Highest-risk line in the file.** `Number#toString()` is JS's shortest-round-trip double formatting, which switches to exponential notation only for \|v\| ≥ 1e21 or < 1e-6, and never emits a `+` on the mantissa. `strconv.FormatFloat(v, 'g', -1, 64)` uses different exponent thresholds and prints `1e+21`. To match, format with `'f'`/`'e'` chosen by the JS thresholds, or refuse float64 input in the Go API and require a decimal string. |
| `const exponent = match[5] === undefined ? 0 : Number(match[5]);` then `if (!Number.isSafeInteger(exponent))` | src/values/packed-decimal.ts:49-56 | The exponent is parsed as a float64; a >2^53 exponent becomes non-safe and is routed to a *specific* error depending on sign and whether the coefficient is non-zero. Use `strconv.ParseInt(s, 10, 64)` and treat `ErrRange` as the "not safe integer" branch, preserving the zero-coefficient short-circuit `if (!nonzero) return { digits: "0", negative: false };`. |
| `const shift = exponent + decimals - fraction.length; if (!Number.isSafeInteger(shift))` | src/values/packed-decimal.ts:58-65 | Same: the sum is float64. In Go use `int64` with explicit overflow checks; do not rely on wraparound. |
| `scaled = ${coefficient}${"0".repeat(shift)};` | src/values/packed-decimal.ts:73 | `shift` is bounded above only by the preceding capacity check at :70. In Go, bound the allocation before `strings.Repeat`. |
| `Number.parseInt(nibbles.slice(index * 2, index * 2 + 2), 16)` | src/values/packed-decimal.ts:127 | Nibble pairs are assembled through hex text. In Go, build bytes directly: `(d[2i]-'0')<<4 \| (d[2i+1]-'0')`, with the last low nibble being `0xC`/`0xD`. |
| `const hex = bytes.toString("hex").toUpperCase();` then `hex.slice(0, capacity)`, `hex.at(-1)` | src/values/packed-decimal.ts:144-147 | Decoding goes through an uppercase hex string. In Go, read nibbles from `[]byte` directly; `hex.at(-1)` is the **low nibble of the last byte**, and `capacity == byteLength*2-1` so the digit run is exactly all nibbles except that one. |
| `const nonzero = /[1-9]/u.test(digits); return \`${negative && nonzero ? "-" : ""}${integer}\` ...` | src/values/packed-decimal.ts:155-157 | A negative sign nibble over an all-zero digit run produces **no** minus sign. There is no "-0" in packed-decimal output. |

### Go mapping notes

- All arithmetic here is decimal-string manipulation, not numeric. Port it as `string`/`[]byte` work; do **not** introduce `math/big.Float` or a decimal library — that would change rounding behaviour, and this code never rounds (it errors instead: src/values/packed-decimal.ts:78).
- `PackedDecimalInput`'s `{ toString(): string }` arm has no Go equivalent. Replace with an interface `interface{ String() string }` (note `fmt.Stringer`) or drop it.
- `reflectApply = Reflect.apply` (src/values/packed-decimal.ts:9) is a JS hardening idiom (avoid a poisoned `Function.prototype.call`); no Go analogue, delete it.
- Sentinels: `ErrPackedCapacity` (:53/:62/:71/:85), `ErrPackedFractionLoss` (:55/:64/:78), `ErrPackedGeometry` (:95/:100), `ErrPackedNibble` (:146/:150).
- `geometry()` and `scaledDigits()` should be unexported.

---

## decimal-float.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `DecimalFloatInput` | type | `export type DecimalFloatInput =\n  \| string\n  \| number\n  \| bigint\n  \| { toString(): string };` | src/values/decimal-float.ts:6-10 |
| `encodeDecimalFloat16` | function | `export function encodeDecimalFloat16(\n  value: DecimalFloatInput,\n  path = "DECF16",\n): Buffer` | src/values/decimal-float.ts:446-449 |
| `decodeDecimalFloat16` | function | `export function decodeDecimalFloat16(\n  value: Uint8Array,\n  path = "DECF16",\n): string` | src/values/decimal-float.ts:454-457 |
| `encodeDecimalFloat34` | function | `export function encodeDecimalFloat34(\n  value: DecimalFloatInput,\n  path = "DECF34",\n): Buffer` | src/values/decimal-float.ts:462-465 |
| `decodeDecimalFloat34` | function | `export function decodeDecimalFloat34(\n  value: Uint8Array,\n  path = "DECF34",\n): string` | src/values/decimal-float.ts:470-473 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `DECF16.label` | `"DECF16"` | format label used in messages | src/values/decimal-float.ts:26 |
| `DECF16.byteLength` | `8` | wire width | src/values/decimal-float.ts:27 |
| `DECF16.precision` | `16` | significant decimal digits | src/values/decimal-float.ts:28 |
| `DECF16.exponentContinuationBits` | `8` | exponent continuation field width | src/values/decimal-float.ts:29 |
| `DECF16.coefficientContinuationBits` | `50` | coefficient continuation field width | src/values/decimal-float.ts:30 |
| `DECF16.exponentBias` | `398` | exponent bias | src/values/decimal-float.ts:31 |
| `DECF34.label` | `"DECF34"` | format label | src/values/decimal-float.ts:35 |
| `DECF34.byteLength` | `16` | wire width | src/values/decimal-float.ts:36 |
| `DECF34.precision` | `34` | significant decimal digits | src/values/decimal-float.ts:37 |
| `DECF34.exponentContinuationBits` | `12` | exponent continuation field width | src/values/decimal-float.ts:38 |
| `DECF34.coefficientContinuationBits` | `110` | coefficient continuation field width | src/values/decimal-float.ts:39 |
| `DECF34.exponentBias` | `6176` | exponent bias | src/values/decimal-float.ts:40 |
| `MAX_DECIMAL_FLOAT_TEXT_LENGTH` | `4_096` | pre-parse text bound | src/values/decimal-float.ts:55 |
| infinity combination field | `0b11110n` (encode) / `combination === 0b11110` (decode) | 5-bit combination value denoting Infinity | src/values/decimal-float.ts:317, 408 |
| NaN combination field | `0b11111n` (encode) / `combination === 0b11111` (decode) | 5-bit combination value denoting NaN/sNaN | src/values/decimal-float.ts:328, 411 |
| signaling-NaN bit position | `format.coefficientContinuationBits + format.exponentContinuationBits - 1` | bit index of the sNaN flag | src/values/decimal-float.ts:323-325, 412-414 |
| combination encoding (MSD ≤ 7) | `(exponentMostSignificant << 3) \| mostSignificantDigit` | | src/values/decimal-float.ts:355 |
| combination encoding (MSD 8/9) | `0b11000 \| (exponentMostSignificant << 1) \| (mostSignificantDigit - 8)` | | src/values/decimal-float.ts:356 |
| maximum encoded exponent | `3n * (1n << BigInt(format.exponentContinuationBits)) - 1n` | | src/values/decimal-float.ts:263 |
| minimum exponent | `BigInt(-format.exponentBias)` | | src/values/decimal-float.ts:262 |
| special lexical: infinity | `/^([+-]?)(?:inf\|infinity)$/iu` | case-insensitive | src/values/decimal-float.ts:206 |
| special lexical: NaN | `/^([+-]?)(s?nan)(\d*)$/iu` | case-insensitive, optional diagnostic payload | src/values/decimal-float.ts:215 |
| NaN payload capacity | `const capacity = format.precision - 1;` | 15 digits for DECF16, 33 for DECF34 | src/values/decimal-float.ts:218 |
| finite lexical | `/^([+-]?)(?:(\d+)(?:\.(\d*))?\|\.(\d+))(?:[eE]([+-]?\d+))?$/u` | | src/values/decimal-float.ts:234-235 |
| plain-vs-scientific threshold | `if (exponent <= 0 && adjustedExponent >= -6)` | selects plain notation | src/values/decimal-float.ts:370 |

### Errors

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `"unreachable DPD digit classification"` (Error) | `largePattern` outside 0..7 | src/values/decimal-float.ts:91 |
| `` `${path} expects a string, number, bigint, or decimal object` `` (TypeError) | non-object input, or object without a callable `toString` | src/values/decimal-float.ts:178-180, 184-186 |
| `` `${path} decimal object's toString() must return a string` `` (TypeError) | `toString()` returned non-string | src/values/decimal-float.ts:190 |
| `` `${path} exceeds its ${capacity}-digit NaN payload` `` (RangeError) | NaN diagnostic longer than `precision - 1` digits | src/values/decimal-float.ts:220 |
| `` `${path} expects a valid decimal` `` (TypeError) | text matches neither special nor finite grammar | src/values/decimal-float.ts:236 |
| `` `${path} has an exponent too large to represent` `` (RangeError) | `BigInt(match[5])` threw | src/values/decimal-float.ts:247 |
| `` `${path} exceeds ${format.precision} significant digits without rounding` `` (RangeError) | coefficient too long and not removable via trailing zeros | src/values/decimal-float.ts:254-256 |
| `` `${path} is outside ${format.label} range without rounding` `` (RangeError) | exponent above max / below min and not fixable by shifting zeros | src/values/decimal-float.ts:273, 281 |
| `` `${path} decimal text exceeds ${MAX_DECIMAL_FLOAT_TEXT_LENGTH} characters` `` (RangeError) | text longer than 4096 characters | src/values/decimal-float.ts:340-342 |
| `` `${path} expects exactly ${format.byteLength} bytes` `` (RangeError) | decode input not a `Uint8Array`, or wrong intrinsic byte length | src/values/decimal-float.ts:394, 398 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "IEEE 754 decimal interchange using Cowlishaw's Densely Packed Decimal mapping (https://speleotrove.com/decimal/dbspec.html). SAP stores these fields in host-little-endian byte order on supported RFC platforms." | src/values/decimal-float.ts:12-14 |
| "The pattern is the a/e/i high bit from each input BCD digit." | src/values/decimal-float.ts:66 |
| "The published expansion table classifies operands by v/w/x then s/t. Its final branch deliberately accepts the 24 redundant DPD encodings." | src/values/decimal-float.ts:106-107 |
| "Encode an exact IEEE 754 decimal64 DPD value in SAP's little-endian DECF16 form." | src/values/decimal-float.ts:445 |
| "Decode SAP DECF16 to node-rfc's precision-preserving string representation." | src/values/decimal-float.ts:453 |
| "Encode an exact IEEE 754 decimal128 DPD value in SAP's little-endian DECF34 form." | src/values/decimal-float.ts:461 |
| "Decode SAP DECF34 to node-rfc's precision-preserving string representation." | src/values/decimal-float.ts:469 |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| the eleven DECF16 `[input, hex, output]` triples encode and decode exactly (little-endian hex) | `"matches IEEE 754-2008 decimal64 DPD vectors"` | test/decimal-float.test.ts:94-113 |
| the eleven DECF34 `[input, hex, output]` triples encode and decode exactly | `"matches IEEE 754-2008 decimal128 DPD vectors"` | test/decimal-float.test.ts:115-138 |
| for every value 0..999 the low 10 bits of the encoding equal Cowlishaw's Boolean-equation oracle | `"exhaustively emits every canonical DPD declet"` | test/decimal-float.test.ts:140-150 |
| all 1024 declet codes decode to the oracle value, and exactly `24` of them are redundant encodings | `"decodes all 1,024 DPD declets, including 24 redundant encodings"` | test/decimal-float.test.ts:152-162 |
| `"1.2300"` round-trips as `"1.2300"` (cohort preserved) | `"preserves exact cohorts, signed zero, subnormals, and range-edge values"` | test/decimal-float.test.ts:165 |
| `encodeDecimalFloat16("-0.00") === "00000000000030a2"`, decodes to `"-0.00"` | same | test/decimal-float.test.ts:166-167 |
| `encodeDecimalFloat16("1E-398") === "0100000000000000"`; `"10E-399"` decodes to `"1E-398"` | same | test/decimal-float.test.ts:168-170 |
| `encodeDecimalFloat16("1E+384") === "000000000000fc47"` decodes to `"1.000000000000000E+384"` | same | test/decimal-float.test.ts:171-175 |
| `"0E-999"` clamps to `"0E-398"`; `"-0E+999"` clamps to `"-0E+369"` | same | test/decimal-float.test.ts:176-177 |
| `encodeDecimalFloat34("1E-6176") === "01000000000000000000000000000000"`; `encodeDecimalFloat34("1E+6144") === "00000000000000000000000000c0ff47"` | same | test/decimal-float.test.ts:179-191 |
| `"0E-99999"` clamps to `"0E-6176"` | same | test/decimal-float.test.ts:192 |
| `encodeDecimalFloat16("12345678901234560") === "568ee2c1b9343d26"` → `"1.234567890123456E+16"` | `"rescales excess trailing zeros exactly before enforcing precision and qmin"` | test/decimal-float.test.ts:196-203 |
| `encodeDecimalFloat34("12345678901234567890123456789012340") === "3435827771123c6fe5281e9c4b530826"` | same | test/decimal-float.test.ts:213-222 |
| `"12345678901234567"` (17 sig digits) throws `/exceeds 16 significant digits/`; DECF34 analogue throws `/exceeds 34 significant digits/` | same | test/decimal-float.test.ts:238-245 |
| `"inf"`, `"+INFINITY"`, `"sNaN"`, `"-NaN8275"`, `"sNaN123456789"` all round-trip | `"accepts General Decimal Arithmetic specials and diagnostic NaNs"` | test/decimal-float.test.ts:248-253 |
| all-ones DECF16 `"ffffffffffffffff"` decodes to `"-sNaN999999999999999"` | same | test/decimal-float.test.ts:257-258 |
| `"fffffffffffffffb"` decodes to `"-Infinity"` — an infinity payload is ignored | same | test/decimal-float.test.ts:259-260 |
| `"1E+385"`, `"1E-399"` throw `/outside DECF16 range/`; `"1E+6145"`, `"1E-6177"` throw `/outside DECF34 range/` | `"rejects rounding, overflow, underflow, malformed syntax, and bad geometry"` | test/decimal-float.test.ts:272-275 |
| `"NaN1234567890123456"` throws `/15-digit NaN payload/` | same | test/decimal-float.test.ts:276 |
| `""`, `" 1"`, `"1,2"`, `"."` all throw `/valid decimal/` | same | test/decimal-float.test.ts:277-280 |
| `Buffer.alloc(7)` throws `/exactly 8 bytes/`; `Buffer.alloc(17)` throws `/exactly 16 bytes/` | same | test/decimal-float.test.ts:282-283 |
| a decimal object's `toString` is read once and called once | `"converts decimal objects exactly once and supports bigint and number inputs"` | test/decimal-float.test.ts:286-303 |
| `decodeDecimalFloat16(encodeDecimalFloat16(123.45)) === "123.45"` (JS double input) | same | test/decimal-float.test.ts:304 |
| `decodeDecimalFloat34(encodeDecimalFloat34(12345678901234567890n)) === "12345678901234567890"` | same | test/decimal-float.test.ts:305 |
| decode consults intrinsic byte geometry, never own `byteLength`/`byteOffset`/`buffer` accessors | `"uses intrinsic byte geometry and snapshots without consulting own accessors"` | test/decimal-float.test.ts:308-333 |
| text longer than 4096 characters is refused before significand/exponent/NaN/object parsing | `"bounds decimal text before significand, exponent, NaN, or object parsing"` | test/decimal-float.test.ts:335-361 |

### JavaScript number-semantics dependencies

**This is the densest BigInt file in the layer.** Every item below is load-bearing for byte-exactness.

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `let encoded = 0n; ... encoded = (encoded << 10n) \| BigInt(encodeDpdDeclet(declet));` | src/values/decimal-float.ts:149-156 | The coefficient continuation is **110 bits** for DECF34 — wider than `uint64`. Use `math/big.Int`, or (preferred, allocation-free) a fixed `[2]uint64`/`[16]byte` shift-accumulate. Do **not** use `uint64`. |
| `groups[index] = String(decodeDpdDeclet(Number(remainder & 0x3ffn))).padStart(3, "0"); remainder >>= 10n;` | src/values/decimal-float.ts:163-164 | Same 110-bit shift register in the decode direction. `Number(x & 0x3ffn)` is safe (≤ 1023). |
| `explicitExponent = BigInt(match[5] ?? "0");` inside `try { } catch { throw new RangeError(...) }` | src/values/decimal-float.ts:243-248 | The literal exponent is parsed as an **arbitrary-precision** integer, so `1E+<4000 digits>` parses successfully and is then range-checked. `strconv.ParseInt` would return `ErrRange` first; either use `big.Int.SetString` or map `ErrRange` onto "outside range without rounding" while preserving the zero-coefficient clamp at :266-268. |
| `let exponent = explicitExponent - BigInt(fraction.length);` and `exponent += BigInt(excessDigits);` | src/values/decimal-float.ts:249, 259 | Exponent arithmetic is bigint throughout `parseFinite`; only the final result is narrowed. |
| `return { ..., exponent: Number(exponent) };` | src/values/decimal-float.ts:287-291 | The clamps at :262-285 guarantee the value fits `−6176..6111`, so `int` is safe — but only *after* the clamps. Port the clamps first. |
| `const maximumEncodedExponent = 3n * (1n << BigInt(format.exponentContinuationBits)) - 1n;` | src/values/decimal-float.ts:263 | DECF16: 3·256−1 = 767; DECF34: 3·4096−1 = 12287. Computable in `int`; kept bigint here only for type uniformity. |
| `coefficient += "0".repeat(Number(requiredZeros));` and `coefficient.slice(0, coefficient.length - Number(requiredZeros))` | src/values/decimal-float.ts:275, 283 | `requiredZeros` is bigint but is proven ≤ `precision` by the preceding comparison. Narrow to `int` only after that check. |
| `writeLittleEndian(value: bigint, byteLength)` / `readLittleEndian(value: Uint8Array): bigint` | src/values/decimal-float.ts:294-310 | The whole 64/128-bit encoded word is a bigint. In Go, assemble `[8]byte`/`[16]byte` directly little-endian; `binary.LittleEndian.PutUint64` covers DECF16, DECF34 needs two words with the sign/combination in the **high** word. |
| `const sign = value.negative ? 1n << BigInt(totalBits - 1) : 0n;` and `(BigInt(combination) << BigInt(totalBits - 6))` | src/values/decimal-float.ts:314-315, 358-360 | Shifts of 122/127 bits — beyond `uint64`. Must be 128-bit-aware for DECF34. |
| `const mostSignificantDigit = Number(digits[0]!);` | src/values/decimal-float.ts:348 | `Number("7")` on a one-character string. In Go: `int(digits[0] - '0')`. |
| `const exponentMostSignificant = encodedExponent >> format.exponentContinuationBits;` and `const exponentContinuationMask = (1 << format.exponentContinuationBits) - 1;` | src/values/decimal-float.ts:351-353 | JS `>>`/`<<` coerce to **int32**. `encodedExponent ≤ 12287` and the shift is ≤ 12, so this is safe — but a Go port must not blindly copy `>>` semantics elsewhere. |
| `if (Object.is(value, -0)) return "-0";` before `value.toString()` | src/values/decimal-float.ts:170-173 | Go's `strconv.FormatFloat(math.Copysign(0,-1), ...)` already yields `"-0"`, but `v == 0` comparisons will not distinguish it. Use `math.Signbit`. |
| `return value.toString();` for a `number` input | src/values/decimal-float.ts:172 | Same JS shortest-round-trip formatting hazard as packed-decimal. `encodeDecimalFloat16(123.45)` must produce the encoding of the string `"123.45"`, not of `123.4500000000000028...` (test/decimal-float.test.ts:304). |
| `Number.parseInt(digits.slice(index, index + 3), 10)` | src/values/decimal-float.ts:152 | Three-digit declet from text. In Go compute from bytes. |
| `const decletCount = digitCount / 3;` | src/values/decimal-float.ts:159 | `precision - 1` is 15 and 33, both divisible by 3. Non-integer division would silently produce a fractional array length in JS; in Go use integer division and assert. |
| decoder returns `` `${negative ? "-" : ""}Infinity` `` for combination `0b11110` **without** reading the payload | src/values/decimal-float.ts:408-410 | Reproduce exactly: test/decimal-float.test.ts:259-260 requires `"fffffffffffffffb"` → `"-Infinity"`. |
| `formatFinite` selects plain notation when `exponent <= 0 && adjustedExponent >= -6`, else `` `${significand}E${adjustedExponent >= 0 ? "+" : ""}${adjustedExponent}` `` | src/values/decimal-float.ts:366-386 | This is General Decimal Arithmetic `to-scientific-string`. Go's `%g`/`FormatFloat` will not match. Implement the branch literally, including the explicit `+` on non-negative exponents. |

### Go mapping notes

- `encodeDpdDeclet` (src/values/decimal-float.ts:58-93) is a case table on the high bits of three BCD digits; `decodeDpdDeclet` (src/values/decimal-float.ts:95-147) is the published expansion table. Both are pure `int → int` and port verbatim to Go. Consider replacing both with 1000-entry and 1024-entry lookup tables generated at init — the tests at test/decimal-float.test.ts:140-162 verify both directions exhaustively, so a table is safe.
- `DecimalFloatFormat` is a value struct — port as an unexported `type decimalFloatFormat struct` with two package-level values.
- `FiniteDecimal`/`SpecialDecimal` (src/values/decimal-float.ts:43-53) are internal; keep unexported.
- Sentinels: `ErrDecimalFloatPrecision` (:254), `ErrDecimalFloatRange` (:273/:281), `ErrDecimalFloatSyntax` (:236), `ErrDecimalFloatNaNPayload` (:220), `ErrDecimalFloatWidth` (:394/:398).
- The `reflectApply` hardening (src/values/decimal-float.ts:56, 188) and the intrinsic-geometry helpers have no Go analogue; the corresponding tests (test/decimal-float.test.ts:308-333, 363-382) have **no Go port** and should be recorded as such in provenance.

---

## classic-temporal.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `assertClassicDate` | function | `export function assertClassicDate(value: string, path: string): void` | src/values/classic-temporal.ts:7 |
| `assertClassicTime` | function | `export function assertClassicTime(value: string, path: string): void` | src/values/classic-temporal.ts:14 |
| `classicDateWireText` | function | `export function classicDateWireText(value: string, path: string): string` | src/values/classic-temporal.ts:21 |
| `classicDatePublicText` | function | `export function classicDatePublicText(value: string, path: string): string` | src/values/classic-temporal.ts:27 |
| `classicTimeWireText` | function | `export function classicTimeWireText(value: string, path: string): string` | src/values/classic-temporal.ts:35 |
| `classicTimePublicText` | function | `export function classicTimePublicText(value: string, path: string): string` | src/values/classic-temporal.ts:41 |
| `ClassicTemporalExid` | type | `export type ClassicTemporalExid =\n  \| "p" // UTCLONG\n  \| "n" // UTCSECOND\n  \| "w" // UTCMINUTE\n  \| "d" // DTDAY\n  \| "7" // DTWEEK\n  \| "x" // DTMONTH\n  \| "t" // TSECOND\n  \| "i" // TMINUTE\n  \| "c"; // CDAY` | src/values/classic-temporal.ts:49-58 |
| `isClassicTemporalExid` | function | `export function isClassicTemporalExid(value: string): value is ClassicTemporalExid` | src/values/classic-temporal.ts:138 |
| `classicTemporalByteLength` | function | `export function classicTemporalByteLength(exid: ClassicTemporalExid): number` | src/values/classic-temporal.ts:163 |
| `classicTemporalInitialValue` | function | `export function classicTemporalInitialValue(exid: ClassicTemporalExid): string` | src/values/classic-temporal.ts:168 |
| `encodeClassicTemporal` | function | `export function encodeClassicTemporal(\n  exid: ClassicTemporalExid,\n  value: string,\n  path = "classic temporal value",\n): Buffer` | src/values/classic-temporal.ts:443-447 |
| `decodeClassicTemporal` | function | `export function decodeClassicTemporal(\n  exid: ClassicTemporalExid,\n  value: Uint8Array,\n  path = "classic temporal value",\n): string` | src/values/classic-temporal.ts:604-608 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| DATS lexical | `/^(?:\d{8}\| {8}\|)$/u` | "expects YYYYMMDD, an empty string, or eight spaces" | src/values/classic-temporal.ts:8-9 |
| TIMS lexical | `/^(?:\d{6}\| {6}\|)$/u` | "expects HHMMSS, an empty string, or six spaces" | src/values/classic-temporal.ts:15-16 |
| DATS wire blank | `return value === "" ? "        " : value;` | empty string maps to eight spaces on the wire | src/values/classic-temporal.ts:23 |
| TIMS wire blank | `return value === "" ? "      " : value;` | empty string maps to six spaces on the wire | src/values/classic-temporal.ts:37 |
| `p` / UTCLONG | `{ name: "UTCLONG", byteLength: 8, maximumRaw: 3_155_380_704_000_000_000n }` | | src/values/classic-temporal.ts:68-72 |
| `n` / UTCSECOND | `{ name: "UTCSECOND", byteLength: 8, maximumRaw: 315_538_070_400n }` | | src/values/classic-temporal.ts:73-77 |
| `w` / UTCMINUTE | `{ name: "UTCMINUTE", byteLength: 8, maximumRaw: 5_258_967_840n }` | | src/values/classic-temporal.ts:78-82 |
| `d` / DTDAY | `{ name: "DTDAY", byteLength: 4, maximumRaw: 3_652_061n }` | | src/values/classic-temporal.ts:83-87 |
| `7` / DTWEEK | `{ name: "DTWEEK", byteLength: 4, maximumRaw: 521_725n }` | | src/values/classic-temporal.ts:88-92 |
| `x` / DTMONTH | `{ name: "DTMONTH", byteLength: 4, maximumRaw: 119_988n }` | | src/values/classic-temporal.ts:93-97 |
| `t` / TSECOND | `{ name: "TSECOND", byteLength: 4, maximumRaw: 86_401n }` | | src/values/classic-temporal.ts:98-102 |
| `i` / TMINUTE | `{ name: "TMINUTE", byteLength: 2, maximumRaw: 1_441n }` | | src/values/classic-temporal.ts:103-107 |
| `c` / CDAY | `{ name: "CDAY", byteLength: 2, maximumRaw: 366n }` | | src/values/classic-temporal.ts:108-112 |
| `DAYS_BY_MONTH` | `[\n  31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31,\n]` | note February is **29** in this table; February length is overridden per-year at :204 and CDAY uses the 366-day table directly at :589-595 | src/values/classic-temporal.ts:115-117 |
| `UTCLONG_INITIAL` | `"0000-00-00T00:00:00.0000000"` | compatibility-facing initial string for `p` | src/values/classic-temporal.ts:119, 170 |
| `SECONDS_PER_DAY` | `86_400` | | src/values/classic-temporal.ts:120 |
| `MINUTES_PER_DAY` | `1_440` | | src/values/classic-temporal.ts:121 |
| `FRACTIONS_PER_SECOND` | `10_000_000n` | 100 ns ticks per second | src/values/classic-temporal.ts:122 |
| `FRACTIONS_PER_DAY` | `864_000_000_000n` | 100 ns ticks per day | src/values/classic-temporal.ts:123 |
| leap rule | `if (year < 1582) return year % 4 === 0;\n  return year % 4 === 0 && (year % 100 !== 0 \|\| year % 400 === 0);` | Julian before 1582, Gregorian from 1582 | src/values/classic-temporal.ts:198-201 |
| October 1582 length | `if (year === 1582 && month === 10) return 21;` | the reform month has 21 days | src/values/classic-temporal.ts:205 |
| calendar gap | `if (year === 1582 && month === 10 && day >= 5 && day <= 14)` | 1582-10-05..1582-10-14 do not exist | src/values/classic-temporal.ts:231-235 |
| year domain | `if (year < 1 \|\| year > 9999)` | "year must be in 0001..9999" | src/values/classic-temporal.ts:217-219 |
| `p` lexical | `/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})\.(\d{7})$/u`, expected `"YYYY-MM-DDTHH:MM:SS.fffffff"` | | src/values/classic-temporal.ts:459-460 |
| `n` lexical | `/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})$/u`, `"YYYY-MM-DDTHH:MM:SS"` | | src/values/classic-temporal.ts:475-476 |
| `w` lexical | `/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/u`, `"YYYY-MM-DDTHH:MM"` | | src/values/classic-temporal.ts:490-491 |
| `d` lexical | `/^(\d{4})-(\d{2})-(\d{2})$/u`, `"YYYY-MM-DD"` | | src/values/classic-temporal.ts:505-506 |
| `7` lexical | `/^(\d{4})-W(\d{2})$/u`, `"YYYY-Www"` | | src/values/classic-temporal.ts:516-517 |
| `x` lexical | `/^(\d{4})-(\d{2})$/u`, `"YYYY-MM"` | | src/values/classic-temporal.ts:532-533 |
| `t` lexical | `/^(\d{2}):(\d{2}):(\d{2})$/u`, `"HH:MM:SS"` | | src/values/classic-temporal.ts:551-552 |
| `i` lexical | `/^(\d{2}):(\d{2})$/u`, `"HH:MM"` | | src/values/classic-temporal.ts:565-566 |
| `c` lexical | `/^(\d{2})-(\d{2})$/u`, `"MM-DD"` | | src/values/classic-temporal.ts:579-580 |

### Raw-value formulas (what the code does, quoted)

| EXID | Formula | Citation |
|---|---|---|
| `p` | `raw = (BigInt(dateOrdinal(date) * SECONDS_PER_DAY + clockSeconds(clock))\n        * FRACTIONS_PER_SECOND) + BigInt(match[7]!) + 1n;` | src/values/classic-temporal.ts:468-469 |
| `n` | `raw = BigInt(dateOrdinal(date) * SECONDS_PER_DAY + clockSeconds(clock)) + 1n;` | src/values/classic-temporal.ts:484 |
| `w` | `raw = BigInt(dateOrdinal(date) * MINUTES_PER_DAY + clock.hour * 60 + clock.minute) + 1n;` | src/values/classic-temporal.ts:499 |
| `d` | `raw = BigInt(dateOrdinal(parseDateParts(match, path, spec.name))) + 1n;` | src/values/classic-temporal.ts:510 |
| `7` | `raw = BigInt(weekOrdinal(year, week, path)) + 1n;` | src/values/classic-temporal.ts:526 |
| `x` | `raw = BigInt((year - 1) * 12 + month);` (**no `+ 1n`**) | src/values/classic-temporal.ts:545 |
| `t` | `raw = BigInt(clockSeconds(clock)) + 1n;` | src/values/classic-temporal.ts:559 |
| `i` | `raw = BigInt(clock.hour * 60 + clock.minute) + 1n;` | src/values/classic-temporal.ts:573 |
| `c` | `raw = BigInt(ordinal);` where `ordinal` starts at `day` and adds `DAYS_BY_MONTH` of prior months (**no `+ 1n`**) | src/values/classic-temporal.ts:592-596 |
| all | raw `0` means the initial value in both directions: `if (value.length === 0 \|\| (exid === "p" && value === UTCLONG_INITIAL)) return encodeRaw(exid, 0n, path);` / `if (raw === 0n) return classicTemporalInitialValue(exid);` | src/values/classic-temporal.ts:450-452, 610 |
| all | decode subtracts one: `const ordinal = raw - 1n;` | src/values/classic-temporal.ts:611 |
| `7` | week ordinal: `return priorLongYears * 53 + (year - 1 - priorLongYears) * 52 + week;`, with `year === 0` permitting only week 53 → ordinal 0 | src/values/classic-temporal.ts:372-382 |
| `p` | fraction is rendered `fraction.toString().padStart(7, "0")` | src/values/classic-temporal.ts:620 |

### Errors

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `` `${path} expects YYYYMMDD, an empty string, or eight spaces` `` (TypeError) | DATS input fails the lexical test | src/values/classic-temporal.ts:9 |
| `` `${path} expects HHMMSS, an empty string, or six spaces` `` (TypeError) | TIMS input fails the lexical test | src/values/classic-temporal.ts:16 |
| `` `${path} expects YYYYMMDD or eight spaces from the wire` `` (TypeError) | wire DATS is neither 8 digits nor 8 spaces | src/values/classic-temporal.ts:29 |
| `` `${path} expects HHMMSS or six spaces from the wire` `` (TypeError) | wire TIMS is neither 6 digits nor 6 spaces | src/values/classic-temporal.ts:43 |
| `"unsupported classic temporal EXID"` (TypeError) | EXID is not one of the nine | src/values/classic-temporal.ts:157 |
| `` `${path} ${name} expects a string in ${expected} form` `` (TypeError) | value is not a string | src/values/classic-temporal.ts:180 |
| `` `${path} ${name} expects ${expected}` `` (TypeError) | value fails the per-EXID lexical form | src/values/classic-temporal.ts:193 |
| `` `${path} ${name} year must be in 0001..9999` `` (RangeError) | | src/values/classic-temporal.ts:218, 540 |
| `` `${path} ${name} month must be in 01..12` `` (RangeError) | | src/values/classic-temporal.ts:221, 543, 587 |
| `` `${path} ${name} has invalid day ${String(day).padStart(2, "0")}` `` (RangeError) | day out of the conventional month length | src/values/classic-temporal.ts:227-229, 590 |
| `` `${path} ${name} is in the Gregorian calendar gap 1582-10-05..1582-10-14` `` (RangeError) | | src/values/classic-temporal.ts:232-234 |
| `` `${path} ${name} hours must be in 00..${String(maximumHour)}` `` (RangeError) | hour > 23 (or > 24 when end-of-day allowed) | src/values/classic-temporal.ts:319-321 |
| `` `${path} ${name} minutes must be in 00..59` `` (RangeError) | | src/values/classic-temporal.ts:324 |
| `` `${path} ${name} seconds must be in 00..59` `` (RangeError) | | src/values/classic-temporal.ts:327 |
| `` `${path} ${name} must not exceed ${maximum}` `` (RangeError), `maximum` is `"24:00"` for TMINUTE else `"24:00:00"` | hour 24 with non-zero minute/second | src/values/classic-temporal.ts:329-331 |
| `` `${path} DTWEEK year zero permits only 0000-W53` `` (RangeError) | | src/values/classic-temporal.ts:375 |
| `` `${path} DTWEEK year ${fourDigits(year)} does not have week 53` `` (RangeError) | | src/values/classic-temporal.ts:379 |
| `` `${path} ${spec.name} week must be in 01..53` `` (RangeError) | | src/values/classic-temporal.ts:524 |
| `` `${path} ${spec.name} is outside its valid raw range` `` (RangeError) | raw < 0 or raw > `maximumRaw`, in both encode and decode | src/values/classic-temporal.ts:406, 437 |
| `` `${path} ${spec.name} expects Uint8Array raw bytes` `` (TypeError) | | src/values/classic-temporal.ts:422 |
| `` `${path} ${spec.name} expects ${spec.byteLength} raw bytes; received ${byteLength}` `` (RangeError) | | src/values/classic-temporal.ts:426-428 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Validate a node-rfc-compatible DATS input without inventing a calendar value." | src/values/classic-temporal.ts:6 |
| "Validate a node-rfc-compatible TIMS input without inventing a clock value." | src/values/classic-temporal.ts:13 |
| "Convert a public DATE value into the exact eight-character wire form." | src/values/classic-temporal.ts:20 |
| "Convert an exact DATE wire value to node-rfc's trailing-space-trimmed form." | src/values/classic-temporal.ts:26 |
| "Classic RFC EXIDs backed by SAP's compact integer temporal values." | src/values/classic-temporal.ts:48 |
| "Return the fixed SAP raw width for a compact temporal EXID." | src/values/classic-temporal.ts:162 |
| "Return the compatibility-facing initial string for a compact temporal EXID." | src/values/classic-temporal.ts:167 |
| "Encode a compact SAP temporal string to its signed little-endian RFC value." | src/values/classic-temporal.ts:442 |
| "Decode one fixed-width compact SAP temporal value to its compatibility string." | src/values/classic-temporal.ts:603 |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| `""`, `"00000000"`, `"19000229"`, `"20260229"`, `"99991231"`, `"        "` are all accepted DATS inputs — **no calendar validation** | `"keeps classic DATS and TIMS as fixed raw character forms"` | test/classic-temporal.test.ts:18-21 |
| `""`, `"000000"`, `"235959"`, `"240000"`, `"999999"`, `"      "` are all accepted TIMS inputs | same | test/classic-temporal.test.ts:22-24 |
| full-width digits `"１２３４５６７８"` are rejected | same | test/classic-temporal.test.ts:27 |
| the nine little-endian reference vectors encode and decode exactly (see conformance section) | `"matches the compact-temporal little-endian reference vectors"` | test/classic-temporal.test.ts:42-63 |
| `hex("p", "0000-00-00T00:00:00.0000000") === "0000000000000000"` and `hex("p", "") === "0000000000000000"` | `"uses raw zero only for initial values and preserves node-rfc UTCLONG initial"` | test/classic-temporal.test.ts:65-72 |
| for the other eight EXIDs `hex(exid, "")` is all-zero of the fixed width and decodes to `""` | same | test/classic-temporal.test.ts:74-78 |
| every minimum/maximum pair encodes and decodes exactly (see conformance section) | `"covers every compact temporal minimum and maximum"` | test/classic-temporal.test.ts:81-120 |
| `hex("d", "1582-10-04") === "c9d00800"` and `hex("d", "1582-10-15") === "cad00800"` — consecutive ordinals across the gap | `"uses consecutive ordinals across the historical Julian-to-Gregorian gap"` | test/classic-temporal.test.ts:122-132 |
| `"1500-02-29"`, `"1600-02-29"`, `"2000-02-29"` accepted; `"1700-02-29"`, `"1900-02-29"` rejected | same | test/classic-temporal.test.ts:134-140 |
| `hex("7", "0000-W53") === "01000000"`, `hex("7", "0001-W01") === "02000000"`, `hex("7", "0005-W53") === "06010000"` | `"validates hybrid-calendar week 53 and its reserved year-zero value"` | test/classic-temporal.test.ts:161-165 |
| `"0004-W53"` and `"2021-W53"` rejected; `"0000-W52"` rejected | same | test/classic-temporal.test.ts:166-168 |
| `decodeClassicTemporal("d", Buffer.from("deb93700","hex"))` throws `/outside its valid raw range/` (one past DTDAY max) | `"rejects malformed compact temporal forms and invalid raw values"` | test/classic-temporal.test.ts:198-201 |
| encoding does not coerce a `toString`-bearing object; decoding does not consult a caller `byteLength` getter | `"does not coerce temporal inputs or consult caller-defined byte geometry"` | test/classic-temporal.test.ts:212-237 |
| exactly nine EXIDs with widths `p:8 n:8 w:8 d:4 7:4 x:4 t:4 i:2 c:2` | `"exposes only the nine compact temporal EXIDs and their fixed widths"` | test/classic-temporal.test.ts:239-258 |

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `maximumRaw: 3_155_380_704_000_000_000n` and the other eight bigint bounds | src/values/classic-temporal.ts:66-113 | ~3.16e18 exceeds `Number.MAX_SAFE_INTEGER` (9.007e15) so bigint is mandatory in JS. It fits `int64` (max 9.22e18) — use `int64`, not `math/big`. |
| `raw = (BigInt(dateOrdinal(date) * SECONDS_PER_DAY + clockSeconds(clock)) * FRACTIONS_PER_SECOND) + BigInt(match[7]!) + 1n;` | src/values/classic-temporal.ts:468-469 | The inner product `dateOrdinal * 86400 + clockSeconds` is computed in **float64** and only then widened. Max is 3_652_060·86400 + 86399 = 315_538_070_399 < 2^53, so it is exact — but in Go compute the whole expression in `int64` and add an overflow assertion; do not mirror the float64 intermediate. |
| `BigInt(match[7]!)` on a 7-digit fraction string | src/values/classic-temporal.ts:469 | `strconv.ParseInt(match[7], 10, 64)`; the regex guarantees exactly 7 digits so leading zeros are fine. |
| `if (spec.byteLength === 8) result.writeBigInt64LE(raw); else if (spec.byteLength === 4) result.writeInt32LE(Number(raw)); else result.writeInt16LE(Number(raw));` | src/values/classic-temporal.ts:408-412 | **Signed** little-endian writes at all three widths. `binary.LittleEndian.PutUint64/32/16` with `uint64(int64(v))` etc. The bounds check at :405 runs first, so no width can overflow. |
| `bytes.readBigInt64LE(0)` / `BigInt(bytes.readInt32LE(0))` / `BigInt(bytes.readInt16LE(0))` | src/values/classic-temporal.ts:431-435 | Reads are **signed**; a raw value with the high bit set decodes negative and then fails the `raw < 0n` check at :436. In Go, sign-extend explicitly (`int32(binary.LittleEndian.Uint32(b))`), do not read as unsigned. |
| `const dayOrdinal = Number(ordinal / FRACTIONS_PER_DAY); const withinDay = ordinal % FRACTIONS_PER_DAY;` | src/values/classic-temporal.ts:615-618 | BigInt division truncates toward zero; `ordinal ≥ 0` here so it matches Go's `/` and `%` on non-negative `int64`. |
| `fraction.toString().padStart(7, "0")` where `fraction` is a `bigint` | src/values/classic-temporal.ts:620 | `fmt.Sprintf("%07d", fraction)`. |
| `let days = Math.floor(through1600 / 100) * 36_525 + Math.floor(withinCentury / 4) * 1_461 + (withinCentury % 4) * 365;` | src/values/classic-temporal.ts:243-245 | All-integer arithmetic in float64. Values are small (< 4e6); port to `int`. `Math.floor` on non-negative operands equals Go integer division. |
| `const candidate = Math.floor((lower + upper) / 2);` (binary search over years 1..9999) | src/values/classic-temporal.ts:275, 366, 389 | Straight integer bisection; port as-is with `int`. |
| `const YEARS_WITH_WEEK_53: readonly number[] = Object.freeze((() => { ... for (let year = 1; year <= 9999; year += 1) ... })());` | src/values/classic-temporal.ts:353-359 | A module-load-time computed table of every ISO-long year in 1..9999. In Go build it in `func init()` or as a generated literal; it must be identical or DTWEEK ordinals shift. |
| `const januaryFirst = (5 + daysInPreviousYears(year)) % 7;` and `return januaryFirst === 3 \|\| (januaryFirst === 2 && isLeapYear(year));` | src/values/classic-temporal.ts:348-351 | Both operands are non-negative so JS `%` matches Go `%`. If `daysInPreviousYears` is ever made to return a negative, Go's `%` (truncated, like JS) still matches — but keep the non-negativity. |
| `const year = Number(match[1]);` / `Number(match[2])` / `Number(match[3])` etc. | src/values/classic-temporal.ts:214-216, 466, 521-522, 537-538, 556-557, 570-571, 584-585 | Fixed-width digit runs guaranteed by the regex; `strconv.Atoi` is exact. Note leading zeros: `Number("07")` is 7, `strconv.Atoi("07")` is also 7 — safe. |
| `if ((paddingLength & 1) !== 0)` style bit tests do not appear here, but `DAYS_BY_MONTH[month - 1]!` is indexed by an unchecked `month` in `daysInMonth` | src/values/classic-temporal.ts:206 | Go will panic on an out-of-range index where JS yields `undefined`; every call site validates `month` first (:221, :543, :587), so add an explicit guard rather than relying on that. |

### Go mapping notes

- `SPECIFICATIONS` is a frozen record keyed by EXID character. In Go use `map[byte]temporalSpec` or a `switch`; keep it unexported.
- `CalendarDate`/`ClockTime` (src/values/classic-temporal.ts:125-135) are plain value structs — direct port.
- **Do not use `time.Time`.** The calendar here is a proleptic Julian/Gregorian hybrid with a deliberate 1582 gap (src/values/classic-temporal.ts:198-207, 231-235, 265-267, 293) that Go's `time` package does not model. All arithmetic must stay on the hand-rolled ordinals.
- `dateOrdinal`/`dateFromOrdinal`/`weekOrdinal`/`weekFromOrdinal` should be unexported.
- Sentinels: `ErrTemporalLexical` (:180/:193), `ErrTemporalCalendar` (:218/:221/:227/:232), `ErrTemporalClock` (:319/:324/:327/:329), `ErrTemporalRawRange` (:406/:437), `ErrTemporalWidth` (:426), `ErrTemporalExid` (:157).

---

## unicode-scalar.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `assertUnicodeScalarText` | function | `export function assertUnicodeScalarText(value: string, path: string): void` | src/values/unicode-scalar.ts:2 |
| `assertNulFreeUnicodeScalarText` | function | `export function assertNulFreeUnicodeScalarText(\n  value: string,\n  path: string,\n): void` | src/values/unicode-scalar.ts:20-23 |
| `decodeXmlEntityReference` | function | `export function decodeXmlEntityReference(\n  raw: string,\n  start: number,\n  path: string,\n): { codePoint: number; length: number }` | src/values/unicode-scalar.ts:80-84 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `NAMED_XML_ENTITY_CODE_POINTS` | `new Map([\n  ["amp", 0x26],\n  ["lt", 0x3c],\n  ["gt", 0x3e],\n  ["quot", 0x22],\n  ["apos", 0x27],\n])` | "The five entities XML predefines; `&amp;` and `&lt;` are mandatory escapes." | src/values/unicode-scalar.ts:30-37 |
| `MAXIMUM_CHARACTER_REFERENCE_RUN` | `32` | maximum raw digit-run length in a character reference | src/values/unicode-scalar.ts:46 |
| `DECIMAL_RUN` | `/^[0-9]+$/u` | | src/values/unicode-scalar.ts:47 |
| `HEXADECIMAL_RUN` | `/^[0-9A-Fa-f]+$/u` | | src/values/unicode-scalar.ts:48 |
| high-surrogate range | `codeUnit >= 0xd800 && codeUnit <= 0xdbff` | | src/values/unicode-scalar.ts:5 |
| low-surrogate range | `low >= 0xdc00 && low <= 0xdfff` | | src/values/unicode-scalar.ts:7, 13 |
| code-point ceiling | `if (codePoint > 0x10ffff \|\| (codePoint >= 0xd800 && codePoint <= 0xdfff))` | reference must denote a Unicode scalar | src/values/unicode-scalar.ts:102 |
| hex marker | `} else if (body[1] === "x") {` — lowercase `x` only | `&#X41;` is not accepted (falls to the decimal branch and fails `DECIMAL_RUN`) | src/values/unicode-scalar.ts:97-100 |

### Errors

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `` `${path} contains an isolated surrogate code unit` `` (RangeError) | unpaired high or low surrogate | src/values/unicode-scalar.ts:8, 14 |
| `` `${path} contains NUL` `` (RangeError) | string contains `\0` | src/values/unicode-scalar.ts:26 |
| `` `${path} contains an unsupported XML entity` `` (Error) | digit run too long, wrong radix characters, or unknown named entity | src/values/unicode-scalar.ts:58, 94 |
| `` `${path} contains a truncated XML entity` `` (Error) | no `;` after the `&` | src/values/unicode-scalar.ts:86 |
| `` `${path} contains an empty XML entity` `` (Error) | `&;` | src/values/unicode-scalar.ts:90 |
| `` `${path} contains an out-of-range XML entity` `` (Error) | code point above U+10FFFF or in the surrogate range | src/values/unicode-scalar.ts:103 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Reject isolated UTF-16 surrogates without normalizing valid scalar text." | src/values/unicode-scalar.ts:1 |
| "Reject NUL where the classic wire uses NUL as a value terminator." | src/values/unicode-scalar.ts:19 |
| "A character reference may carry any number of digits: XML 1.0 spells both forms with `+`, so zero padding is a spelling choice and not a different reference. These patterns therefore bound the raw run only, far above any deliberate spelling, so that a very long run cannot become a parse cost. What the reference actually denotes is decided by its value, below." | src/values/unicode-scalar.ts:39-45 |
| "An all-zero run denotes U+0000 rather than nothing, so keep one digit. Our own writers emit `&#00;`, and the readers admit C0 controls in reference position, so this path is exercised by ordinary round-trips." | src/values/unicode-scalar.ts:60-62 |
| "The admitted grammar is the whole XML 1.0 reference grammar: the five predefined named entities plus decimal `&#N;` and hexadecimal `&#xH;` character references of any legal width. Our writers emit a narrow canonical subset of that grammar, but a producer following the specification may send any of it, so the readers accept all of it. Digit runs are bounded so a long reference cannot become a decode cost, zero padding is transparent, and the result is guaranteed to be a Unicode scalar. Callers apply their own code-point policy on top." | src/values/unicode-scalar.ts:67-79 |

### Wire facts asserted by tests

There is a dedicated `test/xml-entity-reference.test.ts`, which is **outside the stated scope**. In-scope coverage:

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| all five named entities decode; every legal spelling (bare decimal, 7-digit zero-padded decimal, lowercase hex, 6-digit uppercase hex) of 15 sampled code points decodes identically | `"admits the whole XML entity grammar a conforming peer may send"` | test/recursive-xrfc.test.ts:1260, 1276-1302 |
| `"&#xD800;"`, `"&#55296;"`, `"&#xDFFF;"`, `"&#57343;"`, `"&#x110000;"`, `"&#1114112;"`, `"&#xFFFE;"`, `"&#65535;"`, `"&#38"`, `"&amp"`, `"&nbsp;"`, `"&AMP;"`, `"&;"`, `"&#;"`, `"&#x;"`, `"&#X41;"`, a 4096-zero-padded reference, and `"a]]>b"` are all rejected | same | test/recursive-xrfc.test.ts:1318-1332 |
| the classic-xRFC reader accepts the same grammar | `"accepts the whole XML entity grammar a conforming peer may send"` | test/classic-xrfc.test.ts:650 |

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `const codeUnit = value.charCodeAt(index);` and the whole surrogate-pair walk | src/values/unicode-scalar.ts:3-16 | **`assertUnicodeScalarText` has no meaningful Go analogue.** Go `string` is UTF-8; an isolated surrogate cannot be represented as a valid UTF-8 sequence. The Go port must instead validate `utf8.ValidString` **and** reject the CESU-8/WTF-8 encodings of U+D800–U+DFFF (`ED A0 80`–`ED BF BF`), because those *can* appear in bytes read off the wire. Record this as a semantic difference in provenance. |
| `const significant = digits.replace(/^0+/u, "") \|\| "0"; return Number.parseInt(significant, radix);` | src/values/unicode-scalar.ts:63-64 | `Number.parseInt` on a ≤32-character run cannot exceed float64 exactness for radix 16 (max 16^32 ≈ 3.4e38 — **this does exceed 2^53**). The result is then compared against `0x10ffff` at :102, so an imprecise large value still fails the range check. In Go, use `strconv.ParseUint(significant, radix, 32)` and treat `ErrRange` as the out-of-range error at :103 — do **not** use `ParseUint(..., 64)` and compare, since a 32-hex-digit run overflows 64 bits too. |
| `if (digits.length > MAXIMUM_CHARACTER_REFERENCE_RUN \|\| !run.test(digits))` — the length bound is applied to the **raw** run, before zero-stripping | src/values/unicode-scalar.ts:57 | A 33-zero-padded reference is refused even though it denotes a valid scalar (test/recursive-xrfc.test.ts:1328 relies on this). Apply the bound before stripping. |
| `const semicolon = raw.indexOf(";", start + 1); ... return { codePoint, length: semicolon + 1 - start };` | src/values/unicode-scalar.ts:85-105 | `length` is in **UTF-16 code units** but the reference body is ASCII-only, so it equals the byte length. Safe to port as a byte offset. |
| `String.fromCodePoint(codePoint)` at the call sites | src/values/classic-xrfc.ts:751, src/values/recursive-xrfc.ts:1443 | In Go, `string(rune(codePoint))` — but note `rune` conversion of an invalid code point silently yields U+FFFD, so the range check at src/values/unicode-scalar.ts:102 must run first. |

### Go mapping notes

- `assertNulFreeUnicodeScalarText` becomes `strings.IndexByte(s, 0) >= 0` plus the UTF-8 validation above.
- `NAMED_XML_ENTITY_CODE_POINTS` → `map[string]rune`, unexported.
- Sentinels: `ErrIsolatedSurrogate`, `ErrNulInText`, `ErrXmlEntityUnsupported`, `ErrXmlEntityTruncated`, `ErrXmlEntityEmpty`, `ErrXmlEntityOutOfRange`. The distinct message texts are matched by test regexes (test/recursive-xrfc.test.ts:1331 matches `/entity|non-canonical/u`), so keep the word "entity" in the rendered messages.

---

## classic-structure.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `ClassicStructureInput` | type | `export type ClassicStructureInput = Readonly<Record<string, unknown>>;` | src/values/classic-structure.ts:51 |
| `ClassicStructureOutput` | type | `export type ClassicStructureOutput = Readonly<Record<string, unknown>>;` | src/values/classic-structure.ts:52 |
| `ClassicStructureCodecOptions` | interface | `readonly int8Mode?: ClassicInt8Mode;` / `readonly bcd?: ClassicBcdMode;` | src/values/classic-structure.ts:54-59 |
| `snapshotClassicStructureDefinition` | function | `export function snapshotClassicStructureDefinition(\n  definition: RfcStructureDefinition,\n  requestedName?: string,\n): RfcStructureDefinition` | src/values/classic-structure.ts:68-71 |
| `validateClassicStructureCodec` | function | `export function validateClassicStructureCodec(\n  definition: RfcStructureDefinition,\n  requestedName?: string,\n): RfcStructureDefinition` | src/values/classic-structure.ts:300-303 |
| `classicStructureHasDynamicFields` | function | `export function classicStructureHasDynamicFields(\n  definition: RfcStructureDefinition,\n): boolean` | src/values/classic-structure.ts:312-314 |
| `encodeClassicStructure` | function | `export function encodeClassicStructure(\n  definition: RfcStructureDefinition,\n  input: ClassicStructureInput,\n  options: ClassicStructureCodecOptions = {},\n): Buffer` | src/values/classic-structure.ts:543-547 |
| `decodeClassicStructure` | function | `export function decodeClassicStructure(\n  definition: RfcStructureDefinition,\n  value: Uint8Array,\n  options: ClassicStructureCodecOptions = {},\n): ClassicStructureOutput` | src/values/classic-structure.ts:598-602 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `MAX_CLASSIC_STRUCTURE_FIELDS` | `100_000` | field-count ceiling | src/values/classic-structure.ts:61 |
| `MAX_CLASSIC_STRUCTURE_BYTE_LENGTH` | `DEFAULT_MAX_CPIC_FIELD_LENGTH` | "A fixed structure travels in one CPIC field, so it must not exceed the transport's per-field allocation policy." Numeric value **not stated in source within scope**. | src/values/classic-structure.ts:62-64 |
| Unicode character width | `return field.internalLength / 2;` after `if ((field.internalLength & 1) !== 0)` | every CHAR-like field is UTF-16LE, two bytes per character | src/values/classic-structure.ts:175-180 |
| `D` (DATE) width | `if (field.internalLength !== 16)` → "DATE must occupy 16 Unicode bytes" | 8 characters × 2 | src/values/classic-structure.ts:234-236, 370-372 |
| `T` (TIME) width | `if (field.internalLength !== 12)` → "TIME must occupy 12 Unicode bytes" | 6 characters × 2 | src/values/classic-structure.ts:238-241, 381-383 |
| `F` width | `8` | FLOAT | src/values/classic-structure.ts:246-248 |
| `I` width | `4` | INT4 | src/values/classic-structure.ts:251-253 |
| `s` width | `2` | INT2 | src/values/classic-structure.ts:256-258 |
| `b` width | `1` | INT1 | src/values/classic-structure.ts:261-263 |
| `8` width | `8` | INT8 | src/values/classic-structure.ts:266-268 |
| `a` width | `8` | DECF16 | src/values/classic-structure.ts:275-277 |
| `e` width | `16` | DECF34 | src/values/classic-structure.ts:280-282 |
| `g` width | `8` | "STRING descriptor must occupy 8 bytes" | src/values/classic-structure.ts:285-287 |
| `y` width | `8` | "XSTRING descriptor must occupy 8 bytes" | src/values/classic-structure.ts:290-292 |
| INT4 domain | `integer(value, -0x8000_0000, 0x7fff_ffff, fieldPath)` | | src/values/classic-structure.ts:413 |
| INT2 domain | `integer(value, -0x8000, 0x7fff, fieldPath)` | | src/values/classic-structure.ts:417 |
| INT1 domain | `integer(value, 0, 0xff, fieldPath)` — **unsigned** | written with `writeUInt8` | src/values/classic-structure.ts:421 |
| initial values by EXID | `"C" → ""`, `"N" → "0".repeat(characterLength(...))`, `"D" → "00000000"`, `"T" → "000000"`, `"X" → Buffer.alloc(0)`, `"F"/"I"/"s"/"b" → 0`, `"8" → classicInt8InitialValue(int8Mode)`, `"P" → "0"`, `"a"/"e" → "0"` | | src/values/classic-structure.ts:190-215 |
| CHAR/NUMC alignment fill | `const fill = field.exid === "C" ? " " : "0";` written as UTF-16LE into the gap to the next field | trailing alignment tail | src/values/classic-structure.ts:576-592 |

### Errors

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `"classic structure definition must be an object"` (TypeError) | | src/values/classic-structure.ts:73 |
| `` `${expectedName} structure definition has an invalid name` `` | name missing/empty/mismatched | src/values/classic-structure.ts:82 |
| `` `${name} structure byteLength must be a non-negative safe integer` `` (RangeError) | | src/values/classic-structure.ts:85 |
| `` `${name} structure byteLength exceeds ${MAX_CLASSIC_STRUCTURE_BYTE_LENGTH}` `` (RangeError) | | src/values/classic-structure.ts:88-90 |
| `` `${name} structure fields must be an array` `` (TypeError) | | src/values/classic-structure.ts:93 |
| `` `${name} structure field count exceeds ${MAX_CLASSIC_STRUCTURE_FIELDS}` `` (RangeError) | | src/values/classic-structure.ts:97-99 |
| `` `${name} structure field ${index} must be an object` `` (TypeError) | | src/values/classic-structure.ts:108 |
| `` `${name} structure field ${index} has invalid geometry` `` | wrong tableName, empty/duplicate fieldName, wrong position, non-monotonic or unsafe offset, negative internalLength/decimals | src/values/classic-structure.ts:119-133 |
| `` `${name}.${field.fieldName} exceeds the structure byteLength` `` | field end past `byteLength` | src/values/classic-structure.ts:136 |
| `` `${fieldPath} must be an integer in ${minimum}..${maximum}` `` (RangeError) | integer field out of range | src/values/classic-structure.ts:168-170 |
| `` `${fieldPath} Unicode character width must be even` `` | odd `internalLength` on a character field | src/values/classic-structure.ts:177 |
| `` `${fieldPath} classic RFC type ${field.exid} is not implemented` `` | unknown EXID (three sites) | src/values/classic-structure.ts:214, 295, 458, 538 |
| `` `${fieldPath} compact temporal type ${field.exid} must occupy ${byteLength} bytes` `` | | src/values/classic-structure.ts:222-224, 344-346 |
| `` `${definition.name} contains STRING/XSTRING fields and requires xRFC XML serialization` `` | fixed codec used on a dynamic structure | src/values/classic-structure.ts:325-327 |
| `` `${fieldPath} expects a string` `` (TypeError) | `C` field with non-string | src/values/classic-structure.ts:355 |
| `` `${fieldPath} expects at most ${characters} decimal digits` `` (TypeError) | `N` field non-digit or too long | src/values/classic-structure.ts:362-364 |
| `` `${fieldPath} expects YYYYMMDD` `` / `` `${fieldPath} expects HHMMSS` `` (TypeError) | non-string `D`/`T` | src/values/classic-structure.ts:373, 384 |
| `` `${fieldPath} expects Uint8Array bytes` `` (TypeError) | `X` field non-bytes | src/values/classic-structure.ts:393 |
| `` `${fieldPath} accepts at most ${field.internalLength} bytes` `` (RangeError) | `X` overflow | src/values/classic-structure.ts:397-399 |
| `` `${fieldPath} expects a finite 8-byte float` `` (TypeError) | `F` non-number or non-finite | src/values/classic-structure.ts:406 |
| `` `${fieldPath} received a non-finite 8-byte float` `` (TypeError) | decoded `F` is NaN/Inf | src/values/classic-structure.ts:499 |
| `"classic structure codec options must be an object"` (TypeError) | | src/values/classic-structure.ts:549, 604 |
| `` `${normalized.name} contains unknown field ${name}` `` | input key not in the definition | src/values/classic-structure.ts:560 |
| `` `${path(normalized, field)} has an odd Unicode alignment tail` `` | odd padding gap after a C/N field | src/values/classic-structure.ts:581-583 |
| `` `${normalized.name} structure expects Uint8Array bytes` `` (TypeError) | | src/values/classic-structure.ts:611 |
| `` `${normalized.name} structure must contain exactly ${normalized.byteLength} bytes; received ${byteLength}` `` (RangeError) | | src/values/classic-structure.ts:615-618 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "A fixed structure travels in one CPIC field, so it must not exceed the transport's per-field allocation policy." | src/values/classic-structure.ts:62-63 |
| "Snapshot and validate fixed structure geometry before value allocation." | src/values/classic-structure.ts:67 |
| "The encoder owns the exact 1..16 byte and decimal-scale rules." | src/values/classic-structure.ts:271 |
| "Validate every field and the captured Unicode classic-row geometry." | src/values/classic-structure.ts:299 |
| "True when the structure requires the xRFC XML deep-value serializer." | src/values/classic-structure.ts:311 |
| "IEEE-754 binary64 for the admitted Unicode-4103 profile" (on the `F` case) | src/values/classic-structure.ts:404 |
| "Encode one fixed-layout classic Unicode structure." | src/values/classic-structure.ts:542 |
| "Decode one fixed-layout classic Unicode structure into plain values." | src/values/classic-structure.ts:597 |

### Wire facts asserted by tests

No in-scope test file targets `classic-structure.ts` directly. The `EXTENDED_ROW` and `STFC_ROW` definitions in test/classic-xrfc.test.ts:11-106 exercise `validateClassicStructureCodec` indirectly through `normalizeDefinition` (src/values/classic-xrfc.ts:244).

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `if (!Number.isSafeInteger(byteLength) \|\| byteLength < 0)` and the same guard on `offset`, `internalLength`, `decimals`, `end`, `count` | src/values/classic-structure.ts:84, 96, 125-130, 135 | Every geometry field arrives as an untrusted JS number and must be integral and within ±2^53−1. In Go these are already `int`; the port must instead validate that the JSON/metadata source did not overflow, and must keep the `< 0` checks. |
| `if ((field.internalLength & 1) !== 0)` | src/values/classic-structure.ts:176, 580 | JS `&` coerces to **int32**: an `internalLength` above 2^31 would be truncated before the parity test. Bounded by `MAX_CLASSIC_STRUCTURE_BYTE_LENGTH` at :87, so safe here — in Go use `%2` on `int` and keep the bound. |
| `target.writeDoubleLE(value, offset)` / `source.readDoubleLE(field.offset)` with `Number.isFinite` guards on both sides | src/values/classic-structure.ts:405-408, 497-501 | `math.Float64bits` + `binary.LittleEndian.PutUint64`, and `math.Float64frombits`. The finiteness check on **decode** (:498) means a wire NaN/Inf is an error, not a value — reproduce it. Note this also rejects −0? No: `Number.isFinite(-0) === true`, so −0 decodes normally. |
| `target.writeBigInt64LE(encodeClassicInt8(value, int8Mode, fieldPath), offset)` and `decodeClassicInt8(source.readBigInt64LE(field.offset), ...)` | src/values/classic-structure.ts:425-428, 514-518 | `int64` little-endian; see classic-int8.ts notes. |
| `target.writeInt32LE(...)`, `writeInt16LE(...)`, `writeUInt8(...)`; `readInt32LE`, `readInt16LE`, `readUInt8` | src/values/classic-structure.ts:413-421, 505-511 | INT4/INT2 are **signed**, INT1 is **unsigned** in both directions. Do not unify them. |
| `Buffer.from(classicDateWireText(value, fieldPath), "utf16le")` and the C/N fill `Buffer.from(fill.repeat(paddingLength / 2), "utf16le")` | src/values/classic-structure.ts:374-376, 587-590 | UTF-16**LE** on the wire. In Go use `unicode/utf16` + explicit LE byte writes; `[]byte(s)` would produce UTF-8 and silently corrupt every character field. |
| `value.padStart(characters, "0")` for NUMC | src/values/classic-structure.ts:366 | Pads by **character count**, not bytes; the byte count is `characters * 2`. |
| `const supplied = Object.prototype.hasOwnProperty.call(input, field.fieldName);` | src/values/classic-structure.ts:566 | Distinguishes "absent" from "present and undefined". In Go use a `map[string]any` with the two-value lookup, or an explicit presence set. |

### Go mapping notes

- `validatedStructureDefinitions = new WeakSet<object>()` (src/values/classic-structure.ts:65, 75, 148) memoizes "already validated" by object identity. Go has no `WeakSet`; either validate every time (cheap) or return a distinct `ValidatedStructure` type that can only be constructed by the validator — the latter matches the intent at :75.
- `Object.freeze` on the snapshot (src/values/classic-structure.ts:110, 143-147) → return by value / unexported fields.
- `Object.defineProperty(result, field.fieldName, {...})` on decode (src/values/classic-structure.ts:623-634) exists so that a field literally named `__proto__` becomes an own data property rather than mutating the prototype (asserted for the xRFC sibling at test/classic-xrfc.test.ts:149-178). In Go a `map[string]any` has no such hazard — drop the mechanism, keep the test as a map-key test.
- `assertFixedStructure` (src/values/classic-structure.ts:321-329) is the fixed-vs-xRFC fork; keep it unexported and make the error a sentinel `ErrRequiresXrfc` so the caller can route.

---

## classic-xrfc.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `ClassicXrfcKind` | type | `export type ClassicXrfcKind = "structure" \| "table";` | src/values/classic-xrfc.ts:45 |
| `ClassicXrfcLimits` | interface | `maxCellBytes?`, `maxRowBytes?`, `maxParameterBytes?`, `maxRows?` (all `readonly … number`) | src/values/classic-xrfc.ts:47-55 |
| `ClassicXrfcOptions` | interface | `export interface ClassicXrfcOptions extends ClassicXrfcLimits {\n  readonly int8Mode?: ClassicInt8Mode;\n  readonly bcd?: ClassicBcdMode;\n}` | src/values/classic-xrfc.ts:57-60 |
| `NormalizedClassicXrfcLimits` | interface | four required `readonly … number` fields | src/values/classic-xrfc.ts:62-67 |
| `normalizeClassicXrfcLimits` | function | `export function normalizeClassicXrfcLimits(\n  limits: ClassicXrfcLimits,\n): NormalizedClassicXrfcLimits` | src/values/classic-xrfc.ts:133-135 |
| `assertClassicXrfcXmlName` | function | `export function assertClassicXrfcXmlName(value: string, label: string): void` | src/values/classic-xrfc.ts:167 |
| `classicXrfcOpenTagByteLength` | function | `export function classicXrfcOpenTagByteLength(name: string): number` — body `return name.length + 2;` | src/values/classic-xrfc.ts:175-177 |
| `classicXrfcCloseTagByteLength` | function | `export function classicXrfcCloseTagByteLength(name: string): number` — body `return name.length + 3;` | src/values/classic-xrfc.ts:179-181 |
| `checkedClassicXrfcLength` | function | `export function checkedClassicXrfcLength(\n  current: number,\n  additional: number,\n  label: string,\n): number` | src/values/classic-xrfc.ts:183-187 |
| `escapedClassicXrfcXmlByteLength` | function | `export function escapedClassicXrfcXmlByteLength(\n  value: string,\n  path: string,\n): number` | src/values/classic-xrfc.ts:206-209 |
| `validateClassicXrfcDefinition` | function | `export function validateClassicXrfcDefinition(\n  definition: RfcStructureDefinition,\n): RfcStructureDefinition` | src/values/classic-xrfc.ts:263-265 |
| `writeClassicXrfcOpenTag` | function | `export function writeClassicXrfcOpenTag(\n  target: Buffer,\n  offset: number,\n  name: string,\n): number` | src/values/classic-xrfc.ts:538-542 |
| `writeClassicXrfcCloseTag` | function | `export function writeClassicXrfcCloseTag(\n  target: Buffer,\n  offset: number,\n  name: string,\n): number` | src/values/classic-xrfc.ts:546-550 |
| `writeEscapedClassicXrfcText` | function | `export function writeEscapedClassicXrfcText(\n  target: Buffer,\n  offset: number,\n  value: string,\n): number` | src/values/classic-xrfc.ts:554-558 |
| `encodeClassicXrfcParameter` | function | `export function encodeClassicXrfcParameter(\n  parameterName: string,\n  definition: RfcStructureDefinition,\n  kind: ClassicXrfcKind,\n  value: unknown,\n  options: ClassicXrfcOptions = {},\n): Buffer` | src/values/classic-xrfc.ts:601-607 |
| `ExactClassicXrfcParser` | class | `export class ExactClassicXrfcParser` with `constructor(text: string, limits: NormalizedClassicXrfcLimits)` and methods `startsWithTag`, `open`, `close`, `cell`, `rowByteLength`, `position`, `finish` | src/values/classic-xrfc.ts:673-737 |
| `decodeClassicXrfcBase64` | function | `export function decodeClassicXrfcBase64(\n  value: string,\n  path: string,\n  maximum: number,\n): Buffer` | src/values/classic-xrfc.ts:761-765 |
| `decodeClassicXrfcParameterName` | function | `export function decodeClassicXrfcParameterName(\n  value: Uint8Array,\n  limits: ClassicXrfcLimits = {},\n): string` | src/values/classic-xrfc.ts:976-979 |
| `decodeClassicXrfcParameter` | function | `export function decodeClassicXrfcParameter(\n  parameterName: string,\n  definition: RfcStructureDefinition,\n  kind: ClassicXrfcKind,\n  value: Uint8Array,\n  options: ClassicXrfcOptions = {},\n): Readonly<Record<string, unknown>> \| readonly Readonly<Record<string, unknown>>[]` | src/values/classic-xrfc.ts:1008-1014 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `SIMPLE_XML_NAME` | `/^[A-Za-z_][A-Za-z0-9_]*$/u` | "must be a simple XML name supported by the proven xRFC subset" | src/values/classic-xrfc.ts:95, 167-172 |
| `SUPPORTED_XRFC_FIELD_TYPES` | `new Set([\n  "I",\n  "C",\n  "N",\n  "D",\n  "T",\n  "X",\n  "P",\n  "F",\n  "8",\n  "g",\n  "y",\n])` | eleven EXIDs implemented for the flat xRFC subset | src/values/classic-xrfc.ts:96-108 |
| `CANONICAL_INTEGER` | `/^(?:0\|-[1-9][0-9]*\|[1-9][0-9]*)$/u` | canonical INT8 spelling on the wire | src/values/classic-xrfc.ts:109 |
| `FINITE_FLOAT_LEXICAL` | `/^[+-]?(?:(?:[0-9]+(?:\.[0-9]*)?)\|(?:\.[0-9]+))(?:[eE][+-]?[0-9]+)?$/u` | accepted FLOAT spellings on read | src/values/classic-xrfc.ts:113-114 |
| open-tag cost | `name.length + 2` | `<NAME>` | src/values/classic-xrfc.ts:176 |
| close-tag cost | `name.length + 3` | `</NAME>` | src/values/classic-xrfc.ts:180 |
| item wrapper cost | `let encodedByteLength = itemWrapper ? 13 : 0; // <item></item>` | | src/values/classic-xrfc.ts:505 |
| escape cost `&` | `checkedClassicXrfcLength(byteLength, 5, path); // &#38;` | | src/values/classic-xrfc.ts:216-218 |
| escape cost `<`/`>` | `checkedClassicXrfcLength(byteLength, 5, path); // &#60; / &#62;` | | src/values/classic-xrfc.ts:219-222 |
| escapes written | `"&#38;"`, `"&#60;"`, `"&#62;"` for `&`, `<`, `>` | numeric references, **not** `&amp;`/`&lt;`/`&gt;` | src/values/classic-xrfc.ts:559-573 |
| XML 1.0 code-point policy | `codePoint === 0 \|\| (codePoint < 0x20 && codePoint !== 0x09 && codePoint !== 0x0a && codePoint !== 0x0d) \|\| codePoint === 0xfffe \|\| codePoint === 0xffff` | refused characters | src/values/classic-xrfc.ts:195-204 |
| base64 encoded size | `const encodedByteLength = Math.ceil(plannedByteLength / 3) * 4;` | | src/values/classic-xrfc.ts:341 |
| base64 canonical grammar | `/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==\|[A-Za-z0-9+/]{3}=)?$/u` plus `(value.length & 3) !== 0` | | src/values/classic-xrfc.ts:768-771 |
| base64 decoded size | `const decodedByteLength = (value.length / 4) * 3 -\n    (value.endsWith("==") ? 2 : value.endsWith("=") ? 1 : 0);` | | src/values/classic-xrfc.ts:773-774 |
| DATE wire form | `date.replace(/^(\d{4})(\d{2})(\d{2})$/u, "$1-$2-$3")`, blank → `""` | | src/values/classic-xrfc.ts:313-319 |
| TIME wire form | `time.replace(/^(\d{2})(\d{2})(\d{2})$/u, "$1:$2:$3")`, blank → `""` | | src/values/classic-xrfc.ts:321-327 |
| DATE read form | `/^\d{4}-\d{2}-\d{2}$/u` then `value.replaceAll("-", "")` | | src/values/classic-xrfc.ts:826-834 |
| TIME read form | `/^\d{2}:\d{2}:\d{2}$/u` then `value.replaceAll(":", "")` | | src/values/classic-xrfc.ts:835-843 |
| `maxRows` ceiling | `boundedLimit(limits.maxRows, DEFAULT_MAX_CPIC_FIELD_COUNT, 0xffff_ffff, "maxRows")` | | src/values/classic-xrfc.ts:158-163 |
| BOM rejection | `encoded[0] === 0xef && encoded[1] === 0xbb && encoded[2] === 0xbf` | | src/values/classic-xrfc.ts:991-998, 1033-1040 |
| top-level tag grammar | `/^<([A-Za-z_][A-Za-z0-9_]*)>/u` | no XML prolog accepted | src/values/classic-xrfc.ts:1000 |
| initial cell values | `"I" → 0`, `"C"/"g" → ""`, `"N" → ""`, `"D" → "00000000"`, `"T" → "000000"`, `"X"/"y" → Buffer.alloc(0)`, `"P" → "0"`, `"F" → 0`, `"8" → classicInt8InitialValue(int8Mode)` | | src/values/classic-xrfc.ts:281-311 |

### Errors

Selected, verbatim:

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `` `${label} must be an integer in 0..${maximum}` `` (RangeError) | limit not a safe integer in range | src/values/classic-xrfc.ts:128 |
| `"xRFC limits must be an object"` (TypeError) | | src/values/classic-xrfc.ts:137 |
| `` `${label} must be a simple XML name supported by the proven xRFC subset` `` | name fails `SIMPLE_XML_NAME` | src/values/classic-xrfc.ts:169-171 |
| `` `${label} encoded length is unsafe` `` (RangeError) | length sum is not a safe integer | src/values/classic-xrfc.ts:190 |
| `` `${path} contains a character unsupported by XML 1.0` `` (RangeError) | | src/values/classic-xrfc.ts:202 |
| `` `${normalized.name} has no STRING/XSTRING field requiring xRFC XML` `` | xRFC codec used on a purely fixed structure | src/values/classic-xrfc.ts:246-248 |
| `` `${normalized.name}.${field.fieldName} type ${field.exid} is not implemented for the proven xRFC XML subset` `` | EXID outside `SUPPORTED_XRFC_FIELD_TYPES` | src/values/classic-xrfc.ts:253-256 |
| `` `${path} expects a signed 32-bit integer` `` (RangeError) | INT4 cell out of range | src/values/classic-xrfc.ts:276 |
| `` `${path} accepts at most ${exactLength} bytes` `` (RangeError) | fixed `X` overflow | src/values/classic-xrfc.ts:338 |
| `` `${path} base64 value exceeds the configured encoded-byte limits` `` (RangeError) | | src/values/classic-xrfc.ts:348-350 |
| `` `${path} does not fit CHAR(${capacity})` `` (RangeError) | | src/values/classic-xrfc.ts:386, 814 |
| `` `${path} padded NUM value exceeds the configured encoded-byte limits` `` (RangeError) | declared NUMC width exceeds the byte budget **before** any padding is materialized | src/values/classic-xrfc.ts:399-401 |
| `` `${path} expects at most ${capacity} decimal digits` `` (TypeError) | | src/values/classic-xrfc.ts:408-410 |
| `` `${path} expects a finite number` `` (TypeError) | FLOAT cell | src/values/classic-xrfc.ts:446 |
| `` `${path} expects Unicode text` `` (TypeError) | `g` cell | src/values/classic-xrfc.ts:455 |
| `` `${path} XML value exceeds ${limits.maxCellBytes} encoded bytes` `` (RangeError) | | src/values/classic-xrfc.ts:471-473, 717-719 |
| `` `${rowPath} expects a structure object` `` (TypeError) | | src/values/classic-xrfc.ts:495 |
| `` `${rowPath} contains unknown field ${name}` `` | | src/values/classic-xrfc.ts:501 |
| `` `${rowPath} XML row exceeds ${limits.maxRowBytes} encoded bytes` `` (RangeError) | | src/values/classic-xrfc.ts:522-524, 968-970 |
| `` `${parameterName} xRFC XML exceeds ${normalizedLimits.maxParameterBytes} bytes` `` (RangeError) | | src/values/classic-xrfc.ts:618-620, 625-627 |
| `` `${parameterName} row count exceeds ${normalizedLimits.maxRows}` `` (RangeError) | | src/values/classic-xrfc.ts:637-639, 1065-1067 |
| `` `${parameterName} xRFC XML encoder length invariant failed` `` | written offset ≠ preflighted length | src/values/classic-xrfc.ts:668 |
| `` `xRFC XML expected ${token} at character ${this.#offset}` `` | tag mismatch | src/values/classic-xrfc.ts:691, 700 |
| `` `xRFC XML ${path} is truncated` `` | no `<` after a cell | src/values/classic-xrfc.ts:708 |
| `` `${path} contains invalid XML character data` `` | cell contains `]]>` | src/values/classic-xrfc.ts:712 |
| `` `xRFC XML has trailing content at character ${this.#offset}` `` | | src/values/classic-xrfc.ts:734 |
| `` `${path} contains non-canonical base64` `` | grammar failure or re-encode mismatch | src/values/classic-xrfc.ts:771, 780 |
| `` `${path} decoded bytes exceed ${maximum}` `` (RangeError) | | src/values/classic-xrfc.ts:776 |
| `` `${path} contains a non-canonical INT4 value` `` / `` `${path} INT4 value is out of range` `` | | src/values/classic-xrfc.ts:798, 806 |
| `` `${path} contains a non-canonical NUM value` `` | | src/values/classic-xrfc.ts:821 |
| `` `${path} contains a non-canonical xRFC DATE` `` / `` ... xRFC TIME` `` | | src/values/classic-xrfc.ts:829, 838 |
| `` `${path} fixed byte value must contain ${field.internalLength} bytes` `` (RangeError) | | src/values/classic-xrfc.ts:856-858 |
| `` `${path} contains an invalid FLOAT` `` | lexical failure or non-finite | src/values/classic-xrfc.ts:879, 883 |
| `` `${path} contains a non-canonical INT8 value` `` | fails `CANONICAL_INTEGER` or longer than 20 characters | src/values/classic-xrfc.ts:889 |
| `` `${path} decoded value exceeds the ${budget.limits.maxCellBytes}-byte cell limit` `` (RangeError) | | src/values/classic-xrfc.ts:917-919 |
| `` `${path} decoded output exceeds the ${budget.limits.maxParameterBytes}-byte parameter limit` `` (RangeError) | | src/values/classic-xrfc.ts:927-929 |
| `"xRFC XML parameter must not contain a UTF-8 BOM"` / `` `${parameterName} xRFC XML must not contain a UTF-8 BOM` `` | | src/values/classic-xrfc.ts:997, 1039 |
| `"xRFC XML parameter lacks a supported top-level tag"` | | src/values/classic-xrfc.ts:1002 |
| `"xRFC parameter kind must be structure or table"` (TypeError) | | src/values/classic-xrfc.ts:610, 1017 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Maximum UTF-8/base64 bytes in one XML field value." | src/values/classic-xrfc.ts:48 |
| "Maximum encoded bytes in one structure or table row." | src/values/classic-xrfc.ts:50 |
| "Maximum encoded bytes in one complete xRFC XML parameter." | src/values/classic-xrfc.ts:52 |
| "The full lexical space a conforming producer may write for a float: a leading \"+\", leading zeros, \"1.\" and \".5\" are all legal spellings of an unambiguous value, so the reader takes them. Matches the recursive sibling." | src/values/classic-xrfc.ts:110-112 |
| "Validate the supported flat xRFC row subset without touching values." | src/values/classic-xrfc.ts:262 |
| "The normal planner validates the declared width before padding. Do not materialize metadata-controlled output in this default-value helper." | src/values/classic-xrfc.ts:286-288 |
| "Encode the supported xRFC XML subset for one flat structure or table. The returned buffer owns snapshots of every caller-supplied binary value." | src/values/classic-xrfc.ts:597-600 |
| "Return the strict top-level parameter name without accepting XML prologs." | src/values/classic-xrfc.ts:975 |
| "Decode the exact, attribute-free flat xRFC XML subset." | src/values/classic-xrfc.ts:1007 |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| exact request XML for the two captured rows (see conformance section) | `"encodes the STFC_DEEP_TABLE xRFC XML request exactly"` | test/classic-xrfc.test.ts:108-123 |
| a response containing `&#60;`, `&#38;`, `&#34;` decodes to `A<&"-nested`, and `"3q2+7w=="` to `deadbeef` | `"decodes the STFC_DEEP_TABLE response and numeric entities"` | test/classic-xrfc.test.ts:125-147 |
| a field named `__proto__` becomes an own data property, prototype unchanged | `"constructs flat __proto__ fields as own data without prototype mutation"` | test/classic-xrfc.test.ts:149-178 |
| the Unicode request is exactly `163` bytes; the response is exactly `240` bytes | `"matches the Unicode and explicit-empty STFC vector"` | test/classic-xrfc.test.ts:180, 191, 206 |
| empty `g` and `y` cells serialize as `<STR></STR><XSTR></XSTR>` | `"round-trips initial, Unicode, astral, combining, and arbitrary binary cells"` | test/classic-xrfc.test.ts:217, 241 |
| exact extended-scalar XML: `<NUM>0012</NUM><DATE>2026-07-17</DATE><TIME>15:45:30</TIME><BYTE>qgA=</BYTE><BCD>12.34</BCD><FLOAT>-0</FLOAT><INT8>-9007199254740993</INT8><TEXT>ready</TEXT>` | `"round-trips the extended flat xRFC scalar set with compatibility modes"` | test/classic-xrfc.test.ts:244, 261-267 |
| a 1-byte value in a 2-byte `X` field is zero-padded to `qgA=` and decodes to `Buffer.from([0xaa, 0])` | same | test/classic-xrfc.test.ts:253, 264, 279 |
| `-0` survives as the literal text `-0` and `Object.is(decoded.FLOAT, -0) === true` | same | test/classic-xrfc.test.ts:265, 285 |
| accepted FLOAT spellings and their values: `"1.5"→1.5, "+1.5"→1.5, "01.5"→1.5, "1."→1, ".5"→0.5, "-2"→-2, "1e3"→1000, "+1.5E+02"→150, "0"→0` | same | test/classic-xrfc.test.ts:298-303 |
| rejected FLOAT spellings: `"", ".", "+", "1.5.5", "0x10", "1e", "NaN", "Infinity", " 1"` | same | test/classic-xrfc.test.ts:304-306 |
| blank DATE/TIME (`""` or all-spaces) serialize as `<DATE></DATE><TIME></TIME>` and decode back to `""` | `"canonicalizes flat xRFC blank DATE/TIME and rejects malformed extended cells"` | test/classic-xrfc.test.ts:309-329 |
| a structure parameter emits no `<item>` wrapper | `"supports a flat dynamic structure without item wrappers"` | test/classic-xrfc.test.ts:345-358 |

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `const result = current + additional; if (!Number.isSafeInteger(result)) throw new RangeError(...)` | src/values/classic-xrfc.ts:188-192 | This is the project's overflow guard for length accumulation, and it exists **because JS silently loses integer precision above 2^53**. In Go the equivalent hazard is `int` wraparound: implement as `if result < current { overflow }` on `int`/`int64`, or use `math.MaxInt` checks. Do not delete the function as "unnecessary in Go". |
| `if (!Number.isSafeInteger(normalized) \|\| normalized < 0 \|\| normalized > maximum)` in `boundedLimit` | src/values/classic-xrfc.ts:122-129 | Untrusted caller limits. In Go these are `int`; keep the `< 0` and `> maximum` checks, drop the integrality check. |
| `maxRows` maximum is `0xffff_ffff` | src/values/classic-xrfc.ts:162 | `uint32` max as a *limit value*, not a row count type. On 32-bit Go builds `int` cannot hold it — use `int64` or `uint32` for the limit field. |
| `typeof value !== "number" \|\| !Number.isSafeInteger(value) \|\| value < -0x8000_0000 \|\| value > 0x7fff_ffff` then `String(value)` | src/values/classic-xrfc.ts:270-278 | INT4 cell: `int32` + `strconv.Itoa`. |
| `text = Object.is(value, -0) ? "-0" : String(value);` for FLOAT | src/values/classic-xrfc.ts:448 | **JS `String(double)` again.** `String(1.5)` is `"1.5"`, `String(1e21)` is `"1e+21"`, `String(150)` is `"150"`. Go's `strconv.FormatFloat(v,'g',-1,64)` gives `"1.5"`, `"1e+21"`, `"150"` — mostly aligned, but the crossover thresholds differ (JS switches to exponential at ≥1e21 and <1e-6; Go `'g'` switches based on the exponent vs precision). Implement JS's `Number::toString` rules explicitly, and use `math.Signbit` for the `-0` case (test/classic-xrfc.test.ts:265, 285). |
| `const decoded = Number(value);` for INT4, then `Number.isSafeInteger` + range | src/values/classic-xrfc.ts:800-807 | Regex-validated canonical integer text; use `strconv.ParseInt(value, 10, 32)`. The safe-integer test is redundant given the range test but harmless. |
| `const decoded = Number(value); if (!Number.isFinite(decoded))` for FLOAT | src/values/classic-xrfc.ts:881-884 | `strconv.ParseFloat(value, 64)`; note `FINITE_FLOAT_LEXICAL` (:113-114) already excludes `NaN`/`Infinity`/hex, but the finiteness check still catches overflow to `±Inf` from a huge exponent. Go's `ParseFloat` returns `ErrRange` **and** `±Inf` there — treat that as the invalid-FLOAT error. |
| `if (!CANONICAL_INTEGER.test(value) \|\| value.length > 20) throw ...; return decodeClassicInt8(BigInt(value), int8Mode, path);` | src/values/classic-xrfc.ts:888-891 | `strconv.ParseInt(value, 10, 64)` after the regex. The `> 20` bound precedes the parse so a 10000-digit run never reaches it. |
| `const encodedByteLength = Math.ceil(plannedByteLength / 3) * 4;` | src/values/classic-xrfc.ts:341 | `((n + 2) / 3) * 4` in integer arithmetic. `Math.ceil` on a float division is exact for the sizes involved but the Go form is clearer and cannot drift. |
| `const decodedByteLength = (value.length / 4) * 3 - (value.endsWith("==") ? 2 : value.endsWith("=") ? 1 : 0);` | src/values/classic-xrfc.ts:773-774 | `value.length` is guaranteed a multiple of 4 by :768, so the division is exact. In Go use integer division and keep the multiple-of-4 guard. |
| `if (decoded.toString("base64") !== value) throw` (re-encode canonicality check) | src/values/classic-xrfc.ts:779-781 | Go's `base64.StdEncoding.DecodeString` is **not** strict about the final padding bits either; the re-encode comparison is the check. Port it literally rather than trusting `StrictEncoding` (which does check padding bits but not all cases the same way). |
| `let byteLength = classicXrfcOpenTagByteLength(parameterName) + classicXrfcCloseTagByteLength(parameterName);` then the buffer is `Buffer.alloc(byteLength)` and the writer asserts `offset !== encoded.byteLength` | src/values/classic-xrfc.ts:616, 661-669 | The encoder is **two-pass**: it computes the exact byte length first, allocates once, then asserts the write matched. Preserve both passes and the assertion — it is the file's core safety property. |
| `byteLength = checkedClassicXrfcLength(...)` uses `name.length`, i.e. UTF-16 code units | src/values/classic-xrfc.ts:176, 180 | Names are ASCII by `SIMPLE_XML_NAME` (:95), so code units equal bytes. Safe in Go with `len(name)`. |
| `Buffer.byteLength(character, "utf8")` per code point in the length pass, and `target.write(character, offset, "utf8")` in the write pass | src/values/classic-xrfc.ts:224-228, 571 | Iteration is by **code point** (`for (const character of value)`), not code unit. In Go, `for _, r := range s` plus `utf8.RuneLen(r)` matches. |
| `const byteLength = Buffer.byteLength(raw, "utf8"); this.#byteOffset += byteLength;` while `this.#offset` advances in UTF-16 units | src/values/classic-xrfc.ts:706-722 | The parser tracks **two** cursors: a UTF-16 character offset for `startsWith`, and a byte offset for limits. In Go there is only one (byte) offset; row-byte accounting collapses to it, but the error messages say "at character N" (:691, :700, :734) and would change. Record the message drift. |

### Go mapping notes

- `ExactClassicXrfcParser` uses `#private` fields (src/values/classic-xrfc.ts:674-677) → unexported struct fields.
- The parser is deliberately a hand-rolled exact-token matcher, **not** an XML parser (no attributes, no prolog, no namespaces, no CDATA: src/values/classic-xrfc.ts:711-713, 1000-1003). Do **not** substitute `encoding/xml` — it accepts far more and would break the "trailing content" and "non-canonical" refusals.
- `PlannedCell`/`PlannedRow` (src/values/classic-xrfc.ts:69-88) are the two-pass plan; keep them unexported value types.
- Sentinels: `ErrXrfcName` (:169), `ErrXrfcUnsupportedType` (:253), `ErrXrfcLimitExceeded` (:471/:522/:618/:637), `ErrXrfcGrammar` (:691/:700/:708/:734), `ErrXrfcNonCanonical` (:771/:780/:798/:821/:829/:838/:889), `ErrXrfcLengthInvariant` (:668) — the last should be a panic-worthy internal invariant, not a returned error, since it indicates an encoder bug.
- `classicXrfcOpenTagByteLength`/`classicXrfcCloseTagByteLength`/`checkedClassicXrfcLength`/`escapedClassicXrfcXmlByteLength`/`writeClassicXrfcOpenTag`/`writeClassicXrfcCloseTag`/`writeEscapedClassicXrfcText`/`ExactClassicXrfcParser`/`decodeClassicXrfcBase64` are exported **only** for reuse by `recursive-classic-xrfc.ts` (src/values/recursive-classic-xrfc.ts:24-39). In Go they should be package-internal, with the recursive codec in the same package.

---

## recursive-classic-xrfc.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RecursiveClassicXrfcLimits` | interface | `export interface RecursiveClassicXrfcLimits extends ClassicXrfcLimits` with `readonly maxNodes?: number;` and `readonly maxDepth?: number;` | src/values/recursive-classic-xrfc.ts:45-50 |
| `RecursiveClassicXrfcParameterIdentity` | interface | `functionName: string; parameterName: string; parameterClass: "I" \| "E" \| "C" \| "T"; associatedType: string; internalType: string` (all `readonly`) | src/values/recursive-classic-xrfc.ts:52-58 |
| `RecursiveClassicXrfcScalarDescriptor` | interface | `kind: "scalar"; name: string; internalType: "I" \| "C" \| "g" \| "y"; internalLength: number` | src/values/recursive-classic-xrfc.ts:60-65 |
| `RecursiveClassicXrfcStructureDescriptor` | interface | `kind: "structure"; name: string; typeName: string; fields: readonly RecursiveClassicXrfcDescriptor[]` | src/values/recursive-classic-xrfc.ts:67-72 |
| `RecursiveClassicXrfcTableDescriptor` | interface | `kind: "table"; name: string; typeName: string; line: RecursiveClassicXrfcStructureDescriptor` | src/values/recursive-classic-xrfc.ts:74-79 |
| `RecursiveClassicXrfcDescriptor` | type | union of the three descriptor interfaces | src/values/recursive-classic-xrfc.ts:81-84 |
| `ResolvedRecursiveClassicXrfcParameter` | interface | `serializer: "classic-xrfc"; functionName; parameterName; parameterClass; kind: ClassicXrfcKind; root; descriptorNodeCount: number; descriptorMaximumDepth: number` | src/values/recursive-classic-xrfc.ts:86-96 |
| `resolveRecursiveClassicXrfcParameter` | function | `export function resolveRecursiveClassicXrfcParameter(graph: RecursiveMetadataGraph, identity: RecursiveClassicXrfcParameterIdentity, limits: RecursiveClassicXrfcLimits = {}): ResolvedRecursiveClassicXrfcParameter` | src/values/recursive-classic-xrfc.ts:877-881 |
| `resolveRecursiveClassicXrfcParameterFromIndex` | function | `export function resolveRecursiveClassicXrfcParameterFromIndex(graph: RecursiveMetadataGraph, index: RecursiveMetadataParameterIndex, identity: RecursiveClassicXrfcParameterIdentity, limits: RecursiveClassicXrfcLimits = {}): ResolvedRecursiveClassicXrfcParameter` | src/values/recursive-classic-xrfc.ts:892-897 |
| `encodeRecursiveClassicXrfcParameter` | function | `export function encodeRecursiveClassicXrfcParameter(resolved: ResolvedRecursiveClassicXrfcParameter, value: unknown, limits: RecursiveClassicXrfcLimits = {}): Buffer` | src/values/recursive-classic-xrfc.ts:1447-1451 |
| `decodeRecursiveClassicXrfcParameter` | function | `export function decodeRecursiveClassicXrfcParameter(resolved: ResolvedRecursiveClassicXrfcParameter, value: Uint8Array, limits: RecursiveClassicXrfcLimits = {}): Readonly<Record<string, unknown>> \| readonly Readonly<Record<string, unknown>>[]` | src/values/recursive-classic-xrfc.ts:1736-1740 |
| `initialRecursiveClassicXrfcValue` | function | `export function initialRecursiveClassicXrfcValue(resolved: ResolvedRecursiveClassicXrfcParameter): Readonly<Record<string, unknown>> \| readonly unknown[]` | src/values/recursive-classic-xrfc.ts:1806-1808 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `SUPPORTED_SCALARS` | `new Set(["I", "C", "g", "y"])` | the strict subset: only four scalar EXIDs | src/values/recursive-classic-xrfc.ts:176 |
| `DEFAULT_MAX_NODES` | `100_000` | | src/values/recursive-classic-xrfc.ts:177 |
| `ABSOLUTE_MAX_NODES` | `1_000_000` | | src/values/recursive-classic-xrfc.ts:178 |
| `DEFAULT_MAX_DEPTH` | `64` | | src/values/recursive-classic-xrfc.ts:179 |
| `ABSOLUTE_MAX_DEPTH` | `256` | | src/values/recursive-classic-xrfc.ts:180 |
| `ITEM_WRAPPER_BYTE_LENGTH` | `13; // <item></item>` | | src/values/recursive-classic-xrfc.ts:181 |
| `EMPTY_STRUCTURE_VALUE` | `Object.freeze({})` | | src/values/recursive-classic-xrfc.ts:183 |
| `EMPTY_TABLE_VALUE` | `Object.freeze([])` | | src/values/recursive-classic-xrfc.ts:184 |
| `LIMIT_KEYS` | `["maxCellBytes", "maxRowBytes", "maxParameterBytes", "maxRows", "maxNodes", "maxDepth"]` | the exact allowed limit keys | src/values/recursive-classic-xrfc.ts:185-192 |
| `IDENTITY_KEYS` | `["functionName", "parameterName", "parameterClass", "associatedType", "internalType"]` | exactly five required identity keys | src/values/recursive-classic-xrfc.ts:193-199 |
| cache names | `"strict-descriptor-template-v1"`, `"strict-descriptor-subtree-v1"` | | src/values/recursive-classic-xrfc.ts:200-201 |
| `DESCRIPTOR_TEMPLATE_PATH` | `"<recursive-xrfc-root>"` | placeholder substituted back into cached error messages | src/values/recursive-classic-xrfc.ts:202, 858 |
| INT4 geometry | `if (field.internalType === "I" && field.ucLength !== 4)` -> "INT4 must occupy four Unicode bytes" | | src/values/recursive-classic-xrfc.ts:542-544 |
| CHAR capacity | `const capacity = descriptor.internalLength / 2;` | | src/values/recursive-classic-xrfc.ts:1120, 1519, 1567 |
| base64 encoded size | `const contentByteLength = Math.ceil(byteLength / 3) * 4;` | | src/values/recursive-classic-xrfc.ts:1144 |
| base64 decode budget | `Math.floor(limits.maxCellBytes / 4) * 3` | decoded-byte ceiling derived from the encoded cell limit | src/values/recursive-classic-xrfc.ts:1532, 1593 |
| initial scalars | `"I"` to `0`; `"C"`/`"g"` to `""`; `"y"` to `Buffer.alloc(0)` | | src/values/recursive-classic-xrfc.ts:1090-1102 |
| array index grammar | `/^(?:0\|[1-9][0-9]*)$/u` plus `Number(key) >= value.length` | dense-array key validation | src/values/recursive-classic-xrfc.ts:1339 |

### Errors

Representative, verbatim:

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `` `${path} must not be a proxy` `` (TypeError) | proxy anywhere in the value or limits graph | src/values/recursive-classic-xrfc.ts:214, 243, 1327 |
| `` `${path} must not contain symbol properties` `` (TypeError) | | src/values/recursive-classic-xrfc.ts:224 |
| `` `${path} must use Object.prototype or a null prototype` `` (TypeError) | | src/values/recursive-classic-xrfc.ts:247 |
| `` `${path} must be an own data property` `` (TypeError) | accessor-backed property | src/values/recursive-classic-xrfc.ts:263, 284 |
| `` `${path} contains unknown property ${key}` `` (TypeError) | limits or identity has an extra key | src/values/recursive-classic-xrfc.ts:278 |
| `` `${label} must be an integer in 0..${maximum}` `` (RangeError) | | src/values/recursive-classic-xrfc.ts:312 |
| `"recursive xRFC parameter name must be a string"`, `"recursive xRFC function name must be non-empty"`, `"recursive xRFC parameter class must be I, E, C, or T"`, `"recursive xRFC associated type must be a string"`, `"recursive xRFC internal type must contain one character"` (all TypeError) | identity validation | src/values/recursive-classic-xrfc.ts:394, 398, 401, 404, 407 |
| `"recursive xRFC metadata graph must be a normalized recursive metadata graph"` (TypeError) | | src/values/recursive-classic-xrfc.ts:424-426 |
| `"recursive xRFC metadata nodes must be immutable"` (TypeError) | | src/values/recursive-classic-xrfc.ts:429 |
| `` `recursive xRFC metadata identity does not match function ${identity.functionName}` `` | | src/values/recursive-classic-xrfc.ts:436 |
| `` `${identity.functionName}.${identity.parameterName} recursive metadata contains ${matches.length} matching parameters` `` | | src/values/recursive-classic-xrfc.ts:449-451 |
| `` `${identity.functionName}.${identity.parameterName} recursive descriptor does not match flat metadata` `` | | src/values/recursive-classic-xrfc.ts:473 |
| `` `${path} descriptor depth exceeds ${budget.limits.maxDepth}` `` (RangeError) | | src/values/recursive-classic-xrfc.ts:490, 742 |
| `` `${path} descriptor node count exceeds ${budget.limits.maxNodes}` `` (RangeError) | | src/values/recursive-classic-xrfc.ts:495, 747 |
| `` `${path} contains inconsistent scalar metadata` `` | field reference kind or type disagreement | src/values/recursive-classic-xrfc.ts:532 |
| `` `${path} type ${field.internalType} is not implemented for the proven recursive xRFC subset` `` | outside `SUPPORTED_SCALARS` | src/values/recursive-classic-xrfc.ts:536 |
| `` `${path} contains invalid Unicode geometry` `` | `ucLength` not a non-negative safe integer | src/values/recursive-classic-xrfc.ts:540 |
| `` `${path} Unicode character width must be even` `` | odd `ucLength` on a `C` field | src/values/recursive-classic-xrfc.ts:546 |
| `` `${path} contains a cyclic recursive reference` `` and `` `${path} contains a cyclic recursive type ${typeName}` `` | | src/values/recursive-classic-xrfc.ts:569, 688, 779 |
| `` `${path} table ${node.name} must contain one line descriptor` `` and `` `${path} table ${node.name} requires one non-cyclic structured line` `` | | src/values/recursive-classic-xrfc.ts:644, 653-655 |
| `` `${path} value depth exceeds ${budget.limits.maxDepth}` `` and `` `${path} value node count exceeds ${budget.limits.maxNodes}` `` (RangeError) | | src/values/recursive-classic-xrfc.ts:1039, 1043-1045 |
| `` `${path} xRFC XML exceeds ${limits.maxParameterBytes} bytes` `` (RangeError) | | src/values/recursive-classic-xrfc.ts:1055-1057, 1072-1074, 1478 |
| `` `${path} expects a signed 32-bit integer` `` (RangeError) | | src/values/recursive-classic-xrfc.ts:1085 |
| `` `${path} base64 value exceeds ${budget.limits.maxCellBytes} encoded bytes` `` (RangeError) | | src/values/recursive-classic-xrfc.ts:1149-1151 |
| `` `${path} row count exceeds ${budget.limits.maxRows}` `` (RangeError) | | src/values/recursive-classic-xrfc.ts:1334, 1641, 1709 |
| `` `${path} contains unknown array property ${key}` `` (TypeError) | non-index own key on a row array | src/values/recursive-classic-xrfc.ts:1340 |
| `` `${trusted.parameterName} recursive xRFC preflight length invariant failed` `` | planned bytes differ from the computed tagged length | src/values/recursive-classic-xrfc.ts:1482-1484 |
| `` `${trusted.parameterName} recursive xRFC encoder length invariant failed` `` | written offset differs from the length | src/values/recursive-classic-xrfc.ts:1491 |
| `"recursive xRFC plan must be returned by resolveRecursiveClassicXrfcParameter"` (TypeError) | plan not in the trusted `WeakSet` | src/values/recursive-classic-xrfc.ts:1026-1028 |
| `` `${path} contains non-canonical base64` `` | grammar failure or non-zero padding bits | src/values/recursive-classic-xrfc.ts:1581, 1589 |
| `` `${path} decoded bytes exceed its configured limit` `` (RangeError) | | src/values/recursive-classic-xrfc.ts:1594 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Maximum descriptor or runtime value nodes visited for one parameter." | src/values/recursive-classic-xrfc.ts:46 |
| "Maximum descriptor or runtime container depth for one parameter." | src/values/recursive-classic-xrfc.ts:48 |
| "Resolve one independently proven classic/xRFC recursive parameter plan. basXML and fast serialization are deliberately not selected by this API." | src/values/recursive-classic-xrfc.ts:873-876 |
| "Internal indexed resolver used by one captured invocation dispatch." | src/values/recursive-classic-xrfc.ts:891 |
| "Prefer the independently qualified strict codec. When that older subset rejects a supported scalar, require the broader codec to resolve and validate the complete reachable graph before the same graph can obtain a send decision. The adapter supplies no wire geometry: that remains owned by the normalized recursive metadata graph." | src/values/recursive-serializer-classification.ts:216-219 |
| "Encode one bounded recursive parameter using only the classic xRFC path." | src/values/recursive-classic-xrfc.ts:1446 |
| "Decode the strict, attribute-free recursive classic xRFC subset." | src/values/recursive-classic-xrfc.ts:1735 |
| "Return a fresh ABAP-initial JavaScript value for a resolved deep parameter." | src/values/recursive-classic-xrfc.ts:1805 |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| `resolved.serializer === "classic-xrfc"`, `resolved.kind === "structure"`, and the nested value encodes to `EXPECTED_XML` and decodes back to `VALUE` | `"encodes and decodes the bounded recursive classic xRFC subset exactly"` | test/recursive-classic-xrfc.test.ts:597-612 |
| the ABAP-initial value is `{ ID: 0, CHILD: { TEXT: "", LABEL: "" }, ROWS: [], BLOB: Buffer.alloc(0) }` and encodes to `<INPUT><ID>0</ID><CHILD><TEXT></TEXT><LABEL></LABEL></CHILD><ROWS></ROWS><BLOB></BLOB></INPUT>` | `"constructs recursive ABAP initial values without choosing basXML"` | test/recursive-classic-xrfc.test.ts:614-629 |
| a 3-byte XSTRING fits `maxCellBytes: 4`; a 4-byte one throws `/base64 value exceeds 4 encoded bytes/`; decoding `AQIDBA==` throws | `"uses encoded base64 cell limits symmetrically"` | test/recursive-classic-xrfc.test.ts:1388-1430 |
| a field named `__proto__` becomes an own data property | `"constructs __proto__ fields as own data without prototype mutation"` | test/recursive-classic-xrfc.test.ts:1338 |
| descriptor depth/nodes and aggregate value rows/nodes/bytes are all bounded | `"bounds descriptor depth/nodes and aggregate value rows/nodes/bytes"` | test/recursive-classic-xrfc.test.ts:985 |
| aggregate XML bytes are reserved before later XSTRING values are copied | `"reserves aggregate XML bytes before copying later XSTRING values"` | test/recursive-classic-xrfc.test.ts:1059 |
| a later non-canonical base64 cell is rejected before earlier XSTRING values are allocated | `"rejects non-canonical later base64 before allocating earlier XSTRING values"` | test/recursive-classic-xrfc.test.ts:1092 |
| proxy, symbol, prototype, non-enumerable, and exotic array inputs are refused | `"rejects proxy, symbol, prototype, non-enumerable, and exotic array inputs"` | test/recursive-classic-xrfc.test.ts:1183 |

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `if (!Number.isSafeInteger(normalized) \|\| normalized < 0 \|\| normalized > maximum)` | src/values/recursive-classic-xrfc.ts:308-313 | Same limit-normalization pattern as classic-xrfc; port as a bounded `int`. |
| `if (!Number.isSafeInteger(field.ucLength) \|\| field.ucLength < 0)` | src/values/recursive-classic-xrfc.ts:539 | Metadata geometry is untrusted; keep the negative check. |
| `if (field.internalType === "C" && (field.ucLength & 1) !== 0)` | src/values/recursive-classic-xrfc.ts:545 | JS `&` coerces to int32; use `%2` on a range-checked `int`. |
| `const nodes = budget.nodes + template.descriptorNodeCount; if (!Number.isSafeInteger(nodes) \|\| nodes > budget.limits.maxNodes)` | src/values/recursive-classic-xrfc.ts:745-750 | Cached-subtree node accounting. Overflow guard: in Go add an explicit `nodes < budget.nodes` wraparound test. |
| `typeof value !== "number" \|\| !Number.isSafeInteger(value) \|\| value < -0x8000_0000 \|\| value > 0x7fff_ffff` then `String(value)` | src/values/recursive-classic-xrfc.ts:1079-1087 | `int32` plus `strconv.Itoa`. |
| `const contentByteLength = Math.ceil(byteLength / 3) * 4; if (!Number.isSafeInteger(contentByteLength) \|\| contentByteLength > budget.limits.maxCellBytes)` | src/values/recursive-classic-xrfc.ts:1144-1152 | `((n+2)/3)*4` in integer arithmetic with an overflow check. |
| `const decoded = Number(value); if (!Number.isSafeInteger(decoded) \|\| decoded < -0x8000_0000 \|\| decoded > 0x7fff_ffff)` in both `decodeScalar` and `preflightScalar` | src/values/recursive-classic-xrfc.ts:1507-1514, 1555-1562 | `strconv.ParseInt(value, 10, 32)`, twice: decode runs a preflight pass and then a real pass over the same text (src/values/recursive-classic-xrfc.ts:1767-1801). |
| `function base64Sextet(characterCode: number): number { if (characterCode >= 0x41 && characterCode <= 0x5a) return characterCode - 0x41; ... return characterCode === 0x2b ? 62 : 63; }` | src/values/recursive-classic-xrfc.ts:1537-1542 | Pure byte arithmetic, direct port. The trailing `: 63` is an unchecked fallthrough that assumes the base64 alphabet regex at src/values/recursive-classic-xrfc.ts:1579 already ran. |
| `(value.endsWith("==") && (base64Sextet(value.charCodeAt(value.length - 3)) & 0x0f) !== 0) \|\| (value.endsWith("=") && !value.endsWith("==") && (base64Sextet(value.charCodeAt(value.length - 2)) & 0x03) !== 0)` | src/values/recursive-classic-xrfc.ts:1583-1590 | Explicit base64 padding-bit canonicality check: with `==` the last data sextet must have its low 4 bits zero, with `=` its low 2 bits. Go's `base64.StdEncoding` does not enforce this. Port the explicit check rather than relying on `base64.StrictEncoding`, so the error message stays stable. |
| `const decodedByteLength = (value.length / 4) * 3 - (value.endsWith("==") ? 2 : value.endsWith("=") ? 1 : 0); if (decodedByteLength > Math.floor(limits.maxCellBytes / 4) * 3)` | src/values/recursive-classic-xrfc.ts:1591-1595 | Integer division; the budget must be floor-then-multiply (`maxCellBytes/4*3`), not `maxCellBytes*3/4`. |
| `if (!/^(?:0\|[1-9][0-9]*)$/u.test(key) \|\| Number(key) >= value.length)` | src/values/recursive-classic-xrfc.ts:1339 | JS array own-key validation; no Go analogue for a slice. Drop, but keep the "dense, no extra keys" intent if the Go input is a map. |
| `const remainingRows = budget.limits.maxRows - budget.rows; if (value.length > remainingRows)` | src/values/recursive-classic-xrfc.ts:1332-1335 | Subtracting before comparing avoids an overflowing sum. Keep this form. |

### Go mapping notes

- `RESOLVED_PARAMETERS = new WeakSet<object>()` (src/values/recursive-classic-xrfc.ts:182, 1014, 1024) enforces "this plan came from our resolver". In Go make `ResolvedParameter` a struct with unexported fields in the codec package, so it cannot be constructed elsewhere.
- The descriptor-template cache (src/values/recursive-classic-xrfc.ts:781-824, 956-1003) memoizes successes and failures alike (`CachedDescriptorTemplateFailure`, src/values/recursive-classic-xrfc.ts:120-124) and replays a failure by reconstructing an error of the recorded `errorName` with `DESCRIPTOR_TEMPLATE_PATH` substituted (src/values/recursive-classic-xrfc.ts:854-862). In Go, cache a wrapped sentinel and substitute the path at replay so `errors.Is` still works.
- `plainDataRecord` / `ownDataValue` / `exactDataRecord` (src/values/recursive-classic-xrfc.ts:239-298) are JS object-hardening with no Go analogue. Delete them; the tests at test/recursive-classic-xrfc.test.ts:1183 and 1306 have no Go port.
- `bindDescriptorRootName` returns `Object.freeze({ ...root, name })` (src/values/recursive-classic-xrfc.ts:864-871), a shallow copy with one field replaced: a direct Go struct copy.
- Sentinels: `ErrRecursivePlanUntrusted` (:1026), `ErrRecursiveDescriptorDepth` and `ErrRecursiveDescriptorNodes` (:490, :495), `ErrRecursiveCyclic` (:569, :688, :779), `ErrRecursiveUnsupportedScalar` (:536), `ErrRecursiveLimit` (:1039, :1055, :1334). The two internal-invariant failures at :1482 and :1491 should panic, not return errors.

---

## recursive-xrfc.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RecursiveXrfcKind` | type | `export type RecursiveXrfcKind = "structure" \| "table";` | src/values/recursive-xrfc.ts:63 |
| `RecursiveXrfcLimits` | interface | `maxDepth?`, `maxNodes?`, `maxRows?`, `maxCells?`, `maxCellBytes?`, `maxParameterBytes?` (all `readonly ...?: number`) | src/values/recursive-xrfc.ts:65-73 |
| `RecursiveXrfcOptions` | interface | `export interface RecursiveXrfcOptions extends RecursiveXrfcLimits` with `readonly int8Mode?: ClassicInt8Mode;` and `readonly bcd?: ClassicBcdMode;` | src/values/recursive-xrfc.ts:75-78 |
| `ResolvedRecursiveXrfcParameter` | interface | `readonly parameter: RecursiveMetadataParameter;` `readonly kind: RecursiveXrfcKind;` `readonly node: RecursiveMetadataTypeNode;` | src/values/recursive-xrfc.ts:89-93 |
| `resolveRecursiveXrfcParameter` | function | `export function resolveRecursiveXrfcParameter(graph: RecursiveMetadataGraph \| undefined, parameter: RfcFunintParameter): ResolvedRecursiveXrfcParameter \| undefined` | src/values/recursive-xrfc.ts:489-492 |
| `resolveRecursiveXrfcParameterFromIndex` | function | `export function resolveRecursiveXrfcParameterFromIndex(graph: RecursiveMetadataGraph, index: RecursiveMetadataParameterIndex, parameter: RfcFunintParameter): ResolvedRecursiveXrfcParameter \| undefined` | src/values/recursive-xrfc.ts:516-520 |
| `validateRecursiveXrfcParameter` | function | `export function validateRecursiveXrfcParameter(graph: RecursiveMetadataGraph, parameter: RfcFunintParameter, options: Pick<RecursiveXrfcLimits, "maxDepth"> = {}): ResolvedRecursiveXrfcParameter` | src/values/recursive-xrfc.ts:773-777 |
| `validateRecursiveXrfcParameterFromIndex` | function | `export function validateRecursiveXrfcParameterFromIndex(graph: RecursiveMetadataGraph, index: RecursiveMetadataParameterIndex, parameter: RfcFunintParameter, options: Pick<RecursiveXrfcLimits, "maxDepth"> = {}): ResolvedRecursiveXrfcParameter` | src/values/recursive-xrfc.ts:786-791 |
| `escapeRecursiveXrfcTag` | function | `export function escapeRecursiveXrfcTag(name: string): string` | src/values/recursive-xrfc.ts:837 |
| `decodeRecursiveXrfcParameterName` | function | `export function decodeRecursiveXrfcParameterName(value: Uint8Array, limits: Pick<RecursiveXrfcLimits, "maxParameterBytes"> = {}): string` | src/values/recursive-xrfc.ts:892-895 |
| `encodeRecursiveXrfcParameter` | function | `export function encodeRecursiveXrfcParameter(parameter: RfcFunintParameter, graph: RecursiveMetadataGraph, value: unknown, options: RecursiveXrfcOptions = {}): Buffer` | src/values/recursive-xrfc.ts:1327-1332 |
| `encodeResolvedRecursiveXrfcParameter` | function | `export function encodeResolvedRecursiveXrfcParameter(parameter: RfcFunintParameter, graph: RecursiveMetadataGraph, resolved: ResolvedRecursiveXrfcParameter, value: unknown, options: RecursiveXrfcOptions = {}): Buffer` | src/values/recursive-xrfc.ts:1352-1358 |
| `decodeRecursiveXrfcParameter` | function | `export function decodeRecursiveXrfcParameter(parameter: RfcFunintParameter, graph: RecursiveMetadataGraph, value: Uint8Array, options: RecursiveXrfcOptions = {}): unknown` | src/values/recursive-xrfc.ts:1859-1864 |
| `decodeResolvedRecursiveXrfcParameter` | function | `export function decodeResolvedRecursiveXrfcParameter(parameter: RfcFunintParameter, graph: RecursiveMetadataGraph, resolved: ResolvedRecursiveXrfcParameter, value: Uint8Array, options: RecursiveXrfcOptions = {}): unknown` | src/values/recursive-xrfc.ts:1888-1894 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `ABSOLUTE_GRAPH_MAX_NODES` | `20_000` | | src/values/recursive-xrfc.ts:125 |
| `ABSOLUTE_GRAPH_MAX_ROWS` | `100_000` | | src/values/recursive-xrfc.ts:126 |
| `ABSOLUTE_GRAPH_MAX_EDGES` | `100_000` | | src/values/recursive-xrfc.ts:127 |
| `DEFAULT_RUNTIME_MAX_NODES` | `100_000` | | src/values/recursive-xrfc.ts:128 |
| `ABSOLUTE_RUNTIME_MAX_NODES` | `1_000_000` | | src/values/recursive-xrfc.ts:129 |
| `maxDepth` default and ceiling | `bounded(options.maxDepth, 64, 256, "maxDepth")` | | src/values/recursive-xrfc.ts:169, 781, 792 |
| `maxRows` / `maxCells` ceiling | `0xffff_ffff` | | src/values/recursive-xrfc.ts:180, 186 |
| `CANONICAL_INTEGER` | `/^(?:0\|-[1-9][0-9]*\|[1-9][0-9]*)$/u` | | src/values/recursive-xrfc.ts:138 |
| `FINITE_FLOAT_LEXICAL` | `/^[+-]?(?:(?:[0-9]+(?:\.[0-9]*)?)\|(?:\.[0-9]+))(?:[eE][+-]?[0-9]+)?$/u` | | src/values/recursive-xrfc.ts:139 |
| `CANONICAL_ENTITY_CODE_POINTS` | `new Set([0, 1, 2, 3, 4, 5, 6, 7, 8, 11, 12, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 38, 60, 62])` | the code points the writer escapes as numeric references. 9, 10 and 13 (tab, LF, CR) are deliberately absent and are written raw. | src/values/recursive-xrfc.ts:140-144 |
| `SUPPORTED_SCALAR_TYPES` | `new Set(["C", "N", "D", "T", "X", "P", "F", "I", "b", "s", "8", "a", "e", "p", "n", "w", "d", "7", "x", "t", "i", "c", "g", "y"])` | twenty-four scalar EXIDs: the broad codec | src/values/recursive-xrfc.ts:145-149 |
| entity spelling written | `` chunk = `&#${String(codePoint).padStart(2, "0")};` `` | U+0000 becomes `&#00;`, U+0001 becomes `&#01;`, `&` becomes `&#38;`, `<` becomes `&#60;`, `>` becomes `&#62;` | src/values/recursive-xrfc.ts:955, 967-969 |
| non-character refusal | `if (codePoint === 0xfffe \|\| codePoint === 0xffff)` | refused in raw and in reference position | src/values/recursive-xrfc.ts:950, 1419, 1440 |
| xRFC-eligible parameter EXIDs | `parameter.exid !== "u" && parameter.exid !== "v" && parameter.exid !== "h"` returns `undefined` | only `u`, `v`, `h` can be recursive | src/values/recursive-xrfc.ts:495, 522-526 |
| classic TABLES carve-out | `if (parameter.parameterClass === "T" && parameter.exid === "u") { return undefined; }` | classic structured TABLES rows stay on the binary codec | src/values/recursive-xrfc.ts:503-505, 529-531 |
| forced-xRFC EXIDs | `const required = parameter.exid === "h" \|\| parameter.exid === "v" \|\| nodeRequiresXrfc(...)` | | src/values/recursive-xrfc.ts:578-584 |
| tag escape for `/` | `else if (character === "/") result += "_-";` | | src/values/recursive-xrfc.ts:849 |
| tag escape for other code points at or below 0xFF | `` else if (codePoint <= 0xff) result += `_--${codePoint.toString(16).toUpperCase().padStart(2, "0")}`; `` | | src/values/recursive-xrfc.ts:850-851 |
| tag first-character rule | `/[A-Za-z_]/u.test(character) \|\| codePoint > 0xff` | | src/values/recursive-xrfc.ts:847 |
| tag later-character rule | `/[A-Za-z0-9_]/u.test(character) \|\| codePoint > 0xff` | | src/values/recursive-xrfc.ts:848 |
| root tag bound | `if (end < 2 \|\| end > 256 \|\| text.slice(1, end).includes("<"))` | | src/values/recursive-xrfc.ts:927 |
| INT4 / INT2 / INT1 encode domains | `integer(value, -0x8000_0000, 0x7fff_ffff, path)`, `integer(value, -0x8000, 0x7fff, path)`, `integer(value, 0, 0xff, path)` | | src/values/recursive-xrfc.ts:1104-1106 |
| INT4 / INT2 / INT1 decode domains | `-0x8000_0000n .. 0x7fff_ffffn`, `-0x8000n .. 0x7fffn`, `0n .. 0xffn` | compared as bigint on decode | src/values/recursive-xrfc.ts:1682-1696 |
| compact-temporal raw bound on decode | `if (raw < 0n \|\| raw >= (1n << BigInt(width * 8)))` | unsigned width-bounded raw | src/values/recursive-xrfc.ts:1579 |
| initial scalars | `"C"`/`"N"`/`"g"` to `""`; `"D"` to `"00000000"`; `"T"` to `"000000"`; `"X"` and `"y"` to `Buffer.alloc(0)`; `"P"` to `"0"`; `"F"`/`"I"`/`"s"`/`"b"` to `0`; `"8"` to `classicInt8InitialValue(int8Mode)`; `"a"`/`"e"` to `"0"`; temporal EXIDs to `classicTemporalInitialValue` | | src/values/recursive-xrfc.ts:1129-1156 |

### Errors

Representative, verbatim:

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `` `${label} must be an integer in 0..${maximum}` `` (RangeError) | | src/values/recursive-xrfc.ts:159 |
| `"recursive xRFC options must be an object"` (TypeError) | | src/values/recursive-xrfc.ts:165 |
| `` `recursive xRFC graph ${label} is outside 0..${maximum}` `` (RangeError) | declared graph limit out of range | src/values/recursive-xrfc.ts:209 |
| `"recursive xRFC graph must be a version-1 metadata graph"` (TypeError) | `graph.version !== 1` | src/values/recursive-xrfc.ts:216 |
| `"recursive xRFC graph lacks bounded metadata limits"` (TypeError) | | src/values/recursive-xrfc.ts:220 |
| `` `recursive xRFC graph exceeds its node budget ${maxNodes}` `` and `` `recursive xRFC graph exceeds its row budget ${maxRows}` `` (RangeError) | | src/values/recursive-xrfc.ts:245, 248 |
| `` `${path} exceeds recursive xRFC graph node budget ${budget.maxNodes}` ``, `` ... row budget ... ``, `` ... edge budget ... `` (RangeError) | | src/values/recursive-xrfc.ts:260, 267, 277 |
| `` `${path} requires recursive ${kind} node ${name}` `` | | src/values/recursive-xrfc.ts:289 |
| `` `${path} recursive ${kind} node identity ${node.name} disagrees with map key ${name}` `` | graph map alias | src/values/recursive-xrfc.ts:292-294 |
| `` `${path} contains a cyclic recursive RFC type` `` | | src/values/recursive-xrfc.ts:306, 612 |
| `"recursive xRFC metadata lacks a function identity"` | | src/values/recursive-xrfc.ts:316, 353 |
| `` `${identity.name}.${parameter.parameterName} has duplicate recursive metadata` `` | | src/values/recursive-xrfc.ts:322-324 |
| `` `${identity.name}.${parameter.parameterName} recursive metadata disagrees with the function interface` `` | | src/values/recursive-xrfc.ts:363 |
| `` `${identity.name}.${parameter.parameterName} recursive type identity disagrees with the function interface` `` | | src/values/recursive-xrfc.ts:371 |
| `"recursive xRFC plan must be resolved for the same graph and parameter"` (TypeError) | | src/values/recursive-xrfc.ts:405-407 |
| `` `${parameter.parameterName} lacks its recursive table descriptor` `` | `h` parameter without a table reference | src/values/recursive-xrfc.ts:556 |
| `` `${parameter.parameterName} requires recursive table row node ${recursive.reference.targetType}` `` | | src/values/recursive-xrfc.ts:573-575 |
| `` `${path} exceeds recursive xRFC depth ${maximumDepth}` `` (RangeError) | | src/values/recursive-xrfc.ts:602, 607, 621-623, 811, 1751, 1770, 1835 |
| `` `${path} scalar type has an invalid anonymous descriptor` ``, `` `${path} table type has an invalid line descriptor` ``, `` `${path} structure contains an anonymous field` `` | | src/values/recursive-xrfc.ts:637, 643, 646, 1253, 1775 |
| `` `${path} contains duplicate field ${field.name}` `` | | src/values/recursive-xrfc.ts:658 |
| `` `${fieldPath} xRFC scalar type ${field.internalType} is not implemented` `` | | src/values/recursive-xrfc.ts:667, 1051, 1725 |
| `` `${fieldPath} Unicode character width must be even` `` | odd `ucLength` on `C`, `N`, `D` or `T` | src/values/recursive-xrfc.ts:670, 1009 |
| `` `${fieldPath} scalar node contains a container reference` `` | | src/values/recursive-xrfc.ts:675 |
| `` `${fieldPath} contains inconsistent structure metadata` `` and `` `${fieldPath} contains inconsistent table metadata` `` | | src/values/recursive-xrfc.ts:682, 685 |
| `` `${parameter.parameterName} does not require recursive xRFC` `` | | src/values/recursive-xrfc.ts:727, 799 |
| `` `${path} recursive xRFC XML exceeds ${state.limits.maxParameterBytes} bytes` `` (RangeError) | | src/values/recursive-xrfc.ts:828-830 |
| `"xRFC tag name must be a non-empty string"` (TypeError) | | src/values/recursive-xrfc.ts:839 |
| `"xRFC tag name contains an unsupported character"` | a code point above 0xFF that is not a letter, digit or underscore | src/values/recursive-xrfc.ts:852 |
| `"xRFC XML parameter contains an invalid tag escape"` and `"xRFC XML parameter contains a non-canonical tag escape"` | | src/values/recursive-xrfc.ts:879, 886 |
| `"recursive xRFC XML must be Uint8Array bytes"` (TypeError) | | src/values/recursive-xrfc.ts:897, 1866, 1896 |
| `` `recursive xRFC XML must contain 1..${maximum} bytes` `` (RangeError) | | src/values/recursive-xrfc.ts:907 |
| `"recursive xRFC XML must not contain a UTF-8 BOM"` | | src/values/recursive-xrfc.ts:920 |
| `"recursive xRFC XML lacks its top-level tag"` and `"recursive xRFC XML lacks a supported top-level tag"` | | src/values/recursive-xrfc.ts:924, 928 |
| `` `${path} contains an unsupported non-character` `` (RangeError) | U+FFFE or U+FFFF in writer input | src/values/recursive-xrfc.ts:951 |
| `` `${path} XML value exceeds ${maximumBytes} bytes` `` (RangeError) | | src/values/recursive-xrfc.ts:960, 1067, 1089, 1120 |
| `` `${path} xRFC text encoding length changed` `` | two-pass length mismatch inside `escapedText` | src/values/recursive-xrfc.ts:973 |
| `` `${path} byte length is unsafe` `` and `` `${path} base64 length is unsafe` `` (RangeError) | | src/values/recursive-xrfc.ts:980, 985 |
| `` `${path} expects an integer in ${minimum}..${maximum}` `` (RangeError) | | src/values/recursive-xrfc.ts:1002 |
| `` `${path} accepts at most ${length} bytes` `` (RangeError) | fixed `X` overflow, encode and decode | src/values/recursive-xrfc.ts:1020, 1656 |
| `` `${path} is not a compact temporal value` `` | | src/values/recursive-xrfc.ts:1033, 1575 |
| `` `${path} does not fit ${field.internalType}(${capacity})` `` (RangeError) | | src/values/recursive-xrfc.ts:1060 |
| `` `${path} expects at most ${capacity} decimal digits` `` (TypeError) | | src/values/recursive-xrfc.ts:1064 |
| `` `${path} expects a finite number` `` (TypeError) | | src/values/recursive-xrfc.ts:1101 |
| `` `${path} structure must not be a proxy` ``, `` `${path} structure must use Object.prototype or a null prototype` ``, `` `${path} structure must not contain symbol properties` ``, `` `${path}.${key} must be an own data property` `` (TypeError) | | src/values/recursive-xrfc.ts:1168, 1172, 1178, 1182 |
| `` `${path} exceeds recursive xRFC cell count ${state.limits.maxCells}` `` (RangeError) | | src/values/recursive-xrfc.ts:1204, 1488 |
| `` `${path} contains unknown field ${key}` `` | | src/values/recursive-xrfc.ts:1249 |
| `` `${path} scalar table row must not be a proxy` `` and `` `${path} scalar table wrapper must contain only the empty-name field` `` (TypeError) | | src/values/recursive-xrfc.ts:1285, 1294 |
| `` `${path} expects an array of rows` `` (TypeError) | | src/values/recursive-xrfc.ts:1313 |
| `` `${path} exceeds recursive xRFC row count ${state.limits.maxRows}` `` (RangeError) | | src/values/recursive-xrfc.ts:1317, 1512 |
| `"recursive xRFC graph lacks its function identity"` | | src/values/recursive-xrfc.ts:1387 |
| `` `${path} contains non-canonical raw xRFC text` `` | the `]]>` sequence, U+FFFE/U+FFFF, or a raw C0 control other than 9, 10, 13 | src/values/recursive-xrfc.ts:1414, 1423 |
| `` `${path} contains an out-of-range XML entity` `` | escaped U+FFFE or U+FFFF | src/values/recursive-xrfc.ts:1441 |
| `` `${path} expected ${token} at character ${this.#offset}` `` | | src/values/recursive-xrfc.ts:1472, 1480 |
| `` `${path} recursive xRFC XML is truncated` `` | | src/values/recursive-xrfc.ts:1491 |
| `` `recursive xRFC XML has trailing content at character ${this.#offset}` `` | | src/values/recursive-xrfc.ts:1531 |
| `` `${path} contains non-canonical base64` `` | | src/values/recursive-xrfc.ts:1541, 1556 |
| `` `${path} contains a non-canonical integer` `` | fails `CANONICAL_INTEGER` or longer than 20 characters | src/values/recursive-xrfc.ts:1564 |
| `` `${path} compact temporal raw value is out of range` `` (RangeError) | | src/values/recursive-xrfc.ts:1580 |
| `` `${path} decoded value exceeds the ${limits.maxCellBytes}-byte cell limit` `` and `` `${path} decoded value exceeds the ${limits.maxParameterBytes}-byte parameter limit` `` (RangeError) | | src/values/recursive-xrfc.ts:1596-1603 |
| `` `${path} decoded output exceeds the ${this.#limits.maxParameterBytes}-byte parameter limit` `` (RangeError) | | src/values/recursive-xrfc.ts:1522-1524 |
| `` `${path} decoded output byte length is unsafe` `` (RangeError) | | src/values/recursive-xrfc.ts:1519 |
| `` `${path} exceeds ${capacity} characters` `` (RangeError) | | src/values/recursive-xrfc.ts:1624 |
| `` `${path} contains a non-decimal NUM value` `` | | src/values/recursive-xrfc.ts:1627 |
| `` `${path} contains a non-canonical xRFC DATE` `` and `` `${path} contains a non-canonical xRFC TIME` `` | | src/values/recursive-xrfc.ts:1636, 1645 |
| `` `${path} contains an invalid FLOAT` `` | | src/values/recursive-xrfc.ts:1674, 1678 |
| `` `${path} INT4 is out of range` ``, `` `${path} INT2 is out of range` ``, `` `${path} INT1 is out of range` `` (RangeError) | | src/values/recursive-xrfc.ts:1684, 1689, 1694 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Maximum runtime structure/table containers instantiated for one value." | src/values/recursive-xrfc.ts:67 |
| "Resolve a function parameter only when the normalized graph requires xRFC. Flat fixed structures and classic TABLES rows remain on the binary codec." | src/values/recursive-xrfc.ts:485-488 |
| "Classic structured TABLES rows (`T` + `u`) stay on the RFC table codec. Explicit xRFC table descriptors (`h`) still require xRFC even when their direction is TABLES; otherwise broader scalar-line tables are unreachable." | src/values/recursive-xrfc.ts:500-502 |
| "Any immediate container edge makes a classic `u` value recursive. Check the target identity now, while the complete bounded validator owns the deeper walk. This avoids an unbounded resolver recursion on hostile hand graphs before maxDepth can be enforced." | src/values/recursive-xrfc.ts:457-459 |
| "Validate the complete reachable serializer graph without reading a value." | src/values/recursive-xrfc.ts:772 |
| "Resolve and fully validate through one invocation-scoped parameter index." | src/values/recursive-xrfc.ts:785 |
| "Escape the reversible tag grammar used by xRFC for ABAP namespace names." | src/values/recursive-xrfc.ts:836 |
| "Read the canonical root parameter name, including escaped ABAP namespaces." | src/values/recursive-xrfc.ts:891 |
| "fixedBytes performs the bounded zero-padding after base64 preflight." | src/values/recursive-xrfc.ts:1142 |
| "XML forbids \">\" in character data only as the \"]]>\" sequence, so a conforming producer may send it bare. Everything else refused here is a character no conforming producer can put in character data at all." | src/values/recursive-xrfc.ts:1410-1412 |
| "The recursive canonical form transports C0 controls as references by design, so those stay admissible in reference position even though they are refused raw. The two non-characters our writer refuses outright do not become admissible by being escaped." | src/values/recursive-xrfc.ts:1436-1439 |
| "Encode one graph-backed recursive xRFC parameter with bounded allocations." | src/values/recursive-xrfc.ts:1326 |
| "Encode from an invocation-scoped plan without rescanning graph parameters." | src/values/recursive-xrfc.ts:1351 |
| "Strictly decode one complete graph-backed recursive xRFC parameter." | src/values/recursive-xrfc.ts:1858 |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| exact nested XML with numeric escapes, `<item>` wrappers and base64 payloads (see conformance section) | `"encodes and decodes nested structures, tables, and XSTRING exactly"` | test/recursive-xrfc.test.ts:275-301 |
| `escapeRecursiveXrfcTag("/NS/TEXTS") === "_-NS_-TEXTS"`; anonymous scalar table rows encode as `<_-NS_-TEXTS><item>ONE</item><item>TWO</item></_-NS_-TEXTS>`; the name round-trips through `decodeRecursiveXrfcParameterName` | `"supports anonymous scalar table rows and namespace tag escaping"` | test/recursive-xrfc.test.ts:531-558 |
| `P`, `a` and `e` values project through `bcd: "number"` to `12.34`, `56.78`, `-9.5`, `10.125`; a custom converter receives exactly `["12.34", "56.78", "-9.5", "10.125"]` | `"projects BCD values through recursive xRFC structures and tables"` | test/recursive-xrfc.test.ts:867-952 |
| negative zero in a FLOAT scalar table encodes as `<FLOATS><item>-0</item><item>1.5</item></FLOATS>` and `Object.is(decoded[0], -0)` holds | `"preserves negative zero in FLOAT scalar tables and validates wrappers"` | test/recursive-xrfc.test.ts:966-1004 |
| all 24 scalar EXIDs round-trip; the wire shows `<NUM>0012</NUM>`, `<DATE>2026-07-16</DATE>`, `<TIME>15:45:30</TIME>`; the spelling `1.500000E+000` also decodes to `1.5` | `"round-trips every recursive scalar wire form with xRFC DATE/TIME lexical values"` | test/recursive-xrfc.test.ts:1074-1149 |
| blank DATE/TIME encode to exactly `<TEMPORAL><DATE></DATE><TIME></TIME></TEMPORAL>` | `"canonicalizes blank xRFC DATE/TIME inputs to empty elements"` | test/recursive-xrfc.test.ts:1151-1184 |
| a 1-byte value in a 2-byte fixed `X` encodes as `qgA=`; `qg==` decodes to bytes `aa00`; a 3-byte value throws `/at most 2 bytes/` | `"pads short fixed BYTE values and rejects only oversized values"` | test/recursive-xrfc.test.ts:1186-1221 |
| the writer escapes NUL and U+0001 as `&#00;` and `&#01;`, writes tab, LF and CR raw, and escapes `<`, `&`, `>` as `&#60;`, `&#38;`, `&#62;` | `"uses canonical xRFC entities for XML punctuation and control characters"` | test/recursive-xrfc.test.ts:1223-1241 |
| `&#34;` and `&#39;` decode to the quote and apostrophe; a raw U+0001 in wire text is rejected | same | test/recursive-xrfc.test.ts:1242-1257 |
| a bare `>` in character data is accepted (`decode("a>b")`); the sequence `a]]>b` is rejected | `"admits the whole XML entity grammar a conforming peer may send"` | test/recursive-xrfc.test.ts:1305, 1329 |
| structured classic TABLES rows stay on the binary codec even when a graph is present | `"keeps structured classic TABLES rows binary even when a graph is present"` | test/recursive-xrfc.test.ts:688 |
| depth is counted by containers, not scalar leaves or table rows | `"counts recursive depth by containers, not scalar leaves or table rows"` | test/recursive-xrfc.test.ts:1335 |
| very deep hand graphs are bounded before the JavaScript call stack | `"bounds very deep hand graphs before the JavaScript call stack"` | test/recursive-xrfc.test.ts:1482 |
| a graph map alias (node name differing from map key) is rejected before subtree validation can be reused | `"rejects graph map aliases before subtree validation can be reused"` | test/recursive-xrfc.test.ts:1417 |

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `let raw = 0n; for (let index = encoded.byteLength - 1; index >= 0; index -= 1) { raw = (raw << 8n) \| BigInt(encoded[index]!); } return raw.toString();` | src/values/recursive-xrfc.ts:1036-1041 | Compact temporal values travel on the xRFC wire as their **unsigned decimal raw integer**, up to 8 bytes wide. Use `binary.LittleEndian.Uint64` plus `strconv.FormatUint`. Note this reads unsigned, whereas classic-temporal reads the same bytes signed (src/values/classic-temporal.ts:432) - see open questions. |
| `let raw = parseCanonicalBigInt(text, path); if (raw < 0n \|\| raw >= (1n << BigInt(width * 8))) throw ...` then `bytes[index] = Number(raw & 0xffn); raw >>= 8n;` | src/values/recursive-xrfc.ts:1578-1586 | For `width === 8` the bound is 2^64, which does not fit `int64`. Use `strconv.ParseUint(text, 10, 64)`; the 20-character length guard at src/values/recursive-xrfc.ts:1563 keeps the input parseable. |
| `function parseCanonicalBigInt(value: string, path: string): bigint { if (!CANONICAL_INTEGER.test(value) \|\| value.length > 20) throw ...; return BigInt(value); }` | src/values/recursive-xrfc.ts:1562-1567 | Used for INT1, INT2, INT4, INT8 **and** temporal raws. A single signed `ParseInt` is insufficient for the unsigned 8-byte temporal case; split into signed and unsigned helpers. |
| `case "I": { const value = parseCanonicalBigInt(text, path); if (value < -0x8000_0000n \|\| value > 0x7fff_ffffn) throw ...; return Number(value); }` and the `s` and `b` analogues | src/values/recursive-xrfc.ts:1682-1696 | Decode compares in bigint, then narrows. In Go use `strconv.ParseInt(text, 10, 64)` followed by explicit int32/int16/uint8 range checks, so the range error is the codec's own, not `strconv`'s. |
| `const total = state.bytes + chunk.byteLength; if (!Number.isSafeInteger(total) \|\| total > state.limits.maxParameterBytes)` | src/values/recursive-xrfc.ts:826-831 | Streaming byte accounting with an overflow guard; keep an explicit wraparound test. |
| `const maximumBytes = Math.min(state.limits.maxCellBytes, Math.max(0, state.limits.maxParameterBytes - state.bytes));` | src/values/recursive-xrfc.ts:1206-1209 | The per-cell budget is the remaining parameter budget clamped at zero. Reproduce exactly; a naive subtraction can go negative. |
| `bytes += Buffer.byteLength(chunk, "utf8"); if (!Number.isSafeInteger(bytes) \|\| bytes > maximumBytes) throw` followed by a second pass into `Buffer.allocUnsafe(bytes)` and `if (offset !== bytes) throw` | src/values/recursive-xrfc.ts:958-975 | Two-pass escape with a length invariant, like the classic sibling. Keep both passes. |
| `` chunk = `&#${String(codePoint).padStart(2, "0")};` `` | src/values/recursive-xrfc.ts:955, 967-969 | `fmt.Sprintf("&#%02d;", cp)`. The zero padding matters only for code points 0 through 8; everything else in the set is 11 or greater. |
| `` else if (codePoint <= 0xff) result += `_--${codePoint.toString(16).toUpperCase().padStart(2, "0")}`; `` | src/values/recursive-xrfc.ts:850-851 | `fmt.Sprintf("_--%02X", cp)`. |
| `result += String.fromCharCode(Number.parseInt(hex, 16)); index += 5;` then `if (escapeRecursiveXrfcTag(result) !== value) throw` | src/values/recursive-xrfc.ts:881-887 | `String.fromCharCode` yields a UTF-16 code unit, so an escape of 0x80 through 0xFF becomes U+0080 through U+00FF. In Go that is `rune(v)`, not `byte(v)`. The re-escape round-trip is the canonicality gate: port it. |
| `if (!Number.isSafeInteger(byteLength) \|\| byteLength < 0) throw ...; const groups = Math.ceil(byteLength / 3); const encoded = groups * 4; if (!Number.isSafeInteger(encoded)) throw` | src/values/recursive-xrfc.ts:978-988 | `((n+2)/3)*4` with an overflow test. |
| `if (["C", "N", "D", "T"].includes(field.internalType) && (field.ucLength & 1) !== 0)` | src/values/recursive-xrfc.ts:669 | JS `&` coerces to int32; use `%2` on a validated `int`. |
| `return Object.is(value, -0) ? "-0" : String(value);` for FLOAT | src/values/recursive-xrfc.ts:1103 | Same JS `Number::toString` reproduction requirement as elsewhere; asserted at test/recursive-xrfc.test.ts:1000. Use `math.Signbit` for the negative-zero case. |
| `const value = Number(text); if (!Number.isFinite(value)) throw` for FLOAT | src/values/recursive-xrfc.ts:1676-1679 | `strconv.ParseFloat`; treat `ErrRange` and infinities as invalid. test/recursive-xrfc.test.ts:1145 requires `1.500000E+000` to parse to `1.5`, which Go handles. |
| `if (!Number.isSafeInteger(value) \|\| (value as number) < 0 \|\| (value as number) > maximum)` in `declaredGraphLimit`, applied to `graph.limits.*` | src/values/recursive-xrfc.ts:203-212 | Graph-declared limits are untrusted numbers; keep the bounds. |
| `!Number.isSafeInteger(graph.nodes.size) \|\| graph.nodes.size < 0 \|\| graph.nodes.size > maxNodes` | src/values/recursive-xrfc.ts:242-244 | Map size is treated as untrusted because it can be an accessor on a hand graph. In Go `len()` is trustworthy; keep only the upper-bound check. |
| `subtreeHeight = Math.max(subtreeHeight, 1 + childHeight);` and `if (depth + knownHeight - 1 > maximumDepth)` | src/values/recursive-xrfc.ts:700, 606, 620 | Memoized subtree height in `int`. The minus one exists because the cached height counts the node itself; port exactly or depth limits shift by one. |

### Go mapping notes

- `RESOLVED_PARAMETER_STATE = new WeakMap<object, ResolvedRecursiveXrfcParameterState>()` (src/values/recursive-xrfc.ts:130-133, 388-391, 399) binds a plan to the exact graph and parameter it was resolved for. In Go, store the binding as unexported fields inside the resolved struct and compare pointers in `assertResolvedParameterBinding` (src/values/recursive-xrfc.ts:394-410).
- The `Parser` class (src/values/recursive-xrfc.ts:1450-1534) keeps `nodes`, `rows`, `cells`, `projectedBytes` as public mutable counters; port as an unexported struct with methods.
- Encoding here is chunk-accumulating (`readonly chunks: Buffer[]`, `Buffer.concat(state.chunks, state.bytes)`, src/values/recursive-xrfc.ts:107, 1406) rather than two-pass preallocated as in classic-xrfc.ts and recursive-classic-xrfc.ts. In Go use `bytes.Buffer` and keep the running `state.bytes` limit check.
- `escapeRecursiveXrfcTag` is exported and used directly by tests (test/recursive-xrfc.test.ts:548); keep it exported in Go.
- `nodeRequiresXrfc` (src/values/recursive-xrfc.ts:412-483) and `validateNode` (src/values/recursive-xrfc.ts:590-716) both key the invocation cache by `` `${node.kind} ${node.name}` `` (src/values/recursive-xrfc.ts:418, 614) - a NUL separator. Use the same separator in Go or collisions differ.
- Sentinels: `ErrGraphBudget` (:245, :260, :267, :277), `ErrGraphIdentity` (:292, :316, :363, :371), `ErrRecursiveDepth` (:602), `ErrUnsupportedScalar` (:667, :1051, :1725), `ErrTagEscape` (:852, :879, :886), `ErrNonCanonicalText` (:1414, :1423), `ErrCellLimit` (:960, :1204, :1488).

---

## recursive-serializer-classification.ts

This file contains no wire encoding. It is the policy gate that decides whether a recursive value graph may be sent at all.

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RecursiveSerializerProfile` | type | `export type RecursiveSerializerProfile =\n  \| "offline"\n  \| "abap-7.50"\n  \| "abap-7.58";` | src/values/recursive-serializer-classification.ts:18-21 |
| `LiveRecursiveSerializerProfile` | type | `export type LiveRecursiveSerializerProfile = Exclude<RecursiveSerializerProfile, "offline">;` | src/values/recursive-serializer-classification.ts:23-26 |
| `ObservedRecursiveSerializer` | type | `export type ObservedRecursiveSerializer =\n  \| "classic-xrfc"\n  \| "basxml"\n  \| "unsupported";` | src/values/recursive-serializer-classification.ts:28-31 |
| `RecursiveSerializerObservation` | interface | `readonly defaultSerializer: ObservedRecursiveSerializer;` `readonly basxmlDisabledSerializer: "classic-xrfc" \| "unsupported";` | src/values/recursive-serializer-classification.ts:33-36 |
| `LiveRecursiveSerializerPolicy` | interface | `readonly profile: LiveRecursiveSerializerProfile;` `readonly observation: RecursiveSerializerObservation;` | src/values/recursive-serializer-classification.ts:44-47 |
| `RecursiveSerializerClassificationRequest` | interface | `profile`, `graph`, `parameters`, optional `observation` | src/values/recursive-serializer-classification.ts:49-54 |
| `RecursiveSerializerClassification` | interface | `schemaVersion: 1; profile; graphSha256: ` + "`sha256:${string}`" + `; parameterCount: number; parameterNames: readonly string[]; remoteBasxmlSupported: boolean \| undefined; selectedSerializer: "classic-xrfc" \| "basxml-required"; status: "offline" \| "live" \| "blocked"; sendAllowed: boolean; basxmlNegotiation: "unknown" \| "disabled" \| "required"` | src/values/recursive-serializer-classification.ts:56-67 |
| `RecursiveSerializerDecisionRequest` | interface | `readonly graph: RecursiveMetadataGraph;` `readonly parameters: readonly RecursiveClassicXrfcParameterIdentity[];` | src/values/recursive-serializer-classification.ts:69-72 |
| `RecursiveSerializerDecisionProvider` | type | `export type RecursiveSerializerDecisionProvider = (\n  request: RecursiveSerializerDecisionRequest,\n) => RecursiveSerializerClassification;` | src/values/recursive-serializer-classification.ts:75-77 |
| `RecursiveSerializerClassificationError` | class | `export class RecursiveSerializerClassificationError extends Error` with `readonly code: string;` and `` super(`recursive serializer classification rejected: ${code}`) `` | src/values/recursive-serializer-classification.ts:79-87 |
| `recursiveMetadataGraphSha256` | function | `export function recursiveMetadataGraphSha256(\n  graph: RecursiveMetadataGraph,\n): ` + "`sha256:${string}`" | src/values/recursive-serializer-classification.ts:154-156 |
| `snapshotLiveRecursiveSerializerPolicy` | function | `export function snapshotLiveRecursiveSerializerPolicy(\n  value: LiveRecursiveSerializerPolicy,\n): LiveRecursiveSerializerPolicy` | src/values/recursive-serializer-classification.ts:279-281 |
| `createLiveRecursiveSerializerDecisionProvider` | function | `export function createLiveRecursiveSerializerDecisionProvider(\n  policy: LiveRecursiveSerializerPolicy,\n): RecursiveSerializerDecisionProvider` | src/values/recursive-serializer-classification.ts:303-305 |
| `admitLiveRecursiveSerializer` | function | `export function admitLiveRecursiveSerializer(\n  policy: LiveRecursiveSerializerPolicy \| undefined,\n  graph: RecursiveMetadataGraph,\n  parameters: readonly RecursiveClassicXrfcParameterIdentity[],\n): RecursiveSerializerClassification` | src/values/recursive-serializer-classification.ts:324-328 |
| `assertRecursiveSerializerSendDecision` | function | `export function assertRecursiveSerializerSendDecision(\n  request: RecursiveSerializerDecisionRequest,\n  decision: RecursiveSerializerClassification,\n): RecursiveSerializerClassification` | src/values/recursive-serializer-classification.ts:347-350 |
| `classifyRecursiveSerializer` | function | `export function classifyRecursiveSerializer(\n  request: RecursiveSerializerClassificationRequest,\n): RecursiveSerializerClassification` | src/values/recursive-serializer-classification.ts:407-409 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `PROFILES` | `new Set<RecursiveSerializerProfile>([\n  "offline",\n  "abap-7.50",\n  "abap-7.58",\n])` | | src/values/recursive-serializer-classification.ts:89-93 |
| `OBSERVED` | `new Set<ObservedRecursiveSerializer>([\n  "classic-xrfc",\n  "basxml",\n  "unsupported",\n])` | | src/values/recursive-serializer-classification.ts:94-98 |
| parameter-class grammar | `/^[IECT]$/u` | | src/values/recursive-serializer-classification.ts:200 |
| identity key count | `keys.length !== 5` | exactly five own keys required | src/values/recursive-serializer-classification.ts:190 |
| observation key count | `Reflect.ownKeys(record).length !== 2` | | src/values/recursive-serializer-classification.ts:261 |
| graph hash | `` return `sha256:${createHash("sha256").update(bytes).digest("hex")}`; `` over `JSON.stringify(graphProjection(graph))` | stable content identity | src/values/recursive-serializer-classification.ts:154-158 |
| graph projection field order | `version, functionIdentity, nodes (entries sorted by `left.localeCompare(right)`), parameters, rootTypeNames, cycles, limits, statistics` | the exact hashed shape | src/values/recursive-serializer-classification.ts:137-151 |
| offline outcome | `selectedSerializer: "classic-xrfc", status: "offline", sendAllowed: true, basxmlNegotiation: "unknown"` | | src/values/recursive-serializer-classification.ts:435-441 |
| live-permitted outcome | `selectedSerializer: "classic-xrfc", status: "live", sendAllowed: true, basxmlNegotiation: "disabled"` | when `basxmlDisabledSerializer === "classic-xrfc"` and the default is not `"unsupported"` | src/values/recursive-serializer-classification.ts:445-455 |
| blocked outcome | `selectedSerializer: "basxml-required", status: "blocked", sendAllowed: false, basxmlNegotiation: "required"` | when the default serializer is `"basxml"` | src/values/recursive-serializer-classification.ts:458-467 |

### Errors

Every failure is a `RecursiveSerializerClassificationError` whose message is `` `recursive serializer classification rejected: ${code}` `` (src/values/recursive-serializer-classification.ts:83). The complete code set:

| Code (verbatim) | Trigger condition | Citation |
|---|---|---|
| `"untrusted-graph"` | graph fails `isNormalizedRecursiveMetadataGraph` | src/values/recursive-serializer-classification.ts:138, 353, 414 |
| `"parameter-inventory"` | parameters not a plain array, empty, longer than the index, or an accessor-backed element | src/values/recursive-serializer-classification.ts:170, 175, 182 |
| `"parameter-identity"` | wrong key count/types, class not in `IECT`, function-name mismatch, duplicate parameter name | src/values/recursive-serializer-classification.ts:186, 205, 208, 212 |
| `"observation"` | observation shape invalid | src/values/recursive-serializer-classification.ts:269 |
| `"live-policy"` | policy shape invalid, profile `"offline"`, or missing observation | src/values/recursive-serializer-classification.ts:290, 295 |
| `"live-policy-required"` | `admitLiveRecursiveSerializer` called with `undefined` policy | src/values/recursive-serializer-classification.ts:329 |
| `"decision-request"` | request is not a plain record | src/values/recursive-serializer-classification.ts:308, 351 |
| `"untrusted-decision"` | decision not produced by `trustedClassification` | src/values/recursive-serializer-classification.ts:363 |
| `"graph-mismatch"` | `decision.graphSha256` differs from the recomputed hash | src/values/recursive-serializer-classification.ts:366 |
| `"parameter-inventory-mismatch"` | decision's parameter names/count differ from the sorted active inventory | src/values/recursive-serializer-classification.ts:376 |
| `"graph-capability-mismatch"` | `decision.remoteBasxmlSupported` differs from `graph.functionIdentity?.remoteBasxmlSupported` | src/values/recursive-serializer-classification.ts:382 |
| `"offline-decision"` | an offline decision used at a send boundary | src/values/recursive-serializer-classification.ts:384 |
| `"basxml-required"` | decision selected basXML or negotiation is required | src/values/recursive-serializer-classification.ts:389 |
| `"live-decision-required"` | status/serializer/sendAllowed/negotiation not the exact live-classic tuple | src/values/recursive-serializer-classification.ts:397 |
| `"profile"` | profile not in `PROFILES` | src/values/recursive-serializer-classification.ts:412 |
| `"offline-classification"` | offline profile supplied with an observation | src/values/recursive-serializer-classification.ts:434 |
| `"live-classification-required"` | live profile without an observation | src/values/recursive-serializer-classification.ts:443 |
| `"contradictory-classification"` | `basxmlDisabledSerializer === "classic-xrfc"` with default `"unsupported"`, or default not `"basxml"` in the remaining branch | src/values/recursive-serializer-classification.ts:447, 459 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "Explicit partner observation required before a live recursive request may leave a direct CPIC session. This is deliberately separate from the graph: BASXML_SUPPORTED is a capability bit, not proof of the serializer selected for one concrete function/value graph." | src/values/recursive-serializer-classification.ts:38-43 |
| "Synchronous evidence resolver called at the live pre-send boundary." | src/values/recursive-serializer-classification.ts:74 |
| "Stable content identity for one bounded normalized recursive metadata graph." | src/values/recursive-serializer-classification.ts:153 |
| "Prefer the independently qualified strict codec. When that older subset rejects a supported scalar, require the broader codec to resolve and validate the complete reachable graph before the same graph can obtain a send decision. The adapter supplies no wire geometry: that remains owned by the normalized recursive metadata graph." | src/values/recursive-serializer-classification.ts:216-219 |
| "Capture an immutable live policy before session setup performs any I/O." | src/values/recursive-serializer-classification.ts:278 |
| "Create one immutable synchronous provider from a captured observation." | src/values/recursive-serializer-classification.ts:302 |
| "Convert an explicit live policy into the one graph-bound send decision. A basXML-required decision is an error here because the direct session cannot silently substitute that serializer." | src/values/recursive-serializer-classification.ts:319-323 |
| "Validate that a classifier-produced decision authorizes this exact graph and active parameter inventory. A JSON lookalike cannot authorize network I/O." | src/values/recursive-serializer-classification.ts:343-346 |
| "Classify one deep-call graph without silently substituting serializers. Offline classification proves only local classic-xRFC capability. A live profile additionally requires the paired default/basXML-disabled observation." | src/values/recursive-serializer-classification.ts:402-406 |

### Wire facts asserted by tests

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| an offline profile classifies and hashes the bounded graph as classic-xRFC | `"classifies and hashes the bounded recursive graph as offline classic-xRFC"` | test/recursive-classic-xrfc.test.ts:631 |
| a scalar accepted only by the broad codec is still admitted through the exact live graph decision | `"admits a broad-only recursive scalar through the exact live graph decision"` | test/recursive-classic-xrfc.test.ts:654 |
| paired release observations are required before live classic-xRFC is admitted | `"requires paired release observations before admitting live classic-xRFC"` | test/recursive-classic-xrfc.test.ts:686 |
| one exact live policy is snapshotted and admission is bound to the graph | `"snapshots one exact live policy and binds admission to the graph"` | test/recursive-classic-xrfc.test.ts:711 |
| only trusted live decisions for the exact active inventory are accepted | `"accepts only trusted live decisions for the exact active inventory"` | test/recursive-classic-xrfc.test.ts:761 |
| a graph requiring basXML is blocked rather than silently falling back | `"blocks a graph that requires basXML instead of silently falling back"` | test/recursive-classic-xrfc.test.ts:828 |
| contradictory or untrusted serializer observations are rejected | `"rejects contradictory or untrusted serializer observations"` | test/recursive-classic-xrfc.test.ts:858 |
| trusted normalized metadata and internally resolved plans are required | `"requires trusted normalized metadata and internally resolved plans"` | test/recursive-classic-xrfc.test.ts:1306 |

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `const bytes = JSON.stringify(graphProjection(graph));` then SHA-256 over that string | src/values/recursive-serializer-classification.ts:154-158 | **The hash is over `JSON.stringify` output.** Go's `encoding/json` differs from `JSON.stringify` in HTML escaping (`<`, `>`, `&` become `<` etc. unless `SetEscapeHTML(false)`), in float formatting, and in map key ordering. Any of these changes the digest and breaks `"graph-mismatch"` compatibility across implementations. Either reproduce `JSON.stringify` byte-for-byte or declare the digest implementation-local and never compare it across languages. |
| `.sort(([left], [right]) => left.localeCompare(right))` on node entries | src/values/recursive-serializer-classification.ts:141-143 | `localeCompare` is **locale-sensitive** and is not byte order. Go's `sort.Strings` is byte order. For ASCII type names these agree; for non-ASCII they may not. Prefer an explicit byte-order sort and record the deviation. |
| `parameters.map((parameter) => parameter.parameterName).sort()` | src/values/recursive-serializer-classification.ts:369-370, 427-429 | Default `Array.prototype.sort` is UTF-16 code-unit order, which differs from Go's UTF-8 byte order for code points above U+FFFF. Use UTF-16 code-unit ordering if exact parity matters. |
| `schemaVersion: 1 as const` | src/values/recursive-serializer-classification.ts:423 | Plain `int` constant. |
| `if (input.length > parameterIndex.parameterCount)` | src/values/recursive-serializer-classification.ts:174 | Plain length comparison. |

### Go mapping notes

- `trustedClassifications = new WeakSet<object>()` (src/values/recursive-serializer-classification.ts:99, 108-110, 362) is the "a JSON lookalike cannot authorize network I/O" mechanism (src/values/recursive-serializer-classification.ts:345). In Go, make `Classification` a struct with an unexported non-zero marker field set only by `trustedClassification`, or return an interface implemented only by an unexported type.
- `RecursiveSerializerClassificationError.code` is the whole error taxonomy. In Go, define one sentinel per code and wrap, so `errors.Is(err, ErrBasxmlRequired)` works; keep the message prefix `recursive serializer classification rejected: ` so existing log greps still match.
- `createLiveRecursiveSerializerDecisionProvider` returns `Object.freeze((request) => ...)` (src/values/recursive-serializer-classification.ts:307-316) - a closure. In Go return a `func(DecisionRequest) (Classification, error)` or a small struct implementing the provider interface.
- `plainRecord` (src/values/recursive-serializer-classification.ts:113-135) is JS hardening; delete in Go.
- The strict-then-broad fallback in `snapshotParameters` (src/values/recursive-serializer-classification.ts:220-250) rethrows the **strict** error when both codecs reject (`throw strictError` at :248). Preserve that: the broad codec's error is discarded.

---

## rfc-value-snapshot.ts

### Exported symbols

| Symbol | Kind | Signature (verbatim) | Citation |
|---|---|---|---|
| `RfcValueSnapshotOptions` | interface | `readonly accessorPolicy?: "reject" \| "readOnce";` `readonly maxNodes?: number;` `readonly maxArrayLength?: number;` | src/values/rfc-value-snapshot.ts:27-34 |
| `snapshotRfcValue` | function | `export function snapshotRfcValue(\n  value: unknown,\n  path = "RFC value",\n  options: RfcValueSnapshotOptions = {},\n): unknown` | src/values/rfc-value-snapshot.ts:261-265 |

### Numeric and format constants

| Name | Value (verbatim) | Meaning per source | Citation |
|---|---|---|---|
| `MAX_RFC_VALUE_DEPTH` | `64` | container **and** prototype-chain depth ceiling | src/values/rfc-value-snapshot.ts:8, 92, 133 |
| `MAX_RFC_VALUE_NODES` | `1_000_000` | | src/values/rfc-value-snapshot.ts:9 |
| `MAX_RFC_ARRAY_LENGTH` | `100_000` | | src/values/rfc-value-snapshot.ts:10 |
| `MAX_RFC_VALUE_RETAINED_BYTES` | `2 * 1_400_000` | "A value snapshot also accounts for JavaScript containers and property names, so it needs bounded headroom above the 1.4 MB encoded CPIC envelope. The encoder still enforces the smaller wire limit before allocating value buffers." | src/values/rfc-value-snapshot.ts:11-14 |
| byte cost: Uint8Array | `claimBytes(state, 16 + byteLength, path);` | | src/values/rfc-value-snapshot.ts:146 |
| byte cost: array | `claimBytes(state, 16 + value.length * 8, path);` | | src/values/rfc-value-snapshot.ts:165 |
| byte cost: object | `claimBytes(state, 16, path);` plus `claimBytes(state, 8 + utf8ByteLength(key, "utf8"), ...)` per key | | src/values/rfc-value-snapshot.ts:205, 208 |
| byte cost: string | `claimBytes(state, utf8ByteLength(value, "utf8"), path);` | | src/values/rfc-value-snapshot.ts:242 |
| byte cost: number or bigint | `claimBytes(state, 8, path);` | **both** cost 8 bytes regardless of magnitude | src/values/rfc-value-snapshot.ts:243-247 |
| byte cost: boolean, null, undefined | `claimBytes(state, 1, path);` | | src/values/rfc-value-snapshot.ts:248-253 |
| default accessor policy | `const accessorPolicy = options.accessorPolicy ?? "reject";` | | src/values/rfc-value-snapshot.ts:266 |

### Errors

| Message text (verbatim) | Trigger condition | Citation |
|---|---|---|
| `` `${path} must be an integer in 1..${maximum}` `` (RangeError) | `maxNodes`/`maxArrayLength` not a safe integer in `1..maximum` | src/values/rfc-value-snapshot.ts:47 |
| `` `${path} must be an own data property` `` (TypeError) | accessor property under the `"reject"` policy | src/values/rfc-value-snapshot.ts:63, 178, 211 |
| `` `${path} exceeds the ${MAX_RFC_VALUE_RETAINED_BYTES}-byte snapshot limit` `` (RangeError) | byte budget exhausted | src/values/rfc-value-snapshot.ts:70-72 |
| `` `${path} exceeds the ${state.nodeLimit} value-node snapshot limit` `` (RangeError) | node budget exhausted | src/values/rfc-value-snapshot.ts:81-83 |
| `` `${path} must not have a proxy prototype` `` (TypeError) | a proxy in the prototype chain | src/values/rfc-value-snapshot.ts:94 |
| `` `${path} decimal object's toString() must return a string` `` (TypeError) | | src/values/rfc-value-snapshot.ts:108 |
| `` `${path} exceeds the ${MAX_RFC_VALUE_DEPTH}-level prototype depth` `` (RangeError) | prototype chain longer than 64 | src/values/rfc-value-snapshot.ts:117-119 |
| `` `${path} must not be a proxy` `` (TypeError) | | src/values/rfc-value-snapshot.ts:131 |
| `` `${path} exceeds the ${MAX_RFC_VALUE_DEPTH}-level snapshot depth` `` (RangeError) | | src/values/rfc-value-snapshot.ts:134-136 |
| `` `${path} contains a cyclic RFC value` `` (TypeError) | | src/values/rfc-value-snapshot.ts:139 |
| `` `${path} exceeds the ${state.maxArrayLength}-row array snapshot limit` `` (RangeError) | | src/values/rfc-value-snapshot.ts:161-163 |
| `` `${path} must be a dense array without extra keys` `` (TypeError) | sparse array or extra own keys | src/values/rfc-value-snapshot.ts:168, 174 |
| `` `${path}[${index}] must be an own data property` `` (TypeError) | | src/values/rfc-value-snapshot.ts:178 |
| `` `${path} must contain only plain RFC value objects` `` (TypeError) | prototype is neither `Object.prototype` nor `null` | src/values/rfc-value-snapshot.ts:194 |
| `` `${path} must not contain enumerable symbol keys` `` (TypeError) | | src/values/rfc-value-snapshot.ts:200 |
| `` `${path} must not be a function` `` (TypeError) | | src/values/rfc-value-snapshot.ts:239 |
| `"RFC value snapshot accessorPolicy is invalid"` (TypeError) | | src/values/rfc-value-snapshot.ts:268 |

### Invariants stated in comments

| Verbatim quote | Citation |
|---|---|
| "A value snapshot also accounts for JavaScript containers and property names, so it needs bounded headroom above the 1.4 MB encoded CPIC envelope. The encoder still enforces the smaller wire limit before allocating value buffers." | src/values/rfc-value-snapshot.ts:11-13 |
| "Existing low-level seams may retain their historical single getter read." | src/values/rfc-value-snapshot.ts:28 |
| "A caller may tighten, but never raise, the aggregate value-node budget." | src/values/rfc-value-snapshot.ts:30 |
| "A caller may tighten, but never raise, the per-array/table row budget." | src/values/rfc-value-snapshot.ts:32 |
| "A plain RFC structure may legitimately contain a field named `toString`. Only a data-method descriptor opts an object into decimal conversion; every other descriptor continues through ordinary plain object validation without being read here." | src/values/rfc-value-snapshot.ts:100-103 |
| "Capture caller-owned nested RFC values before any asynchronous boundary." | src/values/rfc-value-snapshot.ts:260 |

### Wire facts asserted by tests

This file has no wire format. Behavioural assertions:

| Assertion | Test name (verbatim) | Citation |
|---|---|---|
| a proxy is rejected without any trap executing | `"snapshot rejects proxies without executing their traps"` | test/rfc-value-snapshot.test.ts:6-20 |
| a proxy prototype is rejected without any trap executing | `"snapshot rejects a proxy prototype without executing its traps"` | test/rfc-value-snapshot.test.ts:22-37 |
| the row limit is enforced before any row getter runs: `/TABLE exceeds the 2-row array snapshot limit/` with `reads === 0` | `"snapshot applies a conservative per-array row limit before reading rows"` | test/rfc-value-snapshot.test.ts:39-59 |
| one aggregate node budget spans nested tables: `maxNodes: 5` succeeds, `maxNodes: 4` fails with `/TABLES\[1\]\[0\] exceeds the 4 value-node snapshot limit/` | `"snapshot applies one aggregate node budget across nested tables"` | test/rfc-value-snapshot.test.ts:61-70 |
| `{ maxNodes: 0 }`, `{ maxNodes: 1_000_001 }`, `{ maxArrayLength: 0 }`, `{ maxArrayLength: 100_001 }`, `{ maxNodes: 1.5 }` all throw `/must be an integer in/` | `"snapshot bounds cannot relax the shipped row and node ceilings"` | test/rfc-value-snapshot.test.ts:72-85 |
| a decimal object's `toString` is called exactly once and the captured text does not change afterwards; a class instance is likewise converted once; a function value is rejected | `"snapshot captures decimal-object conversion once and rejects function values"` | test/rfc-value-snapshot.test.ts:87-119 |

### JavaScript number-semantics dependencies

| What the code does (quoted) | Citation | Go porter must do |
|---|---|---|
| `if (!Number.isSafeInteger(candidate) \|\| candidate < 1 \|\| candidate > maximum)` | src/values/rfc-value-snapshot.ts:41-48 | Note the lower bound is **1**, not 0, unlike every other limit normalizer in this layer. test/rfc-value-snapshot.test.ts:74 asserts `maxNodes: 0` is rejected and test/rfc-value-snapshot.test.ts:78 asserts `1.5` is rejected. |
| `if (!Number.isSafeInteger(bytes) \|\| bytes < 0 \|\| bytes > state.remainingBytes)` | src/values/rfc-value-snapshot.ts:69 | Byte-budget guard; keep the negative check. |
| `} else if (typeof value === "number" \|\| typeof value === "bigint") { claimBytes(state, 8, path); }` | src/values/rfc-value-snapshot.ts:243-247 | A `bigint` of any magnitude is charged 8 bytes. In Go the analogue for a big integer is not constant-size; keep the flat 8-byte charge to match the budget arithmetic rather than charging the true size. |
| `claimBytes(state, utf8ByteLength(value, "utf8"), path);` where `utf8ByteLength = Buffer.byteLength.bind(Buffer)` | src/values/rfc-value-snapshot.ts:15, 242 | `len(s)` in Go is already UTF-8 byte length; direct. |
| `claimBytes(state, 16 + value.length * 8, path);` | src/values/rfc-value-snapshot.ts:165 | `value.length` is bounded by `maxArrayLength` (100 000) at :160, so the product cannot overflow. Keep the ordering: the length check precedes the multiplication. |
| `state.remainingNodes -= 1; if (state.remainingNodes < 0) throw` | src/values/rfc-value-snapshot.ts:78-84 | Decrement-then-test; a plain `int` counter. |

### Go mapping notes

- **This file may not need a Go port at all.** Its entire purpose is to defend against caller-owned mutable JavaScript objects (proxies, getters, prototype chains, cycles) crossing an async boundary. Go's value semantics and the absence of proxies remove most of it. What survives is: cycle detection (src/values/rfc-value-snapshot.ts:138-140) if the Go input type is a self-referential `map[string]any`, the aggregate node/byte budgets, and the array-length ceiling.
- `WeakSet`/`WeakMap` for `visiting`/`completed` (src/values/rfc-value-snapshot.ts:23-24, 286-287) become `map[uintptr]bool`-style identity sets only if the Go input is pointer-based; for value types they are unnecessary.
- The `"readOnce"` accessor policy (src/values/rfc-value-snapshot.ts:29, 60-65) has no Go analogue - drop it and the corresponding test at test/rfc-value-snapshot.test.ts:39.
- `snapshotStringConvertible` (src/values/rfc-value-snapshot.ts:86-122) is the decimal-object hook that feeds packed-decimal and decimal-float; in Go it becomes a `fmt.Stringer` check with no prototype walk.
- Sentinels: `ErrSnapshotDepth`, `ErrSnapshotNodes`, `ErrSnapshotBytes`, `ErrSnapshotCyclic`, `ErrSnapshotShape`.

---

## Candidate conformance vectors

These are the pure byte-in/byte-out cases suitable for `conformance/testdata/vectors/`. Every input and expected output is verbatim from the test.

### DECF16 (IEEE 754 decimal64, little-endian hex)

From `"matches IEEE 754-2008 decimal64 DPD vectors"`, test/decimal-float.test.ts:95-107, as `[input, hex, decoded-output]`:

| input | hex | output |
|---|---|---|
| `"0"` | `"0000000000003822"` | `"0"` |
| `"-0"` | `"00000000000038a2"` | `"-0"` |
| `"1"` | `"0100000000003822"` | `"1"` |
| `"-1"` | `"01000000000038a2"` | `"-1"` |
| `"123.45"` | `"c549000000003022"` | `"123.45"` |
| `"123.45E67"` | `"c549000000003c23"` | `"1.2345E+69"` |
| `"9.999999999999999E+384"` | `"fffcf3cf3ffffc77"` | `"9.999999999999999E+384"` |
| `"1E-383"` | `"0100000000003c00"` | `"1E-383"` |
| `"NaN"` | `"000000000000007c"` | `"NaN"` |
| `"Infinity"` | `"0000000000000078"` | `"Infinity"` |
| `"-Infinity"` | `"00000000000000f8"` | `"-Infinity"` |

Additional DECF16 encode-only and decode-only vectors, from `"preserves exact cohorts, signed zero, subnormals, and range-edge values"` (test/decimal-float.test.ts:164-193) and `"rescales excess trailing zeros exactly before enforcing precision and qmin"` (test/decimal-float.test.ts:195-246):

| input | hex | decoded output |
|---|---|---|
| `"-0.00"` | `"00000000000030a2"` | `"-0.00"` (test/decimal-float.test.ts:166-167) |
| `"1E-398"` | `"0100000000000000"` | `"1E-398"` (test/decimal-float.test.ts:168-169) |
| `"10E-399"` | (not asserted) | `"1E-398"` (test/decimal-float.test.ts:170) |
| `"1E+384"` | `"000000000000fc47"` | `"1.000000000000000E+384"` (test/decimal-float.test.ts:171-175) |
| `"0E-999"` | (not asserted) | `"0E-398"` (test/decimal-float.test.ts:176) |
| `"-0E+999"` | (not asserted) | `"-0E+369"` (test/decimal-float.test.ts:177) |
| `"12345678901234560"` | `"568ee2c1b9343d26"` | `"1.234567890123456E+16"` (test/decimal-float.test.ts:196-203) |
| `"10000000000000000E-414"` | `"0100000000000000"` | `"1E-398"` (test/decimal-float.test.ts:204-211) |
| `"1.2300"` | (not asserted) | `"1.2300"` (test/decimal-float.test.ts:165) |
| decode-only `"ffffffffffffffff"` | - | `"-sNaN999999999999999"` (test/decimal-float.test.ts:257-258) |
| decode-only `"fffffffffffffffb"` | - | `"-Infinity"` (test/decimal-float.test.ts:259-260) |
| `123.45` (JS number) | (not asserted) | `"123.45"` (test/decimal-float.test.ts:304) |

### DECF34 (IEEE 754 decimal128, little-endian hex)

From `"matches IEEE 754-2008 decimal128 DPD vectors"`, test/decimal-float.test.ts:116-132:

| input | hex | output |
|---|---|---|
| `"0"` | `"00000000000000000000000000000822"` | `"0"` |
| `"-0"` | `"000000000000000000000000000008a2"` | `"-0"` |
| `"1"` | `"01000000000000000000000000000822"` | `"1"` |
| `"-1"` | `"010000000000000000000000000008a2"` | `"-1"` |
| `"123.45"` | `"c5490000000000000000000000800722"` | `"123.45"` |
| `"123.45E67"` | `"c5490000000000000000000000401822"` | `"1.2345E+69"` |
| `"9.999999999999999999999999999999999E+6144"` | `"fffcf3cf3ffffcf3cf3ffffcf3cfff77"` | `"9.999999999999999999999999999999999E+6144"` |
| `"1E-6143"` | `"01000000000000000000000000400800"` | `"1E-6143"` |
| `"NaN"` | `"0000000000000000000000000000007c"` | `"NaN"` |
| `"Infinity"` | `"00000000000000000000000000000078"` | `"Infinity"` |
| `"-Infinity"` | `"000000000000000000000000000000f8"` | `"-Infinity"` |

Additional DECF34, test/decimal-float.test.ts:179-236:

| input | hex | decoded output |
|---|---|---|
| `"1E-6176"` | `"01000000000000000000000000000000"` | `"1E-6176"` |
| `"1E+6144"` | `"00000000000000000000000000c0ff47"` | `` `1.${"0".repeat(33)}E+6144` `` |
| `"0E-99999"` | (not asserted) | `"0E-6176"` |
| `"12345678901234567890123456789012340"` | `"3435827771123c6fe5281e9c4b530826"` | `"1.234567890123456789012345678901234E+34"` |
| `"10000000000000000000000000000000000E-6210"` | `"01000000000000000000000000000000"` | `"1E-6176"` |
| `12345678901234567890n` (bigint) | (not asserted) | `"12345678901234567890"` (test/decimal-float.test.ts:305) |

### Compact temporal (little-endian hex)

From `"matches the compact-temporal little-endian reference vectors"`, test/classic-temporal.test.ts:43-53, as `[exid, value, hex]`, verified in both directions:

| exid | value | hex |
|---|---|---|
| `"p"` | `"2002-02-04T20:15:01.1234567"` | `"08272f17627dc308"` |
| `"n"` | `"2002-02-04T20:15:01"` | `"c685f3b30e000000"` |
| `"w"` | `"2002-02-04T20:15"` | `"8086bb3e00000000"` |
| `"d"` | `"2002-02-04"` | `"07270b00"` |
| `"7"` | `"2020-W53"` | `"b99b0100"` |
| `"x"` | `"2002-02"` | `"ce5d0000"` |
| `"t"` | `"20:15:01"` | `"c61c0100"` |
| `"i"` | `"20:15"` | `"c004"` |
| `"c"` | `"02-04"` | `"2300"` |

From `"covers every compact temporal minimum and maximum"`, test/classic-temporal.test.ts:85-112, as `[exid, minimum, minimumRaw, maximum, maximumRaw]`, verified in both directions:

| exid | minimum | minimumRaw | maximum | maximumRaw |
|---|---|---|---|---|
| `"p"` | `"0001-01-01T00:00:00.0000000"` | `"0100000000000000"` | `"9999-12-31T23:59:59.9999999"` | `"00c00a49082aca2b"` |
| `"n"` | `"0001-01-01T00:00:00"` | `"0100000000000000"` | `"9999-12-31T23:59:59"` | `"80db887749000000"` |
| `"w"` | `"0001-01-01T00:00"` | `"0100000000000000"` | `"9999-12-31T23:59"` | `"207b753901000000"` |
| `"d"` | `"0001-01-01"` | `"01000000"` | `"9999-12-31"` | `"ddb93700"` |
| `"7"` | `"0000-W53"` | `"01000000"` | `"9999-W52"` | `"fdf50700"` |
| `"x"` | `"0001-01"` | `"01000000"` | `"9999-12"` | `"b4d40100"` |
| `"t"` | `"00:00:00"` | `"01000000"` | `"24:00:00"` | `"81510100"` |
| `"i"` | `"00:00"` | `"0100"` | `"24:00"` | `"a105"` |
| `"c"` | `"01-01"` | `"0100"` | `"12-31"` | `"6e01"` |

Calendar-gap vectors, from `"uses consecutive ordinals across the historical Julian-to-Gregorian gap"`, test/classic-temporal.test.ts:123-132:

| exid | value | hex |
|---|---|---|
| `"d"` | `"1582-10-04"` | `"c9d00800"` |
| `"d"` | `"1582-10-15"` | `"cad00800"` |

Week-53 vectors, from `"validates hybrid-calendar week 53 and its reserved year-zero value"`, test/classic-temporal.test.ts:162-165 (encode direction only):

| exid | value | hex |
|---|---|---|
| `"7"` | `"0000-W53"` | `"01000000"` |
| `"7"` | `"0001-W01"` | `"02000000"` |
| `"7"` | `"0005-W53"` | `"06010000"` |
| `"7"` | `"2020-W53"` | `"b99b0100"` |

Initial-value vectors, from `"uses raw zero only for initial values and preserves node-rfc UTCLONG initial"`, test/classic-temporal.test.ts:65-78: `encodeClassicTemporal("p", "0000-00-00T00:00:00.0000000")` and `encodeClassicTemporal("p", "")` both give `"0000000000000000"`; for each of `n w d 7 x t i c`, the empty string gives all-zero bytes of the fixed width and all-zero bytes decode to `""`.

Invalid-raw vectors: `"deb93700"` for `"d"` and `"ffff"` for `"i"` must both fail with "outside its valid raw range" (test/classic-temporal.test.ts:198-205).

### Flat xRFC XML (UTF-8 text)

From `"encodes the STFC_DEEP_TABLE xRFC XML request exactly"`, test/classic-xrfc.test.ts:108-123. Definition `STFC_ROW` at test/classic-xrfc.test.ts:11-52; rows `CAPTURED_ROWS` at test/classic-xrfc.test.ts:54-67. Expected output:

```
<IMPORT_TAB><item><I>42</I><C>ROW_ONE</C><STR>A&#60;&#38;"-nested</STR><XSTR>AKX/</XSTR></item><item><I>-7</I><C>ROW_TWO</C><STR>second-row</STR><XSTR>ECAwQA==</XSTR></item></IMPORT_TAB>
```

From `"decodes the STFC_DEEP_TABLE response and numeric entities"`, test/classic-xrfc.test.ts:126-146, the input document is the same shape with `EXPORT_TAB`, `&#34;` in place of the raw quote, and a third row `<item><I>10</I><C>Appended</C><STR>20260716</STR><XSTR>3q2+7w==</XSTR></item>`; expected decoded rows include `XSTR: Buffer.from("deadbeef", "hex")`.

From `"matches the Unicode and explicit-empty STFC vector"`, test/classic-xrfc.test.ts:180-215 - request byte length exactly `163`, response byte length exactly `240`:

```
<IMPORT_TAB><item><I>42</I><C>UNICODE</C><STR>Grüße 🌍</STR><XSTR>3q2+7w==</XSTR></item><item><I>-7</I><C>EMPTY</C><STR></STR><XSTR></XSTR></item></IMPORT_TAB>
```

From `"round-trips the extended flat xRFC scalar set with compatibility modes"`, test/classic-xrfc.test.ts:244-284. Definition `EXTENDED_ROW` at test/classic-xrfc.test.ts:69-106; input at test/classic-xrfc.test.ts:249-259 with `{ int8Mode: "string" }`; expected output:

```
<ROW><NUM>0012</NUM><DATE>2026-07-17</DATE><TIME>15:45:30</TIME><BYTE>qgA=</BYTE><BCD>12.34</BCD><FLOAT>-0</FLOAT><INT8>-9007199254740993</INT8><TEXT>ready</TEXT></ROW>
```

Decoding that document with `{ int8Mode: "string", bcd: "number" }` must yield `{ NUM: "0012", DATE: "20260717", TIME: "154530", BYTE: Buffer.from([0xaa, 0]), BCD: 12.34, FLOAT: -0, INT8: "-9007199254740993", TEXT: "ready" }` (test/classic-xrfc.test.ts:275-284).

FLOAT lexical acceptance table (test/classic-xrfc.test.ts:298-306), decoding `<FLOAT>X</FLOAT>` in the document above: accepted `"1.5"→1.5`, `"+1.5"→1.5`, `"01.5"→1.5`, `"1."→1`, `".5"→0.5`, `"-2"→-2`, `"1e3"→1000`, `"+1.5E+02"→150`, `"0"→0`; rejected `""`, `"."`, `"+"`, `"1.5.5"`, `"0x10"`, `"1e"`, `"NaN"`, `"Infinity"`, `" 1"`.

### Recursive xRFC XML (broad codec, UTF-8 text)

From `"encodes and decodes nested structures, tables, and XSTRING exactly"`, test/recursive-xrfc.test.ts:275-301; input value at test/recursive-xrfc.test.ts:276-284:

```
<ROOT><TEXT>A&#60;&#38;</TEXT><CHILD><COUNT>42</COUNT></CHILD><ROWS><item><VALUE>0001</VALUE><PAYLOAD>AKX/</PAYLOAD></item><item><VALUE>0002</VALUE><PAYLOAD></PAYLOAD></item></ROWS><BLOB>3q2+7w==</BLOB></ROOT>
```

From `"supports anonymous scalar table rows and namespace tag escaping"`, test/recursive-xrfc.test.ts:531-558 - input `["ONE", "TWO"]` for parameter `/NS/TEXTS`:

```
<_-NS_-TEXTS><item>ONE</item><item>TWO</item></_-NS_-TEXTS>
```

with `escapeRecursiveXrfcTag("/NS/TEXTS") === "_-NS_-TEXTS"` (test/recursive-xrfc.test.ts:548) and `decodeRecursiveXrfcParameterName(encoded) === "/NS/TEXTS"` (test/recursive-xrfc.test.ts:553).

From `"preserves negative zero in FLOAT scalar tables and validates wrappers"`, test/recursive-xrfc.test.ts:993-1001 - input `[-0, { "": 1.5 }]`:

```
<FLOATS><item>-0</item><item>1.5</item></FLOATS>
```

and via the invocation envelope, input `{ FLOAT_TABLES: [-0] }` produces the `CpicTag.XRfcData` field `<FLOAT_TABLES><item>-0</item></FLOAT_TABLES>` (test/recursive-xrfc.test.ts:1068-1071).

From `"canonicalizes blank xRFC DATE/TIME inputs to empty elements"`, test/recursive-xrfc.test.ts:1166-1183 - both `{ DATE: "", TIME: "" }` and `{ DATE: "        ", TIME: "      " }` produce:

```
<TEMPORAL><DATE></DATE><TIME></TIME></TEMPORAL>
```

and both decode back to `{ DATE: "", TIME: "" }`.

From `"pads short fixed BYTE values and rejects only oversized values"`, test/recursive-xrfc.test.ts:1196-1212 - input `{ BYTE: Buffer.from("aa", "hex") }` into a 2-byte fixed field:

```
<BYTES><BYTE>qgA=</BYTE><MARKER></MARKER></BYTES>
```

and decoding `<BYTES><BYTE>qg==</BYTE><MARKER></MARKER></BYTES>` yields `{ BYTE: Buffer.from("aa00", "hex"), MARKER: "" }`.

From `"uses canonical xRFC entities for XML punctuation and control characters"`, test/recursive-xrfc.test.ts:1232-1249 - the input text is NUL, U+0001, tab, LF, CR, `<`, `&`, `>` and the output is `<VALUE><TEXT>` followed by `&#00;` `&#01;` then a literal tab, LF and CR, then `&#60;&#38;&#62;` and `</TEXT></VALUE>`. Decoding `<VALUE><TEXT>&#34;quoted&#39;</TEXT></VALUE>` yields the text `"quoted'`.

Entity-grammar acceptance set (test/recursive-xrfc.test.ts:1276-1302): the five named entities, and for each of the code points `0, 1, 0x1f, 0x20, 0x41, 0x7f, 0x80, 0xff, 0x100, 0xd7ff, 0xe000, 0xfffd, 0x10000, 0x1f600, 0x10ffff` all four spellings `&#N;`, `&#0000000N;` (7-digit zero-padded), `&#xh;` (lowercase), `&#X000HHH;` as 6-digit uppercase padded hex. Rejection set at test/recursive-xrfc.test.ts:1318-1332.

### Recursive classic xRFC XML (strict codec, UTF-8 text)

From `"encodes and decodes the bounded recursive classic xRFC subset exactly"`, test/recursive-classic-xrfc.test.ts:597-612; input `VALUE` at test/recursive-classic-xrfc.test.ts:400-418; expected `EXPECTED_XML` at test/recursive-classic-xrfc.test.ts:420-426:

```
<INPUT><ID>7</ID><CHILD><TEXT>A&#38;B</TEXT><LABEL>Grüße 🌍</LABEL></CHILD><ROWS><item><COUNT>1</COUNT><DETAIL><TEXT>ONE</TEXT><LABEL>first</LABEL></DETAIL><CHUNKS><item><DATA>3q2+7w==</DATA></item></CHUNKS></item><item><COUNT>-2</COUNT><DETAIL><TEXT>TWO</TEXT><LABEL></LABEL></DETAIL><CHUNKS></CHUNKS></item></ROWS><BLOB>AAEC</BLOB></INPUT>
```

From `"constructs recursive ABAP initial values without choosing basXML"`, test/recursive-classic-xrfc.test.ts:614-629 - encoding the empty object `{}` against the same plan:

```
<INPUT><ID>0</ID><CHILD><TEXT></TEXT><LABEL></LABEL></CHILD><ROWS></ROWS><BLOB></BLOB></INPUT>
```

with `initialRecursiveClassicXrfcValue(resolved)` equal to `{ ID: 0, CHILD: { TEXT: "", LABEL: "" }, ROWS: [], BLOB: Buffer.alloc(0) }`.

From `"uses encoded base64 cell limits symmetrically"`, test/recursive-classic-xrfc.test.ts:1419-1429 - decoding this document under `{ maxCellBytes: 4 }` must fail:

```
<INPUT><ID>0</ID><CHILD><TEXT></TEXT><LABEL></LABEL></CHILD><ROWS></ROWS><BLOB>AQIDBA==</BLOB></INPUT>
```

### DPD exhaustive vectors (generated, not literal)

Two exhaustive properties are better expressed as generators than as stored vectors:

- for every integer `0..999`, the low ten bits of `encodeDecimalFloat16(String(value))` equal Cowlishaw's Boolean-equation encoding (test/decimal-float.test.ts:140-150); the oracle implementation is at test/decimal-float.test.ts:31-60;
- for every declet code `0..1023`, `decodeDecimalFloat16` of a DECF16 word carrying that code decodes to the oracle value, and exactly `24` codes are redundant encodings (test/decimal-float.test.ts:152-162); the oracle is at test/decimal-float.test.ts:62-92.

Porting the two oracles is the highest-value single item in this list: they verify the DPD tables independently of the implementation.

### Not covered by any in-scope vector

- **Packed decimal (`TYPE P`) byte layout.** No in-scope test asserts packed bytes. The layout is only stated by src/values/packed-decimal.ts:122-129 (digits zero-padded to `byteLength*2-1` nibbles, then a `C` or `D` sign nibble, packed two nibbles per byte, most significant first). A conformance vector set for this should be authored from the source, not derived from a test.
- **Classic fixed-structure byte layout.** src/values/classic-structure.ts has no in-scope test file; its UTF-16LE field placement, alignment fill (src/values/classic-structure.ts:576-592) and integer widths are unverified by any vector in scope.

---

## Open questions for the porter

1. **`DEFAULT_MAX_CPIC_FIELD_LENGTH`, `DEFAULT_MAX_CPIC_FIELD_CHAIN_LENGTH`, `DEFAULT_MAX_CPIC_FIELD_COUNT` are outside this inventory's scope.** They are imported at src/values/classic-xrfc.ts:6-9, src/values/classic-structure.ts:10 and src/values/recursive-xrfc.ts:16-19 and become the default and ceiling for `maxCellBytes`, `maxRowBytes`, `maxParameterBytes`, `maxRows` and `MAX_CLASSIC_STRUCTURE_BYTE_LENGTH`. Their numeric values are **not stated in source within scope**; test/classic-xrfc-protocol.test.ts:37 and :59 name a "16 KiB request chunk" and a "16,384-byte boundary" but that is the chunking boundary, not these limits. Resolve before porting any limit test.

2. **Signed vs unsigned compact-temporal raw reads.** src/values/classic-temporal.ts:431-435 reads the raw with `readBigInt64LE` / `readInt32LE` / `readInt16LE` (**signed**), then rejects `raw < 0n`. src/values/recursive-xrfc.ts:1036-1041 assembles the same bytes **unsigned** (`raw = (raw << 8n) | BigInt(encoded[index])`) and stringifies that for the XML wire, and src/values/recursive-xrfc.ts:1579 bounds the decode at `1n << BigInt(width * 8)` (unsigned). For any raw with the high bit set, the two paths disagree: the binary codec errors, the xRFC codec produces a large positive decimal. No in-scope test covers this. Confirm which is intended before choosing `int64` or `uint64` in Go.

3. **`JSON.stringify` in the graph digest.** src/values/recursive-serializer-classification.ts:157 hashes `JSON.stringify(graphProjection(graph))`. Go's `encoding/json` will not produce the same bytes by default (HTML escaping, number formatting). Decide whether the digest is a cross-implementation contract (then reproduce `JSON.stringify` exactly) or implementation-local (then document that Go and TS digests are incomparable and that `"graph-mismatch"` only guards within one runtime).

4. **`localeCompare` in the same projection.** src/values/recursive-serializer-classification.ts:142 sorts node entries with `left.localeCompare(right)`. That is locale-sensitive and differs from byte order for non-ASCII type names, which would change the digest under a different ICU locale even within Node. Is byte order acceptable?

5. **JS `Number#toString` reproduction.** Four sites depend on it for wire output: src/values/packed-decimal.ts:14, src/values/decimal-float.ts:172, src/values/classic-xrfc.ts:448, src/values/recursive-xrfc.ts:1103. Decide once whether the Go port reproduces ECMAScript `Number::toString` exactly, or whether the Go API refuses `float64` input for BCD/packed fields and accepts only decimal strings. The second option is far safer and only affects the FLOAT (`F`) EXID, which genuinely is a binary64.

6. **Is `rfc-value-snapshot.ts` in the port at all?** Its entire threat model (proxies, getters, prototype chains, `WeakSet` identity) is JavaScript-specific. Confirm whether the Go port keeps only the cycle detection and the node/byte budgets, and record the dropped tests in provenance.

7. **`test/bytes.test.ts` is in the stated scope but targets `src/protocol/bytes.ts`.** It exercises `CheckedByteReader` / `CheckedByteWriter` (test/bytes.test.ts:4), including little-endian signed INT4 reads (test/bytes.test.ts:34-42) and the error text `/APPC header\.uid.*2 bytes.*offset 1.*1 remain/` (test/bytes.test.ts:28-31). Those primitives underpin every codec here but belong to a different surface inventory. Confirm whether `src/protocol/` gets its own document.

8. **`decodePackedDecimal` accepts four positive sign nibbles (`A`, `C`, `E`, `F`) but the encoder writes only `C`.** src/values/packed-decimal.ts:124 vs :148. Is the read-side tolerance deliberate SAP compatibility, and must the Go port keep it? Same question for `B` alongside `D` (:149).

9. **`DAYS_BY_MONTH` stores February as `29`.** src/values/classic-temporal.ts:115-117. It is corrected per-year in `daysInMonth` (src/values/classic-temporal.ts:204) and in `parseDateParts` (src/values/classic-temporal.ts:223-225), but `CDAY` (`c`) indexes it directly (src/values/classic-temporal.ts:589-595, 653-655), making the CDAY calendar a fixed 366-day year. Confirm that is intended before porting, because a Go port that reuses one shared month table for both paths will silently change CDAY ordinals.

10. **`x` (DTMONTH) and `c` (CDAY) do not add the `+ 1n` bias.** src/values/classic-temporal.ts:545 and :596, versus every other EXID at :469, :484, :499, :510, :526, :559, :573. The decoder still subtracts one uniformly at src/values/classic-temporal.ts:611. The reference vectors (test/classic-temporal.test.ts:50, 52) confirm the asymmetry is real, but it is easy to "fix" during a port. Do not.
