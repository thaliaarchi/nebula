package analysis // import "github.com/andrewarchi/nebula/analysis"

import (
	"fmt"
	"math/big"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/ir"
)

// FoldConstArith folds and propagates constant arithmetic expressions
// or identities.
func FoldConstArith(p *ir.Program) {
	for _, block := range p.Blocks {
		i := 0
		for _, node := range block.Nodes {
			switch n := node.(type) {
			case *ir.BinaryExpr:
				val, neg := foldBinaryExpr(p, n)
				if neg {
					expr := &ir.UnaryExpr{Op: ir.Neg}
					ir.AddUse(val, expr, 0)
					n.LHS.Remove()
					n.RHS.Remove()
					n.Def.ReplaceSelf(expr)
					node = expr
				} else if val != nil {
					n.LHS.Remove()
					n.RHS.Remove()
					n.Def.ReplaceSelf(val)
					continue
				}
			case *ir.UnaryExpr:
				if n.Op == ir.Neg {
					if lhs, ok := n.Val.Val.(*ir.ConstVal); ok {
						n.Val.Remove()
						n.Def.ReplaceSelf(p.LookupConst(new(big.Int).Neg(lhs.Int)))
						continue
					}
				}
			}
			block.Nodes[i] = node
			i++
		}
		block.Nodes = block.Nodes[:i]
	}
}

func foldBinaryExpr(p *ir.Program, expr *ir.BinaryExpr) (ir.Value, bool) {
	if lhs, ok := expr.LHS.Val.(*ir.ConstVal); ok {
		if rhs, ok := expr.RHS.Val.(*ir.ConstVal); ok {
			return foldBinaryLR(p, expr, lhs.Int, rhs.Int)
		}
		return foldBinaryL(p, expr, lhs.Int)
	} else if rhs, ok := expr.RHS.Val.(*ir.ConstVal); ok {
		return foldBinaryR(p, expr, rhs.Int)
	}
	return foldBinary(p, expr)
}

func foldBinaryLR(p *ir.Program, expr *ir.BinaryExpr, lhs, rhs *big.Int) (ir.Value, bool) {
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
	case ir.Shl:
		s, ok := bigint.ToUint(rhs)
		if !ok {
			panic(fmt.Sprintf("analysis: shl rhs overflow: %v", rhs))
		}
		result.Lsh(lhs, s)
	case ir.LShr:
		return nil, false
	case ir.AShr:
		s, ok := bigint.ToUint(rhs)
		if !ok {
			panic(fmt.Sprintf("analysis: ashr rhs overflow: %v", rhs))
		}
		result.Rsh(lhs, s)
	case ir.And:
		result.And(lhs, rhs)
	case ir.Or:
		result.Or(lhs, rhs)
	case ir.Xor:
		result.Xor(lhs, rhs)
	default:
		return nil, false
	}
	return p.LookupConst(result), false
}

var (
	bigZero   = big.NewInt(0)
	bigOne    = big.NewInt(1)
	bigNegOne = big.NewInt(-1)
)

func foldBinaryL(p *ir.Program, expr *ir.BinaryExpr, lhs *big.Int) (ir.Value, bool) {
	switch lhs.Sign() {
	case 0:
		switch expr.Op {
		case ir.Add:
			return expr.RHS.Val, false
		case ir.Sub:
			return expr.RHS.Val, true
		case ir.Mul:
			return expr.LHS.Val, false
		case ir.Div, ir.Mod:
			// TODO trap if RHS zero
			return expr.LHS.Val, false
		}
	case 1:
		if expr.Op == ir.Mul && lhs.Cmp(bigOne) == 0 {
			return expr.RHS.Val, false
		}
	case -1:
		if expr.Op == ir.Mul && lhs.Cmp(bigNegOne) == 0 {
			return expr.RHS.Val, true
		}
	}
	return nil, false
}

func foldBinaryR(p *ir.Program, expr *ir.BinaryExpr, rhs *big.Int) (ir.Value, bool) {
	switch rhs.Sign() {
	case 0:
		switch expr.Op {
		case ir.Add, ir.Sub:
			return expr.LHS.Val, false
		case ir.Mul:
			return expr.RHS.Val, false
		case ir.Div, ir.Mod:
			panic("analysis: divide by zero")
		}
	case 1:
		if rhs.Cmp(bigOne) == 0 {
			switch expr.Op {
			case ir.Mul, ir.Div:
				return expr.LHS.Val, false
			case ir.Mod:
				return p.LookupConst(bigZero), false
			}
		} else if ntz := rhs.TrailingZeroBits(); uint(rhs.BitLen()) == ntz+1 {
			switch expr.Op {
			case ir.Mul:
				expr.Op = ir.Shl
				expr.RHS.ReplaceVal(p.LookupConst(new(big.Int).SetUint64(uint64(ntz))))
			case ir.Div:
				expr.Op = ir.AShr
				expr.RHS.ReplaceVal(p.LookupConst(new(big.Int).SetUint64(uint64(ntz))))
			case ir.Mod:
				expr.Op = ir.And
				expr.RHS.ReplaceVal(p.LookupConst(new(big.Int).Sub(rhs, bigOne)))
			}
			return nil, false
		}
	case -1:
		if rhs.Cmp(bigNegOne) == 0 {
			switch expr.Op {
			case ir.Mul, ir.Div:
				return expr.LHS.Val, true
			case ir.Mod:
				return p.LookupConst(bigZero), false
			}
		}
	}
	return nil, false
}

func foldBinary(p *ir.Program, expr *ir.BinaryExpr) (ir.Value, bool) {
	if expr.LHS.Val == expr.RHS.Val {
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
