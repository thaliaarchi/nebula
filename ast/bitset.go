package ast // import "github.com/andrewarchi/nebula/ast"

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

func (bs bitset) Set(i int) {
	bs[i/uintSize] |= 1 << (uint(i) % uintSize)
}

func (bs bitset) Clear(i int) {
	bs[i/uintSize] &^= 1 << (uint(i) % uintSize)
}

func (bs bitset) Test(i int) bool {
	return bs[i/uintSize]&(1<<(uint(i)%uintSize)) != 0
}
