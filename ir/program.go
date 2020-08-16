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

// ErrorRetUnderflow is an error given when ret is executed without a
// caller.
type ErrorRetUnderflow struct {
	Traces [][]*BasicBlock
}

// ConnectEntries connects the block entries.
func (p *Program) ConnectEntries() error {
	p.Entry.Entries = append(p.Entry.Entries, nil)
	for _, block := range p.Blocks {
		if call, ok := block.Terminator.(*CallTerm); ok {
			// Only connect the callee. Next is connected to ret below.
			call.succs[0].Entries = append(call.succs[0].Entries, block)
		} else {
			for _, succ := range block.Terminator.Succs() {
				succ.Entries = append(succ.Entries, block)
			}
		}
	}
	if err := p.Entry.connectCaller(nil); err != nil {
		return err
	}
	p.trimUnreachable()
	return nil
}

func (block *BasicBlock) connectCaller(caller *BasicBlock) *ErrorRetUnderflow {
	for _, c := range block.Callers {
		if c == caller {
			return nil
		}
	}
	block.Callers = append(block.Callers, caller)
	var errs *ErrorRetUnderflow
	switch term := block.Terminator.(type) {
	case *CallTerm:
		errs = errs.addTrace(term.succs[0].connectCaller(block), block)
		errs = errs.addTrace(term.succs[1].connectCaller(caller), block)
		term.succs[1].Entries = appendUnique(term.succs[1].Entries, block.Returns...)
	case *JmpTerm:
		errs = errs.addTrace(term.succs[0].connectCaller(caller), block)
	case *JmpCondTerm:
		errs = errs.addTrace(term.succs[0].connectCaller(caller), block)
		errs = errs.addTrace(term.succs[1].connectCaller(caller), block)
	case *RetTerm:
		if caller == nil {
			errs = errs.addTrace(&ErrorRetUnderflow{[][]*BasicBlock{{}}}, block)
		} else {
			caller.Returns = append(caller.Returns, block)
		}
	case *ExitTerm:
	default:
		panic("ir: unrecognized terminator type")
	}
	return errs
}

func appendUnique(slice []*BasicBlock, blocks ...*BasicBlock) []*BasicBlock {
	l := len(slice)
outer:
	for _, block := range blocks {
		for i := 0; i < l; i++ {
			if slice[i] == block {
				continue outer
			}
		}
		slice = append(slice, block)
	}
	return slice
}

func (p *Program) trimUnreachable() {
	i := 0
	for _, block := range p.Blocks {
		if len(block.Callers) == 0 {
			block.Disconnect()
		} else {
			p.Blocks[i] = block
			i++
		}
	}
	p.Blocks = p.Blocks[:i]
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

// AppendInst appends an instruction to the block.
func (block *BasicBlock) AppendInst(inst Inst) {
	if _, ok := inst.(TermInst); ok {
		panic("AppendInst: terminator not allowed")
	}
	block.Nodes = append(block.Nodes, inst)
}

// SetTerminator sets the block's terminator instruction.
func (block *BasicBlock) SetTerminator(term TermInst) {
	if block.Terminator != nil {
		panic("SetTerminator: terminator already set")
	}
	block.Terminator = term
}

// Succs returns all outgoing edges of the block.
func (block *BasicBlock) Succs() []*BasicBlock {
	switch term := block.Terminator.(type) {
	case *RetTerm:
		exits := make([]*BasicBlock, len(block.Callers))
		for i, caller := range block.Callers {
			if caller != nil {
				exits[i] = caller.Next
			}
		}
		return exits
	default:
		return term.Succs()
	}
}

// Name returns the name of the basic block from either the first label
// or the block address.
func (block *BasicBlock) Name() string {
	if block == nil {
		return "<nil>"
	}
	if block.LabelName != "" {
		return block.LabelName
	}
	if len(block.Labels) != 0 {
		return block.Labels[0].String()
	}
	return fmt.Sprintf("block_%d", block.ID)
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

func (block *BasicBlock) String() string {
	return NewFormatter().FormatBlock(block)
}

func (l *Label) String() string {
	if l.Name != "" {
		return l.Name
	}
	return fmt.Sprintf("label_%v", l.ID)
}

// Position returns the full position information for a given pos.
func (p *Program) Position(pos token.Pos) token.Position {
	return p.File.PositionFor(pos, false)
}

func (err *ErrorRetUnderflow) addTrace(err2 *ErrorRetUnderflow, trace *BasicBlock) *ErrorRetUnderflow {
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

func (err *ErrorRetUnderflow) Error() string {
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
