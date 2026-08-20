// SPDX-License-Identifier: Apache-2.0
//
// Part of the recursive xRFC port; see recursive_xrfc.go for the file's
// provenance header. This file carries the scalar cell codec and the recursive
// encode/decode walk over the metadata graph.

package xrfc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/value"
)

func isNonFinite(f float64) bool { return math.IsInf(f, 0) || math.IsNaN(f) }

func recursiveBase64Len(byteLength int) int { return (byteLength + 2) / 3 * 4 }

func recursiveInteger(v any, min, max int64, path string) (string, error) {
	n, ok := asInt64(v)
	if !ok || n < min || n > max {
		return "", fmt.Errorf("%w: %s expects an integer in %d..%d", ErrRange, path, min, max)
	}
	return strconv.FormatInt(n, 10), nil
}

func recursiveFixedBytes(v any, length int, path string) ([]byte, error) {
	b, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("%w: %s expects bytes", ErrType, path)
	}
	if len(b) > length {
		return nil, fmt.Errorf("%w: %s accepts at most %d bytes", ErrRange, path, length)
	}
	out := make([]byte, length)
	copy(out, b)
	return out, nil
}

func recursiveTemporalRaw(field metadata.MetadataField, v any, path string) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s expects a temporal string", ErrType, path)
	}
	encoded, err := value.EncodeClassicTemporal(value.TemporalExid(field.InternalType), s, path)
	if err != nil {
		return "", err
	}
	raw := new(big.Int)
	for i := len(encoded) - 1; i >= 0; i-- {
		raw.Lsh(raw, 8)
		raw.Or(raw, big.NewInt(int64(encoded[i])))
	}
	return raw.String(), nil
}

func recursiveScalarText(field metadata.MetadataField, v any, path string, maxBytes int) (string, error) {
	if !supportedRecursiveScalarTypes[field.InternalType] {
		return "", fmt.Errorf("%w: %s xRFC scalar type %s is not implemented", ErrProtocol, path, field.InternalType)
	}
	switch field.InternalType {
	case "C", "N":
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("%w: %s expects a string", ErrType, path)
		}
		if err := value.AssertUnicodeScalarText(s, path); err != nil {
			return "", err
		}
		capacity, err := characterCapacity(field2rfc(field), path)
		if err != nil {
			return "", err
		}
		if utf16Len(s) > capacity {
			return "", fmt.Errorf("%w: %s does not fit %s(%d)", ErrRange, path, field.InternalType, capacity)
		}
		if field.InternalType == "N" {
			if !numericDigits.MatchString(s) {
				return "", fmt.Errorf("%w: %s expects at most %d decimal digits", ErrType, path, capacity)
			}
			if capacity > maxBytes {
				return "", fmt.Errorf("%w: %s XML value exceeds %d bytes", ErrRange, path, maxBytes)
			}
			return strings.Repeat("0", capacity-len(s)) + s, nil
		}
		return s, nil
	case "D":
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("%w: %s expects a DATE string", ErrType, path)
		}
		if err := value.AssertClassicDate(s, path); err != nil {
			return "", err
		}
		if s == "" || s == "        " {
			return "", nil
		}
		return s[0:4] + "-" + s[4:6] + "-" + s[6:8], nil
	case "T":
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("%w: %s expects a TIME string", ErrType, path)
		}
		if err := value.AssertClassicTime(s, path); err != nil {
			return "", err
		}
		if s == "" || s == "      " {
			return "", nil
		}
		return s[0:2] + ":" + s[2:4] + ":" + s[4:6], nil
	case "X":
		if recursiveBase64Len(int(field.UcLength)) > maxBytes {
			return "", fmt.Errorf("%w: %s XML value exceeds %d bytes", ErrRange, path, maxBytes)
		}
		b, err := recursiveFixedBytes(v, int(field.UcLength), path)
		if err != nil {
			return "", err
		}
		return base64.StdEncoding.EncodeToString(b), nil
	case "P":
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("%w: %s expects a decimal string", ErrType, path)
		}
		packed, err := value.EncodePackedDecimal(s, int(field.UcLength), int(field.Decimals), path)
		if err != nil {
			return "", err
		}
		return value.DecodePackedDecimal(packed, int(field.Decimals), path)
	case "F":
		f, ok := v.(float64)
		if !ok || isNonFinite(f) {
			return "", fmt.Errorf("%w: %s expects a finite number", ErrType, path)
		}
		return jsFloatString(f), nil
	case "I":
		return recursiveInteger(v, -0x8000_0000, 0x7fff_ffff, path)
	case "s":
		return recursiveInteger(v, -0x8000, 0x7fff, path)
	case "b":
		return recursiveInteger(v, 0, 0xff, path)
	case "8":
		n, ok := asInt64(v)
		if !ok {
			return "", fmt.Errorf("%w: %s expects a 64-bit integer", ErrType, path)
		}
		return strconv.FormatInt(n, 10), nil
	case "a":
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("%w: %s expects a decimal string", ErrType, path)
		}
		enc, err := value.EncodeDecimalFloat16(s, path)
		if err != nil {
			return "", err
		}
		return value.DecodeDecimalFloat16(enc, path)
	case "e":
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("%w: %s expects a decimal string", ErrType, path)
		}
		enc, err := value.EncodeDecimalFloat34(s, path)
		if err != nil {
			return "", err
		}
		return value.DecodeDecimalFloat34(enc, path)
	case "g":
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("%w: %s expects Unicode text", ErrType, path)
		}
		if err := value.AssertUnicodeScalarText(s, path); err != nil {
			return "", err
		}
		return s, nil
	case "y":
		b, ok := v.([]byte)
		if !ok {
			return "", fmt.Errorf("%w: %s expects bytes", ErrType, path)
		}
		if recursiveBase64Len(len(b)) > maxBytes {
			return "", fmt.Errorf("%w: %s XML value exceeds %d bytes", ErrRange, path, maxBytes)
		}
		return base64.StdEncoding.EncodeToString(b), nil
	default:
		return recursiveTemporalRaw(field, v, path)
	}
}

