// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/metadata/recursive-metadata.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. The input is typed Go
// row structs (values are wire strings) rather than caller-supplied JavaScript
// objects, so the hostile-object hardening — proxy/accessor/prototype/symbol-key
// rejection and the `ImmutableMetadataMap` wrapper — collapses; the semantic
// validation (allowed/required fields, text/integer/flag checks, and the row/
// node/edge/depth/property/byte budgets) is preserved, as is the graph
// algorithm (type grouping, offset/length geometry, Kosaraju SCC cycle
// detection, and topological depth). Numerous `reject()` sites are expressed
// with a panic carrying *RecursiveMetadataError, recovered at Normalize's
// boundary and returned as an error. See docs/provenance.md.

// Package metadata normalizes the RFC_METADATA_GET type closure into a bounded,
// cyclic-aware identity graph.
package metadata

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Limits bounds the recursive-metadata normalizer.
type Limits struct {
	MaxRows       int
	MaxNodes      int
	MaxEdges      int
	MaxDepth      int
	MaxProperties int
	MaxBytes      int
}

var defaultLimits = Limits{MaxRows: 20_000, MaxNodes: 4_096, MaxEdges: 20_000, MaxDepth: 64, MaxProperties: 400_000, MaxBytes: 8 * 1024 * 1024}
var absoluteLimits = Limits{MaxRows: 100_000, MaxNodes: 20_000, MaxEdges: 100_000, MaxDepth: 256, MaxProperties: 2_000_000, MaxBytes: 32 * 1024 * 1024}

// Options configures normalization. Limits nil fields take the default; a
// value above the absolute ceiling is rejected.
type Options struct {
	Limits        *Limits
	RootTypeNames []string
}

// Reference is a decoded type-graph edge (or scalar leaf).
type Reference struct {
	Kind         string // "scalar" | "structure" | "table"
	InternalType string // scalar only
	TargetType   string // structure/table only
	Cyclic       bool   // structure/table only
}

// MetadataField is one field of a structure/table node.
type MetadataField struct {
	Name           string
	Position       int
	ComponentType  string
	AssociatedType string
	DataType       string
	InternalType   string
	Description    string
	Decimals       int
	NucOffset      int
	UcOffset       int
	NucLength      int
	UcLength       int
	Reference      Reference
}

// TypeNode is one node of the type graph.
type TypeNode struct {
	Name      string
	Kind      string // "structure" | "table" | "scalar"
	NucLength int
	UcLength  int
	Timestamp string
	Fields    []MetadataField
}

// FunctionIdentity identifies the RFM the graph describes.
type FunctionIdentity struct {
	Name                  string
	RemoteBasxmlSupported bool
	GenerationToken       string
}

// ParameterReference is a parameter's edge into the type graph.
type ParameterReference struct {
	Kind                   string // "scalar" | "structure" | "table" | "exception"
	InternalType           string
	TargetType             string
	Cyclic                 bool
	HasScalarLine          bool // table-of-scalar-line
	ScalarLineInternalType string
}

// Parameter is one function parameter.
type Parameter struct {
	FunctionName   string
	Name           string
	ParameterClass string
	Position       int
	AssociatedType string
	FieldPath      string
	InternalType   string
	InternalLength int
	Decimals       int
	DefaultValue   string
	ParameterText  string
	Optional       bool
	Reference      ParameterReference
}

// Cycle is a strongly connected component of two-plus types (or a self-loop).
type Cycle struct {
	ID        string
	TypeNames []string
}

// Statistics summarizes what was normalized.
type Statistics struct {
	RowCount      int
	NodeCount     int
	EdgeCount     int
	PropertyCount int
	ByteCount     int
	MaximumDepth  int
}

// Graph is the normalized metadata identity graph.
type Graph struct {
	Version          int
	FunctionIdentity *FunctionIdentity
	Nodes            map[string]TypeNode
	Parameters       []Parameter
	RootTypeNames    []string
	Cycles           []Cycle
	Limits           Limits
	Statistics       Statistics
}

// RecursiveMetadataError reports a rejected metadata graph, carrying a stable
// code and the path at which it was rejected.
type RecursiveMetadataError struct {
	Code string
	Path string
}

func (e *RecursiveMetadataError) Error() string {
	return fmt.Sprintf("recursive metadata rejected: %s at %s", e.Code, e.Path)
}

func reject(code, path string) {
	panic(&RecursiveMetadataError{Code: code, Path: path})
}

// ---- typed input rows (values are wire strings) ----

// FunctionRow is a FUNCTIONNAMES row.
type FunctionRow struct{ FunctionName, BasxmlSupported, UDat, UTime string }

// TypeRowInput is a DATATYPESCONT row.
type TypeRowInput struct {
	TypeName, FieldName, CompType, FieldType, DataType     string
	TabLength, TabLengthUC, Description, Decimals          string
	IntType, Offset, OffsetUC, IntLen, IntLenUC, Timestamp string
}

// IndirectRowInput is an INDIRECTTYPES row.
type IndirectRowInput struct{ TabName, FieldName, FieldType string }

// ParameterRowInput is a PARAMETERS row.
type ParameterRowInput struct {
	FuncName, ParamClass, Parameter, TabName, FieldName, Exid           string
	Position, Offset, IntLength, Decimals, Default, ParamText, Optional string
}

// Input is the decoded RFC_METADATA_GET output. FunctionNames and Parameters
// are optional; a nil pointer means the table was absent (distinct from a
// present-but-empty slice).
type Input struct {
	FunctionNames *[]FunctionRow
	DataTypesCont []TypeRowInput
	IndirectTypes []IndirectRowInput
	Parameters    *[]ParameterRowInput
}

