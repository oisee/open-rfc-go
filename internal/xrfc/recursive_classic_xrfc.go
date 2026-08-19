// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/values/recursive-classic-xrfc.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go over the already-
// validated internal/metadata graph, reusing the classic-xrfc primitives in
// this package. The descriptor subtree caches, the WeakSet plan-trust guard,
// and the JS proxy/prototype/getter hardening are dropped (rows are plain Go
// maps/slices from our own metadata.Normalize). The two-pass decode (preflight
// then parse) collapses to one pass: Go returns nil on error, so no partial
// output can leak. Cyclic rejection, depth/node/row/byte budgets, and the
// I/C/g/y scalar restriction are kept. See docs/provenance.md.

package xrfc

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/value"
)

const (
	rcItemWrapperByteLength = 13 // <item></item>
	rcDefaultMaxNodes       = 100_000
	rcAbsoluteMaxNodes      = 1_000_000
	rcDefaultMaxDepth       = 64
	rcAbsoluteMaxDepth      = 256
)

var supportedClassicScalars = map[string]bool{"I": true, "C": true, "g": true, "y": true}

// RecursiveClassicLimits bounds one recursive classic xRFC value.
type RecursiveClassicLimits struct {
	MaxCellBytes      *int
	MaxRowBytes       *int
	MaxParameterBytes *int
	MaxRows           *int
	MaxNodes          *int
	MaxDepth          *int
}

type recursiveClassicLimits struct {
	classic  NormalizedLimits
	maxNodes int
	maxDepth int
}

func normalizeRecursiveClassicLimits(limits RecursiveClassicLimits) (recursiveClassicLimits, error) {
	classic, err := NormalizeLimits(Limits{
		MaxCellBytes:      limits.MaxCellBytes,
		MaxRowBytes:       limits.MaxRowBytes,
		MaxParameterBytes: limits.MaxParameterBytes,
		MaxRows:           limits.MaxRows,
	})
	if err != nil {
		return recursiveClassicLimits{}, err
	}
	maxNodes, err := boundedLimit(limits.MaxNodes, rcDefaultMaxNodes, rcAbsoluteMaxNodes, "maxNodes")
	if err != nil {
		return recursiveClassicLimits{}, err
	}
	maxDepth, err := boundedLimit(limits.MaxDepth, rcDefaultMaxDepth, rcAbsoluteMaxDepth, "maxDepth")
	if err != nil {
		return recursiveClassicLimits{}, err
	}
	return recursiveClassicLimits{classic: classic, maxNodes: maxNodes, maxDepth: maxDepth}, nil
}

// RecursiveClassicIdentity identifies the parameter to resolve.
type RecursiveClassicIdentity struct {
	FunctionName   string
	ParameterName  string
	ParameterClass string // I, E, C, or T
	AssociatedType string
	InternalType   string
}

type rcDescriptor struct {
	kind           string // "scalar" | "structure" | "table"
	name           string
	internalType   string // scalar
	internalLength int    // scalar
	typeName       string // structure/table
	fields         []rcDescriptor
	line           *rcDescriptor // table
}

// ResolvedClassic is a validated recursive classic xRFC plan.
type ResolvedClassic struct {
	FunctionName   string
	ParameterName  string
	ParameterClass string
	Kind           Kind
	root           rcDescriptor
}

type descriptorBudget struct {
	maxNodes, maxDepth int
	nodes              int
}

func (b *descriptorBudget) visit(depth int, path string) error {
	if depth > b.maxDepth {
		return fmt.Errorf("%w: %s descriptor depth exceeds %d", ErrRange, path, b.maxDepth)
	}
	b.nodes++
	if b.nodes > b.maxNodes {
		return fmt.Errorf("%w: %s descriptor node count exceeds %d", ErrRange, path, b.maxNodes)
	}
	return nil
}

