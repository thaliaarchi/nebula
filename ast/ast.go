package ast

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/andrewarchi/wspace/bigint"
	"github.com/andrewarchi/wspace/token"
)

// AST is a set of interconnected basic blocks.
type AST struct {
	Blocks []*BasicBlock
	Entry  *BasicBlock
	NextID int
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
	Trace []*BasicBlock
}

// Parse parses tokens into an AST of basic blocks.
func Parse(tokens []token.Token, labelNames *bigint.Map, trim bool) (*AST, error) {
	if needsImplicitEnd(tokens) {
		tokens = append(tokens, token.Token{Type: token.End})
	}
	ast, branches, labels, err := parseBlocks(tokens, labelNames)
	if err != nil {
		return nil, err
	}
	if err := ast.connectEdges(branches, labels, trim); err != nil {
		return nil, err
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
	var ast AST
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
			branch = block.appendInstruction(tokens[i])
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
	ast.NextID = len(ast.Blocks)
	return &ast, branches, labels, nil
}

func (block *BasicBlock) appendInstruction(tok token.Token) *big.Int {
	switch tok.Type {
	case token.Push:
		block.Stack.PushConst(tok.Arg)
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
		rhs, lhs, assign := block.Stack.Pop(), block.Stack.Pop(), block.Stack.Push()
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
		addr, assign := block.Stack.Pop(), block.Stack.Push()
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

func (ast *AST) connectEdges(branches []*big.Int, labels *bigint.Map, trim bool) error {
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
	if trim {
		ast.trimUnreachable()
	} else {
		for _, block := range ast.Blocks {
			if call, ok := block.Terminator.(*CallStmt); ok {
				call.Callee.connectCaller(block)
				block.Next.Entries = appendUnique(block.Next.Entries, block.Returns...)
			}
		}
	}
	return nil
}

func (block *BasicBlock) connectCaller(caller *BasicBlock) *ErrorRetUnderflow {
	for _, c := range block.Callers {
		if c == caller {
			return nil
		}
	}
	block.Callers = append(block.Callers, caller)
	switch term := block.Terminator.(type) {
	case *CallStmt:
		if err := term.Callee.connectCaller(block); err != nil {
			return err.append(block)
		}
		if err := block.Next.connectCaller(caller); err != nil {
			return err.append(block)
		}
		block.Next.Entries = appendUnique(block.Next.Entries, block.Returns...)
	case *JmpStmt:
		if err := term.Block.connectCaller(caller); err != nil {
			return err.append(block)
		}
	case *JmpCondStmt:
		if err := term.TrueBlock.connectCaller(caller); err != nil {
			return err.append(block)
		}
		if err := term.FalseBlock.connectCaller(caller); err != nil {
			return err.append(caller)
		}
	case *RetStmt:
		if caller == entryBlock {
			return &ErrorRetUnderflow{[]*BasicBlock{block}}
		}
		caller.Returns = append(caller.Returns, block)
	case *EndStmt:
	}
	return nil
}

func (ast *AST) trimUnreachable() {
	visited := newBitset(ast.NextID)
	ast.Entry.dfs(visited)
	i := 0
	for _, block := range ast.Blocks {
		if !visited.Test(uint32(block.ID)) {
			if block.Prev != nil {
				block.Prev.Next = block.Next
			}
			if block.Next != nil {
				block.Next.Prev = block.Prev
			}
			for _, exit := range block.Exits() {
				j := 0
				for _, entry := range exit.Entries {
					if entry != block {
						exit.Entries[j] = entry
						j++
					}
				}
				exit.Entries = exit.Entries[:j]
			}
			continue
		}
		ast.Blocks[i] = block
		i++
	}
	ast.Blocks = ast.Blocks[:i]
}

func (block *BasicBlock) dfs(visited bitset) {
	if visited.Test(uint32(block.ID)) {
		return
	}
	visited.Set(uint32(block.ID))
	switch term := block.Terminator.(type) {
	case *CallStmt:
		term.Callee.dfs(visited)
		if len(block.Returns) != 0 {
			block.Next.dfs(visited)
		}
	case *JmpStmt:
		term.Block.dfs(visited)
	case *JmpCondStmt:
		term.TrueBlock.dfs(visited)
		term.FalseBlock.dfs(visited)
	case *RetStmt, *EndStmt:
	}
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
	for _, block := range ast.Blocks {
		if _, ok := block.Terminator.(*EndStmt); ok {
			fmt.Fprintf(&b, "  block_%d[label=\"%s\" peripheries=2];\n", block.ID, block.Name())
		} else {
			fmt.Fprintf(&b, "  block_%d[label=\"%s\"];\n", block.ID, block.Name())
		}
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
	fmt.Fprintf(&b, "    ; returns: %s\n", formatBlockList(block.Returns))
	fmt.Fprintf(&b, "    ; stack %v, pop %d, access %d\n", &block.Stack, block.Stack.Pops, block.Stack.Access)
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

func (err *ErrorRetUnderflow) append(block *BasicBlock) *ErrorRetUnderflow {
	err.Trace = append(err.Trace, block)
	return err
}

func (err *ErrorRetUnderflow) Error() string {
	if len(err.Trace) == 0 {
		return "call stack underflow"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "call stack underflow at %s: ", err.Trace[0].Name())
	for i := len(err.Trace); i > 0; {
		i--
		b.WriteString(err.Trace[i].Name())
		if i != 0 {
			b.WriteString(" -> ")
		}
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
