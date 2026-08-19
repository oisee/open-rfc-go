// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/rfc-error-envelope.test.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten for the testing package. The
// `{tag: -1}` INVALID_FIELD case has no Go analogue (Tag is uint16); the
// Object.isFrozen / Object.hasOwn assertions are dropped (Go structs are not
// runtime-mutable and carry no dynamic property). A FuzzDecode target is added.
// See docs/provenance.md.

package rfcerr

import (
	"encoding/binary"
	"reflect"
	"testing"
	"unicode/utf16"
)

func p(v int) *int { return &v }

func utf16le(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(out[i*2:], u)
	}
	return out
}

func textField(tag Tag, value string, rightPadding int) Field {
	for i := 0; i < rightPadding; i++ {
		value += " "
	}
	return Field{Tag: uint16(tag), Value: utf16le(value)}
}

func rawField(tag Tag, value []byte) Field {
	return Field{Tag: uint16(tag), Value: append([]byte(nil), value...)}
}
func rawTag(tag uint16, value []byte) Field {
	return Field{Tag: tag, Value: append([]byte(nil), value...)}
}
func endField() Field            { return Field{Tag: EndTag, Value: nil} }
func endFieldVal(v []byte) Field { return Field{Tag: EndTag, Value: v} }

func mustDecode(t *testing.T, fields []Field) Envelope {
	t.Helper()
	env, err := Decode(fields, DecodeOptions{})
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	return env
}

func expectProtocolError(t *testing.T, fields []Field, opts DecodeOptions, reasonCode string) {
	t.Helper()
	_, err := Decode(fields, opts)
	pe, ok := err.(*ProtocolError)
	if !ok || pe.ReasonCode != reasonCode {
		t.Fatalf("want reason %s, got %v", reasonCode, err)
	}
}

func semanticFacts(t *testing.T, fields []Field) RemoteErrorFacts {
	t.Helper()
	f := mustDecode(t, fields).Facts
	f.Provenance = nil
	return f
}

func TestNormalizesDeclaredExceptionFacts(t *testing.T) {
	fields := []Field{
		textField(TagMessageClass, "SR", 0),
		textField(TagMessageType, "E", 0),
		textField(TagMessageNumber, "006", 0),
		textField(TagMessageV1, "Method = 1", 5),
		textField(TagMessageV2, "second", 0),
		textField(TagMessageV3, "third", 0),
		textField(TagMessageV4, "fourth \U0001F642", 0),
		textField(TagT100Text, "Template &1", 0),
		textField(TagErrorMessage, "Rendered message", 0),
		textField(TagCallStack, "stack line", 0),
		textField(TagExceptionKey, "RAISE_EXCEPTION", 0),
		endField(),
	}
	decoded := mustDecode(t, fields)
	if decoded.Outcome != OutcomeAbapException || decoded.SuccessControl != SuccessControlNotApplicable {
		t.Fatalf("outcome = %s / %s", decoded.Outcome, decoded.SuccessControl)
	}
	var wantProv []FactProvenance
	for ordinal, f := range fields[:len(fields)-1] {
		wantProv = append(wantProv, FactProvenance{Tag: f.Tag, Ordinal: ordinal, ByteLength: len(f.Value)})
	}
	want := RemoteErrorFacts{
		ExceptionKey: "RAISE_EXCEPTION", PlainText: "Rendered message", RuntimeID: "",
		T100Text: "Template &1", MessageClass: "SR", MessageType: "E", MessageNumber: "006",
		MessageV1: "Method = 1", MessageV2: "second", MessageV3: "third", MessageV4: "fourth \U0001F642",
		CallStack: "stack line", Provenance: wantProv, Unresolved0420: nil,
	}
	if !reflect.DeepEqual(decoded.Facts, want) {
		t.Fatalf("facts = %+v", decoded.Facts)
	}
}