type budget struct {
	limits     Limits
	rows       int
	properties int
	bytes      int
}

func (b *budget) addRows(count int, path string) {
	b.rows += count
	if b.rows > b.limits.MaxRows {
		reject("ROW_LIMIT", path)
	}
}
func (b *budget) addProperties(count int, path string) {
	b.properties += count
	if b.properties > b.limits.MaxProperties {
		reject("PROPERTY_LIMIT", path)
	}
}
func (b *budget) addBytes(count int, path string) {
	b.bytes += count
	if b.bytes > b.limits.MaxBytes {
		reject("BYTE_LIMIT", path)
	}
}

func codeUnitLen(s string) int { return len(utf16.Encode([]rune(s))) }

var (
	controlOrHigh = func(s string, asciiOnly bool) bool {
		for i := 0; i < len(s); i++ {
			b := s[i]
			if b < 0x20 || b == 0x7f {
				return true
			}
		}
		if asciiOnly {
			for i := 0; i < len(s); i++ {
				if s[i] > 0x7e {
					return true
				}
			}
		}
		return false
	}
	numericPattern   = regexp.MustCompile(`^\d+$`)
	timestampPattern = regexp.MustCompile(`^\d{14}$`)
	datePattern      = regexp.MustCompile(`^\d{8}$`)
	timePattern      = regexp.MustCompile(`^\d{6}$`)
	paramClassP      = regexp.MustCompile(`^[IECXT]$`)
)

// text validates one string field and accounts its bytes.
func text(value, path string, b *budget, maximum int, allowEmpty, ascii bool) string {
	if codeUnitLen(value) > maximum || (!allowEmpty && len(value) == 0) || controlOrHigh(value, ascii) {
		reject("INVALID_TEXT", path)
	}
	b.addBytes(len(value), path)
	return value
}

func integer(value, path string, b *budget) int {
	if !numericPattern.MatchString(value) {
		reject("INVALID_INTEGER", path)
	}
	b.addBytes(len(value), path)
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		reject("INVALID_INTEGER", path)
	}
	return n
}

func flag(value, path string, b *budget) bool {
	v := text(value, path, b, 1, true, true)
	if v != "" && v != "X" {
		reject("INVALID_FLAG", path)
	}
	return v == "X"
}

func metadataName(value, path string, b *budget, allowEmpty bool) string {
	return text(value, path, b, 30, allowEmpty, true)
}

func resolveLimits(o *Limits) Limits {
	if o == nil {
		return defaultLimits
	}
	result := defaultLimits
	check := func(name string, v, abs int, dst *int) {
		if v < 0 || v > abs {
			reject("INVALID_LIMIT", "options.limits."+name)
		}
		*dst = v
	}
	// A zero field in the caller's Limits means "use it" (0 is a valid, tested
	// limit); callers pass a full Limits value, so every field is applied.
	check("maxRows", o.MaxRows, absoluteLimits.MaxRows, &result.MaxRows)
	check("maxNodes", o.MaxNodes, absoluteLimits.MaxNodes, &result.MaxNodes)
	check("maxEdges", o.MaxEdges, absoluteLimits.MaxEdges, &result.MaxEdges)
	check("maxDepth", o.MaxDepth, absoluteLimits.MaxDepth, &result.MaxDepth)
	check("maxProperties", o.MaxProperties, absoluteLimits.MaxProperties, &result.MaxProperties)
	check("maxBytes", o.MaxBytes, absoluteLimits.MaxBytes, &result.MaxBytes)
	return result
}

// ---- internal working types ----

type typeRow struct {
	typeName, fieldName, componentType, associatedType, dataType string
	nucTotalLength, ucTotalLength                                int
	description                                                  string
	decimals                                                     int
	internalType                                                 string
	nucOffset, ucOffset, nucLength, ucLength                     int
	timestamp                                                    string
}

type indirectRow struct{ tableName, fieldPath, targetType string }

type parameterRow struct {
	functionName, parameterClass, parameterName, tableName, fieldPath, internalType string
	position, internalLength, decimals                                              int
	defaultValue, parameterText                                                     string
	optional                                                                        bool
}

type provReference struct {
	kind         string // "scalar" | "structure" | "table"
	internalType string
	targetType   string
}

type provField struct {
	name                                     string
	position                                 int
	componentType                            string
	associatedType                           string
	dataType                                 string
	internalType                             string
	description                              string
	decimals                                 int
	nucOffset, ucOffset, nucLength, ucLength int
	reference                                provReference
}

type provNode struct {
	name      string
	kind      string
	nucLength int
	ucLength  int
	timestamp string
	fields    []provField
}

func snapshotFunctionIdentity(rows *[]FunctionRow, b *budget) *FunctionIdentity {
	if rows == nil {
		return nil
	}
	values := *rows
	b.addRows(len(values), "FUNCTIONNAMES")
	if len(values) == 0 {
		return nil
	}
	if len(values) != 1 {
		reject("MULTIPLE_FUNCTION_IDENTITIES", "FUNCTIONNAMES")
	}
	row := values[0]
	path := "FUNCTIONNAMES[0]"
	b.addProperties(4, path)
	date := text(row.UDat, path+".UDAT", b, 8, false, true)
	tm := text(row.UTime, path+".UTIME", b, 6, false, true)
	if !datePattern.MatchString(date) {
		reject("INVALID_DATE", path+".UDAT")
	}
	if !timePattern.MatchString(tm) {
		reject("INVALID_TIME", path+".UTIME")
	}
	return &FunctionIdentity{
		Name:                  metadataName(row.FunctionName, path+".FUNCTIONNAME", b, false),
		RemoteBasxmlSupported: flag(row.BasxmlSupported, path+".BASXML_SUPPORTED", b),
		GenerationToken:       "function:" + date + ":" + tm,
	}
}

