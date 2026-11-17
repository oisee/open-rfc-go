package cpic

import "testing"

// The first RFCPING answer of an ABAP kernel 9.16 carries two fields that a 7.54 kernel does
// not send: 0x0454 (36 bytes) after 0x0451, and 0x0527 (4 bytes) before the response start.
// Field order and lengths follow the answers of two kernel 9.16 systems (SAP_BASIS 757 and
// 758). The values are zeros.
func kernel916Fields(t *testing.T) []Field {
	return []Field{
		fld(TagProtocolVersion, hx(t, "00000e0b")), fld(TagCapabilities, make([]byte, 11)), fld(TagLogonStatus, []byte{0}),
		fld(TagSystemCodePage, make([]byte, 8)), fld(Tag(0x0450), make([]byte, 6)), fld(Tag(0x0451), make([]byte, 20)),
		fld(Tag(0x0454), make([]byte, 36)), fld(Tag(0x0452), make([]byte, 4)), fld(Tag(0x0453), make([]byte, 36)),
		fld(TagClientAddress, make([]byte, 30)), fld(Tag(0x0020), make([]byte, 92)), fld(Tag(0x0021), make([]byte, 20)),
		fld(TagPartnerSystem, make([]byte, 26)), fld(TagPartnerHost, make([]byte, 26)), fld(TagConnectionType, make([]byte, 2)),
		fld(TagKernelPatch, make([]byte, 8)), fld(TagKernelRelease, make([]byte, 8)), fld(TagProgram, make([]byte, 16)),
		fld(Tag(0x0150), make([]byte, 24)), fld(Tag(0x0151), make([]byte, 6)), fld(Tag(0x0152), make([]byte, 2)),
		fld(Tag(0x0527), make([]byte, 4)),
		fld(TagResponseStart, nil), fld(TagResponseContext, nil), fld(TagSession, make([]byte, 16)),
		fld(TagUnresolved0420, make([]byte, 4)), fld(TagCallContext, nil), fld(TagProgram, make([]byte, 80)),
		fld(Tag(0x0667), make([]byte, 8)), fld(TagEnd, nil),
	}
}

func TestAdmitsKernel916RichComposite(t *testing.T) {
	fields := kernel916Fields(t)
	assertRichSuccess(t, "kernel 9.16", fields)
	assertRichSuccess(t, "kernel 9.16 without 0x0527", filterOutTag(fields, 0x0527))
	assertRichSuccess(t, "kernel 9.16 without 0x0454", filterOutTag(fields, 0x0454))
	assertRichReject(t, "kernel 9.16 short 0x0454", replaceIdx(fields, 6, fld(Tag(0x0454), make([]byte, 34))), "composite shape")
	assertRichReject(t, "kernel 9.16 short 0x0527", replaceIdx(fields, 21, fld(Tag(0x0527), make([]byte, 3))), "composite shape")
	assertRichReject(t, "kernel 9.16 duplicate 0x0527", insertAt(fields, 21, fld(Tag(0x0527), make([]byte, 4))), "duplicate")
}

// A kernel 9.16 function response starts with 0527 0004 <4 bytes> 0527 before 05000000.
func TestDecodesKernel916FunctionResponseLead(t *testing.T) {
	body := concat(hx(t, "05000000"), mustChain(t, TagResponseStart, []Field{
		fld(TagResponseContext, nil), fld(TagSession, bytesRep(1, 16)), fld(TagUnresolved0420, make([]byte, 4)),
		fld(TagCallContext, nil), fld(TagEnd, nil),
	}), []byte{0xff, 0xff})
	plain, err := DecodeFunctionResultFields(body)
	if err != nil || !plain.Success {
		t.Fatalf("plain = %+v, %v", plain, err)
	}
	led, err := DecodeFunctionResultFields(concat(hx(t, "0527000400000000"+"0527"), body))
	if err != nil || !led.Success || len(led.Fields) != len(plain.Fields) {
		t.Fatalf("with lead = %+v, %v", led, err)
	}
	if _, err := DecodeFunctionResultFields(concat(hx(t, "0527000300000005"+"27"), body)); err == nil {
		t.Fatal("a lead field of another length must stay a protocol error")
	}
	if _, err := DecodeFunctionResultFields(concat(hx(t, "0528000400000000"+"0528"), body)); err == nil {
		t.Fatal("another lead tag must stay a protocol error")
	}
}
