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
	Pos() token.Pos
	OpString() string
}

// Value is an expression or constant with a set of uses.
type Value interface {
	Uses() *[]*ValueUse
	Pos() token.Pos
}

// User is an instruction that uses values.
type User interface {
	Inst
	Operands() []*ValueUse
}

// TermInst is a branching instruction that terminates a basic block.
type TermInst interface {
	Inst
	Succs() []*BasicBlock
}

// ValueUse is an edge between a value definition and referrer.
type ValueUse struct {
	Def         Value
	User        User
	UserOperand uint
}

// NewValueUse constructs a ValueUse.
func NewValueUse(def Value, user User, operand uint) *ValueUse {
	use := &ValueUse{def, user, operand}
	if def != nil {
		AddUse(def, use)
	}
	return use
}

// SetDef replaces the value definition and updates uses.
func (use *ValueUse) SetDef(def Value) {
	if use.Def != def {
		if use.Def != nil {
			RemoveUse(use.Def, use)
		}
		if def != nil {
			AddUse(def, use)
		}
		use.Def = def
	}
}

// SetUser replaces the user.
func (use *ValueUse) SetUser(user User, operand uint) {
	use.User = user
	use.UserOperand = operand
}

// IntConst is a constant integer value. The contained ints can be
// compared for pointer equality.
type IntConst struct {
	val  *big.Int
	uses []*ValueUse
	pos  token.Pos
}

var intLookup = bigint.NewMap()

// NewIntConst constructs an IntConst.
func NewIntConst(val *big.Int, pos token.Pos) *IntConst {
	pair, _ := intLookup.GetOrPutPair(val, nil) // keep only one equivalent *big.Int
	return &IntConst{val: pair.K, pos: pos}
}

// Int returns the constant integer.
func (i *IntConst) Int() *big.Int { return i.val }

// Uses returns the set of instructions referring this node.
func (i *IntConst) Uses() *[]*ValueUse { return &i.uses }

// Pos returns the source location of this node.
func (i *IntConst) Pos() token.Pos { return i.pos }

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

// BinaryExpr is an expression with two operands.
type BinaryExpr struct {
	Op       BinaryOp
	operands [2]*ValueUse // LHS, RHS
	uses     []*ValueUse
	pos      token.Pos
}

// NewBinaryExpr constructs a BinaryExpr.
func NewBinaryExpr(op BinaryOp, lhs, rhs Value, pos token.Pos) *BinaryExpr {
	bin := &BinaryExpr{Op: op, pos: pos}
	bin.operands[0] = NewValueUse(lhs, bin, 0)
	bin.operands[1] = NewValueUse(rhs, bin, 1)
	return bin
}

// Operands returns the binary LHS and RHS values.
func (bin *BinaryExpr) Operands() []*ValueUse { return bin.operands[:] }

// Uses returns the set of instructions referring this node.
func (bin *BinaryExpr) Uses() *[]*ValueUse { return &bin.uses }

// Pos returns the source location of this node.
func (bin *BinaryExpr) Pos() token.Pos { return bin.pos }

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

// UnaryExpr is an expression with one operand.
type UnaryExpr struct {
	Op       UnaryOp
	operands [1]*ValueUse // unary value
	uses     []*ValueUse
	pos      token.Pos
}

// NewUnaryExpr constructs a UnaryExpr.
func NewUnaryExpr(op UnaryOp, val Value, pos token.Pos) *UnaryExpr {
	un := &UnaryExpr{Op: op, pos: pos}
	un.operands[0] = NewValueUse(val, un, 0)
	return un
}

// Operands returns the unary value.
func (un *UnaryExpr) Operands() []*ValueUse { return un.operands[:] }

// Uses returns the set of instructions referring this node.
func (un *UnaryExpr) Uses() *[]*ValueUse { return &un.uses }

// Pos returns the source location of this node.
func (un *UnaryExpr) Pos() token.Pos { return un.pos }

// OpString pretty prints the op kind.
func (un *UnaryExpr) OpString() string { return un.Op.String() }

// LoadStackExpr is an expression that loads a value from under the
// current stack frame. A position of 1 is the top of the stack.
type LoadStackExpr struct {
	StackPos int
	uses     []*ValueUse
	pos      token.Pos
}

// NewLoadStackExpr constructs a LoadStackExpr.
func NewLoadStackExpr(stackPos int, pos token.Pos) *LoadStackExpr {
	if stackPos < 0 {
		panic("NewLoadStackExpr: negative stack position")
	}
	return &LoadStackExpr{StackPos: stackPos, pos: pos}
}

