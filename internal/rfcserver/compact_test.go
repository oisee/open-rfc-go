// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oisee/open-rfc-go/internal/appc"
	"github.com/oisee/open-rfc-go/internal/bxml"
	"github.com/oisee/open-rfc-go/internal/classicrfc"
	"github.com/oisee/open-rfc-go/internal/cpic"
)

func discoveryRequest() *bxml.Element {
	text := func(name, value string) *bxml.Element { return &bxml.Element{Name: name, Text: value, HasText: true} }
	return &bxml.Element{Name: "REQUEST", Children: []*bxml.Element{
		{Name: "REQUEST_LINE", Children: []*bxml.Element{text("METHOD", "POST"), text("URI", "/sap/bc/adt/checkruns?reporters=abapCheckRun"), text("VERSION", "HTTP/1.1")}},
		{Name: "HEADER_FIELDS", Attrs: []bxml.Attr{{Name: "lines", Value: "2"}}, Children: []*bxml.Element{
			{Name: "item", Children: []*bxml.Element{text("NAME", "Accept"), text("VALUE", "application/vnd.sap.adt.checkmessages+xml")}},
			{Name: "item", Children: []*bxml.Element{text("NAME", "Connection"), text("VALUE", "keep-alive")}},
		}},
		{Name: "MESSAGE_BODY", Body: "<chkrun:checkObjectList/>", HasBody: true},
	}}
}

// A call the way Eclipse frames it, with the parameter in the 0x4000 family.
func TestDecodeEclipseCompactCall(t *testing.T) {
	doc, err := bxml.Encode(discoveryRequest())
	if err != nil {
		t.Fatal(err)
	}
	guid := bytes.Repeat([]byte{0x5a}, 16)
	for _, compressed := range []bool{false, true} {
		payload := eclipseCall(t, eclipseCallInput{
			function: adtRestFunction, guid: guid, outputs: []string{"RESPONSE"}, compact: doc, compressed: compressed,
		})
		req, err := DecodeFunctionRequest(payload)
		if err != nil {
			t.Fatalf("compressed=%v: %v", compressed, err)
		}
		if !req.Eclipse || req.Order != binary.BigEndian || !bytes.Equal(req.SessionGUID, guid) {
			t.Fatalf("framing not recognised: eclipse=%v order=%v guid=%x", req.Eclipse, req.Order, req.SessionGUID)
		}
		if req.FunctionName != adtRestFunction || len(req.RequestedOutputs) != 1 || req.RequestedOutputs[0] != "RESPONSE" {
			t.Fatalf("%q wants %v", req.FunctionName, req.RequestedOutputs)
		}
		if !bytes.Equal(req.Compact, doc) {
			t.Fatalf("compressed=%v: the compact parameter did not survive (%d bytes, want %d)", compressed, len(req.Compact), len(doc))
		}
	}
}

// One HTTP exchange through the compact framing: Eclipse's request reaches the
// backend as it was sent, and the backend's answer comes back as the document
// the system would have written, compressed the way the system compresses it.
func TestADTHandlerCompactExchange(t *testing.T) {
	var seenMethod, seenPath, seenAccept, seenBody, seenConnection string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod, seenPath = r.Method, r.URL.RequestURI()
		seenAccept, seenConnection = r.Header.Get("Accept"), r.Header.Get("Connection")
		read, _ := io.ReadAll(r.Body)
		seenBody = string(read)
		w.Header().Set("Content-Type", "application/vnd.sap.adt.checkmessages+xml")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("<chkrun:checkRunReports/>"))
	}))
	defer backend.Close()
	handler, err := ADTRestHandler(Backend{URL: backend.URL}, backend.Client())
	if err != nil {
		t.Fatal(err)
	}
	doc, err := bxml.Encode(discoveryRequest())
	if err != nil {
		t.Fatal(err)
	}
	req, err := DecodeFunctionRequest(eclipseCall(t, eclipseCallInput{function: adtRestFunction, outputs: []string{"RESPONSE"}, compact: doc, guid: bytes.Repeat([]byte{1}, 16)}))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if seenMethod != "POST" || seenPath != "/sap/bc/adt/checkruns?reporters=abapCheckRun" || seenAccept != "application/vnd.sap.adt.checkmessages+xml" || seenBody != "<chkrun:checkObjectList/>" {
		t.Fatalf("backend saw %s %s accept=%q body=%q", seenMethod, seenPath, seenAccept, seenBody)
	}
	if seenConnection != "" {
		t.Fatalf("a hop-by-hop header reached the backend")
	}
	if resp.Compact == nil || resp.CompactName != "RESPONSE" {
		t.Fatalf("the answer is not a compact RESPONSE: %+v", resp)
	}
	tree, err := bxml.Decode(resp.Compact)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := bxml.PayloadRoot(tree)
	line := root.Child("STATUS_LINE")
	if line.ChildText("STATUS_CODE") != "201 " || line.ChildText("REASON_PHRASE") != "Created" || line.ChildText("VERSION") != "HTTP/1.1" {
		t.Fatalf("status line: %q %q %q", line.ChildText("VERSION"), line.ChildText("STATUS_CODE"), line.ChildText("REASON_PHRASE"))
	}
	headers := root.Child("HEADER_FIELDS")
	if headers.Children[0].Name != adtHeaderRow || headers.Children[0].ChildText("NAME") != "~server_protocol" {
		t.Fatalf("the first header row is %s %s", headers.Children[0].Name, headers.Children[0].ChildText("NAME"))
	}
	if headers.Attrs[0].Value != itoa(len(headers.Children)) {
		t.Fatalf("lines=%s for %d rows", headers.Attrs[0].Value, len(headers.Children))
	}
	if body := root.Child("MESSAGE_BODY"); !body.HasBody || body.Body != "<chkrun:checkRunReports/>" {
		t.Fatalf("body: %+v", body)
	}
	// on the wire: 0500 0503 0514 0420 0512 0205 4000 4002 4004 0130 0667 0523 ffff
	cut, err := EncodeEclipseResponse(resp, req)
	if err != nil {
		t.Fatal(err)
	}
	fields := responseFields(t, cut)
	want := []uint16{0x0503, 0x0514, 0x0420, 0x0512, 0x0205, tagCompactFlag, tagCompactResponse, tagCompactEnd, 0x0130, 0x0667, tagCompactClose, 0xffff}
	if got := tagsOf(fields); !sameTags(got, want) {
		t.Fatalf("response tags\n got %04x\nwant %04x", got, want)
	}
	if flag := valuesOf(fields, tagCompactFlag)[0]; !bytes.Equal(flag, []byte{1, 1}) {
		t.Fatalf("flag %x", flag)
	}
	var deflated []byte
	for _, c := range valuesOf(fields, tagCompactResponse) {
		deflated = append(deflated, c...)
	}
	inflated, err := inflateBounded(deflated, maxCompactBytes)
	if err != nil || !bytes.Equal(inflated, resp.Compact) {
		t.Fatalf("the compact export does not inflate to the document: %v", err)
	}
}

