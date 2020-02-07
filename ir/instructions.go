package ir // import "github.com/andrewarchi/nebula/ir"

import "math/big"

// Val can be StackVal, HeapVal, ConstVal, or AddrVal. Two vals can be
// compared by address for equality.
type Val interface {
	val()
}

// SSAVal is an SSA register.
type SSAVal struct {
	ID int // TODO apparently not uniquely addressable when empty
}

// ConstVal is a constant value such as from push or an expression with
// constant operands.
type ConstVal struct{ Int *big.Int }

// PhiVal represents an SSA Φ function and stores the set of values it
// could be.
type PhiVal struct {
	Refs []PhiRef
}

// PhiRef is a reference in a PhiVal.
type PhiRef struct {
	Val   *Val
	Block *BasicBlock
}

// Node can be any expr or stmt type.
type Node interface {
	node()
}

// Expr represents an SSA expression that produces a value.
type Expr interface {
	Node
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

// LoadStackExpr loads from a position under the current stack frame. A
// position of 1 is the top of the stack.
type LoadStackExpr struct {
	Assign *Val
	Pos    int
}

// LoadHeapExpr evaluates a retrieve operation.
type LoadHeapExpr struct {
	Assign *Val
	Addr   *Val
}

// StoreHeapStmt evaluates a store operation.
type StoreHeapStmt struct {
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

// ValEq returns whether the two vals reference the same definition.
func ValEq(a, b *Val) bool {
	return a != nil && b != nil && *a == *b
}

// AddIncoming adds a val for an incoming edge to the phi node.
func (phi *PhiVal) AddIncoming(val *Val, block *BasicBlock) {
	phi.Refs = append(phi.Refs, PhiRef{val, block})
}

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

func (SSAVal) val()   {}
func (ConstVal) val() {}
func (PhiVal) val()   {}

func (BinaryExpr) node()    {}
func (UnaryExpr) node()     {}
func (LoadStackExpr) node() {}
func (LoadHeapExpr) node()  {}
func (StoreHeapStmt) node() {}
func (PrintStmt) node()     {}
func (ReadExpr) node()      {}
func (FlushStmt) node()     {}
func (CallStmt) node()      {}
func (JmpStmt) node()       {}
func (JmpCondStmt) node()   {}
func (RetStmt) node()       {}
func (ExitStmt) node()      {}

func (BinaryExpr) exprNode()    {}
func (UnaryExpr) exprNode()     {}
func (LoadStackExpr) exprNode() {}
func (LoadHeapExpr) exprNode()  {}
func (StoreHeapStmt) stmtNode() {}
func (PrintStmt) stmtNode()     {}
func (ReadExpr) exprNode()      {}
func (FlushStmt) stmtNode()     {}
func (CallStmt) termNode()      {}
func (JmpStmt) termNode()       {}
func (JmpCondStmt) termNode()   {}
func (RetStmt) termNode()       {}
func (ExitStmt) termNode()      {}
