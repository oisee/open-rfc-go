// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
)

// What a client asks before it calls: the dictionary of SADT_REST_RFC_ENDPOINT.
//
// Eclipse does not call a function it has not been described. A fresh
// destination — and every bridge is fresh to it, since the cache is keyed by
// the system it was built against — opens with RFC_GET_FUNCTION_INTERFACE for
// the function's parameters and then DDIF_FIELDINFO_GET for each structure
// those name, and only then SADT_REST_RFC_ENDPOINT. Measured on a live
// Eclipse↔A4H session, 2026-09-15: connections 1 and 2 do this, 3 to 10 do
// not, and against the bridge the first call answered FU_NOT_FOUND and Eclipse
// reported "CPIC-CALL: 'CMRCV'; null".
//
// So the two are answered here, from the interface as this file describes it
// and not from anything captured: the rows a dictionary function returns are
// fixed-width records whose layout is what a client reads, and the layout was
// measured (every field boundary of a 1350-byte DFIES row falls where the
// values say it does — see the test) so that it could be written.
//
// The values describe an interface, which is the clean-room line: what the
// parameters are called, what type each is and how long. The texts are ours.

// ddicField is one line of a structure as DDIF_FIELDINFO_GET describes it.
type ddicField struct {
	name       string
	position   int
	offset     int
	rollname   string // the type behind a substructure or a table field
	leng       int
	intlen     int
	datatype   string // STRU, STRG, SSTR, RSTR, TTYP
	inttype    string // v, g, y, h
	precfield  string // the structure an elementary field belongs to
	comptype   string // S substructure, L table, E elementary line row, "" flat
	langu      string
	dynpfld    bool
	noauthch   bool
	lfieldname string
	texts      [5]string // FIELDTEXT, REPTEXT, SCRTEXT_S/M/L
	headlen    string
	scrlen     [3]string
}

// ddicType is one dictionary object: a structure or a table type.
type ddicType struct {
	name     string
	kind     string // INTTAB or TTYP
	length   int    // bytes of one line, all fields being references of eight
	rowtype  string // for a table type: its line type
	fields   []ddicField
	lineOf   []string // table types this structure uses (LINES_DESCR)
	authorid string
}

func stringField(name string, position, offset int, datatype, inttype, precfield, lfieldname string) ddicField {
	return ddicField{name: name, position: position, offset: offset, intlen: 8, datatype: datatype, inttype: inttype,
		precfield: precfield, langu: "E", dynpfld: true, noauthch: true, lfieldname: lfieldname}
}