func TestClassifiesRuntimeAndMessageIndependentOfOrder(t *testing.T) {
	runtime := mustDecode(t, []Field{
		textField(TagMessageV2, "detail 2", 0),
		textField(TagErrorMessage, "Runtime text", 0),
		textField(TagCallStack, "call stack", 0),
		textField(TagRuntimeID, "RUNTIME_ID", 0),
		textField(TagMessageClass, "00", 0),
		endField(),
	})
	if runtime.Outcome != OutcomeAbapRuntime || runtime.Facts.RuntimeID != "RUNTIME_ID" || runtime.Facts.PlainText != "Runtime text" || runtime.Facts.MessageV2 != "detail 2" {
		t.Fatalf("runtime = %+v", runtime)
	}
	message := mustDecode(t, []Field{
		textField(TagMessageV4, "detail 4", 0),
		textField(TagMessageNumber, "123", 0),
		textField(TagMessageType, "A", 0),
		textField(TagMessageClass, "ZZ", 0),
		textField(TagMessageV1, "detail 1", 0),
		textField(TagT100Text, "Message &1", 0),
		endField(),
	})
	if message.Outcome != OutcomeAbapMessage || message.Facts.MessageV1 != "detail 1" || message.Facts.MessageV4 != "detail 4" {
		t.Fatalf("message = %+v", message)
	}
}

