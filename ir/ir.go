package ir // import "github.com/andrewarchi/nebula/ir"

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/digraph"
)

// Program is a set of interconnected basic blocks.
type Program struct {
	Name        string
	Blocks      []*BasicBlock
	Entry       *BasicBlock
	ConstVals   bigint.Map // map[*big.Int]*Val
	NextBlockID int
}

// BasicBlock is a list of consecutive non-branching instructions in a
// program followed by a branch.
type BasicBlock struct {
	ID         int           // Unique block ID for printing
	Labels     []Label       // Labels for this block in source
	Stack      Stack         // Stack frame of this block
	Nodes      []Node        // Non-branching non-stack instructions
	Terminator Terminator    // Terminator control flow instruction
	Entries    []*BasicBlock // Entry blocks; blocks immediately preceding this block in flow
	Callers    []*BasicBlock // Calling blocks; blocks calling this block or its parents
	Returns    []*BasicBlock // Returning blocks; blocks returning to this block
	Prev       *BasicBlock   // Predecessor block in source
	Next       *BasicBlock   // Successor block in source
}

// Val can be StackVal, HeapVal, ConstVal, or AddrVal. Two vals can be
// compared by address for equality.
type Val interface {
	val()
}

// SSAVal is an SSA register.
type SSAVal struct{ int } // TODO apparently not uniquely addressable when empty

// StackVal is a position under the stack.
type StackVal struct{ Pos int }

// ConstVal is a constant value such as from push or an expression with
// constant operands.
type ConstVal struct{ Int *big.Int }

// StringVal is a sequence of characters.
type StringVal struct{ Str string }

// ArrayVal is a sequence of integers.
type ArrayVal struct{ Array []*big.Int }

// PhiVal represents an SSA Φ function and stores the set of values it
// could be.
type PhiVal struct{ Vals []*Val }

// Node can be any expr or stmt type.
type Node interface {
	node()
}

// Expr represents an SSA expression that produces a value.
type Expr interface {
	Node
	Assign() *Val
	exprNode()
}

// Stmt represents an SSA statement that does not produce a value.
type Stmt interface {
	Node
	stmtNode()
}

// Terminator is any control flow statement. Valid types are CallStmt,
// JmpStmt, JmpCondStmt, RetStmt, and ExitStmt.
type Terminator interface {
	Node
	termNode()
}

// BinaryExpr evaluates a binary operation. Valid operator are add, sub,
// mul, div, and mod.
type BinaryExpr struct {
	Op     OpType
	Assign *Val
	LHS    *Val
	RHS    *Val
}

// UnaryExpr evaluates a unary operation. Valid operator is neg.
type UnaryExpr struct {
	Op     OpType
	Assign *Val
	Val    *Val
}

// LoadExpr evaluates a retrieve operation.
type LoadExpr struct {
	Assign *Val
	Addr   *Val
}

// StoreStmt evaluates a store operation.
type StoreStmt struct {
	Addr *Val
	Val  *Val
}

// PrintStmt prints a value. Valid operations are printc and printi.
type PrintStmt struct {
	Op  OpType
	Val *Val
}

// ReadExpr reads a value from stdin. Valid operations are readc and
// readi.
type ReadExpr struct {
	Op     OpType
	Assign *Val
}

// FlushStmt flushes stdout.
type FlushStmt struct{}

// CallStmt represents a call.
type CallStmt struct {
	Dest *BasicBlock
}

// JmpStmt unconditionally jumps to a block. Valid instructions are jmp
// and fallthrough.
type JmpStmt struct {
	Op   OpType
	Dest *BasicBlock
}

// JmpCondStmt conditionally jumps to a block based on a value. Valid
// instructions are jz and jn.
type JmpCondStmt struct {
	Op   OpType
	Cond *Val
	Then *BasicBlock
	Else *BasicBlock
}

// RetStmt represents a ret.
type RetStmt struct{}

// ExitStmt represents an end.
type ExitStmt struct{}

// OpType is the kind of operation of a node.
type OpType uint8

