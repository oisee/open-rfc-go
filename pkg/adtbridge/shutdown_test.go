// SPDX-License-Identifier: Apache-2.0
// Shutdown regression adapted from open-steamgate test/protocol-rfc.mjs at
// 2fb365933d89b903d3f8f42286778894cbd47d73 (MIT there), ported by its
// copyright holder under Apache-2.0. Adds Go cleanup and goroutine assertions.
// See docs/provenance.md.

package adtbridge

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Count bridge workers rather than unrelated HTTP/test runtime goroutines.
func bridgeWorkerCount() int {
	stack := make([]byte, 1<<20)
	n := runtime.Stack(stack, true)
	count := 0
	for _, worker := range []string{"adtbridge.serveConn(", "adtbridge.newIdleConn.func", "adtbridge.Serve.func"} {
		count += strings.Count(string(stack[:n]), worker)
	}
	return count
}

func waitShutdownSignal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestServeShutdownClosesConnectedClientAndWaitsForCleanup(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprintf("listenerFailure=%v", failure), func(t *testing.T) {
			baseline := bridgeWorkerCount()
			logoff, release := make(chan struct{}), make(chan struct{})
			working, finishWork := make(chan struct{}), make(chan struct{})
			var cleaned atomic.Int32
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/sap/public/bc/icf/logoff" {
					if cookie, err := r.Cookie("ctx"); err != nil || cookie.Value != "session" {
						t.Error("logoff lost session cookie")
					}
					close(logoff)
					<-release
					cleaned.Add(1)
					return
				}
				if r.URL.Path == "/slow" {
					close(working)
					<-finishWork
				}
				http.SetCookie(w, &http.Cookie{Name: "ctx", Value: "session", Path: "/"})
				fmt.Fprint(w, "state")
			}))
			defer backend.Close()
			// Always unblock work and cleanup, even when an assertion fails.
			defer func() {
				select {
				case <-finishWork:
				default:
					close(finishWork)
				}
			}()
			defer func() {
				select {
				case <-release:
				default:
					close(release)
				}
			}()
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			const grace = 80 * time.Millisecond
			go func() {
				done <- Serve(ctx, ln, Options{Backend: Backend{URL: backend.URL}, ShutdownGrace: grace, Timeout: 2 * time.Second})
			}()
			c := loggedClient(t, ln.Addr().String(), "DEVELOPER", "001")
			callBody(t, c, "GET", "/state", "state")
			send(t, c, callPacket(t, "GET", "/slow"))
			waitShutdownSignal(t, working, "backend work spanning grace")
			start := time.Now()
			if err := c.SetReadDeadline(start.Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			if failure {
				ln.Close()
			} else {
				cancel()
			}
			// The client deliberately stays connected; idle timeout cannot explain closure.
			if _, err := receive(c); err == nil {
				t.Fatal("shutdown socket still works")
			} else if e, ok := err.(net.Error); ok && e.Timeout() {
				t.Fatal("shutdown did not close socket")
			}
			if time.Since(start) < grace {
				t.Fatal("shutdown skipped grace period")
			}
			select {
			case err := <-done:
				t.Fatalf("Serve returned before backend work finished: %v", err)
			default:
			}
			close(finishWork)
			waitShutdownSignal(t, logoff, "cookie-bearing session logoff")
			select {
			case err := <-done:
				t.Fatalf("Serve returned before cleanup finished: %v", err)
			default:
			}
			close(release)
			select {
			case err := <-done:
				if failure && !errors.Is(err, net.ErrClosed) {
					t.Fatalf("listener error lost: %v", err)
				}
				if !failure && err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Serve did not join connection workers")
			}
			if cleaned.Load() != 1 {
				t.Fatal("Serve returned without session cleanup")
			}
			deadline := time.Now().Add(time.Second)
			for bridgeWorkerCount() != baseline && time.Now().Before(deadline) {
				runtime.Gosched()
			}
			if got := bridgeWorkerCount(); got != baseline {
				t.Fatalf("bridge worker leak: before=%d after=%d", baseline, got)
			}
		})
	}
}

func TestServeShutdownAllowsInflightWorkAndEarlyDrain(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		fmt.Fprint(w, "finished")
	}))
	defer backend.Close()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, ln, Options{Backend: Backend{URL: backend.URL}, ShutdownGrace: 5 * time.Second})
	}()
	c := loggedClient(t, ln.Addr().String(), "DEVELOPER", "001")
	send(t, c, callPacket(t, "GET", "/slow"))
	waitShutdownSignal(t, started, "in-flight backend request")
	cancel()
	select {
	case err := <-done:
		t.Fatalf("Serve abandoned in-flight work: %v", err)
	default:
	}
	close(release)
	if _, err := receive(c); err != nil {
		t.Fatal("in-flight reply lost:", err)
	}
	c.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("drained shutdown waited for full grace period")
	}
}
