package ast

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/andrewarchi/wspace/bigint"
	"github.com/andrewarchi/wspace/token"
)

// AST is a set of interconnected basic blocks.
type AST []*BasicBlock

// BasicBlock is a list of consecutive non-branching instructions in a
// program followed by a branch.
type BasicBlock struct {
	Labels  []*big.Int
	Nodes   []Node
	Edge    FlowStmt
	Callers []*BasicBlock
}

// Node can be any val, expr, or stmt type.
type Node interface {
	String() string
}

// Val can be StackVal, HeapVal, ConstVal, or AddrVal.
type Val Node

// StackVal is a position on the stack.
type StackVal struct{ Val int }

// HeapVal is an address in the heap such as from store or retrieve.
type HeapVal struct{ Val *big.Int }

// ConstVal is a constant value such as from push or an expression with
// constant operands.
type ConstVal struct{ Val *big.Int }

// AddrVal marks a value as being a pointer to a value.
type AddrVal struct{ Val Val }

// BinaryExpr evalutates a binary operation and assigns the result to an
// address. Valid operations are add, sub, mul, div, or mod.
type BinaryExpr struct {
	Op     token.Type
	Assign Val
	LHS    Val
	RHS    Val
}

// UnaryExpr evaluates a unary operation and assigns the result to an
// address. Valid operations are push, store, retrieve.
type UnaryExpr struct {
	Op     token.Type
	Assign Val
	Val    Val
}

// IOStmt prints a value or reads a value to an address. Valid
// operations are printc, printi, readc, or readi.
type IOStmt struct {
	Op  token.Type
	Val Val
}

// FlowStmt can be JmpStmt, JmpCondStmt, RetStmt, EndStmt.
type FlowStmt Node

// JmpStmt unconditionally jumps to a block. Valid instructions are
// call, jmp, and fallthrough.
type JmpStmt struct {
	Op    token.Type
	Block *BasicBlock
}

// JmpCondStmt conditionally jumps to a block based on a value. Valid
// instructions are jz and jn.
type JmpCondStmt struct {
	Op    token.Type
	Val   Val
	True  *BasicBlock
	False *BasicBlock
}

// RetStmt represents a ret.
type RetStmt struct{}

// EndStmt represents an end.
type EndStmt struct{}

