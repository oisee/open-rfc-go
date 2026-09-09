// SPDX-License-Identifier: Apache-2.0
//
// Ported from open-rfc src/protocol/bytes.ts at commit 847036d,
// Copyright 2026 Marian Zeis, licensed under the Apache License, Version 2.0.
// Modified by open-rfc-go contributors: rewritten in Go. Only the
// CheckedByteReader/CheckedByteWriter classes are ported; the intrinsic
// typed-array geometry helpers (intrinsicUint8ArrayByteLength,
// intrinsicUint8ArrayView, snapshotUint8Array) are dropped as inapplicable —
// Go slices expose no user-overridable geometry accessors, so the class of
// attack they defend against is absent (see docs/provenance.md, and the same
// reasoning already recorded for src/protocol/ni.ts). Thrown RangeErrors became
// returned, wrapped sentinel errors; the Number.isSafeInteger range guards on
// written values became Go's fixed-width parameter types.
// See docs/provenance.md.

// Package wire provides bounds-checked, path-aware reads and fixed-size writes
// for the untrusted binary records exchanged above the NI framing layer.
//
// Every read is checked against the remaining length and, on failure, reports
// the record context, the field name, the offset, and how many bytes were
// needed versus available. That path is what turns an opaque short-read into a
// diagnosable protocol fault three layers up.
package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	// ErrShortRead reports a read or skip that would run past the end of the
	// record. It wraps a message naming the field, offset, and shortfall.
	ErrShortRead = errors.New("wire: short read")

	// ErrTrailingBytes reports that Finish was called while bytes remain
	// unread (reader) or unwritten (writer).
	ErrTrailingBytes = errors.New("wire: trailing bytes")

	// ErrNegativeLength reports a negative length passed to a read or skip.
	ErrNegativeLength = errors.New("wire: negative length")
)

// Reader consumes a byte record left to right, checking every read against the
// bytes that remain. It copies the record it is given, so later mutation of the
// caller's slice cannot change what a decode observes.
type Reader struct {
	data    []byte
	context string
	offset  int
}

// NewReader returns a Reader over a private copy of data. The context labels
// the record in error messages (for example "APPC header").
func NewReader(data []byte, context string) *Reader {
	if context == "" {
		context = "byte record"
	}
	return &Reader{data: append([]byte(nil), data...), context: context}
}

// Len reports the total record length.
func (r *Reader) Len() int { return len(r.data) }

// Offset reports how many bytes have been consumed.
func (r *Reader) Offset() int { return r.offset }

// Remaining reports how many bytes are left to read.
func (r *Reader) Remaining() int { return len(r.data) - r.offset }

func (r *Reader) ensure(length int, field string) error {
	if length < 0 {
		return fmt.Errorf("%w: %s.%s length %d", ErrNegativeLength, r.context, field, length)
	}
	if length > r.Remaining() {
		return fmt.Errorf(
			"%w: %s.%s: need %d bytes at offset %d; %d remain",
			ErrShortRead, r.context, field, length, r.offset, r.Remaining(),
		)
	}
	return nil
}

// ReadUint8 reads one byte.
func (r *Reader) ReadUint8(field string) (uint8, error) {
	if err := r.ensure(1, field); err != nil {
		return 0, err
	}
	v := r.data[r.offset]
	r.offset++
	return v, nil
}

// ReadUint16BE reads a big-endian uint16.
func (r *Reader) ReadUint16BE(field string) (uint16, error) {
	if err := r.ensure(2, field); err != nil {
		return 0, err
	}
	v := binary.BigEndian.Uint16(r.data[r.offset:])
	r.offset += 2
	return v, nil
}

// ReadUint32BE reads a big-endian uint32.
func (r *Reader) ReadUint32BE(field string) (uint32, error) {
	if err := r.ensure(4, field); err != nil {
		return 0, err
	}
	v := binary.BigEndian.Uint32(r.data[r.offset:])
	r.offset += 4
	return v, nil
}

// ReadUint32LE reads a little-endian uint32, as classic RFC rows carry.
func (r *Reader) ReadUint32LE(field string) (uint32, error) {
	if err := r.ensure(4, field); err != nil {
		return 0, err
	}
	v := binary.LittleEndian.Uint32(r.data[r.offset:])
	r.offset += 4
	return v, nil
}

