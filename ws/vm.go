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
	instrs  []Instr
	pc      int
	callers []int
	stack   Stack
	heap    Map
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

func (vm *VM) arith(op func(z, x, y *big.Int) *big.Int) {
	y, x := vm.stack.Pop(), vm.stack.Top()
	op(x, x, y)
	vm.pc++
}

func (vm *VM) arithRHS(op func(z, x, y *big.Int) *big.Int, rhs *big.Int) {
	x := vm.stack.Top()
	op(x, x, rhs)
	vm.pc++
}

func (vm *VM) arithLHS(op func(z, x, y *big.Int) *big.Int, lhs *big.Int) {
	x := vm.stack.Top()
	op(x, lhs, x)
	vm.pc++
}

func (vm *VM) jmpSign(sign, label int) {
	if vm.stack.Pop().Sign() == sign {
		vm.pc = label
	} else {
		vm.pc++
	}
}

func (vm *VM) jmpCmp(cmp, label int, val *big.Int) {
	if vm.stack.Pop().Cmp(val) == cmp {
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

func getLabels(tokens []Token) (*Map, error) {
	labels := NewMap(func() interface{} { return 0 })
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
