// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/metadata/rfc-function-interface.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Thrown errors →
// returned wrapped sentinels. The PARAMS row width is bounded below by the
// 402-byte stable prefix, not pinned (recurring-bug-class fix). See
// docs/provenance.md.

package metadata

import (
	"errors"
	"fmt"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
)

// ErrFunctionInterface reports a malformed RFC_GET_FUNCTION_INTERFACE response.
var ErrFunctionInterface = errors.New("metadata: invalid function interface")

var functionInterfaceOutputs = []string{
	"REMOTE_BASXML_SUPPORTED", "REMOTE_CALL", "UPDATE_TASK", "PARAMS", "RESUMABLE_EXCEPTIONS",
}

// RfcFunctionInterface is a decoded RFC_GET_FUNCTION_INTERFACE result.
type RfcFunctionInterface struct {
	Name                       string
	RemoteBasxmlSupported      bool
	RemoteCall                 string
	UpdateTask                 bool
	Parameters                 []classicrfc.FunintParameter
	Exceptions                 []string
	ResumableExceptionRowCount int
}

// BuildRfcGetFunctionInterfaceRequest builds the metadata bootstrap call.
func BuildRfcGetFunctionInterfaceRequest(functionName string) ([]byte, error) {
	funcname, err := classicrfc.EncodeAbapChar(functionName, 30)
	if err != nil {
		return nil, err
	}
	x, err := classicrfc.EncodeAbapChar("X", 1)
	if err != nil {
		return nil, err
	}
	return cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName:     "RFC_GET_FUNCTION_INTERFACE",
		RequestedOutputs: functionInterfaceOutputs,
		Imports: []cpic.NamedValue{
			{Name: "FUNCNAME", Value: funcname},
			{Name: "NONE_UNICODE_LENGTH", Value: x},
		},
	})
}

func requiredScalar(scalars []classicrfc.Scalar, name, context string) ([]byte, error) {
	for _, s := range scalars {
		if s.Name == name {
			return s.Value, nil
		}
	}
	return nil, fmt.Errorf("%w: %s response lacks scalar %s", ErrFunctionInterface, context, name)
}

func decodeFlag(value []byte, name string) (bool, error) {
	decoded, err := classicrfc.DecodeAbapChar(value, 1)
	if err != nil {
		return false, err
	}
	if decoded != "" && decoded != "X" {
		return false, fmt.Errorf("%w: %s contains unsupported flag value %s", ErrFunctionInterface, name, decoded)
	}
	return decoded == "X", nil
}

// DecodeRfcFunctionInterfaceResult normalizes a successful response.
func DecodeRfcFunctionInterfaceResult(functionName string, fields []cpic.Field) (RfcFunctionInterface, error) {
	var zero RfcFunctionInterface
	result, err := classicrfc.DecodeResult(fields)
	if err != nil {
		return zero, err
	}
	var params, resumable *classicrfc.Table
	for i := range result.Tables {
		switch result.Tables[i].Name {
		case "PARAMS":
			params = &result.Tables[i]
		case "RESUMABLE_EXCEPTIONS":
			resumable = &result.Tables[i]
		}
	}
	if params == nil {
		return zero, fmt.Errorf("%w: response lacks PARAMS table", ErrFunctionInterface)
	}
	if params.RowByteLength < classicrfc.RfcFunintUnicodeRowLength {
		return zero, fmt.Errorf("%w: PARAMS row width is %d; expected at least %d", ErrFunctionInterface, params.RowByteLength, classicrfc.RfcFunintUnicodeRowLength)
	}
	if resumable == nil {
		return zero, fmt.Errorf("%w: response lacks RESUMABLE_EXCEPTIONS table", ErrFunctionInterface)
	}

	var parameters []classicrfc.FunintParameter
	var exceptions []string
	for _, row := range params.Rows {
		p, err := classicrfc.DecodeFunintRow(row)
		if err != nil {
			return zero, err
		}
		if p.ParameterClass == "X" {
			exceptions = append(exceptions, p.ParameterName)
		} else {
			parameters = append(parameters, p)
		}
	}

	basxml, err := requiredScalar(result.Scalars, "REMOTE_BASXML_SUPPORTED", "RFC_GET_FUNCTION_INTERFACE")
	if err != nil {
		return zero, err
	}
	remoteBasxml, err := decodeFlag(basxml, "REMOTE_BASXML_SUPPORTED")
	if err != nil {
		return zero, err
	}
	remoteCallVal, err := requiredScalar(result.Scalars, "REMOTE_CALL", "RFC_GET_FUNCTION_INTERFACE")
	if err != nil {
		return zero, err
	}
	remoteCall, err := classicrfc.DecodeAbapChar(remoteCallVal, 1)
	if err != nil {
		return zero, err
	}
	updateTaskVal, err := requiredScalar(result.Scalars, "UPDATE_TASK", "RFC_GET_FUNCTION_INTERFACE")
	if err != nil {
		return zero, err
	}
	updateTask, err := decodeFlag(updateTaskVal, "UPDATE_TASK")
	if err != nil {
		return zero, err
	}
	return RfcFunctionInterface{
		Name:                       functionName,
		RemoteBasxmlSupported:      remoteBasxml,
		RemoteCall:                 remoteCall,
		UpdateTask:                 updateTask,
		Parameters:                 parameters,
		Exceptions:                 exceptions,
		ResumableExceptionRowCount: len(resumable.Rows),
	}, nil
}