func rcRequiredNode(graph metadata.Graph, typeName, kind, path string) (metadata.TypeNode, error) {
	node, ok := graph.Nodes[typeName]
	if !ok || node.Kind != kind || node.Name != typeName {
		return metadata.TypeNode{}, fmt.Errorf("%w: %s requires recursive %s node %s", ErrProtocol, path, kind, typeName)
	}
	return node, nil
}

func rcScalarDescriptor(field metadata.MetadataField, path string, budget *descriptorBudget, depth int) (rcDescriptor, error) {
	if err := budget.visit(depth, path); err != nil {
		return rcDescriptor{}, err
	}
	if field.Reference.Kind != "scalar" || field.Reference.InternalType != field.InternalType {
		return rcDescriptor{}, fmt.Errorf("%w: %s contains inconsistent scalar metadata", ErrProtocol, path)
	}
	if !supportedClassicScalars[field.InternalType] {
		return rcDescriptor{}, fmt.Errorf("%w: %s type %s is not implemented for the proven recursive xRFC subset", ErrProtocol, path, field.InternalType)
	}
	if field.UcLength < 0 {
		return rcDescriptor{}, fmt.Errorf("%w: %s contains invalid Unicode geometry", ErrProtocol, path)
	}
	if field.InternalType == "I" && field.UcLength != 4 {
		return rcDescriptor{}, fmt.Errorf("%w: %s INT4 must occupy four Unicode bytes", ErrProtocol, path)
	}
	if field.InternalType == "C" && field.UcLength&1 != 0 {
		return rcDescriptor{}, fmt.Errorf("%w: %s Unicode character width must be even", ErrProtocol, path)
	}
	return rcDescriptor{kind: "scalar", name: field.Name, internalType: field.InternalType, internalLength: field.UcLength}, nil
}

func rcDescriptorForReference(graph metadata.Graph, field metadata.MetadataField, path string, budget *descriptorBudget, depth int, active map[string]bool) (rcDescriptor, error) {
	if err := AssertXMLName(field.Name, path+" field name"); err != nil {
		return rcDescriptor{}, err
	}
	if field.Reference.Kind == "scalar" {
		return rcScalarDescriptor(field, path, budget, depth)
	}
	if field.Reference.Cyclic {
		return rcDescriptor{}, fmt.Errorf("%w: %s contains a cyclic recursive reference", ErrProtocol, path)
	}
	if field.Reference.Kind == "structure" && field.InternalType != "u" && field.InternalType != "v" {
		return rcDescriptor{}, fmt.Errorf("%w: %s contains inconsistent structure metadata", ErrProtocol, path)
	}
	if field.Reference.Kind == "table" && field.InternalType != "h" {
		return rcDescriptor{}, fmt.Errorf("%w: %s contains inconsistent table metadata", ErrProtocol, path)
	}
	return rcBuildNodeDescriptor(graph, field.Reference.TargetType, field.Reference.Kind, field.Name, path, budget, depth, active)
}

func rcBuildStructureDescriptor(graph metadata.Graph, node metadata.TypeNode, name, path string, budget *descriptorBudget, depth int, active map[string]bool) (rcDescriptor, error) {
	names := map[string]bool{}
	fields := make([]rcDescriptor, 0, len(node.Fields))
	for index, field := range node.Fields {
		fieldPath := path + "." + field.Name
		if field.Name == "" {
			return rcDescriptor{}, fmt.Errorf("%w: %s structure field name must not be empty", ErrProtocol, fieldPath)
		}
		if names[field.Name] {
			return rcDescriptor{}, fmt.Errorf("%w: %s contains duplicate field %s", ErrProtocol, path, field.Name)
		}
		if field.Position != index+1 {
			return rcDescriptor{}, fmt.Errorf("%w: %s has inconsistent field position", ErrProtocol, fieldPath)
		}
		names[field.Name] = true
		d, err := rcDescriptorForReference(graph, field, fieldPath, budget, depth+1, active)
		if err != nil {
			return rcDescriptor{}, err
		}
		fields = append(fields, d)
	}
	return rcDescriptor{kind: "structure", name: name, typeName: node.Name, fields: fields}, nil
}

