package ws

import (
	"fmt"
	"math/big"
)

type Instr interface {
	Exec(vm *VM)
}

type instrVal struct{ val *big.Int }
type instrN struct{ n int }
type instrChar struct{ char rune }
type instrString struct{ str string }
type instrAddr struct{ addr *big.Int }
type instrAddrVal struct{ addr, val *big.Int }
type instrLabel struct{ label int }
type instrLabelVal struct {
	label int
	val   *big.Int
}

type PushInstr instrVal
type DupInstr struct{}
type CopyInstr instrN
type SwapInstr struct{}
type DropInstr struct{}
type DropNInstr instrN
type SlideInstr instrN

// Exec executes a push instruction.
func (push *PushInstr) Exec(vm *VM) {
	vm.stack.Push(push.val)
}

// Exec executes a dup instruction.
func (*DupInstr) Exec(vm *VM) {
	vm.stack.Push(vm.stack.Top())
}

// Exec executes a copy instruction.
func (copy *CopyInstr) Exec(vm *VM) {
	vm.stack.Push(vm.stack.Get(copy.n))
}

// Exec executes a swap instruction.
func (*SwapInstr) Exec(vm *VM) {
	vm.stack.Swap()
}

// Exec executes a drop instruction.
func (*DropInstr) Exec(vm *VM) {
	vm.stack.Pop()
}

// Exec executes n successive drop instructions.
func (dropN *DropNInstr) Exec(vm *VM) {
	vm.stack.PopN(dropN.n)
}

// Exec executes a slide instruction.
func (slide *SlideInstr) Exec(vm *VM) {
	vm.stack.Slide(slide.n)
}

type AddInstr struct{}
type AddVInstr instrVal
type SubInstr struct{}
type SubRInstr instrVal
type SubLInstr instrVal
type MulInstr struct{}
type MulVInstr instrVal
type DivInstr struct{}
type DivRInstr instrVal
type DivLInstr instrVal
type ModInstr struct{}
type ModRInstr instrVal
type ModLInstr instrVal
type NegInstr struct{}

// Exec executes an add instruction.
func (*AddInstr) Exec(vm *VM) {
	vm.arith((*big.Int).Add)
}

// Exec executes an add instruction with a constant value.
func (addV *AddVInstr) Exec(vm *VM) {
	vm.arithRHS((*big.Int).Add, addV.val)
}

// Exec executes a sub instruction.
func (*SubInstr) Exec(vm *VM) {
	vm.arith((*big.Int).Sub)
}

// Exec executes a sub instruction with a constant rhs.
func (subR *SubRInstr) Exec(vm *VM) {
	vm.arithRHS((*big.Int).Sub, subR.val)
}

// Exec executes a sub instruction with a constant lhs.
func (subL *SubLInstr) Exec(vm *VM) {
	vm.arithLHS((*big.Int).Sub, subL.val)
}

// Exec executes a mul instruction.
func (*MulInstr) Exec(vm *VM) {
	vm.arith((*big.Int).Mul)
}

// Exec executes a mul instruction with a constant value.
func (mulV *MulVInstr) Exec(vm *VM) {
	vm.arithRHS((*big.Int).Mul, mulV.val)
}

// Exec executes a div instruction.
func (*DivInstr) Exec(vm *VM) {
	vm.arith((*big.Int).Div)
}

// Exec executes a div instruction with a constant rhs.
func (divR *DivRInstr) Exec(vm *VM) {
	vm.arithRHS((*big.Int).Div, divR.val)
}

// Exec executes a div instruction with a constant lhs.
func (divL *DivLInstr) Exec(vm *VM) {
	vm.arithLHS((*big.Int).Div, divL.val)
}

// Exec executes a mod instruction.
func (*ModInstr) Exec(vm *VM) {
	vm.arith((*big.Int).Mod)
}

// Exec executes a mod instruction with a constant rhs.
func (modR *ModRInstr) Exec(vm *VM) {
	vm.arithLHS((*big.Int).Mod, modR.val)
}

// Exec executes a mod instruction with a constant lhs.
func (modL *ModLInstr) Exec(vm *VM) {
	vm.arithLHS((*big.Int).Mod, modL.val)
}

// Exec executes a neg instruction.
func (*NegInstr) Exec(vm *VM) {
	x := vm.stack.Top()
	x.Neg(x)
}

type StoreInstr struct{}
type StoreAInstr instrAddr
type StoreVInstr instrVal
type StoreAVInstr instrAddrVal
type RetrieveInstr struct{}
type RetrieveAInstr instrAddr

// Exec executes a store instruction.
func (*StoreInstr) Exec(vm *VM) {
	val, addr := vm.stack.Pop(), vm.stack.Pop()
	vm.heap.Retrieve(addr).(*big.Int).Set(val)
}

// Exec executes a store instruction with a constant address.
func (storeA *StoreAInstr) Exec(vm *VM) {
	val := vm.stack.Pop()
	vm.heap.Retrieve(storeA.addr).(*big.Int).Set(val)
}

// Exec executes a store instruction with a constant value.
func (storeV *StoreVInstr) Exec(vm *VM) {
	addr := vm.stack.Pop()
	vm.heap.Retrieve(addr).(*big.Int).Set(storeV.val)
}

// Exec executes a store instruction with a constant address and value.
func (storeAV *StoreAVInstr) Exec(vm *VM) {
	vm.heap.Retrieve(storeAV.addr).(*big.Int).Set(storeAV.val)
}

