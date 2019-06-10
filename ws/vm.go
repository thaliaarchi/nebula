package ws

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"unicode/utf8"
)

type VM struct {
	instrs  []InstrExec
	pc      int
	callers []int
	stack   []*big.Int
	heap    map[int64]*big.Int
	in      *bufio.Reader
}

func NewVM(instrs []Instr) (*VM, error) {
	execs, err := instrExecs(instrs)
	if err != nil {
		return nil, err
	}
	return &VM{
		instrs:  execs,
		pc:      0,
		callers: nil,
		stack:   nil,
		heap:    make(map[int64]*big.Int),
		in:      bufio.NewReader(os.Stdin),
	}, nil
}

func (vm *VM) Run() {
	for vm.pc < len(vm.instrs) {
		vm.instrs[vm.pc].Exec(vm)
	}
}

type InstrExec interface {
	Exec(vm *VM)
}

type PushInstr struct{ arg *big.Int }
type DupInstr struct{}
type CopyInstr struct{ arg int }
type SwapInstr struct{}
type DropInstr struct{}
type SlideInstr struct{ arg int }

func (push *PushInstr) Exec(vm *VM) {
	vm.execPush(push.arg)
}

func (dup *DupInstr) Exec(vm *VM) {
	vm.checkUnderflow(1, "dup")
	vm.execPush(vm.stack[len(vm.stack)-1])
}

func (copy *CopyInstr) Exec(vm *VM) {
	vm.checkUnderflow(copy.arg+1, "copy")
	vm.execPush(vm.stack[len(vm.stack)-copy.arg-1])
}

func (swap *SwapInstr) Exec(vm *VM) {
	vm.checkUnderflow(2, "swap")
	l := len(vm.stack)
	vm.stack[l-1], vm.stack[l-2] = vm.stack[l-2], vm.stack[l-1]
	vm.pc++
}

func (drop *DropInstr) Exec(vm *VM) {
	vm.checkUnderflow(1, "drop")
	vm.stack = vm.stack[:len(vm.stack)-1]
	vm.pc++
}

func (slide *SlideInstr) Exec(vm *VM) {
	vm.checkUnderflow(slide.arg+1, "slide")
	val := vm.stack[len(vm.stack)-1]
	l := len(vm.stack) - slide.arg
	vm.stack[l-1] = val
	vm.stack = vm.stack[:l]
	vm.pc++
}

func (vm *VM) execPush(val *big.Int) {
	vm.stack = append(vm.stack, new(big.Int).Set(val))
	vm.pc++
}

type AddInstr struct{}
type SubInstr struct{}
type MulInstr struct{}
type DivInstr struct{}
type ModInstr struct{}

func (add *AddInstr) Exec(vm *VM) {
	left, right := vm.execArith("add")
	left.Add(left, right)
}

func (sub *SubInstr) Exec(vm *VM) {
	left, right := vm.execArith("sub")
	left.Sub(left, right)
}

func (mul *MulInstr) Exec(vm *VM) {
	left, right := vm.execArith("mul")
	left.Mul(left, right)
}

func (div *DivInstr) Exec(vm *VM) {
	left, right := vm.execArith("div")
	left.Div(left, right)
}

func (mod *ModInstr) Exec(vm *VM) {
	left, right := vm.execArith("mod")
	left.Mod(left, right)
}

func (vm *VM) execArith(name string) (*big.Int, *big.Int) {
	vm.checkUnderflow(2, name)
	left := vm.stack[len(vm.stack)-2]
	right := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	vm.pc++
	return left, right
}

type StoreInstr struct{}
type RetrieveInstr struct{}

func (store *StoreInstr) Exec(vm *VM) {
	vm.checkUnderflow(2, "store")
	l := len(vm.stack)
	addr, val := vm.stack[l-2], vm.stack[l-1]
	vm.checkAddr(addr, "store")
	vm.heap[addr.Int64()] = val
	vm.stack = vm.stack[:l-2]
	vm.pc++
}

func (retrieve *RetrieveInstr) Exec(vm *VM) {
	vm.checkUnderflow(1, "retrieve")
	addr := vm.stack[len(vm.stack)-1]
	vm.checkAddr(addr, "retrieve")
	vm.stack[len(vm.stack)-1] = vm.heap[addr.Int64()]
	vm.pc++
}

type CallInstr struct{ label int }
type JmpInstr struct{ label int }
type JzInstr struct{ label int }
type JnInstr struct{ label int }
type RetInstr struct{}
type EndInstr struct{}

func (call *CallInstr) Exec(vm *VM) {
	vm.callers = append(vm.callers, vm.pc)
	vm.pc = call.label
}

func (jmp *JmpInstr) Exec(vm *VM) {
	vm.pc = jmp.label
}

func (jz *JzInstr) Exec(vm *VM) {
	vm.execJmpSign(0, jz.label, "jz")
}

func (jn *JnInstr) Exec(vm *VM) {
	vm.execJmpSign(-1, jn.label, "jn")
}

func (ret *RetInstr) Exec(vm *VM) {
	if len(vm.callers) == 0 {
		panic("call stack underflow: ret")
	}
	vm.pc = vm.callers[len(vm.callers)-1] + 1
	vm.callers = vm.callers[:len(vm.callers)-1]
}

func (end *EndInstr) Exec(vm *VM) {
	vm.pc = len(vm.instrs)
}

func (vm *VM) execJmpSign(sign int, label int, name string) {
	vm.checkUnderflow(1, name)
	val := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	if val.Sign() == sign {
		vm.pc = label
	} else {
		vm.pc++
	}
}

type PrintcInstr struct{}
type PrintiInstr struct{}
type ReadcInstr struct{}
type ReadiInstr struct{}

