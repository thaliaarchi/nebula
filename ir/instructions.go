// Package ir implements Nebula IR instructions and stack.
//
// Nebula IR is an intermediate language between Whitespace and LLVM IR.
//
package ir // import "github.com/andrewarchi/nebula/ir"

import "math/big"

// Inst is an instruction with a source location.
type Inst interface {
	Source() *Source
	OpString() string
}

// Value is an expression or constant with a set of uses.
type Value interface {
	Uses() *[]*ValueUse
	Source() *Source
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

// Source is the source position of a node.
type Source struct {
	File string
	Line uint
	Col  uint
}

// SourceTODO fills unimplemented source information for now.
// TODO: remove.
var SourceTODO Source

// IntConst is a constant integer value.
type IntConst struct {
	Int  *big.Int
	uses []*ValueUse
	src  Source
}

// NewIntConst constructs an IntConst.
func NewIntConst(i *big.Int, src Source) *IntConst {
	return &IntConst{Int: i, src: src}
}

// Uses returns the set of instructions referring this node.
func (i *IntConst) Uses() *[]*ValueUse { return &i.uses }

// Source returns the source location of this node.
func (i *IntConst) Source() *Source { return &i.src }

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
	src      Source
}

// NewBinaryExpr constructs a BinaryExpr.
func NewBinaryExpr(op BinaryOp, lhs, rhs Value, src Source) *BinaryExpr {
	b := &BinaryExpr{Op: op, src: src}
	b.operands[0] = NewValueUse(lhs, b, 0)
	b.operands[1] = NewValueUse(rhs, b, 1)
	return b
}

// Operands returns the binary LHS and RHS values.
func (b *BinaryExpr) Operands() []*ValueUse { return b.operands[:] }

// Uses returns the set of instructions referring this node.
func (b *BinaryExpr) Uses() *[]*ValueUse { return &b.uses }

// Source returns the source location of this node.
func (b *BinaryExpr) Source() *Source { return &b.src }

// OpString pretty prints the op kind.
func (b *BinaryExpr) OpString() string { return b.Op.String() }

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
	src      Source
}

// NewUnaryExpr constructs a UnaryExpr.
func NewUnaryExpr(op UnaryOp, val Value, src Source) *UnaryExpr {
	u := &UnaryExpr{Op: op, src: src}
	u.operands[0] = NewValueUse(val, u, 0)
	return u
}

// Operands returns the unary value.
func (u *UnaryExpr) Operands() []*ValueUse { return u.operands[:] }

// Uses returns the set of instructions referring this node.
func (u *UnaryExpr) Uses() *[]*ValueUse { return &u.uses }

// Source returns the source location of this node.
func (u *UnaryExpr) Source() *Source { return &u.src }

// OpString pretty prints the op kind.
func (u *UnaryExpr) OpString() string { return u.Op.String() }

// LoadStackExpr is an expression that loads a value from under the
// current stack frame. A position of 1 is the top of the stack.
type LoadStackExpr struct {
	Pos  int // stack position
	uses []*ValueUse
	src  Source
}

// NewLoadStackExpr constructs a LoadStackExpr.
func NewLoadStackExpr(pos int, src Source) *LoadStackExpr {
	return &LoadStackExpr{Pos: pos, src: src}
}

// Uses returns the set of instructions referring this node.
func (l *LoadStackExpr) Uses() *[]*ValueUse { return &l.uses }

// Source returns the source location of this node.
func (l *LoadStackExpr) Source() *Source { return &l.src }

// OpString pretty prints the op kind.
func (l *LoadStackExpr) OpString() string { return "loadstack" }

// StoreStackStmt is a statement that stores a value at a position in
// the stack.
type StoreStackStmt struct {
	Pos      int          // stack position
	operands [1]*ValueUse // value to store
	src      Source
}

// NewStoreStackStmt constructs a StoreStackStmt.
func NewStoreStackStmt(pos int, val Value, src Source) *StoreStackStmt {
	s := &StoreStackStmt{Pos: pos, src: src}
	s.operands[0] = NewValueUse(val, s, 0)
	return s
}

// Operands returns the value to be stored in the stack.
func (s *StoreStackStmt) Operands() []*ValueUse { return s.operands[:] }

// Source returns the source location of this node.
func (s *StoreStackStmt) Source() *Source { return &s.src }

