package ir

import (
	"fmt"
	"math/big"
)

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

func (l *Label) String() string {
	if l.Name != "" {
		return l.Name
	}
	return fmt.Sprintf("label_%v", l.ID)
}

// AppendInst appends an instruction to the block.
func (block *BasicBlock) AppendInst(inst Inst) {
	if _, ok := inst.(TermInst); ok {
		panic("AppendInst: terminator not allowed")
	}
	block.Nodes = append(block.Nodes, inst)
}

// SetTerminator sets the terminator instruction of the block.
func (block *BasicBlock) SetTerminator(term TermInst) {
	if block.Terminator != nil {
		panic("SetTerminator: terminator already set")
	}
	block.Terminator = term
}

// Name returns the name of the basic block from either the first label
// or the block address.
func (block *BasicBlock) Name() string {
	if block == nil {
		return "<nil>"
	}
	if block.LabelName != "" {
		return block.LabelName
	}
	if len(block.Labels) != 0 {
		return block.Labels[0].String()
	}
	return fmt.Sprintf("block_%d", block.ID)
}

func (block *BasicBlock) String() string {
	return NewFormatter().FormatBlock(block)
}

// Succs returns all outgoing edges of the block.
func (block *BasicBlock) Succs() []*BasicBlock {
	switch term := block.Terminator.(type) {
	case *RetTerm:
		exits := make([]*BasicBlock, len(block.Callers))
		for i, caller := range block.Callers {
			if caller != nil {
				exits[i] = caller.Next
			}
		}
		return exits
	default:
		return term.Succs()
	}
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
