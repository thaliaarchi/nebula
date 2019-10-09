package ast

const uintSize = 32 << (^uint(0) >> 32 & 1) // 32 or 64

// bitset is a bit array for dense indexes.
type bitset []uint

func newBitset(n int) bitset {
	return make(bitset, (n+uintSize-1)/uintSize)
}

func (bs bitset) Reset() {
	for i := range bs {
		bs[i] = 0
	}
}

func (bs bitset) Set(i uint32) {
	bs[i/uintSize] |= 1 << (i % uintSize)
}

func (bs bitset) Clear(i uint32) {
	bs[i/uintSize] &^= 1 << (i % uintSize)
}

func (bs bitset) Test(i uint32) bool {
	return bs[i/uintSize]&(1<<(i%uintSize)) != 0
}
