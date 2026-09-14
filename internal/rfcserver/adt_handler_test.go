// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// One HTTP exchange, carried in and out through RFC.
//
// The backend here is an ordinary HTTP server, which is the point: the handler
// has no opinion about ADT and the test has none either. What it checks is
// that what Eclipse asked for is what the backend was asked, and that what the
// backend said is what comes back.
func TestADTRestHandlerCarriesTheExchange(t *testing.T) {
	var seenMethod, seenPath, seenAccept, seenBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod, seenPath = r.Method, r.URL.RequestURI()
		seenAccept = r.Header.Get("Accept")
		read, _ := io.ReadAll(r.Body)
		seenBody = string(read)
		w.Header().Set("Content-Type", "application/atomsvc+xml")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("<?xml version=\"1.0\"?><app:service/>"))
	}))
	defer backend.Close()

	handler, err := ADTRestHandler(backend.URL, backend.Client())
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	graph := ADTRestGraph()
	request := map[string]any{
		"REQUEST_LINE": map[string]any{
			"METHOD": "POST", "URI": "/sap/bc/adt/checkruns?reporters=abapCheckRun", "VERSION": "HTTP/1.1",
		},
		"HEADER_FIELDS": []any{
			map[string]any{"NAME": "Accept", "VALUE": "application/vnd.sap.adt.checkmessages+xml"},
			// hop-by-hop, and it must not reach a backend that was never on
			// the other end of that connection
			map[string]any{"NAME": "Connection", "VALUE": "keep-alive"},
		},
		"MESSAGE_BODY": []byte("<chkrun:checkObjectList/>"),
	}
	xml, err := xrfc.EncodeRecursiveParameter(requestDescriptor, graph, request, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}

	got, err := handler(context.Background(), Request{
		FunctionName:   adtRestFunction,
		XrfcParameters: []cpic.NamedValue{{Name: "REQUEST", Value: xml}},
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	if seenMethod != "POST" {
		t.Errorf("the backend saw method %q", seenMethod)
	}
	if seenPath != "/sap/bc/adt/checkruns?reporters=abapCheckRun" {
		t.Errorf("the backend saw path %q — the query has to survive", seenPath)
	}
	if seenAccept != "application/vnd.sap.adt.checkmessages+xml" {
		t.Errorf("the backend saw Accept %q", seenAccept)
	}
	if seenBody != "<chkrun:checkObjectList/>" {
		t.Errorf("the backend saw body %q", seenBody)
	}

	if len(got.XrfcParameters) != 1 || got.XrfcParameters[0].Name != "RESPONSE" {
		t.Fatalf("the answer is not a RESPONSE: %#v", got.XrfcParameters)
	}
	answer, err := xrfc.DecodeRecursiveParameter(responseDescriptor, graph, got.XrfcParameters[0].Value, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	fields, _ := answer.(map[string]any)
	status, _ := fields["STATUS_LINE"].(map[string]any)
	if status["STATUS_CODE"] != "201" {
		t.Errorf("status code came back as %#v, and it travels as text", status["STATUS_CODE"])
	}
	if status["REASON_PHRASE"] != "Created" {
		t.Errorf("reason phrase came back as %#v", status["REASON_PHRASE"])
	}
	body, _ := fields["MESSAGE_BODY"].([]byte)
	if string(body) != "<?xml version=\"1.0\"?><app:service/>" {
		t.Errorf("the body came back as %q", body)
	}
	headers, _ := fields["HEADER_FIELDS"].([]any)
	var contentType string
	for _, row := range headers {
		h, _ := row.(map[string]any)
		if h["NAME"] == "Content-Type" {
			contentType, _ = h["VALUE"].(string)
		}
	}
	if contentType != "application/atomsvc+xml" {
		t.Errorf("Content-Type came back as %q", contentType)
	}
}

// A call with nothing to make is refused rather than guessed at.
func TestADTRestHandlerRefusesACallWithoutRequest(t *testing.T) {
	handler, err := ADTRestHandler("http://127.0.0.1:1", nil)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if _, err := handler(context.Background(), Request{FunctionName: adtRestFunction}); err == nil {
		t.Fatal("a call with no REQUEST parameter should not be answered")
	}
}

func TestADTRestHandlerRejectsABadBackend(t *testing.T) {
	if _, err := ADTRestHandler("not-an-origin", nil); err == nil {
		t.Fatal("a backend that is not an origin should be refused at construction")
	}
	_ = reflect.TypeOf(0)
}
