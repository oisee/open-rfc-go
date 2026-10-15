// SPDX-License-Identifier: Apache-2.0

package rfc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/oisee/open-rfc-go/internal/metadata"
	"github.com/oisee/open-rfc-go/internal/rfctypes"
	"github.com/oisee/open-rfc-go/internal/xrfc"
)

// The fixture is a verbatim RFC_METADATA_GET (DEEP='X') response for
// SADT_REST_RFC_ENDPOINT, captured on an S/4HANA system (SAP_BASIS 816). It
// carries both wire forms of numeric cells: JSON numbers in PARAMETERS
// (INTLENGTH, OFFSET, ...) and digit strings in DATATYPESCONT (INTLEN,
// OFFSET, ...).
func sadtCapture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/sadt_rest_rfc_endpoint.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return raw
}

// sadtCapture740 is the same response captured on an ECC system (SAP_BASIS
// 740, kernel 742, Unicode). Its DDIC timestamps differ, and it lacks the
// EX_FUNCTIONS, EX_TYPES and EX_TYPE_ERRORS tables that later releases of
// RFC_METADATA_GET return, empty, alongside the others.
func sadtCapture740(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/sadt_rest_rfc_endpoint_740.json")
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return raw
}

// seedableClient is a client with no pool and no connection: any code path
// that reaches for the network panics on the nil pool, so a passing test is
// proof that the seed alone answered.
func seedableClient() *Client { return &Client{} }

func TestSeedMetadataServesFunctionInterfaceWithoutNetwork(t *testing.T) {
	c := seedableClient()
	if err := c.SeedMetadata(sadtCapture(t)); err != nil {
		t.Fatalf("SeedMetadata: %v", err)
	}

	iface, err := c.FunctionInterface(context.Background(), "SADT_REST_RFC_ENDPOINT")
	if err != nil {
		t.Fatalf("FunctionInterface: %v", err)
	}
	if iface.Name != "SADT_REST_RFC_ENDPOINT" {
		t.Fatalf("Name = %q", iface.Name)
	}
	if !iface.RemoteBasxmlSupported {
		t.Error("BASXML_SUPPORTED = 'X' in the capture must survive normalization")
	}
	if len(iface.Parameters) != 2 {
		t.Fatalf("parameters = %d, want REQUEST + RESPONSE", len(iface.Parameters))
	}
	byName := indexParams(iface)
	req, ok := byName["REQUEST"]
	if !ok || req.ParameterClass != "I" || req.Exid != "v" || req.TableName != "SADT_REST_REQUEST" {
		t.Fatalf("REQUEST = %+v", req)
	}
	res, ok := byName["RESPONSE"]
	if !ok || res.ParameterClass != "E" || res.Exid != "v" || res.TableName != "SADT_REST_RESPONSE" {
		t.Fatalf("RESPONSE = %+v", res)
	}
}

func TestSeedMetadataFillsTheGraphCache(t *testing.T) {
	c := seedableClient()
	if err := c.SeedMetadata(sadtCapture(t)); err != nil {
		t.Fatalf("SeedMetadata: %v", err)
	}
	graph, ok := c.cachedGraph("SADT_REST_RFC_ENDPOINT")
	if !ok {
		t.Fatal("the graph cache must hold the seeded function")
	}
	for _, name := range []string{
		"SADT_REST_REQUEST", "SADT_REST_REQUEST_LINE", "SADT_REST_RESPONSE",
		"SADT_REST_STATUS_LINE", "TIHTTPNVP", "IHTTPNVP",
	} {
		if _, ok := graph.Nodes[name]; !ok {
			t.Errorf("graph lacks node %s", name)
		}
	}
}

