// SPDX-License-Identifier: Apache-2.0
//
// Live exploration of read-only function modules against a real SAP system.
// Skipped unless OPEN_RFC_LIVE=1. These tests discover each function's
// interface at runtime (RFC_GET_FUNCTION_INTERFACE) and each structure's
// layout at runtime (RFC_GET_STRUCTURE_DEFINITION) — no interface is hardcoded,
// which is the point of an SDK-free client. Credentials come from the
// environment; none are stored here.

package client

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/structure"
)

func liveSession(t *testing.T) (*Session, context.Context) {
	t.Helper()
	if os.Getenv("OPEN_RFC_LIVE") != "1" {
		t.Skip("set OPEN_RFC_LIVE=1 to run the live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	port := liveAtoiOr(os.Getenv("SAP_PORT"), 3300)
	sess, err := Open(ctx, SessionOptions{
		Host:                     os.Getenv("SAP_HOST"),
		Port:                     port,
		ApplicationServerService: liveEnvOr("SAP_SERVICE", "sapdp00"),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	if err := sess.LogonAndPing(ctx, LogonOptions{
		Client:   liveEnvOr("SAP_CLIENT", "001"),
		User:     os.Getenv("SAP_USER"),
		Password: os.Getenv("SAP_PASSWORD"),
	}); err != nil {
		t.Fatalf("logon: %v", err)
	}
	return sess, ctx
}

func liveInterface(t *testing.T, sess *Session, ctx context.Context, name string) metadata.RfcFunctionInterface {
	t.Helper()
	req, err := metadata.BuildRfcGetFunctionInterfaceRequest(name)
	if err != nil {
		t.Fatalf("build interface request for %s: %v", name, err)
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		t.Fatalf("RFC_GET_FUNCTION_INTERFACE(%s): %v", name, err)
	}
	iface, err := metadata.DecodeRfcFunctionInterfaceResult(name, res.Fields)
	if err != nil {
		t.Fatalf("decode interface for %s: %v", name, err)
	}
	return iface
}

func liveStructDef(t *testing.T, sess *Session, ctx context.Context, name string) rfctypes.RfcStructureDefinition {
	t.Helper()
	req, err := metadata.BuildRfcGetStructureDefinitionRequest(name)
	if err != nil {
		t.Fatalf("build structure request for %s: %v", name, err)
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		t.Fatalf("RFC_GET_STRUCTURE_DEFINITION(%s): %v", name, err)
	}
	def, err := metadata.DecodeRfcStructureDefinitionResult(name, res.Fields)
	if err != nil {
		t.Fatalf("decode structure %s: %v", name, err)
	}
	return def
}

// TestLiveFunctionInterface discovers the interface of a few read-only function
// modules at runtime and prints their parameters.
func TestLiveFunctionInterface(t *testing.T) {
	sess, ctx := liveSession(t)
	for _, name := range []string{"RFC_SYSTEM_INFO", "STFC_STRUCTURE", "STFC_CONNECTION"} {
		iface := liveInterface(t, sess, ctx, name)
		t.Logf("== %s: %d parameters, %d exceptions ==", name, len(iface.Parameters), len(iface.Exceptions))
		for _, p := range iface.Parameters {
			typ := p.TableName
			if typ == "" {
				typ = "(scalar exid " + p.Exid + ")"
			}
			t.Logf("   %-14s class=%s len=%d %s", p.ParameterName, p.ParameterClass, p.InternalLength, typ)
		}
		if len(iface.Exceptions) > 0 {
			t.Logf("   exceptions: %v", iface.Exceptions)
		}
	}
}

// TestLiveSystemInfo calls RFC_SYSTEM_INFO and decodes the returned structure
// with a runtime-discovered definition.
func TestLiveSystemInfo(t *testing.T) {
	sess, ctx := liveSession(t)

	iface := liveInterface(t, sess, ctx, "RFC_SYSTEM_INFO")
	var exportName, structName string
	var bestLen int32 = -1
	for _, p := range iface.Parameters {
		if p.ParameterClass == "E" && p.TableName != "" && p.InternalLength > bestLen {
			exportName, structName, bestLen = p.ParameterName, p.TableName, p.InternalLength
		}
	}
	if exportName == "" {
		t.Fatalf("RFC_SYSTEM_INFO exposed no structured export")
	}
	t.Logf("export %s is structure %s", exportName, structName)

	req, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName:     "RFC_SYSTEM_INFO",
		RequestedOutputs: []string{exportName},
	})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		t.Fatalf("RFC_SYSTEM_INFO: %v", err)
	}
	classic, err := classicrfc.DecodeResult(res.Fields)
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	var raw []byte
	for _, sc := range classic.Scalars {
		if sc.Name == exportName {
			raw = sc.Value
		}
	}
	if raw == nil {
		t.Fatalf("no %s scalar in response", exportName)
	}

	def := liveStructDef(t, sess, ctx, structName)
	decoded, err := structure.Decode(def, raw)
	if err != nil {
		t.Fatalf("structure.Decode(%s): %v", structName, err)
	}
	for _, want := range []string{"RFCPROTO", "RFCCHARTYP", "RFCSYSID", "RFCSAPRL", "RFCDBSYS", "RFCHOST", "RFCKERNRL"} {
		if v, ok := decoded[want]; ok {
			t.Logf("   %-12s = %v", want, v)
		}
	}
	if decoded["RFCSYSID"] == nil {
		t.Fatalf("RFCSYSID missing from decoded RFCSI structure")
	}
}

// TestLiveStructure exercises the canonical structure+table function
// STFC_STRUCTURE: it encodes a structure import, and decodes both the echoed
// structure export and the returned table, all with runtime-discovered layout.
func TestLiveStructure(t *testing.T) {
	sess, ctx := liveSession(t)

	iface := liveInterface(t, sess, ctx, "STFC_STRUCTURE")
	var importName, tableName, structName string
	for _, p := range iface.Parameters {
		switch {
		case p.ParameterClass == "I" && p.TableName != "":
			importName, structName = p.ParameterName, p.TableName
		case p.ParameterClass == "T":
			tableName = p.ParameterName
		}
	}
	var echoName string
	for _, p := range iface.Parameters {
		if p.ParameterClass == "E" && p.TableName == structName && p.ParameterName != importName {
			echoName = p.ParameterName
		}
	}
	t.Logf("import=%s echo=%s table=%s struct=%s", importName, echoName, tableName, structName)
	if importName == "" || echoName == "" || tableName == "" {
		t.Fatalf("STFC_STRUCTURE interface incomplete: %+v", iface.Parameters)
	}

	def := liveStructDef(t, sess, ctx, structName)
	const marker = "open-rfc-go struct roundtrip"
	importBytes, err := structure.Encode(def, map[string]any{"RFCDATA1": marker, "RFCCHAR1": "Z"})
	if err != nil {
		t.Fatalf("structure.Encode: %v", err)
	}

	req, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName:     "STFC_STRUCTURE",
		RequestedOutputs: []string{echoName, "RESPTEXT", tableName},
		Imports:          []cpic.NamedValue{{Name: importName, Value: importBytes}},
	})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		t.Fatalf("STFC_STRUCTURE: %v", err)
	}
	classic, err := classicrfc.DecodeResult(res.Fields)
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}

	// Echoed structure export.
	var echoRaw []byte
	for _, sc := range classic.Scalars {
		switch sc.Name {
		case echoName:
			echoRaw = sc.Value
		case "RESPTEXT":
			resp, _ := classicrfc.DecodeAbapChar(sc.Value)
			t.Logf("RESPTEXT = %q", resp)
		}
	}
	if echoRaw == nil {
		t.Fatalf("no %s export in response", echoName)
	}
	echo, err := structure.Decode(def, echoRaw)
	if err != nil {
		t.Fatalf("decode echo structure: %v", err)
	}
	gotData1, _ := echo["RFCDATA1"].(string)
	if !containsField(gotData1, marker) {
		t.Fatalf("echo RFCDATA1 = %q, want it to contain %q", gotData1, marker)
	}
	t.Logf("echo RFCDATA1 = %q  RFCCHAR1 = %v", gotData1, echo["RFCCHAR1"])

	// Returned table: the server appends one row describing the call.
	for _, tbl := range classic.Tables {
		if tbl.Name != tableName {
			continue
		}
		t.Logf("table %s: %d row(s), rowByteLength=%d encoding=%s", tbl.Name, len(tbl.Rows), tbl.RowByteLength, tbl.RowEncoding)
		for i, row := range tbl.Rows {
			decoded, derr := structure.Decode(def, row)
			if derr != nil {
				t.Fatalf("decode table row %d: %v", i, derr)
			}
			t.Logf("   row[%d] RFCDATA1=%q RFCDATA2=%q", i, decoded["RFCDATA1"], decoded["RFCDATA2"])
		}
	}
}