func snapshotTypeRows(rows []TypeRowInput, b *budget) []typeRow {
	b.addRows(len(rows), "DATATYPESCONT")
	result := make([]typeRow, 0, len(rows))
	for i, row := range rows {
		path := fmt.Sprintf("DATATYPESCONT[%d]", i)
		b.addProperties(15, path)
		timestamp := text(row.Timestamp, path+".TIMESTAMP", b, 14, false, true)
		if !timestampPattern.MatchString(timestamp) {
			reject("INVALID_TIMESTAMP", path+".TIMESTAMP")
		}
		result = append(result, typeRow{
			typeName:       metadataName(row.TypeName, path+".TYPENAME", b, false),
			fieldName:      metadataName(row.FieldName, path+".FIELDNAME", b, true),
			componentType:  text(row.CompType, path+".COMPTYPE", b, 1, true, true),
			associatedType: metadataName(row.FieldType, path+".FIELDTYPE", b, true),
			dataType:       text(row.DataType, path+".DATATYPE", b, 8, false, true),
			nucTotalLength: integer(row.TabLength, path+".TABLENGTH", b),
			ucTotalLength:  integer(row.TabLengthUC, path+".TABLENGTH_UC", b),
			description:    text(row.Description, path+".DESCRIPTION", b, 60, true, false),
			decimals:       integer(row.Decimals, path+".DECIMALS", b),
			internalType:   text(row.IntType, path+".INTTYPE", b, 1, false, true),
			nucOffset:      integer(row.Offset, path+".OFFSET", b),
			ucOffset:       integer(row.OffsetUC, path+".OFFSET_UC", b),
			nucLength:      integer(row.IntLen, path+".INTLEN", b),
			ucLength:       integer(row.IntLenUC, path+".INTLEN_UC", b),
			timestamp:      timestamp,
		})
	}
	return result
}

func snapshotIndirectRows(rows []IndirectRowInput, b *budget) []indirectRow {
	b.addRows(len(rows), "INDIRECTTYPES")
	result := make([]indirectRow, 0, len(rows))
	for i, row := range rows {
		path := fmt.Sprintf("INDIRECTTYPES[%d]", i)
		b.addProperties(3, path)
		result = append(result, indirectRow{
			tableName:  metadataName(row.TabName, path+".TABNAME", b, false),
			fieldPath:  metadataName(row.FieldName, path+".FIELDNAME", b, false),
			targetType: metadataName(row.FieldType, path+".FIELDTYPE", b, false),
		})
	}
	return result
}

func snapshotParameterRows(rows *[]ParameterRowInput, b *budget) []parameterRow {
	if rows == nil {
		return nil
	}
	values := *rows
	b.addRows(len(values), "PARAMETERS")
	result := make([]parameterRow, 0, len(values))
	for i, row := range values {
		path := fmt.Sprintf("PARAMETERS[%d]", i)
		b.addProperties(13, path)
		parameterClass := text(row.ParamClass, path+".PARAMCLASS", b, 1, false, true)
		if !paramClassP.MatchString(parameterClass) {
			reject("INVALID_PARAMETER_CLASS", path+".PARAMCLASS")
		}
		internalType := text(row.Exid, path+".EXID", b, 1, true, true)
		if parameterClass != "X" && len(internalType) == 0 {
			reject("INVALID_TEXT", path+".EXID")
		}
		integer(row.Offset, path+".OFFSET", b)
		defaultValue := text(row.Default, path+".DEFAULT", b, 21, true, false)
		parameterText := text(row.ParamText, path+".PARAMTEXT", b, 79, true, false)
		position := integer(row.Position, path+".POSITION", b)
		result = append(result, parameterRow{
			functionName:   metadataName(row.FuncName, path+".FUNCNAME", b, false),
			parameterClass: parameterClass,
			parameterName:  metadataName(row.Parameter, path+".PARAMETER", b, false),
			tableName:      metadataName(row.TabName, path+".TABNAME", b, true),
			fieldPath:      metadataName(row.FieldName, path+".FIELDNAME", b, true),
			internalType:   internalType,
			position:       position,
			internalLength: integer(row.IntLength, path+".INTLENGTH", b),
			decimals:       integer(row.Decimals, path+".DECIMALS", b),
			defaultValue:   defaultValue,
			parameterText:  parameterText,
			optional:       flag(row.Optional, path+".OPTIONAL", b),
		})
	}
	return result
}

func validateFunctionIdentity(identity *FunctionIdentity, identityWasProvided bool, parameters []parameterRow) {
	names := map[string]bool{}
	for _, p := range parameters {
		names[p.functionName] = true
	}
	if len(names) > 1 {
		reject("MULTIPLE_FUNCTIONS", "PARAMETERS")
	}
	if identityWasProvided && identity == nil && len(names) > 0 {
		reject("MISSING_FUNCTION_IDENTITY", "FUNCTIONNAMES")
	}
	if identity != nil && len(names) == 1 && !names[identity.Name] {
		reject("FOREIGN_FUNCTION_REFERENCE", "PARAMETERS")
	}
}

func classifyReference(internalType, associatedType, path string) provReference {
	if internalType == "u" || internalType == "v" || internalType == "h" {
		if len(associatedType) == 0 {
			reject("MISSING_ASSOCIATED_TYPE", path)
		}
		kind := "structure"
		if internalType == "h" {
			kind = "table"
		}
		return provReference{kind: kind, internalType: internalType, targetType: associatedType}
	}
	return provReference{kind: "scalar", internalType: internalType}
}

