# Choosing the serializer, and what each one puts on the wire

Live A/B, 2026-08-22, against A4H (SAP_BASIS 758, kernel 793) on our own LAN. One
SM59 **type 3** destination pointed at our relay sniffer and forwarded to the real
system, so both ends are genuine ABAP and we only observe. Driver: one function
module calling the same eight FMs in the same order every time —
`Z_DOUBLE`, `Z_GREET`, `RFC_PING`, `RFC_SYSTEM_INFO`, `STFC_CONNECTION`,
`STFC_STRUCTURE` (three populated `RFCTEST` rows), `RFC_READ_TABLE` (`T000`),
`STFC_STRING`. Every call returned `rc=0` in every mode.

**Provenance.** Everything here was established two ways only: by ticking a box or
picking a dropdown entry in SM59 and reading back `RFCDES` (a customer table, read
over RFC with `RFC_READ_TABLE`), and by capturing the resulting traffic with this
repo's own sniffer. No SAP source, SDK or documentation was used to derive any
statement below. Where a fact could not be reached that way it is listed as open.

## The destination decides — through two independent knobs

`RFCDES-RFCOPTIONS` is a comma-separated list of `<char>=<value>` entries. Two of
them govern serialization. Setting the SM59 dropdown and reading the table back
gives the codes directly:

| SM59 "Serializer" | stored as | wire |
|---|---|---|
| Classic serializer | `1=00`, no `7=` | **classic** |
| Fast serializer | `1=00` **and** `7=6041` | **fast** |
| basXML serializer | `1=11` | **classic** — it negotiates down |
| Force basXML serializer | `1=21` | **basXML, binary** |

Classic and Fast store the *same* `1=` value. What separates them is whether the
dropdown also writes `7=6041`. Confirmed directly — one destination, `1=00`
throughout, one token added:

```
1=00, 7= absent  ->  classic   16 738 B, entropy 3.48
1=00, 7=6041     ->  fast       9 636 B, 17 \TYPE= descriptors
```

SM59 omits a token entirely when its value is the default, so an absent `o=` means
a zero flag mask and an absent `7=` means no override.

`1=20` and `1=22` — values not offered by the dropdown — put **textual** basXML on
the wire. Any first digit other than `2` yields classic.

The second knob overrides the first:

| `7=` | `1=` | wire | bytes | entropy |
|---|---|---|---|---|
| `6041` | **anything** | **fast** | ~9 810 | 5.23 |
| empty | `20`, `22` | basXML, textual | 13 685 | 5.60 |
| empty | `21` | basXML, binary | 11 030 | 6.19 |
| empty | anything else | classic | 17 603 | 3.47 |

The A/B that establishes it — same destination, one token apart:

```
7=6041 + 1=20  ->  fast      17 \TYPE= descriptors, 9 812 B
7= empty + 1=20 -> XML text  295 tags,             13 686 B
```

So a destination showing "Fast serializer" while storing `1=00` is not a
contradiction: `7=` is doing the work. **Fast never comes from the dropdown.**

Nothing is compressed in any mode — entropy stays far below the ~7.9 of compressed
data, so every payload decodes directly.

## The flag mask

The SM59 *Mask for Special RFC Flags* dialog shows its own value as "Current Bit
Option in Hex", so each checkbox's bit is readable by ticking it alone:

| bit | checkbox |
|---|---|
| `0x01` | Deactivate delta manager |
| `0x02` | Activate RFC Trace Statistics |
| `0x04` | Deactivate RFC Compression |
| `0x10` | Activate FastRFC |
| `0x20` | Preserve connection to `/n` in target system |
| `0x40` | Activate Runtime Analysis in Target System |
| `0x80` | Activate Delta Manager Trace |
| `0x100` | Apply fixed language value to target system |
| `0x200` | Use Found Communication Code Page |
| `0x1000` | Prevent Execution of `/n` in Target System |

`0x08` is not used by any checkbox. The mask is stored as the `o=` entry.

Measured, one full driver run per value:

- **`0x10` "Activate FastRFC" does not select fast serialization.** With it set and
  `7=` empty the wire stayed classic. `7=` is the only switch that reaches fast.
