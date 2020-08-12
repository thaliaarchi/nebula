package ir

import (
	"fmt"
	"strings"
)

// Formatter pretty prints Nebula IR.
type Formatter struct {
	ids    map[Value]int
	nextID int
}

// NewFormatter constructs a Formatter.
func NewFormatter() *Formatter {
	return &Formatter{
		ids:    make(map[Value]int),
		nextID: 0,
	}
}

// FormatProgram pretty prints a Program.
func (f *Formatter) FormatProgram(p *Program) string {
	var b strings.Builder
	for i, block := range p.Blocks {
		if i != 0 {
			b.WriteByte('\n')
		}
		b.WriteString(f.FormatBlock(block))
	}
	return b.String()
}

// FormatBlock pretty prints a BasicBlock.
func (f *Formatter) FormatBlock(block *BasicBlock) string {
	var b strings.Builder
	if len(block.Labels) == 0 {
		fmt.Fprintf(&b, "block_%d:\n", block.ID)
	}
	for _, label := range block.Labels {
		b.WriteString(label.String())
		b.WriteString(":\n")
	}

	b.WriteString("    ; entries: ")
	writeBlockSlice(&b, block.Entries)
	b.WriteString("\n    ; callers: ")
	writeBlockSlice(&b, block.Callers)
	if len(block.Returns) != 0 {
		b.WriteString("\n    ; returns: ")
		writeBlockSlice(&b, block.Returns)
	}
	b.WriteByte('\n')

	for _, inst := range block.Nodes {
		b.WriteString("    ")
		b.WriteString(f.FormatInst(inst))
		b.WriteByte('\n')
	}

	if block.Stack.Pops > 0 {
		fmt.Fprintf(&b, "    pop %d\n", block.Stack.Pops)
	}
	if len(block.Stack.Values) != 0 {
		b.WriteString("    push [")
		for i, val := range block.Stack.Values {
			if i != 0 {
				b.WriteByte(' ')
			}
			b.WriteString(f.FormatValue(val))
		}
		b.WriteString("]\n")
	}

	b.WriteString("    ")
	b.WriteString(f.FormatInst(block.Terminator))
	b.WriteByte('\n')
	return b.String()
}

// FormatInst pretty prints an Inst.
func (f *Formatter) FormatInst(inst Inst) string {
	var b strings.Builder
	if val, ok := inst.(Value); ok {
		b.WriteString(f.FormatValue(val))
		b.WriteString(" = ")
	}
	b.WriteString(inst.OpString())
	writeStackPos(&b, inst)
	if user, ok := inst.(User); ok {
		for _, op := range user.Operands() {
			b.WriteByte(' ')
			if op == nil {
				b.WriteString("<nil>")
			} else {
				b.WriteString(f.FormatValue(op.Def))
			}
		}
	}
	if term, ok := inst.(TermInst); ok {
		for _, succ := range term.Succs() {
			b.WriteByte(' ')
			b.WriteString(succ.Name())
		}
	}
	return b.String()
}

// FormatValue pretty prints a value.
func (f *Formatter) FormatValue(val Value) string {
	switch v := val.(type) {
	case *IntConst:
		return v.Int.String()
	}
	var id int
	if vid, ok := f.ids[val]; ok {
		id = vid
	} else {
		id = f.nextID
		f.ids[val] = f.nextID
		f.nextID++
	}
	return fmt.Sprintf("%%%d", id)
}

func writeBlockSlice(b *strings.Builder, blocks []*BasicBlock) {
	if len(blocks) == 0 {
		b.WriteString("-")
		return
	}
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
}

func writeStackPos(b *strings.Builder, inst Inst) {
	var pos int
	switch s := inst.(type) {
	case *LoadStackExpr:
		pos = s.StackPos
	case *StoreStackStmt:
		pos = s.StackPos
	case *CheckStackStmt:
		pos = s.StackSize
	default:
		return
	}
	b.WriteByte(' ')
	fmt.Fprintf(b, "%d", pos)
}
