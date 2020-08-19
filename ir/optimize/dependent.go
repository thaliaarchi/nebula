package optimize

import (
	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/nebula/ir"
)

// ControlFlowGraph creates a directed graph with edges representing the
// connections between basic blocks.
func ControlFlowGraph(p *ir.Program) graph.Graph {
	ids := make(map[*ir.BasicBlock]int)
	for _, block := range p.Blocks {
		ids[block] = block.ID
	}
	g := graph.NewGraph(uint(len(p.Blocks)))
	for i, block := range p.Blocks {
		for _, exit := range block.Succs() {
			g.Add(uint(i), uint(ids[exit]))
		}
	}
	return g
}

// DependenceGraph creates an undirected graph with edges representing
// dependencies between nodes.
func DependenceGraph(block *ir.BasicBlock) graph.Graph {
	g := graph.NewGraph(uint(len(block.Nodes)))
	for i, ni := range block.Nodes {
		for j, nj := range block.Nodes[i+1:] {
			if Dependent(ni, nj) {
				g.AddUndirected(uint(i), uint(j))
			}
		}
	}
	return g
}

// Dependent returns whether two non-branching nodes are dependent. True
// is returned when node B is dependent on node A. Nodes are dependent
// when both are I/O instructions, one is I/O and the other can throw,
// both assign to the same value, or one reads the value assigned to by
// the other. Dependent is reflexive.
func Dependent(a, b ir.Inst) bool {
	aIO, bIO := isIO(a), isIO(b)
	return aIO && bIO ||
		aIO && canThrow(b) || bIO && canThrow(a) ||
		references(a, b) || references(b, a)
}

func isIO(node ir.Inst) bool {
	switch node.(type) {
	case *ir.PrintStmt, *ir.ReadExpr:
		return true
	}
	return false
}

// canThrow returns whether the node is a division with a non-constant
// RHS.
// TODO: create div trap to replace this.
func canThrow(node ir.Inst) bool {
	if bin, ok := node.(*ir.BinaryExpr); ok && bin.Op == ir.Div {
		_, ok := bin.Operand(1).Def.(*ir.IntConst)
		return !ok
	}
	return false
}

// references returns whether node B references the assignment of
// node A.
func references(a, b ir.Inst) bool {
	if expr, ok := a.(ir.Value); ok {
		if user, ok := b.(ir.User); ok {
			for _, operand := range user.Operands() {
				if operand.Def == expr {
					return true
				}
			}
		}
	}
	return false
}