func safeEnd(offset, length int, path string) int { return offset + length }

func buildNodes(rows []typeRow, limits Limits) []provNode {
	type group struct {
		name string
		rows []typeRow
	}
	var grouped []*group
	closed := map[string]bool{}
	var current *group
	for i, row := range rows {
		if current == nil || current.name != row.typeName {
			if current != nil {
				closed[current.name] = true
			}
			if closed[row.typeName] {
				reject("NONCONTIGUOUS_TYPE", fmt.Sprintf("DATATYPESCONT[%d].TYPENAME", i))
			}
			current = &group{name: row.typeName}
			grouped = append(grouped, current)
		}
		current.rows = append(current.rows, row)
	}
	if len(grouped) > limits.MaxNodes {
		reject("NODE_LIMIT", "DATATYPESCONT")
	}

	var nodes []provNode
	fieldEdges := 0
	for groupIndex, g := range grouped {
		first := g.rows[0]
		blankCount := 0
		for _, r := range g.rows {
			if len(r.fieldName) == 0 {
				blankCount++
			}
		}
		anonymous := blankCount > 0
		if anonymous && (len(g.rows) != 1 || blankCount != 1) {
			reject("INVALID_TABLE_SHAPE", fmt.Sprintf("DATATYPESCONT:%d", groupIndex))
		}
		names := map[string]bool{}
		var fields []provField
		var anonymousReference provReference
		anonymousScalar := false
		if anonymous {
			anonymousReference = classifyReference(first.internalType, first.associatedType, fmt.Sprintf("DATATYPESCONT:%d:0.INTTYPE", groupIndex))
			anonymousScalar = anonymousReference.kind == "scalar"
		}
		effectiveNucTotal := first.nucTotalLength
		effectiveUcTotal := first.ucTotalLength
		if anonymousScalar && first.nucTotalLength == 0 {
			effectiveNucTotal = safeEnd(first.nucOffset, first.nucLength, fmt.Sprintf("DATATYPESCONT:%d:0", groupIndex))
		}
		if anonymousScalar && first.ucTotalLength == 0 {
			effectiveUcTotal = safeEnd(first.ucOffset, first.ucLength, fmt.Sprintf("DATATYPESCONT:%d:0", groupIndex))
		}
		previousNucEnd, previousUcEnd := 0, 0
		for fieldIndex, row := range g.rows {
			path := fmt.Sprintf("DATATYPESCONT:%d:%d", groupIndex, fieldIndex)
			if row.nucTotalLength != first.nucTotalLength || row.ucTotalLength != first.ucTotalLength {
				reject("INCONSISTENT_TOTAL_LENGTH", path)
			}
			if row.timestamp != first.timestamp {
				reject("INCONSISTENT_TIMESTAMP", path)
			}
			if names[row.fieldName] {
				reject("DUPLICATE_FIELD", path)
			}
			nucEnd := safeEnd(row.nucOffset, row.nucLength, path)
			ucEnd := safeEnd(row.ucOffset, row.ucLength, path)
			if row.nucOffset < previousNucEnd || row.ucOffset < previousUcEnd || nucEnd > effectiveNucTotal || ucEnd > effectiveUcTotal {
				reject("INVALID_GEOMETRY", path)
			}
			names[row.fieldName] = true
			previousNucEnd, previousUcEnd = nucEnd, ucEnd
			reference := classifyReference(row.internalType, row.associatedType, path+".INTTYPE")
			if reference.kind != "scalar" {
				fieldEdges++
			}
			fields = append(fields, provField{
				name: row.fieldName, position: fieldIndex + 1, componentType: row.componentType,
				associatedType: row.associatedType, dataType: row.dataType, internalType: row.internalType,
				description: row.description, decimals: row.decimals,
				nucOffset: row.nucOffset, ucOffset: row.ucOffset, nucLength: row.nucLength, ucLength: row.ucLength,
				reference: reference,
			})
		}
		if fieldEdges > limits.MaxEdges {
			reject("EDGE_LIMIT", "DATATYPESCONT")
		}
		kind := "structure"
		if anonymous {
			if fields[0].reference.kind == "scalar" {
				kind = "scalar"
			} else {
				kind = "table"
			}
		}
		nodes = append(nodes, provNode{name: g.name, kind: kind, nucLength: effectiveNucTotal, ucLength: effectiveUcTotal, timestamp: first.timestamp, fields: fields})
	}
	return nodes
}

func refineTableNodeKinds(nodes []provNode, parameters []parameterRow, indirectRows []indirectRow) []provNode {
	requiredTables := map[string]bool{}
	indirectTargets := map[string]string{}
	ambiguous := map[string]bool{}
	for _, row := range indirectRows {
		key := row.tableName + "\x00" + row.fieldPath
		if _, ok := indirectTargets[key]; ok {
			ambiguous[key] = true
		} else {
			indirectTargets[key] = row.targetType
		}
	}
	for _, node := range nodes {
		for _, field := range node.fields {
			if field.reference.kind == "table" {
				requiredTables[field.reference.targetType] = true
			}
		}
	}
	for _, p := range parameters {
		if p.internalType != "h" {
			continue
		}
		if strings.Contains(p.fieldPath, "-") {
			key := p.tableName + "\x00" + p.fieldPath
			if target, ok := indirectTargets[key]; ok && !ambiguous[key] {
				requiredTables[target] = true
			}
		} else if len(p.fieldPath) == 0 && len(p.tableName) > 0 {
			requiredTables[p.tableName] = true
		}
	}
	out := make([]provNode, len(nodes))
	for i, node := range nodes {
		if node.kind == "scalar" && requiredTables[node.name] {
			node.kind = "table"
		}
		out[i] = node
	}
	return out
}