func containsField(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestLiveFunctionSearch lists function modules matching a pattern via
// RFC_FUNCTION_SEARCH and decodes the returned FUNCTIONS table at runtime.
func TestLiveFunctionSearch(t *testing.T) {
	sess, ctx := liveSession(t)
	iface := liveInterface(t, sess, ctx, "RFC_FUNCTION_SEARCH")
	var patternName, tableName, rowType string
	for _, p := range iface.Parameters {
		if p.ParameterClass == "I" && p.ParameterName == "FUNCNAME" {
			patternName = p.ParameterName
		}
		if p.ParameterClass == "T" {
			tableName, rowType = p.ParameterName, p.TableName
		}
	}
	if patternName == "" || tableName == "" {
		t.Fatalf("unexpected RFC_FUNCTION_SEARCH interface: %+v", iface.Parameters)
	}
	def := liveStructDef(t, sess, ctx, rowType)

	pattern, err := classicrfc.EncodeAbapChar("STFC*", 30)
	if err != nil {
		t.Fatalf("encode pattern: %v", err)
	}
	req, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName:     "RFC_FUNCTION_SEARCH",
		RequestedOutputs: []string{tableName},
		Imports:          []cpic.NamedValue{{Name: patternName, Value: pattern}},
	})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		t.Fatalf("RFC_FUNCTION_SEARCH: %v", err)
	}
	classic, err := classicrfc.DecodeResult(res.Fields)
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}
	found := 0
	for _, tbl := range classic.Tables {
		if tbl.Name != tableName {
			continue
		}
		t.Logf("%s: %d function(s) matching STFC*", tbl.Name, len(tbl.Rows))
		for _, row := range tbl.Rows {
			d, derr := structure.Decode(def, row)
			if derr != nil {
				t.Fatalf("decode row: %v", derr)
			}
			t.Logf("   %-30v group=%v", d["FUNCNAME"], d["GROUPNAME"])
			found++
		}
	}
	if found == 0 {
		t.Fatalf("no functions returned for STFC*")
	}
}

