package ast // import "github.com/andrewarchi/nebula/ast"

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/andrewarchi/nebula/bigint"
	"github.com/andrewarchi/nebula/token"
)

// AST is a set of interconnected basic blocks.
type AST struct {
	Blocks      []*BasicBlock
	Entry       *BasicBlock
	ConstVals   bigint.Map // map[*big.Int]*Val
	NextBlockID int
	NextStackID int
}

// BasicBlock is a list of consecutive non-branching instructions in a
// program followed by a branch.
type BasicBlock struct {
	ID         int           // Unique block ID for printing
	Labels     []Label       // Labels for this block in source
	Stack      Stack         // Stack frame of this block
	Nodes      []Node        // Non-branching non-stack instructions
	Terminator FlowStmt      // Terminator control flow instruction
	Entries    []*BasicBlock // Entry blocks; blocks immediately preceding this block in flow
	Callers    []*BasicBlock // Calling blocks; blocks calling this block or its parents
	Returns    []*BasicBlock // Returning blocks; blocks returning to this block
	Prev       *BasicBlock   // Predecessor block in source
	Next       *BasicBlock   // Successor block in source
}

var entryBlock = &BasicBlock{ID: -1, Labels: []Label{{nil, "@entry"}}}

// Node can be any expr or stmt type.
type Node interface {
	String() string
}

// Val can be StackVal, HeapVal, ConstVal, or AddrVal. Two vals can be
// compared by address for equality.
type Val = Node

// StackVal is a position on the stack.
type StackVal struct{ Val int }

// ConstVal is a constant value such as from push or an expression with
// constant operands.
type ConstVal struct{ Val *big.Int }

// StringVal is a sequence of characters.
type StringVal struct{ Val string }

// ArrayVal is a sequence of integers.
type ArrayVal struct{ Val []*big.Int }

// AddrVal marks a value as being a pointer to a value.
type AddrVal struct{ Val *Val }

// PhiVal represents an SSA Φ function and stores the set of values it
// could be.
type PhiVal struct {
	Vals []*Val
}

// AssignStmt assigns the value of an expression to the stack or heap.
type AssignStmt struct {
	Assign *Val
	Expr   Node
}

// ArithExpr evalutates a binary arithmetic operation. Valid operations
// are add, sub, mul, div, and mod.
type ArithExpr struct {
	Op  token.Type
	LHS *Val
	RHS *Val
}

// HeapExpr evaluates a heap access operation. Valid operations are
// store and retrieve.
type HeapExpr struct {
	Op  token.Type
	Val *Val
}

// PrintStmt prints a value. Valid operations are printc and printi.
type PrintStmt struct {
	Op  token.Type
	Val *Val
}

// ReadExpr reads a value from stdin. Valid operations are readc and
// readi.
type ReadExpr struct {
	Op token.Type
}

// FlowStmt is any flow control statement. Valid types are CallStmt,
// JmpStmt, JmpCondStmt, RetStmt, and EndStmt.
type FlowStmt = Node

// CallStmt represents a call.
type CallStmt struct {
	Callee *BasicBlock
}

// JmpStmt unconditionally jumps to a block. Valid instructions are jmp
// and fallthrough.
type JmpStmt struct {
	Op    token.Type
	Block *BasicBlock
}

// JmpCondStmt conditionally jumps to a block based on a value. Valid
// instructions are jz and jn.
type JmpCondStmt struct {
	Op         token.Type
	Val        *Val
	TrueBlock  *BasicBlock
	FalseBlock *BasicBlock
}

// RetStmt represents a ret.
type RetStmt struct{}

// EndStmt represents an end.
type EndStmt struct{}

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

// Parse parses tokens into an AST of basic blocks.
func Parse(tokens []token.Token, labelNames *bigint.Map) (*AST, error) {
	if needsImplicitEnd(tokens) {
		tokens = append(tokens, token.Token{Type: token.End})
	}
	ast, branches, labels, err := parseBlocks(tokens, labelNames)
	if err != nil {
		return nil, err
	}
	if err := ast.connectEdges(branches, labels); err != nil {
		return ast, err
	}
	return ast, nil
}

func needsImplicitEnd(tokens []token.Token) bool {
	if len(tokens) == 0 {
		return true
	}
	switch tokens[len(tokens)-1].Type {
	case token.Call, token.Jmp, token.Ret, token.End:
		return false
	}
	return true
}

