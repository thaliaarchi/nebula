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
		if instr.Arg == nil {
			return "label_0:"
		}
		return fmt.Sprintf("label_%s:", instr.Arg)
	case instr.Type.HasArg():
		if instr.Arg == nil {
			return fmt.Sprintf("%s 0", instr.Type)
		}
		return fmt.Sprintf("%s %d", instr.Type, instr.Arg)
	default:
		return fmt.Sprintf("%s", instr.Type)
	}
}

type InstrType uint8

const (
	Illegal InstrType = iota

	stackBeg
	// Stack manipulation instructions
	Push
	Dup
	Copy
	Swap
	Drop
	Slide
	stackEnd

	arithBeg
	// Arithmetic instructions
	Add
	Sub
	Mul
	Div
	Mod
	arithEnd

	heapBeg
	// Heap access instructions
	Store
	Retrieve
	heapEnd

	flowBeg
	// Flow control instructions
	Label
	Call
	Jmp
	Jz
	Jn
	Ret
	End
	flowEnd

	ioBeg
	// I/O instructions
	Printc
	Printi
	Readc
	Readi
	ioEnd
)

// IsStack returns true for tokens corresponding to stack manipulation instructions.
func (typ InstrType) IsStack() bool { return stackBeg < typ && typ < stackEnd }

// IsArith returns true for tokens corresponding to arithmetic instructions.
func (typ InstrType) IsArith() bool { return arithBeg < typ && typ < arithEnd }

// IsHeap returns true for tokens corresponding to heap access instructions.
func (typ InstrType) IsHeap() bool { return heapBeg < typ && typ < heapEnd }

// IsFlow returns true for tokens corresponding to flow control instructions.
func (typ InstrType) IsFlow() bool { return flowBeg < typ && typ < flowEnd }

// IsIO returns true for tokens corresponding to i/o instructions.
func (typ InstrType) IsIO() bool { return ioBeg < typ && typ < ioEnd }

// HasArg returns true for instructions that require an argument.
func (typ InstrType) HasArg() bool {
	switch typ {
	case Push, Copy, Slide, Label, Call, Jmp, Jz, Jn:
		return true
	default:
		return false
	}
}

func (typ InstrType) String() string {
	switch typ {
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
	case Store:
		return "store"
	case Retrieve:
		return "retrieve"
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
