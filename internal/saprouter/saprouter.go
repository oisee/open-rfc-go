// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/transport/saprouter-route.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown errors became
// returned wrapped sentinels. The WeakMap password side table becomes an
// unexported field, and the toJSON/inspect redaction becomes a String method
// that never prints passwords. See docs/provenance.md.

// Package saprouter encodes NI_ROUTE requests and decodes their responses, and
// admits SAProuter route strings at the network trust boundary.
package saprouter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Documented SAProuter constants.
const (
	DefaultService          = "3299"
	RouteInformationVersion = 2
	DefaultNIVersion        = 40
	RouteHeaderLength       = 0x18
	MaxRouteBytes           = 2048
	MaxRouteHops            = 255
	MaxResponsePayloadBytes = 1 << 20
)

var (
	// ErrInvalidRoute reports a malformed or disallowed SAProuter route.
	ErrInvalidRoute = errors.New("saprouter: route string is invalid")
	// ErrInvalidResponse reports a malformed SAProuter route response.
	ErrInvalidResponse = errors.New("saprouter: route response is invalid")
	// ErrRange reports an out-of-range argument.
	ErrRange = errors.New("saprouter: value out of range")
)

var (
	routeEyecatcher = []byte("NI_ROUTE\x00")
	pong            = []byte("NI_PONG\x00")
	errEyecatcher   = []byte("NI_RTERR\x00")

	printableASCII  = regexp.MustCompile(`^[\x20-\x7e]+$`)
	hostPattern     = regexp.MustCompile(`^[A-Za-z0-9_.:%\[\]-]+$`)
	servicePattern  = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	passwordPattern = regexp.MustCompile(`^[\x20-\x2e\x30-\x7e]+$`)
)

const (
	routePrefixSentinelHost = "open-rfc-target.invalid"
	routePrefixSentinelPort = "65535"
	redacted                = "[REDACTED]"
)

type internalHop struct {
	host     string
	service  string // "" means the default service 3299 and an empty wire field
	password string // "" means no password
}

func (h internalHop) byteLength() int {
	return len(h.host) + 1 + len(h.service) + 1 + len(h.password) + 1
}

// Hop is one public, password-free view of a route hop.
type Hop struct {
	Host               string
	Service            string
	UsesDefaultService bool
	PasswordProtected  bool
}

// FirstHop is the entry SAProuter a client connects to.
type FirstHop struct {
	Host               string
	Service            string
	UsesDefaultService bool
}

// Route is an admitted, redaction-safe SAProuter route. Passwords are held
// unexported and never printed.
type Route struct {
	HopCount            int
	ByteLength          int
	FirstHop            FirstHop
	Hops                []Hop
	RedactedRouteString string

	hops []internalHop
}

// String returns the redacted route string, so a Route is safe to log.
func (r *Route) String() string { return r.RedactedRouteString }

func routeField(value, kind string) (string, error) {
	if value == "" {
		return "", ErrInvalidRoute
	}
	switch kind {
	case "host":
		if len(value) < 2 || len(value) > 255 || !hostPattern.MatchString(value) {
			return "", ErrInvalidRoute
		}
	case "service":
		if len(value) > 63 || !servicePattern.MatchString(value) {
			return "", ErrInvalidRoute
		}
	case "password":
		if len(value) > 255 || !passwordPattern.MatchString(value) {
			return "", ErrInvalidRoute
		}
	}
	return value, nil
}

func redactedRoute(hops []internalHop) string {
	var b strings.Builder
	for _, hop := range hops {
		b.WriteString("/H/")
		b.WriteString(hop.host)
		if hop.service != "" {
			b.WriteString("/S/")
			b.WriteString(hop.service)
		}
		if hop.password != "" {
			b.WriteString("/W/")
			b.WriteString(redacted)
		}
	}
	return b.String()
}

