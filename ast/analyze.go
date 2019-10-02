package ast

import (
	"math/big"

	"github.com/andrewarchi/wspace/token"
)

// InlineStackConstants eliminates push instructions and inlines
// constants.
func (ast AST) InlineStackConstants() {
	for _, block := range ast {
		constants := make(map[int]*big.Int)
		for i := 0; i < len(block.Nodes); i++ {
			if node, ok := block.Nodes[i].(*UnaryExpr); ok && node.Op == token.Push {
				assign := node.Assign.(*StackVal).Val
				val := node.Val.(*ConstVal).Val
				constants[assign] = val
				copy(block.Nodes[i:], block.Nodes[i+1:])
				block.Nodes = block.Nodes[:len(block.Nodes)-1]
				i--
			} else {
				inlineConstants(&block.Nodes[i], constants)
			}
		}
		inlineConstants(&block.Edge, constants)
	}
}

func inlineConstants(node *Node, constants map[int]*big.Int) {
	switch n := (*node).(type) {
	case *StackVal:
		if c, ok := constants[n.Val]; ok {
			*node = &ConstVal{c}
		}
	case *AddrVal:
		inlineConstants(&n.Val, constants)
	case *UnaryExpr:
		inlineConstants(&n.Assign, constants)
		inlineConstants(&n.Val, constants)
	case *BinaryExpr:
		inlineConstants(&n.Assign, constants)
		inlineConstants(&n.LHS, constants)
		inlineConstants(&n.RHS, constants)
	case *IOStmt:
		inlineConstants(&n.Val, constants)
	case *JmpCondStmt:
		inlineConstants(&n.Val, constants)
	}
}
