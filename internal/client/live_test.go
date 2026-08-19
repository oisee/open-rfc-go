// SPDX-License-Identifier: Apache-2.0
//
// Live smoke test against a real SAP system. Skipped unless OPEN_RFC_LIVE=1.
// It dials the gateway, logs on, and calls STFC_CONNECTION, asserting the
// echo equals the request — the canonical proof the whole stack talks to a real
// system. Credentials come from the environment; none are stored here.

package client

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLiveSTFCConnection(t *testing.T) {
	if os.Getenv("OPEN_RFC_LIVE") != "1" {
		t.Skip("set OPEN_RFC_LIVE=1 to run the live smoke test")
	}
	host := os.Getenv("SAP_HOST")
	service := liveEnvOr("SAP_SERVICE", "sapdp00")
	client := liveEnvOr("SAP_CLIENT", "001")
	user := os.Getenv("SAP_USER")
	password := os.Getenv("SAP_PASSWORD")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	port := 3200
	if p := os.Getenv("SAP_PORT"); p != "" {
		port = liveAtoiOr(p, 3200)
	}
	sess, err := Open(ctx, SessionOptions{Host: host, Port: port, ApplicationServerService: service})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sess.Close()
	t.Logf("connected + gateway/APPC handshake OK (conversation established)")

	if err := sess.LogonAndPing(ctx, LogonOptions{Client: client, User: user, Password: password}); err != nil {
		t.Fatalf("logon: %v", err)
	}
	t.Logf("logon OK (authenticated=%v)", sess.Authenticated())

	echo, resp, err := sess.CallSTFCConnection(ctx, "hello from open-rfc-go")
	if err != nil {
		t.Fatalf("STFC_CONNECTION: %v", err)
	}
	t.Logf("ECHOTEXT=%q", echo)
	t.Logf("RESPTEXT=%q", resp)
	if echo != "hello from open-rfc-go" {
		t.Fatalf("echo mismatch: got %q", echo)
	}
	if resp == "" {
		t.Fatalf("empty RESPTEXT from server")
	}
}

func liveEnvOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func liveAtoiOr(s string, d int) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return d
		}
		n = n*10 + int(c-'0')
	}
	return n
}
