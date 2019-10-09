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
}

// BasicBlock is a list of consecutive non-branching instructions in a
// program followed by a branch.
type BasicBlock struct {
	ID      int           // Unique block ID for printing
	Labels  []*big.Int    // Labels for this block in source
	Stack   Stack         // Stack frame of this block
	Nodes   []Node        // Non-branching non-stack instructions
	Exit    FlowStmt      // Control flow instruction
	Callers []*BasicBlock // Calling blocks; blocks calling this block or its parents
	Entries []*BasicBlock // Entry blocks; blocks immediately preceding this block in flow
	Prev    *BasicBlock   // Predecessor block in source
	Next    *BasicBlock   // Successor block in source
}

// Node can be any expr or stmt type.
type Node interface {
	String() string
}

// Val can be StackVal, HeapVal, ConstVal, or AddrVal.
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

// Parse parses tokens into an AST of basic blocks.
func Parse(tokens []token.Token) (*AST, error) {
	if needsImplicitEnd(tokens) {
		tokens = append(tokens, token.Token{Type: token.End})
	}
	ast, branches, labels, err := parseBlocks(tokens)
	if err != nil {
		return nil, err
	}
	if err := ast.connectEdges(branches, labels); err != nil {
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

func parseBlocks(tokens []token.Token) (*AST, []*big.Int, *bigint.Map, error) {
	var ast AST
	var branches []*big.Int
	labels := bigint.NewMap(nil) // map[*big.Int]int
	for i := 0; i < len(tokens); i++ {
		var block BasicBlock
		block.ID = len(ast.Blocks)
		if len(ast.Blocks) > 0 {
			prev := ast.Blocks[len(ast.Blocks)-1]
			prev.Next = &block
			block.Prev = prev
		}

		for tokens[i].Type == token.Label {
			label := tokens[i].Arg
			if labels.Put(label, len(ast.Blocks)) {
				return nil, nil, nil, fmt.Errorf("ast: label is not unique: %s", label)
			}
			block.Labels = append(block.Labels, label)
			i++
		}

		var branch *big.Int
		for ; i < len(tokens); i++ {
			branch = block.appendToken(tokens[i])
			if block.Exit != nil {
				if tokens[i].Type == token.Label {
					i--
				}
				break
			}
		}

		ast.Blocks = append(ast.Blocks, &block)
		branches = append(branches, branch)
	}
	return &ast, branches, labels, nil
}

func (block *BasicBlock) appendToken(tok token.Token) *big.Int {
	switch tok.Type {
	case token.Push:
		block.Stack.PushConst(tok.Arg)
	case token.Dup:
		block.Stack.Dup()
	case token.Copy:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ast: copy argument overflows size: %v", tok.Arg))
		} else if n < 0 {
			panic(fmt.Sprintf("ast: copy argument negative: %v", tok.Arg))
		}
		block.Stack.Copy(n)
	case token.Swap:
		block.Stack.Swap()
	case token.Drop:
		block.Stack.Pop()
	case token.Slide:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ast: slide argument overflows size: %v", tok.Arg))
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
		val, assign := block.Stack.Pop(), block.Stack.Push()
		block.assign(assign, &HeapExpr{
			Op:  token.Retrieve,
			Val: val,
		})

	case token.Label:
		block.Exit = &JmpStmt{Op: token.Fallthrough}
		return tok.Arg
	case token.Call:
		block.Exit = &CallStmt{}
		return tok.Arg
	case token.Jmp:
		block.Exit = &JmpStmt{Op: tok.Type}
		return tok.Arg
	case token.Jz, token.Jn:
		block.Exit = &JmpCondStmt{
			Op:  tok.Type,
			Val: block.Stack.Pop(),
		}
		return tok.Arg
	case token.Ret:
		block.Exit = &RetStmt{}
	case token.End:
		block.Exit = &EndStmt{}

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
	for i, block := range ast.Blocks {
		branch := branches[i]
		if branch != nil {
			label, ok := labels.Get(branch)
			if !ok {
				return fmt.Errorf("ast: block %s jumps to non-existant label: %v", block.Name(), branch)
			}
			callee := ast.Blocks[label.(int)]
			callee.Entries = append(callee.Entries, block)

			switch exit := block.Exit.(type) {
			case *CallStmt:
				exit.Callee = callee
			case *JmpStmt:
				exit.Block = callee
			case *JmpCondStmt:
				exit.TrueBlock = callee
				exit.FalseBlock = block.Next
				block.Next.Entries = append(block.Next.Entries, block)
			}
		}
	}
	for _, block := range ast.Blocks {
		if call, ok := block.Exit.(*CallStmt); ok {
			call.Callee.annotateCaller(block, make(map[*BasicBlock]bool))
		}
	}
	return nil
}

func (block *BasicBlock) annotateCaller(caller *BasicBlock, visited map[*BasicBlock]bool) {
	if visited[block] {
		return
	}
	visited[block] = true
	block.Callers = append(block.Callers, caller)
	switch exit := block.Exit.(type) {
	case *CallStmt:
		exit.Callee.annotateCaller(caller, visited)
	case *JmpStmt:
		exit.Block.annotateCaller(caller, visited)
	case *JmpCondStmt:
		exit.TrueBlock.annotateCaller(caller, visited)
		exit.FalseBlock.annotateCaller(caller, visited)
	case *RetStmt, *EndStmt:
	}
}

// Exits returns all outgoing edges of the block.
func (block *BasicBlock) Exits() []*BasicBlock {
	switch exit := block.Exit.(type) {
	case *CallStmt:
		return []*BasicBlock{exit.Callee}
	case *JmpStmt:
		return []*BasicBlock{exit.Block}
	case *JmpCondStmt:
		return []*BasicBlock{exit.TrueBlock, exit.FalseBlock}
	case *RetStmt:
		exits := make([]*BasicBlock, len(block.Callers))
		for i, caller := range block.Callers {
			exits[i] = caller.Next
		}
		return exits
	case *EndStmt:
		return nil
	}
	panic(fmt.Errorf("ast: invalid exit type: %T", block.Exit))
}

// Name returns the name of the basic block from either the first label
// or the block address.
func (block *BasicBlock) Name() string {
	if block == nil {
		return "<nil>"
	}
	if block.ID == 0 {
		return "entry"
	}
	if len(block.Labels) != 0 {
		return fmt.Sprintf("label_%v", block.Labels[0])
	}
	return fmt.Sprintf("block_%d", block.ID)
}

func (ast *AST) DotDigraph() string {
	var b strings.Builder
	b.WriteString("digraph {\n")
	for _, block := range ast.Blocks {
		name := block.Name()
		switch stmt := block.Exit.(type) {
		case *CallStmt:
			fmt.Fprintf(&b, "  %s -> %s[label=\"call\"];\n", name, stmt.Callee.Name())
		case *JmpStmt:
			fmt.Fprintf(&b, "  %s -> %s[label=\"jmp\"];\n", name, stmt.Block.Name())
		case *JmpCondStmt:
			fmt.Fprintf(&b, "  %s -> %s[label=\"true\"];\n", name, stmt.TrueBlock.Name())
			fmt.Fprintf(&b, "  %s -> %s[label=\"false\"];\n", name, stmt.FalseBlock.Name())
		case *RetStmt:
			for _, exit := range block.Exits() {
				fmt.Fprintf(&b, "  %s -> %s[label=\"ret\"];\n", name, exit.Name())
			}
		case *EndStmt:
			fmt.Fprintf(&b, "  %s;\n", name)
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
	if block.ID == 0 {
		b.WriteString("entry:\n")
	} else if len(block.Labels) == 0 {
		fmt.Fprintf(&b, "block_%d:\n", block.ID)
	}
	for _, label := range block.Labels {
		b.WriteString("label_")
		b.WriteString(label.String())
		b.WriteString(":\n")
	}
	fmt.Fprintf(&b, "    ; entries: %s\n", formatBlockList(block.Entries))
	fmt.Fprintf(&b, "    ; callers: %s\n", formatBlockList(block.Callers))
	fmt.Fprintf(&b, "    ; stack %v, pop %d, access %d\n", &block.Stack, block.Stack.Pops, block.Stack.Access)
	for _, node := range block.Nodes {
		b.WriteString("    ")
		b.WriteString(node.String())
		b.WriteByte('\n')
	}
	b.WriteString("    ")
	b.WriteString(block.Exit.String())
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

// Op returns the op of the node.
func Op(node Node) token.Type {
	switch expr := node.(type) {
	case *AssignStmt:
		return Op(expr.Expr)
	case *ArithExpr:
		return expr.Op
	case *HeapExpr:
		return expr.Op
	case *PrintStmt:
		return expr.Op
	case *ReadExpr:
		return expr.Op
	case *CallStmt:
		return token.Call
	case *JmpStmt:
		return expr.Op
	case *JmpCondStmt:
		return expr.Op
	case *RetStmt:
		return token.Ret
	case *EndStmt:
		return token.End
	default:
		return token.Illegal
	}
}
