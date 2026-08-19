// SPDX-License-Identifier: Apache-2.0
//
// Original work by open-rfc-go contributors.

// Package rfc is the public API for the SDK-free SAP classic synchronous RFC
// client. Open a [Destination], [Client.Call] a function module with native Go
// values ([Params]), and read a typed [Result]; requested outputs and metadata
// resolution are automatic, and an ABAP-side failure is returned as an
// [ABAPException] (recover it with errors.As). Connections are pooled and
// serialized, and may be routed through a SAProuter or a SOCKS5 proxy.
//
// Everything else lives under internal/, where the compiler keeps an
// implementation detail from becoming a compatibility promise by accident.
// See docs/porting-plan.md and docs/roadmap.md.
package rfc