func recursiveInitialScalar(field metadata.MetadataField) (any, error) {
	if value.IsClassicTemporalExid(field.InternalType) {
		s, err := value.ClassicTemporalInitialValue(value.TemporalExid(field.InternalType))
		return s, err
	}
	switch field.InternalType {
	case "C", "N", "g":
		return "", nil
	case "D":
		return "00000000", nil
	case "T":
		return "000000", nil
	case "X", "y":
		return []byte{}, nil
	case "P", "a", "e":
		return "0", nil
	case "F":
		return float64(0), nil
	case "I", "s", "b":
		return int(0), nil
	case "8":
		return int64(0), nil
	default:
		return nil, fmt.Errorf("%w: unsupported recursive xRFC scalar type %s", ErrProtocol, field.InternalType)
	}
}

// field2rfc adapts a graph field to the shape characterCapacity expects.
func field2rfc(field metadata.MetadataField) rfctypes.RfcStructureField {
	return rfctypes.RfcStructureField{InternalLength: int32(field.UcLength)}
}

// --- encode -----------------------------------------------------------------

type encodeState struct {
	graph  metadata.Graph
	limits normalizedRecursiveLimits
	buf    bytes.Buffer
	bytes  int
	nodes  int
	rows   int
	cells  int
}

func (s *encodeState) emit(chunk string, path string) error {
	total := s.bytes + len(chunk)
	if total > s.limits.maxParameterBytes {
		return fmt.Errorf("%w: %s recursive xRFC XML exceeds %d bytes", ErrRange, path, s.limits.maxParameterBytes)
	}
	s.bytes = total
	s.buf.WriteString(chunk)
	return nil
}

func (s *encodeState) openTag(name, path string) error {
	esc, err := EscapeTag(name)
	if err != nil {
		return err
	}
	return s.emit("<"+esc+">", path)
}

func (s *encodeState) closeTag(name, path string) error {
	esc, err := EscapeTag(name)
	if err != nil {
		return err
	}
	return s.emit("</"+esc+">", path)
}

