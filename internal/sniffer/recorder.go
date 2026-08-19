// SPDX-License-Identifier: Apache-2.0

package sniffer

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"sync"
)

// recordLine is one captured frame as JSON.
type recordLine struct {
	Dir   string `json:"dir"`
	Conn  int    `json:"conn"`
	Label string `json:"label,omitempty"`
	Index int    `json:"index"`
	Note  string `json:"note"`
	Len   int    `json:"len"`
	Hex   string `json:"hex,omitempty"`
}

// JSONLRecorder returns an Observer that appends each frame to w as one JSON
// line (direction, index, note, byte length, and hex payload). maxBytes caps the
// hex captured per frame (0 = the whole payload). The returned Observer is safe
// for concurrent use.
//
// Captures of live traffic may contain credentials (the logon scramble, trusted
// tickets) and application data. Treat a capture file as sensitive: do not
// commit or share it.
func JSONLRecorder(w io.Writer, maxBytes int) Observer {
	var mu sync.Mutex
	enc := json.NewEncoder(w)
	return func(f Frame) {
		payload := f.Payload
		if maxBytes > 0 && len(payload) > maxBytes {
			payload = payload[:maxBytes]
		}
		line := recordLine{
			Dir:   string(f.Direction),
			Conn:  f.ConnID,
			Label: f.Label,
			Index: f.Index,
			Note:  f.Note,
			Len:   len(f.Payload),
			Hex:   hex.EncodeToString(payload),
		}
		mu.Lock()
		defer mu.Unlock()
		_ = enc.Encode(&line)
	}
}
