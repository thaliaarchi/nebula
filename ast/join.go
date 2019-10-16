package ast // import "github.com/andrewarchi/nebula/ast"

// JoinSimpleCalls joins blocks that have only one entry with their
// entry block.
func (ast *AST) JoinSimpleCalls() {
	j := 0
	for i, block := range ast.Blocks {
		if len(block.Entries) == 1 {
			entry := block.Entries[0]
			if _, ok := entry.Terminator.(*JmpStmt); ok {
				entry.Join(block)
				continue
			}
		}
		ast.Blocks[j] = ast.Blocks[i]
		j++
	}
	ast.Blocks = ast.Blocks[:j]
}

// Join concatenates two basic blocks, renumbering the assignments in
// the second block.
func (block *BasicBlock) Join(next *BasicBlock) {
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
	block.Terminator = next.Terminator

	block.Stack.Next += next.Stack.Next
	if next.Stack.Access > 0 {
		block.Stack.At(next.Stack.Access - 1)
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
		renumber(v.Val, stack)
		return
	}
	if v, ok := (*val).(*StackVal); ok {
		if v.Val >= 0 {
			v.Val += stack.Next
		} else {
			*val = *stack.At(-v.Val - 1)
		}
	}
}
