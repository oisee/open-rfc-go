// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/metadata/rfc-metadata-get.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. The decoded metadata
// output is a map[string]any whose tables are []map[string]any; the hostile-JS-
// object hardening (accessor/proxy/prototype checks) collapses to map access;
// the ImmutableMap wrapper becomes a native map; thrown errors → returned
// wrapped sentinels. The hardcoded RFC_METADATA_GET_BOOTSTRAP interface and its
// structures are preserved verbatim — this is the interface with which the
// metadata bootstrap makes its first call before any metadata exists. See
// docs/provenance.md.

package metadata

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/structure"
)

const (
	maxMetadataRows          = 100_000
	maxRecursiveMetadataRows = 20_000
	maxStructureFields       = 9_999
	maxTimestampNamesPerKind = 512
	remoteDdicResolutionErr  = "REMOTE_DDIC_RESOLUTION_ERRORS"
)

// ErrMetadataGet reports a malformed RFC_METADATA_GET response.
var ErrMetadataGet = errors.New("metadata: invalid RFC_METADATA_GET result")

var (
	metadataNamePattern = regexp.MustCompile(`^[\x20-\x7e]{1,30}$`)
	sapLanguagePattern  = regexp.MustCompile(`^[\x20-\x7e]$`)
	metadataControl     = regexp.MustCompile(`[\x00-\x1f\x7f]`)
	exceptionPattern    = regexp.MustCompile(`^[A-Z0-9_]{1,30}$`)
	digitsOnly          = regexp.MustCompile(`^\d+$`)
)

func mgMetadataName(v, path string) (string, error) {
	if !metadataNamePattern.MatchString(v) {
		return "", fmt.Errorf("%w: %s must contain 1..30 ASCII bytes", ErrMetadataGet, path)
	}
	return v, nil
}

func mgLanguage(v string) (string, error) {
	if !sapLanguagePattern.MatchString(v) {
		return "", fmt.Errorf("%w: language must contain one printable SAP language code", ErrMetadataGet)
	}
	return v, nil
}

func bootstrapParameter(name, class string, tableName, exid string, internalLength int, optional bool) classicrfc.FunintParameter {
	if exid == "" {
		if class == "T" {
			exid = "u"
		} else {
			exid = "C"
		}
	}
	return classicrfc.FunintParameter{
		ParameterClass: class, ParameterName: name, TableName: tableName, FieldName: "",
		Exid: exid, Position: 0, Offset: 0, InternalLength: int32(internalLength), Decimals: 0,
		DefaultValue: "", ParameterText: "", Optional: optional,
	}
}

func bootstrapField(table, name string, position, offset, internalLength int, exid string, decimals int) rfctypes.RfcStructureField {
	return rfctypes.RfcStructureField{
		TableName: table, FieldName: name, Position: int32(position), Offset: int32(offset),
		InternalLength: int32(internalLength), Decimals: int32(decimals), Exid: exid,
	}
}

func mustStructure(name string, byteLength int, fields []rfctypes.RfcStructureField) rfctypes.RfcStructureDefinition {
	def, err := structure.ValidateCodec(rfctypes.RfcStructureDefinition{Name: name, ByteLength: int32(byteLength), Fields: fields}, name)
	if err != nil {
		panic(fmt.Sprintf("bootstrap structure %s invalid: %v", name, err))
	}
	return def
}

func f(table, name string, pos, off, ln int, exid string) rfctypes.RfcStructureField {
	return bootstrapField(table, name, pos, off, ln, exid, 0)
}

