// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"context"
	"fmt"
	"strings"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
)

// ToolSchema is an MCP-tool-shaped description of an RFC function module: a name,
// a human description, and JSON Schema for the inputs and outputs. It is produced
// purely from cached metadata (no call is executed), so an MCP-trained model — or
// a person — can read an FM's interface the same way it reads any tool, and drive
// Client.Call from it. inputSchema carries IMPORTING/CHANGING/TABLES parameters;
// outputSchema carries EXPORTING/CHANGING/TABLES.
type ToolSchema struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema"`
}

// DescribeTool renders an RFC function module's interface as an MCP-tool JSON
// Schema. Structure and table parameters are expanded from their DDIC layout.
func (c *Client) DescribeTool(ctx context.Context, functionName string) (ToolSchema, error) {
	iface, err := c.FunctionInterface(ctx, functionName)
	if err != nil {
		return ToolSchema{}, err
	}
	resolve := func(name string) map[string]any {
		def, err := c.StructureDefinition(ctx, name)
		if err != nil {
			return map[string]any{"type": "object", "description": name + " (unresolved)"}
		}
		return structSchema(def)
	}
	inProps := map[string]any{}
	outProps := map[string]any{}
	var required []string
	for _, p := range iface.Parameters {
		sc := paramSchema(p, resolve)
		switch p.ParameterClass {
		case "I":
			inProps[p.ParameterName] = sc
			if !p.Optional {
				required = append(required, p.ParameterName)
			}
		case "E":
			outProps[p.ParameterName] = sc
		case "C", "T":
			inProps[p.ParameterName] = sc
			outProps[p.ParameterName] = sc
		}
	}
	input := map[string]any{"type": "object", "properties": inProps}
	if len(required) > 0 {
		input["required"] = required
	}
	desc := fmt.Sprintf("RFC function module %s", iface.Name)
	if len(iface.Exceptions) > 0 {
		desc += " (exceptions: " + strings.Join(iface.Exceptions, ", ") + ")"
	}
	return ToolSchema{
		Name:         sanitizeToolName(iface.Name),
		Description:  desc,
		InputSchema:  input,
		OutputSchema: map[string]any{"type": "object", "properties": outProps},
	}, nil
}

// sanitizeToolName maps an FM name (RS38L-NAME, ≤30 chars) to the MCP tool-name
// charset. Namespaced names (/NS/NAME) become _NS_NAME.
func sanitizeToolName(fm string) string {
	return strings.Trim(strings.ReplaceAll(fm, "/", "_"), "_")
}

// paramSchema maps one function parameter to a JSON Schema fragment.
func paramSchema(p classicrfc.FunintParameter, resolve func(string) map[string]any) map[string]any {
	switch {
	// A TABLES parameter (class T) is an array even though its row EXID is a
	// structure (u/v) — check the class before the structure EXID.
	case p.ParameterClass == "T" || isTableExid(p.Exid):
		return map[string]any{"type": "array", "items": resolve(p.TableName)}
	case isStructureExid(p.Exid):
		return resolve(p.TableName)
	}
	sc := fieldSchema(p.Exid, p.InternalLength, p.Decimals)
	if p.ParameterText != "" {
		sc["description"] = p.ParameterText
	}
	return sc
}

// structSchema maps a DDIC structure/row layout to a JSON Schema object.
func structSchema(def rfctypes.RfcStructureDefinition) map[string]any {
	props := map[string]any{}
	for _, f := range def.Fields {
		props[f.FieldName] = fieldSchema(f.Exid, f.InternalLength, f.Decimals)
	}
	return map[string]any{"type": "object", "description": def.Name, "properties": props}
}

// fieldSchema maps one classic RFC EXID to a JSON Schema type.
func fieldSchema(exid string, internalLength, decimals int32) map[string]any {
	switch exid {
	case "C", "N":
		s := map[string]any{"type": "string", "maxLength": int(internalLength / 2)}
		if exid == "N" {
			s["pattern"] = "^[0-9]*$"
		}
		return s
	case "g":
		return map[string]any{"type": "string", "description": "STRING"}
	case "I", "s", "b", "8":
		return map[string]any{"type": "integer"}
	case "F", "a", "e":
		return map[string]any{"type": "number"}
	case "P":
		s := map[string]any{"type": "number"}
		if decimals > 0 {
			s["description"] = fmt.Sprintf("DEC(decimals=%d)", decimals)
		}
		return s
	case "X", "y":
		return map[string]any{"type": "string", "contentEncoding": "base64"}
	case "D":
		return map[string]any{"type": "string", "pattern": `^\d{8}$`, "description": "DATS YYYYMMDD"}
	case "T":
		return map[string]any{"type": "string", "pattern": `^\d{6}$`, "description": "TIMS HHMMSS"}
	case "p":
		return map[string]any{"type": "string", "description": "UTCLONG timestamp"}
	}
	return map[string]any{"type": "string", "description": "exid=" + exid}
}
