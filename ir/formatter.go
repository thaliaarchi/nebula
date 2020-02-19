package ir // import "github.com/andrewarchi/nebula/ir"

import (
	"fmt"
	"strings"
)

type formatter struct {
	ids    map[*ValueDef]int
	nextID int
}

func newFormatter() *formatter {
	return &formatter{
		ids:    make(map[*ValueDef]int),
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
	b.WriteString(f.FormatTerminator(block.Terminator))
	b.WriteByte('\n')
	return b.String()
}

func (f *formatter) FormatNode(node Node) string {
	switch n := node.(type) {
	case *PhiExpr:
		var b strings.Builder
		b.WriteString("phi [")
		for i, edge := range n.Edges {
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString(f.FormatUse(edge.Val))
			b.WriteByte(' ')
			b.WriteString(edge.Block.Name())
		}
		b.WriteByte(']')
		return b.String()
	case *BinaryExpr:
		return fmt.Sprintf("%s = %v %s %s", f.FormatDef(n.Def), n.Op, f.FormatUse(n.LHS), f.FormatUse(n.RHS))
	case *UnaryExpr:
		return fmt.Sprintf("%s = %v %s", f.FormatDef(n.Def), n.Op, f.FormatUse(n.Val))
	case *LoadStackExpr:
		return fmt.Sprintf("%s = loadstack %d", f.FormatDef(n.Def), n.Pos)
	case *LoadHeapExpr:
		return fmt.Sprintf("%s = load *%s", f.FormatDef(n.Def), f.FormatUse(n.Addr))
	case *StoreHeapStmt:
		return fmt.Sprintf("store *%s %s", f.FormatUse(n.Addr), f.FormatUse(n.Val))
	case *CheckStackStmt:
		return fmt.Sprintf("checkstack %d", n.Access)
	case *PrintStmt:
		return fmt.Sprintf("%v %s", n.Op, f.FormatUse(n.Val))
	case *ReadExpr:
		return fmt.Sprintf("%s = %v", f.FormatDef(n.Def), n.Op)
	case *FlushStmt:
		return "flush"
	default:
		panic("ir: unknown node type")
	}
}

func (f *formatter) FormatTerminator(term Terminator) string {
	switch t := term.(type) {
	case *CallTerm:
		return fmt.Sprintf("call %s", t.Dest.Name())
	case *JmpTerm:
		return fmt.Sprintf("%v %s", t.Op, t.Dest.Name())
	case *JmpCondTerm:
		return fmt.Sprintf("%v %s %s %s", t.Op, f.FormatUse(t.Cond), t.Then.Name(), t.Else.Name())
	case *RetTerm:
		return "ret"
	case *ExitTerm:
		return "exit"
	default:
		panic("ir: unknown terminator type")
	}
}

func (f *formatter) FormatVal(val Value) string {
	switch v := val.(type) {
	case *ConstVal:
		return v.Int.String()
	default:
		return f.FormatDef(v.ValueDef())
	}
}

func (f *formatter) FormatDef(def *ValueDef) string {
	var id int
	if vid, ok := f.ids[def]; ok {
		id = vid
	} else {
		id = f.nextID
		f.ids[def] = f.nextID
		f.nextID++
	}
	return fmt.Sprintf("%%%d", id)
}

func (f *formatter) FormatUse(use *ValueUse) string {
	return f.FormatVal(use.Val)
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
