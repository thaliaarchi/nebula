package ws

import (
	"fmt"
	"go/token"

	"github.com/andrewarchi/nebula/internal/bigint"
	"github.com/andrewarchi/nebula/ir"
)

// irBuilder lowers a Whitespace AST to SSA form.
type irBuilder struct {
	*ir.Builder
	tokens      []*Token
	tokenBlocks [][]*Token
	stack       *ir.Stack
	labelBlocks *bigint.Map // map[*big.Int]*ir.BasicBlock
	file        *token.File
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

func (ib *irBuilder) Errs() []error {
	return ib.errs
}

// LowerIR lowers a Whitespace program to Nebula IR in SSA form.
func (p *Program) LowerIR() (*ir.Program, []error) {
	ib := &irBuilder{
		Builder:     ir.NewBuilder(p.File),
		tokens:      p.Tokens,
		labelBlocks: bigint.NewMap(),
		file:        p.File,
	}
	ib.stack = &ir.Stack{
		HandleAccess: ib.handleAccess,
		HandleLoad:   ib.handleLoad,
	}
	labelUses := ib.collectLabels()
	ib.splitTokens(labelUses)
	for i, tokens := range ib.tokenBlocks {
		ib.convertBlock(ib.Block(i), tokens)
	}
	ssa, err := ib.Program()
	if err != nil {
		ib.errs = append(ib.errs, err)
	}
	return ssa, ib.errs
}

// collectLabels collects all labels from the tokens into maps and
// enforces that all labels are unique and callees exist.
func (ib *irBuilder) collectLabels() *bigint.Map {
	labels := bigint.NewMap()    // map[*big.Int]bool
	labelUses := bigint.NewMap() // map[*big.Int][]int
	for i, tok := range ib.tokens {
		switch tok.Type {
		case Label:
			if labels.Put(tok.Arg, nil) {
				ib.err("Label is not unique", tok)
			}
		case Call, Jmp, Jz, Jn:
			if l, ok := labelUses.Get(tok.Arg); ok {
				labelUses.Put(tok.Arg, append(l.([]int), i))
			} else {
				labelUses.Put(tok.Arg, []int{i})
			}
		}
	}

	for _, use := range labelUses.Pairs() {
		if !labels.Has(use.K) {
			for _, branch := range use.V.([]int) {
				ib.err("Label does not exist", ib.tokens[branch])
			}
		}
	}
	return labelUses
}

// splitTokens splits the tokens into sequences of non-branching tokens.
func (ib *irBuilder) splitTokens(labelUses *bigint.Map) {
	start := true
	lo := 0
	for i := 0; i < len(ib.tokens); i++ {
		tok := ib.tokens[i]
		if !tok.Type.IsFlow() {
			start = false
			continue
		}
		if tok.Type == Label {
			if start || !labelUses.Has(tok.Arg) {
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

	ib.InitBlocks(len(ib.tokenBlocks))
	for i, block := range ib.Blocks() {
		for _, tok := range ib.tokenBlocks[i] {
			if tok.Type == Label {
				ib.labelBlocks.Put(tok.Arg, block)
			} else {
				break
			}
		}
	}
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

func (ib *irBuilder) convertBlock(block *ir.BasicBlock, tokens []*Token) {
	ib.SetCurrentBlock(block)
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
			ib.AppendInst(ir.NewStoreHeapStmt(addr, val, pos))
		case Retrieve:
			addr := ib.stack.Pop(pos)
			load := ir.NewLoadHeapExpr(addr, pos)
			ib.stack.Push(load)
			ib.AppendInst(load)

		case Label:
			if start {
				block.Labels = append(block.Labels, ir.Label{ID: tok.Arg, Name: tok.ArgString})
			}
		case Call:
			ib.SetTerminator(ir.NewCallTerm(ib.callee(tok), block.Next, pos))
		case Jmp:
			ib.SetTerminator(ir.NewJmpTerm(ir.Jmp, ib.callee(tok), pos))
		case Jz:
			ib.SetTerminator(ir.NewJmpCondTerm(ir.Jz, ib.stack.Pop(pos), ib.callee(tok), block.Next, pos))
		case Jn:
			ib.SetTerminator(ir.NewJmpCondTerm(ir.Jn, ib.stack.Pop(pos), ib.callee(tok), block.Next, pos))
		case Ret:
			ib.SetTerminator(ir.NewRetTerm(pos))
		case End:
			ib.SetTerminator(ir.NewExitTerm(pos))

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
		ib.AppendInst(ir.NewOffsetStackStmt(offset, token.NoPos)) // TODO source position
	}
	for i, val := range ib.stack.Values() {
		ib.AppendInst(ir.NewStoreStackStmt(ib.stack.Len()-i, val, val.Pos()))
	}
	if block.Terminator == nil {
		if block.Next != nil {
			ib.SetTerminator(ir.NewJmpTerm(ir.Fallthrough, block.Next, token.NoPos)) // TODO source position
		} else {
			ib.SetTerminator(ir.NewExitTerm(token.NoPos)) // TODO source position
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
	if !ok || callee.(*ir.BasicBlock) == nil {
		panic(fmt.Sprintf("ws: block %s jumps to non-existent label: label_%v", ib.CurrentBlock().Name(), tok.Arg))
	}
	return callee.(*ir.BasicBlock)
}

func (ib *irBuilder) appendBinary(op ir.BinaryOp, pos token.Pos) {
	lhs, rhs := ib.stack.Pop2(pos)
	bin := ir.NewBinaryExpr(op, lhs, rhs, pos)
	ib.stack.Push(bin)
	ib.AppendInst(bin)
}

func (ib *irBuilder) appendPrint(op ir.PrintOp, pos token.Pos) {
	val := ib.stack.Pop(pos)
	ib.AppendInst(ir.NewPrintStmt(op, val, pos))
	ib.AppendInst(ir.NewFlushStmt(pos))
}

func (ib *irBuilder) appendRead(op ir.ReadOp, pos token.Pos) {
	addr := ib.stack.Pop(pos)
	read := ir.NewReadExpr(op, pos)
	store := ir.NewStoreHeapStmt(addr, read, pos)
	ib.AppendInst(read)
	ib.AppendInst(store)
}

func (ib *irBuilder) handleAccess(n int, pos token.Pos) {
	ib.AppendInst(ir.NewAccessStackStmt(n, pos))
}

func (ib *irBuilder) handleLoad(n int, pos token.Pos) ir.Value {
	load := ir.NewLoadStackExpr(n, pos)
	ib.AppendInst(load)
	return load
}

func (ib *irBuilder) position(pos token.Pos) token.Position {
	return ib.file.PositionFor(pos, false)
}
