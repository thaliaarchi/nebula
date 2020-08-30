// Package ir implements Nebula IR instructions and stack.
//
// Nebula IR is an intermediate language between Whitespace and LLVM IR.
//
package ir // import "github.com/andrewarchi/nebula/ir"

import (
	"go/token"
	"math/big"

	"github.com/andrewarchi/nebula/internal/bigint"
)

// Inst is an instruction with a source location.
type Inst interface {
	OpString() string
	Pos() token.Pos
}

// Value is an expression or constant with a set of uses.
type Value interface {
	Uses() []*ValueUse
	NUses() int
	AddUse(use *ValueUse)
	RemoveUse(use *ValueUse) bool
	ReplaceUsesWith(other Value)
	Pos() token.Pos
}

// User is an instruction that uses values.
type User interface {
	Operands() []*ValueUse
	NOperands() int
	Operand(n int) *ValueUse
	SetOperand(n int, val Value)
	ClearOperands()
	UsesValue(val Value) bool
	Inst
}

// TermInst is a branching instruction that terminates a basic block.
type TermInst interface {
	Succs() []*BasicBlock
	NSuccs() int
	Succ(n int) *BasicBlock
	SetSucc(n int, block *BasicBlock)
	Inst
}

// ValueBase implements the Value interface.
type ValueBase struct {
	uses []*ValueUse
}

// Uses returns the set of instructions referring this value.
func (val *ValueBase) Uses() []*ValueUse { return val.uses }

// NUses returns the number of uses.
func (val *ValueBase) NUses() int { return len(val.uses) }

// AddUse adds a use edge to the value and user.
func (val *ValueBase) AddUse(use *ValueUse) {
	val.uses = append(val.uses, use)
}

// RemoveUse removes a use from the use list.
func (val *ValueBase) RemoveUse(use *ValueUse) bool {
	for i := range val.uses {
		if val.uses[i] == use {
			for ; i+1 < len(val.uses); i++ {
				val.uses[i] = val.uses[i+1]
			}
			val.uses = val.uses[:len(val.uses)-1]
			return true
		}
	}
	return false
}

// ReplaceUsesWith replaces all uses of def with newDef.
func (val *ValueBase) ReplaceUsesWith(other Value) {
	for _, use := range val.uses {
		use.def = other
		other.AddUse(use)
	}
	val.uses = val.uses[:0]
}

// UserBase implements the User interface.
type UserBase struct {
	operands  []*ValueUse
	operands2 [2]*ValueUse // storage for operands
	PosBase
}

// Operands returns the user's operands.
func (user *UserBase) Operands() []*ValueUse {
	var operands2 = user.operands2 // Copy array to prevent indexed writes
	return operands2[:len(user.operands)]
}

// NOperands returns the number of operands.
func (user *UserBase) NOperands() int { return len(user.operands) }

// Operand returns the specified operand.
func (user *UserBase) Operand(n int) *ValueUse { return user.operands[n] }

// SetOperand sets the specified operand to a value and updates the use
// lists.
func (user *UserBase) SetOperand(n int, val Value) {
	operand := user.operands[n]
	if operand.def != val {
		if operand.def != nil {
			operand.def.RemoveUse(operand)
		}
		if val != nil {
			val.AddUse(operand)
		}
		operand.def = val
	}
}

// initOperands initializes user operands. User is passed as a parameter
// because ValueUse needs the full User, not the embedded UserBase.
func (user *UserBase) initOperands(u User, vals ...Value) {
	user.operands = user.operands2[:len(vals)]
	for i, val := range vals {
		user.operands[i] = &ValueUse{val, u, i}
		if val != nil {
			val.AddUse(user.operands[i])
		}
	}
}

// ClearOperands clears all operands and removes the uses.
func (user *UserBase) ClearOperands() {
	for i, operand := range user.operands {
		operand.def.RemoveUse(operand)
		user.operands[i] = nil
	}
}

// UsesValue returns whether an operand uses the value.
func (user *UserBase) UsesValue(val Value) bool {
	for _, operand := range user.Operands() {
		if operand.def == val {
			return true
		}
	}
	return false
}

// ValueUse is an edge between a value definition and referrer.
type ValueUse struct {
	def     Value
	user    User
	operand int
}

// Def returns the value definition.
func (use *ValueUse) Def() Value { return use.def }