// OpString pretty prints the op kind.
func (s *StoreStackStmt) OpString() string { return "storestack" }

// CheckStackStmt is a statement that asserts the stack length.
type CheckStackStmt struct {
	Size int
	src  Source
}

// NewCheckStackStmt constructs a CheckStackStmt.
func NewCheckStackStmt(size int, src Source) *CheckStackStmt {
	return &CheckStackStmt{Size: size, src: src}
}

// Source returns the source location of this node.
func (c *CheckStackStmt) Source() *Source { return &c.src }

// OpString pretty prints the op kind.
func (c *CheckStackStmt) OpString() string { return "checkstack" }

// LoadHeapExpr is an expression that loads a value at an address
// from the heap.
type LoadHeapExpr struct {
	operands [1]*ValueUse // heap address
	uses     []*ValueUse
	src      Source
}

// NewLoadHeapExpr constructs a LoadHeapExpr.
func NewLoadHeapExpr(addr Value, src Source) *LoadHeapExpr {
	l := &LoadHeapExpr{src: src}
	l.operands[0] = NewValueUse(addr, l, 0)
	return l
}

// Operands returns the heap address to load.
func (l *LoadHeapExpr) Operands() []*ValueUse { return l.operands[:] }

// Uses returns the set of instructions referring this node.
func (l *LoadHeapExpr) Uses() *[]*ValueUse { return &l.uses }

// Source returns the source location of this node.
func (l *LoadHeapExpr) Source() *Source { return &l.src }

// OpString pretty prints the op kind.
func (l *LoadHeapExpr) OpString() string { return "loadheap" }

// StoreHeapStmt is a statement that stores a value at an address
// in the heap.
type StoreHeapStmt struct {
	operands [2]*ValueUse // heap address, value
	src      Source
}

// NewStoreHeapStmt constructs a StoreHeapStmt.
func NewStoreHeapStmt(addr, val Value, src Source) *StoreHeapStmt {
	s := &StoreHeapStmt{src: src}
	s.operands[0] = NewValueUse(addr, s, 0)
	s.operands[1] = NewValueUse(val, s, 1)
	return s
}

// Operands returns the heap address and the value to store there.
func (s *StoreHeapStmt) Operands() []*ValueUse { return s.operands[:] }

// Source returns the source location of this node.
func (s *StoreHeapStmt) Source() *Source { return &s.src }

// OpString pretty prints the op kind.
func (s *StoreHeapStmt) OpString() string { return "storeheap" }

// PrintOp is operator kind of a print statement.
type PrintOp uint8

// Print operations.
const (
	Printc PrintOp = iota
	Printi
	Prints
)

func (op PrintOp) String() string {
	switch op {
	case Printc:
		return "printc"
	case Printi:
		return "printi"
	case Prints:
		return "prints"
	}
	return "printerr"
}

// PrintStmt is an expression that prints a value to stdout.
type PrintStmt struct {
	Op       PrintOp
	operands [1]*ValueUse // value to print
	src      Source
}

// NewPrintStmt constructs a PrintStmt.
func NewPrintStmt(op PrintOp, val Value, src Source) *PrintStmt {
	p := &PrintStmt{Op: op, src: src}
	p.operands[0] = NewValueUse(val, p, 0)
	return p
}

// Operands returns the value to print.
func (p *PrintStmt) Operands() []*ValueUse { return p.operands[:] }

// Source returns the source location of this node.
func (p *PrintStmt) Source() *Source { return &p.src }

// OpString pretty prints the op kind.
func (p *PrintStmt) OpString() string { return p.Op.String() }

// ReadOp is the operator kind of a read expression.
type ReadOp uint8

// Read operations.
const (
	Readc ReadOp = iota
	Readi
)

func (op ReadOp) String() string {
	switch op {
	case Readc:
		return "readc"
	case Readi:
		return "readi"
	}
	return "readerr"
}

// ReadExpr is an expression that reads a value from stdin.
type ReadExpr struct {
	Op   ReadOp
	uses []*ValueUse
	src  Source
}

// NewReadExpr constructs a ReadExpr.
func NewReadExpr(op ReadOp, src Source) *ReadExpr {
	return &ReadExpr{Op: op, src: src}
}

// Uses returns the set of instructions referring this node.
func (r *ReadExpr) Uses() *[]*ValueUse { return &r.uses }

