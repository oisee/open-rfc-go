// SPDX-License-Identifier: Apache-2.0
//
// adt-rfc-bridge answers ADT over RFC and forwards it to anything that speaks
// HTTP.
//
// Eclipse's ABAP Development Tools do not reach a system over the ICM: every
// request travels inside one RFC call to SADT_REST_RFC_ENDPOINT, on the
// gateway port, carrying a whole HTTP exchange as its argument. Measured, not
// assumed — a captured session put 579 KB across the gateway and not one byte
// across the ICM.
//
// So this listens where the gateway would, unwraps the exchange, makes it
// against --backend, and wraps the answer. It has no opinion about ADT at all.
//
// That is what makes it testable, and the reason --backend exists from the
// first line rather than later:
//
//	--backend http://a-real-system:50000   proves the TRANSPORT, because the
//	                                       backend is known to be right
//	--backend http://localhost:8099        proves the BACKEND, because the
//	                                       transport already is
//
// Two unknowns that would otherwise hide inside each other, separated by one
// flag. Without it a failure means "something is wrong" and nothing more.
//
// WHO MAY CALL THIS: anyone who can reach the port.
//
// The RFC side does not check the logon. It cannot usefully: the credentials
// that matter are the ones the bridge itself presents to the backend, and they
// are configured here rather than carried by the caller. So Eclipse may log on
// with any user and any password, and every one of them reaches the backend as
// whoever the configuration says.
//
// That is a deliberate choice and not an oversight, and it makes this an open
// door: a port on a machine that speaks to a system, with no gate in front of
// it. Bind it to a loopback or a trusted network, point it at a sandbox, and
// do not leave it running. The gateway protocol is not encrypted either, so
// what crosses it is readable to anything on the path.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

func main() {
	listen := flag.String("listen", ":3300", "gateway address Eclipse connects to")
	backend := flag.String("backend", "", "origin the ADT requests are made against, e.g. http://localhost:8099")
	config := flag.String("config", "", "JSON file with the backend and its credentials; see -help")
	timeout := flag.Duration("timeout", 120*time.Second, "how long one backend request may take")
	verbose := flag.Bool("verbose", false, "log every frame decision")
	flag.Parse()

	// Which binary this is, said out loud at startup.
	//
	// Restarting a service and wondering whether the new one is the one now
	// running is a question that should not need answering twice, and during
	// this bridge's first live session it was asked at nearly every step: a
	// deploy, a test, an unchanged symptom, and no way to tell a fix that did
	// not work from a fix that never shipped. The binary hashes itself, so the
	// log says.
	log.Printf("adt-rfc-bridge: build %s", ownBuildStamp())

	target, err := rfcserver.LoadBackend(*config)
	if err != nil {
		log.Fatalf("adt-rfc-bridge: %v", err)
	}
	if *backend != "" {
		target.URL = *backend
	}
	if target.URL == "" {
		fmt.Fprintln(os.Stderr, "adt-rfc-bridge: no backend; give --backend, or --config a file that names one")
		flag.Usage()
		os.Exit(2)
	}

	// One handler per connection, made below rather than here: it carries the
	// cookie jar, and a jar shared between two clients would share an ADT
	// context between them. Checked once at startup so a bad backend fails now
	// rather than on somebody's first request.
	if _, err := rfcserver.ADTRestHandler(target, nil); err != nil {
		log.Fatalf("adt-rfc-bridge: %v", err)
	}

	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("adt-rfc-bridge: %v", err)
	}
	defer listener.Close()
	log.Printf("adt-rfc-bridge: %s -> %s", *listen, target.Describe())
	log.Printf("adt-rfc-bridge: point an ABAP project at this host, and at the instance this port belongs to")
	log.Printf("adt-rfc-bridge: the RFC logon is NOT checked — every caller reaches the backend as %s", targetUser(target))

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("adt-rfc-bridge: accept: %v", err)
			return
		}
		go func() {
			peer := conn.RemoteAddr().String()
			log.Printf("adt-rfc-bridge: %s connected", peer)
			// the jar is made here so the timeout can be set alongside it:
			// both belong to this one conversation
			jar, err := cookiejar.New(nil)
			if err != nil {
				log.Printf("adt-rfc-bridge: %s: %v", peer, err)
				conn.Close()
				return
			}
			handler, err := rfcserver.ADTRestHandler(target, &http.Client{Timeout: *timeout, Jar: jar})
			if err != nil {
				log.Printf("adt-rfc-bridge: %s: %v", peer, err)
				conn.Close()
				return
			}
			dispatcher := rfcserver.NewDispatcher()
			dispatcher.Handle("SADT_REST_RFC_ENDPOINT", handler)
			dispatcher.Identity = target.LogonIdentity()
			logf := func(string) {}
			if *verbose {
				logf = func(s string) { log.Printf("adt-rfc-bridge: %s: %s", peer, s) }
			}
			rfcserver.ServeConscious(conn, dispatcher, logf, nil)
			log.Printf("adt-rfc-bridge: %s gone", peer)
		}()
	}
}

// targetUser names who the backend will think is calling, for the warning at
// startup. A run that authenticates and one that does not should not look the
// same in a log.
func targetUser(b rfcserver.Backend) string {
	if b.User == "" {
		return "an anonymous client"
	}
	return b.User
}

// ownBuildStamp is the first eight bytes of this executable's SHA-256, which
// is enough to tell two builds apart and short enough to read off a log line.
//
// Unreadable executable, unknown stamp: worth saying rather than guessing,
// because "unknown" is itself the answer to "is this the new one".
func ownBuildStamp() string {
	path, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	file, err := os.Open(path)
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "unknown"
	}
	stamp := hex.EncodeToString(sum.Sum(nil))[:16]

	if info, err := file.Stat(); err == nil {
		return fmt.Sprintf("%s (%s)", stamp, info.ModTime().Format("15:04:05"))
	}
	return stamp
}
