package analysis // import "github.com/andrewarchi/nebula/analysis"

import "github.com/andrewarchi/nebula/ir"

// JoinSimpleEntries joins blocks that have only one entry with their
// entry block.
// TODO: this does not repect the graph dependency ordering.
func JoinSimpleEntries(p *ir.Program) {
	j := 0
	for i, block := range p.Blocks {
		if len(block.Entries) == 1 && block.Entries[0] != nil {
			entry := block.Entries[0]
			if _, ok := entry.Terminator.(*ir.JmpStmt); ok {
				Join(p, entry, block)
				continue
			} else {
				block.Stack.LookupUnderflow(&entry.Stack)
			}
		}
		p.Blocks[j] = p.Blocks[i]
		j++
	}
	p.Blocks = p.Blocks[:j]
}

// Join concatenates two basic blocks.
func Join(p *ir.Program, prev, next *ir.BasicBlock) {
	prev.Stack.Concat(&next.Stack)
	prev.Nodes = append(prev.Nodes, next.Nodes...)
	prev.Terminator = next.Terminator

	if next.Prev != nil {
		next.Prev.Next = next.Next
	}
	if next.Next != nil {
		next.Next.Prev = next.Prev
	}
	for _, exit := range next.Exits() {
		replaceUnique(exit.Entries, next, prev)
	}
	for _, block := range p.Blocks {
		replaceUnique(block.Callers, next, prev)
	}
}

func replaceUnique(blocks []*ir.BasicBlock, block, replace *ir.BasicBlock) {
	for i := range blocks {
		if blocks[i] == block {
			blocks[i] = replace
			break
		}
	}
}