// Source returns the source location of this node.
func (r *ReadExpr) Source() *Source { return &r.src }

// OpString pretty prints the op kind.
func (r *ReadExpr) OpString() string { return r.Op.String() }

// FlushStmt is a statement that flushes stdout.
type FlushStmt struct {
	src Source
}

// NewFlushStmt constructs a FlushStmt.
func NewFlushStmt(src Source) *FlushStmt {
	return &FlushStmt{src: src}
}

// Source returns the source location of this node.
func (f *FlushStmt) Source() *Source { return &f.src }

// OpString pretty prints the op kind.
func (f *FlushStmt) OpString() string { return "flush" }

// CallTerm is terminator that pushes the current location to the call
// stack, then jumps to the callee.
type CallTerm struct {
	succs [2]*BasicBlock // callee, next block
	src   Source
}

// NewCallTerm constructs a CallTerm.
func NewCallTerm(callee, next *BasicBlock, src Source) *CallTerm {
	return &CallTerm{succs: [2]*BasicBlock{callee, next}, src: src}
}

// Succs returns the call destination block and the block to which ret
// transfers control.
func (c *CallTerm) Succs() []*BasicBlock { return c.succs[:] }

// Source returns the source location of this node.
func (c *CallTerm) Source() *Source { return &c.src }

// OpString pretty prints the op kind.
func (c *CallTerm) OpString() string { return "call" }

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
	src   Source
}

// NewJmpTerm constructs a JmpTerm.
func NewJmpTerm(op JmpOp, jumpee *BasicBlock, src Source) *JmpTerm {
	return &JmpTerm{Op: op, succs: [1]*BasicBlock{jumpee}, src: src}
}

// Succs returns the jump destination.
func (j *JmpTerm) Succs() []*BasicBlock { return j.succs[:] }

// Source returns the source location of this node.
func (j *JmpTerm) Source() *Source { return &j.src }

// OpString pretty prints the op kind.
func (j *JmpTerm) OpString() string { return j.Op.String() }

// JmpCondOp is the kind of operator for a conditional jump terminator.
type JmpCondOp uint8

// Conditional jump operations.
const (
	Jz JmpCondOp = iota
	Jn
)

func (op JmpCondOp) String() string {
	switch op {
	case Jz:
		return "jz"
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
	src      Source
}

// NewJmpCondTerm constructs a JmpCondTerm.
func NewJmpCondTerm(op JmpCondOp, val Value, trueBlock, falseBlock *BasicBlock, src Source) *JmpCondTerm {
	j := &JmpCondTerm{Op: op, succs: [2]*BasicBlock{trueBlock, falseBlock}, src: src}
	j.operands[0] = NewValueUse(val, j, 0)
	return j
}

// Operands returns the value used by the condition.
func (j *JmpCondTerm) Operands() []*ValueUse { return j.operands[:] }

// Succs returns the jump destination on true condition and the
// destination on false.
func (j *JmpCondTerm) Succs() []*BasicBlock { return j.succs[:] }

// Source returns the source location of this node.
func (j *JmpCondTerm) Source() *Source { return &j.src }

// OpString pretty prints the op kind.
func (j *JmpCondTerm) OpString() string { return j.Op.String() }

// RetTerm is a terminator that returns to the caller.
type RetTerm struct {
	src Source
}

// NewRetTerm constructs a RetTerm.
func NewRetTerm(src Source) *RetTerm {
	return &RetTerm{src: src}
}

// Succs returns no successors.
func (r *RetTerm) Succs() []*BasicBlock { return nil }

// Source returns the source location of this node.
func (r *RetTerm) Source() *Source { return &r.src }

// OpString pretty prints the op kind.
func (r *RetTerm) OpString() string { return "ret" }

// ExitTerm is a terminator that exits the program.
type ExitTerm struct {
	src Source
}

// NewExitTerm constructs a FlushStmt.
func NewExitTerm(src Source) *ExitTerm {
	return &ExitTerm{src: src}
}

// Succs returns no successors.
func (e *ExitTerm) Succs() []*BasicBlock { return nil }

// Source returns the source location of this node.
func (e *ExitTerm) Source() *Source { return &e.src }

// OpString pretty prints the op kind.
func (e *ExitTerm) OpString() string { return "exit" }

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
		AddUse(val, use)
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
	uses := *def.Uses()
	for _, use := range uses {
		use.Def = newDef
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
