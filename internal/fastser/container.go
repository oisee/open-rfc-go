// SPDX-License-Identifier: Apache-2.0

package fastser

// The item grammar.
//
// What was long read as a bespoke "0x5001 container" is one item of a general
// transport-layer grammar:
//
//	<id:2 BE> <len:2 BE> <data:len> <id:2 BE>
//
// The identifier is repeated after the data as a closing tag. 0x5001 is simply
// the id of the item that carries a serialized parameter block; 0x0130, for
// instance, carries the ABAP program name.
//
// The closing tag is what makes the grammar self-checking, and its absence is
// what makes a naive scan go wrong. Searching a frame for occurrences of 0x5001
// finds each item twice — once opening, once closing — and reading the closing
// one as an opening one takes the *next item's id* as a length. That is exactly
// the "second occurrence gives a length that runs past the end of the frame"
// that stalled an earlier pass over these captures: the 304 it could not explain
// was 0x0130, the id that followed.

// ItemFastSerParams is the item id carrying a serialized parameter block.
const ItemFastSerParams = 0x5001

// Item is one decoded transport item.
type Item struct {
	// ID identifies what the item carries.
	ID uint16
	// Data is a copy of the item's payload.
	Data []byte
	// Offset is where the opening id sat.
	Offset int
}

// DecodeItemAt reads the item starting at off. It requires the closing tag to
// match the opening id, which is the check that keeps a closing tag from being
// mistaken for the start of another item.
func DecodeItemAt(payload []byte, off int) (it Item, next int, ok bool) {
	if off+4 > len(payload) {
		return Item{}, off, false
	}
	id := uint16(payload[off])<<8 | uint16(payload[off+1])
	length := int(payload[off+2])<<8 | int(payload[off+3])

	end := off + 4 + length
	if end+2 > len(payload) {
		return Item{}, off, false
	}
	closing := uint16(payload[end])<<8 | uint16(payload[end+1])
	if closing != id {
		return Item{}, off, false
	}
	return Item{ID: id, Data: clone(payload[off+4 : end]), Offset: off}, end + 2, true
}

// DecodeItems walks the items in a payload from off, stopping at the first
// position that does not open a well-formed item. It does not skip ahead: an
// item stream is contiguous, and a gap means the walk lost alignment rather than
// that an item is merely unrecognised.
func DecodeItems(payload []byte, off int) (items []Item, next int) {
	for off < len(payload) {
		it, after, ok := DecodeItemAt(payload, off)
		if !ok {
			return items, off
		}
		items = append(items, it)
		off = after
	}
	return items, off
}

// FindItems locates every well-formed item in a payload, including ones the
// caller has no offset for. Unlike DecodeItems it may skip bytes, so use it to
// explore a frame whose framing above the item layer is not yet modelled — and
// prefer DecodeItems once an offset is known, because a skipping search can in
// principle admit a run that merely looks like an item.
func FindItems(payload []byte) []Item {
	var out []Item
	for i := 0; i+6 <= len(payload); {
		it, next, ok := DecodeItemAt(payload, i)
		if !ok {
			i++
			continue
		}
		out = append(out, it)
		i = next
	}
	return out
}