// adtDictionary is the type graph of adt_metadata.go, described the way the
// dictionary describes it. The test checks the two agree on names and kinds.
var adtDictionary = func() map[string]ddicType {
	restStructure := func(name, line, lineType string) ddicType {
		lineFields := []string{"METHOD", "URI", "VERSION"}
		if line == "STATUS_LINE" {
			lineFields = []string{"VERSION", "STATUS_CODE", "REASON_PHRASE"}
		}
		fields := []ddicField{{name: line, position: 1, offset: 0, rollname: lineType, intlen: 24, datatype: "STRU", inttype: "v", comptype: "S", lfieldname: line}}
		for i, f := range lineFields {
			datatype := "STRG"
			leng := 0
			if f == "STATUS_CODE" {
				// a short string where everything around it is a string:
				// the one asymmetry of this interface
				datatype, leng = "SSTR", 3
			}
			fd := stringField(f, i+2, i*8, datatype, "g", lineType, line+"-"+f)
			fd.leng = leng
			fields = append(fields, fd)
		}
		fields = append(fields,
			ddicField{name: "HEADER_FIELDS", position: 5, offset: 24, rollname: adtHeaderTable, intlen: 8, datatype: "TTYP", inttype: "h", comptype: "L", lfieldname: "HEADER_FIELDS"},
		)
		body := stringField("MESSAGE_BODY", 6, 32, "RSTR", "y", name, "MESSAGE_BODY")
		body.noauthch = false
		fields = append(fields, body)
		return ddicType{name: name, kind: "INTTAB", length: 40, fields: fields, lineOf: []string{adtHeaderRow, adtHeaderTable}}
	}
	headerRow := ddicType{name: adtHeaderRow, kind: "INTTAB", length: 16, fields: []ddicField{
		{name: "NAME", position: 1, offset: 0, rollname: "IHTTPNAM", intlen: 8, datatype: "STRG", inttype: "g", precfield: adtHeaderRow, comptype: "E", langu: "E", dynpfld: true, lfieldname: "NAME",
			texts: [5]string{"HTTP header name", "Name", "Name", "Name", "Name"}, headlen: "04", scrlen: [3]string{"10", "15", "20"}},
		{name: "VALUE", position: 2, offset: 8, rollname: "IHTTPVAL", intlen: 8, datatype: "STRG", inttype: "g", precfield: adtHeaderRow, comptype: "E", langu: "E", dynpfld: true, lfieldname: "VALUE",
			texts: [5]string{"HTTP header value", "Value", "Value", "Value", "Value"}, headlen: "10", scrlen: [3]string{"10", "15", "20"}},
	}}
	headerTable := ddicType{name: adtHeaderTable, kind: "TTYP", length: 16, rowtype: adtHeaderRow, authorid: "TKN", fields: []ddicField{
		{position: 1, rollname: adtHeaderRow, datatype: "STRU", comptype: "T"},
	}}
	// The line structures are types in their own right. Eclipse never asks for
	// them — it reads them out of the parent's LINES_DESCR — but a client that
	// resolves a structure field by field does, so they are here as well.
	lineStructure := func(name string, names ...string) ddicType {
		var fields []ddicField
		for i, n := range names {
			datatype := "STRG"
			leng := 0
			if n == "STATUS_CODE" {
				datatype, leng = "SSTR", 3
			}
			f := stringField(n, i+1, i*8, datatype, "g", name, n)
			f.leng = leng
			fields = append(fields, f)
		}
		return ddicType{name: name, kind: "INTTAB", length: 24, fields: fields}
	}
	types := []ddicType{
		restStructure(adtRestRequest, "REQUEST_LINE", adtRestRequestLine),
		restStructure(adtRestResponse, "STATUS_LINE", adtRestStatusLine),
		lineStructure(adtRestRequestLine, "METHOD", "URI", "VERSION"),
		lineStructure(adtRestStatusLine, "VERSION", "STATUS_CODE", "REASON_PHRASE"),
		headerRow, headerTable,
	}
	byName := map[string]ddicType{}
	for _, t := range types {
		byName[t.name] = t
	}
	return byName
}()

// The layout of a Unicode DFIES row: 1350 bytes, every field a character
// field of the width given, NUMC fields zero-filled. Measured: slicing the
// system's rows at these widths puts every value in its own field.
var dfiesLayout = []struct {
	name  string
	width int
	numc  bool
}{
	{"TABNAME", 30, false}, {"FIELDNAME", 30, false}, {"LANGU", 1, false}, {"POSITION", 4, true}, {"OFFSET", 6, true},
	{"DOMNAME", 30, false}, {"ROLLNAME", 30, false}, {"CHECKTABLE", 30, false}, {"LENG", 6, true}, {"INTLEN", 6, true},
	{"OUTPUTLEN", 6, true}, {"DECIMALS", 6, true}, {"DATATYPE", 4, false}, {"INTTYPE", 1, false}, {"REFTABLE", 30, false},
	{"REFFIELD", 30, false}, {"PRECFIELD", 30, false}, {"AUTHORID", 3, false}, {"MEMORYID", 20, false}, {"LOGFLAG", 1, false},
	{"MASK", 20, false}, {"MASKLEN", 4, true}, {"CONVEXIT", 5, false}, {"HEADLEN", 2, true}, {"SCRLEN1", 2, true},
	{"SCRLEN2", 2, true}, {"SCRLEN3", 2, true}, {"FIELDTEXT", 60, false}, {"REPTEXT", 55, false}, {"SCRTEXT_S", 10, false},
	{"SCRTEXT_M", 20, false}, {"SCRTEXT_L", 40, false}, {"KEYFLAG", 1, false}, {"LOWERCASE", 1, false}, {"MAC", 1, false},
	{"GENKEY", 1, false}, {"NOFORKEY", 1, false}, {"VALEXI", 1, false}, {"NOAUTHCH", 1, false}, {"SIGN", 1, false},
	{"DYNPFLD", 1, false}, {"F4AVAILABL", 1, false}, {"COMPTYPE", 1, false}, {"LFIELDNAME", 132, false}, {"LTRFLDDIS", 1, false},
	{"BIDICTRLC", 1, false}, {"OUTPUTSTYLE", 2, true}, {"NOHISTORY", 1, false}, {"AMPMFORMAT", 1, false},
}

