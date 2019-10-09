package ast

import (
	"fmt"
	"math/big"
	"strings"
)

// Stack represents the Whitespace stack for registerization. Every
// value is given a unique id. Values from outside the current basic
// block are represented as negative numbers. Operations are expressed
// in terms of push and pop.
type Stack struct {
	Vals   []*Val
	Next   int // Next id to push
	Pops   int // Number of items popped below stack
	Access int // Number of items accessed below stack
}

// Push pushes an item to the stack and returns a val with a unique id.
func (s *Stack) Push() *Val {
	val := Val(&StackVal{s.Next})
	s.Vals = append(s.Vals, &val)
	s.Next++
	return &val
}

// PushConst pushes a constant value to the stack and returns its val.
func (s *Stack) PushConst(c *big.Int) *Val {
	val := Val(&ConstVal{c})
	s.Vals = append(s.Vals, &val)
	return &val
}

// Pop pops an item from the stack and returns the id of the removed
// item.
func (s *Stack) Pop() *Val {
	var val *Val
	if len(s.Vals) == 0 {
		s.Pops++
		v := Val(&StackVal{-s.Pops})
		val = &v
		if s.Pops > s.Access {
			s.Access = s.Pops
		}
	} else {
		val = s.Vals[len(s.Vals)-1]
		s.Vals = s.Vals[:len(s.Vals)-1]
	}
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

// Dup pushes a copy of the top item to the stack without creating an
// id.
func (s *Stack) Dup() *Val {
	top := s.Top()
	s.Vals = append(s.Vals, top)
	return top
}

// Copy pushes a copy of the nth item to the stack without creating an
// id.
func (s *Stack) Copy(n int) *Val {
	if n < 0 {
		panic(fmt.Sprintf("stack: copy index must be positive: %d", n))
	}
	val := s.Nth(n)
	s.Vals = append(s.Vals, val)
	return val
}

// Swap swaps the top two items on the stack.
func (s *Stack) Swap() {
	l := len(s.Vals)
	switch l {
	case 0:
		v1, v2 := Val(&StackVal{-s.Pops - 1}), Val(&StackVal{-s.Pops - 2})
		s.Vals = append(s.Vals, &v1, &v2)
		s.Pops += 2
	case 1:
		v := Val(&StackVal{-s.Pops - 1})
		s.Vals = append(s.Vals, &v)
		s.Pops++
	default:
		s.Vals[l-2], s.Vals[l-1] = s.Vals[l-1], s.Vals[l-2]
	}
	if s.Pops > s.Access {
		s.Access = s.Pops
	}
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

// Top returns the id of the top item on the stack.
func (s *Stack) Top() *Val {
	if len(s.Vals) != 0 {
		return s.Vals[len(s.Vals)-1]
	}
	top := s.Pops + 1
	if top > s.Access {
		s.Access = top
	}
	val := Val(&StackVal{-top})
	return &val
}

// Nth returns the id of the nth item on the stack.
func (s *Stack) Nth(n int) *Val {
	if n < len(s.Vals) {
		return s.Vals[len(s.Vals)-n-1]
	}
	val := s.Pops + n + 1 - len(s.Vals)
	if val > s.Access {
		s.Access = val
	}
	v := Val(&StackVal{-val})
	return &v
}

// Len returns the number of items on the stack.
func (s *Stack) Len() int {
	return len(s.Vals)
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

func (s *Stack) String() string {
	var b strings.Builder
	b.WriteByte('[')
	for i, val := range s.Vals {
		if i != 0 {
			b.WriteByte(' ')
		}
		b.WriteString((*val).String())
	}
	b.WriteByte(']')
	return b.String()
}
