// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An https backend behind a private certificate is reached by naming that
// certificate, and only by naming it: verification stays on, so a bridge that
// has not been told about the certificate refuses to talk to it rather than
// trusting whatever answers.
func TestHTTPSBackendTrustsOnlyTheNamedCertificate(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-CSRF-Token") == "fetch" {
			w.Header().Set("X-CSRF-Token", "tok")
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<over-tls/>"))
	}))
	defer srv.Close()

	caPath := filepath.Join(t.TempDir(), "ca.pem")
	encoded := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(caPath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	// without the certificate: refused, and the refusal names the reason
	bare := Backend{URL: srv.URL}
	tr, err := bare.Transport()
	if err != nil || tr != nil {
		t.Fatalf("a backend with no ca_file should use the default transport: %v %v", tr, err)
	}
	handler, err := ADTRestHandler(bare, &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler(context.Background(), compactGet(t, "/sap/bc/adt/discovery")); err == nil {
		t.Fatal("an untrusted certificate was accepted")
	} else if !strings.Contains(err.Error(), "certificate") && !strings.Contains(err.Error(), "x509") {
		t.Fatalf("refused, but not for the certificate: %v", err)
	}

	// with it: reached
	trusted := Backend{URL: srv.URL, CAFile: caPath}
	tr, err = trusted.Transport()
	if err != nil || tr == nil {
		t.Fatalf("transport: %v %v", tr, err)
	}
	handler, err = ADTRestHandler(trusted, &http.Client{Transport: tr})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := handler(context.Background(), compactGet(t, "/sap/bc/adt/discovery"))
	if err != nil {
		t.Fatalf("with the certificate named: %v", err)
	}
	if got := bodyOfCompact(t, resp); got != "<over-tls/>" {
		t.Fatalf("body %q", got)
	}
}

func TestBackendCAFileMustHoldACertificate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "junk.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (Backend{URL: "https://x", CAFile: path}).Transport(); err == nil {
		t.Fatal("a file with no certificate was accepted")
	}
	if _, err := (Backend{URL: "https://x", CAFile: filepath.Join(t.TempDir(), "absent")}).Transport(); err == nil {
		t.Fatal("a missing ca_file was accepted")
	}
}
