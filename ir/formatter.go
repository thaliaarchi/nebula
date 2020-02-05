package ir // import "github.com/andrewarchi/nebula/ir"

import (
	"fmt"
	"strings"
)

type formatter struct {
	ids    map[*SSAVal]int
	nextID int
}

func newFormatter() *formatter {
	return &formatter{
		ids:    make(map[*SSAVal]int),
		nextID: 0,
	}
}

func (f *formatter) FormatProgram(p *Program) string {
	var b strings.Builder
	for i, block := range p.Blocks {
		if i != 0 {
			b.WriteByte('\n')
		}
		b.WriteString(f.FormatBlock(block))
	}
	return b.String()
}

func (f *formatter) FormatBlock(block *BasicBlock) string {
	var b strings.Builder
	if len(block.Labels) == 0 {
		fmt.Fprintf(&b, "block_%d:\n", block.ID)
	}
	for _, label := range block.Labels {
		b.WriteString(label.String())
		b.WriteString(":\n")
	}

	fmt.Fprintf(&b, "    ; entries: %s\n", formatBlockSlice(block.Entries))
	fmt.Fprintf(&b, "    ; callers: %s\n", formatBlockSlice(block.Callers))
	if len(block.Returns) != 0 {
		fmt.Fprintf(&b, "    ; returns: %s\n", formatBlockSlice(block.Returns))
	}

	if block.Stack.Access > 0 {
		fmt.Fprintf(&b, "    access %d [", block.Stack.Access)
		first := true
		for _, val := range block.Stack.Under {
			if val != nil {
				if !first {
					b.WriteByte(' ')
				}
				b.WriteString(f.FormatVal(val))
				first = false
			}
		}
		b.WriteString("]\n")
	}

	for _, node := range block.Nodes {
		b.WriteString("    ")
		b.WriteString(f.FormatNode(node))
		b.WriteByte('\n')
	}

	if block.Stack.Pops > 0 {
		fmt.Fprintf(&b, "    pop %d\n", block.Stack.Pops)
	}
	if len(block.Stack.Vals) != 0 {
		b.WriteString("    push [")
		for i, val := range block.Stack.Vals {
			if i != 0 {
				b.WriteByte(' ')
			}
			b.WriteString(f.FormatVal(val))
		}
		b.WriteString("]\n")
	}

	b.WriteString("    ")
	b.WriteString(f.FormatNode(block.Terminator))
	b.WriteByte('\n')
	return b.String()
}

func (f *formatter) FormatNode(node Node) string {
	switch n := node.(type) {
	case *BinaryExpr:
		return fmt.Sprintf("%s = %v %s %s", f.FormatVal(n.Assign), n.Op, f.FormatVal(n.LHS), f.FormatVal(n.RHS))
	case *UnaryExpr:
		return fmt.Sprintf("%s = %v %s", f.FormatVal(n.Assign), n.Op, f.FormatVal(n.Val))
	case *LoadExpr:
		return fmt.Sprintf("%s = load *%s", f.FormatVal(n.Assign), f.FormatVal(n.Addr))
	case *StoreStmt:
		return fmt.Sprintf("store *%s %s", f.FormatVal(n.Addr), f.FormatVal(n.Val))
	case *PrintStmt:
		return fmt.Sprintf("%v %s", n.Op, f.FormatVal(n.Val))
	case *ReadExpr:
		return fmt.Sprintf("%s = %v", f.FormatVal(n.Assign), n.Op)
	case *FlushStmt:
		return "flush"
	case *CallStmt:
		return fmt.Sprintf("call %s", n.Dest.Name())
	case *JmpStmt:
		return fmt.Sprintf("%v %s", n.Op, n.Dest.Name())
	case *JmpCondStmt:
		return fmt.Sprintf("%v %s %s %s", n.Op, f.FormatVal(n.Cond), n.Then.Name(), n.Else.Name())
	case *RetStmt:
		return "ret"
	case *ExitStmt:
		return "exit"
	default:
		panic("ir: unrecognized node type")
	}
}

func (f *formatter) FormatVal(val *Val) string {
	switch v := (*val).(type) {
	case *SSAVal:
		var id int
		if vid, ok := f.ids[v]; ok {
			id = vid
		} else {
			id = f.nextID
			f.ids[v] = f.nextID
			f.nextID++
		}
		return fmt.Sprintf("%%%d", id)
	case *StackVal:
		return fmt.Sprintf("%%s%d", v.Pos)
	case *ConstVal:
		return v.Int.String()
	case *PhiVal:
		var b strings.Builder
		b.WriteString("phi [")
		for i, ref := range v.Refs {
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString(f.FormatVal(ref.Val))
			b.WriteByte(' ')
			b.WriteString(ref.Block.Name())
		}
		b.WriteByte(']')
		return b.String()
	default:
		panic("ir: unrecognized val type")
	}
}
func formatBlockSlice(blocks []*BasicBlock) string {
	if len(blocks) == 0 {
		return "-"
	}
	var b strings.Builder
	for i, block := range blocks {
		if i != 0 {
			b.WriteByte(' ')
		}
		if block == nil {
			b.WriteString("<entry>")
		} else {
			b.WriteString(block.Name())
		}
	}
	return b.String()
}
