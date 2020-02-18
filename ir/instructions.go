package ir // import "github.com/andrewarchi/nebula/ir"

import (
	"fmt"
	"math/big"
)

// Node is a non-branching instruction in a basic block.
type Node interface {
	node()
}

// Expr is an expression that produces a value.
type Expr interface {
	Node
	Value
	expr()
}

// Stmt is a statement that does not produce a value.
type Stmt interface {
	Node
	stmt()
}

// Terminator is a branching instruction that terminates a basic block.
type Terminator interface {
	terminator()
}

// Value is an SSA value with a definition and uses.
type Value interface {
	ValueDef() *ValueDef
	value()
}

// ValueDef is the definition of a value.
type ValueDef struct {
	Uses []*ValueUse
}

// ValueUse is an edge between a value and a user.
type ValueUse struct {
	Val     Value
	User    User
	Operand int
}

// User is an expression that has an operand that uses a value.
type User interface {
	Operand(int) *ValueUse
	SetOperand(int, *ValueUse)
	NumOperand() int
	user()
}

// ConstVal is a constant integer value.
type ConstVal struct {
	Def *ValueDef
	Int *big.Int
}

func (*ConstVal) value() {}

// PhiExpr is an SSA φ node. It contains a list of edges to values in
// predecessor blocks.
type PhiExpr struct {
	Def   *ValueDef
	Edges []PhiEdge
}

// PhiEdge is a reference to a value in a predecessor block.
type PhiEdge struct {
	ValueUse *ValueUse
	Block    *BasicBlock
}

func (*PhiExpr) node()  {}
func (*PhiExpr) expr()  {}
func (*PhiExpr) value() {}
func (*PhiExpr) user()  {}

// BinaryExpr is an expression with two operands.
type BinaryExpr struct {
	Def *ValueDef
	Op  OpType
	LHS *ValueUse
	RHS *ValueUse
}

func (*BinaryExpr) node()  {}
func (*BinaryExpr) expr()  {}
func (*BinaryExpr) value() {}
func (*BinaryExpr) user()  {}

// UnaryExpr is an expression with one operand.
type UnaryExpr struct {
	Def *ValueDef
	Op  OpType
	Val *ValueUse
}

func (*UnaryExpr) node()  {}
func (*UnaryExpr) expr()  {}
func (*UnaryExpr) value() {}
func (*UnaryExpr) user()  {}

// LoadStackExpr is an expression that loads a value from under the
// current stack frame. A position of 1 is the top of the stack.
type LoadStackExpr struct {
	Def *ValueDef
	Pos int
}

func (*LoadStackExpr) node()  {}
func (*LoadStackExpr) expr()  {}
func (*LoadStackExpr) value() {}

// LoadHeapExpr is an expression that loads a value at an address
// from the heap.
type LoadHeapExpr struct {
	Def  *ValueDef
	Addr *ValueUse
}

func (*LoadHeapExpr) node()  {}
func (*LoadHeapExpr) expr()  {}
func (*LoadHeapExpr) value() {}
func (*LoadHeapExpr) user()  {}

// StoreHeapStmt is a statement that stores a value at an address
// in the heap.
type StoreHeapStmt struct {
	Addr *ValueUse
	Val  *ValueUse
}

func (*StoreHeapStmt) node() {}
func (*StoreHeapStmt) stmt() {}
func (*StoreHeapStmt) user() {}

// CheckStackStmt is a statement that asserts the stack length.
type CheckStackStmt struct {
	Access int
}

func (*CheckStackStmt) node() {}
func (*CheckStackStmt) stmt() {}

// PrintStmt is an expression that prints a value to stdout.
type PrintStmt struct {
	Op  OpType
	Val *ValueUse
}

func (*PrintStmt) node() {}
func (*PrintStmt) stmt() {}
func (*PrintStmt) user() {}

// ReadExpr is an expression that reads a value from stdin.
type ReadExpr struct {
	Def *ValueDef
	Op  OpType
}

func (*ReadExpr) node()  {}
func (*ReadExpr) expr()  {}
func (*ReadExpr) value() {}

// FlushStmt is a statement that flushes stdout.
type FlushStmt struct{}

func (*FlushStmt) node() {}
func (*FlushStmt) stmt() {}

// CallTerm is terminator that pushes the current location to the call
// stack, then jumps to the callee.
type CallTerm struct {
	Dest *BasicBlock
	Next *BasicBlock
}

func (*CallTerm) node()       {}
func (*CallTerm) terminator() {}

// JmpTerm is a terminator that unconditionally jumps to a block.
type JmpTerm struct {
	Op   OpType
	Dest *BasicBlock
}

func (*JmpTerm) node()       {}
func (*JmpTerm) terminator() {}

// JmpCondTerm is a terminator that conditionally jumps to one of
// two blocks.
type JmpCondTerm struct {
	Op   OpType
	Cond *ValueUse
	Then *BasicBlock
	Else *BasicBlock
}

func (*JmpCondTerm) node()       {}
func (*JmpCondTerm) terminator() {}
func (*JmpCondTerm) user()       {}

// RetTerm is a terminator that returns to the caller.
type RetTerm struct{}

func (*RetTerm) node()       {}
func (*RetTerm) terminator() {}

// ExitTerm is a terminator that exits the program.
type ExitTerm struct{}

func (*ExitTerm) node()       {}
func (*ExitTerm) terminator() {}

// OpType is the kind of operator in a node.
type OpType uint8