func (s *encodeState) assertDepth(depth int, path string) error {
	if depth > s.limits.maxDepth {
		return fmt.Errorf("%w: %s exceeds recursive xRFC depth %d", ErrRange, path, s.limits.maxDepth)
	}
	return nil
}

func (s *encodeState) visitContainer(path string) error {
	s.nodes++
	if s.nodes > s.limits.maxNodes {
		return fmt.Errorf("%w: %s exceeds recursive xRFC runtime node count %d", ErrRange, path, s.limits.maxNodes)
	}
	return nil
}

func recursiveEscapedText(v, path string, maxBytes int) (string, error) {
	if err := value.AssertUnicodeScalarText(v, path); err != nil {
		return "", err
	}
	var b strings.Builder
	bytesUsed := 0
	for _, r := range v {
		cp := int(r)
		if cp == 0xfffe || cp == 0xffff {
			return "", fmt.Errorf("%w: %s contains an unsupported non-character", ErrRange, path)
		}
		var chunk string
		if canonicalEntityCodePoint(cp) {
			chunk = fmt.Sprintf("&#%02d;", cp)
		} else {
			chunk = string(r)
		}
		bytesUsed += len(chunk)
		if bytesUsed > maxBytes {
			return "", fmt.Errorf("%w: %s XML value exceeds %d bytes", ErrRange, path, maxBytes)
		}
		b.WriteString(chunk)
	}
	return b.String(), nil
}

func (s *encodeState) encodeScalar(field metadata.MetadataField, v any, path string) error {
	s.cells++
	if s.cells > s.limits.maxCells {
		return fmt.Errorf("%w: %s exceeds recursive xRFC cell count %d", ErrRange, path, s.limits.maxCells)
	}
	maxBytes := s.limits.maxCellBytes
	if rem := s.limits.maxParameterBytes - s.bytes; rem < maxBytes {
		if rem < 0 {
			rem = 0
		}
		maxBytes = rem
	}
	lexical, err := recursiveScalarText(field, v, path, maxBytes)
	if err != nil {
		return err
	}
	escaped, err := recursiveEscapedText(lexical, path, maxBytes)
	if err != nil {
		return err
	}
	return s.emit(escaped, path)
}

func (s *encodeState) encodeReference(field metadata.MetadataField, v any, depth int, path string) error {
	if field.Reference.Kind == "scalar" {
		return s.encodeScalar(field, v, path)
	}
	if err := s.assertDepth(depth, path); err != nil {
		return err
	}
	node, err := targetNode(s.graph, field.Reference, path)
	if err != nil {
		return err
	}
	if node.Kind == "structure" {
		return s.encodeStructure(node, v, depth, path)
	}
	return s.encodeTable(node, v, depth, path)
}

func (s *encodeState) encodeStructure(node metadata.TypeNode, v any, depth int, path string) error {
	if err := s.assertDepth(depth, path); err != nil {
		return err
	}
	if err := s.visitContainer(path); err != nil {
		return err
	}
	record, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: %s expects a structure object", ErrType, path)
	}
	known := map[string]bool{}
	for _, f := range node.Fields {
		known[f.Name] = true
	}
	for key := range record {
		if !known[key] {
			return fmt.Errorf("%w: %s contains unknown field %s", ErrProtocol, path, key)
		}
	}
	for _, field := range node.Fields {
		if field.Name == "" {
			return fmt.Errorf("%w: %s structure contains an anonymous field", ErrProtocol, path)
		}
		fieldPath := path + "." + field.Name
		fieldValue, supplied := record[field.Name]
		if !supplied {
			switch field.Reference.Kind {
			case "scalar":
				iv, err := recursiveInitialScalar(field)
				if err != nil {
					return err
				}
				fieldValue = iv
			case "table":
				fieldValue = []any{}
			default:
				fieldValue = map[string]any{}
			}
		}
		if err := s.openTag(field.Name, fieldPath); err != nil {
			return err
		}
		if err := s.encodeReference(field, fieldValue, depth+1, fieldPath); err != nil {
			return err
		}
		if err := s.closeTag(field.Name, fieldPath); err != nil {
			return err
		}
	}
	return nil
}

