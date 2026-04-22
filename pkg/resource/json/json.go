// SPDX-FileCopyrightText: 2023 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"io"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

// TFParser is a json parser to marshal/unmarshal using "tf" tag.
var TFParser = newLockedAPI(jsoniter.Config{TagKey: "tf"}.Froze())

// JSParser is a json parser to marshal/unmarshal using "json" tag.
var JSParser = newLockedAPI(jsoniter.Config{
	TagKey: "json",
	// We need to sort the map keys to get consistent output in tests.
	SortMapKeys: true,
}.Froze())

// lockedAPI wraps a jsoniter.API and serializes the reflection-heavy
// entry points (Marshal / Unmarshal / MarshalIndent / Marshal*ToString /
// Unmarshal*FromString) behind a sync.Mutex.
//
// json-iterator v1.1.12 (the latest release, from 2021, essentially
// unmaintained) has a known concurrent-map-writes bug in its reflection
// decoder: see https://github.com/json-iterator/go/issues/682. Under load,
// two goroutines calling Unmarshal concurrently can panic with
// `fatal error: concurrent map writes` or `assignment to entry in nil map`
// in `reflect_map.go:191` → `reflect2.UnsafeMapType.UnsafeSetIndex`.
//
// This has been observed in production on Crossplane/upjet providers with
// many managed resources — every reconcile calls TFParser through a
// generated SetObservation / GetParameters, and enough concurrent
// reconciles will eventually hit the race. The fix is to serialize the
// reflection paths until json-iterator is fixed or replaced.
type lockedAPI struct {
	jsoniter.API

	mu sync.Mutex
}

func newLockedAPI(api jsoniter.API) *lockedAPI {
	return &lockedAPI{API: api}
}

func (l *lockedAPI) Marshal(v any) ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.API.Marshal(v)
}

func (l *lockedAPI) MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.API.MarshalIndent(v, prefix, indent)
}

func (l *lockedAPI) MarshalToString(v any) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.API.MarshalToString(v)
}

func (l *lockedAPI) Unmarshal(data []byte, v any) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.API.Unmarshal(data, v)
}

func (l *lockedAPI) UnmarshalFromString(str string, v any) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.API.UnmarshalFromString(str, v)
}

func (l *lockedAPI) NewEncoder(w io.Writer) *jsoniter.Encoder {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.API.NewEncoder(w)
}

func (l *lockedAPI) NewDecoder(r io.Reader) *jsoniter.Decoder {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.API.NewDecoder(r)
}