func rcBuildTableDescriptor(graph metadata.Graph, node metadata.TypeNode, name, path string, budget *descriptorBudget, depth int, active map[string]bool) (rcDescriptor, error) {
	if len(node.Fields) != 1 {
		return rcDescriptor{}, fmt.Errorf("%w: %s table %s must contain one line descriptor", ErrProtocol, path, node.Name)
	}
	line := node.Fields[0]
	if line.Name != "" || line.Reference.Kind != "structure" || line.Reference.Cyclic {
		return rcDescriptor{}, fmt.Errorf("%w: %s table %s requires one non-cyclic structured line", ErrProtocol, path, node.Name)
	}
	lineDesc, err := rcBuildNodeDescriptor(graph, line.Reference.TargetType, "structure", "", path+"[]", budget, depth+1, active)
	if err != nil {
		return rcDescriptor{}, err
	}
	return rcDescriptor{kind: "table", name: name, typeName: node.Name, line: &lineDesc}, nil
}

func rcBuildNodeDescriptor(graph metadata.Graph, typeName, kind, name, path string, budget *descriptorBudget, depth int, active map[string]bool) (rcDescriptor, error) {
	if err := budget.visit(depth, path); err != nil {
		return rcDescriptor{}, err
	}
	if active[typeName] {
		return rcDescriptor{}, fmt.Errorf("%w: %s contains a cyclic recursive type %s", ErrProtocol, path, typeName)
	}
	node, err := rcRequiredNode(graph, typeName, kind, path)
	if err != nil {
		return rcDescriptor{}, err
	}
	active[typeName] = true
	defer delete(active, typeName)
	if kind == "structure" {
		return rcBuildStructureDescriptor(graph, node, name, path, budget, depth, active)
	}
	return rcBuildTableDescriptor(graph, node, name, path, budget, depth, active)
}

func rcMatchingParameter(graph metadata.Graph, id RecursiveClassicIdentity) (metadata.Parameter, error) {
	if graph.FunctionIdentity == nil || graph.FunctionIdentity.Name != id.FunctionName {
		return metadata.Parameter{}, fmt.Errorf("%w: recursive xRFC metadata identity does not match function %s", ErrProtocol, id.FunctionName)
	}
	var matches []metadata.Parameter
	for _, p := range graph.Parameters {
		if p.Name == id.ParameterName {
			matches = append(matches, p)
		}
	}
	if len(matches) != 1 {
		return metadata.Parameter{}, fmt.Errorf("%w: %s.%s recursive metadata contains %d matching parameters", ErrProtocol, id.FunctionName, id.ParameterName, len(matches))
	}
	p := matches[0]
	if p.FunctionName != id.FunctionName || p.ParameterClass != id.ParameterClass || p.AssociatedType != id.AssociatedType || p.InternalType != id.InternalType {
		return metadata.Parameter{}, fmt.Errorf("%w: %s.%s recursive descriptor does not match flat metadata", ErrProtocol, id.FunctionName, id.ParameterName)
	}
	return p, nil
}

