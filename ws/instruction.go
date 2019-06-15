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

const (
	Invalid InstrType = iota
	Push
	Dup
	Copy
	Swap
	Drop
	Slide
	Add
	Sub
	Mul
	Div
	Mod
	Store
	Retrieve
	Label
	Call
	Jmp
	Jz
	Jn
	Ret
	End
	Printc
	Printi
	Readc
	Readi
)

func (typ *InstrType) String() string {
	switch *typ {
	case Invalid:
		return "invalid"
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
	return "unknown"
}
