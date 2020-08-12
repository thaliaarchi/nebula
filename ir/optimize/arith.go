// Package optimize analyzes and optimizes Nebula IR.
//
package optimize // import "github.com/andrewarchi/nebula/ir/optimize"

import (
	"fmt"
	"math/big"

	"github.com/andrewarchi/nebula/internal/bigint"
	"github.com/andrewarchi/nebula/ir"
)

// FoldConstArith folds and propagates constant arithmetic expressions
// or identities.
func FoldConstArith(p *ir.Program) {
	for _, block := range p.Blocks {
		i := 0
		for _, node := range block.Nodes {
			switch inst := node.(type) {
			case *ir.BinaryExpr:
				val, isNeg := foldBinaryExpr(p, inst)
				if isNeg {
					neg := ir.NewUnaryExpr(ir.Neg, val, inst.Pos())
					ir.ClearOperands(inst)
					ir.ReplaceUses(inst, neg)
					node = neg
				} else if val != nil {
					ir.ClearOperands(inst)
					ir.ReplaceUses(inst, val)
					continue
				}
			case *ir.UnaryExpr:
				if inst.Op == ir.Neg {
					val := ir.Operand(inst, 0).Def
					if lhs, ok := val.(*ir.IntConst); ok {
						constNeg := p.LookupConst(new(big.Int).Neg(lhs.Int), inst.Pos())
						ir.ClearOperands(inst)
						ir.ReplaceUses(inst, constNeg)
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
	ops := expr.Operands()
	_, lhsConst := ops[0].Def.(*ir.IntConst)
	_, rhsConst := ops[1].Def.(*ir.IntConst)
	switch {
	case lhsConst && rhsConst:
		return foldBinaryLR(p, expr)
	case lhsConst:
		return foldBinaryL(p, expr)
	case rhsConst:
		return foldBinaryR(p, expr)
	default:
		return foldBinary(p, expr)
	}
}

func foldBinaryLR(p *ir.Program, expr *ir.BinaryExpr) (ir.Value, bool) {
	ops := expr.Operands()
	lhs, rhs := ops[0].Def.(*ir.IntConst), ops[1].Def.(*ir.IntConst)
	result := new(big.Int)
	switch expr.Op {
	case ir.Add:
		result.Add(lhs.Int, rhs.Int)
	case ir.Sub:
		result.Sub(lhs.Int, rhs.Int)
	case ir.Mul:
		result.Mul(lhs.Int, rhs.Int)
	case ir.Div:
		result.Div(lhs.Int, rhs.Int)
	case ir.Mod:
		result.Mod(lhs.Int, rhs.Int)
	case ir.Shl:
		s, ok := bigint.ToUint(rhs.Int)
		if !ok {
			panic(fmt.Sprintf("optimize: shl rhs overflow: %v", rhs.Int))
		}
		result.Lsh(lhs.Int, s)
	case ir.LShr:
		return nil, false
	case ir.AShr:
		s, ok := bigint.ToUint(rhs.Int)
		if !ok {
			panic(fmt.Sprintf("optimize: ashr rhs overflow: %v", rhs.Int))
		}
		result.Rsh(lhs.Int, s)
	case ir.And:
		result.And(lhs.Int, rhs.Int)
	case ir.Or:
		result.Or(lhs.Int, rhs.Int)
	case ir.Xor:
		result.Xor(lhs.Int, rhs.Int)
	default:
		return nil, false
	}
	return p.LookupConst(result, expr.Pos()), false
}

var (
	bigZero   = big.NewInt(0)
	bigOne    = big.NewInt(1)
	bigNegOne = big.NewInt(-1)
)

func foldBinaryL(p *ir.Program, expr *ir.BinaryExpr) (ir.Value, bool) {
	ops := expr.Operands()
	lhs, rhs := ops[0].Def.(*ir.IntConst), ops[1].Def
	switch lhs.Int.Sign() {
	case 0:
		switch expr.Op {
		case ir.Add:
			return rhs, false
		case ir.Sub:
			return rhs, true
		case ir.Mul:
			return lhs, false
		case ir.Div, ir.Mod:
			// TODO trap if RHS zero
			return lhs, false
		}
	case 1:
		if expr.Op == ir.Mul && lhs.Int.Cmp(bigOne) == 0 {
			return rhs, false
		}
	case -1:
		if expr.Op == ir.Mul && lhs.Int.Cmp(bigNegOne) == 0 {
			return rhs, true
		}
	}
	return nil, false
}

func foldBinaryR(p *ir.Program, expr *ir.BinaryExpr) (ir.Value, bool) {
	ops := expr.Operands()
	lhs, rhs := ops[0].Def, ops[1].Def.(*ir.IntConst)
	switch rhs.Int.Sign() {
	case 0:
		switch expr.Op {
		case ir.Add, ir.Sub:
			return lhs, false
		case ir.Mul:
			return rhs, false
		case ir.Div, ir.Mod:
			panic("optimize: divide by zero")
		}
	case 1:
		if rhs.Int.Cmp(bigOne) == 0 {
			switch expr.Op {
			case ir.Mul, ir.Div:
				return lhs, false
			case ir.Mod:
				return p.LookupConst(bigZero, expr.Pos()), false
			}
		} else if ntz := rhs.Int.TrailingZeroBits(); uint(rhs.Int.BitLen()) == ntz+1 {
			switch expr.Op {
			case ir.Mul:
				expr.Op = ir.Shl
				ops[1].SetDef(p.LookupConst(new(big.Int).SetUint64(uint64(ntz)), expr.Pos()))
			case ir.Div:
				expr.Op = ir.AShr
				ops[1].SetDef(p.LookupConst(new(big.Int).SetUint64(uint64(ntz)), expr.Pos()))
			case ir.Mod:
				expr.Op = ir.And
				ops[1].SetDef(p.LookupConst(new(big.Int).Sub(rhs.Int, bigOne), expr.Pos()))
			}
			return nil, false // overwrite op
		}
	case -1:
		if rhs.Int.Cmp(bigNegOne) == 0 {
			switch expr.Op {
			case ir.Mul, ir.Div:
				return lhs, true
			case ir.Mod:
				return p.LookupConst(bigZero, expr.Pos()), false
			}
		}
	}
	return nil, false
}

func foldBinary(p *ir.Program, expr *ir.BinaryExpr) (ir.Value, bool) {
	ops := expr.Operands()
	lhs, rhs := ops[0], ops[1]
	if lhs.Def == rhs.Def {
		switch expr.Op {
		case ir.Sub:
			return p.LookupConst(bigZero, expr.Pos()), false
		case ir.Div:
			// TODO trap if RHS zero
			return p.LookupConst(bigOne, expr.Pos()), false
		case ir.Mod:
			// TODO trap if RHS zero
			return p.LookupConst(bigZero, expr.Pos()), false
		}
	}
	return nil, false
}
