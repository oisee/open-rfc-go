// SPDX-License-Identifier: Apache-2.0
//
// Live smoke test of the public rfc facade. Skipped unless OPEN_RFC_LIVE=1.
// Credentials come from the environment; none are stored here.

package rfc

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func liveDestination(t *testing.T) Destination {
	t.Helper()
	if os.Getenv("OPEN_RFC_LIVE") != "1" {
		t.Skip("set OPEN_RFC_LIVE=1 to run the live rfc facade test")
	}
	port := 3300
	if p := os.Getenv("SAP_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}
	return Destination{
		Host:     os.Getenv("SAP_HOST"),
		Port:     port,
		Service:  envOr("SAP_SERVICE", "sapdp00"),
		Client:   envOr("SAP_CLIENT", "001"),
		User:     os.Getenv("SAP_USER"),
		Password: os.Getenv("SAP_PASSWORD"),
		Pool:     PoolConfig{MaxSize: 2},
	}
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func TestLiveFacade(t *testing.T) {
	d := liveDestination(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := Open(ctx, d)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer c.Close(context.Background())

	// Scalar round trip through the typed facade.
	res, err := c.Call(ctx, "STFC_CONNECTION", Params{"REQUTEXT": "hello from rfc facade"})
	if err != nil {
		t.Fatalf("STFC_CONNECTION: %v", err)
	}
	echo, _ := res.Get("ECHOTEXT").(string)
	if echo != "hello from rfc facade" {
		t.Fatalf("ECHOTEXT = %q", echo)
	}
	if resp, _ := res.Get("RESPTEXT").(string); resp == "" {
		t.Fatalf("empty RESPTEXT")
	}
	t.Logf("STFC_CONNECTION ECHOTEXT=%q", echo)

	// Structure import/export + table through the typed facade.
	const marker = "rfc-facade struct"
	res, err = c.Call(ctx, "STFC_STRUCTURE", Params{
		"IMPORTSTRUCT": map[string]any{"RFCDATA1": marker, "RFCCHAR1": "Z"},
	})
	if err != nil {
		t.Fatalf("STFC_STRUCTURE: %v", err)
	}
	echoStruct, ok := res.Get("ECHOSTRUCT").(map[string]any)
	if !ok {
		t.Fatalf("ECHOSTRUCT is not a structure: %T", res.Get("ECHOSTRUCT"))
	}
	if s, _ := echoStruct["RFCDATA1"].(string); !strings.Contains(s, marker) {
		t.Fatalf("ECHOSTRUCT.RFCDATA1 = %q, want to contain %q", s, marker)
	}
	rows := res.Table("RFCTABLE")
	if len(rows) == 0 {
		t.Fatalf("RFCTABLE returned no rows")
	}
	t.Logf("STFC_STRUCTURE echo RFCDATA1=%q, RFCTABLE rows=%d", echoStruct["RFCDATA1"], len(rows))

	// A non-existent function surfaces an error (interface lookup fails).
	if _, err := c.Call(ctx, "Z_DOES_NOT_EXIST_OPENRFC", nil); err == nil {
		t.Fatalf("expected an error for a non-existent function")
	} else {
		var exc *ABAPException
		t.Logf("non-existent FM error (ABAPException=%v): %v", errors.As(err, &exc), err)
	}
}