func validateNodeTargets(nodes []provNode) {
	byName := map[string]provNode{}
	for _, n := range nodes {
		byName[n.name] = n
	}
	for nodeIndex, node := range nodes {
		for fieldIndex, field := range node.fields {
			ref := field.reference
			if ref.kind == "scalar" {
				continue
			}
			target, ok := byName[ref.targetType]
			if !ok {
				reject("FOREIGN_TYPE_REFERENCE", fmt.Sprintf("node:%d:%d", nodeIndex, fieldIndex))
			}
			if target.kind != ref.kind {
				reject("REFERENCE_KIND_MISMATCH", fmt.Sprintf("node:%d:%d", nodeIndex, fieldIndex))
			}
		}
	}
}

type provParamRef struct {
	kind         string // scalar|structure|table|scalar-table|exception
	internalType string
	targetType   string
}

type provParameter struct {
	functionName, name, parameterClass      string
	position                                int
	associatedType, fieldPath, internalType string
	internalLength, decimals                int
	defaultValue, parameterText             string
	optional                                bool
	reference                               provParamRef
}

func indirectMap(rows []indirectRow) map[string]indirectRow {
	out := map[string]indirectRow{}
	for i, row := range rows {
		if !strings.Contains(row.fieldPath, "-") {
			reject("INVALID_INDIRECT_PATH", fmt.Sprintf("INDIRECTTYPES[%d].FIELDNAME", i))
		}
		key := row.tableName + "\x00" + row.fieldPath
		if _, ok := out[key]; ok {
			reject("DUPLICATE_INDIRECT_TYPE", fmt.Sprintf("INDIRECTTYPES[%d]", i))
		}
		out[key] = row
	}
	return out
}

func resolveFieldPath(node provNode, fieldPath string, byName map[string]provNode, fieldsByNode map[string]map[string]provField, path string) provReference {
	segments := strings.Split(fieldPath, "-")
	current := node
	var reference provReference
	for index, segment := range segments {
		if len(segment) == 0 {
			reject("INVALID_FIELD_PATH", path)
		}
		field, ok := fieldsByNode[current.name][segment]
		if !ok {
			reject("FOREIGN_FIELD_REFERENCE", path)
		}
		reference = field.reference
		if index == len(segments)-1 {
			return reference
		}
		if reference.kind != "structure" {
			reject("INVALID_FIELD_PATH", path)
		}
		target, ok := byName[reference.targetType]
		if !ok || target.kind != "structure" {
			reject("FOREIGN_TYPE_REFERENCE", path)
		}
		current = target
	}
	return reference
}