// SetDef replaces the value definition and updates uses.
func (use *ValueUse) SetDef(def Value) {
	if use.def != def {
		if use.def != nil {
			use.def.RemoveUse(use)
		}
		if def != nil {
			def.AddUse(use)
		}
		use.def = def
	}
}

// User returns the user and user's operand.
func (use *ValueUse) User() (User, int) { return use.user, use.operand }

// TermBase implements the TermInst interface.
type TermBase struct {
	succs []*BasicBlock
	PosBase
}

// Succs returns the terminator's successor blocks.
func (term *TermBase) Succs() []*BasicBlock { return term.succs }

// NSuccs returns the number of successor blocks.
func (term *TermBase) NSuccs() int { return len(term.succs) }

// Succ returns the specified successor block.
func (term *TermBase) Succ(n int) *BasicBlock { return term.succs[n] }

// SetSucc sets the specified successor block to a given block.
func (term *TermBase) SetSucc(n int, block *BasicBlock) {
	term.succs[n] = block
}

// PosBase stores source position information.
type PosBase struct {
	pos token.Pos
}

// Pos returns the source location of this node.
func (pb *PosBase) Pos() token.Pos { return pb.pos }

// IntConst is a constant integer value. The contained ints can be
// compared for pointer equality.
type IntConst struct {
	val *big.Int
	ValueBase
	PosBase
}

var intLookup = bigint.NewMap()

// NewIntConst constructs an IntConst.
func NewIntConst(val *big.Int, pos token.Pos) *IntConst {
	pair, _ := intLookup.GetOrPutPair(val, nil) // keep only one equivalent *big.Int
	return &IntConst{val: pair.K, PosBase: PosBase{pos: pos}}
}

// Int returns the constant integer.
func (ic *IntConst) Int() *big.Int { return ic.val }

// BinaryOp is the operator kind of a binary expression.
type BinaryOp uint8

// Binary operations.
const (
	Add BinaryOp = iota + 1
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
)

func (op BinaryOp) String() string {
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
	}
	return "binaryerr"
}

// BinaryExpr is an arithmetic expression with two operands.
type BinaryExpr struct {
	Op BinaryOp
	ValueBase
	UserBase
	PosBase
}

// NewBinaryExpr constructs a BinaryExpr.
func NewBinaryExpr(op BinaryOp, lhs, rhs Value, pos token.Pos) *BinaryExpr {
	bin := &BinaryExpr{Op: op, PosBase: PosBase{pos: pos}}
	bin.initOperands(bin, lhs, rhs)
	return bin
}

// OpString pretty prints the op kind.
func (bin *BinaryExpr) OpString() string { return bin.Op.String() }

// UnaryOp is the operator kind of a unary expression.
type UnaryOp uint8

// Unary operations.
const (
	Neg UnaryOp = iota + 1
)

func (op UnaryOp) String() string {
	switch op {
	case Neg:
		return "neg"
	}
	return "unaryerr"
}

// UnaryExpr is an arithmetic expression with one operand.
type UnaryExpr struct {
	Op UnaryOp
	ValueBase
	UserBase
	PosBase
}

// NewUnaryExpr constructs a UnaryExpr.
func NewUnaryExpr(op UnaryOp, val Value, pos token.Pos) *UnaryExpr {
	un := &UnaryExpr{Op: op, PosBase: PosBase{pos: pos}}
	un.initOperands(un, val)
	return un
}

// OpString pretty prints the op kind.
func (un *UnaryExpr) OpString() string { return un.Op.String() }

// LoadStackExpr is an expression that loads a value from under the
// current stack frame. A position of 1 is the top of the stack.
type LoadStackExpr struct {
	StackPos uint
	ValueBase
	PosBase
}

// NewLoadStackExpr constructs a LoadStackExpr.
func NewLoadStackExpr(stackPos uint, pos token.Pos) *LoadStackExpr {
	return &LoadStackExpr{StackPos: stackPos, PosBase: PosBase{pos: pos}}
}

// OpString pretty prints the op kind.
func (*LoadStackExpr) OpString() string { return "loadstack" }

// StoreStackStmt is a statement that stores a value at a position in
// the stack.
type StoreStackStmt struct {
	StackPos uint
	UserBase
	PosBase
}

// NewStoreStackStmt constructs a StoreStackStmt.
func NewStoreStackStmt(stackPos uint, val Value, pos token.Pos) *StoreStackStmt {
	store := &StoreStackStmt{StackPos: stackPos, PosBase: PosBase{pos: pos}}
	store.initOperands(store, val)
	return store
}