func parseBlocks(tokens []token.Token, labelNames *bigint.Map) (*AST, []*big.Int, *bigint.Map, error) {
	ast := &AST{ConstVals: *bigint.NewMap(nil)}
	var branches []*big.Int
	labels := bigint.NewMap(nil) // map[*big.Int]int
	prevLabel := "entry"
	labelIndex := 0

	for i := 0; i < len(tokens); i++ {
		var block BasicBlock
		block.ID = len(ast.Blocks)
		if len(ast.Blocks) > 0 {
			prev := ast.Blocks[len(ast.Blocks)-1]
			prev.Next = &block
			block.Prev = prev
		}

		if tokens[i].Type != token.Label && i != 0 && prevLabel != "" {
			labelIndex++
			block.Labels = append(block.Labels, Label{nil, fmt.Sprintf("%s@%d", prevLabel, labelIndex)})
		}
		for tokens[i].Type == token.Label {
			label := tokens[i].Arg
			if labels.Put(label, len(ast.Blocks)) {
				return nil, nil, nil, fmt.Errorf("ast: label is not unique: %s", label)
			}
			var name string
			if labelNames != nil {
				if n, ok := labelNames.Get(label); ok {
					name = n.(string)
				}
			}
			prevLabel = name
			labelIndex = 0
			block.Labels = append(block.Labels, Label{label, name})
			i++
		}

		var branch *big.Int
		for ; i < len(tokens); i++ {
			branch = ast.appendInstruction(&block, tokens[i])
			if block.Terminator != nil {
				if tokens[i].Type == token.Label {
					i--
				}
				break
			}
		}

		ast.Blocks = append(ast.Blocks, &block)
		branches = append(branches, branch)
	}
	ast.Entry = ast.Blocks[0]
	ast.NextBlockID = len(ast.Blocks)
	return ast, branches, labels, nil
}

func (ast *AST) appendInstruction(block *BasicBlock, tok token.Token) *big.Int {
	switch tok.Type {
	case token.Push:
		block.Stack.Push(ast.lookupConst(tok.Arg))
	case token.Dup:
		block.Stack.Dup()
	case token.Copy:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ast: copy argument overflow: %v", tok.Arg))
		} else if n < 0 {
			panic(fmt.Sprintf("ast: copy argument negative: %v", tok.Arg))
		}
		block.Stack.Copy(n)
	case token.Swap:
		block.Stack.Swap()
	case token.Drop:
		block.Stack.Drop()
	case token.Slide:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ast: slide argument overflow: %v", tok.Arg))
		} else if n < 0 {
			panic(fmt.Sprintf("ast: slide argument negative: %v", tok.Arg))
		}
		block.Stack.Slide(n)

	case token.Add, token.Sub, token.Mul, token.Div, token.Mod:
		rhs, lhs, assign := block.Stack.Pop(), block.Stack.Pop(), ast.nextVal()
		block.Stack.Push(assign)
		block.assign(assign, &ArithExpr{
			Op:  tok.Type,
			LHS: lhs,
			RHS: rhs,
		})

	case token.Store:
		val, addr := block.Stack.Pop(), block.Stack.Pop()
		assign := Val(&AddrVal{addr})
		block.assign(&assign, &HeapExpr{
			Op:  token.Store,
			Val: val,
		})
	case token.Retrieve:
		addr, assign := block.Stack.Pop(), ast.nextVal()
		block.Stack.Push(assign)
		val := Val(&AddrVal{addr})
		block.assign(assign, &HeapExpr{
			Op:  token.Retrieve,
			Val: &val,
		})

	case token.Label:
		block.Terminator = &JmpStmt{Op: token.Fallthrough}
		return tok.Arg
	case token.Call:
		block.Terminator = &CallStmt{}
		return tok.Arg
	case token.Jmp:
		block.Terminator = &JmpStmt{Op: tok.Type}
		return tok.Arg
	case token.Jz, token.Jn:
		block.Terminator = &JmpCondStmt{
			Op:  tok.Type,
			Val: block.Stack.Pop(),
		}
		return tok.Arg
	case token.Ret:
		block.Terminator = &RetStmt{}
	case token.End:
		block.Terminator = &EndStmt{}

	case token.Printc, token.Printi:
		block.Nodes = append(block.Nodes, &PrintStmt{
			Op:  tok.Type,
			Val: block.Stack.Pop(),
		})
	case token.Readc, token.Readi:
		assign := Val(&AddrVal{block.Stack.Pop()})
		block.assign(&assign, &ReadExpr{
			Op: tok.Type,
		})

	default:
		panic(fmt.Sprintf("ast: illegal token: %v", tok.Type))
	}
	return nil
}

