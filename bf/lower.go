package bf

import (
	"fmt"
	"go/token"
	"math/big"

	"github.com/andrewarchi/nebula/ir"
)

type bracketBlock struct {
	Block *ir.BasicBlock
	Pos   token.Pos
}

// LowerIR lowers a Brainfuck program to Nebula IR in SSA form.
func (p *Program) LowerIR() (*ir.Program, []error) {
	b := ir.NewBuilder(p.File)
	b.SetCurrentBlock(b.CreateBlock())
	dataPtr := ir.NewIntConst(big.NewInt(0), token.NoPos)
	one := ir.NewIntConst(big.NewInt(1), token.NoPos)
	b.CreateStoreHeapStmt(dataPtr, one, token.NoPos)
	var bracketStack []bracketBlock
	var errs []error
	for _, tok := range p.Tokens {
		switch tok.Type {
		case IncPtr:
			load := b.CreateLoadHeapExpr(dataPtr, tok.Pos)
			inc := b.CreateBinaryExpr(ir.Add, load, one, tok.Pos)
			b.CreateStoreHeapStmt(dataPtr, inc, tok.Pos)
		case DecPtr:
			load := b.CreateLoadHeapExpr(dataPtr, tok.Pos)
			dec := b.CreateBinaryExpr(ir.Sub, load, one, tok.Pos)
			b.CreateStoreHeapStmt(dataPtr, dec, tok.Pos)
		case IncData:
			data := b.CreateLoadHeapExpr(dataPtr, tok.Pos)
			val := b.CreateLoadHeapExpr(data, tok.Pos)
			inc := b.CreateBinaryExpr(ir.Add, val, one, tok.Pos)
			b.CreateStoreHeapStmt(data, inc, tok.Pos)
		case DecData:
			data := b.CreateLoadHeapExpr(dataPtr, tok.Pos)
			val := b.CreateLoadHeapExpr(data, tok.Pos)
			dec := b.CreateBinaryExpr(ir.Sub, val, one, tok.Pos)
			b.CreateStoreHeapStmt(data, dec, tok.Pos)
		case Print:
			data := b.CreateLoadHeapExpr(dataPtr, tok.Pos)
			val := b.CreateLoadHeapExpr(data, tok.Pos)
			b.CreatePrintStmt(ir.PrintByte, val, tok.Pos)
		case Read:
			val := b.CreateReadExpr(ir.ReadByte, tok.Pos)
			data := b.CreateLoadHeapExpr(dataPtr, tok.Pos)
			b.CreateStoreHeapStmt(data, val, tok.Pos)
		case Bracket:
			data := b.CreateLoadHeapExpr(dataPtr, tok.Pos)
			val := b.CreateLoadHeapExpr(data, tok.Pos)
			next := b.CreateBlock()
			b.CreateJmpCondTerm(ir.Jz, val, nil, next, tok.Pos)
			bracketStack = append(bracketStack, bracketBlock{b.CurrentBlock(), tok.Pos})
			b.SetCurrentBlock(next)
		case EndBracket:
			if len(bracketStack) == 0 {
				errs = append(errs, fmt.Errorf("End bracket not matched at %v", tok.Pos))
				continue
			}
			head := bracketStack[len(bracketStack)-1].Block
			bracketStack = bracketStack[:len(bracketStack)-1]
			head.Terminator.(*ir.JmpCondTerm).Succs()[0] = b.CurrentBlock()

			data := b.CreateLoadHeapExpr(dataPtr, tok.Pos)
			val := b.CreateLoadHeapExpr(data, tok.Pos)
			next := b.CreateBlock()
			b.CreateJmpCondTerm(ir.Jz, val, head, next, tok.Pos)
			b.SetCurrentBlock(next)
		}
	}
	exitPos := token.NoPos
	if len(p.Tokens) > 0 {
		exitPos = p.Tokens[len(p.Tokens)-1].Pos
	}
	b.CreateExitTerm(exitPos)
	for _, bracket := range bracketStack {
		errs = append(errs, fmt.Errorf("Bracket not matched at %v", bracket.Pos))
	}
	ssa, err := b.Program()
	if err != nil {
		errs = append(errs, err)
	}
	return ssa, errs
}
