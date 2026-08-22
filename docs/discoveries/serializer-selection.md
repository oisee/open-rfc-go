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

**Fast** is tagged and self-describing: an ASCII type descriptor and field name,
each length-prefixed, then the raw little-endian value. `Z_DOUBLE` with `N=21`:

```
44 01 50 07 "\TYPE=I" 03 0a "TABLE_LINE" 15000000 45 5001…
       len=7                    len=10     value   'E'
```

Composite types are referenced by **DDIC type name**, not by inline layout —
observed: `I`, `CHAR90`, `STRING`, `RFCSI`, `RFCTEST`, `SYST_LISEL`, `%_T00004S0`.
So a decoder is: parse the tag/length framing, resolve the named type through DDIC,
apply the ordinary structure codec.

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
- Whether the delta manager's effect has a name on the wire. With the manager
  active, classic, and three distinct rows, each row's payload still appears
  exactly once and no `MULTIREF`/`RFCTAB40` name shows up. That is expected: a
  delta manager has nothing to elide unless the *same* table crosses the wire more
  than once on one connection. The driver sends each table once, so the mechanism
  has not yet been given anything to do.