// ResolveRecursiveClassic resolves one recursive classic xRFC parameter into a
// descriptor plan.
func ResolveRecursiveClassic(graph metadata.Graph, id RecursiveClassicIdentity, limits RecursiveClassicLimits) (ResolvedClassic, error) {
	if err := AssertXMLName(id.ParameterName, "xRFC parameter name"); err != nil {
		return ResolvedClassic{}, err
	}
	if id.FunctionName == "" {
		return ResolvedClassic{}, fmt.Errorf("%w: recursive xRFC function name must be non-empty", ErrType)
	}
	if !(id.ParameterClass == "I" || id.ParameterClass == "E" || id.ParameterClass == "C" || id.ParameterClass == "T") {
		return ResolvedClassic{}, fmt.Errorf("%w: recursive xRFC parameter class must be I, E, C, or T", ErrType)
	}
	if len(id.InternalType) != 1 {
		return ResolvedClassic{}, fmt.Errorf("%w: recursive xRFC internal type must contain one character", ErrType)
	}
	norm, err := normalizeRecursiveClassicLimits(limits)
	if err != nil {
		return ResolvedClassic{}, err
	}
	p, err := rcMatchingParameter(graph, id)
	if err != nil {
		return ResolvedClassic{}, err
	}
	if (p.Reference.Kind != "structure" && p.Reference.Kind != "table") || p.Reference.TargetType == "" {
		return ResolvedClassic{}, fmt.Errorf("%w: %s.%s is not a recursive structure or structured table", ErrProtocol, id.FunctionName, id.ParameterName)
	}
	if p.Reference.Cyclic {
		return ResolvedClassic{}, fmt.Errorf("%w: %s.%s contains a cyclic recursive reference", ErrProtocol, id.FunctionName, id.ParameterName)
	}
	budget := &descriptorBudget{maxNodes: norm.maxNodes, maxDepth: norm.maxDepth}
	root, err := rcBuildNodeDescriptor(graph, p.Reference.TargetType, p.Reference.Kind, id.ParameterName, id.FunctionName+"."+id.ParameterName, budget, 1, map[string]bool{})
	if err != nil {
		return ResolvedClassic{}, err
	}
	return ResolvedClassic{
		FunctionName:   id.FunctionName,
		ParameterName:  id.ParameterName,
		ParameterClass: id.ParameterClass,
		Kind:           Kind(p.Reference.Kind),
		root:           root,
	}, nil
}

// --- encode -----------------------------------------------------------------

type rcEncoder struct {
	b              strings.Builder
	limits         recursiveClassicLimits
	nodes          int
	rows           int
	parameterBytes int
}

func (e *rcEncoder) reserve(n int, path string) error {
	e.parameterBytes += n
	if e.parameterBytes > e.limits.classic.MaxParameterBytes {
		return fmt.Errorf("%w: %s xRFC XML exceeds %d bytes", ErrRange, path, e.limits.classic.MaxParameterBytes)
	}
	return nil
}

func (e *rcEncoder) visit(depth int, path string) error {
	if depth > e.limits.maxDepth {
		return fmt.Errorf("%w: %s value depth exceeds %d", ErrRange, path, e.limits.maxDepth)
	}
	e.nodes++
	if e.nodes > e.limits.maxNodes {
		return fmt.Errorf("%w: %s value node count exceeds %d", ErrRange, path, e.limits.maxNodes)
	}
	return nil
}

func rcInitialScalar(d rcDescriptor) any {
	switch d.internalType {
	case "I":
		return int32(0)
	case "C", "g":
		return ""
	default: // y
		return []byte{}
	}
}

func rcInitialValue(d rcDescriptor) any {
	switch d.kind {
	case "scalar":
		return rcInitialScalar(d)
	case "table":
		return []any{}
	default:
		m := map[string]any{}
		for _, f := range d.fields {
			m[f.name] = rcInitialValue(f)
		}
		return m
	}
}

