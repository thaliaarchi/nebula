package ws

import (
	"fmt"
	"math/big"
)

type Instr struct {
	Type InstrType
	Arg  *big.Int
}

func (instr *Instr) String() string {
	switch {
	case instr.Type == Label:
		return fmt.Sprintf("label_%s:", instr.Arg)
	case instr.Arg == nil:
		return fmt.Sprintf("    %s", &instr.Type)
	default:
		return fmt.Sprintf("    %s %d", &instr.Type, instr.Arg)
	}
}

type InstrType uint8

const Illegal InstrType = 0
const (
	StackInstr = InstrType(8)
	Push       = StackInstr | iota
	Dup
	Copy
	Swap
	Drop
	Slide
)
const (
	ArithInstr = InstrType(16)
	Add        = ArithInstr | iota
	Sub
	Mul
	Div
	Mod
)
const (
	HeapInstr = InstrType(32)
	Store     = HeapInstr | iota
	Retrieve
)
const (
	FlowInstr = InstrType(64)
	Label     = FlowInstr | iota
	Call
	Jmp
	Jz
	Jn
	Ret
	End
)
const (
	IOInstr = InstrType(128)
	Printc  = IOInstr | iota
	Printi
	Readc
	Readi
)

func (typ *InstrType) String() string {
	switch *typ {
	case StackInstr:
		return "stack"
	case Push:
		return "push"
	case Dup:
		return "dup"
	case Copy:
		return "copy"
	case Swap:
		return "swap"
	case Drop:
		return "drop"
	case Slide:
		return "slide"
	case ArithInstr:
		return "arith"
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
	case HeapInstr:
		return "heap"
	case Store:
		return "store"
	case Retrieve:
		return "retrieve"
	case FlowInstr:
		return "flow"
	case Label:
		return "label"
	case Call:
		return "call"
	case Jmp:
		return "jmp"
	case Jz:
		return "jz"
	case Jn:
		return "jn"
	case Ret:
		return "ret"
	case End:
		return "end"
	case IOInstr:
		return "io"
	case Printc:
		return "printc"
	case Printi:
		return "printi"
	case Readc:
		return "readc"
	case Readi:
		return "readi"
	}
	return "illegal"
}