// Admit parses the canonical /H/host[/S/service][/W/password]... syntax. At
// least one router and one final target are required; a password on the final
// target is rejected.
func Admit(input string) (*Route, error) {
	if len(input) > MaxRouteBytes || !printableASCII.MatchString(input) {
		return nil, ErrInvalidRoute
	}
	parts := strings.Split(input, "/")
	if len(parts) < 5 || parts[0] != "" {
		return nil, ErrInvalidRoute
	}
	var hops []internalHop
	cursor := 1
	for cursor < len(parts) {
		if parts[cursor] != "H" || cursor+1 >= len(parts) {
			return nil, ErrInvalidRoute
		}
		host, err := routeField(parts[cursor+1], "host")
		if err != nil {
			return nil, err
		}
		cursor += 2
		var service, password string
		passwordSeen := false
		for cursor < len(parts) && parts[cursor] != "H" {
			if cursor+1 >= len(parts) {
				return nil, ErrInvalidRoute
			}
			token, rawValue := parts[cursor], parts[cursor+1]
			switch token {
			case "S":
				if service != "" || passwordSeen {
					return nil, ErrInvalidRoute
				}
				if service, err = routeField(rawValue, "service"); err != nil {
					return nil, err
				}
			case "W":
				if passwordSeen {
					return nil, ErrInvalidRoute
				}
				if password, err = routeField(rawValue, "password"); err != nil {
					return nil, err
				}
				passwordSeen = true
			default:
				return nil, ErrInvalidRoute
			}
			cursor += 2
		}
		hops = append(hops, internalHop{host: host, service: service, password: password})
		if len(hops) > MaxRouteHops {
			return nil, ErrInvalidRoute
		}
	}
	if len(hops) < 2 || hops[len(hops)-1].password != "" {
		return nil, ErrInvalidRoute
	}
	byteLength := 0
	for _, hop := range hops {
		byteLength += hop.byteLength()
	}
	if byteLength > MaxRouteBytes {
		return nil, ErrInvalidRoute
	}

	publicHops := make([]Hop, len(hops))
	for i, hop := range hops {
		publicHops[i] = Hop{
			Host:               hop.host,
			Service:            serviceOrDefault(hop.service),
			UsesDefaultService: hop.service == "",
			PasswordProtected:  hop.password != "",
		}
	}
	first := hops[0]
	return &Route{
		HopCount:            len(hops),
		ByteLength:          byteLength,
		FirstHop:            FirstHop{Host: first.host, Service: serviceOrDefault(first.service), UsesDefaultService: first.service == ""},
		Hops:                publicHops,
		RedactedRouteString: redactedRoute(hops),
		hops:                hops,
	}, nil
}

func serviceOrDefault(service string) string {
	if service == "" {
		return DefaultService
	}
	return service
}

// AssertRoutePrefix validates the RFC SAPROUTER parameter form: a route prefix
// whose terminal /H/ placeholder is completed from the gateway endpoint.
func AssertRoutePrefix(input string) error {
	if !strings.HasSuffix(input, "/H/") || len(input) > MaxRouteBytes || !printableASCII.MatchString(input) {
		return ErrInvalidRoute
	}
	_, err := Admit(input + routePrefixSentinelHost + "/S/" + routePrefixSentinelPort)
	return err
}

// CompleteRoute binds a validated route prefix to one gateway endpoint.
func CompleteRoute(prefix, gatewayHost string, gatewayPort int) (*Route, error) {
	if err := AssertRoutePrefix(prefix); err != nil {
		return nil, err
	}
	host, err := routeField(gatewayHost, "host")
	if err != nil {
		return nil, err
	}
	if gatewayPort < 1 || gatewayPort > 0xffff {
		return nil, ErrInvalidRoute
	}
	return Admit(fmt.Sprintf("%s%s/S/%d", prefix, host, gatewayPort))
}

// EncodeRouteRequestPayload encodes one NI_ROUTE payload (without the outer
// four-byte NI length). Pass niVersion 0 to use DefaultNIVersion.
func EncodeRouteRequestPayload(route *Route, niVersion int) ([]byte, error) {
	if route == nil || len(route.hops) == 0 {
		return nil, ErrInvalidRoute
	}
	if niVersion == 0 {
		niVersion = DefaultNIVersion
	}
	if niVersion < 1 || niVersion > 255 {
		return nil, fmt.Errorf("%w: niVersion must be an integer in 1..255", ErrRange)
	}
	payload := make([]byte, RouteHeaderLength+route.ByteLength)
	copy(payload, routeEyecatcher)
	payload[9] = RouteInformationVersion
	payload[10] = byte(niVersion)
	payload[11] = byte(len(route.hops))
	payload[12] = 0 // NI_MSG_IO: the routed CPIC stream stays NI-framed
	binary.BigEndian.PutUint16(payload[13:], 0)
	payload[15] = byte(len(route.hops) - 1)
	binary.BigEndian.PutUint32(payload[16:], uint32(route.ByteLength))
	binary.BigEndian.PutUint32(payload[20:], uint32(route.hops[0].byteLength()))

	offset := RouteHeaderLength
	for _, hop := range route.hops {
		for _, field := range []string{hop.host, hop.service, hop.password} {
			offset += copy(payload[offset:], field)
			payload[offset] = 0
			offset++
		}
	}
	if offset != len(payload) {
		return nil, fmt.Errorf("%w: internal route length mismatch", ErrInvalidRoute)
	}
	return payload, nil
}

