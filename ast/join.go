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
	next.lookupUnderflow(&block.Stack)
	if next.Stack.Access > 0 {
		block.Stack.At(next.Stack.Access - 1)
	}
	block.Stack.PopN(next.Stack.Pops)
	block.Stack.Vals = append(block.Stack.Vals, next.Stack.Vals...)

	block.Nodes = append(block.Nodes, next.Nodes...)
	block.Terminator = next.Terminator
	next.Disconnect()
}

func (block *BasicBlock) lookupUnderflow(stack *Stack) {
	for _, node := range block.Nodes {
		if assign, ok := node.(*AssignStmt); ok {
			lookupUnderflow(assign.Assign, stack)
			node = assign.Expr
		}
		switch expr := node.(type) {
		case *ArithExpr:
			lookupUnderflow(expr.LHS, stack)
			lookupUnderflow(expr.RHS, stack)
		case *HeapExpr:
			lookupUnderflow(expr.Val, stack)
		case *PrintStmt:
			lookupUnderflow(expr.Val, stack)
		case *ReadExpr:
		}
	}
}

func lookupUnderflow(val *Val, stack *Stack) {
	if v, ok := (*val).(*AddrVal); ok {
		lookupUnderflow(v.Val, stack)
		return
	}
	if v, ok := (*val).(*StackVal); ok && v.Val < 0 {
		*val = *stack.At(-v.Val - 1)
	}
}