var bootstrapStructures = []rfctypes.RfcStructureDefinition{
	mustStructure("RFCFUNCTIONNAME", 90, []rfctypes.RfcStructureField{
		f("RFCFUNCTIONNAME", "FUNCTIONNAME", 1, 0, 60, "C"),
		f("RFCFUNCTIONNAME", "BASXML_SUPPORTED", 2, 60, 2, "C"),
		f("RFCFUNCTIONNAME", "UDAT", 3, 62, 16, "D"),
		f("RFCFUNCTIONNAME", "UTIME", 4, 78, 12, "T"),
	}),
	mustStructure("RFC_MD_DDIC_NAME", 120, []rfctypes.RfcStructureField{
		f("RFC_MD_DDIC_NAME", "TABNAME", 1, 0, 60, "C"),
		f("RFC_MD_DDIC_NAME", "FIELDNAME", 2, 60, 60, "C"),
	}),
	mustStructure("RFC_METADATA_PARAMS", 464, []rfctypes.RfcStructureField{
		f("RFC_METADATA_PARAMS", "FUNCNAME", 1, 0, 60, "C"),
		f("RFC_METADATA_PARAMS", "PARAMCLASS", 2, 60, 2, "C"),
		f("RFC_METADATA_PARAMS", "PARAMETER", 3, 62, 60, "C"),
		f("RFC_METADATA_PARAMS", "TABNAME", 4, 122, 60, "C"),
		f("RFC_METADATA_PARAMS", "FIELDNAME", 5, 182, 60, "C"),
		f("RFC_METADATA_PARAMS", "EXID", 6, 242, 2, "C"),
		f("RFC_METADATA_PARAMS", "POSITION", 7, 244, 4, "I"),
		f("RFC_METADATA_PARAMS", "OFFSET", 8, 248, 4, "I"),
		f("RFC_METADATA_PARAMS", "INTLENGTH", 9, 252, 4, "I"),
		f("RFC_METADATA_PARAMS", "DECIMALS", 10, 256, 4, "I"),
		f("RFC_METADATA_PARAMS", "DEFAULT", 11, 260, 42, "C"),
		f("RFC_METADATA_PARAMS", "PARAMTEXT", 12, 302, 158, "C"),
		f("RFC_METADATA_PARAMS", "OPTIONAL", 13, 460, 2, "C"),
	}),
	mustStructure("RFC_METADATA_DDIC", 424, []rfctypes.RfcStructureField{
		f("RFC_METADATA_DDIC", "TYPENAME", 1, 0, 60, "C"),
		f("RFC_METADATA_DDIC", "FIELDNAME", 2, 60, 60, "C"),
		f("RFC_METADATA_DDIC", "COMPTYPE", 3, 120, 2, "C"),
		f("RFC_METADATA_DDIC", "FIELDTYPE", 4, 122, 60, "C"),
		f("RFC_METADATA_DDIC", "DATATYPE", 5, 182, 8, "C"),
		f("RFC_METADATA_DDIC", "TABLENGTH", 6, 190, 12, "N"),
		f("RFC_METADATA_DDIC", "TABLENGTH_UC", 7, 202, 12, "N"),
		f("RFC_METADATA_DDIC", "DESCRIPTION", 8, 214, 120, "C"),
		f("RFC_METADATA_DDIC", "DECIMALS", 9, 334, 12, "N"),
		f("RFC_METADATA_DDIC", "INTTYPE", 10, 346, 2, "C"),
		f("RFC_METADATA_DDIC", "OFFSET", 11, 348, 12, "N"),
		f("RFC_METADATA_DDIC", "OFFSET_UC", 12, 360, 12, "N"),
		f("RFC_METADATA_DDIC", "INTLEN", 13, 372, 12, "N"),
		f("RFC_METADATA_DDIC", "INTLEN_UC", 14, 384, 12, "N"),
		f("RFC_METADATA_DDIC", "TIMESTAMP", 15, 396, 28, "C"),
	}),
	mustStructure("RFC_METADATA_DDIC_INDIRECT", 180, []rfctypes.RfcStructureField{
		f("RFC_METADATA_DDIC_INDIRECT", "TABNAME", 1, 0, 60, "C"),
		f("RFC_METADATA_DDIC_INDIRECT", "FIELDNAME", 2, 60, 60, "C"),
		f("RFC_METADATA_DDIC_INDIRECT", "FIELDTYPE", 3, 120, 60, "C"),
	}),
	mustStructure("RFC_FUNC_ERROR", 630, []rfctypes.RfcStructureField{
		f("RFC_FUNC_ERROR", "FUNCNAME", 1, 0, 60, "C"),
		f("RFC_FUNC_ERROR", "EXCEPTION", 2, 60, 60, "C"),
		f("RFC_FUNC_ERROR", "EXCEPTION_TEXT", 3, 120, 510, "C"),
	}),
	mustStructure("RFC_DD_ERROR", 690, []rfctypes.RfcStructureField{
		f("RFC_DD_ERROR", "TABNAME", 1, 0, 60, "C"),
		f("RFC_DD_ERROR", "FIELDNAME", 2, 60, 60, "C"),
		f("RFC_DD_ERROR", "EXCEPTION", 3, 120, 60, "C"),
		f("RFC_DD_ERROR", "EXCEPTION_TEXT", 4, 180, 510, "C"),
	}),
}

var bootstrapMetadata = RfcFunctionInterface{
	Name: "RFC_METADATA_GET", RemoteBasxmlSupported: false, RemoteCall: "R", UpdateTask: false,
	Parameters: []classicrfc.FunintParameter{
		bootstrapParameter("DEEP", "I", "", "", 1, true),
		bootstrapParameter("LANGUAGE", "I", "", "", 1, true),
		bootstrapParameter("GET_CLIENT_DEP_FIELDS", "I", "", "", 1, true),
		bootstrapParameter("GET_TIMESTAMPS", "I", "", "", 1, true),
		bootstrapParameter("FUNCTIONNAMES", "T", "RFCFUNCTIONNAME", "", 0, false),
		bootstrapParameter("DATATYPES", "T", "RFC_MD_DDIC_NAME", "", 0, false),
		bootstrapParameter("KNOWN_DATATYPES", "T", "RFC_MD_DDIC_NAME", "", 0, false),
		bootstrapParameter("PARAMETERS", "T", "RFC_METADATA_PARAMS", "", 0, false),
		bootstrapParameter("DATATYPESCONT", "T", "RFC_METADATA_DDIC", "", 0, false),
		bootstrapParameter("INDIRECTTYPES", "T", "RFC_METADATA_DDIC_INDIRECT", "", 0, false),
		bootstrapParameter("FUNC_ERRORS", "T", "RFC_FUNC_ERROR", "", 0, true),
		bootstrapParameter("DD_ERRORS", "T", "RFC_DD_ERROR", "", 0, true),
	},
	Exceptions: []string{"INVALID_MODE", "INTERNAL_ERROR"},
}

// RfcMetadataGetBootstrap is the hardcoded interface plus its structures.
type RfcMetadataGetBootstrap struct {
	Metadata   RfcFunctionInterface
	Structures map[string]rfctypes.RfcStructureDefinition
}

func structureMap(defs []rfctypes.RfcStructureDefinition) map[string]rfctypes.RfcStructureDefinition {
	m := make(map[string]rfctypes.RfcStructureDefinition, len(defs))
	for _, d := range defs {
		m[d.Name] = d
	}
	return m
}

// RfcMetadataGetBootstrapValue is the bootstrap for RFC_METADATA_GET itself.
var RfcMetadataGetBootstrapValue = RfcMetadataGetBootstrap{
	Metadata:   bootstrapMetadata,
	Structures: structureMap(bootstrapStructures),
}

func filterErrorStructures() []rfctypes.RfcStructureDefinition {
	var out []rfctypes.RfcStructureDefinition
	for _, d := range bootstrapStructures {
		if d.Name == "RFC_FUNC_ERROR" || d.Name == "RFC_DD_ERROR" {
			out = append(out, d)
		}
	}
	return out
}