func (e *rcEncoder) encodeScalar(d rcDescriptor, v any, path string) (int, error) {
	switch d.internalType {
	case "I":
		n, ok := asInt64(v)
		if !ok || n < -0x8000_0000 || n > 0x7fff_ffff {
			return 0, fmt.Errorf("%w: %s expects a signed 32-bit integer", ErrRange, path)
		}
		text := strconv.FormatInt(n, 10)
		contentLen, err := EscapedXMLByteLength(text, path)
		if err != nil {
			return 0, err
		}
		if contentLen > e.limits.classic.MaxCellBytes {
			return 0, fmt.Errorf("%w: %s XML value exceeds %d encoded bytes", ErrRange, path, e.limits.classic.MaxCellBytes)
		}
		if err := e.reserve(contentLen, path); err != nil {
			return 0, err
		}
		writeEscapedText(&e.b, text)
		return contentLen, nil
	case "C":
		s, ok := v.(string)
		if !ok {
			return 0, fmt.Errorf("%w: %s expects a string", ErrType, path)
		}
		if err := value.AssertUnicodeScalarText(s, path); err != nil {
			return 0, err
		}
		if utf16Len(s) > d.internalLength/2 {
			return 0, fmt.Errorf("%w: %s does not fit CHAR(%d)", ErrRange, path, d.internalLength/2)
		}
		return e.encodeText(s, path)
	case "g":
		s, ok := v.(string)
		if !ok {
			return 0, fmt.Errorf("%w: %s expects Unicode text", ErrType, path)
		}
		if err := value.AssertNulFreeUnicodeScalarText(s, path); err != nil {
			return 0, err
		}
		return e.encodeText(s, path)
	default: // y
		b, ok := v.([]byte)
		if !ok {
			return 0, fmt.Errorf("%w: %s expects bytes", ErrType, path)
		}
		contentLen := (len(b) + 2) / 3 * 4
		if contentLen > e.limits.classic.MaxCellBytes {
			return 0, fmt.Errorf("%w: %s base64 value exceeds %d encoded bytes", ErrRange, path, e.limits.classic.MaxCellBytes)
		}
		if err := e.reserve(contentLen, path); err != nil {
			return 0, err
		}
		e.b.WriteString(base64.StdEncoding.EncodeToString(b))
		return contentLen, nil
	}
}

func (e *rcEncoder) encodeText(text, path string) (int, error) {
	contentLen, err := EscapedXMLByteLength(text, path)
	if err != nil {
		return 0, err
	}
	if contentLen > e.limits.classic.MaxCellBytes {
		return 0, fmt.Errorf("%w: %s XML value exceeds %d encoded bytes", ErrRange, path, e.limits.classic.MaxCellBytes)
	}
	if err := e.reserve(contentLen, path); err != nil {
		return 0, err
	}
	writeEscapedText(&e.b, text)
	return contentLen, nil
}

func (e *rcEncoder) encodeStructure(d rcDescriptor, v any, path string, depth int) (int, error) {
	record, ok := v.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("%w: %s must be an object", ErrType, path)
	}
	known := map[string]bool{}
	for _, f := range d.fields {
		known[f.name] = true
	}
	for key := range record {
		if !known[key] {
			return 0, fmt.Errorf("%w: %s contains unknown field %s", ErrProtocol, path, key)
		}
	}
	content := 0
	for _, field := range d.fields {
		fieldPath := path + "." + field.name
		if err := e.reserve(OpenTagByteLength(field.name)+CloseTagByteLength(field.name), fieldPath); err != nil {
			return 0, err
		}
		e.b.WriteByte('<')
		e.b.WriteString(field.name)
		e.b.WriteByte('>')
		fv, supplied := record[field.name]
		if !supplied {
			fv = rcInitialValue(field)
		}
		childLen, err := e.encodeNode(field, fv, fieldPath, depth+1)
		if err != nil {
			return 0, err
		}
		e.b.WriteString("</")
		e.b.WriteString(field.name)
		e.b.WriteByte('>')
		content += OpenTagByteLength(field.name) + childLen + CloseTagByteLength(field.name)
		if content > e.limits.classic.MaxParameterBytes {
			return 0, fmt.Errorf("%w: %s xRFC XML exceeds %d bytes", ErrRange, path, e.limits.classic.MaxParameterBytes)
		}
	}
	if content > e.limits.classic.MaxRowBytes {
		return 0, fmt.Errorf("%w: %s XML row exceeds %d encoded bytes", ErrRange, path, e.limits.classic.MaxRowBytes)
	}
	return content, nil
}

