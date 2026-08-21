# Plugging into Eclipse's MCP server — what exists, what it would buy us, and where the rules actually sit

Research note, 2026-08-21. Read-only survey of `open-rfc-go` and
`vibing-steampunk` plus public sources. **Not legal advice** — this is a map of
the terrain with citations, so a human can make the call.

---

## Verdict

Yes, SAP ships an MCP server for ABAP development — and it ships **inside
Eclipse ADT** as well as inside ADT for VS Code, off by default, since ADT
**3.60** (SAP Help, 2026‑08‑17). A third party can contribute tools into it
through the Eclipse extension point `com.sap.adt.mcp.core.adtMcpTools`; a
community plugin already ships 18 tools that way, without reflection. So the
integration the question imagines is technically real.

The compliance answer, though, is not the one anyone expected. The binding
constraint is not the NW RFC SDK licence — we never took that licence — and it
is not really the ADT REST question either. It is the **SAP API Policy
v.4/2026** (published April 2026, and it "forms part of the Documentation for
the SAP solution with which it is provided"). §1.2 says third-party
applications "must not access, invoke, or interact in any manner with APIs that
are not Published APIs". §2.2.2 prohibits API use for "interaction or
integration with (semi-)autonomous or generative AI systems that plan, select,
or execute sequences of API calls" — **"except through and within the limits of
SAP-endorsed architectures … expressly identified and intended for such
purposes."**

SAP's ADT MCP server is the obvious candidate for such a pathway — SAP ships
it, documents it, and built it for exactly this purpose, though SAP nowhere
labels it an "endorsed architecture" in those words (**inferred**). That is the
strongest version of the instinct behind the question, and a better argument
than the one it had in mind. Against it: an Eclipse plugin brings us under the **SAP Developer
License Agreement 3.2**, which we are currently outside of entirely, and the
route to the MCP extension point was bytecode analysis of SAP bundles, not
documentation.

There is one more inversion worth stating plainly. §1.2's carve-out expressly
permits "custom-developed ABAP interfaces in private cloud and on-premise
deployments". Our `ZADT_DEBUG` facade — the thing we were about to retire —
lands inside that carve-out. `SADT_REST_RFC_ENDPOINT`, an SAP-delivered but
non-published interface, does not.

Two smaller things fell out of the licence research and belong in the verdict.
The agreement we are actually inside is not the RFC SDK licence but the **SAP
Developer Center Master Software Developer License Agreement** that governs the
A4H trial image — interoperability carve-out present, but it also bars
non-development use. And `open-rfc`, which this project ports, does **not**
claim clean-room provenance; `docs/provenance.md` should say so.

**Recommendation:** keep the RFC path as the engine, but stop treating "install
nothing" as the compliance story — it isn't one. Do the cheap host-level
experiment first (run SAP's MCP server and vsp side by side, §"what to try
next"), and if the Eclipse route is pursued, pursue it for the §2.2.2
safe-harbour argument, not for tidiness.

---

## Half 1 — what actually exists

### 1.1 SAP ships an ADT MCP server, in both IDEs

| Fact | Source | Date |
|---|---|---|
| "You can enable the ADT MCP Server in **ADT for Eclipse** to perform ABAP development tasks in an agentic workflow." Preference page *ABAP Development → MCP Server*; local HTTP server at `http://localhost:<port>/mcp`; **port defaults to 2234**; bearer token auto-generated on enable. | [Enabling ADT MCP Server (Eclipse), SAP Help — *AI in ABAP Cloud*](https://help.sap.com/docs/ABAP_AI/c7f5ef43ab274d078baf22f995fd2161/6f6e72852b9746ffbe083d5a818fbbec.html) | 2026‑08‑17 |
| Same for **ADT for VS Code**: setting `Adt > McpServer: Port`, default **2236**, range 1024–65535, disabled by default for security. | [Enabling ADT MCP Server (VS Code), SAP Help](https://help.sap.com/docs/ABAP_AI/c7f5ef43ab274d078baf22f995fd2161/75e9a03c9a164c56964340b715659209.html) | 2026‑08‑17 |
| Transport is **HTTP, local**; token in the *MCP Server: Token* preference; configurable into GitHub Copilot "or other MCP hosts such as Amazon Q". | [Configuring ADT MCP Server, SAP Help](https://help.sap.com/docs/ABAP_AI/c7f5ef43ab274d078baf22f995fd2161/ed94320814734d97801f51a5b6deb802.html) | 2026‑08‑17 |
| Shipped in ADT client **3.60**: "Enabling Agentic AI Coding in ABAP Development Tools — You can now enable the ADT MCP Server…" | [Release Notes of ADT for Eclipse 3.60](https://help.sap.com/docs/ABAP_Cloud/7c51591978f14d7ea985ec6609ab7e84/6325a3f573e9478fa5350d2d6961de41.html) | 2026‑08‑17 |
| ADT for VS Code GA'd around Sapphire 2026 (publisher SAP SE, `SAPSE.adt-vscode`); "powered by the same codebase that drives ADT for Eclipse" — a Java ABAP language server. Not open source; releasing the language server standalone is "under discussion". | [SAP Community: ADT for VS Code — Your Questions Answered](https://community.sap.com/t5/technology-blog-posts-by-sap/abap-development-tools-for-visual-studio-code-your-questions-answered/ba-p/14400848) | 2026‑06‑01 |

**The tool set**, per SAP's own
[Model Context Protocol Tools](https://help.sap.com/docs/ABAP_AI/c7f5ef43ab274d078baf22f995fd2161/243d050c1be846e788f38f8c23c45d3a.html)
page (2026‑08‑17), listed as toolsets with a "Joule License" column:
`abap_lists_destinations`, `abap_activate_objects`, `abap_transport-create`,
`abap_transport-get`, `abap_run_unit_tests`, `abap_creation-create_object`,
`abap_generators-list_generators` / `-get_schema` / `-generate_objects`,
`abap_business_services-fetch_services` / `-fetch_service_information`. Most
rows read "Not required", so the server works without a Joule licence; some
capabilities are gated. Joule for Developers itself is separately licensed and
currently offered for BTP ABAP environment and S/4HANA Cloud Public/Private
Edition ([Getting Started (Administrators)](https://help.sap.com/docs/ABAP_AI/c7f5ef43ab274d078baf22f995fd2161/748fe7cd693a4639b765cdc4ddbe4b68.html), 2026‑08‑17).

**No debugger.** Nothing in the documented toolset touches breakpoints,
attach, stepping or variables. The thing we built is precisely the thing SAP's
server does not do.

### 1.2 A third party *can* contribute tools into SAP's server

- Extension point: `com.sap.adt.mcp.core.adtMcpTools`. A tool implements
  `IAdtMCPTool` (`com.sap.adt.mcp.core`) — `getName()`, `getDescription()`,
  `getInputSchema()` (JSON Schema as a string), `execute(String jsonInput)` —
  and is declared in `plugin.xml` as `<mcpTool class="…"/>`. SAP's
  `ToolRegistrationService` picks contributions up when the server starts.
  — [`CLAUDE.md`](https://github.com/arc-mcp/arc1-adt-abap-mcp-ext/blob/main/CLAUDE.md),
  [`plugin.xml`](https://github.com/arc-mcp/arc1-adt-abap-mcp-ext/blob/main/plugin.xml)
- Three SAP bundles already use it (objectgenerator, tm.model,
  cds.servicebinding) — [`docs/decisions.md`](https://github.com/arc-mcp/arc1-adt-abap-mcp-ext/blob/main/docs/decisions.md)
- The plugin binds `com.sap.adt.*;bundle-version="[3.60.0,4.0.0)"`, touches no
  `*.internal.*` package, and reaches the backend through ADT public service
  factories (`AdtRisQuickSearchFactory`, `IAdtLogonService.ensureLoggedOn`) on
  the user's existing ABAP-project logon
  — [`docs/architecture.md`](https://github.com/arc-mcp/arc1-adt-abap-mcp-ext/blob/main/docs/architecture.md)
- Eclipse server: `http://localhost:2234/mcp`, Streamable HTTP, bearer token,
  plus a `DNSRebindingProtectionFilter` requiring `Host: localhost`.

Two caveats the plugin's README glosses:

1. **"SAP's documented extension point" does not appear to be documented by
   SAP.** I searched the SAP Help Portal content index for `adtMcpTools` and for
   ADT MCP extension points and found nothing. The plugin's research notes are
   titled *adt-eclipse-mcp-deep-dive-2026-05-22* and record bytecode-level
   findings (`AdtMCPCorePlugin.startMCPServer(int,String)`,
   `ADTMCPServer.setDestinationId(String)`, `IAdtMCPTool`). **Inferred:** the
   extension point is open and works, but it is *undocumented*, not *released*.
   Versions ≤ 0.3.x reflected into `com.sap.adt.mcp.core.internal` on ADT
   3.58/3.59, and `startMCPServer`'s signature changed at 3.60 — a concrete
   sample of what breaks on upgrade.
2. **SAP does publish an ADT SDK.** "The software development kit for the ABAP
   development tools for Eclipse (SDK for ADT) offers a public API to implement
   or integrate your own tools with SAP's ABAP IDE on the open Eclipse platform"
   — Javadoc `com.sap.adt.core.apidoc-3.60.2.zip` plus a how-to guide
   ([SAP Development Tools](https://tools.hana.ondemand.com/), retrieved
   2026‑08‑21). Building ADT plugins is an intended activity. Whether the MCP
   extension point is *in* that Javadoc I could not determine: the download is
   gated on click-through acceptance of the SAP Developer License Agreement 3.2,
   and I did not accept it on anyone's behalf.

### 1.3 MCP in Eclipse independent of SAP

- **GitHub Copilot for Eclipse** is an MCP **client**: "MCP support with GitHub
  Copilot is now generally available for JetBrains, Eclipse, and Xcode", local
  and remote servers — [GitHub Changelog](https://github.blog/changelog/2025-08-13-model-context-protocol-mcp-support-for-jetbrains-eclipse-and-xcode-is-now-generally-available/),
  2025‑08‑13. This is what makes SAP's in-IDE server useful — and it equally
  happily consumes *our* server.
- **Eclipse Foundation**: no MCP in the Platform or LSP4E; LSP4E covers LSP and
  DAP only ([eclipse-lsp4e/lsp4e](https://github.com/eclipse-lsp4e/lsp4e)).
  Eclipse GLSP has a separate, unrelated MCP story
  ([eclipse.dev/glsp](https://eclipse.dev/glsp/documentation/mcp/)).
- **For comparison**: JetBrains bundles an MCP *server* from 2025.2
  ([IntelliJ IDEA docs](https://www.jetbrains.com/help/idea/mcp-server.html));
  Theia has MCP in its AI features. "IDE as MCP server" is now the norm; SAP is
  following it.
- **Community ABAP MCP servers**, all outside the IDE over ADT HTTP:
  `mario-andreschak/mcp-abap-adt`, `mcp-abap-abap-adt-api`,
  `fr0ster/mcp-abap-adt`, `YahorNovik/mcp-adt`, Marian Zeis's documentation
  server ([blog.zeis.de, 2026‑02‑04](https://blog.zeis.de/posts/2026-02-04-abap-mcp-server/)),
  and ARC‑1's standalone TypeScript server with its own plugin framework
  ([docs.arc-1-mcp.com/extensions](https://docs.arc-1-mcp.com/extensions/), spec
  2026‑06‑17). vsp is one of this family; the RFC transport is what distinguishes it.

Worth noting: `arc1-adt-abap-mcp-ext` is by **Marian Zeis**, author of
`open-rfc` — the project open-rfc-go is a port of (`NOTICE`,
`docs/provenance.md`). He is also one of the consultants who publicly objected
to the new API Policy ([The Register, 2026‑04‑29](https://www.theregister.com/2026/04/29/new_sap_api_policy_provokes/)).
The Eclipse-plugin route has a precedent maintained by someone already adjacent
to this codebase.

### 1.4 Driving Eclipse from outside

Prior art is thin and unattractive.

- **Eclipse EASE** runs JavaScript/Python/Groovy *inside* a running IDE with
  full access to the workspace and classpath
  ([eclipse.dev/ease](https://eclipse.dev/ease/documentation/)). Designed for
  in-IDE automation; no supported external RPC front door — you would build one.
- **The Equinox OSGi console** can listen on a TCP port (`-console <port>`).
  Eclipse's own help warns that "interfering with your running Eclipse IDE via
  the OSGi console may put the Eclipse IDE into a bad state"
  ([Console Shell](https://help.eclipse.org/latest/topic/org.eclipse.platform.doc.isv/guide/console_shell.htm)),
  and an exposed console is a known RCE with a public Metasploit module
  ([exploit-db 44280](https://www.exploit-db.com/exploits/44280)). It manages
  bundles; it is not an application API.
- **SWTBot** drives the UI for testing, not production automation.

There is no ADT "external API" for scripting the Eclipse client. What is
sometimes called the ADT external surface is the *server-side* REST layer —
which is what we already speak.

### 1.5 The one thing the Eclipse route genuinely adds

ADT's debugger is not a bespoke view: "the ABAP debugger is completely
integrated in the Eclipse debug framework" since kernel 7.21 / Basis 7.31 SP4
([SAP Community, *ADT Basics: Debugging ABAP applications in Eclipse*](https://blogs.sap.com/2012/11/30/adt-basics-debugging-abap-applications-in-eclipse/),
2012‑11‑30). So a suspended session is reachable through **Eclipse Platform**
API — `ILaunchManager.getDebugTargets()`, `IThread.getStackFrames()`,
`IStackFrame.getVariables()`, `IStep.stepOver()`,
`DebugPlugin.addDebugEventListener` — all `org.eclipse.debug.core`,
EPL-licensed, public and stable
([Eclipse Platform API](https://help.eclipse.org/latest/topic/org.eclipse.platform.doc.isv/reference/api/org/eclipse/debug/core/package-summary.html)).

**Inferred, not verified:** an MCP tool contributed via `adtMcpTools` could read
and step a live ADT debug session through that API without touching a single SAP
internal. What it could *not* do is start one unattended — register a listener
for a user, catch a background breakpoint hit, post-mortem attach to a dump —
because those are ADT-specific and sit behind `com.sap.adt.debug` internals.
That is the difference between an **attended** debugger and the **unattended**
one we built, and it is the single most important design fact in this report.

---

## Half 2 — the compliance terrain

**Not legal advice.** Below is what is written down, where, and what is missing.

### 2.1 The document that actually governs this: SAP API Policy v.4/2026

Published April 2026, on help.sap.com, and it states that it "forms part of the
Documentation for the SAP solution with which it is provided"
([API_Policy_latest.pdf, v.4.2026a](https://help.sap.com/doc/sap-api-policy/latest/en-US/API_Policy_latest.pdf),
downloaded and read in full, 2026‑08‑21). Verbatim:

> **1.2. Non-Published APIs.** "Customer and third-party applications must not
> access, invoke, or interact in any manner with APIs that are not Published
> APIs, in particular those that are labeled internal, private, or with a
> similar designation … **except as permitted by the Documentation or otherwise
> authorized by SAP (e.g., customers may use custom-developed ABAP interfaces in
> private cloud and on-premise deployments)**. … Customers and partners are
> required to verify that each endpoint for Documented Use is a Published API."

> **2.2.2.** "**Except through and within the limits of SAP-endorsed
> architectures, data services, or service-specific pathways expressly
> identified and intended for such purposes**, SAP prohibits API use for: (a)
> interaction or integration with (semi-) autonomous or generative AI systems
> that plan, select, or execute sequences of API calls, and (b) scraping,
> harvesting, or systematic and/or large-scale data extraction or replication."

> **3. Monitoring and remedies.** "Customers, partners, and third parties must
> not bypass, disable, or otherwise circumvent API Controls, including through
> **intermediary services, custom code or developments, proxies, gateways,
> impersonation techniques**, or similar mechanisms."

"Published API" is defined in §1.1 as an API "published on the SAP Business
Accelerator Hub … or otherwise identified in product-specific Documentation".

Four consequences, and the first is the one that answers the user's question:

1. **§2.2.2 is a safe-harbour clause, and SAP's ADT MCP server is the harbour.**
   It is an SAP-shipped, SAP-documented pathway expressly intended for agentic
   AI. Reaching ADT capability *through* it is the cleanest posture available
   under this policy. That, not the RFC SDK, is the real argument for going into
   Eclipse — and it is a better argument than the one the question assumed.
2. **Our current shape is the §2.2.2 pattern almost verbatim.** An MCP server
   through which an agent plans and executes sequences of ADT calls is exactly
   "(semi-)autonomous … AI systems that plan, select, or execute sequences of
   API calls". This is true of vsp over HTTP today, not just over RFC.
3. **§3's "intermediary services, proxies, gateways" language reads onto an
   HTTP-over-RFC bridge**, at least on a plain reading. *Inferred* — the policy
   never names ADT or RFC.
4. **§1.2's carve-out favours the Z facade over the SAP-delivered endpoint.**
   "Customers may use custom-developed ABAP interfaces in private cloud and
   on-premise deployments" covers `ZADT_DEBUG` — a customer-written RFC-enabled
   function group — squarely. `SADT_REST_RFC_ENDPOINT` is an SAP interface that
   is not a Published API. So on this axis the facade we were about to retire is
   the *safer* of the two, and "install nothing" is a usability argument, not a
   compliance one.

Countervailing, and it matters: the policy is contested. It "may have been
released unintentionally" per reporting, drew immediate criticism from
consultants and the German user group, and SAP's CEO responded that "no
customer, no partner needs to worry. We all want them, and we want to have an
open platform" ([The Register, 2026‑04‑29](https://www.theregister.com/2026/04/29/new_sap_api_policy_provokes/);
[CIO on DSAG criticism](https://www.cio.com/article/4166172/dsag-criticizes-saps-new-api-policy.html);
[Hunton Andrews Kurth analysis, 2026‑05‑26](https://www.hunton.com/insights/legal/saps-new-api-policy-raises-new-compliance-and-continuity-risks)).
The accompanying FAQ reportedly extends scope to the on-premise portfolio
including S/4HANA private cloud — I could not retrieve the FAQ PDF directly
(sap.com returns 403 to automated fetches), so that reading is second-hand.

### 2.2 Where `SADT_REST_RFC_ENDPOINT` sits

The good news first. SAP's **public** configuration guide *Configuring the ABAP
Back-End for ABAP Development Tools for Eclipse and ABAP Development Tools for
Visual Studio Code* (S/4HANA and S/4HANA Cloud Private Edition, marked PUBLIC,
**2026‑08‑13**,
[PDF](https://help.sap.com/doc/2e65ad9a26c84878b1413009f8ac07c3/202510.002/en-US/config_guide_system_backend_abap_development_tools.pdf)),
§2.1.2, says:

> "ABAP development tools **requires remote access** to the following function
> modules that are specified for the authorization object `S_RFC`:"
> `DDIF_FIELDINFO_GET`, `RFCPING`, `RFC_GET_FUNCTION_INTERFACE`,
> **`SADT_REST_RFC_ENDPOINT`**, `SUSR_USER_CHANGE_PASSWORD_RFC`
> (ACTVT 16, RFC_TYPE FUNC)

The same text appears in the 7.57 FPS00 guide (2022‑10‑10) and the 1809 guide.
It is corroborated by public KBA **3569684**, *"No RFC authorization for
function module SADT_REST_RFC_ENDPOINT"*, which fires at ADT logon. §2.1.1 of
the same guide lists the `S_ADT_RES` URI prefixes ADT uses, including
`/sap/bc/adt/debugger` and `/sap/bc/adt/debugger/*` — "ABAP Debugger", available
since back-end 7.31 SP04.

So the scenario is settled, and it is the obvious one: **`SADT_REST_RFC_ENDPOINT`
is the ADT-over-RFC transport SAP's own first-party clients use against
on-premise and private-cloud systems** — Eclipse, and now VS Code, which is why
both are named in one guide. Object facts (via
[sapdatasheet.org](https://www.sapdatasheet.org/abap/func/sadt_rest_rfc_endpoint.html),
a community mirror, not SAP): description "Endpoint for ADT REST Framework",
SAP_BASIS, application component BC‑DWB‑AIE, package/function group `SADT_REST`,
remote-enabled with basXML, last modified 2011‑09‑09.

The bad news: SAP documents that ADT *needs* the FM and that admins must grant
`S_RFC` on it. SAP documents **nothing** about its payload structures
(`SADT_REST_REQUEST` / `SADT_REST_RESPONSE`), and nowhere invites anyone else to
call it. An authorization allow-list is not an API contract. Its formal release
state ("Released for customer" / C‑contract) could not be determined from any
public source; *inferred*, an SAP_BASIS infrastructure FM under BC‑DWB‑AIE with
no release documentation is very unlikely to be C1-released, which would make it
a Non-Published API under §1.2.

Not true, and worth killing: there is **no** evidence that remote ATC, Custom
Code Migration or Solution Manager use this FM. Remote ATC uses its own RFC
object-provider mechanism
([Remote Code Analysis in ATC with a Central Check System](https://help.sap.com/docs/SAP_NETWEAVER_AS_ABAP_752/ba879a6e2ea04d9bb94c7ccd7cdac446/e75d0aa74857455e82746ed198fc494a.html));
Custom Code Migration reaches on-premise over RFC via communication scenario
`SAP_COM_0464`.

### 2.3 Are the ADT REST APIs public, supported, versioned?

No, and SAP has said so once, on the record:

> "the only official API incl. documentation is the Java SDK for building
> Eclipse tools. The underlying REST APIs are not documented."
> — Thomas Fiedler, badged SAP Product and Topic Expert,
> [SAP Community, 2020‑12‑07](https://community.sap.com/t5/application-development-and-automation-discussions/abap-development-tools-api-documentation/td-p/12215157)

Re-asked in 2025, an SAP-badged responder pointed at the ADT User Guide's
*Released APIs* topic, which is about ABAP repository objects with C0–C3
contracts *inside* the system, not about HTTP endpoints
([Q&A 14217047](https://community.sap.com/t5/technology-q-a/looking-for-documentation-on-adt-abap-development-tools-api-endpoints/qaq-p/14217047)).
No versioning, deprecation or compatibility policy for ADT REST exists that I
could find; in practice the same resource "returns HTML in 7.31 and XML in
7.5x" ([Urbani, *Joys and sorrows of the ABAP Developer Tools API*, 2019‑03‑05](https://community.sap.com/t5/application-development-and-automation-blog-posts/joys-and-sorrows-of-the-abap-developer-tools-api/ba-p/13409390)).
Nothing for ADT on api.sap.com.

What SAP *does* document is the ICF activation of `default_host/sap/bc/adt` so
that "the service can now be accessed using HTTP from an external source"
([Activating an HTTP End Point for Accessing ADT Resources](https://help.sap.com/docs/ABAP_PLATFORM_NEW/7bfe8cdcfbb040dcb6702dada8c3e2f0/16acaaa54182420b80a966161f6d9808.html)),
with no statement restricting who may call it.

**The precedents do not defend themselves.** `abap-adt-api`,
`vscode_abap_remote_fs`, `erpl-adt`, `mcp-abap-abap-adt-api` and
`abapGit/ADT_Frontend` carry **no** disclaimer about SAP support, licensing or
reverse engineering — they simply describe what they do. The one project that
frames it explicitly is `enricoandreoli/adt-rfc-bridge`, whose companion SAP
Community post (2026‑06‑09) says "this is not an officially documented
integration point… this is a community solution".

**SAP has objected to none of them.** No takedown, no cease-and-desist, no
public SAP employee objection. The single case where SAP published a
supportedness statement is abapGit, and it is a plain *no* — "SAP does not
provide support for the open source abapGit version … we ask you to refrain from
creating incidents"
([SAP-docs: Install and Set Up abapGit](https://github.com/SAP-docs/btp-cloud-platform/blob/main/docs/30-development/install-and-set-up-abapgit-2002380.md))
— which is a *support* boundary, not a prohibition, and SAP then forked abapGit
itself for BTP and supports that fork. That is the shape of SAP's posture
generally: unsupported, tolerated, occasionally absorbed.

Also relevant: `SADT_REST_RFC_ENDPOINT` is **stateless per call** as far as the
public bridges are concerned — `adt-rfc-bridge`'s README reports that
multi-request stateful flows such as activation do not work through it. Our
result is the opposite, because we pin the RFC conversation: `LOCK` → `PUT` →
`UNLOCK` → `ACTIVATE` across four calls, with the lock handle surviving
(`open-rfc-go/reports/debugger-over-rfc.md`). That appears to be a genuinely
novel finding relative to the public state of the art, and it is worth knowing
that it is novel.

### 2.4 The RFC SDK and the classic RFC protocol

This is the axis the question started from. It turns out to be the least
constraining of the three — but not for the reason we assumed, and it has one
sharp edge we had not noticed.

**Nobody has published the SDK licence.** The NW RFC SDK is distributed only
through the SAP Software Download Center behind an S-user tied to a customer
number ([SAP NW RFC SDK product page](https://support.sap.com/en/product/connectors/nwrfcsdk.html);
download pointer is SAP Note 2573790). No public copy of a licence file shipped
inside the SDK zip, and no standalone "NW RFC SDK EULA", could be found. The
SDK's own technical notes (27517, 413708, 1025361, 2573953) contain no licence
text at all. What is public is the click-through **SAP Developer License
Agreement 3.2** quoted in §2.6, whose §2(d) bars reverse engineering "except to
the extent permitted by applicable law", and which forms only "by clicking 'I
Accept' or by attempting to download, or install, or use". **`open-rfc-go` has
taken none of it** — no SDK code, no headers, no linkage.

**The licence that probably *does* touch this project is the A4H one.** The
trial/developer image is governed by the **SAP Developer Center Master Software
Developer License Agreement (Feb 2024)**, published verbatim by SAP at
[SAP-docs/abap-platform-trial-image](https://raw.githubusercontent.com/SAP-docs/abap-platform-trial-image/main/Master-developer-center-license-multiple-wsa_3.2.txt),
and its "SAP Software" definition expressly includes SAP NetWeaver Application
Server ABAP. §3(a):

> "You may not use the SAP Software or Tools for … (2) any type of
> non-development purposes. Additionally, You may not … (a) duplicate,
> **reverse engineer (unless required by law for interoperability)** the SAP
> Software or Tools, (b) modify, disassemble or decompile the SAP Software …"

Note the shape: the interoperability carve-out attaches to "reverse engineer" in
(a); (b)'s "disassemble or decompile" has none. **Inferred:** black-box
observation of gateway traffic is (a)-shaped, not (b)-shaped. The same agreement
also bars non-development use and AI training on the software. This is a real
agreement we are inside, and it deserves more attention than the SDK licence we
are outside.

**No SAP claim over the protocol, and no enforcement, anywhere in the record.**
No SAP Note, statement, patent assertion or legal action asserting the RFC wire
protocol as protected subject matter. SAP's own ABAP Keyword Documentation
describes the protocol only conceptually
([abenrfc_protocol.htm](https://help.sap.com/doc/abapdocu_750_index_htm/7.50/en-US/abenrfc_protocol.htm))
— "an internal binary format … an XML format known as xRFC is used for deep
parameters" — with no byte-level spec. SAP publishes a wire-protocol reference
for HANA SQL but not for RFC.

The prior art is unusually strong:

- **pysap** (GPLv2, public since 2014, donated by SecureAuth to **OWASP** in
  October 2022, still active) reverse-engineered NI, Diag, Router, MS, Enqueue,
  SNC, IGS, HDB **and RFC**; its `SAPNWRFC.py` says the tag mapping "was
  determined by analysis of live packet captures from multiple NWRFC SDK
  clients" ([OWASP/pysap](https://github.com/OWASP/pysap)).
- **Wireshark ships the dissectors in mainline.** `epan/dissectors/packet-sapdiag.c`
  since 2023‑01‑25 and **`packet-saprfc.c` since 2024‑05‑29**, still maintained
  ([donation thread, 2022‑09‑23](https://lists.wireshark.org/archives/wireshark-dev/202209/msg00004.html)).
- **SAP publicly thanked the researcher** — Martin Gallo appears repeatedly in
  SAP Product Security Response acknowledgements from 2014 to 2021
  ([Wayback capture](http://web.archive.org/web/20220523013125/https://wiki.scn.sap.com/wiki/pages/viewpage.action?pageId=451071888)).
  Friction with Core Security ([CORE-2015-0009](https://seclists.org/fulldisclosure/2015/May/50))
  was about disclosure timing, never about the reverse engineering.
- **The GitHub DMCA archive** holds five or six SAP notices — JCo
  redistribution, a Hybris plugin, an ex-employee's internal Git, the hybris
  Commerce source leak, `@sap/hana-client` redistribution. **Every one concerns
  redistributing SAP's own code or binaries. None concerns protocol
  reimplementation, dissectors, or clean-room clients.**
- **SAP reimplemented its own protocol without librfc.** SAP Connector for
  Microsoft .NET 3.0: "RFC protocol is **re-implemented in C#**, so there is no
  longer a dependency on librfc32.dll"
  ([Wayback, 2024‑06‑10](https://web.archive.org/web/20240610155245/https://support.sap.com/en/product/connectors/msnet.html)).
- **SAP's OSPO archived PyRFC, node-rfc and gorfc** and told the community it
  "cannot provide the necessary prerequisites for a successful restart (e.g.
  access to the RFC SDK under an open-source license…)" and therefore
  "recommend starting a fork or a completely new project"
  ([PyRFC #372](https://github.com/SAP/PyRFC/issues/372), 2024‑12‑13).

**A provenance caveat we should own.** `open-rfc`, which open-rfc-go ports, does
**not** claim clean-room: its author states he worked from the SDK Programming
Guide, the SDK Doxygen docs, the ABAP Keyword Documentation and pysap, and its
`THIRD_PARTY_NOTICES.md` discloses two files adapted from SAP's own Apache‑2.0
`node-rfc` v3.3.1
([github.com/marianfoo/open-rfc](https://github.com/marianfoo/open-rfc),
created 2026‑08‑07; [announcement](https://blog.zeis.de/posts/2026-08-10-open-rfc/),
2026‑08‑10). Two consequences. The Apache‑2.0 pieces are fine — SAP published
them under that licence. But "no SAP code at all" is not quite the claim
open-rfc-go can make about its ancestry, and `docs/provenance.md` currently
tracks the port from open-rfc without recording open-rfc's own upstream sources.
Worth fixing. Also: open-rfc is two weeks old, so SAP's silence about it is "too
early to tell", not acquiescence.

**Where SAP *has* acted on RFC use.** Not on implementations — on *what you
call*. Note **382318** flags `RFC_READ_TABLE` as **not released for customer**
("created to be used as a sample in various training courses"), which matters
because our tooling leans on it. And Note **3255746** re-labelled the ODP data
replication API over RFC from "unsupported" to **"unpermitted"** in February
2024, with a technical block shipping around June 2026 — the sharpest live SAP
action against third-party RFC use, and notably a *licensing/API-policy*
instrument, not an IP claim. That is the same lever as API Policy §1.2, and it
confirms which lever SAP actually pulls.

**Certification is voluntary and marketing-shaped.** SAP calls ICC
certification "an optional service"
([Certify My Solution](https://www.sap.com/partners/partner-program/certify-my-solution.html));
it is mandatory only for higher PartnerEdge tiers. I could verify an ABAP Add-On
Deployment certification and BC‑XAL, but **"BC‑RFC"/"CA‑RFC" appear not to
exist** as scenario codes. Nothing anywhere states that an uncertified interface
is forbidden. The [January 2026 refresh](https://news.sap.com/2026/01/updates-sap-integration-certification-program-partner-built-solutions/)
realigns certification with clean core and the new API policy, and third-party
commentary reads certification as "confirmation that the solution adheres to the
design time guidelines of the new SAP API policy" — i.e. the API Policy is again
the operative document. **Not determined:** whether ICC's list of permitted
"integration technologies" would accept an independent protocol implementation.

**Commercial exposure is orthogonal to all of this.** *SAP UK Ltd v Diageo*
([2017] EWHC 189 (TCC), 16 Feb 2017) held that third-party systems interfacing
with mySAP ERP constituted licensable "use". **Inferred:** indirect-access
exposure attaches to *connecting*, identically whether you connect through SAP's
SDK or your own Go code — an SDK-free client neither creates nor avoids it.

**Background only, not advice.** Directive 2009/24/EC **Art. 5(3)** entitles a
lawful user to "observe, study or test the functioning of the program in order
to determine the ideas and principles which underlie any element of it" while
running it — which is where black-box traffic observation sits, without needing
Art. 6's indispensability test (*inferred*). **Art. 6** covers decompilation for
interoperability but forbids using what you learn to develop "a computer program
substantially similar in its expression". **Art. 8** makes "any contractual
provisions contrary to Article 6 or to the exceptions provided for in Article
5(2) and (3) … null and void" — which is why every SAP agreement carries the
"except to the extent permitted by applicable law" carve-out. *SAS v World
Programming* (CJEU C‑406/10, 2 May 2012) holds that functionality, programming
languages and data-file formats are not protected by copyright; whether a wire
format is analogous to a file format is untested for RFC. Adjacent and live:
the Commission opened **AT.40823** against SAP in September 2025 over the
on-premise ERP maintenance aftermarket and made SAP's commitments binding for
ten years on **9 July 2026**, without a finding of infringement.

**Net:** on the RFC axis we appear to sit outside the SDK licence entirely, in
unremarkable company, with the record showing SAP enforcing redistribution and
policy — never protocol implementation. The exposure is in API Policy §2.2.2
(§2.1), in the A4H developer agreement, and — if we build the plugin — in DLA
3.2 (§2.6).

### 2.5 What SAP's multi-client story implies

SAP's answer to "more than one client" is not publishing the HTTP API — it is
reusing its own Java layer (ADT for VS Code is "powered by the same codebase
that drives ADT for Eclipse"), and, one level up, shipping the ADT MCP server as
the agent-facing surface. *Inferred:* read together with API Policy §2.2.2, the
MCP server looks deliberately positioned as the endorsed pathway precisely so
that agents do not speak ADT REST directly.

That is the argument for the Eclipse route, stated at its strongest. The
argument against is in the next section.

### 2.6 What an Eclipse plugin drags in: SAP Developer License Agreement 3.2

Everything on <https://tools.hana.ondemand.com> — the ADT Eclipse plugin, the
ADT SDK Javadoc — is "provided under the terms of the SAP DEVELOPER LICENSE
AGREEMENT 3.2", and the download is gated on accepting it
([full text](https://tools.hana.ondemand.com/developer-license-3_2.txt),
retrieved 2026‑08‑21). Acceptance is triggered "by clicking 'I Accept' **or by
attempting to download, or install, or use** the SAP software".

| Clause | Text (excerpt) | Why it bites here |
|---|---|---|
| §1(b) | Customer Applications will not "enable the bypassing or circumventing of SAP's license restrictions and/or provide users with access to the Software to which such users are not licensed" | Reimplementing capability SAP gates behind a Joule licence could be argued into this |
| §1(d) | will not "permit mass data extraction from an SAP product to a non-SAP product" | Any tool that pumps repository content into an external model brushes it |
| §2(d) | may not "decompile, disassemble or reverse engineer (**except to the extent permitted by applicable law**) the APIs Tools or Software" | The known route to `adtMcpTools` is bytecode analysis of SAP bundles |
| §2(g) | may not "use the APIs or Tools to modify existing Software or other SAP product functionality or to access the Software or other SAP products' source code or metadata" | Broad, and read literally awkward for *any* ADT plugin that reads ABAP source. Ambiguous |
| §3 | "expressly prohibited from using the Software, Tools or APIs … for the purpose of training (developing) artificial intelligence models", including "text and data mining" per §44b UrhG / Art. 4 of EU Directive 2019/790 | Prohibits *training*. Inference at a user's request is not training, but the clause is drawn broadly |
| §6–8 | no warranty, liability capped, and **you indemnify SAP** for claims arising from your Customer Application or your breach | The indemnity is the sharpest edge for anyone shipping a plugin |

So the Eclipse route is a **trade**, not an upgrade: it buys the §2.2.2
safe-harbour argument and costs an explicit contract with a reverse-engineering
clause and an indemnity — a contract open-rfc-go has never entered. `NOTICE` and
`docs/provenance.md` record that everything in the repo comes from
`marianfoo/open-rfc` under Apache‑2.0; no SAP materials were downloaded, and the
project states plainly that it is "not affiliated with, sponsored by, or
endorsed by SAP SE".

### 2.7 SAP's posture on third-party MCP servers

SAP's [Security Considerations and Recommendations](https://help.sap.com/docs/ABAP_AI/c7f5ef43ab274d078baf22f995fd2161/9bf4d2406dc64ec19a213cbf6470b3fd.html)
(2026‑08‑17) says users "should be fully aware of the potential risks of adding
third-party MCP Servers" and warns about prompt injection. It is a risk
statement, not a prohibition: SAP's own documentation *presupposes* that people
run third-party MCP servers next to theirs. It is the closest thing to an
official comment on our shape of tool, and it is neutral-to-permissive — which
sits a little awkwardly beside API Policy §2.2.2, and I cannot reconcile the two.

---

## Half 3 — the options

| # | Option | Under API Policy §1.2/§2.2.2 | Other licence exposure | Effort | Breaks on upgrade | What we gain |
|---|---|---|---|---|---|---|
| 1 | **vsp alongside Eclipse** (today) | Weak: agent drives non-published ADT REST | None — no SAP materials | zero (shipped) | Low–medium: ADT resources drift silently | Everything we have, incl. the unattended debugger |
| 2 | **vsp as an Eclipse plugin** contributing `adtMcpTools` | **Strongest**: tools run inside SAP's own MCP server, the best candidate for a §2.2.2 "endorsed architecture" (inferred) | Enters DLA 3.2 (§2(d), §2(g), indemnity); extension point undocumented | High — Java/OSGi bundle, JDK 21, distribution | High: signature changed at 3.60; `[3.60.0,4.0.0)` says 4.x is expected to break | Eclipse's authenticated session; tools in the server the user already trusts; read/step of an **attended** debug session via `org.eclipse.debug.core` |
| 3 | **vsp driving Eclipse from outside** | No better than 1 | None | High and fragile | Constantly | Nothing we cannot get otherwise |
| 4 | **RFC tunnel, no Z code** (proved 2026‑08‑21) | Weak: SAP-delivered but non-published endpoint; §3 "proxies/gateways" reads onto it | None | Already done | Low: SAP's own clients depend on the same resources | Unattended debugger, stateful locks, works where ICF/HTTPS is shut |
| 4b | **RFC tunnel + `ZADT_DEBUG` facade** | **Better than 4** — §1.2 expressly permits "custom-developed ABAP interfaces in private cloud and on-premise deployments" | None | Already built | Low — it is our code | Typed parameters, small payloads, works where ADT resources are absent |
| 5a | **vsp as an MCP *client* of SAP's ADT MCP server** | Good for whatever goes through SAP's server | None | Low | Low | Compose SAP's generators/transport tools with ours; no plugin needed |
| 5b | **Both servers in one MCP host** | Good for SAP's half, unchanged for ours | None | ~zero | none | The agent sees SAP's tools *and* ours, today |

Notes the table cannot carry.

**Option 2 does not give us the debugger we built.** SAP's MCP server exposes no
debugger, and a contributed tool reaches the backend through ADT's ordinary
stateless session. The one real debug capability a plugin adds is reading and
stepping a session *a human already started in that Eclipse*. That is a nice
feature — "ask the agent about the frame I am stopped in" — but it is a
different product from "let the agent set a breakpoint, catch a background hit,
and step it with nobody watching".

**Option 4b deserves rehabilitation.** `reports/debugger-over-rfc.md` argues,
correctly on usability grounds, that the ADT path should lead because "install
nothing" is what makes a tool try-able. On the compliance axis the ranking
inverts: a customer-written RFC-enabled function group is the one thing API
Policy §1.2 expressly names as permitted on-premise. Both paths should survive,
and the *reason* each exists should be stated honestly.

**Option 5b is nearly free and is what the question is really reaching for.**
"Let Eclipse's own SAP-supported session do the talking" is already available at
the *host* level: Copilot for Eclipse, Claude Code and Cursor all multiplex
several MCP servers. Turn SAP's server on, add vsp beside it, and the agent uses
SAP's supported tools for what SAP supports and ours for the debugger. No
integration work, no plugin, no new licence.

**Option 5a is the sleeper.** SAP's server speaks Streamable HTTP on localhost
with a bearer token. vsp could call it as a client and route the operations SAP
covers — transports, generators, activation — through SAP's endorsed pathway,
keeping only the debugger on ours. That is a plain HTTP client against a
documented local endpoint, involving no SAP code and no new licence, and it
gives most of option 2's policy benefit for a fraction of the effort. **If any
single idea in this report is worth acting on, it is this one.**

**Something we had not considered:** ARC‑1 also ships a standalone TypeScript
MCP server with a documented plugin framework — declarative `*.tool.json`
manifests and a TypeScript `defineTool()` tier
([docs.arc-1-mcp.com/extensions](https://docs.arc-1-mcp.com/extensions/),
2026‑06‑17). If the goal is distribution rather than transport, a manifest there
is cheaper than an OSGi bundle. It cannot carry the RFC debugger — `ctx.http` is
HTTP-only — but it is a route into an existing user base.

---

## What to try next, in order

1. **Run SAP's ADT MCP server and vsp side by side in one host** (an hour, no
   code). Eclipse ≥ ADT 3.60 → *ABAP Development → MCP Server* → generate token
   → point Copilot or Claude Code at `http://localhost:2234/mcp` **and** at
   `vsp mcp`. Cheapest decisive experiment: it tells you immediately whether
   "plugging into Eclipse" buys anything the host does not already give you.
2. **Check SAP's MCP server against a plain on-premise 7.58 backend (A4H).**
   Its documentation lives under *AI in ABAP Cloud* and targets BTP/S4 Cloud
   editions. If the server or its useful tools do not work against A4H, options
   2, 5a and 5b lose most of their appeal before any code is written.
3. **Prototype option 5a.** A `vsp` subcommand that speaks MCP as a *client* to
   `localhost:2234/mcp` and re-exposes SAP's tools. Half a day, no SAP code, and
   it is the cheapest way to get the §2.2.2 safe-harbour argument for the
   operations SAP covers.
4. **Prove the attended-debugger idea without building a plugin.** Confirm ADT's
   debug target really does surface through `org.eclipse.debug.core` with usable
   frames and variables — an EASE script in a scripting-enabled Eclipse
   (`getDebugTargets()`, walk one thread) answers it in minutes and needs no
   build.
5. **Decide on the Javadoc, deliberately.** Someone with authority accepts DLA
   3.2, downloads `com.sap.adt.core.apidoc-3.60.2.zip`, and greps for
   `IAdtMCPTool`, `adtMcpTools`, `AdtSystemSessionFactory`. That settles whether
   option 2 sits on released API or on bytecode archaeology. Accepting the
   agreement is the substantive act, not the download — do not do it casually.
6. **Only then, if 1–5 justify it, build the plugin** — as a thin bridge:
   contributed `adtMcpTools` that forward to a local `vsp`, so the Eclipse
   surface stays small and the logic stays in Go where it is tested.
7. **Ask SAP two narrow questions.** (a) Is
   `com.sap.adt.mcp.core.adtMcpTools` intended as public API? (b) Does API
   Policy §1.2 apply to `/sap/bc/adt/*` on on-premise systems? Both are
   reasonable to put to the ADT product team, and a clear answer to either
   changes the calculus more than any further research will.

---

## What I could not determine

- Whether `com.sap.adt.mcp.core.adtMcpTools` / `IAdtMCPTool` appear in SAP's
  published ADT SDK Javadoc. The Help Portal index has nothing; the Javadoc sits
  behind DLA 3.2 acceptance, which I did not perform.
- Whether the ADT MCP server works against a plain on-premise ABAP Platform
  backend (SAP_BASIS 7.58) rather than the cloud editions its documentation targets.
- Whether SAP considers `/sap/bc/adt/*` to be within API Policy §1.2. The policy
  never names ADT; no SAP clarification exists. **This is the single most
  consequential open question for the project, and only SAP can answer it.**
- The formal API release state of `SADT_REST_RFC_ENDPOINT`, and which package
  interfaces expose it.
- The exact scope of the API Policy for on-premise: the FAQ reportedly says it
  applies, but sap.com serves 403 to automated fetches and I could not read it
  first-hand.
- The contents of SAP's "SDK for ADT" how-to guide — sap.com 403s.
- SAP Note 2495614 leaves no public trace (not a public KBA); needs an S-user.
  Note 2189853 is unrelated to ADT — it is an ICF `HTTP_WHITELIST` security note.
- Whether SAP has ever taken action against a community ADT client, or against
  any independent implementation of the RFC protocol. I found no instance in
  either case, but that is absence of evidence, not evidence of absence.
- The verbatim text of the NW RFC SDK licence and the JCo EULA. Both sit behind
  Support Portal authentication and **no public copy appears to exist**. This is
  the largest single gap on the RFC axis.
- Primary text of SAP Notes 1025361, 2573790, 2573953, 382318, 3255746 and KBA
  2527926 — all login-gated. Everything reported about them comes from mirrors,
  third-party restatements or the search index.
- Whether ICC's list of permitted "integration technologies" would accept an
  independent protocol implementation in a certified connector — the list could
  not be obtained. "BC-RFC"/"CA-RFC" appear not to exist as scenario codes.
- Whether SAP has ever *privately* objected to pysap, Virtual Forge, or
  `open-rfc`. Only the absence of public record is established, and `open-rfc`
  is two weeks old.
- Any SAP patent asserted over the RFC wire format. SAP holds RFC-adjacent
  patents; none appears to have been asserted against an implementation.
- community.sap.com returns 403 to automated fetching throughout, so every
  community claim here rests on a rendered copy or on search metadata rather
  than a direct read.
