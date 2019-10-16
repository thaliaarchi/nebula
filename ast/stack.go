package ast // import "github.com/andrewarchi/nebula/ast"

import (
	"fmt"
	"strings"
)

// Stack represents the Whitespace stack for registerization. Every
// value is given a unique id. Values from outside the current basic
// block are represented as negative numbers. Operations are expressed
// in terms of push and pop.
type Stack struct {
	Vals   []*Val // Values in the current stack frame
	Under  []*Val // Values under the current stack frame
	Pops   int    // Number of items popped below stack
	Access int    // Number of items accessed below stack
}

// Push pushes a value to the stack.
func (s *Stack) Push(val *Val) {
	s.Vals = append(s.Vals, val)
}

// Pop pops an item from the stack and returns the val of the removed
// item.
func (s *Stack) Pop() *Val {
	val := s.Top()
	s.Drop()
	return val
}

// PopN pops n items from the stack.
func (s *Stack) PopN(n int) {
	l := len(s.Vals)
	switch {
	case n < 0:
		panic(fmt.Sprintf("stack: pop count must be positive: %d", n))
	case n == 0:
		return
	case l == 0:
		s.Pops += n
	case n >= l:
		s.Vals = s.Vals[:0]
		s.Pops += n - l
	default:
		s.Vals = s.Vals[:l-n]
	}
	if s.Pops > s.Access {
		s.Access = s.Pops
	}
}

// Drop pops the top item from the stack without returning a val.
func (s *Stack) Drop() {
	if len(s.Vals) == 0 {
		s.Pops++
		if s.Pops > s.Access {
			s.Access = s.Pops
		}
	} else {
		s.Vals = s.Vals[:len(s.Vals)-1]
	}
}

// Dup pushes a copy of the top item to the stack.
func (s *Stack) Dup() *Val {
	top := s.Top()
	s.Vals = append(s.Vals, top)
	return top
}

// Copy pushes a copy of the nth item to the stack.
func (s *Stack) Copy(n int) *Val {
	if n < 0 {
		panic(fmt.Sprintf("stack: copy index must be positive: %d", n))
	}
	val := s.At(n)
	s.Vals = append(s.Vals, val)
	return val
}

// Swap swaps the top two items on the stack.
func (s *Stack) Swap() {
	s.Vals = append(s.Vals, s.Pop(), s.Pop())
	s.simplify()
}

// Slide slides n items off the stack, leaving the top item.
func (s *Stack) Slide(n int) {
	if n == 0 {
		return
	}
	top := s.Top()
	s.PopN(n + 1)
	s.Vals = append(s.Vals, top)
	s.simplify()
}

// Top returns the val of the top item on the stack.
func (s *Stack) Top() *Val {
	return s.At(0)
}

// At returns the val of the nth item on the stack.
func (s *Stack) At(n int) *Val {
	if n < len(s.Vals) {
		return s.Vals[len(s.Vals)-n-1]
	}
	id := s.Pops + n + 1 - len(s.Vals)
	if id > s.Access {
		s.Access = id
	}
	if id > len(s.Under) {
		s.Under = append(s.Under, make([]*Val, id-len(s.Under))...)
	}
	if s.Under[id-1] == nil {
		v := Val(&StackVal{-id})
		s.Under[id-1] = &v
	}
	return s.Under[id-1]
}

// simplify cleans up low elements.
func (s *Stack) simplify() {
	i := 0
	for ; i < len(s.Vals); i++ {
		if s.Pops <= 0 {
			break
		}
		if val, ok := (*s.Vals[i]).(*StackVal); !ok || val.Val != -s.Pops {
			break
		}
		s.Pops--
	}
	s.Vals = s.Vals[i:]
}

// Len returns the number of items on the stack.
func (s *Stack) Len() int {
	return len(s.Vals)
}

func (s *Stack) String() string {
	var b strings.Builder
	b.WriteString("push [")
	for i, val := range s.Vals {
		if i != 0 {
			b.WriteByte(' ')
		}
		b.WriteString((*val).String())
	}
	fmt.Fprintf(&b, "], pop %d, access %d [", s.Pops, s.Access)
	first := true
	for _, val := range s.Under {
		if val != nil {
			if !first {
				b.WriteByte(' ')
			}
			b.WriteString((*val).String())
			first = false
		}
	}
	b.WriteByte(']')
	return b.String()
}