var timestampBootstrapStructures = append([]rfctypes.RfcStructureDefinition{
	mustStructure("RFC_METADATA_FUNC_TIMESTAMP", 88, []rfctypes.RfcStructureField{
		f("RFC_METADATA_FUNC_TIMESTAMP", "FUNCNAME", 1, 0, 60, "C"),
		f("RFC_METADATA_FUNC_TIMESTAMP", "UDAT", 2, 60, 16, "D"),
		f("RFC_METADATA_FUNC_TIMESTAMP", "UTIME", 3, 76, 12, "T"),
	}),
	mustStructure("RFC_METADATA_DDIC_TIMESTAMP", 88, []rfctypes.RfcStructureField{
		f("RFC_METADATA_DDIC_TIMESTAMP", "TYPENAME", 1, 0, 60, "C"),
		f("RFC_METADATA_DDIC_TIMESTAMP", "TIMESTAMP", 2, 60, 28, "C"),
	}),
}, filterErrorStructures()...)

var timestampBootstrapMetadata = RfcFunctionInterface{
	Name: "RFC_METADATA_GET_TIMESTAMP", RemoteBasxmlSupported: false, RemoteCall: "R", UpdateTask: false,
	Parameters: []classicrfc.FunintParameter{
		bootstrapParameter("FUNCTION_TIMESTAMPS", "T", "RFC_METADATA_FUNC_TIMESTAMP", "", 0, false),
		bootstrapParameter("DDIC_TIMESTAMPS", "T", "RFC_METADATA_DDIC_TIMESTAMP", "", 0, false),
		bootstrapParameter("FUNC_ERRORS", "T", "RFC_FUNC_ERROR", "", 0, true),
		bootstrapParameter("DD_ERRORS", "T", "RFC_DD_ERROR", "", 0, true),
	},
	Exceptions: []string{},
}

// RfcMetadataGetTimestampBootstrapValue is the bootstrap for the timestamp lookup.
var RfcMetadataGetTimestampBootstrapValue = RfcMetadataGetBootstrap{
	Metadata:   timestampBootstrapMetadata,
	Structures: structureMap(timestampBootstrapStructures),
}

// RfcMetadataGetInvocation carries a bounded, validated request input.
type RfcMetadataGetInvocation struct {
	Input map[string]any
}

// RfcMetadataGetTimestampInvocation additionally captures request identities.
type RfcMetadataGetTimestampInvocation struct {
	Input          map[string]any
	FunctionNames  []string
	StructureNames []string
}

func baseInput(language string) map[string]any {
	return map[string]any{
		"DEEP": "X", "LANGUAGE": language, "GET_TIMESTAMPS": "X",
		"FUNCTIONNAMES": []map[string]any{}, "DATATYPES": []map[string]any{},
		"KNOWN_DATATYPES": []map[string]any{}, "PARAMETERS": []map[string]any{},
		"DATATYPESCONT": []map[string]any{}, "INDIRECTTYPES": []map[string]any{},
		"FUNC_ERRORS": []map[string]any{}, "DD_ERRORS": []map[string]any{},
	}
}

// CreateFunctionInvocation builds the input for a function metadata lookup.
func CreateFunctionInvocation(functionName, language string) (RfcMetadataGetInvocation, error) {
	if language == "" {
		language = "E"
	}
	name, err := mgMetadataName(functionName, "functionName")
	if err != nil {
		return RfcMetadataGetInvocation{}, err
	}
	lang, err := mgLanguage(language)
	if err != nil {
		return RfcMetadataGetInvocation{}, err
	}
	input := baseInput(lang)
	input["FUNCTIONNAMES"] = []map[string]any{{"FUNCTIONNAME": name}}
	return RfcMetadataGetInvocation{Input: input}, nil
}

// CreateStructureInvocation builds the input for a structure metadata lookup.
func CreateStructureInvocation(structureName, language string) (RfcMetadataGetInvocation, error) {
	if language == "" {
		language = "E"
	}
	name, err := mgMetadataName(structureName, "structureName")
	if err != nil {
		return RfcMetadataGetInvocation{}, err
	}
	lang, err := mgLanguage(language)
	if err != nil {
		return RfcMetadataGetInvocation{}, err
	}
	input := baseInput(lang)
	input["DATATYPES"] = []map[string]any{{"TABNAME": name}}
	return RfcMetadataGetInvocation{Input: input}, nil
}

func requestedMetadataNames(value []string, kind string) ([]string, error) {
	if len(value) > maxTimestampNamesPerKind {
		return nil, fmt.Errorf("%w: RFC_METADATA_GET_TIMESTAMP accepts at most %d %s names", ErrMetadataGet, maxTimestampNamesPerKind, kind)
	}
	var names []string
	seen := map[string]bool{}
	for index, raw := range value {
		name, err := mgMetadataName(raw, fmt.Sprintf("%s names[%d]", kind, index))
		if err != nil {
			return nil, err
		}
		if seen[name] {
			return nil, fmt.Errorf("%w: duplicate %s name %s", ErrMetadataGet, kind, name)
		}
		seen[name] = true
		names = append(names, name)
	}
	return names, nil
}

// CreateTimestampInvocation snapshots one bounded timestamp batch.
func CreateTimestampInvocation(functionNames, structureNames []string) (RfcMetadataGetTimestampInvocation, error) {
	functions, err := requestedMetadataNames(functionNames, "function")
	if err != nil {
		return RfcMetadataGetTimestampInvocation{}, err
	}
	structures, err := requestedMetadataNames(structureNames, "structure")
	if err != nil {
		return RfcMetadataGetTimestampInvocation{}, err
	}
	funcRows := make([]map[string]any, len(functions))
	for i, n := range functions {
		funcRows[i] = map[string]any{"FUNCNAME": n}
	}
	ddicRows := make([]map[string]any, len(structures))
	for i, n := range structures {
		ddicRows[i] = map[string]any{"TYPENAME": n}
	}
	return RfcMetadataGetTimestampInvocation{
		Input: map[string]any{
			"FUNCTION_TIMESTAMPS": funcRows, "DDIC_TIMESTAMPS": ddicRows,
			"FUNC_ERRORS": []map[string]any{}, "DD_ERRORS": []map[string]any{},
		},
		FunctionNames: functions, StructureNames: structures,
	}, nil
}

// ---- output decoding helpers ----

