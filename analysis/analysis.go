package analysis // import "github.com/andrewarchi/nebula/analysis"

import (
	"math/big"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
)

// ReduceBlock accumulates sequences of nodes and replaces the starting
// node with the accumulation. A sequence of one node is not replaced.
func ReduceBlock(block *ir.BasicBlock, fn func(acc, curr ir.Node, i int) (ir.Node, bool)) {
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
func ConcatStrings(p *ir.Program) {
	for _, block := range p.Blocks {
		ReduceBlock(block, func(acc, curr ir.Node, i int) (ir.Node, bool) {
			if str, ok := checkPrint(curr); ok {
				if acc == nil {
					val := ir.Val(&ir.StringVal{Val: str})
					return &ir.PrintStmt{
						Op:  ir.Prints,
						Val: &val,
					}, true
				}
				val := (*acc.(*ir.PrintStmt).Val).(*ir.StringVal)
				val.Val += str
				return acc, true
			}
			return nil, false
		})
	}
}

func checkPrint(node ir.Node) (string, bool) {
	if p, ok := node.(*ir.PrintStmt); ok {
		if val, ok := (*p.Val).(*ir.ConstVal); ok {
			switch p.Op {
			case ir.Printc:
				return string(bigint.ToRune(val.Val)), true
			case ir.Printi:
				return val.Val.String(), true
			}
		}
		if val, ok := (*p.Val).(*ir.StringVal); ok {
			if p.Op == ir.Prints {
				return val.Val, true
			}
		}
	}
	return "", false
}

// FoldConstArith folds and propagates constant arithmetic expressions
// or identities.
func FoldConstArith(p *ir.Program) {
	for _, block := range p.Blocks {
		j := 0
		for i := 0; i < len(block.Nodes); i++ {
			if expr, ok := block.Nodes[i].(*ir.ArithExpr); ok {
				if val, ok := FoldConst(p, expr); ok {
					*expr.Assign = *val
					continue
				}
			}
			block.Nodes[j] = block.Nodes[i]
			j++
		}
		block.Nodes = block.Nodes[:j]
	}
}

// FoldConst reduces constant arithmetic expressions or identities.
func FoldConst(p *ir.Program, expr *ir.ArithExpr) (*ir.Val, bool) {
	if expr.Op == ir.Neg {
		if lhs, ok := (*expr.LHS).(*ir.ConstVal); ok {
			return p.LookupConst(new(big.Int).Neg(lhs.Val)), true
		}
		return nil, false
	}
	if lhs, ok := (*expr.LHS).(*ir.ConstVal); ok {
		if rhs, ok := (*expr.RHS).(*ir.ConstVal); ok {
			return foldConstLR(p, expr, lhs.Val, rhs.Val)
		}
		return foldConstL(p, expr, lhs.Val)
	} else if rhs, ok := (*expr.RHS).(*ir.ConstVal); ok {
		return foldConstR(p, expr, rhs.Val)
	}
	return foldConst(p, expr)
}

func foldConstLR(p *ir.Program, expr *ir.ArithExpr, lhs, rhs *big.Int) (*ir.Val, bool) {
	result := new(big.Int)
	switch expr.Op {
	case ir.Add:
		result.Add(lhs, rhs)
	case ir.Sub:
		result.Sub(lhs, rhs)
	case ir.Mul:
		result.Mul(lhs, rhs)
	case ir.Div:
		result.Div(lhs, rhs)
	case ir.Mod:
		result.Mod(lhs, rhs)
	}
	return p.LookupConst(result), true
}

var (
	bigZero   = big.NewInt(0)
	bigOne    = big.NewInt(1)
	bigNegOne = big.NewInt(-1)
)

func foldConstL(p *ir.Program, expr *ir.ArithExpr, lhs *big.Int) (*ir.Val, bool) {
	if lhs.Sign() == 0 {
		switch expr.Op {
		case ir.Add:
			return expr.RHS, true
		case ir.Sub:
			expr.Op, expr.LHS, expr.RHS = ir.Neg, expr.RHS, nil
		case ir.Mul, ir.Div, ir.Mod:
			return expr.LHS, true
		}
	} else if lhs.Cmp(bigOne) == 0 {
		switch expr.Op {
		case ir.Mul, ir.Div:
			return expr.RHS, true
		}
	} else if expr.Op == ir.Mul && lhs.Cmp(bigNegOne) == 0 {
		expr.Op, expr.LHS, expr.RHS = ir.Neg, expr.RHS, nil
	}
	return nil, false
}

func foldConstR(p *ir.Program, expr *ir.ArithExpr, rhs *big.Int) (*ir.Val, bool) {
	if rhs.Sign() == 0 {
		switch expr.Op {
		case ir.Add, ir.Sub:
			return expr.LHS, true
		case ir.Mul:
			return expr.RHS, true
		case ir.Div, ir.Mod:
			panic("ir: division by zero")
		}
	} else if rhs.Cmp(bigOne) == 0 {
		switch expr.Op {
		case ir.Mul, ir.Div:
			return expr.LHS, true
		case ir.Mod:
			return p.LookupConst(bigZero), true
		}
	} else if rhs.Cmp(bigNegOne) == 0 {
		switch expr.Op {
		case ir.Mul, ir.Div:
			expr.Op, expr.RHS = ir.Neg, nil
		}
	}
	return nil, false
}

func foldConst(p *ir.Program, expr *ir.ArithExpr) (*ir.Val, bool) {
	if ir.ValEq(expr.LHS, expr.RHS) {
		switch expr.Op {
		case ir.Sub:
			return p.LookupConst(bigZero), true
		case ir.Mod:
			// TODO: trap if zero
			return p.LookupConst(bigZero), true
		case ir.Div:
			// TODO: trap if zero
			return p.LookupConst(bigOne), true
		}
	}
	return nil, false
}
