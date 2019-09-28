package ws

import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/andrewarchi/wspace/bigint"
)

const eofValue = 0

type VM struct {
	entry   *Node
	inst    *Node
	callers []*Node
	stack   bigint.Stack
	heap    bigint.Map
	in      *bufio.Reader
}

func NewVM(entry *Node) (*VM, error) {
	return &VM{
		entry:   entry,
		inst:    entry,
		callers: nil,
		stack:   *bigint.NewStack(),
		heap:    *bigint.NewMap(func() interface{} { return new(big.Int) }),
		in:      bufio.NewReader(os.Stdin),
	}, nil
}

func (vm *VM) Run() {
	vm.Reset()
	vm.Continue()
}

func (vm *VM) Continue() {
	for !vm.Done() {
		vm.Step()
	}
}

func (vm *VM) Step() {
	vm.Exec(vm.inst)
}

func (vm *VM) StepDebug() {
	switch vm.inst.Type {
	case Printc, Printi:
		fmt.Print(">> ")
		vm.Step()
		fmt.Println()
	case Readc, Readi:
		fmt.Print("<< ")
		vm.Step()
		fmt.Println()
	default:
		vm.Step()
	}
}

func (vm *VM) Next() {
	isCall := vm.inst.Type == Call
	vm.StepDebug()
	if isCall {
		for !vm.Done() {
			isRet := vm.inst.Type == Ret
			vm.StepDebug()
			if isRet {
				break
			}
		}
	}
}

func (vm *VM) Exit() {
	vm.inst = nil
}

func (vm *VM) Done() bool {
	return vm.inst == nil
}

func (vm *VM) Reset() {
	vm.inst = vm.entry
	vm.stack.Clear()
	vm.heap.Clear()
}

func (vm *VM) Debug() {
	vm.Reset()
	for !vm.Done() {
		fmt.Println(vm.inst.Display())
	prompt:
		fmt.Print("(ws) ")
		input, err := vm.in.ReadString('\n')
		if err != nil {
			fmt.Println("ERROR:", err)
			break
		}
		input = strings.TrimSuffix(input, "\n")
		switch input {
		case "r", "run":
			vm.Run()
		case "c", "continue":
			vm.Continue()
		case "s", "step":
			vm.StepDebug()
		case "n", "next":
			vm.Next()
		case "q", "quit":
			vm.Exit()
		case "i", "info":
			vm.PrintInfo()
			goto prompt
		default:
			goto prompt
		}
	}
	fmt.Println("-----")
	vm.PrintInfo()
}

func (vm *VM) PrintInfo() {
	fmt.Printf("Stack: %s\n", &vm.stack)
	fmt.Printf("Heap: %s\n", &vm.heap)
}

func (vm *VM) PrintStackTrace() {
	fmt.Println(vm.inst.Display())
	fmt.Println("-----")
	vm.PrintInfo()
}

func (vm *VM) Exec(inst *Node) {
	next := inst.Next
	switch inst.Type {
	case Push:
		vm.stack.Push(inst.Arg)
	case Dup:
		vm.stack.Push(vm.stack.Top())
	case Copy:
		n, ok := bigint.ToInt(inst.Arg)
		if !ok {
			panic(fmt.Sprintf("copy argument overflow: %s", inst.Arg))
		}
		vm.stack.Push(vm.stack.Get(n))
	case Swap:
		vm.stack.Swap()
	case Drop:
		vm.stack.Pop()
	case Slide:
		n, ok := bigint.ToInt(inst.Arg)
		if !ok {
			panic(fmt.Sprintf("slide argument overflow: %s", inst.Arg))
		}
		vm.stack.Slide(n)

	case Add:
		vm.arith((*big.Int).Add)
	case Sub:
		vm.arith((*big.Int).Sub)
	case Mul:
		vm.arith((*big.Int).Mul)
	case Div:
		vm.arith((*big.Int).Div)
	case Mod:
		vm.arith((*big.Int).Mod)

	case Store:
		val, addr := vm.stack.Pop(), vm.stack.Pop()
		vm.heap.Retrieve(addr).(*big.Int).Set(val)
	case Retrieve:
		top := vm.stack.Top()
		top.Set(vm.heap.Retrieve(top).(*big.Int))

	case Call:
		vm.callers = append(vm.callers, inst.Next)
		next = inst.Branch
	case Jmp:
		next = inst.Branch
	case Jz:
		next = vm.jmpSign(0, inst)
	case Jn:
		next = vm.jmpSign(-1, inst)
	case Ret:
		if len(vm.callers) == 0 {
			panic("call stack underflow: ret")
		}
		next = vm.callers[len(vm.callers)-1]
		vm.callers = vm.callers[:len(vm.callers)-1]
	case End:
		next = nil

	case Printc:
		fmt.Printf("%c", bigint.ToRune(vm.stack.Pop()))
	case Printi:
		fmt.Print(vm.stack.Pop().String())
	case Readc:
		vm.readRune(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
	case Readi:
		vm.readInt(vm.heap.Retrieve(vm.stack.Pop()).(*big.Int))
	}
	vm.inst = next
}

func (vm *VM) arith(op func(z, x, y *big.Int) *big.Int) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	op(x, x, y)
}

func (vm *VM) jmpSign(sign int, inst *Node) *Node {
	if vm.stack.Pop().Sign() == sign {
		return inst.Branch
	}
	return inst.Next
}

func (vm *VM) jmpCmp(cmp int, inst *Node, val *big.Int) *Node {
	if vm.stack.Pop().Cmp(val) == cmp {
		return inst.Branch
	}
	return inst.Next
}

func (vm *VM) readRune(x *big.Int) {
	r, _, err := vm.in.ReadRune()
	if err == io.EOF {
		x.SetInt64(eofValue)
		return
	}
	if err != nil {
		panic(fmt.Sprintf("readc: %v", err))
	}
	x.SetInt64(int64(r))
}

func (vm *VM) readInt(x *big.Int) {
	line, err := vm.in.ReadString('\n')
	if err == io.EOF {
		x.SetInt64(eofValue)
		return
	}
	if err != nil {
		panic(fmt.Sprintf("readi: %v", err))
	}
	line = strings.TrimSuffix(line, "\n")
	x, ok := x.SetString(line, 10)
	if !ok {
		panic(fmt.Sprintf("invalid number: %v", line))
	}
}
