// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"sync"
	"testing"
)

// TestTFParserConcurrentMarshalUnmarshal is a smoke test that exercises
// TFParser under heavy concurrency. The production bug it guards against
// is https://github.com/json-iterator/go/issues/682 (and its older
// sibling #618): jsoniter's internal decoder/encoder cache uses a plain
// Go map that can race under concurrent first-touches of a new type,
// surfacing as `fatal error: concurrent map writes` or `assignment to
// entry in nil map` from deep inside `reflect_map.go` /
// `modern-go/reflect2`.
//
// Note: reproducing the race reliably in a unit test is hard — it
// depends on fine-grained goroutine interleaving around cache
// population. This test may pass even without the lockedAPI wrapper on
// some hosts. Its primary value is as a regression guard against future
// changes that might remove the locking, and as a basic smoke check
// that the wrapper preserves correctness under concurrent use.
func TestTFParserConcurrentMarshalUnmarshal(t *testing.T) {
	const (
		goroutines = 128
		iterations = 500
	)

	// Varied payloads force jsoniter to resolve decoders for multiple
	// concrete types (int, float64, string, bool, nested map, nested
	// slice, nil) on each round.
	payloads := []any{
		map[string]any{"id": "x", "tags": map[string]any{"k1": "v1", "k2": "v2"}},
		map[string]any{"id": 42, "tags": []any{"a", "b", "c"}},
		map[string]any{"id": 3.14, "tags": nil},
		map[string]any{"id": true, "nested": map[string]any{"deep": map[string]any{"k": 1}}},
		map[string]any{"id": "y", "list": []any{map[string]any{"inner": "v"}}},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				src := payloads[(id+i)%len(payloads)]
				buf, err := TFParser.Marshal(src)
				if err != nil {
					t.Errorf("Marshal failed: %v", err)
					return
				}
				dst := map[string]any{}
				if err := TFParser.Unmarshal(buf, &dst); err != nil {
					t.Errorf("Unmarshal failed: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}

// TestJSParserConcurrentMarshalUnmarshal is the JSParser equivalent of
// TestTFParserConcurrentMarshalUnmarshal — JSParser is used by the
// Terraform-state read/write paths in upjet and is reached on every
// reconcile for resources managed via the internal state cache.
func TestJSParserConcurrentMarshalUnmarshal(t *testing.T) {
	type state struct {
		Version    int               `json:"version"`
		Attributes map[string]string `json:"attributes"`
	}

	const (
		goroutines = 64
		iterations = 200
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				src := state{
					Version: 4,
					Attributes: map[string]string{
						"id":   "x",
						"arn":  "arn:aws:...",
						"tags": "{}",
					},
				}
				buf, err := JSParser.Marshal(src)
				if err != nil {
					t.Errorf("Marshal failed: %v", err)
					return
				}
				var dst state
				if err := JSParser.Unmarshal(buf, &dst); err != nil {
					t.Errorf("Unmarshal failed: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}
