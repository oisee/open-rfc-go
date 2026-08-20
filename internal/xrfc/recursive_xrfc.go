// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/values/recursive-xrfc.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go over the already-
// validated internal/metadata graph. The int8Mode and bcd representation-mode
// options are dropped (INT8→int64, packed/decimal-float→decimal strings). The
// WeakMap plan binding, the cross-invocation classification/validation caches,
// and the JavaScript proxy/prototype/getter hardening are dropped: the graph is
// produced by our own metadata.Normalize and rows are plain Go maps/slices, so
// none of that hostile-input machinery has a Go analogue. The depth-aware
// subtree-height check, cyclic rejection, node-identity checks and every size
// budget are kept, because those are correctness, not JS defensiveness.
// See docs/provenance.md.

package xrfc

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/value"
)

// FunintParameter is the classic function-interface descriptor a recursive xRFC
// parameter is resolved against.
type FunintParameter = classicrfc.FunintParameter

const (
	absoluteGraphMaxNodes   = 20_000
	absoluteGraphMaxRows    = 100_000
	absoluteGraphMaxEdges   = 100_000
	defaultRuntimeMaxNodes  = 100_000
	absoluteRuntimeMaxNodes = 1_000_000
)

var supportedRecursiveScalarTypes = map[string]bool{
	"C": true, "N": true, "D": true, "T": true, "X": true, "P": true, "F": true,
	"I": true, "b": true, "s": true, "8": true, "a": true, "e": true,
	"p": true, "n": true, "w": true, "d": true, "7": true, "x": true, "t": true,
	"i": true, "c": true, "g": true, "y": true,
}

// canonicalEntityCodePoint reports whether a code point is written as a numeric
// character reference by the recursive canonical form: the C0 controls except
// TAB/LF/CR, plus the three markup characters.
func canonicalEntityCodePoint(cp int) bool {
	return cp <= 8 || cp == 11 || cp == 12 || (cp >= 14 && cp <= 31) || cp == 38 || cp == 60 || cp == 62
}

// RecursiveLimits bounds one recursive xRFC value; a nil field takes the default.
type RecursiveLimits struct {
	MaxDepth          *int
	MaxNodes          *int
	MaxRows           *int
	MaxCells          *int
	MaxCellBytes      *int
	MaxParameterBytes *int
}

type normalizedRecursiveLimits struct {
	maxDepth          int
	maxNodes          int
	maxRows           int
	maxCells          int
	maxCellBytes      int
	maxParameterBytes int
}

func normalizeRecursiveLimits(limits RecursiveLimits) (normalizedRecursiveLimits, error) {
	var out normalizedRecursiveLimits
	var err error
	if out.maxDepth, err = boundedLimit(limits.MaxDepth, 64, 256, "maxDepth"); err != nil {
		return out, err
	}
	if out.maxNodes, err = boundedLimit(limits.MaxNodes, defaultRuntimeMaxNodes, absoluteRuntimeMaxNodes, "maxNodes"); err != nil {
		return out, err
	}
	if out.maxRows, err = boundedLimit(limits.MaxRows, cpic.DefaultMaxFieldCount, maxUint32Limit, "maxRows"); err != nil {
		return out, err
	}
	if out.maxCells, err = boundedLimit(limits.MaxCells, cpic.DefaultMaxFieldCount, maxUint32Limit, "maxCells"); err != nil {
		return out, err
	}
	if out.maxCellBytes, err = boundedLimit(limits.MaxCellBytes, cpic.DefaultMaxFieldLength, cpic.DefaultMaxFieldLength, "maxCellBytes"); err != nil {
		return out, err
	}
	if out.maxParameterBytes, err = boundedLimit(limits.MaxParameterBytes, cpic.DefaultMaxFieldChainLength, cpic.DefaultMaxFieldChainLength, "maxParameterBytes"); err != nil {
		return out, err
	}
	return out, nil
}

// ResolvedParameter is a validated plan: the graph parameter, its kind, and the
// root type node xRFC serialization descends from.
type ResolvedParameter struct {
	Parameter metadata.Parameter
	Kind      Kind
	Node      metadata.TypeNode
}

// --- graph traversal budget -------------------------------------------------

type graphBudget struct {
	maxNodes, maxRows, maxEdges int
	nodes, rows, edges          int
}

