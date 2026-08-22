# Documentation

| Document | What it answers |
|---|---|
| [Cheat sheet](cheatsheet.md) | Ports, auth, constants, layer map, commands — one page |
| [Protocol primer](protocol-primer.md) | What classic RFC does on the wire, layer by layer |
| [Glossary](glossary.md) | What NI, APPC, CPIC, xRFC, LUW, `sysnr` mean |
| [Architecture](architecture.md) | Why the code is shaped this way; ownership invariants |
| [Role state machines](role-state-machines.md) | Who may send what, when — client and each server role, and the keepalive rule |
| [Recurring bug class](recurring-bug-class.md) | The mistake to avoid — **required before touching a decoder** |
| [Development](development.md) | How to build, test, fuzz, and run one test |
| [Live test plan](live-test-plan.md) | You have a real SAP system and want to verify against it |
| [Porting plan](porting-plan.md) | What is next, and why in that order |
| [Provenance](provenance.md) | Which Go file came from which upstream file, and what changed |
| [Surface inventories](surface/) | Cited signatures and constants for the unported layers |

## Reading orders

**"I want to understand SAP RFC."**
[Protocol primer](protocol-primer.md) → [Glossary](glossary.md) →
[Cheat sheet](cheatsheet.md)

**"I want to contribute code."**
[Recurring bug class](recurring-bug-class.md) → [Architecture](architecture.md) →
[Development](development.md) → [`../AGENTS.md`](../AGENTS.md)

**"I want to port the next layer."**
[Porting plan](porting-plan.md) → the relevant [surface inventory](surface/) →
[cross-layer answers](surface/cross-layer-answers.md) → the upstream file itself

**"I have an SAP system to test against."**
[Live test plan](live-test-plan.md) -> [Cheat sheet](cheatsheet.md#authentication) for what auth
is supported -> [Protocol primer](protocol-primer.md)

**"I want to know if I can use this."**
[`../README.md`](../README.md#status) → [`../SECURITY.md`](../SECURITY.md). Short
answer: not yet.

## About the surface inventories

[`surface/`](surface/) holds ~7,800 lines of mechanical inventory of the six
unported upstream layers: exported signatures, constant values, error messages,
comment-stated invariants, and the wire facts the tests assert — each with a
`path:line` citation into the upstream checkout.

They exist so that porting a layer starts from a map rather than from a blank
file. They are **input, not truth.** They were produced by reading upstream at
commit `847036d`, they can be stale, and one inventory's own verification pass
corrected four counts it had asserted from `grep` output rather than from
reading. Nothing is ported without opening the upstream file.

[`surface/cross-layer-answers.md`](surface/cross-layer-answers.md) is the
exception worth reading on its own: it collects the questions each inventory
could not answer within its scope, and three of its answers change how the port
must be written.