const (
	dfiesRowLength  = 1350
	funintRowLength = classicrfc.RfcFunintUnicodeRowLength
	x030lLength     = 416
)

func flag(b bool) string {
	if b {
		return "X"
	}
	return ""
}

// dfiesValues names the value of every DFIES field for one line.
func dfiesValues(tabname string, f ddicField) map[string]string {
	v := map[string]string{
		"TABNAME": tabname, "FIELDNAME": f.name, "LANGU": f.langu,
		"POSITION": strconv.Itoa(f.position), "OFFSET": strconv.Itoa(f.offset),
		"ROLLNAME": f.rollname, "LENG": strconv.Itoa(f.leng), "INTLEN": strconv.Itoa(f.intlen),
		"DATATYPE": f.datatype, "INTTYPE": f.inttype, "PRECFIELD": f.precfield,
		"FIELDTEXT": f.texts[0], "REPTEXT": f.texts[1], "SCRTEXT_S": f.texts[2], "SCRTEXT_M": f.texts[3], "SCRTEXT_L": f.texts[4],
		"NOAUTHCH": flag(f.noauthch), "DYNPFLD": flag(f.dynpfld), "COMPTYPE": f.comptype, "LFIELDNAME": f.lfieldname,
	}
	if f.headlen != "" {
		v["HEADLEN"], v["SCRLEN1"], v["SCRLEN2"], v["SCRLEN3"] = f.headlen, f.scrlen[0], f.scrlen[1], f.scrlen[2]
	}
	return v
}

// dfiesRow writes one line; an empty map writes the initial row (DFIES_WA).
func dfiesRow(values map[string]string) ([]byte, error) {
	row := make([]byte, 0, dfiesRowLength)
	for _, col := range dfiesLayout {
		text := values[col.name]
		if col.numc {
			if text == "" {
				text = "0"
			}
			text = strings.Repeat("0", col.width-len(text)) + text
		}
		enc, err := classicrfc.EncodeAbapChar(text, col.width)
		if err != nil {
			return nil, fmt.Errorf("rfcserver: DFIES %s: %w", col.name, err)
		}
		row = append(row, enc...)
	}
	if len(row) != dfiesRowLength {
		return nil, fmt.Errorf("rfcserver: DFIES row is %d bytes, not %d", len(row), dfiesRowLength)
	}
	return row, nil
}

// dfiesXML writes one line as the item of a LINES_DESCR FIELDS table: the
// same fields, in the same order, as elements.
func dfiesXML(values map[string]string) string {
	var sb strings.Builder
	sb.WriteString("<item>")
	for _, col := range dfiesLayout {
		text := values[col.name]
		if col.numc {
			if text == "" {
				text = "0"
			}
			text = strings.Repeat("0", col.width-len(text)) + text
		}
		sb.WriteString("<" + col.name + ">" + escapeXML(text) + "</" + col.name + ">")
	}
	sb.WriteString("</item>")
	return sb.String()
}

func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// linesDescr describes the table types a structure uses, each with its line
// type first: what DDIF_FIELDINFO_GET returns as LINES_DESCR.
func linesDescr(t ddicType) string {
	var sb strings.Builder
	sb.WriteString("<LINES_DESCR>")
	for _, name := range t.lineOf {
		lt := adtDictionary[name]
		kind := "STRU"
		if lt.kind == "TTYP" {
			kind = "TTYP"
		}
		sb.WriteString("<item><TYPENAME>" + lt.name + "</TYPENAME><TYPEKIND>" + kind + "</TYPEKIND><FIELDS>")
		for _, f := range lt.fields {
			v := dfiesValues(lt.name, f)
			if lt.authorid != "" {
				v["AUTHORID"] = lt.authorid
			}
			sb.WriteString(dfiesXML(v))
		}
		sb.WriteString("</FIELDS></item>")
	}
	sb.WriteString("</LINES_DESCR>")
	return sb.String()
}

