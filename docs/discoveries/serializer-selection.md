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

### Scalar widths, and the compression threshold

A purpose-built probe (`Z_FS_PROBE`, one type and one length per call) drove each
type over the fast destination with everything else held fixed. It calls
`RFC_PING` first: without that warm-up the measured call is the first on the
connection and therefore rides in the logon frame, whose session tokens jitter run
to run and bury a one-byte difference.

| type | cost |
|---|---|
| `STRING` | **1.00 byte per character** |
| `XSTRING` | **1.00 byte per byte** |
| `STRING` nested in a deep structure | 1.00 byte per character, plus ~54 constant |

So `STRING` and `XSTRING` travel one byte per unit, like `char` — not UTF-16, and
not padded to any declared width.

**Above 512 bytes the payload is compressed.** Bisected on `STFC_STRING`: a
512-character argument is literal and the frame has grown exactly one byte per
character all the way up; 513 characters collapse the frame from 919 bytes to 448,
leaving a 26-byte literal remnant. A 3 000-character argument then costs 458.

| characters | frame | longest literal run |
|---|---|---|
| 510 | 917 | 512 |
| 511 | 918 | 513 |
| **512** | **919** | **514** |
| **513** | **448** | **26** |
| 3 000 | 458 | 26 |

The scheme is LZ-like — one literal copy of the repeating input survives, the rest
becomes back-references — and it covers the **whole parameter block, field names
included**. That retro-explains an earlier puzzle: in a large `STFC_STRUCTURE`
frame only 4 of `RFCTEST`'s 12 field names appear literally, and the others show
up as fragments like `HEX3`, `TIME`, `DATA2`. Those are not an elided-prefix naming
scheme, as first guessed; the frame was simply over the threshold.

The trigger is **size, not structure**. Tables looked like they switched at three
rows, but that was only where their block happened to cross 512 bytes; a plain
string crosses it with no table involved.

The **delta manager is not part of this**. Deactivating it (mask bit `0x01`) moved
every frame in the series by a flat +6 bytes and changed nothing else — same
collapse, same slopes, same literal runs. Which fits what it does: it elides on
the response side, and these are requests.

### The STRING field, and what the compression flag is not

A `STRING` rides under tag `0x53` (`'S'`) with its length written **twice** —
once with `0xC000` set, then plain — and then one byte per character:

```
'P' 0c "\TYPE=STRING"  18  0a "TABLE_LINE"  'S' 08c0 0800 "ABCDEFGH"
```

The two copies corroborate each other, which is enough to recognise a STRING
without a surrounding anchor: arbitrary bytes are very unlikely to satisfy
`first == 0xC000 | second`.

`0xC000` is **not** a compression flag. It is set at 510 characters (literal) and
at 3 000 (compressed) alike. What it means is not established.

The declared length is the **original** size, not the encoded one, so above the
threshold it far exceeds the bytes actually present. The decoder therefore refuses
such a field instead of returning a short value or reading past the payload — a
compressed STRING is reported as unaccounted for, not silently mis-decoded.

SM59's **"Deactivate RFC Compression" does not switch this off.** With mask bit
`0x04` set the whole series came back byte-for-byte identical: 512 characters
literal at 919 bytes, 513 compressed at 448, 3 000 at 458. So the folding is
intrinsic to the fast serializer rather than an optional transport feature, and
decoding large fast payloads requires implementing it.

That also corrects an earlier note here. Bit `0x04` was first measured as having
no effect and written up as "compression was not active to begin with" — but that
run used a payload under 512 bytes, where nothing would have been compressed
either way. The flag was tested on a sample that could not show a difference.

We do **not** already own the decompressor. The classic path's "simple
compression" (`decodeSimpleCompressedTableRow`) expands a short row by repeating
its last byte — trailing-run fill, unrelated to the back-referencing scheme here.

### The type metadata — solved, and an earlier reading here was wrong

A parameter is announced by a header, a descriptor, and one description per field:

```
0x44 'D' <fieldcount:1>
0x50 'P' <len:1> "\TYPE=<name>"
repeated fieldcount times:
    <typecode:1> [<width:2 LE>] <namelen:1> <NAME>
```

The bytes that sat between a type and its name are **one family: single-byte ABAP
type codes.** Observed: `0x01` INT1, `0x02` INT2, `0x03` INT4, `0x06` CHAR,
`0x0c` DATS, `0x0e` TIMS, `0x13` FLTP, `0x17` RAW, `0x18` STRING, `0x19` XSTRING.
A width-parameterised code carries a two-byte operand; one whose width follows
from the code does not. The whole table is cross-checked against the live
system's DDIC using `RFCTEST`, which carries a spread of types in one
announcement:

