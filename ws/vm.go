package ws

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"strings"
	"unicode/utf8"
)

const eofValue = 0

type VM struct {
	instrs  []InstrExecer
	pc      int
	callers []int
	stack   Stack
	heap    Map
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
		stack:   *NewStack(),
		heap:    *NewMap(func() interface{} { return new(big.Int) }),
		in:      bufio.NewReader(os.Stdin),
	}, nil
}

func (vm *VM) Run() {
	for vm.pc < len(vm.instrs) {
		vm.instrs[vm.pc].Exec(vm)
	}
	fmt.Printf("\nStack: %s\n", &vm.stack)
	fmt.Printf("Heap: %s\n", &vm.heap)
}

type InstrExecer interface {
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

func (vm *VM) jmpCond(sign int, label int) {
	if vm.stack.Pop().Sign() == sign {
		vm.pc = label
	} else {
		vm.pc++
	}
}

func (vm *VM) readRune(x *big.Int) *big.Int {
	r, _, err := vm.in.ReadRune()
	if err == io.EOF {
		return x.SetInt64(eofValue)
	}
	if err != nil {
		panic("readc: " + err.Error())
	}
	return x.SetInt64(int64(r))
}

func (vm *VM) readInt(x *big.Int) *big.Int {
	line, err := vm.in.ReadString('\n')
	if err == io.EOF {
		return x.SetInt64(eofValue)
	}
	if err != nil {
		panic("readi: " + err.Error())
	}
	line = strings.TrimSuffix(line, "\n")
	x, ok := x.SetString(line, 10)
	if !ok {
		panic("invalid number: " + line)
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

func instrExecs(instrs []Instr) ([]InstrExecer, error) {
	labels, err := getLabels(instrs)
	if err != nil {
		return nil, err
	}
	execs := make([]InstrExecer, 0, len(instrs))
	for _, instr := range instrs {
		var instrExec InstrExecer
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

func getLabels(instrs []Instr) (*Map, error) {
	labels := NewMap(func() interface{} { return 0 })
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

func getLabel(label *big.Int, labels *Map, name string) (int, error) {
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
