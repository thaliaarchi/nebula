package ws

import (
	"fmt"
	"go/token"

	"github.com/andrewarchi/nebula/internal/bigint"
	"github.com/andrewarchi/nebula/ir"
)

// irBuilder lowers a Whitespace AST to SSA form.
type irBuilder struct {
	*ir.Program
	tokens      []*Token
	tokenBlocks [][]*Token
	block       *ir.BasicBlock
	stack       *ir.Stack
	labels      *bigint.Map // map[*big.Int]int
	labelUses   *bigint.Map // map[*big.Int][]int
	labelBlocks *bigint.Map // map[*big.Int]*ir.BasicBlock
	errs        []error
}

// TokenError is an error emitted while lowering to SSA form.
type TokenError struct {
	Token *Token
	Pos   token.Position
	Err   string
}

func (err *TokenError) Error() string {
	return fmt.Sprintf("%s: %v at %v", err.Err, err.Token, err.Pos)
}

func (ib *irBuilder) err(err string, tok *Token) {
	ib.errs = append(ib.errs, &TokenError{tok, ib.position(tok.Start), err})
}

// LowerIR lowers a Whitespace AST to Nebula IR in SSA form.
func (p *Program) LowerIR() (*ir.Program, []error) {
	ib := &irBuilder{
		Program: &ir.Program{
			Name: p.File.Name(),
			File: p.File,
		},
		tokens:      p.Tokens,
		labels:      bigint.NewMap(),
		labelUses:   bigint.NewMap(),
		labelBlocks: bigint.NewMap(),
	}
	ib.stack = &ir.Stack{
		HandleAccess: ib.handleAccess,
		HandleLoad:   ib.handleLoad,
	}
	errs := ib.collectLabels()
	if len(errs) != 0 {
		return nil, errs
	}
	ib.splitTokens()
	for i, tokens := range ib.tokenBlocks {
		ib.block = ib.Blocks[i]
		ib.convertBlock(tokens)
	}
	ib.nameBlocks()
	err := ib.Program.ConnectEntries()
	if err != nil {
		errs = append(errs, err)
	}
	return ib.Program, errs
}

// collectLabels collects all labels from the tokens into maps and
// enforces that all labels are unique and callees exist.
func (ib *irBuilder) collectLabels() []error {
	var errs []error
	for i, tok := range ib.tokens {
		switch tok.Type {
		case Label:
			if ib.labels.Put(tok.Arg, i) {
				ib.err("Label is not unique", tok)
			}
		case Call, Jmp, Jz, Jn:
			if l, ok := ib.labelUses.Get(tok.Arg); ok {
				ib.labelUses.Put(tok.Arg, append(l.([]int), i))
			} else {
				ib.labelUses.Put(tok.Arg, []int{i})
			}
		}
	}

	for _, use := range ib.labelUses.Pairs() {
		if _, ok := ib.labels.Get(use.K); !ok {
			for _, branch := range use.V.([]int) {
				ib.err("Label does not exist", ib.tokens[branch])
			}
		}
	}
	return errs
}

// splitTokens splits the tokens into sequences of non-branching tokens.
func (ib *irBuilder) splitTokens() {
	start := true
	lo := 0
	for i := 0; i < len(ib.tokens); i++ {
		tok := ib.tokens[i]
		if !tok.Type.IsFlow() {
			start = false
			continue
		}
		if tok.Type == Label {
			if start || !ib.labelUses.Has(tok.Arg) {
				continue
			}
			i--
		}
		ib.tokenBlocks = append(ib.tokenBlocks, ib.tokens[lo:i+1])
		lo = i + 1
		start = true
	}
	if lo < len(ib.tokens) {
		ib.tokenBlocks = append(ib.tokenBlocks, ib.tokens[lo:])
	}
	if needsImplicitEnd(ib.tokens) {
		ib.tokenBlocks = append(ib.tokenBlocks, []*Token{})
	}

	ib.Blocks = make([]*ir.BasicBlock, len(ib.tokenBlocks))
	for i := range ib.Blocks {
		block := &ir.BasicBlock{ID: i}
		ib.Blocks[i] = block
		if i > 0 {
			block.Prev = ib.Blocks[i-1]
			ib.Blocks[i-1].Next = block
		}
		for _, tok := range ib.tokenBlocks[i] {
			if tok.Type == Label {
				ib.labelBlocks.Put(tok.Arg, block)
			} else {
				break
			}
		}
	}
	ib.Entry = ib.Blocks[0]
	ib.NextBlockID = len(ib.Blocks)
}

func needsImplicitEnd(tokens []*Token) bool {
	if len(tokens) == 0 {
		return true
	}
	switch tokens[len(tokens)-1].Type {
	case Jmp, Ret, End:
		return false
	}
	return true
}

