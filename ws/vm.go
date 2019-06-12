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
	instrs    []InstrExec
	pc        int
	callers   []int
	stack     []*big.Int
	stackSize int
	heap      map[int64]*big.Int
	in        *bufio.Reader
}

func NewVM(instrs []Instr) (*VM, error) {
	execs, err := instrExecs(instrs)
	if err != nil {
		return nil, err
	}
	stack := make([]*big.Int, 1024)
	for i := range stack {
		stack[i] = new(big.Int)
	}
	return &VM{
		instrs:  execs,
		pc:      0,
		callers: nil,
		stack:   stack,
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
	vm.push(push.arg)
	vm.pc++
}

// Exec executes a dup instruction.
func (dup *DupInstr) Exec(vm *VM) {
	vm.push(vm.top())
	vm.pc++
}

// Exec executes a copy instruction.
func (copy *CopyInstr) Exec(vm *VM) {
	vm.checkUnderflow(copy.arg + 1)
	vm.push(vm.stack[vm.stackSize-copy.arg-1])
	vm.pc++
}

// Exec executes a swap instruction.
func (swap *SwapInstr) Exec(vm *VM) {
	vm.checkUnderflow(2)
	s := vm.stackSize
	vm.stack[s-1], vm.stack[s-2] = vm.stack[s-2], vm.stack[s-1]
	vm.pc++
}

// Exec executes a drop instruction.
func (drop *DropInstr) Exec(vm *VM) {
	vm.pop()
	vm.pc++
}

// Exec executes a slide instruction.
func (slide *SlideInstr) Exec(vm *VM) {
	vm.checkUnderflow(slide.arg + 1)
	i := vm.stackSize - 1
	j := vm.stackSize - 1 - slide.arg
	vm.stack[i], vm.stack[j] = vm.stack[j], vm.stack[i]
	vm.stackSize -= slide.arg
	vm.pc++
}

// Exec executes an add instruction.
func (add *AddInstr) Exec(vm *VM) {
	left, right := vm.arith()
	left.Add(left, right)
	vm.pc++
}

// Exec executes a sub instruction.
func (sub *SubInstr) Exec(vm *VM) {
	left, right := vm.arith()
	left.Sub(left, right)
	vm.pc++
}

// Exec executes a mul instruction.
func (mul *MulInstr) Exec(vm *VM) {
	left, right := vm.arith()
	left.Mul(left, right)
	vm.pc++
}

// Exec executes a div instruction.
func (div *DivInstr) Exec(vm *VM) {
	left, right := vm.arith()
	left.Div(left, right)
	vm.pc++
}

// Exec executes a mod instruction.
func (mod *ModInstr) Exec(vm *VM) {
	left, right := vm.arith()
	left.Mod(left, right)
	vm.pc++
}

// Exec executes a store instruction.
func (store *StoreInstr) Exec(vm *VM) {
	vm.checkUnderflow(2)
	addr, val := vm.stack[vm.stackSize-2], vm.stack[vm.stackSize-1]
	vm.retrieve(addr).Set(val)
	vm.stackSize -= 2
	vm.pc++
}

// Exec executes a retrieve instruction.
func (retrieve *RetrieveInstr) Exec(vm *VM) {
	vm.checkUnderflow(1)
	top := vm.stack[vm.stackSize-1]
	top.Set(vm.retrieve(top))
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
	fmt.Printf("%c", bigIntRune(vm.pop()))
	vm.pc++
}

// Exec executes a printi instruction.
func (printi *PrintiInstr) Exec(vm *VM) {
	fmt.Print(vm.pop().String())
	vm.pc++
}

// Exec executes a readc instruction.
func (readc *ReadcInstr) Exec(vm *VM) {
	vm.readRune(vm.retrieve(vm.pop()))
	vm.pc++
}

// Exec executes a readi instruction.
func (readi *ReadiInstr) Exec(vm *VM) {
	vm.readInt(vm.retrieve(vm.pop()))
	vm.pc++
}

func (vm *VM) push(val *big.Int) {
	vm.checkOverflow()
	vm.stack[vm.stackSize].Set(val)
	vm.stackSize++
}

func (vm *VM) pop() *big.Int {
	vm.checkUnderflow(1)
	vm.stackSize--
	return vm.stack[vm.stackSize]
}

func (vm *VM) top() *big.Int {
	vm.checkUnderflow(1)
	return vm.stack[vm.stackSize-1]
}

func (vm *VM) retrieve(addr *big.Int) *big.Int {
	vm.checkAddr(addr)
	key := addr.Int64()
	if val, ok := vm.heap[key]; ok {
		return val
	}
	val := new(big.Int)
	vm.heap[key] = val
	return val
}

func (vm *VM) arith() (*big.Int, *big.Int) {
	vm.checkUnderflow(2)
	left := vm.stack[vm.stackSize-2]
	right := vm.stack[vm.stackSize-1]
	vm.stackSize--
	return left, right
}

func (vm *VM) jmpCond(sign int, label int) {
	if vm.pop().Sign() == sign {
		vm.pc = label
	} else {
		vm.pc++
	}
}

func (vm *VM) readRune(x *big.Int) *big.Int {
	r, _, err := vm.in.ReadRune()
	if err != nil && err != io.EOF {
		panic("readc: " + err.Error())
	}
	return x.SetInt64(int64(r))
}

func (vm *VM) readInt(x *big.Int) *big.Int {
	line, err := vm.in.ReadString('\n')
	if err != nil && err != io.EOF {
		panic("readi: " + err.Error())
	}
	x, ok := x.SetString(line, 10)
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

func (vm *VM) checkUnderflow(n int) {
	if vm.stackSize < n {
		panic("stack underflow: " + vm.getInstrName())
	}
}

func (vm *VM) checkOverflow() {
	if vm.stackSize >= cap(vm.stack) {
		panic("stack overflow: " + vm.getInstrName())
	}
}

func (vm *VM) checkAddr(addr *big.Int) {
	if !addr.IsInt64() {
		panic("address overflow: " + vm.getInstrName())
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

func (vm *VM) getInstrName() string {
	instr := vm.instrs[vm.pc]
	if instr == nil {
		return "<nil>"
	}
	switch instr.(type) {
	case *PushInstr:
		return "push"
	case *DupInstr:
		return "dup"
	case *CopyInstr:
		return "copy"
	case *SwapInstr:
		return "swap"
	case *DropInstr:
		return "drop"
	case *SlideInstr:
		return "slide"
	case *AddInstr:
		return "add"
	case *SubInstr:
		return "sub"
	case *MulInstr:
		return "mul"
	case *DivInstr:
		return "div"
	case *ModInstr:
		return "mod"
	case *StoreInstr:
		return "store"
	case *RetrieveInstr:
		return "retrieve"
	case *CallInstr:
		return "call"
	case *JmpInstr:
		return "jmp"
	case *JzInstr:
		return "jz"
	case *JnInstr:
		return "jn"
	case *RetInstr:
		return "ret"
	case *EndInstr:
		return "end"
	case *PrintcInstr:
		return "printc"
	case *PrintiInstr:
		return "printi"
	case *ReadcInstr:
		return "readc"
	case *ReadiInstr:
		return "readi"
	}
	return "invalid"
}
