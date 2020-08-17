package ir

import (
	"fmt"
	"go/token"
)

// Builder assists in IR construction.
type Builder struct {
	blocks []*BasicBlock
	curr   *BasicBlock
	file   *token.File
}

// RetUnderflowError is an error given when ret is executed without a
// caller.
type RetUnderflowError struct {
	Traces [][]*BasicBlock
}

// NewBuilder constructs a builder with a given number of basic blocks.
func NewBuilder(n int, file *token.File) *Builder {
	if n < 0 {
		panic(fmt.Sprintf("NewBuilder: negative block count: %d", n))
	}
	blocks := make([]*BasicBlock, n)
	for i := range blocks {
		blocks[i] = &BasicBlock{ID: i}
		if i > 0 {
			blocks[i].Prev = blocks[i-1]
			blocks[i-1].Next = blocks[i]
		}
	}
	return &Builder{blocks: blocks, file: file}
}

// Blocks returns all blocks.
func (b *Builder) Blocks() []*BasicBlock { return b.blocks }

// Block returns the nth block.
func (b *Builder) Block(n int) *BasicBlock { return b.blocks[n] }

// CurrentBlock returns the currently selected block.
func (b *Builder) CurrentBlock() *BasicBlock { return b.curr }

// SetCurrentBlock sets the currently selected block.
func (b *Builder) SetCurrentBlock(n int) *BasicBlock {
	b.curr = b.blocks[n]
	return b.curr
}

// AppendInst appends an instruction to the current block.
func (b *Builder) AppendInst(inst Inst) {
	b.curr.AppendInst(inst)
}

// SetTerminator sets the terminator instruction of the current block.
func (b *Builder) SetTerminator(term TermInst) {
	b.curr.SetTerminator(term)
}

// Program completes IR construction and returns a program.
func (b *Builder) Program() (*Program, error) {
	b.nameBlocks()
	err := connectEntries(b.blocks[0], b.blocks)
	p := &Program{
		Name:        b.file.Name(),
		Blocks:      b.blocks,
		Entry:       b.blocks[0],
		NextBlockID: len(b.blocks),
		File:        b.file,
	}
	return p, err
}

func (b *Builder) nameBlocks() {
	prevLabel := ""
	labelIndex := 0
	for _, block := range b.blocks {
		if len(block.Labels) > 0 {
			prevLabel = ""
			labelIndex = 0
			for _, label := range block.Labels {
				if label.Name != "" && block.LabelName == "" {
					block.LabelName = label.Name
					prevLabel = label.Name
					labelIndex = 1
				}
			}
		}
		if block.LabelName == "" && prevLabel != "" {
			block.LabelName = fmt.Sprintf("%s%d", prevLabel, labelIndex)
			labelIndex++
		}
	}
}

// ConnectEntries connects the block entries.
func connectEntries(entry *BasicBlock, blocks []*BasicBlock) error {
	entry.Entries = append(entry.Entries, nil)
	for _, block := range blocks {
		if call, ok := block.Terminator.(*CallTerm); ok {
			// Only connect the callee. Next is connected to ret below.
			call.succs[0].Entries = append(call.succs[0].Entries, block)
		} else {
			for _, succ := range block.Terminator.Succs() {
				succ.Entries = append(succ.Entries, block)
			}
		}
	}
	if err := connectCaller(entry, nil); err != nil {
		return err
	}
	return nil
}

func connectCaller(block, caller *BasicBlock) *RetUnderflowError {
	for _, c := range block.Callers {
		if c == caller {
			return nil
		}
	}
	block.Callers = append(block.Callers, caller)
	var errs *RetUnderflowError
	if block.Terminator == nil {
		panic("ir: nil terminator")
	}
	switch term := block.Terminator.(type) {
	case *CallTerm:
		errs = errs.addTrace(connectCaller(term.succs[0], block), block)
		errs = errs.addTrace(connectCaller(term.succs[1], caller), block)
		term.succs[1].Entries = appendUnique(term.succs[1].Entries, block.Returns...)
	case *JmpTerm:
		errs = errs.addTrace(connectCaller(term.succs[0], caller), block)
	case *JmpCondTerm:
		errs = errs.addTrace(connectCaller(term.succs[0], caller), block)
		errs = errs.addTrace(connectCaller(term.succs[1], caller), block)
	case *RetTerm:
		if caller == nil {
			errs = errs.addTrace(&RetUnderflowError{[][]*BasicBlock{{}}}, block)
		} else {
			caller.Returns = append(caller.Returns, block)
		}
	case *ExitTerm:
	default:
		panic("ir: unrecognized terminator type")
	}
	return errs
}

func appendUnique(slice []*BasicBlock, blocks ...*BasicBlock) []*BasicBlock {
	l := len(slice)
outer:
	for _, block := range blocks {
		for i := 0; i < l; i++ {
			if slice[i] == block {
				continue outer
			}
		}
		slice = append(slice, block)
	}
	return slice
}
