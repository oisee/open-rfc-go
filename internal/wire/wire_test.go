// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc test/bytes.test.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten for the testing package. The
// "checked byte writers reject overflow" assertion has no Go analogue — a
// 256 argument to WriteUint8 is rejected by the compiler, not at run time — so
// only its bounds-past-end half survives. See docs/provenance.md.

package wire

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

func TestCheckedBytePrimitivesPreserveOffsetsAndEndianSign(t *testing.T) {
	w, err := NewWriter(12, "test record")
	if err != nil {
		t.Fatal(err)
	}
	orPanic(t, w.WriteUint8(0xab, "byte"))
	orPanic(t, w.WriteUint16BE(0xcdef, "short"))
	orPanic(t, w.WriteUint32BE(0x1234_5678, "word"))
	orPanic(t, w.WriteInt32BE(-2, "signed word"))
	orPanic(t, w.WriteBytes([]byte{0x99}, "tail"))
	if w.Offset() != 12 {
		t.Fatalf("offset = %d, want 12", w.Offset())
	}
	record, err := w.Finish()
	if err != nil {
		t.Fatal(err)
	}

	r := NewReader(record, "test record")
	if v, err := r.ReadUint8("byte"); err != nil || v != 0xab {
		t.Fatalf("ReadUint8 = %#x, %v", v, err)
	}
	if v, err := r.ReadUint16BE("short"); err != nil || v != 0xcdef {
		t.Fatalf("ReadUint16BE = %#x, %v", v, err)
	}
	if v, err := r.ReadUint32BE("word"); err != nil || v != 0x1234_5678 {
		t.Fatalf("ReadUint32BE = %#x, %v", v, err)
	}
	if v, err := r.ReadInt32BE("signed word"); err != nil || v != -2 {
		t.Fatalf("ReadInt32BE = %d, %v", v, err)
	}
	if v, err := r.ReadBytes(1, "tail"); err != nil || !bytes.Equal(v, []byte{0x99}) {
		t.Fatalf("ReadBytes = %#x, %v", v, err)
	}
	if r.Remaining() != 0 {
		t.Fatalf("remaining = %d, want 0", r.Remaining())
	}
	if err := r.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}
}

func TestCheckedByteReaderReportsFieldPathOffsetAndLength(t *testing.T) {
	r := NewReader([]byte{1, 2}, "APPC header")
	if _, err := r.ReadUint8("version"); err != nil {
		t.Fatal(err)
	}
	_, err := r.ReadUint16BE("uid")
	if !errors.Is(err, ErrShortRead) {
		t.Fatalf("want ErrShortRead, got %v", err)
	}
	for _, want := range []string{"APPC header.uid", "2 bytes", "offset 1", "1 remain"} {
		if !contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

func TestCheckedByteReaderClassicLittleEndianInt4(t *testing.T) {
	r := NewReader(mustHex(t, "88000000feffffff"), "classic row")
	if v, err := r.ReadUint32LE("position"); err != nil || v != 136 {
		t.Fatalf("ReadUint32LE = %d, %v", v, err)
	}
	if v, err := r.ReadInt32LE("signed"); err != nil || v != -2 {
		t.Fatalf("ReadInt32LE = %d, %v", v, err)
	}
	if err := r.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}
}

func TestCheckedByteWriterRejectsWritingPastEnd(t *testing.T) {
	w, err := NewWriter(1, "small record")
	if err != nil {
		t.Fatal(err)
	}
	orPanic(t, w.WriteUint8(1, "byte"))
	err = w.WriteUint8(2, "extra")
	if !errors.Is(err, ErrShortRead) {
		t.Fatalf("want ErrShortRead, got %v", err)
	}
	for _, want := range []string{"small record.extra", "1 bytes", "0 remain"} {
		if !contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

func TestFinishRejectsUnreadAndUnwrittenTrailingBytes(t *testing.T) {
	r := NewReader([]byte{0, 0}, "reader")
	if _, err := r.ReadUint8("first"); err != nil {
		t.Fatal(err)
	}
	if err := r.Finish(); !errors.Is(err, ErrTrailingBytes) || !contains(err.Error(), "1 unread bytes") {
		t.Fatalf("reader Finish = %v", err)
	}

	w, err := NewWriter(2, "writer")
	if err != nil {
		t.Fatal(err)
	}
	orPanic(t, w.WriteUint8(1, "first"))
	if _, err := w.Finish(); !errors.Is(err, ErrTrailingBytes) || !contains(err.Error(), "1 unwritten bytes") {
		t.Fatalf("writer Finish = %v", err)
	}
}

func TestReaderCopiesCallerData(t *testing.T) {
	src := []byte{0xaa, 0xbb}
	r := NewReader(src, "record")
	src[0] = 0x00
	if v, err := r.ReadUint8("byte"); err != nil || v != 0xaa {
		t.Fatalf("reader aliased caller memory: got %#x, %v", v, err)
	}
}

func orPanic(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func contains(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}
