package bigint

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Map is a hash table for big int keys.
type Map struct {
	m   map[int64][]MapEntry
	new func() interface{}
}

// MapEntry is a key-value pair.
type MapEntry struct {
	K *big.Int
	V interface{}
}

// NewMap constructs a Map.
func NewMap(new func() interface{}) *Map {
	return &Map{make(map[int64][]MapEntry), new}
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

// Retrieve gets the value at the key and inserts when empty.
func (m *Map) Retrieve(k *big.Int) interface{} {
	hash := k.Int64()
	for _, e := range m.m[hash] {
		if e.K.Cmp(k) == 0 {
			return e.V
		}
	}
	v := m.new()
	m.m[hash] = append(m.m[hash], MapEntry{new(big.Int).Set(k), v})
	return v
}

// Put a value at the key.
func (m *Map) Put(k *big.Int, v interface{}) bool {
	hash := k.Int64()
	entries := m.m[hash]
	for _, e := range entries {
		if e.K.Cmp(k) == 0 {
			e.V = v
			return true
		}
	}
	m.m[hash] = append(entries, MapEntry{new(big.Int).Set(k), v})
	return false
}

// Entries returns a channel to iterate the entries of the map.
func (m *Map) Entries() <-chan MapEntry {
	entryChan := make(chan MapEntry)
	go func() {
		for _, entries := range m.m {
			for _, entry := range entries {
				entryChan <- entry
			}
		}
		close(entryChan)
	}()
	return entryChan
}

// Clear removes all entries from the map.
func (m *Map) Clear() {
	m.m = make(map[int64][]MapEntry)
}

func (m Map) String() string {
	var entries []MapEntry
	for _, bucket := range m.m {
		for _, entry := range bucket {
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].K.Cmp(entries[j].K) < 0
	})
	var b strings.Builder
	b.WriteRune('{')
	for i, entry := range entries {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%s:%v", entry.K, entry.V))
	}
	b.WriteRune('}')
	return b.String()
}
