# Mounting the RFC-to-ADT bridge

Validate options on the launcher's synchronous startup path, before starting
`Serve` in a goroutine:

```go
options := adtbridge.Options{
    Backend: adtbridge.Backend{URL: backendURL, User: backendUser, Password: backendPassword},
    AuthMode: "static",
    RFCUser: rfcUser,
    RFCPassword: rfcPassword,
    RFCClient: "001", // optional client restriction
}
if err := adtbridge.Check(options); err != nil {
    return err
}
// Open the listener, then run adtbridge.Serve(ctx, listener, options).
```

For OSG, map `STG_RFC_AUTH_MODE`, `STG_RFC_USER`, `STG_RFC_PASSWORD`, and
`STG_RFC_CLIENT` to these fields. Backend credentials remain separate.
Empty `AuthMode` means `demo`. Static mode
requires both RFC credentials. A backend password without a backend user is
also a synchronous configuration error (`errors.Is(err,
adtbridge.ErrAuthConfiguration)`). `Serve` validates again before accepting;
`ServeConn` closes immediately for invalid options.

Static mode supports the probe and JCo Eclipse logon profiles and printable
ASCII passwords up to 40 bytes, matching OSG. Wrong credentials, tickets,
missing passwords and calls before authentication close the connection without
an RFC reply. `Log`, if supplied, receives `RFC authentication failed` even
when verbose tracing is off. Successful logon replies identify the RFC caller;
HTTP calls use the configured backend credentials and a per-connection cookie
jar and CSRF token. `MaxLogonBytes` bounds logon reassembly and decoding,
defaulting to 64 KiB. The legacy template-only CPIC-init path is refused in
static mode.

Backend origin and subtree confinement, redirect checks, and managed-header
isolation also apply under static authentication. The command-line demo does
not select static mode automatically; launchers opt in through `Options`.

`Timeout` defaults to 120 seconds and controls both each backend HTTP request
(including CSRF probes) and RFC socket inactivity, as OSG's `timeoutMs` does.
Successful socket reads and writes reset inactivity even while HTTP work is
pending. `MaxConnections` defaults to 64 live sockets per `Serve` listener;
excess connections close immediately, and socket close frees a slot before
HTTP cleanup finishes. Zero selects these defaults; negative values fail
`Check` with `ErrLimitsConfiguration`. The CLI exposes `--timeout` and
`--max-connections`. Independent `ServeConn` calls each own a standalone session
and do not share a listener's cap or editing handles.

In demo mode without a configured backend user, HTTP Basic auth carries the
RFC caller's username and an empty password. Configured backend credentials
always take precedence; static mode never substitutes caller credentials.
Backend client and language settings remain independent of the RFC identity.
Tunneled Authorization and Cookie headers cannot override bridge-owned state.
The bridge defaults missing `X-SAP-ADT-Sessiontype` headers to `stateful` and
preserves an explicit value.

A successful POST `_action=LOCK` returning a 40-character hexadecimal
`LOCK_HANDLE` associates that handle with the backend session and the RFC
client/user pair. Another connection on the same listener may use that known
handle only for the same identity. Shared requests serialize cookies, CSRF
probes and token rotation. Successful UNLOCK or explicit backend logoff forgets
the handle; owner disconnect forgets all its handles immediately and waits
for pending HTTP work before best-effort GET `/sap/public/bc/icf/logoff`.
Cleanup sends cookies and backend credentials only if cookies were acquired,
uses the smaller of `Timeout` and five seconds, and never retries or follows
redirects. Existing backend subtree confinement also bounds this cleanup:
if the logoff path is outside the configured subtree, cleanup skips HTTP logoff
and backend idle expiry remains the fallback.
