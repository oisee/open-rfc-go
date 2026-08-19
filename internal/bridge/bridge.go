// SPDX-License-Identifier: Apache-2.0

// Package bridge exposes ordinary functions as RFC function modules. You declare
// a function's name and typed parameters and give it a Go callable; the bridge
// marshals an inbound RFC call into native Go values, invokes the callable, and
// marshals its outputs back onto the wire — the "translator/interfacer" of the
// polyglot RFC server (docs/polyglot-rfc-server.md). A callable that shells out
// to another language turns this into a polyglot bridge.
//
// This first cut covers CHAR and INT scalar parameters, which is enough to expose
// real logic; structures and tables build on the same shape.
package bridge

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/rfcserver"
)

// Kind is a parameter's ABAP data type, as far as the bridge marshals it today.
type Kind int

const (
	Char Kind = iota // fixed-width character; Go value is a string
	Int              // 4-byte integer; Go value is an int32
)

// Direction is whether a parameter is read from the caller or returned to it.
type Direction int

const (
	Import Direction = iota // caller -> function
	Export                  // function -> caller
)

// Param declares one function parameter.
type Param struct {
	Name   string
	Dir    Direction
	Kind   Kind
	Length int // character width for Char (bytes are 2×length on the Unicode wire)
}

// Values carries parameter values by name; Char is a string, Int an int32.
type Values map[string]any

// Func is a registered function module: its name, its parameters, and the Go
// callable that implements it. Call receives the decoded imports and returns the
// exports; returning an error raises SYSTEM_FAILURE to the ABAP caller.
type Func struct {
	Name   string
	Params []Param
	Call   func(ctx context.Context, in Values) (Values, error)
}

// Bridge is a registry of exposed functions.
type Bridge struct {
	funcs map[string]Func
}

// New returns an empty bridge.
func New() *Bridge { return &Bridge{funcs: map[string]Func{}} }

// Register adds or replaces a function module.
func (b *Bridge) Register(f Func) { b.funcs[f.Name] = f }

// Dispatcher builds an rfcserver.Dispatcher that marshals each registered
// function: decode imports -> call -> encode exports.
func (b *Bridge) Dispatcher() *rfcserver.Dispatcher {
	d := rfcserver.NewDispatcher()
	for name, f := range b.funcs {
		f := f
		d.Handle(name, func(ctx context.Context, req rfcserver.Request) (rfcserver.Response, error) {
			in, err := decodeImports(f, req)
			if err != nil {
				return rfcserver.Response{}, err
			}
			out, err := f.Call(ctx, in)
			if err != nil {
				return rfcserver.Response{}, err
			}
			return encodeExports(f, out)
		})
	}
	return d
}

func decodeImports(f Func, req rfcserver.Request) (Values, error) {
	byName := map[string][]byte{}
	for _, imp := range req.Imports {
		byName[imp.Name] = imp.Value
	}
	in := Values{}
	for _, p := range f.Params {
		if p.Dir != Import {
			continue
		}
		raw, ok := byName[p.Name]
		if !ok {
			continue // caller omitted an optional import
		}
		v, err := decodeScalar(p, raw)
		if err != nil {
			return nil, err
		}
		in[p.Name] = v
	}
	return in, nil
}

func encodeExports(f Func, out Values) (rfcserver.Response, error) {
	var exports []cpic.NamedValue
	for _, p := range f.Params {
		if p.Dir != Export {
			continue
		}
		v, ok := out[p.Name]
		if !ok {
			continue
		}
		raw, err := encodeScalar(p, v)
		if err != nil {
			return rfcserver.Response{}, err
		}
		exports = append(exports, cpic.NamedValue{Name: p.Name, Value: raw})
	}
	return rfcserver.Response{Exports: exports}, nil
}

func decodeScalar(p Param, raw []byte) (any, error) {
	switch p.Kind {
	case Char:
		return classicrfc.DecodeAbapChar(raw)
	case Int:
		if len(raw) < 4 {
			return nil, fmt.Errorf("bridge: %s INT value is %d bytes, need 4", p.Name, len(raw))
		}
		return int32(binary.LittleEndian.Uint32(raw)), nil
	default:
		return nil, fmt.Errorf("bridge: %s has an unsupported kind", p.Name)
	}
}

func encodeScalar(p Param, v any) ([]byte, error) {
	switch p.Kind {
	case Char:
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("bridge: %s expects a string", p.Name)
		}
		return classicrfc.EncodeAbapChar(s, p.Length)
	case Int:
		n, ok := toInt32(v)
		if !ok {
			return nil, fmt.Errorf("bridge: %s expects an integer", p.Name)
		}
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(n))
		return b, nil
	default:
		return nil, fmt.Errorf("bridge: %s has an unsupported kind", p.Name)
	}
}

func toInt32(v any) (int32, bool) {
	switch n := v.(type) {
	case int32:
		return n, true
	case int:
		return int32(n), true
	case int64:
		return int32(n), true
	default:
		return 0, false
	}
}