// TestLiveReadTable reads a few rows of a small, safe dictionary table (T000,
// the client table) through RFC_READ_TABLE, using the returned FIELDS layout to
// slice each DATA work area. This is the canonical "look into a table" call.
func TestLiveReadTable(t *testing.T) {
	sess, ctx := liveSession(t)

	queryTable, err := classicrfc.EncodeAbapChar("T000", 30)
	if err != nil {
		t.Fatalf("encode QUERY_TABLE: %v", err)
	}
	rowcount := make([]byte, 4)
	binary.LittleEndian.PutUint32(rowcount, 5)

	req, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName:     "RFC_READ_TABLE",
		RequestedOutputs: []string{"DATA", "FIELDS"},
		Imports: []cpic.NamedValue{
			{Name: "QUERY_TABLE", Value: queryTable},
			{Name: "ROWCOUNT", Value: rowcount},
		},
	})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	res, err := sess.CallRaw(ctx, req)
	if err != nil {
		t.Fatalf("RFC_READ_TABLE: %v", err)
	}
	if !res.Success {
		t.Fatalf("RFC_READ_TABLE returned failure")
	}
	classic, err := classicrfc.DecodeResult(res.Fields)
	if err != nil {
		t.Fatalf("decode result: %v", err)
	}

	fieldsDef := liveStructDef(t, sess, ctx, "RFC_DB_FLD")
	dataDef := liveStructDef(t, sess, ctx, "TAB512")

	type column struct {
		name          string
		offset, width int
	}
	var columns []column
	var dataRows [][]byte
	for _, tbl := range classic.Tables {
		switch tbl.Name {
		case "FIELDS":
			for _, row := range tbl.Rows {
				d, derr := structure.Decode(fieldsDef, row)
				if derr != nil {
					t.Fatalf("decode FIELDS row: %v", derr)
				}
				off, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(d["OFFSET"])))
				w, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(d["LENGTH"])))
				columns = append(columns, column{name: fmt.Sprint(d["FIELDNAME"]), offset: off, width: w})
			}
		case "DATA":
			dataRows = tbl.Rows
		}
	}
	if len(columns) == 0 || len(dataRows) == 0 {
		t.Fatalf("RFC_READ_TABLE returned %d columns and %d rows", len(columns), len(dataRows))
	}
	t.Logf("T000: %d columns, %d rows", len(columns), len(dataRows))
	for i, row := range dataRows {
		d, derr := structure.Decode(dataDef, row)
		if derr != nil {
			t.Fatalf("decode DATA row: %v", derr)
		}
		wa := fmt.Sprint(d["WA"])
		runes := []rune(wa)
		var parts []string
		for _, c := range columns {
			if c.offset+c.width <= len(runes) {
				parts = append(parts, fmt.Sprintf("%s=%q", c.name, strings.TrimRight(string(runes[c.offset:c.offset+c.width]), " ")))
			}
		}
		t.Logf("   row[%d] %s", i, strings.Join(parts, " "))
	}
}
