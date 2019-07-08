package ws

import (
	"fmt"
	"math/big"
)

type Instr interface {
	Exec(vm *VM)
}

type PushInstr struct{ val *big.Int }
type DupInstr struct{}
type CopyInstr struct{ n int }
type SwapInstr struct{}
type DropInstr struct{}
type DropNInstr struct{ n int }
type SlideInstr struct{ n int }

// Exec executes a push instruction.
func (push *PushInstr) Exec(vm *VM) {
	vm.stack.Push(push.val)
	vm.pc++
}

// Exec executes a dup instruction.
func (*DupInstr) Exec(vm *VM) {
	vm.stack.Push(vm.stack.Top())
	vm.pc++
}

// Exec executes a copy instruction.
func (copy *CopyInstr) Exec(vm *VM) {
	vm.stack.Push(vm.stack.Get(copy.n))
	vm.pc++
}

// Exec executes a swap instruction.
func (*SwapInstr) Exec(vm *VM) {
	vm.stack.Swap()
	vm.pc++
}

// Exec executes a drop instruction.
func (*DropInstr) Exec(vm *VM) {
	vm.stack.Pop()
	vm.pc++
}

// Exec executes n successive drop instructions.
func (dropN *DropNInstr) Exec(vm *VM)

// Exec executes a slide instruction.
func (slide *SlideInstr) Exec(vm *VM) {
	vm.stack.Slide(slide.n)
	vm.pc++
}

type AddInstr struct{}
type AddVInstr struct{ val *big.Int }
type SubInstr struct{}
type SubLInstr struct{ lhs *big.Int }
type SubRInstr struct{ rhs *big.Int }
type MulInstr struct{}
type MulVInstr struct{ val *big.Int }
type DivInstr struct{}
type DivLInstr struct{ lhs *big.Int }
type DivRInstr struct{ rhs *big.Int }
type ModInstr struct{}
type ModLInstr struct{ lhs *big.Int }
type ModRInstr struct{ rhs *big.Int }
type NegInstr struct{}
type LshInstr struct{ val *big.Int }
type RshInstr struct{ val *big.Int }

// Exec executes an add instruction.
func (*AddInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Add(x, y)
	vm.pc++
}

// Exec executes an add instruction with a constant value.
func (addV *AddVInstr) Exec(vm *VM)

// Exec executes a sub instruction.
func (*SubInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Sub(x, y)
	vm.pc++
}

// Exec executes a sub instruction with a constant lhs.
func (subL *SubLInstr) Exec(vm *VM)

// Exec executes a sub instruction with a constant rhs.
func (subR *SubRInstr) Exec(vm *VM)

// Exec executes a mul instruction.
func (*MulInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Mul(x, y)
	vm.pc++
}

// Exec executes a mul instruction with a constant value.
func (mulV *MulVInstr) Exec(vm *VM)

// Exec executes a div instruction.
func (*DivInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Div(x, y)
	vm.pc++
}

// Exec executes a div instruction with a constant lhs.
func (divL *DivLInstr) Exec(vm *VM)

// Exec executes a div instruction with a constant rhs.
func (divR *DivRInstr) Exec(vm *VM)

// Exec executes a mod instruction.
func (*ModInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Mod(x, y)
	vm.pc++
}

// Exec executes a mod instruction with a constant lhs.
func (modL *ModLInstr) Exec(vm *VM)

// Exec executes a mod instruction with a constant rhs.
func (modR *ModRInstr) Exec(vm *VM)

// Exec executes a neg instruction.
func (*NegInstr) Exec(vm *VM)

// Exec executes a lsh instruction with a constant value.
func (lsh *LshInstr) Exec(vm *VM)

// Exec executes a rsh instruction with a constant value.
func (rsh *RshInstr) Exec(vm *VM)

type StoreInstr struct{}
type StoreAInstr struct{ addr *big.Int }
type StoreVInstr struct{ val *big.Int }
type StoreAVInstr struct{ addr, val *big.Int }
type RetrieveInstr struct{}
type RetrieveAInstr struct{ addr *big.Int }