func (s *encodeState) encodeTableLine(node metadata.TypeNode, row any, depth int, path string) error {
	if len(node.Fields) == 1 && node.Fields[0].Name == "" {
		field := node.Fields[0]
		v := row
		if field.Reference.Kind == "scalar" {
			if m, ok := row.(map[string]any); ok {
				if _, has := m[""]; has {
					if len(m) != 1 {
						return fmt.Errorf("%w: %s scalar table wrapper must contain only the empty-name field", ErrType, path)
					}
					v = m[""]
				}
			}
		}
		return s.encodeReference(field, v, depth+1, path)
	}
	return s.encodeStructure(node, row, depth, path)
}

func (s *encodeState) encodeTable(node metadata.TypeNode, v any, depth int, path string) error {
	if err := s.assertDepth(depth, path); err != nil {
		return err
	}
	if err := s.visitContainer(path); err != nil {
		return err
	}
	rows, ok := v.([]any)
	if !ok {
		return fmt.Errorf("%w: %s expects an array of rows", ErrType, path)
	}
	for index, row := range rows {
		s.rows++
		if s.rows > s.limits.maxRows {
			return fmt.Errorf("%w: %s exceeds recursive xRFC row count %d", ErrRange, path, s.limits.maxRows)
		}
		rowPath := fmt.Sprintf("%s[%d]", path, index)
		if err := s.openTag("item", rowPath); err != nil {
			return err
		}
		if err := s.encodeTableLine(node, row, depth, rowPath); err != nil {
			return err
		}
		if err := s.closeTag("item", rowPath); err != nil {
			return err
		}
	}
	return nil
}

// EncodeRecursiveParameter encodes one graph-backed recursive xRFC parameter.
// For a structure kind, value must be map[string]any; for a table kind, []any.
func EncodeRecursiveParameter(parameter FunintParameter, graph metadata.Graph, v any, limits RecursiveLimits) ([]byte, error) {
	norm, err := normalizeRecursiveLimits(limits)
	if err != nil {
		return nil, err
	}
	resolved, err := validateAtDepth(graph, parameter, norm.maxDepth, nil)
	if err != nil {
		return nil, err
	}
	if graph.FunctionIdentity == nil {
		return nil, fmt.Errorf("%w: recursive xRFC graph lacks its function identity", ErrProtocol)
	}
	s := &encodeState{graph: graph, limits: norm}
	if err := s.openTag(parameter.ParameterName, parameter.ParameterName); err != nil {
		return nil, err
	}
	if resolved.Kind == KindStructure {
		err = s.encodeStructure(resolved.Node, v, 1, parameter.ParameterName)
	} else {
		err = s.encodeTable(resolved.Node, v, 1, parameter.ParameterName)
	}
	if err != nil {
		return nil, err
	}
	if err := s.closeTag(parameter.ParameterName, parameter.ParameterName); err != nil {
		return nil, err
	}
	return append([]byte(nil), s.buf.Bytes()...), nil
}

// --- decode -----------------------------------------------------------------

func recursiveDecodeEntities(raw, path string) (string, error) {
	if strings.Contains(raw, "]]>") {
		return "", fmt.Errorf("%w: %s contains non-canonical raw xRFC text", ErrProtocol, path)
	}
	for _, r := range raw {
		cp := int(r)
		if cp == 0xfffe || cp == 0xffff || (cp < 0x20 && cp != 9 && cp != 10 && cp != 13) {
			return "", fmt.Errorf("%w: %s contains non-canonical raw xRFC text", ErrProtocol, path)
		}
	}
	var b strings.Builder
	offset := 0
	for offset < len(raw) {
		amp := strings.IndexByte(raw[offset:], '&')
		if amp < 0 {
			b.WriteString(raw[offset:])
			break
		}
		amp += offset
		b.WriteString(raw[offset:amp])
		cp, length, err := value.DecodeXMLEntityReference(raw, amp, path)
		if err != nil {
			return "", err
		}
		if cp == 0xfffe || cp == 0xffff {
			return "", fmt.Errorf("%w: %s contains an out-of-range XML entity", ErrProtocol, path)
		}
		b.WriteRune(rune(cp))
		offset = amp + length
	}
	result := b.String()
	if err := value.AssertUnicodeScalarText(result, path); err != nil {
		return "", err
	}
	return result, nil
}

