package ws

import "math/big"

type intMap struct {
	m map[int64][]pair
}

type pair struct {
	k *big.Int
	v interface{}
}

func newIntMap() *intMap {
	return &intMap{make(map[int64][]pair)}
}

func (m *intMap) Get(k *big.Int) (interface{}, bool) {
	hash := k.Int64()
	pairs, ok := m.m[hash]
	if ok {
		for _, p := range pairs {
			if p.k.Cmp(k) == 0 {
				return p.v, true
			}
		}
	}
	return nil, false
}

func (m *intMap) Put(k *big.Int, v interface{}) bool {
	hash := k.Int64()
	pairs, ok := m.m[hash]
	if ok {
		for _, p := range pairs {
			if p.k.Cmp(k) == 0 {
				p.v = v
				return true
			}
		}
	}
	m.m[hash] = append(pairs, pair{k, v})
	return false
}
