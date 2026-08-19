// SPDX-License-Identifier: Apache-2.0

package bridge

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
)

func int4(v int32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v))
	return b
}

// TestBridgeExposesGoFunctions registers two ordinary Go callables as RFC
// function modules and drives them through the bridge's dispatcher, proving the
// marshal-in / call / marshal-out round trip for INT and CHAR.
func TestBridgeExposesGoFunctions(t *testing.T) {
	b := New()
	b.Register(Func{
		Name: "Z_DOUBLE",
		Params: []Param{
			{Name: "N", Dir: Import, Kind: Int},
			{Name: "RESULT", Dir: Export, Kind: Int},
		},
		Call: func(ctx context.Context, in Values) (Values, error) {
			return Values{"RESULT": in["N"].(int32) * 2}, nil
		},
	})
	b.Register(Func{
		Name: "Z_GREET",
		Params: []Param{
			{Name: "NAME", Dir: Import, Kind: Char, Length: 30},
			{Name: "GREETING", Dir: Export, Kind: Char, Length: 60},
		},
		Call: func(ctx context.Context, in Values) (Values, error) {
			return Values{"GREETING": "Hello, " + in["NAME"].(string)}, nil
		},
	})
	d := b.Dispatcher()
	ctx := context.Background()

	// Z_DOUBLE(N=21) -> RESULT=42
	payload, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName: "Z_DOUBLE",
		Imports:      []cpic.NamedValue{{Name: "N", Value: int4(21)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := d.Dispatch(ctx, payload)
	if err != nil {
		t.Fatal(err)
	}
	env, err := cpic.DecodeFunctionResultFields(resp)
	if err != nil || !env.Success {
		t.Fatalf("Z_DOUBLE response: err=%v success=%v", err, env.Success)
	}
	res, err := classicrfc.DecodeResult(env.Fields)
	if err != nil {
		t.Fatal(err)
	}
	var got int32
	for _, s := range res.Scalars {
		if s.Name == "RESULT" {
			got = int32(binary.LittleEndian.Uint32(s.Value))
		}
	}
	if got != 42 {
		t.Errorf("Z_DOUBLE RESULT = %d, want 42", got)
	}

	// Z_GREET(NAME="world") -> GREETING="Hello, world"
	name, _ := classicrfc.EncodeAbapChar("world", 30)
	payload, err = cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName: "Z_GREET",
		Imports:      []cpic.NamedValue{{Name: "NAME", Value: name}},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err = d.Dispatch(ctx, payload)
	if err != nil {
		t.Fatal(err)
	}
	env, _ = cpic.DecodeFunctionResultFields(resp)
	res, _ = classicrfc.DecodeResult(env.Fields)
	var greeting string
	for _, s := range res.Scalars {
		if s.Name == "GREETING" {
			greeting, _ = classicrfc.DecodeAbapChar(s.Value)
		}
	}
	if greeting != "Hello, world" {
		t.Errorf("Z_GREET GREETING = %q, want %q", greeting, "Hello, world")
	}
}