type rparser struct {
	text           string
	limits         normalizedRecursiveLimits
	offset         int
	nodes          int
	rows           int
	cells          int
	projectedBytes int
}

func (p *rparser) starts(name string, closing bool) (bool, error) {
	esc, err := EscapeTag(name)
	if err != nil {
		return false, err
	}
	token := "<" + esc + ">"
	if closing {
		token = "</" + esc + ">"
	}
	return strings.HasPrefix(p.text[p.offset:], token), nil
}

func (p *rparser) open(name, path string) error {
	esc, err := EscapeTag(name)
	if err != nil {
		return err
	}
	token := "<" + esc + ">"
	if !strings.HasPrefix(p.text[p.offset:], token) {
		return fmt.Errorf("%w: %s expected %s at byte %d", ErrProtocol, path, token, p.offset)
	}
	p.offset += len(token)
	return nil
}

func (p *rparser) close(name, path string) error {
	esc, err := EscapeTag(name)
	if err != nil {
		return err
	}
	token := "</" + esc + ">"
	if !strings.HasPrefix(p.text[p.offset:], token) {
		return fmt.Errorf("%w: %s expected %s at byte %d", ErrProtocol, path, token, p.offset)
	}
	p.offset += len(token)
	return nil
}

func (p *rparser) cell(path string) (string, error) {
	p.cells++
	if p.cells > p.limits.maxCells {
		return "", fmt.Errorf("%w: %s exceeds recursive xRFC cell count %d", ErrRange, path, p.limits.maxCells)
	}
	rel := strings.IndexByte(p.text[p.offset:], '<')
	if rel < 0 {
		return "", fmt.Errorf("%w: %s recursive xRFC XML is truncated", ErrProtocol, path)
	}
	raw := p.text[p.offset : p.offset+rel]
	if len(raw) > p.limits.maxCellBytes {
		return "", fmt.Errorf("%w: %s XML value exceeds %d bytes", ErrRange, path, p.limits.maxCellBytes)
	}
	p.offset += rel
	return recursiveDecodeEntities(raw, path)
}

func (p *rparser) node(path string) error {
	p.nodes++
	if p.nodes > p.limits.maxNodes {
		return fmt.Errorf("%w: %s exceeds recursive xRFC runtime node count %d", ErrRange, path, p.limits.maxNodes)
	}
	return nil
}

func (p *rparser) row(path string) error {
	p.rows++
	if p.rows > p.limits.maxRows {
		return fmt.Errorf("%w: %s exceeds recursive xRFC row count %d", ErrRange, path, p.limits.maxRows)
	}
	return nil
}

func (p *rparser) decodedValue(byteLength int, path string) error {
	p.projectedBytes += byteLength
	if p.projectedBytes > p.limits.maxParameterBytes {
		return fmt.Errorf("%w: %s decoded output exceeds the %d-byte parameter limit", ErrRange, path, p.limits.maxParameterBytes)
	}
	return nil
}

func (p *rparser) finish() error {
	if p.offset != len(p.text) {
		return fmt.Errorf("%w: recursive xRFC XML has trailing content at byte %d", ErrProtocol, p.offset)
	}
	return nil
}

