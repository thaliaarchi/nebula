package analysis // import "github.com/andrewarchi/nebula/analysis"

import (
	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/nebula/ir"
	"github.com/andrewarchi/nebula/token"
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
		for _, exit := range block.Exits() {
			g.Add(uint(i), uint(ids[exit]))
		}
	}
	return g
}

// DependenceGraph creates an undirected graph with edges representing
// dependencies between nodes.
func DependenceGraph(block *ir.BasicBlock) graph.Graph {
	g := graph.NewGraph(uint(len(block.Nodes)))
	for i := range block.Nodes {
		for j := i + 1; j < len(block.Nodes); j++ {
			if Dependent(block.Nodes[i], block.Nodes[j]) {
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
func Dependent(a, b ir.Node) bool {
	aIO, bIO := isIO(a), isIO(b)
	return aIO && bIO ||
		aIO && canThrow(b) || bIO && canThrow(a) ||
		references(a, b) || references(b, a)
}

func isIO(node ir.Node) bool {
	switch node.(type) {
	case *ir.PrintStmt, *ir.ReadExpr:
		return true
	}
	return false
}

// canThrow returns whether the node is a division with a non-constant
// RHS.
func canThrow(node ir.Node) bool {
	if n, ok := node.(*ir.ArithExpr); ok && n.Op == token.Div {
		_, ok := (*n.RHS).(*ir.ConstVal)
		return !ok
	}
	return false
}

// references returns whether node B references the assignment of
// node A.
func references(a, b ir.Node) bool {
	if assignA, ok := a.(*ir.AssignStmt); ok {
		assign := assignA.Assign
		if assignB, ok := b.(*ir.AssignStmt); ok {
			if assignB.Assign == assign {
				return false
			}
			b = assignB.Expr
		}
		switch expr := b.(type) {
		case *ir.ArithExpr:
			return expr.LHS == assign || expr.RHS == assign
		case *ir.StoreExpr:
			return expr.Addr == assign || expr.Val == assign
		case *ir.RetrieveExpr:
			return expr.Addr == assign
		case *ir.PrintStmt:
			return expr.Val == assign
		case *ir.ReadExpr:
		}
	}
	return false
}