// Exec executes a store instruction.
func (*StoreInstr) Exec(vm *VM) {
	val, addr := vm.stack.Pop(), vm.stack.Pop()
	vm.heap.Retrieve(addr).(*big.Int).Set(val)
	vm.pc++
}

// Exec executes a store instruction with a constant address.
func (storeA *StoreAInstr) Exec(vm *VM)

// Exec executes a store instruction with a constant value.
func (storeV *StoreVInstr) Exec(vm *VM)

// Exec executes a store instruction with a constant address and value.
func (storeAV *StoreAVInstr) Exec(vm *VM)

// Exec executes a retrieve instruction.
func (*RetrieveInstr) Exec(vm *VM) {
	top := vm.stack.Top()
	top.Set(vm.heap.Retrieve(top).(*big.Int))
	vm.pc++
}

// Exec executes a retrieve instruction with a constant address.
func (retrieveA *RetrieveAInstr) Exec(vm *VM)

type CallInstr struct{ label int }
type JmpInstr struct{ label int }
type JzInstr struct{ label int }
type JnInstr struct{ label int }
type JpInstr struct{ label int }
type JeInstr struct {
	label int
	val   *big.Int
}
type JlInstr struct {
	label int
	val   *big.Int
}
type JgInstr struct {
	label int
	val   *big.Int
}
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
	vm.jmpCond(0, jz.label)
}

// Exec executes a jn instruction.
func (jn *JnInstr) Exec(vm *VM) {
	vm.jmpCond(-1, jn.label)
}

// Exec executes a jp instruction.
func (jp *JpInstr) Exec(vm *VM) {
	vm.jmpCond(1, jp.label)
}

// Exec executes a je instruction with a constant value.
func (je *JeInstr) Exec(vm *VM)

// Exec executes a jl instruction with a constant value.
func (jl *JlInstr) Exec(vm *VM)

// Exec executes a jg instruction with a constant value.
func (jg *JgInstr) Exec(vm *VM)

// Exec executes a ret instruction.
func (ret *RetInstr) Exec(vm *VM) {
	if len(vm.callers) == 0 {
		panic("call stack underflow: ret")
	}
	vm.pc = vm.callers[len(vm.callers)-1] + 1
	vm.callers = vm.callers[:len(vm.callers)-1]
}

// Exec executes an end instruction.
func (end *EndInstr) Exec(vm *VM) {
	vm.pc = len(vm.instrs)
}

type PrintcInstr struct{}
type PrintcVInstr struct{ char rune }
type PrintiInstr struct{}
type PrintsInstr struct{ str string }
type ReadcInstr struct{}
type ReadcAInstr struct{ addr *big.Int }
type ReadiInstr struct{}
type ReadiAInstr struct{ addr *big.Int }

// Exec executes a printc instruction.
func (printc *PrintcInstr) Exec(vm *VM) {
	fmt.Printf("%c", bigIntRune(vm.stack.Pop()))
	vm.pc++
}

// Exec executes a printc instruction with a constant char.
func (printcV *PrintcVInstr) Exec(vm *VM) {
	fmt.Printf("%c", printcV.char)
	vm.pc++
}

// Exec executes a printi instruction.
func (printi *PrintiInstr) Exec(vm *VM) {
	fmt.Print(vm.stack.Pop().String())
	vm.pc++
}

// Exec executes a prints instruction with a constant string.
func (prints *PrintsInstr) Exec(vm *VM) {
	fmt.Print(prints.str)
	vm.pc++
}

// Exec executes a readc instruction.
func (readc *ReadcInstr) Exec(vm *VM) {
	vm.readRune(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
	vm.pc++
}

// Exec executes a readc instruction with a constant address.
func (readcA *ReadcAInstr) Exec(vm *VM)

// Exec executes a readi instruction.
func (readi *ReadiInstr) Exec(vm *VM) {
	vm.readInt(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
	vm.pc++
}

// Exec executes a readi instruction with a constant address.
func (readiA *ReadiAInstr) Exec(vm *VM)