// Operators for nodes with multiple types.
const (
	Illegal OpType = iota

	Add
	Sub
	Mul
	Div
	Mod
	Neg

	Shl
	LShr
	AShr
	And
	Or
	Xor

	Printc
	Printi
	Prints

	Readc
	Readi

	Jmp
	Fallthrough

	Jz
	Jn
)

// Label is a label with an optional name.
type Label struct {
	ID   *big.Int
	Name string
}

// ErrorRetUnderflow is an error given when ret is executed without a
// caller.
type ErrorRetUnderflow struct {
	Traces [][]*BasicBlock
}

// ConnectEdges connects the CFG edges of a program based on the labels in the source.
func (p *Program) ConnectEdges(branches []*big.Int, labels *bigint.Map /* map[*big.Int]int */) error {
	p.Entry.Entries = append(p.Entry.Entries, nil)
	for i, block := range p.Blocks {
		branch := branches[i]
		if branch != nil {
			label, ok := labels.Get(branch)
			if !ok {
				return fmt.Errorf("ir: block %s jumps to non-existant label: %v", block.Name(), branch)
			}
			callee := p.Blocks[label.(int)]
			callee.Entries = append(callee.Entries, block)

			switch term := block.Terminator.(type) {
			case *CallStmt:
				term.Dest = callee
			case *JmpStmt:
				term.Dest = callee
			case *JmpCondStmt:
				term.Then = callee
				term.Else = block.Next
				block.Next.Entries = append(block.Next.Entries, block)
			case *RetStmt, ExitStmt:
			default:
				panic("ir: unrecognized terminator type")
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
	case *CallStmt:
		errs = errs.addTrace(term.Dest.connectCaller(block), block)
		errs = errs.addTrace(block.Next.connectCaller(caller), block)
		block.Next.Entries = appendUnique(block.Next.Entries, block.Returns...)
	case *JmpStmt:
		errs = errs.addTrace(term.Dest.connectCaller(caller), block)
	case *JmpCondStmt:
		errs = errs.addTrace(term.Then.connectCaller(caller), block)
		errs = errs.addTrace(term.Else.connectCaller(caller), block)
	case *RetStmt:
		if caller == nil {
			errs = errs.addTrace(&ErrorRetUnderflow{[][]*BasicBlock{{}}}, block)
		} else {
			caller.Returns = append(caller.Returns, block)
		}
	case *ExitStmt:
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

// Disconnect removes incoming edges to a basic block. The block is not
// removed from the program block slice and callers are not updated.
func (block *BasicBlock) Disconnect() {
	if block.Prev != nil {
		block.Prev.Next = block.Next
	}
	if block.Next != nil {
		block.Next.Prev = block.Prev
	}
	for _, exit := range block.Exits() {
		i := 0
		for _, entry := range exit.Entries {
			if entry != block {
				exit.Entries[i] = entry
				i++
			}
		}
		exit.Entries = exit.Entries[:i]
	}
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
		for _, edge := range block.Exits() {
			g.AddEdge(block.ID, edge.ID)
		}
	}
	return g
}

// LookupConst creates a val for a constant with matching constants
// having the same val.
func (p *Program) LookupConst(c *big.Int) *Val {
	if val, ok := p.ConstVals.Get(c); ok {
		return val.(*Val)
	}
	val := Val(&ConstVal{c})
	p.ConstVals.Put(c, &val)
	return &val
}

// NextVal creates a unique stack val.
func (p *Program) NextVal() *Val {
	val := Val(&SSAVal{})
	return &val
}

// ValEq returns whether the two vals reference the same definition.
func ValEq(a, b *Val) bool {
	return a != nil && b != nil && *a == *b
}

// AppendNode appends a node to the block.
func (block *BasicBlock) AppendNode(node Node) {
	block.Nodes = append(block.Nodes, node)
}

// Exits returns all outgoing edges of the block.
func (block *BasicBlock) Exits() []*BasicBlock {
	switch term := block.Terminator.(type) {
	case *CallStmt:
		return []*BasicBlock{term.Dest}
	case *JmpStmt:
		return []*BasicBlock{term.Dest}
	case *JmpCondStmt:
		return []*BasicBlock{term.Then, term.Else}
	case *RetStmt:
		exits := make([]*BasicBlock, len(block.Callers))
		for i, caller := range block.Callers {
			exits[i] = caller.Next
		}
		return exits
	case *ExitStmt:
		return nil
	default:
		panic("ir: unrecognized terminator type")
	}
}

// Name returns the name of the basic block from either the first label
// or the block address.
func (block *BasicBlock) Name() string {
	if block == nil {
		return "<nil>"
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
			fmt.Fprintf(&b, "    block_%d[label=\"%s\\n", block.ID, block.Name())
			if block.Stack.Len() != 0 {
				fmt.Fprintf(&b, " +%d", block.Stack.Len())
			}
			if block.Stack.Pops != 0 {
				fmt.Fprintf(&b, " -%d", block.Stack.Pops)
			}
			if block.Stack.Access != 0 {
				fmt.Fprintf(&b, " a%d", block.Stack.Access)
			}
			if len(block.Stack.Under) != 0 {
				fmt.Fprintf(&b, " r%d", len(block.Stack.Under))
			}
			b.WriteByte('"')
			if _, ok := block.Terminator.(*ExitStmt); ok {
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
		case *CallStmt:
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"call\"];\n", block.ID, term.Dest.ID)
		case *JmpStmt:
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"jmp\"];\n", block.ID, term.Dest.ID)
		case *JmpCondStmt:
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"true\"];\n", block.ID, term.Then.ID)
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"false\"];\n", block.ID, term.Else.ID)
		case *RetStmt:
			for _, caller := range block.Callers {
				fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"ret\\n%s\"];\n", block.ID, caller.Next.ID, caller.Name())
			}
		case *ExitStmt:
		default:
			panic("ir: unrecognized terminator type")
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func (p *Program) String() string {
	return newFormatter().FormatProgram(p)
}

func (block *BasicBlock) String() string {
	return newFormatter().FormatBlock(block)
}

func (SSAVal) val()    {}
func (StackVal) val()  {}
func (ConstVal) val()  {}
func (StringVal) val() {}
func (ArrayVal) val()  {}
func (PhiVal) val()    {}

func (BinaryExpr) node()  {}
func (UnaryExpr) node()   {}
func (LoadExpr) node()    {}
func (StoreStmt) node()   {}
func (PrintStmt) node()   {}
func (ReadExpr) node()    {}
func (FlushStmt) node()   {}
func (CallStmt) node()    {}
func (JmpStmt) node()     {}
func (JmpCondStmt) node() {}
func (RetStmt) node()     {}
func (ExitStmt) node()    {}

func (BinaryExpr) exprNode()  {}
func (UnaryExpr) exprNode()   {}
func (LoadExpr) exprNode()    {}
func (StoreStmt) stmtNode()   {}
func (PrintStmt) stmtNode()   {}
func (ReadExpr) exprNode()    {}
func (FlushStmt) stmtNode()   {}
func (CallStmt) termNode()    {}
func (JmpStmt) termNode()     {}
func (JmpCondStmt) termNode() {}
func (RetStmt) termNode()     {}
func (ExitStmt) termNode()    {}

func (op OpType) String() string {
	switch op {
	case Add:
		return "add"
	case Sub:
		return "sub"
	case Mul:
		return "mul"
	case Div:
		return "div"
	case Mod:
		return "mod"
	case Neg:
		return "neg"
	case Shl:
		return "shl"
	case LShr:
		return "lshr"
	case AShr:
		return "ashr"
	case And:
		return "and"
	case Or:
		return "or"
	case Xor:
		return "xor"
	case Printc:
		return "printc"
	case Printi:
		return "printi"
	case Prints:
		return "prints"
	case Readc:
		return "readc"
	case Readi:
		return "readi"
	case Jmp:
		return "jmp"
	case Fallthrough:
		return "fallthrough"
	case Jz:
		return "jz"
	case Jn:
		return "jn"
	}
	return "illegal"
}

func (l *Label) String() string {
	if l.Name != "" {
		return l.Name
	}
	return fmt.Sprintf("label_%v", l.ID)
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
