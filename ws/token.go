package ws // import "github.com/andrewarchi/nebula/ws"

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/andrewarchi/nebula/bigint"
)

// Token is a lexical token in the Whitespace language.
type Token struct {
	Type Type
	Arg  *big.Int
}

// Format formats a token as Whitespace assembly.
func (tok *Token) Format(labelNames *bigint.Map /* map[*big.Int]string */) string {
	switch {
	case tok.Type == Label:
		return fmt.Sprintf("label_%s:", tok.formatArg(labelNames))
	case tok.Type.HasArg():
		return fmt.Sprintf("%s %s", tok.Type, tok.formatArg(labelNames))
	default:
		return tok.Type.String()
	}
}

func (tok *Token) formatArg(labelNames *bigint.Map) string {
	if tok.Type.IsFlow() && labelNames != nil {
		if name, ok := labelNames.Get(tok.Arg); ok {
			return name.(string)
		}
	}
	if tok.Arg == nil {
		return "0"
	}
	return fmt.Sprintf("%d", tok.Arg)
}

func (tok *Token) String() string {
	return tok.Format(nil)
}

// StringWS formats a token as Whitespace.
func (tok *Token) StringWS() string {
	s := tok.Type.StringWS()
	if tok.Type.HasArg() {
		s += tok.formatArgWS()
	}
	return s
}

func (tok *Token) formatArgWS() string {
	var b strings.Builder
	num := tok.Arg
	if !tok.Type.IsFlow() {
		if num.Sign() != -1 {
			b.WriteByte(' ')
		} else {
			b.WriteByte('\t')
		}
	}
	if num.Sign() == -1 {
		num = new(big.Int).Neg(num)
	}
	for i := num.BitLen() - 1; i >= 0; i-- {
		if num.Bit(i) == 0 {
			b.WriteByte(' ')
		} else {
			b.WriteByte('\t')
		}
	}
	b.WriteByte('\n')
	return b.String()
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
	}
	return "illegal"
}

// StringWS returns the string representation of the instruction in
// Whitespace.
func (typ Type) StringWS() string {
	switch typ {
	case Push:
		return "  "
	case Dup:
		return " \n "
	case Copy:
		return " \t "
	case Swap:
		return " \n\t"
	case Drop:
		return " \n\n"
	case Slide:
		return " \t\n"
	case Add:
		return "\t   "
	case Sub:
		return "\t  \t"
	case Mul:
		return "\t  \n"
	case Div:
		return "\t \t "
	case Mod:
		return "\t \t\t"
	case Store:
		return "\t\t "
	case Retrieve:
		return "\t\t\t"
	case Label:
		return "\n  "
	case Call:
		return "\n \t"
	case Jmp:
		return "\n \n"
	case Jz:
		return "\n\t "
	case Jn:
		return "\n\t\t"
	case Ret:
		return "\n\t\n"
	case End:
		return "\n\n\n"
	case Printc:
		return "\t\n  "
	case Printi:
		return "\t\n \t"
	case Readc:
		return "\t\n\t "
	case Readi:
		return "\t\n\t\t"
	}
	return "illegal"
}