func mgRows(output map[string]any, name string, maximum int) ([]map[string]any, error) {
	v, ok := output[name]
	if !ok {
		return nil, fmt.Errorf("%w: output %s must be present", ErrMetadataGet, name)
	}
	source, ok := v.([]map[string]any)
	if !ok || len(source) > maximum {
		return nil, fmt.Errorf("%w: output %s must contain at most %d rows", ErrMetadataGet, name, maximum)
	}
	return source, nil
}

func mgText(row map[string]any, name, path string, maximum int) (string, error) {
	v, ok := row[name]
	s, isStr := v.(string)
	if !ok || !isStr || len([]rune(s)) > maximum || metadataControl.MatchString(s) {
		return "", fmt.Errorf("%w: %s.%s contains invalid text", ErrMetadataGet, path, name)
	}
	return s, nil
}

func mgInt(row map[string]any, name, path string) (int, error) {
	v, ok := row[name]
	if !ok {
		return 0, fmt.Errorf("%w: %s.%s must be a non-negative safe integer", ErrMetadataGet, path, name)
	}
	switch n := v.(type) {
	case int:
		if n >= 0 {
			return n, nil
		}
	case int32:
		if n >= 0 {
			return int(n), nil
		}
	case int64:
		if n >= 0 {
			return int(n), nil
		}
	case string:
		if digitsOnly.MatchString(n) {
			p, err := strconv.Atoi(n)
			if err == nil {
				return p, nil
			}
		}
	}
	return 0, fmt.Errorf("%w: %s.%s must be a non-negative safe integer", ErrMetadataGet, path, name)
}

func mgFlag(value, path string) (bool, error) {
	if value != "" && value != "X" {
		return false, fmt.Errorf("%w: %s must be initial or X", ErrMetadataGet, path)
	}
	return value == "X", nil
}

func mgFixedDigits(row map[string]any, name, path string, length int) (string, error) {
	v, err := mgText(row, name, path, length)
	if err != nil {
		return "", err
	}
	if len(v) != length || !digitsOnly.MatchString(v) {
		return "", fmt.Errorf("%w: %s.%s must contain exactly %d digits", ErrMetadataGet, path, name, length)
	}
	return v, nil
}

func anyToString(v any) string {
	switch n := v.(type) {
	case string:
		return n
	case int:
		return strconv.Itoa(n)
	case int32:
		return strconv.Itoa(int(n))
	case int64:
		return strconv.FormatInt(n, 10)
	default:
		return ""
	}
}

// ---- result types ----

// RfcMetadataGetFunctionResult pairs a descriptor with its generation token.
type RfcMetadataGetFunctionResult struct {
	Value           RfcFunctionInterface
	GenerationToken string
}

// RfcMetadataGetStructureResult pairs a DDIC descriptor with its token.
type RfcMetadataGetStructureResult struct {
	Value           rfctypes.RfcStructureDefinition
	GenerationToken string
}

// RfcMetadataGetRecursiveFunctionResult pairs a type closure with its token.
type RfcMetadataGetRecursiveFunctionResult struct {
	Value           Graph
	GenerationToken string
}

// RfcFunctionMetadataTimestamp is one function generation observed in a batch.
type RfcFunctionMetadataTimestamp struct {
	FunctionName, Date, Time, Token string
}

// RfcStructureMetadataTimestamp is one structure generation observed in a batch.
type RfcStructureMetadataTimestamp struct {
	StructureName, Timestamp, Token string
}

// RfcMetadataTimestampBatch is a normalized timestamp batch.
type RfcMetadataTimestampBatch struct {
	Functions       map[string]RfcFunctionMetadataTimestamp
	Structures      map[string]RfcStructureMetadataTimestamp
	FunctionErrors  map[string]string
	StructureErrors map[string]string
}

func matchingError(output map[string]any, tableName, keyName, objectName string) (string, error) {
	errRows, err := mgRows(output, tableName, maxMetadataRows)
	if err != nil {
		return "", err
	}
	for index, row := range errRows {
		path := fmt.Sprintf("RFC_METADATA_GET output %s[%d]", tableName, index)
		key, err := mgText(row, keyName, path, 30)
		if err != nil {
			return "", err
		}
		if key != objectName {
			continue
		}
		exception, err := mgText(row, "EXCEPTION", path, 30)
		if err != nil {
			return "", err
		}
		if !exceptionPattern.MatchString(exception) {
			return "", fmt.Errorf("%w: %s.EXCEPTION is invalid", ErrMetadataGet, path)
		}
		return exception, nil
	}
	return "", nil
}

func normalizedFunctionInternalLength(exid string, v int, path string) (int, error) {
	switch exid {
	case "C", "N", "D", "T":
		if v&1 != 0 {
			return 0, fmt.Errorf("%w: %s.INTLENGTH has an odd Unicode byte width", ErrMetadataGet, path)
		}
		return v / 2, nil
	default:
		return v, nil
	}
}

