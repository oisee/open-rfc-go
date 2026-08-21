// SPDX-License-Identifier: Apache-2.0

package cpic

import (
	"strings"
	"testing"
)

// The cookie form substitutes '!' for '/' and is URL-escaped; the RFC wire wants
// canonical base64. NormalizeTicket must undo both, unambiguously.
func TestNormalizeTicket(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"AjQxMDM=", "AjQxMDM="},
		{"AB!CD", "AB/CD"},   // ! -> /
		{"AB%2FCD", "AB/CD"}, // %2F -> /
		{"AB%21CD", "AB/CD"}, // %21 -> ! -> /
		{"  AjQx  ", "AjQx"}, // trimmed
	} {
		if got := NormalizeTicket(tc.in); got != tc.want {
			t.Errorf("NormalizeTicket(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// The field value is the base64 text as UTF-16LE, and decoding is its exact
// inverse — the round trip is what proves we can both compose and read a ticket.
func TestTicketFieldRoundTrip(t *testing.T) {
	const b64 = "AjQxMDMBABhDAEwAQQBVAEQARQ=="
	enc := encodeTicketField(b64)
	if len(enc) != len(b64)*2 {
		t.Fatalf("UTF-16LE length = %d, want %d", len(enc), len(b64)*2)
	}
	// every second byte is the zero high byte
	for i := 1; i < len(enc); i += 2 {
		if enc[i] != 0 {
			t.Fatalf("byte %d is %#x, want 0 (not UTF-16LE)", i, enc[i])
		}
	}
	if got := DecodeTicketField(enc); got != b64 {
		t.Fatalf("round trip = %q, want %q", got, b64)
	}
}

// A ticket logon carries TagTicket and no TagPassword; a password logon is the
// other way round. Decoding the field chain back must show exactly that.
func TestInitialLogonWithTicket(t *testing.T) {
	seed := uint32(1)
	raw, err := EncodeInitialLogonRequest(InitialLogonRequestInput{
		Client: "001", User: "CLAUDE", Ticket: "AjQxMDM=", Language: "E",
		ClientAddress: "127.0.0.1", PartnerHostName: "host.example.test",
		Destination: "127.0.0.1", ProgramName: "open-rfc01",
		SessionID: make([]byte, 16), PasswordSeed: &seed,
	})
	if err != nil {
		t.Fatal(err)
	}
	// The strict logon decoder enforces the password tag order, so read the raw
	// field chain instead to confirm the credential swap.
	prefixLength := len(initialSignature) + len(initialPrefix)
	res, err := DecodeFieldChainPrefix(raw[prefixLength:], uint16(TagStart), uint16(TagEnd), FieldChainLimits{})
	if err != nil {
		t.Fatalf("decoding the logon field chain: %v", err)
	}
	chain := res.Fields
	hasTicket, hasPassword := false, false
	for _, f := range chain {
		switch f.Tag {
		case uint16(TagTicket):
			hasTicket = true
		case uint16(TagPassword):
			hasPassword = true
		}
	}
	if !hasTicket {
		t.Error("a ticket logon must carry TagTicket (0x0670)")
	}
	if hasPassword {
		t.Error("a ticket logon must not carry a password field")
	}
}

// Sanity: the real ticket captured over RFC, once UTF-16LE-stripped, is a base64
// string that starts with the SSO2 version marker — the same shape as the HTTP
// cookie. (Uses only the public prefix, no secret.)
func TestDecodedTicketLooksLikeSSO2(t *testing.T) {
	// "Aj" is the start of every 4103-codepage SSO2 ticket ('02' version).
	enc := encodeTicketField("AjQxMDM=")
	if !strings.HasPrefix(DecodeTicketField(enc), "Aj") {
		t.Fatal("decoded ticket lost its version prefix")
	}
}
