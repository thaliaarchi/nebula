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
	instrs  []Instr
	pc      int
	callers []int
	stack   bigint.Stack
	heap    bigint.Map
	in      *bufio.Reader
}

func NewVM(tokens []Token) (*VM, error) {
	instrs, err := tokensToInstrs(tokens)
	if err != nil {
		return nil, err
	}
	return &VM{
		instrs:  instrs,
		pc:      0,
		callers: nil,
		stack:   *bigint.NewStack(),
		heap:    *bigint.NewMap(func() interface{} { return new(big.Int) }),
		in:      bufio.NewReader(os.Stdin),
	}, nil
}

func (vm *VM) Run() {
	vm.pc = 0
	vm.stack.Clear()
	vm.heap.Clear()
	vm.Continue()
}

func (vm *VM) Continue() {
	for !vm.Done() {
		vm.Step()
	}
}

func (vm *VM) Step() {
	vm.pc++
	vm.instrs[vm.pc-1].Exec(vm)
}

func (vm *VM) StepDebug() {
	switch vm.instrs[vm.pc].(type) {
	case *PrintcInstr, *PrintiInstr:
		fmt.Print(">> ")
		vm.Step()
		fmt.Println()
	case *ReadcInstr, *ReadiInstr:
		fmt.Print("<< ")
		vm.Step()
		fmt.Println()
	default:
		vm.Step()
	}
}

func (vm *VM) Next() {
	_, isCall := vm.instrs[vm.pc].(*CallInstr)
	vm.StepDebug()
	if isCall {
		for !vm.Done() {
			_, isRet := vm.instrs[vm.pc].(*RetInstr)
			vm.StepDebug()
			if isRet {
				break
			}
		}
	}
}

func (vm *VM) Done() bool {
	return vm.pc >= len(vm.instrs)
}

func (vm *VM) Debug() {
	for !vm.Done() {
		fmt.Printf("%d:\t%s\n", vm.pc, InstrString(vm.instrs[vm.pc]))
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
			break
		case "c", "continue":
			vm.Continue()
			break
		case "s", "step":
			vm.StepDebug()
		case "n", "next":
			vm.Next()
		case "i", "info":
			vm.PrintInfo()
			goto prompt
		default:
			goto prompt
		}
	}
	vm.PrintInfo()
}

func (vm *VM) PrintInfo() {
	fmt.Println("-----")
	fmt.Printf("Stack: %s\n", &vm.stack)
	fmt.Printf("Heap: %s\n", &vm.heap)
}

func (vm *VM) arith(op func(z, x, y *big.Int) *big.Int) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	op(x, x, y)
}

func (vm *VM) arithRHS(op func(z, x, y *big.Int) *big.Int, rhs *big.Int) {
	x := vm.stack.Top()
	op(x, x, rhs)
}

func (vm *VM) arithLHS(op func(z, x, y *big.Int) *big.Int, lhs *big.Int) {
	x := vm.stack.Top()
	op(x, lhs, x)
}

func (vm *VM) jmpSign(sign, label int) {
	if vm.stack.Pop().Sign() == sign {
		vm.pc = label
	}
}

func (vm *VM) jmpCmp(cmp, label int, val *big.Int) {
	if vm.stack.Pop().Cmp(val) == cmp {
		vm.pc = label
	}
}

func (vm *VM) jmpSignTop(sign, label int) {
	if vm.stack.Top().Sign() == sign {
		vm.pc = label
	}
}

func (vm *VM) jmpCmpTop(cmp, label int, val *big.Int) {
	if vm.stack.Top().Cmp(val) == cmp {
		vm.pc = label
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

func tokensToInstrs(tokens []Token) ([]Instr, error) {
	labels, err := getLabels(tokens)
	if err != nil {
		return nil, err
	}
	instrs := make([]Instr, 0, len(tokens))
	for _, token := range tokens {
		var instr Instr
		switch token.Type {
		case Push:
			instr = &PushInstr{token.Arg}
		case Dup:
			instr = &DupInstr{}
		case Copy:
			arg, err := getArg(token.Arg, "copy")
			if err != nil {
				return nil, err
			}
			instr = &CopyInstr{arg}
		case Swap:
			instr = &SwapInstr{}
		case Drop:
			instr = &DropInstr{}
		case Slide:
			arg, err := getArg(token.Arg, "slide")
			if err != nil {
				return nil, err
			}
			instr = &SlideInstr{arg}
		case Add:
			instr = &AddInstr{}
		case Sub:
			instr = &SubInstr{}
		case Mul:
			instr = &MulInstr{}
		case Div:
			instr = &DivInstr{}
		case Mod:
			instr = &ModInstr{}
		case Store:
			instr = &StoreInstr{}
		case Retrieve:
			instr = &RetrieveInstr{}
		case Label:
			continue
		case Call:
			label, err := getLabel(token.Arg, labels, "call")
			if err != nil {
				return nil, err
			}
			instr = &CallInstr{label}
		case Jmp:
			label, err := getLabel(token.Arg, labels, "jmp")
			if err != nil {
				return nil, err
			}
			instr = &JmpInstr{label}
		case Jz:
			label, err := getLabel(token.Arg, labels, "jz")
			if err != nil {
				return nil, err
			}
			instr = &JzInstr{label}
		case Jn:
			label, err := getLabel(token.Arg, labels, "jn")
			if err != nil {
				return nil, err
			}
			instr = &JnInstr{label}
		case Ret:
			instr = &RetInstr{}
		case End:
			instr = &EndInstr{}
		case Printc:
			instr = &PrintcInstr{}
		case Printi:
			instr = &PrintiInstr{}
		case Readc:
			instr = &ReadcInstr{}
		case Readi:
			instr = &ReadiInstr{}
		default:
			return nil, fmt.Errorf("invalid token type: %d", token.Type)
		}
		instrs = append(instrs, instr)
	}
	return instrs, nil
}

func getLabels(tokens []Token) (*bigint.Map, error) {
	labels := bigint.NewMap(nil)
	var i int
	for _, token := range tokens {
		if token.Type == Label {
			replace := labels.Put(token.Arg, i)
			if replace {
				return nil, fmt.Errorf("duplicate label: %s", token.Arg)
			}
			continue
		}
		i++
	}
	return labels, nil
}

func getArg(arg *big.Int, name string) (int, error) {
	a, ok := bigint.ToInt(arg)
	if !ok {
		return 0, fmt.Errorf("argument overflow: %s %s", name, arg)
	}
	return a, nil
}

func getLabel(label *big.Int, labels *bigint.Map, name string) (int, error) {
	l, ok := labels.Get(label)
	if !ok {
		return 0, fmt.Errorf("label does not exist: %s %s", name, label)
	}
	return l.(int), nil
}
