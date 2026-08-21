// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/transport"
)

// TestServeTypeTFunctionCall drives a full type-T conversation over a pipe: the
// ALLOCATE is accepted, then an STFC_CONNECTION request (wrapped exactly like a
// client CUT) is decoded, dispatched, and its response returned with REQUTEXT
// echoed. This exercises the registered-server serve loop end to end — the path
// we had never driven a function call through.
func TestServeTypeTFunctionCall(t *testing.T) {
	srv, cli := net.Pipe()
	go ServeTypeT(srv, DefaultDispatcher(), nil, nil)

	ct := transport.New(cli, transport.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. ALLOCATE -> the server assigns a conversation id in its accept.
	allocate := make([]byte, 48)
	allocate[0], allocate[1] = 0x06, 0xca
	if err := ct.Send(allocate); err != nil {
		t.Fatal(err)
	}
	accept, err := ct.Receive(ctx)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if len(accept) < 48 || accept[1] != 0xca {
		t.Fatalf("unexpected accept: %d bytes fn=0x%02x", len(accept), frameFn(accept))
	}
	convID := append([]byte(nil), accept[40:48]...)

	// 2. STFC_CONNECTION, wrapped like a client CUT (the mirror of our own reply
	//    framing), fed into the serve loop.
	req, err := classicrfc.EncodeAbapChar("hello type-t", 255)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName: "STFC_CONNECTION",
		Imports:      []cpic.NamedValue{{Name: "REQUTEXT", Value: req}},
	})
	if err != nil {
		t.Fatal(err)
	}
	frame, err := wrapFSapSend(payload, convID, 0x0001)
	if err != nil {
		t.Fatal(err)
	}
	if err := ct.Send(frame); err != nil {
		t.Fatal(err)
	}

	// 3. Read the generated reply and confirm the echo round-trips.
	reply, err := ct.Receive(ctx)
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if len(reply) <= 80 {
		t.Fatalf("reply too short: %d bytes", len(reply))
	}
	env, err := cpic.DecodeFunctionResultFields(reply[80:])
	if err != nil {
		t.Fatalf("decode reply: %v", err)
	}
	if !env.Success {
		t.Fatalf("expected success")
	}
	res, err := classicrfc.DecodeResult(env.Fields)
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	var echo string
	for _, s := range res.Scalars {
		if s.Name == "ECHOTEXT" {
			echo, _ = classicrfc.DecodeAbapChar(s.Value)
		}
	}
	if echo != "hello type-t" {
		t.Fatalf("ECHOTEXT = %q, want %q", echo, "hello type-t")
	}
}
