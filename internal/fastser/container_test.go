// SPDX-License-Identifier: Apache-2.0

package fastser

import "testing"

// pingItem is one whole item from a live RFC_PING response: the id 0x5001, a
// 15-byte payload ending in the 'E' marker, and the id repeated as the closing
// tag.
const pingItem = "5001000f2448030300410300230040200000455001"

func TestDecodeItemRequiresTheClosingTag(t *testing.T) {
	payload := mustHex(t, pingItem)
	it, next, ok := DecodeItemAt(payload, 0)
	if !ok {
		t.Fatal("the item did not decode")
	}
	if it.ID != ItemFastSerParams {
		t.Errorf("id = %#04x, want %#04x", it.ID, ItemFastSerParams)
	}
	if len(it.Data) != 15 {
		t.Errorf("payload = %d bytes, want 15", len(it.Data))
	}
	if it.Data[len(it.Data)-1] != TagEnd {
		t.Errorf("payload should end in the 'E' marker, got %#x", it.Data[len(it.Data)-1])
	}
	if next != len(payload) {
		t.Errorf("next = %d, want %d — the closing tag is part of the item", next, len(payload))
	}
}

func TestClosingTagIsNotAnOpeningOne(t *testing.T) {
	// The trap this grammar exists to close. Scanning a frame for 0x5001 finds
	// each item twice, and reading the closing tag as an opening one takes the
	// NEXT item's id for a length — which is where an earlier pass over these
	// captures got a length that ran past the end of the frame.
	payload := append(mustHex(t, pingItem), 0x01, 0x30, 0x00, 0x02, 'h', 'i', 0x01, 0x30)
	closing := len(mustHex(t, pingItem)) - 2 // where the closing 5001 sits
	if _, _, ok := DecodeItemAt(payload, closing); ok {
		t.Error("a closing tag must not decode as the start of an item")
	}
	items, next := DecodeItems(payload, 0)
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2: %+v", len(items), items)
	}
	if items[1].ID != 0x0130 {
		t.Errorf("second item id = %#04x, want 0x0130", items[1].ID)
	}
	if next != len(payload) {
		t.Errorf("walk stopped at %d of %d", next, len(payload))
	}
}

func TestDecodeItemRefusesMalformed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{"empty", nil},
		{"header only", []byte{0x50, 0x01, 0x00, 0x0f}},
		{"length past the end", []byte{0x50, 0x01, 0xff, 0xff, 0x00}},
		{"closing tag differs", []byte{0x50, 0x01, 0x00, 0x01, 0x41, 0x50, 0x02}},
	} {
		if _, _, ok := DecodeItemAt(tc.payload, 0); ok {
			t.Errorf("%s: should not decode", tc.name)
		}
	}
}

func TestDecodeItemsStopsRatherThanSkips(t *testing.T) {
	payload := append(mustHex(t, pingItem), 0xff, 0xff, 0xff)
	items, next := DecodeItems(payload, 0)
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if next != len(payload)-3 {
		t.Errorf("stopped at %d, want %d — an item stream is contiguous", next, len(payload)-3)
	}
}
