package analysis // import "github.com/andrewarchi/nebula/analysis"

import (
	"math/big"

	"github.com/andrewarchi/nebula/ir"
)

// FoldConstArith folds and propagates constant arithmetic expressions
// or identities.
func FoldConstArith(p *ir.Program) {
	for _, block := range p.Blocks {
		j := 0
		for i := 0; i < len(block.Nodes); i++ {
			node := block.Nodes[i]
			switch n := node.(type) {
			case *ir.BinaryExpr:
				val, neg := foldBinaryExpr(p, n)
				if neg {
					node = &ir.UnaryExpr{Op: ir.Neg, Assign: n.Assign, Val: val}
				} else if val != nil {
					*n.Assign = *val
					continue
				}
			case *ir.UnaryExpr:
				if n.Op == ir.Neg {
					if lhs, ok := (*n.Val).(*ir.ConstVal); ok {
						*n.Assign = *p.LookupConst(new(big.Int).Neg(lhs.Int))
						continue
					}
				}
			}
			block.Nodes[j] = node
			j++
		}
		block.Nodes = block.Nodes[:j]
	}
}

func foldBinaryExpr(p *ir.Program, expr *ir.BinaryExpr) (*ir.Val, bool) {
	if lhs, ok := (*expr.LHS).(*ir.ConstVal); ok {
		if rhs, ok := (*expr.RHS).(*ir.ConstVal); ok {
			return foldBinaryLR(p, expr, lhs.Int, rhs.Int)
		}
		return foldBinaryL(p, expr, lhs.Int)
	} else if rhs, ok := (*expr.RHS).(*ir.ConstVal); ok {
		return foldBinaryR(p, expr, rhs.Int)
	}
	return foldBinary(p, expr)
}

func foldBinaryLR(p *ir.Program, expr *ir.BinaryExpr, lhs, rhs *big.Int) (*ir.Val, bool) {
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
	return p.LookupConst(result), false
}

var (
	bigZero   = big.NewInt(0)
	bigOne    = big.NewInt(1)
	bigNegOne = big.NewInt(-1)
)

func foldBinaryL(p *ir.Program, expr *ir.BinaryExpr, lhs *big.Int) (*ir.Val, bool) {
	if lhs.Sign() == 0 {
		switch expr.Op {
		case ir.Add:
			return expr.RHS, false
		case ir.Sub:
			return expr.RHS, true
		case ir.Mul:
			return expr.LHS, false
		case ir.Div, ir.Mod:
			// TODO trap if RHS zero
			return expr.LHS, false
		}
	} else if expr.Op == ir.Mul && lhs.Cmp(bigOne) == 0 {
		return expr.RHS, false
	} else if expr.Op == ir.Mul && lhs.Cmp(bigNegOne) == 0 {
		return expr.RHS, true
	}
	return nil, false
}

func foldBinaryR(p *ir.Program, expr *ir.BinaryExpr, rhs *big.Int) (*ir.Val, bool) {
	if rhs.Sign() == 0 {
		switch expr.Op {
		case ir.Add, ir.Sub:
			return expr.LHS, false
		case ir.Mul:
			return expr.RHS, false
		case ir.Div, ir.Mod:
			panic("analysis: divide by zero")
		}
	} else if rhs.Cmp(bigOne) == 0 {
		switch expr.Op {
		case ir.Mul, ir.Div:
			return expr.LHS, false
		case ir.Mod:
			return p.LookupConst(bigZero), false
		}
	} else if rhs.Cmp(bigNegOne) == 0 {
		switch expr.Op {
		case ir.Mul, ir.Div:
			return expr.LHS, true
		case ir.Mod:
			return p.LookupConst(bigZero), false
		}
	} else if expr.Op == ir.Mod {
		x := new(big.Int).Sub(rhs, bigOne)
		y := new(big.Int).And(rhs, x)
		if y.Sign() == 0 { // v & (v - 1) == 0
			expr.Op = ir.And
			expr.RHS = p.LookupConst(x)
		}
	}
	return nil, false
}

func foldBinary(p *ir.Program, expr *ir.BinaryExpr) (*ir.Val, bool) {
	if ir.ValEq(expr.LHS, expr.RHS) {
		switch expr.Op {
		case ir.Sub:
			return p.LookupConst(bigZero), false
		case ir.Div:
			// TODO trap if RHS zero
			return p.LookupConst(bigOne), false
		case ir.Mod:
			// TODO trap if RHS zero
			return p.LookupConst(bigZero), false
		}
	}
	return nil, false
}