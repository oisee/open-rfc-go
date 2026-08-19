// SPDX-License-Identifier: Apache-2.0

package rfcserver

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"

	"github.com/oisee/open-rfc-go/internal/transport"
)

// niPing and niPong are the 8-byte NI keepalive frames ("NI_PING\0"/"NI_PONG\0").
var niPing = mustHex("4e495f50494e4700")
var niPong = mustHex("4e495f504f4e4700")

// scriptStep is one step of a function's recorded reply script: a frame to send
// to the client (send=true) or one to expect and consume from it (a callback
// response, send=false).
type scriptStep struct {
	send    bool
	payload []byte
}

// Templates holds a content-addressed set of server replies distilled from a
// capture: the gateway reply, logon-accepts keyed by init length, and a reply
// script per function name (which may include server->client callbacks).
type Templates struct {
	gateway []byte
	accepts map[int][]byte          // init length -> logon-accept
	funcs   map[string][]scriptStep // "acceptLen|FUNCTION" -> reply script
}

func funcKey(acceptLen int, name string) string {
	return fmt.Sprintf("%d|%s", acceptLen, name)
}

type capRow struct {
	Dir, Label, Hex string
	Conn            int
}

// LoadTemplates parses an rfc-sniffer capture into content-addressed reply
// templates. It reads the frames of one connection (the label's, if given) in
// order: the gateway reply, each CPIC-init's following accept (keyed by init
// length), and — after each decoded function request — the frames up to the next
// request as that function's script (server frames are sends, client frames are
// callback receives).
func LoadTemplates(paths []string, label string) (*Templates, error) {
	t := &Templates{accepts: map[int][]byte{}, funcs: map[string][]scriptStep{}}
	type fr struct {
		dir string
		b   []byte
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var seq []fr
		dec := json.NewDecoder(bytes.NewReader(data))
		for {
			var r capRow
			if dec.Decode(&r) != nil {
				break
			}
			if r.Hex == "" || (label != "" && r.Label != label) {
				continue
			}
			b, err := hex.DecodeString(r.Hex)
			if err != nil {
				continue
			}
			seq = append(seq, fr{r.Dir, b})
		}
		curAccept := 0 // length of the accept currently in effect (the session mode)
		for i := 0; i < len(seq); i++ {
			f := seq[i]
			switch {
			case f.dir == "S->C" && len(f.b) == 64 && t.gateway == nil:
				t.gateway = f.b
			case f.dir == "C->S" && isInit(f.b):
				for j := i + 1; j < len(seq); j++ {
					if seq[j].dir == "S->C" && isFSapSend(seq[j].b) {
						if _, ok := t.accepts[len(f.b)]; !ok {
							t.accepts[len(f.b)] = seq[j].b
						}
						curAccept = len(seq[j].b)
						break
					}
				}
			case f.dir == "C->S" && isFuncRequest(f.b):
				req, err := DecodeCutFunctionRequest(f.b[80:])
				if err != nil {
					continue
				}
				key := funcKey(curAccept, req.FunctionName)
				if _, ok := t.funcs[key]; ok {
					continue
				}
				var script []scriptStep
				for j := i + 1; j < len(seq); j++ {
					nb := seq[j].b
					if seq[j].dir == "C->S" && (isFuncRequest(nb) || isInit(nb)) {
						break
					}
					if len(nb) == 8 {
						continue
					}
					if seq[j].dir == "S->C" && isFSapSend(nb) {
						script = append(script, scriptStep{send: true, payload: nb})
					} else if seq[j].dir == "C->S" && isFSapSend(nb) && len(nb) > 80 {
						script = append(script, scriptStep{send: false, payload: nb})
					}
				}
				t.funcs[key] = script
			}
		}
	}
	if t.gateway == nil || len(t.accepts) == 0 {
		return nil, fmt.Errorf("captures have no gateway reply or logon-accept (label %q)", label)
	}
	// Seed the baked handshake accepts so this endpoint also serves the SM59
	// Connection Test (init 1818B -> 817B) and Unicode Test (1444B -> 1079B),
	// which a program-only capture may not contain.
	if _, ok := t.accepts[1818]; !ok {
		t.accepts[1818] = smartAccept
	}
	if _, ok := t.accepts[1444]; !ok {
		t.accepts[1444] = smartAcceptUni
	}
	return t, nil
}

func isInit(b []byte) bool     { return len(b) > 200 && b[0] == 0x06 && b[1] == 0x03 }
func isFSapSend(b []byte) bool { return len(b) >= 80 && b[0] == 0x06 && b[1] == 0xcb }
func isFuncRequest(b []byte) bool {
	return len(b) > 84 && b[0] == 0x06 && b[1] == 0xcb && b[80] == 0x05 && b[81] == 0x02
}

// acceptForLen returns the accept whose captured init length is closest to n.
func (t *Templates) acceptForLen(n int) []byte {
	var best []byte
	bestDelta := 1 << 30
	keys := make([]int, 0, len(t.accepts))
	for k := range t.accepts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		d := n - k
		if d < 0 {
			d = -d
		}
		if d < bestDelta {
			bestDelta = d
			best = t.accepts[k]
		}
	}
	return best
}