func newGraphBudget(graph metadata.Graph) (*graphBudget, error) {
	if graph.Version != 1 {
		return nil, fmt.Errorf("%w: recursive xRFC graph must be a version-1 metadata graph", ErrType)
	}
	clamp := func(v, max int, label string) (int, error) {
		if v < 0 || v > max {
			return 0, fmt.Errorf("%w: recursive xRFC graph %s is outside 0..%d", ErrRange, label, max)
		}
		return v, nil
	}
	var b graphBudget
	var err error
	if b.maxNodes, err = clamp(graph.Limits.MaxNodes, absoluteGraphMaxNodes, "maxNodes"); err != nil {
		return nil, err
	}
	if b.maxRows, err = clamp(graph.Limits.MaxRows, absoluteGraphMaxRows, "maxRows"); err != nil {
		return nil, err
	}
	if b.maxEdges, err = clamp(graph.Limits.MaxEdges, absoluteGraphMaxEdges, "maxEdges"); err != nil {
		return nil, err
	}
	if len(graph.Nodes) > b.maxNodes {
		return nil, fmt.Errorf("%w: recursive xRFC graph exceeds its node budget %d", ErrRange, b.maxNodes)
	}
	if len(graph.Parameters) > b.maxRows {
		return nil, fmt.Errorf("%w: recursive xRFC graph exceeds its row budget %d", ErrRange, b.maxRows)
	}
	return &b, nil
}

func (b *graphBudget) node(node metadata.TypeNode, path string) error {
	b.nodes++
	if b.nodes > b.maxNodes {
		return fmt.Errorf("%w: %s exceeds recursive xRFC graph node budget %d", ErrRange, path, b.maxNodes)
	}
	b.rows += len(node.Fields)
	if b.rows > b.maxRows {
		return fmt.Errorf("%w: %s exceeds recursive xRFC graph row budget %d", ErrRange, path, b.maxRows)
	}
	return nil
}

func (b *graphBudget) edge(path string) error {
	b.edges++
	if b.edges > b.maxEdges {
		return fmt.Errorf("%w: %s exceeds recursive xRFC graph edge budget %d", ErrRange, path, b.maxEdges)
	}
	return nil
}

// --- node resolution --------------------------------------------------------

func requiredNode(graph metadata.Graph, name, kind, path string) (metadata.TypeNode, error) {
	node, ok := graph.Nodes[name]
	if !ok || node.Kind != kind {
		return metadata.TypeNode{}, fmt.Errorf("%w: %s requires recursive %s node %s", ErrProtocol, path, kind, name)
	}
	if node.Name != name {
		return metadata.TypeNode{}, fmt.Errorf("%w: %s recursive %s node identity %s disagrees with map key %s", ErrProtocol, path, kind, node.Name, name)
	}
	return node, nil
}

func targetNode(graph metadata.Graph, ref metadata.Reference, path string) (metadata.TypeNode, error) {
	if ref.Cyclic {
		return metadata.TypeNode{}, fmt.Errorf("%w: %s contains a cyclic recursive RFC type", ErrProtocol, path)
	}
	return requiredNode(graph, ref.TargetType, ref.Kind, path)
}

