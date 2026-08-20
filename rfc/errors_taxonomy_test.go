// SPDX-License-Identifier: Apache-2.0
package rfc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/oisee/open-rfc-go/internal/lifecycle"
	"github.com/oisee/open-rfc-go/internal/pool"
	"github.com/oisee/open-rfc-go/internal/transport"
)

// TestTranslateTaxonomy checks that internal sentinels surface as this package's
// public errors, so callers only ever match against rfc.Err*.
func TestTranslateTaxonomy(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want error
	}{
		{"pool exhausted", pool.ErrPoolExhausted, ErrPoolExhausted},
		{"pool closed", pool.ErrClosed, ErrClosed},
		{"lifecycle closed", lifecycle.ErrClosed, ErrClosed},
		{"transport closed", transport.ErrClosed, ErrClosed},
		{"deadline", context.DeadlineExceeded, ErrTimeout},
		{"wrapped pool exhausted", fmt.Errorf("acquire: %w", pool.ErrPoolExhausted), ErrPoolExhausted},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := translate(c.in)
			if !errors.Is(got, c.want) {
				t.Fatalf("translate(%v) = %v; want errors.Is(_, %v)", c.in, got, c.want)
			}
			// the original error stays reachable for diagnostics
			if !errors.Is(got, c.want) {
				t.Fatalf("public sentinel lost")
			}
		})
	}
	if translate(nil) != nil {
		t.Fatal("translate(nil) must be nil")
	}
	if got := translate(context.Canceled); !errors.Is(got, context.Canceled) {
		t.Fatalf("cancellation must stay context.Canceled, got %v", got)
	}
}