func (e *rcEncoder) encodeTable(d rcDescriptor, v any, path string, depth int) (int, error) {
	rows, ok := v.([]any)
	if !ok {
		return 0, fmt.Errorf("%w: %s expects an array of rows", ErrType, path)
	}
	if e.rows+len(rows) > e.limits.classic.MaxRows {
		return 0, fmt.Errorf("%w: %s row count exceeds %d", ErrRange, path, e.limits.classic.MaxRows)
	}
	e.rows += len(rows)
	content := 0
	for index, row := range rows {
		rowPath := fmt.Sprintf("%s[%d]", path, index)
		if err := e.reserve(rcItemWrapperByteLength, rowPath); err != nil {
			return 0, err
		}
		e.b.WriteString("<item>")
		rowLen, err := e.encodeStructure(*d.line, row, rowPath, depth+1)
		if err != nil {
			return 0, err
		}
		e.b.WriteString("</item>")
		rowByteLen := rcItemWrapperByteLength + rowLen
		if rowByteLen > e.limits.classic.MaxRowBytes {
			return 0, fmt.Errorf("%w: %s XML row exceeds %d encoded bytes", ErrRange, rowPath, e.limits.classic.MaxRowBytes)
		}
		content += rowByteLen
		if content > e.limits.classic.MaxParameterBytes {
			return 0, fmt.Errorf("%w: %s xRFC XML exceeds %d bytes", ErrRange, path, e.limits.classic.MaxParameterBytes)
		}
	}
	return content, nil
}

func (e *rcEncoder) encodeNode(d rcDescriptor, v any, path string, depth int) (int, error) {
	if err := e.visit(depth, path); err != nil {
		return 0, err
	}
	switch d.kind {
	case "scalar":
		return e.encodeScalar(d, v, path)
	case "structure":
		return e.encodeStructure(d, v, path, depth)
	default:
		return e.encodeTable(d, v, path, depth)
	}
}

// EncodeRecursiveClassic encodes a value against a resolved plan. For a
// structure root, value is map[string]any; for a table root, []any.
func EncodeRecursiveClassic(resolved ResolvedClassic, v any, limits RecursiveClassicLimits) ([]byte, error) {
	norm, err := normalizeRecursiveClassicLimits(limits)
	if err != nil {
		return nil, err
	}
	e := &rcEncoder{limits: norm}
	name := resolved.ParameterName
	if err := e.reserve(OpenTagByteLength(name)+CloseTagByteLength(name), name); err != nil {
		return nil, err
	}
	e.b.WriteByte('<')
	e.b.WriteString(name)
	e.b.WriteByte('>')
	if _, err := e.encodeNode(resolved.root, v, resolved.FunctionName+"."+name, 1); err != nil {
		return nil, err
	}
	e.b.WriteString("</")
	e.b.WriteString(name)
	e.b.WriteByte('>')
	return []byte(e.b.String()), nil
}

// --- decode -----------------------------------------------------------------

type rcDecoder struct {
	p      *parser
	limits recursiveClassicLimits
	nodes  int
	rows   int
}

func (d *rcDecoder) visit(depth int, path string) error {
	if depth > d.limits.maxDepth {
		return fmt.Errorf("%w: %s value depth exceeds %d", ErrRange, path, d.limits.maxDepth)
	}
	d.nodes++
	if d.nodes > d.limits.maxNodes {
		return fmt.Errorf("%w: %s value node count exceeds %d", ErrRange, path, d.limits.maxNodes)
	}
	return nil
}

func (d *rcDecoder) decodeScalar(desc rcDescriptor, text, path string) (any, error) {
	switch desc.internalType {
	case "I":
		if !canonicalInteger.MatchString(text) {
			return nil, fmt.Errorf("%w: %s contains a non-canonical INT4 value", ErrProtocol, path)
		}
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil || n < -0x8000_0000 || n > 0x7fff_ffff {
			return nil, fmt.Errorf("%w: %s INT4 value is out of range", ErrRange, path)
		}
		return int32(n), nil
	case "C":
		if err := value.AssertUnicodeScalarText(text, path); err != nil {
			return nil, err
		}
		if utf16Len(text) > desc.internalLength/2 {
			return nil, fmt.Errorf("%w: %s does not fit CHAR(%d)", ErrRange, path, desc.internalLength/2)
		}
		return text, nil
	case "g":
		if err := value.AssertNulFreeUnicodeScalarText(text, path); err != nil {
			return nil, err
		}
		return text, nil
	default: // y
		return DecodeBase64(text, path, d.limits.classic.MaxCellBytes/4*3)
	}
}