func nodeRequiresXrfc(graph metadata.Graph, node metadata.TypeNode, budget *graphBudget) (bool, error) {
	if err := budget.node(node, node.Name); err != nil {
		return false, err
	}
	for _, field := range node.Fields {
		if field.Reference.Kind == "scalar" {
			if field.InternalType == "g" || field.InternalType == "y" {
				return true, nil
			}
			continue
		}
		fieldPath := node.Name + "." + field.Name
		if err := budget.edge(fieldPath); err != nil {
			return false, err
		}
		if _, err := targetNode(graph, field.Reference, fieldPath); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func matchingParameter(graph metadata.Graph, parameter FunintParameter) (*metadata.Parameter, error) {
	if graph.FunctionIdentity == nil {
		return nil, fmt.Errorf("%w: recursive xRFC metadata lacks a function identity", ErrProtocol)
	}
	var match *metadata.Parameter
	for i := range graph.Parameters {
		if graph.Parameters[i].Name != parameter.ParameterName {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("%w: %s.%s has duplicate recursive metadata", ErrProtocol, graph.FunctionIdentity.Name, parameter.ParameterName)
		}
		match = &graph.Parameters[i]
	}
	if match == nil {
		return nil, nil
	}
	if match.FunctionName != graph.FunctionIdentity.Name ||
		match.ParameterClass != parameter.ParameterClass ||
		match.InternalType != parameter.Exid {
		return nil, fmt.Errorf("%w: %s.%s recursive metadata disagrees with the function interface", ErrProtocol, graph.FunctionIdentity.Name, parameter.ParameterName)
	}
	if (match.Reference.Kind == "structure" || match.Reference.Kind == "table") &&
		parameter.TableName != "" && match.AssociatedType != parameter.TableName {
		return nil, fmt.Errorf("%w: %s.%s recursive type identity disagrees with the function interface", ErrProtocol, graph.FunctionIdentity.Name, parameter.ParameterName)
	}
	return match, nil
}

// ResolveParameter returns a plan only when the normalized graph forces xRFC.
// The bool is false (with a nil error) when the parameter stays on the binary
// codec (flat fixed structures and classic TABLES rows).
func ResolveParameter(graph metadata.Graph, parameter FunintParameter) (ResolvedParameter, bool, error) {
	if parameter.Exid != "u" && parameter.Exid != "v" && parameter.Exid != "h" {
		return ResolvedParameter{}, false, nil
	}
	budget, err := newGraphBudget(graph)
	if err != nil {
		return ResolvedParameter{}, false, err
	}
	if parameter.ParameterClass == "T" && parameter.Exid == "u" {
		return ResolvedParameter{}, false, nil
	}
	match, err := matchingParameter(graph, parameter)
	if err != nil {
		return ResolvedParameter{}, false, err
	}
	if match == nil || match.Reference.Kind == "scalar" || match.Reference.Kind == "exception" || match.Reference.TargetType == "" {
		if parameter.Exid == "h" {
			return ResolvedParameter{}, false, fmt.Errorf("%w: %s lacks its recursive table descriptor", ErrProtocol, parameter.ParameterName)
		}
		return ResolvedParameter{}, false, nil
	}

	var node metadata.TypeNode
	if match.Reference.Kind == "table" && parameter.ParameterClass == "T" {
		n, ok := graph.Nodes[match.Reference.TargetType]
		if !ok || n.Name != match.Reference.TargetType || (n.Kind != "table" && n.Kind != "structure") {
			return ResolvedParameter{}, false, fmt.Errorf("%w: %s requires recursive table row node %s", ErrProtocol, parameter.ParameterName, match.Reference.TargetType)
		}
		node = n
	} else {
		ref := metadata.Reference{Kind: match.Reference.Kind, TargetType: match.Reference.TargetType, Cyclic: match.Reference.Cyclic}
		n, err := targetNode(graph, ref, parameter.ParameterName)
		if err != nil {
			return ResolvedParameter{}, false, err
		}
		node = n
	}

	kind := Kind(match.Reference.Kind)
	required := parameter.Exid == "h" || parameter.Exid == "v"
	if !required {
		req, err := nodeRequiresXrfc(graph, node, budget)
		if err != nil {
			return ResolvedParameter{}, false, err
		}
		required = req
	}
	if !required {
		return ResolvedParameter{}, false, nil
	}
	return ResolvedParameter{Parameter: *match, Kind: kind, Node: node}, true, nil
}

// --- validation -------------------------------------------------------------

func validateNode(graph metadata.Graph, node metadata.TypeNode, path string, depth, maxDepth int, visiting map[string]bool, subtreeHeights map[string]int, budget *graphBudget) (int, error) {
	if depth > maxDepth {
		return 0, fmt.Errorf("%w: %s exceeds recursive xRFC depth %d", ErrRange, path, maxDepth)
	}
	if known, ok := subtreeHeights[node.Name]; ok {
		if depth+known-1 > maxDepth {
			return 0, fmt.Errorf("%w: %s exceeds recursive xRFC depth %d", ErrRange, path, maxDepth)
		}
		return known, nil
	}
	if visiting[node.Name] {
		return 0, fmt.Errorf("%w: %s contains a cyclic recursive RFC type", ErrProtocol, path)
	}
	if err := budget.node(node, path); err != nil {
		return 0, err
	}
	switch node.Kind {
	case "scalar":
		if len(node.Fields) != 1 || node.Fields[0].Name != "" {
			return 0, fmt.Errorf("%w: %s scalar type has an invalid anonymous descriptor", ErrProtocol, path)
		}
	case "table":
		anonymous := len(node.Fields) == 1 && node.Fields[0].Name == ""
		named := len(node.Fields) > 0
		for _, f := range node.Fields {
			if f.Name == "" {
				named = false
			}
		}
		if !anonymous && !named {
			return 0, fmt.Errorf("%w: %s table type has an invalid line descriptor", ErrProtocol, path)
		}
	default:
		for _, f := range node.Fields {
			if f.Name == "" {
				return 0, fmt.Errorf("%w: %s structure contains an anonymous field", ErrProtocol, path)
			}
		}
	}

	visiting[node.Name] = true
	subtreeHeight := 1
	names := map[string]bool{}
	for _, field := range node.Fields {
		if names[field.Name] {
			visiting[node.Name] = false
			return 0, fmt.Errorf("%w: %s contains duplicate field %s", ErrProtocol, path, field.Name)
		}
		names[field.Name] = true
		if len(field.Name) > 0 {
			if _, err := EscapeTag(field.Name); err != nil {
				visiting[node.Name] = false
				return 0, err
			}
		}
		fieldPath := path + ".item"
		if field.Name != "" {
			fieldPath = path + "." + field.Name
		}
		if field.Reference.Kind == "scalar" {
			if field.Reference.InternalType != field.InternalType || !supportedRecursiveScalarTypes[field.InternalType] {
				visiting[node.Name] = false
				return 0, fmt.Errorf("%w: %s xRFC scalar type %s is not implemented", ErrProtocol, fieldPath, field.InternalType)
			}
			if (field.InternalType == "C" || field.InternalType == "N" || field.InternalType == "D" || field.InternalType == "T") && field.UcLength&1 != 0 {
				visiting[node.Name] = false
				return 0, fmt.Errorf("%w: %s Unicode character width must be even", ErrRange, fieldPath)
			}
			continue
		}
		if node.Kind == "scalar" {
			visiting[node.Name] = false
			return 0, fmt.Errorf("%w: %s scalar node contains a container reference", ErrProtocol, fieldPath)
		}
		if field.Reference.Kind == "structure" && field.InternalType != "u" && field.InternalType != "v" {
			visiting[node.Name] = false
			return 0, fmt.Errorf("%w: %s contains inconsistent structure metadata", ErrProtocol, fieldPath)
		}
		if field.Reference.Kind == "table" && field.InternalType != "h" {
			visiting[node.Name] = false
			return 0, fmt.Errorf("%w: %s contains inconsistent table metadata", ErrProtocol, fieldPath)
		}
		if err := budget.edge(fieldPath); err != nil {
			visiting[node.Name] = false
			return 0, err
		}
		target, err := targetNode(graph, field.Reference, fieldPath)
		if err != nil {
			visiting[node.Name] = false
			return 0, err
		}
		childHeight, err := validateNode(graph, target, fieldPath, depth+1, maxDepth, visiting, subtreeHeights, budget)
		if err != nil {
			visiting[node.Name] = false
			return 0, err
		}
		if 1+childHeight > subtreeHeight {
			subtreeHeight = 1 + childHeight
		}
	}
	visiting[node.Name] = false
	subtreeHeights[node.Name] = subtreeHeight
	return subtreeHeight, nil
}

func validateAtDepth(graph metadata.Graph, parameter FunintParameter, maxDepth int, plan *ResolvedParameter) (ResolvedParameter, error) {
	var resolved ResolvedParameter
	if plan != nil {
		resolved = *plan
	} else {
		r, required, err := ResolveParameter(graph, parameter)
		if err != nil {
			return ResolvedParameter{}, err
		}
		if !required {
			return ResolvedParameter{}, fmt.Errorf("%w: %s does not require recursive xRFC", ErrProtocol, parameter.ParameterName)
		}
		resolved = r
	}
	if _, err := EscapeTag(parameter.ParameterName); err != nil {
		return ResolvedParameter{}, err
	}
	budget, err := newGraphBudget(graph)
	if err != nil {
		return ResolvedParameter{}, err
	}
	if _, err := validateNode(graph, resolved.Node, parameter.ParameterName, 1, maxDepth, map[string]bool{}, map[string]int{}, budget); err != nil {
		return ResolvedParameter{}, err
	}
	return resolved, nil
}

// ValidateParameter resolves then walks the complete reachable serializer graph
// without reading a value.
func ValidateParameter(graph metadata.Graph, parameter FunintParameter, maxDepth *int) (ResolvedParameter, error) {
	d, err := boundedLimit(maxDepth, 64, 256, "maxDepth")
	if err != nil {
		return ResolvedParameter{}, err
	}
	return validateAtDepth(graph, parameter, d, nil)
}

// --- tag escaping -----------------------------------------------------------

// EscapeTag applies the reversible xRFC tag grammar to an ABAP name.
func EscapeTag(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("%w: xRFC tag name must be a non-empty string", ErrType)
	}
	if err := value.AssertUnicodeScalarText(name, "xRFC tag name"); err != nil {
		return "", err
	}
	var b strings.Builder
	for i, r := range name {
		cp := int(r)
		valid := false
		if i == 0 {
			valid = r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || cp > 0xff
		} else {
			valid = r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || cp > 0xff
		}
		switch {
		case valid:
			b.WriteRune(r)
		case r == '/':
			b.WriteString("_-")
		case cp <= 0xff:
			b.WriteString(fmt.Sprintf("_--%02X", cp))
		default:
			return "", fmt.Errorf("%w: xRFC tag name contains an unsupported character", ErrProtocol)
		}
	}
	return b.String(), nil
}

func unescapeTag(v string) (string, error) {
	var b strings.Builder
	i := 0
	for i < len(v) {
		if v[i] != '_' {
			b.WriteByte(v[i])
			i++
			continue
		}
		if i+1 >= len(v) || v[i+1] != '-' {
			b.WriteByte('_')
			i++
			continue
		}
		if i+2 >= len(v) || v[i+2] != '-' {
			b.WriteByte('/')
			i += 2
			continue
		}
		if i+5 > len(v) {
			return "", fmt.Errorf("%w: xRFC XML parameter contains an invalid tag escape", ErrProtocol)
		}
		hex := v[i+3 : i+5]
		n, err := strconv.ParseUint(hex, 16, 8)
		if err != nil || !isUpperHex(hex) {
			return "", fmt.Errorf("%w: xRFC XML parameter contains an invalid tag escape", ErrProtocol)
		}
		b.WriteByte(byte(n))
		i += 5
	}
	result := b.String()
	if err := value.AssertUnicodeScalarText(result, "xRFC tag name"); err != nil {
		return "", err
	}
	reEscaped, err := EscapeTag(result)
	if err != nil || reEscaped != v {
		return "", fmt.Errorf("%w: xRFC XML parameter contains a non-canonical tag escape", ErrProtocol)
	}
	return result, nil
}

func isUpperHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// DecodeRecursiveParameterName reads the canonical top-level parameter name.
func DecodeRecursiveParameterName(v []byte, limits RecursiveLimits) (string, error) {
	max, err := boundedLimit(limits.MaxParameterBytes, cpic.DefaultMaxFieldChainLength, cpic.DefaultMaxFieldChainLength, "maxParameterBytes")
	if err != nil {
		return "", err
	}
	if len(v) == 0 || len(v) > max {
		return "", fmt.Errorf("%w: recursive xRFC XML must contain 1..%d bytes", ErrRange, max)
	}
	if len(v) >= 3 && v[0] == 0xef && v[1] == 0xbb && v[2] == 0xbf {
		return "", fmt.Errorf("%w: recursive xRFC XML must not contain a UTF-8 BOM", ErrProtocol)
	}
	text, err := decodeUTF8(v, "recursive xRFC XML")
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(text, "<") {
		return "", fmt.Errorf("%w: recursive xRFC XML lacks its top-level tag", ErrProtocol)
	}
	end := strings.IndexByte(text, '>')
	if end < 2 || end > 256 || strings.Contains(text[1:end], "<") {
		return "", fmt.Errorf("%w: recursive xRFC XML lacks a supported top-level tag", ErrProtocol)
	}
	return unescapeTag(text[1:end])
}

func decodeUTF8(v []byte, label string) (string, error) {
	if !utf8.Valid(v) {
		return "", fmt.Errorf("%w: %s is not valid UTF-8", ErrProtocol, label)
	}
	return string(v), nil
}
