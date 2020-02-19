package ir // import "github.com/andrewarchi/nebula/ir"

import (
	"fmt"
	"strings"
)

// Stack represents the Whitespace stack for registerization. Every
// value is given a unique id. Values from outside the current basic
// block are represented as negative numbers. Operations are expressed
// in terms of push and pop.
type Stack struct {
	Vals        []Value // Values in the current stack frame
	Under       []Value // Values under the current stack frame
	Pops        int     // Number of items popped below current stack frame
	Access      int     // Number of items accessed below current stack frame
	LoadHandler LoadHandler
}

// LoadHandler is a handler registered to watch loads of values under
// the current stack frame.
type LoadHandler func(load Node)

// Push pushes a value to the stack.
func (s *Stack) Push(val Value) {
	s.Vals = append(s.Vals, val)
}

// Pop pops an item from the stack and returns the val of the removed
// item.
func (s *Stack) Pop() Value {
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
func (s *Stack) Dup() Value {
	top := s.Top()
	s.Vals = append(s.Vals, top)
	return top
}

// Copy pushes a copy of the nth item to the stack.
func (s *Stack) Copy(n int) Value {
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
func (s *Stack) Top() Value {
	return s.At(0)
}

// At returns the val of the nth item on the stack.
func (s *Stack) At(n int) Value {
	if n < len(s.Vals) {
		return s.Vals[len(s.Vals)-n-1]
	}
	id := s.Pops + n + 1 - len(s.Vals)
	if id > s.Access {
		s.Access = id
	}
	if id > len(s.Under) {
		s.Under = append(s.Under, make([]Value, id-len(s.Under))...)
	}
	if s.Under[id-1] == nil {
		load := &LoadStackExpr{Def: &ValueDef{}, Pos: id}
		if s.LoadHandler != nil {
			s.LoadHandler(load)
		}
		s.Under[id-1] = load
	}
	return s.Under[id-1]
}

// AtExists returns the val of the nth item on the stack or false if
// the lookup would cause an underflow access.
func (s *Stack) AtExists(n int) (Value, bool) {
	var val Value
	if n < len(s.Vals) {
		val = s.Vals[len(s.Vals)-n-1]
	} else if n < len(s.Under)-len(s.Vals) {
		val = s.Under[len(s.Under)-len(s.Vals)-n-1]
	}
	if val != nil {
		return val, true
	}
	return nil, false
}

// LookupUnderflow replaces vals referencing the preceding stack frame
// with the defined val.
// TODO update to new IR structure
/*
func (s *Stack) LookupUnderflow(prev *Stack) {
	for _, val := range s.Under {
		if val != nil {
			if v, ok := (*val).(*StackVal); ok {
				if pv, ok := prev.AtExists(-v.Pos - 1); ok {
					*val = *pv
				}
			}
		}
	}
}

// Concat joins two stacks.
// TODO update to new IR structure
func (s *Stack) Concat(next *Stack) {
	for _, val := range next.Under {
		if val != nil {
			if v, ok := (*val).(*StackVal); ok {
				*val = *s.At(-v.Pos - 1)
			}
		}
	}
	if next.Access > 0 {
		id := s.Pops + next.Access - len(s.Vals)
		if id > s.Access {
			s.Access = id
		}
	}
	s.PopN(next.Pops)
	s.Vals = append(s.Vals, next.Vals...)
}
*/

// simplify cleans up low elements.
// TODO update to new IR structure
func (s *Stack) simplify() {
	// var i int
	// for i = range s.Vals {
	// 	if s.Pops <= 0 {
	// 		break
	// 	}
	// 	if val, ok := (*s.Vals[i]).(*StackVal); !ok || val.Pos != -s.Pops {
	// 		break
	// 	}
	// 	s.Pops--
	// }
	// s.Vals = s.Vals[i:]
}

// Len returns the number of items on the stack.
func (s *Stack) Len() int {
	return len(s.Vals)
}

func (s *Stack) String() string {
	var b strings.Builder
	f := newFormatter()
	b.WriteString("push [")
	for i, val := range s.Vals {
		if i != 0 {
			b.WriteByte(' ')
		}
		b.WriteString(f.FormatDef(val.ValueDef()))
	}
	fmt.Fprintf(&b, "], pop %d, access %d [", s.Pops, s.Access)
	first := true
	for _, val := range s.Under {
		if val != nil {
			if !first {
				b.WriteByte(' ')
			}
			b.WriteString(f.FormatDef(val.ValueDef()))
			first = false
		}
	}
	b.WriteByte(']')
	return b.String()
}
