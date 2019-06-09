package ws

import "math/big"

type Instr struct {
	Type InstrType
	Arg  *big.Int
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
		return "Invalid"
	case Push:
		return "Push"
	case Dup:
		return "Dup"
	case Copy:
		return "Copy"
	case Swap:
		return "Swap"
	case Drop:
		return "Drop"
	case Slide:
		return "Slide"
	case Add:
		return "Add"
	case Sub:
		return "Sub"
	case Mul:
		return "Mul"
	case Div:
		return "Div"
	case Mod:
		return "Mod"
	case Store:
		return "Store"
	case Retrieve:
		return "Retrieve"
	case Label:
		return "Label"
	case Call:
		return "Call"
	case Jmp:
		return "Jmp"
	case Jz:
		return "Jz"
	case Jn:
		return "Jn"
	case Ret:
		return "Ret"
	case End:
		return "End"
	case Printc:
		return "Printc"
	case Printi:
		return "Printi"
	case Readc:
		return "Readc"
	case Readi:
		return "Readi"
	}
	return "Unknown"
}
