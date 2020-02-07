package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"fmt"
	"math/big"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
)

// ConvertSSA converts tokens into Nebula IR in SSA form.
func (p *Program) ConvertSSA() (*ir.Program, error) {
	if needsImplicitEnd(p.Tokens) {
		p.Tokens = append(p.Tokens, Token{Type: End})
	}
	irp, branches, labels, err := p.createBlocks()
	if err != nil {
		return nil, err
	}
	if err := irp.ConnectEdges(branches, labels); err != nil {
		return irp, err
	}
	return irp, nil
}

func needsImplicitEnd(tokens []Token) bool {
	if len(tokens) == 0 {
		return true
	}
	switch tokens[len(tokens)-1].Type {
	case Call, Jmp, Ret, End:
		return false
	}
	return true
}

func (p *Program) createBlocks() (*ir.Program, []*big.Int, *bigint.Map, error) {
	irp := &ir.Program{
		Name:      p.Name,
		ConstVals: *bigint.NewMap(),
	}
	var branches []*big.Int
	labels := bigint.NewMap() // map[*big.Int]int
	prevLabel := ""
	labelIndex := 0

	for i := 0; i < len(p.Tokens); i++ {
		var block ir.BasicBlock
		block.ID = len(irp.Blocks)
		block.Stack.Block = &block
		if len(irp.Blocks) > 0 {
			prev := irp.Blocks[len(irp.Blocks)-1]
			prev.Next = &block
			block.Prev = prev
		}

		if p.Tokens[i].Type != Label && i != 0 && prevLabel != "" {
			labelIndex++
			block.Labels = append(block.Labels, ir.Label{ID: nil, Name: fmt.Sprintf("%s%d", prevLabel, labelIndex)})
		}
		for p.Tokens[i].Type == Label {
			label := p.Tokens[i].Arg
			if labels.Put(label, len(irp.Blocks)) {
				return nil, nil, nil, fmt.Errorf("ir: label is not unique: %s", label)
			}
			var name string
			if p.LabelNames != nil {
				if n, ok := p.LabelNames.Get(label); ok {
					name = n.(string)
				}
			}
			prevLabel = name
			labelIndex = 0
			block.Labels = append(block.Labels, ir.Label{ID: label, Name: name})
			i++
		}

		checkStack := &ir.CheckStackStmt{}
		block.AppendNode(checkStack)

		var branch *big.Int
		for ; i < len(p.Tokens); i++ {
			branch = appendInstruction(irp, &block, p.Tokens[i])
			if block.Terminator != nil {
				if p.Tokens[i].Type == Label {
					i--
				}
				break
			}
		}

		if block.Stack.Access > 0 {
			checkStack.Access = block.Stack.Access
		} else {
			block.Nodes = block.Nodes[1:]
		}

		irp.Blocks = append(irp.Blocks, &block)
		branches = append(branches, branch)
	}
	irp.Entry = irp.Blocks[0]
	irp.NextBlockID = len(irp.Blocks)
	return irp, branches, labels, nil
}

func appendInstruction(p *ir.Program, block *ir.BasicBlock, tok Token) *big.Int {
	switch tok.Type {
	case Push:
		block.Stack.Push(p.LookupConst(tok.Arg))
	case Dup:
		block.Stack.Dup()
	case Copy:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ws: copy argument overflow: %v", tok.Arg))
		} else if n < 0 {
			panic(fmt.Sprintf("ws: copy argument negative: %v", tok.Arg))
		}
		block.Stack.Copy(n)
	case Swap:
		block.Stack.Swap()
	case Drop:
		block.Stack.Drop()
	case Slide:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ws: slide argument overflow: %v", tok.Arg))
		} else if n < 0 {
			panic(fmt.Sprintf("ws: slide argument negative: %v", tok.Arg))
		}
		block.Stack.Slide(n)

	case Add:
		appendArith(p, block, ir.Add)
	case Sub:
		appendArith(p, block, ir.Sub)
	case Mul:
		appendArith(p, block, ir.Mul)
	case Div:
		appendArith(p, block, ir.Div)
	case Mod:
		appendArith(p, block, ir.Mod)

	case Store:
		val, addr := block.Stack.Pop(), block.Stack.Pop()
		block.AppendNode(&ir.StoreHeapStmt{Addr: addr, Val: val})
	case Retrieve:
		addr, assign := block.Stack.Pop(), p.NextVal()
		block.Stack.Push(assign)
		block.AppendNode(&ir.LoadHeapExpr{Assign: assign, Addr: addr})

	case Label:
		block.Terminator = &ir.JmpStmt{Op: ir.Fallthrough}
		return tok.Arg
	case Call:
		block.Terminator = &ir.CallStmt{}
		return tok.Arg
	case Jmp:
		block.Terminator = &ir.JmpStmt{Op: ir.Jmp}
		return tok.Arg
	case Jz:
		block.Terminator = &ir.JmpCondStmt{Op: ir.Jz, Cond: block.Stack.Pop()}
		return tok.Arg
	case Jn:
		block.Terminator = &ir.JmpCondStmt{Op: ir.Jn, Cond: block.Stack.Pop()}
		return tok.Arg
	case Ret:
		block.Terminator = &ir.RetStmt{}
	case End:
		block.Terminator = &ir.ExitStmt{}

	case Printc:
		block.AppendNode(&ir.PrintStmt{Op: ir.Printc, Val: block.Stack.Pop()})
		// block.AppendNode(&ir.FlushStmt{})
	case Printi:
		block.AppendNode(&ir.PrintStmt{Op: ir.Printi, Val: block.Stack.Pop()})
		// block.AppendNode(&ir.FlushStmt{})
	case Readc:
		appendRead(p, block, ir.Readc)
	case Readi:
		appendRead(p, block, ir.Readi)

	default:
		panic(fmt.Sprintf("ws: unrecognized token type: %v", tok.Type))
	}
	return nil
}

func appendArith(p *ir.Program, block *ir.BasicBlock, op ir.OpType) {
	rhs, lhs := block.Stack.Pop(), block.Stack.Pop()
	assign := p.NextVal()
	block.Stack.Push(assign)
	block.AppendNode(&ir.BinaryExpr{
		Op:     op,
		Assign: assign,
		LHS:    lhs,
		RHS:    rhs,
	})
}

func appendRead(p *ir.Program, block *ir.BasicBlock, op ir.OpType) {
	addr, assign := block.Stack.Pop(), p.NextVal()
	block.AppendNode(&ir.ReadExpr{
		Op:     op,
		Assign: assign,
	})
	block.AppendNode(&ir.StoreHeapStmt{
		Addr: addr,
		Val:  assign,
	})
}