// NormalizeFunctionResult decodes a function metadata response into a flat
// descriptor and its generation token.
func NormalizeFunctionResult(functionName string, output map[string]any) (RfcMetadataGetFunctionResult, error) {
	var zero RfcMetadataGetFunctionResult
	name, err := mgMetadataName(functionName, "functionName")
	if err != nil {
		return zero, err
	}
	failure, err := matchingError(output, "FUNC_ERRORS", "FUNCNAME", name)
	if err != nil {
		return zero, err
	}
	if failure != "" {
		return zero, fmt.Errorf("%w: could not resolve function %s (%s)", ErrMetadataGet, name, failure)
	}
	functionRows, err := mgRows(output, "FUNCTIONNAMES", maxMetadataRows)
	if err != nil {
		return zero, err
	}
	var identity map[string]any
	matches := 0
	for index, row := range functionRows {
		fn, err := mgText(row, "FUNCTIONNAME", fmt.Sprintf("RFC_METADATA_GET output FUNCTIONNAMES[%d]", index), 30)
		if err != nil {
			return zero, err
		}
		if fn == name {
			matches++
			identity = row
		}
	}
	if matches != 1 {
		return zero, fmt.Errorf("%w: returned %d identities for function %s", ErrMetadataGet, matches, name)
	}
	basxmlText, err := mgText(identity, "BASXML_SUPPORTED", "RFC_METADATA_GET function identity", 1)
	if err != nil {
		return zero, err
	}
	basxml, err := mgFlag(basxmlText, "RFC_METADATA_GET BASXML_SUPPORTED")
	if err != nil {
		return zero, err
	}
	parameterRows, err := mgRows(output, "PARAMETERS", maxMetadataRows)
	if err != nil {
		return zero, err
	}
	var parameters []classicrfc.FunintParameter
	var exceptions []string
	names := map[string]bool{}
	for index, row := range parameterRows {
		path := fmt.Sprintf("RFC_METADATA_GET output PARAMETERS[%d]", index)
		funcName, err := mgText(row, "FUNCNAME", path, 30)
		if err != nil {
			return zero, err
		}
		if funcName != name {
			continue
		}
		parameterClass, err := mgText(row, "PARAMCLASS", path, 1)
		if err != nil {
			return zero, err
		}
		if !regexp.MustCompile(`^[IECXT]$`).MatchString(parameterClass) {
			return zero, fmt.Errorf("%w: %s.PARAMCLASS is unsupported", ErrMetadataGet, path)
		}
		paramNameRaw, err := mgText(row, "PARAMETER", path, 30)
		if err != nil {
			return zero, err
		}
		parameterName, err := mgMetadataName(paramNameRaw, path+".PARAMETER")
		if err != nil {
			return zero, err
		}
		if names[parameterName] {
			return zero, fmt.Errorf("%w: returned duplicate parameter %s", ErrMetadataGet, parameterName)
		}
		position, err := mgInt(row, "POSITION", path)
		if err != nil {
			return zero, err
		}
		names[parameterName] = true
		if parameterClass == "X" {
			exceptions = append(exceptions, parameterName)
			continue
		}
		exid, err := mgText(row, "EXID", path, 1)
		if err != nil {
			return zero, err
		}
		tabName, err := mgText(row, "TABNAME", path, 30)
		if err != nil {
			return zero, err
		}
		fieldName, err := mgText(row, "FIELDNAME", path, 30)
		if err != nil {
			return zero, err
		}
		offset, err := mgInt(row, "OFFSET", path)
		if err != nil {
			return zero, err
		}
		intLenRaw, err := mgInt(row, "INTLENGTH", path)
		if err != nil {
			return zero, err
		}
		internalLength, err := normalizedFunctionInternalLength(exid, intLenRaw, path)
		if err != nil {
			return zero, err
		}
		decimals, err := mgInt(row, "DECIMALS", path)
		if err != nil {
			return zero, err
		}
		defaultValue, err := mgText(row, "DEFAULT", path, 21)
		if err != nil {
			return zero, err
		}
		parameterText, err := mgText(row, "PARAMTEXT", path, 79)
		if err != nil {
			return zero, err
		}
		optionalText, err := mgText(row, "OPTIONAL", path, 1)
		if err != nil {
			return zero, err
		}
		optional, err := mgFlag(optionalText, path+".OPTIONAL")
		if err != nil {
			return zero, err
		}
		parameters = append(parameters, classicrfc.FunintParameter{
			ParameterClass: parameterClass, ParameterName: parameterName, TableName: tabName, FieldName: fieldName,
			Exid: exid, Position: int32(position), Offset: int32(offset), InternalLength: int32(internalLength),
			Decimals: int32(decimals), DefaultValue: defaultValue, ParameterText: parameterText, Optional: optional,
		})
	}
	date, err := mgFixedDigits(identity, "UDAT", "RFC_METADATA_GET function identity", 8)
	if err != nil {
		return zero, err
	}
	tm, err := mgFixedDigits(identity, "UTIME", "RFC_METADATA_GET function identity", 6)
	if err != nil {
		return zero, err
	}
	return RfcMetadataGetFunctionResult{
		Value: RfcFunctionInterface{
			Name: name, RemoteBasxmlSupported: basxml, RemoteCall: "R", UpdateTask: false,
			Parameters: parameters, Exceptions: exceptions, ResumableExceptionRowCount: 0,
		},
		GenerationToken: "function:" + date + ":" + tm,
	}, nil
}

// NormalizeFunction returns just the descriptor.
func NormalizeFunction(functionName string, output map[string]any) (RfcFunctionInterface, error) {
	r, err := NormalizeFunctionResult(functionName, output)
	return r.Value, err
}

func assertRecursiveMetadataRowBudget(output map[string]any) error {
	total := 0
	for _, name := range []string{"FUNCTIONNAMES", "DATATYPESCONT", "INDIRECTTYPES", "PARAMETERS"} {
		v, ok := output[name]
		src, isArr := v.([]map[string]any)
		if !ok || !isArr {
			return fmt.Errorf("%w: output %s must be an array", ErrMetadataGet, name)
		}
		total += len(src)
		if total > maxRecursiveMetadataRows {
			return fmt.Errorf("%w: recursive metadata must contain at most %d total rows", ErrMetadataGet, maxRecursiveMetadataRows)
		}
	}
	return nil
}