// x030l writes the nametab header of a type: 416 bytes. The fields a client
// reads are the field count and length at 162 and 164 (big-endian), the table
// kind at 172, the line type at 176 and the character width at 248; the rest
// is written as the system writes it, so that a reader that looks further
// finds what it expects. The identity and timestamps name nothing real.
func x030l(t ddicType, uclen byte) ([]byte, error) {
	out := make([]byte, 0, x030lLength)
	put := func(text string, width int) error {
		enc, err := classicrfc.EncodeAbapChar(text, width)
		if err != nil {
			return err
		}
		out = append(out, enc...)
		return nil
	}
	if err := put(t.name, 30); err != nil {
		return nil, err
	}
	_ = put("C", 1)
	id := sha256.Sum256([]byte("open-rfc-go nametab " + t.name))
	out = append(out, id[:16]...)
	_ = put(strings.Repeat("20200101000000", 3), 42)
	// a table type reports its line type's field count, as the system does
	fieldCount := len(t.fields)
	if t.kind == "TTYP" {
		fieldCount = len(adtDictionary[t.rowtype].fields)
	}
	var ints [10]byte
	binary.BigEndian.PutUint16(ints[0:], uint16(fieldCount))
	binary.BigEndian.PutUint32(ints[2:], uint32(t.length))
	tail := []byte{0x10, 0x00, 0x03, 0x00, 0x07, 0x00, 0x20, 0x00, 0x00, 0x03, 0x08, 0x00, uclen, 0, 0, 0, 0, 0}
	kind := "J"
	if t.kind == "TTYP" {
		binary.BigEndian.PutUint16(ints[6:], 0x0400)
		binary.BigEndian.PutUint16(ints[8:], 0x0300)
		tail = []byte{0, 0, 0, 0, 0, 0, 0x53, 0, 0, 0, 0, 0, uclen, 0, 0, 0, 0, 0}
		kind = "L"
	}
	out = append(out, ints[:]...)
	_ = put(kind, 1)
	_ = put("J", 1)
	_ = put(t.rowtype, 30)
	out = append(out, tail...)
	_ = put(strings.Repeat("0", 17), 17)
	out = append(out, make([]byte, x030lLength-len(out))...)
	return out, nil
}

