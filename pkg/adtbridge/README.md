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
Empty `AuthMode` means `demo`, preserving existing behavior. Static mode
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
