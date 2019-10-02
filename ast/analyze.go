package ast

import (
	"math/big"

	"github.com/andrewarchi/wspace/bigint"
	"github.com/andrewarchi/wspace/token"
)

// InlineStackConstants eliminates push instructions and inlines
// constants.
func (ast AST) InlineStackConstants() {
	for _, block := range ast {
		constants := make(map[int]*big.Int)
		j := 0
		for i := range block.Nodes {
			if node, ok := block.Nodes[i].(*UnaryExpr); ok && node.Op == token.Push {
				assign := node.Assign.(*StackVal).Val
				val := node.Val.(*ConstVal).Val
				constants[assign] = val
			} else {
				inlineConstants(&block.Nodes[i], constants)
				block.Nodes[j] = block.Nodes[i]
				j++
			}
		}
		block.Nodes = block.Nodes[:j]
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
	case *PrintStmt:
		inlineConstants(&n.Val, constants)
	case *ReadExpr:
		inlineConstants(&n.Assign, constants)
	case *JmpCondStmt:
		inlineConstants(&n.Val, constants)
	}
}

// ConcatStrings joins consecutive constant print expressions.
func (ast AST) ConcatStrings() {
	for _, block := range ast {
		k := 0
		for i := 0; i < len(block.Nodes); i++ {
			if str, ok := checkPrint(block.Nodes[i]); ok {
				i++
				start := i
				for ; i < len(block.Nodes); i++ {
					if s, ok := checkPrint(block.Nodes[i]); ok {
						str += s
					} else {
						break
					}
				}
				if i > start {
					block.Nodes[k] = &PrintStmt{
						Op:  token.Prints,
						Val: &StringVal{str},
					}
					k++
					continue
				}
			}
			block.Nodes[k] = block.Nodes[i]
			k++
		}
		block.Nodes = block.Nodes[:k]
	}
}

func checkPrint(node Node) (string, bool) {
	if p, ok := node.(*PrintStmt); ok {
		if val, ok := p.Val.(*ConstVal); ok {
			switch p.Op {
			case token.Printc:
				return string(bigint.ToRune(val.Val)), true
			case token.Printi:
				return val.Val.String(), true
			}
		}
		if val, ok := p.Val.(*StringVal); ok {
			if p.Op == token.Prints {
				return val.Val, true
			}
		}
	}
	return "", false
}