// OpString pretty prints the op kind.
func (*StoreStackStmt) OpString() string { return "storestack" }

// AccessStackStmt is a statement that asserts the stack length.
type AccessStackStmt struct {
	StackSize uint
	PosBase
}

// NewAccessStackStmt constructs a AccessStackStmt.
func NewAccessStackStmt(stackSize uint, pos token.Pos) *AccessStackStmt {
	return &AccessStackStmt{StackSize: stackSize, PosBase: PosBase{pos: pos}}
}

// OpString pretty prints the op kind.
func (*AccessStackStmt) OpString() string { return "accessstack" }

// OffsetStackStmt is a statement that changes the stack length
// relatively.
type OffsetStackStmt struct {
	Offset int
	PosBase
}

// NewOffsetStackStmt constructs a OffsetStackStmt.
func NewOffsetStackStmt(offset int, pos token.Pos) *OffsetStackStmt {
	return &OffsetStackStmt{Offset: offset, PosBase: PosBase{pos: pos}}
}

// OpString pretty prints the op kind.
func (*OffsetStackStmt) OpString() string { return "offsetstack" }

// LoadHeapExpr is an expression that loads a value at an address
// from the heap.
type LoadHeapExpr struct {
	ValueBase
	UserBase
	PosBase
}

// NewLoadHeapExpr constructs a LoadHeapExpr.
func NewLoadHeapExpr(addr Value, pos token.Pos) *LoadHeapExpr {
	load := &LoadHeapExpr{PosBase: PosBase{pos: pos}}
	load.initOperands(load, addr)
	return load
}

// OpString pretty prints the op kind.
func (*LoadHeapExpr) OpString() string { return "loadheap" }

// StoreHeapStmt is a statement that stores a value at an address
// in the heap.
type StoreHeapStmt struct {
	UserBase
	PosBase
}

// NewStoreHeapStmt constructs a StoreHeapStmt.
func NewStoreHeapStmt(addr, val Value, pos token.Pos) *StoreHeapStmt {
	store := &StoreHeapStmt{PosBase: PosBase{pos: pos}}
	store.initOperands(store, addr, val)
	return store
}

// OpString pretty prints the op kind.
func (*StoreHeapStmt) OpString() string { return "storeheap" }

// PrintOp is operator kind of a print statement.
type PrintOp uint8

// Print operations.
const (
	PrintByte PrintOp = iota + 1
	PrintInt
)

func (op PrintOp) String() string {
	switch op {
	case PrintByte:
		return "printbyte"
	case PrintInt:
		return "printint"
	}
	return "printerr"
}

// PrintStmt is an expression that prints a value to stdout.
type PrintStmt struct {
	Op PrintOp
	UserBase
	PosBase
}

// NewPrintStmt constructs a PrintStmt.
func NewPrintStmt(op PrintOp, val Value, pos token.Pos) *PrintStmt {
	print := &PrintStmt{Op: op, PosBase: PosBase{pos: pos}}
	print.initOperands(print, val)
	return print
}

// OpString pretty prints the op kind.
func (print *PrintStmt) OpString() string { return print.Op.String() }

// ReadOp is the operator kind of a read expression.
type ReadOp uint8

// Read operations.
const (
	ReadByte ReadOp = iota + 1
	ReadInt
)

func (op ReadOp) String() string {
	switch op {
	case ReadByte:
		return "readbyte"
	case ReadInt:
		return "readint"
	}
	return "readerr"
}

// ReadExpr is an expression that reads a value from stdin.
type ReadExpr struct {
	Op ReadOp
	ValueBase
	PosBase
}

// NewReadExpr constructs a ReadExpr.
func NewReadExpr(op ReadOp, pos token.Pos) *ReadExpr {
	return &ReadExpr{Op: op, PosBase: PosBase{pos: pos}}
}

// OpString pretty prints the op kind.
func (read *ReadExpr) OpString() string { return read.Op.String() }

// FlushStmt is a statement that flushes stdout.
type FlushStmt struct {
	PosBase
}

// NewFlushStmt constructs a FlushStmt.
func NewFlushStmt(pos token.Pos) *FlushStmt {
	return &FlushStmt{PosBase: PosBase{pos: pos}}
}

// OpString pretty prints the op kind.
func (*FlushStmt) OpString() string { return "flush" }