func TestRequiresTextOrCoherentIdentity(t *testing.T) {
	for _, tag := range []Tag{TagErrorMessage, TagT100Text} {
		if mustDecode(t, []Field{textField(tag, "message", 0), endField()}).Outcome != OutcomeAbapMessage {
			t.Fatalf("tag %#x not abapMessage", tag)
		}
	}
	identity := []Field{
		textField(TagMessageClass, "ZZ", 0),
		textField(TagMessageType, "E", 0),
		textField(TagMessageNumber, "123", 0),
	}
	if mustDecode(t, append(append([]Field(nil), identity...), endField())).Outcome != OutcomeAbapMessage {
		t.Fatal("full identity not abapMessage")
	}
	for mask := 1; mask < 0b111; mask++ {
		var partial []Field
		for i, f := range identity {
			if mask&(1<<i) != 0 {
				partial = append(partial, f)
			}
		}
		expectProtocolError(t, append(partial, endField()), DecodeOptions{}, "RFC_ERROR_ENVELOPE_AMBIGUOUS_FACTS")
	}
	for emptyIndex := range identity {
		incomplete := make([]Field, len(identity))
		copy(incomplete, identity)
		incomplete[emptyIndex] = textField(Tag(identity[emptyIndex].Tag), "", 2)
		expectProtocolError(t, append(incomplete, endField()), DecodeOptions{}, "RFC_ERROR_ENVELOPE_AMBIGUOUS_FACTS")
	}
	for _, tag := range []Tag{TagErrorMessage, TagT100Text} {
		expectProtocolError(t, []Field{textField(tag, "", 2), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_AMBIGUOUS_FACTS")
	}
}

func TestIdenticalSemanticFactsForPermutations(t *testing.T) {
	source := []Field{
		textField(TagExceptionKey, "DECLARED", 0),
		textField(TagErrorMessage, "plain", 0),
		textField(TagT100Text, "template", 0),
		textField(TagMessageClass, "AA", 0),
		textField(TagMessageType, "E", 0),
		textField(TagMessageNumber, "001", 0),
		textField(TagMessageV1, "one", 0),
		textField(TagMessageV2, "two", 0),
		textField(TagMessageV3, "three", 0),
		textField(TagMessageV4, "four", 0),
	}
	expected := semanticFacts(t, append(append([]Field(nil), source...), endField()))
	for shift := 0; shift < len(source); shift++ {
		rotated := append(append([]Field(nil), source[shift:]...), source[:shift]...)
		if !reflect.DeepEqual(semanticFacts(t, append(rotated, endField())), expected) {
			t.Fatalf("rotation %d differs", shift)
		}
	}
	forward := mustDecode(t, append(append([]Field(nil), source...), endField()))
	reversed := make([]Field, len(source))
	for i, f := range source {
		reversed[len(source)-1-i] = f
	}
	rev := mustDecode(t, append(reversed, endField()))
	if reflect.DeepEqual(forward.Facts.Provenance, rev.Facts.Provenance) {
		t.Fatal("provenance should differ by order")
	}
}

func TestRecognizesFourByteZeroSuccessControl(t *testing.T) {
	success := mustDecode(t, []Field{
		rawTag(0x0503, nil),
		rawTag(0x0201, utf16le("RESULT")),
		rawField(TagUnresolved0420, make([]byte, 4)),
		endField(),
	})
	if success.Outcome != OutcomeSuccess || success.SuccessControl != SuccessControlZero {
		t.Fatalf("outcome = %s / %s", success.Outcome, success.SuccessControl)
	}
	if !reflect.DeepEqual(success.Facts.Unresolved0420, []UnresolvedControlFact{{Tag: TagUnresolved0420, Ordinal: 2, ByteLength: 4, ValueHex: "00000000"}}) {
		t.Fatalf("controls = %+v", success.Facts.Unresolved0420)
	}
	for _, controls := range [][]Field{
		{},
		{rawField(TagUnresolved0420, []byte{0, 0, 0, 1})},
		{rawField(TagUnresolved0420, make([]byte, 3))},
		{rawField(TagUnresolved0420, make([]byte, 4)), rawField(TagUnresolved0420, make([]byte, 4))},
	} {
		expectProtocolError(t, append(controls, endField()), DecodeOptions{}, "RFC_ERROR_ENVELOPE_UNRESOLVED_SUCCESS_CONTROL")
	}
}

func TestUnresolved0420DoesNotOverrideError(t *testing.T) {
	decoded := mustDecode(t, []Field{
		rawField(TagUnresolved0420, []byte{0xde, 0xad}),
		textField(TagExceptionKey, "DECLARED", 0),
		rawField(TagUnresolved0420, []byte{0xbe, 0xef}),
		endField(),
	})
	if decoded.Outcome != OutcomeAbapException {
		t.Fatalf("outcome = %s", decoded.Outcome)
	}
	var hexes []string
	for _, f := range decoded.Facts.Unresolved0420 {
		hexes = append(hexes, f.ValueHex)
	}
	if !reflect.DeepEqual(hexes, []string{"dead", "beef"}) {
		t.Fatalf("hexes = %v", hexes)
	}
}

func TestRejectsDuplicateConflictingAmbiguousClassUnknown(t *testing.T) {
	expectProtocolError(t, []Field{textField(TagExceptionKey, "ONE", 0), textField(TagExceptionKey, "TWO", 0), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_DUPLICATE_FACT")
	expectProtocolError(t, []Field{textField(TagExceptionKey, "DECLARED", 0), textField(TagRuntimeID, "RUNTIME", 0), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_CONFLICTING_DISCRIMINATORS")
	for _, tag := range []Tag{TagMessageV1, TagMessageV2, TagMessageV3, TagMessageV4, TagCallStack} {
		expectProtocolError(t, []Field{textField(tag, "secondary", 0), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_AMBIGUOUS_FACTS")
	}
	for _, tag := range []Tag{TagClassException, TagClassExceptionEnd} {
		expectProtocolError(t, []Field{rawField(tag, nil), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_CLASS_EXCEPTION_UNSUPPORTED")
	}
	expectProtocolError(t, []Field{rawTag(0x7777, nil), rawField(TagUnresolved0420, make([]byte, 4)), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_UNKNOWN_TAG")
	expectProtocolError(t, []Field{rawField(TagUseClassExceptions, []byte{1}), rawField(TagUseClassExceptions, []byte{0}), textField(TagExceptionKey, "DECLARED", 0), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_DUPLICATE_FACT")

	supplemental := mustDecode(t, []Field{
		textField(TagMessageClass, "SR", 0),
		textField(TagMessageType, "E", 0),
		textField(TagMessageNumber, "006", 0),
		textField(TagMessageV1, "Method = 1", 0),
		textField(TagExceptionKey, "RAISE_EXCEPTION", 0),
		rawField(TagClassExceptionInfo, bytesRepeat(0xa5, 96)),
		endField(),
	})
	if supplemental.Outcome != OutcomeAbapException {
		t.Fatalf("supplemental outcome = %s", supplemental.Outcome)
	}
	last := supplemental.Facts.Provenance[len(supplemental.Facts.Provenance)-1]
	if last != (FactProvenance{Tag: uint16(TagClassExceptionInfo), Ordinal: 5, ByteLength: 96}) {
		t.Fatalf("last provenance = %+v", last)
	}
	for _, fields := range [][]Field{
		{rawField(TagClassExceptionInfo, make([]byte, 96)), endField()},
		{rawField(TagUseClassExceptions, []byte{1}), textField(TagExceptionKey, "RAISE_EXCEPTION", 0), rawField(TagClassExceptionInfo, make([]byte, 96)), endField()},
	} {
		expectProtocolError(t, fields, DecodeOptions{}, "RFC_ERROR_ENVELOPE_CLASS_EXCEPTION_UNSUPPORTED")
	}

	allowed, err := Decode([]Field{rawTag(0x7777, nil), rawField(TagUnresolved0420, make([]byte, 4)), endField()}, DecodeOptions{AdditionalAllowedTags: []uint16{0x7777}})
	if err != nil || allowed.Outcome != OutcomeSuccess {
		t.Fatalf("allowed = %+v, %v", allowed, err)
	}
}

func TestRequiresNonEmptyDiscriminators(t *testing.T) {
	for _, tag := range []Tag{TagExceptionKey, TagRuntimeID} {
		expectProtocolError(t, []Field{textField(tag, "", 3), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_EMPTY_DISCRIMINATOR")
	}
}

func TestStrictUTF16LE(t *testing.T) {
	valid := mustDecode(t, []Field{
		textField(TagMessageV1, "leading \U0001F642 combining é", 4),
		textField(TagExceptionKey, "DECLARED", 0),
		endField(),
	})
	if valid.Facts.MessageV1 != "leading \U0001F642 combining é" {
		t.Fatalf("messageV1 = %q", valid.Facts.MessageV1)
	}
	expectProtocolError(t, []Field{rawField(TagExceptionKey, []byte{0x41}), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_ODD_UTF16_LENGTH")
	expectProtocolError(t, []Field{rawField(TagExceptionKey, utf16le("A\x00B")), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_EMBEDDED_NUL")
	for _, b := range [][]byte{{0x00, 0xd8}, {0x00, 0xdc}, {0x00, 0xd8, 0x41, 0x00}} {
		expectProtocolError(t, []Field{rawField(TagExceptionKey, b), endField()}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_UNPAIRED_SURROGATE")
	}
}

func TestPreservesLeadingSpacesTrimsRightAndEnforcesLimits(t *testing.T) {
	decoded := mustDecode(t, []Field{textField(TagExceptionKey, "  DECLARED", 3), endField()})
	if decoded.Facts.ExceptionKey != "  DECLARED" {
		t.Fatalf("exceptionKey = %q", decoded.Facts.ExceptionKey)
	}
	expectProtocolError(t, []Field{textField(TagExceptionKey, "TOO-LONG", 0), endField()}, DecodeOptions{MaxTextByteLength: p(2)}, "RFC_ERROR_ENVELOPE_TEXT_TOO_LARGE")
	expectProtocolError(t, []Field{rawField(TagUnresolved0420, make([]byte, 5)), endField()}, DecodeOptions{MaxControlByteLength: p(4)}, "RFC_ERROR_ENVELOPE_CONTROL_TOO_LARGE")
	expectProtocolError(t, []Field{textField(TagExceptionKey, "AA", 0), textField(TagMessageV1, "BB", 0), endField()}, DecodeOptions{MaxTotalTextByteLength: p(6)}, "RFC_ERROR_ENVELOPE_TOTAL_TEXT_TOO_LARGE")
}

func TestBoundsFieldsAndAggregateControls(t *testing.T) {
	control := bytesRepeat(0x5a, 4)
	exactControls := []Field{rawField(TagUnresolved0420, control), rawField(TagUnresolved0420, control), rawField(TagUnresolved0420, control)}
	exact, err := Decode(append(append([]Field{textField(TagRuntimeID, "RUNTIME_ID", 0)}, exactControls...), endField()),
		DecodeOptions{MaxFieldCount: p(5), MaxControlCount: p(3), MaxControlByteLength: p(4), MaxTotalControlByteLength: p(12)})
	if err != nil || exact.Outcome != OutcomeAbapRuntime || len(exact.Facts.Unresolved0420) != 3 {
		t.Fatalf("exact = %+v, %v", exact, err)
	}
	expectProtocolError(t, append(append([]Field{textField(TagRuntimeID, "RUNTIME_ID", 0)}, append(append([]Field(nil), exactControls...), rawField(TagUnresolved0420, nil))...), endField()),
		DecodeOptions{MaxFieldCount: p(6), MaxControlCount: p(3), MaxControlByteLength: p(4), MaxTotalControlByteLength: p(12)}, "RFC_ERROR_ENVELOPE_TOO_MANY_CONTROLS")
	expectProtocolError(t, append(append([]Field{textField(TagRuntimeID, "RUNTIME_ID", 0)}, append(append([]Field(nil), exactControls...), rawField(TagUnresolved0420, []byte{1}))...), endField()),
		DecodeOptions{MaxFieldCount: p(6), MaxControlCount: p(4), MaxControlByteLength: p(4), MaxTotalControlByteLength: p(12)}, "RFC_ERROR_ENVELOPE_TOTAL_CONTROL_TOO_LARGE")
	expectProtocolError(t, append(append([]Field{textField(TagRuntimeID, "RUNTIME_ID", 0)}, exactControls...), endField()),
		DecodeOptions{MaxFieldCount: p(4), MaxControlCount: p(3), MaxControlByteLength: p(4), MaxTotalControlByteLength: p(12)}, "RFC_ERROR_ENVELOPE_TOO_MANY_FIELDS")

	var manyTiny []Field
	for i := 0; i < 65; i++ {
		manyTiny = append(manyTiny, rawField(TagUnresolved0420, nil))
	}
	expectProtocolError(t, append(append([]Field{textField(TagRuntimeID, "RUNTIME_ID", 0)}, manyTiny...), endField()), DecodeOptions{}, "RFC_ERROR_ENVELOPE_TOO_MANY_CONTROLS")
}

func TestSnapshotsIndependentlyOfCallerBuffers(t *testing.T) {
	exceptionBytes := utf16le("DECLARED")
	controlBytes := []byte{0xde, 0xad, 0xbe, 0xef}
	fields := []Field{{Tag: uint16(TagExceptionKey), Value: exceptionBytes}, {Tag: uint16(TagUnresolved0420), Value: controlBytes}, endField()}
	decoded := mustDecode(t, fields)
	for i := range exceptionBytes {
		exceptionBytes[i] = 0x20
	}
	for i := range controlBytes {
		controlBytes[i] = 0
	}
	if decoded.Facts.ExceptionKey != "DECLARED" || decoded.Facts.Unresolved0420[0].ValueHex != "deadbeef" {
		t.Fatalf("aliased caller buffers: %+v", decoded.Facts)
	}
	want := []FactProvenance{{Tag: uint16(TagExceptionKey), Ordinal: 0, ByteLength: 16}, {Tag: uint16(TagUnresolved0420), Ordinal: 1, ByteLength: 4}}
	if !reflect.DeepEqual(decoded.Facts.Provenance, want) {
		t.Fatalf("provenance = %+v", decoded.Facts.Provenance)
	}
}

func TestValidatesTerminalEndPlacement(t *testing.T) {
	expectProtocolError(t, []Field{}, DecodeOptions{}, "RFC_ERROR_ENVELOPE_MISSING_END")
	for _, fields := range [][]Field{
		{endFieldVal([]byte{0})},
		{endField(), endField()},
		{endField(), rawField(TagUnresolved0420, make([]byte, 4))},
	} {
		expectProtocolError(t, fields, DecodeOptions{}, "RFC_ERROR_ENVELOPE_INVALID_END")
	}
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

// FuzzDecode asserts the envelope decoder never panics on arbitrary field data.
func FuzzDecode(f *testing.F) {
	f.Add([]byte("DECLARED"), uint16(uint16(TagExceptionKey)))
	f.Fuzz(func(t *testing.T, value []byte, tag uint16) {
		_, _ = Decode([]Field{{Tag: tag, Value: value}, endField()}, DecodeOptions{})
	})
}