func toRecursiveInput(output map[string]any) (Input, error) {
	get := func(name string) ([]map[string]any, error) {
		v, ok := output[name]
		src, isArr := v.([]map[string]any)
		if !ok || !isArr {
			return nil, fmt.Errorf("%w: output %s must be an array", ErrMetadataGet, name)
		}
		return src, nil
	}
	fnRows, err := get("FUNCTIONNAMES")
	if err != nil {
		return Input{}, err
	}
	typeRows, err := get("DATATYPESCONT")
	if err != nil {
		return Input{}, err
	}
	indirectRows, err := get("INDIRECTTYPES")
	if err != nil {
		return Input{}, err
	}
	paramRows, err := get("PARAMETERS")
	if err != nil {
		return Input{}, err
	}
	fns := make([]FunctionRow, len(fnRows))
	for i, r := range fnRows {
		fns[i] = FunctionRow{FunctionName: anyToString(r["FUNCTIONNAME"]), BasxmlSupported: anyToString(r["BASXML_SUPPORTED"]), UDat: anyToString(r["UDAT"]), UTime: anyToString(r["UTIME"])}
	}
	types := make([]TypeRowInput, len(typeRows))
	for i, r := range typeRows {
		types[i] = TypeRowInput{
			TypeName: anyToString(r["TYPENAME"]), FieldName: anyToString(r["FIELDNAME"]), CompType: anyToString(r["COMPTYPE"]),
			FieldType: anyToString(r["FIELDTYPE"]), DataType: anyToString(r["DATATYPE"]), TabLength: anyToString(r["TABLENGTH"]),
			TabLengthUC: anyToString(r["TABLENGTH_UC"]), Description: anyToString(r["DESCRIPTION"]), Decimals: anyToString(r["DECIMALS"]),
			IntType: anyToString(r["INTTYPE"]), Offset: anyToString(r["OFFSET"]), OffsetUC: anyToString(r["OFFSET_UC"]),
			IntLen: anyToString(r["INTLEN"]), IntLenUC: anyToString(r["INTLEN_UC"]), Timestamp: anyToString(r["TIMESTAMP"]),
		}
	}
	indirects := make([]IndirectRowInput, len(indirectRows))
	for i, r := range indirectRows {
		indirects[i] = IndirectRowInput{TabName: anyToString(r["TABNAME"]), FieldName: anyToString(r["FIELDNAME"]), FieldType: anyToString(r["FIELDTYPE"])}
	}
	params := make([]ParameterRowInput, len(paramRows))
	for i, r := range paramRows {
		params[i] = ParameterRowInput{
			FuncName: anyToString(r["FUNCNAME"]), ParamClass: anyToString(r["PARAMCLASS"]), Parameter: anyToString(r["PARAMETER"]),
			TabName: anyToString(r["TABNAME"]), FieldName: anyToString(r["FIELDNAME"]), Exid: anyToString(r["EXID"]),
			Position: anyToString(r["POSITION"]), Offset: anyToString(r["OFFSET"]), IntLength: anyToString(r["INTLENGTH"]),
			Decimals: anyToString(r["DECIMALS"]), Default: anyToString(r["DEFAULT"]), ParamText: anyToString(r["PARAMTEXT"]), Optional: anyToString(r["OPTIONAL"]),
		}
	}
	return Input{FunctionNames: &fns, DataTypesCont: types, IndirectTypes: indirects, Parameters: &params}, nil
}

func isCompleteUtclongScalarFallback(output map[string]any, descriptor RfcFunctionInterface, ddicErrors []map[string]any) bool {
	if len(ddicErrors) != 1 {
		return false
	}
	errorPath := "RFC_METADATA_GET output DD_ERRORS[0]"
	tab, _ := mgText(ddicErrors[0], "TABNAME", errorPath, 30)
	fld, _ := mgText(ddicErrors[0], "FIELDNAME", errorPath, 30)
	exc, _ := mgText(ddicErrors[0], "EXCEPTION", errorPath, 30)
	if tab != "UTCLONG" || fld != "" || exc != "NOT_FOUND" {
		return false
	}
	matches := 0
	for _, p := range descriptor.Parameters {
		if p.TableName != "UTCLONG" {
			continue
		}
		matches++
		if p.ParameterClass != "C" || p.FieldName != "" || p.Exid != "p" || p.InternalLength != 8 || p.Decimals != 0 || p.Optional {
			return false
		}
	}
	if matches == 0 {
		return false
	}
	rawMatches := 0
	parameterRows, err := mgRows(output, "PARAMETERS", maxMetadataRows)
	if err != nil {
		return false
	}
	for index, p := range parameterRows {
		path := fmt.Sprintf("RFC_METADATA_GET output PARAMETERS[%d]", index)
		tn, _ := mgText(p, "TABNAME", path, 30)
		if tn != "UTCLONG" {
			continue
		}
		rawMatches++
		fn, _ := mgText(p, "FUNCNAME", path, 30)
		pc, _ := mgText(p, "PARAMCLASS", path, 1)
		fld, _ := mgText(p, "FIELDNAME", path, 30)
		ex, _ := mgText(p, "EXID", path, 1)
		il, _ := mgInt(p, "INTLENGTH", path)
		dc, _ := mgInt(p, "DECIMALS", path)
		opt, _ := mgText(p, "OPTIONAL", path, 1)
		if fn != descriptor.Name || pc != "C" || fld != "" || ex != "p" || il != 8 || dc != 0 || opt != "" {
			return false
		}
	}
	if rawMatches == 0 {
		return false
	}
	typeRows, _ := mgRows(output, "DATATYPESCONT", maxMetadataRows)
	for index, r := range typeRows {
		path := fmt.Sprintf("RFC_METADATA_GET output DATATYPESCONT[%d]", index)
		for _, n := range []string{"TYPENAME", "FIELDTYPE", "DATATYPE"} {
			if s, _ := mgText(r, n, path, 30); s == "UTCLONG" {
				return false
			}
		}
	}
	indirectRows, _ := mgRows(output, "INDIRECTTYPES", maxMetadataRows)
	for index, r := range indirectRows {
		path := fmt.Sprintf("RFC_METADATA_GET output INDIRECTTYPES[%d]", index)
		tn, _ := mgText(r, "TABNAME", path, 30)
		ft, _ := mgText(r, "FIELDTYPE", path, 30)
		if tn == "UTCLONG" || ft == "UTCLONG" {
			return false
		}
	}
	return true
}

