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
	ib.errs = append(ib.errs, &TokenError{tok, ib.file.Position(tok.Pos), err})
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
		if !tok.Type.IsControl() {
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
	if needsFinalBlock(ib.tokens) {
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

func needsFinalBlock(tokens []*Token) bool {
	if len(tokens) == 0 {
		return true
	}
	switch tokens[len(tokens)-1].Type {
	case Call, Jz, Jn:
		return true
	}
	return false
}

func (ib *irBuilder) convertBlock(block *ir.BasicBlock, tokens []*Token) {
	ib.SetCurrentBlock(block)
	ib.stack.Clear()
	start := true
	for _, tok := range tokens {
		pos := tok.Pos
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
		case Shuffle:
			// Shuffle invalidates SSA value references and in most cases, the
			// stack would need to be preserved.
			ib.err("shuffle instruction not supported", tok)

		case Add:
			lhs, rhs := ib.stack.Pop2(pos)
			ib.stack.Push(ib.CreateBinaryExpr(ir.Add, lhs, rhs, pos))
		case Sub:
			lhs, rhs := ib.stack.Pop2(pos)
			ib.stack.Push(ib.CreateBinaryExpr(ir.Sub, lhs, rhs, pos))
		case Mul:
			lhs, rhs := ib.stack.Pop2(pos)
			ib.stack.Push(ib.CreateBinaryExpr(ir.Mul, lhs, rhs, pos))
		case Div:
			lhs, rhs := ib.stack.Pop2(pos)
			ib.stack.Push(ib.CreateBinaryExpr(ir.Div, lhs, rhs, pos))
		case Mod:
			lhs, rhs := ib.stack.Pop2(pos)
			ib.stack.Push(ib.CreateBinaryExpr(ir.Mod, lhs, rhs, pos))

		case Store:
			addr, val := ib.stack.Pop2(pos)
			ib.CreateStoreHeapStmt(addr, val, pos)
		case Retrieve:
			ib.stack.Push(ib.CreateLoadHeapExpr(ib.stack.Pop(pos), pos))

		case Label:
			if start {
				block.Labels = append(block.Labels, ir.Label{ID: tok.Arg, Name: tok.ArgString})
			}
		case Call:
			ib.CreateCallTerm(ib.callee(tok), block.Next, pos)
		case Jmp:
			ib.CreateJmpTerm(ir.Jmp, ib.callee(tok), pos)
		case Jz:
			ib.CreateJmpCondTerm(ir.Jz, ib.stack.Pop(pos), ib.callee(tok), block.Next, pos)
		case Jn:
			ib.CreateJmpCondTerm(ir.Jn, ib.stack.Pop(pos), ib.callee(tok), block.Next, pos)
		case Ret:
			ib.CreateRetTerm(pos)
		case End:
			ib.CreateExitTerm(pos)

		case Printc:
			ib.CreatePrintStmt(ir.PrintByte, ib.stack.Pop(pos), pos)
			ib.CreateFlushStmt(pos)
		case Printi:
			ib.CreatePrintStmt(ir.PrintInt, ib.stack.Pop(pos), pos)
			ib.CreateFlushStmt(pos)
		case Readc:
			ib.CreateStoreHeapStmt(ib.stack.Pop(pos), ib.CreateReadExpr(ir.ReadByte, pos), pos)
		case Readi:
			ib.CreateStoreHeapStmt(ib.stack.Pop(pos), ib.CreateReadExpr(ir.ReadInt, pos), pos)

		// Aggressive optimizations may discard information needed to dump
		// the stack or heap.
		case Trace:
			ib.err("trace instruction not supported", tok)
		case DumpStack:
			ib.err("dumpstack instruction not supported", tok)
		case DumpHeap:
			ib.err("dumpheap instruction not supported", tok)

		default:
			panic(fmt.Sprintf("ws: unrecognized token type: %v", tok.Type))
		}
		if tok.Type != Label {
			start = false
		}
	}
	if offset := ib.stack.Len() - ib.stack.Pops(); offset != 0 {
		ib.CreateOffsetStackStmt(offset, token.NoPos) // TODO source position
	}
	for i, val := range ib.stack.Values() {
		ib.CreateStoreStackStmt(ib.stack.Len()-i, val, val.Pos())
	}
	if block.Terminator == nil {
		if block.Next != nil {
			ib.CreateJmpTerm(ir.Fallthrough, block.Next, token.NoPos) // TODO source position
		} else {
			ib.CreateExitTerm(token.NoPos) // TODO source position
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

func (ib *irBuilder) handleAccess(n int, pos token.Pos) {
	ib.CreateAccessStackStmt(n, pos)
}

func (ib *irBuilder) handleLoad(n int, pos token.Pos) ir.Value {
	return ib.CreateLoadStackExpr(n, pos)
}