// ReadInt32BE reads a big-endian, two's-complement int32.
func (r *Reader) ReadInt32BE(field string) (int32, error) {
	v, err := r.ReadUint32BE(field)
	return int32(v), err
}

// ReadInt32LE reads a little-endian, two's-complement int32.
func (r *Reader) ReadInt32LE(field string) (int32, error) {
	v, err := r.ReadUint32LE(field)
	return int32(v), err
}

// ReadBytes reads length bytes and returns a copy.
func (r *Reader) ReadBytes(length int, field string) ([]byte, error) {
	if err := r.ensure(length, field); err != nil {
		return nil, err
	}
	v := append([]byte(nil), r.data[r.offset:r.offset+length]...)
	r.offset += length
	return v, nil
}

// Skip advances past length bytes without returning them.
func (r *Reader) Skip(length int, field string) error {
	if err := r.ensure(length, field); err != nil {
		return err
	}
	r.offset += length
	return nil
}

// Finish reports an error if any bytes remain unread. A record decoder calls it
// to assert it consumed exactly what it was given.
func (r *Reader) Finish() error {
	if rem := r.Remaining(); rem != 0 {
		return fmt.Errorf("%w: %s: %d unread bytes remain", ErrTrailingBytes, r.context, rem)
	}
	return nil
}

// Writer fills a fixed-size record left to right, checking every write against
// the space that remains. The record length is chosen up front; Finish fails if
// it was not filled exactly.
type Writer struct {
	data    []byte
	context string
	offset  int
}

// NewWriter returns a Writer over a zeroed record of the given length. It
// returns an error only for a negative length.
func NewWriter(length int, context string) (*Writer, error) {
	if context == "" {
		context = "byte record"
	}
	if length < 0 {
		return nil, fmt.Errorf("%w: %s length %d", ErrNegativeLength, context, length)
	}
	return &Writer{data: make([]byte, length), context: context}, nil
}

// Len reports the total record length.
func (w *Writer) Len() int { return len(w.data) }

// Offset reports how many bytes have been written.
func (w *Writer) Offset() int { return w.offset }

// Remaining reports how many bytes are still to be written.
func (w *Writer) Remaining() int { return len(w.data) - w.offset }

func (w *Writer) ensure(length int, field string) error {
	if length < 0 {
		return fmt.Errorf("%w: %s.%s length %d", ErrNegativeLength, w.context, field, length)
	}
	if length > w.Remaining() {
		return fmt.Errorf(
			"%w: %s.%s: need %d bytes at offset %d; %d remain",
			ErrShortRead, w.context, field, length, w.offset, w.Remaining(),
		)
	}
	return nil
}

// WriteUint8 writes one byte.
func (w *Writer) WriteUint8(value uint8, field string) error {
	if err := w.ensure(1, field); err != nil {
		return err
	}
	w.data[w.offset] = value
	w.offset++
	return nil
}

// WriteUint16BE writes a big-endian uint16.
func (w *Writer) WriteUint16BE(value uint16, field string) error {
	if err := w.ensure(2, field); err != nil {
		return err
	}
	binary.BigEndian.PutUint16(w.data[w.offset:], value)
	w.offset += 2
	return nil
}

// WriteUint32BE writes a big-endian uint32.
func (w *Writer) WriteUint32BE(value uint32, field string) error {
	if err := w.ensure(4, field); err != nil {
		return err
	}
	binary.BigEndian.PutUint32(w.data[w.offset:], value)
	w.offset += 4
	return nil
}

// WriteInt32BE writes a big-endian, two's-complement int32.
func (w *Writer) WriteInt32BE(value int32, field string) error {
	return w.WriteUint32BE(uint32(value), field)
}

// WriteBytes writes value verbatim.
func (w *Writer) WriteBytes(value []byte, field string) error {
	if err := w.ensure(len(value), field); err != nil {
		return err
	}
	copy(w.data[w.offset:], value)
	w.offset += len(value)
	return nil
}

// Finish returns a copy of the completed record, or an error if any byte was
// left unwritten.
func (w *Writer) Finish() ([]byte, error) {
	if rem := w.Remaining(); rem != 0 {
		return nil, fmt.Errorf("%w: %s: %d unwritten bytes remain", ErrTrailingBytes, w.context, rem)
	}
	return append([]byte(nil), w.data...), nil
}
