// Package bigint implements utilities for dealing with arbitrary
// precision integers.
//
package bigint // import "github.com/andrewarchi/nebula/internal/bigint"

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Map is a hash table for big int keys.
type Map struct {
	m   map[int64][]MapPair
	len uint
}

// MapPair is a key-value pair.
type MapPair struct {
	K *big.Int
	V interface{}
}

// NewMap constructs a Map.
func NewMap() *Map {
	return &Map{make(map[int64][]MapPair), 0}
}

// Get the value at the key.
func (m *Map) Get(k *big.Int) (interface{}, bool) {
	for _, e := range m.m[k.Int64()] {
		if e.K.Cmp(k) == 0 {
			return e.V, true
		}
	}
	return nil, false
}

// Has returns whether the key exists.
func (m *Map) Has(k *big.Int) bool {
	_, ok := m.Get(k)
	return ok
}

// Put a value at the key.
func (m *Map) Put(k *big.Int, v interface{}) bool {
	hash := k.Int64()
	bucket := m.m[hash]
	for _, e := range bucket {
		if e.K.Cmp(k) == 0 {
			e.V = v
			return true
		}
	}
	m.m[hash] = append(bucket, MapPair{new(big.Int).Set(k), v})
	m.len++
	return false
}

// Pairs returns a sorted slice of key-value pairs in the map.
func (m *Map) Pairs() []MapPair {
	pairs := make([]MapPair, m.len)
	i := 0
	for _, bucket := range m.m {
		for _, pair := range bucket {
			pairs[i] = pair
			i++
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].K.Cmp(pairs[j].K) < 0
	})
	return pairs
}

// Len returns the number of elements in the map.
func (m Map) Len() int {
	return int(m.len)
}

func (m Map) String() string {
	var b strings.Builder
	b.WriteRune('{')
	for i, pair := range m.Pairs() {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%s:%v", pair.K, pair.V))
	}
	b.WriteRune('}')
	return b.String()
}
