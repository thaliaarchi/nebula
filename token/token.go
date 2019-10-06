package token

import (
	"fmt"
	"math/big"
)

// Token is a lexical token in the Whitespace language.
type Token struct {
	Type Type
	Arg  *big.Int
}

func (tok *Token) String() string {
	switch {
	case tok.Type == Label:
		if tok.Arg == nil {
			return "label_0:"
		}
		return fmt.Sprintf("label_%s:", tok.Arg)
	case tok.Type.HasArg():
		if tok.Arg == nil {
			return fmt.Sprintf("%s 0", tok.Type)
		}
		return fmt.Sprintf("%s %d", tok.Type, tok.Arg)
	default:
		return tok.Type.String()
	}
}

// Type is the instruction type of a token.
type Type uint8

const (
	Illegal Type = iota

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

	// Extended instructions
	Storea
	Fallthrough
	Prints
)

// IsStack returns true for tokens corresponding to stack manipulation instructions.
func (typ Type) IsStack() bool { return stackBeg < typ && typ < stackEnd }

// IsArith returns true for tokens corresponding to arithmetic instructions.
func (typ Type) IsArith() bool { return arithBeg < typ && typ < arithEnd }

// IsHeap returns true for tokens corresponding to heap access instructions.
func (typ Type) IsHeap() bool { return heapBeg < typ && typ < heapEnd }

// IsFlow returns true for tokens corresponding to flow control instructions.
func (typ Type) IsFlow() bool { return flowBeg < typ && typ < flowEnd }

// IsIO returns true for tokens corresponding to i/o instructions.
func (typ Type) IsIO() bool { return ioBeg < typ && typ < ioEnd }

// HasArg returns true for instructions that require an argument.
func (typ Type) HasArg() bool {
	switch typ {
	case Push, Copy, Slide, Label, Call, Jmp, Jz, Jn:
		return true
	}
	return false
}

func (typ Type) String() string {
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
	case Storea:
		return "storea"
	case Fallthrough:
		return "fallthrough"
	case Prints:
		return "prints"
	}
	return "illegal"
}
