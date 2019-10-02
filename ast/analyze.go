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
				concat := false
				for ; i < len(block.Nodes); i++ {
					if s, ok := checkPrint(block.Nodes[i]); ok {
						str += s
						concat = true
					} else {
						break
					}
				}

				if concat {
					block.Nodes[k] = &PrintStmt{
						Op:  token.Prints,
						Val: &StringVal{str},
					}
					k++
					continue
				}
			}

			if i < len(block.Nodes) {
				block.Nodes[k] = block.Nodes[i]
				k++
			}
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

func (ast AST) ConstArith() {
	for _, block := range ast {
		for i, node := range block.Nodes {
			if bin, ok := node.(*BinaryExpr); ok {
				if !bin.Op.IsArith() {
					continue
				}
				if lhs, ok := bin.LHS.(*ConstVal); ok {
					if rhs, ok := bin.RHS.(*ConstVal); ok {
						block.Nodes[i] = constArith(bin, lhs.Val, rhs.Val)
					} else {
						if n := constArithLHS(bin, lhs.Val, bin.RHS); n != nil {
							block.Nodes[i] = n
						}
					}
				} else if rhs, ok := bin.RHS.(*ConstVal); ok {
					if n := constArithRHS(bin, lhs, rhs.Val); n != nil {
						block.Nodes[i] = n
					}
				}
			}
		}
	}
}

func constArith(node *BinaryExpr, lhs, rhs *big.Int) Node {
	result := new(big.Int)
	switch node.Op {
	case token.Add:
		result.Add(lhs, rhs)
	case token.Sub:
		result.Sub(lhs, rhs)
	case token.Mul:
		result.Mul(lhs, rhs)
	case token.Div:
		result.Div(lhs, rhs)
	case token.Mod:
		result.Mod(lhs, rhs)
	}
	return &UnaryExpr{
		Op:     token.Push,
		Assign: node.Assign,
		Val:    &ConstVal{result},
	}
}

var bigOne = new(big.Int).SetInt64(1)

func constArithRHS(node *BinaryExpr, lhs Val, rhs *big.Int) Node {
	var val Val
	if rhs.Sign() == 0 {
		switch node.Op {
		case token.Add, token.Sub:
			val = lhs
		case token.Mul:
			val = &ConstVal{new(big.Int).SetInt64(0)}
		case token.Div:
			panic("ast: division by zero")
		}
	} else if rhs.Cmp(bigOne) == 0 {
		switch node.Op {
		case token.Mul, token.Div:
			val = lhs
		}
	}
	if val == nil {
		return nil
	}
	return &UnaryExpr{
		Op:     token.Push,
		Assign: node.Assign,
		Val:    val,
	}
}

func constArithLHS(node *BinaryExpr, lhs *big.Int, rhs Val) Node {
	var val Val
	if lhs.Sign() == 0 {
		switch node.Op {
		case token.Add:
			val = rhs
		case token.Sub:
			return &UnaryExpr{
				Op:     token.Neg,
				Assign: node.Assign,
				Val:    rhs,
			}
		case token.Mul, token.Div:
			val = &ConstVal{new(big.Int).SetInt64(0)}
		}
	} else if lhs.Cmp(bigOne) == 0 {
		switch node.Op {
		case token.Mul, token.Div:
			val = rhs
		}
	}
	if val == nil {
		return nil
	}
	return &UnaryExpr{
		Op:     token.Push,
		Assign: node.Assign,
		Val:    val,
	}
}
