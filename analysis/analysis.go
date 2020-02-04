package analysis // import "github.com/andrewarchi/nebula/analysis"

import (
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
			default:
				panic("analysis: unrecognized print op")
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
