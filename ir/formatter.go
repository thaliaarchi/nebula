package ir // import "github.com/andrewarchi/nebula/ir"

import (
	"fmt"
	"strings"

	"github.com/andrewarchi/nebula/bigint"
)

type formatter struct {
	IDs    map[Val]int
	NextID int
}

func newFormatter() *formatter {
	return &formatter{
		IDs:    make(map[Val]int),
		NextID: 0,
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
				b.WriteString(f.FormatVal(*val))
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
			b.WriteString(f.FormatVal(*val))
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
		return fmt.Sprintf("%s = %v %s %s", f.FormatVal(*n.Assign), n.Op, f.FormatVal(*n.LHS), f.FormatVal(*n.RHS))
	case *UnaryExpr:
		return fmt.Sprintf("%s = %v %s", f.FormatVal(*n.Assign), n.Op, f.FormatVal(*n.Val))
	case *LoadExpr:
		return fmt.Sprintf("%s = load *%s", f.FormatVal(*n.Assign), f.FormatVal(*n.Addr))
	case *StoreStmt:
		return fmt.Sprintf("store *%s %s", f.FormatVal(*n.Addr), f.FormatVal(*n.Val))
	case *PrintStmt:
		return fmt.Sprintf("%v %s", n.Op, f.FormatVal(*n.Val))
	case *ReadExpr:
		return fmt.Sprintf("%s = %v", f.FormatVal(*n.Assign), n.Op)
	case *FlushStmt:
		return "flush"
	case *CallStmt:
		return fmt.Sprintf("call %s", n.Dest.Name())
	case *JmpStmt:
		return fmt.Sprintf("%v %s", n.Op, n.Dest.Name())
	case *JmpCondStmt:
		return fmt.Sprintf("%v %s %s %s", n.Op, f.FormatVal(*n.Cond), n.Then.Name(), n.Else.Name())
	case *RetStmt:
		return "ret"
	case *ExitStmt:
		return "exit"
	}
	panic("ir: unrecognized node type")
}

func (f *formatter) FormatVal(val Val) string {
	switch v := val.(type) {
	case *StackVal:
		id := v.ID
		if id >= 0 {
			if vid, ok := f.IDs[val]; ok {
				id = vid
			} else {
				id = f.NextID
				f.IDs[val] = f.NextID
				f.NextID++
			}
		}
		return fmt.Sprintf("%%%d", id)
	case *ConstVal:
		return v.Int.String()
	case *StringVal:
		return fmt.Sprintf("%q", v.Str)
	case *ArrayVal:
		return bigint.FormatSlice(v.Array)
	case *PhiVal:
		return fmt.Sprintf("phi(%s)", f.FormatValSlice(v.Vals))
	}
	panic("ir: unrecognized val type")
}

func (f *formatter) FormatValSlice(vals []*Val) string {
	var b strings.Builder
	for i, val := range vals {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(f.FormatVal(*val))
	}
	return b.String()
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
