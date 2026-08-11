// SPDX-License-Identifier: Apache-2.0
//
// Original work by open-rfc-go contributors. No upstream counterpart: open-rfc
// keeps its wire facts inside .test.ts files. See conformance/README.md.

// Package conformance runs the language-neutral wire vectors in testdata
// against this implementation.
package conformance

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisee/open-rfc-go/internal/ni"
)

type niVectorFile struct {
	SchemaVersion int    `json:"schemaVersion"`
	Layer         string `json:"layer"`
	Rule          string `json:"rule"`
	Provenance    string `json:"provenance"`
	Encode        []struct {
		Name       string `json:"name"`
		PayloadHex string `json:"payloadHex"`
		FrameHex   string `json:"frameHex"`
		Why        string `json:"why"`
	} `json:"encode"`
	Decode []struct {
		Name             string   `json:"name"`
		MaxPayloadLength int      `json:"maxPayloadLength"`
		ChunksHex        []string `json:"chunksHex"`
		PayloadsHex      []string `json:"payloadsHex"`
		ResidualBytes    int      `json:"residualBytes"`
		Error            string   `json:"error"`
		FinishError      string   `json:"finishError"`
		Why              string   `json:"why"`
	} `json:"decode"`
}

// niErrors maps the vectors' neutral error names onto this implementation's
// error values. The JSON never names a Go identifier, so that the same file can
// be run by another language.
var niErrors = map[string]error{
	"payload-too-large": ni.ErrPayloadTooLarge,
	"truncated-stream":  ni.ErrTruncatedStream,
}

func mustDecodeHex(t *testing.T, encoded, field string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatalf("%s is not valid hex: %v", field, err)
	}
	return decoded
}

func TestNiFramingVectors(t *testing.T) {
	path := filepath.Join("testdata", "vectors", "ni-framing.v1.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	var file niVectorFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	if file.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", file.SchemaVersion)
	}
	if file.Rule == "" || file.Provenance == "" {
		t.Fatal("every vector file must state its rule and its provenance")
	}
	if len(file.Encode) == 0 || len(file.Decode) == 0 {
		t.Fatal("vector file has no cases")
	}

	for _, vector := range file.Encode {
		t.Run("encode/"+vector.Name, func(t *testing.T) {
			if vector.Why == "" {
				t.Fatal("vector does not say why it matters")
			}
			payload := mustDecodeHex(t, vector.PayloadHex, "payloadHex")
			want := mustDecodeHex(t, vector.FrameHex, "frameHex")

			got, err := ni.EncodeFrame(payload)
			if err != nil {
				t.Fatalf("EncodeFrame returned %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("frame = %s, want %s", hex.EncodeToString(got), vector.FrameHex)
			}
		})
	}

	for _, vector := range file.Decode {
		t.Run("decode/"+vector.Name, func(t *testing.T) {
			if vector.Why == "" {
				t.Fatal("vector does not say why it matters")
			}
			decoder, err := ni.NewFrameDecoder(vector.MaxPayloadLength)
			if err != nil {
				t.Fatalf("NewFrameDecoder returned %v", err)
			}

			var decoded [][]byte
			var pushErr error
			for _, chunkHex := range vector.ChunksHex {
				chunk := mustDecodeHex(t, chunkHex, "chunksHex")
				payloads, err := decoder.Push(chunk)
				if err != nil {
					pushErr = err
					break
				}
				decoded = append(decoded, payloads...)
			}

			if vector.Error != "" {
				want, ok := niErrors[vector.Error]
				if !ok {
					t.Fatalf("vector names unknown error %q", vector.Error)
				}
				if !errors.Is(pushErr, want) {
					t.Fatalf("Push returned %v, want %v", pushErr, want)
				}
				return
			}
			if pushErr != nil {
				t.Fatalf("Push returned %v, want no error", pushErr)
			}

			if len(decoded) != len(vector.PayloadsHex) {
				t.Fatalf("decoded %d payloads, want %d", len(decoded), len(vector.PayloadsHex))
			}
			for index, wantHex := range vector.PayloadsHex {
				want := mustDecodeHex(t, wantHex, "payloadsHex")
				if !bytes.Equal(decoded[index], want) {
					t.Fatalf("payload %d = %s, want %s",
						index, hex.EncodeToString(decoded[index]), wantHex)
				}
			}
			if decoder.Buffered() != vector.ResidualBytes {
				t.Fatalf("Buffered() = %d, want %d", decoder.Buffered(), vector.ResidualBytes)
			}

			finishErr := decoder.Finish()
			if vector.FinishError == "" {
				if finishErr != nil {
					t.Fatalf("Finish returned %v, want no error", finishErr)
				}
				return
			}
			want, ok := niErrors[vector.FinishError]
			if !ok {
				t.Fatalf("vector names unknown error %q", vector.FinishError)
			}
			if !errors.Is(finishErr, want) {
				t.Fatalf("Finish returned %v, want %v", finishErr, want)
			}
		})
	}
}
