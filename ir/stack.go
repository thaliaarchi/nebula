package ir

import "go/token"

// Stack models a stack frame for converting Whitespace stack-oriented
// operations to SSA form. When accessing a position under the stack
// frame, the load handler is invoked to provide that value.
type Stack struct {
	values       []Value       // Values pushed in the stack frame
	under        []Value       // Values accessed under the stack frame
	pops         uint          // Number of values popped under stack frame
	accesses     uint          // Lowest position accessed under stack frame
	HandleAccess AccessHandler // Executed on access
	HandleLoad   LoadHandler   // Executed on load
}

// AccessHandler watches accesses of values under stack frame.
type AccessHandler func(n uint, pos token.Pos)

// LoadHandler watches loads of values under stack frame.
type LoadHandler func(n uint, pos token.Pos) (load Value)

// Push pushes a value to the top of the stack.
func (s *Stack) Push(val Value) {
	s.values = append(s.values, val)
}

// Pop pops the top value from the stack and returns the removed value.
func (s *Stack) Pop(pos token.Pos) (top Value) {
	top = s.Top(pos)
	s.Drop(pos)
	return top
}

// Pop2 pops the top two values form the stack and returns the removed
// values.
func (s *Stack) Pop2(pos token.Pos) (val1, val0 Value) {
	// access low value first to avoid multiple accesses
	val1, val0 = s.At(1, pos), s.At(0, pos)
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
func (s *Stack) DropN(n uint, pos token.Pos) {
	l := uint(len(s.values))
	switch {
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
func (s *Stack) Dup(pos token.Pos) (top Value) {
	top = s.Top(pos)
	s.values = append(s.values, top)
	return top
}

// Copy copies the nth value and pushes it to the stack.
func (s *Stack) Copy(n uint, pos token.Pos) (nth Value) {
	nth = s.At(n, pos)
	s.values = append(s.values, nth)
	return nth
}

// Swap swaps the top two values on the stack.
func (s *Stack) Swap(pos token.Pos) {
	val1, val0 := s.Pop2(pos)
	s.values = append(s.values, val0, val1)
	s.simplify()
}

// Slide discards n values on the stack, leaving the top value.
func (s *Stack) Slide(n uint, pos token.Pos) {
	if n == 0 {
		return
	}
	top := s.Top(pos)
	s.DropN(n+1, pos)
	s.values = append(s.values, top)
	s.simplify()
}

// Top accesses and returns the top value on the stack.
func (s *Stack) Top(pos token.Pos) (top Value) {
	return s.At(0, pos)
}

// At accesses and returns the nth value on the stack.
func (s *Stack) At(n uint, pos token.Pos) (nth Value) {
	vals, under := uint(len(s.values)), uint(len(s.under))
	if n < vals {
		return s.values[vals-n-1]
	}
	u := s.pops + n - vals
	s.Access(u+1, pos)
	if u >= under {
		s.under = append(s.under, make([]Value, u-under+1)...)
	}
	if s.under[u] == nil {
		s.under[u] = s.HandleLoad(u+1, pos)
	}
	return s.under[u]
}

// Get returns the nth value on the stack, if it has already been
// accessed.
func (s *Stack) Get(n uint) (nth Value, ok bool) {
	vals := uint(len(s.values))
	if n < vals {
		return s.values[vals-n-1], true
	}
	u := s.pops + n - vals
	if u < uint(len(s.under)) {
		return s.under[u], s.under[u] != nil
	}
	return nil, false
}

// Access accesses the nth position under the stack frame.
func (s *Stack) Access(n uint, pos token.Pos) {
	if n > s.accesses {
		s.accesses = n
		s.HandleAccess(n, pos)
	}
}

// simplify cleans up low elements where operations result in an
// identity.
func (s *Stack) simplify() {
	var i uint
	for i < s.pops && i < uint(len(s.values)) && s.pops-i-1 < uint(len(s.under)) {
		if s.values[i] != s.under[s.pops-i-1] {
			break
		}
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
func (s *Stack) Pops() uint {
	return s.pops
}

// Accesses returns the lowest position accessed under stack frame.
func (s *Stack) Accesses() uint {
	return s.accesses
}

// Values returns all values in the stack frame.
func (s *Stack) Values() []Value {
	return s.values
}

// Len returns the number of values on the stack.
func (s *Stack) Len() uint {
	return uint(len(s.values))
}

func (s *Stack) String() string {
	return NewFormatter().FormatStack(s)
}
