package ir

import (
	"fmt"
	"go/token"
)

// Stack models a stack frame for converting Whitespace stack-oriented
// operations to SSA form. When accessing a position under the stack
// frame, the load handler is invoked to provide that value.
type Stack struct {
	values       []Value       // Values pushed in the stack frame
	under        []Value       // Values accessed under the stack frame
	pops         int           // Number of values popped under stack frame
	accesses     int           // Lowest position accessed under stack frame
	HandleAccess AccessHandler // Executed on access
	HandleLoad   LoadHandler   // Executed on load
}

// AccessHandler watches accesses of values under stack frame.
type AccessHandler func(n int, pos token.Pos)

// LoadHandler watches loads of values under stack frame.
type LoadHandler func(n int, pos token.Pos) Value

// Push pushes a value to the top of the stack.
func (s *Stack) Push(val Value) {
	s.values = append(s.values, val)
}

// Pop pops the top value from the stack and returns the removed value.
func (s *Stack) Pop(pos token.Pos) Value {
	val := s.Top(pos)
	s.Drop(pos)
	return val
}

// Pop2 pops the top two values form the stack and returns the removed
// values.
func (s *Stack) Pop2(pos token.Pos) (Value, Value) {
	// access low value first to avoid multiple accesses
	val1, val0 := s.At(1, pos), s.At(0, pos)
	s.DropN(2, pos)
	return val1, val0
}

// Drop discards the top value on the stack without accessing it.
func (s *Stack) Drop(pos token.Pos) {
	if len(s.values) == 0 {
		s.pops++
		s.Access(s.pops, pos)
	} else {
		s.values = s.values[:len(s.values)-1]
	}
}

// DropN discards the top n values on the stack without accessing them.
func (s *Stack) DropN(n int, pos token.Pos) {
	l := len(s.values)
	switch {
	case n < 0:
		panic(fmt.Sprintf("stack: drop count must be positive: %d", n))
	case n == 0:
		return
	case l == 0:
		s.pops += n
	case n >= l:
		s.values = s.values[:0]
		s.pops += n - l
	default:
		s.values = s.values[:l-n]
	}
	s.Access(s.pops, pos)
}

// Dup copies the top value and pushes it to the stack.
func (s *Stack) Dup(pos token.Pos) Value {
	top := s.Top(pos)
	s.values = append(s.values, top)
	return top
}

// Copy copies the nth value and pushes it to the stack.
func (s *Stack) Copy(n int, pos token.Pos) Value {
	if n < 0 {
		panic(fmt.Sprintf("stack: copy index must be positive: %d", n))
	}
	val := s.At(n, pos)
	s.values = append(s.values, val)
	return val
}

// Swap swaps the top two values on the stack.
func (s *Stack) Swap(pos token.Pos) {
	val1, val0 := s.Pop2(pos)
	s.values = append(s.values, val0, val1)
	s.simplify()
}

// Slide discards n values on the stack, leaving the top value.
func (s *Stack) Slide(n int, pos token.Pos) {
	if n < 0 {
		panic(fmt.Sprintf("stack: slide count must be positive: %d", n))
	}
	if n == 0 {
		return
	}
	top := s.Top(pos)
	s.DropN(n+1, pos)
	s.values = append(s.values, top)
	s.simplify()
}

// Top accesses and returns the top value on the stack.
func (s *Stack) Top(pos token.Pos) Value {
	return s.At(0, pos)
}

// At accesses and returns the nth value on the stack.
func (s *Stack) At(n int, pos token.Pos) Value {
	if n < len(s.values) {
		return s.values[len(s.values)-n-1]
	}
	access := s.pops + n + 1 - len(s.values)
	s.Access(access, pos)
	if access > len(s.under) {
		s.under = append(s.under, make([]Value, access-len(s.under))...)
	}
	if s.under[access-1] == nil {
		s.under[access-1] = s.HandleLoad(access, pos)
	}
	return s.under[access-1]
}

// Get returns the nth value on the stack, if it has already been
// accessed.
func (s *Stack) Get(n int) (Value, bool) {
	var val Value
	if n < len(s.values) {
		val = s.values[len(s.values)-n-1]
	} else if n < len(s.under)-len(s.values) {
		val = s.under[len(s.under)-len(s.values)-n-1]
	}
	if val != nil {
		return val, true
	}
	return nil, false
}

// Access accesses the nth position under the stack frame.
func (s *Stack) Access(n int, pos token.Pos) {
	if n > s.accesses {
		s.accesses = n
		s.HandleAccess(n, pos)
	}
}

// simplify cleans up low elements.
func (s *Stack) simplify() {
	i := 0
	for i < len(s.values) && i < s.pops &&
		s.pops-i-1 < len(s.under) && s.values[i] == s.under[s.pops-i-1] {
		i++
	}
	s.values = s.values[i:]
	s.pops -= i
}

// Clear resets the stack.
func (s *Stack) Clear() {
	s.values = s.values[:0]
	s.under = s.under[:0]
	s.pops = 0
	s.accesses = 0
}

// Pops returns the numbers of values popped under stack frame.
func (s *Stack) Pops() int {
	return s.pops
}

// Accesses returns the lowest position accessed under stack frame.
func (s *Stack) Accesses() int {
	return s.accesses
}

// Values returns all values in the stack frame.
func (s *Stack) Values() []Value {
	return s.values
}

// Len returns the number of values on the stack.
func (s *Stack) Len() int {
	return len(s.values)
}

func (s *Stack) String() string {
	return NewFormatter().FormatStack(s)
}