func provisionalParameters(rows []parameterRow, indirectRows []indirectRow, nodes []provNode, limits Limits) ([]provParameter, []string, int) {
	byName := map[string]provNode{}
	fieldsByNode := map[string]map[string]provField{}
	for _, node := range nodes {
		byName[node.name] = node
		fm := map[string]provField{}
		for _, f := range node.fields {
			fm[f.name] = f
		}
		fieldsByNode[node.name] = fm
	}
	indirect := indirectMap(indirectRows)
	consumedIndirect := map[string]bool{}
	var result []provParameter
	var roots []string
	rootSet := map[string]bool{}
	parameters := map[string]bool{}
	edgeCount := 0
	for _, node := range nodes {
		for _, f := range node.fields {
			if f.reference.kind != "scalar" {
				edgeCount++
			}
		}
	}
	addRoot := func(name string) {
		if !rootSet[name] {
			rootSet[name] = true
			roots = append(roots, name)
		}
	}

	for index, row := range rows {
		path := fmt.Sprintf("PARAMETERS[%d]", index)
		parameterKey := row.functionName + "\x00" + row.parameterName
		if parameters[parameterKey] {
			reject("DUPLICATE_PARAMETER", path)
		}
		parameters[parameterKey] = true

		var reference provParamRef
		if row.parameterClass == "X" {
			reference = provParamRef{kind: "exception"}
		} else {
			// mappedTarget resolves an indirect (path-with-dash) reference.
			mappedTarget := func() (provNode, bool) {
				if !strings.Contains(row.fieldPath, "-") {
					return provNode{}, false
				}
				if len(row.tableName) == 0 {
					reject("MISSING_ASSOCIATED_TYPE", path+".TABNAME")
				}
				key := row.tableName + "\x00" + row.fieldPath
				mapping, ok := indirect[key]
				if !ok {
					reject("MISSING_INDIRECT_TYPE", path)
				}
				consumedIndirect[key] = true
				target, ok := byName[mapping.targetType]
				if !ok {
					reject("FOREIGN_TYPE_REFERENCE", path)
				}
				addRoot(mapping.targetType)
				return target, true
			}
			directFieldReference := func() (provReference, bool) {
				if len(row.fieldPath) == 0 || strings.Contains(row.fieldPath, "-") {
					return provReference{}, false
				}
				if len(row.tableName) == 0 {
					reject("MISSING_ASSOCIATED_TYPE", path+".TABNAME")
				}
				owner, ok := byName[row.tableName]
				if !ok {
					reject("FOREIGN_TYPE_REFERENCE", path)
				}
				addRoot(row.tableName)
				return resolveFieldPath(owner, row.fieldPath, byName, fieldsByNode, path), true
			}
			scalarLeaf := func(n provNode) bool {
				return len(n.fields) == 1 && n.fields[0].name == "" && n.fields[0].reference.kind == "scalar"
			}

			scalarTable := row.parameterClass == "T" && row.internalType != "u" && row.internalType != "v" && row.internalType != "h"
			isTable := !scalarTable && (row.parameterClass == "T" || row.internalType == "h")
			isStructure := !isTable && !scalarTable && (row.internalType == "u" || row.internalType == "v")

			switch {
			case scalarTable:
				mapped, hasMapped := mappedTarget()
				direct, hasDirect := directFieldReference()
				if hasMapped && !scalarLeaf(mapped) {
					reject("REFERENCE_KIND_MISMATCH", path)
				}
				if hasDirect && direct.kind != "scalar" {
					reject("REFERENCE_KIND_MISMATCH", path)
				}
				if !hasMapped && !hasDirect && len(row.fieldPath) == 0 && len(row.tableName) > 0 {
					if named, ok := byName[row.tableName]; ok {
						if !scalarLeaf(named) {
							reject("REFERENCE_KIND_MISMATCH", path)
						}
						addRoot(row.tableName)
					}
				}
				reference = provParamRef{kind: "scalar-table", internalType: row.internalType}
				edgeCount++
				if edgeCount > limits.MaxEdges {
					reject("EDGE_LIMIT", path)
				}
			case !isTable && !isStructure:
				mapped, hasMapped := mappedTarget()
				direct, hasDirect := directFieldReference()
				if hasMapped {
					if !scalarLeaf(mapped) {
						reject("REFERENCE_KIND_MISMATCH", path)
					}
					edgeCount++
				}
				if hasDirect {
					if direct.kind != "scalar" {
						reject("REFERENCE_KIND_MISMATCH", path)
					}
					edgeCount++
				}
				if !hasMapped && !hasDirect && len(row.fieldPath) == 0 && len(row.tableName) > 0 {
					if named, ok := byName[row.tableName]; ok {
						if !scalarLeaf(named) {
							reject("REFERENCE_KIND_MISMATCH", path)
						}
						addRoot(row.tableName)
						edgeCount++
					}
				}
				if edgeCount > limits.MaxEdges {
					reject("EDGE_LIMIT", path)
				}
				reference = provParamRef{kind: "scalar", internalType: row.internalType}
			default:
				if len(row.tableName) == 0 {
					reject("MISSING_ASSOCIATED_TYPE", path+".TABNAME")
				}
				var targetType string
				if strings.Contains(row.fieldPath, "-") {
					key := row.tableName + "\x00" + row.fieldPath
					mapping, ok := indirect[key]
					if !ok {
						reject("MISSING_INDIRECT_TYPE", path)
					}
					consumedIndirect[key] = true
					targetType = mapping.targetType
				} else if len(row.fieldPath) > 0 {
					owner, ok := byName[row.tableName]
					if !ok {
						reject("FOREIGN_TYPE_REFERENCE", path)
					}
					resolved := resolveFieldPath(owner, row.fieldPath, byName, fieldsByNode, path)
					if resolved.kind == "scalar" {
						reject("REFERENCE_KIND_MISMATCH", path)
					}
					targetType = resolved.targetType
				} else {
					targetType = row.tableName
				}
				target, ok := byName[targetType]
				if !ok {
					reject("FOREIGN_TYPE_REFERENCE", path)
				}
				if isStructure && target.kind != "structure" {
					reject("REFERENCE_KIND_MISMATCH", path)
				}
				if row.internalType == "h" && target.kind != "table" {
					reject("REFERENCE_KIND_MISMATCH", path)
				}
				kind := "structure"
				if isTable {
					kind = "table"
				}
				reference = provParamRef{kind: kind, internalType: row.internalType, targetType: targetType}
				edgeCount++
				if edgeCount > limits.MaxEdges {
					reject("EDGE_LIMIT", path)
				}
				addRoot(targetType)
			}
		}
		result = append(result, provParameter{
			functionName: row.functionName, name: row.parameterName, parameterClass: row.parameterClass,
			position: row.position, associatedType: row.tableName, fieldPath: row.fieldPath, internalType: row.internalType,
			internalLength: row.internalLength, decimals: row.decimals, defaultValue: row.defaultValue,
			parameterText: row.parameterText, optional: row.optional, reference: reference,
		})
	}
	for index, row := range indirectRows {
		key := row.tableName + "\x00" + row.fieldPath
		if !consumedIndirect[key] {
			reject("FOREIGN_INDIRECT_TYPE", fmt.Sprintf("INDIRECTTYPES[%d]", index))
		}
	}
	return result, roots, edgeCount
}

type components struct {
	componentByNode  []int
	nodesByComponent [][]int
	cyclicComponents map[int]bool
	maximumDepth     int
}

