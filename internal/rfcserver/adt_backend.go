// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Who the bridge is, to the thing behind it.
//
// On a real system the logon happens in the RFC conversation and the ICF call
// inherits that session: by the time a request reaches /sap/bc/adt it is
// already somebody. A bridge is an outsider to that. It hands the exchange to
// an HTTP backend which has never seen the RFC logon, so unless it says who it
// is the answer is 401 and nothing else.
//
// So credentials are the backend's business, not the caller's, and they are
// read from a file rather than taken on the command line: an argument is
// visible to every user on the machine through ps, and a password that has
// been in a process list is a password that has been published.
//
// They are optional on purpose. A backend that wants no authentication — our
// own façade, for one — is configured by leaving them out.
type Backend struct {
	URL      string `json:"backend"`
	User     string `json:"user"`
	Password string `json:"password"`
	Client   string `json:"client"`
	Language string `json:"language"`

	// What the bridge calls itself in the logon answer. Eclipse is configured
	// with a system id before it ever connects and does not take kindly to
	// being told a different one, so this has to match the ABAP project's
	// System ID rather than describe the backend.
	SystemID string `json:"system_id"`
	HostName string `json:"host_name"`

	// CAFile names a PEM whose certificates are trusted for an https backend,
	// in addition to the system roots. A lab backend behind a private
	// certificate is reached by naming that certificate here, which keeps
	// verification switched on — there is deliberately no flag to turn it off,
	// because a bridge that skips verification is a bridge that can be
	// answered by anyone on the path.
	CAFile string `json:"ca_file"`
	// ServerName overrides the name the certificate is checked against, for a
	// backend reached by an address its certificate does not carry.
	ServerName string `json:"server_name"`
}

// Transport returns the round-tripper this backend needs: the default one, or
// one that also trusts CAFile.
func (b Backend) Transport() (*http.Transport, error) {
	if b.CAFile == "" && b.ServerName == "" {
		return nil, nil
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	cfg := &tls.Config{ServerName: b.ServerName, MinVersion: tls.VersionTLS12}
	if b.CAFile != "" {
		pem, err := os.ReadFile(b.CAFile)
		if err != nil {
			return nil, fmt.Errorf("rfcserver: backend ca_file: %w", err)
		}
		// the system roots stay: naming a private certificate adds to what is
		// trusted rather than replacing it
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("rfcserver: backend ca_file %s holds no certificate", b.CAFile)
		}
		cfg.RootCAs = pool
	}
	tr.TLSClientConfig = cfg
	return tr, nil
}

// LoadBackend reads a backend description from a JSON file. The environment
// overrides the file for the two secret fields, which is how a container or a
// CI job supplies them without writing them down.
func LoadBackend(path string) (Backend, error) {
	var b Backend
	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return b, fmt.Errorf("rfcserver: backend configuration: %w", err)
		}
		if err := json.Unmarshal(raw, &b); err != nil {
			return b, fmt.Errorf("rfcserver: backend configuration %s: %w", path, err)
		}
	}
	if v := os.Getenv("RFC2ADT_BACKEND"); v != "" {
		b.URL = v
	}
	if v := os.Getenv("RFC2ADT_USER"); v != "" {
		b.User = v
	}
	if v := os.Getenv("RFC2ADT_PASSWORD"); v != "" {
		b.Password = v
	}
	if v := os.Getenv("RFC2ADT_CLIENT"); v != "" {
		b.Client = v
	}
	return b, nil
}

// apply stamps the outgoing request with whatever the backend needs to accept
// it: basic authentication, and the client, which ADT takes as a query
// parameter and which is not optional on a system with more than one.
func (b Backend) apply(request *http.Request) {
	if b.User != "" {
		request.SetBasicAuth(b.User, b.Password)
	}
	if b.Client == "" && b.Language == "" {
		return
	}
	query := request.URL.Query()
	if b.Client != "" && query.Get("sap-client") == "" {
		query.Set("sap-client", b.Client)
	}
	if b.Language != "" && query.Get("sap-language") == "" {
		query.Set("sap-language", b.Language)
	}
	request.URL.RawQuery = query.Encode()
}

// Describe says what the bridge will do, without saying the password. It is
// what the process prints when it starts, so a run that authenticates and a
// run that does not are told apart at a glance rather than by a 401 later.
func (b Backend) Describe() string {
	who := "anonymous"
	if b.User != "" {
		who = b.User + ":" + strings.Repeat("*", 8)
	}
	client := b.Client
	if client == "" {
		client = "(none)"
	}
	return fmt.Sprintf("%s as %s, client %s", b.URL, who, client)
}

func (b Backend) origin() (*url.URL, error) {
	base, err := url.Parse(b.URL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("rfcserver: backend %q is not an origin such as http://host:port", b.URL)
	}
	return base, nil
}

// LogonIdentity is what the bridge tells Eclipse it is. The system id has to
// match the one the ABAP project was configured with; the rest is description.
//
// The defaults name nothing real. A bridge that has not been told a system id
// answers "OSD", which will not match anybody's project and will say so
// immediately — better than a plausible name that fails later and elsewhere.
func (b Backend) LogonIdentity() LogonIdentity {
	who := LogonIdentity{
		SystemID: b.SystemID,
		Host:     b.HostName,
		User:     b.User,
		Client:   b.Client,
		Language: b.Language,
	}
	if who.SystemID == "" {
		who.SystemID = "OSD"
	}
	if who.Host == "" {
		who.Host = "osd-bridge"
	}
	if who.Client == "" {
		who.Client = "001"
	}
	if who.Language == "" {
		who.Language = "en"
	}
	return who
}