// ServeContentAddressed answers a live SM59 type-3 client by matching each
// request to a recorded reply script and patching this session's tokens in. It
// services NI keepalives, selects the logon-accept by init length, and runs each
// function's script (including STFC_CONNECTION's server->client callback).
func ServeContentAddressed(conn net.Conn, t *Templates, logf func(string)) {
	defer conn.Close()
	log := func(s string) {
		if logf != nil {
			logf(s)
		}
	}
	tr := transport.New(conn, transport.Options{})
	ctx := context.Background()
	var convID, guid []byte
	mode := 0 // length of the accept in effect this session (selects the reply set)
	pingStep := 0
	for {
		got, err := tr.Receive(ctx)
		if err != nil {
			log(fmt.Sprintf("client closed: %v", err))
			return
		}
		switch {
		case len(got) == 64:
			reply := append([]byte(nil), got...)
			reply[gatewayAckOffset1] = gatewayAckLevel
			reply[gatewayAckOffset2] = gatewayAckCaps
			if tr.Send(reply) != nil {
				return
			}
			log("CONNECT: gateway acknowledged")
		case len(got) == 8 && bytes.Equal(got, niPing):
			if tr.Send(niPong) != nil {
				return
			}
		case len(got) == 8: // NI_PONG or other keepalive — no reply
		case isInit(got):
			convID = append([]byte(nil), got[40:48]...)
			guid = findRFCGUID(got)
			acc := patchSession(t.acceptForLen(len(got)), convID, guid)
			if tr.Send(acc) != nil {
				return
			}
			mode = len(acc)
			pingStep = 0
			log(fmt.Sprintf("LOGON: init=%dB accept=%dB mode=%d (conv=%s)", len(got), len(acc), mode, string(convID)))
		case isFuncRequest(got):
			req, derr := DecodeCutFunctionRequest(got[80:])
			fn := "?"
			if derr == nil {
				fn = req.FunctionName
				log("  → CALL " + summarizeRequest(req))
			}
			if script, ok := t.funcs[funcKey(mode, fn)]; ok {
				if !runScript(tr, ctx, script, convID, guid, log) {
					return
				}
				log(fmt.Sprintf("SESSION: %s answered (mode=%d, %d step script)", fn, mode, len(script)))
			} else if fn == "RFC_PING" {
				resp := patchSession(smartPingSteps[pingStep%len(smartPingSteps)], convID, guid)
				if tr.Send(resp) != nil {
					return
				}
				pingStep++
				log("SESSION: RFC_PING step answered")
			} else {
				exc, _ := EncodeCutFunctionExceptionResponse("SYSTEM_FAILURE")
				wrapped, _ := wrapFSapSend(exc, convID, uidOf(got))
				_ = tr.Send(wrapped)
				log(fmt.Sprintf("SESSION: %s not in templates -> SYSTEM_FAILURE", fn))
			}
		default:
			// 80-byte control / turn frames need no reply
		}
	}
}

// runScript plays one function's recorded script: send each server frame (patched
// for this session) and consume each callback-receive frame. Returns false on a
// transport error.
func runScript(tr *transport.Transport, ctx context.Context, script []scriptStep, convID, guid []byte, log func(string)) bool {
	for _, st := range script {
		if st.send {
			if tr.Send(patchSession(st.payload, convID, guid)) != nil {
				return false
			}
		} else {
			if _, err := tr.Receive(ctx); err != nil { // consume the client's callback response
				return false
			}
		}
	}
	return true
}

func uidOf(frame []byte) uint16 {
	if len(frame) >= 6 {
		return uint16(frame[4])<<8 | uint16(frame[5])
	}
	return 0xffff
}

// summarizeRequest renders a decoded request for the live log: the function, the
// requested outputs, each import parameter as name=value, and each table by name
// and row count. It turns rfc-lab into a live inspector of what a client asks.
func summarizeRequest(req Request) string {
	out := req.FunctionName
	if len(req.RequestedOutputs) > 0 {
		out += fmt.Sprintf(" outputs=%v", req.RequestedOutputs)
	}
	for _, imp := range req.Imports {
		out += fmt.Sprintf(" %s=%s", imp.Name, valPreview(imp.Value))
	}
	for _, tb := range req.Tables {
		out += fmt.Sprintf(" %s[%d rows×%dB]", tb.Name, len(tb.Rows), tb.RowByteLength)
	}
	return out
}

// valPreview shows a parameter value as text when it is printable (ASCII or
// UTF-16LE), else as a short hex prefix, bounded so the log stays readable.
func valPreview(v []byte) string {
	if len(v) == 0 {
		return "\"\""
	}
	// UTF-16LE (classic char): every other byte is 0
	if len(v) >= 2 && len(v)%2 == 0 {
		ascii := true
		var sb []byte
		for i := 0; i+1 < len(v); i += 2 {
			if v[i+1] != 0 || (v[i] != 0 && (v[i] < 0x20 || v[i] > 0x7e)) {
				ascii = false
				break
			}
			if v[i] != 0 {
				sb = append(sb, v[i])
			}
		}
		if ascii && len(sb) > 0 {
			return fmt.Sprintf("%q", trimSp(string(sb)))
		}
	}
	// plain ASCII
	ascii := true
	for _, c := range v {
		if c != 0 && (c < 0x20 || c > 0x7e) {
			ascii = false
			break
		}
	}
	if ascii {
		return fmt.Sprintf("%q", trimSp(string(bytesNoNul(v))))
	}
	if len(v) > 12 {
		return "0x" + hex.EncodeToString(v[:12]) + "…"
	}
	return "0x" + hex.EncodeToString(v)
}

func trimSp(s string) string {
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}

func bytesNoNul(v []byte) []byte {
	out := make([]byte, 0, len(v))
	for _, c := range v {
		if c != 0 {
			out = append(out, c)
		}
	}
	return out
}
