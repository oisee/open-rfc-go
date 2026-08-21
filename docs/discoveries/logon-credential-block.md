# What the logon credential block actually varies

Captured 2026-08-21 against A4H (SAP_BASIS 758) with `cmd/rfc-lab` between SM59
and the gateway: nine connection tests of one type-3 destination, in three
groups of three — a short user with a password, the same destination with
*Current User* and no password, and a long user with a password.

## The finding

Of the 48 fields in the initial logon, exactly two ever change length:

| Field | user + password | *Current User* | longer user + password |
|---|---|---|---|
| `0x0111` user | 12 | 12 | 24 |
| `0x0117` password | 28 | **20** | 28 |

Everything else is identical, field for field.

Two things follow.

**The user field is UTF-16.** `CLAUDE` is six characters and twelve bytes;
`AVINOGRADOVA` is twelve and twenty-four. Our encoder writes ASCII there and the
system has never complained, which says the field is tolerant, not that it is
ASCII.

**An absent password is a 20-byte field, not an empty one.** With a password the
scrambled block is 28 bytes; with *Current User* it is 20. So the block has a
fixed header of its own and 8 bytes of payload for a password of this length —
and "no password" is expressed by sending the header alone, not by omitting the
field. Anything we build that logs on without a password has to produce those
20 bytes; our scrambler cannot currently express it.

## What the experiment was for, and did not find

The point was to capture an SAP logon ticket on the wire, so ticket-based RFC
logon could be implemented — a real need on landscapes where a browser
single sign-on is the only way a human gets a credential.

**No ticket appeared.** *Current User* on a type-3 destination sends the calling
user's name with the empty-password block, and nothing else: no new tag, and a
frame eight bytes shorter rather than seven hundred longer. That mechanism is
password-less logon for a trust relationship, not ticket forwarding. The
*Send Assertion Ticket* control that would forward one belongs to HTTP
destinations; a type-3 destination has no such option on this release.

So the field that carries `MYSAPSSO2` remains unobserved. Producing one needs a
client that sets the parameter itself — the SAP libraries do, and we have
none — or a scenario we have not found. Parked, and not blocking: the SOAP RFC
endpoint already gives a cookie-authenticated caller the whole remote-enabled
function surface over HTTP.
