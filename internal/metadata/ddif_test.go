// SPDX-License-Identifier: Apache-2.0
//
// Original conformance test for the DDIF_FIELDINFO_GET decoder ported from
// open-rfc src/metadata/ddif-fieldinfo.ts. Upstream has no isolated test for
// it; this states its wire facts, including the DFIES 1074-byte stable-prefix
// bound (recurring-bug-class fix) and X030L/DFIES cross-checks. See
// docs/provenance.md.

package metadata

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/cpic"
)

func putCharAt(buf []byte, off int, s string, chars int) {
	enc := classicMustChar(s, chars)
	copy(buf[off:], enc)
}

func putNumcAt(buf []byte, off, byteLen int, n int) {
	chars := byteLen / 2
	s := strings.Repeat("0", chars)
	d := []byte(s)
	num := []byte(strings.TrimLeft(padLeft(n, chars), ""))
	copy(d[chars-len(num):], num)
	putCharAt(buf, off, string(d), chars)
}

func padLeft(n, width int) string {
	s := itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func dfiesRow(table, field string, position, offset, intlen, decimals int, exid, componentType string) []byte {
	row := make([]byte, dfiesMinimumUnicodeRowLength)
	putCharAt(row, 0, table, 30)
	putCharAt(row, 60, field, 30)
	putNumcAt(row, 122, 8, position)
	putNumcAt(row, 130, 12, offset)
	putNumcAt(row, 334, 12, intlen)
	putNumcAt(row, 358, 12, decimals)
	putCharAt(row, 378, exid, 1)
	putCharAt(row, 1072, componentType, 1)
	return row
}

func x030lWA(name string, fieldCount, byteLength int, tableType string) []byte {
	b := make([]byte, x030lMinimumUnicodeLength)
	putCharAt(b, 0, name, 30)
	binary.BigEndian.PutUint16(b[162:], uint16(fieldCount))
	binary.BigEndian.PutUint32(b[164:], uint32(byteLength))
	putCharAt(b, 172, tableType, 1)
	b[248] = 2
	return b
}

func TestDecodeDdIfFieldInfoGetResult(t *testing.T) {
	fields := []cpic.Field{
		fldm(cpic.TagParameterName, u16m("DDOBJTYPE")), fldm(cpic.TagParameterValue, classicMustChar("TABL", 4)),
		fldm(cpic.TagParameterName, u16m("X030L_WA")), fldm(cpic.TagParameterValue, x030lWA("ZDDIC", 2, 12, "")),
		fldm(cpic.TagTableName, u16m("DFIES_TAB")), fldm(cpic.TagTableHeader, tableHeader(dfiesMinimumUnicodeRowLength, 2)),
		fldm(cpic.TagTableContent, dfiesRow("ZDDIC", "A", 1, 0, 8, 0, "C", "E")),
		fldm(cpic.TagTableContent, dfiesRow("ZDDIC", "B", 2, 8, 4, 0, "I", "E")),
		fldm(cpic.TagEnd, nil),
	}
	def, err := DecodeDdIfFieldInfoGetResult("ZDDIC", fields)
	if err != nil {
		t.Fatal(err)
	}
	if def.Name != "ZDDIC" || def.ByteLength != 12 || len(def.Fields) != 2 {
		t.Fatalf("def = %+v", def)
	}
	if def.Fields[0].FieldName != "A" || def.Fields[0].Exid != "C" || def.Fields[0].InternalLength != 8 {
		t.Fatalf("field A = %+v", def.Fields[0])
	}
	if def.Fields[1].FieldName != "B" || def.Fields[1].Exid != "I" || def.Fields[1].Offset != 8 {
		t.Fatalf("field B = %+v", def.Fields[1])
	}
}

func TestDdIfRejects(t *testing.T) {
	base := func(objType string, x030l []byte, rows [][]byte) []cpic.Field {
		f := []cpic.Field{
			fldm(cpic.TagParameterName, u16m("DDOBJTYPE")), fldm(cpic.TagParameterValue, classicMustChar(objType, 4)),
			fldm(cpic.TagParameterName, u16m("X030L_WA")), fldm(cpic.TagParameterValue, x030l),
			fldm(cpic.TagTableName, u16m("DFIES_TAB")), fldm(cpic.TagTableHeader, tableHeader(dfiesMinimumUnicodeRowLength, uint32(len(rows)))),
		}
		for _, r := range rows {
			f = append(f, fldm(cpic.TagTableContent, r))
		}
		return append(f, fldm(cpic.TagEnd, nil))
	}
	// DTEL unsupported.
	if _, err := DecodeDdIfFieldInfoGetResult("ZDDIC", base("DTEL", x030lWA("ZDDIC", 1, 8, ""), [][]byte{dfiesRow("ZDDIC", "A", 1, 0, 8, 0, "C", "E")})); err == nil || !strings.Contains(err.Error(), "unsupported DDIC object kind DTEL") {
		t.Fatalf("DTEL: %v", err)
	}
	// fieldCount mismatch.
	if _, err := DecodeDdIfFieldInfoGetResult("ZDDIC", base("TABL", x030lWA("ZDDIC", 5, 8, ""), [][]byte{dfiesRow("ZDDIC", "A", 1, 0, 8, 0, "C", "E")})); err == nil || !strings.Contains(err.Error(), "advertises 5 fields") {
		t.Fatalf("count mismatch: %v", err)
	}
	// Table/vector type L refused.
	if _, err := DecodeDdIfFieldInfoGetResult("ZDDIC", base("TABL", x030lWA("ZDDIC", 1, 8, "L"), [][]byte{dfiesRow("ZDDIC", "A", 1, 0, 8, 0, "C", "E")})); err == nil || !strings.Contains(err.Error(), "table/vector type") {
		t.Fatalf("L type: %v", err)
	}
	// Unsupported component type.
	if _, err := DecodeDdIfFieldInfoGetResult("ZDDIC", base("TABL", x030lWA("ZDDIC", 1, 8, ""), [][]byte{dfiesRow("ZDDIC", "A", 1, 0, 8, 0, "C", "S")})); err == nil || !strings.Contains(err.Error(), "unsupported component type") {
		t.Fatalf("component type: %v", err)
	}
}

func TestBuildDdIfRequest(t *testing.T) {
	req, err := BuildDdIfFieldInfoGetRequest("ZDDIC", "E")
	if err != nil || len(req) == 0 {
		t.Fatalf("build: %v", err)
	}
	if _, err := cpic.InspectRequestAppcFraming(req); err != nil {
		t.Fatalf("framing: %v", err)
	}
}
