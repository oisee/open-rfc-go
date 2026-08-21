// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/oisee/open-rfc-go/cmd/rfctool"
	"github.com/oisee/open-rfc-go/rfc"
)

// `rfc session` is the pinned-conversation primitive on the command line. Most
// calls do not care which connection they take, and Client.Call is right for
// them. Some protocols do care, because the server keeps state in the ABAP
// session between calls: an attached debugger (ATTACH_DEBUGGEE returns an
// object reference, and step and stack hang off it), an XMI/XBP logon, an
// enqueue held across calls, a function group's globals. Those need every call
// to land in the same roll area — which is what this command holds open.
//
// Each input line is one call: a function module name, optionally followed by a
// JSON object of parameters.
func runSession(ctx context.Context, args []string) error {
	var script string
	for i := 0; i < len(args); i++ {
		if (args[i] == "-c" || args[i] == "--command") && i+1 < len(args) {
			script = args[i+1]
			i++
		}
	}

	c, _, err := rfctool.OpenWithTimeout(ctx, systemName, callTimeout)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	session, err := c.Pin(ctx)
	if err != nil {
		return err
	}
	defer session.Close()

	if script != "" {
		for _, line := range strings.Split(script, ";") {
			if err := sessionCall(ctx, session, strings.TrimSpace(line)); err != nil {
				return err
			}
		}
		return nil
	}

	fmt.Fprintln(os.Stderr, "pinned session — one call per line: <FM> [json]; empty line or EOF ends it")
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for {
		fmt.Fprint(os.Stderr, "rfc> ")
		if !in.Scan() {
			return nil
		}
		line := strings.TrimSpace(in.Text())
		if line == "" || line == "quit" || line == "exit" {
			return nil
		}
		if err := sessionCall(ctx, session, line); err != nil {
			fmt.Fprintln(os.Stderr, "!", err)
		}
	}
}

// sessionCall runs one "<FM> [json]" line on the pinned session.
func sessionCall(ctx context.Context, session *rfc.Session, line string) error {
	if line == "" {
		return nil
	}
	name, rest, _ := strings.Cut(line, " ")
	params, err := readParams([]string{strings.TrimSpace(rest)})
	if err != nil {
		return err
	}
	res, err := session.Call(ctx, strings.ToUpper(name), params)
	if err != nil {
		return err
	}
	return emit(res)
}