func (block *BasicBlock) assign(assign *Val, expr Val) {
	block.Nodes = append(block.Nodes, &AssignStmt{
		Assign: assign,
		Expr:   expr,
	})
}

func (ast *AST) connectEdges(branches []*big.Int, labels *bigint.Map) error {
	ast.Entry.Entries = append(ast.Entry.Entries, entryBlock)
	for i, block := range ast.Blocks {
		branch := branches[i]
		if branch != nil {
			label, ok := labels.Get(branch)
			if !ok {
				return fmt.Errorf("ast: block %s jumps to non-existant label: %v", block.Name(), branch)
			}
			callee := ast.Blocks[label.(int)]
			callee.Entries = append(callee.Entries, block)

			switch term := block.Terminator.(type) {
			case *CallStmt:
				term.Callee = callee
			case *JmpStmt:
				term.Block = callee
			case *JmpCondStmt:
				term.TrueBlock = callee
				term.FalseBlock = block.Next
				block.Next.Entries = append(block.Next.Entries, block)
			}
		}
	}
	if err := ast.Entry.connectCaller(entryBlock); err != nil {
		return err
	}
	ast.trimUnreachable()
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
		errs = errs.addTrace(term.Callee.connectCaller(block), block)
		errs = errs.addTrace(block.Next.connectCaller(caller), block)
		block.Next.Entries = appendUnique(block.Next.Entries, block.Returns...)
	case *JmpStmt:
		errs = errs.addTrace(term.Block.connectCaller(caller), block)
	case *JmpCondStmt:
		errs = errs.addTrace(term.TrueBlock.connectCaller(caller), block)
		errs = errs.addTrace(term.FalseBlock.connectCaller(caller), caller)
	case *RetStmt:
		if caller == entryBlock {
			errs = errs.addTrace(&ErrorRetUnderflow{[][]*BasicBlock{{}}}, block)
		}
		caller.Returns = append(caller.Returns, block)
	case *EndStmt:
	}
	return errs
}

// Disconnect removes all incoming edges to a basic block. The block is
// not removed from the AST block slice.
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

func (ast *AST) trimUnreachable() {
	i := 0
	for _, block := range ast.Blocks {
		if len(block.Callers) == 0 {
			block.Disconnect()
		} else {
			ast.Blocks[i] = block
			i++
		}
	}
	ast.Blocks = ast.Blocks[:i]
}

// RenumberIDs cleans up block IDs to match the block index.
func (ast *AST) RenumberIDs() {
	for i, block := range ast.Blocks {
		block.ID = i
	}
	ast.NextBlockID = len(ast.Blocks)
}

// Digraph constructs a digraph representing control flow.
func (ast *AST) Digraph() Digraph {
	g := make(Digraph, ast.NextBlockID)
	for _, block := range ast.Blocks {
		for _, edge := range block.Exits() {
			g.AddEdge(block.ID, edge.ID)
		}
	}
	return g
}

func (ast *AST) lookupConst(c *big.Int) *Val {
	if val, ok := ast.ConstVals.Get(c); ok {
		return val.(*Val)
	}
	val := Val(&ConstVal{c})
	ast.ConstVals.Put(c, &val)
	return &val
}

func (ast *AST) nextVal() *Val {
	val := Val(&StackVal{ast.NextStackID})
	ast.NextStackID++
	return &val
}

// Exits returns all outgoing edges of the block.
func (block *BasicBlock) Exits() []*BasicBlock {
	switch term := block.Terminator.(type) {
	case *CallStmt:
		return []*BasicBlock{term.Callee}
	case *JmpStmt:
		return []*BasicBlock{term.Block}
	case *JmpCondStmt:
		return []*BasicBlock{term.TrueBlock, term.FalseBlock}
	case *RetStmt:
		exits := make([]*BasicBlock, len(block.Callers))
		for i, caller := range block.Callers {
			exits[i] = caller.Next
		}
		return exits
	case *EndStmt:
		return nil
	}
	panic(fmt.Errorf("ast: invalid terminator type: %T", block.Terminator))
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
	if block.ID == 0 {
		return "entry"
	}
	return fmt.Sprintf("block_%d", block.ID)
}

