package ast

import "fmt"

// Stack represents the Whitespace stack for registerization. Every
// value is given a unique id. Values from outside the current basic
// block are represented as negative numbers. Operations are expressed
// in terms of push and pop.
type Stack struct {
	Vals []int
	Next int // Next id to push
	Low  int // Lowest value popped below stack
	Min  int // Lowest value accessed below stack
}

// Push pushes an item to the stack and returns the id of the inserted
// item.
func (s *Stack) Push() int {
	n := s.Next
	s.Vals = append(s.Vals, s.Next)
	s.Next++
	return n
}

// Pop pops an item from the stack and returns the id of the removed
// item.
func (s *Stack) Pop() int {
	var val int
	if len(s.Vals) == 0 {
		s.Low--
		val = s.Low
		if s.Low < s.Min {
			s.Min = s.Low
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
		s.Low -= n
	case n >= l:
		s.Vals = s.Vals[:0]
		s.Low -= n - l
	default:
		s.Vals = s.Vals[:l-n]
	}
	if s.Low < s.Min {
		s.Min = s.Low
	}
}

// Dup pushes a copy of the top item to the stack without creating an
// id.
func (s *Stack) Dup() int {
	top := s.Top()
	s.Vals = append(s.Vals, top)
	return top
}

// Copy pushes a copy of the nth item to the stack without creating an
// id.
func (s *Stack) Copy(n int) int {
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
		s.Vals = append(s.Vals, s.Low-1, s.Low-2)
		s.Low -= 2
	case 1:
		s.Vals = append(s.Vals, s.Low-1)
		s.Low--
	default:
		s.Vals[l-2], s.Vals[l-1] = s.Vals[l-1], s.Vals[l-2]
	}
	if s.Low < s.Min {
		s.Min = s.Low
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
func (s *Stack) Top() int {
	if len(s.Vals) != 0 {
		return s.Vals[len(s.Vals)-1]
	}
	top := s.Low - 1
	if top < s.Min {
		s.Min = top
	}
	return top
}

// Nth returns the id of the nth item on the stack.
func (s *Stack) Nth(n int) int {
	if n < len(s.Vals) {
		return s.Vals[len(s.Vals)-n-1]
	}
	val := s.Low - n + len(s.Vals)
	if val < s.Min {
		s.Min = val
	}
	return val
}

// simplify cleans up low elements.
func (s *Stack) simplify() {
	i := 0
	for ; i < len(s.Vals); i++ {
		if s.Low >= 0 || s.Vals[i] != s.Low {
			break
		}
		s.Low++
	}
	s.Vals = s.Vals[i:]
}
