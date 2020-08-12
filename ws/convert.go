package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"fmt"
	"go/token"
	"math/big"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
)

// ConvertSSA converts tokens into Nebula IR in SSA form.
func (p *Program) ConvertSSA() (*ir.Program, []error) {
	if needsImplicitEnd(p.Tokens) {
		p.Tokens = append(p.Tokens, &Token{Type: End})
	}
	labels, labelUses, errs := collectLabels(p)
	if len(errs) != 0 {
		return nil, errs
	}
	irp, branches, blockLabels, err := p.createBlocks(labels, labelUses)
	if err != nil {
		return nil, []error{err}
	}
	if err := irp.ConnectEdges(branches, blockLabels); err != nil {
		return irp, []error{err}
	}
	return irp, nil
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

func collectLabels(p *Program) (*bigint.Map, *bigint.Map, []error) {
	labels := bigint.NewMap()    // map[*big.Int]int
	labelUses := bigint.NewMap() // map[*big.Int][]int
	var errs []error

	for i, tok := range p.Tokens {
		switch tok.Type {
		case Label:
			if labels.Put(tok.Arg, i) {
				errs = append(errs, p.tokenError("Label is not unique", tok))
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
		if _, ok := labels.Get(use.K); !ok {
			for _, branch := range use.V.([]int) {
				errs = append(errs, p.tokenError("Label does not exist", p.Tokens[branch]))
			}
		}
	}
	return labels, labelUses, errs
}

func (p *Program) createBlocks(labels, labelUses *bigint.Map) (*ir.Program, []*big.Int, *bigint.Map, error) {
	irp := &ir.Program{
		Name:      p.File.Name(),
		ConstVals: bigint.NewMap(),
		File:      p.File,
	}
	var branches []*big.Int
	blockLabels := bigint.NewMap() // map[*big.Int]int
	prevLabel := ""
	labelIndex := 0

	for i := 0; i < len(p.Tokens); i++ {
		var block ir.BasicBlock
		block.ID = len(irp.Blocks)
		stack := &block.Stack
		stack.LoadHandler = block.AppendNode
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
			blockLabels.Put(label, len(irp.Blocks)) // TODO remove need for this
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

		checkStack := ir.NewCheckStackStmt(-1, token.NoPos) // TODO source position
		block.AppendNode(checkStack)

		var branch *big.Int
		for ; i < len(p.Tokens); i++ {
			err := appendInstruction(p, irp, &block, p.Tokens[i], labelUses)
			if err != nil {
				return nil, nil, nil, err
			}
			if block.Terminator != nil {
				branch = p.Tokens[i].Arg
				if p.Tokens[i].Type == Label {
					i--
				}
				break
			}
		}

		if stack.Access > 0 {
			checkStack.StackSize = stack.Access
		} else {
			block.Nodes = block.Nodes[1:]
		}

		if offset := len(stack.Values) - stack.Pops; offset != 0 {
			block.AppendNode(ir.NewOffsetStackStmt(offset, token.NoPos)) // TODO source position
		}
		for i, val := range stack.Values {
			block.AppendNode(ir.NewStoreStackStmt(len(stack.Values)-i, val, token.NoPos)) // TODO source position
		}

		irp.Blocks = append(irp.Blocks, &block)
		branches = append(branches, branch)
	}
	irp.Entry = irp.Blocks[0]
	irp.NextBlockID = len(irp.Blocks)
	return irp, branches, blockLabels, nil
}

func appendInstruction(p *Program, irp *ir.Program, block *ir.BasicBlock, tok *Token, labelUses *bigint.Map) error {
	stack := &block.Stack
	switch tok.Type {
	case Push:
		stack.Push(irp.LookupConst(tok.Arg, tok.Start))
	case Dup:
		stack.Dup()
	case Copy:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			return p.tokenError("Argument overflows int", tok)
		} else if n < 0 {
			return p.tokenError("Argument is negative", tok)
		}
		stack.Copy(n)
	case Swap:
		stack.Swap()
	case Drop:
		stack.Drop()
	case Slide:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			return p.tokenError("Argument overflows int", tok)
		} else if n < 0 {
			return p.tokenError("Argument is negative", tok)
		}
		stack.Slide(n)

	case Add:
		appendBinary(block, stack, ir.Add, tok.Start)
	case Sub:
		appendBinary(block, stack, ir.Sub, tok.Start)
	case Mul:
		appendBinary(block, stack, ir.Mul, tok.Start)
	case Div:
		appendBinary(block, stack, ir.Div, tok.Start)
	case Mod:
		appendBinary(block, stack, ir.Mod, tok.Start)

	case Store:
		val, addr := stack.Pop(), stack.Pop()
		block.AppendNode(ir.NewStoreHeapStmt(addr, val, tok.Start))
	case Retrieve:
		addr := stack.Pop()
		load := ir.NewLoadHeapExpr(addr, tok.Start)
		stack.Push(load)
		block.AppendNode(load)

	case Label:
		if labelUses.Has(tok.Arg) { // split blocks at used labels
			block.Terminator = ir.NewJmpTerm(ir.Fallthrough, nil, tok.Start)
		}
	case Call:
		block.Terminator = ir.NewCallTerm(nil, nil, tok.Start)
	case Jmp:
		block.Terminator = ir.NewJmpTerm(ir.Jmp, nil, tok.Start)
	case Jz:
		block.Terminator = ir.NewJmpCondTerm(ir.Jz, stack.Pop(), nil, nil, tok.Start)
	case Jn:
		block.Terminator = ir.NewJmpCondTerm(ir.Jn, stack.Pop(), nil, nil, tok.Start)
	case Ret:
		block.Terminator = ir.NewRetTerm(tok.Start)
	case End:
		block.Terminator = ir.NewExitTerm(tok.Start)

	case Printc:
		block.AppendNode(ir.NewPrintStmt(ir.Printc, stack.Pop(), tok.Start))
		block.AppendNode(ir.NewFlushStmt(tok.Start))
	case Printi:
		block.AppendNode(ir.NewPrintStmt(ir.Printi, stack.Pop(), tok.Start))
		block.AppendNode(ir.NewFlushStmt(tok.Start))
	case Readc:
		appendRead(block, stack, ir.Readc, tok.Start)
	case Readi:
		appendRead(block, stack, ir.Readi, tok.Start)

	default:
		panic(fmt.Sprintf("ws: unrecognized token type: %v", tok.Type))
	}
	return nil
}

func appendBinary(block *ir.BasicBlock, stack *ir.Stack, op ir.BinaryOp, pos token.Pos) {
	rhs, lhs := stack.Pop(), stack.Pop()
	bin := ir.NewBinaryExpr(op, lhs, rhs, pos)
	stack.Push(bin)
	block.AppendNode(bin)
}

func appendRead(block *ir.BasicBlock, stack *ir.Stack, op ir.ReadOp, pos token.Pos) {
	addr := stack.Pop()
	read := ir.NewReadExpr(op, pos)
	store := ir.NewStoreHeapStmt(addr, read, pos)
	block.AppendNode(read)
	block.AppendNode(store)
}

// TODO maybe change into type
func (p *Program) tokenError(err string, tok *Token) error {
	return fmt.Errorf("%s: %s at %v", err, tok.Format(p.LabelNames), p.Position(tok.Start))
}
