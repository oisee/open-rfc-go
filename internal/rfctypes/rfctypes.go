// SPDX-License-Identifier: Apache-2.0
//
// Shared classic-RFC structure-definition types, extracted into a leaf package
// so both the metadata decoder (internal/metadata) and the structure value
// codec (internal/structure) can use them without an import cycle. Original
// work: the fields mirror open-rfc's RfcStructureField/RfcStructureDefinition
// interfaces from src/metadata/rfc-structure-definition.ts. See docs/provenance.md.

// Package rfctypes holds the shared classic-RFC structure-definition types.
package rfctypes

// RfcStructureField is one field of a classic RFC structure definition.
type RfcStructureField struct {
	TableName      string
	FieldName      string
	Position       int32
	Offset         int32
	InternalLength int32
	Decimals       int32
	Exid           string
}

// RfcStructureDefinition is a validated classic RFC structure definition.
type RfcStructureDefinition struct {
	Name       string
	ByteLength int32
	Fields     []RfcStructureField
}
