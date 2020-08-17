package ir

import (
	"fmt"
	"go/token"
)

// Builder assists in IR construction.
type Builder struct {
	blocks []*BasicBlock
	curr   *BasicBlock
	nextID int
	file   *token.File
}

// RetUnderflowError is an error given when ret is executed without a
// caller.
type RetUnderflowError struct {
	Traces [][]*BasicBlock
}

// NewBuilder constructs a builder with a given number of basic blocks.
func NewBuilder(file *token.File) *Builder {
	return &Builder{file: file}
}

// Blocks returns all blocks.
func (b *Builder) Blocks() []*BasicBlock { return b.blocks }

// Block returns the nth block.
func (b *Builder) Block(n int) *BasicBlock { return b.blocks[n] }

// CurrentBlock returns the currently selected block.
func (b *Builder) CurrentBlock() *BasicBlock { return b.curr }

// SetCurrentBlock sets the currently selected block.
func (b *Builder) SetCurrentBlock(block *BasicBlock) {
	b.curr = block
}

// InitBlocks creates n empty blocks and sets the first to be the
// current block.
func (b *Builder) InitBlocks(n int) {
	if n < 0 {
		panic(fmt.Sprintf("NewBuilder: negative block count: %d", n))
	}
	if n == 0 {
		return
	}
	blocks := make([]*BasicBlock, n)
	for i := range blocks {
		blocks[i] = &BasicBlock{ID: b.nextID}
		if i > 0 {
			blocks[i].Prev = blocks[i-1]
			blocks[i-1].Next = blocks[i]
		}
		b.nextID++
	}
	b.blocks = blocks
	b.curr = blocks[0]
}

// CreateBlock creates a block.
func (b *Builder) CreateBlock() *BasicBlock {
	block := &BasicBlock{ID: b.nextID}
	if len(b.blocks) > 0 {
		prev := b.blocks[len(b.blocks)-1]
		block.Prev = prev
		prev.Next = block
	}
	b.blocks = append(b.blocks, block)
	b.nextID++
	return block
}

// AppendInst appends an instruction to the current block.
func (b *Builder) AppendInst(inst Inst) {
	b.curr.AppendInst(inst)
}

// SetTerminator sets the terminator instruction of the current block.
func (b *Builder) SetTerminator(term TermInst) {
	b.curr.SetTerminator(term)
}

// CreateBinaryExpr constructs a BinaryExpr and appends it to the
// current block.
func (b *Builder) CreateBinaryExpr(op BinaryOp, lhs, rhs Value, pos token.Pos) *BinaryExpr {
	bin := NewBinaryExpr(op, lhs, rhs, pos)
	b.AppendInst(bin)
	return bin
}

// CreateUnaryExpr constructs a UnaryExpr and appends it to the current
// block.
func (b *Builder) CreateUnaryExpr(op UnaryOp, val Value, pos token.Pos) *UnaryExpr {
	un := NewUnaryExpr(op, val, pos)
	b.AppendInst(un)
	return un
}

// CreateLoadStackExpr constructs a LoadStackExpr and appends it to the
// current block.
func (b *Builder) CreateLoadStackExpr(stackPos int, pos token.Pos) *LoadStackExpr {
	load := NewLoadStackExpr(stackPos, pos)
	b.AppendInst(load)
	return load
}

// CreateStoreStackStmt constructs a StoreStackStmt and appends it to
// the current block.
func (b *Builder) CreateStoreStackStmt(stackPos int, val Value, pos token.Pos) *StoreStackStmt {
	store := NewStoreStackStmt(stackPos, val, pos)
	b.AppendInst(store)
	return store
}

// CreateAccessStackStmt constructs a AccessStackStmt and appends it to
// the current block.
func (b *Builder) CreateAccessStackStmt(stackSize int, pos token.Pos) *AccessStackStmt {
	access := NewAccessStackStmt(stackSize, pos)
	b.AppendInst(access)
	return access
}

// CreateOffsetStackStmt constructs a OffsetStackStmt and appends it to
// the current block.
func (b *Builder) CreateOffsetStackStmt(offset int, pos token.Pos) *OffsetStackStmt {
	off := NewOffsetStackStmt(offset, pos)
	b.AppendInst(off)
	return off
}

// CreateLoadHeapExpr constructs a LoadHeapExpr and appends it to the
// current block.
func (b *Builder) CreateLoadHeapExpr(addr Value, pos token.Pos) *LoadHeapExpr {
	load := NewLoadHeapExpr(addr, pos)
	b.AppendInst(load)
	return load
}

// CreateStoreHeapStmt constructs a StoreHeapStmt and appends it to the
// current block.
func (b *Builder) CreateStoreHeapStmt(addr, val Value, pos token.Pos) *StoreHeapStmt {
	store := NewStoreHeapStmt(addr, val, pos)
	b.AppendInst(store)
	return store
}

// CreatePrintStmt constructs a PrintStmt and appends it to the current
// block.
func (b *Builder) CreatePrintStmt(op PrintOp, val Value, pos token.Pos) *PrintStmt {
	print := NewPrintStmt(op, val, pos)
	b.AppendInst(print)
	return print
}

// CreateReadExpr constructs a ReadExpr and appends it to the current
// block.
func (b *Builder) CreateReadExpr(op ReadOp, pos token.Pos) *ReadExpr {
	read := NewReadExpr(op, pos)
	b.AppendInst(read)
	return read
}

// CreateFlushStmt constructs a FlushStmt and appends it to the current
// block.
func (b *Builder) CreateFlushStmt(pos token.Pos) *FlushStmt {
	flush := NewFlushStmt(pos)
	b.AppendInst(flush)
	return flush
}

// CreateCallTerm constructs a CallTerm and appends it to the current
// block.
func (b *Builder) CreateCallTerm(callee, next *BasicBlock, pos token.Pos) *CallTerm {
	call := NewCallTerm(callee, next, pos)
	b.AppendInst(call)
	return call
}

// CreateJmpTerm constructs a JmpTerm and appends it to the current
// block.
func (b *Builder) CreateJmpTerm(op JmpOp, jumpee *BasicBlock, pos token.Pos) *JmpTerm {
	jmp := NewJmpTerm(op, jumpee, pos)
	b.AppendInst(jmp)
	return jmp
}

// CreateJmpCondTerm constructs a JmpCondTerm and appends it to the
// current block.
func (b *Builder) CreateJmpCondTerm(op JmpCondOp, val Value, trueBlock, falseBlock *BasicBlock, pos token.Pos) *JmpCondTerm {
	jc := NewJmpCondTerm(op, val, trueBlock, falseBlock, pos)
	b.AppendInst(jc)
	return jc
}

// CreateRetTerm constructs a RetTerm and appends it to the current
// block.
func (b *Builder) CreateRetTerm(pos token.Pos) *RetTerm {
	ret := NewRetTerm(pos)
	b.AppendInst(ret)
	return ret
}

// CreateExitTerm constructs a ExitTerm and appends it to the current
// block.
func (b *Builder) CreateExitTerm(pos token.Pos) *ExitTerm {
	exit := NewExitTerm(pos)
	b.AppendInst(exit)
	return exit
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