| field | code | width on the wire | DDIC |
|---|---|---|---|
| `RFCFLOAT` | `0x13` | — | FLTP(8) |
| `RFCCHAR1` | `0x06` | 2 | CHAR(1) |
| `RFCINT2` | `0x02` | — | INT2 |
| `RFCINT1` | `0x01` | — | INT1 |
| `RFCCHAR4` | `0x06` | 8 | CHAR(4) |
| `RFCINT4` | `0x03` | — | INT4 |
| `RFCHEX3` | `0x17` | **3** | RAW(3) |
| `RFCCHAR2` | `0x06` | 4 | CHAR(2) |
| `RFCTIME` | `0x0e` | — | TIMS |
| `RFCDATE` | `0x0c` | — | DATS |
| `RFCDATA1` | `0x06` | 100 | CHAR(50) |
| `RFCDATA2` | `0x06` | 100 | CHAR(50) |

**The two width conventions differ.** CHAR counts UTF-16 units, so CHAR(50)
travels as 100; RAW counts bytes, so RAW(3) travels as 3. Both sit in that one
structure, which is what settles it — a decoder that doubles everything gets
every hex field wrong.

That table came out of a **compressed** frame. Structures never travel below the
512-byte threshold here, so none of it was readable until the LZ4 decoder existed.

Scalar parameters are a different story: on the path this probe exercises the
serializer normalises them. `NUMC`, `DATS`, `TIMS`, `FLTP`, `RAW` and `STRING`
all arrive as a generated CHAR type, and `INT1`/`INT2`/`INT4` all collapse to
`\TYPE=I` with code `0x03`. So the real codes are visible in structures, not in
scalars — which is why an earlier sweep of scalar types found none of them.

**`0x03` is not a name tag.** It was read as one here because it sat before the
field name in every `INT4` capture. It is the type code for INT4, in the same slot
as `0x01` for INT1 and `0x02` for INT2, exactly where DDIC says those fields are.

The field count is a checksum, and a strong one: over 434 descriptors the
recovered list length equals it exactly, with all 89 mismatches in frames above
512 bytes — that is, in compressed frames — and none below.

#### The width operand, and why it looked contradictory

It is the declared width of the type **the descriptor names**, in bytes, UTF-16
counted. The apparent contradiction came from not noticing what the descriptor
actually said:

```
\TYPE=CHAR30                           06 3c 00 -> 60, whatever the value's length
\TYPE=%_T00006S00000000O0000000298     06 0c 00 -> 12, and the value is "Claude"
```

The second is a type the **serializer generated for that call**, sized to the
value. So the width tracks the value there — not because the rule changed, but
because the declaration did.

This is what invalidated an earlier probe here. Passing the same literal to
parameters declared `CHAR50` and `CHAR210` produced byte-identical frames, which
was taken as proof the width could not be the declared one. In fact **neither
declaration ever reached the wire**: both calls carried
`\TYPE=%_T00006S00000000O0000000297`, so both really did serialise the same
five-character type. The literals `CHAR50` and `CHAR210` appear in no frame of any
capture. The differential varied something the serializer had already erased —
a controlled experiment on an uncontrolled variable.

The lesson generalises: **read the descriptor before trusting what the ABAP source
says a parameter is.**

### The item grammar — what `0x5001` really is

Not a bespoke container. One item of a general transport-layer grammar:

```
<id:2 BE> <len:2 BE> <data:len> <id:2 BE>
```

The identifier repeats after the data as a closing tag. `0x5001` is the id of the
item carrying a serialized parameter block; `0x0130` carries the ABAP program
name.

The closing tag is also the explanation for a long-standing puzzle. Searching a
frame for `0x5001` finds every item **twice**, once opening and once closing, and
reading a closing tag as an opening one takes the *next item's id* for a length.
That is exactly the "second occurrence gives a length that runs past the end of
the frame" that stalled an earlier pass: the unexplained 304 was `0x0130`, the id
that followed.

### The compression is LZ4

Not merely LZ77-shaped — the **standard LZ4 block format**: nibble token,
255-chains on both run lengths, two-byte little-endian match offset, four-byte
minimum match, final sequence literals-only.

Established by decoding rather than by inspection. A strict decoder written to the
published block format reproduces every compressed block across three captures,
each consuming exactly the declared compressed byte count and emitting exactly the
declared uncompressed one. There is no LZ4 frame header, no magic and no checksum;
the surrounding framing supplies both sizes.

**Where a block is** is not guessable, and does not have to be: eight bytes
immediately before it carry both sizes.

```
<uncompressed size:4 LE> <compressed size:4 LE> <block>
```

Guessing was tried and does not work — a block truncated early can still finish
on LZ4's mandatory literals-only final sequence, so several wrong lengths look
valid. With the header there is nothing to infer: across the whole corpus **155
blocks** are located this way and every one decodes to exactly its declared
uncompressed size while consuming exactly its declared compressed size, and no
other offset in any frame satisfies both.

`internal/fastser.DecompressBlock` and `DecodeCompressedAt` implement it, so
payloads above the 512-byte threshold are now readable. This supersedes the note that decompression would be a
substantial separate project: the block format is small and published.

### The resynchronisation trap

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

### The version handshake

The serializer's version is negotiated in the clear. SM59's *Fast Serialization Test* button
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