// Uses returns the set of instructions referring this node.
func (load *LoadStackExpr) Uses() *[]*ValueUse { return &load.uses }

// Pos returns the source location of this node.
func (load *LoadStackExpr) Pos() token.Pos { return load.pos }

// OpString pretty prints the op kind.
func (load *LoadStackExpr) OpString() string { return "loadstack" }

// StoreStackStmt is a statement that stores a value at a position in
// the stack.
type StoreStackStmt struct {
	StackPos int
	operands [1]*ValueUse // value to store
	pos      token.Pos
}

// NewStoreStackStmt constructs a StoreStackStmt.
func NewStoreStackStmt(stackPos int, val Value, pos token.Pos) *StoreStackStmt {
	if stackPos < 0 {
		panic("NewLoadStackExpr: negative stack position")
	}
	s := &StoreStackStmt{StackPos: stackPos, pos: pos}
	s.operands[0] = NewValueUse(val, s, 0)
	return s
}

// Operands returns the value to be stored in the stack.
func (s *StoreStackStmt) Operands() []*ValueUse { return s.operands[:] }

// Pos returns the source location of this node.
func (s *StoreStackStmt) Pos() token.Pos { return s.pos }

// OpString pretty prints the op kind.
func (s *StoreStackStmt) OpString() string { return "storestack" }

// AccessStackStmt is a statement that asserts the stack length.
type AccessStackStmt struct {
	StackSize int
	pos       token.Pos
}

// NewAccessStackStmt constructs a AccessStackStmt.
func NewAccessStackStmt(stackSize int, pos token.Pos) *AccessStackStmt {
	if stackSize < 0 {
		panic("NewLoadStackExpr: negative stack position")
	}
	return &AccessStackStmt{StackSize: stackSize, pos: pos}
}

// Pos returns the source location of this node.
func (c *AccessStackStmt) Pos() token.Pos { return c.pos }

// OpString pretty prints the op kind.
func (c *AccessStackStmt) OpString() string { return "checkstack" }

// OffsetStackStmt is a statement that changes the stack length
// relatively.
type OffsetStackStmt struct {
	Offset int
	pos    token.Pos
}

// NewOffsetStackStmt constructs a OffsetStackStmt.
func NewOffsetStackStmt(offset int, pos token.Pos) *OffsetStackStmt {
	return &OffsetStackStmt{Offset: offset, pos: pos}
}

// Pos returns the source location of this node.
func (o *OffsetStackStmt) Pos() token.Pos { return o.pos }

// OpString pretty prints the op kind.
func (o *OffsetStackStmt) OpString() string { return "offsetstack" }

// LoadHeapExpr is an expression that loads a value at an address
// from the heap.
type LoadHeapExpr struct {
	operands [1]*ValueUse // heap address
	uses     []*ValueUse
	pos      token.Pos
}

// NewLoadHeapExpr constructs a LoadHeapExpr.
func NewLoadHeapExpr(addr Value, pos token.Pos) *LoadHeapExpr {
	load := &LoadHeapExpr{pos: pos}
	load.operands[0] = NewValueUse(addr, load, 0)
	return load
}

// Operands returns the heap address to load.
func (load *LoadHeapExpr) Operands() []*ValueUse { return load.operands[:] }

// Uses returns the set of instructions referring this node.
func (load *LoadHeapExpr) Uses() *[]*ValueUse { return &load.uses }

// Pos returns the source location of this node.
func (load *LoadHeapExpr) Pos() token.Pos { return load.pos }

// OpString pretty prints the op kind.
func (load *LoadHeapExpr) OpString() string { return "loadheap" }

// StoreHeapStmt is a statement that stores a value at an address
// in the heap.
type StoreHeapStmt struct {
	operands [2]*ValueUse // heap address, value
	pos      token.Pos
}

// NewStoreHeapStmt constructs a StoreHeapStmt.
func NewStoreHeapStmt(addr, val Value, pos token.Pos) *StoreHeapStmt {
	store := &StoreHeapStmt{pos: pos}
	store.operands[0] = NewValueUse(addr, store, 0)
	store.operands[1] = NewValueUse(val, store, 1)
	return store
}

// Operands returns the heap address and the value to store there.
func (store *StoreHeapStmt) Operands() []*ValueUse { return store.operands[:] }

// Pos returns the source location of this node.
func (store *StoreHeapStmt) Pos() token.Pos { return store.pos }

// OpString pretty prints the op kind.
func (store *StoreHeapStmt) OpString() string { return "storeheap" }