// TestSeededCallPlanNeedsNoNetwork walks the exact path Call takes up to
// serialization: the layout plan for both deep parameters must come from the
// seeded graph, with no structure resolution and no metadata fetch. The nil
// session and the panicking resolver make any wire attempt a hard test failure.
func TestSeededCallPlanNeedsNoNetwork(t *testing.T) {
	c := seedableClient()
	if err := c.SeedMetadata(sadtCapture(t)); err != nil {
		t.Fatalf("SeedMetadata: %v", err)
	}
	iface, ok := c.cachedFn("SADT_REST_RFC_ENDPOINT")
	if !ok {
		t.Fatal("the function cache must hold the seeded interface")
	}
	resolve := func(name string) (rfctypes.RfcStructureDefinition, error) {
		panic("planLayoutOn resolved " + name + " instead of using the seed")
	}
	plan, err := c.planLayoutOn(context.Background(), nil, iface, resolve)
	if err != nil {
		t.Fatalf("planLayoutOn: %v", err)
	}
	if plan == nil {
		t.Fatal("the seeded function must plan its deep parameters recursively")
	}
	for _, name := range []string{"REQUEST", "RESPONSE"} {
		if _, ok := plan.lookup(name); !ok {
			t.Errorf("%s is missing from the recursive plan", name)
		}
	}
}

func TestSeedMetadataRejectsAnIncompleteClosure(t *testing.T) {
	var top map[string]any
	if err := json.Unmarshal(sadtCapture(t), &top); err != nil {
		t.Fatal(err)
	}
	rows := top["DATATYPESCONT"].([]any)
	var kept []any
	for _, r := range rows {
		if r.(map[string]any)["TYPENAME"] != "TIHTTPNVP" {
			kept = append(kept, r)
		}
	}
	top["DATATYPESCONT"] = kept
	raw, err := json.Marshal(top)
	if err != nil {
		t.Fatal(err)
	}
	c := seedableClient()
	err = c.SeedMetadata(raw)
	if !errors.Is(err, ErrSeed) {
		t.Fatalf("err = %v, want ErrSeed", err)
	}
	if !strings.Contains(err.Error(), "TIHTTPNVP") {
		t.Fatalf("the error must name the missing type, got: %v", err)
	}
	if _, ok := c.cachedFn("SADT_REST_RFC_ENDPOINT"); ok {
		t.Error("a rejected seed must leave the caches untouched")
	}
}

func TestSeedMetadataRejectsGarbage(t *testing.T) {
	c := seedableClient()
	for _, raw := range []string{"", "[]", `{"FUNCTIONNAMES":[]}`, `{"FUNCTIONNAMES":[{},{}]}`} {
		if err := c.SeedMetadata([]byte(raw)); !errors.Is(err, ErrSeed) {
			t.Errorf("SeedMetadata(%q) = %v, want ErrSeed", raw, err)
		}
	}
}

// TestSeedMetadataAcceptsBothNumericForms flips the PARAMETERS numeric cells
// (JSON numbers in the capture) to digit strings and expects the same seed to
// load: both wire forms occur in real captures.
func TestSeedMetadataAcceptsBothNumericForms(t *testing.T) {
	dec := json.NewDecoder(strings.NewReader(string(sadtCapture(t))))
	dec.UseNumber()
	var top map[string]any
	if err := dec.Decode(&top); err != nil {
		t.Fatal(err)
	}
	for _, r := range top["PARAMETERS"].([]any) {
		row := r.(map[string]any)
		for _, k := range []string{"DECIMALS", "INTLENGTH", "OFFSET", "POSITION"} {
			row[k] = row[k].(json.Number).String()
		}
	}
	raw, err := json.Marshal(top)
	if err != nil {
		t.Fatal(err)
	}
	c := seedableClient()
	if err := c.SeedMetadata(raw); err != nil {
		t.Fatalf("SeedMetadata with stringified PARAMETERS numbers: %v", err)
	}
	if _, ok := c.cachedFn("SADT_REST_RFC_ENDPOINT"); !ok {
		t.Fatal("seed did not land")
	}
}

