// SPDX-License-Identifier: Apache-2.0
//
// Original unit tests for the recursive parameter index ported from open-rfc
// src/metadata/recursive-parameter-index.ts. Upstream has no dedicated test
// file (it is exercised via the recursive-xRFC layer); these tests state the
// behaviours the port must hold: version/identity/budget validation, one-shot
// duplicate-name rejection, name lookup, per-namespace caching, and work
// counters. See docs/provenance.md.

package metadata

import (
	"errors"
	"testing"
)

func indexableGraph() Graph {
	return Graph{
		Version:          1,
		FunctionIdentity: &FunctionIdentity{Name: "Z_F", GenerationToken: "function:20260716:010203"},
		Nodes:            map[string]TypeNode{},
		Parameters: []Parameter{
			{FunctionName: "Z_F", Name: "IMPORT", ParameterClass: "I", Reference: ParameterReference{Kind: "scalar", InternalType: "C"}},
			{FunctionName: "Z_F", Name: "EXPORT", ParameterClass: "E", Reference: ParameterReference{Kind: "scalar", InternalType: "C"}},
		},
		Limits: defaultLimits,
	}
}

func TestCreateParameterIndex(t *testing.T) {
	idx, err := CreateParameterIndex(indexableGraph())
	if err != nil {
		t.Fatal(err)
	}
	if idx.FunctionName != "Z_F" || idx.ParameterCount != 2 {
		t.Fatalf("index = %+v", idx)
	}
	if p, ok := idx.Parameter("EXPORT"); !ok || p.ParameterClass != "E" {
		t.Fatalf("lookup EXPORT = %+v, %v", p, ok)
	}
	if _, ok := idx.Parameter("MISSING"); ok {
		t.Fatal("MISSING resolved")
	}
}

func TestCreateParameterIndexRejects(t *testing.T) {
	g := indexableGraph()
	g.Version = 2
	if _, err := CreateParameterIndex(g); !errors.Is(err, ErrParameterIndex) {
		t.Fatalf("version: %v", err)
	}
	g = indexableGraph()
	g.FunctionIdentity = nil
	if _, err := CreateParameterIndex(g); !errors.Is(err, ErrParameterIndex) {
		t.Fatalf("identity: %v", err)
	}
	g = indexableGraph()
	g.Limits.MaxRows = absoluteMaxParameterCount + 1
	if _, err := CreateParameterIndex(g); !errors.Is(err, ErrParameterIndex) {
		t.Fatalf("maxRows: %v", err)
	}
	g = indexableGraph()
	g.Limits.MaxRows = 1 // fewer than 2 parameters
	if _, err := CreateParameterIndex(g); !errors.Is(err, ErrParameterIndex) {
		t.Fatalf("budget: %v", err)
	}
	g = indexableGraph()
	g.Parameters = append(g.Parameters, Parameter{FunctionName: "Z_F", Name: "IMPORT", ParameterClass: "C"})
	if _, err := CreateParameterIndex(g); err == nil || !errors.Is(err, ErrParameterIndex) {
		t.Fatalf("duplicate: %v", err)
	}
	g = indexableGraph()
	g.Parameters = []Parameter{{FunctionName: "Z_F", Name: ""}}
	if _, err := CreateParameterIndex(g); !errors.Is(err, ErrParameterIndex) {
		t.Fatalf("empty name: %v", err)
	}
}

func TestParameterIndexCacheAndWork(t *testing.T) {
	idx, err := CreateParameterIndex(indexableGraph())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.CacheGet("ns", "k"); ok {
		t.Fatal("unexpected cache hit")
	}
	idx.CacheSet("ns", "k", 42)
	if v, ok := idx.CacheGet("ns", "k"); !ok || v.(int) != 42 {
		t.Fatalf("cache = %v, %v", v, ok)
	}
	idx.RecordWork(BroadClassificationNode)
	idx.RecordWork(BroadClassificationNode)
	idx.RecordWork(StrictDescriptorNode)
	d := idx.Diagnostics()
	if d.BroadClassificationNodeVisits != 2 || d.StrictDescriptorNodeVisits != 1 || d.BroadValidationFieldVisits != 0 {
		t.Fatalf("diagnostics = %+v", d)
	}
}
