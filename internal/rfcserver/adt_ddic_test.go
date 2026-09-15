// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
	"github.com/oisee/open-rfc-go/internal/metadata"
)

// The rows this server writes, read by the client-side decoders this
// repository already has. Those were ported from a client that reads live
// systems, so agreeing with them is agreeing with what a system sends.
func TestDfiesRowsReadAsTheClientReadsThem(t *testing.T) {
	want := []struct {
		field, exid      string
		offset, intlen   int32
		comptype, lfield string
	}{
		{"STATUS_LINE", "v", 0, 24, "S", "STATUS_LINE"},
		{"VERSION", "g", 0, 8, "", "STATUS_LINE-VERSION"},
		{"STATUS_CODE", "g", 8, 8, "", "STATUS_LINE-STATUS_CODE"},
		{"REASON_PHRASE", "g", 16, 8, "", "STATUS_LINE-REASON_PHRASE"},
		{"HEADER_FIELDS", "h", 24, 8, "L", "HEADER_FIELDS"},
		{"MESSAGE_BODY", "y", 32, 8, "", "MESSAGE_BODY"},
	}
	typ := adtDictionary[adtRestResponse]
	if len(typ.fields) != len(want) {
		t.Fatalf("%d fields, want %d", len(typ.fields), len(want))
	}
	for i, f := range typ.fields {
		row, err := dfiesRow(dfiesValues(typ.name, f))
		if err != nil {
			t.Fatal(err)
		}
		if len(row) != dfiesRowLength {
			t.Fatalf("row %d is %d bytes", i, len(row))
		}
		got, err := metadata.DecodeDdIfDfiesRow(row)
		if err != nil {
			t.Fatalf("row %d (%s): the client decoder refuses it: %v", i, f.name, err)
		}
		w := want[i]
		if got.TableName != adtRestResponse || got.FieldName != w.field || got.Exid != w.exid ||
			got.Position != int32(i+1) || got.Offset != w.offset || got.InternalLength != w.intlen {
			t.Fatalf("row %d decodes as %+v, want %+v", i, got.RfcStructureField, w)
		}
		// the two columns the client decoder does not read, sliced by the layout
		if got := column(row, "COMPTYPE"); got != w.comptype {
			t.Fatalf("row %d COMPTYPE %q, want %q", i, got, w.comptype)
		}
		if got := column(row, "LFIELDNAME"); got != w.lfield {
			t.Fatalf("row %d LFIELDNAME %q, want %q", i, got, w.lfield)
		}
	}
}

// column slices one DFIES column out of a row by the layout.
func column(row []byte, name string) string {
	off := 0
	for _, col := range dfiesLayout {
		if col.name == name {
			s, _ := classicrfc.DecodeAbapChar(row[off:off+2*col.width], col.width)
			return s
		}
		off += 2 * col.width
	}
	return ""
}

func TestDfiesInitialRow(t *testing.T) {
	row, err := dfiesRow(nil)
	if err != nil {
		t.Fatal(err)
	}
	if column(row, "TABNAME") != "" || column(row, "POSITION") != "0000" || column(row, "OFFSET") != "000000" {
		t.Fatalf("initial row is not blank with zero-filled NUMCs")
	}
	if len(row) != dfiesRowLength {
		t.Fatalf("%d bytes", len(row))
	}
}

