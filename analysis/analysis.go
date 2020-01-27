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
					val := ir.Val(&ir.StringVal{Str: str})
					return &ir.PrintStmt{
						Op:  ir.Prints,
						Val: &val,
					}, true
				}
				val := (*acc.(*ir.PrintStmt).Val).(*ir.StringVal)
				val.Str += str
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
				return string(bigint.ToRune(val.Int)), true
			case ir.Printi:
				return val.Int.String(), true
			}
		}
		if val, ok := (*p.Val).(*ir.StringVal); ok {
			if p.Op == ir.Prints {
				return val.Str, true
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
		case ir.Mul, ir.Div, ir.Mod:
			return expr.LHS, false
		}
	} else if lhs.Cmp(bigOne) == 0 {
		switch expr.Op {
		case ir.Mul, ir.Div:
			return expr.RHS, false
		}
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
			panic("ir: division by zero")
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
		}
	}
	return nil, false
}

func foldBinary(p *ir.Program, expr *ir.BinaryExpr) (*ir.Val, bool) {
	if ir.ValEq(expr.LHS, expr.RHS) {
		switch expr.Op {
		case ir.Sub:
			return p.LookupConst(bigZero), false
		case ir.Mod:
			// TODO trap if zero
			return p.LookupConst(bigZero), false
		case ir.Div:
			// TODO trap if zero
			return p.LookupConst(bigOne), false
		}
	}
	return nil, false
}
