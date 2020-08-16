package ir

import "math/big"

// BasicBlock is a list of consecutive non-branching instructions in a
// program followed by a branch.
type BasicBlock struct {
	ID         int           // Unique block ID for printing
	LabelName  string        // Name derived from label
	Labels     []Label       // Labels for this block in source
	Nodes      []Inst        // Non-branching non-stack instructions
	Terminator TermInst      // Terminator control flow instruction
	Entries    []*BasicBlock // Entry blocks; blocks immediately preceding this block in flow
	Callers    []*BasicBlock // Calling blocks; blocks calling this block or its parents
	Returns    []*BasicBlock // Returning blocks; blocks returning to this block
	Prev       *BasicBlock   // Predecessor block in source
	Next       *BasicBlock   // Successor block in source
}

// Label is a label with an optional name.
type Label struct {
	ID   *big.Int
	Name string
}

// Disconnect removes incoming edges to a basic block. The block is not
// removed from the program block slice and callers are not updated.
func (block *BasicBlock) Disconnect() {
	if block.Prev != nil {
		block.Prev.Next = block.Next
	}
	if block.Next != nil {
		block.Next.Prev = block.Prev
	}
	for _, exit := range block.Succs() {
		i := 0
		for _, entry := range exit.Entries {
			if entry != block {
				exit.Entries[i] = entry
				i++
			}
		}
		exit.Entries = exit.Entries[:i]
	}
}
