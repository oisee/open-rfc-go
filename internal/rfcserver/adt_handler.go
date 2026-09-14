// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// ADT over RFC, delegated to whatever speaks HTTP.
//
// SADT_REST_RFC_ENDPOINT carries a whole HTTP exchange inside one RFC call, so
// the handler for it does not interpret ADT at all: it unwraps a request,
// makes it, and wraps the answer. Everything that knows what /sap/bc/adt means
// stays behind the backend URL.
//
// That is what makes the thing testable. Pointed at a real system it proves
// the transport, because the backend is known good; pointed at an
// implementation it proves the implementation, because the transport already
// is. Two unknowns that would otherwise hide inside each other, separated by
// one flag.
const (
	adtMaxBodyBytes = 64 << 20

	// Hop-by-hop headers belong to the connection that carried them, and this
	// call arrived over RFC where there was no such connection. Forwarding
	// them makes the backend answer about a transport that does not exist.
	hopByHop = "connection,keep-alive,proxy-authenticate,proxy-authorization,te,trailer,transfer-encoding,upgrade"
)

var requestDescriptor = classicrfc.FunintParameter{
	ParameterClass: "I", ParameterName: "REQUEST", TableName: adtRestRequest, Exid: "v",
}

var responseDescriptor = classicrfc.FunintParameter{
	ParameterClass: "E", ParameterName: "RESPONSE", TableName: adtRestResponse, Exid: "v",
}

// ADTRestHandler answers SADT_REST_RFC_ENDPOINT by making the request against
// backend, which is an origin such as "http://localhost:8099".
func ADTRestHandler(backend Backend, client *http.Client) (Handler, error) {
	base, err := backend.origin()
	if err != nil {
		return nil, err
	}
	if client == nil {
		// A jar, and one per handler.
		//
		// ADT keeps a stateful context and selects it with a sap-contextid
		// cookie: lock, write, activate are three calls that must land in the
		// same one. A client that forgets its cookies between calls turns that
		// into three unrelated sessions and the write presents a handle whose
		// session has gone.
		//
		// Per handler rather than per process because a conversation belongs to
		// one caller. Two Eclipses sharing a jar would share a context, which
		// is a data leak wearing the costume of a caching bug.
		//
		// The CSRF dance is deliberately not implemented here. Eclipse does it
		// itself — it asks for a token and sends it back — and a bridge that
		// joined in would be answering a question nobody asked. What a bridge
		// must do is not get in the way: keep the cookies, keep the headers.
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("rfcserver: cookie jar: %w", err)
		}
		client = &http.Client{Jar: jar}
	}
	graph := ADTRestGraph()

	return func(ctx context.Context, req Request) (Response, error) {
		xml, err := namedXrfc(req.XrfcParameters, "REQUEST")
		if err != nil {
			return Response{}, err
		}
		decoded, err := xrfc.DecodeRecursiveParameter(requestDescriptor, graph, xml, xrfc.RecursiveLimits{})
		if err != nil {
			return Response{}, fmt.Errorf("rfcserver: REQUEST could not be read: %w", err)
		}
		outgoing, err := httpRequestOf(ctx, base, decoded)
		if err != nil {
			return Response{}, err
		}
		// who we are to the backend: the RFC logon authenticated the client to
		// us, and the backend saw none of it
		backend.apply(outgoing)
		answer, err := client.Do(outgoing)
		if err != nil {
			return Response{}, fmt.Errorf("rfcserver: the backend did not answer: %w", err)
		}
		defer answer.Body.Close()
		body, err := io.ReadAll(io.LimitReader(answer.Body, adtMaxBodyBytes))
		if err != nil {
			return Response{}, fmt.Errorf("rfcserver: the backend's answer could not be read: %w", err)
		}
		encoded, err := xrfc.EncodeRecursiveParameter(responseDescriptor, graph, responseValue(answer, body), xrfc.RecursiveLimits{})
		if err != nil {
			return Response{}, fmt.Errorf("rfcserver: RESPONSE could not be written: %w", err)
		}
		return Response{XrfcParameters: []cpic.NamedValue{{Name: "RESPONSE", Value: encoded}}}, nil
	}, nil
}

func namedXrfc(parameters []cpic.NamedValue, name string) ([]byte, error) {
	for _, p := range parameters {
		if p.Name == name {
			return p.Value, nil
		}
	}
	return nil, fmt.Errorf("rfcserver: the call carries no %s parameter", name)
}

func httpRequestOf(ctx context.Context, base *url.URL, decoded any) (*http.Request, error) {
	fields, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rfcserver: REQUEST is not a structure")
	}
	line, _ := fields["REQUEST_LINE"].(map[string]any)
	method, _ := line["METHOD"].(string)
	target, _ := line["URI"].(string)
	if method == "" || target == "" {
		return nil, fmt.Errorf("rfcserver: REQUEST_LINE has no method or URI")
	}
	// The URI is a path with a query, as it was on the client's wire; the
	// origin is ours to choose, which is the whole point of the backend flag.
	ref, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("rfcserver: REQUEST_LINE URI %q: %w", target, err)
	}
	body, _ := fields["MESSAGE_BODY"].([]byte)
	outgoing, err := http.NewRequestWithContext(ctx, method, base.ResolveReference(ref).String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("rfcserver: %w", err)
	}
	rows, _ := fields["HEADER_FIELDS"].([]any)
	for _, row := range rows {
		header, _ := row.(map[string]any)
		name, _ := header["NAME"].(string)
		value, _ := header["VALUE"].(string)
		if name == "" || strings.Contains(hopByHop, strings.ToLower(name)) {
			continue
		}
		if strings.EqualFold(name, "Host") {
			outgoing.Host = value
			continue
		}
		outgoing.Header.Add(name, value)
	}
	return outgoing, nil
}

func responseValue(answer *http.Response, body []byte) map[string]any {
	headers := make([]any, 0, len(answer.Header))
	for name, values := range answer.Header {
		if strings.Contains(hopByHop, strings.ToLower(name)) {
			continue
		}
		for _, value := range values {
			headers = append(headers, map[string]any{"NAME": name, "VALUE": value})
		}
	}
	reason := answer.Status
	if at := strings.IndexByte(reason, ' '); at >= 0 {
		reason = reason[at+1:]
	}
	return map[string]any{
		"STATUS_LINE": map[string]any{
			"VERSION": answer.Proto,
			// a string on the wire, not a number: the dictionary calls it a
			// short string and the codec would refuse an integer here
			"STATUS_CODE":   strconv.Itoa(answer.StatusCode),
			"REASON_PHRASE": reason,
		},
		"HEADER_FIELDS": headers,
		"MESSAGE_BODY":  body,
	}
}