// TestSeededGraphResolvesBothParameters proves the seeded graph is usable by
// the recursive codec: both deep parameters resolve, and a realistic request
// value encodes.
func TestSeededGraphResolvesBothParameters(t *testing.T) {
	c := seedableClient()
	if err := c.SeedMetadata(sadtCapture(t)); err != nil {
		t.Fatalf("SeedMetadata: %v", err)
	}
	graph, _ := c.cachedGraph("SADT_REST_RFC_ENDPOINT")
	iface, _ := c.cachedFn("SADT_REST_RFC_ENDPOINT")
	byName := indexParams(iface)

	for _, name := range []string{"REQUEST", "RESPONSE"} {
		rp, needed, err := xrfc.ResolveParameter(graph, byName[name])
		if err != nil {
			t.Fatalf("ResolveParameter %s: %v", name, err)
		}
		if !needed {
			t.Fatalf("%s must travel on the recursive codec", name)
		}
		if rp.Kind != xrfc.KindStructure {
			t.Fatalf("%s kind = %v, want structure", name, rp.Kind)
		}
	}

	rp, _, _ := xrfc.ResolveParameter(graph, byName["REQUEST"])
	var in any
	err := json.Unmarshal([]byte(`{
		"REQUEST_LINE": {"METHOD": "GET", "URI": "/sap/bc/adt/discovery", "VERSION": "HTTP/1.1"},
		"HEADER_FIELDS": [{"NAME": "Accept", "VALUE": "application/xml"}],
		"MESSAGE_BODY": ""
	}`), &in)
	if err != nil {
		t.Fatal(err)
	}
	value, err := coerceGraphParam(graph, rp, in)
	if err != nil {
		t.Fatalf("coerceGraphParam: %v", err)
	}
	encoded, err := xrfc.EncodeRecursiveParameter(byName["REQUEST"], graph, value, xrfc.RecursiveLimits{})
	if err != nil {
		t.Fatalf("EncodeRecursiveParameter: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("empty encoding")
	}
}

// TestSeedIsTheSameAcrossReleases pins the claim a seed rests on: the
// SADT_REST_RFC_ENDPOINT contract does not change between releases. A seed
// captured on SAP_BASIS 740 and one captured on 816 must normalize to the same
// interface and the same type graph, so one embedded seed serves both.
func TestSeedIsTheSameAcrossReleases(t *testing.T) {
	c816, c740 := seedableClient(), seedableClient()
	if err := c816.SeedMetadata(sadtCapture(t)); err != nil {
		t.Fatalf("SeedMetadata 816: %v", err)
	}
	if err := c740.SeedMetadata(sadtCapture740(t)); err != nil {
		t.Fatalf("SeedMetadata 740: %v", err)
	}

	f816, _ := c816.cachedFn("SADT_REST_RFC_ENDPOINT")
	f740, _ := c740.cachedFn("SADT_REST_RFC_ENDPOINT")
	if !reflect.DeepEqual(f816, f740) {
		t.Errorf("interfaces differ:\n816: %+v\n740: %+v", f816, f740)
	}
	g816, _ := c816.cachedGraph("SADT_REST_RFC_ENDPOINT")
	g740, _ := c740.cachedGraph("SADT_REST_RFC_ENDPOINT")
	if !reflect.DeepEqual(*g816.FunctionIdentity, *g740.FunctionIdentity) {
		t.Errorf("function identities differ: 816 %+v, 740 %+v", *g816.FunctionIdentity, *g740.FunctionIdentity)
	}
	if l816, l740 := graphLayout(g816), graphLayout(g740); !reflect.DeepEqual(l816, l740) {
		t.Errorf("type graphs differ:\n816: %+v\n740: %+v", l816, l740)
	}
}

// graphLayout is a graph with what legitimately differs between systems
// removed: the DDIC timestamp of each type, which records when a system
// activated it, not what it looks like.
func graphLayout(g metadata.Graph) metadata.Graph {
	g.FunctionIdentity = nil
	nodes := make(map[string]metadata.TypeNode, len(g.Nodes))
	for name, node := range g.Nodes {
		node.Timestamp = ""
		nodes[name] = node
	}
	g.Nodes = nodes
	return g
}