func TestFunintRowsReadAsTheClientReadsThem(t *testing.T) {
	handler := FunctionInterfaceHandler()
	req := Request{
		Eclipse: true, Order: binary.BigEndian,
		Imports:          []cpic.NamedValue{{Name: "FUNCNAME", Value: utf16BE(adtRestFunction + strings.Repeat(" ", 8))}},
		RequestedOutputs: []string{"REMOTE_BASXML_SUPPORTED"},
		Tables:           []Table{{Name: "PARAMS", RowByteLength: 404, ID: []byte{0, 0, 0, 1}}},
	}
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Outputs) != 1 || resp.Outputs[0].Name != "REMOTE_BASXML_SUPPORTED" || resp.Outputs[0].XML {
		t.Fatalf("outputs: %+v", resp.Outputs)
	}
	if len(resp.Tables) != 1 || resp.Tables[0].Name != "PARAMS" || len(resp.Tables[0].Rows) != 2 || !bytes.Equal(resp.Tables[0].ID, []byte{0, 0, 0, 1}) {
		t.Fatalf("tables: %+v", resp.Tables)
	}
	var names []string
	for _, row := range resp.Tables[0].Rows {
		p, err := classicrfc.DecodeFunintRow(row)
		if err != nil {
			t.Fatalf("the client decoder refuses a PARAMS row: %v", err)
		}
		names = append(names, p.ParameterClass+":"+p.ParameterName+":"+p.TableName+":"+p.Exid)
		if p.InternalLength != 40 {
			t.Fatalf("%s has length %d", p.ParameterName, p.InternalLength)
		}
	}
	if strings.Join(names, " ") != "E:RESPONSE:SADT_REST_RESPONSE:v I:REQUEST:SADT_REST_REQUEST:v" {
		t.Fatalf("parameters: %v", names)
	}
	// and the whole thing decodes as the client's metadata layer decodes it
	cut, err := EncodeEclipseResponse(resp, req)
	if err != nil {
		t.Fatal(err)
	}
	fields := responseFields(t, cut)
	want := []uint16{tagFirstTableID, 0x0503, 0x0420, 0x0512, 0x0205, 0x0201, 0x0203, 0x0130,
		tagTableIDHeader, 0x0302, tagTableRow, tagTableRow, tagTableIDEnd, 0x0667, 0xffff}
	if got := tagsOf(fields); !sameTags(got, want) {
		t.Fatalf("response tags\n got %04x\nwant %04x", got, want)
	}
	if h := valuesOf(fields, tagTableIDHeader)[0]; !bytes.Equal(h, []byte{0, 0, 0, 0x0a, 0, 0, 0, 1, 0, 0, 0, 2}) {
		t.Fatalf("table header %x", h)
	}
	if g := valuesOf(fields, 0x0302)[0]; binary.BigEndian.Uint32(g) != 402 || binary.BigEndian.Uint32(g[4:]) != 2 {
		t.Fatalf("table geometry %x", g)
	}
}

func TestFunctionInterfaceRefusesOtherFunctions(t *testing.T) {
	_, err := FunctionInterfaceHandler()(context.Background(), Request{
		Order:   binary.BigEndian,
		Imports: []cpic.NamedValue{{Name: "FUNCNAME", Value: utf16BE("RFC_PING")}},
	})
	var exc *Exception
	if !errors.As(err, &exc) || exc.Key != UnknownFunctionKey {
		t.Fatalf("err = %v, want %s", err, UnknownFunctionKey)
	}
}