// funintRow writes one RFC_FUNINT row: the layout classicrfc.DecodeFunintRow
// reads, integers little-endian as the system writes them.
func funintRow(p classicrfc.FunintParameter) ([]byte, error) {
	out := make([]byte, 0, funintRowLength)
	put := func(text string, width int) error {
		enc, err := classicrfc.EncodeAbapChar(text, width)
		if err != nil {
			return err
		}
		out = append(out, enc...)
		return nil
	}
	for _, s := range []struct {
		v string
		w int
	}{{p.ParameterClass, 1}, {p.ParameterName, 30}, {p.TableName, 30}, {p.FieldName, 30}, {p.Exid, 1}} {
		if err := put(s.v, s.w); err != nil {
			return nil, err
		}
	}
	var ints [16]byte
	binary.LittleEndian.PutUint32(ints[0:], uint32(p.Position))
	binary.LittleEndian.PutUint32(ints[4:], uint32(p.Offset))
	binary.LittleEndian.PutUint32(ints[8:], uint32(p.InternalLength))
	binary.LittleEndian.PutUint32(ints[12:], uint32(p.Decimals))
	out = append(out, ints[:]...)
	for _, s := range []struct {
		v string
		w int
	}{{p.DefaultValue, 21}, {p.ParameterText, 79}, {flag(p.Optional), 1}} {
		if err := put(s.v, s.w); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// answerTables fills the tables the caller sent: rows for the one named, none
// for the others, every one answered by its number.
//
// A caller need not send a table to receive one. Eclipse declares each table
// parameter in the call, numbered, and is answered by number; an RFC client
// names them only among the outputs it wants and is answered by name. So any
// requested output that is a table of this function and did not arrive as one
// is added here, and the encoder frames it the classic way because it has no
// number to be answered by.
func answerTables(req Request, name string, rowLength int, rows [][]byte, alsoEmpty ...string) []Table {
	var out []Table
	seen := map[string]bool{}
	for _, t := range req.Tables {
		answer := Table{Name: t.Name, ID: t.ID, RowByteLength: t.RowByteLength}
		if t.Name == name {
			answer.RowByteLength = rowLength
			answer.Rows = rows
		}
		seen[t.Name] = true
		out = append(out, answer)
	}
	for _, want := range append([]string{name}, alsoEmpty...) {
		if seen[want] || !wants(req, want) {
			continue
		}
		answer := Table{Name: want}
		if want == name {
			answer.RowByteLength, answer.Rows = rowLength, rows
		}
		out = append(out, answer)
	}
	return out
}

func wants(req Request, name string) bool {
	for _, o := range req.RequestedOutputs {
		if o == name {
			return true
		}
	}
	return false
}

func charExport(name, value string, width int) (cpic.NamedValue, error) {
	enc, err := classicrfc.EncodeAbapChar(value, width)
	return cpic.NamedValue{Name: name, Value: enc}, err
}

// FunctionInterfaceHandler answers RFC_GET_FUNCTION_INTERFACE for the one
// function this bridge serves.
func FunctionInterfaceHandler() Handler {
	return func(ctx context.Context, req Request) (Response, error) {
		name, _ := req.ImportText("FUNCNAME")
		if name != adtRestFunction {
			return Response{}, &Exception{Key: UnknownFunctionKey}
		}
		var resp Response
		for _, o := range []struct{ name, value string }{
			{"REMOTE_BASXML_SUPPORTED", "X"}, {"REMOTE_CALL", "R"}, {"UPDATE_TASK", ""},
		} {
			if wants(req, o.name) {
				e, err := charExport(o.name, o.value, 1)
				if err != nil {
					return Response{}, err
				}
				resp.Outputs = append(resp.Outputs, Output{Name: e.Name, Value: e.Value})
			}
		}
		params := []classicrfc.FunintParameter{
			{ParameterClass: "E", ParameterName: "RESPONSE", TableName: adtRestResponse, Exid: "v", InternalLength: 40, ParameterText: "ADT response"},
			{ParameterClass: "I", ParameterName: "REQUEST", TableName: adtRestRequest, Exid: "v", InternalLength: 40, ParameterText: "ADT request"},
		}
		var rows [][]byte
		for _, p := range params {
			row, err := funintRow(p)
			if err != nil {
				return Response{}, err
			}
			rows = append(rows, row)
		}
		resp.Tables = answerTables(req, "PARAMS", funintRowLength, rows, "RESUMABLE_EXCEPTIONS")
		return resp, nil
	}
}

// FieldInfoHandler answers DDIF_FIELDINFO_GET for the types of that function.
func FieldInfoHandler() Handler {
	return func(ctx context.Context, req Request) (Response, error) {
		name, _ := req.ImportText("TABNAME")
		t, ok := adtDictionary[name]
		if !ok {
			return Response{}, &Exception{Key: "NOT_FOUND"}
		}
		uclen := req.ImportByte("UCLEN")
		if uclen == 0 {
			uclen = 2
		}
		// in the order the system returns them, which is the parameter order
		// of DDIF_FIELDINFO_GET and not the order the client asked: DDOBJTYPE,
		// then the initial DFIES row, then the nested LINES_DESCR, then the
		// nametab header.
		var resp Response
		if wants(req, "DDOBJTYPE") {
			e, err := charExport("DDOBJTYPE", t.kind, 8)
			if err != nil {
				return Response{}, err
			}
			resp.Outputs = append(resp.Outputs, Output{Name: e.Name, Value: e.Value})
		}
		if wants(req, "DFIES_WA") {
			row, err := dfiesRow(nil)
			if err != nil {
				return Response{}, err
			}
			resp.Outputs = append(resp.Outputs, Output{Name: "DFIES_WA", Value: row})
		}
		if wants(req, "LINES_DESCR") {
			resp.Outputs = append(resp.Outputs, Output{Name: "LINES_DESCR", XML: true, Value: []byte(linesDescr(t))})
		}
		if wants(req, "X030L_WA") {
			wa, err := x030l(t, uclen)
			if err != nil {
				return Response{}, err
			}
			resp.Outputs = append(resp.Outputs, Output{Name: "X030L_WA", Value: wa})
		}
		var rows [][]byte
		for _, f := range t.fields {
			row, err := dfiesRow(dfiesValues(t.name, f))
			if err != nil {
				return Response{}, err
			}
			rows = append(rows, row)
		}
		resp.Tables = answerTables(req, "DFIES_TAB", dfiesRowLength, rows)
		return resp, nil
	}
}

// RFC_FIELDS geometry: the row a client reads a structure's fields from.
const (
	rfcFieldsRowLength = 138
	// a reference field — a substructure or a table — occupies eight bytes in
	// the flat layout, which is what the DDIC rows above already say
	adtStructureByteLength = 40
)

// fieldsRow writes one RFC_FIELDS row: two thirty-character names, four
// little-endian integers and the internal type.
func fieldsRow(tabname string, f ddicField) ([]byte, error) {
	out := make([]byte, 0, rfcFieldsRowLength)
	for _, s := range []struct {
		v string
		w int
	}{{tabname, 30}, {f.name, 30}} {
		enc, err := classicrfc.EncodeAbapChar(s.v, s.w)
		if err != nil {
			return nil, err
		}
		out = append(out, enc...)
	}
	var ints [16]byte
	binary.LittleEndian.PutUint32(ints[0:], uint32(f.position))
	binary.LittleEndian.PutUint32(ints[4:], uint32(f.offset))
	binary.LittleEndian.PutUint32(ints[8:], uint32(f.intlen))
	binary.LittleEndian.PutUint32(ints[12:], 0) // decimals
	out = append(out, ints[:]...)
	exid, err := classicrfc.EncodeAbapChar(f.inttype, 1)
	if err != nil {
		return nil, err
	}
	out = append(out, exid...)
	if len(out) != rfcFieldsRowLength {
		return nil, fmt.Errorf("rfcserver: RFC_FIELDS row is %d bytes, not %d", len(out), rfcFieldsRowLength)
	}
	return out, nil
}

// StructureDefinitionHandler answers RFC_GET_STRUCTURE_DEFINITION.
//
// Eclipse learns a structure through DDIF_FIELDINFO_GET; an RFC client asks
// this instead (after trying RFC_METADATA_GET, which this bridge does not
// pretend to have). Same dictionary, a flatter answer: the structure's byte
// length and one row per field.
func StructureDefinitionHandler() Handler {
	return func(ctx context.Context, req Request) (Response, error) {
		name, _ := req.ImportText("TABNAME")
		t, ok := adtDictionary[name]
		if !ok {
			return Response{}, &Exception{Key: "NOT_FOUND"}
		}
		fields := t.fields
		if t.kind == "TTYP" {
			// a table type is described by its line type
			fields = adtDictionary[t.rowtype].fields
		}
		// RFC_FIELDS is a FLAT view: a component of a substructure has an
		// offset inside that substructure, and listing it beside its parent
		// makes the two overlap, which a client rejects. Only the fields this
		// structure owns directly are listed; a client that wants what is
		// inside REQUEST_LINE asks for REQUEST_LINE.
		var top []ddicField
		for _, f := range fields {
			if f.precfield == "" || f.precfield == t.name {
				top = append(top, f)
			}
		}
		var rows [][]byte
		for i, f := range top {
			f.position = i + 1
			row, err := fieldsRow(name, f)
			if err != nil {
				return Response{}, err
			}
			rows = append(rows, row)
		}
		var resp Response
		if wants(req, "TABLENGTH") {
			length := make([]byte, 4)
			binary.LittleEndian.PutUint32(length, uint32(t.length))
			resp.Outputs = append(resp.Outputs, Output{Name: "TABLENGTH", Value: length})
		}
		resp.Tables = answerTables(req, "FIELDS", rfcFieldsRowLength, rows)
		return resp, nil
	}
}