// PrintOp is operator kind of a print statement.
type PrintOp uint8

// Print operations.
const (
	PrintByte PrintOp = iota
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
	Op       PrintOp
	operands [1]*ValueUse // value to print
	pos      token.Pos
}

// NewPrintStmt constructs a PrintStmt.
func NewPrintStmt(op PrintOp, val Value, pos token.Pos) *PrintStmt {
	print := &PrintStmt{Op: op, pos: pos}
	print.operands[0] = NewValueUse(val, print, 0)
	return print
}

// Operands returns the value to print.
func (print *PrintStmt) Operands() []*ValueUse { return print.operands[:] }

// Pos returns the source location of this node.
func (print *PrintStmt) Pos() token.Pos { return print.pos }

// OpString pretty prints the op kind.
func (print *PrintStmt) OpString() string { return print.Op.String() }

// ReadOp is the operator kind of a read expression.
type ReadOp uint8

// Read operations.
const (
	ReadByte ReadOp = iota
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
	Op   ReadOp
	uses []*ValueUse
	pos  token.Pos
}

// NewReadExpr constructs a ReadExpr.
func NewReadExpr(op ReadOp, pos token.Pos) *ReadExpr {
	return &ReadExpr{Op: op, pos: pos}
}

// Uses returns the set of instructions referring this node.
func (read *ReadExpr) Uses() *[]*ValueUse { return &read.uses }

// Pos returns the source location of this node.
func (read *ReadExpr) Pos() token.Pos { return read.pos }

// OpString pretty prints the op kind.
func (read *ReadExpr) OpString() string { return read.Op.String() }

// FlushStmt is a statement that flushes stdout.
type FlushStmt struct {
	pos token.Pos
}

// NewFlushStmt constructs a FlushStmt.
func NewFlushStmt(pos token.Pos) *FlushStmt {
	return &FlushStmt{pos: pos}
}

// Pos returns the source location of this node.
func (flush *FlushStmt) Pos() token.Pos { return flush.pos }

// OpString pretty prints the op kind.
func (flush *FlushStmt) OpString() string { return "flush" }

// PhiExpr is an SSA Φ function with pairs of values and predecessor
// blocks.
type PhiExpr struct {
	values []PhiValue
	uses   []*ValueUse
	pos    token.Pos
}

// PhiValue is a value and predecessor block.
type PhiValue struct {
	Value Value
	Block *BasicBlock
}

// AddIncoming adds a val for an incoming edge to the phi node.
func (phi *PhiExpr) AddIncoming(val Value, block *BasicBlock) {
	phi.values = append(phi.values, PhiValue{val, block})
}

// Values returns pairs of values and predecessor blocks.
func (phi *PhiExpr) Values() []PhiValue {
	return phi.values
}

// Uses returns the set of instructions referring this node.
func (phi *PhiExpr) Uses() *[]*ValueUse { return &phi.uses }

// Pos returns the source location of this node.
func (phi *PhiExpr) Pos() token.Pos { return phi.pos }

// OpString pretty prints the op kind.
func (phi *PhiExpr) OpString() string { return "phi" }

// CallTerm is terminator that pushes the current location to the call
// stack, then jumps to the callee.
type CallTerm struct {
	succs [2]*BasicBlock // callee, next block
	pos   token.Pos
}

// NewCallTerm constructs a CallTerm.
func NewCallTerm(callee, next *BasicBlock, pos token.Pos) *CallTerm {
	return &CallTerm{succs: [2]*BasicBlock{callee, next}, pos: pos}
}

// Succs returns the call destination block and the block to which ret
// transfers control.
func (call *CallTerm) Succs() []*BasicBlock { return call.succs[:] }

// Pos returns the source location of this node.
func (call *CallTerm) Pos() token.Pos { return call.pos }

// OpString pretty prints the op kind.
func (call *CallTerm) OpString() string { return "call" }

// JmpOp is the operator kind of a jump terminator.
type JmpOp uint8

// Jump operations.
const (
	Jmp JmpOp = iota
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
	pos   token.Pos
}

// NewJmpTerm constructs a JmpTerm.
func NewJmpTerm(op JmpOp, jumpee *BasicBlock, pos token.Pos) *JmpTerm {
	return &JmpTerm{Op: op, succs: [1]*BasicBlock{jumpee}, pos: pos}
}

// Succs returns the jump destination.
func (jmp *JmpTerm) Succs() []*BasicBlock { return jmp.succs[:] }

