# open-rfc-go v0.2.0 — the fast serializer, decoded

SAP classic synchronous RFC in pure Go — client **and** server. No SAP NW RFC
SDK, no native library, no cgo.

This release is about one thing: **SAP's fast RFC serialization is no longer
opaque.** A day ago it was a wall of bytes with a working guess or two. It is now
a grammar, implemented in `internal/fastser`, with tests that run against bytes
captured from a live system.

Still a research preview — `0.x` means the API may move, and classic RFC has no
transport encryption.

## The record grammar

Tag-dependent. There is no single length rule, which is why earlier attempts to
find one kept almost working:

```
0x43 'C' char        <len:1> 0x80 <len bytes>   one byte per char, not UTF-16
0x4e 'N' int4        <4 bytes> little-endian    fixed width, no length byte
0x50 'P' descriptor  <len:1> "\TYPE=..."
0x30 '0' padded      <len:2 BE> <value>
0x53 'S' STRING      <0xC000|len:LE16> <len:LE16> <value>
0x45 'E' end         no value
```

## Field lists, and DDIC-verified type codes

Behind a `\TYPE=` descriptor is a field-description list, not more records:

```
0x44 'D' <fieldcount:1>  0x50 'P' <len> "\TYPE=..."
then fieldcount times:   <typecode:1> [<width:2 LE>] <namelen:1> <NAME>
```

The codes are checked field for field against the live system's DDIC, using
`RFCTEST` because it carries a spread of them at once:

| | | | |
|---|---|---|---|
| `0x01` INT1 | `0x02` INT2 | `0x03` INT4 | `0x06` CHAR |
| `0x0c` DATS | `0x0e` TIMS | `0x13` FLTP | `0x17` RAW |
| `0x18` STRING | `0x19` XSTRING | | |

**`CHAR` widths count UTF-16 units; `RAW` counts bytes.** `CHAR(50)` travels as
100, `RAW(3)` as 3 — both in that one structure. A decoder that doubles
everything is wrong by a factor of two on every hex field, silently.

The field count is a real checksum: across 434 descriptors the recovered list
length matches exactly, and every mismatch is in a compressed frame.

## `0x5001` was never a container

It is one id of a general item grammar:

```
<id:2 BE> <len:2 BE> <data> <id:2 BE>
```

The id repeats as a closing tag. This also explains a puzzle that stalled an
earlier pass: searching a frame for `0x5001` finds every item **twice**, and
reading a closing tag as an opening one takes the next item's id for a length.

## The compression is LZ4

The published block format — nibble token, 255-chains, two-byte little-endian
offset, four-byte minimum match, literals-only final sequence. It engages above
512 bytes of payload and is intrinsic to the serializer: SM59's *Deactivate RFC
Compression* does nothing to it.

Blocks are located by eight bytes immediately before them carrying both sizes:

```
<uncompressed size:4 LE> <compressed size:4 LE> <block>
```

Every compressed block in the captures decodes to exactly its declared
uncompressed size while consuming exactly its declared compressed size. This
supersedes the earlier note that decompression would be a substantial separate
project — and it is what unlocked the type table above, since structures never
travel below the threshold here.

## How it was found — and four readings that were wrong

**Controlled differentials.** Vary one parameter, hold everything else, capture
both ends. Byte archaeology on large frames had failed at exactly the questions
this answered in minutes.

**Then adversarial review.** Independent agents were assigned to *refute* each
claim against the whole corpus rather than to confirm it. Every wrong reading fit
the samples that had been examined; each was killed by a frame nobody had opened.
Four are written up so they are not rediscovered:

1. **Byte-stepping resynchronisation.** The tags are ASCII letters, so `'E'` and
   `'N'` occur inside ordinary field names like `TABLE_LINE`. A skipping decoder
   reads them as records and drifts.
2. **`0x03` as a "name tag".** It is the type code for INT4.
3. **A width reading "refuted" by a broken probe.** Passing one literal to
   parameters declared `CHAR50` and `CHAR210` gave byte-identical frames — but
   neither declaration ever reached the wire; the serializer had replaced both
   with a generated type sized to the value. A controlled experiment on an
   uncontrolled variable.
4. **Guessing a block's length.** LZ4's final sequence is literals-only, so a
   block truncated early still "ends cleanly" and several wrong lengths look
   valid.

## Scope

Decode only. We do not yet produce fast serialization, and the client negotiates
classic, so none of this sits on the client's critical path today. It matters for
the server role, where an ABAP client calls us and chooses.

## Credit

A Go port of **[`open-rfc`](https://github.com/marianfoo/open-rfc)** by
[Marian Zeis](https://github.com/marianfoo), Apache-2.0. The fast serialization
codec is not part of that port — it is clean-room work from our own captures, as
[`docs/provenance.md`](docs/provenance.md) records file by file.
