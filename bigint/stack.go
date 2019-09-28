package bigint

import (
	"math/big"
	"strings"
)

// Stack is a container for big ints that allocates items as needed.
type Stack struct {
	items []*big.Int
	len   int
	cap   int // number of non-nil items
}

// NewStack constructs a Stack.
func NewStack() *Stack {
	return &Stack{nil, 0, 0}
}

// Push pushes an item onto the stack. The item given is not modified.
func (s *Stack) Push(x *big.Int) {
	if s.len >= s.cap {
		s.items = append(s.items, new(big.Int).Set(x))
		s.cap++
	} else {
		s.items[s.len].Set(x)
	}
	s.len++
}

// Pop pops and gets the top item on the stack.
func (s *Stack) Pop() *big.Int {
	s.checkUnderflow(1)
	s.len--
	return s.items[s.len]
}

// Top gets the top item on the stack.
func (s *Stack) Top() *big.Int {
	s.checkUnderflow(1)
	return s.items[s.len-1]
}

// Get returns the nth item on the stack. The top is n == 0.
func (s *Stack) Get(n int) *big.Int {
	s.checkUnderflow(n + 1)
	return s.items[s.len-n-1]
}

// Swap swaps the top two items on the stack.
func (s *Stack) Swap() {
	s.checkUnderflow(2)
	s.items[s.len-1], s.items[s.len-2] = s.items[s.len-2], s.items[s.len-1]
}

// PopN pops n items off the stack.
func (s *Stack) PopN(n int) {
	s.checkUnderflow(n)
	s.len -= n
}

// Slide slides the top n items off the stack, keeping the top item.
func (s *Stack) Slide(n int) {
	s.checkUnderflow(n + 1)
	i := s.len - 1
	j := s.len - n - 1
	s.items[i], s.items[j] = s.items[j], s.items[i]
	s.len -= n
}

// Clear sets the length of the stack to zero.
func (s *Stack) Clear() {
	s.len = 0
}

func (s *Stack) String() string {
	var b strings.Builder
	b.WriteRune('[')
	for i := 0; i < s.len; i++ {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(s.items[i].String())
	}
	b.WriteRune(']')
	return b.String()
}

func (s *Stack) checkUnderflow(n int) {
	if s.len < n {
		panic("Stack underflow")
	}
}