// Pos returns the source location of this node.
func (jmp *JmpTerm) Pos() token.Pos { return jmp.pos }

// OpString pretty prints the op kind.
func (jmp *JmpTerm) OpString() string { return jmp.Op.String() }

// JmpCondOp is the kind of operator for a conditional jump terminator.
type JmpCondOp uint8

// Conditional jump operations.
const (
	Jz JmpCondOp = iota
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
	Op       JmpCondOp
	operands [1]*ValueUse   // condition value
	succs    [2]*BasicBlock // true block, false block
	pos      token.Pos
}

// NewJmpCondTerm constructs a JmpCondTerm.
func NewJmpCondTerm(op JmpCondOp, val Value, trueBlock, falseBlock *BasicBlock, pos token.Pos) *JmpCondTerm {
	jc := &JmpCondTerm{Op: op, succs: [2]*BasicBlock{trueBlock, falseBlock}, pos: pos}
	jc.operands[0] = NewValueUse(val, jc, 0)
	return jc
}

// Operands returns the value used by the condition.
func (jc *JmpCondTerm) Operands() []*ValueUse { return jc.operands[:] }

// Succs returns the jump destination on true condition and the
// destination on false.
func (jc *JmpCondTerm) Succs() []*BasicBlock { return jc.succs[:] }

// Pos returns the source location of this node.
func (jc *JmpCondTerm) Pos() token.Pos { return jc.pos }

// OpString pretty prints the op kind.
func (jc *JmpCondTerm) OpString() string { return jc.Op.String() }

// RetTerm is a terminator that returns to the caller.
type RetTerm struct {
	pos token.Pos
}

// NewRetTerm constructs a RetTerm.
func NewRetTerm(pos token.Pos) *RetTerm {
	return &RetTerm{pos: pos}
}

// Succs returns no successors.
func (ret *RetTerm) Succs() []*BasicBlock { return nil }

// Pos returns the source location of this node.
func (ret *RetTerm) Pos() token.Pos { return ret.pos }

// OpString pretty prints the op kind.
func (ret *RetTerm) OpString() string { return "ret" }

// ExitTerm is a terminator that exits the program.
type ExitTerm struct {
	pos token.Pos
}

// NewExitTerm constructs a FlushStmt.
func NewExitTerm(pos token.Pos) *ExitTerm {
	return &ExitTerm{pos: pos}
}

// Succs returns no successors.
func (exit *ExitTerm) Succs() []*BasicBlock { return nil }

// Pos returns the source location of this node.
func (exit *ExitTerm) Pos() token.Pos { return exit.pos }

// OpString pretty prints the op kind.
func (exit *ExitTerm) OpString() string { return "exit" }

// Operand returns the specified operand.
func Operand(user User, operand uint) *ValueUse {
	return user.Operands()[operand]
}

// SetOperand sets the specified operand to a value and updates the use
// lists.
func SetOperand(user User, operand uint, val Value) {
	use := Operand(user, operand)
	if use.Def != val {
		if use.Def != nil {
			RemoveUse(use.Def, use)
		}
		if val != nil {
			AddUse(val, use)
		}
		use.Def = val
	}
}

// AddUse adds a use edge to the value and user.
func AddUse(def Value, use *ValueUse) {
	uses := def.Uses()
	*uses = append(*uses, use)
}

// RemoveUse removes a use from the use list.
func RemoveUse(def Value, use *ValueUse) bool {
	usesPtr := def.Uses()
	uses := *usesPtr
	for i := range uses {
		if uses[i] == use {
			for ; i+1 < len(uses); i++ {
				uses[i] = uses[i+1]
			}
			uses = uses[:len(uses)-1]
			*usesPtr = uses
			return true
		}
	}
	return false
}

// ReplaceUses replaces all uses of def with newDef.
func ReplaceUses(def, newDef Value) {
	if def != newDef {
		uses := def.Uses()
		for _, use := range *uses {
			use.Def = newDef
			AddUse(newDef, use)
		}
		*uses = (*uses)[:0]
	}
}

// ClearOperands clears all operands and removes the uses.
func ClearOperands(user User) {
	ops := user.Operands()
	for i, operand := range ops {
		RemoveUse(operand.Def, operand)
		ops[i] = nil
	}
}

// Succ returns the specified successor block.
func Succ(term TermInst, succ int) *BasicBlock {
	return term.Succs()[succ]
}

// SetSucc sets the specified successor block to a given block.
func SetSucc(term TermInst, succ int, block *BasicBlock) {
	term.Succs()[succ] = block
}
