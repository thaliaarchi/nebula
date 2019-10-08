package ast

// MergeSimpleCalls merges blocks that are only called once with their
// predecessor and removes uncalled blocks.
// TODO: this is flawed because Callers does not contain all in-edges.
func (ast *AST) MergeSimpleCalls() {
	j := 0
	for i, block := range ast.Blocks {
		if len(block.Callers) == 0 && block.ID != 0 {
			continue
		}
		if len(block.Callers) == 1 {
			caller := block.Callers[0]
			if _, ok := caller.Exit.(*JmpStmt); ok {
				caller.Merge(block)
				continue
			}
		}
		ast.Blocks[j] = ast.Blocks[i]
		j++
	}
	ast.Blocks = ast.Blocks[:j]
}

// Merge concatenates two basic blocks, renumbering the assignments in
// the second block.
func (block *BasicBlock) Merge(next *BasicBlock) {
	for i := range next.Nodes {
		node := next.Nodes[i]
		if assign, ok := node.(*AssignStmt); ok {
			renumber(assign.Assign, &block.Stack)
			node = assign.Expr
		}
		switch expr := node.(type) {
		case *ArithExpr:
			renumber(expr.LHS, &block.Stack)
			renumber(expr.RHS, &block.Stack)
		case *HeapExpr:
			renumber(expr.Val, &block.Stack)
		case *PrintStmt:
			renumber(expr.Val, &block.Stack)
		case *ReadExpr:
		}
	}
	block.Nodes = append(block.Nodes, next.Nodes...)
	block.Exit = next.Exit

	block.Stack.Next += next.Stack.Next
	if next.Stack.Access > 0 {
		block.Stack.Nth(next.Stack.Access - 1)
	}
	block.Stack.PopN(next.Stack.Pops)
	block.Stack.Vals = append(block.Stack.Vals, next.Stack.Vals...)
}

func renumber(val *Val, stack *Stack) {
	if v, ok := (*val).(*AddrVal); ok {
		val = v.Val
	}
	if v, ok := (*val).(*StackVal); ok {
		if v.Val >= 0 {
			v.Val += stack.Next
		} else {
			*val = *stack.Nth(-v.Val - 1)
		}
	}
}
