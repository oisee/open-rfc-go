// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/metadata/recursive-parameter-index.ts at commit
// 847036d, Copyright 2026 Marian Zeis, licensed under the Apache License,
// Version 2.0. Modified by open-rfc-go contributors: rewritten in Go. The
// WeakMap that brands an index to its graph and the proxy/accessor/length-
// descriptor hardening collapse — a Go *ParameterIndex owns its own state and
// cannot be rebound; the (graph, index) pair threaded through every upstream
// function becomes ordinary methods. `cacheable` is always true (the only
// producer of a Graph is Normalize; a hand-built graph carries no identity to
// distinguish, and caching stays correct regardless). Thrown errors → returned
// wrapped sentinels. See docs/provenance.md.

package metadata

import (
	"errors"
	"fmt"
)

const absoluteMaxParameterCount = 100_000

// ErrParameterIndex reports an invalid metadata graph handed to the indexer.
var ErrParameterIndex = errors.New("metadata: invalid recursive parameter index")

// ParameterIndexWork counts traversal work for regression diagnostics.
type ParameterIndexWork struct {
	BroadClassificationNodeVisits  int
	BroadClassificationFieldVisits int
	BroadValidationNodeVisits      int
	BroadValidationFieldVisits     int
	StrictDescriptorNodeVisits     int
}

// WorkKind selects one traversal counter.
type WorkKind int

const (
	BroadClassificationNode WorkKind = iota
	BroadClassificationField
	BroadValidationNode
	BroadValidationField
	StrictDescriptorNode
)

// ParameterIndex is an invocation-scoped lookup over one metadata graph's
// parameters, plus a per-namespace cache and traversal counters.
type ParameterIndex struct {
	FunctionName   string
	ParameterCount int
	parameters     map[string]Parameter
	cacheable      bool
	caches         map[string]map[string]any
	work           ParameterIndexWork
}

// CreateParameterIndex validates and indexes every parameter name once, before
// recursive dispatch. Duplicate names are rejected even when only one would be
// active.
func CreateParameterIndex(graph Graph) (*ParameterIndex, error) {
	if graph.Version != 1 {
		return nil, fmt.Errorf("%w: graph must be a version-1 metadata graph", ErrParameterIndex)
	}
	if graph.FunctionIdentity == nil || graph.FunctionIdentity.Name == "" {
		return nil, fmt.Errorf("%w: metadata lacks a function identity", ErrParameterIndex)
	}
	declaredMaximum := graph.Limits.MaxRows
	if declaredMaximum < 0 || declaredMaximum > absoluteMaxParameterCount {
		return nil, fmt.Errorf("%w: graph maxRows is outside 0..%d", ErrParameterIndex, absoluteMaxParameterCount)
	}
	parameterCount := len(graph.Parameters)
	if parameterCount > declaredMaximum {
		return nil, fmt.Errorf("%w: graph exceeds its row budget %d", ErrParameterIndex, declaredMaximum)
	}

	parameters := make(map[string]Parameter, parameterCount)
	for position, parameter := range graph.Parameters {
		if parameter.Name == "" {
			return nil, fmt.Errorf("%w: parameter %d name must be non-empty", ErrParameterIndex, position)
		}
		if _, ok := parameters[parameter.Name]; ok {
			return nil, fmt.Errorf("%w: %s.%s has duplicate recursive metadata", ErrParameterIndex, graph.FunctionIdentity.Name, parameter.Name)
		}
		parameters[parameter.Name] = parameter
	}

	return &ParameterIndex{
		FunctionName:   graph.FunctionIdentity.Name,
		ParameterCount: parameterCount,
		parameters:     parameters,
		cacheable:      true,
		caches:         map[string]map[string]any{},
	}, nil
}

// Parameter resolves one parameter by name.
func (idx *ParameterIndex) Parameter(name string) (Parameter, bool) {
	p, ok := idx.parameters[name]
	return p, ok
}

// CacheGet reads one cache entry.
func (idx *ParameterIndex) CacheGet(namespace, key string) (any, bool) {
	if !idx.cacheable {
		return nil, false
	}
	ns, ok := idx.caches[namespace]
	if !ok {
		return nil, false
	}
	v, ok := ns[key]
	return v, ok
}

// CacheSet stores one cache entry and returns the value.
func (idx *ParameterIndex) CacheSet(namespace, key string, value any) any {
	if !idx.cacheable {
		return value
	}
	ns, ok := idx.caches[namespace]
	if !ok {
		ns = map[string]any{}
		idx.caches[namespace] = ns
	}
	ns[key] = value
	return value
}

// RecordWork increments one traversal counter.
func (idx *ParameterIndex) RecordWork(kind WorkKind) {
	switch kind {
	case BroadClassificationNode:
		idx.work.BroadClassificationNodeVisits++
	case BroadClassificationField:
		idx.work.BroadClassificationFieldVisits++
	case BroadValidationNode:
		idx.work.BroadValidationNodeVisits++
	case BroadValidationField:
		idx.work.BroadValidationFieldVisits++
	case StrictDescriptorNode:
		idx.work.StrictDescriptorNodeVisits++
	}
}

// Diagnostics returns the traversal counters.
func (idx *ParameterIndex) Diagnostics() ParameterIndexWork { return idx.work }