func (ib *irBuilder) convertBlock(tokens []*Token) {
	ib.stack.Clear()
	start := true
	for _, tok := range tokens {
		pos := tok.Start
		switch tok.Type {
		case Push:
			ib.stack.Push(ir.NewIntConst(tok.Arg, pos))
		case Dup:
			ib.stack.Dup(pos)
		case Copy:
			if n, ok := ib.intArg(tok); ok {
				ib.stack.Copy(n, pos)
			}
		case Swap:
			ib.stack.Swap(pos)
		case Drop:
			ib.stack.Drop(pos)
		case Slide:
			if n, ok := ib.intArg(tok); ok {
				ib.stack.Slide(n, pos)
			}

		case Add:
			ib.appendBinary(ir.Add, pos)
		case Sub:
			ib.appendBinary(ir.Sub, pos)
		case Mul:
			ib.appendBinary(ir.Mul, pos)
		case Div:
			ib.appendBinary(ir.Div, pos)
		case Mod:
			ib.appendBinary(ir.Mod, pos)

		case Store:
			addr, val := ib.stack.Pop2(pos)
			ib.block.AppendInst(ir.NewStoreHeapStmt(addr, val, pos))
		case Retrieve:
			addr := ib.stack.Pop(pos)
			load := ir.NewLoadHeapExpr(addr, pos)
			ib.stack.Push(load)
			ib.block.AppendInst(load)

		case Label:
			if start {
				ib.block.Labels = append(ib.block.Labels, ir.Label{ID: tok.Arg, Name: tok.ArgString})
			}
		case Call:
			ib.block.SetTerminator(ir.NewCallTerm(ib.callee(tok), ib.block.Next, pos))
		case Jmp:
			ib.block.SetTerminator(ir.NewJmpTerm(ir.Jmp, ib.callee(tok), pos))
		case Jz:
			ib.block.SetTerminator(ir.NewJmpCondTerm(ir.Jz, ib.stack.Pop(pos), ib.callee(tok), ib.block.Next, pos))
		case Jn:
			ib.block.SetTerminator(ir.NewJmpCondTerm(ir.Jn, ib.stack.Pop(pos), ib.callee(tok), ib.block.Next, pos))
		case Ret:
			ib.block.SetTerminator(ir.NewRetTerm(pos))
		case End:
			ib.block.SetTerminator(ir.NewExitTerm(pos))

		case Printc:
			ib.appendPrint(ir.PrintByte, pos)
		case Printi:
			ib.appendPrint(ir.PrintInt, pos)
		case Readc:
			ib.appendRead(ir.ReadByte, pos)
		case Readi:
			ib.appendRead(ir.ReadInt, pos)

		default:
			panic(fmt.Sprintf("ws: unrecognized token type: %v", tok.Type))
		}
		if tok.Type != Label {
			start = false
		}
	}
	if offset := ib.stack.Len() - ib.stack.Pops(); offset != 0 {
		ib.block.AppendInst(ir.NewOffsetStackStmt(offset, token.NoPos)) // TODO source position
	}
	for i, val := range ib.stack.Values() {
		ib.block.AppendInst(ir.NewStoreStackStmt(ib.stack.Len()-i, val, val.Pos()))
	}
	if ib.block.Terminator == nil {
		if ib.block.Next != nil {
			ib.block.SetTerminator(ir.NewJmpTerm(ir.Fallthrough, ib.block.Next, token.NoPos)) // TODO source position
		} else {
			ib.block.SetTerminator(ir.NewExitTerm(token.NoPos)) // TODO source position
		}
	}
}

func (ib *irBuilder) intArg(tok *Token) (int, bool) {
	n, ok := bigint.ToInt(tok.Arg)
	if !ok {
		ib.err("Argument overflows int", tok)
	} else if n < 0 {
		ib.err("Argument is negative", tok)
	}
	return n, ok
}

func (ib *irBuilder) callee(tok *Token) *ir.BasicBlock {
	callee, ok := ib.labelBlocks.Get(tok.Arg)
	if !ok {
		panic(fmt.Sprintf("ws: block %s jumps to non-existent label: label_%v", ib.block.Name(), tok.Arg))
	}
	return callee.(*ir.BasicBlock)
}

func (ib *irBuilder) appendBinary(op ir.BinaryOp, pos token.Pos) {
	lhs, rhs := ib.stack.Pop2(pos)
	bin := ir.NewBinaryExpr(op, lhs, rhs, pos)
	ib.stack.Push(bin)
	ib.block.AppendInst(bin)
}

func (ib *irBuilder) appendPrint(op ir.PrintOp, pos token.Pos) {
	val := ib.stack.Pop(pos)
	ib.block.AppendInst(ir.NewPrintStmt(op, val, pos))
	ib.block.AppendInst(ir.NewFlushStmt(pos))
}

func (ib *irBuilder) appendRead(op ir.ReadOp, pos token.Pos) {
	addr := ib.stack.Pop(pos)
	read := ir.NewReadExpr(op, pos)
	store := ir.NewStoreHeapStmt(addr, read, pos)
	ib.block.AppendInst(read)
	ib.block.AppendInst(store)
}

func (ib *irBuilder) handleAccess(n int, pos token.Pos) {
	ib.block.AppendInst(ir.NewAccessStackStmt(n, pos))
}

func (ib *irBuilder) handleLoad(n int, pos token.Pos) ir.Value {
	load := ir.NewLoadStackExpr(n, pos)
	ib.block.AppendInst(load)
	return load
}

func (ib *irBuilder) nameBlocks() {
	prevLabel := ""
	labelIndex := 0
	for _, block := range ib.Blocks {
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

func (ib *irBuilder) position(pos token.Pos) token.Position {
	return ib.File.PositionFor(pos, false)
}