// PhiExpr is an SSA Φ function with pairs of values and predecessor
// blocks.
type PhiExpr struct {
	values []PhiValue
	ValueBase
	PosBase
}

// PhiValue is a value and predecessor block.
type PhiValue struct {
	Value Value
	Block *BasicBlock
}

// AddIncoming adds a val for an incoming edge to the phi expression.
func (phi *PhiExpr) AddIncoming(val Value, block *BasicBlock) {
	phi.values = append(phi.values, PhiValue{val, block})
}

// Values returns pairs of values and predecessor blocks.
func (phi *PhiExpr) Values() []PhiValue { return phi.values }

// OpString pretty prints the op kind.
func (phi *PhiExpr) OpString() string { return "phi" }

// CallTerm is terminator that pushes the current location to the call
// stack, then jumps to the callee.
type CallTerm struct {
	succs [2]*BasicBlock // callee, next block
	TermBase
	PosBase
}

// NewCallTerm constructs a CallTerm.
func NewCallTerm(callee, next *BasicBlock, pos token.Pos) *CallTerm {
	call := &CallTerm{succs: [2]*BasicBlock{callee, next}, PosBase: PosBase{pos: pos}}
	call.TermBase.succs = call.succs[:]
	return call
}

// OpString pretty prints the op kind.
func (*CallTerm) OpString() string { return "call" }

// JmpOp is the operator kind of a jump terminator.
type JmpOp uint8

// Jump operations.
const (
	Jmp JmpOp = iota + 1
	Fallthrough
)

func (op JmpOp) String() string {
	switch op {
	case Jmp:
		return "jmp"
	case Fallthrough:
		return "fallthrough"
	}
	return "jmperr"
}

// JmpTerm is a terminator that unconditionally jumps to a block.
type JmpTerm struct {
	Op    JmpOp
	succs [1]*BasicBlock // jump destination
	TermBase
	PosBase
}

// NewJmpTerm constructs a JmpTerm.
func NewJmpTerm(op JmpOp, jumpee *BasicBlock, pos token.Pos) *JmpTerm {
	jmp := &JmpTerm{Op: op, succs: [1]*BasicBlock{jumpee}, PosBase: PosBase{pos: pos}}
	jmp.TermBase.succs = jmp.succs[:]
	return jmp
}

// OpString pretty prints the op kind.
func (jmp *JmpTerm) OpString() string { return jmp.Op.String() }

// JmpCondOp is the kind of operator for a conditional jump terminator.
type JmpCondOp uint8

// Conditional jump operations.
const (
	Jz JmpCondOp = iota + 1
	Jnz
	Jn
)

func (op JmpCondOp) String() string {
	switch op {
	case Jz:
		return "jz"
	case Jnz:
		return "jnz"
	case Jn:
		return "jn"
	}
	return "jmpconderr"
}

// JmpCondTerm is a terminator that conditionally jumps to one of
// two blocks.
type JmpCondTerm struct {
	Op    JmpCondOp
	succs [2]*BasicBlock // true block, false block
	UserBase
	TermBase
	PosBase
}

// NewJmpCondTerm constructs a JmpCondTerm.
func NewJmpCondTerm(op JmpCondOp, val Value, trueBlock, falseBlock *BasicBlock, pos token.Pos) *JmpCondTerm {
	jc := &JmpCondTerm{Op: op, succs: [2]*BasicBlock{trueBlock, falseBlock}, PosBase: PosBase{pos: pos}}
	jc.initOperands(jc, val)
	jc.TermBase.succs = jc.succs[:]
	return jc
}

// OpString pretty prints the op kind.
func (jc *JmpCondTerm) OpString() string { return jc.Op.String() }

// RetTerm is a terminator that returns to the caller.
type RetTerm struct {
	TermBase
	PosBase
}

// NewRetTerm constructs a RetTerm.
func NewRetTerm(pos token.Pos) *RetTerm {
	return &RetTerm{PosBase: PosBase{pos: pos}}
}

// OpString pretty prints the op kind.
func (*RetTerm) OpString() string { return "ret" }

// ExitTerm is a terminator that exits the program.
type ExitTerm struct {
	TermBase
	PosBase
}

// NewExitTerm constructs a FlushStmt.
func NewExitTerm(pos token.Pos) *ExitTerm {
	return &ExitTerm{PosBase: PosBase{pos: pos}}
}

// OpString pretty prints the op kind.
func (*ExitTerm) OpString() string { return "exit" }