// NormalizeRecursiveFunctionResult decodes a DEEP function response into the
// full bounded type graph and its generation token.
func NormalizeRecursiveFunctionResult(functionName string, output map[string]any) (RfcMetadataGetRecursiveFunctionResult, error) {
	var zero RfcMetadataGetRecursiveFunctionResult
	name, err := mgMetadataName(functionName, "functionName")
	if err != nil {
		return zero, err
	}
	if err := assertRecursiveMetadataRowBudget(output); err != nil {
		return zero, err
	}
	flat, err := NormalizeFunctionResult(name, output)
	if err != nil {
		return zero, err
	}
	functionErrors, err := mgRows(output, "FUNC_ERRORS", maxMetadataRows)
	if err != nil {
		return zero, err
	}
	if len(functionErrors) != 0 {
		return zero, fmt.Errorf("%w: recursive metadata returned a foreign function error", ErrMetadataGet)
	}
	ddicErrors, err := mgRows(output, "DD_ERRORS", maxMetadataRows)
	if err != nil {
		return zero, err
	}
	if len(ddicErrors) != 0 && !isCompleteUtclongScalarFallback(output, flat.Value, ddicErrors) {
		return zero, &RecursiveMetadataError{Code: remoteDdicResolutionErr, Path: fmt.Sprintf("DD_ERRORS:%d", len(ddicErrors))}
	}
	recursiveInput, err := toRecursiveInput(output)
	if err != nil {
		return zero, err
	}
	graph, err := Normalize(recursiveInput, nil)
	if err != nil {
		return zero, err
	}
	if graph.FunctionIdentity == nil || graph.FunctionIdentity.Name != name {
		return zero, fmt.Errorf("%w: recursive metadata identity does not match function %s", ErrMetadataGet, name)
	}
	if graph.FunctionIdentity.GenerationToken != flat.GenerationToken {
		return zero, fmt.Errorf("%w: recursive metadata generation does not match function %s", ErrMetadataGet, name)
	}
	return RfcMetadataGetRecursiveFunctionResult{Value: graph, GenerationToken: flat.GenerationToken}, nil
}

// NormalizeStructureResult decodes a structure metadata response.
func NormalizeStructureResult(structureName string, output map[string]any) (RfcMetadataGetStructureResult, error) {
	var zero RfcMetadataGetStructureResult
	name, err := mgMetadataName(structureName, "structureName")
	if err != nil {
		return zero, err
	}
	failure, err := matchingError(output, "DD_ERRORS", "TABNAME", name)
	if err != nil {
		return zero, err
	}
	if failure != "" {
		return zero, fmt.Errorf("%w: could not resolve structure %s (%s)", ErrMetadataGet, name, failure)
	}
	typeRows, err := mgRows(output, "DATATYPESCONT", maxMetadataRows)
	if err != nil {
		return zero, err
	}
	var matches []map[string]any
	for index, row := range typeRows {
		tn, err := mgText(row, "TYPENAME", fmt.Sprintf("RFC_METADATA_GET output DATATYPESCONT[%d]", index), 30)
		if err != nil {
			return zero, err
		}
		if tn == name {
			matches = append(matches, row)
		}
	}
	if len(matches) == 0 {
		return zero, fmt.Errorf("%w: returned no type rows for structure %s", ErrMetadataGet, name)
	}
	if len(matches) > maxStructureFields {
		return zero, fmt.Errorf("%w: structure %s exceeds %d fields", ErrMetadataGet, name, maxStructureFields)
	}
	var fields []rfctypes.RfcStructureField
	fieldNames := map[string]bool{}
	byteLength := -1
	generationTimestamp := ""
	previousEnd := 0
	for index, row := range matches {
		path := fmt.Sprintf("RFC_METADATA_GET structure %s[%d]", name, index)
		candidateByteLength, err := mgInt(row, "TABLENGTH_UC", path)
		if err != nil {
			return zero, err
		}
		if byteLength == -1 {
			byteLength = candidateByteLength
		} else if candidateByteLength != byteLength {
			return zero, fmt.Errorf("%w: structure %s has inconsistent lengths", ErrMetadataGet, name)
		}
		candidateTimestamp, err := mgFixedDigits(row, "TIMESTAMP", path, 14)
		if err != nil {
			return zero, err
		}
		if generationTimestamp == "" {
			generationTimestamp = candidateTimestamp
		} else if candidateTimestamp != generationTimestamp {
			return zero, fmt.Errorf("%w: structure %s has inconsistent timestamps", ErrMetadataGet, name)
		}
		fnRaw, err := mgText(row, "FIELDNAME", path, 30)
		if err != nil {
			return zero, err
		}
		fieldName, err := mgMetadataName(fnRaw, path+".FIELDNAME")
		if err != nil {
			return zero, err
		}
		if fieldNames[fieldName] {
			return zero, fmt.Errorf("%w: structure %s has duplicate field %s", ErrMetadataGet, name, fieldName)
		}
		componentType, err := mgText(row, "COMPTYPE", path, 1)
		if err != nil {
			return zero, err
		}
		exid, err := mgText(row, "INTTYPE", path, 1)
		if err != nil {
			return zero, err
		}
		if (componentType != "" && componentType != "E") || exid == "u" || exid == "h" || exid == "v" {
			return zero, fmt.Errorf("%w: structure %s.%s requires a negotiated recursive serializer", ErrMetadataGet, name, fieldName)
		}
		offset, err := mgInt(row, "OFFSET_UC", path)
		if err != nil {
			return zero, err
		}
		internalLength, err := mgInt(row, "INTLEN_UC", path)
		if err != nil {
			return zero, err
		}
		end := offset + internalLength
		if offset < previousEnd || end > candidateByteLength {
			return zero, fmt.Errorf("%w: structure %s.%s has invalid geometry", ErrMetadataGet, name, fieldName)
		}
		decimals, err := mgInt(row, "DECIMALS", path)
		if err != nil {
			return zero, err
		}
		fieldNames[fieldName] = true
		previousEnd = end
		fields = append(fields, bootstrapField(name, fieldName, index+1, offset, internalLength, exid, decimals))
	}
	descriptor, err := structure.ValidateCodec(rfctypes.RfcStructureDefinition{Name: name, ByteLength: int32(byteLength), Fields: fields}, name)
	if err != nil {
		return zero, err
	}
	return RfcMetadataGetStructureResult{Value: descriptor, GenerationToken: "structure:" + generationTimestamp}, nil
}