func itoa(n int) string { return strings.TrimSpace(strings.Repeat(" ", 0) + string(rune('0'+n))) }

// An empty answer body is an empty element, as the system writes it.
func TestResponseElementEmptyBody(t *testing.T) {
	answer := &http.Response{Proto: "HTTP/1.1", Status: "304 Not Modified", StatusCode: 304, Header: http.Header{}}
	el := responseElement(answer, nil)
	body := el.Child("MESSAGE_BODY")
	if body.HasBody || body.HasText {
		t.Fatalf("empty body written as %+v", body)
	}
	if el.Child("STATUS_LINE").ChildText("STATUS_CODE") != "304 " {
		t.Fatalf("status code %q", el.Child("STATUS_LINE").ChildText("STATUS_CODE"))
	}
}

// An answer longer than a record is sent as the system sends one.
func TestEclipseRecordsSplitLongAnswers(t *testing.T) {
	conv := []byte("00000042")
	for _, size := range []int{70_000, 2*maxRecordData + 3, maxRecordData, maxRecordData + 1} {
		cut := bytes.Repeat([]byte{0xee}, size)
		records, err := eclipseRecords(cut, conv, 7, []byte{0, 0, 0, 0})
		if err != nil {
			t.Fatal(err)
		}
		var joined []byte
		for i, rec := range records {
			if len(rec) < 80 || rec[0] != 0x06 {
				t.Fatalf("size %d: record %d is not an APPC record", size, i)
			}
			data := rec[80:]
			final := i == len(records)-1
			switch {
			case len(records) == 1:
				if rec[1] != 0xcb || rec[30] != recordInfoFinal {
					t.Fatalf("a single record must be a final F_SAP_SEND: %x %x", rec[1], rec[30])
				}
			case i == 0:
				// full, unless the last record had to be topped up to eight bytes
				if rec[1] != 0xcb || rec[30] != 0x01 || rec[31] != 0x08 || len(data) < maxRecordData-7 || len(data) > maxRecordData {
					t.Fatalf("size %d: first record fn %x info %x vector %x data %d", size, rec[1], rec[30], rec[31], len(data))
				}
			case final:
				if rec[1] != 0x09 || rec[30] != recordInfoFinal || rec[31] != 0x0c || len(data) < 8 {
					t.Fatalf("size %d: last record fn %x info %x vector %x data %d", size, rec[1], rec[30], rec[31], len(data))
				}
			default:
				if rec[1] != 0x09 || rec[30] == recordInfoFinal {
					t.Fatalf("size %d: middle record fn %x info %x", size, rec[1], rec[30])
				}
			}
			if !bytes.Equal(rec[40:48], conv) {
				t.Fatalf("record %d carries conversation %q", i, rec[40:48])
			}
			joined = append(joined, data...)
		}
		if !bytes.Equal(joined, cut) {
			t.Fatalf("size %d: the records do not join back into the answer", size)
		}
	}
}

