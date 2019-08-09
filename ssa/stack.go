package ssa

// Stack is an immutable stack for modeling the program stack in SSA.
type Stack []Node

// Push pushes a node onto the stack.
func (s Stack) Push(n Node) Stack {
	s2 := make([]Node, len(s)+1)
	copy(s2, s)
	s2[len(s)] = n
	return s2
}

// Dup duplicates the top item on the stack.
func (s Stack) Dup() (Stack, bool) {
	if len(s) == 0 {
		return nil, false
	}
	return s.Push(s[len(s)-1]), true
}

// Copy duplicates the nth item on the stack. The top is at 0.
func (s Stack) Copy(n int) (Stack, bool) {
	if len(s) <= n {
		return nil, false
	}
	return s.Push(s[len(s)-n-1]), true
}

// Swap the top two items on the stack.
func (s Stack) Swap() (Stack, bool) {
	if len(s) < 2 {
		return nil, false
	}
	s2 := make([]Node, len(s))
	copy(s2, s)
	s2[len(s)-1], s2[len(s)-2] = s2[len(s)-2], s2[len(s)-1]
	return s2, true
}

// Pop pops the top item on the stack.
func (s Stack) Pop() (Stack, bool) {
	if len(s) == 0 {
		return nil, false
	}
	return s[:len(s)-1], true
}

// Slide slides n items off the stack, keeping the top item.
func (s Stack) Slide(n int) (Stack, bool) {
	if len(s) <= n || n < 0 {
		return nil, false
	}
	s2 := make([]Node, len(s)-n)
	s2[len(s2)-1] = s[len(s)-1]
	return s2, true
}