// Operators for nodes with multiple types.
const (
	Illegal OpType = iota

	Add
	Sub
	Mul
	Div
	Mod
	Shl
	LShr
	AShr
	And
	Or
	Xor

	Neg

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

// AddIncoming adds a val for an incoming edge to the phi node.
func (phi *PhiExpr) AddIncoming(use *ValueUse, block *BasicBlock) {
	phi.Edges = append(phi.Edges, PhiEdge{use, block})
}

// Value methods

// ValueDef returns the definition information of a Value.
func (c *ConstVal) ValueDef() *ValueDef { return c.Def }

// ValueDef returns the definition information of a Value.
func (phi *PhiExpr) ValueDef() *ValueDef { return phi.Def }

// ValueDef returns the definition information of a Value.
func (b *BinaryExpr) ValueDef() *ValueDef { return b.Def }

// ValueDef returns the definition information of a Value.
func (u *UnaryExpr) ValueDef() *ValueDef { return u.Def }

// ValueDef returns the definition information of a Value.
func (l *LoadStackExpr) ValueDef() *ValueDef { return l.Def }

// ValueDef returns the definition information of a Value.
func (l *LoadHeapExpr) ValueDef() *ValueDef { return l.Def }

// ValueDef returns the definition information of a Value.
func (r *ReadExpr) ValueDef() *ValueDef { return r.Def }

// User methods

// Operand returns the nth operand, panicking if out of range.
func (phi *PhiExpr) Operand(n int) *ValueUse {
	if n < 0 || n > len(phi.Edges) {
		panic(invalidOperand(n))
	}
	return phi.Edges[n].ValueUse
}

// Operand returns the nth operand, panicking if out of range.
func (b *BinaryExpr) Operand(n int) *ValueUse { return getOperand2(n, b.LHS, b.RHS) }

// Operand returns the nth operand, panicking if out of range.
func (u *UnaryExpr) Operand(n int) *ValueUse { return getOperand1(n, u.Val) }

// Operand returns the nth operand, panicking if out of range.
func (l *LoadHeapExpr) Operand(n int) *ValueUse { return getOperand1(n, l.Addr) }

// Operand returns the nth operand, panicking if out of range.
func (s *StoreHeapStmt) Operand(n int) *ValueUse { return getOperand2(n, s.Addr, s.Val) }

// Operand returns the nth operand, panicking if out of range.
func (p *PrintStmt) Operand(n int) *ValueUse { return getOperand1(n, p.Val) }

// Operand returns the nth operand, panicking if out of range.
func (j *JmpCondTerm) Operand(n int) *ValueUse { return getOperand1(n, j.Cond) }

// SetOperand sets the nth operand, panicking if out of range.
func (phi *PhiExpr) SetOperand(n int, use *ValueUse) {
	if n < 0 || n > len(phi.Edges) {
		panic(invalidOperand(n))
	}
	phi.Edges[n].ValueUse = use
}

// SetOperand sets the nth operand, panicking if out of range.
func (b *BinaryExpr) SetOperand(n int, use *ValueUse) { setOperand2(n, &b.LHS, &b.RHS, use) }

// SetOperand sets the nth operand, panicking if out of range.
func (u *UnaryExpr) SetOperand(n int, use *ValueUse) { setOperand1(n, &u.Val, use) }

// SetOperand sets the nth operand, panicking if out of range.
func (l *LoadHeapExpr) SetOperand(n int, use *ValueUse) { setOperand1(n, &l.Addr, use) }

// SetOperand sets the nth operand, panicking if out of range.
func (s *StoreHeapStmt) SetOperand(n int, use *ValueUse) { setOperand2(n, &s.Addr, &s.Val, use) }

// SetOperand sets the nth operand, panicking if out of range.
func (p *PrintStmt) SetOperand(n int, use *ValueUse) { setOperand1(n, &p.Val, use) }

// SetOperand sets the nth operand, panicking if out of range.
func (j *JmpCondTerm) SetOperand(n int, use *ValueUse) { setOperand1(n, &j.Cond, use) }

// NumOperand returns the number of operands.
func (phi *PhiExpr) NumOperand() int { return len(phi.Edges) }

// NumOperand returns the number of operands.
func (b *BinaryExpr) NumOperand() int { return 2 }

// NumOperand returns the number of operands.
func (u *UnaryExpr) NumOperand() int { return 1 }

// NumOperand returns the number of operands.
func (l *LoadHeapExpr) NumOperand() int { return 1 }

// NumOperand returns the number of operands.
func (s *StoreHeapStmt) NumOperand() int { return 2 }

// NumOperand returns the number of operands.
func (p *PrintStmt) NumOperand() int { return 1 }

// NumOperand returns the number of operands.
func (j *JmpCondTerm) NumOperand() int { return 1 }

func getOperand1(n int, operand0 *ValueUse) *ValueUse {
	if n != 0 {
		panic(invalidOperand(n))
	}
	return operand0
}

func getOperand2(n int, operand0, operand1 *ValueUse) *ValueUse {
	switch n {
	case 0:
		return operand0
	case 1:
		return operand1
	default:
		panic(invalidOperand(n))
	}
}

func setOperand1(n int, operand0 **ValueUse, use *ValueUse) {
	if n != 0 {
		panic(invalidOperand(n))
	}
	*operand0 = use
}

func setOperand2(n int, operand0, operand1 **ValueUse, use *ValueUse) {
	switch n {
	case 0:
		*operand0 = use
	case 1:
		*operand1 = use
	default:
		panic(invalidOperand(n))
	}
}

func invalidOperand(n int) string {
	return fmt.Sprintf("ir: invalid operand index: %d", n)
}