// Parse parses tokens into an AST of basic blocks.
func Parse(tokens []token.Token) (AST, error) {
	if needsImplicitEnd(tokens) {
		tokens = append(tokens, token.Token{Type: token.End})
	}
	ast, labels, err := parseBlocks(tokens)
	if err != nil {
		return nil, err
	}
	_ = labels
	// TODO
	// if err := annotateBlockCalls(ast, labels); err != nil {
	// 	return nil, err
	// }
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

func parseBlocks(tokens []token.Token) (AST, *bigint.Map, error) {
	labels := bigint.NewMap(nil) // map[*big.Int]int
	var blocks AST
	for i := 0; i < len(tokens); i++ {
		var block BasicBlock
		for tokens[i].Type == token.Label {
			label := tokens[i].Arg
			if labels.Put(label, len(blocks)) {
				return nil, nil, fmt.Errorf("ast: label is not unique: %s", label)
			}
			block.Labels = append(block.Labels, label)
			i++
		}

		stack := NewStack()
		for ; i < len(tokens); i++ {
			block.Nodes, block.Edge = tokenToNode(block.Nodes, tokens[i], stack)
			if block.Edge != nil {
				if tokens[i].Type == token.Label {
					i--
				}
				break
			}
		}

		blocks = append(blocks, &block)
	}
	return blocks, labels, nil
}

// TODO
/*func annotateBlockCalls(blocks []*BasicBlock, labels *bigint.Map) error {
	for i, block := range blocks {
		switch block.Flow.Type {
		case token.Fallthrough:
			if i < len(blocks)-1 {
				block.Branch = blocks[i+1]
				block.Flow.Type = token.Jmp
				blocks[i+1].Callers = append(blocks[i+1].Callers, block)
			} else {
				panic(fmt.Sprintf("ast: unexpected illegal instruction"))
			}
		case token.Call, token.Jmp, token.Jz, token.Jn:
			label, ok := labels.Get(block.Flow.Arg)
			if !ok {
				return fmt.Errorf("ast: label does not exist: %s", block.Flow.Arg)
			}
			callee := blocks[label.(int)]
			callee.Callers = append(callee.Callers, block)
			block.Branch = callee
			if i < len(blocks)-1 {
				block.Next = blocks[i+1]
			}
		}
	}
	return nil
}*/

func tokenToNode(nodes []Node, tok token.Token, stack *Stack) ([]Node, FlowStmt) {
	switch tok.Type {
	case token.Push:
		return append(nodes, UnaryExpr{
			Op:     token.Push,
			Assign: stack.Push(),
			Val:    ConstVal{tok.Arg},
		}), nil
	case token.Dup:
		stack.Dup()
	case token.Copy:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ast: copy argument out of range: %v", tok.Arg))
		}
		stack.Copy(n)
	case token.Swap:
		stack.Swap()
	case token.Drop:
		stack.Pop()
	case token.Slide:
		n, ok := bigint.ToInt(tok.Arg)
		if !ok {
			panic(fmt.Sprintf("ast: slide argument out of range: %v", tok.Arg))
		}
		stack.Slide(n)

	case token.Add, token.Sub, token.Mul, token.Div, token.Mod:
		rhs, lhs, assign := stack.Pop(), stack.Pop(), stack.Push()
		return append(nodes, BinaryExpr{
			Op:     tok.Type,
			Assign: assign,
			LHS:    lhs,
			RHS:    rhs,
		}), nil

	case token.Store:
		val, assign := stack.Pop(), stack.Pop()
		return append(nodes, UnaryExpr{
			Op:     token.Store,
			Assign: AddrVal{assign},
			Val:    val,
		}), nil
	case token.Retrieve:
		val, assign := stack.Pop(), stack.Push()
		return append(nodes, UnaryExpr{
			Op:     token.Retrieve,
			Assign: assign,
			Val:    val,
		}), nil

		// jump addresses are now lost because there is no lookup so far
	case token.Label:
		return nodes, JmpStmt{Op: token.Fallthrough}
	case token.Call, token.Jmp:
		return nodes, JmpStmt{Op: tok.Type}
	case token.Jz, token.Jn:
		return nodes, JmpCondStmt{
			Op:  tok.Type,
			Val: stack.Pop(),
		}
	case token.Ret:
		return nodes, RetStmt{}
	case token.End:
		return nodes, EndStmt{}
	case token.Fallthrough:
		panic("ast: unexpected fallthrough")

	case token.Printc, token.Printi, token.Readc, token.Readi:
		return append(nodes, IOStmt{
			Op:  tok.Type,
			Val: stack.Pop(),
		}), nil

	default:
		panic(fmt.Sprintf("ast: illegal token: %v", tok.Type))
	}
	return nodes, nil
}

// Name returns the name of the basic block from either the first label
// or the block address.
func (block *BasicBlock) Name() string {
	if block == nil {
		return "<nil>"
	}
	if len(block.Labels) != 0 {
		return fmt.Sprintf("label_%v", block.Labels[0])
	}
	return fmt.Sprintf("%p", block)
}

func (ast AST) String() string {
	var b strings.Builder
	for i, block := range ast {
		if i != 0 {
			b.WriteString("----\n")
		}
		b.WriteString(block.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func (block *BasicBlock) String() string {
	var b strings.Builder
	for _, label := range block.Labels {
		b.WriteString("label_")
		b.WriteString(label.String())
		b.WriteString(":\n")
	}
	for _, node := range block.Nodes {
		b.WriteString("    ")
		b.WriteString(node.String())
		b.WriteByte('\n')
	}
	b.WriteString("    ")
	b.WriteString(block.Edge.String())
	return b.String()
}

func (s StackVal) String() string    { return fmt.Sprintf("%%%d", s.Val) }
func (h HeapVal) String() string     { return fmt.Sprintf("*%v", h.Val) }
func (c ConstVal) String() string    { return fmt.Sprintf("%v", c.Val) }
func (a AddrVal) String() string     { return fmt.Sprintf("*%v", a.Val) }
func (b BinaryExpr) String() string  { return fmt.Sprintf("%v = %v %v %v", b.Assign, b.Op, b.LHS, b.RHS) }
func (u UnaryExpr) String() string   { return fmt.Sprintf("%v = %v %v", u.Assign, u.Op, u.Val) }
func (i IOStmt) String() string      { return fmt.Sprintf("%v %v", i.Op, i.Val) }
func (j JmpStmt) String() string     { return fmt.Sprintf("%v %s", j.Op, j.Block.Name()) }
func (j JmpCondStmt) String() string { return fmt.Sprintf("%v %v %p %p", j.Op, j.Val, j.True, j.True) }
func (RetStmt) String() string       { return "ret" }
func (EndStmt) String() string       { return "end" }
