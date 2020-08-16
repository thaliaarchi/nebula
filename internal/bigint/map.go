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

// Map is a hash table for big int keys. Keys are not copied and must
// not be changed after insertion.
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
func (m *Map) Get(key *big.Int) (interface{}, bool) {
	pair, ok := m.GetPair(key)
	return pair.V, ok
}

// GetPair gets the key-value pair at the key.
func (m *Map) GetPair(key *big.Int) (MapPair, bool) {
	for _, pair := range m.m[key.Int64()] {
		if pair.K.Cmp(key) == 0 {
			return pair, true
		}
	}
	return MapPair{}, false
}

// Has returns whether the key exists.
func (m *Map) Has(key *big.Int) bool {
	_, ok := m.GetPair(key)
	return ok
}

// Put a value at the key.
func (m *Map) Put(key *big.Int, v interface{}) bool {
	hash := key.Int64()
	bucket := m.m[hash]
	for _, pair := range bucket {
		if pair.K.Cmp(key) == 0 {
			pair.V = v
			return true
		}
	}
	m.m[hash] = append(bucket, MapPair{key, v}) // key not copied
	m.len++
	return false
}

// GetOrPut gets the value at the key, if it exists, and puts a value
// at the key otherwise.
func (m *Map) GetOrPut(key *big.Int, v interface{}) (interface{}, bool) {
	hash := key.Int64()
	bucket := m.m[hash]
	for _, pair := range bucket {
		if pair.K.Cmp(key) == 0 {
			return pair.V, true
		}
	}
	m.m[hash] = append(bucket, MapPair{key, v}) // key not copied
	m.len++
	return v, false
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