func (d *rcDecoder) parseNode(desc rcDescriptor, path string, depth int, closingTag string) (any, error) {
	if err := d.visit(depth, path); err != nil {
		return nil, err
	}
	switch desc.kind {
	case "scalar":
		text, err := d.p.cell(path)
		if err != nil {
			return nil, err
		}
		return d.decodeScalar(desc, text, path)
	case "structure":
		start := d.p.offset
		result := make(map[string]any, len(desc.fields))
		for _, field := range desc.fields {
			fieldPath := path + "." + field.name
			if err := d.p.open(field.name); err != nil {
				return nil, err
			}
			v, err := d.parseNode(field, fieldPath, depth+1, field.name)
			if err != nil {
				return nil, err
			}
			result[field.name] = v
			if err := d.p.close(field.name); err != nil {
				return nil, err
			}
		}
		if d.p.offset-start > d.limits.classic.MaxRowBytes {
			return nil, fmt.Errorf("%w: %s XML row exceeds %d encoded bytes", ErrRange, path, d.limits.classic.MaxRowBytes)
		}
		return result, nil
	default: // table
		rows := []any{}
		for !d.p.startsWithTag(closingTag, true) {
			d.rows++
			if d.rows > d.limits.classic.MaxRows {
				return nil, fmt.Errorf("%w: %s row count exceeds %d", ErrRange, path, d.limits.classic.MaxRows)
			}
			rowPath := fmt.Sprintf("%s[%d]", path, len(rows))
			start := d.p.offset
			if err := d.p.open("item"); err != nil {
				return nil, err
			}
			row, err := d.parseNode(*desc.line, rowPath, depth+1, "item")
			if err != nil {
				return nil, err
			}
			if err := d.p.close("item"); err != nil {
				return nil, err
			}
			if d.p.offset-start > d.limits.classic.MaxRowBytes {
				return nil, fmt.Errorf("%w: %s XML row exceeds %d encoded bytes", ErrRange, rowPath, d.limits.classic.MaxRowBytes)
			}
			rows = append(rows, row)
		}
		return rows, nil
	}
}

// DecodeRecursiveClassic decodes bytes against a resolved plan. For a structure
// root it returns map[string]any; for a table root, []any.
func DecodeRecursiveClassic(resolved ResolvedClassic, v []byte, limits RecursiveClassicLimits) (any, error) {
	norm, err := normalizeRecursiveClassicLimits(limits)
	if err != nil {
		return nil, err
	}
	name := resolved.ParameterName
	if len(v) == 0 || len(v) > norm.classic.MaxParameterBytes {
		return nil, fmt.Errorf("%w: %s xRFC XML must contain 1..%d bytes", ErrRange, name, norm.classic.MaxParameterBytes)
	}
	if len(v) >= 3 && v[0] == 0xef && v[1] == 0xbb && v[2] == 0xbf {
		return nil, fmt.Errorf("%w: %s xRFC XML must not contain a UTF-8 BOM", ErrProtocol, name)
	}
	text, err := decodeUTF8(v, name+" xRFC XML")
	if err != nil {
		return nil, err
	}
	dec := &rcDecoder{p: &parser{text: text, limits: norm.classic}, limits: norm}
	if err := dec.p.open(name); err != nil {
		return nil, err
	}
	result, err := dec.parseNode(resolved.root, resolved.FunctionName+"."+name, 1, name)
	if err != nil {
		return nil, err
	}
	if err := dec.p.close(name); err != nil {
		return nil, err
	}
	if err := dec.p.finish(); err != nil {
		return nil, err
	}
	return result, nil
}

// InitialRecursiveClassicValue returns the ABAP-initial value for a plan.
func InitialRecursiveClassicValue(resolved ResolvedClassic) any {
	return rcInitialValue(resolved.root)
}
