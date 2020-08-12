// Package bitset implements a bit array for dense indexes.
//
package bitset // import "github.com/andrewarchi/nebula/internal/bitset"

const uintSize = 32 << (^uint(0) >> 32 & 1) // 32 or 64

// Bitset is a bit array for dense indexes.
type Bitset []uint

// NewBitset constructs a Bitset.
func NewBitset(n int) Bitset {
	return make(Bitset, (n+uintSize-1)/uintSize)
}

// Reset clears the bitset.
func (bs Bitset) Reset() {
	for i := range bs {
		bs[i] = 0
	}
}

// Set sets the bit at index i.
func (bs Bitset) Set(i int) {
	bs[i/uintSize] |= 1 << (uint(i) % uintSize)
}

// Clear clears bit at index i.
func (bs Bitset) Clear(i int) {
	bs[i/uintSize] &^= 1 << (uint(i) % uintSize)
}

// Test tests bit at index i.
func (bs Bitset) Test(i int) bool {
	return bs[i/uintSize]&(1<<(uint(i)%uintSize)) != 0
}