func (printc *PrintcInstr) Exec(vm *VM) {
	vm.checkUnderflow(1, "printi")
	fmt.Printf("%c", bigIntRune(vm.stack[len(vm.stack)-1]))
	vm.stack = vm.stack[:len(vm.stack)-1]
	vm.pc++
}

func (printi *PrintiInstr) Exec(vm *VM) {
	vm.checkUnderflow(1, "printi")
	fmt.Print(vm.stack[len(vm.stack)-1].String())
	vm.stack = vm.stack[:len(vm.stack)-1]
	vm.pc++
}

func (readc *ReadcInstr) Exec(vm *VM) {
	vm.execRead(vm.readRune, "readc")
}

func (readi *ReadiInstr) Exec(vm *VM) {
	vm.execRead(vm.readInt, "readi")
}

func (vm *VM) execRead(read func() *big.Int, name string) {
	vm.checkUnderflow(1, name)
	addr := vm.stack[len(vm.stack)-1]
	vm.checkAddr(addr, name)
	vm.heap[addr.Int64()] = read()
	vm.stack = vm.stack[:len(vm.stack)-1]
	vm.pc++
}

func (vm *VM) readRune() *big.Int {
	r, _, err := vm.in.ReadRune()
	if err != nil && err != io.EOF {
		panic("readc: " + err.Error())
	}
	return new(big.Int).SetInt64(int64(r))
}

func (vm *VM) readInt() *big.Int {
	line, err := vm.in.ReadString('\n')
	if err != nil && err != io.EOF {
		panic("readi: " + err.Error())
	}
	x, ok := new(big.Int).SetString(line, 10)
	if !ok {
		panic("invalid number")
	}
	return x
}

func bigIntRune(x *big.Int) rune {
	invalid := '\uFFFD' // � replacement character
	if !x.IsInt64() {
		return invalid
	}
	v := x.Int64()
	if v >= math.MaxInt32 || !utf8.ValidRune(rune(v)) { // rune is int32
		return invalid
	}
	return rune(v)
}

func (vm *VM) checkUnderflow(n int, name string) {
	if len(vm.stack) < n {
		panic("stack underflow: " + name)
	}
}

func (vm *VM) checkAddr(addr *big.Int, name string) {
	if !addr.IsInt64() {
		panic("address overflow: " + name)
	}
}

func instrExecs(instrs []Instr) ([]InstrExec, error) {
	labels, err := getLabels(instrs)
	if err != nil {
		return nil, err
	}
	execs := make([]InstrExec, 0, len(instrs))
	for _, instr := range instrs {
		var instrExec InstrExec
		switch instr.Type {
		case Push:
			instrExec = &PushInstr{instr.Arg}
		case Dup:
			instrExec = &DupInstr{}
		case Copy:
			arg, err := getArg(instr.Arg, "copy")
			if err != nil {
				return nil, err
			}
			instrExec = &CopyInstr{arg}
		case Swap:
			instrExec = &SwapInstr{}
		case Drop:
			instrExec = &DropInstr{}
		case Slide:
			arg, err := getArg(instr.Arg, "slide")
			if err != nil {
				return nil, err
			}
			instrExec = &SlideInstr{arg}
		case Add:
			instrExec = &AddInstr{}
		case Sub:
			instrExec = &SubInstr{}
		case Mul:
			instrExec = &MulInstr{}
		case Div:
			instrExec = &DivInstr{}
		case Mod:
			instrExec = &ModInstr{}
		case Store:
			instrExec = &StoreInstr{}
		case Retrieve:
			instrExec = &RetrieveInstr{}
		case Label:
			continue
		case Call:
			label, err := getLabel(instr.Arg, labels, "call")
			if err != nil {
				return nil, err
			}
			instrExec = &CallInstr{label}
		case Jmp:
			label, err := getLabel(instr.Arg, labels, "jmp")
			if err != nil {
				return nil, err
			}
			instrExec = &JmpInstr{label}
		case Jz:
			label, err := getLabel(instr.Arg, labels, "jz")
			if err != nil {
				return nil, err
			}
			instrExec = &JzInstr{label}
		case Jn:
			label, err := getLabel(instr.Arg, labels, "jn")
			if err != nil {
				return nil, err
			}
			instrExec = &JnInstr{label}
		case Ret:
			instrExec = &RetInstr{}
		case End:
			instrExec = &EndInstr{}
		case Printc:
			instrExec = &PrintcInstr{}
		case Printi:
			instrExec = &PrintiInstr{}
		case Readc:
			instrExec = &ReadcInstr{}
		case Readi:
			instrExec = &ReadiInstr{}
		default:
			return nil, fmt.Errorf("invalid instruction type: %d", instr.Type)
		}
		execs = append(execs, instrExec)
	}
	return execs, nil
}

func getLabels(instrs []Instr) (*intMap, error) {
	labels := newIntMap()
	var i int
	for _, instr := range instrs {
		if instr.Type == Label {
			replace := labels.Put(instr.Arg, i)
			if replace {
				return nil, fmt.Errorf("duplicate label: %s", instr.Arg)
			}
			continue
		}
		i++
	}
	return labels, nil
}

const maxInt int = int(^uint(0) >> 1)

func getArg(arg *big.Int, name string) (int, error) {
	if !arg.IsInt64() {
		return 0, fmt.Errorf("argument overflow: %s %s", name, arg)
	}
	a := arg.Int64()
	if a > int64(maxInt) {
		return 0, fmt.Errorf("argument overflow: %s %s", name, arg)
	}
	return int(a), nil
}

func getLabel(label *big.Int, labels *intMap, name string) (int, error) {
	l, ok := labels.Get(label)
	if !ok {
		return 0, fmt.Errorf("label does not exist: %s %s", name, label)
	}
	return l.(int), nil
}