// ResponseKind distinguishes an accepted route from a rejected one.
type ResponseKind int

const (
	// Accepted means SAProuter completed the route (NI_PONG).
	Accepted ResponseKind = iota
	// Rejected means SAProuter refused the route (NI_RTERR).
	Rejected
)

// Response is the decoded route-completion acknowledgement or error status.
type Response struct {
	Kind                ResponseKind
	NIVersion           int
	ReturnCode          int32
	ErrorTextByteLength uint32
}

// DecodeRouteResponse decodes the NI_PONG acknowledgement or the bounded
// NI_RTERR error status.
func DecodeRouteResponse(input []byte) (Response, error) {
	if len(input) > MaxResponsePayloadBytes {
		return Response{}, fmt.Errorf("%w: payload bounds", ErrInvalidResponse)
	}
	if bytesEqual(input, pong) {
		return Response{Kind: Accepted}, nil
	}
	if !hasPrefix(input, errEyecatcher) {
		return Response{}, fmt.Errorf("%w: unexpected acknowledgement", ErrInvalidResponse)
	}
	if len(input) < 20 {
		return Response{}, fmt.Errorf("%w: truncated error header", ErrInvalidResponse)
	}
	niVersion := int(input[9])
	opcode := input[10]
	padding := input[11]
	returnCode := int32(binary.BigEndian.Uint32(input[12:]))
	errorTextByteLength := binary.BigEndian.Uint32(input[16:])
	if niVersion == 0 || opcode != 0 || padding != 0 || returnCode >= 0 {
		return Response{}, fmt.Errorf("%w: invalid error status", ErrInvalidResponse)
	}
	documentedLength := 20 + int(errorTextByteLength)
	modernLength := documentedLength + 4
	if len(input) != documentedLength && len(input) != modernLength {
		return Response{}, fmt.Errorf("%w: inconsistent error text length", ErrInvalidResponse)
	}
	if len(input) == modernLength && binary.BigEndian.Uint32(input[documentedLength:]) != 0 {
		return Response{}, fmt.Errorf("%w: invalid error trailer", ErrInvalidResponse)
	}
	return Response{Kind: Rejected, NIVersion: niVersion, ReturnCode: returnCode, ErrorTextByteLength: errorTextByteLength}, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasPrefix(value, prefix []byte) bool {
	if len(value) < len(prefix) {
		return false
	}
	return bytesEqual(value[:len(prefix)], prefix)
}

// PerformRoute performs the NI_ROUTE handshake over an established connection to
// the first router hop: it sends the route request as one NI record and reads
// the NI_PONG acknowledgement (or an NI_RTERR error). On success the connection
// is tunneled to the route's final hop. Pass niVersion 0 for the default.
func PerformRoute(conn io.ReadWriter, route *Route, niVersion int) error {
	payload, err := EncodeRouteRequestPayload(route, niVersion)
	if err != nil {
		return err
	}
	frame := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(frame[0:], uint32(len(payload)))
	copy(frame[4:], payload)
	if _, err := conn.Write(frame); err != nil {
		return fmt.Errorf("%w: sending NI_ROUTE: %v", ErrInvalidResponse, err)
	}
	var lengthPrefix [4]byte
	if _, err := io.ReadFull(conn, lengthPrefix[:]); err != nil {
		return fmt.Errorf("%w: reading response length: %v", ErrInvalidResponse, err)
	}
	responseLength := binary.BigEndian.Uint32(lengthPrefix[:])
	if responseLength > MaxResponsePayloadBytes {
		return fmt.Errorf("%w: response exceeds %d bytes", ErrInvalidResponse, MaxResponsePayloadBytes)
	}
	response := make([]byte, responseLength)
	if _, err := io.ReadFull(conn, response); err != nil {
		return fmt.Errorf("%w: reading response: %v", ErrInvalidResponse, err)
	}
	decoded, err := DecodeRouteResponse(response)
	if err != nil {
		return err
	}
	if decoded.Kind != Accepted {
		return fmt.Errorf("%w: router rejected the route (return code %d)", ErrInvalidRoute, decoded.ReturnCode)
	}
	return nil
}
