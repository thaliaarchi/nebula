package ast

import (
	"github.com/andrewarchi/graph"
	"github.com/andrewarchi/wspace/token"
)

// FlowDependenceGraph creates a directed graph with edges representing
// the connections between basic blocks.
func (ast *AST) FlowDependenceGraph() graph.Graph {
	ids := make(map[*BasicBlock]uint)
	for _, block := range ast.Blocks {
		ids[block] = uint(block.ID)
	}
	g := graph.NewGraph(uint(len(ast.Blocks)))
	for i, block := range ast.Blocks {
		switch exit := block.Exit.(type) {
		case *CallStmt:
			g.Add(uint(i), ids[exit.Callee])
		case *JmpStmt:
			g.Add(uint(i), ids[exit.Block])
		case *JmpCondStmt:
			g.Add(uint(i), ids[exit.TrueBlock])
			g.Add(uint(i), ids[exit.FalseBlock])
		case *RetStmt:
			for _, caller := range block.Callers {
				g.Add(uint(i), ids[caller])
			}
		case *EndStmt:
		}
	}
	return g
}

// DependenceGraph creates an undirected graph with edges representing
// dependencies between nodes.
func (block *BasicBlock) DependenceGraph() graph.Graph {
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
func Dependent(a, b Node) bool {
	aIO, bIO := isIO(a), isIO(b)
	return aIO && bIO ||
		aIO && canThrow(b) || bIO && canThrow(a) ||
		references(a, b) || references(b, a)
}

func isIO(node Node) bool {
	switch node.(type) {
	case *PrintStmt, *ReadExpr:
		return true
	}
	return false
}

// canThrow returns whether the node is a division with a non-constant
// RHS.
func canThrow(node Node) bool {
	if n, ok := node.(*ArithExpr); ok && n.Op == token.Div {
		_, ok := (*n.RHS).(*ConstVal)
		return !ok
	}
	return false
}

// references returns whether node B references the assignment of
// node A.
func references(a, b Node) bool {
	if assignA, ok := a.(*AssignStmt); ok {
		assign := assignA.Assign
		if assignB, ok := b.(*AssignStmt); ok {
			if assignB.Assign == assign {
				return false
			}
			b = assignB.Expr
		}
		switch expr := b.(type) {
		case *ArithExpr:
			return expr.LHS == assign || expr.RHS == assign
		case *HeapExpr:
			return expr.Val == assign
		case *PrintStmt:
			return expr.Val == assign
		case *ReadExpr:
		}
	}
	return false
}