func TestFieldInfoAnswerHasTheSystemsShape(t *testing.T) {
	guid := bytes.Repeat([]byte{0xab}, 16)
	req := Request{
		Eclipse: true, Order: binary.BigEndian, SessionGUID: guid,
		Imports: []cpic.NamedValue{
			{Name: "TABNAME", Value: utf16BE(adtRestRequest + strings.Repeat(" ", 13))},
			{Name: "ALL_TYPES", Value: utf16BE("X")},
			{Name: "UCLEN", Value: []byte{2}},
		},
		RequestedOutputs: []string{"DFIES_WA", "X030L_WA", "LINES_DESCR"},
		Tables:           []Table{{Name: "DFIES_TAB", RowByteLength: 1338, ID: []byte{0, 0, 0, 4}}},
	}
	resp, err := FieldInfoHandler()(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	cut, err := EncodeEclipseResponse(resp, req)
	if err != nil {
		t.Fatal(err)
	}
	fields := responseFields(t, cut)
	// measured: 0500 0331 0503 0514 0420 0512 0205×3 0201 0203 0201 0203 3c02 3c05… 3c02 0130 0335 0302 0303×6 0336 0667 ffff
	// DFIES_WA scalar, then the LINES_DESCR xRFC, then the X030L_WA scalar —
	// the parameter order of DDIF_FIELDINFO_GET, which the oracle confirms.
	want := []uint16{tagFirstTableID, 0x0503, 0x0514, 0x0420, 0x0512, 0x0205, 0x0205, 0x0205,
		0x0201, 0x0203, 0x3c02}
	got := tagsOf(fields)
	if !sameTags(got[:len(want)], want) {
		t.Fatalf("response opens\n got %04x\nwant %04x", got[:len(want)], want)
	}
	// after the LINES_DESCR chunks: the X030L_WA scalar
	x030lAt := indexOf(got, 0x0201, indexOf(got, 0x3c02, 0)+1)
	if x030lAt < 0 || got[x030lAt+1] != 0x0203 || len(valuesOf(fields, 0x0203)[1]) != x030lLength {
		t.Fatalf("X030L_WA does not follow the LINES_DESCR: %04x", got)
	}
	// ends with the X030L_WA scalar, then the program, then the six DFIES rows
	tail := []uint16{0x0203, 0x0130, tagTableIDHeader, 0x0302, tagTableRow, tagTableRow, tagTableRow, tagTableRow, tagTableRow, tagTableRow, tagTableIDEnd, 0x0667, 0xffff}
	if !sameTags(got[len(got)-len(tail):], tail) {
		t.Fatalf("response ends\n got %04x\nwant %04x", got[len(got)-len(tail):], tail)
	}
	if !bytes.Equal(valuesOf(fields, 0x0514)[0], guid) {
		t.Fatalf("the session GUID is not echoed")
	}
	values := valuesOf(fields, 0x0203)
	if len(values[0]) != dfiesRowLength || len(values[1]) != x030lLength {
		t.Fatalf("DFIES_WA is %d bytes and X030L_WA %d", len(values[0]), len(values[1]))
	}
	x := values[1]
	if binary.BigEndian.Uint16(x[162:]) != 6 || binary.BigEndian.Uint32(x[164:]) != 40 || x[172] != 'J' || x[248] != 2 {
		t.Fatalf("X030L_WA geometry: count %d length %d kind %c uclen %d",
			binary.BigEndian.Uint16(x[162:]), binary.BigEndian.Uint32(x[164:]), x[172], x[248])
	}
	chunks := valuesOf(fields, 0x3c05)
	if string(chunks[0]) != "<LINES_DESCR>" {
		t.Fatalf("the first xRFC chunk is %q, the system sends the open tag alone", chunks[0])
	}
	var xml []byte
	for _, c := range chunks {
		xml = append(xml, c...)
	}
	for _, needle := range []string{"<TYPENAME>IHTTPNVP</TYPENAME><TYPEKIND>STRU</TYPEKIND>", "<TYPENAME>TIHTTPNVP</TYPENAME><TYPEKIND>TTYP</TYPEKIND>",
		"<FIELDNAME>NAME</FIELDNAME>", "<ROLLNAME>IHTTPVAL</ROLLNAME>", "<COMPTYPE>T</COMPTYPE>", "</LINES_DESCR>"} {
		if !strings.Contains(string(xml), needle) {
			t.Fatalf("LINES_DESCR lacks %s", needle)
		}
	}
	// the table type, asked on its own
	req.Imports[0].Value = utf16BE(adtHeaderTable + strings.Repeat(" ", 21))
	req.RequestedOutputs = []string{"X030L_WA"}
	req.Tables = nil
	resp, err = FieldInfoHandler()(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	x = resp.Outputs[0].Value
	if binary.BigEndian.Uint16(x[162:]) != 2 || binary.BigEndian.Uint32(x[164:]) != 16 || x[172] != 'L' {
		t.Fatalf("TIHTTPNVP X030L_WA: count %d length %d kind %c", binary.BigEndian.Uint16(x[162:]), binary.BigEndian.Uint32(x[164:]), x[172])
	}
	if got, _ := classicrfc.DecodeAbapChar(x[176:236], 30); got != adtHeaderRow {
		t.Fatalf("line type %q", got)
	}
}

func indexOf(tags []uint16, tag uint16, from int) int {
	for i := from; i < len(tags); i++ {
		if tags[i] == tag {
			return i
		}
	}
	return -1
}

func TestFieldInfoRefusesUnknownTypes(t *testing.T) {
	_, err := FieldInfoHandler()(context.Background(), Request{
		Order:   binary.BigEndian,
		Imports: []cpic.NamedValue{{Name: "TABNAME", Value: utf16BE("T100")}},
	})
	var exc *Exception
	if !errors.As(err, &exc) || exc.Key != "NOT_FOUND" {
		t.Fatalf("err = %v", err)
	}
}

// The dictionary and the type graph describe the same interface.
func TestDictionaryAgreesWithTheGraph(t *testing.T) {
	graph := ADTRestGraph()
	for _, p := range graph.Parameters {
		typ, ok := adtDictionary[p.AssociatedType]
		if !ok {
			t.Fatalf("the graph's %s is not in the dictionary", p.AssociatedType)
		}
		node := graph.Nodes[p.AssociatedType]
		for _, gf := range node.Fields {
			found := false
			for _, df := range typ.fields {
				if df.name == gf.Name && df.inttype == gf.InternalType {
					found = true
				}
			}
			if !found {
				t.Fatalf("%s.%s (%s) is in the graph and not in the dictionary", node.Name, gf.Name, gf.InternalType)
			}
		}
	}
	row := adtDictionary[adtHeaderRow]
	for i, gf := range graph.Nodes[adtHeaderRow].Fields {
		if row.fields[i].name != gf.Name {
			t.Fatalf("header row field %d is %s in the dictionary and %s in the graph", i, row.fields[i].name, gf.Name)
		}
	}
}

// The rows RFC_GET_STRUCTURE_DEFINITION returns, read by the client-side
// decoder this repository already ports.
func TestStructureDefinitionReadsAsTheClientReadsIt(t *testing.T) {
	req := Request{
		Eclipse: true, Order: binary.BigEndian,
		Imports:          []cpic.NamedValue{{Name: "TABNAME", Value: utf16BE(adtRestRequest + strings.Repeat(" ", 13))}},
		RequestedOutputs: []string{"TABLENGTH", "FIELDS"},
	}
	resp, err := StructureDefinitionHandler()(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Outputs) != 1 || resp.Outputs[0].Name != "TABLENGTH" || binary.LittleEndian.Uint32(resp.Outputs[0].Value) != adtStructureByteLength {
		t.Fatalf("TABLENGTH: %+v", resp.Outputs)
	}
	// the flat view: three fields, not the six of the nested DDIC description
	if len(resp.Tables) != 1 || resp.Tables[0].Name != "FIELDS" || len(resp.Tables[0].Rows) != 3 {
		t.Fatalf("FIELDS: %+v", resp.Tables)
	}
	want := []string{"REQUEST_LINE:v", "HEADER_FIELDS:h", "MESSAGE_BODY:y"}
	for i, row := range resp.Tables[0].Rows {
		got, err := metadata.DecodeRfcFieldsRow(row)
		if err != nil {
			t.Fatalf("row %d: the client decoder refuses it: %v", i, err)
		}
		if got.TableName != adtRestRequest || got.FieldName+":"+got.Exid != want[i] || got.Position != int32(i+1) {
			t.Fatalf("row %d decodes as %s:%s pos %d, want %s", i, got.FieldName, got.Exid, got.Position, want[i])
		}
	}
	// a caller that named the table among its outputs but sent no table
	// parameter is still answered, in the classic framing
	cut, err := EncodeEclipseResponse(resp, req)
	if err != nil {
		t.Fatal(err)
	}
	fields := responseFields(t, cut)
	if len(valuesOf(fields, uint16(cpic.TagTableName))) != 1 {
		t.Fatalf("the FIELDS table was not named in the answer: %04x", tagsOf(fields))
	}
	if n := len(valuesOf(fields, tagTableRow)); n != 3 {
		t.Fatalf("%d rows on the wire", n)
	}
	// and the line structure a client asks for next
	req.Imports[0].Value = utf16BE(adtRestRequestLine + strings.Repeat(" ", 8))
	line, err := StructureDefinitionHandler()(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(line.Tables[0].Rows) != 3 {
		t.Fatalf("%s has %d fields", adtRestRequestLine, len(line.Tables[0].Rows))
	}
	for i, row := range line.Tables[0].Rows {
		got, _ := metadata.DecodeRfcFieldsRow(row)
		if got.Offset != int32(i*8) || got.InternalLength != 8 {
			t.Fatalf("%s field %d at %d len %d", adtRestRequestLine, i, got.Offset, got.InternalLength)
		}
	}
}