// A call that arrives in two records is answered once, from the whole of it.
func TestServeConsciousJoinsContinuedRecords(t *testing.T) {
	text, err := classicrfc.EncodeAbapChar("split across two records", 255)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := cpic.EncodeCutFunctionRequest(cpic.CutFunctionRequestInput{
		FunctionName: "STFC_CONNECTION",
		Imports:      []cpic.NamedValue{{Name: "REQUTEXT", Value: text}},
	})
	if err != nil {
		t.Fatal(err)
	}
	half := len(payload) / 2
	conv := []byte("00000001")
	notFinal, final := false, true
	sapSend, receive := appc.FuncSapSend, appc.FuncReceive
	first, err := appc.EncodeDataRecord(appc.DataRecordInput{RecordHeaderInput: appc.RecordHeaderInput{ConversationID: conv}, FunctionCode: &sapSend, Data: payload[:half], IsFinal: &notFinal})
	if err != nil {
		t.Fatal(err)
	}
	second, err := appc.EncodeDataRecord(appc.DataRecordInput{RecordHeaderInput: appc.RecordHeaderInput{ConversationID: conv}, FunctionCode: &receive, Data: payload[half:], IsFinal: &final})
	if err != nil {
		t.Fatal(err)
	}

	client, server := net.Pipe()
	defer client.Close()
	go func() {
		defer server.Close()
		ServeConscious(server, DefaultDispatcher(), func(string) {}, nil)
	}()
	_ = client.SetDeadline(time.Now().Add(5 * time.Second))
	for _, rec := range [][]byte{first, second} {
		if _, err := client.Write(niFrame(rec)); err != nil {
			t.Fatal(err)
		}
	}
	var length [4]byte
	if _, err := io.ReadFull(client, length[:]); err != nil {
		t.Fatalf("no answer: %v", err)
	}
	answer := make([]byte, binary.BigEndian.Uint32(length[:]))
	if _, err := io.ReadFull(client, answer); err != nil {
		t.Fatal(err)
	}
	env, err := cpic.DecodeFunctionResultFields(answer[80:])
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if !env.Success {
		t.Fatalf("the joined call was not answered with success")
	}
	res, err := classicrfc.DecodeResult(env.Fields)
	if err != nil {
		t.Fatal(err)
	}
	var echo string
	for _, s := range res.Scalars {
		if s.Name == "ECHOTEXT" {
			echo, _ = classicrfc.DecodeAbapChar(s.Value)
		}
	}
	if echo != "split across two records" {
		t.Fatalf("ECHOTEXT = %q", echo)
	}
}

// The single-record header, masked to what the gateway holds constant, is the
// header the system sends. The bytes below are a captured response header with
// the uid, conversation, and message length blanked — routing only, no
// identifiers — and are the shape Eclipse receives without CMRCV.
func TestSingleRecordMatchesGatewayHeader(t *testing.T) {
	// 06cb0200 <uid> 0006 0000 0000 00010000 00 000001f4 02 00000001 0008 0000 05 0c
	// 00000000 00000000 <conv> <len> 00..00 00060002
	data := bytes.Repeat([]byte{0xa5}, 700)
	rec := buildEclipseRecord(data, []byte("00000001"), 0x1234, appcFSapSend, true, len(data), []byte{0, 0, 0, 2})
	want := mustDecode(t, "06cb0200"+"1234"+"0007"+"0000"+"0000"+"00010000"+"01"+"ffffffff"+"02"+"00000001"+"0008"+"0000"+"05"+"0c"+"00000000"+"00000000"+"3030303030303031"+
		"00006d60"+"00000002"+hexU32(uint32(len(data)))+"00000001"+"00000000"+"0034313033"+"000000"+"00000002")
	if !bytes.Equal(rec[:80], want) {
		t.Fatalf("record header differs\n got %x\nwant %x", rec[:80], want)
	}
	if len(rec) != 80+len(data) {
		t.Fatalf("record is %d bytes", len(rec))
	}
}

func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func hexU32(n uint32) string {
	return fmt.Sprintf("%08x", n)
}

// A reply carries a zero communication index and the connection index the call
// came in on. A caller may verify both; one that does hangs up on a reply that
// hands back the 0xffff "unset" the request was sent with.
func TestResponseIndicesAreAnsweredNotEchoed(t *testing.T) {
	cut := bytes.Repeat([]byte{0x11}, 200)
	conv := []byte("00000007")
	cases := []struct{ caller, want []byte }{
		{[]byte{0xff, 0xff, 0, 0}, []byte{0, 0, 0, 0}}, // the usual: unset in, zero out
		{[]byte{0xff, 0xff, 0, 3}, []byte{0, 0, 0, 3}}, // the connection index survives
		{[]byte{0, 0, 0, 0}, []byte{0, 0, 0, 0}},
	}
	for _, c := range cases {
		records, err := responseRecords(cut, conv, 5, c.caller)
		if err != nil {
			t.Fatal(err)
		}
		rec := records[0]
		if got := binary.BigEndian.Uint32(rec[48:]); got != 0x00006d60 {
			t.Fatalf("not the gateway block: %08x", got)
		}
		if !bytes.Equal(rec[76:80], c.want) {
			t.Fatalf("caller %x answered with %x, want %x", c.caller, rec[76:80], c.want)
		}
		if !bytes.Equal(rec[40:48], conv) {
			t.Fatalf("conversation id lost: %q", rec[40:48])
		}
	}
}
