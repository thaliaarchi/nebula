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
	case Jmp, Ret, End:
		return false
	}
	return true
}

func (p *Program) createBlocks() (*ir.Program, []*big.Int, *bigint.Map, error) {
	irp := &ir.Program{
		Name:      p.File.Name(),
		ConstVals: bigint.NewMap(),
	}
	var branches []*big.Int
	labels := bigint.NewMap()           // map[*big.Int]int
	labelUses := getLabelUses(p.Tokens) // map[*big.Int]struct{}
	prevLabel := ""
	labelIndex := 0

	for i := 0; i < len(p.Tokens); i++ {
		var block ir.BasicBlock
		block.ID = len(irp.Blocks)
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
			branch = appendInstruction(irp, &block, p.Tokens[i], labelUses)
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

func getLabelUses(tokens []Token) *bigint.Map {
	labelUses := bigint.NewMap() // map[*big.Int]struct{}
	for _, token := range tokens {
		switch token.Type {
		case Call, Jmp, Jz, Jn:
			labelUses.Put(token.Arg, nil)
		}
	}
	return labelUses
}

func appendInstruction(p *ir.Program, block *ir.BasicBlock, tok Token, labelUses *bigint.Map) *big.Int {
	stack := &block.Stack
	switch tok.Type {
	case Push:
		stack.Push(p.LookupConst(tok.Arg))
	case Dup:
		stack.Dup()
	case Copy:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ws: copy argument overflow: %v", tok.Arg))
		} else if n < 0 {
			panic(fmt.Sprintf("ws: copy argument negative: %v", tok.Arg))
		}
		stack.Copy(n)
	case Swap:
		stack.Swap()
	case Drop:
		stack.Drop()
	case Slide:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ws: slide argument overflow: %v", tok.Arg))
		} else if n < 0 {
			panic(fmt.Sprintf("ws: slide argument negative: %v", tok.Arg))
		}
		stack.Slide(n)

	case Add:
		appendArith(block, stack, ir.Add)
	case Sub:
		appendArith(block, stack, ir.Sub)
	case Mul:
		appendArith(block, stack, ir.Mul)
	case Div:
		appendArith(block, stack, ir.Div)
	case Mod:
		appendArith(block, stack, ir.Mod)

	case Store:
		val, addr := stack.Pop(), stack.Pop()
		store := &ir.StoreHeapStmt{}
		ir.AddUse(addr, store, 0)
		ir.AddUse(val, store, 1)
		block.AppendNode(store)
	case Retrieve:
		addr := stack.Pop()
		load := &ir.LoadHeapExpr{Def: &ir.ValueDef{}}
		ir.AddUse(addr, load, 0)
		stack.Push(load)
		block.AppendNode(load)

	case Label:
		if _, ok := labelUses.Get(tok.Arg); ok { // split blocks at used labels
			block.Terminator = &ir.JmpTerm{Op: ir.Fallthrough}
			return tok.Arg
		}
	case Call:
		block.Terminator = &ir.CallTerm{}
		return tok.Arg
	case Jmp:
		block.Terminator = &ir.JmpTerm{Op: ir.Jmp}
		return tok.Arg
	case Jz:
		appendJmpCond(block, stack, ir.Jz)
		return tok.Arg
	case Jn:
		appendJmpCond(block, stack, ir.Jn)
		return tok.Arg
	case Ret:
		block.Terminator = &ir.RetTerm{}
	case End:
		block.Terminator = &ir.ExitTerm{}

	case Printc:
		appendPrint(block, stack, ir.Printc)
	case Printi:
		appendPrint(block, stack, ir.Printi)
	case Readc:
		appendRead(block, stack, ir.Readc)
	case Readi:
		appendRead(block, stack, ir.Readi)

	default:
		panic(fmt.Sprintf("ws: unrecognized token type: %v", tok.Type))
	}
	return nil
}

func appendArith(block *ir.BasicBlock, stack *ir.Stack, op ir.OpType) {
	rhs, lhs := stack.Pop(), stack.Pop()
	bin := &ir.BinaryExpr{Def: &ir.ValueDef{}, Op: op}
	ir.AddUse(lhs, bin, 0)
	ir.AddUse(rhs, bin, 1)
	stack.Push(bin)
	block.AppendNode(bin)
}

func appendJmpCond(block *ir.BasicBlock, stack *ir.Stack, op ir.OpType) {
	cond := stack.Pop()
	jmp := &ir.JmpCondTerm{Op: op}
	ir.AddUse(cond, jmp, 0)
	block.Terminator = jmp
}

func appendPrint(block *ir.BasicBlock, stack *ir.Stack, op ir.OpType) {
	val := stack.Pop()
	print := &ir.PrintStmt{Op: op}
	ir.AddUse(val, print, 0)
	block.AppendNode(print)
	// block.AppendNode(&ir.FlushStmt{})
}

func appendRead(block *ir.BasicBlock, stack *ir.Stack, op ir.OpType) {
	addr := stack.Pop()
	read := &ir.ReadExpr{Def: &ir.ValueDef{}, Op: op}
	store := &ir.StoreHeapStmt{}
	ir.AddUse(addr, store, 0)
	ir.AddUse(read, store, 1)
	block.AppendNode(read)
	block.AppendNode(store)
}