// DotDigraph creates a control flow graph in the Graphviz DOT format.
func (ast *AST) DotDigraph() string {
	var b strings.Builder
	b.WriteString("digraph {\n")
	b.WriteString("  entry[shape=point];\n")
	ast.RenumberIDs()
	for i, scc := range ast.Digraph().SCCs() {
		fmt.Fprintf(&b, "  subgraph cluster_%d {\n", i)
		for _, node := range scc {
			block := ast.Blocks[node]
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
			if _, ok := block.Terminator.(*EndStmt); ok {
				b.WriteString(" peripheries=2")
			}
			b.WriteString("];\n")
		}
		b.WriteString("  }\n")
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "  entry -> block_%d;\n", ast.Entry.ID)
	for _, block := range ast.Blocks {
		switch stmt := block.Terminator.(type) {
		case *CallStmt:
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"call\"];\n", block.ID, stmt.Callee.ID)
		case *JmpStmt:
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"jmp\"];\n", block.ID, stmt.Block.ID)
		case *JmpCondStmt:
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"true\"];\n", block.ID, stmt.TrueBlock.ID)
			fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"false\"];\n", block.ID, stmt.FalseBlock.ID)
		case *RetStmt:
			for _, caller := range block.Callers {
				fmt.Fprintf(&b, "  block_%d -> block_%d[label=\"ret\\n%s\"];\n", block.ID, caller.Next.ID, caller.Name())
			}
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func (ast *AST) String() string {
	var b strings.Builder
	for i, block := range ast.Blocks {
		if i != 0 {
			b.WriteByte('\n')
		}
		b.WriteString(block.String())
	}
	return b.String()
}

func (block *BasicBlock) String() string {
	var b strings.Builder
	if len(block.Labels) == 0 {
		if block.ID == 0 {
			b.WriteString("entry:\n")
		} else {
			fmt.Fprintf(&b, "block_%d:\n", block.ID)
		}
	}
	for _, label := range block.Labels {
		b.WriteString(label.String())
		b.WriteString(":\n")
	}
	fmt.Fprintf(&b, "    ; entries: %s\n", formatBlockList(block.Entries))
	fmt.Fprintf(&b, "    ; callers: %s\n", formatBlockList(block.Callers))
	if len(block.Returns) != 0 {
		fmt.Fprintf(&b, "    ; returns: %s\n", formatBlockList(block.Returns))
	}
	fmt.Fprintf(&b, "    ; stack: %s\n", &block.Stack)
	for _, node := range block.Nodes {
		b.WriteString("    ")
		b.WriteString(node.String())
		b.WriteByte('\n')
	}
	b.WriteString("    ")
	b.WriteString(block.Terminator.String())
	b.WriteByte('\n')
	return b.String()
}

func formatBlockList(blocks []*BasicBlock) string {
	if len(blocks) == 0 {
		return "-"
	}
	var b strings.Builder
	for i, block := range blocks {
		if i != 0 {
			b.WriteByte(' ')
		}
		b.WriteString(block.Name())
	}
	return b.String()
}

func (s *StackVal) String() string   { return fmt.Sprintf("%%%d", s.Val) }
func (c *ConstVal) String() string   { return c.Val.String() }
func (s *StringVal) String() string  { return fmt.Sprintf("%q", s.Val) }
func (a *ArrayVal) String() string   { return bigint.FormatSlice(a.Val) }
func (a *AddrVal) String() string    { return fmt.Sprintf("*%v", *a.Val) }
func (p *PhiVal) String() string     { return fmt.Sprintf("phi(%s)", formatValSlice(p.Vals)) }
func (a *AssignStmt) String() string { return fmt.Sprintf("%v = %v", *a.Assign, a.Expr) }
func (b *ArithExpr) String() string  { return fmt.Sprintf("%v %v %v", b.Op, *b.LHS, *b.RHS) }
func (u *HeapExpr) String() string   { return fmt.Sprintf("%v %v", u.Op, *u.Val) }
func (p *PrintStmt) String() string  { return fmt.Sprintf("%v %v", p.Op, *p.Val) }
func (r *ReadExpr) String() string   { return r.Op.String() }
func (c *CallStmt) String() string   { return fmt.Sprintf("call %s", c.Callee.Name()) }
func (j *JmpStmt) String() string    { return fmt.Sprintf("%v %s", j.Op, j.Block.Name()) }
func (j *JmpCondStmt) String() string {
	return fmt.Sprintf("%v %v %s %s", j.Op, *j.Val, j.TrueBlock.Name(), j.FalseBlock.Name())
}
func (r *RetStmt) String() string { return fmt.Sprintf("ret") }
func (*EndStmt) String() string   { return "end" }

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
		for i := len(trace); i > 0; {
			i--
			b.WriteString(trace[i].Name())
			if i != 0 {
				b.WriteString(" -> ")
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
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

func formatValSlice(vals []*Val) string {
	var b strings.Builder
	for i, val := range vals {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString((*val).String())
	}
	return b.String()
}
