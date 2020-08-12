package ir

import (
	"fmt"
)

// Stack models a stack frame for converting Whitespace stack-oriented
// operations to SSA form. When accessing a position under the stack
// frame, the load handler is invoked to provide that value.
type Stack struct {
	Values     []Value // Values pushed in the stack frame
	Under      []Value // Values accessed under the stack frame
	Pops       int     // Number of values popped under stack frame
	Access     int     // Lowest position accessed under stack frame
	HandleLoad LoadHandler
}

// LoadHandler is a handler registered to watch loads of values under
// the current stack frame.
type LoadHandler func(pos int) Value

// Push pushes a value to the top of the stack.
func (s *Stack) Push(val Value) {
	s.Values = append(s.Values, val)
}

// Pop pops the top value from the stack and returns the removed value.
func (s *Stack) Pop() Value {
	val := s.Top()
	s.Drop()
	return val
}

// Drop discards the top value on the stack without accessing it.
func (s *Stack) Drop() {
	if len(s.Values) == 0 {
		s.Pops++
		if s.Pops > s.Access {
			s.Access = s.Pops
		}
	} else {
		s.Values = s.Values[:len(s.Values)-1]
	}
}

// DropN discards the top n values on the stack without accessing them.
func (s *Stack) DropN(n int) {
	l := len(s.Values)
	switch {
	case n < 0:
		panic(fmt.Sprintf("stack: drop count must be positive: %d", n))
	case n == 0:
		return
	case l == 0:
		s.Pops += n
	case n >= l:
		s.Values = s.Values[:0]
		s.Pops += n - l
	default:
		s.Values = s.Values[:l-n]
	}
	if s.Pops > s.Access {
		s.Access = s.Pops
	}
}

// Dup copies the top value and pushes it to the stack.
func (s *Stack) Dup() Value {
	top := s.Top()
	s.Values = append(s.Values, top)
	return top
}

// Copy copies the nth value and pushes it to the stack.
func (s *Stack) Copy(n int) Value {
	if n < 0 {
		panic(fmt.Sprintf("stack: copy index must be positive: %d", n))
	}
	val := s.At(n)
	s.Values = append(s.Values, val)
	return val
}

// Swap swaps the top two values on the stack.
func (s *Stack) Swap() {
	s.Values = append(s.Values, s.Pop(), s.Pop())
	s.simplify()
}

// Slide discards n values on the stack, leaving the top value.
func (s *Stack) Slide(n int) {
	if n < 0 {
		panic(fmt.Sprintf("stack: slide count must be positive: %d", n))
	}
	if n == 0 {
		return
	}
	top := s.Top()
	s.DropN(n + 1)
	s.Values = append(s.Values, top)
	s.simplify()
}

// Top accesses and returns the top value on the stack.
func (s *Stack) Top() Value {
	return s.At(0)
}

// At accesses and returns the nth value on the stack.
func (s *Stack) At(n int) Value {
	if n < len(s.Values) {
		return s.Values[len(s.Values)-n-1]
	}
	pos := s.Pops + n + 1 - len(s.Values)
	if pos > s.Access {
		s.Access = pos
	}
	if pos > len(s.Under) {
		s.Under = append(s.Under, make([]Value, pos-len(s.Under))...)
	}
	if s.Under[pos-1] == nil {
		s.Under[pos-1] = s.HandleLoad(pos)
	}
	return s.Under[pos-1]
}

// AtExists returns the nth value on the stack, if it has already been
// accessed.
func (s *Stack) AtExists(n int) (Value, bool) {
	var val Value
	if n < len(s.Values) {
		val = s.Values[len(s.Values)-n-1]
	} else if n < len(s.Under)-len(s.Values) {
		val = s.Under[len(s.Under)-len(s.Values)-n-1]
	}
	if val != nil {
		return val, true
	}
	return nil, false
}

// simplify cleans up low elements.
func (s *Stack) simplify() {
	i := 0
	for i < len(s.Values) && i < s.Pops &&
		s.Pops-i-1 < len(s.Under) && s.Values[i] == s.Under[s.Pops-i-1] {
		i++
	}
	s.Values = s.Values[i:]
	s.Pops -= i
}

// Len returns the number of values on the stack.
func (s *Stack) Len() int {
	return len(s.Values)
}

func (s *Stack) String() string {
	return NewFormatter().FormatStack(s)
}
