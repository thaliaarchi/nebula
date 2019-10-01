package ast

// Stack represents the Whitespace stack for registerization.
type Stack struct {
	Vals []int
	Next int
	Min  int
}

// Push pushes an item to the stack and appends the id of the item.
func (s *Stack) Push() int {
	n := s.Next
	s.Vals = append(s.Vals, s.Next)
	s.Next++
	return n
}

// Pop pops an item from the stack.
func (s *Stack) Pop() {
	if len(s.Vals) == 0 {
		s.Min--
	} else {
		s.Vals = s.Vals[:len(s.Vals)-1]
	}
}
