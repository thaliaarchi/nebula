package ast // import "github.com/andrewarchi/nebula/ast"

// JoinSimpleEntries joins blocks that have only one entry with their
// entry block.
// TODO: this does not repect the graph dependency ordering.
func (ast *AST) JoinSimpleEntries() {
	j := 0
	for i, block := range ast.Blocks {
		if len(block.Entries) == 1 {
			entry := block.Entries[0]
			if _, ok := entry.Terminator.(*JmpStmt); ok {
				entry.Join(block)
				continue
			} else {
				block.Stack.LookupUnderflow(&entry.Stack)
			}
		}
		ast.Blocks[j] = ast.Blocks[i]
		j++
	}
	ast.Blocks = ast.Blocks[:j]
}

// Join concatenates two basic blocks, renumbering the assignments in
// the second block.
// TODO: this does not update Callers.
func (block *BasicBlock) Join(next *BasicBlock) {
	block.Stack.Concat(&next.Stack)
	block.Nodes = append(block.Nodes, next.Nodes...)
	block.Terminator = next.Terminator
	exits := next.Exits()
	next.Disconnect()
	for _, exit := range exits {
		exit.Entries = appendUnique(exit.Entries, block)
	}
}
