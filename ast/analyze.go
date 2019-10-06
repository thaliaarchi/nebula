package ast

import (
	"math/big"

	"github.com/andrewarchi/wspace/bigint"
	"github.com/andrewarchi/wspace/token"
)

// Reduce accumulates sequences of nodes and replaces the starting node
// with the accumulation. A sequence of one node is not replaced.
func (block *BasicBlock) Reduce(fn func(acc, curr Node, i int) (Node, bool)) {
	k := 0
	for i := 0; i < len(block.Nodes); i++ {
		if acc, ok := fn(nil, block.Nodes[i], i); ok {
			i++
			concat := false
			for ; i < len(block.Nodes); i++ {
				if next, ok := fn(acc, block.Nodes[i], i); ok {
					acc = next
					concat = true
				} else {
					i--
					break
				}
			}

			if concat {
				block.Nodes[k] = acc
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

// ConcatStrings joins consecutive constant print expressions.
func (ast *AST) ConcatStrings() {
	for _, block := range ast.Blocks {
		block.Reduce(func(acc, curr Node, i int) (Node, bool) {
			if str, ok := checkPrint(curr); ok {
				if acc == nil {
					val := Val(&StringVal{str})
					return &PrintStmt{
						Op:  token.Prints,
						Val: &val,
					}, true
				}
				val := (*acc.(*PrintStmt).Val).(*StringVal)
				val.Val += str
				return acc, true
			}
			return nil, false
		})
	}
}

func checkPrint(node Node) (string, bool) {
	if p, ok := node.(*PrintStmt); ok {
		if val, ok := (*p.Val).(*ConstVal); ok {
			switch p.Op {
			case token.Printc:
				return string(bigint.ToRune(val.Val)), true
			case token.Printi:
				return val.Val.String(), true
			}
		}
		if val, ok := (*p.Val).(*StringVal); ok {
			if p.Op == token.Prints {
				return val.Val, true
			}
		}
	}
	return "", false
}

// FoldConstArith folds and propagates constant arithmetic expressions
// or identities.
func (ast *AST) FoldConstArith() {
	for _, block := range ast.Blocks {
		j := 0
		for i := 0; i < len(block.Nodes); i++ {
			if assign, ok := block.Nodes[i].(*AssignStmt); ok {
				if expr, ok := assign.Expr.(*ArithExpr); ok {
					if val, ok := expr.FoldConst(); ok {
						*assign.Assign = val
						continue
					}
				}
			}
			block.Nodes[j] = block.Nodes[i]
			j++
		}
		block.Nodes = block.Nodes[:j]
	}
}

// FoldConst reduces constant arithmetic expressions or identities.
func (expr *ArithExpr) FoldConst() (Val, bool) {
	if lhs, ok := (*expr.LHS).(*ConstVal); ok {
		if rhs, ok := (*expr.RHS).(*ConstVal); ok {
			return expr.foldConstLR(lhs.Val, rhs.Val)
		}
		return expr.foldConstL(lhs.Val)
	} else if rhs, ok := (*expr.RHS).(*ConstVal); ok {
		return expr.foldConstR(rhs.Val)
	}
	return expr.foldConst()
}

func (expr *ArithExpr) foldConstLR(lhs, rhs *big.Int) (Val, bool) {
	result := new(big.Int)
	switch expr.Op {
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
	return &ConstVal{result}, true
}

var bigOne = big.NewInt(1)

func (expr *ArithExpr) foldConstL(lhs *big.Int) (Val, bool) {
	if lhs.Sign() == 0 {
		switch expr.Op {
		case token.Add:
			return *expr.RHS, true
		case token.Sub:
			// negation
		case token.Mul, token.Div, token.Mod:
			return *expr.LHS, true
		}
	} else if lhs.Cmp(bigOne) == 0 {
		switch expr.Op {
		case token.Mul, token.Div:
			return *expr.RHS, true
		}
	}
	return nil, false
}

func (expr *ArithExpr) foldConstR(rhs *big.Int) (Val, bool) {
	if rhs.Sign() == 0 {
		switch expr.Op {
		case token.Add, token.Sub:
			return *expr.LHS, true
		case token.Mul:
			return *expr.RHS, true
		case token.Div, token.Mod:
			panic("ast: division by zero")
		}
	} else if rhs.Cmp(bigOne) == 0 {
		switch expr.Op {
		case token.Mul, token.Div:
			return *expr.LHS, true
		case token.Mod:
			return &ConstVal{big.NewInt(0)}, true
		}
	}
	return nil, false
}

func (expr *ArithExpr) foldConst() (Val, bool) {
	if expr.LHS == expr.RHS {
		switch expr.Op {
		case token.Sub, token.Mod:
			return &ConstVal{big.NewInt(0)}, true
		case token.Div:
			return &ConstVal{big.NewInt(1)}, true
		}
	}
	return nil, false
}