func graphComponents(nodes []provNode, roots []string, maxDepth int) components {
	indexByName := map[string]int{}
	for i, n := range nodes {
		indexByName[n.name] = i
	}
	adjacency := make([][]int, len(nodes))
	for i, n := range nodes {
		for _, f := range n.fields {
			if f.reference.kind != "scalar" {
				adjacency[i] = append(adjacency[i], indexByName[f.reference.targetType])
			}
		}
	}
	reverse := make([][]int, len(nodes))
	for from := range adjacency {
		for _, to := range adjacency[from] {
			reverse[to] = append(reverse[to], from)
		}
	}

	visited := make([]bool, len(nodes))
	var finish []int
	type frame struct{ node, next int }
	for start := range nodes {
		if visited[start] {
			continue
		}
		visited[start] = true
		stack := []frame{{start, 0}}
		for len(stack) > 0 {
			f := &stack[len(stack)-1]
			neighbors := adjacency[f.node]
			if f.next < len(neighbors) {
				target := neighbors[f.next]
				f.next++
				if !visited[target] {
					visited[target] = true
					stack = append(stack, frame{target, 0})
				}
			} else {
				finish = append(finish, f.node)
				stack = stack[:len(stack)-1]
			}
		}
	}

	componentByNode := make([]int, len(nodes))
	for i := range componentByNode {
		componentByNode[i] = -1
	}
	var nodesByComponent [][]int
	for order := len(finish) - 1; order >= 0; order-- {
		start := finish[order]
		if componentByNode[start] != -1 {
			continue
		}
		component := len(nodesByComponent)
		var members []int
		stack := []int{start}
		componentByNode[start] = component
		for len(stack) > 0 {
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			members = append(members, node)
			for _, source := range reverse[node] {
				if componentByNode[source] == -1 {
					componentByNode[source] = component
					stack = append(stack, source)
				}
			}
		}
		sort.Ints(members)
		nodesByComponent = append(nodesByComponent, members)
	}

	cyclic := map[int]bool{}
	for component, members := range nodesByComponent {
		if len(members) > 1 {
			cyclic[component] = true
		} else {
			for _, t := range adjacency[members[0]] {
				if t == members[0] {
					cyclic[component] = true
				}
			}
		}
	}

	componentAdjacency := make([]map[int]bool, len(nodesByComponent))
	for i := range componentAdjacency {
		componentAdjacency[i] = map[int]bool{}
	}
	for from := range adjacency {
		fromC := componentByNode[from]
		for _, to := range adjacency[from] {
			toC := componentByNode[to]
			if fromC != toC {
				componentAdjacency[fromC][toC] = true
			}
		}
	}

	var rootComponents []int
	if len(roots) == 0 {
		for i := range nodesByComponent {
			rootComponents = append(rootComponents, i)
		}
	} else {
		for _, name := range roots {
			rootComponents = append(rootComponents, componentByNode[indexByName[name]])
		}
	}
	reachable := map[int]bool{}
	reachStack := append([]int(nil), rootComponents...)
	for len(reachStack) > 0 {
		c := reachStack[len(reachStack)-1]
		reachStack = reachStack[:len(reachStack)-1]
		if reachable[c] {
			continue
		}
		reachable[c] = true
		for target := range componentAdjacency[c] {
			reachStack = append(reachStack, target)
		}
	}

	indegree := make([]int, len(nodesByComponent))
	for source := range reachable {
		for target := range componentAdjacency[source] {
			if reachable[target] {
				indegree[target]++
			}
		}
	}
	var queue []int
	for c := range reachable {
		if indegree[c] == 0 {
			queue = append(queue, c)
		}
	}
	sort.Ints(queue)
	depth := make([]int, len(nodesByComponent))
	for _, root := range rootComponents {
		if depth[root] < 1 {
			depth[root] = 1
		}
	}
	maximumDepth := 0
	for cursor := 0; cursor < len(queue); cursor++ {
		source := queue[cursor]
		if depth[source] == 0 {
			depth[source] = 1
		}
		if depth[source] > maximumDepth {
			maximumDepth = depth[source]
		}
		targets := make([]int, 0, len(componentAdjacency[source]))
		for target := range componentAdjacency[source] {
			targets = append(targets, target)
		}
		sort.Ints(targets)
		for _, target := range targets {
			if !reachable[target] {
				continue
			}
			if depth[target] < depth[source]+1 {
				depth[target] = depth[source] + 1
			}
			indegree[target]--
			if indegree[target] == 0 {
				queue = append(queue, target)
			}
		}
	}
	if maximumDepth > maxDepth {
		reject("DEPTH_LIMIT", "metadata-graph")
	}
	return components{componentByNode: componentByNode, nodesByComponent: nodesByComponent, cyclicComponents: cyclic, maximumDepth: maximumDepth}
}

func finalNodes(nodes []provNode, comp components) map[string]TypeNode {
	indexByName := map[string]int{}
	for i, n := range nodes {
		indexByName[n.name] = i
	}
	out := map[string]TypeNode{}
	for nodeIndex, node := range nodes {
		fields := make([]MetadataField, 0, len(node.fields))
		for _, field := range node.fields {
			var reference Reference
			if field.reference.kind == "scalar" {
				reference = Reference{Kind: "scalar", InternalType: field.reference.internalType}
			} else {
				targetIndex := indexByName[field.reference.targetType]
				component := comp.componentByNode[nodeIndex]
				reference = Reference{
					Kind:       field.reference.kind,
					TargetType: field.reference.targetType,
					Cyclic:     component == comp.componentByNode[targetIndex] && comp.cyclicComponents[component],
				}
			}
			fields = append(fields, MetadataField{
				Name: field.name, Position: field.position, ComponentType: field.componentType,
				AssociatedType: field.associatedType, DataType: field.dataType, InternalType: field.internalType,
				Description: field.description, Decimals: field.decimals,
				NucOffset: field.nucOffset, UcOffset: field.ucOffset, NucLength: field.nucLength, UcLength: field.ucLength,
				Reference: reference,
			})
		}
		out[node.name] = TypeNode{Name: node.name, Kind: node.kind, NucLength: node.nucLength, UcLength: node.ucLength, Timestamp: node.timestamp, Fields: fields}
	}
	return out
}