func recursiveBase64Decode(v, path string, maximum int) ([]byte, int, error) {
	// The server wraps an XSTRING cell at 76 columns; see unwrapBase64.
	v = unwrapBase64(v)
	if len(v) > maximum || len(v)&3 != 0 || !canonicalBase64.MatchString(v) {
		return nil, 0, fmt.Errorf("%w: %s contains non-canonical base64", ErrProtocol, path)
	}
	decodedLen := len(v) / 4 * 3
	switch {
	case strings.HasSuffix(v, "=="):
		decodedLen -= 2
	case strings.HasSuffix(v, "="):
		decodedLen -= 1
	}
	if len(v) == 0 {
		return []byte{}, 0, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil || base64.StdEncoding.EncodeToString(decoded) != v {
		return nil, 0, fmt.Errorf("%w: %s contains non-canonical base64", ErrProtocol, path)
	}
	return decoded, decodedLen, nil
}

func parseCanonicalBig(text, path string) (*big.Int, error) {
	if !canonicalInteger.MatchString(text) || len(text) > 20 {
		return nil, fmt.Errorf("%w: %s contains a non-canonical integer", ErrProtocol, path)
	}
	n, ok := new(big.Int).SetString(text, 10)
	if !ok {
		return nil, fmt.Errorf("%w: %s contains a non-canonical integer", ErrProtocol, path)
	}
	return n, nil
}

func assertDecodedValueBytes(byteLength int, limits normalizedRecursiveLimits, path string) error {
	if byteLength > limits.maxCellBytes {
		return fmt.Errorf("%w: %s decoded value exceeds the %d-byte cell limit", ErrRange, path, limits.maxCellBytes)
	}
	if byteLength > limits.maxParameterBytes {
		return fmt.Errorf("%w: %s decoded value exceeds the %d-byte parameter limit", ErrRange, path, limits.maxParameterBytes)
	}
	return nil
}

func recursiveRawTemporal(field metadata.MetadataField, text, path string) (string, error) {
	width, err := value.ClassicTemporalByteLength(value.TemporalExid(field.InternalType))
	if err != nil {
		return "", err
	}
	raw, err := parseCanonicalBig(text, path)
	if err != nil {
		return "", err
	}
	limit := new(big.Int).Lsh(big.NewInt(1), uint(width*8))
	if raw.Sign() < 0 || raw.Cmp(limit) >= 0 {
		return "", fmt.Errorf("%w: %s compact temporal raw value is out of range", ErrRange, path)
	}
	buf := make([]byte, width)
	tmp := new(big.Int).Set(raw)
	mask := big.NewInt(0xff)
	for i := 0; i < width; i++ {
		buf[i] = byte(new(big.Int).And(tmp, mask).Int64())
		tmp.Rsh(tmp, 8)
	}
	return value.DecodeClassicTemporal(value.TemporalExid(field.InternalType), buf, path)
}

func decodedScalar(field metadata.MetadataField, text string, limits normalizedRecursiveLimits, parser *rparser, path string) (any, error) {
	switch field.InternalType {
	case "C", "N":
		capacity, err := characterCapacity(field2rfc(field), path)
		if err != nil {
			return nil, err
		}
		if field.InternalType == "N" {
			if err := assertDecodedValueBytes(capacity, limits, path); err != nil {
				return nil, err
			}
			if err := parser.decodedValue(capacity, path); err != nil {
				return nil, err
			}
		}
		if utf16Len(text) > capacity {
			return nil, fmt.Errorf("%w: %s exceeds %d characters", ErrRange, path, capacity)
		}
		if field.InternalType == "N" {
			if !numericDigits.MatchString(text) {
				return nil, fmt.Errorf("%w: %s contains a non-decimal NUM value", ErrProtocol, path)
			}
			return strings.Repeat("0", capacity-len(text)) + text, nil
		}
		return text, nil
	case "D":
		if len(text) == 0 {
			return "", nil
		}
		if !xrfcDatePattern.MatchString(text) {
			return nil, fmt.Errorf("%w: %s contains a non-canonical xRFC DATE", ErrProtocol, path)
		}
		date := strings.ReplaceAll(text, "-", "")
		if err := value.AssertClassicDate(date, path); err != nil {
			return nil, err
		}
		return date, nil
	case "T":
		if len(text) == 0 {
			return "", nil
		}
		if !xrfcTimePattern.MatchString(text) {
			return nil, fmt.Errorf("%w: %s contains a non-canonical xRFC TIME", ErrProtocol, path)
		}
		t := strings.ReplaceAll(text, ":", "")
		if err := value.AssertClassicTime(t, path); err != nil {
			return nil, err
		}
		return t, nil
	case "X":
		if err := assertDecodedValueBytes(int(field.UcLength), limits, path); err != nil {
			return nil, err
		}
		if err := parser.decodedValue(int(field.UcLength), path); err != nil {
			return nil, err
		}
		decoded, _, err := recursiveBase64Decode(text, path, limits.maxCellBytes)
		if err != nil {
			return nil, err
		}
		if len(decoded) > int(field.UcLength) {
			return nil, fmt.Errorf("%w: %s accepts at most %d bytes", ErrRange, path, field.UcLength)
		}
		padded := make([]byte, field.UcLength)
		copy(padded, decoded)
		return padded, nil
	case "P":
		packed, err := value.EncodePackedDecimal(text, int(field.UcLength), int(field.Decimals), path)
		if err != nil {
			return nil, err
		}
		return value.DecodePackedDecimal(packed, int(field.Decimals), path)
	case "F":
		if !finiteFloatLexical.MatchString(text) {
			return nil, fmt.Errorf("%w: %s contains an invalid FLOAT", ErrProtocol, path)
		}
		f, err := strconv.ParseFloat(text, 64)
		if err != nil || isNonFinite(f) {
			return nil, fmt.Errorf("%w: %s contains an invalid FLOAT", ErrProtocol, path)
		}
		return f, nil
	case "I":
		n, err := parseCanonicalBig(text, path)
		if err != nil {
			return nil, err
		}
		if !n.IsInt64() || n.Int64() < -0x8000_0000 || n.Int64() > 0x7fff_ffff {
			return nil, fmt.Errorf("%w: %s INT4 is out of range", ErrRange, path)
		}
		return int32(n.Int64()), nil
	case "s":
		n, err := parseCanonicalBig(text, path)
		if err != nil {
			return nil, err
		}
		if !n.IsInt64() || n.Int64() < -0x8000 || n.Int64() > 0x7fff {
			return nil, fmt.Errorf("%w: %s INT2 is out of range", ErrRange, path)
		}
		return int16(n.Int64()), nil
	case "b":
		n, err := parseCanonicalBig(text, path)
		if err != nil {
			return nil, err
		}
		if !n.IsInt64() || n.Int64() < 0 || n.Int64() > 0xff {
			return nil, fmt.Errorf("%w: %s INT1 is out of range", ErrRange, path)
		}
		return byte(n.Int64()), nil
	case "8":
		n, err := parseCanonicalBig(text, path)
		if err != nil {
			return nil, err
		}
		if !n.IsInt64() {
			return nil, fmt.Errorf("%w: %s INT8 is out of range", ErrRange, path)
		}
		return n.Int64(), nil
	case "a":
		enc, err := value.EncodeDecimalFloat16(text, path)
		if err != nil {
			return nil, err
		}
		return value.DecodeDecimalFloat16(enc, path)
	case "e":
		enc, err := value.EncodeDecimalFloat34(text, path)
		if err != nil {
			return nil, err
		}
		return value.DecodeDecimalFloat34(enc, path)
	case "g":
		if err := value.AssertUnicodeScalarText(text, path); err != nil {
			return nil, err
		}
		return text, nil
	case "y":
		_, byteLength, err := recursiveBase64Decode(text, path, limits.maxCellBytes)
		if err != nil {
			return nil, err
		}
		if err := assertDecodedValueBytes(byteLength, limits, path); err != nil {
			return nil, err
		}
		if err := parser.decodedValue(byteLength, path); err != nil {
			return nil, err
		}
		decoded, _, err := recursiveBase64Decode(text, path, limits.maxCellBytes)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	default:
		if value.IsClassicTemporalExid(field.InternalType) {
			return recursiveRawTemporal(field, text, path)
		}
		return nil, fmt.Errorf("%w: %s xRFC scalar type %s is not implemented", ErrProtocol, path, field.InternalType)
	}
}

func parseReference(graph metadata.Graph, field metadata.MetadataField, parser *rparser, limits normalizedRecursiveLimits, depth int, path string) (any, error) {
	if field.Reference.Kind == "scalar" {
		text, err := parser.cell(path)
		if err != nil {
			return nil, err
		}
		return decodedScalar(field, text, limits, parser, path)
	}
	if depth > limits.maxDepth {
		return nil, fmt.Errorf("%w: %s exceeds recursive xRFC depth %d", ErrRange, path, limits.maxDepth)
	}
	node, err := targetNode(graph, field.Reference, path)
	if err != nil {
		return nil, err
	}
	if node.Kind == "structure" {
		return parseStructure(graph, node, parser, limits, depth, path)
	}
	return parseTable(graph, node, parser, limits, depth, path)
}

func parseStructure(graph metadata.Graph, node metadata.TypeNode, parser *rparser, limits normalizedRecursiveLimits, depth int, path string) (map[string]any, error) {
	if depth > limits.maxDepth {
		return nil, fmt.Errorf("%w: %s exceeds recursive xRFC depth %d", ErrRange, path, limits.maxDepth)
	}
	if err := parser.node(path); err != nil {
		return nil, err
	}
	result := make(map[string]any, len(node.Fields))
	for _, field := range node.Fields {
		if field.Name == "" {
			return nil, fmt.Errorf("%w: %s structure contains an anonymous field", ErrProtocol, path)
		}
		fieldPath := path + "." + field.Name
		if err := parser.open(field.Name, fieldPath); err != nil {
			return nil, err
		}
		v, err := parseReference(graph, field, parser, limits, depth+1, fieldPath)
		if err != nil {
			return nil, err
		}
		result[field.Name] = v
		if err := parser.close(field.Name, fieldPath); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func parseTableLine(graph metadata.Graph, node metadata.TypeNode, parser *rparser, limits normalizedRecursiveLimits, depth int, path string) (any, error) {
	if len(node.Fields) == 1 && node.Fields[0].Name == "" {
		return parseReference(graph, node.Fields[0], parser, limits, depth+1, path)
	}
	return parseStructure(graph, node, parser, limits, depth, path)
}

func parseTable(graph metadata.Graph, node metadata.TypeNode, parser *rparser, limits normalizedRecursiveLimits, depth int, path string) ([]any, error) {
	if depth > limits.maxDepth {
		return nil, fmt.Errorf("%w: %s exceeds recursive xRFC depth %d", ErrRange, path, limits.maxDepth)
	}
	if err := parser.node(path); err != nil {
		return nil, err
	}
	rows := []any{}
	for {
		starts, err := parser.starts("item", false)
		if err != nil {
			return nil, err
		}
		if !starts {
			break
		}
		rowPath := fmt.Sprintf("%s[%d]", path, len(rows))
		if err := parser.row(rowPath); err != nil {
			return nil, err
		}
		if err := parser.open("item", rowPath); err != nil {
			return nil, err
		}
		row, err := parseTableLine(graph, node, parser, limits, depth, rowPath)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
		if err := parser.close("item", rowPath); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

// DecodeRecursiveParameter strictly decodes one recursive xRFC parameter. For a
// structure kind it returns map[string]any; for a table kind, []any.
func DecodeRecursiveParameter(parameter FunintParameter, graph metadata.Graph, v []byte, limits RecursiveLimits) (any, error) {
	norm, err := normalizeRecursiveLimits(limits)
	if err != nil {
		return nil, err
	}
	resolved, err := validateAtDepth(graph, parameter, norm.maxDepth, nil)
	if err != nil {
		return nil, err
	}
	if len(v) > norm.maxParameterBytes {
		return nil, fmt.Errorf("%w: %s recursive xRFC XML exceeds %d bytes", ErrRange, parameter.ParameterName, norm.maxParameterBytes)
	}
	text, err := decodeUTF8(v, parameter.ParameterName+" recursive xRFC XML")
	if err != nil {
		return nil, err
	}
	parser := &rparser{text: text, limits: norm}
	if err := parser.open(parameter.ParameterName, parameter.ParameterName); err != nil {
		return nil, err
	}
	var result any
	if resolved.Kind == KindStructure {
		result, err = parseStructure(graph, resolved.Node, parser, norm, 1, parameter.ParameterName)
	} else {
		result, err = parseTable(graph, resolved.Node, parser, norm, 1, parameter.ParameterName)
	}
	if err != nil {
		return nil, err
	}
	if err := parser.close(parameter.ParameterName, parameter.ParameterName); err != nil {
		return nil, err
	}
	if err := parser.finish(); err != nil {
		return nil, err
	}
	return result, nil
}