- **`0x04` "Deactivate RFC Compression" changes nothing** — 17 603 B against
  17 604 B for the baseline. Compression was not active to begin with, which agrees
  with the entropy figures above.
- **`0x02` "Activate RFC Trace Statistics"** leaves the wire format untouched, as a
  statistics flag should.
- **Delta manager on** (clearing `0x01`) shrinks the conversation from 17 604 B to
  16 737 B, so the mechanism is doing something. No `MULTIREF` or `RFCTAB40` name
  appears in the frames — but that is a search by name, and absence of the name is
  not absence of the mechanism.

## What each format looks like

**Fast** is tagged and self-describing, and **the tag decides how its value is
framed** — there is no single length rule. Established by a controlled
differential: one caller parameter varied at a time, everything else held fixed.

| tag | | framing |
|---|---|---|
| `0x43` | `'C'` char | `<len:1> 0x80 <len bytes>` — single-byte, **not** UTF-16 |
| `0x4e` | `'N'` int4 | `<4 bytes>` little-endian, no length |
| `0x50` | `'P'` descriptor | `<len:1> <len bytes>` — the `\TYPE=` announcement |
| `0x30` | `'0'` padded text | `<len:2 BE> <len bytes>` — padded UTF-16LE |
| `0x45` | `'E'` end | no value |

Two whole parameters, captured verbatim:

```
Z_DOUBLE, N=256   50 07 "\TYPE=I"       03 0a "TABLE_LINE"  4e 00010000        45
Z_GREET, NAME=ABCD 50 0c "\TYPE=CHAR30"  06 3c 00  0a "TABLE_LINE"  43 04 80 "ABCD" 45
```

The evidence behind each claim:

- **Little-endian.** `N=1 -> 4e 01000000`, `N=2 -> 4e 02000000`, `N=256 -> 4e
  00010000`. 256 is the value that settles it; the frame length never moved, so
  the width is fixed at four.
- **One byte per character.** The `Z_GREET` frame measured 386, 387, 389 and 393
  bytes for names of 1, 2, 4 and 8 characters — exactly one byte each. That rules
  out UTF-16 for char values, and rules out padding to the declared width: the
  parameter is `CHAR30` and only the significant bytes travel.
- **The `0x80` flag is real.** It sits between a character field's length and its
  value. Its meaning is unknown; skipping it is required to read the value at all,
  and a decoder that treats the field as `<tag><len><value>` is wrong by one byte
  for every character field on the wire.

Composite types are referenced by **DDIC type name** rather than by an inline
layout — observed: `I`, `CHAR30`, `CHAR90`, `STRING`, `RFCSI`, `RFCTEST`,
`SYST_LISEL`, `%_T00004S0`. So a decoder is: parse the framing, resolve the named
type through DDIC, apply the ordinary structure codec.

### What is not modelled, and why it is left alone

The bytes between a `\TYPE=` descriptor and the field name are type metadata of a
shape we have not pinned down: `03` for `I`, `18` for `STRING`, `06 3c 00` for
`CHAR30`, `13 86 00` for `RFCTEST`. Reading the last two as a little-endian width
fits `CHAR30` at 60 = 30x2 and does **not** fit `RFCTEST`, so it is recorded here
and not implemented.

One negative result is worth keeping, because it is a trap. Recovering from an
unmodelled region by stepping one byte at a time does not work here: the tags are
ASCII letters, so `'E'` (end) and `'N'` (int4) occur inside ordinary field names
like `TABLE_LINE`. A skipping decoder reads those as records, drifts, and swallows
the real value that follows. Phantom records are worse than parsing less. The
decoder therefore parses strictly and stops, and finds values by anchoring on the
`\TYPE=` signature — which carries its own length and so cannot be faked by
accident — then requiring corroboration before accepting a value: a character
record must have its flag byte, an integer must be followed by the end marker.

The `0x5001` container's nesting is also still open.

Its version is negotiated in the clear. SM59's *Fast Serialization Test* button
queries three names and the answer carries the value:

```
0205 <len> <name>          in the request  (name only)
0201 <len> <name>          in the response
0203 <len> <value>
0201 001a "FAST_SER_VERS"  0203 0004 03000000   -> FAST_SER_VERS = 3
```

