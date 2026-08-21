# What an SM59 type-H destination actually sends

Captured 2026-08-21 with a plain HTTP catcher on our own network, against an
SM59 *HTTP Connection to ABAP System* destination pointed at it. Four logon
modes, one connection test each, then three repeats of the last two. The
system is A4H, SAP_BASIS 758.

This was a side experiment in a hunt for how an SAP logon ticket travels over
**classic RFC**. It did not answer that — but it settled what the HTTP path
does, which is worth its own note because none of it is published either.

## Every request carries these, regardless of mode

| Header | Content |
|---|---|
| `Sap-Passport` | 460 bytes, the distributed-trace passport — and it carries the **calling user's name** in clear text |
| `Sap-Dtrace` | `v=1,i=RFC 793 <id>,r=<id>,c=<id>,s=1` |
| `User-Agent` | `SAP NetWeaver Application Server (1.0;758)` — release included |

Worth knowing on its own: a destination that sends *no* credential still tells
the receiver who is calling, and which release is calling.

## What each mode adds

**Do Not Use a User, no ticket** — nothing at all beyond the three above.

**Basic Authentication** — `Authorization: Basic …` over whatever transport the
destination is configured for. On a destination with SSL inactive that is a
password in reversible encoding over plain HTTP, and SM59 will not stop you.

**Send assertion for dedicated target system** — a header, not a cookie:

```
MYSAPSSO2: AjQxMDM…
```

500 base64 characters, and the same shape as the ticket a browser logon puts in
the `MYSAPSSO2` cookie: version `02`, code page `4103`, the user in UTF-16, the
client, the system id and a creation timestamp. The assertion variant is bound
to the target named in the destination (System ID and Client), hence "for
dedicated target system". Same format, different carrier — so a ticket obtained
either way is usable against the HTTP surfaces.

**SAP RFC Logon (with a ticket configured)** — both `MYSAPSSO2` and one more:

```
SAP-R3AUTH: 646A3078…   (2064 hex characters)
```

That decodes in three layers: hex → base64 → the ASCII string `v=1U,` followed
by 768 hex characters, i.e. **384 opaque bytes**.

Those 384 bytes are not a field chain and not readable. Measured entropy is
7.45 bits per byte over 200 distinct values, and **all three captures differ from
the first byte** — so the envelope is encrypted and carries a nonce or a
timestamp. 384 bytes is exactly a 3072-bit block, which is consistent with the
credentials being sealed with the target system's key from its PSE.

## The ticket itself decodes — and that is useful on its own

Both the browser cookie and the assertion header parse cleanly with our own
code: a 5-byte header (`02` version, `4103` code page) followed by a TLV run,
`id(1) len(2 BE) value`. The fields, confirmed against two real tickets:

| id | meaning | form |
|---|---|---|
| `0x01` | user | UTF-16 |
| `0x02` | client | UTF-16 digits |
| `0x03` | issuing system | UTF-16 |
| `0x04` | creation timestamp | UTF-16 `YYYYMMDDhhmmss` |
| `0x0f` | (portal codepage marker) | |
| `0xff` | signature | **PKCS#7 SignedData** (`06 09 2a 86 48 86 f7 0d 01 07 02`) |

The assertion ticket differs from the browser one exactly where it should: it
adds `0x10` = the **recipient system** (`A4H`) and `0x08`, and drops the
browser's `0x06` portal flag. So an assertion ticket is *addressed* — the target
is inside the payload, which is what "for dedicated target system" means — while
the browser ticket is general. Same envelope, one extra field.

The practical consequence: because we can read the envelope, we can tell whose
ticket we hold, for which system, and whether it has expired, before making any
network call — a clear error instead of a rejected logon. That is worth a small
reader on its own, independent of whether ticket-based *RFC* logon ever lands.

## The conclusion for the ticket hunt

Nothing here reveals how a ticket rides classic RFC. The HTTP path either sends
the ticket as a header in the clear, or seals the credential into an opaque
block — neither tells us the CPIC field. What it does confirm is the ticket
*format* is one format across carriers, and that our HTTP routes (ADT, and the
SOAP RFC endpoint) can be driven by a ticket alone, which was verified
separately the same day.