// Exec executes a retrieve instruction.
func (*RetrieveInstr) Exec(vm *VM) {
	top := vm.stack.Top()
	top.Set(vm.heap.Retrieve(top).(*big.Int))
}

// Exec executes a retrieve instruction with a constant address.
func (retrieveA *RetrieveAInstr) Exec(vm *VM) {
	vm.stack.Push(vm.heap.Retrieve(retrieveA.addr).(*big.Int))
}

type CallInstr instrLabel
type JmpInstr instrLabel
type JzInstr instrLabel
type JnInstr instrLabel
type JpInstr instrLabel
type JeInstr instrLabelVal
type JlInstr instrLabelVal
type JgInstr instrLabelVal
type JzTopInstr instrLabel
type JnTopInstr instrLabel
type JpTopInstr instrLabel
type JeTopInstr instrLabelVal
type JlTopInstr instrLabelVal
type JgTopInstr instrLabelVal
type RetInstr struct{}
type EndInstr struct{}

// Exec executes a call instruction.
func (call *CallInstr) Exec(vm *VM) {
	vm.callers = append(vm.callers, vm.pc)
	vm.pc = call.label
}

// Exec executes a jmp instruction.
func (jmp *JmpInstr) Exec(vm *VM) {
	vm.pc = jmp.label
}

// Exec executes a jz instruction.
func (jz *JzInstr) Exec(vm *VM) {
	vm.jmpSign(0, jz.label)
}

// Exec executes a jn instruction.
func (jn *JnInstr) Exec(vm *VM) {
	vm.jmpSign(-1, jn.label)
}

// Exec executes a jp instruction.
func (jp *JpInstr) Exec(vm *VM) {
	vm.jmpSign(1, jp.label)
}

// Exec executes a je instruction with a constant value.
func (je *JeInstr) Exec(vm *VM) {
	vm.jmpCmp(0, je.label, je.val)
}

// Exec executes a jl instruction with a constant value.
func (jl *JlInstr) Exec(vm *VM) {
	vm.jmpCmp(-1, jl.label, jl.val)
}

// Exec executes a jg instruction with a constant value.
func (jg *JgInstr) Exec(vm *VM) {
	vm.jmpCmp(1, jg.label, jg.val)
}

// Exec executes a jz instruction and leaving the top stack value.
func (jzTop *JzTopInstr) Exec(vm *VM) {
	vm.jmpSignTop(0, jzTop.label)
}

// Exec executes a jn instruction and leaving the top stack value.
func (jnTop *JnTopInstr) Exec(vm *VM) {
	vm.jmpSignTop(-1, jnTop.label)
}

// Exec executes a jp instruction and leaving the top stack value.
func (jpTop *JpTopInstr) Exec(vm *VM) {
	vm.jmpSignTop(1, jpTop.label)
}

// Exec executes a je instruction with a constant value and leaving the top stack value.
func (jeTop *JeTopInstr) Exec(vm *VM) {
	vm.jmpCmpTop(0, jeTop.label, jeTop.val)
}

// Exec executes a jl instruction with a constant value and leaving the top stack value.
func (jlTop *JlTopInstr) Exec(vm *VM) {
	vm.jmpCmpTop(-1, jlTop.label, jlTop.val)
}

// Exec executes a jg instruction with a constant value and leaving the top stack value.
func (jgTop *JgTopInstr) Exec(vm *VM) {
	vm.jmpCmpTop(1, jgTop.label, jgTop.val)
}

// Exec executes a ret instruction.
func (ret *RetInstr) Exec(vm *VM) {
	if len(vm.callers) == 0 {
		panic("call stack underflow: ret")
	}
	vm.pc = vm.callers[len(vm.callers)-1]
	vm.callers = vm.callers[:len(vm.callers)-1]
}

// Exec executes an end instruction.
func (end *EndInstr) Exec(vm *VM) {
	vm.pc = len(vm.instrs)
}

type PrintcInstr struct{}
type PrintcVInstr instrChar
type PrintiInstr struct{}
type PrintsInstr instrString
type ReadcInstr struct{}
type ReadcAInstr instrAddr
type ReadiInstr struct{}
type ReadiAInstr instrAddr

// Exec executes a printc instruction.
func (printc *PrintcInstr) Exec(vm *VM) {
	fmt.Printf("%c", bigIntRune(vm.stack.Pop()))
}

// Exec executes a printc instruction with a constant char.
func (printcV *PrintcVInstr) Exec(vm *VM) {
	fmt.Printf("%c", printcV.char)
}

// Exec executes a printi instruction.
func (printi *PrintiInstr) Exec(vm *VM) {
	fmt.Print(vm.stack.Pop().String())
}

// Exec executes a prints instruction with a constant string.
func (prints *PrintsInstr) Exec(vm *VM) {
	fmt.Print(prints.str)
}

// Exec executes a readc instruction.
func (readc *ReadcInstr) Exec(vm *VM) {
	vm.readRune(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
}

// Exec executes a readc instruction with a constant address.
func (readcA *ReadcAInstr) Exec(vm *VM) {
	vm.readRune(vm.heap.Retrieve(readcA.addr).(*big.Int))
}

// Exec executes a readi instruction.
func (readi *ReadiInstr) Exec(vm *VM) {
	vm.readInt(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
}

// Exec executes a readi instruction with a constant address.
func (readiA *ReadiAInstr) Exec(vm *VM) {
	vm.readInt(vm.heap.Retrieve(readiA.addr).(*big.Int))
}
