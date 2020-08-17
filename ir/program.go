package ir

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/andrewarchi/nebula/internal/digraph"
)

// Program is a set of interconnected basic blocks.
type Program struct {
	Name        string
	Blocks      []*BasicBlock
	Entry       *BasicBlock
	NextBlockID int
	File        *token.File
}

// TrimUnreachable removes uncalled blocks.
func (p *Program) TrimUnreachable() {
	// TODO traverse in topological order
	i := 0
	for _, block := range p.Blocks {
		if len(block.Callers) == 0 {
			block.Disconnect()
		} else {
			p.Blocks[i] = block
			i++
		}
	}
	if i != len(p.Blocks)-1 {
		p.Blocks = p.Blocks[:i]
		p.RenumberBlockIDs()
	}
}

// RenumberBlockIDs cleans up block IDs to match the block index.
func (p *Program) RenumberBlockIDs() {
	for i, block := range p.Blocks {
		block.ID = i
	}
	p.NextBlockID = len(p.Blocks)
}

// Digraph constructs a digraph representing control flow.
func (p *Program) Digraph() digraph.Digraph {
	g := make(digraph.Digraph, p.NextBlockID)
	for _, block := range p.Blocks {
		for _, edge := range block.Succs() {
			g.AddEdge(block.ID, edge.ID)
		}
	}
	return g
}

// DotDigraph creates a control flow graph in the Graphviz DOT format.
func (p *Program) DotDigraph() string {
	var b strings.Builder
	b.WriteString("digraph {\n")
	b.WriteString("  entry[shape=point];\n")
	p.RenumberBlockIDs()
	for i, scc := range p.Digraph().SCCs() {
		fmt.Fprintf(&b, "  subgraph cluster_%d {\n", i)
		for _, node := range scc {
			block := p.Blocks[node]
			fmt.Fprintf(&b, "    block_%d[label=\"%s\"", block.ID, block.Name())
			if _, ok := block.Terminator.(*ExitTerm); ok {
				b.WriteString(" peripheries=2")
			}
			b.WriteString("];\n")
		}
		b.WriteString("  }\n")
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "  entry -> block_%d;\n", p.Entry.ID)
	for _, block := range p.Blocks {
		switch term := block.Terminator.(type) {
		case *CallTerm:
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"call\"];\n", block.ID, term.succs[0].ID)
		case *JmpTerm:
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"jmp\"];\n", block.ID, term.succs[0].ID)
		case *JmpCondTerm:
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"true\"];\n", block.ID, term.succs[0].ID)
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"false\"];\n", block.ID, term.succs[1].ID)
		case *RetTerm:
			for _, caller := range block.Callers {
				fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"ret\\n%s\"];\n", block.ID, caller.Next.ID, caller.Name())
			}
		case *ExitTerm:
		default:
			panic("ir: unrecognized terminator type")
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func (p *Program) String() string {
	return NewFormatter().FormatProgram(p)
}

// Position returns the full position information for a given pos.
func (p *Program) Position(pos token.Pos) token.Position {
	return p.File.PositionFor(pos, false)
}

func (err *RetUnderflowError) addTrace(err2 *RetUnderflowError, trace *BasicBlock) *RetUnderflowError {
	if err2 == nil {
		return err
	}
	for i := range err2.Traces {
		err2.Traces[i] = append(err2.Traces[i], trace)
	}
	if err == nil {
		return err2
	}
	err.Traces = append(err.Traces, err2.Traces...)
	return err
}

func (err *RetUnderflowError) Error() string {
	if err == nil {
		return "<nil>"
	}
	if len(err.Traces) == 0 {
		return "call stack underflow"
	}
	var b strings.Builder
	b.WriteString("call stack underflow\n")
	for _, trace := range err.Traces {
		fmt.Fprintf(&b, "  %s: ", trace[0].Name())
		for i := len(trace) - 1; i >= 0; i-- {
			b.WriteString(trace[i].Name())
			if i != 0 {
				b.WriteString(" -> ")
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