// NormalizeStructure returns just the descriptor.
func NormalizeStructure(structureName string, output map[string]any) (rfctypes.RfcStructureDefinition, error) {
	r, err := NormalizeStructureResult(structureName, output)
	return r.Value, err
}

func timestampErrorRows(output map[string]any, tableName, keyName string, requested map[string]bool, outcomes map[string]bool, kind string) (map[string]string, error) {
	errors_ := map[string]string{}
	errRows, err := mgRows(output, tableName, len(requested))
	if err != nil {
		return nil, err
	}
	for index, row := range errRows {
		path := fmt.Sprintf("RFC_METADATA_GET_TIMESTAMP output %s[%d]", tableName, index)
		keyRaw, err := mgText(row, keyName, path, 30)
		if err != nil {
			return nil, err
		}
		objectName, err := mgMetadataName(keyRaw, path+"."+keyName)
		if err != nil {
			return nil, err
		}
		if !requested[objectName] {
			return nil, fmt.Errorf("%w: TIMESTAMP returned unrequested %s %s", ErrMetadataGet, kind, objectName)
		}
		if outcomes[objectName] {
			return nil, fmt.Errorf("%w: TIMESTAMP returned duplicate outcome for %s %s", ErrMetadataGet, kind, objectName)
		}
		exception, err := mgText(row, "EXCEPTION", path, 30)
		if err != nil {
			return nil, err
		}
		if !exceptionPattern.MatchString(exception) {
			return nil, fmt.Errorf("%w: %s.EXCEPTION is invalid", ErrMetadataGet, path)
		}
		outcomes[objectName] = true
		errors_[objectName] = exception
	}
	return errors_, nil
}

// NormalizeTimestamps normalizes a complete timestamp batch.
func NormalizeTimestamps(functionNames, structureNames []string, output map[string]any) (RfcMetadataTimestampBatch, error) {
	var zero RfcMetadataTimestampBatch
	requestedFunctions, err := requestedMetadataNames(functionNames, "function")
	if err != nil {
		return zero, err
	}
	requestedStructures, err := requestedMetadataNames(structureNames, "structure")
	if err != nil {
		return zero, err
	}
	functionSet := setOf(requestedFunctions)
	structureSet := setOf(requestedStructures)
	functionOutcomes := map[string]bool{}
	structureOutcomes := map[string]bool{}
	functions := map[string]RfcFunctionMetadataTimestamp{}
	structures := map[string]RfcStructureMetadataTimestamp{}

	funcRows, err := mgRows(output, "FUNCTION_TIMESTAMPS", len(requestedFunctions))
	if err != nil {
		return zero, err
	}
	for index, row := range funcRows {
		path := fmt.Sprintf("RFC_METADATA_GET_TIMESTAMP output FUNCTION_TIMESTAMPS[%d]", index)
		fnRaw, err := mgText(row, "FUNCNAME", path, 30)
		if err != nil {
			return zero, err
		}
		functionName, err := mgMetadataName(fnRaw, path+".FUNCNAME")
		if err != nil {
			return zero, err
		}
		if !functionSet[functionName] {
			return zero, fmt.Errorf("%w: TIMESTAMP returned unrequested function %s", ErrMetadataGet, functionName)
		}
		if functionOutcomes[functionName] {
			return zero, fmt.Errorf("%w: TIMESTAMP returned duplicate outcome for function %s", ErrMetadataGet, functionName)
		}
		date, err := mgFixedDigits(row, "UDAT", path, 8)
		if err != nil {
			return zero, err
		}
		tm, err := mgFixedDigits(row, "UTIME", path, 6)
		if err != nil {
			return zero, err
		}
		functionOutcomes[functionName] = true
		functions[functionName] = RfcFunctionMetadataTimestamp{FunctionName: functionName, Date: date, Time: tm, Token: "function:" + date + ":" + tm}
	}

	structureRows, err := mgRows(output, "DDIC_TIMESTAMPS", len(requestedStructures))
	if err != nil {
		return zero, err
	}
	for index, row := range structureRows {
		path := fmt.Sprintf("RFC_METADATA_GET_TIMESTAMP output DDIC_TIMESTAMPS[%d]", index)
		tnRaw, err := mgText(row, "TYPENAME", path, 30)
		if err != nil {
			return zero, err
		}
		structureName, err := mgMetadataName(tnRaw, path+".TYPENAME")
		if err != nil {
			return zero, err
		}
		if !structureSet[structureName] {
			return zero, fmt.Errorf("%w: TIMESTAMP returned unrequested structure %s", ErrMetadataGet, structureName)
		}
		if structureOutcomes[structureName] {
			return zero, fmt.Errorf("%w: TIMESTAMP returned duplicate outcome for structure %s", ErrMetadataGet, structureName)
		}
		timestamp, err := mgFixedDigits(row, "TIMESTAMP", path, 14)
		if err != nil {
			return zero, err
		}
		structureOutcomes[structureName] = true
		structures[structureName] = RfcStructureMetadataTimestamp{StructureName: structureName, Timestamp: timestamp, Token: "structure:" + timestamp}
	}

	functionErrors, err := timestampErrorRows(output, "FUNC_ERRORS", "FUNCNAME", functionSet, functionOutcomes, "function")
	if err != nil {
		return zero, err
	}
	structureErrors, err := timestampErrorRows(output, "DD_ERRORS", "TABNAME", structureSet, structureOutcomes, "structure")
	if err != nil {
		return zero, err
	}
	for _, name := range requestedFunctions {
		if !functionOutcomes[name] {
			return zero, fmt.Errorf("%w: TIMESTAMP returned no outcome for function %s", ErrMetadataGet, name)
		}
	}
	for _, name := range requestedStructures {
		if !structureOutcomes[name] {
			return zero, fmt.Errorf("%w: TIMESTAMP returned no outcome for structure %s", ErrMetadataGet, name)
		}
	}
	return RfcMetadataTimestampBatch{Functions: functions, Structures: structures, FunctionErrors: functionErrors, StructureErrors: structureErrors}, nil
}

func setOf(names []string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}
