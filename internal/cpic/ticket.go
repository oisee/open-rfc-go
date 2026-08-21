// SPDX-License-Identifier: Apache-2.0

package cpic

import (
	"net/url"
	"strings"
)

// An SAP logon ticket rides the classic-RFC logon in field TagTicket (0x0670).
// Its value is the ticket's base64 text — the same string a browser holds in the
// MYSAPSSO2 cookie — laid down as UTF-16LE (each ASCII base64 character followed
// by a zero byte). The wire format was derived clean-room from our own capture;
// see docs/discoveries/registered-server-conversation.md.

// NormalizeTicket turns a ticket in any of its wire forms into canonical base64.
// A browser cookie is URL-escaped and substitutes '!' for '/'; a value copied
// from an HTTP header may still be URL-escaped. Canonical base64 is what goes on
// the RFC wire.
func NormalizeTicket(raw string) string {
	s := strings.TrimSpace(raw)
	if unescaped, err := url.QueryUnescape(s); err == nil {
		s = unescaped
	}
	// SAP's cookie encoding uses '!' where base64 has '/'. '!' is not in the
	// base64 alphabet, so the substitution is unambiguous to undo.
	s = strings.ReplaceAll(s, "!", "/")
	return s
}

// encodeTicketField renders a ticket as the TagTicket field value: canonical
// base64 text encoded UTF-16LE.
func encodeTicketField(raw string) []byte {
	b64 := NormalizeTicket(raw)
	out := make([]byte, 0, len(b64)*2)
	for i := 0; i < len(b64); i++ {
		out = append(out, b64[i], 0x00)
	}
	return out
}

// DecodeTicketField is the inverse: given a TagTicket field value, recover the
// canonical base64 ticket text. Bytes that are not UTF-16LE ASCII are dropped,
// so a value with a trailing pad or stray byte still decodes cleanly.
func DecodeTicketField(value []byte) string {
	var sb strings.Builder
	for i := 0; i+1 < len(value); i += 2 {
		lo, hi := value[i], value[i+1]
		if hi == 0x00 && lo >= 0x20 && lo < 0x7f {
			sb.WriteByte(lo)
		}
	}
	return sb.String()
}

// TicketFromLogonFields returns the base64 ticket carried by a decoded logon
// field chain, or "" if the logon carries no ticket.
func TicketFromLogonFields(fields []Field) string {
	for _, f := range fields {
		if f.Tag == uint16(TagTicket) {
			return DecodeTicketField(f.Value)
		}
	}
	return ""
}
