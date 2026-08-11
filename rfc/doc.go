// SPDX-License-Identifier: Apache-2.0
//
// Original work by open-rfc-go contributors.

// Package rfc will hold the public client API for SAP classic synchronous RFC.
//
// It is empty. The port is at its first milestone — record framing — and
// nothing here connects to an SAP system yet. See docs/porting-plan.md.
//
// The package exists now so that the module's public surface has exactly one
// home from the start: everything else lives under internal/, where the
// compiler prevents an implementation detail from becoming a compatibility
// promise by accident.
package rfc