Three consecutive runs were byte-identical, so this handshake is deterministic.

**Textual basXML is the semantic oracle.** `1=20` puts ZBXML on the wire
(`ZBXML VER 0.7 ENC utf-8`, namespace `http://www.sap.com/abapxml`), naming every
field and the table line count:

```
values
  ECHOSTRUCT
    RFCFLOAT T 0 · RFCCHAR1 T A · RFCINT2 T 0 · RFCINT1 T 0 · RFCCHAR4
    RFCINT4 T 0 · RFCHEX3 · RFCCHAR2 · RFCTIME T 00:00:00 · RFCDATE T 0000-00-00
    RFCDATA1 T struct via ZCL_RFC_TEST · RFCDATA2
  RESPTEXT T@SAP R/3 Rel. 758   Sysid: A4H ...
  RFCTABLE
    lines @ A 4
    RFCTEST  T 0 · T 1 · T 100 · T 1 · T ROWS · T 10000 · T RW · T row 3 ...
```

Because the driver, the call order and the inputs are identical across modes, the
field names, order and values are known for every byte the fast serializer emits.
That is what makes the fast format an alignment exercise rather than a guess.

Every mode keeps the function module name in plaintext UTF-16LE at the frame layer,
so a capture is self-labelling whichever serializer is in force.

## Open

- Why `1=11` ("basXML serializer") negotiates down to classic between two systems
  that both support basXML.
- Why `1=11` ("basXML serializer") negotiates down to classic between two systems
  that both support basXML.

## The delta manager, caught working

Sending the same table three times on one conversation gives it something to
elide. The driver builds three `RFCTEST` rows, restores a pristine copy before
each call, and calls `STFC_STRUCTURE` three times, so the content crossing the
wire is byte-identical every time. Classic serializer, A/B on mask bit `0x01`:

| delta manager | request | response | rows in response |
|---|---|---|---|
| active | 1 557 B, 3 rows | **1 434 B** | **0** |
| deactivated (`0x01`) | 1 553 B, 3 rows | **2 242 B** | 3 |

808 bytes a call, exactly the table content. The caller still sees `rows=4` either
way, so nothing is lost — the table is reconstructed.

Two things follow.

**It is a response-side mechanism.** The request always carries the full table; it
is the *server* that elides. Anything we send as a client is unaffected.

**The delta is taken against the request, not against a previous call.** Elision
happens on call #1 already. `STFC_STRUCTURE` echoes its input table back, and with
the manager active the server declines to return data the caller had just handed
it.

For the server track this is a matched pair: one request with two legal responses.
The 2 242-byte form is the degenerate full encoding we would emit; the 1 434-byte
form is what we must be able to read. Which is exactly the asymmetry below.

## What the delta manager means for our server — decode and encode are not symmetric

Reading a delta and writing one are different obligations, and conflating them is
what made the delta manager look like a blocker for the server track.

**Decoding, we have no choice.** A peer may send a real delta, so a client has to
understand the general form. That is a decoder problem, and decoding is the side we
already do well.

**Encoding, we have every choice.** Nothing requires a server to produce an
*efficient* delta. A delta encoding always has a degenerate form — "here is the
whole table" — because that is what the mechanism must emit the first time a table
crosses a connection, when there is nothing to diff against. So a deliberately
dumb server can always answer with the full table and remain correct. It costs
bandwidth and nothing else.

This matters more than it sounds, because of where the evidence comes from: the
first transmission of any table in our captures **is** that degenerate form. We do
not need to observe, understand, or reproduce a real delta in order to write the
encoder — we already hold worked examples of exactly the bytes we would emit.

So the delta manager comes off the critical path for the server:

- it is a decode requirement, handled where decoding is handled;
- it is *not* an encode requirement, because the trivial encoding is legal;
- and the client-side flag that disables it (`0x01` in the mask) is set on the
  *caller's* destination, so we cannot depend on it being off — which is precisely
  why the degenerate-encoding argument is the one that carries.

The same reasoning applies to any negotiated efficiency feature we meet later:
implement the decoder for the general case, emit the simplest legal form, and
optimise only if something actually requires it.
