package ast

// MergeSimpleCalls merges blocks that have only one entry with their
// entry block.
func (ast *AST) MergeSimpleCalls() {
	j := 0
	for i, block := range ast.Blocks {
		if len(block.Entries) == 1 {
			entry := block.Entries[0]
			if _, ok := entry.Exit.(*JmpStmt); ok {
				entry.Merge(block)
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

	if next.Prev != nil {
		next.Prev.Next = next.Next
	}
	if next.Next != nil {
		next.Next.Prev = next.Prev
	}
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