func finalParameters(values []provParameter, nodes []provNode, comp components) []Parameter {
	indexByName := map[string]int{}
	for i, n := range nodes {
		indexByName[n.name] = i
	}
	out := make([]Parameter, 0, len(values))
	for _, p := range values {
		var reference ParameterReference
		switch p.reference.kind {
		case "exception":
			reference = ParameterReference{Kind: "exception"}
		case "scalar-table":
			reference = ParameterReference{Kind: "table", HasScalarLine: true, ScalarLineInternalType: p.reference.internalType}
		case "scalar":
			reference = ParameterReference{Kind: "scalar", InternalType: p.reference.internalType}
		default:
			targetIndex := indexByName[p.reference.targetType]
			component := comp.componentByNode[targetIndex]
			reference = ParameterReference{Kind: p.reference.kind, TargetType: p.reference.targetType, Cyclic: comp.cyclicComponents[component]}
		}
		out = append(out, Parameter{
			FunctionName: p.functionName, Name: p.name, ParameterClass: p.parameterClass, Position: p.position,
			AssociatedType: p.associatedType, FieldPath: p.fieldPath, InternalType: p.internalType,
			InternalLength: p.internalLength, Decimals: p.decimals, DefaultValue: p.defaultValue,
			ParameterText: p.parameterText, Optional: p.optional, Reference: reference,
		})
	}
	return out
}

func finalCycles(nodes []provNode, comp components) []Cycle {
	var result []Cycle
	for component := 0; component < len(comp.nodesByComponent); component++ {
		if !comp.cyclicComponents[component] {
			continue
		}
		typeNames := make([]string, 0, len(comp.nodesByComponent[component]))
		for _, index := range comp.nodesByComponent[component] {
			typeNames = append(typeNames, nodes[index].name)
		}
		result = append(result, Cycle{ID: fmt.Sprintf("cycle:%d", len(result)), TypeNames: typeNames})
	}
	return result
}

// Normalize normalizes the RFC_METADATA_GET closure into a bounded identity
// graph, returning a *RecursiveMetadataError on rejection.
func Normalize(input Input, opts *Options) (g Graph, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rme, ok := r.(*RecursiveMetadataError); ok {
				err = rme
				return
			}
			panic(r)
		}
	}()

	var optLimits *Limits
	var rootTypeNames []string
	if opts != nil {
		optLimits = opts.Limits
		rootTypeNames = opts.RootTypeNames
	}
	limits := resolveLimits(optLimits)
	b := &budget{limits: limits}

	// Top-level record accounting (present keys among the four).
	topKeys := []struct {
		name    string
		present bool
	}{
		{"FUNCTIONNAMES", input.FunctionNames != nil},
		{"DATATYPESCONT", true},
		{"INDIRECTTYPES", true},
		{"PARAMETERS", input.Parameters != nil},
	}
	present := 0
	for _, k := range topKeys {
		if k.present {
			present++
			b.addBytes(len(k.name), "metadata")
		}
	}
	b.addProperties(present, "metadata")

	identityWasProvided := input.FunctionNames != nil
	functionIdentity := snapshotFunctionIdentity(input.FunctionNames, b)
	typeRows := snapshotTypeRows(input.DataTypesCont, b)
	indirectRows := snapshotIndirectRows(input.IndirectTypes, b)
	parameterRows := snapshotParameterRows(input.Parameters, b)
	validateFunctionIdentity(functionIdentity, identityWasProvided, parameterRows)

	provisionalNodes := refineTableNodeKinds(buildNodes(typeRows, limits), parameterRows, indirectRows)
	validateNodeTargets(provisionalNodes)
	paramValues, paramRoots, totalEdges := provisionalParameters(parameterRows, indirectRows, provisionalNodes, limits)

	nodeNames := map[string]bool{}
	for _, n := range provisionalNodes {
		nodeNames[n.name] = true
	}
	var roots []string
	rootSet := map[string]bool{}
	for index, name := range rootTypeNames {
		if !nodeNames[name] {
			reject("FOREIGN_ROOT", fmt.Sprintf("options.rootTypeNames[%d]", index))
		}
		if !rootSet[name] {
			rootSet[name] = true
			roots = append(roots, name)
		}
	}
	for _, name := range paramRoots {
		if !rootSet[name] {
			rootSet[name] = true
			roots = append(roots, name)
		}
	}
	if len(roots) == 0 {
		for _, n := range provisionalNodes {
			roots = append(roots, n.name)
		}
	}
	comp := graphComponents(provisionalNodes, roots, limits.MaxDepth)

	reachable := map[string]bool{}
	provByName := map[string]provNode{}
	for _, n := range provisionalNodes {
		provByName[n.name] = n
	}
	stack := append([]string(nil), roots...)
	for len(stack) > 0 {
		name := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if reachable[name] {
			continue
		}
		reachable[name] = true
		for _, field := range provByName[name].fields {
			if field.reference.kind != "scalar" {
				stack = append(stack, field.reference.targetType)
			}
		}
	}
	if len(reachable) != len(provisionalNodes) {
		reject("FOREIGN_TYPE_NODE", "DATATYPESCONT")
	}

	nodes := finalNodes(provisionalNodes, comp)
	return Graph{
		Version:          1,
		FunctionIdentity: functionIdentity,
		Nodes:            nodes,
		Parameters:       finalParameters(paramValues, provisionalNodes, comp),
		RootTypeNames:    roots,
		Cycles:           finalCycles(provisionalNodes, comp),
		Limits:           limits,
		Statistics: Statistics{
			RowCount:      b.rows,
			NodeCount:     len(nodes),
			EdgeCount:     totalEdges,
			PropertyCount: b.properties,
			ByteCount:     b.bytes,
			MaximumDepth:  comp.maximumDepth,
		},
	}, nil
}
