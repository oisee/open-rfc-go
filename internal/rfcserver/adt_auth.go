// SPDX-License-Identifier: Apache-2.0
// Demo logon identity decoding adapted from open-steamgate
// tools/protocols/rfc-server.mjs at 7d8e0617a8e5ded4ead65da534acf1f1b2021863 (MIT there),
// ported by its copyright holder under Apache-2.0. See docs/provenance.md.
// Adapted from open-steamgate tools/protocols/rfc.mjs (decodeRfcLogon) and
// tools/protocols/rfc-server.mjs (authenticateLogon) at
// 13a3c5292eddb240d0c65b15a37a8788661410a4, ported by its copyright holder, who contributes it here under Apache-2.0.
// Modified for Go: bounded CPIC decoder, wrapped sentinel, and comparison of
// re-scrambled expected bytes instead of allocating a cleartext password.
// See docs/provenance.md.

package rfcserver

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
	"errors"

	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/scramble"
)

const DefaultMaxLogonBytes = 64 << 10

var ErrAuthentication = errors.New("RFC authentication failed")

// AuthenticateStaticLogon validates OSG's password logon profile. No network
// bytes survive the call; returned identity strings own their storage.
func AuthenticateStaticLogon(payload []byte, user, password, client string, identity LogonIdentity, maxBytes int) (LogonIdentity, error) {
	return authenticateLogon(payload, user, password, client, identity, maxBytes, false)
}

// DecodeDemoLogon names the caller without authenticating its password, as OSG's
// demo mode does. Identity strings own their storage.
func DecodeDemoLogon(payload []byte, identity LogonIdentity, maxBytes int) (LogonIdentity, error) {
	return authenticateLogon(payload, "", "", "", identity, maxBytes, true)
}

func authenticateLogon(payload []byte, user, password, client string, identity LogonIdentity, maxBytes int, demo bool) (LogonIdentity, error) {
	fail := func() (LogonIdentity, error) { return LogonIdentity{}, ErrAuthentication }
	if maxBytes == 0 {
		maxBytes = DefaultMaxLogonBytes
	}
	const prefixLength = 18
	if len(payload) > maxBytes || len(payload) < prefixLength+10 ||
		!bytes.Equal(payload[:12], []byte{0xd9, 0xc6, 0xc3, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0, 0xf0}) ||
		!bytes.Equal(payload[12:16], []byte{1, 1, 0, 8}) ||
		(!bytes.Equal(payload[16:18], []byte{3, 1}) && !bytes.Equal(payload[16:18], []byte{1, 2})) {
		return fail()
	}
	chain, err := cpic.DecodeFieldChainPrefix(payload[prefixLength:], 0x0101, 0xffff,
		cpic.FieldChainLimits{MaxFieldLength: &maxBytes, MaxChainLength: &maxBytes})
	if err != nil {
		return fail()
	}
	trailer := payload[prefixLength+chain.BytesConsumed:]
	if len(trailer) != 10 || binary.BigEndian.Uint16(trailer) != 0xffff || binary.BigEndian.Uint16(trailer[2:]) != 0 ||
		int(binary.BigEndian.Uint16(trailer[4:])) != prefixLength+chain.BytesConsumed+2 {
		return fail()
	}
	fields := make(map[uint16][]byte, len(chain.Fields))
	for _, field := range chain.Fields {
		fields[field.Tag] = field.Value
	}
	ascii := func(tag uint16) (string, bool) {
		value := fields[tag]
		if len(value) == 0 {
			return "", false
		}
		for _, b := range value {
			if b < 0x20 || b > 0x7e {
				return "", false
			}
		}
		return string(value), true
	}
	callerUser, uok := ascii(0x0111)
	callerClient, cok := ascii(0x0114)
	language, lok := ascii(0x0115)
	field := fields[0x0117]
	_, ticket := fields[0x0670]
	if !uok || !cok || !lok {
		return fail()
	}
	identity.User, identity.Client, identity.Language = callerUser, callerClient, language
	if demo {
		return identity, nil
	}
	if ticket || len(field) < 4 || len(field) > 44 {
		return fail()
	}
	expected, err := scramble.ScrambleRfcPassword(password, binary.LittleEndian.Uint32(field))
	if err != nil {
		return fail()
	}
	if subtle.ConstantTimeCompare([]byte(callerUser), []byte(user)) != 1 ||
		subtle.ConstantTimeCompare(field, expected) != 1 ||
		(client != "" && subtle.ConstantTimeCompare([]byte(callerClient), []byte(client)) != 1) {
		return fail()
	}
	identity.User, identity.Client, identity.Language = callerUser, callerClient, language
	return identity, nil
}
