package ws

import (
	"math/big"
	"testing"
)

func TestInstrString(t *testing.T) {
	arg := big.NewInt(123)
	tests := []struct {
		Instr  *Instr
		String string
	}{
		{&Instr{Push, arg}, "push 123"},
		{&Instr{Push, nil}, "push 0"},
		{&Instr{Add, arg}, "add"},
		{&Instr{Add, nil}, "add"},
		{&Instr{Label, arg}, "label_123:"},
		{&Instr{Label, nil}, "label_0:"},
	}

	for i, test := range tests {
		if str := test.Instr.String(); str != test.String {
			t.Errorf("test %d: String() = %q, want %q", i+1, str, test.String)
		}
	}
}

func TestInstrTypeGroups(t *testing.T) {
	tests := []struct {
		IsStack, IsArith, IsHeap, IsFlow, IsIO bool
		InstrTypes                             []InstrType
	}{
		{true, false, false, false, false, []InstrType{Push, Dup, Copy, Swap, Drop, Slide}},
		{false, true, false, false, false, []InstrType{Add, Sub, Mul, Div, Mod}},
		{false, false, true, false, false, []InstrType{Store, Retrieve}},
		{false, false, false, true, false, []InstrType{Label, Call, Jmp, Jz, Jn, Ret, End}},
		{false, false, false, false, true, []InstrType{Printc, Printi, Readc, Readi}},
	}

	for _, test := range tests {
		for _, instr := range test.InstrTypes {
			if instr.IsStack() != test.IsStack {
				t.Errorf("(%s).IsStack() = %t, want %t", instr, instr.IsStack(), test.IsStack)
			}
			if instr.IsArith() != test.IsArith {
				t.Errorf("(%s).IsArith() = %t, want %t", instr, instr.IsArith(), test.IsArith)
			}
			if instr.IsHeap() != test.IsHeap {
				t.Errorf("(%s).IsHeap() = %t, want %t", instr, instr.IsHeap(), test.IsHeap)
			}
			if instr.IsFlow() != test.IsFlow {
				t.Errorf("(%s).IsFlow() = %t, want %t", instr, instr.IsFlow(), test.IsFlow)
			}
			if instr.IsIO() != test.IsIO {
				t.Errorf("(%s).IsIO() = %t, want %t", instr, instr.IsIO(), test.IsIO)
			}
		}
	}
}

func TestInstrTypeString(t *testing.T) {
	tests := []struct {
		InstrType InstrType
		String    string
	}{
		{Push, "push"},
		{Dup, "dup"},
		{Copy, "copy"},
		{Swap, "swap"},
		{Drop, "drop"},
		{Slide, "slide"},
		{Add, "add"},
		{Sub, "sub"},
		{Mul, "mul"},
		{Div, "div"},
		{Mod, "mod"},
		{Store, "store"},
		{Retrieve, "retrieve"},
		{Label, "label"},
		{Call, "call"},
		{Jmp, "jmp"},
		{Jz, "jz"},
		{Jn, "jn"},
		{Ret, "ret"},
		{End, "end"},
		{Printc, "printc"},
		{Printi, "printi"},
		{Readc, "readc"},
		{Readi, "readi"},
		{Illegal, "illegal"},
		{stackBeg, "illegal"},
		{stackEnd, "illegal"},
		{arithBeg, "illegal"},
		{arithEnd, "illegal"},
		{heapBeg, "illegal"},
		{heapEnd, "illegal"},
		{flowBeg, "illegal"},
		{flowEnd, "illegal"},
		{ioBeg, "illegal"},
		{ioEnd, "illegal"},
	}

	for i, test := range tests {
		if str := test.InstrType.String(); str != test.String {
			t.Errorf("test %d: String() = %q, want %q", i+1, str, test.String)
		}
	}
}
