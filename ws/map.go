package ws

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Map is a hash table for big int keys.
type Map struct {
	m   map[int64][]mapEntry
	new func() interface{}
}

type mapEntry struct {
	k *big.Int
	v interface{}
}

// NewMap constructs a Map.
func NewMap(new func() interface{}) *Map {
	return &Map{make(map[int64][]mapEntry), new}
}

// Get the value at the key.
func (m *Map) Get(k *big.Int) (interface{}, bool) {
	for _, e := range m.m[k.Int64()] {
		if e.k.Cmp(k) == 0 {
			return e.v, true
		}
	}
	return nil, false
}

// Retrieve gets the value at the key and inserts when empty.
func (m *Map) Retrieve(k *big.Int) interface{} {
	hash := k.Int64()
	for _, e := range m.m[hash] {
		if e.k.Cmp(k) == 0 {
			return e.v.(*big.Int)
		}
	}
	v := m.new()
	m.m[hash] = append(m.m[hash], mapEntry{k, v})
	return v
}

// Put a value at the key.
func (m *Map) Put(k *big.Int, v interface{}) bool {
	hash := k.Int64()
	entries := m.m[hash]
	for _, e := range entries {
		if e.k.Cmp(k) == 0 {
			e.v = v
			return true
		}
	}
	m.m[hash] = append(entries, mapEntry{k, v})
	return false
}

func (m Map) String() string {
	var entries []mapEntry
	for _, bucket := range m.m {
		for _, e := range bucket {
			entries = append(entries, e)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].k.Cmp(entries[j].k) < 0
	})
	var b strings.Builder
	b.WriteRune('{')
	for i, e := range entries {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%d: %s", e.k, e.v))
	}
	b.WriteRune('}')
	return b.String()
}
