package ws

import (
	"fmt"
	"math/big"
)

type Instr interface {
	Exec(vm *VM)
}

type PushInstr struct{ arg *big.Int }
type DupInstr struct{}
type CopyInstr struct{ arg int }
type SwapInstr struct{}
type DropInstr struct{}
type SlideInstr struct{ arg int }
type AddInstr struct{}
type SubInstr struct{}
type MulInstr struct{}
type DivInstr struct{}
type ModInstr struct{}
type StoreInstr struct{}
type RetrieveInstr struct{}
type CallInstr struct{ label int }
type JmpInstr struct{ label int }
type JzInstr struct{ label int }
type JnInstr struct{ label int }
type RetInstr struct{}
type EndInstr struct{}
type PrintcInstr struct{}
type PrintiInstr struct{}
type ReadcInstr struct{}
type ReadiInstr struct{}

// Exec executes a push instruction.
func (push *PushInstr) Exec(vm *VM) {
	vm.stack.Push(push.arg)
	vm.pc++
}

// Exec executes a dup instruction.
func (dup *DupInstr) Exec(vm *VM) {
	vm.stack.Push(vm.stack.Top())
	vm.pc++
}

// Exec executes a copy instruction.
func (copy *CopyInstr) Exec(vm *VM) {
	vm.stack.Push(vm.stack.Get(copy.arg))
	vm.pc++
}

// Exec executes a swap instruction.
func (swap *SwapInstr) Exec(vm *VM) {
	vm.stack.Swap()
	vm.pc++
}

// Exec executes a drop instruction.
func (drop *DropInstr) Exec(vm *VM) {
	vm.stack.Pop()
	vm.pc++
}

// Exec executes a slide instruction.
func (slide *SlideInstr) Exec(vm *VM) {
	vm.stack.Slide(slide.arg)
	vm.pc++
}

// Exec executes an add instruction.
func (add *AddInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Add(x, y)
	vm.pc++
}

// Exec executes a sub instruction.
func (sub *SubInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Sub(x, y)
	vm.pc++
}

// Exec executes a mul instruction.
func (mul *MulInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Mul(x, y)
	vm.pc++
}

// Exec executes a div instruction.
func (div *DivInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Div(x, y)
	vm.pc++
}

// Exec executes a mod instruction.
func (mod *ModInstr) Exec(vm *VM) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	x.Mod(x, y)
	vm.pc++
}

// Exec executes a store instruction.
func (store *StoreInstr) Exec(vm *VM) {
	val, addr := vm.stack.Pop(), vm.stack.Pop()
	vm.heap.Retrieve(addr).(*big.Int).Set(val)
	vm.pc++
}

// Exec executes a retrieve instruction.
func (retrieve *RetrieveInstr) Exec(vm *VM) {
	top := vm.stack.Top()
	top.Set(vm.heap.Retrieve(top).(*big.Int))
	vm.pc++
}

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

// Exec executes a printc instruction.
func (printc *PrintcInstr) Exec(vm *VM) {
	fmt.Printf("%c", bigIntRune(vm.stack.Pop()))
	vm.pc++
}

// Exec executes a printi instruction.
func (printi *PrintiInstr) Exec(vm *VM) {
	fmt.Print(vm.stack.Pop().String())
	vm.pc++
}

// Exec executes a readc instruction.
func (readc *ReadcInstr) Exec(vm *VM) {
	vm.readRune(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
	vm.pc++
}

// Exec executes a readi instruction.
func (readi *ReadiInstr) Exec(vm *VM) {
	vm.readInt(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
	vm.pc++
}
